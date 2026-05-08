<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Activity,
  Boxes,
  Container,
  Database,
  FileText,
  Gauge,
  Play,
  RotateCw,
  Server,
  SlidersHorizontal,
  Square,
  TerminalSquare,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { ComposeStack, DockerContainer } from '../../api/types';

type ContainerStatus = '运行中' | '已停止' | '重启中' | string;
type StartStopStatus = '运行中' | '已停止';
type DetailMode = '日志' | '端口';

const composeStacks = ref<ComposeStack[]>([
  { name: 'media-stack', status: '健康', services: 4, ports: '8096 / 51413', volume: '/volume1/docker/media', network: 'isolated-media' },
  { name: 'home-ai', status: '需更新', services: 3, ports: '3000 / 11434', volume: '/volume1/docker/ai', network: 'ai-sandbox' },
  { name: 'edge-gateway', status: '健康', services: 2, ports: '80 / 443', volume: '/volume1/docker/gateway', network: 'dmz-proxy' },
]);

const containers = ref<DockerContainer[]>([
  {
    id: 'jellyfin',
    name: 'jellyfin-media',
    image: 'jellyfin/jellyfin:10.9',
    stack: 'media-stack',
    status: '运行中',
    cpu: 18,
    memory: 42,
    memoryText: '1.7 GB / 4 GB',
    ports: ['8096:8096/tcp', '8920:8920/tcp'],
    mounts: ['/volume1/media:/media:ro', '/volume1/docker/media/jellyfin:/config'],
    env: ['TZ=Asia/Shanghai', 'PUID=1000', 'PGID=1000'],
    limitCpu: 4,
    limitMemory: 4096,
    restarts: 1,
    isolation: '只读媒体库 · 无系统目录',
    log: ['媒体库扫描完成', '硬件转码队列 2 个任务', '端口 8096 已绑定到局域网'],
  },
  {
    id: 'transmission',
    name: 'transmission',
    image: 'linuxserver/transmission:latest',
    stack: 'media-stack',
    status: '运行中',
    cpu: 9,
    memory: 21,
    memoryText: '620 MB / 3 GB',
    ports: ['9091:9091/tcp', '51413:51413/tcp', '51413:51413/udp'],
    mounts: ['/volume1/downloads:/downloads', '/volume1/docker/media/transmission:/config'],
    env: ['USER=family', 'PEERPORT=51413'],
    limitCpu: 3,
    limitMemory: 3072,
    restarts: 0,
    isolation: '下载目录写入 · 应用配置隔离',
    log: ['订阅下载队列同步完成', '上传限速 4 MB/s', 'DHT 节点已连接'],
  },
  {
    id: 'ollama',
    name: 'ollama-local',
    image: 'ollama/ollama:0.5',
    stack: 'home-ai',
    status: '运行中',
    cpu: 36,
    memory: 68,
    memoryText: '5.4 GB / 8 GB',
    ports: ['11434:11434/tcp'],
    mounts: ['/volume1/ai/models:/root/.ollama', '/volume1/docker/ai/ollama:/cache'],
    env: ['OLLAMA_KEEP_ALIVE=30m', 'MODEL_POLICY=local-first'],
    limitCpu: 6,
    limitMemory: 8192,
    restarts: 2,
    isolation: '模型目录白名单 · 禁止外发敏感数据',
    log: ['qwen2.5:7b 已加载', '向量任务等待 GPU 调度', '本地推理端口 11434 正常'],
  },
  {
    id: 'gateway',
    name: 'caddy-gateway',
    image: 'caddy:2.8',
    stack: 'edge-gateway',
    status: '已停止',
    cpu: 0,
    memory: 0,
    memoryText: '0 MB / 512 MB',
    ports: ['80:80/tcp', '443:443/tcp'],
    mounts: ['/volume1/docker/gateway/Caddyfile:/etc/caddy/Caddyfile:ro'],
    env: ['ACME_AGREE=true', 'TRUSTED_PROXIES=lan'],
    limitCpu: 1,
    limitMemory: 512,
    restarts: 4,
    isolation: 'DMZ 网络 · 只读反代配置',
    log: ['用户手动停止服务', '证书续期任务暂停', '端口 443 已释放'],
  },
]);

