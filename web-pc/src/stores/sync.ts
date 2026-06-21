import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { SyncAuditEntry, SyncConflict, SyncPair } from '../api/types';

type RecordPayload = Record<string, unknown>;

const pairs = ref<SyncPair[]>([]);
const conflicts = ref<SyncConflict[]>([]);
const audit = ref<SyncAuditEntry[]>([]);
const loading = ref(false);
const usingFallback = ref(false);

const pendingConflicts = computed(() => conflicts.value.filter((c) => !c.resolved));

export const syncStore = {
  pairs: readonly(pairs),
  conflicts: readonly(conflicts),
  pendingConflicts,
  audit: readonly(audit),
  loading: readonly(loading),
  usingFallback: readonly(usingFallback),
  load,
  reloadAudit,
  createPair,
  updatePair,
  deletePair,
  runPair,
  verifyPair,
  resolveConflict,
};

export async function load() {
  loading.value = true;
  try {
    const [nextPairs, nextConflicts, nextAudit] = await Promise.all([
      apiClient.sync.getPairs(),
      apiClient.sync.getConflicts(),
      apiClient.sync.getAudit(),
    ]);
    pairs.value = nextPairs;
    conflicts.value = nextConflicts;
    audit.value = nextAudit;
    usingFallback.value = false;
    return nextPairs;
  } catch {
    usingFallback.value = true;
    if (pairs.value.length === 0) pairs.value = fallbackPairs();
    return pairs.value;
  } finally {
    loading.value = false;
  }
}

export async function reloadAudit() {
  audit.value = await apiClient.sync.getAudit();
}

export async function createPair(payload: RecordPayload) {
  const pair = await apiClient.sync.createPair(payload);
  await load();
  return pair;
}

export async function updatePair(id: string, payload: RecordPayload) {
  const pair = await apiClient.sync.updatePair(id, payload);
  pairs.value = pairs.value.map((p) => (p.id === pair.id ? pair : p));
  void reloadAudit();
  return pair;
}

export async function deletePair(id: string) {
  await apiClient.sync.deletePair(id);
  await load();
}

export async function runPair(id: string) {
  const pair = await apiClient.sync.runPair(id, { actor: 'sync-ui' });
  // Runs are async tasks; refresh shortly to pick up the settled state.
  pairs.value = pairs.value.map((p) => (p.id === pair.id ? pair : p));
  window.setTimeout(() => void load(), 800);
  return pair;
}

export async function verifyPair(id: string) {
  const pair = await apiClient.sync.verifyPair(id, { actor: 'sync-ui' });
  window.setTimeout(() => void load(), 800);
  return pair;
}

export async function resolveConflict(id: string, side: 'source' | 'target') {
  await apiClient.sync.resolveConflict(id, { side, actor: 'sync-ui' });
  await load();
}

function fallbackPairs(): SyncPair[] {
  return [
    {
      id: 'sync-001',
      name: '工作目录 → 备份盘',
      source: '/家庭空间/工作',
      target: '/备份/工作',
      direction: 'mirror',
      conflictPolicy: 'newer',
      enabled: true,
      intervalHours: 6,
      state: '空闲',
      progress: 0,
      createdAt: '2026-06-01T09:00:00Z',
    },
  ];
}
