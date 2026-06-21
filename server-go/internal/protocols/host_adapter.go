package protocols

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// HostAdapter drives the real protocol stack on a Linux NAS host. Every
// service-level action shells out through the injectable runner; every config
// change is an atomic whole-file (or whole-block) rewrite of a file HiGoOS
// owns — the user's own /etc/samba/smb.conf and /etc/exports are never
// rewritten, only referenced.
//
// Safe-config strategy:
//   - Samba: a `include = <sambaInclude>` directive added once to smb.conf; the
//     include file itself is fully HiGoOS-owned and regenerated wholesale.
//   - NFS: a drop-in under /etc/exports.d/, never /etc/exports.
//   - minidlna: a delimited managed block inside /etc/minidlna.conf; lines
//     outside the block are preserved.
//   - WebDAV: a fully HiGoOS-owned apache include, regenerated wholesale.
type HostAdapter struct {
	runner       commandRunner
	sambaMain    string // /etc/samba/smb.conf
	sambaInclude string // /etc/samba/smb.conf.d/higoos.conf
	exportsPath  string // /etc/exports.d/higoos.exports
	davConf      string // /etc/higoos/webdav/higoos-dav.conf
	minidlnaConf string // /etc/minidlna.conf
}

const (
	dlnaBlockStart = "# >>> HiGoOS media_dir >>>"
	dlnaBlockEnd   = "# <<< HiGoOS media_dir <<<"
	managedHeader  = "# HiGoOS managed — do not edit by hand"
)

func NewHostAdapter() *HostAdapter {
	return NewHostAdapterWithRunner(runCommand)
}

func NewHostAdapterWithRunner(runner commandRunner) *HostAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &HostAdapter{
		runner:       runner,
		sambaMain:    "/etc/samba/smb.conf",
		sambaInclude: "/etc/samba/smb.conf.d/higoos.conf",
		exportsPath:  "/etc/exports.d/higoos.exports",
		davConf:      "/etc/higoos/webdav/higoos-dav.conf",
		minidlnaConf: "/etc/minidlna.conf",
	}
}

func unitsFor(key ProtocolKey) []string {
	switch key {
	case ProtocolSMB:
		return []string{"smbd", "nmbd"}
	case ProtocolNFS:
		return []string{"nfs-kernel-server"}
	case ProtocolWebDAV:
		return []string{"higoos-webdav"}
	case ProtocolDLNA:
		return []string{"minidlna"}
	default:
		return nil
	}
}

func primaryUnit(key ProtocolKey) string {
	if u := unitsFor(key); len(u) > 0 {
		return u[0]
	}
	return ""
}

func (a *HostAdapter) Status(ctx context.Context, desired []Protocol) ([]Protocol, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out := cloneProtocols(desired)
	for i := range out {
		unit := primaryUnit(out[i].Key)
		if unit == "" {
			continue
		}
		// A missing unit must not fail the whole list — it just means the
		// protocol's package isn't installed yet.
		out[i].Installed = a.unitInstalled(ctx, unit)
		out[i].Running = out[i].Installed && a.unitActive(ctx, unit)
	}
	return out, nil
}

func (a *HostAdapter) unitInstalled(ctx context.Context, unit string) bool {
	out, err := a.runner(ctx, "systemctl", "list-unit-files", unit+".service", "--no-legend")
	if err != nil {
		return false
	}
	return strings.Contains(string(out), unit)
}

func (a *HostAdapter) unitActive(ctx context.Context, unit string) bool {
	out, err := a.runner(ctx, "systemctl", "is-active", unit+".service")
	if err != nil {
		// is-active exits non-zero for inactive units but still prints state.
		return strings.TrimSpace(string(out)) == "active"
	}
	return strings.TrimSpace(string(out)) == "active"
}

