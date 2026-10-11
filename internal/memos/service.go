package memos

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"fmt"
	stdhtml "html"
	"net/url"
	"strings"
	"unicode/utf8"

	nethtml "golang.org/x/net/html"
)

const (
	maxTitleRunes   = 200
	maxContentBytes = 300 * 1024
	maxExcerptRunes = 180
)

var ErrNotFound = errors.New("memo not found")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type Memo struct {
	ID          int64   `json:"id"`
	Title       string  `json:"title"`
	ContentHTML string  `json:"content_html"`
	Excerpt     string  `json:"excerpt"`
	Archived    bool    `json:"archived"`
	ArchivedAt  *string `json:"archived_at"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

type Stats struct {
	Total    int `json:"total"`
	Active   int `json:"active"`
	Archived int `json:"archived"`
}

type ListResponse struct {
	Items []Memo `json:"items"`
	Stats Stats  `json:"stats"`
}

type Input struct {
	Title       string `json:"title"`
	ContentHTML string `json:"content_html"`
}

type ListOptions struct {
	Archived *bool
	Search   string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context, options ListOptions) (*ListResponse, error) {
	where := make([]string, 0, 2)
	args := make([]any, 0, 3)
	if options.Archived != nil {
		where = append(where, "m.archived = ?")
		args = append(args, boolValue(*options.Archived))
	}
	if search := strings.TrimSpace(options.Search); search != "" {
		pattern := "%" + strings.ToLower(search) + "%"
		where = append(where, "(LOWER(m.title) LIKE ? OR LOWER(m.content_html) LIKE ?)")
		args = append(args, pattern, pattern)
	}
	condition := ""
	if len(where) > 0 {
		condition = "WHERE " + strings.Join(where, " AND ")
	}

	items, err := s.query(ctx, condition, args...)
	if err != nil {
		return nil, err
	}
	stats, err := s.stats(ctx)
	if err != nil {
		return nil, err
	}
	return &ListResponse{Items: items, Stats: stats}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Memo, error) {
	items, err := s.query(ctx, "WHERE m.id = ?", id)
	if err != nil {
		return nil, err
	}
	if len(items) == 0 {
		return nil, ErrNotFound
	}
	return &items[0], nil
}

func (s *Service) Create(ctx context.Context, input Input) (*Memo, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `
		INSERT INTO memos (title, content_html)
		VALUES (?, ?)
	`, normalized.Title, normalized.ContentHTML)
	if err != nil {
		return nil, fmt.Errorf("create memo: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read created memo id: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input Input) (*Memo, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE memos
		SET title = ?, content_html = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, normalized.Title, normalized.ContentHTML, id)
	if err != nil {
		return nil, fmt.Errorf("update memo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated memo count: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *Service) SetArchived(ctx context.Context, id int64, archived bool) (*Memo, error) {
	result, err := s.db.ExecContext(ctx, `
		UPDATE memos
		SET archived = ?,
		    archived_at = CASE WHEN ? = 1 THEN CURRENT_TIMESTAMP ELSE NULL END,
		    updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, boolValue(archived), boolValue(archived), id)
	if err != nil {
		return nil, fmt.Errorf("update memo archive state: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read archived memo count: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM memos WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete memo: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted memo count: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) query(ctx context.Context, condition string, args ...any) ([]Memo, error) {
	query := fmt.Sprintf(`
		SELECT m.id, m.title, m.content_html, m.archived, m.archived_at, m.created_at, m.updated_at
		FROM memos m
		%s
		ORDER BY m.archived ASC, m.updated_at DESC, m.id DESC
	`, condition)
	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("query memos: %w", err)
	}
	defer rows.Close()

	items := make([]Memo, 0)
	for rows.Next() {
		item, err := scanMemo(rows)
		if err != nil {
			return nil, fmt.Errorf("scan memo: %w", err)
		}
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read memos: %w", err)
	}
	return items, nil
}

