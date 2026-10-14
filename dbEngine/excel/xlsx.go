package excel

import (
	"bytes"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/xuri/excelize/v2"
)

// readXLSX reads an Office Open XML workbook (.xlsx, .xlsm, .xltx, .xltm).
func readXLSX(data []byte, password string) (*Workbook, error) {
	f, err := excelize.OpenReader(bytes.NewReader(data), excelize.Options{Password: password})
	if err != nil {
		if password != "" {
			return nil, fmt.Errorf("open encrypted xlsx: %w (%v)", ErrWrongPassword, err)
		}
		return nil, fmt.Errorf("open xlsx: %w", err)
	}
	defer f.Close()

	date1904 := false
	if p, err := f.GetWorkbookProps(); err == nil && p.Date1904 != nil {
		date1904 = *p.Date1904
	}

	type styleEntry struct {
		fi  fmtInfo
		fmt *CellFormat
	}
	styleCache := map[int]styleEntry{}
	styleInfo := func(idx int) styleEntry {
		if e, ok := styleCache[idx]; ok {
			return e
		}
		var e styleEntry
		if st, err := f.GetStyle(idx); err == nil && st != nil {
			code := ""
			if st.CustomNumFmt != nil {
				code = *st.CustomNumFmt
			}
			e.fi = classifyFormat(st.NumFmt, code)
			cf := &CellFormat{}
			if st.Fill.Type == "pattern" && st.Fill.Pattern > 0 && len(st.Fill.Color) > 0 {
				cf.Fill = strings.ToUpper(strings.Join(st.Fill.Color, ","))
			} else if st.Fill.Type == "gradient" {
				cf.Fill = "gradient:" + strings.ToUpper(strings.Join(st.Fill.Color, ","))
			}
			if st.Font != nil {
				cf.Bold, cf.Italic, cf.FontSize = st.Font.Bold, st.Font.Italic, st.Font.Size
				cf.FontColor = strings.ToUpper(st.Font.Color)
				if st.Font.ColorTheme != nil {
					cf.FontColor += fmt.Sprintf("theme%d", *st.Font.ColorTheme)
				}
			}
			e.fmt = cf
		}
		styleCache[idx] = e
		return e
	}

	wb := &Workbook{}
	for _, name := range f.GetSheetList() {
		sh := &Sheet{Name: name}
		if vis, err := f.GetSheetVisible(name); err == nil {
			sh.Hidden = !vis
		}
		rows, err := f.GetRows(name, excelize.Options{RawCellValue: true})
		if err != nil {
			// chart sheets / dialog sheets have no cells
			continue
		}
		styleIdx := func(ref string) int {
			idx, _ := f.GetCellStyle(name, ref)
			return idx
		}
		for r, row := range rows {
			for c, raw := range row {
				if raw == "" {
					continue
				}
				ref, _ := excelize.CoordinatesToCellName(c+1, r+1)
				typ, _ := f.GetCellType(name, ref)
				var cell Cell
				switch typ {
				case excelize.CellTypeSharedString, excelize.CellTypeInlineString, excelize.CellTypeFormula:
					cell = Cell{Kind: CellString, Str: raw}
				case excelize.CellTypeBool:
					cell = Cell{Kind: CellBool, Bool: raw == "1" || strings.EqualFold(raw, "true")}
				case excelize.CellTypeError:
					cell = Cell{Kind: CellError, Str: raw}
				case excelize.CellTypeDate:
					if t, ok := parseISOTime(raw); ok {
						cell = Cell{Kind: CellDate, Time: t}
					} else {
						cell = Cell{Kind: CellString, Str: raw}
					}
				default: // number or unset
					v, err := strconv.ParseFloat(raw, 64)
					if err != nil {
						cell = Cell{Kind: CellString, Str: raw}
						break
					}
					cell = numberCell(v, styleInfo(styleIdx(ref)).fi, date1904)
				}
				cell.Fmt = styleInfo(styleIdx(ref)).fmt
				sh.set(r, c, cell)
			}
		}

		if mcs, err := f.GetMergeCells(name, true); err == nil {
			for _, mc := range mcs {
				if rg, ok := parseRef(mc.GetStartAxis() + ":" + mc.GetEndAxis()); ok {
					sh.Merges = append(sh.Merges, rg)
				}
			}
		}

		if tbls, err := f.GetTables(name); err == nil {
			for _, t := range tbls {
				rg, ok := parseRef(t.Range)
				if !ok {
					continue
				}
				h := 1
				if t.ShowHeaderRow != nil && !*t.ShowHeaderRow {
					h = 0
				}
				sh.Tables = append(sh.Tables, TableHint{Name: t.Name, Ref: rg, HeaderRows: h})
			}
		}

		if cells, err := f.GetPictureCells(name); err == nil {
			seen := map[string]bool{}
			for _, ref := range cells {
				if seen[ref] {
					continue
				}
				seen[ref] = true
				c, r, err := excelize.CellNameToCoordinates(ref)
				if err != nil {
					continue
				}
				pics, err := f.GetPictures(name, ref)
				if err != nil {
					continue
				}
				for _, p := range pics {
					if len(p.File) == 0 {
						continue
					}
					ext := strings.ToLower(p.Extension)
					if ext == "" {
						ext = sniffImageExt(p.File)
					}
					sh.Images = append(sh.Images, Image{Row: r - 1, Col: c - 1, Data: p.File, Ext: ext})
				}
			}
		}
		wb.Sheets = append(wb.Sheets, sh)
	}
	return wb, nil
}

