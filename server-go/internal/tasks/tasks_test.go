package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"sync/atomic"
	"testing"
	"time"
)

// waitFor polls until cond is true or the deadline elapses.
func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatalf("condition not met within deadline")
}

func newTestManager(t *testing.T, statePath string) *Manager {
	t.Helper()
	m, err := NewManager(statePath, WithWorkers(2), WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	return m
}

func TestEnqueueRunsHandlerToSuccess(t *testing.T) {
	m := newTestManager(t, "")
	m.Register("echo", func(ctx context.Context, h *Handle) (json.RawMessage, error) {
		var in struct {
			Value string `json:"value"`
		}
		if err := h.Unmarshal(&in); err != nil {
			return nil, err
		}
		h.Progress(50, "halfway")
		return json.Marshal(map[string]string{"echo": in.Value})
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	task, err := m.Enqueue("echo", map[string]string{"value": "hi"})
	if err != nil {
		t.Fatalf("enqueue: %v", err)
	}
	waitFor(t, func() bool {
		got, _ := m.Get(task.ID)
		return got.Status == StatusSucceeded
	})
	got, _ := m.Get(task.ID)
	if got.Progress != 100 {
		t.Fatalf("expected progress 100, got %d", got.Progress)
	}
	if string(got.Result) != `{"echo":"hi"}` {
		t.Fatalf("unexpected result: %s", got.Result)
	}
	if got.Attempts != 1 {
		t.Fatalf("expected 1 attempt, got %d", got.Attempts)
	}
}

func TestHandlerErrorMarksFailed(t *testing.T) {
	m := newTestManager(t, "")
	m.Register("boom", func(ctx context.Context, h *Handle) (json.RawMessage, error) {
		return nil, errors.New("kaboom")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	task, _ := m.Enqueue("boom", nil)
	waitFor(t, func() bool {
		got, _ := m.Get(task.ID)
		return got.Status == StatusFailed
	})
	got, _ := m.Get(task.ID)
	if got.Error != "kaboom" {
		t.Fatalf("expected error 'kaboom', got %q", got.Error)
	}
}

func TestPanicInHandlerIsRecovered(t *testing.T) {
	m := newTestManager(t, "")
	m.Register("panic", func(ctx context.Context, h *Handle) (json.RawMessage, error) {
		panic("oops")
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	task, _ := m.Enqueue("panic", nil)
	waitFor(t, func() bool {
		got, _ := m.Get(task.ID)
		return got.Status == StatusFailed
	})
	got, _ := m.Get(task.ID)
	if got.Error == "" {
		t.Fatalf("expected panic recorded as error")
	}
}

func TestUnknownKindFailsTask(t *testing.T) {
	m := newTestManager(t, "")
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m.Start(ctx)
	defer m.Stop()

	task, _ := m.Enqueue("nonexistent", nil)
	waitFor(t, func() bool {
		got, _ := m.Get(task.ID)
		return got.Status == StatusFailed
	})
}

func TestCancelQueuedTask(t *testing.T) {
	// No Start: task stays queued so we can cancel it deterministically.
	m := newTestManager(t, "")
	task, _ := m.Enqueue("later", nil)
	canceled, err := m.Cancel(task.ID)
	if err != nil {
		t.Fatalf("cancel: %v", err)
	}
	if canceled.Status != StatusCanceled {
		t.Fatalf("expected canceled, got %s", canceled.Status)
	}
}

func TestRestartRecoveryRequeuesRunningTasks(t *testing.T) {
	dir := t.TempDir()
	statePath := dir + "/tasks.json"

	// First manager: persist a task stuck in "running" by writing through Enqueue
	// then forcing the on-disk state to running (simulating a crash mid-flight).
	m1, _ := NewManager(statePath, WithDispatchInterval(20*time.Millisecond))
	task, _ := m1.Enqueue("resume", nil)
	// Force on-disk record to running, as if the process died mid-execution.
	m1.mu.Lock()
	rec := m1.tasks[task.ID]
	rec.Status = StatusRunning
	m1.tasks[task.ID] = rec
	_ = m1.saveLocked()
	m1.mu.Unlock()

	// Second manager loads the same state file and recovers on Start.
	m2, err := NewManager(statePath, WithWorkers(1), WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new manager 2: %v", err)
	}
	var ran int32
	m2.Register("resume", func(ctx context.Context, h *Handle) (json.RawMessage, error) {
		atomic.AddInt32(&ran, 1)
		return nil, nil
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	m2.Start(ctx)
	defer m2.Stop()

	waitFor(t, func() bool {
		got, _ := m2.Get(task.ID)
		return got.Status == StatusSucceeded
	})
	if atomic.LoadInt32(&ran) != 1 {
		t.Fatalf("recovered task should run exactly once, ran=%d", ran)
	}
}

func TestPersistenceSurvivesReload(t *testing.T) {
	dir := t.TempDir()
	statePath := dir + "/tasks.json"
	m1, _ := NewManager(statePath)
	task, _ := m1.Enqueue("persist", map[string]int{"n": 7})

	m2, err := NewManager(statePath)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := m2.Get(task.ID)
	if err != nil {
		t.Fatalf("get after reload: %v", err)
	}
	if got.Kind != "persist" || got.Status != StatusQueued {
		t.Fatalf("unexpected reloaded task: %#v", got)
	}
}

func TestListNewestFirstAndFilter(t *testing.T) {
	m := newTestManager(t, "")
	a, _ := m.Enqueue("kindA", nil)
	_, _ = m.Enqueue("kindB", nil)
	c, _ := m.Enqueue("kindA", nil)

	all := m.List("")
	if len(all) != 3 {
		t.Fatalf("expected 3 tasks, got %d", len(all))
	}
	onlyA := m.List("kindA")
	if len(onlyA) != 2 {
		t.Fatalf("expected 2 kindA tasks, got %d", len(onlyA))
	}
	// Newest first: c before a.
	if onlyA[0].ID != c.ID || onlyA[1].ID != a.ID {
		t.Fatalf("expected newest-first order, got %s then %s", onlyA[0].ID, onlyA[1].ID)
	}
}
