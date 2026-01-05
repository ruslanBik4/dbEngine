// Copyright 2020 Author: Ruslan Bikchentaev. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package _go

import (
	"context"
	"fmt"
	"go/types"
	"maps"
	"os"
	"path"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/jackc/pgtype"
	"gopkg.in/yaml.v3"

	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/dbEngine/generators/go/tpl"
	"github.com/ruslanBik4/gotools/typesExt"

	"github.com/iancoleman/strcase"
	"github.com/pkg/errors"

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
	*dbEngine.DB
	cfg        *CfgCreator
	Types      map[string]string
	imports    map[string]struct{}
	initValues string
	types      []string
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
		"errors",
		"fmt",
		"time",
		"strings",

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

	_, ok := DB.Types["citext"]
	if ok {
		imports["bytes"] = struct{}{}
		imports["github.com/jackc/pgx/v4"] = struct{}{}
	}

	return &Creator{
		DB:      DB,
		cfg:     cfg,
		imports: imports,
	}, nil
}

// makeDBUsersTypes create interface of DB
func (c *Creator) makeDBUsersTypes() error {
	for tName, t := range c.DB.Types {
		for i, tAttr := range t.Attr {
			name := tAttr.Name
			ud := &t
			if tAttr.Name == "domain" {
				logs.StatusLog("%s, %c %v", tName, t.Type, tAttr)
				ud = nil
			}
			typeCol, _ := c.chkTypes(
				&psql.Column{
					UdtName:     tAttr.Type,
					DataType:    tAttr.Type,
					UserDefined: ud,
				},
				strcase.ToCamel(name))
			if typeCol == "" {
				logs.ErrorLog(dbEngine.NewErrNotFoundType(name, tAttr.Type), tAttr)
			}
			tAttr.Type = typeCol
			t.Attr[i] = tAttr
			if len(t.Enumerates) == 0 {
				c.addImport(moduloPgType, moduloGoTools)
			}
		}
		c.DB.Types[tName] = t
		c.types = append(c.types, tName)
	}

	sort.Strings(c.types)

	return nil
}

// MakeInterfaceDB create interface of DB
func (c *Creator) MakeInterfaceDB() error {

	f, err := os.Create(path.Join(c.cfg.Dst, "database") + ".go")
	if err != nil && !os.IsExist(err) {
		// err.(*os.PathError).Err
		return errors.Wrap(err, "creator")
	}

	err = c.makeDBUsersTypes()
	if err != nil {
		return err
	}
	tables := slices.Collect(maps.Keys(c.Tables))
	slices.Sort(tables)
	routines := slices.Collect(maps.Keys(c.Routines))
	slices.Sort(routines)
	imports := slices.Collect(maps.Keys(c.imports))
	slices.SortFunc(imports, sortImports())
	tpl.WriteCreateDatabase(f, c, imports, tables, routines)

	return err
}

func sortImports() func(a string, b string) int {
	return func(a, b string) int {
		c := strings.Count(a, "/")
		d := strings.Count(b, "/")
		if c < 2 && d < 2 || c > 1 && d > 1 {
			return strings.Compare(a, b)
		}
		if c < 2 {
			return -1
		}
		return 1
	}
}

// when type is tables record or DB  type
func (c *Creator) chkDefineType(udtName string) string {
	isArray := strings.HasPrefix(udtName, "_") || strings.HasSuffix(udtName, "[]")
	prefix := ""
	if isArray {
		udtName = strings.TrimPrefix(udtName, "_")
		udtName = strings.TrimSuffix(udtName, "[]")
		prefix = "[]"
	}

	if _, ok := c.Tables[udtName]; ok {
		return fmt.Sprintf("%s%sFields", prefix, strcase.ToCamel(udtName))
	}

	if t, ok := c.DB.Types[udtName]; ok {
		if len(t.Enumerates) > 0 {
			return prefix + "string"
		}
		for _, tAttr := range t.Attr {
			if tAttr.Name == "domain" {
				typeCol, _ := c.chkTypes(
					&psql.Column{
						UdtName:     tAttr.Type,
						DataType:    tAttr.Type,
						UserDefined: nil,
					},
					strcase.ToCamel(tAttr.Name))
				return fmt.Sprintf("%s%s", prefix, strcase.ToCamel(typeCol))
			}
		}
		return fmt.Sprintf("%s%s", prefix, strcase.ToCamel(udtName))
	}

	return ""
}

func (c *Creator) prepareReturn(r *psql.Routine) (string, string) {
	toType := psql.UdtNameToType(r.UdtName, nil, nil)
	sType := typesExt.Basic(toType).String()
	switch toType {
	case types.UntypedNil, types.Invalid:
		sType = c.chkDefineType(r.UdtName)
		if sType == "" {
			sType = "*" + strcase.ToCamel(r.UdtName)
		}

	case types.UntypedFloat:
		sType = "float64"
	default:
	}

	return "res " + sType + ", ", "&res,"
}

