import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { Task, TaskStatus } from '../api/types';

/**
 * Central task store. Subscribes to the SSE task stream (one frame per task
 * state change) and keeps a live reactive map; falls back to polling when
 * EventSource is unavailable or the stream errors repeatedly.
 */

const byId = ref<Map<string, Task>>(new Map());
const connected = ref(false);
const error = ref<Error | null>(null);

let source: EventSource | undefined;
let pollTimer: number | undefined;
let refCount = 0;

const tasks = computed(() => {
  const list = [...byId.value.values()];
  list.sort((a, b) => {
    // active first, then most-recently-updated
    const rank = (t: Task) => (t.status === 'running' ? 0 : t.status === 'queued' ? 1 : 2);
    const r = rank(a) - rank(b);
    if (r !== 0) return r;
    return (b.updatedAt ?? '').localeCompare(a.updatedAt ?? '');
  });
  return list;
});

const stats = computed(() => {
  const s = { total: 0, queued: 0, running: 0, succeeded: 0, failed: 0, canceled: 0 };
  for (const t of byId.value.values()) {
    s.total += 1;
    s[t.status] += 1;
  }
  return s;
});

function upsert(task: Task) {
  const next = new Map(byId.value);
  next.set(task.id, task);
  byId.value = next;
}

async function loadOnce() {
  try {
    const list = await apiClient.tasks.list();
    const next = new Map<string, Task>();
    for (const t of list) next.set(t.id, t);
    byId.value = next;
    error.value = null;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  }
}

function startPolling() {
  if (pollTimer !== undefined) return;
  void loadOnce();
  pollTimer = window.setInterval(() => void loadOnce(), 4000);
}

function stopPolling() {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer);
    pollTimer = undefined;
  }
}

function connect() {
  if (typeof EventSource === 'undefined') {
    startPolling();
    return;
  }
  try {
    source = apiClient.tasks.stream();
  } catch {
    startPolling();
    return;
  }
  source.onopen = () => {
    connected.value = true;
    error.value = null;
    stopPolling(); // SSE replays a full snapshot on connect, so polling is redundant
  };
  source.onmessage = (event) => {
    try {
      upsert(JSON.parse(event.data) as Task);
    } catch {
      /* ignore malformed frame */
    }
  };
  source.onerror = () => {
    // EventSource auto-reconnects; meanwhile poll so the UI stays fresh.
    connected.value = false;
    startPolling();
  };
}

/** Begin streaming (ref-counted across windows). Seeds an initial snapshot. */
function start() {
  refCount += 1;
  if (refCount > 1) return;
  void loadOnce();
  connect();
}

/** Release one subscriber; tears down stream/polling when the last one leaves. */
function stop() {
  refCount = Math.max(0, refCount - 1);
  if (refCount > 0) return;
  source?.close();
  source = undefined;
  connected.value = false;
  stopPolling();
}

async function cancel(id: string) {
  const updated = await apiClient.tasks.cancel(id);
  upsert(updated);
  void import('./activity').then(({ activityStore }) =>
    activityStore.record({ type: 'action', category: 'task-center', action: 'cancel-task', target: updated.kind, detail: id }),
  );
  return updated;
}

export const tasksStore = {
  tasks,
  stats,
  connected: readonly(connected),
  error: readonly(error),
  start,
  stop,
  cancel,
  reload: loadOnce,
};

export type { Task, TaskStatus };
