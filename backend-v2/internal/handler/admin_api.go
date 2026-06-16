package handler

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// backupTableDef mirrors the hardcoded list from backend/global/backup_tables.js.
type backupTableDef struct {
	Value string `json:"value"`
	Name  string `json:"name"`
}

var backupTables = []backupTableDef{
	{Value: "jf_libraries", Name: "Libraries"},
	{Value: "jf_library_items", Name: "Library Items"},
	{Value: "jf_library_seasons", Name: "Seasons"},
	{Value: "jf_library_episodes", Name: "Episodes"},
	{Value: "jf_users", Name: "Users"},
	{Value: "jf_playback_activity", Name: "Activity"},
	{Value: "jf_playback_reporting_plugin_data", Name: "Playback Reporting Plugin Data"},
	{Value: "jf_item_info", Name: "Item Info"},
}

// AdminApiHandler handles admin-level library, item, history and backup-table endpoints.
type AdminApiHandler struct {
	repos *repository.Container
	db    *gorm.DB
}

// NewAdminApiHandler constructs an AdminApiHandler.
func NewAdminApiHandler(repos *repository.Container, db *gorm.DB) *AdminApiHandler {
	return &AdminApiHandler{repos: repos, db: db}
}

// ---------------------------------------------------------------------------
// Library / Item helpers
// ---------------------------------------------------------------------------

// GET /api/getLibraries
// Returns all rows from jf_libraries (including archived), same as old backend.
func (h *AdminApiHandler) GetLibraries(c *gin.Context) {
	type row struct {
		Id               string  `json:"Id"`
		Name             *string `json:"Name"`
		ServerId         *string `json:"ServerId"`
		IsFolder         *bool   `json:"IsFolder"`
		Type             *string `json:"Type"`
		CollectionType   *string `json:"CollectionType"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		Archived         bool    `json:"archived"`
	}
	var rows []row
	if err := h.db.Raw(`SELECT "Id","Name","ServerId","IsFolder","Type","CollectionType","ImageTagsPrimary",archived FROM jf_libraries`).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// POST /api/getLibrary
// Body: { "libraryid": "<uuid>" }
func (h *AdminApiHandler) GetLibrary(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	lib, err := h.repos.Library.GetByID(c.Request.Context(), body.LibraryId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "library not found"})
		return
	}
	c.JSON(http.StatusOK, lib)
}

// POST /api/getLibraryItems
// Body: { "libraryid": "<uuid>" }
// Returns ALL items for the library (including archived) matching the old Node behaviour.
func (h *AdminApiHandler) GetLibraryItems(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	var rows []map[string]interface{}
	if err := h.db.Raw(`SELECT * FROM jf_library_items WHERE "ParentId" = ?`, body.LibraryId).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// POST /api/getSeasons
// Body: { "Id": "<seriesId>" }
func (h *AdminApiHandler) GetSeasons(c *gin.Context) {
	var body struct {
		Id string `json:"Id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id is required"})
		return
	}
	var rows []map[string]interface{}
	if err := h.db.Raw(`
		SELECT
			s.*,
			i."PrimaryImageHash",
			(SELECT COUNT(e.*) FROM jf_library_episodes e WHERE e."SeasonId" = s."Id") AS "Episodes",
			(SELECT SUM(ii."Size") FROM jf_library_episodes e
				JOIN jf_item_info ii ON ii."Id" = e."EpisodeId"
				WHERE e."SeasonId" = s."Id") AS "Size"
		FROM jf_library_seasons s
		LEFT JOIN jf_library_items i ON i."Id" = s."SeriesId"
		WHERE s."SeriesId" = ?`, body.Id).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// POST /api/getEpisodes
