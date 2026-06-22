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

export type MetricsSnapshot = {
  metrics: Metric[];
  services: ServiceStatus[];
  collectedAt?: string;
};

export type TrendPoint = {
  at?: string;
  value: number;
};

export type HardwareHostInfo = {
  hostname: string;
  osName: string;
  kernel: string;
  arch: string;
  uptime: string;
  bootedAt?: string;
};

export type HardwareBoardInfo = {
  vendor: string;
  product: string;
  serial: string;
  bios: string;
};

export type HardwareCpuInfo = {
  model: string;
  physicalCores: number;
  logicalCores: number;
  mhz: number;
};

export type HardwareMemoryInfo = {
  totalBytes: number;
  usedBytes: number;
  usedPercent: number;
};

export type HardwareNetworkInterface = {
  name: string;
  mac: string;
  ipv4: string[];
  speedMbps: number;
  state: string;
};

export type HardwareSensorReading = {
  label: string;
  celsius?: number;
  rpm?: number;
};

export type HardwareSensors = {
  temperatures: HardwareSensorReading[];
  fans: HardwareSensorReading[];
};

export type HardwareInventory = {
  host: HardwareHostInfo;
  board: HardwareBoardInfo;
  cpu: HardwareCpuInfo;
  memory: HardwareMemoryInfo;
  network: HardwareNetworkInterface[];
  sensors: HardwareSensors;
  adapter: string;
  collectedAt?: string;
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
  sizeBytes?: number;
  modified: string;
  tags: string[];
  permission: string;
  aiSummary: string;
  path?: string;
  previewUrl?: string;
  isDir?: boolean;
};

export type FileTreeNode = {
  id: string;
  name: string;
  type: 'space' | 'folder' | 'file' | string;
  path: string;
  space?: string;
  size?: string;
  sizeBytes?: number;
  modifiedAt?: string;
  modified?: string;
  tags?: string[];
  permission?: string;
  aiSummary?: string;
  isDir?: boolean;
  category?: 'personal' | 'shared' | 'user' | string;
  spaceId?: string;
  children?: FileTreeNode[];
};

export type FileShare = {
  id: string;
  name: string;
  target: string;
  url?: string;
  access: string;
  downloads: number;
  risk: RiskLevel;
  active: boolean;
};

export type FileTrashEntry = {
  id: string;
  name: string;
  originalPath: string;
  space: string;
  size: string;
  sizeBytes: number;
  deletedAt: string;
};

// --- network sharing protocols (SMB / NFS / WebDAV / DLNA) ------------------

export type ProtocolKey = 'smb' | 'nfs' | 'webdav' | 'dlna';
export type ProtocolAccessLevel = 'public' | 'password' | 'account' | 'readonly';
export type ProtocolRisk = 'low' | 'medium' | 'high';

export type ProtocolConfig = {
  // SMB
  serverName?: string;
  workgroup?: string;
  minProtocol?: string; // SMB2 | SMB3
  guestAccess?: boolean;
  // NFS
  squash?: string; // root_squash | all_squash | no_root_squash
  allowedNetwork?: string;
  // WebDAV
  httpsEnabled?: boolean;
  // DLNA
  friendlyName?: string;
};

export type Protocol = {
  key: ProtocolKey;
  displayName: string;
  enabled: boolean;
  running: boolean;
  installed: boolean;
  mountHint: string;
  port: number;
  compatibility: string;
  config: ProtocolConfig;
};

export type ProtocolShare = {
  id: string;
  protocol: ProtocolKey;
  name: string;
  path: string;
  accessLevel: ProtocolAccessLevel;
  allowedUsers?: string[];
  writeUsers?: string[];
  guest: boolean;
  enabled: boolean;
  createdAt: string;
  createdBy?: string;
};

// --- shared folders (unified user/group permission + SMB/NFS) ---------------

export type SharedFolderAccess = 'none' | 'read' | 'read_write' | 'deny';

export type SharedFolderPermission = {
  subjectType: 'user' | 'group';
  subjectId: string;
  subjectName?: string;
  access: SharedFolderAccess;
};

export type SharedFolderService = {
  protocol: ProtocolKey;
  enabled: boolean;
  guest?: boolean;
  mountHint: string;
};

