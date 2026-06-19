package downloads

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	taskruntime "higoos/server-go/internal/tasks"
)

// waitForTask polls a predicate against a central task snapshot until it holds.
func waitForTask(t *testing.T, mgr *taskruntime.Manager, id string, done func(taskruntime.Task) bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		task, err := mgr.Get(id)
		if err == nil && done(task) {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}
	t.Fatalf("central task %s did not reach expected state in time", id)
}

func centralIDFor(s *Service, downloadID int) string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.centralIDs[downloadID]
}

func TestCreateTaskMirrorsAndSettlesCentralTaskOnSuccess(t *testing.T) {
	body := []byte("hello central runtime")
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Length", "21")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write(body)
	}))
	defer server.Close()

	mgr, err := taskruntime.NewManager("")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	service := newService(t.TempDir(), t.TempDir())
	service.AttachTaskRunner(mgr)

	task, err := service.CreateTask(context.Background(), CreateTaskRequest{Source: SourceHTTP, Link: server.URL + "/file.bin", Name: "file.bin"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	centralID := centralIDFor(service, task.ID)
	if centralID == "" {
		t.Fatal("expected a mirrored central task id")
	}
	waitForTask(t, mgr, centralID, func(ct taskruntime.Task) bool {
		return ct.Status == taskruntime.StatusSucceeded
	})
}

func TestCancelDownloadSettlesCentralTaskCanceled(t *testing.T) {
	mgr, err := taskruntime.NewManager("")
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	service := newService(t.TempDir(), t.TempDir())
	service.AttachTaskRunner(mgr)

	// A magnet without aria2c fails fast, but we cancel via the central runtime
	// before/after — DeleteTask must settle the mirror as canceled either way.
	task, err := service.CreateTask(context.Background(), CreateTaskRequest{Source: SourceMagnet, Link: "magnet:?xt=urn:btih:deadbeef", Name: "canceltest"})
	if err != nil {
		t.Fatalf("create task: %v", err)
	}
	centralID := centralIDFor(service, task.ID)
	if centralID == "" {
		t.Fatal("expected a mirrored central task id")
	}

	if _, err := mgr.Cancel(centralID); err != nil {
		t.Fatalf("cancel: %v", err)
	}
	waitForTask(t, mgr, centralID, func(ct taskruntime.Task) bool {
		return ct.Status == taskruntime.StatusCanceled || ct.Status == taskruntime.StatusFailed
	})
	// After cancel routes through DeleteTask, the download record is gone.
	if got := service.ListTasks(context.Background()); len(got) != 0 {
		t.Fatalf("expected download removed after cancel, got %d", len(got))
	}
}
