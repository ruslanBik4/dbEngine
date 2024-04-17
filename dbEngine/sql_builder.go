// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"go/types"
	"strings"
	"time"

	"github.com/jackc/pgtype"
	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"
)

// SQLBuilder implement sql native constructor
type SQLBuilder struct {
	Args          []any
	columns       []string
	filter        []string
	posFilter     int
	Table         Table
	onConflict    string
	OrderBy       []string
	Offset, Limit int
}

// NewSQLBuilder create SQLBuilder for table
func NewSQLBuilder(t Table, Options ...BuildSqlOptions) (*SQLBuilder, error) {
	b := &SQLBuilder{Table: t}
	for _, setOption := range Options {
		err := setOption(b)
		if err != nil {
			return nil, errors.Wrap(err, "setOption")
		}
	}

	return b, nil
}

// InsertSql construct insert sql
func (b SQLBuilder) InsertSql() (string, error) {
	if len(b.columns) != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.columns, b.Args)
	}

	return b.insertSql(), nil
}

func (b SQLBuilder) insertSql() string {
	return fmt.Sprintf(`INSERT INTO %s(%s) VALUES (%s) %s`, b.Table.Name(), b.Select(), b.values(), b.OnConflict())
}

// UpdateSql construct update sql
func (b SQLBuilder) UpdateSql() (string, error) {
	if len(b.columns)+len(b.filter) != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.columns, b.Args)
	}

	s, err := b.Set()
	if err != nil {
		return "", err
	}
	return "UPDATE " + b.Table.Name() + s + b.Where(), nil
}

func (b SQLBuilder) upsertSql() (string, error) {
	s, err := b.SetUpsert()
	if err != nil {
		return "", err
	}
	return " DO UPDATE" + s, nil
}

// UpsertSql perform sql-script for insert with update according onConflict
func (b SQLBuilder) UpsertSql() (string, error) {
	if len(b.columns) != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.columns, b.Args)
	}

	if len(b.filter) == 0 {
		b.filter = make([]string, 0)

		for _, name := range b.columns {
			if col := b.Table.FindColumn(name); col == (Column)(nil) {
				return "", NewErrNotFoundColumn(b.Table.Name(), name)
			} else if col.Primary() {
				b.filter = append(b.filter, name)
			}
		}
		if len(b.filter) == 0 {
			for _, ind := range b.Table.Indexes() {
				// we get first unique index for onConflict
				if ind.Unique {
					if strings.TrimSpace(ind.Expr) > "" {
						b.filter = append(b.filter, ind.Expr)
					} else {
						b.filter = append(b.filter, strings.Join(ind.Columns, ","))
					}
					break
				}
			}
		}
	}

	if b.onConflict == "" {
		if len(b.filter) == 0 {
			return b.insertSql(), nil
		}

		onConflict := strings.Join(b.filter, ",")
		b.onConflict = onConflict
	}

	s := b.insertSql()
	b.posFilter = 0

	u, err := b.upsertSql()
	if err != nil {
		return "", err
	}

	return s + u, nil
}

// DeleteSql construct delete sql
func (b SQLBuilder) DeleteSql() (string, error) {
	// todo check routine
	if len(b.filter)+strings.Count(b.Table.Name(), "$") != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.filter, b.Args)
	}

	sql := "DELETE FROM " + b.Table.Name() + b.Where()

	return sql, nil
}

// SelectSql construct select sql
func (b *SQLBuilder) SelectSql() (string, error) {
	// todo check routine
	lenFilter := len(b.filter) + strings.Count(b.Table.Name(), "$")
	if lenFilter != len(b.Args) {
		// rm from counter filter without params
		for _, name := range b.filter {
			yes, hasTempl := b.isComplexCondition(name)
			if yes && !hasTempl {
				lenFilter--
			}
		}
		if lenFilter != len(b.Args) {
			return "", NewErrWrongArgsLen(b.Table.Name(), b.filter, b.Args)
		}
	}

	sql := "SELECT " + b.Select() + " FROM " + b.Table.Name() + b.Where()

	if len(b.OrderBy) > 0 {
		// todo add column checking
		sql += " order by " + strings.Join(b.OrderBy, ",")
	}

	if b.Offset > 0 {
		sql += fmt.Sprintf(" offset %d ", b.Offset)
	}

	if b.Limit > 0 {
		sql += fmt.Sprintf(" fetch first %d rows only ", b.Limit)
	}

	return sql, nil
}

// SelectColumns construct select clause for sql
func (b *SQLBuilder) SelectColumns() []Column {
	if b.Table == nil {
		return nil
	}

	if len(b.columns) == 0 {
		selectColumns := make([]Column, len(b.Table.Columns()))
		for i, col := range b.Table.Columns() {
			selectColumns[i] = col
		}

		return selectColumns
	}

	selectColumns := make([]Column, len(b.columns))
	for i, name := range b.columns {
		col, ok := CheckColumn(name, b.Table)
		if ok {
			selectColumns[i] = col
		} else {
			logs.ErrorLog(NewErrNotFoundColumn(b.Table.Name(), name))
		}
	}

	return selectColumns
}

