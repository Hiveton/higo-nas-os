package devstub

import (
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

type Store struct {
	mu             sync.Mutex
	bootedAt       time.Time
	desktopSession *DesktopSession
	sessionPath    string
}

type App struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Icon    string `json:"icon"`
	Badge   int    `json:"badge,omitempty"`
	Utility bool   `json:"utility,omitempty"`
}

type Window struct {
	ID         string `json:"id"`
	Title      string `json:"title"`
	Subtitle   string `json:"subtitle"`
	Status     string `json:"status"`
	StatusTone string `json:"statusTone"`
	X          int    `json:"x"`
	Y          int    `json:"y"`
	Width      int    `json:"width"`
	Height     int    `json:"height"`
	Z          int    `json:"z"`
}

type Status struct {
	Adapter   string `json:"adapter"`
	Status    string `json:"status"`
	HostOS    string `json:"hostOS"`
	Arch      string `json:"arch"`
	BootedAt  string `json:"bootedAt"`
	AppCount  int    `json:"appCount"`
	ReadyNote string `json:"readyNote"`
}

type IconPosition struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type WindowGeometry struct {
	X      int `json:"x,omitempty"`
	Y      int `json:"y,omitempty"`
	Width  int `json:"width,omitempty"`
	Height int `json:"height,omitempty"`
}

type DesktopSession struct {
	OpenWindowIDs        []string                  `json:"openWindowIds"`
	MinimizedWindowIDs   []string                  `json:"minimizedWindowIds"`
	ActiveWindowID       string                    `json:"activeWindowId"`
	UtilityAppID         string                    `json:"utilityAppId"`
	AssistantVisible     bool                      `json:"assistantVisible"`
	IsCompact            bool                      `json:"isCompact"`
	MaximizedWindowID    string                    `json:"maximizedWindowId"`
	DockOrder            []string                  `json:"dockOrder"`
	PinnedDockAppIDs     []string                  `json:"pinnedDockAppIds"`
	DesktopIconPositions map[string]IconPosition   `json:"desktopIconPositions"`
	WindowGeometries     map[string]WindowGeometry `json:"windowGeometries"`
}

type DesktopSessionPatch struct {
	OpenWindowIDs        *[]string                  `json:"openWindowIds"`
	MinimizedWindowIDs   *[]string                  `json:"minimizedWindowIds"`
	ActiveWindowID       *string                    `json:"activeWindowId"`
	UtilityAppID         *string                    `json:"utilityAppId"`
	AssistantVisible     *bool                      `json:"assistantVisible"`
	IsCompact            *bool                      `json:"isCompact"`
	MaximizedWindowID    *string                    `json:"maximizedWindowId"`
	DockOrder            *[]string                  `json:"dockOrder"`
	PinnedDockAppIDs     *[]string                  `json:"pinnedDockAppIds"`
	DesktopIconPositions *map[string]IconPosition   `json:"desktopIconPositions"`
	WindowGeometries     *map[string]WindowGeometry `json:"windowGeometries"`
}

type SystemInfo struct {
	AppName     string `json:"appName"`
	Environment string `json:"environment"`
	Version     string `json:"version"`
	Adapter     string `json:"adapter"`
	Status      string `json:"status"`
	HostOS      string `json:"hostOS"`
	Arch        string `json:"arch"`
	BootedAt    string `json:"bootedAt"`
}

type DoctorReport struct {
	Adapter     string
	HostOS      string
	Arch        string
	BootedAt    string
	AppCount    int
	WindowCount int
	Notes       []string
}

func NewStore() *Store {
	return &Store{bootedAt: time.Now().UTC()}
}

func NewStoreWithStateDir(stateDir string) (*Store, error) {
	store := NewStore()
	if stateDir == "" {
		return store, nil
	}
	store.sessionPath = filepath.Join(stateDir, "desktop-session.json")
	var session DesktopSession
	if err := state.LoadJSON(store.sessionPath, &session); err != nil {
		return nil, err
	}
	if !isZeroDesktopSession(session) {
		if err := store.validateDesktopSession(session); err != nil {
			return nil, fmt.Errorf("desktop session state invalid: %w", err)
		}
		store.desktopSession = &session
	}
	return store, nil
}

