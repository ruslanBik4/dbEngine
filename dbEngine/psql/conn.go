// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"go/types"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/gotools/typesExt"
	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type FncConn func(context.Context, *pgx.Conn) error
type FncAcqu func(context.Context, *pgx.Conn) bool

// BuildRouteOptions implement 'Functional Option' pattern for ApiRoute settings
type BuildConnOptions func(route *Conn)

// AfterConnect set custom AfterConnect method on ApiRoute
func AfterConnect(fnc FncConn) BuildConnOptions {
	return func(c *Conn) {
		c.AfterConnect = fnc
	}
}

// BeforeAcquire set same custom method on ApiRoute
func BeforeAcquire(fnc FncAcqu) BuildConnOptions {
	return func(c *Conn) {
		c.BeforeAcquire = fnc
	}
}

// ChannelHandler set same custom method on ApiRoute
func ChannelHandler(fnc pgconn.NotificationHandler) BuildConnOptions {
	return func(c *Conn) {
		c.ChannelHandler = fnc
	}
}

// NoticeHandler set same custom method on ApiRoute
func NoticeHandler(fnc pgconn.NoticeHandler) BuildConnOptions {
	return func(c *Conn) {
		c.NoticeHandler = fnc
	}
}

// NoticeHandler set same custom method on ApiRoute
func Channels(channels ...string) BuildConnOptions {
	return func(c *Conn) {
		c.channels = channels
	}
}

// Conn implement connection to DB over pgx
type Conn struct {
	*pgxpool.Pool
	*pgxpool.Config
	AfterConnect   FncConn
	BeforeAcquire  FncAcqu
	ChannelHandler pgconn.NotificationHandler
	NoticeHandler  pgconn.NoticeHandler
	NoticeMap      map[uint32]*pgconn.Notice
	channels       []string
	ctxPool        context.Context
	lastComTag     pgconn.CommandTag
	Cancel         context.CancelFunc
	lock           sync.RWMutex
}

// NewConn create new instance
func NewConn(afterConnect FncConn, beforeAcquire FncAcqu, noticeHandler pgconn.NoticeHandler, channels ...string) *Conn {
	return &Conn{
		AfterConnect:  afterConnect,
		BeforeAcquire: beforeAcquire,
		NoticeHandler: noticeHandler,
		NoticeMap:     make(map[uint32]*pgconn.Notice, 0),
		channels:      channels,
	}
}

// NewConnWithOptions create new instance
func NewConnWithOptions(options ...BuildConnOptions) *Conn {
	c := &Conn{
		NoticeMap: make(map[uint32]*pgconn.Notice, 0),
	}
	for _, opt := range options {
		opt(c)
	}
	return c
}

// InitConn create pool of connection
func (c *Conn) InitConn(ctx context.Context, dbURL string) error {
	poolCfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return errors.Wrap(err, "cannot parse config")
	}

	// schema := os.Getenv("PGX_DB_SCHEMA")
	// if schema > "" {
	// 	poolCfg.
	// }
	maxConns := os.Getenv("PGX_MAX_CONNS")
	if maxConns > "" {
		i, err := strconv.Atoi(maxConns)
		if err != nil {
			logs.ErrorLog(err, maxConns)
		} else {
			poolCfg.MaxConns = int32(i)
		}
	}

	poolCfg.ConnConfig.LogLevel = SetLogLevel(os.Getenv("PGX_LOG"))
	poolCfg.ConnConfig.Logger = &pgxLog{c}

	poolCfg.AfterConnect = c.AfterConnect
	poolCfg.BeforeAcquire = c.BeforeAcquire
	poolCfg.ConnConfig.OnNotice = func(conn *pgconn.PgConn, notice *pgconn.Notice) {
		c.addNotice(conn.PID(), notice)
		if c.NoticeHandler != nil {
			c.NoticeHandler(conn, notice)
		}
	}
	// clear notice
	poolCfg.AfterRelease = func(conn *pgx.Conn) bool {
		c.lock.Lock()
		delete(c.NoticeMap, conn.PgConn().PID())
		c.lock.Unlock()

		return true
	}

	c.Pool, err = pgxpool.ConnectConfig(ctx, poolCfg)
	if err != nil {
		return errors.Wrap(err, "Unable to connect to database")
	}

	c.ctxPool, c.Cancel = context.WithCancel(ctx)

	c.StartChannels()

	return nil
}

