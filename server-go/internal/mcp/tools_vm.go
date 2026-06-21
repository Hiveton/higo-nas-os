package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

// VMNameInput identifies a VM.
type VMNameInput struct {
	Name  string `json:"name" jsonschema:"Virtual machine name"`
	Actor string `json:"actor,omitempty" jsonschema:"Actor performing the operation for audit"`
}

// VMActionInput applies a lifecycle action to a VM.
type VMActionInput struct {
	Name   string `json:"name" jsonschema:"Virtual machine name"`
	Action string `json:"action" jsonschema:"start, shutdown, reboot, force-stop, autostart-on, autostart-off or delete"`
	Actor  string `json:"actor,omitempty"`
}

func registerVM(r *registry) {
	addTool(r, "vm", "higo.vm.list",
		"List virtual machines with their state, vCPUs and memory.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VMList(ctx)
		})

	addTool(r, "vm", "higo.vm.get",
		"Get a single virtual machine by name.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, in VMNameInput) (json.RawMessage, error) {
			return c.VMGet(ctx, in.Name)
		})

	addTool(r, "vm", "higo.vm.capabilities",
		"Report host hypervisor capabilities (libvirt/KVM availability).",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VMCapabilities(ctx)
		})

	addTool(r, "vm", "higo.vm.audit.list",
		"List the virtual-machine action audit log.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.VMAudit(ctx)
		})

	addTool(r, "vm", "higo.vm.action",
		"Apply a power/lifecycle action to a VM (start/shutdown/reboot/force-stop/autostart-on/autostart-off). 'delete' is destructive.",
		mutating(),
		func(ctx context.Context, c *apiclient.Client, in VMActionInput) (json.RawMessage, error) {
			return c.VMAction(ctx, in.Name, in.Action, map[string]any{"actor": in.Actor})
		})
}
