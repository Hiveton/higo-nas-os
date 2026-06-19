package docker

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
	"higoos/server-go/internal/tasks"
)

const pullTaskKind = "docker.pull"

type DevService struct {
	mu         sync.RWMutex
	stacks     []ComposeStack
	containers []Container
	logs       map[string][]ContainerLog
	pulls      map[string]ImagePullStatus
	logSeq     int
	statePath  string

	runner      *tasks.Manager
	pullCancels map[string]context.CancelFunc // central task id -> pull cancel func
}

type snapshot struct {
	Stacks     []ComposeStack            `json:"stacks"`
	Containers []Container               `json:"containers"`
	Logs       map[string][]ContainerLog `json:"logs"`
	LogSeq     int                       `json:"logSeq"`
}

func NewDevService() *DevService {
	service := &DevService{
		stacks: []ComposeStack{
			{Name: "media-stack", Status: "健康", Services: 4, Ports: "8096 / 51413", Volume: "/volume1/docker/media", Network: "isolated-media"},
			{Name: "home-ai", Status: "需更新", Services: 3, Ports: "3000 / 11434", Volume: "/volume1/docker/ai", Network: "ai-sandbox"},
			{Name: "edge-gateway", Status: "健康", Services: 2, Ports: "80 / 443", Volume: "/volume1/docker/gateway", Network: "dmz-proxy"},
		},
		containers: []Container{
			{
				ID: "media-server", Name: "media-server", Image: "local/media-server:latest", Stack: "media-stack",
				Status: ContainerStatusRunning, CPU: 18, Memory: 42, MemoryText: "1.7 GB / 4 GB",
				Ports:  []string{"8096:8096/tcp", "8920:8920/tcp"},
				Mounts: []string{"/volume1/media:/media:ro", "/volume1/docker/media/server:/config"},
				Env:    []string{"TZ=Asia/Shanghai", "PUID=1000", "PGID=1000"},
				Limit:  ResourceLimit{CPU: 4, MemoryMB: 4096}, Restarts: 1,
				Isolation: "只读媒体库 · 无系统目录",
			},
			{
				ID: "transmission", Name: "transmission", Image: "linuxserver/transmission:latest", Stack: "media-stack",
				Status: ContainerStatusRunning, CPU: 9, Memory: 21, MemoryText: "620 MB / 3 GB",
				Ports:     []string{"9091:9091/tcp", "51413:51413/tcp", "51413:51413/udp"},
				Mounts:    []string{"/volume1/downloads:/downloads", "/volume1/docker/media/transmission:/config"},
				Env:       []string{"USER=family", "PEERPORT=51413"},
				Limit:     ResourceLimit{CPU: 3, MemoryMB: 3072},
				Isolation: "下载目录写入 · 应用配置隔离",
			},
			{
				ID: "ollama", Name: "ollama-local", Image: "ollama/ollama:0.5", Stack: "home-ai",
				Status: ContainerStatusRunning, CPU: 36, Memory: 68, MemoryText: "5.4 GB / 8 GB",
				Ports:  []string{"11434:11434/tcp"},
				Mounts: []string{"/volume1/ai/models:/root/.ollama", "/volume1/docker/ai/ollama:/cache"},
				Env:    []string{"OLLAMA_KEEP_ALIVE=30m", "MODEL_POLICY=local-first"},
				Limit:  ResourceLimit{CPU: 6, MemoryMB: 8192}, Restarts: 2,
				Isolation: "模型目录白名单 · 禁止外发敏感数据",
			},
			{
				ID: "gateway", Name: "caddy-gateway", Image: "caddy:2.8", Stack: "edge-gateway",
				Status: ContainerStatusStopped, CPU: 0, Memory: 0, MemoryText: "0 MB / 512 MB",
				Ports:  []string{"80:80/tcp", "443:443/tcp"},
				Mounts: []string{"/volume1/docker/gateway/Caddyfile:/etc/caddy/Caddyfile:ro"},
				Env:    []string{"ACME_AGREE=true", "TRUSTED_PROXIES=lan"},
				Limit:  ResourceLimit{CPU: 1, MemoryMB: 512}, Restarts: 4,
				Isolation: "DMZ 网络 · 只读反代配置",
			},
		},
		logs:        make(map[string][]ContainerLog),
		pulls:       make(map[string]ImagePullStatus),
		pullCancels: make(map[string]context.CancelFunc),
	}
	service.seedLogs()
	return service
}

