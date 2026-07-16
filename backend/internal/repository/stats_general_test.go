package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/gorm"
)

// seedMovies builds a movie library with two users and four play events:
//
//	user-1 (Alice): movie-1 (1h), movie-1 (0.5h), movie-2 (1h)   → 3 plays, 2.5h
//	user-2 (Bob):   movie-1 (0.25h)                              → 1 play,  0.25h
//
// Timestamps are anchored to now() so date-window queries are stable.
func seedMovies(t *testing.T, db *gorm.DB) {
	t.Helper()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	must(db.Create(&[]models.JFUser{
		{Id: "user-1", Name: p("Alice")},
		{Id: "user-2", Name: p("Bob")},
	}).Error)

	collType := "movies"
	must(db.Create(&models.JFLibrary{Id: "lib-movies", Name: p("Movies"), CollectionType: &collType}).Error)

	movie := "Movie"
	must(db.Create(&[]models.JFLibraryItem{
		{Id: "movie-1", Name: p("Inception"), Type: &movie, ParentId: p("lib-movies"), Genres: emptyGenres},
		{Id: "movie-2", Name: p("Interstellar"), Type: &movie, ParentId: p("lib-movies"), Genres: emptyGenres},
	}).Error)

	ts := func(agoMinutes int) *string {
		s := time.Now().UTC().Add(-time.Duration(agoMinutes) * time.Minute).Format("2006-01-02 15:04:05-07:00")
		return &s
	}
	must(db.Create(&[]models.JFPlaybackActivity{
		{Id: "a1", UserId: p("user-1"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: ptrI64(3600), ActivityDateInserted: ts(10), Source: "watchdog"},
		{Id: "a2", UserId: p("user-1"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: ptrI64(1800), ActivityDateInserted: ts(20), Source: "watchdog"},
		{Id: "a3", UserId: p("user-1"), NowPlayingItemId: p("movie-2"), NowPlayingItemName: p("Interstellar"), PlaybackDuration: ptrI64(3600), ActivityDateInserted: ts(30), Source: "watchdog"},
		{Id: "a4", UserId: p("user-2"), NowPlayingItemId: p("movie-1"), NowPlayingItemName: p("Inception"), PlaybackDuration: ptrI64(900), ActivityDateInserted: ts(40), Source: "watchdog"},
	}).Error)
}

func TestGetGlobalStats(t *testing.T) {
	db := setupTestDB(t)
	seedMovies(t, db)
	repos := repository.New(db)

	got, err := repos.Stats.GetGlobalStats(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if got.TotalPlays != 4 {
		t.Errorf("TotalPlays = %d, want 4", got.TotalPlays)
	}
	if got.TotalDuration != 9900 { // 3600+1800+3600+900
		t.Errorf("TotalDuration = %d, want 9900 seconds", got.TotalDuration)
	}
	if got.TotalUsers != 2 {
		t.Errorf("TotalUsers = %d, want 2", got.TotalUsers)
	}
	if got.TotalLibraries != 1 {
		t.Errorf("TotalLibraries = %d, want 1", got.TotalLibraries)
	}
}

func TestGetMostPlayedItems(t *testing.T) {
	db := setupTestDB(t)
	seedMovies(t, db)
	repos := repository.New(db)

	got, err := repos.Stats.GetMostPlayedItems(context.Background(), "lib-movies", 10)
	if err != nil {
		t.Fatal(err)
	}
	byId := map[string]repository.ItemPlayStat{}
	for _, r := range got {
		byId[r.Id] = r
	}
	// movie-1 appears in 3 plays with 3600+1800+900 = 6300s of duration.
	if byId["movie-1"].TimesPlayed != 3 {
		t.Errorf("movie-1 plays = %d, want 3", byId["movie-1"].TimesPlayed)
	}
	if byId["movie-1"].TotalDuration != 6300 {
		t.Errorf("movie-1 duration = %d, want 6300", byId["movie-1"].TotalDuration)
	}
	if byId["movie-2"].TimesPlayed != 1 {
		t.Errorf("movie-2 plays = %d, want 1", byId["movie-2"].TimesPlayed)
	}
	// Ranked most-played first.
	if len(got) == 0 || got[0].Id != "movie-1" {
		t.Errorf("expected movie-1 first, got %+v", got)
	}
}

func TestGetTopUsers(t *testing.T) {
	db := setupTestDB(t)
	seedMovies(t, db)
	repos := repository.New(db)

	got, err := repos.Stats.GetTopUsers(context.Background(), 10)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 {
		t.Fatalf("want 2 users, got %d", len(got))
	}
	// Alice ranks first: 3 plays, 2.5h (9000s / 3600).
	if got[0].UserId != "user-1" {
		t.Fatalf("top user = %s, want user-1", got[0].UserId)
	}
	if got[0].UserName != "Alice" {
		t.Errorf("user name = %q, want Alice", got[0].UserName)
	}
	if got[0].TotalPlays != 3 {
		t.Errorf("Alice plays = %d, want 3", got[0].TotalPlays)
	}
	if got[0].TotalHours != 2.5 {
		t.Errorf("Alice hours = %v, want 2.5", got[0].TotalHours)
	}
}

func TestGetActivityOverTime(t *testing.T) {
	db := setupTestDB(t)
	seedMovies(t, db)
	repos := repository.New(db)

	got, err := repos.Stats.GetActivityOverTime(context.Background(), 30)
	if err != nil {
		t.Fatal(err)
	}
	var total int64
	for _, d := range got {
		total += d.Count
	}
	// All four plays fall within the last 30 days.
	if total != 4 {
		t.Errorf("sum of daily counts = %d, want 4", total)
	}
}

func TestGetUserHistory_Pagination(t *testing.T) {
	db := setupTestDB(t)
	seedMovies(t, db)
	repos := repository.New(db)
	ctx := context.Background()

	page1, total, err := repos.Stats.GetUserHistory(ctx, "user-1", 1, 2)
	if err != nil {
		t.Fatal(err)
	}
	if total != 3 {
		t.Errorf("total for user-1 = %d, want 3", total)
	}
	if len(page1) != 2 {
		t.Fatalf("page 1 size = %d, want 2", len(page1))
	}
	// Newest first: a1 (10m ago) then a2 (20m ago).
	if page1[0].Id != "a1" || page1[1].Id != "a2" {
		t.Errorf("page 1 order = [%s, %s], want [a1, a2]", page1[0].Id, page1[1].Id)
	}

	page2, _, err := repos.Stats.GetUserHistory(ctx, "user-1", 2, 2)
	if err != nil {
		t.Fatal(err)
	}
	if len(page2) != 1 || page2[0].Id != "a3" {
		t.Errorf("page 2 = %+v, want single [a3]", page2)
	}
}
