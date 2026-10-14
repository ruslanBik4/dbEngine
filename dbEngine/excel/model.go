// Package excel reads Excel workbooks of any version (.xlsx/.xlsm/.xltx/.xltm,
// legacy .xls BIFF8, password protected files) and imports a worksheet into
// an existing table: the header may span several rows and need not be the
// first row, columns are matched to the table by name or by the original
// caption kept in the column comment, "category" rows fill the parent table
// referenced by a foreign key, and pictures are stored as files.
package excel

import (
	"fmt"
	"time"
)

// CellKind is the logical type of a cell value.
type CellKind uint8

const (
	CellEmpty CellKind = iota
	CellString
	CellNumber
	CellBool
	CellDate // date, time or date-time (see Cell.Time / Cell.TimeOnly)
	CellError
)

// Cell is a single, already-decoded worksheet value.
type Cell struct {
	Kind     CellKind
	Str      string      // CellString, CellError (e.g. "#N/A")
	Num      float64     // CellNumber
	Bool     bool        // CellBool
	Time     time.Time   // CellDate
	TimeOnly bool        // CellDate formatted as a time of day without a date
	Fmt      *CellFormat // visual style (shared between cells), may be nil
}

// CellFormat is the part of a cell style used to tell "special" rows (e.g.
// category rows painted with another colour) from ordinary data rows.
type CellFormat struct {
	Fill      string // fill colour / pattern, "" = no fill
	FontColor string
	Bold      bool
	Italic    bool
	FontSize  float64
}

// Key identifies the visual style; equal keys look the same.
func (f *CellFormat) Key() string {
	if f == nil {
		return ""
	}
	return fmt.Sprintf("%s|%s|%v|%v|%g", f.Fill, f.FontColor, f.Bold, f.Italic, f.FontSize)
}

func (c Cell) IsEmpty() bool {
	return c.Kind == CellEmpty || (c.Kind == CellString && isBlank(c.Str))
}

// Range is a 0-based inclusive cell rectangle.
type Range struct{ R1, C1, R2, C2 int }

func (r Range) Contains(row, col int) bool {
	return row >= r.R1 && row <= r.R2 && col >= r.C1 && col <= r.C2
}

// Image is a picture anchored (top-left) at Row/Col (0-based).
type Image struct {
	Row, Col int
	Data     []byte
	Ext      string // with dot: ".png", ".jpeg", ...
}

// TableHint is an Excel "Format as Table" (ListObject) definition.
type TableHint struct {
	Name       string
	Ref        Range
	HeaderRows int // 0 or 1
}

// Sheet is the format-independent representation of a worksheet.
type Sheet struct {
	Name   string
	Hidden bool
	Rows   [][]Cell // jagged: Rows[r][c]
	Merges []Range
	Images []Image
	Tables []TableHint
}

func (s *Sheet) set(r, c int, v Cell) {
	for len(s.Rows) <= r {
		s.Rows = append(s.Rows, nil)
	}
	row := s.Rows[r]
	if len(row) <= c {
		n := make([]Cell, c+1)
		copy(n, row)
		row = n
	}
	row[c] = v
	s.Rows[r] = row
}

// Workbook is the parsed workbook.
type Workbook struct {
	Sheets []*Sheet
}

func isBlank(s string) bool {
	for _, r := range s {
		switch r {
		case ' ', '\t', '\n', '\r', '\u00a0', '\u200b', '\ufeff':
		default:
			return false
		}
	}
	return true
}
