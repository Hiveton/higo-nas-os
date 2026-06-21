package iscsi

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HostAdapter drives the real LIO iSCSI target. Listing reads configfs (a stable
// directory tree) for robustness; mutations shell out to `targetcli`. configfs
// paths and the backstore dir are overridable for off-Linux unit tests.
type HostAdapter struct {
	runner       commandRunner
	configfsRoot string // /sys/kernel/config/target/iscsi
	backstoreDir string // where fileio LUN images live
}

func NewHostAdapter() *HostAdapter { return NewHostAdapterWithRunner(runCommand) }

func NewHostAdapterWithRunner(runner commandRunner) *HostAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &HostAdapter{
		runner:       runner,
		configfsRoot: "/sys/kernel/config/target/iscsi",
		backstoreDir: "/var/lib/higoos/iscsi",
	}
}

func (a *HostAdapter) List(ctx context.Context) ([]Target, error) {
	entries, err := os.ReadDir(a.configfsRoot)
	if err != nil {
		// No targets defined yet / target module not loaded — empty, not an error.
		return nil, nil
	}
	var targets []Target
	for _, e := range entries {
		if !e.IsDir() || !strings.HasPrefix(e.Name(), "iqn") {
			continue
		}
		tpg := filepath.Join(a.configfsRoot, e.Name(), "tpgt_1")
		targets = append(targets, Target{
			IQN:     e.Name(),
			LUNs:    countDir(filepath.Join(tpg, "lun")),
			ACLs:    listDirNames(filepath.Join(tpg, "acls")),
			Portals: listDirNames(filepath.Join(tpg, "np")),
		})
	}
	return targets, nil
}

func (a *HostAdapter) Capabilities(ctx context.Context) (HostCaps, error) {
	caps := HostCaps{Backend: "LIO", Portal: "0.0.0.0:3260"}
	if _, err := a.runner(ctx, "targetcli", "version"); err != nil {
		caps.Available = false
		caps.Note = "未检测到 targetcli（iSCSI 目标栈不可用）"
		return caps, nil
	}
	caps.Available = true
	if _, err := os.Stat(filepath.Dir(a.configfsRoot)); err != nil {
		caps.Note = "targetcli 可用，但内核 target 模块尚未加载（首次创建目标时自动加载）"
	}
	return caps, nil
}

func (a *HostAdapter) CreateTarget(ctx context.Context, iqn string) (string, error) {
	args := []string{"/iscsi", "create"}
	if iqn != "" {
		args = append(args, iqn)
	}
	out, err := a.runner(ctx, "targetcli", args...)
	if err != nil {
		return "", fmt.Errorf("targetcli create target: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	created := iqn
	if created == "" {
		created = parseCreatedIQN(string(out))
	}
	_ = a.save(ctx)
	return created, nil
}

func (a *HostAdapter) DeleteTarget(ctx context.Context, iqn string) error {
	if out, err := a.runner(ctx, "targetcli", "/iscsi", "delete", iqn); err != nil {
		return fmt.Errorf("targetcli delete target: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return a.save(ctx)
}

func (a *HostAdapter) AddLUN(ctx context.Context, iqn, name string, sizeMB int) error {
	if sizeMB <= 0 {
		sizeMB = 1024
	}
	if err := os.MkdirAll(a.backstoreDir, 0o755); err != nil {
		return fmt.Errorf("create backstore dir: %w", err)
	}
	img := filepath.Join(a.backstoreDir, name+".img")
	if out, err := a.runner(ctx, "targetcli", "/backstores/fileio", "create", name, img, fmt.Sprintf("%dM", sizeMB)); err != nil {
		return fmt.Errorf("targetcli create backstore: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	if out, err := a.runner(ctx, "targetcli", "/iscsi/"+iqn+"/tpg1/luns", "create", "/backstores/fileio/"+name); err != nil {
		return fmt.Errorf("targetcli map lun: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return a.save(ctx)
}

func (a *HostAdapter) AddACL(ctx context.Context, iqn, initiator string) error {
	if out, err := a.runner(ctx, "targetcli", "/iscsi/"+iqn+"/tpg1/acls", "create", initiator); err != nil {
		return fmt.Errorf("targetcli create acl: %w (%s)", err, strings.TrimSpace(string(out)))
	}
	return a.save(ctx)
}

func (a *HostAdapter) save(ctx context.Context) error {
	_, err := a.runner(ctx, "targetcli", "saveconfig")
	return err
}

// --- parsing helpers --------------------------------------------------------

func countDir(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return 0
	}
	n := 0
	for _, e := range entries {
		if e.IsDir() {
			n++
		}
	}
	return n
}

func listDirNames(path string) []string {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil
	}
	var names []string
	for _, e := range entries {
		if e.IsDir() {
			names = append(names, e.Name())
		}
	}
	return names
}

// parseCreatedIQN extracts the auto-generated IQN from targetcli's
// "Created target iqn.2003-01.org...." output.
func parseCreatedIQN(out string) string {
	for _, line := range strings.Split(out, "\n") {
		if i := strings.Index(line, "iqn."); i >= 0 {
			rest := line[i:]
			rest = strings.TrimRight(strings.TrimSpace(rest), ".")
			if f := strings.Fields(rest); len(f) > 0 {
				return f[0]
			}
		}
	}
	return ""
}
