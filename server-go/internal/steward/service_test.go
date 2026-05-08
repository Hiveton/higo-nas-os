package steward

import (
	"context"
	"testing"
)

func TestServiceSeedsSuggestionsAndAuditEntries(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	suggestions := service.ListSuggestions(ctx)
	if len(suggestions) != 3 {
		t.Fatalf("suggestions = %d, want 3", len(suggestions))
	}
	if suggestions[0].Title != "下载目录智能整理" || suggestions[1].Risk != RiskHigh {
		t.Fatalf("suggestion seeds mismatch: %#v", suggestions)
	}

	audit := service.Audit(ctx)
	if len(audit) != 3 {
		t.Fatalf("audit entries = %d, want 3", len(audit))
	}
}

func TestSuggestionPreviewConfirmDismissAndRollback(t *testing.T) {
	service := NewService()
	ctx := context.Background()

	preview, err := service.Preview(ctx, "download-cleanup", PreviewRequest{ActorID: "admin"})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if !preview.RequiresConfirmation || preview.ConfirmationID == "" || preview.Risk != RiskMedium {
		t.Fatalf("medium-risk preview should require confirmation: %#v", preview)
	}

	confirmed, err := service.Confirm(ctx, "download-cleanup", ConfirmRequest{
		ActorID:        "admin",
		ConfirmationID: preview.ConfirmationID,
	})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if confirmed.Suggestion.Status != SuggestionConfirmed || confirmed.AuditEntry.RollbackID == "" {
		t.Fatalf("confirmed suggestion missing audit rollback: %#v", confirmed)
	}

	rolledBack, err := service.Rollback(ctx, confirmed.AuditEntry.ID, RollbackRequest{
		ActorID: "admin",
		Reason:  "test rollback",
	})
	if err != nil {
		t.Fatalf("rollback: %v", err)
	}
	if rolledBack.Result != AuditRolledBack {
		t.Fatalf("rollback result = %q, want rolled_back", rolledBack.Result)
	}

	dismissed, err := service.Dismiss(ctx, "similar-photo-cleanup", DismissRequest{
		ActorID: "admin",
		Reason:  "skip now",
	})
	if err != nil {
		t.Fatalf("dismiss: %v", err)
	}
	if dismissed.Status != SuggestionDismissed {
		t.Fatalf("dismissed status = %q, want dismissed", dismissed.Status)
	}
}
