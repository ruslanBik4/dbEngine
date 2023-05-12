// Code generated by qtc from "database_tpl.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line database_tpl.qtpl:1
package _go

//line database_tpl.qtpl:2
import (
	"fmt"

	"github.com/iancoleman/strcase"
	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
)

//line database_tpl.qtpl:11
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line database_tpl.qtpl:11
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line database_tpl.qtpl:11
func (c *Creator) StreamCreateDatabase(qw422016 *qt422016.Writer, listRoutines []string) {
//line database_tpl.qtpl:11
	qw422016.N().S(`
// Code generated by dbEngine-gen-go. DO NOT EDIT!
// versions:
// 	dbEngine v1.1.6
// source: %s %s

package db

import (
	"fmt"
	"time"
	"strings"
`)
//line database_tpl.qtpl:23
	for _, lib := range c.packages {
//line database_tpl.qtpl:23
		qw422016.N().S(`"`)
//line database_tpl.qtpl:23
		qw422016.E().S(lib)
//line database_tpl.qtpl:23
		qw422016.N().S(`"
`)
//line database_tpl.qtpl:24
	}
//line database_tpl.qtpl:24
	qw422016.N().S(`
	"github.com/jackc/pgx/v4"
	"github.com/jackc/pgconn"

	"github.com/ruslanBik4/logs"
	"github.com/ruslanBik4/dbEngine/dbEngine"
    "github.com/ruslanBik4/dbEngine/dbEngine/psql"

	"golang.org/x/net/context"
	"github.com/pkg/errors"
)
`)
//line database_tpl.qtpl:36
	for name, typ := range c.db.Types {
//line database_tpl.qtpl:36
		c.StreamCreateTypeInterface(qw422016, typ, strcase.ToCamel(name), name, c.Types[name])
//line database_tpl.qtpl:36
	}
//line database_tpl.qtpl:36
	qw422016.N().S(`
// Database is root interface for operation for %s.%s
type Database struct {
	*dbEngine.DB
	CreateAt time.Time
}
type CitextArray struct {
	pgtype.TextArray
}

func (dst CitextArray) MarshalJSON() ([]byte, error) {
	buf := bytes.NewBufferString("[")
	for i, text := range dst.Elements {
		if i > 0 {
			buf.WriteString(",")
		}

		buf.WriteString(text.String)
	}

	buf.WriteString("]")

	return buf.Bytes(), nil
}

var customTypes = map[string]*pgtype.DataType{
	"citext": &pgtype.DataType{
		Value: &pgtype.Text{},
		Name:  "citext",
	},
	"_citext": &pgtype.DataType{
		Value: &pgtype.TextArray{}, // (*pgtype.ArrayType)(nil),
		Name:  "[]string",
	},
}
var initCustomTypes bool

const sqlGetTypes = "SELECT typname, oid FROM pg_type WHERE typname::text=ANY($1)"

func getOidCustomTypes(ctx context.Context, conn *pgx.Conn) error {
	params := make([]string, 0, len(customTypes))
	for name := range customTypes {
		params = append(params, name)
	}

	rows, err := conn.Query(ctx, sqlGetTypes, params)
	if err != nil {
		return err
	}

	for rows.Next() {
		var name string
		var oid uint32
		err = rows.Scan(&name, &oid)
		if err != nil {
			return err
		}
		if c, ok := customTypes[name]; ok && c.Value == (*pgtype.ArrayType)(nil) {
			c.Value = pgtype.NewArrayType(name, oid, func() pgtype.ValueTranscoder {
				return &pgtype.Text{}
			}).NewTypeValue()
			c.OID = oid
			logs.DebugLog(c)
		} else if ok {
			customTypes[name].OID = oid
		}
	}

	if rows.Err() != nil {
		logs.ErrorLog(rows.Err(), " cannot get oid for customTypes")
	}

	return err
}

// AfterConnect create need types & register on conn
func AfterConnect(ctx context.Context, conn *pgx.Conn) error {
	// Override registered handler for point
	if !initCustomTypes {
		err := getOidCustomTypes(ctx, conn)
		if err != nil {
			return err
		}

		initCustomTypes = true
	}

	mess := "DB registered type (name, oid): "
	for name, val := range customTypes {
		conn.ConnInfo().RegisterDataType(*val)
		mess += fmt.Sprintf("(%s,%v, %T) ", name, val.OID, val.Value)
	}

	logs.StatusLog(conn.PgConn().Conn().LocalAddr().String(), mess)

	return nil
}

// NewDatabase create new Database with minimal necessary handlers
func NewDatabase(ctx context.Context, noticeHandler pgconn.NoticeHandler, channelHandler pgconn.NotificationHandler, channels ...string) (*Database, error) {
	if noticeHandler == nil {
		noticeHandler = printNotice
	}
	conn := psql.NewConnWithOptions(
		psql.AfterConnect(AfterConnect),
		psql.NoticeHandler(noticeHandler),
		psql.ChannelHandler(channelHandler),
		psql.Channels(channels...),
	)

	DB, err := dbEngine.NewDB(ctx, conn)
	if err != nil {
		logs.ErrorLog(err, "new DB")
		return nil, err
	}

	return &Database{DB, time.Now()}, nil
}
// PsqlConn return connection as *psql.Conn
// need for some low-level operation,
// invoke Conn.Select...(custom sql),
//        New{table_name}FromConn, etc.
func (d *Database) PsqlConn() *psql.Conn {
	return (d.Conn).(*psql.Conn)
}
`)
//line database_tpl.qtpl:162
	for name := range c.db.Tables {
//line database_tpl.qtpl:162
		c.StreamCreateTableConstructor(qw422016, strcase.ToCamel(name), name)
//line database_tpl.qtpl:162
	}
//line database_tpl.qtpl:163
	for _, name := range listRoutines {
//line database_tpl.qtpl:163
		c.StreamCreateRoutinesInvoker(qw422016, c.db.Routines[name].(*psql.Routine), name)
//line database_tpl.qtpl:163
	}
//line database_tpl.qtpl:163
	qw422016.N().S(`// printNotice logging some psql messages (invoked command 'RAISE ...')
func printNotice(c *pgconn.PgConn, n *pgconn.Notice) {

	switch {
    case n.Code == "42P07" || strings.Contains(n.Message, "skipping") :
		logs.DebugLog("skip operation: %s", n.Message)

	case n.Severity == "INFO" :
		logs.StatusLog(n.Message)

	case n.Code > "00000" :
		err := (*pgconn.PgError)(n)
		logs.CustomLog(logs.CRITICAL, "DB_EXEC",  err.File, int(err.Line),
			fmt.Sprintf("%v, hint: %s, where: %s, %s %s", err, n.Hint, n.Where,  err.SQLState(), err.Routine), logs.FgErr)

	case strings.HasPrefix(n.Message, "[[ERROR]]") :
		logs.ErrorLog(errors.New(strings.TrimPrefix(n.Message, "[[ERROR]]") + n.Severity))

	default: // DEBUG
		logs.DebugLog("%+v %s (PID:%d)", n.Severity, n.Message, c.PID())
	}
}
`)
//line database_tpl.qtpl:186
}

