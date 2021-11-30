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

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/typesExt"
)

type Creator struct {
	dst string
}

func NewCreator(dst string) (*Creator, error) {
	err := os.Mkdir(dst, os.ModePerm)

	if err != nil && !os.IsExist(err) {
		return nil, errors.Wrap(err, "mkDirAll")
	}

	return &Creator{dst: dst}, nil
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

	fields, caseRefFields, caseColFields, packages, initValues := "", "", "", "", ""
	for _, col := range table.Columns() {
		bTypeCol := col.BasicType()
		typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())
		propName := strcase.ToCamel(col.Name())
		defValue := col.Default()

		switch {
		case bTypeCol == types.Invalid:
			typeCol = "sql.RawBytes"
			packages += c.addImport(packages, moduloSql)

		case bTypeCol == types.UntypedFloat:
			switch col.Type() {
			case "numeric", "decimal":
				typeCol = "psql.Numeric"
				if defValue != nil {
					initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("psql.NewNumericFromFloat64(%v)", defValue))
					// prevent finally check default
					defValue = nil
				} else {
					initValues += fmt.Sprintf(initFormat, propName, "psql.NewNumericNull()")
				}
			}

		case bTypeCol < 0:
			switch col.Type() {
			case "inet":
				typeCol = "pgtype.Inet"
				packages += c.addImport(packages, moduloPgType)

			case "json":
				typeCol = "interface{}"

			case "date", "timestampt", "timestamptz", "time":
				if col.IsNullable() {
					typeCol = "*time.Time"
					initValues += fmt.Sprintf(initFormat, propName, "&time.Time{}")
				} else {
					typeCol = "time.Time"
				}

			case "timerange", "tsrange", "_date", "_timestampt", "_timestamptz", "_time":
				typeCol = "[]time.Time"
			default:
				typeCol = "interface{}"
			}

		case col.IsNullable():
			if bTypeCol == types.UnsafePointer || bTypeCol == types.Invalid {
				typeCol = "interface{}"
			} else {
				typeCol = "sql.Null" + strings.Title(typeCol)
				packages += c.addImport(packages, moduloSql)
			}
		}

		if strings.HasPrefix(col.Type(), "_") {
			typeCol = "[]" + typeCol
		}

		if !col.AutoIncrement() && defValue != nil {
			initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("%v", defValue))
		}

		fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	_, err = fmt.Fprintf(f, title, DB.Name, DB.Schema, packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, typeTitle, name, fields, table.Name())
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), initValues)

	return err
}

func (c *Creator) addImport(packages, moduloName string) string {
	if !strings.Contains(packages, moduloName) {
		return `"` + moduloName + `"
	`
	}

	return ""
}
