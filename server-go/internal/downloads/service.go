package downloads

import (
	"context"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

const downloadTaskKind = "downloads.fetch"

var (
	ErrTaskNotFound     = errors.New("download task not found")
	ErrProfileNotFound  = errors.New("speed profile not found")
	ErrInvalidTaskInput = errors.New("invalid download task input")
)

type Service struct {
	mu          sync.RWMutex
	tasks       []DownloadTask
	profiles    []SpeedProfile
	nextID      int
	statePath   string
	downloadDir string
	active      map[int]context.CancelFunc
	client      *http.Client

	// maxConcurrent caps simultaneously-running downloads; excess stay queued
	// and are pumped as slots free up. 0 means unlimited.
	maxConcurrent int

	runner     *tasks.Manager
	centralIDs map[int]string // download task id -> central task id
}

type snapshot struct {
	Tasks         []DownloadTask `json:"tasks"`
	Profiles      []SpeedProfile `json:"profiles"`
	NextID        int            `json:"nextId"`
	MaxConcurrent int            `json:"maxConcurrent"`
}

const defaultMaxConcurrent = 3

type feed struct {
	Channel struct {
		Items []feedItem `xml:"item"`
	} `xml:"channel"`
	Entries []feedEntry `xml:"entry"`
}

type feedItem struct {
	Title      string          `xml:"title"`
	Link       string          `xml:"link"`
	Enclosures []feedEnclosure `xml:"enclosure"`
}

type feedEntry struct {
	Title string     `xml:"title"`
	Links []atomLink `xml:"link"`
}

type feedEnclosure struct {
	URL string `xml:"url,attr"`
}

type atomLink struct {
	Href string `xml:"href,attr"`
	Rel  string `xml:"rel,attr"`
	Type string `xml:"type,attr"`
}

type httpProbe struct {
	StatusCode   int
	Size         int64
	Filename     string
	AcceptRanges bool
	Body         string
}

func NewService() *Service {
	return newService("", "")
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	return NewServiceWithStateDirAndDownloadDir(stateDir, "")
}

func NewServiceWithStateDirAndDownloadDir(stateDir string, downloadDir string) (*Service, error) {
	service := newService(stateDir, downloadDir)
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "downloads.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Tasks) > 0 {
		service.tasks = filterPersistedTasks(persisted.Tasks)
	}
	if len(persisted.Profiles) > 0 {
		service.profiles = append([]SpeedProfile(nil), persisted.Profiles...)
	}
	if persisted.NextID > 0 {
		service.nextID = max(persisted.NextID, nextTaskID(service.tasks))
	} else {
		service.nextID = nextTaskID(service.tasks)
	}
	if persisted.MaxConcurrent > 0 {
		service.maxConcurrent = persisted.MaxConcurrent
	}
	service.backfillProfileLimits()
	_ = service.saveLocked()
	return service, nil
}

// backfillProfileLimits derives the numeric bytes/sec fields from the display
// strings for any profile persisted before the numeric fields existed.
func (s *Service) backfillProfileLimits() {
	for i := range s.profiles {
		if s.profiles[i].DownloadLimitBytesPerSecond == 0 {
			s.profiles[i].DownloadLimitBytesPerSecond = parseSpeedLimit(s.profiles[i].DownloadLimit)
		}
		if s.profiles[i].UploadLimitBytesPerSecond == 0 {
			s.profiles[i].UploadLimitBytesPerSecond = parseSpeedLimit(s.profiles[i].UploadLimit)
		}
	}
}

func newService(stateDir string, downloadDir string) *Service {
	downloadDir = strings.TrimSpace(downloadDir)
	if downloadDir == "" {
		if stateDir != "" {
			downloadDir = filepath.Join(stateDir, "downloads")
		} else {
			downloadDir = filepath.Join(os.TempDir(), "higoos-downloads")
		}
	}
	svc := &Service{
		profiles:      seedSpeedProfiles(),
		nextID:        1,
		downloadDir:   downloadDir,
		active:        make(map[int]context.CancelFunc),
		client:        &http.Client{},
		centralIDs:    make(map[int]string),
		maxConcurrent: defaultMaxConcurrent,
	}
	svc.backfillProfileLimits()
	return svc
}

// AttachTaskRunner wires the shared task runtime so download tasks are mirrored
// into the central task ledger (kind "downloads.fetch") and can be canceled
// centrally. Pause/resume stay download-local and are not routed via Cancel.
// A nil runner preserves the legacy standalone behavior.
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.mu.Lock()
	s.runner = m
	if s.centralIDs == nil {
		s.centralIDs = make(map[int]string)
	}
	s.mu.Unlock()
	m.RegisterCanceler(downloadTaskKind, s.cancelDownload)
}

// cancelDownload maps a central task id back to the download and removes it,
// which cancels any in-flight transfer and drops the task record.
func (s *Service) cancelDownload(centralID string) error {
	s.mu.RLock()
	downloadID := -1
	for id, cid := range s.centralIDs {
		if cid == centralID {
			downloadID = id
			break
		}
	}
	s.mu.RUnlock()
	if downloadID < 0 {
		return nil
	}
	_, err := s.DeleteTask(context.Background(), downloadID, DeleteTaskOptions{})
	return err
}

