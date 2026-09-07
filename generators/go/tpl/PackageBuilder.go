package tpl

import (
	"context"
	"fmt"
	"go/types"
	"io"
	"maps"
	"slices"
	"strings"

	"github.com/iancoleman/strcase"
	"github.com/jackc/pgx/v5/pgtype"

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

	// rawTypeAttrs preserves each user-defined type's attributes exactly as Postgres
	// reports them (raw type names, e.g. "int4", "text", "numeric") keyed by type
	// name, captured before MakeDBUsersTypes overwrites TypesAttr.Type in place with
	// the resolved Go type. CreateTypeInterface needs the raw names to resolve each
	// field's/range element's pgtype.Type (and so its OID) when building PlanEncode.
	rawTypeAttrs map[string][]dbEngine.TypesAttr
}

// PrepareDatabase sort databases properties
func (c *PackageBuilder) PrepareDatabase(f io.Writer) error {
	err := c.MakeDBUsersTypes()
	if err != nil {
		return err
	}

	// NOTE: c.SortImports() below is evaluated as an argument expression, i.e.
	// *before* WriteCreateDatabase (and the CreateDatabase/CreateTypeInterface
	// template bodies it runs) ever executes. Any c.addImport call made from
	// inside those template bodies is too late to affect the emitted import block,
	// so every import CreateDatabase's output unconditionally or conditionally
	// needs must be added here first.

	// ValueDecoder/WrapArray in the generated output reference pgtype.Codec
	// unconditionally, so this import is always required.
	c.addImport(moduloPgType)

	// database/sql/driver.Value is only used by CreateTypeInterface's generated
	// DecodeDatabaseSQLValue body, which only exists for types that actually
	// produce a struct+codec - mirror the exact condition CreateTypeInterface
	// itself gates on (composites and ranges, not citext, domains, or bare enums)
	// so we don't add an import that ends up unused, which is a compile error in Go.
	needsCompositeCodecImports := false
	for _, name := range c.types {
		if name == "citext" {
			continue
		}
		t := c.DB.Types[name]
		if len(t.Enumerates) == 0 && len(t.Attr) > 0 && t.Attr[0].Name != "domain" {
			needsCompositeCodecImports = true
			break
		}
	}
	if needsCompositeCodecImports {
		c.addImport("database/sql/driver")
	}

	// registerDataTypes (emitted whenever there is at least one custom type,
	// citext included) uses fmt.Errorf in its retry loop and needs *pgx.Conn from
	// the base pgx package (not just pgtype/pgconn) for LoadType/TypeMap - and
	// CreateTypeInterface's own Scan/DecodeDatabaseSQLValue/PlanEncode/DecodeValue
	// bodies additionally use fmt.Errorf/fmt.Sprintf whenever needsCompositeCodecImports
	// is true, so "fmt" covers both.
	if len(c.types) > 0 {
		c.addImport("fmt")
		c.addImport(moduloPgx)
	}

	tables := slices.Collect(maps.Keys(c.Tables))
	slices.Sort(tables)
	routines := slices.Collect(maps.Keys(c.Routines))
	slices.Sort(routines)
	c.WriteCreateDatabase(f, c.DB.Schema, c.SortImports(), tables, routines)

	return nil
}

func (c *PackageBuilder) SortImports() []string {
	imports := slices.Collect(maps.Keys(c.Imports))
	slices.SortFunc(imports, sortImports())
	return imports
}

