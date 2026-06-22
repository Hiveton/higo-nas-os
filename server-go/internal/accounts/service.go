package accounts

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// ErrInvalidCredentials is returned when a username/password pair does not
// authenticate. It is deliberately indistinguishable from "no such user".
var ErrInvalidCredentials = errors.New("invalid username or password")

// ErrAccountInactive is returned when the account exists but is disabled or
// locked.
var ErrAccountInactive = errors.New("account is disabled or locked")

// ErrInvalidMFACode is returned when a TOTP code does not match.
var ErrInvalidMFACode = errors.New("invalid verification code")

type Service struct {
	mu        sync.RWMutex
	seq       int
	users     []User
	groups    []Group
	grants    []SpaceGrant
	now       func() time.Time
	statePath string
	dir       Directory
	mfa       *mfaStore
	prov      Provisioner
	nasRoot   string
}

type snapshot struct {
	Seq    int          `json:"seq"`
	Users  []User       `json:"users"`
	Groups []Group      `json:"groups"`
	Grants []SpaceGrant `json:"grants"`
}

func NewService() *Service {
	service := &Service{now: time.Now}
	dir, _ := newDevDirectory("")
	service.dir = dir
	service.mfa, _ = newMFAStore("")
	service.prov = DevProvisioner{}
	service.seed()
	return service
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	return NewServiceWithConfig(Config{StateDir: stateDir})
}

// Config tunes the accounts service: where state lives and which identity
// directory backend authenticates/provisions users.
type Config struct {
	StateDir   string
	Backend    string // "system", "devstub", or "" (auto by OS)
	UIDBase    int
	Group      string
	AdminGroup string
	NASRoot    string // file storage root; personal/group folders live under it
}

// NewServiceWithConfig builds the accounts service with an explicit identity
// directory backend. The directory owns credentials and (on Linux) mirrors
// users to real system accounts; the service owns app-level metadata.
func NewServiceWithConfig(cfg Config) (*Service, error) {
	dir, err := newDirectory(directoryConfig{
		Backend:    cfg.Backend,
		StateDir:   cfg.StateDir,
		UIDBase:    cfg.UIDBase,
		Group:      cfg.Group,
		AdminGroup: cfg.AdminGroup,
	})
	if err != nil {
		return nil, err
	}
	mfa, err := newMFAStore(cfg.StateDir)
	if err != nil {
		return nil, err
	}
	group := cfg.Group
	if group == "" {
		group = "higoos"
	}
	service := &Service{
		now:     time.Now,
		dir:     dir,
		mfa:     mfa,
		prov:    newProvisioner(cfg.Backend, cfg.NASRoot, group),
		nasRoot: cfg.NASRoot,
	}
	service.seed()
	if cfg.StateDir != "" {
		service.statePath = filepath.Join(cfg.StateDir, "accounts.json")
		var persisted snapshot
		if err := state.LoadJSON(service.statePath, &persisted); err != nil {
			return nil, err
		}
		if len(persisted.Users) > 0 || len(persisted.Groups) > 0 || len(persisted.Grants) > 0 {
			service.seq = persisted.Seq
			service.users = cloneUsers(persisted.Users)
			service.groups = cloneGroups(persisted.Groups)
			service.grants = cloneGrants(persisted.Grants)
		}
	}
	// Pull existing OS accounts into the user list (no-op on the dev backend).
	service.reconcileSystemUsers(context.Background())
	return service, nil
}

