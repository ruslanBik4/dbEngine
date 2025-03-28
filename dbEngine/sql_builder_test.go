// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArgsForSelect(t *testing.T) {
	type args struct {
		args []any
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
		Args      []any
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
				[]any{time.Now()},
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
				[]any{1, "ruslan"},
				[]string{"last_login", "name"},
				nil,
				0,
				TableString{name: "StringTable"},
			},
			`INSERT INTO StringTable(last_login,name) VALUES ($1,$2)`,
			false,
		},
		{
			"two columns insert according two filter columns",
			fields{
				[]any{"ruslan", time.Now()},
				[]string{"last_login", "name"},
				nil,
				0,
				TableString{name: "StringTable"},
			},
			`INSERT INTO StringTable(last_login,name) VALUES ($1,$2)`,
			false,
		},
		{
			"two columns insert according two filter columns & wrong args",
			fields{
				[]any{1, "ruslan", time.Now()},
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
			assert.Equal(t, tt.want, strings.TrimSpace(got), "InsertSql() got = %v, want %v")
		})
	}
}

func TestSQLBuilder_Select(t *testing.T) {
	type fields struct {
		Args          []any
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
				[]any{1, time.Now()},
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
				[]any{1, "ruslan", time.Now()},
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
				[]any{1, 2, "ruslan", time.Now()},
				[]string{"last_login", "count(*) as allCount"},
				[]string{"id", "id_roles"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"last_login,count(*) as allCount",
		},
		{
			"two columns update according four filter columns",
			fields{
				[]any{1, "ruslan", time.Now()},
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
			require.Equal(t, tt.want, b.Select())
		})
	}
}

