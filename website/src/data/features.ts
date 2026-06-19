export type FeatureTabKey = 'files' | 'ai' | 'security' | 'ecosystem';

export type FeatureCard = {
  title: string;
  detail: string;
  badge: string;
};

export const featureCards: FeatureCard[] = [
  {
    title: '文件管理',
    detail: '家庭空间、团队空间、标签、分享、搜索与回收站在一个桌面窗口内完成。',
    badge: 'Files',
  },
  {
    title: '存储管理',
    detail: '容量、磁盘健康、SMART、快照、修复任务和系统资源持续可见。',
    badge: 'Storage',
  },
  {
    title: '相册媒体',
    detail: '照片时间线、人物地点、影视刮削、转码和音乐库面向家庭内容中心。',
    badge: 'Media',
  },
  {
    title: '备份同步',
    detail: '手机备份、异地同步、计划任务、校验恢复和任务进度都可追踪。',
    badge: 'Backup',
  },
  {
    title: '下载中心',
    detail: 'HTTP、BT、磁力任务与完成后自动归档，适合长期运行的 NAS 下载流。',
    badge: 'Download',
  },
  {
    title: 'Docker 应用',
    detail: '应用中心、容器、Compose、日志、端口与资源限制构成可扩展生态。',
    badge: 'Apps',
  },
  {
    title: '远程访问',
    detail: '设备绑定、直连/中继、域名令牌、登录告警和分享扫描面向外网访问。',
    badge: 'Remote',
  },
  {
    title: '设备监控',
    detail: 'CPU、内存、网络、硬盘、日志、告警和诊断形成实时运维台。',
    badge: 'Monitor',
  },
];

export const featureTabs = [
  {
    key: 'files' as const,
    label: '文件与媒体',
    title: 'NAS 桌面把文件、相册、下载和备份放在同一个工作台',
    detail: '保留真实桌面窗口密度，不把 NAS 简化成普通网盘。用户进入后看到的是文件空间、AI 建议、任务进度和设备状态。',
    stats: ['家庭空间 / 团队空间', '照片与视频中心', '任务流持续运行'],
  },
  {
    key: 'ai' as const,
    label: 'AI 原生',
    title: 'AI 文件管家先预览，再执行，最后可审计回滚',
    detail: '语义搜索、智能整理、Agent 工作台和助手会围绕用户可访问的数据工作，并把风险动作交给人确认。',
    stats: ['6 条整理建议', 'mimo-v2.5-pro', '权限内检索'],
  },
  {
    key: 'security' as const,
    label: '安全治理',
    title: '中高风险动作必须确认，所有执行都有审计线索',
    detail: '分享、权限、删除、移动、Docker、网络和 Agent 动作都遵循预览、确认、审计、回滚的治理路径。',
    stats: ['风险分级', '确认 token', '回滚记录'],
  },
  {
    key: 'ecosystem' as const,
    label: '应用生态',
    title: 'Docker、应用中心、共享协议和远程访问构成 NASOS 扩展层',
    detail: 'HiGoOS 面向可部署系统：WebDAV/SMB/NFS、Docker 应用、远程访问、桌面发现工具和 Web 初始化都在同一产品路径内。',
    stats: ['Docker / Compose', 'SMB / WebDAV', '桌面扫描工具'],
  },
];
