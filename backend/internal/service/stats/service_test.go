package stats_test

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/stats"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func p(s string) *string    { return &s }
func i64(v int64) *int64     { return &v }
var emptyGenres = datatypes.JSON([]byte("[]"))

// seed builds a minimal but complete dataset covering movies and music so every
// stats method has real rows to delegate to:
//
//	users:      user-1 (Alice)
//	libraries:  lib-movies (movies), lib-music (music)
//	items:      movie-1 (Movie), album-1 (MusicAlbum)
//	artists:    artist-1
//	tracks:     track-1 (album-1 / artist-1)
//	playback:   3 movie plays + 1 music play (PlaybackDuration in SECONDS)
//
// Music mapping mirrors production: NowPlayingItemId=AlbumId, EpisodeId=trackId.
func seed(t *testing.T, db *gorm.DB) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	must(db.Create(&[]models.JFUser{{Id: "user-1", Name: p("Alice")}}).Error)

	movies, music := "movies", "music"
	must(db.Create(&[]models.JFLibrary{
		{Id: "lib-movies", Name: p("Movies"), CollectionType: &movies},
		{Id: "lib-music", Name: p("Music"), CollectionType: &music},
	}).Error)

	movie, album := "Movie", "MusicAlbum"
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "movie-1", Name: p("Inception"), Type: &movie, ParentId: p("lib-movies"), Genres: emptyGenres},
		{Id: "album-1", Name: p("Discovery"), Type: &album, ParentId: p("lib-music"), Genres: emptyGenres},
	}).Error)

	must(db.Create(&[]models.JFMusicArtist{
		{Id: "artist-1", Name: p("Daft Punk"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}).Error)
	must(db.Create(&[]models.JFMusicTrack{
		{Id: "track-1", Name: p("One More Time"), AlbumId: p("album-1"), AlbumName: p("Discovery"), ArtistId: p("artist-1"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}).Error)

	ts := func(agoMinutes int) *string {
		s := time.Now().UTC().Add(-time.Duration(agoMinutes) * time.Minute).Format("2006-01-02 15:04:05-07:00")
		return &s
	}
	must(db.Create(&[]models.JFPlaybackActivity{
		// user-1 movie plays: 3600 + 1800 + 3600 = 9000s => 2.5h
		{Id: "a1", UserId: p("user-1"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: i64(3600), ActivityDateInserted: ts(10), Source: "watchdog"},
		{Id: "a2", UserId: p("user-1"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: i64(1800), ActivityDateInserted: ts(20), Source: "watchdog"},
		{Id: "a3", UserId: p("user-1"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: i64(3600), ActivityDateInserted: ts(30), Source: "watchdog"},
		// music play: album=NowPlayingItemId, track=EpisodeId
		{Id: "a4", UserId: p("user-1"), NowPlayingItemId: p("album-1"), NowPlayingItemName: p("Discovery"), EpisodeId: p("track-1"), PlaybackDuration: i64(600), ActivityDateInserted: ts(5), Source: "watchdog"},
	}).Error)
}

func newSvc(t *testing.T) (*stats.Service, *gorm.DB) {
	db := testutil.NewDB(t)
	seed(t, db)
	return stats.New(repository.New(db)), db
}

// TestGlobalStats verifies delegation. GlobalStats takes no limit args.
func TestGlobalStats(t *testing.T) {
	svc, _ := newSvc(t)
	got, err := svc.GlobalStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalPlays != 4 {
		t.Errorf("TotalPlays = %d, want 4", got.TotalPlays)
	}
	if got.TotalUsers != 1 {
		t.Errorf("TotalUsers = %d, want 1", got.TotalUsers)
	}
	if got.TotalLibraries != 2 {
		t.Errorf("TotalLibraries = %d, want 2", got.TotalLibraries)
	}
}

// TestLibraryStats verifies delegation with a library id (no default logic).
func TestLibraryStats(t *testing.T) {
	svc, _ := newSvc(t)
	got, err := svc.LibraryStats(context.Background(), "lib-movies")
	if err != nil {
		t.Fatal(err)
	}
	if got.Id != "lib-movies" {
		t.Errorf("Id = %q, want lib-movies", got.Id)
	}
}

// TestMostViewedLibraries_DefaultLimit proves limit<=0 substitutes 10 and still
// returns rows (the default should not zero-out the result set).
func TestMostViewedLibraries_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -5} {
		got, err := svc.MostViewedLibraries(context.Background(), limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) == 0 {
			t.Errorf("limit %d: got 0 libraries, default should not clamp to empty", limit)
		}
	}
	// Explicit positive limit is honoured.
	got, err := svc.MostViewedLibraries(context.Background(), 1)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 {
		t.Errorf("limit 1: len = %d, want 1", len(got))
	}
}

// TestActivityOverTime_DefaultDays proves days<=0 substitutes 90. All seeded
// plays are recent so both default and explicit windows include them.
func TestActivityOverTime_DefaultDays(t *testing.T) {
	svc, _ := newSvc(t)
	sum := func(days int) int64 {
		got, err := svc.ActivityOverTime(context.Background(), days)
		if err != nil {
			t.Fatalf("days %d: %v", days, err)
		}
		var total int64
		for _, d := range got {
			total += d.Count
		}
		return total
	}
	// days<=0 -> default 90 window contains all 4 plays.
	if got := sum(0); got != 4 {
		t.Errorf("days 0 (default 90): count = %d, want 4", got)
	}
	if got := sum(-1); got != 4 {
		t.Errorf("days -1 (default 90): count = %d, want 4", got)
	}
	if got := sum(90); got != 4 {
		t.Errorf("days 90: count = %d, want 4", got)
	}
}

// TestTopUsers_DefaultLimit proves limit<=0 substitutes 10 and hours=sec/3600.
func TestTopUsers_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -3, 10} {
		got, err := svc.TopUsers(context.Background(), limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) != 1 {
			t.Fatalf("limit %d: users = %d, want 1", limit, len(got))
		}
		if got[0].UserId != "user-1" {
			t.Errorf("limit %d: top user = %s, want user-1", limit, got[0].UserId)
		}
		// 9000 (movies) + 600 (music) = 9600s => 2.666..h
		wantHours := 9600.0 / 3600.0
		if got[0].TotalHours != wantHours {
			t.Errorf("limit %d: hours = %v, want %v (sec/3600)", limit, got[0].TotalHours, wantHours)
		}
	}
}

// TestMostPlayedItems_DefaultLimit proves limit<=0 substitutes 10.
func TestMostPlayedItems_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -2} {
		got, err := svc.MostPlayedItems(context.Background(), "lib-movies", limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) == 0 {
			t.Errorf("limit %d: got 0 items, default should not clamp to empty", limit)
		}
		if got[0].Id != "movie-1" {
			t.Errorf("limit %d: top item = %s, want movie-1", limit, got[0].Id)
		}
	}
}

