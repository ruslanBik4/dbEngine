// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package psql

import "github.com/pkg/errors"

// ErrUnknownRoutineType, ErrFunctionWithoutResultType are some errors
var (
	ErrUnknownType               = errors.New("Can't define unknown type!")
	ErrUnknownRoutineType        = errors.New("Can't add routine unknown type!")
	ErrFunctionWithoutResultType = errors.New("Can't add function without results type!")
)
