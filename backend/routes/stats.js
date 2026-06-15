// api.js
const express = require("express");
const db = require("../db");
const dbHelper = require("../classes/db-helper");

const statsRepo = require("../repositories/stats-repository");

const dayjs = require("dayjs");

function sendRepoResult(res, promise, empty = null) {
  promise
    .then((data) => res.send(data ?? empty))
    .catch((error) => {
      console.log(error);
      res.status(503).json({ error: "Failed to load statistics" });
    });
}

const router = express.Router();

//functions
function countOverlapsPerHour(records) {
  const hourCounts = {};

  records.forEach((record) => {
    const start = dayjs(record.StartTime).subtract(1, "hour");
    const end = dayjs(record.EndTime).add(1, "hour");

    // Iterate through each hour from start to end
    for (let hour = start.clone().startOf("hour"); hour.isBefore(end); hour.add(1, "hour")) {
      const hourKey = hour.format("MMM DD, YY HH:00");
      if (!hourCounts[hourKey]) {
        hourCounts[hourKey] = { Transcodes: 0, DirectPlays: 0 };
      }
      if (record.PlayMethod === "Transcode") {
        hourCounts[hourKey].Transcodes++;
      } else {
        hourCounts[hourKey].DirectPlays++;
      }
    }
  });

  // Convert the hourCounts object to an array of key-value pairs, sort it, and convert it back to an object
  const sortedHourCounts = Object.fromEntries(Object.entries(hourCounts).sort(([keyA], [keyB]) => keyA.localeCompare(keyB)));

  return sortedHourCounts;
}

const sortMap = [
  { field: "UserName", column: "UserName" },
  { field: "RemoteEndPoint", column: "RemoteEndPoint" },
  { field: "NowPlayingItemName", column: "NowPlayingItemName" },
  { field: "Client", column: "Client" },
  { field: "DeviceName", column: "DeviceName" },
  { field: "ActivityDateInserted", column: "ActivityDateInserted" },
  { field: "PlaybackDuration", column: "PlaybackDuration" },
  { field: "PlayMethod", column: "PlayMethod" },
];

const filterFields = [
  { field: "Id", column: "Id", isColumn: true },
  { field: "IsPaused", column: "IsPaused", isColumn: true },
  { field: "UserId", column: "UserId", isColumn: true },
  { field: "UserName", column: `LOWER(a."UserName")` },
  { field: "Client", column: `LOWER(a."Client")` },
  { field: "DeviceName", column: `LOWER(a."DeviceName")` },
  { field: "DeviceId", column: "DeviceId", isColumn: true },
  { field: "ApplicationVersion", column: "ApplicationVersion", isColumn: true },
  { field: "NowPlayingItemId", column: "NowPlayingItemId", isColumn: true },
  { field: "NowPlayingItemName", column: `LOWER(a."NowPlayingItemName")` },
  { field: "SeasonId", column: "SeasonId", isColumn: true },
  { field: "SeriesName", column: `LOWER(a."SeriesName")` },
  { field: "EpisodeId", column: "EpisodeId", isColumn: true },
  { field: "PlaybackDuration", column: "PlaybackDuration", isColumn: true },
  { field: "ActivityDateInserted", column: "ActivityDateInserted", isColumn: true },
  { field: "PlayMethod", column: `LOWER(a."PlayMethod")` },
  { field: "OriginalContainer", column: "OriginalContainer", isColumn: true },
  { field: "RemoteEndPoint", column: "RemoteEndPoint", isColumn: true },
  { field: "ServerId", column: "ServerId", isColumn: true },
  { field: "imported", column: "imported", isColumn: true },
];

//endpoints

router.get("/getLibraryOverview", async (req, res) => {
  try {
    const { rows } = await db.query("SELECT * FROM jf_library_count_view");
    res.send(rows);
  } catch (error) {
    res.status(503);
    res.send(error);
  }
});

router.post("/getMostViewedLibraries", async (req, res) => {
  const { days } = req.body;
  sendRepoResult(res, statsRepo.getMostViewedLibraries({ days }), []);
});

