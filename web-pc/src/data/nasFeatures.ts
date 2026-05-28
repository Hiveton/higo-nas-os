export type NasFeatureKey =
  | 'storage'
  | 'files'
  | 'media'
  | 'photos'
  | 'remote'
  | 'settings'
  | 'protocols'
  | 'vm'
  | 'sync'
  | 'backup'
  | 'iscsi'
  | 'ai'
  | 'security'
  | 'downloads'
  | 'apps'
  | 'docker'
  | 'monitoring'
  | 'hardware';

export type NasFeatureAction = {
  label: string;
  state: 'ready' | 'mock' | 'todo';
};

export type NasFeature = {
  title: string;
  detail: string;
  actions: NasFeatureAction[];
};

export const nasFeatures: Record<NasFeatureKey, NasFeature[]> = {
  storage: [
    {
      title: '存储空间向导',
      detail: '覆盖文件系统选择、硬盘选择、RAID/Basic 模式、用户配额、格式化方式和创建后检测。',
      actions: [
        { label: '创建存储空间', state: 'mock' },
        { label: '添加硬盘', state: 'mock' },
        { label: '删除存储空间', state: 'mock' },
      ],
    },
    {
      title: '磁盘与缓存',
      detail: '展示硬盘休眠、读写检测、SMART、SSD 缓存、外接盘风险和阵列修复。',
      actions: [
        { label: 'SSD 缓存', state: 'mock' },
        { label: '读写检测', state: 'mock' },
        { label: '硬盘休眠', state: 'mock' },
      ],
    },
  ],
  files: [
    {
      title: '文件基础操作',
      detail: '覆盖上传、下载、新建文件夹、重命名、移动、复制、删除、回收站和批量任务。',
      actions: [
        { label: '上传', state: 'mock' },
        { label: '下载', state: 'mock' },
        { label: '回收站', state: 'mock' },
      ],
    },
    {
      title: '共享与权限',
      detail: '覆盖我的文件共享、外部分享链接、团队文件夹、父子级权限继承和访问级别。',
      actions: [
        { label: '共享给用户', state: 'mock' },
        { label: '外链分享', state: 'mock' },
        { label: '团队权限', state: 'mock' },
      ],
    },
  ],
  protocols: [
    {
      title: '共享协议',
      detail: '集中管理 SMB、NFS、WebDAV、DLNA，展示启停状态、挂载地址、权限和兼容性提示。',
      actions: [
        { label: 'SMB', state: 'mock' },
        { label: 'NFS', state: 'mock' },
        { label: 'WebDAV', state: 'mock' },
      ],
    },
  ],
  media: [
    {
      title: '影视服务器',
      detail: '覆盖安装影视应用、创建媒体库、家庭账号、Apple TV Infuse/VidHub 接入和刮削修正。',
      actions: [
        { label: '创建媒体库', state: 'mock' },
        { label: '账号授权', state: 'mock' },
        { label: '匹配修正', state: 'mock' },
      ],
    },
  ],
  photos: [
    {
      title: '相册与手机备份',
      detail: '覆盖手机自动备份、共享相册、AI 相册、Google Photos 导入、鸿蒙设备备份和视频兼容。',
      actions: [
        { label: '手机备份', state: 'mock' },
        { label: '共享相册', state: 'mock' },
        { label: 'AI 相册', state: 'mock' },
      ],
    },
  ],
  remote: [
    {
      title: '远程访问',
      detail: '覆盖远程连接、权益管理、直连/中继、域名令牌、设备绑定和登录告警。',
      actions: [
        { label: '开启通道', state: 'mock' },
        { label: '设备绑定', state: 'mock' },
        { label: '权益管理', state: 'mock' },
      ],
    },
  ],
  settings: [
    {
      title: '系统配置',
      detail: '覆盖用户、无线网卡、蓝牙配网、端口修改、邮件通知、2FA 和系统配置备份还原。',
      actions: [
        { label: '创建用户', state: 'mock' },
        { label: '修改端口', state: 'mock' },
        { label: '配置备份', state: 'mock' },
      ],
    },
  ],
  vm: [
    {
      title: '虚拟机',
      detail: '覆盖虚拟机安装、镜像、CPU/内存/磁盘配置、硬件直通、启动控制和控制台访问。',
      actions: [
        { label: '新建虚拟机', state: 'mock' },
        { label: '硬件直通', state: 'mock' },
        { label: '控制台', state: 'mock' },
      ],
    },
  ],
  sync: [
    {
      title: '同步服务',
      detail: '覆盖设备同步、按需同步、冲突处理、选择性同步、带宽策略和常见问题诊断。',
      actions: [
        { label: '新增同步', state: 'mock' },
        { label: '按需同步', state: 'mock' },
        { label: '冲突处理', state: 'mock' },
      ],
    },
  ],
  backup: [
    {
      title: '备份任务',
      detail: '覆盖加密备份、计划任务、校验、恢复、异地目标、保留策略和执行日志。',
      actions: [
        { label: '新建备份', state: 'mock' },
        { label: '加密策略', state: 'mock' },
        { label: '恢复演练', state: 'mock' },
      ],
    },
  ],
  iscsi: [
    {
      title: 'iSCSI',
      detail: '覆盖 Target、LUN、用户组、CHAP、启动器连接、容量分配和常见问题。',
      actions: [
        { label: '新建 Target', state: 'mock' },
        { label: '新建 LUN', state: 'mock' },
        { label: 'CHAP', state: 'mock' },
      ],
    },
  ],
  ai: [
    {
      title: 'AI 与 OpenClaw',
      detail: '覆盖微信、钉钉、飞书机器人配置，本地/云端模型策略和工具权限审批。',
      actions: [
        { label: '微信机器人', state: 'mock' },
        { label: '钉钉机器人', state: 'mock' },
        { label: '飞书机器人', state: 'mock' },
      ],
    },
  ],
  security: [
    {
      title: '安全治理',
      detail: '覆盖 2FA、安全披露、漏洞报告、分享链接风险、账号权限和审计回滚。',
      actions: [
        { label: '启用 2FA', state: 'mock' },
        { label: '风险扫描', state: 'mock' },
        { label: '审计回滚', state: 'mock' },
      ],
    },
  ],
  downloads: [
    {
      title: '下载与归档',
      detail: '覆盖 HTTP、BT、磁力、订阅、限速、完成后归档和失败重试。',
      actions: [
        { label: 'HTTP', state: 'mock' },
        { label: '磁力', state: 'mock' },
        { label: '自动归档', state: 'mock' },
      ],
    },
  ],
  apps: [
    {
      title: '应用中心',
      detail: '覆盖应用安装、更新、启动停止、端口、数据目录、权限和资源占用。',
      actions: [
        { label: '安装', state: 'mock' },
        { label: '更新', state: 'mock' },
        { label: '权限', state: 'mock' },
      ],
    },
  ],
  docker: [
    {
      title: '容器与 Compose',
      detail: '覆盖容器启停、日志、资源限制、端口、卷挂载、镜像和 Compose 栈。',
      actions: [
        { label: 'Compose', state: 'mock' },
        { label: '日志', state: 'mock' },
        { label: '资源限制', state: 'mock' },
      ],
    },
  ],
  monitoring: [
    {
      title: '设备监控',
      detail: '覆盖 CPU、内存、网络、磁盘、温度、服务、告警、诊断和系统日志。',
      actions: [
        { label: '诊断', state: 'mock' },
        { label: '告警规则', state: 'mock' },
        { label: '日志', state: 'mock' },
      ],
    },
  ],
  hardware: [
    {
      title: '硬件资料',
      detail: '覆盖 NAS 官方硬件、零刻合作机型、ARM 公测、Rockchip/Amlogic 刷机教程和兼容性说明。',
      actions: [
        { label: '官方硬件', state: 'mock' },
        { label: '兼容性资料', state: 'mock' },
        { label: '刷机教程', state: 'mock' },
      ],
    },
  ],
};
