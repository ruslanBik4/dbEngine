// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"golang.org/x/net/context"
)

type TableString struct {
	columns       []Column
	name, comment string
}

func (t TableString) Comment() string {
	return t.comment
}

func (t TableString) Upsert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t TableString) Columns() []Column {
	return t.columns
}

func (t TableString) FindColumn(name string) Column {
	for _, col := range t.columns {
		if col.Name() == name {
			return col
		}
	}

	return nil
}

func (t TableString) FindIndex(name string) *Index {
	panic("implement me")
}

func (t TableString) GetColumns(ctx context.Context) error {
	panic("implement me")
}

func (t TableString) Insert(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t TableString) Update(ctx context.Context, Options ...BuildSqlOptions) (int64, error) {
	panic("implement me")
}

func (t TableString) Name() string {
	return t.name
}

func (t TableString) ReReadColumn(name string) Column {
	panic("implement me")
}

func (t TableString) Select(ctx context.Context, Options ...BuildSqlOptions) error {
	panic("implement me")
}

func (t TableString) SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error {
	panic("implement me")
}

func (t TableString) SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error {
	panic("implement me")
}

func (t TableString) SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error {
	panic("implement me")
}
