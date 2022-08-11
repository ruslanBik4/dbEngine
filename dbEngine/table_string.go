// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/logs"
)

// TableString implements Table interface for test
type TableString struct {
	columns       []Column
	indexes       Indexes
	name, comment string
	Rows          [][]string
}

func NewTableString(name string, comment string, columns []Column, indexes Indexes, rows [][]string) *TableString {
	return &TableString{columns: columns, indexes: indexes, name: name, comment: comment, Rows: rows}
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
	b, err := t.testSqlOptions(Options)
	if err != nil {
		return 0, err
	}
	logs.DebugLog(b)

	return 1, nil
}

// Insert new row & return new ID or rowsAffected if there not autoinc field
func (t TableString) Insert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	b, err := t.testSqlOptions(Options)
	if err != nil {
		return 0, err
	}
	logs.DebugLog(b)

	return 1, nil
}

// Update table according to Options
func (t TableString) Update(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	b, err := t.testSqlOptions(Options)
	if err != nil {
		return 0, err
	}
	logs.DebugLog(b)

	return 1, nil
}

// Upsert preforms INSERT sql or UPDATE if record with primary keys exists
func (t TableString) Upsert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	b, err := t.testSqlOptions(Options)
	if err != nil {
		return 0, err
	}
	logs.DebugLog(b)

	return 1, nil
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
	b, err := t.testSqlOptions(Options)
	if err != nil {
		return err
	}

	if len(b.columns) == 0 {
		for _, col := range t.columns {
			b.columns = append(b.columns, col.Name())
		}
	}
	logs.StatusLog(b)

	selCols := make([]Column, len(b.columns))
	indCols := make([]int, len(b.columns))
	for j, name := range b.columns {
		for i, col := range t.columns {
			if col.Name() == name {
				selCols[j] = col
				indCols[j] = i
				break
			}
		}
	}

	refs := rowValue.GetFields(selCols)

ReadRows:
	for _, row := range t.Rows {
		// filter row
		for j, name := range b.filter {
			for i, col := range t.columns {
				if col.Name() == name {
					if row[i] != fmt.Sprintf("%v", b.Args[j]) {
						continue ReadRows
					}
					break
				}
			}
		}
		// scanning row
		for j := range b.columns {
			n, err := fmt.Sscan(row[indCols[j]], refs[j])
			if err != nil {
				logs.ErrorLog(err, n)
			}
		}
		err := each()
		if err != nil {
			return err
		}
	}

	return nil
}

func (t TableString) testSqlOptions(Options []BuildSqlOptions) (*SQLBuilder, error) {
	b, err := NewSQLBuilder(t, Options...)
	if err != nil {
		return nil, errors.Wrap(err, "setOption")
	}

	for _, name := range b.columns {
		isFound := false
		for i, col := range t.columns {
			if col.Name() == name {
				isFound = true
				if col.Type() == fmt.Sprintf("%T", b.Args[i]) {
					return nil, errors.New("bad type of col" + name)

				}
			}
		}
		if !isFound {
			return nil, errors.New("nod found col" + name)
		}
	}

	return b, nil
}

// SelectAndRunEach run sql of table with Options & performs each every row of query results
func (t TableString) SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error {
	panic("implement me")
}

// SelectOneAndScan run sqlof table  with Options & return rows into rowValues
func (t TableString) SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error {
	panic("implement me")
}
