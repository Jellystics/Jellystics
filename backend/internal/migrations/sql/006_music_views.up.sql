-- Update view to also join music tracks for ParentId resolution.
CREATE OR REPLACE VIEW jf_playback_activity_with_metadata AS
SELECT
  a.*,
  e."IndexNumber"::int        AS "EpisodeNumber",
  e."ParentIndexNumber"::int  AS "SeasonNumber",
  COALESCE(i."ParentId", ep_item."ParentId", mt."LibraryId") AS "ParentId"
FROM jf_playback_activity a
LEFT JOIN jf_library_episodes e       ON e."Id"  = a."EpisodeId"
LEFT JOIN jf_library_items    i       ON i."Id"  = a."NowPlayingItemId"
LEFT JOIN jf_library_items    ep_item ON ep_item."Id" = a."EpisodeId"
LEFT JOIN jf_music_tracks     mt      ON mt."Id" = a."EpisodeId";

-- Update procedure to also handle music tracks in plugin import.
CREATE OR REPLACE PROCEDURE ji_insert_playback_plugin_data_to_activity_table()
LANGUAGE plpgsql
AS $$
BEGIN
  INSERT INTO jf_playback_activity (
    "Id", "IsPaused", "UserId", "UserName", "Client", "DeviceName", "DeviceId",
    "ApplicationVersion", "NowPlayingItemId", "NowPlayingItemName", "EpisodeId",
    "SeasonId", "SeriesName", "PlaybackDuration", "PlayMethod", "ActivityDateInserted",
    "MediaStreams", "TranscodingInfo", "PlayState", "OriginalContainer",
    "RemoteEndPoint", "ServerId", source, imported
  )
  SELECT
    'plugin-' || p.rowid,
    false,
    p."UserId",
    COALESCE(u."Name", p."UserId"),
    p."ClientName",
    p."DeviceName",
    NULL, NULL,
    COALESCE(e."SeriesId", mt."AlbumId", p."ItemId"),
    p."ItemName",
    CASE WHEN e."Id" IS NOT NULL OR mt."Id" IS NOT NULL THEN p."ItemId" ELSE NULL END,
    e."SeasonId",
    COALESCE(s."Name", mt."AlbumName"),
    p."PlayDuration",
    p."PlaybackMethod",
    p."DateCreated",
    NULL, NULL, NULL, NULL, NULL, NULL,
    'plugin',
    true
  FROM jf_playback_reporting_plugin_data p
  LEFT JOIN jf_users u ON u."Id" = p."UserId"
  LEFT JOIN jf_library_episodes e ON e."Id" = p."ItemId" OR e."EpisodeId" = p."ItemId"
  LEFT JOIN jf_library_items s ON s."Id" = e."SeriesId"
  LEFT JOIN jf_music_tracks mt ON mt."Id" = p."ItemId"
  WHERE NOT EXISTS (
    SELECT 1 FROM jf_playback_activity a
    WHERE a."Id" = 'plugin-' || p.rowid
  );

  UPDATE jf_playback_reporting_plugin_data
  SET imported = true
  WHERE imported = false
    AND EXISTS (
      SELECT 1 FROM jf_playback_activity a
      WHERE a."Id" = 'plugin-' || jf_playback_reporting_plugin_data.rowid
    );
END;
$$;
