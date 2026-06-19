package mcp

import (
	"context"
	"encoding/json"

	"higoos/server-go/internal/apiclient"
)

func registerHardware(r *registry) {
	addTool(r, "hardware", "higo.hardware.inventory",
		"Get the host hardware inventory: host, board, cpu, memory, network interfaces and sensors.",
		readOnly(),
		func(ctx context.Context, c *apiclient.Client, _ noInput) (json.RawMessage, error) {
			return c.HardwareInventory(ctx)
		})
}