func (s *Service) Summary(ctx context.Context) (Summary, error) {
	if err := ctx.Err(); err != nil {
		return Summary{}, err
	}
	// Refresh from the OS so newly-added system users appear without a restart.
	s.reconcileSystemUsers(ctx)
	s.mu.RLock()
	defer s.mu.RUnlock()
	return Summary{
		Users:  cloneUsers(s.users),
		Groups: cloneGroups(s.groups),
		Grants: cloneGrants(s.grants),
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, request CreateUserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	username := strings.TrimSpace(request.Username)
	if username == "" {
		return User{}, fmt.Errorf("username is required")
	}
	if err := validatePassword(request.Password); err != nil {
		return User{}, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.findUserByUsernameLocked(username); ok {
		return User{}, fmt.Errorf("username already exists: %s", username)
	}
	now := s.now().UTC()
	s.seq++
	user := User{
		ID:          fmt.Sprintf("user-%03d-%s", s.seq, slugID(username)),
		Username:    username,
		DisplayName: strings.TrimSpace(request.DisplayName),
		Role:        defaultRole(request.Role),
		Status:      StatusActive,
		QuotaBytes:  request.QuotaBytes,
		Groups:      uniqueStrings(request.Groups),
		HomeSpaceID: strings.TrimSpace(request.HomeSpaceID),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	if user.DisplayName == "" {
		user.DisplayName = username
	}
	// Mirror the identity to the host directory and set its credential before
	// committing to our own store, so a useradd/chpasswd failure doesn't leave a
	// metadata record without a backing system account.
	ref := IdentityRef{Username: user.Username, DisplayName: user.DisplayName, Admin: user.Role == RoleAdmin}
	if err := s.dir.EnsureUser(ctx, ref); err != nil {
		return User{}, err
	}
	if err := s.dir.SetPassword(ctx, user.Username, request.Password); err != nil {
		_ = s.dir.RemoveUser(ctx, user.Username)
		return User{}, err
	}
	s.provisionUserFolder(ctx, user)
	s.users = append(s.users, user)
	s.syncGroupMembershipLocked(user.ID, user.Groups)
	return user, s.saveLocked()
}

// provisionUserFolder creates the user's personal folder on the NAS (best-effort:
// a host/storage hiccup must not fail account creation; no-op on the dev backend).
func (s *Service) provisionUserFolder(ctx context.Context, user User) {
	if s.prov == nil {
		return
	}
	uid := 0
	if id, ok, _ := s.dir.Lookup(ctx, user.Username); ok {
		uid = id.UID
	}
	_ = s.prov.EnsureUserFolder(ctx, user.Username, uid, user.QuotaBytes)
}

func (s *Service) UpdateUser(ctx context.Context, id string, request UpdateUserRequest) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findUserLocked(id)
	if !ok {
		return User{}, fmt.Errorf("user not found: %s", id)
	}
	user := &s.users[index]
	if request.DisplayName != "" {
		user.DisplayName = strings.TrimSpace(request.DisplayName)
	}
	if request.Password != "" {
		if err := validatePassword(request.Password); err != nil {
			return User{}, err
		}
	}
	if request.Role != "" {
		user.Role = request.Role
	}
	if request.Status != "" {
		user.Status = request.Status
	}
	if request.QuotaBytes != nil {
		user.QuotaBytes = *request.QuotaBytes
	}
	if request.HomeSpaceID != nil {
		user.HomeSpaceID = strings.TrimSpace(*request.HomeSpaceID)
	}
	if request.Groups != nil {
		user.Groups = uniqueStrings(request.Groups)
		s.syncGroupMembershipLocked(user.ID, user.Groups)
	}
	// Mirror identity/credential/lock changes to the host directory.
	if err := s.dir.EnsureUser(ctx, IdentityRef{Username: user.Username, DisplayName: user.DisplayName, Admin: user.Role == RoleAdmin}); err != nil {
		return User{}, err
	}
	if request.Password != "" {
		if err := s.dir.SetPassword(ctx, user.Username, request.Password); err != nil {
			return User{}, err
		}
	}
	if request.Status != "" {
		locked := user.Status == StatusLocked || user.Status == StatusDisabled
		if err := s.dir.SetLocked(ctx, user.Username, locked); err != nil {
			return User{}, err
		}
	}
	user.UpdatedAt = s.now().UTC()
	return *user, s.saveLocked()
}

func (s *Service) DeleteUser(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findUserLocked(id)
	if !ok {
		return fmt.Errorf("user not found: %s", id)
	}
	username := s.users[index].Username
	if err := s.dir.RemoveUser(ctx, username); err != nil {
		return err
	}
	s.users = append(s.users[:index], s.users[index+1:]...)
	for i := range s.groups {
		s.groups[i].UserIDs = removeString(s.groups[i].UserIDs, id)
	}
	nextGrants := s.grants[:0]
	for _, grant := range s.grants {
		if !(grant.SubjectType == SubjectUser && grant.SubjectID == id) {
			nextGrants = append(nextGrants, grant)
		}
	}
	s.grants = nextGrants
	return s.saveLocked()
}

func (s *Service) CreateGroup(ctx context.Context, request CreateGroupRequest) (Group, error) {
	if err := ctx.Err(); err != nil {
		return Group{}, err
	}
	name := strings.TrimSpace(request.Name)
	if name == "" {
		return Group{}, fmt.Errorf("group name is required")
	}
	s.mu.Lock()
	now := s.now().UTC()
	s.seq++
	group := Group{
		ID:          fmt.Sprintf("group-%03d-%s", s.seq, slugID(name)),
		Name:        name,
		Description: strings.TrimSpace(request.Description),
		UserIDs:     uniqueStrings(request.UserIDs),
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	s.groups = append(s.groups, group)
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return group, err
	}
	// Realize a system group (named by the stable HiGoOS group ID) and its
	// shared folder (best-effort; no-op on the dev backend).
	_ = s.dir.EnsureGroup(ctx, group.ID)
	if s.prov != nil {
		_ = s.prov.EnsureGroupFolder(ctx, group.ID)
	}
	return group, nil
}

func (s *Service) UpdateGroupMembers(ctx context.Context, id string, request UpdateGroupMembersRequest) (Group, error) {
	if err := ctx.Err(); err != nil {
		return Group{}, err
	}
	s.mu.Lock()
	index, ok := s.findGroupLocked(id)
	if !ok {
		s.mu.Unlock()
		return Group{}, fmt.Errorf("group not found: %s", id)
	}
	group := &s.groups[index]
	group.UserIDs = uniqueStrings(request.UserIDs)
	group.UpdatedAt = s.now().UTC()
	var memberNames []string
	for i := range s.users {
		if containsString(group.UserIDs, s.users[i].ID) {
			if !containsString(s.users[i].Groups, group.ID) {
				s.users[i].Groups = append(s.users[i].Groups, group.ID)
			}
			memberNames = append(memberNames, s.users[i].Username)
		} else {
			s.users[i].Groups = removeString(s.users[i].Groups, group.ID)
		}
	}
	result := *group
	groupSysName := group.ID
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return result, err
	}
	// Sync the real system group's membership so the group folder grants access.
	_ = s.dir.SetGroupMembers(ctx, groupSysName, memberNames)
	return result, nil
}

func (s *Service) GrantSpace(ctx context.Context, request SpaceGrantRequest) (SpaceGrant, error) {
	if err := ctx.Err(); err != nil {
		return SpaceGrant{}, err
	}
	if request.SubjectID == "" || request.SpaceID == "" {
		return SpaceGrant{}, fmt.Errorf("subjectId and spaceId are required")
	}
	subjectType := request.SubjectType
	if subjectType == "" {
		subjectType = SubjectUser
	}
	access := request.Access
	if access == "" {
		access = AccessReadOnly
	}
	s.mu.Lock()
	now := s.now().UTC()
	var grant SpaceGrant
	for index := range s.grants {
		g := &s.grants[index]
		if g.SubjectID == request.SubjectID && g.SubjectType == subjectType && g.SpaceID == request.SpaceID {
			g.Access = access
			g.QuotaBytes = request.QuotaBytes
			g.UpdatedAt = now
			grant = *g
			break
		}
	}
	if grant.ID == "" {
		s.seq++
		grant = SpaceGrant{
			ID:          fmt.Sprintf("grant-%03d", s.seq),
			SubjectID:   request.SubjectID,
			SubjectType: subjectType,
			SpaceID:     request.SpaceID,
			Access:      access,
			QuotaBytes:  request.QuotaBytes,
			CreatedAt:   now,
			UpdatedAt:   now,
		}
		s.grants = append(s.grants, grant)
	}
	spec := s.grantSpecLocked(grant)
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return grant, err
	}
	// Project the grant onto the filesystem as a POSIX ACL (best-effort).
	if s.prov != nil {
		_ = s.prov.ApplyGrant(ctx, spec)
	}
	return grant, nil
}

