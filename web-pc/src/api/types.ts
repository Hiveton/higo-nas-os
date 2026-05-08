export type StatusTone = 'blue' | 'green' | 'orange' | 'red' | 'cyan';
export type RiskLevel = '低风险' | '中风险' | '高风险';
export type RiskState = '待处理' | '已确认' | '已阻止';

export type DesktopApp = {
  id: string;
  name: string;
  icon: string;
  badge?: number;
  utility?: boolean;
  status?: string;
};

export type DesktopWindowConfig = {
  id: string;
  title: string;
  subtitle: string;
  status: string;
  statusTone: StatusTone;
  x: number;
  y: number;
  width: number;
  height: number;
  z: number;
};

export type DesktopSession = {
  openWindowIds: string[];
  minimizedWindowIds: string[];
  activeWindowId: string;
  utilityAppId?: string;
  assistantVisible: boolean;
  isCompact?: boolean;
  maximizedWindowId?: string;
  dockOrder: string[];
  pinnedDockAppIds: string[];
  desktopIconPositions?: Record<string, { x: number; y: number }>;
  windowGeometries?: Record<string, Partial<{ x: number; y: number; width: number; height: number }>>;
};

export type SystemInfo = {
  appName?: string;
  deviceName?: string;
  hostname?: string;
  version: string;
  environment?: string;
  adapter?: string;
  status?: string;
  hostOS?: string;
  arch?: string;
  bootedAt?: string;
  uptime?: string;
  updateStatus?: string;
  channelState?: string;
  modelPolicy?: string;
  localAi?: boolean;
  cloudAi?: boolean;
};

export type DesktopBootstrap = {
  apps: DesktopApp[];
  windows: DesktopWindowConfig[];
  status: DesktopSession | SystemInfo | Record<string, unknown>;
  fallback: boolean;
};

export type Metric = {
  key?: string;
  label: string;
  value: string | number;
  unit?: string;
  trend?: string;
  detail?: string;
  tone?: StatusTone | string;
  icon?: unknown;
};

export type ServiceStatus = {
  key?: string;
  label: string;
  value: string | number;
  detail: string;
  tone?: StatusTone | string;
};

export type TrendPoint = {
  at?: string;
  value: number;
};

export type SystemLog = {
  id: string;
  level: string;
  source: string;
  message: string;
  at: string;
  timestamp?: string;
};

export type Alert = {
  id?: string;
  title: string;
  detail: string;
  tone?: StatusTone | string;
  severity?: RiskLevel | string;
  source?: string;
  muted?: boolean;
  state?: string;
  icon?: unknown;
};

export type DiagnosticResult = {
  id: string;
  status?: string;
  state?: string;
  message?: string;
  summary?: string;
};

export type FileRow = {
  id?: string;
  name: string;
  type: string;
  space: string;
  size: string;
  modified: string;
  tags: string[];
  permission: string;
  aiSummary: string;
  path?: string;
  previewUrl?: string;
};

export type FileTreeNode = {
  id: string;
  name: string;
  type: 'space' | 'folder' | 'file' | string;
  path: string;
  children?: FileTreeNode[];
};

export type FileShare = {
  id: string;
  name: string;
  target: string;
  access: string;
  downloads: number;
  risk: RiskLevel;
  active: boolean;
};

export type StoragePool = {
  id?: string;
  name: string;
  type: string;
  used: number;
  total: string;
  health: string;
  temp: string;
};

export type Disk = {
  id?: string;
  slot: string;
  size: string;
  state: string;
  temp: string;
  serial?: string;
  health?: string;
  role?: string;
  poolId?: string;
  model?: string;
  interface?: string;
  smart?: string;
};

export type StorageTask = {
  id: string;
  kind: 'smart-scan' | 'repair' | 'snapshot' | string;
  state: string;
  progress?: number;
  message?: string;
};

export type StewardSuggestion = {
  id?: string;
  title: string;
  detail: string;
  count: string;
  risk: RiskLevel;
  action: string;
};

export type AgentTemplate = {
  id?: string;
  name: string;
  desc: string;
  tools: string[];
  risk: string;
};

export type WorkflowNode = {
  id?: string;
  label: string;
  value: string;
  icon?: unknown;
};

export type AssistantRole = 'user' | 'assistant' | 'system' | 'tool';

export type AssistantMessage = {
  id?: string;
  role: AssistantRole;
  text: string;
  createdAt?: string;
  citations?: Array<{ title: string; path?: string; url?: string }>;
  pendingActionId?: string;
};

export type AssistantThread = {
  id: string;
  title?: string;
  messages: AssistantMessage[];
};

