<script setup lang="ts">
import { computed, onMounted, ref, watch, type Component } from 'vue';
import {
  Activity,
  AlertTriangle,
  Archive,
  Bell,
  CheckCircle2,
  Cpu,
  Download,
  Fan,
  Gauge,
  HardDrive,
  LineChart,
  ListChecks,
  MemoryStick,
  Network,
  Plus,
  RefreshCw,
  ServerCog,
  Thermometer,
  Volume2,
  VolumeX,
} from 'lucide-vue-next';
import { monitoringStore } from '../../stores/monitoring';
import type { Alert, Metric, ServiceStatus, SystemLog } from '../../api/types';

type MetricKey = string;
type TimeRange = '1H' | '6H' | '24H' | '7D';
type ViewMetric = Metric & { key: string; icon: Component; tone: string };
type ViewService = ServiceStatus & { icon: Component; tone: string };

const metricIcons: Record<string, Component> = {
  cpu: Cpu,
  memory: MemoryStick,
  network: Network,
  disk: HardDrive,
  temperature: Thermometer,
  fan: Fan,
};

const serviceIcons: Record<string, Component> = {
  containers: ServerCog,
  apps: Gauge,
  tasks: ListChecks,
  backups: Archive,
  downloads: Download,
};

const fallbackMetrics: ViewMetric[] = [
  {
    key: 'cpu',
    label: 'CPU',
    value: 38,
    unit: '%',
    detail: '4C / 8T · 2.8GHz boost',
    icon: Cpu,
    tone: 'green',
  },
  {
    key: 'memory',
    label: '内存',
    value: 62,
    unit: '%',
    detail: '19.8GB / 32GB · ZFS ARC 8.4GB',
    icon: MemoryStick,
    tone: 'blue',
  },
  {
    key: 'network',
    label: '网络',
    value: 71,
    unit: '%',
    detail: '2.3Gbps 下行 · 840Mbps 上行',
    icon: Network,
    tone: 'blue',
  },
  {
    key: 'disk',
    label: '磁盘',
    value: 46,
    unit: '%',
    detail: '主机卷 I/O · 812MB/s',
    icon: HardDrive,
    tone: 'green',
  },
  {
    key: 'temperature',
    label: '温度',
    value: 43,
    unit: '°C',
    detail: 'CPU 43°C · 硬盘均值 36°C',
    icon: Thermometer,
    tone: 'orange',
  },
  {
    key: 'fan',
    label: '风扇',
    value: 1280,
    unit: 'RPM',
    detail: '静音曲线 · 双风扇同步',
    icon: Fan,
    tone: 'green',
  },
] as const satisfies ViewMetric[];

const fallbackServiceStates: ViewService[] = [
  { label: '容器', value: '18 / 20', detail: 'Plex 转码容器限速中', icon: ServerCog, tone: 'orange' },
  { label: '应用', value: '42', detail: '2 个应用等待更新', icon: Gauge, tone: 'blue' },
  { label: '任务', value: '7', detail: '照片识别队列运行中', icon: ListChecks, tone: 'green' },
  { label: '备份', value: '3', detail: 'MacBook Pro 增量备份 82%', icon: Archive, tone: 'green' },
  { label: '下载', value: '11', detail: '2 个任务因低速排队', icon: Download, tone: 'orange' },
] as const satisfies ViewService[];

const trendSeries: Record<TimeRange, number[]> = {
  '1H': [34, 38, 41, 45, 43, 39, 42, 50, 47, 44, 40, 38],
  '6H': [28, 33, 39, 55, 48, 52, 61, 58, 46, 43, 49, 44],
  '24H': [31, 44, 39, 36, 58, 64, 52, 47, 42, 56, 62, 49],
  '7D': [26, 38, 45, 41, 53, 69, 57, 51, 48, 59, 63, 54],
};

const fallbackLogs: SystemLog[] = [
  {
    id: 'log-1',
    level: 'info',
    source: '备份中心',
    message: 'Time Machine 增量备份已校验 82%',
    at: '14:26',
  },
  {
    id: 'log-2',
    level: 'warn',
    source: '容器运行时',
    message: 'plex-transcoder CPU 峰值持续 6 分钟',
    at: '14:19',
  },
  {
    id: 'log-3',
    level: 'info',
    source: '下载服务',
    message: '下载任务已切换到夜间限速策略',
    at: '14:08',
  },
  {
    id: 'log-4',
    level: 'warn',
    source: '硬盘健康',
    message: '槽位 4 温度高于 38°C，风扇曲线已提升',
    at: '13:56',
  },
];

