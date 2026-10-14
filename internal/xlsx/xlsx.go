package xlsx

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

const MaxFileSize = 10 * 1024 * 1024

type Row struct {
	Number int
	Cells  map[int]string
}

type NumFmt struct {
	ID   int    `xml:"numFmtId,attr"`
	Code string `xml:"formatCode,attr"`
}

type CellXf struct {
	NumFmtID int `xml:"numFmtId,attr"`
}

type Styles struct {
	NumFmts []NumFmt `xml:"numFmts>numFmt"`
	CellXfs []CellXf `xml:"cellXfs>xf"`
}

type RichText struct {
	Text string    `xml:"t"`
	Runs []TextRun `xml:"r"`
}

type TextRun struct {
	Text string `xml:"t"`
}

type Cell struct {
	Ref    string   `xml:"r,attr"`
	Type   string   `xml:"t,attr"`
	Style  int      `xml:"s,attr"`
	Value  string   `xml:"v"`
	Inline RichText `xml:"is"`
}

type Error struct {
	Message string
}

func (e *Error) Error() string { return e.Message }

type workbook struct {
	Sheets []sheet `xml:"sheets>sheet"`
}

type sheet struct {
	Name  string `xml:"name,attr"`
	RelID string `xml:"id,attr"`
}

type relationships struct {
	Items []relationship `xml:"Relationship"`
}

type relationship struct {
	ID     string `xml:"Id,attr"`
	Target string `xml:"Target,attr"`
}

type sharedStrings struct {
	Items []RichText `xml:"si"`
}

type worksheet struct {
	Rows []worksheetRow `xml:"sheetData>row"`
}

type worksheetRow struct {
	Number int    `xml:"r,attr"`
	Cells  []Cell `xml:"c"`
}

// ParseRows reads the first worksheet, preferring preferredSheet when it exists.
// The label is included in user-facing errors so different importers can explain
// which worksheet could not be read.
func ParseRows(reader io.Reader, preferredSheet, label string) ([]Row, error) {
	data, err := io.ReadAll(io.LimitReader(reader, MaxFileSize+1))
	if err != nil {
		return nil, fmt.Errorf("读取 Excel 文件失败：%w", err)
	}
	if len(data) > MaxFileSize {
		return nil, &Error{Message: "Excel 文件不能超过 10 MB。"}
	}
	if len(data) == 0 {
		return nil, &Error{Message: "Excel 文件不能为空。"}
	}

	archive, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, &Error{Message: "Excel 文件无法读取，请上传有效的 .xlsx 文件。"}
	}

	workbookXML, err := readEntry(archive, "xl/workbook.xml")
	if err != nil {
		return nil, &Error{Message: "Excel 文件缺少工作簿信息。"}
	}
	var book workbook
	if err := xml.Unmarshal(workbookXML, &book); err != nil {
		return nil, &Error{Message: "Excel 工作簿格式不正确。"}
	}
	if len(book.Sheets) == 0 {
		return nil, &Error{Message: "Excel 文件中没有可导入的工作表。"}
	}

	relationshipMap := map[string]string{}
	if relXML, relErr := readEntry(archive, "xl/_rels/workbook.xml.rels"); relErr == nil {
		var rels relationships
		if err := xml.Unmarshal(relXML, &rels); err == nil {
			for _, relation := range rels.Items {
				relationshipMap[relation.ID] = relation.Target
			}
		}
	}

	selected := book.Sheets[0]
	for _, candidate := range book.Sheets {
		if strings.TrimSpace(candidate.Name) == strings.TrimSpace(preferredSheet) {
			selected = candidate
			break
		}
	}
	sheetPath := relationshipMap[selected.RelID]
	if sheetPath == "" {
		sheetPath = "worksheets/sheet1.xml"
	}
	sheetPath = entryPath(sheetPath)
	sheetXML, err := readEntry(archive, sheetPath)
	if err != nil {
		return nil, &Error{Message: fmt.Sprintf("Excel 文件无法读取%s工作表。", label)}
	}

	var styles Styles
	if stylesXML, stylesErr := readEntry(archive, "xl/styles.xml"); stylesErr == nil {
		_ = xml.Unmarshal(stylesXML, &styles)
	}

	stringsTable := []string{}
	if sharedXML, sharedErr := readEntry(archive, "xl/sharedStrings.xml"); sharedErr == nil {
		var shared sharedStrings
		if err := xml.Unmarshal(sharedXML, &shared); err == nil {
			stringsTable = make([]string, 0, len(shared.Items))
			for _, item := range shared.Items {
				stringsTable = append(stringsTable, RichTextValue(item))
			}
		}
	}

	var sheetData worksheet
	if err := xml.Unmarshal(sheetXML, &sheetData); err != nil {
		return nil, &Error{Message: fmt.Sprintf("Excel %s工作表格式不正确。", label)}
	}
	rows := make([]Row, 0, len(sheetData.Rows))
	for index, row := range sheetData.Rows {
		rowNumber := row.Number
		if rowNumber <= 0 {
			rowNumber = index + 1
		}
		cells := make(map[int]string, len(row.Cells))
		for _, cell := range row.Cells {
			column := columnIndex(cell.Ref)
			if column <= 0 {
				continue
			}
			cells[column] = CellValue(cell, stringsTable, styles)
		}
		rows = append(rows, Row{Number: rowNumber, Cells: cells})
	}
	return rows, nil
}

