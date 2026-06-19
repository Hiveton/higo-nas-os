# HiGoOS 助手 (higo-assistant)

跨平台桌面工具(Qt 6 / C++),对标群晖 *Synology Assistant*。职责**只覆盖浏览器做不到的"上 Web 之前"那几步**:

- **局域网发现**:UDP 广播探测,实时列出同网段的 HiGoOS 设备(协议 `HIGOOS/1`,见 [`docs/desktop-scanner.md`](../../docs/desktop-scanner.md) §4)。
- **首次配置 IP / 主机名**:对全新设备做两段式(确认 + 审计)网络配置。
- **打开管理界面**:其余所有业务功能一律 `打开管理界面` 跳到设备内置 Web UI —— 本工具不重复实现业务。

## 目录结构

```
tools/higo-assistant/
  CMakeLists.txt
  src/
    main.cpp
    discovery/   UdpDiscoverer (QUdpSocket 广播发现) + Device 模型
    api/         HigoClient (REST,解 {ok,data,error} 信封;identity/login/network)
    ui/          MainWindow (设备列表) + NetworkConfigDialog (配 IP)
```

## 构建

依赖:CMake ≥ 3.21、Qt 6(Widgets + Network)。

```bash
cd tools/higo-assistant
cmake -B build -DCMAKE_PREFIX_PATH=<你的 Qt6 安装路径>   # 如 ~/Qt/6.7.0/macos
cmake --build build
./build/bin/higo-assistant            # macOS/Linux
# Windows: build\bin\higo-assistant.exe,再用 windeployqt 收集依赖
```

> Qt 安装目录示例:macOS `~/Qt/6.7.0/macos`、Windows `C:\Qt\6.7.0\msvc2019_64`、Linux 发行版包或 `~/Qt/6.7.0/gcc_64`。

## 与后端的依赖关系

本工具按 [`docs/desktop-scanner.md`](../../docs/desktop-scanner.md) 的接口编写,后端配套**已实现**:

1. ✅ `internal/discovery` UDP 发现守护(`HIGO_DISCOVERY_ADDR`,默认 `:19999`,回应 `announce` 指纹)。
2. ✅ 公开端点 `GET /api/v1/system/identity`(无鉴权,设备指纹)。
3. ✅ `internal/network` 域 + `/api/v1/network/*`(两段式预览/确认配 IP + 审计 + 回滚)。
4. ✅ `POST /api/v1/auth/login`(下发 `higo_session` 会话 + CSRF)。

在同一局域网启动后端(NAS 上 `higo-api`,或开发机 `go run ./cmd/higo-api`)后,本工具即可发现设备。开发机同机调试可设 `HIGO_DISCOVERY_ADDR=127.0.0.1:19999`。

## 已知约束

HTTP 配 IP 存在"鸡生蛋":Qt 必须先能 TCP 连到设备当前 IP 才能配。设备在错网段 / link-local(169.254)时,UDP 能发现但配不了——详见文档 §7。
