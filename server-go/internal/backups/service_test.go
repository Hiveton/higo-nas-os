package backups

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"higoos/server-go/internal/tasks"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

func TestSyncTreeCopiesNewFilesAndSkipsUnchanged(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "a.txt"), "hello")
	writeFile(t, filepath.Join(src, "nested", "b.txt"), "world")

	res, err := syncTree(src, dst)
	if err != nil {
		t.Fatalf("first sync: %v", err)
	}
	if res.Copied != 2 {
		t.Fatalf("expected 2 files copied, got %d", res.Copied)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "nested", "b.txt")); string(got) != "world" {
		t.Fatalf("nested file not copied: %q", got)
	}

	// Second run is incremental: unchanged files are skipped.
	res2, err := syncTree(src, dst)
	if err != nil {
		t.Fatalf("second sync: %v", err)
	}
	if res2.Copied != 0 || res2.Skipped != 2 {
		t.Fatalf("expected incremental skip (copied=0 skipped=2), got copied=%d skipped=%d", res2.Copied, res2.Skipped)
	}

	// Changing a file makes it copy again.
	writeFile(t, filepath.Join(src, "a.txt"), "changed-content")
	res3, err := syncTree(src, dst)
	if err != nil {
		t.Fatalf("third sync: %v", err)
	}
	if res3.Copied != 1 {
		t.Fatalf("expected 1 changed file re-copied, got %d", res3.Copied)
	}
}

func TestSyncTreeFailsOnMissingSource(t *testing.T) {
	if _, err := syncTree(filepath.Join(t.TempDir(), "does-not-exist"), t.TempDir()); err == nil {
		t.Fatal("expected error for missing source")
	}
}

func TestRunExecutesRealBackupSyncViaRunner(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "doc.txt"), "backup me")

	service := NewService()
	// Point a demo job at real temp directories.
	service.mu.Lock()
	service.jobs[0].Source = src
	service.jobs[0].Target = dst
	jobID := service.jobs[0].ID
	service.mu.Unlock()

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	if _, err := service.Run(ctx, jobID); err != nil {
		t.Fatalf("run: %v", err)
	}

	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := service.Jobs(ctx)
		if jobs[0].State == "已完成" {
			if _, err := os.Stat(filepath.Join(dst, "doc.txt")); err != nil {
				t.Fatalf("file should have been backed up: %v", err)
			}
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	jobs, _ := service.Jobs(ctx)
	t.Fatalf("backup run did not complete, final state=%q health=%q", jobs[0].State, jobs[0].Health)
}

func TestVerifyDetectsMatchingAndCorruptedBackups(t *testing.T) {
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "doc.txt"), "verify me")

	service := NewService()
	service.mu.Lock()
	service.jobs[0].Source = src
	service.jobs[0].Target = dst
	jobID := service.jobs[0].ID
	service.mu.Unlock()

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	// Back up, then verify: should pass.
	if _, err := service.Run(ctx, jobID); err != nil {
		t.Fatalf("run: %v", err)
	}
	waitState := func(want string) string {
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			jobs, _ := service.Jobs(ctx)
			if jobs[0].State == want {
				return jobs[0].Health
			}
			time.Sleep(5 * time.Millisecond)
		}
		jobs, _ := service.Jobs(ctx)
		t.Fatalf("did not reach state %q, final=%q", want, jobs[0].State)
		return ""
	}
	waitState("已完成")

	if _, err := service.Verify(ctx, jobID); err != nil {
		t.Fatalf("verify: %v", err)
	}
	if health := waitState("已完成"); health != "校验通过" {
		t.Fatalf("expected 校验通过, got %q", health)
	}

	// Corrupt the backup copy and verify again: should fail.
	writeFile(t, filepath.Join(dst, "doc.txt"), "tampered!!")
	if _, err := service.Verify(ctx, jobID); err != nil {
		t.Fatalf("verify after corruption: %v", err)
	}
	waitState("校验未通过")
}

func TestRunSurfacesBlockedStateForInvalidPaths(t *testing.T) {
	service := NewService() // default demo jobs have non-filesystem source paths
	jobID := service.jobs[0].ID

	mgr, err := tasks.NewManager("", tasks.WithWorkers(1), tasks.WithDispatchInterval(20*time.Millisecond))
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}
	service.AttachTaskRunner(mgr)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	mgr.Start(ctx)
	defer mgr.Stop()

	if _, err := service.Run(ctx, jobID); err != nil {
		t.Fatalf("run: %v", err)
	}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		jobs, _ := service.Jobs(ctx)
		if jobs[0].State == "源路径不可用" {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("expected demo job to surface blocked state for invalid source path")
}
