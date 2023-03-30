// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"fmt"
	"strings"
	"testing"

	"github.com/iancoleman/strcase"
	"github.com/stretchr/testify/assert"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/csv"
	"github.com/ruslanBik4/logs"
)

func TestCreator_MakeStruct(t *testing.T) {
	type fields struct {
		dst string
	}
	type args struct {
		table dbEngine.Table
	}

	table, err := csv.NewTable("/Users/ruslan/work/src/github.com/ruslanBik4/polymer/data/polymers.csv")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	_ = table.GetColumns(nil)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// {
		// 	"first",
		// 	fields{
		// 		dst: "../../test/db",
		// 	},
		// 	args{
		// 		table: table,
		// 	},
		// 	false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Creator{dst: tt.fields.dst}
			// if assert.NotNil(t, err) {
			// 	t.Error(err)
			// 	return
			// }

			if err := c.MakeStruct(tt.args.table); (err != nil) != tt.wantErr {
				t.Errorf("MakeStruct() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNewCreator(t *testing.T) {
	type args struct {
		dst string
	}
	tests := []struct {
		name string
		args args
		want *Creator
	}{
		// TODO: Add test cases.
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got, err := NewCreator(tt.args.dst, &dbEngine.DB{Name: "test", Schema: "test"}); !assert.Equal(t, tt.want, got) && assert.NotNil(t, err) {
				t.Errorf("NewCreator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringsUpper(t *testing.T) {
	logs.StatusLog(fmt.Sprintf(`%-21s:    psql.Get%sFromByte(ci, srcPart[%d], "%s")`,
		"Accounts",
		"Array"+strcase.ToCamel(strings.TrimPrefix("[]int32", "[]")), 1, ""),
	)
}
