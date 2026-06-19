package video

import (
	"bufio"
	"bytes"
	"context"
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"html"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

var videoExtensions = map[string]string{
	".mp4":  "MP4",
	".m4v":  "MP4",
	".mkv":  "Matroska",
	".avi":  "AVI",
	".mov":  "QuickTime",
	".wmv":  "WMV",
	".webm": "WebM",
	".ts":   "MPEG-TS",
	".m2ts": "MPEG-TS",
}

var (
	episodePattern  = regexp.MustCompile(`(?i)s(\d{1,2})e(\d{1,3})`)
	yearPattern     = regexp.MustCompile(`(?:^|[^0-9])((?:19|20)\d{2})(?:[^0-9]|$)`)
	qualityPattern  = regexp.MustCompile(`(?i)\b(2160p|1080p|720p|480p|4k|8k|uhd|hdr|x264|x265|h264|h265|hevc|bluray|web[- ]?dl|webrip|bdrip|dvdrip)\b`)
	releasePattern  = regexp.MustCompile(`(?i)(中英双语双字|中英双语|国粤双语|双语|双字|中字|中文字幕|简繁字幕|内嵌字幕|外挂字幕|国语|粤语|英语|日语|韩语|无删减|加长版|导演剪辑版)`)
	doubanIDPattern = regexp.MustCompile(`(?i)[\[\{（(](?:doubanid|dbid)-?(\d+)[\]\}）)]`)
)

type Service struct {
	mu           sync.RWMutex
	settings     LibrarySettings
	items        []Item
	tasks        []Task
	liveSources  []LiveSource
	channels     []LiveChannel
	guideSources []LiveGuideSource
	programs     []LiveProgram
	dvrSettings  DVRSettings
	timers       []RecordingTimer
	recordings   []RecordingItem
	nextTaskSeq  int
	statePath    string
	activeDVR    map[string]*exec.Cmd
	dvrOnce      sync.Once
	runner       *tasks.Manager
}

type snapshot struct {
	Settings     LibrarySettings   `json:"settings"`
	Items        []Item            `json:"items"`
	Tasks        []Task            `json:"tasks"`
	LiveSources  []LiveSource      `json:"liveSources"`
	Channels     []LiveChannel     `json:"channels"`
	GuideSources []LiveGuideSource `json:"guideSources"`
	Programs     []LiveProgram     `json:"programs"`
	DVRSettings  DVRSettings       `json:"dvrSettings"`
	Timers       []RecordingTimer  `json:"timers"`
	Recordings   []RecordingItem   `json:"recordings"`
	NextTaskSeq  int               `json:"nextTaskSeq"`
}

type metadataDocument struct {
	Title           string   `json:"title"`
	OriginalTitle   string   `json:"originalTitle"`
	SeriesTitle     string   `json:"seriesTitle"`
	EpisodeTitle    string   `json:"episodeTitle"`
	Year            string   `json:"year"`
	Season          int      `json:"season"`
	Episode         int      `json:"episode"`
	Overview        string   `json:"overview"`
	EpisodeOverview string   `json:"episodeOverview"`
	Rating          string   `json:"rating"`
	ContentRating   string   `json:"contentRating"`
	ReleaseDate     string   `json:"releaseDate"`
	Tagline         string   `json:"tagline"`
	Poster          string   `json:"poster"`
	Backdrop        string   `json:"backdrop"`
	MetadataSource  string   `json:"metadataSource"`
	ProviderID      string   `json:"providerId"`
	Genres          []string `json:"genres"`
	Tags            []string `json:"tags"`
	Directors       []string `json:"directors"`
	Writers         []string `json:"writers"`
	Actors          []string `json:"actors"`
	Studios         []string `json:"studios"`
	Countries       []string `json:"countries"`
}

type nfoDocument struct {
	Title         string     `xml:"title"`
	Original      string     `xml:"originaltitle"`
	SortTitle     string     `xml:"sorttitle"`
	Year          string     `xml:"year"`
	Premiered     string     `xml:"premiered"`
	ReleaseDate   string     `xml:"releasedate"`
	Plot          string     `xml:"plot"`
	Outline       string     `xml:"outline"`
	Rating        string     `xml:"rating"`
	ContentRating string     `xml:"mpaa"`
	Tagline       string     `xml:"tagline"`
	Genres        []string   `xml:"genre"`
	Tags          []string   `xml:"tag"`
	Directors     []string   `xml:"director"`
	Writers       []string   `xml:"writer"`
	Studios       []string   `xml:"studio"`
	Countries     []string   `xml:"country"`
	Actors        []nfoActor `xml:"actor"`
	Season        int        `xml:"season"`
	Episode       int        `xml:"episode"`
	EpisodeTitle  string     `xml:"episodetitle"`
	ShowTitle     string     `xml:"showtitle"`
}

type nfoActor struct {
	Name string `xml:"name"`
}

type suggestionDocument struct {
	ID       string `json:"id"`
	Title    string `json:"title"`
	SubTitle string `json:"sub_title"`
	Year     string `json:"year"`
	Image    string `json:"img"`
	Type     string `json:"type"`
}

type probeResult struct {
	Codec           string
	Resolution      string
	DurationSeconds int
	VideoTracks     []MediaTrack
	AudioTracks     []MediaTrack
	SubtitleTracks  []MediaTrack
}

type ffprobeDocument struct {
	Streams []ffprobeStream `json:"streams"`
	Format  struct {
		Duration string `json:"duration"`
	} `json:"format"`
}

type ffprobeStream struct {
	Index       int               `json:"index"`
	CodecType   string            `json:"codec_type"`
	CodecName   string            `json:"codec_name"`
	Width       int               `json:"width"`
	Height      int               `json:"height"`
	Channels    int               `json:"channels"`
	Tags        map[string]string `json:"tags"`
	Disposition map[string]int    `json:"disposition"`
}

func NewService() *Service {
	return &Service{
		settings: LibrarySettings{
			Libraries: []Library{},
			Status:    "未设置媒体库",
		},
		dvrSettings: defaultDVRSettings(),
		nextTaskSeq: 1,
		activeDVR:   map[string]*exec.Cmd{},
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "video.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Settings.Libraries) > 0 || len(persisted.Items) > 0 || len(persisted.LiveSources) > 0 || len(persisted.GuideSources) > 0 || len(persisted.Timers) > 0 {
		service.settings = persisted.Settings
		service.items = cloneItems(persisted.Items)
		service.tasks = append([]Task(nil), persisted.Tasks...)
		service.liveSources = cloneLiveSources(persisted.LiveSources)
		service.channels = cloneLiveChannels(persisted.Channels)
		service.guideSources = cloneGuideSources(persisted.GuideSources)
		service.programs = clonePrograms(persisted.Programs)
		if persisted.DVRSettings != (DVRSettings{}) {
			service.dvrSettings = normalizeDVRSettings(persisted.DVRSettings)
		}
		service.timers = cloneRecordingTimers(persisted.Timers)
		service.recordings = cloneRecordings(persisted.Recordings)
		service.nextTaskSeq = persisted.NextTaskSeq
		if service.nextTaskSeq <= 0 {
			service.nextTaskSeq = 1
		}
		service.refreshCountsLocked()
	}
	return service, nil
}

func (s *Service) Settings(ctx context.Context) (LibrarySettings, error) {
	if err := ctx.Err(); err != nil {
		return LibrarySettings{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneSettings(s.settings), nil
}

func (s *Service) UpdateSettings(ctx context.Context, request LibraryUpdateRequest) (LibrarySettings, error) {
	if err := ctx.Err(); err != nil {
		return LibrarySettings{}, err
	}
	libraries, err := normalizeLibraries(request.Libraries)
	if err != nil {
		return LibrarySettings{}, err
	}
	s.mu.Lock()
	s.settings.Libraries = libraries
	s.settings.Status = "媒体库已设置，等待扫描"
	s.refreshCountsLocked()
	settings := cloneSettings(s.settings)
	err = s.saveLocked()
	s.mu.Unlock()
	return settings, err
}

func (s *Service) CreateLibrary(ctx context.Context, request CreateLibraryRequest) (Library, error) {
	if err := ctx.Err(); err != nil {
		return Library{}, err
	}
	libraries, err := normalizeLibraries([]Library{{
		Name:              request.Name,
		Type:              request.Type,
		Paths:             request.Paths,
		MetadataLanguage:  request.MetadataLanguage,
		AllowAdultContent: request.AllowAdultContent,
		AutoSubtitles:     request.AutoSubtitles,
		SubtitleLanguage:  request.SubtitleLanguage,
	}})
	if err != nil {
		return Library{}, err
	}
	library := libraries[0]
	s.mu.Lock()
	s.settings.Libraries = append(s.settings.Libraries, library)
	s.settings.Status = "媒体库已设置，等待扫描"
	err = s.saveLocked()
	s.mu.Unlock()
	return library, err
}

func (s *Service) DeleteLibrary(ctx context.Context, id string) (DeleteLibraryResult, error) {
	if err := ctx.Err(); err != nil {
		return DeleteLibraryResult{}, err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return DeleteLibraryResult{}, fmt.Errorf("library id is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	libraryIndex := -1
	for idx, library := range s.settings.Libraries {
		if library.ID == id {
			libraryIndex = idx
			break
		}
	}
	if libraryIndex < 0 {
		return DeleteLibraryResult{}, fmt.Errorf("video library not found: %s", id)
	}

	s.settings.Libraries = append(s.settings.Libraries[:libraryIndex], s.settings.Libraries[libraryIndex+1:]...)

	removedItemIDs := map[string]struct{}{}
	nextItems := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if item.LibraryID == id {
			removedItemIDs[item.ID] = struct{}{}
			continue
		}
		nextItems = append(nextItems, item)
	}
	s.items = nextItems

	nextTasks := make([]Task, 0, len(s.tasks))
	removedTasks := 0
	for _, task := range s.tasks {
		if _, ok := removedItemIDs[task.ItemID]; ok {
			removedTasks++
			continue
		}
		nextTasks = append(nextTasks, task)
	}
	s.tasks = nextTasks

	switch {
	case len(s.settings.Libraries) == 0:
		s.settings.Status = "未设置媒体库"
	case len(s.items) == 0:
		s.settings.Status = "已设置媒体库，等待扫描"
	default:
		s.settings.Status = fmt.Sprintf("已删除媒体库，剩余 %d 个影视文件", len(s.items))
	}
	s.refreshCountsLocked()

	settings := cloneSettings(s.settings)
	result := DeleteLibraryResult{
		ID:           id,
		RemovedItems: len(removedItemIDs),
		RemovedTasks: removedTasks,
		Settings:     settings,
	}
	return result, s.saveLocked()
}

func (s *Service) Scan(ctx context.Context) (ScanResult, error) {
	if err := ctx.Err(); err != nil {
		return ScanResult{}, err
	}
	s.mu.RLock()
	libraries := cloneLibraries(s.settings.Libraries)
	s.mu.RUnlock()
	if len(libraries) == 0 {
		return ScanResult{}, fmt.Errorf("video library path is required")
	}

	now := time.Now().UTC()
	items := make([]Item, 0)
	for _, library := range libraries {
		for _, root := range library.Paths {
			if err := ctx.Err(); err != nil {
				return ScanResult{}, err
			}
			info, err := os.Stat(root)
			if err != nil {
				return ScanResult{}, fmt.Errorf("stat library path %s: %w", root, err)
			}
			if !info.IsDir() {
				return ScanResult{}, fmt.Errorf("library path is not a directory: %s", root)
			}
			err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
				if walkErr != nil {
					return nil
				}
				if entry.IsDir() {
					if strings.HasPrefix(entry.Name(), ".") && path != root {
						return filepath.SkipDir
					}
					return nil
				}
				ext := strings.ToLower(filepath.Ext(entry.Name()))
				container, ok := videoExtensions[ext]
				if !ok {
					return nil
				}
				item, err := itemFromFile(root, path, library, container, now)
				if err == nil {
					items = append(items, item)
				}
				return nil
			})
			if err != nil {
				return ScanResult{}, err
			}
		}
	}
	sort.Slice(items, func(i, j int) bool {
		if items[i].LibraryName == items[j].LibraryName {
			return items[i].Title < items[j].Title
		}
		return items[i].LibraryName < items[j].LibraryName
	})

	s.mu.Lock()
	s.items = items
	s.settings.ItemCount = len(items)
	s.settings.LastScan = now.Format(time.RFC3339)
	s.settings.Status = "扫描完成"
	s.refreshCountsLocked()
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return ScanResult{}, err
	}
	return ScanResult{
		ID:        "video-scan-" + now.Format("20060102150405"),
		State:     "done",
		Message:   fmt.Sprintf("已扫描 %d 个影视文件。", len(items)),
		ItemCount: len(items),
		ScannedAt: now.Format(time.RFC3339),
	}, nil
}

func (s *Service) Items(ctx context.Context, query, libraryID, kind string) ([]Item, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	query = strings.ToLower(strings.TrimSpace(query))
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]Item, 0, len(s.items))
	for _, item := range s.items {
		if libraryID != "" && item.LibraryID != libraryID {
			continue
		}
		if kind != "" && item.Kind != kind {
			continue
		}
		if query != "" {
			haystack := strings.ToLower(item.Title + " " + item.OriginalTitle + " " + item.FileName + " " + strings.Join(item.Genres, " "))
			if !strings.Contains(haystack, query) {
				continue
			}
		}
		result = append(result, item)
	}
	return cloneItems(result), nil
}

