-- Revert view to original (without music track join).
CREATE OR REPLACE VIEW jf_playback_activity_with_metadata AS
SELECT
  a.*,
  e."IndexNumber"::int        AS "EpisodeNumber",
  e."ParentIndexNumber"::int  AS "SeasonNumber",
  COALESCE(i."ParentId", ep_item."ParentId") AS "ParentId"
FROM jf_playback_activity a
LEFT JOIN jf_library_episodes e      ON e."Id"  = a."EpisodeId"
LEFT JOIN jf_library_items    i      ON i."Id"  = a."NowPlayingItemId"
LEFT JOIN jf_library_items    ep_item ON ep_item."Id" = a."EpisodeId";
