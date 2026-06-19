import SwiftUI

struct HiGoCard<Content: View>: View {
    var padding: CGFloat = 16
    @ViewBuilder var content: Content

    var body: some View {
        content
            .padding(padding)
            .frame(maxWidth: .infinity, alignment: .leading)
            .background(HiGoTheme.card)
            .clipShape(RoundedRectangle(cornerRadius: HiGoTheme.corner, style: .continuous))
            .shadow(color: HiGoTheme.shadow, radius: 18, x: 0, y: 8)
    }
}

struct StatusPill: View {
    var text: String
    var tone: Tone = .blue
    var symbol: String? = nil

    var body: some View {
        HStack(spacing: 5) {
            if let symbol {
                Image(systemName: symbol)
                    .font(.caption2.weight(.bold))
            }
            Text(text)
                .font(.caption.weight(.semibold))
        }
        .foregroundStyle(tone.color)
        .padding(.horizontal, 10)
        .padding(.vertical, 6)
        .background(tone.color.opacity(0.11))
        .clipShape(Capsule())
    }
}

struct MetricCard: View {
    var title: String
    var value: String
    var detail: String
    var symbol: String
    var tone: Tone

    var body: some View {
        HiGoCard(padding: 14) {
            VStack(alignment: .leading, spacing: 12) {
                HStack {
                    Image(systemName: symbol)
                        .font(.title2)
                        .foregroundStyle(tone.color)
                    Spacer()
                    Text(title)
                        .font(.subheadline.weight(.semibold))
                        .foregroundStyle(HiGoTheme.ink)
                }
                Text(value)
                    .font(.title3.weight(.bold))
                    .foregroundStyle(HiGoTheme.ink)
                Text(detail)
                    .font(.caption)
                    .foregroundStyle(tone.color)
            }
        }
    }
}

struct FeatureTile: View {
    var title: String
    var symbol: String
    var tone: Tone

    var body: some View {
        VStack(spacing: 10) {
            Image(systemName: symbol)
                .font(.title2.weight(.semibold))
                .foregroundStyle(tone.color)
                .frame(width: 44, height: 44)
                .background(tone.color.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
            Text(title)
                .font(.subheadline.weight(.medium))
                .foregroundStyle(HiGoTheme.ink)
                .lineLimit(1)
                .minimumScaleFactor(0.8)
        }
        .frame(maxWidth: .infinity)
        .padding(.vertical, 14)
        .background(HiGoTheme.card)
        .clipShape(RoundedRectangle(cornerRadius: HiGoTheme.compactCorner, style: .continuous))
        .shadow(color: HiGoTheme.shadow, radius: 12, x: 0, y: 5)
    }
}

struct ActionRow: View {
    var title: String
    var subtitle: String
    var symbol: String
    var tone: Tone = .blue
    var trailing: String? = nil

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: symbol)
                .font(.headline)
                .foregroundStyle(tone.color)
                .frame(width: 34, height: 34)
                .background(tone.color.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            VStack(alignment: .leading, spacing: 3) {
                Text(title)
                    .font(.subheadline.weight(.semibold))
                    .foregroundStyle(HiGoTheme.ink)
                Text(subtitle)
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.muted)
                    .lineLimit(2)
            }
            Spacer(minLength: 8)
            if let trailing {
                Text(trailing)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(tone.color)
            }
            Image(systemName: "chevron.right")
                .font(.caption.weight(.bold))
                .foregroundStyle(.secondary)
        }
        .contentShape(Rectangle())
    }
}

struct TaskRow: View {
    var job: BackupJob

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "photo.on.rectangle.angled")
                .font(.headline)
                .foregroundStyle(job.tone.color)
                .frame(width: 36, height: 36)
                .background(job.tone.color.opacity(0.12))
                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
            VStack(alignment: .leading, spacing: 7) {
                HStack {
                    Text(job.title)
                        .font(.subheadline.weight(.semibold))
                    Spacer()
                    Text(job.displayStatus)
                        .font(.caption.weight(.semibold))
                        .foregroundStyle(job.tone.color)
                }
                Text(job.subtitle)
                    .font(.caption)
                    .foregroundStyle(HiGoTheme.muted)
                ProgressView(value: job.progress)
                    .tint(job.tone.color)
            }
        }
    }
}

