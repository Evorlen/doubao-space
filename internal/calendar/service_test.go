package calendar

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/Icingworld/doubao-space/internal/database"
	"github.com/Icingworld/doubao-space/internal/schedules"
	"github.com/Icingworld/doubao-space/internal/tasks"
)

func newTestService(t *testing.T) (*Service, *sql.DB) {
	t.Helper()
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	t.Cleanup(func() { db.Close() })
	return NewService(db), db
}

func TestListAggregatesTasksAndRecurringPlans(t *testing.T) {
	service, db := newTestService(t)
	ctx := context.Background()
	today := localDate(time.Now())
	end := today.AddDate(0, 0, 7)

	taskService := tasks.NewService(db)
	if _, err := taskService.Create(ctx, tasks.CreateInput{
		Title:    "提交报告",
		DueAt:    time.Date(today.Year(), today.Month(), today.Day(), 23, 0, 0, 0, time.Local).Format(time.RFC3339),
		Priority: tasks.PriorityHigh,
	}); err != nil {
		t.Fatalf("create task: %v", err)
	}

	if _, err := service.doubao.Create(ctx, schedules.ScheduleInput{
		Title:        "剪指甲",
		StartDate:    today.AddDate(0, 0, -5).Format(dateLayout),
		IntervalDays: 3,
	}); err != nil {
		t.Fatalf("create doubao schedule: %v", err)
	}
	if _, err := service.personal.Create(ctx, schedules.ScheduleInput{
		Title:        "运动",
		StartDate:    today.AddDate(0, 0, 2).Format(dateLayout),
		IntervalDays: 7,
	}); err != nil {
		t.Fatalf("create personal plan: %v", err)
	}

	response, err := service.List(ctx, ListOptions{
		Start: today.Format(dateLayout),
		End:   end.Format(dateLayout),
	})
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(response.Events) != 3 {
		t.Fatalf("events = %#v, want 3 events", response.Events)
	}
	if response.Summary.Today != 2 || response.Summary.Overdue != 1 {
		t.Fatalf("summary = %#v, want two today events and one overdue", response.Summary)
	}

	var overdue Event
	for _, event := range response.Events {
		if event.Source == SourceDoubaoSchedule {
			overdue = event
		}
	}
	if overdue.Date != today.Format(dateLayout) || overdue.OriginalDate == "" || overdue.Status != "overdue" {
		t.Fatalf("overdue event = %#v, want anchored today with original date", overdue)
	}

	filtered, err := service.List(ctx, ListOptions{
		Start:   today.Format(dateLayout),
		End:     end.Format(dateLayout),
		Sources: []string{SourcePersonalPlan},
	})
	if err != nil {
		t.Fatalf("List(filtered) error = %v", err)
	}
	if len(filtered.Events) != 1 || filtered.Events[0].Source != SourcePersonalPlan {
		t.Fatalf("filtered events = %#v, want one personal plan", filtered.Events)
	}
}

func TestListValidatesRangeAndSources(t *testing.T) {
	service, _ := newTestService(t)
	ctx := context.Background()

	if _, err := service.List(ctx, ListOptions{Start: "2026-10-20", End: "2026-10-01"}); err == nil {
		t.Fatal("List(reversed range) error = nil")
	}
	if _, err := service.List(ctx, ListOptions{Sources: []string{"unknown"}}); err == nil {
		t.Fatal("List(unknown source) error = nil")
	}
}
