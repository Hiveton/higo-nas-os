package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

type SecurityIdentityPermissionsUpdateInput struct {
	ID       string `json:"id" jsonschema:"identity id to update"`
	MFA      bool   `json:"mfa" jsonschema:"require multi-factor authentication"`
	FileACL  bool   `json:"fileAcl" jsonschema:"allow managing file access control lists"`
	AppAdmin bool   `json:"appAdmin" jsonschema:"grant app administration rights"`
	AITools  bool   `json:"aiTools" jsonschema:"allow use of AI tools"`
}

type SecurityAIPolicyUpdateInput struct {
	ID         string `json:"id" jsonschema:"AI policy id to update"`
	Indexed    *bool  `json:"indexed,omitempty" jsonschema:"whether the space is AI-indexed"`
	CloudModel *bool  `json:"cloudModel,omitempty" jsonschema:"whether cloud models may access the space"`
	Sensitive  string `json:"sensitive,omitempty" jsonschema:"sensitive data handling mode"`
}

type SecurityRiskActionConfirmInput struct {
	ID      string `json:"id" jsonschema:"risk action id to confirm"`
	ActorID string `json:"actorId,omitempty" jsonschema:"identity confirming the action"`
}

type SecurityRiskActionBlockInput struct {
	ID      string `json:"id" jsonschema:"risk action id to block"`
	ActorID string `json:"actorId,omitempty" jsonschema:"identity blocking the action"`
	Reason  string `json:"reason,omitempty" jsonschema:"reason for blocking the action"`
}

type SecurityAuditRollbackInput struct {
	ID      string `json:"id" jsonschema:"audit entry id to roll back"`
	ActorID string `json:"actorId,omitempty" jsonschema:"identity performing the rollback"`
}

type SecurityShareRevokeInput struct {
	ID string `json:"id" jsonschema:"share link id to revoke"`
}

func registerSecurity(r *registry) {
	addTool(r, "security", "higo.security.inspect",
		"Run an AI security sweep: scan public shares and identities for risks and queue findings as pending risk actions.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityInspect(ctx)
		})

	addTool(r, "security", "higo.security.identities.list",
		"List security identities and their effective permissions.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityIdentities(ctx)
		})

	addTool(r, "security", "higo.security.identities.permissions-update",
		"Update an identity's permissions (MFA, file ACL, app admin, AI tools).",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SecurityIdentityPermissionsUpdateInput) (json.RawMessage, error) {
			return c.SecurityIdentityPermissionsUpdate(ctx, in.ID, map[string]any{
				"mfa":      in.MFA,
				"fileAcl":  in.FileACL,
				"appAdmin": in.AppAdmin,
				"aiTools":  in.AITools,
			})
		})

	addTool(r, "security", "higo.security.ai-policies.list",
		"List AI access policies for each space.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityAIPolicies(ctx)
		})

	addTool(r, "security", "higo.security.ai-policies.update",
		"Update an AI access policy for a space.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SecurityAIPolicyUpdateInput) (json.RawMessage, error) {
			body := map[string]any{}
			if in.Indexed != nil {
				body["indexed"] = *in.Indexed
			}
			if in.CloudModel != nil {
				body["cloudModel"] = *in.CloudModel
			}
			if in.Sensitive != "" {
				body["sensitive"] = in.Sensitive
			}
			return c.SecurityAIPolicyUpdate(ctx, in.ID, body)
		})

	addTool(r, "security", "higo.security.risk-actions.list",
		"List pending and resolved high-risk security actions.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityRiskActions(ctx)
		})

	addTool(r, "security", "higo.security.risk-actions.confirm",
		"Confirm a pending high-risk security action by its id; the backend handles the confirmationId flow.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SecurityRiskActionConfirmInput) (json.RawMessage, error) {
			body := map[string]any{}
			if in.ActorID != "" {
				body["actorId"] = in.ActorID
			}
			return c.SecurityRiskActionConfirm(ctx, in.ID, body)
		})

	addTool(r, "security", "higo.security.risk-actions.block",
		"Block a pending high-risk security action by its id.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SecurityRiskActionBlockInput) (json.RawMessage, error) {
			body := map[string]any{}
			if in.ActorID != "" {
				body["actorId"] = in.ActorID
			}
			if in.Reason != "" {
				body["reason"] = in.Reason
			}
			return c.SecurityRiskActionBlock(ctx, in.ID, body)
		})

	addTool(r, "security", "higo.security.audit.list",
		"List the security audit log entries.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityAudit(ctx)
		})

	addTool(r, "security", "higo.security.audit.rollback",
		"Roll back an audited security action by its audit entry id.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in SecurityAuditRollbackInput) (json.RawMessage, error) {
			body := map[string]any{}
			if in.ActorID != "" {
				body["actorId"] = in.ActorID
			}
			return c.SecurityAuditRollback(ctx, in.ID, body)
		})

	addTool(r, "security", "higo.security.shares.list",
		"List active share links and their risk assessment.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.SecurityShares(ctx)
		})

	addTool(r, "security", "higo.security.shares.revoke",
		"Revoke an active share link by its id.",
		destructive(),
		func(ctx context.Context, c *apiclient.Client, in SecurityShareRevokeInput) (json.RawMessage, error) {
			return c.SecurityShareRevoke(ctx, in.ID)
		})
}
