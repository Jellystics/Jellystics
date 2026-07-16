package repository_test

import (
	"context"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"gorm.io/datatypes"
)

// TestConfigRepo verifies the single-row app config round-trips through Save/Get
// and that Save updates in place (never creating a second row).
func TestConfigRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	cfg := &models.AppConfig{
		ID: 1, JFHost: p("http://jf:8096"), RequireLogin: true, AppUrl: "http://app",
		Settings: datatypes.JSON([]byte("{}")), ApiKeys: datatypes.JSON([]byte("[]")),
	}
	if err := repos.Config.Save(ctx, cfg); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := repos.Config.Get(ctx)
	if err != nil {
		t.Fatalf("Get: %v", err)
	}
	if got.JFHost == nil || *got.JFHost != "http://jf:8096" {
		t.Errorf("JFHost = %v, want http://jf:8096", got.JFHost)
	}
	if !got.RequireLogin {
		t.Error("RequireLogin should be true")
	}

	// Update in place.
	got.RequireLogin = false
	if err := repos.Config.Save(ctx, got); err != nil {
		t.Fatalf("Save update: %v", err)
	}
	after, _ := repos.Config.Get(ctx)
	if after.RequireLogin {
		t.Error("RequireLogin should be false after update")
	}
}

// TestUserRepo verifies Upsert inserts then updates by Id, List returns all, and
// GetByID fetches one.
func TestUserRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.User.Upsert(ctx, []models.JFUser{
		{Id: "u1", Name: p("Alice"), IsAdministrator: true},
		{Id: "u2", Name: p("Bob")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, err := repos.User.List(ctx)
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("List len = %d, want 2", len(list))
	}

	u, err := repos.User.GetByID(ctx, "u1")
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if u.Name == nil || *u.Name != "Alice" || !u.IsAdministrator {
		t.Errorf("u1 = %+v, want Alice admin", u)
	}

	// Upsert same Id updates the name (no duplicate row).
	if err := repos.User.Upsert(ctx, []models.JFUser{{Id: "u1", Name: p("Alice2")}}); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	u, _ = repos.User.GetByID(ctx, "u1")
	if u.Name == nil || *u.Name != "Alice2" {
		t.Errorf("u1 name = %v, want Alice2 after upsert", u.Name)
	}
	list, _ = repos.User.List(ctx)
	if len(list) != 2 {
		t.Errorf("List len = %d after upsert, want 2 (no duplicate)", len(list))
	}
}

// TestUserRepo_UpsertEmpty verifies an empty slice is a no-op.
func TestUserRepo_UpsertEmpty(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	if err := repos.User.Upsert(context.Background(), nil); err != nil {
		t.Fatalf("Upsert(nil) should be a no-op, got %v", err)
	}
}

// TestLibraryRepo verifies Upsert/List/GetByID and that List excludes archived
// rows while ArchiveNotIn archives everything outside the kept id set.
func TestLibraryRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-1", Name: p("Movies"), CollectionType: p("movies")},
		{Id: "lib-2", Name: p("Shows"), CollectionType: p("tvshows")},
		{Id: "lib-3", Name: p("Stale")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	list, _ := repos.Library.List(ctx)
	if len(list) != 3 {
		t.Fatalf("List len = %d, want 3", len(list))
	}

	// Keep lib-1 and lib-2; lib-3 should be archived and excluded from List.
	if err := repos.Library.ArchiveNotIn(ctx, []string{"lib-1", "lib-2"}); err != nil {
		t.Fatalf("ArchiveNotIn: %v", err)
	}
	list, _ = repos.Library.List(ctx)
	if len(list) != 2 {
		t.Errorf("List len = %d after archive, want 2", len(list))
	}

	lib, err := repos.Library.GetByID(ctx, "lib-1")
	if err != nil || lib.Name == nil || *lib.Name != "Movies" {
		t.Errorf("GetByID lib-1 = %+v, err %v", lib, err)
	}
}

// TestItemRepo verifies items list by parent (archived excluded) and ArchiveNotIn
// scopes to a parent.
func TestItemRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	movie := "Movie"
	if err := repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: p("A"), Type: &movie, ParentId: p("lib-1"), Genres: emptyGenres},
		{Id: "m2", Name: p("B"), Type: &movie, ParentId: p("lib-1"), Genres: emptyGenres},
		{Id: "m3", Name: p("C"), Type: &movie, ParentId: p("lib-2"), Genres: emptyGenres},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	byParent, _ := repos.Item.ListByParent(ctx, "lib-1")
	if len(byParent) != 2 {
		t.Fatalf("ListByParent lib-1 = %d, want 2", len(byParent))
	}

	// Archiving within lib-1 must not touch lib-2's item.
	if err := repos.Item.ArchiveNotIn(ctx, "lib-1", []string{"m1"}); err != nil {
		t.Fatalf("ArchiveNotIn: %v", err)
	}
	byParent, _ = repos.Item.ListByParent(ctx, "lib-1")
	if len(byParent) != 1 || byParent[0].Id != "m1" {
		t.Errorf("ListByParent lib-1 after archive = %+v, want just m1", byParent)
	}
	byParent2, _ := repos.Item.ListByParent(ctx, "lib-2")
	if len(byParent2) != 1 {
		t.Errorf("lib-2 items = %d, want 1 (untouched)", len(byParent2))
	}

	item, err := repos.Item.GetByID(ctx, "m1")
	if err != nil || item.Id != "m1" {
		t.Errorf("GetByID m1 = %+v, err %v", item, err)
	}
}

