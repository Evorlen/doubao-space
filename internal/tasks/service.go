package tasks

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const MaxDepth = 5

const (
	PriorityUrgent = "urgent"
	PriorityHigh   = "high"
	PriorityMedium = "medium"
	PriorityLow    = "low"
)

var (
	ErrNotFound = errors.New("task not found")
)

// InputError means the request cannot be accepted as a task payload.
type InputError struct {
	Message string
}

func (e *InputError) Error() string { return e.Message }

// ConflictError means the request would violate a task-tree rule.
type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string { return e.Message }

type LinkInput struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type CreateInput struct {
	ParentID    *int64      `json:"parent_id"`
	Title       string      `json:"title"`
	Description string      `json:"description"`
	DueAt       string      `json:"due_at"`
	Priority    string      `json:"priority"`
	Tags        []string    `json:"tags"`
	Links       []LinkInput `json:"links"`
}

type UpdateInput struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	DueAt       string      `json:"due_at"`
	Priority    string      `json:"priority"`
	Tags        []string    `json:"tags"`
	Links       []LinkInput `json:"links"`
}

type ListOptions struct {
	IncludeCompleted bool
	Search           string
	Status           string
	Priority         string
	Tag              string
	Sort             string
}

type Link struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	URL       string `json:"url"`
	CreatedAt string `json:"created_at"`
}

type Attachment struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	MimeType  string `json:"mime_type"`
	Size      int64  `json:"size"`
	CreatedAt string `json:"created_at"`
}

type Progress struct {
	Completed int    `json:"completed"`
	Total     int    `json:"total"`
	Label     string `json:"label"`
}

type Task struct {
	ID                int64        `json:"id"`
	ParentID          *int64       `json:"parent_id"`
	Depth             int          `json:"depth"`
	Title             string       `json:"title"`
	Description       string       `json:"description"`
	DueAt             *time.Time   `json:"due_at"`
	Priority          string       `json:"priority"`
	EffectivePriority string       `json:"effective_priority"`
	Completed         bool         `json:"completed"`
	CreatedAt         string       `json:"created_at"`
	UpdatedAt         string       `json:"updated_at"`
	ImportKey         string       `json:"-"`
	Path              []string     `json:"path"`
	Tags              []string     `json:"tags"`
	Links             []Link       `json:"links"`
	Attachments       []Attachment `json:"attachments"`
	Progress          Progress     `json:"progress"`
	Children          []*Task      `json:"children"`
}

type Tag struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
}

type Service struct {
	db        *sql.DB
	uploadDir string
}

func NewService(db *sql.DB, dataDir ...string) *Service {
	service := &Service{db: db}
	if len(dataDir) > 0 && strings.TrimSpace(dataDir[0]) != "" {
		service.uploadDir = filepath.Join(dataDir[0], "uploads")
	}
	return service
}

func (s *Service) List(ctx context.Context, options ListOptions) ([]*Task, error) {
	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateForest(roots); err != nil {
		return nil, err
	}

	if strings.TrimSpace(options.Search) != "" {
		var matches []*Task
		for _, task := range flatten(roots) {
			if matchesTask(task, options) {
				copy := *task
				copy.Children = []*Task{}
				matches = append(matches, &copy)
			}
		}
		sortTasks(matches, options.Sort)
		return matches, nil
	}

	filtered := filterTree(roots, options)
	sortTree(filtered, options.Sort)
	return filtered, nil
}

