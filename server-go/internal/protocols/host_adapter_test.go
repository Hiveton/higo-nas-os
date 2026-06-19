package protocols

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recordingRunner captures invoked commands and lets the test script responses.
type recordingRunner struct {
	calls   [][]string
	respond func(name string, args ...string) ([]byte, error)
}

func (r *recordingRunner) run(_ context.Context, name string, args ...string) ([]byte, error) {
	r.calls = append(r.calls, append([]string{name}, args...))
	if r.respond != nil {
		return r.respond(name, args...)
	}
	return nil, nil
}

func (r *recordingRunner) invoked(sub ...string) bool {
	for _, call := range r.calls {
		joined := strings.Join(call, " ")
		if strings.Contains(joined, strings.Join(sub, " ")) {
			return true
		}
	}
	return false
}

func tempHostAdapter(t *testing.T, runner commandRunner) *HostAdapter {
	t.Helper()
	dir := t.TempDir()
	a := NewHostAdapterWithRunner(runner)
	a.sambaMain = filepath.Join(dir, "smb.conf")
	a.sambaInclude = filepath.Join(dir, "smb.conf.d", "higoos.conf")
	a.exportsPath = filepath.Join(dir, "exports.d", "higoos.exports")
	a.davConf = filepath.Join(dir, "webdav", "higoos-dav.conf")
	a.minidlnaConf = filepath.Join(dir, "minidlna.conf")
	return a
}

func TestHostAdapterStatusParsesSystemctl(t *testing.T) {
	rr := &recordingRunner{respond: func(name string, args ...string) ([]byte, error) {
		joined := strings.Join(args, " ")
		switch {
		case strings.HasPrefix(joined, "list-unit-files smbd"):
			return []byte("smbd.service enabled enabled"), nil
		case strings.HasPrefix(joined, "is-active smbd"):
			return []byte("active\n"), nil
		case strings.HasPrefix(joined, "list-unit-files nfs-kernel-server"):
			return []byte("nfs-kernel-server.service enabled"), nil
		case strings.HasPrefix(joined, "is-active nfs-kernel-server"):
			return []byte("inactive\n"), nil
		// webdav + minidlna units missing -> non-zero exit
		default:
			return []byte(""), os.ErrNotExist
		}
	}}
	a := tempHostAdapter(t, rr.run)
	desired := []Protocol{
		{Key: ProtocolSMB}, {Key: ProtocolNFS}, {Key: ProtocolWebDAV}, {Key: ProtocolDLNA},
	}
	out, err := a.Status(context.Background(), desired)
	if err != nil {
		t.Fatalf("status: %v", err)
	}
	if !out[0].Installed || !out[0].Running {
		t.Fatalf("smb should be installed+running: %#v", out[0])
	}
	if !out[1].Installed || out[1].Running {
		t.Fatalf("nfs should be installed but not running: %#v", out[1])
	}
	if out[2].Installed || out[3].Installed {
		t.Fatalf("missing units should be not-installed without failing the list: %#v %#v", out[2], out[3])
	}
}

