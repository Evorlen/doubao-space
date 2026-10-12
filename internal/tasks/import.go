package tasks

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"
	"time"
)

const maxImportKeyLength = 100

type TaskImportRowResult struct {
	Row     int    `json:"row"`
	Title   string `json:"title"`
	Status  string `json:"status"`
	Message string `json:"message"`
}

type TaskImportReport struct {
	Total          int                   `json:"total"`
	NewCount       int                   `json:"new_count"`
	DuplicateCount int                   `json:"duplicate_count"`
	ErrorCount     int                   `json:"error_count"`
	CreatedCount   int                   `json:"created_count"`
	CanImport      bool                  `json:"can_import"`
	Rows           []TaskImportRowResult `json:"rows"`
}

type taskImportRow struct {
	Number          int
	Title           string
	ImportKey       string
	Description     string
	DueAt           string
	Priority        string
	Tags            []string
	Links           []LinkInput
	ValidationError string
}

type taskImportCandidate struct {
	ReportIndex int
	Task        *Task
	Links       []LinkInput
}

func (s *Service) PreviewImport(ctx context.Context, parentID int64, reader io.Reader) (TaskImportReport, error) {
	return s.processImport(ctx, parentID, reader, false)
}

func (s *Service) Import(ctx context.Context, parentID int64, reader io.Reader) (TaskImportReport, error) {
	return s.processImport(ctx, parentID, reader, true)
}

