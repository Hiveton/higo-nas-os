<script setup lang="ts">
import { computed, onMounted, onUnmounted, reactive, ref, watch, type Component, type CSSProperties } from 'vue';
import {
  ArrowLeft,
  ArrowRight,
  AppWindow,
  Copy,
  FolderPlus,
  Maximize2,
  Minimize2,
  MousePointerClick,
  Move,
  Pin,
  PinOff,
  Power,
  RefreshCcw,
  RotateCcw,
  Settings,
  ShieldCheck,
  Square,
  Trash2,
  X,
} from 'lucide-vue-next';
import TopBar from './components/TopBar.vue';
import DesktopAppGrid from './components/DesktopAppGrid.vue';
import DesktopDock from './components/DesktopDock.vue';
import DesktopWindow from './components/DesktopWindow.vue';
import DesktopContextMenu from './components/DesktopContextMenu.vue';
import DesktopWidgets from './components/DesktopWidgets.vue';
import AiAssistantWindow from './components/windows/AiAssistantWindow.vue';
import FileManagerWindow from './components/windows/FileManagerWindow.vue';
import AiStewardWindow from './components/windows/AiStewardWindow.vue';
import AgentWorkbenchWindow from './components/windows/AgentWorkbenchWindow.vue';
import StorageMonitorWindow from './components/windows/StorageMonitorWindow.vue';
import PhotoMediaWindow from './components/windows/PhotoMediaWindow.vue';
import MusicCenterWindow from './components/windows/MusicCenterWindow.vue';
import VideoCenterWindow from './components/windows/VideoCenterWindow.vue';
import DownloadCenterWindow from './components/windows/DownloadCenterWindow.vue';
import BackupSyncWindow from './components/windows/BackupSyncWindow.vue';
import AppCenterWindow from './components/windows/AppCenterWindow.vue';
import AppFrameWindow from './components/windows/AppFrameWindow.vue';
import DockerWindow from './components/windows/DockerWindow.vue';
import SecurityCenterWindow from './components/windows/SecurityCenterWindow.vue';
import DeviceMonitorWindow from './components/windows/DeviceMonitorWindow.vue';
import HardwareCenterWindow from './components/windows/HardwareCenterWindow.vue';
import SystemSettingsWindow from './components/windows/SystemSettingsWindow.vue';
import RemoteAccessWindow from './components/windows/RemoteAccessWindow.vue';
import ProtocolsWindow from './components/windows/ProtocolsWindow.vue';
import TaskCenterWindow from './components/windows/TaskCenterWindow.vue';
import FeatureModuleWindow from './components/windows/FeatureModuleWindow.vue';
import { desktopStore } from './stores/desktop';
import { settingsStore } from './stores/settings';
import { activityStore } from './stores/activity';
import { setLocale } from './i18n';
import { UiToastHost, UiConfirmHost, useToast } from './components/ui';
import { apiClient } from './api/client';
import { assistantStore } from './stores/assistant';
import type { DesktopApp, DesktopSession, DesktopWindowConfig } from './api/types';
import { desktopWindows as seedDesktopWindows, dockApps as seedDockApps } from './data/higoos';
import type { NasFeatureKey } from './data/nasFeatures';
import wallpaperUrl from './assets/higoos-dock/wallpaper.png';

type WindowGeometry = {
  x: number;
  y: number;
  width: number;
  height: number;
};

type IconPosition = {
  x: number;
  y: number;
};

type DesktopIconArrangeMode = 'right' | 'left' | 'name' | 'status';
type DockPosition = 'bottom' | 'left' | 'right';
type DockStyle = 'floating' | 'side' | 'compact';
type DockIconSize = 'small' | 'default' | 'large';

type AppContextMenuItem = {
  id: string;
  label: string;
  hint?: string;
  icon?: Component;
  danger?: boolean;
  disabled?: boolean;
};

type ContextMenuState = {
  visible: boolean;
  x: number;
  y: number;
  title: string;
  subtitle?: string;
  source: 'desktop' | 'desktop-app' | 'dock' | 'window' | 'interactive';
  sourceId?: string;
  items: AppContextMenuItem[];
};

const toast = useToast();

const defaultPinnedDockAppIds = [
  'file-manager',
  'music-center',
  'video-center',
  'ai-file-steward',
  'ai-assistant',
  'system-settings',
];
const desktopIconWidth = 94;
const desktopIconHeight = 88;
const desktopIconGapX = 26;
const desktopIconGapY = 12;
const compactBreakpointWidth = 980;
const compactBreakpointHeight = 760;
const windowFrameMargin = 16;
const windowFrameTop = 78;
const windowFrameBottom = 118;
const minWindowWidth = 360;
const minWindowHeight = 300;
const uiSettings = computed(() => settingsStore.settings.value.ui ?? {});
const uiTheme = computed(() => normalizeOption(uiSettings.value.theme, ['auto', 'light', 'dark'] as const, 'auto'));
const uiLocale = computed(() => normalizeOption(uiSettings.value.locale, ['zh-CN', 'en-US'] as const, 'zh-CN'));
watch(uiLocale, (locale) => setLocale(locale), { immediate: true });
const uiWindowRadius = computed(() =>
  normalizeOption(uiSettings.value.windowRadius, ['compact', 'default', 'rounded'] as const, 'default'),
);
const uiDockPosition = computed<DockPosition>(() =>
  normalizeOption(uiSettings.value.dockPosition, ['bottom', 'left', 'right'] as const, 'bottom'),
);
const uiDockStyle = computed<DockStyle>(() =>
  normalizeOption(uiSettings.value.dockStyle, ['floating', 'side', 'compact'] as const, 'floating'),
);
const uiDockIconSize = computed<DockIconSize>(() =>
  normalizeOption(uiSettings.value.dockIconSize, ['small', 'default', 'large'] as const, 'default'),
);
const isSideDock = computed(() => uiDockPosition.value === 'left' || uiDockPosition.value === 'right');
const desktopUiClasses = computed(() => [
  `desktop--theme-${uiTheme.value}`,
  `desktop--radius-${uiWindowRadius.value}`,
  `desktop--dock-${uiDockPosition.value}`,
  `desktop--dock-style-${uiDockStyle.value}`,
  `desktop--dock-size-${uiDockIconSize.value}`,
]);
const desktopUiStyle = computed<CSSProperties>(() => ({
  backgroundImage: `url(${wallpaperUrl})`,
  '--dock-height': isSideDock.value ? '0px' : dockHeightForIconSize(uiDockIconSize.value),
  '--dock-side-width': isSideDock.value ? dockSideWidthForIconSize(uiDockIconSize.value) : '0px',
  '--radius-sm': radiusTokens(uiWindowRadius.value).sm,
  '--radius-md': radiusTokens(uiWindowRadius.value).md,
  '--radius-lg': radiusTokens(uiWindowRadius.value).lg,
  '--radius-xl': radiusTokens(uiWindowRadius.value).xl,
}));
const dockApps = reactive<DesktopApp[]>([]);
const desktopWindows = reactive<DesktopWindowConfig[]>([]);
const apiDesktopAppIds = ref<Set<string>>(new Set());
const apiDesktopWindowIds = ref<Set<string>>(new Set());
const openWindowIds = ref<string[]>([]);
const minimizedWindowIds = ref<string[]>([]);
const activeWindowId = ref('');
const windowOpenTokens = ref<Record<string, number>>({});
const utilityAppId = ref('');
const assistantVisible = ref(false);
const isCompact = ref(false);
const maximizedWindowId = ref('');
const dockOrder = ref(dockApps.map((app) => app.id));
const pinnedDockAppIds = ref([...defaultPinnedDockAppIds]);
const desktopIconPositions = ref<Record<string, IconPosition>>(createDesktopIconLayout(dockApps, 'left'));
const windowGeometries = ref<Record<string, Partial<WindowGeometry>>>({});
const windowLayerOrder = ref<string[]>([]);
// Dynamically-registered third-party app frames (id -> iframe source). Lets an
// installed app present its own UI in a desktop window without hardcoding it.
const appFrameSources = ref<Record<string, { src: string; name: string }>>({});
const launchProgress = ref(0);
const contextTarget = ref<HTMLElement | null>(null);
const contextMenu = ref<ContextMenuState>({
  visible: false,
  x: 24,
  y: 96,
  title: '桌面',
  source: 'desktop',
  items: [],
});
const featureModuleByWindowId: Record<string, NasFeatureKey> = {
  'virtual-machine': 'vm',
  'sync-service': 'sync',
  'iscsi-manager': 'iscsi',
};
let launchTimer: number | undefined;
let sessionSaveTimer: number | undefined;
let isHydratingSession = false;

