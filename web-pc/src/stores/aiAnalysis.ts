import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type {
  AiAnalysisDomain,
  AiAnalysisRecord,
  AiAnalysisReanalyzePayload,
  AiAnalysisState,
  AiAnalysisStatus,
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

async function loadRecords(domain?: AiAnalysisDomain, state?: AiAnalysisState, page = 1) {
  recordsLoading.value = true;
  try {
    const result = await apiClient.aiAnalysis.listRecords({
      domain,
      state,
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

export const aiAnalysisStore = {
  overview: readonly(overview),
  records: readonly(records),
  recordsPage: readonly(recordsPage),
  recordsLoading: readonly(recordsLoading),
  connected: readonly(connected),
  error: readonly(error),
  start,
  stop,
  loadStatus,
  loadRecords,
  reanalyze,
  pause,
  resume,
  rescan,
};
