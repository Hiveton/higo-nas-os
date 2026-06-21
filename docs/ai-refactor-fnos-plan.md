# HiGoOS AI 重构方案 —— 对齐飞牛 fnOS 逻辑

> 目标:把 HiGoOS 的 AI 能力从「云 LLM/sidecar 代理、能力分散在多个窗口」重构为飞牛 fnOS 的核心逻辑——
> **本地下载模型、设备上推理、无需 API key、完全私有,且以「相册」为中心**;对话/Agent 仍走可配置 LLM(混合架构)。
> 目标硬件 **x86+GPU 与 ARM64 纯 CPU 双支持**,运行时按硬件自适应。

---

## 0. 背景与差距(为什么要做)

### 飞牛 fnOS 的 AI 逻辑(参考)
- **AI 相册是产品中心**。两个**本地可下载** CV 模型:`人脸识别` + `智能识别`;应用自动检测硬件并推荐档位(基础 ≥2GB 内存 / 增强 ≥8GB),支持 GPU 加速(NVIDIA GeForce 700+/Intel 核显),模型占用 ≥20GB 存储。多通道下载(种子/迅雷/网盘/应用内)。
- 能力全部内置于 Photos:**人物相册**(命名/合并/隐藏/移除误识别/手动建人/添加相似人脸,三个可调阈值)、**以文搜图**(自然语言描述搜图,如「抱着孩子」「吃面」「笑的很开心」)、**智能分类**(物体/场景类目侧栏自动生成)、**视频增强识别**(.mp4/.mov)。`执行未识别的照片` / `重置识别` 任务。全程本地、隐私安全。
- **OpenClaw**:AI 助手接入消息平台(微信/钉钉/飞书),配置 1 个 LLM 模型,扫码绑定渠道。

### HiGoOS 现状(已核实)
- `internal/aianalysis/` 是**真实**的可恢复后台分析引擎(media/file/video,级别 off/basic/standard/deep),但 vision/face/embedding/asr **全部代理给 LLM provider 或 HTTP face sidecar**,无「下载本地模型即用」流程,需 API key。
- `internal/llm/` 多 provider(openai/anthropic/gemini)、多用途。`internal/index/` 可选 pgvector 语义索引(文本)。Face 走 `FaceEmbedder` 接口(remote sidecar 或文本描述回退)+ `facetrain.go` 自训练。
- UI 分散:`AIAnalysisWindow`(独立管理窗)、`PhotoMediaWindow`(Faces 面板)、`TopSearch`(语义)、`AgentWorkbench`/`AiAssistant`(MCP)。**OpenClaw 已被移除**(commit `5248ec3`)。

### 核心差距 = 三件事
1. **没有本地推理**:一切 CV 走云/sidecar,无模型下载/硬件检测/GPU 概念。
2. **AI 未以相册为中心**:能力散落,Photos 只是薄壳。
3. **无消息机器人层**:OpenClaw 已删。

---

## 1. 总体架构决策

- **推理:混合**。新增**统一本地推理 sidecar**(Python + ONNX Runtime)负责 `人脸检测+ArcFace embedding` / `CLIP 图文 embedding(以文搜图)` / `场景物体分类(智能分类)`;**对话/Agent/分析回退仍走 `internal/llm` 可配置 provider**。
- **硬件自适应**:sidecar 用 ORT 执行提供器自动择优 `CUDA → OpenVINO(Intel) → CPU`;Go 侧检测 GPU、按内存推荐档位。
- **Go 仍是控制面**,绝不 import ML 库;sidecar 以独立容器/二进制发布(x86 CUDA 镜像 + ARM CPU 镜像各一)。
- **降级安全**:无本地模型时,分析回退到现有 LLM vision/text-embed 路径——Mac 开发与未装模型环境零回归。

---

## 2. 支柱一:本地推理 sidecar + 模型管理(后端)

### 2.1 统一推理 sidecar(新目录 `sidecar/`,Python,**不属于 Go module**)
单进程多端点,共享 ORT session 与执行提供器决策:

