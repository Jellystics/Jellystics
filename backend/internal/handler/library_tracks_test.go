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
)

// TestGetLibraryTracks_PlayCount verifies each track's PlayCount comes from
// playback rows whose EpisodeId (the child/track id) matches — NOT the parent
// NowPlayingItemId (the album). A decoy row with NowPlayingItemId=track-1 must
// not inflate track-1's count.
func TestGetLibraryTracks_PlayCount(t *testing.T) {
	db := testutil.NewDB(t)
	genres := datatypes.JSON([]byte("[]"))

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	must(db.Create(&[]models.JFMusicTrack{
		{Id: "track-1", Name: sp("One More Time"), AlbumId: sp("album-1"), AlbumName: sp("Discovery"), AlbumArtist: sp("Daft Punk"), LibraryId: sp("lib-music"), Genres: genres},
		{Id: "track-2", Name: sp("Aerodynamic"), AlbumId: sp("album-1"), AlbumName: sp("Discovery"), AlbumArtist: sp("Daft Punk"), LibraryId: sp("lib-music"), Genres: genres},
	}).Error)

	date := "2026-07-12 10:00:00+00:00"
	must(db.Create(&[]models.JFPlaybackActivity{
		// track-1 played 3× (EpisodeId = track id, NowPlayingItemId = album).
		{Id: "p1", NowPlayingItemId: sp("album-1"), EpisodeId: sp("track-1"), PlaybackDuration: i64(180), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "p2", NowPlayingItemId: sp("album-1"), EpisodeId: sp("track-1"), PlaybackDuration: i64(180), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "p3", NowPlayingItemId: sp("album-1"), EpisodeId: sp("track-1"), PlaybackDuration: i64(180), ActivityDateInserted: &date, Source: "watchdog"},
		// Decoy: NowPlayingItemId equals track-1 but EpisodeId is empty — must NOT
		// count toward track-1 (would if the join used NowPlayingItemId).
		{Id: "decoy", NowPlayingItemId: sp("track-1"), PlaybackDuration: i64(180), ActivityDateInserted: &date, Source: "watchdog"},
	}).Error)

	gin.SetMode(gin.TestMode)
	h := handler.NewStatsFrontendHandler(db, repository.New(db))
	r := gin.New()
	r.GET("/tracks", h.GetLibraryTracks)

	req := httptest.NewRequest(http.MethodGet, "/tracks?libraryId=lib-music", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var rows []struct {
		Id        string `json:"Id"`
		PlayCount int    `json:"PlayCount"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	counts := map[string]int{}
	for _, r := range rows {
		counts[r.Id] = r.PlayCount
	}
	if counts["track-1"] != 3 {
		t.Errorf("track-1 PlayCount = %d, want 3 (counted by EpisodeId, decoy excluded)", counts["track-1"])
	}
	if counts["track-2"] != 0 {
		t.Errorf("track-2 PlayCount = %d, want 0", counts["track-2"])
	}
}
