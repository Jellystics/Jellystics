package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// StatsFrontendHandler serves all stats endpoints expected by the frontend.
type StatsFrontendHandler struct {
	db    *gorm.DB
	repos *repository.Container
}

// NewStatsFrontendHandler constructs a StatsFrontendHandler.
func NewStatsFrontendHandler(db *gorm.DB, repos *repository.Container) *StatsFrontendHandler {
	return &StatsFrontendHandler{db: db, repos: repos}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func parseDays(value string, fallback int) int {
	n, err := strconv.Atoi(value)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}

// parseDaysAllTime returns (days, allTime).
// days=0 → allTime=true; days=N → allTime=false, days=N; invalid → fallback, false.
func parseDaysAllTime(c *gin.Context, fallback int) (int, bool) {
	val := c.DefaultQuery("days", strconv.Itoa(fallback))
	if val == "0" {
		return 0, true
	}
	n, err := strconv.Atoi(val)
	if err != nil || n <= 0 {
		return fallback, false
	}
	return n, false
}

type pageResult struct {
	CurrentPage int         `json:"current_page"`
	Pages       int         `json:"pages"`
	Size        int         `json:"size"`
	Results     interface{} `json:"results"`
}

func paginate(total int, pageSize int, page int, results interface{}) pageResult {
	pages := 0
	if total > 0 && pageSize > 0 {
		pages = (total + pageSize - 1) / pageSize
	}
	return pageResult{CurrentPage: page, Pages: pages, Size: pageSize, Results: results}
}

// ---------------------------------------------------------------------------
// Live session merging
// ---------------------------------------------------------------------------

type liveSession struct {
	UserId              string
	UserName            string
	Client              string
	PlayMethod          string
	NowPlayingItemId    string
	EpisodeId           string
	NowPlayingItemName  string
	Type                string
	Genres              []string
	LibraryId           string
	PlaybackDuration    int // minutes
	date                string
	hour                int
	day                 int
}

func (h *StatsFrontendHandler) getLiveActivity(ctx context.Context) []liveSession {
	cfg, err := h.repos.Config.Get(ctx)
	if err != nil || cfg.JFHost == nil || *cfg.JFHost == "" {
		return nil
	}
	apiKey := ""
	if cfg.JFApiKey != nil {
		apiKey = *cfg.JFApiKey
	}
	jf := jellyfin.NewClient(*cfg.JFHost, apiKey)
	sessions, err := jf.GetSessions(ctx)
	if err != nil {
		return nil
	}

	var active []jellyfin.SessionInfo
	for _, s := range sessions {
		if s.NowPlayingItem != nil {
			active = append(active, s)
		}
	}
	if len(active) == 0 {
		return nil
	}

	itemIds := make(map[string]struct{})
	for _, s := range active {
		itemIds[s.NowPlayingItem.Id] = struct{}{}
		if s.NowPlayingItem.SeriesId != nil && *s.NowPlayingItem.SeriesId != "" {
			itemIds[*s.NowPlayingItem.SeriesId] = struct{}{}
		}
		if s.NowPlayingItem.AlbumId != nil && *s.NowPlayingItem.AlbumId != "" {
			itemIds[*s.NowPlayingItem.AlbumId] = struct{}{}
		}
	}
	ids := make([]string, 0, len(itemIds))
	for id := range itemIds {
		ids = append(ids, id)
	}

	type LibRow struct {
		Id       string
		ParentId string
		Type     string
		Genres   json.RawMessage
	}
	var libRows []LibRow
	if len(ids) > 0 {
		h.db.Raw(`
			SELECT "Id", "ParentId", "Type", "Genres" FROM jf_library_items WHERE "Id" IN ?
			UNION ALL
			SELECT "Id", "LibraryId" AS "ParentId", 'Audio' AS "Type", "Genres" FROM jf_music_tracks WHERE "Id" IN ?
		`, ids, ids).Scan(&libRows)
	}

	now := time.Now()
	var result []liveSession
	for _, s := range active {
		item := s.NowPlayingItem
		lookupId := item.Id
		if item.SeriesId != nil && *item.SeriesId != "" {
			lookupId = *item.SeriesId
		} else if item.AlbumId != nil && *item.AlbumId != "" {
			lookupId = *item.AlbumId
		}

		var libItem *LibRow
		for i := range libRows {
			if libRows[i].Id == lookupId || libRows[i].Id == item.Id {
				libItem = &libRows[i]
				break
			}
		}

		var genres []string
		if libItem != nil && libItem.Genres != nil {
			_ = json.Unmarshal(libItem.Genres, &genres)
		}

		itemType := "Unknown"
		if item.SeriesId != nil && *item.SeriesId != "" {
			itemType = "Series"
		} else if item.AlbumId != nil && *item.AlbumId != "" {
			itemType = "Audio"
		} else if libItem != nil {
			itemType = libItem.Type
		}

		libraryId := ""
		if libItem != nil {
			libraryId = libItem.ParentId
		}

		positionTicks := int64(0)
		if s.PlayState != nil && s.PlayState.PositionTicks != nil {
			positionTicks = *s.PlayState.PositionTicks
		}
		playbackMinutes := int(positionTicks / 600_000_000)

		playMethod := "Unknown"
		if s.PlayState != nil && s.PlayState.PlayMethod != nil {
			playMethod = *s.PlayState.PlayMethod
		}

		episodeId := ""
		if item.SeriesId != nil && *item.SeriesId != "" {
			episodeId = item.Id
		} else if item.AlbumId != nil && *item.AlbumId != "" {
			episodeId = item.Id
		}

		itemName := item.Name
		if item.SeriesName != nil && *item.SeriesName != "" {
			itemName = *item.SeriesName
		} else if item.Album != nil && *item.Album != "" {
			itemName = *item.Album
		}

		ls := liveSession{
			UserId:             s.UserId,
			UserName:           s.UserName,
			Client:             s.Client,
			PlayMethod:         playMethod,
			NowPlayingItemId:   lookupId,
			EpisodeId:          episodeId,
			NowPlayingItemName: itemName,
			Type:               itemType,
			Genres:             genres,
			LibraryId:          libraryId,
			PlaybackDuration:   playbackMinutes,
			date:               now.Format("2006-01-02"),
			hour:               now.Hour(),
			day:                int(now.Weekday()),
		}
		result = append(result, ls)
	}
	return result
}

// ---------------------------------------------------------------------------
// GET /stats/getGlobalStats
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGlobalStats(c *gin.Context) {
	var stats struct {
		TotalPlays     int `json:"TotalPlays"`
		TotalWatchTime int `json:"TotalWatchTime"`
		ActiveUsers    int `json:"ActiveUsers"`
		TotalUsers     int `json:"TotalUsers"`
		TotalLibraries int `json:"TotalLibraries"`
		TotalItems     int `json:"TotalItems"`
	}
	h.db.Raw(`
		SELECT
		  (SELECT COUNT(*)::int FROM jf_playback_activity) AS "TotalPlays",
		  (SELECT FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int FROM jf_playback_activity) AS "TotalWatchTime",
		  (SELECT COUNT(DISTINCT "UserId")::int FROM jf_playback_activity
		    WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - INTERVAL '30 days') AS "ActiveUsers",
		  (SELECT COUNT(*)::int FROM jf_users) AS "TotalUsers",
		  (SELECT COUNT(*)::int FROM jf_libraries WHERE archived = false) AS "TotalLibraries",
		  ((SELECT COUNT(*)::int FROM jf_library_items WHERE archived = false AND "Type" NOT IN ('Season', 'Folder'))
		   + (SELECT COUNT(*)::int FROM jf_music_tracks WHERE archived = false)) AS "TotalItems"
	`).Scan(&stats)

	live := h.getLiveActivity(c.Request.Context())
	stats.TotalPlays += len(live)
	for _, ls := range live {
		stats.TotalWatchTime += ls.PlaybackDuration
	}

	c.JSON(http.StatusOK, stats)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostPlayedItems?type=all&limit=5&days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostPlayedItems(c *gin.Context) {
	itemType := c.DefaultQuery("type", "all")
	limit := parseDays(c.DefaultQuery("limit", "5"), 5)
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days

	type Item struct {
		Id        string `json:"Id"`
		Name      string `json:"Name"`
		PlayCount int    `json:"PlayCount"`
		Type      string `json:"Type"`
	}
	var dbItems []Item

	baseSQLAllTime := `
		SELECT
		  a."NowPlayingItemId" AS "Id",
		  COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemName") AS "Name",
		  COUNT(*)::int AS "PlayCount",
		  CASE
		    WHEN mt."Id" IS NOT NULL THEN 'Audio'
		    WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Series'
		    ELSE COALESCE(i."Type", 'Unknown')
		  END AS "Type"
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_music_tracks mt ON mt."Id" = a."EpisodeId"
		WHERE 1=1
	`
	baseSQLDays := `
		SELECT
		  a."NowPlayingItemId" AS "Id",
		  COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemName") AS "Name",
		  COUNT(*)::int AS "PlayCount",
		  CASE
		    WHEN mt."Id" IS NOT NULL THEN 'Audio'
		    WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Series'
		    ELSE COALESCE(i."Type", 'Unknown')
		  END AS "Type"
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_music_tracks mt ON mt."Id" = a."EpisodeId"
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
	`
	suffix := ` GROUP BY a."NowPlayingItemId", 2, 4 ORDER BY "PlayCount" DESC LIMIT ?`

	if allTime {
		baseSQL := baseSQLAllTime
		switch itemType {
		case "Series":
			h.db.Raw(baseSQL+` AND a."SeriesName" IS NOT NULL AND a."SeriesName" <> ''`+suffix, limit).Scan(&dbItems)
		case "Audio":
			h.db.Raw(baseSQL+` AND mt."Id" IS NOT NULL`+suffix, limit).Scan(&dbItems)
		case "Movie":
			h.db.Raw(baseSQL+` AND COALESCE(i."Type", '') IN ('Movie', 'Video')`+suffix, limit).Scan(&dbItems)
		case "all":
			h.db.Raw(baseSQL+suffix, limit).Scan(&dbItems)
		default:
			h.db.Raw(baseSQL+` AND i."Type" = ?`+suffix, itemType, limit).Scan(&dbItems)
		}
	} else {
		baseSQL := baseSQLDays
		switch itemType {
		case "Series":
			h.db.Raw(baseSQL+` AND a."SeriesName" IS NOT NULL AND a."SeriesName" <> ''`+suffix, daysArg, limit).Scan(&dbItems)
		case "Audio":
			h.db.Raw(baseSQL+` AND mt."Id" IS NOT NULL`+suffix, daysArg, limit).Scan(&dbItems)
		case "Movie":
			h.db.Raw(baseSQL+` AND COALESCE(i."Type", '') IN ('Movie', 'Video')`+suffix, daysArg, limit).Scan(&dbItems)
		case "all":
			h.db.Raw(baseSQL+suffix, daysArg, limit).Scan(&dbItems)
		default:
			h.db.Raw(baseSQL+` AND i."Type" = ?`+suffix, daysArg, itemType, limit).Scan(&dbItems)
		}
	}

	if dbItems == nil {
		dbItems = []Item{}
	}

	live := h.getLiveActivity(c.Request.Context())
	liveMap := map[string]*Item{}
	for _, ls := range live {
		key := ls.NowPlayingItemId
		if _, ok := liveMap[key]; !ok {
			liveMap[key] = &Item{Id: ls.NowPlayingItemId, Name: ls.NowPlayingItemName, PlayCount: 0, Type: ls.Type}
		}
		liveMap[key].PlayCount++
	}

	merged := make([]Item, len(dbItems))
	copy(merged, dbItems)
	for _, liveItem := range liveMap {
		found := false
		for i := range merged {
			if merged[i].Id == liveItem.Id {
				merged[i].PlayCount += liveItem.PlayCount
				found = true
				break
			}
		}
		if !found {
			merged = append(merged, *liveItem)
		}
	}
	sort.Slice(merged, func(i, j int) bool { return merged[i].PlayCount > merged[j].PlayCount })
	if len(merged) > limit {
		merged = merged[:limit]
	}

	c.JSON(http.StatusOK, merged)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostActiveUsers?limit=5&days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostActiveUsers(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days
	limit := parseDays(c.DefaultQuery("limit", "5"), 5)

	type UserRow struct {
		UserId         string `json:"UserId"`
		UserName       string `json:"UserName"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	var dbRows []UserRow
	if allTime {
		h.db.Raw(`
			SELECT
			  a."UserId",
			  COALESCE(u."Name", MAX(a."UserName"), a."UserId") AS "UserName",
			  COUNT(*)::int AS "TotalPlays",
			  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime"
			FROM jf_playback_activity a
			LEFT JOIN jf_users u ON u."Id" = a."UserId"
			GROUP BY a."UserId", u."Name" ORDER BY "TotalPlays" DESC LIMIT ?
		`, limit).Scan(&dbRows)
	} else {
		h.db.Raw(`
			SELECT
			  a."UserId",
			  COALESCE(u."Name", MAX(a."UserName"), a."UserId") AS "UserName",
			  COUNT(*)::int AS "TotalPlays",
			  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime"
			FROM jf_playback_activity a
			LEFT JOIN jf_users u ON u."Id" = a."UserId"
			WHERE a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
			GROUP BY a."UserId", u."Name"
			ORDER BY "TotalPlays" DESC
			LIMIT ?
		`, daysArg, limit).Scan(&dbRows)
	}

	if dbRows == nil {
		dbRows = []UserRow{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		found := false
		for i := range dbRows {
			if dbRows[i].UserId == ls.UserId {
				dbRows[i].TotalPlays++
				dbRows[i].TotalWatchTime += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			dbRows = append(dbRows, UserRow{
				UserId:         ls.UserId,
				UserName:       ls.UserName,
				TotalPlays:     1,
				TotalWatchTime: ls.PlaybackDuration,
			})
		}
	}

	sort.Slice(dbRows, func(i, j int) bool { return dbRows[i].TotalPlays > dbRows[j].TotalPlays })
	if len(dbRows) > limit {
		dbRows = dbRows[:limit]
	}

	c.JSON(http.StatusOK, dbRows)
}

// ---------------------------------------------------------------------------
// GET /stats/getWatchStatisticsOverTime?days=30&userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetWatchStatisticsOverTime(c *gin.Context) {
	daysParam := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysParam)
	if err != nil {
		days = 30
	}
	allTime := days <= 0
	userId := c.Query("userId")

	type Row struct {
		Date     string `json:"date"`
		Plays    int    `json:"plays"`
		Duration int    `json:"duration"`
	}
	var rows []Row

	if userId != "" {
		if allTime {
			h.db.Raw(`
				SELECT
				  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "UserId" = ?
				GROUP BY ("ActivityDateInserted"::timestamptz)::date
				ORDER BY date
			`, userId).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
				  AND "UserId" = ?
				GROUP BY ("ActivityDateInserted"::timestamptz)::date
				ORDER BY date
			`, days, userId).Scan(&rows)
		}
	} else {
		if allTime {
			h.db.Raw(`
				SELECT
				  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				GROUP BY ("ActivityDateInserted"::timestamptz)::date
				ORDER BY date
			`).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
				GROUP BY ("ActivityDateInserted"::timestamptz)::date
				ORDER BY date
			`, days).Scan(&rows)
		}
	}

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		if userId != "" && ls.UserId != userId {
			continue
		}
		found := false
		for i := range rows {
			if rows[i].Date == ls.date {
				rows[i].Plays++
				rows[i].Duration += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, Row{Date: ls.date, Plays: 1, Duration: ls.PlaybackDuration})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPopularHourOfDay?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPopularHourOfDay(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days
	userId := c.Query("userId")

	type Row struct {
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	var rows []Row
	if userId != "" {
		if allTime {
			h.db.Raw(`
				SELECT
				  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "UserId" = ?
				GROUP BY hour ORDER BY hour
			`, userId).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "UserId" = ?
				  AND "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
				GROUP BY hour ORDER BY hour
			`, userId, daysArg).Scan(&rows)
		}
	} else if allTime {
		h.db.Raw(`
			SELECT
			  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
			  COUNT(*)::int AS plays,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			GROUP BY hour ORDER BY hour
		`).Scan(&rows)
	} else {
		h.db.Raw(`
			SELECT
			  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
			  COUNT(*)::int AS plays,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
			GROUP BY hour ORDER BY hour
		`, daysArg).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}

	if userId == "" {
		live := h.getLiveActivity(c.Request.Context())
		for _, ls := range live {
			found := false
			for i := range rows {
				if rows[i].Hour == ls.hour {
					rows[i].Plays++
					rows[i].Duration += ls.PlaybackDuration
					found = true
					break
				}
			}
			if !found {
				rows = append(rows, Row{Hour: ls.hour, Plays: 1, Duration: ls.PlaybackDuration})
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Hour < rows[j].Hour })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPopularDayOfWeek?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPopularDayOfWeek(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days
	userId := c.Query("userId")

	type Row struct {
		Day      int `json:"day"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	var rows []Row
	if userId != "" {
		if allTime {
			h.db.Raw(`
				SELECT
				  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "UserId" = ?
				GROUP BY day ORDER BY day
			`, userId).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
				  COUNT(*)::int AS plays,
				  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
				FROM jf_playback_activity
				WHERE "UserId" = ?
				  AND "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
				GROUP BY day ORDER BY day
			`, userId, daysArg).Scan(&rows)
		}
	} else if allTime {
		h.db.Raw(`
			SELECT
			  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
			  COUNT(*)::int AS plays,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			GROUP BY day ORDER BY day
		`).Scan(&rows)
	} else {
		h.db.Raw(`
			SELECT
			  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
			  COUNT(*)::int AS plays,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
			GROUP BY day ORDER BY day
		`, daysArg).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}

	if userId == "" {
		live := h.getLiveActivity(c.Request.Context())
		for _, ls := range live {
			found := false
			for i := range rows {
				if rows[i].Day == ls.day {
					rows[i].Plays++
					rows[i].Duration += ls.PlaybackDuration
					found = true
					break
				}
			}
			if !found {
				rows = append(rows, Row{Day: ls.day, Plays: 1, Duration: ls.PlaybackDuration})
			}
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Day < rows[j].Day })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostUsedPlaybackMethod?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostUsedPlaybackMethod(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days

	type Row struct {
		Method   string `json:"method"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	var rows []Row
	if allTime {
		h.db.Raw(`
			SELECT
			  COALESCE(NULLIF("PlayMethod", ''), 'Unknown') AS method,
			  COUNT(*)::int AS count,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			GROUP BY method ORDER BY count DESC
		`).Scan(&rows)
	} else {
		h.db.Raw(`
			SELECT
			  COALESCE(NULLIF("PlayMethod", ''), 'Unknown') AS method,
			  COUNT(*)::int AS count,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
			GROUP BY method ORDER BY count DESC
		`, daysArg).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		method := ls.PlayMethod
		if method == "" {
			method = "Unknown"
		}
		found := false
		for i := range rows {
			if rows[i].Method == method {
				rows[i].Count++
				rows[i].Duration += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, Row{Method: method, Count: 1, Duration: ls.PlaybackDuration})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Count > rows[j].Count })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostUsedClients?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostUsedClients(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days

	type Row struct {
		Client   string `json:"client"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	var rows []Row
	if allTime {
		h.db.Raw(`
			SELECT
			  "Client" AS client,
			  COUNT(*)::int AS count,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			WHERE "Client" IS NOT NULL AND "Client" <> ''
			GROUP BY "Client" ORDER BY count DESC LIMIT 10
		`).Scan(&rows)
	} else {
		h.db.Raw(`
			SELECT
			  "Client" AS client,
			  COUNT(*)::int AS count,
			  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
			FROM jf_playback_activity
			WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
			  AND "Client" IS NOT NULL AND "Client" <> ''
			GROUP BY "Client" ORDER BY count DESC LIMIT 10
		`, daysArg).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		if ls.Client == "" {
			continue
		}
		found := false
		for i := range rows {
			if rows[i].Client == ls.Client {
				rows[i].Count++
				rows[i].Duration += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, Row{Client: ls.Client, Count: 1, Duration: ls.PlaybackDuration})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Count > rows[j].Count })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserStats?userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserStats(c *gin.Context) {
	userId := c.Query("userId")

	type UserStat struct {
		UserId          string  `json:"UserId"`
		UserName        string  `json:"UserName"`
		TotalPlays      int     `json:"TotalPlays"`
		TotalWatchTime  int     `json:"TotalWatchTime"`
		UniqueItems     int     `json:"UniqueItems"`
		LastSeen        *string `json:"LastSeen"`
		FirstSeen       *string `json:"FirstSeen"`
		MostUsedClient  *string `json:"MostUsedClient"`
		MostUsedDevice  *string `json:"MostUsedDevice"`
	}

	const userStatSQL = `
		SELECT
		  u."Id" AS "UserId",
		  u."Name" AS "UserName",
		  COUNT(a."Id")::int AS "TotalPlays",
		  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
		  COUNT(DISTINCT a."NowPlayingItemId")::int AS "UniqueItems",
		  COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
		  MIN(a."ActivityDateInserted") AS "FirstSeen",
		  (
		    SELECT a2."Client" FROM jf_playback_activity a2
		    WHERE a2."UserId" = u."Id" AND a2."Client" IS NOT NULL AND a2."Client" <> ''
		    GROUP BY a2."Client" ORDER BY COUNT(*) DESC LIMIT 1
		  ) AS "MostUsedClient",
		  (
		    SELECT a2."DeviceName" FROM jf_playback_activity a2
		    WHERE a2."UserId" = u."Id" AND a2."DeviceName" IS NOT NULL AND a2."DeviceName" <> ''
		    GROUP BY a2."DeviceName" ORDER BY COUNT(*) DESC LIMIT 1
		  ) AS "MostUsedDevice"
		FROM jf_users u
		LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
	`

	live := h.getLiveActivity(c.Request.Context())

	if userId != "" {
		var row UserStat
		h.db.Raw(userStatSQL+`WHERE u."Id" = ? GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"`, userId).Scan(&row)

		for _, ls := range live {
			if ls.UserId == userId {
				row.TotalPlays++
				row.TotalWatchTime += ls.PlaybackDuration
			}
		}
		c.JSON(http.StatusOK, row)
		return
	}

	var rows []UserStat
	h.db.Raw(userStatSQL + `GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate" ORDER BY "TotalPlays" DESC`).Scan(&rows)

	if rows == nil {
		rows = []UserStat{}
	}

	for _, ls := range live {
		found := false
		for i := range rows {
			if rows[i].UserId == ls.UserId {
				rows[i].TotalPlays++
				rows[i].TotalWatchTime += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, UserStat{
				UserId:     ls.UserId,
				UserName:   ls.UserName,
				TotalPlays: 1,
				TotalWatchTime: ls.PlaybackDuration,
			})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].TotalPlays > rows[j].TotalPlays })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getAllUserActivity
// ---------------------------------------------------------------------------

type activityRow struct {
	Id                   *string `json:"Id"`
	UserId               *string `json:"UserId"`
	UserName             *string `json:"UserName"`
	ItemId               *string `json:"ItemId"`
	NowPlayingItemName   *string `json:"NowPlayingItemName"`
	NowPlayingItemType   *string `json:"NowPlayingItemType"`
	SeriesName           *string `json:"SeriesName"`
	SeasonId             *string `json:"SeasonId"`
	EpisodeId            *string `json:"EpisodeId"`
	ParentId             *string `json:"ParentId"`
	Client               *string `json:"Client"`
	DeviceName           *string `json:"DeviceName"`
	DeviceId             *string `json:"DeviceId"`
	ApplicationVersion   *string `json:"ApplicationVersion"`
	PlayMethod           *string `json:"PlayMethod"`
	IsPaused             *bool   `json:"IsPaused"`
	IsActive             bool    `json:"IsActive"`
	PlayDuration         *int64  `json:"PlayDuration"`
	ActivityDateInserted *string `json:"ActivityDateInserted"`
	RemoteEndPoint       *string `json:"RemoteEndPoint"`
}

func splitCSV(s string) []string {
	if s == "" {
		return nil
	}
	parts := strings.Split(s, ",")
	out := parts[:0]
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

func inClause(col string, n int) string {
	ph := make([]string, n)
	for i := range ph {
		ph[i] = "?"
	}
	return col + " IN (" + strings.Join(ph, ",") + ")"
}

func (h *StatsFrontendHandler) GetAllUserActivity(c *gin.Context) {
	dateFrom   := c.Query("dateFrom")
	dateTo     := c.Query("dateTo")
	clients    := splitCSV(c.Query("client"))
	methods    := splitCSV(c.Query("playMethod"))
	mediaTypes := splitCSV(c.Query("mediaType"))
	devices    := splitCSV(c.Query("deviceName"))
	users      := splitCSV(c.Query("userName"))
	durMinStr  := c.Query("durMin")
	durMaxStr  := c.Query("durMax")

	const baseSelect = `
		SELECT
		  a."Id", a."UserId", a."UserName",
		  a."NowPlayingItemId" AS "ItemId",
		  a."NowPlayingItemName",
		  CASE
		    WHEN mt."Id" IS NOT NULL THEN 'Audio'
		    WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Episode'
		    ELSE COALESCE(i."Type", 'Unknown')
		  END AS "NowPlayingItemType",
		  a."SeriesName", a."SeasonId", a."EpisodeId",
		  i."ParentId",
		  a."Client", a."DeviceName", a."DeviceId", a."ApplicationVersion",
		  a."PlayMethod",
		  a."IsPaused",
		  false AS "IsActive",
		  (a."PlaybackDuration" * 10000000)::bigint AS "PlayDuration",
		  a."ActivityDateInserted",
		  a."RemoteEndPoint"
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_music_tracks mt ON mt."Id" = a."EpisodeId"`

	var wheres []string
	var args   []interface{}

	if dateFrom != "" {
		wheres = append(wheres, `a."ActivityDateInserted"::date >= ?::date`)
		args = append(args, dateFrom)
	}
	if dateTo != "" {
		wheres = append(wheres, `a."ActivityDateInserted"::date <= ?::date`)
		args = append(args, dateTo)
	}
	if len(clients) > 0 {
		wheres = append(wheres, inClause(`a."Client"`, len(clients)))
		for _, v := range clients { args = append(args, v) }
	}
	if len(methods) > 0 {
		wheres = append(wheres, inClause(`a."PlayMethod"`, len(methods)))
		for _, v := range methods { args = append(args, v) }
	}
	if len(mediaTypes) > 0 {
		expr := `CASE WHEN mt."Id" IS NOT NULL THEN 'Audio' WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Episode' ELSE COALESCE(i."Type", 'Unknown') END`
		wheres = append(wheres, inClause(expr, len(mediaTypes)))
		for _, v := range mediaTypes { args = append(args, v) }
	}
	if len(devices) > 0 {
		wheres = append(wheres, inClause(`a."DeviceName"`, len(devices)))
		for _, v := range devices { args = append(args, v) }
	}
	if len(users) > 0 {
		wheres = append(wheres, inClause(`a."UserName"`, len(users)))
		for _, v := range users { args = append(args, v) }
	}
	if durMinStr != "" {
		if durMin, err := strconv.Atoi(durMinStr); err == nil {
			wheres = append(wheres, `a."PlaybackDuration" >= ?`)
			args = append(args, durMin*60)
		}
	}
	if durMaxStr != "" {
		if durMax, err := strconv.Atoi(durMaxStr); err == nil {
			wheres = append(wheres, `a."PlaybackDuration" <= ?`)
			args = append(args, durMax*60)
		}
	}

	query := baseSelect
	if len(wheres) > 0 {
		query += "\n\t\tWHERE " + strings.Join(wheres, "\n\t\t  AND ")
	}
	query += `
		ORDER BY a."ActivityDateInserted"::timestamptz DESC`

	var rows []activityRow
	h.db.Raw(query, args...).Scan(&rows)

	if rows == nil {
		rows = []activityRow{}
	}

	if len(wheres) > 0 {
		c.JSON(http.StatusOK, rows)
		return
	}

	live := h.getLiveActivity(c.Request.Context())
	var liveRows []activityRow
	now := time.Now().Format(time.RFC3339)
	for i := range live {
		ls := live[i]
		userId := ls.UserId
		userName := ls.UserName
		client := ls.Client
		pm := ls.PlayMethod
		itemId := ls.NowPlayingItemId
		itemName := ls.NowPlayingItemName
		itemType := ls.Type
		if itemType == "" {
			itemType = "Unknown"
		}
		dur := int64(ls.PlaybackDuration) * 600_000_000
		isActive := true
		isPaused := false
		liveRows = append(liveRows, activityRow{
			UserId:               &userId,
			UserName:             &userName,
			ItemId:               &itemId,
			NowPlayingItemName:   &itemName,
			NowPlayingItemType:   &itemType,
			Client:               &client,
			PlayMethod:           &pm,
			IsPaused:             &isPaused,
			IsActive:             isActive,
			PlayDuration:         &dur,
			ActivityDateInserted: &now,
		})
	}

	result := append(liveRows, rows...)
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserActivity?userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserActivity(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	var rows []activityRow
	h.db.Raw(`
		SELECT
		  "Id", "UserId", "UserName",
		  "NowPlayingItemId" AS "ItemId",
		  "NowPlayingItemName",
		  "SeriesName", "SeasonId", "EpisodeId",
		  "Client", "DeviceName", "DeviceId", "ApplicationVersion",
		  "PlayMethod",
		  "IsPaused",
		  false AS "IsActive",
		  ("PlaybackDuration" * 10000000)::bigint AS "PlayDuration",
		  "ActivityDateInserted",
		  "RemoteEndPoint"
		FROM jf_playback_activity
		WHERE "UserId" = ?
		ORDER BY "ActivityDateInserted"::timestamptz DESC
	`, userId).Scan(&rows)

	if rows == nil {
		rows = []activityRow{}
	}

	live := h.getLiveActivity(c.Request.Context())
	var liveRows []activityRow
	now := time.Now().Format(time.RFC3339)
	for i := range live {
		ls := live[i]
		if ls.UserId != userId {
			continue
		}
		uid := ls.UserId
		uname := ls.UserName
		client := ls.Client
		pm := ls.PlayMethod
		itemId := ls.NowPlayingItemId
		itemName := ls.NowPlayingItemName
		dur := int64(ls.PlaybackDuration) * 600_000_000
		isPaused := false
		liveRows = append(liveRows, activityRow{
			UserId:               &uid,
			UserName:             &uname,
			ItemId:               &itemId,
			NowPlayingItemName:   &itemName,
			Client:               &client,
			PlayMethod:           &pm,
			IsPaused:             &isPaused,
			IsActive:             true,
			PlayDuration:         &dur,
			ActivityDateInserted: &now,
		})
	}

	result := append(liveRows, rows...)
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserActivityByDate?userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserActivityByDate(c *gin.Context) {
	userId := c.Query("userId")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}

	type Row struct {
		Date     string `json:"date"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
		  COUNT(*)::int AS count,
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
		FROM jf_playback_activity
		WHERE "UserId" = ?
		GROUP BY ("ActivityDateInserted"::timestamptz)::date
		ORDER BY date
	`, userId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		if ls.UserId != userId {
			continue
		}
		found := false
		for i := range rows {
			if rows[i].Date == ls.date {
				rows[i].Count++
				rows[i].Duration += ls.PlaybackDuration
				found = true
				break
			}
		}
		if !found {
			rows = append(rows, Row{Date: ls.date, Count: 1, Duration: ls.PlaybackDuration})
		}
	}

	sort.Slice(rows, func(i, j int) bool { return rows[i].Date < rows[j].Date })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraries
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraries(c *gin.Context) {
	type Row struct {
		Id             string  `json:"Id"`
		Name           string  `json:"Name"`
		CollectionType string  `json:"CollectionType"`
		ItemCount      int     `json:"ItemCount"`
		EpisodeCount   int     `json:"EpisodeCount"`
		SeasonCount    int     `json:"SeasonCount"`
		TotalSize      int64   `json:"TotalSize"`
		TotalPlayCount int     `json:"TotalPlayCount"`
		TotalWatchTime int     `json:"TotalWatchTime"`
		LastActivity   *string `json:"LastActivity"`
	}
	var rows []Row
	h.db.Raw(`
		WITH item_stats AS (
		  -- Top-level items: Movies, Series, MusicAlbums — direct children of library
		  SELECT
		    i."ParentId" AS "LibraryId",
		    COUNT(*) FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount",
		    COALESCE(SUM(ii."Size") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder')), 0)::bigint AS "DirectSize"
		  FROM jf_library_items i
		  LEFT JOIN jf_item_info ii ON ii."Id" = i."Id"
		  WHERE i.archived = false
		  GROUP BY i."ParentId"
		),
		episode_stats AS (
		  -- Episodes live in jf_library_episodes; their sizes are in jf_item_info
		  SELECT
		    series."ParentId" AS "LibraryId",
		    COUNT(DISTINCT e."Id")::int AS "EpisodeCount",
		    COALESCE(SUM(ii."Size"), 0)::bigint AS "EpisodeSize"
		  FROM jf_library_episodes e
		  JOIN jf_library_items series
		    ON series."Id" = e."SeriesId" AND series.archived = false
		  LEFT JOIN jf_item_info ii ON ii."Id" = e."Id"
		  WHERE e.archived = false
		  GROUP BY series."ParentId"
		),
		season_stats AS (
		  -- Seasons live in jf_library_seasons
		  SELECT
		    series."ParentId" AS "LibraryId",
		    COUNT(DISTINCT s."Id")::int AS "SeasonCount"
		  FROM jf_library_seasons s
		  JOIN jf_library_items series
		    ON series."Id" = s."SeriesId" AND series.archived = false
		  WHERE s.archived = false
		  GROUP BY series."ParentId"
		),
		track_size_stats AS (
		  -- Music tracks live in jf_music_tracks with a direct LibraryId
		  SELECT
		    t."LibraryId",
		    COALESCE(SUM(ii."Size"), 0)::bigint AS "TrackSize"
		  FROM jf_music_tracks t
		  LEFT JOIN jf_item_info ii ON ii."Id" = t."Id"
		  WHERE t.archived = false AND t."LibraryId" IS NOT NULL
		  GROUP BY t."LibraryId"
		),
		play_stats AS (
		  SELECT
		    COALESCE(i."ParentId", mt."LibraryId") AS "LibraryId",
		    COUNT(a."Id")::int AS "TotalPlayCount",
		    FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
		    MAX(a."ActivityDateInserted") AS "LastActivity"
		  FROM jf_playback_activity a
		  LEFT JOIN jf_library_items i
		    ON (a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id") AND i.archived = false
		  LEFT JOIN jf_music_tracks mt
		    ON a."NowPlayingItemId" = mt."Id" AND mt.archived = false
		  GROUP BY COALESCE(i."ParentId", mt."LibraryId")
		)
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COALESCE(ist."ItemCount", 0)                                          AS "ItemCount",
		  COALESCE(epst."EpisodeCount", 0)                                      AS "EpisodeCount",
		  COALESCE(ssst."SeasonCount", 0)                                       AS "SeasonCount",
		  COALESCE(ist."DirectSize", 0) + COALESCE(epst."EpisodeSize", 0) + COALESCE(tsst."TrackSize", 0) AS "TotalSize",
		  COALESCE(pst."TotalPlayCount", 0)                                     AS "TotalPlayCount",
		  COALESCE(pst."TotalWatchTime", 0)                                     AS "TotalWatchTime",
		  pst."LastActivity"
		FROM jf_libraries l
		LEFT JOIN item_stats   ist  ON ist."LibraryId"  = l."Id"
		LEFT JOIN episode_stats epst ON epst."LibraryId" = l."Id"
		LEFT JOIN season_stats      ssst ON ssst."LibraryId" = l."Id"
		LEFT JOIN track_size_stats  tsst ON tsst."LibraryId" = l."Id"
		LEFT JOIN play_stats        pst  ON pst."LibraryId"  = l."Id"
		WHERE l.archived = false
		ORDER BY l."Name"
	`).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryStats?libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryStats(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type LibStat struct {
		Name           string `json:"Name"`
		TotalItems     int    `json:"TotalItems"`
		TotalPlayCount int    `json:"TotalPlayCount"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	var stat LibStat
	h.db.Raw(`
		SELECT
		  l."Name",
		  COUNT(DISTINCT all_items."Id") FILTER (WHERE all_items."type" NOT IN ('Season', 'Folder'))::int AS "TotalItems",
		  COUNT(a."Id")::int AS "TotalPlayCount",
		  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime"
		FROM jf_libraries l
		LEFT JOIN (
		  SELECT "Id", "ParentId" AS "LibraryId", "Type" AS "type" FROM jf_library_items WHERE archived = false
		  UNION ALL
		  SELECT "Id", "LibraryId", 'Audio' AS "type" FROM jf_music_tracks WHERE archived = false
		) all_items ON all_items."LibraryId" = l."Id"
		LEFT JOIN jf_playback_activity a
		  ON a."NowPlayingItemId" = all_items."Id" OR a."EpisodeId" = all_items."Id"
		WHERE l."Id" = ?
		GROUP BY l."Id", l."Name"
	`, libraryId).Scan(&stat)

	type TopItem struct {
		Id        string `json:"Id"`
		Name      string `json:"Name"`
		Type      string `json:"Type"`
		PlayCount int    `json:"PlayCount"`
	}
	var topItems []TopItem
	h.db.Raw(`
		SELECT
		  all_items."Id",
		  all_items."Name",
		  all_items."type" AS "Type",
		  COUNT(a."Id")::int AS "PlayCount"
		FROM (
		  SELECT "Id", "ParentId" AS "LibraryId", "Name", "Type" AS "type"
		  FROM jf_library_items WHERE "ParentId" = ? AND archived = false AND "Type" NOT IN ('Season', 'Folder')
		  UNION ALL
		  SELECT "Id", "LibraryId", "Name", 'Audio' AS "type"
		  FROM jf_music_tracks WHERE "LibraryId" = ? AND archived = false
		) all_items
		LEFT JOIN jf_playback_activity a
		  ON a."NowPlayingItemId" = all_items."Id" OR a."EpisodeId" = all_items."Id"
		GROUP BY all_items."Id", all_items."Name", all_items."type"
		ORDER BY "PlayCount" DESC, all_items."Name" ASC
		LIMIT 1
	`, libraryId, libraryId).Scan(&topItems)

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		if ls.LibraryId == libraryId {
			stat.TotalPlayCount++
			stat.TotalWatchTime += ls.PlaybackDuration
		}
	}

	var mostPlayed *TopItem
	if len(topItems) > 0 {
		mostPlayed = &topItems[0]
	}

	c.JSON(http.StatusOK, gin.H{
		"Name":           stat.Name,
		"TotalItems":     stat.TotalItems,
		"TotalPlayCount": stat.TotalPlayCount,
		"TotalWatchTime": stat.TotalWatchTime,
		"MostPlayedItem": mostPlayed,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryItems?libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryItems(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type Row struct {
		Id               string   `json:"Id"`
		Name             string   `json:"Name"`
		Type             string   `json:"Type"`
		ProductionYear   *int     `json:"ProductionYear"`
		CommunityRating  *float64 `json:"CommunityRating"`
		Size             *int64   `json:"Size"`
		PlayCount        int      `json:"PlayCount"`
		LastPlayed       *string  `json:"LastPlayed"`
		SeriesName       *string  `json:"SeriesName"`
		IndexNumber      *int     `json:"IndexNumber"`
		ParentIndexNumber *int    `json:"ParentIndexNumber"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  i."Id",
		  i."Name",
		  i."Type",
		  i."ProductionYear",
		  i."CommunityRating",
		  info."Size",
		  COALESCE(pc.play_count, 0)::int AS "PlayCount",
		  pc.last_played AS "LastPlayed",
		  NULL::text AS "SeriesName",
		  NULL::int AS "IndexNumber",
		  NULL::int AS "ParentIndexNumber"
		FROM jf_library_items i
		LEFT JOIN (
		  SELECT
		    COALESCE("EpisodeId", "NowPlayingItemId") AS item_id,
		    COUNT(*)::int AS play_count,
		    MAX("ActivityDateInserted") AS last_played
		  FROM jf_playback_activity
		  GROUP BY item_id
		) pc ON pc.item_id = i."Id"
		LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		WHERE i."ParentId" = ?
		  AND i.archived = false
		  AND i."Type" NOT IN ('Season', 'Folder')
		ORDER BY "PlayCount" DESC NULLS LAST
	`, libraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	for _, ls := range live {
		if ls.LibraryId != libraryId {
			continue
		}
		for i := range rows {
			if rows[i].Id == ls.NowPlayingItemId {
				rows[i].PlayCount++
				break
			}
		}
	}

	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getItemDetails?itemId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetItemDetails(c *gin.Context) {
	itemId := c.Query("itemId")
	if itemId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "itemId is required"})
		return
	}

	type ItemInfo struct {
		Id              string          `json:"Id"`
		Name            string          `json:"Name"`
		Type            string          `json:"Type"`
		ProductionYear  *int            `json:"ProductionYear"`
		CommunityRating *float64        `json:"CommunityRating"`
		PremiereDate    *string         `json:"PremiereDate"`
		DateCreated     *string         `json:"DateCreated"`
		RunTimeTicks    *int64          `json:"RunTimeTicks"`
		Genres          json.RawMessage `json:"Genres"`
		ParentId        *string         `json:"ParentId"`
		Size            *int64          `json:"Size"`
		Path            *string         `json:"Path"`
		Bitrate         *int64          `json:"Bitrate"`
		AlbumId         *string         `json:"AlbumId"`
		AlbumName       *string         `json:"AlbumName"`
		Artist          *string         `json:"Artist"`
	}
	var item ItemInfo
	h.db.Raw(`
		SELECT * FROM (
		  SELECT
		    i."Id",
		    i."Name",
		    i."Type",
		    i."ProductionYear",
		    i."CommunityRating",
		    i."PremiereDate",
		    i."DateCreated",
		    i."RunTimeTicks",
		    i."Genres",
		    i."ParentId",
		    info."Size",
		    info."Path",
		    info."Bitrate",
		    NULL AS "AlbumId",
		    NULL AS "AlbumName",
		    NULL AS "Artist"
		  FROM jf_library_items i
		  LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		  WHERE i."Id" = ?

		  UNION ALL

		  SELECT
		    t."Id",
		    t."Name",
		    'Audio' AS "Type",
		    t."ProductionYear",
		    NULL::float8 AS "CommunityRating",
		    NULL AS "PremiereDate",
		    t."DateCreated",
		    t."RunTimeTicks",
		    t."Genres",
		    t."LibraryId" AS "ParentId",
		    NULL::bigint AS "Size",
		    NULL AS "Path",
		    NULL::bigint AS "Bitrate",
		    t."AlbumId",
		    t."AlbumName",
		    COALESCE(t."AlbumArtist", a."Name") AS "Artist"
		  FROM jf_music_tracks t
		  LEFT JOIN jf_music_artists a ON a."Id" = t."ArtistId"
		  WHERE t."Id" = ?
		) sub LIMIT 1
	`, itemId, itemId).Scan(&item)

	type HistEntry struct {
		Id                   *string `json:"Id"`
		UserId               *string `json:"UserId"`
		UserName             *string `json:"UserName"`
		Client               *string `json:"Client"`
		DeviceName           *string `json:"DeviceName"`
		PlayMethod           *string `json:"PlayMethod"`
		PlaybackDuration     int     `json:"PlaybackDuration"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		RemoteEndPoint       *string `json:"RemoteEndPoint"`
		IsActive             bool    `json:"IsActive"`
	}
	var history []HistEntry
	h.db.Raw(`
		SELECT
		  "Id",
		  "UserId",
		  "UserName",
		  "Client",
		  "DeviceName",
		  "PlayMethod",
		  FLOOR(COALESCE("PlaybackDuration", 0) / 60.0)::int AS "PlaybackDuration",
		  "ActivityDateInserted",
		  "RemoteEndPoint",
		  false AS "IsActive"
		FROM jf_playback_activity
		WHERE "NowPlayingItemId" = ? OR "EpisodeId" = ?
		ORDER BY "ActivityDateInserted"::timestamptz DESC
		LIMIT 200
	`, itemId, itemId).Scan(&history)

	if history == nil {
		history = []HistEntry{}
	}

	live := h.getLiveActivity(c.Request.Context())
	now := time.Now().Format(time.RFC3339)
	for i := range live {
		ls := live[i]
		if ls.NowPlayingItemId != itemId && ls.EpisodeId != itemId {
			continue
		}
		uid := ls.UserId
		uname := ls.UserName
		client := ls.Client
		pm := ls.PlayMethod
		history = append([]HistEntry{{
			UserId:               &uid,
			UserName:             &uname,
			Client:               &client,
			PlayMethod:           &pm,
			PlaybackDuration:     ls.PlaybackDuration,
			ActivityDateInserted: &now,
			IsActive:             true,
		}}, history...)
	}

	// Build stats
	totalPlays := len(history)
	totalWatchTime := 0
	type userAggState struct {
		UserName     string
		PlayCount    int
		WatchTime    int
		LastWatched  *string
	}
	userMap := map[string]*userAggState{}
	var lastWatched *string
	isActive := false
	for _, h2 := range history {
		totalWatchTime += h2.PlaybackDuration
		if h2.UserId != nil {
			uid := *h2.UserId
			if _, ok := userMap[uid]; !ok {
				uname := uid
				if h2.UserName != nil {
					uname = *h2.UserName
				}
				userMap[uid] = &userAggState{UserName: uname}
			}
			st := userMap[uid]
			if h2.UserName != nil {
				st.UserName = *h2.UserName
			}
			st.PlayCount++
			st.WatchTime += h2.PlaybackDuration
			if st.LastWatched == nil && h2.ActivityDateInserted != nil {
				st.LastWatched = h2.ActivityDateInserted
			}
		}
		if lastWatched == nil && h2.ActivityDateInserted != nil {
			lastWatched = h2.ActivityDateInserted
		}
		if h2.IsActive {
			isActive = true
		}
	}

	type UserAgg struct {
		UserId        string  `json:"UserId"`
		UserName      string  `json:"UserName"`
		PlayCount     int     `json:"PlayCount"`
		TotalWatchTime int    `json:"TotalWatchTime"`
		LastWatched   *string `json:"LastWatched"`
	}
	var users []UserAgg
	for uid, st := range userMap {
		users = append(users, UserAgg{
			UserId:         uid,
			UserName:       st.UserName,
			PlayCount:      st.PlayCount,
			TotalWatchTime: st.WatchTime,
			LastWatched:    st.LastWatched,
		})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].PlayCount > users[j].PlayCount })

	c.JSON(http.StatusOK, gin.H{
		"item": item,
		"stats": gin.H{
			"TotalPlays":     totalPlays,
			"TotalWatchTime": totalWatchTime,
			"UniqueUsers":    len(userMap),
			"LastWatched":    lastWatched,
			"IsActive":       isActive,
		},
		"users":   users,
		"history": history,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getGenreStats?libraryId=&userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGenreStats(c *gin.Context) {
	libraryId := c.Query("libraryId")
	userId := c.Query("userId")

	type Row struct {
		Genre     string `json:"Genre"`
		Count     int    `json:"Count"`
		PlayCount int    `json:"PlayCount"`
	}
	var rows []Row

	if userId != "" {
		if libraryId != "" {
			h.db.Raw(`
				SELECT
				  genre AS "Genre",
				  COUNT(DISTINCT COALESCE(a."EpisodeId", a."NowPlayingItemId"))::int AS "Count",
				  COUNT(*)::int AS "PlayCount"
				FROM jf_playback_activity a
				LEFT JOIN jf_library_items i
				  ON a."NowPlayingItemId" = i."Id" AND i."ParentId" = ?
				CROSS JOIN LATERAL jsonb_array_elements_text(
				  CASE
				    WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
				    ELSE i."Genres"
				  END
				) AS genre
				WHERE a."UserId" = ?
				GROUP BY genre
				ORDER BY "PlayCount" DESC, genre ASC
				LIMIT 100
			`, libraryId, userId).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  genre AS "Genre",
				  COUNT(DISTINCT COALESCE(a."EpisodeId", a."NowPlayingItemId"))::int AS "Count",
				  COUNT(*)::int AS "PlayCount"
				FROM jf_playback_activity a
				LEFT JOIN jf_library_items i
				  ON a."NowPlayingItemId" = i."Id"
				CROSS JOIN LATERAL jsonb_array_elements_text(
				  CASE
				    WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
				    ELSE i."Genres"
				  END
				) AS genre
				WHERE a."UserId" = ?
				GROUP BY genre
				ORDER BY "PlayCount" DESC, genre ASC
				LIMIT 100
			`, userId).Scan(&rows)
		}
	} else if libraryId != "" {
		h.db.Raw(`
			SELECT
			  genre AS "Genre",
			  COUNT(*)::int AS "Count",
			  0::int AS "PlayCount"
			FROM (
			  SELECT "Genres" FROM jf_library_items
			  WHERE "ParentId" = ? AND archived = false AND "Type" NOT IN ('Season', 'Folder')
			  UNION ALL
			  SELECT "Genres" FROM jf_music_tracks
			  WHERE "LibraryId" = ? AND archived = false
			) combined
			CROSS JOIN LATERAL jsonb_array_elements_text(
			  CASE
			    WHEN jsonb_array_length(COALESCE(combined."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
			    ELSE combined."Genres"
			  END
			) AS genre
			GROUP BY genre
			ORDER BY "Count" DESC, genre ASC
		`, libraryId, libraryId).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getUserGenreStats?userId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserGenreStats(c *gin.Context) {
	// Delegate to GetGenreStats — userId comes from query param
	h.GetGenreStats(c)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryTracks?libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryTracks(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type Row struct {
		Id           string  `json:"Id"`
		Name         string  `json:"Name"`
		Artist       *string `json:"Artist"`
		AlbumName    *string `json:"AlbumName"`
		AlbumId      *string `json:"AlbumId"`
		IndexNumber  *int    `json:"IndexNumber"`
		DiscNumber   *int    `json:"DiscNumber"`
		RunTimeTicks *int64  `json:"RunTimeTicks"`
		PlayCount    int     `json:"PlayCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  t."Id",
		  t."Name",
		  t."AlbumArtist" AS "Artist",
		  t."AlbumName",
		  t."AlbumId",
		  t."IndexNumber",
		  t."DiscNumber",
		  t."RunTimeTicks",
		  COALESCE(pc.play_count, 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN (
		  SELECT "EpisodeId" AS track_id, COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" <> ''
		  GROUP BY "EpisodeId"
		) pc ON pc.track_id = t."Id"
		WHERE t."LibraryId" = ?
		  AND t.archived = false
		ORDER BY
		  COALESCE(t."AlbumArtist", '') ASC,
		  COALESCE(t."AlbumName", '') ASC,
		  t."DiscNumber" NULLS LAST,
		  t."IndexNumber" NULLS LAST,
		  t."Name" ASC
		LIMIT 5000
	`, libraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryAlbums?libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryAlbums(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		AlbumArtist      *string `json:"AlbumArtist"`
		ArtistId         *string `json:"ArtistId"`
		ProductionYear   *int    `json:"ProductionYear"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  album."Id",
		  album."Name",
		  album."AlbumArtist",
		  album."ArtistId",
		  album."ProductionYear",
		  album."ImageTagsPrimary",
		  COUNT(t."Id")::int AS "TrackCount",
		  COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_library_items album
		LEFT JOIN jf_music_tracks t ON t."AlbumId" = album."Id" AND t.archived = false
		LEFT JOIN (
		  SELECT "EpisodeId" AS track_id, COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" <> ''
		  GROUP BY "EpisodeId"
		) pc ON pc.track_id = t."Id"
		WHERE album."ParentId" = ?
		  AND album.archived = false
		  AND album."Type" = 'MusicAlbum'
		GROUP BY album."Id", album."Name", album."AlbumArtist", album."ArtistId", album."ProductionYear", album."ImageTagsPrimary"
		ORDER BY album."Name"
	`, libraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryArtists?libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryArtists(c *gin.Context) {
	libraryId := c.Query("libraryId")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId is required"})
		return
	}

	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		AlbumCount       int     `json:"AlbumCount"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  COALESCE(t."ArtistId", '') AS "Id",
		  COALESCE(t."AlbumArtist", 'Unknown') AS "Name",
		  ar."ImageTagsPrimary",
		  COUNT(DISTINCT t."AlbumId")::int AS "AlbumCount",
		  COUNT(t."Id")::int AS "TrackCount",
		  COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN jf_music_artists ar ON ar."Id" = t."ArtistId"
		LEFT JOIN (
		  SELECT "EpisodeId" AS track_id, COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" <> ''
		  GROUP BY "EpisodeId"
		) pc ON pc.track_id = t."Id"
		WHERE t."LibraryId" = ?
		  AND t.archived = false
		GROUP BY COALESCE(t."ArtistId", ''), COALESCE(t."AlbumArtist", 'Unknown'), ar."ImageTagsPrimary"
		ORDER BY COALESCE(t."AlbumArtist", 'Unknown')
	`, libraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getArtistAlbums?libraryId=&artistId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetArtistAlbums(c *gin.Context) {
	libraryId := c.Query("libraryId")
	artistId := c.Query("artistId")
	if libraryId == "" || artistId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryId and artistId are required"})
		return
	}

	type Row struct {
		Id               string  `json:"Id"`
		Name             string  `json:"Name"`
		AlbumArtist      *string `json:"AlbumArtist"`
		ArtistId         *string `json:"ArtistId"`
		ProductionYear   *int    `json:"ProductionYear"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		TrackCount       int     `json:"TrackCount"`
		PlayCount        int     `json:"PlayCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  t."AlbumId" AS "Id",
		  COALESCE(album."Name", t."AlbumName") AS "Name",
		  t."AlbumArtist",
		  t."ArtistId",
		  album."ProductionYear",
		  album."ImageTagsPrimary",
		  COUNT(t."Id")::int AS "TrackCount",
		  COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN jf_library_items album ON album."Id" = t."AlbumId" AND album.archived = false
		LEFT JOIN (
		  SELECT "EpisodeId" AS track_id, COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" <> ''
		  GROUP BY "EpisodeId"
		) pc ON pc.track_id = t."Id"
		WHERE t."ArtistId" = ?
		  AND t."LibraryId" = ?
		  AND t.archived = false
		  AND t."AlbumId" IS NOT NULL
		GROUP BY t."AlbumId", album."Name", t."AlbumName", t."AlbumArtist", t."ArtistId", album."ProductionYear", album."ImageTagsPrimary"
		ORDER BY COALESCE(album."Name", t."AlbumName")
	`, artistId, libraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getAlbumTracks?albumId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetAlbumTracks(c *gin.Context) {
	albumId := c.Query("albumId")
	if albumId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "albumId is required"})
		return
	}

	type Row struct {
		Id           string  `json:"Id"`
		Name         string  `json:"Name"`
		IndexNumber  *int    `json:"IndexNumber"`
		DiscNumber   *int    `json:"DiscNumber"`
		RunTimeTicks *int64  `json:"RunTimeTicks"`
		AlbumId      *string `json:"AlbumId"`
		AlbumName    *string `json:"AlbumName"`
		Artist       *string `json:"Artist"`
		PlayCount    int     `json:"PlayCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  t."Id",
		  t."Name",
		  t."IndexNumber",
		  t."DiscNumber",
		  t."RunTimeTicks",
		  t."AlbumId",
		  t."AlbumName",
		  t."AlbumArtist" AS "Artist",
		  COALESCE(pc.play_count, 0)::int AS "PlayCount"
		FROM jf_music_tracks t
		LEFT JOIN (
		  SELECT "EpisodeId" AS track_id, COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" <> ''
		  GROUP BY "EpisodeId"
		) pc ON pc.track_id = t."Id"
		WHERE t."AlbumId" = ?
		  AND t.archived = false
		ORDER BY t."DiscNumber" NULLS LAST, t."IndexNumber" NULLS LAST, t."Name"
	`, albumId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getActivityTimeline
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetActivityTimeline(c *gin.Context) {
	now := time.Now()
	year, _ := strconv.Atoi(c.DefaultQuery("year", strconv.Itoa(now.Year())))
	month, _ := strconv.Atoi(c.DefaultQuery("month", strconv.Itoa(int(now.Month()))))
	if year == 0 {
		year = now.Year()
	}
	if month < 1 || month > 12 {
		month = int(now.Month())
	}

	type Row struct {
		Id         *string `json:"Id"`
		UserId     *string `json:"UserId"`
		UserName   *string `json:"UserName"`
		ItemId     *string `json:"ItemId"`
		ItemName   *string `json:"ItemName"`
		StartTime  *string `json:"StartTime"`
		EndTime    *string `json:"EndTime"`
		Duration   int     `json:"Duration"`
		Client     *string `json:"Client"`
		PlayMethod *string `json:"PlayMethod"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  a."Id",
		  a."UserId",
		  a."UserName",
		  a."NowPlayingItemId" AS "ItemId",
		  a."NowPlayingItemName" AS "ItemName",
		  (a."ActivityDateInserted"::timestamptz - (COALESCE(a."PlaybackDuration", 0) * INTERVAL '1 second'))::text AS "StartTime",
		  a."ActivityDateInserted" AS "EndTime",
		  COALESCE(a."PlaybackDuration", 0)::int AS "Duration",
		  a."Client",
		  a."PlayMethod"
		FROM jf_playback_activity a
		WHERE a."ActivityDateInserted"::timestamptz >= make_date(?, ?, 1)::timestamptz
		  AND a."ActivityDateInserted"::timestamptz  < make_date(?, ?, 1)::timestamptz + INTERVAL '1 month'
		ORDER BY a."ActivityDateInserted"::timestamptz DESC
	`, year, month, year, month).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	var liveRows []Row
	for i := range live {
		ls := live[i]
		endTime := now.Format(time.RFC3339)
		startTime := now.Add(-time.Duration(ls.PlaybackDuration) * time.Minute).Format(time.RFC3339)
		uid := ls.UserId
		uname := ls.UserName
		client := ls.Client
		pm := ls.PlayMethod
		itemId := ls.NowPlayingItemId
		itemName := ls.NowPlayingItemName
		liveRows = append(liveRows, Row{
			UserId:     &uid,
			UserName:   &uname,
			ItemId:     &itemId,
			ItemName:   &itemName,
			StartTime:  &startTime,
			EndTime:    &endTime,
			Duration:   ls.PlaybackDuration,
			Client:     &client,
			PlayMethod: &pm,
		})
	}

	result := append(liveRows, rows...)
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// POST /stats/getMostViewedLibraries  body: {days}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostViewedLibraries(c *gin.Context) {
	var body struct {
		Days *int `json:"days"`
	}
	_ = c.ShouldBindJSON(&body)
	days := 30
	if body.Days != nil && *body.Days > 0 {
		days = *body.Days
	}

	type Row struct {
		Name  string `json:"Name"`
		Count int    `json:"Count"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT l."Name", COUNT(DISTINCT a."Id")::int AS "Count"
		FROM jf_libraries l
		LEFT JOIN (
		  SELECT "Id", "ParentId" AS "LibraryId" FROM jf_library_items WHERE archived = false
		  UNION ALL
		  SELECT "Id", "LibraryId" FROM jf_music_tracks WHERE archived = false
		) all_items ON all_items."LibraryId" = l."Id"
		LEFT JOIN jf_playback_activity a
		  ON (a."NowPlayingItemId" = all_items."Id" OR a."EpisodeId" = all_items."Id")
		  AND a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		WHERE l.archived = false
		GROUP BY l."Id", l."Name"
		ORDER BY "Count" DESC
	`, days).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryLastPlayed  body: {libraryid}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryLastPlayed(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	type Row struct {
		Id                   *string `json:"Id"`
		UserId               *string `json:"UserId"`
		UserName             *string `json:"UserName"`
		NowPlayingItemId     *string `json:"NowPlayingItemId"`
		NowPlayingItemName   *string `json:"NowPlayingItemName"`
		SeriesName           *string `json:"SeriesName"`
		Client               *string `json:"Client"`
		PlayMethod           *string `json:"PlayMethod"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		PlaybackDuration     int     `json:"PlaybackDuration"`
		ItemType             *string `json:"ItemType"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  a."Id", a."UserId", a."UserName", a."NowPlayingItemId", a."NowPlayingItemName", a."SeriesName",
		  a."Client", a."PlayMethod", a."ActivityDateInserted",
		  FLOOR(COALESCE(a."PlaybackDuration", 0) / 60.0)::int AS "PlaybackDuration",
		  COALESCE(i."Type", mt."Id"::text) AS "ItemType"
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId" AND i."ParentId" = ?
		LEFT JOIN jf_music_tracks mt ON mt."Id" = a."NowPlayingItemId" AND mt."LibraryId" = ?
		WHERE i."Id" IS NOT NULL OR mt."Id" IS NOT NULL
		ORDER BY a."ActivityDateInserted"::timestamptz DESC
		LIMIT 15
	`, body.LibraryId, body.LibraryId).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getGlobalUserStats  body: {userid, hours?}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGlobalUserStats(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
		Hours  *int   `json:"hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	hours := 24
	if body.Hours != nil && *body.Hours > 0 {
		hours = *body.Hours
	}

	var stat struct {
		TotalPlays   int `json:"TotalPlays"`
		TotalWatchTime int `json:"TotalWatchTime"`
		UniqueItems  int `json:"UniqueItems"`
	}
	h.db.Raw(`
		SELECT
		  COUNT(*)::int AS "TotalPlays",
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
		  COUNT(DISTINCT "NowPlayingItemId")::int AS "UniqueItems"
		FROM jf_playback_activity
		WHERE "UserId" = ?
		  AND "ActivityDateInserted"::timestamptz >= NOW() - MAKE_INTERVAL(hours => ?)
	`, body.UserId, hours).Scan(&stat)

	c.JSON(http.StatusOK, stat)
}

// ---------------------------------------------------------------------------
// POST /stats/getUserLastPlayed  body: {userid}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUserLastPlayed(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}

	var rows []map[string]interface{}
	h.db.Raw(`
		SELECT * FROM jf_playback_activity
		WHERE "UserId" = ?
		ORDER BY "ActivityDateInserted"::timestamptz DESC
		LIMIT 15
	`, body.UserId).Scan(&rows)

	if rows == nil {
		rows = []map[string]interface{}{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryCardStats  (no libraryid = all)
// POST /stats/getLibraryCardStats  body: {libraryid}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryCardStatsGET(c *gin.Context) {
	type Row struct {
		Id                   string  `json:"Id"`
		Name                 string  `json:"Name"`
		CollectionType       string  `json:"CollectionType"`
		ItemCount            int     `json:"ItemCount"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		LastActivity         *string `json:"LastActivity"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder', 'MusicAlbum'))::int AS "ItemCount",
		  (SELECT MAX(a."ActivityDateInserted")::timestamptz FROM jf_playback_activity a
		   JOIN jf_library_items li ON li."Id" = a."NowPlayingItemId" AND li."ParentId" = l."Id") AS "ActivityDateInserted",
		  (now() - (SELECT MAX(a."ActivityDateInserted")::timestamptz FROM jf_playback_activity a
		   JOIN jf_library_items li ON li."Id" = a."NowPlayingItemId" AND li."ParentId" = l."Id"))::text AS "LastActivity"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
		WHERE l.archived = false
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
		ORDER BY l."Name"
	`).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

func (h *StatsFrontendHandler) GetLibraryCardStatsPOST(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	type Row struct {
		Id                   string  `json:"Id"`
		Name                 string  `json:"Name"`
		CollectionType       string  `json:"CollectionType"`
		ItemCount            int     `json:"ItemCount"`
		ActivityDateInserted *string `json:"ActivityDateInserted"`
		LastActivity         *string `json:"LastActivity"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder', 'MusicAlbum'))::int AS "ItemCount",
		  (SELECT MAX(a."ActivityDateInserted")::timestamptz FROM jf_playback_activity a
		   JOIN jf_library_items li ON li."Id" = a."NowPlayingItemId" AND li."ParentId" = l."Id") AS "ActivityDateInserted",
		  (now() - (SELECT MAX(a."ActivityDateInserted")::timestamptz FROM jf_playback_activity a
		   JOIN jf_library_items li ON li."Id" = a."NowPlayingItemId" AND li."ParentId" = l."Id"))::text AS "LastActivity"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
		WHERE l.archived = false AND l."Id" = ?
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
	`, body.LibraryId).Scan(&rows)

	if len(rows) == 0 {
		c.JSON(http.StatusOK, nil)
		return
	}
	c.JSON(http.StatusOK, rows[0])
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryMetadata
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryMetadata(c *gin.Context) {
	type Row struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		ItemCount      int    `json:"ItemCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(DISTINCT i."Id") FILTER (WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id"
		WHERE l.archived = false
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
		ORDER BY l."Name"
	`).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getLibraryOverview
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryOverview(c *gin.Context) {
	type Row struct {
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		TotalItems     int    `json:"TotalItems"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "TotalItems"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
		WHERE l.archived = false
		GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
		ORDER BY l."Name"
	`).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getViewsByLibraryType?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetViewsByLibraryType(c *gin.Context) {
	days := parseDays(c.DefaultQuery("days", "30"), 30)

	type Row struct {
		Type  *string `json:"type"`
		Count int     `json:"count"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT COALESCE(i."Type", 'Other') AS type, COUNT(a."NowPlayingItemId") AS count
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		WHERE a."ActivityDateInserted"::timestamptz >= NOW() - MAKE_INTERVAL(days => ?)
		GROUP BY i."Type"
	`, days).Scan(&rows)

	result := map[string]int{
		"Audio":  0,
		"Movie":  0,
		"Series": 0,
		"Other":  0,
	}
	for _, r := range rows {
		t := "Other"
		if r.Type != nil {
			t = *r.Type
		}
		switch t {
		case "Audio":
			result["Audio"] += r.Count
		case "Movie", "Video":
			result["Movie"] += r.Count
		case "Series", "Episode":
			result["Series"] += r.Count
		default:
			result["Other"] += r.Count
		}
	}
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// GET /stats/getGenreUserStats?userid=&type=Movie|Episode|Audio&size=10&page=1
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGenreUserStats(c *gin.Context) {
	userId := c.Query("userid")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	mediaType := c.Query("type") // optional filter: Movie, Episode, Audio
	size := parseDays(c.DefaultQuery("size", "10"), 10)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size

	// Optional type filter injected via fmt.Sprintf — the ? is a GORM placeholder
	typeFilter := ""
	if mediaType != "" {
		typeFilter = `AND i."Type" = ?`
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM (
		  SELECT COALESCE(g.genre, 'No Genre') AS genre
		  FROM jf_playback_activity a
		  INNER JOIN jf_library_items i ON a."NowPlayingItemId" = i."Id"
		  LEFT JOIN LATERAL (
		    SELECT jsonb_array_elements_text(
		      CASE WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0
		           THEN '["No Genre"]'::jsonb
		           ELSE i."Genres" END
		    ) AS genre
		  ) g ON true
		  WHERE a."UserId" = ? %s
		  GROUP BY COALESCE(g.genre, 'No Genre')
		) sub
	`, typeFilter)

	dataQuery := fmt.Sprintf(`
		SELECT COALESCE(g.genre, 'No Genre') AS genre,
		       SUM(a."PlaybackDuration") AS duration,
		       COUNT(*) AS plays
		FROM jf_playback_activity a
		INNER JOIN jf_library_items i ON a."NowPlayingItemId" = i."Id"
		LEFT JOIN LATERAL (
		  SELECT jsonb_array_elements_text(
		    CASE WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0
		         THEN '["No Genre"]'::jsonb
		         ELSE i."Genres" END
		  ) AS genre
		) g ON true
		WHERE a."UserId" = ? %s
		GROUP BY COALESCE(g.genre, 'No Genre')
		ORDER BY plays DESC
		LIMIT ? OFFSET ?
	`, typeFilter)

	var total int
	if mediaType != "" {
		h.db.Raw(countQuery, userId, mediaType).Scan(&total)
	} else {
		h.db.Raw(countQuery, userId).Scan(&total)
	}

	type Row struct {
		Genre    string `json:"genre"`
		Duration int64  `json:"duration"`
		Plays    int    `json:"plays"`
	}
	var rows []Row
	if mediaType != "" {
		h.db.Raw(dataQuery, userId, mediaType, size, offset).Scan(&rows)
	} else {
		h.db.Raw(dataQuery, userId, size, offset).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, paginate(total, size, page, rows))
}

// ---------------------------------------------------------------------------
// GET /stats/getGenreLibraryStats?libraryid=&size=50&page=1
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGenreLibraryStats(c *gin.Context) {
	libraryId := c.Query("libraryid")
	if libraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	size := parseDays(c.DefaultQuery("size", "50"), 50)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size

	var total int
	h.db.Raw(`
		SELECT COUNT(*) FROM (
		  SELECT COALESCE(g.genre, 'No Genre') AS genre
		  FROM jf_playback_activity a
		  INNER JOIN jf_library_items i ON a."NowPlayingItemId" = i."Id"
		  LEFT JOIN LATERAL (
		    SELECT jsonb_array_elements_text(
		      CASE WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0
		           THEN '["No Genre"]'::jsonb
		           ELSE i."Genres" END
		    ) AS genre
		  ) g ON true
		  WHERE i."ParentId" = ?
		  GROUP BY COALESCE(g.genre, 'No Genre')
		) sub
	`, libraryId).Scan(&total)

	type Row struct {
		Genre    string `json:"genre"`
		Duration int64  `json:"duration"`
		Plays    int    `json:"plays"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT COALESCE(g.genre, 'No Genre') AS genre,
		       SUM(a."PlaybackDuration") AS duration,
		       COUNT(*) AS plays
		FROM jf_playback_activity a
		INNER JOIN jf_library_items i ON a."NowPlayingItemId" = i."Id"
		LEFT JOIN LATERAL (
		  SELECT jsonb_array_elements_text(
		    CASE WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0
		         THEN '["No Genre"]'::jsonb
		         ELSE i."Genres" END
		  ) AS genre
		) g ON true
		WHERE i."ParentId" = ?
		GROUP BY COALESCE(g.genre, 'No Genre')
		ORDER BY genre ASC
		LIMIT ? OFFSET ?
	`, libraryId, size, offset).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, paginate(total, size, page, rows))
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybackActivity?size=50&page=1&search=&sort=&desc=true
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybackActivity(c *gin.Context) {
	size := parseDays(c.DefaultQuery("size", "50"), 50)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size
	search := c.DefaultQuery("search", "")
	sortCol := c.DefaultQuery("sort", "ActivityDateInserted")
	descStr := c.DefaultQuery("desc", "true")
	isDesc := descStr != "false"

	// Whitelist sort columns
	allowedSort := map[string]string{
		"ActivityDateInserted": `a."ActivityDateInserted"`,
		"UserName":             `a."UserName"`,
		"NowPlayingItemName":   `a."NowPlayingItemName"`,
		"PlaybackDuration":     `a."PlaybackDuration"`,
		"Client":               `a."Client"`,
		"PlayMethod":           `a."PlayMethod"`,
	}
	sortExpr, ok := allowedSort[sortCol]
	if !ok {
		sortExpr = `a."ActivityDateInserted"`
	}
	order := "DESC"
	if !isDesc {
		order = "ASC"
	}

	var total int
	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		h.db.Raw(`
			SELECT COUNT(*) FROM jf_playback_activity a
			WHERE LOWER(CASE WHEN a."SeriesName" IS NULL THEN a."NowPlayingItemName" ELSE a."SeriesName" END) LIKE ?
		`, searchLike).Scan(&total)
	} else {
		h.db.Raw(`SELECT COUNT(*) FROM jf_playback_activity a`).Scan(&total)
	}

	var rows []map[string]interface{}
	q := `SELECT * FROM jf_playback_activity a`
	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		q += ` WHERE LOWER(CASE WHEN a."SeriesName" IS NULL THEN a."NowPlayingItemName" ELSE a."SeriesName" END) LIKE ?`
		q += fmt.Sprintf(` ORDER BY %s %s LIMIT ? OFFSET ?`, sortExpr, order)
		h.db.Raw(q, searchLike, size, offset).Scan(&rows)
	} else {
		q += fmt.Sprintf(` ORDER BY %s %s LIMIT ? OFFSET ?`, sortExpr, order)
		h.db.Raw(q, size, offset).Scan(&rows)
	}

	if rows == nil {
		rows = []map[string]interface{}{}
	}

	pr := paginate(total, size, page, rows)
	c.JSON(http.StatusOK, gin.H{
		"current_page": pr.CurrentPage,
		"pages":        pr.Pages,
		"size":         pr.Size,
		"sort":         sortCol,
		"desc":         isDesc,
		"results":      pr.Results,
	})
}

// ---------------------------------------------------------------------------
// POST /stats/getPlaybackMethodStats  body: {days?}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybackMethodStats(c *gin.Context) {
	var body struct {
		Days *int `json:"days"`
	}
	_ = c.ShouldBindJSON(&body)
	days := 30
	if body.Days != nil && *body.Days > 0 {
		days = *body.Days
	}

	type Row struct {
		Name  *string `json:"Name"`
		Count int     `json:"Count"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT a."PlayMethod" AS "Name", count(a."PlayMethod") AS "Count"
		FROM jf_playback_activity a
		WHERE a."ActivityDateInserted"::timestamptz BETWEEN CURRENT_DATE - MAKE_INTERVAL(days => ?) AND NOW()
		GROUP BY a."PlayMethod"
		ORDER BY (count(*)) DESC
	`, days).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryItemsWithStats  body: {libraryid}
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetLibraryItemsWithStats(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	size := parseDays(c.DefaultQuery("size", "999999999"), 999999999)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * size
	search := c.DefaultQuery("search", "")
	sortParam := c.DefaultQuery("sort", "Date")
	descStr := c.DefaultQuery("desc", "true")
	isDesc := descStr != "false"

	sortMap := map[string]string{
		"Date":      `i."DateCreated"`,
		"Views":     `pc.times_played`,
		"Size":      `info."Size"`,
		"WatchTime": `pc.total_play_time`,
		"Title":     `i."Name"`,
	}
	sortExpr, ok := sortMap[sortParam]
	if !ok {
		sortExpr = `i."DateCreated"`
	}
	order := "DESC"
	if !isDesc {
		order = "ASC"
	}

	baseFrom := `
		FROM jf_library_items i
		LEFT JOIN (
		  SELECT COALESCE("EpisodeId", "NowPlayingItemId") AS item_id,
		    COUNT(*)::int AS times_played,
		    COALESCE(SUM("PlaybackDuration"), 0)::int AS total_play_time
		  FROM jf_playback_activity
		  GROUP BY item_id
		) pc ON pc.item_id = i."Id"
		LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		WHERE i."ParentId" = ?
		  AND i.archived = false
		  AND i."Type" NOT IN ('Season', 'Folder')
	`

	var total int
	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		h.db.Raw(`SELECT COUNT(*) `+baseFrom+` AND LOWER(i."Name") LIKE ?`, body.LibraryId, searchLike).Scan(&total)
	} else {
		h.db.Raw(`SELECT COUNT(*) `+baseFrom, body.LibraryId).Scan(&total)
	}

	type Row struct {
		Id             string   `json:"Id"`
		Name           string   `json:"Name"`
		Type           string   `json:"Type"`
		ProductionYear *int     `json:"ProductionYear"`
		CommunityRating *float64 `json:"CommunityRating"`
		DateCreated    *string  `json:"DateCreated"`
		Size           *int64   `json:"Size"`
		TimesPlayed    int      `json:"times_played"`
		TotalPlayTime  int      `json:"total_play_time"`
	}
	var rows []Row

	selectClause := `SELECT i."Id", i."Name", i."Type", i."ProductionYear", i."CommunityRating", i."DateCreated", info."Size",
		COALESCE(pc.times_played, 0) AS times_played, COALESCE(pc.total_play_time, 0) AS total_play_time`

	orderClause := fmt.Sprintf(` ORDER BY %s %s NULLS LAST LIMIT ? OFFSET ?`, sortExpr, order)

	if search != "" {
		searchLike := "%" + strings.ToLower(search) + "%"
		h.db.Raw(selectClause+baseFrom+` AND LOWER(i."Name") LIKE ?`+orderClause,
			body.LibraryId, searchLike, size, offset).Scan(&rows)
	} else {
		h.db.Raw(selectClause+baseFrom+orderClause, body.LibraryId, size, offset).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}

	pr := paginate(total, size, page, rows)
	c.JSON(http.StatusOK, gin.H{
		"current_page": pr.CurrentPage,
		"pages":        pr.Pages,
		"size":         pr.Size,
		"sort":         sortParam,
		"desc":         isDesc,
		"results":      pr.Results,
	})
}

// ---------------------------------------------------------------------------
// POST /stats/getLibraryItemsPlayMethodStats  body: {libraryid, startDate?, endDate?, hours?}
// ---------------------------------------------------------------------------

type playMethodRecord struct {
	StartTime  time.Time
	EndTime    time.Time
	PlayMethod string
}

func countOverlapsPerHour(records []playMethodRecord) map[string]struct{ Transcodes, DirectPlays int } {
	type hourCount struct{ Transcodes, DirectPlays int }
	hourCounts := map[string]hourCount{}

	for _, r := range records {
		start := r.StartTime.Add(-time.Hour)
		end := r.EndTime.Add(time.Hour)

		for t := start.Truncate(time.Hour); t.Before(end); t = t.Add(time.Hour) {
			key := t.Format("Jan 02, 06 15:00")
			hc := hourCounts[key]
			if r.PlayMethod == "Transcode" {
				hc.Transcodes++
			} else {
				hc.DirectPlays++
			}
			hourCounts[key] = hc
		}
	}

	result := map[string]struct{ Transcodes, DirectPlays int }{}
	for k, v := range hourCounts {
		result[k] = struct{ Transcodes, DirectPlays int }{v.Transcodes, v.DirectPlays}
	}
	return result
}

func (h *StatsFrontendHandler) GetLibraryItemsPlayMethodStats(c *gin.Context) {
	var body struct {
		LibraryId string  `json:"libraryid"`
		StartDate *string `json:"startDate"`
		EndDate   *string `json:"endDate"`
		Hours     *int    `json:"hours"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}

	now := time.Now()

	hasTimeFilter := (body.Hours != nil && *body.Hours > 0) ||
		(body.StartDate != nil && *body.StartDate != "") ||
		(body.EndDate != nil && *body.EndDate != "")

	startTime := time.Time{}
	endTime := now
	if hasTimeFilter {
		hours := 24
		if body.Hours != nil && *body.Hours > 0 {
			hours = *body.Hours
		}
		startTime = now.Add(-time.Duration(hours) * time.Hour)
		if body.StartDate != nil && *body.StartDate != "" {
			if t, err := time.Parse("2006-01-02", *body.StartDate); err == nil {
				startTime = t
			}
		}
		if body.EndDate != nil && *body.EndDate != "" {
			if t, err := time.Parse("2006-01-02", *body.EndDate); err == nil {
				endTime = t.Add(24 * time.Hour)
			}
		}
	}

	type RawRow struct {
		ActivityDateInserted string  `gorm:"column:ActivityDateInserted"`
		PlaybackDuration     *int64  `gorm:"column:PlaybackDuration"`
		PlayMethod           *string `gorm:"column:PlayMethod"`
	}
	var rawRows []RawRow
	query := h.db.Raw(`
		SELECT a."ActivityDateInserted", a."PlaybackDuration", a."PlayMethod"
		FROM jf_playback_activity a
		JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		WHERE i."ParentId" = ?
		ORDER BY a."ActivityDateInserted" DESC
	`, body.LibraryId)
	if hasTimeFilter {
		query = h.db.Raw(`
			SELECT a."ActivityDateInserted", a."PlaybackDuration", a."PlayMethod"
			FROM jf_playback_activity a
			JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
			WHERE i."ParentId" = ?
			AND a."ActivityDateInserted" BETWEEN ? AND ?
			ORDER BY a."ActivityDateInserted" DESC
		`, body.LibraryId, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339))
	}
	query.Scan(&rawRows)

	var records []playMethodRecord
	for _, r := range rawRows {
		endT, err := time.Parse(time.RFC3339, r.ActivityDateInserted)
		if err != nil {
			endT = now
		}
		dur := int64(0)
		if r.PlaybackDuration != nil {
			dur = *r.PlaybackDuration
		}
		pm := ""
		if r.PlayMethod != nil {
			pm = *r.PlayMethod
		}
		records = append(records, playMethodRecord{
			StartTime:  endT.Add(-time.Duration(dur) * time.Second),
			EndTime:    endT,
			PlayMethod: pm,
		})
	}

	overlaps := countOverlapsPerHour(records)

	type StatEntry struct {
		Key         string `json:"Key"`
		Transcodes  int    `json:"Transcodes"`
		DirectPlays int    `json:"DirectPlays"`
	}
	var stats []StatEntry
	for key, v := range overlaps {
		stats = append(stats, StatEntry{Key: key, Transcodes: v.Transcodes, DirectPlays: v.DirectPlays})
	}
	sort.Slice(stats, func(i, j int) bool { return stats[i].Key < stats[j].Key })

	c.JSON(http.StatusOK, gin.H{
		"types": []gin.H{
			{"Id": "Transcodes", "Name": "Transcodes"},
			{"Id": "DirectPlays", "Name": "DirectPlays"},
		},
		"stats": stats,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybacksByLibraryOverTime?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybacksByLibraryOverTime(c *gin.Context) {
	rawDays, _ := strconv.Atoi(c.DefaultQuery("days", "30"))
	allTime := rawDays <= 0

	type Row struct {
		Date        string `json:"date"        gorm:"column:date"`
		LibraryId   string `json:"libraryId"   gorm:"column:LibraryId"`
		LibraryName string `json:"libraryName" gorm:"column:LibraryName"`
		Count       int    `json:"count"       gorm:"column:count"`
	}
	var rows []Row

	baseQuery := `
		SELECT
		  TO_CHAR(("ActivityDateInserted"::timestamptz)::date, 'YYYY-MM-DD') AS date,
		  l."Id" AS "LibraryId",
		  l."Name" AS "LibraryName",
		  COUNT(DISTINCT a."Id")::int AS count
		FROM jf_libraries l
		JOIN (
		  SELECT "Id", "ParentId" AS "LibraryId" FROM jf_library_items WHERE archived = false
		  UNION ALL
		  SELECT "Id", "LibraryId" FROM jf_music_tracks WHERE archived = false
		) all_items ON all_items."LibraryId" = l."Id"
		JOIN jf_playback_activity a
		  ON (a."NowPlayingItemId" = all_items."Id" OR a."EpisodeId" = all_items."Id")`

	if allTime {
		h.db.Raw(baseQuery+`
		WHERE l.archived = false
		GROUP BY date, l."Id", l."Name"
		ORDER BY date, l."Name"
		`).Scan(&rows)
	} else {
		days := rawDays - 1
		h.db.Raw(baseQuery+`
		  AND a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		WHERE l.archived = false
		GROUP BY date, l."Id", l."Name"
		ORDER BY date, l."Name"
		`, days).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPlaybacksScatter?days=30  — one point per play (x=time, y=minutes)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPlaybacksScatter(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	daysArg := days

	type Row struct {
		Ts       string `json:"ts"       gorm:"column:ts"`
		Duration int    `json:"duration" gorm:"column:duration"`
		Name     string `json:"name"     gorm:"column:name"`
		Type     string `json:"type"     gorm:"column:type"`
	}
	var rows []Row

	base := `
		SELECT
		  TO_CHAR(a."ActivityDateInserted"::timestamptz, 'YYYY-MM-DD"T"HH24:MI:SS"Z"') AS ts,
		  GREATEST(FLOOR(a."PlaybackDuration" / 60.0)::int, 1)                          AS duration,
		  COALESCE(a."NowPlayingItemName", '')                                            AS name,
		  CASE
		    WHEN a."EpisodeId" IS NOT NULL AND a."EpisodeId" != '' THEN 'Episode'
		    WHEN mt."Id" IS NOT NULL                               THEN 'Audio'
		    ELSE 'Movie'
		  END AS type
		FROM jf_playback_activity a
		LEFT JOIN jf_music_tracks mt ON mt."Id" = a."NowPlayingItemId" AND mt.archived = false
		WHERE a."PlaybackDuration" > 0`

	if allTime {
		h.db.Raw(base+`
		ORDER BY a."ActivityDateInserted"::timestamptz DESC
		LIMIT 3000
		`).Scan(&rows)
	} else {
		h.db.Raw(base+`
		  AND a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		ORDER BY a."ActivityDateInserted"::timestamptz DESC
		LIMIT 3000
		`, daysArg).Scan(&rows)
	}

	if rows == nil {
		rows = []Row{}
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// Watch Heatmap (hour × day matrix)
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetWatchHeatmap(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	userId := c.Query("userId")

	query := `
SELECT
  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
  COUNT(*)::int AS plays,
  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
FROM jf_playback_activity
WHERE 1=1`

	args := []interface{}{}

	if !allTime {
		query += ` AND "ActivityDateInserted"::timestamptz >= NOW() - MAKE_INTERVAL(days => ?)`
		args = append(args, days)
	}
	if userId != "" {
		query += ` AND "UserId" = ?`
		args = append(args, userId)
	}

	query += ` GROUP BY day, hour ORDER BY day, hour`

	type sqlRow struct {
		Day      int `json:"day"`
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}

	var sqlRows []sqlRow
	if err := h.db.Raw(query, args...).Scan(&sqlRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Build lookup from SQL results
	type key struct{ day, hour int }
	lookup := make(map[key]sqlRow, len(sqlRows))
	for _, r := range sqlRows {
		lookup[key{r.Day, r.Hour}] = r
	}

	// Fill 7×24 grid
	type cell struct {
		Day      int `json:"day"`
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	cells := make([]cell, 0, 7*24)
	maxPlays := 0
	maxDuration := 0

	for d := 0; d < 7; d++ {
		for hr := 0; hr < 24; hr++ {
			cl := cell{Day: d, Hour: hr}
			if r, ok := lookup[key{d, hr}]; ok {
				cl.Plays = r.Plays
				cl.Duration = r.Duration
			}
			if cl.Plays > maxPlays {
				maxPlays = cl.Plays
			}
			if cl.Duration > maxDuration {
				maxDuration = cl.Duration
			}
			cells = append(cells, cl)
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"cells":       cells,
		"maxPlays":    maxPlays,
		"maxDuration": maxDuration,
	})
}

// ---------------------------------------------------------------------------
// GetTimeToWatch – time between item added and first playback
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetTimeToWatch(c *gin.Context) {
	libraryID := c.Query("libraryId")
	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	type row struct {
		Id           string
		Name         string
		Type         string
		DaysToWatch  float64
		DateAdded    string
		FirstWatched string
	}

	query := `
WITH first_watch AS (
  SELECT
    COALESCE(a."NowPlayingItemId", a."EpisodeId") AS item_id,
    MIN(a."ActivityDateInserted"::timestamptz) AS first_played
  FROM jf_playback_activity a
  WHERE a."PlaybackDuration" > 0
  GROUP BY COALESCE(a."NowPlayingItemId", a."EpisodeId")
),
item_dates AS (
  SELECT i."Id", i."Name", i."Type", i."DateCreated"::timestamptz AS date_added, i."ParentId"
  FROM jf_library_items i
  WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder')
  UNION ALL
  SELECT e."Id", e."Name", e."Type", e."DateCreated"::timestamptz AS date_added, s."ParentId"
  FROM jf_library_episodes e
  JOIN jf_library_items s ON s."Id" = e."SeriesId"
  WHERE e.archived = false
)
SELECT id."Id", id."Name", id."Type",
       EXTRACT(EPOCH FROM (fw.first_played - id.date_added)) / 86400.0 AS days_to_watch,
       TO_CHAR(id.date_added, 'YYYY-MM-DD') AS date_added,
       TO_CHAR(fw.first_played, 'YYYY-MM-DD') AS first_watched
FROM item_dates id
JOIN first_watch fw ON fw.item_id = id."Id"
WHERE id.date_added IS NOT NULL AND fw.first_played >= id.date_added`

	var args []interface{}
	if libraryID != "" {
		query += ` AND id."ParentId" = ?`
		args = append(args, libraryID)
	}
	query += ` ORDER BY days_to_watch DESC`

	var rows []row
	if err := h.db.Raw(query, args...).Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if len(rows) == 0 {
		c.JSON(http.StatusOK, gin.H{
			"avgDaysToWatch":    0,
			"medianDaysToWatch": 0,
			"distribution":     []gin.H{},
			"slowestItems":     []gin.H{},
			"fastestItems":     []gin.H{},
		})
		return
	}

	// Compute avg
	var total float64
	for _, r := range rows {
		total += r.DaysToWatch
	}
	avg := total / float64(len(rows))

	// Compute median (rows are sorted desc by days_to_watch)
	n := len(rows)
	var median float64
	if n%2 == 0 {
		median = (rows[n/2-1].DaysToWatch + rows[n/2].DaysToWatch) / 2.0
	} else {
		median = rows[n/2].DaysToWatch
	}

	// Distribution buckets
	type bucket struct {
		Label string
		Max   float64 // exclusive upper bound; -1 means infinity
	}
	buckets := []bucket{
		{"Same day", 1},
		{"1-3 days", 4},
		{"4-7 days", 8},
		{"1-2 weeks", 15},
		{"2-4 weeks", 29},
		{"1-3 months", 91},
		{"3+ months", -1},
	}
	counts := make([]int, len(buckets))
	for _, r := range rows {
		for i, b := range buckets {
			if b.Max < 0 || r.DaysToWatch < b.Max {
				counts[i]++
				break
			}
		}
	}
	type distEntry struct {
		Bucket string `json:"bucket"`
		Count  int    `json:"count"`
	}
	distribution := make([]distEntry, len(buckets))
	for i, b := range buckets {
		distribution[i] = distEntry{Bucket: b.Label, Count: counts[i]}
	}

	// Slowest (rows already sorted desc)
	slowLimit := limit
	if slowLimit > len(rows) {
		slowLimit = len(rows)
	}
	slowest := rows[:slowLimit]

	// Fastest (end of slice)
	fastLimit := limit
	if fastLimit > len(rows) {
		fastLimit = len(rows)
	}
	fastest := make([]row, fastLimit)
	copy(fastest, rows[len(rows)-fastLimit:])
	// Reverse so fastest first
	for i, j := 0, len(fastest)-1; i < j; i, j = i+1, j-1 {
		fastest[i], fastest[j] = fastest[j], fastest[i]
	}

	// Build JSON items
	toJSON := func(items []row) []gin.H {
		out := make([]gin.H, len(items))
		for i, r := range items {
			out[i] = gin.H{
				"id":           r.Id,
				"name":         r.Name,
				"type":         r.Type,
				"daysToWatch":  math.Round(r.DaysToWatch*100) / 100,
				"dateAdded":    r.DateAdded,
				"firstWatched": r.FirstWatched,
			}
		}
		return out
	}

	c.JSON(http.StatusOK, gin.H{
		"avgDaysToWatch":    math.Round(avg*100) / 100,
		"medianDaysToWatch": math.Round(median*100) / 100,
		"distribution":     distribution,
		"slowestItems":     toJSON(slowest),
		"fastestItems":     toJSON(fastest),
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getUnwatchedContent?libraryId=&type=&page=1&pageSize=25
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetUnwatchedContent(c *gin.Context) {
	libraryId := c.Query("libraryId")
	typeFilter := c.Query("type")
	pageSize := parseDays(c.DefaultQuery("pageSize", "25"), 25)
	page := parseDays(c.DefaultQuery("page", "1"), 1)
	offset := (page - 1) * pageSize

	// ── Summary: count total vs unwatched per type ──────────────────────────
	type TypeStat struct {
		Type      string `json:"type"`
		Total     int    `json:"total"`
		Unwatched int    `json:"unwatched"`
	}
	var typeStats []TypeStat

	summaryQuery := `
		SELECT
			i."Type" AS type,
			COUNT(*)::int AS total,
			COUNT(*) FILTER (WHERE a."NowPlayingItemId" IS NULL)::int AS unwatched
		FROM jf_library_items i
		LEFT JOIN (
			SELECT DISTINCT "NowPlayingItemId" FROM jf_playback_activity
		) a ON a."NowPlayingItemId" = i."Id"
		WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder')
	`
	summaryArgs := []interface{}{}
	if libraryId != "" {
		summaryQuery += ` AND i."ParentId" = ?`
		summaryArgs = append(summaryArgs, libraryId)
	}
	summaryQuery += ` GROUP BY i."Type"`

	h.db.Raw(summaryQuery, summaryArgs...).Scan(&typeStats)
	if typeStats == nil {
		typeStats = []TypeStat{}
	}

	totalItems := 0
	unwatchedItems := 0
	for _, ts := range typeStats {
		totalItems += ts.Total
		unwatchedItems += ts.Unwatched
	}

	unwatchedPercent := 0.0
	if totalItems > 0 {
		unwatchedPercent = math.Round(float64(unwatchedItems)/float64(totalItems)*1000) / 10
	}

	// ── Paginated unwatched items ───────────────────────────────────────────
	var total int64
	countQuery := `
		SELECT COUNT(*)
		FROM jf_library_items i
		LEFT JOIN (
			SELECT DISTINCT "NowPlayingItemId" FROM jf_playback_activity
		) a ON a."NowPlayingItemId" = i."Id"
		WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder')
		  AND a."NowPlayingItemId" IS NULL
	`
	countArgs := []interface{}{}
	if libraryId != "" {
		countQuery += ` AND i."ParentId" = ?`
		countArgs = append(countArgs, libraryId)
	}
	if typeFilter != "" {
		countQuery += ` AND i."Type" = ?`
		countArgs = append(countArgs, typeFilter)
	}
	h.db.Raw(countQuery, countArgs...).Scan(&total)

	type ItemRow struct {
		Id          string          `json:"id"`
		Name        string          `json:"name"`
		Type        string          `json:"type"`
		DateAdded   *string         `json:"dateAdded"`
		Genres      json.RawMessage `json:"genres"`
		LibraryName string          `json:"libraryName"`
	}
	var rows []ItemRow

	itemsQuery := `
		SELECT i."Id" AS id, i."Name" AS name, i."Type" AS type,
		       i."DateCreated" AS date_added,
		       i."Genres" AS genres, l."Name" AS library_name
		FROM jf_library_items i
		LEFT JOIN jf_libraries l ON l."Id" = i."ParentId"
		LEFT JOIN (
			SELECT DISTINCT "NowPlayingItemId" FROM jf_playback_activity
		) a ON a."NowPlayingItemId" = i."Id"
		WHERE i.archived = false AND i."Type" NOT IN ('Season', 'Folder')
		  AND a."NowPlayingItemId" IS NULL
	`
	itemsArgs := []interface{}{}
	if libraryId != "" {
		itemsQuery += ` AND i."ParentId" = ?`
		itemsArgs = append(itemsArgs, libraryId)
	}
	if typeFilter != "" {
		itemsQuery += ` AND i."Type" = ?`
		itemsArgs = append(itemsArgs, typeFilter)
	}
	itemsQuery += ` ORDER BY i."DateCreated" DESC NULLS LAST LIMIT ? OFFSET ?`
	itemsArgs = append(itemsArgs, pageSize, offset)

	h.db.Raw(itemsQuery, itemsArgs...).Scan(&rows)
	if rows == nil {
		rows = []ItemRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"summary": gin.H{
			"totalItems":     totalItems,
			"unwatchedItems": unwatchedItems,
			"unwatchedPercent": unwatchedPercent,
			"byType":         typeStats,
		},
		"items": paginate(int(total), pageSize, page, rows),
	})
}

// ---------------------------------------------------------------------------
// Binge-watching detection
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetBingeStats(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	userId := c.Query("userId")

	dateFilter := ""
	if !allTime {
		cutoff := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
		dateFilter = fmt.Sprintf(`AND "ActivityDateInserted" >= '%s'`, cutoff)
	}

	userFilter := ""
	if userId != "" {
		userFilter = fmt.Sprintf(`AND "UserId" = '%s'`, userId)
	}

	bingeSessionsCTE := fmt.Sprintf(`
WITH episode_plays AS (
  SELECT "UserId", "UserName",
         "NowPlayingItemId" AS series_id, "SeriesName",
         "ActivityDateInserted"::timestamptz AS played_at,
         LAG("ActivityDateInserted"::timestamptz) OVER (
           PARTITION BY "UserId", "NowPlayingItemId"
           ORDER BY "ActivityDateInserted"::timestamptz
         ) AS prev_played_at
  FROM jf_playback_activity
  WHERE "EpisodeId" IS NOT NULL AND "EpisodeId" != ''
    %s %s
),
session_markers AS (
  SELECT *,
    SUM(CASE WHEN prev_played_at IS NULL
              OR played_at - prev_played_at > INTERVAL '24 hours'
         THEN 1 ELSE 0 END)
      OVER (PARTITION BY "UserId", series_id ORDER BY played_at) AS session_id
  FROM episode_plays
),
binge_sessions AS (
  SELECT "UserId", MAX("UserName") AS user_name,
         series_id, MAX("SeriesName") AS series_name,
         session_id, COUNT(*) AS episode_count
  FROM session_markers
  GROUP BY "UserId", series_id, session_id
  HAVING COUNT(*) >= 3
)`, dateFilter, userFilter)

	// --- Total binge sessions ---
	var totalBingeSessions int64
	h.db.Raw(bingeSessionsCTE + `
SELECT COALESCE(COUNT(*), 0) FROM binge_sessions`).Scan(&totalBingeSessions)

	// --- Top binged series ---
	type SeriesRow struct {
		SeriesId             string  `json:"seriesId"`
		SeriesName           string  `json:"seriesName"`
		BingeCount           int64   `json:"bingeCount"`
		TotalEpisodesWatched int64   `json:"totalEpisodesWatched"`
		AvgEpisodesPerBinge  float64 `json:"avgEpisodesPerBinge"`
	}
	var topSeries []SeriesRow
	h.db.Raw(bingeSessionsCTE + `
SELECT series_id AS "series_id",
       MAX(series_name) AS "series_name",
       COUNT(*) AS "binge_count",
       SUM(episode_count) AS "total_episodes_watched",
       ROUND(AVG(episode_count)::numeric, 1) AS "avg_episodes_per_binge"
FROM binge_sessions
GROUP BY series_id
ORDER BY "binge_count" DESC
LIMIT 20`).Scan(&topSeries)

	if topSeries == nil {
		topSeries = []SeriesRow{}
	}

	// --- Top binge users ---
	type UserRow struct {
		UserId               string `json:"userId"`
		UserName             string `json:"userName"`
		BingeCount           int64  `json:"bingeCount"`
		TotalEpisodesWatched int64  `json:"totalEpisodesWatched"`
	}
	var topUsers []UserRow
	h.db.Raw(bingeSessionsCTE + `
SELECT "UserId" AS "user_id",
       MAX(user_name) AS "user_name",
       COUNT(*) AS "binge_count",
       SUM(episode_count) AS "total_episodes_watched"
FROM binge_sessions
GROUP BY "UserId"
ORDER BY "binge_count" DESC
LIMIT 20`).Scan(&topUsers)

	if topUsers == nil {
		topUsers = []UserRow{}
	}

	c.JSON(http.StatusOK, gin.H{
		"totalBingeSessions": totalBingeSessions,
		"topBingedSeries":    topSeries,
		"topBingeUsers":      topUsers,
	})
}

// ---------------------------------------------------------------------------
// GET /stats/getCompletionRate?days=30&userId=&libraryId=
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetCompletionRate(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	userId := c.Query("userId")
	libraryId := c.Query("libraryId")

	// Build dynamic WHERE clause
	where := `WHERE COALESCE(a."PlaybackDuration", 0) > 0`
	args := []interface{}{}

	if !allTime {
		where += ` AND a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)`
		args = append(args, days)
	}
	if userId != "" {
		where += ` AND a."UserId" = ?`
		args = append(args, userId)
	}
	if libraryId != "" {
		where += ` AND a."NowPlayingItemId" IN (SELECT "Id" FROM jf_library_items WHERE "ParentId" = ?)`
		args = append(args, libraryId)
	}

	// CTE that computes completion ratio per playback row
	cte := `
	WITH completion AS (
		SELECT
			a."PlaybackDuration",
			CASE
				WHEN mt."Id" IS NOT NULL
					THEN 'Audio'
				WHEN a."EpisodeId" IS NOT NULL AND a."EpisodeId" <> ''
					THEN 'Episode'
				ELSE 'Movie'
			END AS item_type,
			LEAST(
				a."PlaybackDuration"::float
				/ NULLIF(
					COALESCE(
						CASE
							WHEN mt."Id" IS NOT NULL THEN mt."RunTimeTicks"
							WHEN a."EpisodeId" IS NOT NULL AND a."EpisodeId" <> ''
								THEN e."RunTimeTicks"
							ELSE i."RunTimeTicks"
						END,
						0
					) / 10000000.0,
					0
				),
				1.0
			) AS completion_ratio
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i    ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_library_episodes e ON e."Id" = a."EpisodeId"
		LEFT JOIN jf_music_tracks mt    ON mt."Id" = a."EpisodeId"
		` + where + `
			AND COALESCE(
				CASE
					WHEN mt."Id" IS NOT NULL THEN mt."RunTimeTicks"
					WHEN a."EpisodeId" IS NOT NULL AND a."EpisodeId" <> ''
						THEN e."RunTimeTicks"
					ELSE i."RunTimeTicks"
				END,
				0
			) > 0
	)`

	// 1. Overall stats
	type OverallResult struct {
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
		CompletedPlays    int     `json:"completedPlays"`
		AbandonedPlays    int     `json:"abandonedPlays"`
	}
	var overall OverallResult
	h.db.Raw(cte+`
		SELECT
			COALESCE(AVG(completion_ratio), 0)::float AS "avgCompletionRate",
			COUNT(*)::int AS "totalPlays",
			COUNT(*) FILTER (WHERE completion_ratio >= 0.75)::int AS "completedPlays",
			COUNT(*) FILTER (WHERE completion_ratio < 0.75)::int AS "abandonedPlays"
		FROM completion
	`, args...).Scan(&overall)

	// 2. By type
	type ByTypeRow struct {
		Type              string  `json:"type"`
		AvgCompletionRate float64 `json:"avgCompletionRate"`
		TotalPlays        int     `json:"totalPlays"`
	}
	var byType []ByTypeRow
	h.db.Raw(cte+`
		SELECT
			item_type AS "type",
			COALESCE(AVG(completion_ratio), 0)::float AS "avgCompletionRate",
			COUNT(*)::int AS "totalPlays"
		FROM completion
		GROUP BY item_type
		ORDER BY "totalPlays" DESC
	`, args...).Scan(&byType)
	if byType == nil {
		byType = []ByTypeRow{}
	}

	// 3. Distribution buckets
	type DistRow struct {
		Bucket string `json:"bucket"`
		Count  int    `json:"count"`
	}
	var dist []DistRow
	h.db.Raw(cte+`
		SELECT
			b.bucket,
			COALESCE(cnt, 0)::int AS "count"
		FROM (VALUES ('0-25%'), ('25-50%'), ('50-75%'), ('75-100%')) AS b(bucket)
		LEFT JOIN (
			SELECT
				CASE
					WHEN completion_ratio < 0.25 THEN '0-25%'
					WHEN completion_ratio < 0.50 THEN '25-50%'
					WHEN completion_ratio < 0.75 THEN '50-75%'
					ELSE '75-100%'
				END AS bucket,
				COUNT(*)::int AS cnt
			FROM completion
			GROUP BY 1
		) d ON d.bucket = b.bucket
		ORDER BY CASE b.bucket
			WHEN '0-25%' THEN 1
			WHEN '25-50%' THEN 2
			WHEN '50-75%' THEN 3
			WHEN '75-100%' THEN 4
		END
	`, args...).Scan(&dist)
	if dist == nil {
		dist = []DistRow{}
	}

	// Round avgCompletionRate to 2 decimal places
	overall.AvgCompletionRate = math.Round(overall.AvgCompletionRate*100) / 100
	for i := range byType {
		byType[i].AvgCompletionRate = math.Round(byType[i].AvgCompletionRate*100) / 100
	}

	c.JSON(http.StatusOK, gin.H{
		"overall":      overall,
		"byType":       byType,
		"distribution": dist,
	})
}

// ---------------------------------------------------------------------------
// GetViewingDiversity - diversity of viewing habits per user
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetViewingDiversity(c *gin.Context) {
	days, allTime := parseDaysAllTime(c, 30)
	userId := c.Query("userId")

	// Build date filter fragment
	dateFilter := ""
	dateArgs := []interface{}{}
	if !allTime {
		dateFilter = `AND a."ActivityDateInserted" >= NOW() - MAKE_INTERVAL(days => ?)`
		dateArgs = append(dateArgs, days)
	}

	userFilter := ""
	userArgs := []interface{}{}
	if userId != "" {
		userFilter = `AND a."UserId" = ?`
		userArgs = append(userArgs, userId)
	}

	// Genre query
	genreSQL := fmt.Sprintf(`
		SELECT a."UserId",
		       COALESCE(u."Name", a."UserName", a."UserId") AS user_name,
		       genre,
		       COUNT(*)::int AS plays
		FROM jf_playback_activity a
		LEFT JOIN jf_users u ON u."Id" = a."UserId"
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId" AND i.archived = false
		CROSS JOIN LATERAL jsonb_array_elements_text(
		  CASE WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["Unknown"]'::jsonb
		       ELSE i."Genres" END
		) AS genre
		WHERE 1=1 %s %s
		GROUP BY a."UserId", u."Name", a."UserName", genre
	`, dateFilter, userFilter)

	genreArgs := append(append([]interface{}{}, dateArgs...), userArgs...)

	type genreRow struct {
		UserId   string `gorm:"column:UserId"`
		UserName string `gorm:"column:user_name"`
		Genre    string `gorm:"column:genre"`
		Plays    int    `gorm:"column:plays"`
	}
	var genreRows []genreRow
	if err := h.db.Raw(genreSQL, genreArgs...).Scan(&genreRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Library query
	libSQL := fmt.Sprintf(`
		SELECT a."UserId", l."Id" AS library_id, l."Name" AS library_name, COUNT(*)::int AS plays
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_libraries l ON l."Id" = i."ParentId"
		WHERE l."Id" IS NOT NULL %s %s
		GROUP BY a."UserId", l."Id", l."Name"
	`, dateFilter, userFilter)

	libArgs := append(append([]interface{}{}, dateArgs...), userArgs...)

	type libRow struct {
		UserId      string `gorm:"column:UserId"`
		LibraryId   string `gorm:"column:library_id"`
		LibraryName string `gorm:"column:library_name"`
		Plays       int    `gorm:"column:plays"`
	}
	var libRows []libRow
	if err := h.db.Raw(libSQL, libArgs...).Scan(&libRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Unique items query
	itemSQL := fmt.Sprintf(`
		SELECT a."UserId", COUNT(DISTINCT a."NowPlayingItemId")::int AS unique_items, COUNT(*)::int AS total_plays
		FROM jf_playback_activity a
		WHERE 1=1 %s %s
		GROUP BY a."UserId"
	`, dateFilter, userFilter)

	itemArgs := append(append([]interface{}{}, dateArgs...), userArgs...)

	type itemRow struct {
		UserId      string `gorm:"column:UserId"`
		UniqueItems int    `gorm:"column:unique_items"`
		TotalPlays  int    `gorm:"column:total_plays"`
	}
	var itemRows []itemRow
	if err := h.db.Raw(itemSQL, itemArgs...).Scan(&itemRows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Aggregate per user

	type genreInfo struct {
		Genre string
		Plays int
	}
	userGenres := map[string][]genreInfo{}
	userNames := map[string]string{}
	for _, r := range genreRows {
		userGenres[r.UserId] = append(userGenres[r.UserId], genreInfo{Genre: r.Genre, Plays: r.Plays})
		userNames[r.UserId] = r.UserName
	}

	type libInfo struct {
		LibraryId   string
		LibraryName string
		Plays       int
	}
	userLibs := map[string][]libInfo{}
	for _, r := range libRows {
		userLibs[r.UserId] = append(userLibs[r.UserId], libInfo{LibraryId: r.LibraryId, LibraryName: r.LibraryName, Plays: r.Plays})
	}

	userItems := map[string]itemRow{}
	for _, r := range itemRows {
		userItems[r.UserId] = r
	}

	userSet := map[string]bool{}
	for _, r := range genreRows {
		userSet[r.UserId] = true
	}
	for _, r := range itemRows {
		userSet[r.UserId] = true
	}

	// Shannon entropy: normalized to [0,1]
	shannonDiversity := func(genres []genreInfo) float64 {
		total := 0
		for _, g := range genres {
			total += g.Plays
		}
		if total == 0 || len(genres) <= 1 {
			return 0
		}
		entropy := 0.0
		for _, g := range genres {
			p := float64(g.Plays) / float64(total)
			if p > 0 {
				entropy -= p * math.Log(p)
			}
		}
		maxEntropy := math.Log(float64(len(genres)))
		if maxEntropy == 0 {
			return 0
		}
		score := entropy / maxEntropy
		return math.Round(score*100) / 100
	}

	// Build response

	if userId != "" {
		genres := userGenres[userId]
		libs := userLibs[userId]
		items := userItems[userId]
		userName := userNames[userId]

		totalGenrePlays := 0
		for _, g := range genres {
			totalGenrePlays += g.Plays
		}

		genreBreakdown := make([]gin.H, 0, len(genres))
		for _, g := range genres {
			pct := 0.0
			if totalGenrePlays > 0 {
				pct = math.Round(float64(g.Plays)/float64(totalGenrePlays)*10000) / 100
			}
			genreBreakdown = append(genreBreakdown, gin.H{
				"genre":   g.Genre,
				"plays":   g.Plays,
				"percent": pct,
			})
		}
		sort.Slice(genreBreakdown, func(i, j int) bool {
			return genreBreakdown[i]["plays"].(int) > genreBreakdown[j]["plays"].(int)
		})

		totalLibPlays := 0
		for _, l := range libs {
			totalLibPlays += l.Plays
		}

		libraryBreakdown := make([]gin.H, 0, len(libs))
		for _, l := range libs {
			pct := 0.0
			if totalLibPlays > 0 {
				pct = math.Round(float64(l.Plays)/float64(totalLibPlays)*10000) / 100
			}
			libraryBreakdown = append(libraryBreakdown, gin.H{
				"libraryId":   l.LibraryId,
				"libraryName": l.LibraryName,
				"plays":       l.Plays,
				"percent":     pct,
			})
		}
		sort.Slice(libraryBreakdown, func(i, j int) bool {
			return libraryBreakdown[i]["plays"].(int) > libraryBreakdown[j]["plays"].(int)
		})

		c.JSON(http.StatusOK, gin.H{
			"userId":           userId,
			"userName":         userName,
			"diversityScore":   shannonDiversity(genres),
			"uniqueGenres":     len(genres),
			"uniqueLibraries":  len(libs),
			"uniqueItems":      items.UniqueItems,
			"genreBreakdown":   genreBreakdown,
			"libraryBreakdown": libraryBreakdown,
		})
		return
	}

	// All users ranking
	type userSummary struct {
		UserId         string  `json:"userId"`
		UserName       string  `json:"userName"`
		DiversityScore float64 `json:"diversityScore"`
		UniqueGenres   int     `json:"uniqueGenres"`
		UniqueLibs     int     `json:"uniqueLibraries"`
		UniqueItems    int     `json:"uniqueItems"`
		TotalPlays     int     `json:"totalPlays"`
	}

	users := make([]userSummary, 0, len(userSet))
	for uid := range userSet {
		genres := userGenres[uid]
		libs := userLibs[uid]
		items := userItems[uid]
		name := userNames[uid]
		if name == "" {
			name = uid
		}
		users = append(users, userSummary{
			UserId:         uid,
			UserName:       name,
			DiversityScore: shannonDiversity(genres),
			UniqueGenres:   len(genres),
			UniqueLibs:     len(libs),
			UniqueItems:    items.UniqueItems,
			TotalPlays:     items.TotalPlays,
		})
	}
	sort.Slice(users, func(i, j int) bool {
		return users[i].DiversityScore > users[j].DiversityScore
	})

	c.JSON(http.StatusOK, gin.H{"users": users})
}

// ensure math is used
var _ = math.Floor
