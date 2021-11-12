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
	regColumns = regexp.MustCompile(`\(([^():]+)`)
)

const (
	DB_URL        = "dbURL"
	DB_GET_SCHEMA = "fillSchema"
	DB_MIGRATION  = "migration"
)

var ddlIndex = regexp.MustCompile(`create\s+(?P<unique>unique)?\s*index(?:\s+if\s+not\s+exists)?\s+(?P<index>\w+)\s+on\s+(?P<table>\w+)(?:\s+using\s+\w+)?\s*\((?P<columns>[^;]+?)\)\s*(where\s+[^)]\))?`)