router.get("/getPlaybackActivity", async (req, res) => {
  const { size = 50, page = 1, search, sort = "ActivityDateInserted", desc = true, filters } = req.query;
  let filtersArray = [];
  if (filters) {
    try {
      filtersArray = JSON.parse(filters);
    } catch (error) {
      return res.status(400).json({
        error: "Invalid filters parameter",
        example: [
          { field: "UserName", value: "User" },
          { field: "Client", in: "Android TV,Web" },
          { field: "PlaybackDuration", min: 1000, max: 5000 },
          { field: "PlayMethod", value: "DirectPlay" },
          { field: "ActivityDateInserted", min: "2025-01-01", max: "2025-12-31" },
          { field: "IsPaused", value: false },
        ],
        allowed_fields: [
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
          "SeasonId",
          "SeriesName",
          "EpisodeId",
          "PlaybackDuration",
          "ActivityDateInserted",
          "PlayMethod",
          "OriginalContainer",
          "RemoteEndPoint",
          "ServerId",
          "imported",
        ],
      });
    }
  }

  const sortField = sortMap.find((item) => item.field === sort)?.column || "ActivityDateInserted";
  const values = [];
  try {
    const query = {
      select: ["*"],
      table: "jf_playback_activity",
      alias: "a",
      order_by: sortField,
      sort_order: desc ? "desc" : "asc",
      pageNumber: page,
      pageSize: size,
    };

    if (search && search.length > 0) {
      query.where = [
        {
          field: `LOWER(
          CASE 
            WHEN a."SeriesName" is null THEN a."NowPlayingItemName"
            ELSE a."SeriesName"
          END 
          )`,
          operator: "LIKE",
          value: `$${values.length + 1}`,
        },
      ];

      values.push(`%${search.toLowerCase()}%`);
    }

    query.values = values;
    dbHelper.buildFilterList(query, filtersArray, filterFields);

    const result = await dbHelper.query(query);
    const response = { current_page: page, pages: result.pages, size: size, sort: sort, desc: desc, results: result.results };
    if (search && search.length > 0) {
      response.search = search;
    }

    if (filtersArray.length > 0) {
      response.filters = filtersArray;
    }
    res.send(response);
  } catch (error) {
    res.status(503);
    res.send(error);
  }
});

router.get("/getAllUserActivity", async (req, res) => {
  sendRepoResult(res, statsRepo.getAllUserActivity(), []);
});

router.post("/getUserLastPlayed", async (req, res) => {
  try {
    const { userid } = req.body;
    if (!userid) return res.status(400).json({ error: 'userid is required' });
    const { rows = [] } = await db.query(
      `SELECT * FROM jf_playback_activity WHERE "UserId" = $1 ORDER BY "ActivityDateInserted"::timestamptz DESC LIMIT 15`,
      [userid]
    );
    res.send(rows);
  } catch (error) {
    console.log(error);
    res.status(503).json({ error: 'Failed to load data' });
  }
});

//Global Stats
router.post("/getGlobalUserStats", async (req, res) => {
  const { hours, userid } = req.body;
  if (!userid) return res.status(400).json({ error: 'userid is required' });
  sendRepoResult(res, statsRepo.getGlobalUserStats({ userId: userid, hours }), null);
});

router.get("/getLibraryCardStats", async (req, res) => {
  try {
    const { rows } = await db.query(
      `select *, now() - js_library_stats_overview."ActivityDateInserted" AS "LastActivity" from js_library_stats_overview`
    );
    res.send(rows);
  } catch (error) {
    res.status(503);
    res.send(error);
  }
});

router.post("/getLibraryCardStats", async (req, res) => {
  try {
    const { libraryid } = req.body;
    if (libraryid === undefined) {
      res.status(503);
      return res.send("Invalid Library Id");
    }

    const { rows } = await db.query(
      `select *, now() - js_library_stats_overview."ActivityDateInserted" AS "LastActivity" from js_library_stats_overview where "Id"=$1`,
      [libraryid]
    );
    res.send(rows[0]);
  } catch (error) {
    console.log(error);
    res.status(503);
    res.send(error);
  }
});

router.get("/getLibraryMetadata", async (req, res) => {
  try {
    const { rows } = await db.query("select * from js_library_metadata");
    res.send(rows);
  } catch (error) {
    res.status(503);
    res.send(error);
  }
});

