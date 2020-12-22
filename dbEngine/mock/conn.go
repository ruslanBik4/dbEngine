// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"fmt"
	"regexp"

	"github.com/jackc/pgtype/pgxtype"
	"github.com/pkg/errors"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type Conn struct {
	pgxtype.Querier
	mock.Call
}

func (c *Conn) InitConn(ctx context.Context, dbURL string) error {
	panic("implement me")
}

func (c *Conn) GetRoutines(ctx context.Context) (map[string]dbEngine.Routine, error) {
	panic("implement me")
}

func (c *Conn) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	panic("implement me")
}

func (c *Conn) GetStat() string {
	panic("implement me")
}

var regSQl = regexp.MustCompile(`select\s+(\.+)\s+from\s+`)

func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) error {
	fmt.Print(sql)
	if regSQl.MatchString(sql) {
		return nil
	}

	return errors.New(sql)
}

func (c *Conn) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	panic("implement me")
}

func (c *Conn) NewTable(name, typ string) dbEngine.Table {
	panic("implement me")
}

func (c *Conn) LastRowAffected() int64 {
	panic("implement me")
}

func (c *Conn) SelectOneAndScan(ctx context.Context, rowValues interface{}, sql string, args ...interface{}) error {
	panic("implement me")
}

func (c *Conn) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, sql string, args ...interface{}) error {
	panic("implement me")
}

func (c *Conn) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, sql string, args ...interface{}) error {
	panic("implement me")
}

func (c *Conn) SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {
	panic("implement me")
}

func (c *Conn) SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	panic("implement me")
}

func (c *Conn) SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []dbEngine.Column, error) {
	panic("implement me")
}
