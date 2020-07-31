// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"github.com/jackc/pgconn"
	"github.com/jackc/pgproto3/v2"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/httpgo/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type fncConn func(context.Context, *pgx.Conn) error

type Conn struct {
	*pgxpool.Pool
	*pgxpool.Config
	*pgconn.Notice
	AfterConnect  fncConn
	BeforeAcquire func(context.Context, *pgx.Conn) bool
	channels      []string
	ctxPool       context.Context
	Cancel        context.CancelFunc
}

func NewConn(afterConnect fncConn, beforeAcquire func(context.Context, *pgx.Conn) bool, channels ...string) *Conn {
	return &Conn{AfterConnect: afterConnect, BeforeAcquire: beforeAcquire, channels: channels}
}

func (c *Conn) InitConn(ctx context.Context, dbURL string) error {
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return errors.Wrap(err, "cannot parse config")
	}

	poolCfg.AfterConnect = c.AfterConnect
	poolCfg.BeforeAcquire = c.BeforeAcquire
	c.Pool, err = pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		return errors.Wrap(err, "Unable to connect to database")
	}

	c.ctxPool, c.Cancel = context.WithCancel(ctx)

	c.StartChannels()

	return nil
}

func (c *Conn) GetSchema(ctx context.Context) (map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	tables, err := c.GetTablesProp(ctx)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "GetTablesProp")
	}
	routines, err := c.GetRoutines(ctx)
	if err != nil {
		return nil, nil, nil, errors.Wrap(err, "GetRoutines")
	}
	types := make(map[string]dbEngine.Types)

	err = c.selectAndRunEach(ctx,
		func(values []interface{}, columns []pgproto3.FieldDescription) error {
			types[values[0].(string)] = dbEngine.Types{
				0,
				values[0].(string),
				[]string{},
			}
			return nil
		}, sqlTypesList)
	return tables, routines, types, err
}

// GetTablesProp получение данных таблиц по условию
func (c *Conn) GetTablesProp(ctx context.Context) (SchemaCache map[string]dbEngine.Table, err error) {
	// buf for scan table fileds from query
	row := &Table{
		conn: c,
	}

	SchemaCache = make(map[string]dbEngine.Table, 0)

	err = c.SelectAndScanEach(
		ctx,
		func() error {

			t := &Table{
				conn:    c,
				name:    row.Name(),
				Type:    row.Type,
				comment: row.comment,
			}

			err := t.GetColumns(ctx)
			if err != nil {
				return errors.Wrap(err, "during get columns")
			}

			SchemaCache[row.Name()] = t

			return nil
		},
		row, sqlTableList)

	return
}

// GetRoutines get params ect of DB routines
func (c *Conn) GetRoutines(ctx context.Context) (RoutinesCache map[string]dbEngine.Routine, err error) {

	RoutinesCache = make(map[string]dbEngine.Routine, 0)

	err = c.selectAndRunEach(ctx,
		func(values []interface{}, columns []pgproto3.FieldDescription) error {

			// use only func knows types
			rowType, ok := values[2].(string)
			if !ok {
				return nil
			}

			row := &Routine{
				conn:  c,
				name:  values[1].(string),
				sName: values[0].(string),
				Type:  rowType,
			}
			row.DataType, ok = values[3].(string)
			row.UdtName, ok = values[4].(string)
			name := values[1].(string)

			fnc, ok := RoutinesCache[name].(*Routine)
			if ok {
				for fnc.Overlay != nil {
					fnc = fnc.Overlay
				}
				fnc.Overlay = row

			} else {
				RoutinesCache[name] = row
			}

			return row.GetParams(ctx)
		}, sqlFuncList)

	return
}

func (c *Conn) NewTable(name, typ string) dbEngine.Table {
	return &Table{conn: c, name: name, Type: typ}
}

func (c *Conn) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner,
	sql string, args ...interface{}) error {

	// sqlTypesList = convertSQLFromFuncIsNeed(sqlTypesList, args)
	rows, err := c.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(sql, args, rows)...)
		return err
	}

	defer rows.Close()

	var columns []dbEngine.Column
	for rows.Next() && (err == nil) {
		if columns == nil {
			columns = make([]dbEngine.Column, len(rows.FieldDescriptions()))
			for i, val := range rows.FieldDescriptions() {
				// todo chk DateOId
				columns[i] = &Column{name: string(val.Name)}
			}
		}

		err = rows.Scan(rowValue.GetFields(columns)...)
		if err != nil {
			break
		}

		err = each()
	}

	if rows.Err() != nil {
		err = rows.Err()
	}

	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog("%+v", sql, rows.FieldDescriptions())...)
		return err
	}

	return nil
}

