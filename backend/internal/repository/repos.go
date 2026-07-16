package repository

import (
	"context"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// ---- config ----

type configRepo struct{ db *gorm.DB }

func (r *configRepo) Get(ctx context.Context) (*models.AppConfig, error) {
	var cfg models.AppConfig
	if err := r.db.WithContext(ctx).First(&cfg, 1).Error; err != nil {
		return nil, err
	}
	return &cfg, nil
}

func (r *configRepo) Save(ctx context.Context, cfg *models.AppConfig) error {
	return r.db.WithContext(ctx).Save(cfg).Error
}

// ---- users ----

type userRepo struct{ db *gorm.DB }

func (r *userRepo) Upsert(ctx context.Context, users []models.JFUser) error {
	if len(users) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "PrimaryImageTag", "LastLoginDate", "LastActivityDate", "IsAdministrator"}),
		}).
		CreateInBatches(users, 500).Error
}

func (r *userRepo) List(ctx context.Context) ([]models.JFUser, error) {
	var out []models.JFUser
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *userRepo) GetByID(ctx context.Context, id string) (*models.JFUser, error) {
	var u models.JFUser
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&u).Error
	return &u, err
}

// ---- libraries ----

type libraryRepo struct{ db *gorm.DB }

func (r *libraryRepo) Upsert(ctx context.Context, libs []models.JFLibrary) error {
	if len(libs) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "ServerId", "IsFolder", "Type", "CollectionType", "ImageTagsPrimary", "archived"}),
		}).
		CreateInBatches(libs, 500).Error
}

func (r *libraryRepo) List(ctx context.Context) ([]models.JFLibrary, error) {
	var out []models.JFLibrary
	return out, r.db.WithContext(ctx).Where("archived = false").Find(&out).Error
}

func (r *libraryRepo) GetByID(ctx context.Context, id string) (*models.JFLibrary, error) {
	var lib models.JFLibrary
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&lib).Error
	return &lib, err
}

func (r *libraryRepo) ArchiveNotIn(ctx context.Context, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibrary{}).
		Where(`"Id" NOT IN ?`, ids).
		Update("archived", true).Error
}

// ---- items ----

type itemRepo struct{ db *gorm.DB }

var itemUpdateCols = []string{
	"Name", "ServerId", "PremiereDate", "DateCreated", "EndDate",
	"CommunityRating", "RunTimeTicks", "ProductionYear", "IsFolder",
	"Type", "Status", "ImageTagsPrimary", "ImageTagsBanner", "ImageTagsLogo",
	"ImageTagsThumb", "BackdropImageTags", "ParentId", "PrimaryImageHash",
	"archived", "Genres", "AlbumArtist", "ArtistId",
}

func (r *itemRepo) Upsert(ctx context.Context, items []models.JFLibraryItem) error {
	if len(items) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(itemUpdateCols),
		}).
		CreateInBatches(items, 500).Error
}

func (r *itemRepo) ArchiveNotIn(ctx context.Context, parentId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibraryItem{}).
		Where(`"ParentId" = ? AND "Id" NOT IN ?`, parentId, ids).
		Update("archived", true).Error
}

func (r *itemRepo) ListByParent(ctx context.Context, parentId string) ([]models.JFLibraryItem, error) {
	var out []models.JFLibraryItem
	return out, r.db.WithContext(ctx).
		Where(`"ParentId" = ? AND archived = false`, parentId).
		Find(&out).Error
}

func (r *itemRepo) GetByID(ctx context.Context, id string) (*models.JFLibraryItem, error) {
	var item models.JFLibraryItem
	err := r.db.WithContext(ctx).Where(`"Id" = ?`, id).First(&item).Error
	return &item, err
}

// ---- seasons ----

type seasonRepo struct{ db *gorm.DB }

var seasonUpdateCols = []string{
	"Name", "ServerId", "IndexNumber", "Type", "ParentLogoItemId",
	"ParentBackdropItemId", "ParentBackdropImageTags", "SeriesName",
	"SeriesId", "SeriesPrimaryImageTag", "archived",
}

func (r *seasonRepo) Upsert(ctx context.Context, seasons []models.JFLibrarySeason) error {
	if len(seasons) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(seasonUpdateCols),
		}).
		CreateInBatches(seasons, 500).Error
}

func (r *seasonRepo) ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibrarySeason{}).
		Where(`"SeriesId" = ? AND "Id" NOT IN ?`, seriesId, ids).
		Update("archived", true).Error
}

