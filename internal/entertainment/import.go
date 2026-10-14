package entertainment

import (
	"context"
	"fmt"
	"io"
	"math"
	"strconv"
	"strings"
	"time"

	sharedxlsx "github.com/Icingworld/doubao-space/internal/xlsx"
)

type EntertainmentImportRowResult struct {
	Row     int    `json:"row"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type EntertainmentImportReport struct {
	Total          int                            `json:"total"`
	NewCount       int                            `json:"new_count"`
	DuplicateCount int                            `json:"duplicate_count"`
	ErrorCount     int                            `json:"error_count"`
	CreatedCount   int                            `json:"created_count"`
	CanImport      bool                           `json:"can_import"`
	Rows           []EntertainmentImportRowResult `json:"rows"`
}

type entertainmentImportRow struct {
	Number          int
	Title           string
	Input           Input
	ValidationError string
}

type entertainmentImportCandidate struct {
	ReportIndex int
	Input       Input
}

func (s *Service) PreviewImport(ctx context.Context, reader io.Reader) (EntertainmentImportReport, error) {
	return s.processImport(ctx, reader, false)
}

func (s *Service) Import(ctx context.Context, reader io.Reader) (EntertainmentImportReport, error) {
	return s.processImport(ctx, reader, true)
}

func (s *Service) processImport(ctx context.Context, reader io.Reader, commit bool) (EntertainmentImportReport, error) {
	rows, err := parseEntertainmentImportRows(reader)
	if err != nil {
		return EntertainmentImportReport{}, err
	}

	items, err := s.listItems(ctx, ListOptions{})
	if err != nil {
		return EntertainmentImportReport{}, err
	}
	existingKeys := make(map[string]struct{}, len(items))
	for _, item := range items {
		existingKeys[entertainmentImportKey(Input{
			MediaType: item.MediaType,
			Title:     item.Title,
			Creator:   item.Creator,
		})] = struct{}{}
	}

	report := EntertainmentImportReport{
		Total: len(rows),
		Rows:  make([]EntertainmentImportRowResult, 0, len(rows)),
	}
	seenKeys := make(map[string]struct{})
	candidates := make([]entertainmentImportCandidate, 0, len(rows))
	for _, row := range rows {
		result := EntertainmentImportRowResult{Row: row.Number, Title: row.Title}
		report.Rows = append(report.Rows, result)
		reportIndex := len(report.Rows) - 1

		if row.ValidationError != "" {
			report.Rows[reportIndex].Status = "error"
			report.Rows[reportIndex].Message = row.ValidationError
			report.ErrorCount++
			continue
		}

		normalized, normalizeErr := normalizeInput(row.Input)
		if normalizeErr != nil {
			report.Rows[reportIndex].Status = "error"
			report.Rows[reportIndex].Message = normalizeErr.Error()
			report.ErrorCount++
			continue
		}

		key := entertainmentImportKey(normalized)
		if _, exists := existingKeys[key]; exists {
			report.Rows[reportIndex].Status = "duplicate"
			report.Rows[reportIndex].Message = "相同类型、名称和作者/导演的作品已存在，已跳过。"
			report.DuplicateCount++
			continue
		}
		if _, exists := seenKeys[key]; exists {
			report.Rows[reportIndex].Status = "duplicate"
			report.Rows[reportIndex].Message = "文件内存在相同类型、名称和作者/导演的作品，已跳过。"
			report.DuplicateCount++
			continue
		}
		seenKeys[key] = struct{}{}
		candidates = append(candidates, entertainmentImportCandidate{
			ReportIndex: reportIndex,
			Input:       normalized,
		})
		report.Rows[reportIndex].Status = "new"
		report.Rows[reportIndex].Message = "待导入"
	}

	report.NewCount = len(candidates)
	report.CanImport = report.ErrorCount == 0 && report.NewCount > 0
	if report.ErrorCount > 0 || !commit || report.NewCount == 0 {
		return report, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return EntertainmentImportReport{}, fmt.Errorf("begin import entertainment items: %w", err)
	}
	defer tx.Rollback()
	for _, candidate := range candidates {
		id, insertErr := insertItem(ctx, tx, candidate.Input)
		if insertErr != nil {
			return EntertainmentImportReport{}, insertErr
		}
		if tagErr := replaceTags(ctx, tx, id, candidate.Input.Tags); tagErr != nil {
			return EntertainmentImportReport{}, tagErr
		}
		report.Rows[candidate.ReportIndex].Status = "created"
		report.Rows[candidate.ReportIndex].Message = "已创建"
	}
	if err := tx.Commit(); err != nil {
		return EntertainmentImportReport{}, fmt.Errorf("commit imported entertainment items: %w", err)
	}
	report.CreatedCount = len(candidates)
	report.CanImport = false
	return report, nil
}

func parseEntertainmentImportRows(reader io.Reader) ([]entertainmentImportRow, error) {
	spreadsheetRows, err := sharedxlsx.ParseRows(reader, "娱乐导入", "娱乐")
	if err != nil {
		return nil, &ValidationError{Message: err.Error()}
	}
	if len(spreadsheetRows) == 0 {
		return nil, &ValidationError{Message: "Excel 文件中没有娱乐数据。"}
	}

	headerIndex := -1
	for index, row := range spreadsheetRows {
		if !entertainmentSpreadsheetRowIsBlank(row) {
			headerIndex = index
			break
		}
	}
	if headerIndex < 0 {
		return nil, &ValidationError{Message: "Excel 文件中没有娱乐数据。"}
	}

	columns := make(map[string]int)
	for column, value := range spreadsheetRows[headerIndex].Cells {
		name := normalizeEntertainmentImportHeader(value)
		if name == "" {
			continue
		}
		canonical, ok := entertainmentImportHeaderAliases[name]
		if !ok {
			return nil, &ValidationError{Message: fmt.Sprintf("Excel 包含无法识别的列“%s”。", strings.TrimSpace(value))}
		}
		if _, exists := columns[canonical]; exists {
			return nil, &ValidationError{Message: fmt.Sprintf("Excel 中重复出现了列“%s”。", strings.TrimSpace(value))}
		}
		columns[canonical] = column
	}
	if _, ok := columns["media_type"]; !ok {
		return nil, &ValidationError{Message: "Excel 必须包含“作品类型”列。"}
	}
	if _, ok := columns["title"]; !ok {
		return nil, &ValidationError{Message: "Excel 必须包含“作品名称”列。"}
	}

	result := make([]entertainmentImportRow, 0, len(spreadsheetRows)-headerIndex-1)
	for _, spreadsheetRow := range spreadsheetRows[headerIndex+1:] {
		if entertainmentSpreadsheetRowIsBlank(spreadsheetRow) {
			continue
		}
		row := entertainmentImportRow{Number: spreadsheetRow.Number}
		row.Title = entertainmentImportCell(spreadsheetRow, columns, "title")
		row.Input.MediaType = normalizeImportedMediaType(entertainmentImportCell(spreadsheetRow, columns, "media_type"))
		row.Input.Title = row.Title
		row.Input.Creator = entertainmentImportCell(spreadsheetRow, columns, "creator")
		row.Input.Source = entertainmentImportCell(spreadsheetRow, columns, "source")
		row.Input.Status = normalizeImportedStatus(entertainmentImportCell(spreadsheetRow, columns, "status"))
		row.Input.Review = entertainmentImportCell(spreadsheetRow, columns, "review")
		row.Input.Note = entertainmentImportCell(spreadsheetRow, columns, "note")
		row.Input.Tags = splitEntertainmentImportTags(entertainmentImportCell(spreadsheetRow, columns, "tags"))

		var messages []string
		row.Input.ReleaseYear, messages = parseEntertainmentImportOptionalInt(entertainmentImportCell(spreadsheetRow, columns, "release_year"), "年份", messages)
		row.Input.ProgressCurrent, messages = parseEntertainmentImportIntValue(row.Input.ProgressCurrent, entertainmentImportCell(spreadsheetRow, columns, "progress_current"), "当前进度", messages)
		row.Input.ProgressTotal, messages = parseEntertainmentImportOptionalInt(entertainmentImportCell(spreadsheetRow, columns, "progress_total"), "总进度", messages)
		row.Input.Rating, messages = parseEntertainmentImportIntValue(row.Input.Rating, entertainmentImportCell(spreadsheetRow, columns, "rating"), "评分", messages)
		row.Input.Favorite, messages = parseEntertainmentImportBool(entertainmentImportCell(spreadsheetRow, columns, "favorite"), messages)
		row.Input.StartedOn, messages = parseEntertainmentImportDate(entertainmentImportCell(spreadsheetRow, columns, "started_on"), "开始日期", messages)
		row.Input.CompletedOn, messages = parseEntertainmentImportDate(entertainmentImportCell(spreadsheetRow, columns, "completed_on"), "完成日期", messages)
		row.ValidationError = strings.Join(messages, "；")
		result = append(result, row)
	}
	if len(result) == 0 {
		return nil, &ValidationError{Message: "Excel 中没有可导入的娱乐行。"}
	}
	return result, nil
}

var entertainmentImportHeaderAliases = map[string]string{
	"作品类型":             "media_type",
	"媒体类型":             "media_type",
	"类型":               "media_type",
	"media_type":       "media_type",
	"作品名称":             "title",
	"作品名":              "title",
	"标题":               "title",
	"名称":               "title",
	"title":            "title",
	"作者或导演":            "creator",
	"作者/导演":            "creator",
	"作者导演":             "creator",
	"作者":               "creator",
	"导演":               "creator",
	"创作者":              "creator",
	"creator":          "creator",
	"年份":               "release_year",
	"发行年份":             "release_year",
	"上映年份":             "release_year",
	"出版年份":             "release_year",
	"release_year":     "release_year",
	"来源":               "source",
	"source":           "source",
	"状态":               "status",
	"阅读状态":             "status",
	"观看状态":             "status",
	"status":           "status",
	"当前进度":             "progress_current",
	"当前章节":             "progress_current",
	"当前集数":             "progress_current",
	"progress_current": "progress_current",
	"总进度":              "progress_total",
	"总章节":              "progress_total",
	"总集数":              "progress_total",
	"progress_total":   "progress_total",
	"评分":               "rating",
	"rating":           "rating",
	"个人评价":             "review",
	"评价":               "review",
	"观后感":              "review",
	"书评":               "review",
	"review":           "review",
	"备注":               "note",
	"note":             "note",
	"收藏":               "favorite",
	"是否收藏":             "favorite",
	"favorite":         "favorite",
	"标签":               "tags",
	"tags":             "tags",
	"开始日期":             "started_on",
	"开始时间":             "started_on",
	"started_on":       "started_on",
	"完成日期":             "completed_on",
	"完成时间":             "completed_on",
	"completed_on":     "completed_on",
}

func entertainmentSpreadsheetRowIsBlank(row sharedxlsx.Row) bool {
	for _, value := range row.Cells {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func normalizeEntertainmentImportHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "　", "")
	value = strings.ReplaceAll(value, "\n", "")
	value = strings.ReplaceAll(value, "\r", "")
	return strings.ToLower(value)
}

func entertainmentImportCell(row sharedxlsx.Row, columns map[string]int, name string) string {
	column, ok := columns[name]
	if !ok {
		return ""
	}
	return strings.TrimSpace(row.Cells[column])
}

func normalizeImportedMediaType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "小说", "书", "novel":
		return MediaTypeNovel
	case "电视剧", "剧集", "电视", "series":
		return MediaTypeSeries
	case "电影", "movie":
		return MediaTypeMovie
	default:
		return strings.TrimSpace(value)
	}
}

func normalizeImportedStatus(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "计划", "计划中", "想读", "想看", "待看", "planned":
		return StatusPlanned
	case "进行中", "在读", "在看", "阅读中", "观看中", "in_progress", "in progress":
		return StatusInProgress
	case "完成", "已完成", "已读", "已看", "completed":
		return StatusCompleted
	case "搁置", "暂停", "paused":
		return StatusPaused
	case "放弃", "弃读", "弃看", "dropped":
		return StatusDropped
	default:
		return strings.TrimSpace(value)
	}
}

func splitEntertainmentImportTags(value string) []string {
	values := strings.FieldsFunc(value, func(character rune) bool {
		return character == ';' || character == '；' || character == ',' || character == '，' || character == '\n' || character == '\r'
	})
	result := make([]string, 0, len(values))
	for _, item := range values {
		item = strings.TrimSpace(item)
		if item != "" {
			result = append(result, item)
		}
	}
	return result
}

func parseEntertainmentImportIntValue(current int, value, label string, messages []string) (int, []string) {
	if strings.TrimSpace(value) == "" {
		return current, messages
	}
	parsed, err := parseEntertainmentImportInteger(value)
	if err != nil {
		return current, append(messages, label+"必须是整数")
	}
	return parsed, messages
}

func parseEntertainmentImportOptionalInt(value, label string, messages []string) (*int, []string) {
	if strings.TrimSpace(value) == "" {
		return nil, messages
	}
	parsed, err := parseEntertainmentImportInteger(value)
	if err != nil {
		return nil, append(messages, label+"必须是整数")
	}
	return &parsed, messages
}

func parseEntertainmentImportInteger(value string) (int, error) {
	value = strings.TrimSpace(value)
	if parsed, err := strconv.Atoi(value); err == nil {
		return parsed, nil
	}
	number, err := strconv.ParseFloat(value, 64)
	if err != nil || math.Trunc(number) != number || math.IsInf(number, 0) || math.IsNaN(number) {
		return 0, fmt.Errorf("invalid integer")
	}
	return strconv.Atoi(strconv.FormatFloat(number, 'f', 0, 64))
}

func parseEntertainmentImportBool(value string, messages []string) (bool, []string) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "否", "false", "0", "no", "n":
		return false, messages
	case "是", "true", "1", "yes", "y", "收藏":
		return true, messages
	default:
		return false, append(messages, "收藏必须填写“是”或“否”")
	}
}

func parseEntertainmentImportDate(value, label string, messages []string) (string, []string) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", messages
	}
	value = strings.ReplaceAll(value, "/", "-")
	datePart := strings.Fields(value)
	if len(datePart) > 0 {
		for _, layout := range []string{"2006-01-02", "2006-1-2"} {
			if parsed, err := time.ParseInLocation(layout, datePart[0], time.Local); err == nil {
				return parsed.Format(dateLayout), messages
			}
		}
	}
	return "", append(messages, label+"格式不正确，请使用 YYYY-MM-DD")
}

func entertainmentImportKey(input Input) string {
	return strings.ToLower(strings.TrimSpace(input.MediaType)) + "\x00" +
		strings.ToLower(strings.TrimSpace(input.Title)) + "\x00" +
		strings.ToLower(strings.TrimSpace(input.Creator))
}
