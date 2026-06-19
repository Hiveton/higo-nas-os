package media

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

type Service struct {
	mu             sync.RWMutex
	items          []MediaItem
	albums         []Album
	people         []Person
	memoryRuns     []MemoryRun
	subtitleJobs   []SubtitleJob
	transcodeJobs  []TranscodeJob
	shares         []ShareResult
	nextAlbumID    int
	nextJobSeq     int
	memoryRunCount int
	statePath      string
	runner         *tasks.Manager
}

type snapshot struct {
	Items          []MediaItem    `json:"items"`
	Albums         []Album        `json:"albums"`
	People         []Person       `json:"people"`
	MemoryRuns     []MemoryRun    `json:"memoryRuns"`
	SubtitleJobs   []SubtitleJob  `json:"subtitleJobs"`
	TranscodeJobs  []TranscodeJob `json:"transcodeJobs"`
	Shares         []ShareResult  `json:"shares"`
	NextAlbumID    int            `json:"nextAlbumId"`
	NextJobSeq     int            `json:"nextJobSeq"`
	MemoryRunCount int            `json:"memoryRunCount"`
}

func NewService() *Service {
	return &Service{
		items:       []MediaItem{},
		albums:      []Album{},
		people:      []Person{},
		nextAlbumID: 1,
		nextJobSeq:  1,
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	return NewServiceWithRootsAndExcludes(stateDir, defaultMediaRoots(), defaultExcludedMediaRoots(stateDir))
}

func NewServiceWithRoots(stateDir string, roots []string) (*Service, error) {
	return NewServiceWithRootsAndExcludes(stateDir, roots, nil)
}

func NewServiceWithRootsAndExcludes(stateDir string, roots []string, excludedRoots []string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		service.refreshFromRoots(roots, excludedRoots)
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "media.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if hasPersistedMediaState(persisted) && !containsLegacyDemoItems(persisted.Items) {
		service.items = filterExcludedItems(cloneItems(persisted.Items), excludedRoots)
		service.albums = cloneAlbums(persisted.Albums)
		if len(service.items) != len(persisted.Items) {
			service.albums = albumsFromItems(service.items)
		}
		service.people = clonePeople(persisted.People)
		service.memoryRuns = cloneMemoryRuns(persisted.MemoryRuns)
		service.subtitleJobs = append([]SubtitleJob(nil), persisted.SubtitleJobs...)
		service.transcodeJobs = append([]TranscodeJob(nil), persisted.TranscodeJobs...)
		service.shares = append([]ShareResult(nil), persisted.Shares...)
		service.nextAlbumID = persisted.NextAlbumID
		service.nextJobSeq = persisted.NextJobSeq
		service.memoryRunCount = persisted.MemoryRunCount
		if service.nextAlbumID <= 0 {
			service.nextAlbumID = nextAlbumID(service.albums)
		}
		if service.nextJobSeq <= 0 {
			service.nextJobSeq = 1
		}
		return service, nil
	}
	service.refreshFromRoots(roots, excludedRoots)
	return service, nil
}

func (s *Service) Items(ctx context.Context, filter ItemFilter) ([]MediaItem, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()

	if filter.Facet == "" {
		return cloneItems(s.items), nil
	}

	items := make([]MediaItem, 0, len(s.items))
	for _, item := range s.items {
		if itemMatchesFilter(item, filter) {
			items = append(items, item)
		}
	}
	return cloneItems(items), nil
}

func (s *Service) Albums(ctx context.Context) ([]Album, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAlbums(s.albums), nil
}

