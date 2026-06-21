// Package protocols manages the NAS network file-sharing protocols
// (SMB/Samba, NFS, WebDAV, DLNA). On Linux it drives the real systemd services
// and writes HiGoOS-owned configuration files; on dev hosts it uses a
// deterministic devstub. Mutating actions flow through a preview -> confirm ->
// rollback governance loop with an append-only audit trail, mirroring the
// steward and security domains.
package protocols

import (
	"time"

	"higoos/server-go/internal/audit"
)

// ProtocolKey identifies one of the supported sharing protocols.
type ProtocolKey string

const (
	ProtocolSMB    ProtocolKey = "smb"
	ProtocolNFS    ProtocolKey = "nfs"
	ProtocolWebDAV ProtocolKey = "webdav"
	ProtocolDLNA   ProtocolKey = "dlna"
)

// AccessLevel describes how a shared directory is exposed.
type AccessLevel string

const (
	AccessPublic   AccessLevel = "public"   // guest access, read/write
	AccessPassword AccessLevel = "password" // share-level password
	AccessAccount  AccessLevel = "account"  // named NAS users only
	AccessReadOnly AccessLevel = "readonly" // read-only for everyone
)

// Protocol is the desired + live state of one sharing protocol.
type Protocol struct {
	Key           ProtocolKey    `json:"key"`
	DisplayName   string         `json:"displayName"`
	Enabled       bool           `json:"enabled"`   // desired state (our toggle)
	Running       bool           `json:"running"`   // live: systemd unit active
	Installed     bool           `json:"installed"` // live: package/unit present
	MountHint     string         `json:"mountHint"` // e.g. \\host\<name>, nfs://host/<path>
	Port          int            `json:"port"`
	Compatibility string         `json:"compatibility"`
	Config        ProtocolConfig `json:"config"`
}

// ProtocolConfig holds the per-protocol settings a user can tune. Each protocol
// uses the subset relevant to it; the rest stay at their zero value.
type ProtocolConfig struct {
	// SMB
	ServerName  string `json:"serverName,omitempty"`
	Workgroup   string `json:"workgroup,omitempty"`
	MinProtocol string `json:"minProtocol,omitempty"` // SMB2 / SMB3
	GuestAccess bool   `json:"guestAccess,omitempty"`
	// NFS
	Squash         string `json:"squash,omitempty"`         // root_squash / all_squash / no_root_squash
	AllowedNetwork string `json:"allowedNetwork,omitempty"` // CIDR or *
	// WebDAV
	HTTPSEnabled bool `json:"httpsEnabled,omitempty"`
	// DLNA
	FriendlyName string `json:"friendlyName,omitempty"`
}

// Share is a single exported directory for a protocol.
type Share struct {
	ID           string      `json:"id"`
	Protocol     ProtocolKey `json:"protocol"`
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	AccessLevel  AccessLevel `json:"accessLevel"`
	AllowedUsers []string    `json:"allowedUsers,omitempty"`
	// WriteUsers is the subset of AllowedUsers granted read-write; the rest are
	// read-only. Empty preserves the legacy all-or-nothing behavior.
	WriteUsers []string `json:"writeUsers,omitempty"`
	// DenyUsers are explicitly denied (Samba "invalid users"), overriding any
	// group-level access — mirrors the filesystem deny ACL.
	DenyUsers []string `json:"denyUsers,omitempty"`
	// Recycle enables the Samba recycle VFS (deleted files go to #recycle).
	Recycle   bool      `json:"recycle,omitempty"`
	Guest     bool      `json:"guest"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"createdAt"`
	CreatedBy string    `json:"createdBy,omitempty"`
}

// ProtocolBaseConfig carries host-level knobs the adapter needs when enabling a
// protocol. Kept thin for the first cut.
type ProtocolBaseConfig struct {
	NASRoot    string
	ServerName string
	Workgroup  string
}

// changeKind enumerates the governed mutations.
type changeKind string

const (
	changeEnable       changeKind = "enable"
	changeDisable      changeKind = "disable"
	changeShareCreate  changeKind = "share-create"
	changeShareDelete  changeKind = "share-delete"
	changeConfigUpdate changeKind = "config-update"
)

// pendingChange is an in-flight, previewed-but-not-applied mutation keyed by its
// confirmation id (mirrors steward.previews, extended to carry the operation).
type pendingChange struct {
	Kind           changeKind      `json:"kind"`
	Protocol       ProtocolKey     `json:"protocol"`
	Share          Share           `json:"share,omitempty"`
	ConfirmationID string          `json:"confirmationId"`
	RollbackID     string          `json:"rollbackId"`
	Risk           audit.RiskLevel `json:"risk"`
	Impact         string          `json:"impact"`
	Actor          string          `json:"actor,omitempty"`
}

// ProtocolPreview is returned by the *preview endpoints: it states impact + risk
// and, for medium/high-risk actions, the confirmation id required to proceed.
type ProtocolPreview struct {
	Kind                 changeKind      `json:"kind"`
	Protocol             ProtocolKey     `json:"protocol"`
	Impact               string          `json:"impact"`
	Risk                 audit.RiskLevel `json:"risk"`
	RiskLabel            string          `json:"riskLabel"`
	RequiresConfirmation bool            `json:"requiresConfirmation"`
	ConfirmationID       string          `json:"confirmationId,omitempty"`
	RollbackID           string          `json:"rollbackId,omitempty"`
}