func (s *Service) Get(ctx context.Context, id int64) (*Task, error) {
	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateForest(roots); err != nil {
		return nil, err
	}
	task := findTask(roots, id)
	if task == nil {
		return nil, ErrNotFound
	}
	return task, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (*Task, error) {
	title, description, priority, dueAt, tags, links, err := normalizePayload(input.Title, input.Description, input.Priority, input.DueAt, input.Tags, input.Links)
	if err != nil {
		return nil, err
	}

	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateForest(roots); err != nil {
		return nil, err
	}

	candidate := &Task{
		ParentID:    input.ParentID,
		Title:       title,
		Description: description,
		DueAt:       dueAt,
		Priority:    priority,
		Tags:        tags,
		Links:       linksToTasks(links),
		Children:    []*Task{},
	}
	if input.ParentID != nil {
		parent := findTask(roots, *input.ParentID)
		if parent == nil {
			return nil, ErrNotFound
		}
		if parent.Completed {
			return nil, &ConflictError{Message: "已完成任务不能添加子任务。"}
		}
		if parent.Depth >= MaxDepth {
			return nil, &ConflictError{Message: fmt.Sprintf("任务树最多支持 %d 层，当前父任务已经位于第 %d 层。", MaxDepth, parent.Depth)}
		}
		parent.Children = append(parent.Children, candidate)
		if err := validateForest(roots); err != nil {
			parent.Children = parent.Children[:len(parent.Children)-1]
			return nil, err
		}
		parent.Children = parent.Children[:len(parent.Children)-1]
	} else if err := validateForest([]*Task{candidate}); err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create task: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO tasks (parent_id, title, description, due_at, priority)
		VALUES (?, ?, ?, ?, ?)
	`, nullableID(input.ParentID), title, description, nullableTime(dueAt), priority)
	if err != nil {
		return nil, fmt.Errorf("insert task: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read task id: %w", err)
	}
	if err := saveAssociations(ctx, tx, id, tags, links); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit task: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (*Task, error) {
	title, description, priority, dueAt, tags, links, err := normalizePayload(input.Title, input.Description, input.Priority, input.DueAt, input.Tags, input.Links)
	if err != nil {
		return nil, err
	}

	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	if err := validateForest(roots); err != nil {
		return nil, err
	}
	task := findTask(roots, id)
	if task == nil {
		return nil, ErrNotFound
	}
	if task.Completed {
		return nil, &ConflictError{Message: "已完成任务不可编辑。"}
	}

	old := *task
	oldTags := append([]string(nil), task.Tags...)
	task.Title = title
	task.Description = description
	task.Priority = priority
	task.DueAt = dueAt
	task.Tags = tags
	if err := validateForest(roots); err != nil {
		*task = old
		task.Tags = oldTags
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update task: %w", err)
	}
	defer tx.Rollback()
	if _, err := tx.ExecContext(ctx, `
		UPDATE tasks
		SET title = ?, description = ?, due_at = ?, priority = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, title, description, nullableTime(dueAt), priority, id); err != nil {
		return nil, fmt.Errorf("update task: %w", err)
	}
	if err := replaceAssociations(ctx, tx, id, tags, links); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit task update: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Complete(ctx context.Context, id int64) (*Task, error) {
	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	task := findTask(roots, id)
	if task == nil {
		return nil, ErrNotFound
	}
	if task.Completed {
		return nil, &ConflictError{Message: "已完成任务不可恢复。"}
	}
	var pending []string
	for _, descendant := range flatten(task.Children) {
		if !descendant.Completed {
			pending = append(pending, descendant.Title)
		}
	}
	if len(pending) > 0 {
		if len(pending) > 4 {
			pending = append(pending[:4], "等")
		}
		return nil, &ConflictError{Message: fmt.Sprintf("请先完成以下任务：%s。", strings.Join(pending, "、"))}
	}
	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET completed = 1, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, id); err != nil {
		return nil, fmt.Errorf("complete task: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Move(ctx context.Context, id int64, parentID *int64) (*Task, error) {
	roots, err := s.loadForest(ctx)
	if err != nil {
		return nil, err
	}
	task := findTask(roots, id)
	if task == nil {
		return nil, ErrNotFound
	}
	if task.Completed {
		return nil, &ConflictError{Message: "已完成任务不可移动。"}
	}
	if parentID != nil && *parentID == id {
		return nil, &ConflictError{Message: "任务不能移动到自己下面。"}
	}
	if parentID != nil {
		newParent := findTask(roots, *parentID)
		if newParent == nil {
			return nil, ErrNotFound
		}
		if newParent.Completed {
			return nil, &ConflictError{Message: "已完成任务不能作为新的父任务。"}
		}
		if findTask(task.Children, *parentID) != nil {
			return nil, &ConflictError{Message: "任务不能移动到自己的后代下面。"}
		}
		if task.ParentID != nil && *task.ParentID == *parentID {
			return task, nil
		}
	}

	oldParentID := copyID(task.ParentID)
	oldParent := findTask(roots, valueOrZero(oldParentID))
	oldIndex := indexOfTask(roots, id)
	if oldParent != nil {
		oldIndex = indexOfTask(oldParent.Children, id)
	}
	if detachTask(&roots, id) == nil {
		return nil, ErrNotFound
	}
	task.ParentID = parentID
	if parentID == nil {
		roots = append(roots, task)
	} else {
		newParent := findTask(roots, *parentID)
		if newParent == nil {
			return nil, ErrNotFound
		}
		newParent.Children = append(newParent.Children, task)
	}
	if err := validateForest(roots); err != nil {
		// Put the in-memory tree back so the service can return safely after a rejected move.
		detachTask(&roots, id)
		task.ParentID = oldParentID
		if oldParent != nil {
			insertAt(&oldParent.Children, oldIndex, task)
		} else {
			insertAt(&roots, oldIndex, task)
		}
		return nil, err
	}

	if _, err := s.db.ExecContext(ctx, `UPDATE tasks SET parent_id = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?`, nullableID(parentID), id); err != nil {
		return nil, fmt.Errorf("move task: %w", err)
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	files, err := s.attachmentFilesForSubtree(ctx, id)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM tasks WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete task: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted task count: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	for _, file := range files {
		if s.uploadDir != "" {
			_ = os.Remove(filepath.Join(s.uploadDir, file))
		}
	}
	return nil
}

const maxAttachmentSize = 25 * 1024 * 1024

func (s *Service) AddAttachment(ctx context.Context, taskID int64, name, mimeType string, reader io.Reader) (Attachment, error) {
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return Attachment{}, err
	}
	if task.Completed {
		return Attachment{}, &ConflictError{Message: "已完成任务不可添加附件。"}
	}
	if s.uploadDir == "" {
		return Attachment{}, &InputError{Message: "附件目录未配置。"}
	}
	name = strings.TrimSpace(filepath.Base(name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return Attachment{}, &InputError{Message: "附件名称不能为空。"}
	}
	if len([]rune(name)) > 255 {
		return Attachment{}, &InputError{Message: "附件名称不能超过 255 个字符。"}
	}
	if strings.TrimSpace(mimeType) == "" {
		mimeType = "application/octet-stream"
	}
	if err := os.MkdirAll(s.uploadDir, 0o700); err != nil {
		return Attachment{}, fmt.Errorf("prepare attachment directory: %w", err)
	}
	var randomID [8]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return Attachment{}, fmt.Errorf("create attachment id: %w", err)
	}
	storedName := hex.EncodeToString(randomID[:]) + "_" + name
	temporaryPath := filepath.Join(s.uploadDir, "."+storedName+".tmp")
	finalPath := filepath.Join(s.uploadDir, storedName)
	file, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Attachment{}, fmt.Errorf("create attachment file: %w", err)
	}
	written, copyErr := io.Copy(file, io.LimitReader(reader, maxAttachmentSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temporaryPath)
		return Attachment{}, fmt.Errorf("write attachment: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(temporaryPath)
		return Attachment{}, fmt.Errorf("close attachment: %w", closeErr)
	}
	if written > maxAttachmentSize {
		_ = os.Remove(temporaryPath)
		return Attachment{}, &InputError{Message: "单个附件不能超过 25 MB。"}
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		_ = os.Remove(temporaryPath)
		return Attachment{}, fmt.Errorf("store attachment: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO task_attachments (task_id, name, stored_name, mime_type, size)
		VALUES (?, ?, ?, ?, ?)
	`, taskID, name, storedName, mimeType, written)
	if err != nil {
		_ = os.Remove(finalPath)
		return Attachment{}, fmt.Errorf("save attachment: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		_ = os.Remove(finalPath)
		return Attachment{}, fmt.Errorf("read attachment id: %w", err)
	}
	attachment, _, err := s.GetAttachment(ctx, taskID, id)
	if err != nil {
		_ = os.Remove(finalPath)
		return Attachment{}, err
	}
	return attachment, nil
}

func (s *Service) GetAttachment(ctx context.Context, taskID, attachmentID int64) (Attachment, string, error) {
	var attachment Attachment
	var storedName string
	err := s.db.QueryRowContext(ctx, `
		SELECT id, name, stored_name, mime_type, size, created_at
		FROM task_attachments
		WHERE task_id = ? AND id = ?
	`, taskID, attachmentID).Scan(&attachment.ID, &attachment.Name, &storedName, &attachment.MimeType, &attachment.Size, &attachment.CreatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return Attachment{}, "", ErrNotFound
	}
	if err != nil {
		return Attachment{}, "", fmt.Errorf("get attachment: %w", err)
	}
	if s.uploadDir == "" {
		return Attachment{}, "", &InputError{Message: "附件目录未配置。"}
	}
	return attachment, filepath.Join(s.uploadDir, storedName), nil
}

func (s *Service) DeleteAttachment(ctx context.Context, taskID, attachmentID int64) error {
	task, err := s.Get(ctx, taskID)
	if err != nil {
		return err
	}
	if task.Completed {
		return &ConflictError{Message: "已完成任务不可删除附件。"}
	}
	_, path, err := s.GetAttachment(ctx, taskID, attachmentID)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM task_attachments WHERE task_id = ? AND id = ?`, taskID, attachmentID)
	if err != nil {
		return fmt.Errorf("delete attachment: %w", err)
	}
	count, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted attachment count: %w", err)
	}
	if count == 0 {
		return ErrNotFound
	}
	_ = os.Remove(path)
	return nil
}

func (s *Service) attachmentFilesForSubtree(ctx context.Context, id int64) ([]string, error) {
	rows, err := s.db.QueryContext(ctx, `
		WITH RECURSIVE descendants(id) AS (
			SELECT id FROM tasks WHERE id = ?
			UNION ALL
			SELECT tasks.id FROM tasks JOIN descendants ON tasks.parent_id = descendants.id
		)
		SELECT task_attachments.stored_name
		FROM task_attachments JOIN descendants ON descendants.id = task_attachments.task_id
	`, id)
	if err != nil {
		return nil, fmt.Errorf("list subtree attachments: %w", err)
	}
	defer rows.Close()
	var files []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, fmt.Errorf("scan subtree attachment: %w", err)
		}
		files = append(files, name)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read subtree attachments: %w", err)
	}
	return files, nil
}

