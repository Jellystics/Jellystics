package handler_test

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	statssvc "github.com/Jellystics/Jellystics/internal/service/stats"
	tasksvc "github.com/Jellystics/Jellystics/internal/service/task"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// UtilsHandler.GeolocateIP
// ---------------------------------------------------------------------------

// TestGeolocateIP_MissingConfig verifies that when the GeoLite env vars are
// unset the endpoint short-circuits to 501 Not Implemented (never calling out).
func TestGeolocateIP_MissingConfig(t *testing.T) {
	t.Setenv("JS_GEOLITE_ACCOUNT_ID", "")
	t.Setenv("JS_GEOLITE_LICENSE_KEY", "")

	gin.SetMode(gin.TestMode)
	h := handler.NewUtilsHandler()
	r := gin.New()
	r.POST("/geo", h.GeolocateIP)

	w := postJSON(t, r, "/geo", `{"ipAddress":"8.8.8.8"}`)
	if w.Code != http.StatusNotImplemented {
		t.Errorf("status = %d, want 501", w.Code)
	}
}

// TestGeolocateIP_InvalidIP verifies that with config present but an invalid or
// private IP, the endpoint returns 400 before making any network call.
func TestGeolocateIP_InvalidIP(t *testing.T) {
	t.Setenv("JS_GEOLITE_ACCOUNT_ID", "acct")
	t.Setenv("JS_GEOLITE_LICENSE_KEY", "key")

	gin.SetMode(gin.TestMode)
	h := handler.NewUtilsHandler()
	r := gin.New()
	r.POST("/geo", h.GeolocateIP)

	cases := []string{
		`{"ipAddress":"not-an-ip"}`,   // unparseable
		`{"ipAddress":"192.168.1.5"}`, // private range
		`{"ipAddress":"127.0.0.1"}`,   // loopback
		`{"ipAddress":"10.0.0.1"}`,    // private range
	}
	for _, body := range cases {
		w := postJSON(t, r, "/geo", body)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %s status = %d, want 400", body, w.Code)
		}
	}
}

// ---------------------------------------------------------------------------
// StatsHandler (service-backed)
// ---------------------------------------------------------------------------

// seedStatsService inserts one user, one movie library + item, and two plays so
// the service-layer aggregates have known values.
func seedStatsService(t *testing.T, db *gorm.DB) {
	t.Helper()
	repos := repository.New(db)
	ctx := t.Context()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	genres := datatypes.JSON([]byte("[]"))

	must(repos.User.Upsert(ctx, []models.JFUser{{Id: "u1", Name: sp("Alice")}}))
	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
	}))
	movie := "Movie"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"), Genres: genres},
	}))
	d := "2026-07-12 14:00:00+00:00"
	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "p1", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), PlaybackDuration: i64(3600), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p2", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), PlaybackDuration: i64(3600), ActivityDateInserted: &d, Source: "watchdog"},
	}))
}

func statsHandler(t *testing.T) (*handler.StatsHandler, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	seedStatsService(t, db)
	repos := repository.New(db)
	svcs := &service.Container{Stats: statssvc.New(repos)}
	gin.SetMode(gin.TestMode)
	return handler.NewStatsHandler(svcs), gin.New()
}

