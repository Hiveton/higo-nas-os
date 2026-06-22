# HiGoOS 按绿联 NAS 完整重构方案

生成日期：2026-06-22  
调研目标：`https://ug.link/higonas`  
登录账号：`hiveton`

> 本文档基于对绿联 NAS 网页端的实际登录浏览、桌面入口截图、控制面板截图、文件管理截图、应用中心截图，以及前端语言包/静态资源文案提取整理。本文只作为产品与架构重构方案，不包含任何代码修改。

## 1. 重构目标

HiGoOS 当前已经具备较完整的 NAS 能力骨架，但产品体验仍是自研 AI NAS 桌面系统。若目标是“完全按照绿联走”，重构重点不是简单补几个页面，而是将信息架构、桌面交互、控制面板分组、应用中心模型、文件管理空间、任务/通知/日志横切能力全部改成绿联 NAS 的组织方式。

最终目标：

1. 首页桌面、顶部状态栏、应用入口、窗口行为对齐绿联 NAS。
2. 控制面板按绿联的“连接与访问 / 通用设置 / 系统服务”重排。
3. 文件管理按绿联的“个人文件夹 / 共享文件夹 / 用户文件夹 / 标签 / 回收站”重排。
4. 应用中心按绿联的分类、安装状态、打开/安装/更新行为重构。
5. 任务中心、通知中心、日志中心、全局搜索、客户端设置成为系统级横切能力。
6. 远程访问、UGREENlink、账号、MFA、设备连接、安全治理统一收敛到控制面板。
7. 保留 HiGoOS 现有后端域能力，但把前端产品呈现完全绿联化。

## 2. 绿联 NAS 一级功能清单

登录后桌面确认到的一级入口如下：

| 序号 | 功能入口 | 说明 |
| --- | --- | --- |
| 1 | UGREEN AI | AI 助手/AI NAS 入口 |
| 2 | 应用中心 | 应用安装、打开、分类、已安装管理 |
| 3 | 控制面板 | 用户、文件服务、设备连接、系统设置、更新等 |
| 4 | 文件管理 | 文件空间、上传下载、分享、权限、标签等 |
| 5 | 回收站 | 文件删除后的恢复/清理 |
| 6 | 帮助中心 | 使用帮助、教程、支持 |
| 7 | 日志中心 | 系统日志、操作日志、告警日志 |
| 8 | 存储管理 | 存储池、硬盘、RAID、SMART、健康检查 |
| 9 | 任务管理器 | 上传、下载、同步、备份、扫描等任务 |
| 10 | 文本编辑 | 文本文件编辑 |
| 11 | 下载中心 | HTTP、BT、磁力、迅雷等下载 |
| 12 | 同步与备份 | 同步、备份、恢复、版本保留 |
| 13 | Docker | 容器、镜像、网络、卷、Compose |
| 14 | 影视中心 | 视频媒体库、刮削、字幕、转码、播放器 |
| 15 | 音乐 | 音乐库、专辑、歌词、播放 |
| 16 | 网盘工具 | 网盘上传/下载/同步工具 |
| 17 | 相册 | 照片、时间线、人物、地点、回忆、备份 |
| 18 | Hermes Agent | 智能代理/自动化助手入口 |

## 3. 绿联 NAS 功能树

