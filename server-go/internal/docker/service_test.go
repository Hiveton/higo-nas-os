package docker

import (
	"context"
	"strings"
	"testing"
)

func TestDevServiceSeedsFrontendAlignedStacksAndContainers(t *testing.T) {
	service := NewDevService()

	stacks, err := service.Stacks(context.Background())
	if err != nil {
		t.Fatalf("stacks: %v", err)
	}
	containers, err := service.Containers(context.Background())
	if err != nil {
		t.Fatalf("containers: %v", err)
	}

	if len(stacks) != 3 {
		t.Fatalf("expected 3 compose stacks, got %d", len(stacks))
	}
	wantStacks := []string{"media-stack", "home-ai", "edge-gateway"}
	for index, want := range wantStacks {
		if stacks[index].Name != want {
			t.Fatalf("stack %d: got %q want %q", index, stacks[index].Name, want)
		}
		if stacks[index].Ports == "" || stacks[index].Volume == "" || stacks[index].Network == "" {
			t.Fatalf("stack %q missing frontend fields: %#v", stacks[index].Name, stacks[index])
		}
	}

	if len(containers) != 4 {
		t.Fatalf("expected 4 containers, got %d", len(containers))
	}
	wantContainers := []string{"jellyfin", "transmission", "ollama", "gateway"}
	for index, want := range wantContainers {
		if containers[index].ID != want {
			t.Fatalf("container %d: got %q want %q", index, containers[index].ID, want)
		}
		if containers[index].Image == "" || len(containers[index].Ports) == 0 || len(containers[index].Mounts) == 0 || len(containers[index].Env) == 0 {
			t.Fatalf("container %q missing runtime fields: %#v", containers[index].ID, containers[index])
		}
	}
	if containers[0].Status != ContainerStatusRunning {
		t.Fatalf("expected jellyfin running, got %q", containers[0].Status)
	}
	if containers[3].Status != ContainerStatusStopped {
		t.Fatalf("expected gateway stopped, got %q", containers[3].Status)
	}
}

func TestContainerActionsUpdateStatusResourcesAndLogs(t *testing.T) {
	service := NewDevService()

	started, err := service.Start(context.Background(), "gateway")
	if err != nil {
		t.Fatalf("start gateway: %v", err)
	}
	if started.Status != ContainerStatusRunning || started.CPU == 0 || started.Memory == 0 {
		t.Fatalf("expected started gateway to have runtime usage, got %#v", started)
	}
	if !logContains(t, service, "gateway", "启动操作") {
		t.Fatal("expected start operation log")
	}

	stopped, err := service.Stop(context.Background(), "gateway")
	if err != nil {
		t.Fatalf("stop gateway: %v", err)
	}
	if stopped.Status != ContainerStatusStopped || stopped.CPU != 0 || stopped.Memory != 0 {
		t.Fatalf("expected stopped gateway to clear runtime usage, got %#v", stopped)
	}

	restarting, err := service.Restart(context.Background(), "gateway")
	if err != nil {
		t.Fatalf("restart gateway: %v", err)
	}
	if restarting.Status != ContainerStatusRestarting || restarting.Restarts != 5 {
		t.Fatalf("expected restart state and incremented restart count, got %#v", restarting)
	}

	completed, err := service.CompleteRestart(context.Background(), "gateway")
	if err != nil {
		t.Fatalf("complete restart gateway: %v", err)
	}
	if completed.Status != ContainerStatusRunning || completed.CPU < 10 || completed.Memory < 18 {
		t.Fatalf("expected completed restart to restore runtime usage, got %#v", completed)
	}
	if !logContains(t, service, "gateway", "健康检查通过") {
		t.Fatal("expected health-check completion log")
	}
}

func TestUpdateLimitsAndLogsAreBoundedAndCloned(t *testing.T) {
	service := NewDevService()

	updated, err := service.UpdateLimits(context.Background(), "jellyfin", ResourceLimit{CPU: 5, MemoryMB: 6144})
	if err != nil {
		t.Fatalf("update jellyfin limits: %v", err)
	}
	if updated.Limit.CPU != 5 || updated.Limit.MemoryMB != 6144 {
		t.Fatalf("unexpected updated limits: %#v", updated.Limit)
	}
	if !logContains(t, service, "jellyfin", "资源限制调整") {
		t.Fatal("expected limit update log")
	}

	logs, err := service.Logs(context.Background(), "jellyfin", 2)
	if err != nil {
		t.Fatalf("logs: %v", err)
	}
	if len(logs) != 2 {
		t.Fatalf("expected bounded 2 log entries, got %d", len(logs))
	}
	logs[0].Message = "mutated by test"
	nextLogs, err := service.Logs(context.Background(), "jellyfin", 1)
	if err != nil {
		t.Fatalf("logs again: %v", err)
	}
	if nextLogs[0].Message == "mutated by test" {
		t.Fatal("logs must be cloned before returning")
	}
}

func logContains(t *testing.T, service *DevService, containerID string, text string) bool {
	t.Helper()

	logs, err := service.Logs(context.Background(), containerID, 0)
	if err != nil {
		t.Fatalf("logs %s: %v", containerID, err)
	}
	for _, entry := range logs {
		if strings.Contains(entry.Message, text) {
			return true
		}
	}
	return false
}
