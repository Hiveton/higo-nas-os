package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/agents"
	"higoos/server-go/internal/appcenter"
	"higoos/server-go/internal/assistant"
	"higoos/server-go/internal/backups"
	"higoos/server-go/internal/devstub"
	hdocker "higoos/server-go/internal/docker"
	"higoos/server-go/internal/downloads"
	"higoos/server-go/internal/files"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/monitoring"
	"higoos/server-go/internal/music"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/remote"
	"higoos/server-go/internal/security"
	"higoos/server-go/internal/settings"
	"higoos/server-go/internal/steward"
	"higoos/server-go/internal/storage"
	"higoos/server-go/internal/video"
)

type Dependencies struct {
	Config     platform.Config
	Dev        *devstub.Store
	Files      *files.Service
	Monitoring *monitoring.Service
	Settings   *settings.Store
	Storage    *storage.Service
	Downloads  *downloads.Service
	Docker     *hdocker.DevService
	Backups    *backups.Service
	AppCenter  *appcenter.Service
	Remote     *remote.Service
	Media      *media.Service
	Music      *music.Service
	Video      *video.Service
	Assistant  *assistant.Service
	Agents     *agents.Service
	Accounts   *accounts.Service
	Steward    *steward.Service
	Security   *security.Service
	Logger     *slog.Logger
}

