package calendar

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/Icingworld/doubao-space/internal/schedules"
	"github.com/Icingworld/doubao-space/internal/tasks"
)

const dateLayout = "2006-01-02"

const (
	SourceTask           = "task"
	SourceDoubaoSchedule = "doubao_schedule"
	SourcePersonalPlan   = "personal_plan"
)

var sourceLabels = map[string]string{
	SourceTask:           "任务",
	SourceDoubaoSchedule: "豆包日程",
	SourcePersonalPlan:   "个人计划",
}

var sourceRoutes = map[string]string{
	SourceTask:           "/dashboard/tasks",
	SourceDoubaoSchedule: "/dashboard/doubao#doubao-schedule",
	SourcePersonalPlan:   "/dashboard/plans#personal-plans",
}

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string { return e.Message }

type ListOptions struct {
	Start            string
	End              string
	Sources          []string
	IncludeCompleted bool
}

type Event struct {
	ID              string   `json:"id"`
	Source          string   `json:"source"`
	SourceID        int64    `json:"source_id"`
	SourceLabel     string   `json:"source_label"`
	Title           string   `json:"title"`
	Date            string   `json:"date"`
	StartAt         string   `json:"start_at,omitempty"`
	AllDay          bool     `json:"all_day"`
	Status          string   `json:"status"`
	StatusLabel     string   `json:"status_label"`
	Completed       bool     `json:"completed"`
	CanComplete     bool     `json:"can_complete"`
	Priority        string   `json:"priority,omitempty"`
	PriorityLabel   string   `json:"priority_label,omitempty"`
	Path            []string `json:"path,omitempty"`
	Description     string   `json:"description,omitempty"`
	Note            string   `json:"note,omitempty"`
	Route           string   `json:"route,omitempty"`
	OriginalDate    string   `json:"original_date,omitempty"`
	OverdueDays     int      `json:"overdue_days,omitempty"`
	IntervalDays    int      `json:"interval_days,omitempty"`
	NextDueDate     string   `json:"next_due_date,omitempty"`
	LastCompletedOn string   `json:"last_completed_on,omitempty"`
}

type Summary struct {
	Total           int `json:"total"`
	Today           int `json:"today"`
	Overdue         int `json:"overdue"`
	Tasks           int `json:"tasks"`
	DoubaoSchedules int `json:"doubao_schedules"`
	PersonalPlans   int `json:"personal_plans"`
}

type Response struct {
	Today   string  `json:"today"`
	Start   string  `json:"start"`
	End     string  `json:"end"`
	Events  []Event `json:"events"`
	Summary Summary `json:"summary"`
}

type Service struct {
	tasks    *tasks.Service
	doubao   *schedules.Service
	personal *schedules.Service
}

func NewService(db *sql.DB) *Service {
	return &Service{
		tasks:    tasks.NewService(db),
		doubao:   schedules.NewService(db),
		personal: schedules.NewPersonalService(db),
	}
}

func (s *Service) List(ctx context.Context, options ListOptions) (*Response, error) {
	start, end, err := normalizeRange(options.Start, options.End)
	if err != nil {
		return nil, err
	}
	selected, err := normalizeSources(options.Sources)
	if err != nil {
		return nil, err
	}

	today := localDate(time.Now())
	events := make([]Event, 0)
	if selected[SourceTask] {
		tree, err := s.tasks.List(ctx, tasks.ListOptions{IncludeCompleted: options.IncludeCompleted, Sort: "due"})
		if err != nil {
			return nil, fmt.Errorf("list calendar tasks: %w", err)
		}
		for _, task := range flattenTasks(tree) {
			if event, ok := taskEvent(task, today, start, end); ok {
				events = append(events, event)
			}
		}
	}

	if selected[SourceDoubaoSchedule] {
		plan, err := s.doubao.GetPlan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list calendar doubao schedules: %w", err)
		}
		items, err := recurringEvents(plan, SourceDoubaoSchedule, today, start, end)
		if err != nil {
			return nil, err
		}
		events = append(events, items...)
	}

	if selected[SourcePersonalPlan] {
		plan, err := s.personal.GetPlan(ctx)
		if err != nil {
			return nil, fmt.Errorf("list calendar personal plans: %w", err)
		}
		items, err := recurringEvents(plan, SourcePersonalPlan, today, start, end)
		if err != nil {
			return nil, err
		}
		events = append(events, items...)
	}

	sortEvents(events)
	return &Response{
		Today:   today.Format(dateLayout),
		Start:   start.Format(dateLayout),
		End:     end.Format(dateLayout),
		Events:  events,
		Summary: summarize(events, today),
	}, nil
}

