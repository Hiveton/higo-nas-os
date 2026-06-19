// Package tasks is the control plane's persistent, in-process task runtime.
//
// Background work in HiGoOS used to be scattered across ad-hoc inline
// goroutines (and, worse, several domains created a task record that was
// persisted as "queued" but never advanced by anyone). This package converges
// that into a single managed model:
//
//   - Tasks are persisted via the JSON state layer, so they survive restarts.
//   - A fixed worker pool claims queued tasks and runs the handler registered
//     for the task kind, recording progress, result and errors.
//   - On Start, tasks left mid-flight by a previous process (status "running")
//     are recovered back to "queued" so they get re-driven.
//
// It is hosted in-process by higo-api. Cross-process consumption (a separate
// higo-worker driving the same queue) requires a shared transactional store and
// is deferred to the database phase — a JSON file cannot be safely shared by two
// writers.
package tasks

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"sort"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Status is the lifecycle state of a task.
type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCanceled  Status = "canceled"
)

// ErrNotFound is returned when a task id is unknown.
var ErrNotFound = errors.New("task not found")

// Task is a persisted unit of background work.
type Task struct {
	ID         string          `json:"id"`
	Kind       string          `json:"kind"`
	Status     Status          `json:"status"`
	Progress   int             `json:"progress"`
	Message    string          `json:"message,omitempty"`
	Payload    json.RawMessage `json:"payload,omitempty"`
	Result     json.RawMessage `json:"result,omitempty"`
	Error      string          `json:"error,omitempty"`
	Attempts   int             `json:"attempts"`
	CreatedAt  time.Time       `json:"createdAt"`
	StartedAt  *time.Time      `json:"startedAt,omitempty"`
	FinishedAt *time.Time      `json:"finishedAt,omitempty"`
	UpdatedAt  time.Time       `json:"updatedAt"`
}

// Handle is given to a Handler to read its payload and report progress.
type Handle struct {
	id      string
	kind    string
	payload json.RawMessage
	manager *Manager
}

// ID returns the running task's id.
func (h *Handle) ID() string { return h.id }

// Kind returns the running task's kind.
func (h *Handle) Kind() string { return h.kind }

// Payload returns the raw payload the task was enqueued with.
func (h *Handle) Payload() json.RawMessage { return h.payload }

// Unmarshal decodes the payload into v.
func (h *Handle) Unmarshal(v any) error {
	if len(h.payload) == 0 {
		return nil
	}
	return json.Unmarshal(h.payload, v)
}

// Progress records intermediate progress (0-100) and an optional message.
func (h *Handle) Progress(pct int, msg string) {
	h.manager.setProgress(h.id, pct, msg)
}

// Handler executes one task. The returned RawMessage is stored as the task
// result; a non-nil error marks the task failed. The ctx is canceled when the
// task is canceled (cooperative cancellation) — long-running handlers should
// honor it (ctx.Err()/select on ctx.Done()).
type Handler func(ctx context.Context, h *Handle) (json.RawMessage, error)

// Canceler cancels an externally-driven (adopted) task by id. Registered per
// kind so the central Cancel can route back to the owning domain (e.g. aria2
// remove for downloads).
type Canceler func(id string) error

// Manager owns the task store and worker pool.
type Manager struct {
	mu        sync.Mutex
	tasks     map[string]Task
	inflight  map[string]bool
	seq       int
	handlers  map[string]Handler
	cancels   map[string]context.CancelFunc // running handler-driven tasks
	cancelers map[string]Canceler           // per-kind, for adopted tasks
	subs      map[int]chan Task             // SSE subscribers
	subSeq    int
	statePath string
	now       func() time.Time
	logger    *slog.Logger
	workers   int
	dispatch  time.Duration

	queue   chan string
	started bool
	stop    chan struct{}
	wg      sync.WaitGroup
}

type snapshot struct {
	Seq   int             `json:"seq"`
	Tasks map[string]Task `json:"tasks"`
}

// Option configures a Manager.
type Option func(*Manager)

// WithWorkers sets the worker pool size (default 4).
func WithWorkers(n int) Option {
	return func(m *Manager) {
		if n > 0 {
			m.workers = n
		}
	}
}

// WithLogger sets the structured logger.
func WithLogger(l *slog.Logger) Option {
	return func(m *Manager) {
		if l != nil {
			m.logger = l
		}
	}
}

// WithClock overrides the time source (used in tests).
func WithClock(now func() time.Time) Option {
	return func(m *Manager) {
		if now != nil {
			m.now = now
		}
	}
}

// WithDispatchInterval overrides how often the dispatcher re-scans for queued
// work that was not delivered immediately (default 2s).
func WithDispatchInterval(d time.Duration) Option {
	return func(m *Manager) {
		if d > 0 {
			m.dispatch = d
		}
	}
}

