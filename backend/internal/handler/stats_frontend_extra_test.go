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

// ---------------------------------------------------------------------------
// seedRichStats builds a deterministic multi-library dataset used by the
// stats_frontend endpoints. It intentionally spans a movie library, a TV
// library (series → season → episode), and a music library (artist → album →
// track) so aggregate/genre/library queries have exact expected values.
//
// Timestamps use a fixed "recent" date (relative to the test run) so day/hour
// window filters (days=30) include them; queries that key on all-time use
// days=0. To stay window-agnostic, tests here pass days=0 unless noted.
//
// Playback rows (PlaybackDuration is SECONDS):
//   Movie "Inception" (m1, lib-movies, genre Action):
//     p1: Alice, Web/DirectPlay, 600s
//     p2: Alice, Web/DirectPlay, 600s
//     p3: Bob,   Android/Transcode, 300s
//   Episode "Pilot" (ep1 of series-1, lib-tv, genre Drama):
//     p4: Alice, Web/DirectPlay, 1200s  (NowPlayingItemId=series-1, EpisodeId=ep1)
//   Track "Song" (tr1 of album alb1, lib-music, genre Rock):
//     p5: Bob, Android/DirectPlay, 180s (NowPlayingItemId=alb1, EpisodeId=tr1)
// ---------------------------------------------------------------------------
func seedRichStats(t *testing.T, db *gorm.DB) {
	t.Helper()
	repos := repository.New(db)
	ctx := t.Context()
	must := func(err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("seed: %v", err)
		}
	}
	action := datatypes.JSON([]byte(`["Action"]`))
	drama := datatypes.JSON([]byte(`["Drama"]`))
	rock := datatypes.JSON([]byte(`["Rock"]`))

	must(repos.User.Upsert(ctx, []models.JFUser{
		{Id: "u1", Name: sp("Alice")},
		{Id: "u2", Name: sp("Bob")},
	}))

	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
		{Id: "lib-tv", Name: sp("Shows"), CollectionType: sp("tvshows")},
		{Id: "lib-music", Name: sp("Music"), CollectionType: sp("music")},
	}))

	movie := "Movie"
	series := "Series"
	album := "MusicAlbum"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"),
			Genres: action, RunTimeTicks: i64(1200 * secTicks), ProductionYear: ip(2010)},
		{Id: "series-1", Name: sp("Lost"), Type: &series, ParentId: sp("lib-tv"), Genres: drama},
		// Albums are stored as MusicAlbum library items parented to the music library.
		{Id: "alb1", Name: sp("Greatest Hits"), Type: &album, ParentId: sp("lib-music"),
			AlbumArtist: sp("The Band"), ArtistId: sp("art1"), Genres: rock},
	}))
	must(repos.Season.Upsert(ctx, []models.JFLibrarySeason{
		{Id: "s1", Name: sp("Season 1"), SeriesId: sp("series-1")},
	}))
	must(repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{
		{Id: "ep1", Name: sp("Pilot"), SeriesId: sp("series-1"), SeasonId: sp("s1"), RunTimeTicks: i64(1500 * secTicks)},
	}))

	must(repos.MusicArtist.Upsert(ctx, []models.JFMusicArtist{
		{Id: "art1", Name: sp("The Band"), LibraryId: sp("lib-music")},
	}))
	must(repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "tr1", Name: sp("Song"), AlbumId: sp("alb1"), AlbumName: sp("Greatest Hits"),
			AlbumArtist: sp("The Band"), ArtistId: sp("art1"),
			LibraryId: sp("lib-music"), Genres: rock, RunTimeTicks: i64(240 * secTicks)},
	}))

	// Item info for size aggregation.
	must(db.Create(&[]models.JFItemInfo{
		{Id: "m1", Size: i64(1000)},
		{Id: "ep1", Size: i64(500)},
		{Id: "tr1", Size: i64(50)},
	}).Error)

	d := "2026-07-12 14:00:00+00:00"
	dEp := "2026-07-12 09:00:00+00:00"
	must(repos.Playback.Upsert(ctx, []models.JFPlaybackActivity{
		{Id: "p1", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Web"), DeviceName: sp("Firefox"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p2", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Web"), DeviceName: sp("Firefox"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(600), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p3", UserId: sp("u2"), UserName: sp("Bob"), NowPlayingItemId: sp("m1"), NowPlayingItemName: sp("Inception"),
			Client: sp("Android"), DeviceName: sp("Pixel"), PlayMethod: sp("Transcode"), PlaybackDuration: i64(300), ActivityDateInserted: &dEp, Source: "watchdog"},
		{Id: "p4", UserId: sp("u1"), UserName: sp("Alice"), NowPlayingItemId: sp("series-1"), NowPlayingItemName: sp("Lost"), SeriesName: sp("Lost"),
			EpisodeId: sp("ep1"), Client: sp("Web"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(1200), ActivityDateInserted: &d, Source: "watchdog"},
		{Id: "p5", UserId: sp("u2"), UserName: sp("Bob"), NowPlayingItemId: sp("alb1"), NowPlayingItemName: sp("Greatest Hits"),
			EpisodeId: sp("tr1"), Client: sp("Android"), PlayMethod: sp("DirectPlay"), PlaybackDuration: i64(180), ActivityDateInserted: &dEp, Source: "watchdog"},
	}))
}

