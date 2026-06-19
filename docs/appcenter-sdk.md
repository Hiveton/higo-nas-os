# App Center SDK — 三方应用开发指南

HiGoOS 应用中心是 **声明式 manifest 驱动** 的模块化系统。三方开发者只需提交一个
`manifest.json`，即可让自己的容器化应用被目录收录、以真实 docker 容器安装、并以
**内嵌 iframe 窗口** 在桌面外壳里呈现自己的界面 —— 全程经过 confirm / audit / rollback
治理，**无需改动或重新编译控制平面**。

- 机器可校验的 JSON Schema：[`appcenter-manifest.schema.json`](./appcenter-manifest.schema.json)
- 最小可跑示例：[`examples/appcenter/hello-app/`](../examples/appcenter/hello-app/)
- 后端实现：`server-go/internal/appcenter/`（`manifest.go` / `catalog.go` / `installer.go` /
  `governance.go` / `service.go`）

---

## 1. 应用如何被收录(三种目录来源)

应用中心目录(catalog)聚合三类来源，同 id 优先级 **local > remote > builtin**：

| 来源 | 位置 | 适用 |
| --- | --- | --- |
| **builtin** | 控制平面内置 `go:embed`(`internal/appcenter/seed/*.json`) | 官方种子应用 |
| **local** | `$HIGO_STATE_DIR/appcenter/apps/<id>/manifest.json` | **三方开发者放包即生效**，离线可用 |
| **remote** | 远程 registry 的 `index.json` | 应用市场 / 分发 |

> 放置本地包后，调用 `POST /api/v1/app-center/catalog/refresh` 或重启控制平面即可被扫描收录。

### 远程 registry index 格式

`index.json` 可以是 **manifest 数组**，或带 `apps` 字段的对象：

```json
{ "schemaVersion": "1", "apps": [ { /* manifest */ }, { /* manifest */ } ] }
```

通过 `POST /api/v1/app-center/registries` `{ "name": "...", "url": "https://.../index.json" }`
注册；非法 manifest 会被逐个校验跳过，不影响其余条目。

---

## 2. Manifest 字段速览

完整约束见 JSON Schema。要点：

```jsonc
{
  "schemaVersion": "1",            // 必填，固定 "1"
  "id": "hello-app",               // 必填，kebab-case 唯一
  "name": "Hello App",             // 必填
  "version": "1.0.0",              // 必填，目录发布的最新版本
  "category": "示例",
  "description": "...",
  "author": { "name": "...", "url": "..." },
  "iconUrl": "https://.../icon.png",
  "risk": "low",                   // 必填 low|medium|high — 决定治理强度
  "source": "third-party",         // official|community|third-party

  "containers": [                  // 必填，支持多容器
    {
      "name": "web",               // 容器名(应用内唯一)
      "image": "nginx:stable-alpine",
      "ports":   [{ "container": 80, "host": 18900, "protocol": "tcp" }],
      "volumes": [{ "name": "site", "path": "/usr/share/nginx/html" }],
      "env":     [{ "key": "GREETING", "value": "{{config.greeting}}" }],
      "command": "",
      "resources": { "cpu": 1, "memoryMb": 128 },
      "restartPolicy": "unless-stopped"
    }
  ],

  "webEntry": { "container": "web", "port": 18900, "path": "/", "display": "embed" },
  "permissions": ["network.publish:18900"],
  "config": [
    { "key": "greeting", "label": "欢迎语", "type": "text", "default": "Hello!" }
  ]
}
```

### 容器规格 → docker 的完整映射

安装器(`installer.go`)把每个 `ContainerSpec` 完整翻译为 docker 创建请求，**不丢字段**：

| manifest | docker |
| --- | --- |
| `image` | 镜像 |
| `ports[]` | `8080:80/tcp`(有 host)或 `80/tcp` |
| `volumes[]` | 命名卷 `<appId>_<name>:<path>`(按应用命名空间隔离) |
| `env[]` | `KEY=VALUE`，`VALUE` 支持 `{{config.KEY}}` 插值 |
| `resources.cpu` / `memoryMb` | CPU / 内存限制 |
| `restartPolicy` | 重启策略 |

单容器应用的容器名即 `id`；多容器为 `<id>-<containerName>`。

### Config 与插值

`config[]` 声明安装时向用户征集的参数。安装对话框据此渲染表单(`password`/`number`/`text`)。
容器 `env.value` 里的 `{{config.KEY}}` 在安装时按用户输入(或 `default`)插值。`secret: true`
的字段在前端以密码框输入，且不会回显到应用列表。

### Web UI 集成(`webEntry`)

- `display: "embed"`(默认)：应用安装并运行后，应用中心的 **打开** 按钮会以
  `http://<NAS主机>:<port><path>` 在一个 **内嵌 iframe 桌面窗口**(`AppFrameWindow`)里加载它。
  窗口动态注册，无需在 `App.vue` 写任何应用专属代码。
- `display: "external"`：在新浏览器标签打开。

---

## 3. 生命周期与治理(confirm / audit / rollback)

所有写操作都是 **两段式**：

1. **Preview** — `POST /api/v1/app-center/apps/{id}/{action}`
   返回 `confirmationId` + 人类可读的 **影响摘要**(镜像、端口、卷、权限、资源、风险)，**无副作用**。
2. **Confirm** — `POST /api/v1/app-center/apps/{id}/{action}/confirm`，body `{confirmationId, actor, config?}`
   校验后执行，写入审计并登记可回滚快照。

`action ∈ {install, update, start, stop, uninstall}`。`uninstall` 视为高风险(销毁容器)。
每次确认都会在 `GET /api/v1/app-center/audit` 留痕，可经
`POST /api/v1/app-center/audit/{id}/rollback` 反向回滚(install↔uninstall、start↔stop、
update 回退版本)。

docker 不可用时安装会 **乐观降级** 为本地状态(状态里注明降级原因)，不把 UI 留在半坏状态。

---

## 4. 三步上手

1. 复制 `examples/appcenter/hello-app/` 到 `$HIGO_STATE_DIR/appcenter/apps/hello-app/`。
2. `POST /api/v1/app-center/catalog/refresh`(或重启)→ 在应用中心“发现”页看到它。
3. 点击 **安装** → 填写 config → 确认影响摘要 → 应用以真实容器启动 → **打开** 看到内嵌界面。

---

## 5. MCP 工具

应用中心的每个端点都暴露为 MCP 工具(`higo.app-center.*`)：
`catalog.list/get/refresh`、`apps.list`、`apps.preview`、`apps.confirm`(destructive)、
`audit.list`、`audit.rollback`、`registries.list/add/remove`。治理在 HTTP 层执行，MCP 仅转发。