// NewManager builds an unstarted manager. statePath is the JSON file the task
// ledger is persisted to; pass "" to run purely in memory (tests).
func NewManager(statePath string, opts ...Option) (*Manager, error) {
	m := &Manager{
		tasks:     map[string]Task{},
		inflight:  map[string]bool{},
		handlers:  map[string]Handler{},
		cancels:   map[string]context.CancelFunc{},
		cancelers: map[string]Canceler{},
		subs:      map[int]chan Task{},
		statePath: statePath,
		now:       func() time.Time { return time.Now().UTC() },
		logger:    slog.Default(),
		workers:   4,
		dispatch:  2 * time.Second,
		queue:     make(chan string, 256),
		stop:      make(chan struct{}),
	}
	for _, opt := range opts {
		opt(m)
	}
	if statePath != "" {
		var persisted snapshot
		if err := state.LoadJSON(statePath, &persisted); err != nil {
			return nil, err
		}
		if persisted.Tasks != nil {
			m.tasks = persisted.Tasks
		}
		m.seq = persisted.Seq
	}
	return m, nil
}

// Register binds a handler to a task kind. It must be called before Start.
func (m *Manager) Register(kind string, h Handler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.handlers[kind] = h
}

// RegisterCanceler binds a per-kind canceler used to abort externally-driven
// (adopted) tasks of that kind when Cancel is called.
func (m *Manager) RegisterCanceler(kind string, c Canceler) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.cancelers[kind] = c
}

// Subscribe returns a channel that receives a Task snapshot on every state
// change, plus an unsubscribe func. Sends are non-blocking (a slow consumer
// drops events) so the runtime is never stalled by a subscriber.
func (m *Manager) Subscribe() (<-chan Task, func()) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.subSeq++
	id := m.subSeq
	ch := make(chan Task, 64)
	m.subs[id] = ch
	return ch, func() {
		m.mu.Lock()
		defer m.mu.Unlock()
		if c, ok := m.subs[id]; ok {
			delete(m.subs, id)
			close(c)
		}
	}
}

// emitLocked fans a task snapshot out to subscribers. Caller must hold m.mu.
func (m *Manager) emitLocked(task Task) {
	for _, ch := range m.subs {
		select {
		case ch <- task:
		default:
		}
	}
}

// Start recovers interrupted tasks and launches the worker pool. The pool runs
// until ctx is canceled or Stop is called.
func (m *Manager) Start(ctx context.Context) {
	m.mu.Lock()
	if m.started {
		m.mu.Unlock()
		return
	}
	m.started = true
	// Recover tasks that were running when the previous process exited.
	for id, task := range m.tasks {
		if task.Status == StatusRunning {
			task.Status = StatusQueued
			task.Message = "进程重启，任务已重新排队"
			task.UpdatedAt = m.now()
			m.tasks[id] = task
		}
	}
	_ = m.saveLocked()
	m.mu.Unlock()

	for i := 0; i < m.workers; i++ {
		m.wg.Add(1)
		go m.worker(ctx)
	}
	m.wg.Add(1)
	go m.dispatcher(ctx)
}

// Stop signals the pool to drain and waits for workers to exit.
func (m *Manager) Stop() {
	m.mu.Lock()
	if !m.started {
		m.mu.Unlock()
		return
	}
	select {
	case <-m.stop:
	default:
		close(m.stop)
	}
	m.mu.Unlock()
	m.wg.Wait()
}

// Enqueue persists a new queued task and wakes a worker. payload is marshaled to
// JSON; pass nil for no payload.
func (m *Manager) Enqueue(kind string, payload any) (Task, error) {
	var raw json.RawMessage
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return Task{}, fmt.Errorf("encode task payload: %w", err)
		}
		raw = encoded
	}
	m.mu.Lock()
	m.seq++
	now := m.now()
	task := Task{
		ID:        fmt.Sprintf("task-%s-%04d", kind, m.seq),
		Kind:      kind,
		Status:    StatusQueued,
		Payload:   raw,
		CreatedAt: now,
		UpdatedAt: now,
	}
	m.tasks[task.ID] = task
	if err := m.saveLocked(); err != nil {
		m.mu.Unlock()
		return Task{}, err
	}
	m.emitLocked(task)
	m.mu.Unlock()

	m.offer(task.ID)
	return task, nil
}

// Get returns a task by id.
func (m *Manager) Get(id string) (Task, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok {
		return Task{}, ErrNotFound
	}
	return task, nil
}

