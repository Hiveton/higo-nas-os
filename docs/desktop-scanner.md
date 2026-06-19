# 桌面端 NAS 扫描配置工具 — 设计文档

> 状态:设计草案(待评审)
> 目标:用户拿到一台全新的 HiGoOS NAS 后,在同一局域网用一个桌面工具(Qt)**发现设备 → 识别确认 → 配置 IP/主机名 → 跳转 Web 初始化**。对标群晖 *Synology Assistant* / 威联通 *Qfinder*。

## 1. 已确认的方向

| 决策点 | 选择 |
|---|---|
| 发现机制 | **UDP 广播发现守护**(后端新增轻量守护 + Qt 端广播探测) |
| IP 配置落地 | **新增 HTTP 网络配置端点**(`/api/v1/network/*`,走现有信封 + 治理) |
| 本次产出 | 详细方案 + 协议设计(本文档) |

## 2. 后端现状(为什么这些都得新建)

并行排查结论,关键能力几乎都没有:

- **无任何 LAN 广播/发现**:`higo-api` 只起一个 TCP HTTP server,不广播自己。搜索 `mdns/zeroconf/ssdp/udp/broadcast` 均无命中。
- **无设备指纹端点**:只有 `/healthz`、`/readyz` 返回 `{"status":"ok"}`,无机型/版本/deviceId,无法据此判定"这是 HiGoOS"。`GET /api/v1/system/info` 有信息但**要鉴权**(prod 下 401)。
- **无网络配置端点**:`Settings` 只有 Model/Privacy/UI,无 IP/主机名/网关/DNS 任何字段。
- **无首次开机/认领设备流程**。
- **无稳定 deviceId**:`SystemInfo` 结构里没有持久设备标识,需要新建。
- 登录侧 `VerifyPassword` + `SessionStore`(新加的 `internal/auth/store.go`)已就绪,但 `/api/v1/auth/login` 的 HTTP handler **还没接**。

## 3. 总体架构

```
┌─────────────────────────┐         UDP :19999 广播/单播          ┌──────────────────────────────┐
│   Qt 桌面工具            │  ──── discover 探测 ───────────────▶ │  HiGoOS NAS (higo-api 进程)    │
│  desktop-scanner/        │  ◀─── announce 指纹应答 ──────────── │                                │
│                          │                                       │  internal/discovery (新)        │
│  - UdpDiscoverer         │                                       │    UDP 发现守护(goroutine)     │
│  - HigoClient (REST)     │         HTTP :8080 (确认 IP 后)       │                                │
│  - MainWindow 设备列表   │  ──── GET /api/v1/system/identity ─▶ │  internal/httpapi (改)          │
│  - NetworkConfigDialog   │  ──── PUT /api/v1/network/config ──▶ │    + 公开 identity 端点         │
│  - 跳转浏览器到 Web      │  ──── POST .../config/confirm ─────▶ │    + network 域路由 + roleGate  │
└─────────────────────────┘                                       │  internal/network (新)          │
                                                                   │    host 适配器(netplan) / devstub│
                                                                   └──────────────────────────────┘
```

设计原则:**沿用本仓库既有约定**——
- 单进程多业务域:发现守护与 network 域都挂在 `higo-api`,构造时按环境(Mac=devstub / Linux=host 真实命令)分流,与 storage/monitoring 一致。
- 响应信封 `{ok,data,error,requestId}`、`platform.WriteJSON/WriteError`、`allowMethod`。
- 写操作走治理:改 IP 属高风险,两段式 `confirmationId` + 影响摘要 + 审计 + 回滚(见 `docs/security-governance.md`)。
- network 域同时**自动暴露为 MCP 工具**(`internal/apiclient` + `internal/mcp/tools_network.go`),与全仓一致。

## 4. UDP 发现协议(v1)

### 4.1 传输
- 端口:UDP `19999`,可配 `HIGO_DISCOVERY_ADDR`(默认 `:19999`)。
- 发现守护监听 `0.0.0.0:19999`,对收到的 `discover` 包**单播回应**到来源地址。
- 同一广播域即可被发现(跨网段需路由器转发广播,通常收不到——见 §7 鸡生蛋说明)。
- 所有包为单条 UDP datagram,UTF-8 JSON,首字段 `magic` 用于快速过滤非本协议流量。

### 4.2 探测包(Qt → NAS,广播)
发往各网卡的广播地址(`255.255.255.255` 及每个接口的定向广播地址)`:19999`:
```json
{ "magic": "HIGOOS/1", "type": "discover", "nonce": "9f2c…", "replyPort": 0 }
```
- `nonce`:客户端随机串,用于把应答和本次扫描配对、防陈旧包。
- `replyPort`:为 0 时回到来源端口;非 0 时回到指定端口(便于客户端固定收包端口)。

