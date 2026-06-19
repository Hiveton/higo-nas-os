import Foundation
import SwiftUI

enum Tone: String, CaseIterable, Identifiable, Codable {
    case blue, green, orange, red, gray

    var id: String { rawValue }

    var color: Color {
        switch self {
        case .blue: Color.higoBlue
        case .green: Color.higoGreen
        case .orange: Color.higoOrange
        case .red: Color.higoRed
        case .gray: HiGoTheme.muted
        }
    }
}

enum FileKind: String, CaseIterable, Identifiable, Codable {
    case folder = "文件夹"
    case pdf = "PDF"
    case docx = "DOCX"
    case spreadsheet = "XLSX"
    case image = "图片"

    var id: String { rawValue }
    var isFolder: Bool { self == .folder }
}

enum FileSpace: String, CaseIterable, Identifiable, Codable {
    case personal = "个人"
    case family = "家庭"
    case shared = "共享"
    case team = "团队"

    var id: String { rawValue }
}

enum BackupJobState: String, CaseIterable, Identifiable, Codable {
    case queued = "排队中"
    case running = "运行中"
    case paused = "暂停"
    case completed = "完成"
    case failed = "失败"

    var id: String { rawValue }

    var tone: Tone {
        switch self {
        case .queued: .gray
        case .running: .blue
        case .paused: .orange
        case .completed: .green
        case .failed: .red
        }
    }
}

enum NotificationCategory: String, CaseIterable, Identifiable, Codable {
    case system = "系统"
    case backup = "备份"
    case agent = "Agent"

    var id: String { rawValue }
}

enum DockerStatus: String, CaseIterable, Identifiable, Codable {
    case running = "运行中"
    case stopped = "已停止"

    var id: String { rawValue }
    var tone: Tone { self == .running ? .green : .gray }
}

enum AppInstallStatus: String, CaseIterable, Identifiable, Codable {
    case installed = "已安装"
    case running = "运行中"
    case updateAvailable = "可更新"

    var id: String { rawValue }

    var tone: Tone {
        switch self {
        case .installed: .blue
        case .running: .green
        case .updateAvailable: .orange
        }
    }
}

enum AuditResult: String, CaseIterable, Identifiable, Codable {
    case pending = "等待确认"
    case success = "成功"
    case blocked = "已拦截"

    var id: String { rawValue }

    var tone: Tone {
        switch self {
        case .pending: .orange
        case .success: .green
        case .blocked: .red
        }
    }
}

struct SystemStatus: Identifiable, Codable, Equatable {
    var id: String = "system"
    var name: String
    var online: Bool
    var capacityUsedTB: Double
    var capacityTotalTB: Double
    var healthScore: Int
    var healthLabel: String
    var uptime: String
    var remoteEnabled: Bool
}

struct NasDevice: Identifiable, Codable, Equatable {
    var id: String
    var name: String
    var imageSystemName: String
    var version: String
    var location: String
}

struct StorageSummary: Identifiable, Codable, Equatable {
    var id: String = "storage"
    var cpu: Int
    var memory: Int
    var networkDown: String
    var networkUp: String
    var diskUsedTB: Double
    var diskTotalTB: Double
}

struct FileItem: Identifiable, Codable, Equatable {
    var id: String
    var name: String
    var kind: FileKind
    var size: String
    var date: String
    var space: FileSpace
    var tags: [String]
    var icon: String
    var tone: Tone
}

struct PhotoAsset: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var date: String
    var count: Int
    var symbol: String
    var tone: Tone
}

struct BackupJob: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var progress: Double
    var state: BackupJobState
    var statusText: String?

    var displayStatus: String { statusText ?? state.rawValue }
    var tone: Tone { state.tone }
}

struct NotificationItem: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var message: String
    var time: String
    var category: NotificationCategory
    var unread: Bool
    var tone: Tone
    var primaryAction: String?
    var secondaryAction: String?
}

