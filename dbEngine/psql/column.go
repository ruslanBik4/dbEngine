// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"go/types"
	"slices"
	"strings"

	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/gotools/typesExt"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// Column implement store data of column of table
type Column struct {
	table                  *Table
	name                   string
	DataType               string
	colDefault             any
	isNullable             bool
	CharacterSetName       string
	comment                string
	UdtName                string
	characterMaximumLength int
	autoInc                bool
	PrimaryKey             bool
	Constraints            map[string]*dbEngine.ForeignKey
	IsHidden               bool
	Position               int32
	UserDefined            *dbEngine.Types
	basicKind              types.BasicKind
}

// UserDefinedType return define type
func (col *Column) UserDefinedType() *dbEngine.Types {
	return col.UserDefined
}

// NewColumnForTableBuf create Column for scanning operation of Table
func NewColumnForTableBuf(table *Table) *Column {
	return &Column{
		table:       table,
		Constraints: make(map[string]*dbEngine.ForeignKey),
	}
}

// Foreign return  first foreign key of column
func (col *Column) Foreign() *dbEngine.ForeignKey {
	for _, c := range col.Constraints {
		if c != nil {
			return c
		}
	}

	return nil
}

// IsNullable return isNullable flag
func (col *Column) IsNullable() bool {
	return col.isNullable
}

// AutoIncrement return true if column is autoincrement
func (col *Column) AutoIncrement() bool {
	return col.autoInc
}

// Copy column & return new instance
func (col *Column) Copy() *Column {
	return &Column{
		table:                  col.table,
		name:                   col.name,
		basicKind:              col.basicKind,
		DataType:               col.DataType,
		colDefault:             col.colDefault,
		isNullable:             col.isNullable,
		CharacterSetName:       col.CharacterSetName,
		comment:                col.comment,
		UdtName:                col.UdtName,
		characterMaximumLength: col.characterMaximumLength,
		autoInc:                col.autoInc,
		PrimaryKey:             col.PrimaryKey,
		Constraints:            col.Constraints,
		IsHidden:               col.IsHidden,
		UserDefined:            col.UserDefined,
	}
}

// GetFields implement RowColumn interface
func (col *Column) GetFields(columns []dbEngine.Column) []any {
	v := make([]any, len(columns))
	for i, column := range columns {
		v[i] = col.RefColValue(column.Name())
	}

	return v
}

// NewColumnPone create new column with several properties
func NewColumnPone(name string, comment string, characterMaximumLength int) *Column {
	return &Column{name: name, comment: comment, characterMaximumLength: characterMaximumLength}
}

