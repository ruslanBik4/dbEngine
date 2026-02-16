// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"maps"
	"os"
	"path"
	"path/filepath"

	"gopkg.in/yaml.v3"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/generators/go/tpl"

	"github.com/ruslanBik4/logs"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

type CfgCreator struct {
	Dst      string
	Excluded []string
	Imports  []string
	Included []string
}

func LoadCfg(filename string) (cfg *CfgCreator, err error) {
	f, err := os.Open(filename)
	if err != nil {
		logs.ErrorLog(err)
		return nil, err
	}

	defer func() {
		if err := f.Close(); err != nil {
			logs.ErrorLog(err, "close file '%s' failed", filename)
		}
	}()

	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&cfg); err != nil {
		logs.ErrorLog(err, "decoding error")
		return nil, err
	}

	return
}

// Creator is interface for generate go-interface according to DB structures (tables & routines)
type Creator struct {
	*tpl.PackageBuilder
	cfg *CfgCreator
	//Types      map[string]string
}

// NewCreator create with destination directory 'dst'
func NewCreator(DB *dbEngine.DB, cfg *CfgCreator) (*Creator, error) {
	if DB == nil {
		return nil, dbEngine.ErrDBNotFound
	}

	if cfg == nil {
		c, err := LoadCfg("creator.yaml")
		if err != nil {
			return nil, err
		}
		cfg = c
	}

	err := os.Mkdir(cfg.Dst, os.ModePerm)

	if os.IsExist(err) {
		files, err := filepath.Glob(path.Join(cfg.Dst, "*.go"))
		if err != nil {
			logs.ErrorLog(err)
		} else {
			for _, file := range files {
				if err := os.Remove(file); err != nil {
					logs.ErrorLog(err)
				}
			}
		}
	} else if err != nil {
		return nil, errors.Wrap(err, "mkDirAll")
	}

	packagesAsDefault := []string{
		"io",
		"encoding/gob",
		"time",

		"github.com/jackc/pgconn",
		"github.com/jackc/pgtype",
		"golang.org/x/net/context",

		"github.com/ruslanBik4/gotools",
		"github.com/ruslanBik4/logs",
		"github.com/ruslanBik4/dbEngine/dbEngine",
		"github.com/ruslanBik4/dbEngine/dbEngine/psql",
	}

	imports := maps.Collect(func(yield func(string, struct{}) bool) {
		for _, name := range packagesAsDefault {
			if !yield(name, struct{}{}) {
				return
			}
		}
	})

	if _, ok := DB.Types["citext"]; ok {
		imports["bytes"] = struct{}{}
		imports["github.com/jackc/pgx/v4"] = struct{}{}
	}

	return &Creator{
		PackageBuilder: &tpl.PackageBuilder{DB: DB, Imports: imports},
		cfg:            cfg,
	}, nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) MakeInterfaceDB() error {

	f, err := os.Create(path.Join(c.cfg.Dst, "database") + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	defer func(f *os.File) {
		err := f.Close()
		if err != nil {
			logs.ErrorLog(err, "close file '%s' failed", f.Name())
		}
	}(f)
	return c.PrepareDatabase(f)
}

//	func (c *Creator) prepareReturn(r *psql.Routine) (string, string) {
//		toType := psql.UdtNameToType(r.UdtName, nil, nil)
//		sType := typesExt.Basic(toType).String()
//		switch toType {
//		case types.UntypedNil, types.Invalid:
//			sType = c.chkDefineType(r.UdtName)
//			if sType == "" {
//				sType = "*" + strcase.ToCamel(r.UdtName)
//			}
//
//		case types.UntypedFloat:
//			sType = "float64"
//		default:
//		}
//
//		return "res " + sType + ", ", "&res,"
//	}
//
//	func (c *Creator) prepareReturns(r *psql.Routine, name string) (sRecord string, sReturn string, sResult string) {
//		if len(r.Columns()) == 0 {
//			sReturn, sResult = c.prepareReturn(r)
//		}
//		for _, col := range r.Columns() {
//			typeCol, _ := c.ChkTypes(col, name)
//			if typeCol == "" {
//				logs.ErrorLog(dbEngine.NewErrNotFoundType(col.Type(), col.Name()), name)
//			}
//			s := strings.Trim(col.Name(), "_")
//			if s == "type" {
//				s += "_"
//			}
//			sReturn += s + " " + typeCol + ", "
//			sResult += "&" + s + ", "
//		}
//		if len(r.Columns()) > 1 {
//			sRecord = strings.ReplaceAll(strings.TrimSuffix(sResult, ","), "&", "&r.")
//			sResult = fmt.Sprintf(paramsFormat, sResult)
//		}
//		return
//	}
func (c *Creator) prepareParams(r *psql.Routine, name string) (sParams string, sParamsTitle string) {
	// args = make([]any, len(r.Params()))
	for _, param := range r.Params() {
		typeCol, _ := c.ChkTypes(param, name)
		s := strcase.ToLowerCamel(param.Name())
		if param.Default() == nil {
			typeCol = "*" + typeCol
		}
		sParamsTitle += ", " + s + " " + typeCol
		sParams += s + `, `
		// args[i] = param
	}

	if sParams > "" {
		sParams += `
			`
	}
	return
}

// MakeStruct create table interface with Columns operations
func (c *Creator) MakeStruct(table dbEngine.Table) error {
	logs.SetDebug(true)
	f, err := os.Create(path.Join(c.cfg.Dst, table.Name()) + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	defer func() {
		err := f.Close()
		if err != nil {
			logs.ErrorLog(err)
		}
	}()

	c.PrepareTable(table).WriteTable(f, c.Schema)
	tpl.NewColumnType(table).WriteColumnType(f)

	return err
}

//	func (c *Creator) ChkTypes(col dbEngine.Column, propName string) (string, any) {
//		bTypeCol := col.BasicType()
//		defValue := col.Default()
//		if ud := col.UserDefinedType(); ud != nil {
//			for _, tAttr := range ud.Attr {
//				if tAttr.Name == "domain" {
//					return tAttr.Type, defValue
//				}
//			}
//		}
//		typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())
//		isArray := strings.HasPrefix(col.Type(), "_") || strings.HasSuffix(col.Type(), "[]")
//
//		switch {
//		case bTypeCol == types.UnsafePointer:
//			typeCol = "[]byte"
//
//		case (bTypeCol == types.UntypedNil || bTypeCol < 0) && strings.HasPrefix(col.Type(), "any"):
//			typeCol = "any"
//		//too: chk
//		case bTypeCol == types.UntypedNil || bTypeCol < 0:
//			typeCol = c.chkDefineType(col.Type())
//			if typeCol == "" {
//				name, ok := c.ChkDataType(col.Type())
//				if ok {
//					typeCol = strings.TrimPrefix(fmt.Sprintf("%T", name.Value), "*")
//				} else {
//					logs.StatusLog(typeCol, col.Type())
//					typeCol = "sql.RawBytes"
//					c.addImport(moduloSql)
//				}
//			}
//			if a, ok := strings.CutPrefix(typeCol, "*"); ok {
//				c.InitValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("&%s{}", a))
//				defValue = nil
//			}
//
//		case bTypeCol == types.UntypedFloat:
//			switch col.Type() {
//			case "numeric", "decimal":
//				typeCol = "psql.Numeric"
//				if defValue != nil {
//					c.InitValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("psql.NewNumericFromFloat64(%v)", defValue))
//					// prevent finally check default
//					defValue = nil
//				} else {
//					c.InitValues += fmt.Sprintf(initFormat, propName, "psql.NewNumericNull()")
//				}
//			case "_numeric", "_decimal", "numeric[]", "decimal[]":
//				typeCol = "[]psql.Numeric"
//			default:
//				logs.ErrorLog(dbEngine.ErrNotFoundColumn{
//					Table:  propName,
//					Column: col.Type(),
//				}, col)
//			}
//
//		case isArray:
//			typeCol = "[]" + typeCol
//
//		case col.IsNullable():
//			typeCol = "sql.Null" + strcase.ToCamel(typeCol)
//			c.addImport(moduloSql)
//		default:
//		}
//
//		return typeCol, defValue
//	}
