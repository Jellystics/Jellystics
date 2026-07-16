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

func sp(s string) *string { return &s }
func i64(v int64) *int64  { return &v }

const secTicks = 10_000_000 // ticks per second

// seedCompletion inserts one Movie play (fully watched), one Movie play
// (abandoned), one Episode play, and one music Track play. Runtimes are chosen
// so completion ratios are exact. The music play is the regression guard: it
// must be counted and classified as "Audio", not silently dropped.
func seedCompletion(t *testing.T, db *gorm.DB) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	genres := datatypes.JSON([]byte("[]"))

	movie := "Movie"
	must(db.Create(&models.JFLibraryItem{
		Id: "movie-1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"),
		RunTimeTicks: i64(100 * secTicks), Genres: genres,
	}).Error)

	// Episode with a 100s runtime.
	must(db.Create(&models.JFLibraryEpisode{
		Id: "ep-1", Name: sp("Pilot"), RunTimeTicks: i64(100 * secTicks), SeriesId: sp("series-1"),
	}).Error)

	// Music track with a 200s runtime.
	must(db.Create(&models.JFMusicTrack{
		Id: "track-1", Name: sp("One More Time"), AlbumId: sp("album-1"),
		RunTimeTicks: i64(200 * secTicks), LibraryId: sp("lib-music"), Genres: genres,
	}).Error)

	date := "2026-07-12 10:00:00+00:00"
	must(db.Create(&[]models.JFPlaybackActivity{
		// Movie fully watched: 100/100 = 1.0.
		{Id: "c1", UserId: sp("u1"), NowPlayingItemId: sp("movie-1"), PlaybackDuration: i64(100), ActivityDateInserted: &date, Source: "watchdog"},
		// Movie abandoned: 25/100 = 0.25.
		{Id: "c2", UserId: sp("u1"), NowPlayingItemId: sp("movie-1"), PlaybackDuration: i64(25), ActivityDateInserted: &date, Source: "watchdog"},
		// Episode: 80/100 = 0.8.
		{Id: "c3", UserId: sp("u1"), NowPlayingItemId: sp("series-1"), EpisodeId: sp("ep-1"), PlaybackDuration: i64(80), ActivityDateInserted: &date, Source: "watchdog"},
		// Audio: 200/200 = 1.0 (must be included and labelled Audio).
		{Id: "c4", UserId: sp("u1"), NowPlayingItemId: sp("album-1"), EpisodeId: sp("track-1"), PlaybackDuration: i64(200), ActivityDateInserted: &date, Source: "watchdog"},
	}).Error)
}

type completionResponse struct {
	Overall struct {
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
		CompletedPlays    int     `json:"completedPlays"`
		AbandonedPlays    int     `json:"abandonedPlays"`
	} `json:"overall"`
	ByType []struct {
		Type              string  `json:"type"`
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
	} `json:"byType"`
}

func TestGetCompletionRate_IncludesAudio(t *testing.T) {
	db := testutil.NewDB(t)
	seedCompletion(t, db)

	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	r.GET("/completion", h.GetCompletionRate)

	req := httptest.NewRequest(http.MethodGet, "/completion?days=0", nil) // days=0 → all time
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	var resp completionResponse
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}

	// All four rows (including the music track) must be counted.
	if resp.Overall.TotalPlays != 4 {
		t.Errorf("overall totalPlays = %d, want 4 (audio must not be dropped)", resp.Overall.TotalPlays)
	}
	// 3 completed (>=0.75): movie 1.0, episode 0.8, audio 1.0. 1 abandoned: movie 0.25.
	if resp.Overall.CompletedPlays != 3 {
		t.Errorf("completedPlays = %d, want 3", resp.Overall.CompletedPlays)
	}
	if resp.Overall.AbandonedPlays != 1 {
		t.Errorf("abandonedPlays = %d, want 1", resp.Overall.AbandonedPlays)
	}

	byType := map[string]struct {
		avg   float64
		plays int
	}{}
	for _, r := range resp.ByType {
		byType[r.Type] = struct {
			avg   float64
			plays int
		}{r.AvgCompletionRate, r.TotalPlays}
	}

	// The fix: music plays are classified as Audio and included.
	audio, ok := byType["Audio"]
	if !ok {
		t.Fatalf("byType missing 'Audio' — music completion is not counted; got %+v", resp.ByType)
	}
	if audio.plays != 1 || audio.avg != 1.0 {
		t.Errorf("Audio = {plays:%d avg:%v}, want {1, 1.0}", audio.plays, audio.avg)
	}
	if m := byType["Movie"]; m.plays != 2 || m.avg != 0.63 { // (1.0+0.25)/2 = 0.625 → rounded 0.63
		t.Errorf("Movie = {plays:%d avg:%v}, want {2, 0.63}", m.plays, m.avg)
	}
	if e := byType["Episode"]; e.plays != 1 || e.avg != 0.8 {
		t.Errorf("Episode = {plays:%d avg:%v}, want {1, 0.8}", e.plays, e.avg)
	}
}
