package relay

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Provisioner forwards account-provisioning to a NAS over the relay tunnel. It
// satisfies bindings.Provisioner without importing that package (Go is
// structurally typed at the interface boundary). Wire it in production:
//
//	bindings.NewService(store, devices, tokens, relay.NewProvisioner(hub), ttl)
//
// In dev the Hub is empty, so this returns ErrOffline and callers fall back to
// the stub — but production keeps the NAS as the source of truth for local users.
type Provisioner struct {
	hub *Hub
}

// NewProvisioner builds a relay-backed provisioner.
func NewProvisioner(hub *Hub) *Provisioner { return &Provisioner{hub: hub} }

type provisionRequest struct {
	CloudUserID string `json:"cloudUserId"`
	Role        string `json:"role"`
}

type provisionResponse struct {
	LocalUserID string `json:"localUserId"`
}

// LinkCloudAccount asks the NAS to create/link a local user for the cloud account.
func (p *Provisioner) LinkCloudAccount(ctx context.Context, deviceID, cloudUserID, role string) (string, error) {
	body, _ := json.Marshal(provisionRequest{CloudUserID: cloudUserID, Role: role})
	resp, err := p.hub.Call(ctx, deviceID, "POST", "/api/v1/cloud/provision", nil, body)
	if err != nil {
		return "", err
	}
	if resp.Status < 200 || resp.Status >= 300 {
		return "", fmt.Errorf("nas provision failed: status %d", resp.Status)
	}
	// Unwrap the server-go envelope {ok,data:{localUserId}}.
	var env struct {
		OK   bool              `json:"ok"`
		Data provisionResponse `json:"data"`
	}
	if err := json.Unmarshal(resp.Body, &env); err != nil {
		return "", err
	}
	if !env.OK || env.Data.LocalUserID == "" {
		return "", errors.New("nas provision returned no local user")
	}
	return env.Data.LocalUserID, nil
}

// Unlink asks the NAS to drop the local mapping for a cloud account.
func (p *Provisioner) Unlink(ctx context.Context, deviceID, cloudUserID string) error {
	body, _ := json.Marshal(provisionRequest{CloudUserID: cloudUserID})
	_, err := p.hub.Call(ctx, deviceID, "POST", "/api/v1/cloud/unprovision", nil, body)
	return err
}