func (s *Service) Item(ctx context.Context, id string) (Item, error) {
	if err := ctx.Err(); err != nil {
		return Item{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, item := range s.items {
		if item.ID == id {
			return cloneItem(item), nil
		}
	}
	return Item{}, fmt.Errorf("video item not found: %s", id)
}

// ApplyAnalysis writes the result of a background AI analysis pass back onto a
// video item: an LLM/vision overview, derived tags, and ffprobe technical
// metadata. It accepts plain values so the video package never depends on the
// analysis engine. Empty values are left untouched so a partial (degraded) pass
// does not clobber existing metadata.
func (s *Service) ApplyAnalysis(id, overview string, tags []string, container, codec, resolution string, durationSeconds int) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID != id {
			continue
		}
		if ov := strings.TrimSpace(overview); ov != "" {
			s.items[i].Overview = ov
		}
		if len(tags) > 0 {
			s.items[i].Tags = mergeUniqueStrings(s.items[i].Tags, tags)
		}
		if container != "" {
			s.items[i].Container = container
		}
		if codec != "" {
			s.items[i].Codec = codec
		}
		if resolution != "" {
			s.items[i].Resolution = resolution
		}
		if durationSeconds > 0 {
			s.items[i].DurationSeconds = durationSeconds
		}
		s.items[i].Status = "AI 已分析"
		return s.saveLocked()
	}
	return fmt.Errorf("video item not found: %s", id)
}

func mergeUniqueStrings(existing, additions []string) []string {
	seen := make(map[string]bool, len(existing))
	out := append([]string(nil), existing...)
	for _, v := range existing {
		seen[strings.ToLower(strings.TrimSpace(v))] = true
	}
	for _, v := range additions {
		v = strings.TrimSpace(v)
		key := strings.ToLower(v)
		if v == "" || seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, v)
	}
	return out
}

func (s *Service) OpenItem(ctx context.Context, id string) (*os.File, Item, error) {
	item, err := s.Item(ctx, id)
	if err != nil {
		return nil, Item{}, err
	}
	file, err := os.Open(item.Path)
	if err != nil {
		return nil, Item{}, err
	}
	return file, item, nil
}

func (s *Service) Poster(ctx context.Context, id string) ([]byte, string, time.Time, error) {
	item, err := s.Item(ctx, id)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if path := item.PosterPath; path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, contentTypeForPath(path), modTime(path), nil
		}
	}
	if path := findPosterPath(item.Path); path != "" {
		data, err := os.ReadFile(path)
		if err == nil {
			return data, contentTypeForPath(path), modTime(path), nil
		}
	}
	if item.PosterRemoteURL != "" {
		data, contentType, err := fetchRemoteImage(ctx, item.PosterRemoteURL)
		if err == nil {
			return data, contentType, time.Now(), nil
		}
	}
	return placeholderPoster(item.Title), "image/svg+xml; charset=utf-8", time.Now(), nil
}

func (s *Service) Subtitle(ctx context.Context, id string) ([]byte, string, time.Time, error) {
	item, err := s.Item(ctx, id)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	path := findSubtitlePath(item.Path)
	if path == "" {
		return nil, "", time.Time{}, fmt.Errorf("subtitle not found")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return data, contentTypeForPath(path), modTime(path), nil
}

func (s *Service) CreateScrapeTask(ctx context.Context, request CreateTaskRequest) (Task, error) {
	return s.createItemTask(ctx, "scrape", request.ItemID, "", func(item *Item) string {
		return s.scrapeItemMetadata(ctx, item)
	})
}

func (s *Service) CreateSubtitleTask(ctx context.Context, request CreateTaskRequest) (Task, error) {
	return s.createItemTask(ctx, "subtitle", request.ItemID, "", func(item *Item) string {
		if item.SubtitleURL == "" {
			item.Status = "未发现旁路字幕"
			return "未发现同名 srt/vtt 字幕，可放入同目录后重新扫描。"
		}
		item.Status = "字幕已关联"
		return "已关联同名旁路字幕。"
	})
}

func (s *Service) CreateTranscodeTask(ctx context.Context, request CreateTaskRequest) (Task, error) {
	profile := strings.TrimSpace(request.Profile)
	if profile == "" {
		profile = "1080p H.264"
	}
	if s.runner != nil {
		return s.startTranscode(ctx, request.ItemID, profile)
	}
	return s.createItemTask(ctx, "transcode", request.ItemID, profile, func(item *Item) string {
		if _, err := exec.LookPath("ffmpeg"); err != nil {
			item.Status = "等待安装转码组件"
			return "未检测到 ffmpeg，任务已记录，安装后可执行转码。"
		}
		item.Status = "转码任务已准备"
		return fmt.Sprintf("%s 转码任务已创建。", profile)
	})
}

// AttachTaskRunner wires the shared task runtime so transcode tasks really run
// ffmpeg in the background instead of completing synchronously without output.
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.runner = m
	m.Register(transcodeTaskKind, s.runTranscode)
}

const transcodeTaskKind = "video.transcode"

type transcodePayload struct {
	TaskID  string `json:"taskId"`
	ItemID  string `json:"itemId"`
	Source  string `json:"source"`
	Profile string `json:"profile"`
}

// startTranscode records a running transcode task and schedules the real ffmpeg
// job. The source path is captured now so the worker has it without re-locking.
func (s *Service) startTranscode(ctx context.Context, itemID, profile string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	idx := -1
	for i := range s.items {
		if s.items[i].ID == itemID {
			idx = i
			break
		}
	}
	if idx < 0 {
		s.mu.Unlock()
		return Task{}, fmt.Errorf("video item not found: %s", itemID)
	}
	source := s.items[idx].Path
	s.items[idx].Status = "移动端转码中"
	task := Task{
		ID:        s.nextJobIDLocked("transcode"),
		Type:      "transcode",
		ItemID:    itemID,
		Title:     s.items[idx].Title,
		Status:    JobRunning,
		Message:   fmt.Sprintf("%s 转码中…", profile),
		Progress:  5,
		Profile:   profile,
		CreatedAt: time.Now().UTC().Format(time.RFC3339),
	}
	s.tasks = append([]Task{task}, s.tasks...)
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return Task{}, err
	}
	s.mu.Unlock()

	if _, err := s.runner.Enqueue(transcodeTaskKind, transcodePayload{TaskID: task.ID, ItemID: itemID, Source: source, Profile: profile}); err != nil {
		return task, err
	}
	return task, nil
}

// runTranscode is the task handler that performs the real ffmpeg transcode.
func (s *Service) runTranscode(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload transcodePayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		s.updateTaskStatus(payload.TaskID, JobFailed, 100, "未检测到 ffmpeg，无法执行转码。")
		s.updateItemStatus(payload.ItemID, "等待安装转码组件")
		return nil, fmt.Errorf("ffmpeg not available")
	}
	if payload.Source == "" {
		s.updateTaskStatus(payload.TaskID, JobFailed, 100, "源文件路径缺失。")
		s.updateItemStatus(payload.ItemID, "转码失败：源文件缺失")
		return nil, fmt.Errorf("missing source path")
	}

	height := transcodeHeight(payload.Profile)
	// Write outputs into a dot-prefixed sibling dir so the library scanner (which
	// skips entries starting with ".") never re-indexes a transcode as a new
	// media item.
	base := strings.TrimSuffix(filepath.Base(payload.Source), filepath.Ext(payload.Source))
	outputDir := filepath.Join(filepath.Dir(payload.Source), ".transcoded")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		s.updateTaskStatus(payload.TaskID, JobFailed, 100, "无法创建转码输出目录："+err.Error())
		s.updateItemStatus(payload.ItemID, "转码失败")
		return nil, err
	}
	output := filepath.Join(outputDir, fmt.Sprintf("%s.%dp.mp4", base, height))
	h.Progress(20, "转码中")

	args := []string{
		"-y", "-i", payload.Source,
		"-vf", fmt.Sprintf("scale=-2:%d", height),
		"-c:v", "libx264", "-preset", "ultrafast",
		"-c:a", "aac",
		output,
	}
	cmd := exec.CommandContext(ctx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		s.updateTaskStatus(payload.TaskID, JobFailed, 100, "转码失败："+trimFFmpegError(stderr.String()))
		s.updateItemStatus(payload.ItemID, "转码失败")
		return nil, fmt.Errorf("ffmpeg transcode: %w", err)
	}
	s.updateTaskStatus(payload.TaskID, JobDone, 100, fmt.Sprintf("已完成 %dp 转码：%s", height, filepath.Base(output)))
	s.updateItemStatus(payload.ItemID, fmt.Sprintf("已生成 %dp 移动端版本", height))
	return json.Marshal(map[string]string{"taskId": payload.TaskID, "output": output})
}

func (s *Service) updateTaskStatus(taskID string, status JobStatus, progress int, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.tasks {
		if s.tasks[i].ID == taskID {
			s.tasks[i].Status = status
			s.tasks[i].Progress = progress
			s.tasks[i].Message = message
			_ = s.saveLocked()
			return
		}
	}
}

// updateItemStatus resets the media item's status after a transcode finishes so
// it does not stay stuck at "移动端转码中".
func (s *Service) updateItemStatus(itemID, status string) {
	if itemID == "" {
		return
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.items {
		if s.items[i].ID == itemID {
			s.items[i].Status = status
			_ = s.saveLocked()
			return
		}
	}
}

// transcodeHeight extracts a target vertical resolution from a profile label
// such as "720p H.264"; it defaults to 720 when none is present.
func transcodeHeight(profile string) int {
	for _, p := range []int{2160, 1440, 1080, 720, 480, 360} {
		if strings.Contains(profile, strconv.Itoa(p)) {
			return p
		}
	}
	return 720
}

func trimFFmpegError(stderr string) string {
	stderr = strings.TrimSpace(stderr)
	lines := strings.Split(stderr, "\n")
	if len(lines) == 0 {
		return "ffmpeg error"
	}
	return lines[len(lines)-1]
}

func (s *Service) Tasks(ctx context.Context) ([]Task, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]Task(nil), s.tasks...), nil
}

func (s *Service) LiveSources(ctx context.Context) ([]LiveSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneLiveSources(s.liveSources), nil
}

