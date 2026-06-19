import { DELETE, GET, POST, PUT, buildApiUrl, createEventStream, streamSSE } from './runtime';
import type { ChatStreamHandlers } from './runtime';
import type {
  AiPolicy,
  AiProvider,
  AiProviderInput,
  AiProviderTestResult,
  Alert,
  AlbumItem,
  AccountGroup,
  AccountSpaceGrant,
  AccountSummary,
  AccountUser,
  ActivityEntry,
  AppCenterApp,
  AppAction,
  AppActionPreview,
  AppActionResult,
  AppAuditRecord,
  AppCatalogEntry,
  AppRegistry,
  AssistantMessage,
  AssistantThread,
  ThreadSummary,
  AuditEntry,
  BackupJob,
  ComposeStack,
  DesktopApp,
  DesktopSession,
  DesktopWindowConfig,
  Disk,
  DockerContainer,
  DockerExecResult,
  DockerImage,
  DockerImagePullStatus,
  DockerImageSearchResult,
  DockerNetwork,
  DockerVolume,
  DownloadTaskActionResult,
  DomainToken,
  DownloadTask,
  DeleteVideoLibraryResult,
  FileRow,
  FileShare,
  FileTreeNode,
  HardwareInventory,
  IdentityPolicy,
  MediaItem,
  MusicAlbum,
  MusicLibrarySettings,
  MusicScanResult,
  MusicTrack,
  DvrSettings,
  LiveChannel,
  LiveGuideSource,
  LiveProgram,
  LiveSource,
  Metric,
  MetricsSnapshot,
  RecordingItem,
  RecordingTimer,
  DiagnosticResult,
  Protocol,
  ProtocolAuditEntry,
  ProtocolConfirmResult,
  ProtocolPreview,
  ProtocolShare,
  RemoteDevice,
  RemoteLoginAlert,
  RemoteStatus,
  RiskAction,
  SemanticSearchResult,
  ShareScanResult,
  SystemLog,
  TrendPoint,
  SettingsState,
  SpeedProfile,
  StewardSuggestion,
  StorageSpace,
  StoragePool,
  StorageTask,
  SystemInfo,
  Task,
  TaskResponse,
  AgentPreset,
  ToolCatalogEntry,
  VideoItem,
  VideoLibrary,
  VideoLibrarySettings,
  VideoScanResult,
  VideoTask,
} from './types';

type Id = string | number;
type RecordPayload = Record<string, unknown>;
type UploadFilesPayload = {
  space?: string;
  path?: string;
  actor?: string;
  files: File[];
  names?: string[];
};

const pathId = (value: Id) => encodeURIComponent(String(value));

function uploadFilesWithProgress(payload: UploadFilesPayload, onProgress?: (progress: number) => void) {
  const body = new FormData();
  if (payload.space) body.set('space', payload.space);
  if (payload.path) body.set('path', payload.path);
  if (payload.actor) body.set('actor', payload.actor);
  payload.files.forEach((file, index) => body.append('file', file, payload.names?.[index] ?? file.name));

  return new Promise<FileRow[]>((resolve, reject) => {
    const request = new XMLHttpRequest();
    request.open('POST', buildApiUrl('/api/v1/files/upload'));
    request.withCredentials = true;
    request.upload.onprogress = (event) => {
      if (event.lengthComputable && event.total > 0) {
        onProgress?.(Math.round((event.loaded / event.total) * 100));
      }
    };
    request.onerror = () => reject(new Error('上传连接失败'));
    request.onload = () => {
      let payloadBody: any;
      try {
        payloadBody = request.responseText ? JSON.parse(request.responseText) : undefined;
      } catch {
        payloadBody = request.responseText;
      }
      if (request.status < 200 || request.status >= 300 || payloadBody?.ok === false || payloadBody?.success === false) {
        const message =
          typeof payloadBody?.error === 'string'
            ? payloadBody.error
            : payloadBody?.error?.message || payloadBody?.message || request.statusText || '上传失败';
        reject(new Error(message));
        return;
      }
      onProgress?.(100);
      resolve((payloadBody?.data ?? payloadBody ?? []) as FileRow[]);
    };
    request.send(body);
  });
}

