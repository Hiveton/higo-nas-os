package httpapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"time"

	hdocker "higoos/server-go/internal/docker"
	"higoos/server-go/internal/downloads"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/remote"
	"higoos/server-go/internal/storage"
)

func (a *API) storagePools(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	pools, err := a.storage.Pools(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "storage_pools_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapStoragePools(pools))
}

func (a *API) storageSpaces(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		spaces, err := a.storage.Spaces(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "storage_spaces_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, spaces)
	case http.MethodPost:
		var body storage.CreateSpaceRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		space, err := a.storage.CreateSpace(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "storage_space_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, space)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) storageSpaceByID(w http.ResponseWriter, r *http.Request) {
	id := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/storage/spaces/"), "/")
	if id == "" {
		platform.WriteError(w, r, http.StatusNotFound, "storage_space_route_not_found", "storage space id is required")
		return
	}
	if !allowMethod(w, r, http.MethodDelete) {
		return
	}
	var body storage.DeleteSpaceRequest
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	task, err := a.storage.DeleteSpace(r.Context(), id, body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "storage_space_delete_failed", err.Error())
		return
	}
	if _, err := a.accounts.DeleteSpaceGrants(r.Context(), id); err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "storage_space_grants_cleanup_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

func (a *API) storageDisks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		disks, err := a.storage.Disks(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "storage_disks_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapStorageDisks(disks))
	case http.MethodPost:
		var body storage.AddDiskRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		disk, err := a.storage.AddDisk(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "storage_disk_add_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapStorageDisk(disk))
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) storageDiskByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/storage/disks/"), "/"), "/")
	if len(parts) == 0 || parts[0] == "" {
		platform.WriteError(w, r, http.StatusNotFound, "storage_disk_route_not_found", "storage disk route not found")
		return
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		var body storage.RemoveDiskRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		task, err := a.storage.RemoveDisk(r.Context(), parts[0], body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "storage_disk_remove_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, task)
		return
	}
	if len(parts) == 2 && parts[1] == "settings" {
		if !allowMethod(w, r, http.MethodPut) {
			return
		}
		var body storage.DiskSettingsRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		disk, err := a.storage.UpdateDiskSettings(r.Context(), parts[0], body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "storage_disk_settings_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapStorageDisk(disk))
		return
	}
	platform.WriteError(w, r, http.StatusNotFound, "storage_disk_route_not_found", "storage disk route not found")
}

func (a *API) storageSmartReports(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	reports, err := a.storage.SmartReports(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "storage_smart_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, reports)
}

func (a *API) storageSmartScan(w http.ResponseWriter, r *http.Request) {
	a.storageTask(w, r, "smart")
}

func (a *API) storageRepair(w http.ResponseWriter, r *http.Request) {
	a.storageTask(w, r, "repair")
}

func (a *API) storageSnapshot(w http.ResponseWriter, r *http.Request) {
	a.storageTask(w, r, "snapshot")
}

func (a *API) storageTask(w http.ResponseWriter, r *http.Request, kind string) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var target storage.TaskTarget
	if r.ContentLength != 0 {
		if err := decodeJSON(r, &target); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
	}
	var (
		task storage.StorageTask
		err  error
	)
	switch kind {
	case "smart":
		task, err = a.storage.StartSMARTScan(r.Context(), target)
	case "repair":
		task, err = a.storage.StartRepair(r.Context(), target)
	case "snapshot":
		task, err = a.storage.CreateSnapshot(r.Context(), target)
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "storage_task_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

func (a *API) storageTaskByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	id := strings.TrimPrefix(r.URL.Path, "/api/v1/storage/tasks/")
	task, err := a.storage.GetTask(r.Context(), strings.Trim(id, "/"))
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "storage_task_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

// tasksList returns the background task ledger from the shared task runtime,
// optionally filtered by ?kind=.
func (a *API) tasksList(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	kind := strings.TrimSpace(r.URL.Query().Get("kind"))
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"tasks": a.tasks.List(kind)})
}

// taskByID returns a single task (GET /api/v1/tasks/{id}) or cancels a queued
// one (POST /api/v1/tasks/{id}/cancel).
func (a *API) taskByID(w http.ResponseWriter, r *http.Request) {
	rest := strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/tasks/"), "/")
	if rest == "" {
		platform.WriteError(w, r, http.StatusNotFound, "task_not_found", "task id is required")
		return
	}
	parts := strings.Split(rest, "/")
	id := parts[0]
	if len(parts) == 2 && parts[1] == "cancel" {
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		task, err := a.tasks.Cancel(id)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "task_cancel_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, task)
		return
	}
	if len(parts) != 1 {
		platform.WriteError(w, r, http.StatusNotFound, "task_not_found", "unknown task route")
		return
	}
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	task, err := a.tasks.Get(id)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "task_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

// tasksStream streams live task updates as SSE. It first replays the current
// ledger (one `data: {task}` frame each), then pushes a frame on every task
// state change until the client disconnects. A periodic comment keeps the
// connection alive through proxies.
func (a *API) tasksStream(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		platform.WriteError(w, r, http.StatusInternalServerError, "stream_unsupported", "streaming is not supported")
		return
	}
	w.Header().Set("Content-Type", "text/event-stream; charset=utf-8")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no")
	w.WriteHeader(http.StatusOK)

	writeTask := func(task any) {
		buf, err := json.Marshal(task)
		if err != nil {
			return
		}
		fmt.Fprintf(w, "data: %s\n\n", buf)
		flusher.Flush()
	}

	// Subscribe before replaying the snapshot so no update is missed in between.
	ch, unsubscribe := a.tasks.Subscribe()
	defer unsubscribe()
	for _, task := range a.tasks.List("") {
		writeTask(task)
	}

	ctx := r.Context()
	ticker := time.NewTicker(25 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case task, ok := <-ch:
			if !ok {
				return
			}
			writeTask(task)
		case <-ticker.C:
			fmt.Fprint(w, ": ping\n\n")
			flusher.Flush()
		}
	}
}

func (a *API) downloadTasks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		platform.WriteJSON(w, r, http.StatusOK, a.downloads.ListTasks(r.Context()))
	case http.MethodPost:
		var body downloads.CreateTaskRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		task, err := a.downloads.CreateTask(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "download_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, task)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) downloadTaskByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/downloads/tasks/"), "/"), "/")
	id, err := strconv.Atoi(parts[0])
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_download_id", err.Error())
		return
	}
	if len(parts) == 1 && r.Method == http.MethodDelete {
		result, err := a.downloads.DeleteTask(r.Context(), id, downloads.DeleteTaskOptions{
			DeleteFile: parseBoolQuery(r, "deleteFile"),
		})
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "download_task_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, result)
		return
	}
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "download_route_not_found", "download route not found")
		return
	}
	switch parts[1] {
	case "pause":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		task, err := a.downloads.PauseTask(r.Context(), id)
		writeDownloadTaskResult(w, r, task, err)
	case "resume":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		task, err := a.downloads.ResumeTask(r.Context(), id)
		writeDownloadTaskResult(w, r, task, err)
	case "archive":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		result, err := a.downloads.ArchiveTask(r.Context(), id)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "download_task_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, result)
	default:
		platform.WriteError(w, r, http.StatusNotFound, "download_route_not_found", "download route not found")
	}
}