func (r *seasonRepo) ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibrarySeason, error) {
	var out []models.JFLibrarySeason
	return out, r.db.WithContext(ctx).
		Where(`"SeriesId" = ? AND archived = false`, seriesId).
		Find(&out).Error
}

// ---- episodes ----

type episodeRepo struct{ db *gorm.DB }

var episodeUpdateCols = []string{
	"EpisodeId", "Name", "ServerId", "PremiereDate", "DateCreated",
	"OfficialRating", "CommunityRating", "RunTimeTicks", "ProductionYear",
	"IndexNumber", "ParentIndexNumber", "Type", "ParentLogoItemId",
	"ParentBackdropItemId", "ParentBackdropImageTags", "SeriesId", "SeasonId",
	"SeasonName", "SeriesName", "PrimaryImageHash", "archived",
}

func (r *episodeRepo) Upsert(ctx context.Context, eps []models.JFLibraryEpisode) error {
	if len(eps) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(episodeUpdateCols),
		}).
		CreateInBatches(eps, 500).Error
}

func (r *episodeRepo) ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFLibraryEpisode{}).
		Where(`"SeriesId" = ? AND "Id" NOT IN ?`, seriesId, ids).
		Update("archived", true).Error
}

func (r *episodeRepo) ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibraryEpisode, error) {
	var out []models.JFLibraryEpisode
	return out, r.db.WithContext(ctx).
		Where(`"SeriesId" = ? AND archived = false`, seriesId).
		Find(&out).Error
}

func (r *episodeRepo) ListBySeason(ctx context.Context, seasonId string) ([]models.JFLibraryEpisode, error) {
	var out []models.JFLibraryEpisode
	return out, r.db.WithContext(ctx).
		Where(`"SeasonId" = ? AND archived = false`, seasonId).
		Find(&out).Error
}

// ---- music artists ----

type musicArtistRepo struct{ db *gorm.DB }

var artistUpdateCols = []string{"LibraryId", "Name", "Overview", "ImageTagsPrimary", "Genres", "archived"}

func (r *musicArtistRepo) Upsert(ctx context.Context, artists []models.JFMusicArtist) error {
	if len(artists) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(artistUpdateCols),
		}).
		CreateInBatches(artists, 500).Error
}

func (r *musicArtistRepo) ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFMusicArtist{}).
		Where(`"LibraryId" = ? AND "Id" NOT IN ?`, libraryId, ids).
		Update("archived", true).Error
}

func (r *musicArtistRepo) ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicArtist, error) {
	var out []models.JFMusicArtist
	return out, r.db.WithContext(ctx).
		Where(`"LibraryId" = ? AND archived = false`, libraryId).
		Find(&out).Error
}

// ---- music tracks ----

type musicTrackRepo struct{ db *gorm.DB }

var trackUpdateCols = []string{
	"LibraryId", "AlbumId", "ArtistId", "Name", "AlbumName", "AlbumArtist",
	"IndexNumber", "DiscNumber", "RunTimeTicks", "DateCreated", "ProductionYear",
	"ImageTagsPrimary", "Genres", "archived",
}

func (r *musicTrackRepo) Upsert(ctx context.Context, tracks []models.JFMusicTrack) error {
	if len(tracks) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns(trackUpdateCols),
		}).
		CreateInBatches(tracks, 500).Error
}

func (r *musicTrackRepo) ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error {
	return r.db.WithContext(ctx).
		Model(&models.JFMusicTrack{}).
		Where(`"LibraryId" = ? AND "Id" NOT IN ?`, libraryId, ids).
		Update("archived", true).Error
}

func (r *musicTrackRepo) ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"LibraryId" = ? AND archived = false`, libraryId).
		Find(&out).Error
}

func (r *musicTrackRepo) ListByAlbum(ctx context.Context, albumId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"AlbumId" = ? AND archived = false`, albumId).
		Order(`"DiscNumber" ASC, "IndexNumber" ASC`).
		Find(&out).Error
}

func (r *musicTrackRepo) ListByArtist(ctx context.Context, artistId string) ([]models.JFMusicTrack, error) {
	var out []models.JFMusicTrack
	return out, r.db.WithContext(ctx).
		Where(`"ArtistId" = ? AND archived = false`, artistId).
		Find(&out).Error
}

// ---- item info ----

type itemInfoRepo struct{ db *gorm.DB }

