// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"strings"
	"sync"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/gotools"
	"github.com/ruslanBik4/logs"
)

type CfgCreatorDB struct {
	RecreateMaterView *struct{}
}

// CfgDB consist of setting for creating new DB
type CfgDB struct {
	Url        string
	GetSchema  *struct{}
	CfgCreator *CfgCreatorDB
	// obsolete - change on CfgCreatorDB properties
	Excluded []string
	Included []string
	PathCfg  *string
	TestInit *string
}

// TypeCfgDB is type for context values
type TypeCfgDB string

// DB name & schema
type DB struct {
	sync.RWMutex
	Cfg            map[string]any
	Conn           Connection
	ctx            context.Context
	Name           string
	Schema         string
	Tables         map[string]Table
	Types          map[string]Types
	Routines       map[string]Routine
	FuncsReplaced  []string
	FuncsAdded     []string
	relationTables map[string][]string
	DbSet          map[string]*string
}

// NewDB create new DB instance & performs something migrations
func NewDB(ctx context.Context, conn Connection) (*DB, error) {
	db := &DB{
		Conn:           conn,
		ctx:            ctx,
		relationTables: map[string][]string{},
	}

	if cfg, ok := ctx.Value(DB_SETTING).(CfgDB); ok {
		err := conn.InitConn(ctx, cfg.Url)
		if err != nil {
			return nil, err
		}

		if cfg.GetSchema != nil {
			db.DbSet, db.Tables, db.Routines, db.Types, err = conn.GetSchema(ctx, &cfg)
			if err != nil {
				return nil, err
			}

			db.Name = *db.DbSet["db_name"]
			db.Schema = *db.DbSet["db_schema"]
		}

		if cfg.CfgCreator != nil {
			if cfg.CfgCreator.RecreateMaterView != nil {
				db.Cfg[string(RECREATE_MATERIAZE_VIEW)] = true
			}
		}
		if cfg.PathCfg != nil {

			err := db.readCfg(ctx, &cfg)
			if err != nil {
				return nil, err
			}

		}

		if cfg.TestInit != nil {
			db.runTestInitScript(*cfg.TestInit)
		}
	}

	return db, nil
}

func (db *DB) readCfg(ctx context.Context, cfg *CfgDB) error {
	var (
		migrationOrder = []string{
			"roles", "types", "table", "view", "func",
		}
	)

	migrationParts := map[string]fs.WalkDirFunc{
		"roles": db.readAndReplaceRoles,
		"types": db.readAndReplaceTypes,
		"table": db.ReadTableSQL,
		"view":  db.ReadViewSQL,
		"func":  db.readAndReplaceFunc,
	}

	for _, name := range migrationOrder {
		err := filepath.WalkDir(filepath.Join(*cfg.PathCfg, name), migrationParts[name])
		if err != nil {
			return errors.Wrap(err, "migration "+name)
		}
	}

	logInfo(preDB_CONFIG, "Create or replace functions on DB", strings.Join(db.FuncsAdded, ","), len(db.FuncsAdded))
	if len(db.FuncsReplaced) > 0 {
		logInfo(preDB_CONFIG, "Modify func on DB", strings.Join(db.FuncsReplaced, ","), len(db.FuncsReplaced))
	}

	var err error
	_, db.Tables, db.Routines, db.Types, err = db.Conn.GetSchema(ctx, cfg)
	if err != nil {
		return err
	}

	return nil
}

func (db *DB) runTestInitScript(name string) {
	if name == "" {
		name = filepath.Join("cfg/DB", "test_init.ddl")
	}

	ddl, err := os.ReadFile(name)
	if err != nil {
		logs.ErrorLog(err, "db.Conn.ExecDDL")
	} else {

		err = db.Conn.ExecDDL(context.TODO(), string(ddl))
		if err != nil {
			logs.ErrorLog(err, "db.Conn.ExecDDL")
		}
	}
}

// ReadTableSQL performs ddl script for tables
func (db *DB) ReadTableSQL(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	return db.syncTableDDL(path, "table")
}

func (db *DB) syncTableDDL(path string, tType string) error {
	switch ext := filepath.Ext(path); ext {
	case ".ddl":
		fileName := filepath.Base(path)
		tableName := strings.TrimSuffix(fileName, ext)
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		ddl := gotools.BytesToString(b)
		table, ok := db.Tables[tableName]
		if ok {
			return NewParserCfgDDL(db, table).Parse(ddl)
		}

		//check tables dependencies
		lower := strings.ToLower(ddl)
		if submatch := regForeignIndex.FindStringSubmatch(lower); len(submatch) > 0 {
			parent := submatch[regForeignIndex.SubexpIndex("fTable")]
			if _, ok := db.Tables[parent]; !ok {
				db.addRelationTable(path, parent)
				logWarning("TABLES", path, "wait for parent:"+parent, 0)
				return nil
			}
		}
		return db.createTable(path, ddl, tableName, tType)

	default:
		return nil
	}
}

