import SwiftUI

struct BackupScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            SectionHeader(title: "备份状态")
            HiGoCard {
                if let job = model.backupJobs.first {
                    ModuleNavigationRow(title: job.title, subtitle: job.subtitle, symbol: "photo.on.rectangle.angled", tone: job.tone, trailing: job.displayStatus) {
                        backupDetail(job)
                    }
                }
            }

            SectionHeader(title: "备份设备")
            HiGoCard {
                VStack(spacing: 14) {
                    ForEach(model.backupJobs.dropFirst()) { job in
                        ModuleNavigationRow(title: job.title, subtitle: job.subtitle, symbol: "iphone", tone: job.tone, trailing: job.displayStatus) {
                            backupDetail(job)
                        }
                    }
                }
            }

            SectionHeader(title: "备份计划")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "每日备份", subtitle: "22:00 · 仅 Wi-Fi", symbol: "clock.fill", tone: .blue, trailing: "已开启") {
                        ManagementDetailScreen(
                            title: "每日备份",
                            subtitle: "照片、设备配置和家庭资料增量备份",
                            symbol: "clock.fill",
                            tone: .blue,
                            status: "已开启",
                            metrics: [
                                DetailMetric(title: "窗口", value: "22:00", detail: "避开白天使用", tone: .blue),
                                DetailMetric(title: "网络", value: "Wi-Fi", detail: "蜂窝网络暂停", tone: .green)
                            ],
                            sections: [
                                DetailSection(title: "策略", rows: [
                                    DetailRow(title: "照片", subtitle: "原图和视频保留 EXIF，后台自动续传", symbol: "photo.stack.fill", tone: .blue, trailing: "开启"),
                                    DetailRow(title: "设备配置", subtitle: "桌面、权限和 App 配置每日快照", symbol: "gearshape.2.fill", tone: .green, trailing: "开启"),
                                    DetailRow(title: "失败处理", subtitle: "连续失败 3 次进入通知和审计", symbol: "exclamationmark.triangle.fill", tone: .orange, trailing: "自动")
                                ])
                            ],
                            actions: ["立即执行", "编辑计划"]
                        )
                    }
                    HStack {
                        ModuleActionButton(title: "暂停", detail: "备份任务将暂停并写入审计记录。", isPrimary: false)
                        ModuleActionButton(title: "立即备份", detail: "已进入备份任务命令边界；真实任务队列后续接入。", isPrimary: true)
                        ModuleActionButton(title: "查看记录", detail: "打开备份记录筛选结果。", isPrimary: false)
                    }
                }
            }
        }
        .navigationTitle("备份")
    }

    private func backupDetail(_ job: BackupJob) -> ManagementDetailScreen {
        ManagementDetailScreen(
            title: job.title,
            subtitle: job.subtitle,
            symbol: "icloud.and.arrow.up.fill",
            tone: job.tone,
            status: job.displayStatus,
            metrics: [
                DetailMetric(title: "进度", value: "\(Int(job.progress * 100))%", detail: "当前批次", tone: job.tone),
                DetailMetric(title: "策略", value: "Wi-Fi", detail: "后台续传", tone: .green)
            ],
            sections: [
                DetailSection(title: "备份内容", rows: [
                    DetailRow(title: "照片原图", subtitle: "保留原始文件、位置和时间信息", symbol: "photo.fill", tone: .blue, trailing: "开启"),
                    DetailRow(title: "视频", subtitle: "完成后生成移动端预览版本", symbol: "play.rectangle.fill", tone: .orange, trailing: "队列中"),
                    DetailRow(title: "审计", subtitle: "每次暂停、恢复和失败都会记录", symbol: "doc.text.magnifyingglass", tone: .green, trailing: "已记录")
                ])
            ],
            actions: ["继续备份", "查看记录"]
        )
    }
}

struct SharesScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            HiGoCard {
                ModuleNavigationRow(title: "安全", subtitle: "无风险分享链接", symbol: "checkmark.shield.fill", tone: .green) {
                    ManagementDetailScreen(
                        title: "分享安全",
                        subtitle: "统一检查公开链接、访问密码和过期时间",
                        symbol: "checkmark.shield.fill",
                        tone: .green,
                        status: "正常",
                        metrics: [
                            DetailMetric(title: "公开链接", value: "0", detail: "无高风险", tone: .green),
                            DetailMetric(title: "到期", value: "2", detail: "7 天内", tone: .orange)
                        ],
                        sections: [
                            DetailSection(title: "安全策略", rows: [
                                DetailRow(title: "默认密码", subtitle: "外部分享自动生成访问密码", symbol: "lock.fill", tone: .green, trailing: "开启"),
                                DetailRow(title: "到期提醒", subtitle: "分享过期前 24 小时发送通知", symbol: "bell.badge.fill", tone: .blue, trailing: "开启")
                            ])
                        ],
                        actions: ["扫描链接", "导出清单"]
                    )
                }
            }

            SectionHeader(title: "分享链接")
            ForEach(model.shares) { share in
                HiGoCard {
                    VStack(alignment: .leading, spacing: 12) {
                        ModuleNavigationRow(title: share.title, subtitle: "有效期至 \(share.expires) · 访问 \(share.visits) 次", symbol: "link", tone: share.secure ? .green : .orange, trailing: share.secure ? "安全" : "检查") {
                            ManagementDetailScreen(
                                title: share.title,
                                subtitle: "分享链接、访问记录和安全设置",
                                symbol: "link.circle.fill",
                                tone: share.secure ? .green : .orange,
                                status: share.secure ? "安全" : "需检查",
                                metrics: [
                                    DetailMetric(title: "访问", value: "\(share.visits)", detail: "累计次数", tone: .blue),
                                    DetailMetric(title: "有效期", value: share.expires, detail: "自动失效", tone: share.secure ? .green : .orange)
                                ],
                                sections: [
                                    DetailSection(title: "权限", rows: [
                                        DetailRow(title: "访问密码", subtitle: share.secure ? "已启用独立密码" : "建议补充访问密码", symbol: "key.fill", tone: share.secure ? .green : .orange, trailing: share.secure ? "开启" : "建议"),
                                        DetailRow(title: "下载权限", subtitle: "允许预览和下载，访问会进入审计", symbol: "arrow.down.circle.fill", tone: .blue, trailing: "允许")
                                    ])
                                ],
                                actions: ["复制链接", "取消分享"]
                            )
                        }
                        HStack {
                            ModuleActionButton(title: "复制链接", detail: "\(share.title) 的链接已在 Mock 模式中记录。", isPrimary: true)
                            ModuleActionButton(title: "取消分享", detail: "\(share.title) 将取消外部访问并写入审计。", isPrimary: false)
                        }
                    }
                }
            }
        }
        .navigationTitle("分享")
    }
}

