<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  Boxes,
  DownloadCloud,
  ExternalLink,
  Play,
  RefreshCw,
  RotateCcw,
  Search,
  ShieldCheck,
  Square,
  Tags,
  Trash2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type {
  AppAction,
  AppActionPreview,
  AppAuditRecord,
  AppCatalogEntry,
  AppCenterApp,
  AppConfigField,
  AppRegistry,
} from '../../api/types';
import { UiButton, UiInput, UiStat, UiStatGrid, UiTabs, UiToolbar, UiWindowPage } from '../ui';

const emit = defineEmits<{ (e: 'open-frame', payload: { id: string; name: string; src: string }): void }>();

const fallbackApps: AppCenterApp[] = [
  {
    id: 'home-assistant',
    name: 'Home Assistant',
    category: '智能家居',
    version: '2026.4.1',
    latestVersion: '2026.5.0',
    status: '需更新',
    description: '家庭自动化中枢，已接入本地 MQTT 与设备监控。',
    source: '官方镜像',
    risk: '低风险',
    resource: '1 CPU / 1024 MB',
    ports: ['8123/tcp'],
    installed: true,
    running: true,
    updateAvailable: true,
    webEntry: { port: 8123, path: '/', display: 'embed' },
  },
  {
    id: 'paperless',
    name: 'Paperless-ngx',
    category: '文档归档',
    version: '',
    latestVersion: '2.16.1',
    status: '可安装',
    description: '发票、合同和保修单 OCR 归档，可写入文件管理标签。',
    source: '社区精选',
    risk: '中风险',
    resource: '2 CPU / 1536 MB',
    ports: ['8000/tcp'],
    installed: false,
    running: false,
    updateAvailable: false,
    webEntry: { port: 8000, path: '/', display: 'embed' },
  },
];

type Tab = 'discover' | 'installed' | 'audit';

const tab = ref<Tab>('discover');
const apps = ref<AppCenterApp[]>(fallbackApps);
const catalog = ref<AppCatalogEntry[]>([]);
const audit = ref<AppAuditRecord[]>([]);
const registries = ref<AppRegistry[]>([]);
const query = ref('');
const activeCategory = ref('全部');
const selectedAppId = ref(fallbackApps[0].id);
const actionState = ref('应用中心正在连接后端套件目录。');
const registryUrl = ref('');

const tabs = [
  { key: 'discover', label: '发现' },
  { key: 'installed', label: '已安装' },
  { key: 'audit', label: '审计' },
];

// Governed action dialog state.
const dialog = ref<{
  open: boolean;
  app: AppCenterApp | null;
  action: AppAction;
  preview: AppActionPreview | null;
  config: Record<string, string>;
  fields: AppConfigField[];
  busy: boolean;
  error: string;
}>({ open: false, app: null, action: 'install', preview: null, config: {}, fields: [], busy: false, error: '' });

const manifestById = computed(() => {
  const map = new Map<string, AppCatalogEntry>();
  for (const entry of catalog.value) map.set(entry.manifest.id, entry);
  return map;
});

const categories = computed(() => ['全部', ...Array.from(new Set(apps.value.map((app) => app.category)))]);
const categoryTabs = computed(() => categories.value.map((category) => ({ key: category, label: category })));

const filteredApps = computed(() => {
  const keyword = query.value.trim().toLowerCase();
  return apps.value.filter((app) => {
    const categoryMatch = activeCategory.value === '全部' || app.category === activeCategory.value;
    const text = `${app.name} ${app.description} ${app.category}`.toLowerCase();
    return categoryMatch && (!keyword || text.includes(keyword));
  });
});

const installedApps = computed(() => apps.value.filter((app) => app.installed));
const selectedApp = computed(
  () => apps.value.find((app) => app.id === selectedAppId.value) ?? filteredApps.value[0] ?? apps.value[0],
);
const installedCount = computed(() => apps.value.filter((app) => app.installed).length);
const runningCount = computed(() => apps.value.filter((app) => app.running).length);
const updateCount = computed(() => apps.value.filter((app) => app.updateAvailable).length);