func (c *Conn) SelectOneAndScan(ctx context.Context, sql string, args ...interface{}) error {
	return c.QueryRow(context.Background(), sql, args...).Scan(args...)
}

func (c *Conn) SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {

	rows := make(map[string]interface{})

	err := c.selectAndRunEach(ctx,
		func(values []interface{}, columns []pgproto3.FieldDescription) error {
			for i, val := range values {
				rows[string(columns[i].Name)] = val
			}

			return nil
		},
		sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "selectAndRunEach")
	}

	return rows, nil
}

func (c *Conn) SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error) {

	maps := make([]map[string]interface{}, 0)

	err := c.selectAndRunEach(ctx,
		func(values []interface{}, columns []pgproto3.FieldDescription) error {
			row := make(map[string]interface{}, len(columns))

			for i, val := range values {
				row[string(columns[i].Name)] = val
			}

			maps = append(maps, row)

			return nil
		},
		sql, args...)
	if err != nil {
		return nil, errors.Wrap(err, "selectAndRunEach")
	}

	return maps, nil
}

func (c *Conn) SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []dbEngine.Column, error) {
	columns := make([]dbEngine.Column, 0)
	rows := make([][]interface{}, 0)

	err := c.selectAndRunEach(ctx,
		func(values []interface{}, fields []pgproto3.FieldDescription) error {
			rows = append(rows, values)
			if len(columns) == 0 {
				for _, val := range fields {
					columns = append(columns, &Column{name: string(val.Name)})
				}
			}

			return nil
		},
		sql, args...)
	if err != nil {
		return nil, nil, err
	}

	return rows, columns, nil
}

func (c *Conn) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, sql string, args ...interface{}) error {

	return c.selectAndRunEach(ctx,
		func(values []interface{}, fields []pgproto3.FieldDescription) error {
			columns := make([]dbEngine.Column, len(fields))
			for i, val := range fields {
				columns[i] = &Column{name: string(val.Name)}
			}
			return each(values, columns)
		},
		sql, args...)
}

func (c *Conn) selectAndRunEach(ctx context.Context, each func(values []interface{}, columns []pgproto3.FieldDescription) error,
	sql string, args ...interface{}) error {

	rows, err := c.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(sql, args, rows)...)
		return err
	}

	defer rows.Close()

	var values []interface{}

	for rows.Next() {
		values, err = rows.Values()
		if err != nil {
			break
		}

		err = each(values, rows.FieldDescriptions())
	}

	if rows.Err() != nil {
		err = rows.Err()
	}

	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(sql, rows.FieldDescriptions())...)
		return err
	}

	return nil
}

func (c *Conn) GetStat() string {
	// todo: implements marshal
	return "c.Stat()"
}

func (c *Conn) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	_, err := c.Exec(ctx, sql, args...)
	// if err != nil {
	// 	logs.DebugLog("%v '%s' %s", comTag, err, strings.Split(sqlTypesList, "\n")[0])
	// }

	return err
}

func (c *Conn) StartChannels() {
	for _, ch := range c.channels {
		go c.listen(ch)
	}
}

func (c *Conn) GetNotice() string {
	if c.Notice == nil {
		return ""
	}

	return c.Message
}

func (c *Conn) listen(ch string) {
	conn, err := c.Acquire(c.ctxPool)
	if err != nil {
		logs.ErrorLog(err, "Error acquiring connection:")
		return
	}
	defer conn.Release()

	cTag, err := conn.Exec(c.ctxPool, "listen "+ch)
	if err != nil {
		logs.ErrorLog(err, "cannot open listen channel")
		return
	}

	logs.StatusLog("listen chan %+v", cTag)
	defer func() {
		err := recover()
		if err != nil {
			logs.ErrorLog(errors.Wrap(err.(error), "recover listen"))
		}
	}()

	for {
		n, err := conn.Conn().WaitForNotification(c.ctxPool)
		if err != nil {
			logs.ErrorLog(err, "Error waiting for notification:")
			return
		}

		// todo: implements performs of messages
		switch n.Payload {
		// case "all_calc":
		// 	c.block = true
		// case "finish_calc":
		// 	c.block = false
		case "exit":
			break
		default:
			logs.DebugLog("PID: %d, Channel: %s, Payload: %s", n.PID, n.Channel, n.Payload)
		}
	}
}

func (c *Conn) addNoticeToErrLog(args ...interface{}) []interface{} {
	if c.Notice != nil {
		return append(args, c.Notice)
	} else {
		return args
	}
}