export const apiClient = {
  desktop: {
    getApps: () => GET<DesktopApp[]>('/api/v1/desktop/apps'),
    getWindows: () => GET<DesktopWindowConfig[]>('/api/v1/desktop/windows'),
    getSession: () => GET<DesktopSession>('/api/v1/desktop/session'),
    updateSession: (session: Partial<DesktopSession>) => PUT<DesktopSession>('/api/v1/desktop/session', session),
    streamEvents: () => createEventStream('/api/v1/events/stream'),
  },

  system: {
    getInfo: () => GET<SystemInfo>('/api/v1/system/info'),
    getUpdates: () => GET<RecordPayload>('/api/v1/system/updates'),
    checkUpdates: () => POST<TaskResponse>('/api/v1/system/updates/check'),
    createBackup: (payload?: RecordPayload) => POST<TaskResponse>('/api/v1/system/backups', payload ?? {}),
  },

  files: {
    getTree: (space?: string) => GET<FileTreeNode>('/api/v1/files/tree', { query: { space } }),
    search: (query: { q?: string; space?: string; type?: string; tags?: string[] }) =>
      GET<FileRow[]>('/api/v1/files/search', { query }),
    getFile: (id: Id) => GET<FileRow>(`/api/v1/files/${pathId(id)}`),
    getPreview: (id: Id) => GET<RecordPayload>(`/api/v1/files/${pathId(id)}/preview`),
    createFolder: (payload: RecordPayload) => POST<FileRow>('/api/v1/files/folders', payload),
    uploadFiles: (payload: UploadFilesPayload) => uploadFilesWithProgress(payload),
    uploadFilesWithProgress,
    downloadUrl: (id: Id) => buildApiUrl(`/api/v1/files/${pathId(id)}/download`),
    addTags: (id: Id, tags: string[]) => POST<FileRow>(`/api/v1/files/${pathId(id)}/tags`, { tags }),
    createShare: (id: Id, payload: RecordPayload) => POST<FileShare>(`/api/v1/files/${pathId(id)}/shares`, payload),
    rename: (id: Id, payload: RecordPayload) => POST<FileRow>(`/api/v1/files/${pathId(id)}/rename`, payload),
    move: (id: Id, payload: RecordPayload) => POST<FileRow>(`/api/v1/files/${pathId(id)}/move`, payload),
    delete: (id: Id, payload?: RecordPayload) => POST<FileRow>(`/api/v1/files/${pathId(id)}/delete`, payload ?? {}),
    moveBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/move', payload),
    renameBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/rename', payload),
    deleteBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/delete', payload),
    restore: (id: Id) => POST<TaskResponse>(`/api/v1/files/${pathId(id)}/restore`),
  },

  storage: {
    getPools: () => GET<StoragePool[]>('/api/v1/storage/pools'),
    getSpaces: () => GET<StorageSpace[]>('/api/v1/storage/spaces'),
    createSpace: (payload: RecordPayload) => POST<StorageSpace>('/api/v1/storage/spaces', payload),
    deleteSpace: (id: Id, payload?: RecordPayload) =>
      DELETE<StorageTask>(`/api/v1/storage/spaces/${pathId(id)}`, { body: payload ?? {} }),
    getDisks: () => GET<Disk[]>('/api/v1/storage/disks'),
    addDisk: (payload: RecordPayload) => POST<Disk>('/api/v1/storage/disks', payload),
    removeDisk: (slot: Id, payload?: RecordPayload) =>
      DELETE<StorageTask>(`/api/v1/storage/disks/${pathId(slot)}`, { body: payload ?? {} }),
    updateDiskSettings: (slot: Id, payload: RecordPayload) =>
      PUT<Disk>(`/api/v1/storage/disks/${pathId(slot)}/settings`, payload),
    getSmartReports: () => GET<RecordPayload[]>('/api/v1/storage/smart'),
    startSmartScan: (payload?: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/smart-scan', payload ?? {}),
    startRepair: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/repair', payload),
    createSnapshot: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/snapshot', payload),
    getTask: (id: Id) => GET<StorageTask>(`/api/v1/storage/tasks/${pathId(id)}`),
  },

  accounts: {
    getSummary: () => GET<AccountSummary>('/api/v1/accounts/summary'),
    getUsers: () => GET<AccountUser[]>('/api/v1/accounts/users'),
    createUser: (payload: RecordPayload) => POST<AccountUser>('/api/v1/accounts/users', payload),
    updateUser: (id: Id, payload: RecordPayload) => PUT<AccountUser>(`/api/v1/accounts/users/${pathId(id)}`, payload),
    deleteUser: (id: Id) => DELETE<TaskResponse>(`/api/v1/accounts/users/${pathId(id)}`),
    getGroups: () => GET<AccountGroup[]>('/api/v1/accounts/groups'),
    createGroup: (payload: RecordPayload) => POST<AccountGroup>('/api/v1/accounts/groups', payload),
    updateGroupMembers: (id: Id, userIds: Id[]) =>
      PUT<AccountGroup>(`/api/v1/accounts/groups/${pathId(id)}/members`, { userIds }),
    grantSpace: (payload: RecordPayload) => POST<AccountSpaceGrant>('/api/v1/accounts/grants', payload),
    deleteGrant: (id: Id) => DELETE<TaskResponse>(`/api/v1/accounts/grants/${pathId(id)}`),
  },

  steward: {
    getSuggestions: () => GET<StewardSuggestion[]>('/api/v1/steward/suggestions'),
    refresh: () => POST<StewardSuggestion[]>('/api/v1/steward/suggestions', {}),
    previewSuggestion: (id: Id) => POST<RecordPayload>(`/api/v1/steward/suggestions/${pathId(id)}/preview`),
    confirmSuggestion: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/steward/suggestions/${pathId(id)}/confirm`, payload ?? {}),
    dismissSuggestion: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/steward/suggestions/${pathId(id)}/dismiss`, payload ?? {}),
    getAudit: () => GET<AuditEntry[]>('/api/v1/steward/audit'),
    rollbackAudit: (id: Id) => POST<TaskResponse>(`/api/v1/steward/audit/${pathId(id)}/rollback`),
  },

  assistant: {
    semanticSearch: (payload: RecordPayload) => POST<SemanticSearchResult>('/api/v1/search/semantic', payload),
    getPresets: () => GET<AgentPreset[]>('/api/v1/assistant/presets'),
    getToolCatalog: () => GET<ToolCatalogEntry[]>('/api/v1/assistant/tools'),
    listThreads: () => GET<ThreadSummary[]>('/api/v1/assistant/threads'),
    createThread: (payload?: { title?: string; presetId?: string }) =>
      POST<AssistantThread>('/api/v1/assistant/threads', payload ?? {}),
    getThread: (id: Id) => GET<AssistantThread>(`/api/v1/assistant/threads/${pathId(id)}`),
    deleteThread: (id: Id) => DELETE<{ id: string; deleted: boolean }>(`/api/v1/assistant/threads/${pathId(id)}`),
    sendMessage: (threadId: Id, message: Pick<AssistantMessage, 'role' | 'text'>) =>
      POST<AssistantMessage>(`/api/v1/assistant/threads/${pathId(threadId)}/messages`, message),
    streamMessage: (threadId: Id, message: Pick<AssistantMessage, 'role' | 'text'>, handlers: ChatStreamHandlers) =>
      streamSSE(`/api/v1/assistant/threads/${pathId(threadId)}/messages`, message, handlers),
    confirmAction: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/assistant/actions/${pathId(id)}/confirm`, payload ?? {}),
    cancelAction: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/assistant/actions/${pathId(id)}/cancel`, payload ?? {}),
  },

  ai: {
    listProviders: () => GET<AiProvider[]>('/api/v1/ai/providers'),
    createProvider: (payload: AiProviderInput) => POST<AiProvider>('/api/v1/ai/providers', payload),
    updateProvider: (id: Id, payload: AiProviderInput) => PUT<AiProvider>(`/api/v1/ai/providers/${pathId(id)}`, payload),
    deleteProvider: (id: Id) => DELETE<{ id: string; deleted: boolean }>(`/api/v1/ai/providers/${pathId(id)}`),
    testProvider: (id: Id) => POST<AiProviderTestResult>(`/api/v1/ai/providers/${pathId(id)}/test`, {}),
    indexStatus: () => GET<RecordPayload>('/api/v1/ai/index/status'),
    reindexFiles: (payload?: { space?: string }) => POST<RecordPayload>('/api/v1/ai/index/files', payload ?? {}),
    reindexMedia: () => POST<RecordPayload>('/api/v1/ai/index/media', {}),
  },

  media: {
    getItems: (query?: { dimension?: string; facet?: string }) => GET<MediaItem[]>('/api/v1/media/items', { query }),
    getAlbums: () => GET<AlbumItem[]>('/api/v1/media/albums'),
    createAlbum: (payload: RecordPayload) => POST<AlbumItem>('/api/v1/media/albums', payload),
    createMemory: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/media/memories', payload),
    mergePeople: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/media/people/merge', payload),
    createSubtitleJob: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/media/subtitles/jobs', payload),
    createTranscodeJob: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/media/transcode/jobs', payload),
    createShare: (payload: RecordPayload) => POST<FileShare>('/api/v1/media/shares', payload),
  },

  music: {
    getLibrary: () => GET<MusicLibrarySettings>('/api/v1/music/library'),
    updateLibrary: (payload: { paths: string[]; autoScan?: boolean }) =>
      PUT<MusicLibrarySettings>('/api/v1/music/library', payload),
    scan: () => POST<MusicScanResult>('/api/v1/music/scan'),
    getTracks: (query?: { q?: string }) => GET<MusicTrack[]>('/api/v1/music/tracks', { query }),
    getAlbums: () => GET<MusicAlbum[]>('/api/v1/music/albums'),
    getLyrics: (id: Id) => GET<string>(`/api/v1/music/tracks/${pathId(id)}/lyrics`, { parseAs: 'text' }),
    streamUrl: (id: Id) => buildApiUrl(`/api/v1/music/tracks/${pathId(id)}/stream`),
    coverUrl: (id: Id) => buildApiUrl(`/api/v1/music/tracks/${pathId(id)}/cover`),
  },

  video: {
    getLibrary: () => GET<VideoLibrarySettings>('/api/v1/videos/library'),
    updateLibrary: (payload: { libraries: VideoLibrary[] }) => PUT<VideoLibrarySettings>('/api/v1/videos/library', payload),
    createLibrary: (payload: {
      name: string;
      type: string;
      paths: string[];
      metadataLanguage?: string;
      allowAdultContent?: boolean;
      autoSubtitles?: boolean;
      subtitleLanguage?: string;
    }) => POST<VideoLibrary>('/api/v1/videos/library', payload),
    deleteLibrary: (id: Id) => DELETE<DeleteVideoLibraryResult>(`/api/v1/videos/library/${pathId(id)}`),
    scan: () => POST<VideoScanResult>('/api/v1/videos/scan'),
    getItems: (query?: { q?: string; libraryId?: string; kind?: string }) =>
      GET<VideoItem[]>('/api/v1/videos/items', { query }),
    getItem: (id: Id) => GET<VideoItem>(`/api/v1/videos/items/${pathId(id)}`),
    getTasks: () => GET<VideoTask[]>('/api/v1/videos/tasks'),
    scrape: (payload: { itemId: string }) => POST<VideoTask>('/api/v1/videos/tasks/scrape', payload),
    subtitle: (payload: { itemId: string }) => POST<VideoTask>('/api/v1/videos/tasks/subtitle', payload),
    transcode: (payload: { itemId: string; profile?: string }) => POST<VideoTask>('/api/v1/videos/tasks/transcode', payload),
    getLiveSources: () => GET<LiveSource[]>('/api/v1/videos/live/sources'),
    createLiveSource: (payload: { name: string; url: string; userAgent?: string; streamLimit?: number }) =>
      POST<LiveSource>('/api/v1/videos/live/sources', payload),
    getLiveChannels: (sourceId?: string) => GET<LiveChannel[]>('/api/v1/videos/live/channels', { query: { sourceId } }),
    getGuideSources: () => GET<LiveGuideSource[]>('/api/v1/videos/live/guide-sources'),
    createGuideSource: (payload: { name: string; url: string; userAgent?: string }) =>
      POST<LiveGuideSource>('/api/v1/videos/live/guide-sources', payload),
    getPrograms: (query?: { sourceId?: string; channelId?: string; from?: string; to?: string }) =>
      GET<LiveProgram[]>('/api/v1/videos/live/programs', { query }),
    getDvrSettings: () => GET<DvrSettings>('/api/v1/videos/live/dvr/settings'),
    updateDvrSettings: (payload: DvrSettings) => PUT<DvrSettings>('/api/v1/videos/live/dvr/settings', payload),
    getRecordingTimers: () => GET<RecordingTimer[]>('/api/v1/videos/live/recording-timers'),
    createRecordingTimer: (payload: { programId?: string; channelId?: string; name?: string; startAt?: string; endAt?: string }) =>
      POST<RecordingTimer>('/api/v1/videos/live/recording-timers', payload),
    cancelRecordingTimer: (id: Id) => DELETE<RecordingTimer>(`/api/v1/videos/live/recording-timers/${pathId(id)}`),
    getRecordings: () => GET<RecordingItem[]>('/api/v1/videos/live/recordings'),
    streamUrl: (id: Id) => buildApiUrl(`/api/v1/videos/items/${pathId(id)}/stream`),
    posterUrl: (id: Id) => buildApiUrl(`/api/v1/videos/items/${pathId(id)}/poster`),
    subtitleUrl: (id: Id) => buildApiUrl(`/api/v1/videos/items/${pathId(id)}/subtitle`),
  },

  downloads: {
    getTasks: () => GET<DownloadTask[]>('/api/v1/downloads/tasks'),
    createTask: (payload: RecordPayload) => POST<DownloadTask>('/api/v1/downloads/tasks', payload),
    pauseTask: (id: Id) => POST<DownloadTask>(`/api/v1/downloads/tasks/${pathId(id)}/pause`),
    resumeTask: (id: Id) => POST<DownloadTask>(`/api/v1/downloads/tasks/${pathId(id)}/resume`),
    archiveTask: (id: Id) => POST<DownloadTaskActionResult>(`/api/v1/downloads/tasks/${pathId(id)}/archive`),
    deleteTask: (id: Id, deleteFile = false) =>
      DELETE<DownloadTaskActionResult>(`/api/v1/downloads/tasks/${pathId(id)}`, { query: { deleteFile } }),
    getSpeedProfiles: () => GET<SpeedProfile[]>('/api/v1/downloads/speed-profiles'),
    updateSpeedProfile: (payload: SpeedProfile) => PUT<SpeedProfile>('/api/v1/downloads/speed-profile', payload),
  },

  tasks: {
    list: async (kind?: string) =>
      (await GET<{ tasks: Task[] }>('/api/v1/tasks', kind ? { query: { kind } } : undefined)).tasks ?? [],
    get: (id: Id) => GET<Task>(`/api/v1/tasks/${pathId(id)}`),
    cancel: (id: Id) => POST<Task>(`/api/v1/tasks/${pathId(id)}/cancel`),
    /** Live SSE stream of task snapshots (one frame per task state change). */
    stream: () => createEventStream('/api/v1/tasks/stream'),
  },

  activity: {
    list: async (type?: string, limit?: number) =>
      (await GET<{ entries: ActivityEntry[] }>('/api/v1/activity', { query: { type, limit } })).entries ?? [],
    append: (entries: ActivityEntry[]) =>
      POST<{ entries: ActivityEntry[]; count: number }>('/api/v1/activity', { entries }),
    clear: () => DELETE<{ cleared: boolean }>('/api/v1/activity'),
  },

  docker: {
    getStacks: () => GET<ComposeStack[]>('/api/v1/docker/stacks'),
    getContainers: () => GET<DockerContainer[]>('/api/v1/docker/containers'),
    getImages: () => GET<DockerImage[]>('/api/v1/docker/images'),
    getVolumes: () => GET<DockerVolume[]>('/api/v1/docker/volumes'),
    getNetworks: () => GET<DockerNetwork[]>('/api/v1/docker/networks'),
    searchImages: (q: string) => GET<DockerImageSearchResult[]>('/api/v1/docker/images/search', { query: { q } }),
    getImagePulls: () => GET<DockerImagePullStatus[]>('/api/v1/docker/images/pulls'),
    pullImage: (payload: RecordPayload) => POST<DockerImagePullStatus>('/api/v1/docker/images/pull', payload),
    removeImage: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/docker/images/remove', payload),
    createVolume: (payload: RecordPayload) => POST<DockerVolume>('/api/v1/docker/volumes', payload),
    removeVolume: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/docker/volumes/remove', payload),
    createNetwork: (payload: RecordPayload) => POST<DockerNetwork>('/api/v1/docker/networks', payload),
    removeNetwork: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/docker/networks/remove', payload),
    connectNetwork: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/docker/networks/connect', payload),
    disconnectNetwork: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/docker/networks/disconnect', payload),
    createContainer: (payload: RecordPayload) => POST<DockerContainer>('/api/v1/docker/containers', payload),
    removeContainer: (id: Id, payload: RecordPayload) =>
      DELETE<RecordPayload>(`/api/v1/docker/containers/${pathId(id)}`, { body: payload }),
    getContainerLogs: (id: Id, tail = 20) => GET<string[]>(`/api/v1/docker/containers/${pathId(id)}/logs`, { query: { tail } }),
    terminalUrl: (id: Id, shell = '/bin/sh') => {
      const url = new URL(
        buildApiUrl(`/api/v1/docker/containers/${pathId(id)}/terminal`, { shell }),
        typeof window === 'undefined' ? 'http://localhost' : window.location.href,
      );
      url.protocol = url.protocol === 'https:' ? 'wss:' : 'ws:';
      return url.toString();
    },
    execContainer: (id: Id, payload: RecordPayload) =>
      POST<DockerExecResult>(`/api/v1/docker/containers/${pathId(id)}/exec`, payload),
    startContainer: (id: Id) => POST<DockerContainer>(`/api/v1/docker/containers/${pathId(id)}/start`),
    stopContainer: (id: Id) => POST<DockerContainer>(`/api/v1/docker/containers/${pathId(id)}/stop`),
    restartContainer: (id: Id) => POST<DockerContainer>(`/api/v1/docker/containers/${pathId(id)}/restart`),
    completeRestart: (id: Id) => POST<DockerContainer>(`/api/v1/docker/containers/${pathId(id)}/complete-restart`),
    updateContainerLimits: (id: Id, payload: RecordPayload) =>
      PUT<DockerContainer>(`/api/v1/docker/containers/${pathId(id)}/limits`, payload),
  },

  backup: {
    getJobs: () => GET<BackupJob[]>('/api/v1/backups/jobs'),
    runJob: (id: Id) => POST<BackupJob>(`/api/v1/backups/jobs/${pathId(id)}/run`),
    pauseJob: (id: Id) => POST<BackupJob>(`/api/v1/backups/jobs/${pathId(id)}/pause`),
    resumeJob: (id: Id) => POST<BackupJob>(`/api/v1/backups/jobs/${pathId(id)}/resume`),
    verifyJob: (id: Id) => POST<BackupJob>(`/api/v1/backups/jobs/${pathId(id)}/verify`),
  },

  appCenter: {
    getApps: () => GET<AppCenterApp[]>('/api/v1/app-center/apps'),
    getCatalog: () => GET<AppCatalogEntry[]>('/api/v1/app-center/catalog'),
    getCatalogItem: (id: Id) => GET<AppCatalogEntry>(`/api/v1/app-center/catalog/${pathId(id)}`),
    refreshCatalog: () => POST<AppCatalogEntry[]>('/api/v1/app-center/catalog/refresh'),
    preview: (id: Id, action: AppAction, config?: Record<string, string>) =>
      POST<AppActionPreview>(`/api/v1/app-center/apps/${pathId(id)}/${action}`, config ? { config } : {}),
    confirm: (id: Id, action: AppAction, confirmationId: string, actor: string, config?: Record<string, string>) =>
      POST<AppActionResult>(`/api/v1/app-center/apps/${pathId(id)}/${action}/confirm`, {
        confirmationId,
        actor,
        ...(config ? { config } : {}),
      }),
    getAudit: () => GET<AppAuditRecord[]>('/api/v1/app-center/audit'),
    rollback: (auditId: Id, actor: string) =>
      POST<AppAuditRecord>(`/api/v1/app-center/audit/${pathId(auditId)}/rollback`, { actor }),
    getRegistries: () => GET<AppRegistry[]>('/api/v1/app-center/registries'),
    addRegistry: (name: string, url: string) =>
      POST<AppRegistry>('/api/v1/app-center/registries', { name, url }),
    removeRegistry: (name: string) => DELETE<AppRegistry[]>(`/api/v1/app-center/registries/${pathId(name)}`),
  },

  security: {
    getIdentities: () => GET<IdentityPolicy[]>('/api/v1/security/identities'),
    updateIdentityPermissions: (id: Id, payload: RecordPayload) =>
      PUT<IdentityPolicy>(`/api/v1/security/identities/${pathId(id)}/permissions`, payload),
    getAiPolicies: () => GET<AiPolicy[]>('/api/v1/security/ai-policies'),
    updateAiPolicy: (id: Id, payload: Partial<AiPolicy>) =>
      PUT<AiPolicy>(`/api/v1/security/ai-policies/${pathId(id)}`, payload),
    inspect: () => POST<RiskAction[]>('/api/v1/security/inspect', {}),
    getRiskActions: () => GET<RiskAction[]>('/api/v1/security/risk-actions'),
    confirmRiskAction: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/security/risk-actions/${pathId(id)}/confirm`, payload ?? {}),
    blockRiskAction: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/security/risk-actions/${pathId(id)}/block`, payload ?? {}),
    getAudit: () => GET<AuditEntry[]>('/api/v1/security/audit'),
    rollbackAudit: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/security/audit/${pathId(id)}/rollback`, payload ?? {}),
    getShares: () => GET<FileShare[]>('/api/v1/shares'),
    deleteShare: (id: Id) => DELETE<TaskResponse>(`/api/v1/shares/${pathId(id)}`),
  },

  protocols: {
    list: () => GET<Protocol[]>('/api/v1/protocols'),
    get: (key: string) => GET<Protocol>(`/api/v1/protocols/${pathId(key)}`),
    getShares: (key?: string) =>
      key
        ? GET<ProtocolShare[]>(`/api/v1/protocols/${pathId(key)}/shares`)
        : GET<ProtocolShare[]>('/api/v1/protocols/shares'),
    getAudit: () => GET<ProtocolAuditEntry[]>('/api/v1/protocols/audit'),
    updateConfig: (key: string, payload: RecordPayload) =>
      PUT<Protocol>(`/api/v1/protocols/${pathId(key)}/config`, payload),
    previewEnable: (key: string, payload?: RecordPayload) =>
      POST<ProtocolPreview>(`/api/v1/protocols/${pathId(key)}/enable/preview`, payload ?? {}),
    confirmEnable: (key: string, payload: RecordPayload) =>
      POST<ProtocolConfirmResult>(`/api/v1/protocols/${pathId(key)}/enable/confirm`, payload),
    previewDisable: (key: string, payload?: RecordPayload) =>
      POST<ProtocolPreview>(`/api/v1/protocols/${pathId(key)}/disable/preview`, payload ?? {}),
    confirmDisable: (key: string, payload: RecordPayload) =>
      POST<ProtocolConfirmResult>(`/api/v1/protocols/${pathId(key)}/disable/confirm`, payload),
    previewShare: (key: string, payload: RecordPayload) =>
      POST<ProtocolPreview>(`/api/v1/protocols/${pathId(key)}/shares/preview`, payload),
    confirmShare: (key: string, payload: RecordPayload) =>
      POST<ProtocolConfirmResult>(`/api/v1/protocols/${pathId(key)}/shares/confirm`, payload),
    previewDeleteShare: (id: Id, payload?: RecordPayload) =>
      POST<ProtocolPreview>(`/api/v1/protocols/shares/${pathId(id)}/delete/preview`, payload ?? {}),
    confirmDeleteShare: (id: Id, payload: RecordPayload) =>
      POST<ProtocolConfirmResult>(`/api/v1/protocols/shares/${pathId(id)}/delete/confirm`, payload),
    rollbackAudit: (id: Id, payload?: RecordPayload) =>
      POST<ProtocolAuditEntry>(`/api/v1/protocols/audit/${pathId(id)}/rollback`, payload ?? {}),
  },

  monitoring: {
    getMetricsSnapshot: () => GET<MetricsSnapshot>('/api/v1/monitoring/metrics/snapshot'),
    getCurrentMetrics: () => GET<Metric[]>('/api/v1/monitoring/metrics/current'),
    getMetricTrend: (range: string, metric = 'cpu') =>
      GET<TrendPoint[]>('/api/v1/monitoring/metrics/trend', { query: { range, metric } }),
    getLogs: () => GET<SystemLog[]>('/api/v1/monitoring/logs'),
    getAlerts: () => GET<Alert[]>('/api/v1/monitoring/alerts'),
    createAlert: (payload: RecordPayload) => POST<Alert>('/api/v1/monitoring/alerts', payload),
    muteAlert: (id: Id, muted = true) => POST<Alert>(`/api/v1/monitoring/alerts/${pathId(id)}/mute`, { muted }),
    runDiagnostics: (payload?: RecordPayload) => POST<DiagnosticResult>('/api/v1/monitoring/diagnostics', payload ?? {}),
  },

  hardware: {
    getInventory: () => GET<HardwareInventory>('/api/v1/hardware/inventory'),
  },

  settings: {
    getSettings: () => GET<SettingsState>('/api/v1/settings'),
    updateSettings: (payload: SettingsState) => PUT<SettingsState>('/api/v1/settings', payload),
    restoreDefaults: () => POST<SettingsState>('/api/v1/settings/defaults'),
    getUpdates: () => GET<RecordPayload>('/api/v1/system/updates'),
    checkUpdates: () => POST<TaskResponse>('/api/v1/system/updates/check'),
    createSystemBackup: (payload?: RecordPayload) => POST<TaskResponse>('/api/v1/system/backups', payload ?? {}),
  },

  remote: {
    getStatus: () => GET<RemoteStatus>('/api/v1/remote/status'),
    startChannel: () => POST<RemoteStatus>('/api/v1/remote/channel/start'),
    stopChannel: () => POST<RemoteStatus>('/api/v1/remote/channel/stop'),
    updateTunnelMode: (payload: { mode: RemoteStatus['tunnelMode']; reason?: string }) =>
      PUT<RemoteStatus>('/api/v1/remote/tunnel-mode', payload),
    updateMfa: (enabled: boolean) => PUT<RemoteStatus>('/api/v1/remote/mfa', { enabled }),
    selectPolicy: (key: string) => PUT<RemoteStatus>('/api/v1/remote/policy', { key }),
    createDomainToken: (payload?: RecordPayload) => POST<DomainToken>('/api/v1/remote/domain-token', payload ?? {}),
    rotateDomainToken: () => POST<DomainToken>('/api/v1/remote/domain-token/rotate'),
    getDevices: () => GET<RemoteDevice[]>('/api/v1/remote/devices'),
    bindDevice: (id: Id) => POST<RemoteDevice>(`/api/v1/remote/devices/${pathId(id)}/bind`),
    unbindDevice: (id: Id) => POST<RemoteDevice>(`/api/v1/remote/devices/${pathId(id)}/unbind`),
    getLoginAlerts: () => GET<RemoteLoginAlert[]>('/api/v1/remote/login-alerts'),
    scanShare: (payload?: RecordPayload) => POST<ShareScanResult>('/api/v1/remote/share-scan', payload ?? {}),
  },
};

export type ApiClient = typeof apiClient;
