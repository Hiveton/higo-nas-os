<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Boxes, CheckCircle2, DownloadCloud, Play, RefreshCw, Search, ShieldCheck, Square, Tags } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AppCenterApp } from '../../api/types';

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
  },
  {
    id: 'qdrant',
    name: 'Qdrant 向量库',
    category: 'AI 数据层',
    version: '1.14.0',
    latestVersion: '1.14.0',
    status: '运行中',
    description: '为文件语义搜索和 Agent 检索提供本地向量索引。',
    source: '官方镜像',
    risk: '中风险',
    resource: '2 CPU / 2048 MB',
    ports: ['6333/tcp'],
    installed: true,
    running: true,
    updateAvailable: false,
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
  },
];

const apps = ref<AppCenterApp[]>(fallbackApps);
const query = ref('');
const activeCategory = ref('全部');
const selectedAppId = ref(fallbackApps[0].id);
const actionState = ref('应用中心正在连接后端套件目录。');

const categories = computed(() => ['全部', ...Array.from(new Set(apps.value.map((app) => app.category)))]);
const filteredApps = computed(() => {
  const keyword = query.value.trim().toLowerCase();
  return apps.value.filter((app) => {
    const categoryMatch = activeCategory.value === '全部' || app.category === activeCategory.value;
    const text = `${app.name} ${app.description} ${app.category}`.toLowerCase();
    return categoryMatch && (!keyword || text.includes(keyword));
  });
});
const selectedApp = computed(() => apps.value.find((app) => app.id === selectedAppId.value) ?? filteredApps.value[0] ?? apps.value[0]);
const installedCount = computed(() => apps.value.filter((app) => app.installed).length);
const runningCount = computed(() => apps.value.filter((app) => app.running).length);
const updateCount = computed(() => apps.value.filter((app) => app.updateAvailable).length);

async function loadApps() {
  try {
    const nextApps = await apiClient.appCenter.getApps();
    if (nextApps.length) {
      apps.value = nextApps;
      selectedAppId.value = nextApps.some((app) => app.id === selectedAppId.value) ? selectedAppId.value : nextApps[0].id;
    }
    actionState.value = '应用目录已从后端同步，安装、更新和运行状态会写入审计。';
  } catch (error) {
    actionState.value = `后端暂不可用，继续使用本地应用目录：${errorMessage(error)}`;
  }
}

function selectCategory(category: string) {
  activeCategory.value = category;
  selectedAppId.value = filteredApps.value[0]?.id ?? selectedAppId.value;
}

function selectApp(id: string) {
  selectedAppId.value = id;
  actionState.value = `正在查看 ${selectedApp.value.name}。`;
}

async function installApp(id = selectedApp.value.id) {
  await mutateApp(id, () => apiClient.appCenter.installApp(id), '应用已安装并启动。');
}

async function updateApp(id = selectedApp.value.id) {
  await mutateApp(id, () => apiClient.appCenter.updateApp(id), '应用已更新到最新版本。');
}

async function startApp(id = selectedApp.value.id) {
  await mutateApp(id, () => apiClient.appCenter.startApp(id), '应用已启动。');
}

async function stopApp(id = selectedApp.value.id) {
  await mutateApp(id, () => apiClient.appCenter.stopApp(id), '应用已停止。');
}

async function mutateApp(id: string, request: () => Promise<AppCenterApp>, message: string) {
  try {
    const nextApp = await request();
    replaceApp(nextApp);
    actionState.value = `${nextApp.name}：${message}`;
  } catch (error) {
    actionState.value = `${apps.value.find((app) => app.id === id)?.name ?? '应用'} 操作失败：${errorMessage(error)}`;
  }
}

