package docker

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type dockerInspectContainer struct {
	ID           string           `json:"Id"`
	Name         string           `json:"Name"`
	Config       dockerConfig     `json:"Config"`
	State        dockerState      `json:"State"`
	HostConfig   dockerHostConfig `json:"HostConfig"`
	Mounts       []dockerMount    `json:"Mounts"`
	Network      dockerNetwork    `json:"NetworkSettings"`
	RestartCount int              `json:"RestartCount"`
}

type dockerConfig struct {
	Image  string            `json:"Image"`
	Env    []string          `json:"Env"`
	Labels map[string]string `json:"Labels"`
}

type dockerState struct {
	Status string `json:"Status"`
	Health *struct {
		Status string `json:"Status"`
	} `json:"Health"`
}

type dockerHostConfig struct {
	NanoCPUs int64 `json:"NanoCpus"`
	Memory   int64 `json:"Memory"`
}

type dockerMount struct {
	Source      string `json:"Source"`
	Destination string `json:"Destination"`
	Mode        string `json:"Mode"`
	RW          bool   `json:"RW"`
	Type        string `json:"Type"`
	Name        string `json:"Name"`
}

type dockerNetwork struct {
	Ports    map[string][]dockerPortBinding `json:"Ports"`
	Networks map[string]any                 `json:"Networks"`
}

type dockerPortBinding struct {
	HostIP   string `json:"HostIp"`
	HostPort string `json:"HostPort"`
}

type dockerStatsLine struct {
	ID       string `json:"ID"`
	Name     string `json:"Name"`
	CPUPerc  string `json:"CPUPerc"`
	MemPerc  string `json:"MemPerc"`
	MemUsage string `json:"MemUsage"`
}

func liveStacks(ctx context.Context) ([]ComposeStack, error) {
	containers, err := liveContainers(ctx)
	if err != nil {
		return nil, err
	}
	grouped := make(map[string][]Container)
	for _, container := range containers {
		grouped[container.Stack] = append(grouped[container.Stack], container)
	}
	stacks := make([]ComposeStack, 0, len(grouped))
	for name, items := range grouped {
		ports := collectUnique(items, func(container Container) []string { return container.Ports })
		mounts := collectUnique(items, func(container Container) []string { return container.Mounts })
		running := 0
		for _, item := range items {
			if item.Status == ContainerStatusRunning {
				running++
			}
		}
		stacks = append(stacks, ComposeStack{
			Name:     name,
			Status:   stackStatus(running, len(items)),
			Services: len(items),
			Ports:    shortJoin(ports, "无端口"),
			Volume:   firstMountSource(mounts),
			Network:  stackNetwork(items),
		})
	}
	sort.Slice(stacks, func(i, j int) bool {
		if stacks[i].Name == "standalone" {
			return false
		}
		if stacks[j].Name == "standalone" {
			return true
		}
		return stacks[i].Name < stacks[j].Name
	})
	return stacks, nil
}

func liveContainers(ctx context.Context) ([]Container, error) {
	idsOutput, err := runDocker(ctx, "ps", "-aq")
	if err != nil {
		return nil, err
	}
	ids := strings.Fields(strings.TrimSpace(idsOutput))
	if len(ids) == 0 {
		return []Container{}, nil
	}
	args := append([]string{"inspect"}, ids...)
	inspectOutput, err := runDocker(ctx, args...)
	if err != nil {
		return nil, err
	}
	var inspected []dockerInspectContainer
	if err := json.Unmarshal([]byte(inspectOutput), &inspected); err != nil {
		return nil, fmt.Errorf("decode docker inspect: %w", err)
	}
	stats := liveStats(ctx)
	containers := make([]Container, 0, len(inspected))
	for _, item := range inspected {
		containers = append(containers, mapInspectContainer(item, stats))
	}
	sort.Slice(containers, func(i, j int) bool {
		if containers[i].Stack == containers[j].Stack {
			return containers[i].Name < containers[j].Name
		}
		return containers[i].Stack < containers[j].Stack
	})
	return containers, nil
}

