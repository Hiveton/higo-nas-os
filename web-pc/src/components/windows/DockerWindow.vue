<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import {
  Activity,
  Boxes,
  CheckCircle2,
  CircleDot,
  Container,
  Database,
  FileText,
  Gauge,
  HardDrive,
  ListFilter,
  Network,
  Play,
  RefreshCw,
  RotateCw,
  Search,
  Server,
  Settings2,
  Square,
  TerminalSquare,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { ComposeStack, DockerContainer } from '../../api/types';

type ContainerStatus = '全部' | '运行中' | '已停止' | '重启中';
type DetailMode = '概览' | '日志' | '端口' | '挂载' | '环境';
type StartStopStatus = '运行中' | '已停止';

const composeStacks = ref<ComposeStack[]>([]);
const containers = ref<DockerContainer[]>([]);
const selectedStackName = ref('all');
const selectedContainerId = ref('');
const detailMode = ref<DetailMode>('概览');
const statusFilter = ref<ContainerStatus>('全部');
const query = ref('');
const loading = ref(false);
const actionState = ref('正在连接 Docker 后端。');
let refreshTimer: number | undefined;

const stackScopes = computed(() => {
  const allScope: ComposeStack = {
    name: 'all',
    status: runningCount.value ? '运行中' : '空',
    services: containers.value.length,
    ports: String(allExposedPorts.value.length),
    volume: '全部容器',
    network: 'all',
  };
  return [allScope, ...composeStacks.value];
});

const scopedContainers = computed(() => {
  if (selectedStackName.value === 'all') return containers.value;
  return containers.value.filter((item) => item.stack === selectedStackName.value);
});

const filteredContainers = computed(() => {
  const text = query.value.trim().toLowerCase();
  return scopedContainers.value.filter((item) => {
    const statusMatched = statusFilter.value === '全部' || item.status === statusFilter.value;
    const textMatched =
      !text ||
      [item.name, item.image, item.stack, item.status, ...item.ports, ...item.mounts]
        .join(' ')
        .toLowerCase()
        .includes(text);
    return statusMatched && textMatched;
  });
});

const selectedContainer = computed(() => containers.value.find((item) => item.id === selectedContainerId.value));
const runningCount = computed(() => containers.value.filter((item) => item.status === '运行中').length);
const stoppedCount = computed(() => containers.value.filter((item) => item.status === '已停止').length);
const restartingCount = computed(() => containers.value.filter((item) => item.status === '重启中').length);
const totalCpu = computed(() => Math.min(100, containers.value.reduce((sum, item) => sum + Number(item.cpu || 0), 0)));
const totalMemory = computed(() => Math.min(100, Math.round(containers.value.reduce((sum, item) => sum + Number(item.memory || 0), 0))));
const imageCount = computed(() => new Set(containers.value.map((item) => item.image)).size);
const allExposedPorts = computed(() => containers.value.flatMap((item) => item.ports).filter((port) => port && port !== '未暴露端口'));
const statusCounts = computed<Record<ContainerStatus, number>>(() => ({
  全部: scopedContainers.value.length,
  运行中: scopedContainers.value.filter((item) => item.status === '运行中').length,
  已停止: scopedContainers.value.filter((item) => item.status === '已停止').length,
  重启中: scopedContainers.value.filter((item) => item.status === '重启中').length,
}));

const selectedStack = computed(() =>
  selectedStackName.value === 'all'
    ? stackScopes.value[0]
    : composeStacks.value.find((stack) => stack.name === selectedStackName.value) ?? stackScopes.value[0],
);

function selectStack(name: string) {
  selectedStackName.value = name;
  const first = scopedContainers.value[0];
  selectedContainerId.value = first?.id ?? '';
  if (first) void refreshSelectedContainerLogs(first.id);
}

async function selectContainer(id: string) {
  selectedContainerId.value = id;
  await refreshSelectedContainerLogs(id);
}

async function loadDockerRuntime(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [nextStacks, nextContainers] = await Promise.all([
      apiClient.docker.getStacks(),
      apiClient.docker.getContainers(),
    ]);
    composeStacks.value = nextStacks;
    containers.value = nextContainers.map((container) => withContainerLog(container));
    reconcileSelection();
    if (selectedContainerId.value) await refreshSelectedContainerLogs(selectedContainerId.value);
    actionState.value = containers.value.length
      ? `已同步 ${containers.value.length} 个容器、${composeStacks.value.length} 个栈。`
      : 'Docker 已连接，但当前没有容器。';
  } catch (error) {
    composeStacks.value = [];
    containers.value = [];
    selectedContainerId.value = '';
    actionState.value = `Docker 后端不可用：${errorMessage(error)}`;
  } finally {
    loading.value = false;
  }
}

