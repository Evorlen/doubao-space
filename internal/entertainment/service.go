package entertainment

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

const (
	MediaTypeNovel  = "novel"
	MediaTypeSeries = "series"
	MediaTypeMovie  = "movie"
)

const (
	StatusPlanned    = "planned"
	StatusInProgress = "in_progress"
	StatusCompleted  = "completed"
	StatusPaused     = "paused"
	StatusDropped    = "dropped"
)

var ErrNotFound = errors.New("entertainment item not found")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type Item struct {
	ID              int64    `json:"id"`
	MediaType       string   `json:"media_type"`
	MediaTypeLabel  string   `json:"media_type_label"`
	Title           string   `json:"title"`
	Creator         string   `json:"creator"`
	ReleaseYear     *int     `json:"release_year"`
	Source          string   `json:"source"`
	Status          string   `json:"status"`
	StatusLabel     string   `json:"status_label"`
	ProgressCurrent int      `json:"progress_current"`
	ProgressTotal   *int     `json:"progress_total"`
	ProgressPercent int      `json:"progress_percent"`
	ProgressLabel   string   `json:"progress_label"`
	Rating          int      `json:"rating"`
	Review          string   `json:"review"`
	Note            string   `json:"note"`
	Favorite        bool     `json:"favorite"`
	Tags            []string `json:"tags"`
	StartedOn       *string  `json:"started_on"`
	CompletedOn     *string  `json:"completed_on"`
	CreatedAt       string   `json:"created_at"`
	UpdatedAt       string   `json:"updated_at"`
}

type Stats struct {
	Total      int `json:"total"`
	InProgress int `json:"in_progress"`
	Completed  int `json:"completed"`
	Favorites  int `json:"favorites"`
	Unrated    int `json:"unrated"`
}

type ListResponse struct {
	Items []Item `json:"items"`
	Stats Stats  `json:"stats"`
}

type Input struct {
	MediaType       string   `json:"media_type"`
	Title           string   `json:"title"`
	Creator         string   `json:"creator"`
	ReleaseYear     *int     `json:"release_year"`
	Source          string   `json:"source"`
	Status          string   `json:"status"`
	ProgressCurrent int      `json:"progress_current"`
	ProgressTotal   *int     `json:"progress_total"`
	Rating          int      `json:"rating"`
	Review          string   `json:"review"`
	Note            string   `json:"note"`
	Favorite        bool     `json:"favorite"`
	Tags            []string `json:"tags"`
	StartedOn       string   `json:"started_on"`
	CompletedOn     string   `json:"completed_on"`
}

