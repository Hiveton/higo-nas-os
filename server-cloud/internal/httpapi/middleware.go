package httpapi

import (
	"bufio"
	"log/slog"
	"net"
	"net/http"
	"strings"
	"time"

	"higoos/server-cloud/internal/platform"
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
				allow := origin
				if publicOrigin == "*" && origin == "" {
					allow = "*"
				}
				w.Header().Set("Access-Control-Allow-Origin", allow)
				w.Header().Set("Access-Control-Allow-Credentials", "true")
				w.Header().Set("Vary", "Origin")
			}
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Content-Type, X-Request-ID, Authorization, X-Device-Id, X-Device-Secret")
			if r.Method == http.MethodOptions {
				w.WriteHeader(http.StatusNoContent)
				return
			}
			next.ServeHTTP(w, r)
		})
	}
}

// cloudAuthGuard validates the cloud access token (Bearer JWT) on guarded routes
// and stows the principal. In dev, when no token is present and auth is not
// required, it injects an implicit principal so the API is curl-pokeable — the
// same affordance server-go's sessionGuard offers the desktop.
func cloudAuthGuard(api *API) middleware {
	authRequired := api.config.AuthRequired || !api.config.IsDev()
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodOptions || !requiresAuth(r.URL.Path) {
				next.ServeHTTP(w, r)
				return
			}
			if tok := bearerToken(r); tok != "" {
				if claims, err := api.tokens.VerifyAccess(tok); err == nil {
					r = r.WithContext(platform.WithPrincipal(r.Context(),
						platform.Principal{UserID: claims.Subject, Display: claims.Display}))
					next.ServeHTTP(w, r)
					return
				}
				platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "invalid or expired access token")
				return
			}
			if !authRequired {
				r = r.WithContext(platform.WithPrincipal(r.Context(),
					platform.Principal{UserID: "cu-dev", Display: "Dev User", Implicit: true}))
				next.ServeHTTP(w, r)
				return
			}
			platform.WriteError(w, r, http.StatusUnauthorized, "unauthorized", "authentication required")
		})
	}
}

// requiresAuth reports whether a path needs a cloud access token. Open paths:
// auth (login/refresh), device register, the agent WebSocket (device-authed) and
// health. The relay forward path (/d/...) and account/binding/push paths are
// guarded.
func requiresAuth(path string) bool {
	switch {
	case path == "/healthz" || path == "/v1/health":
		return false
	case strings.HasPrefix(path, "/v1/auth/"):
		return false
	case path == "/v1/devices/register":
		return false
	case path == "/v1/agent/connect":
		return false
	case path == "/v1/bindings/pairing/issue":
		return false // device-authenticated, not cloud-account-authenticated
	}
	return strings.HasPrefix(path, "/v1/") || strings.HasPrefix(path, "/d/")
}

func recoverPanic(logger *slog.Logger) middleware {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if rec := recover(); rec != nil {
					logger.Error("panic recovered",
						slog.Any("panic", rec),
						slog.String("path", r.URL.Path),
						slog.String("request_id", platform.RequestIDFromContext(r.Context())))
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
				slog.String("request_id", platform.RequestIDFromContext(r.Context())))
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

// Hijack is required so the relay agent WebSocket upgrade works through the
// middleware-wrapped ResponseWriter.
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