// ip returns a pointer to an int (unique helper name to avoid collisions).
func ip(n int) *int { return &n }

// richStats builds a handler over a rich-seeded DB and returns a fresh engine.
func richStats(t *testing.T) (*handler.StatsFrontendHandler, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	seedRichStats(t, db)
	gin.SetMode(gin.TestMode)
	return handler.NewStatsFrontendHandler(db, repository.New(db)), gin.New()
}

// getOK issues a GET and requires 200, returning the recorder.
func getOK(t *testing.T, r *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := do(t, r, target)
	if w.Code != http.StatusOK {
		t.Fatalf("GET %s status = %d, body %s", target, w.Code, w.Body.String())
	}
	return w
}

// postOK issues a POST with a JSON body and requires 200.
func postOK(t *testing.T, r *gin.Engine, target, body string) *httptest.ResponseRecorder {
	t.Helper()
	w := postJSON(t, r, target, body)
	if w.Code != http.StatusOK {
		t.Fatalf("POST %s status = %d, body %s", target, w.Code, w.Body.String())
	}
	return w
}

// ---------------------------------------------------------------------------
// GetWatchStatisticsOverTime
// ---------------------------------------------------------------------------

func TestGetWatchStatisticsOverTime(t *testing.T) {
	h, r := richStats(t)
	r.GET("/w", h.GetWatchStatisticsOverTime)
	w := getOK(t, r, "/w?days=0")

	var rows []struct {
		Date     string `json:"date"`
		Plays    int    `json:"plays"`
		Duration int    `json:"duration"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// All 5 plays land on the same calendar day (2026-07-12).
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 day bucket", len(rows))
	}
	if rows[0].Plays != 5 {
		t.Errorf("plays = %d, want 5", rows[0].Plays)
	}
	// Total watch: 600+600+300+1200+180 = 2880s → floor(2880/60) = 48 min.
	if rows[0].Duration != 48 {
		t.Errorf("duration = %d, want 48 min", rows[0].Duration)
	}
}

func TestGetWatchStatisticsOverTime_FilterByUser(t *testing.T) {
	h, r := richStats(t)
	r.GET("/w", h.GetWatchStatisticsOverTime)
	w := getOK(t, r, "/w?days=0&userId=u2")

	var rows []struct {
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1", len(rows))
	}
	// Bob: p3 (300s) + p5 (180s) = 480s → 8 min, 2 plays.
	if rows[0].Plays != 2 || rows[0].Duration != 8 {
		t.Errorf("bob = {plays:%d dur:%d}, want {2, 8}", rows[0].Plays, rows[0].Duration)
	}
}

// ---------------------------------------------------------------------------
// GetUserStats
// ---------------------------------------------------------------------------

func TestGetUserStats_All(t *testing.T) {
	h, r := richStats(t)
	r.GET("/us", h.GetUserStats)
	w := getOK(t, r, "/us")

	var rows []struct {
		UserId         string `json:"UserId"`
		UserName       string `json:"UserName"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalWatchTime int    `json:"TotalWatchTime"`
		UniqueItems    int    `json:"UniqueItems"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 2 {
		t.Fatalf("rows = %d, want 2", len(rows))
	}
	// Ordered by plays desc → Alice (3 plays) first.
	if rows[0].UserId != "u1" || rows[0].TotalPlays != 3 {
		t.Errorf("rows[0] = %+v, want u1 with 3 plays", rows[0])
	}
	// Alice watch: 600+600+1200 = 2400s → 40 min. Unique items: m1, series-1 = 2.
	if rows[0].TotalWatchTime != 40 || rows[0].UniqueItems != 2 {
		t.Errorf("alice = {watch:%d unique:%d}, want {40, 2}", rows[0].TotalWatchTime, rows[0].UniqueItems)
	}
}

func TestGetUserStats_Single(t *testing.T) {
	h, r := richStats(t)
	r.GET("/us", h.GetUserStats)
	w := getOK(t, r, "/us?userId=u2")

	var row struct {
		UserId         string `json:"UserId"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalWatchTime int    `json:"TotalWatchTime"`
		MostUsedClient *string `json:"MostUsedClient"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &row)
	if row.UserId != "u2" || row.TotalPlays != 2 {
		t.Errorf("row = %+v, want u2 with 2 plays", row)
	}
	// Bob watch: 300+180 = 480s → 8 min. Client Android used twice.
	if row.TotalWatchTime != 8 {
		t.Errorf("watch = %d, want 8", row.TotalWatchTime)
	}
	if row.MostUsedClient == nil || *row.MostUsedClient != "Android" {
		t.Errorf("MostUsedClient = %v, want Android", row.MostUsedClient)
	}
}

// ---------------------------------------------------------------------------
// GetAllUserActivity / GetUserActivity / GetUserActivityByDate
// ---------------------------------------------------------------------------

func TestGetAllUserActivity(t *testing.T) {
	h, r := richStats(t)
	r.GET("/aua", h.GetAllUserActivity)
	w := getOK(t, r, "/aua")

	var rows []struct {
		UserId       string `json:"UserId"`
		PlayDuration int64  `json:"PlayDuration"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// 5 playback rows total.
	if len(rows) != 5 {
		t.Errorf("rows = %d, want 5", len(rows))
	}
}

func TestGetAllUserActivity_FilterClient(t *testing.T) {
	h, r := richStats(t)
	r.GET("/aua", h.GetAllUserActivity)
	w := getOK(t, r, "/aua?client=Android")

	var rows []struct {
		UserId string `json:"UserId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// Android rows: p3, p5 = 2.
	if len(rows) != 2 {
		t.Errorf("android rows = %d, want 2", len(rows))
	}
}

func TestGetUserActivity_RequiresUserId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/ua", h.GetUserActivity)
	if w := do(t, r, "/ua"); w.Code != http.StatusBadRequest {
		t.Errorf("no userId status = %d, want 400", w.Code)
	}
}

func TestGetUserActivity(t *testing.T) {
	h, r := richStats(t)
	r.GET("/ua", h.GetUserActivity)
	w := getOK(t, r, "/ua?userId=u1")

	var rows []struct {
		UserId string `json:"UserId"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// Alice: p1, p2, p4 = 3.
	if len(rows) != 3 {
		t.Errorf("alice rows = %d, want 3", len(rows))
	}
}

func TestGetUserActivityByDate(t *testing.T) {
	h, r := richStats(t)
	r.GET("/uad", h.GetUserActivityByDate)
	w := getOK(t, r, "/uad?userId=u1")

	var rows []struct {
		Date     string `json:"date"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Fatalf("rows = %d, want 1 day", len(rows))
	}
	// Alice on that day: 3 plays, 600+600+1200 = 2400s → 40 min.
	if rows[0].Count != 3 || rows[0].Duration != 40 {
		t.Errorf("row = %+v, want count 3 duration 40", rows[0])
	}
}

// ---------------------------------------------------------------------------
// GetLibraries / GetLibraryStats / GetLibraryItems / GetLibraryOverview
// ---------------------------------------------------------------------------

func TestGetLibraries(t *testing.T) {
	h, r := richStats(t)
	r.GET("/libs", h.GetLibraries)
	w := getOK(t, r, "/libs")

	var rows []struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		ItemCount      int    `json:"ItemCount"`
		EpisodeCount   int    `json:"EpisodeCount"`
		SeasonCount    int    `json:"SeasonCount"`
		TotalPlayCount int    `json:"TotalPlayCount"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(rows) != 3 {
		t.Fatalf("libraries = %d, want 3", len(rows))
	}
	byId := map[string]int{}
	for i, row := range rows {
		byId[row.Id] = i
	}
	movies := rows[byId["lib-movies"]]
	// lib-movies: 1 top-level item (m1), 3 plays (p1,p2,p3), watch 600+600+300=1500s → 25 min.
	if movies.ItemCount != 1 || movies.TotalPlayCount != 3 || movies.TotalWatchTime != 25 {
		t.Errorf("movies = %+v, want item 1 plays 3 watch 25", movies)
	}
	tv := rows[byId["lib-tv"]]
	// lib-tv: 1 series item, 1 season, 1 episode, 1 play (p4) via EpisodeId, watch 1200s → 20 min.
	if tv.EpisodeCount != 1 || tv.SeasonCount != 1 || tv.TotalPlayCount != 1 || tv.TotalWatchTime != 20 {
		t.Errorf("tv = %+v, want episode 1 season 1 plays 1 watch 20", tv)
	}
}

func TestGetLibraryStats_RequiresId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/ls", h.GetLibraryStats)
	if w := do(t, r, "/ls"); w.Code != http.StatusBadRequest {
		t.Errorf("no libraryId status = %d, want 400", w.Code)
	}
}

func TestGetLibraryStats(t *testing.T) {
	h, r := richStats(t)
	r.GET("/ls", h.GetLibraryStats)
	w := getOK(t, r, "/ls?libraryId=lib-movies")

	var got struct {
		Name           string `json:"Name"`
		TotalItems     int    `json:"TotalItems"`
		TotalPlayCount int    `json:"TotalPlayCount"`
		TotalWatchTime int    `json:"TotalWatchTime"`
		MostPlayedItem *struct {
			Id        string `json:"Id"`
			PlayCount int    `json:"PlayCount"`
		} `json:"MostPlayedItem"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Name != "Movies" || got.TotalItems != 1 {
		t.Errorf("got = %+v, want Movies with 1 item", got)
	}
	if got.TotalPlayCount != 3 || got.TotalWatchTime != 25 {
		t.Errorf("plays/watch = %d/%d, want 3/25", got.TotalPlayCount, got.TotalWatchTime)
	}
	if got.MostPlayedItem == nil || got.MostPlayedItem.Id != "m1" || got.MostPlayedItem.PlayCount != 3 {
		t.Errorf("mostPlayed = %+v, want m1 with 3 plays", got.MostPlayedItem)
	}
}

func TestGetLibraryItems_RequiresId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/li", h.GetLibraryItems)
	if w := do(t, r, "/li"); w.Code != http.StatusBadRequest {
		t.Errorf("no libraryId status = %d, want 400", w.Code)
	}
}

func TestGetLibraryItems(t *testing.T) {
	h, r := richStats(t)
	r.GET("/li", h.GetLibraryItems)
	w := getOK(t, r, "/li?libraryId=lib-movies")

	var rows []struct {
		Id        string `json:"Id"`
		PlayCount int    `json:"PlayCount"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 || rows[0].Id != "m1" || rows[0].PlayCount != 3 {
		t.Errorf("items = %+v, want m1 with 3 plays", rows)
	}
}

func TestGetLibraryOverview(t *testing.T) {
	h, r := richStats(t)
	r.GET("/lo", h.GetLibraryOverview)
	w := getOK(t, r, "/lo")

	var rows []struct {
		Id         string `json:"Id"`
		TotalItems int    `json:"TotalItems"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 3 {
		t.Errorf("overview rows = %d, want 3", len(rows))
	}
}

func TestGetLibraryCardStatsGET(t *testing.T) {
	h, r := richStats(t)
	r.GET("/lcs", h.GetLibraryCardStatsGET)
	w := getOK(t, r, "/lcs")

	var rows []struct {
		Id        string `json:"Id"`
		ItemCount int    `json:"ItemCount"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 3 {
		t.Errorf("card stats rows = %d, want 3", len(rows))
	}
}

func TestGetLibraryCardStatsPOST_RequiresId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/lcs", h.GetLibraryCardStatsPOST)
	if w := postJSON(t, r, "/lcs", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("no libraryid status = %d, want 400", w.Code)
	}
}

func TestGetLibraryCardStatsPOST(t *testing.T) {
	h, r := richStats(t)
	r.POST("/lcs", h.GetLibraryCardStatsPOST)
	w := postOK(t, r, "/lcs", `{"libraryid":"lib-movies"}`)

	var got struct {
		Id        string `json:"Id"`
		Name      string `json:"Name"`
		ItemCount int    `json:"ItemCount"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Id != "lib-movies" || got.Name != "Movies" || got.ItemCount != 1 {
		t.Errorf("got = %+v, want lib-movies Movies item 1", got)
	}
}

// ---------------------------------------------------------------------------
// GetItemDetails
// ---------------------------------------------------------------------------

func TestGetItemDetails_RequiresId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/id", h.GetItemDetails)
	if w := do(t, r, "/id"); w.Code != http.StatusBadRequest {
		t.Errorf("no itemId status = %d, want 400", w.Code)
	}
}

func TestGetItemDetails_Movie(t *testing.T) {
	h, r := richStats(t)
	r.GET("/id", h.GetItemDetails)
	w := getOK(t, r, "/id?itemId=m1")

	var got struct {
		Item struct {
			Id   string `json:"Id"`
			Name string `json:"Name"`
			Type string `json:"Type"`
		} `json:"item"`
		Stats struct {
			TotalPlays     int `json:"TotalPlays"`
			TotalWatchTime int `json:"TotalWatchTime"`
			UniqueUsers    int `json:"UniqueUsers"`
		} `json:"stats"`
		Users []struct {
			UserId    string `json:"UserId"`
			PlayCount int    `json:"PlayCount"`
		} `json:"users"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if got.Item.Id != "m1" || got.Item.Type != "Movie" {
		t.Errorf("item = %+v, want m1 Movie", got.Item)
	}
	// 3 plays. Watch time is per-row floor(sec/60): 10+10+5 = 25 min.
	if got.Stats.TotalPlays != 3 || got.Stats.TotalWatchTime != 25 || got.Stats.UniqueUsers != 2 {
		t.Errorf("stats = %+v, want plays 3 watch 25 users 2", got.Stats)
	}
	// Alice (2 plays) ranks first.
	if len(got.Users) == 0 || got.Users[0].UserId != "u1" || got.Users[0].PlayCount != 2 {
		t.Errorf("users[0] = %+v, want u1 with 2 plays", got.Users)
	}
}

func TestGetItemDetails_Audio(t *testing.T) {
	h, r := richStats(t)
	r.GET("/id", h.GetItemDetails)
	w := getOK(t, r, "/id?itemId=tr1")

	var got struct {
		Item struct {
			Id   string `json:"Id"`
			Type string `json:"Type"`
		} `json:"item"`
		Stats struct {
			TotalPlays int `json:"TotalPlays"`
		} `json:"stats"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.Item.Id != "tr1" || got.Item.Type != "Audio" {
		t.Errorf("item = %+v, want tr1 Audio", got.Item)
	}
	// Track has one play (p5 via EpisodeId).
	if got.Stats.TotalPlays != 1 {
		t.Errorf("track plays = %d, want 1", got.Stats.TotalPlays)
	}
}

// ---------------------------------------------------------------------------
// GetGenreStats / GetViewsByLibraryType
// ---------------------------------------------------------------------------

func TestGetGenreStats_ByLibrary(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gs", h.GetGenreStats)
	w := getOK(t, r, "/gs?libraryId=lib-movies")

	var rows []struct {
		Genre string `json:"Genre"`
		Count int    `json:"Count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 || rows[0].Genre != "Action" || rows[0].Count != 1 {
		t.Errorf("genres = %+v, want single Action count 1", rows)
	}
}

func TestGetGenreStats_ByUser(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gs", h.GetGenreStats)
	w := getOK(t, r, "/gs?userId=u1")

	var rows []struct {
		Genre     string `json:"Genre"`
		PlayCount int    `json:"PlayCount"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	byGenre := map[string]int{}
	for _, row := range rows {
		byGenre[row.Genre] = row.PlayCount
	}
	// Alice plays: 2 on m1 (Action), 1 on series-1 (Drama).
	if byGenre["Action"] != 2 {
		t.Errorf("Action plays = %d, want 2", byGenre["Action"])
	}
	if byGenre["Drama"] != 1 {
		t.Errorf("Drama plays = %d, want 1", byGenre["Drama"])
	}
}

func TestGetViewsByLibraryType(t *testing.T) {
	h, r := richStats(t)
	r.GET("/vlt", h.GetViewsByLibraryType)
	// days window includes the seeded recent rows only if within 30 days of run;
	// the seed date is fixed, so use a large window to be safe.
	w := getOK(t, r, "/vlt?days=100000")

	var got map[string]int
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Movie plays: p1,p2,p3 on m1 = 3. Series: p4 on series-1 = 1.
	// Audio: p5's NowPlayingItemId=alb1 has no jf_library_items row → Type NULL → "Other".
	if got["Movie"] != 3 {
		t.Errorf("Movie = %d, want 3", got["Movie"])
	}
	if got["Series"] != 1 {
		t.Errorf("Series = %d, want 1", got["Series"])
	}
}

// ---------------------------------------------------------------------------
// GetLibraryAlbums / GetLibraryArtists / GetArtistAlbums / GetAlbumTracks
// ---------------------------------------------------------------------------

func TestGetLibraryArtists(t *testing.T) {
	h, r := richStats(t)
	r.GET("/la", h.GetLibraryArtists)
	w := getOK(t, r, "/la?libraryId=lib-music")

	var rows []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Errorf("artists = %d, want 1", len(rows))
	}
}

func TestGetLibraryAlbums(t *testing.T) {
	h, r := richStats(t)
	r.GET("/lal", h.GetLibraryAlbums)
	w := getOK(t, r, "/lal?libraryId=lib-music")

	var rows []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Errorf("albums = %d, want 1", len(rows))
	}
}

func TestGetArtistAlbums_RequiresParams(t *testing.T) {
	h, r := richStats(t)
	r.GET("/aa", h.GetArtistAlbums)
	if w := do(t, r, "/aa?libraryId=lib-music"); w.Code != http.StatusBadRequest {
		t.Errorf("missing artistId status = %d, want 400", w.Code)
	}
}

func TestGetAlbumTracks_RequiresAlbumId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/at", h.GetAlbumTracks)
	if w := do(t, r, "/at"); w.Code != http.StatusBadRequest {
		t.Errorf("missing albumId status = %d, want 400", w.Code)
	}
}

func TestGetAlbumTracks(t *testing.T) {
	h, r := richStats(t)
	r.GET("/at", h.GetAlbumTracks)
	w := getOK(t, r, "/at?albumId=alb1")

	var rows []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 1 {
		t.Errorf("tracks = %d, want 1", len(rows))
	}
}

// ---------------------------------------------------------------------------
// GetActivityTimeline / GetMostViewedLibraries / GetGlobalUserStats /
// GetUserLastPlayed / GetLibraryLastPlayed
// ---------------------------------------------------------------------------

func TestGetActivityTimeline(t *testing.T) {
	h, r := richStats(t)
	r.GET("/tl", h.GetActivityTimeline)
	w := getOK(t, r, "/tl?year=2026&month=7")

	var rows []struct {
		Duration int `json:"Duration"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// All 5 plays fall in July 2026.
	if len(rows) != 5 {
		t.Errorf("timeline rows = %d, want 5", len(rows))
	}
}

func TestGetActivityTimeline_OtherMonthEmpty(t *testing.T) {
	h, r := richStats(t)
	r.GET("/tl", h.GetActivityTimeline)
	w := getOK(t, r, "/tl?year=2020&month=1")

	var rows []interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	if len(rows) != 0 {
		t.Errorf("Jan 2020 rows = %d, want 0", len(rows))
	}
}

func TestGetMostViewedLibraries(t *testing.T) {
	h, r := richStats(t)
	r.POST("/mvl", h.GetMostViewedLibraries)
	w := postOK(t, r, "/mvl", `{"days":100000}`)

	var rows []struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	byName := map[string]int{}
	for _, row := range rows {
		byName[row.Name] = row.Count
	}
	// Movies library: p1,p2,p3 = 3. Shows: p4 = 1. Music: p5 = 1.
	if byName["Movies"] != 3 {
		t.Errorf("Movies count = %d, want 3", byName["Movies"])
	}
	if byName["Shows"] != 1 {
		t.Errorf("Shows count = %d, want 1", byName["Shows"])
	}
}

func TestGetGlobalUserStats_RequiresUserId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/gus", h.GetGlobalUserStats)
	if w := postJSON(t, r, "/gus", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("no userid status = %d, want 400", w.Code)
	}
}

func TestGetGlobalUserStats(t *testing.T) {
	h, r := richStats(t)
	r.POST("/gus", h.GetGlobalUserStats)
	// Large hours window to include fixed seed date.
	w := postOK(t, r, "/gus", `{"userid":"u1","hours":8760000}`)

	var got struct {
		TotalPlays     int `json:"TotalPlays"`
		TotalWatchTime int `json:"TotalWatchTime"`
		UniqueItems    int `json:"UniqueItems"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	// Alice: 3 plays, 40 min, unique NowPlayingItemId = {m1, series-1} = 2.
	if got.TotalPlays != 3 || got.TotalWatchTime != 40 || got.UniqueItems != 2 {
		t.Errorf("got = %+v, want plays 3 watch 40 unique 2", got)
	}
}

func TestGetUserLastPlayed_RequiresUserId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/ulp", h.GetUserLastPlayed)
	if w := postJSON(t, r, "/ulp", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("no userid status = %d, want 400", w.Code)
	}
}

func TestGetUserLastPlayed(t *testing.T) {
	h, r := richStats(t)
	r.POST("/ulp", h.GetUserLastPlayed)
	w := postOK(t, r, "/ulp", `{"userid":"u1"}`)

	var rows []map[string]interface{}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// Alice has 3 plays.
	if len(rows) != 3 {
		t.Errorf("rows = %d, want 3", len(rows))
	}
}

func TestGetLibraryLastPlayed_RequiresId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/llp", h.GetLibraryLastPlayed)
	if w := postJSON(t, r, "/llp", `{}`); w.Code != http.StatusBadRequest {
		t.Errorf("no libraryid status = %d, want 400", w.Code)
	}
}

func TestGetLibraryLastPlayed(t *testing.T) {
	h, r := richStats(t)
	r.POST("/llp", h.GetLibraryLastPlayed)
	w := postOK(t, r, "/llp", `{"libraryid":"lib-movies"}`)

	var rows []struct {
		NowPlayingItemName *string `json:"NowPlayingItemName"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// m1 has 3 plays whose NowPlayingItemId matches lib-movies items.
	if len(rows) != 3 {
		t.Errorf("rows = %d, want 3", len(rows))
	}
}

// ---------------------------------------------------------------------------
// GetPlaybackMethodStats / GetPlaybacksByLibraryOverTime / GetPlaybacksScatter
// ---------------------------------------------------------------------------

func TestGetPlaybackMethodStats(t *testing.T) {
	h, r := richStats(t)
	r.POST("/pms", h.GetPlaybackMethodStats)
	w := postOK(t, r, "/pms", `{"days":100000}`)

	var rows []struct {
		Name  *string `json:"Name"`
		Count int     `json:"Count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	byName := map[string]int{}
	for _, row := range rows {
		if row.Name != nil {
			byName[*row.Name] = row.Count
		}
	}
	// DirectPlay: p1,p2,p4,p5 = 4. Transcode: p3 = 1.
	if byName["DirectPlay"] != 4 {
		t.Errorf("DirectPlay = %d, want 4", byName["DirectPlay"])
	}
	if byName["Transcode"] != 1 {
		t.Errorf("Transcode = %d, want 1", byName["Transcode"])
	}
}

func TestGetPlaybacksByLibraryOverTime(t *testing.T) {
	h, r := richStats(t)
	r.GET("/pblot", h.GetPlaybacksByLibraryOverTime)
	w := getOK(t, r, "/pblot?days=0")

	var rows []struct {
		LibraryName string `json:"libraryName"`
		Count       int    `json:"count"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	total := 0
	for _, row := range rows {
		total += row.Count
	}
	// All 5 plays are attributable to a library.
	if total != 5 {
		t.Errorf("total plays = %d, want 5", total)
	}
}

func TestGetPlaybacksScatter(t *testing.T) {
	h, r := richStats(t)
	r.GET("/scat", h.GetPlaybacksScatter)
	w := getOK(t, r, "/scat?days=0")

	var rows []struct {
		Duration int    `json:"duration"`
		Type     string `json:"type"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &rows)
	// 5 plays with PlaybackDuration > 0.
	if len(rows) != 5 {
		t.Errorf("scatter points = %d, want 5", len(rows))
	}
}

// ---------------------------------------------------------------------------
// GetWatchHeatmap
// ---------------------------------------------------------------------------

func TestGetWatchHeatmap(t *testing.T) {
	h, r := richStats(t)
	r.GET("/hm", h.GetWatchHeatmap)
	w := getOK(t, r, "/hm?days=0")

	var got struct {
		Cells []struct {
			Day   int `json:"day"`
			Hour  int `json:"hour"`
			Plays int `json:"plays"`
		} `json:"cells"`
		MaxPlays int `json:"maxPlays"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Full 7x24 grid always returned.
	if len(got.Cells) != 7*24 {
		t.Fatalf("cells = %d, want 168", len(got.Cells))
	}
	total := 0
	for _, cl := range got.Cells {
		total += cl.Plays
	}
	if total != 5 {
		t.Errorf("total plays across grid = %d, want 5", total)
	}
	// 2026-07-12 is a Sunday (DOW 0), hour 14 has p1,p2,p4 = 3 plays → maxPlays 3.
	if got.MaxPlays != 3 {
		t.Errorf("maxPlays = %d, want 3", got.MaxPlays)
	}
}

// ---------------------------------------------------------------------------
// GetUnwatchedContent
// ---------------------------------------------------------------------------

func TestGetUnwatchedContent(t *testing.T) {
	h, r := richStats(t)
	r.GET("/uc", h.GetUnwatchedContent)
	w := getOK(t, r, "/uc?libraryId=lib-tv")

	var got struct {
		Summary struct {
			TotalItems     int `json:"totalItems"`
			UnwatchedItems int `json:"unwatchedItems"`
		} `json:"summary"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// lib-tv has one series item (series-1). It was played via EpisodeId, but the
	// unwatched summary keys on NowPlayingItemId matching the item Id. series-1 IS
	// the NowPlayingItemId of p4, so it counts as watched → 0 unwatched.
	if got.Summary.TotalItems != 1 {
		t.Errorf("totalItems = %d, want 1", got.Summary.TotalItems)
	}
	if got.Summary.UnwatchedItems != 0 {
		t.Errorf("unwatchedItems = %d, want 0 (series-1 watched)", got.Summary.UnwatchedItems)
	}
}

// ---------------------------------------------------------------------------
// GetViewingDiversity
// ---------------------------------------------------------------------------

func TestGetViewingDiversity_AllUsers(t *testing.T) {
	h, r := richStats(t)
	r.GET("/vd", h.GetViewingDiversity)
	w := getOK(t, r, "/vd?days=0")

	var got struct {
		Users []struct {
			UserId      string `json:"userId"`
			UniqueItems int    `json:"uniqueItems"`
			TotalPlays  int    `json:"totalPlays"`
		} `json:"users"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(got.Users) != 2 {
		t.Fatalf("users = %d, want 2", len(got.Users))
	}
	byUser := map[string]int{}
	for _, u := range got.Users {
		byUser[u.UserId] = u.TotalPlays
	}
	// Alice: 3 plays. Bob: 2 plays.
	if byUser["u1"] != 3 || byUser["u2"] != 2 {
		t.Errorf("plays = %+v, want u1:3 u2:2", byUser)
	}
}

func TestGetViewingDiversity_SingleUser(t *testing.T) {
	h, r := richStats(t)
	r.GET("/vd", h.GetViewingDiversity)
	w := getOK(t, r, "/vd?days=0&userId=u1")

	var got struct {
		UserId       string `json:"userId"`
		UniqueItems  int    `json:"uniqueItems"`
		UniqueGenres int    `json:"uniqueGenres"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &got)
	if got.UserId != "u1" {
		t.Errorf("userId = %q, want u1", got.UserId)
	}
	// Alice unique NowPlayingItemId = {m1, series-1} = 2.
	if got.UniqueItems != 2 {
		t.Errorf("uniqueItems = %d, want 2", got.UniqueItems)
	}
}

// ---------------------------------------------------------------------------
// GetBingeStats (no binge in seed → zero sessions but valid shape)
// ---------------------------------------------------------------------------

func TestGetBingeStats_NoBinges(t *testing.T) {
	h, r := richStats(t)
	r.GET("/binge", h.GetBingeStats)
	w := getOK(t, r, "/binge?days=0")

	var got struct {
		TotalBingeSessions int           `json:"totalBingeSessions"`
		TopBingedSeries    []interface{} `json:"topBingedSeries"`
		TopBingeUsers      []interface{} `json:"topBingeUsers"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// Only one episode play → no binge session (needs >= 3 in a series).
	if got.TotalBingeSessions != 0 {
		t.Errorf("totalBingeSessions = %d, want 0", got.TotalBingeSessions)
	}
}

// ---------------------------------------------------------------------------
// GetTimeToWatch
// ---------------------------------------------------------------------------

func TestGetTimeToWatch(t *testing.T) {
	h, r := richStats(t)
	r.GET("/ttw", h.GetTimeToWatch)
	w := getOK(t, r, "/ttw")

	var got struct {
		AvgDaysToWatch    float64       `json:"avgDaysToWatch"`
		Distribution      []interface{} `json:"distribution"`
		SlowestItems      []interface{} `json:"slowestItems"`
	}
	// Items in the seed have no DateCreated, so first_played >= date_added fails
	// and no rows qualify → zeroed response with empty slices. Assert the shape
	// stays valid regardless.
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
}
