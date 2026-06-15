const db = require("../db");
const API = require("../classes/api-loader");

/** ActivityDateInserted is stored as text — always cast before date math. */
const ACTIVITY_TS = `"ActivityDateInserted"::timestamptz`;
const PLAYBACK_MINUTES = `FLOOR(COALESCE("PlaybackDuration", 0) / 60.0)`;
const PLAYBACK_MINUTES_SUM = `FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)`;

async function rows(sql, params = []) {
  const result = await db.query(sql, params);
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

function ticksToMinutes(ticks) {
  if (!ticks) return 0;
  return Math.floor(Number(ticks) / 600000000);
}

async function getLiveActivity() {
  const sessions = await API.getSessions();
  const activeSessions = sessions.filter((session) => session.NowPlayingItem);

  if (activeSessions.length === 0) return [];

  const itemIds = [
    ...new Set(
      activeSessions
        .flatMap((session) => [session.NowPlayingItem.Id, session.NowPlayingItem.SeriesId])
        .filter(Boolean)
    ),
  ];

  const libraryRows =
    itemIds.length > 0
      ? await rows(
          `SELECT "Id", "ParentId", "Type", "Name", "Genres" FROM jf_library_items WHERE "Id" = ANY($1)`,
          [itemIds]
        )
      : [];

  return activeSessions.map((session) => {
    const nowPlaying = session.NowPlayingItem;
    const lookupId = nowPlaying.SeriesId || nowPlaying.Id;
    const libraryItem = libraryRows.find((item) => item.Id === lookupId || item.Id === nowPlaying.Id);

    return {
      UserId: session.UserId,
      UserName: session.UserName,
      Client: session.Client,
      PlayMethod: session.PlayState?.PlayMethod || "Unknown",
      NowPlayingItemId: lookupId,
      EpisodeId: nowPlaying.SeriesId ? nowPlaying.Id : null,
      NowPlayingItemName: nowPlaying.SeriesName || nowPlaying.Name,
      Type: nowPlaying.SeriesId ? "Series" : libraryItem?.Type || nowPlaying.Type || "Unknown",
      Genres: libraryItem?.Genres ?? [],
      LibraryId: libraryItem?.ParentId,
      PlaybackDuration: ticksToMinutes(session.PlayState?.PositionTicks),
      date: new Date().toISOString().slice(0, 10),
      hour: new Date().getHours(),
      day: new Date().getDay(),
    };
  });
}

async function getGlobalStats() {
  const stats = await one(`
    SELECT
      (SELECT COUNT(*)::int FROM jf_playback_activity) AS "TotalPlays",
      (SELECT ${PLAYBACK_MINUTES_SUM}::int FROM jf_playback_activity) AS "TotalWatchTime",
      (SELECT COUNT(DISTINCT "UserId")::int FROM jf_playback_activity
        WHERE ${ACTIVITY_TS} >= CURRENT_DATE - INTERVAL '30 days') AS "ActiveUsers",
      (SELECT COUNT(*)::int FROM jf_users) AS "TotalUsers",
      (SELECT COUNT(*)::int FROM jf_libraries WHERE archived = false) AS "TotalLibraries",
      (SELECT COUNT(*)::int FROM jf_library_items
        WHERE archived = false AND "Type" NOT IN ('Season', 'Folder')) AS "TotalItems"
  `);
  const live = await getLiveActivity();

  return {
    ...stats,
    TotalPlays: (stats?.TotalPlays ?? 0) + live.length,
    TotalWatchTime: (stats?.TotalWatchTime ?? 0) + live.reduce((sum, item) => sum + item.PlaybackDuration, 0),
    ActiveUsers: new Set([...(live.map((item) => item.UserId)), ...(stats?.ActiveUsers ? [] : [])]).size || stats?.ActiveUsers || 0,
  };
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

  const dbItems = await rows(
    `
    SELECT
      a."NowPlayingItemId" AS "Id",
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
    GROUP BY a."NowPlayingItemId", 2, 4
    ORDER BY "PlayCount" DESC
    LIMIT $${params.length}
    `,
    params
  );
  const live = await getLiveActivity();
  const liveItems = live.reduce((acc, session) => {
    const key = session.NowPlayingItemId;
    if (!acc[key]) {
      acc[key] = {
        Id: session.NowPlayingItemId,
        Name: session.NowPlayingItemName,
        PlayCount: 0,
        Type: session.Type,
      };
    }
    acc[key].PlayCount += 1;
    return acc;
  }, {});

  return [...dbItems, ...Object.values(liveItems)]
    .reduce((acc, item) => {
      const existing = acc.find((row) => row.Id === item.Id);
      if (existing) {
        existing.PlayCount += item.PlayCount;
      } else {
        acc.push({ ...item });
      }
      return acc;
    }, [])
    .sort((a, b) => b.PlayCount - a.PlayCount)
    .slice(0, itemLimit);
}

async function getMostActiveUsers({ limit = 5, days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;
  const userLimit = parseInt(limit, 10) || 5;

  const dbUsers = await rows(
    `
    SELECT
      "UserId",
      MAX("UserName") AS "UserName",
      COUNT(*)::int AS "TotalPlays",
      ${PLAYBACK_MINUTES_SUM}::int AS "TotalWatchTime"
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY "UserId"
    ORDER BY "TotalPlays" DESC
    LIMIT $2
    `,
    [daySpan, userLimit]
  );
  const live = await getLiveActivity();
  const merged = [...dbUsers];

  live.forEach((session) => {
    const existing = merged.find((user) => user.UserId === session.UserId);
    if (existing) {
      existing.TotalPlays += 1;
      existing.TotalWatchTime += session.PlaybackDuration;
    } else {
      merged.push({
        UserId: session.UserId,
        UserName: session.UserName,
        TotalPlays: 1,
        TotalWatchTime: session.PlaybackDuration,
      });
    }
  });

  return merged.sort((a, b) => b.TotalPlays - a.TotalPlays).slice(0, userLimit);
}

async function getWatchStatisticsOverTime({ days = 30, userId } = {}) {
  const daySpan = parseDays(days) - 1;
  const params = [daySpan];
  let userFilter = "";

  if (userId) {
    params.push(userId);
    userFilter = `AND "UserId" = $${params.length}`;
  }

  const dbStats = await rows(
    `
    SELECT
      TO_CHAR((${ACTIVITY_TS})::date, 'YYYY-MM-DD') AS date,
      COUNT(*)::int AS plays,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
      ${userFilter}
    GROUP BY (${ACTIVITY_TS})::date
    ORDER BY date
    `,
    params
  );
  const live = (await getLiveActivity()).filter((session) => !userId || session.UserId === userId);
  const merged = [...dbStats];

  live.forEach((session) => {
    const existing = merged.find((point) => point.date === session.date);
    if (existing) {
      existing.plays += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      merged.push({ date: session.date, plays: 1, duration: session.PlaybackDuration });
    }
  });

  return merged.sort((a, b) => a.date.localeCompare(b.date));
}

async function getPopularHourOfDay({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  const dbHours = await rows(
    `
    SELECT
      EXTRACT(HOUR FROM ${ACTIVITY_TS})::int AS hour,
      COUNT(*)::int AS plays,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY hour
    ORDER BY hour
    `,
    [daySpan]
  );

  const live = await getLiveActivity();

  live.forEach((session) => {
    const existing = dbHours.find((row) => row.hour === session.hour);
    if (existing) {
      existing.plays += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      dbHours.push({ hour: session.hour, plays: 1, duration: session.PlaybackDuration });
    }
  });

  return dbHours.sort((a, b) => a.hour - b.hour);
}

async function getPopularDayOfWeek({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  const dbDays = await rows(
    `
    SELECT
      EXTRACT(DOW FROM ${ACTIVITY_TS})::int AS day,
      COUNT(*)::int AS plays,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY day
    ORDER BY day
    `,
    [daySpan]
  );
  const live = await getLiveActivity();

  live.forEach((session) => {
    const existing = dbDays.find((row) => row.day === session.day);
    if (existing) {
      existing.plays += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      dbDays.push({ day: session.day, plays: 1, duration: session.PlaybackDuration });
    }
  });

  return dbDays.sort((a, b) => a.day - b.day);
}

async function getMostUsedPlaybackMethod({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  const dbMethods = await rows(
    `
    SELECT
      COALESCE(NULLIF("PlayMethod", ''), 'Unknown') AS method,
      COUNT(*)::int AS count,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    GROUP BY method
    ORDER BY count DESC
    `,
    [daySpan]
  );
  const live = await getLiveActivity();

  live.forEach((session) => {
    const existing = dbMethods.find((row) => row.method === session.PlayMethod);
    if (existing) {
      existing.count += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      dbMethods.push({ method: session.PlayMethod, count: 1, duration: session.PlaybackDuration });
    }
  });

  return dbMethods.sort((a, b) => b.count - a.count);
}

async function getMostUsedClients({ days = 30 } = {}) {
  const daySpan = parseDays(days) - 1;

  const dbClients = await rows(
    `
    SELECT
      "Client" AS client,
      COUNT(*)::int AS count,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE ${ACTIVITY_TS} >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
      AND "Client" IS NOT NULL AND "Client" <> ''
    GROUP BY "Client"
    ORDER BY count DESC
    LIMIT 10
    `,
    [daySpan]
  );
  const live = await getLiveActivity();

  live.forEach((session) => {
    const existing = dbClients.find((row) => row.client === session.Client);
    if (existing) {
      existing.count += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      dbClients.push({ client: session.Client, count: 1, duration: session.PlaybackDuration });
    }
  });

  return dbClients.sort((a, b) => b.count - a.count);
}

async function getUserStats(userId) {
  if (userId) {
    const user = await one(
      `
      SELECT
        u."Id" AS "UserId",
        u."Name" AS "UserName",
        COUNT(a."Id")::int AS "TotalPlays",
        FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
        COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
        NULL::text AS "FavoriteGenre"
      FROM jf_users u
      LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
      WHERE u."Id" = $1
      GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
      `,
      [userId]
    );
    const live = (await getLiveActivity()).filter((session) => session.UserId === userId);
    if (!user) return null;
    return {
      ...user,
      TotalPlays: (user.TotalPlays ?? 0) + live.length,
      TotalWatchTime: (user.TotalWatchTime ?? 0) + live.reduce((sum, item) => sum + item.PlaybackDuration, 0),
    };
  }

  const users = await rows(
    `
    SELECT
      u."Id" AS "UserId",
      u."Name" AS "UserName",
      COUNT(a."Id")::int AS "TotalPlays",
      FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
      COALESCE(MAX(a."ActivityDateInserted"), u."LastActivityDate", u."LastLoginDate") AS "LastSeen",
      NULL::text AS "FavoriteGenre"
    FROM jf_users u
    LEFT JOIN jf_playback_activity a ON a."UserId" = u."Id"
    GROUP BY u."Id", u."Name", u."LastActivityDate", u."LastLoginDate"
    ORDER BY "TotalPlays" DESC
    `
  );
  const live = await getLiveActivity();

  live.forEach((session) => {
    const existing = users.find((user) => user.UserId === session.UserId);
    if (existing) {
      existing.TotalPlays += 1;
      existing.TotalWatchTime += session.PlaybackDuration;
    } else {
      users.push({
        UserId: session.UserId,
        UserName: session.UserName,
        TotalPlays: 1,
        TotalWatchTime: session.PlaybackDuration,
        LastSeen: new Date().toISOString(),
        FavoriteGenre: null,
      });
    }
  });

  return users.sort((a, b) => b.TotalPlays - a.TotalPlays);
}

async function getUserActivity(userId) {
  const activity = await rows(
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
      ("PlaybackDuration" * 10000000)::bigint AS "PlayDuration",
      "ActivityDateInserted",
      "RemoteEndPoint"
    FROM jf_playback_activity
    WHERE "UserId" = $1
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 200
    `,
    [userId]
  );
  const live = (await getLiveActivity()).filter((session) => session.UserId === userId);
  const liveActivity = live.map((session) => ({
    Id: `live-${session.UserId}-${session.NowPlayingItemId}`,
    UserId: session.UserId,
    UserName: session.UserName,
    ItemId: session.NowPlayingItemId,
    NowPlayingItemName: session.NowPlayingItemName,
    SeriesName: null,
    SeasonId: null,
    EpisodeId: session.EpisodeId,
    Client: session.Client,
    DeviceName: null,
    DeviceId: null,
    ApplicationVersion: null,
    PlayMethod: session.PlayMethod,
    IsPaused: false,
    IsActive: true,
    PlayDuration: session.PlaybackDuration * 600000000,
    ActivityDateInserted: new Date().toISOString(),
    RemoteEndPoint: null,
  }));

  return [...liveActivity, ...activity];
}

async function getAllUserActivity() {
  const activity = await rows(
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
      ("PlaybackDuration" * 10000000)::bigint AS "PlayDuration",
      "ActivityDateInserted",
      "RemoteEndPoint"
    FROM jf_playback_activity
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 500
    `
  );

  const live = await getLiveActivity();
  const liveActivity = live.map((session) => ({
    Id: `live-${session.UserId}-${session.NowPlayingItemId}`,
    UserId: session.UserId,
    UserName: session.UserName,
    ItemId: session.NowPlayingItemId,
    NowPlayingItemName: session.NowPlayingItemName,
    SeriesName: null,
    SeasonId: null,
    EpisodeId: session.EpisodeId,
    Client: session.Client,
    DeviceName: null,
    DeviceId: null,
    ApplicationVersion: null,
    PlayMethod: session.PlayMethod,
    IsPaused: false,
    IsActive: true,
    PlayDuration: session.PlaybackDuration * 600000000,
    ActivityDateInserted: new Date().toISOString(),
    RemoteEndPoint: null,
  }));

  return [...liveActivity, ...activity];
}

async function getUserActivityByDate(userId) {
  const activity = await rows(
    `
    SELECT
      TO_CHAR((${ACTIVITY_TS})::date, 'YYYY-MM-DD') AS date,
      COUNT(*)::int AS count,
      ${PLAYBACK_MINUTES_SUM}::int AS duration
    FROM jf_playback_activity
    WHERE "UserId" = $1
    GROUP BY (${ACTIVITY_TS})::date
    ORDER BY date
    `,
    [userId]
  );
  const live = (await getLiveActivity()).filter((session) => session.UserId === userId);

  live.forEach((session) => {
    const existing = activity.find((point) => point.date === session.date);
    if (existing) {
      existing.count += 1;
      existing.duration += session.PlaybackDuration;
    } else {
      activity.push({ date: session.date, count: 1, duration: session.PlaybackDuration });
    }
  });

  return activity.sort((a, b) => a.date.localeCompare(b.date));
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
    where.push(`a."UserId" = $${values.length}`);
  }
  if (libraryId) {
    values.push(libraryId);
    where.push(`i."ParentId" = $${values.length}`);
  }

  const dbGenres = await rows(
    `
    SELECT
      genre AS "Genre",
      COUNT(DISTINCT COALESCE(a."EpisodeId", a."NowPlayingItemId"))::int AS "Count",
      COUNT(*)::int AS "PlayCount"
    FROM jf_playback_activity a
    LEFT JOIN jf_library_items i
      ON a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id"
    CROSS JOIN LATERAL jsonb_array_elements_text(
      CASE
        WHEN jsonb_array_length(COALESCE(i."Genres", '[]'::jsonb)) = 0 THEN '["No Genre"]'::jsonb
        ELSE i."Genres"
      END
    ) AS genre
    ${where.length ? `WHERE ${where.join(" AND ")}` : ""}
    GROUP BY genre
    ORDER BY "PlayCount" DESC, genre ASC
    LIMIT 100
    `,
    values
  );

  const live = (await getLiveActivity()).filter((session) => {
    if (userId && session.UserId !== userId) return false;
    if (libraryId && session.LibraryId !== libraryId) return false;
    return true;
  });

  live.forEach((session) => {
    const genres = Array.isArray(session.Genres) && session.Genres.length > 0 ? session.Genres : ["No Genre"];
    genres.forEach((genre) => {
      const existing = dbGenres.find((row) => row.Genre === genre);
      if (existing) {
        existing.Count += 1;
        existing.PlayCount += 1;
      } else {
        dbGenres.push({ Genre: genre, Count: 1, PlayCount: 1 });
      }
    });
  });

  return dbGenres.sort((a, b) => b.PlayCount - a.PlayCount || a.Genre.localeCompare(b.Genre));
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
      FLOOR(COALESCE(SUM(a."PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime"
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
  const live = (await getLiveActivity()).filter((session) => session.LibraryId === libraryId);
  const liveTop = live[0]
    ? {
        Id: live[0].NowPlayingItemId,
        Name: live[0].NowPlayingItemName,
        Type: live[0].Type,
        PlayCount: 1,
      }
    : null;

  const mostPlayedItem = topItem
    ? {
        Id: topItem.Id,
        Name: topItem.Name,
        Type: topItem.Type,
        PlayCount: topItem.PlayCount + live.filter((session) => session.NowPlayingItemId === topItem.Id).length,
      }
    : liveTop ?? undefined;

  return {
    Name: stats.Name,
    TotalItems: stats.TotalItems ?? 0,
    TotalPlayCount: (stats.TotalPlayCount ?? 0) + live.length,
    TotalWatchTime: (stats.TotalWatchTime ?? 0) + live.reduce((sum, item) => sum + item.PlaybackDuration, 0),
    MostPlayedItem: mostPlayedItem,
  };
}

async function getLibraryItems(libraryId) {
  const items = await rows(
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
  const live = (await getLiveActivity()).filter((session) => session.LibraryId === libraryId);

  live.forEach((session) => {
    const existing = items.find((item) => item.Id === session.NowPlayingItemId || item.Id === session.EpisodeId);
    if (existing) existing.PlayCount += 1;
  });

  return items;
}

async function getItemDetails(itemId) {
  const item = await one(
    `
    SELECT
      i."Id",
      i."Name",
      i."Type",
      i."ProductionYear",
      i."CommunityRating",
      i."PremiereDate",
      i."DateCreated",
      i."RunTimeTicks",
      i."Genres",
      i."ParentId",
      info."Size",
      info."Path",
      info."Bitrate"
    FROM jf_library_items i
    LEFT JOIN jf_item_info info ON info."Id" = i."Id"
    WHERE i."Id" = $1
    LIMIT 1
    `,
    [itemId]
  );

  if (!item) return null;

  const history = await rows(
    `
    SELECT
      "Id",
      "UserId",
      "UserName",
      "Client",
      "DeviceName",
      "PlayMethod",
      ${PLAYBACK_MINUTES}::int AS "PlaybackDuration",
      "ActivityDateInserted",
      "RemoteEndPoint",
      false AS "IsActive"
    FROM jf_playback_activity
    WHERE "NowPlayingItemId" = $1 OR "EpisodeId" = $1
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 200
    `,
    [itemId]
  );

  const live = (await getLiveActivity()).filter((session) => session.NowPlayingItemId === itemId || session.EpisodeId === itemId);
  const liveHistory = live.map((session) => ({
    Id: `live-${session.UserId}-${session.NowPlayingItemId}`,
    UserId: session.UserId,
    UserName: session.UserName,
    Client: session.Client,
    DeviceName: null,
    PlayMethod: session.PlayMethod,
    PlaybackDuration: session.PlaybackDuration,
    ActivityDateInserted: new Date().toISOString(),
    RemoteEndPoint: null,
    IsActive: true,
  }));

  const allHistory = [...liveHistory, ...history];
  const users = allHistory.reduce((acc, row) => {
    let user = acc.find((entry) => entry.UserId === row.UserId);
    if (!user) {
      user = {
        UserId: row.UserId,
        UserName: row.UserName,
        PlayCount: 0,
        TotalWatchTime: 0,
        LastWatched: null,
        IsActive: false,
      };
      acc.push(user);
    }

    user.PlayCount += 1;
    user.TotalWatchTime += row.PlaybackDuration || 0;
    user.IsActive = user.IsActive || row.IsActive;
    if (!user.LastWatched || new Date(row.ActivityDateInserted) > new Date(user.LastWatched)) {
      user.LastWatched = row.ActivityDateInserted;
    }
    return acc;
  }, []);

  return {
    item,
    stats: {
      TotalPlays: allHistory.length,
      TotalWatchTime: allHistory.reduce((sum, row) => sum + (row.PlaybackDuration || 0), 0),
      UniqueUsers: users.length,
      LastWatched: allHistory[0]?.ActivityDateInserted ?? null,
      IsActive: live.length > 0,
    },
    users: users.sort((a, b) => b.PlayCount - a.PlayCount),
    history: allHistory,
  };
}

async function getActivityTimeline() {
  const timeline = await rows(
    `
    SELECT
      a."Id",
      a."UserId",
      a."UserName",
      a."NowPlayingItemId" AS "ItemId",
      a."NowPlayingItemName" AS "ItemName",
      (a."ActivityDateInserted"::timestamptz - (COALESCE(a."PlaybackDuration", 0) * INTERVAL '1 second'))::text AS "StartTime",
      a."ActivityDateInserted" AS "EndTime",
      COALESCE(a."PlaybackDuration", 0)::int AS "Duration",
      a."Client",
      a."PlayMethod"
    FROM jf_playback_activity a
    ORDER BY ${ACTIVITY_TS} DESC
    LIMIT 500
    `
  );
  const live = await getLiveActivity();
  const now = Date.now();
  const liveTimeline = live.map((session) => {
    const durationSeconds = session.PlaybackDuration * 60;
    return {
      Id: `live-${session.UserId}-${session.NowPlayingItemId}`,
      UserId: session.UserId,
      UserName: session.UserName,
      ItemId: session.NowPlayingItemId,
      ItemName: session.NowPlayingItemName,
      StartTime: new Date(now - durationSeconds * 1000).toISOString(),
      EndTime: new Date(now).toISOString(),
      Duration: durationSeconds,
      Client: session.Client,
      PlayMethod: session.PlayMethod,
    };
  });

  return [...liveTimeline, ...timeline];
}

async function getMostViewedLibraries({ days = 30 } = {}) {
  return rows(
    `
    SELECT l."Name", COUNT(a."Id")::int AS "Count"
    FROM jf_libraries l
    LEFT JOIN jf_library_items i ON i."ParentId" = l."Id" AND i.archived = false
    LEFT JOIN jf_playback_activity a
      ON (a."NowPlayingItemId" = i."Id" OR a."EpisodeId" = i."Id")
      AND a."ActivityDateInserted"::timestamptz >= CURRENT_DATE - MAKE_INTERVAL(days => $1)
    WHERE l.archived = false
    GROUP BY l."Id", l."Name"
    ORDER BY "Count" DESC
    `,
    [parseDays(days) - 1]
  );
}

async function getLibraryLastPlayed({ libraryId } = {}) {
  if (!libraryId) return [];
  return rows(
    `
    SELECT
      a."Id", a."UserId", a."UserName", a."NowPlayingItemName", a."SeriesName",
      a."Client", a."PlayMethod", a."ActivityDateInserted",
      FLOOR(COALESCE(a."PlaybackDuration", 0) / 60.0)::int AS "PlaybackDuration",
      i."Type" AS "ItemType"
    FROM jf_playback_activity a
    JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
    WHERE i."ParentId" = $1
    ORDER BY a."ActivityDateInserted"::timestamptz DESC
    LIMIT 15
    `,
    [libraryId]
  );
}

async function getLibraryTracks(libraryId) {
  return rows(
    `
    SELECT
      i."Id",
      i."Name",
      i."AlbumArtist" AS "Artist",
      i."Album",
      i."AlbumId",
      i."IndexNumber",
      i."RunTimeTicks",
      COALESCE(pc.play_count, 0)::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
      FROM jf_playback_activity
      GROUP BY "NowPlayingItemId"
    ) pc ON pc."NowPlayingItemId" = i."Id"
    WHERE i."ParentId" = $1
      AND i."Type" = 'Audio'
      AND i.archived = false
    ORDER BY
      COALESCE(i."AlbumArtist", '') ASC,
      COALESCE(i."Album", '') ASC,
      i."IndexNumber" NULLS LAST,
      i."Name" ASC
    LIMIT 5000
    `,
    [libraryId]
  );
}

async function getLibraryAlbums(libraryId) {
  return rows(
    `
    SELECT
      i."AlbumId" AS "Id",
      i."Album" AS "Name",
      i."AlbumArtist" AS "Artist",
      MAX(i."ProductionYear") AS "ProductionYear",
      COUNT(i."Id")::int AS "TrackCount",
      COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
      FROM jf_playback_activity
      GROUP BY "NowPlayingItemId"
    ) pc ON pc."NowPlayingItemId" = i."Id"
    WHERE i."ParentId" = $1
      AND i.archived = false
      AND i."Type" = 'Audio'
      AND i."AlbumId" IS NOT NULL
    GROUP BY i."AlbumId", i."Album", i."AlbumArtist"
    ORDER BY i."Album"
    `,
    [libraryId]
  );
}

async function getLibraryArtists(libraryId) {
  return rows(
    `
    SELECT
      i."AlbumArtist" AS "Name",
      COUNT(DISTINCT i."AlbumId")::int AS "AlbumCount",
      COUNT(i."Id")::int AS "TrackCount",
      COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
      FROM jf_playback_activity
      GROUP BY "NowPlayingItemId"
    ) pc ON pc."NowPlayingItemId" = i."Id"
    WHERE i."ParentId" = $1
      AND i.archived = false
      AND i."Type" = 'Audio'
      AND i."AlbumArtist" IS NOT NULL
    GROUP BY i."AlbumArtist"
    ORDER BY i."AlbumArtist"
    `,
    [libraryId]
  );
}

async function getArtistAlbums(libraryId, artistName) {
  return rows(
    `
    SELECT
      i."AlbumId" AS "Id",
      i."Album" AS "Name",
      i."AlbumArtist" AS "Artist",
      MAX(i."ProductionYear") AS "ProductionYear",
      COUNT(i."Id")::int AS "TrackCount",
      COALESCE(SUM(pc.play_count), 0)::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
      FROM jf_playback_activity
      GROUP BY "NowPlayingItemId"
    ) pc ON pc."NowPlayingItemId" = i."Id"
    WHERE i."ParentId" = $1
      AND i.archived = false
      AND i."Type" = 'Audio'
      AND i."AlbumArtist" = $2
      AND i."AlbumId" IS NOT NULL
    GROUP BY i."AlbumId", i."Album", i."AlbumArtist"
    ORDER BY i."Album"
    `,
    [libraryId, artistName]
  );
}

async function getAlbumTracks(albumId) {
  return rows(
    `
    SELECT
      i."Id",
      i."Name",
      i."IndexNumber",
      i."RunTimeTicks",
      i."AlbumId",
      i."Album" AS "AlbumName",
      i."AlbumArtist" AS "Artist",
      COALESCE(pc.play_count, 0)::int AS "PlayCount"
    FROM jf_library_items i
    LEFT JOIN (
      SELECT "NowPlayingItemId", COUNT(*)::int AS play_count
      FROM jf_playback_activity
      GROUP BY "NowPlayingItemId"
    ) pc ON pc."NowPlayingItemId" = i."Id"
    WHERE i."AlbumId" = $1
      AND i."Type" = 'Audio'
      AND i.archived = false
    ORDER BY i."IndexNumber" NULLS LAST, i."Name"
    `,
    [albumId]
  );
}

async function getGlobalUserStats({ userId, hours = 24 } = {}) {
  if (!userId) return null;
  return one(
    `
    SELECT
      COUNT(*)::int AS "TotalPlays",
      FLOOR(COALESCE(SUM("PlaybackDuration"), 0) / 60.0)::int AS "TotalWatchTime",
      COUNT(DISTINCT "NowPlayingItemId")::int AS "UniqueItems"
    FROM jf_playback_activity
    WHERE "UserId" = $1
      AND "ActivityDateInserted"::timestamptz >= NOW() - MAKE_INTERVAL(hours => $2)
    `,
    [userId, parseInt(hours, 10) || 24]
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
  getAllUserActivity,
  getUserActivity,
  getUserActivityByDate,
  getGenreStats,
  getLibraries,
  getLibraryStats,
  getLibraryItems,
  getItemDetails,
  getActivityTimeline,
  getMostViewedLibraries,
  getLibraryLastPlayed,
  getGlobalUserStats,
  getLibraryTracks,
  getLibraryAlbums,
  getLibraryArtists,
  getArtistAlbums,
  getAlbumTracks,
};
