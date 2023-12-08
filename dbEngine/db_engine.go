package dbEngine

import (
	"bytes"
	"encoding/json"
	"go/types"

	"github.com/jackc/pgtype"
	"golang.org/x/net/context"

	"github.com/ruslanBik4/gotools"
	"github.com/ruslanBik4/logs"
)

// FncEachRow & FncRawRow are types of function which use as callback for select methods
type (
	FncEachRow func(values []any, columns []Column) error
	FncRawRow  func(values [][]byte, columns []Column) error
)

// Connection implement conn operation
type Connection interface {
	InitConn(ctx context.Context, dbURL string) error
	GetRoutines(ctx context.Context) (map[string]Routine, error)
	GetSchema(ctx context.Context) (database map[string]*string, tables map[string]Table, routines map[string]Routine, dbTypes map[string]Types, err error)
	GetStat() string
	ExecDDL(ctx context.Context, sql string, args ...any) error
	NewTable(name, typ string) Table
	LastRowAffected() int64
	SelectOneAndScan(ctx context.Context, rowValues any, sql string, args ...any) error
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, sql string, args ...any) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, sql string, args ...any) error
	SelectAndPerformRaw(ctx context.Context, each FncRawRow, sql string, args ...any) error
	SelectToMap(ctx context.Context, sql string, args ...any) (map[string]any, error)
	SelectToMaps(ctx context.Context, sql string, args ...any) ([]map[string]any, error)
	SelectToMultiDimension(ctx context.Context, sql string, args ...any) ([][]any, []Column, error)
}

// Types consists of parameters of DB types
type TypesAttr struct {
	Name      string
	Type      string
	IsNotNull bool
}

func (dst *TypesAttr) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = TypesAttr{}
		return nil
	}

	srcPart := bytes.Split(src[1:len(src)-1], []byte(","))
	*dst = TypesAttr{
		Name:      gotools.BytesToString(bytes.Trim(srcPart[0], `\"`)),
		Type:      gotools.BytesToString(bytes.Trim(srcPart[1], `\"`)),
		IsNotNull: gotools.BytesToString(bytes.Trim(srcPart[2], `\"`)) == "true",
	}
	return nil
}

type TypesAttrs []TypesAttr

func (dst *TypesAttrs) DecodeText(ci *pgtype.ConnInfo, src []byte) error {
	if len(src) == 0 {
		*dst = TypesAttrs{}
		return nil
	}
	err := json.Unmarshal(src, dst)
	if err != nil {
		logs.ErrorLog(err)
		return err
	}

	return nil
}

// Types consists of parameters of DB types
type Types struct {
	Id         uint32
	Name       string
	Type       rune
	Attr       TypesAttrs
	Enumerates []string
}

func NewTypes() *Types {
	return &Types{}
}

func (t *Types) GetFields(columns []Column) []any {
	if len(columns) == 0 {
		return []any{&t.Id, t.Type, &t.Type, &t.Enumerates}
	}

	v := make([]any, len(columns))
	for i, col := range columns {
		switch name := col.Name(); name {
		case "oid":
			v[i] = &t.Id
		case "typname":
			v[i] = &t.Name
		case "typtype":
			v[i] = &t.Type
		case "attr":
			v[i] = &t.Attr
		case "enumerates":
			v[i] = &t.Enumerates
		}
	}

	return v
}

// Table describes methods for table operations
type Table interface {
	Columns() []Column
	Comment() string
	FindColumn(name string) Column
	FindIndex(name string) *Index
	Delete(ctx context.Context, Options ...BuildSqlOptions) (int64, error)
	Indexes() Indexes
	GetColumns(ctx context.Context) error
	Insert(ctx context.Context, Options ...BuildSqlOptions) (int64, error)
	Update(ctx context.Context, Options ...BuildSqlOptions) (int64, error)
	Upsert(ctx context.Context, Options ...BuildSqlOptions) (int64, error)
	Name() string
	ReReadColumn(ctx context.Context, name string) Column
	Select(ctx context.Context, Options ...BuildSqlOptions) error
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error
	SelectOneAndScan(ctx context.Context, row any, Options ...BuildSqlOptions) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error
}

// Routine describes methods for function/procedures operations
type Routine interface {
	Name() string
	BuildSql(Options ...BuildSqlOptions) (sql string, args []any, err error)
	Select(ctx context.Context, args ...any) error
	Call(ctx context.Context, args ...any) error
	Overlay() Routine
	Params() []Column
	ReturnType() string
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error
	SelectOneAndScan(ctx context.Context, row any, Options ...BuildSqlOptions) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error
}

// ForeignKey consists of parameters of foreign key
type ForeignKey struct {
	Parent     string `json:"parent"`
	Column     string `json:"column"`
	UpdateRule string `json:"update_rule"`
	DeleteRule string `json:"delete_rule"`
	ForeignCol Column `json:"-"`
}

// Column describes methods for table/view/function builderOpts
type Column interface {
	BasicType() types.BasicKind                //+
	BasicTypeInfo() types.BasicInfo            //+
	CheckAttr(fieldDefine string) []FlagColumn //+
	CharacterMaximumLength() int               //+
	Comment() string                           //+
	Name() string                              //+
	AutoIncrement() bool                       //+
	IsNullable() bool                          //+
	Default() any                              //+
	SetDefault(any)                            //+
	Foreign() *ForeignKey
	UserDefinedType() *Types
	Table() Table
	Primary() bool    //+
	Type() string     //+
	Required() bool   //+
	SetNullable(bool) //+
}

// Index consists of index properties
type Index struct {
	Name                         string
	foreignTable, foreignColumn  string
	updateCascade, deleteCascade string
	Expr                         string
	Where                        string
	Unique                       bool
	Columns                      []string
}

func (ind *Index) AddColumn(name string) bool {
	for _, col := range ind.Columns {
		if col == name {
			return false
		}
	}

	ind.Columns = append(ind.Columns, name)

	return true
}

// GetFields implements interface RowScanner
func (ind *Index) GetFields(columns []Column) []any {
	fields := make([]any, len(columns))
	for i, col := range columns {
		switch col.Name() {
		case "index_name":
			fields[i] = &ind.Name
		case "ind_expr":
			fields[i] = &ind.Expr
		case "ind_unique":
			fields[i] = &ind.Unique
		case "column_names":
			fields[i] = &ind.Columns
		default:
			logs.DebugLog("unknown column %s", col.Name())
		}
	}

	return fields
}

// Indexes cluster index of table
type Indexes []*Index

// GetFields implements interface RowScanner
func (i *Indexes) GetFields(columns []Column) []any {
	ind := &Index{}
	*i = append(*i, ind)

	return ind.GetFields(columns)
}

// LastIndex returns last index of list
func (i Indexes) LastIndex() *Index {
	return i[len(i)-1]
}

// RowScanner must return slice variables for pgx.Rows.Scan
type RowScanner interface {
	GetFields(columns []Column) []any
}
