package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/platform"
)

func (a *API) vmMachines(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	vms, err := a.vm.List(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "vm_list_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, vms)
}

func (a *API) vmCapabilities(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	caps, err := a.vm.Capabilities(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "vm_caps_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, caps)
}

func (a *API) vmAudit(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	entries, err := a.vm.Audit(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "vm_audit_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, entries)
}

// vmMachineByID handles GET /api/v1/vm/machines/{name} and
// POST /api/v1/vm/machines/{name}/{action}.
func (a *API) vmMachineByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/vm/machines/"), "/")
	parts := strings.Split(rest, "/")
	name := parts[0]
	if name == "" {
		platform.WriteError(w, r, http.StatusNotFound, "vm_not_found", "vm name required")
		return
	}
	if len(parts) == 1 {
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		v, err := a.vm.Get(r.Context(), name)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "vm_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, v)
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	v, err := a.vm.Action(r.Context(), name, parts[1], actorFromRequest(r))
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "vm_action_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, v)
}