func (s *Service) ListTags(ctx context.Context) ([]Tag, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT id, name, created_at FROM tags ORDER BY name COLLATE NOCASE`)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()
	var result []Tag
	for rows.Next() {
		var tag Tag
		if err := rows.Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		result = append(result, tag)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read tags: %w", err)
	}
	return result, nil
}

func (s *Service) CreateTag(ctx context.Context, name string) (Tag, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Tag{}, &InputError{Message: "标签名称不能为空。"}
	}
	if len([]rune(name)) > 50 {
		return Tag{}, &InputError{Message: "标签名称不能超过 50 个字符。"}
	}
	if _, err := s.db.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, name); err != nil {
		return Tag{}, fmt.Errorf("create tag: %w", err)
	}
	var tag Tag
	if err := s.db.QueryRowContext(ctx, `SELECT id, name, created_at FROM tags WHERE name = ?`, name).Scan(&tag.ID, &tag.Name, &tag.CreatedAt); err != nil {
		return Tag{}, fmt.Errorf("read tag: %w", err)
	}
	return tag, nil
}

func (s *Service) loadForest(ctx context.Context) ([]*Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, parent_id, title, description, due_at, priority, completed, created_at, updated_at, import_key
		FROM tasks
		ORDER BY created_at ASC, id ASC
	`)
	if err != nil {
		return nil, fmt.Errorf("list tasks: %w", err)
	}
	var all []*Task
	for rows.Next() {
		task, err := scanTask(rows)
		if err != nil {
			rows.Close()
			return nil, err
		}
		all = append(all, task)
	}
	if err := rows.Err(); err != nil {
		rows.Close()
		return nil, fmt.Errorf("read tasks: %w", err)
	}
	rows.Close()

	nodes := make(map[int64]*Task, len(all))
	for _, task := range all {
		task.Tags = []string{}
		task.Links = []Link{}
		task.Attachments = []Attachment{}
		task.Children = []*Task{}
		nodes[task.ID] = task
	}
	roots := make([]*Task, 0)
	for _, task := range all {
		if task.ParentID == nil {
			roots = append(roots, task)
			continue
		}
		parent := nodes[*task.ParentID]
		if parent == nil {
			return nil, fmt.Errorf("task %d references missing parent %d", task.ID, *task.ParentID)
		}
		parent.Children = append(parent.Children, task)
	}

	tagRows, err := s.db.QueryContext(ctx, `
		SELECT task_tags.task_id, tags.name
		FROM task_tags JOIN tags ON tags.id = task_tags.tag_id
		ORDER BY task_tags.task_id, tags.name COLLATE NOCASE
	`)
	if err != nil {
		return nil, fmt.Errorf("list task tags: %w", err)
	}
	for tagRows.Next() {
		var taskID int64
		var name string
		if err := tagRows.Scan(&taskID, &name); err != nil {
			tagRows.Close()
			return nil, fmt.Errorf("scan task tag: %w", err)
		}
		if task := nodes[taskID]; task != nil {
			task.Tags = append(task.Tags, name)
		}
	}
	if err := tagRows.Err(); err != nil {
		tagRows.Close()
		return nil, fmt.Errorf("read task tags: %w", err)
	}
	tagRows.Close()

	linkRows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, title, url, created_at
		FROM task_links
		ORDER BY task_id, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list task links: %w", err)
	}
	for linkRows.Next() {
		var link Link
		var taskID int64
		if err := linkRows.Scan(&link.ID, &taskID, &link.Title, &link.URL, &link.CreatedAt); err != nil {
			linkRows.Close()
			return nil, fmt.Errorf("scan task link: %w", err)
		}
		if task := nodes[taskID]; task != nil {
			task.Links = append(task.Links, link)
		}
	}
	if err := linkRows.Err(); err != nil {
		linkRows.Close()
		return nil, fmt.Errorf("read task links: %w", err)
	}
	linkRows.Close()

	attachmentRows, err := s.db.QueryContext(ctx, `
		SELECT id, task_id, name, mime_type, size, created_at
		FROM task_attachments
		ORDER BY task_id, id
	`)
	if err != nil {
		return nil, fmt.Errorf("list task attachments: %w", err)
	}
	for attachmentRows.Next() {
		var attachment Attachment
		var taskID int64
		if err := attachmentRows.Scan(&attachment.ID, &taskID, &attachment.Name, &attachment.MimeType, &attachment.Size, &attachment.CreatedAt); err != nil {
			attachmentRows.Close()
			return nil, fmt.Errorf("scan task attachment: %w", err)
		}
		if task := nodes[taskID]; task != nil {
			task.Attachments = append(task.Attachments, attachment)
		}
	}
	if err := attachmentRows.Err(); err != nil {
		attachmentRows.Close()
		return nil, fmt.Errorf("read task attachments: %w", err)
	}
	attachmentRows.Close()

	for _, root := range roots {
		calculateMetrics(root, 1)
		setPaths(root, nil)
	}
	return roots, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanTask(row scanner) (*Task, error) {
	var (
		task                                   Task
		parentID                               sql.NullInt64
		dueAt, createdAt, updatedAt, importKey sql.NullString
		completed                              int
	)
	if err := row.Scan(&task.ID, &parentID, &task.Title, &task.Description, &dueAt, &task.Priority, &completed, &createdAt, &updatedAt, &importKey); err != nil {
		return nil, fmt.Errorf("scan task: %w", err)
	}
	if parentID.Valid {
		value := parentID.Int64
		task.ParentID = &value
	}
	if dueAt.Valid && strings.TrimSpace(dueAt.String) != "" {
		value, err := parseStoredTime(dueAt.String)
		if err != nil {
			return nil, fmt.Errorf("parse task %d due time: %w", task.ID, err)
		}
		task.DueAt = &value
	}
	task.Completed = completed != 0
	task.CreatedAt = createdAt.String
	task.UpdatedAt = updatedAt.String
	if importKey.Valid {
		task.ImportKey = importKey.String
	}
	return &task, nil
}

