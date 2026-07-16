package repository_test

import (
	"testing"

	"github.com/Jellystics/Jellystics/internal/testutil"
	"gorm.io/gorm"
)

// setupTestDB provisions a fresh migrated Postgres database (or skips if none is
// reachable). See internal/testutil.NewDB.
func setupTestDB(t *testing.T) *gorm.DB {
	return testutil.NewDB(t)
}