| 端点 | 用途 | 契约 |
|---|---|---|
| `POST /faces` | 人脸检测+512d ArcFace | **沿用现契约**:`{image,mime}` → `{faces:[{box,score,embedding}]}`(`faces.go` 的 `faceVectorThreshold=0.45` 已匹配 ArcFace 余弦域)。**Go 侧零改动即可上线人脸路径** |
| `POST /clip/image` | 图片 embedding | `{image,mime}` → `{embedding,dim,model}`(建议 Chinese-CLIP,支持中文查询) |
| `POST /clip/text` | 文本 embedding(以文搜图查询侧) | `{text}` → `{embedding,dim,model}` |
| `POST /classify` | 场景/物体分类(智能分类) | `{image,mime,topK}` → `{category, sceneTags[], labels[{label,score}]}` |
| `GET /health` | 探活+硬件 | `{ok, provider, device, models{...}, tier}` |
| `GET /models`、`POST /reload`、`POST /config{gpu}` | 模型生命周期/GPU 开关 | 供模型管理调用 |

字节流复用现成:`analyzers.go` 已有 `readCapped`(8MiB)与 `extractFrame`(关键帧),同一 buffer 喂给 clip/classify,无需新增提取代码。

### 2.2 硬件自适应(Go 侧)
- 新建 `internal/hardware/gpu_linux.go`:检测 NVIDIA(`/dev/nvidia0`/`nvidia-smi`)、Intel 核显(`/dev/dri/renderD*` vendor `0x8086`);`Inventory` 加 `GPU []GPUInfo`,在 `linux_adapter.go` 填充。
- 新建 `internal/aimodels/tier.go`:`RecommendTier(inv)` 按内存映射 `basic(≥2GB)`/`enhanced(≥8GB)`,GPU 在场则提升档位并默认开启 GPU。`runtime.GOARCH` 区分 ARM64/x86。

### 2.3 模型管理域(新包 `internal/aimodels/`,JSON 状态,无 DB)
镜像 `media.Service`/`downloads.Service` 模式:
- `types.go`:`ModelDescriptor{ID,Kind(face|clip|scene),Tier,Files,StorageBytes,MinRAMBytes,Source,SHA256,Active,Installed}`、`InstallStatus`、`ModelsStatus{Tier,GPUEnabled,Provider,Device,Models,FreeStorage,TotalRAM,SidecarReady}`。
- `catalog.go`:`//go:embed models_catalog.json` 内置模型目录(含存储/内存要求与多通道下载 URL)。
- `service.go`:`List/Status/Install(→taskID)/Activate/Uninstall/SetGPU/AttachTaskRunner`,持久化 `$stateDir/aimodels.json`(`state.SaveJSON` 原子写)。
- `install.go`:`aimodels.install` 任务——流式下载 `ModelFile.URL` → `$stateDir/models/<id>/`,SHA256 校验,原子 rename;种子/镜像可选 shell 到 aria2(同 `downloads/service.go`)。
- 模型目录 `$stateDir/models/<id>/` + `active.json`(sidecar 读取的活动指针)。新增 `HIGO_MODELS_DIR`。

### 2.4 REST / apiclient / MCP(沿用信封+confirm 约定)
新文件 `internal/httpapi/aimodels_handlers.go`:
```
GET    /api/v1/ai-models/status            -> ModelsStatus
GET    /api/v1/ai-models                   -> 目录∪已装
POST   /api/v1/ai-models/{id}/install      -> {taskId}   (mutating)
POST   /api/v1/ai-models/{id}/activate     -> ModelsStatus(mutating)
DELETE /api/v1/ai-models/{id}              -> ModelsStatus(destructive,confirm)
POST   /api/v1/ai-models/gpu               -> ModelsStatus(mutating)
GET    /api/v1/ai-models/install/stream    -> SSE 安装进度(仿 aiAnalysisProgressStream)
```
+ `internal/apiclient/aimodels.go`、`internal/mcp/tools_aimodels.go`(`registerAIModels` 加入 `BuildServer`)。在 `router.go` 构造服务、挂载路由、加 `API` 字段。

