# server-cloud — HiGoOS 云控制面 + 中继

`server-cloud/` 是 HiGoOS 的云端后端,服务移动 App,定位为**账号 + 中继控制面**:它只负责云账号、设备注册/绑定、中继隧道与推送;NAS 的实际数据(文件/照片/存储)仍归每台 NAS 的 `server-go`,云端做鉴权、编排与转发。

技术栈与 `server-go` 同构:Go,响应信封 `{ok,data,error:{code,message},requestId}`,`platform.WriteJSON/WriteError`,`allowMethod` 限方法,中间件链 + nil-fallback 注入。纯 Go 无 CGO,交叉编译 `linux/arm64` 干净。

## 架构

```
                ┌──────────── server-cloud (公网) ────────────┐
  iOS App ──────┤ 控制面 REST  /v1/auth /v1/account /v1/devices │
   (cloud JWT)  │             /v1/bindings /v1/push            │
       │        │ 中继数据面  /d/{deviceId}/api/v1/... ←→ 隧道  │
       │        └──────▲──────────────────────────────▲────────┘
       │ 直连/中继       │ outbound WSS (deviceSecret)   │ APNs
       └──► server-go (NAS) ── internal/cloud connector ┘
```

- **App ↔ Cloud**:登录云账号、列出已绑定 NAS、换取设备访问令牌、注册推送。
- **App ↔ NAS**:拿到设备访问令牌后,**LAN 直连优先** server-go `/api/v1/*`,不可达回退**云中继** `/d/{deviceId}/api/v1/*`。两条路径同一个令牌。
- **NAS ↔ Cloud**:首启 `POST /v1/devices/register` 拿 `deviceSecret` + `serial` + 云公钥;之后 `internal/cloud` 连接器维持一条 outbound WSS 隧道,云端经隧道多路复用 App 请求。

## 包结构

| 包 | 职责 |
|---|---|
| `internal/platform` | 信封、requestId、principal、config、logger(复刻 server-go) |
| `internal/store` | 持久化边界:`Store` 接口 + `Memory`(dev 默认)+ `Postgres`(生产,连接即建表)。两者共用一致性测试 |
| `internal/token` | JWT:HS256(云访问令牌)、Ed25519(设备访问令牌),纯 stdlib |
| `internal/auth` | 签名材料 + TTL 策略;签发/校验访问令牌、设备令牌;不透明 refresh 哈希 |
| `internal/verification` | 短信/邮件验证码(内存 TTL + 可插拔 Sender,dev 打日志) |
| `internal/oauth` | Apple / WeChat provider(dev stub 本地解码) |
| `internal/account` | 云账号:四种登录 → 账号 + 令牌对;refresh 旋转 |
| `internal/devices` | 设备注册表:注册、secret 校验、心跳、serial |
| `internal/bindings` | 账号↔设备绑定 + 三种流程 + 设备令牌签发;`Provisioner` 接缝 |
| `internal/relay` | 数据面:agent WSS 注册表 + 请求转发;`Provisioner` 经隧道 provision |
| `internal/push` | APNs token 注册(发送为 M5 待办) |
| `internal/httpapi` | router + 中间件 + handlers |

## 令牌模型

- **云访问令牌**(access):HS256,~15 分钟,`aud=cloud`,仅云端验签。
- **Refresh**:不透明随机串,长期、可撤销、**旋转**(用一次即撤销旧的);库里只存 SHA-256。
- **设备访问令牌**:Ed25519,~10 分钟,`aud=device:{deviceId}`,claims 含云账号 id + 映射的 NAS 本地用户 id + 角色。**云用私钥签,NAS 用注册时拿到的云公钥验**(见 `server-go/internal/cloud/token.go`),取代旧的"任意 Bearer = admin"。

## REST API

