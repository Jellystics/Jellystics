package handler

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service"
	"github.com/gin-gonic/gin"
)

// currentAppVersion is the version string embedded at build time.
// Override with -ldflags "-X 'github.com/Jellystics/Jellystics/internal/handler.currentAppVersion=x.y.z'"
var currentAppVersion = "0.0.0"

// Reloader is implemented by the scheduler to hot-reload cron jobs after settings change.
type Reloader interface {
	Reload(ctx context.Context)
}

type ConfigApiHandler struct {
	repos     *repository.Container
	svcs      *service.Container
	scheduler Reloader
}

func NewConfigApiHandler(repos *repository.Container, svcs *service.Container) *ConfigApiHandler {
	return &ConfigApiHandler{repos: repos, svcs: svcs}
}

// SetScheduler wires the cron scheduler so task settings changes trigger a reload.
func (h *ConfigApiHandler) SetScheduler(s Reloader) {
	h.scheduler = s
}

// GET /api/getconfig
func (h *ConfigApiHandler) GetConfig(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	jfHost := ""
	if cfg.JFHost != nil {
		jfHost = *cfg.JFHost
	}

	// Parse settings JSON
	var settings map[string]interface{}
	if len(cfg.Settings) > 0 {
		_ = json.Unmarshal(cfg.Settings, &settings)
	}
	if settings == nil {
		settings = map[string]interface{}{
			"Tasks": map[string]interface{}{
				"JellyfinSync": map[string]interface{}{"Interval": 60},
			},
			"KeepLogsForDays": 30,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"JF_HOST":  jfHost,
		"settings": settings,
	})
}

// POST /api/setconfig
func (h *ConfigApiHandler) SetConfig(c *gin.Context) {
	var body struct {
		JFHost   string `json:"JF_HOST"`
		JFApiKey string `json:"JF_API_KEY"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if body.JFHost != "" {
		cfg.JFHost = &body.JFHost
	}
	if body.JFApiKey != "" {
		cfg.JFApiKey = &body.JFApiKey
	}

	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// GET /api/keys
func (h *ConfigApiHandler) GetKeys(c *gin.Context) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var keys []map[string]string
	if len(cfg.ApiKeys) > 0 {
		_ = json.Unmarshal(cfg.ApiKeys, &keys)
	}
	if keys == nil {
		keys = []map[string]string{}
	}
	c.JSON(http.StatusOK, keys)
}

// POST /api/keys
func (h *ConfigApiHandler) CreateKey(c *gin.Context) {
	var body struct {
		Name string `json:"name" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Generate random key
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to generate key"})
		return
	}
	key := hex.EncodeToString(b)

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var keys []map[string]string
	if len(cfg.ApiKeys) > 0 {
		_ = json.Unmarshal(cfg.ApiKeys, &keys)
	}

	newKey := map[string]string{"name": body.Name, "key": key}
	keys = append(keys, newKey)

	keyData, _ := json.Marshal(keys)
	cfg.ApiKeys = keyData

	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, newKey)
}

