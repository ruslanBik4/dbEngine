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

	"github.com/ruslanBik4/logs"
)

// ParserTableDDL is interface for parsing DDL file
type ParserTableDDL struct {
	Table
	*DB
	err          error
	filename     string
	line         int
	mapParse     []func(string) bool
	isCreateDone bool
}

// NewParserTableDDL create new instance of ParserTableDDL
func NewParserTableDDL(table Table, db *DB) *ParserTableDDL {
	t := &ParserTableDDL{Table: table, filename: table.Name() + ".ddl", DB: db}
	t.mapParse = []func(string) bool{
		t.updateTable,
		t.updateView,
		t.addComment,
		t.updateIndex,
		t.skipPartition,
		t.performsInsert,
		t.performsUpdate,
		t.performsCreateExt,
		t.alterTable,
	}

	return t
}

// Parse perform queries from ddl text
func (p *ParserTableDDL) Parse(ddl string) error {
	p.line = 1
	for _, sql := range strings.Split(ddl, ";") {
		p.line += strings.Count(sql, "\n")

		sql = strings.TrimSpace(strings.TrimPrefix(sql, "\n"))
		if strings.TrimSpace(strings.Replace(sql, "\n", "", -1)) == "" ||
			strings.HasPrefix(sql, "--") {
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

func (p *ParserTableDDL) performsCreateExt(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "create extension") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserTableDDL) alterTable(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "alter table") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserTableDDL) performsInsert(ddl string) bool {
	if !strings.Contains(strings.ToLower(ddl), "insert") {
		return false
	}

	if !strings.Contains(strings.ToLower(ddl), "on conflict") {
		ddl += " ON CONFLICT  DO NOTHING "
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserTableDDL) performsUpdate(ddl string) bool {
	if !strings.Contains(strings.ToLower(ddl), "update") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserTableDDL) addComment(ddl string) bool {
	switch lwrDdl := strings.ToLower(ddl); {
	case !strings.Contains(lwrDdl, "comment"):
		return false
	case strings.Contains(lwrDdl, "table") && strings.Contains(ddl, "'"+p.Table.Comment()+"'"):
		return true
	case strings.Contains(lwrDdl, "view") && strings.Contains(ddl, "'"+p.Table.Comment()+"'"):
		return true
	case strings.Contains(lwrDdl, "column"):
		posP := strings.Index(lwrDdl, ".")
		posI := strings.Index(lwrDdl, " is ")
		if posP < 1 || posI < 1 {
			logError(&ErrUnknownSql{Line: p.line, Msg: "not found column name"}, ddl, p.filename)
			return true
		}

		colName := strings.TrimSpace(lwrDdl[posP+1 : posI])
		col := p.Table.FindColumn(colName)
		if col == nil {
			logError(&ErrUnknownSql{Line: p.line, Msg: "not found column " + colName}, ddl, p.filename)
			return true
		}

		if strings.Contains(ddl, "'"+col.Comment()+"'") {
			return true
		}
	}

	p.runDDL(ddl)

	return true
}

var regPartitionTable = regexp.MustCompile(`create\s+table\s+(\w+)\s+partition`)

func (p *ParserTableDDL) skipPartition(ddl string) bool {
	fields := regPartitionTable.FindStringSubmatch(strings.ToLower(ddl))
	if len(fields) == 0 {
		return false
	}

	_, ok := p.Tables[fields[1]]
	if !ok {
		p.runDDL(ddl)
	}

	return true
}

func (p *ParserTableDDL) runDDL(ddl string) {
	err := p.Conn.ExecDDL(context.TODO(), ddl)
	if err == nil {
		if p.Conn.LastRowAffected() > 0 {
			logInfo(prefix, p.filename, ddl, p.line)
		}
	} else if IsErrorAlreadyExists(err) {
		err = nil
	} else if err != nil {
		logError(err, ddl, p.filename)
		p.err = err
	}
}

var regView = regexp.MustCompile(`create\s+or\s+replace\s+view\s+(?P<name>\w+)\s+as\s+select`)

func (p *ParserTableDDL) updateView(ddl string) bool {
	fields := regView.FindStringSubmatch(strings.ToLower(ddl))
	if len(fields) == 0 {
		return false
	}

	for i, name := range regView.SubexpNames() {
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

			err := p.Conn.ExecDDL(context.TODO(), ddl)
			if err != nil {
				if IsErrorCntChgView(err) {
					err = p.Conn.ExecDDL(context.TODO(), "DROP VIEW "+p.Name()+" CASCADE")
					if err == nil {
						err = p.Conn.ExecDDL(context.TODO(), ddl)
					}
				}

				if err != nil {
					p.err = err
				}
			}

			return true
		}
	}

	return false

}

var regTable = regexp.MustCompile(`create\s+(or\s+replace\s+view|table)\s+(?P<name>\w+)\s*\((?P<fields>(\s*(\w*)\s*(?P<define>[\w\[\]':\s]*(\(\d+(,\d+)?\))?[\w\s]*)('[^']*')?,?)*)\s*(primary\s+key\s*\([^)]+\))?\s*\)`)

var regField = regexp.MustCompile(`(\w+)\s+([\w()\[\]\s_]+)`)

func (p *ParserTableDDL) updateTable(ddl string) bool {
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
				fieldDefine := title[2]
				if fs := p.FindColumn(fieldName); fs == nil {
					p.runDDL("ALTER TABLE " + p.Name() + " ADD COLUMN " + name)
				} else if fs.Primary() {
					p.checkPrimary(fs, fieldDefine)

				} else {
					// don't chg primary column
					p.err = p.checkColumn(fs, fieldDefine)
				}

			}
		}
	}

	return true
}

