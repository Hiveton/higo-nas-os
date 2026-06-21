package aianalysis

import (
	"context"
	"log/slog"
	"sync"
	"testing"
	"time"

	"higoos/server-go/internal/settings"
	"higoos/server-go/internal/tasks"
)

type fakeSource struct {
	refs    []ItemRef
	mu      sync.Mutex
	applied map[string]AnalyzerResult
	enum    int
}

func (f *fakeSource) Enumerate(ctx context.Context) ([]ItemRef, error) {
	f.mu.Lock()
	f.enum++
	f.mu.Unlock()
	return f.refs, nil
}
func (f *fakeSource) Apply(ctx context.Context, key string, res AnalyzerResult) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.applied == nil {
		f.applied = map[string]AnalyzerResult{}
	}
	f.applied[key] = res
	return nil
}
func (f *fakeSource) enumCount() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.enum
}

func testStore(t *testing.T, level string) *settings.Store {
	t.Helper()
	store := settings.NewStore()
	s := settings.DefaultSettings()
	s.Analysis.Level = level
	if _, err := store.Update(s); err != nil {
		t.Fatalf("settings update: %v", err)
	}
	return store
}

func newTestEngine(t *testing.T, store *settings.Store, src DomainSource) (*Engine, *tasks.Manager) {
	t.Helper()
	ledger, err := newLedger(DomainFile, "", func() time.Time { return time.Now().UTC() })
	if err != nil {
		t.Fatalf("newLedger: %v", err)
	}
	mgr, err := tasks.NewManager("")
	if err != nil {
		t.Fatalf("tasks.NewManager: %v", err)
	}
	e := &Engine{
		logger:   slog.Default(),
		interval: time.Minute,
		settings: store,
		sources:  map[DomainKind]DomainSource{DomainFile: src},
		ledgers:  map[DomainKind]*Ledger{DomainFile: ledger},
		now:      func() time.Time { return time.Now().UTC() },
		enqueued: map[string]bool{},
	}
	e.AttachTaskRunner(mgr)
	return e, mgr
}

func TestRescanEnqueuesPendingOncePerItem(t *testing.T) {
	src := &fakeSource{refs: []ItemRef{
		{Key: "file:a", Domain: DomainFile, Sig: "1"},
		{Key: "file:b", Domain: DomainFile, Sig: "1"},
	}}
	e, mgr := newTestEngine(t, testStore(t, "basic"), src)

	e.Rescan(context.Background())
	if got := len(mgr.List(analyzeTaskKind)); got != 2 {
		t.Fatalf("expected 2 enqueued tasks, got %d", got)
	}

	// A second rescan with no changes must not double-enqueue (dedup guard).
	e.Rescan(context.Background())
	if got := len(mgr.List(analyzeTaskKind)); got != 2 {
		t.Fatalf("rescan should not duplicate, got %d tasks", got)
	}
}

func TestNotifyDebouncesRescans(t *testing.T) {
	src := &fakeSource{refs: []ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}}
	e, _ := newTestEngine(t, testStore(t, "basic"), src)
	e.notifyCh = make(chan DomainKind, 64)
	e.debounce = 40 * time.Millisecond

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	e.Start(ctx)

	// Wait for the initial Rescan that Start performs.
	waitFor(t, func() bool { return src.enumCount() >= 1 })
	base := src.enumCount()

	// A burst of notifications inside the debounce window must coalesce into a
	// single rescan, not one rescan per signal.
	for i := 0; i < 6; i++ {
		e.Notify(DomainFile)
	}
	time.Sleep(150 * time.Millisecond)
	if got := src.enumCount() - base; got != 1 {
		t.Fatalf("expected exactly 1 coalesced rescan after a burst, got %d", got)
	}
}

func waitFor(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("condition not met before deadline")
}

func TestRescanLevelOffDispatchesNothing(t *testing.T) {
	src := &fakeSource{refs: []ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}}
	e, mgr := newTestEngine(t, testStore(t, "off"), src)

	e.Rescan(context.Background())
	if got := len(mgr.List(analyzeTaskKind)); got != 0 {
		t.Fatalf("level off should enqueue nothing, got %d", got)
	}
}

func TestReanalyzeResolvesDomainFromKey(t *testing.T) {
	src := &fakeSource{refs: []ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}}
	e, _ := newTestEngine(t, testStore(t, "basic"), src)
	e.Rescan(context.Background())
	e.ledgers[DomainFile].complete("file:a", AnalyzerResult{Summary: "done"})

	if err := e.Reanalyze("", "file:a"); err != nil {
		t.Fatalf("reanalyze: %v", err)
	}
	rec, _ := e.ledgers[DomainFile].get("file:a")
	if rec.State != StatePending {
		t.Fatalf("reanalyze should reset to pending, got %s", rec.State)
	}
	if err := e.Reanalyze("", "file:missing"); err == nil {
		t.Fatalf("reanalyze of unknown key should error")
	}
}

func TestPauseBlocksDispatch(t *testing.T) {
	src := &fakeSource{refs: []ItemRef{{Key: "file:a", Domain: DomainFile, Sig: "1"}}}
	e, mgr := newTestEngine(t, testStore(t, "basic"), src)
	e.Pause()
	e.Rescan(context.Background())
	if got := len(mgr.List(analyzeTaskKind)); got != 0 {
		t.Fatalf("paused engine should not dispatch, got %d", got)
	}
}
