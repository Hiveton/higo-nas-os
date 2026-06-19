package accounts

import (
	"context"
	"strings"
	"testing"
)

func fakeProvisioner(calls *[]recordedCall) *HostProvisioner {
	p := newHostProvisioner("/srv/higoos/nas", "higoos")
	p.runner = func(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
		*calls = append(*calls, recordedCall{stdin: stdin, name: name, args: args})
		return nil, nil
	}
	return p
}

func joinCall(c recordedCall) string {
	return c.name + " " + strings.Join(c.args, " ")
}

func TestHostProvisionerUserFolder(t *testing.T) {
	var calls []recordedCall
	p := fakeProvisioner(&calls)
	if err := p.EnsureUserFolder(context.Background(), "alice", 3001, 0); err != nil {
		t.Fatalf("ensure user folder: %v", err)
	}
	var sawMkdir, sawChown, sawChmod bool
	for _, c := range calls {
		switch c.name {
		case "mkdir":
			sawMkdir = strings.HasSuffix(joinCall(c), "/srv/higoos/nas/homes/alice")
		case "chown":
			sawChown = joinCall(c) == "chown alice:higoos /srv/higoos/nas/homes/alice"
		case "chmod":
			sawChmod = joinCall(c) == "chmod 0700 /srv/higoos/nas/homes/alice"
		}
	}
	if !sawMkdir || !sawChown || !sawChmod {
		t.Fatalf("expected mkdir+chown(alice:higoos)+chmod 0700; calls=%+v", calls)
	}
}

func TestHostProvisionerGroupFolder(t *testing.T) {
	var calls []recordedCall
	p := fakeProvisioner(&calls)
	if err := p.EnsureGroupFolder(context.Background(), "group-001-team"); err != nil {
		t.Fatalf("ensure group folder: %v", err)
	}
	var sawChown, sawChmod bool
	for _, c := range calls {
		if c.name == "chown" && joinCall(c) == "chown root:group-001-team /srv/higoos/nas/groups/group-001-team" {
			sawChown = true
		}
		if c.name == "chmod" && joinCall(c) == "chmod 2770 /srv/higoos/nas/groups/group-001-team" {
			sawChmod = true
		}
	}
	if !sawChown || !sawChmod {
		t.Fatalf("expected chown root:group + chmod 2770 (setgid); calls=%+v", calls)
	}
}

func TestHostProvisionerApplyGrantACL(t *testing.T) {
	var calls []recordedCall
	p := fakeProvisioner(&calls)
	// read_write grant to a user → rwX, plus a default ACL.
	err := p.ApplyGrant(context.Background(), GrantSpec{
		SubjectName: "alice", SubjectKind: SubjectUser, SpaceDir: "team-space", Access: AccessReadWrite,
	})
	if err != nil {
		t.Fatalf("apply grant: %v", err)
	}
	var sawACL, sawDefault bool
	for _, c := range calls {
		j := joinCall(c)
		if j == "setfacl -R -m u:alice:rwX /srv/higoos/nas/team-space" {
			sawACL = true
		}
		if j == "setfacl -R -d -m u:alice:rwX /srv/higoos/nas/team-space" {
			sawDefault = true
		}
	}
	if !sawACL || !sawDefault {
		t.Fatalf("expected recursive + default ACL; calls=%+v", calls)
	}

	// read-only grant to a group → rX, subject prefix g:.
	calls = nil
	_ = p.ApplyGrant(context.Background(), GrantSpec{
		SubjectName: "group-001-team", SubjectKind: SubjectGroup, SpaceDir: "home-space", Access: AccessReadOnly,
	})
	if joinCall(calls[0]) != "setfacl -R -m g:group-001-team:rX /srv/higoos/nas/home-space" {
		t.Fatalf("expected read-only group ACL rX; got %q", joinCall(calls[0]))
	}
}

func TestHostProvisionerPathEscapeRejected(t *testing.T) {
	var calls []recordedCall
	p := fakeProvisioner(&calls)
	if err := p.ApplyGrant(context.Background(), GrantSpec{SubjectName: "x", SubjectKind: SubjectUser, SpaceDir: "../../etc", Access: AccessReadOnly}); err == nil {
		t.Fatal("expected path escaping NAS root to be rejected")
	}
}
