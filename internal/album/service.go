package album

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"mime"
	"os"
	"path/filepath"
	"strings"
	"unicode/utf8"
)

const (
	MaxMediaSize  = 100 * 1024 * 1024
	MaxNoteLength = 500
)

var ErrNotFound = errors.New("doubao album item not found")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type Item struct {
	ID             int64  `json:"id"`
	MediaType      string `json:"media_type"`
	MediaTypeLabel string `json:"media_type_label"`
	Name           string `json:"name"`
	MimeType       string `json:"mime_type"`
	Size           int64  `json:"size"`
	Note           string `json:"note"`
	MediaURL       string `json:"media_url"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

type Stats struct {
	Total  int `json:"total"`
	Photos int `json:"photos"`
	Videos int `json:"videos"`
}

type Album struct {
	Items []Item `json:"items"`
	Stats Stats  `json:"stats"`
}

type UploadInput struct {
	Name     string
	MimeType string
	Note     string
	Reader   io.Reader
}

type UpdateInput struct {
	Note string `json:"note"`
}

type Service struct {
	db        *sql.DB
	uploadDir string
}

func NewService(db *sql.DB, dataDir string) *Service {
	service := &Service{db: db}
	if strings.TrimSpace(dataDir) != "" {
		service.uploadDir = filepath.Join(dataDir, "uploads", "doubao-album")
	}
	return service
}

func (s *Service) List(ctx context.Context) (*Album, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, media_type, name, mime_type, size, note, created_at, updated_at
		FROM doubao_album_items
		ORDER BY created_at DESC, id DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("list doubao album: %w", err)
	}
	defer rows.Close()

	items := make([]Item, 0)
	stats := Stats{}
	for rows.Next() {
		var item Item
		if err := rows.Scan(
			&item.ID,
			&item.MediaType,
			&item.Name,
			&item.MimeType,
			&item.Size,
			&item.Note,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan doubao album item: %w", err)
		}
		decorate(&item)
		items = append(items, item)
		stats.Total++
		if item.MediaType == MediaTypeVideo {
			stats.Videos++
		} else {
			stats.Photos++
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read doubao album: %w", err)
	}
	return &Album{Items: items, Stats: stats}, nil
}

func (s *Service) Get(ctx context.Context, id int64) (Item, error) {
	item, _, err := s.getStoredItem(ctx, id)
	return item, err
}

func (s *Service) Create(ctx context.Context, input UploadInput) (Item, error) {
	normalized, err := normalizeUpload(input)
	if err != nil {
		return Item{}, err
	}
	if s.uploadDir == "" {
		return Item{}, &ValidationError{Message: "相册目录未配置。"}
	}
	if err := os.MkdirAll(s.uploadDir, 0o700); err != nil {
		return Item{}, fmt.Errorf("prepare doubao album directory: %w", err)
	}

	var randomID [16]byte
	if _, err := rand.Read(randomID[:]); err != nil {
		return Item{}, fmt.Errorf("create doubao album id: %w", err)
	}
	storedName := hex.EncodeToString(randomID[:])
	temporaryPath := filepath.Join(s.uploadDir, "."+storedName+".tmp")
	finalPath := filepath.Join(s.uploadDir, storedName)
	file, err := os.OpenFile(temporaryPath, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return Item{}, fmt.Errorf("create doubao album file: %w", err)
	}

	written, copyErr := io.Copy(file, io.LimitReader(normalized.Reader, MaxMediaSize+1))
	closeErr := file.Close()
	if copyErr != nil {
		_ = os.Remove(temporaryPath)
		return Item{}, fmt.Errorf("write doubao album file: %w", copyErr)
	}
	if closeErr != nil {
		_ = os.Remove(temporaryPath)
		return Item{}, fmt.Errorf("close doubao album file: %w", closeErr)
	}
	if written == 0 {
		_ = os.Remove(temporaryPath)
		return Item{}, &ValidationError{Message: "不能上传空文件。"}
	}
	if written > MaxMediaSize {
		_ = os.Remove(temporaryPath)
		return Item{}, &ValidationError{Message: "单个照片或视频不能超过 100 MB。"}
	}
	if err := os.Rename(temporaryPath, finalPath); err != nil {
		_ = os.Remove(temporaryPath)
		return Item{}, fmt.Errorf("store doubao album file: %w", err)
	}

	result, err := s.db.ExecContext(ctx, `
		INSERT INTO doubao_album_items (media_type, name, stored_name, mime_type, size, note)
		VALUES (?, ?, ?, ?, ?, ?)
	`, normalized.MediaType, normalized.Name, storedName, normalized.MimeType, written, normalized.Note)
	if err != nil {
		_ = os.Remove(finalPath)
		return Item{}, fmt.Errorf("save doubao album item: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		_ = os.Remove(finalPath)
		return Item{}, fmt.Errorf("read doubao album item id: %w", err)
	}

	item, _, err := s.getStoredItem(ctx, id)
	if err != nil {
		_ = os.Remove(finalPath)
		return Item{}, err
	}
	return item, nil
}

func (s *Service) Update(ctx context.Context, id int64, input UpdateInput) (Item, error) {
	note, err := normalizeNote(input.Note)
	if err != nil {
		return Item{}, err
	}
	result, err := s.db.ExecContext(ctx, `
		UPDATE doubao_album_items
		SET note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, note, id)
	if err != nil {
		return Item{}, fmt.Errorf("update doubao album note: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return Item{}, fmt.Errorf("read updated doubao album item count: %w", err)
	}
	if rows == 0 {
		return Item{}, ErrNotFound
	}
	return s.Get(ctx, id)
}

