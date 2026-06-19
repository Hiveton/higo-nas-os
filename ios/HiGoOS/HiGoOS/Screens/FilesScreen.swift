import SwiftUI

private enum FileSortMode: String, CaseIterable, Identifiable {
    case updated = "最近更新"
    case name = "名称"
    case type = "类型"
    case size = "大小"

    var id: String { rawValue }
}

struct FilesScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var scope: FileSpace = .personal
    @State private var query = ""
    @State private var sortMode: FileSortMode = .updated
    @State private var selectedFile: FileItem?
    @State private var showingUpload = false
    @State private var showingNewFolder = false
    @State private var showingSort = false
    @State private var selectionMode = false
    @State private var selectedIDs: Set<String> = []
    @State private var showingFilters = false
    @State private var createdFolders: [FileItem] = []

    private var folders: [FileItem] {
        sorted(filtered(model.filesStore.folders(in: scope) + createdFolders.filter { $0.space == scope }))
    }

    private var recentFiles: [FileItem] {
        sorted(filtered(model.filesStore.recentFiles(in: scope)))
    }

    var body: some View {
        PageContainer {
            Picker("文件范围", selection: $scope) {
                ForEach(FileSpace.allCases) { Text($0.rawValue).tag($0) }
            }
            .pickerStyle(.segmented)

            SearchInput(text: $query, placeholder: "搜索文件或文件夹", symbol: "magnifyingglass")

            HiGoCard {
                HStack {
                    Label("HiGoOS / \(scope.rawValue)空间", systemImage: "chevron.right")
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(HiGoTheme.ink)
                    Spacer()
                    StatusPill(text: selectionMode ? "已选 \(selectedIDs.count)" : "离线可用", tone: selectionMode ? .orange : .green, symbol: selectionMode ? "checkmark.circle.fill" : "icloud.and.arrow.down.fill")
                }
            }

            HStack(spacing: 10) {
                Button("上传", systemImage: "square.and.arrow.up") { showingUpload = true }
                    .buttonStyle(.borderedProminent)
                Button("新建", systemImage: "plus") { showingNewFolder = true }
                    .buttonStyle(.bordered)
                Button("筛选", systemImage: "line.3.horizontal.decrease.circle") { showingFilters = true }
                    .buttonStyle(.bordered)
                Button(sortMode.rawValue, systemImage: "arrow.up.arrow.down") { showingSort = true }
                    .buttonStyle(.bordered)
            }
            .controlSize(.regular)

            if selectionMode {
                HiGoCard {
                    HStack {
                        ModuleActionButton(title: "分享", detail: "已选择 \(selectedIDs.count) 个文件进入分享确认。", isPrimary: true)
                        ModuleActionButton(title: "移动", detail: "已选择 \(selectedIDs.count) 个文件进入移动确认。", isPrimary: false)
                        ModuleActionButton(title: "删除", detail: "已选择 \(selectedIDs.count) 个文件进入回收站确认。", isPrimary: false)
                    }
                }
            } else {
                Button("进入多选", systemImage: "checklist") { selectionMode = true }
                    .buttonStyle(.bordered)
            }

            SectionHeader(title: "文件夹", action: "\(folders.count) 项")
            HiGoCard {
                if folders.isEmpty {
                    EmptyStateView(title: "暂无文件夹", message: "当前空间没有匹配的文件夹。", symbol: "folder.badge.questionmark")
                } else {
                    VStack(spacing: 4) {
                        ForEach(folders) { item in
                            Button { handleTap(item) } label: { SelectableFileRow(item: item, selected: selectedIDs.contains(item.id), selectionMode: selectionMode) }
                                .buttonStyle(.plain)
                            if item.id != folders.last?.id { Divider().padding(.leading, 46) }
                        }
                    }
                }
            }

            SectionHeader(title: "最近文件", action: "\(recentFiles.count) 项")
            HiGoCard {
                if recentFiles.isEmpty {
                    EmptyStateView(title: "暂无文件", message: "切换空间或清空搜索条件后再查看。", symbol: "doc.text.magnifyingglass")
                } else {
                    VStack(spacing: 4) {
                        ForEach(recentFiles) { item in
                            Button { handleTap(item) } label: { SelectableFileRow(item: item, selected: selectedIDs.contains(item.id), selectionMode: selectionMode) }
                                .buttonStyle(.plain)
                            if item.id != recentFiles.last?.id { Divider().padding(.leading, 46) }
                        }
                    }
                }
            }
        }
        .navigationTitle("文件")
        .sheet(item: $selectedFile) { file in
            FileDetailSheet(item: file)
        }
        .sheet(isPresented: $showingUpload) {
            UploadFileSheet(defaultSpace: scope)
        }
        .sheet(isPresented: $showingNewFolder) {
            NewFolderSheet(defaultSpace: scope) { name, space in
                createdFolders.append(
                    FileItem(
                        id: "local-folder-\(UUID().uuidString)",
                        name: name,
                        kind: .folder,
                        size: "0 项",
                        date: "刚刚",
                        space: space,
                        tags: ["新建"],
                        icon: "folder.fill",
                        tone: space == .family ? .green : .blue
                    )
                )
            }
        }
        .confirmationDialog("排序方式", isPresented: $showingSort, titleVisibility: .visible) {
            ForEach(FileSortMode.allCases) { mode in
                Button(mode.rawValue) { sortMode = mode }
            }
        }
        .confirmationDialog("筛选文件", isPresented: $showingFilters, titleVisibility: .visible) {
            Button("只看图片") { query = "图片" }
            Button("只看文档") { query = "文档" }
            Button("只看协作") { query = "协作" }
            Button("回收站") { query = "回收站" }
            Button("清除筛选") { query = "" }
        }
        .toolbar {
            if selectionMode {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") {
                        selectionMode = false
                        selectedIDs.removeAll()
                    }
                }
            }
        }
    }

    private func handleTap(_ item: FileItem) {
        if selectionMode {
            if selectedIDs.contains(item.id) {
                selectedIDs.remove(item.id)
            } else {
                selectedIDs.insert(item.id)
            }
        } else {
            selectedFile = item
        }
    }

    private func filtered(_ items: [FileItem]) -> [FileItem] {
        let keyword = query.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !keyword.isEmpty else { return items }
        return items.filter {
            $0.name.localizedCaseInsensitiveContains(keyword)
                || $0.tags.contains { $0.localizedCaseInsensitiveContains(keyword) }
                || $0.kind.rawValue.localizedCaseInsensitiveContains(keyword)
        }
    }

    private func sorted(_ items: [FileItem]) -> [FileItem] {
        switch sortMode {
        case .updated:
            return items.sorted { $0.date > $1.date }
        case .name:
            return items.sorted { $0.name.localizedCompare($1.name) == .orderedAscending }
        case .type:
            return items.sorted { $0.kind.rawValue < $1.kind.rawValue }
        case .size:
            return items.sorted { $0.size > $1.size }
        }
    }
}

