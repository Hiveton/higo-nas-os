# server-cloud

HiGoOS 云控制面 + 中继(Go,`module higoos/server-cloud`)。服务移动 App:云账号、设备注册/绑定、中继隧道、推送。NAS 数据仍归各台 `server-go`,云端做鉴权、编排与转发。

完整说明见 [`docs/server-cloud.md`](../docs/server-cloud.md);账号绑定流程见 [`docs/account-binding.md`](../docs/account-binding.md)。

## 快速开始

```bash
HIGO_CLOUD_STORE=memory HIGO_CLOUD_ENV=dev go run ./cmd/higo-cloud   # :8090
go test ./...                                                        # 端到端 + 中继往返
CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build ./cmd/higo-cloud      # 部署
```

## 冒烟(dev,memory store)

```bash
BASE=http://127.0.0.1:8090
# 1) 短信登录(dev 回显 debugCode)
CODE=$(curl -s $BASE/v1/auth/sms/start -d '{"phone":"13800000000"}' | jq -r .data.debugCode)
ACCESS=$(curl -s $BASE/v1/auth/sms/verify -d "{\"phone\":\"13800000000\",\"code\":\"$CODE\"}" | jq -r .data.accessToken)
# 2) NAS 注册
SECRET=$(curl -s $BASE/v1/devices/register -d '{"deviceId":"hg-demo","model":"HiGoOS"}' | jq -r .data.deviceSecret)
# 3) NAS 出示绑定码
PCODE=$(curl -s $BASE/v1/bindings/pairing/issue -H "X-Device-Id: hg-demo" -H "X-Device-Secret: $SECRET" -d '{"kind":"code"}' | jq -r .data.secret)
# 4) App 认领 → 拿设备访问令牌
curl -s $BASE/v1/bindings/pairing/claim -H "Authorization: Bearer $ACCESS" -d "{\"code\":\"$PCODE\"}" | jq .data
# 5) 列出已绑定设备
curl -s $BASE/v1/bindings -H "Authorization: Bearer $ACCESS" | jq .data
```

## 部署

二进制装到 `/opt/higoos/bin/higo-cloud`,环境文件 `/etc/higoos/cloud.env`(见 `.env.example`),systemd 单元 `deploy/higo-cloud.service`。
