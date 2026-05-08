import type { Component } from 'vue';
import {
  ArchiveRestore,
  Bell,
  Bot,
  CheckCircle2,
  CircleAlert,
  Database,
  FileSearch,
  FolderClosed,
  HardDrive,
  ShieldCheck,
  Sparkles,
  Workflow,
} from 'lucide-vue-next';

import fileManagerIcon from '../assets/higoos-dock/icons/01-file-manager.png';
import storageManagerIcon from '../assets/higoos-dock/icons/02-storage-manager.png';
import aiFileStewardIcon from '../assets/higoos-dock/icons/03-ai-file-steward.png';
import agentWorkbenchIcon from '../assets/higoos-dock/icons/04-agent-workbench.png';
import aiAssistantIcon from '../assets/higoos-dock/icons/05-ai-assistant.png';
import backupSyncIcon from '../assets/higoos-dock/icons/06-backup-sync.png';
import photoMediaIcon from '../assets/higoos-dock/icons/07-photo-media.png';
import downloadCenterIcon from '../assets/higoos-dock/icons/08-download-center.png';
import appCenterIcon from '../assets/higoos-dock/icons/09-app-center.png';
import dockerIcon from '../assets/higoos-dock/icons/10-docker.png';
import securityCenterIcon from '../assets/higoos-dock/icons/11-security-center.png';
import deviceMonitorIcon from '../assets/higoos-dock/icons/12-device-monitor.png';
import systemSettingsIcon from '../assets/higoos-dock/icons/13-system-settings.png';
import remoteAccessIcon from '../assets/higoos-dock/icons/14-remote-access.png';

export type DockApp = {
  id: string;
  name: string;
  icon: string;
  badge?: number;
  utility?: boolean;
};

export type DesktopWindowConfig = {
  id: string;
  title: string;
  subtitle: string;
  status: string;
  statusTone: 'blue' | 'green' | 'orange' | 'red';
  x: number;
  y: number;
  width: number;
  height: number;
  z: number;
};

export type FileRow = {
  name: string;
  type: string;
  space: string;
  size: string;
  modified: string;
  tags: string[];
  permission: string;
  aiSummary: string;
};

export type StewardSuggestion = {
  title: string;
  detail: string;
  count: string;
  risk: '低风险' | '中风险' | '高风险';
  action: string;
};

export type AgentTemplate = {
  name: string;
  desc: string;
  tools: string[];
  risk: string;
};

export const dockApps: DockApp[] = [
  { id: 'file-manager', name: '文件管理', icon: fileManagerIcon, badge: 2 },
  { id: 'storage-monitor', name: '存储管理', icon: storageManagerIcon },
  { id: 'ai-file-steward', name: 'AI 文件管家', icon: aiFileStewardIcon, badge: 6 },
  { id: 'agent-workbench', name: 'Agent 工作台', icon: agentWorkbenchIcon },
  { id: 'ai-assistant', name: 'AI 助手', icon: aiAssistantIcon },
  { id: 'backup-sync', name: '备份同步', icon: backupSyncIcon, badge: 1 },
  { id: 'photo-media', name: '相册媒体', icon: photoMediaIcon },
  { id: 'download-center', name: '下载中心', icon: downloadCenterIcon },
  { id: 'app-center', name: '应用中心', icon: appCenterIcon },
  { id: 'docker', name: 'Docker', icon: dockerIcon },
  { id: 'security-center', name: '安全中心', icon: securityCenterIcon, badge: 3 },
  { id: 'device-monitor', name: '设备监控', icon: deviceMonitorIcon },
  { id: 'system-settings', name: '系统设置', icon: systemSettingsIcon },
  { id: 'remote-access', name: '远程访问', icon: remoteAccessIcon },
];

