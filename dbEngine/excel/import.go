package excel

import (
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"errors"
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"time"
	"unicode"

	"github.com/jackc/pgx/v5"

	"github.com/ruslanBik4/dbEngine/dbEngine"
)

// DefaultImageViewURL renders a picture link appended to text when a cell
// holds both text and a picture (same template as the converter uses).
const DefaultImageViewURL = `<a href='%s' data-fancybox hx-swap='none' ><img src='%[1]s'></a>`

// ErrNoColumns is returned when no worksheet column matches the table.
var ErrNoColumns = errors.New("no column of the workbook matches the table")

// Options of Import.
type Options struct {
	// Password opens encrypted workbooks.
	Password string
	// SheetName imports this worksheet; by default the sheet whose columns
	// match the table best.
	SheetName string
	// ImageDir is where pictures are written ("" = pictures are skipped).
	ImageDir string
	// ImagePathPrefix is stored in the table instead of the file system path:
	// prefix + "/" + file name, e.g. "/img/tikkurila".
	ImagePathPrefix string
	// ImageViewURL renders pictures found in text columns (DefaultImageViewURL).
	ImageViewURL string
	// MaxHeaderScan is how many top rows may hold the header (default 50).
	MaxHeaderScan int
}

// Match tells which worksheet column filled which table column.
type Match struct {
	Caption     string `json:"caption"`
	ExcelColumn string `json:"excel_column"`
	Column      string `json:"column"`
	By          string `json:"by"` // name, comment, ident, translit
}

// ParentStat counts rows of a parent (foreign key) table.
type ParentStat struct {
	Table   string `json:"table"`
	Column  string `json:"column"` // foreign key column of the imported table
	Found   int    `json:"found"`  // existing rows reused
	Created int    `json:"created"`
}

// Result of Import.
type Result struct {
	Sheet      string       `json:"sheet"`
	HeaderRows [2]int       `json:"header_rows"` // 1-based, {0,0} = no header
	Rows       int64        `json:"rows"`
	Matched    []Match      `json:"matched"`
	Unmatched  []string     `json:"unmatched,omitempty"` // worksheet captions not imported
	Missing    []string     `json:"missing,omitempty"`   // table columns not in the file
	Parents    []ParentStat `json:"parents,omitempty"`
	Images     int          `json:"images"`
	Warnings   []string     `json:"warnings,omitempty"`
}

// Summary is a one-line human readable result.
func (r *Result) Summary() string {
	s := fmt.Sprintf("sheet %q: %d rows imported, %d columns matched", r.Sheet, r.Rows, len(r.Matched))
	for _, p := range r.Parents {
		s += fmt.Sprintf(", %s: %d found, %d created", p.Table, p.Found, p.Created)
	}
	if len(r.Unmatched) > 0 {
		s += ", skipped: " + strings.Join(r.Unmatched, ", ")
	}
	return s
}

// Import appends the rows of a workbook (any Excel version) to table.
// Worksheet columns are matched to table columns by name, by the original
// caption kept in the column comment or by the converted (snake_case,
// transliterated, shortened) name, so a file with renamed or reordered
// columns is still imported. Identity / serial columns are never filled.
//
// Foreign keys split the data: "category" rows of the worksheet (one filled,
// differently styled cell) and worksheet columns matched to a foreign key
// column are looked up by name in the parent table and inserted there when
// missing (keeping parent_id hierarchy); the imported rows get their ids.
//
// Everything runs in one transaction: either all rows are imported or none.
func Import(ctx context.Context, conn *pgx.Conn, table dbEngine.Table, data []byte, opts Options) (*Result, error) {
	return ImportTable(ctx, conn, table.Name(), data, opts)
}

// ImportTable is Import for a table given by name.
func ImportTable(ctx context.Context, conn *pgx.Conn, tableName string, data []byte, opts Options) (*Result, error) {
	if opts.MaxHeaderScan <= 0 {
		opts.MaxHeaderScan = 50
	}
	if opts.ImageViewURL == "" {
		opts.ImageViewURL = DefaultImageViewURL
	}
	wb, err := Read(data, opts.Password)
	if err != nil {
		return nil, err
	}

	tx, err := conn.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	cols, err := loadColumns(ctx, tx, tableName)
	if err != nil {
		return nil, err
	}
	if len(cols) == 0 {
		return nil, fmt.Errorf("table %q not found", tableName)
	}

	p, err := newPlan(wb, cols, opts)
	if err != nil {
		return nil, err
	}
	if err := p.resolveParents(ctx, tx); err != nil {
		return nil, err
	}
	n, err := p.copyRows(ctx, tx, tableName)
	if err != nil {
		return nil, err
	}
	p.res.Rows = n
	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}
	return p.res, nil
}

