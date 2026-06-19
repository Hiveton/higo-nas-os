package docker

import (
	"context"
	"testing"

	taskruntime "higoos/server-go/internal/tasks"
)

// TestMirrorPullToCentralMapsTerminalStatuses verifies the docker pull status
// is mirrored onto the adopted central task: running -> Update, completed ->
// Succeeded, failed -> Failed.
func TestMirrorPullToCentralMapsTerminalStatuses(t *testing.T) {
	mgr, err := taskruntime.NewManager("")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	svc := NewDevService()
	svc.AttachTaskRunner(mgr)

	cases := []struct {
		name   string
		status string
		want   taskruntime.Status
	}{
		{"running", "running", taskruntime.StatusRunning},
		{"completed", "completed", taskruntime.StatusSucceeded},
		{"failed", "failed", taskruntime.StatusFailed},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			adopted, err := mgr.Adopt(pullTaskKind, map[string]any{"image": "alpine"})
			if err != nil {
				t.Fatalf("adopt: %v", err)
			}
			svc.mirrorPullToCentral(mgr, adopted.ID, ImagePullStatus{
				Image:    "alpine",
				Status:   tc.status,
				Progress: 50,
				Message:  "msg",
				Error:    "boom",
			})
			got, err := mgr.Get(adopted.ID)
			if err != nil {
				t.Fatalf("get: %v", err)
			}
			if got.Status != tc.want {
				t.Fatalf("status = %s, want %s", got.Status, tc.want)
			}
		})
	}
}

// TestCancelPullInvokesStoredCancel verifies cancelPull resolves a central task
// id to its stored cancel func and calls it.
func TestCancelPullInvokesStoredCancel(t *testing.T) {
	svc := NewDevService()
	_, cancel := context.WithCancel(context.Background())
	called := make(chan struct{}, 1)
	svc.pullCancels["task-docker.pull-0001"] = func() {
		called <- struct{}{}
		cancel()
	}
	if err := svc.cancelPull("task-docker.pull-0001"); err != nil {
		t.Fatalf("cancelPull: %v", err)
	}
	select {
	case <-called:
	default:
		t.Fatal("expected stored cancel func to be invoked")
	}
	// Unknown id is a no-op, not an error.
	if err := svc.cancelPull("nope"); err != nil {
		t.Fatalf("cancelPull unknown: %v", err)
	}
}
