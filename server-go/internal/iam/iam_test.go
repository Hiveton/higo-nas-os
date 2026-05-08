package iam

import "testing"

func TestEvaluatorAllowsAdministrator(t *testing.T) {
	evaluator := NewEvaluator()

	decision := evaluator.Evaluate(EvaluationInput{
		ActorID: "admin-1",
		Role:    RoleAdmin,
		Space:   Space{ID: "system", Type: SpaceSystem},
		Scope:   Scope{Type: ScopeTool, ID: "docker.restart"},
		Action:  ActionManage,
	})

	if !decision.Allowed {
		t.Fatalf("expected administrator to be allowed, got %#v", decision)
	}
}

func TestEvaluatorAllowsFamilyMemberInFamilySpace(t *testing.T) {
	evaluator := NewEvaluator()

	decision := evaluator.Evaluate(EvaluationInput{
		ActorID: "member-1",
		Role:    RoleFamilyMember,
		Space:   Space{ID: "family", Type: SpaceFamily},
		Scope:   Scope{Type: ScopeFile, ID: "photo.jpg"},
		Action:  ActionWrite,
	})

	if !decision.Allowed {
		t.Fatalf("expected family member write in family space to be allowed, got %#v", decision)
	}
}

func TestEvaluatorLimitsGuestToSharedRead(t *testing.T) {
	evaluator := NewEvaluator()

	read := evaluator.Evaluate(EvaluationInput{
		ActorID: "guest-1",
		Role:    RoleGuest,
		Space:   Space{ID: "shared", Type: SpaceShared},
		Scope:   Scope{Type: ScopeFile, ID: "readme.txt"},
		Action:  ActionRead,
	})
	if !read.Allowed {
		t.Fatalf("expected guest shared read to be allowed, got %#v", read)
	}

	write := evaluator.Evaluate(EvaluationInput{
		ActorID: "guest-1",
		Role:    RoleGuest,
		Space:   Space{ID: "shared", Type: SpaceShared},
		Scope:   Scope{Type: ScopeFile, ID: "readme.txt"},
		Action:  ActionWrite,
	})
	if write.Allowed {
		t.Fatalf("expected guest write to be denied, got %#v", write)
	}
}

func TestEvaluatorExplicitDenyWinsOverAllow(t *testing.T) {
	evaluator := NewEvaluator(Permission{
		ActorID: "member-1",
		Effect:  EffectDeny,
		Scope:   Scope{Type: ScopeTool, ID: "agent.delete"},
		Action:  ActionExecute,
		Reason:  "tool disabled by policy",
	})

	decision := evaluator.Evaluate(EvaluationInput{
		ActorID: "member-1",
		Role:    RoleFamilyMember,
		Space:   Space{ID: "family", Type: SpaceFamily},
		Scope:   Scope{Type: ScopeTool, ID: "agent.delete"},
		Action:  ActionExecute,
	})

	if decision.Allowed {
		t.Fatalf("expected explicit deny to win, got %#v", decision)
	}
	if decision.Reason != "tool disabled by policy" {
		t.Fatalf("expected deny reason to be preserved, got %q", decision.Reason)
	}
}
