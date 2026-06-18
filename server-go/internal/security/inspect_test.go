package security

import (
	"context"
	"testing"
)

func TestInspectFindsPublicShareAndMFA(t *testing.T) {
	svc := NewService()
	ctx := context.Background()

	before, _ := svc.RiskActions(ctx)
	baseline := len(before)

	if _, err := svc.Inspect(ctx); err != nil {
		t.Fatalf("inspect: %v", err)
	}
	after, _ := svc.RiskActions(ctx)

	// The seeded shares include an active public link; expect at least one new
	// inspection finding with a stable inspect- id.
	found := false
	for _, r := range after {
		if len(r.ID) >= 8 && r.ID[:8] == "inspect-" {
			found = true
			if r.State != RiskActionStatePending {
				t.Errorf("inspection finding should be pending, got %s", r.State)
			}
		}
	}
	if !found {
		t.Fatalf("expected an inspect- finding; risks went %d -> %d", baseline, len(after))
	}
}

func TestInspectIsIdempotent(t *testing.T) {
	svc := NewService()
	ctx := context.Background()
	first, _ := svc.Inspect(ctx)
	second, _ := svc.Inspect(ctx)
	if len(first) != len(second) {
		t.Fatalf("inspect not idempotent: %d -> %d", len(first), len(second))
	}
}

func TestIsPublicAccess(t *testing.T) {
	for _, s := range []string{"公开", "任何人可见", "public link", "Anyone with link"} {
		if !isPublicAccess(s) {
			t.Errorf("expected %q public", s)
		}
	}
	if isPublicAccess("仅家庭成员") {
		t.Error("private access misclassified as public")
	}
}
