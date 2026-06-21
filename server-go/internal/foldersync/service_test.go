package foldersync

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func writeFile(t *testing.T, path, content string, mtime time.Time) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	if !mtime.IsZero() {
		if err := os.Chtimes(path, mtime, mtime); err != nil {
			t.Fatal(err)
		}
	}
}

func TestMirrorSyncCopiesFiles(t *testing.T) {
	t.Setenv("HIGO_NAS_ROOT", "")
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "a.txt"), "hello", time.Time{})
	writeFile(t, filepath.Join(src, "sub", "b.txt"), "world", time.Time{})

	svc := NewService()
	ctx := context.Background()
	pair, err := svc.Create(ctx, CreatePairRequest{Name: "mirror", Source: src, Target: dst, Direction: DirectionMirror})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Run(ctx, pair.ID, "tester"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "a.txt")); string(got) != "hello" {
		t.Fatalf("a.txt not mirrored: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(dst, "sub", "b.txt")); string(got) != "world" {
		t.Fatalf("sub/b.txt not mirrored: %q", got)
	}
	after, _ := svc.Get(ctx, pair.ID)
	if after.State != "已完成" || after.LastStats == nil || after.LastStats.Copied != 2 {
		t.Fatalf("unexpected post-run pair: state=%s stats=%+v", after.State, after.LastStats)
	}
}

func TestTwoWaySyncMergesAndDetectsConflict(t *testing.T) {
	t.Setenv("HIGO_NAS_ROOT", "")
	a := t.TempDir()
	b := t.TempDir()
	mt := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	// only-on-a → should land on b
	writeFile(t, filepath.Join(a, "onlyA.txt"), "A", time.Time{})
	// only-on-b → should land on a
	writeFile(t, filepath.Join(b, "onlyB.txt"), "B", time.Time{})
	// conflict: same path, different content, EQUAL mtime
	writeFile(t, filepath.Join(a, "clash.txt"), "from-a", mt)
	writeFile(t, filepath.Join(b, "clash.txt"), "from-b", mt)

	svc := NewService()
	ctx := context.Background()
	pair, err := svc.Create(ctx, CreatePairRequest{Name: "twoway", Source: a, Target: b, Direction: DirectionTwoWay, ConflictPolicy: ConflictNewer})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Run(ctx, pair.ID, "tester"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(b, "onlyA.txt")); string(got) != "A" {
		t.Fatalf("onlyA not merged to b: %q", got)
	}
	if got, _ := os.ReadFile(filepath.Join(a, "onlyB.txt")); string(got) != "B" {
		t.Fatalf("onlyB not merged to a: %q", got)
	}
	conflicts, _ := svc.Conflicts(ctx)
	if len(conflicts) != 1 || conflicts[0].RelPath != "clash.txt" {
		t.Fatalf("expected 1 clash.txt conflict, got %+v", conflicts)
	}
	after, _ := svc.Get(ctx, pair.ID)
	if after.State != "有冲突" {
		t.Fatalf("expected 有冲突 state, got %s", after.State)
	}

	// Resolve by taking the source side; the conflict file should become from-a.
	if _, err := svc.ResolveConflict(ctx, conflicts[0].ID, "source", "tester"); err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got, _ := os.ReadFile(filepath.Join(b, "clash.txt")); string(got) != "from-a" {
		t.Fatalf("conflict not resolved to source content: %q", got)
	}
}

func TestSelectiveSyncFilter(t *testing.T) {
	t.Setenv("HIGO_NAS_ROOT", "")
	src := t.TempDir()
	dst := t.TempDir()
	writeFile(t, filepath.Join(src, "docs", "keep.txt"), "k", time.Time{})
	writeFile(t, filepath.Join(src, "tmp", "skip.txt"), "s", time.Time{})

	svc := NewService()
	ctx := context.Background()
	pair, err := svc.Create(ctx, CreatePairRequest{Name: "sel", Source: src, Target: dst, Direction: DirectionMirror, Includes: []string{"docs/*"}})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if _, err := svc.Run(ctx, pair.ID, "tester"); err != nil {
		t.Fatalf("run: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "docs", "keep.txt")); err != nil {
		t.Fatalf("included file should sync: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "tmp", "skip.txt")); !os.IsNotExist(err) {
		t.Fatalf("excluded file should NOT sync")
	}
}

func TestCreateRejectsOutsideNASRoot(t *testing.T) {
	root := t.TempDir()
	t.Setenv("HIGO_NAS_ROOT", root)
	svc := NewService()
	_, err := svc.Create(context.Background(), CreatePairRequest{Name: "x", Source: "/etc", Target: filepath.Join(root, "t"), Direction: DirectionMirror})
	if err == nil {
		t.Fatalf("expected rejection for source outside NAS root")
	}
}
