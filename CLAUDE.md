# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## 项目概述

HiGoOS 是一个 AI 原生 NAS。仓库是一个分两部分的 monorepo:

- `server-go/` — Go 控制平面(`module higoos/server-go`)。对外提供版本化 REST API、托管前端静态资源、内嵌 MCP 服务器。
- `web-pc/` — Vue 3 + Vite 的"桌面操作系统"风格前端(窗口、Dock、小组件),通过 Go API 通信。
- `deploy/` — systemd 单元 + `deploy.sh`。`docs/` — API、治理与架构参考文档。

没有顶层构建工具;两部分各自独立构建。主分支用于发版,日常开发在 `devp` 分支。

## 测试 / 部署服务器

本地 Parallels 开发虚拟机 **HivetonDevelop**(Ubuntu 24.04 ARM64),用于部署验证:

- 地址:`10.211.55.3`(线上 UI:http://10.211.55.3:8080/,MCP:http://10.211.55.3:8080/mcp)
- 账号:`hiveton` ·  密码:`123qwe`(同时用作 sudo 密码)
- 一键部署:`SSH_PASS=123qwe ./deploy/deploy.sh`
- 该 VM 时钟容易在挂起后漂移,导致 apt 报 "Release file is not valid yet";需要时校时:`sudo date -u -s "<当前UTC>" && sudo timedatectl set-ntp true`。
- 注意:这是隔离的本地开发机,非生产环境;凭据仅用于本地验证。

## 常用命令

后端(`cd server-go`):
- 全量构建:`go build ./...`  ·  静态检查:`go vet ./...`
- 本地运行 API:`go run ./cmd/higo-api`(监听 `HIGO_HTTP_ADDR`,默认 `:8080`)
- 全部测试:`go test ./...`  ·  单包:`go test ./internal/video`  ·  单测:`go test ./internal/video -run TestScan`
- 交叉编译(目标 **linux/arm64**):`CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/higo-api` —— 纯 Go 无 CGO,交叉编译干净。`go.mod` 要求的 Go 版本比 NAS 主机自带的高,**永远在开发机编译,不要在主机编译**。

前端(`cd web-pc`):
- 安装:`npm ci`  ·  开发服务器:`npm run dev`(端口 5173,把 `/api` 代理到 `VITE_HIGOOS_API_BASE_URL` 或 `http://127.0.0.1:18082`)
- 构建(类型检查 + 打包):`npm run build`(先 `vue-tsc -b` 再 `vite build`,**类型错误会让构建失败**)
- 交互冒烟检查:`npm run test:interactions`(对关键组件/store 的静态断言脚本,非单元测试)

## 后端架构

**单进程、多业务域。** `cmd/higo-api` 是唯一的 HTTP 服务器。`httpapi.NewRouter(Dependencies)` 是组装入口:
- 构造约 19 个业务域服务(files、storage、docker、downloads、video、music、media、monitoring、settings、remote、backups、appcenter、accounts、security、steward、assistant、agents、desktop/devstub),用 **nil-fallback 模式**:`deps.X` 有传就用,否则 `X.NewServiceWithStateDir(cfg.StateDir)`,再有无状态兜底。测试通过 `Dependencies` 注入假实现。
- 服务都是带方法的普通 struct(不是接口)。路由器把它们挂在 `*API` 上,在 `*http.ServeMux` 注册约 130 条路由,再用中间件链包裹:`RequestID → recoverPanic → secureHeaders → cors → sessionGuard → accessLog`。
- 其它入口:`cmd/higo-worker`(后台进程)、`cmd/higoctl`(本地管理 CLI,`doctor`)、`cmd/higo-mcp`(MCP stdio 二进制)。

**响应信封。** 每个 handler 通过 `platform.WriteJSON`/`WriteError` 返回 `{ok, data, error:{code,message}, requestId}`。调用方(前端、MCP 的 apiclient)解开 `data` 或暴露 `error`。新增端点用 `allowMethod` 限制方法,沿用这套信封。

**Dev(Mac)与 NAS(Linux)差异。** 同一套服务在不同环境行为不同,差异在**构造时**决定:
- storage/monitoring 用 host 适配器,Linux 上 shell out 调 `lsblk`/`smartctl`/`df`;Mac 用确定性 `devstub` 假数据。
- files:设了 `HIGO_NAS_ROOT` → 真实文件系统仓库(`RootRepository`);未设 → `server-go/fixtures/nas-root` 下的 fixture 仓库。
- 可变开发态以 JSON 文件持久化在 `HIGO_STATE_DIR`(每域一个文件,如 `assistant.json`),即 `internal/state` 层,**不是数据库**。

## 架构要点(重要,容易踩坑)

- **当前是治理/编排骨架 + 部分真实适配器,不是完整生产实现。** 多数业务域的可变状态是 devstub/fixture + JSON 持久化;Linux 适配器(storage/monitoring/docker 等)在主机上才走真实命令。
- **鉴权安全现状 ⚠️**:`sessionGuard` 在 `HIGO_ENV=dev`/`test` 下**完全跳过**;其它环境只对 `/api/v1/*` 校验 cookie `higo_session` 或 `Authorization` 头。`internal/auth` 只有内存版 `DevSessionStore`(无持久化用户/密码后端)。**当前线上部署 `server.env` 是 `HIGO_ENV=dev`,等于无鉴权**——上生产前必须改环境并接入真实身份。
- **worker 只是心跳**:`cmd/higo-worker` 每 30s 打一次健康日志,**不驱动任何后台任务**(索引/媒体/备份扫描)。
- **异步任务是内联 goroutine**:返回 `taskId` 的端点(docker image pull、video transcode/scrape/subtitle、storage tasks、media jobs)由 service 方法内 spawn 的 goroutine 推进,**没有中央任务队列**。
- **状态并发模型**:每个 service 用一把 `sync.RWMutex` 粗粒度锁;`state.SaveJSON` 用「写临时文件 + `os.Rename`」做**原子写**,无事务、无跨域一致性保证。
- **流式 / 长连接端点**(不能走普通 request/response,故不映射为普通 MCP 工具):
  - WebSocket:Docker 容器终端 PTY(`internal/httpapi/docker_terminal.go`,gorilla/websocket + creack/pty)。
  - SSE:`/api/v1/events/stream`、`/api/v1/workflows/runs/{id}/events`、AI 助手消息流(`ai_handlers.go`)。
- **外部二进制硬依赖(仅 Linux 主机)**:`lsblk`/`smartctl`/`df`(存储/监控)、`docker`(Docker 域)、`ffmpeg`/`ffprobe`(视频/媒体)、`aria2c`(下载)。缺失则相关功能不可用——部署时需 `apt install ffmpeg aria2`。
- **数据是中文种子数据**:fixture/默认 space 名是中文(`家庭空间`/`团队空间`/`财务票据`…)。`GET /api/v1/files/tree` 的 `space` 查询参数要传中文值,或留空取整树——传 `home`/`team` 会 404。产品面向中文用户。

## MCP 层

后端把**每个** REST 端点都暴露为 MCP 工具(约 185 个)。两部分:
- `internal/apiclient` — 针对 `/api/v1/*` 的类型化 Go 客户端,解开信封。两种形态:`NewRemote(baseURL,…)`(真实 HTTP)、`NewInProcess(handler,…)`(`http.RoundTripper` 直接派发进 API mux,不走 TCP)。`URLResult` 用于二进制/流式端点(返回"去这个 URL 取"占位,不把字节流灌进 MCP)。
- `internal/mcp` — 工具目录。`BuildServer(cfg, client)` 为每个域注册一个 `register<Domain>`。每个工具 = 一个类型化输入 struct(SDK 反射出输入 schema)+ 调用某个 `apiclient` 方法、把 API JSON 以文本返回的 handler。`Out` 故意设为 `any`,SDK 因而不对任意 payload 做输出校验。注解 `readOnly()`/`mutating()`/`destructive()` 设风险提示。`HIGO_MCP_DOMAINS`(逗号分隔)过滤加载哪些域。

同一套目录两种方式提供:`higo-api` 内嵌 `/mcp`(Streamable HTTP,绑定进程内 client),以及独立 `cmd/higo-mcp` stdio 二进制(经 `HIGO_MCP_API_BASE`/`HIGO_MCP_API_TOKEN` 绑定远程 client,供 Claude Desktop/Code)。**MCP 层自身不做治理** —— 只转发 HTTP handler 的确认/审计流程。用官方 `github.com/modelcontextprotocol/go-sdk`。详见 `docs/mcp.md`。

新增工具:先在 `internal/apiclient/<domain>.go` 加端点方法,再在 `internal/mcp/tools_<domain>.go` 注册。

## 治理模型(横切,见 `docs/security-governance.md`)

AI/agent 动作受治理:风险等级 low/medium/high;medium/high 写操作端点先返回 `confirmationId` + 影响摘要,在调用对应 `*/confirm` 端点前不产生副作用;动作写只追加审计并登记回滚操作。由 `iam`/`audit`/`security`/`steward` 包实现。**新增写操作端点时,沿用现有 confirm/audit 模式,不要直接执行。**

## 前端架构

Vue 3 + Vite,**无路由** —— 是一个桌面外壳(`App.vue`),在 Dock/网格/小组件桌面上打开"窗口"(`src/components/windows/*Window.vue`)。状态在 `src/stores/*.ts`(desktop、monitoring、remote、settings)。所有后端访问走 `src/api/`:`runtime.ts`(fetch 封装、`ApiError`、SSE 辅助、从 `VITE_HIGOOS_API_*` 解析 base-URL/credentials)和 `client.ts` + `generated/`(各域类型化调用)。

## 配置 / 环境变量

后端(`internal/platform/config.go`):`HIGO_HTTP_ADDR`、`HIGO_ENV`、`HIGO_VERSION`、`HIGO_PUBLIC_ORIGIN`(CORS,`*` 放行任意来源)、`HIGO_STATE_DIR`(JSON 状态根)、`HIGO_NAS_ROOT`(真实 FS 根)、`HIGO_STATIC_DIR`(托管 web 产物)、`HIGO_MCP_ENABLED`、`HIGO_MCP_DOMAINS`。前端:`VITE_HIGOOS_API_BASE_URL`、`VITE_HIGOOS_API_CREDENTIALS`。

## 部署

`deploy/deploy.sh` 交叉编译三个 Go 二进制(linux/arm64)、构建 `web-pc`、rsync 到主机并重启 systemd。可复用:`SSH_PASS=<密码> [HOST=10.211.55.3] [SKIP_DEPS=1] ./deploy/deploy.sh`。安装路径:二进制 `/opt/higoos/bin/`、web `/opt/higoos/web/`、环境文件 `/etc/higoos/server.env`,以 systemd 服务运行 `higo-api`/`higo-worker`。视频/下载功能需主机有 `ffmpeg`、`aria2`。`docs/deployment-installed-software.md` 记录主机软件包基线。
