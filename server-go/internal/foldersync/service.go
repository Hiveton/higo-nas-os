package foldersync

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/filesync"
	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

const (
	syncRunTaskKind    = "sync.run"
	syncVerifyTaskKind = "sync.verify"
)

// Service owns sync pairs, conflicts and the audit log. Real synchronization
// runs through the central task runtime; config CRUD is direct + audited.
type Service struct {
	mu        sync.RWMutex
	now       func() time.Time
	seq       int
	pairs     []SyncPair
	conflicts []Conflict
	audit     []AuditEntry
	statePath string
	runner    *tasks.Manager
}

type snapshot struct {
	Seq       int          `json:"seq"`
	Pairs     []SyncPair   `json:"pairs"`
	Conflicts []Conflict   `json:"conflicts"`
	Audit     []AuditEntry `json:"audit"`
}

type runPayload struct {
	PairID string `json:"pairId"`
}

func NewService() *Service {
	s := &Service{now: time.Now}
	s.seed()
	return s
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	s := NewService()
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "sync.json")
	var persisted snapshot
	if err := state.LoadJSON(s.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Pairs) > 0 || len(persisted.Audit) > 0 || len(persisted.Conflicts) > 0 {
		s.seq = persisted.Seq
		s.pairs = clonePairs(persisted.Pairs)
		s.conflicts = append([]Conflict(nil), persisted.Conflicts...)
		s.audit = append([]AuditEntry(nil), persisted.Audit...)
	}
	return s, nil
}

func (s *Service) seed() {
	created := time.Date(2026, 6, 1, 9, 0, 0, 0, time.UTC)
	s.pairs = []SyncPair{
		{ID: "sync-001", Name: "工作目录 → 备份盘", Source: "/家庭空间/工作", Target: "/备份/工作", Direction: DirectionMirror, ConflictPolicy: ConflictNewer, Enabled: true, IntervalHours: 6, State: "空闲", CreatedAt: created},
		{ID: "sync-002", Name: "手机相册 ↔ 团队相册", Source: "/家庭空间/相册", Target: "/团队空间/相册", Direction: DirectionTwoWay, ConflictPolicy: ConflictNewer, Enabled: false, State: "空闲", CreatedAt: created},
	}
	s.seq = 2
}

// AttachTaskRunner registers the sync task handlers. Must run before the pool starts.
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.runner = m
	m.Register(syncRunTaskKind, s.runSyncPair)
	m.Register(syncVerifyTaskKind, s.verifySyncPair)
}

// --- reads -----------------------------------------------------------------

func (s *Service) List(ctx context.Context) ([]SyncPair, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePairs(s.pairs), nil
}

func (s *Service) Get(ctx context.Context, id string) (SyncPair, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.findLocked(id); ok {
		return clonePair(p), nil
	}
	return SyncPair{}, fmt.Errorf("sync pair not found: %s", id)
}

func (s *Service) Conflicts(ctx context.Context) ([]Conflict, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Conflict(nil), s.conflicts...), nil
}

func (s *Service) Audit(ctx context.Context) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AuditEntry(nil), s.audit...), nil
}

// --- config CRUD (direct + audited) ----------------------------------------

func (s *Service) Create(ctx context.Context, req CreatePairRequest) (SyncPair, error) {
	if strings.TrimSpace(req.Name) == "" {
		return SyncPair{}, fmt.Errorf("sync pair name is required")
	}
	if err := validatePath(req.Source); err != nil {
		return SyncPair{}, fmt.Errorf("source: %w", err)
	}
	if err := validatePath(req.Target); err != nil {
		return SyncPair{}, fmt.Errorf("target: %w", err)
	}
	dir := req.Direction
	if dir == "" {
		dir = DirectionMirror
	}
	if !validDirection(dir) {
		return SyncPair{}, fmt.Errorf("invalid direction: %s", dir)
	}
	policy := req.ConflictPolicy
	if policy == "" {
		policy = ConflictNewer
	}
	if !validConflictPolicy(policy) {
		return SyncPair{}, fmt.Errorf("invalid conflict policy: %s", policy)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	pair := SyncPair{
		ID:             fmt.Sprintf("sync-%03d", s.seq),
		Name:           strings.TrimSpace(req.Name),
		Source:         filepath.Clean(strings.TrimSpace(req.Source)),
		Target:         filepath.Clean(strings.TrimSpace(req.Target)),
		Direction:      dir,
		ConflictPolicy: policy,
		Includes:       append([]string(nil), req.Includes...),
		BandwidthLimit: strings.TrimSpace(req.BandwidthLimit),
		Enabled:        true,
		IntervalHours:  req.IntervalHours,
		State:          "空闲",
		CreatedAt:      s.now().UTC(),
		CreatedBy:      req.Actor,
	}
	s.pairs = append(s.pairs, pair)
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("创建同步任务：%s", pair.Name), Actor: req.Actor, PairID: pair.ID, Result: "ok"})
	return clonePair(pair), s.saveLocked()
}