func (db *DB) createTable(path, ddl, tableName, tType string) error {

	fileName := filepath.Base(path)
	switch err := db.Conn.ExecDDL(db.ctx, ddl); {
	case err == nil:
		table := db.Conn.NewTable(tableName, "table")
		err = table.GetColumns(db.ctx, nil)
		if err != nil {
			return err
		}
		db.Tables[tableName] = table
		logInfo(preDB_CONFIG, fileName, fmt.Sprintf("New %s added to DB: %s", tType, tableName), 1)

		// create all relations tables
		if rel, ok := db.relationTables[tableName]; ok {
			for _, relPath := range rel {
				err = db.syncTableDDL(relPath, "")
				if err != nil {
					return err
				}
			}
			return nil
		}

	case IsErrorAlreadyExists(err) && !strings.Contains(err.Error(), tableName):
		logError(err, "Already exists - "+tableName+" but it don't found on schema", fileName)

	//	DDL has relation into non-creating tables - save path for creating after relations tables
	case IsErrorDoesNotExists(err):
		if errParts := regRelationNotExist.FindStringSubmatch(err.Error()); len(errParts) > 0 {
			db.addRelationTable(path, errParts[1])
		} else {
			logs.ErrorLog(err, "performs not implement")
		}

	default:
		logError(err, "During create- "+tableName, fileName)
	}

	return nil
}

func (db *DB) addRelationTable(path string, tableName string) {
	if val, ok := db.relationTables[tableName]; ok {
		db.relationTables[tableName] = append(val, path)
	} else {
		db.relationTables[tableName] = []string{path}
	}
	logs.DebugLog(db.relationTables)
}

// ReadViewSQL performs ddl script for view
func (db *DB) ReadViewSQL(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	return db.syncTableDDL(path, "view")
}

func (db *DB) readAndReplaceRoles(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	switch ext := filepath.Ext(path); ext {
	case ".ddl", ".sql", ".role":
		ddl, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ddlType := gotools.BytesToString(ddl)
		fileName := filepath.Base(path)
		roleName := strings.ToLower(strings.TrimSuffix(fileName, ext))
		// this local err - not return for parent method
		err = db.Conn.ExecDDL(db.ctx, ddlType)
		if IsErrorAlreadyExists(err) {
		} else if err != nil {
			logError(err, ddlType, fileName)
			return err
		} else {
			logs.StatusLog("New role added to DB", roleName)
		}
	}
	return nil
}

func (db *DB) readAndReplaceTypes(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	switch ext := filepath.Ext(path); ext {
	case ".ddl":
		ddl, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		ddlType := gotools.BytesToString(ddl)
		fileName := filepath.Base(path)
		typeName := strings.ToLower(strings.TrimSuffix(fileName, ext))
		if t, ok := db.Types[typeName]; ok {
			return db.alterType(&t, fileName, typeName, strings.Replace(ddlType, "\n", "", -1))
		}

		// this local err - not return for parent method
		err = db.Conn.ExecDDL(db.ctx, ddlType)
		switch {
		case err == nil:
			logs.StatusLog("New types added to DB", typeName)
			db.Types[typeName] = Types{
				Id:         0,
				Name:       typeName,
				Type:       0,
				Attr:       nil,
				Enumerates: nil,
			}

		case IsErrorAlreadyExists(err):
			return db.alterType(nil, fileName, typeName, strings.ToLower(strings.Replace(ddlType, "\n", "", -1)))
		case IsErrorForReplace(err):
			logError(err, ddlType, fileName)
		case err != nil:
			logError(err, ddlType, fileName)
			return err
		}
		return nil

	default:
		return nil
	}
}

var regTypeAttr = regexp.MustCompile(`(?i)create\s+type\s+\w+\s+as\s*\((?P<builderOpts>(\s*\w+\s+\w*\s*[\w\[\]()]*,?)+)\s*\);`)
var regTypeEnum = regexp.MustCompile(`(?i)create\s+type\s+\w+\s+as\s*enum\s*\(\s*(?P<builderOpts>('[^,)]+',?\s*)+)\s*\);`)

// var regFieldAttr = regexp.MustCompile(`(\w+)\s+([\w()\[\]\s]+)`)

func (db *DB) alterType(t *Types, fileName, typeName, ddl string) error {

	if t == nil {
		logs.ErrorLog(errors.New("alter without known DB type!"))
		return nil
	}

	if fields := regTypeAttr.FindStringSubmatch(ddl); len(fields) > 0 {
		return db.alterCompositeType(t, fileName, typeName, fields)
	} else if enumerates := regTypeEnum.FindStringSubmatch(ddl); len(enumerates) > 0 {
		return db.alterEnumType(t, fileName, typeName, enumerates)
	}

	return nil
}

const addEnumBefore = ` ADD VALUE %s BEFORE '%s'`
const addEnumAfter = ` ADD VALUE %s AFTER '%s'`

