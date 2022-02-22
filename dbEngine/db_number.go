// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import "go/types"

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

// BasicType return GoLangs type of column
func (c *NumberColumn) BasicType() types.BasicKind {
	return types.Int
}

// BasicTypeInfo return types.BasicInfo of column
func (c *NumberColumn) BasicTypeInfo() types.BasicInfo {
	return types.IsInteger
}

// CheckAttr check attributes of column on DB schema according to ddl-file
func (c *NumberColumn) CheckAttr(fieldDefine string) string {
	return ""
}

// CharacterMaximumLength return max of length text columns
func (c *NumberColumn) CharacterMaximumLength() int {
	return 0
}

// Comment of column
func (c *NumberColumn) Comment() string {
	return c.comment
}

// Name of column
func (c *NumberColumn) Name() string {
	return c.name
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
func (c *NumberColumn) SetDefault(str interface{}) {
	c.colDefault = str
}

func (c *NumberColumn) Foreign() *ForeignKey {
	return nil
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

// SetNullable set nullable flag of column
func (c *NumberColumn) SetNullable(f bool) {
	c.isNullable = f
}
