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
	Constraints            map[string]*dbEngine.ForeignKey
	IsHidden               bool
}

func NewColumnForTableBuf(table dbEngine.Table) *Column {
	return &Column{
		Table:       table,
		Constraints: make(map[string]*dbEngine.ForeignKey),
	}
}

func (c *Column) Foreign() *dbEngine.ForeignKey {
	for _, c := range c.Constraints {
		if c != nil {
			return c
		}
	}

	return nil
}

func (c *Column) IsNullable() bool {
	return c.isNullable
}

func (c *Column) AutoIncrement() bool {
	return c.autoInc
}

func (c *Column) Copy() *Column {
	return &Column{
		Table:                  c.Table,
		name:                   c.name,
		DataType:               c.DataType,
		colDefault:             c.colDefault,
		isNullable:             c.isNullable,
		CharacterSetName:       c.CharacterSetName,
		comment:                c.comment,
		UdtName:                c.UdtName,
		characterMaximumLength: c.characterMaximumLength,
		autoInc:                c.autoInc,
		PrimaryKey:             c.PrimaryKey,
		Constraints:            c.Constraints,
		IsHidden:               c.IsHidden,
	}
}

func (c *Column) Default() interface{} {
	return c.colDefault
}

func (c *Column) GetFields(columns []dbEngine.Column) []interface{} {
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		v[i] = c.RefColValue(col.Name())
	}

	return v
}

func NewColumnPone(name string, comment string, characterMaximumLength int) *Column {
	return &Column{name: name, comment: comment, characterMaximumLength: characterMaximumLength}
}

func NewColumn(
	table dbEngine.Table,
	name string,
	dataType string,
	colDefault interface{},
	isNullable bool,
	characterSetName string,
	comment string,
	udtName string,
	characterMaximumLength int,
	primaryKey bool,
	isHidden bool,
) *Column {
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
	case "date", "timestamp", "timestamptz", "time", "_date", "_timestamp", "_timestamptz", "_time", "timerange", "tsrange", "daterange":
		return typesExt.TStruct
	case "json":
		return typesExt.TMap
	case "_numeric":
		// todo add check ranges
		return typesExt.TArray
	case "char", "_char", "varchar", "_varchar", "text", "_text", "citext", "_citext",
		"character varying", "_character varying", "bpchar", "_bpchar":
		return types.String
	case "bytea", "_bytea":
		return types.UnsafePointer
	default:
		logs.DebugLog("unknown type ", dtName)
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

	if str == "NULL" {
		c.colDefault = nil
		return
	}

	c.colDefault = strings.Trim(strings.TrimPrefix(str, "nextval("), "'")
	// todo add other case of autogenerate column value
	c.autoInc = strings.HasPrefix(str, "nextval(") || c.colDefault == "CURRENT_TIMESTAMP" || c.colDefault == "CURRENT_USER"
}

func (c *Column) RefColValue(name string) interface{} {
	switch name {
	case "data_type":
		return &c.DataType
	case "column_name":
		return &c.name
	case "column_default":
		return &c.colDefault
	case "is_nullable":
		return &c.isNullable
	case "character_set_name":
		return &c.CharacterSetName
	case "character_maximum_length":
		return &c.characterMaximumLength
	case "udt_name":
		return &c.UdtName
	case "column_comment":
		return &c.comment
	case "keys":
		return &c.Constraints
	default:
		panic("not implement scan for field " + name)
	}

}