// ---------------- table schema ----------------

type dbCol struct {
	name     string
	udt      string // pg type name: varchar, int4, numeric, bool...
	nullable bool
	auto     bool // identity, serial or generated: never filled from the file
	comment  string
	fk       *fkRef
}

type fkRef struct {
	parent    string // parent table (regclass text, already quoted if needed)
	parentCol string // referenced column, e.g. id
}

func (c *dbCol) isText() bool {
	switch c.udt {
	case "varchar", "text", "bpchar", "name", "citext":
		return true
	}
	return false
}

func (c *dbCol) isImage() bool { return strings.Contains(c.comment, `"view_url"`) }

// caption is the column comment without the JSON tail ({"view_url": ...}).
func (c *dbCol) caption() string {
	s := c.comment
	if i := strings.Index(s, " {\""); i >= 0 {
		s = s[:i]
	}
	return strings.TrimSpace(s)
}

type querier interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

func loadColumns(ctx context.Context, q querier, table string) ([]*dbCol, error) {
	rows, err := q.Query(ctx, `
SELECT a.attname, t.typname, NOT a.attnotnull,
       a.attidentity <> '' OR a.attgenerated <> '' OR COALESCE(pg_get_expr(d.adbin, d.adrelid), '') LIKE 'nextval(%',
       COALESCE(col_description(a.attrelid, a.attnum), ''),
       COALESCE(fk.parent, ''), COALESCE(fk.parent_col, '')
FROM pg_attribute a
JOIN pg_type t ON t.oid = a.atttypid
LEFT JOIN pg_attrdef d ON d.adrelid = a.attrelid AND d.adnum = a.attnum
LEFT JOIN LATERAL (
    SELECT c.confrelid::regclass::text AS parent, pa.attname AS parent_col
    FROM pg_constraint c
    JOIN pg_attribute pa ON pa.attrelid = c.confrelid AND pa.attnum = c.confkey[1]
    WHERE c.contype = 'f' AND c.conrelid = a.attrelid AND c.conkey = ARRAY[a.attnum]
    LIMIT 1
) fk ON true
WHERE a.attrelid = to_regclass($1) AND a.attnum > 0 AND NOT a.attisdropped
ORDER BY a.attnum`, table)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var cols []*dbCol
	for rows.Next() {
		c := &dbCol{}
		var parent, parentCol string
		if err := rows.Scan(&c.name, &c.udt, &c.nullable, &c.auto, &c.comment, &parent, &parentCol); err != nil {
			return nil, err
		}
		if parent != "" {
			c.fk = &fkRef{parent: parent, parentCol: parentCol}
		}
		cols = append(cols, c)
	}
	return cols, rows.Err()
}

func findCol(cols []*dbCol, name string) *dbCol {
	for _, c := range cols {
		if c.name == name {
			return c
		}
	}
	return nil
}

// ---------------- column matching ----------------

// norms returns the comparable forms of a caption or column name.
func norms(s string) []string {
	var out []string
	add := func(v string) {
		if v != "" && !slices.Contains(out, v) {
			out = append(out, v)
		}
	}
	for _, tr := range []bool{false, true} {
		full := makeIdentFull(s, tr)
		add(full)
		add(truncIdent(full, 63))
		add(truncIdent(shrinkIdent(full, 30), 63))
	}
	return out
}

// categoryWords are captions of a "category" column in several languages.
var categoryWords = map[string]bool{
	"category": true, "categories": true, "group": true, "section": true,
	"категорія": true, "категорії": true, "категория": true, "категории": true,
	"група": true, "группа": true, "розділ": true, "раздел": true,
}

func compact(s string) string {
	return strings.Map(func(r rune) rune {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			return unicode.ToLower(r)
		}
		return -1
	}, s)
}

