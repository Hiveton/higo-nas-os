package httpapi

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"strings"

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

func (a *API) storageDisks(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	disks, err := a.storage.Disks(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "storage_disks_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapStorageDisks(disks))
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
		result, err := a.downloads.DeleteTask(r.Context(), id)
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
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	containers, err := a.docker.Containers(r.Context())
	if err != nil {
		platform.WriteError(w, r, http.StatusInternalServerError, "docker_containers_failed", err.Error())
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, mapDockerContainers(containers))
}

func (a *API) dockerContainerByID(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.Trim(strings.TrimPrefix(r.URL.Path, "/api/v1/docker/containers/"), "/"), "/")
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
			"id":     pool.ID,
			"name":   pool.Name,
			"type":   pool.Type,
			"used":   pool.UsedPercent,
			"total":  pool.Total,
			"health": pool.Health,
			"temp":   pool.Temperature,
		})
	}
	return out
}

func mapStorageDisks(disks []storage.Disk) []map[string]any {
	out := make([]map[string]any, 0, len(disks))
	for _, disk := range disks {
		out = append(out, map[string]any{
			"slot":   disk.Slot,
			"size":   disk.Size,
			"state":  disk.State,
			"temp":   disk.Temperature,
			"serial": disk.Serial,
			"health": disk.Health,
			"role":   disk.Role,
			"poolId": disk.PoolID,
		})
	}
	return out
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
