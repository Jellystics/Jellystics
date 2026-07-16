package repository_test

import (
	"context"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/gorm"
)

// seedLibraryStats builds a movie library (2 movies, 3 plays) and a music
// library (1 album, 1 track, 2 plays) so library-level aggregates can be
// asserted exactly.
func seedLibraryStats(t *testing.T, db *gorm.DB) {
	t.Helper()
	repos := repository.New(db)
	ctx := context.Background()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: p("Movies"), CollectionType: p("movies")},
		{Id: "lib-music", Name: p("Music"), CollectionType: p("music")},
	}))

	movie := "Movie"
	album := "MusicAlbum"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: p("Inception"), Type: &movie, ParentId: p("lib-movies"), Genres: emptyGenres},
		{Id: "m2", Name: p("Tenet"), Type: &movie, ParentId: p("lib-movies"), Genres: emptyGenres},
		{Id: "alb-1", Name: p("Discovery"), Type: &album, ParentId: p("lib-music"), Genres: emptyGenres},
	}))
	must(repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "tr-1", Name: p("One More Time"), AlbumId: p("alb-1"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}))

	date := "2026-07-12 10:00:00+00:00"
	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		// 3 movie plays in lib-movies.
		{Id: "a1", NowPlayingItemId: p("m1"), PlaybackDuration: ptrI64(100), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "a2", NowPlayingItemId: p("m1"), PlaybackDuration: ptrI64(100), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "a3", NowPlayingItemId: p("m2"), PlaybackDuration: ptrI64(100), ActivityDateInserted: &date, Source: "watchdog"},
		// 2 music plays in lib-music (track counted via NowPlayingItemId = track id).
		{Id: "a4", NowPlayingItemId: p("tr-1"), EpisodeId: p("tr-1"), PlaybackDuration: ptrI64(200), ActivityDateInserted: &date, Source: "watchdog"},
		{Id: "a5", NowPlayingItemId: p("tr-1"), EpisodeId: p("tr-1"), PlaybackDuration: ptrI64(200), ActivityDateInserted: &date, Source: "watchdog"},
	}))
}

// TestGetMostViewedLibraries verifies play counts are grouped per library and
// ordered by total plays descending.
func TestGetMostViewedLibraries(t *testing.T) {
	db := setupTestDB(t)
	seedLibraryStats(t, db)

	got, err := repository.New(db).Stats.GetMostViewedLibraries(context.Background(), 10)
	if err != nil {
		t.Fatalf("GetMostViewedLibraries: %v", err)
	}
	counts := map[string]int64{}
	for _, r := range got {
		counts[r.Id] = r.TotalPlays
	}
	if counts["lib-movies"] != 3 {
		t.Errorf("lib-movies plays = %d, want 3", counts["lib-movies"])
	}
	if counts["lib-music"] != 2 {
		t.Errorf("lib-music plays = %d, want 2", counts["lib-music"])
	}
	// Ordered descending: movies (3) before music (2).
	if len(got) >= 2 && got[0].TotalPlays < got[1].TotalPlays {
		t.Errorf("results not ordered by plays desc: %+v", got)
	}
}

// TestGetLibraryStats verifies per-library item/play aggregates.
func TestGetLibraryStats(t *testing.T) {
	db := setupTestDB(t)
	seedLibraryStats(t, db)
	repos := repository.New(db)
	ctx := context.Background()

	movies, err := repos.Stats.GetLibraryStats(ctx, "lib-movies")
	if err != nil {
		t.Fatalf("GetLibraryStats movies: %v", err)
	}
	if movies.TotalItems != 2 {
		t.Errorf("movies TotalItems = %d, want 2", movies.TotalItems)
	}
	if movies.TotalPlays != 3 {
		t.Errorf("movies TotalPlays = %d, want 3", movies.TotalPlays)
	}

	music, err := repos.Stats.GetLibraryStats(ctx, "lib-music")
	if err != nil {
		t.Fatalf("GetLibraryStats music: %v", err)
	}
	// One track counts as an item; the album is a Folder-like container but
	// MusicAlbum is not excluded, so TotalItems includes the album + track.
	if music.TotalPlays != 2 {
		t.Errorf("music TotalPlays = %d, want 2", music.TotalPlays)
	}
}

// TestRefreshViews verifies the materialized-view refresh procedure runs without
// error against a migrated schema.
func TestRefreshViews(t *testing.T) {
	db := setupTestDB(t)
	seedLibraryStats(t, db)
	if err := repository.New(db).Stats.RefreshViews(context.Background()); err != nil {
		t.Fatalf("RefreshViews: %v", err)
	}
}
