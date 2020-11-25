// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"

	"github.com/pkg/errors"
	"github.com/ruslanBik4/dbEngine/typesExt"
	"github.com/ruslanBik4/logs"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	_go "github.com/ruslanBik4/dbEngine/generators/go"
)

var (
	fDstPath  = flag.String("dst_path", "./db", "path for generated files")
	fCfgPath  = flag.String("src_path", "cfg/DB", "path to cfg DB files")
	fURL      = flag.String("db_url", "", "URL for DB connection")
	fOnlyShow = flag.Bool("read_only", true, "only show DB schema")
)

func main() {

	conn := psql.NewConn(nil, nil, nil)
	ctx := context.WithValue(context.Background(), "dbURL", *fURL)
	ctx = context.WithValue(ctx, "fillSchema", true)
	ctx = context.WithValue(ctx, "migration", *fCfgPath)
	db, err := dbEngine.NewDB(ctx, conn)
	if err != nil {
		logs.ErrorLog(err, "dbEngine.NewDB")
		return
	}

	if *fOnlyShow {
		printTables(db)
		printRoutines(db)
		return
	}

	creator, err := _go.NewCreator(*fDstPath)
	if err != nil {
		logs.ErrorLog(errors.Wrap(err, "NewCreator"))
		return
	}

	for name, table := range db.Tables {
		err = creator.MakeStruct(table)
		if err != nil {
			logs.ErrorLog(errors.Wrap(err, "makeStruct - "+name))
		}
	}
}

func printTables(db *dbEngine.DB) {
	logs.StatusLog("list tables:")
	for key, table := range db.Tables {
		logs.StatusLog(key, table.Name())
		for _, col := range table.Columns() {
			logs.StatusLog("%s %s %s %v", col.Name(), col.Type(), typesExt.StringTypeKinds(col.BasicType()), col.Default())
		}
	}
}

func printRoutines(db *dbEngine.DB) {
	logs.StatusLog("list routines:")
	for key, r := range db.Routines {
		logs.StatusLog(key, r.Name())
		for _, param := range r.Params() {
			logs.StatusLog("%s %s %s", param.Name(), param.Type(), typesExt.StringTypeKinds(param.BasicType()))
		}
	}
}
