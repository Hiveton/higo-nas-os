import XCTest
@testable import HiGoOSMediaApp

@MainActor
final class HiGoOSMediaAppTests: XCTestCase {
    func testStaticRepositoryPopulatesMediaDomains() {
        let repository = StaticMediaRepository()

        XCTAssertEqual(Set(repository.mediaItems().map(\.kind)), [.movies, .tv])
        XCTAssertEqual(repository.musicTracks().count, 3)
        XCTAssertEqual(repository.photoMemories().first?.title, "Summer Family Roll")
        XCTAssertEqual(repository.downloads().first?.title, "Deep Sea Archive")
        XCTAssertTrue(repository.nasDevice().isOnline)
    }

    func testViewModelDemoStateMutates() {
        let model = MediaAppViewModel(repository: StaticMediaRepository())

        model.searchText = "Deep"
        XCTAssertEqual(model.filteredMedia.map(\.id), ["movie-sea"])

        model.selectTrack(model.musicTracks[1])
        XCTAssertEqual(model.selectedTrack?.title, "Offline Morning")
        XCTAssertTrue(model.playback.isPlaying)

        model.toggleDownload("dl-1")
        XCTAssertEqual(model.downloads.first?.speed, "Paused")

        let oldCapacity = model.nasDevice.usedTB
        model.simulateCapacityChange()
        XCTAssertNotEqual(model.nasDevice.usedTB, oldCapacity)
    }
}
