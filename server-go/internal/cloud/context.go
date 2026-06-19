package cloud

import (
	"context"

	"higoos/server-go/internal/state"
)

type relayOriginKey struct{}

// WithRelayOrigin marks a request context as having arrived over the cloud relay
// tunnel (i.e. the cloud — which holds this device's secret — dispatched it).
// Cloud-internal endpoints like /api/v1/cloud/provision trust only such requests,
// so a LAN attacker calling the path directly is rejected.
func WithRelayOrigin(ctx context.Context) context.Context {
	return context.WithValue(ctx, relayOriginKey{}, true)
}

// IsRelayOrigin reports whether the request came over the relay tunnel.
func IsRelayOrigin(ctx context.Context) bool {
	v, _ := ctx.Value(relayOriginKey{}).(bool)
	return v
}

// LoadPublicKey returns the cloud Ed25519 public key persisted at registration,
// or "" if the device has not registered. The session guard uses it to verify
// App-presented device access tokens.
func LoadPublicKey(stateDir string) string {
	if stateDir == "" {
		return ""
	}
	var p Persisted
	if err := state.LoadJSON(stateDir+"/cloud.json", &p); err != nil {
		return ""
	}
	return p.CloudPublicKey
}
