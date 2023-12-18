// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"strings"

	"github.com/pkg/errors"

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
		if sql == "" || strings.TrimSpace(strings.Replace(sql, "\n", "", -1)) == "" ||
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
	if !strings.HasPrefix(strings.ToLower(ddl), "create materialized view") {
		return false
	}

	if t, ok := p.DB.Cfg[string(RECREATE_MATERIAZE_VIEW)].(bool); ok && t {
		p.runDDL("DROP materialized view " + p.Table.Name())
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
	if res := regCommentTable.FindAllStringSubmatch(ddl, -1); len(res) > 0 {
		tokens := res[0]
		if tokens[1] != p.Table.Name() {
			logs.StatusLog(ddl)
			logError(errors.Errorf(errWrongTableName.Error(), tokens[1], "comment table"), ddl, p.filename)
			return true
		}
		if tokens[2] == p.Table.Comment() {
			return true
		}
		p.runDDL(ddl)
		return true

	} else if res := regCommentColumn.FindAllStringSubmatch(ddl, -1); len(res) > 0 {
		tokens := res[0]
		if tokens[1] != p.Table.Name() {
			logError(errors.Errorf(errWrongTableName.Error(), tokens[1], "comment column"), ddl, p.filename)
			return true
		}
		colName := strings.ToLower(tokens[2])
		col := p.Table.FindColumn(colName)
		if col == nil {
			logError(&ErrUnknownSql{Line: p.line, Msg: "not found column " + colName}, ddl, p.filename)
			return true
		}

		if col.Comment() != strings.ReplaceAll(tokens[3], "''", "'") {
			logs.StatusLog(col.Comment(), tokens[3])
			p.runDDL(ddl)
		}
		return true
	}

	return false
}

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

func (p *ParserTableDDL) runDDL(ddl string, args ...any) {
	err := p.DB.Conn.ExecDDL(p.DB.ctx, ddl, args...)
	if err == nil {
		if p.DB.Conn.LastRowAffected() > 0 {
			logInfo(prefix, p.filename, ddl, p.line)
		} else if !strings.HasPrefix(strings.ToLower(ddl), "insert") {
			logInfo(prefix, p.filename, "executed: "+ddl, p.line)
		}
	} else if IsErrorAlreadyExists(err) {
		p.err = nil
		logInfo("DEBUG", p.filename, "already exists: "+ddl, p.line)
	} else if err != nil {
		logError(err, ddl, p.filename)
		p.err = err
	}
}

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
				p.err = errors.Errorf("bad table name '%s'", fields[i])
				return false
			}
		case "builderOpts":

			nameFields := strings.Split(fields[i], ",\n")
			for _, name := range nameFields {
				title := regField.FindStringSubmatch(name)
				if len(title) < 3 ||
					strings.HasPrefix(strings.ToLower(title[1]), "primary") ||
					strings.HasPrefix(strings.ToLower(title[1]), "constraint") {
					continue
				}

				colName := title[1]
				colDefine := title[2]
				if col := p.FindColumn(colName); col == nil {
					p.addColumn(ddl, colName, title[0], colDefine)
				} else if flags := col.CheckAttr(colDefine); col.Primary() {
					p.checkPrimary(col, colDefine, flags)
				} else {
					p.err = p.checkColumn(col, colDefine, flags)
				}
			}
		}
	}

	return true
}

func (p *ParserTableDDL) addColumn(ddl string, colName, sAlter, colDefine string) {
	sql := "ALTER TABLE " + p.Name() + " ADD COLUMN "
	p.runDDL(sql + sAlter)
	if p.err != nil && IsErrorNullValues(p.err) {
		logError(p.err, ddl, p.filename)
		p.runDDL(sql + strings.ReplaceAll(sAlter, "not null", ""))
		if p.err != nil {
			logError(p.err, ddl, p.filename)
			return
		}

		defaults := regDefault.FindStringSubmatch(strings.ToLower(colDefine))
		if len(defaults) > 1 && defaults[1] > "" {
			p.runDDL(fmt.Sprintf(`UPDATE %s SET %s=$1`, p.Name(), colName), defaults[1])
			if p.err != nil {
				logError(p.err, ddl, p.filename)
				return
			}

			err := p.alterColumn(colName, " ALTER COLUMN "+colName+" set not null")
			if err != nil {
				logError(err, ddl, p.filename)
			}
		}
	}
}