func liveFindContainer(ctx context.Context, containerID string) (Container, error) {
	output, err := runDocker(ctx, "inspect", containerID)
	if err != nil {
		return Container{}, err
	}
	var inspected []dockerInspectContainer
	if err := json.Unmarshal([]byte(output), &inspected); err != nil {
		return Container{}, fmt.Errorf("decode docker inspect: %w", err)
	}
	if len(inspected) == 0 {
		return Container{}, fmt.Errorf("docker container not found: %s", containerID)
	}
	return mapInspectContainer(inspected[0], liveStats(ctx)), nil
}

func liveLogs(ctx context.Context, containerID string, tail int) ([]ContainerLog, error) {
	if tail <= 0 {
		tail = 100
	}
	output, err := runDocker(ctx, "logs", "--tail", strconv.Itoa(tail), "--timestamps", containerID)
	if err != nil {
		return nil, err
	}
	lines := strings.Split(strings.TrimSpace(output), "\n")
	logs := make([]ContainerLog, 0, len(lines))
	now := time.Now().UTC()
	for index, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		message := line
		timestamp := now
		if len(parts) == 2 {
			if parsed, err := time.Parse(time.RFC3339Nano, parts[0]); err == nil {
				timestamp = parsed
				message = parts[1]
			}
		}
		logs = append(logs, ContainerLog{
			ID:          fmt.Sprintf("docker-live-%03d", index+1),
			ContainerID: containerID,
			Message:     message,
			Timestamp:   timestamp,
		})
	}
	return logs, nil
}

func liveContainerAction(ctx context.Context, containerID string, action string) (Container, error) {
	if _, err := runDocker(ctx, action, containerID); err != nil {
		return Container{}, err
	}
	return liveFindContainer(ctx, containerID)
}

func liveUpdateLimits(ctx context.Context, containerID string, limit ResourceLimit) (Container, error) {
	args := []string{"update", "--cpus", strconv.Itoa(limit.CPU), "--memory", fmt.Sprintf("%dm", limit.MemoryMB), containerID}
	if _, err := runDocker(ctx, args...); err != nil {
		return Container{}, err
	}
	return liveFindContainer(ctx, containerID)
}

func liveStats(ctx context.Context) map[string]dockerStatsLine {
	output, err := runDocker(ctx, "stats", "--no-stream", "--format", "{{json .}}")
	if err != nil {
		return map[string]dockerStatsLine{}
	}
	stats := map[string]dockerStatsLine{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var stat dockerStatsLine
		if err := json.Unmarshal([]byte(line), &stat); err == nil {
			stats[stat.ID] = stat
			stats[stat.Name] = stat
		}
	}
	return stats
}

func mapInspectContainer(item dockerInspectContainer, stats map[string]dockerStatsLine) Container {
	name := strings.TrimPrefix(item.Name, "/")
	stack := item.Config.Labels["com.docker.compose.project"]
	if stack == "" {
		stack = "standalone"
	}
	status := mapDockerStatus(item.State.Status)
	stat := stats[item.ID[:minInt(len(item.ID), 12)]]
	if stat.Name == "" {
		stat = stats[name]
	}
	cpu := parsePercent(stat.CPUPerc)
	memory := parsePercent(stat.MemPerc)
	limit := ResourceLimit{
		CPU:      limitCPU(item.HostConfig.NanoCPUs),
		MemoryMB: limitMemoryMB(item.HostConfig.Memory),
	}
	return Container{
		ID:         item.ID[:minInt(len(item.ID), 12)],
		Name:       name,
		Image:      firstNonEmpty(item.Config.Image, item.Config.Labels["com.docker.compose.image"]),
		Stack:      stack,
		Status:     status,
		CPU:        cpu,
		Memory:     memory,
		MemoryText: firstNonEmpty(stat.MemUsage, runtimeMemoryText(Container{Status: status, Memory: memory, Limit: limit})),
		Ports:      formatPorts(item.Network.Ports),
		Mounts:     formatMounts(item.Mounts),
		Env:        safeEnv(item.Config.Env),
		Limit:      limit,
		Restarts:   item.RestartCount,
		Isolation:  isolationSummary(item),
	}
}

