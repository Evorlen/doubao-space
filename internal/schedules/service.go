package schedules

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

const dateLayout = "2006-01-02"

var ErrNotFound = errors.New("doubao schedule not found")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

type Schedule struct {
	ID              int64   `json:"id"`
	Title           string  `json:"title"`
	Note            string  `json:"note"`
	StartDate       string  `json:"start_date"`
	IntervalDays    int     `json:"interval_days"`
	LastCompletedOn *string `json:"last_completed_on"`
	NextDueDate     string  `json:"next_due_date"`
	IsDue           bool    `json:"is_due"`
	Status          string  `json:"status"`
	OverdueDays     int     `json:"overdue_days"`
	DaysUntilDue    int     `json:"days_until_due"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type Plan struct {
	Today         string     `json:"today"`
	TodayLabel    string     `json:"today_label"`
	Schedules     []Schedule `json:"schedules"`
	DueCount      int        `json:"due_count"`
	UpcomingCount int        `json:"upcoming_count"`
}

type ScheduleInput struct {
	Title        string
	Note         string
	StartDate    string
	IntervalDays int
}

type scheduleRecord struct {
	ID              int64
	Title           string
	Note            string
	StartDate       string
	IntervalDays    int
	LastCompletedOn *string
	CreatedAt       string
	UpdatedAt       string
}

type rowScanner interface {
	Scan(dest ...any) error
}

type Service struct {
	db              *sql.DB
	scheduleTable   string
	completionTable string
	completionKey   string
	resourceName    string
}

func NewService(db *sql.DB) *Service {
	return newScopedService(db, "doubao_schedules", "doubao_schedule_completions", "schedule_id", "日程")
}

func NewPersonalService(db *sql.DB) *Service {
	return newScopedService(db, "personal_plans", "personal_plan_completions", "plan_id", "计划")
}

func newScopedService(db *sql.DB, scheduleTable, completionTable, completionKey, resourceName string) *Service {
	return &Service{
		db:              db,
		scheduleTable:   scheduleTable,
		completionTable: completionTable,
		completionKey:   completionKey,
		resourceName:    resourceName,
	}
}

func (s *Service) GetPlan(ctx context.Context) (*Plan, error) {
	records, err := s.listRecords(ctx)
	if err != nil {
		return nil, err
	}
	today := localDate(time.Now())
	items := make([]Schedule, 0, len(records))
	for _, record := range records {
		item, err := decorate(record, today)
		if err != nil {
			return nil, err
		}
		items = append(items, item)
	}
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].IsDue != items[j].IsDue {
			return items[i].IsDue
		}
		if items[i].NextDueDate != items[j].NextDueDate {
			return items[i].NextDueDate < items[j].NextDueDate
		}
		return items[i].ID < items[j].ID
	})

	dueCount := 0
	for _, item := range items {
		if item.IsDue {
			dueCount++
		}
	}
	return &Plan{
		Today:         today.Format(dateLayout),
		TodayLabel:    today.Format("2006年1月2日"),
		Schedules:     items,
		DueCount:      dueCount,
		UpcomingCount: len(items) - dueCount,
	}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Schedule, error) {
	record, err := s.getRecord(ctx, id)
	if err != nil {
		return nil, err
	}
	item, err := decorate(record, localDate(time.Now()))
	if err != nil {
		return nil, err
	}
	return &item, nil
}

func (s *Service) Create(ctx context.Context, input ScheduleInput) (*Schedule, error) {
	normalized, err := normalizeInput(input, s.resourceName)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (title, note, start_date, interval_days)
		VALUES (?, ?, ?, ?)
	`, s.scheduleTable), normalized.Title, normalized.Note, normalized.StartDate, normalized.IntervalDays)
	if err != nil {
		return nil, fmt.Errorf("create %s: %w", s.resourceName, err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read created %s id: %w", s.resourceName, err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input ScheduleInput) (*Schedule, error) {
	normalized, err := normalizeInput(input, s.resourceName)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET title = ?, note = ?, start_date = ?, interval_days = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, s.scheduleTable), normalized.Title, normalized.Note, normalized.StartDate, normalized.IntervalDays, id)
	if err != nil {
		return nil, fmt.Errorf("update %s: %w", s.resourceName, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated %s count: %w", s.resourceName, err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *Service) Complete(ctx context.Context, id int64) (*Schedule, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin complete %s: %w", s.resourceName, err)
	}
	defer tx.Rollback()

	record, err := getRecordFromRow(tx.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, title, note, start_date, interval_days, last_completed_on, created_at, updated_at
		FROM %s
		WHERE id = ?
	`, s.scheduleTable), id))
	if err != nil {
		return nil, err
	}

	today := localDate(time.Now())
	if record.LastCompletedOn != nil && *record.LastCompletedOn == today.Format(dateLayout) {
		return nil, &ConflictError{Message: fmt.Sprintf("这个%s今天已经完成过了", s.resourceName)}
	}
	nextDue, err := nextDueDate(record)
	if err != nil {
		return nil, err
	}
	if today.Before(nextDue) {
		return nil, &ConflictError{Message: fmt.Sprintf("这个%s还没到提醒日期，暂时不能提前完成", s.resourceName)}
	}

	todayValue := today.Format(dateLayout)
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		INSERT INTO %s (%s, completed_on)
		VALUES (?, ?)
	`, s.completionTable, s.completionKey), id, todayValue); err != nil {
		if isConstraintError(err) {
			return nil, &ConflictError{Message: fmt.Sprintf("这个%s今天已经完成过了", s.resourceName)}
		}
		return nil, fmt.Errorf("record %s completion: %w", s.resourceName, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`
		UPDATE %s
		SET last_completed_on = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, s.scheduleTable), todayValue, id); err != nil {
		return nil, fmt.Errorf("update completed %s: %w", s.resourceName, err)
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit %s completion: %w", s.resourceName, err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, fmt.Sprintf(`DELETE FROM %s WHERE id = ?`, s.scheduleTable), id)
	if err != nil {
		return fmt.Errorf("delete %s: %w", s.resourceName, err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted %s count: %w", s.resourceName, err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) listRecords(ctx context.Context) ([]scheduleRecord, error) {
	rows, err := s.db.QueryContext(ctx, fmt.Sprintf(`
		SELECT id, title, note, start_date, interval_days, last_completed_on, created_at, updated_at
		FROM %s
		ORDER BY id ASC
	`, s.scheduleTable))
	if err != nil {
		return nil, fmt.Errorf("list %s: %w", s.resourceName, err)
	}
	defer rows.Close()

	records := make([]scheduleRecord, 0)
	for rows.Next() {
		record, err := getRecordFromRow(rows)
		if err != nil {
			return nil, fmt.Errorf("scan %s: %w", s.resourceName, err)
		}
		records = append(records, record)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read %s: %w", s.resourceName, err)
	}
	return records, nil
}

func (s *Service) getRecord(ctx context.Context, id int64) (scheduleRecord, error) {
	record, err := getRecordFromRow(s.db.QueryRowContext(ctx, fmt.Sprintf(`
		SELECT id, title, note, start_date, interval_days, last_completed_on, created_at, updated_at
		FROM %s
		WHERE id = ?
	`, s.scheduleTable), id))
	if errors.Is(err, sql.ErrNoRows) {
		return scheduleRecord{}, ErrNotFound
	}
	if err != nil {
		return scheduleRecord{}, fmt.Errorf("get %s: %w", s.resourceName, err)
	}
	return record, nil
}

func getRecordFromRow(row rowScanner) (scheduleRecord, error) {
	var (
		record          scheduleRecord
		lastCompletedOn sql.NullString
	)
	if err := row.Scan(
		&record.ID,
		&record.Title,
		&record.Note,
		&record.StartDate,
		&record.IntervalDays,
		&lastCompletedOn,
		&record.CreatedAt,
		&record.UpdatedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return scheduleRecord{}, ErrNotFound
		}
		return scheduleRecord{}, err
	}
	if lastCompletedOn.Valid {
		value := lastCompletedOn.String
		record.LastCompletedOn = &value
	}
	return record, nil
}

func normalizeInput(input ScheduleInput, resourceName string) (ScheduleInput, error) {
	input.Title = strings.TrimSpace(input.Title)
	input.Note = strings.TrimSpace(input.Note)
	input.StartDate = strings.TrimSpace(input.StartDate)
	if input.Title == "" {
		return ScheduleInput{}, &ValidationError{Message: fmt.Sprintf("请输入%s名称", resourceName)}
	}
	if utf8.RuneCountInString(input.Title) > 100 {
		return ScheduleInput{}, &ValidationError{Message: fmt.Sprintf("%s名称不能超过 100 个字", resourceName)}
	}
	if utf8.RuneCountInString(input.Note) > 300 {
		return ScheduleInput{}, &ValidationError{Message: "备注不能超过 300 个字"}
	}
	if input.IntervalDays < 1 || input.IntervalDays > 3650 {
		return ScheduleInput{}, &ValidationError{Message: "间隔天数必须在 1 到 3650 天之间"}
	}
	if _, err := parseDate(input.StartDate); err != nil {
		return ScheduleInput{}, &ValidationError{Message: "开始日期格式不正确，请使用 YYYY-MM-DD"}
	}
	return input, nil
}

func decorate(record scheduleRecord, today time.Time) (Schedule, error) {
	nextDue, err := nextDueDate(record)
	if err != nil {
		return Schedule{}, err
	}
	nextDue = localDate(nextDue)
	item := Schedule{
		ID:              record.ID,
		Title:           record.Title,
		Note:            record.Note,
		StartDate:       record.StartDate,
		IntervalDays:    record.IntervalDays,
		LastCompletedOn: record.LastCompletedOn,
		NextDueDate:     nextDue.Format(dateLayout),
		CreatedAt:       record.CreatedAt,
		UpdatedAt:       record.UpdatedAt,
	}
	if nextDue.After(today) {
		item.Status = "upcoming"
		item.DaysUntilDue = daysBetween(nextDue, today)
		return item, nil
	}
	item.IsDue = true
	item.Status = "due"
	item.OverdueDays = daysBetween(today, nextDue)
	return item, nil
}

func nextDueDate(record scheduleRecord) (time.Time, error) {
	base, err := parseDate(record.StartDate)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse schedule start date: %w", err)
	}
	if record.LastCompletedOn == nil {
		return base, nil
	}
	lastCompleted, err := parseDate(*record.LastCompletedOn)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse schedule completion date: %w", err)
	}
	return lastCompleted.AddDate(0, 0, record.IntervalDays), nil
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, value, time.Local)
}

func localDate(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

func daysBetween(later, earlier time.Time) int {
	toUTCDate := func(value time.Time) time.Time {
		return time.Date(value.Year(), value.Month(), value.Day(), 0, 0, 0, 0, time.UTC)
	}
	return int(toUTCDate(later).Sub(toUTCDate(earlier)).Hours() / 24)
}

func isConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "constraint") || strings.Contains(message, "unique")
}
