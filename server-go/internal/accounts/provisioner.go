package accounts

import (
	"context"
	"runtime"
)

// GrantSpec describes a space-access grant to project onto the filesystem as a
// POSIX ACL. SubjectName is the OS user or group name; SpaceDir is the NAS-root
// relative directory the grant targets.
type GrantSpec struct {
	SubjectName string      // OS username or group name
	SubjectKind SubjectType // SubjectUser or SubjectGroup
	SpaceDir    string      // NAS-root-relative directory, e.g. "team-space"
	Access      SpaceAccess // read / read_write / manage
}

// Provisioner realizes the user/group/grant model on the real filesystem under
// the NAS root: personal home folders, shared group folders, and per-space ACLs.
// HostProvisioner (Linux) runs mkdir/chown/chmod/setfacl/setquota; DevProvisioner
// (Mac/devstub) is a no-op so development never touches the host.
type Provisioner interface {
	// EnsureUserFolder creates homes/<username> owned by the user (0700) and,
	// best-effort, applies a disk quota when quotaBytes > 0.
	EnsureUserFolder(ctx context.Context, username string, uid int, quotaBytes int64) error
	// EnsureGroupFolder creates groups/<group> owned by the group (2770 setgid).
	EnsureGroupFolder(ctx context.Context, group string) error
	// ApplyGrant writes (recursive + default) ACLs for a space grant.
	ApplyGrant(ctx context.Context, spec GrantSpec) error
	// RemoveGrant clears a subject's ACL entry on a space.
	RemoveGrant(ctx context.Context, spec GrantSpec) error
}

// newProvisioner selects the filesystem provisioner. The system backend (Linux,
// or explicit) provisions for real; everything else is a no-op.
func newProvisioner(backend, nasRoot, group string) Provisioner {
	if backend == "" {
		if runtime.GOOS == "linux" {
			backend = "system"
		} else {
			backend = "devstub"
		}
	}
	if backend == "system" && nasRoot != "" {
		return newHostProvisioner(nasRoot, group)
	}
	return DevProvisioner{}
}

// DevProvisioner does nothing — used on Mac/devstub and when no NAS root is set.
type DevProvisioner struct{}

func (DevProvisioner) EnsureUserFolder(ctx context.Context, _ string, _ int, _ int64) error {
	return ctx.Err()
}
func (DevProvisioner) EnsureGroupFolder(ctx context.Context, _ string) error { return ctx.Err() }
func (DevProvisioner) ApplyGrant(ctx context.Context, _ GrantSpec) error     { return ctx.Err() }
func (DevProvisioner) RemoveGrant(ctx context.Context, _ GrantSpec) error    { return ctx.Err() }
