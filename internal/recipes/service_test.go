package recipes

import (
	"context"
	"errors"
	"reflect"
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

func TestRecipeFoodCRUD(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	food, err := service.CreateFood(ctx, FoodInput{
		Name:     "鱼油",
		Category: "营养补充",
		Note:     "晚餐后半粒",
		Days:     []int{5, 1, 5},
	})
	if err != nil {
		t.Fatalf("CreateFood() error = %v", err)
	}
	if !reflect.DeepEqual(food.Days, []int{1, 5}) {
		t.Fatalf("created days = %#v, want [1 5]", food.Days)
	}

	if _, err := service.CreateFood(ctx, FoodInput{Name: "鱼油", Days: []int{1}}); err == nil {
		t.Fatal("duplicate CreateFood() error = nil")
	} else {
		var conflict *ConflictError
		if !errors.As(err, &conflict) {
			t.Fatalf("duplicate CreateFood() error = %T %v, want ConflictError", err, err)
		}
	}

	updated, err := service.UpdateFood(ctx, food.ID, FoodInput{
		Name:     "鱼油",
		Category: "营养补充",
		Note:     "早餐后半粒",
		Days:     []int{7, 2},
	})
	if err != nil {
		t.Fatalf("UpdateFood() error = %v", err)
	}
	if updated.Note != "早餐后半粒" || !reflect.DeepEqual(updated.Days, []int{2, 7}) {
		t.Fatalf("updated food = %#v", updated)
	}

	plan, err := service.GetPlan(ctx)
	if err != nil {
		t.Fatalf("GetPlan() error = %v", err)
	}
	if len(plan.Weekdays) != 7 || len(plan.Foods) != 1 || plan.Today < 1 || plan.Today > 7 {
		t.Fatalf("plan = %#v", plan)
	}
	if !plan.Weekdays[plan.Today-1].IsToday {
		t.Fatalf("today weekday = %#v", plan.Weekdays)
	}

	if err := service.DeleteFood(ctx, food.ID); err != nil {
		t.Fatalf("DeleteFood() error = %v", err)
	}
	if _, err := service.GetFood(ctx, food.ID); !errors.Is(err, ErrNotFound) {
		t.Fatalf("GetFood(deleted) error = %v, want ErrNotFound", err)
	}
}

func TestRecipeFoodValidation(t *testing.T) {
	service := newTestService(t)
	ctx := context.Background()

	for _, input := range []FoodInput{
		{Name: "没有日期"},
		{Name: "日期不正确", Days: []int{8}},
		{Name: "", Days: []int{1}},
	} {
		if _, err := service.CreateFood(ctx, input); err == nil {
			t.Fatalf("CreateFood(%#v) error = nil", input)
		} else {
			var validation *ValidationError
			if !errors.As(err, &validation) {
				t.Fatalf("CreateFood(%#v) error = %T %v, want ValidationError", input, err, err)
			}
		}
	}
}
