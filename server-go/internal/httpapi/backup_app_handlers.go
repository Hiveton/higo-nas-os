package httpapi

import (
	"net/http"
	"strings"

	"higoos/server-go/internal/platform"
)

func (a *API) backupJobs(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	jobs, err := a.backups.Jobs(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "backup_jobs_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, jobs)
}

func (a *API) backupJobByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/backups/jobs/"), "/"), "/")
	if len(parts) < 2 {
		platform.WriteError(w, r, http.StatusNotFound, "backup_route_not_found", "backup route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	id, action := parts[0], parts[1]
	switch action {
	case "run":
		job, err := a.backups.Run(r.Context(), id)
		writeBackupJob(w, r, job, err)
	case "pause":
		job, err := a.backups.Pause(r.Context(), id)
		writeBackupJob(w, r, job, err)
	case "resume":
		job, err := a.backups.Resume(r.Context(), id)
		writeBackupJob(w, r, job, err)
	case "verify":
		job, err := a.backups.Verify(r.Context(), id)
		writeBackupJob(w, r, job, err)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "backup_route_not_found", "backup route not found")
	}
}

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

func (a *API) appCenterAppByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/app-center/apps/"), "/"), "/")
	if len(parts) < 2 {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
		return
	}
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	id, action := parts[0], parts[1]
	switch action {
	case "install":
		app, err := a.appCenter.Install(r.Context(), id)
		writeAppCenterApp(w, r, app, err)
	case "update":
		app, err := a.appCenter.Update(r.Context(), id)
		writeAppCenterApp(w, r, app, err)
	case "start":
		app, err := a.appCenter.Start(r.Context(), id)
		writeAppCenterApp(w, r, app, err)
	case "stop":
		app, err := a.appCenter.Stop(r.Context(), id)
		writeAppCenterApp(w, r, app, err)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "app_center_route_not_found", "app center route not found")
	}
}

func writeBackupJob(w http.ResponseWriter, r *http.Request, job any, err error) {
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "backup_job_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, job)
}

func writeAppCenterApp(w http.ResponseWriter, r *http.Request, app any, err error) {
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "app_center_item_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, app)
}