// mirrorToCentral pushes a download task's current progress/status onto its
// adopted central task. Terminal download states settle the central record.
func (s *Service) mirrorToCentral(task DownloadTask) {
	s.mu.RLock()
	runner := s.runner
	centralID := s.centralIDs[task.ID]
	s.mu.RUnlock()
	if runner == nil || centralID == "" {
		return
	}
	switch task.Status {
	case StatusCompleted:
		result, _ := json.Marshal(map[string]any{"name": task.Name, "filePath": task.FilePath})
		runner.Settle(centralID, tasks.StatusSucceeded, result, "")
	case StatusFailed:
		runner.Settle(centralID, tasks.StatusFailed, nil, task.Error)
	default:
		runner.Update(centralID, task.Progress, task.Handling)
	}
}

func (s *Service) ListTasks(_ context.Context) []DownloadTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneTasks(s.tasks)
}

func (s *Service) CreateTask(ctx context.Context, request CreateTaskRequest) (DownloadTask, error) {
	link := strings.TrimSpace(request.Link)
	if link == "" {
		return DownloadTask{}, fmt.Errorf("%w: link is required", ErrInvalidTaskInput)
	}

	source, err := normalizeSource(request.Source, link)
	if err != nil {
		return DownloadTask{}, err
	}

	category := normalizeCategory(request.Category, source, link, request.Name)
	rule := archiveRuleFor(category, source, request.Name, link)
	task := DownloadTask{
		Name:        taskName(request.Name, source, link),
		Source:      source,
		Link:        link,
		Category:    category,
		Size:        "解析中",
		Progress:    0,
		Speed:       "排队中",
		Status:      StatusQueued,
		Handling:    handlingFor(category, source, rule, request.Name, link),
		ArchiveRule: rule,

		SpeedLimitBytesPerSecond: request.SpeedLimitBytesPerSecond,
	}

	s.mu.Lock()
	task.ID = s.nextID
	s.nextID++
	s.tasks = append([]DownloadTask{task}, s.tasks...)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return DownloadTask{}, err
	}
	runner := s.runner
	task = cloneTask(task)
	s.mu.Unlock()

	// Mirror into the central task runtime: a download is an externally-driven
	// (adopted) task advanced by the poll/refresh loop and settled on terminus.
	if runner != nil {
		if adopted, err := runner.Adopt(downloadTaskKind, map[string]any{"name": task.Name, "link": task.Link}); err == nil {
			s.mu.Lock()
			if s.centralIDs == nil {
				s.centralIDs = make(map[int]string)
			}
			s.centralIDs[task.ID] = adopted.ID
			s.mu.Unlock()
		}
	}

	s.startTask(ctx, task.ID)
	return task, nil
}

func (s *Service) PauseTask(_ context.Context, id int) (DownloadTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return DownloadTask{}, ErrTaskNotFound
	}
	if cancel := s.active[id]; cancel != nil {
		cancel()
		delete(s.active, id)
	}
	if s.tasks[idx].Status == StatusCompleted {
		return cloneTask(s.tasks[idx]), nil
	}
	s.tasks[idx].Status = StatusPaused
	s.tasks[idx].Speed = "0 KB/s"
	s.tasks[idx].Handling = "已暂停，可继续或清理任务"
	if err := s.saveLocked(); err != nil {
		return DownloadTask{}, err
	}
	return cloneTask(s.tasks[idx]), nil
}

func (s *Service) ResumeTask(ctx context.Context, id int) (DownloadTask, error) {
	s.mu.Lock()
	idx := s.findTaskIndex(id)
	if idx < 0 {
		s.mu.Unlock()
		return DownloadTask{}, ErrTaskNotFound
	}
	if s.tasks[idx].Status == StatusCompleted {
		task := cloneTask(s.tasks[idx])
		s.mu.Unlock()
		return task, nil
	}
	s.tasks[idx].Status = StatusQueued
	s.tasks[idx].Speed = "排队中"
	s.tasks[idx].Error = ""
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return DownloadTask{}, err
	}
	task := cloneTask(s.tasks[idx])
	s.mu.Unlock()

	s.startTask(ctx, id)
	return task, nil
}

func (s *Service) ArchiveTask(_ context.Context, id int) (TaskActionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return TaskActionResult{}, ErrTaskNotFound
	}

	task := &s.tasks[idx]
	task.Progress = 100
	task.Speed = "0 KB/s"
	task.Status = StatusCompleted
	task.Archived = true
	task.Handling = "已归档到文件管家 " + task.ArchiveRule.TargetPath

	// Archiving completes the task; settle the mirrored central record.
	if runner := s.runner; runner != nil {
		if centralID := s.centralIDs[id]; centralID != "" {
			result, _ := json.Marshal(map[string]any{"name": task.Name, "filePath": task.FilePath})
			defer runner.Settle(centralID, tasks.StatusSucceeded, result, "")
		}
	}

	return TaskActionResult{
		Task:     cloneTask(*task),
		Message:  "已归档：" + task.Name,
		FilePath: taskFilePath(*task),
	}, s.saveLocked()
}

