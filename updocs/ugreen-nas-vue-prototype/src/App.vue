<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref } from 'vue';
import {
  AppWindow,
  Bell,
  Bot,
  Boxes,
  BrainCircuit,
  BriefcaseBusiness,
  CalendarClock,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  CircleHelp,
  Cloud,
  CloudDownload,
  Container,
  Copy,
  Database,
  FileText,
  Film,
  Folder,
  Gauge,
  Globe2,
  HardDrive,
  Headphones,
  Image,
  Info,
  LayoutGrid,
  Link2,
  ListFilter,
  LockKeyhole,
  Monitor,
  MoreVertical,
  Music,
  Network,
  Pencil,
  PanelLeft,
  Power,
  Recycle,
  RefreshCcw,
  Search,
  Settings,
  Shield,
  SlidersHorizontal,
  SortAsc,
  SquareTerminal,
  Tags,
  Trash2,
  Upload,
  User,
  Users,
  Wifi,
  X,
} from 'lucide-vue-next';

type AppId =
  | 'ai'
  | 'app-center'
  | 'control-panel'
  | 'file-manager'
  | 'recycle'
  | 'help'
  | 'logs'
  | 'storage'
  | 'tasks'
  | 'text'
  | 'online-docs'
  | 'vm'
  | 'thunder'
  | 'downloads'
  | 'sync'
  | 'docker'
  | 'video'
  | 'music'
  | 'netdisk'
  | 'photos'
  | 'hermes'
  | 'antivirus'
  | 'security-center'
  | 'password-vault'
  | 'remote-access'
  | 'cloud-drive'
  | 'webdav'
  | 'backup-vault'
  | 'snapshot'
  | 'drive-sync'
  | 'qbittorrent'
  | 'transmission'
  | 'jellyfin'
  | 'plex'
  | 'photo-backup'
  | 'alist'
  | 'home-assistant'
  | 'terminal-tool'
  | 'note-station'
  | 'audiobooks'
  | 'mail-server';

type NasApp = {
  id: AppId;
  name: string;
  icon: unknown;
  color: string;
  category: string;
  status: '打开' | '安装' | '更新' | '内置';
  description: string;
};

type WindowState = {
  id: AppId;
  z: number;
  x: number;
  y: number;
  width: number;
  height: number;
  minimized?: boolean;
  maximized?: boolean;
};

type PanelItem = {
  id: string;
  name: string;
  icon: unknown;
  group: string;
};

type FileRow = {
  name: string;
  type: string;
  size: string;
  date: string;
  icon: unknown;
  tags: string[];
};

type NotificationRow = {
  id: number;
  title: string;
  detail: string;
  icon: unknown;
  tone: 'info' | 'warning' | 'success';
};

type ConfirmDialog = {
  title: string;
  detail: string;
  icon: unknown;
  tone: NotificationRow['tone'];
  confirmText: string;
  onConfirm: () => void;
};

type FileOperationDialog = {
  mode: 'copy' | 'move';
  target: string;
};

type AppInstallDialog = {
  app: NasApp;
  mode: 'install' | 'update';
};

type AppInstallTask = {
  app: NasApp;
  mode: 'install' | 'update';
  progress: number;
  stage: string;
};

type TaskRow = {
  name: string;
  detail: string;
  progress: number;
  tone: string;
  status: '运行中' | '已暂停' | '已取消';
};

type GenericSettingState = {
  autostart: boolean;
  notifications: boolean;
  sharedAccess: boolean;
  remoteAccess: boolean;
  cpuLimit: number;
  memoryLimit: number;
};

type AppDetailStatus = {
  appId: AppId;
  mode: 'update' | 'audit';
  title: string;
  detail: string;
  tone: NotificationRow['tone'];
  rows: string[];
};

type PanelConfig = {
  panel: string;
  title: string;
  detail: string;
  status: string;
  icon: unknown;
};

type FileSpace = 'personal' | 'shared' | 'users' | 'tags';
type FileFilter = '全部' | '文件夹' | '文档' | '媒体';

const desktopApps: NasApp[] = [
  { id: 'ai', name: 'UGREEN AI', icon: BrainCircuit, color: 'linear-gradient(145deg,#ffffff,#f5f7ff 48%,#9c71ff)', category: '系统管理', status: '打开', description: '私人 AI 助手、模型管理与 AI NAS 能力入口' },
  { id: 'app-center', name: '应用中心', icon: LayoutGrid, color: 'linear-gradient(145deg,#fff2c8 0%,#ffbf27 28%,#ff5b48 56%,#645bff)', category: '系统管理', status: '打开', description: '41 个应用的安装、更新、打开和权限管理' },
  { id: 'control-panel', name: '控制面板', icon: SlidersHorizontal, color: 'linear-gradient(145deg,#f6f6f7,#838996)', category: '系统管理', status: '内置', description: '用户、服务、网络、安全、设备与系统更新' },
  { id: 'file-manager', name: '文件管理', icon: Folder, color: 'linear-gradient(180deg,#fff5a9 0%,#ffd84e 45%,#f5bf2c 100%)', category: '系统管理', status: '内置', description: '个人、共享、用户文件夹与标签管理' },
  { id: 'recycle', name: '回收站', icon: Recycle, color: 'linear-gradient(145deg,#eff9ff,#a6d7ff 52%,#79aef8)', category: '系统管理', status: '内置', description: '恢复或清理删除文件' },
  { id: 'help', name: '帮助中心', icon: CircleHelp, color: 'linear-gradient(145deg,#7ad7ff,#3267f4)', category: '实用工具', status: '打开', description: '教程、远程访问说明、技术支持和设备日志上传' },
  { id: 'logs', name: '日志中心', icon: FileText, color: 'linear-gradient(145deg,#ffffff,#eef3fb 60%,#ccd7e8)', category: '系统管理', status: '打开', description: '系统日志、登录日志、操作日志和安全日志' },
  { id: 'storage', name: '存储管理', icon: HardDrive, color: 'linear-gradient(145deg,#4b4d55,#15171d)', category: '系统管理', status: '打开', description: '存储池、硬盘、RAID、SMART、快照和修复' },
  { id: 'tasks', name: '任务管理器', icon: CalendarClock, color: 'linear-gradient(145deg,#23354d,#0f1723 62%,#4f9cff)', category: '系统管理', status: '内置', description: '上传、下载、同步、备份、扫描、转码任务' },
  { id: 'text', name: '文本编辑', icon: FileText, color: 'linear-gradient(145deg,#ffffff,#eef2f8 60%,#d8e0ec)', category: '实用工具', status: '打开', description: '轻量编辑 NAS 文本文件' },
  { id: 'downloads', name: '下载中心', icon: CloudDownload, color: 'linear-gradient(145deg,#f8fff7,#3ac54f)', category: '下载', status: '打开', description: 'HTTP、BT、磁力、迅雷、限速和归档' },
  { id: 'sync', name: '同步与备份', icon: RefreshCcw, color: 'linear-gradient(145deg,#71d464,#30a845)', category: '备份', status: '打开', description: '同步、备份、恢复、版本保留和冲突处理' },
  { id: 'docker', name: 'Docker', icon: Container, color: 'linear-gradient(145deg,#ffffff,#e9f7ff 58%,#009ee8)', category: '实用工具', status: '打开', description: '容器、镜像、网络、卷、Compose 和日志' },
  { id: 'video', name: '影视中心', icon: Film, color: 'linear-gradient(145deg,#ffe1ea,#d8698a 58%,#3b4048)', category: '娱乐', status: '打开', description: '媒体库、刮削、字幕、转码、直播和 DVR' },
  { id: 'music', name: '音乐', icon: Music, color: 'linear-gradient(145deg,#ff7467,#ef3e3e)', category: '娱乐', status: '打开', description: '音乐库、专辑、歌词和播放队列' },
  { id: 'netdisk', name: '网盘工具', icon: Cloud, color: 'linear-gradient(145deg,#ffffff,#e8f4ff 58%,#3e88ff)', category: '实用工具', status: '打开', description: '网盘上传下载、同步状态和任务概览' },
  { id: 'photos', name: '相册', icon: Image, color: 'linear-gradient(145deg,#5db9ff 0%,#6bb8ff 50%,#ffd465 51%,#f6c04c)', category: '娱乐', status: '打开', description: '时间线、人物、地点、回忆、共享相册和备份' },
  { id: 'hermes', name: 'Hermes Agent', icon: Bot, color: 'linear-gradient(145deg,#ffffff,#111111)', category: '系统管理', status: '打开', description: '智能代理、自动化任务和工具调用审计' },
];

const appCatalog: NasApp[] = [
  ...desktopApps,
  { id: 'online-docs', name: '在线文档', icon: FileText, color: 'linear-gradient(145deg,#f5fbff 0%,#2f80ed 62%,#6eb8ff)', category: '实用工具', status: '安装', description: '多人协作、文档编辑、表格和团队知识库' },
  { id: 'vm', name: '虚拟机', icon: Monitor, color: 'linear-gradient(145deg,#e9f1ff 0%,#3b7ee5 58%,#1d57bd)', category: '实用工具', status: '安装', description: '虚拟机创建、快照、镜像、网络和远程控制台' },
  { id: 'thunder', name: '迅雷', icon: CloudDownload, color: 'linear-gradient(145deg,#ffffff 0%,#2e7dff 58%,#4ca3ff)', category: '下载', status: '安装', description: '迅雷下载、会员加速、离线任务和下载队列' },
  { id: 'antivirus', name: '杀毒套件', icon: Shield, color: 'linear-gradient(145deg,#f5fff8 0%,#2fc870 58%,#0a8f4d)', category: '安全', status: '安装', description: '病毒扫描、隔离区、定时查杀和威胁报告' },
  { id: 'security-center', name: '安全管家', icon: LockKeyhole, color: 'linear-gradient(145deg,#f7fbff 0%,#2d71ff 58%,#123d9b)', category: '安全', status: '打开', description: '登录保护、风险检测、防火墙和 OTP 安全策略' },
  { id: 'password-vault', name: '密码保险箱', icon: LockKeyhole, color: 'linear-gradient(145deg,#fff9e8 0%,#ffc94a 56%,#d48600)', category: '安全', status: '安装', description: '凭据保存、共享密钥、自动填充和访问审计' },
  { id: 'remote-access', name: '远程访问', icon: Globe2, color: 'linear-gradient(145deg,#eaf8ff 0%,#30a6ff 60%,#1764d4)', category: '实用工具', status: '打开', description: 'UGREENlink、DDNS、反向代理和外网访问诊断' },
  { id: 'cloud-drive', name: 'CloudDrive', icon: Cloud, color: 'linear-gradient(145deg,#ffffff 0%,#66b7ff 58%,#2d70df)', category: '实用工具', status: '安装', description: '挂载云盘、缓存策略、离线同步和容量监控' },
  { id: 'webdav', name: 'WebDAV 服务', icon: Network, color: 'linear-gradient(145deg,#f3fffb 0%,#39cdb2 58%,#168373)', category: '实用工具', status: '打开', description: 'WebDAV 共享、访问令牌、客户端连接和日志' },
  { id: 'backup-vault', name: '备份保险箱', icon: HardDrive, color: 'linear-gradient(145deg,#f7fff2 0%,#7bd954 58%,#329020)', category: '备份', status: '安装', description: '不可变备份、快照锁定、恢复演练和保留策略' },
  { id: 'snapshot', name: '快照管理', icon: Database, color: 'linear-gradient(145deg,#f8fbff 0%,#7894ff 58%,#3953c7)', category: '备份', status: '打开', description: '共享文件夹快照、计划、回滚和空间回收' },
  { id: 'drive-sync', name: '网盘同步', icon: RefreshCcw, color: 'linear-gradient(145deg,#f4fff8 0%,#51d377 58%,#1d9145)', category: '备份', status: '安装', description: '多云同步、冲突处理、版本保留和带宽限制' },
  { id: 'qbittorrent', name: 'qBittorrent', icon: CloudDownload, color: 'linear-gradient(145deg,#eef8ff 0%,#4aa3ff 60%,#1b5fb9)', category: '下载', status: '安装', description: 'BT 下载、RSS 订阅、限速、标签和自动归档' },
  { id: 'transmission', name: 'Transmission', icon: CloudDownload, color: 'linear-gradient(145deg,#fff5f1 0%,#ef6b45 58%,#bd2b1f)', category: '下载', status: '安装', description: '轻量 BT 下载、远程控制、队列和做种规则' },
  { id: 'jellyfin', name: 'Jellyfin', icon: Film, color: 'linear-gradient(145deg,#f9f2ff 0%,#9b5cff 58%,#4a1bb6)', category: '娱乐', status: '安装', description: '个人媒体服务器、刮削、转码、字幕和客户端播放' },
  { id: 'plex', name: 'Plex', icon: Film, color: 'linear-gradient(145deg,#fff8df 0%,#f4c542 55%,#252525)', category: '娱乐', status: '安装', description: '媒体库、家庭共享、远程播放和元数据整理' },
  { id: 'photo-backup', name: '手机备份', icon: Image, color: 'linear-gradient(145deg,#e8f7ff 0%,#5ebcff 55%,#ffd166)', category: '娱乐', status: '打开', description: '手机照片自动备份、相册归档和重复照片检测' },
  { id: 'alist', name: 'AList', icon: Cloud, color: 'linear-gradient(145deg,#f7fbff 0%,#5d83ff 58%,#2436a6)', category: '实用工具', status: '安装', description: '统一聚合网盘、目录索引、分享页和访问控制' },
  { id: 'home-assistant', name: 'Home Assistant', icon: Gauge, color: 'linear-gradient(145deg,#eaffff 0%,#3cc8dc 58%,#1580a4)', category: '实用工具', status: '安装', description: '智能家居自动化、设备集成、仪表盘和场景联动' },
  { id: 'terminal-tool', name: '终端工具', icon: SquareTerminal, color: 'linear-gradient(145deg,#262b35 0%,#0e1118 58%,#45e68a)', category: '实用工具', status: '打开', description: 'Web 终端、SSH 会话、命令审计和快捷脚本' },
  { id: 'note-station', name: '笔记', icon: FileText, color: 'linear-gradient(145deg,#fffdf2 0%,#ffdf6e 55%,#d7a600)', category: '实用工具', status: '安装', description: 'Markdown 笔记、附件、标签、全文搜索和共享' },
  { id: 'audiobooks', name: '有声书', icon: Headphones, color: 'linear-gradient(145deg,#fff0f7 0%,#ff7eb6 55%,#b83278)', category: '娱乐', status: '安装', description: '有声书库、章节进度、倍速播放和多端同步' },
  { id: 'mail-server', name: '邮件服务器', icon: BriefcaseBusiness, color: 'linear-gradient(145deg,#f5f8ff 0%,#6b8cff 58%,#233e9a)', category: '实用工具', status: '安装', description: '邮件收发、域名配置、反垃圾策略和账户管理' },
];

