import Foundation

enum AppDataMode {
    case mock
}

struct AppContainer {
    let repository: HiGoRepositoryProtocol

    static func make(mode: AppDataMode = .mock) -> AppContainer {
        switch mode {
        case .mock:
            AppContainer(repository: MockRepository())
        }
    }
}
