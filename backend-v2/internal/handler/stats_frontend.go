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
		h.db.Raw(`SELECT "Id", "ParentId", "Type", "Genres" FROM jf_library_items WHERE "Id" = ANY(?)`, ids).Scan(&libRows)
	}

	now := time.Now()
	var result []liveSession
	for _, s := range active {
		item := s.NowPlayingItem
		lookupId := item.Id
		if item.SeriesId != nil && *item.SeriesId != "" {
			lookupId = *item.SeriesId
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
		}

		itemName := item.Name
		if item.SeriesName != nil && *item.SeriesName != "" {
			itemName = *item.SeriesName
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
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type Item struct {
		Id        string `json:"Id"`
		Name      string `json:"Name"`
		PlayCount int    `json:"PlayCount"`
		Type      string `json:"Type"`
	}
	var dbItems []Item

	baseSQL := `
		SELECT
		  a."NowPlayingItemId" AS "Id",
		  COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemName") AS "Name",
		  COUNT(*)::int AS "PlayCount",
		  CASE
		    WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Series'
		    ELSE COALESCE(i."Type", 'Unknown')
		  END AS "Type"
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
	`
	suffix := ` GROUP BY a."NowPlayingItemId", 2, 4 ORDER BY "PlayCount" DESC LIMIT ?`

	switch itemType {
	case "Series":
		h.db.Raw(baseSQL+` AND a."SeriesName" IS NOT NULL AND a."SeriesName" <> ''`+suffix, days, limit).Scan(&dbItems)
	case "Audio":
		h.db.Raw(baseSQL+` AND i."Type" = 'Audio'`+suffix, days, limit).Scan(&dbItems)
	case "Movie":
		h.db.Raw(baseSQL+` AND COALESCE(i."Type", '') IN ('Movie', 'Video')`+suffix, days, limit).Scan(&dbItems)
	case "all":
		h.db.Raw(baseSQL+suffix, days, limit).Scan(&dbItems)
	default:
		h.db.Raw(baseSQL+` AND i."Type" = ?`+suffix, days, itemType, limit).Scan(&dbItems)
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
	limit := parseDays(c.DefaultQuery("limit", "5"), 5)
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type UserRow struct {
		UserId         string `json:"UserId"`
		UserName       string `json:"UserName"`
		TotalPlays     int    `json:"TotalPlays"`
		TotalWatchTime int    `json:"TotalWatchTime"`
	}
	var dbRows []UserRow
	h.db.Raw(`
		SELECT
		  "UserId",
		  MAX("UserName") AS "UserName",
		  COUNT(*)::int AS "TotalPlays",
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime"
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		GROUP BY "UserId"
		ORDER BY "TotalPlays" DESC
		LIMIT ?
	`, days, limit).Scan(&dbRows)

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
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1
	userId := c.Query("userId")

	type Row struct {
		Date     string `json:"date"`
		Plays    int    `json:"plays"`
		Duration int    `json:"duration"`
	}
	var rows []Row

	if userId != "" {
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
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type Row struct {
		Hour     int `json:"hour"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  EXTRACT(HOUR FROM "ActivityDateInserted"::timestamptz)::int AS hour,
		  COUNT(*)::int AS plays,
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		GROUP BY hour
		ORDER BY hour
	`, days).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

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

	sort.Slice(rows, func(i, j int) bool { return rows[i].Hour < rows[j].Hour })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getPopularDayOfWeek?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetPopularDayOfWeek(c *gin.Context) {
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type Row struct {
		Day      int `json:"day"`
		Plays    int `json:"plays"`
		Duration int `json:"duration"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  EXTRACT(DOW FROM "ActivityDateInserted"::timestamptz)::int AS day,
		  COUNT(*)::int AS plays,
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		GROUP BY day
		ORDER BY day
	`, days).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

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

	sort.Slice(rows, func(i, j int) bool { return rows[i].Day < rows[j].Day })
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// GET /stats/getMostUsedPlaybackMethod?days=30
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetMostUsedPlaybackMethod(c *gin.Context) {
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type Row struct {
		Method   string `json:"method"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  COALESCE(NULLIF("PlayMethod", ''), 'Unknown') AS method,
		  COUNT(*)::int AS count,
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		GROUP BY method
		ORDER BY count DESC
	`, days).Scan(&rows)

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
	days := parseDays(c.DefaultQuery("days", "30"), 30) - 1

	type Row struct {
		Client   string `json:"client"`
		Count    int    `json:"count"`
		Duration int    `json:"duration"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  "Client" AS client,
		  COUNT(*)::int AS count,
		  FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS duration
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => ?)
		  AND "Client" IS NOT NULL AND "Client" <> ''
		GROUP BY "Client"
		ORDER BY count DESC
		LIMIT 10
	`, days).Scan(&rows)

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
		UserId         string  `json:"UserId"`
		UserName       string  `json:"UserName"`
		TotalPlays     int     `json:"TotalPlays"`
		TotalWatchTime int     `json:"TotalWatchTime"`
		LastSeen       *string `json:"LastSeen"`
		FavoriteGenre  *string `json:"FavoriteGenre"`
	}

	live := h.getLiveActivity(c.Request.Context())

	if userId != "" {
		var row UserStat
		h.db.Raw(`
			SELECT
			  u."Id" AS "UserId",
			  u."Name" AS "UserName",
			  COUNT(a."Id")::int AS "TotalPlays",
			  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
			  COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
			  NULL::text AS "FavoriteGenre"
			FROM jf_users u
			LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
			WHERE u."Id" = ?
			GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
		`, userId).Scan(&row)

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
	h.db.Raw(`
		SELECT
		  u."Id" AS "UserId",
		  u."Name" AS "UserName",
		  COUNT(a."Id")::int AS "TotalPlays",
		  FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
		  COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
		  NULL::text AS "FavoriteGenre"
		FROM jf_users u
		LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
		GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
		ORDER BY "TotalPlays" DESC
	`).Scan(&rows)

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
	SeriesName           *string `json:"SeriesName"`
	SeasonId             *string `json:"SeasonId"`
	EpisodeId            *string `json:"EpisodeId"`
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

func (h *StatsFrontendHandler) GetAllUserActivity(c *gin.Context) {
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
		ORDER BY "ActivityDateInserted"::timestamptz DESC
		LIMIT 500
	`).Scan(&rows)

	if rows == nil {
		rows = []activityRow{}
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
		dur := int64(ls.PlaybackDuration) * 600_000_000
		isActive := true
		isPaused := false
		liveRows = append(liveRows, activityRow{
			UserId:               &userId,
			UserName:             &userName,
			ItemId:               &itemId,
			NowPlayingItemName:   &itemName,
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
		LIMIT 200
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
		Id             string `json:"Id"`
		Name           string `json:"Name"`
		CollectionType string `json:"CollectionType"`
		ItemCount      int    `json:"ItemCount"`
		EpisodeCount   int    `json:"EpisodeCount"`
	}
	var rows []Row
	h.db.Raw(`
		SELECT
		  l."Id",
		  l."Name",
		  COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
		  COUNT(i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount",
		  COUNT(i."Id") FILTER (WHERE i."Type" = 'Episode')::int AS "EpisodeCount"
		FROM jf_libraries l
		LEFT JOIN jf_library_items i
		  ON i."ParentId" = l."Id" AND i.archived = false
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
		LIMIT 500
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
	}
	var item ItemInfo
	h.db.Raw(`
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
		  info."Bitrate"
		FROM jf_library_items i
		LEFT JOIN jf_item_info info ON info."Id" = i."Id"
		WHERE i."Id" = ?
		LIMIT 1
	`, itemId).Scan(&item)

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
	userMap := map[string]int{}
	var lastWatched *string
	isActive := false
	for _, h2 := range history {
		totalWatchTime += h2.PlaybackDuration
		if h2.UserId != nil {
			userMap[*h2.UserId]++
		}
		if lastWatched == nil && h2.ActivityDateInserted != nil {
			lastWatched = h2.ActivityDateInserted
		}
		if h2.IsActive {
			isActive = true
		}
	}

	type UserAgg struct {
		UserId   string `json:"UserId"`
		UserName string `json:"UserName"`
		Plays    int    `json:"Plays"`
	}
	var users []UserAgg
	for uid, plays := range userMap {
		uname := uid
		for _, he := range history {
			if he.UserId != nil && *he.UserId == uid && he.UserName != nil {
				uname = *he.UserName
				break
			}
		}
		users = append(users, UserAgg{UserId: uid, UserName: uname, Plays: plays})
	}
	sort.Slice(users, func(i, j int) bool { return users[i].Plays > users[j].Plays })

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
				  ON a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id"
				CROSS JOIN LATERAL jsonb_array_elements_text(
				  CASE
				    WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
				    ELSE i."Genres"
				  END
				) AS genre
				WHERE a."UserId" = ?
				  AND i."ParentId" = ?
				GROUP BY genre
				ORDER BY "PlayCount" DESC, genre ASC
				LIMIT 100
			`, userId, libraryId).Scan(&rows)
		} else {
			h.db.Raw(`
				SELECT
				  genre AS "Genre",
				  COUNT(DISTINCT COALESCE(a."EpisodeId", a."NowPlayingItemId"))::int AS "Count",
				  COUNT(*)::int AS "PlayCount"
				FROM jf_playback_activity a
				LEFT JOIN jf_library_items i
				  ON a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id"
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
		  SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  GROUP BY "NowPlayingItemId"
		) pc ON pc."NowPlayingItemId" = t."Id"
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
		  SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  GROUP BY "NowPlayingItemId"
		) pc ON pc."NowPlayingItemId" = t."Id"
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
		  ar."Id",
		  ar."Name",
		  ar."ImageTagsPrimary",
		  COUNT(DISTINCT album."Id")::int AS "AlbumCount",
		  COUNT(t."Id")::int AS "TrackCount",
		  COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
		FROM jf_music_artists ar
		LEFT JOIN jf_library_items album ON album."ArtistId" = ar."Id" AND album.archived = false
		LEFT JOIN jf_music_tracks t ON t."AlbumId" = album."Id" AND t.archived = false
		LEFT JOIN (
		  SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  GROUP BY "NowPlayingItemId"
		) pc ON pc."NowPlayingItemId" = t."Id"
		WHERE ar."LibraryId" = ?
		  AND ar.archived = false
		GROUP BY ar."Id", ar."Name", ar."ImageTagsPrimary"
		ORDER BY ar."Name"
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
		  SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  GROUP BY "NowPlayingItemId"
		) pc ON pc."NowPlayingItemId" = t."Id"
		WHERE album."ParentId" = ?
		  AND album.archived = false
		  AND album."Type" = 'MusicAlbum'
		  AND album."ArtistId" = ?
		GROUP BY album."Id", album."Name", album."AlbumArtist", album."ArtistId", album."ProductionYear", album."ImageTagsPrimary"
		ORDER BY album."Name"
	`, libraryId, artistId).Scan(&rows)

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
		  SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
		  FROM jf_playback_activity
		  GROUP BY "NowPlayingItemId"
		) pc ON pc."NowPlayingItemId" = t."Id"
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
	type Row struct {
		Id           *string `json:"Id"`
		UserId       *string `json:"UserId"`
		UserName     *string `json:"UserName"`
		ItemId       *string `json:"ItemId"`
		ItemName     *string `json:"ItemName"`
		StartTime    *string `json:"StartTime"`
		EndTime      *string `json:"EndTime"`
		Duration     int     `json:"Duration"`
		Client       *string `json:"Client"`
		PlayMethod   *string `json:"PlayMethod"`
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
		ORDER BY "ActivityDateInserted"::timestamptz DESC
		LIMIT 500
	`).Scan(&rows)

	if rows == nil {
		rows = []Row{}
	}

	live := h.getLiveActivity(c.Request.Context())
	now := time.Now()
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
		  a."Id", a."UserId", a."UserName", a."NowPlayingItemName", a."SeriesName",
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
// GET /stats/getGenreUserStats?userid=&size=50&page=1
// ---------------------------------------------------------------------------

func (h *StatsFrontendHandler) GetGenreUserStats(c *gin.Context) {
	userId := c.Query("userid")
	if userId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
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
		  WHERE a."UserId" = ?
		  GROUP BY COALESCE(g.genre, 'No Genre')
		) sub
	`, userId).Scan(&total)

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
		WHERE a."UserId" = ?
		GROUP BY COALESCE(g.genre, 'No Genre')
		ORDER BY genre ASC
		LIMIT ? OFFSET ?
	`, userId, size, offset).Scan(&rows)

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
		WHERE a."ActivityDateInserted" BETWEEN CURRENT_DATE - MAKE_INTERVAL(days => ?) AND NOW()
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
	endTime := now
	hours := 24
	if body.Hours != nil && *body.Hours > 0 {
		hours = *body.Hours
	}
	startTime := now.Add(-time.Duration(hours) * time.Hour)

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

	type RawRow struct {
		ActivityDateInserted string  `gorm:"column:ActivityDateInserted"`
		PlaybackDuration     *int64  `gorm:"column:PlaybackDuration"`
		PlayMethod           *string `gorm:"column:PlayMethod"`
	}
	var rawRows []RawRow
	h.db.Raw(`
		SELECT a."ActivityDateInserted", a."PlaybackDuration", a."PlayMethod"
		FROM jf_playback_activity a
		JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		WHERE i."ParentId" = ?
		AND a."ActivityDateInserted" BETWEEN ? AND ?
		ORDER BY a."ActivityDateInserted" DESC
	`, body.LibraryId, startTime.Format(time.RFC3339), endTime.Format(time.RFC3339)).Scan(&rawRows)

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

// ensure math is used
var _ = math.Floor
