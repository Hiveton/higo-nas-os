import SwiftUI

private enum DeviceSheetRoute: Identifiable {
    case metric(String, String, String, String, Tone)
    case notice(NotificationItem)
    case power

    var id: String {
        switch self {
        case .metric(let title, _, _, _, _): "metric-\(title)"
        case .notice(let item): "notice-\(item.id)"
        case .power: "power"
        }
    }
}

struct DeviceScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var route: DeviceSheetRoute?

    var body: some View {
        PageContainer {
            Button {
                route = .metric("NAS 设备", model.nasDevice.name, "运行时间 \(model.systemStatus.uptime)", "externaldrive.badge.wifi", model.systemStatus.online ? .green : .red)
            } label: {
                HiGoCard {
                    HStack(spacing: 16) {
                        Image(systemName: "externaldrive.badge.wifi")
                            .font(.system(size: 54))
                            .foregroundStyle(HiGoTheme.ink)
                            .frame(width: 96, height: 96)
                            .background(Color.gray.opacity(0.12))
                            .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                        VStack(alignment: .leading, spacing: 7) {
                            Text(model.nasDevice.name)
                                .font(.title3.weight(.bold))
                                .foregroundStyle(HiGoTheme.ink)
                            StatusPill(text: model.systemStatus.online ? "在线" : "离线", tone: model.systemStatus.online ? .green : .red)
                            Text("运行时间 \(model.systemStatus.uptime)")
                                .font(.subheadline)
                                .foregroundStyle(HiGoTheme.muted)
                            Text("远程访问  \(model.systemStatus.remoteEnabled ? "已开启" : "未开启")")
                                .font(.subheadline)
                                .foregroundStyle(Color.higoBlue)
                        }
                        Spacer()
                        Image(systemName: "chevron.right")
                            .foregroundStyle(.secondary)
                    }
                }
            }
            .buttonStyle(.plain)

            HStack(spacing: 10) {
                Button("重启", systemImage: "arrow.clockwise") { route = .power }
                    .buttonStyle(.bordered)
                NavigationLink {
                    ModuleScreen(moduleID: .remote)
                } label: {
                    Label("远程访问", systemImage: "network")
                }
                .buttonStyle(.borderedProminent)
            }

            SectionHeader(title: "资源状态")
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                Button {
                    route = .metric("CPU", "\(model.storage.cpu)%", "正常 · 近 5 分钟峰值 24%", "cpu", .green)
                } label: {
                    MetricCard(title: "CPU", value: "\(model.storage.cpu)%", detail: "正常", symbol: "cpu", tone: .green)
                }
                Button {
                    route = .metric("内存", "\(model.storage.memory)%", "5.6 GB / 16 GB", "memorychip", .blue)
                } label: {
                    MetricCard(title: "内存", value: "\(model.storage.memory)%", detail: "5.6 GB / 16 GB", symbol: "memorychip", tone: .blue)
                }
                Button {
                    route = .metric("网络", "↓ \(model.storage.networkDown)", "↑ \(model.storage.networkUp) · 局域网正常", "wifi", .blue)
                } label: {
                    MetricCard(title: "网络", value: "↓ \(model.storage.networkDown)", detail: "↑ \(model.storage.networkUp)", symbol: "wifi", tone: .blue)
                }
                Button {
                    route = .metric("硬盘", String(format: "%.1f TB", model.storage.diskUsedTB), String(format: "/ %.0f TB · SMART 正常", model.storage.diskTotalTB), "internaldrive", .green)
                } label: {
                    MetricCard(
                        title: "硬盘",
                        value: String(format: "%.1f TB", model.storage.diskUsedTB),
                        detail: String(format: "/ %.0f TB", model.storage.diskTotalTB),
                        symbol: "internaldrive",
                        tone: .green
                    )
                }
            }
            .buttonStyle(.plain)

            SectionHeader(title: "需要处理")
            let actionable = model.notifications.filter { $0.primaryAction != nil }
            if actionable.isEmpty {
                HiGoCard {
                    EmptyStateView(title: "暂无告警", message: "当前设备没有待处理风险。", symbol: "checkmark.seal.fill")
                }
            } else {
                ForEach(actionable) { item in
                    HiGoCard {
                        Button { route = .notice(item) } label: {
                            ActionRow(title: item.title, subtitle: item.message, symbol: item.tone == .red ? "exclamationmark.triangle.fill" : "bell.fill", tone: item.tone, trailing: item.primaryAction)
                        }
                        .buttonStyle(.plain)
                    }
                }
            }

            SectionHeader(title: "设备管理")
            HiGoCard {
                VStack(spacing: 12) {
                    NavigationLink { ModuleScreen(moduleID: .remote) } label: {
                        ActionRow(title: "远程访问", subtitle: "域名、设备绑定和隧道策略", symbol: "network", tone: .blue)
                    }
                    .buttonStyle(.plain)
                    NavigationLink { ModuleScreen(moduleID: .storage) } label: {
                        ActionRow(title: "存储管理", subtitle: "空间、硬盘、SMART 与快照", symbol: "internaldrive.fill", tone: .green)
                    }
                    .buttonStyle(.plain)
                    NavigationLink { ModuleScreen(moduleID: .security) } label: {
                        ActionRow(title: "安全中心", subtitle: "审计、风险和回滚记录", symbol: "shield.lefthalf.filled", tone: .red)
                    }
                    .buttonStyle(.plain)
                }
            }
        }
        .navigationTitle("设备")
        .sheet(item: $route) { route in
            switch route {
            case .metric(let title, let value, let detail, let symbol, let tone):
                DeviceMetricSheet(title: title, value: value, detail: detail, symbol: symbol, tone: tone)
            case .notice(let item):
                DeviceNoticeSheet(item: item)
            case .power:
                DevicePowerSheet()
            }
        }
    }
}

