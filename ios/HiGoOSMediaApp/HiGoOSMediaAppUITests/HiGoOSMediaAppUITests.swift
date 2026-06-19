import XCTest

final class HiGoOSMediaAppUITests: XCTestCase {
    func testPrimaryTabsRender() {
        let app = XCUIApplication()
        app.launch()

        for title in ["Library", "Discover", "Downloads", "Devices", "Settings"] {
            let button = app.buttons["nav-\(title.lowercased())"]
            XCTAssertTrue(button.waitForExistence(timeout: 6), "Missing tab: \(title)")
            button.tap()
        }

        XCTAssertTrue(app.staticTexts["Settings"].waitForExistence(timeout: 5))
    }

    func testDownloadsCanPauseAndResume() {
        let app = XCUIApplication()
        app.launch()

        app.buttons["nav-downloads"].tap()
        let toggle = app.buttons["download-toggle-dl-1"]
        XCTAssertTrue(toggle.waitForExistence(timeout: 5))
        toggle.tap()
        XCTAssertTrue(app.staticTexts["Paused"].waitForExistence(timeout: 3))
        toggle.tap()
        XCTAssertTrue(app.staticTexts["9.8 MB/s"].waitForExistence(timeout: 3))
    }
}
