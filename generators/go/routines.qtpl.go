// Code generated by qtc from "routines.qtpl". DO NOT EDIT.
// See https://github.com/valyala/quicktemplate for details.

//line routines.qtpl:1
package _go

//line routines.qtpl:2
import (
	"fmt"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
)

//line routines.qtpl:12
import (
	qtio422016 "io"

	qt422016 "github.com/valyala/quicktemplate"
)

//line routines.qtpl:12
var (
	_ = qtio422016.Copy
	_ = qt422016.AcquireByteBuffer
)

//line routines.qtpl:12
func (c *Creator) StreamCreateRoutinesInvoker(qw422016 *qt422016.Writer, r *psql.Routine, name string) {
//line routines.qtpl:14
	camelName := strcase.ToCamel(name)
	args := make([]any, len(r.Params()))
	sql, _, _ := r.BuildSql(dbEngine.ArgsForSelect(args...))

//line routines.qtpl:18
	if r.Type == psql.ROUTINE_TYPE_PROC {
//line routines.qtpl:18
		qw422016.N().S(`// `)
//line routines.qtpl:19
		qw422016.E().S(camelName)
//line routines.qtpl:19
		qw422016.N().S(` call procedure '`)
//line routines.qtpl:19
		qw422016.E().S(name)
//line routines.qtpl:19
		qw422016.N().S(`'
`)
//line routines.qtpl:20
		if r.Comment > "" {
//line routines.qtpl:20
			qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:20
			qw422016.E().S(r.Comment)
//line routines.qtpl:20
			qw422016.N().S(`'
`)
//line routines.qtpl:21
		}
//line routines.qtpl:21
		qw422016.N().S(`
func (d *Database) `)
//line routines.qtpl:22
		qw422016.E().S(camelName)
//line routines.qtpl:22
		qw422016.N().S(`(
	ctx context.Context,`)
//line routines.qtpl:23
		c.streamparamsTitle(qw422016, r)
//line routines.qtpl:23
		qw422016.N().S(`
) error {
	return d.Conn.ExecDDL(
		ctx,
		`)
//line routines.qtpl:23
		qw422016.N().S("`")
//line routines.qtpl:27
		qw422016.E().S(sql)
//line routines.qtpl:27
		qw422016.N().S(``)
//line routines.qtpl:27
		qw422016.N().S("`")
//line routines.qtpl:27
		qw422016.N().S(`,
		`)
//line routines.qtpl:28
		c.streamparamsArgs(qw422016, r)
//line routines.qtpl:28
		qw422016.N().S(`)
}
`)
//line routines.qtpl:30
	} else {
//line routines.qtpl:31
		c.StreamCreateFunctionInvoker(qw422016, r, name, camelName, sql)
//line routines.qtpl:31
		qw422016.N().S(`
`)
//line routines.qtpl:32
	}
//line routines.qtpl:33
}

//line routines.qtpl:33
func (c *Creator) WriteCreateRoutinesInvoker(qq422016 qtio422016.Writer, r *psql.Routine, name string) {
//line routines.qtpl:33
	qw422016 := qt422016.AcquireWriter(qq422016)
//line routines.qtpl:33
	c.StreamCreateRoutinesInvoker(qw422016, r, name)
//line routines.qtpl:33
	qt422016.ReleaseWriter(qw422016)
//line routines.qtpl:33
}

//line routines.qtpl:33
func (c *Creator) CreateRoutinesInvoker(r *psql.Routine, name string) string {
//line routines.qtpl:33
	qb422016 := qt422016.AcquireByteBuffer()
//line routines.qtpl:33
	c.WriteCreateRoutinesInvoker(qb422016, r, name)
//line routines.qtpl:33
	qs422016 := string(qb422016.B)
//line routines.qtpl:33
	qt422016.ReleaseByteBuffer(qb422016)
//line routines.qtpl:33
	return qs422016
//line routines.qtpl:33
}

// CreateRoutinesInvoker
//

