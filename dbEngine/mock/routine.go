// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/pkg/errors"
	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/logs"
	"github.com/stretchr/testify/assert"
	"golang.org/x/net/context"
)

// Routine imitate routine functional
type Routine struct {
	ParamsIsCallError interface{}
	TypeParams        []interface{}
	Test              assert.TestingT
}

func (r Routine) Name() string {
	panic("implement me")
}

func (r Routine) BuildSql(Options ...dbEngine.BuildSqlOptions) (sql string, args []interface{}, err error) {
	panic("implement me")
}

func (r Routine) Select(ctx context.Context, args ...interface{}) error {
	panic("implement me")
}

func (r Routine) Call(ctx context.Context, args ...interface{}) error {
	for i, val := range args {
		if !assert.IsType(r.Test, r.TypeParams[i], val) ||
			assert.ObjectsAreEqualValues(val, r.ParamsIsCallError) {
			logs.StatusLog(val)
			return errors.New("test error during proc execute")
		}
	}
	return nil
}

func (r Routine) Overlay() dbEngine.Routine {
	panic("implement me")
}

func (r Routine) Params() []dbEngine.Column {
	return nil
}

func (r Routine) ReturnType() string {
	panic("implement me")
}

func (r Routine) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

func (r Routine) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}

func (r Routine) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	panic("implement me")
}