func (s *Service) processImport(ctx context.Context, parentID int64, reader io.Reader, commit bool) (TaskImportReport, error) {
	rows, err := parseTaskImportRows(reader)
	if err != nil {
		return TaskImportReport{}, err
	}

	roots, err := s.loadForest(ctx)
	if err != nil {
		return TaskImportReport{}, err
	}
	if err := validateForest(roots); err != nil {
		return TaskImportReport{}, err
	}
	parent := findTask(roots, parentID)
	if parent == nil {
		return TaskImportReport{}, ErrNotFound
	}
	if parent.Completed {
		return TaskImportReport{}, &ConflictError{Message: "已完成任务不能导入子任务。"}
	}
	if parent.Depth >= MaxDepth {
		return TaskImportReport{}, &ConflictError{Message: fmt.Sprintf("任务树最多支持 %d 层，当前任务已经位于第 %d 层。", MaxDepth, parent.Depth)}
	}

	report := TaskImportReport{
		Total: len(rows),
		Rows:  make([]TaskImportRowResult, 0, len(rows)),
	}
	existingCodes := make(map[string]struct{})
	existingFallbacks := make(map[string]struct{})
	existingStoredKeys := make(map[string]struct{})
	for _, child := range parent.Children {
		if child.ImportKey != "" {
			existingStoredKeys[child.ImportKey] = struct{}{}
		}
		if code, ok := storedImportCode(child.ImportKey); ok {
			existingCodes[code] = struct{}{}
		}
		existingFallbacks[taskImportFallbackKey(child.Title, child.DueAt)] = struct{}{}
	}
	seenCodes := make(map[string]struct{})
	seenFallbacks := make(map[string]struct{})
	candidates := make([]taskImportCandidate, 0, len(rows))

	for _, row := range rows {
		result := TaskImportRowResult{Row: row.Number, Title: row.Title}
		report.Rows = append(report.Rows, result)
		reportIndex := len(report.Rows) - 1

		if row.ValidationError != "" {
			report.Rows[reportIndex].Status = "error"
			report.Rows[reportIndex].Message = row.ValidationError
			report.ErrorCount++
			continue
		}

		title, description, priority, dueAt, tags, links, normalizeErr := normalizePayload(
			row.Title,
			row.Description,
			row.Priority,
			row.DueAt,
			row.Tags,
			row.Links,
		)
		if normalizeErr != nil {
			report.Rows[reportIndex].Status = "error"
			report.Rows[reportIndex].Message = normalizeErr.Error()
			report.ErrorCount++
			continue
		}

		fallbackKey := taskImportFallbackKey(title, dueAt)
		storedKey := taskImportStoredKey(row.ImportKey, fallbackKey)
		if row.ImportKey != "" {
			code := normalizeImportKey(row.ImportKey)
			if _, exists := existingCodes[code]; exists {
				report.Rows[reportIndex].Status = "duplicate"
				report.Rows[reportIndex].Message = "导入编号已存在，已跳过。"
				report.DuplicateCount++
				continue
			}
			if _, exists := seenCodes[code]; exists {
				report.Rows[reportIndex].Status = "duplicate"
				report.Rows[reportIndex].Message = "文件内导入编号重复，已跳过。"
				report.DuplicateCount++
				continue
			}
			seenCodes[code] = struct{}{}
		} else {
			if _, exists := existingStoredKeys[storedKey]; exists {
				report.Rows[reportIndex].Status = "duplicate"
				report.Rows[reportIndex].Message = "该任务已通过此前的 Excel 导入，已跳过。"
				report.DuplicateCount++
				continue
			}
			if _, exists := existingFallbacks[fallbackKey]; exists {
				report.Rows[reportIndex].Status = "duplicate"
				report.Rows[reportIndex].Message = "当前父任务下已存在相同标题和截止时间的任务，已跳过。"
				report.DuplicateCount++
				continue
			}
			if _, exists := seenFallbacks[fallbackKey]; exists {
				report.Rows[reportIndex].Status = "duplicate"
				report.Rows[reportIndex].Message = "文件内存在相同标题和截止时间的任务，已跳过。"
				report.DuplicateCount++
				continue
			}
			seenFallbacks[fallbackKey] = struct{}{}
		}

		parentIDCopy := parentID
		candidate := &Task{
			ParentID:    &parentIDCopy,
			Title:       title,
			Description: description,
			DueAt:       dueAt,
			Priority:    priority,
			ImportKey:   storedKey,
			Tags:        tags,
			Links:       linksToTasks(links),
			Children:    []*Task{},
		}
		parent.Children = append(parent.Children, candidate)
		validationErr := validateForest(roots)
		parent.Children = parent.Children[:len(parent.Children)-1]
		if validationErr != nil {
			report.Rows[reportIndex].Status = "error"
			report.Rows[reportIndex].Message = validationErr.Error()
			report.ErrorCount++
			continue
		}

		report.Rows[reportIndex].Status = "new"
		report.Rows[reportIndex].Message = "待导入"
		candidates = append(candidates, taskImportCandidate{
			ReportIndex: reportIndex,
			Task:        candidate,
			Links:       links,
		})
	}

	report.NewCount = len(candidates)
	report.CanImport = report.ErrorCount == 0 && report.NewCount > 0
	if report.ErrorCount > 0 || !commit || report.NewCount == 0 {
		return report, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return TaskImportReport{}, fmt.Errorf("begin import tasks: %w", err)
	}
	defer tx.Rollback()
	for _, candidate := range candidates {
		result, insertErr := tx.ExecContext(ctx, `
			INSERT INTO tasks (parent_id, title, description, due_at, priority, import_key)
			VALUES (?, ?, ?, ?, ?, ?)
		`, parentID, candidate.Task.Title, candidate.Task.Description, nullableTime(candidate.Task.DueAt), candidate.Task.Priority, candidate.Task.ImportKey)
		if insertErr != nil {
			return TaskImportReport{}, &ConflictError{Message: "导入过程中发现重复任务，请重新预览后再试。"}
		}
		id, idErr := result.LastInsertId()
		if idErr != nil {
			return TaskImportReport{}, fmt.Errorf("read imported task id: %w", idErr)
		}
		if associationErr := saveAssociations(ctx, tx, id, candidate.Task.Tags, candidate.Links); associationErr != nil {
			return TaskImportReport{}, associationErr
		}
		report.Rows[candidate.ReportIndex].Status = "created"
		report.Rows[candidate.ReportIndex].Message = "已创建"
	}
	if err := tx.Commit(); err != nil {
		return TaskImportReport{}, fmt.Errorf("commit imported tasks: %w", err)
	}
	report.CreatedCount = len(candidates)
	report.CanImport = false
	return report, nil
}

