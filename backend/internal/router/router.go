package router

import (
	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/middleware"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/Jellystics/Jellystics/internal/service/scheduler"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

func New(svcs *service.Container, repos *repository.Container, hub *ws.Hub, db *gorm.DB, cfg *config.Config, sched *scheduler.Scheduler) *gin.Engine {
	r := gin.Default()

	// CORS
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Authorization,Content-Type")
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}
		c.Next()
	})

	// ── Handlers ──────────────────────────────────────────────────────────────
	frontAuthH := handler.NewFrontendAuthHandler(repos, cfg)
	configH    := handler.NewConfigApiHandler(repos, svcs)
	configH.SetScheduler(sched)
	sessionsH  := handler.NewSessionsApiHandler(repos)
	tasksH     := handler.NewTasksApiHandler(svcs, repos, db)
	proxyH     := handler.NewProxyApiHandler(repos)
	statsH     := handler.NewStatsFrontendHandler(db, repos)
	webhookH   := handler.NewWebhookHandler(svcs)
	syncH      := handler.NewSyncHandler(svcs)
	libH       := handler.NewLibraryHandler(svcs, repos, db)
	userH      := handler.NewUserHandler(repos)
	backupH    := handler.NewBackupHandler(repos, hub, svcs)
	logsH      := handler.NewLogsHandler(repos)
	adminH     := handler.NewAdminApiHandler(repos, db)

	// ── Socket.IO (public) ────────────────────────────────────────────────────
	// Must be registered before NoRoute SPA fallback.
	r.Any("/socket.io/*path", func(c *gin.Context) {
		hub.ServeHTTP(c.Writer, c.Request)
	})

	// ── Image proxy (public, no auth) ─────────────────────────────────────────
	// getSessions/getAdminUsers/getRecentlyAdded are dispatched inside Proxy().
	r.GET("/proxy/*path", proxyH.Proxy)
	// POST proxy route (validateSettings) — must be separate since wildcard is GET-only.
	r.POST("/proxy/validateSettings", proxyH.ValidateSettings)

	// ── Backup download (public – browser download link, no XHR auth header) ──
	r.GET("/backup/files/:name", backupH.Download)

	// ── Auth routes (public) ─────────────────────────────────────────────────
	auth := r.Group("/auth")
	{
		auth.GET("/isConfigured", frontAuthH.IsConfigured)
		auth.POST("/configSetup", frontAuthH.ConfigSetup)
		auth.POST("/check-server", frontAuthH.CheckServer)
		auth.POST("/jellyfin-login", frontAuthH.JellyfinLogin)
	}

	// ── Authenticated routes ──────────────────────────────────────────────────
	authed := r.Group("", middleware.Auth(svcs.Auth))

	// Sessions
	authed.GET("/sessions/current", sessionsH.Current)

	// Stats (all frontend-compatible endpoints)
	stats := authed.Group("/stats")
	{
		stats.GET("/getGlobalStats", statsH.GetGlobalStats)
		stats.GET("/getMostPlayedItems", statsH.GetMostPlayedItems)
		stats.GET("/getMostActiveUsers", statsH.GetMostActiveUsers)
		stats.GET("/getWatchStatisticsOverTime", statsH.GetWatchStatisticsOverTime)
		stats.GET("/getPopularHourOfDay", statsH.GetPopularHourOfDay)
		stats.GET("/getPopularDayOfWeek", statsH.GetPopularDayOfWeek)
		stats.GET("/getMostUsedPlaybackMethod", statsH.GetMostUsedPlaybackMethod)
		stats.GET("/getMostUsedClients", statsH.GetMostUsedClients)
		stats.GET("/getUserStats", statsH.GetUserStats)
		stats.GET("/getUserActivity", statsH.GetUserActivity)
		stats.GET("/getUserActivityByDate", statsH.GetUserActivityByDate)
		stats.GET("/getUserGenreStats", statsH.GetUserGenreStats)
		stats.GET("/getAllUserActivity", statsH.GetAllUserActivity)
		stats.GET("/getLibraries", statsH.GetLibraries)
		stats.GET("/getLibraryStats", statsH.GetLibraryStats)
		stats.GET("/getLibraryItems", statsH.GetLibraryItems)
		stats.GET("/getItemDetails", statsH.GetItemDetails)
		stats.GET("/getGenreStats", statsH.GetGenreStats)
		stats.GET("/getLibraryTracks", statsH.GetLibraryTracks)
		stats.GET("/getLibraryAlbums", statsH.GetLibraryAlbums)
		stats.GET("/getLibraryArtists", statsH.GetLibraryArtists)
		stats.GET("/getArtistAlbums", statsH.GetArtistAlbums)
		stats.GET("/getAlbumTracks", statsH.GetAlbumTracks)
		stats.GET("/getActivityTimeline", statsH.GetActivityTimeline)
		stats.GET("/getLibraryOverview", statsH.GetLibraryOverview)
		stats.GET("/getLibraryCardStats", statsH.GetLibraryCardStatsGET)
		stats.GET("/getLibraryMetadata", statsH.GetLibraryMetadata)
		stats.GET("/getViewsByLibraryType", statsH.GetViewsByLibraryType)
		stats.GET("/getGenreUserStats", statsH.GetGenreUserStats)
		stats.GET("/getGenreLibraryStats", statsH.GetGenreLibraryStats)
		stats.GET("/getPlaybackActivity", statsH.GetPlaybackActivity)
		stats.POST("/getMostViewedLibraries", statsH.GetMostViewedLibraries)
		stats.POST("/getLibraryCardStats", statsH.GetLibraryCardStatsPOST)
		stats.POST("/getLibraryItemsWithStats", statsH.GetLibraryItemsWithStats)
		stats.POST("/getLibraryItemsPlayMethodStats", statsH.GetLibraryItemsPlayMethodStats)
		stats.GET("/getPlaybacksByLibraryOverTime", statsH.GetPlaybacksByLibraryOverTime)
		stats.GET("/getPlaybacksScatter", statsH.GetPlaybacksScatter)
		stats.POST("/getPlaybackMethodStats", statsH.GetPlaybackMethodStats)
		stats.POST("/getUserLastPlayed", statsH.GetUserLastPlayed)
		stats.POST("/getGlobalUserStats", statsH.GetGlobalUserStats)
		stats.POST("/getLibraryLastPlayed", statsH.GetLibraryLastPlayed)
		stats.GET("/getWatchHeatmap", statsH.GetWatchHeatmap)
		stats.GET("/getTimeToWatch", statsH.GetTimeToWatch)
		stats.GET("/getUnwatchedContent", statsH.GetUnwatchedContent)
		stats.GET("/getBingeStats", statsH.GetBingeStats)
		stats.GET("/getCompletionRate", statsH.GetCompletionRate)
		stats.GET("/getViewingDiversity", statsH.GetViewingDiversity)
	}

	// Utils
	utils := authed.Group("/utils")
	{
		utils.POST("/geolocateIp", handler.NewUtilsHandler().GeolocateIP)
	}

	// Webhooks – note: fixed paths (/event-status, /toggle-event/:eventType)
	// must be registered before the /:id wildcard.
	webhooks := authed.Group("/webhooks")
	{
		webhooks.GET("/", webhookH.List)
		webhooks.GET("/event-status", webhookH.EventStatus)
		webhooks.POST("/toggle-event/:eventType", webhookH.ToggleEvent)
		webhooks.GET("/:id", webhookH.Get)
		webhooks.POST("/", webhookH.Create)
		webhooks.PUT("/:id", webhookH.Update)
		webhooks.DELETE("/:id", webhookH.Delete)
		webhooks.POST("/:id/test", webhookH.Test)
		webhooks.POST("/:id/trigger-monthly", webhookH.TriggerMonthly)
	}

	// Sync
	sync := authed.Group("/sync")
	{
		sync.POST("/full", syncH.FullSync)
		sync.POST("/libraries", syncH.SyncLibraries)
		sync.POST("/users", syncH.SyncUsers)
		sync.GET("/status", syncH.Status)
		sync.POST("/importPlaybackBackup", tasksH.ImportBackup)
		sync.POST("/fetchItem", syncH.FetchItem)
	}

	// Backup (list + create + delete + restore; download is public above)
	backup := authed.Group("/backup")
	{
		backup.GET("/files", backupH.List)
		backup.GET("/beginBackup", backupH.Create)
		backup.DELETE("/files/:name", backupH.Delete)
		backup.GET("/restore/:filename", backupH.Restore)
		backup.POST("/upload", backupH.Upload)
	}

	// Logs
	logs := authed.Group("/logs")
	{
		logs.GET("/getLogs", logsH.GetLogs)
	}

	// API
	api := authed.Group("/api")
	{
		// Config
		api.GET("/getconfig", configH.GetConfig)
		api.POST("/setconfig", configH.SetConfig)
		api.GET("/keys", configH.GetKeys)
		api.POST("/keys", configH.CreateKey)
		api.DELETE("/keys", configH.DeleteKey)
		// Config settings
		api.POST("/setExternalUrl", configH.SetExternalUrl)
		api.POST("/setPreferredAdmin", configH.SetPreferredAdmin)
		api.POST("/setRequireLogin", configH.SetRequireLogin)
		api.POST("/updateCredentials", configH.UpdateCredentials)
		api.GET("/TrackedLibraries", configH.TrackedLibraries)
		api.POST("/setExcludedLibraries", configH.SetExcludedLibraries)
		api.GET("/UntrackedUsers", configH.UntrackedUsers)
		api.POST("/setUntrackedUsers", configH.SetUntrackedUsers)
		api.GET("/getTaskSettings", configH.GetTaskSettings)
		api.POST("/setTaskSettings", configH.SetTaskSettings)
		api.GET("/isFirstRun", configH.IsFirstRun)
		api.GET("/getActivityMonitorSettings", configH.GetActivityMonitorSettings)
		api.POST("/setActivityMonitorSettings", configH.SetActivityMonitorSettings)
		api.GET("/CheckForUpdates", configH.CheckForUpdates)
		// Tasks
		api.GET("/getTasks", tasksH.List)
		api.POST("/runTask/:name", tasksH.Run)
		api.GET("/stopTask", configH.StopTask)
		// Security
		// Admin library/item queries
		api.GET("/getLibraries", adminH.GetLibraries)
		api.POST("/getLibrary", adminH.GetLibrary)
		api.POST("/getLibraryItems", adminH.GetLibraryItems)
		api.POST("/getSeasons", adminH.GetSeasons)
		api.POST("/getEpisodes", adminH.GetEpisodes)
		api.POST("/getItemDetails", adminH.GetItemDetails)
		api.POST("/getUserDetails", adminH.GetUserDetails)
		api.GET("/getRecentlyAdded", adminH.GetRecentlyAdded)
		// Purge
		api.DELETE("/item/purge", adminH.PurgeItem)
		api.DELETE("/library/purge", adminH.PurgeLibrary)
		api.DELETE("/libraryItems/purge", adminH.PurgeLibraryItems)
		// History / activity
		api.GET("/getHistory", adminH.GetHistory)
		api.POST("/getLibraryHistory", adminH.GetLibraryHistory)
		api.POST("/getItemHistory", adminH.GetItemHistory)
		api.POST("/getUserHistory", adminH.GetUserHistory)
		api.POST("/deletePlaybackActivity", adminH.DeletePlaybackActivity)
		api.POST("/getActivityTimeLine", adminH.GetActivityTimeLine)
		// Backup table config
		api.GET("/getBackupTables", adminH.GetBackupTables)
		api.POST("/setExcludedBackupTable", adminH.SetExcludedBackupTable)
	}

	// Libraries (for direct library browsing)
	libs := authed.Group("/api/libraries")
	{
		libs.GET("", libH.List)
		libs.GET("/:id", libH.Get)
		libs.GET("/:id/items", libH.Items)
		libs.GET("/:id/seasons", libH.Seasons)
		libs.GET("/:id/episodes", libH.Episodes)
		libs.GET("/:id/artists", libH.Artists)
		libs.GET("/:id/tracks", libH.Tracks)
		libs.GET("/:id/albums/:albumId/tracks", libH.AlbumTracks)
	}
	authed.GET("/api/items/:id", libH.GetItem)

	// Users
	users := authed.Group("/api/users")
	{
		users.GET("", userH.List)
		users.GET("/:id", userH.Get)
	}

	return r
}
