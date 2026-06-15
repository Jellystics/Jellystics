const db = require("../db");
const dbHelper = require("../classes/db-helper");

/** ActivityDateInserted is stored as text — always cast before date math. */
const ACTIVITY_TS = `"ActivityDateInserted"::timestamptz`;

async function rows(sql, params = []) {
  const result = await db.query(sql, params);
  if (Array.isArray(result)) return [];
  return result?.rows ?? [];
}

async function one(sql, params = []) {
  const list = await rows(sql, params);
  return list[0] ?? null;
}

function parseDays(value, fallback = 30) {
  const n = parseInt(value, 10);
  return Number.isFinite(n) && n > 0 ? n : fallback;
}

async function getGlobalStats() {
  return one(`
    SELECT
      (SELECT COUNT(*)::int FROM jf_playback_activity) AS "TotalPlays",
      (SELECT COALESCE(SUM("PlaybackDuration"), 0)::int FROM jf_playback_activity) AS "TotalWatchTime",
      (SELECT COUNT(DISTINCT "UserId")::int FROM jf_playback_activity
        WHERE ${ACTIVITY_TS} >= CURRENT_DATE - INTERVAL '30 days') AS "ActiveUsers",
      (SELECT COUNT(*)::int FROM jf_users) AS "TotalUsers",
      (SELECT COUNT(*)::int FROM jf_libraries WHERE archived = false) AS "TotalLibraries",
      (SELECT COUNT(*)::int FROM jf_library_items
        WHERE archived = false AND "Type" NOT IN ('Season', 'Folder')) AS "TotalItems"
  `);
}

