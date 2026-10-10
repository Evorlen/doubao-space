package recipes

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

var ErrNotFound = errors.New("recipe food not found")

type ValidationError struct {
	Message string
}

func (e *ValidationError) Error() string {
	return e.Message
}

type ConflictError struct {
	Message string
}

func (e *ConflictError) Error() string {
	return e.Message
}

type Food struct {
	ID        int64  `json:"id"`
	Name      string `json:"name"`
	Category  string `json:"category"`
	Note      string `json:"note"`
	Days      []int  `json:"days"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Weekday struct {
	Value     int    `json:"value"`
	Label     string `json:"label"`
	Date      string `json:"date"`
	DateLabel string `json:"date_label"`
	IsToday   bool   `json:"is_today"`
}

type Plan struct {
	Today      int       `json:"today"`
	TodayLabel string    `json:"today_label"`
	WeekStart  string    `json:"week_start"`
	WeekEnd    string    `json:"week_end"`
	Weekdays   []Weekday `json:"weekdays"`
	Foods      []Food    `json:"foods"`
}

type FoodInput struct {
	Name     string
	Category string
	Note     string
	Days     []int
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) GetPlan(ctx context.Context) (*Plan, error) {
	foods, err := s.listFoods(ctx)
	if err != nil {
		return nil, err
	}

	now := time.Now()
	today := weekdayNumber(now)
	labels := []string{"", "周一", "周二", "周三", "周四", "周五", "周六", "周日"}
	monday := now.AddDate(0, 0, -(today - 1))
	weekdays := make([]Weekday, 0, 7)
	for index := 1; index <= 7; index++ {
		date := monday.AddDate(0, 0, index-1)
		weekdays = append(weekdays, Weekday{
			Value:     index,
			Label:     labels[index],
			Date:      date.Format("2006-01-02"),
			DateLabel: date.Format("01/02"),
			IsToday:   index == today,
		})
	}

	return &Plan{
		Today:      today,
		TodayLabel: labels[today],
		WeekStart:  weekdays[0].Date,
		WeekEnd:    weekdays[len(weekdays)-1].Date,
		Weekdays:   weekdays,
		Foods:      foods,
	}, nil
}

func (s *Service) CreateFood(ctx context.Context, input FoodInput) (*Food, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin create recipe food: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		INSERT INTO recipe_foods (name, category, note)
		VALUES (?, ?, ?)
	`, normalized.Name, normalized.Category, normalized.Note)
	if err != nil {
		if isConstraintError(err) {
			return nil, &ConflictError{Message: "已经有同名食物了，请换一个名称"}
		}
		return nil, fmt.Errorf("create recipe food: %w", err)
	}
	id, err := result.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("read created recipe food id: %w", err)
	}
	if err := insertDays(ctx, tx, id, normalized.Days); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit recipe food: %w", err)
	}
	return s.GetFood(ctx, id)
}

func (s *Service) GetFood(ctx context.Context, id int64) (*Food, error) {
	foods, err := s.listFoods(ctx, id)
	if err != nil {
		return nil, err
	}
	if len(foods) == 0 {
		return nil, ErrNotFound
	}
	return &foods[0], nil
}

func (s *Service) UpdateFood(ctx context.Context, id int64, input FoodInput) (*Food, error) {
	normalized, err := normalizeInput(input)
	if err != nil {
		return nil, err
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin update recipe food: %w", err)
	}
	defer tx.Rollback()

	result, err := tx.ExecContext(ctx, `
		UPDATE recipe_foods
		SET name = ?, category = ?, note = ?, updated_at = CURRENT_TIMESTAMP
		WHERE id = ?
	`, normalized.Name, normalized.Category, normalized.Note, id)
	if err != nil {
		if isConstraintError(err) {
			return nil, &ConflictError{Message: "已经有同名食物了，请换一个名称"}
		}
		return nil, fmt.Errorf("update recipe food: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return nil, fmt.Errorf("read updated recipe food count: %w", err)
	}
	if rows == 0 {
		return nil, ErrNotFound
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM recipe_food_days WHERE food_id = ?`, id); err != nil {
		return nil, fmt.Errorf("replace recipe food days: %w", err)
	}
	if err := insertDays(ctx, tx, id, normalized.Days); err != nil {
		return nil, err
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit recipe food update: %w", err)
	}
	return s.GetFood(ctx, id)
}

func (s *Service) DeleteFood(ctx context.Context, id int64) error {
	result, err := s.db.ExecContext(ctx, `DELETE FROM recipe_foods WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete recipe food: %w", err)
	}
	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read deleted recipe food count: %w", err)
	}
	if rows == 0 {
		return ErrNotFound
	}
	return nil
}