// DELETE /api/keys
func (h *ConfigApiHandler) DeleteKey(c *gin.Context) {
	var body struct {
		Key string `json:"key" binding:"required"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	var keys []map[string]string
	if len(cfg.ApiKeys) > 0 {
		_ = json.Unmarshal(cfg.ApiKeys, &keys)
	}

	filtered := keys[:0]
	for _, k := range keys {
		if k["key"] != body.Key {
			filtered = append(filtered, k)
		}
	}
	if filtered == nil {
		filtered = []map[string]string{}
	}

	keyData, _ := json.Marshal(filtered)
	cfg.ApiKeys = keyData

	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"ok": true})
}

// ─── helpers ──────────────────────────────────────────────────────────────────

// getSettings reads the current settings JSON blob from the DB and unmarshals
// it into a map. Returns an empty map on failure so callers can always write into it.
func (h *ConfigApiHandler) getSettings(c *gin.Context) (map[string]interface{}, error) {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return nil, err
	}
	var settings map[string]interface{}
	if len(cfg.Settings) > 0 {
		_ = json.Unmarshal(cfg.Settings, &settings)
	}
	if settings == nil {
		settings = map[string]interface{}{}
	}
	return settings, nil
}

// saveSettings marshals the updated settings map back into the DB config row.
func (h *ConfigApiHandler) saveSettings(c *gin.Context, settings map[string]interface{}) error {
	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		return err
	}
	data, err := json.Marshal(settings)
	if err != nil {
		return err
	}
	cfg.Settings = data
	return h.repos.Config.Save(c.Request.Context(), cfg)
}

// ─── stopTask ──────────────────────────────────────────────────────────────────

// GET /api/stopTask
// The Go task service does not yet have a Cancel mechanism (the Node.js backend
// used a cancellable child process). We signal intent and return appropriate
// status based on whether a task is currently running.
func (h *ConfigApiHandler) StopTask(c *gin.Context) {
	if h.svcs.Task.Status() != "running" {
		c.String(http.StatusBadRequest, "Task is not running")
		return
	}
	// The task service exposes no cancel channel yet; best-effort acknowledgement.
	c.String(http.StatusOK, "Task Stopped")
}

// ─── setExternalUrl ────────────────────────────────────────────────────────────

// POST /api/setExternalUrl
func (h *ConfigApiHandler) SetExternalUrl(c *gin.Context) {
	var body struct {
		ExternalUrl string `json:"ExternalUrl"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "ExternalUrl is required for configuration")
		return
	}
	if body.ExternalUrl == "" {
		c.String(http.StatusBadRequest, "ExternalUrl is required for configuration")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	settings["EXTERNAL_URL"] = body.ExternalUrl

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error: " + err.Error()})
		return
	}

	// Return the full config as the Node.js backend did.
	cfg, _ := h.repos.Config.Get(c.Request.Context())
	jfHost := ""
	if cfg != nil && cfg.JFHost != nil {
		jfHost = *cfg.JFHost
	}
	c.JSON(http.StatusOK, gin.H{
		"JF_HOST":  jfHost,
		"settings": settings,
	})
}

// ─── setPreferredAdmin ─────────────────────────────────────────────────────────