```text
绿联 NAS
├─ 桌面系统
│  ├─ 桌面图标
│  ├─ Dock / 应用快捷方式
│  ├─ 顶部状态栏
│  │  ├─ CPU / RAM 状态
│  │  ├─ 上行 / 下行网速
│  │  ├─ 任务入口
│  │  ├─ 通知入口
│  │  ├─ 搜索入口
│  │  └─ 用户入口
│  ├─ 应用库
│  ├─ 全局搜索
│  ├─ 通知中心
│  ├─ 任务中心
│  ├─ 桌面快捷方式管理
│  ├─ 壁纸 / 背景
│  ├─ 菜单栏设置
│  ├─ 小组件
│  ├─ 客户端设置
│  │  ├─ 开机自启动
│  │  ├─ 客户端缓存清理
│  │  ├─ 客户端日志查看/上传
│  │  ├─ 客户端版本检查
│  │  └─ 客户端锁屏保护
│  └─ 电源操作
│     ├─ 休眠
│     ├─ 重启
│     ├─ 关机
│     └─ 更新并重启/关机
├─ 控制面板
│  ├─ 连接与访问
│  │  ├─ 用户管理
│  │  │  ├─ 本地用户
│  │  │  ├─ 管理员/普通用户
│  │  │  ├─ 用户组
│  │  │  ├─ 权限
│  │  │  ├─ 容量配额
│  │  │  ├─ 登录日志
│  │  │  ├─ 会话管理
│  │  │  └─ MFA / OTP 双重验证
│  │  ├─ 文件服务
│  │  │  ├─ SMB
│  │  │  ├─ FTP
│  │  │  ├─ WebDAV
│  │  │  ├─ NFS
│  │  │  ├─ DLNA
│  │  │  ├─ SSH / SFTP
│  │  │  └─ 远程 SMB / WebDAV / FTP 连接
│  │  ├─ 设备连接
│  │  │  ├─ UGREENlink ID
│  │  │  ├─ 远程访问
│  │  │  ├─ 直连/中继
│  │  │  ├─ 设备绑定
│  │  │  ├─ 设备解绑
│  │  │  ├─ 所有权转让
│  │  │  ├─ 历史登录设备
│  │  │  └─ 网络唤醒 WOL
│  │  ├─ 域 / LDAP
│  │  └─ 终端机
│  ├─ 通用设置
│  │  ├─ 硬件与电源
│  │  │  ├─ 常规
│  │  │  ├─ 存储
│  │  │  ├─ 应用
│  │  │  ├─ 服务
│  │  │  ├─ 设备分析
│  │  │  ├─ CPU / 内存 / 温度
│  │  │  ├─ 风扇 / 电源
│  │  │  ├─ 硬盘休眠
│  │  │  └─ UPS / 自动开关机
│  │  ├─ 时间和语言
│  │  │  ├─ 默认语言
│  │  │  ├─ 日期格式
│  │  │  ├─ 时间格式
│  │  │  └─ 时区 / NTP
│  │  ├─ 网络设置
│  │  │  ├─ 网卡
│  │  │  ├─ IP / DNS / 网关
│  │  │  ├─ MTU
│  │  │  ├─ HTTPS
│  │  │  ├─ 证书
│  │  │  └─ 网络诊断
│  │  ├─ 安全性
│  │  │  ├─ 账号安全
│  │  │  ├─ 防火墙
│  │  │  ├─ 端口扫描
│  │  │  ├─ 病毒扫描
│  │  │  ├─ 分享链接风险
│  │  │  └─ 审计 / 回滚
│  │  └─ 索引服务
│  │     ├─ 文件索引
│  │     ├─ 媒体索引
│  │     ├─ 搜索范围
│  │     └─ 重建索引库
│  └─ 系统服务
│     ├─ 关于本机
│     │  ├─ 设备名称
│     │  ├─ 型号
│     │  ├─ 序列号
│     │  ├─ 系统版本
│     │  ├─ 设备所有者
│     │  ├─ UGREENlink ID
│     │  ├─ 保修到期
│     │  ├─ 上次开机时间
│     │  └─ 系统运行时间
│     └─ 更新与还原
│        ├─ 系统更新
│        ├─ 手动安装包
│        ├─ 配置备份
│        ├─ 系统还原
│        ├─ 恢复出厂设置
│        └─ 设备日志上传
├─ 文件管理
│  ├─ 个人文件夹
│  ├─ 共享文件夹
│  ├─ 用户文件夹
│  ├─ 标签
│  ├─ 回收站
│  ├─ 基础操作
│  │  ├─ 新建文件夹
│  │  ├─ 上传
│  │  ├─ 下载
│  │  ├─ 复制
│  │  ├─ 移动
│  │  ├─ 剪切
│  │  ├─ 重命名
│  │  ├─ 删除
│  │  ├─ 压缩
│  │  └─ 解压缩
│  ├─ 浏览能力
│  │  ├─ 搜索
│  │  ├─ 筛选
│  │  ├─ 排序
│  │  ├─ 列表视图
│  │  ├─ 图标视图
│  │  └─ 文件详情
│  ├─ 协作与安全
│  │  ├─ 文件分享
│  │  ├─ 外链分享
│  │  ├─ 用户/组权限
│  │  ├─ 父子权限继承
│  │  ├─ 文件属性
│  │  ├─ 文件版本管理
│  │  └─ 潜在重复文件
│  └─ 预览与编辑
│     ├─ 图片预览
│     ├─ 视频播放
│     ├─ 文本编辑
│     └─ 在线文档
├─ 存储管理
│  ├─ 存储池
│  │  ├─ 创建存储池
│  │  ├─ RAID 类型
│  │  ├─ 添加硬盘
│  │  ├─ 更改 RAID 类型
│  │  ├─ 修复存储池
│  │  ├─ 替换硬盘
│  │  └─ 删除存储池
│  ├─ 存储空间
│  │  ├─ 创建空间
│  │  ├─ 容量
│  │  ├─ 配额
│  │  ├─ 文件系统
│  │  └─ 删除空间
│  ├─ 硬盘
│  │  ├─ 硬盘状态
│  │  ├─ SMART 信息
│  │  ├─ 健康检查
│  │  ├─ 读写检测
│  │  ├─ 数据擦除
│  │  └─ 外接盘风险
│  ├─ 快照
│  │  ├─ 手动快照
│  │  ├─ 快照计划
│  │  └─ 快照回滚
│  └─ 告警
│     ├─ 容量不足
│     ├─ 硬盘异常
│     └─ 阵列降级
├─ 应用中心
│  ├─ 分类
│  │  ├─ 全部应用
│  │  ├─ 系统管理
│  │  ├─ 实用工具
│  │  ├─ 娱乐
│  │  ├─ 下载
│  │  ├─ 安全
│  │  ├─ 备份
│  │  └─ 已安装
│  ├─ 应用生命周期
│  │  ├─ 安装
│  │  ├─ 打开
│  │  ├─ 停止
│  │  ├─ 更新
│  │  ├─ 卸载
│  │  └─ 权限/数据目录
│  └─ 已确认应用
│     ├─ 在线文档
│     ├─ 音乐
│     ├─ 虚拟机
│     ├─ 影视中心
│     ├─ 迅雷
│     └─ 相册
├─ 同步与备份
│  ├─ 同步任务
│  ├─ 备份任务
│  ├─ 数据还原
│  ├─ 文件版本保留
│  ├─ 默认保留版本数
│  ├─ 共享文件夹版本策略
│  ├─ 冲突处理
│  ├─ 文件过滤策略
│  ├─ 任务日志
│  └─ 错误诊断
├─ 下载中心
│  ├─ HTTP 下载
│  ├─ BT 下载
│  ├─ 磁力链接
│  ├─ 迅雷
│  ├─ 下载队列
│  ├─ 暂停/继续/重试
│  ├─ 限速策略
│  ├─ 自动归档
│  └─ 失败记录
├─ Docker
│  ├─ 容器
│  ├─ 镜像
│  ├─ 网络
│  ├─ 卷
│  ├─ Compose
│  ├─ 端口映射
│  ├─ 资源限制
│  ├─ 日志
│  └─ 迁移风险提示
├─ 媒体中心
│  ├─ 影视中心
│  │  ├─ 媒体库
│  │  ├─ 扫描
│  │  ├─ 刮削
│  │  ├─ 匹配修正
│  │  ├─ 字幕
│  │  ├─ 在线字幕下载
│  │  ├─ 转码
│  │  ├─ 播放器
│  │  ├─ 音轨
│  │  ├─ 画中画
│  │  ├─ 倍速
│  │  ├─ 兼容模式
│  │  ├─ 片头片尾识别
│  │  ├─ 直播源
│  │  ├─ 节目单
│  │  └─ DVR 录制
│  ├─ 音乐
│  │  ├─ 音乐库
│  │  ├─ 扫描
│  │  ├─ 歌曲
│  │  ├─ 专辑
│  │  ├─ 艺术家
│  │  ├─ 歌词
│  │  └─ 播放队列
│  └─ 相册
│     ├─ 时间线
│     ├─ 相册
│     ├─ 共享相册
│     ├─ 人物
│     ├─ 地点
│     ├─ 回忆
│     ├─ 手机备份
│     ├─ Google Photos 导入
│     ├─ 视频兼容
│     └─ AI 相册
├─ 安全与日志
│  ├─ 日志中心
│  ├─ 通知设置
│  ├─ 操作审计
│  ├─ 登录日志
│  ├─ 权限升级通知
│  ├─ 病毒扫描
│  ├─ 端口扫描
│  ├─ 防火墙
│  ├─ 风险动作
│  ├─ 审计回滚
│  └─ 安全披露/漏洞报告
└─ AI
   ├─ UGREEN AI
   ├─ 私人 AI 助手
   ├─ 启用 AI NAS
   ├─ 下载 AI 模型
   ├─ AI 控制台
   ├─ Uliya
   └─ Hermes Agent
```

