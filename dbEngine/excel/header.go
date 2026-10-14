package excel

import (
	"sort"
	"strings"
)

// region describes where the table lives inside a worksheet.
type region struct {
	hdrTop, hdrBot int // header rows (inclusive); hdrTop == -1 => no header
	dataStart      int
	dataEnd        int
	cols           []int    // worksheet column indexes that form the table
	names          []string // raw header caption per column (may be "")
}

// view is a worksheet with merged ranges expanded (each cell of a merge gets
// the top-left value), used for header analysis.
type view struct {
	rows    [][]Cell
	merges  []Range
	mergeAt map[[2]int]int // cell -> merge index, only for cells near the top
	images  map[int]map[int]bool
	lastRow int
	minCol  int
	maxCol  int
}

func newView(s *Sheet, scanRows int) *view {
	v := &view{merges: s.Merges, mergeAt: map[[2]int]int{}, images: map[int]map[int]bool{}}
	v.rows = make([][]Cell, len(s.Rows))
	for r, row := range s.Rows {
		v.rows[r] = append([]Cell(nil), row...)
	}
	for mi, m := range s.Merges {
		top := v.get(m.R1, m.C1)
		for r := m.R1; r <= m.R2; r++ {
			for c := m.C1; c <= m.C2; c++ {
				if r < scanRows+64 {
					v.mergeAt[[2]int{r, c}] = mi
				}
				if (r != m.R1 || c != m.C1) && !top.IsEmpty() {
					v.set(r, c, top)
				}
			}
		}
	}
	for _, im := range s.Images {
		if v.images[im.Row] == nil {
			v.images[im.Row] = map[int]bool{}
		}
		v.images[im.Row][im.Col] = true
	}
	v.lastRow, v.minCol, v.maxCol = -1, 1<<30, -1
	for r, row := range s.Rows { // use original cells for bounds
		for c, cell := range row {
			if !cell.IsEmpty() {
				v.extend(r, c)
			}
		}
	}
	for _, im := range s.Images {
		v.extend(im.Row, im.Col)
	}
	return v
}

func (v *view) extend(r, c int) {
	if r > v.lastRow {
		v.lastRow = r
	}
	if c < v.minCol {
		v.minCol = c
	}
	if c > v.maxCol {
		v.maxCol = c
	}
}

func (v *view) get(r, c int) Cell {
	if r < 0 || r >= len(v.rows) || c < 0 || c >= len(v.rows[r]) {
		return Cell{}
	}
	return v.rows[r][c]
}

func (v *view) set(r, c int, cell Cell) {
	for len(v.rows) <= r {
		v.rows = append(v.rows, nil)
	}
	if len(v.rows[r]) <= c {
		n := make([]Cell, c+1)
		copy(n, v.rows[r])
		v.rows[r] = n
	}
	v.rows[r][c] = cell
}

func (v *view) nonEmptyCols(r int) []int {
	var out []int
	if r < 0 || r >= len(v.rows) {
		return nil
	}
	for c, cell := range v.rows[r] {
		if !cell.IsEmpty() {
			out = append(out, c)
		}
	}
	return out
}

func (v *view) rowEmpty(r int) bool {
	return len(v.nonEmptyCols(r)) == 0 && len(v.images[r]) == 0
}

// hMerge returns the horizontal (multi-column) merge covering (r,c), if any.
func (v *view) hMerge(r, c int) (Range, bool) {
	if mi, ok := v.mergeAt[[2]int{r, c}]; ok {
		m := v.merges[mi]
		if m.C2 > m.C1 {
			return m, true
		}
	}
	return Range{}, false
}

func cellText(c Cell) string {
	if c.Kind == CellString {
		return strings.TrimSpace(c.Str)
	}
	return formatAsText(c)
}

