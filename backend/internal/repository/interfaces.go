package repository

import (
	"context"

	"github.com/Jellystics/Jellystics/internal/database/models"
)

// ConfigRepository manages app_config.
type ConfigRepository interface {
	Get(ctx context.Context) (*models.AppConfig, error)
	Save(ctx context.Context, cfg *models.AppConfig) error
}

// UserRepository manages jf_users.
type UserRepository interface {
	Upsert(ctx context.Context, users []models.JFUser) error
	List(ctx context.Context) ([]models.JFUser, error)
	GetByID(ctx context.Context, id string) (*models.JFUser, error)
}

// LibraryRepository manages jf_libraries.
type LibraryRepository interface {
	Upsert(ctx context.Context, libs []models.JFLibrary) error
	List(ctx context.Context) ([]models.JFLibrary, error)
	GetByID(ctx context.Context, id string) (*models.JFLibrary, error)
	ArchiveNotIn(ctx context.Context, ids []string) error
}

// ItemRepository manages jf_library_items.
type ItemRepository interface {
	Upsert(ctx context.Context, items []models.JFLibraryItem) error
	ArchiveNotIn(ctx context.Context, parentId string, ids []string) error
	ListByParent(ctx context.Context, parentId string) ([]models.JFLibraryItem, error)
	GetByID(ctx context.Context, id string) (*models.JFLibraryItem, error)
}

// SeasonRepository manages jf_library_seasons.
type SeasonRepository interface {
	Upsert(ctx context.Context, seasons []models.JFLibrarySeason) error
	ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error
	ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibrarySeason, error)
}

// EpisodeRepository manages jf_library_episodes.
type EpisodeRepository interface {
	Upsert(ctx context.Context, episodes []models.JFLibraryEpisode) error
	ArchiveNotIn(ctx context.Context, seriesId string, ids []string) error
	ListBySeries(ctx context.Context, seriesId string) ([]models.JFLibraryEpisode, error)
	ListBySeason(ctx context.Context, seasonId string) ([]models.JFLibraryEpisode, error)
}

// MusicArtistRepository manages jf_music_artists.
type MusicArtistRepository interface {
	Upsert(ctx context.Context, artists []models.JFMusicArtist) error
	ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error
	ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicArtist, error)
}

// MusicTrackRepository manages jf_music_tracks.
type MusicTrackRepository interface {
	Upsert(ctx context.Context, tracks []models.JFMusicTrack) error
	ArchiveNotIn(ctx context.Context, libraryId string, ids []string) error
	ListByLibrary(ctx context.Context, libraryId string) ([]models.JFMusicTrack, error)
	ListByAlbum(ctx context.Context, albumId string) ([]models.JFMusicTrack, error)
	ListByArtist(ctx context.Context, artistId string) ([]models.JFMusicTrack, error)
}

// ItemInfoRepository manages jf_item_info.
type ItemInfoRepository interface {
	Upsert(ctx context.Context, infos []models.JFItemInfo) error
	RemoveOrphaned(ctx context.Context) error
}

// PlaybackRepository manages jf_playback_activity.
type PlaybackRepository interface {
	Upsert(ctx context.Context, activities []models.JFPlaybackActivity) error
	List(ctx context.Context, limit int) ([]models.JFPlaybackActivity, error)
	ListByUser(ctx context.Context, userId string) ([]models.JFPlaybackActivity, error)
}

// WatchdogRepository manages jf_activity_watchdog.
type WatchdogRepository interface {
	Upsert(ctx context.Context, entries []models.JFActivityWatchdog) error
	List(ctx context.Context) ([]models.JFActivityWatchdog, error)
	Delete(ctx context.Context, id string) error
}

// PluginDataRepository manages jf_playback_reporting_plugin_data.
type PluginDataRepository interface {
	Upsert(ctx context.Context, rows []models.JFPluginData) error
	ListUnimported(ctx context.Context) ([]models.JFPluginData, error)
	// GetMaxRowId returns the highest rowid already staged (0 if table is empty).
	GetMaxRowId(ctx context.Context) (int64, error)
	// MergeIntoPlaybackActivity inserts unimported plugin rows into jf_playback_activity.
	MergeIntoPlaybackActivity(ctx context.Context) error
}

