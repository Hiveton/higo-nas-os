package network

import (
	"context"
	"testing"

	"higoos/server-go/internal/audit"
)

func TestPreviewConfigRejectsInvalidStatic(t *testing.T) {
	s := NewService(NewDevAdapter())
	_, err := s.PreviewConfig(context.Background(), ConfigRequest{
		NetworkConfig: NetworkConfig{Interface: "eth0", Mode: ModeStatic, Address: "not-an-ip", Prefix: 24},
	})
	if err == nil {
		t.Fatal("expected validation error for bad static address")
	}
}

func TestPreviewConfirmAppliesAndAudits(t *testing.T) {
	dev := NewDevAdapter()
	s := NewService(dev)
	ctx := context.Background()

	preview, err := s.PreviewConfig(ctx, ConfigRequest{
		NetworkConfig: NetworkConfig{
			Interface: "eth0", Mode: ModeStatic, Address: "192.168.1.50", Prefix: 24,
			Gateway: "192.168.1.1", DNS: []string{"1.1.1.1"}, Hostname: "nas-test",
		},
		Actor: "tester",
	})
	if err != nil {
		t.Fatalf("preview: %v", err)
	}
	if preview.Risk != audit.RiskHigh {
		t.Fatalf("changing the address should be high risk, got %q", preview.Risk)
	}
	if !preview.RequiresConfirmation || preview.ConfirmationID == "" {
		t.Fatal("high-risk change must require a confirmation id")
	}

	// Nothing is applied until confirmation.
	cfg, _ := dev.CurrentConfig(ctx)
	if cfg.Address == "192.168.1.50" {
		t.Fatal("config applied before confirmation")
	}

	entry, err := s.Confirm(ctx, ConfirmRequest{ConfirmationID: preview.ConfirmationID, Actor: "tester"})
	if err != nil {
		t.Fatalf("confirm: %v", err)
	}
	if entry.Result != audit.ResultConfirmed || entry.Config == nil {
		t.Fatalf("confirm audit entry malformed: %+v", entry)
	}

	cfg, _ = dev.CurrentConfig(ctx)
	if cfg.Address != "192.168.1.50" || cfg.Mode != ModeStatic {
		t.Fatalf("config not applied: %+v", cfg)
	}

	// Rolling back restores the previous (DHCP) config.
	if _, err := s.Rollback(ctx, entry.ID, RollbackRequest{Actor: "tester", Reason: "test"}); err != nil {
		t.Fatalf("rollback: %v", err)
	}
	cfg, _ = dev.CurrentConfig(ctx)
	if cfg.Mode != ModeDHCP {
		t.Fatalf("rollback did not restore DHCP: %+v", cfg)
	}
}

func TestConfirmUnknownConfirmation(t *testing.T) {
	s := NewService(NewDevAdapter())
	if _, err := s.Confirm(context.Background(), ConfirmRequest{ConfirmationID: "nope"}); err == nil {
		t.Fatal("expected error for unknown confirmation id")
	}
}
