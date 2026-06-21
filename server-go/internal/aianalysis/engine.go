package aianalysis

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/files"
	"higoos/server-go/internal/index"
	"higoos/server-go/internal/llm"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/settings"
	"higoos/server-go/internal/tasks"
	"higoos/server-go/internal/video"
)

// analyzeTaskKind is the task kind the engine registers with the shared runtime.
const analyzeTaskKind = "aianalysis.analyze"

// faceTrainTaskKind drives a self-training run through the shared task runtime.
const faceTrainTaskKind = "aianalysis.face-train"

const defaultInterval = 5 * time.Minute

// Deps are the engine's collaborators. Domain services may be nil; a nil service
// simply means that domain is not analysed.
type Deps struct {
	StateDir string
	Logger   *slog.Logger
	Interval time.Duration
	Settings *settings.Store
	LLM      *llm.Store
	Index    *index.Service
	Media    *media.Service
	Files    *files.Service
	Video    *video.Service
	// FaceEmbedderURL / FaceTrainerURL wire the self-training framework to
	// cross-arch model sidecars. Empty leaves them reserved but inert.
	FaceEmbedderURL string
	FaceTrainerURL  string
	// Debounce coalesces near-real-time Notify signals into a single rescan.
	// Zero defaults to 2s.
	Debounce time.Duration
}

// analyzePayload is the task payload for one item analysis.
type analyzePayload struct {
	Domain DomainKind `json:"domain"`
	Key    string     `json:"key"`
}

// Engine is the global background AI analysis coordinator.
type Engine struct {
	logger   *slog.Logger
	interval time.Duration
	settings *settings.Store
	llm      *llm.Store
	index    *index.Service
	tasks    *tasks.Manager

	sources map[DomainKind]DomainSource
	ledgers map[DomainKind]*Ledger
	faces   *faceClusterer

	// Self-training framework seams. embedder/trainer are no-ops until a sidecar
	// is configured; training is the persistent labeled-sample store.
	embedder FaceEmbedder
	training *FaceTrainingStore
	trainer  FaceTrainer

	now func() time.Time

	// notifyCh receives near-real-time change signals from domain services; the
	// Start loop debounces them into a single (per-domain) rescan. debounce is the
	// coalescing window.
	notifyCh chan DomainKind
	debounce time.Duration

	mu        sync.Mutex
	paused    bool
	enqueued  map[string]bool
	startOnce sync.Once
}

// New builds the engine, loading each domain's persisted ledger.
func New(deps Deps) (*Engine, error) {
	logger := deps.Logger
	if logger == nil {
		logger = slog.Default()
	}
	interval := deps.Interval
	if interval <= 0 {
		interval = defaultInterval
	}
	debounce := deps.Debounce
	if debounce <= 0 {
		debounce = 2 * time.Second
	}
	now := func() time.Time { return time.Now().UTC() }

	e := &Engine{
		logger:   logger,
		interval: interval,
		settings: deps.Settings,
		llm:      deps.LLM,
		index:    deps.Index,
		sources:  map[DomainKind]DomainSource{},
		ledgers:  map[DomainKind]*Ledger{},
		now:      now,
		notifyCh: make(chan DomainKind, 64),
		debounce: debounce,
		enqueued: map[string]bool{},
	}

	register := func(domain DomainKind, src DomainSource) error {
		ledger, err := newLedger(domain, ledgerPath(deps.StateDir, domain), now)
		if err != nil {
			return err
		}
		e.ledgers[domain] = ledger
		e.sources[domain] = src
		return nil
	}

	if deps.Media != nil {
		if err := register(DomainMedia, mediaSource{svc: deps.Media}); err != nil {
			return nil, err
		}
	}
	if deps.Files != nil {
		if err := register(DomainFile, fileSource{svc: deps.Files}); err != nil {
			return nil, err
		}
	}
	if deps.Video != nil {
		if err := register(DomainVideo, videoSource{svc: deps.Video}); err != nil {
			return nil, err
		}
	}

	faces, err := newFaceClusterer(facePath(deps.StateDir), now)
	if err != nil {
		return nil, err
	}
	e.faces = faces

	training, err := newFaceTrainingStore(faceTrainingPath(deps.StateDir), now)
	if err != nil {
		return nil, err
	}
	e.training = training
	e.embedder = NewFaceEmbedder(deps.FaceEmbedderURL)
	e.trainer = NewFaceTrainer(deps.FaceTrainerURL)

	return e, nil
}