const appMap = new Map(appCatalog.map((app) => [app.id, app]));
const windows = ref<WindowState[]>([
  { id: 'control-panel', z: 2, x: 238, y: 60, width: 964, height: 642 },
]);
const zSeed = ref(3);
const activePanel = ref('overview');
const activeCategory = ref('全部应用');
const activeFileSpace = ref<FileSpace>('personal');
const activeDesktopView = ref<'desktop' | 'library' | 'widgets'>('desktop');
const peekDesktop = ref(false);
const showNotifications = ref(false);
const hasUnreadNotifications = ref(true);
const showGlobalSearch = ref(false);
const showContextMenu = ref(false);
const showUserMenu = ref(false);
const showSystemPanel = ref(false);
const showPowerDialog = ref(false);
const showLockScreen = ref(false);
const showUploadDialog = ref(false);
const showNewFolderDialog = ref(false);
const showHelpDialog = ref(false);
const helpContextApp = ref<AppId | null>(null);
const showWallpaperPanel = ref(false);
const showFileFilter = ref(false);
const showFileMoreMenu = ref(false);
const showAppSettings = ref(false);
const showTagEditor = ref(false);
const showRenameDialog = ref(false);
const showShareDialog = ref(false);
const isFileDragOver = ref(false);
const appDetail = ref<NasApp | null>(null);
const appDetailStatus = ref<AppDetailStatus | null>(null);
const previewFile = ref<FileRow | null>(null);
const confirmDialog = ref<ConfirmDialog | null>(null);
const renameFile = ref<FileRow | null>(null);
const tagEditorFile = ref<FileRow | null>(null);
const fileOperationDialog = ref<FileOperationDialog | null>(null);
const shareFile = ref<FileRow | null>(null);
const appInstallDialog = ref<AppInstallDialog | null>(null);
const appInstallTask = ref<AppInstallTask | null>(null);
const panelConfig = ref<PanelConfig | null>(null);
const appCenterSettings = reactive({
  autoCheck: true,
  nightInstall: true,
  permissionConfirm: true,
  communityApps: false,
  betaChannel: false,
  updateWindow: 2,
});
const query = ref('');
const globalQuery = ref('');
const fileQuery = ref('');
const panelQuery = ref('');
const fileMenuOpen = ref('');
const newFolderName = ref('新建文件夹');
const renameValue = ref('');
const tagDraft = ref('');
const operationDestination = ref<FileSpace>('shared');
const fileHistory = ref<string[]>(['个人文件夹']);
const fileHistoryIndex = ref(0);
const filePathInput = ref('个人文件夹');
const unlockCode = ref('');
const wallpaperTheme = ref<'space' | 'aurora' | 'matrix'>('space');
const activePanelTab = ref('概览');
const fileFilter = ref<FileFilter>('全部');
const fileSortMode = ref<'名称' | '类型' | '时间'>('名称');
const selectedFiles = ref<string[]>([]);
const selectedDesktopApp = ref<AppId | ''>('');
const appContextMenu = ref<{ app: NasApp | null; x: number; y: number }>({ app: null, x: 0, y: 0 });
const toastNotification = ref<NotificationRow | null>(null);
const installedApps = reactive<Record<string, boolean>>({
  'app-center': true,
  'control-panel': true,
  'file-manager': true,
  recycle: true,
  tasks: true,
  music: true,
  video: true,
  photos: true,
  docker: true,
  sync: true,
  downloads: true,
});
const activeGenericTab = reactive<Record<string, string>>({});
const genericSettings = reactive<Record<string, GenericSettingState>>({});
const contextMenu = reactive({ x: 0, y: 0 });
const dragging = ref<{ id: AppId; dx: number; dy: number } | null>(null);
const resizing = ref<{
  id: AppId;
  startX: number;
  startY: number;
  startWidth: number;
  startHeight: number;
} | null>(null);

const appCategories = ['全部应用', '系统管理', '实用工具', '娱乐', '下载', '安全', '备份', '已安装'];
const appCenterOrder = ['online-docs', 'video', 'music', 'thunder', 'vm', 'photos', 'downloads', 'file-manager'];
const fileFilterOptions: FileFilter[] = ['全部', '文件夹', '文档', '媒体'];
const operationDestinations: FileSpace[] = ['personal', 'shared', 'users'];

const panelItems: PanelItem[] = [
  { id: 'users', name: '用户管理', icon: Users, group: '连接与访问' },
  { id: 'file-service', name: '文件服务', icon: Folder, group: '连接与访问' },
  { id: 'device-link', name: '设备连接', icon: Monitor, group: '连接与访问' },
  { id: 'ldap', name: '域/LDAP', icon: BriefcaseBusiness, group: '连接与访问' },
  { id: 'terminal', name: '终端机', icon: SquareTerminal, group: '连接与访问' },
  { id: 'power', name: '硬件与电源', icon: HardDrive, group: '通用设置' },
  { id: 'time', name: '时间和语言', icon: CalendarClock, group: '通用设置' },
  { id: 'network', name: '网络设置', icon: Wifi, group: '通用设置' },
  { id: 'security', name: '安全性', icon: Shield, group: '通用设置' },
  { id: 'index', name: '索引服务', icon: Database, group: '通用设置' },
  { id: 'about', name: '关于本机', icon: Info, group: '系统服务' },
  { id: 'updates', name: '更新与还原', icon: RefreshCcw, group: '系统服务' },
];

const files: FileRow[] = [
  { name: 'Music', type: '文件夹', size: '-', date: '2026-06-22 12:50', icon: Folder, tags: ['媒体库'] },
  { name: 'Photos', type: '文件夹', size: '-', date: '2026-06-22 12:51', icon: Folder, tags: ['相册'] },
  { name: '家庭影像索引.md', type: 'Markdown', size: '28 KB', date: '2026-06-21 19:30', icon: FileText, tags: ['AI 索引'] },
  { name: 'Docker-compose-backup.yml', type: 'YAML', size: '6 KB', date: '2026-06-18 09:12', icon: Container, tags: ['备份'] },
  { name: 'UGREENlink-远程访问.pdf', type: 'PDF', size: '2.4 MB', date: '2026-06-17 16:22', icon: Globe2, tags: ['共享'] },
];

const folderFiles: Record<string, FileRow[]> = {
  Music: [
    { name: 'HiFi 收藏', type: '文件夹', size: '-', date: '2026-06-20 22:10', icon: Folder, tags: ['媒体库'] },
    { name: '家庭歌单.m3u', type: '文档', size: '4 KB', date: '2026-06-20 22:08', icon: Music, tags: ['媒体库'] },
    { name: '歌词匹配报告.txt', type: '文档', size: '18 KB', date: '2026-06-19 10:18', icon: FileText, tags: ['AI 索引'] },
  ],
  Photos: [
    { name: '2026 家庭旅行', type: '文件夹', size: '-', date: '2026-06-21 20:16', icon: Folder, tags: ['相册'] },
    { name: '手机自动备份', type: '文件夹', size: '-', date: '2026-06-21 08:45', icon: Folder, tags: ['相册'] },
    { name: '人物聚类报告.pdf', type: 'PDF', size: '780 KB', date: '2026-06-20 13:32', icon: FileText, tags: ['AI 索引'] },
  ],
  'HiFi 收藏': [
    { name: 'DSD Sample.flac', type: '媒体', size: '84 MB', date: '2026-06-18 22:10', icon: Music, tags: ['媒体库'] },
    { name: '封面整理.md', type: 'Markdown', size: '12 KB', date: '2026-06-18 21:40', icon: FileText, tags: ['AI 索引'] },
  ],
  '2026 家庭旅行': [
    { name: 'IMG_0422.heic', type: '媒体', size: '3.8 MB', date: '2026-06-21 20:16', icon: Image, tags: ['相册'] },
    { name: '旅行相册说明.md', type: 'Markdown', size: '6 KB', date: '2026-06-21 20:11', icon: FileText, tags: ['共享'] },
  ],
  手机自动备份: [
    { name: 'iPhone 15 Pro', type: '文件夹', size: '-', date: '2026-06-21 08:45', icon: Folder, tags: ['相册'] },
    { name: '备份状态.txt', type: '文档', size: '2 KB', date: '2026-06-21 08:45', icon: FileText, tags: ['备份'] },
  ],
};

const tasks = reactive<TaskRow[]>([
  { name: '文件上传', detail: 'Photos / 128 个文件', progress: 72, tone: 'blue', status: '运行中' },
  { name: '影视中心转码', detail: 'HEVC 转 H.264', progress: 43, tone: 'purple', status: '运行中' },
  { name: '硬盘健康检查', detail: 'Disk 02 SMART extended', progress: 18, tone: 'green', status: '运行中' },
  { name: '同步与备份', detail: 'MacBook Pro -> NAS', progress: 91, tone: 'cyan', status: '运行中' },
]);

const notifications = ref<NotificationRow[]>([
  { id: 1, title: '安全模块', detail: '发现 1 个远程登录风险，建议开启 OTP。', icon: Shield, tone: 'warning' },
  { id: 2, title: '存储模块', detail: 'Disk 02 健康检查正在执行。', icon: HardDrive, tone: 'info' },
  { id: 3, title: '系统模块', detail: '当前版本已是最新版本。', icon: RefreshCcw, tone: 'success' },
]);

function sortAppCenterApps(apps: NasApp[]) {
  return [...apps].sort((a, b) => {
    const aIndex = appCenterOrder.indexOf(a.id);
    const bIndex = appCenterOrder.indexOf(b.id);
    if (aIndex === -1 && bIndex === -1) return 0;
    if (aIndex === -1) return 1;
    if (bIndex === -1) return -1;
    return aIndex - bIndex;
  });
}

function appMatchesQuery(app: NasApp, value: string) {
  const q = value.trim().toLowerCase();
  return !q
    || app.name.toLowerCase().includes(q)
    || app.category.toLowerCase().includes(q)
    || app.description.toLowerCase().includes(q);
}

const appCenterApps = computed(() => {
  const base = activeCategory.value === '已安装'
    ? appCatalog.filter((app) => installedApps[app.id])
    : activeCategory.value === '全部应用'
      ? appCatalog
      : appCatalog.filter((app) => app.category === activeCategory.value);
  const sorted = activeCategory.value === '全部应用' ? sortAppCenterApps(base) : base;
  return sorted.filter((app) => appMatchesQuery(app, query.value));
});
const libraryApps = computed(() => sortAppCenterApps(appCatalog).filter((app) => appMatchesQuery(app, query.value)));
const searchApps = computed(() => sortAppCenterApps(appCatalog).filter((app) => appMatchesQuery(app, globalQuery.value)).slice(0, 4));
const displayedAppCount = computed(() => (activeCategory.value === '全部应用' && !query.value.trim() ? appCatalog.length : appCenterApps.value.length));

const visibleWindows = computed(() => (
  peekDesktop.value ? [] : [...windows.value].filter((win) => !win.minimized).sort((a, b) => a.z - b.z)
));
const minimizedWindows = computed(() => windows.value.filter((win) => win.minimized));
const activeWindowZ = computed(() => Math.max(...visibleWindows.value.map((win) => win.z), 0));
const currentFilePath = computed(() => fileHistory.value[fileHistoryIndex.value] || fileSpaceName(activeFileSpace.value));
const currentFileFolder = computed(() => {
  const segments = currentFilePath.value.split('/').filter(Boolean);
  return segments[segments.length - 1] ?? fileSpaceName(activeFileSpace.value);
});
const canGoBackFilePath = computed(() => fileHistoryIndex.value > 0);
const canGoForwardFilePath = computed(() => fileHistoryIndex.value < fileHistory.value.length - 1);
const filteredFiles = computed(() => {
  const q = fileQuery.value.trim().toLowerCase();
  const isRootPath = currentFilePath.value === fileSpaceName(activeFileSpace.value);
  const sourceDefault = activeFileSpace.value === 'personal' && isRootPath && !q && fileFilter.value === '全部';
  const folderRows = isRootPath ? files : folderFiles[currentFileFolder.value] ?? [];
  const base = sourceDefault ? folderRows.filter((file) => ['Music', 'Photos'].includes(file.name)) : folderRows;
  const filtered = base.filter((file) => {
    const matchesQuery = !q || file.name.toLowerCase().includes(q) || file.type.toLowerCase().includes(q);
    const matchesFilter = fileFilter.value === '全部'
      || file.type === fileFilter.value
      || (fileFilter.value === '文档' && ['Markdown', 'YAML', 'PDF'].includes(file.type))
      || (fileFilter.value === '媒体' && file.tags.some((tag) => ['媒体库', '相册'].includes(tag)));
    return matchesQuery && matchesFilter;
  });
  if (sourceDefault) return filtered;
  return [...filtered].sort((a, b) => {
    if (fileSortMode.value === '时间') return b.date.localeCompare(a.date);
    if (fileSortMode.value === '类型') return a.type.localeCompare(b.type);
    return a.name.localeCompare(b.name);
  });
});
const allFileRows = computed(() => [...files, ...Object.values(folderFiles).flat()]);
const selectedFileRows = computed(() => allFileRows.value.filter((file) => selectedFiles.value.includes(file.name)));
const firstSearchFile = computed(() => filteredFiles.value[0]);
const searchFiles = computed(() => {
  const q = globalQuery.value.trim().toLowerCase();
  return files
    .filter((file) => !q || file.name.toLowerCase().includes(q) || file.type.toLowerCase().includes(q))
    .slice(0, 4);
});
const searchSettings = computed(() => {
  const q = globalQuery.value.trim().toLowerCase();
  return panelItems
    .filter((item) => !q || item.name.toLowerCase().includes(q) || item.group.toLowerCase().includes(q))
    .slice(0, 4);
});
const filteredPanelItems = computed(() => {
  const q = panelQuery.value.trim().toLowerCase();
  if (!q) return panelItems;
  return panelItems.filter((item) => item.name.toLowerCase().includes(q) || item.group.toLowerCase().includes(q));
});

