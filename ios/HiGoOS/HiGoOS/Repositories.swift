import Foundation

protocol HomeRepositoryProtocol {
    func loadFeatureCategories() async -> [FeatureCategory]
    func loadDeviceHealthSummary() async -> DeviceHealthSummary
    func loadMediaLibraries() async -> [MediaLibrarySummary]
}

protocol DeviceRepositoryProtocol {
    func loadSystemStatus() async -> SystemStatus
    func loadNasDevice() async -> NasDevice
    func loadStorageSummary() async -> StorageSummary
}

protocol FileRepositoryProtocol {
    func loadFiles() async -> [FileItem]
}

protocol MediaRepositoryProtocol {
    func loadPhotos() async -> [PhotoAsset]
}

protocol BackupRepositoryProtocol {
    func loadBackupJobs() async -> [BackupJob]
}

protocol NotificationRepositoryProtocol {
    func loadNotifications() async -> [NotificationItem]
}

protocol AssistantRepositoryProtocol {
    func loadMessages() async -> [AiMessage]
    func loadSearchResults() async -> [AiSearchResult]
    func loadConfirmationRequests() async -> [ConfirmationRequest]
}

protocol AiRepositoryProtocol: AssistantRepositoryProtocol {
}

protocol AdminRepositoryProtocol {
    func loadShares() async -> [ShareLink]
    func loadContainers() async -> [DockerContainer]
    func loadApps() async -> [AppCenterItem]
    func loadAudit() async -> [AuditEntry]
    func loadModules() async -> [FeatureModule]
}

protocol TaskRepositoryProtocol {
    func loadTasks() async -> [TaskItem]
    func loadDownloads() async -> [DownloadItem]
}

typealias HiGoRepositoryProtocol = DeviceRepositoryProtocol
    & HomeRepositoryProtocol
    & FileRepositoryProtocol
    & MediaRepositoryProtocol
    & BackupRepositoryProtocol
    & NotificationRepositoryProtocol
    & AssistantRepositoryProtocol
    & TaskRepositoryProtocol
    & AdminRepositoryProtocol

struct CommandFeedback: Identifiable {
    let id = UUID()
    let title: String
    let message: String
}

struct MockRepository: HiGoRepositoryProtocol {
    func loadFeatureCategories() async -> [FeatureCategory] {
        Self.featureCategoryFixtures()
    }

    func loadDeviceHealthSummary() async -> DeviceHealthSummary {
        DeviceHealthSummary(cpu: 18, memory: 35, diskTemperature: "39°C", network: "96 KB/s ↓", alerts: 1)
    }

    func loadMediaLibraries() async -> [MediaLibrarySummary] {
        [
            MediaLibrarySummary(id: "media-video", title: "影视中心", subtitle: "海报刮削 312 部 · 2 个转码任务", count: 312, symbol: "play.rectangle.fill", tone: .orange),
            MediaLibrarySummary(id: "media-music", title: "音乐中心", subtitle: "专辑 86 张 · 歌词待扫描", count: 1240, symbol: "music.note.list", tone: .blue),
            MediaLibrarySummary(id: "media-photos", title: "家庭相册", subtitle: "人物地点本地识别中", count: 3456, symbol: "photo.stack.fill", tone: .green)
        ]
    }

    func loadSystemStatus() async -> SystemStatus {
        SystemStatus(
            name: "我的 NAS",
            online: true,
            capacityUsedTB: 13.2,
            capacityTotalTB: 24,
            healthScore: 92,
            healthLabel: "优秀",
            uptime: "18 天 6 小时",
            remoteEnabled: true
        )
    }

    func loadNasDevice() async -> NasDevice {
        NasDevice(
            id: "nas-home",
            name: "HiGoOS-NAS",
            imageSystemName: "externaldrive.connected.to.line.below",
            version: "1.2.0",
            location: "客厅"
        )
    }

    func loadStorageSummary() async -> StorageSummary {
        StorageSummary(cpu: 18, memory: 35, networkDown: "96 KB/s", networkUp: "28 KB/s", diskUsedTB: 13.2, diskTotalTB: 24)
    }

    func loadFiles() async -> [FileItem] {
        Self.fileFixtures()
    }

