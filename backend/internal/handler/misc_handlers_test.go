package handler_test

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/handler"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
)

// jsonBody wraps a JSON string as an io.Reader for request bodies.
func jsonBody(s string) io.Reader { return strings.NewReader(s) }

// TestSessionsCurrent_NoHost verifies that with no JF host configured the
// sessions endpoint returns an empty array (not an error).
func TestSessionsCurrent_NoHost(t *testing.T) {
	db := testutil.NewDB(t)
	gin.SetMode(gin.TestMode)
	h := handler.NewSessionsApiHandler(repository.New(db))
	r := gin.New()
	r.GET("/sessions/current", h.Current)

	w := do(t, r, "/sessions/current")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var arr []interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &arr); err != nil {
		t.Fatalf("decode: %v (body %s)", err, w.Body.String())
	}
	if len(arr) != 0 {
		t.Errorf("sessions = %v, want empty", arr)
	}
}

// TestLogsGetLogs verifies GetLogs returns the seeded log entries and, when
// empty, returns [] not null.
func TestLogsGetLogs(t *testing.T) {
	db := testutil.NewDB(t)
	repos := repository.New(db)
	gin.SetMode(gin.TestMode)
	h := handler.NewLogsHandler(repos)
	r := gin.New()
	r.GET("/logs/getLogs", h.GetLogs)

	// Empty: expect [].
	w := do(t, r, "/logs/getLogs")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	var logs []models.JFLogging
	if err := json.Unmarshal(w.Body.Bytes(), &logs); err != nil {
		t.Fatalf("decode empty: %v (body %s)", err, w.Body.String())
	}
	if len(logs) != 0 {
		t.Errorf("empty logs = %d, want 0", len(logs))
	}

	// Insert one log and expect it back.
	name := "Full Jellyfin Sync"
	res := "success"
	tr := "2026-07-12T10:00:00Z"
	if err := repos.Log.Insert(t.Context(), &models.JFLogging{
		Id: "log-1", Name: &name, Result: &res, TimeRun: &tr,
	}); err != nil {
		t.Fatalf("insert log: %v", err)
	}
	w2 := do(t, r, "/logs/getLogs")
	if err := json.Unmarshal(w2.Body.Bytes(), &logs); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(logs) != 1 || logs[0].Id != "log-1" {
		t.Errorf("logs = %+v, want just log-1", logs)
	}
}

// TestSettingsGet verifies the settings endpoint never leaks the API key/host
// values, only booleans indicating their presence.
func TestSettingsGet(t *testing.T) {
	db := testutil.NewDB(t)
	repos := repository.New(db)
	// Configure host + api key + requireLogin so the boolean flags flip to true.
	cfg, err := repos.Config.Get(t.Context())
	if err != nil {
		t.Fatalf("config get: %v", err)
	}
	cfg.JFHost = sp("http://jf.local")
	cfg.JFApiKey = sp("secret-key")
	cfg.RequireLogin = true
	if err := repos.Config.Save(t.Context(), cfg); err != nil {
		t.Fatalf("config save: %v", err)
	}

	gin.SetMode(gin.TestMode)
	h := handler.NewSettingsHandler(repos)
	r := gin.New()
	r.GET("/api/settings", h.Get)

	w := do(t, r, "/api/settings")
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}
	var got struct {
		RequireLogin bool `json:"requireLogin"`
		HasHost      bool `json:"hasHost"`
		HasApiKey    bool `json:"hasApiKey"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &got); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !got.RequireLogin || !got.HasHost || !got.HasApiKey {
		t.Errorf("got = %+v, want all true", got)
	}
	// Ensure the raw secret is not present in the body.
	if body := w.Body.String(); strings.Contains(body, "secret-key") || strings.Contains(body, "jf.local") {
		t.Errorf("settings body leaked sensitive value: %s", body)
	}
}

// TestSettingsUpdate verifies PUT /api/settings persists provided fields.
func TestSettingsUpdate(t *testing.T) {
	db := testutil.NewDB(t)
	repos := repository.New(db)
	gin.SetMode(gin.TestMode)
	h := handler.NewSettingsHandler(repos)
	r := gin.New()
	r.PUT("/api/settings", h.Update)

	body := `{"jfHost":"http://new-host","requireLogin":true}`
	req := httptest.NewRequest(http.MethodPut, "/api/settings", jsonBody(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body %s", w.Code, w.Body.String())
	}

	cfg, _ := repos.Config.Get(t.Context())
	if cfg.JFHost == nil || *cfg.JFHost != "http://new-host" {
		t.Errorf("JFHost = %v, want http://new-host", cfg.JFHost)
	}
	if !cfg.RequireLogin {
		t.Error("RequireLogin should be true")
	}
}