//line database_tpl.qtpl:186
func (c *Creator) WriteCreateDatabase(qq422016 qtio422016.Writer, listRoutines []string) {
//line database_tpl.qtpl:186
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:186
	c.StreamCreateDatabase(qw422016, listRoutines)
//line database_tpl.qtpl:186
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:186
}

//line database_tpl.qtpl:186
func (c *Creator) CreateDatabase(listRoutines []string) string {
//line database_tpl.qtpl:186
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:186
	c.WriteCreateDatabase(qb422016, listRoutines)
//line database_tpl.qtpl:186
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:186
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:186
	return qs422016
//line database_tpl.qtpl:186
}

//line database_tpl.qtpl:188
func (c *Creator) StreamCreateTypeInterface(qw422016 *qt422016.Writer, t dbEngine.Types, typeName, name, typeCol string) {
//line database_tpl.qtpl:189
	if len(t.Enumerates) > 0 {
//line database_tpl.qtpl:190
	} else {
//line database_tpl.qtpl:190
		qw422016.N().S(`// `)
//line database_tpl.qtpl:191
		qw422016.E().S(typeName)
//line database_tpl.qtpl:191
		qw422016.N().S(` create new instance of type `)
//line database_tpl.qtpl:191
		qw422016.E().S(name)
//line database_tpl.qtpl:191
		qw422016.N().S(`
type `)
//line database_tpl.qtpl:192
		qw422016.E().S(typeName)
//line database_tpl.qtpl:192
		qw422016.N().S(` struct {
`)
//line database_tpl.qtpl:193
		for _, tAttr := range t.Attr {
//line database_tpl.qtpl:193
			qw422016.N().S(`	`)
//line database_tpl.qtpl:194
			qw422016.N().S(fmt.Sprintf("%-21s\t%-13s\t", strcase.ToCamel(tAttr.Name), tAttr.Type))
//line database_tpl.qtpl:194
			qw422016.N().S(` `)
//line database_tpl.qtpl:194
			qw422016.N().S("`")
//line database_tpl.qtpl:194
			qw422016.N().S(`json:"`)
//line database_tpl.qtpl:194
			qw422016.E().S(tAttr.Name)
//line database_tpl.qtpl:194
			qw422016.N().S(`"`)
//line database_tpl.qtpl:194
			qw422016.N().S("`")
//line database_tpl.qtpl:194
			qw422016.N().S(`
`)
//line database_tpl.qtpl:195
		}
//line database_tpl.qtpl:195
		qw422016.N().S(`}
// DecodeText implement pgtype.TextDecoder interface
func (dst *`)
//line database_tpl.qtpl:198
		qw422016.E().S(typeName)
//line database_tpl.qtpl:198
		qw422016.N().S(`) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = `)
//line database_tpl.qtpl:200
		qw422016.E().S(typeName)
//line database_tpl.qtpl:200
		qw422016.N().S(`{}
		return nil
	}
	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))
    *dst = `)
//line database_tpl.qtpl:204
		qw422016.E().S(typeName)
//line database_tpl.qtpl:204
		qw422016.N().S(`{
`)
//line database_tpl.qtpl:205
		for i, tAttr := range t.Attr {
//line database_tpl.qtpl:205
			qw422016.N().S(`		`)
//line database_tpl.qtpl:206
			qw422016.N().S(c.GetFuncForDecode(&tAttr, i))
//line database_tpl.qtpl:206
			qw422016.N().S(`,
`)
//line database_tpl.qtpl:207
		}
//line database_tpl.qtpl:207
		qw422016.N().S(`	}

	return nil
}
`)
//line database_tpl.qtpl:212
	}
//line database_tpl.qtpl:213
}

