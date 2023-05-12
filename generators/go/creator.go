// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"context"
	"fmt"
	"go/types"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/jackc/pgtype"

	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/typesExt"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// Creator is interface for generate go-interface according to DB structures (tables & routines)
type Creator struct {
	Types      map[string]string
	dst        string
	packages   []string
	initValues string
	db         *dbEngine.DB
}

// NewCreator create with destination directory 'dst'
func NewCreator(dst string, DB *dbEngine.DB) (*Creator, error) {
	if DB == nil {
		return nil, dbEngine.ErrDBNotFound
	}
	err := os.Mkdir(dst, os.ModePerm)

	if err != nil && !os.IsExist(err) {
		return nil, errors.Wrap(err, "mkDirAll")
	}

	return &Creator{
		db:  DB,
		dst: dst,
	}, nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) makeDBUsersTypes() error {
	for tName, t := range c.db.Types {
		for i, tAttr := range t.Attr {
			name := tAttr.Name
			propName := strcase.ToCamel(name)
			typeCol, _ := c.chkTypes(
				&psql.Column{
					UdtName:     tAttr.Type,
					DataType:    tAttr.Type,
					UserDefined: &t,
				},
				propName)
			if typeCol == "" {
				logs.ErrorLog(dbEngine.NewErrNotFoundType(name, tAttr.Type), tAttr)
			}
			tAttr.Type = typeCol
			t.Attr[i] = tAttr
			if len(t.Enumerates) == 0 {
				c.addImport("bytes", moduloPgType)
			}
		}
		c.db.Types[tName] = t
	}
	return nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) MakeInterfaceDB() error {
	f, err := os.Create(path.Join(c.dst, "database") + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	err = c.makeDBUsersTypes()
	if err != nil {
		return err
	}
	routines := make([]string, 0, len(c.db.Routines))
	for name := range c.db.Routines {
		routines = append(routines, name)
	}
	sort.Strings(routines)
	c.WriteCreateDatabase(f, routines)

	return err
}

// when type is tables record or DB  type
func (c *Creator) chkDefineType(udtName string) string {
	isArray := strings.HasPrefix(udtName, "_") || strings.HasSuffix(udtName, "[]")
	prefix := ""
	if isArray {
		udtName = strings.TrimPrefix(udtName, "_")
		udtName = strings.TrimSuffix(udtName, "[]")
		prefix = "[]"
	}

	if _, ok := c.db.Tables[udtName]; ok {
		return fmt.Sprintf("%s%sFields", prefix, strcase.ToCamel(udtName))
	}

	if t, ok := c.db.Types[udtName]; ok {
		if len(t.Enumerates) > 0 {
			return prefix + "string"
		}
		return fmt.Sprintf("%s%s", prefix, strcase.ToCamel(udtName))
	}

	return ""
}

func (c *Creator) prepareReturn(r *psql.Routine) (string, string) {
	toType := psql.UdtNameToType(r.UdtName)
	sType := typesExt.Basic(toType).String()
	switch toType {
	case types.Invalid:
		sType = c.chkDefineType(r.UdtName)
		if sType == "" {
			sType = "*" + strcase.ToCamel(r.UdtName)
		}

	case types.UntypedFloat:
		sType = "float64"
	default:
	}

	return "res " + sType + ", ", "&res,"
}

func (c *Creator) prepareReturns(r *psql.Routine, name string) (sRecord string, sReturn string, sResult string) {
	if len(r.Columns()) == 0 {
		sReturn, sResult = c.prepareReturn(r)
	}
	for _, col := range r.Columns() {
		typeCol, _ := c.chkTypes(col, name)
		if typeCol == "" {
			logs.ErrorLog(dbEngine.NewErrNotFoundType(col.Type(), col.Name()), name)
		}
		s := strings.Trim(col.Name(), "_")
		if s == "type" {
			s += "_"
		}
		sReturn += s + " " + typeCol + ", "
		sResult += "&" + s + ", "
	}
	if len(r.Columns()) > 1 {
		sRecord = strings.ReplaceAll(strings.TrimSuffix(sResult, ","), "&", "&r.")
		sResult = fmt.Sprintf(paramsFormat, sResult)
	}
	return
}

func (c *Creator) prepareParams(r *psql.Routine, name string) (sParams string, sParamsTitle string) {
	// args = make([]any, len(r.Params()))
	for _, param := range r.Params() {
		typeCol, _ := c.chkTypes(param, name)
		s := strcase.ToLowerCamel(param.Name())
		if param.Default() == nil {
			typeCol = "*" + typeCol
		}
		sParamsTitle += ", " + s + " " + typeCol
		sParams += s + `, `
		// args[i] = param
	}

	if sParams > "" {
		sParams += `
			`
	}
	return
}

// MakeStruct create table interface with Columns operations
func (c *Creator) MakeStruct(table dbEngine.Table) error {
	logs.SetDebug(true)
	name := strcase.ToCamel(table.Name())
	f, err := os.Create(path.Join(c.dst, table.Name()) + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	defer func() {
		err := f.Close()
		if err != nil {
			logs.ErrorLog(err)
		}
	}()

	c.packages, c.initValues = make([]string, 0), ""
	c.addImport(moduloPgType, "bytes", "sync")
	fields, caseRefFields, caseColFields, sTypeField := "", "", "", ""
	for ind, col := range table.Columns() {
		propName := strcase.ToCamel(col.Name())

		typeCol, defValue := c.chkTypes(col, propName)

		if !col.AutoIncrement() && defValue != nil {
			def, ok := defValue.(string)
			if ok {
				if typeCol == "string" {
					c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf(`"%s"`, def))
				}
			} else {
				c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("%v", defValue))
			}
		}

		sTypeField += fmt.Sprintf(scanFormat,
			c.GetFuncForDecode(&dbEngine.TypesAttr{
				Name:      col.Name(),
				Type:      typeCol,
				IsNotNull: false,
			}, ind))

		fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	packages := ""
	if len(c.packages) > 0 {
		packages = `"` + strings.Join(c.packages, `"
	"`) + `"`
	}
	_, err = fmt.Fprintf(f, title, c.db.Name, c.db.Schema, packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	// todo: add Type to interface Table
	_, err = fmt.Fprintf(f, formatTable, name, fields, table.Name(), table.Comment(), table.(*psql.Table).Type)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), c.initValues)

	_, err = fmt.Fprintf(f, formatType, name, sTypeField)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	return err
}

