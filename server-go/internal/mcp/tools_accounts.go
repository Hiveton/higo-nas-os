package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type AccountsUserCreateInput struct {
	Username    string   `json:"username" jsonschema:"login username for the new user"`
	DisplayName string   `json:"displayName" jsonschema:"display name for the new user"`
	Password    string   `json:"password" jsonschema:"initial password for the new user"`
	Role        string   `json:"role" jsonschema:"user role (admin, user, guest)"`
	QuotaBytes  int64    `json:"quotaBytes" jsonschema:"storage quota in bytes"`
	Groups      []string `json:"groups,omitempty" jsonschema:"group ids the user belongs to"`
}

type AccountsUserUpdateInput struct {
	ID          string   `json:"id" jsonschema:"user id to update"`
	DisplayName string   `json:"displayName,omitempty" jsonschema:"new display name"`
	Password    string   `json:"password,omitempty" jsonschema:"new password"`
	Role        string   `json:"role,omitempty" jsonschema:"new user role (admin, user, guest)"`
	Status      string   `json:"status,omitempty" jsonschema:"new account status (active, disabled, locked)"`
	QuotaBytes  *int64   `json:"quotaBytes,omitempty" jsonschema:"new storage quota in bytes"`
	Groups      []string `json:"groups,omitempty" jsonschema:"new set of group ids"`
}

type AccountsUserDeleteInput struct {
	ID string `json:"id" jsonschema:"user id to delete"`
}

type AccountsGroupCreateInput struct {
	Name        string   `json:"name" jsonschema:"name of the new group"`
	Description string   `json:"description,omitempty" jsonschema:"description of the group"`
	UserIDs     []string `json:"userIds,omitempty" jsonschema:"user ids to add as members"`
}

type AccountsGroupMembersUpdateInput struct {
	ID      string   `json:"id" jsonschema:"group id to update"`
	UserIDs []string `json:"userIds" jsonschema:"full replacement set of member user ids"`
}

type AccountsGrantCreateInput struct {
	SubjectID   string `json:"subjectId" jsonschema:"user or group id receiving access"`
	SubjectType string `json:"subjectType" jsonschema:"subject type (user or group)"`
	SpaceID     string `json:"spaceId" jsonschema:"space id to grant access to"`
	Access      string `json:"access" jsonschema:"access level (read, read_write, manage)"`
	QuotaBytes  int64  `json:"quotaBytes" jsonschema:"quota in bytes for this grant"`
}

type AccountsGrantDeleteInput struct {
	ID string `json:"id" jsonschema:"grant id to revoke"`
}

func registerAccounts(r *registry) {
	addTool(r, "accounts", "higo.accounts.summary",
		"Get a summary of users, groups and space grants.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AccountsSummary(ctx)
		})

	addTool(r, "accounts", "higo.accounts.users.list",
		"List user accounts.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AccountUsers(ctx)
		})

	addTool(r, "accounts", "higo.accounts.users.create",
		"Create a new user account.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AccountsUserCreateInput) (json.RawMessage, error) {
			return c.AccountUserCreate(ctx, in)
		})

	addTool(r, "accounts", "higo.accounts.users.update",
		"Update an existing user account.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AccountsUserUpdateInput) (json.RawMessage, error) {
			body := map[string]any{
				"displayName": in.DisplayName,
				"password":    in.Password,
				"role":        in.Role,
				"status":      in.Status,
				"groups":      in.Groups,
			}
			if in.QuotaBytes != nil {
				body["quotaBytes"] = *in.QuotaBytes
			}
			return c.AccountUserUpdate(ctx, in.ID, body)
		})

	addTool(r, "accounts", "higo.accounts.users.delete",
		"Delete a user account.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AccountsUserDeleteInput) (json.RawMessage, error) {
			return c.AccountUserDelete(ctx, in.ID)
		})

	addTool(r, "accounts", "higo.accounts.groups.list",
		"List user groups.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AccountGroups(ctx)
		})

	addTool(r, "accounts", "higo.accounts.groups.create",
		"Create a new user group.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AccountsGroupCreateInput) (json.RawMessage, error) {
			return c.AccountGroupCreate(ctx, in)
		})

	addTool(r, "accounts", "higo.accounts.groups.members-update",
		"Replace the member list of a group.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AccountsGroupMembersUpdateInput) (json.RawMessage, error) {
			return c.AccountGroupMembersUpdate(ctx, in.ID, map[string]any{"userIds": in.UserIDs})
		})

	addTool(r, "accounts", "higo.accounts.grants.create",
		"Grant a user or group access to a space.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in AccountsGrantCreateInput) (json.RawMessage, error) {
			return c.AccountGrantCreate(ctx, in)
		})

	addTool(r, "accounts", "higo.accounts.grants.delete",
		"Revoke a space access grant.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AccountsGrantDeleteInput) (json.RawMessage, error) {
			return c.AccountGrantDelete(ctx, in.ID)
		})
}