    func loadPhotos() async -> [PhotoAsset] {
        [
            PhotoAsset(id: "photo-today", title: "今天", date: "5月20日", count: 38, symbol: "sun.max.fill", tone: .orange),
            PhotoAsset(id: "photo-yesterday", title: "昨天", date: "5月19日", count: 42, symbol: "mountain.2.fill", tone: .blue),
            PhotoAsset(id: "album-family", title: "家庭回忆", date: "智能相册", count: 1256, symbol: "person.2.fill", tone: .green),
            PhotoAsset(id: "album-videos", title: "视频", date: "自动转码", count: 312, symbol: "play.rectangle.fill", tone: .blue)
        ]
    }

    func loadBackupJobs() async -> [BackupJob] {
        Self.backupFixtures()
    }

    func loadNotifications() async -> [NotificationItem] {
        [
            NotificationItem(id: "notice-update", title: "系统更新可用", message: "HiGoOS 1.2.0 版本已发布", time: "10:30", category: .system, unread: true, tone: .blue, primaryAction: nil, secondaryAction: nil),
            NotificationItem(id: "notice-backup-complete", title: "照片备份完成", message: "已备份 1,256 张照片", time: "09:20", category: .backup, unread: false, tone: .green, primaryAction: nil, secondaryAction: nil),
            NotificationItem(id: "notice-backup-failed", title: "备份失败", message: "手机备份失败，请检查网络", time: "昨天", category: .backup, unread: true, tone: .red, primaryAction: "处理", secondaryAction: "审计"),
            NotificationItem(id: "notice-agent-cleanup", title: "智能整理建议", message: "发现 32 项相似照片待处理", time: "昨天", category: .agent, unread: true, tone: .blue, primaryAction: "处理", secondaryAction: "审计")
        ]
    }

    func loadMessages() async -> [AiMessage] {
        [
            AiMessage(id: "ai-user-ask-mountain", text: "找一下上周去山地拍的照片", time: "09:41", fromUser: true),
            AiMessage(id: "ai-assistant-local-index", text: "为你找到以下相关内容，已优先使用本地索引处理。", time: "09:41", fromUser: false)
        ]
    }

    func loadSearchResults() async -> [AiSearchResult] {
        [
            AiSearchResult(id: "ai-result-huangshan-photo", title: "黄山之旅_20240512.jpg", meta: "2024/05/12 · 4.2 MB", summary: "相册 / 黄山之旅", tags: ["预览"], icon: "photo.fill", tone: .blue),
            AiSearchResult(id: "ai-result-warranty-pdf", title: "HiGoOS NAS 保修单.pdf", meta: "PDF · 1.2 MB · 2024/04/18", summary: "官方保修单，包含设备序列号、保修期限与售后服务条款。", tags: ["保修单", "官方文档"], icon: "doc.richtext.fill", tone: .red)
        ]
    }

    func loadConfirmationRequests() async -> [ConfirmationRequest] {
        [
            ConfirmationRequest(id: "confirm-ai-rename", title: "确认相册整理 Agent 批量重命名", target: "32 个相似照片文件", risk: .orange, preview: "会按拍摄时间与地点生成新文件名，执行前保留回滚快照。", notifyMembers: true, writeAudit: true),
            ConfirmationRequest(id: "confirm-public-share", title: "公开分享合同扫描件", target: "共享空间 / 合同扫描件.pdf", risk: .red, preview: "检测到身份证号和签名，建议改为密码 + 3 天有效期。", notifyMembers: false, writeAudit: true)
        ]
    }

    func loadShares() async -> [ShareLink] {
        [
            ShareLink(id: "share-family-trip", title: "家庭旅行照片", expires: "2024/06/20", visits: 12, secure: true),
            ShareLink(id: "share-product-pack", title: "产品资料包", expires: "2024/06/10", visits: 3, secure: true),
            ShareLink(id: "share-work-report", title: "工作汇报文档", expires: "2024/05/30", visits: 2, secure: false)
        ]
    }

    func loadContainers() async -> [DockerContainer] {
        [
            DockerContainer(id: "container-jellyfin", name: "jellyfin", image: "jellyfin/jellyfin:latest", status: .running, cpu: 12, memory: 38),
            DockerContainer(id: "container-home-assistant", name: "home-assistant", image: "ghcr.io/home-assistant", status: .running, cpu: 8, memory: 22),
            DockerContainer(id: "container-photo-indexer", name: "photo-indexer", image: "higoos/indexer:dev", status: .stopped, cpu: 0, memory: 0)
        ]
    }

