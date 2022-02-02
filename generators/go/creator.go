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

	c.packages = `"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"strings"
`
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
			sReturn, sResult := c.prepareReturns(r, camelName)
			if len(r.Columns()) > 1 {
				sResult = fmt.Sprintf(paramsFormat, sResult)
			} else if len(r.Columns()) == 0 {
				sReturn, sResult = c.prepareReturn(r)
			}

			sql, _, err = r.BuildSql(dbEngine.ArgsForSelect(args...))
			if err == nil {
				_, err = fmt.Fprintf(f, newFuncFormat, camelName, sParamsTitle, sReturn, sResult,
					sql, sParams, name, r.Comment)
			}
		}
		if err != nil {
			return errors.Wrap(err, "WriteString Function")
		}
	}

	_, err = fmt.Fprintf(f, formatDBprivate, DB.Name, DB.Schema)

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
		sParamsTitle += ", " + s + " " + typeCol
		sParams += s + `, `
		args[i] = param
	}

	if sParams > "" {
		sParams += "\n"
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

	c.packages, c.initValues = `"sync"
`, ""
	fields, caseRefFields, caseColFields := "", "", ""
	for _, col := range table.Columns() {
		propName := strcase.ToCamel(col.Name())

		typeCol, defValue := c.chkTypes(col, propName)

		if !col.AutoIncrement() && defValue != nil {
			c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("%v", defValue))
		}

		fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	_, err = fmt.Fprintf(f, title, DB.Name, DB.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, typeTitle, name, fields, table.Name(), table.Comment())
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), c.initValues)

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
		switch col.Type() {
		case "inet":
			typeCol = "pgtype.Inet"
			c.packages += c.addImport(moduloPgType)

		case "json":
			typeCol = "interface{}"

		case "date", "timestamp", "timestamptz", "time":
			if col.IsNullable() {
				typeCol = "*time.Time"
				c.initValues += fmt.Sprintf(initFormat, propName, "&time.Time{}")
			} else {
				typeCol = "time.Time"
			}

		case "timerange", "tsrange", "_date", "daterange", "_timestamp", "_timestamptz", "_time":
			typeCol = "[]time.Time"
		default:
			typeCol = "interface{}"
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

func (c *Creator) addImport(moduloName string) string {
	if strings.Contains(c.packages, moduloName) {
		return ""
	}

	return `"` + moduloName + `"
	`
}