struct AiMessage: Identifiable, Codable, Equatable {
    var id: String
    var text: String
    var time: String
    var fromUser: Bool
}

struct AiSearchResult: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var meta: String
    var summary: String
    var tags: [String]
    var icon: String
    var tone: Tone
}

struct ShareLink: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var expires: String
    var visits: Int
    var secure: Bool
}

struct DockerContainer: Identifiable, Codable, Equatable {
    var id: String
    var name: String
    var image: String
    var status: DockerStatus
    var cpu: Int
    var memory: Int
}

struct AppCenterItem: Identifiable, Codable, Equatable {
    var id: String
    var name: String
    var description: String
    var status: AppInstallStatus
    var icon: String

    var tone: Tone { status.tone }
}

struct AuditEntry: Identifiable, Codable, Equatable {
    var id: String
    var actor: String
    var action: String
    var time: String
    var result: AuditResult

    var tone: Tone { result.tone }
}

struct FeatureModule: Identifiable, Codable, Equatable {
    let id: ModuleID
    var title: String
    var subtitle: String
    var icon: String
    var tone: Tone
}

enum ModuleID: String, Identifiable, CaseIterable, Codable {
    case backup, shares, profile, settings, storage, docker, appCenter, security, remote, tasks, protocols
    case musicCenter, videoCenter, downloadCenter, hardwareCenter, virtualMachine, syncService, iscsi

    var id: String { rawValue }
}

enum MobileTab: String, CaseIterable, Identifiable, Codable, Hashable {
    case home, files, photos, assistant, tasks, profile

    var id: String { rawValue }

    var title: String {
        switch self {
        case .home: "首页"
        case .files: "文件"
        case .photos: "相册"
        case .assistant: "助手"
        case .tasks: "任务"
        case .profile: "我的"
        }
    }

    var icon: String {
        switch self {
        case .home: "house.fill"
        case .files: "folder.fill"
        case .photos: "photo.fill"
        case .assistant: "sparkles"
        case .tasks: "checklist"
        case .profile: "person.crop.circle.fill"
        }
    }

    var accessibilityIdentifier: String { "tab-\(rawValue)" }
}

enum FeatureCategoryKind: String, CaseIterable, Identifiable, Codable {
    case data = "数据管理"
    case media = "媒体娱乐"
    case operations = "设备运维"
    case security = "安全与成员"
    case advanced = "高级服务"

    var id: String { rawValue }

    var symbol: String {
        switch self {
        case .data: "externaldrive.fill"
        case .media: "play.rectangle.fill"
        case .operations: "waveform.path.ecg"
        case .security: "shield.lefthalf.filled"
        case .advanced: "square.grid.2x2.fill"
        }
    }

    var tone: Tone {
        switch self {
        case .data: .blue
        case .media: .orange
        case .operations: .green
        case .security: .red
        case .advanced: .gray
        }
    }
}

struct FeatureCategory: Identifiable, Codable, Equatable {
    var id: FeatureCategoryKind
    var title: String
    var subtitle: String
    var entries: [FeatureEntry]
}

struct FeatureEntry: Identifiable, Codable, Equatable {
    var id: ModuleID
    var title: String
    var subtitle: String
    var icon: String
    var tone: Tone
    var badge: String?
}

enum TaskState: String, CaseIterable, Identifiable, Codable {
    case running = "运行中"
    case failed = "失败"
    case needsConfirmation = "需确认"
    case completed = "已完成"
    case queued = "排队中"

    var id: String { rawValue }

    var tone: Tone {
        switch self {
        case .running: .blue
        case .failed: .red
        case .needsConfirmation: .orange
        case .completed: .green
        case .queued: .gray
        }
    }
}

enum TaskKind: String, CaseIterable, Identifiable, Codable {
    case upload = "上传"
    case download = "下载"
    case backup = "备份"
    case transcode = "转码"
    case index = "索引"
    case ai = "AI"
    case docker = "Docker"
    case system = "系统"