func (db *DB) alterEnumType(t *Types, fileName, typeName string, enumerates []string) error {
	ddlType := "alter type " + typeName
	for i, name := range regTypeAttr.SubexpNames() {
		if name == "builderOpts" && (i < len(enumerates)) {
			nameFields := strings.Split(enumerates[i], ",")
			offset := 0
			for ord, enum := range nameFields {
				name := strings.ReplaceAll(strings.TrimSpace(enum), "''", "'")
				if slices.Index(t.Enumerates, strings.Trim(name, "'")) < 0 {
					if ord == 0 {
						ddlAddAttr := ddlType + fmt.Sprintf(addEnumBefore, name, t.Enumerates[0])
						if err := db.Conn.ExecDDL(db.ctx, ddlAddAttr); err != nil {
							return err
						}
						logInfo(preDB_CONFIG, fileName, ddlAddAttr, 1)
					} else if ord-offset < len(t.Enumerates) {
						logs.StatusLog(ddlType+fmt.Sprintf(addEnumAfter, name, t.Enumerates[ord-offset-1]), ord-offset, len(t.Enumerates))
					} else {
						logs.StatusLog(ddlType+fmt.Sprintf(addEnumAfter, name, t.Enumerates[len(t.Enumerates)-1]), ord+offset, len(t.Enumerates))
					}
					offset++
				}
			}
		}
	}
	return nil
}

func (db *DB) alterCompositeType(t *Types, fileName, typeName string, fields []string) error {
	ddlType := "alter type " + typeName
	for i, name := range regTypeAttr.SubexpNames() {
		if name == "builderOpts" && (i < len(fields)) {
			nameFields := strings.Split(fields[i], ",")
			for _, name := range nameFields {
				p := strings.Split(strings.TrimSpace(name), " ")
				//todo aad "Name of wrong" performing
				attrName := strings.ToLower(strings.TrimSpace(p[0]))
				if len(p) < 2 {
					return ErrWrongType{
						Name:     name,
						TypeName: typeName,
						Attr:     attrName,
					}
				}

				i := slices.IndexFunc(t.Attr, func(attr TypesAttr) bool {
					return attr.Name == attrName
				})
				newType := strings.TrimSpace(strings.Join(p[1:], " "))
				if i == -1 {
					ddlAddAttr := ddlType + " add attribute " + name
					err := db.Conn.ExecDDL(db.ctx, ddlAddAttr)
					if err == nil {
						logInfo(preDB_CONFIG, fileName, ddlAddAttr, 1)
					} else if IsErrorAlreadyExists(err) {
						logs.ErrorLog(err)
					}
					continue
				}
				chkAttr := t.Attr[i].CheckAttr(newType)
				if len(chkAttr) == 0 {
					continue
				}

				ddlAlter := ddlType + " alter attribute " + attrName
				for i, flag := range chkAttr {
					logs.StatusLog("%d. %s", i, flag)
					switch flag {
					case MustNotNull:
						ddlAlter += " SET NOT NULL "
					case Nullable:
						ddlAlter += " DROP NOT NULL "
					case ChgType:
						ddlAlter += " SET DATA TYPE " + newType
					case ChgLength:
						ddlAlter += " SET DATA TYPE " + newType
					case ChgToArray:
					default:
						logs.ErrorLog(fmt.Errorf("unhandled default case: %v", flag), fileName)
					}
				}
				err := db.Conn.ExecDDL(db.ctx, ddlAlter)
				if err != nil {
					logs.ErrorLog(err, ddlAlter)
					return err
				}
				logInfo(preDB_CONFIG, fileName, ddlAlter, 1)

				logs.StatusLog(t.Attr[i], t.Attr[i].Column.Type(), attrName, newType)

			}
		}
	}
	return nil
}

var (
	regRoutineTitle = regexp.MustCompile(`(function|procedure)\s+\w+\s*\(([^()]+(\(\d+\))?)*\)`)
	regRoutineDef   = regexp.MustCompile(`\sdefault\s+[^,)]+`)
)

// todo split first row to get procedure title
func (db *DB) readAndReplaceFunc(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		funcName := strings.TrimSuffix(fileName, ext)
		ddl, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		ddlSQL := string(ddl)
		// this local err - not return for parent method
		err = db.Conn.ExecDDL(context.TODO(), ddlSQL)
		if err == nil {
			db.FuncsAdded = append(db.FuncsAdded, funcName)
		} else if IsErrorAlreadyExists(err) {
			err = nil
		} else if IsErrorForReplace(err) {
			err = nil
			for _, funcName := range regRoutineTitle.FindAllString(strings.ToLower(ddlSQL), -1) {
				dropSQL := "DROP " + regRoutineDef.ReplaceAllString(funcName, "")
				logs.DebugLog(dropSQL)
				err = db.Conn.ExecDDL(db.ctx, dropSQL)
				if err != nil {
					break
				}
			}

			if err == nil {
				err = db.Conn.ExecDDL(db.ctx, ddlSQL)
				if err == nil {
					db.FuncsReplaced = append(db.FuncsReplaced, funcName)
				}
			}
		}

		if err != nil {
			logError(err, ddlSQL, fileName)
			err = nil
		}

		return err

	default:
		return nil
	}
}

func logInfo(prefix, fileName, msg string, line int) {
	logs.CustomLog(logs.NOTICE, prefix, fileName, line, msg, logs.FgInfo)
}

func logWarning(prefix, fileName, msg string, line int) {
	logs.CustomLog(logs.WARNING, prefix, fileName, line, msg, logs.FgInfo)
}
