import Foundation

struct HomeStore {
    var modules: [FeatureModule]
    var categories: [FeatureCategory]
    var backupJobs: [BackupJob]
    var notifications: [NotificationItem]
    var tasks: [TaskItem]
    var mediaLibraries: [MediaLibrarySummary]
    var downloads: [DownloadItem]
    var deviceHealth: DeviceHealthSummary

    var quickModules: [FeatureModule] {
        modules.filter { [.backup, .shares, .videoCenter, .downloadCenter].contains($0.id) }
    }

    var todayTasks: ArraySlice<TaskItem> {
        tasks.prefix(3)
    }

    var unreadNotificationCount: Int {
        notifications.filter(\.unread).count
    }

    var riskCount: Int {
        tasks.filter { $0.state == .failed || $0.state == .needsConfirmation }.count + deviceHealth.alerts
    }
}

struct FilesStore {
    var files: [FileItem]

    func visibleFiles(in space: FileSpace) -> [FileItem] {
        files.filter { $0.space == space }
    }

    func folders(in space: FileSpace) -> [FileItem] {
        visibleFiles(in: space).filter(\.kind.isFolder)
    }

    func recentFiles(in space: FileSpace) -> [FileItem] {
        visibleFiles(in: space).filter { !$0.kind.isFolder }
    }
}

struct NotificationsStore {
    var notifications: [NotificationItem]

    var unreadCount: Int {
        notifications.filter(\.unread).count
    }

    func visible(for filter: NotificationFilter) -> [NotificationItem] {
        notifications.filter { item in
            filter == .all || (filter == .unread && item.unread) || item.category == filter.category
        }
    }
}

struct DeviceStore {
    var systemStatus: SystemStatus
    var nasDevice: NasDevice
    var storage: StorageSummary
}

struct AdminStore {
    var backupJobs: [BackupJob]
    var shares: [ShareLink]
    var containers: [DockerContainer]
    var apps: [AppCenterItem]
    var audit: [AuditEntry]
    var modules: [FeatureModule]
    var categories: [FeatureCategory]
}

struct TaskStore {
    var tasks: [TaskItem]
    var notifications: [NotificationItem]
    var confirmations: [ConfirmationRequest]
    var downloads: [DownloadItem]

    var attentionCount: Int {
        tasks.filter { $0.state == .failed || $0.state == .needsConfirmation }.count + confirmations.count
    }

    func visible(for state: TaskState?) -> [TaskItem] {
        guard let state else { return tasks }
        return tasks.filter { $0.state == state }
    }
}

extension AppViewModel {
    var homeStore: HomeStore {
        HomeStore(
            modules: modules,
            categories: featureCategories,
            backupJobs: backupJobs,
            notifications: notifications,
            tasks: tasks,
            mediaLibraries: mediaLibraries,
            downloads: downloads,
            deviceHealth: deviceHealth
        )
    }

    var filesStore: FilesStore {
        FilesStore(files: files)
    }

    var notificationsStore: NotificationsStore {
        NotificationsStore(notifications: notifications)
    }

    var deviceStore: DeviceStore {
        DeviceStore(systemStatus: systemStatus, nasDevice: nasDevice, storage: storage)
    }

    var adminStore: AdminStore {
        AdminStore(backupJobs: backupJobs, shares: shares, containers: containers, apps: apps, audit: audit, modules: modules, categories: featureCategories)
    }

    var taskStore: TaskStore {
        TaskStore(tasks: tasks, notifications: notifications, confirmations: confirmations, downloads: downloads)
    }
}
