package sync

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/Jellystics/Jellystics/internal/ws"
	"gorm.io/gorm"
)

// mockJF is a scripted Jellyfin mock. Each field is a handler keyed by the
// endpoint the sync service hits. GetSessions consumes sessionScript one entry
// per call, so a test can drive SessionTick across a sequence of ticks.
type mockJF struct {
	server        *httptest.Server
	sessionScript [][]jellyfin.SessionInfo
	sessionIdx    int

	users     []jellyfin.User
	libraries []jellyfin.Library
	// items maps IncludeItemTypes (comma-joined) -> items returned for /Items.
	items map[string][]jellyfin.Item
}

func newMockJF(t *testing.T) *mockJF {
	t.Helper()
	m := &mockJF{items: map[string][]jellyfin.Item{}}
	mux := http.NewServeMux()

	mux.HandleFunc("/Sessions", func(w http.ResponseWriter, r *http.Request) {
		var out []jellyfin.SessionInfo
		if m.sessionIdx < len(m.sessionScript) {
			out = m.sessionScript[m.sessionIdx]
			m.sessionIdx++
		}
		writeJSON(w, out)
	})

	mux.HandleFunc("/Users", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, m.users)
	})

	// /Users/{id}/Views for GetLibraries.
	mux.HandleFunc("/Users/", func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/Views") {
			writeJSON(w, jellyfin.LibrariesResponse{Items: m.libraries})
			return
		}
		writeJSON(w, jellyfin.ItemsResponse{})
	})

	mux.HandleFunc("/Items", func(w http.ResponseWriter, r *http.Request) {
		types := r.URL.Query().Get("IncludeItemTypes")
		items := m.items[types]
		writeJSON(w, jellyfin.ItemsResponse{
			Items:            items,
			TotalRecordCount: len(items),
			StartIndex:       0,
		})
	})

	m.server = httptest.NewServer(mux)
	t.Cleanup(m.server.Close)
	return m
}

func writeJSON(w http.ResponseWriter, v any) {
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(v)
}

// newTestService wires a real repository container against a fresh throwaway DB,
// seeds app_config.JF_HOST to the mock server URL (as production does), and
// returns a sync.Service that will talk to the mock via refreshClient.
func newTestService(t *testing.T, mock *mockJF) (*Service, *gorm.DB) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)

	// Point the DB config at the mock server so refreshClient targets it.
	host := mock.server.URL
	if err := db.Exec(`UPDATE app_config SET "JF_HOST" = ? WHERE "ID" = 1`, host).Error; err != nil {
		t.Fatalf("seed JF_HOST: %v", err)
	}

	// A client is still required by New(); refreshClient overwrites it, but pass a
	// correct one anyway to mirror production bootstrap.
	jf := jellyfin.NewClient(host, "")
	svc := New(repos, jf, ws.NewHub())
	return svc, db
}

// rewindTick moves the watchdog LastTickAt for a session back by `secs` seconds
// so the next SessionTick sees a wall-clock gap. In production this gap is real
// (~10s between ticks); in a test the two ticks happen microseconds apart, so we
// simulate elapsed time explicitly. This does NOT touch the position/pause
// accounting — it only supplies the wall-clock the increment is clamped to.
func rewindTick(t *testing.T, db *gorm.DB, sessionID string, secs int) {
	t.Helper()
	err := db.Exec(
		`UPDATE jf_activity_watchdog SET "LastTickAt" = "LastTickAt" - (? * interval '1 second') WHERE "Id" = ?`,
		secs, sessionID,
	).Error
	if err != nil {
		t.Fatalf("rewind tick: %v", err)
	}
}

// --- test data builders ---

func movieSession(sessionID, itemID, name string, posSeconds int64, paused bool) jellyfin.SessionInfo {
	pos := posSeconds * ticksPerSecond
	return jellyfin.SessionInfo{
		Id:       sessionID,
		UserId:   "user-1",
		UserName: "alice",
		Client:   "Jellyfin Web",
		NowPlayingItem: &jellyfin.SessionItem{
			Id:   itemID,
			Name: name,
			Type: "Movie",
		},
		PlayState: &jellyfin.PlayState{PositionTicks: &pos, IsPaused: paused},
	}
}

