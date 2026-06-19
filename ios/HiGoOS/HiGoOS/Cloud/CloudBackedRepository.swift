import Foundation

// CloudBackedRepository is the production replacement for MockRepository: it
// implements the same HiGoRepositoryProtocol the app already consumes, but sources
// data from a bound NAS via NasGateway instead of fixtures.
//
// Each method maps a NAS server-go endpoint to the app's display model. Because
// the server-go DTOs differ from the app's presentation models (which carry
// SF Symbol names + Tone), the mapping lives here. Methods that are not yet mapped
// delegate to a MockRepository fallback so the app stays fully functional during
// incremental wiring — replace each fallback with a real mapping as the endpoints
// are confirmed against docs/api.md.
struct CloudBackedRepository: HiGoRepositoryProtocol {
    let gateway: NasGateway
    let fallback = MockRepository()

    // MARK: Device

    func loadSystemStatus() async -> SystemStatus {
        // Example real mapping: GET /api/v1/system/health (shape per docs/api.md).
        struct Health: Decodable {
            var name: String?
            var online: Bool?
            var uptime: String?
        }
        if let h = try? await gateway.get("/api/v1/system/health", as: Health.self) {
            var status = await fallback.loadSystemStatus()
            status.name = h.name ?? status.name
            status.online = h.online ?? status.online
            status.uptime = h.uptime ?? status.uptime
            return status
        }
        return await fallback.loadSystemStatus()
    }

    func loadNasDevice() async -> NasDevice {
        // GET /api/v1/system/identity → identity.Identity
        struct Identity: Decodable { var deviceId: String?; var model: String?; var version: String?; var hostname: String? }
        if let d = try? await gateway.get("/api/v1/system/identity", as: Identity.self) {
            return NasDevice(
                id: d.deviceId ?? "nas",
                name: d.hostname ?? d.model ?? "HiGoOS-NAS",
                imageSystemName: "externaldrive.connected.to.line.below",
                version: d.version ?? "-",
                location: d.model ?? "-"
            )
        }
        return await fallback.loadNasDevice()
    }

    func loadStorageSummary() async -> StorageSummary {
        // GET /api/v1/monitoring/metrics/current → monitoring.MetricsSnapshot{metrics:[{key,value,detail}]}
        struct Metric: Decodable { let key: String; let value: Double; let detail: String }
        struct Snapshot: Decodable { let metrics: [Metric] }
        if let snap = try? await gateway.get("/api/v1/monitoring/metrics/current", as: Snapshot.self) {
            func metric(_ key: String) -> Metric? { snap.metrics.first { $0.key == key } }
            var s = await fallback.loadStorageSummary()
            if let cpu = metric("cpu") { s.cpu = Int(cpu.value) }
            if let mem = metric("memory") { s.memory = Int(mem.value) }
            if let net = metric("network") { s.networkDown = net.detail }
            return s
        }
        return await fallback.loadStorageSummary()
    }

    func loadFiles() async -> [FileItem] {
        // GET /api/v1/files/tree → files.FileNode (root, with children)
        struct Node: Decodable {
            let id: String; let name: String; let type: String
            let space: String; let size: String; let tags: [String]?; let isDir: Bool
            let children: [Node]?
        }
        if let root = try? await gateway.get("/api/v1/files/tree", as: Node.self) {
            return (root.children ?? []).map { node in
                FileItem(
                    id: node.id,
                    name: node.name,
                    kind: node.isDir ? .folder : Self.fileKind(node.type, node.name),
                    size: node.size,
                    date: "",
                    space: Self.fileSpace(node.space),
                    tags: node.tags ?? [],
                    icon: node.isDir ? "folder.fill" : "doc.fill",
                    tone: node.isDir ? .blue : .gray
                )
            }
        }
        return await fallback.loadFiles()
    }

    // MARK: Domains — TODO: map each to its server-go endpoint via gateway.get(...)

    func loadPhotos() async -> [PhotoAsset] { await fallback.loadPhotos() }          // TODO: /api/v1/media/...
    func loadBackupJobs() async -> [BackupJob] { await fallback.loadBackupJobs() }   // TODO: /api/v1/backups
    func loadNotifications() async -> [NotificationItem] { await fallback.loadNotifications() }
    func loadMessages() async -> [AiMessage] { await fallback.loadMessages() }
    func loadSearchResults() async -> [AiSearchResult] { await fallback.loadSearchResults() }
    func loadShares() async -> [ShareLink] { await fallback.loadShares() }
    func loadContainers() async -> [DockerContainer] { await fallback.loadContainers() }
    func loadApps() async -> [AppCenterItem] { await fallback.loadApps() }
    func loadAudit() async -> [AuditEntry] { await fallback.loadAudit() }
    func loadModules() async -> [FeatureModule] { await fallback.loadModules() }

    // MARK: Mapping helpers

    /// Maps a server-go file type / filename extension to the app's FileKind.
    static func fileKind(_ type: String, _ name: String) -> FileKind {
        let lower = (type.isEmpty ? (name as NSString).pathExtension : type).lowercased()
        switch lower {
        case "pdf": return .pdf
        case "doc", "docx": return .docx
        case "xls", "xlsx", "spreadsheet": return .spreadsheet
        case "jpg", "jpeg", "png", "gif", "heic", "image": return .image
        default: return .pdf
        }
    }

    /// Maps a server-go space label (Chinese seed values) to the app's FileSpace.
    static func fileSpace(_ space: String) -> FileSpace {
        FileSpace(rawValue: space) ?? .shared
    }
}
