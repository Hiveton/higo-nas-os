import SwiftUI

@MainActor
private func previewModel() -> AppViewModel {
    AppViewModel.previewLoaded()
}

#Preview("Root - Light") {
    RootView()
        .environmentObject(previewModel())
}

#Preview("Root - Dark") {
    RootView()
        .environmentObject(previewModel())
        .preferredColorScheme(.dark)
}

#Preview("Root - Large Type") {
    RootView()
        .environmentObject(previewModel())
        .environment(\.dynamicTypeSize, .accessibility3)
}

#Preview("Photos") {
    NavigationStack { PhotosScreen() }
        .environmentObject(previewModel())
}

#Preview("AI") {
    NavigationStack { AiScreen() }
        .environmentObject(previewModel())
}

#Preview("Notifications") {
    NavigationStack { NotificationsScreen() }
        .environmentObject(previewModel())
}

#Preview("Backup") {
    NavigationStack { BackupScreen() }
        .environmentObject(previewModel())
}

#Preview("Shares") {
    NavigationStack { SharesScreen() }
        .environmentObject(previewModel())
}

#Preview("Profile") {
    NavigationStack { ProfileScreen() }
        .environmentObject(previewModel())
}

#Preview("Storage") {
    NavigationStack { StorageModuleScreen() }
        .environmentObject(previewModel())
}

#Preview("Settings") {
    NavigationStack { SettingsModuleScreen() }
        .environmentObject(previewModel())
}

#Preview("Docker") {
    NavigationStack { DockerModuleScreen() }
        .environmentObject(previewModel())
}

#Preview("Security") {
    NavigationStack { SecurityModuleScreen() }
        .environmentObject(previewModel())
}
