// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"regexp"
)

type FlagColumn uint8

const (
	MustNotNull FlagColumn = iota
	Nullable
	ChangeType
	ChangeLength
)

const prefix = "DB_CONFIG"

// regex const's
var (
	regColumns = regexp.MustCompile(`\(([^():]+)`)
)

// regexp const for parsing DDL
var (
	ddlIndex        = regexp.MustCompile(`create\s+(?P<unique>unique)?\s*index(?:\s+if\s+not\s+exists)?\s+(?P<index>\w+)\s+on\s+(?P<table>\w+)(?:\s+using\s+\w+)?\s*\((?P<columns>[^;]+?)\)\s*(where\s+[^)]\))?`)
	ddlForeignIndex = regexp.MustCompile(`alter\s+table\s+(?P<table>\w+)\s+add\s+constraint\s+(?P<index>\w+)\s+foreign\s+key\s+\((?P<columns>[^;]+?)\)\s+references\s+(\w+)`)
	regTable        = regexp.MustCompile(`create\s+(or\s+replace\s+view|table)\s+(?P<name>\w+)\s*\((?P<builderOpts>(\s*(\w*)\s+(?P<define>[\w\[\]':\s]+(\(\d+(,\s*\d+)?\))?[\w.\s]*)('[^']*')?,?)*)\s*(primary\s+key\s*\([^)]+\))?\s*\)`)
	regField        = regexp.MustCompile(`(\w+)\s+([\w\s]+(\(\d+(,\s*\d+)?\))?[\w\[\]\s_]*)`)
	regFieldName    = regexp.MustCompile(`^\w+$`)
	regDefault      = regexp.MustCompile(`default\s+'?([^',\n]+)`)
	regDoesNotExist = regexp.MustCompile(`relation\s+"(\w+)" does not exist`)
)

// DB_SETTING is name of value for setting to context
const (
	DB_SETTING = TypeCfgDB("set of CfgDB")
)

//todo: add hint & where to DB errors
