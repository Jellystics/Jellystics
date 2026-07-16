// Package testutil provides shared helpers for integration tests that need a
// real Postgres database. It is only imported from _test files.
package testutil

import (
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/database"
	"github.com/Jellystics/Jellystics/internal/migrations"
	"gorm.io/gorm"
)

func env(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

// dsn builds a libpq DSN for dbName using the same env vars as the production
// config (defaults match the local `task db` container).
func dsn(dbName string) string {
	return fmt.Sprintf(
		"host=%s port=%s user=%s password=%s dbname=%s sslmode=disable TimeZone=UTC",
		env("POSTGRES_IP", "localhost"),
		env("POSTGRES_PORT", "5432"),
		env("POSTGRES_USER", "postgres"),
		env("POSTGRES_PASSWORD", "mypassword"),
		dbName,
	)
}

// NewDB connects to a real Postgres, provisions a fresh throwaway database, runs
// the full embedded migration set against it, and returns a GORM handle. The
// database is dropped when the test finishes.
//
// If Postgres is not reachable the test is skipped (so `go test ./...` stays
// green without a database) rather than failed.
func NewDB(t *testing.T) *gorm.DB {
	t.Helper()

	adminDB := env("POSTGRES_DB", "postgres")
	admin, err := database.Connect(dsn(adminDB))
	if err != nil {
		t.Skipf("postgres unavailable (connect): %v", err)
	}
	if err := admin.Exec("SELECT 1").Error; err != nil {
		t.Skipf("postgres unavailable (ping): %v", err)
	}

	name := fmt.Sprintf("jellystics_test_%d", time.Now().UnixNano())
	if err := admin.Exec("CREATE DATABASE " + name).Error; err != nil {
		t.Skipf("cannot create test database: %v", err)
	}

	db, err := database.Connect(dsn(name))
	if err != nil {
		t.Fatalf("connect to test database: %v", err)
	}
	if err := database.Migrate(db, migrations.SQL, "sql"); err != nil {
		t.Fatalf("run migrations: %v", err)
	}

	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
		admin.Exec(`SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname = ?`, name)
		admin.Exec("DROP DATABASE IF EXISTS " + name)
		if sqlDB, err := admin.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})

	return db
}
