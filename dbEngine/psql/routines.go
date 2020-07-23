// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"go/types"
	"strconv"
	"sync"

	"github.com/jackc/pgproto3/v2"
	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type PgxRoutineParams struct {
	Fnc                    *Routine `json:"-"`
	name                   string
	DataType               string
	DataName               string
	CharacterSetName       string
	comment                string
	characterMaximumLength int32
	ParameterDefault       string
	Position               int32
}

func (p *PgxRoutineParams) BasicType() types.BasicKind {
	return toType(p.DataType)
}

func (p *PgxRoutineParams) BasicTypeInfo() types.BasicInfo {
	switch p.BasicType() {
	case types.Bool:
		return types.IsBoolean
	case types.Int32, types.Int64:
		return types.IsInteger
	case types.Float32, types.Float64:
		return types.IsFloat
	case types.String:
		return types.IsString
	default:
		return types.IsUntyped
	}
}

func (p *PgxRoutineParams) CheckAttr(fieldDefine string) string {
	panic("implement me")
}

func (p *PgxRoutineParams) CharacterMaximumLength() int {
	return int(p.characterMaximumLength)
}

func (p *PgxRoutineParams) Comment() string {
	return p.comment
}

func (p *PgxRoutineParams) Name() string {
	return p.name
}

func (p *PgxRoutineParams) AutoIncrement() bool {
	panic("implement me")
}

func (p *PgxRoutineParams) IsNullable() bool {
	panic("implement me")
}

func (p *PgxRoutineParams) Default() string {
	return p.ParameterDefault
}

func (p *PgxRoutineParams) Primary() bool {
	panic("implement me")
}

func (p *PgxRoutineParams) Type() string {
	return p.DataType
}

func (p *PgxRoutineParams) Required() bool {
	panic("implement me")
}

func (p *PgxRoutineParams) SetNullable(bool) {
	panic("implement me")
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
			Fnc:                    r,
			name:                   values[0].(string),
			DataType:               values[1].(string),
			DataName:               values[2].(string),
			CharacterSetName:       values[3].(string),
			characterMaximumLength: values[4].(int32),
			ParameterDefault:       values[5].(string),
			Position:               values[6].(int32),
		}

		if values[7].(string) == "IN" {
			r.params = append(r.params, row)
		} else {
			r.columns = append(r.columns, row)
		}

		return nil
	}, sqlGetFuncParams+" ORDER BY ordinal_position", r.sName)
}

func (r *Routine) SelectAndScanEach(ctx context.Context, each func() error, row dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {

	name := r.name + "("
	for i := range r.params {
		if i > 0 {
			name += ","
		}
		name += "$" + strconv.Itoa(i+1)
	}

	table := &Table{name: name + ")"}
	table.columns = make([]*Column, len(r.columns))
	for i, col := range r.columns {
		table.columns[i] = NewColumn(table, col.name, col.DataType, col.Default(), false,
			col.CharacterSetName, col.comment, col.DataName, col.CharacterMaximumLength(), false, false)
	}
	b := &dbEngine.SQLBuilder{Table: table}
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
	name := r.name + "("
	for i := range r.params {
		if i > 0 {
			name += ","
		}
		name += "$" + strconv.Itoa(i+1)
	}

	table := &Table{name: name + ")"}
	table.columns = make([]*Column, len(r.columns))
	for i, col := range r.columns {
		table.columns[i] = NewColumn(table, col.name, col.DataType, col.Default(), false,
			col.CharacterSetName, col.comment, col.DataName, col.CharacterMaximumLength(), false, false)
	}
	b := &dbEngine.SQLBuilder{Table: table}

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
			return each(values, b.SelectColumns())
		},
		sql,
		b.Args...)
}