### 2.5 以文搜图:原生图片向量入 pgvector
- 迁移 `internal/db/migrations/0003_ai_image_embeddings.sql`:`ai_image_embeddings(source_uri UNIQUE, domain, embedding vector, model, indexed_at)`(dim 柔性,同 `0002`)。
- `index.go` 加 `IndexImageEmbedding(...)`(按 `source_uri` upsert,复用 `encodeVector`)。
- `search.go` 加 `ImageSearch(...)`:经 sidecar `/clip/text` 得查询向量 → KNN `ai_image_embeddings` → 解析回 media。
- 新增 `internal/aiembed/`(或 sidecar client):`TextEmbed`/`ImageEmbed`,注入 `search.New` 与 `Engine.Deps`。
- 端点 `GET /api/v1/ai/search/images?q=` + apiclient + MCP `higo.ai.search.images`。
- **关系**:CLIP 图片向量成为照片搜索主信号;原 caption→文本 embed **保留**(关键词/RAG/降级回退)——增强而非替换。

### 2.6 结果写回 media 域
- `AnalyzerResult`(`aianalysis/types.go`)加 `Category string` + `SceneTags []string`。
- `media.MediaItem` 加 `Category` + `SceneTags []string`(JSON 态,无 DB 迁移)。
- `media.Service.ApplyAnalysis` 签名扩展 `+category,+sceneTags`;同步改 `aianalysis/domain.go` 调用点与 `video.Service.ApplyAnalysis`。
- 人物相册:链路已通(`faces.assignVectors`→`People`→`ApplyAnalysis`→`peopleFromItems`),装上真实人脸模型后**只是数据质量提升**。
- 智能分类:`media/types.go` 加 `DimensionCategories = "categories"`,`Items` 按 `Category` 分面。
- 视频:`analyzeVideo` 关键帧跑 `/faces`+`/clip/image`+`/classify` 写回。

### 2.7 迁移与回退
- `remoteFaceEmbedder.Available()` 从 `url!=""` 升级为缓存 `/health` 探活——能力上报与回退判定变真实。
- 文本描述人脸回退(`faceThreshold=0.86`)降级为**已弃用**路径(仅无模型时触发,加 warn 日志)。
- 新增 config:`HIGO_SIDECAR_URL`、`HIGO_MODELS_DIR`、`HIGO_SIDECAR_GPU(on|off|auto)`;保留 `HIGO_FACE_EMBEDDER_URL`/`HIGO_FACE_TRAINER_URL` 向后兼容(设了 SIDECAR_URL 则派生 `+/faces`)。
- `EngineStatus` 加 `HasFaceModel/HasCLIP/HasScene/SidecarProvider`,供 UI 显示「本地 vs API」。

### 2.8 风险
- **embedding 维度一致性**:`ai_image_embeddings` 混用不同 CLIP 模型会污染 KNN——按 `model` 列 gate,活动 CLIP 变更时用现有 `Reanalyze` 全量重嵌。
- **JSON 单写者**:新状态只由 `higo-api` 写,worker 不写。
- **sidecar 打包是真正长杆**(双镜像 + ~20GB 模型 + SHA 校验),Go 胶水反而轻。

---

## 3. 支柱二:AI 相册中心化(前端 + media 域)

### 3.1 把 Photos 改造成 fnOS 式 AI 相册
- `PhotoFilterSidebar.vue` 改为两级导航,顺序对齐 fnOS:`时间线 / 人物 / 智能分类 / 地点 / 设备 / 相册`(`DimensionKey` 加 `'categories'`)。
- 人物维度渲染**人物相册网格**(缩略图+姓名+数量),发 `select-person`,主区改用专用面板。

