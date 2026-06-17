package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

const backupDir = "./data/backups"
const importDir = "./data/imports"

// BackupHandler manages database backup files.
type BackupHandler struct {
	repos *repository.Container
	hub   *ws.Hub
}

func NewBackupHandler(repos *repository.Container, hub *ws.Hub) *BackupHandler {
	_ = os.MkdirAll(backupDir, 0755)
	return &BackupHandler{repos: repos, hub: hub}
}

func (h *BackupHandler) emitLog(msg string) {
	log.Printf("[restore] %s", msg)
	if h.hub != nil {
		h.hub.Emit("TaskLog", msg)
	}
}

func ptrStr(s string) *string { return &s }
func ptrInt(i int64) *int64   { return &i }

// startLogEntry inserts a "running" entry into jf_logging and returns logID + startTime for finishLogEntry.
func (h *BackupHandler) startLogEntry(ctx context.Context, name string) (logID string, startTime string) {
	logID = uuid.New().String()
	startTime = time.Now().UTC().Format(time.RFC3339)
	_ = h.repos.Log.Insert(ctx, &models.JFLogging{
		Id:            logID,
		Name:          ptrStr(name),
		Type:          ptrStr("Task"),
		ExecutionType: ptrStr("Manual"),
		Duration:      ptrInt(0),
		TimeRun:       ptrStr(startTime),
		Log:           ptrStr(`[{}]`),
		Result:        ptrStr("running"),
	})
	return
}

// finishLogEntry upserts the final result into jf_logging.
func (h *BackupHandler) finishLogEntry(ctx context.Context, name, logID, startTime string, elapsed time.Duration, err error) {
	result := "success"
	msg := fmt.Sprintf(`[{"color":"lawngreen","Message":"Task completed in %dms"}]`, elapsed.Milliseconds())
	if err != nil {
		result = "failed"
		msg = fmt.Sprintf(`[{"color":"red","Message":"Task failed after %dms: %s"}]`, elapsed.Milliseconds(), err.Error())
	}
	_ = h.repos.Log.Upsert(ctx, &models.JFLogging{
		Id:            logID,
		Name:          ptrStr(name),
		Type:          ptrStr("Task"),
		ExecutionType: ptrStr("Manual"),
		Duration:      ptrInt(int64(elapsed.Seconds())),
		TimeRun:       ptrStr(startTime),
		Log:           ptrStr(msg),
		Result:        ptrStr(result),
	})
}

// GET /backup/files
func (h *BackupHandler) List(c *gin.Context) {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		c.JSON(http.StatusOK, []any{})
		return
	}

	type fileInfo struct {
		Name        string `json:"name"`
		Size        int64  `json:"size"`
		DateCreated string `json:"datecreated"`
	}

	var files []fileInfo
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, fileInfo{
			Name:        e.Name(),
			Size:        info.Size(),
			DateCreated: info.ModTime().UTC().Format(time.RFC3339),
		})
	}
	if files == nil {
		files = []fileInfo{}
	}
	c.JSON(http.StatusOK, files)
}

// GET /backup/beginBackup
func (h *BackupHandler) Create(c *gin.Context) {
	ctx := c.Request.Context()
	logID, startTime := h.startLogEntry(ctx, "Backup")
	start := time.Now()
	name, err := createBackupFile(ctx, h.repos)
	h.finishLogEntry(ctx, "Backup", logID, startTime, time.Since(start), err)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"name": name, "message": "Backup completed successfully"})
}