func musicSession(sessionID, albumID, trackID, trackName, albumName string, posSeconds int64) jellyfin.SessionInfo {
	pos := posSeconds * ticksPerSecond
	album := albumName
	return jellyfin.SessionInfo{
		Id:       sessionID,
		UserId:   "user-1",
		UserName: "alice",
		Client:   "Jellyfin Web",
		NowPlayingItem: &jellyfin.SessionItem{
			Id:      trackID,
			Name:    trackName,
			Type:    "Audio",
			AlbumId: &albumID,
			Album:   &album,
		},
		PlayState: &jellyfin.PlayState{PositionTicks: &pos, IsPaused: false},
	}
}

// TestSessionTick_PauseNeverCounted proves that while a session is paused the
// media PositionTicks are frozen, so the accumulated WatchedSeconds does not
// grow — regardless of how much wall-clock time passes.
func TestSessionTick_PauseNeverCounted(t *testing.T) {
	mock := newMockJF(t)
	// Tick 1: fresh session at pos 100 (no baseline yet → +0).
	// Tick 2: pos 110, 10s wall elapsed → +10.
	// Tick 3: PAUSED, pos frozen at 110, 10s wall elapsed → +0.
	// Tick 4: still paused, pos still 110, 10s wall → +0.
	// Tick 5: resumed, pos 120, 10s wall → +10. Total real watched = 20.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{movieSession("sess-1", "movie-1", "Inception", 100, false)},
		{movieSession("sess-1", "movie-1", "Inception", 110, false)},
		{movieSession("sess-1", "movie-1", "Inception", 110, true)},
		{movieSession("sess-1", "movie-1", "Inception", 110, true)},
		{movieSession("sess-1", "movie-1", "Inception", 120, false)},
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.SessionTick(ctx)
		// Simulate ~10s between ticks so the wall-clock clamp permits progress.
		rewindTick(t, db, "sess-1", 10)
	}

	var wd models.JFActivityWatchdog
	if err := db.Where(`"Id" = ?`, "sess-1").First(&wd).Error; err != nil {
		t.Fatalf("watchdog row missing: %v", err)
	}
	if wd.WatchedSeconds != 20 {
		t.Fatalf("WatchedSeconds = %d, want 20 (20s of pause must be excluded)", wd.WatchedSeconds)
	}
}

// TestSessionTick_PromoteFinishedSession proves that when a session ends (drops
// out of /Sessions) its accumulated real watch time is promoted to
// jf_playback_activity in SECONDS, and the watchdog row is cleared.
func TestSessionTick_PromoteFinishedSession(t *testing.T) {
	mock := newMockJF(t)
	// Accumulate 40s of real watch time in +10s ticks (per-tick increment is
	// capped at maxTickDelta=15, so we keep each step at 10s), then the session
	// disappears from /Sessions.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{movieSession("sess-1", "movie-1", "Inception", 0, false)},
		{movieSession("sess-1", "movie-1", "Inception", 10, false)}, // +10
		{movieSession("sess-1", "movie-1", "Inception", 20, false)}, // +10
		{movieSession("sess-1", "movie-1", "Inception", 30, false)}, // +10
		{movieSession("sess-1", "movie-1", "Inception", 40, false)}, // +10 → 40 real
		{}, // session ended → promote
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.SessionTick(ctx)
		rewindTick(t, db, "sess-1", 10)
	}
	svc.SessionTick(ctx) // end

	// Watchdog cleared.
	var count int64
	db.Model(&models.JFActivityWatchdog{}).Where(`"Id" = ?`, "sess-1").Count(&count)
	if count != 0 {
		t.Fatalf("watchdog row should be deleted after promotion, found %d", count)
	}

	var act models.JFPlaybackActivity
	if err := db.Where(`"NowPlayingItemId" = ?`, "movie-1").First(&act).Error; err != nil {
		t.Fatalf("promoted activity missing: %v", err)
	}
	if act.PlaybackDuration == nil || *act.PlaybackDuration != 40 {
		t.Fatalf("PlaybackDuration = %v, want 40 (real watched seconds)", act.PlaybackDuration)
	}
	if act.Source != "watchdog" {
		t.Fatalf("Source = %q, want watchdog", act.Source)
	}
}