func ledgerPath(stateDir string, domain DomainKind) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, fmt.Sprintf("aianalysis-%s.json", domain))
}

func facePath(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, "aianalysis-faces.json")
}

func faceTrainingPath(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	return filepath.Join(stateDir, "aianalysis-face-training.json")
}

// AttachTaskRunner wires the shared task runtime and registers the analyze
// handler. Call once before the manager is started.
func (e *Engine) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	e.tasks = m
	m.Register(analyzeTaskKind, e.runAnalyze)
	m.Register(faceTrainTaskKind, e.runFaceTrain)
}

// runFaceTrain is the task handler that exports the confirmed labeled dataset,
// invokes the configured trainer sidecar, and registers the resulting model
// version. It is a clean no-op error when no trainer is configured.
func (e *Engine) runFaceTrain(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	if e.trainer == nil || !e.trainer.Available() {
		return nil, fmt.Errorf("aianalysis: face trainer not configured")
	}
	h.Progress(20, "导出标注数据集")
	samples := e.training.Export(true)
	if len(samples) == 0 {
		return nil, fmt.Errorf("aianalysis: no confirmed face samples to train on")
	}
	h.Progress(50, "训练模型中")
	version, err := e.trainer.Train(ctx, samples)
	if err != nil {
		return nil, err
	}
	e.training.RegisterModel(version)
	h.Progress(100, "训练完成")
	return json.Marshal(version)
}

// Start recovers interrupted items and launches the background scan loop. Safe
// to call once; subsequent calls are no-ops.
func (e *Engine) Start(ctx context.Context) {
	e.startOnce.Do(func() {
		for _, l := range e.ledgers {
			l.recoverInterrupted()
		}
		go func() {
			e.Rescan(ctx)
			ticker := time.NewTicker(e.interval)
			defer ticker.Stop()

			// Debounce near-real-time Notify signals: collect the domains touched
			// within a window and rescan only those once it elapses. An empty domain
			// signals an all-domain rescan.
			var debounce *time.Timer
			var debounceC <-chan time.Time
			pending := map[DomainKind]bool{}
			allPending := false
			for {
				select {
				case <-ctx.Done():
					return
				case <-ticker.C:
					e.Rescan(ctx)
				case d := <-e.notifyCh:
					if d == "" {
						allPending = true
					} else {
						pending[d] = true
					}
					if debounce == nil {
						debounce = time.NewTimer(e.debounce)
						debounceC = debounce.C
					} else {
						debounce.Reset(e.debounce)
					}
				case <-debounceC:
					debounce = nil
					debounceC = nil
					if allPending || len(pending) == 0 {
						e.rescanDomains(ctx, nil)
					} else {
						e.rescanDomains(ctx, pending)
					}
					pending = map[DomainKind]bool{}
					allPending = false
				}
			}
		}()
	})
}

// Notify schedules a debounced rescan of the given domain (or all domains when
// domain is ""). Safe to call from any goroutine and from a hot mutation path;
// the bounded channel drops excess signals because a single pending rescan
// already covers them. No-op before Start wires the consuming loop, but signals
// sent after New (channel buffered) are retained until Start drains them.
func (e *Engine) Notify(domain DomainKind) {
	if e.notifyCh == nil {
		return
	}
	select {
	case e.notifyCh <- domain:
	default:
	}
}

// Rescan enumerates every domain, reconciles the ledgers against the live item
// set, drops pruned items from the index, and dispatches pending work.
func (e *Engine) Rescan(ctx context.Context) { e.rescanDomains(ctx, nil) }