const visibleWindowIds = computed(() => {
  return openWindowIds.value.filter((id) => !minimizedWindowIds.value.includes(id));
});

const openWindows = computed(() =>
  visibleWindowIds.value
    .map((id) => {
      const config = desktopWindows.find((window) => window.id === id);
      if (!config) return null;
      const geometry = normalizeWindowGeometry(id, windowGeometries.value[id]);
      return {
        ...config,
        ...geometry,
        z: 40 + getWindowLayerIndex(id),
      };
    })
    .filter((window): window is DesktopWindowConfig => Boolean(window)),
);

const desktopWindowIds = computed(() => desktopWindows.map((window) => window.id));

function getWindowLayerIndex(id: string) {
  const index = windowLayerOrder.value.indexOf(id);
  if (index >= 0) return index;
  return desktopWindows.findIndex((window) => window.id === id);
}

function prioritizeOpenWindowOrder(windowIds: string[], activeId: string) {
  const knownIds = new Set(windowIds);
  const ordered = windowLayerOrder.value.filter((id) => knownIds.has(id));
  for (const id of windowIds) {
    if (!ordered.includes(id)) ordered.push(id);
  }
  if (activeId && knownIds.has(activeId)) {
    return [...ordered.filter((id) => id !== activeId), activeId];
  }
  return ordered;
}

function bringWindowToFront(id: string) {
  activeWindowId.value = id;
  if (!isDesktopWindow(id)) return;
  windowLayerOrder.value = prioritizeOpenWindowOrder(openWindowIds.value, id);
}

function bumpWindowOpenToken(id: string) {
  windowOpenTokens.value = {
    ...windowOpenTokens.value,
    [id]: (windowOpenTokens.value[id] ?? 0) + 1,
  };
}

function selectNextVisibleWindow(excludedId = '') {
  const candidates = windowLayerOrder.value.filter(
    (id) => id !== excludedId && openWindowIds.value.includes(id) && !minimizedWindowIds.value.includes(id),
  );
  if (candidates.length) return candidates[candidates.length - 1];
  return openWindowIds.value.find((id) => id !== excludedId && !minimizedWindowIds.value.includes(id)) ?? '';
}

const runningDockAppIds = computed(() =>
  dockApps
    .filter((app) => isDockAppRunning(app.id))
    .map((app) => app.id),
);

const visibleDockApps = computed(() =>
  dockOrder.value
    .map((id) => dockApps.find((app) => app.id === id))
    .filter((app): app is (typeof dockApps)[number] => {
      if (!app) return false;
      return pinnedDockAppIds.value.includes(app.id) || runningDockAppIds.value.includes(app.id);
    }),
);

function normalizeOption<T extends string>(value: unknown, allowed: readonly T[], fallback: T): T {
  return typeof value === 'string' && allowed.includes(value as T) ? (value as T) : fallback;
}

function radiusTokens(value: string) {
  if (value === 'compact') {
    return { sm: '6px', md: '9px', lg: '12px', xl: '16px' };
  }
  if (value === 'rounded') {
    return { sm: '12px', md: '18px', lg: '26px', xl: '34px' };
  }
  return { sm: '8px', md: '12px', lg: '18px', xl: '24px' };
}

function dockHeightForIconSize(value: DockIconSize) {
  if (value === 'small') return '82px';
  if (value === 'large') return '112px';
  return '96px';
}

function dockSideWidthForIconSize(value: DockIconSize) {
  if (value === 'small') return '78px';
  if (value === 'large') return '116px';
  return '96px';
}

function dockSideWidthPx() {
  return isSideDock.value ? parseInt(dockSideWidthForIconSize(uiDockIconSize.value), 10) || 96 : 0;
}

async function loadDesktopBootstrapFromApi() {
  isHydratingSession = true;
  try {
    const bootstrap = await desktopStore.loadDesktopBootstrap();
    apiDesktopAppIds.value = new Set(bootstrap.apps.map((app) => app.id));
    apiDesktopWindowIds.value = new Set(bootstrap.windows.map((window) => window.id));
    dockApps.splice(0, dockApps.length, ...mergeDesktopApps(bootstrap.apps));
    desktopWindows.splice(0, desktopWindows.length, ...mergeDesktopWindows(bootstrap.windows));
    applyDesktopSession(readDesktopSession(bootstrap.status));
    updateCompactState();
    if (bootstrap.fallback) {
      showToast('后端暂不可用，已启用本地桌面种子。');
    }
  } finally {
    isHydratingSession = false;
  }
}

function normalizeDesktopApps(apps: DesktopApp[]) {
  const seedIcons = new Map(seedDockApps.map((app) => [app.id, app.icon]));
  return apps.map((app) => {
    const seedIcon = seedIcons.get(app.id);
    const icon = app.icon && !app.icon.startsWith('/src/') ? app.icon : seedIcon;
    return {
      ...app,
      icon: icon ?? app.icon,
    };
  });
}

function mergeDesktopApps(apps: DesktopApp[]) {
  const normalized = normalizeDesktopApps(apps);
  const existingIds = new Set(normalized.map((app) => app.id));
  return [
    ...normalized,
    ...seedDockApps.filter((app) => !existingIds.has(app.id)).map((app) => ({ ...app })),
  ];
}

