package httpapi_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"higoos/server-go/internal/devstub"
	"higoos/server-go/internal/httpapi"
	"higoos/server-go/internal/platform"
)

func TestHealthzReturnsOK(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/healthz", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	var body map[string]any
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode health body: %v", err)
	}
	if body["status"] != "ok" {
		t.Fatalf("expected status ok, got %#v", body["status"])
	}
	if rec.Header().Get("X-Request-ID") == "" {
		t.Fatal("expected response request id header")
	}
}

func TestFileEndpointsSearchPreviewAndTag(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodGet, "/api/v1/files/search?q=合同", nil)
	router.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected files search HTTP 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var searchBody struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID        string   `json:"id"`
			Name      string   `json:"name"`
			AISummary string   `json:"aiSummary"`
			Tags      []string `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchBody); err != nil {
		t.Fatalf("decode files search: %v", err)
	}
	if !searchBody.OK || len(searchBody.Data) == 0 {
		t.Fatalf("expected search hit, got ok=%v len=%d", searchBody.OK, len(searchBody.Data))
	}
	fileID := searchBody.Data[0].ID

	previewRec := httptest.NewRecorder()
	previewReq := httptest.NewRequest(http.MethodGet, "/api/v1/files/"+fileID+"/preview", nil)
	router.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected preview HTTP 200, got %d: %s", previewRec.Code, previewRec.Body.String())
	}
	var previewBody struct {
		Data struct {
			Supported bool   `json:"supported"`
			Summary   string `json:"summary"`
		} `json:"data"`
	}
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewBody); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if !previewBody.Data.Supported || previewBody.Data.Summary == "" {
		t.Fatalf("expected supported preview summary, got %#v", previewBody.Data)
	}

	tagRec := httptest.NewRecorder()
	tagReq := httptest.NewRequest(http.MethodPost, "/api/v1/files/"+fileID+"/tags", bytes.NewBufferString(`{"tags":["AI 已处理"]}`))
	router.ServeHTTP(tagRec, tagReq)
	if tagRec.Code != http.StatusOK {
		t.Fatalf("expected tag HTTP 200, got %d: %s", tagRec.Code, tagRec.Body.String())
	}
	var tagBody struct {
		Data struct {
			Tags []string `json:"tags"`
		} `json:"data"`
	}
	if err := json.Unmarshal(tagRec.Body.Bytes(), &tagBody); err != nil {
		t.Fatalf("decode tag response: %v", err)
	}
	if !containsString(tagBody.Data.Tags, "AI 已处理") {
		t.Fatalf("expected added smart tag, got %#v", tagBody.Data.Tags)
	}
}

func TestMonitoringAndSettingsEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	metricsRec := httptest.NewRecorder()
	metricsReq := httptest.NewRequest(http.MethodGet, "/api/v1/monitoring/metrics/current", nil)
	router.ServeHTTP(metricsRec, metricsReq)
	if metricsRec.Code != http.StatusOK {
		t.Fatalf("expected metrics HTTP 200, got %d: %s", metricsRec.Code, metricsRec.Body.String())
	}
	var metricsBody struct {
		Data []struct {
			Key   string  `json:"key"`
			Value float64 `json:"value"`
		} `json:"data"`
	}
	if err := json.Unmarshal(metricsRec.Body.Bytes(), &metricsBody); err != nil {
		t.Fatalf("decode metrics: %v", err)
	}
	if len(metricsBody.Data) == 0 || metricsBody.Data[0].Key == "" {
		t.Fatalf("expected metrics data, got %#v", metricsBody.Data)
	}

	alertRec := httptest.NewRecorder()
	alertReq := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/alerts", bytes.NewBufferString(`{"metric":"cpu","range":"1H"}`))
	router.ServeHTTP(alertRec, alertReq)
	if alertRec.Code != http.StatusOK {
		t.Fatalf("expected alert create HTTP 200, got %d: %s", alertRec.Code, alertRec.Body.String())
	}
	var alertBody struct {
		Data struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(alertRec.Body.Bytes(), &alertBody); err != nil {
		t.Fatalf("decode alert: %v", err)
	}
	if alertBody.Data.ID == "" {
		t.Fatal("expected alert id")
	}

	muteRec := httptest.NewRecorder()
	muteReq := httptest.NewRequest(http.MethodPost, "/api/v1/monitoring/alerts/"+alertBody.Data.ID+"/mute", bytes.NewBufferString(`{"muted":true}`))
	router.ServeHTTP(muteRec, muteReq)
	if muteRec.Code != http.StatusOK {
		t.Fatalf("expected alert mute HTTP 200, got %d: %s", muteRec.Code, muteRec.Body.String())
	}

	settingsRec := httptest.NewRecorder()
	settingsReq := httptest.NewRequest(http.MethodPut, "/api/v1/settings", bytes.NewBufferString(`{"model":{"mode":"enterprise_local","cloudEnabled":true},"privacy":{"sensitiveDataLocalOnly":true,"auditRetentionDays":365}}`))
	router.ServeHTTP(settingsRec, settingsReq)
	if settingsRec.Code != http.StatusOK {
		t.Fatalf("expected settings update HTTP 200, got %d: %s", settingsRec.Code, settingsRec.Body.String())
	}
	var settingsBody struct {
		Data struct {
			Model struct {
				CloudEnabled bool `json:"cloudEnabled"`
			} `json:"model"`
		} `json:"data"`
	}
	if err := json.Unmarshal(settingsRec.Body.Bytes(), &settingsBody); err != nil {
		t.Fatalf("decode settings: %v", err)
	}
	if settingsBody.Data.Model.CloudEnabled {
		t.Fatal("expected enterprise local mode to disable cloud")
	}
}

func TestStorageDownloadDockerRemoteEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	poolsRec := httptest.NewRecorder()
	poolsReq := httptest.NewRequest(http.MethodGet, "/api/v1/storage/pools", nil)
	router.ServeHTTP(poolsRec, poolsReq)
	if poolsRec.Code != http.StatusOK {
		t.Fatalf("expected storage pools HTTP 200, got %d: %s", poolsRec.Code, poolsRec.Body.String())
	}
	var poolsBody struct {
		Data []struct {
			Name string `json:"name"`
			Used int    `json:"used"`
			Temp string `json:"temp"`
		} `json:"data"`
	}
	if err := json.Unmarshal(poolsRec.Body.Bytes(), &poolsBody); err != nil {
		t.Fatalf("decode pools: %v", err)
	}
	if len(poolsBody.Data) == 0 || poolsBody.Data[0].Name == "" || poolsBody.Data[0].Temp == "" {
		t.Fatalf("unexpected pools: %#v", poolsBody.Data)
	}
	for _, pool := range poolsBody.Data {
		if strings.Contains(pool.Name, "存储池 1") {
			t.Fatalf("storage pools should come from host volumes, got fixture pool: %#v", poolsBody.Data)
		}
	}

	downloadRec := httptest.NewRecorder()
	downloadReq := httptest.NewRequest(http.MethodPost, "/api/v1/downloads/tasks", bytes.NewBufferString(`{"link":"magnet:?xt=urn:btih:test","source":"磁力"}`))
	router.ServeHTTP(downloadRec, downloadReq)
	if downloadRec.Code != http.StatusOK {
		t.Fatalf("expected download create HTTP 200, got %d: %s", downloadRec.Code, downloadRec.Body.String())
	}
	var downloadBody struct {
		Data struct {
			ID     int    `json:"id"`
			Status string `json:"status"`
		} `json:"data"`
	}
	if err := json.Unmarshal(downloadRec.Body.Bytes(), &downloadBody); err != nil {
		t.Fatalf("decode download: %v", err)
	}
	if downloadBody.Data.ID == 0 || downloadBody.Data.Status != "下载中" {
		t.Fatalf("unexpected download task: %#v", downloadBody.Data)
	}

	dockerRec := httptest.NewRecorder()
	dockerReq := httptest.NewRequest(http.MethodPost, "/api/v1/docker/containers/gateway/start", nil)
	router.ServeHTTP(dockerRec, dockerReq)
	if dockerRec.Code != http.StatusOK {
		t.Fatalf("expected docker start HTTP 200, got %d: %s", dockerRec.Code, dockerRec.Body.String())
	}
	var dockerBody struct {
		Data struct {
			Status      string `json:"status"`
			LimitCPU    int    `json:"limitCpu"`
			LimitMemory int    `json:"limitMemory"`
		} `json:"data"`
	}
	if err := json.Unmarshal(dockerRec.Body.Bytes(), &dockerBody); err != nil {
		t.Fatalf("decode docker: %v", err)
	}
	if dockerBody.Data.Status != "运行中" || dockerBody.Data.LimitCPU == 0 || dockerBody.Data.LimitMemory == 0 {
		t.Fatalf("unexpected docker payload: %#v", dockerBody.Data)
	}

	remoteRec := httptest.NewRecorder()
	remoteReq := httptest.NewRequest(http.MethodPut, "/api/v1/remote/tunnel-mode", bytes.NewBufferString(`{"mode":"直连优先"}`))
	router.ServeHTTP(remoteRec, remoteReq)
	if remoteRec.Code != http.StatusOK {
		t.Fatalf("expected remote tunnel HTTP 200, got %d: %s", remoteRec.Code, remoteRec.Body.String())
	}
	var remoteBody struct {
		Data struct {
			Enabled    bool   `json:"enabled"`
			TunnelMode string `json:"tunnelMode"`
			Domain     string `json:"domain"`
		} `json:"data"`
	}
	if err := json.Unmarshal(remoteRec.Body.Bytes(), &remoteBody); err != nil {
		t.Fatalf("decode remote: %v", err)
	}
	if !remoteBody.Data.Enabled || remoteBody.Data.TunnelMode != "直连优先" || remoteBody.Data.Domain == "" {
		t.Fatalf("unexpected remote payload: %#v", remoteBody.Data)
	}
}

func TestRemoteMFAAndPolicyEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	mfaRec := httptest.NewRecorder()
	mfaReq := httptest.NewRequest(http.MethodPut, "/api/v1/remote/mfa", bytes.NewBufferString(`{"enabled":false}`))
	router.ServeHTTP(mfaRec, mfaReq)
	if mfaRec.Code != http.StatusOK {
		t.Fatalf("expected remote MFA HTTP 200, got %d: %s", mfaRec.Code, mfaRec.Body.String())
	}
	var mfaBody struct {
		Data struct {
			MFAEnabled bool   `json:"mfaEnabled"`
			Feedback   string `json:"feedback"`
		} `json:"data"`
	}
	if err := json.Unmarshal(mfaRec.Body.Bytes(), &mfaBody); err != nil {
		t.Fatalf("decode remote MFA: %v", err)
	}
	if mfaBody.Data.MFAEnabled || mfaBody.Data.Feedback == "" {
		t.Fatalf("expected MFA disabled with feedback, got %#v", mfaBody.Data)
	}

	policyRec := httptest.NewRecorder()
	policyReq := httptest.NewRequest(http.MethodPut, "/api/v1/remote/policy", bytes.NewBufferString(`{"key":"team"}`))
	router.ServeHTTP(policyRec, policyReq)
	if policyRec.Code != http.StatusOK {
		t.Fatalf("expected remote policy HTTP 200, got %d: %s", policyRec.Code, policyRec.Body.String())
	}
	var policyBody struct {
		Data struct {
			ActivePolicy struct {
				Key  string `json:"key"`
				Risk string `json:"risk"`
			} `json:"activePolicy"`
			Feedback string `json:"feedback"`
		} `json:"data"`
	}
	if err := json.Unmarshal(policyRec.Body.Bytes(), &policyBody); err != nil {
		t.Fatalf("decode remote policy: %v", err)
	}
	if policyBody.Data.ActivePolicy.Key != "team" || policyBody.Data.ActivePolicy.Risk != "中风险" || policyBody.Data.Feedback == "" {
		t.Fatalf("expected team policy with feedback, got %#v", policyBody.Data)
	}
}

func TestDockerLimitsLogsAndCompleteRestartEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	limitsRec := httptest.NewRecorder()
	limitsReq := httptest.NewRequest(http.MethodPut, "/api/v1/docker/containers/jellyfin/limits", bytes.NewBufferString(`{"limitCpu":5,"limitMemory":6144}`))
	router.ServeHTTP(limitsRec, limitsReq)
	if limitsRec.Code != http.StatusOK {
		t.Fatalf("expected docker limits HTTP 200, got %d: %s", limitsRec.Code, limitsRec.Body.String())
	}
	var limitsBody struct {
		Data struct {
			LimitCPU    int    `json:"limitCpu"`
			LimitMemory int    `json:"limitMemory"`
			MemoryText  string `json:"memoryText"`
		} `json:"data"`
	}
	if err := json.Unmarshal(limitsRec.Body.Bytes(), &limitsBody); err != nil {
		t.Fatalf("decode docker limits: %v", err)
	}
	if limitsBody.Data.LimitCPU != 5 || limitsBody.Data.LimitMemory != 6144 || limitsBody.Data.MemoryText == "" {
		t.Fatalf("unexpected docker limits payload: %#v", limitsBody.Data)
	}

	restartRec := httptest.NewRecorder()
	restartReq := httptest.NewRequest(http.MethodPost, "/api/v1/docker/containers/jellyfin/restart", nil)
	router.ServeHTTP(restartRec, restartReq)
	if restartRec.Code != http.StatusOK {
		t.Fatalf("expected docker restart HTTP 200, got %d: %s", restartRec.Code, restartRec.Body.String())
	}

	completeRec := httptest.NewRecorder()
	completeReq := httptest.NewRequest(http.MethodPost, "/api/v1/docker/containers/jellyfin/complete-restart", nil)
	router.ServeHTTP(completeRec, completeReq)
	if completeRec.Code != http.StatusOK {
		t.Fatalf("expected docker complete restart HTTP 200, got %d: %s", completeRec.Code, completeRec.Body.String())
	}
	var completeBody struct {
		Data struct {
			Status   string `json:"status"`
			Restarts int    `json:"restarts"`
		} `json:"data"`
	}
	if err := json.Unmarshal(completeRec.Body.Bytes(), &completeBody); err != nil {
		t.Fatalf("decode docker complete restart: %v", err)
	}
	if completeBody.Data.Status != "运行中" || completeBody.Data.Restarts == 0 {
		t.Fatalf("unexpected docker complete restart payload: %#v", completeBody.Data)
	}

	logsRec := httptest.NewRecorder()
	logsReq := httptest.NewRequest(http.MethodGet, "/api/v1/docker/containers/jellyfin/logs?tail=3", nil)
	router.ServeHTTP(logsRec, logsReq)
	if logsRec.Code != http.StatusOK {
		t.Fatalf("expected docker logs HTTP 200, got %d: %s", logsRec.Code, logsRec.Body.String())
	}
	var logsBody struct {
		Data []string `json:"data"`
	}
	if err := json.Unmarshal(logsRec.Body.Bytes(), &logsBody); err != nil {
		t.Fatalf("decode docker logs: %v", err)
	}
	if len(logsBody.Data) == 0 || !containsString(logsBody.Data, "健康检查通过，容器已恢复服务") {
		t.Fatalf("expected complete restart log, got %#v", logsBody.Data)
	}
}

func TestBackupAndAppCenterEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	backupRec := httptest.NewRecorder()
	backupReq := httptest.NewRequest(http.MethodGet, "/api/v1/backups/jobs", nil)
	router.ServeHTTP(backupRec, backupReq)
	if backupRec.Code != http.StatusOK {
		t.Fatalf("expected backup jobs HTTP 200, got %d: %s", backupRec.Code, backupRec.Body.String())
	}
	var backupBody struct {
		Data []struct {
			ID       string `json:"id"`
			Name     string `json:"name"`
			Progress int    `json:"progress"`
		} `json:"data"`
	}
	if err := json.Unmarshal(backupRec.Body.Bytes(), &backupBody); err != nil {
		t.Fatalf("decode backup jobs: %v", err)
	}
	if len(backupBody.Data) < 2 || backupBody.Data[0].ID == "" || backupBody.Data[0].Progress == 0 {
		t.Fatalf("unexpected backup jobs payload: %#v", backupBody.Data)
	}

	runRec := httptest.NewRecorder()
	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/backups/jobs/family-photo/run", nil)
	router.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected backup run HTTP 200, got %d: %s", runRec.Code, runRec.Body.String())
	}
	var runBody struct {
		Data struct {
			ID       string `json:"id"`
			State    string `json:"state"`
			Progress int    `json:"progress"`
		} `json:"data"`
	}
	if err := json.Unmarshal(runRec.Body.Bytes(), &runBody); err != nil {
		t.Fatalf("decode backup run: %v", err)
	}
	if runBody.Data.ID != "family-photo" || runBody.Data.State != "同步中" || runBody.Data.Progress < 80 {
		t.Fatalf("unexpected backup run payload: %#v", runBody.Data)
	}

	appsRec := httptest.NewRecorder()
	appsReq := httptest.NewRequest(http.MethodGet, "/api/v1/app-center/apps", nil)
	router.ServeHTTP(appsRec, appsReq)
	if appsRec.Code != http.StatusOK {
		t.Fatalf("expected app center HTTP 200, got %d: %s", appsRec.Code, appsRec.Body.String())
	}
	var appsBody struct {
		Data []struct {
			ID              string `json:"id"`
			Name            string `json:"name"`
			UpdateAvailable bool   `json:"updateAvailable"`
		} `json:"data"`
	}
	if err := json.Unmarshal(appsRec.Body.Bytes(), &appsBody); err != nil {
		t.Fatalf("decode app center: %v", err)
	}
	if len(appsBody.Data) < 3 || appsBody.Data[0].ID == "" {
		t.Fatalf("unexpected app center payload: %#v", appsBody.Data)
	}

	updateRec := httptest.NewRecorder()
	updateReq := httptest.NewRequest(http.MethodPost, "/api/v1/app-center/apps/home-assistant/update", nil)
	router.ServeHTTP(updateRec, updateReq)
	if updateRec.Code != http.StatusOK {
		t.Fatalf("expected app update HTTP 200, got %d: %s", updateRec.Code, updateRec.Body.String())
	}
	var updateBody struct {
		Data struct {
			ID              string `json:"id"`
			Status          string `json:"status"`
			UpdateAvailable bool   `json:"updateAvailable"`
		} `json:"data"`
	}
	if err := json.Unmarshal(updateRec.Body.Bytes(), &updateBody); err != nil {
		t.Fatalf("decode app update: %v", err)
	}
	if updateBody.Data.ID != "home-assistant" || updateBody.Data.UpdateAvailable || updateBody.Data.Status != "已更新" {
		t.Fatalf("unexpected app update payload: %#v", updateBody.Data)
	}
}

func TestMediaAssistantAndAgentEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	itemsRec := httptest.NewRecorder()
	itemsReq := httptest.NewRequest(http.MethodGet, "/api/v1/media/items?dimension=timeline&facet=2026+%E6%98%A5%E8%8A%82", nil)
	router.ServeHTTP(itemsRec, itemsReq)
	if itemsRec.Code != http.StatusOK {
		t.Fatalf("expected media items HTTP 200, got %d: %s", itemsRec.Code, itemsRec.Body.String())
	}
	var itemsBody struct {
		Data []struct {
			ID    int    `json:"id"`
			Title string `json:"title"`
		} `json:"data"`
	}
	if err := json.Unmarshal(itemsRec.Body.Bytes(), &itemsBody); err != nil {
		t.Fatalf("decode media items: %v", err)
	}
	if len(itemsBody.Data) == 0 || itemsBody.Data[0].ID == 0 {
		t.Fatalf("expected media items, got %#v", itemsBody.Data)
	}

	memoryRec := httptest.NewRecorder()
	memoryReq := httptest.NewRequest(http.MethodPost, "/api/v1/media/memories", bytes.NewBufferString(`{"dimension":"timeline","facet":"2026 春节"}`))
	router.ServeHTTP(memoryRec, memoryReq)
	if memoryRec.Code != http.StatusOK {
		t.Fatalf("expected media memory HTTP 200, got %d: %s", memoryRec.Code, memoryRec.Body.String())
	}
	var taskBody struct {
		Data struct {
			ID      string `json:"id"`
			State   string `json:"state"`
			Message string `json:"message"`
		} `json:"data"`
	}
	if err := json.Unmarshal(memoryRec.Body.Bytes(), &taskBody); err != nil {
		t.Fatalf("decode memory task: %v", err)
	}
	if taskBody.Data.ID == "" || taskBody.Data.State == "" {
		t.Fatalf("expected task response, got %#v", taskBody.Data)
	}

	searchRec := httptest.NewRecorder()
	searchReq := httptest.NewRequest(http.MethodPost, "/api/v1/search/semantic", bytes.NewBufferString(`{"query":"合同 备份","scopes":["team"],"limit":2}`))
	router.ServeHTTP(searchRec, searchReq)
	if searchRec.Code != http.StatusOK {
		t.Fatalf("expected semantic search HTTP 200, got %d: %s", searchRec.Code, searchRec.Body.String())
	}
	var searchBody struct {
		Data struct {
			Answer string `json:"answer"`
			Items  []struct {
				Name      string `json:"name"`
				AISummary string `json:"aiSummary"`
			} `json:"items"`
		} `json:"data"`
	}
	if err := json.Unmarshal(searchRec.Body.Bytes(), &searchBody); err != nil {
		t.Fatalf("decode semantic search: %v", err)
	}
	if searchBody.Data.Answer == "" || len(searchBody.Data.Items) == 0 {
		t.Fatalf("expected semantic results, got %#v", searchBody.Data)
	}

	threadRec := httptest.NewRecorder()
	threadReq := httptest.NewRequest(http.MethodPost, "/api/v1/assistant/threads/thread-current/messages", bytes.NewBufferString(`{"role":"user","text":"整理 下载目录","modelPolicy":"local-first"}`))
	router.ServeHTTP(threadRec, threadReq)
	if threadRec.Code != http.StatusOK {
		t.Fatalf("expected assistant message HTTP 200, got %d: %s", threadRec.Code, threadRec.Body.String())
	}
	var messageBody struct {
		Data struct {
			Role            string `json:"role"`
			Text            string `json:"text"`
			PendingActionID string `json:"pendingActionId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(threadRec.Body.Bytes(), &messageBody); err != nil {
		t.Fatalf("decode assistant message: %v", err)
	}
	if messageBody.Data.Role != "assistant" || messageBody.Data.PendingActionID == "" {
		t.Fatalf("unexpected assistant payload: %#v", messageBody.Data)
	}

	templatesRec := httptest.NewRecorder()
	templatesReq := httptest.NewRequest(http.MethodGet, "/api/v1/agents/templates", nil)
	router.ServeHTTP(templatesRec, templatesReq)
	if templatesRec.Code != http.StatusOK {
		t.Fatalf("expected agent templates HTTP 200, got %d: %s", templatesRec.Code, templatesRec.Body.String())
	}
	var templatesBody struct {
		Data []struct {
			Name  string   `json:"name"`
			Desc  string   `json:"desc"`
			Tools []string `json:"tools"`
			Risk  string   `json:"risk"`
		} `json:"data"`
	}
	if err := json.Unmarshal(templatesRec.Body.Bytes(), &templatesBody); err != nil {
		t.Fatalf("decode templates: %v", err)
	}
	if len(templatesBody.Data) == 0 || templatesBody.Data[0].Desc == "" || len(templatesBody.Data[0].Tools) == 0 || templatesBody.Data[0].Risk == "" {
		t.Fatalf("unexpected templates: %#v", templatesBody.Data)
	}

	runRec := httptest.NewRecorder()
	runReq := httptest.NewRequest(http.MethodPost, "/api/v1/workflows/runs", bytes.NewBufferString(`{"templateId":"ops-agent","goal":"检查备份状态","scopes":["monitoring"]}`))
	router.ServeHTTP(runRec, runReq)
	if runRec.Code != http.StatusOK {
		t.Fatalf("expected workflow run HTTP 200, got %d: %s", runRec.Code, runRec.Body.String())
	}
	var runBody struct {
		Data struct {
			ID    string `json:"id"`
			State string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(runRec.Body.Bytes(), &runBody); err != nil {
		t.Fatalf("decode workflow run: %v", err)
	}
	if runBody.Data.ID == "" || runBody.Data.State != "completed" {
		t.Fatalf("unexpected workflow run: %#v", runBody.Data)
	}
}

func TestStewardAndSecurityEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	suggestionsRec := httptest.NewRecorder()
	suggestionsReq := httptest.NewRequest(http.MethodGet, "/api/v1/steward/suggestions", nil)
	router.ServeHTTP(suggestionsRec, suggestionsReq)
	if suggestionsRec.Code != http.StatusOK {
		t.Fatalf("expected steward suggestions HTTP 200, got %d: %s", suggestionsRec.Code, suggestionsRec.Body.String())
	}
	var suggestionsBody struct {
		Data []struct {
			ID   string `json:"id"`
			Risk string `json:"risk"`
		} `json:"data"`
	}
	if err := json.Unmarshal(suggestionsRec.Body.Bytes(), &suggestionsBody); err != nil {
		t.Fatalf("decode suggestions: %v", err)
	}
	if len(suggestionsBody.Data) == 0 || suggestionsBody.Data[0].Risk != "中风险" {
		t.Fatalf("unexpected suggestions: %#v", suggestionsBody.Data)
	}

	previewRec := httptest.NewRecorder()
	previewReq := httptest.NewRequest(http.MethodPost, "/api/v1/steward/suggestions/download-cleanup/preview", bytes.NewBufferString(`{}`))
	router.ServeHTTP(previewRec, previewReq)
	if previewRec.Code != http.StatusOK {
		t.Fatalf("expected steward preview HTTP 200, got %d: %s", previewRec.Code, previewRec.Body.String())
	}
	var previewBody struct {
		Data struct {
			ConfirmationID string `json:"confirmationId"`
		} `json:"data"`
	}
	if err := json.Unmarshal(previewRec.Body.Bytes(), &previewBody); err != nil {
		t.Fatalf("decode preview: %v", err)
	}
	if previewBody.Data.ConfirmationID == "" {
		t.Fatal("expected steward confirmation id")
	}

	confirmRec := httptest.NewRecorder()
	confirmReq := httptest.NewRequest(http.MethodPost, "/api/v1/steward/suggestions/download-cleanup/confirm", bytes.NewBufferString(`{"confirmationId":"`+previewBody.Data.ConfirmationID+`"}`))
	router.ServeHTTP(confirmRec, confirmReq)
	if confirmRec.Code != http.StatusOK {
		t.Fatalf("expected steward confirm HTTP 200, got %d: %s", confirmRec.Code, confirmRec.Body.String())
	}

	risksRec := httptest.NewRecorder()
	risksReq := httptest.NewRequest(http.MethodGet, "/api/v1/security/risk-actions", nil)
	router.ServeHTTP(risksRec, risksReq)
	if risksRec.Code != http.StatusOK {
		t.Fatalf("expected security risk HTTP 200, got %d: %s", risksRec.Code, risksRec.Body.String())
	}
	var risksBody struct {
		Data []struct {
			ID    string `json:"id"`
			Level string `json:"level"`
			State string `json:"state"`
		} `json:"data"`
	}
	if err := json.Unmarshal(risksRec.Body.Bytes(), &risksBody); err != nil {
		t.Fatalf("decode risks: %v", err)
	}
	if len(risksBody.Data) == 0 || risksBody.Data[0].Level != "中风险" || risksBody.Data[0].State != "待处理" {
		t.Fatalf("unexpected risks: %#v", risksBody.Data)
	}

	blockRec := httptest.NewRecorder()
	blockReq := httptest.NewRequest(http.MethodPost, "/api/v1/security/risk-actions/r2/block", bytes.NewBufferString(`{"actorId":"tester","reason":"too broad"}`))
	router.ServeHTTP(blockRec, blockReq)
	if blockRec.Code != http.StatusOK {
		t.Fatalf("expected risk block HTTP 200, got %d: %s", blockRec.Code, blockRec.Body.String())
	}

	sharesRec := httptest.NewRecorder()
	sharesReq := httptest.NewRequest(http.MethodGet, "/api/v1/shares", nil)
	router.ServeHTTP(sharesRec, sharesReq)
	if sharesRec.Code != http.StatusOK {
		t.Fatalf("expected shares HTTP 200, got %d: %s", sharesRec.Code, sharesRec.Body.String())
	}
	var sharesBody struct {
		Data []struct {
			ID     string `json:"id"`
			Risk   string `json:"risk"`
			Active bool   `json:"active"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sharesRec.Body.Bytes(), &sharesBody); err != nil {
		t.Fatalf("decode shares: %v", err)
	}
	if len(sharesBody.Data) == 0 || sharesBody.Data[0].Risk == "" || !sharesBody.Data[0].Active {
		t.Fatalf("unexpected shares: %#v", sharesBody.Data)
	}

	deleteRec := httptest.NewRecorder()
	deleteReq := httptest.NewRequest(http.MethodDelete, "/api/v1/shares/s2", nil)
	router.ServeHTTP(deleteRec, deleteReq)
	if deleteRec.Code != http.StatusOK {
		t.Fatalf("expected share delete HTTP 200, got %d: %s", deleteRec.Code, deleteRec.Body.String())
	}
}

func containsString(values []string, needle string) bool {
	for _, value := range values {
		if value == needle {
			return true
		}
	}
	return false
}

func TestSystemInfoReturnsDevstubStatus(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{
			AppName:     "HiGoOS",
			Environment: "test",
			Version:     "test-version",
		},
		Dev: devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/system/info", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	var body struct {
		OK   bool `json:"ok"`
		Data struct {
			AppName     string `json:"appName"`
			Environment string `json:"environment"`
			Version     string `json:"version"`
			Adapter     string `json:"adapter"`
			Status      string `json:"status"`
		} `json:"data"`
		RequestID string `json:"requestId"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode system info: %v", err)
	}
	if !body.OK {
		t.Fatal("expected ok envelope")
	}
	if body.Data.AppName != "HiGoOS" || body.Data.Version != "test-version" {
		t.Fatalf("unexpected system info: %#v", body.Data)
	}
	if body.Data.Adapter != "devstub" || body.Data.Status != "ready" {
		t.Fatalf("expected devstub ready status, got adapter=%q status=%q", body.Data.Adapter, body.Data.Status)
	}
	if body.RequestID == "" {
		t.Fatal("expected envelope request id")
	}
}

func TestSystemMaintenanceEndpoints(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{
			AppName:     "HiGoOS",
			Environment: "test",
			Version:     "test-version",
		},
		Dev: devstub.NewStore(),
	})

	updatesRec := httptest.NewRecorder()
	updatesReq := httptest.NewRequest(http.MethodGet, "/api/v1/system/updates", nil)
	router.ServeHTTP(updatesRec, updatesReq)
	if updatesRec.Code != http.StatusOK {
		t.Fatalf("expected updates HTTP 200, got %d: %s", updatesRec.Code, updatesRec.Body.String())
	}
	var updatesBody struct {
		Data struct {
			Current      string `json:"current"`
			UpdateStatus string `json:"updateStatus"`
		} `json:"data"`
	}
	if err := json.Unmarshal(updatesRec.Body.Bytes(), &updatesBody); err != nil {
		t.Fatalf("decode updates: %v", err)
	}
	if updatesBody.Data.Current != "test-version" || updatesBody.Data.UpdateStatus == "" {
		t.Fatalf("unexpected updates payload: %#v", updatesBody.Data)
	}

	checkRec := httptest.NewRecorder()
	checkReq := httptest.NewRequest(http.MethodPost, "/api/v1/system/updates/check", nil)
	router.ServeHTTP(checkRec, checkReq)
	if checkRec.Code != http.StatusOK {
		t.Fatalf("expected update check HTTP 200, got %d: %s", checkRec.Code, checkRec.Body.String())
	}

	eventsRec := httptest.NewRecorder()
	eventsReq := httptest.NewRequest(http.MethodGet, "/api/v1/events/stream", nil)
	eventsReq.Header.Set("Accept", "text/event-stream")
	router.ServeHTTP(eventsRec, eventsReq)
	if eventsRec.Code != http.StatusOK {
		t.Fatalf("expected events HTTP 200, got %d: %s", eventsRec.Code, eventsRec.Body.String())
	}
	if eventsRec.Header().Get("Content-Type") != "text/event-stream; charset=utf-8" {
		t.Fatalf("expected event stream content type, got %q", eventsRec.Header().Get("Content-Type"))
	}
}