function replaceApp(app: AppCenterApp) {
  apps.value = apps.value.map((item) => (item.id === app.id ? app : item));
  selectedAppId.value = app.id;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

onMounted(loadApps);
</script>

<template>
  <div class="app-center">
    <section class="app-center__summary" aria-label="应用中心摘要">
      <article>
        <Boxes :size="18" />
        <span>已安装</span>
        <strong>{{ installedCount }}</strong>
      </article>
      <article>
        <Play :size="18" />
        <span>运行中</span>
        <strong>{{ runningCount }}</strong>
      </article>
      <article>
        <RefreshCw :size="18" />
        <span>可更新</span>
        <strong>{{ updateCount }}</strong>
      </article>
    </section>

    <main class="app-center__main">
      <aside class="app-center__catalog" aria-label="应用目录">
        <label class="app-center__search">
          <Search :size="14" />
          <input v-model="query" type="search" placeholder="搜索应用、分类或能力" />
        </label>
        <div class="app-center__tabs">
          <button
            v-for="category in categories"
            :key="category"
            type="button"
            :class="{ 'app-center__tab--active': category === activeCategory }"
            @click="selectCategory(category)"
          >
            {{ category }}
          </button>
        </div>
        <button
          v-for="app in filteredApps"
          :key="app.id"
          class="app-center__item"
          :class="{ 'app-center__item--active': app.id === selectedAppId }"
          type="button"
          @click="selectApp(app.id)"
        >
          <strong>{{ app.name }}</strong>
          <span>{{ app.category }} · {{ app.status }}</span>
          <small>{{ app.version || app.latestVersion }} · {{ app.risk }}</small>
        </button>
      </aside>

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
          <article>
            <Tags :size="14" />
            <span>版本</span>
            <strong>{{ selectedApp.version || '未安装' }} / {{ selectedApp.latestVersion }}</strong>
          </article>
          <article>
            <ShieldCheck :size="14" />
            <span>风险</span>
            <strong>{{ selectedApp.risk }}</strong>
          </article>
          <article>
            <Boxes :size="14" />
            <span>资源</span>
            <strong>{{ selectedApp.resource }}</strong>
          </article>
        </div>

        <div class="app-center__ports" aria-label="端口">
          <span v-for="port in selectedApp.ports" :key="port">{{ port }}</span>
        </div>

        <div class="app-center__actions">
          <button v-if="!selectedApp.installed" type="button" @click="installApp()">
            <DownloadCloud :size="14" /> 安装
          </button>
          <button v-if="selectedApp.updateAvailable" type="button" @click="updateApp()">
            <RefreshCw :size="14" /> 更新
          </button>
          <button v-if="selectedApp.installed && !selectedApp.running" type="button" @click="startApp()">
            <Play :size="14" /> 启动
          </button>
          <button v-if="selectedApp.running" type="button" @click="stopApp()">
            <Square :size="14" /> 停止
          </button>
        </div>
      </section>
    </main>

    <section class="app-center__audit" aria-label="应用中心操作反馈">
      <CheckCircle2 :size="15" />
      <span>{{ actionState }}</span>
    </section>
  </div>
</template>

<style scoped>
.app-center {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.app-center__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.app-center__summary article {
  display: grid;
  gap: 4px;
  justify-items: center;
  padding: 12px 8px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.56);
}

.app-center__summary span,
.app-center__detail header p,
.app-center__grid span,
.app-center__item span,
.app-center__item small {
  color: var(--text-soft);
  font-size: 11px;
}

.app-center__summary strong,
.app-center__grid strong {
  color: var(--text-strong);
  font-size: 13px;
}

.app-center__main {
  display: grid;
  grid-template-columns: 260px minmax(0, 1fr);
  gap: 12px;
  min-height: 0;
}

.app-center__catalog,
.app-center__detail,
.app-center__audit {
  min-width: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.app-center__catalog {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 8px;
  overflow: hidden;
  padding: 10px;
}

.app-center__search {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 32px;
  padding: 0 9px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.app-center__search input {
  min-width: 0;
  width: 100%;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
  font: inherit;
  font-size: 12px;
}

.app-center__tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.app-center__tabs button,
.app-center__actions button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.app-center__tab--active {
  color: #fff !important;
  background: var(--accent) !important;
  border-color: transparent !important;
}

.app-center__item {
  display: grid;
  gap: 5px;
  padding: 10px;
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.app-center__item--active {
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.app-center__item strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-center__detail {
  display: grid;
  grid-template-rows: auto auto auto auto auto;
  align-content: start;
  overflow: hidden;
}

.app-center__detail header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.app-center__detail h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 18px;
}

.app-center__detail header strong {
  color: var(--accent);
  font-size: 12px;
}

.app-center__description {
  margin: 0;
  padding: 13px 14px;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.55;
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
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.app-center__ports {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  padding: 0 14px 14px;
}

.app-center__ports span {
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
}

.app-center__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 14px;
}

.app-center__audit {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  color: var(--text-strong);
  font-size: 12px;
}
</style>
