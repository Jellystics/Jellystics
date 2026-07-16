package config_test

import (
	"strings"
	"testing"

	"github.com/Jellystics/Jellystics/internal/config"
)

// clearEnv unsets every variable Load reads so each test starts clean. t.Setenv
// restores them (and this cleared state) at the end of the test.
func clearEnv(t *testing.T) {
	t.Helper()
	for _, k := range []string{
		"DATABASE_URL", "JWT_SECRET", "PORT",
		"POSTGRES_IP", "POSTGRES_PORT", "POSTGRES_USER", "POSTGRES_PASSWORD", "POSTGRES_DB",
		"JF_HOST", "JF_API_KEY",
	} {
		t.Setenv(k, "")
	}
}

// TestLoad_RequiresJWTSecret verifies Load errors when JWT_SECRET is absent.
func TestLoad_RequiresJWTSecret(t *testing.T) {
	clearEnv(t)
	if _, err := config.Load(); err == nil {
		t.Fatal("expected error when JWT_SECRET is missing")
	}
}

// TestLoad_Defaults verifies default Port and a DBUrl assembled from POSTGRES_*
// defaults when DATABASE_URL is unset.
func TestLoad_Defaults(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "s")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "3000" {
		t.Errorf("Port = %q, want 3000", cfg.Port)
	}
	if cfg.JWTSecret != "s" {
		t.Errorf("JWTSecret = %q, want s", cfg.JWTSecret)
	}
	if !strings.Contains(cfg.DBUrl, "host=localhost") || !strings.Contains(cfg.DBUrl, "dbname=jellystics") {
		t.Errorf("DBUrl = %q, want assembled from POSTGRES defaults", cfg.DBUrl)
	}
}

// TestLoad_DatabaseURLOverride verifies DATABASE_URL takes precedence over the
// assembled POSTGRES_* connection string.
func TestLoad_DatabaseURLOverride(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "s")
	t.Setenv("DATABASE_URL", "postgres://custom/url")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.DBUrl != "postgres://custom/url" {
		t.Errorf("DBUrl = %q, want the DATABASE_URL override", cfg.DBUrl)
	}
}

// TestLoad_CustomValues verifies explicit env values flow through.
func TestLoad_CustomValues(t *testing.T) {
	clearEnv(t)
	t.Setenv("JWT_SECRET", "sekret")
	t.Setenv("PORT", "8080")
	t.Setenv("JF_HOST", "http://jf:8096")
	t.Setenv("JF_API_KEY", "apikey")
	t.Setenv("POSTGRES_IP", "db.internal")

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != "8080" {
		t.Errorf("Port = %q, want 8080", cfg.Port)
	}
	if cfg.JFHost != "http://jf:8096" {
		t.Errorf("JFHost = %q", cfg.JFHost)
	}
	if cfg.JFApiKey != "apikey" {
		t.Errorf("JFApiKey = %q", cfg.JFApiKey)
	}
	if !strings.Contains(cfg.DBUrl, "host=db.internal") {
		t.Errorf("DBUrl = %q, want host=db.internal", cfg.DBUrl)
	}
}