func (p *ParserTableDDL) checkPrimary(col Column, colDefine string, flags []FlagColumn) {

	for _, flag := range flags {
		// change only type
		if flag == ChgType {
			sAlter := p.alterColumnsType(col, colDefine, false)
			err := p.alterColumn(col.Name(), sAlter)
			if err != nil {
				logs.DebugLog(`Field %s.%s, different with define: '%s' %v`, p.Name(), col.Name(), colDefine, col)
				logError(err, sAlter, p.filename)
			}
		}
	}
}

func (p *ParserTableDDL) checkColumn(col Column, colDefine string, flags []FlagColumn) (err error) {
	defaults := regDefault.FindStringSubmatch(strings.ToLower(colDefine))
	colDef, hasDefault := col.Default().(string)
	chgDefault := len(defaults) > 1 && (!hasDefault || strings.ToLower(colDef) != strings.Trim(defaults[1], "'\n"))

	if !hasDefault && col.Default() != nil {
		logs.StatusLog("%#v", col.Default())
	}
	if len(flags) == 0 && !chgDefault {
		return nil
	}

	sAlter := make([]string, len(flags))
	colName := col.Name()
	for i, token := range flags {
		switch token {
		// change length
		case ChgLength:
			sAlter[i] = p.alterColumnsLength(col, colDefine)

		// change type
		case ChgType:
			sAlter[i] = p.alterColumnsType(col, colDefine, false)

		// change type to Array
		case ChgToArray:
			sAlter[i] = p.alterColumnsType(col, colDefine, true)

		// set not nullable
		case MustNotNull:
			sAlter[i] = fmt.Sprintf("ALTER COLUMN %s set not null", col.Name())

		// set nullable
		case Nullable:
			sAlter[i] = fmt.Sprintf("ALTER COLUMN %s drop not null", col.Name())
		}
	}

	if chgDefault {
		logs.DebugLog(hasDefault, strings.ToLower(colDef), defaults[1])
		sAlter = append(sAlter, fmt.Sprintf("ALTER COLUMN %s set %s", colName, defaults[0]))
		col.SetDefault(defaults[1])
		colDef = defaults[1]
	}

	err = p.alterColumn(col.Name(), sAlter...)
	if IsErrorNullValues(err) && hasDefault {
		// set defult value into ALL null the column
		ddl := fmt.Sprintf(`UPDATE %s SET %s=$1 WHERE %[2]s is null`, p.Name(), colName)
		p.runDDL(ddl, colDef)
		if p.err != nil {
			logError(p.err, ddl, p.filename)
		} else {
			err = p.alterColumn(col.Name(), sAlter...)
			if err != nil {
				logError(p.err, ddl, p.filename)
			}
		}
	}

	//if err != nil {
	//	logs.ErrorLog(err, hasDefault, colDef)
	//} else {
	//	col.SetNullable(true)
	//}
	//if err != nil {
	//	logs.ErrorLog(err)
	//} else {
	//	col.SetNullable(false)
	//}

	return err
}

func (p *ParserTableDDL) alterColumnsType(col Column, colDefine string, toArray bool) string {
	attr := strings.Split(colDefine, " ")
	typeDef := attr[0]
	colType := col.Type()
	switch typeDef {
	case "serial":
		typeDef = "integer"
	case "bigserial":
		typeDef = "bigint"
	case "money":
		if colType == "double precision" {
			typeDef = "numeric::" + typeDef
		}
	case "character":
		if strings.HasPrefix(attr[1], "varying") {
			typeDef += " " + attr[1]
		}
	case "double":
		if colType == "money" {
			typeDef = "numeric::" + typeDef
		}
		typeDef += " " + attr[1]
	}

	if toArray {
		return fmt.Sprintf("ALTER COLUMN %s drop default, ALTER COLUMN %[1]s type %s using array[%[1]s::%[3]s]::%[2]s",
			col.Name(),
			typeDef,
			strings.TrimSuffix(typeDef, "[]"),
		)
	}

	return fmt.Sprintf(" ALTER COLUMN %s drop default, ALTER COLUMN %[1]s type %s using %[1]s::%s", col.Name(), typeDef)

}

