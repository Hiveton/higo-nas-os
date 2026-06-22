package accounts

import "time"

type UserRole string

const (
	RoleAdmin UserRole = "admin"
	RoleUser  UserRole = "user"
	RoleGuest UserRole = "guest"
)

type UserStatus string

const (
	StatusActive   UserStatus = "active"
	StatusDisabled UserStatus = "disabled"
	StatusLocked   UserStatus = "locked"
)

type SubjectType string

const (
	SubjectUser  SubjectType = "user"
	SubjectGroup SubjectType = "group"
)

type SpaceAccess string

const (
	AccessReadOnly  SpaceAccess = "read"
	AccessReadWrite SpaceAccess = "read_write"
	AccessManage    SpaceAccess = "manage"
	// AccessDeny is an explicit deny that overrides any allow the subject would
	// otherwise inherit (e.g. via group membership) — Synology DSM's "拒绝".
	AccessDeny SpaceAccess = "deny"
)

type User struct {
	ID          string     `json:"id"`
	Username    string     `json:"username"`
	DisplayName string     `json:"displayName"`
	Role        UserRole   `json:"role"`
	Status      UserStatus `json:"status"`
	QuotaBytes  int64      `json:"quotaBytes"`
	Groups      []string   `json:"groups"`
	// HomeSpaceID is the storage space this user's personal folder lives on.
	// Empty → the system-wide default storage space.
	HomeSpaceID string    `json:"homeSpaceId,omitempty"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Group struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserIDs     []string  `json:"userIds"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type SpaceGrant struct {
	ID          string      `json:"id"`
	SubjectID   string      `json:"subjectId"`
	SubjectType SubjectType `json:"subjectType"`
	SpaceID     string      `json:"spaceId"`
	Access      SpaceAccess `json:"access"`
	QuotaBytes  int64       `json:"quotaBytes"`
	CreatedAt   time.Time   `json:"createdAt"`
	UpdatedAt   time.Time   `json:"updatedAt"`
}

type Summary struct {
	Users  []User       `json:"users"`
	Groups []Group      `json:"groups"`
	Grants []SpaceGrant `json:"grants"`
}

type CreateUserRequest struct {
	Username    string   `json:"username"`
	DisplayName string   `json:"displayName"`
	Password    string   `json:"password"`
	Role        UserRole `json:"role"`
	QuotaBytes  int64    `json:"quotaBytes"`
	Groups      []string `json:"groups"`
	HomeSpaceID string   `json:"homeSpaceId,omitempty"`
}

type UpdateUserRequest struct {
	DisplayName string     `json:"displayName"`
	Password    string     `json:"password"`
	Role        UserRole   `json:"role"`
	Status      UserStatus `json:"status"`
	QuotaBytes  *int64     `json:"quotaBytes,omitempty"`
	Groups      []string   `json:"groups"`
	HomeSpaceID *string    `json:"homeSpaceId,omitempty"`
}

type CreateGroupRequest struct {
	Name        string   `json:"name"`
	Description string   `json:"description"`
	UserIDs     []string `json:"userIds"`
}

type UpdateGroupMembersRequest struct {
	UserIDs []string `json:"userIds"`
}

type SpaceGrantRequest struct {
	SubjectID   string      `json:"subjectId"`
	SubjectType SubjectType `json:"subjectType"`
	SpaceID     string      `json:"spaceId"`
	Access      SpaceAccess `json:"access"`
	QuotaBytes  int64       `json:"quotaBytes"`
}

// LoginRequest authenticates a username/password pair, plus an optional TOTP
// code when the account has MFA enabled.
type LoginRequest struct {
	Username       string `json:"username"`
	Password       string `json:"password"`
	Code           string `json:"code"`
	RememberDevice bool   `json:"rememberDevice"`
}

// ChangePasswordRequest updates the caller's own credential.
type ChangePasswordRequest struct {
	CurrentPassword string `json:"currentPassword"`
	NewPassword     string `json:"newPassword"`
}
