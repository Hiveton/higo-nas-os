package vm

import (
	"context"
	"fmt"
	"path/filepath"
	"runtime"
	"sync"
	"time"

	"higoos/server-go/internal/state"
)

// Service exposes virtual-machine management. The adapter is the real libvirt
// driver on Linux and a devstub elsewhere — chosen at construction so router
// wiring stays platform-agnostic.
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
		s.adapter = NewLibvirtAdapter()
	} else {
		s.adapter = NewDevAdapter()
	}
	return s
}

// NewServiceWithAdapter lets tests inject an adapter.
func NewServiceWithAdapter(adapter Adapter) *Service {
	return &Service{now: time.Now, adapter: adapter}
}

func NewServiceWithStateDir(stateDir string) (*Service, error) {
	s := NewService()
	if stateDir == "" {
		return s, nil
	}
	s.statePath = filepath.Join(stateDir, "vm.json")
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

func (s *Service) List(ctx context.Context) ([]VM, error) {
	return s.adapter.List(ctx)
}

func (s *Service) Get(ctx context.Context, name string) (VM, error) {
	return s.adapter.Get(ctx, name)
}

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

// Action applies a lifecycle/power action to a VM, then records an audit entry.
// Returns the VM's post-action state (best-effort).
func (s *Service) Action(ctx context.Context, name, action, actor string) (VM, error) {
	if !validAction(action) {
		return VM{}, fmt.Errorf("unsupported vm action: %s", action)
	}
	if err := s.adapter.Action(ctx, name, action); err != nil {
		s.appendAudit(AuditEntry{Event: fmt.Sprintf("虚拟机 %s 执行 %s 失败：%s", name, actionLabel(action), err.Error()), VM: name, Action: action, Actor: actor, Result: "blocked"})
		return VM{}, err
	}
	s.appendAudit(AuditEntry{Event: fmt.Sprintf("虚拟机 %s 已%s", name, actionLabel(action)), VM: name, Action: action, Actor: actor, Result: "ok"})
	if action == ActionDelete {
		return VM{Name: name, State: "deleted"}, nil
	}
	// Best-effort post-action state.
	if v, err := s.adapter.Get(ctx, name); err == nil {
		return v, nil
	}
	return VM{Name: name}, nil
}

func (s *Service) appendAudit(e AuditEntry) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	e.ID = fmt.Sprintf("vm-audit-%03d", s.seq)
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

func actionLabel(action string) string {
	switch action {
	case ActionStart:
		return "启动"
	case ActionShutdown:
		return "关机"
	case ActionReboot:
		return "重启"
	case ActionForceStop:
		return "强制停止"
	case ActionAutostartOn:
		return "开启自启动"
	case ActionAutostartOff:
		return "关闭自启动"
	case ActionDelete:
		return "删除"
	default:
		return action
	}
}