// Body: { "Id": "<seasonId>" }
func (h *AdminApiHandler) GetEpisodes(c *gin.Context) {
	var body struct {
		Id string `json:"Id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id is required"})
		return
	}
	var rows []map[string]interface{}
	if err := h.db.Raw(`
		SELECT e.*, i."PrimaryImageHash", ii."Size"
		FROM jf_library_episodes e
		LEFT JOIN jf_library_items i ON i."Id" = e."SeriesId"
		JOIN jf_item_info ii ON ii."Id" = e."EpisodeId"
		WHERE e."SeasonId" = ?`, body.Id).
		Scan(&rows).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// POST /api/getItemDetails
// Body: { "Id": "<uuid>" }
// Tries jf_library_items, then jf_library_seasons, then jf_library_episodes.
// Annotates each result with LastActivityDate, times_played, total_play_time.
func (h *AdminApiHandler) GetItemDetails(c *gin.Context) {
	var body struct {
		Id string `json:"Id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Id is required"})
		return
	}
	id := body.Id

	// Helper to query activity stats.
	type activityStats struct {
		LastActivityDate *string `json:"LastActivityDate"`
		TimesPlayed      int64   `json:"times_played"`
		TotalPlayTime    int64   `json:"total_play_time"`
	}
	queryStats := func(whereClause string, args ...interface{}) activityStats {
		var s activityStats
		h.db.Raw(fmt.Sprintf(`
			SELECT
				MAX("ActivityDateInserted") AS "LastActivityDate",
				COUNT("ActivityDateInserted") AS "TimesPlayed",
				COALESCE(SUM("PlaybackDuration"), 0) AS "TotalPlayTime"
			FROM jf_playback_activity
			WHERE %s`, whereClause), args...).Scan(&s)
		return s
	}

	// 1. Try jf_library_items
	var items []map[string]interface{}
	h.db.Raw(`
		SELECT
			im."Name" AS "FileName",
			im."Id", im."Path", im."Name", im."Bitrate", im."MediaStreams", im."Type",
			COALESCE(im."Size", (
				SELECT SUM(im2."Size")
				FROM jf_library_seasons s2
				JOIN jf_library_episodes e2 ON s2."Id" = e2."SeasonId"
				JOIN jf_item_info im2 ON im2."Id" = e2."EpisodeId"
				WHERE s2."SeriesId" = i."Id"
			)) AS "Size",
			i.*,
			(SELECT "Name" FROM jf_libraries l WHERE l."Id" = i."ParentId") AS "LibraryName"
		FROM jf_library_items i
		LEFT JOIN jf_item_info im ON im."Id" = i."Id"
		WHERE i."Id" = ?`, id).Scan(&items)

	if len(items) > 0 {
		stats := queryStats(`"NowPlayingItemId" = ?`, id)
		for i := range items {
			items[i]["LastActivityDate"] = stats.LastActivityDate
			items[i]["times_played"] = stats.TimesPlayed
			items[i]["total_play_time"] = stats.TotalPlayTime
		}
		c.JSON(http.StatusOK, items)
		return
	}

	// 2. Try jf_library_seasons
	var seasons []map[string]interface{}
	h.db.Raw(`
		SELECT
			s."Name",
			(SELECT SUM(im."Size")
				FROM jf_library_episodes e
				JOIN jf_item_info im ON im."Id" = e."EpisodeId"
				WHERE s."Id" = e."SeasonId") AS "Size",
			s.*,
			i."PrimaryImageHash",
			i."ParentId",
			(SELECT "Name" FROM jf_libraries l WHERE l."Id" = i."ParentId") AS "LibraryName"
		FROM jf_library_seasons s
		LEFT JOIN jf_library_items i ON i."Id" = s."SeriesId"
		WHERE s."Id" = ?`, id).Scan(&seasons)

	if len(seasons) > 0 {
		stats := queryStats(`"SeasonId" = ?`, id)
		for i := range seasons {
			seasons[i]["LastActivityDate"] = stats.LastActivityDate
			seasons[i]["times_played"] = stats.TimesPlayed
			seasons[i]["total_play_time"] = stats.TotalPlayTime
		}
		c.JSON(http.StatusOK, seasons)
		return
	}

	// 3. Try jf_library_episodes
	var episodes []map[string]interface{}
	h.db.Raw(`
		SELECT
			im."Name" AS "FileName",
			im.*,
			e.*,
			e.archived,
			i."PrimaryImageHash",
			i."ParentId",
			(SELECT "Name" FROM jf_libraries l WHERE l."Id" = i."ParentId") AS "LibraryName"
		FROM jf_library_episodes e
		JOIN jf_item_info im ON e."EpisodeId" = im."Id"
		LEFT JOIN jf_library_items i ON i."Id" = e."SeriesId"
		WHERE e."EpisodeId" = ?`, id).Scan(&episodes)

	if len(episodes) > 0 {
		stats := queryStats(`"EpisodeId" = ?`, id)
		for i := range episodes {
			episodes[i]["LastActivityDate"] = stats.LastActivityDate
			episodes[i]["times_played"] = stats.TimesPlayed
			episodes[i]["total_play_time"] = stats.TotalPlayTime
		}
		c.JSON(http.StatusOK, episodes)
		return
	}

	c.JSON(http.StatusNotFound, gin.H{"error": "item not found"})
}