    func loadApps() async -> [AppCenterItem] {
        [
            AppCenterItem(id: "app-media-center", name: "影视中心", description: "媒体库、刮削、转码与直播", status: .installed, icon: "play.rectangle.fill"),
            AppCenterItem(id: "app-download-center", name: "下载中心", description: "BT、HTTP、磁力和自动归档", status: .running, icon: "arrow.down.circle.fill"),
            AppCenterItem(id: "app-family-photos", name: "家庭相册", description: "照片时间线和人物地点聚合", status: .updateAvailable, icon: "photo.on.rectangle.angled")
        ]
    }

    func loadAudit() async -> [AuditEntry] {
        [
            AuditEntry(id: "audit-agent-rename", actor: "Agent 文件管家", action: "申请批量重命名照片", time: "今天 12:30", result: .pending),
            AuditEntry(id: "audit-share-created", actor: "张小明", action: "创建家庭资料分享", time: "昨天 21:15", result: .success),
            AuditEntry(id: "audit-public-share-blocked", actor: "系统", action: "阻止公开分享", time: "昨天 09:12", result: .blocked)
        ]
    }

    func loadModules() async -> [FeatureModule] {
        Self.moduleFixtures()
    }

    func loadTasks() async -> [TaskItem] {
        Self.taskFixtures()
    }

    func loadDownloads() async -> [DownloadItem] {
        [
            DownloadItem(id: "download-ubuntu", title: "Ubuntu Server 镜像", subtitle: "下载完成后归档到 ISO 库", progress: 0.64, state: .running),
            DownloadItem(id: "download-family-video", title: "家庭旅行视频转存", subtitle: "等待自动归档到影视中心", progress: 0.18, state: .queued),
            DownloadItem(id: "download-magnet", title: "离线下载任务", subtitle: "连接超时，等待重试", progress: 0.22, state: .failed)
        ]
    }
}

@MainActor
final class AppViewModel: ObservableObject {
    private let repository: HiGoRepositoryProtocol

    @Published var systemStatus = SystemStatus(name: "我的 NAS", online: true, capacityUsedTB: 0, capacityTotalTB: 1, healthScore: 0, healthLabel: "加载中", uptime: "-", remoteEnabled: false)
    @Published var nasDevice = NasDevice(id: "nas-loading", name: "HiGoOS-NAS", imageSystemName: "externaldrive", version: "-", location: "-")
    @Published var storage = StorageSummary(cpu: 0, memory: 0, networkDown: "-", networkUp: "-", diskUsedTB: 0, diskTotalTB: 1)
    @Published var files: [FileItem] = []
    @Published var photos: [PhotoAsset] = []
    @Published var backupJobs: [BackupJob] = []
    @Published var notifications: [NotificationItem] = []
    @Published var aiMessages: [AiMessage] = []
    @Published var aiResults: [AiSearchResult] = []
    @Published var shares: [ShareLink] = []
    @Published var containers: [DockerContainer] = []
    @Published var apps: [AppCenterItem] = []
    @Published var audit: [AuditEntry] = []
    @Published var modules: [FeatureModule] = []
    @Published var featureCategories: [FeatureCategory] = []
    @Published var deviceHealth = DeviceHealthSummary(cpu: 0, memory: 0, diskTemperature: "-", network: "-", alerts: 0)
    @Published var mediaLibraries: [MediaLibrarySummary] = []
    @Published var tasks: [TaskItem] = []
    @Published var downloads: [DownloadItem] = []
    @Published var confirmations: [ConfirmationRequest] = []
    @Published var isLoaded = false
    @Published var feedback: CommandFeedback?

    init(repository: HiGoRepositoryProtocol) {
        self.repository = repository
    }