const selectedStackName = ref(composeStacks.value[0].name);
const selectedContainerId = ref(containers.value[0].id);
const detailMode = ref<DetailMode>('日志');
const actionState = ref('容器运行时已连接，资源统计每 30 秒刷新。');

const stackContainers = computed(() => containers.value.filter((item) => item.stack === selectedStackName.value));
const selectedContainer = computed(() => containers.value.find((item) => item.id === selectedContainerId.value) ?? containers.value[0]);
const selectedStack = computed(() => composeStacks.value.find((item) => item.name === selectedStackName.value) ?? composeStacks.value[0]);
const resourceLimit = computed(() => `${selectedContainer.value.limitCpu} CPU / ${selectedContainer.value.limitMemory} MB`);
const runningCount = computed(() => containers.value.filter((item) => item.status === '运行中').length);
const totalCpu = computed(() => Math.min(100, containers.value.reduce((sum, item) => sum + item.cpu, 0)));
const totalMemory = computed(() => Math.min(100, Math.round(containers.value.reduce((sum, item) => sum + item.memory, 0) / containers.value.length)));

function selectStack(name: string) {
  selectedStackName.value = name;
  selectedContainerId.value = containers.value.find((item) => item.stack === name)?.id ?? selectedContainerId.value;
  actionState.value = `已切换到 Compose 栈 ${name}`;
  void refreshSelectedContainerLogs();
}

async function selectContainer(id: string) {
  selectedContainerId.value = id;
  actionState.value = `正在查看 ${selectedContainer.value.name} 状态`;
  await refreshSelectedContainerLogs(id);
}

async function setContainerStatus(status: StartStopStatus) {
  const current = selectedContainer.value;
  try {
    const next =
      status === '运行中'
        ? await apiClient.docker.startContainer(current.id)
        : await apiClient.docker.stopContainer(current.id);
    await replaceContainerWithLogs(next);
    actionState.value = `${next.name} 已${status === '运行中' ? '启动' : '停止'}，审计事件已写入。`;
  } catch (error) {
    mutateLocalContainerStatus(status);
    actionState.value = `${selectedContainer.value.name} 已在本地切换状态，后端同步失败：${errorMessage(error)}`;
  }
}

async function restartContainer() {
  try {
    const next = await apiClient.docker.restartContainer(selectedContainer.value.id);
    await replaceContainerWithLogs(next);
    actionState.value = `${next.name} 正在重启，Compose 栈保持 ${selectedStack.value.status}。`;
  } catch (error) {
    mutateLocalRestart();
    actionState.value = `${selectedContainer.value.name} 已进入本地重启状态，后端同步失败：${errorMessage(error)}`;
  }
}

async function completeRestart() {
  try {
    const next = await apiClient.docker.completeRestart(selectedContainer.value.id);
    await replaceContainerWithLogs(next);
    actionState.value = `${next.name} 重启完成，端口 ${next.ports[0]} 可访问。`;
  } catch (error) {
    mutateLocalCompleteRestart();
    actionState.value = `${selectedContainer.value.name} 已在本地完成检查，后端同步失败：${errorMessage(error)}`;
  }
}

async function updateLimit(kind: 'cpu' | 'memory') {
  const current = selectedContainer.value;
  try {
    const next = await apiClient.docker.updateContainerLimits(current.id, {
      limitCpu: current.limitCpu,
      limitMemory: current.limitMemory,
    });
    await replaceContainerWithLogs(next);
    actionState.value =
      kind === 'cpu'
        ? `${next.name} CPU 限制调整为 ${next.limitCpu} 核`
        : `${next.name} 内存限制调整为 ${next.limitMemory} MB`;
  } catch (error) {
    actionState.value =
      kind === 'cpu'
        ? `${current.name} CPU 限制已本地调整为 ${current.limitCpu} 核，后端同步失败：${errorMessage(error)}`
        : `${current.name} 内存限制已本地调整为 ${current.limitMemory} MB，后端同步失败：${errorMessage(error)}`;
  }
}

