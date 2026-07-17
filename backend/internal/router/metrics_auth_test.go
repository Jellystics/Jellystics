package router

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/gin-gonic/gin"
	"gorm.io/datatypes"
)

// metricsRouter builds a minimal gin engine with metricsAuth protecting /metrics.
func metricsRouter(t *testing.T) (*repository.Container, *gin.Engine) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/metrics", metricsAuth(repos), func(c *gin.Context) {
		c.Status(http.StatusOK)
	})
	return repos, r
}

// seedKey saves an API key into the existing app_config row.
func seedKey(t *testing.T, repos *repository.Container, key string) {
	t.Helper()
	ctx := context.Background()
	cfg, err := repos.Config.Get(ctx)
	if err != nil {
		t.Fatalf("seedKey Get: %v", err)
	}
	raw, _ := json.Marshal([]map[string]string{{"key": key, "name": "test"}})
	cfg.ApiKeys = datatypes.JSON(raw)
	if err := repos.Config.Save(ctx, cfg); err != nil {
		t.Fatalf("seedKey Save: %v", err)
	}
}

func doMetrics(t *testing.T, r *gin.Engine, authHeader string) int {
	t.Helper()
	req := httptest.NewRequest("GET", "/metrics", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

// TestMetricsAuth_NoHeader — no Authorization header → 401.
func TestMetricsAuth_NoHeader(t *testing.T) {
	repos, r := metricsRouter(t)
	seedKey(t, repos, "secret123")
	if code := doMetrics(t, r, ""); code != http.StatusUnauthorized {
		t.Errorf("no header: status = %d, want 401", code)
	}
}

// TestMetricsAuth_MalformedHeader — Authorization is not "Bearer ..." → 401.
func TestMetricsAuth_MalformedHeader(t *testing.T) {
	repos, r := metricsRouter(t)
	seedKey(t, repos, "secret123")
	for _, h := range []string{"secret123", "Token secret123", "Bearer", "bear secret123"} {
		if code := doMetrics(t, r, h); code != http.StatusUnauthorized {
			t.Errorf("malformed %q: status = %d, want 401", h, code)
		}
	}
}

// TestMetricsAuth_WrongToken — valid Bearer format but wrong key → 401.
func TestMetricsAuth_WrongToken(t *testing.T) {
	repos, r := metricsRouter(t)
	seedKey(t, repos, "secret123")
	if code := doMetrics(t, r, "Bearer wrong"); code != http.StatusUnauthorized {
		t.Errorf("wrong token: status = %d, want 401", code)
	}
}

// TestMetricsAuth_ValidToken — correct key → request passes through (200).
func TestMetricsAuth_ValidToken(t *testing.T) {
	repos, r := metricsRouter(t)
	seedKey(t, repos, "secret123")
	if code := doMetrics(t, r, "Bearer secret123"); code != http.StatusOK {
		t.Errorf("valid token: status = %d, want 200", code)
	}
}

// TestMetricsAuth_NoKeysConfigured — even with no keys, endpoint is not public → 401.
func TestMetricsAuth_NoKeysConfigured(t *testing.T) {
	_, r := metricsRouter(t)
	// No seedKey call — app_config.api_keys is empty.
	if code := doMetrics(t, r, "Bearer anything"); code != http.StatusUnauthorized {
		t.Errorf("no keys: status = %d, want 401", code)
	}
}
