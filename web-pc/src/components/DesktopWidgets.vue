<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { Component } from 'vue';
import {
  ArchiveRestore,
  Bell,
  Bot,
  Boxes,
  CheckCircle2,
  Cpu,
  Database,
  HardDrive,
  LockKeyhole,
  MemoryStick,
  Network,
  ShieldAlert,
  ShieldCheck,
  UploadCloud,
} from 'lucide-vue-next';
import { apiClient } from '../api/client';
import type { Alert, BackupJob, DockerContainer, Metric, RiskAction, StoragePool } from '../api/types';
import { alerts as seedAlerts, metrics as seedMetrics } from '../data/higoos';

type Tone = 'blue' | 'green' | 'orange' | 'red' | 'cyan';

type MiniCard = {
  label: string;
  value: string;
  detail: string;
  tone: Tone;
  icon: Component;
};

const fallbackBackupJobs: BackupJob[] = [
  {
    id: 'family-photo',
    name: '家庭相册增量备份',
    source: '照片与视频',
    target: '照片与视频 -> 异地备份卷',
    state: '同步中',
    schedule: '每 6 小时',
    progress: 72,
    speed: '18 MB/s',
    eta: '剩余 16 分钟',
    lastRun: '今天 08:30',
    nextRun: '今天 14:30',
    retention: '保留 180 天',
    policy: '去重 + 加密 + 远端校验',
    health: '正常',
    enabled: true,
  },
  {
    id: 'team-snapshot',
    name: '团队空间快照',
    source: '项目资料',
    target: '项目资料 -> 每日快照',
    state: '校验中',
    schedule: '每天 02:00',
    progress: 94,
    speed: '已校验 1.8 TB',
    eta: '等待归档索引',
    lastRun: '今天 02:00',
    nextRun: '明天 02:00',
    retention: '保留 365 天',
    policy: '只读快照 + 变更审计',
    health: '正常',
    enabled: true,
  },
];

const fallbackDockerContainers: DockerContainer[] = [
  {
    id: 'jellyfin',
    name: 'jellyfin-media',
    image: 'jellyfin/jellyfin:10.9',
    stack: 'media-stack',
    status: '运行中',
    cpu: 18,
    memory: 42,
    memoryText: '1.7 GB / 4 GB',
    ports: ['8096:8096/tcp'],
    mounts: [],
    env: [],
    limitCpu: 4,
    limitMemory: 4096,
    restarts: 1,
    isolation: '只读媒体库',
    log: [],
  },
  {
    id: 'qdrant',
    name: 'Qdrant 向量库',
    image: 'qdrant/qdrant:1.14',
    stack: 'home-ai',
    status: '运行中',
    cpu: 14,
    memory: 28,
    memoryText: '1.2 GB / 2 GB',
    ports: ['6333:6333/tcp'],
    mounts: [],
    env: [],
    limitCpu: 2,
    limitMemory: 2048,
    restarts: 0,
    isolation: 'AI 数据层',
    log: [],
  },
];

const fallbackSecurityWarnings = [
  { title: '公开分享链接', detail: '3 个团队空间链接建议收紧权限', tone: 'orange' as const },
  { title: '登录保护', detail: '管理员 2FA 与异地登录风控正常', tone: 'green' as const },
  { title: 'AI 执行审计', detail: '高风险文件操作保持人工确认', tone: 'blue' as const },
];

const storagePools = ref<StoragePool[]>([]);
const metrics = ref<Metric[]>(seedMetrics);
const alerts = ref<Array<Alert & { icon?: Component }>>(seedAlerts);
const backupJobs = ref<BackupJob[]>(fallbackBackupJobs);
const dockerContainers = ref<DockerContainer[]>(fallbackDockerContainers);
const riskActions = ref<RiskAction[]>([]);
const widgetNotice = ref('桌面概览正在从后端同步真实存储卷、监控和告警。');

const dockerStatus = computed(() => {
  const containers = dockerContainers.value.length ? dockerContainers.value : fallbackDockerContainers;
  const running = containers.filter((container) => container.status === '运行中').length;
  const paused = containers.filter((container) => container.status !== '运行中').length;
  const cpu = `${Math.min(100, containers.reduce((sum, container) => sum + container.cpu, 0))}%`;
  const memory = `${Math.round(containers.reduce((sum, container) => sum + container.memory, 0) / Math.max(containers.length, 1))}%`;
  return {
    running,
    paused,
    cpu,
    memory,
    services: containers.slice(0, 3).map((container) => container.name),
  };
});