// NewColumn create new column with many properties
func NewColumn(
	table *Table,
	name string,
	dataType string,
	colDefault any,
	isNullable bool,
	characterSetName string,
	comment string,
	udtName string,
	characterMaximumLength int,
	primaryKey bool,
	isHidden bool,
) *Column {
	col := &Column{
		table:                  table,
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

// BasicTypeInfo of columns value
func (col *Column) BasicTypeInfo() types.BasicInfo {
	switch col.BasicType() {
	case types.Bool:
		return types.IsBoolean
	case types.Int32, types.Int64:
		return types.IsInteger
	case types.Float32, types.Float64, types.UntypedFloat:
		return types.IsFloat
	case types.String, types.UntypedString:
		return types.IsString
	default:
		return types.IsUntyped
	}
}

// BasicType return golang type of column
func (col *Column) BasicType() types.BasicKind {
	if col.basicKind != types.Invalid {
		return col.basicKind
	}

	col.defineBasicType(nil, nil)
	if col.basicKind == types.UntypedNil {
		logs.StatusLog(col, col.UdtName, col.UserDefined)
		logs.ErrorStack(errors.New("invalid type"), col.UdtName)
	}

	return col.basicKind
}

func (col *Column) defineBasicType(dbTypes map[string]dbEngine.Types, tables map[string]dbEngine.Table) {
	udtName := col.UdtName
	// it's non-standard type we need chk its user defined
	if col.DataType == "USER-DEFINED" {
		t, ok := dbTypes[udtName]
		if ok {
			col.UserDefined = &t
		} else if _, ok := tables[udtName]; ok {
		} else {
			logs.DebugLog("Routine %s use unknown type %s for params %s", col.Name(), udtName, col.Name())
		}
	}

	if col.UserDefined != nil {
		// enumerate always string
		if len(col.UserDefined.Enumerates) > 0 {
			col.basicKind = types.String
			return
		}

		// we must seek domain type
		for _, tAttr := range col.UserDefined.Attr {
			if tAttr.Name == "domain" {
				logs.StatusLog(col.name, udtName, col.UserDefined, tAttr.Type)
				udtName = tAttr.Type
				break
			}
		}
	}

	col.basicKind = UdtNameToType(udtName, dbTypes, tables)
	if col.BasicType() == types.UntypedNil {
		udtName = strings.TrimPrefix(udtName, "_")
		if t, ok := dbTypes[udtName]; ok {
			col.basicKind = typesExt.TStruct
			col.UserDefined = &t
		} else if _, ok := tables[udtName]; ok {
			col.basicKind = typesExt.TStruct
			//col.UserDefined = &t
		} else if udtName == "anyrange" {
			col.basicKind = typesExt.TStruct
			col.UserDefined = &dbEngine.Types{
				Id:         0,
				Name:       udtName,
				Type:       'r',
				Attr:       nil,
				Enumerates: nil,
			}
		} else {
			logs.ErrorLog(ErrUnknownType, "%s: %s", col.Name(), col.UdtName)
		}
	}
}

// Table implement dbEngine.Column interface
// return table of column
func (col *Column) Table() dbEngine.Table {
	return col.table
}

// UdtNameToType return types.BasicKind according to psql udtName
func UdtNameToType(udtName string, dbTypes map[string]dbEngine.Types, tables map[string]dbEngine.Table) types.BasicKind {
	switch udtName {
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
	case "float8", "_float8", "money", "_money", "double precision":
		return types.Float64
	case "numeric", "decimal", "_numeric", "_decimal":
		// todo add check field length UntypedFloat
		return types.Float64
	case "date", "timestamp", "timestamptz", "time", "_date", "_timestamp", "_timestamptz", "_time", "timerange", "tsrange", "daterange",
		"numrange", "cube", "point":
		return typesExt.TStruct
	case "json", "jsonb":
		return types.UnsafePointer
	case "char", "_char", "varchar", "_varchar", "text", "_text", "citext", "_citext",
		"character varying", "_character varying", "bpchar", "_bpchar":
		return types.String
	case "bytea", "_bytea":
		return types.UnsafePointer
	case "inet", "interval":
		return typesExt.TMap
	case "anyrange":
		return typesExt.TStruct

	default:

		a, _ := strings.CutPrefix(udtName, "_")
		_, isType := dbTypes[a]
		_, isTable := tables[a]
		if isType || isTable {
			return types.UntypedNil
		}
		logs.DebugLog("unknown type: %s", udtName)

		return types.UntypedNil
	}
}

const (
	isNotNullable = "not null"
	isDefineArray = "[]"
)

var dataTypeAlias = map[string][]string{
	"character varying":           {"varchar(255)", "varchar"},
	"bpchar":                      {"char"},
	"character":                   {"char"},
	"varchar":                     {"character varying"},
	"int4":                        {"serial", "int", "integer"},
	"smallint":                    {"smallserial", "int2"},
	"bigint":                      {"bigserial", "biginteger", "int8"},
	"int8":                        {"bigserial", "bigint", "biginteger"},
	"double precision":            {"float", "float8", "real"},
	"timestamp without time zone": {"timestamp"},
	"timestamp with time zone":    {"timestamptz"},
	// todo: add check user-defined types
	"USER-DEFINED": {"timerange"},
	//"ARRAY":        {"integer[]", "character varying[]", "citext[]", "bpchar[]", "char"},
}

// CheckAttr check attributes of column on DB schema according to ddl-file
func (col *Column) CheckAttr(colDefine string) (flags []dbEngine.FlagColumn) {
	colDefine = strings.ToLower(colDefine)
	// todo: add check arrays
	lenCol := col.CharacterMaximumLength()
	udtName := col.UdtName
	a, isArray := strings.CutPrefix(udtName, "_")
	if isArray {
		udtName = a + "[]"
	}
	isTypeValid := isTypeValid(colDefine, col.DataType, a)

	if isTypeValid {
		if isArray != strings.Contains(colDefine, isDefineArray) {
			logs.StatusLog(isArray, colDefine)
			flags = append(flags, dbEngine.ChgToArray)
		}
		if strings.HasPrefix(col.DataType, "character") && (lenCol > 0) &&
			!strings.Contains(colDefine, fmt.Sprintf("char(%d)", lenCol)) {

			flags = append(flags, dbEngine.ChgLength)
		}
	} else {
		logs.DebugLog("Dif types of col '%s': '%s-%s(%d)' <-> '%s'", col.name, col.DataType, udtName, lenCol, colDefine)
		flags = append(flags, dbEngine.ChgType)
	}

	isNotNull := strings.Contains(colDefine, isNotNullable)
	if col.isNullable && isNotNull {
		flags = append(flags, dbEngine.MustNotNull)
	} else if !col.isNullable && !isNotNull {
		flags = append(flags, dbEngine.Nullable)
	}

	colDef, hasDefault := col.Default().(string)
	if newDef := dbEngine.RegDefault.FindStringSubmatch(strings.ToLower(colDefine)); len(newDef) > 0 && (!hasDefault || strings.ToLower(colDef) != strings.Trim(newDef[1], "'\n")) {
		flags = append(flags, dbEngine.ChgDefault)
	}

	return
}

func isTypeValid(colDefine string, dataType, udtName string) bool {
	if strings.HasPrefix(colDefine, dataType) || strings.HasPrefix(colDefine, udtName) {
		return true
	}

	aliases, ok := dataTypeAlias[udtName]
	if ok {
		return slices.ContainsFunc(aliases,
			func(alias string) bool {
				return strings.HasPrefix(colDefine, alias)
			})
	} else {
		for name, aliases := range dataTypeAlias {
			if slices.Contains(aliases, udtName) && strings.HasPrefix(colDefine, name) {
				return true
			}
		}
	}
	return false
}

// CharacterMaximumLength return max of length text columns
func (col *Column) CharacterMaximumLength() int {
	return col.characterMaximumLength
}

// Comment of column
func (col *Column) Comment() string {
	return col.comment
}

// Name of column
func (col *Column) Name() string {
	return col.name
}

// Primary return true if column is primary key
func (col *Column) Primary() bool {
	return col.PrimaryKey
}

// Type of column (psql native)
func (col *Column) Type() string {
	return col.UdtName
}

// IsArray of column (psql native)
func (col *Column) IsArray() bool {
	return strings.HasPrefix(col.UdtName, "_")
}

// Required return true if column need a value
func (col *Column) Required() bool {
	return !col.isNullable && (col.colDefault == nil)
}

// SetNullable set nullable flag of column
func (col *Column) SetNullable(f bool) {
	col.isNullable = f
}

// Default return default value of column
func (col *Column) Default() any {
	return col.colDefault
}

// SetDefault set default value into column
func (col *Column) SetDefault(d any) {
	str, ok := d.(string)
	if !ok {
		col.colDefault = nil
		return
	}

	if !(strings.HasPrefix(str, "(") && strings.HasSuffix(str, ")")) {
		str = (strings.Split(str, "::"))[0]

		if str == "NULL" {
			col.colDefault = nil
			return
		}
	}

	const DEFAULT_SERIAL = "nextval("
	isSerial := strings.HasPrefix(str, DEFAULT_SERIAL)
	if isSerial {
		col.colDefault = strings.Trim(strings.TrimPrefix(str, DEFAULT_SERIAL), "'")
	} else {
		col.colDefault = strings.Trim(str, "'")
	}
	// todo add other case of autogenerate column value
	upperS := strings.ToUpper(str)
	col.autoInc = isSerial ||
		strings.Contains(upperS, "CURRENT_TIMESTAMP") ||
		strings.Contains(upperS, "CURRENT_DATE") ||
		strings.Contains(upperS, "CURRENT_USER") ||
		strings.Contains(upperS, "NOW()")
}

// RefColValue referral of column property 'name'
func (col *Column) RefColValue(name string) any {
	switch name {
	case "data_type":
		return &col.DataType
	case "column_name":
		return &col.name
	case "column_default":
		return &col.colDefault
	case "is_nullable":
		return &col.isNullable
	case "character_set_name":
		return &col.CharacterSetName
	case "character_maximum_length":
		return &col.characterMaximumLength
	case "udt_name":
		return &col.UdtName
	case "column_comment":
		return &col.comment
	case "keys":
		return &col.Constraints
	case "ordinal_position":
		return &col.Position
	default:
		panic("not implement scan for field " + name)
	}

}
