package accounts

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// hostRunner shells out with optional stdin. Mirrors the protocols adapter's
// injectable runner so tests can drive HostDirectory without a real host.
type hostRunner func(ctx context.Context, stdin string, name string, args ...string) ([]byte, error)

func runHostCommand(ctx context.Context, stdin string, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	if stdin != "" {
		cmd.Stdin = strings.NewReader(stdin)
	}
	return cmd.CombinedOutput()
}

// HostDirectory provisions real Linux system users and authenticates against
// /etc/shadow. HiGoOS-managed users live in a dedicated primary group, get UIDs
// at or above a configured base (keeping them out of the system range), and are
// shell-less (/usr/sbin/nologin) — they exist for NAS access, not interactive
// login. Passwords are always set as SHA-512 crypt ($6$) so verification is a
// pure-Go path with no cgo/PAM dependency.
type HostDirectory struct {
	runner     hostRunner
	shadowPath string
	uidBase    int
	group       string
	adminGroup  string
	nologin     string
	sambaSync   bool
	loginUIDMin int // lowest UID treated as a real (human) login account
	loginUIDMax int // highest such UID
}

func newHostDirectory(cfg directoryConfig) *HostDirectory {
	group := cfg.Group
	if group == "" {
		group = "higoos"
	}
	adminGroup := cfg.AdminGroup
	if adminGroup == "" {
		adminGroup = "higoos-admins"
	}
	uidBase := cfg.UIDBase
	if uidBase == 0 {
		uidBase = 3000
	}
	nologin := "/usr/sbin/nologin"
	if _, err := os.Stat(nologin); err != nil {
		nologin = "/sbin/nologin"
	}
	return &HostDirectory{
		runner:     runHostCommand,
		shadowPath: "/etc/shadow",
		uidBase:    uidBase,
		group:       group,
		adminGroup:  adminGroup,
		nologin:     nologin,
		sambaSync:   true,
		loginUIDMin: 1000,
		loginUIDMax: 60000,
	}
}

// List enumerates real human login accounts from getent passwd (UID in the
// login range), tagging each as admin when it belongs to the admin/sudo/wheel
// group. This makes pre-existing system users — the installer's sudo user, any
// hand-created account — visible to HiGoOS without HiGoOS having created them.
func (d *HostDirectory) List(ctx context.Context) ([]SystemIdentity, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	out, err := d.runner(ctx, "", "getent", "passwd")
	if err != nil {
		return nil, fmt.Errorf("accounts: getent passwd: %w", err)
	}
	admins := d.adminMemberSet(ctx)
	var identities []SystemIdentity
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		id, ok := d.parsePasswd(scanner.Text(), admins)
		if ok {
			identities = append(identities, id)
		}
	}
	return identities, nil
}

// Lookup resolves a single account by username.
func (d *HostDirectory) Lookup(ctx context.Context, username string) (SystemIdentity, bool, error) {
	if err := ctx.Err(); err != nil {
		return SystemIdentity{}, false, err
	}
	out, err := d.runner(ctx, "", "getent", "passwd", username)
	if err != nil {
		return SystemIdentity{}, false, nil
	}
	id, ok := d.parsePasswd(strings.TrimSpace(string(out)), d.adminMemberSet(ctx))
	return id, ok, nil
}

// parsePasswd turns a passwd line into a SystemIdentity, filtering to the human
// login UID range and excluding nobody.
func (d *HostDirectory) parsePasswd(line string, admins map[string]bool) (SystemIdentity, bool) {
	fields := strings.Split(line, ":")
	if len(fields) < 7 {
		return SystemIdentity{}, false
	}
	name := fields[0]
	uid, err := strconv.Atoi(fields[2])
	if err != nil || uid < d.loginUIDMin || uid > d.loginUIDMax || name == "nobody" {
		return SystemIdentity{}, false
	}
	display := strings.TrimSpace(strings.Split(fields[4], ",")[0])
	if display == "" {
		display = name
	}
	return SystemIdentity{Username: name, UID: uid, DisplayName: display, Admin: admins[name]}, true
}

