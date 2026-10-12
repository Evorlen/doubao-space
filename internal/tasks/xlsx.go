package tasks

import (
	"archive/zip"
	"bytes"
	"encoding/xml"
	"fmt"
	"io"
	"math"
	"path"
	"strconv"
	"strings"
	"time"
)

const maxTaskImportFileSize = 10 * 1024 * 1024

var taskImportTemplateHeaders = []string{
	"任务标题",
	"导入编号",
	"任务描述",
	"截止时间",
	"优先级",
	"标签",
	"外部链接",
}

type spreadsheetRow struct {
	Number int
	Cells  map[int]string
}

type xlsxWorkbook struct {
	Sheets []xlsxSheet `xml:"sheets>sheet"`
}

type xlsxSheet struct {
	Name  string `xml:"name,attr"`
	RelID string `xml:"id,attr"`
}

type xlsxRelationships struct {
	Items []xlsxRelationship `xml:"Relationship"`
}

type xlsxRelationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type xlsxStyles struct {
	NumFmts []xlsxNumFmt `xml:"numFmts>numFmt"`
	CellXfs []xlsxCellXf `xml:"cellXfs>xf"`
}

type xlsxNumFmt struct {
	ID   int    `xml:"numFmtId,attr"`
	Code string `xml:"formatCode,attr"`
}

type xlsxCellXf struct {
	NumFmtID int `xml:"numFmtId,attr"`
}

type xlsxSharedStrings struct {
	Items []xlsxRichText `xml:"si"`
}

type xlsxRichText struct {
	Text string            `xml:"t"`
	Runs []xlsxRichTextRun `xml:"r"`
}

type xlsxRichTextRun struct {
	Text string `xml:"t"`
}

type xlsxWorksheet struct {
	Rows []xlsxWorksheetRow `xml:"sheetData>row"`
}

type xlsxWorksheetRow struct {
	Number int        `xml:"r,attr"`
	Cells  []xlsxCell `xml:"c"`
}

type xlsxCell struct {
	Ref    string       `xml:"r,attr"`
	Type   string       `xml:"t,attr"`
	Style  int          `xml:"s,attr"`
	Value  string       `xml:"v"`
	Inline xlsxRichText `xml:"is"`
}

func taskImportTemplateXLSX() ([]byte, error) {
	var buffer bytes.Buffer
	archive := zip.NewWriter(&buffer)
	entries := []struct {
		name    string
		content string
	}{
		{
			name:    "[Content_Types].xml",
			content: `<?xml version="1.0" encoding="UTF-8"?><Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types"><Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/><Default Extension="xml" ContentType="application/xml"/><Override PartName="/xl/workbook.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet.main+xml"/><Override PartName="/xl/worksheets/sheet1.xml" ContentType="application/vnd.openxmlformats-officedocument.spreadsheetml.worksheet+xml"/></Types>`,
		},
		{
			name:    "_rels/.rels",
			content: `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="xl/workbook.xml"/></Relationships>`,
		},
		{
			name:    "xl/workbook.xml",
			content: `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="任务导入" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		},
		{
			name:    "xl/_rels/workbook.xml.rels",
			content: `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		},
	}

	var sheet bytes.Buffer
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1">`)
	for index, header := range taskImportTemplateHeaders {
		sheet.WriteString(`<c r="`)
		sheet.WriteString(xlsxColumnName(index + 1))
		sheet.WriteString(`1" t="inlineStr"><is><t>`)
		if err := xml.EscapeText(&sheet, []byte(header)); err != nil {
			return nil, fmt.Errorf("escape template header: %w", err)
		}
		sheet.WriteString(`</t></is></c>`)
	}
	sheet.WriteString(`</row></sheetData></worksheet>`)
	entries = append(entries, struct {
		name    string
		content string
	}{name: "xl/worksheets/sheet1.xml", content: sheet.String()})

	for _, entry := range entries {
		file, err := archive.Create(entry.name)
		if err != nil {
			return nil, fmt.Errorf("create template entry %s: %w", entry.name, err)
		}
		if _, err := io.WriteString(file, entry.content); err != nil {
			return nil, fmt.Errorf("write template entry %s: %w", entry.name, err)
		}
	}
	if err := archive.Close(); err != nil {
		return nil, fmt.Errorf("close task import template: %w", err)
	}
	return buffer.Bytes(), nil
}

