// Package sharedfolders is the single source of truth for "shared folder →
// user/group permissions → SMB/NFS service". It owns only the folder registry;
// the actual ACLs, Samba accounts and protocol exports live in the accounts,
// storage and protocols domains, which this facade orchestrates so the
// permission table is the one place access is defined.
package sharedfolders

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/protocols"
	"higoos/server-go/internal/state"
	"higoos/server-go/internal/storage"
)

const confirmationTTL = 5 * time.Minute

// Ports are the narrow slices of the backing services this facade needs (kept
// as interfaces so it can be unit-tested with fakes). The concrete
// *accounts.Service / *storage.Service / *protocols.Service satisfy them.
type AccountsPort interface {
	Summary(ctx context.Context) (accounts.Summary, error)
	GrantSpace(ctx context.Context, req accounts.SpaceGrantRequest) (accounts.SpaceGrant, error)
	DeleteGrant(ctx context.Context, id string) error
	DeleteSpaceGrants(ctx context.Context, spaceID string) (int, error)
	GetUser(ctx context.Context, userID string) (accounts.User, bool)
}

type StoragePort interface {
	Spaces(ctx context.Context) ([]storage.StorageSpace, error)
}

type ProtocolsPort interface {
	UpsertShare(ctx context.Context, key protocols.ProtocolKey, req protocols.CreateShareRequest) (protocols.Share, error)
	RemoveShareByPath(ctx context.Context, key protocols.ProtocolKey, path string, actor ...string) error
	Shares(ctx context.Context, key protocols.ProtocolKey) ([]protocols.Share, error)
}

type Deps struct {
	Accounts   AccountsPort
	Storage    StoragePort
	Protocols  ProtocolsPort
	NASRoot    string
	ServerHost string
}

type Service struct {
	mu         sync.Mutex
	seq        int
	folders    []SharedFolder
	pending    map[string]pendingDelete
	now        func() time.Time
	statePath  string
	nasRoot    string
	serverHost string
	accounts   AccountsPort
	storage    StoragePort
	protocols  ProtocolsPort
	runner     cmdRunner
}

// cmdRunner shells out for the host-level advanced ops (btrfs/zfs/recycle). It is
// injectable so tests can run without a real host.
type cmdRunner func(ctx context.Context, name string, args ...string) ([]byte, error)

type pendingDelete struct {
	FolderID       string    `json:"folderId"`
	ConfirmationID string    `json:"confirmationId"`
	CreatedAt      time.Time `json:"createdAt"`
}

type snapshot struct {
	Seq     int                      `json:"seq"`
	Folders []SharedFolder           `json:"folders"`
	Pending map[string]pendingDelete `json:"pending"`
}

func NewService(deps Deps) *Service {
	host := strings.TrimSpace(deps.ServerHost)
	if host == "" {
		host = "nas.local"
	}
	return &Service{
		folders:    []SharedFolder{},
		pending:    map[string]pendingDelete{},
		now:        func() time.Time { return time.Now().UTC() },
		nasRoot:    deps.NASRoot,
		serverHost: host,
		accounts:   deps.Accounts,
		storage:    deps.Storage,
		protocols:  deps.Protocols,
		runner:     runSharedCommand,
	}
}

func NewServiceWithStateDir(deps Deps, stateDir string) (*Service, error) {
	svc := NewService(deps)
	if stateDir == "" {
		return svc, nil
	}
	svc.statePath = filepath.Join(stateDir, "sharedfolders.json")
	var persisted snapshot
	if err := state.LoadJSON(svc.statePath, &persisted); err != nil {
		return nil, err
	}
	svc.seq = persisted.Seq
	svc.folders = append([]SharedFolder(nil), persisted.Folders...)
	if len(persisted.Pending) > 0 {
		svc.pending = persisted.Pending
	}
	return svc, nil
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{
		Seq:     s.seq,
		Folders: append([]SharedFolder(nil), s.folders...),
		Pending: s.pending,
	})
}

// List returns every shared folder enriched with its live permission table and
// service status.
func (s *Service) List(ctx context.Context) ([]SharedFolderView, error) {
	s.mu.Lock()
	folders := append([]SharedFolder(nil), s.folders...)
	s.mu.Unlock()

	spaces, err := s.spaceIndex(ctx)
	if err != nil {
		return nil, err
	}
	summary, err := s.accounts.Summary(ctx)
	if err != nil {
		return nil, err
	}
	views := make([]SharedFolderView, 0, len(folders))
	for _, f := range folders {
		views = append(views, s.viewFor(ctx, f, spaces, summary))
	}
	return views, nil
}