// createBackupFile writes a backup JSON file in the same format as the old Node.js backend:
//
//	[ {"tableName": [rows...]}, ... ]
//
// It is called both by the HTTP handler and by the task runner.
func createBackupFile(ctx context.Context, repos *repository.Container) (string, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return "", fmt.Errorf("failed to create backup directory: %w", err)
	}

	backupData := []map[string]any{}

	// --- jf_libraries ---
	libs, err := repos.Library.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read jf_libraries: %w", err)
	}
	backupData = append(backupData, map[string]any{"jf_libraries": libs})

	// --- jf_library_items (all items across all libraries) ---
	allItems := []models.JFLibraryItem{}
	for _, lib := range libs {
		items, err := repos.Item.ListByParent(ctx, lib.Id)
		if err != nil {
			return "", fmt.Errorf("failed to read jf_library_items for %s: %w", lib.Id, err)
		}
		allItems = append(allItems, items...)
	}
	backupData = append(backupData, map[string]any{"jf_library_items": allItems})

	// --- jf_library_seasons ---
	allSeasons := []models.JFLibrarySeason{}
	for _, item := range allItems {
		t := ""
		if item.Type != nil {
			t = *item.Type
		}
		if t != "Series" {
			continue
		}
		seasons, err := repos.Season.ListBySeries(ctx, item.Id)
		if err != nil {
			continue
		}
		allSeasons = append(allSeasons, seasons...)
	}
	backupData = append(backupData, map[string]any{"jf_library_seasons": allSeasons})

	// --- jf_library_episodes ---
	allEpisodes := []models.JFLibraryEpisode{}
	for _, item := range allItems {
		t := ""
		if item.Type != nil {
			t = *item.Type
		}
		if t != "Series" {
			continue
		}
		eps, err := repos.Episode.ListBySeries(ctx, item.Id)
		if err != nil {
			continue
		}
		allEpisodes = append(allEpisodes, eps...)
	}
	backupData = append(backupData, map[string]any{"jf_library_episodes": allEpisodes})

	// --- jf_users ---
	users, err := repos.User.List(ctx)
	if err != nil {
		return "", fmt.Errorf("failed to read jf_users: %w", err)
	}
	backupData = append(backupData, map[string]any{"jf_users": users})

	// --- jf_playback_activity ---
	activities, err := repos.Playback.List(ctx, 0)
	if err != nil {
		return "", fmt.Errorf("failed to read jf_playback_activity: %w", err)
	}
	backupData = append(backupData, map[string]any{"jf_playback_activity": activities})

	// --- jf_playback_reporting_plugin_data ---
	pluginData, _ := repos.PluginData.ListUnimported(ctx)
	if pluginData == nil {
		pluginData = []models.JFPluginData{}
	}
	backupData = append(backupData, map[string]any{"jf_playback_reporting_plugin_data": pluginData})

	// --- jf_item_info (no ListAll available, include empty) ---
	backupData = append(backupData, map[string]any{"jf_item_info": []any{}})

	// Build file name matching old backend format: "backup_YYYY-MM-DD HH-mm-ss.json"
	name := fmt.Sprintf("backup_%s.json", time.Now().Local().Format("2006-01-02 15-04-05"))
	path := filepath.Join(backupDir, name)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer f.Close()

	envelope := map[string]any{
		"source": "Jellystics",
		"data":   backupData,
	}

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(envelope); err != nil {
		return "", fmt.Errorf("failed to write backup: %w", err)
	}

	// Keep only the 5 most recent backups.
	pruneOldBackups()

	return name, nil
}

// pruneOldBackups removes backup files beyond the 5 most recent.
func pruneOldBackups() {
	entries, err := os.ReadDir(backupDir)
	if err != nil {
		return
	}

	type bf struct {
		name    string
		modTime time.Time
	}
	var files []bf
	for _, e := range entries {
		if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") {
			continue
		}
		info, err := e.Info()
		if err != nil {
			continue
		}
		files = append(files, bf{name: e.Name(), modTime: info.ModTime()})
	}

	// Sort newest first (simple bubble sort – there will rarely be >10 files).
	for i := 0; i < len(files)-1; i++ {
		for j := i + 1; j < len(files); j++ {
			if files[j].modTime.After(files[i].modTime) {
				files[i], files[j] = files[j], files[i]
			}
		}
	}

	// Delete files beyond the 5th.
	for i := 5; i < len(files); i++ {
		_ = os.Remove(filepath.Join(backupDir, files[i].name))
	}
}