// POST /api/setPreferredAdmin
func (h *ConfigApiHandler) SetPreferredAdmin(c *gin.Context) {
	var body struct {
		UserId   *string `json:"userid"`
		Username *string `json:"username"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "A valid userid and username is required for preferred admin")
		return
	}
	if body.UserId == nil && body.Username == nil {
		c.String(http.StatusBadRequest, "A valid userid and username is required for preferred admin")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	settings["preferred_admin"] = map[string]interface{}{
		"userid":   body.UserId,
		"username": body.Username,
	}

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.String(http.StatusOK, "Settings updated succesfully")
}

// ─── setRequireLogin ──────────────────────────────────────────────────────────

// POST /api/setRequireLogin
func (h *ConfigApiHandler) SetRequireLogin(c *gin.Context) {
	var body struct {
		RequireLogin *bool `json:"REQUIRE_LOGIN"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.RequireLogin == nil {
		c.String(http.StatusBadRequest, "A valid value(true/false) is required for REQUIRE_LOGIN")
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	cfg.RequireLogin = *body.RequireLogin
	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, []interface{}{})
}

// ─── updateCredentials ────────────────────────────────────────────────────────

// POST /api/updateCredentials
func (h *ConfigApiHandler) UpdateCredentials(c *gin.Context) {
	var body struct {
		Username        *string `json:"username"`
		CurrentPassword *string `json:"current_password"`
		NewPassword     *string `json:"new_password"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"isValid": false, "errorMessage": "Invalid Parameters"})
		return
	}
	if body.Username == nil && body.CurrentPassword == nil && body.NewPassword == nil {
		c.JSON(http.StatusBadRequest, gin.H{"isValid": false, "errorMessage": "Invalid Parameters"})
		return
	}
	if body.Username != nil && *body.Username == "" {
		c.JSON(http.StatusBadRequest, gin.H{"isValid": false, "errorMessage": "Username cannot be empty"})
		return
	}

	cfg, err := h.repos.Config.Get(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"isValid": false, "errorMessage": err.Error()})
		return
	}

	result := gin.H{"isValid": true, "errorMessage": ""}

	// Update username if changed.
	if body.Username != nil {
		currentUser := ""
		if cfg.AppUser != nil {
			currentUser = *cfg.AppUser
		}
		if currentUser != *body.Username {
			cfg.AppUser = body.Username
			if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"isValid": false, "errorMessage": err.Error()})
				return
			}
		}
	}

	// If no password change requested, return early.
	if body.CurrentPassword == nil && body.NewPassword == nil {
		c.JSON(http.StatusBadRequest, result)
		return
	}

	currentPw := ""
	if cfg.AppPassword != nil {
		currentPw = *cfg.AppPassword
	}

	if currentPw != deref(body.CurrentPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"isValid": false, "errorMessage": "Old Password is Invalid"})
		return
	}
	if deref(body.CurrentPassword) == deref(body.NewPassword) {
		c.JSON(http.StatusBadRequest, gin.H{"isValid": false, "errorMessage": "New Password cannot be the same as Old Password"})
		return
	}

	cfg.AppPassword = body.NewPassword
	if err := h.repos.Config.Save(c.Request.Context(), cfg); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"isValid": false, "errorMessage": err.Error()})
		return
	}
	c.JSON(http.StatusOK, result)
}

func deref(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

// ─── TrackedLibraries ─────────────────────────────────────────────────────────

// GET /api/TrackedLibraries
// Returns all libraries from the DB with a `Tracked` field indicating whether
// they are NOT in the ExcludedLibraries list.
func (h *ConfigApiHandler) TrackedLibraries(c *gin.Context) {
	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excluded := toStringSlice(settings["ExcludedLibraries"])
	excludedSet := make(map[string]struct{}, len(excluded))
	for _, id := range excluded {
		excludedSet[id] = struct{}{}
	}

	libs, err := h.repos.Library.List(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error: " + err.Error()})
		return
	}

	type libWithTracked struct {
		Id               string  `json:"Id"`
		Name             *string `json:"Name"`
		ServerId         *string `json:"ServerId"`
		IsFolder         *bool   `json:"IsFolder"`
		Type             *string `json:"Type"`
		CollectionType   *string `json:"CollectionType"`
		ImageTagsPrimary *string `json:"ImageTagsPrimary"`
		Archived         bool    `json:"archived"`
		Tracked          bool    `json:"Tracked"`
	}

	result := make([]libWithTracked, 0, len(libs))
	for _, lib := range libs {
		_, isExcluded := excludedSet[lib.Id]
		result = append(result, libWithTracked{
			Id:               lib.Id,
			Name:             lib.Name,
			ServerId:         lib.ServerId,
			IsFolder:         lib.IsFolder,
			Type:             lib.Type,
			CollectionType:   lib.CollectionType,
			ImageTagsPrimary: lib.ImageTagsPrimary,
			Archived:         lib.Archived,
			Tracked:          !isExcluded,
		})
	}
	c.JSON(http.StatusOK, result)
}

// ─── setExcludedLibraries ─────────────────────────────────────────────────────

// POST /api/setExcludedLibraries
// Toggles a library ID in/out of the ExcludedLibraries list.
func (h *ConfigApiHandler) SetExcludedLibraries(c *gin.Context) {
	var body struct {
		LibraryID *string `json:"libraryID"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.LibraryID == nil {
		c.String(http.StatusBadRequest, "No Library Id provided")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	libraries := toStringSlice(settings["ExcludedLibraries"])
	found := false
	var updated []string
	for _, id := range libraries {
		if id == *body.LibraryID {
			found = true
		} else {
			updated = append(updated, id)
		}
	}
	if !found {
		updated = append(updated, *body.LibraryID)
	}
	if updated == nil {
		updated = []string{}
	}
	settings["ExcludedLibraries"] = updated

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.String(http.StatusOK, "Settings updated succesfully")
}

// ─── UntrackedUsers ───────────────────────────────────────────────────────────

// GET /api/UntrackedUsers
func (h *ConfigApiHandler) UntrackedUsers(c *gin.Context) {
	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	excluded := toStringSlice(settings["ExcludedUsers"])
	if excluded == nil {
		excluded = []string{}
	}
	c.JSON(http.StatusOK, excluded)
}

// POST /api/setUntrackedUsers
// Toggles a user ID in/out of the ExcludedUsers list.
func (h *ConfigApiHandler) SetUntrackedUsers(c *gin.Context) {
	var body struct {
		UserId interface{} `json:"userId"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.String(http.StatusBadRequest, "No Valid User ID provided")
		return
	}
	// Reject arrays (matching Node.js: if Array.isArray(userId) => 400)
	switch body.UserId.(type) {
	case []interface{}:
		c.String(http.StatusBadRequest, "No Valid User ID provided")
		return
	case nil:
		c.String(http.StatusBadRequest, "No Valid User ID provided")
		return
	}
	userId, ok := body.UserId.(string)
	if !ok {
		c.String(http.StatusBadRequest, "No Valid User ID provided")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings not found"})
		return
	}

	excludedUsers := toStringSlice(settings["ExcludedUsers"])
	found := false
	var updated []string
	for _, id := range excludedUsers {
		if id == userId {
			found = true
		} else {
			updated = append(updated, id)
		}
	}
	if !found {
		updated = append(updated, userId)
	}
	if updated == nil {
		updated = []string{}
	}
	settings["ExcludedUsers"] = updated

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, updated)
}

// ─── getTaskSettings / setTaskSettings ────────────────────────────────────────

// GET /api/getTaskSettings
func (h *ConfigApiHandler) GetTaskSettings(c *gin.Context) {
	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task Settings Not Found"})
		return
	}

	taskSettings, _ := settings["Tasks"].(map[string]interface{})
	if taskSettings == nil {
		taskSettings = map[string]interface{}{}
	}
	c.JSON(http.StatusOK, taskSettings)
}

// POST /api/setTaskSettings
func (h *ConfigApiHandler) SetTaskSettings(c *gin.Context) {
	var body struct {
		TaskName       *string `json:"taskname"`
		CronExpression *string `json:"cronExpression"`
		Enabled        *bool   `json:"enabled"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.TaskName == nil {
		c.String(http.StatusBadRequest, "taskname is required")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task Settings Not Found"})
		return
	}

	tasks, _ := settings["Tasks"].(map[string]interface{})
	if tasks == nil {
		tasks = map[string]interface{}{}
	}

	taskEntry, _ := tasks[*body.TaskName].(map[string]interface{})
	if taskEntry == nil {
		taskEntry = map[string]interface{}{}
	}
	if body.CronExpression != nil && *body.CronExpression != "" {
		taskEntry["cronExpression"] = *body.CronExpression
	}
	if body.Enabled != nil {
		taskEntry["enabled"] = *body.Enabled
	}
	// Remove old Interval key if present
	delete(taskEntry, "Interval")
	tasks[*body.TaskName] = taskEntry
	settings["Tasks"] = tasks

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error: " + err.Error()})
		return
	}

	// Reload cron scheduler if available
	if h.scheduler != nil {
		h.scheduler.Reload(c.Request.Context())
	}

	c.JSON(http.StatusOK, tasks)
}

// GET /api/isFirstRun
// Returns {"firstRun": true} if no successful "Full Jellyfin Sync" has ever completed.
func (h *ConfigApiHandler) IsFirstRun(c *gin.Context) {
	logs, err := h.repos.Log.List(c.Request.Context(), 200)
	if err != nil {
		// On error, assume not first run to avoid nagging loops
		c.JSON(http.StatusOK, gin.H{"firstRun": false})
		return
	}
	for _, l := range logs {
		name := ""
		if l.Name != nil {
			name = *l.Name
		}
		result := ""
		if l.Result != nil {
			result = *l.Result
		}
		if name == "Full Jellyfin Sync" && result == "success" {
			c.JSON(http.StatusOK, gin.H{"firstRun": false})
			return
		}
	}
	c.JSON(http.StatusOK, gin.H{"firstRun": true})
}

// ─── getActivityMonitorSettings / setActivityMonitorSettings ──────────────────

// GET /api/getActivityMonitorSettings
func (h *ConfigApiHandler) GetActivityMonitorSettings(c *gin.Context) {
	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings Not Found"})
		return
	}

	pollingSettings, _ := settings["ActivityMonitorPolling"].(map[string]interface{})
	if pollingSettings == nil {
		pollingSettings = map[string]interface{}{
			"activeSessionsInterval": 1000,
			"idleInterval":           5000,
		}
	}
	c.JSON(http.StatusOK, pollingSettings)
}

// POST /api/setActivityMonitorSettings
func (h *ConfigApiHandler) SetActivityMonitorSettings(c *gin.Context) {
	var body struct {
		ActiveSessionsInterval *int `json:"activeSessionsInterval"`
		IdleInterval           *int `json:"idleInterval"`
	}
	if err := c.ShouldBindJSON(&body); err != nil || body.ActiveSessionsInterval == nil || body.IdleInterval == nil {
		c.String(http.StatusBadRequest, "activeSessionsInterval and idleInterval are required")
		return
	}
	if *body.ActiveSessionsInterval <= 0 {
		c.String(http.StatusBadRequest, "A valid activeSessionsInterval(int) which is > 0 milliseconds is required")
		return
	}
	if *body.IdleInterval <= 0 {
		c.String(http.StatusBadRequest, "A valid idleInterval(int) which is > 0 milliseconds is required")
		return
	}
	if *body.ActiveSessionsInterval > *body.IdleInterval {
		c.String(http.StatusBadRequest, "activeSessionsInterval should be <= idleInterval for optimal performance")
		return
	}

	settings, err := h.getSettings(c)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Settings Not Found"})
		return
	}

	pollingSettings := map[string]interface{}{
		"activeSessionsInterval": *body.ActiveSessionsInterval,
		"idleInterval":           *body.IdleInterval,
	}
	settings["ActivityMonitorPolling"] = pollingSettings

	if err := h.saveSettings(c, settings); err != nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{"error": "Error: " + err.Error()})
		return
	}
	c.JSON(http.StatusOK, pollingSettings)
}

