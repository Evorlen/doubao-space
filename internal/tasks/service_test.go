package tasks

import (
	"bytes"
	"context"
	"errors"
	"os"
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

func TestTaskAttachments(t *testing.T) {
	t.Helper()
	dataDir := t.TempDir()
	db, err := database.Open(dataDir)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()
	service := NewService(db, dataDir)
	task, err := service.Create(context.Background(), CreateInput{Title: "附件任务", Priority: PriorityMedium})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	attachment, err := service.AddAttachment(context.Background(), task.ID, "说明.txt", "text/plain", bytes.NewBufferString("hello"))
	if err != nil {
		t.Fatalf("AddAttachment() error = %v", err)
	}
	if attachment.Name != "说明.txt" || attachment.Size != 5 {
		t.Fatalf("attachment = %#v", attachment)
	}
	_, path, err := service.GetAttachment(context.Background(), task.ID, attachment.ID)
	if err != nil {
		t.Fatalf("GetAttachment() error = %v", err)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("attachment file stat error = %v", err)
	}
	if err := service.DeleteAttachment(context.Background(), task.ID, attachment.ID); err != nil {
		t.Fatalf("DeleteAttachment() error = %v", err)
	}
	if _, err := os.Stat(path); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("attachment file stat after delete = %v, want not exists", err)
	}
}

func TestCreateListAndCompleteTaskTree(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	root, err := service.Create(ctx, CreateInput{
		Title:    "准备一次旅行",
		DueAt:    "2026-10-20T18:00:00Z",
		Priority: PriorityLow,
		Tags:     []string{"生活", "生活"},
		Links:    []LinkInput{{Title: "参考行程", URL: "https://example.com/trip"}},
	})
	if err != nil {
		t.Fatalf("Create(root) error = %v", err)
	}
	if root.Depth != 1 || len(root.Tags) != 1 || len(root.Links) != 1 {
		t.Fatalf("root = %#v, want depth 1, one tag and one link", root)
	}

	child, err := service.Create(ctx, CreateInput{
		ParentID: &root.ID,
		Title:    "预订酒店",
		DueAt:    "2026-10-18T18:00:00Z",
		Priority: PriorityUrgent,
	})
	if err != nil {
		t.Fatalf("Create(child) error = %v", err)
	}
	if child.Depth != 2 {
		t.Fatalf("child depth = %d, want 2", child.Depth)
	}

	got, err := service.Get(ctx, root.ID)
	if err != nil {
		t.Fatalf("Get(root) error = %v", err)
	}
	if got.EffectivePriority != PriorityUrgent || got.Progress.Label != "0 / 1" {
		t.Fatalf("root metrics = %#v, want urgent and 0 / 1", got)
	}

	if _, err := service.Complete(ctx, root.ID); err == nil {
		t.Fatal("Complete(root) error = nil, want pending-descendant conflict")
	} else {
		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("Complete(root) error = %T %v, want ConflictError", err, err)
		}
	}
	if _, err := service.Complete(ctx, child.ID); err != nil {
		t.Fatalf("Complete(child) error = %v", err)
	}
	if _, err := service.Complete(ctx, root.ID); err != nil {
		t.Fatalf("Complete(root after child) error = %v", err)
	}

	visible, err := service.List(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("List(default) error = %v", err)
	}
	if len(visible) != 0 {
		t.Fatalf("List(default) length = %d, want completed tasks hidden", len(visible))
	}
	all, err := service.List(ctx, ListOptions{IncludeCompleted: true})
	if err != nil {
		t.Fatalf("List(include completed) error = %v", err)
	}
	if len(all) != 1 || !all[0].Completed || len(all[0].Children) != 1 || !all[0].Children[0].Completed {
		t.Fatalf("List(include completed) = %#v", all)
	}
}