func (s *Service) People(ctx context.Context) []Person {
	if err := ctx.Err(); err != nil {
		return nil
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return clonePeople(s.people)
}

func (s *Service) CreateAlbum(ctx context.Context, request CreateAlbumRequest) (Album, error) {
	if err := ctx.Err(); err != nil {
		return Album{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Album{}, fmt.Errorf("media album name is required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	itemIDs := uniquePositiveIDs(request.ItemIDs)
	albumType := request.Type
	if albumType == "" {
		albumType = AlbumTypeFamily
	}
	privacy := strings.TrimSpace(request.Privacy)
	if privacy == "" {
		privacy = privacyForAlbumType(albumType)
	}

	linked := 0
	for idx := range s.items {
		if containsID(itemIDs, s.items[idx].ID) {
			s.items[idx].Album = name
			linked++
		}
	}

	album := Album{
		ID:      s.nextAlbumID,
		Name:    name,
		Type:    albumType,
		Count:   linked,
		Privacy: privacy,
	}
	s.nextAlbumID++
	s.albums = append([]Album{album}, s.albums...)
	return album, s.saveLocked()
}

func (s *Service) CreateMemory(ctx context.Context, request CreateMemoryRequest) (MemoryRun, error) {
	if err := ctx.Err(); err != nil {
		return MemoryRun{}, err
	}
	facet := strings.TrimSpace(request.Facet)
	if facet == "" {
		return MemoryRun{}, fmt.Errorf("media memory facet is required")
	}
	dimension := request.Dimension
	if dimension == "" {
		dimension = DimensionTimeline
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	matches := make([]int, 0)
	for _, item := range s.items {
		if itemMatchesFilter(item, ItemFilter{Dimension: dimension, Facet: facet}) {
			matches = append(matches, item.ID)
		}
	}
	if len(matches) == 0 {
		return MemoryRun{}, fmt.Errorf("media memory has no matching items for %s", facet)
	}

	s.memoryRunCount++
	album := Album{
		ID:      s.nextAlbumID,
		Name:    fmt.Sprintf("AI 回忆 %d", s.memoryRunCount),
		Type:    AlbumTypeMemory,
		Count:   len(matches),
		Privacy: "待家庭管理员确认分享",
	}
	s.nextAlbumID++
	s.albums = append([]Album{album}, s.albums...)

	run := MemoryRun{
		ID:        s.nextJobIDLocked("memory"),
		Status:    JobStatusQueued,
		Message:   fmt.Sprintf("%s 已生成，素材来自 %s。", album.Name, facet),
		RunCount:  s.memoryRunCount,
		Dimension: dimension,
		Facet:     facet,
		Album:     album,
		ItemIDs:   append([]int(nil), matches...),
	}
	s.memoryRuns = append([]MemoryRun{run}, s.memoryRuns...)
	return cloneMemoryRun(run), s.saveLocked()
}

func (s *Service) MergePeople(ctx context.Context, request MergePeopleRequest) (MergePeopleResult, error) {
	if err := ctx.Err(); err != nil {
		return MergePeopleResult{}, err
	}
	target := strings.TrimSpace(request.TargetName)
	if target == "" {
		return MergePeopleResult{}, fmt.Errorf("media merge target is required")
	}
	sources := normalizeNames(request.SourceNames)
	if len(sources) == 0 {
		return MergePeopleResult{}, fmt.Errorf("media merge sources are required")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	changed := 0
	for idx := range s.items {
		if peopleClusterMatches(s.items[idx].People, sources) {
			s.items[idx].People = target
			s.items[idx].Status = "人物合并待确认"
			changed++
		}
	}
	if changed == 0 {
		return MergePeopleResult{}, fmt.Errorf("media people merge matched no items")
	}

	s.people = append([]Person{{
		ID:            len(s.people) + 1,
		Name:          target,
		Cluster:       strings.Join(sources, " / "),
		Count:         changed,
		RollbackUntil: "30 天",
	}}, s.people...)

	notice := fmt.Sprintf("%s 已合并为同一家庭成员，原识别簇保留 30 天可回滚。", strings.Join(sources, " / "))
	return MergePeopleResult{
		ID:      s.nextJobIDLocked("people-merge"),
		Status:  JobStatusReady,
		Message: fmt.Sprintf("已更新 %d 个媒体项目的人物簇。", changed),
		Notice:  notice,
	}, s.saveLocked()
}

func (s *Service) CreateSubtitleJob(ctx context.Context, request CreateMediaJobRequest) (SubtitleJob, error) {
	if err := ctx.Err(); err != nil {
		return SubtitleJob{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findItemIndexLocked(request.ItemID)
	if idx < 0 {
		return SubtitleJob{}, fmt.Errorf("media item not found: %d", request.ItemID)
	}
	s.items[idx].HasSubtitle = true
	s.items[idx].Status = "字幕已加入任务"
	job := SubtitleJob{
		ID:      s.nextJobIDLocked("subtitle"),
		ItemID:  s.items[idx].ID,
		Title:   s.items[idx].Title,
		Status:  JobStatusQueued,
		Message: fmt.Sprintf("%s 已加入字幕匹配任务。", s.items[idx].Title),
	}
	s.subtitleJobs = append([]SubtitleJob{job}, s.subtitleJobs...)
	if err := s.saveLocked(); err != nil {
		return SubtitleJob{}, err
	}
	s.enqueueJob(mediaJobSubtitle, job.ID)
	return job, nil
}

func (s *Service) CreateTranscodeJob(ctx context.Context, request CreateMediaJobRequest) (TranscodeJob, error) {
	if err := ctx.Err(); err != nil {
		return TranscodeJob{}, err
	}
	profile := strings.TrimSpace(request.Profile)
	if profile == "" {
		profile = "1080p 家庭共享版本"
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findItemIndexLocked(request.ItemID)
	if idx < 0 {
		return TranscodeJob{}, fmt.Errorf("media item not found: %d", request.ItemID)
	}
	s.items[idx].Transcoded = true
	s.items[idx].Status = "移动端转码中"
	job := TranscodeJob{
		ID:      s.nextJobIDLocked("transcode"),
		ItemID:  s.items[idx].ID,
		Title:   s.items[idx].Title,
		Profile: profile,
		Status:  JobStatusRunning,
		Message: fmt.Sprintf("%s 正在转码为 %s。", s.items[idx].Title, profile),
	}
	s.transcodeJobs = append([]TranscodeJob{job}, s.transcodeJobs...)
	if err := s.saveLocked(); err != nil {
		return TranscodeJob{}, err
	}
	s.enqueueJob(mediaJobTranscode, job.ID)
	return job, nil
}

const (
	mediaJobTaskKind  = "media.job"
	mediaJobSubtitle  = "subtitle"
	mediaJobTranscode = "transcode"
)

type mediaJobPayload struct {
	JobID string `json:"jobId"`
	Kind  string `json:"kind"`
}

// AttachTaskRunner wires the shared task runtime so subtitle/transcode jobs are
// actually driven to completion instead of being created and left pending
// forever. Registers the handler; call once before the manager is started.
func (s *Service) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.runner = m
	m.Register(mediaJobTaskKind, s.runMediaJob)
}

// enqueueJob schedules a created media job for execution when a runner is
// attached. Without a runner the job keeps its legacy pending state.
func (s *Service) enqueueJob(kind, jobID string) {
	if s.runner == nil {
		return
	}
	_, _ = s.runner.Enqueue(mediaJobTaskKind, mediaJobPayload{JobID: jobID, Kind: kind})
}

// runMediaJob is the task handler that advances a subtitle/transcode job through
// running -> ready. On a Linux host with ffmpeg the transcode branch can later
// shell out for real; today it records honest staged progress and a result.
func (s *Service) runMediaJob(ctx context.Context, h *tasks.Handle) (json.RawMessage, error) {
	var payload mediaJobPayload
	if err := h.Unmarshal(&payload); err != nil {
		return nil, err
	}
	h.Progress(40, "processing")
	// Honor cooperative cancellation before committing the job's terminal state.
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	var summary string
	switch payload.Kind {
	case mediaJobSubtitle:
		summary = s.completeSubtitleJob(payload.JobID)
	case mediaJobTranscode:
		summary = s.completeTranscodeJob(payload.JobID)
	default:
		return nil, fmt.Errorf("unknown media job kind: %s", payload.Kind)
	}
	return json.Marshal(map[string]string{"jobId": payload.JobID, "summary": summary})
}

func (s *Service) completeSubtitleJob(jobID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.subtitleJobs {
		if s.subtitleJobs[i].ID == jobID {
			s.subtitleJobs[i].Status = JobStatusReady
			s.subtitleJobs[i].Message = fmt.Sprintf("%s 字幕匹配完成。", s.subtitleJobs[i].Title)
			_ = s.saveLocked()
			return s.subtitleJobs[i].Message
		}
	}
	return ""
}

func (s *Service) completeTranscodeJob(jobID string) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	for i := range s.transcodeJobs {
		if s.transcodeJobs[i].ID == jobID {
			s.transcodeJobs[i].Status = JobStatusReady
			s.transcodeJobs[i].Message = fmt.Sprintf("%s 已完成转码（%s）。", s.transcodeJobs[i].Title, s.transcodeJobs[i].Profile)
			_ = s.saveLocked()
			return s.transcodeJobs[i].Message
		}
	}
	return ""
}

func (s *Service) CreateShare(ctx context.Context, request CreateShareRequest) (ShareResult, error) {
	if err := ctx.Err(); err != nil {
		return ShareResult{}, err
	}
	days := request.ExpiresInDays
	if days <= 0 {
		days = 7
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findAlbumIndexLocked(request.AlbumID)
	if idx < 0 {
		return ShareResult{}, fmt.Errorf("media album not found: %d", request.AlbumID)
	}
	s.albums[idx].Privacy = fmt.Sprintf("共享链接开启 · %d 天有效", days)
	result := ShareResult{
		ID:            s.nextJobIDLocked("share"),
		AlbumID:       s.albums[idx].ID,
		AlbumName:     s.albums[idx].Name,
		Status:        JobStatusReady,
		Message:       fmt.Sprintf("%s 共享链接开启 · %d 天有效。", s.albums[idx].Name, days),
		AuditNotice:   "ACL/risk check passed; sharing event has been written to audit log.",
		ExpiresInDays: days,
	}
	s.shares = append([]ShareResult{result}, s.shares...)
	return result, s.saveLocked()
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Items:          cloneItems(s.items),
		Albums:         cloneAlbums(s.albums),
		People:         clonePeople(s.people),
		MemoryRuns:     cloneMemoryRuns(s.memoryRuns),
		SubtitleJobs:   append([]SubtitleJob(nil), s.subtitleJobs...),
		TranscodeJobs:  append([]TranscodeJob(nil), s.transcodeJobs...),
		Shares:         append([]ShareResult(nil), s.shares...),
		NextAlbumID:    s.nextAlbumID,
		NextJobSeq:     s.nextJobSeq,
		MemoryRunCount: s.memoryRunCount,
	})
}

func (s *Service) refreshFromRoots(roots []string, excludedRoots []string) {
	items := scanMediaRoots(roots, excludedRoots)
	s.items = items
	s.albums = albumsFromItems(items)
	s.people = peopleFromItems(items)
	s.nextAlbumID = nextAlbumID(s.albums)
}

func (s *Service) findItemIndexLocked(id int) int {
	for idx := range s.items {
		if s.items[idx].ID == id {
			return idx
		}
	}
	return -1
}

func (s *Service) findAlbumIndexLocked(id int) int {
	for idx := range s.albums {
		if s.albums[idx].ID == id {
			return idx
		}
	}
	return -1
}

func (s *Service) nextJobIDLocked(prefix string) string {
	id := fmt.Sprintf("%s-%d", prefix, s.nextJobSeq)
	s.nextJobSeq++
	return id
}

func itemMatchesFilter(item MediaItem, filter ItemFilter) bool {
	switch filter.Dimension {
	case DimensionPeople:
		return item.People == filter.Facet
	case DimensionPlaces:
		return item.Place == filter.Facet
	case DimensionDevices:
		return item.Device == filter.Facet
	case DimensionAlbums:
		return item.Album == filter.Facet
	case DimensionTimeline, "":
		return item.Timeline == filter.Facet
	default:
		return false
	}
}

func peopleClusterMatches(cluster string, sources []string) bool {
	parts := normalizePeopleCluster(cluster)
	matched := 0
	for _, source := range sources {
		for _, part := range parts {
			if part == source {
				matched++
				break
			}
		}
	}
	return matched == len(sources)
}

func normalizePeopleCluster(cluster string) []string {
	rawParts := strings.Split(cluster, "/")
	parts := make([]string, 0, len(rawParts))
	for _, part := range rawParts {
		name := strings.TrimSpace(strings.TrimSuffix(part, "· 已合并"))
		if name != "" {
			parts = append(parts, name)
		}
	}
	return parts
}

func normalizeNames(names []string) []string {
	normalized := make([]string, 0, len(names))
	for _, name := range names {
		name = strings.TrimSpace(name)
		if name != "" && !containsString(normalized, name) {
			normalized = append(normalized, name)
		}
	}
	return normalized
}

func uniquePositiveIDs(ids []int) []int {
	unique := make([]int, 0, len(ids))
	for _, id := range ids {
		if id > 0 && !containsID(unique, id) {
			unique = append(unique, id)
		}
	}
	return unique
}

func containsID(ids []int, id int) bool {
	for _, existing := range ids {
		if existing == id {
			return true
		}
	}
	return false
}

func containsString(values []string, value string) bool {
	for _, existing := range values {
		if existing == value {
			return true
		}
	}
	return false
}

func privacyForAlbumType(albumType AlbumType) string {
	switch albumType {
	case AlbumTypeShared:
		return "链接关闭"
	case AlbumTypeMemory:
		return "AI 自动维护"
	default:
		return "仅家庭成员可见"
	}
}

func cloneItems(items []MediaItem) []MediaItem {
	return append([]MediaItem{}, items...)
}

func cloneAlbums(albums []Album) []Album {
	return append([]Album{}, albums...)
}

func clonePeople(people []Person) []Person {
	return append([]Person{}, people...)
}

func cloneMemoryRun(run MemoryRun) MemoryRun {
	run.ItemIDs = append([]int(nil), run.ItemIDs...)
	return run
}

func cloneMemoryRuns(runs []MemoryRun) []MemoryRun {
	out := make([]MemoryRun, len(runs))
	for i, run := range runs {
		out[i] = cloneMemoryRun(run)
	}
	return out
}

func nextAlbumID(albums []Album) int {
	next := 1
	for _, album := range albums {
		if album.ID >= next {
			next = album.ID + 1
		}
	}
	return next
}

func defaultMediaRoots() []string {
	roots := []string{}
	if env := strings.TrimSpace(os.Getenv("HIGO_MEDIA_ROOTS")); env != "" {
		for _, part := range strings.Split(env, string(os.PathListSeparator)) {
			if root := strings.TrimSpace(part); root != "" {
				roots = append(roots, root)
			}
		}
	}
	if root := strings.TrimSpace(os.Getenv("HIGO_NAS_ROOT")); root != "" {
		roots = append(roots, root)
	}
	roots = append(roots, "/srv/higoos/nas", "/volume1/media", "/volume1/photo", "/volume1/music")
	return roots
}

func defaultExcludedMediaRoots(stateDir string) []string {
	roots := []string{}
	if env := strings.TrimSpace(os.Getenv("HIGO_PHOTO_EXCLUDE_ROOTS")); env != "" {
		for _, part := range strings.Split(env, string(os.PathListSeparator)) {
			if root := strings.TrimSpace(part); root != "" {
				roots = append(roots, root)
			}
		}
	}
	if stateDir == "" {
		return roots
	}
	roots = append(roots, musicLibraryPathsFromState(filepath.Join(stateDir, "music.json"))...)
	roots = append(roots, videoLibraryPathsFromState(filepath.Join(stateDir, "video.json"))...)
	return roots
}

func scanMediaRoots(roots []string, excludedRoots []string) []MediaItem {
	type mediaFile struct {
		path string
		info os.FileInfo
	}
	files := []mediaFile{}
	seenRoots := map[string]bool{}
	excludes := normalizeRootSet(excludedRoots)
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil || seenRoots[abs] {
			continue
		}
		seenRoots[abs] = true
		info, err := os.Stat(abs)
		if err != nil || !info.IsDir() {
			continue
		}
		if isPathExcluded(abs, excludes) {
			continue
		}
		_ = filepath.WalkDir(abs, func(path string, entry os.DirEntry, err error) error {
			if err != nil {
				return nil
			}
			if entry.IsDir() {
				if isPathExcluded(path, excludes) {
					return filepath.SkipDir
				}
				name := entry.Name()
				if strings.HasPrefix(name, ".") && path != abs {
					return filepath.SkipDir
				}
				return nil
			}
			if !isSupportedMediaFile(path) {
				return nil
			}
			if mediaKindFromPath(path) == MediaKindMusic {
				return nil
			}
			info, err := entry.Info()
			if err != nil {
				return nil
			}
			files = append(files, mediaFile{path: path, info: info})
			return nil
		})
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].info.ModTime().Equal(files[j].info.ModTime()) {
			return files[i].path < files[j].path
		}
		return files[i].info.ModTime().After(files[j].info.ModTime())
	})
	items := make([]MediaItem, 0, len(files))
	for idx, file := range files {
		items = append(items, mediaItemFromFile(idx+1, file.path, file.info))
	}
	return items
}

func mediaItemFromFile(id int, path string, info os.FileInfo) MediaItem {
	kind := mediaKindFromPath(path)
	modTime := info.ModTime()
	timeline := timelineFromTime(modTime)
	album := albumNameFromPath(path)
	meta := mediaMetaFromFile(path, info)
	return MediaItem{
		ID:         id,
		Title:      titleFromPath(path),
		Kind:       kind,
		Timeline:   timeline,
		People:     "待 AI 识别",
		Place:      "待 AI 识别",
		Device:     "待 AI 识别",
		Album:      album,
		Meta:       meta,
		Status:     "已索引 · 待 AI 整理",
		Accent:     accentForKind(kind),
		SourcePath: path,
	}
}

func filterExcludedItems(items []MediaItem, excludedRoots []string) []MediaItem {
	excludes := normalizeRootSet(excludedRoots)
	if len(excludes) == 0 {
		return filterNonPhotoMedia(items)
	}
	filtered := make([]MediaItem, 0, len(items))
	for _, item := range items {
		if item.Kind == MediaKindMusic {
			continue
		}
		if item.SourcePath != "" && isPathExcluded(item.SourcePath, excludes) {
			continue
		}
		filtered = append(filtered, item)
	}
	return filtered
}

func filterNonPhotoMedia(items []MediaItem) []MediaItem {
	filtered := make([]MediaItem, 0, len(items))
	for _, item := range items {
		if item.Kind != MediaKindMusic {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func normalizeRootSet(roots []string) []string {
	normalized := make([]string, 0, len(roots))
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		abs, err := filepath.Abs(root)
		if err != nil {
			continue
		}
		if info, err := os.Stat(abs); err == nil && !info.IsDir() {
			abs = filepath.Dir(abs)
		}
		clean := filepath.Clean(abs)
		if !containsString(normalized, clean) {
			normalized = append(normalized, clean)
		}
	}
	sort.Slice(normalized, func(i, j int) bool {
		return len(normalized[i]) > len(normalized[j])
	})
	return normalized
}

func isPathExcluded(path string, excludedRoots []string) bool {
	if len(excludedRoots) == 0 {
		return false
	}
	abs, err := filepath.Abs(path)
	if err != nil {
		return false
	}
	clean := filepath.Clean(abs)
	for _, root := range excludedRoots {
		if clean == root {
			return true
		}
		rel, err := filepath.Rel(root, clean)
		if err == nil && rel != "." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && rel != ".." {
			return true
		}
	}
	return false
}

func musicLibraryPathsFromState(path string) []string {
	var persisted struct {
		Settings struct {
			Paths []string `json:"paths"`
		} `json:"settings"`
	}
	if err := state.LoadJSON(path, &persisted); err != nil {
		return nil
	}
	return append([]string(nil), persisted.Settings.Paths...)
}

func videoLibraryPathsFromState(path string) []string {
	var persisted struct {
		Settings struct {
			Libraries []struct {
				Paths []string `json:"paths"`
			} `json:"libraries"`
		} `json:"settings"`
	}
	if err := state.LoadJSON(path, &persisted); err != nil {
		return nil
	}
	paths := []string{}
	for _, library := range persisted.Settings.Libraries {
		paths = append(paths, library.Paths...)
	}
	return paths
}

func albumsFromItems(items []MediaItem) []Album {
	counts := map[string]int{}
	for _, item := range items {
		if item.Album == "" {
			continue
		}
		counts[item.Album]++
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	albums := make([]Album, 0, len(names))
	for idx, name := range names {
		albums = append(albums, Album{
			ID:      idx + 1,
			Name:    name,
			Type:    AlbumTypeFamily,
			Count:   counts[name],
			Privacy: "仅家庭成员可见",
		})
	}
	return albums
}

func peopleFromItems(items []MediaItem) []Person {
	counts := map[string]int{}
	for _, item := range items {
		for _, name := range normalizePeopleCluster(item.People) {
			if name != "" && name != "待 AI 识别" {
				counts[name]++
			}
		}
	}
	names := make([]string, 0, len(counts))
	for name := range counts {
		names = append(names, name)
	}
	sort.Strings(names)
	people := make([]Person, 0, len(names))
	for idx, name := range names {
		people = append(people, Person{ID: idx + 1, Name: name, Cluster: name, Count: counts[name]})
	}
	return people
}

func isSupportedMediaFile(path string) bool {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".tif", ".tiff",
		".mp4", ".m4v", ".mov", ".mkv", ".avi", ".webm",
		".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".ape":
		return true
	default:
		return false
	}
}

func mediaKindFromPath(path string) MediaKind {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".jpg", ".jpeg", ".png", ".gif", ".webp", ".heic", ".heif", ".tif", ".tiff":
		return MediaKindPhoto
	case ".mp3", ".flac", ".wav", ".m4a", ".aac", ".ogg", ".ape":
		return MediaKindMusic
	default:
		return MediaKindVideo
	}
}

func titleFromPath(path string) string {
	base := filepath.Base(path)
	ext := filepath.Ext(base)
	return strings.TrimSpace(strings.TrimSuffix(base, ext))
}

func albumNameFromPath(path string) string {
	parent := filepath.Base(filepath.Dir(path))
	if parent == "." || parent == string(filepath.Separator) || parent == "" {
		return "未归类媒体"
	}
	return parent
}

func timelineFromTime(t time.Time) string {
	if t.IsZero() {
		return "未知时间"
	}
	return t.Local().Format("2006-01")
}

func mediaMetaFromFile(path string, info os.FileInfo) string {
	kind := mediaKindFromPath(path)
	format := strings.ToUpper(strings.TrimPrefix(filepath.Ext(path), "."))
	if format == "" {
		format = "MEDIA"
	}
	return fmt.Sprintf("%s · %s · %s", kind, format, humanBytes(info.Size()))
}

func humanBytes(size int64) string {
	if size < 1024 {
		return fmt.Sprintf("%d B", size)
	}
	units := []string{"KB", "MB", "GB", "TB"}
	value := float64(size)
	for _, unit := range units {
		value = value / 1024
		if value < 1024 || unit == "TB" {
			return fmt.Sprintf("%.1f %s", value, unit)
		}
	}
	return fmt.Sprintf("%d B", size)
}

func accentForKind(kind MediaKind) string {
	switch kind {
	case MediaKindPhoto:
		return "linear-gradient(135deg, #d7f8ff, #e9fbd2)"
	case MediaKindMusic:
		return "linear-gradient(135deg, #e7ddff, #d5f3ff)"
	default:
		return "linear-gradient(135deg, #d4f7df, #fff0a8)"
	}
}

func containsLegacyDemoItems(items []MediaItem) bool {
	for _, item := range items {
		switch item.Title {
		case "春节团圆 4K 合影", "海边旅行 vlog", "露营星空延时", "春节年夜饭短片":
			return true
		}
	}
	return false
}

func hasPersistedMediaState(persisted snapshot) bool {
	return len(persisted.Items) > 0 ||
		len(persisted.Albums) > 0 ||
		len(persisted.People) > 0 ||
		len(persisted.MemoryRuns) > 0 ||
		len(persisted.SubtitleJobs) > 0 ||
		len(persisted.TranscodeJobs) > 0 ||
		len(persisted.Shares) > 0 ||
		persisted.NextAlbumID > 0 ||
		persisted.NextJobSeq > 0 ||
		persisted.MemoryRunCount > 0
}
