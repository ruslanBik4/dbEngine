// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"sync"

	"github.com/jackc/pgproto3/v2"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type PgxRoutineParams struct {
	Column
	Fnc      *Routine `json:"-"`
	Position int32
}

func (p *PgxRoutineParams) Type() string {
	if p.DataType == "ARRAY" {
		return p.UdtName[1:] + "[]"
	}

	return p.UdtName
}

type Routine struct {
	conn     *Conn
	name     string
	ID       int
	Comment  string
	columns  []*PgxRoutineParams
	params   []*PgxRoutineParams
	Overlay  *Routine
	Type     string
	lock     sync.RWMutex
	sName    string
	DataType string
	UdtName  string
}

func (r *Routine) Name() string {
	return r.name
}

func (r *Routine) Columns() []dbEngine.Column {
	res := make([]dbEngine.Column, len(r.columns))
	for i, col := range r.columns {
		res[i] = col
	}

	return res
}

func (r *Routine) Select(ctx context.Context, args ...interface{}) error {
	panic("implement me")
}

func (r *Routine) Call(context.Context) error {
	panic("implement me")
}

func (r *Routine) Params() []dbEngine.Column {
	res := make([]dbEngine.Column, len(r.params))
	for i, col := range r.params {
		res[i] = col
	}

	return res
}

// GetParams получение значений полей для форматирования данных
// получение значений полей для таблицы
func (r *Routine) GetParams(ctx context.Context) error {

	return r.conn.selectAndRunEach(ctx, func(values []interface{}, columns []pgproto3.FieldDescription) error {

		if values[0] == nil {
			return nil
		}

		row := &PgxRoutineParams{
			Fnc:      r,
			Position: values[6].(int32),
		}

		row.name = values[0].(string)
		row.DataType = values[1].(string)
		row.UdtName = values[2].(string)
		row.CharacterSetName = values[3].(string)
		row.characterMaximumLength = int(values[4].(int32))
		row.SetDefault(values[5])

		if values[7].(string) == "IN" {
			r.params = append(r.params, row)
		} else {
			r.columns = append(r.columns, row)
		}

		return nil
	}, sqlGetFuncParams+" ORDER BY ordinal_position", r.sName)
}

func (r *Routine) SelectAndScanEach(ctx context.Context, each func() error, row dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {

	b := &dbEngine.SQLBuilder{Table: r.newTableForSQL()}
	for _, setOption := range Options {
		err := setOption(b)
		if err != nil {
			return errors.Wrap(err, "setOption")
		}
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	return r.conn.SelectAndScanEach(ctx, each, row, sql, b.Args...)
}

func (r *Routine) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	b := &dbEngine.SQLBuilder{Table: r.newTableForSQL()}

	for _, setOption := range Options {
		err := setOption(b)
		if err != nil {
			return errors.Wrap(err, "setOption")
		}
	}

	sql, err := b.SelectSql()
	if err != nil {
		return err
	}

	return r.conn.selectAndRunEach(
		ctx,
		func(values []interface{}, columns []pgproto3.FieldDescription) error {
			if each != nil {
				return each(values, b.SelectColumns())
			}

			return nil
		},
		sql,
		b.Args...)
}

func (r *Routine) newTableForSQL() *Table {
	name := r.name + "("
	for i, p := range r.params {
		if i > 0 {
			name += ","
		}

		name += fmt.Sprintf("$%d :: %s", i+1, p.Type())
	}

	table := &Table{name: name + ")"}
	table.columns = make([]*Column, len(r.columns))
	for i, col := range r.columns {
		table.columns[i] = &(col.Column)
	}

	return table

}