// POST /api/getUserDetails
// Body: { "userid": "<uuid>" }
func (h *AdminApiHandler) GetUserDetails(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	user, err := h.repos.User.GetByID(c.Request.Context(), body.UserId)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "user not found"})
		return
	}
	c.JSON(http.StatusOK, user)
}

// GET /api/getRecentlyAdded
// Query params: libraryid (optional), limit (default 50)
func (h *AdminApiHandler) GetRecentlyAdded(c *gin.Context) {
	libraryId := c.Query("libraryid")
	limitStr := c.DefaultQuery("limit", "50")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 50
	}

	type recentRow struct {
		Name             *string `json:"Name"`
		SeriesName       *string `json:"SeriesName"`
		Id               string  `json:"Id"`
		SeriesId         *string `json:"SeriesId"`
		SeasonId         *string `json:"SeasonId"`
		EpisodeId        *string `json:"EpisodeId"`
		SeasonNumber     *int    `json:"SeasonNumber"`
		EpisodeNumber    *int    `json:"EpisodeNumber"`
		PrimaryImageHash *string `json:"PrimaryImageHash"`
		DateCreated      *string `json:"DateCreated"`
		Type             *string `json:"Type"`
		ParentId         *string `json:"ParentId"`
	}

	var items []recentRow
	var episodes []recentRow

	if libraryId != "" {
		h.db.Raw(`
			SELECT
				i."Name", NULL AS "SeriesName", "Id",
				NULL AS "SeriesId", NULL AS "SeasonId", NULL AS "EpisodeId",
				NULL AS "SeasonNumber", NULL AS "EpisodeNumber",
				"PrimaryImageHash", i."DateCreated", "Type", i."ParentId"
			FROM jf_library_items i
			WHERE i.archived = false
				AND i."Type" != 'Series'
				AND i."ParentId" = ?
			ORDER BY "DateCreated" DESC
			LIMIT ?`, libraryId, limit).Scan(&items)

		h.db.Raw(`
			SELECT
				e."Name", e."SeriesName", e."Id",
				e."SeriesId", e."SeasonId", e."EpisodeId",
				e."ParentIndexNumber" AS "SeasonNumber",
				e."IndexNumber" AS "EpisodeNumber",
				e."PrimaryImageHash", e."DateCreated", e."Type", i."ParentId"
			FROM jf_library_episodes e
			JOIN jf_library_items i ON i."Id" = e."SeriesId"
			WHERE e."DateCreated" IS NOT NULL
				AND e.archived = false
				AND i."ParentId" = ?
			ORDER BY e."DateCreated" DESC
			LIMIT ?`, libraryId, limit).Scan(&episodes)
	} else {
		h.db.Raw(`
			SELECT
				i."Name", NULL AS "SeriesName", "Id",
				NULL AS "SeriesId", NULL AS "SeasonId", NULL AS "EpisodeId",
				NULL AS "SeasonNumber", NULL AS "EpisodeNumber",
				"PrimaryImageHash", i."DateCreated", "Type", i."ParentId"
			FROM jf_library_items i
			WHERE i.archived = false
			ORDER BY "DateCreated" DESC
			LIMIT ?`, limit).Scan(&items)

		h.db.Raw(`
			SELECT
				e."Name", e."SeriesName", e."Id",
				e."SeriesId", e."SeasonId", e."EpisodeId",
				e."ParentIndexNumber" AS "SeasonNumber",
				e."IndexNumber" AS "EpisodeNumber",
				e."PrimaryImageHash", e."DateCreated", e."Type", i."ParentId"
			FROM jf_library_episodes e
			JOIN jf_library_items i ON i."Id" = e."SeriesId"
			WHERE e."DateCreated" IS NOT NULL
				AND e.archived = false
			ORDER BY e."DateCreated" DESC
			LIMIT ?`, limit).Scan(&episodes)
	}

	// Combine and return both lists together.
	result := make([]recentRow, 0, len(items)+len(episodes))
	result = append(result, items...)
	result = append(result, episodes...)
	c.JSON(http.StatusOK, result)
}

// ---------------------------------------------------------------------------
// Purge endpoints
// ---------------------------------------------------------------------------