// GET /backup/restore/:filename
// Restores supported tables from a backup JSON file.
func (h *BackupHandler) Restore(c *gin.Context) {
	filename := filepath.Base(c.Param("filename"))
	if !strings.HasSuffix(filename, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
		return
	}
	path := filepath.Join(backupDir, filename)
	isImport := false
	data, err := os.ReadFile(path)
	if err != nil {
		// Try import directory (uploaded-for-restore files)
		importPath := filepath.Join(importDir, filename)
		data, err = os.ReadFile(importPath)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": "backup file not found"})
			return
		}
		path = importPath
		isImport = true
	}

	ctx := c.Request.Context()
	logID, startTime := h.startLogEntry(ctx, "Import JSON Backup")
	restoreStart := time.Now()
	restored := map[string]int{}

	// Detect format:
	//   New Jellystics:  {"source":"Jellystics","data":[{"jf_table":[rows...]},…]}
	//   Old Jellystics:  [{"jf_table":[rows...]},…]
	//   Jellystat:       [{"tableName":"jf_table","rows":[rows...]},…]
	var tables []map[string]json.RawMessage

	var envelope struct {
		Source string          `json:"source"`
		Data   json.RawMessage `json:"data"`
	}
	if err := json.Unmarshal(data, &envelope); err == nil && envelope.Source == "Jellystics" {
		// New Jellystics envelope
		if err := json.Unmarshal(envelope.Data, &tables); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid Jellystics backup data: " + err.Error()})
			return
		}
	} else {
		// Try plain array (old Jellystics or Jellystat)
		var rawArray []json.RawMessage
		if err := json.Unmarshal(data, &rawArray); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup format: " + err.Error()})
			return
		}
		// Detect Jellystat format: [{"tableName":"...","rows":[...]}]
		isJellystat := false
		if len(rawArray) > 0 {
			var probe map[string]json.RawMessage
			if json.Unmarshal(rawArray[0], &probe) == nil {
				_, isJellystat = probe["tableName"]
			}
		}
		if isJellystat {
			type jellystatEntry struct {
				TableName string          `json:"tableName"`
				Rows      json.RawMessage `json:"rows"`
				Data      json.RawMessage `json:"data"`
			}
			for _, raw := range rawArray {
				var entry jellystatEntry
				if err := json.Unmarshal(raw, &entry); err != nil || entry.TableName == "" {
					continue
				}
				rows := entry.Rows
				if rows == nil {
					rows = entry.Data
				}
				tables = append(tables, map[string]json.RawMessage{entry.TableName: rows})
			}
		} else {
			// Old Jellystics: [{"jf_table": [...]}]
			for _, raw := range rawArray {
				var entry map[string]json.RawMessage
				if err := json.Unmarshal(raw, &entry); err != nil {
					continue
				}
				tables = append(tables, entry)
			}
		}
	}

	h.emitLog("Starting restore...")
	for _, tableEntry := range tables {
		for tableName, rawRows := range tableEntry {
			count, err := restoreTableData(ctx, h.repos, tableName, rawRows)
			if err != nil {
				h.emitLog(fmt.Sprintf("ERROR restoring %s: %s", tableName, err.Error()))
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("failed to restore table %s: %s", tableName, err.Error()),
				})
				return
			}
			restored[tableName] = count
			h.emitLog(fmt.Sprintf("Restored %s: %d rows", tableName, count))
		}
	}

	// Run post-restore tasks in background to avoid blocking the HTTP response.
	pluginDataCount := restored["jf_playback_reporting_plugin_data"]
	total := 0
	for _, n := range restored {
		total += n
	}
	h.emitLog(fmt.Sprintf("Successfully restored %d rows", total))

	go func() {
		bgCtx := context.Background()
		if pluginDataCount > 0 {
			h.emitLog("Merging plugin data into playback activity...")
			_ = h.repos.PluginData.MergeIntoPlaybackActivity(bgCtx)
		}
		h.emitLog("Refreshing stats views...")
		_ = h.repos.Stats.RefreshViews(bgCtx)
		if isImport {
			_ = os.Remove(path)
		}
		h.finishLogEntry(bgCtx, "Import JSON Backup", logID, startTime, time.Since(restoreStart), nil)
		h.emitLog("Restore complete.")
	}()

	c.JSON(http.StatusOK, gin.H{"message": "Restore completed successfully", "restored": restored})
}

