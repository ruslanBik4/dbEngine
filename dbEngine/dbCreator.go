// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/httpgo/logs"
)

type ParserTableDDL struct {
	Table
	*DB
	err          error
	filename     string
	line         int
	mapParse     []func(string) bool
	isCreateDone bool
}

func NewParserTableDDL(table Table, db *DB) *ParserTableDDL {
	t := &ParserTableDDL{Table: table, filename: table.Name() + ".dll", DB: db}
	t.mapParse = []func(string) bool{
		t.updateTable,
		t.addComment,
		t.updateIndex,
		t.skipPartition,
	}

	return t
}

func (p *ParserTableDDL) Parse(ddl string) error {
	p.line = 1
	for _, sql := range strings.Split(ddl, ";") {
		p.line += strings.Count(sql, "\n")

		if strings.TrimSpace(strings.Replace(sql, "\n", "", -1)) == "" {
			continue
		}

		if !p.execSql(sql) {
			logError(NewErrUnknownSql(sql, p.line), ddl, p.filename)
		}

		if p.err != nil {
			logError(p.err, ddl, p.filename)

		}

		p.err = nil

	}

	return nil
}

func (p *ParserTableDDL) execSql(sql string) bool {
	for i, fnc := range p.mapParse {
		if (!p.isCreateDone || (i > 0)) && fnc(sql) {
			p.isCreateDone = p.isCreateDone || (i == 0)
			return true
		}
	}

	return false
}

func (p *ParserTableDDL) addComment(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "comment") {
		return false
	}

	err := p.Conn.ExecDDL(context.TODO(), ddl)
	if err == nil {
		logInfo(prefix, p.filename, ddl, p.line)
	} else if isErrorAlreadyExists(err) {
		err = nil
	} else if err != nil {
		logError(err, ddl, p.filename)
	}

	return true
}

var regPartionTable = regexp.MustCompile(`create\s+table\s+(\w+)\s+partition`)

func (p *ParserTableDDL) skipPartition(ddl string) bool {
	fields := regPartionTable.FindStringSubmatch(ddl)
	if len(fields) == 0 {
		return false
	}

	_, ok := p.Tables[fields[1]]
	if !ok {
		err := p.Conn.ExecDDL(context.TODO(), ddl)
		if err == nil {
			logInfo(prefix, p.filename, ddl, p.line)
		} else if isErrorAlreadyExists(err) {
			err = nil
		} else if err != nil {
			logError(err, ddl, p.filename)
		}
	}

	return true
}

var regTable = regexp.MustCompile(`create\s+table\s+(?P<name>\w+)\s+\((?P<fields>(\s*(\w*)\s*(?P<define>[\w\[\]':\s]*(\(\d+\))?[\w\s]*)('[^']*')?,?)*)\s*(primary\s+key\s*\([^)]+\))?\s*\)`)

var regField = regexp.MustCompile(`(\w+)\s+([\w()\[\]\s]+)`)

func (p *ParserTableDDL) updateTable(ddl string) bool {
	var err error
	fields := regTable.FindStringSubmatch(strings.ToLower(ddl))
	if len(fields) == 0 {
		return false
	}

	for i, name := range regTable.SubexpNames() {
		if !(i < len(fields)) {
			return false
		}

		switch name {
		case "":
		case "name":
			if fields[i] != p.Name() {
				p.err = errors.New("bad table name! " + fields[i])
				return false
			}
		case "fields":

			nameFields := strings.Split(fields[i], ",")
			for _, name := range nameFields {

				title := regField.FindStringSubmatch(name)
				if len(title) < 3 ||
					strings.HasPrefix(strings.ToLower(title[1]), "primary") ||
					strings.HasPrefix(strings.ToLower(title[1]), "constraint") {
					continue
				}

				fieldName := title[1]
				if fs := p.FindColumn(fieldName); fs == nil {
					sql := " ADD COLUMN " + name
					err = p.addColumn(sql, fieldName)
				} else if !fs.Primary() {
					// don't chg primary column
					err = p.checkColumn(title[2], fs)
				}

			}
		}
	}

	p.err = err
	return true
}