export type RiskAction = {
  id: string;
  title: string;
  level: RiskLevel;
  scope: string;
  actor: string;
  state: RiskState;
  confirmed: boolean;
  rollback: string;
};

export type AuditEntry = {
  id: string;
  event: string;
  actor: string;
  risk: RiskLevel;
  reverted: boolean;
  rollback: string;
};

export type IdentityPolicy = {
  id?: string;
  role: string;
  name: string;
  mfa: boolean;
  fileAcl: boolean;
  appAdmin: boolean;
  aiTools: boolean;
};

export type AiPolicy = {
  id?: string;
  space: string;
  indexed: boolean;
  cloudModel: boolean;
  sensitive: string;
};

export type DownloadTask = {
  id: number | string;
  name: string;
  source: 'BT' | 'HTTP' | '磁力' | '订阅' | string;
  category: string;
  size: string;
  progress: number;
  speed: string;
  status: '下载中' | '暂停' | '已完成' | string;
  handling: string;
  archived: boolean;
};

export type SpeedProfile = {
  id?: string;
  name?: string;
  down?: string;
  up?: string;
  note?: string;
  downloadLimitBytesPerSecond?: number;
  uploadLimitBytesPerSecond?: number;
  schedule?: string;
  active?: boolean;
};

export type ComposeStack = {
  name: string;
  status: string;
  services: number;
  ports: string;
  volume: string;
  network: string;
};

export type DockerContainer = {
  id: string;
  name: string;
  image: string;
  stack: string;
  status: '运行中' | '已停止' | '重启中' | string;
  cpu: number;
  memory: number;
  memoryText: string;
  ports: string[];
  mounts: string[];
  env: string[];
  limitCpu: number;
  limitMemory: number;
  restarts: number;
  isolation: string;
  log: string[];
};

export type BackupJob = {
  id: string;
  name: string;
  source: string;
  target: string;
  state: string;
  schedule: string;
  progress: number;
  speed: string;
  eta: string;
  lastRun: string;
  nextRun: string;
  retention: string;
  policy: string;
  health: string;
  enabled: boolean;
};

export type AppCenterApp = {
  id: string;
  name: string;
  category: string;
  version: string;
  latestVersion: string;
  status: string;
  description: string;
  source: string;
  risk: RiskLevel | string;
  resource: string;
  ports: string[];
  installed: boolean;
  running: boolean;
  updateAvailable: boolean;
};

export type MediaItem = {
  id: number | string;
  title: string;
  kind: '照片' | '视频' | '音乐' | string;
  timeline: string;
  people: string;
  place: string;
  device: string;
  album: string;
  meta: string;
  status: string;
  accent?: string;
  hasSubtitle?: boolean;
  transcoded?: boolean;
};

export type AlbumItem = {
  id: number | string;
  name: string;
  type: '家庭相册' | '共享相册' | '智能回忆' | string;
  count: number;
  privacy: string;
};

export type SettingsState = {
  model?: {
    mode?: 'family_hybrid' | 'provider' | 'enterprise_local' | string;
    provider?: string;
    localModel?: string;
    cloudModel?: string;
    cloudEnabled?: boolean;
  };
  privacy?: {
    sensitiveDataLocalOnly?: boolean;
    auditRetentionDays?: number;
  };
};

export type AccessPolicy = {
  key: string;
  name: string;
  scope: string;
  risk: RiskLevel | string;
};

export type DomainToken = {
  version: number;
  domain: string;
  token: string;
  expiresAt?: string;
};

export type ShareScanResult = {
  state: 'idle' | 'safe' | 'risk' | string;
  message: string;
  checks: string[];
};

export type RemoteStatus = {
  enabled: boolean;
  channelEnabled?: boolean;
  channelState?: string;
  mfaEnabled: boolean;
  tunnelMode: '智能中继' | '直连优先' | string;
  tunnelState?: string;
  domain: string;
  token?: DomainToken;
  tokenState?: string;
  boundDeviceCount?: number;
  deviceCount?: number;
  activePolicy?: AccessPolicy;
  policies?: AccessPolicy[];
  feedback?: string;
};

export type RemoteDevice = {
  id: string;
  name: string;
  role: string;
  location: string;
  bound: boolean;
  lastSeen: string;
};

export type RemoteLoginAlert = {
  id: string;
  location: string;
  device: string;
  action: string;
  state: string;
};

export type SemanticSearchResult = {
  answer: string;
  items: FileRow[];
  citations?: Array<{ fileId?: string; title: string; snippet?: string; path?: string }>;
};

export type TaskResponse = {
  id: string;
  state: string;
  message?: string;
};
