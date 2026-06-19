import SwiftUI

struct AiScreen: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var mode = "文件搜索"
    @State private var prompt = ""
    @State private var localMessages: [AiMessage] = []
    @State private var selectedResult: AiSearchResult?
    @State private var selectedConfirmation: ConfirmationRequest?
    @State private var showingPrivacy = false
    private let modes = ["全局问答", "文件管家", "相册整理", "系统诊断", "Docker/存储"]

    private var messages: [AiMessage] {
        model.aiMessages + localMessages
    }

    var body: some View {
        PageContainer {
            Picker("助手模式", selection: $mode) {
                ForEach(modes, id: \.self) { Text($0) }
            }
            .pickerStyle(.segmented)

            HiGoCard {
                VStack(alignment: .leading, spacing: 12) {
                    HStack {
                        StatusPill(text: "本地优先", tone: .green, symbol: "shield.checkered")
                        Spacer()
                        Button("策略") { showingPrivacy = true }
                            .font(.caption.weight(.semibold))
                    }
                    Text(modeDescription)
                        .font(.subheadline)
                        .foregroundStyle(HiGoTheme.muted)
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

            VStack(spacing: 10) {
                ForEach(messages) { message in
                    ChatBubble(message: message)
                }
            }

            SectionHeader(title: "建议")
            SuggestionGrid(mode: mode) { suggestion in
                submit(suggestion)
            }

            SectionHeader(title: "搜索结果")
            ForEach(model.aiResults) { result in
                Button { selectedResult = result } label: {
                    AiResultCard(result: result)
                }
                .buttonStyle(.plain)
            }

            HiGoCard {
                HStack(spacing: 10) {
                    TextField("继续追问", text: $prompt)
                        .textInputAutocapitalization(.never)
                        .disableAutocorrection(true)
                    Button {
                        submit(prompt)
                    } label: {
                        Image(systemName: "paperplane.fill")
                            .font(.headline)
                    }
                    .disabled(prompt.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty)
                }
            }
        }
        .navigationTitle("助手")
        .sheet(item: $selectedResult) { result in
            AiResultDetailSheet(result: result)
        }
        .sheet(item: $selectedConfirmation) { request in
            ConfirmationRequestSheet(request: request)
        }
        .sheet(isPresented: $showingPrivacy) {
            AiPrivacySheet()
        }
    }

    private var modeDescription: String {
        switch mode {
        case "文件管家":
            return "在文件名、标签和本地语义索引中搜索，结果可预览、分享或继续追问。"
        case "相册整理":
            return "识别重复照片、人物地点和相册结构，任何批量修改都会进入确认队列。"
        case "系统诊断":
            return "可读取设备健康、任务、通知和审计记录，输出诊断建议。"
        case "Docker/存储":
            return "查询容器、存储池、硬盘健康和日志，高风险操作只生成确认请求。"
        default:
            return "聚合文件、相册、任务、应用和系统设置，本地优先处理敏感数据。"
        }
    }

    private func submit(_ text: String) {
        let value = text.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !value.isEmpty else { return }
        localMessages.append(AiMessage(id: "local-user-\(UUID().uuidString)", text: value, time: "刚刚", fromUser: true))
        localMessages.append(AiMessage(id: "local-ai-\(UUID().uuidString)", text: "已基于\(mode)模式生成本地 Mock 回答，真实推理后续接入 Repository。", time: "刚刚", fromUser: false))
        prompt = ""
    }
}

struct AiResultCard: View {
    var result: AiSearchResult

    var body: some View {
        HiGoCard {
            HStack(alignment: .top, spacing: 12) {
                Image(systemName: result.icon)
                    .font(.title2)
                    .foregroundStyle(result.tone.color)
                    .frame(width: 44, height: 44)
                    .background(result.tone.color.opacity(0.12))
                    .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                VStack(alignment: .leading, spacing: 7) {
                    Text(result.title)
                        .font(.headline)
                        .foregroundStyle(HiGoTheme.ink)
                    Text(result.meta)
                        .font(.caption)
                        .foregroundStyle(HiGoTheme.muted)
                    Text(result.summary)
                        .font(.subheadline)
                        .foregroundStyle(HiGoTheme.ink)
                    HStack {
                        ForEach(result.tags, id: \.self) { StatusPill(text: $0, tone: result.tone) }
                    }
                }
                Spacer()
                Image(systemName: "chevron.right")
                    .font(.caption.weight(.bold))
                    .foregroundStyle(.secondary)
            }
        }
    }
}

struct SuggestionGrid: View {
    var mode: String
    var onSelect: (String) -> Void

    private var suggestions: [String] {
        switch mode {
        case "文件管家":
            return ["找 PDF", "最近图片", "共享空间文档"]
        case "相册整理":
            return ["重复照片", "家庭人物", "旅行回忆"]
        case "系统诊断":
            return ["今天失败任务", "备份进度", "设备风险"]
        case "Docker/存储":
            return ["容器状态", "硬盘温度", "存储快照"]
        default:
            return ["找保修单", "家庭资料", "最近任务"]
        }
    }

    var body: some View {
        LazyVGrid(columns: [GridItem(.flexible()), GridItem(.flexible())], spacing: 10) {
            ForEach(suggestions, id: \.self) { suggestion in
                Button(suggestion) { onSelect(suggestion) }
                    .buttonStyle(.bordered)
                    .frame(maxWidth: .infinity)
            }
        }
    }
}

struct ChatBubble: View {
    var message: AiMessage

    var body: some View {
        HStack {
            if message.fromUser { Spacer(minLength: 36) }
            VStack(alignment: .leading, spacing: 6) {
                Text(message.text)
                    .font(.subheadline)
                Text(message.time)
                    .font(.caption)
                    .foregroundStyle(message.fromUser ? Color.white.opacity(0.75) : HiGoTheme.muted)
            }
            .padding(14)
            .background(message.fromUser ? HiGoTheme.blue : Color.white)
            .foregroundStyle(message.fromUser ? .white : HiGoTheme.ink)
            .clipShape(RoundedRectangle(cornerRadius: 18, style: .continuous))
            if !message.fromUser { Spacer(minLength: 36) }
        }
    }
}

struct AiResultDetailSheet: View {
    @Environment(\.dismiss) private var dismiss
    @EnvironmentObject private var model: AppViewModel
    @State private var showingShare = false
    var result: AiSearchResult

    var body: some View {
        NavigationStack {
            PageContainer {
                AiResultCard(result: result)
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "来源", subtitle: result.meta, symbol: result.icon, tone: result.tone, trailing: "本地")
                        ActionRow(title: "摘要", subtitle: result.summary, symbol: "text.quote", tone: .blue)
                        ActionRow(title: "权限", subtitle: "预览和分享均需要当前用户权限", symbol: "lock.fill", tone: .green, trailing: "通过")
                    }
                }
                HiGoCard {
                    HStack {
                        Button("预览", systemImage: "eye") {
                            model.recordMockCommand("AI 结果预览", detail: "\(result.title) 已进入本地预览流程。")
                        }
                        .buttonStyle(.borderedProminent)
                        Button("分享", systemImage: "square.and.arrow.up") { showingShare = true }
                            .buttonStyle(.bordered)
                        Button("追问", systemImage: "bubble.left.and.bubble.right") {
                            model.recordMockCommand("继续追问", detail: "已把 \(result.title) 作为追问上下文。")
                        }
                        .buttonStyle(.bordered)
                    }
                }
            }
            .navigationTitle("搜索结果")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
            .sheet(isPresented: $showingShare) {
                AiShareSheet(result: result)
            }
        }
    }
}