// restoreTableData restores rows for a specific table using typed repositories.
// Tables not handled here are counted but not re-inserted.
func restoreTableData(ctx context.Context, repos *repository.Container, tableName string, rawRows json.RawMessage) (int, error) {
	switch tableName {
	case "jf_playback_activity":
		var rows []models.JFPlaybackActivity
		if err := unmarshalBackupRows(rawRows, &rows); err != nil {
			return 0, err
		}
		for i := range rows {
			if rows[i].Source == "" {
				rows[i].Source = "restore"
			}
		}
		if len(rows) > 0 {
			if err := repos.Playback.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_users":
		var rows []models.JFUser
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.User.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_libraries":
		var rows []models.JFLibrary
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.Library.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_library_items":
		var rows []models.JFLibraryItem
		if err := unmarshalBackupRows(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.Item.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_library_seasons":
		var rows []models.JFLibrarySeason
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.Season.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_library_episodes":
		var rows []models.JFLibraryEpisode
		if err := unmarshalBackupRows(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.Episode.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_item_info":
		var rows []models.JFItemInfo
		if err := unmarshalBackupRows(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.ItemInfo.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	case "jf_playback_reporting_plugin_data":
		var rows []models.JFPluginData
		if err := unmarshalBackupRows(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.PluginData.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	default:
		// Count without restoring unsupported legacy tables.
		var rows []json.RawMessage
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, nil
		}
		return len(rows), nil
	}
}

var backupIntegerFields = map[string]struct{}{
	"RunTimeTicks":      {},
	"ProductionYear":    {},
	"IndexNumber":       {},
	"ParentIndexNumber": {},
	"Size":              {},
	"Bitrate":           {},
	"PlaybackDuration":  {},
	"PlayDuration":      {},
	"total_play_time":   {},
	"item_count":        {},
	"season_count":      {},
	"episode_count":     {},
}

var backupFloatFields = map[string]struct{}{
	"CommunityRating": {},
}

func unmarshalBackupRows(rawRows json.RawMessage, target any) error {
	var rows []map[string]any
	if err := json.Unmarshal(rawRows, &rows); err != nil {
		return err
	}
	for _, row := range rows {
		for key, value := range row {
			s, ok := value.(string)
			if !ok || strings.TrimSpace(s) == "" {
				continue
			}
			if _, ok := backupIntegerFields[key]; ok {
				if parsed, err := strconv.ParseInt(s, 10, 64); err == nil {
					row[key] = parsed
				}
				continue
			}
			if _, ok := backupFloatFields[key]; ok {
				if parsed, err := strconv.ParseFloat(s, 64); err == nil {
					row[key] = parsed
				}
			}
		}
	}
	normalized, err := json.Marshal(rows)
	if err != nil {
		return err
	}
	return json.Unmarshal(normalized, target)
}

// DELETE /backup/files/:name
func (h *BackupHandler) Delete(c *gin.Context) {
	name := filepath.Base(c.Param("name"))
	if !strings.HasSuffix(name, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
		return
	}
	path := filepath.Join(backupDir, name)
	if err := os.Remove(path); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /backup/files/:name  (public – download link from browser)
func (h *BackupHandler) Download(c *gin.Context) {
	name := filepath.Base(c.Param("name"))
	if !strings.HasSuffix(name, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid file name"})
		return
	}
	path := filepath.Join(backupDir, name)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "file not found"})
		return
	}
	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="%s"`, name))
	c.File(path)
}

// POST /backup/upload
// Accepts a multipart file upload and saves it to the backup directory.
func (h *BackupHandler) Upload(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "file is required"})
		return
	}

	filename := filepath.Base(file.Filename)
	if !strings.HasSuffix(filename, ".json") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "only .json files are accepted"})
		return
	}

	src, err := file.Open()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to open uploaded file"})
		return
	}
	defer src.Close()

	if err := os.MkdirAll(importDir, 0o755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create import directory"})
		return
	}
	destPath := filepath.Join(importDir, filename)
	dst, err := os.Create(destPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to save file"})
		return
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to write file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"fileName": filename,
		"filePath": destPath,
	})
}
