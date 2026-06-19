package activity

import "testing"

func TestAppendListClear(t *testing.T) {
	s := NewService()
	s.Append([]Entry{{Type: "page", Action: "visit", Category: "docker"}, {Action: "run-task", Target: "transcode"}})
	all := s.List("", 0)
	if len(all) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(all))
	}
	if all[0].ID <= all[1].ID {
		t.Fatalf("expected newest-first ordering")
	}
	if pages := s.List("page", 0); len(pages) != 1 || pages[0].Category != "docker" {
		t.Fatalf("type filter failed: %+v", pages)
	}
	if limited := s.List("", 1); len(limited) != 1 {
		t.Fatalf("limit failed: %d", len(limited))
	}
	s.Clear()
	if len(s.List("", 0)) != 0 {
		t.Fatalf("clear failed")
	}
}

func TestRetentionPrune(t *testing.T) {
	s := NewService()
	s.SetRetention(3)
	for i := 0; i < 10; i++ {
		s.Append([]Entry{{Action: "x"}})
	}
	if got := len(s.List("", 0)); got != 3 {
		t.Fatalf("expected prune to 3, got %d", got)
	}
}