// rescanDomains is Rescan restricted to a domain set (nil = all domains). The
// per-domain form lets a near-real-time file upload re-enumerate only the file
// domain instead of paying for a full media-disk + video-library walk.
func (e *Engine) rescanDomains(ctx context.Context, only map[DomainKind]bool) {
	if e.isPaused() {
		return
	}
	level := e.currentLevel()
	if level == LevelOff {
		return
	}
	for _, domain := range domainOrder {
		if only != nil && !only[domain] {
			continue
		}
		src := e.sources[domain]
		ledger := e.ledgers[domain]
		if src == nil || ledger == nil {
			continue
		}
		if ctx.Err() != nil {
			return
		}
		refs, err := src.Enumerate(ctx)
		if err != nil {
			e.logger.Warn("aianalysis enumerate failed", slog.String("domain", string(domain)), slog.Any("error", err))
			continue
		}
		pruned := ledger.reconcile(refs, level)
		for _, rec := range pruned {
			if e.indexEnabled() {
				_ = e.index.DeleteDocument(ctx, rec.Key)
			}
		}
	}
	e.dispatchPending()
}

// dispatchPending enqueues every pending record not already in flight.
func (e *Engine) dispatchPending() {
	if e.tasks == nil || e.isPaused() {
		return
	}
	for _, domain := range domainOrder {
		ledger := e.ledgers[domain]
		if ledger == nil {
			continue
		}
		for _, rec := range ledger.pending() {
			if !e.markEnqueued(rec.Key) {
				continue
			}
			if _, err := e.tasks.Enqueue(analyzeTaskKind, analyzePayload{Domain: domain, Key: rec.Key}); err != nil {
				e.clearEnqueued(rec.Key)
				e.logger.Warn("aianalysis enqueue failed", slog.String("key", rec.Key), slog.Any("error", err))
			}
		}
	}
}

// runAnalyze is the task handler that analyses one item.
func (e *Engine) runAnalyze(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload analyzePayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	defer e.clearEnqueued(payload.Key)

	ledger := e.ledgers[payload.Domain]
	src := e.sources[payload.Domain]
	if ledger == nil || src == nil {
		return nil, fmt.Errorf("aianalysis: unknown domain %q", payload.Domain)
	}

	if e.isPaused() {
		// Leave the record pending; resume will re-dispatch it.
		return json.Marshal(map[string]string{"status": "paused"})
	}
	level := e.currentLevel()
	if level == LevelOff {
		ledger.skip(payload.Key)
		return json.Marshal(map[string]string{"status": "skipped"})
	}

	rec, ok := ledger.begin(payload.Key)
	if !ok {
		return json.Marshal(map[string]string{"status": "missing"})
	}
	h.Progress(10, "分析中: "+rec.Title)

	result, err := e.analyze(ctx, rec, level, h)
	if err != nil {
		if ctx.Err() != nil {
			ledger.requeue(payload.Key)
			return nil, ctx.Err()
		}
		ledger.fail(payload.Key, err.Error())
		return nil, err
	}
	if err := src.Apply(ctx, payload.Key, result); err != nil {
		ledger.fail(payload.Key, err.Error())
		return nil, err
	}
	ledger.complete(payload.Key, result)
	h.Progress(100, "完成")
	out, _ := json.Marshal(result)
	return out, nil
}

func (e *Engine) analyze(ctx context.Context, rec Record, level Level, h *tasks.Handle) (AnalyzerResult, error) {
	switch rec.Domain {
	case DomainMedia:
		return e.analyzeMedia(ctx, rec, level, h)
	case DomainFile:
		return e.analyzeFile(ctx, rec, level, h)
	case DomainVideo:
		return e.analyzeVideo(ctx, rec, level, h)
	default:
		return AnalyzerResult{}, fmt.Errorf("aianalysis: unknown domain %q", rec.Domain)
	}
}

