package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/platform"
)

// appCenterActionBody is the request payload shared by preview and confirm. On
// preview, Config seeds the impact summary; on confirm, ConfirmationID gates the
// side effect and Config supplies env interpolation values.
type appCenterActionBody struct {
	ConfirmationID string            `json:"confirmationId"`
	Actor          string            `json:"actor"`
	Config         map[string]string `json:"config"`
}

type appCenterRegistryBody struct {
	Name string `json:"name"`
	URL  string `json:"url"`
}

// appCenterApps lists every known app (installed and installable).
func (a *API) appCenterApps(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	apps, err := a.appCenter.Apps(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "app_center_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, apps)
}

// appCenterAppByID handles the governed two-phase lifecycle:
//
//	POST /apps/{id}/{action}          -> preview (returns confirmationId + impact)
//	POST /apps/{id}/{action}/confirm  -> execute the previewed action
//
// where action is install|update|start|stop|uninstall.
func (a *API) appCenterAppByID(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path, "/api/v1/app-center/apps/")
	if len(parts) < 2 {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	id, action := parts[0], parts[1]
	confirm := len(parts) >= 3 && parts[2] == "confirm"

	var body appCenterActionBody
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "app_center_bad_request", err.Error())
			return
		}
	}

	if !confirm {
		preview, err := a.appCenter.PreviewAction(r.Context(), id, action, body.Config)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "app_center_item_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, preview)
		return
	}

	result, err := a.appCenter.ConfirmAction(r.Context(), id, action, body.ConfirmationID, body.Actor)
	if err != nil {
		platform.WriteError(w, r, http.StatusConflict, "app_center_confirm_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

// appCenterCatalog lists every installable app manifest across all sources.
func (a *API) appCenterCatalog(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.appCenter.CatalogSource().Entries())
}

// appCenterCatalogByID serves a single catalog entry, or refreshes remote
// registries on POST /catalog/refresh.
func (a *API) appCenterCatalogByID(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path, "/api/v1/app-center/catalog/")
	if len(parts) < 1 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
		return
	}
	if parts[0] == "refresh" {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		if err := a.appCenter.CatalogSource().RefreshRemote(r.Context()); err != nil {
			platform.WriteError(w, r, http.StatusBadGateway, "app_center_refresh_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, a.appCenter.CatalogSource().Entries())
		return
	}
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entry, ok := a.appCenter.CatalogSource().Lookup(parts[0])
	if !ok {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_item_not_found", "catalog item not found")
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entry)
}

// appCenterRegistries lists (GET) or adds (POST) remote registry sources.
func (a *API) appCenterRegistries(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, a.appCenter.CatalogSource().Registries())
	case http.MethodPost:
		var body appCenterRegistryBody
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "app_center_bad_request", err.Error())
			return
		}
		reg, err := a.appCenter.CatalogSource().AddRegistry(body.Name, body.URL)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "app_center_registry_failed", err.Error())
			return
		}
		_ = a.appCenter.CatalogSource().RefreshRemote(r.Context())
		platform.WriteJSON(w, r, http.StatusOK, reg)
	default:
		platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	}
}

// appCenterRegistryByName removes a remote registry source.
func (a *API) appCenterRegistryByName(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path, "/api/v1/app-center/registries/")
	if len(parts) < 1 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
		return
	}
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	if err := a.appCenter.CatalogSource().RemoveRegistry(parts[0]); err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "app_center_registry_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.appCenter.CatalogSource().Registries())
}

// appCenterAudit lists the app-center audit log.
func (a *API) appCenterAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	log, err := a.appCenter.AuditLog(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "app_center_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, log)
}

// appCenterAuditByID reverses a previously-confirmed action.
func (a *API) appCenterAuditByID(w http.ResponseWriter, r *http.Request) {
	parts := splitPath(r.URL.Path, "/api/v1/app-center/audit/")
	if len(parts) != 2 || parts[1] != "rollback" {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body appCenterActionBody
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "app_center_bad_request", err.Error())
			return
		}
	}
	record, err := a.appCenter.Rollback(r.Context(), parts[0], body.Actor)
	if err != nil {
		platform.WriteError(w, r, http.StatusConflict, "app_center_rollback_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, record)
}

func splitPath(path, prefix string) []string {
	return strings.Split(strings.Trim(strings.TrimPrefix(path, prefix), "/"), "/")
}