// CheckColumn check ddl for consists any columns of table
func CheckColumn(ddl string, table Table) (col Column, trueColumn bool) {
	fullStr := regColumns.FindAllStringSubmatch(ddl, -1)
	if len(fullStr) > 0 {
		for _, list := range fullStr {
			if len(list) > 0 {
				col, trueColumn = checkParams(strings.Split(list[len(list)-1], ","), table)
				if trueColumn {
					return
				}
			}
		}

		return nil, false
	}

	name := shrinkColName(ddl)
	col = table.FindColumn(name)
	if !strings.Contains(name, " as ") && col == nil {
		return nil, false
	}

	return col, true
}

func checkParams(columns []string, table Table) (Column, bool) {
	for _, colName := range columns {
		name := strings.TrimSpace(colName)
		if strings.HasPrefix(name, "'") {
			continue
		}

		col := table.FindColumn(shrinkColName(name))
		if col != nil {
			return col, true
		}
	}

	return nil, false
}

func shrinkColName(name string) string {
	return strings.TrimSpace(strings.Split(name, "::")[0])
}

// Select return select clause of sql query
func (b *SQLBuilder) Select() string {
	if len(b.columns) == 0 {
		if b.Table != nil && len(b.Table.Columns()) > 0 {
			b.fillColumns()
		} else {
			// todo - chk for insert request
			return "*"
		}
	}

	return strings.Join(b.columns, ",")
}

func (b *SQLBuilder) fillColumns() {
	b.columns = make([]string, len(b.Table.Columns()))
	for i, col := range b.Table.Columns() {
		b.columns[i] = col.Name()
	}
}

// Set return SET clause of sql update query
func (b *SQLBuilder) Set() (string, error) {
	s, comma := " SET ", ""
	if len(b.columns) == 0 {
		return "", errors.Wrap(NewErrWrongType("columns list", "update", "nil"),
			"Set")
	}

	for _, name := range b.columns {
		b.posFilter++
		s += fmt.Sprintf(comma+" %s=$%d", name, b.posFilter)
		comma = ","
	}

	return s, nil
}

// SetUpsert return SET clause of sql query for handling error on insert
func (b *SQLBuilder) SetUpsert() (string, error) {
	s, comma := " SET", " "
	if len(b.columns) == 0 {
		if b.Table != nil && len(b.Table.Columns()) > 0 {
			b.fillColumns()
		} else {
			return "", errors.Wrap(NewErrWrongType("columns list", "table", "nil"),
				"SetUpsert")
		}
	}

loop_columns:
	for _, name := range b.columns {
		if c := b.onConflict; c == name || b.chkFilter(c, name) {
			continue loop_columns
		}
		for _, f := range b.filter {
			if f == name || b.chkFilter(f, name) {
				continue loop_columns
			}
		}
		s += fmt.Sprintf(comma+"%s=EXCLUDED.%[1]s", name)
		comma = ", "
	}

	return s, nil
}

func (b *SQLBuilder) chkFilter(filter, name string) bool {
	if parts := regColumn.FindAllString(filter, -1); len(parts) > 0 {
		for _, part := range parts {
			if name == part {
				return true
			}
		}
	}

	return false
}

// Where return where-clause of sql query
func (b *SQLBuilder) Where() string {

	where, comma := "", " "
	for _, name := range b.filter {
		isComplexCondition, hasTempl := b.isComplexCondition(name)
		// 'is null, 'is not null', 'CASE WHEN ..END' write as is when they not consist of '%s'
		if isComplexCondition && !hasTempl {
			where += comma + name
			comma = " AND "
			continue
		}

		b.posFilter++

		where += comma + b.writeCondition(name, hasTempl)
		comma = " AND "
	}

	if where > "" {
		return " WHERE " + where
	}

	return ""
}

func (b *SQLBuilder) isComplexCondition(name string) (bool, bool) {
	isComplexCondition := strings.Index(name, " ") > 0
	return isComplexCondition, isComplexCondition && strings.Contains(name, "%s")
}

func (b *SQLBuilder) writeCondition(name string, hasTempl bool) string {
	switch pre := name[0]; {
	case isOperator(pre):
		preStr := string(pre)
		switch {
		case isOperatorPre(name[1]):
			preStr += string(name[1])
			name = name[2:]
		default:
			name = name[1:]
		}

		switch pre {
		case '$':
			return fmt.Sprintf("%s ~ concat('.*', $%d, '$')", name, b.posFilter)
		case '^':
			return fmt.Sprintf("%s ~ concat('^.*', $%d)", name, b.posFilter)
		default:
			return fmt.Sprintf("%s %s $%d", name, preStr, b.posFilter)
		}

	default:
		return b.chkSpecialParams(name, hasTempl)
	}
}

