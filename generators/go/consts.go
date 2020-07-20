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
	Record *%[1]sFields
	dbEngine.Table
	rows sql.Rows
}

type %[1]sFields struct {
`
	colFormat  = "\n\t%-15s\t%s\t`json:\"%s\"`"
	caseFormat = `
	case "%s":
		return &t.Record.%s
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

func (t *%[1]s) NewRecord() *%[1]sFields{
   t.Record = &%[1]sFields{}
	return t.Record
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

	t.NewRecord()
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		v[i] = t.GetColValue( col.Name() )
	}

	return v
}
	
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func(record *%[1]sFields) error, Options ...dbEngine.BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, 
			 func() error {
				return each(t.Record)
			}, t, Options ... )
}

func (t *Table) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, len(columns))
		columns := make([]interface{}, len(columns))
		for i, col := range t.Columns() {
			columns[i] = col.Name()
			v[i] = t.GetColValue( col.Name() )
		}
		Options = append(Options, dbEngine.ColumnsForSelect(columns), dbEngine.ArgsForSelect(v) )
	}

	return t.Insert(ctx, Options...)
}

func (t *Table) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, len(columns))
		priV := make([]interface{}, 0)
		columns := make([]interface{}, 0, len(columns))
		priColumns := make([]interface{}, 0, len(columns))
		for i, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV[len(priColumns-1)] = t.GetColValue( col.Name() )
				continue
			}
			columns = append( columns, col.Name() )
			v[len(column-1)] = t.GetColValue( col.Name() )
		}

		Options = append(
			Options, 
			dbEngine.ColumnsForSelect(columns), 
			dbEngine.WhereForSelect(priColumns), 
			dbEngine.ArgsForSelect(append(v, priV...) ),
		)
	}

	return t.Insert(ctx, Options...)
}
`
)
