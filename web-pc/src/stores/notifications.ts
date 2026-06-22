import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import { monitoringStore, loadMonitoringSnapshot } from './monitoring';
import { activityStore } from './activity';
import type { StatusTone, StewardSuggestion } from '../api/types';

/**
 * Notification center store. Aggregates three live sources into one read-aware
 * feed: monitoring alerts (device health), steward suggestions (AI file
 * housekeeping) and recent user actions (activity log). Read state is tracked
 * per-item id in localStorage so the badge survives reloads.
 */

export type NotificationKind = 'alert' | 'steward' | 'activity';

export type NotificationItem = {
  id: string;
  kind: NotificationKind;
  title: string;
  detail: string;
  tone: StatusTone;
  at?: string;
  read: boolean;
  /** Optional deep-link: opens the given desktop app when actioned. */
  appId?: string;
};

const READ_KEY = 'higoos.notifications.read';

const suggestions = ref<StewardSuggestion[]>([]);
const readIds = ref<Set<string>>(loadReadIds());
const loading = ref(false);
let refCount = 0;
let pollTimer: number | undefined;

function loadReadIds(): Set<string> {
  try {
    return new Set(JSON.parse(localStorage.getItem(READ_KEY) || '[]'));
  } catch {
    return new Set();
  }
}

function persistReadIds() {
  try {
    localStorage.setItem(READ_KEY, JSON.stringify([...readIds.value]));
  } catch {
    /* quota / unavailable — ignore */
  }
}

function riskTone(risk?: string): StatusTone {
  if (risk === '高风险') return 'red';
  if (risk === '中风险') return 'orange';
  return 'blue';
}

const items = computed<NotificationItem[]>(() => {
  const out: NotificationItem[] = [];

  for (const alert of monitoringStore.alerts.value) {
    if (alert.muted) continue;
    const id = `alert:${alert.id ?? alert.title}`;
    const tone = (['blue', 'green', 'orange', 'red', 'cyan'].includes(String(alert.tone))
      ? alert.tone
      : alert.severity === '高风险'
        ? 'red'
        : 'orange') as StatusTone;
    out.push({
      id,
      kind: 'alert',
      title: alert.title,
      detail: alert.detail,
      tone,
      read: readIds.value.has(id),
      appId: 'device-monitor',
    });
  }

  for (const s of suggestions.value) {
    if (s.status && s.status !== 'pending') continue;
    const id = `steward:${s.id ?? s.title}`;
    out.push({
      id,
      kind: 'steward',
      title: s.title,
      detail: `${s.count ? `${s.count} · ` : ''}${s.detail}`,
      tone: riskTone(s.risk),
      at: s.updatedAt,
      read: readIds.value.has(id),
      appId: 'ai-file-steward',
    });
  }

  for (const entry of activityStore.entries.value.slice(0, 30)) {
    if (entry.type !== 'action') continue;
    const id = `activity:${entry.id ?? `${entry.at}-${entry.action}`}`;
    out.push({
      id,
      kind: 'activity',
      title: `${entry.category ?? '操作'} · ${entry.action}`,
      detail: [entry.target, entry.detail].filter(Boolean).join(' — ') || '已记录一次操作',
      tone: 'cyan',
      at: entry.at,
      read: readIds.value.has(id),
    });
  }

  // Newest first; unread floats above read at equal time.
  return out.sort((a, b) => {
    if (a.read !== b.read) return a.read ? 1 : -1;
    return (b.at ?? '').localeCompare(a.at ?? '');
  });
});

const unreadCount = computed(() => items.value.filter((item) => !item.read).length);

const grouped = computed(() => ({
  alert: items.value.filter((i) => i.kind === 'alert'),
  steward: items.value.filter((i) => i.kind === 'steward'),
  activity: items.value.filter((i) => i.kind === 'activity'),
}));

async function load() {
  loading.value = true;
  try {
    const [nextSuggestions] = await Promise.all([
      apiClient.steward.getSuggestions().catch(() => suggestions.value),
      loadMonitoringSnapshot().catch(() => undefined),
      activityStore.hydrate().catch(() => undefined),
    ]);
    suggestions.value = nextSuggestions ?? [];
  } finally {
    loading.value = false;
  }
}

function markRead(id: string) {
  if (readIds.value.has(id)) return;
  const next = new Set(readIds.value);
  next.add(id);
  readIds.value = next;
  persistReadIds();
}

function markAllRead() {
  const next = new Set(readIds.value);
  for (const item of items.value) next.add(item.id);
  readIds.value = next;
  persistReadIds();
}

/** Begin background refresh (ref-counted across the bell + the window). */
function start() {
  refCount += 1;
  if (refCount > 1) return;
  void load();
  pollTimer = window.setInterval(() => void load(), 30000);
}

function stop() {
  refCount = Math.max(0, refCount - 1);
  if (refCount > 0) return;
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer);
    pollTimer = undefined;
  }
}

export const notificationsStore = {
  items,
  grouped,
  unreadCount,
  loading: readonly(loading),
  load,
  markRead,
  markAllRead,
  start,
  stop,
};
