package handler

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/gin-gonic/gin"
)

const backupDir = "./data/backups"

// BackupHandler manages database backup files.
type BackupHandler struct {
	repos *repository.Container
}

func NewBackupHandler(repos *repository.Container) *BackupHandler {
	_ = os.MkdirAll(backupDir, 0755)
	return &BackupHandler{repos: repos}
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
	name, err := createBackupFile(ctx, h.repos)
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
	name := fmt.Sprintf("backup_%s.json", time.Now().UTC().Format("2006-01-02 15-04-05"))
	path := filepath.Join(backupDir, name)

	f, err := os.Create(path)
	if err != nil {
		return "", fmt.Errorf("failed to create backup file: %w", err)
	}
	defer f.Close()

	enc := json.NewEncoder(f)
	enc.SetIndent("", "  ")
	if err := enc.Encode(backupData); err != nil {
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

	data, err := os.ReadFile(path)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "backup file not found"})
		return
	}

	// Old-style backup format: [ {"tableName": [rows...]}, ... ]
	var tables []map[string]json.RawMessage
	if err := json.Unmarshal(data, &tables); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid backup format: " + err.Error()})
		return
	}

	ctx := c.Request.Context()
	restored := map[string]int{}
	for _, tableEntry := range tables {
		for tableName, rawRows := range tableEntry {
			count, err := restoreTableData(ctx, h.repos, tableName, rawRows)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": fmt.Sprintf("failed to restore table %s: %s", tableName, err.Error()),
				})
				return
			}
			restored[tableName] = count
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "Restore completed successfully", "restored": restored})
}

// restoreTableData restores rows for a specific table using typed repositories.
// Tables not handled here are counted but not re-inserted.
func restoreTableData(ctx context.Context, repos *repository.Container, tableName string, rawRows json.RawMessage) (int, error) {
	switch tableName {
	case "jf_playback_activity":
		var rows []models.JFPlaybackActivity
		if err := json.Unmarshal(rawRows, &rows); err != nil {
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
		if err := json.Unmarshal(rawRows, &rows); err != nil {
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
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, err
		}
		if len(rows) > 0 {
			if err := repos.Episode.Upsert(ctx, rows); err != nil {
				return 0, err
			}
		}
		return len(rows), nil

	default:
		// Count without restoring (e.g. jf_item_info, jf_playback_reporting_plugin_data).
		var rows []json.RawMessage
		if err := json.Unmarshal(rawRows, &rows); err != nil {
			return 0, nil
		}
		return len(rows), nil
	}
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

	destPath := filepath.Join(backupDir, filename)
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