// 卡片三态:运行中 / 已安装(已停) / 可安装,外加"需更新"高亮。
function appState(app: AppCenterApp): 'running' | 'stopped' | 'available' {
  if (app.running) return 'running';
  if (app.installed) return 'stopped';
  return 'available';
}
function stateLabel(app: AppCenterApp): string {
  if (app.updateAvailable) return '需更新';
  return { running: '运行中', stopped: '已停止', available: '可安装' }[appState(app)];
}
function appInitial(app: AppCenterApp): string {
  return (app.name.trim()[0] ?? '#').toUpperCase();
}

async function loadAll() {
  try {
    const [nextApps, nextCatalog] = await Promise.all([apiClient.appCenter.getApps(), apiClient.appCenter.getCatalog()]);
    if (nextApps.length) {
      apps.value = nextApps;
      selectedAppId.value = nextApps.some((app) => app.id === selectedAppId.value) ? selectedAppId.value : nextApps[0].id;
    }
    catalog.value = nextCatalog;
    actionState.value = '应用目录已从后端同步，安装、更新和卸载会经过确认并写入审计。';
    void refreshAudit();
    void refreshRegistries();
  } catch (error) {
    actionState.value = `后端暂不可用，继续使用本地应用目录：${errorMessage(error)}`;
  }
}

async function refreshAudit() {
  try {
    audit.value = await apiClient.appCenter.getAudit();
  } catch {
    /* audit is best-effort */
  }
}

async function refreshRegistries() {
  try {
    registries.value = await apiClient.appCenter.getRegistries();
  } catch {
    /* registries optional */
  }
}

watch(activeCategory, () => {
  selectedAppId.value = filteredApps.value[0]?.id ?? selectedAppId.value;
});

function selectApp(id: string) {
  selectedAppId.value = id;
}

// ----- governed action flow: preview -> dialog -> confirm -----

async function beginAction(app: AppCenterApp, action: AppAction) {
  dialog.value = {
    open: true,
    app,
    action,
    preview: null,
    config: {},
    fields: action === 'install' ? manifestById.value.get(app.id)?.manifest.config ?? [] : [],
    busy: true,
    error: '',
  };
  // Seed config defaults.
  for (const field of dialog.value.fields) {
    dialog.value.config[field.key] = field.default ?? '';
  }
  try {
    dialog.value.preview = await apiClient.appCenter.preview(app.id, action, dialog.value.config);
  } catch (error) {
    dialog.value.error = errorMessage(error);
  } finally {
    dialog.value.busy = false;
  }
}

function closeDialog() {
  dialog.value.open = false;
}

async function confirmDialog() {
  const { app, action, preview, config } = dialog.value;
  if (!app || !preview) return;
  dialog.value.busy = true;
  dialog.value.error = '';
  try {
    const result = await apiClient.appCenter.confirm(app.id, action, preview.confirmationId, 'operator', config);
    replaceApp(result.app);
    actionState.value = result.message || `${app.name}：${action} 已确认。`;
    dialog.value.open = false;
    void refreshAudit();
  } catch (error) {
    dialog.value.error = errorMessage(error);
  } finally {
    dialog.value.busy = false;
  }
}

async function rollback(record: AppAuditRecord) {
  try {
    await apiClient.appCenter.rollback(record.id, 'operator');
    actionState.value = `已回滚：${record.appName} · ${record.action}`;
    await loadAll();
  } catch (error) {
    actionState.value = `回滚失败：${errorMessage(error)}`;
  }
}

function openFrame(app: AppCenterApp) {
  const src = frameSrc(app);
  if (!src) {
    actionState.value = `${app.name} 未声明 Web 入口，无法打开界面。`;
    return;
  }
  emit('open-frame', { id: `appwin:${app.id}`, name: app.name, src });
}

function frameSrc(app: AppCenterApp): string {
  const we = app.webEntry;
  if (!we || !we.port) return '';
  const proto = typeof window !== 'undefined' && window.location.protocol === 'https:' ? 'https:' : 'http:';
  const host = typeof window !== 'undefined' ? window.location.hostname || 'localhost' : 'localhost';
  const path = we.path && we.path.startsWith('/') ? we.path : `/${we.path ?? ''}`;
  return `${proto}//${host}:${we.port}${path}`;
}

async function refreshCatalog() {
  try {
    catalog.value = await apiClient.appCenter.refreshCatalog();
    actionState.value = '已刷新远程仓库目录。';
  } catch (error) {
    actionState.value = `刷新远程仓库失败：${errorMessage(error)}`;
  }
}