// Create registers a new shared folder (a space root or a sub-folder) and
// creates the backing directory.
func (s *Service) Create(ctx context.Context, req CreateRequest) (SharedFolderView, error) {
	if err := ctx.Err(); err != nil {
		return SharedFolderView{}, err
	}
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return SharedFolderView{}, fmt.Errorf("folder name is required")
	}
	spaces, err := s.spaceIndex(ctx)
	if err != nil {
		return SharedFolderView{}, err
	}
	space, ok := spaces[req.SpaceID]
	if !ok {
		return SharedFolderView{}, fmt.Errorf("storage space not found: %s", req.SpaceID)
	}
	rel := cleanRel(req.RelPath)
	absPath := filepath.Join(space.MountPath, rel)
	if !underDir(space.MountPath, absPath) {
		return SharedFolderView{}, fmt.Errorf("folder path escapes its storage space")
	}
	dirKey, err := s.dirKey(absPath)
	if err != nil {
		return SharedFolderView{}, err
	}

	// Host side: create the backing directory (best-effort; dev hosts without a
	// real NAS root skip it). On a Btrfs space the folder is created as a
	// subvolume so per-folder quota + snapshots are possible (fnOS behavior);
	// otherwise a plain directory.
	subvolume := false
	if strings.TrimSpace(s.nasRoot) != "" {
		if strings.EqualFold(string(space.FileSystem), "btrfs") && s.btrfsSubvolCreate(ctx, absPath) == nil {
			subvolume = true
		} else if err := os.MkdirAll(absPath, 0o2770); err != nil {
			return SharedFolderView{}, fmt.Errorf("create folder %s: %w", absPath, err)
		}
	}

	s.mu.Lock()
	for _, f := range s.folders {
		if f.DirKey == dirKey {
			s.mu.Unlock()
			return SharedFolderView{}, fmt.Errorf("a shared folder already exists for this directory")
		}
	}
	s.seq++
	folder := SharedFolder{
		ID:        fmt.Sprintf("sf-%03d", s.seq),
		Name:      name,
		SpaceID:   req.SpaceID,
		RelPath:   rel,
		DirKey:    dirKey,
		Subvolume: subvolume,
		CreatedAt: s.now(),
		CreatedBy: req.Actor,
	}
	s.folders = append(s.folders, folder)
	if err := s.saveLocked(); err != nil {
		s.folders = s.folders[:len(s.folders)-1]
		s.mu.Unlock()
		return SharedFolderView{}, err
	}
	s.mu.Unlock()

	summary, err := s.accounts.Summary(ctx)
	if err != nil {
		return SharedFolderView{}, err
	}
	return s.viewFor(ctx, folder, spaces, summary), nil
}

// SetPermissions reconciles a folder's permission table: it diffs the requested
// entries against the existing grants (writing/removing setfacl ACLs via the
// accounts domain) and re-derives the protocol exports.
func (s *Service) SetPermissions(ctx context.Context, folderID string, req SetPermissionsRequest) (SharedFolderView, error) {
	folder, ok := s.folderByID(folderID)
	if !ok {
		return SharedFolderView{}, fmt.Errorf("shared folder not found: %s", folderID)
	}
	summary, err := s.accounts.Summary(ctx)
	if err != nil {
		return SharedFolderView{}, err
	}

	desired := map[string]string{} // subjectKey -> ui access
	for _, e := range req.Entries {
		if e.Access == AccessNone || e.Access == "" {
			continue
		}
		st := e.SubjectType
		if st == "" {
			st = accounts.SubjectUser
		}
		desired[string(st)+"/"+e.SubjectID] = e.Access
	}
	current := map[string]accounts.SpaceGrant{}
	for _, g := range summary.Grants {
		if g.SpaceID == folder.DirKey {
			current[string(g.SubjectType)+"/"+g.SubjectID] = g
		}
	}

	// Adds / changes.
	for key, access := range desired {
		st, id := splitKey(key)
		existing, has := current[key]
		if has && uiAccess(existing.Access) == access {
			continue
		}
		if _, err := s.accounts.GrantSpace(ctx, accounts.SpaceGrantRequest{
			SubjectID:   id,
			SubjectType: st,
			SpaceID:     folder.DirKey,
			Access:      toAccountsAccess(access),
		}); err != nil {
			return SharedFolderView{}, err
		}
	}
	// Removals.
	for key, grant := range current {
		if _, keep := desired[key]; keep {
			continue
		}
		if err := s.accounts.DeleteGrant(ctx, grant.ID); err != nil {
			return SharedFolderView{}, err
		}
	}

	if err := s.deriveAll(ctx, folder); err != nil {
		return SharedFolderView{}, err
	}
	return s.refresh(ctx, folder)
}