// adminMemberSet returns the union of members of the HiGoOS admin group plus
// the conventional sudo/wheel groups — membership maps to the admin role.
func (d *HostDirectory) adminMemberSet(ctx context.Context) map[string]bool {
	admins := make(map[string]bool)
	for _, g := range []string{d.adminGroup, "sudo", "wheel"} {
		out, err := d.runner(ctx, "", "getent", "group", g)
		if err != nil {
			continue
		}
		// group line: name:passwd:gid:member1,member2,...
		fields := strings.Split(strings.TrimSpace(string(out)), ":")
		if len(fields) < 4 {
			continue
		}
		for _, m := range strings.Split(fields[3], ",") {
			if m = strings.TrimSpace(m); m != "" {
				admins[m] = true
			}
		}
	}
	return admins
}

func (d *HostDirectory) EnsureUser(ctx context.Context, ref IdentityRef) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	username := strings.TrimSpace(ref.Username)
	if username == "" {
		return fmt.Errorf("accounts: username required")
	}
	if err := d.ensureGroup(ctx, d.group); err != nil {
		return err
	}
	if err := d.ensureGroup(ctx, d.adminGroup); err != nil {
		return err
	}
	if d.userExists(ctx, username) {
		if _, err := d.runner(ctx, "", "usermod", "-c", ref.DisplayName, username); err != nil {
			return fmt.Errorf("accounts: usermod %s: %w", username, err)
		}
	} else {
		uid, err := d.nextFreeUID(ctx)
		if err != nil {
			return err
		}
		args := []string{"-m", "-g", d.group, "-s", d.nologin, "-c", ref.DisplayName}
		if uid > 0 {
			args = append(args, "-u", strconv.Itoa(uid))
		}
		args = append(args, username)
		if out, err := d.runner(ctx, "", "useradd", args...); err != nil {
			return fmt.Errorf("accounts: useradd %s: %w (%s)", username, err, strings.TrimSpace(string(out)))
		}
	}
	// Reconcile admin-group membership.
	if ref.Admin {
		_, _ = d.runner(ctx, "", "gpasswd", "-a", username, d.adminGroup)
	} else {
		_, _ = d.runner(ctx, "", "gpasswd", "-d", username, d.adminGroup)
	}
	return nil
}

func (d *HostDirectory) RemoveUser(ctx context.Context, username string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !d.userExists(ctx, username) {
		return nil
	}
	// Safety: only delete accounts HiGoOS manages (UID >= configured base). A
	// pre-existing system/sudo user (e.g. the installer account) must never be
	// removed through the user center.
	if id, ok, _ := d.Lookup(ctx, username); ok && id.UID < d.uidBase {
		return fmt.Errorf("accounts: refusing to delete unmanaged system user %q (UID %d)", username, id.UID)
	}
	if out, err := d.runner(ctx, "", "userdel", "-r", username); err != nil {
		// userdel returns 12 when the mail spool/home is already gone; tolerate.
		if !strings.Contains(string(out), "mail spool") && !strings.Contains(string(out), "home directory") {
			return fmt.Errorf("accounts: userdel %s: %w (%s)", username, err, strings.TrimSpace(string(out)))
		}
	}
	return nil
}

func (d *HostDirectory) SetPassword(ctx context.Context, username, plaintext string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	// chpasswd -c SHA512 forces a $6$ hash regardless of the host default
	// (Ubuntu 24.04 defaults to yescrypt), keeping verification pure-Go.
	stdin := username + ":" + plaintext + "\n"
	if out, err := d.runner(ctx, stdin, "chpasswd", "-c", "SHA512"); err != nil {
		return fmt.Errorf("accounts: chpasswd %s: %w (%s)", username, err, strings.TrimSpace(string(out)))
	}
	d.syncSamba(ctx, username, plaintext)
	return nil
}

// syncSamba mirrors the credential into the Samba password database so SMB
// shares authenticate with the same account. Best-effort: hosts without Samba
// installed simply skip it.
func (d *HostDirectory) syncSamba(ctx context.Context, username, plaintext string) {
	if !d.sambaSync {
		return
	}
	// smbpasswd -s reads the password twice from stdin; -a adds the user.
	stdin := plaintext + "\n" + plaintext + "\n"
	_, _ = d.runner(ctx, stdin, "smbpasswd", "-s", "-a", username)
}

