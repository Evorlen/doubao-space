package entertainment

import (
	"context"
	"errors"
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

func intPointer(value int) *int {
	return &value
}

func TestEntertainmentItemLifecycle(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	item, err := service.Create(ctx, Input{
		MediaType:       MediaTypeNovel,
		Title:           "夜航西飞",
		Creator:         "柏瑞尔·马卡姆",
		ReleaseYear:     intPointer(1942),
		Source:          "纸书",
		Status:          StatusInProgress,
		ProgressCurrent: 12,
		ProgressTotal:   intPointer(48),
		Rating:          5,
		Review:          "文字很有画面感。",
		Favorite:        true,
		Tags:            []string{"旅行", "非虚构", "旅行"},
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if item.MediaTypeLabel != "小说" || item.StatusLabel != "在读" || item.ProgressPercent != 25 {
		t.Fatalf("created item = %#v", item)
	}
	if item.StartedOn == nil || item.CompletedOn != nil {
		t.Fatalf("created dates = started %v, completed %v", item.StartedOn, item.CompletedOn)
	}
	if len(item.Tags) != 2 || item.Tags[0] != "旅行" || item.Tags[1] != "非虚构" {
		t.Fatalf("created tags = %#v", item.Tags)
	}

	response, err := service.List(ctx, ListOptions{MediaType: MediaTypeNovel, Rating: 5, Favorite: true})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(response.Items) != 1 || response.Stats.Total != 1 || response.Stats.InProgress != 1 || response.Stats.Favorites != 1 {
		t.Fatalf("list response = %#v", response)
	}

	item, err = service.Update(ctx, item.ID, Input{
		MediaType:       MediaTypeNovel,
		Title:           item.Title,
		Creator:         item.Creator,
		Status:          StatusCompleted,
		ProgressCurrent: 12,
		ProgressTotal:   intPointer(48),
		Rating:          4,
		Tags:            []string{"完成"},
	})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	if item.StatusLabel != "已读" || item.ProgressCurrent != 48 || item.ProgressPercent != 100 || item.CompletedOn == nil {
		t.Fatalf("updated item = %#v", item)
	}

	if err := service.Delete(ctx, item.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(ctx, item.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
}

func TestEntertainmentValidation(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	for _, input := range []Input{
		{Title: "没有类型"},
		{MediaType: MediaTypeMovie},
		{MediaType: MediaTypeSeries, Title: "错误状态", Status: "unknown"},
		{MediaType: MediaTypeNovel, Title: "进度超出", ProgressCurrent: 10, ProgressTotal: intPointer(9)},
		{MediaType: MediaTypeMovie, Title: "评分超出", Rating: 6},
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
