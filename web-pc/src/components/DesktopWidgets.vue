<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type Component } from 'vue';
import {
  Activity,
  ArchiveRestore,
  Bell,
  Bot,
  Boxes,
  CheckCircle2,
  ChevronDown,
  ChevronUp,
  Cpu,
  HardDrive,
  MemoryStick,
  Network,
  Settings2,
  ShieldAlert,
  ShieldCheck,
  UploadCloud,
} from 'lucide-vue-next';
import { apiClient } from '../api/client';
import type { Alert, BackupJob, DockerContainer, Metric, RiskAction, StoragePool } from '../api/types';
import { alerts as seedAlerts, metrics as seedMetrics } from '../data/higoos';

type Tone = 'blue' | 'green' | 'orange' | 'red' | 'cyan';
type WidgetId = 'system' | 'cpu' | 'memory' | 'network' | 'disk' | 'storage' | 'backup' | 'docker' | 'security' | 'alerts';

type OverviewCard = {
  id: WidgetId;
  title: string;
  value: string;
  detail: string;
  tone: Tone;
  icon: Component;
  percent?: number;
};

type SparklinePoint = {
  x: number;
  y: number;
};

type WidgetOption = {
  id: WidgetId;
  label: string;
};

const widgetOptions: WidgetOption[] = [
  { id: 'system', label: '系统状况' },
  { id: 'cpu', label: 'CPU' },
  { id: 'memory', label: '内存' },
  { id: 'network', label: '网速' },
  { id: 'disk', label: '硬盘' },
  { id: 'storage', label: '存储空间' },
  { id: 'backup', label: '备份同步' },
  { id: 'docker', label: 'Docker' },
  { id: 'security', label: '安全' },
  { id: 'alerts', label: '告警' },
];

const defaultVisibility = widgetOptions.reduce<Record<WidgetId, boolean>>((state, option) => {
  state[option.id] = true;
  return state;
}, {} as Record<WidgetId, boolean>);

