package aianalysis

import (
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// ledgerSnapshot is the on-disk shape of a domain ledger.
type ledgerSnapshot struct {
	Records map[string]Record `json:"records"`
}

// Ledger is the persistent per-item analysis state for one domain. It is the
// item-level checkpoint that makes background analysis resumable. The single
// writer is the higo-api process, satisfying the JSON state single-writer rule.
type Ledger struct {
	mu      sync.Mutex
	domain  DomainKind
	path    string
	records map[string]Record
	now     func() time.Time
}

func newLedger(domain DomainKind, path string, now func() time.Time) (*Ledger, error) {
	l := &Ledger{domain: domain, path: path, records: map[string]Record{}, now: now}
	if path != "" {
		var snap ledgerSnapshot
		if err := state.LoadJSON(path, &snap); err != nil {
			return nil, err
		}
		if snap.Records != nil {
			l.records = snap.Records
		}
	}
	return l, nil
}

// reconcile diffs the live item set against the ledger: new items, items whose
// content signature changed, and items whose analysis level changed are (re)set
// to pending. Records for items no longer present are pruned and returned so the
// caller can drop their index documents.
func (l *Ledger) reconcile(refs []ItemRef, level Level) []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	live := make(map[string]bool, len(refs))
	for _, ref := range refs {
		live[ref.Key] = true
		rec, ok := l.records[ref.Key]
		if ok && rec.ContentSig == ref.Sig && rec.Level == level {
			continue // up to date — leave state untouched
		}
		rec.Key = ref.Key
		rec.Domain = l.domain
		rec.Title = ref.Title
		rec.SourcePath = ref.SourcePath
		rec.Kind = ref.Kind
		rec.ContentSig = ref.Sig
		rec.Level = level
		rec.State = StatePending
		rec.Progress = 0
		rec.Error = ""
		rec.UpdatedAt = now
		l.records[ref.Key] = rec
	}
	var pruned []Record
	for key, rec := range l.records {
		if !live[key] {
			pruned = append(pruned, rec)
			delete(l.records, key)
		}
	}
	_ = l.saveLocked()
	return pruned
}

// recoverInterrupted resets records left mid-analysis (StateAnalyzing) back to
// pending so they are re-driven after a restart / power loss.
func (l *Ledger) recoverInterrupted() {
	l.mu.Lock()
	defer l.mu.Unlock()
	changed := false
	now := l.now()
	for key, rec := range l.records {
		if rec.State == StateAnalyzing {
			rec.State = StatePending
			rec.Progress = 0
			rec.UpdatedAt = now
			l.records[key] = rec
			changed = true
		}
	}
	if changed {
		_ = l.saveLocked()
	}
}

// pending returns a snapshot of all records awaiting analysis.
func (l *Ledger) pending() []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	var out []Record
	for _, rec := range l.records {
		if rec.State == StatePending {
			out = append(out, rec)
		}
	}
	return out
}

// get returns a record snapshot.
// recordMatches reports whether a record matches a lowercased free-text query
// against its title, source path and error message.
func recordMatches(rec Record, q string) bool {
	return strings.Contains(strings.ToLower(rec.Title), q) ||
		strings.Contains(strings.ToLower(rec.SourcePath), q) ||
		strings.Contains(strings.ToLower(rec.Error), q)
}

func (l *Ledger) get(key string) (Record, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	return rec, ok
}

// begin transitions a pending record to analyzing and returns it. ok is false if
// the record vanished.
func (l *Ledger) begin(key string) (Record, bool) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return Record{}, false
	}
	rec.State = StateAnalyzing
	rec.Attempts++
	rec.Progress = 0
	rec.Error = ""
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
	return rec, true
}

func (l *Ledger) progress(key string, pct int) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok || rec.State != StateAnalyzing {
		return
	}
	rec.Progress = pct
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
}

func (l *Ledger) complete(key string, res AnalyzerResult) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return
	}
	now := l.now()
	rec.State = StateDone
	rec.Progress = 100
	rec.Error = ""
	rec.Result = res
	rec.AnalyzedAt = &now
	rec.UpdatedAt = now
	l.records[key] = rec
	_ = l.saveLocked()
}

func (l *Ledger) fail(key, msg string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return
	}
	rec.State = StateFailed
	rec.Error = msg
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
}

func (l *Ledger) skip(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return
	}
	rec.State = StateSkipped
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
}

// requeue puts a record back to pending (e.g. after a cooperative cancel).
func (l *Ledger) requeue(key string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return
	}
	rec.State = StatePending
	rec.Progress = 0
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
}

// reset forces a single record back to pending for re-analysis. Returns false if
// the key is unknown.
func (l *Ledger) reset(key string) bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	rec, ok := l.records[key]
	if !ok {
		return false
	}
	rec.State = StatePending
	rec.Progress = 0
	rec.Error = ""
	rec.UpdatedAt = l.now()
	l.records[key] = rec
	_ = l.saveLocked()
	return true
}

// resetAll forces every record back to pending for re-analysis.
func (l *Ledger) resetAll() {
	l.mu.Lock()
	defer l.mu.Unlock()
	now := l.now()
	for key, rec := range l.records {
		rec.State = StatePending
		rec.Progress = 0
		rec.Error = ""
		rec.UpdatedAt = now
		l.records[key] = rec
	}
	_ = l.saveLocked()
}

func (l *Ledger) stats() DomainStats {
	l.mu.Lock()
	defer l.mu.Unlock()
	st := DomainStats{Domain: l.domain, Total: len(l.records)}
	for _, rec := range l.records {
		switch rec.State {
		case StatePending:
			st.Pending++
		case StateAnalyzing:
			st.Analyzing++
		case StateDone:
			st.Done++
		case StateFailed:
			st.Failed++
		case StateSkipped:
			st.Skipped++
		}
	}
	if st.Total > 0 {
		st.Percent = (st.Done + st.Failed + st.Skipped) * 100 / st.Total
	} else {
		st.Percent = 100
	}
	return st
}

// filtered returns records (optionally filtered by state) sorted newest first.
func (l *Ledger) filtered(state ItemState, q string) []Record {
	l.mu.Lock()
	defer l.mu.Unlock()
	q = strings.ToLower(strings.TrimSpace(q))
	out := make([]Record, 0, len(l.records))
	for _, rec := range l.records {
		if state != "" && rec.State != state {
			continue
		}
		if q != "" && !recordMatches(rec, q) {
			continue
		}
		out = append(out, rec)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].UpdatedAt.Equal(out[j].UpdatedAt) {
			return out[i].Key < out[j].Key
		}
		return out[i].UpdatedAt.After(out[j].UpdatedAt)
	})
	return out
}

func (l *Ledger) saveLocked() error {
	if l.path == "" {
		return nil
	}
	clone := make(map[string]Record, len(l.records))
	for k, v := range l.records {
		clone[k] = v
	}
	return state.SaveJSON(l.path, ledgerSnapshot{Records: clone})
}
