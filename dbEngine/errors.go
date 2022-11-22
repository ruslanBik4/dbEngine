// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/jackc/pgx/v4"
	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"
)

// ErrDBNotFound error about wrong DB
var ErrDBNotFound = errors.New("DB not found")

// ErrNotFoundTable if not found table by name {Table}
type ErrNotFoundTable struct {
	Table string
}

// NewErrNotFoundTable create new error
func NewErrNotFoundTable(table string) *ErrNotFoundTable {
	return &ErrNotFoundTable{Table: table}
}

// Error implement error interface
func (err ErrNotFoundTable) Error() string {

	return fmt.Sprintf("Not table `%s` in schema ", err.Table)
}

// ErrNotFoundRoutine if not found table by name {Table}
type ErrNotFoundRoutine struct {
	Name, SName string
}

// Error implement error interface
func (err ErrNotFoundRoutine) Error() string {

	return fmt.Sprintf("Not routine `%s`(%s) in schema ", err.Name, err.SName)
}

// ErrNotFoundColumn if not found in table {Table} field by name {Column}
type ErrNotFoundColumn struct {
	Table  string
	Column string
}

// NewErrNotFoundColumn create new error
func NewErrNotFoundColumn(table string, column string) *ErrNotFoundColumn {
	return &ErrNotFoundColumn{Table: table, Column: column}
}

// Error implement error interface
func (err ErrNotFoundColumn) Error() string {

	return fmt.Sprintf("Not field `%s` for table `%s` in schema ", err.Column, err.Table)

}

// ErrNotFoundColumn if not found in table {Table} field by name {Column}
type ErrNotFoundType struct {
	Name string
	Type string
}

// NewErrNotFoundColumn create new error
func NewErrNotFoundType(name string, sType string) *ErrNotFoundType {
	return &ErrNotFoundType{Name: name, Type: sType}
}

// Error implement error interface
func (err ErrNotFoundType) Error() string {

	return fmt.Sprintf("Not field type `%s` (`%s`) in schema ", err.Name, err.Type)

}

// ErrWrongArgsLen if not found in table {Table} field by name {Column}
type ErrWrongArgsLen struct {
	Table  string
	Filter []string
	Args   []interface{}
}

// NewErrWrongArgsLen create new error
func NewErrWrongArgsLen(table string, column []string, args []interface{}) *ErrWrongArgsLen {
	return &ErrWrongArgsLen{Table: table, Filter: column, Args: args}
}

// Error implement error interface
func (err ErrWrongArgsLen) Error() string {
	return fmt.Sprintf("Wrong argument len %d (expect %d) for table `%s` ", len(err.Args), len(err.Filter), err.Table)
}

// IsErrorNullValues indicates about column can't add because has NOT NULL
func IsErrorNullValues(err error) bool {
	if err == nil {
		return false
	}

	return strings.Contains(err.Error(), "contains null values")
}

// IsErrorAlreadyExists indicates about errors duplicated
func IsErrorAlreadyExists(err error) bool {
	if err == nil {
		return false
	}

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

// IsErrorDoesNotExists indicates about errors not exists
func IsErrorDoesNotExists(err error) bool {
	if err == nil {
		return false
	}

	ignoreErrors := []string{
		"does not exist",
	}

	for _, val := range ignoreErrors {
		if strings.Contains(err.Error(), val) {
			return true
		}
	}

	return false
}

// IsErrorForReplace indicates about errors 'cannot change or replace"
func IsErrorForReplace(err error) bool {
	if err == nil {
		return false
	}

	ignoreErrors := []string{
		"cannot change return type of existing function",
		"cannot change name of input parameter",
		"cannot alter type of a column used by a view or rule",
	}
	for _, val := range ignoreErrors {
		if strings.Contains(err.Error(), val) {
			return true
		}

	}

	return false
}

// IsErrorCntChgView indicates about errors 'cannot change name of view column'
func IsErrorCntChgView(err error) bool {
	if err == nil {
		return false
	}

	ignoreErrors := []string{
		"cannot change name of view column",
	}
	for _, val := range ignoreErrors {
		if strings.Contains(err.Error(), val) {
			return true
		}

	}

	return false
}

var (
	regKeyWrong   = regexp.MustCompile(`Key\s+\((\w+)\)=\((.+)\)([^.]+)`)
	regDuplicated = regexp.MustCompile(`duplicate key value violates unique constraint "(\w*)"`)
)

// IsErrorDuplicated indicate about abort updating because there is a duplicated key found
func IsErrorDuplicated(err error) (map[string]string, bool) {
	if err == nil {
		return nil, false
	}

	if err == pgx.ErrNoRows {
		return nil, false
	}
	msg := err.Error()
	e, ok := errors.Cause(err).(*pgconn.PgError)
	if ok {
		msg = e.Detail
	}

	if s := regKeyWrong.FindStringSubmatch(msg); len(s) > 0 {
		return map[string]string{
			s[1]: "`" + s[2] + "`" + s[3],
		}, true
	}

	if s := regDuplicated.FindStringSubmatch(msg); len(s) > 0 {
		logs.DebugLog("%#v %[1]T", errors.Cause(err))
		return map[string]string{
			s[1]: "duplicate key value violates unique constraint",
		}, true
	}

	return nil, false
}

// ErrWrongType if not found in field {Name} field by name {Column}
type ErrWrongType struct {
	Name     string
	TypeName string
	Attr     string
}

// NewErrWrongType create new error
func NewErrWrongType(typeName, name, attr string) *ErrWrongType {
	return &ErrWrongType{
		Name:     name,
		TypeName: typeName,
		Attr:     attr,
	}
}

// Error implement error interface
func (err ErrWrongType) Error() string {

	return fmt.Sprintf("Wrong type `%s` name attr `%s` `%s` ", err.TypeName, err.Name, err.Attr)

}

// ErrUnknownSql if not found in field {Name} field by name {Column}
type ErrUnknownSql struct {
	sql  string
	Line int
	Msg  string
}

// NewErrUnknownSql create new error
func NewErrUnknownSql(sql string, line int) *ErrUnknownSql {
	return &ErrUnknownSql{
		sql:  sql,
		Line: line,
	}
}

// Error implement error interface
func (err *ErrUnknownSql) Error() string {
	return fmt.Sprintf("unknow sql `%s` for DB migration ", err.sql)
}