func (p *ParserTableDDL) checkPrimary(fs Column, fieldDefine string) {
	res := fs.CheckAttr(fieldDefine)
	fieldName := fs.Name()
	// change type
	if strings.Contains(res, "type") {
		attr := strings.Split(fieldDefine, " ")
		if attr[0] == "double" {
			attr[0] += " " + attr[1]
		} else if attr[0] == "serial" {
			attr[0] = "integer"
		}

		sql := fmt.Sprintf(" type %s using %s::%[1]s", attr[0], fieldName)
		if attr[0] == "money" && fs.Type() == "double precision" {
			sql = fmt.Sprintf(
				" type %s using %s::numeric::%[1]s",
				attr[0], fieldName)
		}

		p.err = p.alterColumn(sql, fieldName, fieldDefine, fs)
	}
}

var regDefault = regexp.MustCompile(`default\s+'?([^',\n]+)`)

func (p ParserTableDDL) checkColumn(fs Column, title string) (err error) {
	res := fs.CheckAttr(title)
	fieldName := fs.Name()
	defaults := regDefault.FindStringSubmatch(strings.ToLower(title))
	colDef, ok := fs.Default().(string)
	if len(defaults) > 1 && (!ok || strings.ToLower(colDef) != defaults[1]) {
		err = p.alterColumn(" set "+defaults[0], fieldName, title, fs)
		if err != nil {
			logs.DebugLog(defaults, title)
		}
	}

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

func (p *ParserTableDDL) updateIndex(ddl string) bool {
	columns := ddlIndex.FindStringSubmatch(strings.ToLower(ddl))
	if len(columns) == 0 {
		return false
	}

	ind, err := p.createIndex(columns, ddlIndex)
	if err != nil {
		p.err = err
		return true
	}

	if pInd := p.FindIndex(ind.Name); pInd != nil {
		if pInd.Expr != ind.Expr {
			logInfo(prefix, p.filename,
				"index '"+ind.Name+"' exists! New expr '"+ind.Expr+"' (old ="+pInd.Expr,
				p.line)
			return true
		}

		columns := pInd.Columns
		for i, name := range ind.Columns {

			if i < len(columns) && columns[i] == name {
				continue
			}
			isFound := false
			for _, col := range columns {
				if col == name {
					isFound = true
					break
				}
			}
			if !isFound {
				logInfo(prefix, p.filename,
					"index '"+pInd.Name+"' exists! New column '"+name+"'"+strings.Join(pInd.Columns, ","),
					p.line)
				p.runDDL("DROP INDEX " + pInd.Name)
				p.runDDL(ddl)
			}

		}

		return true
	}

	p.runDDL(ddl)

	return true
}

func (p ParserTableDDL) createIndex(columns []string, regexp *regexp.Regexp) (*Index, error) {

	var ind Index
	for i, name := range regexp.SubexpNames() {
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
			for i, colDdl := range ind.Columns {
				col, isLegal := CheckColumn(colDdl, p)
				if !isLegal {
					return nil, ErrNotFoundColumn{p.Name(), colDdl}
				}
				ind.Columns[i] = col.Name()
			}
		case "unique":
			ind.Unique = columns[i] == name
		default:
			logInfo(prefix, p.filename, name+columns[i], p.line)
		}

	}

	// todo: chg after implement method
	return &ind, nil
}

func (p ParserTableDDL) alterColumn(sAlter string, fieldName, title string, fs Column) error {
	ddl := "ALTER TABLE " + p.Name() + " ALTER COLUMN " + fieldName + sAlter
	p.runDDL(ddl)
	if p.err == nil {
		logInfo(prefix, p.filename, ddl, p.line)
		p.ReReadColumn(fieldName)
	} else if !IsErrorForReplace(p.err) {
		logs.DebugLog(`Field %s.%s, different with define: '%s' %v`, p.Name(), fieldName, title, fs)
	}

	return p.err
}