// detectRegion finds the header block and data rows of a worksheet.
func detectRegion(v *view, maxScan int) (region, bool) {
	if v.lastRow < 0 {
		return region{}, false
	}
	scanEnd := v.lastRow
	if scanEnd > maxScan-1 {
		scanEnd = maxScan - 1
	}

	type cand struct {
		row   int
		score float64
	}
	var cands []cand
	best := -1e9
	for h := 0; h <= scanEnd; h++ {
		H := v.nonEmptyCols(h)
		if len(H) == 0 {
			continue
		}
		s := scoreHeader(v, h, H)
		cands = append(cands, cand{h, s})
		if s > best {
			best = s
		}
	}
	if len(cands) == 0 {
		return region{}, false
	}
	// Earliest row that is "almost as good" as the best one: protects against a
	// text data row out-scoring a header that has a few blank captions.
	h := cands[0].row
	thr := best - 0.6
	if best > 0 {
		thr = best * 0.85
	}
	for _, c := range cands {
		if c.score >= thr {
			h = c.row
			break
		}
	}

	var reg region
	if !rowHasText(v, h) {
		// first candidate is pure numbers/dates: the sheet has no header
		reg = region{hdrTop: -1, hdrBot: h - 1, dataStart: h, dataEnd: v.lastRow}
	} else {
		top, bot := expandHeader(v, h)
		reg = region{hdrTop: top, hdrBot: bot, dataStart: bot + 1, dataEnd: v.lastRow}
	}
	top := reg.hdrTop
	if top < 0 {
		top = reg.dataStart
	}

	// Columns: anything with a caption, data or image inside the table rows.
	colSet := map[int]bool{}
	for r := top; r <= v.lastRow; r++ {
		for _, c := range v.nonEmptyCols(r) {
			colSet[c] = true
		}
		if r >= reg.dataStart {
			for c := range v.images[r] {
				colSet[c] = true
			}
		}
	}
	for c := range colSet {
		reg.cols = append(reg.cols, c)
	}
	sort.Ints(reg.cols)
	reg.names = headerNames(v, reg.hdrTop, reg.hdrBot, reg.cols)
	return reg, true
}

// scoreHeader rates how likely row h is the (last) header row.
func scoreHeader(v *view, h int, H []int) float64 {
	// sample of data rows below
	sample := make([]int, 0, 25)
	for r := h + 1; r <= v.lastRow && len(sample) < 25; r++ {
		if !v.rowEmpty(r) {
			sample = append(sample, r)
		}
	}
	dataCols := map[int]bool{}
	nonText := map[int]int{}
	total := map[int]int{}
	for _, r := range sample {
		for _, c := range v.nonEmptyCols(r) {
			dataCols[c] = true
			total[c]++
			if k := v.get(r, c).Kind; k != CellString {
				nonText[c]++
			}
		}
		for c := range v.images[r] {
			dataCols[c] = true
		}
	}

	text, covered, contrast := 0, 0, 0
	uniq := map[string]bool{}
	units := map[int]bool{} // distinct merged areas / single cells
	for _, c := range H {
		if mi, ok := v.mergeAt[[2]int{h, c}]; ok {
			units[-1-mi] = true
		} else {
			units[c] = true
		}
		cell := v.get(h, c)
		isText := cell.Kind == CellString
		if isText {
			text++
		}
		uniq[strings.ToLower(cellText(cell))] = true
		if dataCols[c] {
			covered++
			if isText && total[c] > 0 && nonText[c]*2 > total[c] {
				contrast++
			}
		}
	}
	nH := float64(len(H))
	textRatio := float64(text) / nH
	uniqRatio := float64(len(uniq)) / nH
	if len(dataCols) == 0 { // last row / header without data
		return 2*textRatio + 0.5*uniqRatio - 2
	}
	covRatio := float64(covered) / float64(len(dataCols))
	precision := float64(covered) / nH
	contrastRatio := 0.0
	if covered > 0 {
		contrastRatio = float64(contrast) / float64(covered)
	}
	score := 3*covRatio + precision + 2*textRatio + 1.5*contrastRatio + 0.5*uniqRatio
	if len(units) == 1 && len(dataCols) > 1 {
		score -= 3 // title / caption line (single cell or one wide merge)
	}
	return score
}

