package protocols

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/state"
)

type Service struct {
	mu        sync.RWMutex
	now       func() time.Time
	adapter   Adapter
	seq       int
	protocols []Protocol
	shares    []Share
	pending   map[string]pendingChange
	audit     []AuditEntry
	statePath string
}

type snapshot struct {
	Seq       int                      `json:"seq"`
	Protocols []Protocol               `json:"protocols"`
	Shares    []Share                  `json:"shares"`
	Pending   map[string]pendingChange `json:"pending"`
	Audit     []AuditEntry             `json:"audit"`
}

// NewService builds the protocols service. A nil adapter selects the real host
// adapter on Linux and the devstub elsewhere — the dev/NAS split decided at
// construction, like the storage domain.
func NewService(adapter Adapter) *Service {
	if adapter == nil {
		if runtime.GOOS == "linux" {
			adapter = NewHostAdapter()
		} else {
			adapter = NewDevAdapter()
		}
	}
	service := &Service{
		now:     time.Now,
		adapter: adapter,
		pending: make(map[string]pendingChange),
	}
	service.seed()
	return service
}

func NewServiceWithStateDir(adapter Adapter, stateDir string) (*Service, error) {
	service := NewService(adapter)
	if stateDir == "" {
		return service, nil
	}
	service.statePath = filepath.Join(stateDir, "protocols.json")
	var persisted snapshot
	if err := state.LoadJSON(service.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Protocols) > 0 || len(persisted.Shares) > 0 || len(persisted.Audit) > 0 {
		service.seq = persisted.Seq
		service.protocols = cloneProtocols(persisted.Protocols)
		service.shares = cloneShares(persisted.Shares)
		if persisted.Pending != nil {
			service.pending = clonePending(persisted.Pending)
		}
		service.audit = cloneAudit(persisted.Audit)
		if service.seq < len(service.audit) {
			service.seq = len(service.audit)
		}
	}
	return service, nil
}

// List returns every protocol with its live running/installed state filled in
// by the adapter. Read-only.
func (s *Service) List(ctx context.Context) ([]Protocol, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	desired := cloneProtocols(s.protocols)
	s.mu.RUnlock()
	return s.adapter.Status(ctx, desired)
}

// Get returns a single protocol (with live state) by key.
func (s *Service) Get(ctx context.Context, key ProtocolKey) (Protocol, error) {
	list, err := s.List(ctx)
	if err != nil {
		return Protocol{}, err
	}
	for _, p := range list {
		if p.Key == key {
			return p, nil
		}
	}
	return Protocol{}, fmt.Errorf("protocol not found: %s", key)
}

// Shares returns the configured shares, optionally filtered to one protocol
// (empty key = all). Read-only.
func (s *Service) Shares(ctx context.Context, key ProtocolKey) ([]Share, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Share, 0, len(s.shares))
	for _, sh := range s.shares {
		if key == "" || sh.Protocol == key {
			out = append(out, cloneShare(sh))
		}
	}
	return out, nil
}

// Audit returns the append-only governance log (newest first). Read-only.
func (s *Service) Audit(ctx context.Context) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return cloneAudit(s.audit), nil
}