struct ProfileScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            HiGoCard {
                HStack(spacing: 14) {
                    Image(systemName: "person.crop.circle.fill")
                        .font(.system(size: 58))
                        .foregroundStyle(Color.higoBlue)
                    VStack(alignment: .leading, spacing: 4) {
                        Text("张小明")
                            .font(.title3.weight(.bold))
                        Text("家庭管理员")
                            .font(.subheadline)
                            .foregroundStyle(HiGoTheme.muted)
                        Text("zhangxiaoming@higoos.com")
                            .font(.caption)
                            .foregroundStyle(HiGoTheme.muted)
                    }
                    Spacer()
                }
            }

            SectionHeader(title: "权限与隐私")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "成员管理", subtitle: "6 位成员", symbol: "person.2.fill") {
                        ManagementDetailScreen(title: "成员管理", subtitle: "家庭成员、访客和设备授权", symbol: "person.2.fill", tone: .blue, status: "6 位成员", metrics: [DetailMetric(title: "管理员", value: "2", detail: "可管理系统", tone: .blue), DetailMetric(title: "访客", value: "1", detail: "受限访问", tone: .orange)], sections: [DetailSection(title: "成员", rows: [DetailRow(title: "张小明", subtitle: "家庭管理员 · 全部权限", symbol: "person.crop.circle.fill", tone: .blue, trailing: "管理员"), DetailRow(title: "家庭成员", subtitle: "照片、文件和分享权限", symbol: "person.2.fill", tone: .green, trailing: "4 人"), DetailRow(title: "访客", subtitle: "仅可访问指定分享链接", symbol: "person.badge.key.fill", tone: .orange, trailing: "1 人")])], actions: ["邀请成员", "调整权限"])
                    }
                    ModuleNavigationRow(title: "权限管理", subtitle: "精细化控制", symbol: "lock.fill") {
                        ManagementDetailScreen(title: "权限管理", subtitle: "按空间、成员和设备控制访问范围", symbol: "lock.fill", tone: .blue, status: "正常", metrics: [DetailMetric(title: "空间", value: "3", detail: "个人/家庭/共享", tone: .blue), DetailMetric(title: "规则", value: "18", detail: "已生效", tone: .green)], sections: [DetailSection(title: "规则", rows: [DetailRow(title: "家庭资料", subtitle: "家庭成员可读写，访客不可见", symbol: "folder.fill", tone: .green, trailing: "读写"), DetailRow(title: "共享空间", subtitle: "创建外链前需要确认", symbol: "link.circle.fill", tone: .orange, trailing: "需确认")])], actions: ["新增规则", "审计权限"])
                    }
                    ModuleNavigationRow(title: "隐私设置", subtitle: "本地优先存储", symbol: "hand.raised.fill", tone: .green) {
                        ManagementDetailScreen(title: "隐私设置", subtitle: "本地索引、云端增强和数据保留策略", symbol: "hand.raised.fill", tone: .green, status: "本地优先", metrics: [DetailMetric(title: "本地", value: "优先", detail: "默认策略", tone: .green), DetailMetric(title: "云端", value: "确认", detail: "每次授权", tone: .orange)], sections: [DetailSection(title: "策略", rows: [DetailRow(title: "文件索引", subtitle: "文件名和语义索引保留在 NAS", symbol: "doc.text.magnifyingglass", tone: .green, trailing: "本地"), DetailRow(title: "照片识别", subtitle: "人物地点在本机模型处理", symbol: "person.crop.rectangle.stack", tone: .green, trailing: "本地")])], actions: ["查看策略", "清理缓存"])
                    }
                }
            }

            SectionHeader(title: "设备与管理")
            ForEach(model.featureCategories) { category in
                HiGoCard {
                    VStack(alignment: .leading, spacing: 12) {
                        HStack {
                            Image(systemName: category.id.symbol)
                                .foregroundStyle(category.id.tone.color)
                            VStack(alignment: .leading, spacing: 3) {
                                Text(category.title)
                                    .font(.headline)
                                Text(category.subtitle)
                                    .font(.caption)
                                    .foregroundStyle(HiGoTheme.muted)
                            }
                            Spacer()
                        }
                        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
                            ForEach(category.entries) { entry in
                                NavigationLink { ModuleScreen(moduleID: entry.id) } label: {
                                    FeatureTile(title: entry.title, symbol: entry.icon, tone: entry.tone)
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                }
            }

            SectionHeader(title: "模型策略")
            HiGoCard {
                VStack(alignment: .leading, spacing: 10) {
                    HStack {
                        Text("本地优先")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(Color.higoBlue)
                        Slider(value: .constant(0.62))
                        Text("云端增强")
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(HiGoTheme.muted)
                    }
                    ModuleNavigationRow(title: "审计日志", subtitle: model.audit.first?.action ?? "暂无记录", symbol: "doc.text.magnifyingglass", tone: .blue) {
                        ManagementDetailScreen(title: "审计日志", subtitle: "Agent、分享、权限和系统变更记录", symbol: "doc.text.magnifyingglass", tone: .blue, status: "\(model.audit.count) 条", sections: [DetailSection(title: "最近记录", rows: model.audit.map { DetailRow(title: $0.action, subtitle: "\($0.actor) · \($0.time)", symbol: "shield.lefthalf.filled", tone: $0.tone, trailing: $0.result.rawValue) })], actions: ["筛选日志", "导出"])
                    }
                }
            }

            SectionHeader(title: "设置")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "通用设置", subtitle: "语言、主题和通知", symbol: "gearshape.fill", tone: .gray) {
                        ManagementDetailScreen(title: "通用设置", subtitle: "移动端语言、主题、通知摘要和默认行为", symbol: "gearshape.fill", tone: .gray, status: "跟随系统", sections: [DetailSection(title: "偏好", rows: [DetailRow(title: "语言", subtitle: "简体中文", symbol: "character.bubble.fill", tone: .gray, trailing: "中文"), DetailRow(title: "外观", subtitle: "跟随系统，浅色卡片优先", symbol: "paintpalette.fill", tone: .blue, trailing: "自动")])], actions: ["保存设置"])
                    }
                    ModuleNavigationRow(title: "关于 HiGoOS", subtitle: "版本 \(model.nasDevice.version)", symbol: "info.circle.fill", tone: .blue) {
                        ManagementDetailScreen(title: "关于 HiGoOS", subtitle: "设备版本、许可和移动端构建信息", symbol: "info.circle.fill", tone: .blue, status: model.nasDevice.version, metrics: [DetailMetric(title: "系统", value: model.nasDevice.version, detail: "Mock 版本", tone: .blue), DetailMetric(title: "设备", value: model.nasDevice.location, detail: "当前位置", tone: .green)], sections: [DetailSection(title: "信息", rows: [DetailRow(title: "设备名", subtitle: model.nasDevice.name, symbol: model.nasDevice.imageSystemName, tone: .blue), DetailRow(title: "数据模式", subtitle: "本地 Mock Fixture 驱动", symbol: "shippingbox.fill", tone: .green, trailing: "UI 阶段")])])
                    }
                }
            }
        }
        .navigationTitle("我的")
    }
}

struct AllModulesScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var query = ""

    var body: some View {
        PageContainer {
            SearchInput(text: $query, placeholder: "搜索功能、管理项或协议", symbol: "magnifyingglass")
                .accessibilityIdentifier("all-modules-search")
            ForEach(filteredCategories) { category in
                SectionHeader(title: category.title, action: category.subtitle)
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                    ForEach(category.entries) { entry in
                        NavigationLink { ModuleScreen(moduleID: entry.id) } label: {
                            VStack(alignment: .leading, spacing: 12) {
                                HStack {
                                    Image(systemName: entry.icon)
                                        .font(.title2)
                                        .foregroundStyle(entry.tone.color)
                                    Spacer()
                                    if let badge = entry.badge {
                                        StatusPill(text: badge, tone: entry.tone)
                                    }
                                }
                                Text(entry.title)
                                    .font(.headline)
                                    .foregroundStyle(HiGoTheme.ink)
                                Text(entry.subtitle)
                                    .font(.caption)
                                    .foregroundStyle(HiGoTheme.muted)
                                    .lineLimit(2)
                            }
                            .padding()
                            .frame(maxWidth: .infinity, minHeight: 132, alignment: .leading)
                            .background(HiGoTheme.card)
                            .clipShape(RoundedRectangle(cornerRadius: HiGoTheme.compactCorner, style: .continuous))
                        }
                        .accessibilityIdentifier("module-\(entry.id.rawValue)")
                        .buttonStyle(.plain)
                    }
                }
            }
        }
        .navigationTitle("全部功能")
    }

    private var filteredCategories: [FeatureCategory] {
        let trimmed = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return model.featureCategories }
        return model.featureCategories.compactMap { category in
            let entries = category.entries.filter { entry in
                entry.title.localizedCaseInsensitiveContains(trimmed)
                    || entry.subtitle.localizedCaseInsensitiveContains(trimmed)
                    || entry.id.rawValue.localizedCaseInsensitiveContains(trimmed)
                    || category.title.localizedCaseInsensitiveContains(trimmed)
            }
            guard !entries.isEmpty else { return nil }
            return FeatureCategory(id: category.id, title: category.title, subtitle: category.subtitle, entries: entries)
        }
    }
}

struct ModuleScreen: View {
    var moduleID: ModuleID

