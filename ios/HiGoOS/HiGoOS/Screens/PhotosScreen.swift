import SwiftUI

struct PhotosScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var backupEnabled = true
    @State private var selectedAsset: PhotoAsset?

    var body: some View {
        PageContainer {
            NavigationLink {
                PhotoBackupSettingsScreen(backupEnabled: $backupEnabled)
            } label: {
                HiGoCard {
                    ActionRow(
                        title: backupEnabled ? "自动备份已开启" : "自动备份已暂停",
                        subtitle: "已备份 3,456 张照片，1,230 个视频",
                        symbol: "icloud.and.arrow.up.fill",
                        tone: backupEnabled ? .blue : .orange,
                        trailing: "设置"
                    )
                }
            }
            .buttonStyle(.plain)

            HStack {
                SectionHeader(title: "时间线")
                Spacer()
                NavigationLink("查看全部") {
                    PhotoCollectionScreen(title: "时间线", assets: Array(model.photos.prefix(2)))
                }
                .font(.caption.weight(.semibold))
            }
            VStack(spacing: 12) {
                ForEach(model.photos.prefix(2)) { asset in
                    Button { selectedAsset = asset } label: {
                        PhotoTimelineRow(asset: asset)
                    }
                    .buttonStyle(.plain)
                }
            }

            HStack {
                SectionHeader(title: "智能相册")
                Spacer()
                NavigationLink("查看全部") {
                    PhotoCollectionScreen(title: "智能相册", assets: Array(model.photos.dropFirst(2)))
                }
                .font(.caption.weight(.semibold))
            }
            HiGoCard {
                VStack(spacing: 8) {
                    ForEach(model.photos.dropFirst(2)) { asset in
                        Button { selectedAsset = asset } label: {
                            ActionRow(title: asset.title, subtitle: "\(asset.count) 项 · \(asset.date)", symbol: asset.symbol, tone: asset.tone)
                        }
                        .buttonStyle(.plain)
                        if asset.id != model.photos.last?.id { Divider() }
                    }
                }
            }

            SectionHeader(title: "浏览入口")
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "人物", symbol: "person.2.fill", tone: .green)
                } label: {
                    FeatureTile(title: "人物", symbol: "person.2.fill", tone: .green)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "地点", symbol: "map.fill", tone: .blue)
                } label: {
                    FeatureTile(title: "地点", symbol: "map.fill", tone: .blue)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "视频", symbol: "play.rectangle.fill", tone: .orange)
                } label: {
                    FeatureTile(title: "视频", symbol: "play.rectangle.fill", tone: .orange)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "重复项", symbol: "rectangle.stack.badge.minus", tone: .red)
                } label: {
                    FeatureTile(title: "重复项", symbol: "rectangle.stack.badge.minus", tone: .red)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "RAW 原片", symbol: "camera.aperture", tone: .gray)
                } label: {
                    FeatureTile(title: "RAW", symbol: "camera.aperture", tone: .gray)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "实况照片", symbol: "livephoto", tone: .blue)
                } label: {
                    FeatureTile(title: "实况", symbol: "livephoto", tone: .blue)
                }
                NavigationLink {
                    PhotoSmartCategoryScreen(title: "共享相册", symbol: "person.2.crop.square.stack.fill", tone: .green)
                } label: {
                    FeatureTile(title: "共享", symbol: "person.2.crop.square.stack.fill", tone: .green)
                }
                NavigationLink {
                    PhotoBackupRepairScreen()
                } label: {
                    FeatureTile(title: "修复", symbol: "wrench.and.screwdriver.fill", tone: .orange)
                }
            }
            .buttonStyle(.plain)
        }
        .navigationTitle("相册")
        .sheet(item: $selectedAsset) { asset in
            PhotoAssetDetailSheet(asset: asset)
        }
    }
}

struct PhotoBackupRepairScreen: View {
    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(title: "备份失败修复", subtitle: "检查权限、后台刷新、网络和目标文件夹", symbol: "wrench.and.screwdriver.fill", tone: .orange, trailing: "3 项")
            }
            HiGoCard {
                VStack(spacing: 12) {
                    ActionRow(title: "照片权限", subtitle: "允许访问所有照片，避免后台漏传", symbol: "photo.badge.checkmark.fill", tone: .green, trailing: "通过")
                    ActionRow(title: "后台运行", subtitle: "建议开启后台 App 刷新，保证自动续传", symbol: "clock.arrow.circlepath", tone: .orange, trailing: "建议")
                    ActionRow(title: "目标文件夹", subtitle: "/个人文件夹/Photos/iPhone 15 Pro", symbol: "folder.fill", tone: .blue, trailing: "可写")
                    ActionRow(title: "蜂窝网络", subtitle: "默认仅 Wi-Fi，临时任务可授权使用流量", symbol: "antenna.radiowaves.left.and.right", tone: .gray, trailing: "关闭")
                }
            }
            HiGoCard {
                HStack {
                    ModuleActionButton(title: "重新扫描", detail: "照片备份将重新扫描缺失项。", isPrimary: true)
                    ModuleActionButton(title: "修复设置", detail: "打开备份策略确认流程。", isPrimary: false)
                }
            }
        }
        .navigationTitle("备份修复")
    }
}

struct PhotoTimelineRow: View {
    var asset: PhotoAsset

