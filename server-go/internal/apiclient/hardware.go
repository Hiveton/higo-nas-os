package apiclient

import (
	"context"
	"encoding/json"
	"net/http"
)

// HardwareInventory returns the consolidated hardware inventory (host, board,
// cpu, memory, network, sensors) — GET /api/v1/hardware/inventory.
func (c *Client) HardwareInventory(ctx context.Context) (json.RawMessage, error) {
	return c.Do(ctx, http.MethodGet, "/api/v1/hardware/inventory", nil, nil)
}