//line routines.qtpl:35
func (c *Creator) StreamCreateFunctionInvoker(qw422016 *qt422016.Writer, r *psql.Routine, name, camelName, sql string) {
//line routines.qtpl:37
	typeReturn, initReturn, needReference := "", "", true

//line routines.qtpl:39
	switch len(r.Columns()) {
//line routines.qtpl:40
	case 0:
//line routines.qtpl:42
		typeReturn = c.udtToReturnType(r.UdtName)

//line routines.qtpl:44
	case 1:
//line routines.qtpl:46
		param := r.Columns()[0]
		typeReturn, _ = c.chkTypes(param, strcase.ToCamel(param.Name()))
		if a, ok := strings.CutPrefix(typeReturn, "[]"); param.BasicType() < 0 && ok {
			typeReturn = "WrapArray[*" + a + "]"
		}

//line routines.qtpl:52
	default:
//line routines.qtpl:53
		c.StreamCreateRowScanner(qw422016, r, camelName)
//line routines.qtpl:53
		qw422016.N().S(`
`)
//line routines.qtpl:55
		typeReturn = fmt.Sprintf("*%sRowScanner", camelName)
		initReturn = fmt.Sprintf("res = &%sRowScanner{}", camelName)
		needReference = false

//line routines.qtpl:59
	}
//line routines.qtpl:59
	qw422016.N().S(`// `)
//line routines.qtpl:60
	qw422016.E().S(camelName)
//line routines.qtpl:60
	qw422016.N().S(` run query with select DB function '`)
//line routines.qtpl:60
	qw422016.E().S(name)
//line routines.qtpl:60
	qw422016.N().S(`: `)
//line routines.qtpl:60
	qw422016.E().S(r.UdtName)
//line routines.qtpl:60
	qw422016.N().S(`'
`)
//line routines.qtpl:61
	if r.Comment > "" {
//line routines.qtpl:61
		qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:61
		qw422016.E().S(r.Comment)
//line routines.qtpl:61
		qw422016.N().S(`'
`)
//line routines.qtpl:62
	}
//line routines.qtpl:62
	qw422016.N().S(`// ATTENTION! It returns only 1 row `)
//line routines.qtpl:63
	qw422016.E().S(typeReturn)
//line routines.qtpl:63
	qw422016.N().S(`
// `)
//line routines.qtpl:64
	qw422016.N().S(fmt.Sprintf("%+v", r))
//line routines.qtpl:64
	qw422016.N().S(`
func (d *Database) `)
//line routines.qtpl:65
	qw422016.E().S(camelName)
//line routines.qtpl:65
	qw422016.N().S(`(
	ctx context.Context,`)
//line routines.qtpl:66
	c.streamparamsTitle(qw422016, r)
//line routines.qtpl:66
	qw422016.N().S(`
) (res `)
//line routines.qtpl:67
	qw422016.E().S(typeReturn)
//line routines.qtpl:67
	qw422016.N().S(`, err error) {
	`)
//line routines.qtpl:68
	qw422016.N().S(initReturn)
//line routines.qtpl:68
	qw422016.N().S(`
	err = d.Conn.SelectOneAndScan(ctx,
		`)
//line routines.qtpl:70
	if needReference {
//line routines.qtpl:70
		qw422016.N().S(`&`)
//line routines.qtpl:70
	}
//line routines.qtpl:70
	qw422016.N().S(`res,
		`)
//line routines.qtpl:70
	qw422016.N().S("`")
//line routines.qtpl:71
	qw422016.E().S(sql)
//line routines.qtpl:71
	qw422016.N().S(`
		FETCH FIRST 1 ROW ONLY`)
//line routines.qtpl:71
	qw422016.N().S("`")
//line routines.qtpl:71
	qw422016.N().S(`,
		`)
//line routines.qtpl:73
	c.streamparamsArgs(qw422016, r)
//line routines.qtpl:73
	qw422016.N().S(`)

	return
}
`)
//line routines.qtpl:77
	if r.ReturnType() == "json" || r.ReturnType() == "jsonb" {
//line routines.qtpl:77
		qw422016.N().S(`// `)
//line routines.qtpl:78
		qw422016.E().S(camelName)
//line routines.qtpl:78
		qw422016.N().S(`Out run query with select DB function '`)
//line routines.qtpl:78
		qw422016.E().S(name)
//line routines.qtpl:78
		qw422016.N().S(`: `)
//line routines.qtpl:78
		qw422016.E().S(r.UdtName)
//line routines.qtpl:78
		qw422016.N().S(`'
`)
//line routines.qtpl:79
		if r.Comment > "" {
//line routines.qtpl:79
			qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:79
			qw422016.E().S(r.Comment)
//line routines.qtpl:79
			qw422016.N().S(`'
`)
//line routines.qtpl:80
		}
//line routines.qtpl:80
		qw422016.N().S(`// ATTENTION! It returns only 1 row `)
//line routines.qtpl:81
		qw422016.E().S(typeReturn)
//line routines.qtpl:81
		qw422016.N().S(`
func (d *Database) `)
//line routines.qtpl:82
		qw422016.E().S(camelName)
//line routines.qtpl:82
		qw422016.N().S(`Out(
	ctx context.Context,
	res any,`)
//line routines.qtpl:84
		c.streamparamsTitle(qw422016, r)
//line routines.qtpl:84
		qw422016.N().S(`
) error {
	 return d.Conn.SelectOneAndScan(ctx,
		res,
		`)
//line routines.qtpl:84
		qw422016.N().S("`")
//line routines.qtpl:88
		qw422016.E().S(sql)
//line routines.qtpl:88
		qw422016.N().S(`
		FETCH FIRST 1 ROW ONLY`)
//line routines.qtpl:88
		qw422016.N().S("`")
//line routines.qtpl:88
		qw422016.N().S(`,
		`)
//line routines.qtpl:90
		c.streamparamsArgs(qw422016, r)
//line routines.qtpl:90
		qw422016.N().S(`)
}
`)
//line routines.qtpl:92
	}
//line routines.qtpl:93
	if r.ReturnType() == "record" {
//line routines.qtpl:93
		qw422016.N().S(`// `)
//line routines.qtpl:94
		qw422016.E().S(camelName)
//line routines.qtpl:94
		qw422016.N().S(`Out run query with select DB function '`)
//line routines.qtpl:94
		qw422016.E().S(name)
//line routines.qtpl:94
		qw422016.N().S(`'
`)
//line routines.qtpl:95
		if r.Comment > "" {
//line routines.qtpl:95
			qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:95
			qw422016.E().S(r.Comment)
//line routines.qtpl:95
			qw422016.N().S(`'
`)
//line routines.qtpl:96
		}
//line routines.qtpl:96
		qw422016.N().S(`// each will get every row from &`)
//line routines.qtpl:97
		qw422016.E().S(camelName)
//line routines.qtpl:97
		qw422016.N().S(`sRowScanner
func (d *Database) `)
//line routines.qtpl:98
		qw422016.E().S(camelName)
//line routines.qtpl:98
		qw422016.N().S(`Out(
	ctx context.Context,
	each func() error,
	res dbEngine.RowScanner,`)
//line routines.qtpl:101
		c.streamparamsTitle(qw422016, r)
//line routines.qtpl:101
		qw422016.N().S(`
) error {
	return d.Conn.SelectAndScanEach(ctx,
		each,
		res,
		`)
//line routines.qtpl:101
		qw422016.N().S("`")
//line routines.qtpl:106
		qw422016.E().S(sql)
//line routines.qtpl:106
		qw422016.N().S(``)
//line routines.qtpl:106
		qw422016.N().S("`")
//line routines.qtpl:106
		qw422016.N().S(`,
		`)
//line routines.qtpl:107
		c.streamparamsArgs(qw422016, r)
//line routines.qtpl:107
		qw422016.N().S(`)
}

`)
//line routines.qtpl:110
		typeReturn = fmt.Sprintf("%sRowScanner", camelName)

//line routines.qtpl:110
		qw422016.N().S(`// `)
//line routines.qtpl:111
		qw422016.E().S(camelName)
//line routines.qtpl:111
		qw422016.N().S(`Each run query with select DB function '`)
//line routines.qtpl:111
		qw422016.E().S(name)
//line routines.qtpl:111
		qw422016.N().S(`'
`)
//line routines.qtpl:112
		if r.Comment > "" {
//line routines.qtpl:112
			qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:112
			qw422016.E().S(r.Comment)
//line routines.qtpl:112
			qw422016.N().S(`'
`)
//line routines.qtpl:113
		}
//line routines.qtpl:113
		qw422016.N().S(`// each will get every row from &`)
//line routines.qtpl:114
		qw422016.E().S(camelName)
//line routines.qtpl:114
		qw422016.N().S(`sRowScanner
func (d *Database) `)
//line routines.qtpl:115
		qw422016.E().S(camelName)
//line routines.qtpl:115
		qw422016.N().S(`Each(
	ctx context.Context,
	each func(record *`)
//line routines.qtpl:117
		qw422016.E().S(typeReturn)
//line routines.qtpl:117
		qw422016.N().S(`) error,`)
//line routines.qtpl:117
		c.streamparamsTitle(qw422016, r)
//line routines.qtpl:117
		qw422016.N().S(`
) error {
	res := &`)
//line routines.qtpl:119
		qw422016.E().S(typeReturn)
//line routines.qtpl:119
		qw422016.N().S(`{}
	err := d.Conn.SelectAndScanEach(ctx,
		func() error {
			defer func() {
			//	create new record
				*res = `)
//line routines.qtpl:124
		qw422016.E().S(typeReturn)
//line routines.qtpl:124
		qw422016.N().S(`{}
			}()

			if each != nil {
				return each(res)
			}
			return nil
		},
		res,
		`)
//line routines.qtpl:124
		qw422016.N().S("`")
//line routines.qtpl:133
		qw422016.E().S(sql)
//line routines.qtpl:133
		qw422016.N().S(``)
//line routines.qtpl:133
		qw422016.N().S("`")
//line routines.qtpl:133
		qw422016.N().S(`,
		`)
//line routines.qtpl:134
		c.streamparamsArgs(qw422016, r)
//line routines.qtpl:134
		qw422016.N().S(`)

	return err
}

// `)
//line routines.qtpl:139
		qw422016.E().S(camelName)
//line routines.qtpl:139
		qw422016.N().S(`All run query with select DB function '`)
//line routines.qtpl:139
		qw422016.E().S(name)
//line routines.qtpl:139
		qw422016.N().S(`'
`)
//line routines.qtpl:140
		if r.Comment > "" {
//line routines.qtpl:140
			qw422016.N().S(`// DB comment: '`)
//line routines.qtpl:140
			qw422016.E().S(r.Comment)
//line routines.qtpl:140
			qw422016.N().S(`'
`)
//line routines.qtpl:141
		}
//line routines.qtpl:141
		qw422016.N().S(`// WARNING! It return ALL rows as Slice of &`)
//line routines.qtpl:142
		qw422016.E().S(camelName)
//line routines.qtpl:142
		qw422016.N().S(`sRowScanner
func (d *Database) `)
//line routines.qtpl:143
		qw422016.E().S(camelName)
//line routines.qtpl:143
		qw422016.N().S(`All(
	ctx context.Context,`)
//line routines.qtpl:144
		c.streamparamsTitle(qw422016, r)
//line routines.qtpl:144
		qw422016.N().S(`
) (res []`)
//line routines.qtpl:145
		qw422016.E().S(typeReturn)
//line routines.qtpl:145
		qw422016.N().S(`, err error) {
	buf := `)
//line routines.qtpl:146
		qw422016.E().S(typeReturn)
//line routines.qtpl:146
		qw422016.N().S(`{}
	err = d.Conn.SelectAndScanEach(ctx,
		func() error {
			res = append(res, buf)
			//	create new record
			buf = `)
//line routines.qtpl:151
		qw422016.E().S(typeReturn)
//line routines.qtpl:151
		qw422016.N().S(`{}

			return nil
		},
		&buf,
		`)
//line routines.qtpl:151
		qw422016.N().S("`")
//line routines.qtpl:156
		qw422016.E().S(sql)
//line routines.qtpl:156
		qw422016.N().S(``)
//line routines.qtpl:156
		qw422016.N().S("`")
//line routines.qtpl:156
		qw422016.N().S(`,
		`)
//line routines.qtpl:157
		c.streamparamsArgs(qw422016, r)
//line routines.qtpl:157
		qw422016.N().S(`)

	return
}
`)
//line routines.qtpl:161
	}
//line routines.qtpl:161
	qw422016.N().S(`
`)
//line routines.qtpl:162
}

