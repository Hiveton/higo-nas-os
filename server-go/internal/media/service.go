package media

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"sync"

	"higoos/server-go/internal/state"
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
		items:       seedItems(),
		albums:      seedAlbums(),
		people:      seedPeople(),
		nextAlbumID: 7,
		nextJobSeq:  1,
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "media.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Items) > 0 {
		service.items = cloneItems(persisted.Items)
		service.albums = cloneAlbums(persisted.Albums)
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
	}
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
	return job, s.saveLocked()
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
	return job, s.saveLocked()
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
	return append([]MediaItem(nil), items...)
}

func cloneAlbums(albums []Album) []Album {
	return append([]Album(nil), albums...)
}

func clonePeople(people []Person) []Person {
	return append([]Person(nil), people...)
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

func seedItems() []MediaItem {
	return []MediaItem{
		{
			ID:       1,
			Title:    "春节团圆 4K 合影",
			Kind:     MediaKindPhoto,
			Timeline: "2026 春节",
			People:   "爸爸 / 妈妈",
			Place:    "杭州",
			Device:   "iPhone 17 Pro",
			Album:    "家庭年度相册",
			Meta:     "48MP · HEIC · 18 张连拍",
			Status:   "已完成人物识别",
			Accent:   "linear-gradient(135deg, #bfe7ff, #fff1bf)",
		},
		{
			ID:          2,
			Title:       "海边旅行 vlog",
			Kind:        MediaKindVideo,
			Timeline:    "2025 暑假",
			People:      "小雨 / 爸爸",
			Place:       "三亚",
			Device:      "Sony A7C II",
			Album:       "旅行视频",
			Meta:        "42:16 · H.265 · 4K",
			Status:      "海报墙已刮削",
			Accent:      "linear-gradient(135deg, #bdebd6, #c6d7ff)",
			HasSubtitle: true,
		},
		{
			ID:       3,
			Title:    "家庭钢琴练习",
			Kind:     MediaKindMusic,
			Timeline: "2026 五月",
			People:   "小雨",
			Place:    "客厅",
			Device:   "HiGo 麦克风",
			Album:    "孩子成长记录",
			Meta:     "08:24 · FLAC · 96kHz",
			Status:   "已生成波形索引",
			Accent:   "linear-gradient(135deg, #e8d7ff, #c5f6ff)",
		},
		{
			ID:       4,
			Title:    "露营星空延时",
			Kind:     MediaKindVideo,
			Timeline: "2025 秋游",
			People:   "妈妈 / 小雨",
			Place:    "安吉",
			Device:   "DJI Osmo",
			Album:    "共享露营相册",
			Meta:     "12:02 · ProRes · 4K",
			Status:   "等待转码",
			Accent:   "linear-gradient(135deg, #d1fae5, #fde68a)",
		},
		{
			ID:       5,
			Title:    "春节年夜饭短片",
			Kind:     MediaKindVideo,
			Timeline: "2026 春节",
			People:   "爸爸 / 妈妈 / 小雨",
			Place:    "杭州",
			Device:   "iPhone 17 Pro",
			Album:    "春节回忆",
			Meta:     "03:18 · Dolby Vision · 4K",
			Status:   "智能回忆素材",
			Accent:   "linear-gradient(135deg, #ffd7bf, #d9f99d)",
		},
		{
			ID:       6,
			Title:    "五一湖边骑行",
			Kind:     MediaKindPhoto,
			Timeline: "2026 五一",
			People:   "小雨 / 妈妈",
			Place:    "千岛湖",
			Device:   "GoPro Hero",
			Album:    "五一回忆",
			Meta:     "24MP · JPEG · 126 张",
			Status:   "地点聚类完成",
			Accent:   "linear-gradient(135deg, #bbf7d0, #bfdbfe)",
		},
	}
}

func seedAlbums() []Album {
	return []Album{
		{ID: 1, Name: "家庭年度相册", Type: AlbumTypeFamily, Count: 3862, Privacy: "仅家庭成员可见"},
		{ID: 2, Name: "共享露营相册", Type: AlbumTypeShared, Count: 214, Privacy: "链接关闭"},
		{ID: 3, Name: "旅行视频", Type: AlbumTypeShared, Count: 87, Privacy: "亲友可见"},
		{ID: 4, Name: "孩子成长记录", Type: AlbumTypeMemory, Count: 642, Privacy: "AI 自动维护"},
		{ID: 5, Name: "春节回忆", Type: AlbumTypeMemory, Count: 96, Privacy: "待家庭管理员确认分享"},
		{ID: 6, Name: "五一回忆", Type: AlbumTypeMemory, Count: 128, Privacy: "待家庭管理员确认分享"},
	}
}

func seedPeople() []Person {
	return []Person{
		{ID: 1, Name: "爸爸", Cluster: "爸爸", Count: 3},
		{ID: 2, Name: "妈妈", Cluster: "妈妈", Count: 4},
		{ID: 3, Name: "小雨", Cluster: "小雨", Count: 4},
	}
}