// DELETE /api/item/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeItem(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}
	id := body.Id
	db := h.db

	// Check if there is a library item with this id.
	type idRow struct{ Id string }
	var libItems []idRow
	db.Raw(`SELECT "Id" FROM jf_library_items WHERE "Id" = ?`, id).Scan(&libItems)

	// Get seasons associated with this id (as series or directly).
	type seasonRow struct {
		Id       string
		Archived bool
	}
	var seasons []seasonRow
	db.Raw(`SELECT "Id", archived FROM jf_library_seasons WHERE "SeriesId" = ? OR "Id" = ?`, id, id).Scan(&seasons)

	itemArchived := len(libItems) == 0 || func() bool {
		var r struct{ Archived bool }
		db.Raw(`SELECT archived FROM jf_library_items WHERE "Id" = ?`, id).Scan(&r)
		return r.Archived
	}()

	if len(seasons) > 0 {
		for _, season := range seasons {
			// Delete archived episodes for each season (or all if season/item is archived).
			episodeQuery := `DELETE FROM jf_library_episodes WHERE "SeasonId" = ?`
			if !season.Archived && !itemArchived {
				episodeQuery += ` AND archived = true`
			}
			db.Exec(episodeQuery, season.Id)
			// Delete the season itself if archived.
			if season.Archived || itemArchived {
				db.Exec(`DELETE FROM jf_library_seasons WHERE "Id" = ?`, season.Id)
			}
		}
	} else {
		// Try as episode.
		var archivedEpisodes []struct{ EpisodeId *string }
		db.Raw(`SELECT "EpisodeId" FROM jf_library_episodes WHERE "EpisodeId" = ? AND archived = true`, id).Scan(&archivedEpisodes)
		if len(archivedEpisodes) > 0 {
			db.Exec(`DELETE FROM jf_library_episodes WHERE "EpisodeId" = ? AND archived = true`, id)
		}
		if itemArchived && len(libItems) > 0 {
			db.Exec(`DELETE FROM jf_library_episodes WHERE "SeriesId" = ?`, id)
			db.Exec(`DELETE FROM jf_library_seasons WHERE "SeriesId" = ?`, id)
			db.Exec(`DELETE FROM jf_library_items WHERE "Id" = ?`, id)
		}
	}

	if body.WithActivity {
		// Collect episode IDs from seasons.
		var episodeIds []string
		if len(seasons) > 0 {
			for _, s := range seasons {
				var eids []struct{ Id string }
				db.Raw(`SELECT "Id" FROM jf_library_episodes WHERE "SeasonId" = ?`, s.Id).Scan(&eids)
				for _, e := range eids {
					episodeIds = append(episodeIds, e.Id)
				}
			}
		}
		var seasonIds []string
		for _, s := range seasons {
			seasonIds = append(seasonIds, s.Id)
		}
		h.deletePlaybackForItem(db, id, episodeIds, seasonIds)
	}

	// Refresh materialized views.
	db.Exec("CALL ju_update_library_stats_data()")
	c.JSON(http.StatusOK, gin.H{"message": "item purged successfully"})
}

// DELETE /api/library/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeLibrary(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	h.purgeLibraryItems(body.Id, body.WithActivity, true)
	h.db.Exec(`DELETE FROM jf_libraries WHERE "Id" = ?`, body.Id)
	h.db.Exec("CALL ju_update_library_stats_data()")
	c.JSON(http.StatusOK, gin.H{"message": "library purged successfully"})
}