//line database_tpl.qtpl:213
func (c *Creator) WriteCreateTypeInterface(qq422016 qtio422016.Writer, t dbEngine.Types, typeName, name, typeCol string) {
//line database_tpl.qtpl:213
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:213
	c.StreamCreateTypeInterface(qw422016, t, typeName, name, typeCol)
//line database_tpl.qtpl:213
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:213
}

//line database_tpl.qtpl:213
func (c *Creator) CreateTypeInterface(t dbEngine.Types, typeName, name, typeCol string) string {
//line database_tpl.qtpl:213
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:213
	c.WriteCreateTypeInterface(qb422016, t, typeName, name, typeCol)
//line database_tpl.qtpl:213
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:213
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:213
	return qs422016
//line database_tpl.qtpl:213
}

//line database_tpl.qtpl:215
func (c *Creator) StreamCreateRoutinesInvoker(qw422016 *qt422016.Writer, r *psql.Routine, name string) {
//line database_tpl.qtpl:217
	camelName := strcase.ToCamel(name)
	args := make([]any, len(r.Params()))
	sql, _, _ := r.BuildSql(dbEngine.ArgsForSelect(args...))
	typeReturn := ""

//line database_tpl.qtpl:222
	if r.Type == psql.ROUTINE_TYPE_PROC {
//line database_tpl.qtpl:222
		qw422016.N().S(`// `)
//line database_tpl.qtpl:223
		qw422016.E().S(camelName)
//line database_tpl.qtpl:223
		qw422016.N().S(` call procedure '`)
//line database_tpl.qtpl:223
		qw422016.E().S(name)
//line database_tpl.qtpl:223
		qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:224
		qw422016.E().S(r.Comment)
//line database_tpl.qtpl:224
		qw422016.N().S(`'
func (d *Database) `)
//line database_tpl.qtpl:225
		qw422016.E().S(camelName)
//line database_tpl.qtpl:225
		qw422016.N().S(`(ctx context.Context,
				`)
//line database_tpl.qtpl:226
		c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:226
		qw422016.N().S(`
) error {
	return d.Conn.ExecDDL(ctx,
				`)
//line database_tpl.qtpl:226
		qw422016.N().S("`")
//line database_tpl.qtpl:229
		qw422016.E().S(sql)
//line database_tpl.qtpl:229
		qw422016.N().S(``)
//line database_tpl.qtpl:229
		qw422016.N().S("`")
//line database_tpl.qtpl:229
		qw422016.N().S(`,
				`)
//line database_tpl.qtpl:230
		c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:230
		qw422016.N().S(`
			)
}
`)
//line database_tpl.qtpl:233
	} else {
//line database_tpl.qtpl:234
		switch len(r.Columns()) {
//line database_tpl.qtpl:235
		case 0:
//line database_tpl.qtpl:236
			typeReturn = c.udtToReturnType(r.UdtName)

//line database_tpl.qtpl:237
		case 1:
//line database_tpl.qtpl:239
			param := r.Columns()[0]
			typeReturn, _ = c.chkTypes(param, strcase.ToCamel(param.Name()))

//line database_tpl.qtpl:242
		default:
//line database_tpl.qtpl:243
			typeReturn = fmt.Sprintf("*%sRowScanner", camelName)

//line database_tpl.qtpl:244
			c.StreamCreateRowScanner(qw422016, r, camelName)
//line database_tpl.qtpl:244
			qw422016.N().S(`
`)
//line database_tpl.qtpl:245
		}
//line database_tpl.qtpl:245
		qw422016.N().S(`// `)
//line database_tpl.qtpl:246
		qw422016.E().S(camelName)
//line database_tpl.qtpl:246
		qw422016.N().S(` run query with select DB function '`)
//line database_tpl.qtpl:246
		qw422016.E().S(name)
//line database_tpl.qtpl:246
		qw422016.N().S(`: `)
//line database_tpl.qtpl:246
		qw422016.E().S(r.UdtName)
//line database_tpl.qtpl:246
		qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:247
		qw422016.E().S(r.Comment)
//line database_tpl.qtpl:247
		qw422016.N().S(`'
// ATTENTION! It returns only 1 row `)
//line database_tpl.qtpl:248
		qw422016.E().S(typeReturn)
//line database_tpl.qtpl:248
		qw422016.N().S(`
func (d *Database) `)
//line database_tpl.qtpl:249
		qw422016.E().S(camelName)
//line database_tpl.qtpl:249
		qw422016.N().S(`(ctx context.Context,
				`)
//line database_tpl.qtpl:250
		c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:250
		qw422016.N().S(`
) (res `)
//line database_tpl.qtpl:251
		qw422016.E().S(typeReturn)
//line database_tpl.qtpl:251
		qw422016.N().S(`, err error) {
	err = d.Conn.SelectOneAndScan(ctx,
				&res,
				`)
//line database_tpl.qtpl:251
		qw422016.N().S("`")
//line database_tpl.qtpl:254
		qw422016.E().S(sql)
//line database_tpl.qtpl:254
		qw422016.N().S(`
				FETCH FIRST 1 ROW ONLY`)
//line database_tpl.qtpl:254
		qw422016.N().S("`")
//line database_tpl.qtpl:254
		qw422016.N().S(`,
				`)
//line database_tpl.qtpl:256
		c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:256
		qw422016.N().S(`
			)

	return
}
`)
//line database_tpl.qtpl:261
		if r.ReturnType() == "record" {
//line database_tpl.qtpl:262
			typeReturn = fmt.Sprintf("%sRowScanner", camelName)

//line database_tpl.qtpl:262
			qw422016.N().S(`// `)
//line database_tpl.qtpl:263
			qw422016.E().S(camelName)
//line database_tpl.qtpl:263
			qw422016.N().S(`Each run query with select DB function '`)
//line database_tpl.qtpl:263
			qw422016.E().S(name)
//line database_tpl.qtpl:263
			qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:264
			qw422016.E().S(r.Comment)
//line database_tpl.qtpl:264
			qw422016.N().S(`'
// each will get every row from &`)
//line database_tpl.qtpl:265
			qw422016.E().S(camelName)
//line database_tpl.qtpl:265
			qw422016.N().S(`sRowScanner
func (d *Database) `)
//line database_tpl.qtpl:266
			qw422016.E().S(camelName)
//line database_tpl.qtpl:266
			qw422016.N().S(`Each(ctx context.Context,
	each func(record *`)
//line database_tpl.qtpl:267
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:267
			qw422016.N().S(`) error,
	`)
//line database_tpl.qtpl:268
			c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:268
			qw422016.N().S(`
) error {
	res := &`)
//line database_tpl.qtpl:270
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:270
			qw422016.N().S(`{}
	err := d.Conn.SelectAndScanEach(ctx,
				func () error {
					defer func () {
					//	create new record
						*res = `)
//line database_tpl.qtpl:275
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:275
			qw422016.N().S(`{}
					}()

					if each != nil {
						return each(res)
					}
					return nil
				},
				res,
				`)
//line database_tpl.qtpl:275
			qw422016.N().S("`")
//line database_tpl.qtpl:284
			qw422016.E().S(sql)
//line database_tpl.qtpl:284
			qw422016.N().S(``)
//line database_tpl.qtpl:284
			qw422016.N().S("`")
//line database_tpl.qtpl:284
			qw422016.N().S(`,
				`)
//line database_tpl.qtpl:285
			c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:285
			qw422016.N().S(`
		)
	return err
}
// `)
//line database_tpl.qtpl:289
			qw422016.E().S(camelName)
//line database_tpl.qtpl:289
			qw422016.N().S(`All run query with select DB function '`)
//line database_tpl.qtpl:289
			qw422016.E().S(name)
//line database_tpl.qtpl:289
			qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:290
			qw422016.E().S(r.Comment)
//line database_tpl.qtpl:290
			qw422016.N().S(`'
// each will get every row from &`)
//line database_tpl.qtpl:291
			qw422016.E().S(camelName)
//line database_tpl.qtpl:291
			qw422016.N().S(`sRowScanner
func (d *Database) `)
//line database_tpl.qtpl:292
			qw422016.E().S(camelName)
//line database_tpl.qtpl:292
			qw422016.N().S(`All(ctx context.Context,
	`)
//line database_tpl.qtpl:293
			c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:293
			qw422016.N().S(`
) (res []`)
//line database_tpl.qtpl:294
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:294
			qw422016.N().S(`, err error) {
	buf := `)
//line database_tpl.qtpl:295
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:295
			qw422016.N().S(`{}
	err = d.Conn.SelectAndScanEach(ctx,
				func () error {
					res = append(res, buf)
					//	create new record
					buf = `)
//line database_tpl.qtpl:300
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:300
			qw422016.N().S(`{}

					return nil
				},
				&buf,
				`)
//line database_tpl.qtpl:300
			qw422016.N().S("`")
//line database_tpl.qtpl:305
			qw422016.E().S(sql)
//line database_tpl.qtpl:305
			qw422016.N().S(``)
//line database_tpl.qtpl:305
			qw422016.N().S("`")
//line database_tpl.qtpl:305
			qw422016.N().S(`,
				`)
//line database_tpl.qtpl:306
			c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:306
			qw422016.N().S(`
		)
	return
}
`)
//line database_tpl.qtpl:310
		}
//line database_tpl.qtpl:310
		qw422016.N().S(`

`)
//line database_tpl.qtpl:312
	}
//line database_tpl.qtpl:312
	qw422016.N().S(`
`)
//line database_tpl.qtpl:313
}

