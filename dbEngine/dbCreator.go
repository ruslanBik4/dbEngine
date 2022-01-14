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
	DB           *DB
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
		t.alterMaterializedView,
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

func (p *ParserTableDDL) alterMaterializedView(ddl string) bool {
	if !strings.Contains(strings.ToLower(ddl), "materialized view") {
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

	_, ok := p.DB.Tables[fields[1]]
	if !ok {
		p.runDDL(ddl)
	}

	return true
}

func (p *ParserTableDDL) runDDL(ddl string) {
	err := p.DB.Conn.ExecDDL(context.TODO(), ddl)
	if err == nil {
		if p.DB.Conn.LastRowAffected() > 0 {
			logInfo(prefix, p.filename, ddl, p.line)
		} else if !strings.HasPrefix(ddl, "insert") {
			logInfo(prefix, p.filename, "executed: "+ddl, p.line)
		}
	} else if IsErrorAlreadyExists(err) {
		err = nil
		//logInfo("DEBUG", p.filename, "already exists: "+ddl, p.line)
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

			err := p.DB.Conn.ExecDDL(context.TODO(), ddl)
			if err != nil {
				if IsErrorCntChgView(err) {
					err = p.DB.Conn.ExecDDL(context.TODO(), "DROP VIEW "+p.Name()+" CASCADE")
					if err == nil {
						err = p.DB.Conn.ExecDDL(context.TODO(), ddl)
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

			nameFields := strings.Split(fields[i], ",\n")
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
			err = p.alterColumnsLength(fs, title, fieldName)
		}

		// change type
		if strings.Contains(res, "type") {
			err = p.alterColumnsType(fs, title, fieldName)
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

func (p ParserTableDDL) alterColumnsType(fs Column, title, fieldName string) error {
	attr := strings.Split(title, " ")
	typeDef := attr[0]
	if typeDef == "double" ||
		(strings.Contains(typeDef, "(") && !strings.Contains(typeDef, ")")) {
		typeDef += " " + attr[1]
	}
	sql := fmt.Sprintf(" type %s using %s::%[1]s", typeDef, fieldName)
	if attr[0] == "money" && fs.Type() == "double precision" {
		sql = fmt.Sprintf(
			" type %s using %s::numeric::%[1]s",
			attr[0], fieldName)
	}

	return p.alterColumn(sql, fieldName, title, fs)
}

func (p ParserTableDDL) alterColumnsLength(fs Column, title, fieldName string) error {
	attr := strings.Split(title, " ")

	typeDef := attr[0]
	if typeDef == "character" ||
		(strings.Contains(typeDef, "(") && !strings.Contains(typeDef, ")")) {

		typeDef += " " + attr[1]
	}

	sql := fmt.Sprintf(" type %s using %s::%[1]s", typeDef, fieldName)

	return p.alterColumn(sql, fieldName, title, fs)
}

func (p *ParserTableDDL) updateIndex(ddl string) bool {
	ind, err := p.checkDdlCreateIndex(ddl)
	if err != nil {
		p.err = err
		return true
	}

	if ind == nil {
		return false
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

func (p ParserTableDDL) checkDdlCreateIndex(ddl string) (*Index, error) {

	regIndex := ddlIndex
	columns := ddlIndex.FindStringSubmatch(strings.ToLower(ddl))
	if len(columns) == 0 {
		columns = ddlForeignIndex.FindStringSubmatch(strings.ToLower(ddl))
		if len(columns) == 0 {
			return nil, nil
		}
		regIndex = ddlForeignIndex
	}

	var ind Index
	for i, name := range regIndex.SubexpNames() {

		if !(i < len(columns)) {
			return nil, errors.New("out if columns!" + name)
		}

		value := columns[i]
		switch name {
		case "":
		case "table":
			if value != p.Name() {
				return nil, errors.New("bad table name! " + value)
			}
		case "index":
			// todo implement
			ind.Name = value
		case "columns":
			for _, colDdl := range strings.Split(value, ",") {
				col, isLegal := CheckColumn(colDdl, p)
				if isLegal {
					ind.Columns = append(ind.Columns, col.Name())
				}
			}

			if len(ind.Columns) == 0 {
				return nil, ErrNotFoundColumn{p.Name(), value}
			}

			if strings.Join(ind.Columns, ",") != value {
				ind.Expr = value
			}
		case "unique":
			ind.Unique = value == name
		default:
			logInfo(prefix, p.filename, name+value, p.line)
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