//line routines.qtpl:162
func (c *Creator) WriteCreateFunctionInvoker(qq422016 qtio422016.Writer, r *psql.Routine, name, camelName, sql string) {
//line routines.qtpl:162
	qw422016 := qt422016.AcquireWriter(qq422016)
//line routines.qtpl:162
	c.StreamCreateFunctionInvoker(qw422016, r, name, camelName, sql)
//line routines.qtpl:162
	qt422016.ReleaseWriter(qw422016)
//line routines.qtpl:162
}

//line routines.qtpl:162
func (c *Creator) CreateFunctionInvoker(r *psql.Routine, name, camelName, sql string) string {
//line routines.qtpl:162
	qb422016 := qt422016.AcquireByteBuffer()
//line routines.qtpl:162
	c.WriteCreateFunctionInvoker(qb422016, r, name, camelName, sql)
//line routines.qtpl:162
	qs422016 := string(qb422016.B)
//line routines.qtpl:162
	qt422016.ReleaseByteBuffer(qb422016)
//line routines.qtpl:162
	return qs422016
//line routines.qtpl:162
}

// CreateFunctionInvoker
//

//line routines.qtpl:164
func (c *Creator) StreamCreateRowScanner(qw422016 *qt422016.Writer, r *psql.Routine, camelName string) {
//line routines.qtpl:164
	qw422016.N().S(`
// `)
//line routines.qtpl:166
	qw422016.E().S(camelName)
//line routines.qtpl:166
	qw422016.N().S(`RowScanner run query with select
type `)
//line routines.qtpl:167
	qw422016.E().S(camelName)
//line routines.qtpl:167
	qw422016.N().S(`RowScanner struct {
`)
//line routines.qtpl:169
	maxLen := 0
	for _, param := range r.Columns() {
		maxLen = max(maxLen, len(param.Name()))
	}

//line routines.qtpl:174
	for _, param := range r.Columns() {
//line routines.qtpl:176
		s := strcase.ToCamel(param.Name())
		typeCol, _ := c.chkTypes(param, s)
		if a, ok := strings.CutPrefix(typeCol, "[]"); param.BasicType() < 0 && ok {
			typeCol = "WrapArray[*" + a + "]"
		}

//line routines.qtpl:181
		qw422016.N().S(`	`)
//line routines.qtpl:182
		qw422016.E().S(fmt.Sprintf("%-[2]*[1]s ", s, maxLen))
//line routines.qtpl:182
		qw422016.E().S(typeCol)
//line routines.qtpl:182
		qw422016.E().S("\t\t")
//line routines.qtpl:182
		qw422016.N().S(``)
//line routines.qtpl:182
		qw422016.N().S("`")
//line routines.qtpl:182
		qw422016.N().S(`json:"`)
//line routines.qtpl:182
		qw422016.E().S(param.Name())
//line routines.qtpl:182
		qw422016.N().S(`"`)
//line routines.qtpl:182
		qw422016.N().S("`")
//line routines.qtpl:182
		qw422016.N().S(`
`)
//line routines.qtpl:183
	}
//line routines.qtpl:183
	qw422016.N().S(`}

// GetFields implement dbEngine.RowScanner interface
func (r *`)
//line routines.qtpl:187
	qw422016.E().S(camelName)
//line routines.qtpl:187
	qw422016.N().S(`RowScanner) GetFields(columns []dbEngine.Column) []any {
	v := make([]any, len(columns))
	for i, col := range columns {
		switch col.Name() {
`)
//line routines.qtpl:191
	for _, param := range r.Columns() {
//line routines.qtpl:191
		qw422016.N().S(`		case "`)
//line routines.qtpl:192
		qw422016.E().S(param.Name())
//line routines.qtpl:192
		qw422016.N().S(`":
			v[i] = &r.`)
//line routines.qtpl:193
		qw422016.N().S(strcase.ToCamel(param.Name()))
//line routines.qtpl:193
		qw422016.N().S(`
`)
//line routines.qtpl:194
	}
//line routines.qtpl:194
	qw422016.N().S(`		}
	}

	return v
}
`)
//line routines.qtpl:200
}