router.post("/getLibraryItemsWithStats", async (req, res) => {
  const { size = 999999999, page = 1, search, sort = "Date", desc = true } = req.query;
  const { libraryid } = req.body;
  if (libraryid === undefined) {
    res.status(400).send({ error: "Invalid Library Id" });
  }

  const sortMap = [
    { field: "Date", column: "DateCreated" },
    { field: "Views", column: "times_played" },
    { field: "Size", column: "Size" },
    { field: "WatchTime", column: "total_play_time" },
    { field: "Title", column: `REGEXP_REPLACE(a."Name", '^(A |An |The )', '', 'i')` },
  ];

  const sortField = sortMap.find((item) => item.field === sort)?.column || "DateCreated";
  const values = [];
  try {
    const query = {
      select: ["*"],
      table: "js_library_items_with_playcount_playtime",
      alias: "a",
      order_by: sortField,
      sort_order: desc ? "desc" : "asc",
      pageNumber: page,
      pageSize: size,
      where: [
        {
          field: `a."ParentId"`,
          operator: "=",
          value: `$${values.length + 1}`,
        },
      ],
    };

    values.push(libraryid);

    if (search && search.length > 0) {
      query.where.push({
        field: `LOWER(a."Name")`,
        operator: "LIKE",
        value: `$${values.length + 1}`,
      });

      values.push(`%${search.toLowerCase()}%`);
    }

    query.values = values;

    // const { rows } = await db.query(`SELECT * FROM jf_library_items_with_playcount_playtime where "ParentId"=$1`, [libraryid]);
    // res.send(rows);
    const result = await dbHelper.query(query);
    const response = { current_page: page, pages: result.pages, size: size, sort: sort, desc: desc, results: result.results };
    if (search && search.length > 0) {
      response.search = search;
    }

    res.send(response);
  } catch (error) {
    console.log(error);
  }
});

router.post("/getLibraryItemsPlayMethodStats", async (req, res) => {
  try {
    let { libraryid, startDate, endDate = dayjs(), hours = 24 } = req.body;

    // Validate startDate and endDate using dayjs
    if (
      startDate !== undefined &&
      (!dayjs(startDate, "YYYY-MM-DDTHH:mm:ss.SSSZ", true).isValid() ||
        !dayjs(endDate, "YYYY-MM-DDTHH:mm:ss.SSSZ", true).isValid())
    ) {
      return res.status(400).send({ error: "Invalid date format" });
    }

    if (hours < 1) {
      return res.status(400).send({ error: "Hours cannot be less than 1" });
    }

    if (libraryid === undefined) {
      return res.status(400).send({ error: "Invalid Library Id" });
    }

    if (startDate === undefined) {
      startDate = dayjs(endDate).subtract(hours, "hour").format("YYYY-MM-DD HH:mm:ss");
    }

    const { rows } = await db.query(
      `select a.*,i."ParentId"
      from jf_playback_activity a
	    left
	    join jf_library_episodes e
	    on a."EpisodeId"=e."EpisodeId"
	    join jf_library_items i
	    on i."Id"=a."NowPlayingItemId" or e."SeriesId"=i."Id"
      where i."ParentId"=$1
      and a."ActivityDateInserted" BETWEEN $2 AND $3
      order by a."ActivityDateInserted" desc;
      `,
      [libraryid, startDate, endDate]
    );

    const stats = rows.map((item) => {
      return {
        Id: item.NowPlayingItemId,
        UserId: item.UserId,
        UserName: item.UserName,
        Client: item.Client,
        DeviceName: item.DeviceName,
        NowPlayingItemName: item.NowPlayingItemName,
        EpisodeId: item.EpisodeId || null,
        SeasonId: item.SeasonId || null,
        StartTime: dayjs(item.ActivityDateInserted).subtract(item.PlaybackDuration, "seconds").format("YYYY-MM-DD HH:mm:ss"),
        EndTime: dayjs(item.ActivityDateInserted).format("YYYY-MM-DD HH:mm:ss"),
        PlaybackDuration: item.PlaybackDuration,
        PlayMethod: item.PlayMethod,
        TranscodedVideo: item.TranscodingInfo?.IsVideoDirect || false,
        TranscodedAudio: item.TranscodingInfo?.IsAudioDirect || false,
        ParentId: item.ParentId,
      };
    });

    let countedstats = countOverlapsPerHour(stats);

    let hoursRes = {
      types: [
        { Id: "Transcodes", Name: "Transcodes" },
        { Id: "DirectPlays", Name: "DirectPlays" },
      ],

      stats: Object.keys(countedstats).map((key) => {
        return {
          Key: key,
          Transcodes: countedstats[key].Transcodes,
          DirectPlays: countedstats[key].DirectPlays,
        };
      }),
    };
    res.send(hoursRes);
  } catch (error) {
    console.log(error);
    res.send(error);
  }
});