//line database_tpl.qtpl:313
func (c *Creator) WriteCreateRoutinesInvoker(qq422016 qtio422016.Writer, r *psql.Routine, name string) {
//line database_tpl.qtpl:313
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:313
	c.StreamCreateRoutinesInvoker(qw422016, r, name)
//line database_tpl.qtpl:313
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:313
}

//line database_tpl.qtpl:313
func (c *Creator) CreateRoutinesInvoker(r *psql.Routine, name string) string {
//line database_tpl.qtpl:313
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:313
	c.WriteCreateRoutinesInvoker(qb422016, r, name)
//line database_tpl.qtpl:313
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:313
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:313
	return qs422016
//line database_tpl.qtpl:313
}

//line database_tpl.qtpl:315
func (c *Creator) StreamCreateRowScanner(qw422016 *qt422016.Writer, r *psql.Routine, camelName string) {
//line database_tpl.qtpl:315
	qw422016.N().S(`// `)
//line database_tpl.qtpl:316
	qw422016.E().S(camelName)
//line database_tpl.qtpl:316
	qw422016.N().S(`RowScanner run query with select
type `)
//line database_tpl.qtpl:317
	qw422016.E().S(camelName)
//line database_tpl.qtpl:317
	qw422016.N().S(`RowScanner struct {
`)
//line database_tpl.qtpl:318
	for _, param := range r.Columns() {
//line database_tpl.qtpl:320
		s := strcase.ToCamel(param.Name())
		typeCol, _ := c.chkTypes(param, s)

//line database_tpl.qtpl:322
		qw422016.N().S(`	`)
//line database_tpl.qtpl:323
		qw422016.E().S(s)
//line database_tpl.qtpl:323
		qw422016.N().S(`    `)
//line database_tpl.qtpl:323
		qw422016.E().S(typeCol)
//line database_tpl.qtpl:323
		qw422016.N().S(`  `)
//line database_tpl.qtpl:323
		qw422016.N().S("`")
//line database_tpl.qtpl:323
		qw422016.N().S(`json:"`)
//line database_tpl.qtpl:323
		qw422016.E().S(param.Name())
//line database_tpl.qtpl:323
		qw422016.N().S(`"`)
//line database_tpl.qtpl:323
		qw422016.N().S("`")
//line database_tpl.qtpl:323
		qw422016.N().S(`
`)
//line database_tpl.qtpl:324
	}
//line database_tpl.qtpl:324
	qw422016.N().S(`}
// GetFields implement dbEngine.RowScanner interface
func (r *`)
//line database_tpl.qtpl:327
	qw422016.E().S(camelName)
//line database_tpl.qtpl:327
	qw422016.N().S(`RowScanner) GetFields(columns []dbEngine.Column) []any {
	v := make([]any, len(columns))
	for i, col := range columns {
		switch col.Name() {
`)
//line database_tpl.qtpl:331
	for _, param := range r.Columns() {
//line database_tpl.qtpl:331
		qw422016.N().S(`		case "`)
//line database_tpl.qtpl:332
		qw422016.E().S(param.Name())
//line database_tpl.qtpl:332
		qw422016.N().S(`":
			v[i] =  &r.`)
//line database_tpl.qtpl:333
		qw422016.N().S(strcase.ToCamel(param.Name()))
//line database_tpl.qtpl:333
		qw422016.N().S(`
`)
//line database_tpl.qtpl:334
	}
//line database_tpl.qtpl:334
	qw422016.N().S(`		}
	}

	return v
}
`)
//line database_tpl.qtpl:340
}