// matchColumn finds the table column for a worksheet caption.
func matchColumn(caption string, cols []*dbCol, used map[*dbCol]bool) (*dbCol, string) {
	if strings.TrimSpace(caption) == "" {
		return nil, ""
	}
	capNorms := norms(caption)
	capCompact := compact(caption)
	try := func(by string, ok func(c *dbCol) bool) (*dbCol, string) {
		for _, c := range cols {
			if !used[c] && !c.auto && ok(c) {
				return c, by
			}
		}
		return nil, ""
	}
	if c, by := try("name", func(c *dbCol) bool { return strings.EqualFold(c.name, strings.TrimSpace(caption)) }); c != nil {
		return c, by
	}
	if c, by := try("comment", func(c *dbCol) bool {
		cp := c.caption()
		return cp != "" && (strings.EqualFold(strings.Join(strings.Fields(cp), " "), strings.Join(strings.Fields(caption), " ")) ||
			compact(cp) == capCompact)
	}); c != nil {
		return c, by
	}
	if c, by := try("ident", func(c *dbCol) bool {
		if slices.Contains(capNorms, c.name) {
			return true
		}
		if c.fk != nil { // "Category" -> category_id, "Категорія" -> category_id
			base := strings.TrimSuffix(c.name, "_id")
			if slices.Contains(capNorms, base) {
				return true
			}
			if strings.HasPrefix(base, "categor") || strings.Contains(strings.ToLower(c.fk.parent), "categor") {
				return categoryWords[compact(caption)]
			}
		}
		return false
	}); c != nil {
		return c, by
	}
	return try("compact", func(c *dbCol) bool { return capCompact != "" && compact(c.name) == capCompact })
}

// ---------------- plan ----------------

type outCol struct {
	col     *dbCol
	src     int    // worksheet column, -1 = generated
	gen     string // "category" (from category rows), "excel_row"
	textOf  *dbCol // image column whose cell text goes here (<col>_text)
	caption string
}

type plan struct {
	opts    Options
	sh      *Sheet
	v       *view
	reg     region
	cols    []*dbCol
	out     []outCol
	rows    []int
	imgAt   map[[2]int][]int
	cats    []*category
	catOf   map[int]*category
	catFK   *dbCol
	catIDs  map[*category]int64
	fkNames map[*dbCol]map[string]int64 // name -> id for worksheet columns mapped to foreign keys
	saved   map[int]string
	res     *Result
}

func newPlan(wb *Workbook, cols []*dbCol, opts Options) (*plan, error) {
	type cand struct {
		sh      *Sheet
		v       *view
		reg     region
		matches int
	}
	var best *cand
	var captions []string
	for _, sh := range wb.Sheets {
		if opts.SheetName != "" && sh.Name != opts.SheetName {
			continue
		}
		v := newView(sh, opts.MaxHeaderScan)
		var reg region
		if len(sh.Tables) > 0 {
			reg = regionFromHint(v, sh.Tables[0])
		} else if r, ok := detectRegion(v, opts.MaxHeaderScan); ok {
			reg = r
		} else {
			continue
		}
		used := map[*dbCol]bool{}
		n := 0
		for _, name := range reg.names {
			if c, _ := matchColumn(name, cols, used); c != nil {
				used[c] = true
				n++
			}
		}
		captions = append(captions, reg.names...)
		if best == nil || n > best.matches {
			best = &cand{sh, v, reg, n}
		}
	}
	if best == nil || best.matches == 0 {
		var names []string
		for _, c := range cols {
			names = append(names, c.name)
		}
		return nil, fmt.Errorf("%w: worksheet captions [%s], table columns [%s]", ErrNoColumns,
			strings.Join(nonEmpty(captions), ", "), strings.Join(names, ", "))
	}

	p := &plan{opts: opts, sh: best.sh, v: best.v, reg: best.reg, cols: cols, saved: map[int]string{},
		catIDs: map[*category]int64{}, fkNames: map[*dbCol]map[string]int64{},
		res: &Result{Sheet: best.sh.Name}}
	if p.reg.hdrTop >= 0 {
		p.res.HeaderRows = [2]int{p.reg.hdrTop + 1, p.reg.hdrBot + 1}
	}

	// worksheet column -> table column
	used := map[*dbCol]bool{}
	for i, c := range p.reg.cols {
		caption := p.reg.names[i]
		col, by := matchColumn(caption, cols, used)
		if col == nil {
			if caption != "" {
				p.res.Unmatched = append(p.res.Unmatched, caption)
			}
			continue
		}
		used[col] = true
		oc := outCol{col: col, src: c, caption: caption}
		if col.isImage() { // text written over the pictures -> <col>_text
			if t := findCol(cols, col.name+"_text"); t != nil && !used[t] && t.isText() {
				oc.textOf = t
				used[t] = true
			}
		}
		p.out = append(p.out, oc)
		p.res.Matched = append(p.res.Matched, Match{Caption: caption, ExcelColumn: colLetter(c), Column: col.name, By: by})
	}

	// data rows, pictures, category rows
	p.collectRows()
	p.catFK = pickCategoryFK(cols, used)
	if len(p.cats) > 0 {
		if p.catFK == nil {
			p.res.Warnings = append(p.res.Warnings, fmt.Sprintf("%d category rows skipped: the table has no foreign key for them", len(p.cats)))
		} else {
			used[p.catFK] = true
			p.out = append(p.out, outCol{col: p.catFK, src: -1, gen: "category"})
		}
	}
	if c := findCol(cols, "excel_row"); c != nil && !used[c] && !c.auto && strings.HasPrefix(c.udt, "int") {
		used[c] = true
		p.out = append(p.out, outCol{col: c, src: -1, gen: "excel_row"})
	}
	for _, c := range cols {
		if !used[c] && !c.auto && c.name != "is_visible" {
			p.res.Missing = append(p.res.Missing, c.name)
		}
	}
	return p, nil
}

