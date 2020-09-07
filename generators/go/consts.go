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
	dbEngine.Table
	Record *%[1]sFields
	rows sql.Rows
}

type %[1]sFields struct {
`
	colFormat     = "\n\t%-15s\t%s\t`json:\"%s\"`"
	caseRefFormat = `
	case "%s":
		return &r.%s
`
	caseColFormat = `
	case "%s":
		return r.%s
`
	footer = `
}

func (r *%sFields) RefColValue(name string) interface{}{
	switch name {	%s
   	default:
		return nil
	}
}

func (r *%[1]sFields) ColValue(name string) interface{}{
	switch name {	%[3]s
   	default:
		return nil
	}
}

func New%[1]s( db *dbEngine.DB) (*%[1]s, error) {
	table, ok := db.Tables["%[4]s"]
    if !ok {
      return nil, dbEngine.ErrNotFoundTable{Table: "%[4]s"}
    }

    return &%[1]s{
		Table: table,
    }, nil
}

func (t *%[1]s) NewRecord() *%[1]sFields{
   t.Record = &%[1]sFields{}
	return t.Record
}

func (t *%[1]s) GetFields(columns []dbEngine.Column) []interface{} {
	if len(columns) == 0 {
		columns = t.Columns()
	}

	t.NewRecord()
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		v[i] = t.Record.RefColValue( col.Name() )
	}

	return v
}
	
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func(record *%[1]sFields) error, Options ...dbEngine.BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, 
			func() error {
			 	if each != nil {
					return each(t.Record)
				}

				 return nil
			}, t, Options ... )
}

func (t *%[1]s) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, len(t.Columns()))
		columns := make([]string, len(t.Columns()))
		for i, col := range t.Columns() {
			columns[i] = col.Name()
			v[i] = t.Record.ColValue( col.Name() )
		}
		Options = append(Options, 
			dbEngine.ColumnsForSelect(columns...), 
			dbEngine.ArgsForSelect(v...) )
	}

	return t.Table.Insert(ctx, Options...)
}

func (t *%[1]s) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, len(t.Columns()))
		priV := make([]interface{}, 0)
		columns := make([]string, 0, len(t.Columns()))
		priColumns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV[len(priColumns)-1] = t.Record.ColValue( col.Name() )
				continue
			}

			columns = append( columns, col.Name() )
			v[len(columns)-1] = t.Record.ColValue( col.Name() )
		}

		Options = append(
			Options, 
			dbEngine.ColumnsForSelect(columns...), 
			dbEngine.WhereForSelect(priColumns...), 
			dbEngine.ArgsForSelect(append(v, priV...)... ),
		)
	}

	return t.Table.Update(ctx, Options...)
}
`
)