// AuditEntry is one append-only governance record. It carries enough of the
// applied change (Kind/Protocol/Share) to be reversed on rollback.
type AuditEntry struct {
	ID             string            `json:"id"`
	Event          string            `json:"event"`
	Actor          string            `json:"actor,omitempty"`
	Risk           audit.RiskLevel   `json:"risk"`
	RiskLabel      string            `json:"riskLabel"`
	Result         audit.AuditResult `json:"result"`
	Kind           changeKind        `json:"kind,omitempty"`
	Protocol       ProtocolKey       `json:"protocol,omitempty"`
	Share          *Share            `json:"share,omitempty"`
	Config         *ProtocolConfig   `json:"config,omitempty"` // previous config, for config-update rollback
	ConfirmationID string            `json:"confirmationId,omitempty"`
	RollbackID     string            `json:"rollbackId,omitempty"`
	Reverted       bool              `json:"reverted"`
	Rollback       string            `json:"rollback,omitempty"`
	Time           time.Time         `json:"time"`
}

// --- request/response payloads ---------------------------------------------

// CreateShareRequest is the body of the share preview endpoint.
type CreateShareRequest struct {
	Name         string      `json:"name"`
	Path         string      `json:"path"`
	AccessLevel  AccessLevel `json:"accessLevel"`
	AllowedUsers []string    `json:"allowedUsers,omitempty"`
	WriteUsers   []string    `json:"writeUsers,omitempty"`
	DenyUsers    []string    `json:"denyUsers,omitempty"`
	Recycle      bool        `json:"recycle,omitempty"`
	Guest        bool        `json:"guest"`
	Actor        string      `json:"actor,omitempty"`
}

// ActorRequest is the body for enable/disable previews and share-delete previews.
type ActorRequest struct {
	Actor string `json:"actor,omitempty"`
}

// ConfigUpdateRequest updates one protocol's settings (direct apply + audit).
type ConfigUpdateRequest struct {
	Config ProtocolConfig `json:"config"`
	Actor  string         `json:"actor,omitempty"`
}

// ConfirmRequest applies a previously previewed change.
type ConfirmRequest struct {
	ConfirmationID string `json:"confirmationId"`
	Actor          string `json:"actor,omitempty"`
}

// RollbackRequest reverses a confirmed change by its audit id.
type RollbackRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// ConfirmResult is returned by the *confirm endpoints.
type ConfirmResult struct {
	Protocol *Protocol  `json:"protocol,omitempty"`
	Share    *Share     `json:"share,omitempty"`
	Audit    AuditEntry `json:"audit"`
}

// --- helpers ----------------------------------------------------------------

func riskLabel(level audit.RiskLevel) string {
	switch level {
	case audit.RiskLow:
		return "低风险"
	case audit.RiskMedium:
		return "中风险"
	case audit.RiskHigh:
		return "高风险"
	default:
		return string(level)
	}
}

func requiresConfirmation(level audit.RiskLevel) bool {
	return level == audit.RiskMedium || level == audit.RiskHigh
}

// shareRisk classifies the risk of exposing a directory at a given access level.
func shareRisk(level AccessLevel) audit.RiskLevel {
	switch level {
	case AccessPublic, AccessPassword:
		return audit.RiskHigh
	case AccessAccount, AccessReadOnly:
		return audit.RiskMedium
	default:
		return audit.RiskMedium
	}
}

func cloneProtocols(in []Protocol) []Protocol {
	return append([]Protocol(nil), in...)
}

func cloneShare(in Share) Share {
	in.AllowedUsers = append([]string(nil), in.AllowedUsers...)
	in.WriteUsers = append([]string(nil), in.WriteUsers...)
	in.DenyUsers = append([]string(nil), in.DenyUsers...)
	return in
}

func cloneShares(in []Share) []Share {
	out := make([]Share, 0, len(in))
	for _, s := range in {
		out = append(out, cloneShare(s))
	}
	return out
}

func cloneAudit(in []AuditEntry) []AuditEntry {
	out := make([]AuditEntry, 0, len(in))
	for _, e := range in {
		if e.Share != nil {
			s := cloneShare(*e.Share)
			e.Share = &s
		}
		if e.Config != nil {
			c := *e.Config
			e.Config = &c
		}
		out = append(out, e)
	}
	return out
}

func clonePending(in map[string]pendingChange) map[string]pendingChange {
	out := make(map[string]pendingChange, len(in))
	for k, v := range in {
		v.Share = cloneShare(v.Share)
		out[k] = v
	}
	return out
}

func accessLabel(level AccessLevel) string {
	switch level {
	case AccessPublic:
		return "公开访问"
	case AccessPassword:
		return "密码访问"
	case AccessAccount:
		return "指定账号"
	case AccessReadOnly:
		return "只读访问"
	default:
		return string(level)
	}
}
