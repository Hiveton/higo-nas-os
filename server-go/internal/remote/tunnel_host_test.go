package remote

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

func TestWireGuardStatusParsesWgShow(t *testing.T) {
	a := NewWireGuardAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "wg" {
			return nil, fmt.Errorf("unexpected cmd %s", name)
		}
		switch strings.Join(args, " ") {
		case "show higoos public-key":
			return []byte("abcdEFGHpublicKEY1234567890abcdefghijklmnopq=\n"), nil
		case "show higoos listen-port":
			return []byte("51820\n"), nil
		case "show higoos peers":
			return []byte("peerkey1\npeerkey2\n"), nil
		}
		return nil, fmt.Errorf("unexpected args %v", args)
	})
	info, err := a.Status(context.Background())
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !info.Up || info.Backend != "wireguard" || info.ListenPort != 51820 || info.Peers != 2 {
		t.Fatalf("unexpected tunnel info: %+v", info)
	}
	if !strings.HasPrefix(info.PublicKey, "abcdEFGH") {
		t.Fatalf("public key not parsed: %+v", info)
	}
}

func TestWireGuardStatusDownWhenInterfaceMissing(t *testing.T) {
	a := NewWireGuardAdapterWithRunner(func(_ context.Context, _ string, _ ...string) ([]byte, error) {
		return nil, fmt.Errorf("Unable to access interface: No such device")
	})
	info, err := a.Status(context.Background())
	if err != nil {
		t.Fatalf("status should not error when iface is down: %v", err)
	}
	if info.Up {
		t.Fatalf("expected tunnel down, got %+v", info)
	}
}

func TestWireGuardUpBringsInterfaceUp(t *testing.T) {
	dir := t.TempDir()
	var calls []string
	a := NewWireGuardAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		switch {
		case name == "wg" && strings.Join(args, " ") == "show higoos public-key":
			// Down before up, then up afterwards. Simplest: report up.
			return []byte("PUBKEY=\n"), nil
		case name == "wg" && args[0] == "genkey":
			return []byte("PRIVKEYgenerated=\n"), nil
		case name == "wg" && strings.HasPrefix(strings.Join(args, " "), "show higoos"):
			return []byte("51820\n"), nil
		case name == "wg-quick":
			return nil, nil
		}
		return nil, nil
	})
	a.confPath = dir + "/higoos.conf"
	if _, err := a.Up(context.Background()); err != nil {
		t.Fatalf("up: %v", err)
	}
	// Since Status reports up immediately, wg-quick up may be skipped (idempotent).
	// Force the not-up path by checking config generation directly.
	a2 := NewWireGuardAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "wg" && strings.Join(args, " ") == "show higoos public-key" {
			return nil, fmt.Errorf("No such device")
		}
		if name == "wg" && args[0] == "genkey" {
			return []byte("PRIVKEY=\n"), nil
		}
		calls = append(calls, name+" "+strings.Join(args, " "))
		return []byte("51820\n"), nil
	})
	a2.confPath = dir + "/higoos2.conf"
	if _, err := a2.Up(context.Background()); err != nil {
		t.Fatalf("up2: %v", err)
	}
	joined := strings.Join(calls, " | ")
	if !strings.Contains(joined, "wg-quick up higoos") {
		t.Fatalf("expected wg-quick up to be invoked, calls: %s", joined)
	}
}