func TestDesktopAppsReturnsSeedApps(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/apps", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusOK {
		t.Fatalf("expected HTTP 200, got %d", rec.Code)
	}

	var body struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID      string `json:"id"`
			Name    string `json:"name"`
			Icon    string `json:"icon"`
			Badge   int    `json:"badge,omitempty"`
			Utility bool   `json:"utility,omitempty"`
		} `json:"data"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode desktop apps: %v", err)
	}
	if !body.OK {
		t.Fatal("expected ok envelope")
	}
	if len(body.Data) != 14 {
		t.Fatalf("expected 14 seed apps, got %d", len(body.Data))
	}
	if body.Data[0].ID != "file-manager" || body.Data[0].Name != "文件管理" || body.Data[0].Badge != 2 {
		t.Fatalf("unexpected first app: %#v", body.Data[0])
	}
	if body.Data[13].ID != "remote-access" || body.Data[13].Name != "远程访问" {
		t.Fatalf("unexpected last app: %#v", body.Data[13])
	}
}

func TestDesktopBootstrapEndpointsReturnSeedState(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	windowsRec := httptest.NewRecorder()
	windowsReq := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/windows", nil)
	router.ServeHTTP(windowsRec, windowsReq)
	if windowsRec.Code != http.StatusOK {
		t.Fatalf("expected desktop windows HTTP 200, got %d", windowsRec.Code)
	}
	var windowsBody struct {
		OK   bool `json:"ok"`
		Data []struct {
			ID string `json:"id"`
		} `json:"data"`
	}
	if err := json.Unmarshal(windowsRec.Body.Bytes(), &windowsBody); err != nil {
		t.Fatalf("decode desktop windows: %v", err)
	}
	if !windowsBody.OK || len(windowsBody.Data) != 13 {
		t.Fatalf("expected 13 ok seed windows, got ok=%v len=%d", windowsBody.OK, len(windowsBody.Data))
	}

	sessionRec := httptest.NewRecorder()
	sessionReq := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/session", nil)
	router.ServeHTTP(sessionRec, sessionReq)
	if sessionRec.Code != http.StatusOK {
		t.Fatalf("expected desktop session HTTP 200, got %d", sessionRec.Code)
	}
	var sessionBody struct {
		OK   bool `json:"ok"`
		Data struct {
			OpenWindowIDs    []string `json:"openWindowIds"`
			PinnedDockAppIDs []string `json:"pinnedDockAppIds"`
			DockOrder        []string `json:"dockOrder"`
		} `json:"data"`
	}
	if err := json.Unmarshal(sessionRec.Body.Bytes(), &sessionBody); err != nil {
		t.Fatalf("decode desktop session: %v", err)
	}
	if !sessionBody.OK || len(sessionBody.Data.OpenWindowIDs) != 3 || len(sessionBody.Data.DockOrder) != 14 {
		t.Fatalf("unexpected desktop session: %#v", sessionBody.Data)
	}
	if len(sessionBody.Data.PinnedDockAppIDs) != 4 {
		t.Fatalf("expected 4 pinned dock apps, got %d", len(sessionBody.Data.PinnedDockAppIDs))
	}
}

func TestDesktopSessionPutPersistsMutableLayout(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	putRec := httptest.NewRecorder()
	putReq := httptest.NewRequest(http.MethodPut, "/api/v1/desktop/session", bytes.NewBufferString(`{
		"activeWindowId":"docker",
		"assistantVisible":true,
		"pinnedDockAppIds":["docker","file-manager"],
		"desktopIconPositions":{"docker":{"x":222,"y":144}},
		"windowGeometries":{"docker":{"x":260,"y":112,"width":760,"height":520}}
	}`))
	router.ServeHTTP(putRec, putReq)
	if putRec.Code != http.StatusOK {
		t.Fatalf("expected desktop session PUT HTTP 200, got %d: %s", putRec.Code, putRec.Body.String())
	}

	getRec := httptest.NewRecorder()
	getReq := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/session", nil)
	router.ServeHTTP(getRec, getReq)
	if getRec.Code != http.StatusOK {
		t.Fatalf("expected desktop session GET HTTP 200, got %d: %s", getRec.Code, getRec.Body.String())
	}
	var body struct {
		Data struct {
			ActiveWindowID       string   `json:"activeWindowId"`
			AssistantVisible     bool     `json:"assistantVisible"`
			PinnedDockAppIDs     []string `json:"pinnedDockAppIds"`
			DesktopIconPositions map[string]struct {
				X int `json:"x"`
				Y int `json:"y"`
			} `json:"desktopIconPositions"`
			WindowGeometries map[string]struct {
				X      int `json:"x"`
				Y      int `json:"y"`
				Width  int `json:"width"`
				Height int `json:"height"`
			} `json:"windowGeometries"`
		} `json:"data"`
	}
	if err := json.Unmarshal(getRec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode persisted desktop session: %v", err)
	}
	if body.Data.ActiveWindowID != "docker" || !body.Data.AssistantVisible {
		t.Fatalf("expected persisted active window and assistant state, got %#v", body.Data)
	}
	if len(body.Data.PinnedDockAppIDs) != 2 || body.Data.PinnedDockAppIDs[0] != "docker" {
		t.Fatalf("expected persisted pinned dock apps, got %#v", body.Data.PinnedDockAppIDs)
	}
	if body.Data.DesktopIconPositions["docker"].X != 222 || body.Data.WindowGeometries["docker"].Width != 760 {
		t.Fatalf("expected persisted desktop geometry, got icons=%#v windows=%#v", body.Data.DesktopIconPositions, body.Data.WindowGeometries)
	}
}

func TestDesktopSessionPutRejectsUnknownAppID(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPut, "/api/v1/desktop/session", bytes.NewBufferString(`{"activeWindowId":"missing-app"}`))
	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Fatalf("expected desktop session PUT HTTP 400, got %d: %s", rec.Code, rec.Body.String())
	}
}

func TestCORSPreflightUsesConfiguredPublicOrigin(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "test", Version: "test", PublicOrigin: "http://localhost:5173"},
		Dev:    devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodOptions, "/api/v1/desktop/apps", nil)
	req.Header.Set("Origin", "http://localhost:5173")

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusNoContent {
		t.Fatalf("expected preflight 204, got %d", rec.Code)
	}
	if rec.Header().Get("Access-Control-Allow-Origin") != "http://localhost:5173" {
		t.Fatalf("unexpected allow origin: %q", rec.Header().Get("Access-Control-Allow-Origin"))
	}
	if rec.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Fatal("expected credentials to be allowed")
	}
}

func TestSessionGuardRequiresCookieOutsideDev(t *testing.T) {
	router := httpapi.NewRouter(httpapi.Dependencies{
		Config: platform.Config{Environment: "prod", Version: "test"},
		Dev:    devstub.NewStore(),
	})

	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/api/v1/desktop/apps", nil)

	router.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Fatalf("expected HTTP 401 without session, got %d", rec.Code)
	}
	var body struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode unauthorized envelope: %v", err)
	}
	if body.OK {
		t.Fatal("expected error envelope ok=false")
	}
}
