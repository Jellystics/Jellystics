package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	webhooksvc "github.com/Jellystics/Jellystics/internal/service/webhook"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// postJSON issues a POST with a JSON body and returns the recorder.
func postJSON(t *testing.T, r *gin.Engine, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, target, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// deleteJSON issues a DELETE with a JSON body and returns the recorder.
func deleteJSON(t *testing.T, r *gin.Engine, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodDelete, target, jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w
}

// configHandler builds a ConfigApiHandler with a webhook-backed service container.
func configHandler(t *testing.T) (*handler.ConfigApiHandler, *repository.Container, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	svcs := &service.Container{Webhook: webhooksvc.New(repos)}
	gin.SetMode(gin.TestMode)
	return handler.NewConfigApiHandler(repos, svcs), repos, gin.New()
}

// ─── config_api.go ─────────────────────────────────────────────────────────

func TestGetConfig_Defaults(t *testing.T) {
	h, _, r := configHandler(t)
	r.GET("/api/getconfig", h.GetConfig)
	w := do(t, r, "/api/getconfig")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		JFHost   string                 `json:"JF_HOST"`
		Settings map[string]interface{} `json:"settings"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.JFHost != "" {
		t.Errorf("JF_HOST = %q, want empty", got.JFHost)
	}
}

func TestSetConfig_PersistsHostAndKeepLogs(t *testing.T) {
	h, repos, r := configHandler(t)
	r.POST("/api/setconfig", h.SetConfig)

	w := postJSON(t, r, "/api/setconfig", `{"JF_HOST":"http://jf","JF_API_KEY":"abc","KeepLogsForDays":7,"app_url":"http://app"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	cfg, _ := repos.Config.Get(t.Context())
	if cfg.JFHost == nil || *cfg.JFHost != "http://jf" {
		t.Errorf("JFHost = %v, want http://jf", cfg.JFHost)
	}
	if cfg.AppUrl != "http://app" {
		t.Errorf("AppUrl = %q, want http://app", cfg.AppUrl)
	}
	var settings map[string]interface{}
	_ = json.Unmarshal(cfg.Settings, &settings)
	if settings["KeepLogsForDays"] != float64(7) {
		t.Errorf("KeepLogsForDays = %v, want 7", settings["KeepLogsForDays"])
	}
}

func TestApiKeys_CreateListDelete(t *testing.T) {
	h, _, r := configHandler(t)
	r.GET("/api/keys", h.GetKeys)
	r.POST("/api/keys", h.CreateKey)
	r.DELETE("/api/keys", h.DeleteKey)

	// Initially empty.
	w := do(t, r, "/api/keys")
	var keys []map[string]string
	_ = json.Unmarshal(w.Body.Bytes(), &keys)
	if len(keys) != 0 {
		t.Fatalf("initial keys = %d, want 0", len(keys))
	}

	// Create.
	cw := postJSON(t, r, "/api/keys", `{"name":"prod"}`)
	if cw.Code != http.StatusOK {
		t.Fatalf("create status = %d, body %s", cw.Code, cw.Body.String())
	}
	var created map[string]string
	_ = json.Unmarshal(cw.Body.Bytes(), &created)
	if created["name"] != "prod" || created["key"] == "" {
		t.Fatalf("created = %v, want name prod and a key", created)
	}

	// List shows the new key.
	lw := do(t, r, "/api/keys")
	_ = json.Unmarshal(lw.Body.Bytes(), &keys)
	if len(keys) != 1 || keys[0]["name"] != "prod" {
		t.Fatalf("keys = %v, want one prod key", keys)
	}

	// Delete it.
	dw := deleteJSON(t, r, "/api/keys", `{"key":"`+created["key"]+`"}`)
	if dw.Code != http.StatusOK {
		t.Fatalf("delete status = %d", dw.Code)
	}
	lw2 := do(t, r, "/api/keys")
	_ = json.Unmarshal(lw2.Body.Bytes(), &keys)
	if len(keys) != 0 {
		t.Errorf("keys after delete = %d, want 0", len(keys))
	}
}

func TestCreateKey_MissingName(t *testing.T) {
	h, _, r := configHandler(t)
	r.POST("/api/keys", h.CreateKey)
	w := postJSON(t, r, "/api/keys", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestTaskSettings_GetAndSet(t *testing.T) {
	h, _, r := configHandler(t)
	r.GET("/api/getTaskSettings", h.GetTaskSettings)
	r.POST("/api/setTaskSettings", h.SetTaskSettings)

	sw := postJSON(t, r, "/api/setTaskSettings", `{"taskname":"JellyfinSync","cronExpression":"0 * * * *","enabled":true}`)
	if sw.Code != http.StatusOK {
		t.Fatalf("set status = %d, body %s", sw.Code, sw.Body.String())
	}
	gw := do(t, r, "/api/getTaskSettings")
	var tasks map[string]interface{}
	_ = json.Unmarshal(gw.Body.Bytes(), &tasks)
	entry, ok := tasks["JellyfinSync"].(map[string]interface{})
	if !ok {
		t.Fatalf("tasks missing JellyfinSync: %v", tasks)
	}
	if entry["cronExpression"] != "0 * * * *" || entry["enabled"] != true {
		t.Errorf("entry = %v, want cron set and enabled", entry)
	}
}

func TestSetTaskSettings_MissingName(t *testing.T) {
	h, _, r := configHandler(t)
	r.POST("/api/setTaskSettings", h.SetTaskSettings)
	w := postJSON(t, r, "/api/setTaskSettings", `{"enabled":true}`)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
}

func TestSetExternalUrl(t *testing.T) {
	h, _, r := configHandler(t)
	r.POST("/api/setExternalUrl", h.SetExternalUrl)

	if w := postJSON(t, r, "/api/setExternalUrl", `{"ExternalUrl":""}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty url status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/setExternalUrl", `{"ExternalUrl":"http://ext"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		Settings map[string]interface{} `json:"settings"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Settings["EXTERNAL_URL"] != "http://ext" {
		t.Errorf("EXTERNAL_URL = %v, want http://ext", got.Settings["EXTERNAL_URL"])
	}
}

func TestSetRequireLogin(t *testing.T) {
	h, repos, r := configHandler(t)
	r.POST("/api/setRequireLogin", h.SetRequireLogin)

	if w := postJSON(t, r, "/api/setRequireLogin", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing value status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/setRequireLogin", `{"REQUIRE_LOGIN":true}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	cfg, _ := repos.Config.Get(t.Context())
	if !cfg.RequireLogin {
		t.Error("RequireLogin should be true")
	}
}

func TestUpdateCredentials_Password(t *testing.T) {
	h, repos, r := configHandler(t)
	// Seed an initial password.
	cfg, _ := repos.Config.Get(t.Context())
	cfg.AppUser = sp("admin")
	cfg.AppPassword = sp("oldpw")
	_ = repos.Config.Save(t.Context(), cfg)

	r.POST("/api/updateCredentials", h.UpdateCredentials)

	// Wrong current password.
	if w := postJSON(t, r, "/api/updateCredentials", `{"current_password":"wrong","new_password":"newpw"}`); w.Code != http.StatusBadRequest {
		t.Errorf("wrong pw status = %d, want 400", w.Code)
	}
	// Correct change.
	w := postJSON(t, r, "/api/updateCredentials", `{"current_password":"oldpw","new_password":"newpw"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	cfg2, _ := repos.Config.Get(t.Context())
	if cfg2.AppPassword == nil || *cfg2.AppPassword != "newpw" {
		t.Errorf("password = %v, want newpw", cfg2.AppPassword)
	}
}

func TestTrackedLibrariesAndExclude(t *testing.T) {
	h, repos, r := configHandler(t)
	if err := repos.Library.Upsert(t.Context(), []models.JFLibrary{
		{Id: "lib-a", Name: sp("A")},
		{Id: "lib-b", Name: sp("B")},
	}); err != nil {
		t.Fatalf("seed libs: %v", err)
	}
	r.GET("/api/TrackedLibraries", h.TrackedLibraries)
	r.POST("/api/setExcludedLibraries", h.SetExcludedLibraries)

	// Initially both tracked.
	w := do(t, r, "/api/TrackedLibraries")
	var libs []struct {
		Id      string `json:"Id"`
		Tracked bool   `json:"Tracked"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &libs)
	tracked := map[string]bool{}
	for _, l := range libs {
		tracked[l.Id] = l.Tracked
	}
	if !tracked["lib-a"] || !tracked["lib-b"] {
		t.Fatalf("both libs should be tracked, got %v", tracked)
	}

	// Exclude lib-a.
	if ew := postJSON(t, r, "/api/setExcludedLibraries", `{"libraryID":"lib-a"}`); ew.Code != http.StatusOK {
		t.Fatalf("exclude status = %d", ew.Code)
	}
	w2 := do(t, r, "/api/TrackedLibraries")
	_ = json.Unmarshal(w2.Body.Bytes(), &libs)
	for _, l := range libs {
		tracked[l.Id] = l.Tracked
	}
	if tracked["lib-a"] {
		t.Error("lib-a should be untracked after exclude")
	}
	if !tracked["lib-b"] {
		t.Error("lib-b should remain tracked")
	}
}

func TestUntrackedUsers_Toggle(t *testing.T) {
	h, _, r := configHandler(t)
	r.GET("/api/UntrackedUsers", h.UntrackedUsers)
	r.POST("/api/setUntrackedUsers", h.SetUntrackedUsers)

	// Empty initially.
	w := do(t, r, "/api/UntrackedUsers")
	var users []string
	_ = json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 0 {
		t.Fatalf("initial excluded users = %d, want 0", len(users))
	}

	// Reject array.
	if aw := postJSON(t, r, "/api/setUntrackedUsers", `{"userId":["u1"]}`); aw.Code != http.StatusBadRequest {
		t.Errorf("array userId status = %d, want 400", aw.Code)
	}

	// Add u1.
	sw := postJSON(t, r, "/api/setUntrackedUsers", `{"userId":"u1"}`)
	if sw.Code != http.StatusOK {
		t.Fatalf("status = %d", sw.Code)
	}
	_ = json.Unmarshal(sw.Body.Bytes(), &users)
	if len(users) != 1 || users[0] != "u1" {
		t.Errorf("excluded = %v, want [u1]", users)
	}
}

func TestActivityMonitorSettings(t *testing.T) {
	h, _, r := configHandler(t)
	r.GET("/api/getActivityMonitorSettings", h.GetActivityMonitorSettings)
	r.POST("/api/setActivityMonitorSettings", h.SetActivityMonitorSettings)

	// Default values.
	w := do(t, r, "/api/getActivityMonitorSettings")
	var got map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got["activeSessionsInterval"] != float64(1000) {
		t.Errorf("default activeSessionsInterval = %v, want 1000", got["activeSessionsInterval"])
	}

	// active > idle → 400.
	if bw := postJSON(t, r, "/api/setActivityMonitorSettings", `{"activeSessionsInterval":9000,"idleInterval":1000}`); bw.Code != http.StatusBadRequest {
		t.Errorf("active>idle status = %d, want 400", bw.Code)
	}
	// valid.
	sw := postJSON(t, r, "/api/setActivityMonitorSettings", `{"activeSessionsInterval":2000,"idleInterval":8000}`)
	if sw.Code != http.StatusOK {
		t.Fatalf("status = %d", sw.Code)
	}
	w2 := do(t, r, "/api/getActivityMonitorSettings")
	_ = json.Unmarshal(w2.Body.Bytes(), &got)
	if got["activeSessionsInterval"] != float64(2000) || got["idleInterval"] != float64(8000) {
		t.Errorf("settings = %v, want 2000/8000", got)
	}
}

func TestIsFirstRun(t *testing.T) {
	h, repos, r := configHandler(t)
	r.GET("/api/isFirstRun", h.IsFirstRun)

	// No sync log → first run.
	w := do(t, r, "/api/isFirstRun")
	var got map[string]bool
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if !got["firstRun"] {
		t.Error("expected firstRun true with no sync log")
	}

	// Add a successful Full Jellyfin Sync log → not first run.
	name := "Full Jellyfin Sync"
	res := "success"
	tr := "2026-07-12T10:00:00Z"
	_ = repos.Log.Insert(t.Context(), &models.JFLogging{Id: "l1", Name: &name, Result: &res, TimeRun: &tr})
	w2 := do(t, r, "/api/isFirstRun")
	_ = json.Unmarshal(w2.Body.Bytes(), &got)
	if got["firstRun"] {
		t.Error("expected firstRun false after successful sync log")
	}
}

// ─── admin_api.go ──────────────────────────────────────────────────────────

// seedAdmin inserts a library, movie item, series/season/episode and playback
// rows so admin read endpoints return deterministic data.
func seedAdmin(t *testing.T, db *gorm.DB) {
	t.Helper()
	repos := repository.New(db)
	ctx := t.Context()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seedAdmin: %v", err)
		}
	}
	movie := "Movie"
	series := "Series"
	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
	}))
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies")},
		{Id: "series-1", Name: sp("Lost"), Type: &series, ParentId: sp("lib-movies")},
	}))
	must(repos.User.Upsert(ctx, []models.JFUser{{Id: "u1", Name: sp("Alice")}}))

	date := "2026-07-12 10:00:00+00:00"
	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "p1", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"), PlaybackDuration: i64(600), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "p2", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"), PlaybackDuration: i64(300), ActivityDateInserted: &date, Source: "watchdog"},
	}))
}

