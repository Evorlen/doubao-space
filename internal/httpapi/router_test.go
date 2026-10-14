package httpapi

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
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

	schedule := httptest.NewRecorder()
	router.ServeHTTP(schedule, httptest.NewRequest(http.MethodPost, "/api/tasks/"+strconv.FormatInt(created.ID, 10)+"/schedule", nil))
	if schedule.Code != http.StatusOK {
		t.Fatalf("schedule task status = %d, body = %s", schedule.Code, schedule.Body.String())
	}
	var scheduled struct {
		Scheduled bool `json:"scheduled"`
	}
	if err := json.Unmarshal(schedule.Body.Bytes(), &scheduled); err != nil {
		t.Fatalf("decode scheduled task: %v", err)
	}
	if !scheduled.Scheduled {
		t.Fatal("scheduled task = false, want true")
	}

	unschedule := httptest.NewRecorder()
	router.ServeHTTP(unschedule, httptest.NewRequest(http.MethodDelete, "/api/tasks/"+strconv.FormatInt(created.ID, 10)+"/schedule", nil))
	if unschedule.Code != http.StatusOK {
		t.Fatalf("unschedule task status = %d, body = %s", unschedule.Code, unschedule.Body.String())
	}
	if err := json.Unmarshal(unschedule.Body.Bytes(), &scheduled); err != nil {
		t.Fatalf("decode unscheduled task: %v", err)
	}
	if scheduled.Scheduled {
		t.Fatal("unscheduled task = true, want false")
	}

	template := httptest.NewRecorder()
	router.ServeHTTP(template, httptest.NewRequest(http.MethodGet, "/api/tasks/import/template", nil))
	if template.Code != http.StatusOK {
		t.Fatalf("task import template status = %d, body = %s", template.Code, template.Body.String())
	}
	if got := template.Header().Get("Content-Type"); got != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Fatalf("task import template content type = %q", got)
	}
	if body := template.Body.Bytes(); len(body) < 2 || body[0] != 'P' || body[1] != 'K' {
		t.Fatalf("task import template is not an xlsx archive")
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

	reorder := httptest.NewRecorder()
	reorderRequest := httptest.NewRequest(
		http.MethodPut,
		"/api/recipes/order",
		bytes.NewBufferString(`{"food_ids":[`+strconv.FormatInt(created.ID, 10)+`],"weekday_values":[3,1,2,4,5,6,7]}`),
	)
	reorderRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(reorder, reorderRequest)
	if reorder.Code != http.StatusOK {
		t.Fatalf("reorder recipe status = %d, body = %s", reorder.Code, reorder.Body.String())
	}
	var reordered struct {
		Weekdays []struct {
			Value int `json:"value"`
		} `json:"weekdays"`
	}
	if err := json.Unmarshal(reorder.Body.Bytes(), &reordered); err != nil {
		t.Fatalf("decode reordered recipe: %v", err)
	}
	if len(reordered.Weekdays) != 7 || reordered.Weekdays[0].Value != 3 {
		t.Fatalf("reordered weekdays = %#v, want Wednesday first", reordered.Weekdays)
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

func TestCalendarRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/calendar?start=2026-10-01&end=2026-10-31", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list calendar status = %d, body = %s", list.Code, list.Body.String())
	}
	var response struct {
		Today  string `json:"today"`
		Events []struct {
			ID string `json:"id"`
		} `json:"events"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode calendar response: %v", err)
	}
	if response.Today == "" || response.Events == nil {
		t.Fatalf("calendar response = %#v, want today and events", response)
	}
}

func TestEntertainmentRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/entertainment", bytes.NewBufferString(`{"media_type":"movie","title":"海街日记","creator":"是枝裕和","status":"planned","rating":5,"favorite":true,"tags":["治愈"]}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create entertainment item status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID             int64  `json:"id"`
		MediaTypeLabel string `json:"media_type_label"`
		Rating         int    `json:"rating"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created entertainment item: %v", err)
	}
	if created.ID == 0 || created.MediaTypeLabel != "电影" || created.Rating != 5 {
		t.Fatalf("created entertainment item = %#v", created)
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/entertainment?type=movie&favorite=true", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list entertainment items status = %d, body = %s", list.Code, list.Body.String())
	}
	var response struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Stats struct {
			Total int `json:"total"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode entertainment response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != created.ID || response.Stats.Total != 1 {
		t.Fatalf("entertainment response = %#v", response)
	}

	update := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/entertainment/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(`{"media_type":"movie","title":"海街日记","creator":"是枝裕和","status":"completed","rating":5,"favorite":true,"review":"很温柔。"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(update, updateRequest)
	if update.Code != http.StatusOK {
		t.Fatalf("update entertainment item status = %d, body = %s", update.Code, update.Body.String())
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/entertainment/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete entertainment item status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

func TestMemoRoutes(t *testing.T) {
	db, err := database.Open(t.TempDir())
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouter(db, "test", true)
	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/memos", bytes.NewBufferString(`{"title":"随手记","content_html":"<p><strong>记得买牛奶</strong></p>"}`))
	request.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create memo status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID      int64  `json:"id"`
		Excerpt string `json:"excerpt"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created memo: %v", err)
	}
	if created.ID == 0 || created.Excerpt != "记得买牛奶" {
		t.Fatalf("created memo = %#v", created)
	}

	archive := httptest.NewRecorder()
	router.ServeHTTP(archive, httptest.NewRequest(http.MethodPost, "/api/memos/"+strconv.FormatInt(created.ID, 10)+"/archive", nil))
	if archive.Code != http.StatusOK {
		t.Fatalf("archive memo status = %d, body = %s", archive.Code, archive.Body.String())
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/memos?archived=true", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list archived memos status = %d, body = %s", list.Code, list.Body.String())
	}
	var response struct {
		Items []struct {
			ID       int64 `json:"id"`
			Archived bool  `json:"archived"`
		} `json:"items"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode memo response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != created.ID || !response.Items[0].Archived {
		t.Fatalf("memo response = %#v", response)
	}

	restore := httptest.NewRecorder()
	router.ServeHTTP(restore, httptest.NewRequest(http.MethodPost, "/api/memos/"+strconv.FormatInt(created.ID, 10)+"/restore", nil))
	if restore.Code != http.StatusOK {
		t.Fatalf("restore memo status = %d, body = %s", restore.Code, restore.Body.String())
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/memos/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete memo status = %d, body = %s", remove.Code, remove.Body.String())
	}
}

func TestDoubaoAlbumRoutes(t *testing.T) {
	dataDir := t.TempDir()
	db, err := database.Open(dataDir)
	if err != nil {
		t.Fatalf("database.Open() error = %v", err)
	}
	defer db.Close()

	router := NewRouterWithDataDir(db, "test", true, dataDir)
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)
	part, err := writer.CreateFormFile("file", "豆包.png")
	if err != nil {
		t.Fatalf("create multipart file: %v", err)
	}
	if _, err := part.Write([]byte("fake image")); err != nil {
		t.Fatalf("write multipart file: %v", err)
	}
	if err := writer.WriteField("note", "第一次记录"); err != nil {
		t.Fatalf("write multipart note: %v", err)
	}
	if err := writer.Close(); err != nil {
		t.Fatalf("close multipart writer: %v", err)
	}

	create := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/doubao/album", body)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	router.ServeHTTP(create, request)
	if create.Code != http.StatusCreated {
		t.Fatalf("create album item status = %d, body = %s", create.Code, create.Body.String())
	}
	var created struct {
		ID        int64  `json:"id"`
		MediaType string `json:"media_type"`
		Note      string `json:"note"`
	}
	if err := json.Unmarshal(create.Body.Bytes(), &created); err != nil {
		t.Fatalf("decode created album item: %v", err)
	}
	if created.ID == 0 || created.MediaType != "photo" || created.Note != "第一次记录" {
		t.Fatalf("created album item = %#v", created)
	}

	media := httptest.NewRecorder()
	router.ServeHTTP(media, httptest.NewRequest(http.MethodGet, "/api/doubao/album/"+strconv.FormatInt(created.ID, 10)+"/media", nil))
	if media.Code != http.StatusOK || media.Header().Get("Content-Type") != "image/png" || media.Body.String() != "fake image" {
		t.Fatalf("media response status = %d, content-type = %q, body = %q", media.Code, media.Header().Get("Content-Type"), media.Body.String())
	}

	update := httptest.NewRecorder()
	updateRequest := httptest.NewRequest(http.MethodPut, "/api/doubao/album/"+strconv.FormatInt(created.ID, 10), bytes.NewBufferString(`{"note":"更新备注"}`))
	updateRequest.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(update, updateRequest)
	if update.Code != http.StatusOK {
		t.Fatalf("update album item status = %d, body = %s", update.Code, update.Body.String())
	}

	list := httptest.NewRecorder()
	router.ServeHTTP(list, httptest.NewRequest(http.MethodGet, "/api/doubao/album", nil))
	if list.Code != http.StatusOK {
		t.Fatalf("list album status = %d, body = %s", list.Code, list.Body.String())
	}
	var response struct {
		Items []struct {
			ID int64 `json:"id"`
		} `json:"items"`
		Stats struct {
			Total int `json:"total"`
		} `json:"stats"`
	}
	if err := json.Unmarshal(list.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode album response: %v", err)
	}
	if len(response.Items) != 1 || response.Items[0].ID != created.ID || response.Stats.Total != 1 {
		t.Fatalf("album response = %#v", response)
	}

	remove := httptest.NewRecorder()
	router.ServeHTTP(remove, httptest.NewRequest(http.MethodDelete, "/api/doubao/album/"+strconv.FormatInt(created.ID, 10), nil))
	if remove.Code != http.StatusNoContent {
		t.Fatalf("delete album item status = %d, body = %s", remove.Code, remove.Body.String())
	}
}