// grantSpecLocked resolves a grant to a filesystem ACL spec. Caller holds s.mu.
func (s *Service) grantSpecLocked(grant SpaceGrant) GrantSpec {
	subjectName := grant.SubjectID
	if grant.SubjectType == SubjectUser {
		if idx, ok := s.findUserLocked(grant.SubjectID); ok {
			subjectName = s.users[idx].Username
		}
	}
	// For groups the SubjectID is the HiGoOS group ID, which is also the system
	// group name (see CreateGroup).
	return GrantSpec{
		SubjectName: subjectName,
		SubjectKind: grant.SubjectType,
		SpaceDir:    grant.SpaceID,
		Access:      grant.Access,
	}
}

func (s *Service) DeleteSpaceGrants(ctx context.Context, spaceID string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	if strings.TrimSpace(spaceID) == "" {
		return 0, fmt.Errorf("spaceId is required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	removed := 0
	nextGrants := s.grants[:0]
	for _, grant := range s.grants {
		if grant.SpaceID == spaceID {
			removed++
			continue
		}
		nextGrants = append(nextGrants, grant)
	}
	s.grants = nextGrants
	if removed == 0 {
		return 0, nil
	}
	return removed, s.saveLocked()
}

func (s *Service) DeleteGrant(ctx context.Context, id string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("grant id is required")
	}
	s.mu.Lock()
	index := -1
	for i, grant := range s.grants {
		if grant.ID == id {
			index = i
			break
		}
	}
	if index < 0 {
		s.mu.Unlock()
		return fmt.Errorf("grant not found: %s", id)
	}
	spec := s.grantSpecLocked(s.grants[index])
	s.grants = append(s.grants[:index], s.grants[index+1:]...)
	err := s.saveLocked()
	s.mu.Unlock()
	if err != nil {
		return err
	}
	if s.prov != nil {
		_ = s.prov.RemoveGrant(ctx, spec)
	}
	return nil
}

// VerifyPassword authenticates a username/password against the directory and
// returns the matching active user. The error is intentionally generic so the
// caller cannot distinguish "no such user" from "wrong password".
func (s *Service) VerifyPassword(ctx context.Context, username, password string) (User, error) {
	if err := ctx.Err(); err != nil {
		return User{}, err
	}
	// Allow any real OS account to authenticate, materializing its record on
	// first login if HiGoOS hasn't seen it yet.
	s.ensureSystemUser(ctx, strings.TrimSpace(username))
	s.mu.RLock()
	index, ok := s.findUserByUsernameLocked(strings.TrimSpace(username))
	var user User
	if ok {
		user = s.users[index]
	}
	s.mu.RUnlock()
	if !ok {
		return User{}, ErrInvalidCredentials
	}
	if user.Status != StatusActive {
		return User{}, ErrAccountInactive
	}
	valid, err := s.dir.VerifyPassword(ctx, user.Username, password)
	if err != nil {
		return User{}, err
	}
	if !valid {
		return User{}, ErrInvalidCredentials
	}
	return user, nil
}

// SetUserPassword updates a user's credential (used by self-service change-
// password). It validates strength and mirrors to the directory.
func (s *Service) SetUserPassword(ctx context.Context, userID, password string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if err := validatePassword(password); err != nil {
		return err
	}
	s.mu.RLock()
	index, ok := s.findUserLocked(userID)
	var username string
	if ok {
		username = s.users[index].Username
	}
	s.mu.RUnlock()
	if !ok {
		return fmt.Errorf("user not found: %s", userID)
	}
	return s.dir.SetPassword(ctx, username, password)
}

// LockUser marks a user locked (status + host lock). Used by the failed-login
// lockout policy.
func (s *Service) LockUser(ctx context.Context, userID string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	index, ok := s.findUserLocked(userID)
	if !ok {
		return fmt.Errorf("user not found: %s", userID)
	}
	s.users[index].Status = StatusLocked
	s.users[index].UpdatedAt = s.now().UTC()
	username := s.users[index].Username
	if err := s.dir.SetLocked(ctx, username, true); err != nil {
		return err
	}
	return s.saveLocked()
}

// GetUserByUsername returns a user by username.
func (s *Service) GetUserByUsername(ctx context.Context, username string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index, ok := s.findUserByUsernameLocked(strings.TrimSpace(username))
	if !ok {
		return User{}, false
	}
	return s.users[index], true
}

// GetUser returns a user by ID.
func (s *Service) GetUser(ctx context.Context, userID string) (User, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index, ok := s.findUserLocked(userID)
	if !ok {
		return User{}, false
	}
	return s.users[index], true
}

// BootstrapAdmin ensures the seed admin has a credential on first run. It
// returns the freshly-generated password (to be logged once) or "" when a
// credential already exists. A fixed password may be supplied for imaging.
func (s *Service) BootstrapAdmin(ctx context.Context, fixed string) (string, error) {
	s.mu.RLock()
	var admin User
	found := false
	for _, u := range s.users {
		if u.Role == RoleAdmin {
			admin = u
			found = true
			break
		}
	}
	s.mu.RUnlock()
	if !found {
		return "", nil
	}
	if s.dir.HasCredential(ctx, admin.Username) {
		return "", nil
	}
	password := strings.TrimSpace(fixed)
	if password == "" {
		password = genRandomPassword()
	}
	if err := s.dir.EnsureUser(ctx, IdentityRef{Username: admin.Username, DisplayName: admin.DisplayName, Admin: true}); err != nil {
		return "", err
	}
	if err := s.dir.SetPassword(ctx, admin.Username, password); err != nil {
		return "", err
	}
	return password, nil
}

// SambaUserStatus reports whether a HiGoOS user currently has a Samba account.
type SambaUserStatus struct {
	UserID   string `json:"userId"`
	Username string `json:"username"`
	InSamba  bool   `json:"inSamba"`
}

// SambaSyncReport summarizes which managed users can authenticate over SMB.
type SambaSyncReport struct {
	Users   []SambaUserStatus `json:"users"`
	Missing int               `json:"missing"`
	Note    string            `json:"note"`
}

// SyncSamba reports, per managed user, whether a Samba account exists. SHA-512
// login hashes cannot be replayed into Samba's NT hash, so missing users must
// have a password set through HiGoOS once (which runs smbpasswd); this surfaces
// exactly which users still need that.
func (s *Service) SyncSamba(ctx context.Context) (SambaSyncReport, error) {
	present, err := s.dir.SambaPresent(ctx)
	if err != nil {
		return SambaSyncReport{}, err
	}
	s.mu.RLock()
	users := append([]User(nil), s.users...)
	s.mu.RUnlock()

	report := SambaSyncReport{Users: make([]SambaUserStatus, 0, len(users))}
	for _, u := range users {
		in := present[u.Username]
		if !in {
			report.Missing++
		}
		report.Users = append(report.Users, SambaUserStatus{UserID: u.ID, Username: u.Username, InSamba: in})
	}
	if report.Missing > 0 {
		report.Note = "有用户尚未建立 SMB 凭据,请在用户中心为其重设一次密码即可启用 SMB 访问。"
	}
	return report, nil
}

// EffectiveAccess resolves a user's access to a space from the grant table,
// considering both direct user grants and grants on the groups they belong to.
// `controlled` reports whether the space has ANY grant at all — callers treat an
// uncontrolled space as open (no per-space restriction configured yet), and a
// controlled space as deny-by-default.
func (s *Service) EffectiveAccess(ctx context.Context, userID, spaceID string) (access SpaceAccess, controlled bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index, ok := s.findUserLocked(userID)
	var groups []string
	if ok {
		groups = s.users[index].Groups
	}
	best := -1
	rank := map[SpaceAccess]int{AccessReadOnly: 0, AccessReadWrite: 1, AccessManage: 2}
	for _, grant := range s.grants {
		if grant.SpaceID != spaceID {
			continue
		}
		controlled = true
		match := false
		if grant.SubjectType == SubjectUser && grant.SubjectID == userID {
			match = true
		}
		if grant.SubjectType == SubjectGroup && containsString(groups, grant.SubjectID) {
			match = true
		}
		if !match {
			continue
		}
		// Deny wins over any allow (DSM semantics): a single matching deny grant
		// blocks the subject regardless of group-inherited read/write.
		if grant.Access == AccessDeny {
			return AccessDeny, true
		}
		if r, ok := rank[grant.Access]; ok && r > best {
			best = r
			access = grant.Access
		}
	}
	return access, controlled
}

// CanWriteSpace reports whether a user may write to a space. Admins always can;
// the user's own personal/group spaces are always writable; otherwise the space
// must be either uncontrolled or grant the user read_write or manage.
func (s *Service) CanWriteSpace(ctx context.Context, userID, spaceID string) bool {
	// An explicit deny on this space overrides everything below (including the
	// personal/group auto-write).
	if access, controlled := s.EffectiveAccess(ctx, userID, spaceID); controlled && access == AccessDeny {
		return false
	}
	if user, ok := s.GetUser(ctx, userID); ok {
		if user.Role == RoleAdmin {
			return true
		}
		// Personal home ("homes" / matches own username) and own group folders.
		if spaceID == "homes" || spaceID == user.Username || containsString(user.Groups, spaceID) {
			return true
		}
	}
	access, controlled := s.EffectiveAccess(ctx, userID, spaceID)
	if !controlled {
		return true
	}
	return access == AccessReadWrite || access == AccessManage
}

// Entitlement is the file-visibility scope for a user, used to filter the file
// tree and gate reads. It is derived from the user's identity, group membership,
// and space grants.
type Entitlement struct {
	Username      string
	Admin         bool
	GroupDirs     []string // system group names (== HiGoOS group IDs) the user belongs to
	GrantedSpaces []string // shared-space directory names the user may see
	DeniedSpaces  []string // spaces explicitly denied (override group grants)
}

// UserEntitlements computes a user's file-visibility scope.
func (s *Service) UserEntitlements(ctx context.Context, userID string) (Entitlement, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	index, ok := s.findUserLocked(userID)
	if !ok {
		return Entitlement{}, fmt.Errorf("user not found: %s", userID)
	}
	u := s.users[index]
	ent := Entitlement{
		Username:  u.Username,
		Admin:     u.Role == RoleAdmin,
		GroupDirs: append([]string{}, u.Groups...),
	}
	// A user-level deny on a space overrides any group grant to that space.
	denied := map[string]bool{}
	for _, g := range s.grants {
		if g.Access == AccessDeny && g.SubjectType == SubjectUser && g.SubjectID == userID {
			denied[g.SpaceID] = true
		}
	}
	seen := map[string]bool{}
	for _, g := range s.grants {
		if g.Access == AccessDeny || denied[g.SpaceID] {
			continue
		}
		match := (g.SubjectType == SubjectUser && g.SubjectID == userID) ||
			(g.SubjectType == SubjectGroup && containsString(u.Groups, g.SubjectID))
		if match && !seen[g.SpaceID] {
			seen[g.SpaceID] = true
			ent.GrantedSpaces = append(ent.GrantedSpaces, g.SpaceID)
		}
	}
	for space := range denied {
		ent.DeniedSpaces = append(ent.DeniedSpaces, space)
	}
	return ent, nil
}

// reconcileSystemUsers pulls the directory's OS accounts into the sidecar so
// pre-existing system users (the installer's sudo user, etc.) show up in the
// user center. It only adds missing users; existing metadata is preserved. A
// no-op on the dev backend (List returns nothing).
func (s *Service) reconcileSystemUsers(ctx context.Context) {
	if s.dir == nil {
		return
	}
	identities, err := s.dir.List(ctx)
	if err != nil || len(identities) == 0 {
		return
	}
	s.mu.Lock()
	var added []SystemIdentity
	for _, id := range identities {
		if _, ok := s.findUserByUsernameLocked(id.Username); ok {
			continue
		}
		s.users = append(s.users, s.systemUser(id))
		added = append(added, id)
	}
	if len(added) > 0 {
		_ = s.saveLocked()
	}
	s.mu.Unlock()
	// Provision personal folders for the freshly-discovered OS users outside the
	// lock (best-effort; no-op on the dev backend).
	for _, id := range added {
		if s.prov != nil {
			_ = s.prov.EnsureUserFolder(ctx, id.Username, id.UID, 0)
		}
	}
}

// ensureSystemUser materializes a single OS account into the sidecar on demand
// (used at login so any real system user can authenticate even before a full
// reconcile).
func (s *Service) ensureSystemUser(ctx context.Context, username string) {
	if s.dir == nil {
		return
	}
	s.mu.RLock()
	_, exists := s.findUserByUsernameLocked(username)
	s.mu.RUnlock()
	if exists {
		return
	}
	id, ok, err := s.dir.Lookup(ctx, username)
	if err != nil || !ok {
		return
	}
	s.mu.Lock()
	created := false
	if _, exists := s.findUserByUsernameLocked(username); !exists {
		s.users = append(s.users, s.systemUser(id))
		_ = s.saveLocked()
		created = true
	}
	s.mu.Unlock()
	if created && s.prov != nil {
		_ = s.prov.EnsureUserFolder(ctx, id.Username, id.UID, 0)
	}
}

// systemUser builds a sidecar record for an OS account. The role follows native
// group membership (admin/sudo/wheel → admin), keyed by username as the ID.
func (s *Service) systemUser(id SystemIdentity) User {
	now := s.now().UTC()
	role := RoleUser
	if id.Admin {
		role = RoleAdmin
	}
	display := id.DisplayName
	if display == "" {
		display = id.Username
	}
	return User{
		ID:          id.Username,
		Username:    id.Username,
		DisplayName: display,
		Role:        role,
		Status:      StatusActive,
		Groups:      []string{},
		CreatedAt:   now,
		UpdatedAt:   now,
	}
}

func (s *Service) seed() {
	now := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)
	s.users = []User{
		{ID: "admin", Username: "admin", DisplayName: "管理员", Role: RoleAdmin, Status: StatusActive, QuotaBytes: 0, Groups: []string{"admins"}, CreatedAt: now, UpdatedAt: now},
	}
	s.groups = []Group{
		{ID: "admins", Name: "管理员组", Description: "系统管理权限", UserIDs: []string{"admin"}, CreatedAt: now, UpdatedAt: now},
		{ID: "family", Name: "家庭成员", Description: "家庭空间访问", UserIDs: []string{}, CreatedAt: now, UpdatedAt: now},
	}
	s.grants = []SpaceGrant{}
	s.seq = 2
}

