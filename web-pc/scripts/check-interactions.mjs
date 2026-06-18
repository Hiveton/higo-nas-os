import { readFile } from 'node:fs/promises';
import { resolve } from 'node:path';

const root = process.cwd();

const files = {
  index: 'index.html',
  app: 'src/App.vue',
  desktopStore: 'src/stores/desktop.ts',
  monitoringStore: 'src/stores/monitoring.ts',
  settingsStore: 'src/stores/settings.ts',
  remoteStore: 'src/stores/remote.ts',
  topbar: 'src/components/TopBar.vue',
  window: 'src/components/DesktopWindow.vue',
  desktopApps: 'src/components/DesktopAppGrid.vue',
  desktopWidgets: 'src/components/DesktopWidgets.vue',
  dock: 'src/components/DesktopDock.vue',
  contextMenu: 'src/components/DesktopContextMenu.vue',
  fileManager: 'src/components/windows/FileManagerWindow.vue',
  steward: 'src/components/windows/AiStewardWindow.vue',
  aiAssistant: 'src/components/windows/AiAssistantWindow.vue',
  agent: 'src/components/windows/AgentWorkbenchWindow.vue',
  storage: 'src/components/windows/StorageMonitorWindow.vue',
  photo: 'src/components/windows/PhotoMediaWindow.vue',
  download: 'src/components/windows/DownloadCenterWindow.vue',
  docker: 'src/components/windows/DockerWindow.vue',
  backup: 'src/components/windows/BackupSyncWindow.vue',
  appCenter: 'src/components/windows/AppCenterWindow.vue',
  security: 'src/components/windows/SecurityCenterWindow.vue',
  device: 'src/components/windows/DeviceMonitorWindow.vue',
  system: 'src/components/windows/SystemSettingsWindow.vue',
  remote: 'src/components/windows/RemoteAccessWindow.vue',
  baseCss: 'src/styles/base.css',
  viteConfig: 'vite.config.ts',
  favicon: 'public/favicon.svg',
};

const source = {};
await Promise.all(
  Object.entries(files).map(async ([key, file]) => {
    try {
      source[key] = await readFile(resolve(root, file), 'utf8');
    } catch (error) {
      if (error?.code !== 'ENOENT') throw error;
      source[key] = '';
    }
  }),
);