### 3.2 新建/改造组件(`web-pc/src/components/windows/photo/`)
| 操作 | 文件 | 用途 |
|---|---|---|
| 改造 | `PhotoMediaWindow.vue` | 加 categories 维度、AI 搜索态、onboarding gate、人物面板路由;次要操作收进「更多」 |
| 编辑 | `PhotoFilterSidebar.vue` | 加 categories;发 `select-person` |
| 新建 | `PhotoPeoplePanel.vue` | 人物相册网格 + 命名/合并/隐藏/移除/添加相似(取代现 `.photo-faces` 块) |
| 新建 | `PhotoPersonDetail.vue` | 单人相册:照片网格、移出此人、添加相似人脸、隐藏 |
| 新建 | `PhotoCategoriesPanel.vue` | 智能分类网格,点击筛选 |
| 新建 | `PhotoSearchBar.vue` | 以文搜图输入,替换现客户端子串过滤 |
| 新建 | `PhotoOnboarding.vue` | 启用/首启向导 |
| 编辑 | `PhotoMediaGrid.vue` | 托管搜索栏,渲染照片+视频混合结果 |

### 3.3 AIAnalysisWindow 去向:**保留但降级为「高级」**
- **移入 Photos**:media 进度、`执行未识别`/`重置识别`、能力/模型状态、人脸库(由 `PhotoPeoplePanel` 全面取代,删 `FacesPanel.vue`)。
- **留在高级窗**:file/video 跨域分析记录表、批量重分析、暂停/恢复、级别控制。
- `data/higoos.ts` 中 `ai-analysis` 降低 dock 权重(`utility: true`/收进「更多」),副标题改「高级:文件/视频分析记录」。

### 3.4 启用/引导流程(`PhotoOnboarding.vue`)
1. 启用 AI 相册 CTA → 2. **硬件检测**(CPU/GPU/RAM + 推荐档位,新端点 `GET /api/v1/ai-analysis/onboarding`)→ 3. 选择/下载模型档位(`POST .../onboarding/download{tier}` → taskId,走 tasks 进度)→ 4. 进度→就绪,`setLevel('standard'|'deep')`。
- `执行未识别`:`reanalyze{scope:'domain',domain:'media',onlyUnrecognized:true}`(扩展 payload)。
- `重置识别`:新 `POST /api/v1/ai-analysis/reset{domain:'media'}` 清记录/人脸/分类。

### 3.5 智能分类 / 以文搜图 UX
- 分类:`GET /api/v1/media/categories` → `MediaCategory[]`(后端按 `sceneTags` 聚合,达阈值才出现);点击 → `getItems({dimension:'categories',facet:key})`。
- 搜图:`PhotoSearchBar` 置顶,占位「描述你想找的画面,如『抱着孩子』『吃面』」;`POST /api/v1/media/search{query,limit}` → `MediaItem[]`(视频带角标);debounce 250ms;`TopSearch` 加「在 AI 相册中搜索『{q}』」跨链。

### 3.6 设置整合:区分「本地相册模型」与「对话用 LLM」
- `SettingsAiPanel.vue`(现 3 开关 stub)重做为 **AI 相册/本地模型**面板:档位、**三可调阈值**(人脸置信度/形成人物最少照片数/人物相似度差值——把 `faces.go` 硬编码常量升级为可配置 `SettingsState.media`)、视频增强识别开关、模型下载/状态/重置入口。
- `SettingsModelsPanel.vue` 保持 **对话用 LLM 提供商**(provider CRUD/任务路由),仅澄清标题。
- `SystemSettingsWindow.vue` 导航 `ai` 改名「AI 相册/本地模型」,`models`「对话模型/提供商」。