func (c *Creator) prepareReturns(r *psql.Routine, name string) (sRecord string, sReturn string, sResult string) {
	if len(r.Columns()) == 0 {
		sReturn, sResult = c.prepareReturn(r)
	}
	for _, col := range r.Columns() {
		typeCol, _ := c.chkTypes(col, name)
		if typeCol == "" {
			logs.ErrorLog(dbEngine.NewErrNotFoundType(col.Type(), col.Name()), name)
		}
		s := strings.Trim(col.Name(), "_")
		if s == "type" {
			s += "_"
		}
		sReturn += s + " " + typeCol + ", "
		sResult += "&" + s + ", "
	}
	if len(r.Columns()) > 1 {
		sRecord = strings.ReplaceAll(strings.TrimSuffix(sResult, ","), "&", "&r.")
		sResult = fmt.Sprintf(paramsFormat, sResult)
	}
	return
}

func (c *Creator) prepareParams(r *psql.Routine, name string) (sParams string, sParamsTitle string) {
	// args = make([]any, len(r.Params()))
	for _, param := range r.Params() {
		typeCol, _ := c.chkTypes(param, name)
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
	name := strcase.ToCamel(table.Name())
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

	c.initValues = ""
	c.imports = map[string]struct{}{
		"github.com/jackc/pgtype": struct{}{},
		"fmt": struct {
		}{},
		"sync": struct {
		}{},
		"time": struct {
		}{},

		"github.com/ruslanBik4/dbEngine/dbEngine": struct {
		}{},
		"github.com/ruslanBik4/dbEngine/dbEngine/psql": struct {
		}{},
		"golang.org/x/net/context": struct {
		}{},
		"github.com/ruslanBik4/logs": struct {
		}{},
	}

	c.addImport(moduloPgType, "sync")
	fields, caseRefFields, caseColFields, sTypeField := "", "", "", ""
	columns := slices.Collect(func(yield func(string2 string) bool) {

		for ind, col := range table.Columns() {
			propName := strcase.ToCamel(col.Name())

			typeCol, defValue := c.chkTypes(col, propName)

			if !col.AutoIncrement() && defValue != nil {
				def, ok := defValue.(string)
				if ok {
					if typeCol == "string" {
						c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf(`"%s"`, def))
					}
				} else {
					c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("%v", defValue))
				}
			}

			sTypeField += fmt.Sprintf(scanFormat,
				c.GetFuncForDecode(&dbEngine.TypesAttr{
					Name:      col.Name(),
					Type:      typeCol,
					IsNotNull: false,
				}, ind))

			fields += fmt.Sprintf(colFormat, propName, typeCol, strings.ToLower(col.Name()))
			caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
			caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
			if !yield(col.Name()) {
				return
			}
		}
	})

	imports := slices.Collect(maps.Keys(c.imports))
	slices.SortFunc(imports, sortImports())

	tpl.Newtable(name, table.Name(), table.Comment(), table.(*psql.Table).Type, columns).WriteTable(f, imports, c.Schema)
	//_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), c.initValues)

	tpl.NewColumnType(name, table.Name(), table.Columns()).WriteColumnType(f)

	return err
}

func (c *Creator) chkTypes(col dbEngine.Column, propName string) (string, any) {
	bTypeCol := col.BasicType()
	defValue := col.Default()
	if ud := col.UserDefinedType(); ud != nil {
		for _, tAttr := range ud.Attr {
			if tAttr.Name == "domain" {
				return tAttr.Type, defValue
			}
		}
	}
	typeCol := strings.TrimSpace(typesExt.Basic(bTypeCol).String())
	isArray := strings.HasPrefix(col.Type(), "_") || strings.HasSuffix(col.Type(), "[]")

	switch {
	case bTypeCol == types.UnsafePointer:
		typeCol = "[]byte"

	case (bTypeCol == types.UntypedNil || bTypeCol < 0) && strings.HasPrefix(col.Type(), "any"):
		typeCol = "any"
	//too: chk
	case bTypeCol == types.UntypedNil || bTypeCol < 0:
		typeCol = c.chkDefineType(col.Type())
		if typeCol == "" {
			name, ok := c.ChkDataType(col.Type())
			if ok {
				typeCol = strings.TrimPrefix(fmt.Sprintf("%T", name.Value), "*")
			} else {
				logs.StatusLog(typeCol, col.Type())
				typeCol = "sql.RawBytes"
				c.addImport(moduloSql)
			}
		}
		if a, ok := strings.CutPrefix(typeCol, "*"); ok {
			c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("&%s{}", a))
			defValue = nil
		}

	case bTypeCol == types.UntypedFloat:
		switch col.Type() {
		case "numeric", "decimal":
			typeCol = "psql.Numeric"
			if defValue != nil {
				c.initValues += fmt.Sprintf(initFormat, propName, fmt.Sprintf("psql.NewNumericFromFloat64(%v)", defValue))
				// prevent finally check default
				defValue = nil
			} else {
				c.initValues += fmt.Sprintf(initFormat, propName, "psql.NewNumericNull()")
			}
		case "_numeric", "_decimal", "numeric[]", "decimal[]":
			typeCol = "[]psql.Numeric"
		default:
			logs.ErrorLog(dbEngine.ErrNotFoundColumn{
				Table:  propName,
				Column: col.Type(),
			}, col)
		}

	case isArray:
		typeCol = "[]" + typeCol

	case col.IsNullable():
		typeCol = "sql.Null" + strcase.ToCamel(typeCol)
		c.addImport(moduloSql)
	default:
	}

	return typeCol, defValue
}