## 4. 当前 HiGoOS 项目能力对照

当前项目是 monorepo，核心如下：

| 区域 | 路径 | 现状 |
| --- | --- | --- |
| 前端桌面 | `web-pc/src/App.vue` | Vue 3 + Vite 桌面壳，无传统路由 |
| 桌面应用定义 | `web-pc/src/data/higoos.ts` | 已有 24 个应用入口 |
| 功能描述矩阵 | `web-pc/src/data/nasFeatures.ts` | 已覆盖存储、文件、媒体、协议、VM、同步、备份、安全、下载等 |
| 前端窗口组件 | `web-pc/src/components/windows/*` | 已有多数 NAS 应用窗口 |
| 前端状态 | `web-pc/src/stores/*` | 已有 accounts、auth、hardware、sync、tasks、remote 等 store |
| 后端 API | `server-go/internal/httpapi/router.go` | 已注册大量 `/api/v1/*` REST 端点 |
| 后端业务域 | `server-go/internal/*` | 已有 files、storage、docker、downloads、media、music、video、remote、accounts、安全等域 |
| 云端控制 | `server-cloud/*` | 已有账号、绑定、relay、push、oauth 等 |

### 4.1 当前前端应用入口

当前 `web-pc/src/data/higoos.ts` 已有应用：