func (s *Service) Update(ctx context.Context, id string, req UpdatePairRequest) (SyncPair, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(id)
	if idx < 0 {
		return SyncPair{}, fmt.Errorf("sync pair not found: %s", id)
	}
	p := &s.pairs[idx]
	if req.Name != nil {
		p.Name = strings.TrimSpace(*req.Name)
	}
	if req.Direction != nil && validDirection(*req.Direction) {
		p.Direction = *req.Direction
	}
	if req.ConflictPolicy != nil && validConflictPolicy(*req.ConflictPolicy) {
		p.ConflictPolicy = *req.ConflictPolicy
	}
	if req.Includes != nil {
		p.Includes = append([]string(nil), (*req.Includes)...)
	}
	if req.BandwidthLimit != nil {
		p.BandwidthLimit = strings.TrimSpace(*req.BandwidthLimit)
	}
	if req.Enabled != nil {
		p.Enabled = *req.Enabled
	}
	if req.IntervalHours != nil {
		p.IntervalHours = *req.IntervalHours
	}
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("更新同步任务：%s", p.Name), Actor: req.Actor, PairID: p.ID, Result: "ok"})
	updated := clonePair(*p)
	return updated, s.saveLocked()
}

func (s *Service) Delete(ctx context.Context, id, actor string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(id)
	if idx < 0 {
		return fmt.Errorf("sync pair not found: %s", id)
	}
	name := s.pairs[idx].Name
	s.pairs = append(s.pairs[:idx], s.pairs[idx+1:]...)
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("删除同步任务：%s", name), Actor: actor, PairID: id, Result: "ok"})
	return s.saveLocked()
}

// --- runs (task-driven) ----------------------------------------------------

func (s *Service) Run(ctx context.Context, id, actor string) (SyncPair, error) {
	s.mu.Lock()
	idx := s.indexLocked(id)
	if idx < 0 {
		s.mu.Unlock()
		return SyncPair{}, fmt.Errorf("sync pair not found: %s", id)
	}
	s.pairs[idx].State = "同步中"
	s.pairs[idx].Progress = 5
	updated := clonePair(s.pairs[idx])
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("启动同步：%s", updated.Name), Actor: actor, PairID: id, Result: "ok"})
	_ = s.saveLocked()
	runner := s.runner
	s.mu.Unlock()

	if runner == nil {
		// No task runtime (unit tests): run inline so behaviour is deterministic.
		_, _ = s.doSyncRun(ctx, id)
		return s.Get(ctx, id)
	}
	if _, err := runner.Enqueue(syncRunTaskKind, runPayload{PairID: id}); err != nil {
		return SyncPair{}, err
	}
	return updated, nil
}

func (s *Service) Verify(ctx context.Context, id, actor string) (SyncPair, error) {
	s.mu.RLock()
	_, ok := s.findLocked(id)
	s.mu.RUnlock()
	if !ok {
		return SyncPair{}, fmt.Errorf("sync pair not found: %s", id)
	}
	if s.runner == nil {
		_, _ = s.doVerify(ctx, id)
		return s.Get(ctx, id)
	}
	if _, err := s.runner.Enqueue(syncVerifyTaskKind, runPayload{PairID: id}); err != nil {
		return SyncPair{}, err
	}
	return s.Get(ctx, id)
}