func (a *API) downloadSpeedProfiles(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.downloads.SpeedProfiles(r.Context()))
}

func (a *API) downloadSpeedProfile(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	var body downloads.SpeedProfile
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	profile, err := a.downloads.UpdateActiveSpeedProfile(r.Context(), body.Name)
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "speed_profile_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, profile)
}

func (a *API) dockerStacks(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	stacks, err := a.docker.Stacks(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "docker_stacks_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, stacks)
}

func (a *API) dockerContainers(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		containers, err := a.docker.Containers(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "docker_containers_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapDockerContainers(containers))
	case http.MethodPost:
		var body hdocker.CreateContainerRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		container, err := a.docker.CreateContainer(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_container_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapDockerContainer(container))
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) dockerImages(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	images, err := a.docker.Images(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "docker_images_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, images)
}

func (a *API) dockerImageSearch(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	results, err := a.docker.SearchImages(r.Context(), r.URL.Query().Get("q"))
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_image_search_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, results)
}

func (a *API) dockerImagePull(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.PullImageRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	task, err := a.docker.StartPullImage(r.Context(), body)
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_image_pull_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusAccepted, task)
}

func (a *API) dockerImagePulls(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	tasks, err := a.docker.ImagePulls(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "docker_image_pulls_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, tasks)
}

func (a *API) dockerImageRemove(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.RemoveImageRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.docker.RemoveImage(r.Context(), body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_image_remove_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"removed": true})
}

func (a *API) dockerVolumes(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		volumes, err := a.docker.Volumes(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "docker_volumes_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, volumes)
	case http.MethodPost:
		var body hdocker.CreateVolumeRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		volume, err := a.docker.CreateVolume(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_volume_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, volume)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) dockerVolumeRemove(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.RemoveVolumeRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.docker.RemoveVolume(r.Context(), body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_volume_remove_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"removed": true})
}

func (a *API) dockerNetworks(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		networks, err := a.docker.Networks(r.Context())
		if err != nil {
			platform.WriteError(w, r, http.StatusInternalServerError, "docker_networks_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, networks)
	case http.MethodPost:
		var body hdocker.CreateNetworkRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		network, err := a.docker.CreateNetwork(r.Context(), body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_network_create_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, network)
	default:
		allowMethod(w, r, http.MethodGet, http.MethodPost)
	}
}

func (a *API) dockerNetworkRemove(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.RemoveNetworkRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.docker.RemoveNetwork(r.Context(), body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_network_remove_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"removed": true})
}

func (a *API) dockerNetworkConnect(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.NetworkConnectRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.docker.ConnectNetwork(r.Context(), body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_network_connect_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"connected": true})
}