func parseISOTime(s string) (time.Time, bool) {
	for _, l := range []string{time.RFC3339Nano, "2006-01-02T15:04:05.999999999", "2006-01-02T15:04:05", "2006-01-02", "15:04:05"} {
		if t, err := time.Parse(l, s); err == nil {
			return t, true
		}
	}
	return time.Time{}, false
}

// parseRef parses "A1:C5" or "B2" into a 0-based Range.
func parseRef(ref string) (Range, bool) {
	ref = strings.ReplaceAll(ref, "$", "")
	parts := strings.SplitN(ref, ":", 2)
	c1, r1, err := excelize.CellNameToCoordinates(parts[0])
	if err != nil {
		return Range{}, false
	}
	c2, r2 := c1, r1
	if len(parts) == 2 {
		if c2, r2, err = excelize.CellNameToCoordinates(parts[1]); err != nil {
			return Range{}, false
		}
	}
	return Range{R1: r1 - 1, C1: c1 - 1, R2: r2 - 1, C2: c2 - 1}, true
}

func sniffImageExt(b []byte) string {
	switch {
	case len(b) > 8 && string(b[1:4]) == "PNG":
		return ".png"
	case len(b) > 3 && b[0] == 0xFF && b[1] == 0xD8:
		return ".jpeg"
	case len(b) > 6 && string(b[:3]) == "GIF":
		return ".gif"
	case len(b) > 2 && string(b[:2]) == "BM":
		return ".bmp"
	case len(b) > 4 && (string(b[:4]) == "II*\x00" || string(b[:4]) == "MM\x00*"):
		return ".tiff"
	case len(b) > 44 && string(b[40:44]) == " EMF":
		return ".emf"
	case len(b) > 4 && b[0] == 0xD7 && b[1] == 0xCD && b[2] == 0xC6 && b[3] == 0x9A:
		return ".wmf"
	case len(b) > 12 && string(b[:4]) == "RIFF" && string(b[8:12]) == "WEBP":
		return ".webp"
	case len(b) > 5 && (strings.HasPrefix(string(b[:5]), "<?xml") || strings.HasPrefix(string(b[:4]), "<svg")):
		return ".svg"
	}
	return ".bin"
}
