package httpapi

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"higoos/server-go/internal/platform"
)

type middleware func(http.Handler) http.Handler

func chain(handler http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		handler = middlewares[i](handler)
	}
	return handler
}

func secureHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

func cors(publicOrigin string) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			origin := r.Header.Get("Origin")
			if publicOrigin == "*" || (origin != "" && origin == publicOrigin) {
				w.Header().Set("Access-Control-Allow-Origin", origin)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, X-CSRF-Token, Authorization")

			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// sessionGuard validates the higo_session cookie, resolves the authenticated
// principal and stows it on the request context. When auth is not required (the
// dev default) and no valid session exists, it injects an implicit admin so the
// desktop keeps working without logging in. The login route is always allowed
// through so unauthenticated callers can obtain a session.
func sessionGuard(api *API) middleware {
	cfg := api.config
	// Auth is required outside dev/test, or whenever explicitly switched on
	// (HIGO_AUTH_REQUIRED=true lets you exercise the login flow in dev).
	authRequired := cfg.AuthRequired || (cfg.Environment != "dev" && cfg.Environment != "test")
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || !requiresSession(r.URL.Path) || isAuthWhitelisted(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if principal, ok := api.resolvePrincipal(r); ok {
				r = r.WithContext(platform.WithPrincipal(r.Context(), principal))
				next.ServeHTTP(w, r)
				return
			}
			// A cloud-signed device access token (Bearer, presented by the mobile
			// App via the relay or LAN) is verified with the cloud public key
			// distributed at device registration. On success the request runs as
			// the mapped NAS-local user — not as a blanket admin.
			if principal, ok := api.resolveDeviceToken(r); ok {
				r = r.WithContext(platform.WithPrincipal(r.Context(), principal))
				next.ServeHTTP(w, r)
				return
			}
			// A Bearer token (external API / MCP client) is accepted as-is; token
			// validation is layered in P1. Treat it as an admin service principal.
			if r.Header.Get("Authorization") != "" {
				r = r.WithContext(platform.WithPrincipal(r.Context(), platform.Principal{UserID: "api-token", Username: "api-token", Role: "admin"}))
				next.ServeHTTP(w, r)
				return
			}
			if !authRequired {
				r = r.WithContext(platform.WithPrincipal(r.Context(), api.devAdminPrincipal(r)))
				next.ServeHTTP(w, r)
				return
			}
			platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "missing or invalid session")
		})
	}
}

// csrfGuard enforces double-submit CSRF on cookie-authenticated write requests.
// Bearer-token flows and the login route are exempt; dev disables it entirely.
func csrfGuard(api *API) middleware {
	cfg := api.config
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if cfg.CSRFDisabled || !isWriteMethod(r.Method) || !requiresSession(r.URL.Path) || isAuthWhitelisted(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if r.Header.Get("Authorization") != "" {
				next.ServeHTTP(w, r)
				return
			}
			cookie, err := r.Cookie(csrfCookieName)
			header := r.Header.Get("X-CSRF-Token")
			if err != nil || cookie.Value == "" || header == "" || header != cookie.Value {
				platform.WriteError(w, r, http.StatusForbidden, "csrf_failed", "missing or invalid CSRF token")
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// isAuthWhitelisted reports whether a guarded path is reachable without a
// session: the login endpoint, and the read-only device fingerprint the LAN
// discovery flow relies on (the HTTP twin of the UDP announce).
func isAuthWhitelisted(path string) bool {
	return path == "/api/v1/auth/login" || path == "/api/v1/system/identity"
}

func hasAPIPrefix(path string) bool {
	return len(path) >= len("/api/v1/") && path[:len("/api/v1/")] == "/api/v1/"
}

// requiresSession reports whether a path is guarded by the session check. The
// embedded MCP endpoint is guarded like the API surface so the same auth
// protects it outside dev/test.
func requiresSession(path string) bool {
	return hasAPIPrefix(path) || path == "/mcp" || strings.HasPrefix(path, "/mcp/")
}

func recoverPanic(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if recovered := recover(); recovered != nil {
					logger.Error("panic recovered",
						slog.Any("panic", recovered),
						slog.String("request_id", platform.RequestIDFromContext(r.Context())),
						slog.String("path", r.URL.Path),
					)
					platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "internal server error")
				}
			}()
			next.ServeHTTP(w, r)
		})
	}
}

func accessLog(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			started := time.Now()
			rec := &statusRecorder{ResponseWriter: w, status: http.StatusOK}
			next.ServeHTTP(rec, r)
			logger.Info("http request",
				slog.String("method", r.Method),
				slog.String("path", r.URL.Path),
				slog.Int("status", rec.status),
				slog.Duration("duration", time.Since(started)),
				slog.String("request_id", platform.RequestIDFromContext(r.Context())),
			)
		})
	}
}

type statusRecorder struct {
	http.ResponseWriter
	status int
}

func (r *statusRecorder) WriteHeader(status int) {
	r.status = status
	r.ResponseWriter.WriteHeader(status)
}

func (r *statusRecorder) Hijack() (net.Conn, *bufio.ReadWriter, error) {
	hijacker, ok := r.ResponseWriter.(http.Hijacker)
	if !ok {
		return nil, nil, http.ErrNotSupported
	}
	return hijacker.Hijack()
}

func (r *statusRecorder) Flush() {
	if flusher, ok := r.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (r *statusRecorder) Push(target string, opts *http.PushOptions) error {
	pusher, ok := r.ResponseWriter.(http.Pusher)
	if !ok {
		return http.ErrNotSupported
	}
	return pusher.Push(target, opts)
}