// runSyncPair is the task-runtime handler; the real work lives in doSyncRun so
// the nil-runner inline path (tests) can reuse it without a *tasks.Handle.
func (s *Service) runSyncPair(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var p runPayload
	if err := h.Unmarshal(&p); err != nil {
		return nil, err
	}
	h.Progress(20, "同步中")
	return s.doSyncRun(ctx, p.PairID)
}

func (s *Service) doSyncRun(ctx context.Context, pairID string) (json.RawMessage, error) {
	pair, ok := s.snapshotPair(pairID)
	if !ok {
		return nil, fmt.Errorf("sync pair not found: %s", pairID)
	}
	p := runPayload{PairID: pairID}
	filter := buildFilter(pair.Includes)

	var stats RunStats
	var conflicts []conflictRec
	var err error
	if pair.Direction == DirectionTwoWay {
		var res filesync.SyncResult
		res, conflicts, err = twoWaySync(ctx, pair.Source, pair.Target, pair.ConflictPolicy, filter)
		stats = RunStats{Copied: res.Copied, Skipped: res.Skipped, Bytes: res.Bytes, Conflicts: len(conflicts)}
	} else {
		var res filesync.SyncResult
		res, err = filesync.SyncTree(ctx, pair.Source, pair.Target, filter)
		stats = RunStats{Copied: res.Copied, Skipped: res.Skipped, Bytes: res.Bytes}
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	idx := s.indexLocked(p.PairID)
	if idx < 0 {
		return nil, fmt.Errorf("sync pair vanished: %s", p.PairID)
	}
	now := s.now().UTC()
	s.pairs[idx].LastRun = &now
	s.pairs[idx].Progress = 100
	if err != nil {
		s.pairs[idx].State = "失败"
		s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("同步失败：%s（%s）", s.pairs[idx].Name, err.Error()), PairID: p.PairID, Result: "blocked"})
		_ = s.saveLocked()
		return nil, err
	}
	statsCopy := stats
	s.pairs[idx].LastStats = &statsCopy
	if len(conflicts) > 0 {
		s.pairs[idx].State = "有冲突"
		for _, c := range conflicts {
			s.seq++
			s.conflicts = append(s.conflicts, Conflict{ID: fmt.Sprintf("conflict-%03d", s.seq), PairID: p.PairID, RelPath: c.rel, Detail: c.detail, DetectedAt: now})
		}
		s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("同步完成（%d 个冲突待处理）：%s", len(conflicts), s.pairs[idx].Name), PairID: p.PairID, Result: "conflict"})
	} else {
		s.pairs[idx].State = "已完成"
		s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("同步完成：%s（复制 %d，跳过 %d）", s.pairs[idx].Name, stats.Copied, stats.Skipped), PairID: p.PairID, Result: "ok"})
	}
	_ = s.saveLocked()
	return json.Marshal(stats)
}

func (s *Service) verifySyncPair(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var p runPayload
	if err := h.Unmarshal(&p); err != nil {
		return nil, err
	}
	h.Progress(30, "校验中")
	return s.doVerify(ctx, p.PairID)
}

func (s *Service) doVerify(ctx context.Context, pairID string) (json.RawMessage, error) {
	pair, ok := s.snapshotPair(pairID)
	if !ok {
		return nil, fmt.Errorf("sync pair not found: %s", pairID)
	}
	res, err := filesync.VerifyTree(ctx, pair.Source, pair.Target)
	s.mu.Lock()
	defer s.mu.Unlock()
	if err != nil {
		s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("校验失败：%s（%s）", pair.Name, err.Error()), PairID: pairID, Result: "blocked"})
		_ = s.saveLocked()
		return nil, err
	}
	result := "ok"
	if res.Mismatch > 0 || res.Missing > 0 {
		result = "conflict"
	}
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("校验完成：%s（检查 %d，缺失 %d，不一致 %d）", pair.Name, res.Checked, res.Missing, res.Mismatch), PairID: pairID, Result: result})
	_ = s.saveLocked()
	return json.Marshal(res)
}

