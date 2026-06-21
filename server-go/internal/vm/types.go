// Package vm is the virtual-machine domain. On Linux it drives libvirt/KVM via
// `virsh`; on dev hosts it uses a deterministic devstub. Power actions are
// applied directly and audited (append-only); deleting a VM is destructive.
package vm

import "time"

// VM is one libvirt domain.
type VM struct {
	Name       string `json:"name"`
	UUID       string `json:"uuid,omitempty"`
	State      string `json:"state"` // running / shut off / paused / ...
	VCPUs      int    `json:"vcpus"`
	MemoryMB   int    `json:"memoryMB"`
	Autostart  bool   `json:"autostart"`
	Persistent bool   `json:"persistent"`
	Title      string `json:"title,omitempty"`
}

// HostCaps reports the hypervisor capability of the host.
type HostCaps struct {
	LibvirtAvailable bool   `json:"libvirtAvailable"`
	KVMAvailable     bool   `json:"kvmAvailable"`
	Version          string `json:"version,omitempty"`
	Hypervisor       string `json:"hypervisor"`
	Note             string `json:"note,omitempty"`
}

// AuditEntry is the append-only action log (newest first).
type AuditEntry struct {
	ID     string    `json:"id"`
	Event  string    `json:"event"`
	Actor  string    `json:"actor,omitempty"`
	VM     string    `json:"vm,omitempty"`
	Action string    `json:"action"`
	Result string    `json:"result"` // ok / blocked
	Time   time.Time `json:"time"`
}

// ActionRequest carries the audit actor for a power/delete action.
type ActionRequest struct {
	Actor string `json:"actor,omitempty"`
}

func cloneVMs(in []VM) []VM {
	return append([]VM(nil), in...)
}
