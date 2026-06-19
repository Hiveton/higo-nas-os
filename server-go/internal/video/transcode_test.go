package video

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
)

func TestCreateTranscodeTaskRunsRealFFmpeg(t *testing.T) {
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		t.Skip("ffmpeg not available; skipping real transcode test")
	}
	ctx := context.Background()
	stateDir := t.TempDir()
	mediaDir := t.TempDir()
	source := filepath.Join(mediaDir, "Clip.2024.1080p.mp4")

	// Generate a real (tiny) input clip with ffmpeg.
	gen := exec.CommandContext(ctx, "ffmpeg", "-y",
		"-f", "lavfi", "-i", "testsrc=duration=1:size=320x240:rate=10",
		"-pix_fmt", "yuv420p", source)
	if out, err := gen.CombinedOutput(); err != nil {
		t.Fatalf("generate test clip: %v\n%s", err, out)
	}

	service, err := NewServiceWithStateDir(stateDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.CreateLibrary(ctx, CreateLibraryRequest{Name: "电影", Type: LibraryMovie, Paths: []string{mediaDir}}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Scan(ctx); err != nil {
		t.Fatal(err)
	}
	items, err := service.Items(ctx, "", "", "")
	if err != nil || len(items) == 0 {
		t.Fatalf("need a scanned item: err=%v len=%d", err, len(items))
	}
	itemID := items[0].ID

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("task manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	mgr.Start(runCtx)
	defer mgr.Stop()

	task, err := service.CreateTranscodeTask(ctx, CreateTaskRequest{ItemID: itemID, Profile: "480p H.264"})
	if err != nil {
		t.Fatalf("create transcode task: %v", err)
	}
	if task.Status != JobRunning {
		t.Fatalf("expected running task on creation, got %q", task.Status)
	}

	output := filepath.Join(filepath.Dir(source), ".transcoded", "Clip.2024.1080p.480p.mp4")
	deadline := time.Now().Add(20 * time.Second)
	for time.Now().Before(deadline) {
		all, _ := service.Tasks(ctx)
		for _, tk := range all {
			if tk.ID == task.ID && (tk.Status == JobDone || tk.Status == JobFailed) {
				if tk.Status != JobDone {
					t.Fatalf("transcode failed: %s", tk.Message)
				}
				if _, err := os.Stat(output); err != nil {
					t.Fatalf("transcoded output not produced: %v", err)
				}
				// The media item status must no longer be stuck at "转码中".
				items2, _ := service.Items(ctx, "", "", "")
				if len(items2) > 0 && strings.Contains(items2[0].Status, "转码中") {
					t.Fatalf("item status stuck mid-transcode: %q", items2[0].Status)
				}
				return
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	t.Fatal("transcode task did not finish in time")
}
