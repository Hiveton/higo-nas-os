package docker

import (
	"context"
	"testing"
)

func TestCountComposeServices(t *testing.T) {
	cases := map[string]int{
		"services:\n  web:\n    image: nginx\n  db:\n    image: postgres\n": 2,
		"services:\n  only:\n    image: redis\n":                            1,
		"version: \"3\"\nservices:\n  a:\n    image: x\n  b:\n    image: y\n  c:\n    image: z\nnetworks:\n  default: {}\n": 3,
		"# no services here\nvolumes:\n  data: {}\n": 0,
	}
	for yaml, want := range cases {
		if got := countComposeServices(yaml); got != want {
			t.Errorf("countComposeServices(%q) = %d, want %d", yaml, got, want)
		}
	}
}

func TestSafeStackName(t *testing.T) {
	cases := map[string]string{
		"media stack":    "media-stack",
		"  My/Stack!!  ": "My-Stack",
		"":               "stack",
		"ok_name.1":      "ok_name.1",
	}
	for in, want := range cases {
		if got := safeStackName(in); got != want {
			t.Errorf("safeStackName(%q) = %q, want %q", in, got, want)
		}
	}
}

// TestStackYamlFromRecord verifies the in-memory yaml is returned without
// needing a docker daemon.
func TestStackYamlFromRecord(t *testing.T) {
	s := NewDevService()
	s.stacks = append(s.stacks, ComposeStack{Name: "rec-stack", Yaml: "services:\n  a:\n    image: x\n"})
	yaml, err := s.StackYaml(context.Background(), "rec-stack")
	if err != nil {
		t.Fatalf("StackYaml: %v", err)
	}
	if want := "services:\n  a:\n    image: x\n"; yaml != want {
		t.Fatalf("StackYaml = %q, want %q", yaml, want)
	}
	if _, err := s.StackYaml(context.Background(), "missing"); err == nil {
		t.Fatal("expected error for unknown stack")
	}
}
