// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"go/types"
	"strings"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/typesExt"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type Column struct {
	Table                  dbEngine.Table `json:"-"`
	name                   string
	DataType               string
	colDefault             interface{}
	isNullable             bool
	CharacterSetName       string
	comment                string
	UdtName                string
	characterMaximumLength int
	autoInc                bool
	PrimaryKey             bool
	IsHidden               bool
}

func (c *Column) IsNullable() bool {
	return c.isNullable
}

func (c *Column) AutoIncrement() bool {
	return c.autoInc
}

func (c *Column) Default() interface{} {
	return c.colDefault
}

func (c *Column) GetFields(columns []dbEngine.Column) []interface{} {
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		switch name := col.Name(); name {
		case "data_type":
			v[i] = &c.DataType
		case "column_default":
			v[i] = &c.colDefault
		case "is_nullable":
			v[i] = &c.isNullable
		case "character_set_name":
			v[i] = &c.CharacterSetName
		case "character_maximum_length":
			v[i] = &c.characterMaximumLength
		case "udt_name":
			v[i] = &c.UdtName
		case "column_comment":
			v[i] = &c.comment
		default:
			panic("not implement scan for field " + name)
		}
	}

	return v
}

func NewColumnPone(name string, comment string, characterMaximumLength int) *Column {
	return &Column{name: name, comment: comment, characterMaximumLength: characterMaximumLength}
}

func NewColumn(table dbEngine.Table, name string, dataType string, colDefault interface{}, isNullable bool, characterSetName string, comment string, udtName string, characterMaximumLength int, primaryKey bool, isHidden bool) *Column {
	col := &Column{
		Table:                  table,
		name:                   name,
		DataType:               dataType,
		isNullable:             isNullable,
		CharacterSetName:       characterSetName,
		comment:                comment,
		UdtName:                udtName,
		characterMaximumLength: characterMaximumLength,
		PrimaryKey:             primaryKey,
		IsHidden:               isHidden,
	}

	col.SetDefault(colDefault)

	return col
}

func (c *Column) BasicTypeInfo() types.BasicInfo {
	switch c.BasicType() {
	case types.Bool:
		return types.IsBoolean
	case types.Int32, types.Int64:
		return types.IsInteger
	case types.Float32, types.Float64:
		return types.IsFloat
	case types.String:
		return types.IsString
	default:
		return types.IsUntyped
	}
}

func (c *Column) BasicType() types.BasicKind {
	return toType(c.UdtName)
}

func toType(dtName string) types.BasicKind {
	switch dtName {
	case "bool":
		return types.Bool
	case "int2", "_int2":
		return types.Int16
	case "int4", "_int4":
		return types.Int32
	case "int8", "_int8":
		return types.Int64
	case "float4", "_float4":
		return types.Float32
	case "float8", "_float8":
		return types.Float64
	case "numeric", "decimal":
		// todo add check field length
		return types.Float64
	case "date", "timestamp", "timestamptz", "time", "_date", "_timestamp", "_timestamptz", "_time":
		return typesExt.TStruct
	case "json":
		return typesExt.TMap
	case "timerange", "tsrange":
		// todo add check ranges
		return typesExt.TArray
	case "char", "_char", "varchar", "_varchar", "text", "_text", "citext", "_citext",
		"character varying", "_character varying", "bpchar", "_bpchar":
		return types.String
	case "bytea", "_bytea":
		return types.UnsafePointer
	default:
		logs.DebugLog("unknow type ", dtName)
		return types.Invalid
	}
}

const (
	isNotNullable = "not null"
)

var dataTypeAlias = map[string][]string{
	"character varying":           {"varchar(255)", "varchar"},
	"character":                   {"char"},
	"integer":                     {"serial", "int"},
	"bigint":                      {"bigserial"},
	"double precision":            {"float", "real"},
	"timestamp without time zone": {"timestamp"},
	"timestamp with time zone":    {"timestamptz"},
	//todo: add check user-defined types
	"USER-DEFINED": {"timerange"},
	"ARRAY":        {"integer[]", "character varying[]", "citext[]", "bpchar[]", "char"},
}

// todo: add check arrays
func (c *Column) CheckAttr(fieldDefine string) (res string) {
	fieldDefine = strings.ToLower(fieldDefine)
	isMayNull := strings.Contains(fieldDefine, isNotNullable)
	if c.isNullable && isMayNull {
		res += " is nullable "
	} else if !c.isNullable && !isMayNull {
		res += " is not nullable "
	}

	lenCol := c.CharacterMaximumLength()
	udtName := c.UdtName
	if strings.HasPrefix(udtName, "_") {
		udtName = strings.TrimPrefix(udtName, "_") + "[]"
	}
	isTypeValid := strings.HasPrefix(fieldDefine, c.DataType) ||
		strings.HasPrefix(fieldDefine, udtName)
	if !isTypeValid {
		for _, alias := range dataTypeAlias[c.DataType] {
			isTypeValid = strings.HasPrefix(fieldDefine, alias)
			if isTypeValid {
				break
			}
		}
	}

	if isTypeValid {
		if strings.HasPrefix(c.DataType, "character") &&
			(lenCol > 0) &&
			!strings.Contains(fieldDefine, fmt.Sprintf("char(%d)", lenCol)) {
			res += fmt.Sprintf(" has length %d symbols", lenCol)
		}
	} else {
		res += " has type " + c.DataType
		logs.DebugLog(c.DataType, c.UdtName, lenCol)
	}

	return
}

func (c *Column) CharacterMaximumLength() int {
	return c.characterMaximumLength
}

func (c *Column) Comment() string {
	return c.comment
}

func (c *Column) Name() string {
	return c.name
}

func (c *Column) Primary() bool {
	return c.PrimaryKey
}

func (c *Column) Type() string {
	return c.UdtName
}

func (c *Column) Required() bool {
	return !c.isNullable && (c.colDefault == nil)
}

func (c *Column) SetNullable(f bool) {
	c.isNullable = f
}

func (c *Column) SetDefault(d interface{}) {
	str, ok := d.(string)
	if !ok {
		c.colDefault = nil
		return
	}

	str = (strings.Split(str, "::"))[0]

	c.colDefault = strings.Trim(strings.TrimPrefix(str, "nextval("), "'")
	// todo add other case of autogenerae column value
	c.autoInc = strings.HasPrefix(str, "nextval(") || c.colDefault == "CURRENT_TIMESTAMP" || c.colDefault == "CURRENT_USER"
}
