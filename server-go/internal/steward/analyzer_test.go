package steward

import (
	"context"
	"testing"
	"time"
)

func TestAnalyzeDuplicatesAndArchive(t *testing.T) {
	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	files := []FileInfo{
		{ID: "f-a", Name: "报告.docx", Path: "/家庭空间/a/报告.docx", SizeBytes: 100, Modified: now},
		{ID: "f-b", Name: "报告.docx", Path: "/家庭空间/b/报告.docx", SizeBytes: 100, Modified: now.Add(-time.Hour)},             // older duplicate
		{ID: "f-vid", Name: "video.mp4", Path: "/照片与视频/m/video.mp4", SizeBytes: 200 << 20, Modified: now},                // large
		{ID: "f-old", Name: "old.txt", Path: "/家庭空间/o/old.txt", SizeBytes: 10, Modified: now.Add(-400 * 24 * time.Hour)}, // stale
	}
	out := Analyze(files, now)
	if len(out) != 2 {
		t.Fatalf("expected duplicate + archive suggestions, got %d: %#v", len(out), out)
	}
	ids := map[string]Suggestion{}
	for _, s := range out {
		ids[s.ID] = s
	}
	dup, ok := ids["dup-files"]
	if !ok {
		t.Fatal("missing dup-files suggestion")
	}
	// The newer duplicate (f-a) is kept; only the older one (f-b) is deleted.
	if len(dup.Operations) != 1 || dup.Operations[0].Type != "delete" || dup.Operations[0].FileID != "f-b" {
		t.Errorf("dup suggestion should delete only the older duplicate f-b: %#v", dup.Operations)
	}
	archive, ok := ids["archive-candidates"]
	if !ok || archive.Risk != RiskLow {
		t.Fatalf("missing/incorrect archive suggestion: %#v", archive)
	}
	// Both the large and the stale file are proposed for a move to 备份归档.
	if len(archive.Operations) != 2 {
		t.Fatalf("archive should propose 2 moves, got %#v", archive.Operations)
	}
	for _, op := range archive.Operations {
		if op.Type != "move" || op.Dest != archiveSpace {
			t.Errorf("archive op should be a move to %s: %#v", archiveSpace, op)
		}
	}
}

// Files already in the archive space, and files without ids, are not flagged.
func TestAnalyzeSkipsArchivedAndUnidentified(t *testing.T) {
	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	files := []FileInfo{
		{ID: "f-archived", Name: "big.bin", Path: "/备份归档/big.bin", Space: archiveSpace, SizeBytes: 300 << 20, Modified: now},
		{Name: "noid.bin", Path: "/家庭空间/noid.bin", SizeBytes: 300 << 20, Modified: now}, // missing id
	}
	if out := Analyze(files, now); len(out) != 0 {
		t.Fatalf("archived/unidentified files should yield no suggestions, got %#v", out)
	}
}

func TestAnalyzeNothingActionable(t *testing.T) {
	now := time.Now()
	files := []FileInfo{{ID: "f1", Name: "a.txt", Path: "/a.txt", SizeBytes: 10, Modified: now}}
	if out := Analyze(files, now); len(out) != 0 {
		t.Fatalf("expected no suggestions, got %#v", out)
	}
}

func TestReplaceSuggestionsKeepsResolved(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	// Confirm a seeded suggestion so it becomes resolved (kept across refresh).
	seeded := svc.ListSuggestions(ctx)
	if len(seeded) == 0 {
		t.Fatal("expected seeded suggestions")
	}
	target := seeded[0]
	if _, err := svc.Preview(ctx, target.ID, PreviewRequest{}); err != nil {
		t.Fatalf("preview: %v", err)
	}
	pv := svc.previews[target.ID]
	if _, err := svc.Confirm(ctx, target.ID, ConfirmRequest{ConfirmationID: pv.ConfirmationID}); err != nil {
		t.Fatalf("confirm: %v", err)
	}

	out, err := svc.ReplaceSuggestions(ctx, []Suggestion{
		{ID: "dup-files", Title: "清理重复文件", Risk: RiskMedium},
	})
	if err != nil {
		t.Fatalf("replace: %v", err)
	}
	var sawConfirmed, sawNew bool
	for _, s := range out {
		if s.ID == target.ID && s.Status == SuggestionConfirmed {
			sawConfirmed = true
		}
		if s.ID == "dup-files" && s.Status == SuggestionPending {
			sawNew = true
		}
	}
	if !sawConfirmed {
		t.Error("resolved suggestion should be kept")
	}
	if !sawNew {
		t.Error("new analyzed suggestion should be added as pending")
	}
}
