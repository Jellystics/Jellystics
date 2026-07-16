package task

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/Jellystics/Jellystics/internal/repository"
	"github.com/Jellystics/Jellystics/internal/testutil"
	"github.com/Jellystics/Jellystics/internal/ws"
)

// newSvc builds a task.Service backed by a fresh test DB. The hub has no
// connected clients so Emit is a safe no-op.
func newSvc(t *testing.T) (*Service, *repository.Container) {
	t.Helper()
	db := testutil.NewDB(t)
	repos := repository.New(db)
	return New(repos, ws.NewHub()), repos
}

// findLog returns the jf_logging row with the given name (most recent first).
func findLog(t *testing.T, repos *repository.Container, name string) (result string, ok bool) {
	t.Helper()
	logs, err := repos.Log.List(context.Background(), 50)
	if err != nil {
		t.Fatalf("Log.List: %v", err)
	}
	for _, l := range logs {
		if l.Name != nil && *l.Name == name {
			if l.Result != nil {
				return *l.Result, true
			}
			return "", true
		}
	}
	return "", false
}

func TestRun_StatusTransitions(t *testing.T) {
	svc, _ := newSvc(t)

	if got := svc.Status(); got != StatusIdle {
		t.Fatalf("initial status = %q, want %q", got, StatusIdle)
	}

	var duringStatus Status
	err := svc.Run(context.Background(), "Full Jellyfin Sync", func(ctx context.Context) error {
		duringStatus = svc.Status()
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if duringStatus != StatusRunning {
		t.Errorf("status during fn = %q, want %q", duringStatus, StatusRunning)
	}
	if got := svc.Status(); got != StatusIdle {
		t.Errorf("status after Run = %q, want %q", got, StatusIdle)
	}
}

func TestRun_SuccessWritesLog(t *testing.T) {
	svc, repos := newSvc(t)

	ran := false
	err := svc.Run(context.Background(), "Backup", func(ctx context.Context) error {
		ran = true
		return nil
	})
	if err != nil {
		t.Fatalf("Run returned error: %v", err)
	}
	if !ran {
		t.Fatal("fn was not executed")
	}
	result, ok := findLog(t, repos, "Backup")
	if !ok {
		t.Fatal("no jf_logging row written for Backup")
	}
	if result != "success" {
		t.Errorf("Result = %q, want success", result)
	}
}

func TestRun_ErrorWritesFailedLogAndReturnsError(t *testing.T) {
	svc, repos := newSvc(t)

	wantErr := errors.New("boom")
	err := svc.Run(context.Background(), "Recently Added Sync", func(ctx context.Context) error {
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("Run error = %v, want %v", err, wantErr)
	}
	result, ok := findLog(t, repos, "Recently Added Sync")
	if !ok {
		t.Fatal("no jf_logging row written for failed task")
	}
	if result != "failed" {
		t.Errorf("Result = %q, want failed", result)
	}
	// Status must return to idle even after an error.
	if got := svc.Status(); got != StatusIdle {
		t.Errorf("status after failed Run = %q, want %q", got, StatusIdle)
	}
}

// TestRun_Reentrancy proves a second Run while one is in flight is a no-op:
// its fn never executes and Run returns nil immediately.
func TestRun_Reentrancy(t *testing.T) {
	svc, _ := newSvc(t)

	release := make(chan struct{})
	started := make(chan struct{})
	firstDone := make(chan error, 1)

	go func() {
		firstDone <- svc.Run(context.Background(), "long-task", func(ctx context.Context) error {
			close(started)
			<-release // block until the test lets it finish
			return nil
		})
	}()

	// Wait for the first task to be running.
	select {
	case <-started:
	case <-time.After(2 * time.Second):
		t.Fatal("first task did not start in time")
	}

	// Second Run must return nil immediately without executing its fn.
	secondRan := false
	err := svc.Run(context.Background(), "second-task", func(ctx context.Context) error {
		secondRan = true
		return errors.New("should never run")
	})
	if err != nil {
		t.Errorf("re-entrant Run error = %v, want nil", err)
	}
	if secondRan {
		t.Error("second fn executed while first task was running; re-entrancy guard failed")
	}

	// Let the first task complete.
	close(release)
	select {
	case err := <-firstDone:
		if err != nil {
			t.Errorf("first Run error = %v, want nil", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("first task did not finish in time")
	}

	if got := svc.Status(); got != StatusIdle {
		t.Errorf("status after completion = %q, want %q", got, StatusIdle)
	}
}

// TestSetFireFunc_SuccessEvents asserts task_start and task_complete fire on a
// successful run. fireEvent runs the callback in a goroutine, so events are
// collected over a buffered channel and drained with a timeout.
func TestSetFireFunc_SuccessEvents(t *testing.T) {
	svc, _ := newSvc(t)

	events := make(chan string, 8)
	svc.SetFireFunc(func(ctx context.Context, eventType string, data map[string]any) {
		events <- eventType
	})

	if err := svc.Run(context.Background(), "Backup", func(ctx context.Context) error {
		return nil
	}); err != nil {
		t.Fatalf("Run error: %v", err)
	}

	got := collectEvents(t, events, 2)
	if !got["task_start"] {
		t.Error("task_start event did not fire")
	}
	if !got["task_complete"] {
		t.Error("task_complete event did not fire")
	}
	if got["task_failed"] {
		t.Error("task_failed fired on a successful run")
	}
}

// TestSetFireFunc_FailureEvents asserts task_start and task_failed fire (and
// task_complete does not) when the task fn returns an error.
func TestSetFireFunc_FailureEvents(t *testing.T) {
	svc, _ := newSvc(t)

	events := make(chan string, 8)
	svc.SetFireFunc(func(ctx context.Context, eventType string, data map[string]any) {
		events <- eventType
	})

	_ = svc.Run(context.Background(), "Recently Added Sync", func(ctx context.Context) error {
		return errors.New("kaboom")
	})

	got := collectEvents(t, events, 2)
	if !got["task_start"] {
		t.Error("task_start event did not fire")
	}
	if !got["task_failed"] {
		t.Error("task_failed event did not fire")
	}
	if got["task_complete"] {
		t.Error("task_complete fired on a failed run")
	}
}

// collectEvents drains at least want events (or times out) and returns the set
// of event types seen. It waits a short extra window to catch any stragglers so
// we can assert on events that must NOT appear.
func collectEvents(t *testing.T, ch <-chan string, want int) map[string]bool {
	t.Helper()
	seen := map[string]bool{}
	deadline := time.After(3 * time.Second)
	for len(seen) < want {
		select {
		case ev := <-ch:
			seen[ev] = true
		case <-deadline:
			t.Fatalf("timed out waiting for events; got %v, want at least %d", seen, want)
		}
	}
	// Drain a brief tail to catch any unexpected extra events.
	settle := time.After(150 * time.Millisecond)
	for {
		select {
		case ev := <-ch:
			seen[ev] = true
		case <-settle:
			return seen
		}
	}
}
