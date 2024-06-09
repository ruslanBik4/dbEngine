package dbEngine

import (
	"fmt"
	"slices"
	"strings"

	"github.com/go-errors/errors"

	"github.com/ruslanBik4/logs"
)

func (p *ParserCfgDDL) updateTable(ddl string) bool {
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
			newNotNulls := make([]string, 0)
			chgNotNulls := make([]string, 0)

			for _, name := range nameFields {
				title := regField.FindStringSubmatch(name)
				if len(title) < 3 ||
					strings.HasPrefix(strings.ToLower(title[1]), "primary") ||
					strings.HasPrefix(strings.ToLower(title[1]), "constraint") {
					continue
				}

				sAlter, colName, colDefine := title[0], title[1], title[2]
				defaults := ""
				if newDef := RegDefault.FindStringSubmatch(strings.ToLower(colDefine)); len(newDef) > 0 {
					defaults = newDef[1]
				}
				if col := p.FindColumn(colName); col == nil {
					if strings.Contains(sAlter, "not null") && defaults > "" {
						newNotNulls = append(newNotNulls, colName)
					} else {
						logWarning("COLUMN", colName, "has't default will be add without flag SET nNULL", 0)
					}
					p.addColumn(colName, sAlter, colDefine)

				} else if flags := col.CheckAttr(colDefine); col.Primary() {
					p.checkPrimary(col, colDefine, flags)
				} else {
					p.checkColumn(col, colDefine, flags)
					if slices.Contains(flags, ChgDefault) && defaults > "" {
						chgNotNulls = append(chgNotNulls, colName)
					}
				}
			}

			if p.updDLL != nil {
				if len(newNotNulls) > 0 {
					p.updDLL.WriteString(";\nUPDATE " + p.Name() + " SET " + strings.Join(newNotNulls, "=DEFAULT, ") + "=DEFAULT;\n")
					for j, col := range newNotNulls {
						if j == 0 {
							p.updDLL.WriteString("ALTER TABLE " + p.Name())
						} else {
							p.updDLL.WriteRune(',')
						}
						p.updDLL.WriteString(fmt.Sprintf(tplAlterNotNull, col))
					}
				}
				for _, col := range chgNotNulls {
					p.updDLL.WriteString(";\nUPDATE " + p.Name() + " SET " + col + `=DEFAULT
where ` + col + ` is null;
ALTER TABLE ` + p.Name() + fmt.Sprintf(tplAlterNotNull, col) + `;
`)
				}
				sql := p.updDLL.String() + ";"
				logs.StatusLog(sql)
				p.runDDL(p.updDLL.String())
				if p.err != nil {
					logError(p.err, sql, p.filename)
				}
				p.updDLL = nil
				err := p.Table.GetColumns(p.DB.ctx, p.DB.Types)
				if err != nil {
					logs.ErrorLog(err, "during reread table columns")
				}
			}
		}
	}

	return true
}

func (p *ParserCfgDDL) chkAlterBuilder() {
	if p.updDLL == nil {
		p.updDLL = &strings.Builder{}
		p.updDLL.WriteString("ALTER TABLE " + p.Name() + " ")
	} else {
		p.updDLL.WriteRune(',')
	}
}

func (p *ParserCfgDDL) addColumn(colName, sAlter, colDefine string) {
	if strings.Contains(sAlter, "not null") {
		sAlter = strings.ReplaceAll(sAlter, "not null", "")
	}

	p.chkAlterBuilder()
	p.updDLL.WriteString(" ADD COLUMN " + sAlter)
}

func (p *ParserCfgDDL) checkPrimary(col Column, colDefine string, flags []FlagColumn) {
	for _, flag := range flags {
		// change only type
		if flag == ChgType {
			p.chkAlterBuilder()
			p.updDLL.WriteString(fmt.Sprintf(tplAlterColumnType, col.Name(), colDefine))
			return
		}
	}
}

const tplAlterColumnType = " ALTER COLUMN %s type %s using %[1]s::%s"
const tplAlterNotNull = " ALTER COLUMN %s set not null"

func (p *ParserCfgDDL) checkColumn(col Column, colDefine string, flags []FlagColumn) {

	if len(flags) == 0 {
		return
	}

	typeDef := getNewTypeDef(col, colDefine)
	p.chkAlterBuilder()

	for _, token := range flags {
		switch token {
		// change length
		case ChgLength:
			p.updDLL.WriteString(fmt.Sprintf(tplAlterColumnType, col.Name(), typeDef))

		// change defaults
		case ChgDefault:
			p.updDLL.WriteString(fmt.Sprintf(tplAlterColumnType, col.Name(), typeDef))

		// change type
		case ChgType:
			p.updDLL.WriteString(fmt.Sprintf(" ALTER COLUMN %s drop default, ALTER COLUMN %[1]s type %s using %[1]s::%s", col.Name(), typeDef))

		// change type to Array
		case ChgToArray:
			p.updDLL.WriteString(fmt.Sprintf("ALTER COLUMN %s drop default, ALTER COLUMN %[1]s type %s using array[%[1]s::%[3]s]::%[2]s",
				col.Name(),
				typeDef,
				strings.TrimSuffix(typeDef, "[]"),
			))

		// set not nullable
		case MustNotNull:
			p.updDLL.WriteString(fmt.Sprintf(tplAlterNotNull, col.Name()))

		// set nullable
		case Nullable:
			p.updDLL.WriteString(fmt.Sprintf("ALTER COLUMN %s drop not null", col.Name()))
		}
	}
}

func getNewTypeDef(col Column, colDefine string) string {
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
		if strings.HasPrefix(attr[1], "varying") ||
			(strings.Contains(typeDef, "(") && !strings.Contains(typeDef, ")")) {
			typeDef += " " + attr[1]
		}
	case "double":
		if colType == "money" {
			typeDef = "numeric::" + typeDef
		}
		typeDef += " " + attr[1]
	}
	return typeDef
}

func (p *ParserCfgDDL) addComment(ddl string) bool {
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

func (p *ParserCfgDDL) skipPartition(ddl string) bool {
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

func (p *ParserCfgDDL) updateIndex(ddl string) bool {
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

func (p *ParserCfgDDL) alterColumn(colName string, sAlter ...string) error {
	ddl := fmt.Sprintf(`ALTER TABLE %s %s`, p.Name(), strings.Join(sAlter, ","))

	p.runDDL(ddl)
	switch {
	case p.err == nil:
		logInfo(prefix, p.filename, ddl, p.line)
		p.ReReadColumn(p.DB.ctx, colName)
	case IsErrorForReplace(p.err):
		logs.ErrorLog(p.err, `Field %s.%s, different with define: '%s' %v`, p.Name(), ddl)
	case IsErrorNullValues(p.err):
		defaults := RegDefault.FindStringSubmatch(strings.ToLower(ddl))
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

func (p *ParserCfgDDL) alterTable(ddl string) bool {
	if !strings.HasPrefix(strings.ToLower(ddl), "alter table") {
		return false
	}

	p.runDDL(ddl)

	return true
}