func (s *Store) Apps() []App {
	apps := []App{
		{ID: "file-manager", Name: "文件管理", Icon: "/src/assets/higoos-dock/icons/01-file-manager.png", Badge: 2},
		{ID: "storage-monitor", Name: "存储管理", Icon: "/src/assets/higoos-dock/icons/02-storage-manager.png"},
		{ID: "ai-file-steward", Name: "AI 文件管家", Icon: "/src/assets/higoos-dock/icons/03-ai-file-steward.png", Badge: 6},
		{ID: "agent-workbench", Name: "Agent 工作台", Icon: "/src/assets/higoos-dock/icons/04-agent-workbench.png"},
		{ID: "ai-assistant", Name: "AI 助手", Icon: "/src/assets/higoos-dock/icons/05-ai-assistant.png"},
		{ID: "backup-sync", Name: "备份同步", Icon: "/src/assets/higoos-dock/icons/06-backup-sync.png", Badge: 1},
		{ID: "photo-media", Name: "相册媒体", Icon: "/src/assets/higoos-dock/icons/07-photo-media.png"},
		{ID: "download-center", Name: "下载中心", Icon: "/src/assets/higoos-dock/icons/08-download-center.png"},
		{ID: "app-center", Name: "应用中心", Icon: "/src/assets/higoos-dock/icons/09-app-center.png"},
		{ID: "docker", Name: "Docker", Icon: "/src/assets/higoos-dock/icons/10-docker.png"},
		{ID: "security-center", Name: "安全中心", Icon: "/src/assets/higoos-dock/icons/11-security-center.png", Badge: 3},
		{ID: "device-monitor", Name: "设备监控", Icon: "/src/assets/higoos-dock/icons/12-device-monitor.png"},
		{ID: "system-settings", Name: "系统设置", Icon: "/src/assets/higoos-dock/icons/13-system-settings.png"},
		{ID: "remote-access", Name: "远程访问", Icon: "/src/assets/higoos-dock/icons/14-remote-access.png"},
	}
	return append([]App(nil), apps...)
}

func (s *Store) Windows() []Window {
	windows := []Window{
		{ID: "file-manager", Title: "文件管理", Subtitle: "家庭空间 / 团队空间 / 语义搜索", Status: "AI 索引已同步", StatusTone: "green", X: 48, Y: 92, Width: 700, Height: 560, Z: 4},
		{ID: "ai-file-steward", Title: "AI 文件管家", Subtitle: "智能整理 / 权限审计 / 回滚", Status: "6 条建议", StatusTone: "orange", X: 778, Y: 108, Width: 472, Height: 520, Z: 5},
		{ID: "agent-workbench", Title: "Agent 工作台", Subtitle: "工作流 / 工具权限 / 执行确认", Status: "需要确认", StatusTone: "blue", X: 548, Y: 388, Width: 690, Height: 360, Z: 6},
		{ID: "backup-sync", Title: "备份同步", Subtitle: "快照 / 异地同步 / 校验", Status: "1 个任务同步中", StatusTone: "blue", X: 188, Y: 126, Width: 740, Height: 500, Z: 7},
		{ID: "storage-monitor", Title: "存储管理", Subtitle: "主机卷 / SMART / 容量", Status: "后端同步", StatusTone: "green", X: 960, Y: 80, Width: 360, Height: 296, Z: 3},
		{ID: "photo-media", Title: "相册媒体", Subtitle: "时间线 / 人物地点 / 媒体转码", Status: "回忆生成", StatusTone: "blue", X: 118, Y: 118, Width: 760, Height: 536, Z: 8},
		{ID: "download-center", Title: "下载中心", Subtitle: "BT / HTTP / 磁力 / 自动归档", Status: "队列运行", StatusTone: "green", X: 228, Y: 136, Width: 720, Height: 500, Z: 9},
		{ID: "app-center", Title: "应用中心", Subtitle: "套件 / 容器应用 / 更新", Status: "1 个更新", StatusTone: "orange", X: 248, Y: 118, Width: 760, Height: 520, Z: 10},
		{ID: "docker", Title: "Docker", Subtitle: "容器 / Compose / 端口资源", Status: "4 个运行", StatusTone: "green", X: 260, Y: 112, Width: 760, Height: 520, Z: 11},
		{ID: "security-center", Title: "安全中心", Subtitle: "权限 / 风险 / 审计回滚", Status: "3 条风险", StatusTone: "red", X: 300, Y: 126, Width: 760, Height: 530, Z: 12},
		{ID: "device-monitor", Title: "设备监控", Subtitle: "性能趋势 / 告警 / 系统日志", Status: "实时", StatusTone: "green", X: 170, Y: 104, Width: 760, Height: 520, Z: 13},
		{ID: "system-settings", Title: "系统设置", Subtitle: "网络 / 模型 / 隐私 / 更新", Status: "已同步", StatusTone: "blue", X: 210, Y: 96, Width: 760, Height: 536, Z: 14},
		{ID: "remote-access", Title: "远程访问", Subtitle: "域名 / 穿透 / MFA / 分享检查", Status: "安全", StatusTone: "green", X: 248, Y: 116, Width: 740, Height: 514, Z: 15},
	}
	return append([]Window(nil), windows...)
}