// TestSessionTick_ShortSessionDiscarded proves that a session shorter than
// minPlaybackSeconds (30s) is discarded on end and never promoted.
func TestSessionTick_ShortSessionDiscarded(t *testing.T) {
	mock := newMockJF(t)
	// Only ~10s of real watch time, then end.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{movieSession("sess-1", "movie-short", "Trailer", 0, false)},
		{movieSession("sess-1", "movie-short", "Trailer", 10, false)}, // +10
		{}, // end
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	svc.SessionTick(ctx)
	rewindTick(t, db, "sess-1", 10)
	svc.SessionTick(ctx)
	rewindTick(t, db, "sess-1", 10)
	svc.SessionTick(ctx) // end

	var count int64
	db.Model(&models.JFPlaybackActivity{}).Where(`"NowPlayingItemId" = ?`, "movie-short").Count(&count)
	if count != 0 {
		t.Fatalf("short session (<30s) must not be promoted, found %d activity row(s)", count)
	}
	// Watchdog row also removed.
	db.Model(&models.JFActivityWatchdog{}).Where(`"Id" = ?`, "sess-1").Count(&count)
	if count != 0 {
		t.Fatalf("short-session watchdog row should be deleted, found %d", count)
	}
}

// TestSessionTick_MediaSwitchPromotesPrevious proves that when a session switches
// to a different item, the previous item is promoted to jf_playback_activity with
// its accumulated PlaybackDuration (seconds) and a fresh watchdog begins for the
// new item.
func TestSessionTick_MediaSwitchPromotesPrevious(t *testing.T) {
	mock := newMockJF(t)
	// movie-A accumulates 40s (+10s per tick, under the 15s cap), then the
	// session switches to movie-B which starts a fresh count.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{movieSession("sess-1", "movie-A", "First Film", 0, false)},
		{movieSession("sess-1", "movie-A", "First Film", 10, false)}, // +10
		{movieSession("sess-1", "movie-A", "First Film", 20, false)}, // +10
		{movieSession("sess-1", "movie-A", "First Film", 30, false)}, // +10
		{movieSession("sess-1", "movie-A", "First Film", 40, false)}, // +10 → 40 real
		{movieSession("sess-1", "movie-B", "Second Film", 5, false)}, // SWITCH → promote movie-A(40)
		{movieSession("sess-1", "movie-B", "Second Film", 15, false)}, // movie-B +10
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	// Five ticks build movie-A to 40s. Do NOT rewind after the final build tick:
	// the switch promotion credits one last wall-clock leg (fallback path with
	// curPos=nil), so leaving ~0 wall-clock before the switch keeps the promoted
	// duration at the accumulated 40s.
	for i := 0; i < 5; i++ {
		svc.SessionTick(ctx)
		if i < 4 {
			rewindTick(t, db, "sess-1", 10)
		}
	}
	svc.SessionTick(ctx) // switch → movie-A promoted, movie-B watchdog starts fresh
	rewindTick(t, db, "sess-1", 10)
	svc.SessionTick(ctx) // movie-B accumulates +10

	// movie-A promoted with 40s.
	var actA models.JFPlaybackActivity
	if err := db.Where(`"NowPlayingItemId" = ?`, "movie-A").First(&actA).Error; err != nil {
		t.Fatalf("movie-A should be promoted on media switch: %v", err)
	}
	if actA.PlaybackDuration == nil || *actA.PlaybackDuration != 40 {
		t.Fatalf("movie-A PlaybackDuration = %v, want 40", actA.PlaybackDuration)
	}

	// A fresh watchdog now tracks movie-B.
	var wd models.JFActivityWatchdog
	if err := db.Where(`"Id" = ?`, "sess-1").First(&wd).Error; err != nil {
		t.Fatalf("watchdog for switched session missing: %v", err)
	}
	if wd.NowPlayingItemId == nil || *wd.NowPlayingItemId != "movie-B" {
		t.Fatalf("watchdog now-playing = %v, want movie-B", wd.NowPlayingItemId)
	}
	// movie-B accumulated its own real time (started fresh at 0 → +10).
	if wd.WatchedSeconds != 10 {
		t.Fatalf("movie-B WatchedSeconds = %d, want 10 (fresh count after switch)", wd.WatchedSeconds)
	}
}

