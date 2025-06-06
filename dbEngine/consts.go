// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"regexp"
)

type FlagColumn uint8

func (f FlagColumn) String() string {
	switch f {
	case MustNotNull:
		return "MustNotNull"
	case Nullable:
		return "Nullable"
	case ChgType:
		return "ChgType"
	case ChgDefault:
		return "ChgDefault"
	case ChgLength:
		return "ChgLength"
	case ChgToArray:
		return "ChgToArray"
	default:
		return "Unknown"
	}
}

const (
	MustNotNull FlagColumn = iota
	Nullable
	ChgType
	ChgDefault
	ChgLength
	ChgToArray
)

const (
	tplAlterColumnType = " ALTER COLUMN %s TYPE %s USING %[1]s::%s"
	tplAlterNotNull    = " ALTER COLUMN %s SET not null"
	tplAlterSetDefault = " ALTER COLUMN %s SET DEFAULT '%s'"
)

const prefix = "DB_CONFIG"

// regex const's
var (
	regColumns       = regexp.MustCompile(`\(([^():]+)`)
	regColumn        = regexp.MustCompile(`'?\b[^'():,]+\b`)
	regExprSeparator = regexp.MustCompile(`[^(\s]*\([^)]*\)|[^'():,\s]+`)
)

// regexp const for parsing DDL
var (
	regIndex            = regexp.MustCompile(`create\s+(?P<unique>unique)?\s*index(?:\s+if\s+not\s+exists)?\s+(?P<index>\w+)\s+on\s+(?P<table>\w+)(?:\s+using\s+\w+)?\s*\((?P<expr>(?:\s*(?:[^(,]*\([^)]*\)|[^'():,]+),?)+)\)\s*(?P<where>where\s+[^)]\))?`)
	regForeignIndex     = regexp.MustCompile(`alter\s+table\s+(?P<table>\w+)\s+add\s+constraint\s+(?P<index>\w+)\s+foreign\s+key\s+\((?P<expr>[^;]+?)\)\s+references\s+(?P<fTable>\w+)[\S\s]+?on\s+update\s+(?P<onUpdate>[\s\w]+?)\s+on\s+delete\s+(?P<onDelete>[\s\w]+?)\s*`)
	regCommentTable     = regexp.MustCompile(`^\s*(?i)COMMENT\s+ON\s+(?:TABLE|VIEW)\s+(\w+)\s+IS\s+'(.+)'`)
	regCommentColumn    = regexp.MustCompile(`^\s*(?i)COMMENT\s+ON\s+COLUMN\s+("[^"]+"|\w+)\.("[^"]+"|\w+)\s+IS\s+'(.+)'`)
	regPartitionTable   = regexp.MustCompile(`create\s+table(?:\s+if\s+not\s+exists)?\s+(\w+)\s+partition`)
	regTable            = regexp.MustCompile(`(?i)create\s+(or\s+replace\s+view|table)\s+(?P<name>\w+)\s*\((?P<builderOpts>(\s*(\S+)\s+(?P<define>[\w\[\]':\s]+(\(\s*\d+(,\s*\d+)?\s*\))?[\w.+\s]*)(?:[\s\w]*\(?[\w.+\s]*(?:'[^']*')?(?:\s*::\s*\w+)?\)?)?,?)*)\s*(primary\s+key\s*\([^)]+\))?\s*\)`)
	regField            = regexp.MustCompile(`^\s*((?:"[^"]+")|\w+)\s+((?:[\w\s]+(?:\(\s*\d+(?:,\s*\d+)?\))?(?:\[\d*])*)?(?:[\s\w]*\(?[\w.+\s]*(?:'[^']*')?(?:\s*::\s*\w+)?\)?)?)`)
	regFieldName        = regexp.MustCompile(`^\w+$`)
	RegDefault          = regexp.MustCompile(`(?i)default\s+(\(?[\w.+\s]*(?:'[^']*')?(?:\s*::\s*\w+)?\)?)`)
	regView             = regexp.MustCompile(`create\s+or\s+replace\s+view\s+(?P<name>\w+)\s+(with\s*\([\w,=\s]+\)\s*)?as\s+select`)
	regRelationNotExist = regexp.MustCompile(`relation\s+"(\w+)" does not exist`)
	regTypeNotExist     = regexp.MustCompile(`type\s+"(\w+)"\s+does not exist`)
)

// DB_SETTING is name of value for setting to context
const (
	DB_SETTING              = TypeCfgDB("set of CfgDB")
	RECREATE_MATERIAZE_VIEW = TypeCfgDB("drop materiaze view before create")
)

// todo: add hint & where to DB errors