func (s *Service) CreateLiveSource(ctx context.Context, request CreateLiveSourceRequest) (LiveSource, error) {
	if err := ctx.Err(); err != nil {
		return LiveSource{}, err
	}
	name := strings.TrimSpace(request.Name)
	sourceURL := strings.TrimSpace(request.URL)
	if name == "" || sourceURL == "" {
		return LiveSource{}, fmt.Errorf("live source name and url are required")
	}
	channels, err := parseM3U(ctx, sourceURL)
	if err != nil {
		return LiveSource{}, err
	}
	sourceID := stableID(name + "\x00" + sourceURL)
	source := LiveSource{
		ID:           sourceID,
		Name:         name,
		URL:          sourceURL,
		UserAgent:    strings.TrimSpace(request.UserAgent),
		StreamLimit:  request.StreamLimit,
		ChannelCount: len(channels),
		Status:       "已加载",
	}
	for idx := range channels {
		channels[idx].SourceID = sourceID
		channels[idx].ID = stableID(sourceID + "\x00" + channels[idx].Name + "\x00" + channels[idx].URL)
	}
	s.mu.Lock()
	nextSources := make([]LiveSource, 0, len(s.liveSources)+1)
	for _, existing := range s.liveSources {
		if existing.ID != sourceID {
			nextSources = append(nextSources, existing)
		}
	}
	nextSources = append([]LiveSource{source}, nextSources...)
	nextChannels := make([]LiveChannel, 0, len(s.channels)+len(channels))
	for _, existing := range s.channels {
		if existing.SourceID != sourceID {
			nextChannels = append(nextChannels, existing)
		}
	}
	nextChannels = append(nextChannels, channels...)
	s.liveSources = nextSources
	s.channels = nextChannels
	err = s.saveLocked()
	s.mu.Unlock()
	return source, err
}

func (s *Service) LiveChannels(ctx context.Context, sourceID string) ([]LiveChannel, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]LiveChannel, 0, len(s.channels))
	for _, channel := range s.channels {
		if sourceID == "" || channel.SourceID == sourceID {
			result = append(result, channel)
		}
	}
	return cloneLiveChannels(result), nil
}

func (s *Service) GuideSources(ctx context.Context) ([]LiveGuideSource, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneGuideSources(s.guideSources), nil
}

func (s *Service) CreateGuideSource(ctx context.Context, request CreateGuideSourceRequest) (LiveGuideSource, error) {
	if err := ctx.Err(); err != nil {
		return LiveGuideSource{}, err
	}
	name := strings.TrimSpace(request.Name)
	sourceURL := strings.TrimSpace(request.URL)
	if name == "" || sourceURL == "" {
		return LiveGuideSource{}, fmt.Errorf("guide source name and url are required")
	}
	programs, err := parseXMLTV(ctx, sourceURL, strings.TrimSpace(request.UserAgent), s.snapshotChannels())
	if err != nil {
		return LiveGuideSource{}, err
	}
	now := time.Now().UTC().Format(time.RFC3339)
	sourceID := stableID("guide\x00" + name + "\x00" + sourceURL)
	source := LiveGuideSource{ID: sourceID, Name: name, URL: sourceURL, UserAgent: strings.TrimSpace(request.UserAgent), ProgramCount: len(programs), Status: "已加载", LastRefresh: now}
	for idx := range programs {
		programs[idx].GuideID = sourceID
	}
	s.mu.Lock()
	nextSources := make([]LiveGuideSource, 0, len(s.guideSources)+1)
	for _, existing := range s.guideSources {
		if existing.ID != sourceID {
			nextSources = append(nextSources, existing)
		}
	}
	nextSources = append([]LiveGuideSource{source}, nextSources...)
	nextPrograms := make([]LiveProgram, 0, len(s.programs)+len(programs))
	for _, existing := range s.programs {
		if existing.GuideID != sourceID {
			nextPrograms = append(nextPrograms, existing)
		}
	}
	nextPrograms = append(nextPrograms, programs...)
	sort.Slice(nextPrograms, func(i, j int) bool {
		if nextPrograms[i].StartAt == nextPrograms[j].StartAt {
			return nextPrograms[i].ChannelName < nextPrograms[j].ChannelName
		}
		return nextPrograms[i].StartAt < nextPrograms[j].StartAt
	})
	s.guideSources = nextSources
	s.programs = nextPrograms
	err = s.saveLocked()
	s.mu.Unlock()
	return source, err
}

func (s *Service) LivePrograms(ctx context.Context, query LiveProgramQuery) ([]LiveProgram, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	from, _ := parseOptionalRFC3339(query.From)
	to, _ := parseOptionalRFC3339(query.To)
	channelSource := map[string]string{}
	s.mu.RLock()
	for _, channel := range s.channels {
		channelSource[channel.ID] = channel.SourceID
	}
	result := make([]LiveProgram, 0, len(s.programs))
	for _, program := range s.programs {
		if query.ChannelID != "" && program.ChannelID != query.ChannelID {
			continue
		}
		if query.SourceID != "" && channelSource[program.ChannelID] != query.SourceID {
			continue
		}
		start, err := time.Parse(time.RFC3339, program.StartAt)
		if err == nil {
			if !from.IsZero() && start.Before(from) {
				continue
			}
			if !to.IsZero() && !start.Before(to) {
				continue
			}
		}
		result = append(result, cloneProgram(program))
	}
	s.mu.RUnlock()
	return result, nil
}

