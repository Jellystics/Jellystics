-- app_config: single-row app settings
CREATE TABLE IF NOT EXISTS app_config (
  "ID"             integer PRIMARY KEY,
  "JF_HOST"        text,
  "JF_API_KEY"     text,
  "APP_USER"       text,
  "APP_PASSWORD"   text,
  "REQUIRE_LOGIN"  boolean NOT NULL DEFAULT false,
  settings         jsonb   NOT NULL DEFAULT '{}',
  api_keys         jsonb   NOT NULL DEFAULT '[]'
);
INSERT INTO app_config ("ID", settings, api_keys)
VALUES (1, '{}', '[]')
ON CONFLICT ("ID") DO NOTHING;

-- jf_users
CREATE TABLE IF NOT EXISTS jf_users (
  "Id"                 text PRIMARY KEY,
  "Name"               text,
  "PrimaryImageTag"    text,
  "LastLoginDate"      text,
  "LastActivityDate"   text,
  "IsAdministrator"    boolean NOT NULL DEFAULT false
);

-- jf_libraries
CREATE TABLE IF NOT EXISTS jf_libraries (
  "Id"               text PRIMARY KEY,
  "Name"             text,
  "ServerId"         text,
  "IsFolder"         boolean,
  "Type"             text,
  "CollectionType"   text,
  "ImageTagsPrimary" text,
  archived           boolean NOT NULL DEFAULT false
);

-- jf_library_items: Movie, Series, MusicAlbum
CREATE TABLE IF NOT EXISTS jf_library_items (
  "Id"               text PRIMARY KEY,
  "Name"             text,
  "ServerId"         text,
  "PremiereDate"     text,
  "DateCreated"      text,
  "EndDate"          text,
  "CommunityRating"  numeric,
  "RunTimeTicks"     bigint,
  "ProductionYear"   integer,
  "IsFolder"         boolean,
  "Type"             text,
  "Status"           text,
  "ImageTagsPrimary" text,
  "ImageTagsBanner"  text,
  "ImageTagsLogo"    text,
  "ImageTagsThumb"   text,
  "BackdropImageTags" text,
  "ParentId"         text,
  "PrimaryImageHash" text,
  archived           boolean NOT NULL DEFAULT false,
  "Genres"           jsonb   NOT NULL DEFAULT '[]',
  "AlbumArtist"      text,
  "ArtistId"         text
);

-- jf_library_seasons
CREATE TABLE IF NOT EXISTS jf_library_seasons (
  "Id"                       text PRIMARY KEY,
  "Name"                     text,
  "ServerId"                 text,
  "IndexNumber"              integer,
  "Type"                     text,
  "ParentLogoItemId"         text,
  "ParentBackdropItemId"     text,
  "ParentBackdropImageTags"  text,
  "SeriesName"               text,
  "SeriesId"                 text,
  "SeriesPrimaryImageTag"    text,
  archived                   boolean NOT NULL DEFAULT false
);

-- jf_library_episodes
CREATE TABLE IF NOT EXISTS jf_library_episodes (
  "Id"                       text PRIMARY KEY,
  "EpisodeId"                text,
  "Name"                     text,
  "ServerId"                 text,
  "PremiereDate"             text,
  "DateCreated"              text,
  "OfficialRating"           text,
  "CommunityRating"          numeric,
  "RunTimeTicks"             bigint,
  "ProductionYear"           integer,
  "IndexNumber"              integer,
  "ParentIndexNumber"        integer,
  "Type"                     text,
  "ParentLogoItemId"         text,
  "ParentBackdropItemId"     text,
  "ParentBackdropImageTags"  text,
  "SeriesId"                 text,
  "SeasonId"                 text,
  "SeasonName"               text,
  "SeriesName"               text,
  "PrimaryImageHash"         text,
  archived                   boolean NOT NULL DEFAULT false
);

-- jf_music_artists
CREATE TABLE IF NOT EXISTS jf_music_artists (
  "Id"               text PRIMARY KEY,
  "LibraryId"        text,
  "Name"             text,
  "Overview"         text,
  "ImageTagsPrimary" text,
  "Genres"           jsonb NOT NULL DEFAULT '[]',
  archived           boolean NOT NULL DEFAULT false
);

