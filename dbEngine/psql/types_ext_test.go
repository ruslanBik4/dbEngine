package psql

import (
	"github.com/jackc/pgtype"
	"github.com/stretchr/testify/assert"
	"math/big"
	"testing"
)

func TestNewNumericFromFloat64(t *testing.T) {
	tests := []struct {
		name  string
		value float64
	}{
		// TODO: Add test cases.
		{
			"zero",
			0,
		},
		{
			"simple",
			100,
		},
		{
			"double",
			112.67,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			n := NewNumericFromFloat64(tt.value)
			assert.NotNil(t, n)
			assert.Implements(t, (*pgtype.Value)(nil), &n)
			assert.Implements(t, (*pgtype.TextDecoder)(nil), &n)
			assert.Implements(t, (*pgtype.BinaryDecoder)(nil), &n)
			var got float64
			err := n.AssignTo(&got)
			assert.Nil(t, err)
			assert.Equal(t, tt.value, got)
		})
	}
}

func TestNewNumericNull(t *testing.T) {
	n := NewNumericNull()
	assert.NotNil(t, n)
	assert.Implements(t, (*pgtype.Value)(nil), &n)
	assert.Implements(t, (*pgtype.TextDecoder)(nil), &n)
	assert.Implements(t, (*pgtype.BinaryDecoder)(nil), &n)
	assert.Nil(t, n.Int)
	assert.Equal(t, n.Status, pgtype.Null)
}

func TestNumeric_AssignTo(t *testing.T) {
	type fields struct {
		Numeric *pgtype.Numeric
	}
	type args struct {
		dst interface{}
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
			src := &Numeric{
				Numeric: tt.fields.Numeric,
			}
			if err := src.AssignTo(tt.args.dst); (err != nil) != tt.wantErr {
				t.Errorf("AssignTo() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumeric_DecodeBinary(t *testing.T) {
	type fields struct {
		Numeric *pgtype.Numeric
	}
	type args struct {
		ci  *pgtype.ConnInfo
		src []byte
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
			if err := dst.DecodeBinary(tt.args.ci, tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("DecodeBinary() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumeric_DecodeText(t *testing.T) {
	type fields struct {
		Numeric *pgtype.Numeric
	}
	type args struct {
		ci  *pgtype.ConnInfo
		src []byte
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
			if err := dst.DecodeText(tt.args.ci, tt.args.src); (err != nil) != tt.wantErr {
				t.Errorf("DecodeText() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestNumeric_Set(t *testing.T) {
	tests := []struct {
		name    string
		Numeric Numeric
		src     interface{}
		wantErr bool
	}{
		// TODO: Add test cases.
		{
			"simple",
			NewNumericNull(),
			[]byte("sds"),
			false,
		},
		{
			"Int",
			NewNumericNull(),
			big.NewInt(100),
			false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dst := tt.Numeric
			if err := dst.Set(tt.src); (err != nil) != tt.wantErr {
				t.Errorf("Set() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}