function mergeDesktopWindows(windows: DesktopWindowConfig[]) {
  const existingIds = new Set(windows.map((window) => window.id));
  return [
    ...windows,
    ...seedDesktopWindows.filter((window) => !existingIds.has(window.id)).map((window) => ({ ...window })),
  ];
}

function readDesktopSession(value: unknown): DesktopSession {
  if (Array.isArray((value as DesktopSession).openWindowIds)) {
    return value as DesktopSession;
  }
  return createDefaultDesktopSession();
}

function createDefaultDesktopSession(): DesktopSession {
  const openWindowIds = desktopWindows.slice(0, 3).map((window) => window.id);
  return {
    openWindowIds,
    minimizedWindowIds: [],
    activeWindowId: openWindowIds[0] ?? '',
    utilityAppId: '',
    assistantVisible: false,
    isCompact: false,
    maximizedWindowId: '',
    dockOrder: dockApps.map((app) => app.id),
    pinnedDockAppIds: defaultPinnedDockAppIds.filter((id) => dockApps.some((app) => app.id === id)),
    desktopIconPositions: {},
    windowGeometries: {},
  };
}

function applyDesktopSession(session: DesktopSession) {
  const appIds = new Set(dockApps.map((app) => app.id));
  const windowIds = new Set(desktopWindows.map((window) => window.id));
  const allAppIds = dockApps.map((app) => app.id);
  const nextOpenWindowIds = filterKnownIds(session.openWindowIds, windowIds);
  const nextDockOrder = filterKnownIds(session.dockOrder, appIds);
  const nextPinnedDockAppIds = filterKnownIds(session.pinnedDockAppIds, appIds);
  const nextActiveWindowId = appIds.has(session.activeWindowId) ? session.activeWindowId : nextOpenWindowIds[0] ?? '';
  const nextMaximizedWindowId = windowIds.has(session.maximizedWindowId ?? '') ? session.maximizedWindowId ?? '' : '';
  const nextUtilityAppId = appIds.has(session.utilityAppId ?? '') ? session.utilityAppId ?? '' : '';

  openWindowIds.value = nextOpenWindowIds.length ? nextOpenWindowIds : desktopWindows.slice(0, 3).map((window) => window.id);
  minimizedWindowIds.value = filterKnownIds(session.minimizedWindowIds, windowIds);
  activeWindowId.value = nextActiveWindowId || openWindowIds.value[0] || '';
  windowLayerOrder.value = prioritizeOpenWindowOrder(openWindowIds.value, activeWindowId.value);
  utilityAppId.value = nextUtilityAppId;
  assistantVisible.value = Boolean(session.assistantVisible);
  isCompact.value = Boolean(session.isCompact);
  maximizedWindowId.value = nextMaximizedWindowId;
  dockOrder.value = [...nextDockOrder, ...allAppIds.filter((id) => !nextDockOrder.includes(id))];
  pinnedDockAppIds.value = nextPinnedDockAppIds.length
    ? nextPinnedDockAppIds
    : defaultPinnedDockAppIds.filter((id) => appIds.has(id));
  const defaultIconLayout = createDesktopIconLayout(dockApps, 'left');
  const savedIconLayout = filterIconPositions(session.desktopIconPositions ?? {}, appIds);
  desktopIconPositions.value = isSideDock.value ? defaultIconLayout : { ...defaultIconLayout, ...savedIconLayout };
  windowGeometries.value = filterWindowGeometries(session.windowGeometries ?? {}, windowIds);
  normalizeOpenWindowGeometries();
}

function filterKnownIds(values: string[] | undefined, allowed: Set<string>) {
  const seen = new Set<string>();
  return (values ?? []).filter((value) => {
    if (!allowed.has(value) || seen.has(value)) return false;
    seen.add(value);
    return true;
  });
}

function normalizeIconPosition(position: IconPosition) {
  const bounds = getDesktopStageBounds();
  return {
    x: Math.round(clampNumber(position.x, bounds.left, bounds.right - desktopIconWidth)),
    y: Math.round(clampNumber(position.y, bounds.top, bounds.bottom - desktopIconHeight)),
  };
}

function filterIconPositions(values: Record<string, IconPosition>, allowed: Set<string>) {
  return Object.fromEntries(
    Object.entries(values)
      .filter(([id]) => allowed.has(id))
      .map(([id, position]) => [id, normalizeIconPosition(position)]),
  );
}

function filterWindowGeometries(values: NonNullable<DesktopSession['windowGeometries']>, allowed: Set<string>) {
  return Object.fromEntries(
    Object.entries(values)
      .filter(([id]) => allowed.has(id))
      .map(([id, geometry]) => [id, normalizeWindowGeometry(id, geometry)]),
  );
}

function snapshotDesktopSession(): DesktopSession {
  return {
    openWindowIds: [...openWindowIds.value],
    minimizedWindowIds: [...minimizedWindowIds.value],
    activeWindowId: activeWindowId.value,
    utilityAppId: utilityAppId.value,
    assistantVisible: assistantVisible.value,
    isCompact: isCompact.value,
    maximizedWindowId: maximizedWindowId.value,
    dockOrder: [...dockOrder.value],
    pinnedDockAppIds: [...pinnedDockAppIds.value],
    desktopIconPositions: { ...desktopIconPositions.value },
    windowGeometries: { ...windowGeometries.value },
  };
}

function snapshotDesktopSessionForApi(): DesktopSession {
  const appIds = apiDesktopAppIds.value.size ? apiDesktopAppIds.value : new Set(dockApps.map((app) => app.id));
  const windowIds = apiDesktopWindowIds.value.size
    ? apiDesktopWindowIds.value
    : new Set(desktopWindows.map((window) => window.id));
  const snapshot = snapshotDesktopSession();

  return {
    ...snapshot,
    openWindowIds: filterKnownIds(snapshot.openWindowIds, windowIds),
    minimizedWindowIds: filterKnownIds(snapshot.minimizedWindowIds, windowIds),
    activeWindowId: appIds.has(snapshot.activeWindowId) || windowIds.has(snapshot.activeWindowId) ? snapshot.activeWindowId : '',
    utilityAppId: appIds.has(snapshot.utilityAppId ?? '') ? snapshot.utilityAppId ?? '' : '',
    maximizedWindowId: windowIds.has(snapshot.maximizedWindowId ?? '') ? snapshot.maximizedWindowId ?? '' : '',
    dockOrder: filterKnownIds(snapshot.dockOrder, appIds),
    pinnedDockAppIds: filterKnownIds(snapshot.pinnedDockAppIds, appIds),
    desktopIconPositions: filterIconPositions(snapshot.desktopIconPositions ?? {}, appIds),
    windowGeometries: filterWindowGeometries(snapshot.windowGeometries ?? {}, windowIds),
  };
}