    var body: some View {
        switch moduleID {
        case .backup: BackupScreen()
        case .shares: SharesScreen()
        case .profile: ProfileScreen()
        case .settings: SettingsModuleScreen()
        case .storage: StorageModuleScreen()
        case .docker: DockerModuleScreen()
        case .appCenter: AppCenterModuleScreen()
        case .security: SecurityModuleScreen()
        case .remote: RemoteModuleScreen()
        case .tasks: TasksModuleScreen()
        case .protocols: ProtocolsModuleScreen()
        case .musicCenter: GenericModuleScreen(moduleID: .musicCenter)
        case .videoCenter: GenericModuleScreen(moduleID: .videoCenter)
        case .downloadCenter: GenericModuleScreen(moduleID: .downloadCenter)
        case .hardwareCenter: GenericModuleScreen(moduleID: .hardwareCenter)
        case .virtualMachine: GenericModuleScreen(moduleID: .virtualMachine)
        case .syncService: GenericModuleScreen(moduleID: .syncService)
        case .iscsi: GenericModuleScreen(moduleID: .iscsi)
        }
    }
}

struct GenericModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel
    var moduleID: ModuleID

    private var entry: FeatureEntry {
        model.featureCategories.flatMap(\.entries).first { $0.id == moduleID }
            ?? FeatureEntry(id: moduleID, title: "功能", subtitle: "移动端管理入口", icon: "square.grid.2x2.fill", tone: .blue, badge: nil)
    }

    var body: some View {
        ManagementDetailScreen(
            title: entry.title,
            subtitle: entry.subtitle,
            symbol: entry.icon,
            tone: entry.tone,
            status: entry.badge ?? "就绪",
            metrics: metrics,
            sections: [
                DetailSection(title: "移动端能力", rows: rows),
                DetailSection(title: "确认与审计", rows: [
                    DetailRow(title: "操作确认", subtitle: "高风险操作进入统一确认请求", symbol: "checkmark.shield.fill", tone: .orange, trailing: "开启"),
                    DetailRow(title: "审计回滚", subtitle: "记录执行前后状态，支持安全中心回滚", symbol: "arrow.uturn.backward.circle.fill", tone: .green, trailing: "支持")
                ])
            ],
            actions: actions
        )
    }

    private var metrics: [DetailMetric] {
        switch moduleID {
        case .videoCenter:
            return [DetailMetric(title: "影片", value: "312", detail: "已入库", tone: .orange), DetailMetric(title: "转码", value: "2", detail: "队列中", tone: .blue)]
        case .musicCenter:
            return [DetailMetric(title: "歌曲", value: "1240", detail: "已索引", tone: .blue), DetailMetric(title: "歌词", value: "86%", detail: "匹配率", tone: .green)]
        case .downloadCenter:
            return [DetailMetric(title: "运行", value: "1", detail: "下载任务", tone: .green), DetailMetric(title: "失败", value: "1", detail: "待重试", tone: .red)]
        case .hardwareCenter:
            return [DetailMetric(title: "CPU", value: "\(model.deviceHealth.cpu)%", detail: "实时", tone: .green), DetailMetric(title: "硬盘", value: model.deviceHealth.diskTemperature, detail: "最高温", tone: .orange)]
        case .virtualMachine:
            return [DetailMetric(title: "虚拟机", value: "0", detail: "未运行", tone: .gray), DetailMetric(title: "镜像", value: "2", detail: "可用", tone: .blue)]
        case .syncService:
            return [DetailMetric(title: "设备", value: "3", detail: "已绑定", tone: .blue), DetailMetric(title: "冲突", value: "0", detail: "当前", tone: .green)]
        case .iscsi:
            return [DetailMetric(title: "Target", value: "1", detail: "配置中", tone: .orange), DetailMetric(title: "风险", value: "高", detail: "需确认", tone: .red)]
        default:
            return []
        }
    }

    private var rows: [DetailRow] {
        switch moduleID {
        case .videoCenter:
            return [
                DetailRow(title: "媒体库", subtitle: "电影、剧集、直播和录制统一管理", symbol: "film.stack.fill", tone: .orange, trailing: "已建"),
                DetailRow(title: "刮削", subtitle: "自动匹配海报、字幕和简介", symbol: "sparkles", tone: .blue, trailing: "运行中"),
                DetailRow(title: "播放器", subtitle: "移动端继续观看，支持转码预览", symbol: "play.circle.fill", tone: .green, trailing: "可用")
            ]
        case .musicCenter:
            return [
                DetailRow(title: "音乐库", subtitle: "专辑、艺术家、文件夹和歌词", symbol: "music.note.list", tone: .blue, trailing: "已索引"),
                DetailRow(title: "播放队列", subtitle: "移动端收藏和最近播放", symbol: "play.fill", tone: .green, trailing: "同步")
            ]
        case .downloadCenter:
            return [
                DetailRow(title: "新建任务", subtitle: "HTTP、BT、磁力和订阅下载", symbol: "plus.circle.fill", tone: .green, trailing: "支持"),
                DetailRow(title: "自动归档", subtitle: "完成后移动到影视、音乐或文件空间", symbol: "archivebox.fill", tone: .blue, trailing: "开启")
            ]
        case .hardwareCenter:
            return [
                DetailRow(title: "硬件清单", subtitle: "CPU、内存、网卡、风扇和传感器", symbol: "cpu.fill", tone: .green, trailing: "实时"),
                DetailRow(title: "系统日志", subtitle: "按来源和风险筛选日志", symbol: "doc.text.magnifyingglass", tone: .gray, trailing: "12 条")
            ]
        case .virtualMachine:
            return [
                DetailRow(title: "创建向导", subtitle: "选择镜像、CPU、内存、磁盘和网络", symbol: "desktopcomputer", tone: .orange, trailing: "可配置"),
                DetailRow(title: "控制台", subtitle: "移动端查看状态，复杂操作建议桌面端", symbol: "terminal.fill", tone: .gray, trailing: "只读")
            ]
        case .syncService:
            return [
                DetailRow(title: "设备同步", subtitle: "按需同步、双向同步和冲突处理", symbol: "arrow.triangle.2.circlepath", tone: .blue, trailing: "运行"),
                DetailRow(title: "冲突策略", subtitle: "保留两份、覆盖或手动确认", symbol: "exclamationmark.triangle.fill", tone: .orange, trailing: "手动")
            ]
        case .iscsi:
            return [
                DetailRow(title: "Target", subtitle: "Target、LUN、CHAP 和用户组", symbol: "externaldrive.badge.icloud", tone: .red, trailing: "高风险"),
                DetailRow(title: "连接审计", subtitle: "启动器连接和容量变更写入审计", symbol: "shield.lefthalf.filled", tone: .orange, trailing: "开启")
            ]
        default:
            return []
        }
    }

    private var actions: [String] {
        switch moduleID {
        case .videoCenter: ["创建媒体库", "扫描"]
        case .musicCenter: ["扫描音乐", "匹配歌词"]
        case .downloadCenter: ["新建下载", "重试失败"]
        case .hardwareCenter: ["运行诊断", "导出日志"]
        case .virtualMachine: ["新建虚拟机", "导入镜像"]
        case .syncService: ["新增同步", "处理冲突"]
        case .iscsi: ["新建 Target", "查看审计"]
        default: ["查看详情"]
        }
    }
}

struct DetailMetric: Identifiable {
    var id: String { title }
    var title: String
    var value: String
    var detail: String
    var tone: Tone
}

struct DetailRow: Identifiable {
    var id: String { title + subtitle }
    var title: String
    var subtitle: String
    var symbol: String
    var tone: Tone = .blue
    var trailing: String? = nil
}

struct DetailSection: Identifiable {
    var id: String { title }
    var title: String
    var rows: [DetailRow]
}

private enum DetailInspectorRoute: Identifiable {
    case metric(DetailMetric)
    case row(DetailRow)

    var id: String {
        switch self {
        case .metric(let metric): "metric-\(metric.id)"
        case .row(let row): "row-\(row.id)"
        }
    }
}

