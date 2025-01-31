// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"strings"

	"github.com/pkg/errors"

	"github.com/ruslanBik4/logs"
)

func (p *ParserCfgDDL) runDDL(ddl string, args ...any) error {
	err := p.DB.Conn.ExecDDL(p.DB.ctx, ddl, args...)
	if err == nil {
		if p.DB.Conn.LastRowAffected() > 0 {
			logInfo(prefix, p.filename, ddl, p.line)
		} else if !strings.HasPrefix(strings.ToLower(ddl), "insert") {
			logInfo(prefix, p.filename, "executed: "+ddl, p.line)
		}
		p.err = nil
	} else if IsErrorAlreadyExists(err) {
		p.err = nil
		logInfo("DEBUG", p.filename, "already exists: "+ddl, p.line)
	} else if IsErrorForReplace(err) {
		p.err = err
	} else if err != nil {
		logError(err, ddl, p.filename)
		p.err = err
	}
	return p.err
}

func (p *ParserCfgDDL) checkDDLCreateIndex(ddl string) (*Index, error) {

	regIndex := regIndex
	lower := strings.ToLower(ddl)
	columns := regIndex.FindStringSubmatch(lower)
	if len(columns) == 0 {
		columns = regForeignIndex.FindStringSubmatch(lower)
		if len(columns) == 0 {
			return nil, nil
		}
		regIndex = regForeignIndex
	}

	var ind Index
	for i, name := range regIndex.SubexpNames() {

		if !(i < len(columns)) {
			return nil, errors.Errorf("out if columns '%s'!", name)
		}

		switch token := columns[i]; name {
		case "":
		case "fTable":
			ind.foreignTable = token
		case "onUpdate":
			ind.updateCascade = token
		case "onDelete":
			ind.deleteCascade = token
		case "index":
			ind.Name = token
		case "unique":
			ind.Unique = token == name
		case "where":
			ind.Where = token

		case "table":
			if token != p.Name() {
				return nil, errors.Errorf("bad table name '%s'! %s", token, ddl)
			}

		case "expr":
			expr := make([]string, 0)
			for _, name := range regExprSeparator.FindAllString(token, -1) {
				expr = append(expr, regColumn.FindAllString(name, -1)...)
				if col := p.FindColumn(name); col != nil {
					ind.AddColumn(name)
				} else if ind.Expr > "" {
					ind.Expr += ", " + name
				} else {
					ind.Expr += name
				}
			}

			if len(ind.Columns) == 0 {
				if len(expr) > 0 {
					//we must add ONLY first column
					name := expr[0]
					if isColFunc(name) {
						name = expr[1]
					}
					if col := p.FindColumn(name); col != nil {
						ind.AddColumn(col.Name())
					}
				} else if ind.Expr == "" {
					logs.StatusLog(token)
					return nil, ErrNotFoundColumn{p.Name(), token}
				}
			}

		default:
			logInfo(prefix, p.filename, name+token, p.line)
		}
	}

	return &ind, nil
}

func isColFunc(name string) bool {
	return name == "digest"
}

// ParserCfgDDL is interface for parsing DDL file
type ParserCfgDDL struct {
	Table
	DB         *DB
	err        error
	filename   string
	line       int
	parseOrder []func(string) bool
	updDLL     *strings.Builder
}

// NewParserCfgDDL create new instance of ParserCfgDDL
func NewParserCfgDDL(db *DB, table Table) *ParserCfgDDL {
	t := &ParserCfgDDL{Table: table, filename: table.Name() + ".ddl", DB: db}
	t.parseOrder = []func(string) bool{
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
		t.performsGrants,
	}

	return t
}

// Parse perform queries from ddl text
func (p *ParserCfgDDL) Parse(ddl string) error {
	p.line = 1
	for _, sql := range strings.Split(ddl, ";") {
		p.line += strings.Count(sql, "\n")

		sql = strings.TrimSpace(strings.TrimPrefix(sql, "\n"))
		if sql == "" || strings.TrimSpace(strings.Replace(sql, "\n", "", -1)) == "" ||
			strings.HasPrefix(sql, "--") {
			continue
		}

		if err := p.execSql(sql); err != nil {
			logError(err, ddl, p.filename)
		}

		if p.err != nil {
			logError(p.err, ddl, p.filename)
		}

		p.err = nil

	}

	return nil
}

func (p *ParserCfgDDL) execSql(sql string) error {
	for _, fnc := range p.parseOrder {
		if fnc(sql) {
			return nil
		}
	}

	return NewErrUnknownSql(sql, p.line)
}

func (p *ParserCfgDDL) performsCreateExt(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "create extension") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserCfgDDL) alterMaterializedView(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "create materialized view") {
		return false
	}

	if t, ok := p.DB.Cfg[string(RECREATE_MATERIAZE_VIEW)].(bool); ok && t {
		p.runDDL("DROP materialized view " + p.Table.Name())
	}
	p.runDDL(ddl)

	return true
}

func (p *ParserCfgDDL) performsInsert(ddl string) bool {
	if !strings.Contains(strings.ToLower(ddl), "insert") {
		return false
	}

	if !strings.Contains(strings.ToLower(ddl), "on conflict") {
		ddl += " ON CONFLICT  DO NOTHING "
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserCfgDDL) performsUpdate(ddl string) bool {
	if !strings.Contains(strings.ToLower(ddl), "update") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserCfgDDL) performsGrants(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "grant") {
		return false
	}

	p.runDDL(ddl)

	return true
}

func (p *ParserCfgDDL) updateView(ddl string) bool {
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
				p.err = errors.Errorf(errWrongTableName.Error(), fields[i])
				return false
			}

			err := p.DB.Conn.ExecDDL(p.DB.ctx, ddl)
			if err != nil {
				if IsErrorCntChgView(err) {
					err = p.DB.Conn.ExecDDL(p.DB.ctx, "DROP VIEW "+p.Name()+" CASCADE")
					if err == nil {
						err = p.DB.Conn.ExecDDL(p.DB.ctx, ddl)
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