const fallbackBackupJobs: BackupJob[] = [
  {
    id: 'family-photo',
    name: '家庭相册增量备份',
    source: '照片与视频',
    target: '异地备份卷',
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
    target: '每日快照',
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
    id: 'media-server',
    name: 'media-server',
    image: 'local/media-server:latest',
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
const widgetNotice = ref('正在同步设备状态。');
const expanded = ref(true);
const settingsOpen = ref(false);
const visibleWidgets = ref<Record<WidgetId, boolean>>(loadVisibility());
const chartSamples = ref<Record<string, number[]>>({
  cpu: [],
  memory: [],
  network_down: [],
  network_up: [],
  disk_read: [],
  disk_write: [],
});
let widgetPollingTimer: number | undefined;
const chartSampleLimit = 24;

const cpuMetric = computed(() => findMetric('cpu', 'CPU') ?? metrics.value[0]);
const memoryMetric = computed(() => findMetric('memory', '内存') ?? metrics.value[1]);
const diskMetric = computed(() => findMetric('disk', '磁盘') ?? findMetric('storage', '存储'));
const networkDownText = computed(() => formatMetricValue(findDirectionalMetric(['network_down', 'download', 'down', '下载', '下行']), '88.5 KB/s'));
const networkUpText = computed(() => formatMetricValue(findDirectionalMetric(['network_up', 'upload', 'up', '上传', '上行']), '19.5 KB/s'));
const diskReadText = computed(() => formatMetricValue(findDirectionalMetric(['disk_read', 'read', '读取']), '23.22 KB/s'));
const diskWriteText = computed(() => formatMetricValue(findDirectionalMetric(['disk_write', 'write', '写入']), '12 KB/s'));

const activeBackup = computed(() => backupJobs.value.find((job) => job.state === '同步中' || job.state === '校验中') ?? backupJobs.value[0]);
const primaryPool = computed(() => storagePools.value[0]);

const dockerStatus = computed(() => {
  const containers = dockerContainers.value.length ? dockerContainers.value : fallbackDockerContainers;
  const running = containers.filter((container) => container.status === '运行中').length;
  const cpu = Math.min(100, containers.reduce((sum, container) => sum + container.cpu, 0));
  return {
    running,
    cpu,
    memory: Math.round(containers.reduce((sum, container) => sum + container.memory, 0) / Math.max(containers.length, 1)),
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

const systemCards = computed<OverviewCard[]>(() => [
  {
    id: 'cpu',
    title: 'CPU',
    value: formatMetricValue(cpuMetric.value, '38%'),
    detail: cpuMetric.value?.detail ?? '当前负载',
    tone: 'cyan',
    icon: Cpu,
    percent: percentFromMetric(cpuMetric.value, 38),
  },
  {
    id: 'memory',
    title: '内存',
    value: formatMetricValue(memoryMetric.value, '62%'),
    detail: memoryMetric.value?.detail ?? '系统内存',
    tone: 'green',
    icon: MemoryStick,
    percent: percentFromMetric(memoryMetric.value, 62),
  },
  {
    id: 'network',
    title: '网速',
    value: `↓ ${networkDownText.value}`,
    detail: `↑ ${networkUpText.value}`,
    tone: 'blue',
    icon: Network,
  },
  {
    id: 'disk',
    title: '硬盘',
    value: `R ${diskReadText.value}`,
    detail: `W ${diskWriteText.value}`,
    tone: 'orange',
    icon: HardDrive,
    percent: diskMetric.value ? percentFromMetric(diskMetric.value, 46) : primaryPool.value?.used,
  },
]);

const chartCards = computed(() => visibleSystemCards.value.filter((card) => ['cpu', 'memory', 'network', 'disk'].includes(card.id)));
const summaryCards = computed<OverviewCard[]>(() => [
  {
    id: 'storage',
    title: '存储空间',
    value: primaryPool.value ? `${clampPercent(primaryPool.value.used)}%` : '待同步',
    detail: primaryPool.value ? `${primaryPool.value.name} · ${primaryPool.value.total}` : '等待后端磁盘数据',
    tone: 'green',
    icon: HardDrive,
    percent: primaryPool.value?.used,
  },
  {
    id: 'backup',
    title: '备份同步',
    value: activeBackup.value ? `${activeBackup.value.progress}%` : '待同步',
    detail: activeBackup.value ? activeBackup.value.name : '暂无任务',
    tone: 'blue',
    icon: UploadCloud,
    percent: activeBackup.value?.progress,
  },
  {
    id: 'docker',
    title: 'Docker',
    value: `${dockerStatus.value.running} 个运行`,
    detail: `CPU ${dockerStatus.value.cpu}% · 内存 ${dockerStatus.value.memory}%`,
    tone: 'cyan',
    icon: Boxes,
    percent: dockerStatus.value.cpu,
  },
  {
    id: 'security',
    title: '安全',
    value: `${securityWarnings.value.length} 项`,
    detail: securityWarnings.value[0]?.title ?? '无风险',
    tone: securityWarnings.value.some((item) => item.tone === 'orange' || item.tone === 'red') ? 'orange' : 'green',
    icon: ShieldCheck,
  },
  {
    id: 'alerts',
    title: '告警',
    value: `${alerts.value.length} 条`,
    detail: alerts.value[0]?.title ?? '暂无告警',
    tone: 'orange',
    icon: Bell,
  },
]);

const visibleSystemCards = computed(() => systemCards.value.filter((card) => visibleWidgets.value[card.id]));
const visibleSummaryCards = computed(() => summaryCards.value.filter((card) => visibleWidgets.value[card.id]));

const hasVisibleContent = computed(() => visibleWidgets.value.system || visibleSystemCards.value.length || visibleSummaryCards.value.length);

async function loadWidgetState() {
  try {
    const [nextPools, snapshot, nextAlerts] = await Promise.all([
      apiClient.storage.getPools(),
      apiClient.monitoring.getMetricsSnapshot(),
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
    metrics.value = snapshot.metrics;
    appendChartSamples(snapshot.metrics);
    alerts.value = nextAlerts.map((alert) => ({ ...alert, icon: iconForAlert(alert) }));
    widgetNotice.value = '桌面总览已同步。';
  } catch (error) {
    storagePools.value = [];
    widgetNotice.value = `后端暂不可用，存储池不使用本地演示数据：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

function loadVisibility() {
  if (typeof window === 'undefined') return { ...defaultVisibility };
  try {
    const parsed = JSON.parse(window.localStorage.getItem('higoos.desktopOverview.visibility') ?? '{}') as Partial<
      Record<WidgetId, boolean>
    >;
    return { ...defaultVisibility, ...parsed };
  } catch {
    return { ...defaultVisibility };
  }
}

function updateVisibility(id: WidgetId, event: Event) {
  visibleWidgets.value = {
    ...visibleWidgets.value,
    [id]: (event.target as HTMLInputElement).checked,
  };
}

function resetVisibility() {
  visibleWidgets.value = { ...defaultVisibility };
}

function findMetric(key: string, label: string) {
  const lowerKey = key.toLowerCase();
  return metrics.value.find((metric) => metric.key?.toLowerCase().includes(lowerKey) || metric.label.includes(label));
}

function findDirectionalMetric(terms: string[]) {
  const exact = metrics.value.find((metric) => metric.key && terms.includes(metric.key.toLowerCase()));
  if (exact) return exact;
  return metrics.value.find((metric) => {
    const text = `${metric.key ?? ''} ${metric.label} ${metric.detail ?? ''}`.toLowerCase();
    return terms.some((term) => text.includes(term.toLowerCase()));
  });
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
  return metric.unit ? `${metric.value}${metric.unit}` : String(metric.value);
}

function percentFromMetric(metric: Metric | undefined, fallback: number) {
  if (!metric) return fallback;
  const value = typeof metric.value === 'number' ? metric.value : Number.parseFloat(metric.value);
  return clampPercent(Number.isFinite(value) ? value : fallback);
}

function clampPercent(value: number | undefined) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value ?? 0));
}

function appendChartSamples(nextMetrics: Metric[]) {
  const next: Record<string, number[]> = { ...chartSamples.value };
  for (const key of Object.keys(next)) {
    const value = metricNumber(nextMetrics.find((metric) => metric.key === key));
    const history = next[key] ?? [];
    next[key] = [...history, value].slice(-chartSampleLimit);
  }
  chartSamples.value = next;
}

function metricNumber(metric: Metric | undefined) {
  if (!metric) return 0;
  const value = typeof metric.value === 'number' ? metric.value : Number.parseFloat(metric.value);
  return Number.isFinite(value) ? Math.max(0, value) : 0;
}

function sparklinePointList(card: OverviewCard, lane: 'down' | 'up' = 'down'): SparklinePoint[] {
  const values = chartSeries(card, lane);
  const paired = card.id === 'network' || card.id === 'disk'
    ? [...chartSeries(card, 'down'), ...chartSeries(card, 'up')]
    : values;
  const maxValue = chartMax(card, paired);
  return values
    .map((value, index) => {
      const x = values.length <= 1 ? 0 : (index / (values.length - 1)) * 100;
      const ratio = maxValue > 0 ? Math.min(1, value / maxValue) : 0;
      const y = 74 - ratio * 58;
      return { x, y };
    });
}

function sparklinePath(card: OverviewCard, lane: 'down' | 'up' = 'down') {
  return smoothSparklinePath(sparklinePointList(card, lane));
}

function sparklineAreaPath(card: OverviewCard, lane: 'down' | 'up' = 'down') {
  const points = sparklinePointList(card, lane);
  const path = smoothSparklinePath(points);
  if (!path || points.length === 0) return '';
  const first = points[0];
  const last = points[points.length - 1];
  return `${path} L ${formatChartCoord(last.x)} 80 L ${formatChartCoord(first.x)} 80 Z`;
}

function smoothSparklinePath(points: SparklinePoint[]) {
  if (points.length === 0) return '';
  const [first] = points;
  if (points.length === 1) return `M ${formatChartCoord(first.x)} ${formatChartCoord(first.y)}`;
  const segments = [`M ${formatChartCoord(first.x)} ${formatChartCoord(first.y)}`];

  for (let index = 0; index < points.length - 1; index += 1) {
    const previous = points[index - 1] ?? points[index];
    const current = points[index];
    const next = points[index + 1];
    const after = points[index + 2] ?? next;
    const cp1 = {
      x: current.x + (next.x - previous.x) / 6,
      y: current.y + (next.y - previous.y) / 6,
    };
    const cp2 = {
      x: next.x - (after.x - current.x) / 6,
      y: next.y - (after.y - current.y) / 6,
    };
    segments.push(
      `C ${formatChartCoord(cp1.x)} ${formatChartCoord(cp1.y)} ${formatChartCoord(cp2.x)} ${formatChartCoord(cp2.y)} ${formatChartCoord(next.x)} ${formatChartCoord(next.y)}`,
    );
  }

  return segments.join(' ');
}

function formatChartCoord(value: number) {
  return Number.isInteger(value) ? String(value) : value.toFixed(1);
}

function chartSeries(card: OverviewCard, lane: 'down' | 'up') {
  const key = chartSeriesKey(card, lane);
  const values = chartSamples.value[key] ?? [];
  const fallback = card.percent ?? metricNumber(findDirectionalMetric([key]));
  const series = values.length ? values : [fallback];
  if (series.length >= 8) return series;
  const first = series[0] ?? 0;
  return [...Array.from({ length: 8 - series.length }, () => first), ...series];
}

function chartSeriesKey(card: OverviewCard, lane: 'down' | 'up') {
  if (card.id === 'memory') return 'memory';
  if (card.id === 'network') return lane === 'up' ? 'network_up' : 'network_down';
  if (card.id === 'disk') return lane === 'up' ? 'disk_write' : 'disk_read';
  return 'cpu';
}

function chartMax(card: OverviewCard, values: number[]) {
  if (card.id === 'cpu' || card.id === 'memory') return 100;
  return Math.max(...values, 0.01);
}

function iconForAlert(alert: Alert): Component {
  const text = `${alert.title} ${alert.detail}`;
  if (text.includes('备份')) return ArchiveRestore;
  if (text.includes('模型')) return Bot;
  if (text.includes('权限') || text.includes('审计')) return ShieldCheck;
  if (text.includes('CPU')) return Cpu;
  if (text.includes('硬盘') || text.includes('存储')) return HardDrive;
  if (alert.tone === 'green') return CheckCircle2;
  return Bell;
}

watch(
  visibleWidgets,
  (next) => {
    if (typeof window !== 'undefined') {
      window.localStorage.setItem('higoos.desktopOverview.visibility', JSON.stringify(next));
    }
  },
  { deep: true },
);

onMounted(() => {
  void loadWidgetState();
  widgetPollingTimer = window.setInterval(() => {
    void loadWidgetState();
  }, 5000);
});

onUnmounted(() => {
  if (widgetPollingTimer !== undefined) {
    window.clearInterval(widgetPollingTimer);
    widgetPollingTimer = undefined;
  }
});
</script>

<template>
  <aside class="desktop-overview" :class="{ 'desktop-overview--collapsed': !expanded }" aria-label="桌面总览">
    <button v-if="!expanded" class="overview-collapsed" type="button" @click="expanded = true">
      <Activity :size="18" />
      <span>总览</span>
      <ChevronUp :size="16" />
    </button>

    <template v-else>
      <header class="overview-header">
        <div>
          <strong>总览</strong>
          <span>{{ widgetNotice }}</span>
        </div>
        <div class="overview-header__actions">
          <button type="button" title="自定义卡片" @click="settingsOpen = !settingsOpen">
            <Settings2 :size="16" />
          </button>
          <button type="button" title="收起总览" @click="expanded = false">
            <ChevronDown :size="16" />
          </button>
        </div>
      </header>

      <section v-if="settingsOpen" class="overview-settings" aria-label="总览显示设置">
        <div class="overview-settings__head">
          <strong>显示卡片</strong>
          <button type="button" @click="resetVisibility">全部显示</button>
        </div>
        <label v-for="option in widgetOptions" :key="option.id">
          <input
            type="checkbox"
            :checked="visibleWidgets[option.id]"
            @change="updateVisibility(option.id, $event)"
          />
          <span>{{ option.label }}</span>
        </label>
      </section>

      <section v-if="visibleWidgets.system" class="overview-system" aria-label="系统状况">
        <div>
          <span>系统状况</span>
          <strong>HiGoNAS</strong>
          <small>{{ hasVisibleContent ? '设备运行正常' : '已隐藏全部卡片' }}</small>
        </div>
        <div class="overview-rings">
          <article v-for="card in visibleSystemCards.slice(0, 4)" :key="card.id" :class="toneClass(card.tone)">
            <span class="ring" :style="{ '--value': `${clampPercent(card.percent ?? 0) * 3.6}deg` }">
              <component :is="card.icon" :size="14" />
            </span>
            <strong>{{ card.value }}</strong>
            <small>{{ card.title }}</small>
          </article>
        </div>
      </section>

      <section class="overview-chart-list" aria-label="资源图表">
        <article
          v-for="card in chartCards"
          :key="card.id"
          :class="['overview-chart-card', toneClass(card.tone), { 'overview-chart-card--dual': card.id === 'network' || card.id === 'disk' }]"
        >
          <div class="overview-chart-card__head">
            <div>
              <component :is="card.icon" :size="17" />
              <strong>{{ card.title }}</strong>
            </div>
            <p v-if="card.id === 'network'">
              <b>↓ {{ networkDownText }}</b>
              <b>↑ {{ networkUpText }}</b>
            </p>
            <p v-else-if="card.id === 'disk'">
              <b>R {{ diskReadText }}</b>
              <b>W {{ diskWriteText }}</b>
            </p>
            <p v-else>
              <b>{{ card.value }}</b>
              <span>{{ card.detail }}</span>
            </p>
          </div>
          <div class="overview-sparkline" aria-hidden="true">
            <svg viewBox="0 0 100 80" preserveAspectRatio="none">
              <path class="spark-area spark-area--down" :d="sparklineAreaPath(card, 'down')" />
              <path class="spark-line spark-line--down" :d="sparklinePath(card, 'down')" />
              <template v-if="card.id === 'network' || card.id === 'disk'">
                <path class="spark-area spark-area--up" :d="sparklineAreaPath(card, 'up')" />
                <path class="spark-line spark-line--up" :d="sparklinePath(card, 'up')" />
              </template>
            </svg>
          </div>
        </article>
      </section>

      <section class="overview-info-list" aria-label="其他总览卡片">
        <article v-for="card in visibleSummaryCards" :key="card.id" :class="['overview-info-row', toneClass(card.tone)]">
          <component :is="card.icon" :size="17" />
          <div>
            <strong>{{ card.title }}</strong>
            <span>{{ card.detail }}</span>
          </div>
          <b>{{ card.value }}</b>
        </article>
      </section>
    </template>
  </aside>
</template>

<style scoped>
.desktop-overview {
  position: fixed;
  right: 14px;
  bottom: 14px;
  z-index: 8;
  width: min(420px, calc(100vw - 28px));
  height: min(68vh, calc(100vh - var(--topbar-height, 64px) - 24px));
  max-height: calc(100vh - var(--topbar-height, 64px) - 24px);
  padding: 10px;
  overflow: auto;
  pointer-events: auto;
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: 22px;
  background:
    linear-gradient(145deg, rgba(255, 255, 255, 0.74), rgba(232, 246, 255, 0.52)),
    var(--surface-glass);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(24px) saturate(155%);
}

.desktop-overview--collapsed {
  width: auto;
  height: auto;
  max-height: none;
  min-height: 0;
  padding: 0;
  overflow: visible;
  background: transparent;
  border: 0;
  box-shadow: none;
  backdrop-filter: none;
}

.overview-collapsed,
.overview-header__actions button,
.overview-settings__head button {
  font-family: inherit;
}

.overview-collapsed {
  display: inline-flex;
  align-items: center;
  gap: 7px;
  min-height: 40px;
  padding: 0 14px;
  color: var(--text-strong);
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(255, 255, 255, 0.66);
  border-radius: 999px;
  box-shadow: var(--shadow-sm);
  font-size: 12px;
  font-weight: 800;
}

.overview-header {
  position: sticky;
  top: 0;
  z-index: 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 42px;
  padding: 0 4px 8px;
  background: linear-gradient(180deg, rgba(244, 251, 255, 0.95), rgba(244, 251, 255, 0.78));
  backdrop-filter: blur(14px);
}

.overview-header strong,
.overview-system strong,
.overview-settings__head strong,
.overview-chart-card strong,
.overview-info-row strong,
.overview-rings strong {
  color: var(--text-strong);
}

.overview-header strong {
  display: block;
  font-size: 15px;
}

.overview-header span {
  display: block;
  margin-top: 2px;
  color: var(--text-soft);
  font-size: 11px;
}

.overview-header__actions {
  display: flex;
  gap: 6px;
}

.overview-header__actions button {
  display: grid;
  width: 32px;
  height: 32px;
  place-items: center;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
}

.overview-settings,
.overview-system,
.overview-chart-card,
.overview-info-row {
  background: rgba(255, 255, 255, 0.66);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-md);
}

.overview-settings {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin-bottom: 10px;
  padding: 10px;
}

.overview-settings__head {
  grid-column: 1 / -1;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.overview-settings__head button {
  color: var(--accent);
  background: rgba(231, 247, 255, 0.8);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.overview-settings label {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 720;
}

.overview-system {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 12px;
  margin-bottom: 10px;
  padding: 14px;
}

.overview-system span,
.overview-system small,
.overview-chart-card span,
.overview-chart-card small,
.overview-info-row span,
.overview-rings small {
  color: var(--text-muted);
}

.overview-system span,
.overview-chart-card span {
  display: block;
  font-size: 11px;
  font-weight: 760;
}

.overview-system strong {
  display: block;
  margin: 5px 0 3px;
  font-size: 17px;
}

.overview-system small {
  font-size: 11px;
}

.overview-rings {
  display: flex;
  flex-wrap: wrap;
  justify-content: end;
  gap: 8px;
}

.overview-rings article {
  display: grid;
  justify-items: center;
  gap: 3px;
  min-width: 48px;
}

.ring {
  position: relative;
  display: grid;
  width: 38px;
  height: 38px;
  place-items: center;
  color: currentColor;
  background:
    radial-gradient(circle at center, rgba(255, 255, 255, 0.95) 0 58%, transparent 59%),
    conic-gradient(var(--overview-color, currentColor) var(--value), rgba(148, 163, 184, 0.18) 0);
  border-radius: 999px;
}

.ring svg {
  position: absolute;
  inset: 50% auto auto 50%;
  transform: translate(-50%, -50%);
  color: var(--overview-color, currentColor);
  stroke-width: 2.4;
}

.overview-rings strong {
  font-size: 11px;
}

.overview-rings small {
  font-size: 10px;
}

.overview-chart-list,
.overview-info-list {
  display: grid;
  gap: 10px;
}

.overview-chart-list {
  grid-template-columns: 1fr;
  margin-bottom: 10px;
}

.overview-chart-card {
  display: grid;
  grid-template-columns: 1fr;
  gap: 10px;
  min-width: 0;
  min-height: 172px;
  padding: 14px;
}

.overview-chart-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
}

.overview-chart-card__head > div {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.overview-chart-card__head strong {
  display: block;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 15px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-chart-card__head p {
  display: grid;
  justify-items: end;
  gap: 3px;
  margin: 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.2;
  white-space: nowrap;
}

.overview-chart-card__head p b {
  color: var(--overview-color, var(--accent));
  font-size: 14px;
}

.overview-chart-card__head p span {
  color: var(--text-muted);
}

.overview-sparkline {
  height: 108px;
  padding: 0;
  overflow: hidden;
  background: transparent;
  border: 0;
  border-radius: 12px;
}

.overview-sparkline svg {
  display: block;
  width: 100%;
  height: 100%;
}

.spark-area {
  opacity: 0.22;
}

.spark-line {
  fill: none;
  stroke-width: 1.05;
  stroke-linecap: round;
  stroke-linejoin: round;
}

.spark-area--down {
  fill: #3b82f6;
}

.spark-line--down {
  stroke: #3b82f6;
}

.spark-area--up {
  fill: #22b573;
}

.spark-line--up {
  stroke: #22b573;
}

.overview-chart-card:not(.overview-chart-card--dual) .spark-area--down {
  fill: var(--overview-color, #3b82f6);
}

.overview-chart-card:not(.overview-chart-card--dual) .spark-line--down {
  stroke: var(--overview-color, #3b82f6);
}

.overview-info-list {
  grid-template-columns: 1fr;
  padding-bottom: 2px;
}

.overview-info-row {
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  align-items: center;
  gap: 9px;
  min-height: 54px;
  padding: 10px 12px;
}

.overview-info-row strong,
.overview-info-row span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.overview-info-row strong {
  font-size: 12px;
}

.overview-info-row span {
  margin-top: 3px;
  font-size: 10px;
}

.overview-info-row b {
  color: var(--text-strong);
  font-size: 13px;
  white-space: nowrap;
}

.is-blue {
  color: var(--accent);
  --overview-color: var(--accent);
}

.is-green {
  color: var(--accent-green);
  --overview-color: var(--accent-green);
}

.is-orange {
  color: var(--accent-orange);
  --overview-color: var(--accent-orange);
}

.is-red {
  color: var(--accent-red);
  --overview-color: var(--accent-red);
}

.is-cyan {
  color: var(--accent-cyan);
  --overview-color: var(--accent-cyan);
}

@media (max-width: 860px) {
  .desktop-overview {
    right: 14px;
    bottom: 14px;
    width: min(340px, calc(100vw - 28px));
    height: min(66vh, calc(100vh - var(--topbar-height, 64px) - 24px));
    max-height: calc(100vh - var(--topbar-height, 64px) - 24px);
  }

  .overview-settings {
    grid-template-columns: 1fr;
  }

  .overview-chart-card {
    min-height: 152px;
  }

  .overview-system {
    grid-template-columns: 1fr;
  }

  .overview-rings {
    justify-content: start;
  }
}
</style>
