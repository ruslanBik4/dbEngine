// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"fmt"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/typesExt"
	"go/types"
	"os"
	"path"
	"strings"

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
}

// NewCreator create with destination directory 'dst'
func NewCreator(dst string) (*Creator, error) {
	err := os.Mkdir(dst, os.ModePerm)

	if err != nil && !os.IsExist(err) {
		return nil, errors.Wrap(err, "mkDirAll")
	}

	return &Creator{dst: dst}, nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) MakeInterfaceDB(DB *dbEngine.DB) error {
	f, err := os.Create(path.Join(c.dst, "database") + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	c.packages += c.addImport("github.com/jackc/pgconn")
	c.packages += c.addImport("strings")
	c.packages += c.addImport("fmt")

	sql := ""
	_, err = fmt.Fprintf(f, title, DB.Name, DB.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, formatDatabase, DB.Name, DB.Schema)
	if err != nil {
		return errors.Wrap(err, "WriteString Database")
	}

	for _, name := range append(DB.FuncsAdded, DB.FuncsReplaced...) {
		r, ok := DB.Routines[name].(*psql.Routine)
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
	if toType == types.Invalid {
		sType = "*" + strcase.ToCamel(r.UdtName)
	}

	return "res " + sType + ", ", "&res,"
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

func (c *Creator) prepareParams(r *psql.Routine, name string) (sParams string, sParamsTitle string, args []interface{}) {
	args = make([]interface{}, len(r.Params()))
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
func (c *Creator) MakeStruct(DB *dbEngine.DB, table dbEngine.Table) error {
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

	_, err = fmt.Fprintf(f, title, DB.Name, DB.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	//todo: add Type to interface Table
	_, err = fmt.Fprintf(f, formatTable, name, fields, table.Name(), table.Comment(), table.(*psql.Table).Type)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), c.initValues)

	_, err = fmt.Fprintf(f, formatType, name)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprint(f, sTypeField)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, formatEnd, name)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	return err
}

func (c *Creator) chkTypes(col dbEngine.Column, propName string) (string, interface{}) {
	bTypeCol := col.BasicType()
	defValue := col.Default()
	typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())

	switch {
	case bTypeCol == types.Invalid:
		typeCol = "sql.RawBytes"
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

	case col.IsNullable():
		if bTypeCol == types.UnsafePointer || bTypeCol == types.Invalid {
			typeCol = "interface{}"
		} else {
			typeCol = "sql.Null" + strings.Title(typeCol)
			c.packages += c.addImport(moduloSql)
		}
	}

	if strings.HasPrefix(col.Type(), "_") ||
		strings.HasSuffix(col.Type(), "[]") {
		typeCol = "[]" + typeCol
	}

	return typeCol, defValue
}

func (c *Creator) getFuncForDecode(col dbEngine.Column, propName string, ind int) string {
	const decodeFncTmp = `psql.Get%sFromByte(ci, srcPart[%d], "%s")`
	bTypeCol := col.BasicType()
	typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())

	switch {
	case bTypeCol == types.Invalid:
		return fmt.Sprintf(decodeFncTmp, "RawBytes", ind, propName)

	case bTypeCol == types.UntypedFloat:
		switch col.Type() {
		case "numeric", "decimal":
			return fmt.Sprintf(decodeFncTmp, "Numeric", ind, propName)
		default:
			return fmt.Sprintf(decodeFncTmp, col.Type(), ind, propName)
		}

	case bTypeCol < 0:
		typeCol = c.getTypeCol(col)

		return fmt.Sprintf(decodeFncTmp, typeCol, ind, propName)

	//case col.IsNullable():
	//	if bTypeCol == types.UnsafePointer || bTypeCol == types.Invalid {
	//		typeCol = "interface{}"
	//	} else {
	//		typeCol = "sql.Null" + strings.Title(typeCol)
	//		c.packages += c.addImport(moduloSql)
	//	}
	default:

		if strings.HasPrefix(col.Type(), "_") || strings.HasSuffix(col.Type(), "[]") {
			return fmt.Sprintf(decodeFncTmp, "Array"+strcase.ToCamel(typeCol), ind, propName)
		}

		if col.IsNullable() {
			titleType := strings.Title(typeCol)
			return "sql.Null" + titleType + `{
` + titleType + ":" + fmt.Sprintf(decodeFncTmp, strcase.ToCamel(typeCol), ind, propName) + `,
}`

		}

		return fmt.Sprintf(decodeFncTmp, strcase.ToCamel(typeCol), ind, propName)
	}
}

func (c *Creator) getTypeCol(col dbEngine.Column) string {
	switch typeName := col.Type(); typeName {
	case "inet", "interval":
		c.packages += c.addImport(moduloPgType)
		return strings.Title(typeName)

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
	"Json":      "interface{}",
	"jsonb":     "interface{}",
	"RefTime":   "*time.Time",
	"Time":      "time.Time",
	"ArrayTime": "[]time.Time",
	"Interface": "interface{}",
}

func (c *Creator) addImport(moduloName string) string {
	if strings.Contains(c.packages, moduloName) {
		return ""
	}

	return `"` + moduloName + `"
	`
}
