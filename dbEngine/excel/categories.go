package excel

import "strings"

// category is a "group" row inside the data area: one filled text cell
// (usually merged across the table and/or painted with another colour) that
// applies to the following data rows.
type category struct {
	id     int
	parent *category
	level  int // depth, used only to build the hierarchy
	name   string
	row    int // 0-based worksheet row
	col    int
	style  string
}

// detectCategories splits the data rows into category rows and real data
// rows. Hierarchy comes from the style: a category row with a style already
// open on the stack closes that level (sibling), a new style opens a child
// level.
func detectCategories(sh *Sheet, reg region, rows []int, imgAt map[[2]int][]int) ([]*category, map[int]*category, []int) {
	type cand struct {
		row, col int
		cell     Cell
		wide     bool // horizontal merge over >= 2 table columns
	}
	inCols := map[int]bool{}
	for _, c := range reg.cols {
		inCols[c] = true
	}
	var cands []cand
	isCand := map[int]bool{}
	for _, r := range rows {
		n, col := 0, -1
		for _, c := range reg.cols {
			if !origCell(sh, r, c).IsEmpty() {
				n++
				col = c
			}
		}
		if n != 1 {
			continue
		}
		cell := origCell(sh, r, col)
		if cell.Kind != CellString {
			continue
		}
		wide := false
		for _, m := range sh.Merges {
			if m.R1 == r && m.C1 == col {
				w := 0
				for c := m.C1; c <= m.C2; c++ {
					if inCols[c] {
						w++
					}
				}
				wide = w >= 2
			}
		}
		cands = append(cands, cand{r, col, cell, wide})
		isCand[r] = true
	}
	if len(cands) == 0 || len(cands) == len(rows) {
		return nil, nil, rows
	}

	// dominant style of ordinary data cells: per column and overall
	perCol := map[int]map[string]int{}
	overall := map[string]int{}
	for _, r := range rows {
		if isCand[r] {
			continue
		}
		for _, c := range reg.cols {
			cell := origCell(sh, r, c)
			if cell.IsEmpty() {
				continue
			}
			k := cell.Fmt.Key()
			if perCol[c] == nil {
				perCol[c] = map[string]int{}
			}
			perCol[c][k]++
			overall[k]++
		}
	}
	dominant := func(m map[string]int) (string, int) {
		best, n, total := "", -1, 0
		for k, v := range m {
			total += v
			if v > n || (v == n && k < best) {
				best, n = k, v
			}
		}
		return best, total
	}
	gKey, _ := dominant(overall)

	var cats []*category
	catRow := map[int]bool{}
	for _, cd := range cands {
		key := cd.cell.Fmt.Key()
		ref := gKey
		if k, total := dominant(perCol[cd.col]); total >= 3 {
			ref = k
		}
		styled := key != ref
		if !cd.wide && !styled {
			continue // just a sparse data row
		}
		cats = append(cats, &category{row: cd.row, col: cd.col, name: strings.Join(strings.Fields(cellText(cd.cell)), " "), style: key})
		catRow[cd.row] = true
	}
	if len(cats) == 0 {
		return nil, nil, rows
	}

	// hierarchy + assignment of data rows
	var stack []*category
	catOf := map[int]*category{}
	var data []int
	ci := 0
	for _, r := range rows {
		if catRow[r] {
			c := cats[ci]
			ci++
			c.id = ci
			for i := len(stack) - 1; i >= 0; i-- {
				if stack[i].style == c.style {
					stack = stack[:i]
					break
				}
			}
			if len(stack) > 0 {
				c.parent = stack[len(stack)-1]
				c.level = c.parent.level + 1
			}
			stack = append(stack, c)
			continue
		}
		data = append(data, r)
		if len(stack) > 0 {
			catOf[r] = stack[len(stack)-1]
		}
	}
	return cats, catOf, data
}
