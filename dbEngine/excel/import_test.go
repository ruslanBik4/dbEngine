package excel

import (
	"context"
	"errors"
	"os"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
)

func TestMatchColumn(t *testing.T) {
	cols := []*dbCol{
		{name: "id", udt: "int4", auto: true},
		{name: "category_id", udt: "int4", nullable: true, comment: "category (прайс_categories)", fk: &fkRef{parent: "прайс_categories", parentCol: "id"}},
		{name: "найменування", udt: "varchar", comment: "Найменування"},
		{name: "фото", udt: "varchar", comment: `Фото {"view_url": "<a href='%s'>"}`},
		{name: "ціна_грн", udt: "numeric", nullable: true, comment: "Ціна, грн"},
		{name: "akts_tsin_rrts_fkhu_01_10_2026", udt: "int4", nullable: true},
		{name: "is_visible", udt: "bool"},
	}
	for caption, want := range map[string]string{
		"Найменування": "найменування", // comment
		"найменування": "найменування", // name
		"Фото":         "фото",         // comment without the view_url JSON
		"Ціна грн":     "ціна_грн",     // comment, punctuation ignored
		"ЦІНА, ГРН":    "ціна_грн",
		"Категорія":    "category_id", // synonym of a category foreign key
		"Category":     "category_id",
		"АКЦІЯ Ціна РРЦ від ФХУ З 01.10.2026": "akts_tsin_rrts_fkhu_01_10_2026", // converter's shortened translit name
		"id":       "", // identity is never filled
		"Примітка": "",
	} {
		c, _ := matchColumn(caption, cols, map[*dbCol]bool{})
		got := ""
		if c != nil {
			got = c.name
		}
		if got != want {
			t.Errorf("matchColumn(%q) = %q, want %q", caption, got, want)
		}
	}
}

func TestConvert(t *testing.T) {
	for _, tc := range []struct {
		cell Cell
		udt  string
		want string
		ok   bool
	}{
		{Cell{Kind: CellString, Str: "89,50"}, "numeric", "89.5", true},
		{Cell{Kind: CellString, Str: "2 999"}, "int4", "2999", true},
		{Cell{Kind: CellString, Str: "1 250 грн"}, "numeric", "1250", true},
		{Cell{Kind: CellNumber, Num: 12.5}, "int4", "", false},
		{Cell{Kind: CellString, Str: "так"}, "bool", "true", true},
		{Cell{Kind: CellString, Str: "01.11.2026"}, "date", "2026-11-01", true},
		{Cell{Kind: CellNumber, Num: 46327}, "date", "2026-11-01", true}, // Excel serial date
		{Cell{Kind: CellString, Str: "abc"}, "int4", "", false},
	} {
		got, ok := convert(tc.cell, tc.udt)
		if ok != tc.ok || (ok && got != tc.want) {
			t.Errorf("convert(%+v, %s) = %q, %v; want %q, %v", tc.cell, tc.udt, got, ok, tc.want, tc.ok)
		}
	}
}

func TestIsWorkbook(t *testing.T) {
	data, err := os.ReadFile("testdata/price.xlsx")
	if err != nil {
		t.Fatal(err)
	}
	if !IsWorkbook(data) || IsWorkbook([]byte("id,name\n1,a\n")) {
		t.Error("IsWorkbook")
	}
	if _, err := Read([]byte("PK\x03\x04garbage"), ""); err == nil {
		t.Error("broken zip must fail")
	}
}

// ---------------- integration (needs PostgreSQL) ----------------

// testDB connects to EXCEL_TEST_PG_URL and creates a fresh schema with the
// tables the converter generated for testdata/price.xlsx.
func testDB(t *testing.T) (*pgx.Conn, context.Context) {
	t.Helper()
	url := os.Getenv("EXCEL_TEST_PG_URL")
	if url == "" {
		t.Skip("EXCEL_TEST_PG_URL is not set")
	}
	ctx := context.Background()
	conn, err := pgx.Connect(ctx, url)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.Close(ctx) })
	schema := "excel_test_" + strings.ToLower(strings.NewReplacer("/", "_").Replace(t.Name()))
	script, err := os.ReadFile("testdata/price.sql")
	if err != nil {
		t.Fatal(err)
	}
	sql := "DROP SCHEMA IF EXISTS " + schema + " CASCADE; CREATE SCHEMA " + schema + "; SET search_path = " + schema + ";\n" + string(script)
	if _, err := conn.PgConn().Exec(ctx, sql).ReadAll(); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { conn.PgConn().Exec(ctx, "DROP SCHEMA IF EXISTS "+schema+" CASCADE").ReadAll() })
	return conn, ctx
}

func importFile(t *testing.T, conn *pgx.Conn, ctx context.Context, table, file string) *Result {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	res, err := ImportTable(ctx, conn, table, data, Options{ImageDir: t.TempDir(), ImagePathPrefix: "/img/прайс"})
	if err != nil {
		t.Fatal(err)
	}
	t.Log(res.Summary())
	return res
}

