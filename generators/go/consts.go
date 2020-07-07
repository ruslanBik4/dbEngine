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
	sql.Rows
	%[1]sFields
}

type %[1]sFields struct {
`
	colFormat = `
	%s %s`
	caseFormat = `
		case "%s":
			v[i] = &%s.%s
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

	t.%[1]sFields = &%[1]sFields{}
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		switch name := col.Name(); name { %[3]s
		default:
			panic("not implement scan for field " + name)
		}
	}

	return v
}
	
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func() error, Options ...BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, each, t, Options ... )
}
`
)