const fallbackAlerts: Alert[] = [
  {
    id: 'alert-1',
    severity: '中风险',
    title: '容器 CPU 峰值',
    source: 'plex-transcoder',
    detail: '过去 10 分钟 CPU 平均 78%，建议限制转码并发。',
    muted: false,
    state: '待处理',
  },
  {
    id: 'alert-2',
    severity: '低风险',
    title: '下载任务低速',
    source: '下载中心',
    detail: '2 个任务低于 300KB/s，已等待下一轮自动重试。',
    muted: false,
    state: '观察中',
  },
];

const activeMetricKey = ref<MetricKey>('cpu');
const selectedLogId = ref(fallbackLogs[0].id);
const selectedAlertId = ref(fallbackAlerts[0].id);
const activeRange = ref<TimeRange>('1H');
const diagnosticCount = ref(0);

const metrics = computed<ViewMetric[]>(() => {
  const next = monitoringStore.metrics.value.flatMap((metric) => {
    if (!metric.key) return [];
    const key = metric.key;
    return [{
      ...metric,
      key,
      icon: metricIcons[key] ?? Cpu,
      tone: metric.tone ?? 'blue',
    }];
  });
  return next.length ? next : fallbackMetrics;
});

const serviceStates = computed<ViewService[]>(() => {
  const next = monitoringStore.services.value.map((service) => ({
    ...service,
    icon: serviceIcons[service.key ?? ''] ?? ServerCog,
    tone: service.tone ?? 'blue',
  }));
  return next.length ? next : fallbackServiceStates;
});

const systemLogs = computed(() => (monitoringStore.logs.value.length ? monitoringStore.logs.value : fallbackLogs));
const alerts = computed(() => (monitoringStore.alerts.value.length ? monitoringStore.alerts.value : fallbackAlerts));
const selectedMetric = computed(() => metrics.value.find((metric) => metric.key === activeMetricKey.value) ?? metrics.value[0]);
const selectedLog = computed(() => systemLogs.value.find((log) => log.id === selectedLogId.value) ?? systemLogs.value[0]);
const selectedAlert = computed(() => alerts.value.find((alert) => alert.id === selectedAlertId.value) ?? alerts.value[0]);
const visibleTrend = computed(() => {
  if (monitoringStore.trendPoints.value.length) {
    return monitoringStore.trendPoints.value.map((point) => normalizeTrendPoint(point.value, selectedMetric.value.key));
  }
  const scale = selectedMetric.value.key === 'temperature' ? 0.72 : selectedMetric.value.key === 'fan' ? 0.05 : 1;
  return trendSeries[activeRange.value].map((value) => Math.max(14, Math.min(96, Math.round(value * scale))));
});
const unresolvedAlerts = computed(() => alerts.value.filter((alert) => !alert.muted).length);
const diagnosticState = computed(() =>
  monitoringStore.diagnostic.value?.message ??
  monitoringStore.diagnostic.value?.summary ??
  (diagnosticCount.value === 0
    ? '最近诊断：全量巡检于 14:00 完成'
    : `刷新诊断 #${diagnosticCount.value}：指标、日志和风扇曲线已重新采样`),
);
const selectedMetricRingValue = computed(() => {
  const value = Number(selectedMetric.value.value);
  return Number.isFinite(value) ? Math.min(value, 100) : 0;
});

function selectMetric(key: MetricKey) {
  activeMetricKey.value = key;
  void monitoringStore.loadMetricTrend(key, activeRange.value);
}

function selectLog(id: string) {
  selectedLogId.value = id;
}

function selectAlert(id: string) {
  selectedAlertId.value = id;
}

async function createAlert() {
  try {
    const alert = await monitoringStore.createAlert(selectedMetric.value.key, activeRange.value);
    selectedAlertId.value = alert.id ?? selectedAlertId.value;
  } catch {
    const id = `alert-${Date.now()}`;
    selectedAlertId.value = id;
    diagnosticCount.value += 1;
  }
}

