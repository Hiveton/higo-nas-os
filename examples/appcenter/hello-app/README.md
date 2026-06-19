# hello-app — App Center 三方应用示例

一个最小可运行的 HiGoOS 应用包：用 `nginx` 提供静态页面，并以内嵌 iframe 窗口在桌面打开。

## 试用

1. 把本目录复制到状态目录下：

   ```sh
   cp -r examples/appcenter/hello-app "$HIGO_STATE_DIR/appcenter/apps/hello-app"
   ```

2. 让目录重新扫描：

   ```sh
   curl -X POST http://<host>:8080/api/v1/app-center/catalog/refresh
   ```

   或重启控制平面。

3. 在应用中心“发现”页找到 **Hello App** → 安装(填写“欢迎语”→ 确认影响摘要)→
   运行后点 **打开**，会以 `http://<host>:18900/` 内嵌显示。

## 结构

只有一个文件 `manifest.json` —— 它就是完整的包格式。字段说明见
[`docs/appcenter-sdk.md`](../../../docs/appcenter-sdk.md)，校验用
[`docs/appcenter-manifest.schema.json`](../../../docs/appcenter-manifest.schema.json)。

把 `image` 换成你自己的镜像、调整 `ports` / `volumes` / `env` / `config` / `webEntry`，
即可发布你自己的应用。