    static func previewLoaded() -> AppViewModel {
        let model = AppViewModel(repository: MockRepository())
        model.systemStatus = SystemStatus(
            name: "我的 NAS",
            online: true,
            capacityUsedTB: 13.2,
            capacityTotalTB: 24,
            healthScore: 92,
            healthLabel: "优秀",
            uptime: "18 天 6 小时",
            remoteEnabled: true
        )
        model.nasDevice = NasDevice(id: "nas-home", name: "HiGoOS-NAS", imageSystemName: "externaldrive.connected.to.line.below", version: "1.2.0", location: "客厅")
        model.storage = StorageSummary(cpu: 18, memory: 35, networkDown: "96 KB/s", networkUp: "28 KB/s", diskUsedTB: 13.2, diskTotalTB: 24)
        model.files = Array(MockRepository.fileFixtures().prefix(3))
        model.photos = [
            PhotoAsset(id: "photo-today", title: "今天", date: "5月20日", count: 38, symbol: "sun.max.fill", tone: .orange),
            PhotoAsset(id: "album-family", title: "家庭回忆", date: "智能相册", count: 1256, symbol: "person.2.fill", tone: .green),
            PhotoAsset(id: "album-videos", title: "视频", date: "自动转码", count: 312, symbol: "play.rectangle.fill", tone: .blue)
        ]
        model.backupJobs = Array(MockRepository.backupFixtures().prefix(2))
        model.notifications = [
            NotificationItem(id: "notice-update", title: "系统更新可用", message: "HiGoOS 1.2.0 版本已发布", time: "10:30", category: .system, unread: true, tone: .blue, primaryAction: nil, secondaryAction: nil),
            NotificationItem(id: "notice-backup-failed", title: "备份失败", message: "手机备份失败，请检查网络", time: "昨天", category: .backup, unread: true, tone: .red, primaryAction: "处理", secondaryAction: "审计")
        ]
        model.aiMessages = [
            AiMessage(id: "ai-user-ask-mountain", text: "找一下上周去山地拍的照片", time: "09:41", fromUser: true),
            AiMessage(id: "ai-assistant-local-index", text: "为你找到以下相关内容。", time: "09:41", fromUser: false)
        ]
        model.aiResults = [
            AiSearchResult(id: "ai-result-huangshan-photo", title: "黄山之旅_20240512.jpg", meta: "2024/05/12 · 4.2 MB", summary: "相册 / 黄山之旅", tags: ["预览"], icon: "photo.fill", tone: .blue)
        ]
        model.shares = [
            ShareLink(id: "share-family-trip", title: "家庭旅行照片", expires: "2024/06/20", visits: 12, secure: true)
        ]
        model.containers = [
            DockerContainer(id: "container-jellyfin", name: "jellyfin", image: "jellyfin/jellyfin:latest", status: .running, cpu: 12, memory: 38)
        ]
        model.apps = [
            AppCenterItem(id: "app-media-center", name: "影视中心", description: "媒体库、刮削、转码与直播", status: .installed, icon: "play.rectangle.fill")
        ]
        model.audit = [
            AuditEntry(id: "audit-agent-rename", actor: "Agent 文件管家", action: "申请批量重命名照片", time: "今天 12:30", result: .pending)
        ]
        model.modules = MockRepository.moduleFixtures()
        model.featureCategories = MockRepository.featureCategoryFixtures()
        model.deviceHealth = DeviceHealthSummary(cpu: 18, memory: 35, diskTemperature: "39°C", network: "96 KB/s ↓", alerts: 1)
        model.mediaLibraries = [
            MediaLibrarySummary(id: "media-video", title: "影视中心", subtitle: "海报刮削 312 部", count: 312, symbol: "play.rectangle.fill", tone: .orange)
        ]
        model.tasks = MockRepository.taskFixtures()
        model.downloads = [
            DownloadItem(id: "download-ubuntu", title: "Ubuntu Server 镜像", subtitle: "下载完成后归档到 ISO 库", progress: 0.64, state: .running)
        ]
        model.confirmations = [
            ConfirmationRequest(id: "confirm-ai-rename", title: "确认相册整理 Agent 批量重命名", target: "32 个相似照片文件", risk: .orange, preview: "执行前保留回滚快照。", notifyMembers: true, writeAudit: true)
        ]
        model.isLoaded = true
        return model
    }

    func load() async {
        async let status = repository.loadSystemStatus()
        async let device = repository.loadNasDevice()
        async let storage = repository.loadStorageSummary()
        async let files = repository.loadFiles()
        async let photos = repository.loadPhotos()
        async let backups = repository.loadBackupJobs()
        async let notifications = repository.loadNotifications()
        async let messages = repository.loadMessages()
        async let results = repository.loadSearchResults()
        async let shares = repository.loadShares()
        async let containers = repository.loadContainers()
        async let apps = repository.loadApps()
        async let audit = repository.loadAudit()
        async let modules = repository.loadModules()
        async let featureCategories = repository.loadFeatureCategories()
        async let deviceHealth = repository.loadDeviceHealthSummary()
        async let mediaLibraries = repository.loadMediaLibraries()
        async let tasks = repository.loadTasks()
        async let downloads = repository.loadDownloads()
        async let confirmations = repository.loadConfirmationRequests()

        self.systemStatus = await status
        self.nasDevice = await device
        self.storage = await storage
        self.files = await files
        self.photos = await photos
        self.backupJobs = await backups
        self.notifications = await notifications
        self.aiMessages = await messages
        self.aiResults = await results
        self.shares = await shares
        self.containers = await containers
        self.apps = await apps
        self.audit = await audit
        self.modules = await modules
        self.featureCategories = await featureCategories
        self.deviceHealth = await deviceHealth
        self.mediaLibraries = await mediaLibraries
        self.tasks = await tasks
        self.downloads = await downloads
        self.confirmations = await confirmations
        self.isLoaded = true
    }