func (s *Service) DVRSettings(ctx context.Context) (DVRSettings, error) {
	if err := ctx.Err(); err != nil {
		return DVRSettings{}, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.dvrSettings, nil
}

func (s *Service) UpdateDVRSettings(ctx context.Context, settings DVRSettings) (DVRSettings, error) {
	if err := ctx.Err(); err != nil {
		return DVRSettings{}, err
	}
	normalized := normalizeDVRSettings(settings)
	s.mu.Lock()
	s.dvrSettings = normalized
	err := s.saveLocked()
	s.mu.Unlock()
	return normalized, err
}

func (s *Service) RecordingTimers(ctx context.Context) ([]RecordingTimer, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRecordingTimers(s.timers), nil
}

func (s *Service) CreateRecordingTimer(ctx context.Context, request CreateRecordingTimerRequest) (RecordingTimer, error) {
	if err := ctx.Err(); err != nil {
		return RecordingTimer{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	timer, err := s.buildRecordingTimerLocked(request)
	if err != nil {
		return RecordingTimer{}, err
	}
	s.timers = append([]RecordingTimer{timer}, s.timers...)
	recording := RecordingItem{
		ID:        stableID("recording\x00" + timer.ID),
		TimerID:   timer.ID,
		ProgramID: timer.ProgramID,
		Title:     timer.Name,
		ChannelID: timer.ChannelID,
		Status:    RecordingScheduled,
	}
	s.recordings = append([]RecordingItem{recording}, s.recordings...)
	if err := s.saveLocked(); err != nil {
		return RecordingTimer{}, err
	}
	if start, err := time.Parse(time.RFC3339, timer.StartAt); err == nil && !time.Now().UTC().Before(start) {
		go s.RunDVROnce(context.Background(), time.Now())
	}
	return timer, nil
}

func (s *Service) CancelRecordingTimer(ctx context.Context, id string) (RecordingTimer, error) {
	if err := ctx.Err(); err != nil {
		return RecordingTimer{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for idx := range s.timers {
		if s.timers[idx].ID == id {
			s.timers[idx].Status = RecordingCancelled
			if cmd := s.activeDVR[id]; cmd != nil && cmd.Process != nil {
				_ = cmd.Process.Kill()
				delete(s.activeDVR, id)
			}
			for recIdx := range s.recordings {
				if s.recordings[recIdx].TimerID == id {
					s.recordings[recIdx].Status = RecordingCancelled
				}
			}
			timer := s.timers[idx]
			return timer, s.saveLocked()
		}
	}
	return RecordingTimer{}, fmt.Errorf("recording timer not found")
}

func (s *Service) StartDVR(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = 20 * time.Second
	}
	s.dvrOnce.Do(func() {
		go func() {
			s.RunDVROnce(ctx, time.Now())
			ticker := time.NewTicker(interval)
			defer ticker.Stop()
			for {
				select {
				case <-ctx.Done():
					return
				case now := <-ticker.C:
					s.RunDVROnce(ctx, now)
				}
			}
		}()
	})
}

func (s *Service) RunDVROnce(ctx context.Context, now time.Time) {
	if err := ctx.Err(); err != nil {
		return
	}
	type recordingStart struct {
		timer   RecordingTimer
		channel LiveChannel
		path    string
	}
	starts := []recordingStart{}
	s.mu.Lock()
	settings := normalizeDVRSettings(s.dvrSettings)
	activeCount := len(s.activeDVR)
	for idx := range s.timers {
		if s.timers[idx].Status != RecordingScheduled {
			continue
		}
		start, startErr := time.Parse(time.RFC3339, s.timers[idx].StartAt)
		end, endErr := time.Parse(time.RFC3339, s.timers[idx].EndAt)
		if startErr != nil || endErr != nil {
			s.markRecordingFailedLocked(s.timers[idx].ID, "录制时间格式错误")
			continue
		}
		if !now.Before(end) {
			s.markRecordingFailedLocked(s.timers[idx].ID, "节目已结束，未执行录制")
			continue
		}
		if now.Before(start) {
			continue
		}
		if settings.MaxConcurrentRecord > 0 && activeCount >= settings.MaxConcurrentRecord {
			continue
		}
		channel, ok := s.channelByIDLocked(s.timers[idx].ChannelID)
		if !ok || strings.TrimSpace(channel.URL) == "" {
			s.markRecordingFailedLocked(s.timers[idx].ID, "未找到可录制的直播频道")
			continue
		}
		path := recordingOutputPath(s.timers[idx], settings)
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			s.markRecordingFailedLocked(s.timers[idx].ID, "创建录制目录失败："+err.Error())
			continue
		}
		s.timers[idx].Status = RecordingRunning
		nowText := now.UTC().Format(time.RFC3339)
		for recIdx := range s.recordings {
			if s.recordings[recIdx].TimerID == s.timers[idx].ID {
				s.recordings[recIdx].Status = RecordingRunning
				s.recordings[recIdx].Path = path
				s.recordings[recIdx].StartedAt = nowText
				s.recordings[recIdx].Message = "正在录制"
			}
		}
		starts = append(starts, recordingStart{timer: s.timers[idx], channel: channel, path: path})
		activeCount++
	}
	_ = s.saveLocked()
	s.mu.Unlock()
	for _, start := range starts {
		s.startRecordingProcess(ctx, start.timer, start.channel, start.path, now)
	}
}

func (s *Service) startRecordingProcess(ctx context.Context, timer RecordingTimer, channel LiveChannel, path string, now time.Time) {
	end, err := time.Parse(time.RFC3339, timer.EndAt)
	if err != nil {
		s.completeRecording(timer.ID, RecordingFailed, path, "录制结束时间格式错误")
		return
	}
	seconds := int(time.Until(end).Seconds())
	if !now.IsZero() {
		seconds = int(end.Sub(now).Seconds())
	}
	if seconds <= 0 {
		s.completeRecording(timer.ID, RecordingFailed, path, "节目已结束，未执行录制")
		return
	}
	if _, err := exec.LookPath("ffmpeg"); err != nil {
		s.completeRecording(timer.ID, RecordingFailed, path, "未检测到 ffmpeg，无法执行录制")
		return
	}
	recordCtx, cancel := context.WithTimeout(ctx, time.Duration(seconds+120)*time.Second)
	defer cancel()
	args := []string{
		"-y",
		"-hide_banner",
		"-loglevel", "warning",
		"-i", channel.URL,
		"-t", strconv.Itoa(seconds),
		"-c", "copy",
		"-movflags", "+faststart",
		path,
	}
	cmd := exec.CommandContext(recordCtx, "ffmpeg", args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	s.mu.Lock()
	if s.activeDVR == nil {
		s.activeDVR = map[string]*exec.Cmd{}
	}
	s.activeDVR[timer.ID] = cmd
	s.mu.Unlock()
	err = cmd.Run()
	message := strings.TrimSpace(stderr.String())
	if err != nil {
		if message == "" {
			message = err.Error()
		}
		s.completeRecording(timer.ID, RecordingFailed, path, message)
		return
	}
	if info, statErr := os.Stat(path); statErr != nil || info.Size() == 0 {
		if statErr != nil {
			message = statErr.Error()
		} else {
			message = "录制文件为空"
		}
		s.completeRecording(timer.ID, RecordingFailed, path, message)
		return
	}
	s.writeRecordingSidecars(timer, path)
	s.completeRecording(timer.ID, RecordingCompleted, path, "录制完成")
}

func (s *Service) completeRecording(timerID string, status RecordingStatus, path string, message string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.activeDVR, timerID)
	nowText := time.Now().UTC().Format(time.RFC3339)
	for idx := range s.timers {
		if s.timers[idx].ID == timerID {
			s.timers[idx].Status = status
		}
	}
	for idx := range s.recordings {
		if s.recordings[idx].TimerID == timerID {
			s.recordings[idx].Status = status
			s.recordings[idx].Path = path
			s.recordings[idx].EndedAt = nowText
			s.recordings[idx].Message = message
		}
	}
	_ = s.saveLocked()
}

func (s *Service) markRecordingFailedLocked(timerID string, message string) {
	nowText := time.Now().UTC().Format(time.RFC3339)
	for idx := range s.timers {
		if s.timers[idx].ID == timerID {
			s.timers[idx].Status = RecordingFailed
		}
	}
	for idx := range s.recordings {
		if s.recordings[idx].TimerID == timerID {
			s.recordings[idx].Status = RecordingFailed
			s.recordings[idx].EndedAt = nowText
			s.recordings[idx].Message = message
		}
	}
}

func (s *Service) channelByIDLocked(id string) (LiveChannel, bool) {
	for _, channel := range s.channels {
		if channel.ID == id {
			return sanitizeLiveChannel(channel), true
		}
	}
	return LiveChannel{}, false
}

func (s *Service) Recordings(ctx context.Context) ([]RecordingItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneRecordings(s.recordings), nil
}

func (s *Service) snapshotChannels() []LiveChannel {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneLiveChannels(s.channels)
}

func (s *Service) buildRecordingTimerLocked(request CreateRecordingTimerRequest) (RecordingTimer, error) {
	settings := normalizeDVRSettings(s.dvrSettings)
	var program LiveProgram
	if strings.TrimSpace(request.ProgramID) != "" {
		for _, candidate := range s.programs {
			if candidate.ID == request.ProgramID {
				program = candidate
				break
			}
		}
		if program.ID == "" {
			return RecordingTimer{}, fmt.Errorf("program not found")
		}
	}
	channelID := strings.TrimSpace(request.ChannelID)
	channelName := ""
	name := strings.TrimSpace(request.Name)
	overview := strings.TrimSpace(request.Overview)
	startText := strings.TrimSpace(request.StartAt)
	endText := strings.TrimSpace(request.EndAt)
	if program.ID != "" {
		channelID = program.ChannelID
		channelName = program.ChannelName
		name = program.Title
		overview = program.Overview
		startText = program.StartAt
		endText = program.EndAt
	}
	if channelID == "" {
		return RecordingTimer{}, fmt.Errorf("recording timer channel is required")
	}
	if channelName == "" {
		for _, channel := range s.channels {
			if channel.ID == channelID {
				channelName = channel.Name
				break
			}
		}
	}
	if channelName == "" {
		return RecordingTimer{}, fmt.Errorf("recording timer channel not found")
	}
	if name == "" {
		name = channelName
	}
	start, err := time.Parse(time.RFC3339, startText)
	if err != nil {
		return RecordingTimer{}, fmt.Errorf("recording timer startAt must be RFC3339")
	}
	end, err := time.Parse(time.RFC3339, endText)
	if err != nil {
		return RecordingTimer{}, fmt.Errorf("recording timer endAt must be RFC3339")
	}
	start = start.Add(-time.Duration(settings.PrePaddingSeconds) * time.Second)
	end = end.Add(time.Duration(settings.PostPaddingSeconds) * time.Second)
	if !end.After(start) {
		return RecordingTimer{}, fmt.Errorf("recording timer endAt must be after startAt")
	}
	if !end.After(time.Now().UTC()) {
		return RecordingTimer{}, fmt.Errorf("节目已结束，不能预约录制")
	}
	priority := request.Priority
	if priority <= 0 {
		priority = 1
	}
	targetPath := strings.TrimSpace(request.TargetPath)
	if targetPath == "" {
		targetPath = settings.RecordingPath
	}
	now := time.Now().UTC().Format(time.RFC3339)
	id := stableID("timer\x00" + channelID + "\x00" + name + "\x00" + start.UTC().Format(time.RFC3339))
	return RecordingTimer{
		ID:                 id,
		ProgramID:          program.ID,
		ChannelID:          channelID,
		ChannelName:        channelName,
		Name:               name,
		Overview:           overview,
		StartAt:            start.UTC().Format(time.RFC3339),
		EndAt:              end.UTC().Format(time.RFC3339),
		PrePaddingSeconds:  settings.PrePaddingSeconds,
		PostPaddingSeconds: settings.PostPaddingSeconds,
		Priority:           priority,
		Status:             RecordingScheduled,
		TargetPath:         targetPath,
		CreatedAt:          now,
	}, nil
}

func (s *Service) createItemTask(ctx context.Context, taskType, itemID, profile string, update func(*Item) string) (Task, error) {
	if err := ctx.Err(); err != nil {
		return Task{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.items {
		if s.items[i].ID == itemID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Task{}, fmt.Errorf("video item not found: %s", itemID)
	}
	message := update(&s.items[idx])
	now := time.Now().UTC().Format(time.RFC3339)
	task := Task{
		ID:        s.nextJobIDLocked(taskType),
		Type:      taskType,
		ItemID:    itemID,
		Title:     s.items[idx].Title,
		Status:    JobDone,
		Message:   message,
		Progress:  100,
		Profile:   profile,
		CreatedAt: now,
	}
	if taskType == "transcode" && strings.Contains(message, "未检测到") {
		task.Status = JobQueued
		task.Progress = 0
	}
	s.tasks = append([]Task{task}, s.tasks...)
	return task, s.saveLocked()
}

func itemFromFile(root, path string, library Library, container string, now time.Time) (Item, error) {
	info, err := os.Stat(path)
	if err != nil {
		return Item{}, err
	}
	name := info.Name()
	title, year := titleAndYear(name)
	season, episode := seasonEpisode(name)
	kind := "movie"
	if library.Type == LibrarySeries || season > 0 || episode > 0 {
		kind = "episode"
	}
	probe := probeVideo(path)
	codec, resolution, duration := probe.Codec, probe.Resolution, probe.DurationSeconds
	if codec == "" {
		codec = "未知"
	}
	if resolution == "" {
		resolution = resolutionFromName(name)
	}
	videoTracks := probe.VideoTracks
	if len(videoTracks) == 0 {
		videoTracks = []MediaTrack{{
			ID:    "v0",
			Title: strings.TrimSpace(strings.Join([]string{resolution, codec}, " ")),
			Codec: codec,
		}}
	}
	id := stableID(path)
	providerID := doubanProviderIDFromName(name)
	item := Item{
		ID:              stableID(path),
		LibraryID:       library.ID,
		LibraryName:     library.Name,
		Title:           title,
		Kind:            kind,
		Year:            year,
		Season:          season,
		Episode:         episode,
		Container:       container,
		Codec:           codec,
		Resolution:      resolution,
		DurationSeconds: duration,
		SizeBytes:       info.Size(),
		Size:            formatBytes(info.Size()),
		ModifiedAt:      info.ModTime().Format(time.RFC3339),
		DiscoveredAt:    now.Format(time.RFC3339),
		FileName:        name,
		Path:            path,
		PosterURL:       "/api/v1/videos/items/" + id + "/poster",
		PosterPath:      findPosterPath(path),
		StreamURL:       "/api/v1/videos/items/" + id + "/stream",
		Overview:        fmt.Sprintf("来自 %s 的本地媒体，路径相对目录为 %s。", library.Name, relPath(root, path)),
		Genres:          []string{kindLabel(kind)},
		Rating:          "本地",
		ProviderID:      providerID,
		Status:          "已索引",
		VideoTracks:     videoTracks,
		AudioTracks:     probe.AudioTracks,
		SubtitleTracks:  probe.SubtitleTracks,
	}
	if findSubtitlePath(path) != "" {
		item.SubtitleURL = "/api/v1/videos/items/" + item.ID + "/subtitle"
		item.SubtitleTracks = append(item.SubtitleTracks, MediaTrack{
			ID:       "s0",
			Title:    "旁路字幕",
			Language: library.SubtitleLanguage,
			Default:  true,
		})
	}
	if meta, err := readLocalMetadata(path); err == nil {
		applyMetadata(&item, meta, true)
	}
	return item, nil
}

func (s *Service) scrapeItemMetadata(ctx context.Context, item *Item) string {
	sources := make([]string, 0, 3)
	localApplied := false
	if meta, err := readLocalMetadata(item.Path); err == nil {
		applyMetadata(item, meta, true)
		sources = append(sources, sourceName(meta.MetadataSource, "本地元数据"))
		localApplied = true
	}
	if meta, err := fetchConfiguredMetadata(ctx, item); err == nil {
		applyMetadata(item, meta, !localApplied)
		sources = append(sources, sourceName(meta.MetadataSource, "中文数据源"))
	}
	if item.PosterPath == "" && item.PosterRemoteURL != "" {
		if path, err := s.cacheRemotePoster(ctx, item.ID, item.PosterRemoteURL); err == nil {
			item.PosterPath = path
		}
	}
	if len(sources) == 0 {
		if item.Overview == "" {
			item.Overview = fmt.Sprintf("%s 已通过文件名、目录和旁路资源生成基础元数据。", item.Title)
		}
		if len(item.Genres) == 0 {
			item.Genres = []string{kindLabel(item.Kind)}
		}
		sources = append(sources, "文件名与本地资源")
	}
	item.MetadataSource = strings.Join(dedupeStrings(sources), " / ")
	item.ScrapedAt = time.Now().UTC().Format(time.RFC3339)
	item.Status = "已完成刮削"
	return fmt.Sprintf("已更新 %s 的元数据：%s。", item.Title, item.MetadataSource)
}

func readLocalMetadata(videoPath string) (metadataDocument, error) {
	for _, candidate := range metadataCandidates(videoPath) {
		data, err := os.ReadFile(candidate)
		if err != nil {
			continue
		}
		switch strings.ToLower(filepath.Ext(candidate)) {
		case ".json":
			var meta metadataDocument
			if err := json.Unmarshal(data, &meta); err != nil {
				return metadataDocument{}, err
			}
			resolveMetadataPaths(filepath.Dir(candidate), &meta)
			if meta.MetadataSource == "" {
				meta.MetadataSource = "本地元数据"
			}
			return meta, nil
		case ".nfo":
			meta, err := parseNFO(data)
			if err != nil {
				return metadataDocument{}, err
			}
			resolveMetadataPaths(filepath.Dir(candidate), &meta)
			return meta, nil
		}
	}
	return metadataDocument{}, fmt.Errorf("metadata not found")
}

func metadataCandidates(videoPath string) []string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	return []string{
		filepath.Join(dir, base+".metadata.json"),
		filepath.Join(dir, base+".json"),
		filepath.Join(dir, base+".nfo"),
		filepath.Join(dir, "movie.metadata.json"),
		filepath.Join(dir, "movie.json"),
		filepath.Join(dir, "movie.nfo"),
		filepath.Join(dir, "tvshow.nfo"),
	}
}

func parseNFO(data []byte) (metadataDocument, error) {
	var doc nfoDocument
	if err := xml.Unmarshal(data, &doc); err != nil {
		return metadataDocument{}, err
	}
	overview := strings.TrimSpace(doc.Plot)
	if overview == "" {
		overview = strings.TrimSpace(doc.Outline)
	}
	actors := make([]string, 0, len(doc.Actors))
	for _, actor := range doc.Actors {
		if name := strings.TrimSpace(actor.Name); name != "" {
			actors = append(actors, name)
		}
	}
	releaseDate := strings.TrimSpace(doc.ReleaseDate)
	if releaseDate == "" {
		releaseDate = strings.TrimSpace(doc.Premiered)
	}
	title := strings.TrimSpace(doc.Title)
	if title == "" {
		title = strings.TrimSpace(doc.EpisodeTitle)
	}
	return metadataDocument{
		Title:          title,
		OriginalTitle:  strings.TrimSpace(doc.Original),
		SeriesTitle:    strings.TrimSpace(doc.ShowTitle),
		EpisodeTitle:   strings.TrimSpace(doc.EpisodeTitle),
		Year:           strings.TrimSpace(doc.Year),
		Season:         doc.Season,
		Episode:        doc.Episode,
		Overview:       overview,
		Rating:         strings.TrimSpace(doc.Rating),
		ContentRating:  strings.TrimSpace(doc.ContentRating),
		ReleaseDate:    releaseDate,
		Tagline:        strings.TrimSpace(doc.Tagline),
		Genres:         cleanStrings(doc.Genres),
		Tags:           cleanStrings(doc.Tags),
		Directors:      cleanStrings(doc.Directors),
		Writers:        cleanStrings(doc.Writers),
		Actors:         cleanStrings(actors),
		Studios:        cleanStrings(doc.Studios),
		Countries:      cleanStrings(doc.Countries),
		MetadataSource: "NFO",
	}, nil
}

func resolveMetadataPaths(dir string, meta *metadataDocument) {
	meta.Poster = resolveMediaPath(dir, meta.Poster)
	meta.Backdrop = resolveMediaPath(dir, meta.Backdrop)
}

func resolveMediaPath(dir, value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.HasPrefix(value, "http://") || strings.HasPrefix(value, "https://") || filepath.IsAbs(value) {
		return value
	}
	return filepath.Join(dir, value)
}

func applyMetadata(item *Item, meta metadataDocument, overwrite bool) {
	setString := func(target *string, value string) {
		value = strings.TrimSpace(value)
		if value == "" {
			return
		}
		if overwrite || strings.TrimSpace(*target) == "" || *target == "本地" {
			*target = value
		}
	}
	setString(&item.Title, meta.Title)
	setString(&item.OriginalTitle, meta.OriginalTitle)
	setString(&item.SeriesTitle, meta.SeriesTitle)
	setString(&item.EpisodeTitle, meta.EpisodeTitle)
	setString(&item.Year, meta.Year)
	if meta.Season > 0 && (overwrite || item.Season == 0) {
		item.Season = meta.Season
	}
	if meta.Episode > 0 && (overwrite || item.Episode == 0) {
		item.Episode = meta.Episode
	}
	overview := meta.Overview
	if item.Kind == "episode" && strings.TrimSpace(meta.EpisodeOverview) != "" {
		overview = meta.EpisodeOverview
	}
	setString(&item.Overview, overview)
	setString(&item.Rating, meta.Rating)
	setString(&item.ContentRating, meta.ContentRating)
	setString(&item.ReleaseDate, meta.ReleaseDate)
	setString(&item.Tagline, meta.Tagline)
	setString(&item.MetadataSource, meta.MetadataSource)
	setString(&item.ProviderID, meta.ProviderID)
	if meta.Poster != "" {
		if strings.HasPrefix(meta.Poster, "http://") || strings.HasPrefix(meta.Poster, "https://") {
			setString(&item.PosterRemoteURL, meta.Poster)
		} else if info, err := os.Stat(meta.Poster); err == nil && !info.IsDir() {
			item.PosterPath = meta.Poster
		}
	}
	if meta.Backdrop != "" {
		if strings.HasPrefix(meta.Backdrop, "http://") || strings.HasPrefix(meta.Backdrop, "https://") {
			setString(&item.BackdropRemoteURL, meta.Backdrop)
		}
	}
	setStrings := func(target *[]string, values []string) {
		values = cleanStrings(values)
		if len(values) == 0 {
			return
		}
		if overwrite || len(*target) == 0 || (len(*target) == 1 && ((*target)[0] == "电影" || (*target)[0] == "电视剧" || (*target)[0] == "本地媒体")) {
			*target = values
			return
		}
		*target = dedupeStrings(append(*target, values...))
	}
	setStrings(&item.Genres, meta.Genres)
	setStrings(&item.Tags, meta.Tags)
	setStrings(&item.Directors, meta.Directors)
	setStrings(&item.Writers, meta.Writers)
	setStrings(&item.Actors, meta.Actors)
	setStrings(&item.Studios, meta.Studios)
	setStrings(&item.Countries, meta.Countries)
}

func fetchConfiguredMetadata(ctx context.Context, item *Item) (metadataDocument, error) {
	endpoint := strings.TrimSpace(os.Getenv("HIGO_VIDEO_CHINESE_METADATA_URL"))
	if endpoint == "" {
		endpoint = strings.TrimSpace(os.Getenv("HIGO_VIDEO_DOUBAN_ENDPOINT"))
	}
	if endpoint != "" {
		return fetchMetadataEndpoint(ctx, endpoint, item)
	}
	if item.ProviderID != "" {
		return fetchDoubanMetadataByID(ctx, item.ProviderID)
	}
	return fetchSuggestionMetadata(ctx, item)
}

func fetchMetadataEndpoint(ctx context.Context, endpoint string, item *Item) (metadataDocument, error) {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return metadataDocument{}, err
	}
	query := parsed.Query()
	query.Set("q", metadataQueryTitle(item))
	if item.Year != "" {
		query.Set("year", item.Year)
	}
	query.Set("type", item.Kind)
	parsed.RawQuery = query.Encode()
	data, err := fetchJSON(ctx, parsed.String())
	if err != nil {
		return metadataDocument{}, err
	}
	var meta metadataDocument
	if err := json.Unmarshal(data, &meta); err != nil {
		return metadataDocument{}, err
	}
	if meta.MetadataSource == "" {
		meta.MetadataSource = "中文数据源"
	}
	return meta, nil
}

func fetchSuggestionMetadata(ctx context.Context, item *Item) (metadataDocument, error) {
	title := metadataQueryTitle(item)
	if title == "" {
		return metadataDocument{}, fmt.Errorf("empty query")
	}
	suggestURL := strings.TrimSpace(os.Getenv("HIGO_VIDEO_DOUBAN_SUGGEST_URL"))
	if suggestURL == "" {
		suggestURL = "https://movie.douban.com/j/subject_suggest"
	}
	parsed, err := url.Parse(suggestURL)
	if err != nil {
		return metadataDocument{}, err
	}
	query := parsed.Query()
	query.Set("q", title)
	parsed.RawQuery = query.Encode()
	data, err := fetchJSON(ctx, parsed.String())
	if err != nil {
		return metadataDocument{}, err
	}
	var suggestions []suggestionDocument
	if err := json.Unmarshal(data, &suggestions); err != nil {
		return metadataDocument{}, err
	}
	suggestion, ok := bestSuggestion(title, item.Year, suggestions)
	if !ok {
		return metadataDocument{}, fmt.Errorf("metadata suggestion not found")
	}
	meta := metadataDocument{
		Title:          suggestion.Title,
		OriginalTitle:  suggestion.SubTitle,
		Year:           suggestion.Year,
		Poster:         suggestion.Image,
		MetadataSource: "豆瓣",
		ProviderID:     suggestion.ID,
	}
	if suggestion.ID != "" {
		if detail, err := fetchDoubanMetadataByID(ctx, suggestion.ID); err == nil {
			applyMetadataDocument(&meta, detail)
		}
		if meta.Title == "" {
			meta.Title = suggestion.Title
		}
		if meta.OriginalTitle == "" {
			meta.OriginalTitle = suggestion.SubTitle
		}
		if meta.Year == "" {
			meta.Year = suggestion.Year
		}
		if meta.Poster == "" {
			meta.Poster = suggestion.Image
		}
		meta.MetadataSource = "豆瓣"
		meta.ProviderID = suggestion.ID
	}
	return meta, nil
}

func fetchDoubanMetadataByID(ctx context.Context, subjectID string) (metadataDocument, error) {
	if detail, err := fetchDoubanRexxarMetadata(ctx, subjectID); err == nil {
		return detail, nil
	}
	if detail, err := fetchDoubanAbstractMetadata(ctx, subjectID); err == nil {
		return detail, nil
	}
	return fetchDoubanSubjectMetadata(ctx, subjectID)
}

func bestSuggestion(title, year string, suggestions []suggestionDocument) (suggestionDocument, bool) {
	for _, suggestion := range suggestions {
		if suggestion.Title != "" && looseTitleMatch(title, suggestion.Title) {
			return suggestion, true
		}
	}
	if year != "" {
		for _, suggestion := range suggestions {
			if suggestion.Title != "" && suggestion.Year == year {
				return suggestion, true
			}
		}
	}
	for _, suggestion := range suggestions {
		if suggestion.Title != "" {
			return suggestion, true
		}
	}
	return suggestionDocument{}, false
}

func fetchDoubanSubjectMetadata(ctx context.Context, subjectID string) (metadataDocument, error) {
	baseURL := strings.TrimSpace(os.Getenv("HIGO_VIDEO_DOUBAN_SUBJECT_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://movie.douban.com/subject/"
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	target := baseURL + url.PathEscape(subjectID) + "/"
	data, err := fetchHTML(ctx, target)
	if err != nil {
		return metadataDocument{}, err
	}
	meta, err := parseDoubanSubjectHTML(data)
	if err != nil {
		return metadataDocument{}, err
	}
	meta.MetadataSource = "豆瓣"
	meta.ProviderID = subjectID
	return meta, nil
}

func fetchDoubanRexxarMetadata(ctx context.Context, subjectID string) (metadataDocument, error) {
	baseURL := strings.TrimSpace(os.Getenv("HIGO_VIDEO_DOUBAN_REXXAR_BASE_URL"))
	if baseURL == "" {
		baseURL = "https://m.douban.com/rexxar/api/v2/movie/"
	}
	if !strings.HasSuffix(baseURL, "/") {
		baseURL += "/"
	}
	data, err := fetchJSON(ctx, baseURL+url.PathEscape(subjectID))
	if err != nil {
		return metadataDocument{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return metadataDocument{}, err
	}
	meta := metadataDocument{
		Title:          stringFromAny(payload["title"]),
		OriginalTitle:  stringFromAny(payload["original_title"]),
		Year:           stringFromAny(payload["year"]),
		Overview:       stringFromAny(payload["intro"]),
		ReleaseDate:    firstDateFromAny(payload["pubdate"]),
		Genres:         stringsFromAny(payload["genres"]),
		Tags:           stringsFromAny(payload["aka"]),
		Directors:      namesFromAny(payload["directors"]),
		Writers:        namesFromAny(payload["writers"]),
		Actors:         namesFromAny(payload["actors"]),
		Countries:      stringsFromAny(payload["countries"]),
		MetadataSource: "豆瓣",
		ProviderID:     subjectID,
	}
	if rating, ok := payload["rating"].(map[string]any); ok {
		meta.Rating = stringFromAny(rating["value"])
	}
	if pic, ok := payload["pic"].(map[string]any); ok {
		meta.Poster = stringFromAny(pic["normal"])
		if meta.Poster == "" {
			meta.Poster = stringFromAny(pic["large"])
		}
	}
	if meta.Poster == "" {
		meta.Poster = stringFromAny(payload["cover_url"])
	}
	if meta.Title == "" && meta.Overview == "" && meta.Poster == "" {
		return metadataDocument{}, fmt.Errorf("douban rexxar metadata not found")
	}
	return meta, nil
}

func fetchDoubanAbstractMetadata(ctx context.Context, subjectID string) (metadataDocument, error) {
	abstractURL := strings.TrimSpace(os.Getenv("HIGO_VIDEO_DOUBAN_ABSTRACT_URL"))
	if abstractURL == "" {
		abstractURL = "https://movie.douban.com/j/subject_abstract"
	}
	parsed, err := url.Parse(abstractURL)
	if err != nil {
		return metadataDocument{}, err
	}
	query := parsed.Query()
	query.Set("subject_id", subjectID)
	parsed.RawQuery = query.Encode()
	data, err := fetchJSON(ctx, parsed.String())
	if err != nil {
		return metadataDocument{}, err
	}
	var payload map[string]any
	if err := json.Unmarshal(data, &payload); err != nil {
		return metadataDocument{}, err
	}
	subject, ok := payload["subject"].(map[string]any)
	if !ok {
		return metadataDocument{}, fmt.Errorf("douban abstract subject not found")
	}
	meta := metadataDocument{
		Title:          cleanAbstractTitle(stringFromAny(subject["title"])),
		Year:           stringFromAny(subject["release_year"]),
		Rating:         stringFromAny(subject["rate"]),
		Genres:         stringsFromAny(subject["types"]),
		Countries:      cleanStrings([]string{stringFromAny(subject["region"])}),
		Directors:      stringsFromAny(subject["directors"]),
		Actors:         stringsFromAny(subject["actors"]),
		MetadataSource: "豆瓣",
		ProviderID:     subjectID,
	}
	if meta.Title == "" && meta.Rating == "" && len(meta.Actors) == 0 {
		return metadataDocument{}, fmt.Errorf("douban abstract metadata not found")
	}
	return meta, nil
}

func parseDoubanSubjectHTML(data []byte) (metadataDocument, error) {
	text := string(data)
	scriptPattern := regexp.MustCompile(`(?is)<script[^>]+type=["']application/ld\+json["'][^>]*>(.*?)</script>`)
	matches := scriptPattern.FindAllStringSubmatch(text, -1)
	for _, match := range matches {
		if len(match) < 2 {
			continue
		}
		raw := strings.TrimSpace(html.UnescapeString(match[1]))
		if raw == "" {
			continue
		}
		var payload map[string]any
		if err := json.Unmarshal([]byte(raw), &payload); err != nil {
			continue
		}
		meta := metadataDocument{
			Title:          stringFromAny(payload["name"]),
			OriginalTitle:  stringFromAny(payload["alternateName"]),
			ReleaseDate:    stringFromAny(payload["datePublished"]),
			Overview:       stringFromAny(payload["description"]),
			Poster:         stringFromAny(payload["image"]),
			Genres:         stringsFromAny(payload["genre"]),
			Directors:      namesFromAny(payload["director"]),
			Writers:        namesFromAny(payload["author"]),
			Actors:         namesFromAny(payload["actor"]),
			Countries:      stringsFromAny(payload["countryOfOrigin"]),
			MetadataSource: "豆瓣",
		}
		if meta.Year == "" && len(meta.ReleaseDate) >= 4 {
			meta.Year = meta.ReleaseDate[:4]
		}
		if rating, ok := payload["aggregateRating"].(map[string]any); ok {
			meta.Rating = stringFromAny(rating["ratingValue"])
		}
		if meta.Title != "" || meta.Overview != "" || meta.Poster != "" {
			return meta, nil
		}
	}
	return metadataDocument{}, fmt.Errorf("douban subject metadata not found")
}

func applyMetadataDocument(target *metadataDocument, source metadataDocument) {
	setString := func(field *string, value string) {
		value = strings.TrimSpace(value)
		if value != "" {
			*field = value
		}
	}
	setString(&target.Title, source.Title)
	setString(&target.OriginalTitle, source.OriginalTitle)
	setString(&target.SeriesTitle, source.SeriesTitle)
	setString(&target.EpisodeTitle, source.EpisodeTitle)
	setString(&target.Year, source.Year)
	setString(&target.Overview, source.Overview)
	setString(&target.EpisodeOverview, source.EpisodeOverview)
	setString(&target.Rating, source.Rating)
	setString(&target.ContentRating, source.ContentRating)
	setString(&target.ReleaseDate, source.ReleaseDate)
	setString(&target.Tagline, source.Tagline)
	setString(&target.Poster, source.Poster)
	setString(&target.Backdrop, source.Backdrop)
	setString(&target.MetadataSource, source.MetadataSource)
	setString(&target.ProviderID, source.ProviderID)
	if source.Season > 0 {
		target.Season = source.Season
	}
	if source.Episode > 0 {
		target.Episode = source.Episode
	}
	setStrings := func(field *[]string, values []string) {
		if cleaned := cleanStrings(values); len(cleaned) > 0 {
			*field = cleaned
		}
	}
	setStrings(&target.Genres, source.Genres)
	setStrings(&target.Tags, source.Tags)
	setStrings(&target.Directors, source.Directors)
	setStrings(&target.Writers, source.Writers)
	setStrings(&target.Actors, source.Actors)
	setStrings(&target.Studios, source.Studios)
	setStrings(&target.Countries, source.Countries)
}

func stringFromAny(value any) string {
	switch typed := value.(type) {
	case string:
		return strings.TrimSpace(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case json.Number:
		return typed.String()
	default:
		return ""
	}
}

func firstDateFromAny(value any) string {
	values := stringsFromAny(value)
	if len(values) == 0 {
		return ""
	}
	date := strings.TrimSpace(values[0])
	if idx := strings.Index(date, "("); idx > 0 {
		date = strings.TrimSpace(date[:idx])
	}
	if idx := strings.Index(date, "（"); idx > 0 {
		date = strings.TrimSpace(date[:idx])
	}
	return date
}

func cleanAbstractTitle(value string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return ""
	}
	value = strings.ReplaceAll(value, "\u200e", "")
	if idx := strings.Index(value, " ("); idx > 0 {
		value = strings.TrimSpace(value[:idx])
	}
	if idx := strings.Index(value, "（"); idx > 0 {
		value = strings.TrimSpace(value[:idx])
	}
	return value
}

func stringsFromAny(value any) []string {
	switch typed := value.(type) {
	case string:
		return cleanStrings([]string{typed})
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			out = append(out, stringFromAny(item))
		}
		return cleanStrings(out)
	case map[string]any:
		return cleanStrings([]string{stringFromAny(typed["name"])})
	default:
		return nil
	}
}

func namesFromAny(value any) []string {
	switch typed := value.(type) {
	case []any:
		out := make([]string, 0, len(typed))
		for _, item := range typed {
			if object, ok := item.(map[string]any); ok {
				out = append(out, stringFromAny(object["name"]))
				continue
			}
			out = append(out, stringFromAny(item))
		}
		return cleanStrings(out)
	case map[string]any:
		return cleanStrings([]string{stringFromAny(typed["name"])})
	default:
		return stringsFromAny(value)
	}
}

func fetchHTML(ctx context.Context, target string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 HiGoOS/1.0")
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	req.Header.Set("Referer", "https://movie.douban.com/")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata page request failed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
}

func suggestionMetadataDocument(suggestion suggestionDocument) metadataDocument {
	return metadataDocument{
		Title:          suggestion.Title,
		Year:           suggestion.Year,
		Poster:         suggestion.Image,
		MetadataSource: "豆瓣",
		ProviderID:     suggestion.ID,
	}
}

func metadataQueryTitle(item *Item) string {
	if item.Kind == "episode" && item.SeriesTitle != "" {
		return item.SeriesTitle
	}
	if item.OriginalTitle != "" {
		return item.OriginalTitle
	}
	return item.Title
}

func looseTitleMatch(a, b string) bool {
	normalize := func(value string) string {
		value = strings.ToLower(value)
		value = strings.Join(strings.Fields(value), "")
		value = strings.NewReplacer("：", ":", " ", "", ".", "", "-", "", "_", "").Replace(value)
		return value
	}
	aa, bb := normalize(a), normalize(b)
	return aa == bb || strings.Contains(aa, bb) || strings.Contains(bb, aa)
}

func fetchJSON(ctx context.Context, target string) ([]byte, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 HiGoOS/1.0")
	req.Header.Set("Accept", "application/json,text/plain,*/*")
	if strings.Contains(target, "douban.com") {
		req.Header.Set("Referer", "https://m.douban.com/movie/")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("metadata request failed: %s", resp.Status)
	}
	return io.ReadAll(io.LimitReader(resp.Body, 2*1024*1024))
}

func fetchRemoteImage(ctx context.Context, target string) ([]byte, string, error) {
	data, contentType, err := fetchRemoteImageOnce(ctx, target, "")
	if err != nil {
		return nil, "", err
	}
	if isImageResponse(contentType, data) {
		return data, contentType, nil
	}
	if cookie, ok := botChallengeCookie(string(data)); ok {
		data, contentType, err = fetchRemoteImageOnce(ctx, target, cookie)
		if err != nil {
			return nil, "", err
		}
		if isImageResponse(contentType, data) {
			return data, contentType, nil
		}
	}
	return nil, "", fmt.Errorf("poster response body is not an image")
}

func fetchRemoteImageOnce(ctx context.Context, target, cookie string) ([]byte, string, error) {
	reqCtx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(reqCtx, http.MethodGet, target, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 HiGoOS/1.0")
	req.Header.Set("Accept", "image/avif,image/webp,image/*,*/*")
	if strings.Contains(target, "doubanio.com") || strings.Contains(target, "douban.com") {
		req.Header.Set("Referer", "https://movie.douban.com/")
	}
	if cookie != "" {
		req.Header.Set("Cookie", cookie)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, "", fmt.Errorf("poster request failed: %s", resp.Status)
	}
	contentType := resp.Header.Get("Content-Type")
	if contentType == "" {
		contentType = "image/jpeg"
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, 5*1024*1024))
	if err != nil {
		return nil, "", err
	}
	if len(data) == 0 {
		return nil, "", fmt.Errorf("empty image")
	}
	return data, contentType, nil
}

func isImageResponse(contentType string, data []byte) bool {
	return strings.HasPrefix(strings.ToLower(contentType), "image/") && looksLikeImage(data)
}

func botChallengeCookie(body string) (string, bool) {
	if !strings.Contains(body, "__tst_status") || !strings.Contains(body, "EO_Bot_Ssid") {
		return "", false
	}
	statusMatches := regexp.MustCompile(`(?:WTKkN|bOYDu|wyeCN):(\d+)`).FindAllStringSubmatch(body, -1)
	if len(statusMatches) < 3 {
		return "", false
	}
	status := 0
	for _, match := range statusMatches[:3] {
		part, err := strconv.Atoi(match[1])
		if err != nil {
			return "", false
		}
		status += part
	}
	ssidMatch := regexp.MustCompile(`\(t,(\d+)\)`).FindStringSubmatch(body)
	if len(ssidMatch) < 2 {
		return "", false
	}
	return fmt.Sprintf("__tst_status=%d#; EO_Bot_Ssid=%s", status, ssidMatch[1]), true
}

func looksLikeImage(data []byte) bool {
	if len(data) >= 3 && data[0] == 0xff && data[1] == 0xd8 && data[2] == 0xff {
		return true
	}
	if len(data) >= 8 && string(data[:8]) == "\x89PNG\r\n\x1a\n" {
		return true
	}
	if len(data) >= 12 && string(data[:4]) == "RIFF" && string(data[8:12]) == "WEBP" {
		return true
	}
	if len(data) >= 6 && (string(data[:6]) == "GIF87a" || string(data[:6]) == "GIF89a") {
		return true
	}
	return false
}

func (s *Service) cacheRemotePoster(ctx context.Context, itemID, target string) (string, error) {
	if s.statePath == "" {
		return "", fmt.Errorf("video state path is not configured")
	}
	data, contentType, err := fetchRemoteImage(ctx, target)
	if err != nil {
		return "", err
	}
	ext := imageExtension(contentType, target)
	dir := filepath.Join(filepath.Dir(s.statePath), "video-posters")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, itemID+ext)
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return "", err
	}
	if err := os.Rename(tmp, path); err != nil {
		_ = os.Remove(tmp)
		return "", err
	}
	return path, nil
}

func imageExtension(contentType, target string) string {
	lowerType := strings.ToLower(contentType)
	switch {
	case strings.Contains(lowerType, "png"):
		return ".png"
	case strings.Contains(lowerType, "webp"):
		return ".webp"
	case strings.Contains(lowerType, "jpeg"), strings.Contains(lowerType, "jpg"):
		return ".jpg"
	}
	ext := strings.ToLower(filepath.Ext(target))
	switch ext {
	case ".png", ".webp", ".jpg", ".jpeg":
		if ext == ".jpeg" {
			return ".jpg"
		}
		return ext
	default:
		return ".jpg"
	}
}

func sourceName(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" {
		return fallback
	}
	return value
}

func defaultDVRSettings() DVRSettings {
	return DVRSettings{
		RecordingPath:       "/srv/higoos/recordings",
		PrePaddingSeconds:   60,
		PostPaddingSeconds:  180,
		MaxConcurrentRecord: 1,
		SaveNFO:             true,
		SaveImages:          true,
	}
}

func normalizeDVRSettings(settings DVRSettings) DVRSettings {
	defaults := defaultDVRSettings()
	settings.RecordingPath = strings.TrimSpace(settings.RecordingPath)
	settings.MovieRecordingPath = strings.TrimSpace(settings.MovieRecordingPath)
	settings.SeriesRecordingPath = strings.TrimSpace(settings.SeriesRecordingPath)
	settings.PostProcessCommand = strings.TrimSpace(settings.PostProcessCommand)
	if settings.RecordingPath == "" {
		settings.RecordingPath = defaults.RecordingPath
	}
	if settings.PrePaddingSeconds < 0 {
		settings.PrePaddingSeconds = 0
	}
	if settings.PostPaddingSeconds < 0 {
		settings.PostPaddingSeconds = 0
	}
	if settings.MaxConcurrentRecord <= 0 {
		settings.MaxConcurrentRecord = defaults.MaxConcurrentRecord
	}
	return settings
}

func recordingOutputPath(timer RecordingTimer, settings DVRSettings) string {
	dir := strings.TrimSpace(timer.TargetPath)
	if dir == "" {
		dir = settings.RecordingPath
	}
	start, err := time.Parse(time.RFC3339, timer.StartAt)
	startText := timer.CreatedAt
	if err == nil {
		startText = start.In(time.Local).Format("20060102-1504")
	}
	name := safeFileName(strings.Join([]string{startText, timer.ChannelName, timer.Name}, "-"))
	if name == "" {
		name = safeFileName(timer.ID)
	}
	return filepath.Join(dir, name+".mp4")
}

func safeFileName(value string) string {
	value = cleanChannelName(value)
	replacer := strings.NewReplacer(
		"/", "-", "\\", "-", ":", "-", "*", "-", "?", "", `"`, "", "<", "(", ">", ")", "|", "-",
		"\n", " ", "\r", " ", "\t", " ",
	)
	value = replacer.Replace(value)
	value = strings.Join(strings.Fields(value), " ")
	value = strings.Trim(value, " .-")
	runes := []rune(value)
	if len(runes) > 120 {
		value = string(runes[:120])
	}
	return value
}

func (s *Service) writeRecordingSidecars(timer RecordingTimer, path string) {
	settings := normalizeDVRSettings(s.dvrSettings)
	if !settings.SaveNFO {
		return
	}
	nfoPath := strings.TrimSuffix(path, filepath.Ext(path)) + ".nfo"
	type recordingNFO struct {
		XMLName     xml.Name `xml:"movie"`
		Title       string   `xml:"title"`
		Plot        string   `xml:"plot"`
		Channel     string   `xml:"studio"`
		Premiered   string   `xml:"premiered"`
		ReleaseDate string   `xml:"releasedate"`
	}
	start, _ := time.Parse(time.RFC3339, timer.StartAt)
	date := start.Format("2006-01-02")
	data, err := xml.MarshalIndent(recordingNFO{
		Title:       timer.Name,
		Plot:        timer.Overview,
		Channel:     timer.ChannelName,
		Premiered:   date,
		ReleaseDate: date,
	}, "", "  ")
	if err != nil {
		return
	}
	_ = os.WriteFile(nfoPath, append([]byte(xml.Header), data...), 0o644)
}

func normalizeLibraries(input []Library) ([]Library, error) {
	result := make([]Library, 0, len(input))
	for _, library := range input {
		name := strings.TrimSpace(library.Name)
		if name == "" {
			return nil, fmt.Errorf("library name is required")
		}
		paths := make([]string, 0, len(library.Paths))
		for _, path := range library.Paths {
			clean := filepath.Clean(strings.TrimSpace(path))
			if clean != "." && clean != "" {
				paths = append(paths, clean)
			}
		}
		if len(paths) == 0 {
			return nil, fmt.Errorf("library path is required")
		}
		libraryType := library.Type
		if libraryType == "" {
			libraryType = LibraryMixed
		}
		lang := strings.TrimSpace(library.MetadataLanguage)
		if lang == "" {
			lang = "zh-CN"
		}
		subLang := strings.TrimSpace(library.SubtitleLanguage)
		if subLang == "" {
			subLang = "zh-CN"
		}
		id := strings.TrimSpace(library.ID)
		if id == "" {
			id = stableID(name + "\x00" + strings.Join(paths, "\x00"))
		}
		result = append(result, Library{
			ID:                id,
			Name:              name,
			Type:              libraryType,
			Paths:             paths,
			MetadataLanguage:  lang,
			AllowAdultContent: library.AllowAdultContent,
			AutoSubtitles:     library.AutoSubtitles,
			SubtitleLanguage:  subLang,
			Status:            "已配置",
		})
	}
	return result, nil
}

func parseM3U(ctx context.Context, sourceURL string) ([]LiveChannel, error) {
	var reader io.ReadCloser
	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return nil, err
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return nil, fmt.Errorf("live source returned %s", resp.Status)
		}
		reader = resp.Body
	} else {
		file, err := os.Open(filepath.Clean(sourceURL))
		if err != nil {
			return nil, err
		}
		reader = file
	}
	defer reader.Close()

	channels := make([]LiveChannel, 0)
	scanner := bufio.NewScanner(reader)
	var pending LiveChannel
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, "#EXTINF") {
			pending = LiveChannel{
				Name:      cleanChannelName(extinfName(line)),
				GuideID:   strings.TrimSpace(attr(line, "tvg-id")),
				GuideName: cleanChannelName(attr(line, "tvg-name")),
				Group:     strings.TrimSpace(attr(line, "group-title")),
				Logo:      strings.TrimSpace(attr(line, "tvg-logo")),
			}
			continue
		}
		if strings.HasPrefix(line, "#") {
			continue
		}
		if pending.Name == "" {
			pending.Name = fmt.Sprintf("频道 %d", len(channels)+1)
		}
		pending.URL = line
		channels = append(channels, pending)
		pending = LiveChannel{}
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	if len(channels) == 0 {
		return nil, fmt.Errorf("live source has no channels")
	}
	return channels, nil
}