// DELETE /api/libraryItems/purge
// Body: { "id": "<uuid>", "withActivity": true|false }
func (h *AdminApiHandler) PurgeLibraryItems(c *gin.Context) {
	var body struct {
		Id           string `json:"id"`
		WithActivity bool   `json:"withActivity"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Id == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "id is required"})
		return
	}

	h.purgeLibraryItems(body.Id, body.WithActivity, false)
	h.db.Exec("CALL ju_update_library_stats_data()")
	c.JSON(http.StatusOK, gin.H{"message": "library items purged successfully"})
}

// purgeLibraryItems replicates the Node.js purgeLibraryItems() helper.
// purgeAll=true deletes all items; purgeAll=false deletes only archived ones.
func (h *AdminApiHandler) purgeLibraryItems(libraryId string, withActivity bool, purgeAll bool) {
	db := h.db

	type itemRow struct {
		Id       string
		Archived bool
	}
	var items []itemRow
	db.Raw(`SELECT "Id", archived FROM jf_library_items WHERE "ParentId" = ?`, libraryId).Scan(&items)

	var allSeasonIds []string
	var allEpisodeIds []string

	for _, item := range items {
		type seasonRow struct {
			Id       string
			Archived bool
		}
		var seasons []seasonRow
		seasonQuery := `SELECT "Id", archived FROM jf_library_seasons WHERE "SeriesId" = ?`
		if !item.Archived && !purgeAll {
			seasonQuery += ` AND archived = true`
		}
		db.Raw(seasonQuery, item.Id).Scan(&seasons)

		for _, season := range seasons {
			allSeasonIds = append(allSeasonIds, season.Id)

			var episodeIds []struct{ Id string }
			epQuery := `SELECT "Id" FROM jf_library_episodes WHERE "SeasonId" = ?`
			if !item.Archived && !season.Archived && !purgeAll {
				epQuery += ` AND archived = true`
			}
			db.Raw(epQuery, season.Id).Scan(&episodeIds)
			for _, e := range episodeIds {
				allEpisodeIds = append(allEpisodeIds, e.Id)
			}
		}

		if len(seasons) == 0 {
			// No seasons — try episodes directly under series.
			var episodeIds []struct{ Id string }
			epQuery := `SELECT "Id" FROM jf_library_episodes WHERE "SeriesId" = ?`
			if !item.Archived && !purgeAll {
				epQuery += ` AND archived = true`
			}
			db.Raw(epQuery, item.Id).Scan(&episodeIds)
			for _, e := range episodeIds {
				allEpisodeIds = append(allEpisodeIds, e.Id)
			}
		}
	}

	// Delete episodes.
	if len(allEpisodeIds) > 0 {
		db.Exec(`DELETE FROM jf_library_episodes WHERE "Id" = ANY(?)`, allEpisodeIds)
	}
	// Delete seasons.
	if len(allSeasonIds) > 0 {
		db.Exec(`DELETE FROM jf_library_seasons WHERE "Id" = ANY(?)`, allSeasonIds)
	}
	// Delete library items.
	itemsQuery := `DELETE FROM jf_library_items WHERE "ParentId" = ?`
	if !purgeAll {
		itemsQuery += ` AND archived = true`
	}
	db.Exec(itemsQuery, libraryId)

	if withActivity {
		h.deletePlaybackForItem(db, libraryId, allEpisodeIds, allSeasonIds)
	}
}

// deletePlaybackForItem removes playback_activity rows related to a purged item.
func (h *AdminApiHandler) deletePlaybackForItem(db *gorm.DB, itemId string, episodeIds, seasonIds []string) {
	parts := []string{`"NowPlayingItemId" = ?`}
	args := []interface{}{itemId}

	if len(episodeIds) > 0 {
		parts = append(parts, `"EpisodeId" = ANY(?)`)
		args = append(args, episodeIds)
	}
	if len(seasonIds) > 0 {
		parts = append(parts, `"SeasonId" = ANY(?)`)
		args = append(args, seasonIds)
	}

	query := `DELETE FROM jf_playback_activity WHERE ` + strings.Join(parts, " OR ")
	db.Exec(query, args...)
}

// ---------------------------------------------------------------------------
// History endpoints
// ---------------------------------------------------------------------------

// historyQueryParams holds common pagination/sort params.
type historyQueryParams struct {
	Size    int
	Page    int
	Search  string
	Sort    string
	Desc    bool
	Filters []historyFilter
}

type historyFilter struct {
	Field string  `json:"field"`
	Value *string `json:"value,omitempty"`
	Min   *string `json:"min,omitempty"`
	Max   *string `json:"max,omitempty"`
}

// sortMap mirrors the Node.js unGroupedSortMap (column names from jf_playback_activity_with_metadata).
var historySortMap = map[string]string{
	"UserName":             `a."UserName"`,
	"RemoteEndPoint":       `a."RemoteEndPoint"`,
	"NowPlayingItemName":   `"FullName"`,
	"Client":               `a."Client"`,
	"DeviceName":           `a."DeviceName"`,
	"ActivityDateInserted": `a."ActivityDateInserted"`,
	"PlaybackDuration":     `a."PlaybackDuration"`,
	"PlayMethod":           `a."PlayMethod"`,
}

func parseHistoryParams(c *gin.Context) historyQueryParams {
	p := historyQueryParams{
		Size: 50,
		Page: 1,
		Sort: "ActivityDateInserted",
		Desc: true,
	}
	if v := c.Query("size"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Size = n
		}
	}
	if v := c.Query("page"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			p.Page = n
		}
	}
	if v := c.Query("sort"); v != "" {
		p.Sort = v
	}
	if v := c.Query("desc"); v != "" {
		p.Desc = v != "false" && v != "0"
	}
	p.Search = c.Query("search")

	if f := c.Query("filters"); f != "" {
		var filters []historyFilter
		if err := json.Unmarshal([]byte(f), &filters); err == nil {
			p.Filters = filters
		}
	}
	return p
}

// buildHistorySQL produces the WHERE clause additions and args for history queries.
// baseSQL must already alias the activity view/table as "a".
func buildHistoryWhere(params historyQueryParams, extraWhere string, extraArgs []interface{}) (string, []interface{}) {
	clauses := []string{}
	args := append([]interface{}{}, extraArgs...)

	if extraWhere != "" {
		clauses = append(clauses, extraWhere)
	}

	if params.Search != "" {
		clauses = append(clauses, `LOWER(
			CASE
				WHEN a."SeriesName" IS NULL THEN a."NowPlayingItemName"
				ELSE CONCAT(a."SeriesName", ' : S', a."SeasonNumber", 'E', a."EpisodeNumber", ' - ', a."NowPlayingItemName")
			END
		) LIKE ?`)
		args = append(args, "%"+strings.ToLower(params.Search)+"%")
	}

	for _, f := range params.Filters {
		switch f.Field {
		case "ActivityDateInserted":
			if f.Min != nil {
				clauses = append(clauses, `a."ActivityDateInserted" >= ?`)
				args = append(args, *f.Min)
			}
			if f.Max != nil {
				clauses = append(clauses, `a."ActivityDateInserted" <= ?`)
				args = append(args, *f.Max)
			}
		case "PlaybackDuration":
			if f.Min != nil {
				clauses = append(clauses, `a."PlaybackDuration" >= ?`)
				args = append(args, *f.Min)
			}
			if f.Max != nil {
				clauses = append(clauses, `a."PlaybackDuration" <= ?`)
				args = append(args, *f.Max)
			}
		case "UserName":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(a."UserName") LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		case "Client":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(a."Client") LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		case "DeviceName":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(a."DeviceName") LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		case "RemoteEndPoint":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(a."RemoteEndPoint") LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		case "NowPlayingItemName":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(
					CASE
						WHEN a."SeriesName" IS NULL THEN a."NowPlayingItemName"
						ELSE CONCAT(a."SeriesName", ' : S', a."SeasonNumber", 'E', a."EpisodeNumber", ' - ', a."NowPlayingItemName")
					END
				) LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		case "PlayMethod":
			if f.Value != nil {
				clauses = append(clauses, `LOWER(a."PlayMethod") LIKE ?`)
				args = append(args, "%"+strings.ToLower(*f.Value)+"%")
			}
		}
	}

	where := ""
	if len(clauses) > 0 {
		where = "WHERE " + strings.Join(clauses, " AND ")
	}
	return where, args
}

func historyOrderClause(params historyQueryParams) string {
	col, ok := historySortMap[params.Sort]
	if !ok {
		col = `a."ActivityDateInserted"`
	}
	order := "DESC"
	if !params.Desc {
		order = "ASC"
	}
	return col + " " + order
}

// runHistoryQuery executes a paginated history query against jf_playback_activity_with_metadata.
// extraJoin is an optional JOIN clause (e.g. to filter by libraryId).
// extraWhere is an optional pre-built WHERE fragment (without "WHERE").
func (h *AdminApiHandler) runHistoryQuery(
	params historyQueryParams,
	extraJoin string,
	extraWhere string,
	extraArgs []interface{},
) (map[string]interface{}, error) {
	where, args := buildHistoryWhere(params, extraWhere, extraArgs)
	order := historyOrderClause(params)
	offset := (params.Page - 1) * params.Size

	baseSQL := fmt.Sprintf(`
		FROM jf_playback_activity_with_metadata a
		%s
		%s`, extraJoin, where)

	// Count total.
	var total int64
	if err := h.db.Raw("SELECT COUNT(*) "+baseSQL, args...).Scan(&total).Error; err != nil {
		return nil, err
	}

	pages := 0
	if total > 0 && params.Size > 0 {
		pages = int((total + int64(params.Size) - 1) / int64(params.Size))
	}

	// Fetch rows.
	var rows []map[string]interface{}
	selectSQL := fmt.Sprintf(`
		SELECT
			a.*,
			a."EpisodeNumber",
			a."SeasonNumber",
			a."ParentId",
			CASE
				WHEN a."SeriesName" IS NULL THEN a."NowPlayingItemName"
				ELSE CONCAT(a."SeriesName", ' : S', a."SeasonNumber", 'E', a."EpisodeNumber", ' - ', a."NowPlayingItemName")
			END AS "FullName"
		%s
		ORDER BY %s
		LIMIT ? OFFSET ?`, baseSQL, order)

	queryArgs := append(args, params.Size, offset)
	if err := h.db.Raw(selectSQL, queryArgs...).Scan(&rows).Error; err != nil {
		return nil, err
	}

	resp := map[string]interface{}{
		"current_page": params.Page,
		"pages":        pages,
		"size":         params.Size,
		"sort":         params.Sort,
		"desc":         params.Desc,
		"results":      rows,
	}
	if params.Search != "" {
		resp["search"] = params.Search
	}
	if len(params.Filters) > 0 {
		resp["filters"] = params.Filters
	}
	return resp, nil
}

// GET /api/getHistory
// Query params: size, page, search, sort, desc, filters (JSON)
func (h *AdminApiHandler) GetHistory(c *gin.Context) {
	params := parseHistoryParams(c)
	result, err := h.runHistoryQuery(params, "", "", nil)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/getLibraryHistory
// Body: { "libraryid": "<uuid>" }
// Query params: size, page, search, sort, desc, filters
func (h *AdminApiHandler) GetLibraryHistory(c *gin.Context) {
	var body struct {
		LibraryId string `json:"libraryid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraryid is required"})
		return
	}
	params := parseHistoryParams(c)

	// Join to both jf_library_items and jf_music_tracks to support regular and music libraries.
	extraJoin := `INNER JOIN (
		SELECT "Id", "ParentId" AS "LibraryId" FROM jf_library_items WHERE archived = false
		UNION ALL
		SELECT "Id", "LibraryId" FROM jf_music_tracks WHERE archived = false
	) _lib_items ON _lib_items."Id" = a."NowPlayingItemId" AND _lib_items."LibraryId" = ?`
	result, err := h.runHistoryQuery(params, extraJoin, "", []interface{}{body.LibraryId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/getItemHistory
// Body: { "itemid": "<uuid>" }
// Query params: size, page, search, sort, desc, filters
func (h *AdminApiHandler) GetItemHistory(c *gin.Context) {
	var body struct {
		ItemId string `json:"itemid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ItemId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "itemid is required"})
		return
	}
	params := parseHistoryParams(c)
	// Remove TotalPlays filter (not applicable for ungrouped item history).
	filtered := params.Filters[:0]
	for _, f := range params.Filters {
		if f.Field != "TotalPlays" {
			filtered = append(filtered, f)
		}
	}
	params.Filters = filtered

	extraWhere := `(a."EpisodeId" = ? OR a."SeasonId" = ? OR a."NowPlayingItemId" = ?)`
	result, err := h.runHistoryQuery(params, "", extraWhere, []interface{}{body.ItemId, body.ItemId, body.ItemId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/getUserHistory
// Body: { "userid": "<uuid>" }
// Query params: size, page, search, sort, desc, filters
func (h *AdminApiHandler) GetUserHistory(c *gin.Context) {
	var body struct {
		UserId string `json:"userid"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userid is required"})
		return
	}
	params := parseHistoryParams(c)
	// Remove TotalPlays filter.
	filtered := params.Filters[:0]
	for _, f := range params.Filters {
		if f.Field != "TotalPlays" {
			filtered = append(filtered, f)
		}
	}
	params.Filters = filtered

	extraWhere := `a."UserId" = ?`
	result, err := h.runHistoryQuery(params, "", extraWhere, []interface{}{body.UserId})
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/deletePlaybackActivity
// Body: { "ids": ["id1","id2",...] }
func (h *AdminApiHandler) DeletePlaybackActivity(c *gin.Context) {
	var body struct {
		Ids []string `json:"ids"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || len(body.Ids) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "a non-empty list of ids is required"})
		return
	}
	if err := h.db.Exec(`DELETE FROM jf_playback_activity WHERE "Id" = ANY(?)`, body.Ids).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	// Refresh materialized views.
	h.db.Exec("CALL ju_update_library_stats_data()")
	c.JSON(http.StatusOK, gin.H{"message": fmt.Sprintf("%d records deleted", len(body.Ids))})
}

// POST /api/getActivityTimeLine
// Body: { "userId": "<uuid>", "libraries": ["libId1","libId2"] }
// Groups playback activity by date for the given user and libraries.
func (h *AdminApiHandler) GetActivityTimeLine(c *gin.Context) {
	var body struct {
		UserId    string   `json:"userId"`
		Libraries []string `json:"libraries"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request body"})
		return
	}
	if body.UserId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "userId is required"})
		return
	}
	if len(body.Libraries) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "libraries list is required"})
		return
	}

	type timelineRow struct {
		Date        string  `json:"date"`
		TotalPlays  int64   `json:"total_plays"`
		TotalTime   int64   `json:"total_time"`
		LibraryId   *string `json:"library_id"`
		LibraryName *string `json:"library_name"`
	}

	var rows []timelineRow
	err := h.db.Raw(`
		SELECT
			DATE(a."ActivityDateInserted"::timestamptz) AS date,
			COUNT(a."Id") AS total_plays,
			COALESCE(SUM(a."PlaybackDuration"), 0) AS total_time,
			i."ParentId" AS library_id,
			l."Name" AS library_name
		FROM jf_playback_activity a
		LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
		LEFT JOIN jf_libraries l ON l."Id" = i."ParentId"
		WHERE a."UserId" = ?
			AND i."ParentId" = ANY(?)
		GROUP BY DATE(a."ActivityDateInserted"::timestamptz), i."ParentId", l."Name"
		ORDER BY date ASC
	`, body.UserId, body.Libraries).Scan(&rows).Error

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, rows)
}