// Status returns the aggregate engine status for the API.
func (e *Engine) Status() EngineStatus {
	st := EngineStatus{
		Level:           e.currentLevel(),
		Paused:          e.isPaused(),
		HasChat:         e.hasProvider(llm.PurposeChat),
		HasVision:       e.hasProvider(llm.PurposeVision),
		HasEmbedding:    e.hasProvider(llm.PurposeEmbedding),
		HasASR:          e.hasProvider(llm.PurposeASR),
		IndexEnabled:    e.indexEnabled(),
		FFmpegAvailable: ffmpegAvailable(),
		UpdatedAt:       e.now(),
	}
	total, processed := 0, 0
	for _, domain := range domainOrder {
		ledger := e.ledgers[domain]
		if ledger == nil {
			st.Domains = append(st.Domains, DomainStats{Domain: domain, Percent: 100})
			continue
		}
		ds := ledger.stats()
		st.Domains = append(st.Domains, ds)
		total += ds.Total
		processed += ds.Done + ds.Failed + ds.Skipped
	}
	if total > 0 {
		st.TotalPercent = processed * 100 / total
	} else {
		st.TotalPercent = 100
	}
	return st
}

// Records returns a page of analysis records. An empty domain merges all
// domains; q free-text filters title/sourcePath/error.
func (e *Engine) Records(domain DomainKind, state ItemState, q string, page, size int) ([]Record, int) {
	if page <= 0 {
		page = 1
	}
	if size <= 0 {
		size = 50
	}
	var all []Record
	if domain == "" {
		for _, d := range domainOrder {
			if ledger := e.ledgers[d]; ledger != nil {
				all = append(all, ledger.filtered(state, q)...)
			}
		}
		sort.Slice(all, func(i, j int) bool {
			if all[i].UpdatedAt.Equal(all[j].UpdatedAt) {
				return all[i].Key < all[j].Key
			}
			return all[i].UpdatedAt.After(all[j].UpdatedAt)
		})
	} else if ledger := e.ledgers[domain]; ledger != nil {
		all = ledger.filtered(state, q)
	}
	total := len(all)
	start := (page - 1) * size
	if start >= total {
		return []Record{}, total
	}
	end := start + size
	if end > total {
		end = total
	}
	return all[start:end], total
}

// RecordByKey returns one record's full state (including the complete
// AnalyzerResult), looked up by its domain-prefixed key.
func (e *Engine) RecordByKey(key string) (Record, bool) {
	if ledger := e.ledgers[domainFromKey(key)]; ledger != nil {
		return ledger.get(key)
	}
	return Record{}, false
}

// ReanalyzeKeys resets a set of records to pending and dispatches. Unknown keys
// are skipped; returns the count actually reset.
func (e *Engine) ReanalyzeKeys(keys []string) int {
	n := 0
	for _, key := range keys {
		if ledger := e.ledgers[domainFromKey(key)]; ledger != nil {
			if ledger.reset(key) {
				n++
			}
		}
	}
	if n > 0 {
		go e.dispatchPending()
	}
	return n
}

// Reanalyze forces re-analysis: a single item (key set), a whole domain (domain
// set, key empty), or everything (both empty). It then dispatches the work.
func (e *Engine) Reanalyze(domain DomainKind, key string) error {
	switch {
	case key != "":
		d := domain
		if d == "" {
			d = domainFromKey(key)
		}
		ledger := e.ledgers[d]
		if ledger == nil {
			return fmt.Errorf("aianalysis: unknown domain for key %q", key)
		}
		if !ledger.reset(key) {
			return fmt.Errorf("aianalysis: record not found: %s", key)
		}
	case domain != "":
		ledger := e.ledgers[domain]
		if ledger == nil {
			return fmt.Errorf("aianalysis: unknown domain %q", domain)
		}
		ledger.resetAll()
	default:
		for _, ledger := range e.ledgers {
			ledger.resetAll()
		}
	}
	go e.dispatchPending()
	return nil
}

// Pause stops new work from being dispatched. In-flight items finish.
func (e *Engine) Pause() {
	e.mu.Lock()
	e.paused = true
	e.mu.Unlock()
}

// Resume clears the pause flag and re-dispatches pending work.
func (e *Engine) Resume() {
	e.mu.Lock()
	e.paused = false
	e.mu.Unlock()
	go e.Rescan(context.Background())
}

