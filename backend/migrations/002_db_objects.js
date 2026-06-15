exports.up = async function up(knex) {
  // View: jf_all_user_activity
  await knex.raw(`
    CREATE OR REPLACE VIEW jf_all_user_activity AS
    SELECT a.*, u."Name" AS "UserDisplayName"
    FROM jf_playback_activity a
    LEFT JOIN jf_users u ON a."UserId" = u."Id"
  `);

  // View: jf_playback_activity_with_metadata
  // Adds EpisodeNumber, SeasonNumber, and ParentId from library episode/item tables
  await knex.raw(`
    CREATE OR REPLACE VIEW jf_playback_activity_with_metadata AS
    SELECT
      a.*,
      e."IndexNumber"::int        AS "EpisodeNumber",
      e."ParentIndexNumber"::int  AS "SeasonNumber",
      COALESCE(i."ParentId", ep_item."ParentId") AS "ParentId"
    FROM jf_playback_activity a
    LEFT JOIN jf_library_episodes e      ON e."Id"  = a."EpisodeId"
    LEFT JOIN jf_library_items    i      ON i."Id"  = a."NowPlayingItemId"
    LEFT JOIN jf_library_items    ep_item ON ep_item."Id" = a."EpisodeId"
  `);

  // Materialized view: js_latest_playback_activity
  // One row per (item, episode, user) — the most recent play session
  await knex.raw(`
    CREATE MATERIALIZED VIEW js_latest_playback_activity AS
    SELECT DISTINCT ON (
      a."NowPlayingItemId",
      COALESCE(a."EpisodeId", ''),
      a."UserId"
    ) a.*
    FROM jf_playback_activity_with_metadata a
    ORDER BY
      a."NowPlayingItemId",
      COALESCE(a."EpisodeId", ''),
      a."UserId",
      a."ActivityDateInserted"::timestamptz DESC
  `);

  // Materialized view: js_library_stats_overview
  // Per-library aggregated stats used by library card endpoints
  await knex.raw(`
    CREATE MATERIALIZED VIEW js_library_stats_overview AS
    SELECT
      l."Id",
      l."Name",
      l."Type",
      l."CollectionType",
      COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "TotalItems",
      COUNT(DISTINCT ep."Id")::int                                                     AS "TotalEpisodes",
      COUNT(a."Id")::int                                                                AS "TotalPlays",
      MAX(a."ActivityDateInserted")::timestamptz                                        AS "ActivityDateInserted"
    FROM jf_libraries l
    LEFT JOIN jf_library_items    i  ON i."ParentId"  = l."Id" AND i.archived  = false
    LEFT JOIN jf_library_episodes ep ON ep."SeriesId" = i."Id"
    LEFT JOIN jf_playback_activity a ON a."NowPlayingItemId" = i."Id"
                                     OR a."EpisodeId"        = ep."Id"
    WHERE l.archived = false
    GROUP BY l."Id", l."Name", l."Type", l."CollectionType"
  `);

  // Materialized view: js_library_items_with_playcount_playtime
  // Library items enriched with play count, total play time and file size
  await knex.raw(`
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
  `);

  // View: js_library_metadata
  await knex.raw(`
    CREATE OR REPLACE VIEW js_library_metadata AS
    SELECT
      l.*,
      COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "TotalItems",
      COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" = 'Episode')::int                 AS "TotalEpisodes"
    FROM jf_libraries l
    LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
    WHERE l.archived = false
    GROUP BY l."Id"
  `);

  // Procedure: ji_insert_playback_plugin_data_to_activity_table
  // Merges rows from jf_playback_reporting_plugin_data into jf_playback_activity.
  // Uses a stable Id = 'plugin-' || rowid to prevent duplicates on re-run.
  await knex.raw(`
    CREATE OR REPLACE PROCEDURE ji_insert_playback_plugin_data_to_activity_table()
    LANGUAGE plpgsql
    AS $$
    BEGIN
      INSERT INTO jf_playback_activity (
        "Id",
        "IsPaused",
        "UserId",
        "UserName",
        "Client",
        "DeviceName",
        "DeviceId",
        "ApplicationVersion",
        "NowPlayingItemId",
        "NowPlayingItemName",
        "EpisodeId",
        "SeasonId",
        "SeriesName",
        "PlaybackDuration",
        "PlayMethod",
        "ActivityDateInserted",
        "MediaStreams",
        "TranscodingInfo",
        "PlayState",
        "OriginalContainer",
        "RemoteEndPoint",
        "ServerId"
      )
      SELECT
        'plugin-' || p.rowid,
        false,
        p."UserId",
        COALESCE(u."Name", p."UserId"),
        p."ClientName",
        p."DeviceName",
        NULL,
        NULL,
        p."ItemId",
        p."ItemName",
        NULL,
        NULL,
        CASE WHEN p."ItemType" = 'Episode' THEN p."ItemName" ELSE NULL END,
        p."PlayDuration",
        p."PlaybackMethod",
        p."DateCreated",
        NULL,
        NULL,
        NULL,
        NULL,
        NULL,
        NULL
      FROM jf_playback_reporting_plugin_data p
      LEFT JOIN jf_users u ON u."Id" = p."UserId"
      WHERE NOT EXISTS (
        SELECT 1 FROM jf_playback_activity a
        WHERE a."Id" = 'plugin-' || p.rowid
      );
    END;
    $$
  `);

  // Procedure: ju_update_library_stats_data
  // Refreshes all materialized views — called after full sync
  await knex.raw(`
    CREATE OR REPLACE PROCEDURE ju_update_library_stats_data()
    LANGUAGE plpgsql
    AS $$
    BEGIN
      REFRESH MATERIALIZED VIEW js_latest_playback_activity;
      REFRESH MATERIALIZED VIEW js_library_stats_overview;
      REFRESH MATERIALIZED VIEW js_library_items_with_playcount_playtime;
    END;
    $$
  `);

  // Function: fs_last_user_activity(text)
  // Returns playback history for a given user, newest first
  await knex.raw(`
    CREATE OR REPLACE FUNCTION fs_last_user_activity(p_userid text)
    RETURNS SETOF jf_playback_activity
    LANGUAGE sql
    AS $$
      SELECT * FROM jf_playback_activity
      WHERE "UserId" = p_userid
      ORDER BY "ActivityDateInserted"::timestamptz DESC;
    $$
  `);
};

exports.down = async function down(knex) {
  await knex.raw('DROP FUNCTION  IF EXISTS fs_last_user_activity(text)');
  await knex.raw('DROP PROCEDURE IF EXISTS ju_update_library_stats_data()');
  await knex.raw('DROP PROCEDURE IF EXISTS ji_insert_playback_plugin_data_to_activity_table()');
  await knex.raw('DROP VIEW      IF EXISTS js_library_metadata');
  await knex.raw('DROP MATERIALIZED VIEW IF EXISTS js_library_items_with_playcount_playtime');
  await knex.raw('DROP MATERIALIZED VIEW IF EXISTS js_library_stats_overview');
  await knex.raw('DROP MATERIALIZED VIEW IF EXISTS js_latest_playback_activity');
  await knex.raw('DROP VIEW      IF EXISTS jf_playback_activity_with_metadata');
  await knex.raw('DROP VIEW      IF EXISTS jf_all_user_activity');
};