    func recordMockCommand(_ title: String, detail: String = "当前为 UI Mock 模式，已记录本地操作反馈。") {
        feedback = CommandFeedback(title: title, message: detail)
    }
}

extension MockRepository {
    static func fileFixtures() -> [FileItem] {
        [
            FileItem(id: "folder-documents", name: "文档", kind: .folder, size: "120 项", date: "2024/05/20", space: .personal, tags: [], icon: "folder.fill", tone: .blue),
            FileItem(id: "folder-work", name: "工作资料", kind: .folder, size: "86 项", date: "2024/05/18", space: .personal, tags: ["项目"], icon: "folder.fill", tone: .blue),
            FileItem(id: "folder-family", name: "家庭资料", kind: .folder, size: "256 项", date: "2024/05/10", space: .family, tags: ["家庭"], icon: "folder.fill", tone: .green),
            FileItem(id: "folder-team", name: "团队共享", kind: .folder, size: "42 项", date: "2024/05/08", space: .team, tags: ["团队"], icon: "folder.fill", tone: .orange),
            FileItem(id: "file-product-requirements", name: "产品需求文档.pdf", kind: .pdf, size: "2.4 MB", date: "2024/05/20", space: .shared, tags: ["文档"], icon: "doc.richtext.fill", tone: .red),
            FileItem(id: "file-invoice-docx", name: "购买凭证与发票.docx", kind: .docx, size: "456 KB", date: "2024/04/18", space: .family, tags: ["发票", "订单"], icon: "doc.text.fill", tone: .blue),
            FileItem(id: "file-team-plan", name: "家庭媒体整理计划.xlsx", kind: .spreadsheet, size: "88 KB", date: "2024/05/12", space: .team, tags: ["协作"], icon: "tablecells.fill", tone: .green)
        ]
    }

    static func backupFixtures() -> [BackupJob] {
        [
            BackupJob(id: "backup-photos", title: "手机照片备份", subtitle: "正在备份 1,256 / 3,456 张", progress: 0.72, state: .running, statusText: nil),
            BackupJob(id: "backup-iphone", title: "我的 iPhone 15 Pro", subtitle: "正在备份 · Wi-Fi", progress: 0.72, state: .running, statusText: "72%"),
            BackupJob(id: "backup-ipad", title: "iPad Air", subtitle: "上次备份：昨天 22:30", progress: 1, state: .completed, statusText: nil),
            BackupJob(id: "backup-macbook", title: "MacBook Pro", subtitle: "上次备份：昨天 21:15", progress: 1, state: .completed, statusText: nil)
        ]
    }

    static func moduleFixtures() -> [FeatureModule] {
        [
            FeatureModule(id: .backup, title: "备份", subtitle: "照片、设备和计划", icon: "icloud.and.arrow.up.fill", tone: .blue),
            FeatureModule(id: .shares, title: "分享", subtitle: "链接、安全和访问", icon: "link.circle.fill", tone: .green),
            FeatureModule(id: .profile, title: "我的", subtitle: "账户、权限和模型", icon: "person.crop.circle.fill", tone: .blue),
            FeatureModule(id: .settings, title: "系统设置", subtitle: "网络、通知和更新", icon: "gearshape.fill", tone: .gray),
            FeatureModule(id: .storage, title: "存储管理", subtitle: "空间、硬盘和快照", icon: "internaldrive.fill", tone: .green),
            FeatureModule(id: .docker, title: "Docker", subtitle: "容器、镜像和日志", icon: "shippingbox.fill", tone: .blue),
            FeatureModule(id: .appCenter, title: "应用中心", subtitle: "安装、更新和权限", icon: "square.grid.2x2.fill", tone: .orange),
            FeatureModule(id: .security, title: "安全中心", subtitle: "风险、审计和回滚", icon: "shield.lefthalf.filled", tone: .red),
            FeatureModule(id: .remote, title: "远程访问", subtitle: "域名、设备和隧道", icon: "network", tone: .blue),
            FeatureModule(id: .tasks, title: "任务中心", subtitle: "下载、转码和扫描", icon: "checklist", tone: .green),
            FeatureModule(id: .protocols, title: "共享协议", subtitle: "SMB、NFS、WebDAV", icon: "server.rack", tone: .gray),
            FeatureModule(id: .musicCenter, title: "音乐中心", subtitle: "媒体库、歌词和专辑", icon: "music.note.list", tone: .blue),
            FeatureModule(id: .videoCenter, title: "影视中心", subtitle: "海报、字幕和转码", icon: "play.rectangle.fill", tone: .orange),
            FeatureModule(id: .downloadCenter, title: "下载中心", subtitle: "BT、HTTP、磁力和归档", icon: "arrow.down.circle.fill", tone: .green),
            FeatureModule(id: .hardwareCenter, title: "硬件中心", subtitle: "硬件清单、温度和风扇", icon: "cpu.fill", tone: .green),
            FeatureModule(id: .virtualMachine, title: "虚拟机", subtitle: "镜像、资源和控制台", icon: "desktopcomputer", tone: .orange),
            FeatureModule(id: .syncService, title: "同步服务", subtitle: "设备同步和冲突处理", icon: "arrow.triangle.2.circlepath", tone: .blue),
            FeatureModule(id: .iscsi, title: "iSCSI", subtitle: "Target、LUN 和 CHAP", icon: "externaldrive.badge.icloud", tone: .red)
        ]
    }