func (s *Service) findUserLocked(id string) (int, bool) {
	for index, user := range s.users {
		if user.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findUserByUsernameLocked(username string) (int, bool) {
	for index, user := range s.users {
		if strings.EqualFold(user.Username, username) {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) findGroupLocked(id string) (int, bool) {
	for index, group := range s.groups {
		if group.ID == id {
			return index, true
		}
	}
	return -1, false
}

func (s *Service) syncGroupMembershipLocked(userID string, groupIDs []string) {
	for i := range s.groups {
		if containsString(groupIDs, s.groups[i].ID) {
			if !containsString(s.groups[i].UserIDs, userID) {
				s.groups[i].UserIDs = append(s.groups[i].UserIDs, userID)
			}
		} else {
			s.groups[i].UserIDs = removeString(s.groups[i].UserIDs, userID)
		}
	}
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Seq:    s.seq,
		Users:  cloneUsers(s.users),
		Groups: cloneGroups(s.groups),
		Grants: cloneGrants(s.grants),
	})
}

func validatePassword(password string) error {
	// Complexity rules intentionally removed: any non-empty password is accepted
	// (NAS-style convenience; SMB/login accept the same value verbatim).
	if password == "" {
		return fmt.Errorf("password is required")
	}
	return nil
}

// genRandomPassword returns a strong random password that satisfies
// validatePassword (>=8 chars, letters + digits).
func genRandomPassword() string {
	const alphabet = "ABCDEFGHJKLMNPQRSTUVWXYZabcdefghijkmnpqrstuvwxyz23456789"
	const length = 20
	var b strings.Builder
	for i := 0; i < length; i++ {
		n, err := rand.Int(rand.Reader, big.NewInt(int64(len(alphabet))))
		if err != nil {
			b.WriteByte(alphabet[i%len(alphabet)])
			continue
		}
		b.WriteByte(alphabet[n.Int64()])
	}
	// Guarantee at least one digit and one letter.
	return "Hg9" + b.String()
}

func defaultRole(role UserRole) UserRole {
	if role == "" {
		return RoleUser
	}
	return role
}

func slugID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	var b strings.Builder
	for _, r := range value {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		if b.Len() > 0 && b.String()[b.Len()-1] != '-' {
			b.WriteByte('-')
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "item"
	}
	return out
}

func uniqueStrings(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func removeString(values []string, target string) []string {
	out := values[:0]
	for _, value := range values {
		if value != target {
			out = append(out, value)
		}
	}
	return out
}

func cloneUsers(users []User) []User {
	out := make([]User, 0, len(users))
	for _, user := range users {
		user.Groups = append([]string{}, user.Groups...)
		out = append(out, user)
	}
	return out
}

func cloneGroups(groups []Group) []Group {
	out := make([]Group, 0, len(groups))
	for _, group := range groups {
		group.UserIDs = append([]string{}, group.UserIDs...)
		out = append(out, group)
	}
	return out
}

func cloneGrants(grants []SpaceGrant) []SpaceGrant {
	return append([]SpaceGrant{}, grants...)
}