async function getMostPlayedItems({ type = "all", limit = 5, days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;
  const itemLimit = parseInt(limit, 10) || 5;
  let typeFilter = "";
  const params = [daySpan];

  if (type === "Series") {
    typeFilter = `AND a."SeriesName" IS NOT NULL AND a."SeriesName" <> ''`;
  } else if (type === "Audio") {
    typeFilter = `AND i."Type" = 'Audio'`;
  } else if (type === "Movie") {
    typeFilter = `AND COALESCE(i."Type", '') IN ('Movie', 'Video')`;
  } else if (type !== "all") {
    params.push(type);
    typeFilter = `AND i."Type" = $${params.length}`;
  }

  params.push(itemLimit);

  return rows(
    `
    SELECT
      COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemId") AS "Id",
      COALESCE(NULLIF(a."SeriesName", ''), a."NowPlayingItemName") AS "Name",
      COUNT(*)::int AS "PlayCount",
      CASE
        WHEN a."SeriesName" IS NOT NULL AND a."SeriesName" <> '' THEN 'Series'
        ELSE COALESCE(i."Type", 'Unknown')
      END AS "Type"
    FROM jf_playback_activity a
    LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
      ${typeFilter}
    GROUP BY 1, 2, 4
    ORDER BY "PlayCount" DESC
    LIMIT $${params.length}
    `,
    params
  );
}

async function getMostActiveUsers({ limit = 5, days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;
  const userLimit = parseInt(limit, 10) || 5;

  return rows(
    `
    SELECT
      "UserId",
      MAX("UserName") AS "UserName",
      COUNT(*)::int AS "TotalPlays",
      COALESCE(SUM("PlaybackDuration"), 0)::int AS "TotalWatchTime"
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY "UserId"
    ORDER BY "TotalPlays" DESC
    LIMIT $2
    `,
    [daySpan, userLimit]
  );
}

async function getWatchStatisticsOverTime({ days = 30, userId } = {}) {
  const daySpan = parseDays(days) - 1;
  const params = [daySpan];
  let userFilter = "";

  if (userId) {
    params.push(userId);
    userFilter = `AND "UserId" = $${params.length}`;
  }

  return rows(
    `
    SELECT
      TO_CHAR((${ACTIVITY_TS})::date, 'YYYY-MM-DD') AS date,
      COUNT(*)::int AS plays,
      COALESCE(SUM("PlaybackDuration"), 0)::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
      ${userFilter}
    GROUP BY (${ACTIVITY_TS})::date
    ORDER BY date
    `,
    params
  );
}

async function getPopularHourOfDay({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  return rows(
    `
    SELECT
      EXTRACT(HOUR FROM ${ACTIVITY_TS})::int AS hour,
      COUNT(*)::int AS plays
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY hour
    ORDER BY hour
    `,
    [daySpan]
  );
}

async function getPopularDayOfWeek({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  return rows(
    `
    SELECT
      EXTRACT(DOW FROM ${ACTIVITY_TS})::int AS day,
      COUNT(*)::int AS plays
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY day
    ORDER BY day
    `,
    [daySpan]
  );
}

async function getMostUsedPlaybackMethod({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  return rows(
    `
    SELECT
      COALESCE(NULLIF("PlayMethod", ''), 'Unknown') AS method,
      COUNT(*)::int AS count
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY method
    ORDER BY count DESC
    `,
    [daySpan]
  );
}

async function getMostUsedClients({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  return rows(
    `
    SELECT
      "Client" AS client,
      COUNT(*)::int AS count
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
      AND "Client" IS NOT NULL AND "Client" <> ''
    GROUP BY "Client"
    ORDER BY count DESC
    LIMIT 10
    `,
    [daySpan]
  );
}

async function getUserStats(userId) {
  if (userId) {
    return one(
      `
      SELECT
        u."Id" AS "UserId",
        u."Name" AS "UserName",
        COUNT(a."Id")::int AS "TotalPlays",
        COALESCE(SUM(a."PlaybackDuration"), 0)::int AS "TotalWatchTime",
        COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
        NULL::text AS "FavoriteGenre"
      FROM jf_users u
      LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
      WHERE u."Id" = $1
      GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
      `,
      [userId]
    );
  }

  return rows(
    `
    SELECT
      u."Id" AS "UserId",
      u."Name" AS "UserName",
      COUNT(a."Id")::int AS "TotalPlays",
      COALESCE(SUM(a."PlaybackDuration"), 0)::int AS "TotalWatchTime",
      COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
      NULL::text AS "FavoriteGenre"
    FROM jf_users u
    LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
    GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
    ORDER BY "TotalPlays" DESC
    `
  );
}

async function getUserActivity(userId) {
  return rows(
    `
    SELECT
      "Id", "UserId", "UserName",
      "NowPlayingItemId" AS "ItemId",
      "NowPlayingItemName",
      "SeriesName", "SeasonId", "EpisodeId",
      "Client", "DeviceName", "DeviceId", "ApplicationVersion",
      "PlayMethod",
      "IsPaused",
      false AS "IsActive",
      ("PlaybackDuration" * 600000000)::bigint AS "PlayDuration",
      "ActivityDateInserted",
      "RemoteEndPoint"
    FROM jf_playback_activity
    WHERE "UserId" = $1
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 200
    `,
    [userId]
  );
}

async function getUserActivityByDate(userId) {
  return rows(
    `
    SELECT
      TO_CHAR((${ACTIVITY_TS})::date, 'YYYY-MM-DD') AS date,
      COUNT(*)::int AS count
    FROM jf_playback_activity
    WHERE "UserId" = $1
    GROUP BY (${ACTIVITY_TS})::date
    ORDER BY date
    `,
    [userId]
  );
}

async function getGenreStats({ userId, libraryId }) {
  if (libraryId && !userId) {
    return rows(
      `
      SELECT
        genre AS "Genre",
        COUNT(*)::int AS "Count",
        0::int AS "PlayCount"
      FROM jf_library_items i
      CROSS JOIN LATERAL jsonb_array_elements_text(
        CASE
          WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
          ELSE i."Genres"
        END
      ) AS genre
      WHERE i."ParentId" = $1
        AND i.archived = false
        AND i."Type" NOT IN ('Season', 'Folder')
      GROUP BY genre
      ORDER BY "Count" DESC, genre ASC
      `,
      [libraryId]
    );
  }

  const values = [];
  const where = [];

  if (userId) {
    values.push(userId);
    where.push([{ column: "a.UserId", operator: "=", value: `$${values.length}` }]);
  }
  if (libraryId) {
    values.push(libraryId);
    where.push([{ column: "i.ParentId", operator: "=", value: `$${values.length}` }]);
  }

  const query = {
    select: ["COALESCE(g.genre, 'No Genre') AS genre", "COUNT(*) AS plays"],
    table: "jf_playback_activity",
    alias: "a",
    joins: [
      {
        type: "inner",
        table: "jf_library_items",
        alias: "i",
        conditions: [{ first: "a.NowPlayingItemId", operator: "=", second: "i.Id" }],
      },
      {
        type: "left",
        table: `
          LATERAL (
            SELECT jsonb_array_elements_text(
              CASE
                WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
                ELSE i."Genres"
              END
            ) AS genre
          )
        `,
        alias: "g",
        conditions: [{ first: 1, operator: "=", value: 1, wrap: false }],
      },
    ],
    where,
    group_by: [`COALESCE(g.genre, 'No Genre')`],
    order_by: "plays",
    sort_order: "desc",
    pageNumber: 1,
    pageSize: 100,
    values,
  };

  const result = await dbHelper.query(query);
  return (result?.results ?? []).map((row) => ({
    Genre: row.genre,
    Count: parseInt(row.plays, 10) || 0,
    PlayCount: parseInt(row.plays, 10) || 0,
  }));
}

async function getLibraries() {
  return rows(
    `
    SELECT
      l."Id",
      l."Name",
      COALESCE(l."CollectionType", l."Type", 'unknown') AS "CollectionType",
      COUNT(i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "ItemCount",
      COUNT(i."Id") FILTER (WHERE i."Type" = 'Episode')::int AS "EpisodeCount"
    FROM jf_libraries l
    LEFT JOIN jf_library_items i
      ON i."ParentId" = l."Id" AND i.archived = false
    WHERE l.archived = false
    GROUP BY l."Id", l."Name", l."CollectionType", l."Type"
    ORDER BY l."Name"
    `
  );
}

async function getLibraryStats(libraryId) {
  const stats = await one(
    `
    SELECT
      l."Name",
      COUNT(DISTINCT i."Id") FILTER (WHERE i."Type" NOT IN ('Season', 'Folder'))::int AS "TotalItems",
      COUNT(a."Id")::int AS "TotalPlayCount",
      COALESCE(SUM(a."PlaybackDuration"), 0)::int AS "TotalWatchTime"
    FROM jf_libraries l
    LEFT JOIN jf_library_items i
      ON i."ParentId" = l."Id" AND i.archived = false
    LEFT JOIN jf_playback_activity a
      ON a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id"
    WHERE l."Id" = $1
    GROUP BY l."Id", l."Name"
    `,
    [libraryId]
  );

  if (!stats) return null;

  const topItem = await one(
    `
    SELECT
      i."Id",
      i."Name",
      i."Type",
      COUNT(a."Id")::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN jf_playback_activity a
      ON a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id"
    WHERE i."ParentId" = $1
      AND i.archived = false
      AND i."Type" NOT IN ('Season', 'Folder')
    GROUP BY i."Id", i."Name", i."Type"
    ORDER BY "PlayCount" DESC, i."Name" ASC
    LIMIT 1
    `,
    [libraryId]
  );

  return {
    Name: stats.Name,
    TotalItems: stats.TotalItems ?? 0,
    TotalPlayCount: stats.TotalPlayCount ?? 0,
    TotalWatchTime: stats.TotalWatchTime ?? 0,
    MostPlayedItem: topItem
      ? { Id: topItem.Id, Name: topItem.Name, Type: topItem.Type, PlayCount: topItem.PlayCount }
      : undefined,
  };
}

async function getLibraryItems(libraryId) {
  return rows(
    `
    SELECT
      i."Id",
      i."Name",
      i."Type",
      i."ProductionYear",
      i."CommunityRating",
      info."Size",
      COALESCE(pc.play_count, 0)::int AS "PlayCount",
      pc.last_played AS "LastPlayed",
      NULL::text AS "SeriesName",
      NULL::int AS "IndexNumber",
      NULL::int AS "ParentIndexNumber"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT
        COALESCE("EpisodeId", "NowPlayingItemId") AS item_id,
        COUNT(*)::int AS play_count,
        MAX("ActivityDateInserted") AS last_played
      FROM jf_playback_activity
      GROUP BY item_id
    ) pc ON pc.item_id = i."Id"
    LEFT JOIN jf_item_info info ON info."Id" = i."Id"
    WHERE i."ParentId" = $1
      AND i.archived = false
      AND i."Type" NOT IN ('Season', 'Folder')
    ORDER BY "PlayCount" DESC NULLS LAST
    LIMIT 500
    `,
    [libraryId]
  );
}

async function getActivityTimeline() {
  return rows(
    `
    SELECT
      a."Id",
      a."UserId",
      a."UserName",
      a."NowPlayingItemId" AS "ItemId",
      a."NowPlayingItemName" AS "ItemName",
      a."ActivityDateInserted" AS "StartTime",
      (a."ActivityDateInserted"::timestamptz + (a."PlaybackDuration" * INTERVAL '1 minute'))::text AS "EndTime",
      (a."PlaybackDuration" * 60)::int AS "Duration",
      a."Client",
      a."PlayMethod"
    FROM jf_playback_activity a
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 500
    `
  );
}

module.exports = {
  getGlobalStats,
  getMostPlayedItems,
  getMostActiveUsers,
  getWatchStatisticsOverTime,
  getPopularHourOfDay,
  getPopularDayOfWeek,
  getMostUsedPlaybackMethod,
  getMostUsedClients,
  getUserStats,
  getUserActivity,
  getUserActivityByDate,
  getGenreStats,
  getLibraries,
  getLibraryStats,
  getLibraryItems,
  getActivityTimeline,
};
