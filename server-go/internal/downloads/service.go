package downloads

import (
	"context"
	"errors"
	"fmt"
	"path"
	"path/filepath"
	"strings"
	"sync"

	"higoos/server-go/internal/state"
)

var (
	ErrTaskNotFound     = errors.New("download task not found")
	ErrProfileNotFound  = errors.New("speed profile not found")
	ErrInvalidTaskInput = errors.New("invalid download task input")
)

type Service struct {
	mu        sync.RWMutex
	tasks     []DownloadTask
	profiles  []SpeedProfile
	nextID    int
	statePath string
}

type snapshot struct {
	Tasks    []DownloadTask `json:"tasks"`
	Profiles []SpeedProfile `json:"profiles"`
	NextID   int            `json:"nextId"`
}

func NewService() *Service {
	return &Service{
		tasks:    seedTasks(),
		profiles: seedSpeedProfiles(),
		nextID:   5,
	}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "downloads.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Tasks) > 0 {
		service.tasks = cloneTasks(persisted.Tasks)
	}
	if len(persisted.Profiles) > 0 {
		service.profiles = append([]SpeedProfile(nil), persisted.Profiles...)
	}
	if persisted.NextID > 0 {
		service.nextID = persisted.NextID
	} else {
		service.nextID = nextTaskID(service.tasks)
	}
	return service, nil
}

func (s *Service) ListTasks(_ context.Context) []DownloadTask {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return cloneTasks(s.tasks)
}

func (s *Service) CreateTask(_ context.Context, request CreateTaskRequest) (DownloadTask, error) {
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
		Progress:    3,
		Speed:       "排队中",
		Status:      StatusRunning,
		Handling:    handlingFor(category, source, rule, request.Name, link),
		Archived:    false,
		ArchiveRule: rule,
	}
	if source == SourceRSS {
		task.Size = "等待解析"
		task.Progress = 0
		task.Speed = "等待订阅"
		task.Status = StatusPaused
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	task.ID = s.nextID
	s.nextID++
	s.tasks = append([]DownloadTask{task}, s.tasks...)
	if err := s.saveLocked(); err != nil {
		return DownloadTask{}, err
	}
	return cloneTask(task), nil
}

func (s *Service) PauseTask(_ context.Context, id int) (DownloadTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return DownloadTask{}, ErrTaskNotFound
	}
	if s.tasks[idx].Status == StatusCompleted {
		return cloneTask(s.tasks[idx]), nil
	}
	s.tasks[idx].Status = StatusPaused
	s.tasks[idx].Speed = "0 KB/s"
	if err := s.saveLocked(); err != nil {
		return DownloadTask{}, err
	}
	return cloneTask(s.tasks[idx]), nil
}

func (s *Service) ResumeTask(_ context.Context, id int) (DownloadTask, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return DownloadTask{}, ErrTaskNotFound
	}
	if s.tasks[idx].Status == StatusCompleted {
		return cloneTask(s.tasks[idx]), nil
	}
	s.tasks[idx].Status = StatusRunning
	if s.tasks[idx].Source == SourceRSS && s.tasks[idx].Progress == 0 {
		s.tasks[idx].Speed = "等待 RSS"
	} else {
		s.tasks[idx].Speed = s.activeProfileLocked().DownloadLimit
	}
	if err := s.saveLocked(); err != nil {
		return DownloadTask{}, err
	}
	return cloneTask(s.tasks[idx]), nil
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
	task.Handling = "已归档到文件管家 /" + task.Category

	return TaskActionResult{
		Task:     cloneTask(*task),
		Message:  "文件管家已自动归档：" + task.Name,
		FilePath: task.ArchiveRule.TargetPath,
	}, s.saveLocked()
}

func (s *Service) DeleteTask(_ context.Context, id int) (TaskActionResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	idx := s.findTaskIndex(id)
	if idx < 0 {
		return TaskActionResult{}, ErrTaskNotFound
	}
	task := cloneTask(s.tasks[idx])
	s.tasks = append(s.tasks[:idx], s.tasks[idx+1:]...)

	message := "已删除任务记录：" + task.Name
	if task.Archived {
		message = "已清理 1 条已归档记录，原文件保留在文件管家。"
	}
	return TaskActionResult{Task: task, Message: message, FilePath: task.ArchiveRule.TargetPath}, s.saveLocked()
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

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Tasks:    cloneTasks(s.tasks),
		Profiles: append([]SpeedProfile(nil), s.profiles...),
		NextID:   s.nextID,
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
	return s.profiles[0]
}

