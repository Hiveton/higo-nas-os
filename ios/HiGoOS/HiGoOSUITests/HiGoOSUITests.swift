import XCTest

final class HiGoOSUITests: XCTestCase {
    func testSixPrimaryTabsAreVisibleAndNavigable() {
        let app = XCUIApplication()
        app.launch()

        for identifier in ["tab-home", "tab-files", "tab-photos", "tab-assistant", "tab-tasks", "tab-profile"] {
            let button = app.buttons[identifier]
            XCTAssertTrue(button.waitForExistence(timeout: 5), "Missing tab: \(identifier)")
            button.tap()
            XCTAssertTrue(button.exists)
        }

        XCTAssertFalse(app.buttons["More"].exists)
    }

    func testPrimaryScreensRenderExpectedContent() {
        let app = XCUIApplication()
        app.launch()

        XCTAssertTrue(app.staticTexts["HiGoOS"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["我的 NAS"].exists)

        app.buttons["tab-files"].tap()
        XCTAssertTrue(app.navigationBars["文件"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["上传"].exists)

        app.buttons["tab-photos"].tap()
        XCTAssertTrue(app.navigationBars["相册"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["智能相册"].exists)

        app.buttons["tab-assistant"].tap()
        XCTAssertTrue(app.navigationBars["助手"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["本地优先"].exists)

        app.buttons["tab-tasks"].tap()
        XCTAssertTrue(app.navigationBars["任务"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["确认队列"].exists)

        app.buttons["tab-profile"].tap()
        XCTAssertTrue(app.navigationBars["我的"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["设备与管理"].exists)
    }

    func testSecondaryModuleSearchAndDetailInteraction() {
        let app = XCUIApplication()
        app.launch()

        let allModules = app.buttons["home-all-modules"]
        XCTAssertTrue(allModules.waitForExistence(timeout: 5))
        if !allModules.isHittable {
            app.scrollViews.firstMatch.swipeUp()
        }
        allModules.tap()
        XCTAssertTrue(app.navigationBars["全部功能"].waitForExistence(timeout: 5))

        let search = app.textFields["搜索功能、管理项或协议"]
        XCTAssertTrue(search.waitForExistence(timeout: 5))
        search.tap()
        search.typeText("Docker")

        let docker = app.buttons["module-docker"]
        XCTAssertTrue(docker.waitForExistence(timeout: 5))
        docker.tap()
        XCTAssertTrue(app.navigationBars["Docker"].waitForExistence(timeout: 5))

        XCTAssertTrue(app.staticTexts["jellyfin"].waitForExistence(timeout: 5))
        app.staticTexts["jellyfin"].tap()
        XCTAssertTrue(app.navigationBars["jellyfin"].waitForExistence(timeout: 5))

        let cpuMetric = app.buttons["detail-metric-CPU"]
        XCTAssertTrue(cpuMetric.waitForExistence(timeout: 5))
        cpuMetric.tap()
        XCTAssertTrue(app.navigationBars["详情"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["审计策略"].waitForExistence(timeout: 5))
    }

    func testFileSharePhotoBackupAiConfirmationAndTaskFilter() {
        let app = XCUIApplication()
        app.launch()

        app.buttons["tab-files"].tap()
        XCTAssertTrue(app.navigationBars["文件"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.buttons["上传"].exists)
        XCTAssertTrue(app.buttons["进入多选"].exists)

        app.buttons["tab-photos"].tap()
        XCTAssertTrue(app.navigationBars["相册"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["自动备份已开启"].waitForExistence(timeout: 5))
        if !app.staticTexts["RAW"].exists {
            app.scrollViews.firstMatch.swipeUp()
        }
        XCTAssertTrue(app.staticTexts["RAW"].waitForExistence(timeout: 5))

        app.buttons["tab-assistant"].tap()
        XCTAssertTrue(app.navigationBars["助手"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.staticTexts["确认队列"].waitForExistence(timeout: 5))

        app.buttons["tab-tasks"].tap()
        XCTAssertTrue(app.navigationBars["任务"].waitForExistence(timeout: 5))
        XCTAssertTrue(app.segmentedControls["task-filter"].waitForExistence(timeout: 5))
    }
}
