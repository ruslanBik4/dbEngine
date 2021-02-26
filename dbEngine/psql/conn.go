// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"os"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type fncConn func(context.Context, *pgx.Conn) error
type fncAcqu func(context.Context, *pgx.Conn) bool

type Conn struct {
	*pgxpool.Pool
	*pgxpool.Config
	*pgconn.Notice
	AfterConnect  fncConn
	BeforeAcquire fncAcqu
	NoticeHandler pgconn.NoticeHandler
	channels      []string
	ctxPool       context.Context
	lastComTag    pgconn.CommandTag
	Cancel        context.CancelFunc
}

func NewConn(afterConnect fncConn, beforeAcquire fncAcqu, noticeHandler pgconn.NoticeHandler, channels ...string) *Conn {
	return &Conn{
		AfterConnect:  afterConnect,
		BeforeAcquire: beforeAcquire,
		NoticeHandler: noticeHandler,
		channels:      channels,
	}
}

func (c *Conn) InitConn(ctx context.Context, dbURL string) error {
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return errors.Wrap(err, "cannot parse config")
	}

	poolCfg.ConnConfig.LogLevel = SetLogLevel(os.Getenv("PGX_LOG"))
	poolCfg.ConnConfig.Logger = &pgxLog{}

	poolCfg.AfterConnect = c.AfterConnect
	poolCfg.BeforeAcquire = c.BeforeAcquire
	poolCfg.ConnConfig.OnNotice = c.NoticeHandler

	c.Pool, err = pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		return errors.Wrap(err, "Unable to connect to database")
	}

	c.ctxPool, c.Cancel = context.WithCancel(ctx)

	c.StartChannels()

	return nil
}

// LastRowAffeted return number of insert/deleted/updated rows
func (c *Conn) LastRowAffected() int64 {
	return c.lastComTag.RowsAffected()
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
		func(values []interface{}, columns []dbEngine.Column) error {
			types[values[0].(string)] = dbEngine.Types{
				0,
				values[0].(string),
				[]string{},
			}
			return nil
		}, sqlTypesList)

	// logs.DebugLog("types:")
	// for name, tip := range types {
	// 	logs.DebugLog("%s: %+v", name, tip)
	// }

	return tables, routines, types, err
}

// GetTablesProp получение данных таблиц по условию
func (c *Conn) GetTablesProp(ctx context.Context) (SchemaCache map[string]dbEngine.Table, err error) {
	// buf for scan table fields from query
	table := &Table{
		conn: c,
	}

	SchemaCache = make(map[string]dbEngine.Table, 0)

	err = c.SelectAndScanEach(
		ctx,
		func() error {

			t := &Table{
				conn:    c,
				name:    table.Name(),
				Type:    table.Type,
				comment: table.comment,
			}

			err := t.GetColumns(ctx)
			if err != nil {
				return errors.Wrap(err, "during get columns")
			}

			err = t.GetIndexes(ctx)
			if err != nil {
				return errors.Wrap(err, "during get indexes")
			}

			SchemaCache[t.Name()] = t

			return nil
		},
		table, sqlTableList)

	return
}

// GetRoutines get params ect of DB routines
func (c *Conn) GetRoutines(ctx context.Context) (RoutinesCache map[string]dbEngine.Routine, err error) {

	RoutinesCache = make(map[string]dbEngine.Routine, 0)

	err = c.selectAndRunEach(ctx,
		func(values []interface{}, columns []dbEngine.Column) error {

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
				for fnc.overlay != nil {
					fnc = fnc.overlay
				}
				fnc.overlay = row

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
		logs.DebugLog(c.addNoticeToErrLog(sql, args, rows)...)
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

		if each != nil {
			err = each()
		}
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

func (c *Conn) SelectOneAndScan(ctx context.Context, rowValues interface{}, sql string, args ...interface{}) error {
	if rowValues == nil {
		return dbEngine.ErrWrongType{
			Name:     "rowValues",
			TypeName: fmt.Sprintf("%T", rowValues),
			Attr:     "nil",
		}
	}

	conn, err := c.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	row, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(sql, args, row)...)
		return err
	}

	defer row.Close()
	if row.Err() != nil {
		return row.Err()
	}

	if !row.Next() {
		return pgx.ErrNoRows
	}

	switch r := rowValues.(type) {
	case dbEngine.RowScanner:
		return row.Scan(r.GetFields(c.getColumns(row, conn))...)

	case []interface{}:
		return row.Scan(r...)

	case []string:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	case []int32:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	case []int64:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	case []float32:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	case []float64:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	case []time.Time:
		v := make([]interface{}, len(r))
		for i := range r {
			v[i] = &(r[i])
		}

		return row.Scan(v...)

	default:
		return row.Scan(rowValues)
	}
}

func (c *Conn) SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error) {

	rows := make(map[string]interface{})

	err := c.selectAndRunEach(ctx,
		func(values []interface{}, columns []dbEngine.Column) error {
			for i, val := range values {
				rows[columns[i].Name()] = val
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
		func(values []interface{}, columns []dbEngine.Column) error {
			row := make(map[string]interface{}, len(columns))

			for i, val := range values {
				row[columns[i].Name()] = val
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

func (c *Conn) SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) (
	rows [][]interface{}, cols []dbEngine.Column, err error) {

	err = c.selectAndRunEach(ctx,
		func(values []interface{}, columns []dbEngine.Column) error {
			rows = append(rows, values)
			if len(cols) == 0 {
				cols = columns
			}

			return nil
		},
		sql, args...)
	if err != nil {
		return nil, nil, err
	}

	return rows, cols, nil
}

func (c *Conn) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, sql string, args ...interface{}) error {

	return c.selectAndRunEach(ctx, each, sql, args...)
}

func (c *Conn) selectAndRunEach(ctx context.Context, each dbEngine.FncEachRow,
	sql string, args ...interface{}) error {

	conn, err := c.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(sql, args, rows)...)
		return err
	}

	defer rows.Close()

	var columns []dbEngine.Column

	for rows.Next() {
		values, err := rows.Values()
		if err != nil {
			break
		}

		if each != nil {
			if len(columns) == 0 {
				columns = c.getColumns(rows, conn)
			}
			err = each(values, columns)
		}
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

func (c *Conn) getColumns(rows pgx.Rows, conn *pgxpool.Conn) []dbEngine.Column {
	fields := rows.FieldDescriptions()
	columns := make([]dbEngine.Column, len(fields))
	for i, col := range fields {
		dType, ok := conn.Conn().ConnInfo().DataTypeForOID(col.DataTypeOID)
		if ok {
			columns[i] = &Column{name: string(col.Name), DataType: dType.Name, UdtName: dType.Name}
		} else {
			columns[i] = &Column{name: string(col.Name)}
		}
	}

	return columns
}

func (c *Conn) GetStat() string {
	// todo: implements marshal
	return "c.Stat()"
}

func (c *Conn) ExecDDL(ctx context.Context, sql string, args ...interface{}) error {
	comTag, err := c.Exec(ctx, sql, args...)
	// if err != nil {
	// 	logs.DebugLog("%v '%s' %s", comTag., err, strings.Split(sqlTypesList, "\n")[0])
	// }

	c.lastComTag = comTag

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
