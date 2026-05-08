package assistant

import (
	"context"
	"testing"
)

func TestServiceSeedsCurrentThreadAndAddsConfirmableMessage(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	thread, err := service.GetThread(ctx, "thread-current")
	if err != nil {
		t.Fatalf("get seeded thread: %v", err)
	}
	if len(thread.Messages) != 3 {
		t.Fatalf("seeded messages = %d, want 3", len(thread.Messages))
	}
	if thread.Messages[0].Role != RoleUser || thread.Messages[1].Role != RoleAssistant {
		t.Fatalf("seeded roles mismatch: %#v", thread.Messages)
	}

	result, err := service.AddMessage(ctx, "thread-current", MessageRequest{
		ActorID: "admin",
		Text:    "请生成下载目录整理计划",
		Scopes:  []string{"home", "team", "finance"},
	})
	if err != nil {
		t.Fatalf("add message: %v", err)
	}
	if result.UserMessage.Role != RoleUser || result.AssistantMessage.Role != RoleAssistant {
		t.Fatalf("message roles mismatch: %#v", result)
	}
	if !result.RequiresConfirmation || result.AssistantMessage.ActionID == "" {
		t.Fatalf("assistant action should require confirmation: %#v", result)
	}
	if result.Action == nil || result.Action.Risk != RiskMedium || result.Action.Status != ActionPending {
		t.Fatalf("pending medium-risk action mismatch: %#v", result.Action)
	}

	confirmed, err := service.ConfirmAction(ctx, result.Action.ID, ConfirmActionRequest{
		ActorID: "admin",
		Intent:  result.Action.Intent,
	})
	if err != nil {
		t.Fatalf("confirm action: %v", err)
	}
	if confirmed.Status != ActionConfirmed {
		t.Fatalf("action status = %q, want confirmed", confirmed.Status)
	}
}

func TestSemanticSearchFiltersByDevstubScopesAndReturnsCitations(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	teamResult, err := service.SemanticSearch(ctx, SemanticSearchRequest{
		Query:  "客户 A 合同 备份",
		Scopes: []string{"team"},
	})
	if err != nil {
		t.Fatalf("semantic search: %v", err)
	}
	if teamResult.Answer == "" || len(teamResult.Items) == 0 || len(teamResult.Citations) == 0 {
		t.Fatalf("semantic response missing answer/items/citations: %#v", teamResult)
	}
	if teamResult.Items[0].Scope != "team" || teamResult.Items[0].Title != "客户 A 合同最终版" {
		t.Fatalf("top team result mismatch: %#v", teamResult.Items[0])
	}

	homeResult, err := service.SemanticSearch(ctx, SemanticSearchRequest{
		Query:  "客户 A 合同",
		Scopes: []string{"home"},
	})
	if err != nil {
		t.Fatalf("semantic search with home scope: %v", err)
	}
	for _, item := range homeResult.Items {
		if item.Scope == "team" {
			t.Fatalf("team-scoped item leaked into home-only result: %#v", homeResult.Items)
		}
	}
}