func (s *Store) DesktopSession() DesktopSession {
	s.mu.Lock()
	defer s.mu.Unlock()

	session := s.desktopSessionLocked()
	return cloneDesktopSession(session)
}

func (s *Store) UpdateDesktopSession(patch DesktopSessionPatch) (DesktopSession, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	next := cloneDesktopSession(s.desktopSessionLocked())
	if patch.OpenWindowIDs != nil {
		next.OpenWindowIDs = append([]string(nil), (*patch.OpenWindowIDs)...)
	}
	if patch.MinimizedWindowIDs != nil {
		next.MinimizedWindowIDs = append([]string(nil), (*patch.MinimizedWindowIDs)...)
	}
	if patch.ActiveWindowID != nil {
		next.ActiveWindowID = *patch.ActiveWindowID
	}
	if patch.UtilityAppID != nil {
		next.UtilityAppID = *patch.UtilityAppID
	}
	if patch.AssistantVisible != nil {
		next.AssistantVisible = *patch.AssistantVisible
	}
	if patch.IsCompact != nil {
		next.IsCompact = *patch.IsCompact
	}
	if patch.MaximizedWindowID != nil {
		next.MaximizedWindowID = *patch.MaximizedWindowID
	}
	if patch.DockOrder != nil {
		next.DockOrder = append([]string(nil), (*patch.DockOrder)...)
	}
	if patch.PinnedDockAppIDs != nil {
		next.PinnedDockAppIDs = append([]string(nil), (*patch.PinnedDockAppIDs)...)
	}
	if patch.DesktopIconPositions != nil {
		next.DesktopIconPositions = cloneIconPositions(*patch.DesktopIconPositions)
	}
	if patch.WindowGeometries != nil {
		next.WindowGeometries = cloneWindowGeometries(*patch.WindowGeometries)
	}
	if err := s.validateDesktopSession(next); err != nil {
		return DesktopSession{}, err
	}

	s.desktopSession = &next
	if err := s.saveDesktopSessionLocked(next); err != nil {
		return DesktopSession{}, err
	}
	return cloneDesktopSession(next), nil
}

func (s *Store) saveDesktopSessionLocked(session DesktopSession) error {
	if s.sessionPath == "" {
		return nil
	}
	return state.SaveJSON(s.sessionPath, session)
}

func (s *Store) desktopSessionLocked() DesktopSession {
	if s.desktopSession != nil {
		return *s.desktopSession
	}
	session := s.defaultDesktopSession()
	s.desktopSession = &session
	return session
}

func (s *Store) defaultDesktopSession() DesktopSession {
	apps := s.Apps()
	dockOrder := make([]string, 0, len(apps))
	for _, app := range apps {
		dockOrder = append(dockOrder, app.ID)
	}

	return DesktopSession{
		OpenWindowIDs:        []string{"file-manager", "ai-file-steward", "agent-workbench"},
		MinimizedWindowIDs:   []string{},
		ActiveWindowID:       "file-manager",
		UtilityAppID:         "",
		AssistantVisible:     false,
		IsCompact:            false,
		MaximizedWindowID:    "",
		DockOrder:            dockOrder,
		PinnedDockAppIDs:     []string{"file-manager", "ai-file-steward", "ai-assistant", "system-settings"},
		DesktopIconPositions: map[string]IconPosition{},
		WindowGeometries:     map[string]WindowGeometry{},
	}
}

