package repository_test

import (
	"context"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func p(s string) *string { return &s }

var emptyGenres = datatypes.JSON([]byte("[]"))

// seedMusic populates a realistic music library:
//
//	Library "lib-music"
//	  Artist "artist-1" (Daft Punk)
//	    Album "album-1" (Discovery)  → track-1, track-2
//	    Album "album-2" (Homework)   → track-3
//	  Artist "artist-2" (Justice)
//	    Album "album-3" (Cross)      → track-4
//
// Playback rows follow the mapping NowPlayingItemId = AlbumId, EpisodeId = trackId:
//
//	track-1 played 3×, track-2 played 1×, track-3 played 2×, track-4 played 1×
//
// So per album: album-1 = 4 plays, album-2 = 2 plays, album-3 = 1 play.
// Per artist: artist-1 = 6 plays, artist-2 = 1 play.
func seedMusic(t *testing.T, db *gorm.DB) {
	t.Helper()

	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}

	musicAlbum := "MusicAlbum"
	must(db.Create(&[]models.JFMusicArtist{
		{Id: "artist-1", Name: p("Daft Punk"), LibraryId: p("lib-music"), Genres: emptyGenres},
		{Id: "artist-2", Name: p("Justice"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}).Error)

	must(db.Create(&[]models.JFLibraryItem{
		{Id: "album-1", Name: p("Discovery"), Type: &musicAlbum, ParentId: p("lib-music"), Genres: emptyGenres},
		{Id: "album-2", Name: p("Homework"), Type: &musicAlbum, ParentId: p("lib-music"), Genres: emptyGenres},
		{Id: "album-3", Name: p("Cross"), Type: &musicAlbum, ParentId: p("lib-music"), Genres: emptyGenres},
	}).Error)

	must(db.Create(&[]models.JFMusicTrack{
		{Id: "track-1", Name: p("One More Time"), AlbumId: p("album-1"), AlbumName: p("Discovery"), ArtistId: p("artist-1"), LibraryId: p("lib-music"), Genres: emptyGenres},
		{Id: "track-2", Name: p("Aerodynamic"), AlbumId: p("album-1"), AlbumName: p("Discovery"), ArtistId: p("artist-1"), LibraryId: p("lib-music"), Genres: emptyGenres},
		{Id: "track-3", Name: p("Around the World"), AlbumId: p("album-2"), AlbumName: p("Homework"), ArtistId: p("artist-1"), LibraryId: p("lib-music"), Genres: emptyGenres},
		{Id: "track-4", Name: p("Genesis"), AlbumId: p("album-3"), AlbumName: p("Cross"), ArtistId: p("artist-2"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}).Error)

	// Build playback rows: play(albumId, trackId, n times).
	date := "2026-07-12 10:00:00+00:00"
	var acts []models.JFPlaybackActivity
	play := func(album, track string, n int) {
		for i := 0; i < n; i++ {
			id := track + "-play-" + string(rune('a'+i))
			acts = append(acts, models.JFPlaybackActivity{
				Id:                   id,
				UserId:               p("user-1"),
				NowPlayingItemId:     p(album),
				EpisodeId:            p(track),
				PlaybackDuration:     ptrI64(180),
				ActivityDateInserted: p(date),
				Source:               "watchdog",
			})
		}
	}
	play("album-1", "track-1", 3)
	play("album-1", "track-2", 1)
	play("album-2", "track-3", 2)
	play("album-3", "track-4", 1)
	must(db.Create(&acts).Error)
}

func ptrI64(v int64) *int64 { return &v }

func TestGetMostPlayedTracks(t *testing.T) {
	db := setupTestDB(t)
	seedMusic(t, db)
	repos := repository.New(db)
	ctx := context.Background()

	t.Run("whole library ranks tracks by real play count", func(t *testing.T) {
		got, err := repos.Stats.GetMostPlayedTracks(ctx, "lib-music", "", 10)
		if err != nil {
			t.Fatal(err)
		}
		want := map[string]int64{"track-1": 3, "track-2": 1, "track-3": 2, "track-4": 1}
		assertTrackCounts(t, got, want)
		// Highest first.
		if len(got) == 0 || got[0].TrackId != "track-1" {
			t.Fatalf("expected track-1 ranked first, got %+v", got)
		}
	})

	t.Run("album filter restricts to that album's tracks", func(t *testing.T) {
		got, err := repos.Stats.GetMostPlayedTracks(ctx, "lib-music", "album-1", 10)
		if err != nil {
			t.Fatal(err)
		}
		assertTrackCounts(t, got, map[string]int64{"track-1": 3, "track-2": 1})
		for _, r := range got {
			if r.TrackId == "track-3" || r.TrackId == "track-4" {
				t.Fatalf("album filter leaked foreign track: %s", r.TrackId)
			}
		}
	})
}

func TestGetMostPlayedArtists(t *testing.T) {
	db := setupTestDB(t)
	seedMusic(t, db)
	repos := repository.New(db)

	got, err := repos.Stats.GetMostPlayedArtists(context.Background(), "lib-music", 10)
	if err != nil {
		t.Fatal(err)
	}
	counts := map[string]int64{}
	for _, r := range got {
		counts[r.ArtistId] = r.TimesPlayed
	}
	// artist-1 has 3+1+2 = 6 track plays; artist-2 has 1.
	if counts["artist-1"] != 6 {
		t.Fatalf("artist-1 = %d, want 6 (sum of its track plays)", counts["artist-1"])
	}
	if counts["artist-2"] != 1 {
		t.Fatalf("artist-2 = %d, want 1", counts["artist-2"])
	}
}

func TestGetMostPlayedAlbums(t *testing.T) {
	db := setupTestDB(t)
	seedMusic(t, db)
	repos := repository.New(db)
	ctx := context.Background()

	t.Run("counts album plays directly, never multiplied by track count", func(t *testing.T) {
		got, err := repos.Stats.GetMostPlayedAlbums(ctx, "lib-music", "", 10)
		if err != nil {
			t.Fatal(err)
		}
		counts := map[string]int64{}
		for _, r := range got {
			counts[r.AlbumId] = r.TimesPlayed
		}
		// album-1 has 4 plays across its 2 tracks. A naive track-join would
		// report 8; the correct behavior counts playback rows once (= 4).
		if counts["album-1"] != 4 {
			t.Fatalf("album-1 = %d, want 4 (must not multiply by track count)", counts["album-1"])
		}
		if counts["album-2"] != 2 {
			t.Fatalf("album-2 = %d, want 2", counts["album-2"])
		}
		if counts["album-3"] != 1 {
			t.Fatalf("album-3 = %d, want 1", counts["album-3"])
		}
	})

	t.Run("artist filter keeps only that artist's albums", func(t *testing.T) {
		got, err := repos.Stats.GetMostPlayedAlbums(ctx, "lib-music", "artist-1", 10)
		if err != nil {
			t.Fatal(err)
		}
		for _, r := range got {
			if r.AlbumId == "album-3" {
				t.Fatalf("artist-1 filter leaked album-3 (belongs to artist-2)")
			}
		}
		if len(got) != 2 {
			t.Fatalf("artist-1 should have 2 albums, got %d: %+v", len(got), got)
		}
	})
}

func assertTrackCounts(t *testing.T, got []repository.TrackPlayStat, want map[string]int64) {
	t.Helper()
	counts := map[string]int64{}
	for _, r := range got {
		counts[r.TrackId] = r.TimesPlayed
	}
	for id, n := range want {
		if counts[id] != n {
			t.Fatalf("track %s = %d, want %d (all: %+v)", id, counts[id], n, counts)
		}
	}
}