func (p *ParserTableDDL) alterColumnsLength(col Column, colDefine string) string {
	attr := strings.Split(colDefine, " ")

	typeDef := attr[0]
	if typeDef == "character" ||
		(strings.Contains(typeDef, "(") && !strings.Contains(typeDef, ")")) {

		typeDef += " " + attr[1]
	}

	return fmt.Sprintf("ALTER COLUMN %s type %s using %[1]s::%s", col.Name(), typeDef)
}

func (p *ParserTableDDL) updateIndex(ddl string) bool {
	ind, err := p.checkDDLCreateIndex(ddl)
	if err != nil {
		p.err = err
		return true
	}

	if ind == nil {
		return false
	}

	if oldInd := p.FindIndex(ind.Name); oldInd != nil {
		columns := oldInd.Columns
		hasChanges := !(len(columns) == len(ind.Columns))
		if hasChanges {
			logInfo(prefix, p.filename,
				fmt.Sprintf("index columns '%v' exists! New columns '%v'", oldInd.Columns, ind.Columns),
				p.line)

		}
		for i, name := range ind.Columns {

			hasChanges = !(i < len(columns) && columns[i] == name)
			if !hasChanges {
				continue
			}

			logInfo(prefix, p.filename,
				fmt.Sprintf("index '%s' exists! New column '%s'", oldInd.Name, name),
				p.line)
		}

		if oldInd.Expr != ind.Expr {
			if strings.Replace(oldInd.Expr, ")", "", -1) == strings.Replace(ind.Expr, ")", "", -1) {
				logWarning(prefix, p.filename,
					fmt.Sprintf("index '%s' exists & expr: '%s' diff with config:'%s' but this is some index expression", ind.Name, ind.Expr, oldInd.Expr),
					p.line)
			} else {
				logInfo(prefix, p.filename,
					fmt.Sprintf("index '%s' exists! New expr '%s' (old ='%s')", ind.Name, ind.Expr, oldInd.Expr),
					p.line)
				hasChanges = true
			}
		}

		if ind.foreignTable > "" && ind.deleteCascade == "set null" {
			logInfo(prefix, p.filename,
				fmt.Sprintf("reference to '%s' exists! Update  '%s' delete '%s'", ind.foreignTable,
					ind.updateCascade, ind.deleteCascade),
				p.line)
		}

		if oldInd.Unique != ind.Unique {
			logInfo(prefix, p.filename,
				fmt.Sprintf("New unique condition '%v' exists! Old  '%v'", ind.Unique, oldInd.Unique),
				p.line)
			hasChanges = true
		}
		//}

		if hasChanges {
			logs.StatusLog(ind)
			if ind.foreignColumn > "" {
				p.runDDL("DROP CONSTRAINT " + oldInd.Name)
			} else {
				p.runDDL("DROP INDEX " + oldInd.Name)
			}
			p.runDDL(ddl)
		}

		return true
	}
	// create new index
	p.runDDL(ddl)

	return true
}

func (p *ParserTableDDL) checkDDLCreateIndex(ddl string) (*Index, error) {

	regIndex := ddlIndex
	lower := strings.ToLower(ddl)
	columns := ddlIndex.FindStringSubmatch(lower)
	if len(columns) == 0 {
		columns = ddlForeignIndex.FindStringSubmatch(lower)
		if len(columns) == 0 {
			return nil, nil
		}
		regIndex = ddlForeignIndex
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

func (p *ParserTableDDL) alterColumn(colName string, sAlter ...string) error {
	ddl := fmt.Sprintf(`ALTER TABLE %s %s`, p.Name(), strings.Join(sAlter, ","))

	p.runDDL(ddl)
	switch {
	case p.err == nil:
		logInfo(prefix, p.filename, ddl, p.line)
		p.ReReadColumn(p.DB.ctx, colName)
	case IsErrorForReplace(p.err):
		logs.ErrorLog(p.err, `Field %s.%s, different with define: '%s' %v`, p.Name(), ddl)
	case IsErrorNullValues(p.err):
		defaults := regDefault.FindStringSubmatch(strings.ToLower(ddl))
		if len(defaults) > 1 && defaults[1] > "" {
			p.runDDL(fmt.Sprintf(`UPDATE %s SET %s=$1`, p.Name(), colName), defaults[1])
			if p.err != nil {
				logError(p.err, ddl, p.filename)
				return p.err
			}
		}
	default:
	}

	return p.err
}