func (c *Conn) addNotice(pid uint32, notice *pgconn.Notice) {
	c.lock.Lock()
	c.NoticeMap[pid] = notice
	c.lock.Unlock()
}

// LastRowAffected return number of insert/deleted/updated rows
func (c *Conn) LastRowAffected() int64 {
	return c.lastComTag.RowsAffected()
}

// GetSchema read DB schema & store it
func (c *Conn) GetSchema(ctx context.Context, cfg *dbEngine.CfgDB) (map[string]*string, map[string]dbEngine.Table, map[string]dbEngine.Routine, map[string]dbEngine.Types, error) {
	dbTypes := make(map[string]dbEngine.Types)
	typeBuf := &dbEngine.Types{}
	err := c.SelectAndScanEach(ctx,
		func() error {
			for i, attr := range typeBuf.Attr {
				attr.Column = &Column{
					name:       attr.Name,
					DataType:   attr.Type,
					isNullable: !attr.IsNotNull,
					UdtName:    attr.Type,
				}
				typeBuf.Attr[i] = attr
			}
			dbTypes[typeBuf.Name] = *typeBuf
			*typeBuf = dbEngine.Types{}
			return nil
		},
		typeBuf,
		sqlTypesList)
	if err != nil {
		logs.ErrorLog(err, "during getting databases dbTypes")
	}

	tables, err := c.GetTablesProp(ctx, dbTypes, cfg)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "GetTablesProp")
	}

	routines, err := c.GetRoutines(ctx, dbTypes, tables, cfg)
	if err != nil {
		return nil, nil, nil, nil, errors.Wrap(err, "GetRoutines")
	}

	database := make(map[string]*string)

	err = c.SelectOneAndScan(ctx, database, sqlDBSetting)
	if err != nil {
		logs.ErrorLog(err, "during getting settings")
	}

	return database, tables, routines, dbTypes, err
}

// GetTablesProp populate tables schemas data
func (c *Conn) GetTablesProp(ctx context.Context, dbTypes map[string]dbEngine.Types, cfg *dbEngine.CfgDB) (map[string]dbEngine.Table, error) {
	// buf for scan table fields from query
	table := &Table{
		conn: c,
	}

	tables := make(map[string]dbEngine.Table, 0)

	sql := sqlTableList
	if len(cfg.Excluded) > 0 {
		sql = "SELECT * from (" + sql + `) t WHERE table_name !~ '(`
		for i, name := range cfg.Excluded {
			sql += fmt.Sprintf(`%s)`, name)
			if i < len(cfg.Excluded)-1 {
				sql += "|("
			}
		}
		sql += `'`
	}

	err := c.SelectAndScanEach(
		ctx,
		func() error {

			t := &Table{
				conn:    c,
				name:    table.Name(),
				Type:    table.Type,
				comment: table.comment,
			}

			err := t.GetColumns(ctx, dbTypes)
			if err != nil {
				return errors.Wrapf(err, "during get columns of table '%s'", table.Name())
			}

			err = t.GetIndexes(ctx)
			if err != nil {
				return errors.Wrap(err, "during get indexes")
			}

			tables[t.Name()] = t

			return nil
		},
		table, sql)
	if err != nil {
		return nil, err
	}

	for _, table := range tables {
		for _, col := range table.(*Table).columns {
			if col.DataType == "USER-DEFINED" {

				udtName := strings.TrimPrefix(col.UdtName, "_")
				if t, ok := dbTypes[udtName]; ok {
					col.UserDefined = &t
					//	todo: research how to determinate CITEXT on database
				} else if col.BasicType() == types.UntypedNil {
					if _, ok := tables[udtName]; ok {
						col.basicKind = typesExt.TStruct
					}
					if _, ok := dbTypes[udtName]; ok {
						col.basicKind = typesExt.TStruct
					}
					logs.StatusLog(col.basicKind)
				} else if udtName != "citext" {
					logs.ErrorLog(dbEngine.ErrNotFoundType{
						Name: udtName,
						Type: col.DataType,
					})
				}
			}
			for _, key := range col.Constraints {
				if key != nil && key.ForeignCol == nil {
					if p, ok := tables[key.Parent]; ok {
						key.ForeignCol = p.FindColumn(key.Column)
					}
				}
			}
		}
	}

	return tables, nil
}

