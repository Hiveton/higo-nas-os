package iscsi

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Service exposes iSCSI target management. The adapter is the real LIO driver on
// Linux and a devstub elsewhere, chosen at construction so router wiring stays
// platform-agnostic.
type Service struct {
	mu        sync.RWMutex
	adapter   Adapter
	audit     []AuditEntry
	seq       int
	now       func() time.Time
	statePath string
}

type snapshot struct {
	Seq   int          `json:"seq"`
	Audit []AuditEntry `json:"audit"`
}

func NewService() *Service {
	s := &Service{now: time.Now}
	if runtime.GOOS == "linux" {
		s.adapter = NewHostAdapter()
	} else {
		s.adapter = NewDevAdapter()
	}
	return s
}

func NewServiceWithAdapter(adapter Adapter) *Service {
	return &Service{now: time.Now, adapter: adapter}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	s := NewService()
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "iscsi.json")
	var persisted snapshot
	if err := state.LoadJSON(s.statePath, &persisted); err != nil {
		return nil, err
	}
	if len(persisted.Audit) > 0 {
		s.seq = persisted.Seq
		s.audit = append([]AuditEntry(nil), persisted.Audit...)
	}
	return s, nil
}

func (s *Service) Targets(ctx context.Context) ([]Target, error) { return s.adapter.List(ctx) }

func (s *Service) Capabilities(ctx context.Context) (HostCaps, error) {
	return s.adapter.Capabilities(ctx)
}

func (s *Service) Audit(ctx context.Context) ([]AuditEntry, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.RLock()
	defer s.mu.RUnlock()
	return append([]AuditEntry(nil), s.audit...), nil
}

func (s *Service) CreateTarget(ctx context.Context, req CreateTargetRequest) (Target, error) {
	iqn, err := s.adapter.CreateTarget(ctx, strings.TrimSpace(req.IQN))
	if err != nil {
		s.appendAudit(AuditEntry{Event: "创建 iSCSI 目标失败：" + err.Error(), Action: "create-target", Actor: req.Actor, Result: "blocked"})
		return Target{}, err
	}
	s.appendAudit(AuditEntry{Event: "创建 iSCSI 目标：" + iqn, Target: iqn, Action: "create-target", Actor: req.Actor, Result: "ok"})
	return Target{IQN: iqn, Portals: []string{"0.0.0.0:3260"}}, nil
}

func (s *Service) DeleteTarget(ctx context.Context, iqn, actor string) error {
	if err := s.adapter.DeleteTarget(ctx, iqn); err != nil {
		s.appendAudit(AuditEntry{Event: "删除 iSCSI 目标失败：" + err.Error(), Target: iqn, Action: "delete-target", Actor: actor, Result: "blocked"})
		return err
	}
	s.appendAudit(AuditEntry{Event: "删除 iSCSI 目标：" + iqn, Target: iqn, Action: "delete-target", Actor: actor, Result: "ok"})
	return nil
}

func (s *Service) AddLUN(ctx context.Context, iqn string, req AddLUNRequest) error {
	name := strings.TrimSpace(req.Name)
	if name == "" {
		return fmt.Errorf("lun name is required")
	}
	if err := s.adapter.AddLUN(ctx, iqn, name, req.SizeMB); err != nil {
		s.appendAudit(AuditEntry{Event: fmt.Sprintf("为 %s 添加 LUN 失败：%s", iqn, err.Error()), Target: iqn, Action: "add-lun", Actor: req.Actor, Result: "blocked"})
		return err
	}
	s.appendAudit(AuditEntry{Event: fmt.Sprintf("为 %s 添加 LUN「%s」（%dMB）", iqn, name, req.SizeMB), Target: iqn, Action: "add-lun", Actor: req.Actor, Result: "ok"})
	return nil
}

func (s *Service) AddACL(ctx context.Context, iqn string, req AddACLRequest) error {
	initiator := strings.TrimSpace(req.Initiator)
	if initiator == "" {
		return fmt.Errorf("initiator iqn is required")
	}
	if err := s.adapter.AddACL(ctx, iqn, initiator); err != nil {
		s.appendAudit(AuditEntry{Event: fmt.Sprintf("为 %s 授权启动器失败：%s", iqn, err.Error()), Target: iqn, Action: "add-acl", Actor: req.Actor, Result: "blocked"})
		return err
	}
	s.appendAudit(AuditEntry{Event: fmt.Sprintf("为 %s 授权启动器 %s", iqn, initiator), Target: iqn, Action: "add-acl", Actor: req.Actor, Result: "ok"})
	return nil
}

func (s *Service) appendAudit(e AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	e.ID = fmt.Sprintf("iscsi-audit-%03d", s.seq)
	if e.Time.IsZero() {
		e.Time = s.now().UTC()
	}
	s.audit = append([]AuditEntry{e}, s.audit...)
	if len(s.audit) > 200 {
		s.audit = s.audit[:200]
	}
	if s.statePath != "" {
		_ = state.SaveJSON(s.statePath, snapshot{Seq: s.seq, Audit: s.audit})
	}
}
