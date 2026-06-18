package docker

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"sync"
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

type dockerImageLine struct {
	ID           string `json:"ID"`
	Repository   string `json:"Repository"`
	Tag          string `json:"Tag"`
	CreatedSince string `json:"CreatedSince"`
	CreatedAt    string `json:"CreatedAt"`
	Size         string `json:"Size"`
}

type dockerVolumeLine struct {
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Scope      string `json:"Scope"`
	Mountpoint string `json:"Mountpoint"`
}

type dockerNetworkLine struct {
	ID     string `json:"ID"`
	Name   string `json:"Name"`
	Driver string `json:"Driver"`
	Scope  string `json:"Scope"`
}

type dockerNetworkInspect struct {
	ID         string `json:"Id"`
	Name       string `json:"Name"`
	Driver     string `json:"Driver"`
	Scope      string `json:"Scope"`
	Internal   bool   `json:"Internal"`
	Attachable bool   `json:"Attachable"`
	IPAM       struct {
		Config []struct {
			Subnet  string `json:"Subnet"`
			Gateway string `json:"Gateway"`
		} `json:"Config"`
	} `json:"IPAM"`
	Containers map[string]struct {
		Name string `json:"Name"`
	} `json:"Containers"`
}

type dockerSearchLine struct {
	Name        string `json:"Name"`
	Description string `json:"Description"`
	Stars       string `json:"StarCount"`
	Official    string `json:"IsOfficial"`
	Automated   string `json:"IsAutomated"`
}

var pullSizePattern = regexp.MustCompile(`(?i)([0-9]+(?:\.[0-9]+)?\s*(?:[KMGT]?I?B|B))\s*/\s*([0-9]+(?:\.[0-9]+)?\s*(?:[KMGT]?I?B|B))`)

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