function queueSessionSave() {
  if (isHydratingSession || dockApps.length === 0 || desktopStore.usingFallback.value) return;
  if (sessionSaveTimer) window.clearTimeout(sessionSaveTimer);
  sessionSaveTimer = window.setTimeout(() => {
    void saveSessionPatch(snapshotDesktopSessionForApi());
  }, 450);
}

async function saveSessionPatch(patch: Partial<DesktopSession>) {
  try {
    await desktopStore.saveSessionPatch(patch);
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason);
    showToast(`桌面布局暂未保存：${message}`);
  }
}

function clampNumber(value: number, min: number, max: number) {
  return Math.min(Math.max(value, min), Math.max(min, max));
}

function getWindowFrameBounds() {
  const viewportWidth = typeof window === 'undefined' ? 1440 : window.innerWidth;
  const viewportHeight = typeof window === 'undefined' ? 900 : window.innerHeight;
  const side = viewportWidth <= compactBreakpointWidth ? 14 : windowFrameMargin;
  const top = viewportWidth <= compactBreakpointWidth ? 84 : windowFrameTop;
  const bottom = isSideDock.value && viewportWidth > compactBreakpointWidth ? 12 : windowFrameBottom;
  const sideDockInset = isSideDock.value && viewportWidth > compactBreakpointWidth ? dockSideWidthPx() + 12 : 0;
  const left = uiDockPosition.value === 'left' ? side + sideDockInset : side;
  const right = uiDockPosition.value === 'right' ? viewportWidth - side - sideDockInset : viewportWidth - side;

  return {
    left,
    top,
    right: Math.max(left + minWindowWidth, right),
    bottom: Math.max(top + minWindowHeight, viewportHeight - bottom),
  };
}

function normalizeWindowGeometry(id: string, geometry: Partial<WindowGeometry> = {}): WindowGeometry {
  const config = desktopWindows.find((window) => window.id === id);
  const bounds = getWindowFrameBounds();
  const maxWidth = Math.max(280, bounds.right - bounds.left);
  const maxHeight = Math.max(240, bounds.bottom - bounds.top);
  const width = clampNumber(geometry.width ?? config?.width ?? 640, Math.min(minWindowWidth, maxWidth), maxWidth);
  const height = clampNumber(geometry.height ?? config?.height ?? 480, Math.min(minWindowHeight, maxHeight), maxHeight);
  const x = clampNumber(geometry.x ?? config?.x ?? bounds.left, bounds.left, bounds.right - width);
  const y = clampNumber(geometry.y ?? config?.y ?? bounds.top, bounds.top, bounds.bottom - height);

  return {
    x: Math.round(x),
    y: Math.round(y),
    width: Math.round(width),
    height: Math.round(height),
  };
}

function normalizeOpenWindowGeometries() {
  const nextGeometries = { ...windowGeometries.value };
  for (const id of openWindowIds.value) {
    nextGeometries[id] = normalizeWindowGeometry(id, nextGeometries[id]);
  }
  windowGeometries.value = nextGeometries;
}

function getDesktopStageSize() {
  const viewportWidth = typeof window === 'undefined' ? 1440 : window.innerWidth;
  const viewportHeight = typeof window === 'undefined' ? 900 : window.innerHeight;
  return {
    width: viewportWidth,
    height: Math.max(
      420,
      viewportHeight - (isSideDock.value ? 0 : 64 + (parseInt(dockHeightForIconSize(uiDockIconSize.value), 10) || 96)),
    ),
  };
}

function getDesktopStageBounds() {
  const stage = getDesktopStageSize();
  const narrow = stage.width <= 900;
  const sideDockInset = isSideDock.value && stage.width > 820 ? getSideDockReservedInset() : 0;
  const edgeInset = narrow ? 8 : 20;
  const left = uiDockPosition.value === 'left' ? sideDockInset + edgeInset : edgeInset;
  const right = stage.width - (uiDockPosition.value === 'right' ? sideDockInset + edgeInset : edgeInset);
  const top = narrow ? 8 : 68;
  const bottom = stage.height - 8;
  return {
    left,
    top,
    right: Math.max(left + desktopIconWidth, right),
    bottom: Math.max(top + desktopIconHeight, bottom),
  };
}

function getSideDockReservedInset() {
  return dockSideWidthPx() + 36;
}

function createDesktopIconLayout(apps: typeof dockApps, mode: DesktopIconArrangeMode) {
  const bounds = getDesktopStageBounds();
  const stage = getDesktopStageSize();
  const marginY = bounds.top + (stage.width <= 900 ? 2 : 10);
  const usableHeight = Math.max(desktopIconHeight, bounds.bottom - marginY);
  const rows = Math.max(1, Math.floor((usableHeight + desktopIconGapY) / (desktopIconHeight + desktopIconGapY)));
  const orderedApps = [...apps].sort((a, b) => {
    if (mode === 'name') return a.name.localeCompare(b.name, 'zh-Hans-CN');
    if (mode === 'status') {
      const aWeight = Number(isDockAppRunning(a.id)) * 2 + Number(pinnedDockAppIds.value.includes(a.id));
      const bWeight = Number(isDockAppRunning(b.id)) * 2 + Number(pinnedDockAppIds.value.includes(b.id));
      return bWeight - aWeight || a.name.localeCompare(b.name, 'zh-Hans-CN');
    }
    return dockApps.findIndex((item) => item.id === a.id) - dockApps.findIndex((item) => item.id === b.id);
  });

  return orderedApps.reduce<Record<string, IconPosition>>((positions, app, index) => {
    const column = Math.floor(index / rows);
    const row = index % rows;
    const x =
      mode === 'left'
        ? bounds.left + column * (desktopIconWidth + desktopIconGapX)
        : bounds.right - desktopIconWidth - column * (desktopIconWidth + desktopIconGapX);
    const y = marginY + row * (desktopIconHeight + desktopIconGapY);

    positions[app.id] = {
      x: Math.round(clampNumber(x, bounds.left, bounds.right - desktopIconWidth)),
      y: Math.round(clampNumber(y, bounds.top, bounds.bottom - desktopIconHeight)),
    };
    return positions;
  }, {});
}

function isDesktopWindow(id: string) {
  return desktopWindowIds.value.includes(id);
}

function isDockAppRunning(id: string) {
  if (isDesktopWindow(id)) return openWindowIds.value.includes(id);
  if (id === 'ai-assistant') return assistantVisible.value;
  return utilityAppId.value === id;
}

function getWindowTitle(id: string) {
  return desktopWindows.find((window) => window.id === id)?.title ?? dockApps.find((app) => app.id === id)?.name ?? '窗口';
}

function clampContextMenuPosition(x: number, y: number, itemCount = 0) {
  const estimatedWidth = 268;
  const estimatedHeight = Math.min(window.innerHeight - 24, 62 + itemCount * 38);

  return {
    x: Math.min(Math.max(12, x), Math.max(12, window.innerWidth - estimatedWidth)),
    y: Math.min(Math.max(12, y), Math.max(12, window.innerHeight - estimatedHeight - 12)),
  };
}

