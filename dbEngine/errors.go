// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"strings"
)

// ErrNotFoundTable if not found table by name {Table}
type ErrNotFoundTable struct {
	Table string
}

func (err ErrNotFoundTable) Error() string {

	return fmt.Sprintf("Not table `%s` in schema ", err.Table)
}

// ErrNotFoundTable if not found table by name {Table}
type ErrNotFoundRoutine struct {
	Name, SName string
}

func (err ErrNotFoundRoutine) Error() string {

	return fmt.Sprintf("Not routine `%s`(%s) in schema ", err.Name, err.SName)
}

// ErrNotFoundColumn if not found in table {Table} field by name {Column}
type ErrNotFoundColumn struct {
	Table  string
	Column string
}

func NewErrNotFoundColumn(table string, column string) *ErrNotFoundColumn {
	return &ErrNotFoundColumn{Table: table, Column: column}
}

func (err ErrNotFoundColumn) Error() string {

	return fmt.Sprintf("Not field `%s` for table `%s` in schema ", err.Column, err.Table)

}

// ErrNotFoundColumn if not found in table {Table} field by name {Column}
type ErrWrongArgsLen struct {
	Table  string
	Filter []string
	Args   []interface{}
}

func NewErrWrongArgsLen(table string, column []string, args []interface{}) *ErrWrongArgsLen {
	return &ErrWrongArgsLen{Table: table, Filter: column, Args: args}
}

func (err ErrWrongArgsLen) Error() string {

	return fmt.Sprintf("Wrong argument len %d (expect %d) for table `%s` ", len(err.Args), len(err.Filter), err.Table)

}

func isErrorAlreadyExists(err error) bool {
	ignoreErrors := []string{
		"already exists",
	}

	for _, val := range ignoreErrors {
		if strings.Contains(err.Error(), val) {
			return true
		}
	}

	return false
}

func isErrorForReplace(err error) bool {
	ignoreErrors := []string{
		"cannot change return type of existing function",
		"cannot change name of input parameter",
	}
	for _, val := range ignoreErrors {
		if strings.Contains(err.Error(), val) {
			return true
		}

	}

	return false
}

// ErrWrongType if not found in field {Name} field by name {Column}
type ErrWrongType struct {
	Name     string
	TypeName string
	Attr     string
}

func NewErrWrongType(typeName, name, attr string) *ErrWrongType {
	return &ErrWrongType{
		Name:     name,
		TypeName: typeName,
		Attr:     attr,
	}
}

func (err ErrWrongType) Error() string {

	return fmt.Sprintf("Wrong type `%s` name attr `%s` `%s` ", err.TypeName, err.Name, err.Attr)

}

// ErrWrongType if not found in field {Name} field by name {Column}
type ErrUnknownSql struct {
	sql  string
	Line int
}

func NewErrUnknownSql(sql string, line int) *ErrUnknownSql {
	return &ErrUnknownSql{
		sql:  sql,
		Line: line,
	}
}

func (err *ErrUnknownSql) Error() string {

	return fmt.Sprintf("unknow sql `%s` for DB migration ", err.sql)
}