type xmltvDocument struct {
	Channels []xmltvChannel   `xml:"channel"`
	Programs []xmltvProgramme `xml:"programme"`
}

type xmltvChannel struct {
	ID           string   `xml:"id,attr"`
	DisplayNames []string `xml:"display-name"`
}

type xmltvProgramme struct {
	Start      string   `xml:"start,attr"`
	Stop       string   `xml:"stop,attr"`
	Channel    string   `xml:"channel,attr"`
	Titles     []string `xml:"title"`
	Descs      []string `xml:"desc"`
	Categories []string `xml:"category"`
}

func parseXMLTV(ctx context.Context, sourceURL string, userAgent string, channels []LiveChannel) ([]LiveProgram, error) {
	reader, err := openTextSource(ctx, sourceURL, userAgent)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	var document xmltvDocument
	if err := xml.NewDecoder(reader).Decode(&document); err != nil {
		return nil, err
	}
	channelNames := map[string]string{}
	for _, channel := range document.Channels {
		if len(channel.DisplayNames) > 0 {
			channelNames[channel.ID] = cleanChannelName(channel.DisplayNames[0])
		}
	}
	channelByGuide := map[string]LiveChannel{}
	channelByName := map[string]LiveChannel{}
	for _, channel := range channels {
		if channel.GuideID != "" {
			channelByGuide[normalizeChannelKey(channel.GuideID)] = channel
		}
		if channel.GuideName != "" {
			channelByName[normalizeChannelKey(channel.GuideName)] = channel
		}
		channelByName[normalizeChannelKey(channel.Name)] = channel
	}
	programs := make([]LiveProgram, 0, len(document.Programs))
	for _, entry := range document.Programs {
		start, err := parseXMLTVTime(entry.Start)
		if err != nil {
			continue
		}
		end, err := parseXMLTVTime(entry.Stop)
		if err != nil || !end.After(start) {
			continue
		}
		guideChannelName := strings.TrimSpace(channelNames[entry.Channel])
		channel, ok := channelByGuide[normalizeChannelKey(entry.Channel)]
		if !ok && guideChannelName != "" {
			channel, ok = channelByName[normalizeChannelKey(guideChannelName)]
		}
		if !ok {
			channel, ok = channelByName[normalizeChannelKey(entry.Channel)]
		}
		if !ok {
			continue
		}
		title := firstNonEmpty(entry.Titles)
		if title == "" {
			title = "未命名节目"
		}
		startText := start.UTC().Format(time.RFC3339)
		endText := end.UTC().Format(time.RFC3339)
		programs = append(programs, LiveProgram{
			ID:          stableID("program\x00" + channel.ID + "\x00" + title + "\x00" + startText),
			ChannelID:   channel.ID,
			ChannelName: channel.Name,
			Title:       title,
			Overview:    firstNonEmpty(entry.Descs),
			Categories:  trimNonEmpty(entry.Categories),
			StartAt:     startText,
			EndAt:       endText,
			Duration:    int(end.Sub(start).Seconds()),
		})
	}
	if len(programs) == 0 {
		return nil, fmt.Errorf("节目指南没有匹配到直播频道，请确认 XMLTV 的 channel/display-name 与直播源的 tvg-id、tvg-name 或频道名一致")
	}
	return programs, nil
}

