import SwiftUI

enum NotificationFilter: String, CaseIterable, Identifiable {
    case all = "全部"
    case unread = "未读"
    case system = "系统"
    case backup = "备份"
    case agent = "Agent"

    var id: String { rawValue }

    var category: NotificationCategory? {
        switch self {
        case .all, .unread: nil
        case .system: .system
        case .backup: .backup
        case .agent: .agent
        }
    }
}

private enum NotificationRoute: Identifiable {
    case detail(NotificationItem)
    case handling(NotificationItem)
    case audit(NotificationItem)

    var id: String {
        switch self {
        case .detail(let item): "detail-\(item.id)"
        case .handling(let item): "handling-\(item.id)"
        case .audit(let item): "audit-\(item.id)"
        }
    }
}

struct NotificationsScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var filter: NotificationFilter = .all
    @State private var readIDs: Set<String> = []
    @State private var handledIDs: Set<String> = []
    @State private var route: NotificationRoute?

    private var visible: [NotificationItem] {
        model.notifications.filter { item in
            let unread = item.unread && !readIDs.contains(item.id)
            let matchesFilter = filter == .all || (filter == .unread && unread) || item.category == filter.category
            return matchesFilter
        }
    }

    var body: some View {
        PageContainer {
            Picker("通知筛选", selection: $filter) {
                ForEach(NotificationFilter.allCases) { Text($0.rawValue).tag($0) }
            }
            .pickerStyle(.segmented)

            HStack(spacing: 10) {
                Button("全部已读", systemImage: "checkmark.circle") {
                    readIDs.formUnion(model.notifications.map(\.id))
                }
                .buttonStyle(.bordered)
                Button("只看待处理", systemImage: "line.3.horizontal.decrease.circle") {
                    filter = .unread
                }
                .buttonStyle(.bordered)
            }

            SectionHeader(title: filter == .all ? "通知" : filter.rawValue, action: "\(visible.count) 条")
            if visible.isEmpty {
                HiGoCard {
                    EmptyStateView(title: "暂无通知", message: "当前筛选下没有需要处理的消息。", symbol: "bell.slash")
                }
            } else {
                ForEach(visible) { item in
                    HiGoCard {
                        NotificationInteractiveRow(
                            item: item,
                            isRead: readIDs.contains(item.id) || !item.unread,
                            isHandled: handledIDs.contains(item.id),
                            onOpen: {
                                readIDs.insert(item.id)
                                route = .detail(item)
                            },
                            onHandle: {
                                readIDs.insert(item.id)
                                route = .handling(item)
                            },
                            onAudit: {
                                readIDs.insert(item.id)
                                route = .audit(item)
                            }
                        )
                    }
                }
            }
        }
        .navigationTitle("通知")
        .sheet(item: $route) { route in
            switch route {
            case .detail(let item):
                NotificationDetailSheet(item: item, isHandled: handledIDs.contains(item.id)) {
                    handledIDs.insert(item.id)
                }
            case .handling(let item):
                NotificationHandleSheet(item: item) {
                    handledIDs.insert(item.id)
                }
            case .audit(let item):
                NotificationAuditSheet(item: item)
            }
        }
    }
}

private struct NotificationInteractiveRow: View {
    var item: NotificationItem
    var isRead: Bool
    var isHandled: Bool
    var onOpen: () -> Void
    var onHandle: () -> Void
    var onAudit: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            Button(action: onOpen) {
                HStack(spacing: 12) {
                    Circle()
                        .fill(item.tone.color.opacity(0.15))
                        .frame(width: 34, height: 34)
                        .overlay {
                            Image(systemName: item.tone == .red ? "exclamationmark" : "bell.fill")
                                .font(.caption.weight(.bold))
                                .foregroundStyle(item.tone.color)
                        }
                    VStack(alignment: .leading, spacing: 4) {
                        HStack {
                            Text(item.title)
                                .font(.subheadline.weight(.semibold))
                                .foregroundStyle(HiGoTheme.ink)
                            if !isRead {
                                Circle().fill(Color.higoBlue).frame(width: 7, height: 7)
                            }
                            if isHandled {
                                StatusPill(text: "已处理", tone: .green)
                            }
                            Spacer()
                            Text(item.time)
                                .font(.caption)
                                .foregroundStyle(HiGoTheme.muted)
                        }
                        Text(item.message)
                            .font(.caption)
                            .foregroundStyle(HiGoTheme.muted)
                    }
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                }
            }
            .buttonStyle(.plain)

            if item.primaryAction != nil || item.secondaryAction != nil {
                HStack {
                    Spacer()
                    if let primary = item.primaryAction {
                        Button(primary, action: onHandle)
                            .buttonStyle(.borderedProminent)
                            .controlSize(.small)
                    }
                    if let secondary = item.secondaryAction {
                        Button(secondary, action: onAudit)
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                    }
                }
            }
        }
    }
}

private struct NotificationDetailSheet: View {
    @Environment(\.dismiss) private var dismiss
    var item: NotificationItem
    var isHandled: Bool
    var onHandle: () -> Void

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: item.title, subtitle: item.message, symbol: item.tone == .red ? "exclamationmark.triangle.fill" : "bell.fill", tone: item.tone, trailing: isHandled ? "已处理" : item.category.rawValue)
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "时间", subtitle: item.time, symbol: "clock.fill", tone: .gray)
                        ActionRow(title: "分类", subtitle: item.category.rawValue, symbol: "tray.full.fill", tone: .blue)
                        ActionRow(title: "状态", subtitle: isHandled ? "已完成处理闭环" : "等待用户查看或处理", symbol: isHandled ? "checkmark.circle.fill" : "hourglass", tone: isHandled ? .green : .orange)
                    }
                }
                if item.primaryAction != nil {
                    Button("进入处理", systemImage: "checkmark.shield.fill") {
                        onHandle()
                    }
                    .buttonStyle(.borderedProminent)
                }
            }
            .navigationTitle("通知详情")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}

private struct NotificationHandleSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var createAudit = true
    @State private var notifyFamily = false
    var item: NotificationItem
    var onComplete: () -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("处理项") {
                    Label(item.title, systemImage: item.tone == .red ? "exclamationmark.triangle.fill" : "sparkles")
                    Text(item.message)
                        .foregroundStyle(HiGoTheme.muted)
                }
                Section("操作") {
                    Toggle("写入审计日志", isOn: $createAudit)
                    Toggle("通知家庭管理员", isOn: $notifyFamily)
                }
                Section("结果") {
                    Label("当前为本地 UI 处理，不会调用后端。", systemImage: "shippingbox.fill")
                }
            }
            .navigationTitle("处理通知")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成处理") {
                        onComplete()
                        dismiss()
                    }
                }
            }
        }
    }
}

private struct NotificationAuditSheet: View {
    @Environment(\.dismiss) private var dismiss
    var item: NotificationItem

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: "审计记录", subtitle: item.title, symbol: "doc.text.magnifyingglass", tone: .blue, trailing: "可导出")
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "触发来源", subtitle: item.category.rawValue, symbol: "tray.full.fill", tone: .blue)
                        ActionRow(title: "风险级别", subtitle: item.tone == .red ? "高，需要处理" : "低，可记录", symbol: "shield.lefthalf.filled", tone: item.tone)
                        ActionRow(title: "保留周期", subtitle: "180 天", symbol: "archivebox.fill", tone: .green)
                    }
                }
            }
            .navigationTitle("通知审计")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}