func (s *Service) Delete(ctx context.Context, id int64) error {
	item, storedPath, err := s.getStoredItem(ctx, id)
	if err != nil {
		return err
	}
	result, err := s.db.ExecContext(ctx, `DELETE FROM doubao_album_items WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete doubao album item: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted doubao album item count: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	if storedPath != "" {
		if err := os.Remove(storedPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove doubao album file %q: %w", item.Name, err)
		}
	}
	return nil
}

func (s *Service) GetMedia(ctx context.Context, id int64) (Item, string, error) {
	return s.getStoredItem(ctx, id)
}

type normalizedUpload struct {
	Name      string
	MimeType  string
	MediaType string
	Note      string
	Reader    io.Reader
}

func normalizeUpload(input UploadInput) (normalizedUpload, error) {
	if input.Reader == nil {
		return normalizedUpload{}, &ValidationError{Message: "请选择要上传的文件。"}
	}
	name := strings.TrimSpace(filepath.Base(input.Name))
	if name == "" || name == "." || name == string(filepath.Separator) {
		return normalizedUpload{}, &ValidationError{Message: "文件名称不能为空。"}
	}
	if utf8.RuneCountInString(name) > 255 {
		return normalizedUpload{}, &ValidationError{Message: "文件名称不能超过 255 个字符。"}
	}
	note, err := normalizeNote(input.Note)
	if err != nil {
		return normalizedUpload{}, err
	}

	mimeType := strings.ToLower(strings.TrimSpace(strings.Split(input.MimeType, ";")[0]))
	if mimeType == "" || mimeType == "application/octet-stream" {
		mimeType = strings.ToLower(strings.TrimSpace(mime.TypeByExtension(filepath.Ext(name))))
	}
	if mimeType == "image/jpg" {
		mimeType = "image/jpeg"
	}
	mediaType, ok := supportedMimeTypes[mimeType]
	if !ok {
		return normalizedUpload{}, &ValidationError{Message: "目前支持 JPG、PNG、GIF、WebP 等照片，以及 MP4、WebM、MOV 等视频。"}
	}
	return normalizedUpload{
		Name:      name,
		MimeType:  mimeType,
		MediaType: mediaType,
		Note:      note,
		Reader:    input.Reader,
	}, nil
}

func normalizeNote(note string) (string, error) {
	note = strings.TrimSpace(note)
	if utf8.RuneCountInString(note) > MaxNoteLength {
		return "", &ValidationError{Message: fmt.Sprintf("备注不能超过 %d 个字。", MaxNoteLength)}
	}
	return note, nil
}

func (s *Service) getStoredItem(ctx context.Context, id int64) (Item, string, error) {
	var (
		item       Item
		storedName string
	)
	err := s.db.QueryRowContext(ctx, `
		SELECT id, media_type, name, stored_name, mime_type, size, note, created_at, updated_at
		FROM doubao_album_items
		WHERE id = ?
	`, id).Scan(
		&item.ID,
		&item.MediaType,
		&item.Name,
		&storedName,
		&item.MimeType,
		&item.Size,
		&item.Note,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if errors.Is(err, sql.ErrNoRows) {
		return Item{}, "", ErrNotFound
	}
	if err != nil {
		return Item{}, "", fmt.Errorf("get doubao album item: %w", err)
	}
	decorate(&item)
	var storedPath string
	if s.uploadDir != "" {
		storedPath = filepath.Join(s.uploadDir, storedName)
	}
	return item, storedPath, nil
}

func decorate(item *Item) {
	if item.MediaType == MediaTypeVideo {
		item.MediaTypeLabel = "视频"
	} else {
		item.MediaTypeLabel = "照片"
	}
	item.MediaURL = fmt.Sprintf("/api/doubao/album/%d/media", item.ID)
}

const (
	MediaTypePhoto = "photo"
	MediaTypeVideo = "video"
)

var supportedMimeTypes = map[string]string{
	"image/jpeg":       MediaTypePhoto,
	"image/png":        MediaTypePhoto,
	"image/gif":        MediaTypePhoto,
	"image/webp":       MediaTypePhoto,
	"image/avif":       MediaTypePhoto,
	"image/heic":       MediaTypePhoto,
	"image/heif":       MediaTypePhoto,
	"video/mp4":        MediaTypeVideo,
	"video/webm":       MediaTypeVideo,
	"video/quicktime":  MediaTypeVideo,
	"video/x-matroska": MediaTypeVideo,
	"video/ogg":        MediaTypeVideo,
	"video/3gpp":       MediaTypeVideo,
	"video/mpeg":       MediaTypeVideo,
}
