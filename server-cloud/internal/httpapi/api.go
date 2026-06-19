package httpapi

import (
	"log/slog"
	"net/http"
	"strings"

	"higoos/server-cloud/internal/account"
	"higoos/server-cloud/internal/auth"
	"higoos/server-cloud/internal/bindings"
	"higoos/server-cloud/internal/devices"
	"higoos/server-cloud/internal/platform"
	"higoos/server-cloud/internal/push"
	"higoos/server-cloud/internal/relay"
)

// API is the assembled cloud handler set: config, logger and the domain services
// the handlers call. It mirrors server-go's *API aggregate.
type API struct {
	config  platform.Config
	logger  *slog.Logger
	tokens  *auth.Manager
	account *account.Service
	devices *devices.Service
	binding *bindings.Service
	relay   *relay.Hub
	push    *push.Service
}

// allowMethod enforces the HTTP method(s) a handler accepts, writing a 405 with
// the standard envelope otherwise. Mirrors server-go's helper so handlers that
// branch on method (list vs create on one path) read identically.
func allowMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, m := range methods {
		if r.Method == m {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	return false
}

// principal returns the authenticated cloud principal resolved by cloudAuthGuard.
func principal(r *http.Request) (platform.Principal, bool) {
	return platform.PrincipalFromContext(r.Context())
}

// requireUser returns the cloud user id or writes 401 and returns false.
func requireUser(w http.ResponseWriter, r *http.Request) (string, bool) {
	p, ok := principal(r)
	if !ok || p.UserID == "" {
		platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		return "", false
	}
	return p.UserID, true
}

// bearerToken extracts a Bearer token from the Authorization header.
func bearerToken(r *http.Request) string {
	h := r.Header.Get("Authorization")
	if strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(h[len("Bearer "):])
	}
	return ""
}