function openContextMenu(next: Omit<ContextMenuState, 'visible' | 'x' | 'y'> & { x: number; y: number }) {
  const position = clampContextMenuPosition(next.x, next.y, next.items.length);
  contextMenu.value = {
    ...next,
    ...position,
    visible: true,
  };
}

function closeContextMenu() {
  contextMenu.value.visible = false;
  contextTarget.value = null;
}

function getInteractiveLabel(element: HTMLElement) {
  const ariaLabel = element.getAttribute('aria-label');
  const visibleText = element.textContent?.replace(/\s+/g, ' ').trim();
  const formLabel = element instanceof HTMLInputElement ? element.value || element.placeholder : '';
  return ariaLabel || visibleText || formLabel || '当前控件';
}

function getInteractiveTarget(target: EventTarget | null) {
  if (!(target instanceof HTMLElement)) return null;
  return target.closest<HTMLElement>(
    'button, a, input, select, textarea, [role="button"], [role="menuitem"], [data-context-label]',
  );
}

function openDesktopContextMenu(event: MouseEvent) {
  const interactive = getInteractiveTarget(event.target);
  if (interactive) {
    contextTarget.value = interactive;
    openContextMenu({
      x: event.clientX,
      y: event.clientY,
      source: 'interactive',
      title: getInteractiveLabel(interactive),
      subtitle: '交互控件',
      items: [
        { id: 'run-interactive', label: '执行此操作', hint: 'Enter', icon: MousePointerClick },
        { id: 'copy-interactive-label', label: '复制控件名称', icon: Copy },
        { id: 'inspect-permission', label: '查看权限影响', icon: ShieldCheck },
        { id: 'agent-shortcut', label: '加入 Agent 工作流', icon: AppWindow },
      ],
    });
    return;
  }

  openContextMenu({
    x: event.clientX,
    y: event.clientY,
    source: 'desktop',
    title: 'HiGoOS 桌面',
    subtitle: '桌面操作',
    items: [
      { id: 'new-folder', label: '新建文件夹', icon: FolderPlus },
      { id: 'arrange-icons-right', label: '整理图标到右侧', icon: Move },
      { id: 'arrange-icons-left', label: '整理图标到左侧', icon: Move },
      { id: 'sort-icons-name', label: '按名称排序', icon: RefreshCcw },
      { id: 'sort-icons-status', label: '按状态排序', icon: ShieldCheck },
      { id: 'arrange-desktop', label: '整理桌面窗口', icon: Move },
      { id: 'refresh-desktop', label: '刷新桌面', hint: 'F5', icon: RefreshCcw },
      { id: 'open-system-settings', label: '打开系统设置', icon: Settings },
      { id: 'dock-reset', label: '恢复默认 Dock', icon: RotateCcw },
    ],
  });
}

function openDockContextMenu(payload: { id: string; x: number; y: number }) {
  const app = dockApps.find((item) => item.id === payload.id);
  const isWindow = isDesktopWindow(payload.id);
  const isRunning = isDockAppRunning(payload.id);
  const isPinned = pinnedDockAppIds.value.includes(payload.id);
  const visibleIndex = visibleDockApps.value.findIndex((item) => item.id === payload.id);

  openContextMenu({
    x: payload.x,
    y: payload.y,
    source: 'dock',
    sourceId: payload.id,
    title: app?.name ?? 'Dock 应用',
    subtitle: isWindow ? 'Dock 窗口操作' : 'Dock 应用操作',
    items: [
      { id: 'dock-open', label: isRunning ? '切换到此应用' : '打开', icon: AppWindow },
      { id: 'dock-close', label: isWindow ? '关闭窗口' : '退出应用', icon: Power, danger: true, disabled: !isRunning },
      { id: 'dock-minimize', label: '最小化窗口', icon: Minimize2, disabled: !isWindow || !openWindowIds.value.includes(payload.id) },
      { id: 'dock-maximize', label: '最大化窗口', icon: Maximize2, disabled: !isWindow },
      { id: 'dock-pin-toggle', label: isPinned ? '取消固定' : '固定到 Dock', icon: isPinned ? PinOff : Pin },
      { id: 'dock-remove', label: '从 Dock 移除', icon: Trash2, danger: !isRunning },
      { id: 'dock-move-left', label: '向左移动', icon: ArrowLeft, disabled: visibleIndex <= 0 },
      {
        id: 'dock-move-right',
        label: '向右移动',
        icon: ArrowRight,
        disabled: visibleIndex < 0 || visibleIndex >= visibleDockApps.value.length - 1,
      },
      { id: 'dock-reset', label: '恢复默认 Dock', icon: RotateCcw },
    ],
  });
}

function openDesktopAppContextMenu(payload: { id: string; x: number; y: number }) {
  const app = dockApps.find((item) => item.id === payload.id);
  const isWindow = isDesktopWindow(payload.id);
  const isRunning = isDockAppRunning(payload.id);
  const isPinned = pinnedDockAppIds.value.includes(payload.id);

  openContextMenu({
    x: payload.x,
    y: payload.y,
    source: 'desktop-app',
    sourceId: payload.id,
    title: app?.name ?? '桌面应用',
    subtitle: isRunning ? '桌面应用 / 正在运行' : '桌面应用',
    items: [
      { id: 'dock-open', label: isRunning ? '切换到此应用' : '打开应用', icon: AppWindow },
      { id: 'dock-pin-toggle', label: isPinned ? '取消固定到 Dock' : '固定到 Dock', icon: isPinned ? PinOff : Pin },
      { id: 'dock-remove', label: '从 Dock 移除', icon: Trash2, danger: !isRunning, disabled: !isPinned },
      { id: 'dock-close', label: isWindow ? '关闭窗口' : '退出应用', icon: Power, danger: true, disabled: !isRunning },
      { id: 'agent-shortcut', label: '加入 Agent 工作流', icon: AppWindow },
    ],
  });
}

function openWindowContextMenu(event: MouseEvent, id: string) {
  const interactive = getInteractiveTarget(event.target);
  if (interactive && !interactive.closest('.desktop-window__titlebar')) {
    contextTarget.value = interactive;
    openContextMenu({
      x: event.clientX,
      y: event.clientY,
      source: 'interactive',
      sourceId: id,
      title: getInteractiveLabel(interactive),
      subtitle: `${getWindowTitle(id)} 内的交互`,
      items: [
        { id: 'run-interactive', label: '执行此操作', hint: 'Enter', icon: MousePointerClick },
        { id: 'copy-interactive-label', label: '复制控件名称', icon: Copy },
        { id: 'inspect-permission', label: '查看权限影响', icon: ShieldCheck },
        { id: 'agent-shortcut', label: '加入 Agent 工作流', icon: AppWindow },
      ],
    });
    return;
  }

  openContextMenu({
    x: event.clientX,
    y: event.clientY,
    source: 'window',
    sourceId: id,
    title: getWindowTitle(id),
    subtitle: '窗口操作',
    items: [
      { id: 'window-front', label: '置于最前', icon: AppWindow },
      { id: 'window-minimize', label: '最小化', icon: Minimize2 },
      { id: 'window-maximize', label: maximizedWindowId.value === id ? '还原窗口' : '最大化', icon: Square },
      { id: 'window-reset', label: '重置位置和尺寸', icon: RotateCcw },
      { id: 'window-close', label: '关闭窗口', icon: X, danger: true },
    ],
  });
}

