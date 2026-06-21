package llm

import (
	"path/filepath"
	"testing"
)

func strptr(s string) *string { return &s }
func boolptr(b bool) *bool     { return &b }

func TestStoreCreateMasksKeyAndValidates(t *testing.T) {
	s := NewStore()

	if _, err := s.Create(ProviderInput{Name: strptr("no model"), Kind: kindPtr(KindOpenAI)}); err == nil {
		t.Fatal("expected error when model missing")
	}

	view, err := s.Create(ProviderInput{
		Name:   strptr("OpenAI"),
		Kind:   kindPtr(KindOpenAI),
		Model:  strptr("gpt-4o-mini"),
		APIKey: strptr("sk-secret-1234"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if view.KeyHint != "····1234" {
		t.Fatalf("expected masked hint ····1234, got %q", view.KeyHint)
	}
	if !view.HasKey {
		t.Fatal("expected HasKey true")
	}

	// The masked view must not be marshalable back into a raw key.
	list := s.List()
	if len(list) != 1 {
		t.Fatalf("expected 1 provider, got %d", len(list))
	}
}

func TestStoreUpdatePreservesKeyOnEmpty(t *testing.T) {
	s := NewStore()
	created, err := s.Create(ProviderInput{
		Name: strptr("p"), Kind: kindPtr(KindOpenAI), Model: strptr("m"), APIKey: strptr("topsecret"),
	})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	// Updating with an empty key must keep the existing secret.
	if _, err := s.Update(created.ID, ProviderInput{APIKey: strptr(""), Model: strptr("m2")}); err != nil {
		t.Fatalf("update: %v", err)
	}
	full, err := s.Get(created.ID)
	if err != nil {
		t.Fatalf("get: %v", err)
	}
	if full.APIKey != "topsecret" {
		t.Fatalf("expected key preserved, got %q", full.APIKey)
	}
	if full.Model != "m2" {
		t.Fatalf("expected model updated, got %q", full.Model)
	}
}

func TestStoreSingleDefaultAndDefaultResolution(t *testing.T) {
	s := NewStore()
	a, _ := s.Create(ProviderInput{Name: strptr("a"), Kind: kindPtr(KindOpenAI), Model: strptr("m"), IsDefault: boolptr(true)})
	b, _ := s.Create(ProviderInput{Name: strptr("b"), Kind: kindPtr(KindAnthropic), Model: strptr("m"), IsDefault: boolptr(true)})

	// Setting b default must clear a's default.
	def, err := s.Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if def.ID != b.ID {
		t.Fatalf("expected default %s, got %s", b.ID, def.ID)
	}
	aView, _ := s.Get(a.ID)
	if aView.IsDefault {
		t.Fatal("expected a no longer default")
	}
}

func TestStoreDefaultFallsBackToSingleEnabled(t *testing.T) {
	s := NewStore()
	only, _ := s.Create(ProviderInput{Name: strptr("solo"), Kind: kindPtr(KindOpenAI), Model: strptr("m")})
	def, err := s.Default()
	if err != nil {
		t.Fatalf("default: %v", err)
	}
	if def.ID != only.ID {
		t.Fatalf("expected single enabled provider as default")
	}

	// With two non-default enabled providers, Default is ambiguous.
	s.Create(ProviderInput{Name: strptr("second"), Kind: kindPtr(KindOpenAI), Model: strptr("m")})
	if _, err := s.Default(); err == nil {
		t.Fatal("expected error when default ambiguous")
	}
}

func TestStorePersistsAcrossReload(t *testing.T) {
	dir := t.TempDir()
	s, err := NewStoreWithStateDir(dir)
	if err != nil {
		t.Fatalf("new store: %v", err)
	}
	created, err := s.Create(ProviderInput{Name: strptr("p"), Kind: kindPtr(KindGemini), Model: strptr("gemini-1.5-pro"), APIKey: strptr("key1234")})
	if err != nil {
		t.Fatalf("create: %v", err)
	}

	reloaded, err := NewStoreWithStateDir(dir)
	if err != nil {
		t.Fatalf("reload: %v", err)
	}
	got, err := reloaded.Get(created.ID)
	if err != nil {
		t.Fatalf("get after reload: %v", err)
	}
	if got.APIKey != "key1234" || got.Model != "gemini-1.5-pro" {
		t.Fatalf("unexpected reloaded provider: %+v", got)
	}
	if filepath.Base(reloaded.statePath) != "llm_providers.json" {
		t.Fatalf("unexpected state path %q", reloaded.statePath)
	}
}

func TestSeedDefaultsIfEmpty(t *testing.T) {
	s := NewStore()
	n, err := s.SeedDefaultsIfEmpty(SeedConfig{
		BaseURL:     "http://127.0.0.1:11434/v1/",
		ChatModel:   "qwen2.5:7b",
		VisionModel: "qwen2.5-vl:7b",
	})
	if err != nil {
		t.Fatalf("seed: %v", err)
	}
	if n != 2 {
		t.Fatalf("expected 2 seeded providers (chat+vision), got %d", n)
	}
	chat, err := s.DefaultFor(PurposeChat)
	if err != nil {
		t.Fatalf("default chat: %v", err)
	}
	if chat.BaseURL != "http://127.0.0.1:11434/v1" || chat.Model != "qwen2.5:7b" || chat.Kind != KindOpenAI {
		t.Fatalf("unexpected seeded chat provider: %+v", chat)
	}
	if _, err := s.DefaultFor(PurposeVision); err != nil {
		t.Fatalf("default vision should resolve: %v", err)
	}
	// Idempotent / never clobbers existing providers.
	again, err := s.SeedDefaultsIfEmpty(SeedConfig{BaseURL: "http://other", ChatModel: "x"})
	if err != nil || again != 0 {
		t.Fatalf("second seed should be a no-op, got n=%d err=%v", again, err)
	}
}

func TestSeedDefaultsSkippedWhenIncomplete(t *testing.T) {
	s := NewStore()
	if n, _ := s.SeedDefaultsIfEmpty(SeedConfig{ChatModel: "x"}); n != 0 {
		t.Fatalf("missing base url should skip seeding, got %d", n)
	}
	if n, _ := s.SeedDefaultsIfEmpty(SeedConfig{BaseURL: "http://x"}); n != 0 {
		t.Fatalf("missing chat model should skip seeding, got %d", n)
	}
}

func kindPtr(k ProviderKind) *ProviderKind { return &k }
