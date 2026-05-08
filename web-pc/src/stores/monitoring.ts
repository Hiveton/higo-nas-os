import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { Alert, DiagnosticResult, Metric, ServiceStatus, SystemLog, TrendPoint } from '../api/types';

const metrics = ref<Metric[]>([]);
const services = ref<ServiceStatus[]>([
  { key: 'containers', label: '容器', value: '18 / 20', detail: 'Plex 转码容器限速中', tone: 'orange' },
  { key: 'apps', label: '应用', value: '42', detail: '2 个应用等待更新', tone: 'blue' },
  { key: 'tasks', label: '任务', value: '7', detail: '照片识别队列运行中', tone: 'green' },
  { key: 'backups', label: '备份', value: '3', detail: 'MacBook Pro 增量备份 82%', tone: 'green' },
  { key: 'downloads', label: '下载', value: '11', detail: '2 个任务因低速排队', tone: 'orange' },
]);
const logs = ref<SystemLog[]>([]);
const alerts = ref<Alert[]>([]);
const trendPoints = ref<TrendPoint[]>([]);
const diagnostic = ref<DiagnosticResult | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

export const monitoringStore = {
  metrics: readonly(metrics),
  services: readonly(services),
  logs: readonly(logs),
  alerts: readonly(alerts),
  trendPoints: readonly(trendPoints),
  diagnostic: readonly(diagnostic),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  loadMonitoringSnapshot,
  loadMonitoringDashboard,
  loadMetricTrend,
  createAlert,
  muteAlert,
  runDiagnostics,
};

export async function loadMonitoringSnapshot() {
  loading.value = true;
  error.value = null;

  try {
    const [nextMetrics, nextAlerts] = await Promise.all([
      apiClient.monitoring.getCurrentMetrics(),
      apiClient.monitoring.getAlerts(),
    ]);
    metrics.value = nextMetrics;
    alerts.value = nextAlerts;
    usingFallback.value = false;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
  } finally {
    loading.value = false;
  }
}

export async function loadMonitoringDashboard(metric = 'cpu', range = '1H') {
  loading.value = true;
  error.value = null;

  try {
    const [nextMetrics, nextLogs, nextAlerts, nextTrend] = await Promise.all([
      apiClient.monitoring.getCurrentMetrics(),
      apiClient.monitoring.getLogs(),
      apiClient.monitoring.getAlerts(),
      apiClient.monitoring.getMetricTrend(range, metric),
    ]);
    metrics.value = nextMetrics;
    logs.value = nextLogs;
    alerts.value = nextAlerts;
    trendPoints.value = nextTrend;
    usingFallback.value = false;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
  } finally {
    loading.value = false;
  }
}

export async function loadMetricTrend(metric: string, range: string) {
  try {
    trendPoints.value = await apiClient.monitoring.getMetricTrend(range, metric);
    usingFallback.value = false;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
  }
}

export async function createAlert(metric: string, range: string) {
  const alert = await apiClient.monitoring.createAlert({ metric, range });
  alerts.value = [alert, ...alerts.value.filter((item) => item.id !== alert.id)];
  usingFallback.value = false;
  return alert;
}

export async function muteAlert(id: string, muted: boolean) {
  const alert = await apiClient.monitoring.muteAlert(id, muted);
  alerts.value = alerts.value.map((item) => (item.id === alert.id ? alert : item));
  usingFallback.value = false;
  return alert;
}

export async function runDiagnostics() {
  const result = await apiClient.monitoring.runDiagnostics();
  diagnostic.value = result;
  usingFallback.value = false;
  return result;
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}