func queryStrings(t *testing.T, conn *pgx.Conn, ctx context.Context, sql string) []string {
	t.Helper()
	rows, err := conn.Query(ctx, sql)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()
	var out []string
	for rows.Next() {
		var s string
		if err := rows.Scan(&s); err != nil {
			t.Fatal(err)
		}
		out = append(out, s)
	}
	return out
}

// The same price list again: rows are appended, categories reused.
func TestImportSameFile(t *testing.T) {
	conn, ctx := testDB(t)
	res := importFile(t, conn, ctx, "прайс", "testdata/price.xlsx")
	if res.Rows != 5 || len(res.Parents) != 1 || res.Parents[0].Found != 4 || res.Parents[0].Created != 0 {
		t.Errorf("result %+v", res)
	}
	if res.Images != 4 {
		t.Errorf("images %d, want 4", res.Images)
	}
	got := queryStrings(t, conn, ctx, `SELECT count(*)::text || '/' || count(DISTINCT category_id)::text FROM прайс`)
	if got[0] != "10/3" {
		t.Errorf("rows/categories = %s, want 10/3", got[0])
	}
	// photo column: plain path; picture in the name cell: text + link
	got = queryStrings(t, conn, ctx, `SELECT фото FROM прайс WHERE id > 5 AND фото <> '' ORDER BY id LIMIT 1`)
	if len(got) != 1 || !strings.HasPrefix(got[0], "/img/прайс/") || strings.Contains(got[0], "<a") {
		t.Errorf("photo %v", got)
	}
	got = queryStrings(t, conn, ctx, `SELECT найменування FROM прайс WHERE id > 5 AND найменування LIKE 'Набір%'`)
	if len(got) != 1 || !strings.Contains(got[0], "<a href='/img/прайс/") {
		t.Errorf("text + picture %v", got)
	}
}

// Next month's file: other title, reordered / renamed / extra / missing
// columns, new categories and sub-categories.
func TestImportChangedFile(t *testing.T) {
	conn, ctx := testDB(t)
	res := importFile(t, conn, ctx, "прайс", "testdata/price_v2.xlsx")
	if res.Rows != 4 {
		t.Errorf("rows %d, want 4", res.Rows)
	}
	if strings.Join(res.Unmatched, ",") != "Примітка" || strings.Join(res.Missing, ",") != "наявність" {
		t.Errorf("unmatched %v, missing %v", res.Unmatched, res.Missing)
	}
	if p := res.Parents[0]; p.Found != 3 || p.Created != 2 { // found: Ручний, Електро, Дрилі; created: Викрутки, Садовий
		t.Errorf("parents %+v", p)
	}
	got := queryStrings(t, conn, ctx, `
		SELECT t.артикул || ' ' || t.ціна_грн || ' ' || coalesce(p.name || ' > ', '') || c.name
		FROM прайс t JOIN прайс_categories c ON c.id = t.category_id LEFT JOIN прайс_categories p ON p.id = c.parent_id
		WHERE t.id > 5 ORDER BY t.id`)
	want := []string{
		"B-010 89.5 Ручний інструмент > Викрутки",
		"B-011 95 Ручний інструмент > Викрутки",
		"C-001 420 Садовий інструмент",
		"A-102 2999 Електроінструмент > Дрилі",
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Errorf("imported\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
	// the new rows show up in the view with their category names
	got = queryStrings(t, conn, ctx, `SELECT category FROM прайс_visible WHERE артикул = 'C-001'`)
	if len(got) != 1 || got[0] != "Садовий інструмент" {
		t.Errorf("view %v", got)
	}
}

// A flat list with a "Категорія" column: names are resolved / created in the
// parent table.
func TestImportCategoryColumn(t *testing.T) {
	conn, ctx := testDB(t)
	res := importFile(t, conn, ctx, "прайс", "testdata/price_flat.xlsx")
	if res.Rows != 3 || res.Parents[0].Created != 1 {
		t.Errorf("result %+v", res)
	}
	got := queryStrings(t, conn, ctx, `SELECT t.артикул || ' ' || c.name FROM прайс t JOIN прайс_categories c ON c.id = t.category_id WHERE t.id > 5 ORDER BY t.id`)
	if strings.Join(got, ";") != "B-020 Ручний інструмент;D-001 Вимірювальний інструмент;D-002 Вимірювальний інструмент" {
		t.Errorf("got %v", got)
	}
}

// Nothing matches: clear error, nothing written.
func TestImportNoMatch(t *testing.T) {
	conn, ctx := testDB(t)
	data, _ := os.ReadFile("testdata/price_flat.xlsx")
	_, err := ImportTable(ctx, conn, "прайс_categories", data, Options{})
	if !errors.Is(err, ErrNoColumns) {
		t.Fatalf("err = %v, want ErrNoColumns", err)
	}
	if got := queryStrings(t, conn, ctx, `SELECT count(*)::text FROM прайс_categories`); got[0] != "4" {
		t.Errorf("categories changed: %v", got)
	}
}