func (a *API) dockerNetworkDisconnect(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	var body hdocker.NetworkDisconnectRequest
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	if err := a.docker.DisconnectNetwork(r.Context(), body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "docker_network_disconnect_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, map[string]any{"disconnected": true})
}

func (a *API) dockerContainerByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/docker/containers/"), "/"), "/")
	if len(parts) == 1 && parts[0] != "" {
		if !allowMethod(w, r, http.MethodDelete) {
			return
		}
		var body hdocker.RemoveContainerRequest
		if r.ContentLength != 0 {
			if err := decodeJSON(r, &body); err != nil {
				platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
				return
			}
		}
		if err := a.docker.RemoveContainer(r.Context(), parts[0], body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_container_remove_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, map[string]any{"removed": true})
		return
	}
	if len(parts) < 2 {
		platform.WriteError(w, r, http.StatusNotFound, "docker_route_not_found", "docker route not found")
		return
	}
	id := parts[0]
	action := parts[1]
	switch action {
	case "logs":
		if !allowMethod(w, r, http.MethodGet) {
			return
		}
		tail, _ := strconv.Atoi(r.URL.Query().Get("tail"))
		logs, err := a.docker.Logs(r.Context(), id, tail)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "docker_container_not_found", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapDockerLogMessages(logs))
	case "terminal":
		a.dockerContainerTerminal(w, r, id)
	case "start", "stop", "restart", "complete-restart":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		container, err := a.runDockerAction(r.Context(), id, action)
		if err != nil {
			platform.WriteError(w, r, http.StatusNotFound, "docker_action_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapDockerContainer(container))
	case "exec":
		if !allowMethod(w, r, http.MethodPost) {
			return
		}
		var body hdocker.ContainerExecRequest
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		result, err := a.docker.Exec(r.Context(), id, body)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_exec_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, result)
	case "limits":
		if !allowMethod(w, r, http.MethodPut) {
			return
		}
		var body struct {
			CPU         int `json:"cpu"`
			MemoryMB    int `json:"memoryMb"`
			LimitCPU    int `json:"limitCpu"`
			LimitMemory int `json:"limitMemory"`
		}
		if err := decodeJSON(r, &body); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		limit := hdocker.ResourceLimit{CPU: firstNonZero(body.CPU, body.LimitCPU), MemoryMB: firstNonZero(body.MemoryMB, body.LimitMemory)}
		container, err := a.docker.UpdateLimits(r.Context(), id, limit)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "docker_limit_failed", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, mapDockerContainer(container))
	default:
		platform.WriteError(w, r, http.StatusNotFound, "docker_route_not_found", "docker route not found")
	}
}

