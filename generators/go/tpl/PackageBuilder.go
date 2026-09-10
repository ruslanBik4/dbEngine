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

	// usedCompositeTypes records which table names and c.types composite/range
	// names are actually resolved to a Postgres composite VALUE somewhere in the
	// generated output ([]*T, or a scalar *T composite routine param/return) -
	// populated by computeUsedCompositeTypes before generation starts.
	// registerDataTypes (database_tpl.qtpl) only loads and registers entries in
	// this set, not every declared type/table, since the rest are never
	// scanned/encoded as a whole composite value at all and would just be
	// wasted LoadType round trips at connection time. citext is exempt - it's
	// commonly used as a plain per-column string, never through this composite
	// path, so it's always registered regardless of this set. This same set
	// also drives CustomCodecNames below, since only a type that's actually
	// used as a composite value needs its generated Codec registered.
	usedCompositeTypes map[string]bool
}

// markCompositeTypeUsed records name (a table or c.types composite/range name)
// as needing its own pgtype registration, then walks its fields/columns to mark
// any OTHER custom composite/range type or table it references, transitively -
// conn.LoadType(ctx, name) requires every such dependency already registered
// (see getCompositeFields in pgx's own source), so a dependency missing from
// this set would make LoadType fail at connection time with no way to recover.
// The recursion terminates because an already-marked name returns immediately,
// which also protects against a cycle (e.g. two composite types referencing
// each other).
func (c *PackageBuilder) markCompositeTypeUsed(name string) {
	if c.usedCompositeTypes == nil {
		c.usedCompositeTypes = make(map[string]bool)
	}
	if c.usedCompositeTypes[name] {
		return
	}
	c.usedCompositeTypes[name] = true

	markFieldTypeIfCustom := func(rawType string) {
		fieldTypeName := strings.TrimPrefix(rawType, "_")
		fieldTypeName = strings.TrimSuffix(fieldTypeName, "[]")
		if _, ok := c.DB.Types[fieldTypeName]; ok {
			c.markCompositeTypeUsed(fieldTypeName)
		} else if _, ok := c.Tables[fieldTypeName]; ok {
			c.markCompositeTypeUsed(fieldTypeName)
		}
	}

	// rawTypeAttrs (not t.Attr, which MakeDBUsersTypes has already overwritten
	// with the resolved Go type) has the original Postgres field type names for
	// a c.types entry - empty/absent for a table name, which is fine, tables are
	// walked via their Columns() below instead.
	for _, rawAttr := range c.rawTypeAttrs[name] {
		if rawAttr.Name == "domain" {
			continue
		}
		markFieldTypeIfCustom(rawAttr.Type)
	}

	if table, ok := c.Tables[name]; ok {
		for _, col := range table.Columns() {
			markFieldTypeIfCustom(col.Type())
		}
	}
}

// computeUsedCompositeTypes scans every routine's parameters and result
// columns and records, via markCompositeTypeUsed, which table and c.types
// composite/range names are actually referenced as a composite Postgres VALUE
// - the same condition ChkTypes/chkDefineType use to decide a column needs
// []*T or a dedicated *T composite type rather than a plain Go type.
//
// It must run before WriteCreateDatabase/CreateDatabase, since
// CreateDatabase's own registerDataTypes output is textually emitted before
// the routine invokers that reference these types - by the time that
// generation would otherwise populate this set as a side effect, the pending
// list has already been built. It deliberately does NOT call
// ChkTypes/chkDefineType directly to determine this: those have other side
// effects (c.initValues, c.addImport) that must fire exactly once, during the
// real generation pass - calling them again here would duplicate them in the
// output. This instead re-derives just the "does this column resolve to a
// table or composite/range type" check.
func (c *PackageBuilder) computeUsedCompositeTypes() {
	scan := func(col dbEngine.Column) {
		bTypeCol := col.BasicType()
		if !(bTypeCol == types.UntypedNil || bTypeCol < 0) {
			return
		}

		udtName := strings.TrimPrefix(col.Type(), "_")
		udtName = strings.TrimSuffix(udtName, "[]")

		if _, ok := c.Tables[udtName]; ok {
			c.markCompositeTypeUsed(udtName)
			return
		}

		t, ok := c.DB.Types[udtName]
		if !ok || len(t.Enumerates) > 0 {
			return
		}

		for _, tAttr := range t.Attr {
			if tAttr.Name == "domain" {
				return // domains reuse their base type's codec - no registration of their own
			}
		}

		c.markCompositeTypeUsed(udtName)
	}

	for _, r := range c.Routines {
		routine, ok := r.(*psql.Routine)
		if !ok {
			continue
		}

		for _, param := range routine.Params() {
			scan(param)
		}
		for _, col := range routine.Columns() {
			scan(col)
		}
	}
}

