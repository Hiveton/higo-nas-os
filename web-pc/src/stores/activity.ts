import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import { settingsStore } from './settings';
import type { ActivityEntry } from '../api/types';

/**
 * User-activity log store. Records meaningful actions + page (window) visits to
 * a localStorage buffer for resilience, batches them to the backend, and
 * hydrates the authoritative history on demand. Recording honors the
 * `settings.ui` activity toggle (default on).
 */

const BUFFER_KEY = 'higoos.activity.buffer';
const MAX_DISPLAY = 400;

const entries = ref<ActivityEntry[]>([]); // newest-first display list
const buffer: ActivityEntry[] = loadBuffer(); // unsynced, oldest-first
let flushTimer: number | undefined;
let hydrated = false;

function recordingEnabled(): boolean {
  return settingsStore.settings.value.activity?.enabled !== false;
}

function loadBuffer(): ActivityEntry[] {
  try {
    return JSON.parse(localStorage.getItem(BUFFER_KEY) || '[]');
  } catch {
    return [];
  }
}

function saveBuffer() {
  try {
    localStorage.setItem(BUFFER_KEY, JSON.stringify(buffer));
  } catch {
    /* quota / unavailable — ignore */
  }
}

/** Record one activity entry (no-op when recording is disabled). */
function record(entry: ActivityEntry) {
  if (!recordingEnabled()) return;
  const stamped: ActivityEntry = { ...entry, at: entry.at ?? new Date().toISOString() };
  buffer.push(stamped);
  saveBuffer();
  entries.value = [stamped, ...entries.value].slice(0, MAX_DISPLAY);
  scheduleFlush();
}

/** Convenience: record a page (window) visit. */
function recordPage(windowId: string, title: string) {
  record({ type: 'page', category: windowId, action: 'visit', target: title });
}

function scheduleFlush() {
  if (flushTimer !== undefined) return;
  flushTimer = window.setTimeout(() => {
    flushTimer = undefined;
    void flush();
  }, 1500);
}

async function flush() {
  if (buffer.length === 0) return;
  const batch = buffer.splice(0, buffer.length);
  saveBuffer();
  try {
    await apiClient.activity.append(batch);
  } catch {
    // restore on failure so nothing is lost
    buffer.unshift(...batch);
    saveBuffer();
  }
}

/** Fetch authoritative history from the backend (call when opening a viewer). */
async function hydrate(force = false) {
  if (hydrated && !force) return;
  hydrated = true;
  await flush();
  try {
    const list = await apiClient.activity.list(undefined, MAX_DISPLAY);
    const pending = [...buffer].reverse();
    entries.value = [...pending, ...list].slice(0, MAX_DISPLAY);
  } catch {
    /* keep local buffer view */
  }
}

async function clear() {
  buffer.length = 0;
  saveBuffer();
  entries.value = [];
  try {
    await apiClient.activity.clear();
  } catch {
    /* ignore */
  }
}

if (typeof window !== 'undefined') {
  window.addEventListener('beforeunload', () => void flush());
  document.addEventListener('visibilitychange', () => {
    if (document.visibilityState === 'hidden') void flush();
  });
}

export const activityStore = {
  entries: readonly(entries),
  record,
  recordPage,
  flush,
  hydrate,
  clear,
};

export type { ActivityEntry };