func (a *HostAdapter) EnableProtocol(ctx context.Context, key ProtocolKey, cfg ProtocolBaseConfig) error {
	switch key {
	case ProtocolSMB:
		if err := a.ensureSambaInclude(); err != nil {
			return err
		}
	case ProtocolNFS:
		if err := os.MkdirAll(filepath.Dir(a.exportsPath), 0o755); err != nil {
			return fmt.Errorf("create exports.d: %w", err)
		}
	case ProtocolWebDAV:
		if err := os.MkdirAll(filepath.Dir(a.davConf), 0o755); err != nil {
			return fmt.Errorf("create webdav config dir: %w", err)
		}
	}
	for _, unit := range unitsFor(key) {
		if _, err := a.runner(ctx, "systemctl", "enable", "--now", unit); err != nil {
			return fmt.Errorf("enable %s: %w", unit, err)
		}
	}
	return nil
}

func (a *HostAdapter) DisableProtocol(ctx context.Context, key ProtocolKey) error {
	for _, unit := range unitsFor(key) {
		if _, err := a.runner(ctx, "systemctl", "disable", "--now", unit); err != nil {
			return fmt.Errorf("disable %s: %w", unit, err)
		}
	}
	return nil
}

func (a *HostAdapter) Apply(ctx context.Context, key ProtocolKey, config ProtocolConfig, shares []Share) error {
	enabled := make([]Share, 0, len(shares))
	for _, s := range shares {
		if s.Enabled {
			enabled = append(enabled, s)
		}
	}
	switch key {
	case ProtocolSMB:
		if err := writeFileAtomic(a.sambaInclude, renderSamba(config, enabled)); err != nil {
			return err
		}
		if _, err := a.runner(ctx, "smbcontrol", "all", "reload-config"); err != nil {
			// Fall back to a service reload when smbcontrol is unavailable.
			if _, ferr := a.runner(ctx, "systemctl", "reload", "smbd"); ferr != nil {
				return fmt.Errorf("reload samba: %w", err)
			}
		}
		return nil
	case ProtocolNFS:
		if err := writeFileAtomic(a.exportsPath, renderExports(config, enabled)); err != nil {
			return err
		}
		if _, err := a.runner(ctx, "exportfs", "-ra"); err != nil {
			return fmt.Errorf("exportfs: %w", err)
		}
		return nil
	case ProtocolWebDAV:
		if err := writeFileAtomic(a.davConf, renderWebDAV(config, enabled)); err != nil {
			return err
		}
		if _, err := a.runner(ctx, "systemctl", "reload", "higoos-webdav"); err != nil {
			return fmt.Errorf("reload webdav: %w", err)
		}
		return nil
	case ProtocolDLNA:
		if err := a.writeDLNAManagedBlock(config, enabled); err != nil {
			return err
		}
		if _, err := a.runner(ctx, "systemctl", "restart", "minidlna"); err != nil {
			return fmt.Errorf("restart minidlna: %w", err)
		}
		return nil
	default:
		return fmt.Errorf("unknown protocol: %s", key)
	}
}

// ensureSambaInclude makes /etc/samba/smb.conf reference the HiGoOS-owned
// include exactly once, and creates the include file if absent. The user's
// smb.conf is only ever appended-to (the include line), never rewritten —
// mirrors storage.ensureFstabEntry.
func (a *HostAdapter) ensureSambaInclude() error {
	if err := os.MkdirAll(filepath.Dir(a.sambaInclude), 0o755); err != nil {
		return fmt.Errorf("create samba include dir: %w", err)
	}
	if _, err := os.Stat(a.sambaInclude); os.IsNotExist(err) {
		if err := writeFileAtomic(a.sambaInclude, renderSamba(ProtocolConfig{}, nil)); err != nil {
			return err
		}
	}
	includeLine := fmt.Sprintf("   include = %s", a.sambaInclude)
	content, err := os.ReadFile(a.sambaMain)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read smb.conf: %w", err)
	}
	if bytes.Contains(content, []byte(a.sambaInclude)) {
		return nil
	}
	// Insert the include INSIDE [global] so the include file's leading global
	// params (server string / workgroup / min protocol) apply globally. If no
	// [global] section exists yet, create one.
	lines := strings.Split(string(content), "\n")
	out := make([]string, 0, len(lines)+2)
	inserted := false
	for _, line := range lines {
		out = append(out, line)
		if !inserted && strings.EqualFold(strings.TrimSpace(line), "[global]") {
			out = append(out, includeLine)
			inserted = true
		}
	}
	if !inserted {
		if len(content) > 0 && content[len(content)-1] != '\n' {
			out = append(out, "")
		}
		out = append(out, "[global]", includeLine)
	}
	return writeFileAtomic(a.sambaMain, strings.Join(out, "\n")+"\n")
}