async function addRegistry() {
  const url = registryUrl.value.trim();
  if (!url) return;
  try {
    const name = url.replace(/^https?:\/\//, '').split('/')[0] || url;
    await apiClient.appCenter.addRegistry(name, url);
    registryUrl.value = '';
    actionState.value = `已添加仓库 ${name}。`;
    await Promise.all([refreshRegistries(), refreshCatalog()]);
    await loadAll();
  } catch (error) {
    actionState.value = `添加仓库失败：${errorMessage(error)}`;
  }
}

function replaceApp(app: AppCenterApp) {
  apps.value = apps.value.map((item) => (item.id === app.id ? app : item));
  selectedAppId.value = app.id;
}

function originLabel(app: AppCenterApp): string {
  const entry = manifestById.value.get(app.id);
  if (!entry) return '';
  if (entry.origin === 'builtin') return '内置';
  if (entry.origin === 'local') return '本地';
  return `仓库 ${entry.registry ?? ''}`.trim();
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

onMounted(loadAll);
</script>

<template>
  <div class="app-center">
    <UiWindowPage
      layout="master-detail"
      :icon="Boxes"
      title="应用中心"
      :subtitle="tabs.find((item) => item.key === tab)?.label"
      :status="actionState"
    >
      <template #toolbar>
        <UiToolbar>
          <template #start>
            <UiTabs v-model="tab" :tabs="tabs" variant="pill" size="sm" />
          </template>
        </UiToolbar>
      </template>

      <template v-if="tab === 'discover'" #nav>
        <aside class="app-center__catalog" aria-label="应用目录">
          <UiInput v-model="query" type="search" size="sm" :prefix-icon="Search" placeholder="搜索应用、分类或能力" />
          <nav class="app-center__cats" aria-label="应用分类">
            <button
              v-for="cat in categories"
              :key="cat"
              class="app-center__cat"
              :class="{ 'app-center__cat--active': activeCategory === cat }"
              type="button"
              @click="activeCategory = cat"
            >
              <Boxes :size="15" /><span>{{ cat }}</span>
            </button>
          </nav>
        </aside>
      </template>

      <template v-if="tab === 'discover'">
        <UiStatGrid min="160px">
          <UiStat :icon="Boxes" label="已安装" :value="installedCount" tone="primary" />
          <UiStat :icon="Play" label="运行中" :value="runningCount" tone="success" />
          <UiStat :icon="RefreshCw" label="可更新" :value="updateCount" tone="warning" />
        </UiStatGrid>

        <section class="app-center__cards" aria-label="应用卡片">
          <article
            v-for="app in filteredApps"
            :key="app.id"
            class="app-center__card"
            :class="{ 'app-center__card--active': app.id === selectedAppId }"
            tabindex="0"
            role="button"
            @click="selectApp(app.id)"
            @keydown.enter="selectApp(app.id)"
          >
            <header class="app-center__card-head">
              <span class="app-center__card-icon" :data-state="appState(app)">{{ appInitial(app) }}</span>
              <span class="app-center__badge" :data-state="appState(app)" :class="{ 'app-center__badge--update': app.updateAvailable }">
                {{ stateLabel(app) }}
              </span>
            </header>
            <strong class="app-center__card-name">{{ app.name }}</strong>
            <p class="app-center__card-desc">{{ app.description }}</p>
            <footer class="app-center__card-foot">
              <small>{{ app.category }} · v{{ app.version || app.latestVersion }}</small>
              <div class="app-center__card-actions" @click.stop>
                <UiButton v-if="!app.installed" variant="soft" tone="primary" size="sm" :icon-left="DownloadCloud" @click="beginAction(app, 'install')">安装</UiButton>
                <UiButton v-if="app.updateAvailable" variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="beginAction(app, 'update')">更新</UiButton>
                <UiButton v-if="app.installed && !app.running" variant="soft" tone="primary" size="sm" :icon-left="Play" @click="beginAction(app, 'start')">启动</UiButton>
                <UiButton v-if="app.running" variant="soft" size="sm" :icon-left="Square" @click="beginAction(app, 'stop')">停止</UiButton>
                <UiButton v-if="app.running && app.webEntry" variant="soft" tone="primary" size="sm" :icon-left="ExternalLink" @click="openFrame(app)">打开</UiButton>
              </div>
            </footer>
          </article>
          <p v-if="!filteredApps.length" class="app-center__empty">没有匹配的应用,换个分类或关键词试试。</p>
        </section>
      </template>

      <template v-if="tab === 'discover' && selectedApp" #inspector>
        <section class="app-center__detail" aria-label="应用详情">
          <header>
            <div>
              <p>{{ selectedApp.category }} · {{ selectedApp.source }}</p>
              <h3>{{ selectedApp.name }}</h3>
            </div>
            <strong>{{ selectedApp.status }}</strong>
          </header>

          <p class="app-center__description">{{ selectedApp.description }}</p>

          <div class="app-center__grid">
            <article><Tags :size="14" /><span>版本</span><strong>{{ selectedApp.version || '未安装' }} / {{ selectedApp.latestVersion }}</strong></article>
            <article><ShieldCheck :size="14" /><span>风险</span><strong>{{ selectedApp.risk }}</strong></article>
            <article><Boxes :size="14" /><span>资源</span><strong>{{ selectedApp.resource }}</strong></article>
          </div>

          <div v-if="selectedApp.permissions?.length" class="app-center__perms">
            <span v-for="perm in selectedApp.permissions" :key="perm">{{ perm }}</span>
          </div>

          <div class="app-center__ports" aria-label="端口">
            <span v-for="port in selectedApp.ports" :key="port">{{ port }}</span>
          </div>

          <div class="app-center__actions">
            <UiButton v-if="!selectedApp.installed" variant="soft" tone="primary" size="sm" :icon-left="DownloadCloud" @click="beginAction(selectedApp, 'install')">安装</UiButton>
            <UiButton v-if="selectedApp.updateAvailable" variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="beginAction(selectedApp, 'update')">更新</UiButton>
            <UiButton v-if="selectedApp.installed && !selectedApp.running" variant="soft" tone="primary" size="sm" :icon-left="Play" @click="beginAction(selectedApp, 'start')">启动</UiButton>
            <UiButton v-if="selectedApp.running" variant="soft" size="sm" :icon-left="Square" @click="beginAction(selectedApp, 'stop')">停止</UiButton>
            <UiButton v-if="selectedApp.running && selectedApp.webEntry" variant="soft" tone="primary" size="sm" :icon-left="ExternalLink" @click="openFrame(selectedApp)">打开</UiButton>
            <UiButton v-if="selectedApp.installed" variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="beginAction(selectedApp, 'uninstall')">卸载</UiButton>
          </div>
        </section>
      </template>

      <!-- 已安装 -->
      <section v-if="tab === 'installed'" class="app-center__installed">
        <p v-if="!installedApps.length" class="app-center__empty">尚无已安装应用，去“发现”页安装一个吧。</p>
        <article v-for="app in installedApps" :key="app.id" class="app-center__row">
          <div class="app-center__row-info">
            <strong>{{ app.name }}</strong>
            <span>{{ app.category }} · {{ app.status }} · v{{ app.version || app.latestVersion }}</span>
          </div>
          <div class="app-center__row-actions">
            <UiButton v-if="app.updateAvailable" variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="beginAction(app, 'update')">更新</UiButton>
            <UiButton v-if="!app.running" variant="soft" tone="primary" size="sm" :icon-left="Play" @click="beginAction(app, 'start')">启动</UiButton>
            <UiButton v-if="app.running" variant="soft" size="sm" :icon-left="Square" @click="beginAction(app, 'stop')">停止</UiButton>
            <UiButton v-if="app.webEntry" variant="soft" tone="primary" size="sm" :icon-left="ExternalLink" @click="openFrame(app)">打开</UiButton>
            <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="beginAction(app, 'uninstall')">卸载</UiButton>
          </div>
        </article>
      </section>

      <!-- 审计 -->
      <section v-else-if="tab === 'audit'" class="app-center__audit-log">
        <div class="app-center__registry">
          <UiInput v-model="registryUrl" size="sm" placeholder="远程仓库 index.json 地址" />
          <UiButton variant="soft" size="sm" :icon-left="DownloadCloud" @click="addRegistry">添加仓库</UiButton>
          <UiButton variant="soft" size="sm" :icon-left="RefreshCw" @click="refreshCatalog">刷新</UiButton>
        </div>
        <p v-if="!audit.length" class="app-center__empty">还没有审计记录。</p>
        <article v-for="record in audit" :key="record.id" class="app-center__row">
          <div class="app-center__row-info">
            <strong>{{ record.appName }} · {{ record.action }}</strong>
            <span>{{ record.message }}</span>
            <small>{{ record.actor }} · {{ record.risk }} · {{ record.result }}</small>
          </div>
          <div class="app-center__row-actions">
            <UiButton v-if="record.rollbackable" variant="soft" size="sm" :icon-left="RotateCcw" @click="rollback(record)">回滚</UiButton>
          </div>
        </article>
      </section>
    </UiWindowPage>

    <!-- 治理确认对话框 -->
    <div v-if="dialog.open" class="app-center__dialog" role="dialog" aria-modal="true">
      <div class="app-center__dialog-card">
        <header>
          <h4>{{ dialog.app?.name }} · {{ dialog.action }}</h4>
          <span class="app-center__risk">{{ dialog.preview?.risk || '…' }}</span>
        </header>

        <p class="app-center__impact">{{ dialog.busy && !dialog.preview ? '正在生成影响摘要…' : dialog.preview?.impact }}</p>

        <div v-if="dialog.fields.length" class="app-center__form">
          <label v-for="field in dialog.fields" :key="field.key">
            <span>{{ field.label }}<em v-if="field.required">*</em></span>
            <input
              v-model="dialog.config[field.key]"
              :type="field.secret ? 'password' : field.type === 'number' ? 'number' : 'text'"
              :placeholder="field.default"
            />
          </label>
        </div>

        <p v-if="dialog.error" class="app-center__error">{{ dialog.error }}</p>

        <footer>
          <UiButton variant="ghost" size="sm" @click="closeDialog">取消</UiButton>
          <UiButton
            variant="soft"
            :tone="dialog.action === 'uninstall' ? 'danger' : 'primary'"
            size="sm"
            :disabled="dialog.busy || !dialog.preview"
            @click="confirmDialog"
          >
            确认{{ dialog.action === 'uninstall' ? '卸载' : '执行' }}
          </UiButton>
        </footer>
      </div>
    </div>
  </div>
</template>

<style scoped>
.app-center {
  position: relative;
  height: 100%;
  min-height: 0;
}

.app-center__detail header p,
.app-center__grid span,
.app-center__item span,
.app-center__item small {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}

.app-center__grid strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.app-center__catalog,
.app-center__detail {
  min-width: 0;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.app-center__catalog {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 8px;
  overflow: hidden;
  height: 100%;
  padding: 10px;
}

.app-center__cats {
  display: grid;
  gap: 3px;
  overflow-y: auto;
  align-content: start;
}

.app-center__cat {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 36px;
  padding: 0 10px;
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-control);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard),
    color var(--duration-fast) var(--ease-standard);
}
.app-center__cat svg {
  flex: 0 0 auto;
  color: var(--text-soft);
}
.app-center__cat:hover {
  background: var(--accent-soft);
  color: var(--accent-deep);
}
.app-center__cat--active {
  color: var(--accent-deep);
  background: var(--accent-soft);
}
.app-center__cat--active svg {
  color: var(--accent);
}

.app-center__cards {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(240px, 1fr));
  gap: 12px;
  align-content: start;
}

