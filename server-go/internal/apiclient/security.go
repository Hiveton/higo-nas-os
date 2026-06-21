package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
	"net/url"
)

// SecurityIdentities lists security identities and their permissions
// (GET /api/v1/security/identities).
func (c *Client) SecurityIdentities(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/identities", nil, nil)
}

// SecurityIdentityPermissionsUpdate updates an identity's permissions
// (PUT /api/v1/security/identities/{id}/permissions).
func (c *Client) SecurityIdentityPermissionsUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/security/identities/" + url.PathEscape(id) + "/permissions"
	return c.Do(ctx, http.MethodPut, path, nil, body)
}

// SecurityAIPolicies lists AI access policies per space
// (GET /api/v1/security/ai-policies).
func (c *Client) SecurityAIPolicies(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/ai-policies", nil, nil)
}

// SecurityAIPolicyUpdate updates an AI access policy
// (PUT /api/v1/security/ai-policies/{id}).
func (c *Client) SecurityAIPolicyUpdate(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/security/ai-policies/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodPut, path, nil, body)
}

// SecurityInspect runs an AI security sweep and returns risk actions
// (POST /api/v1/security/inspect).
func (c *Client) SecurityInspect(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/security/inspect", nil, nil)
}

// SecurityRiskActions lists pending and resolved high-risk actions
// (GET /api/v1/security/risk-actions).
func (c *Client) SecurityRiskActions(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/risk-actions", nil, nil)
}

// SecurityRiskActionConfirm confirms a pending high-risk action
// (POST /api/v1/security/risk-actions/{id}/confirm).
func (c *Client) SecurityRiskActionConfirm(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/security/risk-actions/" + url.PathEscape(id) + "/confirm"
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// SecurityRiskActionBlock blocks a pending high-risk action
// (POST /api/v1/security/risk-actions/{id}/block).
func (c *Client) SecurityRiskActionBlock(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/security/risk-actions/" + url.PathEscape(id) + "/block"
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// SecurityAudit lists the security audit log (GET /api/v1/security/audit).
func (c *Client) SecurityAudit(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/audit", nil, nil)
}

// SecurityAuditRollback rolls back an audited security action
// (POST /api/v1/security/audit/{id}/rollback).
func (c *Client) SecurityAuditRollback(ctx context.Context, id string, body any) (json.RawMessage, error) {
	path := "/api/v1/security/audit/" + url.PathEscape(id) + "/rollback"
	return c.Do(ctx, http.MethodPost, path, nil, body)
}

// SecurityShares lists active share links and their risk (GET /api/v1/shares).
func (c *Client) SecurityShares(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/shares", nil, nil)
}

// SecurityShareRevoke revokes a share link (DELETE /api/v1/shares/{id}).
func (c *Client) SecurityShareRevoke(ctx context.Context, id string) (json.RawMessage, error) {
	path := "/api/v1/shares/" + url.PathEscape(id)
	return c.Do(ctx, http.MethodDelete, path, nil, nil)
}

// SecurityHostPorts lists listening ports (GET /api/v1/security/host/ports).
func (c *Client) SecurityHostPorts(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/host/ports", nil, nil)
}

// SecurityHostFirewall returns firewall state (GET /api/v1/security/host/firewall).
func (c *Client) SecurityHostFirewall(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/security/host/firewall", nil, nil)
}

// SecurityHostScan runs a full host security scan (POST /api/v1/security/host/scan).
func (c *Client) SecurityHostScan(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodPost, "/api/v1/security/host/scan", nil, nil)
}