-- jf_music_tracks
CREATE TABLE IF NOT EXISTS jf_music_tracks (
  "Id"               text PRIMARY KEY,
  "LibraryId"        text,
  "AlbumId"          text,
  "ArtistId"         text,
  "Name"             text,
  "AlbumName"        text,
  "AlbumArtist"      text,
  "IndexNumber"      integer,
  "DiscNumber"       integer,
  "RunTimeTicks"     bigint,
  "DateCreated"      text,
  "ProductionYear"   integer,
  "ImageTagsPrimary" text,
  "Genres"           jsonb NOT NULL DEFAULT '[]',
  archived           boolean NOT NULL DEFAULT false
);

-- jf_item_info
CREATE TABLE IF NOT EXISTS jf_item_info (
  "Id"           text PRIMARY KEY,
  "Path"         text,
  "Name"         text,
  "Size"         bigint,
  "Bitrate"      bigint,
  "MediaStreams" jsonb,
  "Type"         text
);

-- jf_playback_activity
CREATE TABLE IF NOT EXISTS jf_playback_activity (
  "Id"                   text PRIMARY KEY,
  "IsPaused"             boolean,
  "UserId"               text,
  "UserName"             text,
  "Client"               text,
  "DeviceName"           text,
  "DeviceId"             text,
  "ApplicationVersion"   text,
  "NowPlayingItemId"     text,
  "NowPlayingItemName"   text,
  "EpisodeId"            text,
  "SeasonId"             text,
  "SeriesName"           text,
  "PlaybackDuration"     bigint,
  "PlayMethod"           text,
  "ActivityDateInserted" text,
  "MediaStreams"         jsonb,
  "TranscodingInfo"      jsonb,
  "PlayState"            jsonb,
  "OriginalContainer"    text,
  "RemoteEndPoint"       text,
  "ServerId"             text,
  source                 text DEFAULT 'watchdog',
  imported               boolean DEFAULT false
);

-- jf_activity_watchdog
CREATE TABLE IF NOT EXISTS jf_activity_watchdog (
  "Id"                   text PRIMARY KEY,
  "ActivityId"           text,
  "IsPaused"             boolean,
  "UserId"               text,
  "UserName"             text,
  "Client"               text,
  "DeviceName"           text,
  "DeviceId"             text,
  "ApplicationVersion"   text,
  "NowPlayingItemId"     text,
  "NowPlayingItemName"   text,
  "EpisodeId"            text,
  "SeasonId"             text,
  "SeriesName"           text,
  "PlaybackDuration"     bigint,
  "PlayMethod"           text,
  "ActivityDateInserted" text,
  "MediaStreams"         jsonb,
  "TranscodingInfo"      jsonb,
  "PlayState"            jsonb,
  "OriginalContainer"    text,
  "RemoteEndPoint"       text,
  "ServerId"             text
);

-- jf_playback_reporting_plugin_data
CREATE TABLE IF NOT EXISTS jf_playback_reporting_plugin_data (
  rowid            text PRIMARY KEY,
  "DateCreated"    text,
  "UserId"         text,
  "ItemId"         text,
  "ItemType"       text,
  "ItemName"       text,
  "PlaybackMethod" text,
  "ClientName"     text,
  "DeviceName"     text,
  "PlayDuration"   bigint,
  imported         boolean DEFAULT false,
  source           text    DEFAULT 'plugin'
);

-- jf_logging
CREATE TABLE IF NOT EXISTS jf_logging (
  "Id"            text PRIMARY KEY,
  "Name"          text,
  "Type"          text,
  "ExecutionType" text,
  "Duration"      bigint,
  "TimeRun"       text,
  "Log"           text,
  "Result"        text
);

-- webhooks
CREATE TABLE IF NOT EXISTS webhooks (
  id                serial PRIMARY KEY,
  name              text    NOT NULL,
  url               text    NOT NULL,
  method            text    DEFAULT 'POST',
  trigger_type      text    NOT NULL,
  event_type        text,
  schedule          text,
  enabled           boolean DEFAULT true,
  headers           jsonb   DEFAULT '{}',
  payload           jsonb   DEFAULT '{}',
  retry_on_failure  boolean DEFAULT false,
  max_retries       integer DEFAULT 3,
  last_triggered    timestamptz,
  created_at        timestamptz DEFAULT now(),
  updated_at        timestamptz DEFAULT now()
);
