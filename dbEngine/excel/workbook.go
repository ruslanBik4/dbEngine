package excel

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"strings"
)

// ErrUnsupportedFormat is returned for spreadsheets that are not Excel 97+
// workbooks (.xlsb, Excel 2-95, .ods ...): save them as .xlsx.
var ErrUnsupportedFormat = errors.New("unsupported spreadsheet format: save the file as .xlsx")

var (
	magicZip = []byte("PK\x03\x04")
	magicOLE = []byte{0xD0, 0xCF, 0x11, 0xE0, 0xA1, 0xB1, 0x1A, 0xE1}
)

// IsWorkbook reports whether data looks like an Excel workbook (an OOXML zip
// package or an OLE2 compound file), as opposed to CSV text.
func IsWorkbook(data []byte) bool {
	return bytes.HasPrefix(data, magicZip) || bytes.HasPrefix(data, magicOLE)
}

// Read parses a workbook held in memory. password opens encrypted
// workbooks (may be empty).
func Read(data []byte, password string) (*Workbook, error) {
	switch {
	case bytes.HasPrefix(data, magicZip):
		switch zipKind(data) {
		case "xlsx":
			return readXLSX(data, "")
		case "xlsb":
			return nil, fmt.Errorf("%w (.xlsb)", ErrUnsupportedFormat)
		}
		return nil, fmt.Errorf("%w (zip without xl/workbook.xml)", ErrUnsupportedFormat)

	case bytes.HasPrefix(data, magicOLE):
		wb, err := readXLS(data, password)
		switch {
		case err == nil:
			return wb, nil
		case errors.Is(err, errEncryptedOOXML):
			if password == "" {
				return nil, ErrPasswordRequired
			}
			wb, err := readXLSX(data, password)
			return wb, err
		case errors.Is(err, errNeedConversion):
			return nil, fmt.Errorf("%w (Excel 95 or older)", ErrUnsupportedFormat)
		default:
			return nil, err
		}
	}
	return nil, ErrUnsupportedFormat
}

func zipKind(data []byte) string {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return ""
	}
	for _, f := range zr.File {
		switch strings.ToLower(f.Name) {
		case "xl/workbook.xml":
			return "xlsx"
		case "xl/workbook.bin":
			return "xlsb"
		}
	}
	return "other"
}

func origCell(sh *Sheet, r, c int) Cell {
	if r >= 0 && r < len(sh.Rows) && c >= 0 && c < len(sh.Rows[r]) {
		return sh.Rows[r][c]
	}
	return Cell{}
}