func (s *Service) DeleteTask(_ context.Context, id int, options DeleteTaskOptions) (TaskActionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return TaskActionResult{}, ErrTaskNotFound
	}
	// Settle and drop the mirrored central task: deleting a download is a
	// cancellation from the runtime's point of view (a no-op once terminal).
	if runner := s.runner; runner != nil {
		if centralID := s.centralIDs[id]; centralID != "" {
			defer runner.Settle(centralID, tasks.StatusCanceled, nil, "")
		}
	}
	delete(s.centralIDs, id)
	if cancel := s.active[id]; cancel != nil {
		cancel()
		delete(s.active, id)
	}
	task := cloneTask(s.tasks[idx])
	s.tasks = append(s.tasks[:idx], s.tasks[idx+1:]...)

	message := "已删除任务记录：" + task.Name
	if task.Archived {
		message = "已清理 1 条已归档记录，原文件保留在文件管家。"
	}
	if options.DeleteFile {
		deleted, err := s.deleteTaskFilesLocked(task)
		if err != nil {
			return TaskActionResult{}, err
		}
		if deleted > 0 {
			message = fmt.Sprintf("已删除任务和文件：%s", task.Name)
		} else {
			message = fmt.Sprintf("已删除任务记录，未找到可删除文件：%s", task.Name)
		}
	}
	return TaskActionResult{Task: task, Message: message, FilePath: taskFilePath(task)}, s.saveLocked()
}

func (s *Service) SpeedProfiles(_ context.Context) []SpeedProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles := make([]SpeedProfile, len(s.profiles))
	copy(profiles, s.profiles)
	return profiles
}

func (s *Service) UpdateActiveSpeedProfile(_ context.Context, name string) (SpeedProfile, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	name = strings.TrimSpace(name)
	activeIdx := -1
	for idx := range s.profiles {
		s.profiles[idx].Active = s.profiles[idx].Name == name
		if s.profiles[idx].Active {
			activeIdx = idx
		}
	}
	if activeIdx < 0 {
		for idx := range s.profiles {
			s.profiles[idx].Active = idx == 0
		}
		return SpeedProfile{}, ErrProfileNotFound
	}

	profile := s.profiles[activeIdx]
	for idx := range s.tasks {
		if s.tasks[idx].Status == StatusRunning {
			s.tasks[idx].Speed = profile.DownloadLimit
		}
	}
	return profile, s.saveLocked()
}

func (s *Service) startTask(ctx context.Context, id int) {
	s.mu.Lock()
	s.startTaskLocked(ctx, id)
	s.mu.Unlock()
}

// startTaskLocked starts a task if a concurrency slot is free; otherwise it
// leaves the task queued for a later pump. Caller must hold s.mu.
func (s *Service) startTaskLocked(ctx context.Context, id int) {
	if _, exists := s.active[id]; exists {
		return
	}
	idx := s.findTaskIndex(id)
	if idx < 0 || s.tasks[idx].Status == StatusCompleted {
		return
	}
	if s.maxConcurrent > 0 && len(s.active) >= s.maxConcurrent {
		// No free slot: keep it queued; pumpQueueLocked will pick it up.
		s.tasks[idx].Status = StatusQueued
		s.tasks[idx].Speed = "排队中"
		_ = s.saveLocked()
		return
	}
	task := cloneTask(s.tasks[idx])
	runCtx, cancel := context.WithCancel(context.WithoutCancel(ctx))
	s.active[id] = cancel
	s.tasks[idx].Status = StatusRunning
	s.tasks[idx].Speed = s.activeProfileLocked().DownloadLimit
	_ = s.saveLocked()

	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.active, id)
			_ = s.saveLocked()
			// A slot just freed — promote the next queued task.
			s.pumpQueueLocked(context.Background())
			s.mu.Unlock()
		}()
		switch task.Source {
		case SourceHTTP:
			s.runHTTPDownload(runCtx, task)
		case SourceRSS:
			s.runRSS(runCtx, task)
		case SourceBT, SourceMagnet:
			s.runAria2(runCtx, task)
		}
	}()
}

// pumpQueueLocked starts queued tasks until the concurrency cap is reached.
// Caller must hold s.mu.
func (s *Service) pumpQueueLocked(ctx context.Context) {
	for i := range s.tasks {
		if s.maxConcurrent > 0 && len(s.active) >= s.maxConcurrent {
			return
		}
		if s.tasks[i].Status == StatusQueued {
			if _, running := s.active[s.tasks[i].ID]; !running {
				s.startTaskLocked(ctx, s.tasks[i].ID)
			}
		}
	}
}

// effectiveLimit returns the bytes/sec cap for a task: its per-task override if
// set, else the active profile's download limit (0 = unlimited).
func (s *Service) effectiveLimit(taskID int) int64 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if idx := s.findTaskIndex(taskID); idx >= 0 && s.tasks[idx].SpeedLimitBytesPerSecond > 0 {
		return s.tasks[idx].SpeedLimitBytesPerSecond
	}
	return s.activeProfileLocked().DownloadLimitBytesPerSecond
}

