import SwiftUI

struct HomeScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var showingSearch = false

    private var quickModules: [FeatureModule] {
        model.homeStore.quickModules
    }

    var body: some View {
        PageContainer {
            HStack {
                Text("HiGoOS")
                    .font(.largeTitle.weight(.bold))
                Spacer()
                NavigationLink(value: ModuleID.profile) {
                    Image(systemName: "person.crop.circle")
                        .font(.title3.weight(.semibold))
                }
                NavigationLink {
                    MobileTasksScreen()
                } label: {
                    Image(systemName: "checklist")
                        .overlay(alignment: .topTrailing) {
                            if model.homeStore.riskCount > 0 {
                                Circle().fill(Color.higoRed).frame(width: 9, height: 9)
                            }
                        }
                }
            }

            NavigationLink {
                DeviceScreen()
            } label: {
                NasSummaryCard(status: model.systemStatus, device: model.nasDevice)
            }
            .buttonStyle(.plain)

            Button { showingSearch = true } label: {
                SearchBarPlaceholder(text: "搜文件、照片、任务、应用和设置")
            }
            .buttonStyle(.plain)

            HiGoCard {
                HStack(spacing: 12) {
                    MetricInline(title: "CPU", value: "\(model.deviceHealth.cpu)%", tone: .green)
                    MetricInline(title: "内存", value: "\(model.deviceHealth.memory)%", tone: .blue)
                    MetricInline(title: "硬盘", value: model.deviceHealth.diskTemperature, tone: .orange)
                    MetricInline(title: "风险", value: "\(model.homeStore.riskCount)", tone: model.homeStore.riskCount > 0 ? .red : .green)
                }
            }

            SectionHeader(title: "常用入口")
            LazyVGrid(columns: Array(repeating: GridItem(.flexible(), spacing: 12), count: 4), spacing: 12) {
                NavigationLink { FilesScreen() } label: { FeatureTile(title: "文件", symbol: "folder.fill", tone: .blue) }
                NavigationLink { PhotosScreen() } label: { FeatureTile(title: "相册", symbol: "photo.on.rectangle.angled", tone: .orange) }
                NavigationLink { BackupScreen() } label: { FeatureTile(title: "备份", symbol: "icloud.and.arrow.up.fill", tone: .green) }
                NavigationLink { ModuleScreen(moduleID: .videoCenter) } label: { FeatureTile(title: "影视", symbol: "play.rectangle.fill", tone: .orange) }
            }
            .buttonStyle(.plain)

            SectionHeader(title: "今日任务", action: "查看全部")
            HiGoCard {
                VStack(spacing: 16) {
                    ForEach(model.homeStore.todayTasks) { TaskItemRow(task: $0) }
                    NavigationLink {
                        AllModulesScreen()
                    } label: {
                        ActionRow(title: "全部功能", subtitle: "数据、媒体、设备、安全、高级服务", symbol: "square.grid.2x2.fill", tone: .blue)
                    }
                    .buttonStyle(.plain)
                    .accessibilityIdentifier("home-all-modules")
                }
            }

            SectionHeader(title: "影音与下载")
            ForEach(model.mediaLibraries.prefix(2)) { library in
                NavigationLink {
                    ModuleScreen(moduleID: library.id == "media-video" ? .videoCenter : .musicCenter)
                } label: {
                    HiGoCard {
                        ActionRow(title: library.title, subtitle: library.subtitle, symbol: library.symbol, tone: library.tone, trailing: "\(library.count)")
                    }
                }
                .buttonStyle(.plain)
            }
            ForEach(model.downloads.prefix(1)) { item in
                DownloadProgressCard(item: item)
            }

            SectionHeader(title: "家庭动态")
            HiGoCard {
                VStack(spacing: 12) {
                    ActionRow(title: "妈妈上传了旅行照片", subtitle: "家庭相册新增 38 项", symbol: "person.2.fill", tone: .green, trailing: "刚刚")
                    ActionRow(title: "访客链接即将过期", subtitle: "产品资料包 24 小时后失效", symbol: "link.circle.fill", tone: .orange, trailing: "提醒")
                    ActionRow(title: "客厅电视播放记录", subtitle: "影视中心继续观看《家庭旅行》", symbol: "tv.fill", tone: .blue, trailing: "继续")
                }
            }
        }
        .navigationDestination(for: ModuleID.self) { ModuleScreen(moduleID: $0) }
        .sheet(isPresented: $showingSearch) {
            GlobalSearchSheet()
        }
    }
}

struct NasSummaryCard: View {
    var status: SystemStatus
    var device: NasDevice

    private var capacityProgress: Double {
        guard status.capacityTotalTB > 0 else { return 0 }
        return status.capacityUsedTB / status.capacityTotalTB
    }