// GetRoutines get properties of DB routines & returns them as map
func (c *Conn) GetRoutines(ctx context.Context, dbTypes map[string]dbEngine.Types, tables map[string]dbEngine.Table, cfg *dbEngine.CfgDB) (routines map[string]dbEngine.Routine, err error) {

	routines = make(map[string]dbEngine.Routine, 0)

	sql := sqlRoutineList
	if len(cfg.Excluded) > 0 {
		sql = "SELECT * from (" + sql + `) t WHERE routine_name !~ '(`
		for i, name := range cfg.Excluded {
			sql += fmt.Sprintf(`%s)`, name)
			if i < len(cfg.Excluded)-1 {
				sql += "|("
			}
		}
		sql += `'`
	}

	err = c.selectAndRunEach(ctx,
		func(values []any, columns []dbEngine.Column) error {

			// use only func knows types
			rType := values[2]
			if rType == nil {
				// not use
				return nil
			}

			rowType, ok := rType.(string)
			if !ok {
				logs.ErrorLog(errors.Wrapf(ErrUnknownRoutineType, " %+v", values))
				return nil
			}

			row := &Routine{
				conn:  c,
				name:  values[1].(string),
				sName: values[0].(string),
				Type:  rowType,
			}
			row.DataType, ok = values[3].(string)
			if !ok && row.Type == "FUNCTION" {
				logs.ErrorLog(errors.Wrapf(ErrFunctionWithoutResultType, " %+v", values))
				return nil
			}

			row.UdtName, ok = values[4].(string)
			if !ok && row.Type == "FUNCTION" {
				logs.ErrorLog(errors.Wrapf(ErrUnknownRoutineType, " %+v", values))
				return nil
			}

			row.Comment, _ = values[5].(string)
			name := values[1].(string)

			fnc, ok := routines[name].(*Routine)
			if ok {
				for fnc.overlay != nil {
					fnc = fnc.overlay
				}
				fnc.overlay = row

			} else {
				routines[name] = row
			}

			return row.GetParams(ctx, dbTypes, tables)
		}, sql)

	return
}

// NewTable create new empty Table with name & type
func (c *Conn) NewTable(name, typ string) dbEngine.Table {
	return &Table{conn: c, name: name, Type: typ}
}

// NewTableWithCheck create new Table with name, check the table from schema, populate columns and indexes
func (c *Conn) NewTableWithCheck(ctx context.Context, name string) (*Table, error) {
	table := &Table{
		conn: c,
	}

	isFound := false

	err := c.SelectAndScanEach(
		ctx,
		func() error {
			isFound = true
			return nil
		},
		table, sqlGetTable, name)
	if err != nil {
		return nil, err
	}

	if !isFound {
		return nil, dbEngine.ErrNotFoundTable{Table: name}
	}

	err = table.GetColumns(ctx, nil)
	if err != nil {
		return nil, errors.Wrap(err, "during get columns")
	}

	err = table.GetIndexes(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "during get indexes")
	}

	return table, nil
}

