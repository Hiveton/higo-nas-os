package agents

import (
	"context"
	"testing"
)

func TestServiceSeedsTemplatesToolsAndWorkflowNodes(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	templates := service.ListTemplates(ctx)
	if len(templates) != 3 {
		t.Fatalf("templates = %d, want 3", len(templates))
	}
	if templates[0].Name != "家庭资料助手" || templates[1].Name != "项目资料 Agent" {
		t.Fatalf("template seeds mismatch: %#v", templates)
	}

	tools, err := service.Tools(ctx, templates[1].ID, []string{"team", "files"})
	if err != nil {
		t.Fatalf("tools: %v", err)
	}
	if len(tools) == 0 {
		t.Fatal("expected permission-filtered tools")
	}

	nodes := service.WorkflowNodes(ctx)
	if len(nodes) != 4 || nodes[2].Label != "确认" {
		t.Fatalf("workflow node seeds mismatch: %#v", nodes)
	}
}

func TestWorkflowPreviewRunConfirmCancelAndEvents(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	preview, err := service.PreviewWorkflow(ctx, WorkflowPreviewRequest{
		ActorID:    "admin",
		TemplateID: "project-agent",
		Goal:       "汇总合同并生成资料包",
		Scopes:     []string{"team", "files"},
	})
	if err != nil {
		t.Fatalf("preview workflow: %v", err)
	}
	if !preview.RequiresConfirmation || preview.ConfirmationID == "" || preview.Risk != RiskMedium {
		t.Fatalf("medium-risk preview should require confirmation: %#v", preview)
	}
	if len(preview.Nodes) != 4 || len(preview.Checkpoints) == 0 {
		t.Fatalf("preview missing nodes/checkpoints: %#v", preview)
	}

	run, err := service.StartRun(ctx, WorkflowRunRequest{
		ActorID:    "admin",
		TemplateID: "project-agent",
		Goal:       "汇总合同并生成资料包",
		Scopes:     []string{"team", "files"},
	})
	if err != nil {
		t.Fatalf("start run: %v", err)
	}
	if run.Status != RunWaitingConfirmation || !run.RequiresConfirmation || run.ConfirmationID == "" {
		t.Fatalf("run should pause for confirmation: %#v", run)
	}

	events, err := service.Events(ctx, run.ID)
	if err != nil {
		t.Fatalf("events: %v", err)
	}
	if len(events) == 0 || events[0].RunID != run.ID {
		t.Fatalf("run events missing or wrong run id: %#v", events)
	}

	confirmed, err := service.ConfirmRun(ctx, run.ID, ConfirmRunRequest{
		ActorID:        "admin",
		ConfirmationID: run.ConfirmationID,
	})
	if err != nil {
		t.Fatalf("confirm run: %v", err)
	}
	if confirmed.Status != RunCompleted || confirmed.RequiresConfirmation {
		t.Fatalf("confirmed run should complete in dev core: %#v", confirmed)
	}

	cancelRun, err := service.StartRun(ctx, WorkflowRunRequest{
		ActorID:    "admin",
		TemplateID: "project-agent",
		Goal:       "取消测试",
		Scopes:     []string{"team", "files"},
	})
	if err != nil {
		t.Fatalf("start cancel run: %v", err)
	}
	canceled, err := service.CancelRun(ctx, cancelRun.ID, CancelRunRequest{
		ActorID: "admin",
		Reason:  "user canceled",
	})
	if err != nil {
		t.Fatalf("cancel run: %v", err)
	}
	if canceled.Status != RunCanceled || canceled.RollbackID == "" {
		t.Fatalf("canceled run should carry rollback id: %#v", canceled)
	}
}