function pushNotification(title: string, detail: string, icon: unknown = Info, tone: NotificationRow['tone'] = 'info') {
  const item = { id: Date.now(), title, detail, icon, tone };
  notifications.value.unshift(item);
  toastNotification.value = item;
  hasUnreadNotifications.value = true;
}

function toggleNotifications() {
  showNotifications.value = !showNotifications.value;
  if (showNotifications.value) hasUnreadNotifications.value = false;
}

function openApp(id: AppId) {
  peekDesktop.value = false;
  activeDesktopView.value = 'desktop';
  const existing = windows.value.find((win) => win.id === id);
  if (existing) {
    existing.minimized = false;
    existing.z = zSeed.value++;
    return;
  }
  const count = windows.value.length;
  const sourceSizedWindow = id === 'app-center' || id === 'file-manager';
  const windowPreset = id === 'app-center'
    ? { x: 278, y: 100, width: 964, height: 642 }
    : id === 'file-manager'
      ? { x: 170, y: 60, width: 1100, height: 642 }
      : { x: 238 + (count % 4) * 18, y: 60 + (count % 5) * 16, width: 964, height: 642 };
  windows.value.push({
    id,
    z: zSeed.value++,
    x: windowPreset.x,
    y: windowPreset.y,
    width: windowPreset.width,
    height: sourceSizedWindow ? windowPreset.height : 642,
  });
  activeGenericTab[id] ||= '概览';
  showContextMenu.value = false;
  showGlobalSearch.value = false;
}

function closeWindow(id: AppId) {
  windows.value = windows.value.filter((win) => win.id !== id);
  if (toastNotification.value?.title === '窗口管理') toastNotification.value = null;
}

function focusWindow(id: AppId) {
  const win = windows.value.find((item) => item.id === id);
  if (win) win.z = zSeed.value++;
}

function minimizeWindow(id: AppId) {
  const win = windows.value.find((item) => item.id === id);
  if (win) {
    win.minimized = true;
    pushNotification('窗口管理', `${appName(id)} 已最小化。`, AppWindow, 'info');
  }
}

function minimizeAllWindows() {
  windows.value.forEach((win) => {
    win.minimized = true;
  });
  showContextMenu.value = false;
  pushNotification('窗口管理', '所有窗口已最小化。', Monitor, 'info');
}

function restoreAllWindows() {
  windows.value.forEach((win) => {
    win.minimized = false;
    win.z = zSeed.value++;
  });
  pushNotification('窗口管理', '已恢复所有窗口。', Monitor, 'success');
}

function toggleDesktopFromTopbar() {
  activeDesktopView.value = 'desktop';
  closeTransientLayers();
  const hasVisibleWindow = windows.value.some((win) => !win.minimized);
  peekDesktop.value = hasVisibleWindow ? !peekDesktop.value : false;
  if (!peekDesktop.value) {
    windows.value.filter((win) => !win.minimized).forEach((win) => {
      win.z = zSeed.value++;
    });
  }
}

function snapWindow(id: AppId, side: 'left' | 'right') {
  const win = windows.value.find((item) => item.id === id);
  if (!win) return;
  win.maximized = false;
  win.x = side === 'left' ? 8 : Math.floor(window.innerWidth / 2) + 4;
  win.y = 8;
  win.width = Math.floor(window.innerWidth / 2) - 18;
  win.height = window.innerHeight - 68;
  win.z = zSeed.value++;
  pushNotification('窗口管理', `${appName(id)} 已贴靠到${side === 'left' ? '左侧' : '右侧'}。`, AppWindow, 'info');
}

function toggleMaximize(id: AppId) {
  const win = windows.value.find((item) => item.id === id);
  if (!win) return;
  win.maximized = !win.maximized;
  win.z = zSeed.value++;
}

function appName(id: AppId) {
  return appMap.get(id)?.name ?? id;
}

function appIcon(id: AppId) {
  return appMap.get(id)?.icon ?? AppWindow;
}

function windowStyle(win: WindowState) {
  if (win.maximized) {
    return {
      zIndex: win.z,
      left: '8px',
      top: '8px',
      width: 'calc(100vw - 16px)',
      height: 'calc(100vh - 62px)',
    };
  }
  return {
    zIndex: win.z,
    left: `${win.x}px`,
    top: `${win.y}px`,
    width: `min(${win.width}px, calc(100vw - 24px))`,
    height: `min(${win.height}px, calc(100vh - 72px))`,
  };
}

function beginDrag(event: PointerEvent, win: WindowState) {
  if (win.maximized) return;
  const target = event.target as HTMLElement;
  if (target.closest('button')) return;
  dragging.value = { id: win.id, dx: event.clientX - win.x, dy: event.clientY - win.y };
  window.addEventListener('pointermove', onDrag);
  window.addEventListener('pointerup', stopDrag);
}

function onDrag(event: PointerEvent) {
  if (!dragging.value) return;
  const win = windows.value.find((item) => item.id === dragging.value?.id);
  if (!win) return;
  win.x = Math.max(0, Math.min(window.innerWidth - 180, event.clientX - dragging.value.dx));
  win.y = Math.max(0, Math.min(window.innerHeight - 120, event.clientY - dragging.value.dy));
}

function stopDrag() {
  dragging.value = null;
  window.removeEventListener('pointermove', onDrag);
  window.removeEventListener('pointerup', stopDrag);
}

function beginResize(event: PointerEvent, win: WindowState) {
  if (win.maximized) return;
  event.preventDefault();
  event.stopPropagation();
  resizing.value = {
    id: win.id,
    startX: event.clientX,
    startY: event.clientY,
    startWidth: win.width,
    startHeight: win.height,
  };
  focusWindow(win.id);
  window.addEventListener('pointermove', onResize);
  window.addEventListener('pointerup', stopResize);
}

function onResize(event: PointerEvent) {
  if (!resizing.value) return;
  const win = windows.value.find((item) => item.id === resizing.value?.id);
  if (!win) return;
  const nextWidth = resizing.value.startWidth + event.clientX - resizing.value.startX;
  const nextHeight = resizing.value.startHeight + event.clientY - resizing.value.startY;
  win.width = Math.max(620, Math.min(window.innerWidth - win.x - 18, nextWidth));
  win.height = Math.max(390, Math.min(window.innerHeight - win.y - 58, nextHeight));
}

function stopResize() {
  resizing.value = null;
  window.removeEventListener('pointermove', onResize);
  window.removeEventListener('pointerup', stopResize);
}

function toggleFile(name: string) {
  selectedFiles.value = selectedFiles.value.includes(name)
    ? selectedFiles.value.filter((item) => item !== name)
    : [...selectedFiles.value, name];
}

function installOrOpen(app: NasApp) {
  if (!installedApps[app.id]) {
    appInstallDialog.value = { app, mode: 'install' };
    appDetail.value = null;
    showContextMenu.value = false;
    appContextMenu.value.app = null;
    return;
  }
  if (app.status === '更新') {
    appInstallDialog.value = { app, mode: 'update' };
    appDetail.value = null;
    showContextMenu.value = false;
    appContextMenu.value.app = null;
    return;
  }
  openApp(app.id);
}

function confirmAppInstall() {
  if (!appInstallDialog.value) return;
  const { app, mode } = appInstallDialog.value;
  appInstallTask.value = {
    app,
    mode,
    progress: mode === 'install' ? 18 : 36,
    stage: mode === 'install' ? '下载应用包' : '下载更新包',
  };
  appInstallDialog.value = null;
  pushNotification('应用中心', `${app.name} 已加入${mode === 'install' ? '安装' : '更新'}队列。`, app.icon, 'info');
}

function advanceAppInstallTask() {
  if (!appInstallTask.value) return;
  const task = appInstallTask.value;
  if (task.progress < 54) {
    task.progress = 58;
    task.stage = '校验签名与权限';
    return;
  }
  if (task.progress < 86) {
    task.progress = 88;
    task.stage = task.mode === 'install' ? '写入应用目录' : '替换应用版本';
    return;
  }
  installedApps[task.app.id] = true;
  pushNotification('应用中心', `${task.app.name} 已${task.mode === 'install' ? '安装完成' : '更新完成'}。`, task.app.icon, 'success');
  appInstallTask.value = null;
}

function cancelAppInstallTask() {
  if (!appInstallTask.value) return;
  pushNotification('应用中心', `${appInstallTask.value.app.name} 的${appInstallTask.value.mode === 'install' ? '安装' : '更新'}任务已取消。`, AppWindow, 'warning');
  appInstallTask.value = null;
}

function appDetailAction(app: NasApp, action: string) {
  if (action === '卸载') {
    requestConfirm({
      title: '卸载应用',
      detail: `确认卸载“${app.name}”？应用数据目录 /volume1/@app/${app.id} 将保留，权限会立即回收。`,
      icon: Trash2,
      tone: 'warning',
      confirmText: '确认卸载',
      onConfirm: () => {
        installedApps[app.id] = false;
        appDetail.value = null;
        pushNotification('应用中心', `${app.name} 已卸载，应用数据已保留。`, app.icon, 'warning');
      },
    });
    return;
  }
  if (action === '检查更新') {
    const hasUpdate = app.status === '更新';
    appDetailStatus.value = {
      appId: app.id,
      mode: 'update',
      title: hasUpdate ? '发现可用更新' : '当前已是最新版本',
      detail: hasUpdate ? `${app.name} 可更新到 1.${app.id.length + 1}.0，建议在空闲时段安装。` : `${app.name} 1.${app.id.length}.6 已通过签名校验。`,
      tone: hasUpdate ? 'warning' : 'success',
      rows: hasUpdate
        ? ['签名校验通过', '保留当前配置', '更新后自动重启应用服务']
        : ['官方源响应正常', '本地包完整', '无需执行更新任务'],
    };
    pushNotification('应用中心', `${app.name} 更新检查完成。`, RefreshCcw, hasUpdate ? 'warning' : 'success');
    return;
  }
  if (action === '打开权限审计') {
    appDetailStatus.value = {
      appId: app.id,
      mode: 'audit',
      title: '权限审计完成',
      detail: `${app.name} 最近 24 小时权限调用已汇总。`,
      tone: 'info',
      rows: [
        installedApps[app.id] ? '共享文件夹访问 12 次' : '应用未安装，暂无运行权限',
        '桌面通知权限 3 次',
        app.category === '安全' ? '高敏感权限：安全扫描' : '未发现高风险提权',
      ],
    };
    pushNotification('应用中心', `${app.name} 权限审计已生成。`, Shield, 'info');
    return;
  }
  pushNotification('应用中心', `${app.name} 已${action}。`, app.icon, 'info');
}

function appActionLabel(app: NasApp) {
  if (!installedApps[app.id]) return '安装';
  if (app.status === '更新') return '更新';
  return '打开';
}

function appUpdateWindowLabel() {
  const labels = ['立即', '夜间', '空闲时'];
  return labels[appCenterSettings.updateWindow] ?? '夜间';
}

function saveAppCenterSettings() {
  showAppSettings.value = false;
  pushNotification('应用中心', `设置已保存：${appCenterSettings.autoCheck ? '自动检查' : '手动检查'}，安装时段为${appUpdateWindowLabel()}。`, Settings, 'success');
}

function resetAppCenterSettings() {
  appCenterSettings.autoCheck = true;
  appCenterSettings.nightInstall = true;
  appCenterSettings.permissionConfirm = true;
  appCenterSettings.communityApps = false;
  appCenterSettings.betaChannel = false;
  appCenterSettings.updateWindow = 2;
  pushNotification('应用中心', '应用中心设置已恢复推荐值。', RefreshCcw, 'warning');
}

function checkAppUpdatesNow() {
  activeCategory.value = '已安装';
  pushNotification('应用中心', '已开始检查已安装应用更新。', RefreshCcw, 'info');
}

function clampMenuPosition(event: MouseEvent, width = 205, height = 226) {
  return {
    x: Math.min(event.clientX, window.innerWidth - width - 12),
    y: Math.min(event.clientY, window.innerHeight - height - 12),
  };
}

function showDesktopMenu(event: MouseEvent) {
  event.preventDefault();
  appContextMenu.value.app = null;
  const position = clampMenuPosition(event, 190, 226);
  contextMenu.x = position.x;
  contextMenu.y = position.y;
  showContextMenu.value = true;
}

function showAppMenu(event: MouseEvent, app: NasApp) {
  event.preventDefault();
  event.stopPropagation();
  const position = clampMenuPosition(event, 205, 184);
  selectedDesktopApp.value = app.id;
  appContextMenu.value = { app, x: position.x, y: position.y };
  showContextMenu.value = false;
}

function setDesktopView(view: 'desktop' | 'library' | 'widgets') {
  activeDesktopView.value = view;
  peekDesktop.value = view !== 'desktop';
  showContextMenu.value = false;
}

function currentFileSpaceName() {
  return fileSpaceName(activeFileSpace.value);
}

function fileSpaceName(space: FileSpace) {
  const names: Record<FileSpace, string> = {
    personal: '个人文件夹',
    shared: '共享文件夹',
    users: '用户文件夹',
    tags: '标签',
  };
  return names[space];
}

function resetFileSelection() {
  selectedFiles.value = [];
  fileMenuOpen.value = '';
  previewFile.value = null;
  showFileFilter.value = false;
  showFileMoreMenu.value = false;
}

function pushFilePath(path: string, notify = true) {
  const normalized = path.trim().replace(/\/+/g, '/') || fileSpaceName(activeFileSpace.value);
  if (normalized === currentFilePath.value) {
    filePathInput.value = normalized;
    return;
  }
  fileHistory.value = fileHistory.value.slice(0, fileHistoryIndex.value + 1);
  fileHistory.value.push(normalized);
  fileHistoryIndex.value = fileHistory.value.length - 1;
  filePathInput.value = normalized;
  fileQuery.value = '';
  resetFileSelection();
  if (notify) pushNotification('文件管理', `已定位到${normalized}。`, Folder, 'info');
}

function setFileSpace(space: FileSpace) {
  activeFileSpace.value = space;
  fileFilter.value = '全部';
  pushFilePath(fileSpaceName(space), false);
}