struct ManagementDetailScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var selectedInspector: DetailInspectorRoute?
    var title: String
    var subtitle: String
    var symbol: String
    var tone: Tone
    var status: String
    var metrics: [DetailMetric] = []
    var sections: [DetailSection] = []
    var actions: [String] = []

    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(title: title, subtitle: subtitle, symbol: symbol, tone: tone, trailing: status)
            }

            if !metrics.isEmpty {
                LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                    ForEach(metrics) { metric in
                        Button {
                            selectedInspector = .metric(metric)
                        } label: {
                            MetricCard(title: metric.title, value: metric.value, detail: metric.detail, symbol: "chart.bar.fill", tone: metric.tone)
                        }
                        .buttonStyle(.plain)
                        .accessibilityIdentifier("detail-metric-\(metric.id)")
                    }
                }
            }

            ForEach(sections) { section in
                SectionHeader(title: section.title)
                HiGoCard {
                    VStack(spacing: 12) {
                        ForEach(section.rows) { row in
                            Button {
                                selectedInspector = .row(row)
                            } label: {
                                ActionRow(title: row.title, subtitle: row.subtitle, symbol: row.symbol, tone: row.tone, trailing: row.trailing)
                            }
                            .buttonStyle(.plain)
                            .accessibilityIdentifier("detail-row-\(row.id)")
                        }
                    }
                }
            }

            if !actions.isEmpty {
                SectionHeader(title: "操作")
                HiGoCard {
                    HStack {
                        ForEach(Array(actions.enumerated()), id: \.offset) { index, action in
                            let detail = "\(title) 已记录本地 Mock 操作，后续接入真实后端。"
                            ModuleActionButton(title: action, detail: detail, isPrimary: index == 0)
                        }
                    }
                }
            }
        }
        .navigationTitle(title)
        .sheet(item: $selectedInspector) { route in
            DetailInspectorSheet(parentTitle: title, route: route)
                .environmentObject(model)
        }
    }
}

private struct DetailInspectorSheet: View {
    @Environment(\.dismiss) private var dismiss
    var parentTitle: String
    var route: DetailInspectorRoute

    private var title: String {
        switch route {
        case .metric(let metric): metric.title
        case .row(let row): row.title
        }
    }

    private var subtitle: String {
        switch route {
        case .metric(let metric): "\(metric.value) · \(metric.detail)"
        case .row(let row): row.subtitle
        }
    }

    private var symbol: String {
        switch route {
        case .metric: "chart.bar.fill"
        case .row(let row): row.symbol
        }
    }

    private var tone: Tone {
        switch route {
        case .metric(let metric): metric.tone
        case .row(let row): row.tone
        }
    }

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: title, subtitle: subtitle, symbol: symbol, tone: tone, trailing: currentStatus)
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "归属模块", subtitle: parentTitle, symbol: "square.grid.2x2.fill", tone: .blue, trailing: "当前")
                        ActionRow(title: "状态来源", subtitle: "本地 Fixture 驱动，保留 Repository 替换边界", symbol: "shippingbox.fill", tone: .green, trailing: "Mock")
                        ActionRow(title: "审计策略", subtitle: "配置、启停和高风险操作都会写入审计流", symbol: "doc.text.magnifyingglass", tone: .orange, trailing: "开启")
                    }
                }
                HiGoCard {
                    HStack {
                        ModuleActionButton(title: primaryAction, detail: "\(parentTitle) / \(title) 已进入配置确认流程。", isPrimary: true)
                        ModuleActionButton(title: "查看审计", detail: "\(parentTitle) / \(title) 的审计记录已筛选。", isPrimary: false)
                    }
                }
            }
            .navigationTitle("详情")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("关闭") { dismiss() }
                }
            }
        }
        .presentationDetents([.medium, .large])
    }

    private var currentStatus: String {
        switch route {
        case .metric(let metric): metric.value
        case .row(let row): row.trailing ?? "详情"
        }
    }

    private var primaryAction: String {
        if title.contains("权限") || title.contains("策略") || title.contains("配置") { return "编辑配置" }
        if title.contains("日志") || title.contains("审计") { return "导出" }
        if title.contains("扫描") || title.contains("检查") { return "立即检查" }
        return "调整"
    }
}

struct ModuleActionButton: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var showingOperation = false
    var title: String
    var detail: String
    var isPrimary: Bool

    var body: some View {
        Group {
            if isPrimary {
                Button(title) {
                    showingOperation = true
                }
                .buttonStyle(.borderedProminent)
            } else {
                Button(title) {
                    showingOperation = true
                }
                .buttonStyle(.bordered)
            }
        }
        .sheet(isPresented: $showingOperation) {
            ModuleOperationSheet(title: title, detail: detail, isPrimary: isPrimary) {
                model.recordMockCommand(title, detail: detail)
            }
        }
    }
}

struct ModuleOperationSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var target = "当前项目"
    @State private var reason = ""
    @State private var dryRunFirst = true
    @State private var notifyMembers = true
    @State private var writeAudit = true
    var title: String
    var detail: String
    var isPrimary: Bool
    var onConfirm: () -> Void

    private var requiresConfirmation: Bool {
        ["删除", "取消", "重启", "轮换", "恢复", "暂停", "确认", "拒绝", "清理", "更新", "安装"].contains { title.contains($0) }
    }

    private var tone: Tone {
        if ["删除", "取消", "拒绝", "清理"].contains(where: { title.contains($0) }) { return .red }
        if requiresConfirmation { return .orange }
        return isPrimary ? .blue : .green
    }

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: title, subtitle: detail, symbol: requiresConfirmation ? "shield.lefthalf.filled" : "checkmark.circle.fill", tone: tone, trailing: requiresConfirmation ? "需确认" : "可执行")
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "执行方式", subtitle: "当前为本地 UI Mock，真实执行后续通过 Repository 接入", symbol: "shippingbox.fill", tone: .blue, trailing: "Mock")
                        ActionRow(title: "审计记录", subtitle: "操作会保留操作者、时间和影响范围", symbol: "doc.text.magnifyingglass", tone: .green, trailing: "开启")
                        ActionRow(title: "回滚策略", subtitle: requiresConfirmation ? "高影响操作会要求后端返回回滚点" : "低风险操作保留状态快照", symbol: "arrow.uturn.backward.circle.fill", tone: requiresConfirmation ? .orange : .green)
                    }
                }
                HiGoCard {
                    VStack(alignment: .leading, spacing: 14) {
                        Text("执行参数")
                            .font(.headline)
                        TextField("影响范围", text: $target)
                            .textFieldStyle(.roundedBorder)
                        TextField("备注原因", text: $reason, axis: .vertical)
                            .lineLimit(2...4)
                            .textFieldStyle(.roundedBorder)
                        Toggle("先预演再执行", isOn: $dryRunFirst)
                        Toggle("通知相关成员", isOn: $notifyMembers)
                        Toggle("写入审计日志", isOn: $writeAudit)
                    }
                }
                Button(requiresConfirmation ? "确认\(title)" : "执行\(title)") {
                    onConfirm()
                    dismiss()
                }
                .buttonStyle(.borderedProminent)
                .tint(tone.color)
            }
            .navigationTitle("操作确认")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
            }
        }
    }
}

struct ModuleNavigationRow<Destination: View>: View {
    var title: String
    var subtitle: String
    var symbol: String
    var tone: Tone = .blue
    var trailing: String? = nil
    @ViewBuilder var destination: () -> Destination

    var body: some View {
        NavigationLink {
            destination()
        } label: {
            ActionRow(title: title, subtitle: subtitle, symbol: symbol, tone: tone, trailing: trailing)
        }
        .buttonStyle(.plain)
    }
}

