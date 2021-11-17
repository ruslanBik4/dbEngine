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

// Set has performing []byte src
func (dst *Numeric) Set(src interface{}) error {
	if dst.Numeric == nil {
		dst.Numeric = &pgtype.Numeric{Status: pgtype.Null}
	}

	if src == nil {
		return nil
	}

	switch value := src.(type) {
	case []byte:
		dst.Numeric = &pgtype.Numeric{Int: (&big.Int{}).SetBytes(value), Status: pgtype.Present}
	default:
		return dst.Numeric.Set(src)
	}

	return nil
}

// Numeric expanded pgtype.Numeric
type Decimal struct {
	*pgtype.Numeric
}

// Set has performing []byte src
func (dst *Decimal) Set(src interface{}) error {
	if dst.Numeric == nil {
		dst.Numeric = &pgtype.Numeric{Status: pgtype.Null}
	}

	if src == nil {
		return nil
	}

	switch value := src.(type) {
	case []byte:
		dst.Numeric = &pgtype.Numeric{Int: (&big.Int{}).SetBytes(value), Status: pgtype.Present}
	default:
		return dst.Numeric.Set(src)
	}

	return nil
}
