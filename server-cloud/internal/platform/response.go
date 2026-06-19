// Package platform holds the cross-cutting plumbing shared by every server-cloud
// domain: the JSON response envelope, request-id propagation, the authenticated
// principal, config loading and the logger. It deliberately mirrors
// server-go/internal/platform so the two control planes speak the same wire
// contract ({ok,data,error,requestId}) and a developer moving between them pays
// zero context switch.
package platform

import (
	"encoding/json"
	"net/http"
)

// Envelope is the single response shape every handler returns.
type Envelope struct {
	OK        bool      `json:"ok"`
	Data      any       `json:"data,omitempty"`
	Error     *APIError `json:"error,omitempty"`
	RequestID string    `json:"requestId,omitempty"`
}

type APIError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// WriteJSON writes a success envelope wrapping data.
func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeEnvelope(w, r, status, Envelope{OK: true, Data: data})
}

// WriteError writes a failure envelope with a stable machine code and a
// human-readable message.
func WriteError(w http.ResponseWriter, r *http.Request, status int, code, message string) {
	writeEnvelope(w, r, status, Envelope{
		OK:    false,
		Error: &APIError{Code: code, Message: message},
	})
}

func writeEnvelope(w http.ResponseWriter, r *http.Request, status int, envelope Envelope) {
	envelope.RequestID = RequestIDFromContext(r.Context())
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(envelope); err != nil {
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}
}

// DecodeJSON reads and unmarshals a JSON request body into dst. It returns false
// and writes a 400 envelope when the body is missing or malformed, so callers
// can `if !DecodeJSON(...) { return }`.
func DecodeJSON(w http.ResponseWriter, r *http.Request, dst any) bool {
	if r.Body == nil {
		WriteError(w, r, http.StatusBadRequest, "invalid_body", "request body is required")
		return false
	}
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(dst); err != nil {
		WriteError(w, r, http.StatusBadRequest, "invalid_body", "request body is not valid JSON: "+err.Error())
		return false
	}
	return true
}
