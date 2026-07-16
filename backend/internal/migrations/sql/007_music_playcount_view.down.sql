-- Revert to original view (library items only, no music tracks).
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
) stats ON stats.item_id = i."Id";
