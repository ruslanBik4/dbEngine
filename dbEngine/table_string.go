// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"golang.org/x/net/context"
)

// TableString implements Table interface for test
type TableString struct {
	columns       []Column
	indexes       Indexes
	name, comment string
}

// GetIndexes collect index of table
func (t TableString) Indexes() Indexes {
	return t.indexes
}

// Comment of Table
func (t TableString) Comment() string {
	return t.comment
}

// Columns of Table
func (t TableString) Columns() []Column {
	return t.columns
}

// FindColumn return column 'name' on Table or nil
func (t TableString) FindColumn(name string) Column {
	for _, col := range t.columns {
		if col.Name() == name {
			return col
		}
	}

	return nil
}

// FindIndex get index according to name
func (t TableString) FindIndex(name string) *Index {
	panic("implement me")
}

// GetColumns получение значений полей для форматирования данных
func (t TableString) GetColumns(ctx context.Context) error {
	panic("implement me")
}

// Delete row of table according to Options
func (t TableString) Delete(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Insert new row & return new ID or rowsAffected if there not autoinc field
func (t TableString) Insert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Update table according to Options
func (t TableString) Update(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Upsert preforms INSERT sql or UPDATE if record with primary keys exists
func (t TableString) Upsert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Name of Table
func (t TableString) Name() string {
	return t.name
}

// ReReadColumn renew properties of column 'name'
func (t TableString) ReReadColumn(name string) Column {
	panic("implement me")
}

// Select run sql with Options (deprecated)
func (t TableString) Select(ctx context.Context, Options ...BuildSqlOptions) error {
	panic("implement me")
}

// SelectAndScanEach run sql of table with Options & return every row into rowValues & run each
func (t TableString) SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error {
	panic("implement me")
}

// SelectAndRunEach run sql of table with Options & performs each every row of query results
func (t TableString) SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error {
	panic("implement me")
}

// SelectOneAndScan run sqlof table  with Options & return rows into rowValues
func (t TableString) SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error {
	panic("implement me")
}