| 当前应用 | 对应绿联能力 | 处理建议 |
| --- | --- | --- |
| 文件管理 | 文件管理 | 保留能力，UI/信息架构改绿联 |
| 存储管理 | 存储管理 | 保留，入口和页面布局改绿联 |
| AI 文件管家 | UGREEN AI / Hermes Agent | 降级为 AI 子能力，不抢主入口 |
| AI 分析中心 | UGREEN AI / 相册 AI | 作为 AI/相册增强 |
| Agent 工作台 | Hermes Agent | 改名/包装为 Hermes Agent |
| AI 助手 | UGREEN AI | 改名/入口对齐 |
| 备份同步 | 同步与备份 | 改名为“同步与备份” |
| 相册媒体 | 相册 | 拆为“相册”，媒体能力内聚 |
| 音乐中心 | 音乐 | 改名为“音乐” |
| 影视中心 | 影视中心 | 保留 |
| 下载中心 | 下载中心 | 保留 |
| 应用中心 | 应用中心 | 重做分类和卡片 |
| Docker | Docker | 保留 |
| 安全中心 | 控制面板-安全性 / 安全应用 | 收敛到控制面板，同时可保留应用入口 |
| 设备监控 | 控制面板-硬件与电源 / 任务栏状态 | 收敛到控制面板与顶部状态栏 |
| 任务中心 | 任务管理器 | 改名为“任务管理器” |
| 系统设置 | 控制面板 | 合并进控制面板 |
| 用户中心 | 用户管理 / 个人中心 | 拆分到控制面板和用户菜单 |
| 远程访问 | 设备连接 / UGREENlink | 收敛到控制面板-设备连接 |
| 共享协议 | 文件服务 | 收敛到控制面板-文件服务 |
| 虚拟机 | 应用中心/虚拟机 | 作为应用中心应用 |
| 同步服务 | 同步与备份 | 合并 |
| iSCSI | 存储/应用中心 | 作为高级存储工具 |
| 硬件中心 | 硬件与电源 | 合并进控制面板 |

