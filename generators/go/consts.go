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
	"golang.org/x/net/context"
)

`
	typeTitle = `type %s struct {
	*%[1]sFields
	dbEngine.Table
	rows sql.Rows
}

type %[1]sFields struct {
`
	colFormat = `
	%s %s`
	caseFormat = `
	case "%s":
		return &t.%s
`
	footer = `
}

func New%s( db *dbEngine.DB) (*%[1]s, error) {
	table, ok := db.Tables["%s"]
    if !ok {
      return nil, dbEngine.ErrNotFoundTable{Table: "%[2]s"}
    }

    return &%[1]s{
		Table: table,
    }, nil
}

func (t *%[1]s) GetNewFields() *%[1]sFields{
   return &%[1]sFields{}
}

func (t *%[1]s) GetColValue(name string) interface{}{
	switch name {	%[3]s
   	default:
		return nil
	}
}

func (t *%[1]s) GetFields(columns []dbEngine.Column) []interface{} {
	if len(columns) == 0 {
		columns = t.Columns()
	}

	t.%[1]sFields = &%[1]sFields{}
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		v[i] = t.GetColValue( col.Name() )
	}

	return v
}
	
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func() error, Options ...dbEngine.BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, each, t, Options ... )
}
`
)