// PrepareTable create defined for columns
func (c *PackageBuilder) PrepareTable(table dbEngine.Table) *Table {
	name := strcase.ToCamel(table.Name())
	c.initValues = ""
	c.Imports = maps.Collect(func(yield func(string, struct{}) bool) {
		for _, name := range []string{
			"fmt",
			"slices",
			"sync",
			"time",

			"context",
			"github.com/ruslanBik4/logs",
			"database/sql/driver",
			moduloPgType,
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

			fields += fmt.Sprintf("\n\t"+colFormat, propName, typeCol, strings.ToLower(col.Name()))
			caseRefFields += fmt.Sprintf(caseRefFormat, col.Name(), propName)
			caseColFields += fmt.Sprintf(caseColFormat, col.Name(), propName)
			if !yield(col.Name()) {
				return
			}
			properties[col.Name()] = typeCol
		}
	})

	return NewTable(name, table.Name(), table.Comment(), table.(*psql.Table).Type, columns, c.SortImports(), properties)
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
		c.Imports[moduloPgType] = struct{}{}
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
				typeReturn = fmt.Sprintf("%T", name.Codec)
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
	c.rawTypeAttrs = make(map[string][]dbEngine.TypesAttr, len(c.DB.Types))

	c.types = slices.AppendSeq(c.types, func(yield func(string) bool) {
		for tName, t := range c.DB.Types {
			// snapshot the raw (pre-ChkTypes) attributes before the loop below
			// overwrites tAttr.Type with the resolved Go type
			c.rawTypeAttrs[tName] = append([]dbEngine.TypesAttr(nil), t.Attr...)

			for i, tAttr := range t.Attr {
				name := tAttr.Name
				ud := &t
				if tAttr.Name == "domain" {
					logs.StatusLog("%s: '%c' %+v", tName, t.Type, tAttr)
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
					c.addImport(moduloPgType) //, moduloGoTools, "fmt")
				}
			}
			c.DB.Types[tName] = t
			if !yield(tName) {
				return
			}
		}
	})

	slices.Sort(c.types)

	return nil
}

// ChkTypes decided type of column as Golang type
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
			colType, ok := c.ChkDataType(col.Type())
			if ok {
				typeCol = c.getCodecType(col, colType)
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

func (c *PackageBuilder) getCodecType(col dbEngine.Column, colType *pgtype.Type) (typeCol string) {
	switch t := colType.Codec.(type) {
	case *pgtype.MultirangeCodec:
		return fmt.Sprintf("pgtype.Multirange[%s]", c.getCodecType(col, t.ElementType))

	case *pgtype.RangeCodec:
		return fmt.Sprintf("pgtype.Range[%s]", c.getCodecType(col, t.ElementType))

	case *pgtype.ArrayCodec:
		return fmt.Sprintf("pgtype.Array[%s]", c.getCodecType(col, t.ElementType))

	default:
		typeCol = strings.TrimSuffix(strings.TrimPrefix(fmt.Sprintf("%T", colType.Codec), "*"), "Codec")
		if b, ok := strings.CutSuffix(colType.Name, "range"); ok {
			m := pgtype.NewMap()

			value, err := colType.Codec.DecodeValue(m, colType.OID, pgtype.TextFormatCode, []byte("(0,0)"))
			if err != nil {
				logs.ErrorLog(err)
				typeCol += "[pgtype." + strcase.ToCamel(b) + "]"
			} else {
				typeCol = fmt.Sprintf("%T", value)
			}
			logs.StatusLog(typeCol, b, col.Type(), colType)
		}

		return typeCol
	}
}

func (c *PackageBuilder) ChkDataType(typeCol string) (*pgtype.Type, bool) {
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
		// PsqlType, not Fields: whenever a table's row type is used as a Postgres
		// composite VALUE - a routine parameter/return of that table's type, or (via
		// WrapArray[T ValueDecoder[T]] in database_tpl.qtpl) an array of it - the
		// destination must implement the full pgtype.Codec interface (PlanEncode,
		// PlanScan, DecodeValue, ...), which only {Table}PsqlType (column_type.qtpl)
		// does. {Table}Fields is embedded inside {Table}PsqlType, so every existing
		// per-column accessor (RefColValue/ColValue/GetFields) still works unchanged
		// through that embedding - this only widens what's returned, it doesn't
		// narrow it. Using Fields here directly used to compile-fail with e.g.
		// "*ExecutionsFields does not satisfy ValueDecoder[*ExecutionsFields]
		// (missing method DecodeDatabaseSQLValue)" for any routine returning an
		// array of a table's row type.
		return fmt.Sprintf("%s%sPsqlType", prefix, strcase.ToCamel(udtName))
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