function arrangeDesktopWindows() {
  const nextGeometries: Record<string, Partial<WindowGeometry>> = {};
  openWindowIds.value.forEach((id, index) => {
    const config = desktopWindows.find((window) => window.id === id);
    if (!config) return;
    nextGeometries[id] = normalizeWindowGeometry(id, {
      x: 42 + (index % 4) * 34,
      y: 92 + (index % 5) * 28,
      width: config.width,
      height: config.height,
    });
  });
  windowGeometries.value = nextGeometries;
  windowLayerOrder.value = prioritizeOpenWindowOrder(openWindowIds.value, activeWindowId.value);
  maximizedWindowId.value = '';
  showToast('桌面窗口已重新排列。');
}

function arrangeDesktopIcons(mode: DesktopIconArrangeMode) {
  desktopIconPositions.value = createDesktopIconLayout(dockApps, mode);
  const messages: Record<DesktopIconArrangeMode, string> = {
    right: '桌面图标已整理到右侧。',
    left: '桌面图标已整理到左侧。',
    name: '桌面图标已按名称排序。',
    status: '桌面图标已按固定和运行状态排序。',
  };
  showToast(messages[mode]);
}

function moveDesktopIcon(payload: { id: string; x: number; y: number }) {
  desktopIconPositions.value = {
    ...desktopIconPositions.value,
    [payload.id]: {
      x: payload.x,
      y: payload.y,
    },
  };
}

function resetWindowAdjustments(id: string) {
  const { [id]: _removed, ...rest } = windowGeometries.value;
  windowGeometries.value = rest;
  if (maximizedWindowId.value === id) {
    maximizedWindowId.value = '';
  }
  showToast(`${getWindowTitle(id)} 已恢复默认位置和尺寸。`);
}

function moveWindow(id: string, geometry: Pick<WindowGeometry, 'x' | 'y'>) {
  windowGeometries.value = {
    ...windowGeometries.value,
    [id]: normalizeWindowGeometry(id, {
      ...windowGeometries.value[id],
      ...geometry,
    }),
  };
}

function resizeWindow(id: string, geometry: WindowGeometry) {
  windowGeometries.value = {
    ...windowGeometries.value,
    [id]: normalizeWindowGeometry(id, {
      ...windowGeometries.value[id],
      ...geometry,
    }),
  };
}

function pinDockApp(id: string) {
  if (!pinnedDockAppIds.value.includes(id)) {
    pinnedDockAppIds.value = [...pinnedDockAppIds.value, id];
  }
  if (!dockOrder.value.includes(id)) {
    dockOrder.value = [...dockOrder.value, id];
  }
  showToast(`${getWindowTitle(id)} 已固定到 Dock。`);
}

function unpinDockApp(id: string) {
  pinnedDockAppIds.value = pinnedDockAppIds.value.filter((appId) => appId !== id);
  showToast(`${getWindowTitle(id)} 已取消固定。${isDockAppRunning(id) ? '关闭后会从 Dock 消失。' : ''}`);
}

function removeDockApp(id: string) {
  pinnedDockAppIds.value = pinnedDockAppIds.value.filter((appId) => appId !== id);
  showToast(`${getWindowTitle(id)} 已从 Dock 移除。${isDockAppRunning(id) ? '当前仍在运行，关闭后不再保留图标。' : ''}`);
}

function closeDockApp(id: string) {
  if (isDesktopWindow(id)) {
    closeWindow(id);
    showToast(`${getWindowTitle(id)} 已关闭。`);
    return;
  }


  if (utilityAppId.value === id) {
    utilityAppId.value = '';
    launchProgress.value = 0;
    showToast(`${getWindowTitle(id)} 已退出。`);
  }
}

function reorderDockApp(id: string, direction: -1 | 1) {
  const orderedVisibleIds = visibleDockApps.value.map((app) => app.id);
  const visibleIndex = orderedVisibleIds.indexOf(id);
  const swapVisibleId = orderedVisibleIds[visibleIndex + direction];
  if (!swapVisibleId) return;

  const currentIndex = dockOrder.value.indexOf(id);
  const swapIndex = dockOrder.value.indexOf(swapVisibleId);
  if (currentIndex < 0 || swapIndex < 0) return;

  const nextOrder = [...dockOrder.value];
  [nextOrder[currentIndex], nextOrder[swapIndex]] = [nextOrder[swapIndex], nextOrder[currentIndex]];
  dockOrder.value = nextOrder;
  showToast(`${getWindowTitle(id)} 已${direction < 0 ? '左移' : '右移'}。`);
}

function restoreDefaultDock() {
  dockOrder.value = dockApps.map((app) => app.id);
  pinnedDockAppIds.value = [...defaultPinnedDockAppIds];
  showToast('Dock 已恢复默认固定和排序。');
}

function runContextTargetAction() {
  const target = contextTarget.value;
  if (!target) {
    showToast('当前右键目标已不可用。');
    return;
  }
  if (target instanceof HTMLInputElement || target instanceof HTMLTextAreaElement || target instanceof HTMLSelectElement) {
    target.focus();
    showToast('已聚焦到该输入控件。');
    return;
  }
  target.click();
  showToast(`已执行：${getInteractiveLabel(target)}`);
}

async function copyContextTargetLabel() {
  const label = contextTarget.value ? getInteractiveLabel(contextTarget.value) : contextMenu.value.title;
  await navigator.clipboard?.writeText(label);
  showToast(`已复制：${label}`);
}