func NewRouter(deps Dependencies) http.Handler {
	cfg := deps.Config.WithDefaults()
	logger := deps.Logger
	if logger == nil {
		logger = platform.NewLogger(cfg.Environment)
	}
	dev := deps.Dev
	if dev == nil {
		var err error
		dev, err = devstub.NewStoreWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("desktop state unavailable", slog.Any("error", err))
			dev = devstub.NewStore()
		}
	}
	fileService := deps.Files
	if fileService == nil {
		var (
			repo files.Repository
			err  error
		)
		if strings.TrimSpace(cfg.NASRoot) != "" {
			repo, err = files.NewRootRepository(cfg.NASRoot)
		} else {
			repo, err = files.NewFixtureRepositoryWithStateDir("", cfg.StateDir)
		}
		if err == nil {
			fileService, err = files.NewService(repo)
		}
		if err != nil {
			logger.Warn("files dev fixture unavailable", slog.Any("error", err))
		}
	}
	monitoringService := deps.Monitoring
	if monitoringService == nil {
		var err error
		monitoringService, err = monitoring.NewServiceWithStateDir(nil, cfg.StateDir)
		if err != nil {
			logger.Warn("monitoring state unavailable", slog.Any("error", err))
			monitoringService = monitoring.NewService(nil)
		}
	}
	settingsStore := deps.Settings
	if settingsStore == nil {
		var err error
		settingsStore, err = settings.NewStoreWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("settings state unavailable", slog.Any("error", err))
			settingsStore = settings.NewStore()
		}
	}
	storageService := deps.Storage
	if storageService == nil {
		var err error
		storageService, err = storage.NewServiceWithStateDir(nil, cfg.StateDir)
		if err != nil {
			logger.Warn("storage state unavailable", slog.Any("error", err))
			storageService = storage.NewService(nil)
		}
	}
	downloadsService := deps.Downloads
	if downloadsService == nil {
		var err error
		downloadsService, err = downloads.NewServiceWithStateDirAndDownloadDir(cfg.StateDir, downloadRoot(cfg))
		if err != nil {
			logger.Warn("downloads state unavailable", slog.Any("error", err))
			downloadsService = downloads.NewService()
		}
	}
	dockerService := deps.Docker
	if dockerService == nil {
		var err error
		dockerService, err = hdocker.NewDevServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("docker state unavailable", slog.Any("error", err))
			dockerService = hdocker.NewDevService()
		}
	}
	backupService := deps.Backups
	if backupService == nil {
		var err error
		backupService, err = backups.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("backup state unavailable", slog.Any("error", err))
			backupService = backups.NewService()
		}
	}
	appCenterService := deps.AppCenter
	if appCenterService == nil {
		var err error
		appCenterService, err = appcenter.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("app center state unavailable", slog.Any("error", err))
			appCenterService = appcenter.NewService()
		}
	}
	remoteService := deps.Remote
	if remoteService == nil {
		var err error
		remoteService, err = remote.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("remote state unavailable", slog.Any("error", err))
			remoteService = remote.NewService()
		}
	}
	mediaService := deps.Media
	if mediaService == nil {
		var err error
		mediaService, err = media.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("media state unavailable", slog.Any("error", err))
			mediaService = media.NewService()
		}
	}
	musicService := deps.Music
	if musicService == nil {
		var err error
		musicService, err = music.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("music state unavailable", slog.Any("error", err))
			musicService = music.NewService()
		}
	}
	videoService := deps.Video
	if videoService == nil {
		var err error
		videoService, err = video.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("video state unavailable", slog.Any("error", err))
			videoService = video.NewService()
		}
	}
	videoService.StartDVR(context.Background(), 20*time.Second)
	assistantService := deps.Assistant
	if assistantService == nil {
		var err error
		assistantService, err = assistant.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("assistant state unavailable", slog.Any("error", err))
			assistantService = assistant.NewService()
		}
	}
	agentsService := deps.Agents
	if agentsService == nil {
		var err error
		agentsService, err = agents.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("agents state unavailable", slog.Any("error", err))
			agentsService = agents.NewService()
		}
	}
	accountsService := deps.Accounts
	if accountsService == nil {
		var err error
		accountsService, err = accounts.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("accounts state unavailable", slog.Any("error", err))
			accountsService = accounts.NewService()
		}
	}
	stewardService := deps.Steward
	if stewardService == nil {
		var err error
		stewardService, err = steward.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("steward state unavailable", slog.Any("error", err))
			stewardService = steward.NewService()
		}
	}
	securityService := deps.Security
	if securityService == nil {
		var err error
		securityService, err = security.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("security state unavailable", slog.Any("error", err))
			securityService = security.NewService()
		}
	}

	api := &API{
		config:     cfg,
		dev:        dev,
		files:      fileService,
		monitoring: monitoringService,
		settings:   settingsStore,
		storage:    storageService,
		downloads:  downloadsService,
		docker:     dockerService,
		backups:    backupService,
		appCenter:  appCenterService,
		remote:     remoteService,
		media:      mediaService,
		music:      musicService,
		video:      videoService,
		assistant:  assistantService,
		agents:     agentsService,
		accounts:   accountsService,
		steward:    stewardService,
		security:   securityService,
		staticDir:  strings.TrimSpace(cfg.StaticDir),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", api.healthz)
	mux.HandleFunc("/readyz", api.readyz)
	mux.HandleFunc("/api/v1/system/info", api.systemInfo)
	mux.HandleFunc("/api/v1/system/updates", api.systemUpdates)
	mux.HandleFunc("/api/v1/system/updates/check", api.systemUpdateCheck)
	mux.HandleFunc("/api/v1/system/backups", api.systemBackups)
	mux.HandleFunc("/api/v1/events/stream", api.eventsStream)
	mux.HandleFunc("/api/v1/desktop/apps", api.desktopApps)
	mux.HandleFunc("/api/v1/desktop/windows", api.desktopWindows)
	mux.HandleFunc("/api/v1/desktop/session", api.desktopSession)
	mux.HandleFunc("/api/v1/files/tree", api.filesTree)
	mux.HandleFunc("/api/v1/files/search", api.filesSearch)
	mux.HandleFunc("/api/v1/files/folders", api.filesFolders)
	mux.HandleFunc("/api/v1/files/upload", api.filesUpload)
	mux.HandleFunc("/api/v1/files/batch/move", api.filesBatchMove)
	mux.HandleFunc("/api/v1/files/batch/rename", api.filesBatchRename)
	mux.HandleFunc("/api/v1/files/batch/delete", api.filesBatchDelete)
	mux.HandleFunc("/api/v1/files/", api.fileByID)
	mux.HandleFunc("/api/v1/monitoring/metrics/current", api.monitoringCurrentMetrics)
	mux.HandleFunc("/api/v1/monitoring/metrics/snapshot", api.monitoringMetricsSnapshot)
	mux.HandleFunc("/api/v1/monitoring/metrics/trend", api.monitoringMetricTrend)
	mux.HandleFunc("/api/v1/monitoring/logs", api.monitoringLogs)
	mux.HandleFunc("/api/v1/monitoring/alerts", api.monitoringAlerts)
	mux.HandleFunc("/api/v1/monitoring/alerts/", api.monitoringAlertByID)
	mux.HandleFunc("/api/v1/monitoring/diagnostics", api.monitoringDiagnostics)
	mux.HandleFunc("/api/v1/settings", api.settingsRoot)
	mux.HandleFunc("/api/v1/settings/defaults", api.settingsDefaults)
	mux.HandleFunc("/api/v1/storage/pools", api.storagePools)
	mux.HandleFunc("/api/v1/storage/spaces", api.storageSpaces)
	mux.HandleFunc("/api/v1/storage/spaces/", api.storageSpaceByID)
	mux.HandleFunc("/api/v1/storage/disks", api.storageDisks)
	mux.HandleFunc("/api/v1/storage/disks/", api.storageDiskByID)
	mux.HandleFunc("/api/v1/storage/smart", api.storageSmartReports)
	mux.HandleFunc("/api/v1/storage/tasks/smart-scan", api.storageSmartScan)
	mux.HandleFunc("/api/v1/storage/tasks/repair", api.storageRepair)
	mux.HandleFunc("/api/v1/storage/tasks/snapshot", api.storageSnapshot)
	mux.HandleFunc("/api/v1/storage/tasks/", api.storageTaskByID)
	mux.HandleFunc("/api/v1/downloads/tasks", api.downloadTasks)
	mux.HandleFunc("/api/v1/downloads/tasks/", api.downloadTaskByID)
	mux.HandleFunc("/api/v1/downloads/speed-profiles", api.downloadSpeedProfiles)
	mux.HandleFunc("/api/v1/downloads/speed-profile", api.downloadSpeedProfile)
	mux.HandleFunc("/api/v1/docker/stacks", api.dockerStacks)
	mux.HandleFunc("/api/v1/docker/images/search", api.dockerImageSearch)
	mux.HandleFunc("/api/v1/docker/images/pull", api.dockerImagePull)
	mux.HandleFunc("/api/v1/docker/images/pulls", api.dockerImagePulls)
	mux.HandleFunc("/api/v1/docker/images/remove", api.dockerImageRemove)
	mux.HandleFunc("/api/v1/docker/images", api.dockerImages)
	mux.HandleFunc("/api/v1/docker/volumes/remove", api.dockerVolumeRemove)
	mux.HandleFunc("/api/v1/docker/volumes", api.dockerVolumes)
	mux.HandleFunc("/api/v1/docker/networks/remove", api.dockerNetworkRemove)
	mux.HandleFunc("/api/v1/docker/networks/connect", api.dockerNetworkConnect)
	mux.HandleFunc("/api/v1/docker/networks/disconnect", api.dockerNetworkDisconnect)
	mux.HandleFunc("/api/v1/docker/networks", api.dockerNetworks)
	mux.HandleFunc("/api/v1/docker/containers", api.dockerContainers)
	mux.HandleFunc("/api/v1/docker/containers/", api.dockerContainerByID)
	mux.HandleFunc("/api/v1/backups/jobs", api.backupJobs)
	mux.HandleFunc("/api/v1/backups/jobs/", api.backupJobByID)
	mux.HandleFunc("/api/v1/app-center/apps", api.appCenterApps)
	mux.HandleFunc("/api/v1/app-center/apps/", api.appCenterAppByID)
	mux.HandleFunc("/api/v1/remote/status", api.remoteStatus)
	mux.HandleFunc("/api/v1/remote/channel/start", api.remoteStartChannel)
	mux.HandleFunc("/api/v1/remote/channel/stop", api.remoteStopChannel)
	mux.HandleFunc("/api/v1/remote/tunnel-mode", api.remoteTunnelMode)
	mux.HandleFunc("/api/v1/remote/mfa", api.remoteMFA)
	mux.HandleFunc("/api/v1/remote/policy", api.remotePolicy)
	mux.HandleFunc("/api/v1/remote/domain-token", api.remoteDomainToken)
	mux.HandleFunc("/api/v1/remote/domain-token/rotate", api.remoteRotateDomainToken)
	mux.HandleFunc("/api/v1/remote/devices", api.remoteDevices)
	mux.HandleFunc("/api/v1/remote/devices/", api.remoteDeviceByID)
	mux.HandleFunc("/api/v1/remote/login-alerts", api.remoteLoginAlerts)
	mux.HandleFunc("/api/v1/remote/share-scan", api.remoteShareScan)
	mux.HandleFunc("/api/v1/media/items", api.mediaItems)
	mux.HandleFunc("/api/v1/media/albums", api.mediaAlbums)
	mux.HandleFunc("/api/v1/media/memories", api.mediaMemories)
	mux.HandleFunc("/api/v1/media/people/merge", api.mediaMergePeople)
	mux.HandleFunc("/api/v1/media/subtitles/jobs", api.mediaSubtitleJobs)
	mux.HandleFunc("/api/v1/media/transcode/jobs", api.mediaTranscodeJobs)
	mux.HandleFunc("/api/v1/media/shares", api.mediaShares)
	mux.HandleFunc("/api/v1/music/library", api.musicLibrary)
	mux.HandleFunc("/api/v1/music/scan", api.musicScan)
	mux.HandleFunc("/api/v1/music/tracks", api.musicTracks)
	mux.HandleFunc("/api/v1/music/albums", api.musicAlbums)
	mux.HandleFunc("/api/v1/music/tracks/", api.musicTrackByID)
	mux.HandleFunc("/api/v1/videos/library", api.videoLibrary)
	mux.HandleFunc("/api/v1/videos/library/", api.videoLibraryByID)
	mux.HandleFunc("/api/v1/videos/scan", api.videoScan)
	mux.HandleFunc("/api/v1/videos/items", api.videoItems)
	mux.HandleFunc("/api/v1/videos/items/", api.videoItemByID)
	mux.HandleFunc("/api/v1/videos/tasks", api.videoTasks)
	mux.HandleFunc("/api/v1/videos/tasks/scrape", api.videoScrapeTask)
	mux.HandleFunc("/api/v1/videos/tasks/subtitle", api.videoSubtitleTask)
	mux.HandleFunc("/api/v1/videos/tasks/transcode", api.videoTranscodeTask)
	mux.HandleFunc("/api/v1/videos/live/sources", api.videoLiveSources)
	mux.HandleFunc("/api/v1/videos/live/channels", api.videoLiveChannels)
	mux.HandleFunc("/api/v1/videos/live/guide-sources", api.videoLiveGuideSources)
	mux.HandleFunc("/api/v1/videos/live/programs", api.videoLivePrograms)
	mux.HandleFunc("/api/v1/videos/live/dvr/settings", api.videoLiveDVRSettings)
	mux.HandleFunc("/api/v1/videos/live/recording-timers", api.videoLiveRecordingTimers)
	mux.HandleFunc("/api/v1/videos/live/recording-timers/", api.videoLiveRecordingTimerByID)
	mux.HandleFunc("/api/v1/videos/live/recordings", api.videoLiveRecordings)
	mux.HandleFunc("/api/v1/search/semantic", api.assistantSemanticSearch)
	mux.HandleFunc("/api/v1/assistant/threads", api.assistantThreads)
	mux.HandleFunc("/api/v1/assistant/threads/", api.assistantThreadByID)
	mux.HandleFunc("/api/v1/assistant/actions/", api.assistantActionByID)
	mux.HandleFunc("/api/v1/agents/templates", api.agentTemplates)
	mux.HandleFunc("/api/v1/agents", api.agentsRoot)
	mux.HandleFunc("/api/v1/agents/", api.agentByID)
	mux.HandleFunc("/api/v1/accounts/summary", api.accountsSummary)
	mux.HandleFunc("/api/v1/accounts/users", api.accountUsers)
	mux.HandleFunc("/api/v1/accounts/users/", api.accountUserByID)
	mux.HandleFunc("/api/v1/accounts/groups", api.accountGroups)
	mux.HandleFunc("/api/v1/accounts/groups/", api.accountGroupByID)
	mux.HandleFunc("/api/v1/accounts/grants", api.accountGrants)
	mux.HandleFunc("/api/v1/accounts/grants/", api.accountGrantByID)
	mux.HandleFunc("/api/v1/workflows/preview", api.workflowPreview)
	mux.HandleFunc("/api/v1/workflows/runs", api.workflowRuns)
	mux.HandleFunc("/api/v1/workflows/runs/", api.workflowRunByID)
	mux.HandleFunc("/api/v1/steward/suggestions", api.stewardSuggestions)
	mux.HandleFunc("/api/v1/steward/suggestions/", api.stewardSuggestionByID)
	mux.HandleFunc("/api/v1/steward/audit", api.stewardAudit)
	mux.HandleFunc("/api/v1/steward/audit/", api.stewardAuditByID)
	mux.HandleFunc("/api/v1/security/identities", api.securityIdentities)
	mux.HandleFunc("/api/v1/security/identities/", api.securityIdentityByID)
	mux.HandleFunc("/api/v1/security/ai-policies", api.securityAIPolicies)
	mux.HandleFunc("/api/v1/security/ai-policies/", api.securityAIPolicyByID)
	mux.HandleFunc("/api/v1/security/risk-actions", api.securityRiskActions)
	mux.HandleFunc("/api/v1/security/risk-actions/", api.securityRiskActionByID)
	mux.HandleFunc("/api/v1/security/audit", api.securityAudit)
	mux.HandleFunc("/api/v1/security/audit/", api.securityAuditByID)
	mux.HandleFunc("/api/v1/shares", api.securityShares)
	mux.HandleFunc("/api/v1/shares/", api.securityShareByID)
	mux.HandleFunc("/", api.staticAssets)

	return chain(
		mux,
		platform.RequestIDMiddleware,
		recoverPanic(logger),
		secureHeaders,
		cors(cfg.PublicOrigin),
		sessionGuard(cfg),
		accessLog(logger),
	)
}

