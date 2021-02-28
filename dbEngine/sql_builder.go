// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
)

type SQLBuilder struct {
	Args          []interface{}
	columns       []string
	filter        []string
	posFilter     int
	Table         Table
	onConflict    string
	OrderBy       []string
	Offset, Limit int
}

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

func (b SQLBuilder) InsertSql() (string, error) {
	if len(b.columns) != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.columns, b.Args)
	}

	return b.insertSql(), nil
}

func (b SQLBuilder) insertSql() string {
	return "INSERT INTO " + b.Table.Name() + "(" + b.Select() + ") VALUES (" + b.values() + ")" + b.OnConflict()
}

func (b SQLBuilder) UpdateSql() (string, error) {
	if len(b.columns)+len(b.filter) != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.columns, b.Args)
	}

	return b.updateSql()
}

func (b SQLBuilder) updateSql() (string, error) {
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
				// we get firts unique index for onConflict
				if ind.Unique {
					for _, name := range ind.Columns {
						b.filter = append(b.filter, name)
					}
					break
				}
			}
		}
	}

	b.onConflict = strings.Join(b.filter, ",")

	s := b.insertSql()
	b.posFilter = 0

	u, err := b.upsertSql()
	if err != nil {
		return "", err
	}

	return s + u, nil
}

func (b SQLBuilder) SelectSql() (string, error) {
	// todo check routine
	if len(b.filter)+strings.Count(b.Table.Name(), "$") != len(b.Args) {
		return "", NewErrWrongArgsLen(b.Table.Name(), b.filter, b.Args)
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

func (b *SQLBuilder) Set() (string, error) {
	s, comma := " SET ", ""
	if len(b.columns) == 0 {
		if b.Table != nil && len(b.Table.Columns()) > 0 {
			b.fillColumns()
		} else {
			// todo add return error
			return "", errors.Wrap(NewErrWrongType("columns list", "table", "nil"),
				"Set")
		}
	}

	for _, name := range b.columns {
		b.posFilter++
		s += fmt.Sprintf(comma+" %s=$%d", name, b.posFilter)
		comma = ","
	}

	return s, nil
}

func (b *SQLBuilder) SetUpsert() (string, error) {
	s, comma := " SET ", ""
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
		for _, col := range b.filter {
			if col == name {
				continue loop_columns
			}
		}
		s += fmt.Sprintf(comma+" %s=EXCLUDED.%[1]s", name)
		comma = ","
	}

	return s, nil
}

func (b *SQLBuilder) Where() string {

	where, comma := "", " "
	for _, name := range b.filter {
		b.posFilter++

		switch pre := name[0]; pre {
		case '>', '<', '$', '~', '^':
			preStr := ""
			if name[1] == '=' {
				preStr = "="
				name = name[2:]
			} else {
				name = name[1:]
			}

			switch pre {
			case '$':
				where += fmt.Sprintf(comma+"%s ~ concat('.*', $%d, '$')", name, b.posFilter)
			case '^':
				where += fmt.Sprintf(comma+"%s ~ concat('^.*', $%d)", name, b.posFilter)
			default:
				where += fmt.Sprintf(comma+"%s %s $%d", name, string(pre)+preStr, b.posFilter)
			}
		default:
			cond := ""
			switch b.Args[b.posFilter-1].(type) {
			case []int32, []int64, []string:
				// todo: chk column type
				cond = "ANY($%d)"
			default:
				cond = "$%d"
			}

			if strings.Contains(name, " in (") {
				cond = fmt.Sprintf(name, cond)
			} else {
				cond = name + "=" + cond
			}

			where += fmt.Sprintf(comma+cond, b.posFilter)
		}
		comma = " AND "
	}

	if where > "" {
		return " WHERE " + where
	}

	return ""
}

func (b *SQLBuilder) OnConflict() string {
	if b.onConflict == "" {
		return ""
	}

	if b.onConflict == "DO NOTHING" {
		return " ON CONFLICT " + b.onConflict
	}

	return " ON CONFLICT (" + b.onConflict + ")"
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

type BuildSqlOptions func(b *SQLBuilder) error

func ColumnsForSelect(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.columns = columns

		return nil
	}
}

func WhereForSelect(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.filter = make([]string, len(columns))
		if b.Table != nil {
			for _, name := range columns {

				switch pre := name[0]; pre {
				case '>', '<', '$', '~', '^':
					if name[1] == '=' {
						pre += '='
						name = name[2:]
					} else {
						name = name[1:]
					}
				default:
					if strings.Contains(name, " in (") {
						name = strings.Split(name, " ")[0]
					}
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

func OrderBy(columns ...string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.OrderBy = columns

		return nil
	}
}

func ArgsForSelect(args ...interface{}) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Args = args

		return nil
	}
}

func InsertOnConflict(onConflict string) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.onConflict = onConflict

		return nil
	}
}

func InsertOnConflictDoNothing() BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.onConflict = "DO NOTHING"

		return nil
	}
}

func FetchOnlyRows(i int) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Limit = i

		return nil
	}
}

func Offset(i int) BuildSqlOptions {
	return func(b *SQLBuilder) error {

		b.Offset = i

		return nil
	}
}