// AttachTaskRunner wires the shared task runtime so image pulls are mirrored
// into the central task ledger (kind "docker.pull") and can be cooperatively
// canceled. A nil runner preserves the legacy standalone-pull behavior.
func (s *DevService) AttachTaskRunner(m *tasks.Manager) {
	if m == nil {
		return
	}
	s.mu.Lock()
	s.runner = m
	if s.pullCancels == nil {
		s.pullCancels = make(map[string]context.CancelFunc)
	}
	s.mu.Unlock()
	m.RegisterCanceler(pullTaskKind, s.cancelPull)
}

// cancelPull aborts a running adopted pull identified by its central task id.
// The pull goroutine, ending on a canceled context, settles the task Canceled.
func (s *DevService) cancelPull(centralID string) error {
	s.mu.Lock()
	cancel := s.pullCancels[centralID]
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	return nil
}

func NewDevServiceWithStateDir(stateDir string) (*DevService, error) {
	service := NewDevService()
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "docker.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Containers) > 0 {
		service.stacks = cloneStacks(persisted.Stacks)
		service.containers = cloneContainers(persisted.Containers)
		service.logs = cloneLogMap(persisted.Logs)
		service.logSeq = persisted.LogSeq
		if service.logs == nil {
			service.logs = make(map[string][]ContainerLog)
		}
	}
	return service, nil
}

func (s *DevService) Stacks(ctx context.Context) ([]ComposeStack, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		if stacks, err := liveStacks(ctx); err == nil {
			return stacks, nil
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneStacks(s.stacks), nil
}

func (s *DevService) Containers(ctx context.Context) ([]Container, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		if containers, err := liveContainers(ctx); err == nil {
			return containers, nil
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneContainers(s.containers), nil
}

func (s *DevService) Images(ctx context.Context) ([]Image, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		return liveImages(ctx)
	}
	return []Image{}, nil
}

func (s *DevService) Volumes(ctx context.Context) ([]Volume, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		return liveVolumes(ctx)
	}
	return []Volume{}, nil
}

func (s *DevService) Networks(ctx context.Context) ([]Network, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		return liveNetworks(ctx)
	}
	return []Network{}, nil
}

func (s *DevService) SearchImages(ctx context.Context, query string) ([]ImageSearchResult, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if !dockerCLIAvailable() {
		return nil, fmt.Errorf("docker CLI is not installed")
	}
	return liveSearchImages(ctx, query)
}

func (s *DevService) PullImage(ctx context.Context, request PullImageRequest) error {
	_, err := s.StartPullImage(ctx, request)
	return err
}

func (s *DevService) StartPullImage(ctx context.Context, request PullImageRequest) (ImagePullStatus, error) {
	if err := ctx.Err(); err != nil {
		return ImagePullStatus{}, err
	}
	if !dockerCLIAvailable() {
		return ImagePullStatus{}, fmt.Errorf("docker CLI is not installed")
	}
	image := strings.TrimSpace(request.Image)
	if image == "" {
		return ImagePullStatus{}, fmt.Errorf("docker image is required")
	}
	now := time.Now()
	taskID := image
	s.mu.Lock()
	if s.pulls == nil {
		s.pulls = make(map[string]ImagePullStatus)
	}
	if existing, ok := s.pulls[taskID]; ok && (existing.Status == "queued" || existing.Status == "running") {
		s.mu.Unlock()
		return existing, nil
	}
	task := ImagePullStatus{
		ID:        taskID,
		Image:     image,
		Status:    "queued",
		Message:   "等待拉取",
		Progress:  1,
		StartedAt: now,
		UpdatedAt: now,
	}
	s.pulls[taskID] = task
	runner := s.runner
	s.mu.Unlock()

	// Mirror the pull into the central task runtime when wired. The pull is an
	// externally-driven (adopted) task: progress is pushed from the callback and
	// the central record is settled when the pull reaches a terminal state.
	var centralID string
	if runner != nil {
		if adopted, err := runner.Adopt(pullTaskKind, map[string]any{"image": image}); err == nil {
			centralID = adopted.ID
		}
	}

	// Derive a cancelable context (still bounded by the 45m timeout) so the
	// central Cancel can abort an in-flight pull.
	pullCtx, cancel := context.WithTimeout(context.Background(), 45*time.Minute)
	if centralID != "" {
		s.mu.Lock()
		if s.pullCancels == nil {
			s.pullCancels = make(map[string]context.CancelFunc)
		}
		s.pullCancels[centralID] = cancel
		s.mu.Unlock()
	}

	go func() {
		defer cancel()
		defer func() {
			if centralID != "" {
				s.mu.Lock()
				delete(s.pullCancels, centralID)
				s.mu.Unlock()
			}
		}()
		livePullImageProgress(pullCtx, image, func(next ImagePullStatus) {
			next.ID = taskID
			next.Image = image
			s.updateImagePull(next)
			// When the context is canceled the pull surfaces as a "failed"
			// callback; route it to a canceled settle instead of failed.
			if centralID != "" && pullCtx.Err() != nil {
				runner.Settle(centralID, tasks.StatusCanceled, nil, "")
				return
			}
			s.mirrorPullToCentral(runner, centralID, next)
		})
		// Belt-and-suspenders: if the loop exited on cancellation without a
		// terminal callback, settle the central task as canceled so it doesn't
		// linger in "running".
		if centralID != "" && pullCtx.Err() != nil {
			runner.Settle(centralID, tasks.StatusCanceled, nil, "")
		}
	}()
	return task, nil
}