router.post("/getPlaybackMethodStats", async (req, res) => {
  try {
    const { days = 30 } = req.body;

    if (days < 0) {
      res.status(503);
      return res.send("Days cannot be less than 0");
    }

    const { rows } = await db.query(
      `select a."PlayMethod" "Name",count(a."PlayMethod") "Count"
      from jf_playback_activity a
      WHERE a."ActivityDateInserted" BETWEEN CURRENT_DATE - MAKE_INTERVAL(days => $1) AND NOW()
		  Group by a."PlayMethod"
      ORDER BY (count(*)) DESC;
      `,
      [days - 1]
    );

    res.send(rows);
  } catch (error) {
    console.log(error);
    res.send(error);
  }
});

router.post("/getLibraryLastPlayed", async (req, res) => {
  const { libraryid } = req.body;
  if (!libraryid) return res.status(400).json({ error: 'libraryid is required' });
  sendRepoResult(res, statsRepo.getLibraryLastPlayed({ libraryId: libraryid }), []);
});

router.get("/getViewsByLibraryType", async (req, res) => {
  try {
    const { days = 30 } = req.query;

    const { rows = [] } = await db.query(
      `
      SELECT COALESCE(i."Type", 'Other') AS type, COUNT(a."NowPlayingItemId") AS count
      FROM jf_playback_activity a LEFT JOIN jf_library_items i ON i."Id" = a."NowPlayingItemId"
      WHERE a."ActivityDateInserted"::timestamptz >= NOW() - MAKE_INTERVAL(days => $1::integer)
      GROUP BY i."Type"
    `,
      [parseInt(days, 10) || 30]
    );

    const supportedTypes = new Set(["Audio", "Movie", "Series", "Other"]);
    /** @type {Map<string, number>} */
    const reorganizedData = new Map();

    rows.forEach((item) => {
      const { type, count } = item;

      if (!supportedTypes.has(type)) return;
      reorganizedData.set(type, count);
    });

    supportedTypes.forEach((type) => {
      if (reorganizedData.has(type)) return;
      reorganizedData.set(type, 0);
    });

    res.send(Object.fromEntries(reorganizedData));
  } catch (error) {
    console.log(error);
    res.status(503);
    res.send(error);
  }
});

