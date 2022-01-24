package dbEngine

import (
	"go/types"

	"github.com/ruslanBik4/logs"
	"golang.org/x/net/context"
)

// type of function which use as callback for select methods
type (
	FncEachRow func(values []interface{}, columns []Column) error
	FncRawRow  func(values [][]byte, columns []Column) error
)

// Connection implement conn operation
type Connection interface {
	InitConn(ctx context.Context, dbURL string) error
	GetRoutines(ctx context.Context) (map[string]Routine, error)
	GetSchema(ctx context.Context) (database map[string]*string, tables map[string]Table, routines map[string]Routine, dbTypes map[string]Types, err error)
	GetStat() string
	ExecDDL(ctx context.Context, sql string, args ...interface{}) error
	NewTable(name, typ string) Table
	LastRowAffected() int64
	SelectOneAndScan(ctx context.Context, rowValues interface{}, sql string, args ...interface{}) error
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, sql string, args ...interface{}) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, sql string, args ...interface{}) error
	SelectAndPerformRaw(ctx context.Context, each FncRawRow, sql string, args ...interface{}) error
	SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error)
	SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error)
	SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []Column, error)
}

type Types struct {
	Id   int
	Name string
	Attr []string
}

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
	ReReadColumn(name string) Column
	Select(ctx context.Context, Options ...BuildSqlOptions) error
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error
	SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error
}

type Routine interface {
	Name() string
	BuildSql(Options ...BuildSqlOptions) (sql string, args []interface{}, err error)
	Select(ctx context.Context, args ...interface{}) error
	Call(ctx context.Context, args ...interface{}) error
	Overlay() Routine
	Params() []Column
	ReturnType() string
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error
	SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error
}

type ForeignKey struct {
	Parent     string `json:"parent"`
	Column     string `json:"column"`
	UpdateRule string `json:"update_rule"`
	DeleteRule string `json:"delete_rule"`
}

// Column implements methods of table/view/function builderOpts
type Column interface {
	BasicType() types.BasicKind
	BasicTypeInfo() types.BasicInfo
	CheckAttr(fieldDefine string) string
	CharacterMaximumLength() int
	Comment() string
	Name() string
	AutoIncrement() bool
	IsNullable() bool
	Default() interface{}
	SetDefault(interface{})
	Foreign() *ForeignKey
	Primary() bool
	Type() string
	Required() bool
	SetNullable(bool)
}

// Index
type Index struct {
	Name    string
	Expr    string
	Unique  bool
	Columns []string
}

func (ind *Index) GetFields(columns []Column) []interface{} {
	fields := make([]interface{}, len(columns))
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

func (i *Indexes) GetFields(columns []Column) []interface{} {
	ind := &Index{}
	*i = append(*i, ind)

	return ind.GetFields(columns)
}

func (i Indexes) LastIndex() *Index {
	return i[len(i)-1]
}

// RowScanner must return slice variables for pgx.Rows.Scan
type RowScanner interface {
	GetFields(columns []Column) []interface{}
}
