package vm

import (
	"context"
	"fmt"
	"os"
	"strconv"
	"strings"
)

// LibvirtAdapter drives real KVM/QEMU virtual machines through `virsh` on the
// system libvirt connection. Every command goes through the injectable runner
// so the parsing is unit-testable off-Linux.
type LibvirtAdapter struct {
	runner  commandRunner
	connURI string
}

func NewLibvirtAdapter() *LibvirtAdapter { return NewLibvirtAdapterWithRunner(runCommand) }

func NewLibvirtAdapterWithRunner(runner commandRunner) *LibvirtAdapter {
	if runner == nil {
		runner = runCommand
	}
	return &LibvirtAdapter{runner: runner, connURI: "qemu:///system"}
}

func (a *LibvirtAdapter) virsh(ctx context.Context, args ...string) ([]byte, error) {
	full := append([]string{"-c", a.connURI}, args...)
	return a.runner(ctx, "virsh", full...)
}

func (a *LibvirtAdapter) List(ctx context.Context) ([]VM, error) {
	out, err := a.virsh(ctx, "list", "--all", "--name")
	if err != nil {
		// libvirt not installed/running — surface an empty list, not a failure.
		return nil, nil
	}
	var vms []VM
	for _, name := range strings.Split(string(out), "\n") {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		v, err := a.Get(ctx, name)
		if err != nil {
			continue
		}
		vms = append(vms, v)
	}
	return vms, nil
}

func (a *LibvirtAdapter) Get(ctx context.Context, name string) (VM, error) {
	out, err := a.virsh(ctx, "dominfo", name)
	if err != nil {
		return VM{}, fmt.Errorf("vm not found: %s", name)
	}
	v := VM{Name: name}
	for _, line := range strings.Split(string(out), "\n") {
		key, val, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		val = strings.TrimSpace(val)
		switch key {
		case "Name":
			v.Name = val
		case "UUID":
			v.UUID = val
		case "State":
			v.State = val
		case "CPU(s)":
			v.VCPUs, _ = strconv.Atoi(val)
		case "Max memory":
			v.MemoryMB = parseKiBToMB(val)
		case "Persistent":
			v.Persistent = val == "yes"
		case "Autostart":
			v.Autostart = val == "enable"
		}
	}
	return v, nil
}

func (a *LibvirtAdapter) Capabilities(ctx context.Context) (HostCaps, error) {
	caps := HostCaps{Hypervisor: "qemu/kvm"}
	if _, err := os.Stat("/dev/kvm"); err == nil {
		caps.KVMAvailable = true
	}
	out, err := a.virsh(ctx, "version", "--daemon")
	if err != nil {
		caps.LibvirtAvailable = false
		caps.Note = "未检测到 libvirt（virsh 不可用）"
		return caps, nil
	}
	caps.LibvirtAvailable = true
	for _, line := range strings.Split(string(out), "\n") {
		if strings.Contains(line, "libvirt") && strings.Contains(line, "version") {
			caps.Version = strings.TrimSpace(line)
			break
		}
	}
	if !caps.KVMAvailable {
		caps.Note = "libvirt 可用，但本机无 /dev/kvm（无硬件虚拟化，性能受限）"
	}
	return caps, nil
}

func (a *LibvirtAdapter) Action(ctx context.Context, name, action string) error {
	var args []string
	switch action {
	case ActionStart:
		args = []string{"start", name}
	case ActionShutdown:
		args = []string{"shutdown", name}
	case ActionReboot:
		args = []string{"reboot", name}
	case ActionForceStop:
		args = []string{"destroy", name}
	case ActionAutostartOn:
		args = []string{"autostart", name}
	case ActionAutostartOff:
		args = []string{"autostart", "--disable", name}
	case ActionDelete:
		args = []string{"undefine", name, "--nvram"}
	default:
		return fmt.Errorf("unsupported vm action: %s", action)
	}
	if _, err := a.virsh(ctx, args...); err != nil {
		// `undefine --nvram` errors when there's no nvram; retry plainly.
		if action == ActionDelete {
			if _, e2 := a.virsh(ctx, "undefine", name); e2 == nil {
				return nil
			}
		}
		return fmt.Errorf("virsh %s: %w", action, err)
	}
	return nil
}

// parseKiBToMB turns "2097152 KiB" into 2048.
func parseKiBToMB(val string) int {
	fields := strings.Fields(val)
	if len(fields) == 0 {
		return 0
	}
	kib, err := strconv.Atoi(fields[0])
	if err != nil {
		return 0
	}
	return kib / 1024
}
