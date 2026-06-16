package scheduler

import (
	"context"
	"encoding/json"
	"log"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/service/task"
	"github.com/robfig/cron/v3"
)

// TaskRunner wraps the functions needed to run each task by name.
type TaskRunner interface {
	RunByName(ctx context.Context, name string) error
}

// fullSyncTask is always scheduled (no opt-in toggle).
const fullSyncTask = "Full Jellyfin Sync"

// defaultFullSyncCron is used when no cronExpression is configured for the full sync.
const defaultFullSyncCron = "0 0 * * *"

// optionalDefaultCron holds the default cron for optional tasks (used only when enabled).
var optionalDefaultCron = map[string]string{
	"Recently Added Sync":                     "0 * * * *",
	"Backup":                                  "0 3 * * *",
	"Jellyfin Playback Reporting Plugin Sync": "0 4 * * *",
}

// Scheduler reads cron expressions from app_config and schedules tasks accordingly.
type Scheduler struct {
	repos  *repository.Container
	runner TaskRunner
	c      *cron.Cron
}

func New(repos *repository.Container, runner TaskRunner) *Scheduler {
	return &Scheduler{repos: repos, runner: runner}
}

// Start loads task settings from the DB and launches the cron scheduler.
// Call this once at startup. Call Reload() to pick up settings changes at runtime.
func (s *Scheduler) Start(ctx context.Context) {
	s.stop()
	s.c = cron.New()

	settings, err := s.loadTaskSettings(ctx)
	if err != nil {
		log.Printf("[scheduler] failed to load task settings: %v", err)
		return
	}

	for name, expr := range settings {
		if expr == "" {
			continue
		}
		taskName := name
		cronExpr := expr
		if _, err := s.c.AddFunc(cronExpr, func() {
			if err := s.runner.RunByName(context.Background(), taskName); err != nil {
				log.Printf("[scheduler] task %q failed: %v", taskName, err)
			}
		}); err != nil {
			log.Printf("[scheduler] invalid cron expression %q for task %q: %v", cronExpr, taskName, err)
			continue
		}
		log.Printf("[scheduler] scheduled %q with expression %q", taskName, cronExpr)
	}

	s.c.Start()
	log.Printf("[scheduler] started with %d task(s)", len(s.c.Entries()))
}

// Reload stops the existing scheduler and restarts it with fresh settings from the DB.
func (s *Scheduler) Reload(ctx context.Context) {
	s.Start(ctx)
}

func (s *Scheduler) stop() {
	if s.c != nil {
		s.c.Stop()
	}
}

func (s *Scheduler) loadTaskSettings(ctx context.Context) (map[string]string, error) {
	cfg, err := s.repos.Config.Get(ctx)
	if err != nil {
		return nil, err
	}

	var settings map[string]interface{}
	if len(cfg.Settings) > 0 {
		_ = json.Unmarshal(cfg.Settings, &settings)
	}
	if settings == nil {
		return map[string]string{}, nil
	}

	tasks, _ := settings["Tasks"].(map[string]interface{})
	if tasks == nil {
		return map[string]string{}, nil
	}

	result := make(map[string]string)

	// Full Jellyfin Sync: always scheduled, default 0 0 * * *.
	fullSyncExpr := defaultFullSyncCron
	if entry, ok := tasks[fullSyncTask].(map[string]interface{}); ok {
		if expr, ok := entry["cronExpression"].(string); ok && expr != "" {
			fullSyncExpr = expr
		}
	}
	result[fullSyncTask] = fullSyncExpr

	// Optional tasks: only schedule when enabled: true.
	for name := range optionalDefaultCron {
		entry, _ := tasks[name].(map[string]interface{})
		if entry == nil {
			continue
		}
		enabled, _ := entry["enabled"].(bool)
		if !enabled {
			continue
		}
		expr, _ := entry["cronExpression"].(string)
		if expr == "" {
			expr = optionalDefaultCron[name]
		}
		result[name] = expr
	}
	return result, nil
}

// Runner is a concrete TaskRunner backed by task.Service and a dispatch function.
type Runner struct {
	taskSvc  *task.Service
	dispatch func(ctx context.Context, name string) error
}

func NewRunner(taskSvc *task.Service, dispatch func(ctx context.Context, name string) error) *Runner {
	return &Runner{taskSvc: taskSvc, dispatch: dispatch}
}

func (r *Runner) RunByName(ctx context.Context, name string) error {
	return r.dispatch(ctx, name)
}