    var body: some View {
        HiGoCard {
            HStack {
                VStack(alignment: .leading, spacing: 5) {
                    Text(asset.title)
                        .font(.headline)
                        .foregroundStyle(HiGoTheme.ink)
                    Text(asset.date)
                        .font(.caption)
                        .foregroundStyle(HiGoTheme.muted)
                    StatusPill(text: "\(asset.count) 项", tone: asset.tone)
                }
                Spacer()
                HStack(spacing: -8) {
                    ForEach(0..<5, id: \.self) { index in
                        RoundedRectangle(cornerRadius: 8, style: .continuous)
                            .fill(asset.tone.color.opacity(0.18 + Double(index) * 0.07))
                            .frame(width: 44, height: 44)
                            .overlay(Image(systemName: asset.symbol).font(.caption).foregroundStyle(asset.tone.color))
                    }
                }
            }
        }
    }
}

struct PhotoBackupSettingsScreen: View {
    @Binding var backupEnabled: Bool
    @State private var onlyWiFi = true
    @State private var backupVideos = true
    @State private var keepOriginals = true

    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(
                    title: backupEnabled ? "备份运行中" : "备份已暂停",
                    subtitle: "下一次增量扫描：今晚 22:00",
                    symbol: "icloud.and.arrow.up.fill",
                    tone: backupEnabled ? .green : .orange,
                    trailing: backupEnabled ? "开启" : "暂停"
                )
            }
            HiGoCard {
                VStack(spacing: 14) {
                    Toggle("自动备份", isOn: $backupEnabled)
                    Toggle("仅 Wi-Fi", isOn: $onlyWiFi)
                    Toggle("备份视频", isOn: $backupVideos)
                    Toggle("保留原图", isOn: $keepOriginals)
                }
            }
            SectionHeader(title: "最近记录")
            HiGoCard {
                VStack(spacing: 12) {
                    ActionRow(title: "今天 09:20", subtitle: "完成 1,256 张照片", symbol: "checkmark.circle.fill", tone: .green, trailing: "成功")
                    ActionRow(title: "昨天 22:30", subtitle: "增量备份 312 项", symbol: "clock.fill", tone: .blue, trailing: "完成")
                    ActionRow(title: "前天 22:30", subtitle: "网络中断后自动续传", symbol: "wifi.exclamationmark", tone: .orange, trailing: "已恢复")
                }
            }
        }
        .navigationTitle("自动备份")
    }
}

struct PhotoCollectionScreen: View {
    var title: String
    var assets: [PhotoAsset]
    @State private var selectedAsset: PhotoAsset?

    var body: some View {
        PageContainer {
            ForEach(assets) { asset in
                Button { selectedAsset = asset } label: {
                    PhotoTimelineRow(asset: asset)
                }
                .buttonStyle(.plain)
            }
        }
        .navigationTitle(title)
        .sheet(item: $selectedAsset) { asset in
            PhotoAssetDetailSheet(asset: asset)
        }
    }
}

struct PhotoSmartCategoryScreen: View {
    var title: String
    var symbol: String
    var tone: Tone

    var body: some View {
        PageContainer {
            HiGoCard {
                ActionRow(title: title, subtitle: "本地模型生成，数据不离开 NAS", symbol: symbol, tone: tone, trailing: "本地")
            }
            LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 12) {
                ForEach(0..<6, id: \.self) { index in
                    VStack(spacing: 10) {
                        RoundedRectangle(cornerRadius: 10, style: .continuous)
                            .fill(tone.color.opacity(0.12 + Double(index % 3) * 0.08))
                            .aspectRatio(1, contentMode: .fit)
                            .overlay {
                                Image(systemName: symbol)
                                    .font(.title2)
                                    .foregroundStyle(tone.color)
                            }
                        Text("\(title) \(index + 1)")
                            .font(.caption.weight(.semibold))
                            .lineLimit(1)
                    }
                    .padding(10)
                    .background(HiGoTheme.card)
                    .clipShape(RoundedRectangle(cornerRadius: HiGoTheme.compactCorner, style: .continuous))
                }
            }
        }
        .navigationTitle(title)
    }
}

struct PhotoAssetDetailSheet: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var model: AppViewModel
    var asset: PhotoAsset

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    VStack(alignment: .leading, spacing: 14) {
                        RoundedRectangle(cornerRadius: 16, style: .continuous)
                            .fill(asset.tone.color.opacity(0.16))
                            .frame(height: 190)
                            .overlay {
                                Image(systemName: asset.symbol)
                                    .font(.system(size: 64, weight: .semibold))
                                    .foregroundStyle(asset.tone.color)
                            }
                        Text(asset.title)
                            .font(.title3.weight(.bold))
                        Text("\(asset.date) · \(asset.count) 项")
                            .foregroundStyle(HiGoTheme.muted)
                    }
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "备份状态", subtitle: "原图和缩略图均已同步", symbol: "checkmark.icloud.fill", tone: .green, trailing: "完成")
                        ActionRow(title: "AI 索引", subtitle: "人物、地点和内容描述已在本地生成", symbol: "sparkles", tone: .blue, trailing: "本地")
                        ActionRow(title: "共享", subtitle: "可创建临时分享链接", symbol: "link.circle.fill", tone: .orange, trailing: "需确认")
                    }
                }
                HiGoCard {
                    HStack {
                        Button("播放回忆", systemImage: "play.fill") {
                            model.recordMockCommand("播放回忆", detail: "\(asset.title) 已进入本地回忆播放流程。")
                        }
                        .buttonStyle(.borderedProminent)
                        Button("分享", systemImage: "square.and.arrow.up") {
                            model.recordMockCommand("分享相册", detail: "\(asset.title) 将创建带密码的临时分享。")
                        }
                        .buttonStyle(.bordered)
                    }
                }
            }
            .navigationTitle("相册详情")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}