private struct DeviceMetricSheet: View {
    @Environment(\.dismiss) private var dismiss
    var title: String
    var value: String
    var detail: String
    var symbol: String
    var tone: Tone

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: title, subtitle: detail, symbol: symbol, tone: tone, trailing: value)
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "趋势", subtitle: "近 24 小时处于正常范围", symbol: "chart.line.uptrend.xyaxis", tone: .green, trailing: "正常")
                        ActionRow(title: "阈值", subtitle: "超过阈值会推送通知并写入审计", symbol: "bell.badge.fill", tone: .orange, trailing: "开启")
                        ActionRow(title: "诊断", subtitle: "可从设备管理进入更完整的检查流程", symbol: "stethoscope", tone: .blue)
                    }
                }
            }
            .navigationTitle(title)
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}

private struct DeviceNoticeSheet: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var model: AppViewModel
    var item: NotificationItem

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: item.title, subtitle: item.message, symbol: "exclamationmark.triangle.fill", tone: item.tone, trailing: item.primaryAction)
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "影响范围", subtitle: item.category.rawValue, symbol: "scope", tone: .blue)
                        ActionRow(title: "建议动作", subtitle: item.primaryAction ?? "查看详情", symbol: "checkmark.shield.fill", tone: .green)
                        ActionRow(title: "审计", subtitle: "处理结果将保留在安全中心", symbol: "doc.text.magnifyingglass", tone: .orange, trailing: "开启")
                    }
                }
                Button("完成处理", systemImage: "checkmark.circle.fill") {
                    model.recordMockCommand("处理设备告警", detail: "\(item.title) 已在本地 UI 中标记处理。")
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
            }
            .navigationTitle("告警处理")
        }
    }
}

private struct DevicePowerSheet: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var model: AppViewModel
    @State private var createSnapshot = true
    @State private var notifyMembers = true

    var body: some View {
        NavigationStack {
            Form {
                Section("电源操作") {
                    Label("重启 HiGoOS-NAS", systemImage: "arrow.clockwise")
                    Toggle("操作前创建配置快照", isOn: $createSnapshot)
                    Toggle("通知家庭成员", isOn: $notifyMembers)
                }
                Section("风险") {
                    Label("当前为 UI Mock，不会执行真实重启。", systemImage: "shippingbox.fill")
                }
            }
            .navigationTitle("重启确认")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("确认") {
                        model.recordMockCommand("确认重启", detail: "已记录重启确认，真实后端接入后会进入任务中心。")
                        dismiss()
                    }
                }
            }
        }
    }
}
