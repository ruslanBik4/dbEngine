// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package dbEngine

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/httpgo/logs"
	"golang.org/x/net/context"
)

type DB struct {
	Cfg      map[string]interface{}
	Conn     Connection
	Tables   map[string]Table
	Types    map[string]Types
	Routines map[string]Routine
	modFuncs []string
	newFuncs []string
}

func NewDB(ctx context.Context, conn Connection) (*DB, error) {
	db := &DB{Conn: conn}
	if dbUrl, ok := ctx.Value("dbURL").(string); ok {
		logs.DebugLog("init conn with url - ", dbUrl)
		err := conn.InitConn(ctx, dbUrl)
		if err != nil {
			return nil, errors.Wrap(err, "initConn")
		}

		if doRead, ok := ctx.Value("fillSchema").(bool); ok && doRead {
			db.Tables, db.Routines, db.Types, err = conn.GetSchema(ctx)
			if err != nil {
				return nil, errors.Wrap(err, "initConn")
			}
			if doRead, ok = ctx.Value("makeStruct").(bool); ok && doRead {

			}
		}
		if mPath, ok := ctx.Value("migration").(string); ok {
			err = filepath.Walk(filepath.Join(mPath, "types"), db.readAndReplaceTypes)
			if err != nil {
				return nil, errors.Wrap(err, "migration types")
			}

			err = filepath.Walk(filepath.Join(mPath, "table"), db.ReadTableSQL)
			if err != nil {
				return nil, errors.Wrap(err, "migration tables")
			}

			err = filepath.Walk(filepath.Join(mPath, "func"), db.readAndReplaceFunc)
			if err != nil {
				return nil, errors.Wrap(err, "migration func")
			}

			logs.StatusLog("New func add to DB: '%s'", strings.Join(db.newFuncs, "', '"))
			if len(db.modFuncs) > 0 {
				logs.StatusLog("Modify func in DB : '%s'", strings.Join(db.modFuncs, "', '"))
			}

			db.Routines, err = conn.GetRoutines(ctx)
			if err != nil {
				logs.ErrorLog(err, "refresh func")
			}
		}
	}

	return db, nil
}

func (db *DB) ReadTableSQL(path string, info os.FileInfo, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		tableName := strings.TrimSuffix(fileName, ext)
		ddl, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		table, ok := db.Tables[tableName]
		if !ok {
			err = db.Conn.ExecDDL(context.TODO(), string(ddl))
			if err == nil {
				table = db.Conn.NewTable(tableName, "table")
				err = table.GetColumns(context.TODO())
				if err == nil {
					db.Tables[tableName] = table
					logs.StatusLog("New table add to DB", tableName)
				}
				return err
			} else {
				logs.ErrorLog(err, "table - "+tableName)
			}
		} else {
			return NewParserTableDDL(table, db).Parse(string(ddl))
		}

	default:
		return nil
	}

	return err
}

var regTypeAttr = regexp.MustCompile(`create\s+type\s+\w+\s+as\s*\((?P<fields>(\s*\w+\s+\w*\s*[\w\[\]()]*,?)+)\s*\);`)

//var regFieldAttr = regexp.MustCompile(`(\w+)\s+([\w()\[\]\s]+)`)

func (db *DB) readAndReplaceTypes(path string, info os.FileInfo, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		typeName := strings.TrimSuffix(fileName, ext)
		if _, ok := db.Types[typeName]; ok {
			return nil
		}

		ddl, err := ioutil.ReadFile(path)
		if err == nil {
			ddlType := string(ddl)
			// this local err - not return for parent method
			err := db.Conn.ExecDDL(context.TODO(), ddlType)
			if err == nil {
				logs.StatusLog("New types add to DB", typeName)
			} else if isErrorAlreadyExists(err) {
				ddl := strings.ToLower(string(bytes.Replace(ddl, []byte("\n"), []byte(""), -1)))
				fields := regTypeAttr.FindStringSubmatch(ddl)
				err = nil

				for i, name := range regTypeAttr.SubexpNames() {
					if name == "fields" && (i < len(fields)) {

						nameFields := strings.Split(fields[i], ",")
						for _, name := range nameFields {

							ddlAlterType := "alter type " + typeName
							ddlType := ddlAlterType + " add attribute " + name
							err = db.Conn.ExecDDL(context.TODO(), ddlType)
							if err == nil {
								logs.StatusLog("[DB CONFIG]", ddlType)
							} else if isErrorAlreadyExists(err) {
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
									err = db.Conn.ExecDDL(context.TODO(), ddlAlterType)
									if err == nil {
										logs.StatusLog("[DB CONFIG]", ddlAlterType)
									}
								}
							}

							if err != nil {
								logs.ErrorLog(err, name, ddlAlterType)
							}
						}
					}
				}
			} else if isErrorForReplace(err) {
				logs.ErrorLog(err, ddl)
				err = nil
			}

			if err != nil {
				logError(err, ddlType, fileName)
			}

		}
	default:
		return nil
	}

	return err
}

var regFuncTitle = regexp.MustCompile(`(function|procedure)\s+\w+\s*\(([^()]+(\(\d+\))?)*\)`)

func (db *DB) readAndReplaceFunc(path string, info os.FileInfo, err error) error {
	if (err != nil) || ((info != nil) && info.IsDir()) {
		return nil
	}

	ext := filepath.Ext(path)
	switch ext {
	case ".ddl":
		fileName := filepath.Base(path)
		funcName := strings.TrimSuffix(fileName, ext)
		ddl, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		ddlSQL := string(ddl)
		// this local err - not return for parent method
		err = db.Conn.ExecDDL(context.TODO(), ddlSQL)
		if err == nil {
			db.newFuncs = append(db.newFuncs, funcName)
		} else if isErrorAlreadyExists(err) {
			err = nil
		} else if isErrorForReplace(err) {
			err = nil
			for _, funcName := range regFuncTitle.FindAllString(strings.ToLower(ddlSQL), -1) {
				dropSQL := "DROP " + strings.Replace(funcName, "default null", "", -1)
				logs.DebugLog(dropSQL)
				err = db.Conn.ExecDDL(context.TODO(), dropSQL)
				if err != nil {
					break
				}
			}

			if err == nil {
				err = db.Conn.ExecDDL(context.TODO(), ddlSQL)
				if err == nil {
					db.modFuncs = append(db.modFuncs, funcName)
				}
			}
		}

		if err != nil {
			logError(err, ddlSQL, fileName)
			err = nil
		}

	default:
		// logs.DebugLog(" unknow type file = %+s", path)
		return nil
	}

	return err
}

func logError(err error, ddlSQL string, fileName string) {
	pgErr, ok := err.(*pgconn.PgError)
	if ok && pgErr.Position > 0 {
		line := strings.Count(ddlSQL[:pgErr.Position-1], "\n") + 1
		fmt.Printf("\033[%d;1m%s\033[0m %v:%d: %s %#v\n", 35, "[[ERROR]]", fileName, line, pgErr.Message, pgErr)
	} else {
		logs.ErrorLog(err, prefix, fileName)
	}
}
