package excel

import (
	"math"
	"strings"
	"time"
)

// Built-in number format ids that represent dates / times.
// 14-22 and 45-47 are universal, 27-36 and 50-58 are locale (CJK) date formats.
func isBuiltinDateFmt(id int) bool {
	return (id >= 14 && id <= 22) || (id >= 27 && id <= 36) || (id >= 45 && id <= 47) || (id >= 50 && id <= 58)
}

func isBuiltinTimeOnlyFmt(id int) bool {
	return (id >= 18 && id <= 21) || (id >= 45 && id <= 47)
}

// fmtInfo tells whether a number format renders a date and whether it is a
// pure time-of-day format.
type fmtInfo struct{ date, timeOnly bool }

// classifyFormat inspects a number format by id and (optional) format code.
func classifyFormat(id int, code string) fmtInfo {
	if code == "" {
		if isBuiltinDateFmt(id) {
			return fmtInfo{true, isBuiltinTimeOnlyFmt(id)}
		}
		return fmtInfo{}
	}
	// Only the first section (positive numbers) matters.
	var b strings.Builder
	inQuote := false
	for i := 0; i < len(code); i++ {
		ch := code[i]
		switch {
		case inQuote:
			if ch == '"' {
				inQuote = false
			}
		case ch == '"':
			inQuote = true
		case ch == '\\' || ch == '_' || ch == '*':
			i++ // escaped / padding char
		case ch == '[':
			j := strings.IndexByte(code[i:], ']')
			if j < 0 {
				i = len(code)
				break
			}
			inner := strings.ToLower(code[i+1 : i+j])
			// elapsed time [h] [mm] [ss] is a time format; colours, locales, conditions are not
			if inner != "" && strings.Trim(inner, "hms") == "" {
				b.WriteByte('h')
			}
			i += j
		case ch == ';':
			i = len(code)
		default:
			b.WriteByte(ch)
		}
	}
	s := strings.ToLower(b.String())
	if s == "general" || s == "" {
		return fmtInfo{}
	}
	hasTime := strings.ContainsAny(s, "hs")
	hasDate := strings.ContainsAny(s, "dy") ||
		(strings.Contains(s, "m") && !hasTime && !strings.ContainsAny(s, "0#?")) // "mmm yyyy"/"mmmm"
	if !hasDate && !hasTime {
		return fmtInfo{}
	}
	return fmtInfo{date: true, timeOnly: hasTime && !hasDate}
}

// excelSerialToTime converts an Excel serial date to time.Time (UTC), rounded
// to milliseconds.
func excelSerialToTime(v float64, date1904 bool) time.Time {
	var base time.Time
	if date1904 {
		base = time.Date(1904, 1, 1, 0, 0, 0, 0, time.UTC)
	} else {
		base = time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
		if v < 60 { // Excel's fictitious 1900-02-29
			v++
		}
	}
	days := math.Floor(v)
	ms := math.Round((v - days) * 86400000)
	return base.AddDate(0, 0, int(days)).Add(time.Duration(ms) * time.Millisecond)
}

// numberCell builds a cell for a numeric value given its format info.
func numberCell(v float64, fi fmtInfo, date1904 bool) Cell {
	if fi.date && v >= 0 && v < 2958466 { // up to 9999-12-31
		return Cell{Kind: CellDate, Time: excelSerialToTime(v, date1904), TimeOnly: fi.timeOnly && v < 1}
	}
	return Cell{Kind: CellNumber, Num: v}
}