// mirrorPullToCentral pushes an ImagePullStatus update onto the adopted central
// task: progress while running, terminal Settle on success/failure.
func (s *DevService) mirrorPullToCentral(runner *tasks.Manager, centralID string, next ImagePullStatus) {
	if runner == nil || centralID == "" {
		return
	}
	switch next.Status {
	case "succeeded", "success", "completed":
		result, _ := json.Marshal(map[string]string{"image": next.Image})
		runner.Settle(centralID, tasks.StatusSucceeded, result, "")
	case "failed", "error":
		msg := next.Error
		if msg == "" {
			msg = next.Message
		}
		runner.Settle(centralID, tasks.StatusFailed, nil, msg)
	default:
		runner.Update(centralID, next.Progress, next.Message)
	}
}

func (s *DevService) ImagePulls(ctx context.Context) ([]ImagePullStatus, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	pulls := make([]ImagePullStatus, 0, len(s.pulls))
	for _, pull := range s.pulls {
		pulls = append(pulls, pull)
	}
	sort.Slice(pulls, func(i, j int) bool {
		return pulls[i].UpdatedAt.After(pulls[j].UpdatedAt)
	})
	return pulls, nil
}

func (s *DevService) updateImagePull(next ImagePullStatus) {
	if next.ID == "" {
		next.ID = next.Image
	}
	next.UpdatedAt = time.Now()
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.pulls == nil {
		s.pulls = make(map[string]ImagePullStatus)
	}
	if previous, ok := s.pulls[next.ID]; ok {
		if next.StartedAt.IsZero() {
			next.StartedAt = previous.StartedAt
		}
		if next.Progress < previous.Progress && next.Status == "running" {
			next.Progress = previous.Progress
		}
	}
	s.pulls[next.ID] = next
}

func (s *DevService) RemoveImage(ctx context.Context, request RemoveImageRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveRemoveImage(ctx, request)
}

func (s *DevService) CreateVolume(ctx context.Context, request CreateVolumeRequest) (Volume, error) {
	if err := ctx.Err(); err != nil {
		return Volume{}, err
	}
	if !dockerCLIAvailable() {
		return Volume{}, fmt.Errorf("docker CLI is not installed")
	}
	return liveCreateVolume(ctx, request)
}

func (s *DevService) RemoveVolume(ctx context.Context, request RemoveVolumeRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveRemoveVolume(ctx, request)
}

func (s *DevService) CreateNetwork(ctx context.Context, request CreateNetworkRequest) (Network, error) {
	if err := ctx.Err(); err != nil {
		return Network{}, err
	}
	if !dockerCLIAvailable() {
		return Network{}, fmt.Errorf("docker CLI is not installed")
	}
	return liveCreateNetwork(ctx, request)
}

func (s *DevService) RemoveNetwork(ctx context.Context, request RemoveNetworkRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveRemoveNetwork(ctx, request)
}

func (s *DevService) ConnectNetwork(ctx context.Context, request NetworkConnectRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveConnectNetwork(ctx, request)
}

func (s *DevService) DisconnectNetwork(ctx context.Context, request NetworkDisconnectRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveDisconnectNetwork(ctx, request)
}