// SubjectFolderPermission is one (folder, access) pair for a single subject.
type SubjectFolderPermission struct {
	FolderID string `json:"folderId"`
	Access   string `json:"access"` // none / read / read_write / deny
}

// SetSubjectPermissions sets one subject's (user or group) access across many
// folders at once — the "shared-folder permissions" tab of a user/group editor.
// For each folder it replaces just this subject's row and reconciles, leaving
// other subjects' permissions untouched.
func (s *Service) SetSubjectPermissions(ctx context.Context, subjectType accounts.SubjectType, subjectID string, perms []SubjectFolderPermission, actor string) error {
	if subjectID == "" {
		return fmt.Errorf("subjectId is required")
	}
	if subjectType == "" {
		subjectType = accounts.SubjectUser
	}
	for _, p := range perms {
		folder, ok := s.folderByID(p.FolderID)
		if !ok {
			return fmt.Errorf("shared folder not found: %s", p.FolderID)
		}
		summary, err := s.accounts.Summary(ctx)
		if err != nil {
			return err
		}
		// Rebuild this folder's full entry set: keep every other subject, set ours.
		entries := []PermissionEntry{}
		for _, g := range summary.Grants {
			if g.SpaceID != folder.DirKey {
				continue
			}
			if g.SubjectType == subjectType && g.SubjectID == subjectID {
				continue // replaced below
			}
			entries = append(entries, PermissionEntry{SubjectType: g.SubjectType, SubjectID: g.SubjectID, Access: uiAccess(g.Access)})
		}
		if p.Access != "" && p.Access != AccessNone {
			entries = append(entries, PermissionEntry{SubjectType: subjectType, SubjectID: subjectID, Access: p.Access})
		}
		if _, err := s.SetPermissions(ctx, folder.ID, SetPermissionsRequest{Entries: entries, Actor: actor}); err != nil {
			return err
		}
	}
	return nil
}

// SetService toggles a protocol export for a folder and re-derives it.
func (s *Service) SetService(ctx context.Context, folderID string, req SetServiceRequest) (SharedFolderView, error) {
	s.mu.Lock()
	idx := s.indexOf(folderID)
	if idx < 0 {
		s.mu.Unlock()
		return SharedFolderView{}, fmt.Errorf("shared folder not found: %s", folderID)
	}
	switch protocols.ProtocolKey(req.Protocol) {
	case protocols.ProtocolSMB:
		s.folders[idx].SMBEnabled = req.Enabled
		s.folders[idx].Guest = req.Guest
	case protocols.ProtocolNFS:
		s.folders[idx].NFSEnabled = req.Enabled
	default:
		s.mu.Unlock()
		return SharedFolderView{}, fmt.Errorf("unsupported protocol: %s", req.Protocol)
	}
	folder := s.folders[idx]
	if err := s.saveLocked(); err != nil {
		s.mu.Unlock()
		return SharedFolderView{}, err
	}
	s.mu.Unlock()

	if err := s.deriveAll(ctx, folder); err != nil {
		return SharedFolderView{}, err
	}
	return s.refresh(ctx, folder)
}

// PreviewDelete registers a single-use confirmation for removing a shared folder.
func (s *Service) PreviewDelete(ctx context.Context, folderID, actor string) (DeletePreview, error) {
	folder, ok := s.folderByID(folderID)
	if !ok {
		return DeletePreview{}, fmt.Errorf("shared folder not found: %s", folderID)
	}
	abs := s.absPath(ctx, folder)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	cid := fmt.Sprintf("sf-confirm-%03d", s.seq)
	s.pending[cid] = pendingDelete{FolderID: folder.ID, ConfirmationID: cid, CreatedAt: s.now()}
	if err := s.saveLocked(); err != nil {
		return DeletePreview{}, err
	}
	return DeletePreview{
		FolderID:             folder.ID,
		Name:                 folder.Name,
		AbsPath:              abs,
		Impact:               fmt.Sprintf("将移除共享文件夹「%s」(%s):清除其全部权限授权与 SMB/NFS 共享。可选删除目录及其数据(不可恢复)。", folder.Name, abs),
		ConfirmationID:       cid,
		RequiresConfirmation: true,
	}, nil
}

