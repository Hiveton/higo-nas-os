# HiGoOS 账号绑定流程

把一台 NAS 绑定到一个 HiGoOS **云账号**,使 App 能在任意网络下安全访问它。涉及三方:**App**、**server-cloud**(云)、**server-go**(NAS,内含 `internal/cloud` 连接器)。

## 角色与令牌一览

| 名称 | 谁持有 | 用途 |
|---|---|---|
| 云账号 access JWT | App | 调云端 `/v1/*`(HS256,~15min) |
| 云账号 refresh | App | 换 access(旋转、可撤销) |
| `deviceSecret` | NAS | 设备向云鉴权(注册时获得,持久化 `cloud.json`) |
| 云公钥 | NAS | 验证 App 出示的设备访问令牌(注册时获得) |
| 设备访问令牌 | App | 访问某台 NAS(Ed25519,`aud=device:{id}`,~10min) |

## 前置:NAS 设备注册(一次性)

```
NAS(server-go, HIGO_CLOUD_ENABLED=true)
  └─POST /v1/devices/register {deviceId, model, version}
       └─► Cloud 创建 device 记录
            └─◄ {deviceSecret, serial(HG-XXXX-XXXX), cloudPublicKey, relayPath}
  NAS 持久化到 HIGO_STATE_DIR/cloud.json,并用 deviceSecret 拨 wss /v1/agent/connect 维持隧道
```

`deviceId` 复用 `server-go/internal/identity` 的稳定 `hg-xxxx`。

## 绑定流程 A — NAS 出示绑定码 / 二维码(主推)

```
NAS(Web 桌面/屏显)
  └─POST /v1/bindings/pairing/issue {kind:"code"}  (X-Device-Id + X-Device-Secret)
       └─◄ {secret(6位码/二维码), expiresAt(~5min)}        # NAS 显示

App(已登录云账号)
  └─POST /v1/bindings/pairing/claim {code}  (Bearer access)
       └─► Cloud 校验码未过期/单次 → 创建 binding(account↔device)
            → Provisioner 经隧道让 NAS 创建本地用户(POST /api/v1/cloud/provision)
            → 签发设备访问令牌
       └─◄ {binding, device, deviceToken, tokenExpires}     # 绑定完成
```

首位绑定者角色 `admin`,后续 `user`。

## 绑定流程 B — 局域网发现 + 云账号确认

```
App ── UDP 发现 ──► NAS  (server-go GET /api/v1/system/identity,LAN 白名单)
   拿到 {deviceId, LAN 地址}
App ─POST /v1/bindings/lan-confirm {deviceId} (Bearer access)─► Cloud
   Cloud 确认设备已注册 → 同 A 的 binding + provision + 签发
```

(NAS 侧可要求一次本地确认动作,防同网攻击。)

## 绑定流程 C — 云端预注册 + 序列号 + PIN

```
设备出厂即注册,serial 印在机身/包装
NAS 屏显/Web 出示一次性 PIN(POST /v1/bindings/pairing/issue {kind:"pin"})
App ─POST /v1/bindings/serial {serial, pin} (Bearer access)─► Cloud
   Cloud 按 serial 找设备 + 校验 PIN(且 PIN 属于该设备)→ 同 A 的 binding + provision + 签发
```

## 访问已绑定 NAS(绑定之后)

```
App ─GET /v1/bindings─► 列出设备 + 在线态
App ─POST /v1/devices/{id}/access-ticket─► 取短期设备访问令牌(令牌临期时刷新)

读取数据(直连优先):
  1) LAN: GET http://<nas-lan>:8080/api/v1/...   Authorization: Bearer <deviceToken>
  2) 回退中继: GET <cloud>/d/{deviceId}/api/v1/...
        Authorization: Bearer <cloud access>   X-Device-Token: <deviceToken>
     云端校验 binding → 把 X-Device-Token 改写进上游 Authorization → 经隧道转发到 NAS
```

NAS 的 `sessionGuard`(`server-go/internal/httpapi/middleware.go`)用云公钥验证设备访问令牌,解析出**映射的本地用户**(角色受限),而非旧的"任意 Bearer = admin"。

## 解绑

```
App ─DELETE /v1/bindings/{deviceId}─► Cloud
   删除 binding → Provisioner 经隧道让 NAS 移除本地用户(/api/v1/cloud/unprovision)
   该设备的设备访问令牌随 aud/过期自然失效
```

## 安全要点

- `/api/v1/cloud/provision`、`/unprovision` **只接受经中继隧道到达的请求**(`cloud.IsRelayOrigin`),LAN 直连调用被拒——防止同网攻击者凭空创建本地用户。
- 绑定码 / PIN:短 TTL、单次消费。
- refresh 令牌旋转:用一次即撤销,重放被拒(见 `router_test.go`)。
- 设备访问令牌 `aud=device:{id}`,不能跨设备重放。
