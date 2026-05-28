package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/platform"
)

func (a *API) accountsSummary(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	summary, err := a.accounts.Summary(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "accounts_summary_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, summary)
}

func (a *API) accountUsers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		summary, err := a.accounts.Summary(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "accounts_users_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, summary.Users)
	case http.MethodPost:
		var body accounts.CreateUserRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		user, err := a.accounts.CreateUser(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "account_user_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, user)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) accountUserByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/users/"), "/")
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "account_user_route_not_found", "account user id is required")
		return
	}
	switch r.Method {
	case http.MethodPut:
		var body accounts.UpdateUserRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		user, err := a.accounts.UpdateUser(r.Context(), id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "account_user_update_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, user)
	case http.MethodDelete:
		if err := a.accounts.DeleteUser(r.Context(), id); err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "account_user_delete_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(id, "deleted", "user deleted"))
	default:
		allowMethod(w, r, http.MethodPut, http.MethodDelete)
	}
}

func (a *API) accountGroups(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		summary, err := a.accounts.Summary(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "accounts_groups_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, summary.Groups)
	case http.MethodPost:
		var body accounts.CreateGroupRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		group, err := a.accounts.CreateGroup(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "account_group_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, group)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) accountGroupByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/groups/"), "/"), "/")
	if len(parts) != 2 || parts[1] != "members" {
		platform.WriteError(w, r, http.StatusNotFound, "account_group_route_not_found", "account group route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	var body accounts.UpdateGroupMembersRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	group, err := a.accounts.UpdateGroupMembers(r.Context(), parts[0], body)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "account_group_update_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, group)
}

func (a *API) accountGrants(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body accounts.SpaceGrantRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	grant, err := a.accounts.GrantSpace(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "account_grant_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, grant)
}

func (a *API) accountGrantByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/accounts/grants/"), "/")
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "account_grant_route_not_found", "account grant id is required")
		return
	}
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	if err := a.accounts.DeleteGrant(r.Context(), id); err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "account_grant_delete_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapTaskResponse(id, "deleted", "grant deleted"))
}
