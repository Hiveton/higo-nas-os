import type { Metric } from '../../../api/types';

/** Human-readable byte size (binary units), e.g. 34359738368 -> "32.0 GB". */
export function formatBytes(bytes?: number): string {
  if (!bytes || bytes <= 0) return '—';
  const units = ['B', 'KB', 'MB', 'GB', 'TB', 'PB'];
  let value = bytes;
  let unit = 0;
  while (value >= 1024 && unit < units.length - 1) {
    value /= 1024;
    unit += 1;
  }
  const digits = value >= 100 || unit === 0 ? 0 : 1;
  return `${value.toFixed(digits)} ${units[unit]}`;
}

/** Index live monitoring metrics by their key for quick lookup in panels. */
export function indexMetrics(metrics: readonly Metric[]): Record<string, Metric> {
  const map: Record<string, Metric> = {};
  for (const metric of metrics) {
    if (metric.key) map[metric.key] = metric;
  }
  return map;
}

/** Non-empty string or an em dash placeholder. */
export function orDash(value?: string | number | null): string {
  if (value === null || value === undefined) return '—';
  const text = String(value).trim();
  return text === '' ? '—' : text;
}