func (r *itemInfoRepo) Upsert(ctx context.Context, infos []models.JFItemInfo) error {
	if len(infos) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Path", "Name", "Size", "Bitrate", "MediaStreams", "Type"}),
		}).
		CreateInBatches(infos, 500).Error
}

func (r *itemInfoRepo) RemoveOrphaned(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		DELETE FROM jf_item_info
		WHERE "Id" NOT IN (SELECT "Id" FROM jf_library_items)
		  AND "Id" NOT IN (SELECT "Id" FROM jf_library_episodes)
		  AND "Id" NOT IN (SELECT "Id" FROM jf_music_tracks)
	`).Error
}

// ---- playback ----

type playbackRepo struct{ db *gorm.DB }

func (r *playbackRepo) Upsert(ctx context.Context, acts []models.JFPlaybackActivity) error {
	if len(acts) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"IsPaused", "UserId", "UserName", "Client", "DeviceName", "DeviceId",
				"ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName",
				"EpisodeId", "SeasonId", "SeriesName", "PlaybackDuration", "PlayMethod",
				"ActivityDateInserted", "MediaStreams", "TranscodingInfo", "PlayState",
				"OriginalContainer", "RemoteEndPoint", "ServerId", "source", "imported",
			}),
		}).
		CreateInBatches(acts, 500).Error
}

func (r *playbackRepo) List(ctx context.Context, limit int) ([]models.JFPlaybackActivity, error) {
	var out []models.JFPlaybackActivity
	q := r.db.WithContext(ctx).Order(`"ActivityDateInserted" DESC`)
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

func (r *playbackRepo) ListByUser(ctx context.Context, userId string) ([]models.JFPlaybackActivity, error) {
	var out []models.JFPlaybackActivity
	return out, r.db.WithContext(ctx).
		Where(`"UserId" = ?`, userId).
		Order(`"ActivityDateInserted" DESC`).
		Find(&out).Error
}

// ---- watchdog ----

type watchdogRepo struct{ db *gorm.DB }

func (r *watchdogRepo) Upsert(ctx context.Context, entries []models.JFActivityWatchdog) error {
	if len(entries) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"ActivityId", "IsPaused", "UserId", "UserName", "Client", "DeviceName",
				"DeviceId", "ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName",
				"EpisodeId", "SeasonId", "SeriesName", "PlaybackDuration", "PlayMethod",
				"ActivityDateInserted", "MediaStreams", "TranscodingInfo", "PlayState",
				"OriginalContainer", "RemoteEndPoint", "ServerId",
				"WatchedSeconds", "LastTickAt",
			}),
		}).
		CreateInBatches(entries, 500).Error
}

func (r *watchdogRepo) List(ctx context.Context) ([]models.JFActivityWatchdog, error) {
	var out []models.JFActivityWatchdog
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *watchdogRepo) Delete(ctx context.Context, id string) error {
	return r.db.WithContext(ctx).Delete(&models.JFActivityWatchdog{}, `"Id" = ?`, id).Error
}

// ---- plugin data ----

type pluginDataRepo struct{ db *gorm.DB }

func (r *pluginDataRepo) Upsert(ctx context.Context, rows []models.JFPluginData) error {
	if len(rows) == 0 {
		return nil
	}
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "rowid"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"DateCreated", "UserId", "ItemId", "ItemType", "ItemName",
				"PlaybackMethod", "ClientName", "DeviceName", "PlayDuration", "imported", "source",
			}),
		}).
		CreateInBatches(rows, 500).Error
}

func (r *pluginDataRepo) ListUnimported(ctx context.Context) ([]models.JFPluginData, error) {
	var out []models.JFPluginData
	return out, r.db.WithContext(ctx).Where("imported = false").Find(&out).Error
}

func (r *pluginDataRepo) GetMaxRowId(ctx context.Context) (int64, error) {
	type result struct{ MaxRowId int64 }
	var res result
	err := r.db.WithContext(ctx).Raw(
		`SELECT COALESCE(MAX("rowid"::bigint), 0) AS "MaxRowId" FROM jf_playback_reporting_plugin_data`,
	).Scan(&res).Error
	return res.MaxRowId, err
}

func (r *pluginDataRepo) MergeIntoPlaybackActivity(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`
		INSERT INTO jf_playback_activity (
			"Id", "IsPaused", "UserId", "UserName", "Client", "DeviceName", "DeviceId",
			"ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName",
			"SeasonId", "SeriesName", "EpisodeId",
			"PlaybackDuration", "ActivityDateInserted", "PlayMethod",
			"MediaStreams", "TranscodingInfo", "PlayState", "OriginalContainer", "RemoteEndPoint", "ServerId",
			"imported"
		)
		SELECT
			p."rowid",
			false,
			p."UserId",
			COALESCE(u."Name", p."UserId"),
			p."ClientName",
			p."DeviceName",
			NULL, NULL,
			COALESCE(e."SeriesId", mt."AlbumId", p."ItemId"),
			p."ItemName",
			e."SeasonId",
			COALESCE(s."Name", mt."AlbumName"),
			CASE WHEN e."Id" IS NOT NULL OR mt."Id" IS NOT NULL THEN p."ItemId" ELSE NULL END,
			p."PlayDuration",
			p."DateCreated",
			p."PlaybackMethod",
			NULL, NULL, NULL, NULL, NULL, NULL,
			true
		FROM jf_playback_reporting_plugin_data p
		LEFT JOIN jf_users u ON u."Id" = p."UserId"
		LEFT JOIN jf_library_episodes e ON e."Id" = p."ItemId" OR e."EpisodeId" = p."ItemId"
		LEFT JOIN jf_library_items s ON s."Id" = e."SeriesId"
		LEFT JOIN jf_music_tracks mt ON mt."Id" = p."ItemId"
		ON CONFLICT ("Id") DO NOTHING
	`).Error
}

// ---- logging ----

type logRepo struct{ db *gorm.DB }

func (r *logRepo) Insert(ctx context.Context, entry *models.JFLogging) error {
	return r.db.WithContext(ctx).Create(entry).Error
}

func (r *logRepo) Upsert(ctx context.Context, entry *models.JFLogging) error {
	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "Id"}},
			DoUpdates: clause.AssignmentColumns([]string{"Name", "Type", "ExecutionType", "Duration", "TimeRun", "Log", "Result"}),
		}).
		Create(entry).Error
}

func (r *logRepo) List(ctx context.Context, limit int) ([]models.JFLogging, error) {
	var out []models.JFLogging
	q := r.db.WithContext(ctx).Order(`"TimeRun" DESC`)
	if limit > 0 {
		q = q.Limit(limit)
	}
	return out, q.Find(&out).Error
}

// ---- webhooks ----

type webhookRepo struct{ db *gorm.DB }

func (r *webhookRepo) List(ctx context.Context) ([]models.Webhook, error) {
	var out []models.Webhook
	return out, r.db.WithContext(ctx).Find(&out).Error
}

func (r *webhookRepo) GetByID(ctx context.Context, id int) (*models.Webhook, error) {
	var wh models.Webhook
	err := r.db.WithContext(ctx).First(&wh, id).Error
	return &wh, err
}

func (r *webhookRepo) Create(ctx context.Context, wh *models.Webhook) error {
	return r.db.WithContext(ctx).Create(wh).Error
}

func (r *webhookRepo) Update(ctx context.Context, wh *models.Webhook) error {
	return r.db.WithContext(ctx).Save(wh).Error
}

func (r *webhookRepo) Delete(ctx context.Context, id int) error {
	return r.db.WithContext(ctx).Delete(&models.Webhook{}, id).Error
}

// ---- stats ----

type statsRepo struct{ db *gorm.DB }

func (r *statsRepo) RefreshViews(ctx context.Context) error {
	return r.db.WithContext(ctx).Exec(`CALL ju_update_library_stats_data()`).Error
}

func (r *statsRepo) GetGlobalStats(ctx context.Context) (*GlobalStats, error) {
	var stats GlobalStats
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			COUNT(*)                           AS total_plays,
			COALESCE(SUM("PlaybackDuration"),0) AS total_duration,
			(SELECT COUNT(*) FROM jf_users)    AS total_users,
			(SELECT COUNT(*) FROM jf_libraries WHERE archived = false) AS total_libraries
		FROM jf_playback_activity
	`).Scan(&stats).Error
	return &stats, err
}