func calculateMetrics(task *Task, depth int) (completed, total int) {
	task.Depth = depth
	maxPriority := priorityRank(task.Priority)
	descendantCompleted := 0
	descendantTotal := 0
	if task.Completed {
		completed = 1
	}
	total = 1
	for _, child := range task.Children {
		childCompleted, childTotal := calculateMetrics(child, depth+1)
		completed += childCompleted
		total += childTotal
		descendantCompleted += childCompleted
		descendantTotal += childTotal
		if !child.Completed && priorityRank(child.EffectivePriority) > maxPriority {
			maxPriority = priorityRank(child.EffectivePriority)
		}
	}
	task.EffectivePriority = priorityName(maxPriority)
	task.Progress = Progress{Completed: descendantCompleted, Total: descendantTotal}
	if descendantTotal > 0 {
		task.Progress.Label = fmt.Sprintf("%d / %d", descendantCompleted, descendantTotal)
	}
	return completed, total
}

func setPaths(task *Task, parentPath []string) {
	task.Path = append(append([]string(nil), parentPath...), task.Title)
	for _, child := range task.Children {
		setPaths(child, task.Path)
	}
}

func validateForest(roots []*Task) error {
	seen := make(map[*Task]bool)
	var walk func(*Task, *Task, int) error
	walk = func(task, parent *Task, depth int) error {
		if seen[task] {
			return &ConflictError{Message: "任务树存在循环关系。"}
		}
		seen[task] = true
		if depth > MaxDepth {
			return &ConflictError{Message: fmt.Sprintf("任务树最多支持 %d 层。", MaxDepth)}
		}
		if strings.TrimSpace(task.Title) == "" {
			return &InputError{Message: "任务标题不能为空。"}
		}
		if !validPriority(task.Priority) {
			return &InputError{Message: "任务优先级不合法。"}
		}
		if parent != nil {
			if parent.DueAt == nil && task.DueAt != nil {
				return &ConflictError{Message: fmt.Sprintf("无法设置截止时间：父任务“%s”没有截止时间。", parent.Title)}
			}
			if parent.DueAt != nil && task.DueAt == nil {
				return &ConflictError{Message: fmt.Sprintf("无法清除截止时间：父任务“%s”有截止时间。", parent.Title)}
			}
			if parent.DueAt != nil && task.DueAt != nil && task.DueAt.After(*parent.DueAt) {
				return &ConflictError{Message: fmt.Sprintf("无法保存：子任务“%s”的截止时间晚于父任务“%s”。", task.Title, parent.Title)}
			}
		}
		for _, child := range task.Children {
			if err := walk(child, task, depth+1); err != nil {
				return err
			}
		}
		return nil
	}
	for _, root := range roots {
		if err := walk(root, nil, 1); err != nil {
			return err
		}
	}
	return nil
}

