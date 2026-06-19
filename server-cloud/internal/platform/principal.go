package platform

import "context"

// Principal is the authenticated cloud actor behind a request. The cloudAuthGuard
// resolves it from the access JWT (or injects an implicit test principal in dev)
// and stows it on the request context; handlers read it for authorization and
// audit attribution.
//
// Unlike server-go's Principal (which models a NAS-local user), this models a
// HiGoOS *cloud account*: UserID is the cloud user id (cu-xxx).
type Principal struct {
	UserID   string
	Display  string
	// Implicit is true when synthesized by the dev fallback (no real token).
	Implicit bool
}

type principalContextKey struct{}

func WithPrincipal(ctx context.Context, principal Principal) context.Context {
	return context.WithValue(ctx, principalContextKey{}, principal)
}

func PrincipalFromContext(ctx context.Context) (Principal, bool) {
	principal, ok := ctx.Value(principalContextKey{}).(Principal)
	return principal, ok
}