func runDocker(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "docker", args...)
	var stderr bytes.Buffer
	command.Stderr = &stderr
	output, err := command.Output()
	if err != nil {
		return "", fmt.Errorf("docker %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return string(output), nil
}

func mapDockerStatus(status string) ContainerStatus {
	switch strings.ToLower(status) {
	case "running":
		return ContainerStatusRunning
	case "restarting":
		return ContainerStatusRestarting
	default:
		return ContainerStatusStopped
	}
}

func formatPorts(ports map[string][]dockerPortBinding) []string {
	out := make([]string, 0, len(ports))
	for containerPort, bindings := range ports {
		if len(bindings) == 0 {
			out = append(out, containerPort)
			continue
		}
		for _, binding := range bindings {
			host := binding.HostPort
			if binding.HostIP != "" && binding.HostIP != "0.0.0.0" && binding.HostIP != "::" {
				host = binding.HostIP + ":" + host
			}
			out = append(out, host+":"+containerPort)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{"未暴露端口"}
	}
	return out
}

func formatMounts(mounts []dockerMount) []string {
	out := make([]string, 0, len(mounts))
	for _, mount := range mounts {
		source := firstNonEmpty(mount.Source, mount.Name, mount.Type)
		mode := "ro"
		if mount.RW {
			mode = "rw"
		}
		if mount.Mode != "" {
			mode = mount.Mode
		}
		out = append(out, fmt.Sprintf("%s:%s:%s", source, mount.Destination, mode))
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{"无挂载"}
	}
	return out
}

func safeEnv(env []string) []string {
	out := make([]string, 0, len(env))
	for _, item := range env {
		key, value, ok := strings.Cut(item, "=")
		if !ok {
			out = append(out, item)
			continue
		}
		if isSecretKey(key) {
			out = append(out, key+"=******")
		} else {
			out = append(out, key+"="+value)
		}
	}
	sort.Strings(out)
	if len(out) == 0 {
		return []string{"无环境变量"}
	}
	return out
}

func isSecretKey(key string) bool {
	key = strings.ToLower(key)
	return strings.Contains(key, "password") || strings.Contains(key, "secret") || strings.Contains(key, "token") || strings.Contains(key, "key")
}

func limitCPU(nano int64) int {
	if nano <= 0 {
		return 1
	}
	cpu := int(nano / 1_000_000_000)
	if cpu < 1 {
		return 1
	}
	return cpu
}

func limitMemoryMB(bytes int64) int {
	if bytes <= 0 {
		return 512
	}
	mb := int(bytes / 1024 / 1024)
	if mb < 512 {
		return 512
	}
	return mb
}

func parsePercent(value string) int {
	value = strings.TrimSpace(strings.TrimSuffix(value, "%"))
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return 0
	}
	return int(parsed + 0.5)
}

func stackStatus(running int, total int) string {
	switch {
	case total == 0:
		return "空"
	case running == total:
		return "健康"
	case running == 0:
		return "已停止"
	default:
		return fmt.Sprintf("%d/%d 运行", running, total)
	}
}

func stackNetwork(containers []Container) string {
	for _, container := range containers {
		if container.Stack != "standalone" {
			return container.Stack
		}
	}
	return "bridge"
}

func collectUnique(containers []Container, picker func(Container) []string) []string {
	seen := map[string]bool{}
	for _, container := range containers {
		for _, item := range picker(container) {
			if item != "" && item != "无挂载" && item != "未暴露端口" {
				seen[item] = true
			}
		}
	}
	out := make([]string, 0, len(seen))
	for item := range seen {
		out = append(out, item)
	}
	sort.Strings(out)
	return out
}

func shortJoin(items []string, empty string) string {
	if len(items) == 0 {
		return empty
	}
	if len(items) > 3 {
		return strings.Join(items[:3], " · ") + fmt.Sprintf(" · +%d", len(items)-3)
	}
	return strings.Join(items, " · ")
}

func firstMountSource(mounts []string) string {
	if len(mounts) == 0 {
		return "无挂载"
	}
	source, _, ok := strings.Cut(mounts[0], ":")
	if !ok {
		return mounts[0]
	}
	return source
}

func isolationSummary(item dockerInspectContainer) string {
	if item.State.Health != nil && item.State.Health.Status != "" {
		return "健康检查 " + item.State.Health.Status
	}
	if item.HostConfig.Memory > 0 || item.HostConfig.NanoCPUs > 0 {
		return "已配置资源限制"
	}
	return "默认隔离策略"
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
