import SwiftUI

#Preview("Home") {
    NavigationStack { HomeScreen() }
        .environmentObject(AppViewModel(repository: MockRepository()))
}

#Preview("Files") {
    NavigationStack { FilesScreen() }
        .environmentObject(AppViewModel(repository: MockRepository()))
}

#Preview("Device") {
    NavigationStack { DeviceScreen() }
        .environmentObject(AppViewModel(repository: MockRepository()))
}
