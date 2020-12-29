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

// Routine consist data of DB routine and operation for readint it and perform query
type Routine struct {
	conn       *Conn
	name       string
	ID         int
	Comment    string
	columns    []*PgxRoutineParams
	params     []*PgxRoutineParams
	temp_param *PgxRoutineParams
	paramMode  string
	overlay    *Routine
	Type       string
	lock       *sync.RWMutex
	sName      string
	DataType   string
	UdtName    string
}

func (r *Routine) Name() string {
	return r.name
}

func (r *Routine) Overlay() dbEngine.Routine {
	if r.overlay == nil {
		return nil
	}

	return r.overlay
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

func (r *Routine) Call(ctx context.Context, args ...interface{}) error {
	if r.Type != "PROCEDURE" {
		return dbEngine.ErrWrongType{Name: r.sName, TypeName: r.Type}
	}

	if len(args) > len(r.params) {
		filter := make([]string, len(r.params))
		for i, param := range r.params {
			filter[i] = param.name
		}

		return dbEngine.ErrWrongArgsLen{
			Table:  r.name,
			Filter: filter,
			Args:   args,
		}
	}

	if err := r.checkArgs(r.name, args); err != nil {
		return err
	}

	sql := "CALL " + r.correctName(r.name, args)

	logs.SetDebug(true)
	logs.DebugLog(sql)
	comTag, err := r.conn.Exec(ctx, sql, args...)
	if err != nil {
		logs.ErrorLog(err, "'%s' %s", comTag, strings.Split(sql, "\n")[0])
		return err
	}
	if mes := comTag.String(); mes != "CALL" {
		return errors.New(mes)
	}

	return nil
}

func (r *Routine) Params() []dbEngine.Column {
	res := make([]dbEngine.Column, len(r.params))
	for i, col := range r.params {
		res[i] = col
	}

	return res
}

func (r *Routine) GetFields(columns []dbEngine.Column) []interface{} {
	row := &PgxRoutineParams{
		Fnc: r,
	}

	fields := make([]interface{}, len(columns))
	for i, col := range columns {
		switch col.Name() {
		case "parameter_name":
			fields[i] = &row.name
		case "data_type":
			fields[i] = &row.DataType
		case "udt_name":
			fields[i] = &row.UdtName
		case "character_set_name":
			fields[i] = &row.CharacterSetName
		case "character_maximum_length":
			fields[i] = &row.characterMaximumLength
		case "parameter_default":
			fields[i] = &row.colDefault
		case "ordinal_position":
			fields[i] = &row.Position
		case "parameter_mode":
			fields[i] = &r.paramMode
		default:
			logs.ErrorLog(dbEngine.ErrNotFoundColumn{
				Table:  "sqlGetFuncParams",
				Column: col.Name(),
			})
		}
	}

	r.temp_param = row

	return fields
}

// GetParams получение значений полей для форматирования данных
// получение значений полей для таблицы
func (r *Routine) GetParams(ctx context.Context) error {

	return r.conn.SelectAndScanEach(ctx,
		func() error {
			if r.paramMode == "IN" {
				r.params = append(r.params, r.temp_param)
			} else {
				r.columns = append(r.columns, r.temp_param)
			}

			return nil
		},
		r,
		sqlGetFuncParams+" ORDER BY ordinal_position", r.sName)
}

func (r *Routine) SelectAndScanEach(ctx context.Context, each func() error, row dbEngine.RowScanner, Options ...dbEngine.BuildSqlOptions) error {

	sql, args, err := r.BuildSql(Options...)
	if err != nil {
		return err
	}

	return r.conn.SelectAndScanEach(ctx, each, row, sql, args...)
}

func (r *Routine) BuildSql(Options ...dbEngine.BuildSqlOptions) (string, []interface{}, error) {
	b := &dbEngine.SQLBuilder{Table: r.newTableForSQLBuilder()}
	for _, setOption := range Options {
		err := setOption(b)
		if err != nil {
			return "", nil, errors.Wrap(err, "setOption")
		}
	}

	err := r.checkArgs(r.name, b.Args)
	if err != nil {
		return "", nil, err
	}

	(b.Table).(*Table).name = r.correctName(r.name, b.Args)

	sql, err := b.SelectSql()
	if err != nil {
		return "", nil, err
	}

	return sql, b.Args, nil
}

func (r *Routine) correctName(name string, args []interface{}) string {
	name += "("
	for i := range args {
		if i > 0 {
			name += ","
		}
		// must be type of params
		name += fmt.Sprintf("$%d :: %s", i+1, r.params[i].Type())
	}
	name += ")"

	return name
}

func (r *Routine) checkArgs(tableName string, args []interface{}) error {
	if len(r.params) > len(args) {
		for _, param := range r.params[len(args):] {
			if param.Default() == nil {
				return dbEngine.NewErrWrongArgsLen(tableName,
					strings.Split(tableName, ","),
					args)
			}
		}
	}

	return nil
}

func (r *Routine) SelectAndRunEach(ctx context.Context, each dbEngine.FncEachRow, Options ...dbEngine.BuildSqlOptions) error {
	sql, args, err := r.BuildSql(Options...)
	if err != nil {
		return err
	}

	return r.conn.selectAndRunEach(
		ctx,
		each,
		sql,
		args...)
}

func (r *Routine) SelectOneAndScan(ctx context.Context, row interface{}, Options ...dbEngine.BuildSqlOptions) error {
	sql, args, err := r.BuildSql(Options...)
	if err != nil {
		return err
	}

	return r.conn.SelectOneAndScan(ctx, row, sql, args...)
}

func (r *Routine) newTableForSQLBuilder() *Table {

	table := &Table{name: r.name}
	table.columns = make([]*Column, len(r.columns))
	for i, col := range r.columns {
		table.columns[i] = &(col.Column)
	}

	return table
}
