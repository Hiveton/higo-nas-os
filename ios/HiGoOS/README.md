# HiGoOS iOS

Native SwiftUI UI-first implementation for the HiGoOS mobile app.

## Current Scope

- Mock-only data. No backend dependency and no WebView wrapper.
- Six fixed bottom entries: 首页, 文件, 相册, 助手, 任务, 我的.
- Mobile-first information architecture: 家庭总控台, 文件工作流, 相册备份, HiGoOS 助手, 任务/通知中心, account/admin hub.
- Secondary modules are grouped as 数据管理, 媒体娱乐, 设备运维, 安全与成员, 高级服务.
- Repository protocols mirror future backend domains so real API implementations can replace `MockRepository` later.
- Verification covers build, unit tests, UI tab navigation, simulator launch, and screenshot capture.

## Build

```bash
cd ios/HiGoOS
xcodegen generate
xcodebuild -project HiGoOS.xcodeproj -scheme HiGoOS -destination "generic/platform=iOS Simulator" build
```

## Verify UI

```bash
cd ios/HiGoOS
scripts/verify-ios-ui.sh
```

Optional environment variables:

- `DEVICE_NAME`: simulator name, default `iPhone 17`
- `SCREENSHOT_PATH`: screenshot output path, default `/tmp/higoos-ios-ui.png`
- `BOOT_AND_LAUNCH=0`: build only, skip simulator launch
- `RUN_TESTS=0`: skip unit tests during simulator verification
- `DERIVED_DATA_PATH`: fixed build output path, default `ios/HiGoOS/build/DerivedData`
