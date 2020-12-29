package dbEngine

import (
	"go/types"

	"github.com/ruslanBik4/logs"
	"golang.org/x/net/context"
)

type Connection interface {
	InitConn(ctx context.Context, dbURL string) error
	GetRoutines(ctx context.Context) (map[string]Routine, error)
	GetSchema(ctx context.Context) (map[string]Table, map[string]Routine, map[string]Types, error)
	GetStat() string
	ExecDDL(ctx context.Context, sql string, args ...interface{}) error
	NewTable(name, typ string) Table
	LastRowAffected() int64
	SelectOneAndScan(ctx context.Context, rowValues interface{}, sql string, args ...interface{}) error
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, sql string, args ...interface{}) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, sql string, args ...interface{}) error
	SelectToMap(ctx context.Context, sql string, args ...interface{}) (map[string]interface{}, error)
	SelectToMaps(ctx context.Context, sql string, args ...interface{}) ([]map[string]interface{}, error)
	SelectToMultiDimension(ctx context.Context, sql string, args ...interface{}) ([][]interface{}, []Column, error)
}

type FncEachRow func(values []interface{}, columns []Column) error

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
	BuildSql(Options ...BuildSqlOptions) (string, []interface{}, error)
	Select(ctx context.Context, args ...interface{}) error
	Call(ctx context.Context, args ...interface{}) error
	Overlay() Routine
	Params() []Column
	SelectAndScanEach(ctx context.Context, each func() error, rowValue RowScanner, Options ...BuildSqlOptions) error
	SelectOneAndScan(ctx context.Context, row interface{}, Options ...BuildSqlOptions) error
	SelectAndRunEach(ctx context.Context, each FncEachRow, Options ...BuildSqlOptions) error
}

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
	Primary() bool
	Type() string
	Required() bool
	SetNullable(bool)
}

// Index
type Index struct {
	Name    string
	Columns []string
}

func (ind *Index) GetFields(columns []Column) []interface{} {
	fields := make([]interface{}, len(columns))
	for i, col := range columns {
		switch col.Name() {
		case "index_name":
			fields[i] = &ind.Name
		case "column_names":
			fields[i] = &ind.Columns
		default:
			logs.DebugLog("unknown column %s", col.Name())
		}
	}

	return fields
}

type Indexes []*Index

func (i *Indexes) GetFields(columns []Column) []interface{} {
	ind := &Index{}
	*i = append(*i, ind)

	return ind.GetFields(columns)
}

type RowScanner interface {
	GetFields(columns []Column) []interface{}
}
