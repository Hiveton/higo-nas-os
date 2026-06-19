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
	group      string
	adminGroup string
	nologin    string
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
		group:      group,
		adminGroup: adminGroup,
		nologin:    nologin,
	}
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
	return nil
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