func nonEmpty(ss []string) []string {
	var out []string
	for _, s := range ss {
		if strings.TrimSpace(s) != "" {
			out = append(out, s)
		}
	}
	return out
}

// pickCategoryFK chooses the foreign key that receives the category rows:
// a parent named *categor*, then a key named category*, then the first one.
func pickCategoryFK(cols []*dbCol, used map[*dbCol]bool) *dbCol {
	var fks []*dbCol
	for _, c := range cols {
		if c.fk != nil && !used[c] && !c.auto {
			fks = append(fks, c)
		}
	}
	for _, c := range fks {
		if strings.Contains(strings.ToLower(c.fk.parent), "categor") {
			return c
		}
	}
	for _, c := range fks {
		if strings.HasPrefix(c.name, "categor") {
			return c
		}
	}
	if len(fks) > 0 {
		return fks[0]
	}
	return nil
}

func (p *plan) src(r, c int) Cell { return p.v.get(r, c) } // merged values repeated

func (p *plan) collectRows() {
	sh, reg := p.sh, p.reg
	inCols := map[int]bool{}
	for _, c := range reg.cols {
		inCols[c] = true
	}
	p.imgAt = map[[2]int][]int{}
	for i, im := range sh.Images {
		if im.Row < reg.dataStart || im.Row > reg.dataEnd || !inCols[im.Col] {
			continue
		}
		// one photo for several rows (package sizes of one product)
		for _, r := range pictureRows(sh, i, reg.dataStart, reg.dataEnd, inCols) {
			p.imgAt[[2]int{r, im.Col}] = append(p.imgAt[[2]int{r, im.Col}], i)
		}
	}
	var all []int
	for r := reg.dataStart; r <= reg.dataEnd; r++ {
		for _, c := range reg.cols {
			if !origCell(sh, r, c).IsEmpty() || !p.src(r, c).IsEmpty() || len(p.imgAt[[2]int{r, c}]) > 0 {
				all = append(all, r)
				break
			}
		}
	}
	p.cats, p.catOf, p.rows = detectCategories(sh, reg, all, p.imgAt)
}

// ---------------- parent tables ----------------

type parentTable struct {
	name      string
	key       string
	nameCol   string
	parentCol string // "" = flat list
	imageCol  string
	rowCol    string
	byKey     map[string]int64 // parentID|lower(name) -> id
	byName    map[string]int64 // lower(name) -> first id
	stat      *ParentStat
}

