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
)

func TestCreator_MakeStruct(t *testing.T) {
	type fields struct {
		dst *CfgCreator
	}
	type args struct {
		table dbEngine.Table
	}

	table, err := csv.NewTable("/Users/ruslan_bik/GolandProjects/polymer/data/polymers.csv")
	if !assert.Nil(t, err) {
		t.FailNow()
	}

	_ = table.GetColumns(nil, nil)

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
	}{
		// {
		// 	"first",
		// 	fields{
		// 		cfg: "../../test/db",
		// 	},
		// 	args{
		// 		table: table,
		// 	},
		// 	false,
		// },
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := &Creator{cfg: tt.fields.dst}
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
		cfg *CfgCreator
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
			if got, err := NewCreator(&dbEngine.DB{Name: "test", Schema: "test"}, tt.args.cfg); !assert.Equal(t, tt.want, got) && assert.NotNil(t, err) {
				t.Errorf("NewCreator() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringsUpper(t *testing.T) {
	s := fmt.Sprintf(`%-21s:    psql.Get%sFromByte(ci, srcPart[%d], "%s")`,
		"Accounts",
		"Array"+strcase.ToCamel(strings.TrimPrefix("[]int32", "[]")), 1, "")

	assert.Equal(t, `Accounts             :    psql.GetArrayInt32FromByte(ci, srcPart[1], "")`, s)
}