// TestSessionTick_MusicMapping proves the music parent/child mapping mirrors
// Series/Episode: NowPlayingItemId = albumId, EpisodeId = trackId.
func TestSessionTick_MusicMapping(t *testing.T) {
	mock := newMockJF(t)
	// +10s per tick (under the 15s cap) to reach 40s of real play, then end.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{musicSession("sess-1", "album-1", "track-1", "Song One", "Greatest Hits", 0)},
		{musicSession("sess-1", "album-1", "track-1", "Song One", "Greatest Hits", 10)},
		{musicSession("sess-1", "album-1", "track-1", "Song One", "Greatest Hits", 20)},
		{musicSession("sess-1", "album-1", "track-1", "Song One", "Greatest Hits", 30)},
		{musicSession("sess-1", "album-1", "track-1", "Song One", "Greatest Hits", 40)},
		{}, // end → promote
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		svc.SessionTick(ctx)
		rewindTick(t, db, "sess-1", 10)
	}
	svc.SessionTick(ctx) // end

	var act models.JFPlaybackActivity
	if err := db.Where(`"EpisodeId" = ?`, "track-1").First(&act).Error; err != nil {
		t.Fatalf("promoted music activity missing (queried by track EpisodeId): %v", err)
	}
	if act.NowPlayingItemId == nil || *act.NowPlayingItemId != "album-1" {
		t.Fatalf("NowPlayingItemId = %v, want album-1 (album is the parent)", act.NowPlayingItemId)
	}
	if act.EpisodeId == nil || *act.EpisodeId != "track-1" {
		t.Fatalf("EpisodeId = %v, want track-1 (track is the child)", act.EpisodeId)
	}
	if act.SeriesName == nil || *act.SeriesName != "Greatest Hits" {
		t.Fatalf("SeriesName = %v, want album name Greatest Hits", act.SeriesName)
	}
	if act.PlaybackDuration == nil || *act.PlaybackDuration != 40 {
		t.Fatalf("PlaybackDuration = %v, want 40s", act.PlaybackDuration)
	}
}

// TestSyncSessions_MediaSwitchPromotesPrevious mirrors the SessionTick media-switch
// test for the manual-trigger twin SyncSessions, which must stay behaviourally in
// sync with SessionTick's promotion logic.
func TestSyncSessions_MediaSwitchPromotesPrevious(t *testing.T) {
	mock := newMockJF(t)
	// +10s per tick (under the 15s cap) to build movie-A to 40s, then switch.
	mock.sessionScript = [][]jellyfin.SessionInfo{
		{movieSession("sess-1", "movie-A", "First Film", 0, false)},
		{movieSession("sess-1", "movie-A", "First Film", 10, false)},
		{movieSession("sess-1", "movie-A", "First Film", 20, false)},
		{movieSession("sess-1", "movie-A", "First Film", 30, false)},
		{movieSession("sess-1", "movie-A", "First Film", 40, false)}, // → 40 real
		{movieSession("sess-1", "movie-B", "Second Film", 5, false)}, // switch → promote A
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		if err := svc.SyncSessions(ctx); err != nil {
			t.Fatalf("SyncSessions: %v", err)
		}
		if i < 4 { // no rewind before the switch: promotion adds a final wall-clock leg
			rewindTick(t, db, "sess-1", 10)
		}
	}
	if err := svc.SyncSessions(ctx); err != nil { // switch → promote A
		t.Fatalf("SyncSessions: %v", err)
	}

	var actA models.JFPlaybackActivity
	if err := db.Where(`"NowPlayingItemId" = ?`, "movie-A").First(&actA).Error; err != nil {
		t.Fatalf("SyncSessions should promote movie-A on switch: %v", err)
	}
	if actA.PlaybackDuration == nil || *actA.PlaybackDuration != 40 {
		t.Fatalf("movie-A PlaybackDuration = %v, want 40", actA.PlaybackDuration)
	}
}