func (p *plan) parent(ctx context.Context, tx pgx.Tx, fk *dbCol) (*parentTable, error) {
	cols, err := loadColumns(ctx, tx, fk.fk.parent)
	if err != nil {
		return nil, err
	}
	pt := &parentTable{name: fk.fk.parent, key: fk.fk.parentCol, byKey: map[string]int64{}, byName: map[string]int64{}}
	if c := findCol(cols, "name"); c != nil && c.isText() {
		pt.nameCol = c.name
	} else {
		for _, c := range cols {
			if c.isText() && c.fk == nil && !c.auto && !c.isImage() {
				pt.nameCol = c.name
				break
			}
		}
	}
	if pt.nameCol == "" {
		return nil, fmt.Errorf("parent table %s has no text column for names", pt.name)
	}
	if c := findCol(cols, "parent_id"); c != nil && strings.HasPrefix(c.udt, "int") {
		pt.parentCol = c.name
	}
	if c := findCol(cols, "image"); c != nil && c.isText() {
		pt.imageCol = c.name
	}
	if c := findCol(cols, "excel_row"); c != nil && strings.HasPrefix(c.udt, "int") {
		pt.rowCol = c.name
	}
	parentExpr := "0"
	if pt.parentCol != "" {
		parentExpr = "COALESCE(" + qi(pt.parentCol) + ", 0)"
	}
	rows, err := tx.Query(ctx, fmt.Sprintf(`SELECT %s::bigint, %s::text, %s::bigint FROM %s`,
		qi(pt.key), qi(pt.nameCol), parentExpr, pt.name))
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	for rows.Next() {
		var id, pid int64
		var name string
		if err := rows.Scan(&id, &name, &pid); err != nil {
			return nil, err
		}
		k := normName(name)
		pt.byKey[fmt.Sprint(pid, "|", k)] = id
		if _, ok := pt.byName[k]; !ok {
			pt.byName[k] = id
		}
	}
	pt.stat = &ParentStat{Table: strings.Trim(pt.name, `"`), Column: fk.name}
	return pt, rows.Err()
}

func normName(s string) string { return strings.ToLower(strings.Join(strings.Fields(s), " ")) }

// ensure returns the id of name under parent pid, inserting the row when missing.
func (pt *parentTable) ensure(ctx context.Context, tx pgx.Tx, name string, pid int64, image string, row int) (int64, error) {
	k := normName(name)
	if pt.parentCol == "" {
		pid = 0
	}
	if id, ok := pt.byKey[fmt.Sprint(pid, "|", k)]; ok {
		pt.stat.Found++
		return id, nil
	}
	cols, vals := []string{qi(pt.nameCol)}, []any{strings.Join(strings.Fields(name), " ")}
	if pt.parentCol != "" {
		cols, vals = append(cols, qi(pt.parentCol)), append(vals, pid)
	}
	if pt.imageCol != "" && image != "" {
		cols, vals = append(cols, qi(pt.imageCol)), append(vals, image)
	}
	if pt.rowCol != "" && row > 0 {
		cols, vals = append(cols, qi(pt.rowCol)), append(vals, row)
	}
	ph := make([]string, len(vals))
	for i := range ph {
		ph[i] = fmt.Sprintf("$%d", i+1)
	}
	var id int64
	err := tx.QueryRow(ctx, fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s) RETURNING %s::bigint",
		pt.name, strings.Join(cols, ", "), strings.Join(ph, ", "), qi(pt.key)), vals...).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("insert into %s: %w", pt.name, err)
	}
	pt.byKey[fmt.Sprint(pid, "|", k)] = id
	if _, ok := pt.byName[k]; !ok {
		pt.byName[k] = id
	}
	pt.stat.Created++
	return id, nil
}