async function muteAlert() {
  if (!selectedAlert.value) return;
  if (!selectedAlert.value.id || monitoringStore.usingFallback.value) return;
  const nextMuted = !selectedAlert.value.muted;
  try {
    const alert = await monitoringStore.muteAlert(selectedAlert.value.id, nextMuted);
    selectedAlertId.value = alert.id ?? selectedAlertId.value;
  } catch {
    diagnosticCount.value += 1;
  }
}

async function refreshDiagnostics() {
  diagnosticCount.value += 1;
  try {
    await monitoringStore.runDiagnostics();
    await monitoringStore.loadMonitoringDashboard(selectedMetric.value.key, activeRange.value);
    selectedLogId.value = systemLogs.value[0]?.id ?? selectedLogId.value;
  } catch {
    selectedLogId.value = fallbackLogs[0].id;
  }
}

function normalizeTrendPoint(value: number, metric: string) {
  if (metric === 'fan') return Math.max(14, Math.min(96, Math.round((value / 2200) * 100)));
  return Math.max(14, Math.min(96, Math.round(value)));
}

watch(activeRange, () => {
  void monitoringStore.loadMetricTrend(selectedMetric.value.key, activeRange.value);
});

onMounted(async () => {
  await monitoringStore.loadMonitoringDashboard(activeMetricKey.value, activeRange.value);
  activeMetricKey.value = metrics.value[0]?.key ?? activeMetricKey.value;
  selectedLogId.value = systemLogs.value[0]?.id ?? selectedLogId.value;
  selectedAlertId.value = alerts.value[0]?.id ?? selectedAlertId.value;
});
</script>

<template>
  <div class="device-monitor">
    <section class="device-monitor__metrics" aria-label="设备核心指标">
      <header>
        <h3><Activity :size="15" /> 核心指标</h3>
        <span>{{ unresolvedAlerts }} 条活跃告警</span>
      </header>

      <button
        v-for="metric in metrics"
        :key="metric.key"
        class="device-monitor__metric"
        :class="[
          `device-monitor__metric--${metric.tone}`,
          { 'device-monitor__metric--active': metric.key === activeMetricKey },
        ]"
        type="button"
        @click="selectMetric(metric.key)"
      >
        <component :is="metric.icon" :size="17" />
        <span>
          <strong>{{ metric.label }}</strong>
          <small>{{ metric.detail }}</small>
        </span>
        <b>{{ metric.value }}{{ metric.unit }}</b>
      </button>
    </section>

    <main class="device-monitor__main">
      <section class="device-monitor__hero" aria-label="当前指标详情">
        <div>
          <p>{{ selectedMetric.label }} 当前负载</p>
          <strong>{{ selectedMetric.value }}{{ selectedMetric.unit }}</strong>
          <span>{{ selectedMetric.detail }}</span>
        </div>
        <div class="device-monitor__ring" :style="{ '--value': `${selectedMetricRingValue * 3.6}deg` }">
          <component :is="selectedMetric.icon" :size="26" />
        </div>
      </section>

      <section class="device-monitor__trend" aria-label="性能趋势">
        <header>
          <h3><LineChart :size="15" /> 性能趋势</h3>
          <div class="device-monitor__range">
            <button
              v-for="range in Object.keys(trendSeries)"
              :key="range"
              :class="{ 'device-monitor__range-button--active': range === activeRange }"
              type="button"
              @click="activeRange = range as TimeRange"
            >
              {{ range }}
            </button>
          </div>
        </header>
        <div class="device-monitor__bars">
          <span v-for="(point, index) in visibleTrend" :key="`${activeRange}-${index}`" :style="{ height: `${point}%` }" />
        </div>
      </section>

      <section class="device-monitor__services" aria-label="容器、应用、任务、备份和下载状态">
        <article v-for="service in serviceStates" :key="service.label" :class="`device-monitor__service--${service.tone}`">
          <component :is="service.icon" :size="16" />
          <div>
            <strong>{{ service.label }} · {{ service.value }}</strong>
            <span>{{ service.detail }}</span>
          </div>
        </article>
      </section>

      <section class="device-monitor__logs" aria-label="系统日志">
        <header>
          <h3>系统日志</h3>
          <button type="button" @click="refreshDiagnostics"><RefreshCw :size="14" /> 刷新诊断</button>
        </header>
        <div class="device-monitor__log-grid">
          <button
            v-for="log in systemLogs"
            :key="log.id"
            class="device-monitor__log"
            :class="{ 'device-monitor__log--active': log.id === selectedLogId }"
            type="button"
            @click="selectLog(log.id)"
          >
            <span>{{ log.at }}</span>
            <strong>{{ log.source }}</strong>
            <small>{{ log.message }}</small>
          </button>
        </div>
      </section>
    </main>

    <aside class="device-monitor__side" aria-label="告警和诊断">
      <section class="device-monitor__diagnostic">
        <header>
          <h3><CheckCircle2 :size="15" /> 诊断</h3>
          <button type="button" @click="refreshDiagnostics"><RefreshCw :size="13" /> 重跑</button>
        </header>
        <p>{{ diagnosticState }}</p>
        <div>
          <span>选中日志</span>
          <strong>{{ selectedLog.source }}</strong>
          <small>{{ selectedLog.message }}</small>
        </div>
      </section>

      <section class="device-monitor__alerts">
        <header>
          <h3><Bell :size="15" /> 告警</h3>
          <button type="button" @click="createAlert"><Plus :size="13" /> 创建</button>
        </header>
        <button
          v-for="alert in alerts"
          :key="alert.id ?? alert.title"
          class="device-monitor__alert"
          :class="{ 'device-monitor__alert--active': alert.id === selectedAlertId, 'device-monitor__alert--muted': alert.muted }"
          type="button"
          @click="alert.id && selectAlert(alert.id)"
        >
          <AlertTriangle :size="15" />
          <span>
            <strong>{{ alert.title }}</strong>
            <small>{{ alert.severity }} · {{ alert.state }}</small>
          </span>
        </button>

        <div class="device-monitor__alert-detail">
          <strong>{{ selectedAlert.title }}</strong>
          <p>{{ selectedAlert.detail }}</p>
          <button type="button" @click="muteAlert">
            <component :is="selectedAlert.muted ? Volume2 : VolumeX" :size="14" />
            {{ selectedAlert.muted ? '恢复提醒' : '静音告警' }}
          </button>
        </div>
      </section>
    </aside>
  </div>
