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
	name := strings.Title(table.Name())
	f, err := os.Create(path.Join(c.dst, table.Name()) + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	caseRefFields, caseColFields, packages := "", "", ""
	for _, col := range table.Columns() {
		bTypeCol := col.BasicType()
		typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())

		if col.IsNullable() {
			if bTypeCol == types.UnsafePointer {
				typeCol = "interface{}"
			} else {
				typeCol = "sql.Null" + strings.Title(typeCol)
				if !strings.Contains(packages, `"sql"`) {
					packages += `"sql"
`
				}
			}
		}

		// todo add import for types
		if bTypeCol < 0 {
			switch col.Type() {
			case "json":
				typeCol = "interface{}"
			case "date", "timestampt", "timestamptz", "time":
				typeCol = "time.Time"
				if !strings.Contains(packages, `"time"`) {
					packages += `"time"
`
				}
			case "timerange", "tsrange", "_date", "_timestampt", "_timestamptz", "_time":
				typeCol = "[]time.Time"
				if !strings.Contains(packages, `"time"`) {
					packages += `"time"
`
				}
			}
		} else if bTypeCol == 0 {
			typeCol = "sql.RawBytes"
			if !strings.Contains(packages, `"sql"`) {
				packages += `"sql"
`
			}
		}

		if strings.HasPrefix(col.Type(), "_") {
			typeCol = "[]" + typeCol
		}

		propName := strcase.ToCamel(col.Name())
		_, err = fmt.Fprintf(f, colFormat, propName, typeCol, strings.ToLower(col.Name()))
		caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
		caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
	}

	_, err = fmt.Fprintf(f, title, packages)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, typeTitle, name)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name())

	return err
}
