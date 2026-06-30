package models

import (
	"time"

	"gorm.io/datatypes"
)

// AppConfig holds the single-row application configuration.
type AppConfig struct {
	ID            int            `gorm:"column:ID;primaryKey"`
	JFHost        *string        `gorm:"column:JF_HOST"`
	JFApiKey      *string        `gorm:"column:JF_API_KEY"`
	AppUser       *string        `gorm:"column:APP_USER"`
	AppPassword   *string        `gorm:"column:APP_PASSWORD"`
	RequireLogin  bool           `gorm:"column:REQUIRE_LOGIN;not null;default:false"`
	Settings      datatypes.JSON `gorm:"column:settings;not null;default:'{}'"`
	ApiKeys       datatypes.JSON `gorm:"column:api_keys;not null;default:'[]'"`
	AppUrl        string         `gorm:"column:app_url;not null;default:''"`
}

func (AppConfig) TableName() string { return "app_config" }

// JFUser mirrors the jf_users table.
type JFUser struct {
	Id               string  `gorm:"column:Id;primaryKey"`
	Name             *string `gorm:"column:Name"`
	PrimaryImageTag  *string `gorm:"column:PrimaryImageTag"`
	LastLoginDate    *string `gorm:"column:LastLoginDate"`
	LastActivityDate *string `gorm:"column:LastActivityDate"`
	IsAdministrator  bool    `gorm:"column:IsAdministrator;not null;default:false"`
}

func (JFUser) TableName() string { return "jf_users" }

// JFLibrary mirrors jf_libraries.
type JFLibrary struct {
	Id               string  `gorm:"column:Id;primaryKey"`
	Name             *string `gorm:"column:Name"`
	ServerId         *string `gorm:"column:ServerId"`
	IsFolder         *bool   `gorm:"column:IsFolder"`
	Type             *string `gorm:"column:Type"`
	CollectionType   *string `gorm:"column:CollectionType"`
	ImageTagsPrimary *string `gorm:"column:ImageTagsPrimary"`
	Archived         bool    `gorm:"column:archived;not null;default:false"`
}

func (JFLibrary) TableName() string { return "jf_libraries" }

// JFLibraryItem mirrors jf_library_items (Movie, Series, MusicAlbum).
type JFLibraryItem struct {
	Id               string         `gorm:"column:Id;primaryKey"`
	Name             *string        `gorm:"column:Name"`
	ServerId         *string        `gorm:"column:ServerId"`
	PremiereDate     *string        `gorm:"column:PremiereDate"`
	DateCreated      *string        `gorm:"column:DateCreated"`
	EndDate          *string        `gorm:"column:EndDate"`
	CommunityRating  *float64       `gorm:"column:CommunityRating"`
	RunTimeTicks     *int64         `gorm:"column:RunTimeTicks"`
	ProductionYear   *int           `gorm:"column:ProductionYear"`
	IsFolder         *bool          `gorm:"column:IsFolder"`
	Type             *string        `gorm:"column:Type"`
	Status           *string        `gorm:"column:Status"`
	ImageTagsPrimary *string        `gorm:"column:ImageTagsPrimary"`
	ImageTagsBanner  *string        `gorm:"column:ImageTagsBanner"`
	ImageTagsLogo    *string        `gorm:"column:ImageTagsLogo"`
	ImageTagsThumb   *string        `gorm:"column:ImageTagsThumb"`
	BackdropImageTags *string       `gorm:"column:BackdropImageTags"`
	ParentId         *string        `gorm:"column:ParentId"`
	PrimaryImageHash *string        `gorm:"column:PrimaryImageHash"`
	Archived         bool           `gorm:"column:archived;not null;default:false"`
	Genres           datatypes.JSON `gorm:"column:Genres;not null;default:'[]'"`
	AlbumArtist      *string        `gorm:"column:AlbumArtist"`
	ArtistId         *string        `gorm:"column:ArtistId"`
}

func (JFLibraryItem) TableName() string { return "jf_library_items" }

