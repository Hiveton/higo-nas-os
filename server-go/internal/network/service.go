package network

import (
	"context"
	"fmt"
	"net"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/state"
)

// Adapter abstracts the host-specific network operations. The real host adapter
// (Linux) drives `ip`/netplan; the devstub keeps state in memory for Mac dev.
type Adapter interface {
	Interfaces(ctx context.Context) ([]Interface, error)
	CurrentConfig(ctx context.Context) (NetworkConfig, error)
	Apply(ctx context.Context, cfg NetworkConfig) error
}

// Service owns the governed network configuration loop.
type Service struct {
	mu        sync.RWMutex
	now       func() time.Time
	adapter   Adapter
	seq       int
	pending   map[string]pendingChange
	audit     []AuditEntry
	statePath string
}

type snapshot struct {
	Seq     int                      `json:"seq"`
	Pending map[string]pendingChange `json:"pending"`
	Audit   []AuditEntry             `json:"audit"`
}

// NewService builds the network service. A nil adapter selects the real host
// adapter on Linux and the devstub elsewhere — the dev/NAS split decided at
// construction, like the storage and protocols domains.
func NewService(adapter Adapter) *Service {
	if adapter == nil {
		if runtime.GOOS == "linux" {
			adapter = NewHostAdapter()
		} else {
			adapter = NewDevAdapter()
		}
	}
	s := &Service{
		now:     time.Now,
		adapter: adapter,
		pending: make(map[string]pendingChange),
	}
	s.seed()
	return s
}

// NewServiceWithStateDir adds JSON persistence of the pending/audit state.
func NewServiceWithStateDir(adapter Adapter, stateDir string) (*Service, error) {
	s := NewService(adapter)
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "network.json")
	var persisted snapshot
	if err := state.LoadJSON(s.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Audit) > 0 || len(persisted.Pending) > 0 {
		s.seq = persisted.Seq
		if persisted.Pending != nil {
			s.pending = persisted.Pending
		}
		s.audit = persisted.Audit
		if s.seq < len(s.audit) {
			s.seq = len(s.audit)
		}
	}
	return s, nil
}

// Interfaces returns the live interface inventory. Read-only.
func (s *Service) Interfaces(ctx context.Context) ([]Interface, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return s.adapter.Interfaces(ctx)
}

// Config returns the live effective configuration. Read-only.
func (s *Service) Config(ctx context.Context) (NetworkConfig, error) {
	if err := ctx.Err(); err != nil {
		return NetworkConfig{}, err
	}
	return s.adapter.CurrentConfig(ctx)
}

// Audit returns the append-only governance log (newest first). Read-only.
func (s *Service) Audit(ctx context.Context) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AuditEntry(nil), s.audit...), nil
}

// PreviewConfig validates a desired configuration, computes its impact + risk
// and registers a pending change keyed by a confirmation id.
func (s *Service) PreviewConfig(ctx context.Context, req ConfigRequest) (ConfigPreview, error) {
	if err := ctx.Err(); err != nil {
		return ConfigPreview{}, err
	}
	target := req.NetworkConfig
	if err := validateConfig(target); err != nil {
		return ConfigPreview{}, err
	}
	current, err := s.adapter.CurrentConfig(ctx)
	if err != nil {
		return ConfigPreview{}, err
	}
	if target.Interface == "" {
		target.Interface = current.Interface
	}
	if target.Hostname == "" {
		target.Hostname = current.Hostname
	}

	risk, impact := assessChange(current, target)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	change := pendingChange{
		ConfirmationID: fmt.Sprintf("network-confirm-%03d", s.seq),
		RollbackID:     fmt.Sprintf("network-rollback-%03d", s.seq),
		Target:         target,
		Previous:       current,
		Risk:           risk,
		Impact:         impact,
		Actor:          actorOr(req.Actor, ""),
	}
	s.pending[change.ConfirmationID] = change
	if err := s.saveLocked(); err != nil {
		return ConfigPreview{}, err
	}
	return ConfigPreview{
		Impact:               impact,
		Risk:                 risk,
		RiskLabel:            riskLabel(risk),
		RequiresConfirmation: requiresConfirmation(risk),
		ConfirmationID:       change.ConfirmationID,
		RollbackID:           change.RollbackID,
		Target:               target,
	}, nil
}

// Confirm applies a previously previewed configuration by confirmation id. The
// host side effect runs first; audit only commits on success.
func (s *Service) Confirm(ctx context.Context, req ConfirmRequest) (AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return AuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	change, ok := s.pending[req.ConfirmationID]
	if !ok {
		return AuditEntry{}, fmt.Errorf("no pending change for confirmation: %s", req.ConfirmationID)
	}
	if err := s.adapter.Apply(ctx, change.Target); err != nil {
		return AuditEntry{}, err
	}
	delete(s.pending, req.ConfirmationID)
	prev := change.Previous
	entry := s.appendAuditLocked(AuditEntry{
		Event:          describeConfig("应用网络设置", change.Target),
		Actor:          actorOr(req.Actor, change.Actor),
		Risk:           change.Risk,
		Result:         audit.ResultConfirmed,
		Config:         &prev,
		ConfirmationID: change.ConfirmationID,
		RollbackID:     change.RollbackID,
		Rollback:       "恢复先前的网络设置",
	})
	return entry, s.saveLocked()
}