// PendingDataTypeNames returns the Postgres type/table names
// registerDataTypes (database_tpl.qtpl's loadDataTypes) should load and
// register via conn.LoadType: each c.types composite/range/domain/enum name
// and each name in listTables that's actually in c.usedCompositeTypes (see
// computeUsedCompositeTypes for why anything else is skipped - it's declared
// but never scanned/encoded as a whole composite value anywhere in the
// generated code, so loading it would just be a wasted round trip), plus each
// included name's own array variant ("_<name>" - Postgres auto-creates one
// per type, and it needs its own registration the same way the base type
// does; see database_tpl.qtpl's comment on this for the pgtype doc.go
// reference). citext is never included here: it can't go through
// conn.LoadType at all (see loadDataTypes' own comment), so it's registered
// separately via a dedicated regtype-cast+TextCodec query - only its array
// variant "_citext" belongs in this list, unconditionally, since citext is
// typically used as a plain per-column string rather than through the
// composite-value path this trimming is based on.
//
// This filtering is plain Go rather than inline {% if %}/{% for %} logic in
// the template so it reads and tests like ordinary code; database_tpl.qtpl
// just ranges over the result to emit the pending slice literal.
func (c *PackageBuilder) PendingDataTypeNames(listTables []string) []string {
	var names []string

	for _, name := range c.types {
		if name == "citext" {
			names = append(names, "_citext")
			continue
		}
		if c.usedCompositeTypes[name] {
			names = append(names, name, "_"+name)
		}
	}

	for _, name := range listTables {
		if c.usedCompositeTypes[name] {
			names = append(names, name, "_"+name)
		}
	}

	return names
}

// CustomCodecNames returns the base Postgres type/table names (no array
// variants, no citext) that PendingDataTypeNames also selects for loading -
// i.e. every composite/range type or table actually used as a composite
// value - and that therefore has a hand-written pgtype.Codec struct generated
// for it (CreateTypeInterface in database_tpl.qtpl, or {Table}PsqlType in
// ColumnType.qtpl). database_tpl.qtpl's loadDataTypes uses this to build
// customCodecs, overriding conn.LoadType's own generic
// *pgtype.CompositeCodec/*pgtype.RangeCodec with the generated one so
// PlanEncode/PlanScan/Scan/DecodeValue actually run instead of pgx's generic
// reflection-based struct fallback.
//
// citext is excluded because it has no generated struct at all - it reuses
// pgtype.TextCodec directly and is registered separately in loadDataTypes.
// Enums and domains are excluded implicitly: computeUsedCompositeTypes never
// marks them in usedCompositeTypes (they have no composite Codec of their
// own - enums resolve to a plain "string", domains reuse their base type's
// Codec), so this naturally only lists tables and genuine composite/range
// types just like PendingDataTypeNames' base (non "_"-prefixed) entries.
//
// For each returned name, c.chkDefineType(name) gives the exact generated Go
// struct name to instantiate (e.g. "UsersPsqlType", "AccountsGroup") - this
// mirrors PendingDataTypeNames' own base-name selection so the two stay in
// lockstep by construction rather than by convention.
func (c *PackageBuilder) CustomCodecNames(listTables []string) []string {
	var names []string

	for _, name := range c.types {
		if name == "citext" || !c.usedCompositeTypes[name] {
			continue
		}
		names = append(names, name)
	}

	for _, name := range listTables {
		if c.usedCompositeTypes[name] {
			names = append(names, name)
		}
	}

	slices.Sort(names)

	return names
}