// TestSyncMusicLibrary_Mapping proves the item-sync path upserts artists, albums,
// and tracks with the correct table mapping: MusicAlbum → jf_library_items,
// Audio → jf_music_tracks (with AlbumId/ArtistId linkage).
func TestSyncMusicLibrary_Mapping(t *testing.T) {
	mock := newMockJF(t)
	mock.items["MusicArtist"] = []jellyfin.Item{
		{Id: "artist-1", Name: "The Band", Type: "MusicArtist", Genres: []string{"rock"}},
	}
	mock.items["MusicAlbum"] = []jellyfin.Item{
		{Id: "album-1", Name: "Debut", Type: "MusicAlbum"},
	}
	mock.items["Audio"] = []jellyfin.Item{
		{
			Id: "track-1", Name: "Opener", Type: "Audio",
			AlbumId:      sp("album-1"),
			Album:        sp("Debut"),
			AlbumArtist:  sp("The Band"),
			AlbumArtists: []jellyfin.NamedItem{{Id: "artist-1", Name: "The Band"}},
			IndexNumber:  intp(1),
			RunTimeTicks: i64p(2000000000),
			MediaSources: []jellyfin.MediaSource{{Path: "/music/opener.flac", Size: i64p(9000)}},
		},
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	if err := svc.SyncMusicLibrary(ctx, "lib-music"); err != nil {
		t.Fatalf("SyncMusicLibrary: %v", err)
	}

	// Artist upserted.
	var artist models.JFMusicArtist
	if err := db.Where(`"Id" = ?`, "artist-1").First(&artist).Error; err != nil {
		t.Fatalf("artist not upserted: %v", err)
	}

	// Album upserted into jf_library_items.
	var album models.JFLibraryItem
	if err := db.Where(`"Id" = ?`, "album-1").First(&album).Error; err != nil {
		t.Fatalf("album not upserted into jf_library_items: %v", err)
	}
	if album.Type == nil || *album.Type != "MusicAlbum" {
		t.Fatalf("album Type = %v, want MusicAlbum", album.Type)
	}

	// Track upserted into jf_music_tracks with album/artist linkage.
	var track models.JFMusicTrack
	if err := db.Where(`"Id" = ?`, "track-1").First(&track).Error; err != nil {
		t.Fatalf("track not upserted: %v", err)
	}
	if track.AlbumId == nil || *track.AlbumId != "album-1" {
		t.Fatalf("track AlbumId = %v, want album-1", track.AlbumId)
	}
	if track.ArtistId == nil || *track.ArtistId != "artist-1" {
		t.Fatalf("track ArtistId = %v, want artist-1 (first album artist)", track.ArtistId)
	}
}

// TestSyncTVLibrary_Mapping proves series/seasons/episodes are upserted into their
// respective tables with correct series/season linkage.
func TestSyncTVLibrary_Mapping(t *testing.T) {
	mock := newMockJF(t)
	mock.items["Series"] = []jellyfin.Item{
		{Id: "series-1", Name: "Lost", Type: "Series"},
	}
	mock.items["Season"] = []jellyfin.Item{
		{Id: "season-1", Name: "Season 1", Type: "Season", SeriesId: sp("series-1"), IndexNumber: intp(1)},
	}
	mock.items["Episode"] = []jellyfin.Item{
		{
			Id: "ep-1", Name: "Pilot", Type: "Episode",
			SeriesId: sp("series-1"), SeasonId: sp("season-1"),
			SeriesName: sp("Lost"), IndexNumber: intp(1),
			MediaSources: []jellyfin.MediaSource{{Path: "/tv/pilot.mkv", Size: i64p(500)}},
		},
	}
	svc, db := newTestService(t, mock)
	ctx := context.Background()

	if err := svc.SyncTVLibrary(ctx, "lib-tv"); err != nil {
		t.Fatalf("SyncTVLibrary: %v", err)
	}

	var series models.JFLibraryItem
	if err := db.Where(`"Id" = ?`, "series-1").First(&series).Error; err != nil {
		t.Fatalf("series not upserted: %v", err)
	}
	var season models.JFLibrarySeason
	if err := db.Where(`"Id" = ?`, "season-1").First(&season).Error; err != nil {
		t.Fatalf("season not upserted: %v", err)
	}
	if season.SeriesId == nil || *season.SeriesId != "series-1" {
		t.Fatalf("season SeriesId = %v, want series-1", season.SeriesId)
	}
	var ep models.JFLibraryEpisode
	if err := db.Where(`"Id" = ?`, "ep-1").First(&ep).Error; err != nil {
		t.Fatalf("episode not upserted: %v", err)
	}
	if ep.SeriesId == nil || *ep.SeriesId != "series-1" {
		t.Fatalf("episode SeriesId = %v, want series-1", ep.SeriesId)
	}
	if ep.SeasonId == nil || *ep.SeasonId != "season-1" {
		t.Fatalf("episode SeasonId = %v, want season-1", ep.SeasonId)
	}
}