.app-center__card {
  display: grid;
  gap: 8px;
  padding: 14px;
  text-align: left;
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard),
    border-color var(--duration-fast) var(--ease-standard),
    transform var(--duration-fast) var(--ease-standard),
    box-shadow var(--duration-fast) var(--ease-standard);
}
.app-center__card:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
.app-center__card--active {
  border-color: var(--accent);
  box-shadow: 0 0 0 1px var(--accent);
}

.app-center__card-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.app-center__card-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  color: var(--text-inverse);
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
  border-radius: var(--radius-control);
}
.app-center__card-icon[data-state='available'] {
  background: linear-gradient(135deg, var(--text-soft), var(--text-muted));
}

.app-center__badge {
  padding: 3px 9px;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  background: rgba(var(--surface-rgb), 0.82);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
}
.app-center__badge[data-state='running'] {
  color: var(--ink-green);
  background: var(--accent-green-soft);
  border-color: transparent;
}
.app-center__badge--update {
  color: var(--ink-orange);
  background: var(--accent-orange-soft);
  border-color: transparent;
}

.app-center__card-name {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.app-center__card-desc {
  display: -webkit-box;
  margin: 0;
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}
.app-center__card-foot {
  display: grid;
  gap: 8px;
  margin-top: 2px;
}
.app-center__card-foot small {
  overflow: hidden;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.app-center__card-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.app-center__list {
  display: grid;
  gap: 8px;
  overflow-y: auto;
  align-content: start;
}

.app-center__item {
  display: grid;
  gap: 5px;
  padding: 10px;
  text-align: left;
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  transition: background var(--duration-fast) var(--ease-standard),
    border-color var(--duration-fast) var(--ease-standard),
    transform var(--duration-fast) var(--ease-standard);
}

.app-center__item:hover {
  background: var(--accent-soft);
  transform: translateY(-1px);
}

.app-center__item:active {
  transform: translateY(0);
}

.app-center__item--active {
  border-color: var(--accent);
  box-shadow: inset 3px 0 0 var(--accent);
}

.app-center__item strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-center__detail {
  display: grid;
  align-content: start;
  overflow-y: auto;
}

.app-center__detail header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid var(--border);
}

.app-center__detail h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 18px;
}

