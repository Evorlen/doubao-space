package memos

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/Icingworld/doubao-space/internal/database"
)

func newTestService(t *testing.T) *Service {
	t.Helper()
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(db)
}

func TestMemoLifecycleAndArchive(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	memo, err := service.Create(ctx, Input{
		Title:       "旅行灵感",
		ContentHTML: `<h2>想去的地方</h2><p><strong>海边</strong>和一场慢慢的散步。</p><script>alert('xss')</script><a href="javascript:alert(1)">危险链接</a><a href="https://example.com">参考链接</a>`,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if memo.ID == 0 || memo.Archived || memo.Excerpt == "" {
		t.Fatalf("created memo = %#v", memo)
	}
	if strings.Contains(memo.ContentHTML, "script") || strings.Contains(memo.ContentHTML, "javascript:") {
		t.Fatalf("unsafe content was not removed: %q", memo.ContentHTML)
	}
	if !strings.Contains(memo.ContentHTML, `<strong>海边</strong>`) || !strings.Contains(memo.ContentHTML, `href="https://example.com"`) {
		t.Fatalf("sanitized content = %q", memo.ContentHTML)
	}

	response, err := service.List(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(response.Items) != 1 || response.Stats.Total != 1 || response.Stats.Active != 1 || response.Stats.Archived != 0 {
		t.Fatalf("list response = %#v", response)
	}

	memo, err = service.Update(ctx, memo.ID, Input{Title: "旅行灵感（更新）", ContentHTML: `<p>记得带上相机。</p>`})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if memo.Title != "旅行灵感（更新）" || memo.Excerpt != "记得带上相机。" {
		t.Fatalf("updated memo = %#v", memo)
	}

	memo, err = service.SetArchived(ctx, memo.ID, true)
	if err != nil {
		t.Fatalf("SetArchived(true) error = %v", err)
	}
	if !memo.Archived || memo.ArchivedAt == nil {
		t.Fatalf("archived memo = %#v", memo)
	}
	archived := true
	response, err = service.List(ctx, ListOptions{Archived: &archived})
	if err != nil {
		t.Fatalf("List(archived) error = %v", err)
	}
	if len(response.Items) != 1 || response.Stats.Archived != 1 {
		t.Fatalf("archived response = %#v", response)
	}

	memo, err = service.SetArchived(ctx, memo.ID, false)
	if err != nil {
		t.Fatalf("SetArchived(false) error = %v", err)
	}
	if memo.Archived || memo.ArchivedAt != nil {
		t.Fatalf("restored memo = %#v", memo)
	}

	if err := service.Delete(ctx, memo.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(ctx, memo.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
}

func TestMemoValidation(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	for _, input := range []Input{
		{ContentHTML: `<p>没有标题</p>`},
		{Title: strings.Repeat("太长", 101)},
		{Title: "内容太长", ContentHTML: strings.Repeat("x", maxContentBytes+1)},
	} {
		if _, err := service.Create(ctx, input); err == nil {
			t.Fatalf("Create(%#v) error = nil", input)
		} else {
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("Create(%#v) error = %T %v, want ValidationError", input, err, err)
			}
		}
	}
}
