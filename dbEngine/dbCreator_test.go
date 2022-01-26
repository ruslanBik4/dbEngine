// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"reflect"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewParserTableDDL(t *testing.T) {
	type args struct {
		table Table
		db    *DB
	}
	tests := []struct {
		name string
		args args
		want *ParserTableDDL
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := NewParserTableDDL(tt.args.table, tt.args.db); !reflect.DeepEqual(got, tt.want) {
				t.Errorf("NewParserTableDDL() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_Parse(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
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
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if err := p.Parse(tt.args.ddl); (err != nil) != tt.wantErr {
				t.Errorf("Parse() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParserTableDDL_addComment(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.addComment(tt.args.ddl); got != tt.want {
				t.Errorf("addComment() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_alterColumn(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		sAlter    string
		fieldName string
		title     string
		fs        Column
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
		t.Run(tt.name, func(t *testing.T) {
			p := ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if err := p.alterColumn(tt.args.sAlter, tt.args.fieldName, tt.args.title, tt.args.fs); (err != nil) != tt.wantErr {
				t.Errorf("alterColumn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParserTableDDL_alterTable(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.alterTable(tt.args.ddl); got != tt.want {
				t.Errorf("alterTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_checkColumn(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		fs    Column
		title string
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
		t.Run(tt.name, func(t *testing.T) {
			p := ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if err := p.checkColumn(tt.args.fs, tt.args.title); (err != nil) != tt.wantErr {
				t.Errorf("checkColumn() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestParserTableDDL_checkPrimary(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		fs          Column
		fieldDefine string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			p.checkPrimary(tt.args.fs, tt.args.fieldDefine)
		})
	}
}

func TestParserTableDDL_createIndex(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	tests := []struct {
		name    string
		fields  fields
		ddl     string
		want    *Index
		wantErr bool
	}{
		{
			name: "simple index",
			fields: fields{
				Table: TableString{
					name:    "candidates",
					columns: SimpleColumns("name"),
				},
			},
			ddl: `create  index candidates_name_uindex on candidates (name);`,
			want: &Index{
				Name:    "candidates_name_uindex",
				Expr:    "",
				Unique:  false,
				Columns: []string{"name"},
			},
		},
		{
			name: "simple unique index",
			fields: fields{
				Table: TableString{
					name:    "candidates",
					columns: SimpleColumns("name"),
				},
			},
			ddl: `create unique index candidates_name_uindex on candidates (name);`,
			want: &Index{
				Name:    "candidates_name_uindex",
				Expr:    "",
				Unique:  true,
				Columns: []string{"name"},
			},
		},
		{
			name: "simple index with where",
			fields: fields{
				Table: TableString{
					name:    "candidates",
					columns: SimpleColumns("name", "email"),
				},
			},
			ddl: `create unique index candidates_email_uindex
    on candidates (email)
    where ((email)::text > ''::text);`,
			want: &Index{
				Name:    "candidates_email_uindex",
				Expr:    "",
				Unique:  true,
				Columns: []string{"email"},
			},
		},
		{
			name: "functional index",
			fields: fields{
				Table: TableString{
					name:    "trades",
					columns: SimpleColumns("year", "opendate"),
				},
			},
			ddl: `create index if not exists trades_years
    on trades (date_part('year' :: text, opendate))`,
			want: &Index{
				Name:    "trades_years",
				Expr:    "date_part('year' :: text, opendate",
				Unique:  false,
				Columns: []string{"opendate"},
			},
		},
		{
			name: "functional index (two column)",
			fields: fields{
				Table: TableString{
					name:    "trades",
					columns: SimpleColumns("year", "opendate"),
				},
			},
			ddl: `create index if not exists trades_years
    on trades (year, date_part('year' :: text, opendate))`,
			want: &Index{
				Name:    "trades_years",
				Expr:    "year, date_part('year' :: text, opendate",
				Unique:  false,
				Columns: []string{"year", "opendate"},
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			got, err := p.checkDdlCreateIndex(strings.ToLower(tt.ddl))
			if tt.wantErr && !assert.NotNil(t, err) {
				return
			}
			if !tt.wantErr && !assert.Nil(t, err) {
				return
			}
			if !assert.Equal(t, tt.want, got) {
				return
			}
		})
	}
}

func TestParserTableDDL_execSql(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		sql string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.execSql(tt.args.sql); got != tt.want {
				t.Errorf("execSql() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_performsCreateExt(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.performsCreateExt(tt.args.ddl); got != tt.want {
				t.Errorf("performsCreateExt() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_performsInsert(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.performsInsert(tt.args.ddl); got != tt.want {
				t.Errorf("performsInsert() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_performsUpdate(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.performsUpdate(tt.args.ddl); got != tt.want {
				t.Errorf("performsUpdate() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_runDDL(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	tests := []struct {
		name   string
		fields fields
		ddl    string
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			p.runDDL(tt.ddl)
		})
	}
}

func TestParserTableDDL_skipPartition(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.skipPartition(tt.args.ddl); got != tt.want {
				t.Errorf("skipPartition() = %v, want %v", got, tt.want)
			}
		})
	}
}

const testCandidate = `create table candidates
(
    id serial not null,
    platform_id integer not null default 0,
    platforms integer[],
    name character varying not null,
    salary integer not null default 0,
    email character varying not null default '',
    phone character varying not null default '',
    skype character varying not null default '',
    link character varying not null default '',
    linkedin character varying default '',
    str_companies character varying default '',
    status character varying not null default '',
    tag_id integer not null default 1,
    comments text not null default '',
    date timestamp with time zone not null default CURRENT_TIMESTAMP,
    recruter_id integer not null default 1,
    text_rezume text not null default '',
    sfera character varying not null default '',
    experience character varying not null default '',
    education character varying not null default '',
    language character varying not null default '',
    zapoln_profile integer,
    file character varying not null default '',
    avatar character varying not null default '',
    seniority_id integer not null default 1,
    date_follow_up date,
    vacancies integer[],
        PRIMARY KEY (id)
);
COMMENT ON TABLE candidates IS 'list of candidates';

create unique index candidates_name_uindex
    on candidates (name);

create unique index candidates_email_uindex
    on candidates (email)
    where (((email)::text > ''::text) AND (email IS NOT NULL));

create unique index candidates_mobile_uindex
    on candidates (phone)
    where (((phone)::text > ''::text) AND (phone IS NOT NULL));

create unique index candidates_linkedin_uindex
    on candidates (linkedin)
    where (((linkedin)::text > ''::text) AND (linkedin IS NOT NULL));

alter table candidates
    add constraint candidates_seniorities_id_fk
        foreign key (seniority_id) references seniorities
            on update cascade on delete set default;

alter table candidates
    add constraint candidates_tags_id_fk
        foreign key (tag_id) references tags
            on update cascade on delete set default
`

func TestParserTableDDL_updateIndex(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	testDB := DB{
		Cfg:           nil,
		Conn:          nil,
		Tables:        nil,
		Types:         nil,
		Routines:      nil,
		FuncsReplaced: nil,
		FuncsAdded:    nil,
		Name:          "DB_GET_SCHEMA",
	}
	tests := []struct {
		name   string
		fields fields
		ddl    string
		want   bool
	}{
		// TODO: Add test cases.
		{
			"simple index",
			fields{
				DB: &testDB,
				Table: &TableString{
					columns: nil,
					indexes: nil,
					name:    "test",
					comment: "",
				},
			},
			testCandidate,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.updateIndex(tt.ddl); got != tt.want {
				t.Errorf("updateIndex() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_updateTable(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.updateTable(tt.args.ddl); got != tt.want {
				t.Errorf("updateTable() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestParserTableDDL_updateView(t *testing.T) {
	type fields struct {
		Table        Table
		DB           *DB
		err          error
		filename     string
		line         int
		mapParse     []func(string) bool
		isCreateDone bool
	}
	type args struct {
		ddl string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   bool
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			p := &ParserTableDDL{
				Table:        tt.fields.Table,
				DB:           tt.fields.DB,
				err:          tt.fields.err,
				filename:     tt.fields.filename,
				line:         tt.fields.line,
				mapParse:     tt.fields.mapParse,
				isCreateDone: tt.fields.isCreateDone,
			}
			if got := p.updateView(tt.args.ddl); got != tt.want {
				t.Errorf("updateView() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestDB_ReadTableSQL(t *testing.T) {
	assert.IsType(t, (TypeCfgDB)(""), DB_SETTING, "test")
}