// QueueConfig returns the current concurrency settings.
func (s *Service) QueueConfig(_ context.Context) QueueConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return QueueConfig{MaxConcurrentDownloads: s.maxConcurrent}
}

// UpdateQueueConfig sets the max concurrent downloads and immediately pumps the
// queue so a raised limit starts more tasks (and a lowered one just tightens
// future starts; running tasks are never killed mid-flight).
func (s *Service) UpdateQueueConfig(ctx context.Context, cfg QueueConfig) (QueueConfig, error) {
	if cfg.MaxConcurrentDownloads < 0 {
		return QueueConfig{}, fmt.Errorf("%w: maxConcurrentDownloads must be >= 0", ErrInvalidTaskInput)
	}
	s.mu.Lock()
	s.maxConcurrent = cfg.MaxConcurrentDownloads
	_ = s.saveLocked()
	s.pumpQueueLocked(ctx)
	result := QueueConfig{MaxConcurrentDownloads: s.maxConcurrent}
	s.mu.Unlock()
	return result, nil
}

func (s *Service) runHTTPDownload(ctx context.Context, task DownloadTask) {
	probe, err := s.probeHTTPDownload(ctx, task.Link)
	if err != nil {
		s.failTask(task.ID, err)
		return
	}
	if probe.StatusCode < 200 || probe.StatusCode >= 300 {
		s.failTask(task.ID, fmt.Errorf("http status %d%s", probe.StatusCode, errorDetail(probe.Body)))
		return
	}

	dir := filepath.Join(s.downloadDir, safePathPart(task.Category))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.failTask(task.ID, err)
		return
	}
	name := probe.Filename
	if name == "" {
		name = safeFilename(task.Name)
	}
	finalPath := task.FilePath
	if finalPath == "" {
		finalPath = uniquePath(filepath.Join(dir, name))
	}
	partPath := finalPath + ".part"
	startAt := existingFileSize(partPath)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.Link, nil)
	if err != nil {
		s.failTask(task.ID, err)
		return
	}
	prepareHTTPRequest(req)
	if startAt > 0 {
		req.Header.Set("Range", fmt.Sprintf("bytes=%d-", startAt))
	} else if probe.AcceptRanges {
		req.Header.Set("Range", "bytes=0-")
	}
	resp, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		s.failTask(task.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusRequestedRangeNotSatisfiable && startAt > 0 && probe.Size > 0 && startAt >= probe.Size {
		if err := os.Rename(partPath, finalPath); err != nil && !errors.Is(err, os.ErrNotExist) {
			s.failTask(task.ID, err)
			return
		}
		s.completeHTTPTask(task.ID, finalPath, startAt)
		return
	}
	if startAt > 0 && resp.StatusCode == http.StatusOK {
		startAt = 0
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		s.failTask(task.ID, fmt.Errorf("http status %d%s", resp.StatusCode, errorDetail(string(body))))
		return
	}
	totalSize := probe.Size
	if total := contentRangeTotal(resp.Header.Get("Content-Range")); total > 0 {
		totalSize = total
	} else if totalSize <= 0 && resp.ContentLength > 0 {
		totalSize = resp.ContentLength + startAt
	}
	openFlags := os.O_CREATE | os.O_WRONLY
	if startAt == 0 {
		openFlags |= os.O_TRUNC
	}
	file, err := os.OpenFile(partPath, openFlags, 0o644)
	if err != nil {
		s.failTask(task.ID, err)
		return
	}
	defer file.Close()
	if _, err := file.Seek(startAt, io.SeekStart); err != nil {
		s.failTask(task.ID, err)
		return
	}

	s.updateTask(task.ID, func(t *DownloadTask) {
		t.FilePath = finalPath
		t.Handling = "正在下载到 " + finalPath
		t.Error = ""
		if totalSize > 0 {
			t.Size = formatBytes(totalSize)
		}
	})

	buf := make([]byte, 128*1024)
	written := startAt
	lastBytes := startAt
	lastTick := time.Now()
	limiter := newRateLimiter(s.effectiveLimit(task.ID))
	for {
		n, readErr := resp.Body.Read(buf)
		if n > 0 {
			if _, err := file.Write(buf[:n]); err != nil {
				s.failTask(task.ID, err)
				return
			}
			written += int64(n)
			limiter.wait(n) // enforce the per-task / profile speed cap
		}
		now := time.Now()
		if n > 0 && now.Sub(lastTick) >= time.Second {
			s.updateProgress(task.ID, written, totalSize, written-lastBytes, now.Sub(lastTick))
			limiter.setRate(s.effectiveLimit(task.ID)) // pick up live profile/limit changes
			lastBytes = written
			lastTick = now
		}
		if readErr == io.EOF {
			break
		}
		if readErr != nil {
			if ctx.Err() != nil {
				return
			}
			s.failTask(task.ID, readErr)
			return
		}
	}
	if err := file.Close(); err != nil {
		s.failTask(task.ID, err)
		return
	}
	if err := os.Rename(partPath, finalPath); err != nil {
		s.failTask(task.ID, err)
		return
	}
	s.completeHTTPTask(task.ID, finalPath, written)
}