async function loadDockerRuntime() {
  try {
    const [nextStacks, nextContainers] = await Promise.all([
      apiClient.docker.getStacks(),
      apiClient.docker.getContainers(),
    ]);
    if (nextStacks.length) composeStacks.value = nextStacks;
    if (nextContainers.length) containers.value = nextContainers.map((container) => withContainerLog(container));
    if (!composeStacks.value.some((stack) => stack.name === selectedStackName.value)) {
      selectedStackName.value = composeStacks.value[0].name;
    }
    if (!containers.value.some((container) => container.id === selectedContainerId.value)) {
      selectedContainerId.value = containers.value.find((container) => container.stack === selectedStackName.value)?.id ?? containers.value[0].id;
    }
    await refreshSelectedContainerLogs();
    actionState.value = 'Docker 后端已连接，Compose 栈、容器状态和日志已同步。';
  } catch (error) {
    actionState.value = `后端暂不可用，继续使用本地 Docker 缓存：${errorMessage(error)}`;
  }
}

async function refreshSelectedContainerLogs(containerId = selectedContainerId.value) {
  try {
    const logs = await apiClient.docker.getContainerLogs(containerId, 20);
    const current = containers.value.find((container) => container.id === containerId);
    if (current) replaceContainer(withContainerLog(current, logs));
  } catch {
    // Log refresh should not block container navigation in fallback mode.
  }
}

async function replaceContainerWithLogs(container: DockerContainer) {
  const logs = await apiClient.docker.getContainerLogs(container.id, 20);
  return replaceContainer(withContainerLog(container, logs));
}

function replaceContainer(container: DockerContainer) {
  const index = containers.value.findIndex((item) => item.id === container.id);
  if (index === -1) {
    containers.value = [...containers.value, container];
  } else {
    containers.value = containers.value.map((item) => (item.id === container.id ? container : item));
  }
  selectedContainerId.value = container.id;
  selectedStackName.value = container.stack;
  return container;
}

function withContainerLog(container: DockerContainer, logs?: string[]): DockerContainer {
  const previous = containers.value.find((item) => item.id === container.id);
  return {
    ...container,
    log: logs ?? container.log ?? previous?.log ?? [],
  };
}

function mutateLocalContainerStatus(status: StartStopStatus) {
  selectedContainer.value.status = status;
  selectedContainer.value.cpu = status === '运行中' ? Math.max(6, selectedContainer.value.cpu || 12) : 0;
  selectedContainer.value.memory = status === '运行中' ? Math.max(12, selectedContainer.value.memory || 18) : 0;
  selectedContainer.value.log.unshift(`${status === '运行中' ? '启动' : '停止'}操作已由 Web 桌面触发`);
}

function mutateLocalRestart() {
  selectedContainer.value.status = '重启中';
  selectedContainer.value.restarts += 1;
  selectedContainer.value.cpu = 4;
  selectedContainer.value.log.unshift(`第 ${selectedContainer.value.restarts} 次重启：正在重新创建容器`);
}

