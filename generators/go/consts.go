// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

const (
	title = `// generate file
// don't edit
package db

import (
	"database/sql"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

`
	typeTitle = `type %s struct {
	dbEngine.Table
`
	colFormat = `
	%s %s`
	caseFormat = `
		case "%s":
			%s.%s
`
	footer = `
}

func New%s( db *dbEngine.DB) *%[1]s {
	table, ok := db.Tables["%s"]
    if !ok {
      return nil
    }
    return &%[1]s{
		Table: table,
    }
}

func (t *%[1]s) GetFields(columns []dbEngine.Column) []interface{} {
	if len(columns) == 0 {
		columns = t.Columns()
	}

	v := make([]interface{}, len(columns))
	for i, col := range columns {
		switch name := col.Name(); name { %[3]s
		default:
			panic("not implement scan for field " + name)
		}
	}

	return v
}`
)