### 4.2 当前后端域覆盖

当前 `server-go/internal/httpapi/router.go` 已覆盖这些关键域：

| 后端域 | 已有 API 能力 | 与绿联对齐状态 |
| --- | --- | --- |
| system | info、identity、updates、backups、events | 可支撑关于本机、更新与还原 |
| desktop | apps、windows、session | 可支撑桌面壳 |
| files | tree、search、folders、upload、batch、fileByID | 可支撑文件管理基础 |
| storage | pools、spaces、disks、smart、tasks、snapshots | 可支撑存储管理 |
| tasks | list、stream、cancel | 可支撑任务管理器 |
| activity | activity log | 可支撑日志中心一部分 |
| downloads | tasks、speed profiles | 可支撑下载中心 |
| docker | stacks、images、volumes、networks、containers | 可支撑 Docker |
| backups | jobs | 可支撑备份 |
| vm | machines、capabilities、audit | 可支撑虚拟机 |
| iscsi | targets、capabilities、audit | 可支撑 iSCSI |
| sync | pairs、conflicts、audit | 可支撑同步 |
| app-center | apps、catalog、registries、audit | 可支撑应用中心 |
| remote | status、channel、tunnel、mfa、policy、domain-token、devices、alerts | 可支撑 UGREENlink/远程访问 |
| media | items、albums、memories、people、subtitles、transcode、shares | 可支撑相册和媒体 |
| music | library、scan、tracks、albums | 可支撑音乐 |
| videos | library、scan、items、tasks、live、DVR | 可支撑影视中心 |
| ai-analysis | records、batch、faces、progress | 可支撑 AI 分析 |
| assistant | threads、actions、presets、tools | 可支撑 UGREEN AI / Hermes |
| auth | login、logout、me、password、sessions、mfa、audit | 可支撑账号安全 |
| accounts | users、groups、grants、samba-sync | 可支撑用户管理 |
| steward/security | suggestions、audit、risk、inspect、host scan | 可支撑安全治理 |
| protocols | protocols、shares、audit | 可支撑文件服务 |
| shared-folders | folders、permissions | 可支撑共享文件夹 |
| network | interfaces、config、audit | 可支撑网络设置 |

结论：后端域大量可复用，重构重点应先放在前端产品结构和 API 聚合适配层，而不是推倒后端。

## 5. 差距矩阵

| 绿联能力 | 当前 HiGoOS 状态 | 差距 | 优先级 |
| --- | --- | --- | --- |
| 绿联桌面布局 | 有桌面壳 | 视觉与入口顺序不一致 | P0 |
| 顶部状态栏 | 有监控能力 | 绿联式 CPU/RAM/网速/任务/通知/搜索/用户未统一 | P0 |
| 控制面板 | 分散在多个窗口 | 需要合并成绿联控制面板 | P0 |
| 文件管理三空间 | 有文件树 | 需要改为个人/共享/用户/标签 | P0 |
| 应用中心 41 应用模型 | 有 app-center | 分类、状态、安装/打开模型需重做 | P0 |
| 任务管理器 | 有 tasks | 需要成为系统级任务入口 | P0 |
| 通知中心 | 部分 activity/alerts | 需要统一通知模型 | P1 |
| 日志中心 | 有 activity/monitoring logs | 需要独立日志中心应用 | P1 |
| 用户管理 | 有 accounts/auth | 需要合并控制面板用户管理 | P1 |
| 文件服务 | 有 protocols/shared-folders | 需要按 SMB/FTP/WebDAV/NFS/DLNA/SSH 呈现 | P1 |
| UGREENlink | 有 remote/cloud | 需要改成设备连接/远程访问体验 | P1 |
| MFA/OTP | 有 auth/mfa | 需要前端完整流程 | P1 |
| 回收站 | 前端有入口可能不足 | 需要文件删除/恢复/清空模型 | P1 |
| 文件版本管理 | 文案/部分能力 | 需要专门应用/页面 | P1 |
| 去重 | 有 AI 管家概念 | 需要改成文件管理子功能 | P2 |
| 在线文档 | 未明确完整实现 | 作为应用中心应用 | P2 |
| 文本编辑 | 未明确完整实现 | 轻量窗口应用 | P2 |
| 迅雷 | 未明确实现 | 可先作为外部应用占位 | P2 |
| 网盘工具 | 未明确实现 | 可先做任务状态和配置入口 | P2 |
| 帮助中心 | 未见完整应用 | 可先静态帮助中心 | P2 |
| Hermes Agent | 有 Agent 工作台 | 改名/包装 | P2 |

