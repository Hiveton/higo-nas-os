package storage

import (
	"context"
	"testing"
)

// TestNoFakeSpaceSeed confirms storage spaces are NOT pre-seeded — they are real
// volumes the user creates on a storage pool, not file-category placeholders.
func TestNoFakeSpaceSeed(t *testing.T) {
	svc := NewServiceWithProvisioner(NewDevAdapter(), &recordingProvisioner{})
	spaces, err := svc.Spaces(context.Background())
	if err != nil {
		t.Fatalf("spaces: %v", err)
	}
	if len(spaces) != 0 {
		t.Fatalf("expected 0 seeded spaces, got %d", len(spaces))
	}
	if got := svc.DefaultSpaceID(context.Background()); got != "" {
		t.Fatalf("DefaultSpaceID with no spaces = %q, want empty", got)
	}
}

// TestPoolIDForSpace verifies a space binds to its storage pool: explicit pool
// wins, else it falls back to the pool of the first selected disk.
func TestPoolIDForSpace(t *testing.T) {
	disks := []Disk{{Slot: "2", PoolID: "host-dev-data"}}
	if got := poolIDForSpace("host-dev-data", nil); got != "host-dev-data" {
		t.Fatalf("explicit pool = %q, want host-dev-data", got)
	}
	if got := poolIDForSpace("", disks); got != "host-dev-data" {
		t.Fatalf("derived pool = %q, want host-dev-data (from disk)", got)
	}
	if got := poolIDForSpace("", nil); got != "" {
		t.Fatalf("no pool/disk = %q, want empty", got)
	}
}