struct DockerModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            SectionHeader(title: "容器")
            ForEach(model.containers) { container in
                HiGoCard {
                    ModuleNavigationRow(title: container.name, subtitle: "\(container.image) · CPU \(container.cpu)% · 内存 \(container.memory)%", symbol: "shippingbox.fill", tone: container.status.tone, trailing: container.status.rawValue) {
                        ManagementDetailScreen(
                            title: container.name,
                            subtitle: container.image,
                            symbol: "shippingbox.fill",
                            tone: container.status.tone,
                            status: container.status.rawValue,
                            metrics: [
                                DetailMetric(title: "CPU", value: "\(container.cpu)%", detail: "近 5 分钟", tone: container.status.tone),
                                DetailMetric(title: "内存", value: "\(container.memory)%", detail: "容器配额", tone: .blue)
                            ],
                            sections: [
                                DetailSection(title: "运行配置", rows: [
                                    DetailRow(title: "镜像", subtitle: container.image, symbol: "shippingbox.fill", tone: .blue, trailing: "latest"),
                                    DetailRow(title: "网络", subtitle: "bridge · 端口映射已启用", symbol: "network", tone: .green, trailing: "正常"),
                                    DetailRow(title: "卷挂载", subtitle: "/volume1/appdata/\(container.name)", symbol: "externaldrive.fill", tone: .blue, trailing: "读写")
                                ]),
                                DetailSection(title: "最近日志", rows: [
                                    DetailRow(title: "健康检查", subtitle: "容器响应正常，延迟 28ms", symbol: "checkmark.seal.fill", tone: .green, trailing: "刚刚"),
                                    DetailRow(title: "资源采样", subtitle: "CPU 和内存处于移动端展示阈值内", symbol: "chart.line.uptrend.xyaxis", tone: .blue, trailing: "1 分钟前")
                                ])
                            ],
                            actions: container.status == .running ? ["重启", "查看日志"] : ["启动", "查看日志"]
                        )
                    }
                }
            }
            SectionHeader(title: "日志与资源")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "资源总览", subtitle: "3 个容器 · CPU 20% · 内存 60%", symbol: "chart.bar.xaxis", tone: .blue, trailing: "正常") {
                        ManagementDetailScreen(title: "Docker 资源总览", subtitle: "容器资源、网络流量和卷使用情况", symbol: "chart.bar.xaxis", tone: .blue, status: "正常", metrics: [DetailMetric(title: "容器", value: "\(model.containers.count)", detail: "已配置", tone: .blue), DetailMetric(title: "运行中", value: "\(model.containers.filter { $0.status == .running }.count)", detail: "实时状态", tone: .green)], sections: [DetailSection(title: "资源", rows: model.containers.map { DetailRow(title: $0.name, subtitle: "CPU \($0.cpu)% · 内存 \($0.memory)%", symbol: "shippingbox.fill", tone: $0.status.tone, trailing: $0.status.rawValue) })], actions: ["刷新状态", "查看日志"])
                    }
                    ModuleNavigationRow(title: "日志中心", subtitle: "容器启动、健康检查和错误日志", symbol: "terminal.fill", tone: .gray, trailing: "12 条") {
                        ManagementDetailScreen(title: "日志中心", subtitle: "按容器聚合的运行日志和审计记录", symbol: "terminal.fill", tone: .gray, status: "12 条", sections: [DetailSection(title: "最近日志", rows: [DetailRow(title: "jellyfin", subtitle: "Library scan completed", symbol: "play.rectangle.fill", tone: .green, trailing: "刚刚"), DetailRow(title: "home-assistant", subtitle: "Automation reload finished", symbol: "house.fill", tone: .blue, trailing: "2 分钟前"), DetailRow(title: "photo-indexer", subtitle: "等待手动启动", symbol: "photo.stack.fill", tone: .gray, trailing: "已停止")])], actions: ["导出日志", "清理"])
                    }
                }
            }
        }
        .navigationTitle("Docker")
    }
}

struct StorageModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(
                    title: "主存储池",
                    subtitle: String(format: "%.1f TB / %.0f TB · 健康评分 %d", model.storage.diskUsedTB, model.storage.diskTotalTB, model.systemStatus.healthScore),
                    symbol: "internaldrive.fill",
                    tone: .green,
                    trailing: "正常"
                )
            }

            SectionHeader(title: "硬盘与 SMART")
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                MetricCard(title: "硬盘 1", value: "良好", detail: "温度 38°C", symbol: "checkmark.seal.fill", tone: .green)
                MetricCard(title: "硬盘 2", value: "良好", detail: "温度 39°C", symbol: "checkmark.seal.fill", tone: .green)
                MetricCard(title: "硬盘 3", value: "关注", detail: "建议巡检", symbol: "exclamationmark.triangle.fill", tone: .orange)
                MetricCard(title: "快照", value: "12", detail: "最近 2 小时前", symbol: "clock.arrow.circlepath", tone: .blue)
            }

            SectionHeader(title: "操作")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "创建快照", subtitle: "为家庭资料卷创建可回滚快照", symbol: "camera.aperture", tone: .blue, trailing: "需确认") {
                        ManagementDetailScreen(title: "创建快照", subtitle: "选择卷、保留周期和回滚策略", symbol: "camera.aperture", tone: .blue, status: "需确认", sections: [DetailSection(title: "快照设置", rows: [DetailRow(title: "目标卷", subtitle: "家庭资料 · 256 项", symbol: "folder.fill", tone: .blue, trailing: "已选"), DetailRow(title: "保留", subtitle: "7 天自动清理，可手动锁定", symbol: "clock.fill", tone: .green, trailing: "7 天"), DetailRow(title: "审计", subtitle: "创建和删除都会写入安全中心", symbol: "shield.lefthalf.filled", tone: .orange, trailing: "开启")])], actions: ["创建快照", "调整保留"])
                    }
                    ModuleNavigationRow(title: "SMART 扫描", subtitle: "检查磁盘健康并生成审计记录", symbol: "waveform.path.ecg", tone: .green) {
                        ManagementDetailScreen(title: "SMART 扫描", subtitle: "硬盘健康、温度和坏道巡检", symbol: "waveform.path.ecg", tone: .green, status: "可执行", metrics: [DetailMetric(title: "硬盘", value: "3", detail: "在线", tone: .green), DetailMetric(title: "预计", value: "18m", detail: "快速扫描", tone: .blue)], sections: [DetailSection(title: "扫描范围", rows: [DetailRow(title: "硬盘 1", subtitle: "温度 38°C · 健康良好", symbol: "checkmark.seal.fill", tone: .green, trailing: "良好"), DetailRow(title: "硬盘 3", subtitle: "建议巡检，扫描后生成报告", symbol: "exclamationmark.triangle.fill", tone: .orange, trailing: "关注")])], actions: ["开始扫描", "查看报告"])
                    }
                    ModuleNavigationRow(title: "修复任务", subtitle: "发现异常时进入确认流程", symbol: "wrench.and.screwdriver.fill", tone: .orange) {
                        ManagementDetailScreen(title: "修复任务", subtitle: "针对存储异常的确认、执行和回滚流程", symbol: "wrench.and.screwdriver.fill", tone: .orange, status: "待确认", sections: [DetailSection(title: "任务", rows: [DetailRow(title: "校验文件系统", subtitle: "只读检查，不修改数据", symbol: "checklist", tone: .blue, trailing: "安全"), DetailRow(title: "重建索引", subtitle: "修复文件索引和缩略图缓存", symbol: "arrow.triangle.2.circlepath", tone: .green, trailing: "可执行")])], actions: ["创建任务", "查看审计"])
                    }
                }
            }
        }
        .navigationTitle("存储管理")
    }
}