## 6. 重构总体原则

1. 先信息架构，后功能深挖。先让桌面、控制面板、文件管理、应用中心像绿联。
2. 后端不推倒。已有 Go 域服务继续复用，通过前端聚合层和适配层重排。
3. AI 能力后置。HiGoOS 的 AI/Agent 是差异点，但在绿联化方案中应作为 UGREEN AI 和 Hermes Agent 的子能力。
4. 系统级能力横切。任务、通知、日志、搜索、用户菜单不能散在业务窗口里。
5. 每个应用都要有三态：未安装/已安装未运行/已运行，外加更新态和权限态。
6. 高风险操作必须保持现有治理模型：预览、确认、审计、回滚。

## 7. 推荐重构架构

### 7.1 前端目录建议

```text
web-pc/src
├─ shell
│  ├─ DesktopShell.vue
│  ├─ TopStatusBar.vue
│  ├─ DesktopIconGrid.vue
│  ├─ WindowManager.vue
│  ├─ NotificationCenter.vue
│  ├─ TaskTray.vue
│  ├─ GlobalSearch.vue
│  └─ UserMenu.vue
├─ apps
│  ├─ control-panel
│  ├─ file-manager
│  ├─ app-center
│  ├─ storage-manager
│  ├─ task-manager
│  ├─ log-center
│  ├─ sync-backup
│  ├─ download-center
│  ├─ docker
│  ├─ video-center
│  ├─ music
│  ├─ photos
│  ├─ recycle-bin
│  ├─ text-editor
│  ├─ help-center
│  ├─ netdisk-tool
│  ├─ ugreen-ai
│  └─ hermes-agent
├─ stores
│  ├─ shell.ts
│  ├─ notifications.ts
│  ├─ tasks.ts
│  ├─ controlPanel.ts
│  ├─ fileManager.ts
│  └─ appCenter.ts
├─ api
│  ├─ domains
│  └─ adapters
└─ data
   ├─ ugreenDesktopApps.ts
   ├─ ugreenControlPanel.ts
   └─ ugreenAppCatalog.ts
```

### 7.2 系统应用模型

所有桌面应用统一使用一个模型：

```ts
type UgreenApp = {
  id: string;
  name: string;
  category: 'system' | 'utility' | 'entertainment' | 'download' | 'security' | 'backup';
  icon: string;
  entry: 'desktop' | 'control-panel' | 'app-center' | 'hidden';
  installState: 'built-in' | 'installed' | 'available' | 'updatable' | 'disabled';
  runState: 'closed' | 'opening' | 'running' | 'error';
  permissions: string[];
  window: {
    title: string;
    minWidth: number;
    minHeight: number;
  };
};
```

### 7.3 控制面板模型

```ts
type ControlPanelSection = {
  group: 'connection' | 'general' | 'system';
  id: string;
  name: string;
  icon: string;
  tabs?: string[];
  requiredRole: 'admin' | 'user';
};
```

## 8. 分阶段实施方案

### 阶段 1：桌面壳绿联化

目标：让第一眼完全像绿联 NAS。

范围：

- 重排桌面图标为绿联 18 个入口。
- 顶部状态栏加入 CPU、RAM、上行/下行网速、任务、通知、搜索、用户。
- 应用窗口使用绿联式标题栏、圆点控制、居中窗口。
- 增加应用库入口。
- 增加通知中心和任务中心的壳。

