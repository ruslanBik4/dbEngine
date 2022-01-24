// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csv

import (
	"encoding/csv"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type Conn struct {
}

// Table implement dbEngine interface Table for csv
type Table struct {
	columns  []dbEngine.Column
	indexes  dbEngine.Indexes
	fileName string
	csv      *csv.Reader
}

// NewTable open csv & init conn
func NewTable(filePath string) (*Table, error) {
	t := &Table{}
	if filePath > "" {
		err := t.InitConn(context.Background(), filePath)
		if err != nil {
			return nil, err
		}
	}

	return t, nil
}

// Indexes get indexex according to table
func (t *Table) Indexes() dbEngine.Indexes {
	return t.indexes
}

// Comment of Table
func (t *Table) Comment() string {
	panic("implement me")
}

// InitConn create csv reader
func (t *Table) InitConn(ctx context.Context, filePath string) error {

	f, err := os.Open(filePath)
	if err != nil {
		return errors.Wrap(err, "os.Open "+filePath)
	}

	t.csv = csv.NewReader(f)
	t.fileName = strings.Split(path.Base(filePath), ".")[0]

	return t.GetColumns(ctx)
}

func (t *Table) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, error) {
	return map[string]dbEngine.Table{t.fileName: t}, nil, nil
}

// GetStat return stats of conn
func (t *Table) GetStat() string {
	panic("implement me")
}

// ExecDDL execute sql
func (t *Table) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	panic("implement me")
}

// NewTable create empty Table
func (t *Table) NewTable(name, typ string) dbEngine.Table {
	return &Table{fileName: name}
}

// Columns of Table
func (t *Table) Columns() []dbEngine.Column {
	return t.columns
}

// FindColumn return column 'name' on Table or nil
func (t *Table) FindColumn(name string) dbEngine.Column {
	for _, col := range t.columns {
		if col.Name() == name {
			return col
		}
	}

	return nil
}

func (t *Table) FindIndex(name string) *dbEngine.Index {
	return nil
}

// GetColumns получение значений полей для форматирования данных
func (t *Table) GetColumns(ctx context.Context) error {
	rec, err := t.csv.Read()
	if err != nil {
		return errors.Wrap(err, "csv.Read")
	}

	t.columns = make([]dbEngine.Column, len(rec))
	for i, name := range rec {
		t.columns[i] = dbEngine.NewStringColumn(strings.TrimSpace(name), "", false)
	}

	return nil
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

// Delete row of table according to Options
func (t *Table) Delete(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

// Name of Table
func (t *Table) Name() string {
	return t.fileName
}

// ReReadColumn renew properties of column 'name'
func (t *Table) ReReadColumn(name string) dbEngine.Column {
	panic("implement me")
}

// Select run sql with Options (deprecated)
func (t *Table) Select(ctx context.Context, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

// SelectOneAndScan run sqlof table  with Options & return rows into rowValues
func (t *Table) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

// SelectAndScanEach run sql of table with Options & return every row into rowValues & run each
func (t *Table) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

// SelectAndRunEach run sql of table with Options & performs each every row of query results
func (t *Table) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}
