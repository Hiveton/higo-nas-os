package vm

import (
	"context"
	"fmt"
	"os/exec"
)

// Adapter abstracts the hypervisor. LibvirtAdapter drives real virsh on Linux;
// DevAdapter is an in-memory devstub.
type Adapter interface {
	List(ctx context.Context) ([]VM, error)
	Get(ctx context.Context, name string) (VM, error)
	Capabilities(ctx context.Context) (HostCaps, error)
	// Action applies a power/lifecycle action: start, shutdown, reboot,
	// force-stop, autostart-on, autostart-off, delete.
	Action(ctx context.Context, name, action string) error
}

type commandRunner func(context.Context, string, ...string) ([]byte, error)

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	return exec.CommandContext(ctx, name, args...).Output()
}

// Valid lifecycle actions.
const (
	ActionStart        = "start"
	ActionShutdown     = "shutdown"
	ActionReboot       = "reboot"
	ActionForceStop    = "force-stop"
	ActionAutostartOn  = "autostart-on"
	ActionAutostartOff = "autostart-off"
	ActionDelete       = "delete"
)

func validAction(a string) bool {
	switch a {
	case ActionStart, ActionShutdown, ActionReboot, ActionForceStop, ActionAutostartOn, ActionAutostartOff, ActionDelete:
		return true
	}
	return false
}

// DevAdapter is the Mac/dev devstub: a couple of seed VMs whose state flips
// optimistically on action.
type DevAdapter struct {
	vms []VM
}

func NewDevAdapter() *DevAdapter {
	return &DevAdapter{vms: []VM{
		{Name: "home-assistant", UUID: "dev-uuid-ha", State: "running", VCPUs: 2, MemoryMB: 2048, Autostart: true, Persistent: true, Title: "智能家居"},
		{Name: "ubuntu-lab", UUID: "dev-uuid-lab", State: "shut off", VCPUs: 4, MemoryMB: 4096, Autostart: false, Persistent: true, Title: "实验环境"},
	}}
}

func (a *DevAdapter) List(ctx context.Context) ([]VM, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return cloneVMs(a.vms), nil
}

func (a *DevAdapter) Get(ctx context.Context, name string) (VM, error) {
	for _, v := range a.vms {
		if v.Name == name {
			return v, nil
		}
	}
	return VM{}, fmt.Errorf("vm not found: %s", name)
}

func (a *DevAdapter) Capabilities(ctx context.Context) (HostCaps, error) {
	return HostCaps{LibvirtAvailable: true, KVMAvailable: false, Hypervisor: "devstub", Version: "dev", Note: "开发主机模拟虚拟机（devstub）"}, nil
}

func (a *DevAdapter) Action(ctx context.Context, name, action string) error {
	for i := range a.vms {
		if a.vms[i].Name != name {
			continue
		}
		switch action {
		case ActionStart, ActionReboot:
			a.vms[i].State = "running"
		case ActionShutdown, ActionForceStop:
			a.vms[i].State = "shut off"
		case ActionAutostartOn:
			a.vms[i].Autostart = true
		case ActionAutostartOff:
			a.vms[i].Autostart = false
		case ActionDelete:
			a.vms = append(a.vms[:i], a.vms[i+1:]...)
		}
		return nil
	}
	return fmt.Errorf("vm not found: %s", name)
}