### 4.3 应答包(NAS → Qt,单播)
```json
{
  "magic": "HIGOOS/1",
  "type": "announce",
  "nonce": "9f2c…",
  "deviceId": "hg-7a3f9c12",
  "model": "HiGoOS NAS",
  "version": "1.4.0",
  "hostname": "higoos",
  "initialized": false,
  "httpPort": 8080,
  "https": false,
  "primaryMac": "aa:bb:cc:dd:ee:ff",
  "addrs": ["10.211.55.3"],
  "netMode": "dhcp",
  "uptimeSec": 142
}
```
- **只放安全指纹**,不含任何敏感信息(无序列号细节/无凭据/无内部路径)。
- `initialized`:是否已完成首次初始化(决定 Qt 是引导"初始化"还是"登录")。
- `addrs` + `httpPort`:Qt 据此拼出 `http://<addr>:<httpPort>/` 与 REST base。
- `primaryMac`:作为设备稳定标识与(未来)定向配置的 selector。

### 4.4 主动公告(可选,v1.1)
设备启动时向广播地址发 1~3 次 `type:"announce"`(带空 nonce),让正在监听的工具即时捕获,缩短"插电后多久能看到"。v1 先不做,query/response 足够。

### 4.5 安全约束
- 发现守护**只读、不接受任何配置指令**(配置一律走 HTTP + 鉴权 + 治理),避免无认证 UDP 改设备。
- 限流:对同一来源 IP 的应答做简单速率限制,防放大攻击(UDP 应答体远小于请求,放大比≈1,风险低但仍限)。
- 可配开关 `HIGO_DISCOVERY_ENABLED`(默认开);企业环境可关。

## 5. 后端改造清单

### 5.1 新增 `internal/discovery`
```
internal/discovery/
  daemon.go      // Listen(ctx, addr, IdentityFunc) ;解析 discover、回 announce
  identity.go    // Identity struct + 组装(deviceId/version/hostname/addrs/mac/initialized)
  daemon_test.go // 回环 UDP:发 discover 收 announce,校验 nonce 回显与字段
```
- 在 `cmd/higo-api/main.go` 起一个 goroutine `discovery.Run(ctx, cfg, identityFn)`,随主进程生命周期退出。
- `IdentityFunc` 由路由组装层注入,内部复用与 HTTP identity 端点**同一份** `Identity()`,避免两处漂移。

### 5.2 稳定 deviceId + 公开 identity 端点
- 新建 `internal/identity`(或并入 platform):首次生成 `hg-<8hex>` 持久化到 `HIGO_STATE_DIR/identity.json`,之后只读。
- 新增 `GET /api/v1/system/identity`(**无鉴权**):返回与 UDP announce 同构的安全指纹。需在 `requiresSession()` 里加一条公开白名单(目前它对整个 `/api/v1/*` 一律要鉴权)——只放这一个只读路径。
  - 返回示例(信封内 `data`):同 §4.3 字段集。
- 用途:Qt 拿到 IP 后,二次确认 + 展示详情;也给浏览器/其它客户端统一的身份来源。

### 5.3 新增 `internal/network`(域服务)
```
internal/network/
  service.go         // GetInterfaces / GetConfig / PlanConfig(返回 confirmation)/ ApplyConfig
  types.go           // Interface, NetworkConfig{Mode,Address,Prefix,Gateway,DNS[],Hostname}
  host_linux.go      // 真实适配器:读 `ip -j addr` / `ip -j route`;写 /etc/netplan/99-higoos.yaml + `netplan apply`;回滚保留旧 yaml
  devstub.go         // Mac/dev:内存 + JSON 持久化(HIGO_STATE_DIR/network.json),假数据
  service_test.go
```
路由(`internal/httpapi`,沿用 `allowMethod` + 信封 + 治理):
| 方法 | 路径 | 风险 | 说明 |
|---|---|---|---|
| GET | `/api/v1/network/interfaces` | low | 列网卡:name/mac/link/addrs/mode |
| GET | `/api/v1/network/config` | low | 当前生效配置 |
| PUT | `/api/v1/network/config` | **high** | 计划变更 → 返回 `confirmationId` + 影响摘要(不落地) |
| POST | `/api/v1/network/config/confirm` | high | 凭 `confirmationId` 落地;写审计 + 登记回滚 |

- 改 IP 属高风险(可能切断当前连接),**必须两段式**;`ApplyConfig` 在 apply 后做一次自检(新地址可达性 / netplan try 的回滚窗口),失败自动回退旧 yaml。
- 受 `authz.go` 的 `roleGate` 约束:`/network/` 写操作需 **admin** 会话(prod)。
- MCP:`internal/apiclient/network.go` + `internal/mcp/tools_network.go`,注解 `mutating()`/`destructive()`。