// SelectAndPerformRaw  run sql with args & run each every row
func (c *Conn) SelectAndPerformRaw(ctx context.Context, each dbEngine.FncRawRow, sql string, args ...any) error {
	conn, err := c.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, sql, args)...)
		return err
	}

	defer rows.Close()

	var columns []dbEngine.Column

	for rows.Next() {
		if each != nil {
			if len(columns) == 0 {
				columns = c.getColumns(rows, conn)
			}
			err = each(rows.RawValues(), columns)
		}
	}

	if rows.Err() != nil {
		err = rows.Err()
	}

	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, sql, rows.FieldDescriptions())...)
		return err
	}

	return nil
}

// SelectAndScanEach run sql with args return every row into rowValues & run each
func (c *Conn) SelectAndScanEach(ctx context.Context, each func() error, rowValue dbEngine.RowScanner,
	sql string, args ...any) error {

	conn, err := c.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.DebugLog(c.addNoticeToErrLog(conn, sql, args)...)
		return err
	}

	defer rows.Close()

	var columns []dbEngine.Column
	for rows.Next() && (err == nil) {
		if len(columns) == 0 {
			columns = c.getColumns(rows, conn)
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
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, "%+v", sql, rows.FieldDescriptions())...)
		return err
	}

	return nil
}

// SelectOneAndScan run sql with args return rows into rowValues
func (c *Conn) SelectOneAndScan(ctx context.Context, rowValues any, sql string, args ...any) (err error) {
	if rowValues == nil {
		return dbEngine.ErrWrongType{
			Name:     "rowValues",
			TypeName: fmt.Sprintf("%T", rowValues),
			Attr:     "nil",
		}
	}

	timeoutCtx, _ := context.WithTimeout(ctx, time.Second*5)
	conn, err := c.Acquire(timeoutCtx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	row, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, sql, args)...)
		return err
	}

	defer func() {
		n, ok := c.GetNotice(conn)
		if ok {
			if n.Code > "00000" && n.Code != "42P07" {
				err = (*pgconn.PgError)(n)
			}
		}
		row.Close()
	}()

	if !row.Next() {
		return pgx.ErrNoRows
	}

	columns := c.getColumns(row, conn)
	dest := c.getFieldForScan(rowValues, columns)
	if dest == nil {
		return row.Scan(rowValues)
	}

	return row.Scan(dest...)
}

func mapForScan[T any](r map[string]T, columns []dbEngine.Column) []any {
	v := make([]any, len(r))
	for i, col := range columns {
		v[i] = r[col.Name()]
	}
	logs.StatusLog(v)
	return v
}

func mapPointersForScan[T any](r map[string]*T, columns []dbEngine.Column) []any {
	isEmpty := len(r) == 0
	v := make([]any, len(columns))
	for i, col := range columns {
		if isEmpty {
			r[col.Name()] = new(T)
		}
		v[i] = r[col.Name()]
	}
	return v
}

func (c *Conn) getFieldForScan(rowValues any, columns []dbEngine.Column) []any {
	switch r := rowValues.(type) {
	case []any:
		return r

	case dbEngine.RowScanner:
		return r.GetFields(columns)

	case map[string]*string:
		return mapPointersForScan(r, columns)

	case map[string]string:
		return mapForScan(r, columns)

	case map[string]int:
		return mapForScan(r, columns)

	case map[string]int32:
		return mapForScan(r, columns)

	case map[string]int64:
		return mapForScan(r, columns)

	case map[string]float32:
		return mapForScan(r, columns)

	case map[string]float64:
		return mapForScan(r, columns)

	case map[string]any:
		return mapForScan(r, columns)

	case []string:
		return sliceForScan(r)

	case []int:
		return sliceForScan(r)

	case []int8:
		return sliceForScan(r)

	case []int16:
		return sliceForScan(r)

	case []int32:
		return sliceForScan(r)

	case []int64:
		return sliceForScan(r)

	case []float32:
		return sliceForScan(r)

	case []float64:
		return sliceForScan(r)

	case []time.Time:
		return sliceForScan(r)

	case []*time.Time:
		return sliceForScan(r)

	default:
		return nil
	}
}

