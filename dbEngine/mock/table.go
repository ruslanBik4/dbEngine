// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// Table implement dbEngine interface Table
type Table struct {
	name, Type string
	comment    string
	columns    []dbEngine.Column
}

// Delete row of table according to Options
func (t *Table) Delete(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *Table) Indexes() dbEngine.Indexes {
	//TODO implement me
	panic("implement me")
}

// ReReadColumn renew properties of column 'name'
func (t *Table) ReReadColumn(name string) dbEngine.Column {
	//TODO implement me
	panic("implement me")
}

// SelectAndScanEach run sql of table with Options & return every row into rowValues & run each
func (t *Table) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

// SelectOneAndScan run sqlof table  with Options & return rows into rowValues
func (t *Table) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

// SelectAndRunEach run sql of table with Options & performs each every row of query results
func (t *Table) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

// NewTable create new mock table
func NewTable(name string, typ string, comment string, columns ...dbEngine.Column) *Table {
	return &Table{name: name, Type: typ, comment: comment, columns: columns}
}

// InitConn create pool of connection
func (t *Table) InitConn(ctx context.Context, dbURL string) error {
	return nil
}

func (t *Table) GetRoutines(ctx context.Context) (map[string]dbEngine.Routine, error) {
	panic("implement me")
}

func (t *Table) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	panic("implement me")
}

// GetStat return stats
func (t *Table) GetStat() string {
	panic("implement me")
}

// ExecDDL execute sql
func (t *Table) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	panic("implement me")
}

func (t *Table) NewTable(name, typ string) dbEngine.Table {
	panic("implement me")
}

func (t *Table) SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {
	panic("implement me")
}

func (t *Table) SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	panic("implement me")
}

func (t *Table) SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []dbEngine.Column, error) {
	panic("implement me")
}

// Name of Table
func (t *Table) Name() string {
	return t.name
}

// Columns of Table
func (t *Table) Columns() []dbEngine.Column {
	return t.columns
}

// Comment of Table
func (t *Table) Comment() string {
	return t.comment
}

// FindColumn return column 'name' on Table or nil
func (t *Table) FindColumn(name string) dbEngine.Column {
	panic("implement me")
}

// FindIndex get index according to name
func (t *Table) FindIndex(name string) *dbEngine.Index {
	panic("implement me")
}

// GetColumns получение значений полей для форматирования данных
func (t *Table) GetColumns(ctx context.Context) error {
	panic("implement me")
}

// Insert new row & return new ID or rowsAffected if there not autoinc field
func (t *Table) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Update table according to Options
func (t *Table) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Upsert preforms INSERT sql or UPDATE if record with primary keys exists
func (t *Table) Upsert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// ReReadColumn renew properties of column 'name'
func (t *Table) RereadColumn(name string) dbEngine.Column {
	panic("implement me")
}

// Select run sql with Options (deprecated)
func (t *Table) Select(ctx context.Context, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}
