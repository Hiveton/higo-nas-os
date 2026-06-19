package appcenter

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"

	hdocker "higoos/server-go/internal/docker"
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
	Image           string   `json:"image,omitempty"`
	Ports           []string `json:"ports"`
	ContainerID     string   `json:"containerId,omitempty"`
	Installed       bool     `json:"installed"`
	Running         bool     `json:"running"`
	UpdateAvailable bool     `json:"updateAvailable"`
}

// ContainerRunner is the slice of the docker domain the app center needs to
// orchestrate real containers. *docker.DevService satisfies it (and routes to
// the live docker CLI on a Linux host).
type ContainerRunner interface {
	CreateContainer(ctx context.Context, request hdocker.CreateContainerRequest) (hdocker.Container, error)
	Start(ctx context.Context, containerID string) (hdocker.Container, error)
	Stop(ctx context.Context, containerID string) (hdocker.Container, error)
	RemoveContainer(ctx context.Context, containerID string, request hdocker.RemoveContainerRequest) error
}

type Service struct {
	mu        sync.RWMutex
	apps      []App
	statePath string
	docker    ContainerRunner
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
			Image:           "ghcr.io/home-assistant/home-assistant:stable",
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
			Image:           "qdrant/qdrant:latest",
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
			Image:           "ghcr.io/paperless-ngx/paperless-ngx:latest",
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
			Image:           "tailscale/tailscale:latest",
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

// AttachDocker wires the docker domain so install/start/stop orchestrate real
// containers. Without it the app center keeps its in-memory state behaviour.
func (s *Service) AttachDocker(d ContainerRunner) {
	s.docker = d
}

func (s *Service) Install(ctx context.Context, id string) (App, error) {
	app, err := s.update(ctx, id, func(app *App) {
		app.Installed = true
		app.Running = true
		app.Version = app.LatestVersion
		app.Status = "运行中"
		app.UpdateAvailable = false
	})
	if err != nil {
		return App{}, err
	}
	return s.orchestrateRunning(ctx, app)
}

// orchestrateRunning ensures a real container exists and is started for the app
// when a docker runner is attached. Without docker (or an image) it is a no-op
// and the app keeps its in-memory running state.
func (s *Service) orchestrateRunning(ctx context.Context, app App) (App, error) {
	if s.docker == nil || app.Image == "" {
		return app, nil
	}
	containerID := app.ContainerID
	if containerID == "" {
		container, err := s.docker.CreateContainer(ctx, hdocker.CreateContainerRequest{
			Image: app.Image,
			Name:  app.ID,
			Ports: app.Ports,
		})
		if err != nil {
			return s.degrade(ctx, app.ID, "容器创建失败")
		}
		containerID = container.ID
	}
	if _, err := s.docker.Start(ctx, containerID); err != nil {
		return s.degrade(ctx, app.ID, "容器启动失败")
	}
	return s.update(ctx, app.ID, func(a *App) {
		a.ContainerID = containerID
		a.Running = true
		a.Status = "运行中"
	})
}

// degrade keeps the app's optimistic installed/running state when container
// orchestration is unavailable (e.g. a dev host without the image), recording
// the degradation in status. This matches the platform's dev-fallback norm: a
// real container is created on hosts where docker works, and the UI is never
// left in a broken half-state otherwise.
func (s *Service) degrade(ctx context.Context, id, reason string) (App, error) {
	return s.update(ctx, id, func(a *App) {
		a.Running = true
		a.Status = "运行中（" + reason + "，已降级为本地状态）"
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
	app, err := s.update(ctx, id, func(app *App) {
		app.Installed = true
		app.Running = true
		if app.Version == "" {
			app.Version = app.LatestVersion
		}
		app.Status = "运行中"
	})
	if err != nil {
		return App{}, err
	}
	return s.orchestrateRunning(ctx, app)
}

func (s *Service) Stop(ctx context.Context, id string) (App, error) {
	app, err := s.update(ctx, id, func(app *App) {
		app.Running = false
		app.Status = "已停止"
	})
	if err != nil {
		return App{}, err
	}
	if s.docker != nil && app.ContainerID != "" {
		if _, err := s.docker.Stop(ctx, app.ContainerID); err != nil {
			return s.update(ctx, id, func(a *App) {
				a.Running = false
				a.Status = "已停止（停止容器失败：" + err.Error() + "）"
			})
		}
	}
	return app, nil
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
