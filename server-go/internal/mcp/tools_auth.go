package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type AuthSessionsListInput struct {
	UserID string `json:"userId,omitempty" jsonschema:"optional user id to list another user's sessions (admin only); empty lists the caller's own sessions"`
}

type AuthSessionRevokeInput struct {
	ID string `json:"id" jsonschema:"session id to revoke"`
}

func registerAuth(r *registry) {
	addTool(r, "auth", "higo.auth.me",
		"Get the currently authenticated user (id, role, quota, groups, permissions).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AuthMe(ctx)
		})

	addTool(r, "auth", "higo.auth.sessions.list",
		"List active sessions; admins may pass a userId to view another user's sessions.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in AuthSessionsListInput) (json.RawMessage, error) {
			return c.AuthSessions(ctx, in.UserID)
		})

	addTool(r, "auth", "higo.auth.sessions.revoke",
		"Revoke an active session.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in AuthSessionRevokeInput) (json.RawMessage, error) {
			return c.AuthSessionRevoke(ctx, in.ID)
		})

	addTool(r, "auth", "higo.auth.audit",
		"List the account/auth audit trail (logins, password changes, MFA, lockouts, authz denials). Admin only.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.AuthAudit(ctx)
		})
}
