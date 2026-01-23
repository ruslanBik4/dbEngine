// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"strconv"
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

// ErrNotFoundType if not found in table {Table} field by name {Column}
type ErrNotFoundType struct {
	Name string
	Type string
}

// NewErrNotFoundType create new error
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

	//  constraint "valid_email_check" for relation "users" already exists
	if RegAlreadyExists.MatchString(err.Error()) {
		return true
	}

	return false
}

// IsErrorScanDest indicates about errors duplicated
func IsErrorScanDest(err error) (int, bool) {
	if err == nil {
		return 0, false
	}

	//  constraint "valid_email_check" for relation "users" already exists
	if regErrScanDest.MatchString(err.Error()) {
		// must be as numeric striing
		n, err := strconv.Atoi(regErrScanDest.FindAllStringSubmatch(err.Error(), -1)[0][1])
		if err != nil {
			logs.ErrorLog(err)
		}
		return n, true
	}

	return 0, false
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
		ErrCannotAlterColumnUsedView,
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

// ErrUnknownSql if {sql} is unknown for parser
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

var errWrongTableName = errors.New("wrong table name '%v' %s")

func logError(err error, ddlSQL string, fileName string) {
	if pgErr, ok := err.(*pgconn.PgError); ok {
		pos := int(pgErr.Position - 1)
		if pos <= 0 {
			pos = strings.Index(ddlSQL, pgErr.ConstraintName) + 1
		}
		line := strings.Count(ddlSQL[:pos], "\n") + 1
		msg := fmt.Sprintf("%s: %s", pgErr.Message, pgErr.Detail)
		if pgErr.Where > "" {
			msg += "(" + pgErr.Where + ")"
		}
		if pgErr.Hint > "" {
			msg += "'" + pgErr.Hint + "'"
		}
		printError(fileName, line, msg)
	} else if e, ok := err.(*ErrUnknownSql); ok {
		printError(fileName, e.Line, e.Msg+e.sql+": not parse this SQL")
	} else {
		printError(fileName, 1, err.Error())
	}
}

func printError(fileName string, line int, msg string) {
	logs.CustomLog(logs.CRITICAL, "ERROR_"+preDB_CONFIG, fileName, line, msg, logs.FgErr)
}