func sliceForScan[T any](arr []T) []any {
	v := make([]any, len(arr))
	for i := range arr {
		v[i] = &(arr[i])
	}

	return v
}

// SelectToMap run sql with args return rows as map[{name_column}]
// case of executed - gets one record
func (c *Conn) SelectToMap(ctx context.Context, sql string, args ...any) (map[string]any, error) {

	rows := make(map[string]any)
	// todo: chande on selectScan with map
	err := c.selectAndRunEach(ctx,
		func(values []any, columns []dbEngine.Column) error {
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

// SelectToMaps run sql with args return rows as slice of map[{name_column}]
func (c *Conn) SelectToMaps(ctx context.Context, sql string, args ...any) ([]map[string]any, error) {

	maps := make([]map[string]any, 0)

	err := c.selectAndRunEach(ctx,
		func(values []any, columns []dbEngine.Column) error {
			row := make(map[string]any, len(columns))

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

// SelectToMultiDimension run sql with args and return rows (slice of record) and columns
func (c *Conn) SelectToMultiDimension(ctx context.Context, sql string, args ...any) (
	rows [][]any, cols []dbEngine.Column, err error) {

	err = c.selectAndRunEach(ctx,
		func(values []any, columns []dbEngine.Column) error {
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

// SelectAndRunEach run sql with args and performs each every row of query results
func (c *Conn) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, sql string, args ...any) error {

	return c.selectAndRunEach(ctx, each, sql, args...)
}

func (c *Conn) selectAndRunEach(ctx context.Context, each dbEngine.FncEachRow,
	sql string, args ...any) error {

	conn, err := c.Acquire(ctx)
	if err != nil {
		return errors.Wrap(err, "c.Acquire")
	}

	defer conn.Release()

	rows, err := conn.Query(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, sql, args)...)
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
			if err != nil {
				break
			}

		}
	}

	if rows.Err() != nil {
		err = rows.Err()
	}

	if err != nil {
		logs.ErrorLog(err, c.addNoticeToErrLog(conn, sql, rows.FieldDescriptions())...)
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
		// logs.StatusLog(fields)
	}

	return columns
}

// GetStat return stats of Pool
func (c *Conn) GetStat() string {
	s := c.Pool.Stat()
	return fmt.Sprintf("Acquired: %d/%d %v idle: %d, total: %d, max: %d",
		s.AcquiredConns(),
		s.AcquireCount(),
		s.AcquireDuration(),
		s.IdleConns(),
		s.TotalConns(),
		s.MaxConns(),
	)
}

// ExecDDL execute sql
func (c *Conn) ExecDDL(ctx context.Context, sql string, args ...any) error {
	comTag, err := c.Exec(ctx, sql, args...)
	// if err != nil {
	// 	logs.DebugLog("%v '%s' %s", comTag., err, strings.Split(sqlTypesList, "\n")[0])
	// }

	c.lastComTag = comTag

	return err
}

// StartChannels starts listeners of PSQL channels according to list of channels
func (c *Conn) StartChannels() {
	for _, ch := range c.channels {
		go c.listen(ch)
	}
}

// GetNotice return last notice of conn
func (c *Conn) GetNotice(conn *pgxpool.Conn) (n *pgconn.Notice, ok bool) {
	c.lock.RLock()
	defer c.lock.RUnlock()
	n, ok = c.NoticeMap[conn.Conn().PgConn().PID()]

	return
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

	logs.StatusLog("%s chan %s", cTag, ch)
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

		if c.ChannelHandler != nil {
			c.ChannelHandler(conn.Conn().PgConn(), n)
			continue
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

func (c *Conn) addNoticeToErrLog(conn *pgxpool.Conn, args ...any) []any {
	n, ok := c.GetNotice(conn)
	if ok {
		return append(args, n)
	}

	return args
}