//line database_tpl.qtpl:340
func (c *Creator) WriteCreateRowScanner(qq422016 qtio422016.Writer, r *psql.Routine, camelName string) {
//line database_tpl.qtpl:340
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:340
	c.StreamCreateRowScanner(qw422016, r, camelName)
//line database_tpl.qtpl:340
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:340
}

//line database_tpl.qtpl:340
func (c *Creator) CreateRowScanner(r *psql.Routine, camelName string) string {
//line database_tpl.qtpl:340
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:340
	c.WriteCreateRowScanner(qb422016, r, camelName)
//line database_tpl.qtpl:340
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:340
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:340
	return qs422016
//line database_tpl.qtpl:340
}

//line database_tpl.qtpl:342
func (c *Creator) streamparamsTitle(qw422016 *qt422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:343
	for _, param := range r.Params() {
//line database_tpl.qtpl:345
		s := strcase.ToLowerCamel(param.Name())
		typeCol, _ := c.chkTypes(param, s)

//line database_tpl.qtpl:347
		qw422016.N().S(`	`)
//line database_tpl.qtpl:348
		qw422016.E().S(s)
//line database_tpl.qtpl:348
		qw422016.N().S(` `)
//line database_tpl.qtpl:348
		qw422016.E().S(typeCol)
//line database_tpl.qtpl:348
		qw422016.N().S(`,
`)
//line database_tpl.qtpl:349
	}
//line database_tpl.qtpl:350
}

