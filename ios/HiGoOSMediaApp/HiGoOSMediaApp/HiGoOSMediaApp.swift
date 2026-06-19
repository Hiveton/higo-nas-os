import SwiftUI

@main
struct HiGoOSMediaApp: App {
    @StateObject private var model = MediaAppViewModel(repository: StaticMediaRepository())

    var body: some Scene {
        WindowGroup {
            MediaRootView()
                .environmentObject(model)
        }
    }
}
