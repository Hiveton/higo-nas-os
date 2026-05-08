import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { DesktopApp, DesktopBootstrap, DesktopSession, DesktopWindowConfig } from '../api/types';
import { desktopWindows, dockApps } from '../data/higoos';

const apps = ref<DesktopApp[]>([]);
const windows = ref<DesktopWindowConfig[]>([]);
const status = ref<DesktopBootstrap['status']>({});
const session = ref<DesktopSession | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

export const desktopStore = {
  apps: readonly(apps),
  windows: readonly(windows),
  status: readonly(status),
  session: readonly(session),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  loadDesktopBootstrap,
  saveSessionPatch,
};

export async function loadDesktopBootstrap(): Promise<DesktopBootstrap> {
  loading.value = true;
  error.value = null;

  try {
    const [nextApps, nextWindows, nextStatus] = await Promise.all([
      apiClient.desktop.getApps(),
      apiClient.desktop.getWindows(),
      apiClient.desktop.getSession(),
    ]);

    apps.value = nextApps;
    windows.value = nextWindows;
    status.value = nextStatus;
    session.value = nextStatus;
    usingFallback.value = false;

    return {
      apps: nextApps,
      windows: nextWindows,
      status: nextStatus,
      fallback: false,
    };
  } catch (reason) {
    const fallback = createFallbackBootstrap();

    apps.value = fallback.apps;
    windows.value = fallback.windows;
    status.value = fallback.status;
    session.value = fallback.status;
    error.value = reason instanceof Error ? reason : new Error(String(reason));
    usingFallback.value = true;

    return fallback;
  } finally {
    loading.value = false;
  }
}

export async function saveSessionPatch(patch: Partial<DesktopSession>): Promise<DesktopSession> {
  try {
    const nextSession = await apiClient.desktop.updateSession(patch);
    status.value = nextSession;
    session.value = nextSession;
    usingFallback.value = false;
    return nextSession;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
    throw error.value;
  }
}

function createFallbackBootstrap(): DesktopBootstrap & { status: DesktopSession } {
  const fallbackApps = dockApps.map((app) => ({ ...app }));
  const fallbackWindows = desktopWindows.map((window) => ({ ...window }));

  return {
    apps: fallbackApps,
    windows: fallbackWindows,
    status: createFallbackSession(fallbackApps, fallbackWindows),
    fallback: true,
  };
}

function createFallbackSession(
  fallbackApps: DesktopApp[],
  fallbackWindows: DesktopWindowConfig[],
): DesktopSession {
  const openWindowIds = fallbackWindows.slice(0, 3).map((window) => window.id);

  return {
    openWindowIds,
    minimizedWindowIds: [],
    activeWindowId: openWindowIds[0] ?? '',
    utilityAppId: '',
    assistantVisible: false,
    isCompact: false,
    maximizedWindowId: '',
    dockOrder: fallbackApps.map((app) => app.id),
    pinnedDockAppIds: ['file-manager', 'ai-file-steward', 'ai-assistant', 'system-settings'],
    desktopIconPositions: {},
    windowGeometries: {},
  };
}