//line database_tpl.qtpl:350
func (c *Creator) writeparamsTitle(qq422016 qtio422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:350
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:350
	c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:350
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:350
}

//line database_tpl.qtpl:350
func (c *Creator) paramsTitle(r *psql.Routine) string {
//line database_tpl.qtpl:350
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:350
	c.writeparamsTitle(qb422016, r)
//line database_tpl.qtpl:350
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:350
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:350
	return qs422016
//line database_tpl.qtpl:350
}

//line database_tpl.qtpl:351
func (c *Creator) streamparamsArgs(qw422016 *qt422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:352
	for _, param := range r.Params() {
//line database_tpl.qtpl:352
		qw422016.N().S(`	`)
//line database_tpl.qtpl:353
		qw422016.E().S(strcase.ToLowerCamel(param.Name()))
//line database_tpl.qtpl:353
		qw422016.N().S(`,
`)
//line database_tpl.qtpl:354
	}
//line database_tpl.qtpl:355
}

//line database_tpl.qtpl:355
func (c *Creator) writeparamsArgs(qq422016 qtio422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:355
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:355
	c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:355
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:355
}

//line database_tpl.qtpl:355
func (c *Creator) paramsArgs(r *psql.Routine) string {
//line database_tpl.qtpl:355
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:355
	c.writeparamsArgs(qb422016, r)
//line database_tpl.qtpl:355
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:355
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:355
	return qs422016
//line database_tpl.qtpl:355
}

