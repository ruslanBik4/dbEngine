// Code generated by qtc from "database_tpl.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line database_tpl.qtpl:1
package _go

//line database_tpl.qtpl:2
import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/typesExt"
)

//line database_tpl.qtpl:14
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line database_tpl.qtpl:14
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line database_tpl.qtpl:14
func (c *Creator) StreamCreateDatabase(qw422016 *qt422016.Writer, listRoutines []string) {
//line database_tpl.qtpl:14
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
//line database_tpl.qtpl:25
	for _, lib := range c.packages {
//line database_tpl.qtpl:25
		qw422016.N().S(`"`)
//line database_tpl.qtpl:25
		qw422016.E().S(lib)
//line database_tpl.qtpl:25
		qw422016.N().S(`"
`)
//line database_tpl.qtpl:26
	}
//line database_tpl.qtpl:26
	qw422016.N().S(`
	"github.com/jackc/pgconn"

	"github.com/ruslanBik4/logs"
	"github.com/ruslanBik4/dbEngine/dbEngine"
    "github.com/ruslanBik4/dbEngine/dbEngine/psql"

	"golang.org/x/net/context"
	"github.com/pkg/errors"
)
`)
//line database_tpl.qtpl:37
	for name, typ := range c.db.Types {
//line database_tpl.qtpl:37
		c.StreamCreateTypeInterface(qw422016, typ, strcase.ToCamel(name), name, c.Types[name])
//line database_tpl.qtpl:37
	}
//line database_tpl.qtpl:37
	qw422016.N().S(`
// Database is root interface for operation for %s.%s
type Database struct {
	*dbEngine.DB
	CreateAt time.Time
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
//line database_tpl.qtpl:69
	for name := range c.db.Tables {
//line database_tpl.qtpl:69
		c.StreamCreateTableConstructor(qw422016, strcase.ToCamel(name), name)
//line database_tpl.qtpl:69
	}
//line database_tpl.qtpl:70
	for _, name := range listRoutines {
//line database_tpl.qtpl:70
		c.StreamCreateRoutinesInvoker(qw422016, c.db.Routines[name].(*psql.Routine), name)
//line database_tpl.qtpl:70
	}
//line database_tpl.qtpl:70
	qw422016.N().S(`// printNotice logging some psql messages (invoked command 'RAISE ...')
func printNotice(c *pgconn.PgConn, n *pgconn.Notice) {

	switch {
    case n.Code == "42P07" || strings.Contains(n.Message, "skipping") :
		logs.DebugLog("skip operation: %%s", n.Message)

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
//line database_tpl.qtpl:93
}

//line database_tpl.qtpl:93
func (c *Creator) WriteCreateDatabase(qq422016 qtio422016.Writer, listRoutines []string) {
//line database_tpl.qtpl:93
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:93
	c.StreamCreateDatabase(qw422016, listRoutines)
//line database_tpl.qtpl:93
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:93
}

//line database_tpl.qtpl:93
func (c *Creator) CreateDatabase(listRoutines []string) string {
//line database_tpl.qtpl:93
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:93
	c.WriteCreateDatabase(qb422016, listRoutines)
//line database_tpl.qtpl:93
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:93
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:93
	return qs422016
//line database_tpl.qtpl:93
}

//line database_tpl.qtpl:95
func (c *Creator) StreamCreateTypeInterface(qw422016 *qt422016.Writer, t dbEngine.Types, typeName, name, typeCol string) {
//line database_tpl.qtpl:96
	if len(t.Enumerates) > 0 {
//line database_tpl.qtpl:97
	} else {
//line database_tpl.qtpl:97
		qw422016.N().S(`// `)
//line database_tpl.qtpl:98
		qw422016.E().S(typeName)
//line database_tpl.qtpl:98
		qw422016.N().S(` create new instance of type `)
//line database_tpl.qtpl:98
		qw422016.E().S(name)
//line database_tpl.qtpl:98
		qw422016.N().S(`
type `)
//line database_tpl.qtpl:99
		qw422016.E().S(typeName)
//line database_tpl.qtpl:99
		qw422016.N().S(` struct {
`)
//line database_tpl.qtpl:100
		for _, tAttr := range t.Attr {
//line database_tpl.qtpl:100
			qw422016.N().S(`	`)
//line database_tpl.qtpl:101
			qw422016.N().S(fmt.Sprintf("%-21s\t%-13s\t", strcase.ToCamel(tAttr.Name), tAttr.Type))
//line database_tpl.qtpl:101
			qw422016.N().S(` `)
//line database_tpl.qtpl:101
			qw422016.N().S("`")
//line database_tpl.qtpl:101
			qw422016.N().S(`json:"`)
//line database_tpl.qtpl:101
			qw422016.E().S(tAttr.Name)
//line database_tpl.qtpl:101
			qw422016.N().S(`"`)
//line database_tpl.qtpl:101
			qw422016.N().S("`")
//line database_tpl.qtpl:101
			qw422016.N().S(`
`)
//line database_tpl.qtpl:102
		}
//line database_tpl.qtpl:102
		qw422016.N().S(`}

func (dst *`)
//line database_tpl.qtpl:105
		qw422016.E().S(typeName)
//line database_tpl.qtpl:105
		qw422016.N().S(`) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = `)
//line database_tpl.qtpl:107
		qw422016.E().S(typeName)
//line database_tpl.qtpl:107
		qw422016.N().S(`{}
		return nil
	}
	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))
    *dst = `)
//line database_tpl.qtpl:111
		qw422016.E().S(typeName)
//line database_tpl.qtpl:111
		qw422016.N().S(`{
`)
//line database_tpl.qtpl:112
		for i, tAttr := range t.Attr {
//line database_tpl.qtpl:112
			qw422016.N().S(`		`)
//line database_tpl.qtpl:113
			qw422016.N().S(fmt.Sprintf(`%-21s:    psql.Get%sFromByte(ci, srcPart[%d], "%s")`,
				strcase.ToCamel(tAttr.Name),
				strcase.ToCamel(strings.ReplaceAll(tAttr.Type, "[]", "Array")),
				i,
				tAttr.Name))
//line database_tpl.qtpl:117
			qw422016.N().S(`,
`)
//line database_tpl.qtpl:118
			i++

//line database_tpl.qtpl:119
		}
//line database_tpl.qtpl:119
		qw422016.N().S(`	}

	return nil
}
`)
//line database_tpl.qtpl:124
	}
//line database_tpl.qtpl:125
}

//line database_tpl.qtpl:125
func (c *Creator) WriteCreateTypeInterface(qq422016 qtio422016.Writer, t dbEngine.Types, typeName, name, typeCol string) {
//line database_tpl.qtpl:125
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:125
	c.StreamCreateTypeInterface(qw422016, t, typeName, name, typeCol)
//line database_tpl.qtpl:125
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:125
}

//line database_tpl.qtpl:125
func (c *Creator) CreateTypeInterface(t dbEngine.Types, typeName, name, typeCol string) string {
//line database_tpl.qtpl:125
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:125
	c.WriteCreateTypeInterface(qb422016, t, typeName, name, typeCol)
//line database_tpl.qtpl:125
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:125
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:125
	return qs422016
//line database_tpl.qtpl:125
}

//line database_tpl.qtpl:127
func (c *Creator) StreamCreateRoutinesInvoker(qw422016 *qt422016.Writer, r *psql.Routine, name string) {
//line database_tpl.qtpl:129
	camelName := strcase.ToCamel(name)
	args := make([]any, len(r.Params()))
	sql, _, _ := r.BuildSql(dbEngine.ArgsForSelect(args...))
	typeReturn := ""

//line database_tpl.qtpl:134
	if r.Type == psql.ROUTINE_TYPE_PROC {
//line database_tpl.qtpl:134
		qw422016.N().S(`
// `)
//line database_tpl.qtpl:135
		qw422016.E().S(camelName)
//line database_tpl.qtpl:135
		qw422016.N().S(` call procedure '`)
//line database_tpl.qtpl:135
		qw422016.E().S(name)
//line database_tpl.qtpl:135
		qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:136
		qw422016.E().S(r.Comment)
//line database_tpl.qtpl:136
		qw422016.N().S(`'
func (d *Database) `)
//line database_tpl.qtpl:137
		qw422016.E().S(camelName)
//line database_tpl.qtpl:137
		qw422016.N().S(`(ctx context.Context,
				`)
//line database_tpl.qtpl:138
		c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:138
		qw422016.N().S(`
) error {
	return d.Conn.ExecDDL(ctx,
				`)
//line database_tpl.qtpl:138
		qw422016.N().S("`")
//line database_tpl.qtpl:141
		qw422016.E().S(sql)
//line database_tpl.qtpl:141
		qw422016.N().S(``)
//line database_tpl.qtpl:141
		qw422016.N().S("`")
//line database_tpl.qtpl:141
		qw422016.N().S(`,
				`)
//line database_tpl.qtpl:142
		c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:142
		qw422016.N().S(`
			)
}
`)
//line database_tpl.qtpl:145
	} else {
//line database_tpl.qtpl:145
		qw422016.N().S(`
`)
//line database_tpl.qtpl:146
		switch len(r.Columns()) {
//line database_tpl.qtpl:147
		case 0:
//line database_tpl.qtpl:147
			qw422016.N().S(`
`)
//line database_tpl.qtpl:149
			toType := psql.UdtNameToType(r.UdtName)
			typeReturn = typesExt.Basic(toType).String()

//line database_tpl.qtpl:152
		case 1:
//line database_tpl.qtpl:152
			qw422016.N().S(`
`)
//line database_tpl.qtpl:154
			param := r.Columns()[0]
			typeReturn, _ = c.chkTypes(param, strcase.ToCamel(param.Name()))

//line database_tpl.qtpl:157
		default:
//line database_tpl.qtpl:157
			qw422016.N().S(`// `)
//line database_tpl.qtpl:158
			qw422016.E().S(camelName)
//line database_tpl.qtpl:158
			qw422016.N().S(`RowScanner run query with select
type `)
//line database_tpl.qtpl:159
			qw422016.E().S(camelName)
//line database_tpl.qtpl:159
			qw422016.N().S(`RowScanner struct {
`)
//line database_tpl.qtpl:160
			for _, param := range r.Columns() {
//line database_tpl.qtpl:162
				s := strcase.ToCamel(param.Name())
				typeCol, _ := c.chkTypes(param, s)

//line database_tpl.qtpl:164
				qw422016.N().S(`	`)
//line database_tpl.qtpl:165
				qw422016.E().S(s)
//line database_tpl.qtpl:165
				qw422016.N().S(` `)
//line database_tpl.qtpl:165
				qw422016.E().S(typeCol)
//line database_tpl.qtpl:165
				qw422016.N().S(`
`)
//line database_tpl.qtpl:166
			}
//line database_tpl.qtpl:166
			qw422016.N().S(`}
// GetFields implement dbEngine.RowScanner interface
func (r *`)
//line database_tpl.qtpl:169
			qw422016.E().S(camelName)
//line database_tpl.qtpl:169
			qw422016.N().S(`RowScanner) GetFields(columns []dbEngine.Column) []any {
	v := make([]any, len(columns))
	for i, col := range columns {
		switch col.Name() {
`)
//line database_tpl.qtpl:173
			for _, param := range r.Columns() {
//line database_tpl.qtpl:173
				qw422016.N().S(`		case "`)
//line database_tpl.qtpl:174
				qw422016.E().S(param.Name())
//line database_tpl.qtpl:174
				qw422016.N().S(`":
			v[i] =  &r.`)
//line database_tpl.qtpl:175
				qw422016.N().S(strcase.ToCamel(param.Name()))
//line database_tpl.qtpl:175
				qw422016.N().S(`
`)
//line database_tpl.qtpl:176
			}
//line database_tpl.qtpl:176
			qw422016.N().S(`		}
	}

	return v
}
`)
//line database_tpl.qtpl:182
		}
//line database_tpl.qtpl:182
		qw422016.N().S(`// `)
//line database_tpl.qtpl:183
		qw422016.E().S(camelName)
//line database_tpl.qtpl:183
		qw422016.N().S(` run query with select DB function '`)
//line database_tpl.qtpl:183
		qw422016.E().S(name)
//line database_tpl.qtpl:183
		qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:184
		qw422016.E().S(r.Comment)
//line database_tpl.qtpl:184
		qw422016.N().S(`'
// ATTENTION! It returns only 1 row
func (d *Database) `)
//line database_tpl.qtpl:186
		qw422016.E().S(camelName)
//line database_tpl.qtpl:186
		qw422016.N().S(`(ctx context.Context,
				`)
//line database_tpl.qtpl:187
		c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:187
		qw422016.N().S(`
) (res `)
//line database_tpl.qtpl:188
		if len(r.Columns()) > 1 {
//line database_tpl.qtpl:188
			qw422016.N().S(`*`)
//line database_tpl.qtpl:188
			qw422016.E().S(camelName)
//line database_tpl.qtpl:188
			qw422016.N().S(`RowScanner`)
//line database_tpl.qtpl:188
		} else {
//line database_tpl.qtpl:188
			qw422016.E().S(typeReturn)
//line database_tpl.qtpl:188
		}
//line database_tpl.qtpl:188
		qw422016.N().S(`, err error) {
	err = d.Conn.SelectOneAndScan(ctx,
				&res,
				`)
//line database_tpl.qtpl:188
		qw422016.N().S("`")
//line database_tpl.qtpl:191
		qw422016.E().S(sql)
//line database_tpl.qtpl:191
		qw422016.N().S(`
				FETCH FIRST 1 ROW ONLY`)
//line database_tpl.qtpl:191
		qw422016.N().S("`")
//line database_tpl.qtpl:191
		qw422016.N().S(`,
				`)
//line database_tpl.qtpl:193
		c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:193
		qw422016.N().S(`
			)

	return
}
`)
//line database_tpl.qtpl:198
		if r.ReturnType() == "record" {
//line database_tpl.qtpl:198
			qw422016.N().S(`// `)
//line database_tpl.qtpl:199
			qw422016.E().S(camelName)
//line database_tpl.qtpl:199
			qw422016.N().S(` run query with select DB function '`)
//line database_tpl.qtpl:199
			qw422016.E().S(name)
//line database_tpl.qtpl:199
			qw422016.N().S(`'
// DB comment: '`)
//line database_tpl.qtpl:200
			qw422016.E().S(r.Comment)
//line database_tpl.qtpl:200
			qw422016.N().S(`'
// each will get every row from %[1]sRowScanner
func (d *Database) `)
//line database_tpl.qtpl:202
			qw422016.E().S(camelName)
//line database_tpl.qtpl:202
			qw422016.N().S(`Each(ctx context.Context,
	each func(record *`)
//line database_tpl.qtpl:203
			qw422016.E().S(camelName)
//line database_tpl.qtpl:203
			qw422016.N().S(`RowScanner) error,
	`)
//line database_tpl.qtpl:204
			c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:204
			qw422016.N().S(`
) error {
	res := &`)
//line database_tpl.qtpl:206
			qw422016.E().S(camelName)
//line database_tpl.qtpl:206
			qw422016.N().S(`RowScanner{}
	err := d.Conn.SelectAndScanEach(ctx,
				func () error {
					if each != nil {
						return each(res)
					}
					return nil
				},
				res,
				`)
//line database_tpl.qtpl:206
			qw422016.N().S("`")
//line database_tpl.qtpl:215
			qw422016.E().S(sql)
//line database_tpl.qtpl:215
			qw422016.N().S(``)
//line database_tpl.qtpl:215
			qw422016.N().S("`")
//line database_tpl.qtpl:215
			qw422016.N().S(`,
				`)
//line database_tpl.qtpl:216
			c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:216
			qw422016.N().S(`
		)
	return err
}
`)
//line database_tpl.qtpl:220
		}
//line database_tpl.qtpl:220
		qw422016.N().S(`
`)
//line database_tpl.qtpl:221
	}
//line database_tpl.qtpl:221
	qw422016.N().S(`
`)
//line database_tpl.qtpl:222
}

//line database_tpl.qtpl:222
func (c *Creator) WriteCreateRoutinesInvoker(qq422016 qtio422016.Writer, r *psql.Routine, name string) {
//line database_tpl.qtpl:222
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:222
	c.StreamCreateRoutinesInvoker(qw422016, r, name)
//line database_tpl.qtpl:222
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:222
}

//line database_tpl.qtpl:222
func (c *Creator) CreateRoutinesInvoker(r *psql.Routine, name string) string {
//line database_tpl.qtpl:222
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:222
	c.WriteCreateRoutinesInvoker(qb422016, r, name)
//line database_tpl.qtpl:222
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:222
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:222
	return qs422016
//line database_tpl.qtpl:222
}

//line database_tpl.qtpl:223
func (c *Creator) streamparamsTitle(qw422016 *qt422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:224
	for _, param := range r.Params() {
//line database_tpl.qtpl:226
		s := strcase.ToLowerCamel(param.Name())
		typeCol, _ := c.chkTypes(param, s)

//line database_tpl.qtpl:228
		qw422016.N().S(`		`)
//line database_tpl.qtpl:229
		qw422016.E().S(s)
//line database_tpl.qtpl:229
		qw422016.N().S(` `)
//line database_tpl.qtpl:229
		qw422016.E().S(typeCol)
//line database_tpl.qtpl:229
		qw422016.N().S(`,
`)
//line database_tpl.qtpl:230
	}
//line database_tpl.qtpl:231
}

//line database_tpl.qtpl:231
func (c *Creator) writeparamsTitle(qq422016 qtio422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:231
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:231
	c.streamparamsTitle(qw422016, r)
//line database_tpl.qtpl:231
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:231
}

//line database_tpl.qtpl:231
func (c *Creator) paramsTitle(r *psql.Routine) string {
//line database_tpl.qtpl:231
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:231
	c.writeparamsTitle(qb422016, r)
//line database_tpl.qtpl:231
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:231
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:231
	return qs422016
//line database_tpl.qtpl:231
}

//line database_tpl.qtpl:232
func (c *Creator) streamparamsArgs(qw422016 *qt422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:233
	for _, param := range r.Params() {
//line database_tpl.qtpl:233
		qw422016.N().S(`		`)
//line database_tpl.qtpl:234
		qw422016.E().S(strcase.ToLowerCamel(param.Name()))
//line database_tpl.qtpl:234
		qw422016.N().S(`,
`)
//line database_tpl.qtpl:235
	}
//line database_tpl.qtpl:236
}

//line database_tpl.qtpl:236
func (c *Creator) writeparamsArgs(qq422016 qtio422016.Writer, r *psql.Routine) {
//line database_tpl.qtpl:236
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:236
	c.streamparamsArgs(qw422016, r)
//line database_tpl.qtpl:236
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:236
}

//line database_tpl.qtpl:236
func (c *Creator) paramsArgs(r *psql.Routine) string {
//line database_tpl.qtpl:236
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:236
	c.writeparamsArgs(qb422016, r)
//line database_tpl.qtpl:236
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:236
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:236
	return qs422016
//line database_tpl.qtpl:236
}

//line database_tpl.qtpl:237
func (c *Creator) StreamCreateTableConstructor(qw422016 *qt422016.Writer, cName, name string) {
//line database_tpl.qtpl:237
	qw422016.N().S(`// New`)
//line database_tpl.qtpl:238
	qw422016.E().S(cName)
//line database_tpl.qtpl:238
	qw422016.N().S(` create new instance of table `)
//line database_tpl.qtpl:238
	qw422016.E().S(cName)
//line database_tpl.qtpl:238
	qw422016.N().S(`
func (d *Database) New`)
//line database_tpl.qtpl:239
	qw422016.E().S(cName)
//line database_tpl.qtpl:239
	qw422016.N().S(`(ctx context.Context) (*`)
//line database_tpl.qtpl:239
	qw422016.E().S(cName)
//line database_tpl.qtpl:239
	qw422016.N().S(`, error) {
	const name = "`)
//line database_tpl.qtpl:240
	qw422016.E().S(name)
//line database_tpl.qtpl:240
	qw422016.N().S(`"
	table, ok := d.Tables[name]
    if !ok {
		var err error
		table, err = New`)
//line database_tpl.qtpl:244
	qw422016.E().S(cName)
//line database_tpl.qtpl:244
	qw422016.N().S(`FromConn(ctx, d.PsqlConn())
		if err != nil {
			return nil, err
		}
		d.Tables[name] = table
    }

    return &`)
//line database_tpl.qtpl:251
	qw422016.E().S(cName)
//line database_tpl.qtpl:251
	qw422016.N().S(`{
		Table: table.(*psql.Table),
    }, nil
}
`)
//line database_tpl.qtpl:255
}

//line database_tpl.qtpl:255
func (c *Creator) WriteCreateTableConstructor(qq422016 qtio422016.Writer, cName, name string) {
//line database_tpl.qtpl:255
	qw422016 := qt422016.AcquireWriter(qq422016)
//line database_tpl.qtpl:255
	c.StreamCreateTableConstructor(qw422016, cName, name)
//line database_tpl.qtpl:255
	qt422016.ReleaseWriter(qw422016)
//line database_tpl.qtpl:255
}

//line database_tpl.qtpl:255
func (c *Creator) CreateTableConstructor(cName, name string) string {
//line database_tpl.qtpl:255
	qb422016 := qt422016.AcquireByteBuffer()
//line database_tpl.qtpl:255
	c.WriteCreateTableConstructor(qb422016, cName, name)
//line database_tpl.qtpl:255
	qs422016 := string(qb422016.B)
//line database_tpl.qtpl:255
	qt422016.ReleaseByteBuffer(qb422016)
//line database_tpl.qtpl:255
	return qs422016
//line database_tpl.qtpl:255
}