### 3.7 类型/端点/store
- `types.ts`:`MediaItem` 加 `sceneTags/categories/personIds/previewUrl/width/height/durationSeconds`;新 `Person{id,name,cluster,count,coverUrl,hidden,confidence}`、`MediaCategory`、`MediaSearchResult`、`PhotoOnboardingStatus`;`SettingsState.media` 块。
- 新 media 端点:`/media/categories`、`/media/search`、`/media/people`(+`/{id}/rename|hide|remove-photos|add-similar`、`POST /media/people` 手动建人、`merge` 扩展 `{sourcePersonIds,targetPersonId}`)。
- `media/types.go` 的 `Person` 加 `Hidden/CoverURL/Confidence`;`media/service.go` 已有 `MergePeople`/`People()` 可扩展。
- 新建 store `stores/photoLibrary.ts`(items/albums/categories/people/searchResults/onboarding),`PhotoMediaWindow` 内联逻辑迁入,瘦身 SFC;`stores/aiAnalysis.ts` 加 onboarding/reset。

### 3.8 风险
- 阈值现为编译期常量——改运行时需 clusterer 重读 + 提供「重新聚类」动作。
- `media.search` 依赖 embedding 索引,`indexEnabled=false` 时降级关键词(仿 `TopSearch`)。
- `MediaItem.ID` Go `int` vs TS `number|string`——人物/分类 key 用 string 安全。

---

## 4. 支柱三:OpenClaw 消息机器人层(后端 + 前端)

> 老 OpenClaw 仅前端配置面板,已删,**无遗留后端**——这是干净新建,且**建在现有真实 assistant/agent 之上**。

### 4.1 新包 `internal/messaging`
- `types.go`:`Channel{ID,Name,Platform(wechat|dingtalk|feishu),Status,Credentials,LLMProvider,Policy,InboundPath,WebhookURL,LastError,LastInbound}`;`Credentials`(各平台 union:AppID/Secret/Token/EncodingAESKey/Ding webhook+secret…);`ChannelPolicy{AllowWrites(默认false),ToolAllowlist,MaxRiskInline,AllowedSenders}`;`ChannelView`(掩码,仿 `llm.ProviderView`)。
- `channel.go`:`Adapter` 接口——`Platform()` / `VerifyInbound(签名+握手)` / `ParseInbound(→InboundMessage)` / `Reply(全文推送)` / `ProbeBind(→BindInfo: QR 或 webhook URL+token)`。
- 适配器(全部 webhook 驱动,无轮询):

| 文件 | 平台 | 入站 | 出站 | 公网/HTTPS |
|---|---|---|---|---|
| `adapter_wechat.go` | 微信公众号 | GET echostr 握手(SHA1)+POST XML,可选 AES | 被动回复(<5s)或客服异步推送 | 必须公网 HTTPS,仅 80/443;扫码绑定 |
| `adapter_dingtalk.go` | 钉钉机器人 | HMAC-SHA256 签名 header | sessionWebhook/外发 webhook 异步推送 | 公网 HTTPS 粘贴到钉钉控制台 |
| `adapter_feishu.go` | 飞书机器人 | 事件订阅 + `url_verification` 挑战 + AES | `im/v1/messages` 异步推送(需 tenant_access_token) | 公网 HTTPS 粘贴到飞书开发者后台 |

- `service.go`:镜像 `assistant.Service`/`llm.Store`,CRUD(secret 保留式更新)、`Verify/Enable/Disable/BindInfo/Test/HandleInbound`,持久化 `$stateDir/messaging.json`。
- 通过窄端口避免循环依赖:`AssistantPort{EnsureThread,AddMessage,ConfirmAction,CancelAction}`,由 `*assistant.Service` 满足。

### 4.2 入站→assistant 线程(全文回复,无 SSE)
- **线程身份**:每 (channel,sender) 一线程,key=`msg:<channelID>:<senderID>`。`assistant.Service` 加 `EnsureThread(key,title)` + `Thread.ExternalKey` + `byExternalKey` 索引——保留每发送者会话记忆。
- **管线** `HandleInbound`:验签/握手 → 解析(非文本/重复 `MsgID` 去重)→ 发送者白名单 → `EnsureThread` → `AddMessage(emit=nil)` 得**一次性全文回复** → 若产生 write Action 则替换为确认码提示 → `Reply` 推送(长文 `chunkText` 分片)→ 写 audit。
- **关键**:webhook 立即 ack 200,模型调用放 goroutine/tasks(避免平台超时重试)。