function mutateLocalCompleteRestart() {
  selectedContainer.value.status = '运行中';
  selectedContainer.value.cpu = Math.max(10, selectedContainer.value.cpu + 8);
  selectedContainer.value.memory = Math.max(18, selectedContainer.value.memory + 6);
  selectedContainer.value.log.unshift('健康检查通过，容器已恢复服务');
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

onMounted(() => {
  void loadDockerRuntime();
});
</script>

<template>
  <div class="docker-window">
    <aside class="docker-window__stacks" aria-label="Compose 栈">
      <header>
        <h3><Boxes :size="15" /> Compose 栈</h3>
        <span>{{ composeStacks.length }} 组</span>
      </header>
      <button
        v-for="stack in composeStacks"
        :key="stack.name"
        class="docker-window__stack"
        :class="{ 'docker-window__stack--active': stack.name === selectedStackName }"
        type="button"
        @click="selectStack(stack.name)"
      >
        <strong>{{ stack.name }}</strong>
        <span>{{ stack.services }} services · {{ stack.network }}</span>
        <small>{{ stack.status }} · {{ stack.volume }}</small>
      </button>
    </aside>

    <main class="docker-window__main">
      <section class="docker-window__summary" aria-label="Docker 资源统计">
        <div>
          <Container :size="18" />
          <span>容器</span>
          <strong>{{ runningCount }}/{{ containers.length }} 运行</strong>
        </div>
        <div>
          <Gauge :size="18" />
          <span>CPU</span>
          <strong>{{ totalCpu }}%</strong>
        </div>
        <div>
          <Activity :size="18" />
          <span>内存</span>
          <strong>{{ totalMemory }}%</strong>
        </div>
        <div>
          <Server :size="18" />
          <span>端口</span>
          <strong>{{ selectedStack.ports }}</strong>
        </div>
      </section>

      <section class="docker-window__containers" aria-label="容器列表">
        <header>
          <h3>{{ selectedStackName }}</h3>
          <span>{{ selectedStack.volume }}</span>
        </header>
        <div class="docker-window__table">
          <button
            v-for="container in stackContainers"
            :key="container.id"
            class="docker-window__row"
            :class="{ 'docker-window__row--active': container.id === selectedContainerId }"
            type="button"
            @click="selectContainer(container.id)"
          >
            <span>
              <strong>{{ container.name }}</strong>
              <small>{{ container.image }}</small>
            </span>
            <b :class="`docker-window__status docker-window__status--${container.status}`">{{ container.status }}</b>
            <span>{{ container.cpu }}% CPU</span>
            <span>{{ container.memoryText }}</span>
          </button>
        </div>
      </section>

      <section class="docker-window__details" aria-label="容器端口、存储、环境变量和资源限制">
        <header>
          <h3>{{ selectedContainer.name }}</h3>
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
            <button v-if="selectedContainer.status === '重启中'" type="button" @click="completeRestart">完成检查</button>
          </div>
        </header>

        <div class="docker-window__detail-grid">
          <div class="docker-window__panel">
            <strong><SlidersHorizontal :size="14" /> 资源限制</strong>
            <label>
              <span>CPU 核数 {{ selectedContainer.limitCpu }}</span>
              <input v-model.number="selectedContainer.limitCpu" type="range" min="1" max="8" @change="updateLimit('cpu')" />
            </label>
            <label>
              <span>内存 {{ selectedContainer.limitMemory }} MB</span>
              <input
                v-model.number="selectedContainer.limitMemory"
                type="range"
                min="512"
                max="12288"
                step="512"
                @change="updateLimit('memory')"
              />
            </label>
          </div>

          <div class="docker-window__panel">
            <strong><Database :size="14" /> 存储路径</strong>
            <span v-for="mount in selectedContainer.mounts" :key="mount">{{ mount }}</span>
          </div>

          <div class="docker-window__panel">
            <strong><TerminalSquare :size="14" /> 环境变量</strong>
            <span v-for="env in selectedContainer.env" :key="env">{{ env }}</span>
          </div>
        </div>

        <p class="docker-window__isolation">{{ selectedContainer.isolation }}</p>
      </section>
    </main>

    <aside class="docker-window__inspector" aria-label="日志和端口详情">
      <div class="docker-window__switch">
        <button type="button" :class="{ 'docker-window__switch--active': detailMode === '日志' }" @click="detailMode = '日志'">
          <FileText :size="13" /> 日志
        </button>
        <button type="button" :class="{ 'docker-window__switch--active': detailMode === '端口' }" @click="detailMode = '端口'">
          <Server :size="13" /> 端口
        </button>
      </div>

      <div class="docker-window__inspector-body">
        <template v-if="detailMode === '日志'">
          <strong>状态查看</strong>
          <p>{{ actionState }}</p>
          <ul>
            <li v-for="entry in selectedContainer.log" :key="entry">{{ entry }}</li>
          </ul>
        </template>
        <template v-else>
          <strong>端口详情</strong>
          <p>{{ selectedContainer.name }} 绑定 {{ selectedContainer.ports.length }} 个端口，网络 {{ selectedStack.network }}。</p>
          <span v-for="port in selectedContainer.ports" :key="port">{{ port }}</span>
        </template>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.docker-window {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr) 210px;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.docker-window__stacks,
.docker-window__main,
.docker-window__containers,
.docker-window__details,
.docker-window__inspector {
  min-height: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.docker-window__stacks,
.docker-window__inspector {
  overflow: hidden;
}

.docker-window__stacks {
  display: grid;
  grid-template-rows: auto repeat(3, minmax(0, 1fr));
}

.docker-window header,
.docker-window__stacks header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.docker-window h3,
.docker-window__stacks h3 {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.docker-window header span,
.docker-window__stacks header span {
  color: var(--text-soft);
  font-size: 11px;
}

.docker-window__stack {
  display: grid;
  gap: 4px;
  align-content: center;
  min-width: 0;
  padding: 10px 12px;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.docker-window__stack--active {
  background: rgba(19, 136, 255, 0.08);
  box-shadow: inset 3px 0 0 var(--accent);
}

.docker-window__stack strong,
.docker-window__row strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-window__stack span,
.docker-window__stack small,
.docker-window__row small {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-window__main {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 10px;
  overflow: hidden;
  background: transparent;
  border: 0;
}

.docker-window__summary {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.docker-window__summary div {
  display: grid;
  gap: 3px;
  justify-items: center;
  padding: 10px 6px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.56);
}

.docker-window__summary span {
  color: var(--text-soft);
  font-size: 10px;
}

.docker-window__summary strong {
  overflow: hidden;
  max-width: 100%;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-window__containers {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.docker-window__table {
  display: grid;
  gap: 7px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.docker-window__row {
  display: grid;
  grid-template-columns: minmax(120px, 1.2fr) 78px 70px minmax(92px, 1fr);
  align-items: center;
  gap: 9px;
  min-height: 42px;
  padding: 8px 9px;
  color: var(--text-muted);
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.docker-window__row--active {
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.docker-window__row > span {
  min-width: 0;
  font-size: 11px;
}

.docker-window__row strong,
.docker-window__row small {
  display: block;
}

.docker-window__status {
  justify-self: start;
  padding: 4px 7px;
  font-size: 10px;
  border-radius: 999px;
}

.docker-window__status--运行中 {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.12);
}

.docker-window__status--已停止 {
  color: var(--text-muted);
  background: rgba(148, 163, 184, 0.15);
}

.docker-window__status--重启中 {
  color: #b36a00;
  background: rgba(245, 158, 11, 0.14);
}

.docker-window__details {
  overflow: hidden;
}

.docker-window__actions {
  display: flex;
  flex-wrap: wrap;
  justify-content: flex-end;
  gap: 6px;
}

.docker-window__actions button,
.docker-window__switch button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-width: 0;
  height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.docker-window__detail-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
  padding: 10px;
}

.docker-window__panel {
  display: grid;
  align-content: start;
  gap: 7px;
  min-width: 0;
  min-height: 116px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.docker-window__panel strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-size: 12px;
}

.docker-window__panel span,
.docker-window__panel label {
  min-width: 0;
  overflow-wrap: anywhere;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.docker-window__panel label {
  display: grid;
  gap: 5px;
}

.docker-window__panel input {
  width: 100%;
  accent-color: var(--accent);
}

.docker-window__isolation {
  margin: 0 10px 10px;
  padding: 8px 10px;
  color: var(--text-muted);
  background: rgba(231, 247, 255, 0.62);
  border: 1px solid rgba(19, 136, 255, 0.12);
  border-radius: var(--radius-sm);
  font-size: 11px;
  line-height: 1.35;
}

.docker-window__inspector {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
}

.docker-window__switch {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 6px;
  padding: 10px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.docker-window__switch button {
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border-color: var(--border);
}

.docker-window__switch--active {
  color: var(--accent) !important;
  background: rgba(19, 136, 255, 0.1) !important;
}

.docker-window__inspector-body {
  min-height: 0;
  padding: 12px;
  overflow: auto;
}

.docker-window__inspector-body strong {
  color: var(--text-strong);
  font-size: 13px;
}

.docker-window__inspector-body p,
.docker-window__inspector-body li,
.docker-window__inspector-body span {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.42;
}

.docker-window__inspector-body ul {
  display: grid;
  gap: 7px;
  padding: 0;
  margin: 10px 0 0;
  list-style: none;
}

.docker-window__inspector-body li,
.docker-window__inspector-body span {
  display: block;
  padding: 8px;
  overflow-wrap: anywhere;
  background: rgba(255, 255, 255, 0.56);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

@media (max-width: 860px) {
  .docker-window {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr) auto;
    overflow: auto;
  }

  .docker-window__stacks {
    grid-template-rows: auto;
  }

  .docker-window__summary,
  .docker-window__detail-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .docker-window__row {
    grid-template-columns: minmax(120px, 1fr) 78px;
  }
}
</style>
