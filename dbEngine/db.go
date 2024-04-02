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
	"strings"
	"sync"
	"time"

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
	Cfg           map[string]any
	Conn          Connection
	ctx           context.Context
	Name          string
	Schema        string
	Tables        map[string]Table
	Types         map[string]Types
	Routines      map[string]Routine
	FuncsReplaced []string
	FuncsAdded    []string
	readTables    map[string][]string
	DbSet         map[string]*string
}

// NewDB create new DB instance & performs something migrations
func NewDB(ctx context.Context, conn Connection) (*DB, error) {
	db := &DB{
		Conn:       conn,
		ctx:        ctx,
		readTables: map[string][]string{},
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
			"types", "table", "view", "func",
		}
	)

	migrationParts := map[string]fs.WalkDirFunc{
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

	logs.StatusLog("Create or replace functions on DB: '%s'", strings.Join(db.FuncsAdded, "', '"))
	if len(db.FuncsReplaced) > 0 {
		logs.StatusLog("Modify func on DB : '%s'", strings.Join(db.FuncsReplaced, "', '"))
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

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		tableName := strings.TrimSuffix(fileName, ext)
		ddl, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		table, ok := db.Tables[tableName]
		if !ok {
			err = db.Conn.ExecDDL(db.ctx, string(ddl))
			switch {
			case err == nil:
				table = db.Conn.NewTable(tableName, "table")
				err = table.GetColumns(db.ctx)
				if err == nil {
					db.Tables[tableName] = table
					logs.StatusLog("New table added to DB", tableName)
				}

				if rel, ok := db.readTables[tableName]; ok {
					for _, each := range rel {
						err = db.ReadTableSQL(each, info, err)
						if err != nil {
							return err
						}
					}
					return nil
				}

			case IsErrorAlreadyExists(err) && !strings.Contains(err.Error(), tableName):
				logs.ErrorLog(err, "Already exists - "+tableName+" but it don't found on schema")

			case IsErrorDoesNotExists(err):
				if errParts := regRelationNotExist.FindStringSubmatch(err.Error()); len(errParts) > 0 {
					if val, ok := db.readTables[errParts[1]]; ok {
						db.readTables[errParts[1]] = append(val, path)
					} else {
						db.readTables[errParts[1]] = []string{path}
					}
				} else {
					logs.ErrorLog(err, "performs not implement")
				}

			default:
				logs.ErrorLog(err, "During create- "+tableName)
			}

			return nil
		}

		return NewParserTableDDL(table, db).Parse(string(ddl))

	default:
		return nil
	}
}

// ReadViewSQL performs ddl script for view
func (db *DB) ReadViewSQL(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		tableName := strings.TrimSuffix(fileName, ext)
		ddl, err := os.ReadFile(path)
		if err != nil {
			return err
		}

		table, ok := db.Tables[tableName]
		if !ok {
			err = db.Conn.ExecDDL(db.ctx, string(ddl))
			if err == nil {
				table = db.Conn.NewTable(tableName, "VIEW")
				err = table.GetColumns(db.ctx)
				if err == nil {
					db.Tables[tableName] = table
					logs.StatusLog("New view added to DB", tableName)
				}

				return err

			} else if !IsErrorAlreadyExists(err) {
				logs.ErrorLog(err, "table - "+tableName)
				return err
			}
		}

		return NewParserTableDDL(table, db).Parse(string(ddl))

	default:
		return nil
	}
}

func (db *DB) readAndReplaceTypes(path string, info os.DirEntry, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	switch ext := filepath.Ext(path); ext {
	case ".ddl":
		fileName := filepath.Base(path)
		typeName := strings.TrimSuffix(fileName, ext)
		if _, ok := db.Types[typeName]; ok {
			return nil
		}

		ddl, err := os.ReadFile(path)
		if err == nil {
			ddlType := gotools.BytesToString(ddl)
			// this local err - not return for parent method
			err := db.Conn.ExecDDL(db.ctx, ddlType)
			if err == nil {
				logs.StatusLog("New types added to DB", ddlType)
				return nil
			}

			if IsErrorAlreadyExists(err) {
				err = db.alterType(fileName, typeName, strings.ToLower(strings.Replace(ddlType, "\n", "", -1)))
			} else if IsErrorForReplace(err) {
				logError(err, ddlType, fileName)
				err = nil
			}

			if err != nil {
				logError(err, ddlType, fileName)
			}
		}
		return err

	default:
		return nil
	}
}

var regTypeAttr = regexp.MustCompile(`create\s+type\s+\w+\s+as\s*\((?P<builderOpts>(\s*\w+\s+\w*\s*[\w\[\]()]*,?)+)\s*\);`)

// var regFieldAttr = regexp.MustCompile(`(\w+)\s+([\w()\[\]\s]+)`)

func (db *DB) alterType(fileName, typeName, ddl string) error {
	fields := regTypeAttr.FindStringSubmatch(ddl)

	for i, name := range regTypeAttr.SubexpNames() {
		if name == "builderOpts" && (i < len(fields)) {

			nameFields := strings.Split(fields[i], ",")
			for _, name := range nameFields {

				ddlAlterType := "alter type " + typeName
				ddlType := ddlAlterType + " add attribute " + name
				err := db.Conn.ExecDDL(db.ctx, ddlType)
				if err == nil {
					logInfo(prefix, fileName, ddlType, 1)
				} else if IsErrorAlreadyExists(err) {
					p := strings.Split(strings.TrimSpace(name), " ")
					if len(p) < 2 {
						err = ErrWrongType{
							Name:     name,
							TypeName: typeName,
							Attr:     p[0],
						}
					} else {
						ddlAlterType += " alter attribute " + p[0] + " SET DATA TYPE "
						for _, val := range p[1:] {
							ddlAlterType += " " + val
						}
						err = db.Conn.ExecDDL(db.ctx, ddlAlterType)
						if err == nil {
							logInfo(prefix, fileName, ddlAlterType, 1)
						}
					}
				}

				if err != nil {
					return err
				}
			}
		}
	}

	return nil
}

var (
	regFuncTitle = regexp.MustCompile(`(function|procedure)\s+\w+\s*\(([^()]+(\(\d+\))?)*\)`)
	regFuncDef   = regexp.MustCompile(`\sdefault\s+[^,)]+`)
)

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
			for _, funcName := range regFuncTitle.FindAllString(strings.ToLower(ddlSQL), -1) {
				dropSQL := "DROP " + regFuncDef.ReplaceAllString(funcName, "")
				logs.DebugLog(dropSQL)
				err = db.Conn.ExecDDL(context.TODO(), dropSQL)
				if err != nil {
					break
				}
			}

			if err == nil {
				err = db.Conn.ExecDDL(context.TODO(), ddlSQL)
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

func timeLogFormat() string {
	hh, mm, ss := time.Now().Clock()
	return fmt.Sprintf("%.2d:%.2d:%.2d", hh, mm, ss)
}
