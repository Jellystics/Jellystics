package sync

import (
	"encoding/json"
	"testing"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

// TestGenresJSON verifies genres are title-cased and JSON-encoded. This guards
// the UTF-8 titleCase fix: multi-byte letters must survive intact.
func TestGenresJSON(t *testing.T) {
	tests := []struct {
		name string
		in   []string
		want []string
	}{
		{"empty", []string{}, []string{}},
		{"single lower", []string{"rock"}, []string{"Rock"}},
		{"mixed case", []string{"HIP HOP", "science fiction"}, []string{"Hip Hop", "Science Fiction"}},
		{"utf8 preserved", []string{"éléctronique"}, []string{"Éléctronique"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			raw := genresJSON(tt.in)
			var got []string
			if err := json.Unmarshal(raw, &got); err != nil {
				t.Fatalf("unmarshal: %v (raw %s)", err, raw)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("genresJSON(%v) = %v, want %v", tt.in, got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Fatalf("genresJSON(%v)[%d] = %q, want %q", tt.in, i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TestMapLibraryItem verifies a Jellyfin Item maps to a JFLibraryItem with the
// parent id injected, image tags pulled from the ImageTags map, the primary blur
// hash resolved, and genres title-cased.
func TestMapLibraryItem(t *testing.T) {
	item := jellyfin.Item{
		Id:       "movie-1",
		Name:     "Inception",
		ServerId: "srv",
		Type:     "Movie",
		Genres:   []string{"sci-fi", "thriller"},
		ImageTags: jellyfin.ImageTags{
			"Primary": "ptag",
			"Banner":  "btag",
			"Logo":    "ltag",
			"Thumb":   "ttag",
		},
		ImageBlurHashes: jellyfin.ImageBlurHashes{
			Primary: map[string]string{"ptag": "hash123"},
		},
		BackdropImageTags: []string{"bd1", "bd2"},
		AlbumArtists:      []jellyfin.NamedItem{{Id: "artist-1", Name: "Nolan"}},
		AlbumArtist:       sp("Nolan"),
	}

	got := mapLibraryItem(item, "lib-movies")

	if got.Id != "movie-1" {
		t.Errorf("Id = %q, want movie-1", got.Id)
	}
	if got.Name == nil || *got.Name != "Inception" {
		t.Errorf("Name = %v, want Inception", got.Name)
	}
	if got.ParentId == nil || *got.ParentId != "lib-movies" {
		t.Errorf("ParentId = %v, want lib-movies", got.ParentId)
	}
	if got.ImageTagsPrimary == nil || *got.ImageTagsPrimary != "ptag" {
		t.Errorf("ImageTagsPrimary = %v, want ptag", got.ImageTagsPrimary)
	}
	if got.ImageTagsBanner == nil || *got.ImageTagsBanner != "btag" {
		t.Errorf("ImageTagsBanner = %v, want btag", got.ImageTagsBanner)
	}
	if got.BackdropImageTags == nil || *got.BackdropImageTags != "bd1" {
		t.Errorf("BackdropImageTags = %v, want bd1 (first tag)", got.BackdropImageTags)
	}
	if got.PrimaryImageHash == nil || *got.PrimaryImageHash != "hash123" {
		t.Errorf("PrimaryImageHash = %v, want hash123", got.PrimaryImageHash)
	}
	if got.ArtistId == nil || *got.ArtistId != "artist-1" {
		t.Errorf("ArtistId = %v, want artist-1 (first album artist)", got.ArtistId)
	}
	var genres []string
	if err := json.Unmarshal(got.Genres, &genres); err != nil {
		t.Fatalf("genres unmarshal: %v", err)
	}
	if len(genres) != 2 || genres[0] != "Sci-fi" || genres[1] != "Thriller" {
		t.Errorf("Genres = %v, want [Sci-fi Thriller]", genres)
	}
}

// TestMapLibraryItem_MissingImages verifies absent image tags and blur hashes
// map to nil pointers rather than empty strings.
func TestMapLibraryItem_MissingImages(t *testing.T) {
	item := jellyfin.Item{Id: "x", Name: "n", Type: "Movie"}
	got := mapLibraryItem(item, "parent")

	if got.ImageTagsPrimary != nil {
		t.Errorf("ImageTagsPrimary = %v, want nil", got.ImageTagsPrimary)
	}
	if got.BackdropImageTags != nil {
		t.Errorf("BackdropImageTags = %v, want nil", got.BackdropImageTags)
	}
	if got.PrimaryImageHash != nil {
		t.Errorf("PrimaryImageHash = %v, want nil", got.PrimaryImageHash)
	}
	if got.ArtistId != nil {
		t.Errorf("ArtistId = %v, want nil", got.ArtistId)
	}
}

// TestMapEpisode verifies an Episode item maps to a JFLibraryEpisode carrying
// series/season linkage, and produces a JFItemInfo only when MediaSources exist.
func TestMapEpisode(t *testing.T) {
	item := jellyfin.Item{
		Id:           "ep-1",
		Name:         "Pilot",
		Type:         "Episode",
		SeriesId:     sp("series-1"),
		SeasonId:     sp("season-1"),
		SeriesName:   sp("Lost"),
		IndexNumber:  intp(1),
		RunTimeTicks: i64p(100),
		MediaSources: []jellyfin.MediaSource{
			{Path: "/media/ep1.mkv", Size: i64p(500), Bitrate: i64p(128)},
		},
	}

	ep, info := mapEpisode(item)

	if ep.Id != "ep-1" {
		t.Errorf("Id = %q, want ep-1", ep.Id)
	}
	if ep.SeriesId == nil || *ep.SeriesId != "series-1" {
		t.Errorf("SeriesId = %v, want series-1", ep.SeriesId)
	}
	if ep.SeasonId == nil || *ep.SeasonId != "season-1" {
		t.Errorf("SeasonId = %v, want season-1", ep.SeasonId)
	}
	if info == nil {
		t.Fatal("expected JFItemInfo when MediaSources present")
	}
	if info.Path == nil || *info.Path != "/media/ep1.mkv" {
		t.Errorf("info.Path = %v, want /media/ep1.mkv", info.Path)
	}
	if info.Type == nil || *info.Type != "Episode" {
		t.Errorf("info.Type = %v, want Episode (override)", info.Type)
	}
}

// TestMapEpisode_NoMediaSources verifies info is nil when the item has no media
// sources.
func TestMapEpisode_NoMediaSources(t *testing.T) {
	item := jellyfin.Item{Id: "ep-2", Name: "No Media", Type: "Episode"}
	ep, info := mapEpisode(item)
	if ep.Id != "ep-2" {
		t.Errorf("Id = %q, want ep-2", ep.Id)
	}
	if info != nil {
		t.Errorf("info = %+v, want nil (no MediaSources)", info)
	}
}

// TestMapSeason verifies a Season item maps its series linkage and first parent
// backdrop tag.
func TestMapSeason(t *testing.T) {
	item := jellyfin.Item{
		Id:                      "season-1",
		Name:                    "Season 1",
		Type:                    "Season",
		SeriesId:                sp("series-1"),
		SeriesName:              sp("Lost"),
		IndexNumber:             intp(1),
		ParentBackdropImageTags: []string{"pbd1", "pbd2"},
	}
	got := mapSeason(item)

	if got.Id != "season-1" {
		t.Errorf("Id = %q, want season-1", got.Id)
	}
	if got.SeriesId == nil || *got.SeriesId != "series-1" {
		t.Errorf("SeriesId = %v, want series-1", got.SeriesId)
	}
	if got.ParentBackdropImageTags == nil || *got.ParentBackdropImageTags != "pbd1" {
		t.Errorf("ParentBackdropImageTags = %v, want pbd1", got.ParentBackdropImageTags)
	}
}

// TestMapItemInfo_StreamsFromSource verifies media streams are taken from the
// MediaSource when present, and encoded as JSON.
func TestMapItemInfo_StreamsFromSource(t *testing.T) {
	item := jellyfin.Item{Id: "i1", Name: "Movie", Type: "Movie"}
	ms := jellyfin.MediaSource{
		Path:    "/m.mkv",
		Size:    i64p(1000),
		Bitrate: i64p(256),
		MediaStreams: []jellyfin.MediaStream{
			{Codec: "h264", Type: "Video"},
		},
	}
	got := mapItemInfo(item, ms, "")

	if got.Type == nil || *got.Type != "Movie" {
		t.Errorf("Type = %v, want Movie (falls back to item.Type)", got.Type)
	}
	if got.Path == nil || *got.Path != "/m.mkv" {
		t.Errorf("Path = %v", got.Path)
	}
	var streams []jellyfin.MediaStream
	if err := json.Unmarshal(got.MediaStreams, &streams); err != nil {
		t.Fatalf("streams unmarshal: %v", err)
	}
	if len(streams) != 1 || streams[0].Codec != "h264" {
		t.Errorf("streams = %v, want one h264 stream", streams)
	}
}

// TestMapItemInfo_StreamsFallbackToItem verifies that when the MediaSource has
// no streams, the item-level streams are used instead.
func TestMapItemInfo_StreamsFallbackToItem(t *testing.T) {
	item := jellyfin.Item{
		Id: "i2", Name: "Movie", Type: "Movie",
		MediaStreams: []jellyfin.MediaStream{{Codec: "aac", Type: "Audio"}},
	}
	ms := jellyfin.MediaSource{Path: "/m2.mkv"}
	got := mapItemInfo(item, ms, "Movie")

	var streams []jellyfin.MediaStream
	if err := json.Unmarshal(got.MediaStreams, &streams); err != nil {
		t.Fatalf("streams unmarshal: %v", err)
	}
	if len(streams) != 1 || streams[0].Codec != "aac" {
		t.Errorf("streams = %v, want fallback aac stream", streams)
	}
}

func intp(v int) *int     { return &v }
func i64p(v int64) *int64 { return &v }