func (s *Service) probeHTTPDownload(ctx context.Context, rawURL string) (httpProbe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodHead, rawURL, nil)
	if err != nil {
		return httpProbe{}, err
	}
	prepareHTTPRequest(req)
	resp, err := s.client.Do(req)
	if err != nil {
		return httpProbe{}, err
	}
	defer resp.Body.Close()
	probe := httpProbe{
		StatusCode:   resp.StatusCode,
		Size:         resp.ContentLength,
		Filename:     downloadFileNameFromHeaders(rawURL, resp.Header),
		AcceptRanges: strings.EqualFold(resp.Header.Get("Accept-Ranges"), "bytes"),
	}
	if resp.StatusCode == http.StatusForbidden || resp.StatusCode == http.StatusMethodNotAllowed || resp.StatusCode == http.StatusNotImplemented {
		return s.probeHTTPWithRange(ctx, rawURL)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		probe.Body = string(body)
	}
	return probe, nil
}

func (s *Service) probeHTTPWithRange(ctx context.Context, rawURL string) (httpProbe, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return httpProbe{}, err
	}
	prepareHTTPRequest(req)
	req.Header.Set("Range", "bytes=0-0")
	resp, err := s.client.Do(req)
	if err != nil {
		return httpProbe{}, err
	}
	defer resp.Body.Close()
	probe := httpProbe{
		StatusCode:   resp.StatusCode,
		Size:         contentRangeTotal(resp.Header.Get("Content-Range")),
		Filename:     downloadFileNameFromHeaders(rawURL, resp.Header),
		AcceptRanges: resp.StatusCode == http.StatusPartialContent,
	}
	if probe.Size <= 0 && resp.ContentLength > 0 && resp.StatusCode == http.StatusOK {
		probe.Size = resp.ContentLength
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		probe.Body = string(body)
	}
	return probe, nil
}

func (s *Service) completeHTTPTask(id int, finalPath string, written int64) {
	s.updateTask(id, func(t *DownloadTask) {
		t.Status = StatusCompleted
		t.Progress = 100
		t.Speed = "0 KB/s"
		t.Size = formatBytes(written)
		t.FilePath = finalPath
		t.Handling = "已下载到 " + finalPath
		t.Error = ""
	})
}

func (s *Service) runRSS(ctx context.Context, task DownloadTask) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, task.Link, nil)
	if err != nil {
		s.failTask(task.ID, err)
		return
	}
	prepareHTTPRequest(req)
	resp, err := s.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		s.failTask(task.ID, err)
		return
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		s.failTask(task.ID, fmt.Errorf("rss status %d", resp.StatusCode))
		return
	}
	var parsed feed
	if err := xml.NewDecoder(resp.Body).Decode(&parsed); err != nil {
		s.failTask(task.ID, err)
		return
	}

	title, link := firstDownloadableFeedEntry(parsed)
	if link == "" {
		s.updateTask(task.ID, func(t *DownloadTask) {
			t.Status = StatusCompleted
			t.Progress = 100
			t.Speed = "0 KB/s"
			t.Size = "0 个可下载条目"
			t.Handling = "订阅已读取，未发现可直接下载的附件"
			t.Error = ""
		})
		return
	}
	s.updateTask(task.ID, func(t *DownloadTask) {
		t.Status = StatusCompleted
		t.Progress = 100
		t.Speed = "0 KB/s"
		t.Size = "已解析 1 个条目"
		t.Handling = "已发现订阅条目：" + title
		t.Error = ""
	})
	child, err := s.CreateTask(ctx, CreateTaskRequest{
		Source:   SourceHTTP,
		Link:     link,
		Name:     title,
		Category: task.Category,
	})
	if err != nil {
		s.updateTask(task.ID, func(t *DownloadTask) {
			t.Error = err.Error()
			t.Handling = "订阅条目添加失败：" + err.Error()
		})
		return
	}
	s.updateTask(task.ID, func(t *DownloadTask) {
		t.Handling = "已添加订阅下载：" + child.Name
	})
}

func (s *Service) runAria2(ctx context.Context, task DownloadTask) {
	aria2, err := exec.LookPath("aria2c")
	if err != nil {
		s.failTask(task.ID, errors.New("aria2c 未安装，无法执行 BT 或磁力下载"))
		return
	}
	dir := s.aria2TaskDir(task)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		s.failTask(task.ID, err)
		return
	}
	s.updateTask(task.ID, func(t *DownloadTask) {
		t.FilePath = dir
		t.Handling = "正在下载到 " + dir
		t.Error = ""
	})
	cmd := exec.CommandContext(ctx, aria2,
		"--dir", dir,
		"--auto-file-renaming=false",
		"--allow-overwrite=true",
		"--summary-interval=0",
		"--console-log-level=warn",
		task.Link,
	)
	output, err := cmd.CombinedOutput()
	if err != nil {
		if ctx.Err() != nil {
			return
		}
		s.failTask(task.ID, fmt.Errorf("aria2c failed: %s", strings.TrimSpace(string(output))))
		return
	}
	s.updateTask(task.ID, func(t *DownloadTask) {
		t.Status = StatusCompleted
		t.Progress = 100
		t.Speed = "0 KB/s"
		t.FilePath = dir
		t.Handling = "已下载到 " + dir
		t.Error = ""
	})
}