type ListOptions struct {
	MediaType string
	Status    string
	Search    string
	Rating    int
	Favorite  bool
	Sort      string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, options ListOptions) (*ListResponse, error) {
	items, err := s.listItems(ctx, options)
	if err != nil {
		return nil, err
	}
	stats, err := s.stats(ctx)
	if err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Stats: stats}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Item, error) {
	items, err := s.queryItems(ctx, "i.id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (s *Service) Create(ctx context.Context, input Input) (*Item, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create entertainment item: %w", err)
	}
	defer tx.Rollback()

	id, err := insertItem(ctx, tx, normalized)
	if err != nil {
		return nil, err
	}
	if err := replaceTags(ctx, tx, id, normalized.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit entertainment item: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input Input) (*Item, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update entertainment item: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE entertainment_items
		SET media_type = ?, title = ?, creator = ?, release_year = ?, source = ?, status = ?,
		    progress_current = ?, progress_total = ?, rating = ?, review = ?, note = ?, favorite = ?,
		    started_on = ?, completed_on = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, normalized.MediaType, normalized.Title, normalized.Creator, normalized.ReleaseYear, normalized.Source,
		normalized.Status, normalized.ProgressCurrent, normalized.ProgressTotal, normalized.Rating,
		normalized.Review, normalized.Note, boolValue(normalized.Favorite), nullableDate(normalized.StartedOn),
		nullableDate(normalized.CompletedOn), id)
	if err != nil {
		return nil, fmt.Errorf("update entertainment item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated entertainment item count: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	if err := replaceTags(ctx, tx, id, normalized.Tags); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit entertainment item update: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM entertainment_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete entertainment item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted entertainment item count: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) listItems(ctx context.Context, options ListOptions) ([]Item, error) {
	where := make([]string, 0, 5)
	args := make([]any, 0, 6)
	if options.MediaType != "" {
		if !isMediaType(options.MediaType) {
			return nil, &ValidationError{Message: "作品类型不正确"}
		}
		where = append(where, "i.media_type = ?")
		args = append(args, options.MediaType)
	}
	if options.Status != "" {
		if !isStatus(options.Status) {
			return nil, &ValidationError{Message: "作品状态不正确"}
		}
		where = append(where, "i.status = ?")
		args = append(args, options.Status)
	}
	if query := strings.TrimSpace(options.Search); query != "" {
		pattern := "%" + strings.ToLower(query) + "%"
		where = append(where, "(LOWER(i.title) LIKE ? OR LOWER(i.creator) LIKE ? OR LOWER(i.review) LIKE ? OR LOWER(i.note) LIKE ?)")
		args = append(args, pattern, pattern, pattern, pattern)
	}
	if options.Rating != 0 {
		if options.Rating < 1 || options.Rating > 5 {
			return nil, &ValidationError{Message: "评分必须在 1 到 5 星之间"}
		}
		where = append(where, "i.rating >= ?")
		args = append(args, options.Rating)
	}
	if options.Favorite {
		where = append(where, "i.favorite = 1")
	}

	orderBy := "i.updated_at DESC, i.id DESC"
	switch options.Sort {
	case "title":
		orderBy = "i.title COLLATE NOCASE ASC, i.id ASC"
	case "rating":
		orderBy = "i.rating DESC, i.updated_at DESC, i.id DESC"
	case "completed":
		orderBy = "i.completed_on DESC, i.updated_at DESC, i.id DESC"
	case "created":
		orderBy = "i.created_at DESC, i.id DESC"
	case "", "updated":
	default:
		return nil, &ValidationError{Message: "排序方式不正确"}
	}

	condition := ""
	if len(where) > 0 {
		condition = " WHERE " + strings.Join(where, " AND ")
	}
	query := fmt.Sprintf(`
		SELECT i.id, i.media_type, i.title, i.creator, i.release_year, i.source, i.status,
		       i.progress_current, i.progress_total, i.rating, i.review, i.note, i.favorite,
		       i.started_on, i.completed_on, i.created_at, i.updated_at
		FROM entertainment_items i%s
		ORDER BY %s
	`, condition, orderBy)
	return s.queryItems(ctx, query, args...)
}

func (s *Service) queryItems(ctx context.Context, conditionOrQuery string, args ...any) ([]Item, error) {
	query := conditionOrQuery
	if strings.HasPrefix(strings.TrimSpace(conditionOrQuery), "i.") {
		query = fmt.Sprintf(`
			SELECT i.id, i.media_type, i.title, i.creator, i.release_year, i.source, i.status,
			       i.progress_current, i.progress_total, i.rating, i.review, i.note, i.favorite,
			       i.started_on, i.completed_on, i.created_at, i.updated_at
			FROM entertainment_items i
			WHERE %s
		`, conditionOrQuery)
	}

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query entertainment items: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0)
	for rows.Next() {
		item, err := scanItem(rows)
		if err != nil {
			return nil, fmt.Errorf("scan entertainment item: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read entertainment items: %w", err)
	}
	if err := s.loadTags(ctx, items); err != nil {
		return nil, err
	}
	for index := range items {
		decorate(&items[index])
	}
	return items, nil
}

func (s *Service) stats(ctx context.Context) (Stats, error) {
	var stats Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1),
		       COALESCE(SUM(CASE WHEN status = 'in_progress' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN status = 'completed' THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN favorite = 1 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN rating = 0 THEN 1 ELSE 0 END), 0)
		FROM entertainment_items
	`).Scan(&stats.Total, &stats.InProgress, &stats.Completed, &stats.Favorites, &stats.Unrated)
	if err != nil {
		return Stats{}, fmt.Errorf("read entertainment stats: %w", err)
	}
	return stats, nil
}

func (s *Service) loadTags(ctx context.Context, items []Item) error {
	if len(items) == 0 {
		return nil
	}
	placeholders := make([]string, len(items))
	args := make([]any, len(items))
	indexByID := make(map[int64]int, len(items))
	for index, item := range items {
		placeholders[index] = "?"
		args[index] = item.ID
		indexByID[item.ID] = index
		items[index].Tags = []string{}
	}
	query := fmt.Sprintf(`
		SELECT it.item_id, t.name
		FROM entertainment_item_tags it
		JOIN entertainment_tags t ON t.id = it.tag_id
		WHERE it.item_id IN (%s)
		ORDER BY t.name COLLATE NOCASE ASC
	`, strings.Join(placeholders, ", "))
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("list entertainment tags: %w", err)
	}
	defer rows.Close()
	for rows.Next() {
		var itemID int64
		var name string
		if err := rows.Scan(&itemID, &name); err != nil {
			return fmt.Errorf("scan entertainment tag: %w", err)
		}
		if index, ok := indexByID[itemID]; ok {
			items[index].Tags = append(items[index].Tags, name)
		}
	}
	if err := rows.Err(); err != nil {
		return fmt.Errorf("read entertainment tags: %w", err)
	}
	return nil
}

func insertItem(ctx context.Context, tx *sql.Tx, input Input) (int64, error) {
	result, err := tx.ExecContext(ctx, `
		INSERT INTO entertainment_items (
			media_type, title, creator, release_year, source, status,
			progress_current, progress_total, rating, review, note, favorite,
			started_on, completed_on
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, input.MediaType, input.Title, input.Creator, input.ReleaseYear, input.Source, input.Status,
		input.ProgressCurrent, input.ProgressTotal, input.Rating, input.Review, input.Note,
		boolValue(input.Favorite), nullableDate(input.StartedOn), nullableDate(input.CompletedOn))
	if err != nil {
		return 0, fmt.Errorf("insert entertainment item: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return 0, fmt.Errorf("read entertainment item id: %w", err)
	}
	return id, nil
}

func replaceTags(ctx context.Context, tx *sql.Tx, itemID int64, tags []string) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM entertainment_item_tags WHERE item_id = ?`, itemID); err != nil {
		return fmt.Errorf("clear entertainment tags: %w", err)
	}
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT OR IGNORE INTO entertainment_tags (name) VALUES (?)`, tag); err != nil {
			return fmt.Errorf("create entertainment tag: %w", err)
		}
		var tagID int64
		if err := tx.QueryRowContext(ctx, `SELECT id FROM entertainment_tags WHERE name = ?`, tag).Scan(&tagID); err != nil {
			return fmt.Errorf("read entertainment tag: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO entertainment_item_tags (item_id, tag_id) VALUES (?, ?)
		`, itemID, tagID); err != nil {
			return fmt.Errorf("link entertainment tag: %w", err)
		}
	}
	return nil
}

func scanItem(row interface{ Scan(dest ...any) error }) (Item, error) {
	var (
		item          Item
		releaseYear   sql.NullInt64
		progressTotal sql.NullInt64
		favorite      int64
		startedOn     sql.NullString
		completedOn   sql.NullString
	)
	if err := row.Scan(
		&item.ID, &item.MediaType, &item.Title, &item.Creator, &releaseYear, &item.Source, &item.Status,
		&item.ProgressCurrent, &progressTotal, &item.Rating, &item.Review, &item.Note, &favorite,
		&startedOn, &completedOn, &item.CreatedAt, &item.UpdatedAt,
	); err != nil {
		return Item{}, err
	}
	if releaseYear.Valid {
		value := int(releaseYear.Int64)
		item.ReleaseYear = &value
	}
	if progressTotal.Valid {
		value := int(progressTotal.Int64)
		item.ProgressTotal = &value
	}
	if startedOn.Valid {
		value := startedOn.String
		item.StartedOn = &value
	}
	if completedOn.Valid {
		value := completedOn.String
		item.CompletedOn = &value
	}
	item.Favorite = favorite == 1
	return item, nil
}

func normalizeInput(input Input) (Input, error) {
	input.MediaType = strings.TrimSpace(input.MediaType)
	input.Title = strings.TrimSpace(input.Title)
	input.Creator = strings.TrimSpace(input.Creator)
	input.Source = strings.TrimSpace(input.Source)
	input.Status = strings.TrimSpace(input.Status)
	input.Review = strings.TrimSpace(input.Review)
	input.Note = strings.TrimSpace(input.Note)
	input.StartedOn = strings.TrimSpace(input.StartedOn)
	input.CompletedOn = strings.TrimSpace(input.CompletedOn)

	if !isMediaType(input.MediaType) {
		return Input{}, &ValidationError{Message: "请选择小说、电视剧或电影"}
	}
	if input.Title == "" {
		return Input{}, &ValidationError{Message: "请输入作品名称"}
	}
	if utf8.RuneCountInString(input.Title) > 120 {
		return Input{}, &ValidationError{Message: "作品名称不能超过 120 个字"}
	}
	if utf8.RuneCountInString(input.Creator) > 100 {
		return Input{}, &ValidationError{Message: "作者或导演不能超过 100 个字"}
	}
	if utf8.RuneCountInString(input.Source) > 100 {
		return Input{}, &ValidationError{Message: "来源不能超过 100 个字"}
	}
	if utf8.RuneCountInString(input.Review) > 2000 {
		return Input{}, &ValidationError{Message: "个人评价不能超过 2000 个字"}
	}
	if utf8.RuneCountInString(input.Note) > 500 {
		return Input{}, &ValidationError{Message: "备注不能超过 500 个字"}
	}
	if input.Status == "" {
		input.Status = StatusPlanned
	}
	if !isStatus(input.Status) {
		return Input{}, &ValidationError{Message: "作品状态不正确"}
	}
	if input.ReleaseYear != nil && (*input.ReleaseYear < 1000 || *input.ReleaseYear > 9999) {
		return Input{}, &ValidationError{Message: "年份格式不正确"}
	}
	if input.Rating < 0 || input.Rating > 5 {
		return Input{}, &ValidationError{Message: "评分必须在 0 到 5 星之间"}
	}
	if input.ProgressCurrent < 0 {
		return Input{}, &ValidationError{Message: "当前进度不能小于 0"}
	}
	if input.ProgressTotal != nil {
		if *input.ProgressTotal < 1 {
			return Input{}, &ValidationError{Message: "总进度必须至少为 1"}
		}
		if input.ProgressCurrent > *input.ProgressTotal {
			return Input{}, &ValidationError{Message: "当前进度不能超过总进度"}
		}
	}
	if input.MediaType == MediaTypeMovie {
		input.ProgressCurrent = 0
		input.ProgressTotal = nil
	}
	if input.Status == StatusCompleted && input.ProgressTotal != nil {
		input.ProgressCurrent = *input.ProgressTotal
	}
	if input.StartedOn != "" {
		if _, err := parseDate(input.StartedOn); err != nil {
			return Input{}, &ValidationError{Message: "开始日期格式不正确，请使用 YYYY-MM-DD"}
		}
	}
	if input.CompletedOn != "" {
		if _, err := parseDate(input.CompletedOn); err != nil {
			return Input{}, &ValidationError{Message: "完成日期格式不正确，请使用 YYYY-MM-DD"}
		}
	}
	today := time.Now().Format(dateLayout)
	if input.Status == StatusInProgress && input.StartedOn == "" {
		input.StartedOn = today
	}
	if input.Status == StatusCompleted {
		if input.StartedOn == "" {
			input.StartedOn = today
		}
		if input.CompletedOn == "" {
			input.CompletedOn = today
		}
	} else {
		input.CompletedOn = ""
	}
	input.Tags = normalizeTags(input.Tags)
	return input, nil
}

func normalizeTags(values []string) []string {
	seen := make(map[string]bool, len(values))
	tags := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || utf8.RuneCountInString(value) > 30 {
			continue
		}
		key := strings.ToLower(value)
		if seen[key] {
			continue
		}
		seen[key] = true
		tags = append(tags, value)
		if len(tags) >= 12 {
			break
		}
	}
	sort.SliceStable(tags, func(i, j int) bool {
		return strings.ToLower(tags[i]) < strings.ToLower(tags[j])
	})
	return tags
}

func decorate(item *Item) {
	item.MediaTypeLabel = mediaTypeLabel(item.MediaType)
	item.StatusLabel = statusLabel(item.MediaType, item.Status)
	if item.ProgressTotal != nil {
		item.ProgressPercent = item.ProgressCurrent * 100 / *item.ProgressTotal
		unit := "集"
		if item.MediaType == MediaTypeNovel {
			unit = "章"
		}
		item.ProgressLabel = fmt.Sprintf("第 %d / %d %s", item.ProgressCurrent, *item.ProgressTotal, unit)
	} else if item.ProgressCurrent > 0 {
		unit := "集"
		if item.MediaType == MediaTypeNovel {
			unit = "章"
		}
		item.ProgressLabel = fmt.Sprintf("第 %d %s", item.ProgressCurrent, unit)
	}
	if item.Tags == nil {
		item.Tags = []string{}
	}
}

func mediaTypeLabel(mediaType string) string {
	switch mediaType {
	case MediaTypeNovel:
		return "小说"
	case MediaTypeSeries:
		return "电视剧"
	case MediaTypeMovie:
		return "电影"
	default:
		return "作品"
	}
}

func statusLabel(mediaType, status string) string {
	reading := mediaType == MediaTypeNovel
	switch status {
	case StatusPlanned:
		if reading {
			return "想读"
		}
		return "想看"
	case StatusInProgress:
		if reading {
			return "在读"
		}
		return "在看"
	case StatusCompleted:
		if reading {
			return "已读"
		}
		return "已看"
	case StatusPaused:
		return "搁置"
	case StatusDropped:
		if reading {
			return "弃读"
		}
		return "弃看"
	default:
		return "未分类"
	}
}

func isMediaType(value string) bool {
	return value == MediaTypeNovel || value == MediaTypeSeries || value == MediaTypeMovie
}

func isStatus(value string) bool {
	switch value {
	case StatusPlanned, StatusInProgress, StatusCompleted, StatusPaused, StatusDropped:
		return true
	default:
		return false
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, value, time.Local)
}

func nullableDate(value string) any {
	if value == "" {
		return nil
	}
	return value
}

func boolValue(value bool) int {
	if value {
		return 1
	}
	return 0
}