func seedTasks() []DownloadTask {
	return []DownloadTask{
		{
			ID:       1,
			Name:     "纪录片合集 S02",
			Source:   SourceBT,
			Category: "影视",
			Size:     "86.4 GB",
			Progress: 68,
			Speed:    "12.8 MB/s",
			Status:   StatusRunning,
			Handling: "完成后刮削海报并归档到 /Media/TV",
			ArchiveRule: ArchiveRule{
				Category:       "影视",
				TargetPath:     "/Media/TV",
				Tags:           []string{"影视", "纪录片"},
				IndexAfterMove: true,
				ScrapeMetadata: true,
			},
		},
		{
			ID:       2,
			Name:     "家庭音乐精选 FLAC",
			Source:   SourceHTTP,
			Category: "音乐",
			Size:     "12.1 GB",
			Progress: 100,
			Speed:    "0 KB/s",
			Status:   StatusCompleted,
			Handling: "等待导入音乐库",
			ArchiveRule: ArchiveRule{
				Category:       "音乐",
				TargetPath:     "/Music",
				Tags:           []string{"音乐", "FLAC"},
				IndexAfterMove: true,
			},
		},
		{
			ID:       3,
			Name:     "Ubuntu Server 镜像",
			Source:   SourceMagnet,
			Category: "软件",
			Size:     "5.9 GB",
			Progress: 42,
			Speed:    "6.4 MB/s",
			Status:   StatusRunning,
			Handling: "完成后校验 SHA256",
			ArchiveRule: ArchiveRule{
				Category:       "软件",
				TargetPath:     "/Downloads/Software",
				Tags:           []string{"软件", "镜像"},
				IndexAfterMove: true,
				VerifyChecksum: true,
			},
		},
		{
			ID:       4,
			Name:     "每周公开课订阅",
			Source:   SourceRSS,
			Category: "订阅",
			Size:     "2.8 GB",
			Progress: 0,
			Speed:    "等待 RSS",
			Status:   StatusPaused,
			Handling: "新条目自动下载到 /Downloads/Courses",
			ArchiveRule: ArchiveRule{
				Category:       "订阅",
				TargetPath:     "/Downloads/Subscriptions",
				Tags:           []string{"订阅", "课程"},
				IndexAfterMove: true,
			},
		},
	}
}

func seedSpeedProfiles() []SpeedProfile {
	return []SpeedProfile{
		{Name: "智能限速", DownloadLimit: "18 MB/s", UploadLimit: "2 MB/s", Note: "客厅投屏时自动让路", Active: true},
		{Name: "夜间全速", DownloadLimit: "不限速", UploadLimit: "8 MB/s", Note: "00:00-07:00 开启满速"},
		{Name: "家庭优先", DownloadLimit: "6 MB/s", UploadLimit: "1 MB/s", Note: "视频会议和游戏优先"},
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
	case containsAny(text, ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".txt", "文档"):
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
	if strings.HasPrefix(strings.ToLower(link), "magnet:") || strings.Contains(link, "://") {
		return string(source) + " 新任务"
	}
	base := path.Base(link)
	if base == "." || base == "/" {
		return string(source) + " 新任务"
	}
	return base
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
		return "新条目自动下载到 /Downloads/Courses"
	}
	if category == "软件" || rule.VerifyChecksum {
		return "完成后校验 SHA256"
	}
	if category == "影视" && rule.TargetPath == "/Media/TV" {
		return "完成后刮削海报并归档到 /Media/TV"
	}
	if category == "影视" && containsAny(strings.ToLower(name+" "+link), "电影", "movie", "film") {
		return "完成后自动归档到 /Media/Movies"
	}
	return "完成后自动归档到 " + rule.TargetPath
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

func nextTaskID(tasks []DownloadTask) int {
	next := 1
	for _, task := range tasks {
		if task.ID >= next {
			next = task.ID + 1
		}
	}
	return next
}