func (s *Service) aria2TaskDir(task DownloadTask) string {
	name := safePathPart(task.Name)
	if name == "" {
		name = safePathPart(string(task.Source))
	}
	return filepath.Join(s.downloadDir, safePathPart(task.Category), fmt.Sprintf("%03d-%s", task.ID, name))
}

func (s *Service) updateProgress(id int, written int64, total int64, delta int64, elapsed time.Duration) {
	s.updateTask(id, func(t *DownloadTask) {
		if total > 0 {
			progress := int(float64(written) / float64(total) * 100)
			if progress > 99 {
				progress = 99
			}
			t.Progress = progress
			t.Size = formatBytes(total)
		} else {
			t.Progress = 1
			t.Size = formatBytes(written)
		}
		if elapsed > 0 {
			t.Speed = formatBytes(int64(float64(delta)/elapsed.Seconds())) + "/s"
		}
	})
}

func (s *Service) failTask(id int, err error) {
	s.updateTask(id, func(t *DownloadTask) {
		t.Status = StatusFailed
		t.Speed = "0 KB/s"
		t.Handling = "下载失败：" + err.Error()
		t.Error = err.Error()
	})
}

func (s *Service) updateTask(id int, mutate func(*DownloadTask)) {
	s.mu.Lock()
	idx := s.findTaskIndex(id)
	if idx < 0 {
		s.mu.Unlock()
		return
	}
	mutate(&s.tasks[idx])
	_ = s.saveLocked()
	snapshot := cloneTask(s.tasks[idx])
	s.mu.Unlock()
	// Mirror the new progress/status onto the adopted central task outside the
	// lock so a slow runtime never stalls the download loop.
	s.mirrorToCentral(snapshot)
}

func (s *Service) deleteTaskFilesLocked(task DownloadTask) (int, error) {
	targets := taskDeleteTargets(task)
	deleted := 0
	for _, target := range targets {
		if !s.isSafeDownloadPath(target, task.Category) {
			continue
		}
		if _, err := os.Stat(target); errors.Is(err, os.ErrNotExist) {
			continue
		} else if err != nil {
			return deleted, err
		}
		if err := os.RemoveAll(target); err != nil {
			return deleted, err
		}
		deleted++
	}
	return deleted, nil
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Tasks:         cloneTasks(s.tasks),
		Profiles:      append([]SpeedProfile(nil), s.profiles...),
		NextID:        s.nextID,
		MaxConcurrent: s.maxConcurrent,
	})
}

func (s *Service) findTaskIndex(id int) int {
	for idx := range s.tasks {
		if s.tasks[idx].ID == id {
			return idx
		}
	}
	return -1
}

func (s *Service) activeProfileLocked() SpeedProfile {
	for _, profile := range s.profiles {
		if profile.Active {
			return profile
		}
	}
	if len(s.profiles) == 0 {
		return SpeedProfile{DownloadLimit: "不限速", UploadLimit: "不限速"}
	}
	return s.profiles[0]
}

func seedSpeedProfiles() []SpeedProfile {
	return []SpeedProfile{
		{Name: "智能限速", DownloadLimit: "18 MB/s", UploadLimit: "2 MB/s", Note: "自动限制后台下载，保留前台播放和访问带宽", Active: true},
		{Name: "夜间全速", DownloadLimit: "不限速", UploadLimit: "8 MB/s", Note: "夜间任务优先跑满带宽"},
		{Name: "家庭优先", DownloadLimit: "6 MB/s", UploadLimit: "1 MB/s", Note: "降低下载占用，优先保障播放和远程访问"},
	}
}

func normalizeSource(source SourceType, link string) (SourceType, error) {
	if source != "" {
		switch source {
		case SourceBT, SourceHTTP, SourceMagnet, SourceRSS:
			return source, nil
		default:
			return "", fmt.Errorf("%w: unsupported source %q", ErrInvalidTaskInput, source)
		}
	}

	lower := strings.ToLower(link)
	switch {
	case strings.HasPrefix(lower, "magnet:"):
		return SourceMagnet, nil
	case strings.HasSuffix(lowerPath(lower), ".torrent"):
		return SourceBT, nil
	case strings.HasSuffix(lowerPath(lower), ".rss"), strings.HasSuffix(lowerPath(lower), ".xml"), strings.Contains(lower, "/feed"):
		return SourceRSS, nil
	case strings.HasPrefix(lower, "http://"), strings.HasPrefix(lower, "https://"):
		return SourceHTTP, nil
	default:
		return "", fmt.Errorf("%w: cannot infer source from link", ErrInvalidTaskInput)
	}
}