func normalizePayload(title, description, priority, dueAt string, tags []string, links []LinkInput) (string, string, string, *time.Time, []string, []LinkInput, error) {
	title = strings.TrimSpace(title)
	if title == "" {
		return "", "", "", nil, nil, nil, &InputError{Message: "任务标题不能为空。"}
	}
	if len([]rune(title)) > 500 {
		return "", "", "", nil, nil, nil, &InputError{Message: "任务标题不能超过 500 个字符。"}
	}
	if priority == "" {
		priority = PriorityMedium
	}
	if !validPriority(priority) {
		return "", "", "", nil, nil, nil, &InputError{Message: "任务优先级不合法。"}
	}
	parsedDue, err := parseDue(dueAt)
	if err != nil {
		return "", "", "", nil, nil, nil, err
	}
	normalizedTags := normalizeTags(tags)
	normalizedLinks := make([]LinkInput, 0, len(links))
	for _, link := range links {
		link.Title = strings.TrimSpace(link.Title)
		link.URL = strings.TrimSpace(link.URL)
		if link.URL == "" {
			continue
		}
		parsed, err := url.Parse(link.URL)
		if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
			return "", "", "", nil, nil, nil, &InputError{Message: fmt.Sprintf("链接地址不合法：%s。", link.URL)}
		}
		normalizedLinks = append(normalizedLinks, link)
	}
	return title, strings.TrimSpace(description), priority, parsedDue, normalizedTags, normalizedLinks, nil
}