func TestScheduleTaskLifecycle(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	task, err := service.Create(ctx, CreateInput{Title: "准备演讲", Priority: PriorityMedium})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if task.Scheduled {
		t.Fatal("new task is scheduled, want false")
	}

	scheduled, err := service.Schedule(ctx, task.ID)
	if err != nil {
		t.Fatalf("Schedule() error = %v", err)
	}
	if !scheduled.Scheduled {
		t.Fatal("scheduled task = false, want true")
	}

	listed, err := service.List(ctx, ListOptions{})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(listed) != 1 || !listed[0].Scheduled {
		t.Fatalf("listed scheduled task = %#v, want one scheduled task", listed)
	}

	unscheduled, err := service.Unschedule(ctx, task.ID)
	if err != nil {
		t.Fatalf("Unschedule() error = %v", err)
	}
	if unscheduled.Scheduled {
		t.Fatal("unscheduled task = true, want false")
	}

	if _, err := service.Schedule(ctx, task.ID); err != nil {
		t.Fatalf("Schedule() before completion error = %v", err)
	}
	completed, err := service.Complete(ctx, task.ID)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if !completed.Completed || completed.Scheduled {
		t.Fatalf("completed task = %#v, want completed and unscheduled", completed)
	}
	if _, err := service.Schedule(ctx, task.ID); err == nil {
		t.Fatal("Schedule(completed) error = nil, want conflict")
	}
}

func TestDeadlineRulesAndMoveRules(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	noDeadline, err := service.Create(ctx, CreateInput{Title: "无期限规划", Priority: PriorityMedium})
	if err != nil {
		t.Fatalf("Create(no deadline) error = %v", err)
	}
	if _, err := service.Create(ctx, CreateInput{
		ParentID: &noDeadline.ID,
		Title:    "错误的有期限子任务",
		DueAt:    "2026-10-20T18:00:00Z",
		Priority: PriorityMedium,
	}); err == nil {
		t.Fatal("Create(deadline child) error = nil, want deadline-mode conflict")
	}

	root, err := service.Create(ctx, CreateInput{
		Title:    "有期限规划",
		DueAt:    "2026-10-20T18:00:00Z",
		Priority: PriorityMedium,
	})
	if err != nil {
		t.Fatalf("Create(deadline root) error = %v", err)
	}
	child, err := service.Create(ctx, CreateInput{
		ParentID: &root.ID,
		Title:    "提前完成",
		DueAt:    "2026-10-19T18:00:00Z",
		Priority: PriorityMedium,
	})
	if err != nil {
		t.Fatalf("Create(deadline child) error = %v", err)
	}
	if _, err := service.Update(ctx, child.ID, UpdateInput{
		Title:    child.Title,
		DueAt:    "2026-10-21T18:00:00Z",
		Priority: PriorityMedium,
	}); err == nil {
		t.Fatal("Update(late child) error = nil, want deadline conflict")
	}
	if _, err := service.Move(ctx, root.ID, &child.ID); err == nil {
		t.Fatal("Move(root under child) error = nil, want cycle conflict")
	}

	if err := service.Delete(ctx, root.ID); err != nil {
		t.Fatalf("Delete(root) error = %v", err)
	}
	if _, err := service.Get(ctx, child.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted child) error = %v, want ErrNotFound", err)
	}
}

func TestMaximumTaskDepth(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()
	var parent *Task
	for depth := 1; depth <= MaxDepth; depth++ {
		var parentID *int64
		if parent != nil {
			parentID = &parent.ID
		}
		created, err := service.Create(ctx, CreateInput{
			ParentID: parentID,
			Title:    "层级任务",
			Priority: PriorityMedium,
		})
		if err != nil {
			t.Fatalf("Create(depth %d) error = %v", depth, err)
		}
		parent = created
	}
	if _, err := service.Create(ctx, CreateInput{
		ParentID: &parent.ID,
		Title:    "第六层",
		Priority: PriorityMedium,
	}); err == nil {
		t.Fatal("Create(depth 6) error = nil, want maximum-depth conflict")
	}
}
