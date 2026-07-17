package main

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/assets"
	"github.com/Jellystics/Jellystics/internal/config"
	"github.com/Jellystics/Jellystics/internal/database"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/migrations"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/router"
	"github.com/Jellystics/Jellystics/internal/service"
	authsvc "github.com/Jellystics/Jellystics/internal/service/auth"
	"github.com/Jellystics/Jellystics/internal/service/scheduler"
	statssvc "github.com/Jellystics/Jellystics/internal/service/stats"
	syncsvc "github.com/Jellystics/Jellystics/internal/service/sync"
	tasksvc "github.com/Jellystics/Jellystics/internal/service/task"
	webhooksvc "github.com/Jellystics/Jellystics/internal/service/webhook"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	db, err := database.Connect(cfg.DBUrl)
	if err != nil {
		log.Fatalf("database: %v", err)
	}

	if err := database.Migrate(db, migrations.SQL, "sql"); err != nil {
		log.Fatalf("migrate: %v", err)
	}

	hub := ws.NewHub()
	jfClient := jellyfin.NewClient(cfg.JFHost, cfg.JFApiKey)
	repos := repository.New(db)

	syncSvc := syncsvc.New(repos, jfClient, hub)
	taskSvc := tasksvc.New(repos, hub)

	webhookSvc := webhooksvc.New(repos)
	svcs := &service.Container{
		Auth:    authsvc.New(repos, jfClient, cfg),
		Sync:    syncSvc,
		Stats:   statssvc.New(repos),
		Task:    taskSvc,
		Webhook: webhookSvc,
	}

	taskSvc.SetFireFunc(func(ctx context.Context, eventType string, data map[string]any) {
		webhookSvc.Fire(ctx, eventType, data)
	})

	// Cron scheduler: dispatches tasks based on cron expressions stored in app_config.
	dispatch := func(ctx context.Context, name string) error {
		return taskSvc.Run(ctx, name, func(ctx context.Context) error {
			switch name {
			case "Full Jellyfin Sync":
				return syncSvc.FullSync(ctx)
			case "Recently Added Sync":
				return syncSvc.SyncRecentlyAdded(ctx)
			case "Backup":
				return handler.RunBackupTask(ctx, repos)
			case "Jellyfin Playback Reporting Plugin Sync":
				return syncSvc.SyncPlaybackPlugin(ctx)
			}
			return nil
		})
	}
	runner := scheduler.NewRunner(taskSvc, dispatch)
	sched := scheduler.New(repos, runner)
	sched.Start(context.Background())

	// Every 10 s: broadcast live sessions, update watchdog, promote finished
	// sessions to jf_playback_activity (ActivityMonitor equivalent).
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		ctx := context.Background()
		for range ticker.C {
			syncSvc.SessionTick(ctx)
		}
	}()

	r := router.New(svcs, repos, hub, db, cfg, sched)

	// Serve embedded SPA (production) — disabled when DISABLE_DASHBOARD=true
	if !cfg.DisableDashboard {
		webFS, err := fs.Sub(assets.Web, "web")
		if err == nil {
			r.NoRoute(spaHandler(webFS))
		}
	} else {
		log.Println("Dashboard disabled (DISABLE_DASHBOARD=true) — running as metrics exporter only")
	}

	addr := fmt.Sprintf(":%s", cfg.Port)
	log.Printf("Jellystics listening on %s", addr)
	if err := r.Run(addr); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// spaHandler serves static files and falls back to index.html for client-side routing.
func spaHandler(staticFS fs.FS) gin.HandlerFunc {
	fileServer := http.FileServer(http.FS(staticFS))
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if len(path) > 0 && path[0] == '/' {
			path = path[1:]
		}
		if _, err := staticFS.Open(path); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	}
}
