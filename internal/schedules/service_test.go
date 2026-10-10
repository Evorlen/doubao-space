package schedules

import (
	"context"
	"errors"
	"testing"
	"time"

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

func TestScheduleRemindsUntilCompletedAndResetsFromCompletion(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	schedule, err := service.Create(ctx, ScheduleInput{
		Title:        "剪指甲",
		Note:         "豆包安静的时候进行",
		StartDate:    "2020-01-01",
		IntervalDays: 30,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if !schedule.IsDue || schedule.Status != "due" || schedule.OverdueDays <= 0 {
		t.Fatalf("overdue schedule = %#v", schedule)
	}

	today := localDate(time.Now())
	completed, err := service.Complete(ctx, schedule.ID)
	if err != nil {
		t.Fatalf("Complete() error = %v", err)
	}
	if completed.IsDue || completed.Status != "upcoming" {
		t.Fatalf("completed schedule = %#v, want upcoming", completed)
	}
	if completed.LastCompletedOn == nil || *completed.LastCompletedOn != today.Format(dateLayout) {
		t.Fatalf("last completed = %#v, want %s", completed.LastCompletedOn, today.Format(dateLayout))
	}
	expectedNext := today.AddDate(0, 0, 30).Format(dateLayout)
	if completed.NextDueDate != expectedNext || completed.DaysUntilDue != 30 {
		t.Fatalf("next due = %#v, want %s and 30 days", completed, expectedNext)
	}

	if _, err := service.Complete(ctx, schedule.ID); err == nil {
		t.Fatal("Complete(second time) error = nil")
	} else {
		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("Complete(second time) error = %T %v, want ConflictError", err, err)
		}
	}

	var completionCount int
	if err := service.db.QueryRow(`SELECT COUNT(1) FROM doubao_schedule_completions WHERE schedule_id = ?`, schedule.ID).Scan(&completionCount); err != nil {
		t.Fatalf("read completion history: %v", err)
	}
	if completionCount != 1 {
		t.Fatalf("completion history count = %d, want 1", completionCount)
	}
}

func TestScheduleCannotCompleteBeforeDue(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()
	future := localDate(time.Now()).AddDate(0, 0, 3).Format(dateLayout)

	schedule, err := service.Create(ctx, ScheduleInput{
		Title:        "驱虫",
		StartDate:    future,
		IntervalDays: 90,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if schedule.IsDue || schedule.DaysUntilDue != 3 {
		t.Fatalf("future schedule = %#v", schedule)
	}
	if _, err := service.Complete(ctx, schedule.ID); err == nil {
		t.Fatal("Complete(future) error = nil")
	} else {
		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("Complete(future) error = %T %v, want ConflictError", err, err)
		}
	}
}

func TestScheduleValidationAndDeleteCascade(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	for _, input := range []ScheduleInput{
		{Title: "没有日期", IntervalDays: 30},
		{Title: "间隔不正确", StartDate: "2026-01-01", IntervalDays: 0},
		{StartDate: "2026-01-01", IntervalDays: 30},
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

	schedule, err := service.Create(ctx, ScheduleInput{
		Title:        "清理猫砂盆",
		StartDate:    "2020-01-01",
		IntervalDays: 7,
	})
	if err != nil {
		t.Fatalf("Create(valid) error = %v", err)
	}
	if _, err := service.Complete(ctx, schedule.ID); err != nil {
		t.Fatalf("Complete(valid) error = %v", err)
	}
	if err := service.Delete(ctx, schedule.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if _, err := service.Get(ctx, schedule.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("Get(deleted) error = %v, want ErrNotFound", err)
	}
	var completionCount int
	if err := service.db.QueryRow(`SELECT COUNT(1) FROM doubao_schedule_completions WHERE schedule_id = ?`, schedule.ID).Scan(&completionCount); err != nil {
		t.Fatalf("read deleted completion history: %v", err)
	}
	if completionCount != 0 {
		t.Fatalf("deleted completion history count = %d, want 0", completionCount)
	}
}