struct SelectableFileRow: View {
    var item: FileItem
    var selected: Bool
    var selectionMode: Bool

    var body: some View {
        HStack(spacing: 10) {
            if selectionMode {
                Image(systemName: selected ? "checkmark.circle.fill" : "circle")
                    .font(.title3)
                    .foregroundStyle(selected ? Color.higoBlue : HiGoTheme.muted)
                    .frame(width: 24)
            }
            FileRow(item: item)
        }
    }
}

struct SearchInput: View {
    @Binding var text: String
    var placeholder: String
    var symbol: String

    var body: some View {
        HStack(spacing: 10) {
            Image(systemName: symbol)
                .foregroundStyle(HiGoTheme.muted)
            TextField(placeholder, text: $text)
                .textInputAutocapitalization(.never)
                .disableAutocorrection(true)
            if !text.isEmpty {
                Button {
                    text = ""
                } label: {
                    Image(systemName: "xmark.circle.fill")
                        .foregroundStyle(HiGoTheme.muted)
                }
                .buttonStyle(.plain)
            }
        }
        .padding(14)
        .background(HiGoTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 16, style: .continuous)
                .stroke(Color.higoBlue.opacity(0.28), lineWidth: 1)
        }
    }
}

struct FileRow: View {
    var item: FileItem

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: item.icon)
                .font(.title3)
                .foregroundStyle(item.tone.color)
                .frame(width: 34)
            VStack(alignment: .leading, spacing: 4) {
                Text(item.name)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(HiGoTheme.ink)
                    .lineLimit(1)
                Text("\(item.date) · \(item.size)")
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.muted)
            }
            Spacer()
            if !item.tags.isEmpty {
                StatusPill(text: item.tags[0], tone: item.tone)
            }
            Image(systemName: "chevron.right")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)
        }
        .padding(.vertical, 10)
    }
}

