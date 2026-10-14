package entertainment

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strconv"
	"strings"
	"testing"

	sharedxlsx "github.com/Icingworld/doubao-space/internal/xlsx"
)

func TestEntertainmentImportPreviewAndCommit(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	if _, err := service.Create(ctx, Input{
		MediaType: MediaTypeMovie,
		Title:     "海街日记",
		Creator:   "是枝裕和",
		Status:    StatusPlanned,
	}); err != nil {
		t.Fatalf("Create(existing) error = %v", err)
	}

	workbook := testEntertainmentWorkbook(t, [][]string{
		entertainmentImportTemplateHeaders,
		{"电影", "海街日记", "是枝裕和", "2015", "", "计划中", "", "", "5", "", "", "是", "治愈", "", ""},
		{"电视剧", "黑暗荣耀", "安吉镐", "2022", "Netflix", "在看", "3", "16", "4", "节奏很好", "", "是", "悬疑；韩剧", "2026-10-01", ""},
		{"series", "黑暗荣耀", "安吉镐", "2022", "Netflix", "in_progress", "3", "16", "4", "", "", "true", "悬疑", "2026-10-01", ""},
		{"电影", "错误评分", "", "", "", "计划中", "", "", "6", "", "", "否", "", "", ""},
	})

	preview, err := service.PreviewImport(ctx, bytes.NewReader(workbook))
	if err != nil {
		t.Fatalf("PreviewImport() error = %v", err)
	}
	if preview.Total != 4 || preview.NewCount != 1 || preview.DuplicateCount != 2 || preview.ErrorCount != 1 || preview.CanImport {
		t.Fatalf("preview = %#v, want 4 total, 1 new, 2 duplicates, 1 error and not importable", preview)
	}
	if preview.Rows[0].Status != "duplicate" || preview.Rows[2].Status != "duplicate" || preview.Rows[3].Status != "error" {
		t.Fatalf("preview rows = %#v, want duplicate, duplicate and error statuses", preview.Rows)
	}

	response, err := service.List(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("List(before import) error = %v", err)
	}
	if len(response.Items) != 1 {
		t.Fatalf("items before import = %d, want 1", len(response.Items))
	}

	validWorkbook := testEntertainmentWorkbook(t, [][]string{
		entertainmentImportTemplateHeaders,
		{"电视剧", "黑暗荣耀", "安吉镐", "2022", "Netflix", "在看", "3", "16", "4", "节奏很好", "", "是", "悬疑；韩剧", "2026-10-01", ""},
	})
	result, err := service.Import(ctx, bytes.NewReader(validWorkbook))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedCount != 1 || result.NewCount != 1 || result.ErrorCount != 0 {
		t.Fatalf("import result = %#v, want one created item", result)
	}

	response, err = service.List(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("List(after import) error = %v", err)
	}
	if len(response.Items) != 2 {
		t.Fatalf("items after import = %d, want 2", len(response.Items))
	}
	imported := response.Items[0]
	if imported.Title != "黑暗荣耀" || imported.Status != StatusInProgress || imported.ProgressCurrent != 3 || imported.ProgressTotal == nil || *imported.ProgressTotal != 16 {
		t.Fatalf("imported item = %#v", imported)
	}
	if imported.Rating != 4 || !imported.Favorite || imported.StartedOn == nil || *imported.StartedOn != "2026-10-01" {
		t.Fatalf("imported item metadata = %#v", imported)
	}
	if len(imported.Tags) != 2 {
		t.Fatalf("imported tags = %#v, want two tags", imported.Tags)
	}

	repeated, err := service.PreviewImport(ctx, bytes.NewReader(validWorkbook))
	if err != nil {
		t.Fatalf("PreviewImport(repeated) error = %v", err)
	}
	if repeated.NewCount != 0 || repeated.DuplicateCount != 1 || repeated.ErrorCount != 0 {
		t.Fatalf("repeated preview = %#v, want one duplicate and no new item", repeated)
	}
}

func TestEntertainmentImportTemplate(t *testing.T) {
	content, err := entertainmentImportTemplateXLSX()
	if err != nil {
		t.Fatalf("entertainmentImportTemplateXLSX() error = %v", err)
	}
	if len(content) < 2 || content[0] != 'P' || content[1] != 'K' {
		t.Fatalf("template is not an xlsx zip archive")
	}

	rows, err := sharedxlsx.ParseRows(bytes.NewReader(content), "娱乐导入", "娱乐")
	if err != nil {
		t.Fatalf("ParseRows(template) error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("template rows = %d, want one header row", len(rows))
	}
	for index, header := range entertainmentImportTemplateHeaders {
		if got := rows[0].Cells[index+1]; got != header {
			t.Errorf("template header %d = %q, want %q", index+1, got, header)
		}
	}
}

func testEntertainmentWorkbook(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := map[string]string{
		"[Content_Types].xml":        `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="娱乐导入" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		sheet.WriteString(`<row r="`)
		sheet.WriteString(strconv.Itoa(rowIndex + 1))
		sheet.WriteString(`">`)
		for columnIndex, value := range row {
			sheet.WriteString(`<c r="`)
			sheet.WriteString(testEntertainmentColumnName(columnIndex + 1))
			sheet.WriteString(strconv.Itoa(rowIndex + 1))
			sheet.WriteString(`" t="inlineStr"><is><t>`)
			if err := xml.EscapeText(&sheet, []byte(value)); err != nil {
				t.Fatalf("escape cell: %v", err)
			}
			sheet.WriteString(`</t></is></c>`)
		}
		sheet.WriteString(`</row>`)
	}
	sheet.WriteString(`</sheetData></worksheet>`)
	entries["xl/worksheets/sheet1.xml"] = sheet.String()

	for name, content := range entries {
		entry, err := archive.Create(name)
		if err != nil {
			t.Fatalf("create xlsx entry %s: %v", name, err)
		}
		if _, err := io.WriteString(entry, content); err != nil {
			t.Fatalf("write xlsx entry %s: %v", name, err)
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatalf("close xlsx archive: %v", err)
	}
	return buffer.Bytes()
}

func testEntertainmentColumnName(column int) string {
	name := ""
	for column > 0 {
		column--
		name = string(rune('A'+column%26)) + name
		column /= 26
	}
	return name
}