func parseXLSXRows(reader io.Reader) ([]spreadsheetRow, error) {
	data, err := io.ReadAll(io.LimitReader(reader, maxTaskImportFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取 Excel 文件失败：%w", err)
	}
	if len(data) > maxTaskImportFileSize {
		return nil, &InputError{Message: "Excel 文件不能超过 10 MB。"}
	}
	if len(data) == 0 {
		return nil, &InputError{Message: "Excel 文件不能为空。"}
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, &InputError{Message: "Excel 文件无法读取，请上传有效的 .xlsx 文件。"}
	}

	workbookXML, err := readXLSXEntry(archive, "xl/workbook.xml")
	if err != nil {
		return nil, &InputError{Message: "Excel 文件缺少工作簿信息。"}
	}
	var workbook xlsxWorkbook
	if err := xml.Unmarshal(workbookXML, &workbook); err != nil {
		return nil, &InputError{Message: "Excel 工作簿格式不正确。"}
	}
	if len(workbook.Sheets) == 0 {
		return nil, &InputError{Message: "Excel 文件中没有可导入的工作表。"}
	}

	relationships := map[string]string{}
	if relXML, relErr := readXLSXEntry(archive, "xl/_rels/workbook.xml.rels"); relErr == nil {
		var rels xlsxRelationships
		if err := xml.Unmarshal(relXML, &rels); err == nil {
			for _, relation := range rels.Items {
				relationships[relation.ID] = relation.Target
			}
		}
	}

	selected := workbook.Sheets[0]
	for _, sheet := range workbook.Sheets {
		if strings.TrimSpace(sheet.Name) == "任务导入" {
			selected = sheet
			break
		}
	}
	sheetPath := relationships[selected.RelID]
	if sheetPath == "" {
		sheetPath = "worksheets/sheet1.xml"
	}
	sheetPath = xlsxEntryPath(sheetPath)
	sheetXML, err := readXLSXEntry(archive, sheetPath)
	if err != nil {
		return nil, &InputError{Message: "Excel 文件无法读取任务工作表。"}
	}

	var styles xlsxStyles
	if stylesXML, stylesErr := readXLSXEntry(archive, "xl/styles.xml"); stylesErr == nil {
		_ = xml.Unmarshal(stylesXML, &styles)
	}

	sharedStrings := []string{}
	if sharedXML, sharedErr := readXLSXEntry(archive, "xl/sharedStrings.xml"); sharedErr == nil {
		var shared xlsxSharedStrings
		if err := xml.Unmarshal(sharedXML, &shared); err == nil {
			sharedStrings = make([]string, 0, len(shared.Items))
			for _, item := range shared.Items {
				sharedStrings = append(sharedStrings, richTextValue(item))
			}
		}
	}

	var worksheet xlsxWorksheet
	if err := xml.Unmarshal(sheetXML, &worksheet); err != nil {
		return nil, &InputError{Message: "Excel 任务工作表格式不正确。"}
	}
	rows := make([]spreadsheetRow, 0, len(worksheet.Rows))
	for index, row := range worksheet.Rows {
		rowNumber := row.Number
		if rowNumber <= 0 {
			rowNumber = index + 1
		}
		cells := make(map[int]string, len(row.Cells))
		for _, cell := range row.Cells {
			column := xlsxColumnIndex(cell.Ref)
			if column <= 0 {
				continue
			}
			cells[column] = xlsxCellValue(cell, sharedStrings, styles)
		}
		rows = append(rows, spreadsheetRow{Number: rowNumber, Cells: cells})
	}
	return rows, nil
}

func readXLSXEntry(archive *zip.Reader, name string) ([]byte, error) {
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(io.LimitReader(reader, maxTaskImportFileSize+1))
	}
	return nil, fmt.Errorf("xlsx entry %q not found", name)
}

func xlsxEntryPath(target string) string {
	target = strings.TrimSpace(strings.TrimPrefix(target, "/"))
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean(path.Join("xl", target))
}

func richTextValue(value xlsxRichText) string {
	if len(value.Runs) == 0 {
		return value.Text
	}
	var builder strings.Builder
	if value.Text != "" {
		builder.WriteString(value.Text)
	}
	for _, run := range value.Runs {
		builder.WriteString(run.Text)
	}
	return builder.String()
}

func xlsxCellValue(cell xlsxCell, sharedStrings []string, styles xlsxStyles) string {
	switch cell.Type {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && index >= 0 && index < len(sharedStrings) {
			return sharedStrings[index]
		}
		return ""
	case "inlineStr":
		return richTextValue(cell.Inline)
	case "b":
		if strings.TrimSpace(cell.Value) == "1" {
			return "TRUE"
		}
		return "FALSE"
	}

	value := strings.TrimSpace(cell.Value)
	if value == "" {
		return ""
	}
	if styles.isDateStyle(cell.Style) {
		if serial, err := strconv.ParseFloat(value, 64); err == nil {
			return excelSerialDate(serial).Format("2006-01-02 15:04:05")
		}
	}
	if number, err := strconv.ParseFloat(value, 64); err == nil {
		if math.Trunc(number) == number {
			return strconv.FormatInt(int64(number), 10)
		}
		return strconv.FormatFloat(number, 'f', -1, 64)
	}
	return value
}

func (styles xlsxStyles) isDateStyle(style int) bool {
	if style < 0 || style >= len(styles.CellXfs) {
		return false
	}
	numFmtID := styles.CellXfs[style].NumFmtID
	if (numFmtID >= 14 && numFmtID <= 22) || (numFmtID >= 27 && numFmtID <= 36) || (numFmtID >= 45 && numFmtID <= 47) {
		return true
	}
	for _, format := range styles.NumFmts {
		if format.ID != numFmtID {
			continue
		}
		code := strings.ToLower(format.Code)
		return strings.ContainsAny(code, "yd") || strings.ContainsAny(code, "hs")
	}
	return false
}

func excelSerialDate(serial float64) time.Time {
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.Local)
	return base.Add(time.Duration(serial * float64(24*time.Hour)))
}

func xlsxColumnIndex(reference string) int {
	reference = strings.TrimSpace(reference)
	column := 0
	for _, character := range reference {
		if character < 'A' || character > 'Z' {
			break
		}
		column = column*26 + int(character-'A'+1)
	}
	return column
}

func xlsxColumnName(column int) string {
	name := ""
	for column > 0 {
		column--
		name = string(rune('A'+column%26)) + name
		column /= 26
	}
	return name
}