func liveImages(ctx context.Context) ([]Image, error) {
	output, err := runDocker(ctx, "images", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	images := []Image{}
	for _, line := range nonEmptyLines(output) {
		var item dockerImageLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		images = append(images, Image{
			ID:         item.ID,
			Repository: item.Repository,
			Tag:        item.Tag,
			Size:       item.Size,
			Created:    firstNonEmpty(item.CreatedSince, item.CreatedAt),
			IconURL:    dockerHubIconURL(item.Repository),
		})
	}
	return images, nil
}

func liveVolumes(ctx context.Context) ([]Volume, error) {
	output, err := runDocker(ctx, "volume", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	volumes := []Volume{}
	for _, line := range nonEmptyLines(output) {
		var item dockerVolumeLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		volumes = append(volumes, Volume{
			Name:       item.Name,
			Driver:     item.Driver,
			Scope:      item.Scope,
			Mountpoint: item.Mountpoint,
		})
	}
	return volumes, nil
}

func liveNetworks(ctx context.Context) ([]Network, error) {
	output, err := runDocker(ctx, "network", "ls", "--format", "{{json .}}")
	if err != nil {
		return nil, err
	}
	networks := []Network{}
	for _, line := range nonEmptyLines(output) {
		var item dockerNetworkLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		networks = append(networks, Network{
			ID:     item.ID,
			Name:   item.Name,
			Driver: item.Driver,
			Scope:  item.Scope,
		})
	}
	if len(networks) > 0 {
		inspectArgs := []string{"network", "inspect"}
		for _, network := range networks {
			inspectArgs = append(inspectArgs, network.Name)
		}
		if inspected, err := runDocker(ctx, inspectArgs...); err == nil {
			var details []dockerNetworkInspect
			if json.Unmarshal([]byte(inspected), &details) == nil {
				byName := make(map[string]dockerNetworkInspect, len(details))
				for _, detail := range details {
					byName[detail.Name] = detail
				}
				for index := range networks {
					detail, ok := byName[networks[index].Name]
					if !ok {
						continue
					}
					networks[index].Internal = detail.Internal
					networks[index].Attachable = detail.Attachable
					if len(detail.IPAM.Config) > 0 {
						networks[index].Subnet = detail.IPAM.Config[0].Subnet
						networks[index].Gateway = detail.IPAM.Config[0].Gateway
					}
					names := make([]string, 0, len(detail.Containers))
					for _, container := range detail.Containers {
						if container.Name != "" {
							names = append(names, container.Name)
						}
					}
					sort.Strings(names)
					networks[index].Containers = names
				}
			}
		}
	}
	return networks, nil
}

func liveSearchImages(ctx context.Context, query string) ([]ImageSearchResult, error) {
	query = strings.TrimSpace(query)
	if query == "" {
		return []ImageSearchResult{}, nil
	}
	output, err := runDocker(ctx, "search", "--limit", "20", "--format", "{{json .}}", query)
	if err != nil {
		return nil, err
	}
	results := []ImageSearchResult{}
	for _, line := range nonEmptyLines(output) {
		var item dockerSearchLine
		if err := json.Unmarshal([]byte(line), &item); err != nil {
			continue
		}
		stars, _ := strconv.Atoi(strings.TrimSpace(item.Stars))
		results = append(results, ImageSearchResult{
			Name:        item.Name,
			Description: item.Description,
			Stars:       stars,
			Official:    dockerBool(item.Official),
			Automated:   dockerBool(item.Automated),
			IconURL:     dockerHubIconURL(item.Name),
		})
	}
	return results, nil
}

func dockerHubIconURL(image string) string {
	image = strings.TrimSpace(image)
	if image == "" || image == "<none>" {
		return ""
	}
	image = strings.TrimPrefix(image, "docker.io/")
	image = strings.TrimPrefix(image, "index.docker.io/")
	slashIndex := strings.LastIndex(image, "/")
	colonIndex := strings.LastIndex(image, ":")
	if colonIndex > slashIndex {
		image = image[:colonIndex]
	}
	if atIndex := strings.Index(image, "@"); atIndex >= 0 {
		image = image[:atIndex]
	}
	parts := strings.Split(image, "/")
	if len(parts) == 0 || parts[0] == "" {
		return ""
	}
	if strings.Contains(parts[0], ".") || strings.Contains(parts[0], ":") || parts[0] == "localhost" {
		return ""
	}
	repo := image
	if len(parts) == 1 {
		repo = "library/" + parts[0]
	}
	if strings.TrimSpace(repo) == "" {
		return ""
	}
	return "https://hub.docker.com/api/media/repos_logo/v1/" + strings.ReplaceAll(repo, "/", "%2F")
}

func livePullImage(ctx context.Context, image string) error {
	image = strings.TrimSpace(image)
	if image == "" {
		return fmt.Errorf("docker image is required")
	}
	_, err := runDockerCombined(ctx, "pull", image)
	return err
}

func livePullImageProgress(ctx context.Context, image string, emit func(ImagePullStatus)) error {
	image = strings.TrimSpace(image)
	if image == "" {
		err := fmt.Errorf("docker image is required")
		emit(ImagePullStatus{Image: image, Status: "failed", Message: "镜像名为空", Error: err.Error(), Progress: 0})
		return err
	}
	startedAt := time.Now()
	emit(ImagePullStatus{
		Image:     image,
		Status:    "running",
		Message:   "正在连接镜像仓库",
		Progress:  2,
		StartedAt: startedAt,
		UpdatedAt: startedAt,
	})

	command := exec.CommandContext(ctx, "docker", "pull", image)
	stdout, err := command.StdoutPipe()
	if err != nil {
		emitPullFailure(image, startedAt, err, emit)
		return err
	}
	stderr, err := command.StderrPipe()
	if err != nil {
		emitPullFailure(image, startedAt, err, emit)
		return err
	}
	if err := command.Start(); err != nil {
		emitPullFailure(image, startedAt, err, emit)
		return err
	}

	lines := make(chan string, 64)
	var wg sync.WaitGroup
	scan := func(reader io.Reader) {
		defer wg.Done()
		scanner := bufio.NewScanner(reader)
		scanner.Buffer(make([]byte, 0, 64*1024), 512*1024)
		scanner.Split(scanPullTokens)
		for scanner.Scan() {
			line := strings.TrimSpace(scanner.Text())
			if line != "" {
				lines <- line
			}
		}
	}
	wg.Add(2)
	go scan(stdout)
	go scan(stderr)
	go func() {
		wg.Wait()
		close(lines)
	}()

	tracker := newPullProgressTracker()
	for line := range lines {
		status := tracker.update(line)
		status.Image = image
		status.Status = "running"
		status.StartedAt = startedAt
		status.UpdatedAt = time.Now()
		emit(status)
	}

	if err := command.Wait(); err != nil {
		emitPullFailure(image, startedAt, err, emit)
		return err
	}
	final := tracker.status()
	final.Image = image
	final.Status = "completed"
	final.Message = "拉取完成"
	final.Progress = 100
	final.Speed = "0 B/s"
	final.StartedAt = startedAt
	final.UpdatedAt = time.Now()
	emit(final)
	return nil
}

type pullLayerProgress struct {
	downloaded float64
	total      float64
	complete   bool
}

type pullProgressTracker struct {
	layers         map[string]*pullLayerProgress
	progress       int
	fallback       int
	lastDownloaded float64
	lastAt         time.Time
	downloaded     float64
	total          float64
	speed          string
	message        string
}

func newPullProgressTracker() *pullProgressTracker {
	return &pullProgressTracker{
		layers:   make(map[string]*pullLayerProgress),
		progress: 2,
		fallback: 2,
		lastAt:   time.Now(),
		speed:    "0 B/s",
		message:  "正在拉取",
	}
}

func (t *pullProgressTracker) update(line string) ImagePullStatus {
	t.message = pullMessage(line)
	layerID := pullLayerID(line)
	if layerID != "" {
		layer := t.layers[layerID]
		if layer == nil {
			layer = &pullLayerProgress{}
			t.layers[layerID] = layer
		}
		if strings.Contains(line, "Pull complete") || strings.Contains(line, "Already exists") || strings.Contains(line, "Download complete") {
			layer.complete = true
			if layer.total <= 0 {
				layer.total = maxFloat(layer.downloaded, 1)
			}
			layer.downloaded = layer.total
		}
		if match := pullSizePattern.FindStringSubmatch(line); len(match) == 3 {
			layer.downloaded = parsePullBytes(match[1])
			layer.total = parsePullBytes(match[2])
			if layer.total > 0 && layer.downloaded >= layer.total {
				layer.complete = true
			}
		}
	}
	t.recalculate()
	return t.status()
}

func (t *pullProgressTracker) status() ImagePullStatus {
	return ImagePullStatus{
		Status:     "running",
		Message:    t.message,
		Progress:   t.progress,
		Downloaded: formatPullBytes(t.downloaded),
		Total:      formatPullBytes(t.total),
		Speed:      t.speed,
	}
}

func (t *pullProgressTracker) recalculate() {
	var downloaded float64
	var total float64
	for _, layer := range t.layers {
		layerTotal := layer.total
		layerDownloaded := layer.downloaded
		if layer.complete {
			layerTotal = maxFloat(layerTotal, layerDownloaded, 1)
			layerDownloaded = layerTotal
		}
		downloaded += layerDownloaded
		total += layerTotal
	}
	now := time.Now()
	if total > 0 {
		t.downloaded = downloaded
		t.total = total
		ratio := downloaded / total
		t.progress = minInt(98, maxInt(t.progress, 4+int(ratio*94)))
		if elapsed := now.Sub(t.lastAt).Seconds(); elapsed >= 0.5 {
			speed := (downloaded - t.lastDownloaded) / elapsed
			if speed >= 0 {
				t.speed = formatPullBytes(speed) + "/s"
			}
			t.lastDownloaded = downloaded
			t.lastAt = now
		}
		return
	}
	t.fallback = minInt(88, t.fallback+3)
	t.progress = maxInt(t.progress, t.fallback)
}

func scanPullTokens(data []byte, atEOF bool) (advance int, token []byte, err error) {
	for i, b := range data {
		if b == '\n' || b == '\r' {
			return i + 1, bytes.TrimSpace(data[:i]), nil
		}
	}
	if atEOF && len(data) > 0 {
		return len(data), bytes.TrimSpace(data), nil
	}
	return 0, nil, nil
}

func emitPullFailure(image string, startedAt time.Time, err error, emit func(ImagePullStatus)) {
	now := time.Now()
	emit(ImagePullStatus{
		Image:     image,
		Status:    "failed",
		Message:   "拉取失败",
		Progress:  0,
		Error:     strings.TrimSpace(err.Error()),
		StartedAt: startedAt,
		UpdatedAt: now,
	})
}

func pullLayerID(line string) string {
	index := strings.Index(line, ":")
	if index <= 0 {
		return ""
	}
	prefix := strings.TrimSpace(line[:index])
	if strings.ContainsAny(prefix, " \t") {
		return ""
	}
	if len(prefix) < 8 || len(prefix) > 24 {
		return ""
	}
	return prefix
}

func pullMessage(line string) string {
	line = strings.TrimSpace(line)
	if line == "" {
		return "正在拉取"
	}
	if strings.HasPrefix(line, "Status:") {
		return strings.TrimSpace(strings.TrimPrefix(line, "Status:"))
	}
	if strings.HasPrefix(line, "Digest:") {
		return "校验镜像摘要"
	}
	if strings.Contains(line, "Pulling fs layer") {
		return "正在解析镜像层"
	}
	if strings.Contains(line, "Downloading") {
		return "正在下载镜像层"
	}
	if strings.Contains(line, "Extracting") {
		return "正在解压镜像层"
	}
	if strings.Contains(line, "Pull complete") {
		return "镜像层下载完成"
	}
	return line
}

func parsePullBytes(value string) float64 {
	cleaned := strings.ToUpper(strings.ReplaceAll(strings.TrimSpace(value), " ", ""))
	units := []struct {
		suffix string
		scale  float64
	}{
		{"TIB", 1024 * 1024 * 1024 * 1024},
		{"TB", 1000 * 1000 * 1000 * 1000},
		{"GIB", 1024 * 1024 * 1024},
		{"GB", 1000 * 1000 * 1000},
		{"MIB", 1024 * 1024},
		{"MB", 1000 * 1000},
		{"KIB", 1024},
		{"KB", 1000},
		{"B", 1},
	}
	for _, unit := range units {
		if strings.HasSuffix(cleaned, unit.suffix) {
			number := strings.TrimSuffix(cleaned, unit.suffix)
			parsed, _ := strconv.ParseFloat(number, 64)
			return parsed * unit.scale
		}
	}
	parsed, _ := strconv.ParseFloat(cleaned, 64)
	return parsed
}

func formatPullBytes(value float64) string {
	if value <= 0 {
		return "0 B"
	}
	units := []string{"B", "KB", "MB", "GB", "TB"}
	size := value
	unit := 0
	for size >= 1024 && unit < len(units)-1 {
		size /= 1024
		unit++
	}
	if unit == 0 {
		return fmt.Sprintf("%.0f %s", size, units[unit])
	}
	return fmt.Sprintf("%.1f %s", size, units[unit])
}

func maxFloat(values ...float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, value := range values[1:] {
		if value > max {
			max = value
		}
	}
	return max
}

func liveRemoveImage(ctx context.Context, request RemoveImageRequest) error {
	image := strings.TrimSpace(request.Image)
	if image == "" {
		return fmt.Errorf("docker image is required")
	}
	args := []string{"rmi"}
	if request.Force {
		args = append(args, "-f")
	}
	args = append(args, image)
	_, err := runDocker(ctx, args...)
	return err
}

func liveCreateVolume(ctx context.Context, request CreateVolumeRequest) (Volume, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Volume{}, fmt.Errorf("docker volume name is required")
	}
	args := []string{"volume", "create"}
	if strings.TrimSpace(request.Driver) != "" {
		args = append(args, "--driver", strings.TrimSpace(request.Driver))
	}
	args = append(args, name)
	if _, err := runDocker(ctx, args...); err != nil {
		return Volume{}, err
	}
	volumes, err := liveVolumes(ctx)
	if err != nil {
		return Volume{Name: name, Driver: firstNonEmpty(request.Driver, "local")}, nil
	}
	for _, volume := range volumes {
		if volume.Name == name {
			return volume, nil
		}
	}
	return Volume{Name: name, Driver: firstNonEmpty(request.Driver, "local")}, nil
}

func liveRemoveVolume(ctx context.Context, request RemoveVolumeRequest) error {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return fmt.Errorf("docker volume name is required")
	}
	args := []string{"volume", "rm"}
	if request.Force {
		args = append(args, "-f")
	}
	args = append(args, name)
	_, err := runDocker(ctx, args...)
	return err
}

