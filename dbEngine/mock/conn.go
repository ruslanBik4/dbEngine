// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package mock

import (
	"github.com/jackc/pgtype/pgxtype"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
	"github.com/stretchr/testify/mock"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// Conn for mock connection
type Conn struct {
	pgxtype.Querier
	mock.Call
}

//SelectAndPerformRaw  run sql with args & run each every row
func (c *Conn) SelectAndPerformRaw(ctx context.Context, each dbEngine.FncRawRow, sql string, args ...interface{}) error {

	return c.chkSqlAndArgs(ctx, sql, args)
}

func (c *Conn) chkSqlAndArgs(ctx context.Context, sql string, args []interface{}) error {
	if !regSQl.MatchString(sql) {
		return errors.New(sql)
	}

	if !c.Call.Arguments.Is(args...) {
		logs.DebugLog("args failed")
	}

	c.Call.Run(func(args mock.Arguments) {
		logs.StatusLog(args.String(), ctx.Value(CONN_MOCK_ENV))
	})

	return nil
}

// InitConn create pool of connection
func (c *Conn) InitConn(ctx context.Context, dbURL string) error {
	logs.DebugLog(dbURL)
	return nil
}

// GetRoutines get properties of DB routines & returns them as map
func (c *Conn) GetRoutines(ctx context.Context) (map[string]dbEngine.Routine, error) {
	panic("implement me")
}

// GetSchema read DB schema & store it
func (c *Conn) GetSchema(ctx context.Context) (map[string]*string, map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	panic("implement me")
}

// GetStat return stats of Pool
func (c *Conn) GetStat() string {
	panic("implement me")
}

// Exec mock exec command
func (c *Conn) Exec(ctx context.Context, sql string, args ...interface{}) error {
	logs.StatusLog(sql)
	if regSQl.MatchString(sql) {
		return nil
	}

	return errors.New(sql)
}

// ExecDDL execute sql
func (c *Conn) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	return c.chkSqlAndArgs(ctx, sql, args)
}

// NewTable create new empty Table with name & type
func (c *Conn) NewTable(name, typ string) dbEngine.Table {
	panic("implement me")
}

// LastRowAffected return number of insert/deleted/updated rows
func (c *Conn) LastRowAffected() int64 {
	panic("implement me")
}

// SelectOneAndScan run sql with Options & return rows into rowValues
func (c *Conn) SelectOneAndScan(ctx context.Context, rowValues interface{}, sql string, args ...interface{}) error {
	return c.chkSqlAndArgs(ctx, sql, args)
}

// SelectAndScanEach run sql with Options & return every row into rowValues & run each
func (c *Conn) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner, sql string, args ...interface{}) error {
	return c.chkSqlAndArgs(ctx, sql, args)
}

// SelectAndRunEach run sql with Options & performs each every row of query results
func (c *Conn) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, sql string, args ...interface{}) error {
	return c.chkSqlAndArgs(ctx, sql, args)
}

// SelectToMap run sql with args return rows as map[{name_column}]
// case of executed - gets one record
func (c *Conn) SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {
	return nil, c.chkSqlAndArgs(ctx, sql, args)
}

// SelectToMaps run sql with args return rows as slice of map[{name_column}]
func (c *Conn) SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {
	return nil, c.chkSqlAndArgs(ctx, sql, args)
}

// SelectToMultiDimension run sql with args and return rows (slice of record) and columns
func (c *Conn) SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []dbEngine.Column, error) {
	return nil, nil, c.chkSqlAndArgs(ctx, sql, args)
}
