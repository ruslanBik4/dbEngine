// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/pkg/errors"
	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// Routine imitate routine functional
type Routine struct {
	ParamsIsCallError interface{}
	TypeParams        []interface{}
	Test              assert.TestingT
}

// Name of mock routine
func (r Routine) Name() string {
	return "mock DB  routine"
}

// BuildSql create sql query & arg for call conn.Select...
func (r Routine) BuildSql(Options ...dbEngine.BuildSqlOptions) (sql string, args []interface{}, err error) {
	b := &dbEngine.SQLBuilder{}
	for _, option := range Options {
		_ = option(b)
	}

	return b.Select(), b.Args, nil
}

// Select run sql with Options (deprecated)
func (r Routine) Select(ctx context.Context, args ...interface{}) error {
	panic("implement me")
}

// Call procedure
func (r Routine) Call(ctx context.Context, args ...interface{}) error {
	return r.checkParams(args)
}

func (r Routine) checkParams(args []interface{}) error {
	for i, val := range args {
		if !assert.IsType(r.Test, r.TypeParams[i], val) ||
			assert.ObjectsAreEqualValues(val, r.ParamsIsCallError) {
			return errors.New("test error during proc execute")
		}
	}
	return nil
}

// Overlay return routine with some name if exists
func (r Routine) Overlay() dbEngine.Routine {
	return nil
}

// Params of Routine
func (r Routine) Params() []dbEngine.Column {
	return nil
}

// ReturnType of Routine
func (r Routine) ReturnType() string {
	panic("implement me")
}

// SelectAndScanEach run sql  with Options & return every row into rowValues & run each
func (r Routine) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	b := &dbEngine.SQLBuilder{}
	for _, option := range Options {
		_ = option(b)
	}

	return r.checkParams(b.Args)
}

// SelectOneAndScan run sql with Options & return rows into rowValues
func (r Routine) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	b := &dbEngine.SQLBuilder{}
	for _, option := range Options {
		_ = option(b)
	}

	return r.checkParams(b.Args)
}

// SelectAndRunEach run sql of table with Options & performs each every row of query results
func (r Routine) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	b := &dbEngine.SQLBuilder{}
	for _, option := range Options {
		_ = option(b)
	}

	return r.checkParams(b.Args)
}
