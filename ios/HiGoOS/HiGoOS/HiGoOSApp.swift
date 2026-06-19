import SwiftUI

@main
struct HiGoOSApp: App {
    @StateObject private var model: AppViewModel

    init() {
        _model = StateObject(wrappedValue: AppViewModel(repository: MockRepository()))
    }

    var body: some Scene {
        WindowGroup {
            RootView()
                .environmentObject(model)
        }
    }
}