func (a *API) remoteStatus(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	status, err := a.remote.Status(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_status_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapRemoteStatus(status))
}

func (a *API) remoteStartChannel(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	status, err := a.remote.StartChannel(r.Context())
	writeRemoteStatus(w, r, status, err)
}

func (a *API) remoteStopChannel(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	status, err := a.remote.StopChannel(r.Context())
	writeRemoteStatus(w, r, status, err)
}

func (a *API) remoteTunnelMode(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	var body struct {
		Mode remote.TunnelMode `json:"mode"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	status, err := a.remote.UpdateTunnelMode(r.Context(), body.Mode)
	writeRemoteStatus(w, r, status, err)
}

func (a *API) remoteMFA(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	var body struct {
		Enabled bool `json:"enabled"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	status, err := a.remote.UpdateMFA(r.Context(), body.Enabled)
	writeRemoteStatus(w, r, status, err)
}

func (a *API) remotePolicy(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPut) {
		return
	}
	var body struct {
		Key string `json:"key"`
	}
	if err := decodeJSON(r, &body); err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
		return
	}
	status, err := a.remote.SelectPolicy(r.Context(), body.Key)
	writeRemoteStatus(w, r, status, err)
}

func (a *API) remoteDomainToken(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	token, err := a.remote.CreateDomainToken(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_token_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, token)
}

func (a *API) remoteRotateDomainToken(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	token, err := a.remote.RotateDomainToken(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_token_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, token)
}

func (a *API) remoteDevices(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	devices, err := a.remote.Devices(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_devices_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, devices)
}

func (a *API) remoteDeviceByID(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/remote/devices/"), "/"), "/")
	if len(parts) != 2 {
		platform.WriteError(w, r, http.StatusNotFound, "remote_device_route_not_found", "remote device route not found")
		return
	}
	var (
		device remote.BoundDevice
		err    error
	)
	switch parts[1] {
	case "bind":
		device, err = a.remote.BindDevice(r.Context(), parts[0])
	case "unbind":
		device, err = a.remote.UnbindDevice(r.Context(), parts[0])
	default:
		platform.WriteError(w, r, http.StatusNotFound, "remote_device_route_not_found", "remote device route not found")
		return
	}
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "remote_device_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, device)
}

func (a *API) remoteLoginAlerts(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	alerts, err := a.remote.LoginAlerts(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_alerts_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, alerts)
}

func (a *API) remoteShareScan(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodPost) {
		return
	}
	result, err := a.remote.ScanShareLinks(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "remote_share_scan_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, result)
}

func (a *API) runDockerAction(ctx context.Context, id string, action string) (hdocker.Container, error) {
	switch action {
	case "start":
		return a.docker.Start(ctx, id)
	case "stop":
		return a.docker.Stop(ctx, id)
	case "restart":
		return a.docker.Restart(ctx, id)
	case "complete-restart":
		return a.docker.CompleteRestart(ctx, id)
	default:
		return hdocker.Container{}, fmt.Errorf("unsupported docker action: %s", action)
	}
}

