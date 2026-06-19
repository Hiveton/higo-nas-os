package aianalysis

import (
	"testing"
	"time"
)

func fixedNow() func() time.Time {
	t := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	return func() time.Time {
		t = t.Add(time.Second)
		return t
	}
}

func mustLedger(t *testing.T) *Ledger {
	t.Helper()
	l, err := newLedger(DomainFile, "", fixedNow())
	if err != nil {
		t.Fatalf("newLedger: %v", err)
	}
	return l
}

func TestReconcileQueuesNewAndChangedAndPrunes(t *testing.T) {
	l := mustLedger(t)
	refs := []ItemRef{{Key: "file:a", Domain: DomainFile, Title: "a", Sig: "1"}}

	if pruned := l.reconcile(refs, LevelBasic); len(pruned) != 0 {
		t.Fatalf("expected no pruned, got %d", len(pruned))
	}
	if got := len(l.pending()); got != 1 {
		t.Fatalf("expected 1 pending after first reconcile, got %d", got)
	}

	// Unchanged sig+level: record left untouched (still pending here).
	l.reconcile(refs, LevelBasic)
	if got := len(l.pending()); got != 1 {
		t.Fatalf("unchanged reconcile should not duplicate, got %d pending", got)
	}

	// Mark done; an unchanged reconcile must keep it done.
	l.complete("file:a", AnalyzerResult{Summary: "ok"})
	l.reconcile(refs, LevelBasic)
	if got := len(l.pending()); got != 0 {
		t.Fatalf("done item should stay done, got %d pending", got)
	}

	// Changed signature re-queues.
	l.reconcile([]ItemRef{{Key: "file:a", Domain: DomainFile, Title: "a", Sig: "2"}}, LevelBasic)
	if got := len(l.pending()); got != 1 {
		t.Fatalf("changed sig should re-queue, got %d pending", got)
	}

	// Level change re-queues even when sig is unchanged.
	l.complete("file:a", AnalyzerResult{})
	l.reconcile([]ItemRef{{Key: "file:a", Domain: DomainFile, Title: "a", Sig: "2"}}, LevelDeep)
	if got := len(l.pending()); got != 1 {
		t.Fatalf("level change should re-queue, got %d pending", got)
	}

	// Missing item is pruned.
	pruned := l.reconcile(nil, LevelDeep)
	if len(pruned) != 1 || pruned[0].Key != "file:a" {
		t.Fatalf("expected file:a pruned, got %#v", pruned)
	}
	if _, ok := l.get("file:a"); ok {
		t.Fatalf("pruned record should be gone")
	}
}

func TestRecoverInterruptedResetsAnalyzing(t *testing.T) {
	l := mustLedger(t)
	l.reconcile([]ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}, LevelBasic)
	if _, ok := l.begin("file:a"); !ok {
		t.Fatalf("begin should succeed")
	}
	if rec, _ := l.get("file:a"); rec.State != StateAnalyzing {
		t.Fatalf("expected analyzing, got %s", rec.State)
	}

	l.recoverInterrupted()
	rec, _ := l.get("file:a")
	if rec.State != StatePending {
		t.Fatalf("recoverInterrupted should reset to pending, got %s", rec.State)
	}
	if rec.Progress != 0 {
		t.Fatalf("recovered record should reset progress, got %d", rec.Progress)
	}
}

func TestResetForReanalyze(t *testing.T) {
	l := mustLedger(t)
	l.reconcile([]ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}, LevelBasic)
	l.complete("file:a", AnalyzerResult{Summary: "done"})

	if !l.reset("file:a") {
		t.Fatalf("reset should find the record")
	}
	rec, _ := l.get("file:a")
	if rec.State != StatePending {
		t.Fatalf("reset should set pending, got %s", rec.State)
	}
	if l.reset("file:missing") {
		t.Fatalf("reset of unknown key should return false")
	}
}
