package platform

import "context"

// Principal is the authenticated actor behind a request. The session guard
// resolves it from the higo_session cookie (or injects an implicit admin in
// dev) and stows it on the request context; handlers read it for authorization
// and audit attribution.
type Principal struct {
	UserID    string
	Username  string
	Role      string
	SessionID string
	DeviceID  string
	// Implicit is true when the principal was synthesized by the dev fallback
	// (no real session). Authorization treats it as admin; audit can flag it.
	Implicit bool
}

type principalContextKey struct{}

// WithPrincipal returns a copy of ctx carrying the principal.
func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

// PrincipalFromContext returns the request principal, if any.
func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}

// IsAdmin reports whether the principal holds the admin role.
func (p Principal) IsAdmin() bool {
	return p.Role == "admin"
}