func openTextSource(ctx context.Context, sourceURL string, userAgent string) (io.ReadCloser, error) {
	if strings.HasPrefix(sourceURL, "http://") || strings.HasPrefix(sourceURL, "https://") {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
		if err != nil {
			return nil, err
		}
		if userAgent != "" {
			req.Header.Set("User-Agent", userAgent)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			return nil, err
		}
		if resp.StatusCode >= 400 {
			resp.Body.Close()
			return nil, fmt.Errorf("source returned %s", resp.Status)
		}
		return resp.Body, nil
	}
	return os.Open(filepath.Clean(sourceURL))
}

func parseXMLTVTime(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("empty xmltv time")
	}
	layouts := []string{
		"20060102150405 -0700",
		"20060102150405",
		"200601021504 -0700",
		"200601021504",
	}
	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, value); err == nil {
			return parsed, nil
		}
	}
	return time.Time{}, fmt.Errorf("invalid xmltv time %q", value)
}

func parseOptionalRFC3339(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, nil
	}
	return time.Parse(time.RFC3339, value)
}

func firstNonEmpty(values []string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func cleanChannelName(value string) string {
	value = strings.TrimSpace(value)
	value = strings.TrimPrefix(value, "\ufeff")
	value = strings.TrimSpace(value)
	return value
}

func normalizeChannelKey(value string) string {
	value = strings.ToLower(cleanChannelName(value))
	replacer := strings.NewReplacer(" ", "", "\t", "", "\n", "", "\r", "", "-", "", "_", "", "－", "", "　", "")
	return replacer.Replace(value)
}

func trimNonEmpty(values []string) []string {
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			out = append(out, value)
		}
	}
	return out
}