struct NotificationRow: View {
    @EnvironmentObject private var model: AppViewModel
    var item: NotificationItem

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 12) {
                Circle()
                    .fill(item.tone.color.opacity(0.15))
                    .frame(width: 34, height: 34)
                    .overlay {
                        Image(systemName: item.tone == .red ? "exclamationmark" : "bell.fill")
                            .font(.caption.weight(.bold))
                            .foregroundStyle(item.tone.color)
                    }
                VStack(alignment: .leading, spacing: 3) {
                    HStack {
                        Text(item.title)
                            .font(.subheadline.weight(.semibold))
                        if item.unread {
                                Circle().fill(Color.higoBlue).frame(width: 7, height: 7)
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
            }
            if item.primaryAction != nil || item.secondaryAction != nil {
                HStack {
                    Spacer()
                    if let primary = item.primaryAction {
                        Button(primary) {
                            model.recordMockCommand("\(primary)：\(item.title)")
                        }
                            .buttonStyle(.borderedProminent)
                            .controlSize(.small)
                    }
                    if let secondary = item.secondaryAction {
                        Button(secondary) {
                            model.recordMockCommand("\(secondary)：\(item.title)", detail: "已进入通知审计命令边界。")
                        }
                            .buttonStyle(.bordered)
                            .controlSize(.small)
                    }
                }
            }
        }
    }
}

struct ConfirmSheet: View {
    var title: String
    var impact: String
    var risk: Tone
    var onConfirm: () -> Void

    var body: some View {
        VStack(alignment: .leading, spacing: 18) {
            Capsule()
                .fill(Color.secondary.opacity(0.25))
                .frame(width: 38, height: 5)
                .frame(maxWidth: .infinity)
            StatusPill(text: risk == .red ? "高风险操作" : "需要确认", tone: risk, symbol: "shield.lefthalf.filled")
            Text(title)
                .font(.title2.weight(.bold))
            Text(impact)
                .foregroundStyle(HiGoTheme.muted)
            Button("确认执行", action: onConfirm)
                .buttonStyle(.borderedProminent)
                .controlSize(.large)
                .frame(maxWidth: .infinity)
        }
        .padding(24)
        .presentationDetents([.medium])
    }
}

struct EmptyStateView: View {
    var title: String
    var message: String
    var symbol: String

    var body: some View {
        VStack(spacing: 14) {
            Image(systemName: symbol)
                .font(.system(size: 42))
                .foregroundStyle(HiGoTheme.muted)
            Text(title)
                .font(.headline)
            Text(message)
                .font(.subheadline)
                .foregroundStyle(HiGoTheme.muted)
                .multilineTextAlignment(.center)
        }
        .padding(28)
        .frame(maxWidth: .infinity)
    }
}

struct ErrorStateView: View {
    var message: String

    var body: some View {
        EmptyStateView(title: "加载失败", message: message, symbol: "wifi.exclamationmark")
    }
}

struct SkeletonView: View {
    var body: some View {
        VStack(spacing: 12) {
            ForEach(0..<4, id: \.self) { _ in
                RoundedRectangle(cornerRadius: 16, style: .continuous)
                    .fill(Color.gray.opacity(0.14))
                    .frame(height: 82)
                    .redacted(reason: .placeholder)
            }
        }
        .padding()
    }
}

struct SectionHeader: View {
    var title: String
    var action: String? = nil

    var body: some View {
        HStack {
            Text(title)
                .font(.headline.weight(.bold))
                .foregroundStyle(HiGoTheme.ink)
            Spacer()
            if let action {
                Text(action)
                    .font(.caption.weight(.semibold))
                    .foregroundStyle(Color.higoBlue)
            }
        }
        .padding(.horizontal, 2)
    }
}

struct PageContainer<Content: View>: View {
    @ViewBuilder var content: Content

    var body: some View {
        ScrollView {
            VStack(alignment: .leading, spacing: 18) {
                content
            }
            .padding(18)
            .padding(.bottom, 28)
        }
        .background(HiGoTheme.page.ignoresSafeArea())
    }
}

#Preview("Components") {
    PageContainer {
        HiGoCard {
            ActionRow(title: "家庭相册备份", subtitle: "正在备份 1,256 张照片", symbol: "icloud.and.arrow.up.fill", trailing: "72%")
        }
        MetricCard(title: "CPU", value: "18%", detail: "正常", symbol: "cpu", tone: .green)
        NotificationRow(item: NotificationItem(id: "preview-notice-backup-failed", title: "备份失败", message: "手机备份失败，请检查网络", time: "昨天", category: .backup, unread: true, tone: .red, primaryAction: "处理", secondaryAction: "审计"))
    }
    .environmentObject(AppViewModel(repository: MockRepository()))
}

