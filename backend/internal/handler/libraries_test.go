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

func seedLibraryHandler(t *testing.T, db *gorm.DB) {
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

	must(repos.Library.Upsert(ctx, []models.JFLibrary{
		{Id: "lib-movies", Name: sp("Movies"), CollectionType: sp("movies")},
		{Id: "lib-stale", Name: sp("Stale")},
	}))
	// Archive lib-stale so it is excluded from List.
	must(repos.Library.ArchiveNotIn(ctx, []string{"lib-movies"}))

	movie := "Movie"
	series := "Series"
	must(repos.Item.Upsert(ctx, []models.JFLibraryItem{
		{Id: "m1", Name: sp("Inception"), Type: &movie, ParentId: sp("lib-movies"), Genres: genres},
		{Id: "series-1", Name: sp("Lost"), Type: &series, ParentId: sp("lib-movies"), Genres: genres},
	}))
	must(repos.Season.Upsert(ctx, []models.JFLibrarySeason{
		{Id: "s1", Name: sp("Season 1"), SeriesId: sp("series-1")},
	}))
	must(repos.Episode.Upsert(ctx, []models.JFLibraryEpisode{
		{Id: "e1", Name: sp("Pilot"), SeriesId: sp("series-1"), SeasonId: sp("s1")},
	}))
	must(repos.MusicTrack.Upsert(ctx, []models.JFMusicTrack{
		{Id: "tr-1", Name: sp("Song"), AlbumId: sp("alb-1"), LibraryId: sp("lib-music"), Genres: genres},
	}))
}

func libHandler(t *testing.T) (*handler.LibraryHandler, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	seedLibraryHandler(t, db)
	gin.SetMode(gin.TestMode)
	h := handler.NewLibraryHandler(nil, repository.New(db), db)
	return h, gin.New()
}

func do(t *testing.T, r *gin.Engine, target string) *httptest.ResponseRecorder {
	t.Helper()
	w := httptest.NewRecorder()
	r.ServeHTTP(w, httptest.NewRequest(http.MethodGet, target, nil))
	return w
}

// TestLibraryList verifies archived libraries are excluded.
func TestLibraryList(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/libs", h.List)
	w := do(t, r, "/libs")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var libs []models.JFLibrary
	if err := json.Unmarshal(w.Body.Bytes(), &libs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(libs) != 1 || libs[0].Id != "lib-movies" {
		t.Errorf("libs = %+v, want only lib-movies (stale archived)", libs)
	}
}

// TestLibraryGet_NotFound verifies an unknown id returns 404.
func TestLibraryGet_NotFound(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/libs/:id", h.Get)
	w := do(t, r, "/libs/nope")
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", w.Code)
	}
}

// TestLibraryItems verifies items are listed by parent library.
func TestLibraryItems(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/libs/:id/items", h.Items)
	w := do(t, r, "/libs/lib-movies/items")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var items []models.JFLibraryItem
	if err := json.Unmarshal(w.Body.Bytes(), &items); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(items) != 2 {
		t.Errorf("items = %d, want 2", len(items))
	}
}

// TestLibrarySeasonsAndEpisodes verifies series children resolve.
func TestLibrarySeasonsAndEpisodes(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/libs/:id/seasons", h.Seasons)
	r.GET("/libs/:id/episodes", h.Episodes)

	ws := do(t, r, "/libs/series-1/seasons")
	var seasons []models.JFLibrarySeason
	_ = json.Unmarshal(ws.Body.Bytes(), &seasons)
	if len(seasons) != 1 {
		t.Errorf("seasons = %d, want 1", len(seasons))
	}

	we := do(t, r, "/libs/series-1/episodes")
	var episodes []models.JFLibraryEpisode
	_ = json.Unmarshal(we.Body.Bytes(), &episodes)
	if len(episodes) != 1 {
		t.Errorf("episodes = %d, want 1", len(episodes))
	}
}

// TestLibraryTracksHandler verifies tracks list by library.
func TestLibraryTracksHandler(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/libs/:id/tracks", h.Tracks)
	w := do(t, r, "/libs/lib-music/tracks")
	var tracks []models.JFMusicTrack
	_ = json.Unmarshal(w.Body.Bytes(), &tracks)
	if len(tracks) != 1 || tracks[0].Id != "tr-1" {
		t.Errorf("tracks = %+v, want just tr-1", tracks)
	}
}

// TestGetItem verifies item lookup and 404 on unknown id.
func TestGetItem(t *testing.T) {
	h, r := libHandler(t)
	r.GET("/items/:id", h.GetItem)

	w := do(t, r, "/items/m1")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var item models.JFLibraryItem
	_ = json.Unmarshal(w.Body.Bytes(), &item)
	if item.Id != "m1" {
		t.Errorf("item Id = %q, want m1", item.Id)
	}

	if nf := do(t, r, "/items/ghost"); nf.Code != http.StatusNotFound {
		t.Errorf("unknown item status = %d, want 404", nf.Code)
	}
}

// TestUserHandler verifies List and Get (including 404).
func TestUserHandler(t *testing.T) {
	db := testutil.NewDB(t)
	repos := repository.New(db)
	if err := repos.User.Upsert(t.Context(), []models.JFUser{
		{Id: "u1", Name: sp("Alice")},
		{Id: "u2", Name: sp("Bob")},
	}); err != nil {
		t.Fatalf("seed: %v", err)
	}
	gin.SetMode(gin.TestMode)
	h := handler.NewUserHandler(repos)
	r := gin.New()
	r.GET("/users", h.List)
	r.GET("/users/:id", h.Get)

	w := do(t, r, "/users")
	var users []models.JFUser
	_ = json.Unmarshal(w.Body.Bytes(), &users)
	if len(users) != 2 {
		t.Errorf("users = %d, want 2", len(users))
	}

	one := do(t, r, "/users/u1")
	var u models.JFUser
	_ = json.Unmarshal(one.Body.Bytes(), &u)
	if u.Name == nil || *u.Name != "Alice" {
		t.Errorf("u1 = %+v, want Alice", u)
	}

	if nf := do(t, r, "/users/ghost"); nf.Code != http.StatusNotFound {
		t.Errorf("unknown user status = %d, want 404", nf.Code)
	}
}
