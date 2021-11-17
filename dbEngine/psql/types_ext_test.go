// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"math/big"
	"testing"

	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
)

func TestDecimal_Set(t *testing.T) {
	tests := []struct {
		name    string
		src     interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"one",
			[]byte("1"),
			false,
		},
		{
			"one",
			big.NewInt(10000).Bytes(),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := &Decimal{}
			if err := dst.Set(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			} else {
				assert.Equal(t, tt.src, dst.Int.Bytes())
			}
		})
	}
}

func TestNumeric_Set(t *testing.T) {
	type fields struct {
		Numeric *pgtype.Numeric
	}
	type args struct {
		src interface{}
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
			dst := &Numeric{
				Numeric: tt.fields.Numeric,
			}
			if err := dst.Set(tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