struct SettingsModuleScreen: View {
    var body: some View {
        PageContainer {
            SectionHeader(title: "系统")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "网络设置", subtitle: "LAN、DNS、远程访问基础配置", symbol: "network", tone: .blue) {
                        ManagementDetailScreen(title: "网络设置", subtitle: "局域网地址、DNS、网关和远程访问前置项", symbol: "network", tone: .blue, status: "正常", metrics: [DetailMetric(title: "LAN", value: "1GbE", detail: "已连接", tone: .green), DetailMetric(title: "DNS", value: "2", detail: "主备配置", tone: .blue)], sections: [DetailSection(title: "配置", rows: [DetailRow(title: "IP 地址", subtitle: "192.168.1.101 · DHCP 保留", symbol: "network", tone: .green, trailing: "固定"), DetailRow(title: "DNS", subtitle: "223.5.5.5 / 119.29.29.29", symbol: "globe", tone: .blue, trailing: "正常"), DetailRow(title: "远程访问", subtitle: "开启前会检查端口和安全策略", symbol: "lock.shield.fill", tone: .orange, trailing: "已检查")])], actions: ["保存配置", "网络诊断"])
                    }
                    ModuleNavigationRow(title: "通知设置", subtitle: "系统、备份、Agent 告警策略", symbol: "bell.badge.fill", tone: .orange) {
                        ManagementDetailScreen(title: "通知设置", subtitle: "按类型控制提醒、摘要和免打扰窗口", symbol: "bell.badge.fill", tone: .orange, status: "3 类开启", sections: [DetailSection(title: "类型", rows: [DetailRow(title: "系统", subtitle: "更新、硬盘和安全告警即时提醒", symbol: "gearshape.fill", tone: .blue, trailing: "开启"), DetailRow(title: "备份", subtitle: "完成摘要，失败即时提醒", symbol: "icloud.and.arrow.up.fill", tone: .green, trailing: "开启"), DetailRow(title: "Agent", subtitle: "需要确认的建议进入通知", symbol: "sparkles", tone: .orange, trailing: "开启")]), DetailSection(title: "免打扰", rows: [DetailRow(title: "夜间", subtitle: "23:00-08:00 仅保留高风险告警", symbol: "moon.fill", tone: .gray, trailing: "开启")])], actions: ["保存设置"])
                    }
                    ModuleNavigationRow(title: "系统更新", subtitle: "HiGoOS 1.2.0 可用", symbol: "arrow.triangle.2.circlepath", tone: .blue, trailing: "可更新") {
                        ManagementDetailScreen(title: "系统更新", subtitle: "版本检查、更新包校验和回滚策略", symbol: "arrow.triangle.2.circlepath", tone: .blue, status: "可更新", metrics: [DetailMetric(title: "当前", value: "1.1.8", detail: "稳定版", tone: .gray), DetailMetric(title: "最新", value: "1.2.0", detail: "可安装", tone: .blue)], sections: [DetailSection(title: "更新说明", rows: [DetailRow(title: "移动端体验", subtitle: "通知、备份和文件预览优化", symbol: "iphone", tone: .blue, trailing: "新"), DetailRow(title: "安全补丁", subtitle: "修复分享链接权限边界", symbol: "shield.lefthalf.filled", tone: .green, trailing: "建议")])], actions: ["下载更新", "稍后提醒"])
                    }
                    ModuleNavigationRow(title: "系统备份", subtitle: "备份配置、权限和应用状态", symbol: "archivebox.fill", tone: .green) {
                        ManagementDetailScreen(title: "系统备份", subtitle: "系统配置、用户权限和应用配置快照", symbol: "archivebox.fill", tone: .green, status: "已开启", metrics: [DetailMetric(title: "快照", value: "12", detail: "最近 2 小时前", tone: .green), DetailMetric(title: "保留", value: "30d", detail: "自动清理", tone: .blue)], sections: [DetailSection(title: "范围", rows: [DetailRow(title: "权限配置", subtitle: "成员、空间和分享策略", symbol: "lock.fill", tone: .green, trailing: "包含"), DetailRow(title: "应用配置", subtitle: "Docker 和应用中心配置", symbol: "square.grid.2x2.fill", tone: .blue, trailing: "包含")])], actions: ["立即备份", "恢复配置"])
                    }
                }
            }

            SectionHeader(title: "界面")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "语言", subtitle: "简体中文", symbol: "character.bubble.fill", tone: .gray) {
                        ManagementDetailScreen(title: "语言", subtitle: "移动端显示语言和单位格式", symbol: "character.bubble.fill", tone: .gray, status: "简体中文", sections: [DetailSection(title: "选项", rows: [DetailRow(title: "简体中文", subtitle: "当前使用", symbol: "checkmark.circle.fill", tone: .green, trailing: "已选"), DetailRow(title: "English", subtitle: "后续版本开放", symbol: "globe", tone: .gray, trailing: "计划")])], actions: ["保存"])
                    }
                    ModuleNavigationRow(title: "外观", subtitle: "跟随系统，浅色卡片优先", symbol: "paintpalette.fill", tone: .blue) {
                        ManagementDetailScreen(title: "外观", subtitle: "主题、卡片密度和状态颜色策略", symbol: "paintpalette.fill", tone: .blue, status: "跟随系统", sections: [DetailSection(title: "显示", rows: [DetailRow(title: "主题", subtitle: "跟随 iOS 系统设置", symbol: "circle.lefthalf.filled", tone: .blue, trailing: "自动"), DetailRow(title: "卡片密度", subtitle: "移动端信息密度优化", symbol: "rectangle.grid.1x2.fill", tone: .green, trailing: "标准")])], actions: ["应用外观"])
                    }
                    ModuleNavigationRow(title: "数据模式", subtitle: "当前为本地 Mock UI 模式", symbol: "shippingbox.fill", tone: .green) {
                        ManagementDetailScreen(title: "数据模式", subtitle: "当前页面全部使用本地 Fixture，无真实后端调用", symbol: "shippingbox.fill", tone: .green, status: "Mock", sections: [DetailSection(title: "说明", rows: [DetailRow(title: "Repository", subtitle: "页面依赖协议，后续替换真实 API", symbol: "server.rack", tone: .green, trailing: "已预留"), DetailRow(title: "网络请求", subtitle: "当前 App 不发起后端请求", symbol: "wifi.slash", tone: .gray, trailing: "关闭")])])
                    }
                }
            }
        }
        .navigationTitle("系统设置")
    }
}

struct RemoteModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(
                    title: "远程访问",
                    subtitle: model.systemStatus.remoteEnabled ? "已开启 · 本地优先保护" : "未开启",
                    symbol: "globe.asia.australia.fill",
                    tone: model.systemStatus.remoteEnabled ? .blue : .gray,
                    trailing: model.systemStatus.remoteEnabled ? "已开启" : "关闭"
                )
            }

            SectionHeader(title: "设备")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "我的 iPhone 15 Pro", subtitle: "最近访问：刚刚 · 受信任", symbol: "iphone", tone: .green) {
                        remoteDeviceDetail("我的 iPhone 15 Pro", symbol: "iphone", lastSeen: "刚刚", trusted: true)
                    }
                    ModuleNavigationRow(title: "iPad Air", subtitle: "最近访问：昨天 22:30", symbol: "ipad", tone: .blue) {
                        remoteDeviceDetail("iPad Air", symbol: "ipad", lastSeen: "昨天 22:30", trusted: true)
                    }
                    ModuleNavigationRow(title: "MacBook Pro", subtitle: "最近访问：昨天 21:15", symbol: "laptopcomputer", tone: .blue) {
                        remoteDeviceDetail("MacBook Pro", symbol: "laptopcomputer", lastSeen: "昨天 21:15", trusted: true)
                    }
                }
            }

            SectionHeader(title: "安全")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "域名令牌", subtitle: "用于绑定远程入口，可轮换", symbol: "key.fill", tone: .orange, trailing: "需确认") {
                        ManagementDetailScreen(title: "域名令牌", subtitle: "远程访问域名绑定和令牌轮换", symbol: "key.fill", tone: .orange, status: "需确认", sections: [DetailSection(title: "令牌", rows: [DetailRow(title: "当前令牌", subtitle: "已绑定 higoos-home.local", symbol: "key.fill", tone: .green, trailing: "有效"), DetailRow(title: "轮换策略", subtitle: "轮换后旧设备需要重新确认", symbol: "arrow.triangle.2.circlepath", tone: .orange, trailing: "手动")])], actions: ["轮换令牌", "查看设备"])
                    }
                    ModuleNavigationRow(title: "登录提醒", subtitle: "异常登录会进入通知与审计", symbol: "bell.and.waves.left.and.right.fill", tone: .red) {
                        ManagementDetailScreen(title: "登录提醒", subtitle: "远程登录、异常 IP 和新设备提醒", symbol: "bell.and.waves.left.and.right.fill", tone: .red, status: "高敏感", sections: [DetailSection(title: "规则", rows: [DetailRow(title: "新设备", subtitle: "首次登录必须通知管理员", symbol: "iphone.badge.exclamationmark", tone: .orange, trailing: "开启"), DetailRow(title: "异常地区", subtitle: "阻止并写入审计日志", symbol: "location.slash.fill", tone: .red, trailing: "阻止")])], actions: ["测试提醒", "编辑规则"])
                    }
                    ModuleNavigationRow(title: "共享扫描", subtitle: "检查远程可见分享链接", symbol: "magnifyingglass.circle.fill", tone: .blue) {
                        ManagementDetailScreen(title: "共享扫描", subtitle: "扫描远程入口可见的分享链接和协议", symbol: "magnifyingglass.circle.fill", tone: .blue, status: "可执行", metrics: [DetailMetric(title: "分享", value: "\(model.shares.count)", detail: "待扫描", tone: .blue), DetailMetric(title: "风险", value: "1", detail: "建议检查", tone: .orange)], sections: [DetailSection(title: "范围", rows: model.shares.map { DetailRow(title: $0.title, subtitle: "有效期至 \($0.expires)", symbol: "link.circle.fill", tone: $0.secure ? .green : .orange, trailing: $0.secure ? "安全" : "检查") })], actions: ["开始扫描", "查看报告"])
                    }
                }
            }
        }
        .navigationTitle("远程访问")
    }

    private func remoteDeviceDetail(_ name: String, symbol: String, lastSeen: String, trusted: Bool) -> ManagementDetailScreen {
        ManagementDetailScreen(title: name, subtitle: "远程访问设备、信任状态和最近活动", symbol: symbol, tone: trusted ? .green : .orange, status: trusted ? "受信任" : "需确认", metrics: [DetailMetric(title: "最近", value: lastSeen, detail: "访问时间", tone: .blue), DetailMetric(title: "状态", value: trusted ? "可信" : "待定", detail: "设备策略", tone: trusted ? .green : .orange)], sections: [DetailSection(title: "安全", rows: [DetailRow(title: "设备绑定", subtitle: "需要本机确认后才能远程访问", symbol: "checkmark.shield.fill", tone: .green, trailing: "已绑定"), DetailRow(title: "访问范围", subtitle: "文件、相册和通知，管理操作需二次确认", symbol: "lock.fill", tone: .blue, trailing: "受限")])], actions: ["撤销信任", "查看审计"])
    }
}