export type FolderSnapshot = {
  name: string;
  createdAt: string;
};

export type SharedFolderView = {
  id: string;
  name: string;
  spaceId: string;
  relPath: string;
  dirKey: string;
  smbEnabled: boolean;
  nfsEnabled: boolean;
  guest: boolean;
  // advanced (fnOS/DSM): recycle bin / quota / encryption / Btrfs subvolume
  recycle?: boolean;
  quotaBytes?: number;
  encrypted?: boolean;
  subvolume?: boolean;
  createdAt: string;
  createdBy?: string;
  absPath: string;
  spaceName: string;
  fileSystem?: string;
  advancedOk?: boolean;
  permissions: SharedFolderPermission[];
  services: SharedFolderService[];
  snapshots?: FolderSnapshot[];
};

export type SharedFolderDeletePreview = {
  folderId: string;
  name: string;
  absPath: string;
  impact: string;
  confirmationId: string;
  requiresConfirmation: boolean;
};

export type SambaSyncReport = {
  users: { userId: string; username: string; inSamba: boolean }[];
  missing: number;
  note?: string;
};

export type ProtocolPreview = {
  kind: string;
  protocol: ProtocolKey;
  impact: string;
  risk: ProtocolRisk;
  riskLabel: string;
  requiresConfirmation: boolean;
  confirmationId?: string;
  rollbackId?: string;
};

export type ProtocolAuditEntry = {
  id: string;
  event: string;
  actor?: string;
  risk: ProtocolRisk;
  riskLabel: string;
  result: string;
  kind?: string;
  protocol?: ProtocolKey;
  share?: ProtocolShare;
  confirmationId?: string;
  rollbackId?: string;
  reverted: boolean;
  rollback?: string;
  time: string;
};

export type ProtocolConfirmResult = {
  protocol?: Protocol;
  share?: ProtocolShare;
  audit: ProtocolAuditEntry;
};

// --- iSCSI targets (LIO/targetcli) -----------------------------------------

export type ISCSITarget = {
  iqn: string;
  luns: number;
  acls: readonly string[];
  portals: readonly string[];
};

export type ISCSICaps = {
  available: boolean;
  backend: string;
  portal: string;
  note?: string;
};

export type ISCSIAuditEntry = {
  id: string;
  event: string;
  actor?: string;
  target?: string;
  action: string;
  result: string;
  time: string;
};

// --- virtual machines (libvirt/KVM) ----------------------------------------

export type VM = {
  name: string;
  uuid?: string;
  state: string; // running | shut off | paused | ...
  vcpus: number;
  memoryMB: number;
  autostart: boolean;
  persistent: boolean;
  title?: string;
};

export type VmHostCaps = {
  libvirtAvailable: boolean;
  kvmAvailable: boolean;
  version?: string;
  hypervisor: string;
  note?: string;
};

export type VmAuditEntry = {
  id: string;
  event: string;
  actor?: string;
  vm?: string;
  action: string;
  result: string;
  time: string;
};

// --- folder sync (device/directory synchronization) ------------------------

export type SyncDirection = 'mirror' | 'two-way';
export type SyncConflictPolicy = 'newer' | 'source' | 'target' | 'manual';

export type SyncRunStats = {
  copied: number;
  skipped: number;
  bytes: number;
  conflicts: number;
};

export type SyncPair = {
  id: string;
  name: string;
  source: string;
  target: string;
  direction: SyncDirection;
  conflictPolicy: SyncConflictPolicy;
  includes?: readonly string[];
  bandwidthLimit?: string;
  enabled: boolean;
  intervalHours?: number;
  retention?: SyncRetentionPolicy;
  state: string;
  progress: number;
  lastRun?: string;
  lastStats?: SyncRunStats;
  createdAt: string;
  createdBy?: string;
};

export type SyncConflict = {
  id: string;
  pairId: string;
  relPath: string;
  detail: string;
  resolved: boolean;
  resolution?: string;
  detectedAt: string;
  sourceSize?: number;
  targetSize?: number;
  sourceMtime?: string;
  targetMtime?: string;
};

export type SyncRetentionPolicy = {
  enabled: boolean;
  type: 'count' | 'days' | string;
  value: number;
};