const securityWarnings = computed(() => {
  if (!riskActions.value.length) return fallbackSecurityWarnings;
  return riskActions.value.slice(0, 3).map((risk) => ({
    title: risk.title,
    detail: `${risk.actor} · ${risk.scope} · ${risk.state}`,
    tone: toneForRisk(risk.level),
  }));
});

const miniCards = computed<MiniCard[]>(() => [
  {
    label: metrics.value[0]?.label ?? 'CPU',
    value: formatMetricValue(metrics.value[0], '32%'),
    detail: metrics.value[0]?.detail ?? `容器调度 ${dockerStatus.value.cpu}`,
    tone: 'cyan',
    icon: Cpu,
  },
  {
    label: metrics.value[1]?.label ?? '内存',
    value: formatMetricValue(metrics.value[1], '58%'),
    detail: metrics.value[1]?.detail ?? `模型缓存 ${dockerStatus.value.memory}`,
    tone: 'green',
    icon: MemoryStick,
  },
  {
    label: '网络',
    value: `${formatMetricValue(metrics.value[2], '18 MB/s')} / ${formatMetricValue(metrics.value[3], '42 MB/s')}`,
    detail: '远程访问链路稳定',
    tone: 'blue',
    icon: Network,
  },
]);

async function loadWidgetState() {
  try {
    const [nextPools, nextMetrics, nextAlerts] = await Promise.all([
      apiClient.storage.getPools(),
      apiClient.monitoring.getCurrentMetrics(),
      apiClient.monitoring.getAlerts(),
      apiClient.backup.getJobs().then((jobs) => {
        backupJobs.value = jobs;
      }),
      apiClient.docker.getContainers().then((containers) => {
        dockerContainers.value = containers;
      }),
      apiClient.security.getRiskActions().then((risks) => {
        riskActions.value = risks;
      }),
    ]);
    storagePools.value = nextPools;
    metrics.value = nextMetrics;
    alerts.value = nextAlerts.map((alert) => ({ ...alert, icon: iconForAlert(alert) }));
    widgetNotice.value = '桌面概览已从后端同步。';
  } catch (error) {
    storagePools.value = [];
    widgetNotice.value = `后端暂不可用，存储池不使用本地演示数据：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

function toneForRisk(risk: string): Tone {
  if (risk === '高风险') return 'red';
  if (risk === '中风险') return 'orange';
  if (risk === '低风险') return 'blue';
  return 'green';
}

function toneClass(tone: string) {
  return `is-${tone}`;
}

function formatMetricValue(metric: Metric | undefined, fallback: string) {
  if (!metric) return fallback;
  const value = metric.unit ? `${metric.value}${metric.unit}` : String(metric.value);
  return value;
}

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

function iconForAlert(alert: Alert): Component {
  const text = `${alert.title} ${alert.detail}`;
  if (text.includes('备份')) return ArchiveRestore;
  if (text.includes('模型')) return Bot;
  if (text.includes('权限') || text.includes('审计')) return ShieldCheck;
  if (text.includes('CPU')) return Database;
  if (text.includes('硬盘') || text.includes('存储')) return HardDrive;
  if (alert.tone === 'green') return CheckCircle2;
  return Bell;
}

onMounted(loadWidgetState);
</script>

<template>
  <aside class="desktop-widgets" aria-label="HiGoOS 桌面小组件">
    <section class="widget-card storage-card" aria-label="存储池健康">
      <div class="widget-card__header">
        <div>
          <p class="eyebrow">存储池健康</p>
          <h2 :title="widgetNotice">AI NAS 总览</h2>
        </div>
        <ShieldCheck class="header-icon is-green" :size="22" />
      </div>

      <div class="pool-list">
        <article v-if="storagePools.length === 0" class="pool-row pool-row--empty">
          <div class="pool-row__top">
            <div>
              <strong>等待后端磁盘数据</strong>
              <span>未使用本地演示容量</span>
            </div>
            <em>--</em>
          </div>
        </article>
        <article v-for="pool in storagePools" :key="pool.name" class="pool-row">
          <div class="pool-row__top">
            <div>
              <strong>{{ pool.name }}</strong>
              <span>{{ pool.type }} · {{ pool.total }}</span>
            </div>
            <em>{{ pool.health }}</em>
          </div>
          <div class="meter" aria-hidden="true">
            <span :style="{ width: `${clampPercent(pool.used)}%` }"></span>
          </div>
          <div class="pool-row__meta">
            <span>{{ clampPercent(pool.used) }}% 已用</span>
            <span>{{ pool.temp }}</span>
          </div>
        </article>
      </div>
    </section>

    <section class="widget-card backup-card" aria-label="备份进度">
      <div class="widget-card__header">
        <div>
          <p class="eyebrow">备份进度</p>
          <h2>快照与异地同步</h2>
        </div>
        <UploadCloud class="header-icon is-blue" :size="22" />
      </div>

      <article v-for="job in backupJobs" :key="job.name" class="backup-job">
        <div class="backup-job__copy">
          <strong>{{ job.name }}</strong>
          <span>{{ job.target }}</span>
        </div>
        <b>{{ job.progress }}%</b>
        <div class="meter backup-meter" aria-hidden="true">
          <span :style="{ width: `${job.progress}%` }"></span>
        </div>
        <div class="backup-job__meta">
          <span>{{ job.speed }}</span>
          <span>{{ job.eta }}</span>
        </div>
      </article>
    </section>

    <section class="widget-card docker-card" aria-label="Docker 状态">
      <div class="widget-card__header">
        <div>
          <p class="eyebrow">Docker 状态</p>
          <h2>应用容器运行正常</h2>
        </div>
        <Boxes class="header-icon is-cyan" :size="22" />
      </div>

      <div class="docker-grid">
        <div>
          <b>{{ dockerStatus.running }}</b>
          <span>运行中</span>
        </div>
        <div>
          <b>{{ dockerStatus.paused }}</b>
          <span>待更新</span>
        </div>
        <div>
          <b>{{ dockerStatus.cpu }}</b>
          <span>CPU</span>
        </div>
      </div>

      <div class="service-strip" aria-label="关键容器">
        <span v-for="service in dockerStatus.services" :key="service">{{ service }}</span>
      </div>
    </section>

    <section class="widget-card security-card" aria-label="安全告警">
      <div class="widget-card__header">
        <div>
          <p class="eyebrow">安全告警</p>
          <h2>权限与 AI 执行风控</h2>
        </div>
        <ShieldAlert class="header-icon is-orange" :size="22" />
      </div>

      <div class="warning-list">
        <article v-for="warning in securityWarnings" :key="warning.title" :class="['warning-row', toneClass(warning.tone)]">
          <LockKeyhole :size="16" />
          <div>
            <strong>{{ warning.title }}</strong>
            <span>{{ warning.detail }}</span>
          </div>
        </article>
      </div>
    </section>

    <section class="mini-grid" aria-label="设备资源 mini cards">
      <article v-for="card in miniCards" :key="card.label" :class="['mini-card', toneClass(card.tone)]">
        <component :is="card.icon" :size="18" />
        <div>
          <span>{{ card.label }}</span>
          <strong>{{ card.value }}</strong>
          <small>{{ card.detail }}</small>
        </div>
      </article>
    </section>

    <section class="alert-strip" aria-label="系统提示">
      <article v-for="alert in alerts" :key="alert.title" :class="['alert-pill', toneClass(alert.tone ?? 'blue')]">
        <component :is="alert.icon" :size="15" />
        <div>
          <strong>{{ alert.title }}</strong>
          <span>{{ alert.detail }}</span>
        </div>
      </article>
    </section>
  </aside>
</template>

<style scoped>
.desktop-widgets {
  position: absolute;
  left: 28px;
  top: 78px;
  z-index: 2;
  display: grid;
  width: min(380px, calc(100vw - 420px));
  max-height: calc(100vh - var(--dock-height) - 104px);
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
  overflow-y: auto;
  pointer-events: auto;
  scrollbar-width: none;
}

.desktop-widgets::-webkit-scrollbar {
  display: none;
}

.widget-card,
.mini-card,
.alert-pill {
  border: 1px solid rgba(255, 255, 255, 0.58);
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.82), rgba(241, 248, 255, 0.58)),
    var(--surface-glass);
  box-shadow: var(--shadow-sm);
  backdrop-filter: blur(22px) saturate(150%);
}

.widget-card {
  min-width: 0;
  padding: 14px;
  border-radius: var(--radius-md);
}

.storage-card,
.backup-card,
.security-card,
.mini-grid,
.alert-strip {
  grid-column: 1 / -1;
}

.widget-card__header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 12px;
}

.eyebrow {
  margin: 0 0 4px;
  color: var(--text-soft);
  font-size: 11px;
  font-weight: 800;
  letter-spacing: 0;
}

h2 {
  margin: 0;
  color: var(--text-strong);
  font-size: 15px;
  line-height: 1.25;
}

.header-icon {
  flex: 0 0 auto;
  padding: 5px;
  border-radius: 8px;
  background: rgba(255, 255, 255, 0.66);
}

.pool-list,
.warning-list {
  display: grid;
  gap: 10px;
}

.pool-row {
  padding: 10px;
  border: 1px solid rgba(90, 128, 160, 0.14);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.5);
}

.pool-row--empty {
  border-style: dashed;
}

.pool-row__top,
.pool-row__meta,
.backup-job__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.pool-row__top strong,
.backup-job strong,
.warning-row strong,
.alert-pill strong {
  display: block;
  color: var(--text-strong);
  font-size: 13px;
  line-height: 1.25;
}

.pool-row__top span,
.pool-row__meta,
.backup-job span,
.warning-row span,
.alert-pill span {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.pool-row__top em {
  flex: 0 0 auto;
  color: var(--accent-green);
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
}

.meter {
  height: 6px;
  margin: 9px 0 7px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(120, 151, 178, 0.18);
}

.meter span {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: linear-gradient(90deg, var(--accent-green), var(--accent-cyan));
}

.backup-job {
  position: relative;
  display: grid;
  grid-template-columns: 1fr auto;
  gap: 6px 10px;
  padding: 10px 0;
  border-top: 1px solid rgba(90, 128, 160, 0.14);
}

.backup-job:first-of-type {
  padding-top: 0;
  border-top: 0;
}

.backup-job b {
  color: var(--accent);
  font-size: 18px;
}

.backup-meter {
  grid-column: 1 / -1;
  margin: 2px 0 0;
}

.backup-meter span {
  background: linear-gradient(90deg, var(--accent), #8b5cf6);
}

.backup-job__meta {
  grid-column: 1 / -1;
}

.docker-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
}

.docker-grid div {
  min-width: 0;
  padding: 10px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.52);
}

.docker-grid b {
  display: block;
  color: var(--text-strong);
  font-size: 20px;
  line-height: 1;
}

.docker-grid span {
  display: block;
  margin-top: 6px;
  color: var(--text-muted);
  font-size: 11px;
}

.service-strip {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 10px;
}

.service-strip span {
  padding: 5px 7px;
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  color: var(--text);
  background: rgba(231, 247, 255, 0.72);
  font-size: 11px;
}

.warning-row,
.alert-pill {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 8px;
}

.warning-row {
  padding: 9px;
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.46);
}

.mini-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.mini-card {
  display: flex;
  min-width: 0;
  align-items: flex-start;
  gap: 8px;
  padding: 11px 10px;
  border-radius: var(--radius-md);
}

.mini-card span,
.mini-card small {
  display: block;
  color: var(--text-muted);
  font-size: 10px;
  line-height: 1.25;
}

.mini-card strong {
  display: block;
  margin: 3px 0;
  color: var(--text-strong);
  font-size: 14px;
  line-height: 1.12;
  overflow-wrap: anywhere;
}

.alert-strip {
  display: grid;
  gap: 8px;
}

.alert-pill {
  padding: 9px 10px;
  border-radius: var(--radius-sm);
}

.is-blue {
  color: var(--accent);
}

.is-green {
  color: var(--accent-green);
}

.is-orange {
  color: var(--accent-orange);
}

.is-red {
  color: var(--accent-red);
}

.is-cyan {
  color: var(--accent-cyan);
}

@media (max-width: 1180px) {
  .desktop-widgets {
    width: 320px;
    grid-template-columns: 1fr;
  }

  .docker-card {
    grid-column: 1 / -1;
  }
}

@media (max-height: 840px), (max-width: 980px) {
  .desktop-widgets {
    display: none;
  }
}
</style>
