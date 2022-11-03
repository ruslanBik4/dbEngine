// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package csv

import (
	"bytes"
	"encoding/csv"
	"reflect"
	"testing"

	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

func TestTable_Columns(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	tests := []struct {
		name   string
		fields fields
		want   []dbEngine.Column
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.Columns(); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("Columns() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_ExecDDL(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx  context.Context
		sql  string
		args []interface{}
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.ExecDDL(tt.args.ctx, tt.args.sql, tt.args.args...); (err != nil) != tt.wantErr {
				t1.Errorf("ExecDDL() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_FindColumn(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   dbEngine.Column
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.FindColumn(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("FindColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_FindIndex(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   *dbEngine.Index
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.FindIndex(tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("FindIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_GetColumns(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.GetColumns(tt.args.ctx); (err != nil) != tt.wantErr {
				t1.Errorf("GetColumns() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_GetSchema(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx context.Context
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    map[string]dbEngine.Table
		want1   map[string]dbEngine.Routine
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			got, got1, err := t.GetSchema(tt.args.ctx)
			if (err != nil) != tt.wantErr {
				t1.Errorf("GetSchema() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("GetSchema() got = %v, want %v", got, tt.want)
			}
			if !reflect.DeepEqual(got1, tt.want1) {
				t1.Errorf("GetSchema() got1 = %v, want %v", got1, tt.want1)
			}
		})
	}
}

func TestTable_GetStat(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.GetStat(); got != tt.want {
				t1.Errorf("GetStat() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_InitConn(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx      context.Context
		filePath string
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"first",
			fields{
				columns:  nil,
				fileName: "",
				csv:      csv.NewReader(bytes.NewBufferString("r")),
			},
			args{
				ctx:      context.Background(),
				filePath: "/Users/ruslan/work/src/github.com/ruslanBik4/polymer/data/polymers.csv",
			},
			false,
		},
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.InitConn(tt.args.ctx, tt.args.filePath); (err != nil) != tt.wantErr {
				t1.Errorf("InitConn() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				res := make([]interface{}, len(t.columns))
				for i, col := range t.columns {
					res[i] = col.Name()
				}
				t1.Log(res...)
			}
		})
	}
}

func TestTable_Insert(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx     context.Context
		Options []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			got, err := t.Insert(tt.args.ctx, tt.args.Options...)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Insert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("Insert() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_Name(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.Name(); got != tt.want {
				t1.Errorf("Name() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_NewTable(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		name string
		typ  string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   dbEngine.Table
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.NewTable(tt.args.name, tt.args.typ); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("NewTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_RereadColumn(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		name string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   dbEngine.Column
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if got := t.ReReadColumn(nil, tt.args.name); !reflect.DeepEqual(got, tt.want) {
				t1.Errorf("ReReadColumn() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_Select(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx     context.Context
		Options []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.Select(tt.args.ctx, tt.args.Options...); (err != nil) != tt.wantErr {
				t1.Errorf("Select() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_SelectAndRunEach(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx     context.Context
		each    dbEngine.FncEachRow
		Options []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.SelectAndRunEach(tt.args.ctx, tt.args.each, tt.args.Options...); (err != nil) != tt.wantErr {
				t1.Errorf("SelectAndRunEach() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_SelectAndScanEach(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx      context.Context
		each     func() error
		rowValue dbEngine.RowScanner
		Options  []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			if err := t.SelectAndScanEach(tt.args.ctx, tt.args.each, tt.args.rowValue, tt.args.Options...); (err != nil) != tt.wantErr {
				t1.Errorf("SelectAndScanEach() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestTable_Update(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx     context.Context
		Options []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			got, err := t.Update(tt.args.ctx, tt.args.Options...)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Update() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("Update() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestTable_Upsert(t1 *testing.T) {
	type fields struct {
		columns  []dbEngine.Column
		fileName string
		csv      *csv.Reader
	}
	type args struct {
		ctx     context.Context
		Options []dbEngine.BuildSqlOptions
	}
	tests := []struct {
		name    string
		fields  fields
		args    args
		want    int64
		wantErr bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t1.Run(tt.name, func(t1 *testing.T) {
			t := &Table{
				columns:  tt.fields.columns,
				fileName: tt.fields.fileName,
				csv:      tt.fields.csv,
			}
			got, err := t.Upsert(tt.args.ctx, tt.args.Options...)
			if (err != nil) != tt.wantErr {
				t1.Errorf("Upsert() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t1.Errorf("Upsert() got = %v, want %v", got, tt.want)
			}
		})
	}
}