func normalizeCategory(category string, source SourceType, link string, name string) string {
	category = strings.TrimSpace(category)
	if category != "" && category != "全部" {
		return category
	}
	if source == SourceRSS {
		return "订阅"
	}

	text := strings.ToLower(link + " " + name)
	switch {
	case containsAny(text, ".flac", ".mp3", ".wav", "music", "音乐"):
		return "音乐"
	case containsAny(text, ".iso", ".dmg", ".pkg", ".exe", ".appimage", "ubuntu", "server", "软件"):
		return "软件"
	case containsAny(text, ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".txt", ".md", "文档"):
		return "文档"
	default:
		return "影视"
	}
}

func taskName(name string, source SourceType, link string) string {
	name = strings.TrimSpace(name)
	if name != "" {
		return name
	}
	base := linkBaseName(link)
	if base != "" {
		return base
	}
	return string(source) + " 新任务"
}

func archiveRuleFor(category string, source SourceType, name string, link string) ArchiveRule {
	rule := ArchiveRule{
		Category:       category,
		TargetPath:     "/" + category,
		Tags:           []string{category},
		IndexAfterMove: true,
	}

	switch category {
	case "影视":
		rule.TargetPath = "/Media/TV"
		rule.ScrapeMetadata = true
		rule.Tags = []string{"影视"}
		if containsAny(strings.ToLower(name+" "+link), "电影", "movie", "film") {
			rule.TargetPath = "/Media/Movies"
		}
	case "音乐":
		rule.TargetPath = "/Music"
	case "软件":
		rule.TargetPath = "/Downloads/Software"
		rule.VerifyChecksum = true
	case "文档":
		rule.TargetPath = "/Documents"
	case "订阅":
		rule.TargetPath = "/Downloads/Subscriptions"
	}
	if source == SourceRSS {
		rule.TargetPath = "/Downloads/Subscriptions"
	}
	return rule
}

func handlingFor(category string, source SourceType, rule ArchiveRule, name string, link string) string {
	if source == SourceRSS {
		return "订阅条目会加入下载队列"
	}
	if category == "软件" || rule.VerifyChecksum {
		return "完成后可校验 SHA256"
	}
	if category == "影视" && rule.TargetPath == "/Media/TV" {
		return "完成后可归档到 /Media/TV"
	}
	if category == "影视" && containsAny(strings.ToLower(name+" "+link), "电影", "movie", "film") {
		return "完成后可归档到 /Media/Movies"
	}
	return "完成后可归档到 " + rule.TargetPath
}

func firstDownloadableFeedEntry(parsed feed) (string, string) {
	for _, item := range parsed.Channel.Items {
		title := strings.TrimSpace(item.Title)
		for _, enclosure := range item.Enclosures {
			if isHTTPURL(enclosure.URL) {
				return fallbackTitle(title, enclosure.URL), enclosure.URL
			}
		}
		if isDirectDownloadURL(item.Link) {
			return fallbackTitle(title, item.Link), strings.TrimSpace(item.Link)
		}
	}
	for _, entry := range parsed.Entries {
		title := strings.TrimSpace(entry.Title)
		for _, link := range entry.Links {
			href := strings.TrimSpace(link.Href)
			if isDirectDownloadURL(href) || strings.Contains(strings.ToLower(link.Type), "audio") || strings.Contains(strings.ToLower(link.Type), "video") {
				return fallbackTitle(title, href), href
			}
		}
	}
	return "", ""
}

func prepareHTTPRequest(req *http.Request) {
	req.Header.Set("User-Agent", "HiGoOS DownloadCenter/1.0")
	req.Header.Set("Accept", "*/*")
	req.Header.Set("Accept-Encoding", "identity")
}

func downloadFileNameFromHeaders(rawURL string, header http.Header) string {
	if disposition := header.Get("Content-Disposition"); disposition != "" {
		if _, params, err := mime.ParseMediaType(disposition); err == nil {
			if filename := strings.TrimSpace(params["filename"]); filename != "" {
				return safeFilename(filename)
			}
		}
	}
	if base := linkBaseName(rawURL); base != "" {
		return safeFilename(base)
	}
	return ""
}

func linkBaseName(raw string) string {
	parsed, err := url.Parse(raw)
	if err == nil && parsed.Path != "" {
		base := path.Base(parsed.Path)
		if base != "." && base != "/" && base != "" {
			return base
		}
	}
	base := path.Base(lowerPath(raw))
	if base != "." && base != "/" && base != "" {
		return base
	}
	return ""
}

func uniquePath(raw string) string {
	if _, err := os.Stat(raw); errors.Is(err, os.ErrNotExist) {
		return raw
	}
	ext := filepath.Ext(raw)
	stem := strings.TrimSuffix(raw, ext)
	for i := 1; i < 1000; i++ {
		candidate := stem + "-" + strconv.Itoa(i) + ext
		if _, err := os.Stat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate
		}
	}
	return stem + "-" + strconv.FormatInt(time.Now().UnixNano(), 10) + ext
}

func taskDeleteTargets(task DownloadTask) []string {
	filePath := strings.TrimSpace(task.FilePath)
	if filePath == "" {
		return nil
	}
	targets := []string{filePath}
	if !strings.HasSuffix(filePath, ".part") {
		targets = append(targets, filePath+".part")
	}
	return targets
}