// chkSpecialParams get condition for WHERE include complex params as:
// 'is null, 'is not null'
// in (select ... from ... where field = {param})
func (b *SQLBuilder) chkSpecialParams(name string, hasTempl bool) string {

	cond := "$%[1]d"
	switch arg := b.Args[b.posFilter-1].(type) {
	case []int, []int8, []int16, []int32, []int64, []float32, []float64, []string, types.Slice, []time.Time, []*time.Time,
		pgtype.ArrayType, pgtype.Int2Array, pgtype.Int4Array, pgtype.Int8Array, pgtype.DateArray,
		pgtype.TimestampArray, pgtype.TimestamptzArray,
		pgtype.Float4Array, pgtype.Float8Array, pgtype.NumericArray, pgtype.BPCharArray, pgtype.TextArray:
		// todo: chk column type
		if table := b.Table; table != nil {
			if col, ok := table.FindColumn(name).(interface{ IsArray() bool }); ok && col.IsArray() {
				return fmt.Sprintf("%s@>$%d", name, b.posFilter)
			}
		}
		cond = "ANY($%[1]d)"
	case nil:
		cond = "is null"
	case string:
		if strings.Contains(arg, "is ") {
			cond = arg
		}
	}

	if strings.Contains(cond, "is ") {
		// rm agr from slice
		b.posFilter--
		b.Args = rmElem(b.Args, b.posFilter)
		// condition without psql params
		return name + " " + cond
	}

	if hasTempl {
		cond = fmt.Sprintf(name, cond)
	} else {
		cond = name + "=" + cond
	}

	return fmt.Sprintf(cond, b.posFilter)
}

func rmElem(a []any, i int) []any {
	if i < len(a)-1 {
		copy(a[i:], a[i+1:])
	}
	a[len(a)-1] = 0 // or the zero value of T
	return a[:len(a)-1]
}

// OnConflict return sql-text for handling error on insert
func (b *SQLBuilder) OnConflict() string {
	if b.onConflict == "" {
		return ""
	}

	if b.onConflict == "DO NOTHING" {
		return "ON CONFLICT " + b.onConflict
	}

	return "ON CONFLICT (" + b.onConflict + ")"
}

func (b *SQLBuilder) values() string {
	s, comma := "", ""
	for range b.Args {
		b.posFilter++
		s += fmt.Sprintf("%s$%d", comma, b.posFilter)
		comma = ","
	}

	return s
}

func isOperatorPre(s uint8) bool {
	switch s {
	case '=', '>', '<', '&', '|', '#':
		return true
	default:
		return false
	}

}

func isOperator(s uint8) bool {
	switch s {
	case '>', '<', '$', '~', '^', '@', '&', '+', '-', '*', '#':
		return true
	default:
		return false
	}
}

// BuildSqlOptions set addition property on SQLbuilder
type BuildSqlOptions func(b *SQLBuilder) error

// ColumnsForSelect set columns for SQLBuilder
func ColumnsForSelect(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.columns = columns

		return nil
	}
}

// Columns set inserted columns for SQLBuilder
func Columns(columns ...string) BuildSqlOptions {
	return ColumnsForSelect(columns...)
}

// Args set slice of arguments sql request
func Args(args ...any) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Args = args

		return nil
	}
}

// ArgsForSelect set slice of arguments sql request
func ArgsForSelect(args ...any) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Args = args

		return nil
	}
}

// Values set values sql insert
func Values(values ...any) BuildSqlOptions {
	return ArgsForSelect(values...)
}

// WhereForSelect set columns for WHERE clause
// may consisted first symbol as conditions rule:
//
//	'>', '<', '$', '~', '^', '@', '&', '+', '-', '*'
//
// that will replace equals condition, instead:
//
//	field_name = $1
//	write:
//	field_name > $1, field_name < $1, etc
func WhereForSelect(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.filter = make([]string, len(columns))
		if b.Table != nil {
			for _, name := range columns {
				if isOperator(name[0]) {
					switch {
					case isOperatorPre(name[1]):
						name = name[2:]
					default:
						name = name[1:]
					}
				} else if tokens := strings.Split(name, " "); len(tokens) > 1 {
					if tokens[1] == "in" || tokens[1] == "is" {
						name = tokens[0]
					} else if len(tokens) > 2 && isOperatorPre(tokens[1][0]) {
						name = tokens[0]
						secName := tokens[2]
						if regFieldName.MatchString(secName) && b.Table.FindColumn(secName) == nil {
							return NewErrNotFoundColumn(b.Table.Name(), secName)
						}
					}
				}
				if tokens := strings.Split(name, "::"); len(tokens) > 1 {
					name = tokens[0]
				}

				if b.Table.FindColumn(name) == nil {
					return NewErrNotFoundColumn(b.Table.Name(), name)
				}

			}
		}

		b.filter = columns

		return nil
	}
}

// OrderBy parameter for sql query
func OrderBy(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.OrderBy = columns

		return nil
	}
}

// InsertOnConflict parameter for sql query
func InsertOnConflict(onConflict string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.onConflict = onConflict

		return nil
	}
}

// InsertOnConflictDoNothing parameter for sql query
func InsertOnConflictDoNothing() BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.onConflict = "DO NOTHING"

		return nil
	}
}

// FetchOnlyRows parameter for sql query
func FetchOnlyRows(i int) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Limit = i

		return nil
	}
}

// Offset parameter for sql query
func Offset(i int) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Offset = i

		return nil
	}
}