// expandHeader grows the header block around row h: vertical merges, merged
// or sparse "group" rows above and sub-header rows below group rows.
func expandHeader(v *view, h int) (int, int) {
	top, bot := h, h
	hCols := v.nonEmptyCols(h)
	if len(hCols) == 0 {
		return top, bot
	}
	minC, maxC := hCols[0], hCols[len(hCols)-1]
	width := maxC - minC + 1

	vertical := func() bool {
		changed := false
		for _, m := range v.merges {
			if m.C2 < minC || m.C1 > maxC || m.R2 == m.R1 {
				continue
			}
			// merge crossing the header block boundary
			if m.R1 <= bot && m.R2 > bot && m.R1 >= top-1 && m.R2-bot <= 3 {
				bot = m.R2
				changed = true
			}
			if m.R2 >= top && m.R1 < top && top-m.R1 <= 3 {
				top = m.R1
				changed = true
			}
		}
		return changed
	}
	allText := func(r int, cols []int) bool {
		for _, c := range cols {
			if v.get(r, c).Kind != CellString {
				return false
			}
		}
		return true
	}

	for iter := 0; iter < 8; iter++ {
		changed := vertical()

		// group row(s) above the block
		if u := top - 1; u >= 0 {
			U := v.nonEmptyCols(u)
			if len(U) > 0 && allText(u, U) && U[0] >= minC-1 && U[len(U)-1] <= maxC {
				hasMerge := false
				spanTitle := false
				for _, c := range U {
					if m, ok := v.hMerge(u, c); ok {
						hasMerge = true
						if m.C2-m.C1+1 >= width*8/10 && width > 2 {
							spanTitle = true
						}
					}
				}
				distinct := map[string]bool{}
				for _, c := range U {
					distinct[cellText(v.get(u, c))] = true
				}
				isTitle := spanTitle && len(distinct) == 1 || (!hasMerge && len(U) == 1 && width > 2 && U[0] == minC && len(v.nonEmptyCols(top)) >= width*8/10)
				// a single un-merged label ("October 2026") is a caption, not a group:
				// it goes to the table comment instead of every column name
				groupPattern := !hasMerge && len(U) >= 2 && len(U) < len(v.nonEmptyCols(top)) && sparseLabels(v, u, U, maxC)
				if !isTitle && (hasMerge || groupPattern) {
					top = u
					changed = true
				}
			}
		}

		// sub-header row below a row with horizontal merges
		if b := bot + 1; b <= v.lastRow {
			B := v.nonEmptyCols(b)
			if len(B) > 0 && allText(b, B) {
				for _, c := range v.nonEmptyCols(bot) {
					m, ok := v.hMerge(bot, c)
					if !ok || m.R2 > bot {
						continue
					}
					under := map[string]bool{}
					for cc := m.C1; cc <= m.C2; cc++ {
						if t := cellText(v.get(b, cc)); t != "" {
							under[t] = true
						}
					}
					if len(under) >= 2 {
						bot = b
						changed = true
						break
					}
				}
			}
		}
		if !changed {
			break
		}
	}
	return top, bot
}

// sparseLabels: every label in row u is followed by at least one blank cell
// (group caption pattern "Q1 _ Q2 _"), except possibly the last one.
func sparseLabels(v *view, u int, U []int, maxC int) bool {
	if len(U) == 0 {
		return false
	}
	for i, c := range U {
		next := maxC + 1
		if i+1 < len(U) {
			next = U[i+1]
		}
		if next-c < 2 && i+1 < len(U) {
			return false
		}
	}
	return true
}

// headerNames builds the caption of each column from the header block,
// joining the levels of multi-row headers with "_".
func headerNames(v *view, top, bot int, cols []int) []string {
	names := make([]string, len(cols))
	if top < 0 {
		return names
	}
	for i, c := range cols {
		var parts []string
		for r := top; r <= bot; r++ {
			t := cellText(v.get(r, c))
			if t == "" && r < bot && !rowHasMerges(v, r, cols) {
				// sparse group row without merges: inherit the caption on the left
				for j := i - 1; j >= 0; j-- {
					if lt := cellText(v.get(r, cols[j])); lt != "" {
						t = lt
						break
					}
				}
				if t != "" && cellText(v.get(bot, c)) == "" {
					t = ""
				}
			}
			if t != "" && (len(parts) == 0 || parts[len(parts)-1] != t) {
				parts = append(parts, t)
			}
		}
		names[i] = strings.Join(parts, " ")
	}
	return names
}

func rowHasMerges(v *view, r int, cols []int) bool {
	for _, c := range cols {
		if _, ok := v.mergeAt[[2]int{r, c}]; ok {
			return true
		}
	}
	return false
}

// regionFromHint builds a region from an Excel table (ListObject) definition.
func regionFromHint(v *view, t TableHint) region {
	reg := region{hdrTop: -1, hdrBot: t.Ref.R1 - 1, dataStart: t.Ref.R1, dataEnd: t.Ref.R2}
	if t.HeaderRows > 0 {
		reg.hdrTop, reg.hdrBot, reg.dataStart = t.Ref.R1, t.Ref.R1, t.Ref.R1+1
	}
	for c := t.Ref.C1; c <= t.Ref.C2; c++ {
		reg.cols = append(reg.cols, c)
	}
	// images just to the right of the table on data rows
	extra := map[int]bool{}
	for r := reg.dataStart; r <= reg.dataEnd; r++ {
		for c := range v.images[r] {
			if c > t.Ref.C2 {
				extra[c] = true
			}
		}
	}
	for c := range extra {
		reg.cols = append(reg.cols, c)
	}
	sort.Ints(reg.cols)
	reg.names = headerNames(v, reg.hdrTop, reg.hdrBot, reg.cols)
	return reg
}

func rowHasText(v *view, r int) bool {
	for _, c := range v.nonEmptyCols(r) {
		if v.get(r, c).Kind == CellString {
			return true
		}
	}
	return false
}
