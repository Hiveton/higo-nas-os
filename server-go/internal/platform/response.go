package platform

import (
	"encoding/json"
	"net/http"
)

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

func WriteJSON(w http.ResponseWriter, r *http.Request, status int, data any) {
	writeEnvelope(w, r, status, Envelope{OK: true, Data: data})
}

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
