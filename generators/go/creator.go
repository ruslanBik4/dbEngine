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

func (c *Creator) MakeStruct(table dbEngine.Table) error {
	logs.SetDebug(true)
	name := strcase.ToCamel(table.Name())
	f, err := os.Create(path.Join(c.dst, table.Name()) + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	fields, caseRefFields, caseColFields, packages := "", "", "", ""
	for _, col := range table.Columns() {
		bTypeCol := col.BasicType()
		typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())

		switch {
		case bTypeCol == 0:
			typeCol = "sql.RawBytes"
			packages += c.addImport(packages, "database/sql")

		case bTypeCol < 0:
			switch col.Type() {
			case "inet":
				typeCol = "pgtype.Inet"
				packages += c.addImport(packages, "github.com/jackc/pgtype")

			case "json":
				typeCol = "interface{}"

			case "date", "timestampt", "timestamptz", "time":
				if col.IsNullable() {
					typeCol = "sql.NullTime"
					packages += c.addImport(packages, "database/sql")
				} else {
					typeCol = "time.Time"
					packages += c.addImport(packages, "time")
				}

			case "timerange", "tsrange", "_date", "_timestampt", "_timestamptz", "_time":
				typeCol = "[]time.Time"
				packages += c.addImport(packages, "time")
			default:
				typeCol = "interface{}"
			}

		case col.IsNullable():
			if bTypeCol == types.UnsafePointer || bTypeCol == types.Invalid {
				typeCol = "interface{}"
			} else {
				typeCol = "sql.Null" + strings.Title(typeCol)
				packages += c.addImport(packages, "database/sql")
			}
		}

		if strings.HasPrefix(col.Type(), "_") {
			typeCol = "[]" + typeCol
		}

		propName := strcase.ToCamel(col.Name())
		fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	_, err = fmt.Fprintf(f, title, packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, typeTitle, name, fields)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name())

	return err
}

func (c *Creator) addImport(packages, moduloName string) string {
	if !strings.Contains(packages, moduloName) {
		return `"` + moduloName + `"
	`
	}

	return ""
}