    static func taskFixtures() -> [TaskItem] {
        [
            TaskItem(id: "task-photo-backup", title: "手机照片备份", subtitle: "1,256 / 3,456 张 · 仅 Wi-Fi", kind: .backup, state: .running, progress: 0.72, time: "刚刚", symbol: "photo.on.rectangle.angled"),
            TaskItem(id: "task-video-transcode", title: "家庭旅行转码", subtitle: "4K -> 1080p 移动端预览", kind: .transcode, state: .running, progress: 0.38, time: "2 分钟前", symbol: "film.stack.fill"),
            TaskItem(id: "task-ai-confirm", title: "AI 批量重命名待确认", subtitle: "32 个相似照片文件，需管理员确认", kind: .ai, state: .needsConfirmation, progress: nil, time: "12:30", symbol: "sparkles"),
            TaskItem(id: "task-download-failed", title: "离线下载失败", subtitle: "连接超时，等待重试", kind: .download, state: .failed, progress: 0.22, time: "昨天", symbol: "arrow.down.circle.fill"),
            TaskItem(id: "task-system-update", title: "HiGoOS 1.2.0 更新检查", subtitle: "安全补丁已下载，等待安装", kind: .system, state: .queued, progress: nil, time: "10:30", symbol: "arrow.triangle.2.circlepath"),
            TaskItem(id: "task-docker-restart", title: "Jellyfin 容器健康检查", subtitle: "服务响应正常，延迟 28ms", kind: .docker, state: .completed, progress: 1, time: "09:48", symbol: "shippingbox.fill")
        ]
    }

    static func featureCategoryFixtures() -> [FeatureCategory] {
        let entries: [FeatureEntry] = moduleFixtures().map {
            FeatureEntry(id: $0.id, title: $0.title, subtitle: $0.subtitle, icon: $0.icon, tone: $0.tone, badge: badge(for: $0.id))
        }
        func pick(_ ids: [ModuleID]) -> [FeatureEntry] {
            ids.compactMap { id in entries.first { $0.id == id } }
        }
        return [
            FeatureCategory(id: .data, title: "数据管理", subtitle: "文件、备份、同步、分享", entries: pick([.backup, .shares, .syncService, .storage])),
            FeatureCategory(id: .media, title: "媒体娱乐", subtitle: "相册、影视、音乐、下载", entries: pick([.videoCenter, .musicCenter, .downloadCenter, .appCenter])),
            FeatureCategory(id: .operations, title: "设备运维", subtitle: "设备健康、任务、硬件和远程", entries: pick([.tasks, .hardwareCenter, .remote, .settings])),
            FeatureCategory(id: .security, title: "安全与成员", subtitle: "成员、权限、审计与回滚", entries: pick([.profile, .security, .protocols])),
            FeatureCategory(id: .advanced, title: "高级服务", subtitle: "Docker、虚拟机、iSCSI", entries: pick([.docker, .virtualMachine, .iscsi]))
        ]
    }

    private static func badge(for id: ModuleID) -> String? {
        switch id {
        case .security: "3"
        case .appCenter: "更新"
        case .tasks: "实时"
        case .docker: "2"
        case .downloadCenter: "运行"
        default: nil
        }
    }
}
