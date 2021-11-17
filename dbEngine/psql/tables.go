// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"sync"

	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// FieldsTable for columns parameters in form
type Table struct {
	conn       *Conn
	name, Type string
	ID         int
	comment    string
	columns    []*Column
	indexes    dbEngine.Indexes
	PK         string
	buf        *Column
	lock       sync.RWMutex
}

func (t *Table) Comment() string {
	return t.comment
}

func (t *Table) GetFields(columns []dbEngine.Column) []interface{} {
	if len(columns) == 0 {
		return []interface{}{&t.name, &t.Type, &t.comment}
	}

	v := make([]interface{}, len(columns))
	for i, col := range columns {
		switch name := col.Name(); name {
		case "table_name":
			v[i] = &t.name
		case "table_type":
			v[i] = &t.Type
		case "comment":
			v[i] = &t.comment
		case "oid":
			v[i] = &t.ID
		default:
			if t.buf == nil {
				t.buf = NewColumnForTableBuf(t)
			}
			v[i] = t.buf.RefColValue(name)
		}
	}

	return v

}

func (t *Table) Columns() []dbEngine.Column {
	res := make([]dbEngine.Column, len(t.columns))
	for i, col := range t.columns {
		res[i] = col
	}

	return res
}

func (t *Table) Delete(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return 0, errors.Wrap(err, "setOption")
	}

	sql, err := b.DeleteSql()
	if err != nil {
		return 0, err
	}

	comTag, err := t.conn.Exec(ctx, sql, b.Args...)
	if err != nil {
		return -1, errors.Wrap(err, sql)
	}

	return comTag.RowsAffected(), nil
}

// Insert return new ID or rowsAffected if autoinc field not there
func (t *Table) Insert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return 0, errors.Wrap(err, "setOption")
	}

	sql, err := b.InsertSql()
	if err != nil {
		return 0, err
	}

	return t.doInsertReturning(ctx, sql, b.Args...)
}

func (t *Table) Update(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return 0, errors.Wrap(err, "setOption")
	}

	sql, err := b.UpdateSql()
	if err != nil {
		return 0, err
	}

	comTag, err := t.conn.Exec(ctx, sql, b.Args...)
	if err != nil {
		return -1, errors.Wrap(err, sql)
	}

	return comTag.RowsAffected(), nil
}

// Upsert preforms INSERT sql or UPDATE if record with primary keys exists
func (t *Table) Upsert(ctx context.Context, Options ...dbEngine.BuildSqlOptions) (int64, error) {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return 0, errors.Wrap(err, "setOption")
	}

	sql, err := b.UpsertSql()
	if err != nil {
		return 0, err
	}

	return t.doInsertReturning(ctx, sql, b.Args...)
}

func (t *Table) doInsertReturning(ctx context.Context, sql string, args ...interface{}) (int64, error) {
	for _, col := range t.columns {
		if col.Primary() && col.autoInc {
			sql += " RETURNING " + col.Name()
			id := int64(-1)
			err := t.conn.SelectOneAndScan(ctx, &id, sql, args...)
			if err == pgx.ErrNoRows {
				err = nil
			}
			return id, errors.Wrap(err, sql)
		}
	}

	comTag, err := t.conn.Exec(ctx, sql, args...)
	if err != nil {
		return -1, errors.Wrap(err, sql)
	}

	return comTag.RowsAffected(), nil
}

func (t *Table) Name() string {
	return t.name
}

func (t *Table) Select(ctx context.Context, Options ...dbEngine.BuildSqlOptions) error {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return errors.Wrap(err, "setOption")
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	_, err = t.conn.Query(ctx, sql, b.Args...)

	return err
}

func (t *Table) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return errors.Wrap(err, "setOption")
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	return t.conn.SelectOneAndScan(ctx, row, sql, b.Args...)
}

func (t *Table) SelectAndScanEach(ctx context.Context, each func() error, row dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return errors.Wrap(err, "setOption")
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	return t.conn.SelectAndScanEach(ctx, each, row, sql, b.Args...)
}

func (t *Table) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	b, err := dbEngine.NewSQLBuilder(t, Options...)
	if err != nil {
		return errors.Wrap(err, "setOption")
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	return t.conn.selectAndRunEach(
		ctx,
		func(values []interface{}, columns []dbEngine.Column) error {
			if each != nil {
				return each(values, b.SelectColumns())
			}

			return nil
		},
		sql,
		b.Args...)
}

func (t *Table) FindColumn(name string) dbEngine.Column {
	c := t.findColumn(name)
	if c == nil {
		return nil
	}

	return c
}

func (t *Table) findColumn(name string) *Column {
	for _, col := range t.columns {
		if col.Name() == name {
			return col
		}
	}

	return nil
}

// GetColumns получение значений полей для форматирования данных
// получение значений полей для таблицы
func (t *Table) GetColumns(ctx context.Context) error {

	err := t.conn.SelectAndScanEach(ctx,
		t.readColumnRow,
		t,
		sqlGetTablesColumns+" ORDER BY C.ordinal_position",
		t.name)
	if err != nil {
		return err
	}

	return nil
}

// GetIndexes collect index of table
func (t *Table) GetIndexes(ctx context.Context) error {

	return errors.Wrap(
		t.conn.SelectAndScanEach(ctx,
			func() error {
				i := t.indexes.LastIndex()
				if len(i.Columns) == 0 && i.Expr > "" {
					col, ok := dbEngine.CheckColumn(i.Expr, t)
					if ok {
						// todo refactoring
						i.Columns = []string{col.Name()}
					}
				}

				return nil
			},
			&t.indexes, sqlGetIndexes, t.Name()), t.Name())
}

// FindIndex get index according to name
func (t *Table) FindIndex(name string) *dbEngine.Index {
	for _, ind := range t.indexes {
		if name == ind.Name {
			return ind
		}
	}

	return nil
}

// Indexes get indexex according to table
func (t *Table) Indexes() dbEngine.Indexes {
	return t.indexes
}

func (t *Table) ReReadColumn(name string) dbEngine.Column {
	t.lock.RLock()
	defer t.lock.RUnlock()

	column := t.findColumn(name)
	if column == nil {
		column = NewColumnPone(
			name,
			"new column",
			0,
		)

		column.Table = t
		t.columns = append(t.columns, column)
	}

	// todo implement
	err := t.conn.SelectAndScanEach(
		context.TODO(),
		func() error {
			return nil
		},
		column, sqlGetColumnAttr, t.name, column.Name(),
	)
	if err != nil {
		logs.ErrorLog(err, sqlGetColumnAttr)
		return nil
	}

	return column
}

func (t *Table) readColumnRow() error {

	for name, c := range t.buf.Constraints {
		if c == nil {
			t.PK = name
			t.buf.PrimaryKey = true
		}
	}

	t.buf.SetDefault(t.buf.colDefault)

	t.columns = append(t.columns, t.buf.Copy())

	t.buf = nil

	return nil
}