// PrepareDatabase sort databases properties
func (c *PackageBuilder) PrepareDatabase(f io.Writer) error {
	err := c.MakeDBUsersTypes()
	if err != nil {
		return err
	}

	// Must run after MakeDBUsersTypes (needs rawTypeAttrs/DB.Types resolved) and
	// before WriteCreateDatabase below - see computeUsedCompositeTypes' own
	// comment for why. database_tpl.qtpl's registerDataTypes reads
	// c.usedCompositeTypes to trim what it loads/registers.
	c.computeUsedCompositeTypes()

	// NOTE: c.SortImports() below is evaluated as an argument expression, i.e.
	// *before* WriteCreateDatabase (and the CreateDatabase/CreateTypeInterface
	// template bodies it runs) ever executes. Any c.addImport call made from
	// inside those template bodies is too late to affect the emitted import block,
	// so every import CreateDatabase's output unconditionally or conditionally
	// needs must be added here first.

	// Kept unconditional rather than gated on len(c.types)/len(c.Tables): besides
	// the generated Codec structs and loadDataTypes' customCodecs map (which are
	// gated), routine columns/params can independently need pgtype types
	// (pgtype.Timestamptz, pgtype.Date, ...) regardless of whether this database
	// has any composite type/table, and that's determined per-column at
	// CreateRoutinesInvoker generation time (too late for c.addImport - see the
	// NOTE at the top of CreateDatabase), so this stays a conservative always-add
	// rather than trying to predict every case precisely.
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

	// registerDataTypes/loadDataTypes are emitted whenever there is at least one
	// custom type (citext included) OR at least one table - mirror that exact
	// condition here, not just len(c.types) > 0, since a table-only database (no
	// CREATE TYPE'd types at all) still gets registerDataTypes for its table row
	// types (see database_tpl.qtpl). They use fmt.Errorf/fmt.Sprintf, *pgx.Conn
	// from the base pgx package (not just pgtype/pgconn) for LoadType/TypeMap,
	// and a sync.Mutex to cache the loaded types across connections instead of
	// re-running LoadType's introspection queries on every new connection - and
	// CreateTypeInterface's own Scan/DecodeDatabaseSQLValue/PlanEncode/DecodeValue
	// bodies additionally use fmt.Errorf/fmt.Sprintf whenever needsCompositeCodecImports
	// is true, so "fmt" covers both.
	if len(c.types) > 0 || len(c.Tables) > 0 {
		c.addImport("fmt", moduloPgx, "sync")
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
			// see routines.qtpl's CreateFunctionInvoker for why a plain []*T
			// (not WrapArray[*T]) is used for an array of a composite/range type.
			typeReturn = "[]*" + a
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
		// UdtNameToType maps both "bytea" and its array variant "_bytea" to
		// the same types.UnsafePointer kind (json/jsonb land here too, but
		// have no "_json"/"_jsonb" case in UdtNameToType, so they never
		// reach this branch as arrays). Since this case is checked before
		// `case isArray` below, a bytea[] column used to fall straight into
		// the scalar "[]byte" branch and silently lose its array-ness -
		// isArray was computed above but never consulted here.
		if isArray {
			typeCol = "[][]byte"
		} else {
			typeCol = "[]byte"
		}

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

	// NOTE (not fixed here, needs your call): this case looks unreachable
	// today. UdtNameToType (dbEngine/psql/column.go) maps "numeric" /
	// "decimal" (and their array/float8/money siblings) to types.Float64,
	// never to types.UntypedFloat - matching the "// todo add check field
	// length UntypedFloat" comment right on that line, i.e. it reads like
	// planned-but-not-wired-up work. As written, a numeric/decimal column's
	// bTypeCol is types.Float64, which skips this whole branch (including
	// its psql.Numeric routing and default-value init) and falls through to
	// plain "float64" via `case isArray`/the default typeCol computed above
	// - silently losing the precision psql.Numeric exists for. If
	// psql.Numeric is still wanted here, UdtNameToType needs to actually
	// return types.UntypedFloat for numeric/decimal (or this switch needs
	// to key off col.Type() instead of bTypeCol).
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
		return fmt.Sprintf("[]%s", c.getCodecType(col, t.ElementType))

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
		// composite VALUE - a routine parameter/return of that table's type, or (as
		// a plain []*T in routines.qtpl) an array of it - the
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