### 5.4 登录端点(配置的前置)
prod 下改网络需 admin 会话。需补 `POST /api/v1/auth/login`(已有 `VerifyPassword` + `SessionStore`,只差 handler):接收账号密码 → `VerifyPassword` → `SessionStore.Issue` → 下发 `higo_session` cookie + CSRF。Qt 端用此登录拿会话再配 IP。

## 6. Qt 桌面工程

建议新建顶层目录 `desktop-scanner/`(独立构建,符合"无顶层构建工具、各部分独立"的仓库风格)。技术选型:**Qt 6 + Widgets + CMake**(工具类应用,Widgets 出活快;后续要更漂亮可换 QML)。

```
desktop-scanner/
  CMakeLists.txt
  src/
    main.cpp
    discovery/
      UdpDiscoverer.h/.cpp     // QUdpSocket;枚举网卡广播地址;发 discover;收 announce;按 deviceId 去重聚合
      Device.h                 // 设备模型(指纹字段 + 最后可见时间)
    api/
      HigoClient.h/.cpp        // QNetworkAccessManager;解信封;identity / login / network config 调用
    ui/
      MainWindow.h/.cpp        // 设备表格 + 刷新 + 详情面板 + "打开 Web"/"配置网络"动作
      NetworkConfigDialog.h/.cpp // DHCP/静态切换、地址/掩码/网关/DNS/主机名表单;两段式确认
  resources/                   // 图标等
```
交互流:
1. 启动即周期广播扫描(每 2–3s 一轮),`UdpDiscoverer` 聚合 announce,表格实时增删(超时未见则灰显)。
2. 选中设备 → 右侧详情(`HigoClient.identity(ip)` 二次确认在线 + 取最新字段)。
3. **打开 Web**:`QDesktopServices::openUrl("http://<ip>:<port>/")`,`initialized=false` 走初始化、`true` 走登录。
4. **配置网络**:弹 `NetworkConfigDialog` → 先登录拿会话(prod)→ `PUT /network/config` 取影响摘要 → 用户确认 → `POST /network/config/confirm` → 提示设备可能换 IP,重新扫描。
5. 打包:`windeployqt`/`macdeployqt`(后续阶段)。

## 7. 已知约束:HTTP 配 IP 的"鸡生蛋"

选了 HTTP 配 IP,就有一个固有边界:**Qt 必须先能 TCP 连到 NAS 当前 IP** 才能配。覆盖与不覆盖:

- ✅ NAS 已从 DHCP 拿到地址、和电脑同网段 → UDP 发现拿到 `addrs`,直接 HTTP 配,顺畅。
- ⚠️ NAS 在另一个网段 / 拿了 169.254 link-local / DHCP 不可用 → UDP 应答(同广播域)能看到设备,但 **HTTP 连不上**,配不了。

缓解(按需后续做,不在 v1):
- Qt 检测到目标 IP 不可达且与本机不同网段时,引导用户**临时给本机网卡加一个同网段 IP 别名**以建立连接(群晖 Assistant 类似做法);或
- v2 在发现守护上加一条**带挑战应答的定向"临时设地址"UDP 指令**(仅在 `initialized=false` 未初始化窗口内可用,落地后即关闭),专门破解 link-local/错网段场景。

v1 先在 UI 明确提示这一边界即可。

## 8. 分阶段落地计划

- **阶段 0(后端可独立验证)**:`internal/identity` + 公开 `GET /api/v1/system/identity` + `internal/discovery` 守护。Mac 上 `go run ./cmd/higo-api`,用 `nc -u` 或一段小脚本发 discover 验证 announce。
- **阶段 1**:`internal/network` 域(devstub 先行)+ 四个端点 + 两段式治理 + 审计;`go test ./internal/network`。
- **阶段 2**:Linux host 适配器(`ip -j` 解析 + netplan 写入/apply/回滚),部署到 Parallels VM `HivetonDevelop`(`SSH_PASS=123qwe ./deploy/deploy.sh`)验证真实改 IP + 回滚。
- **阶段 3**:`POST /api/v1/auth/login` handler 接线(打通 prod 鉴权)。
- **阶段 4**:Qt 工程骨架 → UDP 发现 + 设备列表 + 打开 Web(最先可演示)。
- **阶段 5**:Qt 网络配置对话框 + 登录 + 两段式确认;打包。
- **阶段 6(可选)**:主动公告、link-local/错网段破解、MCP 工具补全。

## 9. 待定/需后续拍板

1. **首次配置的鉴权**:全新 NAS 还没设管理员密码时,谁有权改 IP?方案 A:`initialized=false` 窗口内允许免登录配 IP(便利,有风险);方案 B:始终要 `BootstrapAdmin` 打印的一次性密码(安全,需把密码呈现给用户——设备标签/控制台)。倾向 B。
2. **UDP 端口** 19999 是否与现网冲突、是否需要随产品定。
3. Qt UI 用 **Widgets(快)** 还是 **QML(美)**——v1 建议 Widgets。
