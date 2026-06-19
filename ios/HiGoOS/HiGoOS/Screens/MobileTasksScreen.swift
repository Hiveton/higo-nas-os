import SwiftUI

private enum TaskFilter: String, CaseIterable, Identifiable {
    case all = "全部"
    case running = "运行中"
    case failed = "失败"
    case needsConfirmation = "需确认"
    case completed = "已完成"

    var id: String { rawValue }

    var state: TaskState? {
        switch self {
        case .all: nil
        case .running: .running
        case .failed: .failed
        case .needsConfirmation: .needsConfirmation
        case .completed: .completed
        }
    }
}

struct MobileTasksScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var filter: TaskFilter = .all
    @State private var selectedTask: TaskItem?
    @State private var selectedConfirmation: ConfirmationRequest?

    private var visibleTasks: [TaskItem] {
        model.taskStore.visible(for: filter.state)
    }

    var body: some View {
        PageContainer {
            Picker("任务筛选", selection: $filter) {
                ForEach(TaskFilter.allCases) { Text($0.rawValue).tag($0) }
            }
            .pickerStyle(.segmented)
            .accessibilityIdentifier("task-filter")

            HiGoCard {
                HStack(spacing: 12) {
                    MetricInline(title: "运行中", value: "\(model.tasks.filter { $0.state == .running }.count)", tone: .blue)
                    MetricInline(title: "需确认", value: "\(model.confirmations.count)", tone: .orange)
                    MetricInline(title: "失败", value: "\(model.tasks.filter { $0.state == .failed }.count)", tone: .red)
                }
            }

            if !model.confirmations.isEmpty {
                SectionHeader(title: "确认队列", action: "\(model.confirmations.count)")
                ForEach(model.confirmations) { request in
                    Button { selectedConfirmation = request } label: {
                        ConfirmationRequestCard(request: request)
                    }
                    .buttonStyle(.plain)
                }
            }

            SectionHeader(title: "后台任务", action: "\(visibleTasks.count)")
            HiGoCard {
                if visibleTasks.isEmpty {
                    EmptyStateView(title: "暂无任务", message: "上传、下载、备份、转码和 AI 操作会显示在这里。", symbol: "checklist")
                } else {
                    VStack(spacing: 4) {
                        ForEach(visibleTasks) { task in
                            Button { selectedTask = task } label: {
                                TaskItemRow(task: task)
                            }
                            .buttonStyle(.plain)
                            if task.id != visibleTasks.last?.id { Divider().padding(.leading, 46) }
                        }
                    }
                }
            }

            SectionHeader(title: "下载队列", action: "\(model.downloads.count)")
            ForEach(model.downloads) { item in
                DownloadProgressCard(item: item)
            }

            SectionHeader(title: "系统通知", action: "\(model.notifications.count)")
            HiGoCard {
                VStack(spacing: 12) {
                    ForEach(model.notifications) { notice in
                        NotificationRow(item: notice)
                    }
                }
            }
        }
        .navigationTitle("任务")
        .sheet(item: $selectedTask) { task in
            TaskDetailSheet(task: task)
        }
        .sheet(item: $selectedConfirmation) { request in
            ConfirmationRequestSheet(request: request)
        }
    }
}

struct MetricInline: View {
    var title: String
    var value: String
    var tone: Tone

    var body: some View {
        VStack(spacing: 4) {
            Text(value)
                .font(.title3.weight(.bold))
                .foregroundStyle(tone.color)
            Text(title)
                .font(.caption)
                .foregroundStyle(HiGoTheme.muted)
        }
        .frame(maxWidth: .infinity)
    }
}

