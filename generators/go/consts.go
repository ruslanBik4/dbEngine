// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

const (
	moduloPgType = "github.com/jackc/pgtype"
	moduloSql    = "database/sql"
)

const (
	title = `// Code generated by dbEngine-gen-go. DO NOT EDIT.
// versions:
// 	dbEngine v1.0.1
// source: database schema 'test'
package db

import (
	"sync"
	%s
	"github.com/ruslanBik4/logs"
	"github.com/ruslanBik4/dbEngine/dbEngine"
    "github.com/ruslanBik4/dbEngine/dbEngine/psql"

	"golang.org/x/net/context"
)

`
	typeTitle = `// %s object for database operations
type %[1]s struct {
	*psql.Table
	Record *%[1]sFields
	doCopyPoll      []*FinanceMatchFields
	doCopyPoolCount int
	doCopyPoolColumns []string
	doCopyValuesCount int
	doCopyErr       error
	lock       sync.RWMutex
}

// New%[1]s create new instance of table object
func New%[1]s( db *dbEngine.DB) (*%[1]s, error) {
	table, ok := db.Tables["%[3]s"]
    if !ok {
      return nil, dbEngine.ErrNotFoundTable{Table: "%[3]s"}
    }

    return &%[1]s{
		Table: table.(*psql.Table),
    }, nil
}
// Next
func (t *%[1]s) Next() bool {
	t.doCopyValuesCount++
	return t.doCopyValuesCount < len(t.doCopyPoll)
}
// Values
func (t *%[1]s) Values() ([]interface{}, error) {
	res := make([]interface{}, len(t.doCopyPoolColumns))
	for i, col := range t.doCopyPoolColumns {
		res[i] = t.doCopyPoll[t.doCopyValuesCount].ColValue(col)
	}

	return res, nil
}
// Err
func (t *%[1]s) Err() error {
	return t.doCopyErr
}
// InitPoolCopy environments
func (t *%[1]s) InitPoolCopy(ctx context.Context, capOfPool int, d time.Duration, columns ...string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.doCopyPoll = make([]*FinanceMatchFields, 0, capOfPool)
	t.doCopyPoolCount = 0
	if len(columns) > 0 {
		t.doCopyPoolColumns = columns
	} else {
		t.doCopyPoolColumns = make([]string, len(t.Columns()))
		for i, col := range t.Columns() {
			t.doCopyPoolColumns[i] = col.Name()
		}
	}

	go func() {
		ticket := time.NewTicker(d)
		for  {
			select {
			case <-ticket.C:
				t.lock.Lock()
				err := t.doCopy(ctx)
				t.lock.Unlock()
				if err != nil {
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

}
// AddToPoolCopy add 'record' into copy pool
func (t *%[1]s) AddToPoolCopy(ctx context.Context, record *%[1]sFields) error {
	if len(t.doCopyPoll) == 0 {
		t.InitPoolCopy(ctx, 3, 100 * time.Millisecond)
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.doCopyPoll = append(t.doCopyPoll, record)
	t.doCopyPoolCount++
	if t.doCopyPoolCount == cap(t.doCopyPoll) {
		err := t.doCopy(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// %[1]sFields data object for '%[1]s' columns
type %[1]sFields struct {
	// columns of table %s
}`
	// todo add DTO interface & SelectToMaps
	colFormat     = "\n\t%-21s\t%-13s\t`json:\"%s\"`"
	initFormat    = "\n\t\t%-21s:\t%s,"
	caseRefFormat = `
	case "%s":
		return &r.%s
`
	caseColFormat = `
	case "%s":
		return r.%s
`
	footer = `
// New%sFields create new instance & fill struct fill for avoid panic
func New%[1]sFields() *%[1]sFields{
	return &%[1]sFields{
		// init properties %[5]s
	}
}
// RefColValue return referral of column
func (r *%[1]sFields) RefColValue(name string) interface{}{
	switch name {	%s
   	default:
		return nil
	}
}
// ColValue return value of column
func (r *%[1]sFields) ColValue(name string) interface{}{
	switch name {	%[3]s
   	default:
		return nil
	}
}
// GetValue implement RouteDTO interface
func (r *%[1]sFields) GetValue() interface{} {
	return r
}
// NewValue implement RouteDTO interface
func (r *%[1]sFields) NewValue() interface{} {
	return New%[1]sFields()
}
// NewRecord return new row of table
func (t *%[1]s) NewRecord() *%[1]sFields{
   t.Record = New%[1]sFields()
	return t.Record
}
// GetFields implement RowColumn interface
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
// SelectSelfScanEach exec request to DB & populate record & call each for each row of query
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func(record *%[1]sFields) error, Options ...dbEngine.BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, 
			func() error {
			 	if each != nil {
					return each(t.Record)
				}

				 return nil
			}, t, Options ... )
}
// Insert new record into table
func (t *%[1]s) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		columns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.AutoIncrement() {
				continue
			}

			columns = append(columns, col.Name())
			v = append(v, t.Record.ColValue( col.Name() ) )
		}
		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.Values(v... ),
		)
   }

	return t.Table.Insert(ctx, Options...)
}
// Update record of table according to Options
func (t *%[1]s) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		priV := make([]interface{}, 0)
		columns := make([]string, 0, len(t.Columns()))
		priColumns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV = append(priV, t.Record.ColValue( col.Name() ))
				continue
			}

			columns = append( columns, col.Name() )
			v = append(v, t.Record.ColValue( col.Name() ) )
		}

		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.WhereForSelect(priColumns...), 
			dbEngine.Values(append(v, priV...)... ),
		)
	}

	return t.Table.Update(ctx, Options...)
}

// Upsert insert new Record into table according to Options or update if this record exists
func (t *%[1]s) Upsert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		priV := make([]interface{}, 0)
		columns := make([]string, 0, len(t.Columns()))
		priColumns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV = append(priV, t.Record.ColValue( col.Name() ))
				continue
			}

			columns = append( columns, col.Name() )
			v = append(v, t.Record.ColValue( col.Name() ) )
		}

		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.WhereForSelect(priColumns...), 
			dbEngine.Values(v... ),
		)
	}

	return t.Table.Upsert(ctx, Options...)
}

func (t *%[1]s) doCopy(ctx context.Context) error {
	if len(t.doCopyPoll) == 0 {
		return nil
	}

	t.doCopyValuesCount = -1
	i, err := t.DoCopy(ctx, t, t.doCopyPoolColumns...)
	if err != nil {
		logs.ErrorLog(err, "during doCopy")
		return err
	}

	logs.DebugLog("%%d record insert with CopyFrom", i)
	t.doCopyPoll = t.doCopyPoll[:0]
	t.doCopyPoolCount = 0

	return nil
}
`
)
