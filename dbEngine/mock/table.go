// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type Table struct {
	name, Type string
	comment    string
	columns    []dbEngine.Column
}

func (t *Table) Delete(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	//TODO implement me
	panic("implement me")
}

func (t *Table) Indexes() dbEngine.Indexes {
	//TODO implement me
	panic("implement me")
}

func (t *Table) ReReadColumn(name string) dbEngine.Column {
	//TODO implement me
	panic("implement me")
}

func (t *Table) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

func (t *Table) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

func (t *Table) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	//TODO implement me
	panic("implement me")
}

func NewTable(name string, typ string, comment string, columns ...dbEngine.Column) *Table {
	return &Table{name: name, Type: typ, comment: comment, columns: columns}
}

func (t *Table) InitConn(ctx context.Context, dbURL string) error {
	return nil
}

func (t *Table) GetRoutines(ctx context.Context) (map[string]dbEngine.Routine, error) {
	panic("implement me")
}

func (t *Table) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	panic("implement me")
}

func (t *Table) GetStat() string {
	panic("implement me")
}

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

func (t *Table) Columns() []dbEngine.Column {
	return t.columns
}

func (t *Table) Comment() string {
	return t.comment
}

func (t *Table) FindColumn(name string) dbEngine.Column {
	panic("implement me")
}

func (t *Table) FindIndex(name string) *dbEngine.Index {
	panic("implement me")
}

func (t *Table) GetColumns(ctx context.Context) error {
	panic("implement me")
}

func (t *Table) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t *Table) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t *Table) Upsert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t *Table) Name() string {
	return t.name
}

func (t *Table) RereadColumn(name string) dbEngine.Column {
	panic("implement me")
}

func (t *Table) Select(ctx context.Context, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}