export type SyncAuditEntry = {
  id: string;
  event: string;
  actor?: string;
  pairId?: string;
  result: string;
  time: string;
};

// --- security host scan (listening ports + firewall) -----------------------

export type ListeningPort = {
  protocol: string;
  address: string;
  port: number;
  process?: string;
  pid?: number;
  exposure: string; // 公开监听 | 局域网 | 仅本机
  risk: string; // low | medium | high
};

export type FirewallState = {
  backend: string; // nftables | ufw | iptables | none
  active: boolean;
  rules: number;
  summary: string;
  detail?: string[];
};

export type HostScanResult = {
  ports: ListeningPort[];
  firewall: FirewallState;
  openToAll: number;
  scannedAt: string;
};

export type StoragePool = {
  id?: string;
  name: string;
  type: string;
  used: number;
  total: string;
  health: string;
  temp: string;
  mountPath?: string;
};

export type StorageSpace = {
  id: string;
  name: string;
  poolId?: string;
  mode: 'basic' | 'linear' | 'raid0' | 'raid1' | 'raid5' | 'raid6' | 'raid10' | string;
  fileSystem: 'ext4' | 'btrfs' | 'zfs' | string;
  diskSlots: string[];
  mountPath: string;
  usedPercent: number;
  total: string;
  health: string;
  createdAt: string;
  createdBy?: string;
};

export type StorageDeletePreview = {
  spaceId: string;
  name: string;
  mode: string;
  total: string;
  diskSlots: string[];
  impact: string;
  risk: string;
  riskLabel: string;
  confirmationId: string;
  requiresConfirmation: boolean;
  expiresAt?: string;
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
  devicePath?: string;
  deviceType?: string;
  mediaType?: string;
  rotational?: boolean;
  systemDisk?: boolean;
  fileSystem?: string;
  mountPath?: string;
  standbyMinutes?: number;
  ssdCache?: boolean;
  cacheMode?: string;
  partitions?: DiskPartition[];
};

export type DiskPartition = {
  name: string;
  path: string;
  size: string;
  used?: string;
  total?: string;
  fileSystem?: string;
  mountPath?: string;
  system?: boolean;
};

export type StorageTask = {
  id: string;
  kind: 'smart-scan' | 'repair' | 'snapshot' | 'create-space' | 'delete-space' | 'remove-disk' | string;
  state: string;
  progress?: number;
  message?: string;
  targetSlot?: string;
  targetPool?: string;
};

export type ZfsSnapshot = {
  name: string;
  pool: string;
  usedBytes: number;
  used: string;
  createdAt: string;
};

export type ZfsPoolDetail = {
  pool: string;
  sizeBytes: number;
  allocBytes: number;
  freeBytes: number;
  capacityPct: number;
  fragmentation: number;
  dedupRatio: string;
  compressRatio: string;
  health: string;
};

export type ZfsSnapshotSchedule = {
  poolId: string;
  enabled: boolean;
  intervalHours: number;
  keep: number;
  lastRun?: string;
};

export type AccountUser = {
  id: string;
  username: string;
  displayName: string;
  role: 'admin' | 'user' | 'guest' | string;
  status: 'active' | 'disabled' | 'locked' | string;
  quotaBytes: number;
  groups: string[];
  homeSpaceId?: string;
  createdAt: string;
  updatedAt: string;
};

export type AccountGroup = {
  id: string;
  name: string;
  description: string;
  userIds: string[];
  createdAt: string;
  updatedAt: string;
};

export type AccountSpaceGrant = {
  id: string;
  subjectId: string;
  subjectType: 'user' | 'group' | string;
  spaceId: string;
  access: 'read' | 'read_write' | 'manage' | string;
  quotaBytes: number;
  createdAt: string;
  updatedAt: string;
};

export type AccountSummary = {
  users: AccountUser[];
  groups: AccountGroup[];
  grants: AccountSpaceGrant[];
};

export type CurrentUser = {
  id: string;
  username: string;
  displayName: string;
  role: 'admin' | 'user' | 'guest' | string;
  status: 'active' | 'disabled' | 'locked' | string;
  quotaBytes: number;
  groups: string[];
  permissions: string[];
  mfaEnabled?: boolean;
  csrfToken?: string;
};

