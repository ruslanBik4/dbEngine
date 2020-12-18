// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"regexp"
)

const prefix = "DB_CONFIG"

// regex consts
var (
	regColumns = regexp.MustCompile(`\(([^()]+)\)`)
)
