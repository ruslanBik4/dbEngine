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
	DB_TEST_INIT  = "run {cfg}/DB/test_init.ddl"
)

// regexp const for parsing DDL
var (
	ddlIndex        = regexp.MustCompile(`create\s+(?P<unique>unique)?\s*index(?:\s+if\s+not\s+exists)?\s+(?P<index>\w+)\s+on\s+(?P<table>\w+)(?:\s+using\s+\w+)?\s*\((?P<columns>[^;]+?)\)\s*(where\s+[^)]\))?`)
	regTable        = regexp.MustCompile(`create\s+(or\s+replace\s+view|table)\s+(?P<name>\w+)\s*\((?P<fields>(\s*(\w*)\s+(?P<define>[\w\[\]':\s]+(\(\d+(,\s*\d+)?\))?[\w\s]*)('[^']*')?,?)*)\s*(primary\s+key\s*\([^)]+\))?\s*\)`)
	regField        = regexp.MustCompile(`(\w+)\s+([\w\s]+(\(\d+(,\s*\d+)?\))?[\w\[\]\s_]*)`)
	regDefault      = regexp.MustCompile(`default\s+'?([^',\n]+)`)
	regDoesNotExist = regexp.MustCompile(`relation\s+"(\w+)" does not exist`)
)