func TestHostAdapterEnableSMBAddsIncludeIdempotently(t *testing.T) {
	rr := &recordingRunner{}
	a := tempHostAdapter(t, rr.run)
	if err := os.MkdirAll(filepath.Dir(a.sambaMain), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(a.sambaMain, []byte("[global]\n   workgroup = WORKGROUP\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for i := 0; i < 2; i++ {
		if err := a.EnableProtocol(ctx, ProtocolSMB, ProtocolBaseConfig{}); err != nil {
			t.Fatalf("enable: %v", err)
		}
	}
	content, err := os.ReadFile(a.sambaMain)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.Count(string(content), a.sambaInclude); got != 1 {
		t.Fatalf("include should appear exactly once, got %d:\n%s", got, content)
	}
	if !rr.invoked("systemctl", "enable", "--now", "smbd") {
		t.Fatalf("expected systemctl enable smbd, calls: %v", rr.calls)
	}
	if !rr.invoked("systemctl", "enable", "--now", "nmbd") {
		t.Fatalf("expected systemctl enable nmbd, calls: %v", rr.calls)
	}
}

func TestHostAdapterApplySharesSMBWritesStanzas(t *testing.T) {
	rr := &recordingRunner{}
	a := tempHostAdapter(t, rr.run)
	shares := []Share{
		{Name: "家庭", Path: "/srv/家庭", AccessLevel: AccessAccount, AllowedUsers: []string{"alice", "bob"}, Enabled: true},
		{Name: "只读盘", Path: "/srv/ro", AccessLevel: AccessReadOnly, Enabled: true},
		{Name: "停用", Path: "/srv/off", AccessLevel: AccessPublic, Enabled: false},
	}
	config := ProtocolConfig{ServerName: "MyNAS", Workgroup: "HOME", MinProtocol: "SMB3"}
	if err := a.Apply(context.Background(), ProtocolSMB, config, shares); err != nil {
		t.Fatalf("apply: %v", err)
	}
	content, err := os.ReadFile(a.sambaInclude)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "server string = MyNAS") || !strings.Contains(text, "workgroup = HOME") || !strings.Contains(text, "server min protocol = SMB3") {
		t.Fatalf("global config params missing from include:\n%s", text)
	}
	if !strings.Contains(text, "[家庭]") || !strings.Contains(text, "valid users = alice bob") {
		t.Fatalf("account stanza missing valid users:\n%s", text)
	}
	if !strings.Contains(text, "[只读盘]") || !strings.Contains(text, "read only = yes") {
		t.Fatalf("readonly stanza wrong:\n%s", text)
	}
	if strings.Contains(text, "[停用]") {
		t.Fatalf("disabled share should not be rendered:\n%s", text)
	}
	if !rr.invoked("smbcontrol", "all", "reload-config") {
		t.Fatalf("expected smbcontrol reload, calls: %v", rr.calls)
	}
}

func TestHostAdapterApplySharesNFSWritesDropInOnly(t *testing.T) {
	rr := &recordingRunner{}
	a := tempHostAdapter(t, rr.run)
	shares := []Share{
		{Name: "pub", Path: "/srv/pub", AccessLevel: AccessPublic, Enabled: true},
		{Name: "rw", Path: "/srv/rw", AccessLevel: AccessAccount, Enabled: true},
	}
	config := ProtocolConfig{Squash: "root_squash", AllowedNetwork: "192.168.0.0/16"}
	if err := a.Apply(context.Background(), ProtocolNFS, config, shares); err != nil {
		t.Fatalf("apply: %v", err)
	}
	content, err := os.ReadFile(a.exportsPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "/srv/pub 192.168.0.0/16(ro,sync,no_subtree_check,root_squash)") {
		t.Fatalf("public export should be read-only on the configured network:\n%s", text)
	}
	if !strings.Contains(text, "/srv/rw 192.168.0.0/16(rw,sync,no_subtree_check,root_squash)") {
		t.Fatalf("account export should be rw on the configured network:\n%s", text)
	}
	// The user's own /etc/exports must never be opened.
	if _, err := os.Stat(filepath.Join(filepath.Dir(filepath.Dir(a.exportsPath)), "exports")); !os.IsNotExist(err) {
		t.Fatalf("/etc/exports should not be created")
	}
	if !rr.invoked("exportfs", "-ra") {
		t.Fatalf("expected exportfs -ra, calls: %v", rr.calls)
	}
}

func TestHostAdapterApplySharesDLNAPreservesUserLines(t *testing.T) {
	rr := &recordingRunner{}
	a := tempHostAdapter(t, rr.run)
	if err := os.MkdirAll(filepath.Dir(a.minidlnaConf), 0o755); err != nil {
		t.Fatal(err)
	}
	original := "friendly_name=MyNAS\nport=8200\n" + dlnaBlockStart + "\nmedia_dir=/old\n" + dlnaBlockEnd + "\nnotify_interval=900\n"
	if err := os.WriteFile(a.minidlnaConf, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}
	shares := []Share{{Name: "影视", Path: "/srv/影视", AccessLevel: AccessPublic, Enabled: true}}
	if err := a.Apply(context.Background(), ProtocolDLNA, ProtocolConfig{FriendlyName: "客厅NAS"}, shares); err != nil {
		t.Fatalf("apply: %v", err)
	}
	content, err := os.ReadFile(a.minidlnaConf)
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "friendly_name=MyNAS") || !strings.Contains(text, "notify_interval=900") {
		t.Fatalf("user lines outside the managed block must survive:\n%s", text)
	}
	if strings.Contains(text, "media_dir=/old") {
		t.Fatalf("old managed media_dir should be replaced:\n%s", text)
	}
	if !strings.Contains(text, "media_dir=/srv/影视") {
		t.Fatalf("new media_dir missing:\n%s", text)
	}
	if strings.Count(text, dlnaBlockStart) != 1 {
		t.Fatalf("managed block should appear exactly once:\n%s", text)
	}
	if !rr.invoked("systemctl", "restart", "minidlna") {
		t.Fatalf("expected minidlna restart, calls: %v", rr.calls)
	}
}