交付：

- 新桌面应用列表。
- 新窗口管理样式。
- 新顶部状态栏。
- 系统级任务/通知入口。

### 阶段 2：控制面板重构

目标：系统设置全部收敛到绿联控制面板。

范围：

- 新建 `ControlPanelWindow`。
- 一级分组：连接与访问、通用设置、系统服务。
- 页面：用户管理、文件服务、设备连接、域/LDAP、终端机、硬件与电源、时间和语言、网络设置、安全性、索引服务、关于本机、更新与还原。
- 硬件与电源页内增加：常规、存储、应用、服务、设备分析。

后端映射：

- 用户管理：`accounts`、`auth`
- 文件服务：`protocols`、`shared-folders`
- 设备连接：`remote`、`cloud`
- 网络设置：`network`
- 安全性：`security`、`auth/mfa`
- 关于本机：`system/info`、`hardware/inventory`
- 更新与还原：`system/updates`、`system/backups`

### 阶段 3：文件管理重构

目标：文件管理完全改成绿联结构。

范围：

- 左侧空间：个人文件夹、共享文件夹、用户文件夹、标签。
- 单独回收站应用，也可从文件管理进入。
- 工具栏：后退/前进/刷新、路径栏、搜索、新建、上传、复制、移动、剪切、删除、筛选、排序、视图、详情。
- 右键菜单：打开、下载、分享、属性、重命名、复制、移动、删除、标签、版本。
- 文件版本管理、潜在重复文件作为子窗口。

后端映射：

- `files/tree`
- `files/search`
- `files/folders`
- `files/upload`
- `files/batch/*`
- `shared-folders/*`
- `shares/*`

### 阶段 4：应用中心重构

目标：应用中心成为完整应用生命周期管理器。

范围：

- 分类：全部应用、系统管理、实用工具、娱乐、下载、安全、备份、已安装。
- 应用卡片：图标、名称、分类、状态、安装/打开/更新。
- 应用详情：版本、说明、权限、数据目录、端口、资源占用、日志。
- 内置 41 个应用模型，先用可见和已知应用填充，未知应用可保持 catalog 占位。

后端映射：

- `app-center/apps`
- `app-center/catalog`
- `app-center/registries`
- `app-center/audit`
- `docker/*`

### 阶段 5：任务、通知、日志横切能力

目标：所有长任务、告警、日志统一进系统级中心。

范围：

- 任务管理器：上传、下载、同步、备份、转码、扫描、存储修复、健康检查、系统更新。
- 通知中心：普通、重要、警告；来源筛选；时间筛选；删除/清空。
- 日志中心：系统日志、登录日志、操作日志、安全日志、应用日志。
- 顶部栏实时状态联动。

后端映射：

- `tasks`
- `activity`
- `monitoring/logs`
- `monitoring/alerts`
- `auth/audit`
- `security/audit`
- `app-center/audit`
- `protocols/audit`

### 阶段 6：媒体、下载、同步备份、Docker 深化

目标：补齐高频应用的细节体验。

范围：

- 影视中心：媒体库、扫描、刮削、字幕、转码、直播、DVR、片头片尾。
- 音乐：音乐库、专辑、歌曲、歌词、播放队列。
- 相册：时间线、人物、地点、回忆、共享相册、手机备份。
- 下载中心：HTTP、BT、磁力、迅雷、限速、队列。
- 同步与备份：同步对、备份任务、恢复、版本保留、冲突处理。
- Docker：容器、镜像、网络、卷、Compose、日志、终端。

### 阶段 7：AI 能力绿联包装

目标：不丢 HiGoOS 的 AI 优势，但入口和表达对齐绿联。

范围：

- `AI 助手` 改为 `UGREEN AI`。
- `Agent 工作台` 改为 `Hermes Agent`。
- `AI 分析中心` 合并到 UGREEN AI / 相册 AI。
- `AI 文件管家` 变成文件管理里的“智能整理/去重/权限建议”。
- AI 模型下载、AI NAS 启用、AI 控制台做成 UGREEN AI 的设置页。

