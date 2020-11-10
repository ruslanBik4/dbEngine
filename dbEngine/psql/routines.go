// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"fmt"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
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

	return r.conn.selectAndRunEach(ctx,
		func(values []interface{}, columns []dbEngine.Column) error {

			if values[0] == nil {
				return nil
			}

			row := &PgxRoutineParams{
				Fnc: r,
			}

			parameterMode := ""
			for i, col := range columns {
				switch col.Name() {
				case "parameter_name":
					row.name = values[i].(string)
				case "data_type":
					row.DataType = values[i].(string)
				case "udt_name":
					row.UdtName = values[i].(string)
				case "character_set_name":
					row.CharacterSetName = values[i].(string)
				case "character_maximum_length":
					row.characterMaximumLength = int(values[i].(int32))
				case "parameter_default":
					row.SetDefault(values[i])
				case "ordinal_position":
					row.Position = values[i].(int32)
				case "parameter_mode":
					parameterMode = values[i].(string)
				default:
					logs.ErrorLog(dbEngine.ErrNotFoundColumn{
						Table:  "sqlGetFuncParams",
						Column: col.Name(),
					})
				}
			}

			if parameterMode == "IN" {
				r.params = append(r.params, row)
			} else {
				r.columns = append(r.columns, row)
			}

			return nil
		}, sqlGetFuncParams+" ORDER BY ordinal_position", r.sName)
}

func (r *Routine) SelectAndScanEach(ctx context.Context, each func() error, row dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {

	sql, args, err := r.BuildSql(Options)
	if err != nil {
		return err
	}

	return r.conn.SelectAndScanEach(ctx, each, row, sql, args...)
}

func (r *Routine) BuildSql(Options []dbEngine.BuildSqlOptions) (string, []interface{}, error) {
	b := &dbEngine.SQLBuilder{Table: r.newTableForSQL()}
	for _, setOption := range Options {
		err := setOption(b)
		if err != nil {
			return "", nil, errors.Wrap(err, "setOption")
		}
	}

	if len(r.params) > len(b.Args) {
		for _, param := range r.params[len(b.Args):] {
			if param.Default() == nil {
				return "", nil, dbEngine.NewErrWrongArgsLen(b.Table.Name(),
					strings.Split(b.Table.Name(), ","),
					b.Args)
			}
		}
		name := b.Table.Name()
		parts := strings.Split(name, ",")
		name = strings.Join(parts[:len(b.Args)], ",") + ")"
		(b.Table).(*Table).name = name
	}

	sql, err := b.SelectSql()
	if err != nil {
		return "", nil, err
	}

	return sql, b.Args, nil
}

func (r *Routine) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	sql, args, err := r.BuildSql(Options)
	if err != nil {
		return err
	}

	return r.conn.selectAndRunEach(
		ctx,
		each,
		sql,
		args...)
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
