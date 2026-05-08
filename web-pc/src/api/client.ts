import { DELETE, GET, POST, PUT, createEventStream } from './runtime';
import type {
  AiPolicy,
  Alert,
  AlbumItem,
  AppCenterApp,
  AssistantMessage,
  AssistantThread,
  AuditEntry,
  BackupJob,
  ComposeStack,
  DesktopApp,
  DesktopSession,
  DesktopWindowConfig,
  Disk,
  DockerContainer,
  DomainToken,
  DownloadTask,
  FileRow,
  FileShare,
  FileTreeNode,
  IdentityPolicy,
  MediaItem,
  Metric,
  DiagnosticResult,
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
  StoragePool,
  StorageTask,
  SystemInfo,
  TaskResponse,
  AgentTemplate,
} from './types';

type Id = string | number;
type RecordPayload = Record<string, unknown>;

const pathId = (value: Id) => encodeURIComponent(String(value));

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
    getTree: (space?: string) => GET<FileTreeNode[]>('/api/v1/files/tree', { query: { space } }),
    search: (query: { q?: string; space?: string; type?: string; tags?: string[] }) =>
      GET<FileRow[]>('/api/v1/files/search', { query }),
    getFile: (id: Id) => GET<FileRow>(`/api/v1/files/${pathId(id)}`),
    getPreview: (id: Id) => GET<RecordPayload>(`/api/v1/files/${pathId(id)}/preview`),
    addTags: (id: Id, tags: string[]) => POST<FileRow>(`/api/v1/files/${pathId(id)}/tags`, { tags }),
    createShare: (id: Id, payload: RecordPayload) => POST<FileShare>(`/api/v1/files/${pathId(id)}/shares`, payload),
    moveBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/move', payload),
    renameBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/rename', payload),
    deleteBatch: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/files/batch/delete', payload),
    restore: (id: Id) => POST<TaskResponse>(`/api/v1/files/${pathId(id)}/restore`),
  },

  storage: {
    getPools: () => GET<StoragePool[]>('/api/v1/storage/pools'),
    getDisks: () => GET<Disk[]>('/api/v1/storage/disks'),
    getSmartReports: () => GET<RecordPayload[]>('/api/v1/storage/smart'),
    startSmartScan: (payload?: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/smart-scan', payload ?? {}),
    startRepair: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/repair', payload),
    createSnapshot: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/storage/tasks/snapshot', payload),
    getTask: (id: Id) => GET<StorageTask>(`/api/v1/storage/tasks/${pathId(id)}`),
  },

  steward: {
    getSuggestions: () => GET<StewardSuggestion[]>('/api/v1/steward/suggestions'),
    previewSuggestion: (id: Id) => POST<RecordPayload>(`/api/v1/steward/suggestions/${pathId(id)}/preview`),
    confirmSuggestion: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/steward/suggestions/${pathId(id)}/confirm`, payload ?? {}),
    dismissSuggestion: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/steward/suggestions/${pathId(id)}/dismiss`, payload ?? {}),
    getAudit: () => GET<AuditEntry[]>('/api/v1/steward/audit'),
    rollbackAudit: (id: Id) => POST<TaskResponse>(`/api/v1/steward/audit/${pathId(id)}/rollback`),
  },

  agents: {
    getTemplates: () => GET<AgentTemplate[]>('/api/v1/agents/templates'),
    createAgent: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/agents', payload),
    getTools: (id: Id) => GET<RecordPayload[]>(`/api/v1/agents/${pathId(id)}/tools`),
    previewWorkflow: (payload: RecordPayload) => POST<RecordPayload>('/api/v1/workflows/preview', payload),
    runWorkflow: (payload: RecordPayload) => POST<TaskResponse>('/api/v1/workflows/runs', payload),
    confirmWorkflowRun: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/workflows/runs/${pathId(id)}/confirm`, payload ?? {}),
    cancelWorkflowRun: (id: Id) => POST<TaskResponse>(`/api/v1/workflows/runs/${pathId(id)}/cancel`),
    streamWorkflowRun: (id: Id) => createEventStream(`/api/v1/workflows/runs/${pathId(id)}/events`),
  },

  assistant: {
    semanticSearch: (payload: RecordPayload) => POST<SemanticSearchResult>('/api/v1/search/semantic', payload),
    createThread: (payload?: RecordPayload) => POST<AssistantThread>('/api/v1/assistant/threads', payload ?? {}),
    getThread: (id: Id) => GET<AssistantThread>(`/api/v1/assistant/threads/${pathId(id)}`),
    sendMessage: (threadId: Id, message: Pick<AssistantMessage, 'role' | 'text'>) =>
      POST<AssistantMessage>(`/api/v1/assistant/threads/${pathId(threadId)}/messages`, message),
    confirmAction: (id: Id, payload?: RecordPayload) =>
      POST<TaskResponse>(`/api/v1/assistant/actions/${pathId(id)}/confirm`, payload ?? {}),
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

  downloads: {
    getTasks: () => GET<DownloadTask[]>('/api/v1/downloads/tasks'),
    createTask: (payload: RecordPayload) => POST<DownloadTask>('/api/v1/downloads/tasks', payload),
    pauseTask: (id: Id) => POST<TaskResponse>(`/api/v1/downloads/tasks/${pathId(id)}/pause`),
    resumeTask: (id: Id) => POST<TaskResponse>(`/api/v1/downloads/tasks/${pathId(id)}/resume`),
    archiveTask: (id: Id) => POST<TaskResponse>(`/api/v1/downloads/tasks/${pathId(id)}/archive`),
    deleteTask: (id: Id) => DELETE<TaskResponse>(`/api/v1/downloads/tasks/${pathId(id)}`),
    getSpeedProfiles: () => GET<SpeedProfile[]>('/api/v1/downloads/speed-profiles'),
    updateSpeedProfile: (payload: SpeedProfile) => PUT<SpeedProfile>('/api/v1/downloads/speed-profile', payload),
  },

  docker: {
    getStacks: () => GET<ComposeStack[]>('/api/v1/docker/stacks'),
    getContainers: () => GET<DockerContainer[]>('/api/v1/docker/containers'),
    getContainerLogs: (id: Id, tail = 20) => GET<string[]>(`/api/v1/docker/containers/${pathId(id)}/logs`, { query: { tail } }),
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
    installApp: (id: Id) => POST<AppCenterApp>(`/api/v1/app-center/apps/${pathId(id)}/install`),
    updateApp: (id: Id) => POST<AppCenterApp>(`/api/v1/app-center/apps/${pathId(id)}/update`),
    startApp: (id: Id) => POST<AppCenterApp>(`/api/v1/app-center/apps/${pathId(id)}/start`),
    stopApp: (id: Id) => POST<AppCenterApp>(`/api/v1/app-center/apps/${pathId(id)}/stop`),
  },

  security: {
    getIdentities: () => GET<IdentityPolicy[]>('/api/v1/security/identities'),
    updateIdentityPermissions: (id: Id, payload: RecordPayload) =>
      PUT<IdentityPolicy>(`/api/v1/security/identities/${pathId(id)}/permissions`, payload),
    getAiPolicies: () => GET<AiPolicy[]>('/api/v1/security/ai-policies'),
    updateAiPolicy: (id: Id, payload: Partial<AiPolicy>) =>
      PUT<AiPolicy>(`/api/v1/security/ai-policies/${pathId(id)}`, payload),
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

  monitoring: {
    getCurrentMetrics: () => GET<Metric[]>('/api/v1/monitoring/metrics/current'),
    getMetricTrend: (range: string, metric = 'cpu') =>
      GET<TrendPoint[]>('/api/v1/monitoring/metrics/trend', { query: { range, metric } }),
    getLogs: () => GET<SystemLog[]>('/api/v1/monitoring/logs'),
    getAlerts: () => GET<Alert[]>('/api/v1/monitoring/alerts'),
    createAlert: (payload: RecordPayload) => POST<Alert>('/api/v1/monitoring/alerts', payload),
    muteAlert: (id: Id, muted = true) => POST<Alert>(`/api/v1/monitoring/alerts/${pathId(id)}/mute`, { muted }),
    runDiagnostics: (payload?: RecordPayload) => POST<DiagnosticResult>('/api/v1/monitoring/diagnostics', payload ?? {}),
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