func (c *Creator) chkTypes(col dbEngine.Column, propName string) (string, any) {
	bTypeCol := col.BasicType()
	defValue := col.Default()
	typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())
	isArray := strings.HasPrefix(col.Type(), "_") || strings.HasSuffix(col.Type(), "[]")

	switch {
	case bTypeCol == types.UnsafePointer:
		typeCol = "[]byte"

	case bTypeCol == types.Invalid || bTypeCol < 0:
		typeCol = c.chkDefineType(col.Type())
		if typeCol == "" {
			name, ok := c.chkDataType(col.Type())
			if ok {
				typeCol = strings.TrimPrefix(fmt.Sprintf("%T", name.Value), "*")
			} else {
				logs.StatusLog(typeCol, col.Type())
				typeCol = "sql.RawBytes"
				c.addImport(moduloSql)
			}
		}
		if strings.HasPrefix(typeCol, "*") {
			c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("&%s{}", strings.TrimPrefix(typeCol, "*")))
			defValue = nil
		}

	case bTypeCol == types.UntypedFloat:
		switch col.Type() {
		case "numeric", "decimal":
			typeCol = "psql.Numeric"
			if defValue != nil {
				c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("psql.NewNumericFromFloat64(%v)", defValue))
				// prevent finally check default
				defValue = nil
			} else {
				c.initValues += fmt.Sprintf(initFormat, propName, "psql.NewNumericNull()")
			}
		}

	case isArray:
		typeCol = "[]" + typeCol

	case col.IsNullable():
		typeCol = "sql.Null" + strcase.ToCamel(typeCol)
		c.addImport(moduloSql)
	default:
	}

	return typeCol, defValue
}