// TestMostPlayedArtists_DefaultLimit proves limit<=0 substitutes 10.
func TestMostPlayedArtists_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -1} {
		got, err := svc.MostPlayedArtists(context.Background(), "lib-music", limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) == 0 {
			t.Errorf("limit %d: got 0 artists, default should not clamp to empty", limit)
		}
	}
}

// TestMostPlayedAlbums_DefaultLimit proves limit<=0 substitutes 10.
func TestMostPlayedAlbums_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -4} {
		got, err := svc.MostPlayedAlbums(context.Background(), "lib-music", "artist-1", limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) == 0 {
			t.Errorf("limit %d: got 0 albums, default should not clamp to empty", limit)
		}
	}
}

// TestMostPlayedTracks_DefaultLimit proves limit<=0 substitutes 10. Track plays
// are counted via EpisodeId (a4), not NowPlayingItemId.
func TestMostPlayedTracks_DefaultLimit(t *testing.T) {
	svc, _ := newSvc(t)
	for _, limit := range []int{0, -7} {
		got, err := svc.MostPlayedTracks(context.Background(), "lib-music", "album-1", limit)
		if err != nil {
			t.Fatalf("limit %d: %v", limit, err)
		}
		if len(got) == 0 {
			t.Errorf("limit %d: got 0 tracks, default should not clamp to empty", limit)
		}
	}
}

// TestUserHistory_DefaultPaging proves page<=0 -> 1 and pageSize<=0 -> 20, while
// still delegating (returns the user's rows and the correct total).
func TestUserHistory_DefaultPaging(t *testing.T) {
	svc, _ := newSvc(t)
	ctx := context.Background()

	// Defaults: page 0 -> 1, pageSize 0 -> 20. All 4 user-1 rows fit in one page.
	got, total, err := svc.UserHistory(ctx, "user-1", 0, 0)
	if err != nil {
		t.Fatal(err)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
	if len(got) != 4 {
		t.Errorf("default page/size returned %d rows, want 4 (page1 size20)", len(got))
	}

	// Negative values also fall back to the defaults.
	got2, total2, err := svc.UserHistory(ctx, "user-1", -3, -9)
	if err != nil {
		t.Fatal(err)
	}
	if total2 != 4 || len(got2) != 4 {
		t.Errorf("negative page/size: total=%d rows=%d, want 4/4", total2, len(got2))
	}

	// Explicit small pageSize is honoured (proves the default is a substitution,
	// not a hard override).
	page1, _, err := svc.UserHistory(ctx, "user-1", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page1) != 2 {
		t.Errorf("explicit pageSize 2: len = %d, want 2", len(page1))
	}
}
