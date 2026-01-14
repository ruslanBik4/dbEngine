package tpl

import (
	"fmt"
	"go/types"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jackc/pgtype"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/dbEngine/dbEngine/psql"
	"github.com/ruslanBik4/gotools/typesExt"
	"github.com/ruslanBik4/logs"
)

type PackageBuilder struct {
	*dbEngine.DB
	Ctx        context.Context
	Types      map[string]string
	Imports    map[string]struct{}
	initValues string
	types      []string
}

func (c *PackageBuilder) PrepareDatabase(f io.Writer) error {
	err := c.MakeDBUsersTypes()
	if err != nil {
		return err
	}
	tables := slices.Collect(maps.Keys(c.Tables))
	slices.Sort(tables)
	routines := slices.Collect(maps.Keys(c.Routines))
	slices.Sort(routines)
	imports := slices.Collect(maps.Keys(c.Imports))
	slices.SortFunc(imports, sortImports())

	c.WriteCreateDatabase(f, c.DB.Schema, imports, tables, routines)

	return nil
}

func (c *PackageBuilder) PrepareTable(table dbEngine.Table) *Table {
	name := strcase.ToCamel(table.Name())
	c.initValues = ""
	c.Imports = maps.Collect(func(yield func(string, struct{}) bool) {
		for _, name := range []string{
			"fmt",
			"slices",
			"sync",
			"time",
			moduloPgType,
			"golang.org/x/net/context",
			"github.com/ruslanBik4/logs",
			"github.com/jackc/pgtype",
			"github.com/ruslanBik4/dbEngine/dbEngine",
			"github.com/ruslanBik4/dbEngine/dbEngine/psql",
		} {
			if !yield(name, struct{}{}) {
				return
			}
		}
	})

	fields, caseRefFields, caseColFields, sTypeField := "", "", "", ""
	properties := make(map[string]string)
	columns := slices.Collect(func(yield func(string2 string) bool) {

		for ind, col := range table.Columns() {
			propName := strcase.ToCamel(col.Name())

			typeCol, defValue := c.ChkTypes(col, propName)

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
			properties[col.Name()] = typeCol
		}
	})
	imports := slices.Collect(maps.Keys(c.Imports))
	slices.SortFunc(imports, sortImports())

	return NewTable(name, table.Name(), table.Comment(), table.(*psql.Table).Type, columns, imports, properties)
	//_, err = fmt.Fprintf(f, footer, name, caseRefFields, caseColFields, table.Name(), c.initValues)
}

func (c *PackageBuilder) GetFuncForDecode(tAttr *dbEngine.TypesAttr, ind int) string {
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

func (c *PackageBuilder) udtToReturnType(udtName string) string {
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

// MakeDBUsersTypes create interface of DB
func (c *PackageBuilder) MakeDBUsersTypes() error {
	for tName, t := range c.DB.Types {
		for i, tAttr := range t.Attr {
			name := tAttr.Name
			ud := &t
			if tAttr.Name == "domain" {
				logs.StatusLog("%s, %c %v", tName, t.Type, tAttr)
				ud = nil
			}
			typeCol, _ := c.ChkTypes(
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

	slices.Sort(c.types)

	return nil
}

func (c *PackageBuilder) ChkTypes(col dbEngine.Column, propName string) (string, any) {
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

func (c *PackageBuilder) ChkDataType(typeCol string) (*pgtype.DataType, bool) {
	return psql.ChkDataType(context.TODO(), c.DB, typeCol)
}

// when type is tables record or DB  type
func (c *PackageBuilder) chkDefineType(udtName string) string {
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
				typeCol, _ := c.ChkTypes(
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

func (c *PackageBuilder) addImport(moduloNames ...string) {
	for _, name := range moduloNames {
		c.Imports[name] = struct{}{}
	}
}
func (c *PackageBuilder) getTypeCol(col dbEngine.Column) string {
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