//line routines.qtpl:200
func (c *Creator) WriteCreateRowScanner(qq422016 qtio422016.Writer, r *psql.Routine, camelName string) {
//line routines.qtpl:200
	qw422016 := qt422016.AcquireWriter(qq422016)
//line routines.qtpl:200
	c.StreamCreateRowScanner(qw422016, r, camelName)
//line routines.qtpl:200
	qt422016.ReleaseWriter(qw422016)
//line routines.qtpl:200
}

//line routines.qtpl:200
func (c *Creator) CreateRowScanner(r *psql.Routine, camelName string) string {
//line routines.qtpl:200
	qb422016 := qt422016.AcquireByteBuffer()
//line routines.qtpl:200
	c.WriteCreateRowScanner(qb422016, r, camelName)
//line routines.qtpl:200
	qs422016 := string(qb422016.B)
//line routines.qtpl:200
	qt422016.ReleaseByteBuffer(qb422016)
//line routines.qtpl:200
	return qs422016
//line routines.qtpl:200
}

//line routines.qtpl:202
func (c *Creator) streamparamsTitle(qw422016 *qt422016.Writer, r *psql.Routine) {
//line routines.qtpl:204
	maxLen := 0
	for _, param := range r.Params() {
		maxLen = max(maxLen, len(param.Name()))
	}

//line routines.qtpl:208
	qw422016.N().S(`	`)
//line routines.qtpl:209
	for _, param := range r.Params() {
//line routines.qtpl:209
		qw422016.N().S(`
`)
//line routines.qtpl:211
		s := strcase.ToLowerCamel(param.Name())
		typeCol, _ := c.chkTypes(param, s)
		if param.Default() != nil && !strings.HasPrefix(typeCol, "[]") {
			typeCol = "*" + typeCol
		}

//line routines.qtpl:216
		qw422016.N().S(`	`)
//line routines.qtpl:217
		qw422016.E().S(fmt.Sprintf("%-[2]*[1]s ", s, maxLen))
//line routines.qtpl:217
		qw422016.E().S(typeCol)
//line routines.qtpl:217
		qw422016.N().S(`, // `)
//line routines.qtpl:217
		qw422016.E().S(param.Comment())
//line routines.qtpl:217
		qw422016.N().S(` pg type: `)
//line routines.qtpl:217
		qw422016.E().S(param.Type())
//line routines.qtpl:217
		if param.Default() != nil {
//line routines.qtpl:217
			qw422016.N().S(`, def: `)
//line routines.qtpl:217
			qw422016.E().V(param.Default())
//line routines.qtpl:217
		}
//line routines.qtpl:218
	}
//line routines.qtpl:219
}

