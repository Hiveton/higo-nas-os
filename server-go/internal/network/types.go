// Package network manages the NAS host network configuration (IPv4 address,
// DHCP/static mode, gateway, DNS, hostname). On Linux it reads live state via
// `ip` and writes a HiGoOS-owned netplan file; on dev hosts it uses a
// deterministic devstub. Changing the address is high risk (it can cut the
// caller's connection), so writes flow through a preview -> confirm -> rollback
// governance loop with an append-only audit trail, mirroring the protocols and
// security domains.
package network

import (
	"fmt"
	"time"

	"higoos/server-go/internal/audit"
)

// Mode is how an interface obtains its IPv4 address.
type Mode string

const (
	ModeDHCP   Mode = "dhcp"
	ModeStatic Mode = "static"
)

// Interface is the live state of one network interface.
type Interface struct {
	Name    string   `json:"name"`
	MAC     string   `json:"mac"`
	Up      bool     `json:"up"`
	Addrs   []string `json:"addrs"`            // CIDR form, e.g. 10.0.0.5/24
	Primary bool     `json:"primary"`          // carries the default route
	Mode    Mode     `json:"mode,omitempty"`   // best-effort current mode
}

// NetworkConfig is the desired (or current) IPv4 configuration for one interface
// plus the host name.
type NetworkConfig struct {
	Interface string   `json:"interface"`
	Mode      Mode     `json:"mode"`
	Address   string   `json:"address,omitempty"` // IPv4 without prefix, e.g. 10.0.0.5
	Prefix    int      `json:"prefix,omitempty"`  // 1..32
	Gateway   string   `json:"gateway,omitempty"`
	DNS       []string `json:"dns,omitempty"`
	Hostname  string   `json:"hostname,omitempty"`
}

// ConfigPreview is returned by PUT /network/config: it states impact + risk and
// the confirmation id required to actually apply the change.
type ConfigPreview struct {
	Impact               string          `json:"impactSummary"`
	Risk                 audit.RiskLevel `json:"risk"`
	RiskLabel            string          `json:"riskLabel"`
	RequiresConfirmation bool            `json:"requiresConfirmation"`
	ConfirmationID       string          `json:"confirmationId,omitempty"`
	RollbackID           string          `json:"rollbackId,omitempty"`
	Target               NetworkConfig   `json:"target"`
}

// pendingChange is a previewed-but-not-applied configuration keyed by its
// confirmation id.
type pendingChange struct {
	ConfirmationID string          `json:"confirmationId"`
	RollbackID     string          `json:"rollbackId"`
	Target         NetworkConfig   `json:"target"`
	Previous       NetworkConfig   `json:"previous"`
	Risk           audit.RiskLevel `json:"risk"`
	Impact         string          `json:"impact"`
	Actor          string          `json:"actor,omitempty"`
}

// AuditEntry is one append-only governance record carrying the previous config
// so the change can be reversed on rollback.
type AuditEntry struct {
	ID             string            `json:"id"`
	Event          string            `json:"event"`
	Actor          string            `json:"actor,omitempty"`
	Risk           audit.RiskLevel   `json:"risk"`
	RiskLabel      string            `json:"riskLabel"`
	Result         audit.AuditResult `json:"result"`
	Config         *NetworkConfig    `json:"config,omitempty"` // PREVIOUS config, for rollback
	ConfirmationID string            `json:"confirmationId,omitempty"`
	RollbackID     string            `json:"rollbackId,omitempty"`
	Reverted       bool              `json:"reverted"`
	Rollback       string            `json:"rollback,omitempty"`
	Time           time.Time         `json:"time"`
}

// --- request payloads -------------------------------------------------------

// ConfigRequest is the body of PUT /network/config (the preview step).
type ConfigRequest struct {
	NetworkConfig
	Actor string `json:"actor,omitempty"`
}

// ConfirmRequest applies a previewed change by confirmation id.
type ConfirmRequest struct {
	ConfirmationID string `json:"confirmationId"`
	Actor          string `json:"actor,omitempty"`
}

// RollbackRequest reverses a confirmed change.
type RollbackRequest struct {
	Actor  string `json:"actor,omitempty"`
	Reason string `json:"reason,omitempty"`
}

// --- helpers ----------------------------------------------------------------

func riskLabel(level audit.RiskLevel) string {
	switch level {
	case audit.RiskHigh:
		return "高风险"
	case audit.RiskMedium:
		return "中风险"
	default:
		return "低风险"
	}
}

func requiresConfirmation(level audit.RiskLevel) bool {
	return level == audit.RiskMedium || level == audit.RiskHigh
}

func actorOr(actor, fallback string) string {
	if trimmed := trim(actor); trimmed != "" {
		return trimmed
	}
	if trimmed := trim(fallback); trimmed != "" {
		return trimmed
	}
	return "网络中心"
}

func (c NetworkConfig) cidr() string {
	if c.Address == "" || c.Prefix == 0 {
		return c.Address
	}
	return fmt.Sprintf("%s/%d", c.Address, c.Prefix)
}