func taskEvent(task *tasks.Task, today, start, end time.Time) (Event, bool) {
	if task == nil || task.DueAt == nil {
		return Event{}, false
	}

	dueAt := task.DueAt.In(time.Local)
	date := localDate(dueAt)
	if date.Before(start) || date.After(end) {
		return Event{}, false
	}

	status := "upcoming"
	statusLabel := "待处理"
	if task.Completed {
		status = "completed"
		statusLabel = "已完成"
	} else if dueAt.Before(time.Now()) {
		status = "overdue"
		statusLabel = "已逾期"
	} else if sameDate(date, today) {
		status = "due"
		statusLabel = "今天"
	}

	return Event{
		ID:            fmt.Sprintf("%s:%d", SourceTask, task.ID),
		Source:        SourceTask,
		SourceID:      task.ID,
		SourceLabel:   sourceLabels[SourceTask],
		Title:         task.Title,
		Date:          date.Format(dateLayout),
		StartAt:       dueAt.Format(time.RFC3339),
		Status:        status,
		StatusLabel:   statusLabel,
		Completed:     task.Completed,
		CanComplete:   !task.Completed,
		Priority:      task.EffectivePriority,
		PriorityLabel: priorityLabel(task.EffectivePriority),
		Path:          append([]string(nil), task.Path...),
		Description:   task.Description,
		Route:         fmt.Sprintf("%s/%d", sourceRoutes[SourceTask], task.ID),
	}, true
}

func recurringEvents(plan *schedules.Plan, source string, today, start, end time.Time) ([]Event, error) {
	if plan == nil {
		return nil, nil
	}
	items := make([]Event, 0, len(plan.Schedules))
	for _, item := range plan.Schedules {
		nextDue, err := time.ParseInLocation(dateLayout, item.NextDueDate, time.Local)
		if err != nil {
			return nil, fmt.Errorf("parse %s next due date %q: %w", source, item.NextDueDate, err)
		}
		nextDue = localDate(nextDue)
		displayDate := nextDue
		status := "upcoming"
		statusLabel := "即将到期"
		if item.IsDue {
			if nextDue.Before(today) {
				displayDate = today
				status = "overdue"
				statusLabel = "已逾期"
			} else {
				status = "due"
				statusLabel = "今天"
			}
		}
		if displayDate.Before(start) || displayDate.After(end) {
			continue
		}

		event := Event{
			ID:           fmt.Sprintf("%s:%d", source, item.ID),
			Source:       source,
			SourceID:     item.ID,
			SourceLabel:  sourceLabels[source],
			Title:        item.Title,
			Date:         displayDate.Format(dateLayout),
			AllDay:       true,
			Status:       status,
			StatusLabel:  statusLabel,
			CanComplete:  item.IsDue,
			Note:         item.Note,
			Route:        sourceRoutes[source],
			IntervalDays: item.IntervalDays,
			NextDueDate:  item.NextDueDate,
			OverdueDays:  item.OverdueDays,
		}
		if item.LastCompletedOn != nil {
			event.LastCompletedOn = *item.LastCompletedOn
		}
		if !sameDate(displayDate, nextDue) {
			event.OriginalDate = item.NextDueDate
		}
		items = append(items, event)
	}
	return items, nil
}

func summarize(events []Event, today time.Time) Summary {
	result := Summary{Total: len(events)}
	for _, event := range events {
		switch event.Source {
		case SourceTask:
			result.Tasks++
		case SourceDoubaoSchedule:
			result.DoubaoSchedules++
		case SourcePersonalPlan:
			result.PersonalPlans++
		}
		if event.Status == "overdue" {
			result.Overdue++
		}
		if event.Date == today.Format(dateLayout) && !event.Completed {
			result.Today++
		}
	}
	return result
}