struct FileDetailSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var showingPreview = false
    @State private var showingShare = false
    @State private var showingTags = false
    @State private var localTags: [String]
    var item: FileItem

    init(item: FileItem) {
        self.item = item
        _localTags = State(initialValue: item.tags)
    }

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    HStack(spacing: 14) {
                        Image(systemName: item.icon)
                            .font(.largeTitle)
                            .foregroundStyle(item.tone.color)
                        VStack(alignment: .leading, spacing: 5) {
                            Text(item.name)
                                .font(.title3.weight(.bold))
                            Text("\(item.kind.rawValue) · \(item.size)")
                                .foregroundStyle(HiGoTheme.muted)
                        }
                    }
                }

                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "所在空间", subtitle: item.space.rawValue, symbol: "externaldrive.fill", tone: .blue, trailing: "可访问")
                        ActionRow(title: "权限", subtitle: item.space == .team ? "团队成员可协作编辑" : "当前用户可读写", symbol: "lock.fill", tone: .green, trailing: item.space == .shared ? "共享" : "读写")
                        ActionRow(title: "离线", subtitle: "可标记为离线可用，移动端下次自动同步", symbol: "icloud.and.arrow.down.fill", tone: .blue, trailing: "可用")
                        ActionRow(title: "更新时间", subtitle: item.date, symbol: "clock.fill", tone: .gray)
                        ActionRow(title: "标签", subtitle: localTags.isEmpty ? "无标签" : localTags.joined(separator: "、"), symbol: "tag.fill", tone: .orange, trailing: "\(localTags.count)")
                    }
                }

                HiGoCard {
                    HStack {
                        Button("预览", systemImage: "eye") { showingPreview = true }
                            .buttonStyle(.borderedProminent)
                        Button("分享", systemImage: "square.and.arrow.up") { showingShare = true }
                            .buttonStyle(.bordered)
                        Button("标签", systemImage: "tag") { showingTags = true }
                            .buttonStyle(.bordered)
                    }
                }
            }
            .navigationTitle("文件详情")
            .toolbar {
                ToolbarItem(placement: .topBarTrailing) {
                    Button("完成") { dismiss() }
                }
            }
            .sheet(isPresented: $showingPreview) {
                FilePreviewSheet(item: item)
            }
            .sheet(isPresented: $showingShare) {
                ShareSettingsSheet(item: item)
            }
            .sheet(isPresented: $showingTags) {
                TagEditorSheet(tags: $localTags) {
                    model.recordMockCommand("保存标签", detail: "\(item.name) 的标签已在本地 UI 状态中更新。")
                }
            }
        }
        .presentationDetents([.large])
    }
}

struct UploadFileSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var targetSpace: FileSpace
    @State private var uploadMode = "照片和视频"
    @State private var keepOriginals = true

    init(defaultSpace: FileSpace) {
        _targetSpace = State(initialValue: defaultSpace)
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("上传类型") {
                    Picker("类型", selection: $uploadMode) {
                        Text("照片和视频").tag("照片和视频")
                        Text("文件").tag("文件")
                        Text("文件夹").tag("文件夹")
                    }
                    .pickerStyle(.segmented)
                }
                Section("目标") {
                    Picker("空间", selection: $targetSpace) {
                        ForEach(FileSpace.allCases) { Text($0.rawValue).tag($0) }
                    }
                    Toggle("保留原始文件信息", isOn: $keepOriginals)
                }
                Section("队列") {
                    Label("待上传 3 项", systemImage: "tray.and.arrow.up.fill")
                    Label("Wi-Fi 下自动开始", systemImage: "wifi")
                }
            }
            .navigationTitle("上传")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("加入队列") {
                        model.recordMockCommand("加入上传队列", detail: "\(uploadMode) 将上传到\(targetSpace.rawValue)空间。")
                        dismiss()
                    }
                }
            }
        }
    }
}

