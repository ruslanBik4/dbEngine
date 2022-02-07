// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

const (
	moduloPgType = "github.com/jackc/pgtype"
	moduloSql    = "database/sql"
)

const (
	title = `// Code generated by dbEngine-gen-go. DO NOT EDIT.
// versions:
// 	dbEngine v1.1.6
// source: %s %s 
package db

import (
	"time"

	%s
	"github.com/ruslanBik4/logs"
	"github.com/ruslanBik4/dbEngine/dbEngine"
    "github.com/ruslanBik4/dbEngine/dbEngine/psql"

	"golang.org/x/net/context"
	"github.com/pkg/errors"
)

`
	formatDatabase = `// Database is root interface for operation for %s.%s
type Database struct {
	*dbEngine.DB
}

// NewDatabase create new Database with minimal necessary handlers
func NewDatabase(ctx context.Context, noticeHandler pgconn.NoticeHandler, channelHandler pgconn.NotificationHandler, channels ...string) (*Database, error) {
	if noticeHandler == nil {
		noticeHandler = printNotice
	}
	conn := psql.NewConn(nil, nil, noticeHandler, channels...)
	if channelHandler != nil {
		conn.ChannelHandler = channelHandler
	}

	DB, err := dbEngine.NewDB(ctx, conn)
	if err != nil {
		logs.ErrorLog(err, "")
		return nil, err
	}

	return &Database{DB}, nil
}
// PsqlConn return connection as *psql.Conn
// need for some low-level operation, 
// invoke Conn.Select...(custom sql),
//        New{table_name}FromConn, etc.
func (d *Database) PsqlConn() *psql.Conn {
	return (d.Conn).(*psql.Conn)
}
`
	callProcFormat = `// %s call procedure '%[5]s' 
// DB comment: '%s'
func (d *Database) %[1]s(ctx context.Context%s) error {
	return d.Conn.ExecDDL(ctx, 
				"%s",
				%s)
}
`
	newFuncFormat = `// %s run query with select DB function '%[7]s'
// DB comment: '%s'
// ATTENTION! Now returns only 1 row
func (d *Database) %[1]s(ctx context.Context%s) (%serr error) {
	err = d.Conn.SelectOneAndScan(ctx, %s 
				"%s FETCH FIRST 1 ROW ONLY",
				%s)
	
	return
}
`
	colFormat    = "\n\t%-21s\t%-13s\t`json:\"%s\"`"
	initFormat   = "\n\t\t%-21s:\t%s,"
	paramsFormat = `
				[]interface{}{
					%s
				},`
	caseRefFormat = `
	case "%s":
		return &r.%s
`
	caseColFormat = `
	case "%s":
		return r.%s
`
	formatDBprivate = `// printNotice logging some psql messages (invoked command 'RAISE ')
func printNotice(c *pgconn.PgConn, n *pgconn.Notice) {

	switch {
    case n.Code == "42P07" || strings.Contains(n.Message, "skipping") :
		logs.DebugLog("skip operation: %%s", n.Message)
	case n.Severity == "INFO" :
		logs.StatusLog(n.Message)
	case n.Code > "00000" :
		err := (*pgconn.PgError)(n)
		logs.ErrorLog(err, n.Hint, err.SQLState(), err.File, err.Line, err.Routine)
	case strings.HasPrefix(n.Message, "[[ERROR]]") :
		logs.ErrorLog(errors.New(strings.TrimPrefix(n.Message, "[[ERROR]]") + n.Severity))
	default: // DEBUG
		logs.DebugLog("%%+v %%s (PID:%%d)", n.Severity, n.Message, c.PID())
	}
}`
	formatTable = `// %s object for database operations
// DB comment: '%[4]s'
type %[1]s struct {
	*psql.Table
	Record 				*%[1]sFields
	DoCopyPoll      	[]*%[1]sFields
	doCopyPoolCount 	int
	doCopyPoolColumns 	[]string
	doCopyValuesCount 	int
	doCopyErr       	error
	lock       			sync.RWMutex
}

// New%[1]s create new instance of table object
func New%[1]s( db *dbEngine.DB) (*%[1]s, error) {
	table, ok := db.Tables["%[3]s"]
    if !ok {
      return nil, dbEngine.ErrNotFoundTable{Table: "%[3]s"}
    }

    return &%[1]s{
		Table: table.(*psql.Table),
    }, nil
}
// New%[1]s create new instance of table object from Connection 
// it's necessary if Database create without reading schema of DB
func New%[1]sFromConn(ctx context.Context, conn *psql.Conn) (*%[1]s, error) {

	t := conn.NewTable("%[3]s", "%[5]s").(*psql.Table)
	err := t.GetColumns(ctx)
	if err != nil {
		logs.ErrorLog(err, "during GetColumns")
		return nil, err
	}

    return &%[1]s{
		Table: t,
    }, nil
}
// implementation pgx.CopyFromSource
// CopyFromSource is the interface used by *Conn.CopyFrom as the source for copy data.

// Next returns true if there is another row and makes the next row data
// available to Values(). When there are no more rows available or an error
// has occurred it returns false.
func (t *%[1]s) Next() bool {
	t.doCopyValuesCount++
	return t.doCopyValuesCount < len(t.DoCopyPoll)
}
// Values returns the values for the current row.
func (t *%[1]s) Values() ([]interface{}, error) {
	res := make([]interface{}, len(t.doCopyPoolColumns))
	for i, col := range t.doCopyPoolColumns {
		res[i] = t.DoCopyPoll[t.doCopyValuesCount].ColValue(col)
	}

	return res, nil
}
// Err returns any error that has been encountered by the CopyFromSource. If
// this is not nil *Conn.CopyFrom will abort the copy.
func (t *%[1]s) Err() error {
	return t.doCopyErr
}
// InitPoolCopy init environments for CopyFrom operation
// capOfPool define max pool capacity
// d - interval for flash
// columns - names of columns for CopyFrom (all columns of table if it not present)
// chErr - channel for error when CopyFrom return it
// on this case operation will terminated
func (t *%[1]s) InitPoolCopy(ctx context.Context, capOfPool int, chErr *chan error, d time.Duration, columns ...string) {
	t.lock.Lock()
	defer t.lock.Unlock()

	t.DoCopyPoll = make([]*%[1]sFields, 0, capOfPool)
	t.doCopyPoolCount = 0
	if len(columns) > 0 {
		t.doCopyPoolColumns = columns
	} else {
		t.doCopyPoolColumns = make([]string, len(t.Columns()))
		for i, col := range t.Columns() {
			t.doCopyPoolColumns[i] = col.Name()
		}
	}
	t.doCopyErr = nil

	go func() {
		ticket := time.NewTicker(d)
		for {
			select {
			case <-ticket.C:
				t.lock.Lock()
				err := t.doCopy(ctx)
				t.lock.Unlock()
				if err != nil {
					if chErr != nil {
					   *chErr <- err
					}
					t.doCopyErr = err
					return
				}
			case <-ctx.Done():
				return
			}
		}
	}()

}
// AddToPoolCopy add 'record' into copy pool
func (t *%[1]s) AddToPoolCopy(ctx context.Context, record *%[1]sFields) error {
	if t.doCopyErr != nil {
		return t.doCopyErr
	}

	if cap(t.DoCopyPoll) == 0 {
		t.InitPoolCopy(ctx, 3, nil,  100 * time.Millisecond)
	}

	t.lock.Lock()
	defer t.lock.Unlock()

	t.DoCopyPoll = append(t.DoCopyPoll, record)
	t.doCopyPoolCount++
	if t.doCopyPoolCount == cap(t.DoCopyPoll) {
		err := t.doCopy(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

// %[1]sFields data object for '%[1]s' columns
type %[1]sFields struct {
	// columns of table %s
}`
	footer = `
// New%sFields create new instance & fill struct fill for avoid panic
func New%[1]sFields() *%[1]sFields{
	return &%[1]sFields{
		// init properties %[5]s
	}
}
// RefColValue return referral of column
func (r *%[1]sFields) RefColValue(name string) interface{}{
	switch name {	%s
   	default:
		return nil
	}
}
// ColValue return value of column
func (r *%[1]sFields) ColValue(name string) interface{}{
	switch name {	%[3]s
   	default:
		return nil
	}
}
// GetValue implement httpgo.RouteDTO interface
func (r *%[1]sFields) GetValue() interface{} {
	return r
}
// NewValue implement httpgo.RouteDTO interface
func (r *%[1]sFields) NewValue() interface{} {
	return New%[1]sFields()
}
// NewRecord return new row of table
func (t *%[1]s) NewRecord() *%[1]sFields{
   t.Record = New%[1]sFields()
	return t.Record
}
// GetFields implement dbEngine.RowScanner interface
func (t *%[1]s) GetFields(columns []dbEngine.Column) []interface{} {
	if len(columns) == 0 {
		columns = t.Columns()
	}

	t.NewRecord()
	v := make([]interface{}, len(columns))
	for i, col := range columns {
		v[i] = t.Record.RefColValue( col.Name() )
	}

	return v
}
// SelectSelfScanEach exec request to DB & populate record & call each for each row of query
func (t *%[1]s) SelectSelfScanEach(ctx context.Context, each func(record *%[1]sFields) error, Options ...dbEngine.BuildSqlOptions) error {
	return t.SelectAndScanEach(ctx, 
			func() error {
			 	if each != nil {
					return each(t.Record)
				}

				 return nil
			}, t, Options ... )
}
// SelectAll run sql according to Options & return slice of record
func (t *%[1]s) SelectAll(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (res []*%[1]sFields, err error) {
	err = t.SelectAndScanEach(ctx, 
			func() error {
			 	res = append(res, t.Record)
				return nil
			}, t, Options ... )
	if err != nil {
		logs.ErrorLog(err, "during doCopy")
		return nil, err
	}

	return res, nil
}
// Insert new record into table
func (t *%[1]s) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		columns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.AutoIncrement() {
				continue
			}

			columns = append(columns, col.Name())
			v = append(v, t.Record.ColValue( col.Name() ) )
		}
		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.Values(v... ),
		)
   }

	return t.Table.Insert(ctx, Options...)
}
// Update record of table according to Options
func (t *%[1]s) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		priV := make([]interface{}, 0)
		columns := make([]string, 0, len(t.Columns()))
		priColumns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV = append(priV, t.Record.ColValue( col.Name() ))
				continue
			}

			columns = append( columns, col.Name() )
			v = append(v, t.Record.ColValue( col.Name() ) )
		}

		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.WhereForSelect(priColumns...), 
			dbEngine.Values(append(v, priV...)... ),
		)
	}

	return t.Table.Update(ctx, Options...)
}

// Upsert insert new Record into table according to Options or update if this record exists
func (t *%[1]s) Upsert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	if len(Options) == 0 {
		v := make([]interface{}, 0, len(t.Columns()))
		priV := make([]interface{}, 0)
		columns := make([]string, 0, len(t.Columns()))
		priColumns := make([]string, 0, len(t.Columns()))
		for _, col := range t.Columns() {
			if col.Primary() {
				priColumns = append( priColumns, col.Name() )
				priV = append(priV, t.Record.ColValue( col.Name() ))
				continue
			}

			columns = append( columns, col.Name() )
			v = append(v, t.Record.ColValue( col.Name() ) )
		}

		Options = append(
			Options, 
			dbEngine.Columns(columns...), 
			dbEngine.WhereForSelect(priColumns...), 
			dbEngine.Values(v... ),
		)
	}

	return t.Table.Upsert(ctx, Options...)
}

func (t *%[1]s) doCopy(ctx context.Context) error {
	if len(t.DoCopyPoll) == 0 {
		return nil
	}

	t.doCopyValuesCount = -1
	i, err := t.DoCopy(ctx, t, t.doCopyPoolColumns...)
	if err != nil {
		logs.ErrorLog(err, "during doCopy")
		return err
	}

	logs.DebugLog("%%d record insert with CopyFrom", i)
	t.DoCopyPoll = t.DoCopyPoll[:0]
	t.doCopyPoolCount = 0

	return nil
}
`
)
