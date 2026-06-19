package httpapi

import (
	"net/http"

	"higoos/server-cloud/internal/platform"
)

// GET /v1/account/me — current cloud account profile + linked login methods.
func (a *API) accountMe(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	userID, ok := requireUser(w, r)
	if !ok {
		return
	}
	profile, err := a.account.Profile(r.Context(), userID)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "not_found", "account not found")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, profile)
}