struct AiShareSheet: View {
    @EnvironmentObject private var model: AppViewModel
    @Environment(\.dismiss) private var dismiss
    @State private var includeSummary = true
    @State private var expireIn = "7 天"
    var result: AiSearchResult

    var body: some View {
        NavigationStack {
            Form {
                Section("内容") {
                    Label(result.title, systemImage: result.icon)
                    Toggle("包含 AI 摘要", isOn: $includeSummary)
                }
                Section("安全") {
                    Picker("有效期", selection: $expireIn) {
                        Text("1 天").tag("1 天")
                        Text("7 天").tag("7 天")
                        Text("30 天").tag("30 天")
                    }
                }
            }
            .navigationTitle("分享结果")
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("取消") { dismiss() }
                }
                ToolbarItem(placement: .confirmationAction) {
                    Button("生成") {
                        model.recordMockCommand("AI 分享", detail: "\(result.title) 的分享有效期为 \(expireIn)。")
                        dismiss()
                    }
                }
            }
        }
    }
}

struct AiPrivacySheet: View {
    @Environment(\.dismiss) private var dismiss

    var body: some View {
        NavigationStack {
            PageContainer {
                HiGoCard {
                    ActionRow(title: "本地优先", subtitle: "文件名、标签和语义索引默认在 NAS 本地处理", symbol: "shield.checkered", tone: .green, trailing: "默认")
                }
                HiGoCard {
                    VStack(spacing: 12) {
                        ActionRow(title: "云端增强", subtitle: "只有用户确认后才会发送必要片段", symbol: "icloud.fill", tone: .orange, trailing: "确认")
                        ActionRow(title: "审计", subtitle: "每次跨边界处理都会记录在安全中心", symbol: "doc.text.magnifyingglass", tone: .blue, trailing: "开启")
                        ActionRow(title: "敏感资料", subtitle: "家庭资料和分享链接默认不出本机", symbol: "lock.fill", tone: .green, trailing: "保护")
                    }
                }
            }
            .navigationTitle("AI 策略")
            .toolbar {
                ToolbarItem(placement: .confirmationAction) {
                    Button("完成") { dismiss() }
                }
            }
        }
    }
}
