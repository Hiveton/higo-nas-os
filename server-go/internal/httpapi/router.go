package httpapi

import (
	"context"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	mcpsdk "github.com/modelcontextprotocol/go-sdk/mcp"

	"higoos/server-go/internal/accounts"
	"higoos/server-go/internal/activity"
	"higoos/server-go/internal/aianalysis"
	"higoos/server-go/internal/apiclient"
	"higoos/server-go/internal/appcenter"
	"higoos/server-go/internal/assistant"
	"higoos/server-go/internal/audit"
	"higoos/server-go/internal/auth"
	"higoos/server-go/internal/backups"
	"higoos/server-go/internal/db"
	"higoos/server-go/internal/devstub"
	"higoos/server-go/internal/discovery"
	hdocker "higoos/server-go/internal/docker"
	"higoos/server-go/internal/downloads"
	"higoos/server-go/internal/files"
	"higoos/server-go/internal/foldersync"
	"higoos/server-go/internal/hardware"
	"higoos/server-go/internal/identity"
	"higoos/server-go/internal/index"
	"higoos/server-go/internal/iscsi"
	"higoos/server-go/internal/llm"
	higomcp "higoos/server-go/internal/mcp"
	"higoos/server-go/internal/mcpclient"
	"higoos/server-go/internal/media"
	"higoos/server-go/internal/monitoring"
	"higoos/server-go/internal/music"
	"higoos/server-go/internal/network"
	"higoos/server-go/internal/platform"
	"higoos/server-go/internal/protocols"
	"higoos/server-go/internal/remote"
	"higoos/server-go/internal/search"
	"higoos/server-go/internal/security"
	"higoos/server-go/internal/settings"
	"higoos/server-go/internal/sharedfolders"
	"higoos/server-go/internal/steward"
	"higoos/server-go/internal/storage"
	"higoos/server-go/internal/tasks"
	"higoos/server-go/internal/video"
	"higoos/server-go/internal/vm"
)