func TestStatsHandler_GlobalStats(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/g", h.GlobalStats)
	w := getOK(t, r, "/g")

	var got struct {
		TotalPlays     int64 `json:"totalPlays"`
		TotalUsers     int64 `json:"totalUsers"`
		TotalLibraries int64 `json:"totalLibraries"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TotalPlays != 2 || got.TotalUsers != 1 || got.TotalLibraries != 1 {
		t.Errorf("got = %+v, want plays 2 users 1 libs 1", got)
	}
}

func TestStatsHandler_TopUsers(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/tu", h.TopUsers)
	w := getOK(t, r, "/tu?limit=5")

	var rows []struct {
		UserId     string  `json:"userId"`
		TotalPlays int64   `json:"totalPlays"`
		TotalHours float64 `json:"totalHours"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	// 2 plays × 3600s = 7200s = 2.0 hours.
	if rows[0].UserId != "u1" || rows[0].TotalPlays != 2 || rows[0].TotalHours != 2.0 {
		t.Errorf("row = %+v, want u1 plays 2 hours 2.0", rows[0])
	}
}

func TestStatsHandler_LibraryStats(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/ls/:id", h.LibraryStats)
	w := getOK(t, r, "/ls/lib-movies")
	// Shape check: valid JSON object.
	var got map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
}

func TestStatsHandler_MostPlayedItems(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/mpi/:id/items", h.MostPlayedItems)
	w := getOK(t, r, "/mpi/lib-movies/items?limit=5")

	var rows []struct {
		Id string `json:"id"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 1 || rows[0].Id != "m1" {
		t.Errorf("items = %+v, want just m1", rows)
	}
}

func TestStatsHandler_UserHistory(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/uh/:id/history", h.UserHistory)
	w := getOK(t, r, "/uh/u1/history?page=1&pageSize=10")

	var got struct {
		Total    int64 `json:"total"`
		Page     int   `json:"page"`
		PageSize int   `json:"pageSize"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Alice has 2 playback rows.
	if got.Total != 2 || got.Page != 1 || got.PageSize != 10 {
		t.Errorf("got = %+v, want total 2 page 1 size 10", got)
	}
}

func TestStatsHandler_ActivityOverTime(t *testing.T) {
	h, r := statsHandler(t)
	r.GET("/act", h.ActivityOverTime)
	w := getOK(t, r, "/act?days=0")
	var rows []struct {
		Date  string `json:"date"`
		Count int64  `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

// ---------------------------------------------------------------------------
// TasksApiHandler.List / Run
// ---------------------------------------------------------------------------

func tasksHandler(t *testing.T) (*handler.TasksApiHandler, *repository.Container, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	svcs := &service.Container{Task: tasksvc.New(repos, nil)}
	gin.SetMode(gin.TestMode)
	return handler.NewTasksApiHandler(svcs, repos, db), repos, gin.New()
}

func TestTasksList(t *testing.T) {
	h, _, r := tasksHandler(t)
	r.GET("/tasks", h.List)
	w := getOK(t, r, "/tasks")

	var tasks []struct {
		Name        string  `json:"name"`
		DisplayName string  `json:"displayName"`
		Running     bool    `json:"running"`
		LastResult  *string `json:"lastResult"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &tasks); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Four canonical task entries in fixed order.
	if len(tasks) != 4 {
		t.Fatalf("tasks = %d, want 4", len(tasks))
	}
	if tasks[0].Name != "Full Jellyfin Sync" {
		t.Errorf("tasks[0].name = %q, want Full Jellyfin Sync", tasks[0].Name)
	}
	// Idle service → not running, no prior run recorded.
	if tasks[0].Running {
		t.Error("expected running=false on a fresh handler")
	}
}

func TestTasksList_ReportsLastRun(t *testing.T) {
	h, repos, r := tasksHandler(t)
	r.GET("/tasks", h.List)

	// Insert a completed log entry for the full-sync task; List should surface it.
	name := "Full Jellyfin Sync"
	res := "success"
	tr := "2026-07-12T10:00:00Z"
	if err := repos.Log.Insert(t.Context(), &models.JFLogging{
		Id: "log-1", Name: &name, Result: &res, TimeRun: &tr,
	}); err != nil {
		t.Fatalf("insert log: %v", err)
	}

	w := getOK(t, r, "/tasks")
	var tasks []struct {
		Name       string  `json:"name"`
		LastResult *string `json:"lastResult"`
		LastRun    *string `json:"lastRun"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tasks)
	var full *struct {
		Name       string  `json:"name"`
		LastResult *string `json:"lastResult"`
		LastRun    *string `json:"lastRun"`
	}
	for i := range tasks {
		if tasks[i].Name == "Full Jellyfin Sync" {
			full = &tasks[i]
		}
	}
	if full == nil {
		t.Fatalf("Full Jellyfin Sync task missing from %+v", tasks)
	}
	if full.LastResult == nil || *full.LastResult != "success" {
		t.Errorf("lastResult = %v, want success", full.LastResult)
	}
}

func TestTasksRun_UnknownTask(t *testing.T) {
	h, _, r := tasksHandler(t)
	r.POST("/run/:name", h.Run)
	w := postJSON(t, r, "/run/DoesNotExist", `{}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404 for unknown task", w.Code)
	}
	var got struct {
		Error string `json:"error"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Error == "" {
		t.Error("expected error message for unknown task")
	}
}

// Note: the Run handler dispatches recognized task names to background
// goroutines that invoke the Sync service (nil in this lightweight harness), so
// only the synchronous unknown-task branch is exercised here to avoid standing
// up the full Jellyfin sync dependency graph.
