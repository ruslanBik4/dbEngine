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

// Table implement dbEngine interface Table for PostgreSQL
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

// DoCopy run CopyFrom PSQL use src interface
func (t *Table) DoCopy(ctx context.Context, src pgx.CopyFromSource, columns ...string) (int64, error) {
	if len(columns) == 0 {
		columns = make([]string, len(t.columns))
		for i, col := range t.columns {
			columns[i] = col.name
		}
	}

	return t.conn.Pool.CopyFrom(ctx, pgx.Identifier{t.name}, columns, src)
}

// Comment of Table
func (t *Table) Comment() string {
	return t.comment
}

// GetFields implement RowScanner interface
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

// Columns of Table
func (t *Table) Columns() []dbEngine.Column {
	res := make([]dbEngine.Column, len(t.columns))
	for i, col := range t.columns {
		res[i] = col
	}

	return res
}

// Delete row of table according to Options
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

// Insert new row & return new ID or rowsAffected if there not autoinc field
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

// Update table according to Options
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

// Name of Table
func (t *Table) Name() string {
	return t.name
}

// Select run sql with Options (deprecated)
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

// SelectOneAndScan run sqlof table  with Options & return rows into rowValues
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

// SelectAndScanEach run sql of table with Options & return every row into rowValues & run each
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

// SelectAndRunEach run sql of table with Options & performs each every row of query results
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

// FindColumn return column 'name' on Table or nil
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
		sqlGetTablesColumns,
		t.name)
	if err != nil {
		return err
	}

	if len(t.columns) == 0 {
		return errors.New("no columns for table found")
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

// ReReadColumn renew properties of column 'name'
func (t *Table) ReReadColumn(ctx context.Context, name string) dbEngine.Column {
	t.lock.RLock()
	defer t.lock.RUnlock()

	column := t.findColumn(name)
	if column == nil {
		column = NewColumnPone(
			name,
			"new column",
			0,
		)

		column.table = t
		t.columns = append(t.columns, column)
	}

	// todo implement
	err := t.conn.SelectAndScanEach(
		ctx,
		nil,
		column,
		sqlGetColumnAttr,
		t.name,
		column.Name(),
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
