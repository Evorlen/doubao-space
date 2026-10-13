package tasks

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"io"
	"strings"
	"testing"
	"time"
)

func TestTaskImportPreviewAndCommit(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	parent, err := service.Create(ctx, CreateInput{
		Title:    "旅行准备",
		DueAt:    "2026-10-20 18:00",
		Priority: PriorityMedium,
	})
	if err != nil {
		t.Fatalf("Create(parent) error = %v", err)
	}
	if _, err := service.Create(ctx, CreateInput{
		ParentID: &parent.ID,
		Title:    "已有材料",
		DueAt:    "2026-10-19 18:00",
		Priority: PriorityMedium,
	}); err != nil {
		t.Fatalf("Create(existing child) error = %v", err)
	}

	workbook := testTaskWorkbook(t, [][]string{
		{"任务标题", "导入编号", "任务描述", "截止时间", "优先级", "标签", "外部链接"},
		{"整理护照复印件", "TRIP-001", "准备纸质材料", "2026-10-18 18:00", "高", "旅行；材料", "清单|https://example.com/list"},
		{"已有材料", "", "", "2026-10-19 18:00", "中", "", ""},
		{"超期任务", "TRIP-002", "", "2026-10-21 18:00", "低", "", ""},
	})

	preview, err := service.PreviewImport(ctx, parent.ID, bytes.NewReader(workbook))
	if err != nil {
		t.Fatalf("PreviewImport() error = %v", err)
	}
	if preview.Total != 3 || preview.NewCount != 1 || preview.DuplicateCount != 1 || preview.ErrorCount != 1 || preview.CanImport {
		t.Fatalf("preview = %#v, want 3 total, 1 new, 1 duplicate, 1 error and not importable", preview)
	}
	if preview.Rows[1].Status != "duplicate" || preview.Rows[2].Status != "error" {
		t.Fatalf("preview rows = %#v, want duplicate and error statuses", preview.Rows)
	}

	before, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent before commit) error = %v", err)
	}
	if len(before.Children) != 1 {
		t.Fatalf("children before failed import = %d, want 1", len(before.Children))
	}

	validWorkbook := testTaskWorkbook(t, [][]string{
		{"任务标题", "导入编号", "任务描述", "截止时间", "优先级", "标签", "外部链接"},
		{"整理护照复印件", "TRIP-001", "准备纸质材料", "2026-10-18 18:00", "高", "旅行；材料", "清单|https://example.com/list"},
	})
	result, err := service.Import(ctx, parent.ID, bytes.NewReader(validWorkbook))
	if err != nil {
		t.Fatalf("Import() error = %v", err)
	}
	if result.CreatedCount != 1 || result.NewCount != 1 || result.ErrorCount != 0 {
		t.Fatalf("import result = %#v, want one created task", result)
	}

	after, err := service.Get(ctx, parent.ID)
	if err != nil {
		t.Fatalf("Get(parent after commit) error = %v", err)
	}
	if len(after.Children) != 2 {
		t.Fatalf("children after import = %d, want 2", len(after.Children))
	}
	imported := after.Children[1]
	if imported.Title != "整理护照复印件" || imported.Priority != PriorityHigh || imported.Description != "准备纸质材料" {
		t.Fatalf("imported task = %#v", imported)
	}
	if len(imported.Tags) != 2 || len(imported.Links) != 1 || imported.Links[0].Title != "清单" {
		t.Fatalf("imported associations = tags %#v, links %#v", imported.Tags, imported.Links)
	}

	repeated, err := service.PreviewImport(ctx, parent.ID, bytes.NewReader(validWorkbook))
	if err != nil {
		t.Fatalf("PreviewImport(repeated) error = %v", err)
	}
	if repeated.NewCount != 0 || repeated.DuplicateCount != 1 || repeated.ErrorCount != 0 {
		t.Fatalf("repeated preview = %#v, want one duplicate and no new task", repeated)
	}
}

func TestTaskImportTemplate(t *testing.T) {
	content, err := taskImportTemplateXLSX()
	if err != nil {
		t.Fatalf("taskImportTemplateXLSX() error = %v", err)
	}
	if len(content) < 2 || content[0] != 'P' || content[1] != 'K' {
		t.Fatalf("template is not an xlsx zip archive")
	}

	rows, err := parseXLSXRows(bytes.NewReader(content))
	if err != nil {
		t.Fatalf("parseXLSXRows(template) error = %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("template rows = %d, want one header row", len(rows))
	}
	for index, header := range taskImportTemplateHeaders {
		if got := rows[0].Cells[index+1]; got != header {
			t.Errorf("template header %d = %q, want %q", index+1, got, header)
		}
	}
}

func TestXLSXDateSerialPreservesMidnight(t *testing.T) {
	location, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		t.Fatalf("LoadLocation(Asia/Shanghai) error = %v", err)
	}
	previousLocal := time.Local
	time.Local = location
	t.Cleanup(func() { time.Local = previousLocal })

	styles := xlsxStyles{CellXfs: []xlsxCellXf{{}, {NumFmtID: 14}}}
	got := xlsxCellValue(xlsxCell{Style: 1, Value: "46326"}, nil, styles)
	if got != "2026-10-31 00:00:00" {
		t.Fatalf("xlsx date serial = %q, want 2026-10-31 00:00:00", got)
	}

	parsed, err := parseDue(got)
	if err != nil {
		t.Fatalf("parseDue(%q) error = %v", got, err)
	}
	if parsed.Format("2006-01-02 15:04:05") != "2026-10-31 00:00:00" {
		t.Fatalf("parsed xlsx date = %s, want 2026-10-31 00:00:00", parsed.Format("2006-01-02 15:04:05"))
	}
}

func TestParseDueAcceptsSlashDate(t *testing.T) {
	parsed, err := parseDue("2026/10/31 0:00")
	if err != nil {
		t.Fatalf("parseDue(slash date) error = %v", err)
	}
	if parsed.Format("2006-01-02 15:04") != "2026-10-31 00:00" {
		t.Fatalf("parsed slash date = %s, want 2026-10-31 00:00", parsed.Format("2006-01-02 15:04"))
	}
}

func testTaskWorkbook(t *testing.T, rows [][]string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := map[string]string{
		"xl/workbook.xml":            `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="任务导入" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		"xl/_rels/workbook.xml.rels": `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
	}
	var sheet strings.Builder
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData>`)
	for rowIndex, row := range rows {
		sheet.WriteString(`<row r="`)
		sheet.WriteString(strings.TrimSpace(strconvItoa(rowIndex + 1)))
		sheet.WriteString(`">`)
		for columnIndex, value := range row {
			sheet.WriteString(`<c r="`)
			sheet.WriteString(testColumnName(columnIndex + 1))
			sheet.WriteString(strconvItoa(rowIndex + 1))
			sheet.WriteString(`" t="inlineStr"><is><t>`)
			_ = xml.EscapeText(&sheet, []byte(value))
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

func testColumnName(column int) string {
	name := ""
	for column > 0 {
		column--
		name = string(rune('A'+column%26)) + name
		column /= 26
	}
	return name
}

func strconvItoa(value int) string {
	if value == 0 {
		return "0"
	}
	digits := make([]byte, 0, 4)
	for value > 0 {
		digits = append(digits, byte('0'+value%10))
		value /= 10
	}
	for left, right := 0, len(digits)-1; left < right; left, right = left+1, right-1 {
		digits[left], digits[right] = digits[right], digits[left]
	}
	return string(digits)
}
