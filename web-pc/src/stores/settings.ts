import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { SettingsState, TaskResponse } from '../api/types';

const settings = ref<SettingsState>({});
const updates = ref<Record<string, unknown>>({});
const lastTask = ref<TaskResponse | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

export const settingsStore = {
  settings: readonly(settings),
  updates: readonly(updates),
  lastTask: readonly(lastTask),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  loadSettings,
  saveSettings,
  restoreDefaults,
  checkUpdates,
  createSystemBackup,
};

export async function loadSettings() {
  loading.value = true;
  error.value = null;

  try {
    const [nextSettings, nextUpdates] = await Promise.all([
      apiClient.settings.getSettings(),
      apiClient.settings.getUpdates(),
    ]);
    settings.value = nextSettings;
    updates.value = nextUpdates;
    usingFallback.value = false;
    return nextSettings;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
    return settings.value;
  } finally {
    loading.value = false;
  }
}

export async function saveSettings(payload: SettingsState) {
  const nextSettings = await apiClient.settings.updateSettings(payload);
  settings.value = nextSettings;
  usingFallback.value = false;
  return nextSettings;
}

export async function restoreDefaults() {
  const nextSettings = await apiClient.settings.restoreDefaults();
  settings.value = nextSettings;
  usingFallback.value = false;
  return nextSettings;
}

export async function checkUpdates() {
  const task = await apiClient.settings.checkUpdates();
  updates.value = await apiClient.settings.getUpdates();
  lastTask.value = task;
  usingFallback.value = false;
  return task;
}

export async function createSystemBackup() {
  const task = await apiClient.settings.createSystemBackup();
  lastTask.value = task;
  usingFallback.value = false;
  return task;
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}