// ResolveConflict copies the chosen side's file across and marks the conflict
// resolved. side is "source" or "target".
func (s *Service) ResolveConflict(ctx context.Context, conflictID, side, actor string) (Conflict, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	ci := -1
	for i := range s.conflicts {
		if s.conflicts[i].ID == conflictID {
			ci = i
			break
		}
	}
	if ci < 0 {
		return Conflict{}, fmt.Errorf("conflict not found: %s", conflictID)
	}
	c := &s.conflicts[ci]
	pair, ok := s.findLocked(c.PairID)
	if !ok {
		return Conflict{}, fmt.Errorf("sync pair not found: %s", c.PairID)
	}
	srcRoot, dstRoot := pair.Source, pair.Target
	if side == "target" {
		srcRoot, dstRoot = pair.Target, pair.Source
	}
	from := filepath.Join(srcRoot, filepath.FromSlash(c.RelPath))
	to := filepath.Join(dstRoot, filepath.FromSlash(c.RelPath))
	fi, err := os.Stat(from)
	if err != nil {
		return Conflict{}, fmt.Errorf("chosen side unavailable: %w", err)
	}
	if err := filesync.CopyFile(from, to, fi); err != nil {
		return Conflict{}, err
	}
	c.Resolved = true
	c.Resolution = side
	s.appendAuditLocked(AuditEntry{Event: fmt.Sprintf("解决冲突 %s（采用%s）", c.RelPath, sideLabel(side)), Actor: actor, PairID: c.PairID, Result: "ok"})
	resolved := *c
	return resolved, s.saveLocked()
}

// --- locked helpers --------------------------------------------------------

func (s *Service) findLocked(id string) (SyncPair, bool) {
	idx := s.indexLocked(id)
	if idx < 0 {
		return SyncPair{}, false
	}
	return s.pairs[idx], true
}

func (s *Service) indexLocked(id string) int {
	for i := range s.pairs {
		if s.pairs[i].ID == id {
			return i
		}
	}
	return -1
}

func (s *Service) snapshotPair(id string) (SyncPair, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if p, ok := s.findLocked(id); ok {
		return clonePair(p), true
	}
	return SyncPair{}, false
}

func (s *Service) appendAuditLocked(e AuditEntry) {
	s.seq++
	e.ID = fmt.Sprintf("sync-audit-%03d", s.seq)
	if e.Time.IsZero() {
		e.Time = s.now().UTC()
	}
	s.audit = append([]AuditEntry{e}, s.audit...)
	if len(s.audit) > 200 {
		s.audit = s.audit[:200]
	}
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{Seq: s.seq, Pairs: s.pairs, Conflicts: s.conflicts, Audit: s.audit})
}

func sideLabel(side string) string {
	if side == "target" {
		return "目标端"
	}
	return "源端"
}

// validatePath requires a non-empty path, and — when HIGO_NAS_ROOT is set (real
// NAS) — that it resolve under that root, mirroring the protocols domain.
func validatePath(p string) error {
	p = strings.TrimSpace(p)
	if p == "" {
		return fmt.Errorf("path is required")
	}
	root := strings.TrimSpace(os.Getenv("HIGO_NAS_ROOT"))
	if root == "" {
		return nil
	}
	abs := filepath.Clean(p)
	root = filepath.Clean(root)
	if abs != root && !strings.HasPrefix(abs, root+string(os.PathSeparator)) {
		return fmt.Errorf("path must be under the NAS root")
	}
	return nil
}

// buildFilter turns include globs into a filesync.Filter (nil = include all).
func buildFilter(includes []string) filesync.Filter {
	if len(includes) == 0 {
		return nil
	}
	pats := append([]string(nil), includes...)
	return func(rel string) bool {
		for _, pat := range pats {
			if ok, _ := path.Match(pat, rel); ok {
				return true
			}
			// Directory-prefix include, e.g. "docs/*" should match "docs/a/b".
			base := strings.TrimSuffix(strings.TrimSuffix(pat, "*"), "/")
			if base != "" && (rel == base || strings.HasPrefix(rel, base+"/")) {
				return true
			}
		}
		return false
	}
}