// TestSeasonRepo verifies seasons list by series (archived excluded) and
// ArchiveNotIn scoped to a series.
func TestSeasonRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.Season.Upsert(ctx, []models.JFLibrarySeason{
		{Id: "s1", Name: p("S1"), SeriesId: p("series-1")},
		{Id: "s2", Name: p("S2"), SeriesId: p("series-1")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	list, _ := repos.Season.ListBySeries(ctx, "series-1")
	if len(list) != 2 {
		t.Fatalf("ListBySeries = %d, want 2", len(list))
	}
	if err := repos.Season.ArchiveNotIn(ctx, "series-1", []string{"s1"}); err != nil {
		t.Fatalf("ArchiveNotIn: %v", err)
	}
	list, _ = repos.Season.ListBySeries(ctx, "series-1")
	if len(list) != 1 || list[0].Id != "s1" {
		t.Errorf("ListBySeries after archive = %+v, want just s1", list)
	}
}

// TestEpisodeRepo verifies episodes list by series and by season.
func TestEpisodeRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{
		{Id: "e1", Name: p("E1"), SeriesId: p("series-1"), SeasonId: p("s1")},
		{Id: "e2", Name: p("E2"), SeriesId: p("series-1"), SeasonId: p("s2")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	bySeries, _ := repos.Episode.ListBySeries(ctx, "series-1")
	if len(bySeries) != 2 {
		t.Fatalf("ListBySeries = %d, want 2", len(bySeries))
	}
	bySeason, _ := repos.Episode.ListBySeason(ctx, "s1")
	if len(bySeason) != 1 || bySeason[0].Id != "e1" {
		t.Errorf("ListBySeason s1 = %+v, want just e1", bySeason)
	}

	if err := repos.Episode.ArchiveNotIn(ctx, "series-1", []string{"e1"}); err != nil {
		t.Fatalf("ArchiveNotIn: %v", err)
	}
	bySeries, _ = repos.Episode.ListBySeries(ctx, "series-1")
	if len(bySeries) != 1 {
		t.Errorf("ListBySeries after archive = %d, want 1", len(bySeries))
	}
}

// TestMusicArtistRepo verifies artists list by library and ArchiveNotIn.
func TestMusicArtistRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.MusicArtist.Upsert(ctx, []models.JFMusicArtist{
		{Id: "a1", Name: p("Daft Punk"), LibraryId: p("lib-music"), Genres: emptyGenres},
		{Id: "a2", Name: p("Justice"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	list, _ := repos.MusicArtist.ListByLibrary(ctx, "lib-music")
	if len(list) != 2 {
		t.Fatalf("ListByLibrary = %d, want 2", len(list))
	}
	if err := repos.MusicArtist.ArchiveNotIn(ctx, "lib-music", []string{"a1"}); err != nil {
		t.Fatalf("ArchiveNotIn: %v", err)
	}
	list, _ = repos.MusicArtist.ListByLibrary(ctx, "lib-music")
	if len(list) != 1 {
		t.Errorf("ListByLibrary after archive = %d, want 1", len(list))
	}
}

// TestMusicTrackRepo verifies tracks list by library, album (ordered by
// disc/index) and artist.
func TestMusicTrackRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	i := func(v int) *int { return &v }
	if err := repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "t1", Name: p("Track1"), LibraryId: p("lib-music"), AlbumId: p("alb-1"), ArtistId: p("a1"), DiscNumber: i(1), IndexNumber: i(2), Genres: emptyGenres},
		{Id: "t2", Name: p("Track2"), LibraryId: p("lib-music"), AlbumId: p("alb-1"), ArtistId: p("a1"), DiscNumber: i(1), IndexNumber: i(1), Genres: emptyGenres},
		{Id: "t3", Name: p("Track3"), LibraryId: p("lib-music"), AlbumId: p("alb-2"), ArtistId: p("a2"), Genres: emptyGenres},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	byLib, _ := repos.MusicTrack.ListByLibrary(ctx, "lib-music")
	if len(byLib) != 3 {
		t.Fatalf("ListByLibrary = %d, want 3", len(byLib))
	}

	byAlbum, _ := repos.MusicTrack.ListByAlbum(ctx, "alb-1")
	if len(byAlbum) != 2 {
		t.Fatalf("ListByAlbum alb-1 = %d, want 2", len(byAlbum))
	}
	// Ordered by disc then index → t2 (index 1) before t1 (index 2).
	if byAlbum[0].Id != "t2" || byAlbum[1].Id != "t1" {
		t.Errorf("ListByAlbum order = [%s %s], want [t2 t1]", byAlbum[0].Id, byAlbum[1].Id)
	}

	byArtist, _ := repos.MusicTrack.ListByArtist(ctx, "a1")
	if len(byArtist) != 2 {
		t.Errorf("ListByArtist a1 = %d, want 2", len(byArtist))
	}
}

// TestItemInfoRepo verifies upsert and orphan removal (info whose Id matches no
// item/episode/track is deleted).
func TestItemInfoRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	movie := "Movie"
	if err := repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "keep", Name: p("Keep"), Type: &movie, ParentId: p("lib-1"), Genres: emptyGenres},
	}); err != nil {
		t.Fatalf("seed item: %v", err)
	}
	if err := repos.ItemInfo.Upsert(ctx, []models.JFItemInfo{
		{Id: "keep", Path: p("/keep.mkv")},
		{Id: "orphan", Path: p("/orphan.mkv")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := repos.ItemInfo.RemoveOrphaned(ctx); err != nil {
		t.Fatalf("RemoveOrphaned: %v", err)
	}
	var count int64
	db.Model(&models.JFItemInfo{}).Count(&count)
	if count != 1 {
		t.Errorf("item_info count = %d after orphan removal, want 1 (only 'keep')", count)
	}
}

// TestPlaybackRepo verifies upsert, List (limit + newest first) and ListByUser.
func TestPlaybackRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "p1", UserId: p("u1"), NowPlayingItemId: p("m1"), PlaybackDuration: ptrI64(100), ActivityDateInserted: p("2026-07-10 10:00:00+00:00"), Source: "watchdog"},
		{Id: "p2", UserId: p("u1"), NowPlayingItemId: p("m2"), PlaybackDuration: ptrI64(200), ActivityDateInserted: p("2026-07-12 10:00:00+00:00"), Source: "watchdog"},
		{Id: "p3", UserId: p("u2"), NowPlayingItemId: p("m3"), PlaybackDuration: ptrI64(300), ActivityDateInserted: p("2026-07-11 10:00:00+00:00"), Source: "watchdog"},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	all, _ := repos.Playback.List(ctx, 0)
	if len(all) != 3 {
		t.Fatalf("List(0) = %d, want 3", len(all))
	}
	// Newest first: p2 (07-12) then p3 (07-11) then p1 (07-10).
	if all[0].Id != "p2" {
		t.Errorf("List order[0] = %s, want p2 (newest)", all[0].Id)
	}

	limited, _ := repos.Playback.List(ctx, 1)
	if len(limited) != 1 || limited[0].Id != "p2" {
		t.Errorf("List(1) = %+v, want just newest p2", limited)
	}

	byUser, _ := repos.Playback.ListByUser(ctx, "u1")
	if len(byUser) != 2 {
		t.Errorf("ListByUser u1 = %d, want 2", len(byUser))
	}
}

// TestWatchdogRepo verifies upsert, list and delete of active-session rows.
func TestWatchdogRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	if err := repos.Watchdog.Upsert(ctx, []models.JFActivityWatchdog{
		{Id: "w1", UserId: p("u1"), NowPlayingItemId: p("m1"), WatchedSeconds: 10, LastTickAt: ptrTime()},
		{Id: "w2", UserId: p("u2"), NowPlayingItemId: p("m2"), WatchedSeconds: 20},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}
	list, _ := repos.Watchdog.List(ctx)
	if len(list) != 2 {
		t.Fatalf("List = %d, want 2", len(list))
	}

	// Upsert updates WatchedSeconds in place.
	if err := repos.Watchdog.Upsert(ctx, []models.JFActivityWatchdog{
		{Id: "w1", UserId: p("u1"), NowPlayingItemId: p("m1"), WatchedSeconds: 35},
	}); err != nil {
		t.Fatalf("Upsert update: %v", err)
	}
	if err := repos.Watchdog.Delete(ctx, "w2"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ = repos.Watchdog.List(ctx)
	if len(list) != 1 || list[0].Id != "w1" || list[0].WatchedSeconds != 35 {
		t.Errorf("List after update+delete = %+v, want just w1 with 35s", list)
	}
}

// TestPluginDataRepo verifies unimported listing, max rowid and merge into
// playback activity mapping episode/track linkage.
func TestPluginDataRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	// Seed an episode and a music track so the merge can resolve linkage.
	if err := repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{
		{Id: "ep-1", Name: p("Pilot"), SeriesId: p("series-1"), SeasonId: p("s1")},
	}); err != nil {
		t.Fatalf("seed episode: %v", err)
	}
	if err := repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "tr-1", Name: p("Song"), AlbumId: p("alb-1"), AlbumName: p("Album"), LibraryId: p("lib-music"), Genres: emptyGenres},
	}); err != nil {
		t.Fatalf("seed track: %v", err)
	}

	if err := repos.PluginData.Upsert(ctx, []models.JFPluginData{
		{RowId: "10", ItemId: p("ep-1"), ItemName: p("Pilot"), UserId: p("u1"), PlayDuration: ptrI64(80), DateCreated: p("2026-07-12 10:00:00")},
		{RowId: "20", ItemId: p("tr-1"), ItemName: p("Song"), UserId: p("u1"), PlayDuration: ptrI64(200), DateCreated: p("2026-07-12 11:00:00")},
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	unimported, _ := repos.PluginData.ListUnimported(ctx)
	if len(unimported) != 2 {
		t.Fatalf("ListUnimported = %d, want 2", len(unimported))
	}
	max, err := repos.PluginData.GetMaxRowId(ctx)
	if err != nil {
		t.Fatalf("GetMaxRowId: %v", err)
	}
	if max != 20 {
		t.Errorf("GetMaxRowId = %d, want 20", max)
	}

	if err := repos.PluginData.MergeIntoPlaybackActivity(ctx); err != nil {
		t.Fatalf("MergeIntoPlaybackActivity: %v", err)
	}
	// Episode row: EpisodeId set to ItemId, NowPlayingItemId set to SeriesId.
	var ep models.JFPlaybackActivity
	if err := db.Where(`"Id" = ?`, "10").First(&ep).Error; err != nil {
		t.Fatalf("merged episode row not found: %v", err)
	}
	if ep.EpisodeId == nil || *ep.EpisodeId != "ep-1" {
		t.Errorf("merged episode EpisodeId = %v, want ep-1", ep.EpisodeId)
	}
	if ep.NowPlayingItemId == nil || *ep.NowPlayingItemId != "series-1" {
		t.Errorf("merged episode NowPlayingItemId = %v, want series-1", ep.NowPlayingItemId)
	}
	if !ep.Imported {
		t.Error("merged rows should be marked imported")
	}
	// Track row: EpisodeId set to track id, NowPlayingItemId set to AlbumId.
	var tr models.JFPlaybackActivity
	if err := db.Where(`"Id" = ?`, "20").First(&tr).Error; err != nil {
		t.Fatalf("merged track row not found: %v", err)
	}
	if tr.EpisodeId == nil || *tr.EpisodeId != "tr-1" {
		t.Errorf("merged track EpisodeId = %v, want tr-1", tr.EpisodeId)
	}
	if tr.NowPlayingItemId == nil || *tr.NowPlayingItemId != "alb-1" {
		t.Errorf("merged track NowPlayingItemId = %v, want alb-1", tr.NowPlayingItemId)
	}
}

// TestLogRepo verifies Insert, Upsert (updates same Id) and List (limit, newest
// first).
func TestLogRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	running := "running"
	if err := repos.Log.Insert(ctx, &models.JFLogging{
		Id: "job-1", Name: p("Full Sync"), Result: &running, TimeRun: p("2026-07-12 10:00:00"),
	}); err != nil {
		t.Fatalf("Insert: %v", err)
	}

	// Upsert on the same Id updates the result rather than inserting a new row.
	done := "success"
	if err := repos.Log.Upsert(ctx, &models.JFLogging{
		Id: "job-1", Name: p("Full Sync"), Result: &done, TimeRun: p("2026-07-12 10:00:00"),
	}); err != nil {
		t.Fatalf("Upsert: %v", err)
	}

	if err := repos.Log.Insert(ctx, &models.JFLogging{
		Id: "job-2", Name: p("Backup"), Result: &done, TimeRun: p("2026-07-12 11:00:00"),
	}); err != nil {
		t.Fatalf("Insert 2: %v", err)
	}

	all, _ := repos.Log.List(ctx, 0)
	if len(all) != 2 {
		t.Fatalf("List = %d, want 2 (upsert did not duplicate)", len(all))
	}
	if all[0].Id != "job-2" {
		t.Errorf("List order[0] = %s, want job-2 (newest TimeRun)", all[0].Id)
	}
	var j1 models.JFLogging
	db.Where(`"Id" = ?`, "job-1").First(&j1)
	if j1.Result == nil || *j1.Result != "success" {
		t.Errorf("job-1 result = %v, want success after upsert", j1.Result)
	}

	limited, _ := repos.Log.List(ctx, 1)
	if len(limited) != 1 {
		t.Errorf("List(1) = %d, want 1", len(limited))
	}
}

