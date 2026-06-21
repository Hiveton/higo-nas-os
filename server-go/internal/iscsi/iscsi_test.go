package iscsi

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestHostListReadsConfigfs(t *testing.T) {
	root := t.TempDir()
	// Fake configfs: one target with 2 luns, 1 acl, 1 portal.
	iqn := "iqn.2026-06.os.higo:storage.lab"
	tpg := filepath.Join(root, iqn, "tpgt_1")
	for _, d := range []string{"lun/lun_0", "lun/lun_1", "acls/iqn.1994-05.com.redhat:client", "np/0.0.0.0:3260"} {
		if err := os.MkdirAll(filepath.Join(tpg, d), 0o755); err != nil {
			t.Fatal(err)
		}
	}
	a := NewHostAdapterWithRunner(nil)
	a.configfsRoot = root
	targets, err := a.List(context.Background())
	if err != nil {
		t.Fatalf("list: %v", err)
	}
	if len(targets) != 1 {
		t.Fatalf("expected 1 target, got %d", len(targets))
	}
	got := targets[0]
	if got.IQN != iqn || got.LUNs != 2 || len(got.ACLs) != 1 || len(got.Portals) != 1 {
		t.Fatalf("unexpected target: %+v", got)
	}
	if got.ACLs[0] != "iqn.1994-05.com.redhat:client" || got.Portals[0] != "0.0.0.0:3260" {
		t.Fatalf("acl/portal parsed wrong: %+v", got)
	}
}

func TestHostMutationsIssueTargetcli(t *testing.T) {
	var calls []string
	a := NewHostAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		calls = append(calls, name+" "+strings.Join(args, " "))
		if name == "targetcli" && len(args) >= 2 && args[0] == "/iscsi" && args[1] == "create" && len(args) == 2 {
			return []byte("Created target iqn.2003-01.org.linux-iscsi.nas:sn.abc123.\n"), nil
		}
		return nil, nil
	})
	a.backstoreDir = t.TempDir()
	ctx := context.Background()
	iqn, err := a.CreateTarget(ctx, "")
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if iqn != "iqn.2003-01.org.linux-iscsi.nas:sn.abc123" {
		t.Fatalf("auto IQN not parsed: %q", iqn)
	}
	if err := a.AddLUN(ctx, "iqn.x", "disk1", 512); err != nil {
		t.Fatalf("add lun: %v", err)
	}
	if err := a.AddACL(ctx, "iqn.x", "iqn.1994-05.com.redhat:c1"); err != nil {
		t.Fatalf("add acl: %v", err)
	}
	if err := a.DeleteTarget(ctx, "iqn.x"); err != nil {
		t.Fatalf("delete: %v", err)
	}
	joined := strings.Join(calls, " | ")
	for _, want := range []string{
		"targetcli /backstores/fileio create disk1",
		"targetcli /iscsi/iqn.x/tpg1/luns create /backstores/fileio/disk1",
		"targetcli /iscsi/iqn.x/tpg1/acls create iqn.1994-05.com.redhat:c1",
		"targetcli /iscsi delete iqn.x",
		"targetcli saveconfig",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("expected targetcli call %q in: %s", want, joined)
		}
	}
}

func TestServiceCreateAudits(t *testing.T) {
	svc := NewServiceWithAdapter(NewDevAdapter())
	ctx := context.Background()
	tgt, err := svc.CreateTarget(ctx, CreateTargetRequest{Actor: "tester"})
	if err != nil {
		t.Fatalf("create: %v", err)
	}
	if tgt.IQN == "" {
		t.Fatalf("expected generated IQN")
	}
	entries, _ := svc.Audit(ctx)
	if len(entries) != 1 || entries[0].Action != "create-target" || entries[0].Result != "ok" {
		t.Fatalf("audit not recorded: %+v", entries)
	}
}
