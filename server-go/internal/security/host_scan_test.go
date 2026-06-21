package security

import (
	"context"
	"fmt"
	"testing"
)

func TestHostScanParsesSS(t *testing.T) {
	ssOut := "tcp   LISTEN 0 4096  0.0.0.0:445   0.0.0.0:*  users:((\"smbd\",pid=1234,fd=35))\n" +
		"tcp   LISTEN 0 128   127.0.0.1:5432 0.0.0.0:*  users:((\"postgres\",pid=900,fd=7))\n" +
		"tcp   LISTEN 0 4096  [::]:8080      [::]:*     users:((\"higo-api\",pid=500,fd=3))\n" +
		"udp   UNCONN 0 0     0.0.0.0:19999  0.0.0.0:*\n"
	a := NewHostScanAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "ss" {
			return []byte(ssOut), nil
		}
		return nil, fmt.Errorf("unexpected cmd %s", name)
	})
	ports, err := a.ListeningPorts(context.Background())
	if err != nil {
		t.Fatalf("ports: %v", err)
	}
	if len(ports) != 4 {
		t.Fatalf("expected 4 ports, got %d: %+v", len(ports), ports)
	}
	// smb on 0.0.0.0 → public + sensitive → high
	if ports[0].Port != 445 || ports[0].Process != "smbd" || ports[0].PID != 1234 || ports[0].Exposure != "公开监听" || ports[0].Risk != "high" {
		t.Fatalf("smb port parsed wrong: %+v", ports[0])
	}
	// postgres on localhost → 仅本机 → low
	if ports[1].Exposure != "仅本机" || ports[1].Risk != "low" {
		t.Fatalf("postgres exposure wrong: %+v", ports[1])
	}
	// ipv6 [::]:8080 → public, non-sensitive → medium
	if ports[2].Address != "::" || ports[2].Port != 8080 || ports[2].Exposure != "公开监听" || ports[2].Risk != "medium" {
		t.Fatalf("ipv6 8080 parsed wrong: %+v", ports[2])
	}
}

func TestHostScanFirewallNftables(t *testing.T) {
	nft := "table inet filter {\n\tchain input {\n\t\ttype filter hook input priority 0; policy drop;\n\t\tct state established,related accept\n\t\ttcp dport 22 accept\n\t}\n}\n"
	a := NewHostScanAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		if name == "nft" {
			return []byte(nft), nil
		}
		return nil, fmt.Errorf("not available")
	})
	fw, err := a.Firewall(context.Background())
	if err != nil {
		t.Fatalf("firewall: %v", err)
	}
	if fw.Backend != "nftables" || !fw.Active || fw.Rules < 2 {
		t.Fatalf("nft firewall parsed wrong: %+v", fw)
	}
}

func TestHostScanFirewallFallsBackToUfw(t *testing.T) {
	a := NewHostScanAdapterWithRunner(func(_ context.Context, name string, args ...string) ([]byte, error) {
		switch name {
		case "nft":
			return nil, fmt.Errorf("nft missing")
		case "ufw":
			return []byte("Status: active\nTo                         Action      From\n22/tcp                     ALLOW       Anywhere\n"), nil
		}
		return nil, fmt.Errorf("not available")
	})
	fw, err := a.Firewall(context.Background())
	if err != nil {
		t.Fatalf("firewall: %v", err)
	}
	if fw.Backend != "ufw" || !fw.Active || fw.Rules != 1 {
		t.Fatalf("ufw firewall parsed wrong: %+v", fw)
	}
}

func TestScanRecordsAudit(t *testing.T) {
	svc := NewService() // DevScanAdapter on Mac
	res, err := svc.Scan(context.Background(), "tester")
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	if len(res.Ports) == 0 || res.OpenToAll == 0 {
		t.Fatalf("dev scan should return seed ports with exposure: %+v", res)
	}
	entries, _ := svc.Audit(context.Background())
	if len(entries) == 0 || entries[0].Event == "" {
		t.Fatalf("scan should prepend an audit entry")
	}
}
