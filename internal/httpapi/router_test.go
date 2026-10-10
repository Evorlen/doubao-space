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
