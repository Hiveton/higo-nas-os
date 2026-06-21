package iscsi

import (
	"context"
	"fmt"
	"os/exec"
)

// Adapter abstracts the iSCSI target stack. HostAdapter drives real LIO/targetcli
// on Linux; DevAdapter is an in-memory devstub.
type Adapter interface {
	List(ctx context.Context) ([]Target, error)
	Capabilities(ctx context.Context) (HostCaps, error)
	CreateTarget(ctx context.Context, iqn string) (string, error)
	DeleteTarget(ctx context.Context, iqn string) error
	AddLUN(ctx context.Context, iqn, name string, sizeMB int) error
	AddACL(ctx context.Context, iqn, initiator string) error
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).CombinedOutput()
}

// DevAdapter is the Mac/dev devstub: a seed target whose LUNs/ACLs mutate
// optimistically.
type DevAdapter struct {
	targets []Target
	seq     int
}

func NewDevAdapter() *DevAdapter {
	return &DevAdapter{targets: []Target{
		{IQN: "iqn.2026-06.os.higo:storage.lab", LUNs: 1, ACLs: []string{"iqn.1994-05.com.redhat:lab-client"}, Portals: []string{"0.0.0.0:3260"}},
	}, seq: 1}
}

func (a *DevAdapter) List(ctx context.Context) ([]Target, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return cloneTargets(a.targets), nil
}

func (a *DevAdapter) Capabilities(ctx context.Context) (HostCaps, error) {
	return HostCaps{Available: true, Backend: "devstub", Portal: "0.0.0.0:3260", Note: "开发主机模拟 iSCSI（devstub）"}, nil
}

func (a *DevAdapter) CreateTarget(ctx context.Context, iqn string) (string, error) {
	if iqn == "" {
		a.seq++
		iqn = fmt.Sprintf("iqn.2026-06.os.higo:storage.t%d", a.seq)
	}
	a.targets = append(a.targets, Target{IQN: iqn, Portals: []string{"0.0.0.0:3260"}})
	return iqn, nil
}

func (a *DevAdapter) DeleteTarget(ctx context.Context, iqn string) error {
	for i := range a.targets {
		if a.targets[i].IQN == iqn {
			a.targets = append(a.targets[:i], a.targets[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("target not found: %s", iqn)
}

func (a *DevAdapter) AddLUN(ctx context.Context, iqn, name string, sizeMB int) error {
	for i := range a.targets {
		if a.targets[i].IQN == iqn {
			a.targets[i].LUNs++
			return nil
		}
	}
	return fmt.Errorf("target not found: %s", iqn)
}

func (a *DevAdapter) AddACL(ctx context.Context, iqn, initiator string) error {
	for i := range a.targets {
		if a.targets[i].IQN == iqn {
			a.targets[i].ACLs = append(a.targets[i].ACLs, initiator)
			return nil
		}
	}
	return fmt.Errorf("target not found: %s", iqn)
}