    var id: String { rawValue }
}

struct TaskItem: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var kind: TaskKind
    var state: TaskState
    var progress: Double?
    var time: String
    var symbol: String
}

struct DeviceHealthSummary: Identifiable, Codable, Equatable {
    var id: String = "device-health"
    var cpu: Int
    var memory: Int
    var diskTemperature: String
    var network: String
    var alerts: Int
}

struct MediaLibrarySummary: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var count: Int
    var symbol: String
    var tone: Tone
}

struct DownloadItem: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var subtitle: String
    var progress: Double
    var state: TaskState
}

struct ConfirmationRequest: Identifiable, Codable, Equatable {
    var id: String
    var title: String
    var target: String
    var risk: Tone
    var preview: String
    var notifyMembers: Bool
    var writeAudit: Bool
}

extension NasDevice {
    init(name: String, imageSystemName: String, version: String, location: String) {
        self.init(id: name, name: name, imageSystemName: imageSystemName, version: version, location: location)
    }
}

extension FileItem {
    init(name: String, kind: String, size: String, date: String, space: String, tags: [String], icon: String, tone: Tone) {
        self.init(
            id: name,
            name: name,
            kind: FileKind(rawValue: kind) ?? .image,
            size: size,
            date: date,
            space: FileSpace(rawValue: space) ?? .personal,
            tags: tags,
            icon: icon,
            tone: tone
        )
    }
}

extension PhotoAsset {
    init(title: String, date: String, count: Int, symbol: String, tone: Tone) {
        self.init(id: title, title: title, date: date, count: count, symbol: symbol, tone: tone)
    }
}

extension BackupJob {
    init(title: String, subtitle: String, progress: Double, status: String, tone: Tone) {
        let state = BackupJobState(rawValue: status) ?? {
            switch tone {
            case .green: return .completed
            case .orange: return .paused
            case .red: return .failed
            case .gray: return .queued
            case .blue: return .running
            }
        }()
        self.init(id: title, title: title, subtitle: subtitle, progress: progress, state: state, statusText: status)
    }
}

extension NotificationItem {
    init(title: String, message: String, time: String, category: String, unread: Bool, tone: Tone, primaryAction: String?, secondaryAction: String?) {
        self.init(
            id: title,
            title: title,
            message: message,
            time: time,
            category: NotificationCategory(rawValue: category) ?? .system,
            unread: unread,
            tone: tone,
            primaryAction: primaryAction,
            secondaryAction: secondaryAction
        )
    }
}

extension AiMessage {
    init(text: String, time: String, fromUser: Bool) {
        self.init(id: text + time, text: text, time: time, fromUser: fromUser)
    }
}

extension AiSearchResult {
    init(title: String, meta: String, summary: String, tags: [String], icon: String, tone: Tone) {
        self.init(id: title, title: title, meta: meta, summary: summary, tags: tags, icon: icon, tone: tone)
    }
}

extension ShareLink {
    init(title: String, expires: String, visits: Int, secure: Bool) {
        self.init(id: title, title: title, expires: expires, visits: visits, secure: secure)
    }
}

extension DockerContainer {
    init(name: String, image: String, status: String, cpu: Int, memory: Int) {
        self.init(id: name, name: name, image: image, status: DockerStatus(rawValue: status) ?? .stopped, cpu: cpu, memory: memory)
    }
}

extension AppCenterItem {
    init(name: String, description: String, status: String, icon: String, tone: Tone) {
        self.init(id: name, name: name, description: description, status: AppInstallStatus(rawValue: status) ?? .installed, icon: icon)
    }
}

extension AuditEntry {
    init(actor: String, action: String, time: String, result: String, tone: Tone) {
        self.init(id: actor + action + time, actor: actor, action: action, time: time, result: AuditResult(rawValue: result) ?? .pending)
    }
}