// List returns all tasks, newest first. When kind is non-empty it filters by kind.
func (m *Manager) List(kind string) []Task {
	m.mu.Lock()
	defer m.mu.Unlock()
	out := make([]Task, 0, len(m.tasks))
	for _, task := range m.tasks {
		if kind != "" && task.Kind != kind {
			continue
		}
		out = append(out, task)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].CreatedAt.Equal(out[j].CreatedAt) {
			return out[i].ID > out[j].ID
		}
		return out[i].CreatedAt.After(out[j].CreatedAt)
	})
	return out
}

// Cancel cancels a task. A queued task is marked canceled immediately. A running
// task is canceled cooperatively: its handler context is canceled (handler-driven
// tasks) and/or its registered Canceler is invoked (adopted/external tasks); the
// task transitions to canceled asynchronously when it unwinds.
func (m *Manager) Cancel(id string) (Task, error) {
	m.mu.Lock()
	task, ok := m.tasks[id]
	if !ok {
		m.mu.Unlock()
		return Task{}, ErrNotFound
	}
	now := m.now()
	switch task.Status {
	case StatusQueued:
		task.Status = StatusCanceled
		task.Message = "任务已取消"
		task.UpdatedAt = now
		task.FinishedAt = &now
		m.tasks[id] = task
		_ = m.saveLocked()
		m.emitLocked(task)
		m.mu.Unlock()
		return task, nil
	case StatusRunning:
		cancel := m.cancels[id]
		canceler := m.cancelers[task.Kind]
		task.Message = "取消中"
		task.UpdatedAt = now
		m.tasks[id] = task
		_ = m.saveLocked()
		m.emitLocked(task)
		m.mu.Unlock()
		// Route cancellation outside the lock; the task settles asynchronously.
		if cancel != nil {
			cancel()
		}
		if canceler != nil {
			_ = canceler(id)
		}
		return task, nil
	default:
		m.mu.Unlock()
		return task, fmt.Errorf("task %s is %s and cannot be canceled", id, task.Status)
	}
}

// offer tries to deliver an id to a worker without blocking the caller; the
// dispatcher will pick it up later if the channel is momentarily full.
func (m *Manager) offer(id string) {
	select {
	case m.queue <- id:
	default:
	}
}

func (m *Manager) dispatcher(ctx context.Context) {
	defer m.wg.Done()
	ticker := time.NewTicker(m.dispatch)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stop:
			return
		case <-ticker.C:
			for _, id := range m.queuedIDs() {
				m.offer(id)
			}
		}
	}
}

func (m *Manager) queuedIDs() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	var ids []string
	for id, task := range m.tasks {
		if task.Status == StatusQueued && !m.inflight[id] {
			ids = append(ids, id)
		}
	}
	return ids
}

func (m *Manager) worker(ctx context.Context) {
	defer m.wg.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case <-m.stop:
			return
		case id := <-m.queue:
			m.run(ctx, id)
		}
	}
}

// claim transitions a task Queued->Running atomically. It returns the handler
// and a Handle, or ok=false if the task was already taken / not runnable.
func (m *Manager) claim(id string) (Handler, *Handle, bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok || task.Status != StatusQueued || m.inflight[id] {
		return nil, nil, false
	}
	handler, ok := m.handlers[task.Kind]
	if !ok {
		now := m.now()
		task.Status = StatusFailed
		task.Error = fmt.Sprintf("no handler registered for kind %q", task.Kind)
		task.UpdatedAt = now
		task.FinishedAt = &now
		m.tasks[id] = task
		_ = m.saveLocked()
		m.emitLocked(task)
		return nil, nil, false
	}
	now := m.now()
	task.Status = StatusRunning
	task.Attempts++
	task.StartedAt = &now
	task.UpdatedAt = now
	task.Progress = 0
	task.Error = ""
	m.tasks[id] = task
	m.inflight[id] = true
	_ = m.saveLocked()
	m.emitLocked(task)
	return handler, &Handle{id: id, kind: task.Kind, payload: task.Payload, manager: m}, true
}

func (m *Manager) run(ctx context.Context, id string) {
	handler, handle, ok := m.claim(id)
	if !ok {
		return
	}
	// Per-task cancellation context so Cancel can abort a running handler.
	taskCtx, cancel := context.WithCancel(ctx)
	m.mu.Lock()
	m.cancels[id] = cancel
	m.mu.Unlock()
	defer func() {
		m.mu.Lock()
		delete(m.inflight, id)
		delete(m.cancels, id)
		m.mu.Unlock()
		cancel()
	}()

	var (
		result json.RawMessage
		err    error
	)
	func() {
		defer func() {
			if r := recover(); r != nil {
				err = fmt.Errorf("task panicked: %v", r)
			}
		}()
		result, err = handler(taskCtx, handle)
	}()

	m.mu.Lock()
	task := m.tasks[id]
	now := m.now()
	task.UpdatedAt = now
	task.FinishedAt = &now
	switch {
	case err != nil && (errors.Is(err, context.Canceled) || taskCtx.Err() == context.Canceled):
		task.Status = StatusCanceled
		task.Message = "任务已取消"
	case err != nil:
		task.Status = StatusFailed
		task.Error = err.Error()
		m.logger.Warn("task failed", slog.String("id", id), slog.String("kind", task.Kind), slog.Any("error", err))
	default:
		task.Status = StatusSucceeded
		task.Progress = 100
		task.Result = result
		task.Message = "已完成" // replace the last in-flight progress label
	}
	m.tasks[id] = task
	_ = m.saveLocked()
	m.emitLocked(task)
	m.mu.Unlock()
}