struct TasksModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            SectionHeader(title: "运行中")
            ForEach(model.backupJobs.prefix(2)) { job in
                HiGoCard {
                    ModuleNavigationRow(title: job.title, subtitle: job.subtitle, symbol: "checklist", tone: job.tone, trailing: job.displayStatus) {
                        ManagementDetailScreen(title: job.title, subtitle: "任务执行进度、队列位置和操作记录", symbol: "checklist", tone: job.tone, status: job.displayStatus, metrics: [DetailMetric(title: "进度", value: "\(Int(job.progress * 100))%", detail: "当前任务", tone: job.tone), DetailMetric(title: "队列", value: "前台", detail: "高优先级", tone: .blue)], sections: [DetailSection(title: "执行", rows: [DetailRow(title: "任务来源", subtitle: "移动端自动备份", symbol: "iphone", tone: .blue, trailing: "自动"), DetailRow(title: "执行节点", subtitle: model.nasDevice.name, symbol: "server.rack", tone: .green, trailing: "在线"), DetailRow(title: "审计", subtitle: "暂停、恢复和失败进入安全中心", symbol: "shield.lefthalf.filled", tone: .orange, trailing: "开启")])], actions: ["暂停", "调整优先级"])
                    }
                }
            }

            SectionHeader(title: "队列")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "影视转码", subtitle: "2 个任务等待移动端转码", symbol: "film.stack.fill", tone: .blue, trailing: "等待") {
                        ManagementDetailScreen(title: "影视转码", subtitle: "移动端可查看转码队列、格式和目标设备", symbol: "film.stack.fill", tone: .blue, status: "等待", metrics: [DetailMetric(title: "任务", value: "2", detail: "队列中", tone: .blue), DetailMetric(title: "目标", value: "1080p", detail: "移动端预览", tone: .green)], sections: [DetailSection(title: "队列", rows: [DetailRow(title: "家庭旅行.mov", subtitle: "4K -> 1080p · 预计 12 分钟", symbol: "film.fill", tone: .blue, trailing: "等待"), DetailRow(title: "生日记录.mp4", subtitle: "HEVC -> H.264 · 预计 8 分钟", symbol: "film.fill", tone: .green, trailing: "等待")])], actions: ["开始队列", "调整格式"])
                    }
                    ModuleNavigationRow(title: "文件索引", subtitle: "家庭资料语义索引增量扫描", symbol: "doc.text.magnifyingglass", tone: .green, trailing: "计划中") {
                        ManagementDetailScreen(title: "文件索引", subtitle: "为 AI 搜索和文件分类准备本地索引", symbol: "doc.text.magnifyingglass", tone: .green, status: "计划中", metrics: [DetailMetric(title: "文件", value: "256", detail: "待扫描", tone: .blue), DetailMetric(title: "模型", value: "本地", detail: "隐私优先", tone: .green)], sections: [DetailSection(title: "范围", rows: [DetailRow(title: "家庭资料", subtitle: "文档、发票和照片说明", symbol: "folder.fill", tone: .green, trailing: "包含"), DetailRow(title: "共享空间", subtitle: "仅索引用户授权目录", symbol: "lock.fill", tone: .orange, trailing: "受限")])], actions: ["开始扫描", "排除目录"])
                    }
                    ModuleNavigationRow(title: "下载归档", subtitle: "下载完成后自动移动到媒体库", symbol: "arrow.down.circle.fill", tone: .orange, trailing: "暂停") {
                        ManagementDetailScreen(title: "下载归档", subtitle: "下载中心完成后的命名、移动和标签流程", symbol: "arrow.down.circle.fill", tone: .orange, status: "暂停", metrics: [DetailMetric(title: "等待", value: "4", detail: "文件", tone: .orange), DetailMetric(title: "规则", value: "3", detail: "已启用", tone: .green)], sections: [DetailSection(title: "规则", rows: [DetailRow(title: "影视文件", subtitle: "移动到媒体库并触发刮削", symbol: "play.rectangle.fill", tone: .blue, trailing: "开启"), DetailRow(title: "文档", subtitle: "按来源归档到共享空间", symbol: "doc.fill", tone: .green, trailing: "开启")])], actions: ["恢复任务", "编辑规则"])
                    }
                }
            }

            SectionHeader(title: "历史")
            HiGoCard {
                ModuleNavigationRow(title: "查看全部记录", subtitle: "成功、失败、取消和回滚任务", symbol: "clock.fill", tone: .gray) {
                    ManagementDetailScreen(title: "任务记录", subtitle: "所有后台任务的结果、耗时和操作者", symbol: "clock.fill", tone: .gray, status: "24 条", sections: [DetailSection(title: "今天", rows: [DetailRow(title: "照片备份", subtitle: "已完成 1,256 张照片", symbol: "checkmark.circle.fill", tone: .green, trailing: "成功"), DetailRow(title: "文件索引", subtitle: "等待夜间窗口执行", symbol: "clock.fill", tone: .blue, trailing: "计划中"), DetailRow(title: "下载归档", subtitle: "用户暂停", symbol: "pause.circle.fill", tone: .orange, trailing: "暂停")])], actions: ["筛选", "导出"])
                }
            }
        }
        .navigationTitle("任务中心")
    }
}