// Subscribe returns a channel that emits an aggregate status snapshot on every
// analyze-task state change, plus an unsubscribe func. It returns (nil, noop)
// when no task runtime is attached.
func (e *Engine) Subscribe() (<-chan EngineStatus, func()) {
	if e.tasks == nil {
		return nil, func() {}
	}
	taskCh, cancel := e.tasks.Subscribe()
	out := make(chan EngineStatus, 8)
	go func() {
		defer close(out)
		for t := range taskCh {
			if t.Kind != analyzeTaskKind {
				continue
			}
			select {
			case out <- e.Status():
			default:
			}
		}
	}()
	return out, cancel
}

// FaceFrameworkStatus is the API payload describing the self-training framework:
// the configured embedder/trainer, current face clusters, dataset stats, and the
// trained-model registry.
type FaceFrameworkStatus struct {
	EmbedderName  string             `json:"embedderName"`
	EmbedderReady bool               `json:"embedderReady"`
	TrainerName   string             `json:"trainerName"`
	TrainerReady  bool               `json:"trainerReady"`
	Clusters      []faceClusterView  `json:"clusters"`
	Training      FaceTrainingStats  `json:"training"`
	Models        []FaceModelVersion `json:"models"`
}

// FaceFramework returns the current state of the face self-training framework.
func (e *Engine) FaceFramework() FaceFrameworkStatus {
	st := FaceFrameworkStatus{}
	if e.embedder != nil {
		st.EmbedderName = e.embedder.Name()
		st.EmbedderReady = e.embedder.Available()
	}
	if e.trainer != nil {
		st.TrainerName = e.trainer.Name()
		st.TrainerReady = e.trainer.Available()
	}
	if e.faces != nil {
		st.Clusters = e.faces.clustersView()
	}
	if e.training != nil {
		st.Training = e.training.Stats()
		st.Models = e.training.Models()
	}
	return st
}

// LabelFace names a face cluster: it confirms the cluster's training samples and
// relabels the cluster so future photos carry the name. Returns the number of
// samples confirmed.
func (e *Engine) LabelFace(clusterID, name string) (int, error) {
	clusterID = strings.TrimSpace(clusterID)
	name = strings.TrimSpace(name)
	if clusterID == "" || name == "" {
		return 0, fmt.Errorf("aianalysis: clusterId and name are required")
	}
	n := 0
	if e.training != nil {
		n = e.training.Confirm(clusterID, name)
	}
	if e.faces != nil {
		e.faces.rename(clusterID, name)
	}
	return n, nil
}

// Retrain enqueues a self-training run on the confirmed dataset. It returns the
// task id, or an error when no trainer sidecar is configured.
func (e *Engine) Retrain() (string, error) {
	if e.trainer == nil || !e.trainer.Available() {
		return "", fmt.Errorf("aianalysis: face trainer not configured (set HIGO_FACE_TRAINER_URL)")
	}
	if e.tasks == nil {
		return "", fmt.Errorf("aianalysis: task runtime unavailable")
	}
	task, err := e.tasks.Enqueue(faceTrainTaskKind, nil)
	if err != nil {
		return "", err
	}
	return task.ID, nil
}

func (e *Engine) currentLevel() Level {
	if e.settings == nil {
		return LevelBasic
	}
	return ParseLevel(e.settings.Get().Analysis.Level)
}

func (e *Engine) isPaused() bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	return e.paused
}

func (e *Engine) markEnqueued(key string) bool {
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.enqueued[key] {
		return false
	}
	e.enqueued[key] = true
	return true
}

func (e *Engine) clearEnqueued(key string) {
	e.mu.Lock()
	defer e.mu.Unlock()
	delete(e.enqueued, key)
}

func domainFromKey(key string) DomainKind {
	switch {
	case len(key) >= 6 && key[:6] == "media:":
		return DomainMedia
	case len(key) >= 5 && key[:5] == "file:":
		return DomainFile
	case len(key) >= 6 && key[:6] == "video:":
		return DomainVideo
	default:
		return ""
	}
}
