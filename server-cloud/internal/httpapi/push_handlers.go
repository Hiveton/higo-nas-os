package httpapi

import (
	"net/http"

	"higoos/server-cloud/internal/platform"
)

// POST /v1/push/register { token, platform } — register an APNs token for the
// current cloud account.
func (a *API) pushRegister(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	var body struct {
		Token    string `json:"token"`
		Platform string `json:"platform"`
	}
	if !platform.DecodeJSON(w, r, &body) {
		return
	}
	if body.Token == "" {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_input", "token is required")
		return
	}
	if err := a.push.Register(r.Context(), userID, body.Token, body.Platform); err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "internal_error", "could not register token")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"ok": true})
}