router.get("/getGenreUserStats", async (req, res) => {
  try {
    const { size = 50, page = 1, userid } = req.query;

    if (userid === undefined) {
      res.status(400);
      res.send("No User ID provided");
      return;
    }

    const values = [];
    const query = {
      select: ["COALESCE(g.genre, 'No Genre') AS genre", `SUM(a."PlaybackDuration") AS duration`, "COUNT(*) AS plays"],
      table: "jf_playback_activity_with_metadata",
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
                    SELECT 
                      jsonb_array_elements_text(
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

      where: [[{ column: "a.UserId", operator: "=", value: `$${values.length + 1}` }]],
      group_by: [`COALESCE(g.genre, 'No Genre')`],
      order_by: "genre",
      sort_order: "asc",
      pageNumber: page,
      pageSize: size,
    };

    values.push(userid);

    query.values = values;

    const result = await dbHelper.query(query);

    const response = { current_page: page, pages: result.pages, size: size, results: result.results };

    res.send(response);
  } catch (error) {
    console.log(error);
    res.status(503);
    res.send(error);
  }
});

router.get("/getGenreLibraryStats", async (req, res) => {
  try {
    const { size = 50, page = 1, libraryid } = req.query;

    if (libraryid === undefined) {
      res.status(400);
      res.send("No Library ID provided");
      return;
    }

    const values = [];
    const query = {
      select: ["COALESCE(g.genre, 'No Genre') AS genre", `SUM(a."PlaybackDuration") AS duration`, "COUNT(*) AS plays"],
      table: "jf_playback_activity_with_metadata",
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
                    SELECT 
                      jsonb_array_elements_text(
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

      where: [[{ column: "a.ParentId", operator: "=", value: `$${values.length + 1}` }]],
      group_by: [`COALESCE(g.genre, 'No Genre')`],
      order_by: "genre",
      sort_order: "asc",
      pageNumber: page,
      pageSize: size,
    };

    values.push(libraryid);

    query.values = values;

    const result = await dbHelper.query(query);

    const response = { current_page: page, pages: result.pages, size: size, results: result.results };

    res.send(response);
  } catch (error) {
    console.log(error);
    res.status(503);
    res.send(error);
  }
});

// Frontend-compatible GET endpoints (Jellystics React UI)

router.get("/getGlobalStats", (req, res) => {
  sendRepoResult(res, statsRepo.getGlobalStats(), {
    TotalPlays: 0,
    TotalWatchTime: 0,
    ActiveUsers: 0,
    TotalUsers: 0,
    TotalLibraries: 0,
    TotalItems: 0,
  });
});

router.get("/getMostPlayedItems", (req, res) => {
  sendRepoResult(res, statsRepo.getMostPlayedItems(req.query), []);
});

router.get("/getMostActiveUsers", (req, res) => {
  sendRepoResult(res, statsRepo.getMostActiveUsers(req.query), []);
});

router.get("/getWatchStatisticsOverTime", (req, res) => {
  sendRepoResult(res, statsRepo.getWatchStatisticsOverTime(req.query), []);
});

router.get("/getPopularHourOfDay", (req, res) => {
  sendRepoResult(res, statsRepo.getPopularHourOfDay(req.query), []);
});

router.get("/getPopularDayOfWeek", (req, res) => {
  sendRepoResult(res, statsRepo.getPopularDayOfWeek(req.query), []);
});

router.get("/getMostUsedPlaybackMethod", (req, res) => {
  sendRepoResult(res, statsRepo.getMostUsedPlaybackMethod(req.query), []);
});

router.get("/getMostUsedClients", (req, res) => {
  sendRepoResult(res, statsRepo.getMostUsedClients(req.query), []);
});

router.get("/getUserStats", (req, res) => {
  const { userId } = req.query;
  sendRepoResult(res, statsRepo.getUserStats(userId), userId ? null : []);
});

router.get("/getUserActivity", (req, res) => {
  const { userId } = req.query;
  if (!userId) return res.status(400).send("userId is required");
  sendRepoResult(res, statsRepo.getUserActivity(userId), []);
});

router.get("/getUserActivityByDate", (req, res) => {
  const { userId } = req.query;
  if (!userId) return res.status(400).send("userId is required");
  sendRepoResult(res, statsRepo.getUserActivityByDate(userId), []);
});

router.get("/getUserGenreStats", (req, res) => {
  const { userId } = req.query;
  if (!userId) return res.status(400).send("userId is required");
  sendRepoResult(res, statsRepo.getGenreStats({ userId }), []);
});

router.get("/getLibraries", (req, res) => {
  sendRepoResult(res, statsRepo.getLibraries(), []);
});

router.get("/getLibraryStats", (req, res) => {
  const { libraryId } = req.query;
  if (!libraryId) return res.status(400).send("libraryId is required");
  sendRepoResult(res, statsRepo.getLibraryStats(libraryId), null);
});

router.get("/getLibraryItems", (req, res) => {
  const { libraryId } = req.query;
  if (!libraryId) return res.status(400).send("libraryId is required");
  sendRepoResult(res, statsRepo.getLibraryItems(libraryId), []);
});

router.get("/getItemDetails", (req, res) => {
  const { itemId } = req.query;
  if (!itemId) return res.status(400).send("itemId is required");
  sendRepoResult(res, statsRepo.getItemDetails(itemId), null);
});

router.get("/getGenreStats", (req, res) => {
  const { libraryId } = req.query;
  if (!libraryId) return res.status(400).send("libraryId is required");
  sendRepoResult(res, statsRepo.getGenreStats({ libraryId }), []);
});

router.get("/getActivityTimeline", (req, res) => {
  sendRepoResult(res, statsRepo.getActivityTimeline(), []);
});

// Handle other routes
router.use((req, res) => {
  res.status(404).send({ error: "Not Found" });
});

module.exports = router;