func CreateTemplate(sheetName string, headers []string) ([]byte, error) {
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
			content: `<?xml version="1.0" encoding="UTF-8"?><workbook xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main" xmlns:r="http://schemas.openxmlformats.org/officeDocument/2006/relationships"><sheets><sheet name="` + xmlEscape(sheetName) + `" sheetId="1" r:id="rId1"/></sheets></workbook>`,
		},
		{
			name:    "xl/_rels/workbook.xml.rels",
			content: `<?xml version="1.0" encoding="UTF-8"?><Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships"><Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/worksheet" Target="worksheets/sheet1.xml"/></Relationships>`,
		},
	}

	var sheet bytes.Buffer
	sheet.WriteString(`<?xml version="1.0" encoding="UTF-8"?><worksheet xmlns="http://schemas.openxmlformats.org/spreadsheetml/2006/main"><sheetData><row r="1">`)
	for index, header := range headers {
		sheet.WriteString(`<c r="`)
		sheet.WriteString(columnName(index + 1))
		sheet.WriteString(`1" t="inlineStr"><is><t>`)
		sheet.WriteString(xmlEscape(header))
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
		return nil, fmt.Errorf("close xlsx template: %w", err)
	}
	return buffer.Bytes(), nil
}

func CellValue(cell Cell, stringsTable []string, styles Styles) string {
	switch cell.Type {
	case "s":
		index, err := strconv.Atoi(strings.TrimSpace(cell.Value))
		if err == nil && index >= 0 && index < len(stringsTable) {
			return stringsTable[index]
		}
		return ""
	case "inlineStr":
		return RichTextValue(cell.Inline)
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

func RichTextValue(value RichText) string {
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

func (styles Styles) isDateStyle(style int) bool {
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
	base := time.Date(1899, 12, 30, 0, 0, 0, 0, time.UTC)
	nanoseconds := math.Round(serial * float64(24*time.Hour))
	return base.Add(time.Duration(nanoseconds))
}

func readEntry(archive *zip.Reader, name string) ([]byte, error) {
	for _, file := range archive.File {
		if file.Name != name {
			continue
		}
		reader, err := file.Open()
		if err != nil {
			return nil, err
		}
		defer reader.Close()
		return io.ReadAll(io.LimitReader(reader, MaxFileSize+1))
	}
	return nil, fmt.Errorf("xlsx entry %q not found", name)
}

func entryPath(target string) string {
	target = strings.TrimSpace(strings.TrimPrefix(target, "/"))
	if strings.HasPrefix(target, "xl/") {
		return path.Clean(target)
	}
	return path.Clean(path.Join("xl", target))
}

func columnIndex(reference string) int {
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

func columnName(column int) string {
	name := ""
	for column > 0 {
		column--
		name = string(rune('A'+column%26)) + name
		column /= 26
	}
	return name
}

func xmlEscape(value string) string {
	var buffer bytes.Buffer
	if err := xml.EscapeText(&buffer, []byte(value)); err != nil {
		return ""
	}
	return buffer.String()
}
