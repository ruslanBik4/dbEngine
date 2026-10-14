package excel

import (
	"regexp"
	"strconv"
	"strings"
)

var numericTextRe = regexp.MustCompile(`^-?(0|[1-9][0-9]*)(\.[0-9]+)?$`)

// numericText reports whether a text cell is a clean number (no leading
// zeros, no thousands separators), so "00123" stays text.
func numericText(s string) (float64, bool) {
	s = strings.TrimSpace(s)
	if len(s) > 30 || !numericTextRe.MatchString(s) {
		return 0, false
	}
	v, err := strconv.ParseFloat(s, 64)
	return v, err == nil
}

// round15 mimics Excel's 15 significant digits display (0.1+0.2 -> 0.3).
func round15(v float64) float64 {
	r, err := strconv.ParseFloat(strconv.FormatFloat(v, 'g', 15, 64), 64)
	if err != nil {
		return v
	}
	return r
}

func formatNumber(v float64) string { return strconv.FormatFloat(round15(v), 'f', -1, 64) }

func formatAsText(c Cell) string {
	switch c.Kind {
	case CellString, CellError:
		return c.Str
	case CellNumber:
		return formatNumber(c.Num)
	case CellBool:
		if c.Bool {
			return "TRUE"
		}
		return "FALSE"
	case CellDate:
		switch {
		case c.TimeOnly:
			return formatTime(c)
		case isMidnight(c):
			return c.Time.Format("2006-01-02")
		default:
			return formatTimestamp(c)
		}
	}
	return ""
}

func isMidnight(c Cell) bool {
	t := c.Time
	return t.Hour() == 0 && t.Minute() == 0 && t.Second() == 0 && t.Nanosecond() == 0
}

func formatTime(c Cell) string {
	if c.Time.Nanosecond() != 0 {
		return strings.TrimRight(c.Time.Format("15:04:05.000"), "0")
	}
	return c.Time.Format("15:04:05")
}

func formatTimestamp(c Cell) string {
	if c.Time.Nanosecond() != 0 {
		return strings.TrimRight(c.Time.Format("2006-01-02 15:04:05.000"), "0")
	}
	return c.Time.Format("2006-01-02 15:04:05")
}
