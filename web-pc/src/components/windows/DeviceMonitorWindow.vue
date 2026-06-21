<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type Component } from 'vue';
import {
  Activity,
  Archive,
  CheckCircle2,
  Cpu,
  Download,
  Fan,
  Gauge,
  HardDrive,
  ListChecks,
  MemoryStick,
  Network,
  ServerCog,
  Thermometer,
} from 'lucide-vue-next';
import { monitoringStore } from '../../stores/monitoring';
import type { Alert, Metric, ServiceStatus, SystemLog } from '../../api/types';
import DeviceMetricsRail from './device/DeviceMetricsRail.vue';
import DeviceMonitorMain from './device/DeviceMonitorMain.vue';
import DeviceMonitorSide from './device/DeviceMonitorSide.vue';
import { UiWindowPage } from '../ui';
import './device/device-window.css';

type MetricKey = string;
type TimeRange = '1H' | '6H' | '24H' | '7D';
type ViewMetric = Metric & { key: string; icon: Component; tone: string };
type ViewService = ServiceStatus & { icon: Component; tone: string };
type ChartPoint = { x: number; y: number; value: number };

const metricIcons: Record<string, Component> = {
  cpu: Cpu,
  memory: MemoryStick,
  network: Network,
  disk: HardDrive,
  temperature: Thermometer,
  fan: Fan,
};

const serviceIcons: Record<string, Component> = {
  collector: Activity,
  uptime: CheckCircle2,
  network: Network,
  rootfs: HardDrive,
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
let dashboardPollingTimer: number | undefined;

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
const chartPoints = computed<ChartPoint[]>(() => buildChartPoints(visibleTrend.value));
const smoothLinePath = computed(() => buildSmoothPath(chartPoints.value));
const smoothAreaPath = computed(() => {
  const points = chartPoints.value;
  const line = smoothLinePath.value;
  if (!points.length || !line) return '';
  return `${line} L ${points[points.length - 1].x} 168 L ${points[0].x} 168 Z`;
});
const trendMax = computed(() => (visibleTrend.value.length ? Math.max(...visibleTrend.value) : 0));
const trendMin = computed(() => (visibleTrend.value.length ? Math.min(...visibleTrend.value) : 0));
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

function buildChartPoints(values: number[]): ChartPoint[] {
  if (!values.length) return [];
  const width = 560;
  const height = 160;
  const paddingX = 18;
  const paddingY = 14;
  const usableWidth = width - paddingX * 2;
  const usableHeight = height - paddingY * 2;
  const max = Math.max(...values, 100);
  const min = Math.min(...values, 0);
  const span = Math.max(1, max - min);
  return values.map((value, index) => ({
    x: paddingX + (values.length === 1 ? usableWidth / 2 : (usableWidth / (values.length - 1)) * index),
    y: paddingY + usableHeight - ((value - min) / span) * usableHeight,
    value,
  }));
}

function buildSmoothPath(points: ChartPoint[]): string {
  if (points.length === 0) return '';
  if (points.length === 1) return `M ${points[0].x} ${points[0].y}`;
  const segments = [`M ${points[0].x} ${points[0].y}`];
  for (let idx = 0; idx < points.length - 1; idx += 1) {
    const current = points[idx];
    const next = points[idx + 1];
    const previous = points[idx - 1] ?? current;
    const after = points[idx + 2] ?? next;
    const cp1x = current.x + (next.x - previous.x) / 6;
    const cp1y = current.y + (next.y - previous.y) / 6;
    const cp2x = next.x - (after.x - current.x) / 6;
    const cp2y = next.y - (after.y - current.y) / 6;
    segments.push(`C ${cp1x} ${cp1y}, ${cp2x} ${cp2y}, ${next.x} ${next.y}`);
  }
  return segments.join(' ');
}

watch(activeRange, () => {
  void monitoringStore.loadMetricTrend(selectedMetric.value.key, activeRange.value);
});

onMounted(async () => {
  await monitoringStore.loadMonitoringDashboard(activeMetricKey.value, activeRange.value);
  activeMetricKey.value = metrics.value[0]?.key ?? activeMetricKey.value;
  selectedLogId.value = systemLogs.value[0]?.id ?? selectedLogId.value;
  selectedAlertId.value = alerts.value[0]?.id ?? selectedAlertId.value;
  dashboardPollingTimer = window.setInterval(() => {
    void monitoringStore.loadMonitoringDashboard(activeMetricKey.value, activeRange.value);
  }, 5000);
});

onUnmounted(() => {
  if (dashboardPollingTimer !== undefined) {
    window.clearInterval(dashboardPollingTimer);
    dashboardPollingTimer = undefined;
  }
});
</script>

<template>
  <UiWindowPage
    layout="dashboard"
    :icon="Gauge"
    title="设备监控"
    subtitle="实时指标、性能趋势、系统日志与告警诊断"
    :status="diagnosticState"
  >
    <template #nav>
      <DeviceMetricsRail
        :metrics="metrics"
        :active-metric-key="activeMetricKey"
        :unresolved-alerts="unresolvedAlerts"
        @select-metric="selectMetric"
      />
    </template>

    <DeviceMonitorMain
      :selected-metric="selectedMetric"
      :selected-metric-ring-value="selectedMetricRingValue"
      :active-range="activeRange"
      :range-keys="Object.keys(trendSeries)"
      :chart-points="chartPoints"
      :smooth-line-path="smoothLinePath"
      :smooth-area-path="smoothAreaPath"
      :trend-min="trendMin"
      :trend-max="trendMax"
      :system-logs="systemLogs"
      :selected-log-id="selectedLogId"
      @update:range="activeRange = $event as TimeRange"
      @refresh-diagnostics="refreshDiagnostics"
      @select-log="selectLog"
    />

    <template #inspector>
      <DeviceMonitorSide
        :diagnostic-state="diagnosticState"
        :selected-log="selectedLog"
        :alerts="alerts"
        :selected-alert-id="selectedAlertId"
        :selected-alert="selectedAlert"
        :service-states="serviceStates"
        @refresh-diagnostics="refreshDiagnostics"
        @create-alert="createAlert"
        @select-alert="selectAlert"
        @mute-alert="muteAlert"
      />
    </template>
  </UiWindowPage>
</template>