func (c *Creator) ChkDataType(typeCol string) (*pgtype.DataType, bool) {
	return ChkDataType(context.TODO(), c.DB, typeCol)
}

func ChkDataType(ctx context.Context, db *dbEngine.DB, typeCol string) (*pgtype.DataType, bool) {
	conn, err := db.Conn.(*psql.Conn).Acquire(ctx)
	if err != nil {
		logs.ErrorLog(err)
		return nil, false
	}
	defer conn.Release()
	return conn.Conn().ConnInfo().DataTypeForName(typeCol)
}

func (c *Creator) GetFuncForDecode(tAttr *dbEngine.TypesAttr, ind int) string {
	tName, name := tAttr.Type, tAttr.Name
	switch _, isTypes := c.DB.Types[strings.ToLower(tName)]; {
	case strings.HasPrefix(tName, "sql.Null"):
		return fmt.Sprintf(
			`%-21s:	*(psql.GetScanner(ci, srcPart[%d], "%s", &%s{}))`,
			strcase.ToCamel(name),
			ind,
			name,
			tName)

	case strings.HasPrefix(tName, "pgtype.") || strings.HasPrefix(tName, "psql.") || isTypes:
		return fmt.Sprintf(
			`%-21s:	*(psql.GetTextDecoder(ci, srcPart[%d], "%s", &%s{}))`,
			strcase.ToCamel(name),
			ind,
			name,
			tName)

	case strings.HasPrefix(tName, "[]"):
		tName = "Array" + strcase.ToCamel(strings.TrimPrefix(tName, "[]"))
	default:
		tName = strcase.ToCamel(tName)
	}

	return fmt.Sprintf(`%-21s:	psql.Get%sFromByte(ci, srcPart[%d], "%s")`,
		strcase.ToCamel(name),
		tName,
		ind,
		name)
}

func (c *Creator) udtToReturnType(udtName string) string {
	toType := psql.UdtNameToType(udtName, c.DB.Types, c.Tables)
	switch toType {
	case types.UnsafePointer:
		return "[]byte"
	case types.UntypedNil, typesExt.TMap, typesExt.TStruct:
		typeReturn := c.chkDefineType(udtName)
		if typeReturn == "" {
			name, ok := c.ChkDataType(udtName)
			if ok {
				typeReturn = fmt.Sprintf("%T", name.Value)
			} else {
				typeReturn = "*" + strcase.ToCamel(udtName)
			}
		}
		if a, ok := strings.CutPrefix(typeReturn, "[]"); toType < 0 && ok {
			typeReturn = "WrapArray[*" + a + "]"
		}

		return typeReturn

	case types.UntypedFloat:
		return "float64"

	default:
		s := typesExt.Basic(toType).String()
		if s == "" {
			logs.StatusLog(udtName)
		}
		return s
	}
}

func (c *Creator) getTypeCol(col dbEngine.Column) string {
	switch typeName := col.Type(); typeName {
	case "inet", "interval":
		c.addImport(moduloPgType)
		return strcase.ToCamel(typeName)

	case "json", "jsonb":
		return "Json"

	case "date", "timestamp", "timestamptz", "time":
		if col.IsNullable() {
			return "RefTime"
		} else {
			return "Time"
		}

	case "timerange", "tsrange", "_date", "daterange", "_timestamp", "_timestamptz", "_time":
		return "ArrayTime"
	default:
		return "Any"
	}
}

var mapTypes = map[string]string{
	"Inet":      "pgtype.Inet",
	"Interval":  "pgtype.Interval",
	"Json":      "any",
	"jsonb":     "any",
	"RefTime":   "*time.Time",
	"Time":      "time.Time",
	"ArrayTime": "[]time.Time",
	"Any":       "any",
}

func (c *Creator) addImport(moduloNames ...string) {
	for _, name := range moduloNames {
		c.imports[name] = struct{}{}
	}
}