    var body: some View {
        HiGoCard {
            HStack(spacing: 18) {
                Image(systemName: device.imageSystemName)
                    .font(.system(size: 54, weight: .semibold))
                    .foregroundStyle(HiGoTheme.ink)
                    .frame(width: 112, height: 112)
                    .background(.linearGradient(colors: [Color.gray.opacity(0.18), Color.gray.opacity(0.05)], startPoint: .top, endPoint: .bottom))
                    .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
                VStack(alignment: .leading, spacing: 10) {
                    Text(status.name)
                        .font(.title3.weight(.bold))
                    StatusPill(text: status.online ? "在线" : "离线", tone: status.online ? .green : .red)
                    HStack {
                        Text("\(status.capacityUsedTB, specifier: "%.1f") TB / \(status.capacityTotalTB, specifier: "%.0f") TB")
                            .font(.subheadline)
                        Spacer()
                        Text("\(Int(capacityProgress * 100))%")
                            .font(.caption)
                            .foregroundStyle(HiGoTheme.muted)
                    }
                    ProgressView(value: capacityProgress)
                        .tint(Color.higoGreen)
                    HStack {
                        Text("健康评分")
                            .font(.subheadline)
                        Spacer()
                        Text("\(status.healthScore)")
                            .font(.title3.weight(.bold))
                            .foregroundStyle(Color.higoGreen)
                        Text(status.healthLabel)
                            .font(.subheadline.weight(.semibold))
                            .foregroundStyle(Color.higoGreen)
                    }
                }
            }
        }
    }
}

struct GlobalSearchSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var query = ""

    private var fileResults: [FileItem] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !keyword.isEmpty else { return Array(model.files.prefix(3)) }
        return model.files.filter { $0.name.localizedCaseInsensitiveContains(keyword) || $0.tags.contains { $0.localizedCaseInsensitiveContains(keyword) } }
    }

    private var photoResults: [PhotoAsset] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !keyword.isEmpty else { return Array(model.photos.prefix(2)) }
        return model.photos.filter { $0.title.localizedCaseInsensitiveContains(keyword) || $0.date.localizedCaseInsensitiveContains(keyword) }
    }

    private var aiResults: [AiSearchResult] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !keyword.isEmpty else { return model.aiResults }
        return model.aiResults.filter { $0.title.localizedCaseInsensitiveContains(keyword) || $0.summary.localizedCaseInsensitiveContains(keyword) }
    }

    private var taskResults: [TaskItem] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !keyword.isEmpty else { return Array(model.tasks.prefix(3)) }
        return model.tasks.filter { $0.title.localizedCaseInsensitiveContains(keyword) || $0.subtitle.localizedCaseInsensitiveContains(keyword) || $0.kind.rawValue.localizedCaseInsensitiveContains(keyword) }
    }

    private var featureResults: [FeatureEntry] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        let entries = model.featureCategories.flatMap(\.entries)
        guard !keyword.isEmpty else { return Array(entries.prefix(4)) }
        return entries.filter { $0.title.localizedCaseInsensitiveContains(keyword) || $0.subtitle.localizedCaseInsensitiveContains(keyword) }
    }

    var body: some View {
        NavigationStack {
            PageContainer {
                SearchInput(text: $query, placeholder: "搜索文件、相册、任务、应用和设置", symbol: "magnifyingglass")

                SectionHeader(title: "文件", action: "\(fileResults.count)")
                HiGoCard {
                    if fileResults.isEmpty {
                        EmptyStateView(title: "无文件结果", message: "换个关键词试试。", symbol: "doc.text.magnifyingglass")
                    } else {
                        VStack(spacing: 4) {
                            ForEach(fileResults) { FileRow(item: $0) }
                        }
                    }
                }

                SectionHeader(title: "相册", action: "\(photoResults.count)")
                ForEach(photoResults) { asset in
                    PhotoTimelineRow(asset: asset)
                }

                SectionHeader(title: "AI 结果", action: "\(aiResults.count)")
                ForEach(aiResults) { result in
                    AiResultCard(result: result)
                }

                SectionHeader(title: "任务", action: "\(taskResults.count)")
                HiGoCard {
                    VStack(spacing: 4) {
                        ForEach(taskResults) { task in
                            TaskItemRow(task: task)
                        }
                    }
                }

                SectionHeader(title: "功能与设置", action: "\(featureResults.count)")
                HiGoCard {
                    VStack(spacing: 12) {
                        ForEach(featureResults) { entry in
                            NavigationLink {
                                ModuleScreen(moduleID: entry.id)
                            } label: {
                                ActionRow(title: entry.title, subtitle: entry.subtitle, symbol: entry.icon, tone: entry.tone, trailing: entry.badge)
                            }
                            .buttonStyle(.plain)
                        }
                    }
                }
            }
            .navigationTitle("全局搜索")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}

struct SearchBarPlaceholder: View {
    var text: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: "magnifyingglass")
                .foregroundStyle(HiGoTheme.muted)
            Text(text)
                .foregroundStyle(HiGoTheme.muted)
                .lineLimit(1)
            Spacer()
            Image(systemName: "sparkles")
                .foregroundStyle(Color.higoBlue)
        }
        .padding(14)
        .background(HiGoTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(Color.higoBlue.opacity(0.35), lineWidth: 1)
        }
    }
}