function handleContextMenuAction(actionId: string) {
  const sourceId = contextMenu.value.sourceId;

  if (actionId === 'new-folder') showToast('已在家庭空间创建新文件夹入口。');
  if (actionId === 'arrange-icons-right') arrangeDesktopIcons('right');
  if (actionId === 'arrange-icons-left') arrangeDesktopIcons('left');
  if (actionId === 'sort-icons-name') arrangeDesktopIcons('name');
  if (actionId === 'sort-icons-status') arrangeDesktopIcons('status');
  if (actionId === 'arrange-desktop') arrangeDesktopWindows();
  if (actionId === 'refresh-desktop') showToast('桌面状态已刷新。');
  if (actionId === 'open-system-settings') openApp('system-settings');
  if (actionId === 'dock-reset') restoreDefaultDock();

  if (sourceId && actionId === 'dock-open') openApp(sourceId);
  if (sourceId && actionId === 'dock-close') closeDockApp(sourceId);
  if (sourceId && actionId === 'dock-minimize') minimizeWindow(sourceId);
  if (sourceId && actionId === 'dock-maximize') toggleMaximizeWindow(sourceId);
  if (sourceId && actionId === 'dock-pin-toggle') {
    if (pinnedDockAppIds.value.includes(sourceId)) unpinDockApp(sourceId);
    else pinDockApp(sourceId);
  }
  if (sourceId && actionId === 'dock-remove') removeDockApp(sourceId);
  if (sourceId && actionId === 'dock-move-left') reorderDockApp(sourceId, -1);
  if (sourceId && actionId === 'dock-move-right') reorderDockApp(sourceId, 1);

  if (sourceId && actionId === 'window-front') bringWindowToFront(sourceId);
  if (sourceId && actionId === 'window-minimize') minimizeWindow(sourceId);
  if (sourceId && actionId === 'window-maximize') toggleMaximizeWindow(sourceId);
  if (sourceId && actionId === 'window-reset') resetWindowAdjustments(sourceId);
  if (sourceId && actionId === 'window-close') closeWindow(sourceId);

  if (actionId === 'run-interactive') runContextTargetAction();
  if (actionId === 'copy-interactive-label') void copyContextTargetLabel();
  if (actionId === 'inspect-permission') showToast('已打开该操作的权限影响摘要。');
  if (actionId === 'agent-shortcut') showToast('已将该操作加入 Agent 工作流草稿。');

  closeContextMenu();
}

function handleGlobalClick() {
  closeContextMenu();
}

function handleGlobalKeydown(event: KeyboardEvent) {
  if (event.key === 'Escape') closeContextMenu();
}

// openAppFrame registers (once) a synthetic window for an installed third-party
// app and opens it. The window renders AppFrameWindow via the dynamic branch in
// the template — no per-app code in App.vue.
function openAppFrame(payload: { id: string; name: string; src: string }) {
  appFrameSources.value = {
    ...appFrameSources.value,
    [payload.id]: { src: payload.src, name: payload.name },
  };
  if (!desktopWindows.some((window) => window.id === payload.id)) {
    desktopWindows.push({
      id: payload.id,
      title: payload.name,
      subtitle: payload.src,
      status: '运行中',
      statusTone: 'green',
      x: 260,
      y: 130,
      width: 900,
      height: 620,
      z: 20,
    });
  }
  if (!openWindowIds.value.includes(payload.id)) {
    openWindowIds.value.push(payload.id);
    bumpWindowOpenToken(payload.id);
  }
  minimizedWindowIds.value = minimizedWindowIds.value.filter((windowId) => windowId !== payload.id);
  windowGeometries.value = {
    ...windowGeometries.value,
    [payload.id]: normalizeWindowGeometry(payload.id, windowGeometries.value[payload.id]),
  };
  bringWindowToFront(payload.id);
}

function openApp(id: string) {
  const isWindow = desktopWindows.some((window) => window.id === id);

  const wasOpen = openWindowIds.value.includes(id);
  if (isWindow && !wasOpen) {
    openWindowIds.value.push(id);
    bumpWindowOpenToken(id);
  }

  if (isWindow) {
    minimizedWindowIds.value = minimizedWindowIds.value.filter((windowId) => windowId !== id);
    utilityAppId.value = '';
    windowGeometries.value = {
      ...windowGeometries.value,
      [id]: normalizeWindowGeometry(id, windowGeometries.value[id]),
    };
    bringWindowToFront(id);
    activityStore.recordPage(id, getWindowTitle(id));
    return;
  }

  if (!isWindow) {
    utilityAppId.value = id;
    startUtilityLaunch(id);
    activityStore.record({ type: 'action', category: id, action: 'launch-utility', target: getWindowTitle(id) });
  }

  activeWindowId.value = id;
}

function closeWindow(id: string) {
  if (openWindowIds.value.includes(id)) {
    activityStore.record({ type: 'action', category: id, action: 'close-window', target: getWindowTitle(id) });
  }
  openWindowIds.value = openWindowIds.value.filter((windowId) => windowId !== id);
  minimizedWindowIds.value = minimizedWindowIds.value.filter((windowId) => windowId !== id);
  windowLayerOrder.value = windowLayerOrder.value.filter((windowId) => windowId !== id);
  const { [id]: _closedWindowToken, ...remainingWindowOpenTokens } = windowOpenTokens.value;
  windowOpenTokens.value = remainingWindowOpenTokens;
  if (maximizedWindowId.value === id) {
    maximizedWindowId.value = '';
  }
  if (activeWindowId.value === id) {
    activeWindowId.value = selectNextVisibleWindow(id);
  }
}

function minimizeWindow(id: string) {
  if (!minimizedWindowIds.value.includes(id)) {
    minimizedWindowIds.value.push(id);
  }
  if (maximizedWindowId.value === id) {
    maximizedWindowId.value = '';
  }
  activeWindowId.value = selectNextVisibleWindow(id);
  showToast('窗口已最小化，可从 Dock 重新打开。');
}

function toggleMaximizeWindow(id: string) {
  maximizedWindowId.value = maximizedWindowId.value === id ? '' : id;
  bringWindowToFront(id);
}

const activeUtilityApp = computed(() =>
  dockApps.find((app) => app.id === utilityAppId.value),
);

function showToast(message: string) {
  toast.show(message);
  // Toasts funnel most meaningful desktop actions, so mirror them to the log.
  activityStore.record({ type: 'action', category: 'desktop', action: 'ui-action', detail: message });
}

function startUtilityLaunch(id: string) {
  const app = dockApps.find((item) => item.id === id);
  launchProgress.value = 18;
  if (launchTimer) window.clearInterval(launchTimer);
  launchTimer = window.setInterval(() => {
    launchProgress.value = Math.min(100, launchProgress.value + 22);
    if (launchProgress.value >= 100 && launchTimer) {
      window.clearInterval(launchTimer);
      launchTimer = undefined;
      showToast(`${app?.name ?? '应用'} 已准备就绪`);
    }
  }, 220);
}

async function handleTopbarAction(action: string) {
  // Global-search actions use a "<verb>:<arg>" protocol.
  const sep = action.indexOf(':');
  const verb = sep >= 0 ? action.slice(0, sep) : action;
  const arg = sep >= 0 ? action.slice(sep + 1) : '';

  if (verb === 'app' && arg) {
    openApp(arg);
    return;
  }
  if (verb === 'ask' && arg) {
    openApp('ai-assistant');
    void assistantStore.ask(arg);
    return;
  }
  if (action === 'reindex') {
    showToast('正在重新索引文件…');
    try {
      const res = (await apiClient.ai.reindexFiles()) as { indexed?: number };
      showToast(`索引完成：已处理 ${res.indexed ?? 0} 个文件。`);
    } catch (error) {
      showToast(`索引失败：${error instanceof Error ? error.message : '未知错误'}`);
    }
    return;
  }
  if (action === 'inspect') {
    openApp('security-center');
    try {
      const risks = await apiClient.security.inspect();
      showToast(`安全巡检完成，共 ${risks.length} 项风险。`);
    } catch (error) {
      showToast(`巡检失败：${error instanceof Error ? error.message : '未知错误'}`);
    }
    return;
  }

  const messages: Record<string, string> = {
    permissions: '已打开家庭空间权限概览',
    models: '模型策略已切换到设置视图',
    logout: '桌面会话已进入锁定确认',
    notice: '通知已标记为已读',
  };
  showToast(messages[action] ?? '操作已执行');
}

