package assistant

import (
	"context"
	"testing"

	"higoos/server-go/internal/agent"
)

func TestPresetByID(t *testing.T) {
	if _, ok := PresetByID("nope"); ok {
		t.Fatal("unknown preset should not resolve")
	}
	p, ok := PresetByID("storage-admin")
	if !ok || len(p.ToolDomains) == 0 || p.SystemPrompt == "" {
		t.Fatalf("storage-admin preset incomplete: %+v", p)
	}
}

func TestCreateThreadWithPreset(t *testing.T) {
	svc := NewService()
	thread, err := svc.CreateThreadWithPreset(context.Background(), "", "storage-admin")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if thread.Preset != "storage-admin" || thread.SystemPrompt == "" || len(thread.ToolScope) == 0 {
		t.Fatalf("preset fields not set: %+v", thread)
	}
	if thread.Title == "" {
		t.Fatal("expected title defaulted to preset name")
	}
}

func TestToolDomainAndScope(t *testing.T) {
	if d := toolDomain("higo_storage_pools_list"); d != "storage" {
		t.Fatalf("toolDomain = %q", d)
	}
	tools := []agent.Tool{
		{Name: "higo_storage_pools_list"},
		{Name: "higo_monitoring_metrics_current"},
		{Name: "higo_docker_containers_list"},
	}
	scoped := scopeTools(tools, []string{"storage", "monitoring"})
	if len(scoped) != 2 {
		t.Fatalf("expected 2 scoped tools, got %d", len(scoped))
	}
	if got := scopeTools(tools, nil); len(got) != 3 {
		t.Fatalf("empty scope should return all, got %d", len(got))
	}
}
