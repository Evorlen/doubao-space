package album

import (
	"bytes"
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Icingworld/doubao-space/internal/database"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	dataDir := t.TempDir()
	db, err := database.Open(dataDir)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(db, dataDir)
}

func TestAlbumLifecycle(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	item, err := service.Create(ctx, UploadInput{
		Name:     "豆包晒太阳.png",
		MimeType: "image/png",
		Note:     "今天在窗边晒了很久。",
		Reader:   bytes.NewBufferString("fake image content"),
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.MediaType != MediaTypePhoto || item.MediaTypeLabel != "照片" || item.Note == "" || item.MediaURL == "" {
		t.Fatalf("created item = %#v", item)
	}

	got, mediaPath, err := service.GetMedia(ctx, item.ID)
	if err != nil {
		t.Fatalf("GetMedia() error = %v", err)
	}
	if got.ID != item.ID || filepath.Ext(mediaPath) != "" {
		t.Fatalf("media = %#v, path = %q", got, mediaPath)
	}
	if _, err := os.Stat(mediaPath); err != nil {
		t.Fatalf("stored media stat error = %v", err)
	}

	updated, err := service.Update(ctx, item.ID, UpdateInput{Note: "更新后的备注"})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if updated.Note != "更新后的备注" {
		t.Fatalf("updated note = %q", updated.Note)
	}

	video, err := service.Create(ctx, UploadInput{
		Name:     "豆包玩耍.mp4",
		MimeType: "video/mp4",
		Reader:   bytes.NewBufferString("fake video content"),
	})
	if err != nil {
		t.Fatalf("Create(video) error = %v", err)
	}
	if video.MediaType != MediaTypeVideo || video.MediaTypeLabel != "视频" {
		t.Fatalf("created video = %#v", video)
	}

	list, err := service.List(ctx)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(list.Items) != 2 || list.Stats.Total != 2 || list.Stats.Photos != 1 || list.Stats.Videos != 1 {
		t.Fatalf("album = %#v", list)
	}

	if err := service.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := os.Stat(mediaPath); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("deleted media stat error = %v, want not exist", err)
	}
	if _, err := service.Get(ctx, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
}

func TestAlbumValidation(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	tests := []UploadInput{
		{Name: "note.txt", MimeType: "text/plain", Reader: strings.NewReader("not media")},
		{Name: "empty.png", MimeType: "image/png", Reader: strings.NewReader("")},
		{Name: "too-long-note.png", MimeType: "image/png", Note: strings.Repeat("字", MaxNoteLength+1), Reader: strings.NewReader("image")},
	}
	for _, input := range tests {
		if _, err := service.Create(ctx, input); err == nil {
			t.Fatalf("Create(%#v) error = nil", input)
		} else {
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Create(%#v) error = %T %v, want ValidationError", input, err, err)
			}
		}
	}

	if _, err := service.Create(ctx, UploadInput{
		Name:     "fallback.jpg",
		MimeType: "application/octet-stream",
		Reader:   strings.NewReader("image"),
	}); err != nil {
		t.Fatalf("Create() with extension fallback error = %v", err)
	}
}