func (s *DevService) CreateContainer(ctx context.Context, request CreateContainerRequest) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if !dockerCLIAvailable() {
		return Container{}, fmt.Errorf("docker CLI is not installed")
	}
	return liveCreateContainer(ctx, request)
}

func (s *DevService) RemoveContainer(ctx context.Context, containerID string, request RemoveContainerRequest) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !dockerCLIAvailable() {
		return fmt.Errorf("docker CLI is not installed")
	}
	return liveRemoveContainer(ctx, containerID, request)
}

func (s *DevService) Exec(ctx context.Context, containerID string, request ContainerExecRequest) (ContainerExecResult, error) {
	if err := ctx.Err(); err != nil {
		return ContainerExecResult{}, err
	}
	if !dockerCLIAvailable() {
		return ContainerExecResult{}, fmt.Errorf("docker CLI is not installed")
	}
	return liveExecContainer(ctx, containerID, request)
}

func (s *DevService) Logs(ctx context.Context, containerID string, tail int) ([]ContainerLog, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if dockerCLIAvailable() {
		if logs, err := liveLogs(ctx, containerID, tail); err == nil {
			return logs, nil
		}
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	if _, ok := s.findContainerLocked(containerID); !ok {
		return nil, fmt.Errorf("docker container not found: %s", containerID)
	}
	logs := s.logs[containerID]
	if tail > 0 && tail < len(logs) {
		logs = logs[:tail]
	}
	return cloneLogs(logs), nil
}

func (s *DevService) Start(ctx context.Context, containerID string) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if dockerCLIAvailable() {
		if container, err := liveContainerAction(ctx, containerID, "start"); err == nil {
			return container, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findContainerLocked(containerID)
	if !ok {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	container := &s.containers[index]
	container.Status = ContainerStatusRunning
	container.CPU = maxInt(container.CPU, 12)
	container.Memory = maxInt(container.Memory, 18)
	container.MemoryText = runtimeMemoryText(*container)
	s.prependLogLocked(container.ID, "启动操作已由 Web 桌面触发")
	return cloneContainer(*container), s.saveLocked()
}

func (s *DevService) Stop(ctx context.Context, containerID string) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if dockerCLIAvailable() {
		if container, err := liveContainerAction(ctx, containerID, "stop"); err == nil {
			return container, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findContainerLocked(containerID)
	if !ok {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	container := &s.containers[index]
	container.Status = ContainerStatusStopped
	container.CPU = 0
	container.Memory = 0
	container.MemoryText = fmt.Sprintf("0 MB / %d MB", container.Limit.MemoryMB)
	s.prependLogLocked(container.ID, "停止操作已由 Web 桌面触发")
	return cloneContainer(*container), s.saveLocked()
}

func (s *DevService) Restart(ctx context.Context, containerID string) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if dockerCLIAvailable() {
		if container, err := liveContainerAction(ctx, containerID, "restart"); err == nil {
			return container, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findContainerLocked(containerID)
	if !ok {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	container := &s.containers[index]
	container.Status = ContainerStatusRestarting
	container.Restarts++
	container.CPU = 4
	container.Memory = maxInt(container.Memory, 8)
	container.MemoryText = runtimeMemoryText(*container)
	s.prependLogLocked(container.ID, fmt.Sprintf("第 %d 次重启：正在重新创建容器", container.Restarts))
	return cloneContainer(*container), s.saveLocked()
}

func (s *DevService) CompleteRestart(ctx context.Context, containerID string) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if dockerCLIAvailable() {
		if container, err := liveFindContainer(ctx, containerID); err == nil {
			return container, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findContainerLocked(containerID)
	if !ok {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	container := &s.containers[index]
	container.Status = ContainerStatusRunning
	container.CPU = maxInt(container.CPU+8, 10)
	container.Memory = maxInt(container.Memory+6, 18)
	container.MemoryText = runtimeMemoryText(*container)
	s.prependLogLocked(container.ID, "健康检查通过，容器已恢复服务")
	return cloneContainer(*container), s.saveLocked()
}

func (s *DevService) UpdateLimits(ctx context.Context, containerID string, limit ResourceLimit) (Container, error) {
	if err := ctx.Err(); err != nil {
		return Container{}, err
	}
	if err := validateLimit(limit); err != nil {
		return Container{}, err
	}
	if dockerCLIAvailable() {
		if container, err := liveUpdateLimits(ctx, containerID, limit); err == nil {
			return container, nil
		}
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findContainerLocked(containerID)
	if !ok {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	container := &s.containers[index]
	container.Limit = limit
	container.MemoryText = runtimeMemoryText(*container)
	s.prependLogLocked(container.ID, fmt.Sprintf("资源限制调整为 %d CPU / %d MB", limit.CPU, limit.MemoryMB))
	return cloneContainer(*container), s.saveLocked()
}

func dockerCLIAvailable() bool {
	_, err := exec.LookPath("docker")
	return err == nil
}

func (s *DevService) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Stacks:     cloneStacks(s.stacks),
		Containers: cloneContainers(s.containers),
		Logs:       cloneLogMap(s.logs),
		LogSeq:     s.logSeq,
	})
}

func (s *DevService) seedLogs() {
	seed := map[string][]string{
		"media-server": {"媒体库扫描完成", "硬件转码队列 2 个任务", "端口 8096 已绑定到局域网"},
		"transmission": {"订阅下载队列同步完成", "上传限速 4 MB/s", "DHT 节点已连接"},
		"ollama":       {"qwen2.5:7b 已加载", "向量任务等待 GPU 调度", "本地推理端口 11434 正常"},
		"gateway":      {"用户手动停止服务", "证书续期任务暂停", "端口 443 已释放"},
	}
	for _, container := range s.containers {
		for _, message := range seed[container.ID] {
			s.appendLogLocked(container.ID, message)
		}
	}
}

func (s *DevService) findContainerLocked(containerID string) (int, bool) {
	for index, container := range s.containers {
		if container.ID == containerID {
			return index, true
		}
	}
	return -1, false
}

func (s *DevService) appendLogLocked(containerID string, message string) {
	s.logSeq++
	s.logs[containerID] = append(s.logs[containerID], ContainerLog{
		ID:          fmt.Sprintf("docker-log-%03d", s.logSeq),
		ContainerID: containerID,
		Message:     message,
		Timestamp:   time.Now().UTC(),
	})
}

func (s *DevService) prependLogLocked(containerID string, message string) {
	s.logSeq++
	entry := ContainerLog{
		ID:          fmt.Sprintf("docker-log-%03d", s.logSeq),
		ContainerID: containerID,
		Message:     message,
		Timestamp:   time.Now().UTC(),
	}
	s.logs[containerID] = append([]ContainerLog{entry}, s.logs[containerID]...)
}

func validateLimit(limit ResourceLimit) error {
	if limit.CPU < 1 || limit.CPU > 8 {
		return fmt.Errorf("docker resource limit CPU must be between 1 and 8")
	}
	if limit.MemoryMB < 512 || limit.MemoryMB > 12288 {
		return fmt.Errorf("docker resource limit memory must be between 512 and 12288 MB")
	}
	if limit.MemoryMB%512 != 0 {
		return fmt.Errorf("docker resource limit memory must use 512 MB steps")
	}
	return nil
}

func runtimeMemoryText(container Container) string {
	if container.Status == ContainerStatusStopped {
		return fmt.Sprintf("0 MB / %d MB", container.Limit.MemoryMB)
	}
	usedMB := container.Limit.MemoryMB * maxInt(container.Memory, 1) / 100
	if usedMB >= 1024 {
		return fmt.Sprintf("%.1f GB / %d GB", float64(usedMB)/1024, container.Limit.MemoryMB/1024)
	}
	return fmt.Sprintf("%d MB / %d MB", usedMB, container.Limit.MemoryMB)
}

func cloneStacks(stacks []ComposeStack) []ComposeStack {
	return append([]ComposeStack(nil), stacks...)
}

func cloneContainers(containers []Container) []Container {
	cloned := make([]Container, len(containers))
	for index, container := range containers {
		cloned[index] = cloneContainer(container)
	}
	return cloned
}

func cloneContainer(container Container) Container {
	container.Ports = append([]string(nil), container.Ports...)
	container.Mounts = append([]string(nil), container.Mounts...)
	container.Env = append([]string(nil), container.Env...)
	return container
}

func cloneLogs(logs []ContainerLog) []ContainerLog {
	return append([]ContainerLog(nil), logs...)
}

func cloneLogMap(logs map[string][]ContainerLog) map[string][]ContainerLog {
	out := make(map[string][]ContainerLog, len(logs))
	for key, entries := range logs {
		out[key] = cloneLogs(entries)
	}
	return out
}

func maxInt(a int, b int) int {
	if a > b {
		return a
	}
	return b
}