// ---------------------------------------------------------------------------
// Backup table endpoints
// ---------------------------------------------------------------------------

// getExcludedTables reads the ExcludedTables list from app_config settings JSON.
func (h *AdminApiHandler) getExcludedTables(c *gin.Context) ([]string, error) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		return []string{}, nil
	}
	raw, ok := settings["ExcludedTables"]
	if !ok {
		return []string{}, nil
	}
	iface, ok := raw.([]interface{})
	if !ok {
		return []string{}, nil
	}
	out := make([]string, 0, len(iface))
	for _, v := range iface {
		if s, ok := v.(string); ok {
			out = append(out, s)
		}
	}
	return out, nil
}

// setExcludedTables persists the ExcludedTables list back to app_config.
func (h *AdminApiHandler) setExcludedTables(c *gin.Context, excluded []string) error {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return err
	}
	var settings map[string]interface{}
	if err := json.Unmarshal(cfg.Settings, &settings); err != nil {
		settings = map[string]interface{}{}
	}
	settings["ExcludedTables"] = excluded
	raw, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	cfg.Settings = raw
	return h.repos.Config.Save(c.Request.Context(), cfg)
}

// GET /api/getBackupTables
func (h *AdminApiHandler) GetBackupTables(c *gin.Context) {
	excluded, err := h.getExcludedTables(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excludedSet := make(map[string]bool, len(excluded))
	for _, t := range excluded {
		excludedSet[t] = true
	}

	type tableEntry struct {
		Value    string `json:"value"`
		Name     string `json:"name"`
		Excluded bool   `json:"Excluded"`
	}
	result := make([]tableEntry, len(backupTables))
	for i, t := range backupTables {
		result[i] = tableEntry{
			Value:    t.Value,
			Name:     t.Name,
			Excluded: excludedSet[t.Value],
		}
	}
	c.JSON(http.StatusOK, result)
}

// POST /api/setExcludedBackupTable
// Body: { "table": "<tableName>" }
// Toggles the table in/out of the excluded list.
func (h *AdminApiHandler) SetExcludedBackupTable(c *gin.Context) {
	var body struct {
		Table string `json:"table"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.Table == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "table is required"})
		return
	}

	// Validate table name against known list.
	valid := false
	for _, t := range backupTables {
		if t.Value == body.Table {
			valid = true
			break
		}
	}
	if !valid {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid table provided"})
		return
	}

	excluded, err := h.getExcludedTables(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Toggle.
	found := false
	newExcluded := make([]string, 0, len(excluded))
	for _, t := range excluded {
		if t == body.Table {
			found = true
		} else {
			newExcluded = append(newExcluded, t)
		}
	}
	if !found {
		newExcluded = append(newExcluded, body.Table)
	}

	if err := h.setExcludedTables(c, newExcluded); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excludedSet := make(map[string]bool, len(newExcluded))
	for _, t := range newExcluded {
		excludedSet[t] = true
	}

	type tableEntry struct {
		Value    string `json:"value"`
		Name     string `json:"name"`
		Excluded bool   `json:"Excluded"`
	}
	result := make([]tableEntry, len(backupTables))
	for i, t := range backupTables {
		result[i] = tableEntry{
			Value:    t.Value,
			Name:     t.Name,
			Excluded: excludedSet[t.Value],
		}
	}
	c.JSON(http.StatusOK, result)
}