// Rollback reverses a confirmed change by its audit id, re-applying the previous
// configuration.
func (s *Service) Rollback(ctx context.Context, auditID string, req RollbackRequest) (AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return AuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.audit {
		if s.audit[i].ID == auditID {
			idx = i
			break
		}
	}
	if idx < 0 {
		return AuditEntry{}, fmt.Errorf("audit entry not found: %s", auditID)
	}
	entry := &s.audit[idx]
	if entry.Reverted {
		return AuditEntry{}, fmt.Errorf("audit entry already rolled back: %s", auditID)
	}
	if entry.Config == nil {
		return AuditEntry{}, fmt.Errorf("audit entry has no reversible action: %s", auditID)
	}
	if err := s.adapter.Apply(ctx, *entry.Config); err != nil {
		return AuditEntry{}, err
	}
	entry.Reverted = true
	reverted := s.appendAuditLocked(AuditEntry{
		Event:    describeConfig("回滚网络设置", *entry.Config),
		Actor:    actorOr(req.Actor, ""),
		Risk:     entry.Risk,
		Result:   audit.ResultRolledBack,
		Rollback: req.Reason,
	})
	return reverted, s.saveLocked()
}

func (s *Service) appendAuditLocked(entry AuditEntry) AuditEntry {
	s.seq++
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("network-audit-%03d", s.seq)
	}
	if entry.Time.IsZero() {
		entry.Time = s.now().UTC()
	}
	if entry.RiskLabel == "" {
		entry.RiskLabel = riskLabel(entry.Risk)
	}
	s.audit = append([]AuditEntry{entry}, s.audit...)
	return entry
}

func (s *Service) saveLocked() error {
	if s.statePath == "" {
		return nil
	}
	return state.SaveJSON(s.statePath, snapshot{Seq: s.seq, Pending: s.pending, Audit: s.audit})
}

func (s *Service) seed() {
	s.audit = []AuditEntry{{
		ID:        "network-audit-001",
		Event:     "初始化网络配置",
		Actor:     "系统",
		Risk:      audit.RiskLow,
		RiskLabel: riskLabel(audit.RiskLow),
		Result:    audit.ResultAllowed,
		Time:      time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC),
	}}
	s.seq = len(s.audit)
}

// --- validation & impact ----------------------------------------------------

func validateConfig(cfg NetworkConfig) error {
	switch cfg.Mode {
	case ModeDHCP:
		// Address fields are ignored under DHCP.
	case ModeStatic:
		if net.ParseIP(cfg.Address) == nil || net.ParseIP(cfg.Address).To4() == nil {
			return fmt.Errorf("静态地址必须是合法的 IPv4 地址")
		}
		if cfg.Prefix < 1 || cfg.Prefix > 32 {
			return fmt.Errorf("前缀长度必须在 1 到 32 之间")
		}
		if cfg.Gateway != "" && net.ParseIP(cfg.Gateway) == nil {
			return fmt.Errorf("网关必须是合法的 IP 地址")
		}
		for _, dns := range cfg.DNS {
			if strings.TrimSpace(dns) != "" && net.ParseIP(strings.TrimSpace(dns)) == nil {
				return fmt.Errorf("DNS 服务器 %q 不是合法的 IP 地址", dns)
			}
		}
	default:
		return fmt.Errorf("地址分配方式必须是 dhcp 或 static")
	}
	return nil
}

// assessChange returns the risk level and a human impact summary for moving from
// current to target.
func assessChange(current, target NetworkConfig) (audit.RiskLevel, string) {
	addressChanged := target.Mode != current.Mode ||
		(target.Mode == ModeStatic && target.cidr() != current.cidr())
	if addressChanged {
		dest := "自动获取 (DHCP)"
		if target.Mode == ModeStatic {
			dest = target.cidr()
		}
		return audit.RiskHigh, fmt.Sprintf(
			"将把网络地址改为「%s」。修改后设备 IP 可能变化，当前连接会中断，需要重新搜索设备。", dest)
	}
	if target.Hostname != "" && target.Hostname != current.Hostname {
		return audit.RiskMedium, fmt.Sprintf("将把主机名从「%s」改为「%s」。", current.Hostname, target.Hostname)
	}
	return audit.RiskLow, "未检测到实质性变更。"
}

func describeConfig(prefix string, cfg NetworkConfig) string {
	if cfg.Mode == ModeStatic {
		return fmt.Sprintf("%s：静态 %s 网关 %s", prefix, cfg.cidr(), cfg.Gateway)
	}
	return fmt.Sprintf("%s：自动获取 (DHCP)", prefix)
}

func trim(s string) string { return strings.TrimSpace(s) }