func (r *statsRepo) GetMostViewedLibraries(ctx context.Context, limit int) ([]LibraryPlayStat, error) {
	var out []LibraryPlayStat
	err := r.db.WithContext(ctx).Raw(`
		WITH library_activity AS (
			SELECT DISTINCT l."Id" AS library_id, a."Id" AS activity_id
			FROM jf_libraries l
			LEFT JOIN jf_library_items i  ON i."ParentId" = l."Id" AND i.archived = false
			LEFT JOIN jf_library_episodes ep ON ep."SeriesId" = i."Id"
			LEFT JOIN jf_music_tracks mt ON mt."LibraryId" = l."Id" AND mt.archived = false
			LEFT JOIN jf_playback_activity a ON
				a."NowPlayingItemId" = i."Id" OR
				a."EpisodeId" = ep."Id" OR
				a."NowPlayingItemId" = mt."Id"
			WHERE l.archived = false
		)
		SELECT la.library_id AS id, l."Name" AS name, l."CollectionType" AS collection_type,
		       COUNT(la.activity_id)::bigint AS total_plays
		FROM library_activity la
		JOIN jf_libraries l ON l."Id" = la.library_id
		GROUP BY la.library_id, l."Name", l."CollectionType"
		ORDER BY total_plays DESC
		LIMIT ?
	`, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetLibraryStats(ctx context.Context, libraryId string) (*LibraryStats, error) {
	var stats LibraryStats
	err := r.db.WithContext(ctx).Raw(`
		WITH library_plays AS (
			SELECT DISTINCT a."Id" AS activity_id, a."ActivityDateInserted"
			FROM jf_libraries l
			LEFT JOIN jf_library_items i  ON i."ParentId" = l."Id" AND i.archived = false
			LEFT JOIN jf_library_episodes ep ON ep."SeriesId" = i."Id"
			LEFT JOIN jf_music_tracks mt ON mt."LibraryId" = l."Id" AND mt.archived = false
			LEFT JOIN jf_playback_activity a ON
				a."NowPlayingItemId" = i."Id" OR
				a."EpisodeId" = ep."Id" OR
				a."NowPlayingItemId" = mt."Id"
			WHERE l."Id" = ?
		)
		SELECT
			l."Id" AS id,
			(COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season','Folder'))
			 + COUNT(DISTINCT mt."Id"))::int AS total_items,
			COUNT(DISTINCT ep."Id")::int AS total_episodes,
			(SELECT COUNT(activity_id) FROM library_plays)::int AS total_plays,
			(SELECT MAX("ActivityDateInserted") FROM library_plays) AS last_activity
		FROM jf_libraries l
		LEFT JOIN jf_library_items    i  ON i."ParentId"  = l."Id" AND i.archived  = false
		LEFT JOIN jf_library_episodes ep ON ep."SeriesId" = i."Id"
		LEFT JOIN jf_music_tracks     mt ON mt."LibraryId" = l."Id" AND mt.archived = false
		WHERE l."Id" = ?
		GROUP BY l."Id"
	`, libraryId, libraryId).Scan(&stats).Error
	return &stats, err
}

func (r *statsRepo) GetActivityOverTime(ctx context.Context, days int) ([]DailyActivity, error) {
	var out []DailyActivity
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			DATE("ActivityDateInserted"::timestamptz) AS date,
			COUNT(*) AS count
		FROM jf_playback_activity
		WHERE "ActivityDateInserted"::timestamptz >= NOW() - ? * INTERVAL '1 day'
		GROUP BY date
		ORDER BY date ASC
	`, days).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetTopUsers(ctx context.Context, limit int) ([]UserStat, error) {
	var out []UserStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			a."UserId"  AS user_id,
			COALESCE(u."Name", a."UserName", a."UserId") AS user_name,
			COUNT(*)    AS total_plays,
			COALESCE(SUM(a."PlaybackDuration"), 0)::float8 / 3600.0 AS total_hours
		FROM jf_playback_activity a
		LEFT JOIN jf_users u ON u."Id" = a."UserId"
		GROUP BY a."UserId", u."Name", a."UserName"
		ORDER BY total_plays DESC
		LIMIT ?
	`, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetMostPlayedItems(ctx context.Context, libraryId string, limit int) ([]ItemPlayStat, error) {
	var out []ItemPlayStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			i."Id" AS id, i."Name" AS name, i."Type" AS type,
			COUNT(a."Id")::bigint AS times_played,
			COALESCE(SUM(a."PlaybackDuration"),0) AS total_duration
		FROM jf_library_items i
		JOIN jf_playback_activity a ON a."NowPlayingItemId" = i."Id"
		WHERE i."ParentId" = ? AND i.archived = false
		GROUP BY i."Id", i."Name", i."Type"
		ORDER BY times_played DESC
		LIMIT ?
	`, libraryId, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetMostPlayedArtists(ctx context.Context, libraryId string, limit int) ([]ArtistPlayStat, error) {
	var out []ArtistPlayStat
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			ar."Id" AS artist_id, ar."Name" AS artist_name,
			COUNT(a."Id")::bigint AS times_played
		FROM jf_music_artists ar
		JOIN jf_music_tracks t ON t."ArtistId" = ar."Id"
		JOIN jf_playback_activity a ON a."EpisodeId" = t."Id"
		WHERE ar."LibraryId" = ? AND ar.archived = false
		GROUP BY ar."Id", ar."Name"
		ORDER BY times_played DESC
		LIMIT ?
	`, libraryId, limit).Scan(&out).Error
	return out, err
}

func (r *statsRepo) GetMostPlayedAlbums(ctx context.Context, libraryId, artistId string, limit int) ([]AlbumPlayStat, error) {
	var out []AlbumPlayStat
	q := r.db.WithContext(ctx)
	// Album plays are recorded as playback rows whose NowPlayingItemId is the
	// album id (each track play maps parent=AlbumId, child=trackId). Count them
	// directly on the album to avoid multiplying by the track count.
	sql := `
		SELECT
			i."Id" AS album_id, i."Name" AS album_name,
			COUNT(a."Id")::bigint AS times_played
		FROM jf_library_items i
		JOIN jf_playback_activity a ON a."NowPlayingItemId" = i."Id"
		WHERE i."Type" = 'MusicAlbum' AND i.archived = false
	`
	args := []any{}
	if libraryId != "" {
		sql += ` AND i."ParentId" = ?`
		args = append(args, libraryId)
	}
	if artistId != "" {
		sql += ` AND EXISTS (
			SELECT 1 FROM jf_music_tracks t
			WHERE t."AlbumId" = i."Id" AND t."ArtistId" = ?
		)`
		args = append(args, artistId)
	}
	sql += ` GROUP BY i."Id", i."Name" ORDER BY times_played DESC LIMIT ?`
	args = append(args, limit)
	return out, q.Raw(sql, args...).Scan(&out).Error
}

func (r *statsRepo) GetMostPlayedTracks(ctx context.Context, libraryId, albumId string, limit int) ([]TrackPlayStat, error) {
	sql := `
		SELECT
			t."Id" AS track_id, t."Name" AS track_name, t."AlbumName" AS album_name,
			COUNT(a."Id")::bigint AS times_played
		FROM jf_music_tracks t
		JOIN jf_playback_activity a ON a."EpisodeId" = t."Id"
		WHERE t.archived = false
	`
	args := []any{}
	if libraryId != "" {
		sql += ` AND t."LibraryId" = ?`
		args = append(args, libraryId)
	}
	if albumId != "" {
		sql += ` AND t."AlbumId" = ?`
		args = append(args, albumId)
	}
	sql += ` GROUP BY t."Id", t."Name", t."AlbumName" ORDER BY times_played DESC LIMIT ?`
	args = append(args, limit)
	var result []TrackPlayStat
	return result, r.db.WithContext(ctx).Raw(sql, args...).Scan(&result).Error
}

func (r *statsRepo) GetUserHistory(ctx context.Context, userId string, page, pageSize int) ([]ActivityEntry, int64, error) {
	var total int64
	r.db.WithContext(ctx).Model(&models.JFPlaybackActivity{}).
		Where(`"UserId" = ?`, userId).Count(&total)

	var rows []ActivityEntry
	err := r.db.WithContext(ctx).Raw(`
		SELECT
			a."Id" AS id,
			a."UserId" AS user_id,
			COALESCE(u."Name", a."UserName", a."UserId") AS user_name,
			a."NowPlayingItemId" AS now_playing_item_id,
			a."NowPlayingItemName" AS now_playing_item_name,
			a."SeriesName" AS series_name,
			a."EpisodeId" AS episode_id,
			a."PlaybackDuration" AS playback_duration,
			a."PlayMethod" AS play_method,
			a."Client" AS client,
			a."DeviceName" AS device_name,
			a."ActivityDateInserted" AS activity_date_inserted
		FROM jf_playback_activity a
		LEFT JOIN jf_users u ON u."Id" = a."UserId"
		WHERE a."UserId" = ?
		ORDER BY a."ActivityDateInserted" DESC
		LIMIT ? OFFSET ?
	`, userId, pageSize, (page-1)*pageSize).Scan(&rows).Error
	return rows, total, err
}