// writeDLNAManagedBlock replaces (or appends) the HiGoOS-delimited media_dir
// block inside minidlna.conf, preserving every line outside the block.
func (a *HostAdapter) writeDLNAManagedBlock(config ProtocolConfig, shares []Share) error {
	content, err := os.ReadFile(a.minidlnaConf)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read minidlna.conf: %w", err)
	}
	block := renderDLNA(config, shares)
	lines := strings.Split(string(content), "\n")
	var out []string
	inBlock := false
	replaced := false
	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == dlnaBlockStart {
			inBlock = true
			out = append(out, block)
			replaced = true
			continue
		}
		if trimmed == dlnaBlockEnd {
			inBlock = false
			continue
		}
		if inBlock {
			continue
		}
		out = append(out, line)
	}
	rebuilt := strings.Join(out, "\n")
	if !replaced {
		if len(content) > 0 && !strings.HasSuffix(rebuilt, "\n") {
			rebuilt += "\n"
		}
		rebuilt += block
	}
	if !strings.HasSuffix(rebuilt, "\n") {
		rebuilt += "\n"
	}
	return writeFileAtomic(a.minidlnaConf, rebuilt)
}

// --- config rendering -------------------------------------------------------

func renderSamba(config ProtocolConfig, shares []Share) string {
	var b strings.Builder
	b.WriteString(managedHeader + "\n")
	// Global parameters first — this file is included from inside [global], so
	// these leading params apply globally before any [share] section header.
	if config.ServerName != "" {
		fmt.Fprintf(&b, "   server string = %s\n", config.ServerName)
	}
	if config.Workgroup != "" {
		fmt.Fprintf(&b, "   workgroup = %s\n", config.Workgroup)
	}
	if config.MinProtocol != "" {
		fmt.Fprintf(&b, "   server min protocol = %s\n", config.MinProtocol)
	}
	if config.GuestAccess {
		b.WriteString("   map to guest = Bad User\n")
	}
	for _, s := range shares {
		accountScoped := (s.AccessLevel == AccessAccount || s.AccessLevel == AccessPassword) && len(s.AllowedUsers) > 0
		readOnly := "no"
		if s.AccessLevel == AccessReadOnly {
			readOnly = "yes"
		}
		// Per-user read/write split: baseline the share read-only and promote the
		// write-listed users, so granted-but-read-only users can't write.
		if accountScoped && len(s.WriteUsers) > 0 {
			readOnly = "yes"
		}
		guestOK := "no"
		if s.AccessLevel == AccessPublic || s.Guest {
			guestOK = "yes"
		}
		fmt.Fprintf(&b, "\n[%s]\n", sambaShareName(s.Name))
		fmt.Fprintf(&b, "   path = %s\n", s.Path)
		b.WriteString("   browseable = yes\n")
		fmt.Fprintf(&b, "   read only = %s\n", readOnly)
		fmt.Fprintf(&b, "   guest ok = %s\n", guestOK)
		if accountScoped {
			fmt.Fprintf(&b, "   valid users = %s\n", strings.Join(s.AllowedUsers, " "))
			if len(s.WriteUsers) > 0 {
				fmt.Fprintf(&b, "   write list = %s\n", strings.Join(s.WriteUsers, " "))
			}
		}
		// Explicit deny (overrides group-inherited access), mirrors the deny ACL.
		if len(s.DenyUsers) > 0 {
			fmt.Fprintf(&b, "   invalid users = %s\n", strings.Join(s.DenyUsers, " "))
		}
		// Recycle bin: deleted files move to #recycle (Synology/fnOS behavior).
		if s.Recycle {
			b.WriteString("   vfs objects = recycle\n")
			b.WriteString("   recycle:repository = #recycle\n")
			b.WriteString("   recycle:keeptree = yes\n")
			b.WriteString("   recycle:versions = yes\n")
			b.WriteString("   recycle:touch = yes\n")
		}
	}
	return b.String()
}