//line routines.qtpl:219
func (c *Creator) writeparamsTitle(qq422016 qtio422016.Writer, r *psql.Routine) {
//line routines.qtpl:219
	qw422016 := qt422016.AcquireWriter(qq422016)
//line routines.qtpl:219
	c.streamparamsTitle(qw422016, r)
//line routines.qtpl:219
	qt422016.ReleaseWriter(qw422016)
//line routines.qtpl:219
}

//line routines.qtpl:219
func (c *Creator) paramsTitle(r *psql.Routine) string {
//line routines.qtpl:219
	qb422016 := qt422016.AcquireByteBuffer()
//line routines.qtpl:219
	c.writeparamsTitle(qb422016, r)
//line routines.qtpl:219
	qs422016 := string(qb422016.B)
//line routines.qtpl:219
	qt422016.ReleaseByteBuffer(qb422016)
//line routines.qtpl:219
	return qs422016
//line routines.qtpl:219
}

//line routines.qtpl:221
func (c *Creator) streamparamsArgs(qw422016 *qt422016.Writer, r *psql.Routine) {
//line routines.qtpl:222
	for _, param := range r.Params() {
//line routines.qtpl:222
		qw422016.N().S(`	`)
//line routines.qtpl:223
		qw422016.E().S(strcase.ToLowerCamel(param.Name()))
//line routines.qtpl:223
		qw422016.N().S(`,
`)
//line routines.qtpl:224
	}
//line routines.qtpl:225
}

//line routines.qtpl:225
func (c *Creator) writeparamsArgs(qq422016 qtio422016.Writer, r *psql.Routine) {
//line routines.qtpl:225
	qw422016 := qt422016.AcquireWriter(qq422016)
//line routines.qtpl:225
	c.streamparamsArgs(qw422016, r)
//line routines.qtpl:225
	qt422016.ReleaseWriter(qw422016)
//line routines.qtpl:225
}

//line routines.qtpl:225
func (c *Creator) paramsArgs(r *psql.Routine) string {
//line routines.qtpl:225
	qb422016 := qt422016.AcquireByteBuffer()
//line routines.qtpl:225
	c.writeparamsArgs(qb422016, r)
//line routines.qtpl:225
	qs422016 := string(qb422016.B)
//line routines.qtpl:225
	qt422016.ReleaseByteBuffer(qb422016)
//line routines.qtpl:225
	return qs422016
//line routines.qtpl:225
}