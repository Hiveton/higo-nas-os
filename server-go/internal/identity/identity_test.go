package identity

import (
	"testing"
)

func TestProviderGeneratesAndPersistsDeviceID(t *testing.T) {
	dir := t.TempDir()
	p, err := NewProvider(dir, "HiGoOS NAS", "1.2.3", ":8080")
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	id := p.DeviceID()
	if id == "" {
		t.Fatal("expected a generated device id")
	}

	// A second provider over the same state dir reuses the persisted id.
	p2, err := NewProvider(dir, "HiGoOS NAS", "1.2.3", ":8080")
	if err != nil {
		t.Fatalf("new provider 2: %v", err)
	}
	if p2.DeviceID() != id {
		t.Fatalf("device id not stable: %q != %q", p2.DeviceID(), id)
	}
}

func TestSnapshotFields(t *testing.T) {
	p, err := NewProvider(t.TempDir(), "HiGoOS NAS", "9.9.9", ":18082")
	if err != nil {
		t.Fatalf("new provider: %v", err)
	}
	snap := p.Snapshot()
	if snap.HTTPPort != 18082 {
		t.Fatalf("expected http port 18082, got %d", snap.HTTPPort)
	}
	if snap.Version != "9.9.9" || snap.Model != "HiGoOS NAS" {
		t.Fatalf("unexpected snapshot identity: %+v", snap)
	}
	if snap.Initialized {
		t.Fatal("fresh device should not be initialized")
	}
	if err := p.MarkInitialized(); err != nil {
		t.Fatalf("mark initialized: %v", err)
	}
	if !p.Snapshot().Initialized {
		t.Fatal("expected initialized after MarkInitialized")
	}
}