func (s *Store) validateDesktopSession(session DesktopSession) error {
	appIDs := make(map[string]struct{})
	for _, app := range s.Apps() {
		appIDs[app.ID] = struct{}{}
	}
	windowIDs := make(map[string]struct{})
	for _, window := range s.Windows() {
		windowIDs[window.ID] = struct{}{}
	}

	if err := validateIDs("openWindowIds", session.OpenWindowIDs, windowIDs); err != nil {
		return err
	}
	if err := validateIDs("minimizedWindowIds", session.MinimizedWindowIDs, windowIDs); err != nil {
		return err
	}
	if err := validateIDs("dockOrder", session.DockOrder, appIDs); err != nil {
		return err
	}
	if err := validateIDs("pinnedDockAppIds", session.PinnedDockAppIDs, appIDs); err != nil {
		return err
	}
	if err := validateOptionalID("activeWindowId", session.ActiveWindowID, appIDs); err != nil {
		return err
	}
	if err := validateOptionalID("utilityAppId", session.UtilityAppID, appIDs); err != nil {
		return err
	}
	if err := validateOptionalID("maximizedWindowId", session.MaximizedWindowID, windowIDs); err != nil {
		return err
	}
	for id := range session.DesktopIconPositions {
		if _, ok := appIDs[id]; !ok {
			return fmt.Errorf("desktopIconPositions contains unknown app id %q", id)
		}
	}
	for id := range session.WindowGeometries {
		if _, ok := windowIDs[id]; !ok {
			return fmt.Errorf("windowGeometries contains unknown window id %q", id)
		}
	}
	return nil
}

func validateIDs(field string, values []string, allowed map[string]struct{}) error {
	seen := make(map[string]struct{}, len(values))
	for _, value := range values {
		if _, ok := allowed[value]; !ok {
			return fmt.Errorf("%s contains unknown id %q", field, value)
		}
		if _, ok := seen[value]; ok {
			return fmt.Errorf("%s contains duplicate id %q", field, value)
		}
		seen[value] = struct{}{}
	}
	return nil
}

func validateOptionalID(field string, value string, allowed map[string]struct{}) error {
	if value == "" {
		return nil
	}
	if _, ok := allowed[value]; !ok {
		return fmt.Errorf("%s contains unknown id %q", field, value)
	}
	return nil
}

func isZeroDesktopSession(session DesktopSession) bool {
	return len(session.OpenWindowIDs) == 0 &&
		len(session.DockOrder) == 0 &&
		len(session.PinnedDockAppIDs) == 0 &&
		session.ActiveWindowID == "" &&
		session.UtilityAppID == "" &&
		session.MaximizedWindowID == ""
}

func cloneDesktopSession(session DesktopSession) DesktopSession {
	return DesktopSession{
		OpenWindowIDs:        append([]string(nil), session.OpenWindowIDs...),
		MinimizedWindowIDs:   append([]string(nil), session.MinimizedWindowIDs...),
		ActiveWindowID:       session.ActiveWindowID,
		UtilityAppID:         session.UtilityAppID,
		AssistantVisible:     session.AssistantVisible,
		IsCompact:            session.IsCompact,
		MaximizedWindowID:    session.MaximizedWindowID,
		DockOrder:            append([]string(nil), session.DockOrder...),
		PinnedDockAppIDs:     append([]string(nil), session.PinnedDockAppIDs...),
		DesktopIconPositions: cloneIconPositions(session.DesktopIconPositions),
		WindowGeometries:     cloneWindowGeometries(session.WindowGeometries),
	}
}

func cloneIconPositions(input map[string]IconPosition) map[string]IconPosition {
	out := make(map[string]IconPosition, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func cloneWindowGeometries(input map[string]WindowGeometry) map[string]WindowGeometry {
	out := make(map[string]WindowGeometry, len(input))
	for key, value := range input {
		out[key] = value
	}
	return out
}

func (s *Store) Status() Status {
	return Status{
		Adapter:   "devstub",
		Status:    "ready",
		HostOS:    runtime.GOOS,
		Arch:      runtime.GOARCH,
		BootedAt:  s.bootedAt.Format(time.RFC3339),
		AppCount:  len(s.Apps()),
		ReadyNote: "Mac-safe deterministic fixtures are active; Linux adapters are not wired in this skeleton.",
	}
}

func (s *Store) SystemInfo(appName, environment, version string) SystemInfo {
	status := s.Status()
	return SystemInfo{
		AppName:     appName,
		Environment: environment,
		Version:     version,
		Adapter:     status.Adapter,
		Status:      status.Status,
		HostOS:      status.HostOS,
		Arch:        status.Arch,
		BootedAt:    status.BootedAt,
	}
}

func (s *Store) DoctorReport() DoctorReport {
	status := s.Status()
	return DoctorReport{
		Adapter:     status.Adapter,
		HostOS:      status.HostOS,
		Arch:        status.Arch,
		BootedAt:    status.BootedAt,
		AppCount:    len(s.Apps()),
		WindowCount: len(s.Windows()),
		Notes: []string{
			status.ReadyNote,
			"PostgreSQL, queues, Linux storage, Docker, and systemd adapters are intentionally outside this foundation slice.",
		},
	}
}
