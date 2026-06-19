package platform

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"net/http"
	"time"
)

const RequestIDHeader = "X-Request-ID"

type requestIDContextKey struct{}

// RequestIDMiddleware echoes an inbound X-Request-ID or mints a fresh one, then
// stashes it on the context so WriteJSON/WriteError can stamp every envelope.
func RequestIDMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get(RequestIDHeader)
		if requestID == "" {
			requestID = NewID("req")
		}
		w.Header().Set(RequestIDHeader, requestID)
		ctx := context.WithValue(r.Context(), requestIDContextKey{}, requestID)
		next.ServeHTTP(w, r.WithContext(ctx))
	})
}

func RequestIDFromContext(ctx context.Context) string {
	if value, ok := ctx.Value(requestIDContextKey{}).(string); ok {
		return value
	}
	return ""
}

// NewID returns a prefixed random identifier such as "cu-1a2b3c..." used for
// every cloud entity (users, devices, bindings, sessions).
func NewID(prefix string) string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err == nil {
		return prefix + "-" + hex.EncodeToString(b[:])
	}
	return prefix + "-" + time.Now().UTC().Format("20060102T150405.000000000")
}
