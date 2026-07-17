package handler

import (
	"context"
	"fmt"
	"log"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/prometheus/client_golang/prometheus"
	"gorm.io/gorm"
)

type MetricsCollector struct {
	repos *repository.Container
	db    *gorm.DB

	// ── Global ────────────────────────────────────────────────────────────────
	totalPlays     *prometheus.Desc
	totalDuration  *prometheus.Desc
	totalUsers     *prometheus.Desc
	totalLibraries *prometheus.Desc

	// ── Per-user ──────────────────────────────────────────────────────────────
	playsByUser *prometheus.Desc
	hoursByUser *prometheus.Desc

	// ── Per-library ───────────────────────────────────────────────────────────
	playsByLibrary *prometheus.Desc

	// ── Media type / client / playback method ─────────────────────────────────
	playsByMediaType      *prometheus.Desc
	playsByClient         *prometheus.Desc
	playsByPlaybackMethod *prometheus.Desc

	// ── Time patterns ─────────────────────────────────────────────────────────
	playsByHour      *prometheus.Desc // label: hour (0-23)
	playsByDayOfWeek *prometheus.Desc // label: day (0=Sun..6=Sat)
	heatmapPlays     *prometheus.Desc // labels: day, hour


	// ── Top content ───────────────────────────────────────────────────────────
	itemPlayCount *prometheus.Desc // labels: name, type

	// ── Completion ────────────────────────────────────────────────────────────
	completedPlays   *prometheus.Desc
	abandonedPlays   *prometheus.Desc
	completionRate   *prometheus.Desc

	// ── Library inventory ─────────────────────────────────────────────────────
	itemsByType      *prometheus.Desc // label: type
	unwatchedByType  *prometheus.Desc // label: type

	// ── Binge watching ────────────────────────────────────────────────────────
	bingeSessions *prometheus.Desc

	// ── Live ──────────────────────────────────────────────────────────────────
	activeSessions *prometheus.Desc
}

func NewMetricsCollector(repos *repository.Container, db *gorm.DB) *MetricsCollector {
	ns := "jellyfin"
	return &MetricsCollector{
		repos: repos,
		db:    db,

		totalPlays:     prometheus.NewDesc(ns+"_total_plays", "Total number of plays all time", nil, nil),
		totalDuration:  prometheus.NewDesc(ns+"_total_duration_seconds", "Total watch time in seconds all time", nil, nil),
		totalUsers:     prometheus.NewDesc(ns+"_total_users", "Total number of users", nil, nil),
		totalLibraries: prometheus.NewDesc(ns+"_total_libraries", "Total number of libraries", nil, nil),

		playsByUser: prometheus.NewDesc(ns+"_plays_by_user_total", "Total plays per user", []string{"user"}, nil),
		hoursByUser: prometheus.NewDesc(ns+"_hours_by_user_total", "Total watch hours per user", []string{"user"}, nil),

		playsByLibrary: prometheus.NewDesc(ns+"_plays_by_library_total", "Total plays per library", []string{"library"}, nil),

		playsByMediaType:      prometheus.NewDesc(ns+"_plays_by_media_type_total", "Total plays per media type", []string{"type"}, nil),
		playsByClient:         prometheus.NewDesc(ns+"_plays_by_client_total", "Total plays per client", []string{"client"}, nil),
		playsByPlaybackMethod: prometheus.NewDesc(ns+"_plays_by_playback_method_total", "Total plays per playback method", []string{"method"}, nil),

		playsByHour:      prometheus.NewDesc(ns+"_plays_by_hour", "Play count per hour of day (all time) — use as pattern gauge", []string{"hour"}, nil),
		playsByDayOfWeek: prometheus.NewDesc(ns+"_plays_by_day_of_week", "Play count per day of week (all time) — use as pattern gauge", []string{"day"}, nil),
		heatmapPlays:     prometheus.NewDesc(ns+"_heatmap_plays", "Play count per hour+day combination — use for Grafana heatmap", []string{"day", "hour"}, nil),

		itemPlayCount: prometheus.NewDesc(ns+"_item_play_count", "Play count per media item (top 50)", []string{"name", "type"}, nil),

		completedPlays: prometheus.NewDesc(ns+"_completed_plays_total", "Number of plays considered complete (>= 90%)", nil, nil),
		abandonedPlays: prometheus.NewDesc(ns+"_abandoned_plays_total", "Number of plays abandoned (< 90%)", nil, nil),
		completionRate: prometheus.NewDesc(ns+"_completion_rate_percent", "Percentage of plays that are complete", nil, nil),

		itemsByType:     prometheus.NewDesc(ns+"_items_by_type_total", "Total library items per type", []string{"type"}, nil),
		unwatchedByType: prometheus.NewDesc(ns+"_unwatched_items_by_type", "Unwatched items per type", []string{"type"}, nil),

		bingeSessions: prometheus.NewDesc(ns+"_binge_sessions_total", "Total binge-watching sessions (>= 3 episodes in one sitting)", nil, nil),

		activeSessions: prometheus.NewDesc(ns+"_active_sessions", "Number of currently active playback sessions", nil, nil),
	}
}