function reconcileSelection() {
  if (!stackScopes.value.some((stack) => stack.name === selectedStackName.value)) {
    selectedStackName.value = 'all';
  }
  const visible = scopedContainers.value;
  if (!visible.some((container) => container.id === selectedContainerId.value)) {
    selectedContainerId.value = visible[0]?.id ?? containers.value[0]?.id ?? '';
  }
}

async function refreshSelectedContainerLogs(containerId = selectedContainerId.value) {
  if (!containerId) return;
  try {
    const logs = await apiClient.docker.getContainerLogs(containerId, 80);
    const current = containers.value.find((container) => container.id === containerId);
    if (current) replaceContainer(withContainerLog(current, logs));
  } catch {
    // Container lists must remain usable even if a stopped container has no logs.
  }
}

async function setContainerStatus(status: StartStopStatus) {
  const current = selectedContainer.value;
  if (!current) return;
  const next =
    status === '运行中'
      ? await apiClient.docker.startContainer(current.id)
      : await apiClient.docker.stopContainer(current.id);
  replaceContainer(withContainerLog(next));
  await refreshSelectedContainerLogs(next.id);
  actionState.value = `${next.name} 已${status === '运行中' ? '启动' : '停止'}。`;
}

async function restartContainer() {
  const current = selectedContainer.value;
  if (!current) return;
  const next = await apiClient.docker.restartContainer(current.id);
  replaceContainer(withContainerLog(next));
  await refreshSelectedContainerLogs(next.id);
  actionState.value = `${next.name} 已重启。`;
}

async function updateLimit(kind: 'cpu' | 'memory') {
  const current = selectedContainer.value;
  if (!current) return;
  const next = await apiClient.docker.updateContainerLimits(current.id, {
    limitCpu: current.limitCpu,
    limitMemory: current.limitMemory,
  });
  replaceContainer(withContainerLog(next));
  actionState.value =
    kind === 'cpu'
      ? `${next.name} CPU 限制调整为 ${next.limitCpu} 核。`
      : `${next.name} 内存限制调整为 ${next.limitMemory} MB。`;
}

function replaceContainer(container: DockerContainer) {
  containers.value = containers.value.map((item) => (item.id === container.id ? container : item));
  selectedContainerId.value = container.id;
}

function withContainerLog(container: DockerContainer, logs?: string[]): DockerContainer {
  const previous = containers.value.find((item) => item.id === container.id);
  return {
    ...container,
    ports: container.ports?.length ? container.ports : ['未暴露端口'],
    mounts: container.mounts?.length ? container.mounts : ['无挂载'],
    env: container.env?.length ? container.env : ['无环境变量'],
    log: logs ?? container.log ?? previous?.log ?? [],
  };
}

