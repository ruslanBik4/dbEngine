// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"flag"
	"fmt"
	"path"

	"github.com/pkg/errors"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/gotools/typesExt"
	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	_go "github.com/ruslanBik4/dbEngine/generators/go"
)

var (
	fDstPath  = flag.String("dst_path", "./db", "path for generated files")
	fCfgPath  = flag.String("src_path", "cfg", "path to cfg DB files")
	fOnlyShow = flag.Bool("read_only", false, "only show DB schema")
)

func main() {

	conn := psql.NewConn(nil, nil, nil)
	dbCfgPath := path.Join(path.Join(*fCfgPath, "DB"), "DB")
	cfgDB := dbEngine.CfgDB{
		Url:       "",
		GetSchema: &struct{}{},
		PathCfg:   &dbCfgPath,
	}
	ctx := context.WithValue(context.Background(), dbEngine.DB_SETTING, cfgDB)
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

	creator, err := _go.NewCreator(*fDstPath, db)
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

	err = creator.MakeInterfaceDB()
	if err != nil {
		logs.ErrorLog(errors.Wrap(err, "make DB interface"))
		return
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
		params := "params: "
		for _, param := range r.Params() {
			params += fmt.Sprintf("%s %s %s\n", param.Name(), param.Type(), typesExt.StringTypeKinds(param.BasicType()))
		}
		logs.StatusLog(key, r.Name(), params)
		if rr := r.Overlay(); rr != nil {
			params := "columns:"
			for _, param := range rr.Params() {
				params += fmt.Sprintf("%s %s %s\n", param.Name(), param.Type(), typesExt.StringTypeKinds(param.BasicType()))
			}
			logs.StatusLog("Overlay", key, rr.Name(), params)
		}
	}
}
