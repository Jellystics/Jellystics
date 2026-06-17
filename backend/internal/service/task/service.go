package task

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/Jellystics/Jellystics/internal/database/models"
	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/ws"
	"github.com/google/uuid"
)

type Status string

const (
	StatusIdle    Status = "idle"
	StatusRunning Status = "running"
)

// Service manages background tasks (full sync, session watchdog, etc.)
type Service struct {
	repos  *repository.Container
	hub    *ws.Hub
	mu     sync.Mutex
	status Status
}

func New(repos *repository.Container, hub *ws.Hub) *Service {
	return &Service{
		repos:  repos,
		hub:    hub,
		status: StatusIdle,
	}
}

func (s *Service) Status() Status {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.status
}

func (s *Service) setStatus(st Status) {
	s.mu.Lock()
	s.status = st
	s.mu.Unlock()
}

func (s *Service) log(format string, args ...any) {
	msg := fmt.Sprintf(format, args...)
	log.Printf("[task] %s", msg)
	s.hub.Emit("TaskLog", msg)
}

// Run executes fn if no other task is running.
// It inserts a jf_logging entry on start and updates it on completion (matching old Node.js behaviour).
func (s *Service) Run(ctx context.Context, name string, fn func(context.Context) error) error {
	s.mu.Lock()
	if s.status == StatusRunning {
		s.mu.Unlock()
		return nil
	}
	s.status = StatusRunning
	s.mu.Unlock()

	s.log("[%s] Task started", name)

	// Insert start log entry into jf_logging.
	logID := uuid.New().String()
	startTime := time.Now().UTC().Format(time.RFC3339)
	taskType := ptr("Task")
	execType := ptr("Manual")
	resultRunning := ptr("running")
	dur := int64(0)
	initLog := ptr(`[{}]`)

	startEntry := &models.JFLogging{
		Id:            logID,
		Name:          ptr(name),
		Type:          taskType,
		ExecutionType: execType,
		Duration:      &dur,
		TimeRun:       &startTime,
		Log:           initLog,
		Result:        resultRunning,
	}
	_ = s.repos.Log.Insert(ctx, startEntry)

	start := time.Now()
	err := fn(ctx)
	elapsed := time.Since(start)
	s.setStatus(StatusIdle)

	// Update the log entry with final result.
	result := "success"
	logMessage := fmt.Sprintf(`[{"color":"lawngreen","Message":"Task completed in %dms"}]`, elapsed.Milliseconds())
	if err != nil {
		result = "failed"
		logMessage = fmt.Sprintf(`[{"color":"red","Message":"Task failed after %dms: %s"}]`, elapsed.Milliseconds(), err.Error())
		s.log("[%s] Task failed after %dms: %s", name, elapsed.Milliseconds(), err.Error())
	} else {
		s.log("[%s] Task completed in %dms", name, elapsed.Milliseconds())
	}

	finalDur := int64(elapsed.Seconds())
	finalEntry := &models.JFLogging{
		Id:            logID,
		Name:          ptr(name),
		Type:          taskType,
		ExecutionType: execType,
		Duration:      &finalDur,
		TimeRun:       &startTime,
		Log:           ptr(logMessage),
		Result:        ptr(result),
	}
	_ = s.repos.Log.Upsert(ctx, finalEntry)

	// Emit task-specific event so the frontend SocketNotifier can show toast notifications.
	taskEventNames := map[string]string{
		"Full Jellyfin Sync":                          "FullSyncTask",
		"Recently Added Sync":                         "PartialSyncTask",
		"Backup":                                      "BackupTask",
		"Jellyfin Playback Reporting Plugin Sync":     "PlaybackSyncTask",
	}
	if eventName, ok := taskEventNames[name]; ok {
		if err != nil {
			s.hub.Emit(eventName, map[string]string{"type": "Error", "message": "Task failed: " + err.Error()})
		} else {
			s.hub.Emit(eventName, map[string]string{"type": "Success", "message": fmt.Sprintf("Task completed in %dms", elapsed.Milliseconds())})
		}
	}

	return err
}

func ptr(s string) *string { return &s }