// JFLibrarySeason mirrors jf_library_seasons.
type JFLibrarySeason struct {
	Id                      string  `gorm:"column:Id;primaryKey"`
	Name                    *string `gorm:"column:Name"`
	ServerId                *string `gorm:"column:ServerId"`
	IndexNumber             *int    `gorm:"column:IndexNumber"`
	Type                    *string `gorm:"column:Type"`
	ParentLogoItemId        *string `gorm:"column:ParentLogoItemId"`
	ParentBackdropItemId    *string `gorm:"column:ParentBackdropItemId"`
	ParentBackdropImageTags *string `gorm:"column:ParentBackdropImageTags"`
	SeriesName              *string `gorm:"column:SeriesName"`
	SeriesId                *string `gorm:"column:SeriesId"`
	SeriesPrimaryImageTag   *string `gorm:"column:SeriesPrimaryImageTag"`
	Archived                bool    `gorm:"column:archived;not null;default:false"`
}

func (JFLibrarySeason) TableName() string { return "jf_library_seasons" }

// JFLibraryEpisode mirrors jf_library_episodes.
type JFLibraryEpisode struct {
	Id                      string   `gorm:"column:Id;primaryKey"`
	EpisodeId               *string  `gorm:"column:EpisodeId"`
	Name                    *string  `gorm:"column:Name"`
	ServerId                *string  `gorm:"column:ServerId"`
	PremiereDate            *string  `gorm:"column:PremiereDate"`
	DateCreated             *string  `gorm:"column:DateCreated"`
	OfficialRating          *string  `gorm:"column:OfficialRating"`
	CommunityRating         *float64 `gorm:"column:CommunityRating"`
	RunTimeTicks            *int64   `gorm:"column:RunTimeTicks"`
	ProductionYear          *int     `gorm:"column:ProductionYear"`
	IndexNumber             *int     `gorm:"column:IndexNumber"`
	ParentIndexNumber       *int     `gorm:"column:ParentIndexNumber"`
	Type                    *string  `gorm:"column:Type"`
	ParentLogoItemId        *string  `gorm:"column:ParentLogoItemId"`
	ParentBackdropItemId    *string  `gorm:"column:ParentBackdropItemId"`
	ParentBackdropImageTags *string  `gorm:"column:ParentBackdropImageTags"`
	SeriesId                *string  `gorm:"column:SeriesId"`
	SeasonId                *string  `gorm:"column:SeasonId"`
	SeasonName              *string  `gorm:"column:SeasonName"`
	SeriesName              *string  `gorm:"column:SeriesName"`
	PrimaryImageHash        *string  `gorm:"column:PrimaryImageHash"`
	Archived                bool     `gorm:"column:archived;not null;default:false"`
}

func (JFLibraryEpisode) TableName() string { return "jf_library_episodes" }

// JFMusicArtist mirrors jf_music_artists.
type JFMusicArtist struct {
	Id               string         `gorm:"column:Id;primaryKey"`
	LibraryId        *string        `gorm:"column:LibraryId"`
	Name             *string        `gorm:"column:Name"`
	Overview         *string        `gorm:"column:Overview"`
	ImageTagsPrimary *string        `gorm:"column:ImageTagsPrimary"`
	Genres           datatypes.JSON `gorm:"column:Genres;not null;default:'[]'"`
	Archived         bool           `gorm:"column:archived;not null;default:false"`
}

func (JFMusicArtist) TableName() string { return "jf_music_artists" }

// JFMusicTrack mirrors jf_music_tracks.
type JFMusicTrack struct {
	Id               string         `gorm:"column:Id;primaryKey"`
	LibraryId        *string        `gorm:"column:LibraryId"`
	AlbumId          *string        `gorm:"column:AlbumId"`
	ArtistId         *string        `gorm:"column:ArtistId"`
	Name             *string        `gorm:"column:Name"`
	AlbumName        *string        `gorm:"column:AlbumName"`
	AlbumArtist      *string        `gorm:"column:AlbumArtist"`
	IndexNumber      *int           `gorm:"column:IndexNumber"`
	DiscNumber       *int           `gorm:"column:DiscNumber"`
	RunTimeTicks     *int64         `gorm:"column:RunTimeTicks"`
	DateCreated      *string        `gorm:"column:DateCreated"`
	ProductionYear   *int           `gorm:"column:ProductionYear"`
	ImageTagsPrimary *string        `gorm:"column:ImageTagsPrimary"`
	Genres           datatypes.JSON `gorm:"column:Genres;not null;default:'[]'"`
	Archived         bool           `gorm:"column:archived;not null;default:false"`
}