func downloadRoot(cfg platform.Config) string {
	if strings.TrimSpace(cfg.NASRoot) != "" {
		return filepath.Join(cfg.NASRoot, "downloads")
	}
	if strings.TrimSpace(cfg.StateDir) != "" {
		return filepath.Join(cfg.StateDir, "downloads")
	}
	return ""
}

type API struct {
	config     platform.Config
	dev        *devstub.Store
	files      *files.Service
	monitoring *monitoring.Service
	settings   *settings.Store
	storage    *storage.Service
	downloads  *downloads.Service
	docker     *hdocker.DevService
	backups    *backups.Service
	appCenter  *appcenter.Service
	remote     *remote.Service
	media      *media.Service
	music      *music.Service
	video      *video.Service
	assistant  *assistant.Service
	agents     *agents.Service
	accounts   *accounts.Service
	steward    *steward.Service
	security   *security.Service
	staticDir  string
}

func (a *API) healthz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	writeBareJSON(w, http.StatusOK, map[string]string{
		"status": "ok",
	})
}

func (a *API) readyz(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	if !a.config.Ready {
		writeBareJSON(w, http.StatusServiceUnavailable, map[string]string{
			"status": "not_ready",
		})
		return
	}
	writeBareJSON(w, http.StatusOK, map[string]string{
		"status": "ready",
	})
}