function goFileHistory(direction: 'back' | 'forward') {
  const nextIndex = direction === 'back' ? fileHistoryIndex.value - 1 : fileHistoryIndex.value + 1;
  if (nextIndex < 0 || nextIndex >= fileHistory.value.length) return;
  fileHistoryIndex.value = nextIndex;
  filePathInput.value = currentFilePath.value;
  fileQuery.value = '';
  resetFileSelection();
  pushNotification('文件管理', `已${direction === 'back' ? '后退' : '前进'}到${currentFilePath.value}。`, Folder, 'info');
}

function submitFilePathInput() {
  pushFilePath(filePathInput.value);
}

function enterFile(file: FileRow) {
  if (file.type === '文件夹') {
    pushFilePath(`${currentFilePath.value}/${file.name}`);
    return;
  }
  previewFile.value = file;
}

function setFileFilter(filter: FileFilter) {
  fileFilter.value = filter;
  showFileFilter.value = false;
  pushNotification('文件管理', `已筛选${filter}。`, ListFilter, 'info');
}

function switchPanelTab(tab: string) {
  activePanelTab.value = tab;
  pushNotification('控制面板', `已切换到${currentPanelName()} · ${tab}。`, Settings, 'info');
}

function selectPanel(id: string) {
  activePanel.value = id;
  panelQuery.value = '';
  panelConfig.value = null;
}

function returnControlHome() {
  activePanel.value = 'overview';
  panelQuery.value = '';
  panelConfig.value = null;
}

function panelGroupItems(group: string) {
  const q = panelQuery.value.trim().toLowerCase();
  return panelItems.filter((item) => {
    const matchesGroup = item.group === group;
    const matchesQuery = !q || item.name.toLowerCase().includes(q) || item.group.toLowerCase().includes(q);
    return matchesGroup && matchesQuery;
  });
}

function applyWallpaper(theme: typeof wallpaperTheme.value) {
  wallpaperTheme.value = theme;
  showWallpaperPanel.value = false;
  showContextMenu.value = false;
  pushNotification('桌面设置', '壁纸样式已更新。', Image, 'success');
}

function cycleFileSort() {
  const order: typeof fileSortMode.value[] = ['名称', '类型', '时间'];
  const next = order[(order.indexOf(fileSortMode.value) + 1) % order.length];
  fileSortMode.value = next;
  pushNotification('文件管理', `已按${next}排序。`, SortAsc, 'info');
}

function refreshFiles() {
  resetFileSelection();
  pushNotification('文件管理', `${currentFilePath.value} 已刷新。`, RefreshCcw, 'success');
}

function jumpFilePath(label: string) {
  pushFilePath(label);
}

function defaultGenericSetting(appId: AppId): GenericSettingState {
  const heavierApps: AppId[] = ['docker', 'video', 'photos', 'ai', 'vm', 'jellyfin', 'plex', 'home-assistant'];
  return {
    autostart: ['sync', 'downloads', 'docker', 'photos', 'remote-access', 'webdav'].includes(appId),
    notifications: true,
    sharedAccess: true,
    remoteAccess: ['downloads', 'sync', 'netdisk', 'remote-access', 'webdav', 'alist', 'cloud-drive'].includes(appId),
    cpuLimit: heavierApps.includes(appId) ? 72 : 48,
    memoryLimit: heavierApps.includes(appId) ? 64 : 42,
  };
}

function genericSetting(appId: AppId) {
  genericSettings[appId] ||= defaultGenericSetting(appId);
  return genericSettings[appId];
}

function saveGenericSetting(appId: AppId, label: string) {
  const setting = genericSetting(appId);
  pushNotification(appName(appId), `${label}已保存：CPU ${setting.cpuLimit}% · 内存 ${setting.memoryLimit}% · ${setting.remoteAccess ? '允许远程访问' : '仅本地访问'}。`, Settings, 'success');
}

function resetGenericSetting(appId: AppId) {
  genericSettings[appId] = defaultGenericSetting(appId);
  pushNotification(appName(appId), '已恢复推荐应用设置。', RefreshCcw, 'warning');
}

function openWindowHelp(id: AppId) {
  helpContextApp.value = id;
  showHelpDialog.value = true;
}

function openAppDetail(app: NasApp) {
  appDetail.value = app;
  appDetailStatus.value = null;
  showContextMenu.value = false;
}

function submitNewFolder() {
  const name = newFolderName.value.trim() || '新建文件夹';
  pushNotification('文件管理', `已在当前目录创建“${name}”。`, Folder, 'success');
  showNewFolderDialog.value = false;
  newFolderName.value = '新建文件夹';
}

function submitUpload() {
  pushNotification('任务管理器', '上传任务已加入队列，可在任务管理器查看进度。', Upload, 'info');
  showUploadDialog.value = false;
  openApp('tasks');
}

function handleFileDrop(event: DragEvent) {
  event.preventDefault();
  isFileDragOver.value = false;
  const count = event.dataTransfer?.files.length || 3;
  pushNotification('文件管理', `${count} 个文件已加入上传队列。`, Upload, 'success');
  openApp('tasks');
}

function requestConfirm(dialog: ConfirmDialog) {
  confirmDialog.value = dialog;
}

function runConfirmAction() {
  confirmDialog.value?.onConfirm();
  confirmDialog.value = null;
}

function actionTarget(file?: FileRow) {
  return file?.name ?? (selectedFiles.value.join('、') || '所选文件');
}

function startRename(file?: FileRow) {
  const target = file ?? selectedFileRows.value[0] ?? null;
  renameFile.value = target;
  renameValue.value = target?.name ?? selectedFiles.value[0] ?? '';
  showRenameDialog.value = Boolean(renameValue.value);
  fileMenuOpen.value = '';
  if (!renameValue.value) {
    pushNotification('文件管理', '请先选择需要重命名的文件。', FileText, 'warning');
  }
}

function submitRename() {
  const nextName = renameValue.value.trim();
  if (!nextName) {
    pushNotification('文件管理', '文件名不能为空。', FileText, 'warning');
    return;
  }
  const previousName = renameFile.value?.name ?? selectedFiles.value[0] ?? '所选文件';
  pushNotification('文件管理', `“${previousName}”已重命名为“${nextName}”。`, Pencil, 'success');
  showRenameDialog.value = false;
  renameFile.value = null;
  renameValue.value = '';
}

function openTagEditor(file?: FileRow) {
  const target = file ?? selectedFileRows.value[0] ?? null;
  tagEditorFile.value = target;
  tagDraft.value = target?.tags.join('、') ?? '';
  showTagEditor.value = true;
  fileMenuOpen.value = '';
  if (!target && selectedFiles.value.length === 0) {
    showTagEditor.value = false;
    pushNotification('文件管理', '请先选择需要添加标签的文件。', Tags, 'warning');
  }
}

function draftTags() {
  return tagDraft.value
    .split(/[、,\s]+/)
    .map((tag) => tag.trim())
    .filter(Boolean);
}

function submitTags() {
  const tags = draftTags();
  const target = tagEditorFile.value?.name ?? (selectedFiles.value.join('、') || '所选文件');
  if (tags.length === 0) {
    pushNotification('文件管理', '请至少保留一个标签。', Tags, 'warning');
    return;
  }
  pushNotification('文件管理', `“${target}”已更新标签：${tags.join('、')}。`, Tags, 'success');
  tagEditorFile.value = null;
  tagDraft.value = '';
  showTagEditor.value = false;
}

function openFileOperation(mode: FileOperationDialog['mode'], file?: FileRow) {
  const target = actionTarget(file);
  fileOperationDialog.value = { mode, target };
  operationDestination.value = activeFileSpace.value === 'shared' ? 'personal' : 'shared';
  fileMenuOpen.value = '';
}

function submitFileOperation() {
  if (!fileOperationDialog.value) return;
  const mode = fileOperationDialog.value.mode;
  const action = mode === 'copy' ? '复制' : '移动';
  const target = fileOperationDialog.value.target;
  const destination = fileSpaceName(operationDestination.value);
  fileOperationDialog.value = null;
  selectedFiles.value = [];
  pushNotification('文件管理', `“${target}”已加入${action}任务，目标位置：${destination}。`, mode === 'copy' ? Copy : Folder, 'success');
  openApp('tasks');
}

function openShareDialog(file?: FileRow) {
  const target = file ?? selectedFileRows.value[0] ?? null;
  if (!target && selectedFiles.value.length === 0) {
    pushNotification('文件管理', '请先选择需要共享的文件。', Link2, 'warning');
    return;
  }
  shareFile.value = target;
  showShareDialog.value = true;
  fileMenuOpen.value = '';
}

function shareTargetName() {
  return shareFile.value?.name ?? (selectedFiles.value.join('、') || '所选文件');
}

function shareLink() {
  const encoded = encodeURIComponent(shareTargetName().replace(/\s+/g, '-'));
  return `https://ug.link/share/${encoded}`;
}

function copyShareLink() {
  pushNotification('文件管理', `“${shareTargetName()}”的共享链接已复制。`, Link2, 'success');
  showShareDialog.value = false;
  shareFile.value = null;
  selectedFiles.value = [];
}

function applyFileAction(action: string, file?: FileRow) {
  const target = actionTarget(file);
  fileMenuOpen.value = '';
  if (action === 'preview' && file) {
    previewFile.value = file;
    return;
  }
  if (action === 'rename') {
    startRename(file);
    return;
  }
  if (action === 'tag') {
    openTagEditor(file);
    return;
  }
  if (action === 'copy' || action === 'move') {
    openFileOperation(action, file);
    return;
  }
  if (action === 'share') {
    openShareDialog(file);
    return;
  }
  if (action === 'delete') {
    requestConfirm({
      title: '移动到回收站',
      detail: `确认将“${target}”移动到回收站？你可以稍后在回收站恢复或彻底删除。`,
      icon: Trash2,
      tone: 'warning',
      confirmText: '移动到回收站',
      onConfirm: () => {
        selectedFiles.value = [];
        previewFile.value = null;
        pushNotification('文件管理', `“${target}”已移动到回收站。`, Trash2, 'warning');
      },
    });
    return;
  }
  const messageMap: Record<string, string> = {
    download: `“${target}”已加入下载队列。`,
  };
  pushNotification('文件管理', messageMap[action] ?? `已处理“${target}”。`, FileText, 'success');
}

function lockDesktop() {
  showUserMenu.value = false;
  showLockScreen.value = true;
  unlockCode.value = '';
}

function unlockDesktop() {
  showLockScreen.value = false;
  pushNotification('账号安全', '已通过本地会话解锁桌面。', LockKeyhole, 'success');
}

function runPowerAction(action: string) {
  showPowerDialog.value = false;
  if (action === '关机' || action === '重启') {
    requestConfirm({
      title: `${action}设备`,
      detail: `确认立即${action}？系统会先停止同步、下载和转码任务，并通知在线用户。`,
      icon: Power,
      tone: 'warning',
      confirmText: `确认${action}`,
      onConfirm: () => {
        pushNotification('硬件与电源', `${action}指令已进入确认队列。`, Power, 'warning');
      },
    });
    return;
  }
  pushNotification('硬件与电源', `${action}指令已进入确认队列。`, Power, 'info');
}

function clearNotifications() {
  notifications.value = [];
  hasUnreadNotifications.value = false;
}

function dismissNotification(id: number) {
  notifications.value = notifications.value.filter((item) => item.id !== id);
  if (toastNotification.value?.id === id) toastNotification.value = null;
  if (notifications.value.length === 0) hasUnreadNotifications.value = false;
}

function applyPanelCard(title: string, status: string) {
  const card = panelCards().find((item) => item.title === title);
  panelConfig.value = {
    panel: currentPanelName(),
    title,
    detail: card?.detail ?? `${title}配置项`,
    status,
    icon: card?.icon ?? Settings,
  };
}

function savePanelConfig() {
  if (!panelConfig.value) return;
  pushNotification('控制面板', `${panelConfig.value.panel} · ${panelConfig.value.title}设置已保存。`, Settings, 'success');
  panelConfig.value = null;
}

function testPanelConfig() {
  if (!panelConfig.value) return;
  pushNotification('控制面板', `${panelConfig.value.title}连通性检测已完成。`, panelConfig.value.icon, 'info');
}

function resetPanelConfig() {
  if (!panelConfig.value) return;
  pushNotification('控制面板', `${panelConfig.value.title}已恢复推荐配置。`, RefreshCcw, 'warning');
}

function handleTaskAction(taskName: string, action: string) {
  const task = tasks.find((item) => item.name === taskName);
  if (!task) return;
  if (action === '取消') {
    task.status = '已取消';
    task.progress = 0;
    pushNotification('任务管理器', `${taskName} 已取消，相关临时队列已清理。`, CalendarClock, 'warning');
    return;
  }
  task.status = task.status === '已暂停' ? '运行中' : '已暂停';
  pushNotification('任务管理器', `${taskName} 已${task.status === '已暂停' ? '暂停' : '继续'}。`, CalendarClock, task.status === '已暂停' ? 'warning' : 'success');
}

function openWidgetTarget(target: AppId, detail: string) {
  openApp(target);
  pushNotification('小组件', detail, appIcon(target), 'info');
}

function runFeatureAction(appId: AppId, title: string) {
  pushNotification(appName(appId), `${title} 已进入操作流程。`, appIcon(appId), 'info');
}

function closeTransientLayers() {
  showGlobalSearch.value = false;
  showNotifications.value = false;
  showContextMenu.value = false;
  showUserMenu.value = false;
  showSystemPanel.value = false;
  appContextMenu.value.app = null;
  showPowerDialog.value = false;
  showUploadDialog.value = false;
  showNewFolderDialog.value = false;
  showHelpDialog.value = false;
  helpContextApp.value = null;
  showWallpaperPanel.value = false;
  showFileFilter.value = false;
  showFileMoreMenu.value = false;
  showAppSettings.value = false;
  showTagEditor.value = false;
  showRenameDialog.value = false;
  showShareDialog.value = false;
  isFileDragOver.value = false;
  appDetail.value = null;
  previewFile.value = null;
  panelConfig.value = null;
  confirmDialog.value = null;
  renameFile.value = null;
  renameValue.value = '';
  tagEditorFile.value = null;
  tagDraft.value = '';
  fileOperationDialog.value = null;
  shareFile.value = null;
  appInstallDialog.value = null;
  toastNotification.value = null;
}

function onKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') {
    closeTransientLayers();
  }
  if ((event.metaKey || event.ctrlKey) && event.key.toLowerCase() === 'k') {
    event.preventDefault();
    showGlobalSearch.value = true;
  }
}

function genericMetrics(id: AppId) {
  const map: Partial<Record<AppId, string[]>> = {
    storage: ['存储池 1 个', '硬盘 4 块', 'SMART 正常'],
    downloads: ['下载中 3 个', '限速 12 MB/s', '自动归档开启'],
    sync: ['同步任务 4 个', '备份任务 2 个', '版本保留 32'],
    docker: ['容器 8 个', '镜像 21 个', 'Compose 3 组'],
    video: ['影片 126 部', '字幕任务 4 个', 'DVR 已开启'],
    music: ['歌曲 2,184 首', '专辑 186 张', '歌词匹配 92%'],
    photos: ['照片 18,420 张', '人物 42 个', '回忆 8 组'],
    logs: ['系统日志 286 条', '登录日志 12 条', '安全告警 1 条'],
    ai: ['本地模型 3 个', '索引文档 12k', '工具 18 个'],
    hermes: ['自动化 7 个', '待确认 2 个', '审计通过 31 次'],
  };
  return map[id] ?? ['状态正常', '任务联动', '审计开启'];
}

function appFeatureCards(id: AppId) {
  const map: Partial<Record<AppId, { title: string; detail: string; icon: unknown }[]>> = {
    docker: [
      { title: '容器', detail: '启动、停止、重启、终端和日志', icon: Container },
      { title: 'Compose', detail: '以项目方式管理多容器应用', icon: Boxes },
      { title: '网络与卷', detail: '端口映射、桥接网络和持久化目录', icon: Network },
      { title: '镜像仓库', detail: '搜索、拉取、清理和更新镜像', icon: CloudDownload },
    ],
    video: [
      { title: '媒体库', detail: '电影、剧集、海报墙和刮削修正', icon: Film },
      { title: '字幕', detail: '本地字幕、在线下载和时间校准', icon: FileText },
      { title: '转码', detail: 'HEVC 转 H.264 与兼容模式', icon: Gauge },
      { title: '直播 DVR', detail: '频道、节目单和定时录制', icon: CalendarClock },
    ],
    photos: [
      { title: '时间线', detail: '按日期、地点和设备浏览', icon: Image },
      { title: '人物', detail: 'AI 人脸聚类、合并和命名', icon: Users },
      { title: '回忆', detail: '自动生成旅行、家庭和节日相册', icon: BrainCircuit },
      { title: '备份', detail: '手机照片自动备份和共享相册', icon: RefreshCcw },
    ],
    ai: [
      { title: '私人助手', detail: '理解文件、媒体和系统状态', icon: Bot },
      { title: '模型管理', detail: '下载、启用和切换本地模型', icon: BrainCircuit },
      { title: '文件智能整理', detail: '重复文件、权限风险和标签建议', icon: Tags },
      { title: '工具调用', detail: '高风险动作确认、审计和回滚', icon: Shield },
    ],
  };
  return map[id] ?? [
    { title: '状态正常', detail: '服务运行中，所有关键依赖可用。', icon: CheckCircle2 },
    { title: '筛选与搜索', detail: '支持按状态、类型、来源和时间范围筛选。', icon: ListFilter },
    { title: '权限与审计', detail: '高风险操作需要确认，并进入日志中心。', icon: Shield },
    { title: '任务联动', detail: '长任务统一进入任务管理器。', icon: Power },
  ];
}

function currentPanelName() {
  return panelItems.find((item) => item.id === activePanel.value)?.name ?? '硬件与电源';
}

function panelCards() {
  const map: Record<string, { title: string; detail: string; status: string; icon: unknown }[]> = {
    users: [
      { title: '本地用户', detail: '管理员 1 个，普通用户 4 个', status: '可管理', icon: Users },
      { title: '账号安全', detail: 'OTP 双重验证、登录设备与密码策略', status: '已启用', icon: LockKeyhole },
      { title: '容量配额', detail: '个人文件夹与共享文件夹使用上限', status: '3 项规则', icon: Database },
      { title: '会话管理', detail: '查看账号活动并强制退出其他终端', status: '12 条日志', icon: Monitor },
    ],
    'file-service': [
      { title: 'SMB', detail: '局域网文件共享，端口 445', status: '运行中', icon: Folder },
      { title: 'WebDAV', detail: '远程文件访问和第三方客户端挂载', status: '已开启', icon: Globe2 },
      { title: 'FTP/SFTP', detail: '兼容旧设备与自动化脚本', status: '未开启', icon: SquareTerminal },
      { title: '共享权限', detail: '按用户、用户组和文件夹继承规则管理', status: '8 条规则', icon: Shield },
    ],
    'device-link': [
      { title: 'UGREENlink ID', detail: 'higonas，可通过浏览器或客户端远程连接', status: '正常', icon: Cloud },
      { title: '连接模式', detail: '优先直连，网络受限时自动中继', status: '智能', icon: Network },
      { title: '网络唤醒', detail: '历史设备与 MAC 地址唤醒', status: '可用', icon: Power },
      { title: '设备绑定', detail: '设备所有权、解绑、转让和辅助验证', status: '已绑定', icon: Monitor },
    ],
    ldap: [
      { title: '域服务', detail: '加入企业 AD / LDAP 目录', status: '未配置', icon: BriefcaseBusiness },
      { title: '目录同步', detail: '用户、组和权限映射', status: '待启用', icon: RefreshCcw },
      { title: '登录策略', detail: '本地账号与域账号混合登录', status: '可配置', icon: LockKeyhole },
      { title: '审计', detail: '域登录与权限变更记录', status: '开启', icon: FileText },
    ],
    terminal: [
      { title: 'SSH', detail: '受控开启终端访问', status: '未开启', icon: SquareTerminal },
      { title: '密钥', detail: '导入公钥并限制账号范围', status: '0 个', icon: LockKeyhole },
      { title: '端口', detail: '默认 22，可按安全策略调整', status: '待配置', icon: Network },
      { title: '命令审计', detail: '高风险命令进入日志中心', status: '开启', icon: FileText },
    ],
    power: [
      { title: '电源计划', detail: '休眠、重启、关机和更新后重启', status: '正常', icon: Power },
      { title: '硬盘休眠', detail: '空闲 20 分钟进入节能模式', status: '已启用', icon: HardDrive },
      { title: '风扇策略', detail: '静音、均衡、性能三档', status: '均衡', icon: Gauge },
      { title: '设备分析', detail: 'CPU、内存、温度、网络与服务健康', status: '实时', icon: Monitor },
    ],
    time: [
      { title: '语言', detail: '简体中文，跟随浏览器可选', status: '中文', icon: Globe2 },
      { title: '日期格式', detail: 'YYYY-MM-DD', status: '已保存', icon: CalendarClock },
      { title: '时间格式', detail: '24 小时制', status: '已保存', icon: CalendarClock },
      { title: 'NTP', detail: '自动同步网络时间', status: '正常', icon: RefreshCcw },
    ],
    network: [
      { title: 'LAN2', detail: '192.168.1.118 / 1Gbps / MTU1500', status: '在线', icon: Network },
      { title: 'HTTPS', detail: '启用传输加密和证书管理', status: '已开启', icon: LockKeyhole },
      { title: 'DNS', detail: '自动获取，支持手动配置', status: '自动', icon: Globe2 },
      { title: '网络诊断', detail: '连通性、延迟、端口和路由检测', status: '可运行', icon: Gauge },
    ],
    security: [
      { title: 'OTP 双重验证', detail: '登录本地账号时需要验证码', status: '已启用', icon: LockKeyhole },
      { title: '防火墙', detail: '控制入站端口和远程服务暴露', status: '运行中', icon: Shield },
      { title: '病毒扫描', detail: '计划扫描个人与共享文件夹', status: '明日 03:00', icon: Search },
      { title: '风险审计', detail: '分享链接、权限提升和端口风险', status: '1 个警告', icon: Bell },
    ],
    index: [
      { title: '文件索引', detail: '文件名、标签、内容和元数据', status: '已同步', icon: Database },
      { title: '媒体索引', detail: '相册、音乐和影视元数据', status: '扫描中', icon: Image },
      { title: '搜索范围', detail: '个人、共享和团队文件夹', status: '3 个空间', icon: Search },
      { title: '重建索引库', detail: '异常时可重新构建搜索索引', status: '可执行', icon: RefreshCcw },
    ],
    about: [
      { title: '设备信息', detail: 'DXP4800 GT / X48006J65000545B', status: '正常', icon: Monitor },
      { title: '系统版本', detail: 'UGOS Pro 1.16.0.0085', status: '最新', icon: Info },
      { title: '保修', detail: '2028-06-23 12:33 到期', status: '有效', icon: CheckCircle2 },
      { title: '设备日志', detail: '上传给技术支持用于诊断', status: '可上传', icon: FileText },
    ],
    updates: [
      { title: '系统更新', detail: '检测、下载、安装和重启', status: '最新', icon: RefreshCcw },
      { title: '配置备份', detail: '云备份系统配置和服务状态', status: '已开启', icon: Cloud },
      { title: '系统还原', detail: '从备份恢复配置和权限', status: '可用', icon: Recycle },
      { title: '恢复出厂设置', detail: '高风险动作，需要确认和审计', status: '需确认', icon: Shield },
    ],
  };
  return map[activePanel.value] ?? map.power;
}

onMounted(() => {
  window.addEventListener('keydown', onKeydown);
});

onUnmounted(() => {
  stopDrag();
  stopResize();
  window.removeEventListener('keydown', onKeydown);
});
</script>