.app-center__detail header strong {
  color: var(--accent);
  font-size: var(--fs-xs);
}

.app-center__description {
  margin: 0;
  padding: 13px 14px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: var(--lh-normal);
}

.app-center__grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
  padding: 0 14px 12px;
}

.app-center__grid article {
  display: grid;
  gap: 6px;
  min-width: 0;
  padding: 10px;
  color: var(--accent);
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.app-center__perms,
.app-center__ports {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  padding: 0 14px 12px;
}

.app-center__perms span {
  padding: 4px 8px;
  color: var(--text-soft);
  background: rgba(var(--surface-rgb), 0.7);
  border: 1px dashed var(--border);
  border-radius: var(--radius-pill);
  font-size: 10px;
}

.app-center__ports span {
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  font-size: 10px;
  font-weight: var(--fw-bold);
}

.app-center__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 14px;
}

.app-center__installed,
.app-center__audit-log {
  display: grid;
  gap: 8px;
  align-content: start;
  overflow-y: auto;
  min-height: 0;
}

.app-center__registry {
  display: flex;
  gap: 8px;
  align-items: center;
}

.app-center__row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 10px 12px;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.app-center__row-info {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.app-center__row-info strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.app-center__row-info span {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}

.app-center__row-info small {
  color: var(--text-soft);
  font-size: 10px;
}

.app-center__row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  justify-content: flex-end;
}