func (s *Service) listFoods(ctx context.Context, ids ...int64) ([]Food, error) {
	query := `
		SELECT f.id, f.name, f.category, f.note, f.created_at, f.updated_at, d.weekday
		FROM recipe_foods f
		LEFT JOIN recipe_food_days d ON d.food_id = f.id
	`
	args := make([]any, 0, len(ids))
	if len(ids) > 0 {
		query += " WHERE f.id = ?"
		args = append(args, ids[0])
	}
	query += " ORDER BY f.id ASC, d.weekday ASC"

	rows, err := s.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("list recipe foods: %w", err)
	}
	defer rows.Close()

	foodByID := make(map[int64]*Food)
	order := make([]int64, 0)
	for rows.Next() {
		var (
			id        int64
			name      string
			category  string
			note      string
			createdAt string
			updatedAt string
			weekday   sql.NullInt64
		)
		if err := rows.Scan(&id, &name, &category, &note, &createdAt, &updatedAt, &weekday); err != nil {
			return nil, fmt.Errorf("scan recipe food: %w", err)
		}
		food, exists := foodByID[id]
		if !exists {
			food = &Food{
				ID:        id,
				Name:      name,
				Category:  category,
				Note:      note,
				Days:      make([]int, 0, 3),
				CreatedAt: createdAt,
				UpdatedAt: updatedAt,
			}
			foodByID[id] = food
			order = append(order, id)
		}
		if weekday.Valid {
			food.Days = append(food.Days, int(weekday.Int64))
		}
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("read recipe foods: %w", err)
	}

	foods := make([]Food, 0, len(order))
	for _, id := range order {
		foods = append(foods, *foodByID[id])
	}
	return foods, nil
}

func normalizeInput(input FoodInput) (FoodInput, error) {
	input.Name = strings.TrimSpace(input.Name)
	input.Category = strings.TrimSpace(input.Category)
	input.Note = strings.TrimSpace(input.Note)
	if input.Category == "" {
		input.Category = "其他"
	}
	if input.Name == "" {
		return FoodInput{}, &ValidationError{Message: "请输入食物名称"}
	}
	if utf8.RuneCountInString(input.Name) > 100 {
		return FoodInput{}, &ValidationError{Message: "食物名称不能超过 100 个字"}
	}
	if utf8.RuneCountInString(input.Category) > 30 {
		return FoodInput{}, &ValidationError{Message: "食物分类不能超过 30 个字"}
	}
	if utf8.RuneCountInString(input.Note) > 300 {
		return FoodInput{}, &ValidationError{Message: "备注不能超过 300 个字"}
	}
	if len(input.Days) == 0 {
		return FoodInput{}, &ValidationError{Message: "请至少选择一天"}
	}

	seen := make(map[int]bool, len(input.Days))
	days := make([]int, 0, len(input.Days))
	for _, day := range input.Days {
		if day < 1 || day > 7 {
			return FoodInput{}, &ValidationError{Message: "食谱日期必须在周一到周日之间"}
		}
		if !seen[day] {
			seen[day] = true
			days = append(days, day)
		}
	}
	sort.Ints(days)
	input.Days = days
	return input, nil
}

func insertDays(ctx context.Context, tx *sql.Tx, foodID int64, days []int) error {
	for _, day := range days {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO recipe_food_days (food_id, weekday) VALUES (?, ?)
		`, foodID, day); err != nil {
			return fmt.Errorf("save recipe food day: %w", err)
		}
	}
	return nil
}

func isConstraintError(err error) bool {
	message := strings.ToLower(err.Error())
	return strings.Contains(message, "constraint") || strings.Contains(message, "unique")
}

func weekdayNumber(value time.Time) int {
	day := int(value.Weekday())
	if day == 0 {
		return 7
	}
	return day
}