func (s *Service) refreshCountsLocked() {
	counts := map[string]int{}
	for _, item := range s.items {
		counts[item.LibraryID]++
	}
	for idx := range s.settings.Libraries {
		s.settings.Libraries[idx].Count = counts[s.settings.Libraries[idx].ID]
	}
	s.settings.ItemCount = len(s.items)
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Settings:     s.settings,
		Items:        s.items,
		Tasks:        s.tasks,
		LiveSources:  s.liveSources,
		Channels:     s.channels,
		GuideSources: s.guideSources,
		Programs:     s.programs,
		DVRSettings:  s.dvrSettings,
		Timers:       s.timers,
		Recordings:   s.recordings,
		NextTaskSeq:  s.nextTaskSeq,
	})
}

func (s *Service) nextJobIDLocked(prefix string) string {
	id := fmt.Sprintf("%s-%d", prefix, s.nextTaskSeq)
	s.nextTaskSeq++
	return id
}

func stableID(input string) string {
	sum := sha1.Sum([]byte(input))
	return hex.EncodeToString(sum[:])[:16]
}

func titleAndYear(name string) (string, string) {
	base := strings.TrimSuffix(name, filepath.Ext(name))
	year := ""
	if match := yearPattern.FindStringSubmatch(base); len(match) > 1 {
		year = match[1]
	}
	base = doubanIDPattern.ReplaceAllString(base, " ")
	clean := episodePattern.ReplaceAllString(base, "")
	clean = yearPattern.ReplaceAllString(clean, " ")
	replacer := strings.NewReplacer(".", " ", "_", " ", "-", " ", "[", " ", "]", " ", "(", " ", ")", " ")
	clean = strings.Join(strings.Fields(replacer.Replace(clean)), " ")
	clean = qualityPattern.ReplaceAllString(clean, " ")
	clean = releasePattern.ReplaceAllString(clean, " ")
	clean = strings.Join(strings.Fields(clean), " ")
	if clean == "" {
		clean = base
	}
	return clean, year
}