func (m *Manager) setProgress(id string, pct int, msg string) {
	if pct < 0 {
		pct = 0
	}
	if pct > 100 {
		pct = 100
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok || task.Status != StatusRunning {
		return
	}
	task.Progress = pct
	if msg != "" {
		task.Message = msg
	}
	task.UpdatedAt = m.now()
	m.tasks[id] = task
	_ = m.saveLocked()
	m.emitLocked(task)
}

// Adopt creates a task that is already "running" and is driven externally
// (no registered handler) — for domains whose work is owned elsewhere (e.g.
// aria2 downloads). Drive it with Update and finish it with Settle; register a
// Canceler for its kind so Cancel can route back to the domain.
func (m *Manager) Adopt(kind string, payload any) (Task, error) {
	var raw json.RawMessage
	if payload != nil {
		encoded, err := json.Marshal(payload)
		if err != nil {
			return Task{}, fmt.Errorf("encode task payload: %w", err)
		}
		raw = encoded
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	m.seq++
	now := m.now()
	task := Task{
		ID:        fmt.Sprintf("task-%s-%04d", kind, m.seq),
		Kind:      kind,
		Status:    StatusRunning,
		Payload:   raw,
		CreatedAt: now,
		StartedAt: &now,
		UpdatedAt: now,
	}
	m.tasks[task.ID] = task
	_ = m.saveLocked()
	m.emitLocked(task)
	return task, nil
}

// Update advances an adopted (running) task's progress/message. Alias of the
// internal progress path so domains have a clear external API.
func (m *Manager) Update(id string, pct int, msg string) {
	m.setProgress(id, pct, msg)
}

// Settle marks an adopted task terminal (succeeded/failed/canceled). For other
// statuses it is a no-op. result/errMsg are optional.
func (m *Manager) Settle(id string, status Status, result json.RawMessage, errMsg string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	task, ok := m.tasks[id]
	if !ok {
		return
	}
	if task.Status != StatusRunning && task.Status != StatusQueued {
		return
	}
	now := m.now()
	task.Status = status
	task.UpdatedAt = now
	task.FinishedAt = &now
	switch status {
	case StatusSucceeded:
		task.Progress = 100
		task.Result = result
		task.Error = ""
	case StatusFailed:
		task.Error = errMsg
	case StatusCanceled:
		task.Message = "任务已取消"
	}
	m.tasks[id] = task
	_ = m.saveLocked()
	m.emitLocked(task)
}

// Stats summarises a task ledger by status.
type Stats struct {
	Total     int `json:"total"`
	Queued    int `json:"queued"`
	Running   int `json:"running"`
	Succeeded int `json:"succeeded"`
	Failed    int `json:"failed"`
	Canceled  int `json:"canceled"`
}

// ReadLedger loads the persisted task ledger read-only. It is safe for a second
// process (e.g. higo-worker) to call for reporting without coordinating writes.
func ReadLedger(statePath string) ([]Task, error) {
	if statePath == "" {
		return nil, nil
	}
	var persisted snapshot
	if err := state.LoadJSON(statePath, &persisted); err != nil {
		return nil, err
	}
	out := make([]Task, 0, len(persisted.Tasks))
	for _, task := range persisted.Tasks {
		out = append(out, task)
	}
	return out, nil
}

// Summarize counts tasks by status.
func Summarize(list []Task) Stats {
	s := Stats{Total: len(list)}
	for _, task := range list {
		switch task.Status {
		case StatusQueued:
			s.Queued++
		case StatusRunning:
			s.Running++
		case StatusSucceeded:
			s.Succeeded++
		case StatusFailed:
			s.Failed++
		case StatusCanceled:
			s.Canceled++
		}
	}
	return s
}

func (m *Manager) saveLocked() error {
	if m.statePath == "" {
		return nil
	}
	clone := make(map[string]Task, len(m.tasks))
	for id, task := range m.tasks {
		clone[id] = task
	}
	return state.SaveJSON(m.statePath, snapshot{Seq: m.seq, Tasks: clone})
}