// TestWebhookRepo verifies the full CRUD lifecycle: create, get, list, update,
// delete.
func TestWebhookRepo(t *testing.T) {
	db := setupTestDB(t)
	repos := repository.New(db)
	ctx := context.Background()

	wh := &models.Webhook{
		Name: "Discord", Url: "http://discord/hook", Method: "POST", TriggerType: "event",
		EventType: p("task_complete"), Enabled: true,
		Headers: datatypes.JSON([]byte("{}")), Payload: datatypes.JSON([]byte("{}")),
		DiscordEvents: datatypes.JSON([]byte("[]")), BotUsername: "jellystics_bot",
	}
	if err := repos.Webhook.Create(ctx, wh); err != nil {
		t.Fatalf("Create: %v", err)
	}
	if wh.Id == 0 {
		t.Fatal("Create should assign an auto-increment Id")
	}

	got, err := repos.Webhook.GetByID(ctx, wh.Id)
	if err != nil {
		t.Fatalf("GetByID: %v", err)
	}
	if got.Name != "Discord" {
		t.Errorf("Name = %q, want Discord", got.Name)
	}

	got.Enabled = false
	if err := repos.Webhook.Update(ctx, got); err != nil {
		t.Fatalf("Update: %v", err)
	}
	after, _ := repos.Webhook.GetByID(ctx, wh.Id)
	if after.Enabled {
		t.Error("Enabled should be false after update")
	}

	list, _ := repos.Webhook.List(ctx)
	if len(list) != 1 {
		t.Fatalf("List = %d, want 1", len(list))
	}

	if err := repos.Webhook.Delete(ctx, wh.Id); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	list, _ = repos.Webhook.List(ctx)
	if len(list) != 0 {
		t.Errorf("List after delete = %d, want 0", len(list))
	}
}

func ptrTime() *time.Time { now := time.Now(); return &now }