.app-center__empty {
  padding: 24px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  text-align: center;
}

.app-center__dialog {
  position: absolute;
  inset: 0;
  display: grid;
  place-items: center;
  padding: 16px;
  background: rgba(8, 16, 24, 0.42);
  border-radius: var(--radius-card);
  z-index: 20;
}

.app-center__dialog-card {
  display: grid;
  gap: 12px;
  width: min(420px, 100%);
  padding: 16px;
  background: var(--surface);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  box-shadow: 0 18px 48px rgba(8, 16, 24, 0.32);
}

.app-center__dialog-card header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.app-center__dialog-card h4 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.app-center__risk {
  padding: 3px 8px;
  color: var(--accent);
  background: var(--accent-soft);
  border-radius: var(--radius-pill);
  font-size: var(--fs-2xs);
}

.app-center__impact {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: var(--lh-normal);
}

.app-center__form {
  display: grid;
  gap: 10px;
}

.app-center__form label {
  display: grid;
  gap: 4px;
}

.app-center__form span {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}

.app-center__form em {
  color: var(--accent-red);
  font-style: normal;
}

.app-center__form input {
  padding: 7px 9px;
  color: var(--text-strong);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  font-size: var(--fs-xs);
}

.app-center__error {
  margin: 0;
  color: var(--accent-red);
  font-size: var(--fs-2xs);
}

.app-center__dialog-card footer {
  display: flex;
  justify-content: flex-end;
  gap: 8px;
}
</style>