func writeDownloadTaskResult(w http.ResponseWriter, r *http.Request, task downloads.DownloadTask, err error) {
	if err != nil {
		platform.WriteError(w, r, http.StatusNotFound, "download_task_not_found", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, task)
}

func writeRemoteStatus(w http.ResponseWriter, r *http.Request, status remote.RemoteStatus, err error) {
	if err != nil {
		platform.WriteError(w, r, http.StatusBadRequest, "remote_action_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapRemoteStatus(status))
}

func mapStoragePools(pools []storage.StoragePool) []map[string]any {
	out := make([]map[string]any, 0, len(pools))
	for _, pool := range pools {
		out = append(out, map[string]any{
			"id":        pool.ID,
			"name":      pool.Name,
			"type":      pool.Type,
			"used":      pool.UsedPercent,
			"total":     pool.Total,
			"health":    pool.Health,
			"temp":      pool.Temperature,
			"mountPath": pool.MountPath,
		})
	}
	return out
}

func mapStorageDisks(disks []storage.Disk) []map[string]any {
	out := make([]map[string]any, 0, len(disks))
	for _, disk := range disks {
		out = append(out, mapStorageDisk(disk))
	}
	return out
}

func mapStorageDisk(disk storage.Disk) map[string]any {
	return map[string]any{
		"slot":           disk.Slot,
		"size":           disk.Size,
		"state":          disk.State,
		"temp":           disk.Temperature,
		"serial":         disk.Serial,
		"health":         disk.Health,
		"role":           disk.Role,
		"poolId":         disk.PoolID,
		"model":          disk.Model,
		"interface":      disk.Interface,
		"devicePath":     disk.DevicePath,
		"deviceType":     disk.DeviceType,
		"mediaType":      disk.MediaType,
		"rotational":     disk.Rotational,
		"systemDisk":     disk.SystemDisk,
		"fileSystem":     disk.FileSystem,
		"mountPath":      disk.MountPath,
		"standbyMinutes": disk.StandbyMinutes,
		"ssdCache":       disk.SSDCache,
		"cacheMode":      disk.CacheMode,
		"partitions":     disk.Partitions,
	}
}

func mapDockerContainers(containers []hdocker.Container) []map[string]any {
	out := make([]map[string]any, 0, len(containers))
	for _, container := range containers {
		out = append(out, mapDockerContainer(container))
	}
	return out
}

func mapDockerContainer(container hdocker.Container) map[string]any {
	return map[string]any{
		"id":          container.ID,
		"name":        container.Name,
		"image":       container.Image,
		"stack":       container.Stack,
		"status":      container.Status,
		"cpu":         container.CPU,
		"memory":      container.Memory,
		"memoryText":  container.MemoryText,
		"ports":       container.Ports,
		"mounts":      container.Mounts,
		"env":         container.Env,
		"limitCpu":    container.Limit.CPU,
		"limitMemory": container.Limit.MemoryMB,
		"restarts":    container.Restarts,
		"isolation":   container.Isolation,
	}
}

func mapDockerLogMessages(logs []hdocker.ContainerLog) []string {
	out := make([]string, 0, len(logs))
	for _, log := range logs {
		out = append(out, log.Message)
	}
	return out
}

func mapRemoteStatus(status remote.RemoteStatus) map[string]any {
	return map[string]any{
		"enabled":          status.ChannelEnabled,
		"channelEnabled":   status.ChannelEnabled,
		"channelState":     status.ChannelState,
		"domain":           status.Domain,
		"tunnelMode":       status.TunnelMode,
		"tunnelState":      status.TunnelState,
		"mfaEnabled":       status.MFAEnabled,
		"token":            status.Token,
		"boundDeviceCount": status.BoundDeviceCount,
		"deviceCount":      status.DeviceCount,
		"activePolicy":     status.ActivePolicy,
		"policies":         status.Policies,
		"feedback":         status.Feedback,
	}
}

func firstNonZero(values ...int) int {
	for _, value := range values {
		if value != 0 {
			return value
		}
	}
	return 0
}
