// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"fmt"
	"go/types"
	"os"
	"path"
	"strings"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"

	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/typesExt"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// Creator is interface for generate go-interface according to DB structires (tavles & routibes)
type Creator struct {
	dst        string
	packages   string
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
func (c *Creator) makeDBUserTypes(f *os.File) error {
	for name, t := range c.db.Types {
		initValues := ""
		for name, tName := range t.Attr {
			propName := strcase.ToCamel(name)
			typeCol, _ := c.chkTypes(&psql.Column{UdtName: tName}, propName)
			initValues += fmt.Sprintf(colFormat, propName, typeCol, name)

		}
		_, err := fmt.Fprintf(f, newTypeInterface, strcase.ToCamel(name), name, initValues)
		if err != nil {
			return errors.Wrap(err, "WriteNewTable of Database")
		}

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

	c.packages += c.addImport("github.com/jackc/pgconn")
	c.packages += c.addImport("strings")
	c.packages += c.addImport("fmt")

	sql := ""
	_, err = fmt.Fprintf(f, title, c.db.Name, c.db.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	err = c.makeDBUserTypes(f)
	if err != nil {
		return err
	}
	_, err = fmt.Fprintf(f, formatDatabase, c.db.Name, c.db.Schema)
	if err != nil {
		return errors.Wrap(err, "WriteString Database")
	}

	for name := range c.db.Tables {
		_, err = fmt.Fprintf(f, newTableInstance, strcase.ToCamel(name), name)
		if err != nil {
			return errors.Wrap(err, "WriteNewTable of Database")
		}
	}
	for _, name := range append(c.db.FuncsAdded, c.db.FuncsReplaced...) {
		r, ok := c.db.Routines[name].(*psql.Routine)
		if !ok {
			logs.ErrorLog(dbEngine.ErrNotFoundRoutine{
				Name:  name,
				SName: "",
			})
			continue
		}

		if r.ReturnType() == "trigger" {
			continue
		}

		logs.StatusLog(r.Type, name, r.ReturnType())
		camelName := strcase.ToCamel(name)
		sParams, sParamsTitle, args := c.prepareParams(r, camelName)
		if r.Type == psql.ROUTINE_TYPE_PROC {
			sql, _, err = r.BuildSql(dbEngine.ArgsForSelect(args...))
			if err == nil {
				_, err = fmt.Fprintf(f, callProcFormat, camelName, sParamsTitle,
					sql, sParams, name, r.Comment)
			}

		} else {
			sRecord := ""
			sReturn, sResult := c.prepareReturns(r, camelName)
			if len(r.Columns()) > 1 {
				sRecord = strings.ReplaceAll(strings.TrimSuffix(sResult, ","), "&", "&r.")
				sResult = fmt.Sprintf(paramsFormat, sResult)
			} else if len(r.Columns()) == 0 {
				sReturn, sResult = c.prepareReturn(r)
			}

			sql, _, err = r.BuildSql(dbEngine.ArgsForSelect(args...))
			if err == nil {
				_, err = fmt.Fprintf(f, newFuncFormat, camelName, sParamsTitle, sReturn, sResult,
					sql, sParams, name, r.Comment)
				if r.ReturnType() == "record" {
					_, err = fmt.Fprintf(f, newFuncRecordFormat, camelName,
						strings.ReplaceAll(sReturn, ",", "\n\t\t"),
						sParamsTitle,
						sRecord,
						sql, sParams, name, r.Comment)
				}
			}
		}
		if err != nil {
			return errors.Wrap(err, "WriteString Function")
		}
	}

	_, err = f.WriteString(formatDBprivate)

	return err
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

// when type is tables record or DB  type
func (c *Creator) chkDefineType(udtName string) string {
	isArray := strings.HasPrefix(udtName, "_") || strings.HasSuffix(udtName, "[]")
	prefix := ""
	if isArray {
		udtName = strings.TrimPrefix(udtName, "_")
		udtName = strings.TrimSuffix(udtName, "[]")
		prefix = "[] "
		logs.StatusLog(prefix, udtName)
	}
	for name := range c.db.Tables {
		if name == udtName {
			return fmt.Sprintf("%s%sFields", prefix, strcase.ToCamel(udtName))
		}
	}
	for name := range c.db.Types {
		if name == udtName {
			return fmt.Sprintf("%s%s", prefix, strcase.ToCamel(udtName))
		}
	}
	return ""
}

func (c *Creator) prepareReturns(r *psql.Routine, name string) (string, string) {
	sReturn, sResult := "", ""
	for _, col := range r.Columns() {
		typeCol, _ := c.chkTypes(col, name)
		if typeCol == "" {
			logs.StatusLog(col.BasicType())
		}
		s := strings.Trim(col.Name(), "_")
		if s == "type" {
			s += "_"
		}
		sReturn += s + " " + typeCol + ", "
		sResult += "&" + s + ", "
	}

	return sReturn, sResult
}

func (c *Creator) prepareParams(r *psql.Routine, name string) (sParams string, sParamsTitle string, args []any) {
	args = make([]any, len(r.Params()))
	for i, param := range r.Params() {
		typeCol, _ := c.chkTypes(param, name)
		s := strcase.ToLowerCamel(param.Name())
		if param.Default() == nil {
			typeCol = "*" + typeCol
		}
		sParamsTitle += ", " + s + " " + typeCol
		sParams += s + `, `
		args[i] = param
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

	//fileType, err := os.Create(path.Join(c.dst, table.Name()) + "_types.go")
	//if err != nil && !os.IsExist(err) {
	//	// err.(*os.PathError).Err
	//	return errors.Wrap(err, "creator")
	//}
	//
	//defer fileType.Close()
	//
	//_, err = fmt.Fprintf(fileType, title, DB.Name, DB.Schema, c.packages)
	//if err != nil {
	//	return errors.Wrap(err, "WriteString title")
	//}

	c.packages, c.initValues = `"sync"
`, ""
	c.packages += c.addImport(moduloPgType)
	c.packages += c.addImport("bytes")
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

		sTypeField += fmt.Sprintf(initFormat, propName, c.getFuncForDecode(col, propName, ind))

		fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	_, err = fmt.Fprintf(f, title, c.db.Name, c.db.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	//todo: add Type to interface Table
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

	case bTypeCol == types.Invalid:
		typeCol = c.chkDefineType(col.Type())
		if typeCol == "" {
			typeCol = "sql.RawBytes"
		}
		c.packages += c.addImport(moduloSql)

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

	case bTypeCol < 0:
		typeCol = mapTypes[c.getTypeCol(col)]
		if typeCol == "*time.Time" {
			c.initValues += fmt.Sprintf(initFormat, propName, "&time.Time{}")
			defValue = nil
		}

	case isArray:
		typeCol = "[]" + typeCol

	case col.IsNullable():
		switch bTypeCol {
		case types.Invalid:
			typeCol = "any"
		default:
			typeCol = "sql.Null" + strcase.ToCamel(typeCol)
			c.packages += c.addImport(moduloSql)
		}
	default:
	}

	return typeCol, defValue
}

func (c *Creator) getFuncForDecode(col dbEngine.Column, propName string, ind int) string {
	const decodeFncTmp = `psql.Get%sFromByte(ci, srcPart[%d], "%s")`
	bTypeCol := col.BasicType()
	typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())
	titleType := cases.Title(language.English).String(typeCol)
	isArray := strings.HasPrefix(col.Type(), "_") || strings.HasSuffix(col.Type(), "[]")
	if isArray {
		titleType = "Array" + titleType
	}

	switch {
	case bTypeCol == types.Invalid || bTypeCol == types.UnsafePointer:
		titleType = "RawBytes"
		//c.packages += c.addImport(moduloSql)

	case bTypeCol == types.UntypedFloat && (col.Type() == "numeric" || col.Type() == "decimal"):
		titleType = "Numeric"

	case bTypeCol < 0:
		titleType = c.getTypeCol(col)

	default:
		if col.IsNullable() && !isArray {
			return "sql.Null" + titleType + `{
` + titleType + ":" + fmt.Sprintf(decodeFncTmp, titleType, ind, propName) + `,
	Valid: true,
}`
		}

	}

	return fmt.Sprintf(decodeFncTmp, titleType, ind, propName)
}

func (c *Creator) getTypeCol(col dbEngine.Column) string {
	switch typeName := col.Type(); typeName {
	case "inet", "interval":
		c.packages += c.addImport(moduloPgType)
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
		return "Interface"
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
	"Interface": "any",
}

func (c *Creator) addImport(moduloName string) string {
	if strings.Contains(c.packages, moduloName) {
		return ""
	}

	return `"` + moduloName + `"
	`
}