func liveCreateNetwork(ctx context.Context, request CreateNetworkRequest) (Network, error) {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Network{}, fmt.Errorf("docker network name is required")
	}
	args := []string{"network", "create", "--driver", firstNonEmpty(strings.TrimSpace(request.Driver), "bridge")}
	if strings.TrimSpace(request.Subnet) != "" {
		args = append(args, "--subnet", strings.TrimSpace(request.Subnet))
	}
	if strings.TrimSpace(request.Gateway) != "" {
		args = append(args, "--gateway", strings.TrimSpace(request.Gateway))
	}
	if request.Attachable {
		args = append(args, "--attachable")
	}
	if request.Internal {
		args = append(args, "--internal")
	}
	args = append(args, name)
	if _, err := runDocker(ctx, args...); err != nil {
		return Network{}, err
	}
	networks, err := liveNetworks(ctx)
	if err != nil {
		return Network{Name: name, Driver: firstNonEmpty(strings.TrimSpace(request.Driver), "bridge")}, nil
	}
	for _, network := range networks {
		if network.Name == name {
			return network, nil
		}
	}
	return Network{Name: name, Driver: firstNonEmpty(strings.TrimSpace(request.Driver), "bridge")}, nil
}

func liveRemoveNetwork(ctx context.Context, request RemoveNetworkRequest) error {
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return fmt.Errorf("docker network name is required")
	}
	_, err := runDocker(ctx, "network", "rm", name)
	return err
}

