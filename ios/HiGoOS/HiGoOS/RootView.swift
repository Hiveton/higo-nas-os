import SwiftUI

struct RootView: View {
    @EnvironmentObject private var model: AppViewModel
    @State private var selectedTab: MobileTab = .home
    private let tabs: [MobileTab] = MobileTab.allCases

    var body: some View {
        VStack(spacing: 0) {
            selectedContent
            CustomTabBar(
                tabs: tabs,
                selectedTab: $selectedTab,
                attentionCount: model.taskStore.attentionCount
            )
        }
        .background(HiGoTheme.page.ignoresSafeArea())
        .task {
            if !model.isLoaded {
                await model.load()
            }
        }
        .alert(item: $model.feedback) { feedback in
            Alert(
                title: Text(feedback.title),
                message: Text(feedback.message),
                dismissButton: .default(Text("知道了"))
            )
        }
    }

    @ViewBuilder
    private var selectedContent: some View {
        switch selectedTab {
        case .home:
            NavigationStack { HomeScreen() }
        case .files:
            NavigationStack { FilesScreen() }
        case .photos:
            NavigationStack { PhotosScreen() }
        case .assistant:
            NavigationStack { AiScreen() }
        case .tasks:
            NavigationStack { MobileTasksScreen() }
        case .profile:
            NavigationStack { ProfileScreen() }
        }
    }
}

struct CustomTabBar: View {
    var tabs: [MobileTab]
    @Binding var selectedTab: MobileTab
    var attentionCount: Int

    var body: some View {
        HStack(spacing: 4) {
            ForEach(tabs, id: \.self) { tab in
                Button {
                    selectedTab = tab
                } label: {
                    VStack(spacing: 4) {
                        ZStack(alignment: .topTrailing) {
                            Image(systemName: tab.icon)
                                .font(.system(size: 20, weight: .semibold))
                                .frame(height: 22)
                            if tab == .tasks && attentionCount > 0 {
                                Text("\(min(attentionCount, 9))")
                                    .font(.system(size: 10, weight: .bold))
                                    .foregroundStyle(.white)
                                    .frame(width: 16, height: 16)
                                    .background(Color.higoRed)
                                    .clipShape(Circle())
                                    .offset(x: 10, y: -7)
                            }
                        }
                        Text(tab.title)
                            .font(.system(size: 11, weight: .semibold))
                            .lineLimit(1)
                    }
                    .foregroundStyle(selectedTab == tab ? Color.higoBlue : HiGoTheme.ink.opacity(0.72))
                    .frame(maxWidth: .infinity)
                    .padding(.vertical, 9)
                    .background {
                        if selectedTab == tab {
                            Capsule()
                                .fill(Color.higoBlue.opacity(0.12))
                        }
                    }
                }
                .buttonStyle(.plain)
                .accessibilityLabel(tab.title)
                .accessibilityIdentifier(tab.accessibilityIdentifier)
            }
        }
        .padding(.horizontal, 12)
        .padding(.top, 8)
        .padding(.bottom, 10)
        .background(HiGoTheme.card.opacity(0.96))
        .clipShape(RoundedRectangle(cornerRadius: 28, style: .continuous))
        .overlay {
            RoundedRectangle(cornerRadius: 28, style: .continuous)
                .stroke(Color.white.opacity(0.75), lineWidth: 1)
        }
        .shadow(color: HiGoTheme.shadow, radius: 18, x: 0, y: -4)
        .padding(.horizontal, 14)
        .padding(.bottom, 8)
    }
}

#Preview {
    RootView()
        .environmentObject(AppViewModel.previewLoaded())
}
