package handler_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// seedStats builds a deterministic dataset: one movie library with one movie
// played twice by Alice (Web/DirectPlay, hour 14) and once by Bob
// (Android/Transcode, hour 09), all on the same day. With no Jellyfin host
// configured, live-session merging is a no-op, so results are pure DB output.
func seedStats(t *testing.T, db *gorm.DB) {
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

	must(repos.User.Upsert(ctx, []models.JFUser{
		{Id: "u1", Name: sp("Alice")},
		{Id: "u2", Name: sp("Bob")},
	}))
	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
	}))
	movie := "Movie"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"), Genres: genres},
	}))

	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "r1", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"), Client: sp("Web"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: sp("2026-07-12 14:00:00+00:00"), Source: "watchdog"},
		{Id: "r2", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"), Client: sp("Web"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: sp("2026-07-12 14:30:00+00:00"), Source: "watchdog"},
		{Id: "r3", UserId: sp("u2"), UserName: sp("Bob"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"), Client: sp("Android"), PlayMethod: sp("Transcode"), PlaybackDuration: i64(300), ActivityDateInserted: sp("2026-07-12 09:00:00+00:00"), Source: "watchdog"},
	}))
}

// serve registers one handler at path and returns the recorded response.
func serve(t *testing.T, register func(*gin.Engine, *handler.StatsFrontendHandler), method, target string) *httptest.ResponseRecorder {
	t.Helper()
	db := testutil.NewDB(t)
	seedStats(t, db)
	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	register(r, h)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(method, target, nil))
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	return w
}

func TestGetGlobalStats(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/g", h.GetGlobalStats)
	}, http.MethodGet, "/g")

	var got struct {
		TotalPlays     int `json:"TotalPlays"`
		TotalWatchTime int `json:"TotalWatchTime"`
		TotalUsers     int `json:"TotalUsers"`
		TotalLibraries int `json:"TotalLibraries"`
		TotalItems     int `json:"TotalItems"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.TotalPlays != 3 {
		t.Errorf("TotalPlays = %d, want 3", got.TotalPlays)
	}
	if got.TotalWatchTime != 25 { // floor(1500s / 60) = 25 min
		t.Errorf("TotalWatchTime = %d, want 25 min", got.TotalWatchTime)
	}
	if got.TotalUsers != 2 {
		t.Errorf("TotalUsers = %d, want 2", got.TotalUsers)
	}
	if got.TotalLibraries != 1 {
		t.Errorf("TotalLibraries = %d, want 1", got.TotalLibraries)
	}
	if got.TotalItems != 1 {
		t.Errorf("TotalItems = %d, want 1", got.TotalItems)
	}
}

func TestGetMostPlayedItems(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/i", h.GetMostPlayedItems)
	}, http.MethodGet, "/i?days=0&type=all")

	var items []struct {
		Id        string `json:"Id"`
		Name      string `json:"Name"`
		PlayCount int    `json:"PlayCount"`
		Type      string `json:"Type"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("items = %d, want 1", len(items))
	}
	if items[0].Id != "m1" || items[0].PlayCount != 3 || items[0].Type != "Movie" {
		t.Errorf("item = %+v, want m1 PlayCount 3 Movie", items[0])
	}
}

func TestGetMostActiveUsers(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/u", h.GetMostActiveUsers)
	}, http.MethodGet, "/u?days=0")

	var rows []struct {
		UserId         string `json:"UserId"`
		UserName       string `json:"UserName"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	// Ordered by plays desc → Alice (2) first.
	if rows[0].UserId != "u1" || rows[0].TotalPlays != 2 || rows[0].TotalWatchTime != 20 {
		t.Errorf("rows[0] = %+v, want u1 plays 2 watch 20", rows[0])
	}
	if rows[1].UserId != "u2" || rows[1].TotalPlays != 1 || rows[1].TotalWatchTime != 5 {
		t.Errorf("rows[1] = %+v, want u2 plays 1 watch 5", rows[1])
	}
}

func TestGetPopularHourOfDay(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/h", h.GetPopularHourOfDay)
	}, http.MethodGet, "/h?days=0")

	var rows []struct {
		Hour  int `json:"hour"`
		Plays int `json:"plays"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byHour := map[int]int{}
	for _, r := range rows {
		byHour[r.Hour] = r.Plays
	}
	if byHour[14] != 2 {
		t.Errorf("hour 14 plays = %d, want 2", byHour[14])
	}
	if byHour[9] != 1 {
		t.Errorf("hour 9 plays = %d, want 1", byHour[9])
	}
}

func TestGetPopularDayOfWeek(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/d", h.GetPopularDayOfWeek)
	}, http.MethodGet, "/d?days=0")

	var rows []struct {
		Day   int `json:"day"`
		Plays int `json:"plays"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// All three plays fall on the same calendar day → one bucket of 3.
	if len(rows) != 1 || rows[0].Plays != 3 {
		t.Errorf("rows = %+v, want a single day with 3 plays", rows)
	}
}

func TestGetMostUsedPlaybackMethod(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/pm", h.GetMostUsedPlaybackMethod)
	}, http.MethodGet, "/pm?days=0")

	var rows []struct {
		Method string `json:"method"`
		Count  int    `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	byMethod := map[string]int{}
	for _, r := range rows {
		byMethod[r.Method] = r.Count
	}
	if byMethod["DirectPlay"] != 2 {
		t.Errorf("DirectPlay = %d, want 2", byMethod["DirectPlay"])
	}
	if byMethod["Transcode"] != 1 {
		t.Errorf("Transcode = %d, want 1", byMethod["Transcode"])
	}
}

func TestGetMostUsedClients(t *testing.T) {
	w := serve(t, func(r *gin.Engine, h *handler.StatsFrontendHandler) {
		r.GET("/cl", h.GetMostUsedClients)
	}, http.MethodGet, "/cl?days=0")

	var rows []struct {
		Client string `json:"client"`
		Count  int    `json:"count"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) == 0 || rows[0].Client != "Web" || rows[0].Count != 2 {
		t.Errorf("rows[0] = %+v, want Web count 2 first", rows)
	}
}
