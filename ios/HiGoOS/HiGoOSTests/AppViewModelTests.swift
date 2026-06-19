import XCTest
@testable import HiGoOS

@MainActor
final class AppViewModelTests: XCTestCase {
    func testLoadPopulatesAllPrimaryMockDomains() async {
        let model = AppViewModel(repository: MockRepository())

        await model.load()

        XCTAssertTrue(model.isLoaded)
        XCTAssertEqual(model.systemStatus.name, "我的 NAS")
        XCTAssertEqual(model.files.count, 7)
        XCTAssertEqual(model.files.first?.id, "folder-documents")
        XCTAssertEqual(model.files.first?.kind, .folder)
        XCTAssertEqual(model.files.first?.space, .personal)
        XCTAssertEqual(model.photos.count, 4)
        XCTAssertEqual(model.backupJobs.count, 4)
        XCTAssertEqual(model.backupJobs.first?.state, .running)
        XCTAssertEqual(model.notifications.count, 4)
        XCTAssertEqual(model.notifications.first?.category, .system)
        XCTAssertEqual(model.aiMessages.count, 2)
        XCTAssertEqual(model.aiResults.count, 2)
        XCTAssertEqual(model.shares.count, 3)
        XCTAssertEqual(model.containers.count, 3)
        XCTAssertEqual(model.containers.first?.status, .running)
        XCTAssertEqual(model.apps.count, 3)
        XCTAssertEqual(model.audit.count, 3)
        XCTAssertEqual(model.modules.count, ModuleID.allCases.count)
        XCTAssertEqual(model.featureCategories.count, FeatureCategoryKind.allCases.count)
        XCTAssertEqual(model.tasks.count, 6)
        XCTAssertEqual(model.downloads.count, 3)
        XCTAssertEqual(model.confirmations.count, 2)
        XCTAssertEqual(model.mediaLibraries.count, 3)
    }

    func testAllSecondaryModulesHaveNavigationMetadata() async {
        let modules = await MockRepository().loadModules()
        let ids = Set(modules.map(\.id))

        XCTAssertEqual(ids, Set(ModuleID.allCases))
        XCTAssertFalse(modules.contains { $0.title.isEmpty || $0.subtitle.isEmpty || $0.icon.isEmpty })
    }

    func testDomainModelsUseStableCodableIdentity() throws {
        let original = FileItem(
            id: "file-contract-pdf",
            name: "合同.pdf",
            kind: .pdf,
            size: "320 KB",
            date: "2026/06/19",
            space: .shared,
            tags: ["合同"],
            icon: "doc.richtext.fill",
            tone: .red
        )

        let encoded = try JSONEncoder().encode(original)
        let decoded = try JSONDecoder().decode(FileItem.self, from: encoded)

        XCTAssertEqual(decoded, original)
        XCTAssertEqual(decoded.id, "file-contract-pdf")
        XCTAssertEqual(decoded.kind.rawValue, "PDF")
        XCTAssertEqual(decoded.space.rawValue, "共享")
    }

    func testFeatureStoresExposeScreenScopedData() async {
        let model = AppViewModel(repository: MockRepository())

        await model.load()

        XCTAssertEqual(model.homeStore.quickModules.map(\.id), [.backup, .shares, .videoCenter, .downloadCenter])
        XCTAssertEqual(model.homeStore.todayTasks.count, 3)
        XCTAssertGreaterThan(model.homeStore.riskCount, 0)
        XCTAssertEqual(model.taskStore.visible(for: .needsConfirmation).map(\.id), ["task-ai-confirm"])
        XCTAssertEqual(model.taskStore.attentionCount, 4)
        XCTAssertEqual(model.homeStore.unreadNotificationCount, 3)
        XCTAssertEqual(model.filesStore.folders(in: .personal).map(\.id), ["folder-documents", "folder-work"])
        XCTAssertEqual(model.filesStore.folders(in: .team).map(\.id), ["folder-team"])
        XCTAssertEqual(model.filesStore.recentFiles(in: .family).map(\.id), ["file-invoice-docx"])
        XCTAssertEqual(model.notificationsStore.visible(for: .unread).count, 3)
        XCTAssertEqual(model.notificationsStore.visible(for: .backup).map(\.id), ["notice-backup-complete", "notice-backup-failed"])
    }
}