func parseTaskImportRows(reader io.Reader) ([]taskImportRow, error) {
	spreadsheetRows, err := parseXLSXRows(reader)
	if err != nil {
		return nil, err
	}
	if len(spreadsheetRows) == 0 {
		return nil, &InputError{Message: "Excel 文件中没有任务数据。"}
	}

	headerIndex := -1
	for index, row := range spreadsheetRows {
		if !spreadsheetRowIsBlank(row) {
			headerIndex = index
			break
		}
	}
	if headerIndex < 0 {
		return nil, &InputError{Message: "Excel 文件中没有任务数据。"}
	}

	columns := make(map[string]int)
	for column, value := range spreadsheetRows[headerIndex].Cells {
		name := normalizeImportHeader(value)
		if name == "" {
			continue
		}
		canonical, ok := importHeaderAliases[name]
		if !ok {
			return nil, &InputError{Message: fmt.Sprintf("Excel 包含无法识别的列“%s”。", strings.TrimSpace(value))}
		}
		if _, exists := columns[canonical]; exists {
			return nil, &InputError{Message: fmt.Sprintf("Excel 中重复出现了列“%s”。", strings.TrimSpace(value))}
		}
		columns[canonical] = column
	}
	if _, ok := columns["title"]; !ok {
		return nil, &InputError{Message: "Excel 必须包含“任务标题”列。"}
	}

	result := make([]taskImportRow, 0, len(spreadsheetRows)-headerIndex-1)
	for _, spreadsheetRow := range spreadsheetRows[headerIndex+1:] {
		if spreadsheetRowIsBlank(spreadsheetRow) {
			continue
		}
		row := taskImportRow{
			Number:      spreadsheetRow.Number,
			Title:       importCell(spreadsheetRow, columns, "title"),
			ImportKey:   importCell(spreadsheetRow, columns, "import_key"),
			Description: importCell(spreadsheetRow, columns, "description"),
			DueAt:       importCell(spreadsheetRow, columns, "due_at"),
			Priority:    normalizeImportPriority(importCell(spreadsheetRow, columns, "priority")),
			Tags:        splitImportTags(importCell(spreadsheetRow, columns, "tags")),
			Links:       splitImportLinks(importCell(spreadsheetRow, columns, "links")),
		}
		if len([]rune(row.ImportKey)) > maxImportKeyLength {
			row.ValidationError = fmt.Sprintf("导入编号不能超过 %d 个字符。", maxImportKeyLength)
		}
		result = append(result, row)
	}
	if len(result) == 0 {
		return nil, &InputError{Message: "Excel 中没有可导入的任务行。"}
	}
	return result, nil
}

var importHeaderAliases = map[string]string{
	"任务标题": "title",
	"任务名称": "title",
	"导入编号": "import_key",
	"任务编号": "import_key",
	"任务描述": "description",
	"任务说明": "description",
	"截止时间": "due_at",
	"优先级":  "priority",
	"标签":   "tags",
	"外部链接": "links",
	"链接":   "links",
}

func spreadsheetRowIsBlank(row spreadsheetRow) bool {
	for _, value := range row.Cells {
		if strings.TrimSpace(value) != "" {
			return false
		}
	}
	return true
}

func normalizeImportHeader(value string) string {
	value = strings.TrimSpace(value)
	value = strings.ReplaceAll(value, " ", "")
	value = strings.ReplaceAll(value, "　", "")
	value = strings.ReplaceAll(value, "\n", "")
	return value
}

func importCell(row spreadsheetRow, columns map[string]int, name string) string {
	column, ok := columns[name]
	if !ok {
		return ""
	}
	return strings.TrimSpace(row.Cells[column])
}

func normalizeImportPriority(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	switch value {
	case "紧急", "urgent":
		return PriorityUrgent
	case "高", "高优先级", "high":
		return PriorityHigh
	case "中", "中优先级", "medium":
		return PriorityMedium
	case "低", "低优先级", "low":
		return PriorityLow
	default:
		return value
	}
}

func splitImportTags(value string) []string {
	return splitImportValues(value, func(character rune) bool {
		return character == ';' || character == '；' || character == ',' || character == '，' || character == '\n' || character == '\r'
	})
}

func splitImportLinks(value string) []LinkInput {
	entries := splitImportValues(value, func(character rune) bool {
		return character == '；' || character == '\n' || character == '\r'
	})
	links := make([]LinkInput, 0, len(entries))
	for _, entry := range entries {
		parts := strings.SplitN(entry, "|", 2)
		if len(parts) == 2 {
			links = append(links, LinkInput{Title: strings.TrimSpace(parts[0]), URL: strings.TrimSpace(parts[1])})
			continue
		}
		links = append(links, LinkInput{URL: strings.TrimSpace(entry)})
	}
	return links
}

func splitImportValues(value string, separator func(rune) bool) []string {
	values := strings.FieldsFunc(value, separator)
	result := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			result = append(result, value)
		}
	}
	return result
}

func normalizeImportKey(value string) string {
	return strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(value)), " "))
}

func storedImportCode(value string) (string, bool) {
	if !strings.HasPrefix(value, "code:") {
		return "", false
	}
	return normalizeImportKey(strings.TrimPrefix(value, "code:")), true
}

func taskImportFallbackKey(title string, dueAt *time.Time) string {
	due := ""
	if dueAt != nil {
		due = dueAt.UTC().Format(time.RFC3339Nano)
	}
	title = strings.ToLower(strings.Join(strings.Fields(strings.TrimSpace(title)), " "))
	return title + "\x00" + due
}

func taskImportStoredKey(importKey, fallbackKey string) string {
	if normalized := normalizeImportKey(importKey); normalized != "" {
		return "code:" + normalized
	}
	digest := sha256.Sum256([]byte(fallbackKey))
	return "auto:" + hex.EncodeToString(digest[:])
}
