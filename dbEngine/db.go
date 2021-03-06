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
	"time"

	"github.com/jackc/pgconn"
	"github.com/pkg/errors"
	"github.com/ruslanBik4/logs"
	"golang.org/x/net/context"
)

// todo add DB name & schema
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
			return nil, err
		}

		if doRead, ok := ctx.Value("fillSchema").(bool); ok && doRead {
			db.Tables, db.Routines, db.Types, err = conn.GetSchema(ctx)
			if err != nil {
				return nil, err
			}
			if doRead, ok = ctx.Value("makeStruct").(bool); ok && doRead {

			}
		}
		if mPath, ok := ctx.Value(DB_MIGRATION).(string); ok {
			err = filepath.Walk(filepath.Join(mPath, "types"), db.readAndReplaceTypes)
			if err != nil {
				return nil, errors.Wrap(err, "migration types")
			}

			err = filepath.Walk(filepath.Join(mPath, "table"), db.ReadTableSQL)
			if err != nil {
				return nil, errors.Wrap(err, "migration tables")
			}

			err = filepath.Walk(filepath.Join(mPath, "view"), db.ReadViewSQL)
			if err != nil {
				return nil, errors.Wrap(err, "migration views")
			}

			err = filepath.Walk(filepath.Join(mPath, "func"), db.readAndReplaceFunc)
			if err != nil {
				return nil, errors.Wrap(err, "migration func")
			}

			logs.StatusLog("New func add to DB: '%s'", strings.Join(db.newFuncs, "', '"))
			if len(db.modFuncs) > 0 {
				logs.StatusLog("Modify func in DB : '%s'", strings.Join(db.modFuncs, "', '"))
			}

			db.Tables, db.Routines, db.Types, err = conn.GetSchema(ctx)
			if err != nil {
				return nil, err
			}

			// db.Routines, err = conn.GetRoutines(ctx)
			// if err != nil {
			// 	logs.ErrorLog(err, "refresh func")
			// }
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
			switch {
			case err == nil:
				table = db.Conn.NewTable(tableName, "table")
				err = table.GetColumns(context.TODO())
				if err == nil {
					db.Tables[tableName] = table
					logs.StatusLog("New table add to DB", tableName)
				}

			case !IsErrorAlreadyExists(err) || !strings.Contains(err.Error(), tableName):
				logs.ErrorLog(err, "table - "+tableName)

			default:
				logs.ErrorLog(err, "Already exists - "+tableName+" but it don't found on schema")
			}

			return err
		}

		return NewParserTableDDL(table, db).Parse(string(ddl))

	default:
		return nil
	}
}

func (db *DB) ReadViewSQL(path string, info os.FileInfo, err error) error {
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

			} else if !IsErrorAlreadyExists(err) {
				logs.ErrorLog(err, "table - "+tableName)
				return err
			}
			// 	todo: add table new
		}

		return NewParserTableDDL(table, db).Parse(string(ddl))

	default:
		return nil
	}
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
				return nil
			}

			if IsErrorAlreadyExists(err) {
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
								logs.StatusLog(prefix, ddlType)
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
									err = db.Conn.ExecDDL(context.TODO(), ddlAlterType)
									if err == nil {
										logs.StatusLog(prefix, ddlAlterType)
									}
								}
							}

							if err != nil {
								logs.ErrorLog(err, name, ddlAlterType)
							}
						}
					}
				}
			} else if IsErrorForReplace(err) {
				logError(err, ddlType, fileName)
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

var (
	regFuncTitle = regexp.MustCompile(`(function|procedure)\s+\w+\s*\(([^()]+(\(\d+\))?)*\)`)
	regFuncDef   = regexp.MustCompile(`\sdefault\s+[^,)]+`)
)

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
					db.modFuncs = append(db.modFuncs, funcName)
				}
			}
		}

		if err != nil {
			logError(err, ddlSQL, fileName)
			err = nil
		}

	default:
		return nil
	}

	return err
}

func logError(err error, ddlSQL string, fileName string) {
	pgErr, ok := err.(*pgconn.PgError)
	if ok {
		pos := pgErr.Position - 1
		if pos < 0 {
			pos = 0
		}
		line := strings.Count(ddlSQL[:pos], "\n") + 1
		printError(fileName, line, pgErr.Message, pgErr)
	} else if e, ok := err.(*ErrUnknownSql); ok {
		printError(fileName, e.Line, e.Msg, e)
	} else {
		printError(fileName, 1, prefix, err)
	}
}

func printError(fileName string, line int, msg string, err error) {
	// todo mv to logs
	fmt.Printf("[[%s%d;1m%s%s]]%s %s:%d: %s %#v\n",
		logs.LogPutColor,
		33, "ERROR", logs.LogEndColor, timeLogFormat(), fileName, line, msg, err)
}

func logInfo(prefix, fileName, msg string, line int) {
	// todo mv to logs
	fmt.Printf("[[%s%d;1m%s%s]]%s %s:%d: %s\n",
		logs.LogPutColor, 30, prefix, logs.LogEndColor, timeLogFormat(), fileName, line, msg)
}

func timeLogFormat() string {
	hh, mm, ss := time.Now().Clock()
	return fmt.Sprintf("%.2d:%.2d:%.2d", hh, mm, ss)
}