## 9. 优先级排期

### P0：必须先做

1. 桌面入口完全绿联化。
2. 顶部状态栏绿联化。
3. 控制面板信息架构重构。
4. 文件管理三空间结构。
5. 应用中心分类和状态模型。
6. 任务管理器系统级入口。

### P1：第二批

1. 通知中心。
2. 日志中心。
3. 用户管理完整前端。
4. 文件服务完整前端。
5. UGREENlink / 远程访问完整前端。
6. MFA / OTP 完整前端。
7. 回收站完整能力。
8. 文件分享、版本、权限、去重。

### P2：第三批

1. 在线文档。
2. 文本编辑。
3. 迅雷接入或占位。
4. 网盘工具。
5. 帮助中心。
6. Hermes Agent 包装。
7. AI 模型下载和 AI 控制台。

## 10. 关键设计决策

### 10.1 不建议推倒后端

当前后端域已经非常丰富，直接推倒会浪费已有能力。建议做“前端绿联化 + API 聚合适配”：

- 绿联控制面板页面调用现有多个域 API。
- 绿联任务中心聚合所有异步任务。
- 绿联通知中心聚合 monitoring alerts、activity、audit。
- 绿联应用中心聚合 app-center 和 docker。

### 10.2 AI 不再作为桌面主骨架

当前 HiGoOS 很多入口带 AI 色彩，但绿联体验里 AI 是一个应用入口，而不是整个系统的信息架构。建议：

- 桌面只保留 `UGREEN AI` 和 `Hermes Agent`。
- AI 文件整理、AI 分析、AI 搜索放到具体业务里。
- 仍保留后端 AI 能力，不删除。

### 10.3 控制面板是重构核心

绿联 NAS 的系统能力都围绕控制面板组织。当前 HiGoOS 的用户、远程、协议、硬件、设置、安全比较分散，必须收敛。

### 10.4 任务/通知/日志必须统一

上传、下载、同步、备份、转码、扫描、存储修复、系统更新都应该进入同一个任务中心，否则体验不像绿联。

## 11. 建议第一版交付范围

第一版建议只做“外壳和信息架构”，不要急着补所有业务细节：

1. 绿联桌面。
2. 绿联顶部状态栏。
3. 绿联应用中心静态/半动态 catalog。
4. 绿联控制面板分组和 12 个入口。
5. 绿联文件管理布局。
6. 任务管理器统一入口。

这样最快能看到产品形态变化，也最少影响现有后端。

## 12. 验收标准

### 12.1 产品验收

- 打开系统后，第一屏入口与绿联 NAS 一致。
- 控制面板首屏能看到三组：连接与访问、通用设置、系统服务。
- 文件管理左侧能看到：个人文件夹、共享文件夹、用户文件夹、标签。
- 应用中心能看到：全部应用、系统管理、实用工具、娱乐、下载、安全、备份、已安装。
- 顶部栏能看到 CPU、RAM、网速、任务、通知、搜索、用户。
- AI 不再散落为多个主入口，而是集中到 UGREEN AI / Hermes Agent。

### 12.2 技术验收

- 不破坏现有 API。
- 现有 `server-go` 测试通过。
- `web-pc` 能通过类型检查和构建。
- 高风险写操作仍保留确认、审计、回滚流程。
- 任务中心可以聚合至少下载、同步、备份、转码、存储任务。

## 13. 结论

HiGoOS 当前基础并不弱，甚至后端域比绿联可见页面更“AI 原生”。但如果目标是完全按绿联走，重构的主线应该是：

```text
先像绿联，再增强 HiGoOS。
```

推荐顺序：

1. 桌面壳绿联化。
2. 控制面板绿联化。
3. 文件管理绿联化。
4. 应用中心绿联化。
5. 任务/通知/日志系统化。
6. 媒体、同步、下载、Docker 深化。
7. AI 能力包装为 UGREEN AI 与 Hermes Agent。

这条路线可以最大化复用现有项目，又能把产品体验快速拉到绿联 NAS 的结构和观感上。
