-- Recreate the playcount view to include music tracks alongside library items.
DROP MATERIALIZED VIEW IF EXISTS js_library_items_with_playcount_playtime;

CREATE MATERIALIZED VIEW js_library_items_with_playcount_playtime AS
SELECT
  i.*,
  info."Size",
  info."Bitrate",
  info."Path",
  COALESCE(stats."times_played",   0)::int    AS "times_played",
  COALESCE(stats."total_play_time", 0)::bigint AS "total_play_time"
FROM jf_library_items i
LEFT JOIN jf_item_info info ON info."Id" = i."Id"
LEFT JOIN (
  SELECT
    COALESCE("EpisodeId", "NowPlayingItemId") AS item_id,
    COUNT(*)::int                              AS "times_played",
    COALESCE(SUM("PlaybackDuration"), 0)::bigint AS "total_play_time"
  FROM jf_playback_activity
  GROUP BY item_id
) stats ON stats.item_id = i."Id"

UNION ALL

SELECT
  mt."Id",
  mt."Name",
  NULL AS "ServerId",
  NULL AS "PremiereDate",
  mt."DateCreated",
  NULL AS "EndDate",
  NULL AS "CommunityRating",
  mt."RunTimeTicks",
  mt."ProductionYear",
  NULL AS "IsFolder",
  'Audio' AS "Type",
  NULL AS "Status",
  mt."ImageTagsPrimary",
  NULL AS "ImageTagsBanner",
  NULL AS "ImageTagsLogo",
  NULL AS "ImageTagsThumb",
  NULL AS "BackdropImageTags",
  mt."LibraryId" AS "ParentId",
  NULL AS "PrimaryImageHash",
  mt.archived,
  mt."Genres",
  mt."AlbumArtist",
  mt."ArtistId",
  info."Size",
  info."Bitrate",
  info."Path",
  COALESCE(stats."times_played",   0)::int    AS "times_played",
  COALESCE(stats."total_play_time", 0)::bigint AS "total_play_time"
FROM jf_music_tracks mt
LEFT JOIN jf_item_info info ON info."Id" = mt."Id"
LEFT JOIN (
  SELECT
    "EpisodeId" AS item_id,
    COUNT(*)::int                              AS "times_played",
    COALESCE(SUM("PlaybackDuration"), 0)::bigint AS "total_play_time"
  FROM jf_playback_activity
  WHERE "EpisodeId" IS NOT NULL
  GROUP BY "EpisodeId"
) stats ON stats.item_id = mt."Id";