func (c *Creator) chkDataType(typeCol string) (*pgtype.DataType, bool) {
	conn, err := c.db.Conn.(*psql.Conn).Acquire(context.TODO())
	if err != nil {
		logs.ErrorLog(err)
		return nil, false
	}
	defer conn.Release()
	return conn.Conn().ConnInfo().DataTypeForName(typeCol)
}

func (c *Creator) GetFuncForDecode(tAttr *dbEngine.TypesAttr, ind int) string {
	tName, name := tAttr.Type, tAttr.Name
	switch _, isTypes := c.db.Types[strings.ToLower(tName)]; {
	case strings.HasPrefix(tName, "sql.Null"):
		return fmt.Sprintf(
			`%-21s:	*(psql.GetScanner(ci, srcPart[%d], "%s", &%s{}))`,
			strcase.ToCamel(name),
			ind,
			name,
			tName)

	case strings.HasPrefix(tName, "pgtype.") || strings.HasPrefix(tName, "psql.") || isTypes:
		return fmt.Sprintf(
			`%-21s:	*(psql.GetTextDecoder(ci, srcPart[%d], "%s", &%s{}))`,
			strcase.ToCamel(name),
			ind,
			name,
			tName)

	case strings.HasPrefix(tName, "[]"):
		tName = "Array" + strcase.ToCamel(strings.TrimPrefix(tName, "[]"))
	default:
		tName = strcase.ToCamel(tName)
	}

	return fmt.Sprintf(`%-21s:	psql.Get%sFromByte(ci, srcPart[%d], "%s")`,
		strcase.ToCamel(name),
		tName,
		ind,
		name)
}
func (c *Creator) udtToReturnType(udtName string) string {
	toType := psql.UdtNameToType(udtName)
	switch toType {
	case types.UnsafePointer:
		return "[]byte"
	case types.Invalid, typesExt.TMap, typesExt.TStruct:
		typeReturn := c.chkDefineType(udtName)
		if typeReturn == "" {
			name, ok := c.chkDataType(udtName)
			if ok {
				typeReturn = fmt.Sprintf("%T", name.Value)
			} else {
				typeReturn = "*" + strcase.ToCamel(udtName)
			}
		}
		return typeReturn

	case types.UntypedFloat:
		return "float64"

	default:
		s := typesExt.Basic(toType).String()
		if s == "" {
			logs.StatusLog(udtName)
		}
		return s
	}
}

func (c *Creator) getTypeCol(col dbEngine.Column) string {
	switch typeName := col.Type(); typeName {
	case "inet", "interval":
		c.addImport(moduloPgType)
		return strcase.ToCamel(typeName)

	case "json", "jsonb":
		return "Json"

	case "date", "timestamp", "timestamptz", "time":
		if col.IsNullable() {
			return "RefTime"
		} else {
			return "Time"
		}

	case "timerange", "tsrange", "_date", "daterange", "_timestamp", "_timestamptz", "_time":
		return "ArrayTime"
	default:
		return "Any"
	}
}

var mapTypes = map[string]string{
	"Inet":      "pgtype.Inet",
	"Interval":  "pgtype.Interval",
	"Json":      "any",
	"jsonb":     "any",
	"RefTime":   "*time.Time",
	"Time":      "time.Time",
	"ArrayTime": "[]time.Time",
	"Any":       "any",
}

func (c *Creator) addImport(moduloNames ...string) {
	for _, name := range moduloNames {
		isAlready := false
		for _, n := range c.packages {
			if n == name {
				isAlready = true
				break
			}
		}
		if !isAlready {
			c.packages = append(c.packages, name)
		}

	}
}