func (s *Service) isSafeDownloadPath(raw string, category string) bool {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return false
	}
	base, err := filepath.Abs(s.downloadDir)
	if err != nil {
		return false
	}
	target, err := filepath.Abs(raw)
	if err != nil {
		return false
	}
	if target == base {
		return false
	}
	if categoryDir, err := filepath.Abs(filepath.Join(s.downloadDir, safePathPart(category))); err == nil && target == categoryDir {
		return false
	}
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return rel != "." && rel != "" && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".."
}

func existingFileSize(raw string) int64 {
	info, err := os.Stat(raw)
	if err != nil || info.IsDir() {
		return 0
	}
	return info.Size()
}

func contentRangeTotal(raw string) int64 {
	if raw == "" {
		return 0
	}
	idx := strings.LastIndex(raw, "/")
	if idx < 0 || idx == len(raw)-1 {
		return 0
	}
	total, err := strconv.ParseInt(strings.TrimSpace(raw[idx+1:]), 10, 64)
	if err != nil {
		return 0
	}
	return total
}

func errorDetail(body string) string {
	body = strings.TrimSpace(body)
	if body == "" {
		return ""
	}
	body = strings.Join(strings.Fields(body), " ")
	if len(body) > 160 {
		body = body[:160] + "..."
	}
	return ": " + body
}

func safeFilename(name string) string {
	name = strings.TrimSpace(name)
	if name == "" {
		return "download-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	name = filepath.Base(name)
	name = invalidFilename.ReplaceAllString(name, "_")
	if name == "." || name == "/" || name == "" {
		return "download-" + strconv.FormatInt(time.Now().Unix(), 10)
	}
	return name
}

func safePathPart(name string) string {
	name = safeFilename(name)
	if name == "." || name == string(filepath.Separator) {
		return "Downloads"
	}
	return name
}

func formatBytes(size int64) string {
	if size < 0 {
		return "未知"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	value := float64(size)
	unit := units[0]
	for idx := 0; idx < len(units)-1 && value >= 1024; idx++ {
		value /= 1024
		unit = units[idx+1]
	}
	if unit == "B" {
		return fmt.Sprintf("%d B", size)
	}
	return fmt.Sprintf("%.1f %s", value, unit)
}

func taskFilePath(task DownloadTask) string {
	if task.FilePath != "" {
		return task.FilePath
	}
	return task.ArchiveRule.TargetPath
}

func lowerPath(raw string) string {
	if idx := strings.IndexAny(raw, "?#"); idx >= 0 {
		return raw[:idx]
	}
	return raw
}

func containsAny(text string, needles ...string) bool {
	for _, needle := range needles {
		if strings.Contains(text, strings.ToLower(needle)) {
			return true
		}
	}
	return false
}

func isHTTPURL(raw string) bool {
	lower := strings.ToLower(strings.TrimSpace(raw))
	return strings.HasPrefix(lower, "http://") || strings.HasPrefix(lower, "https://")
}

func isDirectDownloadURL(raw string) bool {
	if !isHTTPURL(raw) {
		return false
	}
	lower := strings.ToLower(lowerPath(raw))
	return containsAny(lower, ".mp3", ".m4a", ".mp4", ".mkv", ".flac", ".zip", ".iso", ".torrent", ".pdf")
}

func fallbackTitle(title string, raw string) string {
	title = strings.TrimSpace(title)
	if title != "" {
		return title
	}
	if base := linkBaseName(raw); base != "" {
		return base
	}
	return "订阅下载"
}

func cloneTasks(tasks []DownloadTask) []DownloadTask {
	copied := make([]DownloadTask, len(tasks))
	for idx, task := range tasks {
		copied[idx] = cloneTask(task)
	}
	return copied
}

func cloneTask(task DownloadTask) DownloadTask {
	task.ArchiveRule.Tags = append([]string(nil), task.ArchiveRule.Tags...)
	return task
}

func filterPersistedTasks(tasks []DownloadTask) []DownloadTask {
	filtered := make([]DownloadTask, 0, len(tasks))
	for _, task := range tasks {
		if isLegacyDemoTask(task) {
			continue
		}
		if task.Status == StatusRunning || task.Status == StatusQueued {
			task.Status = StatusPaused
			task.Speed = "0 KB/s"
			task.Handling = "服务重启后已暂停，可继续任务"
		}
		filtered = append(filtered, task)
	}
	return filtered
}

func isLegacyDemoTask(task DownloadTask) bool {
	if strings.TrimSpace(task.Link) != "" || strings.TrimSpace(task.FilePath) != "" {
		return false
	}
	switch task.Name {
	case "纪录片合集 S02", "家庭音乐精选 FLAC", "Ubuntu Server 镜像", "每周公开课订阅":
		return true
	default:
		return false
	}
}

func nextTaskID(tasks []DownloadTask) int {
	next := 1
	for _, task := range tasks {
		if task.ID >= next {
			next = task.ID + 1
		}
	}
	return next
}

var invalidFilename = regexp.MustCompile(`[\\/:*?"<>|]+`)