开放(无需云令牌):

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/healthz`、`/v1/health` | 健康 + 构建信息 |
| POST | `/v1/auth/sms/start` | 下发短信验证码(dev 回显 `debugCode`) |
| POST | `/v1/auth/sms/verify` | 验码登录/自动注册 → session |
| POST | `/v1/auth/email/register` | 邮箱+密码注册 → session |
| POST | `/v1/auth/email/login` | 邮箱+密码登录 → session |
| POST | `/v1/auth/apple` | Apple `identityToken` 登录 |
| POST | `/v1/auth/wechat` | 微信 `code` 登录 |
| POST | `/v1/auth/refresh` | 旋转 refresh,换新令牌对 |
| POST | `/v1/auth/logout` | 撤销 refresh |
| POST | `/v1/devices/register` | NAS 自注册 → `deviceSecret`/`serial`/`cloudPublicKey` |

设备鉴权(`X-Device-Id` + `X-Device-Secret`):

| 方法 | 路径 | 说明 |
|---|---|---|
| POST | `/v1/devices/heartbeat` | 心跳 |
| POST | `/v1/bindings/pairing/issue` | NAS 出示绑定码/PIN(`kind: code\|pin`) |
| GET  | `/v1/agent/connect` | 中继隧道(WebSocket) |

云账号鉴权(`Authorization: Bearer <access>`):

| 方法 | 路径 | 说明 |
|---|---|---|
| GET | `/v1/account/me` | 账号资料 + 已绑定登录方式 |
| POST | `/v1/bindings/pairing/claim` | 绑定流程 A:认领绑定码 |
| POST | `/v1/bindings/lan-confirm` | 绑定流程 B:确认局域网设备 |
| POST | `/v1/bindings/serial` | 绑定流程 C:序列号 + PIN |
| GET | `/v1/bindings` | 列出已绑定设备(含在线态) |
| DELETE | `/v1/bindings/{deviceId}` | 解绑 |
| POST | `/v1/devices/{deviceId}/access-ticket` | 换取设备访问令牌 |
| POST | `/v1/push/register` | 注册 APNs token |
| ANY | `/d/{deviceId}/api/v1/...` | 中继转发到 NAS(头部 `X-Device-Token` 带设备令牌) |

## 配置(`HIGO_CLOUD_*`)

`HIGO_CLOUD_ENV`、`HIGO_CLOUD_HTTP_ADDR`(默认 `:8090`)、`HIGO_CLOUD_PUBLIC_ORIGIN`、`HIGO_CLOUD_STORE`(`memory`|`postgres`)、`HIGO_CLOUD_DB_DSN`、`HIGO_CLOUD_AUTH_REQUIRED`、`HIGO_CLOUD_JWT_SECRET`、`HIGO_CLOUD_DEVICE_TOKEN_SEED`(64 hex,固定云签名公钥)、`HIGO_CLOUD_*_TTL`、`HIGO_CLOUD_VERIFICATION_DEBUG`、`HIGO_CLOUD_APPLE_CLIENT_ID`、`HIGO_CLOUD_WECHAT_*`、`HIGO_CLOUD_APNS_*`。详见 `server-cloud/.env.example`。

## 本地运行 / 测试

```bash
cd server-cloud
HIGO_CLOUD_STORE=memory HIGO_CLOUD_ENV=dev go run ./cmd/higo-cloud   # :8090
go test ./...                                                        # 含端到端 + 中继往返
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/higo-cloud      # 部署目标
```

## 生产能力(已实现)

- **持久化**:`HIGO_CLOUD_STORE=postgres` + `HIGO_CLOUD_DB_DSN`,`ConnectPostgres` 连接即幂等建表(`internal/store/schema.sql`)。Memory/Postgres 共用 `store_test.go` 一致性测试。
- **流式中继**:relay 协议支持 `response-head → response-chunk* → response-end`,SSE(`/events/stream`、AI 消息流)经隧道**边产边传**(`relay_stream_test.go` 验证);buffered `response` 仍兼容。
- **真实第三方登录**:Apple JWKS RS256 验签(aud/iss/exp 校验,`oauth_test.go` 用本地 RSA 验证);微信 `sns/oauth2/access_token` 换 unionid。`HIGO_CLOUD_ENV=dev` 仍走可离线的 stub。
- **APNs**:token-based(ES256 .p8 + HTTP/2),配置齐全即启用(`apns_test.go`);`push.Notify` 向账号全部设备发送。
- **生产 provisioner**:非 dev 自动用 `relay.NewProvisioner(hub)`,经隧道让 NAS 创建本地用户;dev 用 `StubProvisioner`。

## 已知边界(后续)

- Docker 容器 PTY 的 WebSocket 终端暂不经中继代理(SSE/分块已支持)。
- 微信仅取 `unionid/openid`,未拉 `sns/userinfo`(昵称/头像);按需补。