func renderExports(config ProtocolConfig, shares []Share) string {
	var b strings.Builder
	b.WriteString(managedHeader + "\n")
	network := config.AllowedNetwork
	if network == "" {
		network = "*"
	}
	squash := config.Squash
	if squash == "" {
		squash = "root_squash"
	}
	for _, s := range shares {
		var opts string
		if s.AccessLevel == AccessReadOnly || s.AccessLevel == AccessPublic {
			opts = "ro,sync,no_subtree_check," + squash
		} else {
			opts = "rw,sync,no_subtree_check," + squash
		}
		fmt.Fprintf(&b, "%s %s(%s)\n", s.Path, network, opts)
	}
	return b.String()
}

func renderDLNA(config ProtocolConfig, shares []Share) string {
	var b strings.Builder
	b.WriteString(dlnaBlockStart + "\n")
	if config.FriendlyName != "" {
		fmt.Fprintf(&b, "friendly_name=%s\n", config.FriendlyName)
	}
	for _, s := range shares {
		fmt.Fprintf(&b, "media_dir=%s\n", s.Path)
	}
	b.WriteString(dlnaBlockEnd)
	return b.String()
}

func renderWebDAV(config ProtocolConfig, shares []Share) string {
	var b strings.Builder
	b.WriteString(managedHeader + "\n")
	if config.HTTPSEnabled {
		b.WriteString("# https requested — terminate TLS at the higoos-webdav vhost\n")
	}
	for _, s := range shares {
		alias := "/" + sambaShareName(s.Name)
		fmt.Fprintf(&b, "\nAlias %s \"%s\"\n", alias, s.Path)
		fmt.Fprintf(&b, "<Directory \"%s\">\n", s.Path)
		b.WriteString("    DAV On\n")
		if s.AccessLevel == AccessReadOnly {
			b.WriteString("    <LimitExcept GET OPTIONS PROPFIND>\n        Require all denied\n    </LimitExcept>\n")
		}
		if s.AccessLevel == AccessPublic {
			b.WriteString("    Require all granted\n")
		} else {
			b.WriteString("    AuthType Basic\n")
			b.WriteString("    AuthName \"HiGoOS WebDAV\"\n")
			b.WriteString("    AuthUserFile /etc/higoos/webdav/users.htpasswd\n")
			b.WriteString("    Require valid-user\n")
		}
		b.WriteString("</Directory>\n")
	}
	return b.String()
}

// sambaShareName sanitizes a display name into a config-safe share label.
func sambaShareName(name string) string {
	name = strings.TrimSpace(name)
	name = strings.ReplaceAll(name, " ", "_")
	name = strings.ReplaceAll(name, "[", "")
	name = strings.ReplaceAll(name, "]", "")
	if name == "" {
		return "higoos_share"
	}
	return name
}

// writeFileAtomic writes content via a temp file + rename, so a reader never
// sees a half-written config (mirrors state.SaveJSON).
func writeFileAtomic(path, content string) error {
	dir := filepath.Dir(path)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create dir %s: %w", dir, err)
	}
	tmp, err := os.CreateTemp(dir, ".higoos-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp for %s: %w", path, err)
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.WriteString(content); err != nil {
		tmp.Close()
		return fmt.Errorf("write temp for %s: %w", path, err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp for %s: %w", path, err)
	}
	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("rename temp to %s: %w", path, err)
	}
	return nil
}