// LogRepository manages jf_logging.
type LogRepository interface {
	Insert(ctx context.Context, entry *models.JFLogging) error
	Upsert(ctx context.Context, entry *models.JFLogging) error
	List(ctx context.Context, limit int) ([]models.JFLogging, error)
}

// WebhookRepository manages webhooks.
type WebhookRepository interface {
	List(ctx context.Context) ([]models.Webhook, error)
	GetByID(ctx context.Context, id int) (*models.Webhook, error)
	Create(ctx context.Context, wh *models.Webhook) error
	Update(ctx context.Context, wh *models.Webhook) error
	Delete(ctx context.Context, id int) error
}

// StatsRepository provides read-only analytics queries.
type StatsRepository interface {
	RefreshViews(ctx context.Context) error
	GetGlobalStats(ctx context.Context) (*GlobalStats, error)
	GetMostViewedLibraries(ctx context.Context, limit int) ([]LibraryPlayStat, error)
	GetLibraryStats(ctx context.Context, libraryId string) (*LibraryStats, error)
	GetActivityOverTime(ctx context.Context, days int) ([]DailyActivity, error)
	GetTopUsers(ctx context.Context, limit int) ([]UserStat, error)
	GetMostPlayedItems(ctx context.Context, libraryId string, limit int) ([]ItemPlayStat, error)
	GetMostPlayedArtists(ctx context.Context, libraryId string, limit int) ([]ArtistPlayStat, error)
	GetMostPlayedAlbums(ctx context.Context, libraryId string, artistId string, limit int) ([]AlbumPlayStat, error)
	GetMostPlayedTracks(ctx context.Context, libraryId string, albumId string, limit int) ([]TrackPlayStat, error)
	GetUserHistory(ctx context.Context, userId string, page, pageSize int) ([]ActivityEntry, int64, error)
}

// --- DTOs returned by StatsRepository ---

type GlobalStats struct {
	TotalPlays    int64 `json:"totalPlays"`
	TotalDuration int64 `json:"totalDuration"`
	TotalUsers    int64 `json:"totalUsers"`
	TotalLibraries int64 `json:"totalLibraries"`
}

type LibraryPlayStat struct {
	Id             string `json:"id"`
	Name           string `json:"name"`
	CollectionType string `json:"collectionType"`
	TotalPlays     int64  `json:"totalPlays"`
}

type LibraryStats struct {
	Id           string `json:"id"`
	TotalItems   int    `json:"totalItems"`
	TotalEpisodes int   `json:"totalEpisodes"`
	TotalPlays   int    `json:"totalPlays"`
	LastActivity *string `json:"lastActivity"`
}

type DailyActivity struct {
	Date  string `json:"date"`
	Count int64  `json:"count"`
}

type UserStat struct {
	UserId      string  `json:"userId"`
	UserName    string  `json:"userName"`
	TotalPlays  int64   `json:"totalPlays"`
	TotalHours  float64 `json:"totalHours"`
}

type ItemPlayStat struct {
	Id         string  `json:"id"`
	Name       string  `json:"name"`
	Type       string  `json:"type"`
	TimesPlayed int64  `json:"timesPlayed"`
	TotalDuration int64   `json:"totalDuration"`
}

type ArtistPlayStat struct {
	ArtistId   string `json:"artistId"`
	ArtistName string `json:"artistName"`
	TimesPlayed int64 `json:"timesPlayed"`
}

type AlbumPlayStat struct {
	AlbumId    string `json:"albumId"`
	AlbumName  string `json:"albumName"`
	TimesPlayed int64 `json:"timesPlayed"`
}

type TrackPlayStat struct {
	TrackId     string `json:"trackId"`
	TrackName   string `json:"trackName"`
	AlbumName   *string `json:"albumName"`
	TimesPlayed int64  `json:"timesPlayed"`
}

type ActivityEntry struct {
	Id                   string  `json:"id"`
	UserId               string  `json:"userId"`
	UserName             string  `json:"userName"`
	NowPlayingItemId     *string `json:"nowPlayingItemId"`
	NowPlayingItemName   *string `json:"nowPlayingItemName"`
	SeriesName           *string `json:"seriesName"`
	EpisodeId            *string `json:"episodeId"`
	PlaybackDuration     *int64  `json:"playbackDuration"`
	PlayMethod           *string `json:"playMethod"`
	Client               *string `json:"client"`
	DeviceName           *string `json:"deviceName"`
	ActivityDateInserted *string `json:"activityDateInserted"`
}
