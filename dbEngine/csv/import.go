package csv

import (
	"bufio"
	"io"
	"slices"
	"strings"

	"github.com/go-errors/errors"

	"github.com/ruslanBik4/dbEngine/dbEngine"
	"github.com/ruslanBik4/gotools"
	"github.com/ruslanBik4/logs"
)

var ErrInvalidColumnCount = errors.New("invalid column count")
var ErrWrongColumnName = errors.New("wrong column name")

// CsvReader perform smart && fast import from CSV file
type CsvReader struct {
	*bufio.Reader
	Comma   rune
	Result  string
	Columns []string
	Table   dbEngine.Table

	IsPrefix bool
	line     []byte
	row      []any
	err      error
}

func NewCsvRows(f io.Reader, table dbEngine.Table) (c *CsvReader, err error) {
	c = &CsvReader{Reader: bufio.NewReader(f), Table: table}
	c.line, c.IsPrefix, err = c.ReadLine()
	if err != nil {
		logs.ErrorLog(err)
		return
	}

	c.Columns = strings.Split(gotools.BytesToString(c.line), ",")
	countColumns := len(table.Columns())
	if len(c.Columns) > countColumns {
		return nil, ErrInvalidColumnCount
	}

	if len(c.Columns) < countColumns {
		//	we think that used wrong comma (default ',')
		columns := strings.Split(strings.Join(c.Columns, ","), ";")
		if len(columns) > len(c.Columns) {
			c.Comma = ';'
			c.Columns = columns
			//c.FieldsPerRecord = len(c.columns)
		}
	}

	wrongColumns := slices.Collect(
		func(yield func(string2 string) bool) {

			for i, name := range c.Columns {
				if !slices.ContainsFunc(table.Columns(),
					func(col dbEngine.Column) bool {
						if col.Name() == name {
							return true
						}
						if col.Name() == strings.ToLower(name) {
							c.Columns[i] = strings.ToLower(name)
							return true
						}
						return false
					}) && !yield(name) {

					return
				}
			}
		})
	if len(wrongColumns) > 0 {
		err = ErrWrongColumnName
		c.Columns = wrongColumns
	}

	return
}