func (p *plan) resolveParents(ctx context.Context, tx pgx.Tx) error {
	parents := map[*dbCol]*parentTable{}
	get := func(fk *dbCol) (*parentTable, error) {
		if pt, ok := parents[fk]; ok {
			return pt, nil
		}
		pt, err := p.parent(ctx, tx, fk)
		if err != nil {
			return nil, err
		}
		parents[fk] = pt
		return pt, nil
	}
	// category rows (parents first: they appear before their children)
	if len(p.cats) > 0 && p.catFK != nil {
		pt, err := get(p.catFK)
		if err != nil {
			return err
		}
		for _, c := range p.cats {
			var pid int64
			if c.parent != nil {
				pid = p.catIDs[c.parent]
			}
			image := ""
			for _, col := range p.reg.cols {
				if idx := p.imgAt[[2]int{c.row, col}]; len(idx) > 0 {
					if paths := p.saveImages(idx); len(paths) > 0 {
						image = strings.Join(paths, ";")
					}
				}
			}
			id, err := pt.ensure(ctx, tx, c.name, pid, image, c.row+1)
			if err != nil {
				return err
			}
			p.catIDs[c] = id
		}
	}
	// worksheet columns matched to a foreign key: values are parent names
	for _, oc := range p.out {
		if oc.src < 0 || oc.col.fk == nil {
			continue
		}
		pt, err := get(oc.col)
		if err != nil {
			return err
		}
		names := map[string]int64{}
		for _, r := range p.rows {
			cell := p.src(r, oc.src)
			if cell.IsEmpty() || cell.Kind == CellNumber {
				continue // numbers are ids
			}
			name := strings.TrimSpace(formatAsText(cell))
			k := normName(name)
			if _, ok := names[k]; ok {
				continue
			}
			id, ok := pt.byName[k]
			if ok {
				pt.stat.Found++
			} else if id, err = pt.ensure(ctx, tx, name, 0, "", 0); err != nil {
				return err
			}
			names[k] = id
		}
		p.fkNames[oc.col] = names
	}
	for _, c := range p.cols {
		if pt, ok := parents[c]; ok {
			p.res.Parents = append(p.res.Parents, *pt.stat)
		}
	}
	return nil
}

// ---------------- pictures ----------------

// saveImages writes pictures once each (file name = content hash, so a new
// import never overwrites pictures of rows imported earlier).
func (p *plan) saveImages(idxs []int) []string {
	var paths []string
	for _, i := range idxs {
		if path, ok := p.saved[i]; ok {
			paths = append(paths, path)
			continue
		}
		if p.opts.ImageDir == "" {
			if len(p.saved) == 0 {
				p.res.Warnings = append(p.res.Warnings, "pictures skipped: no image directory configured")
			}
			p.saved[i] = ""
			continue
		}
		im := p.sh.Images[i]
		ext := im.Ext
		if ext == "" || ext == ".bin" {
			ext = sniffImageExt(im.Data)
		}
		sum := sha1.Sum(im.Data)
		name := hex.EncodeToString(sum[:])[:16] + ext
		full := filepath.Join(p.opts.ImageDir, name)
		if err := os.MkdirAll(p.opts.ImageDir, 0o755); err == nil {
			if _, err := os.Stat(full); err != nil {
				err = os.WriteFile(full, im.Data, 0o644)
				if err != nil {
					p.res.Warnings = append(p.res.Warnings, fmt.Sprintf("picture %s: %v", cellRef(im.Row, im.Col), err))
					p.saved[i] = ""
					continue
				}
			}
		}
		path := filepath.ToSlash(full)
		if p.opts.ImagePathPrefix != "" {
			path = strings.TrimRight(p.opts.ImagePathPrefix, "/") + "/" + name
		}
		p.saved[i] = path
		p.res.Images++
		paths = append(paths, path)
	}
	return slices.DeleteFunc(paths, func(s string) bool { return s == "" })
}

// ---------------- values + COPY ----------------

func (p *plan) copyRows(ctx context.Context, tx pgx.Tx, table string) (int64, error) {
	if len(p.rows) == 0 || len(p.out) == 0 {
		return 0, nil
	}
	var names []string
	for _, oc := range p.out {
		names = append(names, qi(oc.col.name))
		if oc.textOf != nil {
			names = append(names, qi(oc.textOf.name))
		}
	}
	var buf bytes.Buffer
	for _, r := range p.rows {
		first := true
		put := func(v *string) {
			if !first {
				buf.WriteByte(',')
			}
			first = false
			if v != nil {
				buf.WriteByte('"')
				buf.WriteString(strings.ReplaceAll(strings.ReplaceAll(*v, "\x00", ""), `"`, `""`))
				buf.WriteByte('"')
			}
		}
		for _, oc := range p.out {
			put(p.value(r, oc))
			if oc.textOf != nil {
				t := ""
				if cell := p.src(r, oc.src); !cell.IsEmpty() {
					t = strings.TrimSpace(formatAsText(cell))
				}
				put(&t)
			}
		}
		buf.WriteByte('\n')
	}
	sql := fmt.Sprintf("COPY %s (%s) FROM STDIN WITH (FORMAT csv)", table, strings.Join(names, ", "))
	tag, err := tx.Conn().PgConn().CopyFrom(ctx, &buf, sql)
	if err != nil {
		return 0, fmt.Errorf("copy into %s: %w", table, err)
	}
	return tag.RowsAffected(), nil
}

