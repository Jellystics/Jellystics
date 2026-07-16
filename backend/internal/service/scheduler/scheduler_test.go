package scheduler

import (
	"context"
	"errors"
	"testing"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/task"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/Jellystics/Jellystics/internal/ws"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

// seedSettings writes the given JSON into app_config.settings (row ID=1, which
// the migration seeds). settingsJSON must be a valid JSON object string.
func seedSettings(t *testing.T, db *gorm.DB, settingsJSON string) {
	t.Helper()
	if err := db.Exec(`UPDATE app_config SET settings = ?::jsonb WHERE "ID" = 1`, settingsJSON).Error; err != nil {
		t.Fatalf("seed settings: %v", err)
	}
}

func newScheduler(t *testing.T) (*Scheduler, *gorm.DB) {
	db := testutil.NewDB(t)
	return New(repository.New(db), nil), db
}

func TestLoadTaskSettings(t *testing.T) {
	tests := []struct {
		name         string
		settingsJSON string
		want         map[string]string
	}{
		{
			// Empty settings object -> Tasks nil -> empty map.
			// NOTE: Full Sync is NOT added on this path (verified against code).
			name:         "empty settings object",
			settingsJSON: `{}`,
			want:         map[string]string{},
		},
		{
			// Settings present but no Tasks key -> empty map, no Full Sync.
			name:         "settings without Tasks key",
			settingsJSON: `{"other":"value"}`,
			want:         map[string]string{},
		},
		{
			// Tasks present but empty -> Full Sync added with its default cron.
			name:         "empty Tasks map defaults Full Sync",
			settingsJSON: `{"Tasks":{}}`,
			want: map[string]string{
				"Full Jellyfin Sync": defaultFullSyncCron,
			},
		},
		{
			// Full Sync with a custom cronExpression overrides the default.
			name:         "Full Sync override cron",
			settingsJSON: `{"Tasks":{"Full Jellyfin Sync":{"cronExpression":"30 2 * * *"}}}`,
			want: map[string]string{
				"Full Jellyfin Sync": "30 2 * * *",
			},
		},
		{
			// Empty cronExpression falls back to default for Full Sync.
			name:         "Full Sync empty cron falls back to default",
			settingsJSON: `{"Tasks":{"Full Jellyfin Sync":{"cronExpression":""}}}`,
			want: map[string]string{
				"Full Jellyfin Sync": defaultFullSyncCron,
			},
		},
		{
			// Optional task enabled with no cron -> uses its default cron.
			name:         "optional enabled uses default cron",
			settingsJSON: `{"Tasks":{"Recently Added Sync":{"enabled":true}}}`,
			want: map[string]string{
				"Full Jellyfin Sync":  defaultFullSyncCron,
				"Recently Added Sync": optionalDefaultCron["Recently Added Sync"],
			},
		},
		{
			// Optional task enabled with custom cron -> uses configured cron.
			name:         "optional enabled with custom cron",
			settingsJSON: `{"Tasks":{"Backup":{"enabled":true,"cronExpression":"15 5 * * *"}}}`,
			want: map[string]string{
				"Full Jellyfin Sync": defaultFullSyncCron,
				"Backup":             "15 5 * * *",
			},
		},
		{
			// Optional task present but disabled -> excluded.
			name:         "optional disabled excluded",
			settingsJSON: `{"Tasks":{"Backup":{"enabled":false,"cronExpression":"15 5 * * *"}}}`,
			want: map[string]string{
				"Full Jellyfin Sync": defaultFullSyncCron,
			},
		},
		{
			// All optional tasks enabled alongside a Full Sync override.
			name: "all optionals enabled",
			settingsJSON: `{"Tasks":{
				"Full Jellyfin Sync":{"cronExpression":"0 1 * * *"},
				"Recently Added Sync":{"enabled":true},
				"Backup":{"enabled":true,"cronExpression":"15 5 * * *"},
				"Jellyfin Playback Reporting Plugin Sync":{"enabled":true}
			}}`,
			want: map[string]string{
				"Full Jellyfin Sync":                      "0 1 * * *",
				"Recently Added Sync":                     optionalDefaultCron["Recently Added Sync"],
				"Backup":                                  "15 5 * * *",
				"Jellyfin Playback Reporting Plugin Sync": optionalDefaultCron["Jellyfin Playback Reporting Plugin Sync"],
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			s, db := newScheduler(t)
			seedSettings(t, db, tc.settingsJSON)

			got, err := s.loadTaskSettings(context.Background())
			if err != nil {
				t.Fatalf("loadTaskSettings: %v", err)
			}
			if len(got) != len(tc.want) {
				t.Fatalf("result size = %d (%v), want %d (%v)", len(got), got, len(tc.want), tc.want)
			}
			for name, wantExpr := range tc.want {
				if got[name] != wantExpr {
					t.Errorf("task %q cron = %q, want %q", name, got[name], wantExpr)
				}
			}
		})
	}
}

// TestLoadTaskSettings_EmptySettingsColumn covers the len(cfg.Settings)==0 path.
// Config.Get reads ID=1; we set settings to an empty JSONB value to exercise the
// "settings nil -> empty map" branch.
func TestLoadTaskSettings_NullSettingsBranch(t *testing.T) {
	s, db := newScheduler(t)
	// Persist an AppConfig with an explicitly empty settings body via Save so we
	// exercise the repo round-trip the same way production does.
	cfg := &models.AppConfig{ID: 1, Settings: datatypes.JSON([]byte(`{}`)), ApiKeys: datatypes.JSON([]byte(`[]`))}
	if err := repository.New(db).Config.Save(context.Background(), cfg); err != nil {
		t.Fatalf("save config: %v", err)
	}
	got, err := s.loadTaskSettings(context.Background())
	if err != nil {
		t.Fatalf("loadTaskSettings: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("empty settings -> result = %v, want empty map", got)
	}
}

// TestRunByName_Delegates proves NewRunner.RunByName forwards to the dispatch
// func and returns its result verbatim.
func TestRunByName_Delegates(t *testing.T) {
	taskSvc := task.New(repository.New(testutil.NewDB(t)), ws.NewHub())

	var gotName string
	wantErr := errors.New("dispatch failed")
	runner := NewRunner(taskSvc, func(ctx context.Context, name string) error {
		gotName = name
		return wantErr
	})

	err := runner.RunByName(context.Background(), "Full Jellyfin Sync")
	if !errors.Is(err, wantErr) {
		t.Errorf("RunByName error = %v, want %v", err, wantErr)
	}
	if gotName != "Full Jellyfin Sync" {
		t.Errorf("dispatch received name %q, want %q", gotName, "Full Jellyfin Sync")
	}

	// Success path returns nil.
	okRunner := NewRunner(taskSvc, func(ctx context.Context, name string) error { return nil })
	if err := okRunner.RunByName(context.Background(), "Backup"); err != nil {
		t.Errorf("RunByName success error = %v, want nil", err)
	}
}
