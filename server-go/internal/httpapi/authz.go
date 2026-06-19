package httpapi

import (
	"net"
	"net/http"
	"strings"

	"higoos/server-go/internal/files"
	"higoos/server-go/internal/platform"
)

// adminOnlyPrefixes are API path prefixes whose state-changing methods require
// the admin role. Read methods (GET) stay available to any authenticated user
// so non-admins can still see their own view. This is the coarse, route-level
// authorization gate; resource-level space ACLs are enforced separately in the
// files/storage handlers via authorizeSpace.
var adminOnlyPrefixes = []string{
	"/api/v1/accounts/",
	"/api/v1/security/",
	"/api/v1/settings",
	"/api/v1/remote/",
	"/api/v1/protocols",
	"/api/v1/network/",
	"/api/v1/storage/",
	"/api/v1/ai/providers",
	"/api/v1/system/updates",
	"/api/v1/app-center/registries",
}

// writeMethods are the HTTP methods that mutate state.
func isWriteMethod(method string) bool {
	switch method {
	case http.MethodPost, http.MethodPut, http.MethodPatch, http.MethodDelete:
		return true
	default:
		return false
	}
}

func requiresAdmin(path, method string) bool {
	if !isWriteMethod(method) {
		return false
	}
	for _, prefix := range adminOnlyPrefixes {
		if strings.HasPrefix(path, prefix) {
			return true
		}
	}
	return false
}

// roleGate enforces the admin requirement for sensitive write routes. It runs
// after sessionGuard, so a principal is always present (the dev fallback injects
// an implicit admin). Denials are audited.
func roleGate(api *API) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !requiresAdmin(r.URL.Path, r.Method) {
				next.ServeHTTP(w, r)
				return
			}
			principal, ok := platform.PrincipalFromContext(r.Context())
			if !ok || !principal.IsAdmin() {
				api.auditAuthz(r, principal, "authz_denied", r.URL.Path)
				platform.WriteError(w, r, http.StatusForbidden, "forbidden", "admin role required")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// authorizeSpaceWrite reports whether the request's principal may write to the
// given space, consulting the accounts grant table. Admins and uncontrolled
// spaces always pass; this is the resource-level layer beneath the route-level
// admin gate. An empty space (whole tree) is allowed — those operations are
// already covered by the route gate where relevant.
func (a *API) authorizeSpaceWrite(r *http.Request, space string) bool {
	if strings.TrimSpace(space) == "" {
		return true
	}
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok {
		return false
	}
	if principal.IsAdmin() {
		return true
	}
	return a.accounts.CanWriteSpace(r.Context(), principal.UserID, space)
}

// viewerFor builds the file-visibility scope for the request's principal from
// the accounts entitlements. A missing principal or admin gets the full view.
func (a *API) viewerFor(r *http.Request) files.Viewer {
	principal, ok := platform.PrincipalFromContext(r.Context())
	if !ok || principal.IsAdmin() || a.accounts == nil {
		return files.Viewer{Admin: true}
	}
	ent, err := a.accounts.UserEntitlements(r.Context(), principal.UserID)
	if err != nil {
		// Fail closed: unknown entitlements → only the personal folder.
		return files.Viewer{Username: principal.Username}
	}
	return files.Viewer{
		Admin:         ent.Admin,
		Username:      ent.Username,
		GroupDirs:     ent.GroupDirs,
		GrantedSpaces: ent.GrantedSpaces,
	}
}

// authorizeSpaceRead reports whether the request's principal may read a file node.
func (a *API) authorizeSpaceRead(r *http.Request, node files.FileNode) bool {
	if a.files == nil {
		return true
	}
	return a.files.CanAccess(node, a.viewerFor(r))
}

// clientIP extracts the best-effort client IP, honoring X-Forwarded-For when
// the request came through the reverse proxy.
func clientIP(r *http.Request) string {
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		parts := strings.Split(xff, ",")
		return strings.TrimSpace(parts[0])
	}
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}

// derivePermissions maps a role to the coarse permission tokens the frontend
// uses to gate UI. Admin holds everything; others get a read-only baseline.
func derivePermissions(role string) []string {
	switch role {
	case "admin":
		return []string{"accounts:manage", "security:manage", "system:manage", "files:write", "files:read"}
	case "user":
		return []string{"files:write", "files:read"}
	default: // guest
		return []string{"files:read"}
	}
}