struct ProtocolsModuleScreen: View {
    var body: some View {
        PageContainer {
            SectionHeader(title: "共享协议")
            HiGoCard {
                VStack(spacing: 12) {
                    protocolRow("SMB", subtitle: "家庭空间、团队空间已共享", symbol: "desktopcomputer", tone: .green, trailing: "启用")
                    protocolRow("NFS", subtitle: "开发设备只读挂载", symbol: "server.rack", tone: .blue, trailing: "启用")
                    protocolRow("WebDAV", subtitle: "移动端与远程访问兼容入口", symbol: "globe", tone: .blue, trailing: "启用")
                    protocolRow("DLNA", subtitle: "客厅电视媒体发现", symbol: "tv.fill", tone: .orange, trailing: "待配置")
                }
            }

            SectionHeader(title: "确认与审计")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "新增共享", subtitle: "公开范围、密码和只读策略需确认", symbol: "plus.circle.fill", tone: .blue, trailing: "需确认") {
                        ManagementDetailScreen(title: "新增共享", subtitle: "选择协议、目录、访问范围和密码策略", symbol: "plus.circle.fill", tone: .blue, status: "需确认", sections: [DetailSection(title: "表单", rows: [DetailRow(title: "目录", subtitle: "家庭空间 / 团队空间 / 指定文件夹", symbol: "folder.fill", tone: .blue, trailing: "必选"), DetailRow(title: "权限", subtitle: "只读、读写或临时授权", symbol: "lock.fill", tone: .orange, trailing: "必选"), DetailRow(title: "密码", subtitle: "远程访问默认要求密码", symbol: "key.fill", tone: .green, trailing: "开启")])], actions: ["创建共享", "保存草稿"])
                    }
                    ModuleNavigationRow(title: "协议审计", subtitle: "记录每次共享配置变更", symbol: "doc.text.magnifyingglass", tone: .green) {
                        ManagementDetailScreen(title: "协议审计", subtitle: "共享协议启停、权限变更和访问记录", symbol: "doc.text.magnifyingglass", tone: .green, status: "已开启", sections: [DetailSection(title: "最近", rows: [DetailRow(title: "SMB", subtitle: "张小明更新家庭空间权限", symbol: "desktopcomputer", tone: .green, trailing: "成功"), DetailRow(title: "WebDAV", subtitle: "系统检查远程访问入口", symbol: "globe", tone: .blue, trailing: "正常")])], actions: ["筛选", "导出"])
                    }
                }
            }
        }
        .navigationTitle("共享协议")
    }

    private func protocolRow(_ title: String, subtitle: String, symbol: String, tone: Tone, trailing: String) -> some View {
        ModuleNavigationRow(title: title, subtitle: subtitle, symbol: symbol, tone: tone, trailing: trailing) {
            ManagementDetailScreen(title: title, subtitle: subtitle, symbol: symbol, tone: tone, status: trailing, metrics: [DetailMetric(title: "连接", value: title == "DLNA" ? "0" : "3", detail: "当前会话", tone: tone), DetailMetric(title: "权限", value: title == "NFS" ? "只读" : "读写", detail: "默认策略", tone: .blue)], sections: [DetailSection(title: "配置", rows: [DetailRow(title: "共享目录", subtitle: title == "DLNA" ? "媒体库待配置" : "家庭空间、团队空间", symbol: "folder.fill", tone: tone, trailing: title == "DLNA" ? "待选" : "已选"), DetailRow(title: "访问控制", subtitle: "按成员、设备和远程状态控制", symbol: "lock.fill", tone: .green, trailing: "开启")])], actions: [trailing == "启用" ? "暂停协议" : "完成配置", "查看审计"])
        }
    }
}

struct AppCenterModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel

    var body: some View {
        PageContainer {
            ForEach(model.apps) { app in
                HiGoCard {
                    ModuleNavigationRow(title: app.name, subtitle: app.description, symbol: app.icon, tone: app.tone, trailing: app.status.rawValue) {
                        ManagementDetailScreen(title: app.name, subtitle: app.description, symbol: app.icon, tone: app.tone, status: app.status.rawValue, metrics: [DetailMetric(title: "状态", value: app.status.rawValue, detail: "当前应用", tone: app.tone), DetailMetric(title: "权限", value: "3", detail: "已授权", tone: .blue)], sections: [DetailSection(title: "权限", rows: [DetailRow(title: "存储", subtitle: "访问应用专属目录和媒体库", symbol: "internaldrive.fill", tone: .blue, trailing: "允许"), DetailRow(title: "网络", subtitle: "仅允许局域网和远程入口代理", symbol: "network", tone: .green, trailing: "受控"), DetailRow(title: "通知", subtitle: "更新、异常和任务完成提醒", symbol: "bell.badge.fill", tone: .orange, trailing: "开启")]), DetailSection(title: "活动", rows: [DetailRow(title: "上次打开", subtitle: "今天 10:24", symbol: "clock.fill", tone: .gray), DetailRow(title: "更新策略", subtitle: app.status == .updateAvailable ? "有新版本可安装" : "自动检查更新", symbol: "arrow.triangle.2.circlepath", tone: app.status == .updateAvailable ? .orange : .green, trailing: app.status == .updateAvailable ? "可更新" : "正常")])], actions: app.status == .updateAvailable ? ["更新", "权限"] : ["打开", "权限"])
                    }
                }
            }
            SectionHeader(title: "推荐")
            HiGoCard {
                VStack(spacing: 12) {
                    ModuleNavigationRow(title: "相册索引增强", subtitle: "本地人物地点识别，适合家庭相册", symbol: "sparkles.rectangle.stack.fill", tone: .blue, trailing: "推荐") {
                        ManagementDetailScreen(title: "相册索引增强", subtitle: "安装前查看权限、空间和模型策略", symbol: "sparkles.rectangle.stack.fill", tone: .blue, status: "可安装", metrics: [DetailMetric(title: "空间", value: "420MB", detail: "预计占用", tone: .blue), DetailMetric(title: "权限", value: "相册", detail: "只读索引", tone: .green)], sections: [DetailSection(title: "声明", rows: [DetailRow(title: "隐私", subtitle: "人物地点识别在 NAS 本地执行", symbol: "hand.raised.fill", tone: .green, trailing: "本地"), DetailRow(title: "回滚", subtitle: "卸载后保留原始照片", symbol: "arrow.uturn.backward.circle.fill", tone: .blue, trailing: "支持")])], actions: ["安装", "稍后"])
                    }
                    ModuleNavigationRow(title: "远程下载助手", subtitle: "远程提交下载任务并自动归档", symbol: "arrow.down.circle.fill", tone: .green, trailing: "可安装") {
                        ManagementDetailScreen(title: "远程下载助手", subtitle: "下载任务、归档规则和权限声明", symbol: "arrow.down.circle.fill", tone: .green, status: "可安装", sections: [DetailSection(title: "能力", rows: [DetailRow(title: "远程提交", subtitle: "移动端添加任务，NAS 本地执行", symbol: "paperplane.fill", tone: .blue, trailing: "支持"), DetailRow(title: "自动归档", subtitle: "完成后按类型移动到对应空间", symbol: "archivebox.fill", tone: .green, trailing: "支持")])], actions: ["安装", "查看权限"])
                    }
                }
            }
        }
        .navigationTitle("应用中心")
    }
}

struct SecurityModuleScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var showingConfirm = false

    var body: some View {
        PageContainer {
            SectionHeader(title: "风险与审计")
            ForEach(model.audit) { entry in
                HiGoCard {
                    VStack(alignment: .leading, spacing: 12) {
                        ModuleNavigationRow(title: entry.action, subtitle: "\(entry.actor) · \(entry.time)", symbol: "shield.lefthalf.filled", tone: entry.tone, trailing: entry.result.rawValue) {
                            ManagementDetailScreen(title: entry.action, subtitle: "\(entry.actor) · \(entry.time)", symbol: "shield.lefthalf.filled", tone: entry.tone, status: entry.result.rawValue, metrics: [DetailMetric(title: "风险", value: entry.result == .blocked ? "高" : entry.result == .pending ? "中" : "低", detail: "系统评估", tone: entry.tone), DetailMetric(title: "回滚", value: "支持", detail: "操作记录", tone: .blue)], sections: [DetailSection(title: "影响", rows: [DetailRow(title: "对象", subtitle: entry.action.contains("照片") ? "32 个相似照片文件" : "家庭资料和分享权限", symbol: "folder.fill", tone: .blue, trailing: "已识别"), DetailRow(title: "确认", subtitle: entry.result == .pending ? "需要管理员确认后执行" : "已完成审计闭环", symbol: "checkmark.shield.fill", tone: entry.tone, trailing: entry.result.rawValue), DetailRow(title: "记录", subtitle: "执行前后状态都会保留", symbol: "clock.arrow.circlepath", tone: .green, trailing: "开启")])], actions: entry.result == .pending ? ["确认执行", "拒绝"] : ["查看详情", "导出"])
                        }
                        if entry.tone == .orange {
                            Button("查看确认") { showingConfirm = true }
                                .buttonStyle(.borderedProminent)
                        }
                    }
                }
            }
        }
        .navigationTitle("安全中心")
        .sheet(isPresented: $showingConfirm) {
            ConfirmSheet(title: "确认 Agent 批量重命名照片", impact: "该操作会影响 32 个相似照片文件。执行前会生成审计记录，并预留回滚入口。", risk: .orange) {
                showingConfirm = false
            }
        }
    }
}