func TestSQLBuilder_SelectSql(t *testing.T) {
	type fields struct {
		Args          []any
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
				[]any{1},
				nil,
				[]string{"id"},
				[]string{"id"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT * FROM StringTable WHERE id=$1 order by id",
			false,
		},
		{
			"one columns &one filter select",
			fields{
				[]any{1},
				[]string{"last_login"},
				[]string{"id"},
				[]string{"last_login"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login FROM StringTable WHERE id=$1 order by last_login",
			false,
		},
		{
			"two columns select",
			fields{
				[]any{1},
				[]string{"last_login", "name"},
				[]string{"id"},
				[]string{"last_login", "name"},
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE id=$1 order by last_login,name",
			false,
		},
		{
			"two columns select according two filter columns",
			fields{
				[]any{1, 2},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE id=$1 AND id_roles=$2",
			false,
		},
		{
			"two columns select according two filter columns & wrong args",
			fields{
				[]any{1},
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
				[]any{1, 2},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			"SELECT last_login,name FROM StringTable WHERE id=$1 AND id_roles=$2 offset 5  fetch first 1 rows only ",
			false,
		},
		{
			"two columns with mix name select according two filter columns & fetch & offset",
			fields{
				[]any{1, 2},
				[]string{"last Login", "name of Organization"},
				[]string{"id", "id_roles"},
				nil,
				0,
				TableString{name: "StringTable"},
				nil,
			},
			`SELECT "last Login","name of Organization" FROM StringTable WHERE id=$1 AND id_roles=$2 offset 5  fetch first 1 rows only `,
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
			assert.Equal(t, tt.wantErr, err != nil, "SelectSql() error = %v, wantErr %v", err, tt.wantErr)
			assert.Equal(t, got, tt.want, "SelectSql() got = %v, want %v", got, tt.want)

		})
	}
}

func TestSQLBuilder_Set(t *testing.T) {
	type fields struct {
		Args          []any
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
		Args          []any
		columns       []string
		filter        []string
		posFilter     int
		Table         Table
		SelectColumns []Column
	}
	tableString := TableString{name: "StringTable"}
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr error
	}{
		// TODO: Add test cases.
		{
			"simple update",
			fields{
				[]any{1, time.Now()},
				[]string{"last_login"},
				[]string{"id"},
				0,
				tableString,
				nil,
			},
			"UPDATE StringTable SET  last_login=$1 WHERE id=$2",
			nil,
		},
		{
			"two columns update",
			fields{
				[]any{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id"},
				0,
				tableString,
				nil,
			},
			"UPDATE StringTable SET  last_login=$1, name=$2 WHERE id=$3",
			nil,
		},
		{
			"two columns update according two filter columns",
			fields{
				[]any{1, 2, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				tableString,
				nil,
			},
			"UPDATE StringTable SET  last_login=$1, name=$2 WHERE id=$3 AND id_roles=$4",
			nil,
		},
		{
			"two columns update according two filter columns & wrong args",
			fields{
				[]any{1, "ruslan", time.Now()},
				[]string{"last_login", "name"},
				[]string{"id", "id_roles"},
				0,
				tableString,
				nil,
			},
			"",
			&ErrWrongArgsLen{},
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
			require.IsType(t, tt.wantErr, err)
			require.Equal(t, tt.want, got)
		})
	}
}

type builderOpts struct {
	Args          []any
	columns       []string
	filter        []string
	posFilter     int
	Table         Table
	SelectColumns []Column
}

var (
	testFields = map[string]builderOpts{
		"simple where with id": {
			[]any{1},
			nil,
			[]string{"id"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"simple where with <id": {
			[]any{1},
			nil,
			[]string{"<id"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"simple where with ~name": {
			[]any{1},
			nil,
			[]string{"~name"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"case": {
			[]any{1},
			nil,
			[]string{"CASE WHEN m.wallet_type = 3 THEN m.pair_id = _pair_id ELSE true END"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"case with included param": {
			[]any{1},
			nil,
			[]string{"CASE WHEN m.wallet_type = 3 THEN m.pair_id = %s ELSE true END"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"some params with OR condition one of them included param": {
			[]any{"name", 1, 3},
			nil,
			[]string{
				"name",
				"(m.wallet_type = %s or m.pair_id = %[1]s OR m.wallet_type > m.pair_id)",
				"id",
			},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"null": {
			[]any{nil, "is not null", "is null"},
			nil,
			[]string{"id_parent", "id", "name"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"null with other simples arguments": {
			[]any{nil, 0, "is not null", 4, "is null"},
			nil,
			[]string{"id_parent", "id", "temp is null", "name", "id_user", "comment"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"borrowed > repaid": {
			[]any{1},
			[]string{
				"borrowed > repaid",
			},
			[]string{"borrowed > repaid"},
			0,
			TableString{
				name: "StringTable",
				columns: []Column{
					&StringColumn{
						comment:    "",
						name:       "borrowed",
						colDefault: "",
						req:        false,
						primary:    false,
						isNullable: false,
						maxLen:     0,
					},
					&StringColumn{
						comment:    "",
						name:       "repaid",
						colDefault: "",
						req:        false,
						primary:    false,
						isNullable: false,
						maxLen:     0,
					},
					&StringColumn{
						comment:    "",
						name:       "closed_at",
						colDefault: "",
						req:        false,
						primary:    false,
						isNullable: false,
						maxLen:     0,
					},
					&StringColumn{
						comment:    "",
						name:       "last_interest_at",
						colDefault: "",
						req:        false,
						primary:    false,
						isNullable: false,
						maxLen:     0,
					},
				},
			},
			nil,
		},
		"or": {
			[]any{1},
			nil,
			[]string{"(m.wallet_type = %s or m.pair_id = %[1]s OR m.wallet_type > m.pair_id)"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"one columns & one filter select": {
			[]any{1},
			[]string{"last_login"},
			[]string{">id"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"two columns": {
			[]any{1},
			[]string{"last_login", "name"},
			[]string{"id", "$name"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"two column with <, >": {
			[]any{1, 2},
			[]string{"last_login", "name"},
			[]string{"<id", ">id_roles"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"two column with <, > (hard names)": {
			[]any{1, 2, "test roles"},
			[]string{"last Login", "name Of Organization"},
			[]string{"<id", ">id_roles", "name Of Organization", `"last Login" > ""`},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"two column with array": {
			[]any{[]int8{1, 3}, 2},
			[]string{"last_login", "name"},
			[]string{"id", ">id_roles"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
		"two column with wrong args": {
			[]any{1, 3},
			[]string{"last_login", "name"},
			[]string{"~name", "^name"},
			0,
			TableString{name: "StringTable"},
			nil,
		},
	}
)

func TestSQLBuilder_Where(t *testing.T) {
	tests := []struct {
		name string
		want string
		fnc  func(t assert.TestingT, expected, actual any, msgAndArgs ...any) bool
	}{
		{
			"simple where with id",
			" WHERE id=$1",
			assert.Equal,
		},
		{
			"simple where with <id",
			" WHERE id < $1",
			assert.Equal,
		},
		{
			"simple where with ~name",
			" WHERE name ~ $1",
			assert.Equal,
		},
		{
			"null",
			" WHERE id_parent is null AND id is not null AND name is null",
			assert.Equal,
		},
		{
			"null with other simples arguments",
			" WHERE id_parent is null AND id=$1 AND temp is null AND name is not null AND id_user=$2 AND comment is null",
			assert.Equal,
		},
		{
			"case",
			" WHERE CASE WHEN m.wallet_type = 3 THEN m.pair_id = _pair_id ELSE true END",
			assert.Equal,
		},
		{
			"case with included param",
			" WHERE CASE WHEN m.wallet_type = 3 THEN m.pair_id = $1 ELSE true END",
			assert.Equal,
		},
		{
			"borrowed > repaid",
			" WHERE borrowed > repaid",
			assert.Equal,
		},
		{
			"or",
			" WHERE (m.wallet_type = $1 or m.pair_id = $1 OR m.wallet_type > m.pair_id)",
			assert.Equal,
		},
		{
			"some params with OR condition one of them included param",
			" WHERE name=$1 AND (m.wallet_type = $2 or m.pair_id = $2 OR m.wallet_type > m.pair_id) AND id=$3",
			assert.Equal,
		},
		{
			"one columns & one filter select",
			" WHERE id > $1",
			assert.Equal,
		},
		{
			"two columns",
			" WHERE id=$1 AND name ~ concat('.*', $2, '$')",
			assert.Equal,
		},
		{
			"two column with <, >",
			" WHERE id < $1 AND id_roles > $2",
			assert.Equal,
		},
		{
			"two column with <, > (hard names)",
			` WHERE id < $1 AND id_roles > $2 AND "name Of Organization"=$3 AND "last Login" > ""`,
			assert.Equal,
		},
		{
			"two column with array",
			" WHERE id=ANY($1) AND id_roles > $2",
			assert.Equal,
		},
		{
			"two column with wrong args",
			" WHERE name ~ $1 AND name ~ concat('^.*', $2)",
			assert.Equal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := testFields[tt.name]
			b := &SQLBuilder{
				Args:      opt.Args,
				columns:   opt.columns,
				filter:    opt.filter,
				posFilter: opt.posFilter,
				Table:     opt.Table,
			}
			tt.fnc(t, tt.want, b.Where(), tt.name)
		})
	}
}

func TestSQLBuilder_values(t *testing.T) {
	type fields struct {
		Args          []any
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
	tests := []struct {
		name string
		fnc  func(t assert.TestingT, expected, actual any, msgAndArgs ...any) bool
	}{
		// TODO: Add test cases.
		{
			"case with included param",
			assert.Equal,
		},
		{
			"some params with OR condition one of them included param",
			assert.Equal,
		},
		{
			"case with included param",
			assert.Equal,
		},
		{
			"or",
			assert.Equal,
		},
		{
			"borrowed > repaid",
			assert.Equal,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			opt := testFields[tt.name]
			b := &SQLBuilder{
				Args:      opt.Args,
				columns:   opt.columns,
				filter:    opt.filter,
				posFilter: opt.posFilter,
				Table:     opt.Table,
			}
			err := WhereForSelect(opt.columns...)(b)
			assert.Nil(t, err)
			//tt.fnc(t, )
		})
	}
}

func TestSQLBuilder_UpsertSql(t *testing.T) {
	type fields struct {
		Args       []any
		columns    []string
		posFilter  int
		Table      Table
		onConflict string
	}
	simpleTable := TableString{
		name: "simpleTable",
		columns: append(
			SimpleColumns("last_login", "name", "id_roles", "blob"),
			//&StringColumn{
			//	comment: " ",
			//	name:    " ",
			//	primary: true,
			//}
		),
		//indexes: Indexes{
		//	{
		//		Name:    "photos_test",
		//		Expr:    "",
		//		Unique:  true,
		//		Columns: []string{},
		//	},
		//},
	}

	testTable := TableString{
		name: "StringTable",
		columns: append(
			SimpleColumns("last_login", "name", "id_roles", "blob", "id_props"),
			&StringColumn{
				comment: "id",
				name:    "id",
				primary: true,
			}),
		indexes: Indexes{
			{
				Name:    "photos_test",
				Expr:    "digest(blob, 'sha1')",
				Unique:  true,
				Columns: []string{"blob", "name"},
			},
			{
				Name:    "props_test",
				Expr:    "",
				Unique:  true,
				Columns: []string{"name", "id_props"},
			},
		},
	}
	testTwoColumns := TableString{
		name: "StringTable",
		columns: append(
			// todo column blob change!
			SimpleColumns("id_roles", "blob"),
			&StringColumn{
				comment: "candidate_id",
				name:    "candidate_id",
				primary: true,
			},
			&StringColumn{
				comment: "vacancy_id",
				name:    "vacancy_id",
				primary: true,
			},
		),
	}
	columns := []string{"id", "last_login"}
	threeColumns := append(columns, "name")
	const sqlTmpl2Columns = "INSERT INTO StringTable(%s,%s) VALUES (%s) ON CONFLICT (%[1]s) DO UPDATE SET %s=EXCLUDED.%[2]s"
	const sqlTmpl3Columns = "INSERT INTO StringTable(%s,%s,%s) VALUES (%s) ON CONFLICT (%[1]s) DO UPDATE SET %s=EXCLUDED.%[2]s, %s=EXCLUDED.%[3]s"
	const sqlTmpl4Columns = "INSERT INTO StringTable(%s,%s,%s,%s) VALUES (%s) ON CONFLICT (%[1]s) DO UPDATE SET %s=EXCLUDED.%[2]s, %s=EXCLUDED.%[3]s, %s=EXCLUDED.%[4]s"
	tests := []struct {
		name    string
		fields  fields
		want    string
		wantErr assert.ErrorAssertionFunc
	}{
		{
			"simple table",
			fields{
				[]any{time.Now(), "ruslan", 2, "222"},
				[]string{"last_login", "name", "id_roles", "blob"},
				0,
				simpleTable,
				"",
			},
			"INSERT INTO simpleTable(last_login,name,id_roles,blob) VALUES ($1,$2,$3,$4)",
			assert.NoError,
		},
		{
			"simple insert",
			fields{
				[]any{1, time.Now()},
				columns,
				0,
				testTable,
				"",
			},
			fmt.Sprintf(sqlTmpl2Columns, columns[0], columns[1], "$1,$2"),
			assert.NoError,
		},
		{
			"two columns update",
			fields{
				[]any{1, time.Now(), "ruslan"},
				threeColumns,
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET last_login=EXCLUDED.last_login, name=EXCLUDED.name",
			assert.NoError,
		},
		{
			"two columns update according two filter columns",
			fields{
				[]any{1, time.Now(), "ruslan"},
				threeColumns,
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name) VALUES ($1,$2,$3) ON CONFLICT (id) DO UPDATE SET last_login=EXCLUDED.last_login, name=EXCLUDED.name",
			assert.NoError,
		},
		{
			"two columns update according four filter columns",
			fields{
				[]any{1, time.Now(), "ruslan", 2},
				[]string{"id", "last_login", "name", "id_roles"},
				0,
				testTable,
				"",
			},
			"INSERT INTO StringTable(id,last_login,name,id_roles) VALUES ($1,$2,$3,$4) ON CONFLICT (id) DO UPDATE SET last_login=EXCLUDED.last_login, name=EXCLUDED.name, id_roles=EXCLUDED.id_roles",
			assert.NoError,
		},
		{
			"two columns update according four filter columns & unique index",
			fields{
				[]any{1, time.Now(), "ruslan", 2},
				[]string{"last_login", "name", "id_roles", "blob"},
				0,
				testTable,
				"",
			},
			`INSERT INTO StringTable(last_login,name,id_roles,blob) VALUES ($1,$2,$3,$4) ON CONFLICT (digest(blob, 'sha1')) DO UPDATE SET last_login=EXCLUDED.last_login, name=EXCLUDED.name, id_roles=EXCLUDED.id_roles`,
			assert.NoError,
		},
		{
			"two columns update according four filter columns & primary index",
			fields{
				[]any{1, time.Now(), "ruslan", 2},
				[]string{"candidate_id", "vacancy_id", "id_roles", "blob"},
				0,
				testTwoColumns,
				"",
			},
			fmt.Sprintf(sqlTmpl3Columns, "candidate_id,vacancy_id", "id_roles", "blob", "$1,$2,$3,$4"),
			assert.NoError,
		},
		{
			"two columns update according four filter columns & unique index",
			fields{
				[]any{1, time.Now(), "ruslan", 2},
				[]string{"name", "id_props", "last_login", "id_roles"},
				0,
				testTable,
				"name,id_props",
			},
			fmt.Sprintf(sqlTmpl3Columns, "name,id_props", "last_login", "id_roles", "$1,$2,$3,$4"),
			assert.NoError,
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
			if !tt.wantErr(t, err) {
				return
			}
			assert.Equal(t, tt.want, strings.TrimSpace(got))
		})
	}
}
