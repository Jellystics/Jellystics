package handler

import (
	"bufio"
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

// Task names match the old Node.js backend task names exactly.
const (
	taskNameFullSync    = "Full Jellyfin Sync"
	taskNamePartialSync = "Recently Added Sync"
	taskNameBackup      = "Backup"
	taskNameImport      = "Jellyfin Playback Reporting Plugin Sync"
)

type TasksApiHandler struct {
	svcs  *service.Container
	repos *repository.Container
	db    *gorm.DB
}

func NewTasksApiHandler(svcs *service.Container, repos *repository.Container, db *gorm.DB) *TasksApiHandler {
	return &TasksApiHandler{svcs: svcs, repos: repos, db: db}
}

// GET /api/getTasks
// Returns the same task list as the old Node.js backend (same names).
func (h *TasksApiHandler) List(c *gin.Context) {
	ctx := c.Request.Context()

	// Fetch last run info from jf_logging for each task.
	type taskLastRun struct {
		Name    string  `json:"Name"`
		Result  *string `json:"Result"`
		TimeRun *string `json:"TimeRun"`
	}

	// Build a map of task name -> last log entry.
	lastRunMap := map[string]*models.JFLogging{}
	logs, err := h.repos.Log.List(ctx, 200)
	if err == nil {
		for i := range logs {
			l := &logs[i]
			name := ""
			if l.Name != nil {
				name = *l.Name
			}
			if _, already := lastRunMap[name]; !already {
				lastRunMap[name] = l
			}
		}
	}

	isRunning := h.svcs.Task.Status() == "running"

	buildTask := func(key, displayName string) gin.H {
		last := lastRunMap[key]
		var lastRunTime *string
		var lastResult *string
		if last != nil {
			lastRunTime = last.TimeRun
			lastResult = last.Result
		}
		return gin.H{
			"name":        key,
			"displayName": displayName,
			"running":     isRunning,
			"lastRun":     lastRunTime,
			"lastResult":  lastResult,
		}
	}

	tasks := []gin.H{
		buildTask(taskNameFullSync, "Full Jellyfin Sync"),
		buildTask(taskNamePartialSync, "Recently Added Sync"),
		buildTask(taskNameBackup, "Backup"),
		buildTask(taskNameImport, "Jellyfin Playback Reporting Plugin Sync"),
	}
	c.JSON(http.StatusOK, tasks)
}

// POST /api/runTask/:name
func (h *TasksApiHandler) Run(c *gin.Context) {
	name := c.Param("name")

	switch name {
	case taskNameFullSync, "JellyfinSync":
		go func() {
			_ = h.svcs.Task.Run(context.Background(), taskNameFullSync, func(ctx context.Context) error {
				return h.svcs.Sync.FullSync(ctx)
			})
		}()
		c.JSON(http.StatusOK, gin.H{"message": "task started", "name": taskNameFullSync})

	case taskNamePartialSync, "PartialJellyfinSync":
		go func() {
			_ = h.svcs.Task.Run(context.Background(), taskNamePartialSync, func(ctx context.Context) error {
				return h.svcs.Sync.SyncRecentlyAdded(ctx)
			})
		}()
		c.JSON(http.StatusOK, gin.H{"message": "task started", "name": taskNamePartialSync})

	case taskNameBackup, "BackupTask":
		go func() {
			_ = h.svcs.Task.Run(context.Background(), taskNameBackup, func(ctx context.Context) error {
				return runBackupTask(ctx, h.repos)
			})
		}()
		c.JSON(http.StatusOK, gin.H{"message": "task started", "name": taskNameBackup})

	case taskNameImport, "PlaybackReportingPluginSync":
		go func() {
			_ = h.svcs.Task.Run(context.Background(), taskNameImport, func(ctx context.Context) error {
				return h.svcs.Sync.SyncPlaybackPlugin(ctx)
			})
		}()
		c.JSON(http.StatusOK, gin.H{"message": "task started", "name": taskNameImport})

	case "SessionSync":
		go func() {
			_ = h.svcs.Task.Run(context.Background(), "SessionSync", func(ctx context.Context) error {
				return h.svcs.Sync.SyncSessions(ctx)
			})
		}()
		c.JSON(http.StatusOK, gin.H{"message": "task started", "name": "SessionSync"})

	default:
		c.JSON(http.StatusNotFound, gin.H{"error": "unknown task: " + name})
	}
}

// runBackupTask performs a JSON backup of all tables (same tables as old backend).
// This delegates to the same logic used by BackupHandler.Create so task logging works.
func runBackupTask(ctx context.Context, repos *repository.Container) error {
	_, err := createBackupFile(ctx, repos)
	return err
}

// POST /sync/importPlaybackBackup
func (h *TasksApiHandler) ImportBackup(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	f, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open file"})
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	var activities []models.JFPlaybackActivity
	lineNum := 0
	imported := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()
		if lineNum == 1 && strings.Contains(line, "DateCreated") {
			continue // skip header
		}
		if strings.TrimSpace(line) == "" {
			continue
		}

		parts := strings.Split(line, "\t")
		if len(parts) < 9 {
			continue
		}

		dateCreated := strings.TrimSpace(parts[0])
		userId := strings.TrimSpace(parts[1])
		itemId := strings.TrimSpace(parts[2])
		itemName := strings.TrimSpace(parts[4])
		playbackMethod := strings.TrimSpace(parts[5])
		clientName := strings.TrimSpace(parts[6])
		deviceName := strings.TrimSpace(parts[7])
		playDurationStr := strings.TrimSpace(parts[8])

		playDuration, _ := strconv.ParseInt(playDurationStr, 10, 64)
		playDurationSecs := playDuration / 10_000_000 // ticks to seconds

		// Generate an ID based on content
		id := fmt.Sprintf("import-%s-%s-%s", userId, itemId, dateCreated)

		activity := models.JFPlaybackActivity{
			Id:                   id,
			UserId:               &userId,
			NowPlayingItemId:     &itemId,
			NowPlayingItemName:   &itemName,
			PlayMethod:           &playbackMethod,
			Client:               &clientName,
			DeviceName:           &deviceName,
			PlaybackDuration:     &playDurationSecs,
			ActivityDateInserted: &dateCreated,
			Imported:             true,
			Source:               "import",
		}
		activities = append(activities, activity)
		imported++
	}

	if len(activities) > 0 {
		// Deduplicate by generated ID before upserting (duplicate rows in file
		// would cause "ON CONFLICT DO UPDATE command cannot affect row a second time").
		seen := make(map[string]struct{}, len(activities))
		deduped := activities[:0]
		for _, a := range activities {
			if _, dup := seen[a.Id]; !dup {
				seen[a.Id] = struct{}{}
				deduped = append(deduped, a)
			}
		}
		if err := h.repos.Playback.Upsert(c.Request.Context(), deduped); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to import: " + err.Error()})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{"imported": imported, "total": lineNum})
}

// ptr returns a pointer to a string literal.
func taskPtr(s string) *string { return &s }

// nowStr returns the current time as RFC3339 string.
func nowStr() string { return time.Now().UTC().Format(time.RFC3339) }