func str(s string) *string { return &s }

// value renders one cell for COPY; nil = NULL.
func (p *plan) value(r int, oc outCol) *string {
	col := oc.col
	switch oc.gen {
	case "excel_row":
		return str(strconv.Itoa(r + 1))
	case "category":
		if c := p.catOf[r]; c != nil {
			if id, ok := p.catIDs[c]; ok {
				return str(strconv.FormatInt(id, 10))
			}
		}
		return nil
	}
	cell := p.src(r, oc.src)
	pics := p.imgAt[[2]int{r, oc.src}]

	switch {
	case col.isImage():
		paths := p.saveImages(pics)
		if len(paths) == 0 {
			return p.empty(col)
		}
		return str(strings.Join(paths, ";"))
	case col.fk != nil:
		if cell.IsEmpty() {
			if c := p.catOf[r]; c != nil && col == p.catFK {
				return str(strconv.FormatInt(p.catIDs[c], 10))
			}
			return nil
		}
		if cell.Kind == CellNumber {
			return str(strconv.FormatInt(int64(math.Round(cell.Num)), 10))
		}
		if id, ok := p.fkNames[col][normName(formatAsText(cell))]; ok {
			return str(strconv.FormatInt(id, 10))
		}
		return nil
	}

	if cell.Kind == CellError {
		cell = Cell{}
	}
	if col.isText() || !isKnownType(col.udt) {
		var parts []string
		if !cell.IsEmpty() {
			parts = append(parts, formatAsText(cell))
		}
		if col.isText() {
			for _, path := range p.saveImages(pics) { // picture in a text column -> text + link
				parts = append(parts, fmt.Sprintf(p.opts.ImageViewURL, path))
			}
		}
		if len(parts) == 0 {
			return p.empty(col)
		}
		return str(strings.Join(parts, " "))
	}
	if cell.IsEmpty() {
		return nil
	}
	v, ok := convert(cell, col.udt)
	if !ok {
		p.res.Warnings = append(p.res.Warnings, fmt.Sprintf("%s: %q is not %s, left empty", cellRef(r, oc.src), formatAsText(cell), col.udt))
		return nil
	}
	return &v
}

// empty is the value of a cell without data: ” for NOT NULL text, NULL otherwise.
func (p *plan) empty(col *dbCol) *string {
	if col.isText() && !col.nullable {
		return str("")
	}
	return nil
}

func isKnownType(udt string) bool {
	switch udt {
	case "int2", "int4", "int8", "numeric", "float4", "float8", "money", "bool", "date", "timestamp", "timestamptz", "time", "timetz":
		return true
	}
	return false
}

var dateLayouts = []string{"2006-01-02", "02.01.2006", "2.1.2006", "02/01/2006", "2006-01-02 15:04:05", "2006-01-02T15:04:05",
	"02.01.2006 15:04:05", "02.01.2006 15:04", "2006-01-02 15:04", time.RFC3339}

