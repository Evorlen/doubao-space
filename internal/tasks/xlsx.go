package tasks

import (
	"io"

	sharedxlsx "github.com/Icingworld/doubao-space/internal/xlsx"
)

const maxTaskImportFileSize = sharedxlsx.MaxFileSize

var taskImportTemplateHeaders = []string{
	"任务标题",
	"导入编号",
	"任务描述",
	"截止时间",
	"优先级",
	"标签",
	"外部链接",
}

type spreadsheetRow = sharedxlsx.Row
type xlsxNumFmt = sharedxlsx.NumFmt
type xlsxCellXf = sharedxlsx.CellXf
type xlsxStyles = sharedxlsx.Styles
type xlsxRichText = sharedxlsx.RichText
type xlsxRichTextRun = sharedxlsx.TextRun
type xlsxCell = sharedxlsx.Cell

func taskImportTemplateXLSX() ([]byte, error) {
	return sharedxlsx.CreateTemplate("任务导入", taskImportTemplateHeaders)
}

func parseXLSXRows(reader io.Reader) ([]spreadsheetRow, error) {
	rows, err := sharedxlsx.ParseRows(reader, "任务导入", "任务")
	if err != nil {
		return nil, &InputError{Message: err.Error()}
	}
	return rows, nil
}

func xlsxCellValue(cell xlsxCell, sharedStrings []string, styles xlsxStyles) string {
	return sharedxlsx.CellValue(cell, sharedStrings, styles)
}
