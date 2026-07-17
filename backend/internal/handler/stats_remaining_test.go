package handler_test

// Tests for the six previously-untested stats endpoints:
//   GetGenreUserStats, GetGenreLibraryStats, GetLibraryItemsWithStats,
//   GetLibraryItemsPlayMethodStats, GetLibraryMetadata, GetPlaybackActivity
// and the success path for GetArtistAlbums.
//
// All tests reuse the richStats() helper (seedRichStats dataset):
//   lib-movies / Inception(m1) / Alice×2 + Bob×1
//   lib-tv     / Lost(series-1, ep1) / Alice×1
//   lib-music  / Greatest Hits(alb1, tr1) / Bob×1

import (
	"encoding/json"
	"net/http"
	"testing"
)

// ---------------------------------------------------------------------------
// GetGenreUserStats
// ---------------------------------------------------------------------------

func TestGetGenreUserStats_RequiresUserId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gu", h.GetGenreUserStats)
	w := do(t, r, "/gu")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("no userid: status = %d, want 400", w.Code)
	}
}

func TestGetGenreUserStats_ReturnsGenres(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gu", h.GetGenreUserStats)
	// Alice (u1) watched Inception (Action) twice and Lost/ep1 (Drama) once.
	w := getOK(t, r, "/gu?userid=u1")
	var resp struct {
		Pages   int `json:"pages"`
		Results []struct {
			Genre string `json:"genre"`
			Plays int    `json:"plays"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	if len(resp.Results) != 2 {
		t.Errorf("genres = %d, want 2 (Action + Drama)", len(resp.Results))
	}
	plays := map[string]int{}
	for _, row := range resp.Results {
		plays[row.Genre] = row.Plays
	}
	if plays["Action"] != 2 {
		t.Errorf("Action plays = %d, want 2", plays["Action"])
	}
	if plays["Drama"] != 1 {
		t.Errorf("Drama plays = %d, want 1", plays["Drama"])
	}
}

func TestGetGenreUserStats_TypeFilter(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gu", h.GetGenreUserStats)
	// Alice, Movie type only → only Action (2 plays from Inception).
	w := getOK(t, r, "/gu?userid=u1&type=Movie")
	var resp struct {
		Results []struct {
			Genre string `json:"genre"`
		} `json:"results"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Results) != 1 {
		t.Errorf("genres = %d, want 1 (Action only)", len(resp.Results))
	}
	if len(resp.Results) > 0 && resp.Results[0].Genre != "Action" {
		t.Errorf("genre = %s, want Action", resp.Results[0].Genre)
	}
}

// ---------------------------------------------------------------------------
// GetGenreLibraryStats
// ---------------------------------------------------------------------------

func TestGetGenreLibraryStats_RequiresLibraryId(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gl", h.GetGenreLibraryStats)
	w := do(t, r, "/gl")
	if w.Code != http.StatusBadRequest {
		t.Fatalf("no libraryid: status = %d, want 400", w.Code)
	}
}

func TestGetGenreLibraryStats_ReturnsGenres(t *testing.T) {
	h, r := richStats(t)
	r.GET("/gl", h.GetGenreLibraryStats)
	// lib-movies has only Inception with genre Action; p1+p2+p3 = 3 plays.
	w := getOK(t, r, "/gl?libraryid=lib-movies")
	var resp struct {
		Results []struct {
			Genre string `json:"genre"`
			Plays int    `json:"plays"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	if len(resp.Results) != 1 {
		t.Fatalf("genres = %d, want 1", len(resp.Results))
	}
	if resp.Results[0].Genre != "Action" {
		t.Errorf("genre = %s, want Action", resp.Results[0].Genre)
	}
	if resp.Results[0].Plays != 3 {
		t.Errorf("plays = %d, want 3", resp.Results[0].Plays)
	}
}

// ---------------------------------------------------------------------------
// GetLibraryItemsWithStats
// ---------------------------------------------------------------------------

func TestGetLibraryItemsWithStats_RequiresLibraryId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/liws", h.GetLibraryItemsWithStats)
	w := postJSON(t, r, "/liws", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("no libraryid: status = %d, want 400", w.Code)
	}
}

func TestGetLibraryItemsWithStats_ReturnsItems(t *testing.T) {
	h, r := richStats(t)
	r.POST("/liws", h.GetLibraryItemsWithStats)
	// lib-movies contains Inception; p1+p2+p3 = 3 plays, 1500s total.
	w := postOK(t, r, "/liws", `{"libraryid":"lib-movies"}`)
	var resp struct {
		Results []struct {
			Id          string `json:"Id"`
			Name        string `json:"Name"`
			TimesPlayed int    `json:"times_played"`
			TotalTime   int    `json:"total_play_time"`
		} `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	if len(resp.Results) != 1 {
		t.Fatalf("results = %d, want 1", len(resp.Results))
	}
	item := resp.Results[0]
	if item.Id != "m1" {
		t.Errorf("Id = %s, want m1", item.Id)
	}
	if item.TimesPlayed != 3 {
		t.Errorf("times_played = %d, want 3", item.TimesPlayed)
	}
	// 600+600+300 = 1500s
	if item.TotalTime != 1500 {
		t.Errorf("total_play_time = %d, want 1500", item.TotalTime)
	}
}

// ---------------------------------------------------------------------------
// GetLibraryItemsPlayMethodStats
// ---------------------------------------------------------------------------

func TestGetLibraryItemsPlayMethodStats_RequiresLibraryId(t *testing.T) {
	h, r := richStats(t)
	r.POST("/lipms", h.GetLibraryItemsPlayMethodStats)
	w := postJSON(t, r, "/lipms", `{}`)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("no libraryid: status = %d, want 400", w.Code)
	}
}

func TestGetLibraryItemsPlayMethodStats_Returns200(t *testing.T) {
	h, r := richStats(t)
	r.POST("/lipms", h.GetLibraryItemsPlayMethodStats)
	// No time filter — returns all playback records for lib-movies.
	w := postOK(t, r, "/lipms", `{"libraryid":"lib-movies"}`)
	var resp struct {
		Stats []struct {
			Key         string `json:"Key"`
			Transcodes  int    `json:"Transcodes"`
			DirectPlays int    `json:"DirectPlays"`
		} `json:"stats"`
		Types []struct {
			Id string `json:"Id"`
		} `json:"types"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	// p1 DirectPlay, p2 DirectPlay, p3 Transcode → at least one hour bucket.
	if len(resp.Stats) == 0 {
		t.Errorf("stats is empty, want at least one hour bucket")
	}
	if len(resp.Types) != 2 {
		t.Errorf("types = %d, want 2 (Transcodes + DirectPlays)", len(resp.Types))
	}
}

// ---------------------------------------------------------------------------
// GetLibraryMetadata
// ---------------------------------------------------------------------------

func TestGetLibraryMetadata_ReturnsLibraries(t *testing.T) {
	h, r := richStats(t)
	r.GET("/lm", h.GetLibraryMetadata)
	w := getOK(t, r, "/lm")
	var rows []struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		ItemCount      int    `json:"ItemCount"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	// seedRichStats creates 3 libraries: Movies, Music, Shows.
	if len(rows) != 3 {
		t.Fatalf("libraries = %d, want 3", len(rows))
	}
	counts := map[string]int{}
	for _, row := range rows {
		counts[row.Name] = row.ItemCount
	}
	// Movies has Inception (Movie type, not Season/Folder) → 1 item.
	if counts["Movies"] != 1 {
		t.Errorf("Movies ItemCount = %d, want 1", counts["Movies"])
	}
}

// ---------------------------------------------------------------------------
// GetPlaybackActivity
// ---------------------------------------------------------------------------

func TestGetPlaybackActivity_ReturnsPaginatedRows(t *testing.T) {
	h, r := richStats(t)
	r.GET("/pa", h.GetPlaybackActivity)
	w := getOK(t, r, "/pa?size=50&page=1")
	var resp struct {
		Pages   int              `json:"pages"`
		Results []map[string]any `json:"results"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	// seedRichStats inserts 5 playback rows; size=50 → 1 page.
	if resp.Pages != 1 {
		t.Errorf("pages = %d, want 1", resp.Pages)
	}
	if len(resp.Results) != 5 {
		t.Errorf("results = %d, want 5", len(resp.Results))
	}
}

func TestGetPlaybackActivity_Search(t *testing.T) {
	h, r := richStats(t)
	r.GET("/pa", h.GetPlaybackActivity)
	// Search "inception" should match 3 rows (p1, p2, p3) — 1 page of 50.
	w := getOK(t, r, "/pa?search=inception&size=50")
	var resp struct {
		Results []map[string]any `json:"results"`
	}
	_ = json.Unmarshal(w.Body.Bytes(), &resp)
	if len(resp.Results) != 3 {
		t.Errorf("results = %d, want 3 (Inception plays)", len(resp.Results))
	}
}

// ---------------------------------------------------------------------------
// GetArtistAlbums — success path (error path already covered elsewhere)
// ---------------------------------------------------------------------------

func TestGetArtistAlbums_ReturnsAlbums(t *testing.T) {
	h, r := richStats(t)
	r.GET("/aa", h.GetArtistAlbums)
	// art1 "The Band" in lib-music has one album alb1 with track tr1 (1 play via p5).
	w := getOK(t, r, "/aa?artistId=art1&libraryId=lib-music")
	var rows []struct {
		Id         string `json:"Id"`
		Name       string `json:"Name"`
		TrackCount int    `json:"TrackCount"`
		PlayCount  int    `json:"PlayCount"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &rows); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	if len(rows) != 1 {
		t.Fatalf("albums = %d, want 1", len(rows))
	}
	if rows[0].Id != "alb1" {
		t.Errorf("Id = %s, want alb1", rows[0].Id)
	}
	if rows[0].TrackCount != 1 {
		t.Errorf("TrackCount = %d, want 1", rows[0].TrackCount)
	}
	if rows[0].PlayCount != 1 {
		t.Errorf("PlayCount = %d, want 1", rows[0].PlayCount)
	}
}
