import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type {
  AiAnalysisDomain,
  AiAnalysisRecord,
  AiAnalysisReanalyzePayload,
  AiAnalysisState,
  AiAnalysisStatus,
  FaceFrameworkStatus,
} from '../api/types';

/**
 * Global AI analysis store. Subscribes to the SSE progress stream (one frame per
 * analysis-task state change) for the live overview, falling back to polling when
 * EventSource is unavailable or errors. Record pages are fetched on demand via
 * plain REST (they are not pushed over the stream).
 */

const overview = ref<AiAnalysisStatus | null>(null);
const records = ref<AiAnalysisRecord[]>([]);
const recordsPage = ref({ page: 1, pageSize: 50, total: 0 });
const recordsLoading = ref(false);
const connected = ref(false);
const error = ref<Error | null>(null);
const faces = ref<FaceFrameworkStatus | null>(null);
const facesLoading = ref(false);
// Active record filters, owned by the analysis center toolbar.
const filters = ref<{ domain?: AiAnalysisDomain; state?: AiAnalysisState; q?: string }>({});
// Single-record detail drawer state.
const recordDetail = ref<AiAnalysisRecord | null>(null);
const recordDetailLoading = ref(false);

let source: EventSource | undefined;
let pollTimer: number | undefined;
let refCount = 0;

async function loadStatus() {
  try {
    overview.value = await apiClient.aiAnalysis.getStatus();
    error.value = null;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  }
}

function startPolling() {
  if (pollTimer !== undefined) return;
  void loadStatus();
  pollTimer = window.setInterval(() => void loadStatus(), 4000);
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
    source = apiClient.aiAnalysis.streamProgress();
  } catch {
    startPolling();
    return;
  }
  source.onopen = () => {
    connected.value = true;
    error.value = null;
    stopPolling();
  };
  source.onmessage = (event) => {
    try {
      overview.value = JSON.parse(event.data) as AiAnalysisStatus;
    } catch {
      /* ignore malformed frame */
    }
  };
  source.onerror = () => {
    // EventSource auto-reconnects; poll meanwhile so the UI stays fresh.
    connected.value = false;
    startPolling();
  };
}

/** Begin streaming (ref-counted across windows). Seeds an initial snapshot. */
function start() {
  refCount += 1;
  if (refCount > 1) return;
  void loadStatus();
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

async function loadRecords(page = 1) {
  recordsLoading.value = true;
  try {
    const result = await apiClient.aiAnalysis.listRecords({
      domain: filters.value.domain,
      state: filters.value.state,
      q: filters.value.q?.trim() || undefined,
      page,
      size: recordsPage.value.pageSize,
    });
    records.value = result.records ?? [];
    recordsPage.value = { page: result.page, pageSize: result.pageSize, total: result.total };
    error.value = null;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  } finally {
    recordsLoading.value = false;
  }
}

/** Replace filters and reload from page 1. */
async function setFilters(next: Partial<{ domain?: AiAnalysisDomain; state?: AiAnalysisState; q?: string }>) {
  filters.value = { ...filters.value, ...next };
  await loadRecords(1);
}

async function loadRecord(key: string) {
  recordDetailLoading.value = true;
  try {
    recordDetail.value = await apiClient.aiAnalysis.getRecord(key);
    error.value = null;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  } finally {
    recordDetailLoading.value = false;
  }
}

function clearRecordDetail() {
  recordDetail.value = null;
}

/** Switch the background analysis depth via the settings store (single source). */
async function setLevel(level: 'off' | 'basic' | 'standard' | 'deep') {
  const current = await apiClient.settings.getSettings();
  await apiClient.settings.updateSettings({ ...current, analysis: { ...current.analysis, level } });
  await loadStatus();
}

async function batchReanalyze(keys: string[]) {
  const result = await apiClient.aiAnalysis.batchReanalyze(keys);
  overview.value = result.status;
  await loadRecords(recordsPage.value.page);
  return result.reset;
}

async function reanalyze(payload: AiAnalysisReanalyzePayload) {
  overview.value = await apiClient.aiAnalysis.reanalyze(payload);
}

async function pause() {
  overview.value = await apiClient.aiAnalysis.pause();
}

async function resume() {
  overview.value = await apiClient.aiAnalysis.resume();
}

async function rescan() {
  overview.value = await apiClient.aiAnalysis.rescan();
}

async function loadFaces() {
  facesLoading.value = true;
  try {
    faces.value = await apiClient.aiAnalysis.getFaces();
    error.value = null;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  } finally {
    facesLoading.value = false;
  }
}

async function labelFace(clusterId: string, name: string) {
  const result = await apiClient.aiAnalysis.labelFace(clusterId, name);
  faces.value = result.faces;
  return result.confirmed;
}

async function retrainFaces() {
  const result = await apiClient.aiAnalysis.retrainFaces();
  await loadFaces();
  return result.taskId;
}

export const aiAnalysisStore = {
  overview: readonly(overview),
  records: readonly(records),
  recordsPage: readonly(recordsPage),
  recordsLoading: readonly(recordsLoading),
  connected: readonly(connected),
  error: readonly(error),
  faces: readonly(faces),
  facesLoading: readonly(facesLoading),
  filters: readonly(filters),
  recordDetail: readonly(recordDetail),
  recordDetailLoading: readonly(recordDetailLoading),
  start,
  stop,
  loadStatus,
  loadRecords,
  setFilters,
  loadRecord,
  clearRecordDetail,
  setLevel,
  batchReanalyze,
  reanalyze,
  pause,
  resume,
  rescan,
  loadFaces,
  labelFace,
  retrainFaces,
};
