package sync

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
)

func sp(s string) *string { return &s }

// TestItemKey verifies the media-switch identity used to detect when a session
// moves to a different item. Movies carry only a parent id; TV episodes and
// music tracks carry parent|child.
func TestItemKey(t *testing.T) {
	tests := []struct {
		name    string
		parent  string
		episode *string
		want    string
	}{
		{"movie: parent only", "movie-1", nil, "movie-1"},
		{"tv: series|episode", "series-1", sp("ep-9"), "series-1|ep-9"},
		{"music: album|track", "album-1", sp("track-3"), "album-1|track-3"},
		{"empty parent, no child", "", nil, ""},
		{"empty parent, with child", "", sp("track-3"), "|track-3"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := itemKey(tt.parent, tt.episode); got != tt.want {
				t.Fatalf("itemKey(%q, %v) = %q, want %q", tt.parent, tt.episode, got, tt.want)
			}
		})
	}
}

// TestItemKey_SwitchDetection documents the actual comparison used at both call
// sites: a switch fires only when the previous key is non-empty and differs.
func TestItemKey_SwitchDetection(t *testing.T) {
	switched := func(prevParent string, prevEp *string, newParent string, newEp *string) bool {
		prevKey := itemKey(prevParent, prevEp)
		newKey := itemKey(newParent, newEp)
		return prevKey != "" && prevKey != newKey
	}

	// Same movie → no switch.
	if switched("movie-1", nil, "movie-1", nil) {
		t.Fatal("same movie should not be a switch")
	}
	// Different movie → switch.
	if !switched("movie-1", nil, "movie-2", nil) {
		t.Fatal("different movie should be a switch")
	}
	// Same album, next track → switch (track-level accuracy).
	if !switched("album-1", sp("track-1"), "album-1", sp("track-2")) {
		t.Fatal("next track in same album should be a switch")
	}
	// Same series, next episode → switch.
	if !switched("series-1", sp("ep-1"), "series-1", sp("ep-2")) {
		t.Fatal("next episode should be a switch")
	}
	// No prior baseline (empty prev key) → never a switch (fresh session).
	if switched("", nil, "movie-1", nil) {
		t.Fatal("empty previous key must not count as a switch")
	}
}

func TestTitleCase(t *testing.T) {
	tests := []struct{ in, want string }{
		{"", ""},
		{"rock", "Rock"},
		{"HIP HOP", "Hip Hop"},
		{"science fiction", "Science Fiction"},
		{"  spaced   out  ", "Spaced Out"},
		{"éléctronique", "Éléctronique"},
	}
	for _, tt := range tests {
		if got := titleCase(tt.in); got != tt.want {
			t.Fatalf("titleCase(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestStrPtr(t *testing.T) {
	if strPtr("") != nil {
		t.Fatal("strPtr(\"\") should be nil")
	}
	p := strPtr("x")
	if p == nil || *p != "x" {
		t.Fatalf("strPtr(\"x\") = %v, want pointer to \"x\"", p)
	}
}

func TestDeref(t *testing.T) {
	if deref(nil) != "" {
		t.Fatal("deref(nil) should be empty string")
	}
	if deref(sp("y")) != "y" {
		t.Fatal("deref(&\"y\") should be \"y\"")
	}
}

// TestMapPluginRow covers the PlaybackReporting plugin row parser: JSON numbers
// arrive as float64, missing rowid rejects the row, short rows are ignored, and
// PlayDuration is preserved as-is (already seconds).
func TestMapPluginRow(t *testing.T) {
	t.Run("valid row", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"42", "2026-07-12 10:00:00", "user-1", "item-1", "Movie",
			"Big Buck Bunny", "DirectPlay", "Jellyfin Web", "Chrome", float64(600),
		}
		got := mapPluginRow(row)
		if got == nil {
			t.Fatal("expected non-nil result")
		}
		if got.RowId != "42" {
			t.Fatalf("RowId = %q, want 42", got.RowId)
		}
		if got.PlayDuration == nil || *got.PlayDuration != 600 {
			t.Fatalf("PlayDuration = %v, want 600 (seconds preserved)", got.PlayDuration)
		}
		if got.ItemName == nil || *got.ItemName != "Big Buck Bunny" {
			t.Fatalf("ItemName = %v", got.ItemName)
		}
	})

	t.Run("short row rejected", func(t *testing.T) {
		if got := mapPluginRow(jellyfin.PlaybackReportingRow{"1", "2", "3"}); got != nil {
			t.Fatalf("short row should be nil, got %+v", got)
		}
	})

	t.Run("empty rowid rejected", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"", "d", "u", "i", "t", "n", "m", "c", "dev", float64(10),
		}
		if got := mapPluginRow(row); got != nil {
			t.Fatalf("empty rowid should be nil, got %+v", got)
		}
	})

	t.Run("nil fields become nil pointers", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"7", nil, nil, nil, nil, nil, nil, nil, nil, nil,
		}
		got := mapPluginRow(row)
		if got == nil {
			t.Fatal("row with valid rowid should not be nil")
		}
		if got.DateCreated != nil || got.UserId != nil || got.ItemName != nil {
			t.Fatal("nil source fields should map to nil pointers")
		}
		if got.PlayDuration == nil || *got.PlayDuration != 0 {
			t.Fatalf("nil duration should default to 0, got %v", got.PlayDuration)
		}
	})

	t.Run("negative duration clamped to 0", func(t *testing.T) {
		row := jellyfin.PlaybackReportingRow{
			"9", "d", "u", "i", "t", "n", "m", "c", "dev", float64(-5),
		}
		got := mapPluginRow(row)
		if got == nil || got.PlayDuration == nil || *got.PlayDuration != 0 {
			t.Fatalf("negative duration should clamp to 0, got %v", got)
		}
	})
}
