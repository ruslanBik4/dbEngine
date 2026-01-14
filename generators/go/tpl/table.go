package tpl

type Table struct {
	name       string
	dbName     string
	columns    []string
	comment    string
	imports    []string
	typ        string
	properties map[string]string
}

func NewTable(name, dbName, comment, typ string, columns, imports []string, properties map[string]string) *Table {
	return &Table{
		name:       name,
		dbName:     dbName,
		columns:    columns,
		comment:    comment,
		imports:    imports,
		properties: properties,
		typ:        typ,
	}
}
