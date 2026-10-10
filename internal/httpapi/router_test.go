package httpapi

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/Icingworld/doubao-space/internal/database"
)

func TestHealthAndFrontendRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)

	health := httptest.NewRecorder()
	router.ServeHTTP(health, httptest.NewRequest(http.MethodGet, "/api/health", nil))
	if health.Code != http.StatusOK {
		t.Fatalf("health status = %d, want %d", health.Code, http.StatusOK)
	}

	frontend := httptest.NewRecorder()
	router.ServeHTTP(frontend, httptest.NewRequest(http.MethodGet, "/dashboard", nil))
	if frontend.Code != http.StatusOK {
		t.Fatalf("frontend status = %d, want %d", frontend.Code, http.StatusOK)
	}
}

func TestTaskRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/tasks", bytes.NewBufferString(`{"title":"测试任务","priority":"high"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create task status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created task: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created task id = 0")
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/tasks?include_completed=true", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list tasks status = %d, body = %s", list.Code, list.Body.String())
	}
	var tasks []struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode task list: %v", err)
	}
	if len(tasks) != 1 || tasks[0].ID != created.ID {
		t.Fatalf("task list = %#v, want one created task", tasks)
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/tasks/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete task status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

func TestRecipeRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/recipes/foods", bytes.NewBufferString(`{"name":"罐头","category":"主食","days":[1,3,5]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create recipe food status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created recipe food: %v", err)
	}
	if created.ID == 0 {
		t.Fatal("created recipe food id = 0")
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/recipes", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list recipes status = %d, body = %s", list.Code, list.Body.String())
	}
	var plan struct {
		Foods []struct {
			ID   int64 `json:"id"`
			Days []int `json:"days"`
		} `json:"foods"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode recipe plan: %v", err)
	}
	if len(plan.Foods) != 1 || plan.Foods[0].ID != created.ID {
		t.Fatalf("recipe foods = %#v", plan.Foods)
	}

	update := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/recipes/foods/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(`{"name":"罐头","category":"主食","days":[2,7]}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(update, updateRequest)
	if update.Code != http.StatusOK {
		t.Fatalf("update recipe food status = %d, body = %s", update.Code, update.Body.String())
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/recipes/foods/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete recipe food status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

func TestScheduleRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/schedules", bytes.NewBufferString(`{"title":"剪指甲","note":"每月一次","start_date":"2020-01-01","interval_days":30}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create schedule status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID    int64 `json:"id"`
		IsDue bool  `json:"is_due"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created schedule: %v", err)
	}
	if created.ID == 0 || !created.IsDue {
		t.Fatalf("created schedule = %#v, want due schedule", created)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/schedules", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list schedules status = %d, body = %s", list.Code, list.Body.String())
	}
	var plan struct {
		DueCount int `json:"due_count"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode schedule plan: %v", err)
	}
	if plan.DueCount != 1 {
		t.Fatalf("schedule plan = %#v, want due_count 1", plan)
	}

	complete := httptest.NewRecorder()
	router.ServeHTTP(complete, httptest.NewRequest(http.MethodPost, "/api/schedules/"+strconv.FormatInt(created.ID, 10)+"/complete", nil))
	if complete.Code != http.StatusOK {
		t.Fatalf("complete schedule status = %d, body = %s", complete.Code, complete.Body.String())
	}
	var completed struct {
		IsDue bool `json:"is_due"`
	}
	if err := json.Unmarshal(complete.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode completed schedule: %v", err)
	}
	if completed.IsDue {
		t.Fatalf("completed schedule = %#v, want upcoming", completed)
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/schedules/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete schedule status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

func TestPersonalPlanRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/plans", bytes.NewBufferString(`{"title":"阅读","note":"每天读一点","start_date":"2020-01-01","interval_days":7}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create personal plan status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID    int64 `json:"id"`
		IsDue bool  `json:"is_due"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created personal plan: %v", err)
	}
	if created.ID == 0 || !created.IsDue {
		t.Fatalf("created personal plan = %#v, want due plan", created)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/plans", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list personal plans status = %d, body = %s", list.Code, list.Body.String())
	}
	var plan struct {
		DueCount int `json:"due_count"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &plan); err != nil {
		t.Fatalf("decode personal plan list: %v", err)
	}
	if plan.DueCount != 1 {
		t.Fatalf("personal plan = %#v, want due_count 1", plan)
	}

	complete := httptest.NewRecorder()
	router.ServeHTTP(complete, httptest.NewRequest(http.MethodPost, "/api/plans/"+strconv.FormatInt(created.ID, 10)+"/complete", nil))
	if complete.Code != http.StatusOK {
		t.Fatalf("complete personal plan status = %d, body = %s", complete.Code, complete.Body.String())
	}
	var completed struct {
		IsDue bool `json:"is_due"`
	}
	if err := json.Unmarshal(complete.Body.Bytes(), &completed); err != nil {
		t.Fatalf("decode completed personal plan: %v", err)
	}
	if completed.IsDue {
		t.Fatalf("completed personal plan = %#v, want upcoming", completed)
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/plans/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete personal plan status = %d, body = %s", remove.Code, remove.Body.String())
	}
}