function statusTone(status: string) {
  if (status === '运行中') return 'running';
  if (status === '重启中') return 'restarting';
  return 'stopped';
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

onMounted(() => {
  void loadDockerRuntime();
  refreshTimer = window.setInterval(() => {
    void loadDockerRuntime(true);
  }, 10000);
});

onUnmounted(() => {
  if (refreshTimer !== undefined) {
    window.clearInterval(refreshTimer);
    refreshTimer = undefined;
  }
});
</script>

<template>
  <div class="docker-window">
    <aside class="docker-window__sidebar" aria-label="Docker 资源范围">
      <header>
        <h3><Boxes :size="15" /> Docker</h3>
        <button type="button" :disabled="loading" @click="loadDockerRuntime()">
          <RefreshCw :size="13" />刷新
        </button>
      </header>

      <div class="docker-window__scope-list">
        <button
          v-for="stack in stackScopes"
          :key="stack.name"
          class="docker-window__scope"
          :class="{ 'docker-window__scope--active': stack.name === selectedStackName }"
          type="button"
          @click="selectStack(stack.name)"
        >
          <span>
            <strong>{{ stack.name === 'all' ? '全部容器' : stack.name }}</strong>
            <small>{{ stack.services }} 个容器 · {{ stack.status }}</small>
          </span>
          <b>{{ stack.name === 'all' ? runningCount : stack.services }}</b>
        </button>
      </div>

      <section class="docker-window__quick-stats">
        <div><strong>{{ runningCount }}</strong><span>运行</span></div>
        <div><strong>{{ stoppedCount }}</strong><span>停止</span></div>
        <div><strong>{{ imageCount }}</strong><span>镜像</span></div>
        <div><strong>{{ allExposedPorts.length }}</strong><span>端口</span></div>
      </section>
    </aside>

    <main class="docker-window__workspace">
      <section class="docker-window__topline" aria-label="资源概览">
        <article>
          <Container :size="18" />
          <span>容器</span>
          <strong>{{ runningCount }}/{{ containers.length }}</strong>
        </article>
        <article>
          <Gauge :size="18" />
          <span>CPU</span>
          <strong>{{ totalCpu }}%</strong>
        </article>
        <article>
          <Activity :size="18" />
          <span>内存</span>
          <strong>{{ totalMemory }}%</strong>
        </article>
        <article>
          <Network :size="18" />
          <span>端口</span>
          <strong>{{ allExposedPorts.length }}</strong>
        </article>
        <article>
          <HardDrive :size="18" />
          <span>栈</span>
          <strong>{{ composeStacks.length }}</strong>
        </article>
      </section>

      <section class="docker-window__toolbar" aria-label="筛选和搜索">
        <div class="docker-window__search">
          <Search :size="15" />
          <input v-model="query" type="search" placeholder="搜索容器、镜像、端口、路径" />
        </div>
        <div class="docker-window__filters">
          <button
            v-for="status in (['全部', '运行中', '已停止', '重启中'] as ContainerStatus[])"
            :key="status"
            type="button"
            :class="{ 'docker-window__filter--active': statusFilter === status }"
            @click="statusFilter = status"
          >
            {{ status }} {{ statusCounts[status] }}
          </button>
        </div>
      </section>

      <section class="docker-window__table-card" aria-label="容器列表">
        <header>
          <div>
            <h3>{{ selectedStack.name === 'all' ? '容器列表' : selectedStack.name }}</h3>
            <p>{{ selectedStack.volume }} · {{ selectedStack.network }}</p>
          </div>
          <span>{{ filteredContainers.length }} / {{ scopedContainers.length }}</span>
        </header>

        <div class="docker-window__table-head">
          <span>容器</span>
          <span>状态</span>
          <span>镜像</span>
          <span>CPU</span>
          <span>内存</span>
          <span>端口</span>
        </div>

        <div class="docker-window__table">
          <button
            v-for="container in filteredContainers"
            :key="container.id"
            class="docker-window__row"
            :class="{ 'docker-window__row--active': container.id === selectedContainerId }"
            type="button"
            @click="selectContainer(container.id)"
          >
            <span>
              <strong>{{ container.name }}</strong>
              <small>{{ container.stack }}</small>
            </span>
            <b :class="`docker-window__status docker-window__status--${statusTone(container.status)}`">
              <CircleDot :size="10" />{{ container.status }}
            </b>
            <span>{{ container.image }}</span>
            <span>{{ container.cpu }}%</span>
            <span>{{ container.memoryText }}</span>
            <span>{{ container.ports[0] }}</span>
          </button>

          <div v-if="filteredContainers.length === 0" class="docker-window__empty">
            <Container :size="22" />
            <strong>没有匹配的容器</strong>
            <span>调整搜索或状态筛选后重试。</span>
          </div>
        </div>
      </section>

      <section v-if="selectedContainer" class="docker-window__container-detail" aria-label="选中容器详情">
        <header>
          <div>
            <h3>{{ selectedContainer.name }}</h3>
            <p>{{ selectedContainer.id }} · {{ selectedContainer.image }}</p>
          </div>
          <div class="docker-window__actions">
            <button v-if="selectedContainer.status !== '运行中'" type="button" @click="setContainerStatus('运行中')">
              <Play :size="13" /> 启动
            </button>
            <button v-if="selectedContainer.status === '运行中'" type="button" @click="setContainerStatus('已停止')">
              <Square :size="13" /> 停止
            </button>
            <button type="button" @click="restartContainer">
              <RotateCw :size="13" /> 重启
            </button>
          </div>
        </header>

        <div class="docker-window__detail-grid">
          <article>
            <strong><Gauge :size="14" /> CPU 限制</strong>
            <label>
              <span>{{ selectedContainer.limitCpu }} 核</span>
              <input v-model.number="selectedContainer.limitCpu" type="range" min="1" max="16" @change="updateLimit('cpu')" />
            </label>
          </article>
          <article>
            <strong><Activity :size="14" /> 内存限制</strong>
            <label>
              <span>{{ selectedContainer.limitMemory }} MB</span>
              <input v-model.number="selectedContainer.limitMemory" type="range" min="512" max="32768" step="512" @change="updateLimit('memory')" />
            </label>
          </article>
          <article>
            <strong><CheckCircle2 :size="14" /> 策略</strong>
            <span>{{ selectedContainer.isolation }}</span>
          </article>
        </div>
      </section>
    </main>

    <aside class="docker-window__inspector" aria-label="容器检查器">
      <header>
        <h3><Settings2 :size="15" /> 检查器</h3>
        <span>{{ actionState }}</span>
      </header>

      <template v-if="selectedContainer">
        <nav class="docker-window__tabs">
          <button
            v-for="mode in (['概览', '日志', '端口', '挂载', '环境'] as DetailMode[])"
            :key="mode"
            type="button"
            :class="{ 'docker-window__tab--active': detailMode === mode }"
            @click="detailMode = mode"
          >
            {{ mode }}
          </button>
        </nav>

        <section class="docker-window__inspector-body">
          <div v-if="detailMode === '概览'" class="docker-window__overview">
            <strong>{{ selectedContainer.name }}</strong>
            <span>栈：{{ selectedContainer.stack }}</span>
            <span>状态：{{ selectedContainer.status }}</span>
            <span>重启：{{ selectedContainer.restarts }} 次</span>
            <span>CPU：{{ selectedContainer.cpu }}%</span>
            <span>内存：{{ selectedContainer.memoryText }}</span>
          </div>

          <div v-else-if="detailMode === '日志'" class="docker-window__log-list">
            <button type="button" @click="refreshSelectedContainerLogs()"><RefreshCw :size="13" />刷新日志</button>
            <code v-for="entry in selectedContainer.log" :key="entry">{{ entry }}</code>
            <span v-if="selectedContainer.log.length === 0">暂无日志。</span>
          </div>

          <div v-else-if="detailMode === '端口'" class="docker-window__kv-list">
            <span v-for="port in selectedContainer.ports" :key="port"><Server :size="13" />{{ port }}</span>
          </div>

          <div v-else-if="detailMode === '挂载'" class="docker-window__kv-list">
            <span v-for="mount in selectedContainer.mounts" :key="mount"><Database :size="13" />{{ mount }}</span>
          </div>

          <div v-else class="docker-window__kv-list">
            <span v-for="env in selectedContainer.env" :key="env"><TerminalSquare :size="13" />{{ env }}</span>
          </div>
        </section>
      </template>

      <div v-else class="docker-window__empty docker-window__empty--inspector">
        <ListFilter :size="22" />
        <strong>请选择容器</strong>
        <span>选择一个容器后查看日志、端口、挂载和环境变量。</span>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.docker-window {
  display: grid;
  grid-template-columns: minmax(220px, 260px) minmax(0, 1fr) minmax(280px, 330px);
  gap: 12px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.docker-window :is(section, aside, main, article, div, nav, header, button) {
  min-width: 0;
}

.docker-window__sidebar,
.docker-window__table-card,
.docker-window__container-detail,
.docker-window__inspector,
.docker-window__topline,
.docker-window__toolbar {
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.docker-window header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.docker-window h3,
.docker-window p {
  margin: 0;
}

.docker-window h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-size: 13px;
}

.docker-window p,
.docker-window small,
.docker-window header span {
  color: var(--text-muted);
  font-size: 11px;
}

.docker-window button,
.docker-window input {
  font-family: inherit;
}

.docker-window__sidebar,
.docker-window__inspector {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  overflow: hidden;
}

.docker-window__sidebar header button,
.docker-window__actions button,
.docker-window__log-list button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.docker-window__scope-list {
  display: grid;
  align-content: start;
  min-height: 0;
  overflow: auto;
}

.docker-window__scope {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-height: 68px;
  padding: 10px 12px;
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.docker-window__scope--active {
  background: rgba(19, 136, 255, 0.09);
  box-shadow: inset 3px 0 0 var(--accent);
}

.docker-window__scope strong,
.docker-window__row strong {
  display: block;
  overflow: hidden;
  color: var(--text-strong);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-window__scope small {
  display: block;
  margin-top: 4px;
}

.docker-window__scope b {
  color: var(--text-strong);
  font-size: 12px;
}

.docker-window__quick-stats {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 10px;
  border-top: 1px solid rgba(100, 136, 166, 0.14);
}

.docker-window__quick-stats div,
.docker-window__topline article,
.docker-window__detail-grid article {
  background: rgba(255, 255, 255, 0.56);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.docker-window__quick-stats div {
  display: grid;
  gap: 3px;
  padding: 9px;
}

.docker-window__quick-stats strong,
.docker-window__topline strong {
  color: var(--text-strong);
  font-size: 16px;
}

.docker-window__quick-stats span,
.docker-window__topline span {
  color: var(--text-muted);
  font-size: 10px;
}

.docker-window__workspace {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  gap: 12px;
  min-height: 0;
  overflow: hidden;
}

.docker-window__topline {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 8px;
  padding: 10px;
}

.docker-window__topline article {
  display: grid;
  gap: 4px;
  justify-items: center;
  padding: 10px 8px;
  color: var(--accent);
}

.docker-window__toolbar {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  padding: 10px;
}

.docker-window__search {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 34px;
  padding: 0 11px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.66);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: 999px;
}

.docker-window__search input {
  width: 100%;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
  font-size: 12px;
}

.docker-window__filters {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  justify-content: flex-end;
}

.docker-window__filters button,
.docker-window__tabs button {
  height: 32px;
  padding: 0 10px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.docker-window__filter--active,
.docker-window__tab--active {
  color: var(--accent) !important;
  background: rgba(19, 136, 255, 0.12) !important;
  border-color: rgba(19, 136, 255, 0.24) !important;
}

.docker-window__table-card {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  overflow: hidden;
}

.docker-window__table-head,
.docker-window__row {
  display: grid;
  grid-template-columns: minmax(150px, 1.2fr) 86px minmax(150px, 1.2fr) 58px minmax(100px, 0.85fr) minmax(120px, 1fr);
  gap: 10px;
  align-items: center;
}

.docker-window__table-head {
  padding: 9px 12px;
  color: var(--text-muted);
  background: rgba(231, 247, 255, 0.45);
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
  font-size: 10px;
  font-weight: 760;
}

.docker-window__table {
  display: grid;
  align-content: start;
  gap: 8px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.docker-window__row {
  min-height: 58px;
  padding: 9px 10px;
  color: var(--text-muted);
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.docker-window__row--active {
  border-color: rgba(19, 136, 255, 0.28);
  box-shadow: inset 3px 0 0 var(--accent);
}

.docker-window__row span,
.docker-window__row small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-window__status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  justify-self: start;
  padding: 4px 7px;
  font-size: 10px;
  border-radius: 999px;
}

.docker-window__status--running {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.12);
}

.docker-window__status--stopped {
  color: var(--text-muted);
  background: rgba(148, 163, 184, 0.16);
}

.docker-window__status--restarting {
  color: #b36a00;
  background: rgba(245, 158, 11, 0.14);
}

.docker-window__container-detail {
  display: grid;
  grid-template-rows: auto auto;
  overflow: hidden;
}

.docker-window__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.docker-window__detail-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 10px;
}

.docker-window__detail-grid article {
  display: grid;
  gap: 8px;
  padding: 10px;
}

.docker-window__detail-grid strong,
.docker-window__overview strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-size: 12px;
}

.docker-window__detail-grid span,
.docker-window__overview span {
  color: var(--text-muted);
  font-size: 11px;
  overflow-wrap: anywhere;
}

.docker-window__detail-grid label {
  display: grid;
  gap: 6px;
}

.docker-window__detail-grid input {
  width: 100%;
  accent-color: var(--accent);
}

.docker-window__inspector header {
  align-items: start;
}

.docker-window__inspector header span {
  max-width: 180px;
  line-height: 1.35;
  text-align: right;
}

.docker-window__tabs {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
  padding: 10px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.docker-window__inspector-body {
  min-height: 0;
  padding: 12px;
  overflow: auto;
}

.docker-window__overview,
.docker-window__kv-list,
.docker-window__log-list {
  display: grid;
  align-content: start;
  gap: 8px;
}

.docker-window__overview span,
.docker-window__kv-list span,
.docker-window__log-list code,
.docker-window__empty {
  padding: 9px 10px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.56);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
  font-size: 11px;
  line-height: 1.4;
  overflow-wrap: anywhere;
}

.docker-window__kv-list span {
  display: inline-flex;
  gap: 7px;
  align-items: flex-start;
}

.docker-window__log-list code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace;
}

.docker-window__empty {
  display: grid;
  place-items: center;
  gap: 6px;
  min-height: 160px;
  text-align: center;
}

.docker-window__empty--inspector {
  margin: 12px;
}

.docker-window__empty strong {
  color: var(--text-strong);
  font-size: 12px;
}

@media (max-width: 980px) {
  .docker-window {
    grid-template-columns: minmax(0, 1fr);
    height: auto;
    min-height: 100%;
    overflow: auto;
  }

  .docker-window__sidebar,
  .docker-window__workspace,
  .docker-window__inspector {
    overflow: visible;
  }

  .docker-window__scope-list {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    overflow: visible;
  }

  .docker-window__topline,
  .docker-window__detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .docker-window__toolbar {
    grid-template-columns: minmax(0, 1fr);
  }

  .docker-window__filters {
    justify-content: flex-start;
  }
}

@media (max-width: 640px) {
  .docker-window__scope-list,
  .docker-window__topline,
  .docker-window__detail-grid,
  .docker-window__quick-stats {
    grid-template-columns: minmax(0, 1fr);
  }

  .docker-window__table-head {
    display: none;
  }

  .docker-window__row {
    grid-template-columns: minmax(0, 1fr) auto;
  }

  .docker-window__row > span:nth-of-type(n + 2) {
    grid-column: 1 / -1;
  }

  .docker-window__inspector header {
    display: grid;
  }

  .docker-window__inspector header span {
    max-width: none;
    text-align: left;
  }
}
</style>