</template>

<style scoped>
.device-monitor {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr) 220px;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.device-monitor__metrics,
.device-monitor__main,
.device-monitor__side,
.device-monitor__hero,
.device-monitor__trend,
.device-monitor__services,
.device-monitor__logs,
.device-monitor__diagnostic,
.device-monitor__alerts {
  min-width: 0;
  min-height: 0;
}

.device-monitor__metrics,
.device-monitor__hero,
.device-monitor__trend,
.device-monitor__logs,
.device-monitor__diagnostic,
.device-monitor__alerts {
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.device-monitor header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 11px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.device-monitor h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor header span {
  color: var(--text-soft);
  font-size: 11px;
}

.device-monitor button {
  font-family: inherit;
}

.device-monitor__metrics {
  display: grid;
  grid-template-rows: auto repeat(6, minmax(0, 1fr));
  overflow: hidden;
}

.device-monitor__metric {
  display: grid;
  grid-template-columns: 22px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 9px 10px;
  color: var(--accent);
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.device-monitor__metric--active {
  background: rgba(19, 136, 255, 0.09);
  box-shadow: inset 3px 0 0 var(--accent);
}

.device-monitor__metric--green {
  color: var(--accent-green);
}

.device-monitor__metric--orange {
  color: #b36a00;
}

.device-monitor__metric strong,
.device-monitor__metric small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-monitor__metric strong {
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor__metric small {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
}

.device-monitor__metric b {
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor__main {
  display: grid;
  grid-template-rows: auto 150px auto minmax(0, 1fr);
  gap: 12px;
}

.device-monitor__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 96px;
  padding: 14px 16px;
  background: linear-gradient(135deg, rgba(231, 247, 255, 0.92), rgba(255, 246, 227, 0.78));
}

.device-monitor__hero p,
.device-monitor__hero strong,
.device-monitor__hero span {
  display: block;
}

.device-monitor__hero p {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 700;
}

.device-monitor__hero strong {
  margin-top: 4px;
  color: var(--text-strong);
  font-size: 28px;
}

.device-monitor__hero span {
  margin-top: 5px;
  color: var(--text-muted);
  font-size: 11px;
}

.device-monitor__ring {
  display: grid;
  width: 68px;
  height: 68px;
  flex: 0 0 68px;
  place-items: center;
  color: var(--accent);
  background:
    radial-gradient(circle at center, rgba(255, 255, 255, 0.94) 0 54%, transparent 55%),
    conic-gradient(var(--accent) var(--value), rgba(148, 163, 184, 0.2) 0);
  border-radius: 999px;
}

.device-monitor__trend {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.device-monitor__range {
  display: flex;
  gap: 4px;
}

.device-monitor__range button {
  height: 24px;
  padding: 0 7px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 760;
}

.device-monitor__range .device-monitor__range-button--active {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-color: rgba(19, 136, 255, 0.18);
}

.device-monitor__bars {
  display: flex;
  align-items: end;
  gap: 7px;
  height: 100%;
  padding: 13px;
}

.device-monitor__bars span {
  width: 100%;
  min-width: 8px;
  background: linear-gradient(180deg, var(--accent-cyan), var(--accent));
  border-radius: 999px 999px 3px 3px;
}

.device-monitor__services {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
}

.device-monitor__services article {
  display: flex;
  gap: 8px;
  min-width: 0;
  padding: 10px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.52);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.device-monitor__service--green {
  color: var(--accent-green);
}

.device-monitor__service--orange {
  color: #b36a00;
}

.device-monitor__services strong,
.device-monitor__services span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-monitor__services strong {
  color: var(--text-strong);
  font-size: 11px;
}

.device-monitor__services span {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 10px;
}

.device-monitor__logs {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.device-monitor__logs header button,
.device-monitor__diagnostic header button,
.device-monitor__alerts header button,
.device-monitor__alert-detail button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
  white-space: nowrap;
}

.device-monitor__log-grid {
  display: grid;
  gap: 7px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.device-monitor__log {
  display: grid;
  grid-template-columns: 42px 82px minmax(0, 1fr);
  gap: 7px;
  align-items: center;
  min-height: 34px;
  padding: 7px 8px;
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.device-monitor__log--active {
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.device-monitor__log span,
.device-monitor__log small {
  color: var(--text-muted);
  font-size: 10px;
}

.device-monitor__log strong,
.device-monitor__log small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.device-monitor__log strong {
  color: var(--text-strong);
  font-size: 11px;
}

.device-monitor__side {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 12px;
}

.device-monitor__diagnostic {
  overflow: hidden;
}

.device-monitor__diagnostic p,
.device-monitor__diagnostic div {
  margin: 0;
  padding: 11px;
}

.device-monitor__diagnostic p {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.42;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.device-monitor__diagnostic span,
.device-monitor__diagnostic strong,
.device-monitor__diagnostic small {
  display: block;
}

.device-monitor__diagnostic span,
.device-monitor__diagnostic small {
  color: var(--text-muted);
  font-size: 10px;
  line-height: 1.35;
}

.device-monitor__diagnostic strong {
  margin: 4px 0;
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor__alerts {
  display: grid;
  grid-template-rows: auto auto auto minmax(0, 1fr);
  align-content: start;
  overflow: auto;
}

.device-monitor__alert {
  display: flex;
  gap: 8px;
  align-items: center;
  min-height: 46px;
  padding: 9px 10px;
  color: #b36a00;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.device-monitor__alert--active {
  background: rgba(245, 158, 11, 0.1);
  box-shadow: inset 3px 0 0 #f59e0b;
}

.device-monitor__alert--muted {
  color: var(--text-soft);
  opacity: 0.72;
}

.device-monitor__alert strong,
.device-monitor__alert small {
  display: block;
}

.device-monitor__alert strong {
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor__alert small {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
}

.device-monitor__alert-detail {
  padding: 12px;
}

.device-monitor__alert-detail strong {
  color: var(--text-strong);
  font-size: 12px;
}

.device-monitor__alert-detail p {
  margin: 6px 0 10px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.42;
}

@media (max-width: 760px) {
  .device-monitor {
    display: block;
    overflow: auto;
  }

  .device-monitor__metrics,
  .device-monitor__main,
  .device-monitor__side {
    margin-bottom: 12px;
  }

  .device-monitor__metrics {
    grid-template-rows: auto;
  }

  .device-monitor__main,
  .device-monitor__side {
    display: grid;
  }

  .device-monitor__main {
    grid-template-rows: auto 140px auto minmax(220px, auto);
  }

  .device-monitor__services {
    grid-template-columns: 1fr;
  }

  .device-monitor__log {
    grid-template-columns: 40px minmax(70px, 0.4fr) minmax(0, 1fr);
  }
}
</style>