func normalizeRange(startValue, endValue string) (time.Time, time.Time, error) {
	today := localDate(time.Now())
	startValue = strings.TrimSpace(startValue)
	endValue = strings.TrimSpace(endValue)

	var start, end time.Time
	var err error
	if startValue == "" {
		if endValue == "" {
			start = time.Date(today.Year(), today.Month(), 1, 0, 0, 0, 0, time.Local)
		} else {
			end, err = parseDate(endValue)
			if err != nil {
				return time.Time{}, time.Time{}, &ValidationError{Message: "日历结束日期格式不正确，请使用 YYYY-MM-DD。"}
			}
			start = end.AddDate(0, 0, -41)
		}
	} else {
		start, err = parseDate(startValue)
		if err != nil {
			return time.Time{}, time.Time{}, &ValidationError{Message: "日历开始日期格式不正确，请使用 YYYY-MM-DD。"}
		}
	}
	if end.IsZero() {
		if endValue == "" {
			base := start
			if base.IsZero() {
				base = today
			}
			end = time.Date(base.Year(), base.Month()+1, 0, 0, 0, 0, 0, time.Local)
		} else {
			end, err = parseDate(endValue)
			if err != nil {
				return time.Time{}, time.Time{}, &ValidationError{Message: "日历结束日期格式不正确，请使用 YYYY-MM-DD。"}
			}
		}
	}
	if end.Before(start) {
		return time.Time{}, time.Time{}, &ValidationError{Message: "日历开始日期不能晚于结束日期。"}
	}
	return start, end, nil
}

func normalizeSources(values []string) (map[string]bool, error) {
	selected := map[string]bool{
		SourceTask:           false,
		SourceDoubaoSchedule: false,
		SourcePersonalPlan:   false,
	}
	if len(values) == 0 {
		for source := range selected {
			selected[source] = true
		}
		return selected, nil
	}

	found := false
	for _, value := range values {
		for _, source := range strings.Split(value, ",") {
			source = strings.TrimSpace(source)
			if source == "" {
				continue
			}
			if source == "all" {
				for key := range selected {
					selected[key] = true
				}
				found = true
				continue
			}
			if _, ok := selected[source]; !ok {
				return nil, &ValidationError{Message: fmt.Sprintf("不支持的日历来源：%s。", source)}
			}
			selected[source] = true
			found = true
		}
	}
	if !found {
		return nil, &ValidationError{Message: "日历来源不能为空。"}
	}
	return selected, nil
}

func flattenTasks(nodes []*tasks.Task) []*tasks.Task {
	result := make([]*tasks.Task, 0)
	for _, node := range nodes {
		result = append(result, node)
		result = append(result, flattenTasks(node.Children)...)
	}
	return result
}

func sortEvents(events []Event) {
	sort.SliceStable(events, func(i, j int) bool {
		if events[i].Date != events[j].Date {
			return events[i].Date < events[j].Date
		}
		leftStatus, rightStatus := statusRank(events[i].Status), statusRank(events[j].Status)
		if leftStatus != rightStatus {
			return leftStatus < rightStatus
		}
		if events[i].StartAt != events[j].StartAt {
			return events[i].StartAt < events[j].StartAt
		}
		if events[i].Source != events[j].Source {
			return events[i].Source < events[j].Source
		}
		return events[i].Title < events[j].Title
	})
}

func statusRank(status string) int {
	switch status {
	case "overdue":
		return 0
	case "due":
		return 1
	case "upcoming":
		return 2
	case "completed":
		return 3
	default:
		return 4
	}
}

func priorityLabel(priority string) string {
	switch priority {
	case tasks.PriorityUrgent:
		return "紧急"
	case tasks.PriorityHigh:
		return "高优先级"
	case tasks.PriorityLow:
		return "低优先级"
	default:
		return "中优先级"
	}
}

func parseDate(value string) (time.Time, error) {
	return time.ParseInLocation(dateLayout, strings.TrimSpace(value), time.Local)
}

func localDate(value time.Time) time.Time {
	local := value.In(time.Local)
	return time.Date(local.Year(), local.Month(), local.Day(), 0, 0, 0, 0, time.Local)
}

func sameDate(left, right time.Time) bool {
	return left.Year() == right.Year() && left.YearDay() == right.YearDay()
}
