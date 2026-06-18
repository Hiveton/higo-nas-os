package steward

import (
	"context"
	"testing"
	"time"
)

func TestAnalyzeDuplicatesAndArchive(t *testing.T) {
	now := time.Date(2026, 6, 19, 0, 0, 0, 0, time.UTC)
	files := []FileInfo{
		{Name: "报告.docx", Path: "/a/报告.docx", SizeBytes: 100, Modified: now},
		{Name: "报告.docx", Path: "/b/报告.docx", SizeBytes: 100, Modified: now}, // duplicate
		{Name: "video.mp4", Path: "/m/video.mp4", SizeBytes: 200 << 20, Modified: now}, // large
		{Name: "old.txt", Path: "/o/old.txt", SizeBytes: 10, Modified: now.Add(-400 * 24 * time.Hour)}, // stale
	}
	out := Analyze(files, now)
	if len(out) != 2 {
		t.Fatalf("expected duplicate + archive suggestions, got %d: %#v", len(out), out)
	}
	ids := map[string]Suggestion{}
	for _, s := range out {
		ids[s.ID] = s
	}
	if _, ok := ids["dup-files"]; !ok {
		t.Error("missing dup-files suggestion")
	}
	if a, ok := ids["archive-candidates"]; !ok || a.Risk != RiskLow {
		t.Errorf("missing/incorrect archive suggestion: %#v", a)
	}
}

func TestAnalyzeNothingActionable(t *testing.T) {
	now := time.Now()
	files := []FileInfo{{Name: "a.txt", Path: "/a.txt", SizeBytes: 10, Modified: now}}
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