func parseDue(value string) (*time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, nil
	}
	layouts := []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05",
		"2006-01-02T15:04:05",
		"2006-01-02 15:04",
		"2006-01-02T15:04",
		"2006-01-02",
	}
	for _, layout := range layouts {
		var parsed time.Time
		var err error
		if strings.Contains(layout, "Z07") {
			parsed, err = time.Parse(layout, value)
		} else {
			parsed, err = time.ParseInLocation(layout, value, time.Local)
		}
		if err == nil {
			return &parsed, nil
		}
	}
	return nil, &InputError{Message: "截止时间格式不合法，请使用 YYYY-MM-DD HH:mm。"}
}

func parseStoredTime(value string) (time.Time, error) {
	for _, layout := range []string{time.RFC3339Nano, time.RFC3339, "2006-01-02 15:04:05", "2006-01-02T15:04:05"} {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("unsupported time %q", value)
}

func saveAssociations(ctx context.Context, tx *sql.Tx, taskID int64, tags []string, links []LinkInput) error {
	for _, tag := range tags {
		if _, err := tx.ExecContext(ctx, `INSERT INTO tags (name) VALUES (?) ON CONFLICT(name) DO NOTHING`, tag); err != nil {
			return fmt.Errorf("save task tag: %w", err)
		}
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO task_tags (task_id, tag_id)
			SELECT ?, id FROM tags WHERE name = ?
		`, taskID, tag); err != nil {
			return fmt.Errorf("link task tag: %w", err)
		}
	}
	for _, link := range links {
		if _, err := tx.ExecContext(ctx, `INSERT INTO task_links (task_id, title, url) VALUES (?, ?, ?)`, taskID, link.Title, link.URL); err != nil {
			return fmt.Errorf("save task link: %w", err)
		}
	}
	return nil
}

func replaceAssociations(ctx context.Context, tx *sql.Tx, taskID int64, tags []string, links []LinkInput) error {
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_tags WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("clear task tags: %w", err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM task_links WHERE task_id = ?`, taskID); err != nil {
		return fmt.Errorf("clear task links: %w", err)
	}
	return saveAssociations(ctx, tx, taskID, tags, links)
}

func filterTree(nodes []*Task, options ListOptions) []*Task {
	result := make([]*Task, 0, len(nodes))
	for _, task := range nodes {
		if !options.IncludeCompleted && task.Completed {
			continue
		}
		children := filterTree(task.Children, options)
		if matchesTask(task, options) || len(children) > 0 {
			copy := *task
			copy.Children = children
			result = append(result, &copy)
		}
	}
	return result
}

func matchesTask(task *Task, options ListOptions) bool {
	if !options.IncludeCompleted && task.Completed {
		return false
	}
	if options.Status == "completed" && !task.Completed {
		return false
	}
	if options.Status == "incomplete" && task.Completed {
		return false
	}
	if options.Priority != "" && task.EffectivePriority != options.Priority && task.Priority != options.Priority {
		return false
	}
	if options.Tag != "" {
		found := false
		for _, tag := range task.Tags {
			if strings.EqualFold(tag, options.Tag) {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	if options.Search != "" && !strings.Contains(strings.ToLower(task.Title), strings.ToLower(strings.TrimSpace(options.Search))) {
		return false
	}
	return true
}

func sortTree(nodes []*Task, key string) {
	sortTasks(nodes, key)
	for _, task := range nodes {
		sortTree(task.Children, key)
	}
}

func sortTasks(tasks []*Task, key string) {
	sort.SliceStable(tasks, func(i, j int) bool {
		left, right := tasks[i], tasks[j]
		switch key {
		case "priority":
			if priorityRank(left.EffectivePriority) != priorityRank(right.EffectivePriority) {
				return priorityRank(left.EffectivePriority) > priorityRank(right.EffectivePriority)
			}
		case "created":
			if left.CreatedAt != right.CreatedAt {
				return left.CreatedAt > right.CreatedAt
			}
		default:
			leftDue, rightDue := dueSortValue(left.DueAt), dueSortValue(right.DueAt)
			if !leftDue.Equal(rightDue) {
				if left.DueAt == nil {
					return false
				}
				if right.DueAt == nil {
					return true
				}
				return leftDue.Before(rightDue)
			}
		}
		if priorityRank(left.EffectivePriority) != priorityRank(right.EffectivePriority) {
			return priorityRank(left.EffectivePriority) > priorityRank(right.EffectivePriority)
		}
		return left.ID < right.ID
	})
}

func dueSortValue(value *time.Time) time.Time {
	if value == nil {
		return time.Time{}
	}
	return *value
}

func flatten(nodes []*Task) []*Task {
	var result []*Task
	for _, node := range nodes {
		result = append(result, node)
		result = append(result, flatten(node.Children)...)
	}
	return result
}

func findTask(nodes []*Task, id int64) *Task {
	for _, task := range nodes {
		if task.ID == id {
			return task
		}
		if found := findTask(task.Children, id); found != nil {
			return found
		}
	}
	return nil
}

func detachTask(nodes *[]*Task, id int64) *Task {
	items := *nodes
	for index, task := range items {
		if task.ID == id {
			*nodes = append(items[:index], items[index+1:]...)
			return task
		}
		if found := detachTask(&task.Children, id); found != nil {
			return found
		}
	}
	return nil
}

func indexOfTask(nodes []*Task, id int64) int {
	for index, task := range nodes {
		if task.ID == id {
			return index
		}
	}
	return len(nodes)
}

func insertAt(nodes *[]*Task, index int, task *Task) {
	items := *nodes
	if index < 0 || index > len(items) {
		index = len(items)
	}
	items = append(items, nil)
	copy(items[index+1:], items[index:])
	items[index] = task
	*nodes = items
}

func linksToTasks(links []LinkInput) []Link {
	result := make([]Link, 0, len(links))
	for _, link := range links {
		result = append(result, Link{Title: link.Title, URL: link.URL})
	}
	return result
}

func normalizeTags(tags []string) []string {
	seen := make(map[string]struct{}, len(tags))
	result := make([]string, 0, len(tags))
	for _, tag := range tags {
		tag = strings.TrimSpace(tag)
		if tag == "" || len([]rune(tag)) > 50 {
			continue
		}
		key := strings.ToLower(tag)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		result = append(result, tag)
	}
	return result
}

func validPriority(value string) bool {
	return value == PriorityUrgent || value == PriorityHigh || value == PriorityMedium || value == PriorityLow
}

func priorityRank(value string) int {
	switch value {
	case PriorityUrgent:
		return 4
	case PriorityHigh:
		return 3
	case PriorityMedium:
		return 2
	case PriorityLow:
		return 1
	default:
		return 0
	}
}

func priorityName(rank int) string {
	switch rank {
	case 4:
		return PriorityUrgent
	case 3:
		return PriorityHigh
	case 2:
		return PriorityMedium
	default:
		return PriorityLow
	}
}

func nullableID(value *int64) any {
	if value == nil {
		return nil
	}
	return *value
}

func nullableTime(value *time.Time) any {
	if value == nil {
		return nil
	}
	return value.Format(time.RFC3339)
}

func copyID(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copy := *value
	return &copy
}

func valueOrZero(value *int64) int64 {
	if value == nil {
		return 0
	}
	return *value
}