func (c *MetricsCollector) Describe(ch chan<- *prometheus.Desc) {
	ch <- c.totalPlays
	ch <- c.totalDuration
	ch <- c.totalUsers
	ch <- c.totalLibraries
	ch <- c.playsByUser
	ch <- c.hoursByUser
	ch <- c.playsByLibrary
	ch <- c.playsByMediaType
	ch <- c.playsByClient
	ch <- c.playsByPlaybackMethod
	ch <- c.playsByHour
	ch <- c.playsByDayOfWeek
	ch <- c.heatmapPlays
	ch <- c.itemPlayCount
	ch <- c.completedPlays
	ch <- c.abandonedPlays
	ch <- c.completionRate
	ch <- c.itemsByType
	ch <- c.unwatchedByType
	ch <- c.bingeSessions
	ch <- c.activeSessions
}

func (c *MetricsCollector) Collect(ch chan<- prometheus.Metric) {
	ctx := context.Background()

	// ── Global stats ──────────────────────────────────────────────────────────
	if gs, err := c.repos.Stats.GetGlobalStats(ctx); err == nil {
		ch <- prometheus.MustNewConstMetric(c.totalPlays, prometheus.CounterValue, float64(gs.TotalPlays))
		ch <- prometheus.MustNewConstMetric(c.totalDuration, prometheus.CounterValue, float64(gs.TotalDuration))
		ch <- prometheus.MustNewConstMetric(c.totalUsers, prometheus.GaugeValue, float64(gs.TotalUsers))
		ch <- prometheus.MustNewConstMetric(c.totalLibraries, prometheus.GaugeValue, float64(gs.TotalLibraries))
	} else {
		log.Printf("metrics: GetGlobalStats: %v", err)
	}

	// ── Per-user stats ────────────────────────────────────────────────────────
	if users, err := c.repos.Stats.GetTopUsers(ctx, 500); err == nil {
		for _, u := range users {
			ch <- prometheus.MustNewConstMetric(c.playsByUser, prometheus.CounterValue, float64(u.TotalPlays), u.UserName)
			ch <- prometheus.MustNewConstMetric(c.hoursByUser, prometheus.CounterValue, u.TotalHours, u.UserName)
		}
	} else {
		log.Printf("metrics: GetTopUsers: %v", err)
	}

	// ── Per-library stats ─────────────────────────────────────────────────────
	if libs, err := c.repos.Stats.GetMostViewedLibraries(ctx, 100); err == nil {
		for _, l := range libs {
			ch <- prometheus.MustNewConstMetric(c.playsByLibrary, prometheus.CounterValue, float64(l.TotalPlays), l.Name)
		}
	} else {
		log.Printf("metrics: GetMostViewedLibraries: %v", err)
	}

	// ── Media type (raw SQL) ──────────────────────────────────────────────────
	type typeRow struct {
		Type  *string
		Count int
	}
	var typeRows []typeRow
	c.db.Raw(`
		SELECT COALESCE(i."Type", 'Other') AS type, COUNT(*)::int AS count
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		GROUP BY i."Type"
	`).Scan(&typeRows)
	totals := map[string]int{"Audio": 0, "Movie": 0, "Series": 0, "Other": 0}
	for _, r := range typeRows {
		t := "Other"
		if r.Type != nil {
			t = *r.Type
		}
		switch t {
		case "Audio":
			totals["Audio"] += r.Count
		case "Movie", "Video":
			totals["Movie"] += r.Count
		case "Series", "Episode":
			totals["Series"] += r.Count
		default:
			totals["Other"] += r.Count
		}
	}
	for t, count := range totals {
		ch <- prometheus.MustNewConstMetric(c.playsByMediaType, prometheus.CounterValue, float64(count), t)
	}

	// ── Per-client (raw SQL) ──────────────────────────────────────────────────
	type clientRow struct {
		Client string
		Count  int
	}
	var clientRows []clientRow
	c.db.Raw(`
		SELECT "Client" AS client, COUNT(*)::int AS count
		FROM jf_playback_activity
		WHERE "Client" IS NOT NULL AND "Client" <> ''
		GROUP BY "Client" ORDER BY count DESC LIMIT 50
	`).Scan(&clientRows)
	for _, r := range clientRows {
		ch <- prometheus.MustNewConstMetric(c.playsByClient, prometheus.CounterValue, float64(r.Count), r.Client)
	}

	// ── Playback method (raw SQL) ─────────────────────────────────────────────
	type methodRow struct {
		Method string
		Count  int
	}
	var methodRows []methodRow
	c.db.Raw(`
		SELECT COALESCE("PlayMethod", 'Unknown') AS method, COUNT(*)::int AS count
		FROM jf_playback_activity
		GROUP BY "PlayMethod"
	`).Scan(&methodRows)
	for _, r := range methodRows {
		ch <- prometheus.MustNewConstMetric(c.playsByPlaybackMethod, prometheus.CounterValue, float64(r.Count), r.Method)
	}

	// ── Plays by hour of day (raw SQL) ────────────────────────────────────────
	type hourRow struct {
		Hour  int
		Plays int
	}
	var hourRows []hourRow
	c.db.Raw(`
		SELECT EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour, COUNT(*)::int AS plays
		FROM jf_playback_activity
		GROUP BY hour ORDER BY hour
	`).Scan(&hourRows)
	for _, r := range hourRows {
		ch <- prometheus.MustNewConstMetric(c.playsByHour, prometheus.GaugeValue, float64(r.Plays), fmt.Sprint(r.Hour))
	}

	// ── Plays by day of week (raw SQL) ────────────────────────────────────────
	type dayRow struct {
		Day   int
		Plays int
	}
	var dayRows []dayRow
	c.db.Raw(`
		SELECT EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day, COUNT(*)::int AS plays
		FROM jf_playback_activity
		GROUP BY day ORDER BY day
	`).Scan(&dayRows)
	for _, r := range dayRows {
		ch <- prometheus.MustNewConstMetric(c.playsByDayOfWeek, prometheus.GaugeValue, float64(r.Plays), fmt.Sprint(r.Day))
	}

	// ── Heatmap: plays by day+hour (raw SQL) ──────────────────────────────────
	type heatRow struct {
		Day   int
		Hour  int
		Plays int
	}
	var heatRows []heatRow
	c.db.Raw(`
		SELECT
		  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
		  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
		  COUNT(*)::int AS plays
		FROM jf_playback_activity
		GROUP BY day, hour
	`).Scan(&heatRows)
	for _, r := range heatRows {
		ch <- prometheus.MustNewConstMetric(c.heatmapPlays, prometheus.GaugeValue, float64(r.Plays), fmt.Sprint(r.Day), fmt.Sprint(r.Hour))
	}

	// ── Top 50 items ──────────────────────────────────────────────────────────
	type itemRow struct {
		Name  string
		Type  string
		Plays int
	}
	var itemRows []itemRow
	c.db.Raw(`
		SELECT COALESCE(i."Name", a."NowPlayingItemName") AS name,
		       COALESCE(i."Type", 'Unknown') AS type,
		       COUNT(*)::int AS plays
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		GROUP BY name, type ORDER BY plays DESC LIMIT 50
	`).Scan(&itemRows)
	for _, r := range itemRows {
		ch <- prometheus.MustNewConstMetric(c.itemPlayCount, prometheus.GaugeValue, float64(r.Plays), r.Name, r.Type)
	}

	// ── Completion rate (raw SQL) ─────────────────────────────────────────────
	type completionRow struct {
		Completed int
		Abandoned int
	}
	var comp completionRow
	c.db.Raw(`
		SELECT
		  COUNT(*) FILTER (WHERE COALESCE("PlaybackDuration", 0) > 0 AND
		    CASE WHEN i."RunTimeTicks" > 0
		         THEN ("PlaybackDuration" * 10000000.0 / i."RunTimeTicks") >= 0.9
		         ELSE false END
		  )::int AS completed,
		  COUNT(*) FILTER (WHERE COALESCE("PlaybackDuration", 0) > 0 AND
		    CASE WHEN i."RunTimeTicks" > 0
		         THEN ("PlaybackDuration" * 10000000.0 / i."RunTimeTicks") < 0.9
		         ELSE true END
		  )::int AS abandoned
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		WHERE COALESCE(a."PlaybackDuration", 0) > 0
	`).Scan(&comp)
	total := comp.Completed + comp.Abandoned
	rate := 0.0
	if total > 0 {
		rate = float64(comp.Completed) / float64(total) * 100
	}
	ch <- prometheus.MustNewConstMetric(c.completedPlays, prometheus.CounterValue, float64(comp.Completed))
	ch <- prometheus.MustNewConstMetric(c.abandonedPlays, prometheus.CounterValue, float64(comp.Abandoned))
	ch <- prometheus.MustNewConstMetric(c.completionRate, prometheus.GaugeValue, rate)

	// ── Inventory by type (raw SQL) ───────────────────────────────────────────
	type invRow struct {
		Type  string
		Count int
	}
	var invRows []invRow
	c.db.Raw(`
		SELECT COALESCE("Type", 'Unknown') AS type, COUNT(*)::int AS count
		FROM jf_library_items
		GROUP BY "Type"
	`).Scan(&invRows)
	for _, r := range invRows {
		ch <- prometheus.MustNewConstMetric(c.itemsByType, prometheus.GaugeValue, float64(r.Count), r.Type)
	}

	// ── Unwatched by type (raw SQL) ───────────────────────────────────────────
	var uwRows []invRow
	c.db.Raw(`
		SELECT COALESCE(i."Type", 'Unknown') AS type, COUNT(*)::int AS count
		FROM jf_library_items i
		WHERE NOT EXISTS (
		  SELECT 1 FROM jf_playback_activity a WHERE a."NowPlayingItemId" = i."Id"
		)
		GROUP BY i."Type"
	`).Scan(&uwRows)
	for _, r := range uwRows {
		ch <- prometheus.MustNewConstMetric(c.unwatchedByType, prometheus.GaugeValue, float64(r.Count), r.Type)
	}

	// ── Binge sessions (raw SQL) ──────────────────────────────────────────────
	type bingeRow struct{ Count int }
	var binge bingeRow
	c.db.Raw(`
		SELECT COUNT(*)::int AS count FROM (
		  SELECT "UserId", DATE("ActivityDateInserted"::timestamptz) AS day
		  FROM jf_playback_activity a
		  JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId" AND i."Type" = 'Episode'
		  GROUP BY "UserId", day
		  HAVING COUNT(*) >= 3
		) AS binge_days
	`).Scan(&binge)
	ch <- prometheus.MustNewConstMetric(c.bingeSessions, prometheus.CounterValue, float64(binge.Count))

	// ── Active sessions ───────────────────────────────────────────────────────
	if sessions, err := c.repos.Watchdog.List(ctx); err == nil {
		ch <- prometheus.MustNewConstMetric(c.activeSessions, prometheus.GaugeValue, float64(len(sessions)))
	} else {
		log.Printf("metrics: Watchdog.List: %v", err)
	}

}