export type AuthSession = {
  id: string;
  device: string;
  ipAddress: string;
  userAgent?: string;
  current: boolean;
  createdAt: string;
  lastSeenAt: string;
};

export type AuthAuditEntry = {
  id: string;
  time: string;
  actor: string;
  action: string;
  domain: string;
  result: string;
  risk: string;
  sourceIp: string;
};

export type LoginInput = {
  username: string;
  password: string;
  code?: string;
  rememberDevice?: boolean;
};

export type MfaSetup = {
  secret: string;
  otpauthUri: string;
};

export type ChangePasswordInput = {
  currentPassword: string;
  newPassword: string;
};

export type StewardSuggestionStatus = 'pending' | 'confirmed' | 'dismissed';

export type StewardSuggestion = {
  id?: string;
  title: string;
  detail: string;
  count: string;
  risk: RiskLevel;
  action: string;
  status?: StewardSuggestionStatus | string;
  updatedAt?: string;
};

// AgentPreset is a specialized agent role surfaced in the Agent Workbench.
export type AgentPreset = {
  id: string;
  name: string;
  description: string;
  icon?: string;
  systemPrompt?: string;
  toolDomains: string[];
  starters: string[];
};

// ToolCatalogEntry is one MCP tool's metadata for the capability panel.
export type ToolCatalogEntry = {
  name: string;
  domain: string;
  description: string;
  readOnly: boolean;
};

export type AssistantRole = 'user' | 'assistant' | 'system' | 'tool';

export type AssistantToolTrace = {
  name: string;
  summary?: string;
};

export type AssistantMessage = {
  id?: string;
  role: AssistantRole;
  text: string;
  createdAt?: string;
  citations?: Array<{ title: string; path?: string; url?: string }>;
  tools?: AssistantToolTrace[];
  pendingActionId?: string;
};

export type AssistantThread = {
  id: string;
  title?: string;
  messages: AssistantMessage[];
};

export type ThreadSummary = {
  id: string;
  title: string;
  updatedAt?: string;
  messageCount: number;
};

