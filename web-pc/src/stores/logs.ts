import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { ActivityEntry, AuditEntry, SystemLog } from '../api/types';

/**
 * Unified log center store. There is no single backend logs endpoint yet, so
 * this aggregates the existing per-source feeds — monitoring system logs, the
 * user activity trail, and governance audit records (security + steward) — into
 * one normalized, filterable stream.
 */

export type LogKind = 'system' | 'activity' | 'audit';
export type LogLevel = 'info' | 'warn' | 'error' | 'audit';

export type LogEntry = {
  id: string;
  kind: LogKind;
  level: LogLevel;
  source: string;
  message: string;
  at: string;
  actor?: string;
};

const entries = ref<LogEntry[]>([]);
const loading = ref(false);
const error = ref<string | null>(null);

function normalizeLevel(level?: string): LogLevel {
  const l = (level ?? '').toLowerCase();
  if (l.includes('error') || l.includes('错误') || l.includes('crit')) return 'error';
  if (l.includes('warn') || l.includes('警') || l.includes('告')) return 'warn';
  return 'info';
}

function fromSystem(log: SystemLog): LogEntry {
  return {
    id: `system:${log.id}`,
    kind: 'system',
    level: normalizeLevel(log.level),
    source: log.source || '系统',
    message: log.message,
    at: log.at || log.timestamp || '',
  };
}

function fromActivity(entry: ActivityEntry, index: number): LogEntry {
  return {
    id: `activity:${entry.id ?? index}`,
    kind: 'activity',
    level: 'info',
    source: entry.category || (entry.type === 'page' ? '页面' : '操作'),
    message: [entry.action, entry.target, entry.detail].filter(Boolean).join(' · '),
    at: entry.at || '',
    actor: entry.actor,
  };
}

function fromAudit(entry: AuditEntry, domain: string): LogEntry {
  return {
    id: `audit:${domain}:${entry.id}`,
    kind: 'audit',
    level: 'audit',
    source: `${domain} · ${entry.risk}`,
    message: `${entry.event}${entry.result ? ` （${entry.result}）` : ''}${entry.reverted ? ' · 已回滚' : ''}`,
    at: entry.time || '',
    actor: entry.actor,
  };
}

async function load() {
  loading.value = true;
  error.value = null;
  try {
    const [system, activity, securityAudit, stewardAudit] = await Promise.all([
      apiClient.monitoring.getLogs().catch(() => [] as SystemLog[]),
      apiClient.activity.list(undefined, 200).catch(() => [] as ActivityEntry[]),
      apiClient.security.getAudit().catch(() => [] as AuditEntry[]),
      apiClient.steward.getAudit().catch(() => [] as AuditEntry[]),
    ]);
    const merged: LogEntry[] = [
      ...system.map(fromSystem),
      ...activity.map(fromActivity),
      ...securityAudit.map((a) => fromAudit(a, '安全')),
      ...stewardAudit.map((a) => fromAudit(a, '文件管家')),
    ];
    merged.sort((a, b) => (b.at ?? '').localeCompare(a.at ?? ''));
    entries.value = merged;
  } catch (reason) {
    error.value = reason instanceof Error ? reason.message : String(reason);
  } finally {
    loading.value = false;
  }
}

const counts = computed(() => {
  const c = { all: entries.value.length, system: 0, activity: 0, audit: 0 };
  for (const e of entries.value) c[e.kind] += 1;
  return c;
});

export const logsStore = {
  entries: readonly(entries),
  counts,
  loading: readonly(loading),
  error: readonly(error),
  load,
};