struct NewFolderSheet: View {
    @Environment(\.dismiss) private var dismiss
    @State private var name = "新建文件夹"
    @State private var space: FileSpace
    var onCreate: (String, FileSpace) -> Void

    init(defaultSpace: FileSpace, onCreate: @escaping (String, FileSpace) -> Void) {
        _space = State(initialValue: defaultSpace)
        self.onCreate = onCreate
    }

    var body: some View {
        NavigationStack {
            Form {
                Section("文件夹") {
                    TextField("名称", text: $name)
                    Picker("空间", selection: $space) {
                        ForEach(FileSpace.allCases) { Text($0.rawValue).tag($0) }
                    }
                }
                Section("权限") {
                    Label(space == .personal ? "仅自己可见" : "按空间成员权限访问", systemImage: "lock.fill")
                }
            }
            .navigationTitle("新建文件夹")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("创建") {
                        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
                        onCreate(trimmed.isEmpty ? "新建文件夹" : trimmed, space)
                        dismiss()
                    }
                }
            }
        }
    }
}

struct FilePreviewSheet: View {
    @Environment(\.dismiss) private var dismiss
    var item: FileItem

    var body: some View {
        NavigationStack {
            VStack(spacing: 20) {
                Image(systemName: item.icon)
                    .font(.system(size: 82, weight: .semibold))
                    .foregroundStyle(item.tone.color)
                    .frame(width: 150, height: 150)
                    .background(item.tone.color.opacity(0.1))
                    .clipShape(RoundedRectangle(cornerRadius: 20, style: .continuous))
                Text(item.name)
                    .font(.title3.weight(.bold))
                    .multilineTextAlignment(.center)
                Text(item.kind.isFolder ? "文件夹预览会显示目录、权限和最近变更。" : "当前为本地 Mock 预览，后续接入真实文件解析器。")
                    .font(.subheadline)
                    .foregroundStyle(HiGoTheme.muted)
                    .multilineTextAlignment(.center)
                Spacer()
            }
            .padding(24)
            .navigationTitle("预览")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}

struct ShareSettingsSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var requirePassword = true
    @State private var allowDownload = true
    @State private var expiresIn = "7 天"
    var item: FileItem

    var body: some View {
        NavigationStack {
            Form {
                Section("分享对象") {
                    Label(item.name, systemImage: item.icon)
                    Picker("有效期", selection: $expiresIn) {
                        Text("1 天").tag("1 天")
                        Text("7 天").tag("7 天")
                        Text("30 天").tag("30 天")
                    }
                }
                Section("安全") {
                    Toggle("需要访问密码", isOn: $requirePassword)
                    Toggle("允许下载", isOn: $allowDownload)
                }
                Section("审计") {
                    Label("每次访问会记录设备、时间和来源", systemImage: "doc.text.magnifyingglass")
                }
            }
            .navigationTitle("创建分享")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("生成") {
                        model.recordMockCommand("生成分享链接", detail: "\(item.name) 分享有效期 \(expiresIn)，密码 \(requirePassword ? "开启" : "关闭")。")
                        dismiss()
                    }
                }
            }
        }
    }
}

struct TagEditorSheet: View {
    @Environment(\.dismiss) private var dismiss
    @Binding var tags: [String]
    @State private var draft = ""
    var onSave: () -> Void

    var body: some View {
        NavigationStack {
            Form {
                Section("现有标签") {
                    if tags.isEmpty {
                        Text("暂无标签")
                            .foregroundStyle(HiGoTheme.muted)
                    } else {
                        ForEach(tags, id: \.self) { tag in
                            HStack {
                                Label(tag, systemImage: "tag.fill")
                                Spacer()
                                Button("删除") {
                                    tags.removeAll { $0 == tag }
                                }
                                .foregroundStyle(Color.higoRed)
                            }
                        }
                    }
                }
                Section("添加") {
                    HStack {
                        TextField("标签名称", text: $draft)
                        Button("添加") {
                            let value = draft.trimmingCharacters(in: .whitespacesAndNewlines)
                            if !value.isEmpty && !tags.contains(value) {
                                tags.append(value)
                            }
                            draft = ""
                        }
                    }
                }
            }
            .navigationTitle("标签")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("保存") {
                        onSave()
                        dismiss()
                    }
                }
            }
        }
    }
}