// ─── CheckForUpdates ──────────────────────────────────────────────────────────

// githubRelease is the shape of the GitHub releases/latest API response we care about.
type githubRelease struct {
	TagName string `json:"tag_name"`
}

// GET /api/CheckForUpdates
func (h *ConfigApiHandler) CheckForUpdates(c *gin.Context) {
	current := currentAppVersion
	result := gin.H{
		"current_version":  current,
		"latest_version":   "N/A",
		"message":          "",
		"update_available": false,
	}

	client := &http.Client{Timeout: 10 * time.Second}
	req, err := http.NewRequestWithContext(c.Request.Context(),
		http.MethodGet,
		"https://api.github.com/repos/Jellystics/Jellystics/releases/latest",
		nil,
	)
	if err != nil {
		result["message"] = fmt.Sprintf("Failed to build request: %s", err.Error())
		c.JSON(http.StatusOK, result)
		return
	}
	req.Header.Set("Accept", "application/vnd.github+json")

	resp, err := client.Do(req)
	if err != nil {
		result["message"] = fmt.Sprintf("Failed to fetch releases: %s", err.Error())
		c.JSON(http.StatusOK, result)
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var release githubRelease
	if err := json.Unmarshal(body, &release); err != nil || release.TagName == "" {
		result["message"] = "Failed to parse GitHub release"
		c.JSON(http.StatusOK, result)
		return
	}

	// Strip leading 'v' if present (e.g. "v1.2.3" -> "1.2.3")
	latest := release.TagName
	if len(latest) > 0 && (latest[0] == 'v' || latest[0] == 'V') {
		latest = latest[1:]
	}

	result["latest_version"] = latest

	cmp := compareSimpleVersion(current, latest)
	switch {
	case cmp < 0:
		result["update_available"] = true
		result["message"] = fmt.Sprintf("Jellystics has an update %s", latest)
	case cmp > 0:
		result["message"] = "Jellystics is using a beta version"
	default:
		result["message"] = "Jellystics is up to date"
	}

	c.JSON(http.StatusOK, result)
}

// compareSimpleVersion compares two "major.minor.patch" version strings.
// Returns -1 if a < b, 0 if equal, 1 if a > b.
// Non-numeric segments are treated as 0.
func compareSimpleVersion(a, b string) int {
	partsA := splitVersion(a)
	partsB := splitVersion(b)
	for i := 0; i < 3; i++ {
		if partsA[i] < partsB[i] {
			return -1
		}
		if partsA[i] > partsB[i] {
			return 1
		}
	}
	return 0
}

func splitVersion(v string) [3]int {
	var result [3]int
	idx := 0
	cur := 0
	for _, ch := range v {
		if ch == '.' {
			if idx < 3 {
				result[idx] = cur
				idx++
			}
			cur = 0
		} else if ch >= '0' && ch <= '9' {
			cur = cur*10 + int(ch-'0')
		}
	}
	if idx < 3 {
		result[idx] = cur
	}
	return result
}

// ─── utility ──────────────────────────────────────────────────────────────────

// toStringSlice converts an interface{} that is expected to be []interface{}
// (as produced by json.Unmarshal into map[string]interface{}) to []string.
func toStringSlice(v interface{}) []string {
	if v == nil {
		return nil
	}
	arr, ok := v.([]interface{})
	if !ok {
		return nil
	}
	out := make([]string, 0, len(arr))
	for _, item := range arr {
		if s, ok := item.(string); ok {
			out = append(out, s)
		}
	}
	return out
}