func liveConnectNetwork(ctx context.Context, request NetworkConnectRequest) error {
	network := strings.TrimSpace(request.Network)
	container := strings.TrimSpace(request.Container)
	if network == "" || container == "" {
		return fmt.Errorf("docker network and container are required")
	}
	args := []string{"network", "connect"}
	if strings.TrimSpace(request.Alias) != "" {
		args = append(args, "--alias", strings.TrimSpace(request.Alias))
	}
	args = append(args, network, container)
	_, err := runDocker(ctx, args...)
	return err
}

func liveDisconnectNetwork(ctx context.Context, request NetworkDisconnectRequest) error {
	network := strings.TrimSpace(request.Network)
	container := strings.TrimSpace(request.Container)
	if network == "" || container == "" {
		return fmt.Errorf("docker network and container are required")
	}
	args := []string{"network", "disconnect"}
	if request.Force {
		args = append(args, "-f")
	}
	args = append(args, network, container)
	_, err := runDocker(ctx, args...)
	return err
}

func liveExecContainer(ctx context.Context, containerID string, request ContainerExecRequest) (ContainerExecResult, error) {
	commandText := strings.TrimSpace(request.Command)
	if commandText == "" {
		return ContainerExecResult{}, fmt.Errorf("docker exec command is required")
	}
	command := exec.CommandContext(ctx, "docker", "exec", containerID, "sh", "-lc", commandText)
	output, err := command.CombinedOutput()
	result := ContainerExecResult{
		ExitCode: 0,
		Output:   strings.TrimRight(string(output), "\n"),
	}
	if err != nil {
		if command.ProcessState != nil {
			result.ExitCode = command.ProcessState.ExitCode()
		} else {
			result.ExitCode = 1
		}
		return result, nil
	}
	return result, nil
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
	output, err := runDockerCombined(ctx, "logs", "--tail", strconv.Itoa(tail), "--timestamps", containerID)
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

func liveCreateContainer(ctx context.Context, request CreateContainerRequest) (Container, error) {
	image := strings.TrimSpace(request.Image)
	if image == "" {
		return Container{}, fmt.Errorf("docker image is required")
	}
	args := []string{"run", "-d", "--pull=missing"}
	if strings.TrimSpace(request.Name) != "" {
		args = append(args, "--name", strings.TrimSpace(request.Name))
	}
	if request.AutoRemove {
		args = append(args, "--rm")
	} else if policy := strings.TrimSpace(request.RestartPolicy); policy != "" && policy != "no" {
		args = append(args, "--restart", policy)
	}
	if request.Privileged {
		args = append(args, "--privileged")
	}
	if network := strings.TrimSpace(request.Network); network != "" {
		args = append(args, "--network", network)
	}
	if hostname := strings.TrimSpace(request.Hostname); hostname != "" {
		args = append(args, "--hostname", hostname)
	}
	if user := strings.TrimSpace(request.User); user != "" {
		args = append(args, "--user", user)
	}
	if workdir := strings.TrimSpace(request.WorkingDir); workdir != "" {
		args = append(args, "--workdir", workdir)
	}
	if entrypoint := strings.TrimSpace(request.Entrypoint); entrypoint != "" {
		args = append(args, "--entrypoint", entrypoint)
	}
	if request.LimitCPU > 0 {
		args = append(args, "--cpus", strconv.Itoa(request.LimitCPU))
	}
	if request.LimitMemory > 0 {
		args = append(args, "--memory", fmt.Sprintf("%dm", request.LimitMemory))
	}
	for _, port := range cleanList(request.Ports) {
		args = append(args, "-p", port)
	}
	for _, mount := range cleanList(request.Mounts) {
		args = append(args, "-v", mount)
	}
	for _, env := range cleanList(request.Env) {
		args = append(args, "-e", env)
	}
	for _, label := range cleanList(request.Labels) {
		args = append(args, "--label", label)
	}
	for _, host := range cleanList(request.ExtraHosts) {
		args = append(args, "--add-host", host)
	}
	for _, dns := range cleanList(request.DNS) {
		args = append(args, "--dns", dns)
	}
	args = append(args, image)
	if strings.TrimSpace(request.Command) != "" {
		args = append(args, strings.Fields(request.Command)...)
	}
	output, err := runDocker(ctx, args...)
	if err != nil {
		return Container{}, err
	}
	return liveFindContainer(ctx, strings.TrimSpace(output))
}

func liveRemoveContainer(ctx context.Context, containerID string, request RemoveContainerRequest) error {
	args := []string{"rm"}
	if request.Force {
		args = append(args, "-f")
	}
	if request.RemoveVolumes {
		args = append(args, "-v")
	}
	args = append(args, containerID)
	_, err := runDocker(ctx, args...)
	return err
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

func runDockerCombined(ctx context.Context, args ...string) (string, error) {
	command := exec.CommandContext(ctx, "docker", args...)
	output, err := command.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("docker %s failed: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(string(output)))
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

func cleanList(items []string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		item = strings.TrimSpace(item)
		if item != "" {
			out = append(out, item)
		}
	}
	return out
}

func nonEmptyLines(output string) []string {
	lines := []string{}
	for _, line := range strings.Split(strings.TrimSpace(output), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			lines = append(lines, line)
		}
	}
	return lines
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

func dockerBool(value string) bool {
	value = strings.TrimSpace(strings.ToLower(value))
	return value == "true" || value == "yes" || value == "ok" || value == "[ok]" || value == "1"
}

func minInt(a int, b int) int {
	if a < b {
		return a
	}
	return b
}
