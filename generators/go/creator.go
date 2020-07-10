// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"fmt"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/ruslanBik4/httpgo/typesExt"

	"github.com/ruslanBik4/dbEngine/dbEngine"
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
	name := strings.Title(table.Name())
	f, err := os.Create(path.Join(c.dst, table.Name()) + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	_, err = f.WriteString(title)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	_, err = fmt.Fprintf(f, typeTitle, name)
	if err != nil {
		return errors.Wrap(err, "WriteString title")
	}

	caseFields := ""
	for _, col := range table.Columns() {
		bTypeCol := col.BasicType()
		typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())

		if strings.HasPrefix(col.Type(), "_") {
			typeCol = "[]" + typeCol
		} else if col.IsNullable() {
			typeCol = "sql.Null" + strings.Title(typeCol)
		}

		if bTypeCol < 0 {
			typeCol = "sql.RawBytes"
		}

		_, err = fmt.Fprintf(f, colFormat, strings.Title(col.Name()), typeCol)
		caseFields += fmt.Sprintf(caseFormat, col.Name(), strings.Title(col.Name()))
	}

	_, err = fmt.Fprintf(f, footer, name, table.Name(), caseFields)

	return err
}