export const desktopWindows: DesktopWindowConfig[] = [
  {
    id: 'file-manager',
    title: '文件管理',
    subtitle: '家庭空间 / 团队空间 / 语义搜索',
    status: 'AI 索引已同步',
    statusTone: 'green',
    x: 48,
    y: 92,
    width: 700,
    height: 560,
    z: 4,
  },
  {
    id: 'ai-file-steward',
    title: 'AI 文件管家',
    subtitle: '智能整理 / 权限审计 / 回滚',
    status: '6 条建议',
    statusTone: 'orange',
    x: 778,
    y: 108,
    width: 472,
    height: 520,
    z: 5,
  },
  {
    id: 'agent-workbench',
    title: 'Agent 工作台',
    subtitle: '工作流 / 工具权限 / 执行确认',
    status: '需要确认',
    statusTone: 'blue',
    x: 548,
    y: 388,
    width: 690,
    height: 360,
    z: 6,
  },
  {
    id: 'backup-sync',
    title: '备份同步',
    subtitle: '快照 / 异地同步 / 校验',
    status: '1 个任务同步中',
    statusTone: 'blue',
    x: 188,
    y: 126,
    width: 740,
    height: 500,
    z: 7,
  },
  {
    id: 'storage-monitor',
    title: '存储管理',
    subtitle: '主机卷 / SMART / 容量',
    status: '后端同步',
    statusTone: 'green',
    x: 960,
    y: 80,
    width: 360,
    height: 296,
    z: 3,
  },
  {
    id: 'photo-media',
    title: '相册媒体',
    subtitle: '时间线 / 人物地点 / 媒体转码',
    status: '回忆生成',
    statusTone: 'blue',
    x: 118,
    y: 118,
    width: 760,
    height: 536,
    z: 8,
  },
  {
    id: 'download-center',
    title: '下载中心',
    subtitle: 'BT / HTTP / 磁力 / 自动归档',
    status: '队列运行',
    statusTone: 'green',
    x: 228,
    y: 136,
    width: 720,
    height: 500,
    z: 9,
  },
  {
    id: 'app-center',
    title: '应用中心',
    subtitle: '套件 / 容器应用 / 更新',
    status: '1 个更新',
    statusTone: 'orange',
    x: 248,
    y: 118,
    width: 760,
    height: 520,
    z: 10,
  },
  {
    id: 'docker',
    title: 'Docker',
    subtitle: '容器 / Compose / 端口资源',
    status: '4 个运行',
    statusTone: 'green',
    x: 260,
    y: 112,
    width: 760,
    height: 520,
    z: 11,
  },
  {
    id: 'security-center',
    title: '安全中心',
    subtitle: '权限 / 风险 / 审计回滚',
    status: '3 条风险',
    statusTone: 'red',
    x: 300,
    y: 126,
    width: 760,
    height: 530,
    z: 12,
  },
  {
    id: 'device-monitor',
    title: '设备监控',
    subtitle: '性能趋势 / 告警 / 系统日志',
    status: '实时',
    statusTone: 'green',
    x: 170,
    y: 104,
    width: 760,
    height: 520,
    z: 13,
  },
  {
    id: 'system-settings',
    title: '系统设置',
    subtitle: '网络 / 模型 / 隐私 / 更新',
    status: '已同步',
    statusTone: 'blue',
    x: 210,
    y: 96,
    width: 760,
    height: 536,
    z: 14,
  },
  {
    id: 'remote-access',
    title: '远程访问',
    subtitle: '域名 / 穿透 / MFA / 分享检查',
    status: '安全',
    statusTone: 'green',
    x: 248,
    y: 116,
    width: 740,
    height: 514,
    z: 15,
  },
];

export const folders = ['家庭空间', '团队空间', '照片与视频', '财务票据', '项目资料', 'Docker 数据', '备份归档'];

export const files: FileRow[] = [
  {
    name: '2026 家庭保险与保修资料.pdf',
    type: 'PDF',
    space: '家庭空间',
    size: '18.4 MB',
    modified: '今天 09:42',
    tags: ['保修', '家庭知识库', '需提醒'],
    permission: '家人可见',
    aiSummary: '包含 6 份家电保修单，2 项将在 30 天内到期。',
  },
  {
    name: '客户A_合同_最终版.docx',
    type: 'DOCX',
    space: '团队空间',
    size: '2.8 MB',
    modified: '昨天 18:10',
    tags: ['合同', '客户A', '权限敏感'],
    permission: '项目组',
    aiSummary: '识别到付款节点和保密条款，建议加入项目资料图谱。',
  },
  {
    name: '五一旅行照片精选',
    type: '相册',
    space: '照片与视频',
    size: '4.2 GB',
    modified: '昨天 12:26',
    tags: ['旅行', '人物已识别', '可生成回忆'],
    permission: '家人可见',
    aiSummary: '832 张照片，已筛出 124 张清晰照片和 18 段视频。',
  },
  {
    name: '下载目录/未归档发票',
    type: '文件夹',
    space: '财务票据',
    size: '642 MB',
    modified: '周一 21:05',
    tags: ['待整理', '发票', '重复项'],
    permission: '仅管理员',
    aiSummary: '发现 31 张发票，其中 4 张可能重复，建议按年月归档。',
  },
];