type Dependencies struct {
	Config        platform.Config
	DB            *db.Pool
	Dev           *devstub.Store
	Files         *files.Service
	Monitoring    *monitoring.Service
	Hardware      *hardware.Service
	Settings      *settings.Store
	LLM           *llm.Store
	Storage       *storage.Service
	Downloads     *downloads.Service
	Docker        *hdocker.DevService
	Backups       *backups.Service
	Sync          *foldersync.Service
	AppCenter     *appcenter.Service
	Remote        *remote.Service
	Media         *media.Service
	Music         *music.Service
	Video         *video.Service
	VM            *vm.Service
	ISCSI         *iscsi.Service
	Assistant     *assistant.Service
	Accounts      *accounts.Service
	Auth          *auth.SessionStore
	Audit         *audit.Store
	Steward       *steward.Service
	Security      *security.Service
	Activity      *activity.Service
	Protocols     *protocols.Service
	SharedFolders *sharedfolders.Service
	Network       *network.Service
	Identity      *identity.Provider
	Tasks         *tasks.Manager
	Logger        *slog.Logger
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
	hardwareService := deps.Hardware
	if hardwareService == nil {
		hardwareService = hardware.NewService(nil)
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
	syncService := deps.Sync
	if syncService == nil {
		var err error
		syncService, err = foldersync.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("sync state unavailable", slog.Any("error", err))
			syncService = foldersync.NewService()
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
	if cfg.RemoteProbeAddr != "" {
		remoteService.SetProbeTarget(cfg.RemoteProbeAddr)
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
	vmService := deps.VM
	if vmService == nil {
		var err error
		vmService, err = vm.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("vm state unavailable", slog.Any("error", err))
			vmService = vm.NewService()
		}
	}
	iscsiService := deps.ISCSI
	if iscsiService == nil {
		var err error
		iscsiService, err = iscsi.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("iscsi state unavailable", slog.Any("error", err))
			iscsiService = iscsi.NewService()
		}
	}
	assistantService := deps.Assistant
	if assistantService == nil {
		var err error
		assistantService, err = assistant.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("assistant state unavailable", slog.Any("error", err))
			assistantService = assistant.NewService()
		}
	}
	llmStore := deps.LLM
	if llmStore == nil {
		var err error
		llmStore, err = llm.NewStoreWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("llm provider state unavailable", slog.Any("error", err))
			llmStore = llm.NewStore()
		}
	}
	// Seed a default provider on a fresh install so AI features work out of the
	// box (no-op once any provider exists or when no default is configured).
	if seeded, err := llmStore.SeedDefaultsIfEmpty(llm.SeedConfig{
		BaseURL:     cfg.LLMDefaultBaseURL,
		APIKey:      cfg.LLMDefaultAPIKey,
		ChatModel:   cfg.LLMDefaultChatModel,
		VisionModel: cfg.LLMDefaultVisionModel,
		EmbedModel:  cfg.LLMDefaultEmbedModel,
		ASRModel:    cfg.LLMDefaultASRModel,
	}); err != nil {
		logger.Warn("llm default provider seeding failed", slog.Any("error", err))
	} else if seeded > 0 {
		logger.Info("seeded default llm providers", slog.Int("count", seeded))
	}
	assistantService.WithLLM(llmStore, llm.NewClient)

	accountsService := deps.Accounts
	if accountsService == nil {
		var err error
		accountsService, err = accounts.NewServiceWithConfig(accounts.Config{
			StateDir:   cfg.StateDir,
			Backend:    cfg.AccountsBackend,
			UIDBase:    cfg.AccountsUIDBase,
			Group:      cfg.AccountsGroup,
			AdminGroup: cfg.AccountsAdminGroup,
			NASRoot:    cfg.NASRoot,
		})
		if err != nil {
			logger.Warn("accounts state unavailable", slog.Any("error", err))
			accountsService = accounts.NewService()
		}
	}
	// Ensure the seed admin has a credential on first boot. The generated
	// password is logged once; operators must change it after first login.
	if pw, err := accountsService.BootstrapAdmin(context.Background(), cfg.AdminBootstrapPassword); err != nil {
		logger.Warn("admin bootstrap failed", slog.Any("error", err))
	} else if pw != "" {
		logger.Warn("initial admin password generated — change it after first login",
			slog.String("username", "admin"), slog.String("password", pw))
	}
	authStore := deps.Auth
	if authStore == nil {
		var err error
		authStore, err = auth.NewSessionStoreWithStateDir(cfg.StateDir, cfg.SessionTTL)
		if err != nil {
			logger.Warn("session state unavailable", slog.Any("error", err))
			authStore = auth.NewSessionStore(cfg.SessionTTL)
		}
	}
	auditStore := deps.Audit
	if auditStore == nil {
		auditStore = audit.NewStore()
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
	activityService := deps.Activity
	if activityService == nil {
		var err error
		activityService, err = activity.NewServiceWithStateDir(cfg.StateDir)
		if err != nil {
			logger.Warn("activity state unavailable", slog.Any("error", err))
			activityService = activity.NewService()
		}
	}
	protocolsService := deps.Protocols
	if protocolsService == nil {
		var err error
		protocolsService, err = protocols.NewServiceWithStateDir(nil, cfg.StateDir)
		if err != nil {
			logger.Warn("protocols state unavailable", slog.Any("error", err))
			protocolsService = protocols.NewService(nil)
		}
	}
	sharedFoldersService := deps.SharedFolders
	if sharedFoldersService == nil {
		var err error
		sharedFoldersService, err = sharedfolders.NewServiceWithStateDir(sharedfolders.Deps{
			Accounts:   accountsService,
			Storage:    storageService,
			Protocols:  protocolsService,
			NASRoot:    cfg.NASRoot,
			ServerHost: serverHostFor(cfg),
		}, cfg.StateDir)
		if err != nil {
			logger.Warn("shared folders state unavailable", slog.Any("error", err))
			sharedFoldersService = sharedfolders.NewService(sharedfolders.Deps{
				Accounts: accountsService, Storage: storageService, Protocols: protocolsService,
				NASRoot: cfg.NASRoot, ServerHost: serverHostFor(cfg),
			})
		}
	}
	networkService := deps.Network
	if networkService == nil {
		var err error
		networkService, err = network.NewServiceWithStateDir(nil, cfg.StateDir)
		if err != nil {
			logger.Warn("network state unavailable", slog.Any("error", err))
			networkService = network.NewService(nil)
		}
	}
	identityProvider := deps.Identity
	if identityProvider == nil {
		var err error
		identityProvider, err = identity.NewProvider(cfg.StateDir, cfg.AppName, cfg.Version, cfg.HTTPAddr)
		if err != nil {
			logger.Warn("identity provider unavailable", slog.Any("error", err))
			identityProvider, _ = identity.NewProvider("", cfg.AppName, cfg.Version, cfg.HTTPAddr)
		}
	}
	// LAN discovery responder so the desktop assistant can find a fresh NAS.
	if cfg.DiscoveryEnabled {
		go func() {
			_ = discovery.Run(context.Background(), cfg.DiscoveryAddr, identityProvider, logger)
		}()
	}

	taskManager := deps.Tasks
	if taskManager == nil {
		var err error
		taskManager, err = tasks.NewManager(taskStatePath(cfg.StateDir), tasks.WithLogger(logger))
		if err != nil {
			logger.Warn("task runtime state unavailable", slog.Any("error", err))
			taskManager, _ = tasks.NewManager("", tasks.WithLogger(logger))
		}
	}
	// Register domain task handlers before the pool starts.
	storageService.AttachTaskRunner(taskManager)
	storageService.StartSnapshotScheduler(context.Background(), 5*time.Minute)
	mediaService.AttachTaskRunner(taskManager)
	backupService.AttachTaskRunner(taskManager)
	backupService.StartBackupScheduler(context.Background(), 5*time.Minute)
	syncService.AttachTaskRunner(taskManager)
	videoService.AttachTaskRunner(taskManager)
	dockerService.AttachTaskRunner(taskManager)
	downloadsService.AttachTaskRunner(taskManager)
	appCenterService.AttachDocker(dockerService)
	appCenterService.AttachTaskRunner(taskManager)
	if fileService != nil {
		fileService.AttachTaskRunner(taskManager)
		stewardService.AttachFiles(fileService)
	}

	// Global background AI analysis engine: ledger-backed, resumable, runs in
	// this process and drives item analysis through the shared task runtime.
	analysisEngine, err := aianalysis.New(aianalysis.Deps{
		StateDir:        cfg.StateDir,
		Logger:          logger,
		Settings:        settingsStore,
		LLM:             llmStore,
		Index:           index.New(deps.DB, llmStore),
		Media:           mediaService,
		Files:           fileService,
		Video:           videoService,
		FaceEmbedderURL: cfg.FaceEmbedderURL,
		FaceTrainerURL:  cfg.FaceTrainerURL,
	})
	if err != nil {
		logger.Warn("ai analysis engine unavailable", slog.Any("error", err))
		analysisEngine = nil
	}
	if analysisEngine != nil {
		analysisEngine.AttachTaskRunner(taskManager)
		// Near-real-time trigger: a mutation in files/video promptly (debounced)
		// re-enumerates that domain instead of waiting for the periodic ticker.
		if fileService != nil {
			fileService.SetChangeNotifier(func() { analysisEngine.Notify(aianalysis.DomainFile) })
		}
		if videoService != nil {
			videoService.SetChangeNotifier(func() { analysisEngine.Notify(aianalysis.DomainVideo) })
		}
	}

	taskManager.Start(context.Background())
	if analysisEngine != nil {
		analysisEngine.Start(context.Background())
	}

	api := &API{
		config:        cfg,
		dev:           dev,
		files:         fileService,
		monitoring:    monitoringService,
		hardware:      hardwareService,
		settings:      settingsStore,
		llm:           llmStore,
		storage:       storageService,
		downloads:     downloadsService,
		docker:        dockerService,
		backups:       backupService,
		sync:          syncService,
		appCenter:     appCenterService,
		remote:        remoteService,
		media:         mediaService,
		music:         musicService,
		video:         videoService,
		vm:            vmService,
		iscsi:         iscsiService,
		assistant:     assistantService,
		accounts:      accountsService,
		auth:          authStore,
		audit:         auditStore,
		loginThrottle: newLoginThrottle(),
		steward:       stewardService,
		security:      securityService,
		activity:      activityService,
		protocols:     protocolsService,
		sharedFolders: sharedFoldersService,
		network:       networkService,
		identity:      identityProvider,
		tasks:         taskManager,
		index:         index.New(deps.DB, llmStore),
		search:        search.New(deps.DB, llmStore),
		aianalysis:    analysisEngine,
		staticDir:     strings.TrimSpace(cfg.StaticDir),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", api.healthz)
	mux.HandleFunc("/readyz", api.readyz)
	mux.HandleFunc("/api/v1/system/info", api.systemInfo)
	mux.HandleFunc("/api/v1/system/identity", api.systemIdentity)
	mux.HandleFunc("/api/v1/system/updates", api.systemUpdates)
	mux.HandleFunc("/api/v1/system/updates/check", api.systemUpdateCheck)
	mux.HandleFunc("/api/v1/system/backups", api.systemBackups)
	mux.HandleFunc("/api/v1/events/stream", api.eventsStream)
	mux.HandleFunc("/api/v1/desktop/apps", api.desktopApps)
	mux.HandleFunc("/api/v1/desktop/windows", api.desktopWindows)
	mux.HandleFunc("/api/v1/desktop/session", api.desktopSession)
	mux.HandleFunc("/api/v1/files/tree", api.filesTree)
	mux.HandleFunc("/api/v1/files/trash", api.filesTrash)
	mux.HandleFunc("/api/v1/files/search", api.filesSearch)
	mux.HandleFunc("/api/v1/files/folders", api.filesFolders)
	mux.HandleFunc("/api/v1/files/upload", api.filesUpload)
	mux.HandleFunc("/api/v1/files/batch/move", api.filesBatchMove)
	mux.HandleFunc("/api/v1/files/batch/rename", api.filesBatchRename)
	mux.HandleFunc("/api/v1/files/batch/delete", api.filesBatchDelete)
	mux.HandleFunc("/api/v1/files/batch/execute", api.filesBatchExecute)
	mux.HandleFunc("/api/v1/files/", api.fileByID)
	mux.HandleFunc("/api/v1/monitoring/metrics/current", api.monitoringCurrentMetrics)
	mux.HandleFunc("/api/v1/monitoring/metrics/snapshot", api.monitoringMetricsSnapshot)
	mux.HandleFunc("/api/v1/monitoring/metrics/trend", api.monitoringMetricTrend)
	mux.HandleFunc("/api/v1/monitoring/logs", api.monitoringLogs)
	mux.HandleFunc("/api/v1/monitoring/alerts", api.monitoringAlerts)
	mux.HandleFunc("/api/v1/monitoring/alerts/", api.monitoringAlertByID)
	mux.HandleFunc("/api/v1/monitoring/diagnostics", api.monitoringDiagnostics)
	mux.HandleFunc("/api/v1/hardware/inventory", api.hardwareInventory)
	mux.HandleFunc("/api/v1/settings", api.settingsRoot)
	mux.HandleFunc("/api/v1/settings/defaults", api.settingsDefaults)
	mux.HandleFunc("/api/v1/storage/pools", api.storagePools)
	mux.HandleFunc("/api/v1/storage/spaces", api.storageSpaces)
	mux.HandleFunc("/api/v1/storage/default-space", api.storageDefaultSpace)
	mux.HandleFunc("/api/v1/storage/spaces/", api.storageSpaceByID)
	mux.HandleFunc("/api/v1/storage/disks", api.storageDisks)
	mux.HandleFunc("/api/v1/storage/disks/", api.storageDiskByID)
	mux.HandleFunc("/api/v1/storage/smart", api.storageSmartReports)
	mux.HandleFunc("/api/v1/storage/tasks/smart-scan", api.storageSmartScan)
	mux.HandleFunc("/api/v1/storage/tasks/repair", api.storageRepair)
	mux.HandleFunc("/api/v1/storage/tasks/snapshot", api.storageSnapshot)
	mux.HandleFunc("/api/v1/storage/snapshots/rollback", api.storageSnapshotRollback)
	mux.HandleFunc("/api/v1/storage/tasks/", api.storageTaskByID)
	mux.HandleFunc("/api/v1/tasks", api.tasksList)
	mux.HandleFunc("/api/v1/tasks/stream", api.tasksStream)
	mux.HandleFunc("/api/v1/tasks/", api.taskByID)
	mux.HandleFunc("/api/v1/activity", api.activityLog)
	mux.HandleFunc("/api/v1/downloads/tasks", api.downloadTasks)
	mux.HandleFunc("/api/v1/downloads/tasks/", api.downloadTaskByID)
	mux.HandleFunc("/api/v1/downloads/speed-profiles", api.downloadSpeedProfiles)
	mux.HandleFunc("/api/v1/downloads/speed-profile", api.downloadSpeedProfile)
	mux.HandleFunc("/api/v1/downloads/queue-config", api.downloadQueueConfig)
	mux.HandleFunc("/api/v1/docker/stacks", api.dockerStacks)
	mux.HandleFunc("/api/v1/docker/stacks/", api.dockerStackByName)
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
	mux.HandleFunc("/api/v1/vm/machines", api.vmMachines)
	mux.HandleFunc("/api/v1/vm/machines/", api.vmMachineByID)
	mux.HandleFunc("/api/v1/vm/capabilities", api.vmCapabilities)
	mux.HandleFunc("/api/v1/vm/audit", api.vmAudit)
	mux.HandleFunc("/api/v1/iscsi/targets", api.iscsiTargets)
	mux.HandleFunc("/api/v1/iscsi/targets/", api.iscsiTargetByID)
	mux.HandleFunc("/api/v1/iscsi/capabilities", api.iscsiCapabilities)
	mux.HandleFunc("/api/v1/iscsi/audit", api.iscsiAudit)
	mux.HandleFunc("/api/v1/sync/pairs", api.syncPairs)
	mux.HandleFunc("/api/v1/sync/pairs/", api.syncPairByID)
	mux.HandleFunc("/api/v1/sync/conflicts", api.syncConflicts)
	mux.HandleFunc("/api/v1/sync/conflicts/", api.syncConflictByID)
	mux.HandleFunc("/api/v1/sync/audit", api.syncAudit)
	mux.HandleFunc("/api/v1/app-center/apps", api.appCenterApps)
	mux.HandleFunc("/api/v1/app-center/apps/", api.appCenterAppByID)
	mux.HandleFunc("/api/v1/app-center/catalog", api.appCenterCatalog)
	mux.HandleFunc("/api/v1/app-center/catalog/", api.appCenterCatalogByID)
	mux.HandleFunc("/api/v1/app-center/registries", api.appCenterRegistries)
	mux.HandleFunc("/api/v1/app-center/registries/", api.appCenterRegistryByName)
	mux.HandleFunc("/api/v1/app-center/audit", api.appCenterAudit)
	mux.HandleFunc("/api/v1/app-center/audit/", api.appCenterAuditByID)
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
	mux.HandleFunc("/api/v1/media/scan", api.mediaScan)
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
	mux.HandleFunc("/api/v1/ai/index", api.aiIndexDocument)
	mux.HandleFunc("/api/v1/ai/index/files", api.aiIndexFiles)
	mux.HandleFunc("/api/v1/ai/index/media", api.aiIndexMedia)
	mux.HandleFunc("/api/v1/ai/index/status", api.aiIndexStatus)
	mux.HandleFunc("/api/v1/ai-analysis/status", api.aiAnalysisStatus)
	mux.HandleFunc("/api/v1/ai-analysis/records", api.aiAnalysisRecords)
	mux.HandleFunc("/api/v1/ai-analysis/records/", api.aiAnalysisRecordByKey)
	mux.HandleFunc("/api/v1/ai-analysis/batch", api.aiAnalysisBatch)
	mux.HandleFunc("/api/v1/ai-analysis/reanalyze", api.aiAnalysisReanalyze)
	mux.HandleFunc("/api/v1/ai-analysis/rescan", api.aiAnalysisRescan)
	mux.HandleFunc("/api/v1/ai-analysis/pause", api.aiAnalysisPause)
	mux.HandleFunc("/api/v1/ai-analysis/resume", api.aiAnalysisResume)
	mux.HandleFunc("/api/v1/ai-analysis/progress/stream", api.aiAnalysisProgressStream)
	mux.HandleFunc("/api/v1/ai-analysis/faces", api.aiAnalysisFaces)
	mux.HandleFunc("/api/v1/ai-analysis/faces/label", api.aiAnalysisFaceLabel)
	mux.HandleFunc("/api/v1/ai-analysis/faces/retrain", api.aiAnalysisFaceRetrain)
	mux.HandleFunc("/api/v1/assistant/threads", api.assistantThreads)
	mux.HandleFunc("/api/v1/assistant/threads/", api.assistantThreadByID)
	mux.HandleFunc("/api/v1/assistant/actions/", api.assistantActionByID)
	mux.HandleFunc("/api/v1/assistant/presets", api.assistantPresets)
	mux.HandleFunc("/api/v1/assistant/tools", api.assistantTools)
	mux.HandleFunc("/api/v1/ai/providers", api.aiProviders)
	mux.HandleFunc("/api/v1/ai/providers/", api.aiProviderByID)
	mux.HandleFunc("/api/v1/cloud/provision", api.cloudProvision)
	mux.HandleFunc("/api/v1/cloud/unprovision", api.cloudUnprovision)
	mux.HandleFunc("/api/v1/auth/login", api.authLogin)
	mux.HandleFunc("/api/v1/auth/logout", api.authLogout)
	mux.HandleFunc("/api/v1/auth/me", api.authMe)
	mux.HandleFunc("/api/v1/auth/password", api.authPassword)
	mux.HandleFunc("/api/v1/auth/sessions", api.authSessions)
	mux.HandleFunc("/api/v1/auth/sessions/", api.authSessionByID)
	mux.HandleFunc("/api/v1/auth/mfa/setup", api.authMFASetup)
	mux.HandleFunc("/api/v1/auth/mfa/enable", api.authMFAEnable)
	mux.HandleFunc("/api/v1/auth/mfa/disable", api.authMFADisable)
	mux.HandleFunc("/api/v1/auth/audit", api.authAudit)
	mux.HandleFunc("/api/v1/accounts/summary", api.accountsSummary)
	mux.HandleFunc("/api/v1/accounts/users", api.accountUsers)
	mux.HandleFunc("/api/v1/accounts/users/", api.accountUserByID)
	mux.HandleFunc("/api/v1/accounts/groups", api.accountGroups)
	mux.HandleFunc("/api/v1/accounts/groups/", api.accountGroupByID)
	mux.HandleFunc("/api/v1/accounts/grants", api.accountGrants)
	mux.HandleFunc("/api/v1/accounts/grants/", api.accountGrantByID)
	mux.HandleFunc("/api/v1/accounts/samba-sync", api.accountsSambaSync)
	mux.HandleFunc("/api/v1/steward/suggestions", api.stewardSuggestions)
	mux.HandleFunc("/api/v1/steward/suggestions/", api.stewardSuggestionByID)
	mux.HandleFunc("/api/v1/steward/audit", api.stewardAudit)
	mux.HandleFunc("/api/v1/steward/audit/", api.stewardAuditByID)
	mux.HandleFunc("/api/v1/security/identities", api.securityIdentities)
	mux.HandleFunc("/api/v1/security/identities/", api.securityIdentityByID)
	mux.HandleFunc("/api/v1/security/ai-policies", api.securityAIPolicies)
	mux.HandleFunc("/api/v1/security/ai-policies/", api.securityAIPolicyByID)
	mux.HandleFunc("/api/v1/security/inspect", api.securityInspect)
	mux.HandleFunc("/api/v1/security/risk-actions", api.securityRiskActions)
	mux.HandleFunc("/api/v1/security/risk-actions/", api.securityRiskActionByID)
	mux.HandleFunc("/api/v1/security/audit", api.securityAudit)
	mux.HandleFunc("/api/v1/security/audit/", api.securityAuditByID)
	mux.HandleFunc("/api/v1/security/host/ports", api.securityHostPorts)
	mux.HandleFunc("/api/v1/security/host/firewall", api.securityHostFirewall)
	mux.HandleFunc("/api/v1/security/host/scan", api.securityHostScan)
	mux.HandleFunc("/api/v1/shares", api.securityShares)
	mux.HandleFunc("/api/v1/shares/", api.securityShareByID)
	mux.HandleFunc("/api/v1/protocols", api.protocolsList)
	mux.HandleFunc("/api/v1/protocols/shares", api.protocolsShares)
	mux.HandleFunc("/api/v1/protocols/shares/", api.protocolShareByID)
	mux.HandleFunc("/api/v1/protocols/audit", api.protocolsAudit)
	mux.HandleFunc("/api/v1/protocols/audit/", api.protocolsAuditByID)
	mux.HandleFunc("/api/v1/protocols/", api.protocolByKey)
	mux.HandleFunc("/api/v1/shared-folders", api.sharedFoldersList)
	mux.HandleFunc("/api/v1/shared-folders/permissions/by-subject", api.sharedFolderSubjectPermissions)
	mux.HandleFunc("/api/v1/shared-folders/", api.sharedFolderByID)
	mux.HandleFunc("/api/v1/network/interfaces", api.networkInterfaces)
	mux.HandleFunc("/api/v1/network/config", api.networkConfig)
	mux.HandleFunc("/api/v1/network/config/confirm", api.networkConfigConfirm)
	mux.HandleFunc("/api/v1/network/audit", api.networkAudit)
	mux.HandleFunc("/api/v1/network/audit/", api.networkAuditByID)
	// The assistant drives the FULL tool catalog through an in-process MCP client
	// — the same Model Context Protocol surface external clients use. Read-only
	// tools execute inline; mutating/destructive tools (per their MCP annotation)
	// are gated behind the confirmation flow. Tools dispatch into this same mux.
	assistantLoopback := apiclient.NewInProcess(mux, apiclient.Auth{})
	if session, err := mcpclient.Connect(context.Background(), cfg, assistantLoopback); err != nil {
		logger.Warn("assistant mcp session unavailable", slog.Any("error", err))
	} else if catalog, err := session.Catalog(context.Background()); err != nil {
		logger.Warn("assistant mcp catalog unavailable", slog.Any("error", err))
	} else {
		assistantService.WithTools(assistantLoopback, catalog)
		logger.Info("assistant tools bound via mcp", slog.Int("count", len(catalog)))
	}

	if cfg.MCPEnabled {
		// Embedded Model Context Protocol endpoint. Tool handlers call back into
		// this same mux in-process (no network hop), forwarding the caller's
		// session/Authorization so calls run as the real actor. A fresh catalog
		// is built per MCP session (sessions are long-lived).
		loopback := apiclient.NewInProcess(mux, apiclient.Auth{})
		getServer := func(req *http.Request) *mcpsdk.Server {
			auth := apiclient.Auth{Authorization: req.Header.Get("Authorization")}
			if cookie, err := req.Cookie("higo_session"); err == nil {
				auth.SessionCookie = cookie.Value
			}
			return higomcp.BuildServer(cfg, loopback.WithAuth(auth))
		}
		streamable := mcpsdk.NewStreamableHTTPHandler(getServer, nil)
		mux.Handle("/mcp", streamable)
		mux.Handle("/mcp/", streamable)
	}

	mux.HandleFunc("/", api.staticAssets)

	return chain(
		mux,
		platform.RequestIDMiddleware,
		recoverPanic(logger),
		secureHeaders,
		cors(cfg.PublicOrigin),
		sessionGuard(api),
		csrfGuard(api),
		roleGate(api),
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
	config        platform.Config
	dev           *devstub.Store
	files         *files.Service
	monitoring    *monitoring.Service
	hardware      *hardware.Service
	settings      *settings.Store
	llm           *llm.Store
	storage       *storage.Service
	downloads     *downloads.Service
	docker        *hdocker.DevService
	backups       *backups.Service
	sync          *foldersync.Service
	appCenter     *appcenter.Service
	remote        *remote.Service
	media         *media.Service
	music         *music.Service
	video         *video.Service
	vm            *vm.Service
	iscsi         *iscsi.Service
	assistant     *assistant.Service
	accounts      *accounts.Service
	auth          *auth.SessionStore
	audit         *audit.Store
	loginThrottle *loginThrottle
	steward       *steward.Service
	security      *security.Service
	activity      *activity.Service
	protocols     *protocols.Service
	sharedFolders *sharedfolders.Service
	network       *network.Service
	identity      *identity.Provider
	tasks         *tasks.Manager
	index         *index.Service
	search        *search.Service
	aianalysis    *aianalysis.Engine
	staticDir     string
}

func taskStatePath(stateDir string) string {
	if strings.TrimSpace(stateDir) == "" {
		return ""
	}
	return filepath.Join(stateDir, "tasks.json")
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