// convert renders a cell for a non-text column type.
func convert(c Cell, udt string) (string, bool) {
	num := func() (float64, bool) {
		switch c.Kind {
		case CellNumber:
			return c.Num, true
		case CellBool:
			if c.Bool {
				return 1, true
			}
			return 0, true
		case CellString:
			s := strings.NewReplacer(" ", "", " ", "", ",", ".").Replace(strings.TrimSpace(c.Str))
			s = strings.TrimRight(s, "грнuahUAH$€₴%")
			v, err := strconv.ParseFloat(s, 64)
			return v, err == nil
		}
		return 0, false
	}
	tm := func() (time.Time, bool) {
		switch c.Kind {
		case CellDate:
			return c.Time, true
		case CellNumber:
			return excelSerialToTime(c.Num, false), true
		case CellString:
			s := strings.TrimSpace(c.Str)
			for _, l := range dateLayouts {
				if t, err := time.Parse(l, s); err == nil {
					return t, true
				}
			}
			for _, l := range []string{"15:04:05", "15:04"} {
				if t, err := time.Parse(l, s); err == nil {
					return t, true
				}
			}
		}
		return time.Time{}, false
	}
	switch udt {
	case "int2", "int4", "int8":
		v, ok := num()
		if !ok || math.Abs(v-math.Round(v)) > 1e-9 {
			return "", false
		}
		return strconv.FormatInt(int64(math.Round(v)), 10), true
	case "numeric", "float4", "float8", "money":
		v, ok := num()
		return formatNumber(v), ok
	case "bool":
		if c.Kind == CellBool {
			return strconv.FormatBool(c.Bool), true
		}
		switch strings.ToLower(strings.TrimSpace(formatAsText(c))) {
		case "1", "true", "yes", "y", "так", "да", "+", "x", "✓", "є", "есть":
			return "true", true
		case "0", "false", "no", "n", "ні", "нет", "-", "немає":
			return "false", true
		}
		return "", false
	case "date":
		t, ok := tm()
		return t.Format("2006-01-02"), ok
	case "timestamp", "timestamptz":
		t, ok := tm()
		return t.Format("2006-01-02 15:04:05.999"), ok
	case "time", "timetz":
		t, ok := tm()
		return t.Format("15:04:05.999"), ok
	}
	return formatAsText(c), true
}

// qi quotes an identifier.
func qi(s string) string { return `"` + strings.ReplaceAll(s, `"`, `""`) + `"` }

// pictureRows returns the data rows picture idx belongs to:
//   - its anchor row;
//   - every row of a vertical merge that contains the picture's cell (the
//     picture may be anchored anywhere inside the merge, not only at its top);
//   - empty cells of the picture's column in a product group marked by a
//     vertical merge in another table column (e.g. a merged description):
//     with one picture in the group it covers the whole group; with several,
//     each covers the rows from its own down to the next picture (the first
//     one also the rows above it).
func pictureRows(sh *Sheet, idx, first, last int, inCols map[int]bool) []int {
	im := sh.Images[idx]
	set := map[int]bool{im.Row: true}
	add := func(r1, r2 int) {
		for r := max(r1, first); r <= min(r2, last); r++ {
			set[r] = true
		}
	}
	for _, m := range sh.Merges { // the picture's own merged cell
		if m.R2 > m.R1 && m.C1 <= im.Col && im.Col <= m.C2 && m.R1 <= im.Row && im.Row <= m.R2 {
			add(m.R1, m.R2)
		}
	}
	for _, m := range sh.Merges { // product group merged in another column
		if m.R2 == m.R1 || m.C1 != m.C2 || m.C1 == im.Col || !inCols[m.C1] || im.Row < m.R1 || im.Row > m.R2 {
			continue
		}
		r1, r2 := m.R1, m.R2
		for _, o := range sh.Images { // other pictures of the group split it
			if o.Col != im.Col || o.Row < m.R1 || o.Row > m.R2 || o.Row == im.Row {
				continue
			}
			if o.Row < im.Row && o.Row+1 > r1 {
				r1 = im.Row // a picture above owns the rows before this one
			}
			if o.Row > im.Row && o.Row-1 < r2 {
				r2 = o.Row - 1
			}
		}
		for r := r1; r <= r2; r++ {
			// only empty cells of the picture column take the picture (a column
			// of names with an odd picture is not a photo column), and not
			// cells with a picture of their own
			if r != im.Row && (!origCell(sh, r, im.Col).IsEmpty() || cellHasPicture(sh, r, im.Col, idx)) {
				continue
			}
			add(r, r)
		}
	}
	rows := make([]int, 0, len(set))
	for r := range set {
		rows = append(rows, r)
	}
	slices.Sort(rows)
	return rows
}

// cellHasPicture reports whether another picture is anchored in the merged
// cell (or the cell) that covers row r of column col.
func cellHasPicture(sh *Sheet, r, col, except int) bool {
	r1, r2 := r, r
	for _, m := range sh.Merges {
		if m.C1 <= col && col <= m.C2 && m.R1 <= r && r <= m.R2 {
			r1, r2 = m.R1, m.R2
		}
	}
	for j, o := range sh.Images {
		if j != except && o.Col == col && o.Row >= r1 && o.Row <= r2 {
			return true
		}
	}
	return false
}
