package dbEngine

import (
	"testing"

	"github.com/go-errors/errors"
	"github.com/stretchr/testify/assert"
)

func TestIsErrorScanDest(t *testing.T) {
	type args struct {
		err error
	}
	tests := []struct {
		name string
		args args
		want int
		res  bool
	}{
		// TODO: Add test cases.
		{
			"4",
			args{errors.New("can't scan into dest[4]")},
			4,
			true,
		},
		{
			"35",
			args{errors.New("can't scan into dest[35]")},
			35,
			true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, got1 := IsErrorScanDest(tt.args.err)
			assert.Equalf(t, tt.want, got, "IsErrorScanDest(%v)", tt.args.err)
			assert.Equalf(t, tt.res, got1, "IsErrorScanDest(%v)", tt.args.err)
		})
	}
}