function updateCompactState() {
  isCompact.value = window.innerWidth <= compactBreakpointWidth || window.innerHeight <= compactBreakpointHeight;
}

function handleViewportResize() {
  updateCompactState();
  normalizeOpenWindowGeometries();
  const defaultIconLayout = createDesktopIconLayout(dockApps, 'left');
  const currentIconLayout = filterIconPositions(desktopIconPositions.value, new Set(dockApps.map((app) => app.id)));
  desktopIconPositions.value = isSideDock.value ? defaultIconLayout : { ...defaultIconLayout, ...currentIconLayout };
}

watch([uiDockPosition, uiDockIconSize], () => {
  normalizeOpenWindowGeometries();
  desktopIconPositions.value = createDesktopIconLayout(dockApps, 'left');
});

watch(
  [
    openWindowIds,
    minimizedWindowIds,
    activeWindowId,
    utilityAppId,
    assistantVisible,
    isCompact,
    maximizedWindowId,
    dockOrder,
    pinnedDockAppIds,
    desktopIconPositions,
    windowGeometries,
  ],
  queueSessionSave,
  { deep: true },
);

onMounted(() => {
  handleViewportResize();
  void settingsStore.loadSettings();
  void loadDesktopBootstrapFromApi();
  window.addEventListener('resize', handleViewportResize);
  window.addEventListener('click', handleGlobalClick);
  window.addEventListener('keydown', handleGlobalKeydown);
});

onUnmounted(() => {
  window.removeEventListener('resize', handleViewportResize);
  window.removeEventListener('click', handleGlobalClick);
  window.removeEventListener('keydown', handleGlobalKeydown);
  if (launchTimer) window.clearInterval(launchTimer);
  if (sessionSaveTimer) window.clearTimeout(sessionSaveTimer);
});
</script>

<template>
  <main
    class="desktop"
    :class="desktopUiClasses"
    :style="desktopUiStyle"
    @contextmenu.prevent="openDesktopContextMenu"
  >
    <TopBar @topbar-action="handleTopbarAction" />

    <section class="desktop__stage" aria-label="HiGoOS PC 桌面">
      <DesktopWidgets />
      <DesktopAppGrid
        :apps="dockApps"
        :positions="desktopIconPositions"
        :active-id="activeWindowId"
        :running-ids="runningDockAppIds"
        :pinned-ids="pinnedDockAppIds"
        @open-app="openApp"
        @move-app="moveDesktopIcon"
        @contextmenu-app="openDesktopAppContextMenu"
      />

      <DesktopWindow
        v-for="window in openWindows"
        :key="window.id"
        :window="window"
        :active="activeWindowId === window.id"
        :maximized="maximizedWindowId === window.id"
        @focus="bringWindowToFront(window.id)"
        @close="closeWindow(window.id)"
        @minimize="minimizeWindow(window.id)"
        @toggle-maximize="toggleMaximizeWindow(window.id)"
        @move-window="moveWindow(window.id, $event)"
        @resize-window="resizeWindow(window.id, $event)"
        @contextmenu-window="openWindowContextMenu($event, window.id)"
      >
        <FileManagerWindow v-if="window.id === 'file-manager'" />
        <AiAssistantWindow v-else-if="window.id === 'ai-assistant'" />
        <AiStewardWindow v-else-if="window.id === 'ai-file-steward'" />
        <AgentWorkbenchWindow v-else-if="window.id === 'agent-workbench'" />
        <StorageMonitorWindow v-else-if="window.id === 'storage-monitor'" />
        <BackupSyncWindow v-else-if="window.id === 'backup-sync'" />
        <PhotoMediaWindow v-else-if="window.id === 'photo-media'" />
        <MusicCenterWindow v-else-if="window.id === 'music-center'" />
        <VideoCenterWindow v-else-if="window.id === 'video-center'" />
        <DownloadCenterWindow v-else-if="window.id === 'download-center'" />
        <AppCenterWindow v-else-if="window.id === 'app-center'" @open-frame="openAppFrame" />
        <AppFrameWindow
          v-else-if="appFrameSources[window.id]"
          :src="appFrameSources[window.id].src"
          :name="appFrameSources[window.id].name"
        />
        <DockerWindow v-else-if="window.id === 'docker'" :open-token="windowOpenTokens[window.id] ?? 0" />
        <SecurityCenterWindow v-else-if="window.id === 'security-center'" />
        <DeviceMonitorWindow v-else-if="window.id === 'device-monitor'" />
        <HardwareCenterWindow v-else-if="window.id === 'hardware-center'" />
        <SystemSettingsWindow v-else-if="window.id === 'system-settings'" />
        <TaskCenterWindow v-else-if="window.id === 'task-center'" />
        <RemoteAccessWindow v-else-if="window.id === 'remote-access'" />
        <ProtocolsWindow v-else-if="window.id === 'file-protocols'" />
        <FeatureModuleWindow
          v-else-if="featureModuleByWindowId[window.id]"
          :module-key="featureModuleByWindowId[window.id]"
          :title="window.title"
          :subtitle="window.subtitle"
        />
      </DesktopWindow>

      <section v-if="activeUtilityApp" class="utility-launcher" aria-label="应用启动反馈">
        <img :src="activeUtilityApp.icon" :alt="activeUtilityApp.name" />
        <div>
          <p>已打开</p>
          <h2>{{ activeUtilityApp.name }}</h2>
          <span>正在加载运行视图和权限策略。</span>
          <div class="utility-launcher__progress" aria-label="应用启动进度">
            <i :style="{ width: `${launchProgress}%` }" />
          </div>
        </div>
        <button type="button" @click="utilityAppId = ''">收起</button>
      </section>
    </section>

    <DesktopDock
      :apps="visibleDockApps"
      :active-id="activeWindowId"
      :running-ids="runningDockAppIds"
      :pinned-ids="pinnedDockAppIds"
      :position="uiDockPosition"
      :dock-style="uiDockStyle"
      :icon-size="uiDockIconSize"
      @open-app="openApp"
      @contextmenu-app="openDockContextMenu"
    />

    <DesktopContextMenu
      v-if="contextMenu.visible"
      :x="contextMenu.x"
      :y="contextMenu.y"
      :title="contextMenu.title"
      :subtitle="contextMenu.subtitle"
      :items="contextMenu.items"
      @select="handleContextMenuAction"
      @close="closeContextMenu"
    />

    <UiToastHost />
    <UiConfirmHost />
  </main>
</template>
