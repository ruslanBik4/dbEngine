// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"fmt"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/typesExt"
	"go/types"
	"golang.org/x/net/context"
	"os"
	"path"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type Creator struct {
	dst        string
	packages   string
	initValues string
}

func NewCreator(dst string) (*Creator, error) {
	err := os.Mkdir(dst, os.ModePerm)

	if err != nil && !os.IsExist(err) {
		return nil, errors.Wrap(err, "mkDirAll")
	}

	return &Creator{dst: dst}, nil
}

func Go_get_corom(ctx context.Context, DB *dbEngine.DB) (int64, error) {
	var res int64
	err := DB.Conn.SelectOneAndScan(ctx, &res, "")
	if err != nil {
		return res, err
	}

	return res, nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) MakeInterfaceDB(DB *dbEngine.DB) error {
	f, err := os.Create(path.Join(c.dst, "database") + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	_, err = fmt.Fprintf(f, title, DB.Name, DB.Schema, c.packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, DBformat, DB.Name, DB.Schema)
	if err != nil {
		return errors.Wrap(err, "WriteString Database")
	}

	for _, name := range append(DB.FuncsAdded, DB.FuncsReplaced...) {
		r := DB.Routines[name].(*psql.Routine)
		if r.ReturnType() != "trigger" {
			logs.StatusLog(name, r.ReturnType())
			name := strcase.ToCamel(name)
			sParams, sParamsTitle := "", ""
			args := make([]interface{}, len(r.Params()))
			for i, param := range r.Params() {
				typeCol, _ := c.chkTypes(param, name)
				s := strcase.ToCamel(param.Name())
				sParamsTitle += ", " + s + " " + typeCol
				sParams += ", " + s
				args[i] = param
			}
			if r.Type == psql.ROUTINE_TYPE_PROC {
				sql, _, err := r.BuildSql(dbEngine.ArgsForSelect(args...))
				if err == nil {
					_, err = fmt.Fprintf(f, callProcFormat, name, sParamsTitle,
						sql, sParams, r.Comment)
				}

			} else {
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
				if len(r.Columns()) > 1 {
					sResult = `
			[]interface{}{
					` + sResult + "\n},\n"
				} else if len(r.Columns()) == 0 {

					toType := psql.UdtNameToType(r.UdtName)
					sType := typesExt.Basic(toType).String()
					if toType == types.Invalid {
						sType = "*" + strcase.ToCamel(r.UdtName)
					}
					sReturn = "res " + sType + ", "
					sResult = "&res,"
				}
				sql, _, err := r.BuildSql(dbEngine.ArgsForSelect(args...))
				if err == nil {
					_, err = fmt.Fprintf(f, newFuncFormat, name, sParamsTitle, sReturn, sResult,
						sql, sParams, r.Comment)
				}
			}
			if err != nil {
				return errors.Wrap(err, "WriteString Database")
			}
		}

	}
	return err
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

	c.packages, c.initValues = "", ""
	fields, caseRefFields, caseColFields := "", "", ""
	for _, col := range table.Columns() {
		propName := strcase.ToCamel(col.Name())

		typeCol, defValue := c.chkTypes(col, propName)

		if strings.HasPrefix(col.Type(), "_") {
			typeCol = "[]" + typeCol
		}

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

	_, err = fmt.Fprintf(f, typeTitle, name, fields, table.Name())
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

		case "date", "timestampt", "timestamptz", "time":
			if col.IsNullable() {
				typeCol = "*time.Time"
				c.initValues += fmt.Sprintf(initFormat, propName, "&time.Time{}")
			} else {
				typeCol = "time.Time"
			}

		case "timerange", "tsrange", "_date", "daterange", "_timestampt", "_timestamptz", "_time":
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

	return typeCol, defValue
}

func (c *Creator) addImport(moduloName string) string {
	if !strings.Contains(c.packages, moduloName) {
		return `"` + moduloName + `"
	`
	}

	return ""
}
