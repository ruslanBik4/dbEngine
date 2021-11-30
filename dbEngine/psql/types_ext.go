// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import (
	"math/big"

	"github.com/jackc/pgtype"
)

// Numeric expanded pgtype.Numeric
type Numeric struct {
	*pgtype.Numeric
}

// NewNumericNull create Numeric with NULL
func NewNumericNull() Numeric {
	return Numeric{&pgtype.Numeric{Status: pgtype.Null}}
}

// NewNumericFromFloat64 create Numeric with NULL
func NewNumericFromFloat64(value float64) Numeric {
	numeric := &pgtype.Numeric{Status: pgtype.Present}
	_ = numeric.Set(value)

	return Numeric{numeric}
}

// Set has performing []byte src
func (dst *Numeric) Set(src interface{}) error {

	if dst.Numeric == nil {
		dst.Numeric = &pgtype.Numeric{Status: pgtype.Null}
	}

	switch value := src.(type) {
	case nil:

	case []byte:
		dst.Numeric = &pgtype.Numeric{Int: (&big.Int{}).SetBytes(value), Status: pgtype.Present}
	default:
		return dst.Numeric.Set(src)
	}

	return nil
}

// AssignTo has performing []byte dst
func (src *Numeric) AssignTo(dst interface{}) error {
	switch dst.(type) {
	case nil:
		dst = nil
	case []byte:
		if src.Status == pgtype.Present {
			dst = src.Numeric.Int.Bytes()
		} else {
			dst = nil
		}
	default:
		return src.Numeric.AssignTo(dst)
	}

	return nil
}

// DecodeText expand pgtype.Numeric.DecodeText
func (dst *Numeric) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if dst.Numeric == nil {
		dst.Numeric = &pgtype.Numeric{Status: pgtype.Null}
	}

	return dst.Numeric.DecodeText(ci, src)
}

// DecodeBinary expand pgtype.Numeric.DecodeBinary
func (dst *Numeric) DecodeBinary(ci *pgtype.ConnInfo, src []byte) error {
	if dst.Numeric == nil {
		dst.Numeric = &pgtype.Numeric{Status: pgtype.Null}
	}

	return dst.Numeric.DecodeBinary(ci, src)
}