// PreviewEnable computes the impact of enabling a protocol and registers a
// pending change keyed by its confirmation id.
func (s *Service) PreviewEnable(ctx context.Context, key ProtocolKey, actor string) (ProtocolPreview, error) {
	if err := ctx.Err(); err != nil {
		return ProtocolPreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	proto, ok := s.findProtocolLocked(key)
	if !ok {
		return ProtocolPreview{}, fmt.Errorf("protocol not found: %s", key)
	}
	if proto.Enabled {
		return ProtocolPreview{}, fmt.Errorf("protocol already enabled: %s", key)
	}
	impact := fmt.Sprintf("将启动 %s 服务并开放端口 %d，局域网内的设备即可发现并访问对应共享目录。", proto.DisplayName, proto.Port)
	return s.registerPreviewLocked(pendingChange{Kind: changeEnable, Protocol: key, Risk: audit.RiskMedium, Impact: impact, Actor: actor}), nil
}

// PreviewDisable computes the impact of disabling a protocol.
func (s *Service) PreviewDisable(ctx context.Context, key ProtocolKey, actor string) (ProtocolPreview, error) {
	if err := ctx.Err(); err != nil {
		return ProtocolPreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	proto, ok := s.findProtocolLocked(key)
	if !ok {
		return ProtocolPreview{}, fmt.Errorf("protocol not found: %s", key)
	}
	if !proto.Enabled {
		return ProtocolPreview{}, fmt.Errorf("protocol already disabled: %s", key)
	}
	impact := fmt.Sprintf("将停止 %s 服务，已连接的客户端会断开，该协议下的所有共享目录立即不可访问。", proto.DisplayName)
	return s.registerPreviewLocked(pendingChange{Kind: changeDisable, Protocol: key, Risk: audit.RiskMedium, Impact: impact, Actor: actor}), nil
}

// PreviewCreateShare validates a new share and previews its risk. The directory
// must resolve under HIGO_NAS_ROOT when that root is configured (real NAS).
func (s *Service) PreviewCreateShare(ctx context.Context, key ProtocolKey, req CreateShareRequest) (ProtocolPreview, error) {
	if err := ctx.Err(); err != nil {
		return ProtocolPreview{}, err
	}
	if strings.TrimSpace(req.Name) == "" {
		return ProtocolPreview{}, fmt.Errorf("share name is required")
	}
	if err := validateSharePath(req.Path); err != nil {
		return ProtocolPreview{}, err
	}
	level := req.AccessLevel
	if level == "" {
		level = AccessAccount
	}
	if !validAccessLevel(level) {
		return ProtocolPreview{}, fmt.Errorf("invalid access level: %s", level)
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	proto, ok := s.findProtocolLocked(key)
	if !ok {
		return ProtocolPreview{}, fmt.Errorf("protocol not found: %s", key)
	}
	s.seq++
	share := Share{
		ID:           fmt.Sprintf("share-%s-%03d", key, s.seq),
		Protocol:     key,
		Name:         strings.TrimSpace(req.Name),
		Path:         filepath.Clean(strings.TrimSpace(req.Path)),
		AccessLevel:  level,
		AllowedUsers: append([]string(nil), req.AllowedUsers...),
		Guest:        req.Guest || level == AccessPublic,
		Enabled:      true,
		CreatedAt:    s.now().UTC(),
		CreatedBy:    req.Actor,
	}
	risk := shareRisk(level)
	impact := fmt.Sprintf("将通过 %s 协议把目录 %s 以「%s」方式共享。", proto.DisplayName, share.Path, accessLabel(level))
	if risk == audit.RiskHigh {
		impact += "该方式允许较开放的访问，请确认目录内不含敏感数据。"
	}
	return s.registerPreviewLocked(pendingChange{Kind: changeShareCreate, Protocol: key, Share: share, Risk: risk, Impact: impact, Actor: req.Actor}), nil
}

// PreviewDeleteShare previews removing a share by id.
func (s *Service) PreviewDeleteShare(ctx context.Context, id string, actor string) (ProtocolPreview, error) {
	if err := ctx.Err(); err != nil {
		return ProtocolPreview{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findShareLocked(id)
	if !ok {
		return ProtocolPreview{}, fmt.Errorf("share not found: %s", id)
	}
	share := cloneShare(s.shares[idx])
	impact := fmt.Sprintf("将移除共享目录「%s」(%s)，对应客户端将无法再访问该目录。", share.Name, share.Path)
	return s.registerPreviewLocked(pendingChange{Kind: changeShareDelete, Protocol: share.Protocol, Share: share, Risk: audit.RiskMedium, Impact: impact, Actor: actor}), nil
}

// UpdateConfig applies new settings to one protocol directly — the human user
// in the UI is the actor, so no second confirmation is required (mirrors
// security.UpdateIdentityPermissions). The change is audited and rollbackable,
// and is pushed to the host immediately when the protocol is running.
func (s *Service) UpdateConfig(ctx context.Context, key ProtocolKey, req ConfigUpdateRequest) (Protocol, error) {
	if err := ctx.Err(); err != nil {
		return Protocol{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx := -1
	for i := range s.protocols {
		if s.protocols[i].Key == key {
			idx = i
			break
		}
	}
	if idx < 0 {
		return Protocol{}, fmt.Errorf("protocol not found: %s", key)
	}
	prev := s.protocols[idx].Config
	s.protocols[idx].Config = req.Config
	if err := s.applyLocked(ctx, key, s.sharesForLocked(key)); err != nil {
		s.protocols[idx].Config = prev // revert in-memory on host failure
		return Protocol{}, err
	}
	prevCopy := prev
	s.seq++
	rollbackID := fmt.Sprintf("protocols-rollback-cfg-%03d", s.seq)
	s.appendAuditLocked(AuditEntry{
		Event:      fmt.Sprintf("更新 %s 协议设置", s.protocols[idx].DisplayName),
		Actor:      actorOr(req.Actor, ""),
		Risk:       audit.RiskMedium,
		Result:     audit.ResultAllowed,
		Kind:       changeConfigUpdate,
		Protocol:   key,
		Config:     &prevCopy, // store the PREVIOUS config for rollback
		RollbackID: rollbackID,
		Rollback:   "恢复先前的协议设置",
	})
	updated := s.protocols[idx]
	return updated, s.saveLocked()
}

// registerPreviewLocked allocates ids, stores the pending change and returns the
// preview payload. Caller holds the lock.
func (s *Service) registerPreviewLocked(change pendingChange) ProtocolPreview {
	s.seq++
	change.ConfirmationID = fmt.Sprintf("protocols-confirm-%03d", s.seq)
	change.RollbackID = fmt.Sprintf("protocols-rollback-%03d", s.seq)
	s.pending[change.ConfirmationID] = change
	_ = s.saveLocked()
	return ProtocolPreview{
		Kind:                 change.Kind,
		Protocol:             change.Protocol,
		Impact:               change.Impact,
		Risk:                 change.Risk,
		RiskLabel:            riskLabel(change.Risk),
		RequiresConfirmation: requiresConfirmation(change.Risk),
		ConfirmationID:       change.ConfirmationID,
		RollbackID:           change.RollbackID,
	}
}

// Confirm applies a previously previewed change identified by its confirmation
// id. The adapter side effect runs first; in-memory state and audit only commit
// on success.
func (s *Service) Confirm(ctx context.Context, req ConfirmRequest) (ConfirmResult, error) {
	if err := ctx.Err(); err != nil {
		return ConfirmResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	change, ok := s.pending[req.ConfirmationID]
	if !ok {
		return ConfirmResult{}, fmt.Errorf("no pending change for confirmation: %s", req.ConfirmationID)
	}
	actor := actorOr(req.Actor, change.Actor)
	proto, _ := s.findProtocolLocked(change.Protocol)

	var result ConfirmResult
	var event string
	switch change.Kind {
	case changeEnable:
		if err := s.adapter.EnableProtocol(ctx, change.Protocol, s.baseConfig()); err != nil {
			return ConfirmResult{}, err
		}
		// Flip enabled before applying so applyLocked pushes settings+shares now.
		updated := s.setProtocolEnabledLocked(change.Protocol, true)
		if err := s.applyLocked(ctx, change.Protocol, s.sharesForLocked(change.Protocol)); err != nil {
			s.setProtocolEnabledLocked(change.Protocol, false)
			return ConfirmResult{}, err
		}
		result.Protocol = &updated
		event = fmt.Sprintf("启用 %s 协议", proto.DisplayName)
	case changeDisable:
		if err := s.adapter.DisableProtocol(ctx, change.Protocol); err != nil {
			return ConfirmResult{}, err
		}
		updated := s.setProtocolEnabledLocked(change.Protocol, false)
		result.Protocol = &updated
		event = fmt.Sprintf("停用 %s 协议", proto.DisplayName)
	case changeShareCreate:
		shares := append(s.sharesForLocked(change.Protocol), change.Share)
		if err := s.applyLocked(ctx, change.Protocol, shares); err != nil {
			return ConfirmResult{}, err
		}
		s.shares = append(s.shares, cloneShare(change.Share))
		created := cloneShare(change.Share)
		result.Share = &created
		event = fmt.Sprintf("新增 %s 共享：%s", proto.DisplayName, change.Share.Name)
	case changeShareDelete:
		remaining := s.sharesForExcludingLocked(change.Protocol, change.Share.ID)
		if err := s.applyLocked(ctx, change.Protocol, remaining); err != nil {
			return ConfirmResult{}, err
		}
		s.removeShareLocked(change.Share.ID)
		removed := cloneShare(change.Share)
		result.Share = &removed
		event = fmt.Sprintf("移除 %s 共享：%s", proto.DisplayName, change.Share.Name)
	default:
		return ConfirmResult{}, fmt.Errorf("unknown change kind: %s", change.Kind)
	}

	delete(s.pending, req.ConfirmationID)
	entry := s.appendAuditLocked(AuditEntry{
		Event:          event,
		Actor:          actor,
		Risk:           change.Risk,
		Result:         audit.ResultConfirmed,
		Kind:           change.Kind,
		Protocol:       change.Protocol,
		Share:          shareForAudit(change),
		ConfirmationID: change.ConfirmationID,
		RollbackID:     change.RollbackID,
		Rollback:       rollbackHint(change.Kind, proto.DisplayName),
	})
	result.Audit = entry
	return result, s.saveLocked()
}

// Rollback reverses a confirmed change by its audit id.
func (s *Service) Rollback(ctx context.Context, auditID string, req RollbackRequest) (AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return AuditEntry{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	idx, ok := s.findAuditLocked(auditID)
	if !ok {
		return AuditEntry{}, fmt.Errorf("audit entry not found: %s", auditID)
	}
	entry := &s.audit[idx]
	if entry.Reverted {
		return AuditEntry{}, fmt.Errorf("audit entry already rolled back: %s", auditID)
	}
	if entry.RollbackID == "" || entry.Kind == "" {
		return AuditEntry{}, fmt.Errorf("audit entry has no reversible action: %s", auditID)
	}
	proto, _ := s.findProtocolLocked(entry.Protocol)

	switch entry.Kind {
	case changeEnable: // reverse = disable
		if err := s.adapter.DisableProtocol(ctx, entry.Protocol); err != nil {
			return AuditEntry{}, err
		}
		s.setProtocolEnabledLocked(entry.Protocol, false)
	case changeDisable: // reverse = enable
		if err := s.adapter.EnableProtocol(ctx, entry.Protocol, s.baseConfig()); err != nil {
			return AuditEntry{}, err
		}
		s.setProtocolEnabledLocked(entry.Protocol, true)
		if err := s.applyLocked(ctx, entry.Protocol, s.sharesForLocked(entry.Protocol)); err != nil {
			s.setProtocolEnabledLocked(entry.Protocol, false)
			return AuditEntry{}, err
		}
	case changeConfigUpdate: // reverse = restore the previous config snapshot
		if entry.Config == nil {
			return AuditEntry{}, fmt.Errorf("audit entry missing config snapshot: %s", auditID)
		}
		s.setProtocolConfigLocked(entry.Protocol, *entry.Config)
		if err := s.applyLocked(ctx, entry.Protocol, s.sharesForLocked(entry.Protocol)); err != nil {
			return AuditEntry{}, err
		}
	case changeShareCreate: // reverse = remove the created share
		if entry.Share == nil {
			return AuditEntry{}, fmt.Errorf("audit entry missing share snapshot: %s", auditID)
		}
		remaining := s.sharesForExcludingLocked(entry.Protocol, entry.Share.ID)
		if err := s.applyLocked(ctx, entry.Protocol, remaining); err != nil {
			return AuditEntry{}, err
		}
		s.removeShareLocked(entry.Share.ID)
	case changeShareDelete: // reverse = re-add the deleted share
		if entry.Share == nil {
			return AuditEntry{}, fmt.Errorf("audit entry missing share snapshot: %s", auditID)
		}
		shares := append(s.sharesForLocked(entry.Protocol), *entry.Share)
		if err := s.applyLocked(ctx, entry.Protocol, shares); err != nil {
			return AuditEntry{}, err
		}
		if _, exists := s.findShareLocked(entry.Share.ID); !exists {
			s.shares = append(s.shares, cloneShare(*entry.Share))
		}
	default:
		return AuditEntry{}, fmt.Errorf("audit entry has no reversible action: %s", auditID)
	}

	entry.Reverted = true
	entry.Result = audit.ResultRolledBack
	reverted := *entry
	s.appendAuditLocked(AuditEntry{
		Event:    fmt.Sprintf("回滚：%s", entry.Event),
		Actor:    actorOr(req.Actor, "协议中心"),
		Risk:     entry.Risk,
		Result:   audit.ResultRolledBack,
		Protocol: entry.Protocol,
		Rollback: fmt.Sprintf("已恢复 %s 协议的先前状态", proto.DisplayName),
	})
	return reverted, s.saveLocked()
}

// --- locked helpers ---------------------------------------------------------

func (s *Service) baseConfig() ProtocolBaseConfig {
	return ProtocolBaseConfig{
		NASRoot:    os.Getenv("HIGO_NAS_ROOT"),
		ServerName: "HiGoOS",
		Workgroup:  "WORKGROUP",
	}
}

// applyLocked regenerates the managed host config for one protocol from its
// current settings + the given share list — but only when the protocol is
// enabled. While a protocol is off, settings and shares are still persisted in
// our state and are applied to the host the moment it is enabled. This lets a
// user configure a protocol before turning it on, matching NAS conventions.
func (s *Service) applyLocked(ctx context.Context, key ProtocolKey, shares []Share) error {
	if p, ok := s.findProtocolLocked(key); !ok || !p.Enabled {
		return nil
	}
	return s.adapter.Apply(ctx, key, s.configForLocked(key), shares)
}

func (s *Service) configForLocked(key ProtocolKey) ProtocolConfig {
	if p, ok := s.findProtocolLocked(key); ok {
		return p.Config
	}
	return ProtocolConfig{}
}

func (s *Service) sharesForLocked(key ProtocolKey) []Share {
	out := make([]Share, 0, len(s.shares))
	for _, sh := range s.shares {
		if sh.Protocol == key {
			out = append(out, cloneShare(sh))
		}
	}
	return out
}

func (s *Service) sharesForExcludingLocked(key ProtocolKey, excludeID string) []Share {
	out := make([]Share, 0, len(s.shares))
	for _, sh := range s.shares {
		if sh.Protocol == key && sh.ID != excludeID {
			out = append(out, cloneShare(sh))
		}
	}
	return out
}

func (s *Service) setProtocolEnabledLocked(key ProtocolKey, enabled bool) Protocol {
	for i := range s.protocols {
		if s.protocols[i].Key == key {
			s.protocols[i].Enabled = enabled
			s.protocols[i].Running = enabled
			return s.protocols[i]
		}
	}
	return Protocol{}
}

func (s *Service) setProtocolConfigLocked(key ProtocolKey, config ProtocolConfig) {
	for i := range s.protocols {
		if s.protocols[i].Key == key {
			s.protocols[i].Config = config
			return
		}
	}
}

func (s *Service) removeShareLocked(id string) {
	for i := range s.shares {
		if s.shares[i].ID == id {
			s.shares = append(s.shares[:i], s.shares[i+1:]...)
			return
		}
	}
}

func (s *Service) findProtocolLocked(key ProtocolKey) (Protocol, bool) {
	for _, p := range s.protocols {
		if p.Key == key {
			return p, true
		}
	}
	return Protocol{}, false
}

func (s *Service) findShareLocked(id string) (int, bool) {
	for i := range s.shares {
		if s.shares[i].ID == id {
			return i, true
		}
	}
	return -1, false
}

func (s *Service) findAuditLocked(id string) (int, bool) {
	for i := range s.audit {
		if s.audit[i].ID == id {
			return i, true
		}
	}
	return -1, false
}

func (s *Service) appendAuditLocked(entry AuditEntry) AuditEntry {
	s.seq++
	if entry.ID == "" {
		entry.ID = fmt.Sprintf("protocols-audit-%03d", s.seq)
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
	return state.SaveJSON(s.statePath, snapshot{
		Seq:       s.seq,
		Protocols: cloneProtocols(s.protocols),
		Shares:    cloneShares(s.shares),
		Pending:   clonePending(s.pending),
		Audit:     cloneAudit(s.audit),
	})
}

func (s *Service) seed() {
	s.protocols = []Protocol{
		{Key: ProtocolSMB, DisplayName: "SMB / 文件共享", Enabled: false, MountHint: `\\HiGoOS\<共享名>`, Port: 445, Compatibility: "Windows 资源管理器、macOS 访达、Android 文件管理器", Config: ProtocolConfig{ServerName: "HiGoOS", Workgroup: "WORKGROUP", MinProtocol: "SMB2", GuestAccess: false}},
		{Key: ProtocolNFS, DisplayName: "NFS", Enabled: false, MountHint: "nfs://HiGoOS/<导出路径>", Port: 2049, Compatibility: "Linux、macOS、ESXi 等 *nix 客户端", Config: ProtocolConfig{Squash: "root_squash", AllowedNetwork: "*"}},
		{Key: ProtocolWebDAV, DisplayName: "WebDAV", Enabled: false, MountHint: "http://HiGoOS:8081/<共享名>", Port: 8081, Compatibility: "浏览器、RaiDrive、各类 WebDAV 客户端", Config: ProtocolConfig{HTTPSEnabled: false}},
		{Key: ProtocolDLNA, DisplayName: "DLNA", Enabled: false, MountHint: "DLNA：HiGoOS 媒体服务器", Port: 8200, Compatibility: "智能电视、投影仪、PS / Xbox 等 DLNA 设备", Config: ProtocolConfig{FriendlyName: "HiGoOS 媒体库"}},
	}
	seedTime := time.Date(2026, 5, 7, 9, 0, 0, 0, time.UTC)
	s.shares = []Share{
		{ID: "share-smb-001", Protocol: ProtocolSMB, Name: "家庭共享", Path: seedSharePath("家庭空间"), AccessLevel: AccessAccount, AllowedUsers: []string{"family"}, Enabled: true, CreatedAt: seedTime, CreatedBy: "系统预置"},
		{ID: "share-dlna-001", Protocol: ProtocolDLNA, Name: "影视媒体库", Path: seedSharePath("影视"), AccessLevel: AccessPublic, Guest: true, Enabled: true, CreatedAt: seedTime, CreatedBy: "系统预置"},
	}
	s.audit = []AuditEntry{
		{ID: "protocols-audit-001", Event: "初始化共享协议配置", Actor: "系统", Risk: audit.RiskLow, RiskLabel: riskLabel(audit.RiskLow), Result: audit.ResultAllowed, Time: seedTime},
	}
	s.seq = len(s.audit)
}

// --- free helpers -----------------------------------------------------------

func validAccessLevel(level AccessLevel) bool {
	switch level {
	case AccessPublic, AccessPassword, AccessAccount, AccessReadOnly:
		return true
	default:
		return false
	}
}

// validateSharePath requires the directory to resolve under HIGO_NAS_ROOT when
// that root is configured (real NAS). On dev hosts (root unset) any non-empty
// path is accepted — matching the repo's dev/prod split.
func validateSharePath(path string) error {
	p := strings.TrimSpace(path)
	if p == "" {
		return fmt.Errorf("share path is required")
	}
	root := strings.TrimSpace(os.Getenv("HIGO_NAS_ROOT"))
	if root == "" {
		return nil
	}
	cleanRoot := filepath.Clean(root)
	cleanPath := filepath.Clean(p)
	if !filepath.IsAbs(cleanPath) {
		cleanPath = filepath.Clean(filepath.Join(cleanRoot, cleanPath))
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("共享目录必须位于 %s 之内", cleanRoot)
	}
	return nil
}

func seedSharePath(name string) string {
	if root := strings.TrimSpace(os.Getenv("HIGO_NAS_ROOT")); root != "" {
		return filepath.Join(root, name)
	}
	return "/" + name
}

func shareForAudit(change pendingChange) *Share {
	if change.Kind == changeShareCreate || change.Kind == changeShareDelete {
		s := cloneShare(change.Share)
		return &s
	}
	return nil
}

func rollbackHint(kind changeKind, display string) string {
	switch kind {
	case changeEnable:
		return fmt.Sprintf("停用 %s 协议", display)
	case changeDisable:
		return fmt.Sprintf("重新启用 %s 协议", display)
	case changeShareCreate:
		return "移除新增的共享目录"
	case changeShareDelete:
		return "恢复被移除的共享目录"
	default:
		return ""
	}
}

func actorOr(actor, fallback string) string {
	if strings.TrimSpace(actor) != "" {
		return actor
	}
	if strings.TrimSpace(fallback) != "" {
		return fallback
	}
	return "协议中心"
}