const checks = [
  ['App declares a served favicon asset', source.index.includes('href="/favicon.svg"') && source.favicon.includes('<svg')],
  ['App exposes toast feedback', source.app.includes('showToast') && source.app.includes('UiToastHost')],
  ['App supports utility launch progress', source.app.includes('launchProgress')],
  ['AI assistant window action buttons are wired', source.app.includes('AiAssistantWindow') && source.aiAssistant.includes('@confirm-action="confirmAction"') && source.aiAssistant.includes('assistantStore.confirmAction') === false && source.aiAssistant.includes('apiClient.assistant.confirmAction') && source.aiAssistant.includes('@send="handleSend"')],
  ['TopBar suggestions can be applied', source.topbar.includes('applySuggestion')],
  ['TopBar menu actions produce feedback', source.topbar.includes('emit(') && source.topbar.includes('topbar-action')],
  ['TopBar loads backend monitoring metrics and alerts', source.topbar.includes('monitoringStore.loadMonitoringSnapshot()') && source.topbar.includes('topbarMetrics') && source.monitoringStore.includes('loadMonitoringSnapshot')],
  ['Window minimize control is wired', source.window.includes("emit('minimize')")],
  ['Window maximize control is wired', source.window.includes("emit('toggle-maximize')")],
  ['Maximized windows stay between the top bar and Dock', source.window.includes('.desktop-window--maximized') && source.window.includes('calc(var(--topbar-height) + 30px)') && source.window.includes('calc(var(--dock-height) + 28px)')],
  ['Side Dock maximized-window rule never targets the desktop canvas directly', source.baseCss.includes('.desktop--dock-left .desktop-window--maximized') && source.baseCss.includes('.desktop--dock-right .desktop-window--maximized') && !source.window.includes(':global(.desktop--dock-left) .desktop-window--maximized') && !source.window.includes(':global(.desktop--dock-right) .desktop-window--maximized')],
  ['Compact mode keeps every non-minimized window visible', source.app.includes('return openWindowIds.value.filter((id) => !minimizedWindowIds.value.includes(id));') && !source.app.includes('compactActiveId')],
  ['Narrow viewports keep desktop multi-window geometry instead of forcing every window full-width', !/\\.desktop-window(?::not\\([^)]*\\))?\\s*{[^}]*width:\\s*calc\\(100vw - 28px\\)[^}]*height:\\s*calc\\(100vh - var\\(--dock-height\\) - 132px\\)/s.test(source.window)],
  ['Resize handles remain available in multi-window mode', !/desktop-window__resize-handle\\s*{[^}]*display:\\s*none/s.test(source.window)],
  ['Window focus is driven by pointer clicks for desktop-like stacking', source.window.includes("@pointerdown=\"emit('focus')\"") && source.app.includes('bringWindowToFront')],
  ['Window geometry can extend underneath the Dock overlay', source.app.includes('const bottom = 12') && source.window.includes('const bottom = 12')],
  ['Desktop resize is not overridden by short-height media locks', !source.window.includes('@media (max-height: 840px)') && !/max-height:\\s*calc\\(100vh - var\\(--dock-height\\) - 188px\\)/.test(source.window)],
  ['Window drag and resize capture and cancel pointer interactions', source.window.includes('setPointerCapture') && source.window.includes('pointercancel')],
  ['Window body scrolls real content instead of clipping dense apps', source.window.includes('overscroll-behavior: contain') && source.window.includes('scrollbar-gutter: stable')],
  ['Window content has global anti-overflow guards', source.baseCss.includes('.desktop-window__body :is(main, section, aside, article, div, nav, header, footer, ul, ol, li)') && source.baseCss.includes('overflow-wrap: anywhere')],
  ['Window content uses container queries for resizable app layouts', source.baseCss.includes('container-name: desktop-window-body') && source.baseCss.includes('@container desktop-window-body') && source.baseCss.includes('.download-center') && source.baseCss.includes('.device-monitor')],
  ['Narrow resizable windows collapse fixed sidebars instead of overflowing', source.baseCss.includes('grid-template-columns: minmax(0, 1fr)') && source.baseCss.includes('overflow-x: hidden') && source.baseCss.includes('.file-manager__details')],
  ['File manager folders change active folder', source.fileManager.includes('selectFolder')],
  ['File manager search filters rows', source.fileManager.includes('filteredFiles')],
  ['File manager has preview/share/tag actions', source.fileManager.includes('previewOpen') && source.fileManager.includes('shareOpen') && source.fileManager.includes('addSmartTag')],
  ['File manager upload shows progress and focuses uploaded files', source.fileManager.includes('uploadProgress') && source.fileManager.includes('goToUploadTarget') && source.fileManager.includes('uploadedRows')],
  ['AI steward actions mutate state', source.steward.includes('handleSuggestionAction') && source.steward.includes('dismissedSuggestions')],
  ['Agent templates are selectable', source.agent.includes('selectedTemplateIndex')],
  ['Agent simulation has progress state', source.agent.includes('simulationState')],
  ['Agent confirmation mutates execution state', source.agent.includes('confirmExecution')],
  ['Storage disks are selectable', source.storage.includes('selectedDisk')],
  ['Storage has scan/repair/snapshot actions', source.storage.includes('runSpaceAction')],
  ['Storage create wizard calls backend create and account grant APIs', source.storage.includes('createStorageSpace') && source.storage.includes('apiClient.storage.createSpace') && source.storage.includes('apiClient.accounts.grantSpace')],
  ['Storage window does not fall back to local fixture disks', !source.storage.includes('../../data/higoos') && source.storage.includes('ref<StoragePool[]>([])') && source.storage.includes('ref<Disk[]>([])')],
  ['Desktop storage widget does not fall back to local fixture pools', !source.desktopWidgets.includes('seedStoragePools') && source.desktopWidgets.includes('存储池不使用本地演示数据')],
  ['Dock hover magnification is not clipped by scroll overflow', source.dock.includes('dock__scroller') && !/\\.dock__surface\\s*{[^}]*overflow-x:\\s*auto/s.test(source.dock)],
  ['TopBar and Dock are positioned inside the desktop canvas', source.topbar.includes('position: absolute;') && source.dock.includes('position: absolute;')],
  ['Side Dock reserves a full floating rail before arranging desktop icons', source.app.includes('getSideDockReservedInset') && source.app.includes("uiDockPosition.value === 'left' ? sideDockInset + edgeInset : edgeInset")],
  ['Side Dock keeps a compact floating background centered in its reserved rail', source.dock.includes('top: calc(var(--topbar-height) + 56px);') && source.dock.includes('bottom: max(32px, env(safe-area-inset-bottom));') && source.dock.includes('max-height: 100%;') && source.dock.includes('justify-content: flex-start;') && source.dock.includes('min-height: 0;') && source.dock.includes('.dock--right .dock__surface::after') && source.dock.includes('display: none;')],
  ['Desktop exposes all application icons outside the Dock', source.app.includes('DesktopAppGrid') && source.app.includes(':apps="dockApps"') && source.desktopApps.includes('desktop-apps') && source.desktopApps.includes("emit('open-app'")],
  ['Desktop shell loads apps windows and session from API store', source.app.includes('desktopStore.loadDesktopBootstrap()') && source.app.includes('applyDesktopSession') && source.desktopStore.includes('loadDesktopBootstrap')],
  ['Desktop widgets sync backup docker and security summaries from APIs', source.desktopWidgets.includes('apiClient.backup.getJobs') && source.desktopWidgets.includes('apiClient.docker.getContainers') && source.desktopWidgets.includes('apiClient.security.getRiskActions')],
  ['Desktop session changes are persisted through the API store', source.app.includes('queueSessionSave') && source.app.includes('saveSessionPatch') && source.desktopStore.includes('saveSessionPatch')],
  ['Desktop normalizes window geometry to the visible viewport', source.app.includes('normalizeWindowGeometry') && source.app.includes('normalizeOpenWindowGeometries') && source.app.includes('handleViewportResize')],
  ['Desktop uses a frontmost window order instead of fixed z-only stacking', source.app.includes('windowLayerOrder') && source.app.includes('bringWindowToFront') && source.app.includes('selectNextVisibleWindow')],
  ['Desktop icons can be dragged and moved', source.app.includes('desktopIconPositions') && source.app.includes('moveDesktopIcon') && source.desktopApps.includes("emit('move-app'") && source.desktopApps.includes('startDragging')],
  ['Desktop icons default to left side layout', source.app.includes("createDesktopIconLayout(dockApps, 'left')")],
  ['Desktop blank context menu can sort and arrange icons', source.app.includes('arrangeDesktopIcons') && source.app.includes('sort-icons-name') && source.app.includes('sort-icons-status') && source.app.includes('arrange-icons-right')],
  ['Desktop icons do not show fixed or running text badges', !source.desktopApps.includes('desktop-app__meta') && !source.desktopApps.includes('固定</span>') && !source.desktopApps.includes('运行</span>')],
  ['Dock default state is locked subset plus running apps', source.app.includes('defaultPinnedDockAppIds') && source.app.includes('pinnedDockAppIds = ref([...defaultPinnedDockAppIds])') && source.app.includes('runningDockAppIds')],
  ['Dock has pinned running and temporary app state', source.app.includes('pinnedDockAppIds') && source.app.includes('visibleDockApps') && source.dock.includes('dock__item--running') && source.dock.includes('dock__item--temporary')],
  ['Dock context menu supports pin remove close and reorder', source.app.includes('pinDockApp') && source.app.includes('removeDockApp') && source.app.includes('closeDockApp') && source.app.includes('reorderDockApp')],
  ['Dock can restore default layout', source.app.includes('restoreDefaultDock') && source.app.includes('dock-reset')],
  ['Desktop exposes contextual right-click menus', source.app.includes('DesktopContextMenu') && source.app.includes('openDesktopContextMenu') && source.app.includes('@contextmenu')],
  ['Dock apps expose contextual right-click actions', source.dock.includes("contextmenu-app") && source.dock.includes("emit('contextmenu-app'")],
  ['Context menu supports desktop window and interactive targets', source.contextMenu.includes('ContextMenuItem') && source.contextMenu.includes('context-menu__item') && source.contextMenu.includes('role=\"menu\"')],
  ['Windows can be moved by titlebar drag', source.window.includes('startWindowDrag') && source.window.includes("emit('move-window'")],
  ['Windows can be resized from edges and corners', source.window.includes('startWindowResize') && source.window.includes('resizeHandleDirections') && source.window.includes("emit('resize-window'")],
  ['Photo media window has album interactions', source.photo.includes('selectedAlbum') && source.photo.includes('generateMemory') && source.photo.includes('mergePeople')],
  ['Download center has queue interactions', source.download.includes('addDownloadTask') && source.download.includes('toggleTask') && source.download.includes('archiveCompleted')],
  ['Docker window has container actions', source.docker.includes('selectedContainer') && source.docker.includes('restartContainer') && source.docker.includes('resourceLimit')],
  ['Docker window uses backend Docker API for stacks containers logs actions and limits', source.docker.includes('apiClient.docker.getStacks') && source.docker.includes('apiClient.docker.getContainers') && source.docker.includes('apiClient.docker.getContainerLogs') && source.docker.includes('apiClient.docker.updateContainerLimits')],
  ['Backup sync opens a backend-backed window', source.app.includes('BackupSyncWindow') && source.backup.includes('apiClient.backup.getJobs') && source.backup.includes('runBackupJob') && source.backup.includes('verifyBackupJob')],
  ['App center opens a backend-backed window', source.app.includes('AppCenterWindow') && source.appCenter.includes('apiClient.appCenter.getApps') && source.appCenter.includes('installApp') && source.appCenter.includes('updateApp')],
  ['Security center has risk governance actions', source.security.includes('riskFilter') && source.security.includes('revokeShare') && source.security.includes('rollbackAudit')],
  ['Device monitor has alert and metric actions', source.device.includes('selectedMetric') && source.device.includes('createAlert') && source.device.includes('muteAlert')],
  ['Device monitor uses backend monitoring store for dashboard state', source.device.includes('monitoringStore.loadMonitoringDashboard') && source.device.includes('monitoringStore.createAlert') && source.device.includes('monitoringStore.muteAlert') && source.monitoringStore.includes('loadMonitoringDashboard')],
  ['System settings has stateful settings actions', source.system.includes('activeCategory') && source.system.includes('saveSettings') && source.system.includes('restoreDefaults')],
  ['System settings uses backend settings store for save restore updates and backups', source.system.includes('settingsStore.loadSettings') && source.system.includes('settingsStore.saveSettings') && source.system.includes('settingsStore.restoreDefaults') && source.system.includes('settingsStore.checkUpdates') && source.settingsStore.includes('loadSettings')],
  ['Remote access has tunnel and MFA actions', source.remote.includes('remoteEnabled') && source.remote.includes('toggleMfa') && source.remote.includes('scanShareLinks')],
  ['Remote access uses backend remote store for channel devices policies MFA and scans', source.remote.includes('remoteStore.loadRemoteDashboard') && source.remote.includes('remoteStore.toggleMfa') && source.remote.includes('remoteStore.selectPolicy') && source.remote.includes('remoteStore.scanShareLinks') && source.remoteStore.includes('loadRemoteDashboard')],
  ['Vite dev server proxies API calls to the Go backend by default', source.viteConfig.includes("'/api'") && source.viteConfig.includes('target: env.VITE_HIGOOS_API_BASE_URL') && source.viteConfig.includes('127.0.0.1:18082')],
];

const failed = checks.filter(([, ok]) => !ok);

if (failed.length) {
  console.error('Missing interaction coverage:');
  for (const [name] of failed) {
    console.error(`- ${name}`);
  }
  process.exit(1);
}

console.log(`Interaction coverage OK: ${checks.length} checks`);