func (JFMusicTrack) TableName() string { return "jf_music_tracks" }

// JFItemInfo mirrors jf_item_info.
type JFItemInfo struct {
	Id           string         `gorm:"column:Id;primaryKey"`
	Path         *string        `gorm:"column:Path"`
	Name         *string        `gorm:"column:Name"`
	Size         *int64         `gorm:"column:Size"`
	Bitrate      *int64         `gorm:"column:Bitrate"`
	MediaStreams  datatypes.JSON `gorm:"column:MediaStreams"`
	Type         *string        `gorm:"column:Type"`
}

func (JFItemInfo) TableName() string { return "jf_item_info" }

// JFPlaybackActivity mirrors jf_playback_activity.
type JFPlaybackActivity struct {
	Id                   string         `gorm:"column:Id;primaryKey"`
	IsPaused             *bool          `gorm:"column:IsPaused"`
	UserId               *string        `gorm:"column:UserId"`
	UserName             *string        `gorm:"column:UserName"`
	Client               *string        `gorm:"column:Client"`
	DeviceName           *string        `gorm:"column:DeviceName"`
	DeviceId             *string        `gorm:"column:DeviceId"`
	ApplicationVersion   *string        `gorm:"column:ApplicationVersion"`
	NowPlayingItemId     *string        `gorm:"column:NowPlayingItemId"`
	NowPlayingItemName   *string        `gorm:"column:NowPlayingItemName"`
	EpisodeId            *string        `gorm:"column:EpisodeId"`
	SeasonId             *string        `gorm:"column:SeasonId"`
	SeriesName           *string        `gorm:"column:SeriesName"`
	PlaybackDuration     *int64         `gorm:"column:PlaybackDuration"`
	PlayMethod           *string        `gorm:"column:PlayMethod"`
	ActivityDateInserted *string        `gorm:"column:ActivityDateInserted"`
	MediaStreams          datatypes.JSON `gorm:"column:MediaStreams"`
	TranscodingInfo      datatypes.JSON `gorm:"column:TranscodingInfo"`
	PlayState            datatypes.JSON `gorm:"column:PlayState"`
	OriginalContainer    *string        `gorm:"column:OriginalContainer"`
	RemoteEndPoint       *string        `gorm:"column:RemoteEndPoint"`
	ServerId             *string        `gorm:"column:ServerId"`
	Source               string         `gorm:"column:source;default:watchdog"`
	Imported             bool           `gorm:"column:imported;default:false"`
}

func (JFPlaybackActivity) TableName() string { return "jf_playback_activity" }

// JFActivityWatchdog mirrors jf_activity_watchdog.
type JFActivityWatchdog struct {
	Id                   string         `gorm:"column:Id;primaryKey"`
	ActivityId           *string        `gorm:"column:ActivityId"`
	IsPaused             *bool          `gorm:"column:IsPaused"`
	UserId               *string        `gorm:"column:UserId"`
	UserName             *string        `gorm:"column:UserName"`
	Client               *string        `gorm:"column:Client"`
	DeviceName           *string        `gorm:"column:DeviceName"`
	DeviceId             *string        `gorm:"column:DeviceId"`
	ApplicationVersion   *string        `gorm:"column:ApplicationVersion"`
	NowPlayingItemId     *string        `gorm:"column:NowPlayingItemId"`
	NowPlayingItemName   *string        `gorm:"column:NowPlayingItemName"`
	EpisodeId            *string        `gorm:"column:EpisodeId"`
	SeasonId             *string        `gorm:"column:SeasonId"`
	SeriesName           *string        `gorm:"column:SeriesName"`
	PlaybackDuration     *int64         `gorm:"column:PlaybackDuration"`
	PlayMethod           *string        `gorm:"column:PlayMethod"`
	ActivityDateInserted *string        `gorm:"column:ActivityDateInserted"`
	MediaStreams          datatypes.JSON `gorm:"column:MediaStreams"`
	TranscodingInfo      datatypes.JSON `gorm:"column:TranscodingInfo"`
	PlayState            datatypes.JSON `gorm:"column:PlayState"`
	OriginalContainer    *string        `gorm:"column:OriginalContainer"`
	RemoteEndPoint       *string        `gorm:"column:RemoteEndPoint"`
	ServerId             *string        `gorm:"column:ServerId"`
}