<template>
  <main class="desktop" :class="`wallpaper-${wallpaperTheme}`" @contextmenu="showDesktopMenu" @click="showContextMenu = false; showUserMenu = false; showFileMoreMenu = false; appContextMenu.app = null">
    <div class="wallpaper">
      <div class="planet planet-left"></div>
      <div class="planet planet-right"></div>
      <div class="arc arc-one"></div>
      <div class="arc arc-two"></div>
    </div>

    <header class="topbar">
      <div class="view-switcher">
        <button :class="{ active: activeDesktopView === 'desktop' }" title="桌面" @click.stop="toggleDesktopFromTopbar"><Monitor :size="18" /></button>
        <button :class="{ active: activeDesktopView === 'library' }" title="应用库" @click.stop="setDesktopView('library')"><LayoutGrid :size="18" /></button>
        <button
          v-for="win in windows"
          :key="`switch-${win.id}`"
          class="window-shortcut"
          :class="{ active: !win.minimized && win.z === activeWindowZ, minimized: win.minimized }"
          :title="appName(win.id)"
          @click.stop="openApp(win.id)"
        >
          <span class="switch-app-icon" :style="{ background: appMap.get(win.id)?.color }">
            <component :is="appIcon(win.id)" :size="18" />
          </span>
        </button>
      </div>
      <div class="status-area">
        <button class="meter" title="系统状态" @click.stop="showSystemPanel = !showSystemPanel"><Gauge :size="17" /><span>CPU</span></button>
        <button class="meter" title="内存状态" @click.stop="showSystemPanel = !showSystemPanel"><Database :size="17" /><span>RAM</span></button>
        <button class="net" title="网络状态" @click.stop="showSystemPanel = !showSystemPanel">↑ 10.9KB/s<br />↓ 9.8KB/s</button>
        <button @click="openApp('tasks')" title="任务"><PanelLeft :size="20" /></button>
        <button
          @click="toggleNotifications"
          title="通知"
          class="notice-dot"
          :class="{ 'has-notices': hasUnreadNotifications }"
        >
          <Bell :size="20" />
        </button>
        <button @click="showGlobalSearch = !showGlobalSearch" title="搜索"><Search :size="21" /></button>
        <button title="个人中心" @click.stop="showUserMenu = !showUserMenu"><User :size="21" /></button>
      </div>
    </header>

    <aside v-if="activeDesktopView === 'desktop'" class="desktop-icons">
      <button
        v-for="app in desktopApps"
        :key="app.id"
        class="desktop-icon"
        :class="{ selected: selectedDesktopApp === app.id }"
        @contextmenu="showAppMenu($event, app)"
        @dblclick="openApp(app.id)"
        @click="selectedDesktopApp = app.id; openApp(app.id)"
      >
        <span class="app-tile" :style="{ background: app.color }"><component :is="app.icon" :size="31" /></span>
        <span>{{ app.name }}</span>
      </button>
    </aside>

    <section v-if="showSystemPanel" class="global-popover system-panel" @click.stop>
      <div class="popover-head">
        <strong>系统状态</strong>
        <button @click="openApp('control-panel'); activePanel = 'power'; showSystemPanel = false">详情</button>
      </div>
      <article><span>CPU</span><div class="meter-bar"><b style="width: 12%"></b></div><strong>12%</strong></article>
      <article><span>内存</span><div class="meter-bar"><b style="width: 41%"></b></div><strong>41%</strong></article>
      <article><span>温度</span><div class="meter-bar warm"><b style="width: 42%"></b></div><strong>42°C</strong></article>
      <article><span>网络</span><div>↑ 10.9KB/s · ↓ 9.8KB/s</div><strong>LAN2</strong></article>
      <footer>
        <button @click="openApp('tasks'); showSystemPanel = false"><CalendarClock :size="15" /> 任务</button>
        <button @click="openApp('storage'); showSystemPanel = false"><HardDrive :size="15" /> 存储</button>
        <button @click="openApp('logs'); showSystemPanel = false"><FileText :size="15" /> 日志</button>
      </footer>
    </section>

    <section v-if="activeDesktopView === 'library'" class="app-library">
      <div class="library-head">
        <h1>应用库</h1>
        <label><Search :size="18" /><input v-model="query" placeholder="搜索应用" /></label>
      </div>
      <div class="library-grid">
        <button v-for="app in libraryApps" :key="app.id" @click="installOrOpen(app)">
          <span class="app-tile" :style="{ background: app.color }"><component :is="app.icon" :size="30" /></span>
          <strong>{{ app.name }}</strong>
          <small>{{ app.category }}</small>
        </button>
      </div>
    </section>

    <section v-else-if="activeDesktopView === 'widgets'" class="widget-board">
      <article role="button" tabindex="0" @click="openWidgetTarget('control-panel', '已打开设备状态详情。')">
        <h3>设备状态</h3>
        <div class="widget-ring">42°C</div>
        <p>CPU 12% · RAM 41%</p>
      </article>
      <article>
        <h3>正在执行任务</h3>
        <div v-for="task in tasks.slice(0, 3)" :key="task.name" class="mini-task" @click="openWidgetTarget('tasks', `${task.name} 已定位到任务管理器。`)">
          <span>{{ task.name }}</span><b>{{ task.progress }}%</b>
        </div>
      </article>
      <article role="button" tabindex="0" @click="openWidgetTarget('storage', '已打开存储空间详情。')">
        <h3>存储空间</h3>
        <div class="storage-bars"><span></span><span></span><span></span></div>
        <p>8.2 TB / 16 TB 已使用</p>
      </article>
    </section>

    <aside v-if="toastNotification" class="toast" :class="`tone-${toastNotification.tone}`">
      <component :is="toastNotification.icon" :size="18" />
      <div><strong>{{ toastNotification.title }}</strong><span>{{ toastNotification.detail }}</span></div>
      <button @click="toastNotification = null"><X :size="14" /></button>
    </aside>

    <section v-if="showGlobalSearch" class="global-popover search-popover" @click.stop>
      <Search :size="20" />
      <input v-model="globalQuery" autofocus placeholder="搜索应用、文件、设置" />
      <div class="search-results">
        <p>应用</p>
        <button v-for="app in searchApps" :key="app.id" @click="installOrOpen(app); showGlobalSearch = false">
          <component :is="app.icon" :size="18" /><span>{{ app.name }}</span><small>{{ installedApps[app.id] ? app.category : `${app.category} · 未安装` }}</small>
        </button>
        <p>文件</p>
        <button v-for="file in searchFiles" :key="file.name" @click="openApp('file-manager'); previewFile = file; showGlobalSearch = false">
          <component :is="file.icon" :size="18" /><span>{{ file.name }}</span><small>{{ file.type }}</small>
        </button>
        <p>设置</p>
        <button v-for="item in searchSettings" :key="item.id" @click="openApp('control-panel'); activePanel = item.id; showGlobalSearch = false">
          <component :is="item.icon" :size="18" /><span>{{ item.name }}</span><small>{{ item.group }}</small>
        </button>
        <button @click="openApp('control-panel'); activePanel = 'network'; showGlobalSearch = false">
          <Wifi :size="18" /><span>网络设置</span><small>快捷入口</small>
        </button>
      </div>
    </section>

    <section v-if="showUserMenu" class="global-popover user-menu" @click.stop>
      <div class="user-card">
        <span><User :size="22" /></span>
        <div><strong>hiveton</strong><small>管理员 · UGREENlink 已连接</small></div>
      </div>
      <button @click="openApp('control-panel'); activePanel = 'users'; showUserMenu = false"><Users :size="17" /> 账号与权限</button>
      <button @click="lockDesktop"><LockKeyhole :size="17" /> 锁定屏幕</button>
      <button @click="showPowerDialog = true; showUserMenu = false"><Power :size="17" /> 电源选项</button>
      <button @click="pushNotification('账号安全', '已模拟退出当前网页会话。', User, 'info'); showUserMenu = false"><X :size="17" /> 退出账号</button>
    </section>

    <section v-if="showNotifications" class="global-popover notifications" @click.stop>
      <div class="popover-head">
        <strong>通知中心</strong>
        <button @click="clearNotifications">清除通知</button>
      </div>
      <article v-for="item in notifications" :key="item.id" :class="`tone-${item.tone}`">
        <component :is="item.icon" :size="18" />
        <div><strong>{{ item.title }}</strong><span>{{ item.detail }}</span></div>
        <button title="关闭通知" @click="dismissNotification(item.id)"><X :size="14" /></button>
      </article>
      <p v-if="notifications.length === 0" class="empty-state">暂无消息通知</p>
    </section>

    <section
      v-if="showContextMenu"
      class="context-menu"
      :style="{ left: contextMenu.x + 'px', top: contextMenu.y + 'px' }"
      @click.stop
    >
      <button @click="minimizeAllWindows"><Monitor :size="16" /> 显示桌面</button>
      <button @click="setDesktopView('library')"><LayoutGrid :size="16" /> 应用库</button>
      <button @click="setDesktopView('widgets')"><SlidersHorizontal :size="16" /> 小组件</button>
      <button @click="openApp('app-center')"><AppWindow :size="16" /> 管理应用</button>
      <button @click="showWallpaperPanel = true"><Image :size="16" /> 更换壁纸</button>
    </section>

    <section
      v-if="appContextMenu.app"
      class="context-menu app-context-menu"
      :style="{ left: appContextMenu.x + 'px', top: appContextMenu.y + 'px' }"
      @click.stop
    >
      <button @click="openApp(appContextMenu.app.id); appContextMenu.app = null"><AppWindow :size="16" /> 打开应用</button>
      <button @click="openAppDetail(appContextMenu.app); appContextMenu.app = null"><Info :size="16" /> 查看详情</button>
      <button @click="installOrOpen(appContextMenu.app); appContextMenu.app = null"><CloudDownload :size="16" /> {{ appActionLabel(appContextMenu.app) }}</button>
      <button @click="pushNotification('桌面', `${appContextMenu.app?.name} 已固定到常用。`, AppWindow, 'success'); appContextMenu.app = null"><CheckCircle2 :size="16" /> 固定到常用</button>
    </section>

    <nav v-if="minimizedWindows.length" class="minimized-dock">
      <button @click="restoreAllWindows"><Monitor :size="17" /> 全部恢复</button>
      <button v-for="win in minimizedWindows" :key="win.id" @click="openApp(win.id)">
        <component :is="appIcon(win.id)" :size="17" /> {{ appName(win.id) }}
      </button>
    </nav>

    <section v-if="showPowerDialog" class="modal-scrim" @click="showPowerDialog = false">
      <article class="system-modal power-modal" @click.stop>
        <header><Power :size="22" /><strong>电源选项</strong><button @click="showPowerDialog = false"><X :size="18" /></button></header>
        <p>选择要执行的系统动作。高风险动作会进入审计日志并要求真实环境二次确认。</p>
        <div class="power-actions">
          <button @click="runPowerAction('休眠')"><Monitor :size="20" /> 休眠</button>
          <button @click="runPowerAction('重启')"><RefreshCcw :size="20" /> 重启</button>
          <button class="danger" @click="runPowerAction('关机')"><Power :size="20" /> 关机</button>
        </div>
      </article>
    </section>

    <section v-if="showLockScreen" class="lock-screen">
      <div class="lock-panel">
        <span class="lock-avatar"><User :size="34" /></span>
        <h2>HiGoNAS 已锁定</h2>
        <p>hiveton · 192.168.1.118</p>
        <label><LockKeyhole :size="18" /><input v-model="unlockCode" type="password" autofocus placeholder="输入任意内容解锁" @keyup.enter="unlockDesktop" /></label>
        <button :disabled="!unlockCode.trim()" @click="unlockDesktop">解锁</button>
      </div>
    </section>

    <section class="window-layer">
      <article
        v-for="win in visibleWindows"
        :key="win.id"
        class="nas-window"
        :class="[`window-${win.id}`, { maximized: win.maximized, active: win.z === activeWindowZ }]"
        :style="windowStyle(win)"
        @mousedown="focusWindow(win.id)"
      >
        <header class="window-titlebar" @pointerdown="beginDrag($event, win)" @dblclick="toggleMaximize(win.id)">
          <div class="traffic">
            <button class="close" title="关闭" @click.stop="closeWindow(win.id)"></button>
            <button class="min" title="最小化" @click.stop="minimizeWindow(win.id)"></button>
            <button class="max" title="最大化" @click.stop="toggleMaximize(win.id)"></button>
          </div>
          <div class="window-title"><component :is="appIcon(win.id)" :size="18" /> {{ appName(win.id) }}</div>
          <div class="window-tools">
            <button title="贴靠左侧" @click.stop="snapWindow(win.id, 'left')"><ChevronLeft :size="17" /></button>
            <button title="贴靠右侧" @click.stop="snapWindow(win.id, 'right')"><ChevronRight :size="17" /></button>
            <button class="help" title="帮助" @click.stop="openWindowHelp(win.id)"><CircleHelp :size="20" /></button>
          </div>
        </header>

        <div v-if="win.id === 'control-panel'" class="window-body control-panel" :class="{ 'control-home': activePanel === 'overview' }">
          <aside v-if="activePanel !== 'overview'" class="control-side">
            <button class="grid-button" title="返回首页" @click="returnControlHome"><LayoutGrid :size="20" /></button>
            <label class="panel-search"><Search :size="15" /><input v-model="panelQuery" placeholder="请输入" /></label>
            <template v-for="group in ['连接与访问', '通用设置', '系统服务']" :key="group">
              <p v-if="filteredPanelItems.some((entry) => entry.group === group)" class="side-group">{{ group }}</p>
              <button
                v-for="item in filteredPanelItems.filter((entry) => entry.group === group)"
                :key="item.id"
                :class="{ selected: activePanel === item.id }"
                @click="selectPanel(item.id)"
              >
                <component :is="item.icon" :size="20" /> {{ item.name }}
              </button>
            </template>
            <p v-if="filteredPanelItems.length === 0" class="side-empty">没有找到设置项</p>
          </aside>
          <main class="control-main">
            <section v-if="activePanel === 'overview'" class="control-home-main">
              <label class="control-home-search"><Search :size="16" /><input v-model="panelQuery" placeholder="请输入" /></label>
              <section v-for="group in ['连接与访问', '通用设置', '系统服务']" :key="group" class="control-home-section">
                <h3 v-if="panelGroupItems(group).length">{{ group }}</h3>
                <div v-if="panelGroupItems(group).length" class="control-home-grid">
                  <button v-for="item in panelGroupItems(group)" :key="item.id" @click="selectPanel(item.id)">
                    <span><component :is="item.icon" :size="32" /></span>
                    <strong>{{ item.name }}</strong>
                  </button>
                </div>
              </section>
              <p v-if="filteredPanelItems.length === 0" class="table-empty">没有找到设置项</p>
            </section>
            <template v-else>
            <nav class="tabs">
              <button
                v-for="tab in ['概览', '常规', '存储', '应用', '服务', '设备分析']"
                :key="tab"
                :class="{ active: activePanelTab === tab }"
                @click="switchPanelTab(tab)"
              >
                {{ tab === '概览' ? currentPanelName() : tab }}
              </button>
            </nav>
            <section v-if="activePanel === 'overview' || activePanel === 'power'" class="device-hero">
              <div>
                <span>设备名称</span>
                <h2>HiGoNAS <button @click="applyPanelCard('设备名称', '可编辑')">编辑</button></h2>
                <span>UGREENlink ID</span>
                <h3>higonas</h3>
              </div>
              <div>
                <span>系统版本</span>
                <h3>1.16.0.0085</h3>
                <span>设备所有者</span>
                <h3>186****3878</h3>
              </div>
              <div class="nas-visual">
                <div v-for="bay in 4" :key="bay" class="bay">0{{ bay }}</div>
              </div>
            </section>
            <section v-else class="panel-hero">
              <div>
                <span>{{ currentPanelName() }}</span>
                <h2>{{ currentPanelName() }}</h2>
                <p>{{ panelCards()[0].detail }}</p>
              </div>
              <component :is="panelCards()[0].icon" :size="54" />
            </section>
            <section v-if="activePanel === 'overview' || activePanel === 'power'" class="control-cards">
              <article>
                <h3><Monitor :size="18" /> 设备</h3>
                <dl><dt>型号</dt><dd>DXP4800 GT</dd><dt>序列号</dt><dd>X48006J65000545B</dd><dt>运行时间</dt><dd>03 时 19 分 26 秒</dd></dl>
              </article>
              <article>
                <h3><Gauge :size="18" /> 硬件</h3>
                <dl><dt>CPU</dt><dd>AMD Ryzen Embedded R2514</dd><dt>核心</dt><dd>4 核 / 8 线程 / 45°C</dd><dt>内存</dt><dd>8 GB / 3200 MHz</dd></dl>
              </article>
              <article>
                <h3><Network :size="18" /> 网络</h3>
                <dl><dt>LAN2</dt><dd>192.168.1.118</dd><dt>速率</dt><dd>1Gbps / 全双工 / MTU1500</dd></dl>
              </article>
              <article>
                <h3><LockKeyhole :size="18" /> 安全</h3>
                <dl><dt>MFA</dt><dd>已启用 OTP</dd><dt>远程访问</dt><dd>UGREENlink 正常</dd></dl>
              </article>
            </section>
            <section v-else class="panel-card-grid">
              <article v-for="card in panelCards()" :key="card.title">
                <div><component :is="card.icon" :size="22" /><h3>{{ card.title }}</h3></div>
                <p>{{ card.detail }}</p>
                <footer><span>{{ card.status }}</span><button @click="applyPanelCard(card.title, card.status)">配置</button></footer>
              </article>
            </section>
            <aside v-if="panelConfig" class="panel-config-drawer" @click.stop>
              <header>
                <span><component :is="panelConfig.icon" :size="22" /></span>
                <div><strong>{{ panelConfig.title }}</strong><small>{{ panelConfig.panel }}</small></div>
                <button @click="panelConfig = null"><X :size="18" /></button>
              </header>
              <p>{{ panelConfig.detail }}</p>
              <div class="config-status">
                <span>当前状态</span>
                <strong>{{ panelConfig.status }}</strong>
              </div>
              <section class="config-form">
                <label><span>启用此配置</span><input type="checkbox" checked /></label>
                <label><span>写入操作日志</span><input type="checkbox" checked /></label>
                <label><span>异常时桌面通知</span><input type="checkbox" checked /></label>
                <label><span>策略强度</span><input type="range" min="1" max="3" value="2" /></label>
                <label><span>备注</span><input value="HiGoNAS 推荐配置" /></label>
              </section>
              <footer>
                <button @click="resetPanelConfig"><RefreshCcw :size="15" /> 推荐</button>
                <button @click="testPanelConfig"><Gauge :size="15" /> 测试</button>
                <button class="primary" @click="savePanelConfig">保存</button>
              </footer>
            </aside>
            </template>
          </main>
        </div>

        <div v-else-if="win.id === 'file-manager'" class="window-body file-manager">
          <aside class="file-side">
            <button :class="{ active: activeFileSpace === 'personal' }" @click="setFileSpace('personal')"><ChevronRight :size="15" /> 个人文件夹</button>
            <button :class="{ active: activeFileSpace === 'shared' }" @click="setFileSpace('shared')"><ChevronRight :size="15" /> 共享文件夹</button>
            <button :class="{ active: activeFileSpace === 'users' }" @click="setFileSpace('users')"><ChevronRight :size="15" /> 用户文件夹</button>
            <button :class="{ active: activeFileSpace === 'tags' }" @click="setFileSpace('tags')"><ChevronRight :size="15" /> 标签</button>
            <button class="side-trash" title="打开回收站" @click="openApp('recycle')"><Trash2 :size="20" /></button>
          </aside>
          <main
            class="file-main"
            :class="{ dragging: isFileDragOver }"
            @dragover.prevent="isFileDragOver = true"
            @dragleave="isFileDragOver = false"
            @drop="handleFileDrop"
          >
            <div class="file-toolbar">
              <button :disabled="!canGoBackFilePath" title="后退" @click="goFileHistory('back')"><ChevronLeft :size="18" /></button>
              <button :disabled="!canGoForwardFilePath" title="前进" @click="goFileHistory('forward')"><ChevronRight :size="18" /></button>
              <button @click="refreshFiles"><RefreshCcw :size="18" /></button>
              <input v-model="filePathInput" title="路径" @keydown.enter="submitFilePathInput" @blur="filePathInput = currentFilePath" />
              <label><Search :size="16" /><input v-model="fileQuery" placeholder="请输入" /></label>
            </div>
            <div class="file-actions">
              <div class="quick-operation-box">
                <button title="新建" @click="showNewFolderDialog = true"><Folder :size="16" /></button>
                <button title="上传" @click="showUploadDialog = true"><Upload :size="16" /></button>
                <button title="复制" @click="applyFileAction('copy')"><Copy :size="16" /></button>
                <button title="移动" @click="applyFileAction('move')"><Folder :size="16" /></button>
                <button title="剪切" @click="applyFileAction('move')"><X :size="16" /></button>
                <button title="删除" @click="applyFileAction('delete')"><Trash2 :size="16" /></button>
              </div>
              <div class="tools-box-area">
                <button title="任务" @click="openApp('tasks')"><CalendarClock :size="16" /></button>
                <button title="筛选" :class="{ active: showFileFilter || fileFilter !== '全部' }" @click.stop="showFileFilter = !showFileFilter; showFileMoreMenu = false"><ListFilter :size="16" /></button>
                <button title="排序" @click="cycleFileSort"><SortAsc :size="16" /></button>
                <button title="更多" @click.stop="showFileMoreMenu = !showFileMoreMenu; showFileFilter = false"><MoreVertical :size="16" /></button>
                <button title="详情" @click="previewFile = filteredFiles[0] || null"><Info :size="16" /></button>
              </div>
              <div v-if="showFileFilter" class="floating-menu file-filter-menu" @click.stop>
                <strong>筛选类型</strong>
                <button v-for="filter in fileFilterOptions" :key="filter" :class="{ active: fileFilter === filter }" @click="setFileFilter(filter)">
                  {{ filter }}
                </button>
              </div>
              <div v-if="showFileMoreMenu" class="floating-menu file-more-menu" @click.stop>
                <button @click="selectedFiles = filteredFiles.map((file) => file.name); showFileMoreMenu = false"><CheckCircle2 :size="15" /> 全选</button>
                <button @click="selectedFiles = []; showFileMoreMenu = false"><X :size="15" /> 取消选择</button>
                <button @click="openApp('tasks'); showFileMoreMenu = false"><CalendarClock :size="15" /> 查看任务</button>
              </div>
            </div>
            <div v-if="selectedFiles.length" class="selection-bar">
              已选择 {{ selectedFiles.length }} 项
              <button @click="applyFileAction('download')"><CloudDownload :size="15" /> 下载</button>
              <button @click="applyFileAction('copy')"><Copy :size="15" /> 复制</button>
              <button @click="applyFileAction('move')"><Folder :size="15" /> 移动</button>
              <button @click="applyFileAction('share')"><Link2 :size="15" /> 共享</button>
              <button @click="applyFileAction('tag')"><Tags :size="15" /> 标签</button>
              <button @click="applyFileAction('delete')"><Trash2 :size="15" /> 删除</button>
            </div>
            <table>
              <colgroup>
                <col class="name-col" />
                <col class="size-col" />
                <col class="type-col" />
                <col class="date-col" />
                <col class="more-col" />
              </colgroup>
              <thead><tr><th>名称</th><th>大小</th><th>类型</th><th>修改日期</th><th></th></tr></thead>
              <tbody>
                <tr
                  v-for="file in filteredFiles"
                  :key="file.name"
                  :class="{ selected: selectedFiles.includes(file.name) }"
                  @click="toggleFile(file.name)"
                  @dblclick.stop="enterFile(file)"
                >
                  <td><component :is="file.icon" :size="20" /> {{ file.name }}</td>
                  <td>{{ file.size }}</td>
                  <td>{{ file.type }}</td>
                  <td>{{ file.date }}</td>
                  <td class="file-more">
                    <button @click.stop="fileMenuOpen = fileMenuOpen === file.name ? '' : file.name"><MoreVertical :size="17" /></button>
                    <div v-if="fileMenuOpen === file.name" class="row-menu">
                      <button @click.stop="applyFileAction('preview', file)"><Info :size="15" /> 查看详情</button>
                      <button @click.stop="applyFileAction('rename', file)"><Pencil :size="15" /> 重命名</button>
                      <button @click.stop="applyFileAction('download', file)"><CloudDownload :size="15" /> 下载</button>
                      <button @click.stop="applyFileAction('copy', file)"><Copy :size="15" /> 复制到</button>
                      <button @click.stop="applyFileAction('move', file)"><Folder :size="15" /> 移动到</button>
                      <button @click.stop="applyFileAction('share', file)"><Link2 :size="15" /> 共享链接</button>
                      <button @click.stop="applyFileAction('tag', file)"><Tags :size="15" /> 添加标签</button>
                      <button @click.stop="applyFileAction('delete', file)"><Trash2 :size="15" /> 删除</button>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
            <p v-if="filteredFiles.length === 0" class="table-empty">没有匹配的文件</p>
            <p class="count">共 {{ filteredFiles.length }} 项</p>
            <div v-if="isFileDragOver" class="drop-overlay">
              <Upload :size="36" />
              <strong>释放以上传到 {{ currentFileSpaceName() }}</strong>
            </div>
            <aside v-if="previewFile" class="preview-drawer" @click.stop>
              <header><strong>文件详情</strong><button @click="previewFile = null"><X :size="17" /></button></header>
              <span class="preview-icon"><component :is="previewFile.icon" :size="38" /></span>
              <h3>{{ previewFile.name }}</h3>
              <dl>
                <dt>类型</dt><dd>{{ previewFile.type }}</dd>
                <dt>大小</dt><dd>{{ previewFile.size }}</dd>
                <dt>修改日期</dt><dd>{{ previewFile.date }}</dd>
                <dt>路径</dt><dd>/{{ activeFileSpace }}/{{ previewFile.name }}</dd>
              </dl>
              <div class="tag-row"><span v-for="tag in previewFile.tags" :key="tag">{{ tag }}</span></div>
              <button @click="applyFileAction('download', previewFile)"><CloudDownload :size="16" /> 下载</button>
            </aside>
          </main>
        </div>

        <div v-else-if="win.id === 'app-center'" class="window-body app-center">
          <aside class="app-side">
            <label><Search :size="16" /><input v-model="query" placeholder="搜索" /></label>
            <button
              v-for="category in appCategories"
              :key="category"
              :class="{ active: activeCategory === category }"
              @click="activeCategory = category"
            >
              <LayoutGrid v-if="category === '全部应用'" :size="18" />
              <CloudDownload v-else-if="category === '已安装'" :size="18" />
              <span v-else></span>
              {{ category }}
            </button>
            <button class="settings-button" @click="showAppSettings = true"><Settings :size="19" /></button>
          </aside>
          <main class="app-main">
            <section class="community-banner">
              <div class="banner-copy">
                <span>海量教程 | 官方公告 | 干货分享 | 互助答疑 | 福利活动</span>
                <h2>绿联NAS用户玩家社区</h2>
                <p>加入绿联NAS用户高频互动阵营，轻松玩转NAS</p>
              </div>
              <div class="banner-visual" aria-hidden="true">
                <i class="cloud c1"></i>
                <i class="cloud c2"></i>
                <i class="cloud c3"></i>
                <i class="screen main-screen"></i>
                <i class="screen side-screen"></i>
                <i class="person left-person"></i>
                <i class="person right-person"></i>
                <i class="badge badge-heart"></i>
                <i class="badge badge-play"></i>
              </div>
            </section>
            <div class="app-head">
              <h2>{{ activeCategory }} <span>{{ displayedAppCount }}</span></h2>
              <button @click="activeCategory = activeCategory === '已安装' ? '全部应用' : '已安装'"><ListFilter :size="18" /></button>
            </div>
            <section v-if="appInstallTask" class="app-install-task">
              <span class="app-tile tiny" :style="{ background: appInstallTask.app.color }"><component :is="appInstallTask.app.icon" :size="18" /></span>
              <div>
                <strong>{{ appInstallTask.app.name }} · {{ appInstallTask.stage }}</strong>
                <span>{{ appInstallTask.mode === 'install' ? '安装' : '更新' }}进度 {{ appInstallTask.progress }}%</span>
                <i><em :style="{ width: appInstallTask.progress + '%' }"></em></i>
              </div>
              <button @click="advanceAppInstallTask">继续</button>
              <button @click="cancelAppInstallTask">取消</button>
            </section>
            <section class="app-list">
              <article v-for="app in appCenterApps" :key="app.id" @click="openAppDetail(app)">
                <span class="app-tile small" :style="{ background: app.color }"><component :is="app.icon" :size="24" /></span>
                <div>
                  <h3>{{ app.name }} <small v-if="installedApps[app.id]">已安装</small><small v-else>可安装</small></h3>
                  <p>{{ app.category }} · {{ app.description }}</p>
                </div>
                <button :class="{ update: appActionLabel(app) === '更新' }" @click.stop="installOrOpen(app)">{{ appActionLabel(app) }}</button>
              </article>
            </section>
            <p v-if="appCenterApps.length === 0" class="table-empty">没有匹配的应用</p>
          </main>
        </div>

        <div v-else class="window-body generic-app">
          <aside>
            <button
              v-for="tab in ['概览', '任务', '设置', '日志']"
              :key="tab"
              :class="{ active: (activeGenericTab[win.id] || '概览') === tab }"
              @click="activeGenericTab[win.id] = tab"
            >
              {{ tab }}
            </button>
          </aside>
          <main>
            <section class="generic-hero">
              <span class="app-tile" :style="{ background: appMap.get(win.id)?.color }"><component :is="appIcon(win.id)" :size="34" /></span>
              <div>
                <h2>{{ appName(win.id) }}</h2>
                <p>{{ appMap.get(win.id)?.description }}</p>
              </div>
              <button @click="activeGenericTab[win.id] = '设置'">打开设置</button>
            </section>
            <section class="metric-strip">
              <article v-for="metric in genericMetrics(win.id)" :key="metric">{{ metric }}</article>
            </section>
            <section v-if="win.id === 'tasks' || activeGenericTab[win.id] === '任务'" class="task-list">
              <article v-for="task in tasks" :key="task.name" :class="[`tone-${task.tone}`, task.status === '已暂停' ? 'paused' : '', task.status === '已取消' ? 'cancelled' : '']">
                <div><strong>{{ task.name }}</strong><span>{{ task.detail }}</span><small>{{ task.status }}</small></div>
                <div class="progress"><span :style="{ width: task.progress + '%' }"></span></div>
                <b>{{ task.progress }}%</b>
                <menu>
                  <button :disabled="task.status === '已取消'" @click="handleTaskAction(task.name, '暂停')">{{ task.status === '已暂停' ? '继续' : '暂停' }}</button>
                  <button :disabled="task.status === '已取消'" @click="handleTaskAction(task.name, '取消')">取消</button>
                </menu>
              </article>
            </section>
            <section v-else-if="activeGenericTab[win.id] === '设置'" class="settings-grid">
              <article>
                <h3>通用</h3>
                <label><span>开机自启动</span><input v-model="genericSetting(win.id).autostart" type="checkbox" /></label>
                <label><span>允许通知</span><input v-model="genericSetting(win.id).notifications" type="checkbox" /></label>
                <p>{{ genericSetting(win.id).autostart ? '随系统启动' : '手动启动' }} · {{ genericSetting(win.id).notifications ? '通知开启' : '静默运行' }}</p>
              </article>
              <article>
                <h3>资源</h3>
                <label><span>CPU 限制 {{ genericSetting(win.id).cpuLimit }}%</span><input v-model.number="genericSetting(win.id).cpuLimit" type="range" min="10" max="100" /></label>
                <label><span>内存限制 {{ genericSetting(win.id).memoryLimit }}%</span><input v-model.number="genericSetting(win.id).memoryLimit" type="range" min="10" max="100" /></label>
                <p>资源上限：CPU {{ genericSetting(win.id).cpuLimit }}% · 内存 {{ genericSetting(win.id).memoryLimit }}%</p>
              </article>
              <article>
                <h3>权限</h3>
                <label><span>访问共享文件夹</span><input v-model="genericSetting(win.id).sharedAccess" type="checkbox" /></label>
                <label><span>允许远程访问</span><input v-model="genericSetting(win.id).remoteAccess" type="checkbox" /></label>
                <p>{{ genericSetting(win.id).sharedAccess ? '可访问共享文件夹' : '隔离文件访问' }} · {{ genericSetting(win.id).remoteAccess ? '远程访问开启' : '仅局域网' }}</p>
              </article>
              <footer class="settings-actions">
                <button @click="resetGenericSetting(win.id)"><RefreshCcw :size="15" /> 推荐</button>
                <button class="primary" @click="saveGenericSetting(win.id, '应用配置')">保存</button>
              </footer>
            </section>
            <section v-else-if="activeGenericTab[win.id] === '日志'" class="log-list">
              <article><span>刚刚</span><strong>{{ appName(win.id) }} 已打开</strong><p>用户 hiveton 通过网页端启动应用。</p></article>
              <article><span>12:41</span><strong>任务状态同步</strong><p>已写入任务中心，并发送桌面通知。</p></article>
              <article><span>09:16</span><strong>权限检查通过</strong><p>当前账号具备访问此应用的权限。</p></article>
            </section>
            <section v-else class="feature-grid">
              <article v-for="card in appFeatureCards(win.id)" :key="card.title" role="button" tabindex="0" @click="runFeatureAction(win.id, card.title)">
                <component :is="card.icon" :size="20" />
                <h3>{{ card.title }}</h3>
                <p>{{ card.detail }}</p>
                <button>进入</button>
              </article>
            </section>
          </main>
        </div>
        <button v-if="!win.maximized" class="resize-handle" title="调整窗口大小" @pointerdown="beginResize($event, win)"></button>
      </article>
    </section>

    <section v-if="showHelpDialog" class="modal-scrim" @click="showHelpDialog = false; helpContextApp = null">
      <article class="system-modal help-modal" @click.stop>
        <header>
          <CircleHelp :size="22" />
          <strong>{{ helpContextApp ? `${appName(helpContextApp)}帮助` : '帮助中心' }}</strong>
          <button @click="showHelpDialog = false; helpContextApp = null"><X :size="18" /></button>
        </header>
        <p v-if="helpContextApp" class="help-context">{{ appMap.get(helpContextApp)?.description }}</p>
        <div class="help-grid">
          <article><Search :size="19" /><strong>搜索当前页面</strong><span>定位{{ helpContextApp ? appName(helpContextApp) : '当前页面' }}功能、设置项和常见问题。</span></article>
          <article><Globe2 :size="19" /><strong>远程访问</strong><span>UGREENlink、直连、中继和 HTTPS 说明。</span></article>
          <article><FileText :size="19" /><strong>上传诊断日志</strong><span>打包系统日志给技术支持。</span></article>
          <article><Shield :size="19" /><strong>安全建议</strong><span>检查登录风险、端口暴露和 OTP 状态。</span></article>
        </div>
        <footer>
          <button @click="openApp('help'); showHelpDialog = false; helpContextApp = null">打开帮助中心</button>
          <button class="primary" @click="pushNotification('帮助中心', `${helpContextApp ? appName(helpContextApp) : '当前页面'}诊断摘要已提交。`, CircleHelp, 'success'); showHelpDialog = false; helpContextApp = null">提交诊断</button>
        </footer>
      </article>
    </section>

    <section v-if="showWallpaperPanel" class="modal-scrim" @click="showWallpaperPanel = false">
      <article class="system-modal wallpaper-modal" @click.stop>
        <header><Image :size="22" /><strong>更换壁纸</strong><button @click="showWallpaperPanel = false"><X :size="18" /></button></header>
        <div class="wallpaper-options">
          <button :class="{ active: wallpaperTheme === 'space' }" @click="applyWallpaper('space')"><span class="swatch space"></span><strong>深空蓝</strong></button>
          <button :class="{ active: wallpaperTheme === 'aurora' }" @click="applyWallpaper('aurora')"><span class="swatch aurora"></span><strong>极光绿</strong></button>
          <button :class="{ active: wallpaperTheme === 'matrix' }" @click="applyWallpaper('matrix')"><span class="swatch matrix"></span><strong>数据流</strong></button>
        </div>
      </article>
    </section>

    <section v-if="showAppSettings" class="modal-scrim" @click="showAppSettings = false">
      <article class="system-modal app-settings-modal" @click.stop>
        <header><Settings :size="22" /><strong>应用中心设置</strong><button @click="showAppSettings = false"><X :size="18" /></button></header>
        <div class="settings-grid compact">
          <article>
            <h3>更新策略</h3>
            <label><span>自动检查更新</span><input v-model="appCenterSettings.autoCheck" type="checkbox" /></label>
            <label><span>仅在夜间安装</span><input v-model="appCenterSettings.nightInstall" type="checkbox" /></label>
            <label><span>安装时段</span><input v-model.number="appCenterSettings.updateWindow" type="range" min="0" max="2" /></label>
            <p>当前：{{ appCenterSettings.autoCheck ? '自动检查' : '手动检查' }} · {{ appUpdateWindowLabel() }}</p>
          </article>
          <article>
            <h3>来源与权限</h3>
            <label><span>安装前权限确认</span><input v-model="appCenterSettings.permissionConfirm" type="checkbox" /></label>
            <label><span>显示社区应用</span><input v-model="appCenterSettings.communityApps" type="checkbox" /></label>
            <label><span>接收 Beta 版本</span><input v-model="appCenterSettings.betaChannel" type="checkbox" /></label>
            <p>当前：{{ appCenterSettings.permissionConfirm ? '安装前确认权限' : '信任已安装来源' }} · {{ appCenterSettings.betaChannel ? 'Beta 通道' : '稳定通道' }}</p>
          </article>
        </div>
        <footer>
          <button @click="resetAppCenterSettings"><RefreshCcw :size="15" /> 恢复默认</button>
          <button @click="checkAppUpdatesNow"><RefreshCcw :size="15" /> 立即检查</button>
          <button class="primary" @click="saveAppCenterSettings">保存</button>
        </footer>
      </article>
    </section>

    <section v-if="showUploadDialog" class="modal-scrim" @click="showUploadDialog = false">
      <article class="system-modal upload-modal" @click.stop>
        <header><Upload :size="22" /><strong>上传到文件管理</strong><button @click="showUploadDialog = false"><X :size="18" /></button></header>
        <div class="drop-zone">
          <Upload :size="34" />
          <strong>拖拽文件到这里</strong>
          <span>当前目录：{{ activeFileSpace === 'personal' ? '个人文件夹' : activeFileSpace === 'shared' ? '共享文件夹' : activeFileSpace === 'users' ? '用户文件夹' : '标签' }}</span>
        </div>
        <footer>
          <button @click="showUploadDialog = false">取消</button>
          <button class="primary" @click="submitUpload">开始上传</button>
        </footer>
      </article>
    </section>

    <section v-if="showNewFolderDialog" class="modal-scrim" @click="showNewFolderDialog = false">
      <article class="system-modal folder-modal" @click.stop>
        <header><Folder :size="22" /><strong>新建文件夹</strong><button @click="showNewFolderDialog = false"><X :size="18" /></button></header>
        <label>文件夹名称<input v-model="newFolderName" autofocus @keyup.enter="submitNewFolder" /></label>
        <footer>
          <button @click="showNewFolderDialog = false">取消</button>
          <button class="primary" @click="submitNewFolder">创建</button>
        </footer>
      </article>
    </section>

    <section v-if="showRenameDialog" class="modal-scrim" @click="showRenameDialog = false; renameValue = ''; renameFile = null">
      <article class="system-modal rename-modal" @click.stop>
        <header><Pencil :size="22" /><strong>重命名</strong><button @click="showRenameDialog = false; renameValue = ''; renameFile = null"><X :size="18" /></button></header>
        <label>文件名称<input v-model="renameValue" autofocus @keyup.enter="submitRename" /></label>
        <p>当前对象：{{ renameFile?.name ?? selectedFiles[0] ?? '所选文件' }}</p>
        <footer>
          <button @click="showRenameDialog = false; renameValue = ''; renameFile = null">取消</button>
          <button class="primary" @click="submitRename">保存</button>
        </footer>
      </article>
    </section>

    <section v-if="showTagEditor" class="modal-scrim" @click="showTagEditor = false; tagEditorFile = null; tagDraft = ''">
      <article class="system-modal tag-modal" @click.stop>
        <header><Tags :size="22" /><strong>编辑标签</strong><button @click="showTagEditor = false; tagEditorFile = null; tagDraft = ''"><X :size="18" /></button></header>
        <p>对象：{{ tagEditorFile?.name ?? selectedFiles.join('、') }}</p>
        <label>标签<textarea v-model="tagDraft" rows="3" autofocus placeholder="用顿号、逗号或空格分隔" @keyup.ctrl.enter="submitTags"></textarea></label>
        <div class="tag-preview">
          <span v-for="tag in draftTags()" :key="tag">{{ tag }}</span>
          <small v-if="draftTags().length === 0">暂无标签</small>
        </div>
        <footer>
          <button @click="showTagEditor = false; tagEditorFile = null; tagDraft = ''">取消</button>
          <button class="primary" @click="submitTags">保存标签</button>
        </footer>
      </article>
    </section>

    <section v-if="fileOperationDialog" class="modal-scrim" @click="fileOperationDialog = null">
      <article class="system-modal operation-modal" @click.stop>
        <header>
          <component :is="fileOperationDialog.mode === 'copy' ? Copy : Folder" :size="22" />
          <strong>{{ fileOperationDialog.mode === 'copy' ? '复制到' : '移动到' }}</strong>
          <button @click="fileOperationDialog = null"><X :size="18" /></button>
        </header>
        <p>对象：{{ fileOperationDialog.target }}</p>
        <div class="destination-grid">
          <button
            v-for="space in operationDestinations"
            :key="space"
            :class="{ active: operationDestination === space }"
            @click="operationDestination = space"
          >
            <Folder :size="20" />
            <strong>{{ fileSpaceName(space) }}</strong>
            <span>/volume1/{{ space }}</span>
          </button>
        </div>
        <footer>
          <button @click="fileOperationDialog = null">取消</button>
          <button class="primary" @click="submitFileOperation">加入任务</button>
        </footer>
      </article>
    </section>

    <section v-if="showShareDialog" class="modal-scrim" @click="showShareDialog = false; shareFile = null">
      <article class="system-modal share-modal" @click.stop>
        <header><Link2 :size="22" /><strong>共享链接</strong><button @click="showShareDialog = false; shareFile = null"><X :size="18" /></button></header>
        <p>对象：{{ shareTargetName() }}</p>
        <div class="share-options">
          <article><span>权限</span><strong>仅查看</strong></article>
          <article><span>有效期</span><strong>7 天</strong></article>
          <article><span>访问码</span><strong>NAS6</strong></article>
        </div>
        <label class="share-link"><span>链接</span><input :value="shareLink()" readonly /></label>
        <footer>
          <button @click="showShareDialog = false; shareFile = null">取消</button>
          <button class="primary" @click="copyShareLink"><Copy :size="15" /> 复制链接</button>
        </footer>
      </article>
    </section>

    <section v-if="confirmDialog" class="modal-scrim" @click="confirmDialog = null">
      <article class="system-modal confirm-modal" :class="confirmDialog.tone" @click.stop>
        <header>
          <span class="confirm-icon"><component :is="confirmDialog.icon" :size="22" /></span>
          <strong>{{ confirmDialog.title }}</strong>
          <button @click="confirmDialog = null"><X :size="18" /></button>
        </header>
        <p>{{ confirmDialog.detail }}</p>
        <footer>
          <button @click="confirmDialog = null">取消</button>
          <button class="danger-confirm" @click="runConfirmAction">{{ confirmDialog.confirmText }}</button>
        </footer>
      </article>
    </section>

    <section v-if="appInstallDialog" class="modal-scrim" @click="appInstallDialog = null">
      <article class="system-modal app-install-modal" @click.stop>
        <header>
          <span class="app-tile small" :style="{ background: appInstallDialog.app.color }"><component :is="appInstallDialog.app.icon" :size="24" /></span>
          <strong>{{ appInstallDialog.mode === 'install' ? '安装应用' : '更新应用' }}</strong>
          <button @click="appInstallDialog = null"><X :size="18" /></button>
        </header>
        <p>{{ appInstallDialog.app.name }} 将写入系统应用目录，并按以下权限运行。</p>
        <div class="install-summary">
          <article><span>版本</span><strong>1.{{ appInstallDialog.app.id.length }}.6</strong></article>
          <article><span>包大小</span><strong>{{ appInstallDialog.mode === 'install' ? '86 MB' : '24 MB' }}</strong></article>
          <article><span>来源</span><strong>官方应用源</strong></article>
        </div>
        <section class="permission-list install-permissions">
          <strong>授权确认</strong>
          <label><input type="checkbox" checked /> <span><Folder :size="15" /> 访问共享文件夹</span></label>
          <label><input type="checkbox" checked /> <span><Bell :size="15" /> 发送桌面通知</span></label>
          <label><input type="checkbox" /> <span><Power :size="15" /> 开机自动启动</span></label>
        </section>
        <footer>
          <button @click="appInstallDialog = null">取消</button>
          <button class="primary" @click="confirmAppInstall">{{ appInstallDialog.mode === 'install' ? '确认安装' : '确认更新' }}</button>
        </footer>
      </article>
    </section>

    <section v-if="appDetail" class="modal-scrim" @click="appDetail = null">
      <article class="system-modal app-detail-modal" @click.stop>
        <header>
          <span class="app-tile small" :style="{ background: appDetail.color }"><component :is="appDetail.icon" :size="24" /></span>
          <strong>{{ appDetail.name }}</strong>
          <button @click="appDetail = null"><X :size="18" /></button>
        </header>
        <p>{{ appDetail.description }}</p>
        <div class="detail-grid">
          <article><span>分类</span><strong>{{ appDetail.category }}</strong></article>
          <article><span>状态</span><strong>{{ installedApps[appDetail.id] ? '已安装' : '未安装' }}</strong></article>
          <article><span>版本</span><strong>1.{{ appDetail.id.length }}.6</strong></article>
          <article><span>数据目录</span><strong>/volume1/@app/{{ appDetail.id }}</strong></article>
        </div>
        <section class="permission-list">
          <strong>权限</strong>
          <span><Shield :size="15" /> 访问共享文件夹</span>
          <span><Bell :size="15" /> 发送桌面通知</span>
          <span><FileText :size="15" /> 写入日志中心</span>
        </section>
        <div class="detail-actions">
          <button @click="appDetailAction(appDetail, '检查更新')"><RefreshCcw :size="15" /> 检查更新</button>
          <button @click="appDetailAction(appDetail, '打开权限审计')"><Shield :size="15" /> 权限审计</button>
          <button class="danger" @click="appDetailAction(appDetail, '卸载')"><Trash2 :size="15" /> 卸载</button>
        </div>
        <footer>
          <button @click="appDetail = null">关闭</button>
          <button class="primary" @click="installOrOpen(appDetail); appDetail = null">{{ appActionLabel(appDetail) }}</button>
        </footer>
      </article>
    </section>
  </main>
</template>