struct TaskItemRow: View {
    var task: TaskItem

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: task.symbol)
                .font(.headline)
                .foregroundStyle(task.state.tone.color)
                .frame(width: 36, height: 36)
                .background(task.state.tone.color.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            VStack(alignment: .leading, spacing: 7) {
                HStack {
                    Text(task.title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(HiGoTheme.ink)
                    Spacer()
                    Text(task.state.rawValue)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(task.state.tone.color)
                }
                Text("\(task.subtitle) · \(task.time)")
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.muted)
                if let progress = task.progress {
                    ProgressView(value: progress)
                        .tint(task.state.tone.color)
                }
            }
        }
        .padding(.vertical, 10)
    }
}

struct ConfirmationRequestCard: View {
    var request: ConfirmationRequest

    var body: some View {
        HiGoCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    StatusPill(text: request.risk == .red ? "高风险" : "需确认", tone: request.risk, symbol: "shield.lefthalf.filled")
                    Spacer()
                    Image(systemName: "chevron.right")
                        .font(.caption.weight(.bold))
                        .foregroundStyle(.secondary)
                }
                Text(request.title)
                    .font(.headline)
                    .foregroundStyle(HiGoTheme.ink)
                Text(request.target)
                    .font(.subheadline)
                    .foregroundStyle(HiGoTheme.muted)
                Text(request.preview)
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.ink)
            }
        }
    }
}

struct DownloadProgressCard: View {
    var item: DownloadItem

    var body: some View {
        HiGoCard {
            VStack(alignment: .leading, spacing: 10) {
                HStack {
                    Text(item.title)
                        .font(.subheadline.weight(.semibold))
                    Spacer()
                    StatusPill(text: item.state.rawValue, tone: item.state.tone)
                }
                Text(item.subtitle)
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.muted)
                ProgressView(value: item.progress)
                    .tint(item.state.tone.color)
            }
        }
    }
}

struct TaskDetailSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    var task: TaskItem

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    TaskItemRow(task: task)
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "类型", subtitle: task.kind.rawValue, symbol: task.symbol, tone: task.state.tone, trailing: task.state.rawValue)
                        ActionRow(title: "执行时间", subtitle: task.time, symbol: "clock.fill", tone: .gray)
                        ActionRow(title: "审计", subtitle: "任务变更会保留操作者、时间和结果", symbol: "doc.text.magnifyingglass", tone: .green, trailing: "开启")
                    }
                }
                HiGoCard {
                    HStack {
                        ModuleActionButton(title: task.state == .running ? "暂停" : "重试", detail: "\(task.title) 已进入任务操作确认。", isPrimary: true)
                        ModuleActionButton(title: "查看日志", detail: "\(task.title) 的日志已筛选。", isPrimary: false)
                    }
                }
            }
            .navigationTitle("任务详情")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}

struct ConfirmationRequestSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var notifyMembers: Bool
    @State private var writeAudit: Bool
    var request: ConfirmationRequest

    init(request: ConfirmationRequest) {
        self.request = request
        _notifyMembers = State(initialValue: request.notifyMembers)
        _writeAudit = State(initialValue: request.writeAudit)
    }

    var body: some View {
        NavigationStack {
            PageContainer {
                ConfirmationRequestCard(request: request)
                HiGoCard {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("执行确认")
                            .font(.headline)
                        Toggle("通知相关成员", isOn: $notifyMembers)
                        Toggle("写入审计日志", isOn: $writeAudit)
                        ActionRow(title: "回滚策略", subtitle: "执行前创建状态快照，可从安全中心回滚", symbol: "arrow.uturn.backward.circle.fill", tone: .blue, trailing: "支持")
                    }
                }
                HiGoCard {
                    HStack {
                        Button("拒绝") {
                            model.recordMockCommand("已拒绝", detail: request.title)
                            dismiss()
                        }
                        .buttonStyle(.bordered)
                        Button("确认执行") {
                            model.recordMockCommand("已确认", detail: "\(request.title) 已写入 Mock 审计。")
                            dismiss()
                        }
                        .buttonStyle(.borderedProminent)
                        .tint(request.risk.color)
                    }
                }
            }
            .navigationTitle("确认请求")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("关闭") { dismiss() }
                }
            }
        }
    }
}