func doubanProviderIDFromName(name string) string {
	match := doubanIDPattern.FindStringSubmatch(name)
	if len(match) < 2 {
		return ""
	}
	return strings.TrimSpace(match[1])
}

func seasonEpisode(name string) (int, int) {
	match := episodePattern.FindStringSubmatch(name)
	if len(match) < 3 {
		return 0, 0
	}
	season, _ := strconv.Atoi(match[1])
	episode, _ := strconv.Atoi(match[2])
	return season, episode
}

func probeVideo(path string) probeResult {
	if _, err := exec.LookPath("ffprobe"); err != nil {
		return probeResult{}
	}
	cmd := exec.Command("ffprobe", "-v", "error", "-show_entries", "stream=index,codec_type,codec_name,width,height,channels:stream_tags=language,title:stream_disposition=default", "-show_entries", "format=duration", "-of", "json", path)
	output, err := cmd.Output()
	if err != nil {
		return probeResult{}
	}
	return parseProbeResult(output)
}

func parseProbeResult(data []byte) probeResult {
	var doc ffprobeDocument
	if err := json.Unmarshal(data, &doc); err != nil {
		return probeResult{}
	}
	result := probeResult{}
	if seconds, err := strconv.ParseFloat(doc.Format.Duration, 64); err == nil {
		result.DurationSeconds = int(seconds + 0.5)
	}
	for _, stream := range doc.Streams {
		codec := strings.ToUpper(strings.TrimSpace(stream.CodecName))
		track := MediaTrack{
			ID:       streamTrackID(stream),
			Title:    streamTitle(stream, codec),
			Language: strings.TrimSpace(stream.Tags["language"]),
			Codec:    codec,
			Default:  stream.Disposition["default"] == 1,
		}
		switch stream.CodecType {
		case "video":
			if stream.Width > 0 && stream.Height > 0 {
				resolution := fmt.Sprintf("%dx%d", stream.Width, stream.Height)
				if result.Resolution == "" {
					result.Resolution = resolution
				}
				if !strings.Contains(track.Title, resolution) {
					track.Title = strings.TrimSpace(resolution + " " + track.Title)
				}
			}
			if result.Codec == "" {
				result.Codec = codec
			}
			result.VideoTracks = append(result.VideoTracks, track)
		case "audio":
			track.Channels = channelLabel(stream.Channels)
			result.AudioTracks = append(result.AudioTracks, track)
		case "subtitle":
			result.SubtitleTracks = append(result.SubtitleTracks, track)
		}
	}
	return result
}

func streamTrackID(stream ffprobeStream) string {
	prefix := "t"
	switch stream.CodecType {
	case "video":
		prefix = "v"
	case "audio":
		prefix = "a"
	case "subtitle":
		prefix = "s"
	}
	return fmt.Sprintf("%s%d", prefix, stream.Index)
}

func streamTitle(stream ffprobeStream, codec string) string {
	if title := strings.TrimSpace(stream.Tags["title"]); title != "" {
		return title
	}
	if codec != "" {
		return codec
	}
	return stream.CodecType
}

func channelLabel(channels int) string {
	switch channels {
	case 0:
		return ""
	case 1:
		return "Mono"
	case 2:
		return "Stereo"
	case 6:
		return "5.1"
	case 8:
		return "7.1"
	default:
		return fmt.Sprintf("%d ch", channels)
	}
}

func resolutionFromName(name string) string {
	lower := strings.ToLower(name)
	switch {
	case strings.Contains(lower, "2160") || strings.Contains(lower, "4k"):
		return "4K"
	case strings.Contains(lower, "1080"):
		return "1080p"
	case strings.Contains(lower, "720"):
		return "720p"
	default:
		return "未知"
	}
}

func findPosterPath(videoPath string) string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	candidates := []string{
		filepath.Join(dir, base+".jpg"),
		filepath.Join(dir, base+".png"),
		filepath.Join(dir, "poster.jpg"),
		filepath.Join(dir, "poster.png"),
		filepath.Join(dir, "folder.jpg"),
		filepath.Join(dir, "cover.jpg"),
	}
	for _, candidate := range candidates {
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func findSubtitlePath(videoPath string) string {
	dir := filepath.Dir(videoPath)
	base := strings.TrimSuffix(filepath.Base(videoPath), filepath.Ext(videoPath))
	for _, ext := range []string{".vtt", ".srt", ".zh.vtt", ".zh.srt"} {
		candidate := filepath.Join(dir, base+ext)
		if info, err := os.Stat(candidate); err == nil && !info.IsDir() {
			return candidate
		}
	}
	return ""
}

func placeholderPoster(title string) []byte {
	escaped := strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;").Replace(title)
	return []byte(fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="480" height="720" viewBox="0 0 480 720"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop stop-color="#1d4ed8"/><stop offset="1" stop-color="#020617"/></linearGradient></defs><rect width="480" height="720" rx="34" fill="url(#g)"/><circle cx="240" cy="292" r="82" fill="rgba(255,255,255,.14)"/><path d="M220 248v88l74-44z" fill="white"/><text x="240" y="470" font-size="36" font-family="Arial, sans-serif" font-weight="700" fill="white" text-anchor="middle">%s</text></svg>`, escaped))
}

func contentTypeForPath(path string) string {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".webp":
		return "image/webp"
	case ".vtt":
		return "text/vtt; charset=utf-8"
	case ".srt":
		return "application/x-subrip; charset=utf-8"
	case ".mp4", ".m4v":
		return "video/mp4"
	case ".webm":
		return "video/webm"
	default:
		return "application/octet-stream"
	}
}

func modTime(path string) time.Time {
	info, err := os.Stat(path)
	if err != nil {
		return time.Now()
	}
	return info.ModTime()
}

func relPath(root, path string) string {
	rel, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.Base(path)
	}
	return rel
}

func kindLabel(kind string) string {
	if kind == "episode" {
		return "电视剧"
	}
	return "电影"
}

func cleanStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		key := strings.ToLower(value)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, value)
	}
	return out
}

func dedupeStrings(values []string) []string {
	return cleanStrings(values)
}

func formatBytes(size int64) string {
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d B", size)
	}
	div, exp := int64(unit), 0
	for n := size / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(size)/float64(div), "KMGTPE"[exp])
}

func extinfName(line string) string {
	idx := strings.LastIndex(line, ",")
	if idx < 0 || idx == len(line)-1 {
		return "未命名频道"
	}
	return strings.TrimSpace(line[idx+1:])
}

func attr(line, key string) string {
	marker := key + `="`
	idx := strings.Index(line, marker)
	if idx < 0 {
		return ""
	}
	rest := line[idx+len(marker):]
	end := strings.Index(rest, `"`)
	if end < 0 {
		return ""
	}
	return rest[:end]
}

func cloneSettings(settings LibrarySettings) LibrarySettings {
	settings.Libraries = cloneLibraries(settings.Libraries)
	return settings
}

func cloneLibraries(libraries []Library) []Library {
	out := make([]Library, len(libraries))
	for i, library := range libraries {
		out[i] = library
		out[i].Paths = append([]string(nil), library.Paths...)
	}
	return out
}

func cloneItems(items []Item) []Item {
	out := make([]Item, len(items))
	for i, item := range items {
		out[i] = cloneItem(item)
	}
	return out
}

func cloneItem(item Item) Item {
	item.Genres = append([]string(nil), item.Genres...)
	item.Tags = append([]string(nil), item.Tags...)
	item.Directors = append([]string(nil), item.Directors...)
	item.Writers = append([]string(nil), item.Writers...)
	item.Actors = append([]string(nil), item.Actors...)
	item.Studios = append([]string(nil), item.Studios...)
	item.Countries = append([]string(nil), item.Countries...)
	item.VideoTracks = append([]MediaTrack(nil), item.VideoTracks...)
	item.AudioTracks = append([]MediaTrack(nil), item.AudioTracks...)
	item.SubtitleTracks = append([]MediaTrack(nil), item.SubtitleTracks...)
	return item
}

func cloneLiveSources(sources []LiveSource) []LiveSource {
	return append([]LiveSource(nil), sources...)
}

func cloneLiveChannels(channels []LiveChannel) []LiveChannel {
	out := make([]LiveChannel, 0, len(channels))
	for _, channel := range channels {
		out = append(out, sanitizeLiveChannel(channel))
	}
	return out
}

func sanitizeLiveChannel(channel LiveChannel) LiveChannel {
	channel.GuideID = cleanChannelName(channel.GuideID)
	channel.GuideName = cleanChannelName(channel.GuideName)
	channel.Name = cleanChannelName(channel.Name)
	channel.Group = cleanChannelName(channel.Group)
	return channel
}

func cloneGuideSources(sources []LiveGuideSource) []LiveGuideSource {
	return append([]LiveGuideSource(nil), sources...)
}

func clonePrograms(programs []LiveProgram) []LiveProgram {
	out := make([]LiveProgram, 0, len(programs))
	for _, program := range programs {
		out = append(out, cloneProgram(program))
	}
	return out
}

func cloneProgram(program LiveProgram) LiveProgram {
	program.ChannelName = cleanChannelName(program.ChannelName)
	program.Categories = append([]string(nil), program.Categories...)
	return program
}

func cloneRecordingTimers(timers []RecordingTimer) []RecordingTimer {
	out := make([]RecordingTimer, 0, len(timers))
	return append(out, timers...)
}

func cloneRecordings(recordings []RecordingItem) []RecordingItem {
	out := make([]RecordingItem, 0, len(recordings))
	return append(out, recordings...)
}

func ServeBytes(w http.ResponseWriter, r *http.Request, name string, data []byte, contentType string, modTime time.Time) {
	reader := bytes.NewReader(data)
	w.Header().Set("Content-Type", contentType)
	http.ServeContent(w, r, name, modTime, reader)
}
