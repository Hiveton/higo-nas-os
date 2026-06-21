package sharedfolders

import (
	"time"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/protocols"
)

// SharedFolder is a managed directory (a storage-space root or a sub-folder)
// whose access is governed by a single permission table and whose file-service
// exports (SMB/NFS) are auto-derived from it.
type SharedFolder struct {
	ID         string    `json:"id"`
	Name       string    `json:"name"`
	SpaceID    string    `json:"spaceId"`
	RelPath    string    `json:"relPath"`           // "" = space root
	DirKey     string    `json:"dirKey"`            // NAS-root-relative; == accounts grant SpaceID
	SMBEnabled bool      `json:"smbEnabled"`        // file-service intent (persisted)
	NFSEnabled bool      `json:"nfsEnabled"`
	Guest      bool      `json:"guest"`
	// --- advanced (fnOS/DSM 共享文件夹 高级设置) ---
	Recycle    bool      `json:"recycle"`           // recycle bin (#recycle via Samba vfs)
	QuotaBytes int64     `json:"quotaBytes"`        // per-folder quota (Btrfs qgroup / ZFS)
	Encrypted  bool      `json:"encrypted"`         // folder-level encryption intent
	Subvolume  bool      `json:"subvolume"`         // created as a Btrfs subvolume (enables quota/snapshot)
	CreatedAt  time.Time `json:"createdAt"`
	CreatedBy  string    `json:"createdBy,omitempty"`
}

// Snapshot is one point-in-time snapshot of a folder (Btrfs/ZFS).
type Snapshot struct {
	Name      string `json:"name"`
	CreatedAt string `json:"createdAt"`
}

// PermissionEntry is one row of a folder's permission table. Access is the UI
// vocabulary: "none" (no grant) / "read" / "read_write".
type PermissionEntry struct {
	SubjectType accounts.SubjectType `json:"subjectType"`
	SubjectID   string               `json:"subjectId"`
	SubjectName string               `json:"subjectName,omitempty"`
	Access      string               `json:"access"`
}

// ServiceState reports a protocol export's status for a folder.
type ServiceState struct {
	Protocol  protocols.ProtocolKey `json:"protocol"`
	Enabled   bool                  `json:"enabled"`
	Guest     bool                  `json:"guest"`
	MountHint string                `json:"mountHint"`
}

// SharedFolderView is the enriched record returned to clients.
type SharedFolderView struct {
	SharedFolder
	AbsPath     string            `json:"absPath"`
	SpaceName   string            `json:"spaceName"`
	FileSystem  string            `json:"fileSystem"`           // ext4/btrfs/zfs of the backing space
	AdvancedOK  bool              `json:"advancedOk"`           // quota+snapshot supported (btrfs/zfs)
	Permissions []PermissionEntry `json:"permissions"`
	Services    []ServiceState    `json:"services"`
	Snapshots   []Snapshot        `json:"snapshots,omitempty"`
}

// --- request payloads -------------------------------------------------------

type CreateRequest struct {
	SpaceID string `json:"spaceId"`
	Name    string `json:"name"`
	RelPath string `json:"relPath"`
	Actor   string `json:"actor"`
}

type SetPermissionsRequest struct {
	Entries []PermissionEntry `json:"entries"`
	Actor   string            `json:"actor"`
}

type SetServiceRequest struct {
	Protocol string `json:"protocol"`
	Enabled  bool   `json:"enabled"`
	Guest    bool   `json:"guest"`
	Actor    string `json:"actor"`
}

// SetAdvancedRequest updates a folder's advanced settings (recycle bin, quota,
// encryption intent). Pointers so omitted fields are left unchanged.
type SetAdvancedRequest struct {
	Recycle    *bool  `json:"recycle,omitempty"`
	QuotaBytes *int64 `json:"quotaBytes,omitempty"`
	Encrypted  *bool  `json:"encrypted,omitempty"`
	Actor      string `json:"actor,omitempty"`
}

type SnapshotRequest struct {
	Name  string `json:"name,omitempty"`
	Actor string `json:"actor,omitempty"`
}

type DeletePreview struct {
	FolderID             string `json:"folderId"`
	Name                 string `json:"name"`
	AbsPath              string `json:"absPath"`
	Impact               string `json:"impact"`
	ConfirmationID       string `json:"confirmationId"`
	RequiresConfirmation bool   `json:"requiresConfirmation"`
}

type ConfirmDeleteRequest struct {
	ConfirmationID string `json:"confirmationId"`
	RemoveDir      bool   `json:"removeDir"`
	Actor          string `json:"actor"`
}

// access vocabulary helpers (UI <-> accounts domain). DSM model:
// none (no row) / read / read_write / deny (explicit, overrides group allow).
const (
	AccessNone      = "none"
	AccessRead      = "read"
	AccessReadWrite = "read_write"
	AccessDeny      = "deny"
)

func uiAccess(a accounts.SpaceAccess) string {
	switch a {
	case accounts.AccessDeny:
		return AccessDeny
	case accounts.AccessReadOnly:
		return AccessRead
	default:
		return AccessReadWrite // read_write or manage
	}
}

func toAccountsAccess(ui string) accounts.SpaceAccess {
	switch ui {
	case AccessDeny:
		return accounts.AccessDeny
	case AccessRead:
		return accounts.AccessReadOnly
	default:
		return accounts.AccessReadWrite
	}
}
