package appcenter

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	"higoos/server-go/internal/state"
)

type App struct {
	ID              string   `json:"id"`
	Name            string   `json:"name"`
	Category        string   `json:"category"`
	Version         string   `json:"version"`
	LatestVersion   string   `json:"latestVersion"`
	Status          string   `json:"status"`
	Description     string   `json:"description"`
	Source          string   `json:"source"`
	Risk            string   `json:"risk"`
	Resource        string   `json:"resource"`
	Ports           []string `json:"ports"`
	Installed       bool     `json:"installed"`
	Running         bool     `json:"running"`
	UpdateAvailable bool     `json:"updateAvailable"`
}

type Service struct {
	mu        sync.RWMutex
	apps      []App
	statePath string
}

func NewService() *Service {
	return &Service{apps: []App{
		{
			ID:              "home-assistant",
			Name:            "Home Assistant",
			Category:        "智能家居",
			Version:         "2026.4.1",
			LatestVersion:   "2026.5.0",
			Status:          "需更新",
			Description:     "家庭自动化中枢，已接入本地 MQTT 与设备监控。",
			Source:          "官方镜像",
			Risk:            "低风险",
			Resource:        "1 CPU / 1024 MB",
			Ports:           []string{"8123/tcp"},
			Installed:       true,
			Running:         true,
			UpdateAvailable: true,
		},
		{
			ID:              "qdrant",
			Name:            "Qdrant 向量库",
			Category:        "AI 数据层",
			Version:         "1.14.0",
			LatestVersion:   "1.14.0",
			Status:          "运行中",
			Description:     "为文件语义搜索和 Agent 检索提供本地向量索引。",
			Source:          "官方镜像",
			Risk:            "中风险",
			Resource:        "2 CPU / 2048 MB",
			Ports:           []string{"6333/tcp"},
			Installed:       true,
			Running:         true,
			UpdateAvailable: false,
		},
		{
			ID:              "paperless",
			Name:            "Paperless-ngx",
			Category:        "文档归档",
			Version:         "",
			LatestVersion:   "2.16.1",
			Status:          "可安装",
			Description:     "发票、合同和保修单 OCR 归档，可写入文件管理标签。",
			Source:          "社区精选",
			Risk:            "中风险",
			Resource:        "2 CPU / 1536 MB",
			Ports:           []string{"8000/tcp"},
			Installed:       false,
			Running:         false,
			UpdateAvailable: false,
		},
		{
			ID:              "tailscale",
			Name:            "Tailscale",
			Category:        "远程访问",
			Version:         "1.82.5",
			LatestVersion:   "1.82.5",
			Status:          "已停止",
			Description:     "安全组网工具，可与远程访问策略联动。",
			Source:          "官方镜像",
			Risk:            "低风险",
			Resource:        "1 CPU / 512 MB",
			Ports:           []string{"41641/udp"},
			Installed:       true,
			Running:         false,
			UpdateAvailable: false,
		},
	}}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	service := NewService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "app-center.json")
	var apps []App
	if err := state.LoadJSON(service.statePath, &apps); err != nil {
		return nil, err
	}
	if len(apps) > 0 {
		service.apps = cloneApps(apps)
	}
	return service, nil
}

func (s *Service) Apps(ctx context.Context) ([]App, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneApps(s.apps), nil
}

func (s *Service) Install(ctx context.Context, id string) (App, error) {
	return s.update(ctx, id, func(app *App) {
		app.Installed = true
		app.Running = true
		app.Version = app.LatestVersion
		app.Status = "运行中"
		app.UpdateAvailable = false
	})
}

func (s *Service) Update(ctx context.Context, id string) (App, error) {
	return s.update(ctx, id, func(app *App) {
		app.Installed = true
		app.Running = true
		app.Version = app.LatestVersion
		app.Status = "已更新"
		app.UpdateAvailable = false
	})
}

func (s *Service) Start(ctx context.Context, id string) (App, error) {
	return s.update(ctx, id, func(app *App) {
		app.Installed = true
		app.Running = true
		if app.Version == "" {
			app.Version = app.LatestVersion
		}
		app.Status = "运行中"
	})
}

func (s *Service) Stop(ctx context.Context, id string) (App, error) {
	return s.update(ctx, id, func(app *App) {
		app.Running = false
		app.Status = "已停止"
	})
}

func (s *Service) update(ctx context.Context, id string, mutate func(*App)) (App, error) {
	if err := ctx.Err(); err != nil {
		return App{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	for index := range s.apps {
		if s.apps[index].ID == id {
			mutate(&s.apps[index])
			if err := s.saveLocked(); err != nil {
				return App{}, err
			}
			return cloneApp(s.apps[index]), nil
		}
	}
	return App{}, fmt.Errorf("app center item not found: %s", id)
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, cloneApps(s.apps))
}

func cloneApps(apps []App) []App {
	out := make([]App, 0, len(apps))
	for _, app := range apps {
		out = append(out, cloneApp(app))
	}
	return out
}

func cloneApp(app App) App {
	app.Ports = append([]string(nil), app.Ports...)
	return app
}
