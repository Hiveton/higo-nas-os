package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

func TestStatefulAPIsPersistAcrossRouterRestart(t *testing.T) {
	cfg := platform.Config{Environment: "test", Version: "test", StateDir: t.TempDir()}
	router := httpapi.NewRouter(httpapi.Dependencies{Config: cfg})

	request(t, router, http.MethodPut, "/api/v1/desktop/session", `{"activeWindowId":"docker","assistantVisible":true,"pinnedDockAppIds":["docker","app-center"]}`)
	request(t, router, http.MethodPut, "/api/v1/settings", `{"model":{"mode":"enterprise_local","cloudEnabled":true},"privacy":{"sensitiveDataLocalOnly":true,"auditRetentionDays":365}}`)
	request(t, router, http.MethodPost, "/api/v1/downloads/tasks", `{"link":"https://example.com/ubuntu-26.04.iso","source":"HTTP","name":"Ubuntu 26.04 ISO","category":"软件"}`)
	request(t, router, http.MethodPut, "/api/v1/downloads/speed-profile", `{"name":"夜间全速"}`)
	request(t, router, http.MethodPut, "/api/v1/remote/mfa", `{"enabled":false}`)
	request(t, router, http.MethodPut, "/api/v1/remote/policy", `{"key":"team"}`)
	request(t, router, http.MethodPost, "/api/v1/remote/devices/ipad/bind", `{}`)
	request(t, router, http.MethodPost, "/api/v1/backups/jobs/family-photo/pause", `{}`)
	request(t, router, http.MethodPost, "/api/v1/app-center/apps/paperless/install", `{}`)
	request(t, router, http.MethodPost, "/api/v1/monitoring/alerts", `{"metric":"cpu","range":"1H","title":"CPU persistent smoke"}`)
	var storageTask struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	requestJSON(t, router, http.MethodPost, "/api/v1/storage/tasks/smart-scan", `{"targetSlot":"1"}`, &storageTask)
	request(t, router, http.MethodPut, "/api/v1/docker/containers/jellyfin/limits", `{"limitCpu":5,"limitMemory":6144}`)
	request(t, router, http.MethodPost, "/api/v1/security/risk-actions/r2/block", `{"actorId":"tester","reason":"persistent smoke"}`)
	request(t, router, http.MethodDelete, "/api/v1/shares/s2", ``)
	var fileSearch struct {
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	getJSON(t, router, "/api/v1/files/search?q=%E5%90%88%E5%90%8C", &fileSearch)
	if len(fileSearch.Data) == 0 {
		t.Fatal("expected a file search hit for tag persistence")
	}
	taggedFileID := fileSearch.Data[0].ID
	request(t, router, http.MethodPost, "/api/v1/files/"+taggedFileID+"/tags", `{"tags":["持久化标签"]}`)
	request(t, router, http.MethodPost, "/api/v1/media/albums", `{"name":"持久化家庭相册","type":"家庭相册","itemIds":[1]}`)
	request(t, router, http.MethodPost, "/api/v1/assistant/threads/thread-current/messages", `{"actorId":"tester","text":"整理 下载目录 并生成计划","modelPolicy":"local-first"}`)
	var workflowRun struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	requestJSON(t, router, http.MethodPost, "/api/v1/workflows/runs", `{"templateId":"ops-agent","goal":"检查真实落盘状态","scopes":["monitoring"],"actorId":"tester"}`, &workflowRun)
	var stewardPreview struct {
		Data struct {
			ConfirmationID string `json:"confirmationId"`
		} `json:"data"`
	}
	requestJSON(t, router, http.MethodPost, "/api/v1/steward/suggestions/download-cleanup/preview", `{}`, &stewardPreview)
	request(t, router, http.MethodPost, "/api/v1/steward/suggestions/download-cleanup/confirm", `{"actorId":"tester","confirmationId":"`+stewardPreview.Data.ConfirmationID+`"}`)

	restarted := httpapi.NewRouter(httpapi.Dependencies{Config: cfg})

	var desktop struct {
		OK   bool `json:"ok"`
		Data struct {
			ActiveWindowID   string   `json:"activeWindowId"`
			AssistantVisible bool     `json:"assistantVisible"`
			PinnedDockAppIDs []string `json:"pinnedDockAppIds"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/desktop/session", &desktop)
	if desktop.Data.ActiveWindowID != "docker" || !desktop.Data.AssistantVisible || len(desktop.Data.PinnedDockAppIDs) != 2 {
		t.Fatalf("desktop session was not persisted: %#v", desktop.Data)
	}

	var settings struct {
		Data struct {
			Model struct {
				Mode         string `json:"mode"`
				CloudEnabled bool   `json:"cloudEnabled"`
			} `json:"model"`
			Privacy struct {
				AuditRetentionDays int `json:"auditRetentionDays"`
			} `json:"privacy"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/settings", &settings)
	if settings.Data.Model.Mode != "enterprise_local" || settings.Data.Model.CloudEnabled || settings.Data.Privacy.AuditRetentionDays != 365 {
		t.Fatalf("settings were not persisted and normalized: %#v", settings.Data)
	}

	var downloads struct {
		Data []struct {
			Name string `json:"name"`
			Link string `json:"link"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/downloads/tasks", &downloads)
	if !hasDownload(downloads.Data, "https://example.com/ubuntu-26.04.iso") {
		t.Fatalf("download task was not persisted: %#v", downloads.Data)
	}

	var profiles struct {
		Data []struct {
			Name   string `json:"name"`
			Active bool   `json:"active"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/downloads/speed-profiles", &profiles)
	if !hasActiveProfile(profiles.Data, "夜间全速") {
		t.Fatalf("speed profile was not persisted: %#v", profiles.Data)
	}

	var remoteStatus struct {
		Data struct {
			MFAEnabled   bool `json:"mfaEnabled"`
			ActivePolicy struct {
				Key string `json:"key"`
			} `json:"activePolicy"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/remote/status", &remoteStatus)
	if remoteStatus.Data.MFAEnabled || remoteStatus.Data.ActivePolicy.Key != "team" {
		t.Fatalf("remote status was not persisted: %#v", remoteStatus.Data)
	}

	var devices struct {
		Data []struct {
			ID    string `json:"id"`
			Bound bool   `json:"bound"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/remote/devices", &devices)
	if !hasBoundDevice(devices.Data, "ipad") {
		t.Fatalf("remote device binding was not persisted: %#v", devices.Data)
	}

	var backupJobs struct {
		Data []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/backups/jobs", &backupJobs)
	if !hasBackupState(backupJobs.Data, "family-photo", "已暂停") {
		t.Fatalf("backup job state was not persisted: %#v", backupJobs.Data)
	}

	var apps struct {
		Data []struct {
			ID        string `json:"id"`
			Installed bool   `json:"installed"`
			Running   bool   `json:"running"`
			Version   string `json:"version"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/app-center/apps", &apps)
	if !hasInstalledApp(apps.Data, "paperless") {
		t.Fatalf("app center state was not persisted: %#v", apps.Data)
	}

	var alerts struct {
		Data []struct {
			Title string `json:"title"`
			State string `json:"state"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/monitoring/alerts", &alerts)
	if !hasAlert(alerts.Data, "CPU persistent smoke") {
		t.Fatalf("monitoring alert was not persisted: %#v", alerts.Data)
	}

	var storedTask struct {
		Data struct {
			ID         string `json:"id"`
			TargetSlot string `json:"targetSlot"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/storage/tasks/"+storageTask.Data.ID, &storedTask)
	if storedTask.Data.ID != storageTask.Data.ID || storedTask.Data.TargetSlot != "1" {
		t.Fatalf("storage task was not persisted: %#v", storedTask.Data)
	}

	var containers struct {
		Data []struct {
			ID          string `json:"id"`
			LimitCPU    int    `json:"limitCpu"`
			LimitMemory int    `json:"limitMemory"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/docker/containers", &containers)
	if !hasContainerLimits(containers.Data, "jellyfin", 5, 6144) {
		t.Fatalf("docker limits were not persisted: %#v", containers.Data)
	}

	var risks struct {
		Data []struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/security/risk-actions", &risks)
	if !hasRiskState(risks.Data, "r2", "已阻止") {
		t.Fatalf("security risk state was not persisted: %#v", risks.Data)
	}

	var shares struct {
		Data []struct {
			ID     string `json:"id"`
			Active bool   `json:"active"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/shares", &shares)
	if !hasInactiveShare(shares.Data, "s2") {
		t.Fatalf("share revoke state was not persisted: %#v", shares.Data)
	}

	var taggedFile struct {
		Data struct {
			Tags []string `json:"tags"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/files/"+taggedFileID, &taggedFile)
	if !hasString(taggedFile.Data.Tags, "持久化标签") {
		t.Fatalf("file tags were not persisted: %#v", taggedFile.Data.Tags)
	}

	var albums struct {
		Data []struct {
			Name string `json:"name"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/media/albums", &albums)
	if !hasAlbum(albums.Data, "持久化家庭相册") {
		t.Fatalf("media album was not persisted: %#v", albums.Data)
	}

	var thread struct {
		Data struct {
			Messages []struct {
				Text            string `json:"text"`
				PendingActionID string `json:"pendingActionId"`
			} `json:"messages"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/assistant/threads/thread-current", &thread)
	if !hasMessage(thread.Data.Messages, "整理 下载目录 并生成计划") || !hasPendingAction(thread.Data.Messages) {
		t.Fatalf("assistant thread/action state was not persisted: %#v", thread.Data)
	}

	var workflowEvents struct {
		Data []struct {
			RunID string `json:"runId"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/workflows/runs/"+workflowRun.Data.ID+"/events", &workflowEvents)
	if len(workflowEvents.Data) == 0 || workflowEvents.Data[0].RunID != workflowRun.Data.ID {
		t.Fatalf("workflow run events were not persisted: %#v", workflowEvents.Data)
	}

	var suggestions struct {
		Data []struct {
			ID     string `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	getJSON(t, restarted, "/api/v1/steward/suggestions", &suggestions)
	if !hasSuggestionStatus(suggestions.Data, "download-cleanup", "confirmed") {
		t.Fatalf("steward suggestion state was not persisted: %#v", suggestions.Data)
	}
}

func request(t *testing.T, router http.Handler, method string, path string, body string) {
	t.Helper()
	requestJSON(t, router, method, path, body, nil)
}

func requestJSON(t *testing.T, router http.Handler, method string, path string, body string, target any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(method, path, bytes.NewBufferString(body))
	router.ServeHTTP(rec, req)
	if rec.Code < 200 || rec.Code >= 300 {
		t.Fatalf("%s %s got HTTP %d: %s", method, path, rec.Code, rec.Body.String())
	}
	if target != nil {
		if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
			t.Fatalf("decode %s %s: %v", method, path, err)
		}
	}
}

func getJSON(t *testing.T, router http.Handler, path string, target any) {
	t.Helper()
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	router.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s got HTTP %d: %s", path, rec.Code, rec.Body.String())
	}
	if err := json.Unmarshal(rec.Body.Bytes(), target); err != nil {
		t.Fatalf("decode %s: %v", path, err)
	}
}

func hasDownload(tasks []struct {
	Name string `json:"name"`
	Link string `json:"link"`
}, link string) bool {
	for _, task := range tasks {
		if task.Link == link {
			return true
		}
	}
	return false
}

func hasActiveProfile(profiles []struct {
	Name   string `json:"name"`
	Active bool   `json:"active"`
}, name string) bool {
	for _, profile := range profiles {
		if profile.Name == name && profile.Active {
			return true
		}
	}
	return false
}

func hasBoundDevice(devices []struct {
	ID    string `json:"id"`
	Bound bool   `json:"bound"`
}, id string) bool {
	for _, device := range devices {
		if device.ID == id && device.Bound {
			return true
		}
	}
	return false
}

func hasBackupState(jobs []struct {
	ID    string `json:"id"`
	State string `json:"state"`
}, id string, state string) bool {
	for _, job := range jobs {
		if job.ID == id && job.State == state {
			return true
		}
	}
	return false
}

func hasInstalledApp(apps []struct {
	ID        string `json:"id"`
	Installed bool   `json:"installed"`
	Running   bool   `json:"running"`
	Version   string `json:"version"`
}, id string) bool {
	for _, app := range apps {
		if app.ID == id && app.Installed && app.Running && app.Version != "" {
			return true
		}
	}
	return false
}

func hasAlert(alerts []struct {
	Title string `json:"title"`
	State string `json:"state"`
}, title string) bool {
	for _, alert := range alerts {
		if alert.Title == title {
			return true
		}
	}
	return false
}

func hasContainerLimits(containers []struct {
	ID          string `json:"id"`
	LimitCPU    int    `json:"limitCpu"`
	LimitMemory int    `json:"limitMemory"`
}, id string, cpu int, memory int) bool {
	for _, container := range containers {
		if container.ID == id && container.LimitCPU == cpu && container.LimitMemory == memory {
			return true
		}
	}
	return false
}

func hasRiskState(risks []struct {
	ID    string `json:"id"`
	State string `json:"state"`
}, id string, state string) bool {
	for _, risk := range risks {
		if risk.ID == id && risk.State == state {
			return true
		}
	}
	return false
}

func hasInactiveShare(shares []struct {
	ID     string `json:"id"`
	Active bool   `json:"active"`
}, id string) bool {
	for _, share := range shares {
		if share.ID == id && !share.Active {
			return true
		}
	}
	return false
}

func hasString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}

func hasAlbum(albums []struct {
	Name string `json:"name"`
}, name string) bool {
	for _, album := range albums {
		if album.Name == name {
			return true
		}
	}
	return false
}

func hasMessage(messages []struct {
	Text            string `json:"text"`
	PendingActionID string `json:"pendingActionId"`
}, text string) bool {
	for _, message := range messages {
		if message.Text == text {
			return true
		}
	}
	return false
}

func hasPendingAction(messages []struct {
	Text            string `json:"text"`
	PendingActionID string `json:"pendingActionId"`
}) bool {
	for _, message := range messages {
		if message.PendingActionID != "" {
			return true
		}
	}
	return false
}

func hasSuggestionStatus(suggestions []struct {
	ID     string `json:"id"`
	Status string `json:"status"`
}, id string, status string) bool {
	for _, suggestion := range suggestions {
		if suggestion.ID == id && suggestion.Status == status {
			return true
		}
	}
	return false
}