func (p ParserTableDDL) checkColumn(title string, fs Column) (err error) {
	res := fs.CheckAttr(title)
	fieldName := fs.Name()
	if res > "" {
		err = ErrNotFoundColumn{
			Table:  p.Name(),
			Column: fieldName,
		}
		// change length
		if strings.Contains(res, "has length") {
			logs.DebugLog(res)
			attr := strings.Split(title, " ")
			if attr[0] == "character" {
				attr[0] += " " + attr[1]
			}

			sql := fmt.Sprintf(" type %s using %s::%[1]s", attr[0], fieldName)
			err = p.alterColumn(sql, fieldName, title, fs)
		}

		// change type
		if strings.Contains(res, "type") {
			attr := strings.Split(title, " ")
			if attr[0] == "double" {
				attr[0] += " " + attr[1]
			}
			sql := fmt.Sprintf(" type %s using %s::%[1]s", attr[0], fieldName)
			if attr[0] == "money" && fs.Type() == "double precision" {
				sql = fmt.Sprintf(
					" type %s using %s::numeric::%[1]s",
					attr[0], fieldName)
			}

			err = p.alterColumn(sql, fieldName, title, fs)
		}

		// set not nullable
		if strings.Contains(res, "is nullable") {
			err = p.alterColumn(" set not null", fieldName, title, fs)
			if err != nil {
				logs.ErrorLog(err)
			} else {
				fs.SetNullable(true)
			}
		}

		// set nullable
		if strings.Contains(res, "is not nullable") {
			err = p.alterColumn(" drop not null", fieldName, title, fs)
			if err != nil {
				logs.ErrorLog(err)
			} else {
				fs.SetNullable(false)
			}
		}

	}

	return err
}

func (p ParserTableDDL) updateIndex(ddl string) bool {
	columns := ddlIndex.FindStringSubmatch(strings.ToLower(ddl))
	if len(columns) == 0 {
		return false
	}

	ind, err := p.createIndex(columns)
	if err != nil {
		p.err = err
		return true
	}

	if p.FindIndex(ind.Name) != nil {
		logInfo(prefix, p.filename, "index '"+ind.Name+"' exists! ", p.line)
		//todo: check columns of index
		return true
	}

	err = p.Conn.ExecDDL(context.TODO(), ddl)
	if err == nil {
		logInfo(prefix, p.filename, ddl, p.line)
	} else if isErrorAlreadyExists(err) {
		err = nil
	} else if err != nil {
		p.err = err
	}

	return true
}

var ddlIndex = regexp.MustCompile(`create(?:\s+unique)?\s+index(?:\s+if\s+not\s+exists)?\s+(?P<index>\w+)\s+on\s+(?P<table>\w+)(?:\s+using\s+\w+)?\s*\((?P<columns>[^;]+?)\)\s*(where\s+[^)]\))?`)

func (p ParserTableDDL) createIndex(columns []string) (*Index, error) {

	var ind Index
	for i, name := range ddlIndex.SubexpNames() {
		if !(i < len(columns)) {
			return nil, errors.New("out if columns!" + name)
		}

		switch name {
		case "":
		case "table":
			if columns[i] != p.Name() {
				return nil, errors.New("bad table name! " + columns[i])
			}
		case "index":
			// todo implement
			ind.Name = columns[i]
		case "columns":
			ind.Columns = strings.Split(columns[i], ",")
			for _, name := range ind.Columns {
				if p.FindColumn(name) == nil {
					return nil, ErrNotFoundColumn{p.Name(), name}
				}
				logInfo(prefix, p.filename, "new index column: "+name, p.line)
			}

		default:
			logInfo(prefix, p.filename, name+columns[i], p.line)
		}

	}

	// todo: chg after implement method
	return &ind, nil
}

func (p ParserTableDDL) addColumn(sAlter string, fieldName string) error {
	err := p.Conn.ExecDDL(context.TODO(), "ALTER TABLE "+p.Name()+sAlter)
	if err != nil {
		logs.ErrorLog(err, `. Field %s.%s`, p.Name(), fieldName)
	} else {
		logInfo(prefix, p.filename, sAlter, p.line)
		p.ReReadColumn(fieldName)
	}

	return err
}

func (p ParserTableDDL) alterColumn(sAlter string, fieldName, title string, fs Column) error {
	sql := "ALTER TABLE " + p.Name() + " alter column " + fieldName + sAlter
	err := p.Conn.ExecDDL(context.TODO(), sql)
	if err != nil {
		logs.ErrorLog(err,
			`. Field %s.%s, different with define: '%s' %s, sql: %s`,
			p.Name, fieldName, title, fs, sql)
	} else {
		logInfo(prefix, p.filename, sql, p.line)
		p.ReReadColumn(fieldName)
	}

	return err
}