func (d *HostDirectory) VerifyPassword(ctx context.Context, username, plaintext string) (bool, error) {
	if err := ctx.Err(); err != nil {
		return false, err
	}
	hash, ok, err := d.shadowHash(username)
	if err != nil {
		return false, err
	}
	if !ok || hash == "" || strings.HasPrefix(hash, "!") || strings.HasPrefix(hash, "*") {
		// Locked, disabled, or password-less account.
		return false, nil
	}
	return verifyCryptHash(hash, plaintext)
}

func (d *HostDirectory) SetLocked(ctx context.Context, username string, locked bool) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	flag := "-U"
	if locked {
		flag = "-L"
	}
	if out, err := d.runner(ctx, "", "usermod", flag, username); err != nil {
		return fmt.Errorf("accounts: usermod %s %s: %w (%s)", flag, username, err, strings.TrimSpace(string(out)))
	}
	return nil
}

func (d *HostDirectory) HasCredential(ctx context.Context, username string) bool {
	hash, ok, err := d.shadowHash(username)
	if err != nil || !ok {
		return false
	}
	return hash != "" && !strings.HasPrefix(hash, "!") && !strings.HasPrefix(hash, "*")
}

// --- helpers ---------------------------------------------------------------

func (d *HostDirectory) userExists(ctx context.Context, username string) bool {
	_, err := d.runner(ctx, "", "getent", "passwd", username)
	return err == nil
}

// EnsureGroup creates the system group if it doesn't exist.
func (d *HostDirectory) EnsureGroup(ctx context.Context, name string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	return d.ensureGroup(ctx, name)
}

// SetGroupMembers replaces a system group's full membership (gpasswd -M).
func (d *HostDirectory) SetGroupMembers(ctx context.Context, name string, usernames []string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := d.ensureGroup(ctx, name); err != nil {
		return err
	}
	if out, err := d.runner(ctx, "", "gpasswd", "-M", strings.Join(usernames, ","), name); err != nil {
		return fmt.Errorf("accounts: gpasswd -M %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// SambaPresent parses `pdbedit -L` (username:uid:...) into a set of usernames
// that already have a Samba account. Hosts without Samba return an empty set.
func (d *HostDirectory) SambaPresent(ctx context.Context) (map[string]bool, error) {
	present := map[string]bool{}
	if !d.sambaSync {
		return present, nil
	}
	out, err := d.runner(ctx, "", "pdbedit", "-L")
	if err != nil {
		return present, nil // samba not installed / no accounts — treat as empty
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if name, _, ok := strings.Cut(line, ":"); ok {
			present[strings.TrimSpace(name)] = true
		}
	}
	return present, nil
}

func (d *HostDirectory) ensureGroup(ctx context.Context, name string) error {
	if _, err := d.runner(ctx, "", "getent", "group", name); err == nil {
		return nil
	}
	if out, err := d.runner(ctx, "", "groupadd", name); err != nil {
		return fmt.Errorf("accounts: groupadd %s: %w (%s)", name, err, strings.TrimSpace(string(out)))
	}
	return nil
}

// nextFreeUID returns the lowest unused UID at or above the configured base, or
// 0 to let useradd pick (when enumeration fails).
func (d *HostDirectory) nextFreeUID(ctx context.Context) (int, error) {
	out, err := d.runner(ctx, "", "getent", "passwd")
	if err != nil {
		return 0, nil
	}
	used := make(map[int]bool)
	scanner := bufio.NewScanner(bytes.NewReader(out))
	for scanner.Scan() {
		fields := strings.Split(scanner.Text(), ":")
		if len(fields) < 3 {
			continue
		}
		if uid, err := strconv.Atoi(fields[2]); err == nil {
			used[uid] = true
		}
	}
	for uid := d.uidBase; uid < d.uidBase+10000; uid++ {
		if !used[uid] {
			return uid, nil
		}
	}
	return 0, nil
}

// shadowHash reads the password hash field for username from /etc/shadow.
// Requires the process to run as root (higo-api.service does).
func (d *HostDirectory) shadowHash(username string) (string, bool, error) {
	content, err := os.ReadFile(d.shadowPath)
	if err != nil {
		return "", false, fmt.Errorf("accounts: read shadow: %w", err)
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	for scanner.Scan() {
		line := scanner.Text()
		fields := strings.SplitN(line, ":", 3)
		if len(fields) < 2 {
			continue
		}
		if fields[0] == username {
			return fields[1], true, nil
		}
	}
	return "", false, nil
}
