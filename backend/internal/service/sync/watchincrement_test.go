package sync

import (
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
)

func i64(v int64) *int64 { return &v }

// baseTime is an arbitrary fixed instant so tests are deterministic.
var baseTime = time.Date(2026, 7, 12, 10, 0, 0, 0, time.UTC)

// prevAt builds a previous watchdog snapshot taken `agoSeconds` before baseTime,
// with an optional last observed media position (in seconds, converted to ticks).
func prevAt(agoSeconds int, posSeconds *int64) models.JFActivityWatchdog {
	last := baseTime.Add(-time.Duration(agoSeconds) * time.Second)
	wd := models.JFActivityWatchdog{LastTickAt: &last}
	if posSeconds != nil {
		wd.PlaybackDuration = i64(*posSeconds * ticksPerSecond)
	}
	return wd
}

func curPosTicks(posSeconds int64) *int64 {
	return i64(posSeconds * ticksPerSecond)
}

func TestWatchIncrement(t *testing.T) {
	tests := []struct {
		name     string
		prev     models.JFActivityWatchdog
		isPaused bool
		curPos   *int64
		want     int64
	}{
		// --- Position-based (the accurate path) ---
		{
			name:   "normal play: position advances with wall clock",
			prev:   prevAt(10, i64(100)), // 10s ago, was at 100s
			curPos: curPosTicks(110),     // now at 110s → +10s
			want:   10,
		},
		{
			name:     "paused but flag says playing: position frozen → 0 (pause never counted)",
			prev:     prevAt(10, i64(100)),
			isPaused: false,            // flag missed the pause
			curPos:   curPosTicks(100), // position did NOT move
			want:     0,
		},
		{
			name:     "paused with flag set: position frozen → 0",
			prev:     prevAt(10, i64(100)),
			isPaused: true,
			curPos:   curPosTicks(100),
			want:     0,
		},
		{
			name:   "seek forward: position jumps far, clamp to wall clock",
			prev:   prevAt(10, i64(100)),
			curPos: curPosTicks(400), // +300s of position
			want:   10,               // only 10s really elapsed
		},
		{
			name:   "seek backward / rewatch: negative position delta → 0",
			prev:   prevAt(10, i64(100)),
			curPos: curPosTicks(70), // went back 30s
			want:   0,
		},
		{
			name:   "buffering: position barely advances → count the small real progress",
			prev:   prevAt(10, i64(100)),
			curPos: curPosTicks(102), // only +2s of media played
			want:   2,
		},
		{
			name:   "missed tick: large position delta capped at maxTickDelta",
			prev:   prevAt(30, i64(100)), // 30s since last tick
			curPos: curPosTicks(130),     // +30s position
			want:   maxTickDelta,         // 15
		},
		{
			name:   "clock skew: now before last tick → 0",
			prev:   prevAt(-5, i64(100)), // last tick is 5s in the future
			curPos: curPosTicks(100),
			want:   0,
		},

		// --- Fallback path (no position info: live TV, first tick, etc.) ---
		{
			name:   "no position baseline, not paused → wall clock",
			prev:   prevAt(10, nil), // prev has no PlaybackDuration
			curPos: curPosTicks(110),
			want:   10,
		},
		{
			name:     "no current position, not paused → wall clock",
			prev:     prevAt(10, i64(100)),
			isPaused: false,
			curPos:   nil,
			want:     10,
		},
		{
			name:     "no position, paused → 0 (gated by flag)",
			prev:     prevAt(10, i64(100)),
			isPaused: true,
			curPos:   nil,
			want:     0,
		},
		{
			name:   "no position, missed tick, not paused → capped",
			prev:   prevAt(40, nil),
			curPos: nil,
			want:   maxTickDelta,
		},

		// --- Edge: no baseline timestamp at all ---
		{
			name:   "first tick (LastTickAt nil) → 0",
			prev:   models.JFActivityWatchdog{},
			curPos: curPosTicks(100),
			want:   0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := watchIncrement(tt.prev, tt.isPaused, tt.curPos, baseTime)
			if got != tt.want {
				t.Fatalf("watchIncrement() = %d, want %d", got, tt.want)
			}
		})
	}
}

// TestWatchIncrement_PauseExcludedOverSession simulates a full session with a
// pause and proves total accumulated time equals real played time, not wall time.
func TestWatchIncrement_PauseExcludedOverSession(t *testing.T) {
	// Timeline of 10s ticks. Position (seconds) observed at each tick.
	// t=0   start at pos 0
	// t=10  pos 10   (played 10)
	// t=20  pos 20   (played 10)
	// t=30  pos 20   (PAUSED, position frozen)  → +0
	// t=40  pos 20   (still paused)              → +0
	// t=50  pos 30   (resumed, played 10)        → +10
	positions := []int64{0, 10, 20, 20, 20, 30}
	var total int64
	prevPos := positions[0]
	prevTime := baseTime
	for i := 1; i < len(positions); i++ {
		now := baseTime.Add(time.Duration(i*10) * time.Second)
		prev := models.JFActivityWatchdog{
			LastTickAt:       &prevTime,
			PlaybackDuration: i64(prevPos * ticksPerSecond),
		}
		total += watchIncrement(prev, false, curPosTicks(positions[i]), now)
		prevPos = positions[i]
		prevTime = now
	}
	// Real media played = 30s (from pos 0 to 30), despite 50s of wall-clock time.
	if total != 30 {
		t.Fatalf("accumulated watch time = %ds, want 30s (20s of pause must be excluded)", total)
	}
}
