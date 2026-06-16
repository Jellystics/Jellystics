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
	"github.com/Jellystics/Jellystics/internal/jellyfin"
	"github.com/Jellystics/Jellystics/internal/migrations"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/router"
	"github.com/Jellystics/Jellystics/internal/service"
	authsvc "github.com/Jellystics/Jellystics/internal/service/auth"
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

	svcs := &service.Container{
		Auth:    authsvc.New(repos, jfClient, cfg),
		Sync:    syncSvc,
		Stats:   statssvc.New(repos),
		Task:    tasksvc.New(repos, hub),
		Webhook: webhooksvc.New(repos),
	}

	// Periodically broadcast live Jellyfin sessions to all Socket.IO clients.
	go func() {
		ticker := time.NewTicker(10 * time.Second)
		defer ticker.Stop()
		ctx := context.Background()
		for range ticker.C {
			syncSvc.BroadcastSessions(ctx)
		}
	}()

	r := router.New(svcs, repos, hub, db, cfg)

	// Serve embedded SPA (production)
	webFS, err := fs.Sub(assets.Web, "web")
	if err == nil {
		r.NoRoute(spaHandler(webFS))
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