func (JFActivityWatchdog) TableName() string { return "jf_activity_watchdog" }

// JFPluginData mirrors jf_playback_reporting_plugin_data.
type JFPluginData struct {
	RowId          string  `gorm:"column:rowid;primaryKey"`
	DateCreated    *string `gorm:"column:DateCreated"`
	UserId         *string `gorm:"column:UserId"`
	ItemId         *string `gorm:"column:ItemId"`
	ItemType       *string `gorm:"column:ItemType"`
	ItemName       *string `gorm:"column:ItemName"`
	PlaybackMethod *string `gorm:"column:PlaybackMethod"`
	ClientName     *string `gorm:"column:ClientName"`
	DeviceName     *string `gorm:"column:DeviceName"`
	PlayDuration   *int64  `gorm:"column:PlayDuration"`
	Imported       bool    `gorm:"column:imported;default:false"`
	Source         string  `gorm:"column:source;default:plugin"`
}

func (JFPluginData) TableName() string { return "jf_playback_reporting_plugin_data" }

// JFLogging mirrors jf_logging.
type JFLogging struct {
	Id            string  `gorm:"column:Id;primaryKey"`
	Name          *string `gorm:"column:Name"`
	Type          *string `gorm:"column:Type"`
	ExecutionType *string `gorm:"column:ExecutionType"`
	Duration      *int64  `gorm:"column:Duration"`
	TimeRun       *string `gorm:"column:TimeRun"`
	Log           *string `gorm:"column:Log"`
	Result        *string `gorm:"column:Result"`
}

func (JFLogging) TableName() string { return "jf_logging" }

// Webhook mirrors the webhooks table.
type Webhook struct {
	Id             int            `gorm:"column:id;primaryKey;autoIncrement"                    json:"id"`
	Name           string         `gorm:"column:name;not null"                                  json:"name"`
	Url            string         `gorm:"column:url;not null"                                   json:"url"`
	Method         string         `gorm:"column:method;default:POST"                            json:"method"`
	TriggerType    string         `gorm:"column:trigger_type;not null"                          json:"trigger_type"`
	EventType      *string        `gorm:"column:event_type"                                     json:"event_type"`
	Schedule       *string        `gorm:"column:schedule"                                       json:"schedule"`
	Enabled        bool           `gorm:"column:enabled;default:true"                           json:"enabled"`
	Headers        datatypes.JSON `gorm:"column:headers;default:'{}'"                         json:"headers"`
	Payload        datatypes.JSON `gorm:"column:payload;default:'{}'"                         json:"payload"`
	RetryOnFailure bool           `gorm:"column:retry_on_failure;default:false"                 json:"retry_on_failure"`
	MaxRetries     int            `gorm:"column:max_retries;default:3"                          json:"max_retries"`
	LastTriggered  *time.Time     `gorm:"column:last_triggered"                                 json:"last_triggered"`
	CreatedAt      time.Time      `gorm:"column:created_at;autoCreateTime"                      json:"created_at"`
	UpdatedAt      time.Time      `gorm:"column:updated_at;autoUpdateTime"                      json:"updated_at"`
	BotUsername    string         `gorm:"column:bot_username;not null;default:jellystics_bot"   json:"bot_username"`
	BotAvatarUrl   string         `gorm:"column:bot_avatar_url;not null;default:''"           json:"bot_avatar_url"`
	DiscordEvents  datatypes.JSON `gorm:"column:discord_events;not null;default:'[]'"         json:"discord_events"`
}

func (Webhook) TableName() string { return "webhooks" }
