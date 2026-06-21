package vm

import (
	"context"
	"fmt"
	"strings"
	"testing"
)

const dominfoHA = `Id:             1
Name:           home-assistant
UUID:           1234-uuid
OS Type:        hvm
State:          running
CPU(s):         2
Max memory:     2097152 KiB
Used memory:    2097152 KiB
Persistent:     yes
Autostart:      enable
`

func TestLibvirtListParsesDominfo(t *testing.T) {
	a := NewLibvirtAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name != "virsh" {
			return nil, fmt.Errorf("unexpected %s", name)
		}
		joined := strings.Join(args, " ")
		switch {
		case strings.Contains(joined, "list --all --name"):
			return []byte("home-assistant\n\nubuntu-lab\n"), nil
		case strings.Contains(joined, "dominfo home-assistant"):
			return []byte(dominfoHA), nil
		case strings.Contains(joined, "dominfo ubuntu-lab"):
			return []byte("Name:           ubuntu-lab\nState:          shut off\nCPU(s):         4\nMax memory:     4194304 KiB\nPersistent:     yes\nAutostart:      disable\n"), nil
		}
		return nil, fmt.Errorf("unexpected args %v", args)
	})
	vms, err := a.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(vms) != 2 {
		t.Fatalf("expected 2 vms, got %d", len(vms))
	}
	ha := vms[0]
	if ha.Name != "home-assistant" || ha.State != "running" || ha.VCPUs != 2 || ha.MemoryMB != 2048 || !ha.Autostart || !ha.Persistent {
		t.Fatalf("home-assistant parsed wrong: %+v", ha)
	}
	if vms[1].MemoryMB != 4096 || vms[1].State != "shut off" || vms[1].Autostart {
		t.Fatalf("ubuntu-lab parsed wrong: %+v", vms[1])
	}
}

func TestLibvirtActionIssuesVirsh(t *testing.T) {
	var calls []string
	a := NewLibvirtAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		return nil, nil
	})
	for _, action := range []string{ActionStart, ActionForceStop, ActionAutostartOff, ActionDelete} {
		if err := a.Action(context.Background(), "vm1", action); err != nil {
			t.Fatalf("action %s: %v", action, err)
		}
	}
	joined := strings.Join(calls, " | ")
	for _, want := range []string{"start vm1", "destroy vm1", "autostart --disable vm1", "undefine vm1 --nvram"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected virsh call %q in: %s", want, joined)
		}
	}
}

func TestServiceActionAudits(t *testing.T) {
	svc := NewServiceWithAdapter(NewDevAdapter())
	ctx := context.Background()
	v, err := svc.Action(ctx, "ubuntu-lab", ActionStart, "tester")
	if err != nil {
		t.Fatalf("start: %v", err)
	}
	if v.State != "running" {
		t.Fatalf("expected running after start, got %s", v.State)
	}
	entries, _ := svc.Audit(ctx)
	if len(entries) != 1 || entries[0].Action != ActionStart || entries[0].Result != "ok" {
		t.Fatalf("audit not recorded: %+v", entries)
	}
	// Delete removes it from the list.
	if _, err := svc.Action(ctx, "ubuntu-lab", ActionDelete, "tester"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	vms, _ := svc.List(ctx)
	for _, vm := range vms {
		if vm.Name == "ubuntu-lab" {
			t.Fatalf("ubuntu-lab should be deleted")
		}
	}
}
