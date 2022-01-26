// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"go/types"
)

// NumberColumn implement store data of column for test tables
type NumberColumn struct {
	comment, name   string
	req, isNullable bool
	colDefault      interface{}
}

// NewNumberColumn create new NumberColumn
func NewNumberColumn(name, comment string, req bool) *NumberColumn {
	return &NumberColumn{comment: comment, name: name, req: req}
}

// AutoIncrement return true if column is autoincrement
func (c *NumberColumn) AutoIncrement() bool {
	return false
}

// IsNullable return isNullable flag
func (c *NumberColumn) IsNullable() bool {
	return c.isNullable
}

// Default return default value of column
func (c *NumberColumn) Default() interface{} {
	return "0"
}

// SetDefault set default value into column
func (s *NumberColumn) SetDefault(str interface{}) {
	s.colDefault = str
}

// CheckAttr check attributes of column on DB schema according to ddl-file
func (c *NumberColumn) CheckAttr(fieldDefine string) string {
	return ""
}

// Comment of column
func (c *NumberColumn) Comment() string {
	return c.comment
}

// Primary return true if column is primary key
func (c *NumberColumn) Primary() bool {
	return true
}

// Type of column
func (c *NumberColumn) Type() string {
	return "int"
}

// Required return true if column need a value
func (c *NumberColumn) Required() bool {
	return c.req
}

// Name of column
func (c *NumberColumn) Name() string {
	return c.name
}

// CharacterMaximumLength return max of length text columns
func (c *NumberColumn) CharacterMaximumLength() int {
	return 0
}

// BasicType return golangs type of column
func (c *NumberColumn) BasicType() types.BasicKind {
	return types.Int
}

func (c *NumberColumn) BasicTypeInfo() types.BasicInfo {
	return types.IsInteger
}

// SetNullable set nullable flag of column
func (c *NumberColumn) SetNullable(f bool) {
	c.isNullable = f
}