//line database_tpl.qtpl:357
func (c *Creator) StreamCreateTableConstructor(qw422016 *qt422016.Writer, cName, name string) {
//line database_tpl.qtpl:357
	qw422016.N().S(`// New`)
//line database_tpl.qtpl:358
	qw422016.E().S(cName)
//line database_tpl.qtpl:358
	qw422016.N().S(` create new instance of table `)
//line database_tpl.qtpl:358
	qw422016.E().S(cName)
//line database_tpl.qtpl:358
	qw422016.N().S(`
func (d *Database) New`)
//line database_tpl.qtpl:359
	qw422016.E().S(cName)
//line database_tpl.qtpl:359
	qw422016.N().S(`(ctx context.Context) (*`)
//line database_tpl.qtpl:359
	qw422016.E().S(cName)
//line database_tpl.qtpl:359
	qw422016.N().S(`, error) {
	const name = "`)
//line database_tpl.qtpl:360
	qw422016.E().S(name)
//line database_tpl.qtpl:360
	qw422016.N().S(`"
	table, ok := d.Tables[name]
    if !ok {
		var err error
		table, err = New`)
//line database_tpl.qtpl:364
	qw422016.E().S(cName)
//line database_tpl.qtpl:364
	qw422016.N().S(`FromConn(ctx, d.PsqlConn())
		if err != nil {
			return nil, err
		}
		d.Tables[name] = table
    }

    return &`)
//line database_tpl.qtpl:371
	qw422016.E().S(cName)
//line database_tpl.qtpl:371
	qw422016.N().S(`{
		Table: table.(*psql.Table),
    }, nil
}
`)
//line database_tpl.qtpl:375
}

//line database_tpl.qtpl:375
func (c *Creator) WriteCreateTableConstructor(qq422016 qtio422016.Writer, cName, name string) {
//line database_tpl.qtpl:375
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:375
	c.StreamCreateTableConstructor(qw422016, cName, name)
//line database_tpl.qtpl:375
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:375
}

//line database_tpl.qtpl:375
func (c *Creator) CreateTableConstructor(cName, name string) string {
//line database_tpl.qtpl:375
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:375
	c.WriteCreateTableConstructor(qb422016, cName, name)
//line database_tpl.qtpl:375
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:375
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:375
	return qs422016
//line database_tpl.qtpl:375
}
