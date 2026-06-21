package accounts

import (
	"context"
	"fmt"
	"path/filepath"
	"strconv"
	"strings"
)

// HostProvisioner realizes personal/group folders and per-space ACLs on the real
// filesystem under the NAS root. It shells out through the same injectable runner
// the host directory uses (unit-tested with a fake). Every path is confined to
// the NAS root. Requires the process to run as root (the higo-api unit does) and
// the host to have `acl` (setfacl) and, for quotas, `quota` (setquota) installed.
type HostProvisioner struct {
	runner  hostRunner
	nasRoot string
	group   string
}

func newHostProvisioner(nasRoot, group string) *HostProvisioner {
	if group == "" {
		group = "higoos"
	}
	return &HostProvisioner{runner: runHostCommand, nasRoot: filepath.Clean(nasRoot), group: group}
}

// safePath joins a NAS-root-relative path and refuses anything that escapes the
// root.
func (p *HostProvisioner) safePath(rel string) (string, error) {
	clean := filepath.Clean(filepath.Join(p.nasRoot, rel))
	relToRoot, err := filepath.Rel(p.nasRoot, clean)
	if err != nil || relToRoot == ".." || strings.HasPrefix(relToRoot, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("accounts: path escapes NAS root: %s", rel)
	}
	return clean, nil
}

func (p *HostProvisioner) EnsureUserFolder(ctx context.Context, username string, uid int, quotaBytes int64) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.safePath(filepath.Join("homes", username))
	if err != nil {
		return err
	}
	if out, err := p.runner(ctx, "", "mkdir", "-p", path); err != nil {
		return fmt.Errorf("accounts: mkdir home %s: %w (%s)", path, err, strings.TrimSpace(string(out)))
	}
	if _, err := p.runner(ctx, "", "chown", fmt.Sprintf("%s:%s", username, p.group), path); err != nil {
		return fmt.Errorf("accounts: chown home %s: %w", path, err)
	}
	if _, err := p.runner(ctx, "", "chmod", "0700", path); err != nil {
		return fmt.Errorf("accounts: chmod home %s: %w", path, err)
	}
	// Quota is best-effort: it only takes effect when the filesystem is mounted
	// with usrquota. Failures (no quota support) must not block account creation.
	if quotaBytes > 0 {
		p.applyQuota(ctx, username, quotaBytes)
	}
	return nil
}

func (p *HostProvisioner) EnsureGroupFolder(ctx context.Context, group string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.safePath(filepath.Join("groups", group))
	if err != nil {
		return err
	}
	if out, err := p.runner(ctx, "", "mkdir", "-p", path); err != nil {
		return fmt.Errorf("accounts: mkdir group %s: %w (%s)", path, err, strings.TrimSpace(string(out)))
	}
	if _, err := p.runner(ctx, "", "chown", "root:"+group, path); err != nil {
		return fmt.Errorf("accounts: chown group %s: %w", path, err)
	}
	// 2770: setgid so new files inherit the group; members get rwx, others none.
	if _, err := p.runner(ctx, "", "chmod", "2770", path); err != nil {
		return fmt.Errorf("accounts: chmod group %s: %w", path, err)
	}
	return nil
}

func (p *HostProvisioner) ApplyGrant(ctx context.Context, spec GrantSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.safePath(spec.SpaceDir)
	if err != nil {
		return err
	}
	entry := aclEntry(spec)
	// Recursive ACL on existing content + a default ACL so new children inherit.
	if out, err := p.runner(ctx, "", "setfacl", "-R", "-m", entry, path); err != nil {
		return fmt.Errorf("accounts: setfacl %s on %s: %w (%s)", entry, path, err, strings.TrimSpace(string(out)))
	}
	if _, err := p.runner(ctx, "", "setfacl", "-R", "-d", "-m", entry, path); err != nil {
		return fmt.Errorf("accounts: setfacl default %s on %s: %w", entry, path, err)
	}
	return nil
}

func (p *HostProvisioner) RemoveGrant(ctx context.Context, spec GrantSpec) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	path, err := p.safePath(spec.SpaceDir)
	if err != nil {
		return err
	}
	target := aclSubject(spec) // e.g. "u:alice" / "g:team1"
	_, _ = p.runner(ctx, "", "setfacl", "-R", "-x", target, path)
	_, _ = p.runner(ctx, "", "setfacl", "-R", "-d", "-x", target, path)
	return nil
}

// applyQuota sets a per-user block quota on the filesystem holding the NAS root.
func (p *HostProvisioner) applyQuota(ctx context.Context, username string, quotaBytes int64) {
	mount, err := p.runner(ctx, "", "findmnt", "-n", "-o", "TARGET", "--target", p.nasRoot)
	if err != nil {
		return
	}
	mountPoint := strings.TrimSpace(string(mount))
	if mountPoint == "" {
		return
	}
	blocks := strconv.FormatInt(quotaBytes/1024, 10) // setquota block unit is 1 KiB
	// soft = hard = blocks; no inode limit.
	_, _ = p.runner(ctx, "", "setquota", "-u", username, blocks, blocks, "0", "0", mountPoint)
}

// aclEntry builds a setfacl -m entry string for a grant.
func aclEntry(spec GrantSpec) string {
	return aclSubject(spec) + ":" + aclPerms(spec.Access)
}

func aclSubject(spec GrantSpec) string {
	if spec.SubjectKind == SubjectGroup {
		return "g:" + spec.SubjectName
	}
	return "u:" + spec.SubjectName
}

func aclPerms(access SpaceAccess) string {
	switch access {
	case AccessDeny:
		// A named-user/group ACL entry with no permissions denies the subject —
		// POSIX ACL evaluates the most specific matching entry, so this overrides
		// any access the subject would inherit from the owning/named group.
		return "---"
	case AccessReadOnly:
		return "rX" // read + traverse dirs only
	default: // read_write / manage
		return "rwX"
	}
}
