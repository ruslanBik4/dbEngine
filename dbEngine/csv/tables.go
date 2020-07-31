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
type Table struct {
	columns  []dbEngine.Column
	fileName string
	csv      *csv.Reader
}

func (t *Table) Comment() string {
	panic("implement me")
}

func NewTable(filePath string) (*Table, error) {
	f, err := os.Open(filePath)
	if err != nil {
		return nil, errors.Wrap(err, "os.Open "+filePath)
	}

	return &Table{
		csv:      csv.NewReader(f),
		fileName: strings.Split(path.Base(filePath), ".")[0],
	}, nil
}

func (t *Table) InitConn(ctx context.Context, filePath string) error {

	return t.GetColumns(ctx)
}

func (t *Table) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, error) {
	return map[string]dbEngine.Table{t.fileName: t}, nil, nil
}

func (t *Table) GetStat() string {
	panic("implement me")
}

func (t *Table) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	panic("implement me")
}

func (t *Table) NewTable(name, typ string) dbEngine.Table {
	return &Table{fileName: name}
}

func (t *Table) Columns() []dbEngine.Column {
	return t.columns
}

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
	return t.fileName
}

func (t *Table) ReReadColumn(name string) dbEngine.Column {
	panic("implement me")
}

func (t *Table) Select(ctx context.Context, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

func (t *Table) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

func (t *Table) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}