func adminHandler(t *testing.T) (*handler.AdminApiHandler, *gin.Engine, *gorm.DB) {
	t.Helper()
	db := testutil.NewDB(t)
	seedAdmin(t, db)
	gin.SetMode(gin.TestMode)
	return handler.NewAdminApiHandler(repository.New(db), db), gin.New(), db
}

func TestAdminGetLibraries(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.GET("/api/getLibraries", h.GetLibraries)
	w := do(t, r, "/api/getLibraries")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var libs []struct {
		Id string `json:"Id"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &libs)
	if len(libs) != 1 || libs[0].Id != "lib-movies" {
		t.Errorf("libs = %+v, want just lib-movies", libs)
	}
}

func TestAdminGetLibrary_MissingId(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getLibrary", h.GetLibrary)
	if w := postJSON(t, r, "/api/getLibrary", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/getLibrary", `{"libraryid":"lib-movies"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
}

func TestAdminGetLibraryItems(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getLibraryItems", h.GetLibraryItems)
	w := postJSON(t, r, "/api/getLibraryItems", `{"libraryid":"lib-movies"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var items []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &items)
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}
}

func TestAdminGetItemDetails_Movie(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getItemDetails", h.GetItemDetails)
	w := postJSON(t, r, "/api/getItemDetails", `{"Id":"m1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var rows []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	// 2 plays totalling 900 seconds.
	if rows[0]["times_played"] != float64(2) {
		t.Errorf("times_played = %v, want 2", rows[0]["times_played"])
	}
	if rows[0]["total_play_time"] != float64(900) {
		t.Errorf("total_play_time = %v, want 900", rows[0]["total_play_time"])
	}
}

func TestAdminGetItemDetails_NotFound(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getItemDetails", h.GetItemDetails)
	w := postJSON(t, r, "/api/getItemDetails", `{"Id":"ghost"}`)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

func TestAdminGetHistory(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.GET("/api/getHistory", h.GetHistory)
	w := do(t, r, "/api/getHistory")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		Results []map[string]interface{} `json:"results"`
		Pages   int                      `json:"pages"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Results) != 2 {
		t.Errorf("results = %d, want 2", len(got.Results))
	}
	if got.Pages != 1 {
		t.Errorf("pages = %d, want 1", got.Pages)
	}
}

func TestAdminGetUserHistory(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getUserHistory", h.GetUserHistory)
	if w := postJSON(t, r, "/api/getUserHistory", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing userid status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/getUserHistory", `{"userid":"u1"}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		Results []map[string]interface{} `json:"results"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if len(got.Results) != 2 {
		t.Errorf("results = %d, want 2", len(got.Results))
	}
}

func TestAdminDeletePlaybackActivity(t *testing.T) {
	h, r, db := adminHandler(t)
	r.POST("/api/deletePlaybackActivity", h.DeletePlaybackActivity)
	if w := postJSON(t, r, "/api/deletePlaybackActivity", `{"ids":[]}`); w.Code != http.StatusBadRequest {
		t.Errorf("empty ids status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/deletePlaybackActivity", `{"ids":["p1"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var count int64
	db.Raw(`SELECT COUNT(*) FROM jf_playback_activity`).Scan(&count)
	if count != 1 {
		t.Errorf("remaining rows = %d, want 1", count)
	}
}

func TestAdminGetBackupTables_Toggle(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.GET("/api/getBackupTables", h.GetBackupTables)
	r.POST("/api/setExcludedBackupTable", h.SetExcludedBackupTable)

	w := do(t, r, "/api/getBackupTables")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var tables []struct {
		Value    string `json:"value"`
		Excluded bool   `json:"Excluded"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &tables)
	if len(tables) != 8 {
		t.Fatalf("tables = %d, want 8", len(tables))
	}

	// Invalid table name.
	if bw := postJSON(t, r, "/api/setExcludedBackupTable", `{"table":"nope"}`); bw.Code != http.StatusBadRequest {
		t.Errorf("invalid table status = %d, want 400", bw.Code)
	}

	// Toggle jf_users into excluded.
	tw := postJSON(t, r, "/api/setExcludedBackupTable", `{"table":"jf_users"}`)
	if tw.Code != http.StatusOK {
		t.Fatalf("toggle status = %d", tw.Code)
	}
	_ = json.Unmarshal(tw.Body.Bytes(), &tables)
	excluded := map[string]bool{}
	for _, tb := range tables {
		excluded[tb.Value] = tb.Excluded
	}
	if !excluded["jf_users"] {
		t.Error("jf_users should be excluded after toggle")
	}
}

func TestAdminGetActivityTimeLine(t *testing.T) {
	h, r, _ := adminHandler(t)
	r.POST("/api/getActivityTimeLine", h.GetActivityTimeLine)

	if w := postJSON(t, r, "/api/getActivityTimeLine", `{"libraries":["lib-movies"]}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing userId status = %d, want 400", w.Code)
	}
	if w := postJSON(t, r, "/api/getActivityTimeLine", `{"userId":"u1"}`); w.Code != http.StatusBadRequest {
		t.Errorf("missing libraries status = %d, want 400", w.Code)
	}
	w := postJSON(t, r, "/api/getActivityTimeLine", `{"userId":"u1","libraries":["lib-movies"]}`)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var rows []struct {
		TotalPlays int64 `json:"total_plays"`
		TotalTime  int64 `json:"total_time"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 || rows[0].TotalPlays != 2 || rows[0].TotalTime != 900 {
		t.Errorf("timeline = %+v, want one row 2 plays 900s", rows)
	}
}
