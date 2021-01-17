// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestArgsForSelect(t *testing.T) {
	type args struct {
		args []interface{}
	}
	tests := []struct {
		name string
		args args
		want BuildSqlOptions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ArgsForSelect(tt.args.args...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ArgsForSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestColumnsForSelect(t *testing.T) {
	type args struct {
		columns []string
	}
	tests := []struct {
		name string
		args args
		want BuildSqlOptions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := ColumnsForSelect(tt.args.columns...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("ColumnsForSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLBuilder_InsertSql(t *testing.T) {
	type fields struct {
		Args      []interface{}
		columns   []string
		filter    []string
		posFilter int
		Table     Table
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"simple insert",
			fields{
				[]interface{}{time.Now()},
				[]string{"last_login"},
				nil,
				0,
				TableString{name: "StringTable"},
			},
			"INSERT INTO StringTable(last_login) VALUES ($1)",
			false,
		},
		{
			"two columns insert",
			fields{
				[]interface{}{1, "ruslan"},
				[]string{"last_login", "name"},
				nil,
				0,
				TableString{name: "StringTable"},
			},
			"INSERT INTO StringTable(last_login,name) VALUES ($1,$2)",
			false,
		},
		{
			"two columns insert according two filter columns",
			fields{
				[]interface{}{"ruslan", time.Now()},
				[]string{"last_login", "name"},
				nil,
				0,
				TableString{name: "StringTable"},
			},
			"INSERT INTO StringTable(last_login,name) VALUES ($1,$2)",
			false,
		},
		{
			"two columns insert according two filter columns & wrong args",
			fields{
				[]interface{}{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				TableString{name: "StringTable"},
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}
			got, err := b.InsertSql()
			if (err != nil) != tt.wantErr {
				t.Errorf("InsertSql() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("InsertSql() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLBuilder_Select(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
		{
			"simple insert",
			fields{
				[]interface{}{1, time.Now()},
				[]string{"last_login"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"last_login",
		},
		{
			"two columns update",
			fields{
				[]interface{}{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"last_login,name",
		},
		{
			"two columns update according two filter columns",
			fields{
				[]interface{}{1, 2, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"last_login,name",
		},
		{
			"two columns update according four filter columns",
			fields{
				[]interface{}{1, "ruslan", time.Now()},
				[]string{"last_login", "name", "id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"last_login,name,id,id_roles",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}
			if got := b.Select(); got != tt.want {
				t.Errorf("Select() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLBuilder_SelectSql(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		orderBy       []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"simple select",
			fields{
				nil,
				nil,
				nil,
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT * FROM StringTable",
			false,
		},
		{
			"select full columns",
			fields{
				[]interface{}{1},
				nil,
				[]string{"id"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT * FROM StringTable WHERE  id=$1 order by id",
			false,
		},
		{
			"one columns &one filter select",
			fields{
				[]interface{}{1},
				[]string{"last_login"},
				[]string{"id"},
				[]string{"last_login"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login FROM StringTable WHERE  id=$1 order by last_login",
			false,
		},
		{
			"two columns select",
			fields{
				[]interface{}{1},
				[]string{"last_login", "name"},
				[]string{"id"},
				[]string{"last_login", "name"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE  id=$1 order by last_login,name",
			false,
		},
		{
			"two columns select according two filter columns",
			fields{
				[]interface{}{1, 2},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE  id=$1 AND id_roles=$2",
			false,
		},
		{
			"two columns select according two filter columns & wrong args",
			fields{
				[]interface{}{1},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"",
			true,
		},
		{
			"two columns select according two filter columns & fetch & offset",
			fields{
				[]interface{}{1, 2},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE  id=$1 AND id_roles=$2 offset 5  fetch first 1 rows only ",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				OrderBy:   tt.fields.orderBy,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}

			if strings.Contains(tt.name, "fetch") {
				b.Offset = 5
				b.Limit = 1
			}
			got, err := b.SelectSql()
			assert.Equal(t, tt.wantErr, (err != nil), "SelectSql() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, got, tt.want, "SelectSql() got = %v, want %v", got, tt.want)

		})
	}
}

func TestSQLBuilder_Set(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name   string
		fields fields
		want   string
		err    error
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}

			got, err := b.Set()
			assert.Equal(t, tt.err, err)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestSQLBuilder_UpdateSql(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"simple update",
			fields{
				[]interface{}{1, time.Now()},
				[]string{"last_login"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"UPDATE StringTable SET  last_login=$1 WHERE  id=$2",
			false,
		},
		{
			"two columns update",
			fields{
				[]interface{}{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"UPDATE StringTable SET  last_login=$1, name=$2 WHERE  id=$3",
			false,
		},
		{
			"two columns update according two filter columns",
			fields{
				[]interface{}{1, 2, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"UPDATE StringTable SET  last_login=$1, name=$2 WHERE  id=$3 AND id_roles=$4",
			false,
		},
		{
			"two columns update according two filter columns & wrong args",
			fields{
				[]interface{}{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"",
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}
			got, err := b.UpdateSql()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpdateSql() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UpdateSql() got = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLBuilder_Where(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name   string
		fields fields
		want   string
		fnc    func(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool
	}{
		// TODO: Add test cases.
		{
			"simple where",
			fields{
				[]interface{}{1},
				nil,
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  id=$1",
			assert.Equal,
		},
		{
			"select full columns",
			fields{
				[]interface{}{1},
				nil,
				[]string{"<id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  id < $1",
			assert.Equal,
		},
		{
			"one columns &one filter select",
			fields{
				[]interface{}{1},
				[]string{"last_login"},
				[]string{">id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  id > $1",
			assert.Equal,
		},
		{
			"two columns select",
			fields{
				[]interface{}{1},
				[]string{"last_login", "name"},
				[]string{"id", "$name"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  id=$1 AND name ~ ('.*' + $2 + '$')",
			assert.Equal,
		},
		{
			"two columns select according two filter columns",
			fields{
				[]interface{}{1, 2},
				[]string{"last_login", "name"},
				[]string{"<id", ">id_roles"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  id < $1 AND id_roles > $2",
			func(t assert.TestingT, expected, actual interface{}, msgAndArgs ...interface{}) bool {
				return assert.Equal(t, expected, actual, msgAndArgs...)
			},
		},
		{
			"two columns select according two filter columns & wrong args",
			fields{
				[]interface{}{1, 3},
				[]string{"last_login", "name"},
				[]string{"~name", "^name"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			" WHERE  name ~ $1 AND name ~ ('^.*' + $2)",
			assert.Equal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}
			tt.fnc(t, tt.want, b.Where())
		})
	}
}

func TestSQLBuilder_values(t *testing.T) {
	type fields struct {
		Args          []interface{}
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tests := []struct {
		name   string
		fields fields
		want   string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := &SQLBuilder{
				Args:      tt.fields.Args,
				columns:   tt.fields.columns,
				filter:    tt.fields.filter,
				posFilter: tt.fields.posFilter,
				Table:     tt.fields.Table,
			}
			if got := b.values(); got != tt.want {
				t.Errorf("values() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestWhereForSelect(t *testing.T) {
	type args struct {
		columns []string
	}
	tests := []struct {
		name string
		args args
		want BuildSqlOptions
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := WhereForSelect(tt.args.columns...); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("WhereForSelect() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestSQLBuilder_UpsertSql(t *testing.T) {
	type fields struct {
		Args       []interface{}
		columns    []string
		posFilter  int
		Table      Table
		onConflict string
	}
	testTable := TableString{
		name: "StringTable",
		columns: append(
			SimpleColumns("last_login", "name", "id_roles"),
			&StringColumn{
				comment: "id",
				name:    "id",
				primary: true,
			}),
	}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"simple insert",
			fields{
				[]interface{}{1, time.Now()},
				[]string{"id", "last_login"},
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login) VALUES ($1,$2) ON CONFLICT (id) DO UPDATE SET  last_login=EXCLUDED.last_login",
			false,
		},
		{
			"two columns update",
			fields{
				[]interface{}{1, time.Now(), "ruslan"},
				[]string{"id", "last_login", "name"},
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET  last_login=EXCLUDED.last_login, name=EXCLUDED.name",
			false,
		},
		{
			"two columns update according two filter columns",
			fields{
				[]interface{}{1, time.Now(), "ruslan"},
				[]string{"id", "last_login", "name"},
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET  last_login=EXCLUDED.last_login, name=EXCLUDED.name",
			false,
		},
		{
			"two columns update according four filter columns",
			fields{
				[]interface{}{1, time.Now(), "ruslan", 2},
				[]string{"id", "last_login", "name", "id_roles"},
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name,id_roles) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET  last_login=EXCLUDED.last_login, name=EXCLUDED.name, id_roles=EXCLUDED.id_roles",
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			b := SQLBuilder{
				Args:       tt.fields.Args,
				columns:    tt.fields.columns,
				posFilter:  tt.fields.posFilter,
				Table:      tt.fields.Table,
				onConflict: tt.fields.onConflict,
			}
			got, err := b.UpsertSql()
			if (err != nil) != tt.wantErr {
				t.Errorf("UpsertSql() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("UpsertSql() got = %v, want %v", got, tt.want)
			}
		})
	}
}
