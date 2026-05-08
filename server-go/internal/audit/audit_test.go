package audit

import (
	"testing"
	"time"
)

func TestStoreAppendsAndListsAuditEvents(t *testing.T) {
	store := NewStore()
	event := AuditEvent{
		Time:      time.Date(2026, 5, 6, 12, 0, 0, 0, time.UTC),
		RequestID: "req-1",
		ActorID:   "admin-1",
		Domain:    "settings",
		Action:    "model_policy.update",
		Risk:      RiskMedium,
		Result:    ResultAllowed,
	}

	appended := store.Append(event)
	events := store.List()

	if appended.ID == "" {
		t.Fatal("expected append to assign id")
	}
	if len(events) != 1 || events[0].ID != appended.ID {
		t.Fatalf("unexpected events: %#v", events)
	}
}

func TestStoreConfirmsHighRiskActionAndMarksRollback(t *testing.T) {
	store := NewStore()
	action := store.CreateRiskAction(RiskAction{
		ActorID:       "admin-1",
		Action:        "share.public.create",
		TargetScope:   "space:family",
		Risk:          RiskHigh,
		ImpactSummary: "Create a public share",
		Rollback: &RollbackOperation{
			Type:     "share.revoke",
			TargetID: "share-1",
		},
	})

	confirmed, ok := store.ConfirmRiskAction(action.ID, "admin-1", time.Date(2026, 5, 6, 12, 5, 0, 0, time.UTC))
	if !ok {
		t.Fatal("expected confirmation to succeed")
	}
	if confirmed.Status != RiskActionConfirmed || confirmed.ConfirmedBy != "admin-1" {
		t.Fatalf("unexpected confirmed action: %#v", confirmed)
	}

	rolled, ok := store.MarkRollback(action.ID, RollbackSucceeded, "share revoked")
	if !ok {
		t.Fatal("expected rollback mark to succeed")
	}
	if rolled.Rollback == nil || rolled.Rollback.Status != RollbackSucceeded {
		t.Fatalf("expected rollback status to be marked, got %#v", rolled.Rollback)
	}
}

func TestStoreBlocksMediumRiskAction(t *testing.T) {
	store := NewStore()
	action := store.CreateRiskAction(RiskAction{
		ActorID:       "member-1",
		Action:        "files.batch.rename",
		TargetScope:   "space:family",
		Risk:          RiskMedium,
		ImpactSummary: "Rename 20 files",
	})

	blocked, ok := store.BlockRiskAction(action.ID, "member-1", "user cancelled")
	if !ok {
		t.Fatal("expected block to succeed")
	}
	if blocked.Status != RiskActionBlocked || blocked.BlockReason != "user cancelled" {
		t.Fatalf("unexpected blocked action: %#v", blocked)
	}
}