export const stewardSuggestions: StewardSuggestion[] = [
  {
    title: '下载目录智能整理',
    detail: '31 张发票、12 个安装包和 4 个重复压缩包可按规则归档。',
    count: '47 项',
    risk: '中风险',
    action: '预览整理',
  },
  {
    title: '过期分享链接',
    detail: '发现 3 个公开链接仍可访问，包含团队空间资料。',
    count: '3 个',
    risk: '高风险',
    action: '查看权限',
  },
  {
    title: '相似照片清理',
    detail: '五一旅行相册中有 86 张连拍相似照片，可保留清晰版本。',
    count: '1.6 GB',
    risk: '低风险',
    action: '智能筛选',
  },
];

export const auditEntries = [
  '09:41 文件管家读取 /下载/票据，仅生成建议，未移动文件',
  '09:22 Agent 创建家庭保修提醒，等待管理员确认',
  '昨天 18:36 撤销 12 个文件重命名，已恢复原路径',
];

export const agentTemplates: AgentTemplate[] = [
  {
    name: '家庭资料助手',
    desc: '整理保修单、说明书、证件和医疗资料，提供问答与提醒。',
    tools: ['文件搜索', '摘要', '提醒', '分享'],
    risk: '中风险',
  },
  {
    name: '项目资料 Agent',
    desc: '汇总项目文件、合同、会议纪要和素材，生成资料包。',
    tools: ['语义搜索', '文件夹摘要', '打包', '权限检查'],
    risk: '中风险',
  },
  {
    name: '设备运维 Agent',
    desc: '监控硬盘、备份、Docker 和网络状态，异常时建议处理。',
    tools: ['设备监控', '备份检查', '通知', '日志读取'],
    risk: '低风险',
  },
];

export const workflowNodes = [
  { label: '触发', value: '新文件进入下载目录', icon: FileSearch },
  { label: '理解', value: 'OCR + 发票识别 + 重复检测', icon: Sparkles },
  { label: '确认', value: '中风险，等待用户确认', icon: CircleAlert },
  { label: '执行', value: '重命名、归档、写入审计', icon: ArchiveRestore },
];

export const assistantMessages = [
  {
    role: 'user',
    text: '找一下上个月客户 A 的最终合同，并确认有没有备份。',
  },
  {
    role: 'assistant',
    text: '找到了 1 份最终版合同，位于团队空间/客户A/合同。该文件已进入每日快照和异地备份，权限为项目组可见。',
  },
  {
    role: 'assistant',
    text: '我还发现 3 个相关附件未加入项目资料图谱，是否需要生成整理计划？',
  },
];

export const metrics = [
  { label: 'CPU', value: '32%', trend: '+4%', icon: Database },
  { label: '内存', value: '58%', trend: '-2%', icon: HardDrive },
  { label: '上传', value: '18 MB/s', trend: '稳定', icon: CheckCircle2 },
  { label: '下载', value: '42 MB/s', trend: '+12%', icon: Bell },
];

export const alerts = [
  { title: '备份中', detail: 'MacBook Pro 文档备份 72%', tone: 'blue', icon: ArchiveRestore },
  { title: '权限审计', detail: '3 个分享链接建议收紧', tone: 'orange', icon: ShieldCheck },
  { title: '本地模型', detail: '隐私空间强制本地推理', tone: 'green', icon: Bot },
] satisfies Array<{ title: string; detail: string; tone: string; icon: Component }>;
