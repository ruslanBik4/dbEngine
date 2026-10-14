package excel

import (
	"fmt"
	"strings"
	"unicode"
	"unicode/utf8"
)

var translitTable = map[rune]string{
	'а': "a", 'б': "b", 'в': "v", 'г': "g", 'ґ': "g", 'д': "d", 'е': "e", 'ё': "e", 'є': "ye",
	'ж': "zh", 'з': "z", 'и': "i", 'і': "i", 'ї': "yi", 'й': "y", 'к': "k", 'л': "l", 'м': "m",
	'н': "n", 'о': "o", 'п': "p", 'р': "r", 'с': "s", 'т': "t", 'у': "u", 'ф': "f", 'х': "kh",
	'ц': "ts", 'ч': "ch", 'ш': "sh", 'щ': "shch", 'ъ': "", 'ы': "y", 'ь': "", 'э': "e", 'ю': "yu",
	'я': "ya", 'ў': "u",
}

var symbolWords = map[rune]string{'№': "no", '#': "no", '%': "pct", '$': "usd", '€': "eur", '₴': "uah", '₽': "rub"}

// makeIdent turns arbitrary header / sheet text into a lower-case snake_case
// PostgreSQL identifier (max 63 bytes). Unicode letters are kept unless
// transliterate is set, in which case Cyrillic is converted to Latin.
func makeIdent(s string, transliterate bool) string {
	return truncIdent(makeIdentFull(s, transliterate), 63)
}

// makeIdentFull is makeIdent without the 63 byte limit.
func makeIdentFull(s string, transliterate bool) string {
	var b strings.Builder
	us := false
	for _, r := range strings.ToLower(s) {
		if sym, ok := symbolWords[r]; ok {
			if !us && b.Len() > 0 {
				b.WriteByte('_')
			}
			b.WriteString(sym + "_")
			us = true
			continue
		}
		if transliterate {
			if t, ok := translitTable[r]; ok {
				b.WriteString(t)
				us = false
				continue
			}
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			if transliterate && r > unicode.MaxASCII {
				if !us {
					b.WriteByte('_')
					us = true
				}
				continue
			}
			b.WriteRune(r)
			us = false
		} else if !us {
			b.WriteByte('_')
			us = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out != "" && unicode.IsDigit([]rune(out)[0]) {
		out = "c_" + out
	}
	return out
}

func truncIdent(s string, max int) string {
	for len(s) > max {
		_, n := utf8.DecodeLastRuneInString(s)
		s = s[:len(s)-n]
	}
	return strings.TrimRight(s, "_")
}

// stopWords are dropped first when a column name has to be shortened.
var stopWords = map[string]bool{
	"з": true, "із": true, "зі": true, "від": true, "для": true, "на": true, "і": true, "й": true, "та": true,
	"в": true, "у": true, "до": true, "по": true, "с": true, "со": true, "от": true, "и": true, "к": true,
	"of": true, "the": true, "and": true, "for": true, "with": true, "from": true, "to": true, "in": true,
	"on": true, "a": true, "an": true, "by": true,
	"z": true, "iz": true, "vid": true, "dlya": true, "na": true, "i": true, "ta": true, "v": true, "u": true,
	"do": true, "po": true, "s": true, "ot": true,
}

// shrinkIdent shortens a snake_case name to at most max characters: first it
// drops stop words ("з", "від", "for"...), then cuts long words to 4, 3 and
// finally 2 letters (numbers are kept), and as a last resort truncates.
// max <= 0 leaves the name as is.
func shrinkIdent(name string, max int) string {
	if max <= 0 || utf8.RuneCountInString(name) <= max {
		return name
	}
	words := strings.Split(name, "_")
	join := func(ws []string) string { return strings.Trim(strings.Join(ws, "_"), "_") }
	var kept []string
	for i, w := range words {
		if i > 0 && stopWords[w] { // never drop the first word
			continue
		}
		kept = append(kept, w)
	}
	if s := join(kept); utf8.RuneCountInString(s) <= max {
		return s
	}
	for _, n := range []int{4, 3, 2} {
		cut := make([]string, len(kept))
		for i, w := range kept {
			r := []rune(w)
			if len(r) > n && !unicode.IsDigit(r[0]) {
				w = string(r[:n])
			}
			cut[i] = w
		}
		if s := join(cut); utf8.RuneCountInString(s) <= max {
			return s
		}
		kept = cut
	}
	r := []rune(join(kept))
	return strings.TrimRight(string(r[:max]), "_")
}

func colLetter(c int) string {
	s := ""
	for c++; c > 0; c = (c - 1) / 26 {
		s = string(rune('A'+(c-1)%26)) + s
	}
	return s
}

func cellRef(r, c int) string { return fmt.Sprintf("%s%d", colLetter(c), r+1) }