// ConfirmDelete validates the token, tears down shares + grants, removes the
// record, and optionally deletes the directory.
func (s *Service) ConfirmDelete(ctx context.Context, req ConfirmDeleteRequest) error {
	s.mu.Lock()
	pending, ok := s.pending[req.ConfirmationID]
	if !ok {
		s.mu.Unlock()
		return fmt.Errorf("no pending deletion for confirmation: %s", req.ConfirmationID)
	}
	delete(s.pending, req.ConfirmationID)
	if s.now().Sub(pending.CreatedAt) > confirmationTTL {
		_ = s.saveLocked()
		s.mu.Unlock()
		return fmt.Errorf("confirmation %s has expired", req.ConfirmationID)
	}
	idx := s.indexOf(pending.FolderID)
	if idx < 0 {
		_ = s.saveLocked()
		s.mu.Unlock()
		return fmt.Errorf("shared folder not found: %s", pending.FolderID)
	}
	folder := s.folders[idx]
	s.mu.Unlock()

	abs := s.absPath(ctx, folder)
	// Remove derived exports first, then ACL grants.
	_ = s.protocols.RemoveShareByPath(ctx, protocols.ProtocolSMB, abs, req.Actor)
	_ = s.protocols.RemoveShareByPath(ctx, protocols.ProtocolNFS, abs, req.Actor)
	if _, err := s.accounts.DeleteSpaceGrants(ctx, folder.DirKey); err != nil {
		return err
	}
	if req.RemoveDir && strings.TrimSpace(s.nasRoot) != "" && underDir(s.nasRoot, abs) && filepath.Clean(abs) != filepath.Clean(s.nasRoot) {
		_ = os.RemoveAll(abs)
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.indexOf(folder.ID); i >= 0 {
		s.folders = append(s.folders[:i], s.folders[i+1:]...)
	}
	return s.saveLocked()
}

// --- derivation & enrichment ------------------------------------------------

// deriveAll projects the folder's permission table onto its enabled protocol
// exports (or removes them when disabled / empty).
func (s *Service) deriveAll(ctx context.Context, folder SharedFolder) error {
	abs := s.absPath(ctx, folder)
	summary, err := s.accounts.Summary(ctx)
	if err != nil {
		return err
	}
	allowed, write, deny := s.subjectsFor(ctx, folder.DirKey, summary)

	if err := s.deriveOne(ctx, protocols.ProtocolSMB, folder, abs, allowed, write, deny); err != nil {
		return err
	}
	return s.deriveOne(ctx, protocols.ProtocolNFS, folder, abs, allowed, write, deny)
}

func (s *Service) deriveOne(ctx context.Context, key protocols.ProtocolKey, folder SharedFolder, abs string, allowed, write, deny []string) error {
	enabled := folder.SMBEnabled
	if key == protocols.ProtocolNFS {
		enabled = folder.NFSEnabled
	}
	// An enabled share needs either named users or guest access; otherwise there
	// is nothing meaningful to export, so remove any stale share.
	if !enabled || (len(allowed) == 0 && !folder.Guest) {
		return s.protocols.RemoveShareByPath(ctx, key, abs, folder.CreatedBy)
	}
	level := protocols.AccessAccount
	if len(allowed) == 0 && folder.Guest {
		level = protocols.AccessPublic
	}
	_, err := s.protocols.UpsertShare(ctx, key, protocols.CreateShareRequest{
		Name:         folder.Name,
		Path:         abs,
		AccessLevel:  level,
		AllowedUsers: allowed,
		WriteUsers:   write,
		DenyUsers:    deny,
		Recycle:      key == protocols.ProtocolSMB && folder.Recycle,
		Guest:        folder.Guest,
		Actor:        folder.CreatedBy,
	})
	return err
}

// subjectsFor resolves a folder's grants into Samba subject tokens: usernames
// for users, "@<groupID>" for groups. Denied subjects are returned separately
// (Samba "invalid users") and excluded from allowed/write.
func (s *Service) subjectsFor(ctx context.Context, dirKey string, summary accounts.Summary) (allowed, write, deny []string) {
	for _, g := range summary.Grants {
		if g.SpaceID != dirKey {
			continue
		}
		token := ""
		switch g.SubjectType {
		case accounts.SubjectGroup:
			token = "@" + g.SubjectID
		default:
			if u, ok := s.accounts.GetUser(ctx, g.SubjectID); ok && u.Username != "" {
				token = u.Username
			} else {
				token = g.SubjectID
			}
		}
		if g.Access == accounts.AccessDeny {
			deny = append(deny, token)
			continue
		}
		allowed = append(allowed, token)
		if g.Access == accounts.AccessReadWrite || g.Access == accounts.AccessManage {
			write = append(write, token)
		}
	}
	return allowed, write, deny
}

func (s *Service) viewFor(ctx context.Context, folder SharedFolder, spaces map[string]storage.StorageSpace, summary accounts.Summary) SharedFolderView {
	space := spaces[folder.SpaceID]
	abs := filepath.Join(space.MountPath, folder.RelPath)
	groupNames := map[string]string{}
	for _, gr := range summary.Groups {
		groupNames[gr.ID] = gr.Name
	}
	perms := []PermissionEntry{}
	for _, g := range summary.Grants {
		if g.SpaceID != folder.DirKey {
			continue
		}
		name := g.SubjectID
		if g.SubjectType == accounts.SubjectGroup {
			if n, ok := groupNames[g.SubjectID]; ok {
				name = n
			}
		} else if u, ok := s.accounts.GetUser(ctx, g.SubjectID); ok {
			name = u.DisplayName
			if name == "" {
				name = u.Username
			}
		}
		perms = append(perms, PermissionEntry{
			SubjectType: g.SubjectType,
			SubjectID:   g.SubjectID,
			SubjectName: name,
			Access:      uiAccess(g.Access),
		})
	}
	label := shareLabel(folder.Name)
	services := []ServiceState{
		{Protocol: protocols.ProtocolSMB, Enabled: folder.SMBEnabled, Guest: folder.Guest, MountHint: fmt.Sprintf("smb://%s/%s", s.serverHost, label)},
		{Protocol: protocols.ProtocolNFS, Enabled: folder.NFSEnabled, MountHint: fmt.Sprintf("%s:%s", s.serverHost, abs)},
	}
	fs := strings.ToLower(string(space.FileSystem))
	return SharedFolderView{
		SharedFolder: folder,
		AbsPath:      abs,
		SpaceName:    space.Name,
		FileSystem:   fs,
		AdvancedOK:   fs == "btrfs" || fs == "zfs",
		Permissions:  perms,
		Services:     services,
	}
}

func (s *Service) refresh(ctx context.Context, folder SharedFolder) (SharedFolderView, error) {
	spaces, err := s.spaceIndex(ctx)
	if err != nil {
		return SharedFolderView{}, err
	}
	summary, err := s.accounts.Summary(ctx)
	if err != nil {
		return SharedFolderView{}, err
	}
	// Reload the folder record (intent flags may have changed).
	if f, ok := s.folderByID(folder.ID); ok {
		folder = f
	}
	return s.viewFor(ctx, folder, spaces, summary), nil
}

// --- helpers ----------------------------------------------------------------

func (s *Service) spaceIndex(ctx context.Context) (map[string]storage.StorageSpace, error) {
	spaces, err := s.storage.Spaces(ctx)
	if err != nil {
		return nil, err
	}
	out := make(map[string]storage.StorageSpace, len(spaces))
	for _, sp := range spaces {
		out[sp.ID] = sp
	}
	return out, nil
}

func (s *Service) absPath(ctx context.Context, folder SharedFolder) string {
	spaces, err := s.spaceIndex(ctx)
	if err != nil {
		return folder.DirKey
	}
	return filepath.Join(spaces[folder.SpaceID].MountPath, folder.RelPath)
}

func (s *Service) dirKey(absPath string) (string, error) {
	root := strings.TrimSpace(s.nasRoot)
	if root == "" {
		// Dev hosts without a real NAS root: use the path's base segments as a key.
		return strings.TrimPrefix(filepath.Clean(absPath), "/"), nil
	}
	rel, err := filepath.Rel(root, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return "", fmt.Errorf("folder path must stay under the NAS root")
	}
	return rel, nil
}

func (s *Service) folderByID(id string) (SharedFolder, bool) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if i := s.indexOf(id); i >= 0 {
		return s.folders[i], true
	}
	return SharedFolder{}, false
}

func (s *Service) indexOf(id string) int {
	for i := range s.folders {
		if s.folders[i].ID == id {
			return i
		}
	}
	return -1
}

func splitKey(key string) (accounts.SubjectType, string) {
	parts := strings.SplitN(key, "/", 2)
	if len(parts) != 2 {
		return accounts.SubjectUser, key
	}
	return accounts.SubjectType(parts[0]), parts[1]
}

func cleanRel(rel string) string {
	rel = strings.Trim(strings.TrimSpace(rel), "/")
	if rel == "" {
		return ""
	}
	return filepath.Clean(rel)
}

func underDir(parent, child string) bool {
	parent = filepath.Clean(parent)
	child = filepath.Clean(child)
	if child == parent {
		return true
	}
	rel, err := filepath.Rel(parent, child)
	return err == nil && !strings.HasPrefix(rel, "..")
}

func shareLabel(name string) string {
	return strings.NewReplacer(" ", "_", "/", "_").Replace(strings.TrimSpace(name))
}