func (s *Service) stats(ctx context.Context) (Stats, error) {
	var stats Stats
	err := s.db.QueryRowContext(ctx, `
		SELECT COUNT(1),
		       COALESCE(SUM(CASE WHEN archived = 0 THEN 1 ELSE 0 END), 0),
		       COALESCE(SUM(CASE WHEN archived = 1 THEN 1 ELSE 0 END), 0)
		FROM memos
	`).Scan(&stats.Total, &stats.Active, &stats.Archived)
	if err != nil {
		return Stats{}, fmt.Errorf("read memo stats: %w", err)
	}
	return stats, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanMemo(row scanner) (Memo, error) {
	var (
		memo       Memo
		archived   int
		archivedAt sql.NullString
	)
	if err := row.Scan(
		&memo.ID,
		&memo.Title,
		&memo.ContentHTML,
		&archived,
		&archivedAt,
		&memo.CreatedAt,
		&memo.UpdatedAt,
	); err != nil {
		return Memo{}, err
	}
	memo.Archived = archived == 1
	if archivedAt.Valid {
		memo.ArchivedAt = &archivedAt.String
	}
	memo.Excerpt = excerptFromHTML(memo.ContentHTML)
	return memo, nil
}

func normalizeInput(input Input) (Input, error) {
	title := strings.TrimSpace(input.Title)
	if title == "" {
		return Input{}, &ValidationError{Message: "备忘录标题不能为空"}
	}
	if utf8.RuneCountInString(title) > maxTitleRunes {
		return Input{}, &ValidationError{Message: "备忘录标题不能超过 200 个字"}
	}
	content, err := sanitizeHTML(input.ContentHTML)
	if err != nil {
		return Input{}, err
	}
	return Input{Title: title, ContentHTML: content}, nil
}

func boolValue(value bool) int {
	if value {
		return 1
	}
	return 0
}

var allowedTags = map[string]bool{
	"a": true, "b": true, "blockquote": true, "br": true, "div": true,
	"em": true, "h2": true, "h3": true, "i": true, "li": true,
	"ol": true, "p": true, "s": true, "strong": true, "u": true, "ul": true,
}

var removedTags = map[string]bool{
	"embed": true, "form": true, "iframe": true, "input": true, "link": true,
	"meta": true, "object": true, "script": true, "style": true, "svg": true,
	"textarea": true, "video": true,
}

func sanitizeHTML(value string) (string, error) {
	if len([]byte(value)) > maxContentBytes {
		return "", &ValidationError{Message: "备忘录内容不能超过 300 KB"}
	}
	value = strings.TrimSpace(value)
	if value == "" {
		return "", nil
	}

	nodes, err := nethtml.ParseFragment(strings.NewReader(value), nil)
	if err != nil {
		return "", &ValidationError{Message: "备忘录内容格式不正确"}
	}
	var output bytes.Buffer
	for _, node := range nodes {
		renderSanitized(&output, node)
	}
	content := strings.TrimSpace(output.String())
	if len([]byte(content)) > maxContentBytes {
		return "", &ValidationError{Message: "备忘录内容不能超过 300 KB"}
	}
	return content, nil
}

func renderSanitized(output *bytes.Buffer, node *nethtml.Node) {
	switch node.Type {
	case nethtml.TextNode:
		output.WriteString(stdhtml.EscapeString(node.Data))
	case nethtml.CommentNode:
		return
	case nethtml.ElementNode:
		tag := strings.ToLower(node.Data)
		if removedTags[tag] {
			return
		}
		if !allowedTags[tag] {
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				renderSanitized(output, child)
			}
			return
		}
		output.WriteByte('<')
		output.WriteString(tag)
		if tag == "a" {
			for _, attr := range node.Attr {
				if strings.ToLower(attr.Key) == "href" && safeHref(attr.Val) {
					output.WriteString(` href="`)
					output.WriteString(stdhtml.EscapeString(strings.TrimSpace(attr.Val)))
					output.WriteByte('"')
					break
				}
			}
		}
		output.WriteByte('>')
		if tag == "br" {
			return
		}
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderSanitized(output, child)
		}
		output.WriteString(`</`)
		output.WriteString(tag)
		output.WriteByte('>')
	default:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			renderSanitized(output, child)
		}
	}
}

func safeHref(value string) bool {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil {
		return false
	}
	switch strings.ToLower(parsed.Scheme) {
	case "http", "https", "mailto":
		return true
	case "":
		return strings.HasPrefix(strings.TrimSpace(value), "#")
	default:
		return false
	}
}

func excerptFromHTML(value string) string {
	if strings.TrimSpace(value) == "" {
		return ""
	}
	nodes, err := nethtml.ParseFragment(strings.NewReader(value), nil)
	if err != nil {
		return ""
	}
	var text strings.Builder
	var visit func(*nethtml.Node)
	visit = func(node *nethtml.Node) {
		switch node.Type {
		case nethtml.TextNode:
			text.WriteString(node.Data)
			text.WriteByte(' ')
		case nethtml.ElementNode:
			if strings.EqualFold(node.Data, "br") {
				text.WriteByte(' ')
				return
			}
			for child := node.FirstChild; child != nil; child = child.NextSibling {
				visit(child)
			}
		}
	}
	for _, node := range nodes {
		visit(node)
	}
	content := strings.Join(strings.Fields(text.String()), " ")
	runes := []rune(content)
	if len(runes) <= maxExcerptRunes {
		return content
	}
	return string(runes[:maxExcerptRunes-1]) + "…"
}