export type AssistantToolEvent = {
  phase: 'start' | 'done';
  name: string;
  args?: string;
  summary?: string;
  error?: string;
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
  result?: 'allowed' | 'confirmed' | 'dismissed' | 'rolled_back' | string;
  time?: string;
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

export type AiProviderKind = 'openai' | 'anthropic' | 'gemini';
export type AiProviderPurpose = 'chat' | 'embedding' | 'vision' | 'asr';

export type AiProvider = {
  id: string;
  name: string;
  kind: AiProviderKind;
  purpose: AiProviderPurpose;
  baseUrl: string;
  model: string;
  enabled: boolean;
  isDefault: boolean;
  hasKey: boolean;
  keyHint?: string;
  createdAt?: string;
};

export type AiProviderInput = {
  name?: string;
  kind?: AiProviderKind;
  purpose?: AiProviderPurpose;
  baseUrl?: string;
  apiKey?: string;
  model?: string;
  enabled?: boolean;
  isDefault?: boolean;
};

export type AiProviderTestResult = {
  ok: boolean;
  model: string;
  reply: string;
  latencyMs: number;
};

export type DownloadTask = {
  id: number | string;
  name: string;
  source: 'BT' | 'HTTP' | '磁力' | '订阅' | string;
  link?: string;
  category: string;
  size: string;
  progress: number;
  speed: string;
  status: '排队中' | '下载中' | '暂停' | '已完成' | '失败' | string;
  handling: string;
  archived: boolean;
  archiveRule?: {
    category?: string;
    targetPath?: string;
    tags?: string[];
    indexAfterMove?: boolean;
    scrapeMetadata?: boolean;
    verifyChecksum?: boolean;
  };
  filePath?: string;
  error?: string;
  speedLimitBytesPerSecond?: number;
};

export type DownloadTaskActionResult = {
  task: DownloadTask;
  message?: string;
  filePath?: string;
};

export type SpeedProfile = {
  id?: string;
  name?: string;
  down?: string;
  up?: string;
  downloadLimit?: string;
  uploadLimit?: string;
  note?: string;
  downloadLimitBytesPerSecond?: number;
  uploadLimitBytesPerSecond?: number;
  schedule?: string;
  active?: boolean;
};

export type DownloadQueueConfig = {
  maxConcurrentDownloads: number;
};

export type ComposeStack = {
  name: string;
  status: string;
  services: number;
  ports: string;
  volume: string;
  network: string;
  yaml?: string;
  createdAt?: string;
};

export type ComposeStackYaml = {
  name: string;
  yaml: string;
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

export type DockerImage = {
  id: string;
  repository: string;
  tag: string;
  size: string;
  created: string;
  iconUrl?: string;
};

export type DockerImageSearchResult = {
  name: string;
  description: string;
  stars: number;
  official: boolean;
  automated: boolean;
  iconUrl?: string;
};

export type DockerImagePullStatus = {
  id: string;
  image: string;
  status: 'queued' | 'running' | 'completed' | 'failed' | string;
  message: string;
  progress: number;
  downloaded: string;
  total: string;
  speed: string;
  error: string;
  startedAt: string;
  updatedAt: string;
};

export type DockerVolume = {
  name: string;
  driver: string;
  scope: string;
  mountpoint: string;
};

export type DockerNetwork = {
  id: string;
  name: string;
  driver: string;
  scope: string;
  subnet?: string;
  gateway?: string;
  internal?: boolean;
  attachable?: boolean;
  containers?: string[];
};

export type DockerExecResult = {
  exitCode: number;
  output: string;
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
  intervalHours?: number;
};

export type AppWebEntry = {
  container?: string;
  port: number;
  path?: string;
  display?: 'embed' | 'external';
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
  manifestId?: string;
  webEntry?: AppWebEntry | null;
  permissions?: string[];
};

export type AppAction = 'install' | 'update' | 'start' | 'stop' | 'uninstall';

export type AppConfigField = {
  key: string;
  label: string;
  type: string;
  default?: string;
  required?: boolean;
  secret?: boolean;
};

export type AppContainerSpec = {
  name: string;
  image: string;
  ports?: Array<{ container: number; host?: number; protocol?: string }>;
  resources?: { cpu?: number; memoryMb?: number };
  restartPolicy?: string;
};

export type AppManifest = {
  schemaVersion: string;
  id: string;
  name: string;
  version: string;
  category: string;
  description: string;
  author?: { name?: string; url?: string };
  iconUrl?: string;
  risk: string;
  source: string;
  containers: AppContainerSpec[];
  webEntry?: AppWebEntry | null;
  permissions?: string[];
  config?: AppConfigField[];
};

export type AppCatalogEntry = {
  manifest: AppManifest;
  origin: string;
  registry?: string;
};

export type AppActionPreview = {
  appId: string;
  action: AppAction;
  confirmationId: string;
  rollbackId: string;
  risk: string;
  impact: string;
  requiresConfirmation: boolean;
};

export type AppActionResult = {
  app: AppCenterApp;
  auditId: string;
  result: string;
  message: string;
};

export type AppAuditRecord = {
  id: string;
  appId: string;
  appName: string;
  action: string;
  actor: string;
  risk: string;
  result: string;
  message: string;
  rollbackId: string;
  rollbackable: boolean;
  createdAt: string;
  rolledBackAt?: string | null;
};

export type AppRegistry = {
  name: string;
  url: string;
  enabled: boolean;
  addedAt?: string;
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
  caption?: string;
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

export type MusicLibrarySettings = {
  paths: string[];
  autoScan: boolean;
  lastScanAt?: string;
  trackCount: number;
  status: string;
};

export type MusicTrack = {
  id: string;
  title: string;
  artist: string;
  album: string;
  trackNumber?: number;
  year?: string;
  codec: string;
  format: string;
  durationSeconds?: number;
  sizeBytes: number;
  size: string;
  modifiedAt: string;
  discoveredAt: string;
  fileName: string;
  path?: string;
  streamUrl: string;
  coverUrl?: string;
  lyricsUrl?: string;
  lyrics?: string;
  status: string;
};

export type MusicAlbum = {
  id: string;
  name: string;
  artist: string;
  count: number;
  coverUrl?: string;
};

export type MusicScanResult = {
  id: string;
  state: string;
  message: string;
  trackCount: number;
  scannedAt: string;
};

export type MediaScanResult = {
  id: string;
  state: string;
  message: string;
  itemCount: number;
  scannedAt: string;
};

export type VideoLibrary = {
  id: string;
  name: string;
  type: 'movie' | 'series' | 'mixed' | string;
  paths: string[];
  metadataLanguage: string;
  allowAdultContent: boolean;
  autoSubtitles: boolean;
  subtitleLanguage: string;
  count: number;
  status: string;
};

export type VideoLibrarySettings = {
  libraries: VideoLibrary[];
  status: string;
  itemCount: number;
  lastScan?: string;
};

export type DeleteVideoLibraryResult = {
  id: string;
  removedItems: number;
  removedTasks: number;
  settings: VideoLibrarySettings;
};

export type VideoItem = {
  id: string;
  libraryId: string;
  libraryName: string;
  title: string;
  originalTitle?: string;
  seriesTitle?: string;
  episodeTitle?: string;
  kind: 'movie' | 'episode' | string;
  year?: string;
  season?: number;
  episode?: number;
  container: string;
  codec: string;
  resolution: string;
  durationSeconds?: number;
  sizeBytes: number;
  size: string;
  modifiedAt: string;
  discoveredAt: string;
  fileName: string;
  path?: string;
  posterUrl: string;
  posterRemoteUrl?: string;
  backdropUrl?: string;
  backdropRemoteUrl?: string;
  streamUrl: string;
  subtitleUrl?: string;
  overview: string;
  aiOverview?: string;
  aiTranscript?: string;
  tagline?: string;
  contentRating?: string;
  releaseDate?: string;
  genres: string[];
  tags?: string[];
  directors?: string[];
  writers?: string[];
  actors?: string[];
  studios?: string[];
  countries?: string[];
  videoTracks?: VideoMediaTrack[];
  audioTracks?: VideoMediaTrack[];
  subtitleTracks?: VideoMediaTrack[];
  rating: string;
  metadataSource?: string;
  providerId?: string;
  scrapedAt?: string;
  progress: number;
  status: string;
};

export type VideoMediaTrack = {
  id: string;
  title: string;
  language?: string;
  codec?: string;
  channels?: string;
  default?: boolean;
};

export type VideoTask = {
  id: string;
  type: string;
  itemId?: string;
  title: string;
  status: 'queued' | 'running' | 'done' | 'failed' | string;
  message: string;
  progress: number;
  profile?: string;
  createdAt: string;
};

export type VideoScanResult = {
  id: string;
  state: string;
  message: string;
  itemCount: number;
  scannedAt: string;
};

export type LiveSource = {
  id: string;
  name: string;
  url: string;
  userAgent?: string;
  streamLimit?: number;
  channelCount: number;
  status: string;
};

export type LiveChannel = {
  id: string;
  sourceId: string;
  guideId?: string;
  name: string;
  group: string;
  logo?: string;
  url: string;
};

export type LiveGuideSource = {
  id: string;
  name: string;
  url: string;
  userAgent?: string;
  programCount: number;
  status: string;
  lastRefresh?: string;
};

export type LiveProgram = {
  id: string;
  guideId: string;
  channelId: string;
  channelName: string;
  title: string;
  overview?: string;
  categories?: string[];
  startAt: string;
  endAt: string;
  durationSeconds: number;
};

export type DvrSettings = {
  recordingPath: string;
  movieRecordingPath?: string;
  seriesRecordingPath?: string;
  prePaddingSeconds: number;
  postPaddingSeconds: number;
  maxConcurrentRecord: number;
  saveNfo: boolean;
  saveImages: boolean;
  postProcessCommand?: string;
};

export type RecordingTimer = {
  id: string;
  programId?: string;
  channelId: string;
  channelName: string;
  name: string;
  overview?: string;
  startAt: string;
  endAt: string;
  prePaddingSeconds: number;
  postPaddingSeconds: number;
  priority: number;
  status: string;
  targetPath?: string;
  createdAt: string;
};

export type RecordingItem = {
  id: string;
  timerId: string;
  programId?: string;
  title: string;
  channelId: string;
  status: string;
  path?: string;
  startedAt?: string;
  endedAt?: string;
  message?: string;
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
  ui?: {
    theme?: 'auto' | 'light' | 'dark' | string;
    locale?: 'zh-CN' | 'en-US' | string;
    windowRadius?: 'compact' | 'default' | 'rounded' | string;
    dockPosition?: 'bottom' | 'left' | 'right' | string;
    dockStyle?: 'floating' | 'side' | 'compact' | string;
    dockIconSize?: 'small' | 'default' | 'large' | string;
  };
  activity?: {
    enabled?: boolean;
    maxEntries?: number;
  };
  analysis?: {
    level?: 'off' | 'basic' | 'standard' | 'deep' | string;
  };
};

export type AiAnalysisDomain = 'media' | 'file' | 'video';
export type AiAnalysisState = 'pending' | 'analyzing' | 'done' | 'failed' | 'skipped';

export type AiAnalysisDomainStats = {
  domain: AiAnalysisDomain;
  total: number;
  pending: number;
  analyzing: number;
  done: number;
  failed: number;
  skipped: number;
  percent: number;
};

export type AiAnalysisStatus = {
  level: 'off' | 'basic' | 'standard' | 'deep' | string;
  paused: boolean;
  totalPercent: number;
  domains: AiAnalysisDomainStats[];
  hasChat: boolean;
  hasVision: boolean;
  hasEmbedding: boolean;
  hasAsr: boolean;
  indexEnabled: boolean;
  ffmpegAvailable: boolean;
  updatedAt?: string;
};

export type AiAnalysisResult = {
  summary?: string;
  caption?: string;
  tags?: string[];
  people?: string[];
  place?: string;
  device?: string;
  transcript?: string;
  techMeta?: Record<string, unknown>;
  embedded?: boolean;
};

export type AiAnalysisRecord = {
  key: string;
  domain: AiAnalysisDomain;
  title: string;
  sourcePath?: string;
  kind?: string;
  state: AiAnalysisState;
  level: string;
  progress: number;
  attempts: number;
  error?: string;
  result?: AiAnalysisResult;
  analyzedAt?: string;
  updatedAt?: string;
};

export type AiAnalysisRecordPage = {
  records: AiAnalysisRecord[];
  total: number;
  page: number;
  pageSize: number;
};

export type AiAnalysisReanalyzePayload =
  | { scope: 'item'; itemId: string }
  | { scope: 'domain'; domain: AiAnalysisDomain }
  | { scope: 'all' };

export type AiAnalysisBatchResult = {
  reset: number;
  status: AiAnalysisStatus;
};

export type FaceCluster = {
  label: string;
  count: number;
};

export type FaceTrainingStats = {
  totalSamples: number;
  confirmedSamples: number;
  namedPeople: number;
  models: number;
  activeModel?: string;
};

export type FaceModelVersion = {
  id: string;
  createdAt: string;
  sampleCount: number;
  source: string;
  note?: string;
  active: boolean;
};

export type FaceFrameworkStatus = {
  embedderName: string;
  embedderReady: boolean;
  trainerName: string;
  trainerReady: boolean;
  clusters: FaceCluster[];
  training: FaceTrainingStats;
  models: FaceModelVersion[];
};

export type FaceLabelResult = {
  confirmed: number;
  faces: FaceFrameworkStatus;
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

export type TunnelInfo = {
  backend: string; // wireguard | devstub | none
  up: boolean;
  interface: string;
  publicKey: string;
  listenPort: number;
  address: string;
  peers: number;
  note?: string;
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
  tunnel?: TunnelInfo;
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

/** Status values from the central task runtime (internal/tasks). */
export type TaskStatus = 'queued' | 'running' | 'succeeded' | 'failed' | 'canceled';

/** One recorded user activity (meaningful action or page/window visit). */
export type ActivityEntry = {
  id?: number;
  at?: string;
  actor?: string;
  type: 'action' | 'page';
  category?: string;
  action: string;
  target?: string;
  detail?: string;
  meta?: Record<string, string>;
};

/** A unit of background work from the central runtime (GET /api/v1/tasks). */
export type Task = {
  id: string;
  kind: string;
  status: TaskStatus;
  progress: number;
  message?: string;
  result?: unknown;
  error?: string;
  attempts: number;
  createdAt: string;
  startedAt?: string;
  finishedAt?: string;
  updatedAt: string;
};