func (a *API) systemInfo(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.dev.SystemInfo(a.config.AppName, a.config.Environment, a.config.Version))
}

func (a *API) desktopApps(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.dev.Apps())
}

func (a *API) desktopWindows(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet) {
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.dev.Windows())
}

func (a *API) desktopSession(w http.ResponseWriter, r *http.Request) {
	if !allowMethod(w, r, http.MethodGet, http.MethodPut) {
		return
	}
	if r.Method == http.MethodPut {
		var patch devstub.DesktopSessionPatch
		if err := decodeJSON(r, &patch); err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "invalid_json", err.Error())
			return
		}
		session, err := a.dev.UpdateDesktopSession(patch)
		if err != nil {
			platform.WriteError(w, r, http.StatusBadRequest, "desktop_session_invalid", err.Error())
			return
		}
		platform.WriteJSON(w, r, http.StatusOK, session)
		return
	}
	platform.WriteJSON(w, r, http.StatusOK, a.dev.DesktopSession())
}

func allowMethod(w http.ResponseWriter, r *http.Request, methods ...string) bool {
	for _, method := range methods {
		if r.Method == method {
			return true
		}
	}
	w.Header().Set("Allow", strings.Join(methods, ", "))
	platform.WriteError(w, r, http.StatusMethodNotAllowed, "method_not_allowed", "method not allowed")
	return false
}

func writeBareJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = jsonEncoder(w).Encode(body)
}