### 4.3 绑定流程
`create(draft)` → 配置凭据(PUT)→ `bind`(展示 QR 或 webhook URL+token)→ 运营者粘贴/扫码 → `verify`(握手/探测,状态→connected)→ `enable`。fnOS 的「重启服务」= 此处 enable(处理器常驻,无需重启)。LLM 选择 = `Channel.LLMProvider`(默认 `llm.Store.Default()`)。

### 4.4 治理(消息=远程无人值守,最高风险,纵深防御)
1. **默认只读**:`AllowWrites=false` → 调用前把工具目录裁为只读(`MessageRequest` 加 `ReadOnlyOnly`,`generateReply` 丢弃 `t.Write`)。
2. **工具白名单**:`AllowWrites=true` 时仅暴露 `ToolAllowlist`。
3. **确认码流**:产生 pending Action 时生成短数字码(TTL~5min,绑发送者+线程),回复「此操作有风险:{impact}。回复『确认 1234』执行,『取消』放弃」;下条匹配 `确认 <code>` → `ConfirmAction`,`取消` → `CancelAction`。
4. **发送者白名单** + **每 confirm/execute 写 audit**(Domain=messaging,actor=发送者+平台,SourceIP=webhook)。

### 4.5 REST / apiclient / MCP / 前端
- 管理(session+admin 守卫):`GET/POST /api/v1/messaging/channels`、`GET/PUT/DELETE .../{id}`、`.../verify`、`.../enable|disable`、`.../bind`、`.../test`、`GET /api/v1/messaging/audit`。
- 入站(免 session,**签名验证**):`ANY /api/v1/messaging/webhook/{platform}/{inboundPath}`。
- `middleware.go`:webhook 前缀加入 `isAuthWhitelisted` 并跳过 `csrfGuard`;管理路由保持 `roleGate` admin。入站用 `io.ReadAll` 取原始 body(验签需原文)。
- `internal/apiclient/messaging.go` + MCP `higo_messaging_channels_list(readOnly)`/`higo_messaging_channel_set_enabled(mutating)`(适配器本身不暴露为 agent 工具,避免环)。
- 前端:新窗 `MessagingBotWindow.vue`(渠道列表+状态 chip+创建向导:选平台→填凭据→展示 QR/webhook→verify→enable→选 LLM→「试聊」),`data/higoos.ts` 注册(复用 `Bot` 图标),`data/nasFeatures.ts` 重新加 `messaging` key(`5248ec3` 的逆操作),`api/client.ts` + 可选 `stores/messaging.ts`。

### 4.6 config / 安全 / 顺序
- `config.go`:`HIGO_MESSAGING_ENABLED`(默认 true)、`HIGO_MESSAGING_PUBLIC_ORIGIN`(算 webhook URL/QR,回退 `PublicOrigin`;微信需 80/443+HTTPS)。
- 安全:`messaging.json` secret 在所有 View 掩码;验签 `crypto/subtle` 常量时间;`MsgID` 去重缓存;每渠道/发送者限流;仅 webhook 路径免认证,建议经 `internal/remote` 隧道/反代 TLS 暴露。

---

## 5. 支柱四:保留 MCP Agent/助手

`AgentWorkbenchWindow` / `AiAssistantWindow` / `internal/assistant` / `internal/agent` **原样保留**,定位为「系统运维 AI」,与相册 AI 并存。消息机器人层即建在其 `AddMessageStream`/`ConfirmAction` 之上(见 §4)。仅新增 `EnsureThread`/`ExternalKey`/`ReadOnlyOnly` 三处扩展。

---

## 6. 总体分期与构建顺序

| 阶段 | 内容 | 价值/风险 |
|---|---|---|
| **P1 人脸路径上线** | sidecar `/faces`+`/health`(ORT,SCRFD+ArcFace,提供器自选);`HIGO_FACE_EMBEDDER_URL` 指向它 | **Go 零改动**端到端验证;最小最高价值 |
| **P2 硬件+档位** | `hardware/gpu_linux.go`+`GPUInfo`;`aimodels/tier.go`;`/health` 暴露 provider/tier | — |
| **P3 模型管理** | `internal/aimodels/*` + handlers + apiclient + MCP;sidecar `/models|/reload|/config`;前端 onboarding | Go 主体工作量 |
| **P4 以文搜图** | sidecar `/clip/*`;`0003` 迁移;`IndexImageEmbedding`;`ImageSearch`;`/media/search` + `PhotoSearchBar` | DB/搜索主工作量 |
| **P5 智能分类+写回** | sidecar `/classify`;`Category/SceneTags`;`categories` 维度 + `PhotoCategoriesPanel` | — |
| **P6 相册前端整合** | `PhotoPeoplePanel`/`PhotoPersonDetail`,删 `FacesPanel`,AIAnalysisWindow 降级,设置整合 | 前端主体 |
| **P7 消息机器人** | `internal/messaging` 骨架→assistant 集成(`EnsureThread`/`ReadOnlyOnly`)→飞书→钉钉→微信→治理→前端 | 独立支柱,可并行 |
| **P8 打磨** | `Available()` 健康探活、弃用文本人脸回退、限流/去重/掩码审查、双 sidecar 镜像打包 | — |

> 建议先做 P1–P6(相册主线,飞牛核心卖点),P7(消息机器人)可与之并行;P8 收尾。

---

## 7. 验证方式(端到端)

- **后端编译/静态检查**:`cd server-go && go build ./... && go vet ./...`(交叉编译目标 `CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/higo-api`)。
- **单测**:各域 `go test ./internal/aimodels ./internal/messaging ./internal/aianalysis ...`;镜像 `llm/store_test.go` 写 `aimodels`/`messaging` 状态测试。
- **sidecar**:本地起 sidecar,`curl /health` 验证 provider 自选;`/faces`、`/clip/text`、`/classify` 各打一张样片。
- **前端**:`cd web-pc && npm run build`(`vue-tsc` 类型门禁)+ `npm run test:interactions`;`npm run dev` 走 onboarding→下载→人物相册→以文搜图→智能分类。
- **部署联调**:`SSH_PASS=123qwe ./deploy/deploy.sh` 到 ARM64 VM(10.211.55.3),验证 ARM CPU 路径(无 GPU)与模型下载/激活。
- **消息机器人**:飞书事件订阅握手(`url_verification`)、试聊只读回复、写操作确认码流、audit 落库。

---

## 8. 关键改动文件索引

**新建包/目录**:`sidecar/`(Python)、`internal/aimodels/`、`internal/aiembed/`、`internal/messaging/`、`internal/hardware/gpu_linux.go`、`internal/httpapi/{aimodels_handlers,messaging_handlers,messaging_inbound}.go`、`internal/apiclient/{aimodels,messaging}.go`、`internal/mcp/{tools_aimodels}.go`、`internal/db/migrations/0003_ai_image_embeddings.sql`、`web-pc/src/components/windows/photo/*`、`web-pc/src/components/windows/MessagingBotWindow.vue`、`web-pc/src/stores/{photoLibrary,messaging}.ts`。

**主要修改**:`internal/aianalysis/{analyzers,faceembed,domain,types}.go`、`internal/index/index.go`、`internal/search/search.go`、`internal/media/{service,types}.go`、`internal/assistant/{service,types}.go`、`internal/httpapi/{router,middleware,aianalysis_handlers}.go`、`internal/platform/config.go`、`web-pc/src/components/windows/{PhotoMediaWindow,AIAnalysisWindow}.vue`、`web-pc/src/components/windows/settings/{SettingsAiPanel,SettingsModelsPanel}.vue`、`web-pc/src/api/{client,types}.ts`、`web-pc/src/data/{higoos,nasFeatures}.ts`、`web-pc/src/stores/aiAnalysis.ts`。
