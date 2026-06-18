<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { Terminal as XTerm } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import {
  Activity,
  Boxes,
  ChevronLeft,
  ChevronRight,
  CheckCircle2,
  CircleDot,
  Container,
  Database,
  Gauge,
  HardDrive,
  ImageDown,
  ListFilter,
  Network,
  Play,
  Plug,
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  Server,
  ShieldCheck,
  Square,
  TerminalSquare,
  Trash2,
  X,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type {
  ComposeStack,
  DockerContainer,
  DockerImage,
  DockerImagePullStatus,
  DockerImageSearchResult,
  DockerNetwork,
  DockerVolume,
  Metric,
  TrendPoint,
} from '../../api/types';

const props = defineProps<{
  openToken?: number;
}>();

type ResourceTab = '概览' | '容器' | 'Compose' | '本地镜像' | '镜像仓库' | '网络' | '存储卷';
type ContainerStatus = '全部' | '运行中' | '已停止' | '重启中';
type ContainerDetailTab = '概览' | '日志' | '终端' | '端口' | '挂载' | '环境';
type StartStopStatus = '运行中' | '已停止';
type MonitorKey = 'cpu' | 'memory' | 'network';
type CreateStep = 0 | 1 | 2;
type PortProtocol = 'tcp' | 'udp';
type TerminalStatus = 'idle' | 'connecting' | 'ready' | 'closed' | 'unsupported';

type PortRow = {
  host: string;
  container: string;
  protocol: PortProtocol;
};

type MountRow = {
  host: string;
  container: string;
  mode: 'rw' | 'ro';
};

type KeyValueRow = {
  key: string;
  value: string;
};

type CreateContainerForm = {
  image: string;
  name: string;
  restartPolicy: string;
  privileged: boolean;
  autoRemove: boolean;
  limitCpu: number;
  limitMemory: number;
  network: string;
  hostname: string;
  user: string;
  workingDir: string;
  entrypoint: string;
  command: string;
  ports: PortRow[];
  mounts: MountRow[];
  env: KeyValueRow[];
  labels: KeyValueRow[];
  extraHosts: KeyValueRow[];
  dns: string;
};

const resourceTabs: ResourceTab[] = ['概览', '容器', 'Compose', '本地镜像', '镜像仓库', '网络', '存储卷'];
const statusTabs: ContainerStatus[] = ['全部', '运行中', '已停止', '重启中'];
const detailTabs: ContainerDetailTab[] = ['概览', '日志', '终端', '端口', '挂载', '环境'];
const monitorKeys: MonitorKey[] = ['cpu', 'memory', 'network'];
const createSteps = ['基础', '配置', '确认'] as const;
const dockerIconPath =
  'M205.653333 737.066667c-29.184 0-55.637333-23.893333-55.637333-52.906667s23.893333-53.034667 55.68-53.034667c31.914667 0 55.893333 23.893333 55.893333 52.992s-26.538667 52.906667-55.68 52.906667z m683.178667-288.554667c-5.76-42.325333-32-76.8-66.56-103.253333l-13.44-10.666667-10.837333 13.226667c-21.077333 23.893333-29.44 66.261333-26.88 97.92 2.56 23.978667 10.24 47.786667 23.637333 66.304-10.837333 5.546667-24.234667 10.666667-34.56 16.085333a225.706667 225.706667 0 0 1-71.68 10.666667H4.138667l-2.56 15.786666a297.813333 297.813333 0 0 0 23.978666 151.04l10.410667 18.56v2.56c64 105.941333 177.92 153.6 301.994667 153.6 238.677333 0 434.432-103.253333 527.232-325.674666 60.8 2.645333 122.197333-13.226667 151.04-71.509334l7.68-13.226666-12.8-7.978667c-34.56-21.077333-81.92-23.893333-121.6-13.226667l-0.768 0.085334z m-341.674667-42.325333h-103.594666v103.253333h103.68V406.101333l-0.085334 0.128z m0-129.834667h-103.594666v103.253333h103.68V276.48l-0.085334-0.128z m0-132.437333h-103.594666v103.253333h103.68v-103.253333h-0.085334z m126.72 262.272H570.88v103.253333h103.253333V406.101333l-0.298666 0.128z m-383.914666 0H187.008v103.253333h103.338667V406.101333l-0.426667 0.128z m129.28 0h-102.4v103.253333H419.84V406.101333l-0.64 0.128z m-257.28 0H59.733333v103.253333h103.594667V406.101333l-1.28 0.128z m257.28-129.834667h-102.4v103.253333H419.84V276.48l-0.64-0.128z m-129.92 0H187.178667v103.253333H290.133333V276.48l-0.682666-0.128z';
const restartPolicies = [
  { value: 'unless-stopped', label: '除非手动停止', desc: '重启系统或 Docker 后自动恢复。' },
  { value: 'no', label: '不自动重启', desc: '适合一次性任务或调试容器。' },
  { value: 'always', label: '始终重启', desc: '容器退出后尽量自动拉起。' },
  { value: 'on-failure', label: '失败时重启', desc: '仅异常退出后重启。' },
];
const protocolOptions: PortProtocol[] = ['tcp', 'udp'];

function emptyCreateForm(image = ''): CreateContainerForm {
  return {
    image,
    name: '',
    restartPolicy: 'unless-stopped',
    privileged: false,
    autoRemove: false,
    limitCpu: 0,
    limitMemory: 0,
    network: 'bridge',
    hostname: '',
    user: '',
    workingDir: '',
    entrypoint: '',
    command: '',
    ports: [{ host: '', container: '', protocol: 'tcp' }],
    mounts: [{ host: '', container: '', mode: 'rw' }],
    env: [{ key: '', value: '' }],
    labels: [{ key: '', value: '' }],
    extraHosts: [{ key: '', value: '' }],
    dns: '',
  };
}

const composeStacks = ref<ComposeStack[]>([]);
const containers = ref<DockerContainer[]>([]);
const images = ref<DockerImage[]>([]);
const imagePulls = ref<DockerImagePullStatus[]>([]);
const volumes = ref<DockerVolume[]>([]);
const networks = ref<DockerNetwork[]>([]);
const imageSearchResults = ref<DockerImageSearchResult[]>([]);
const imageIconFailures = ref<Set<string>>(new Set());
const systemMetrics = ref<Metric[]>([]);
const metricHistory = ref<Record<MonitorKey, number[]>>({ cpu: [], memory: [], network: [] });
const selectedStackName = ref('all');
const selectedContainerId = ref('');
const selectedNetworkName = ref('');
const selectedVolumeName = ref('');
const resourceTab = ref<ResourceTab>('概览');
const statusFilter = ref<ContainerStatus>('全部');
const detailTab = ref<ContainerDetailTab>('概览');
const query = ref('');
const imageQuery = ref('');
const loading = ref(false);
const creating = ref(false);
const runningAction = ref('');
const actionState = ref('Docker 后端连接中。');
const terminalStatus = ref<TerminalStatus>('idle');
const terminalOutput = ref('');
const terminalUnsupportedMessage = ref('');
const terminalShell = ref('/bin/sh');
const terminalScreenRef = ref<HTMLElement | null>(null);
const createDialogOpen = ref(false);
const createStep = ref<CreateStep>(0);
const deleteContainerDialogOpen = ref(false);
const deleteContainerRemoveVolumes = ref(false);
const createForm = ref<CreateContainerForm>(emptyCreateForm());
const volumeForm = ref({ name: '', driver: 'local' });
const networkForm = ref({ name: '', driver: 'bridge', subnet: '', gateway: '', attachable: false, internal: false });
const networkAttachForm = ref({ container: '', alias: '' });
const isDarkTheme = ref(false);
let refreshTimer: number | undefined;
let themeObserver: MutationObserver | undefined;
let themeMediaQuery: MediaQueryList | undefined;
let terminalSocket: WebSocket | undefined;
let terminalInstance: XTerm | undefined;
let terminalDataDisposable: { dispose: () => void } | undefined;

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
const selectedLogText = computed(() => selectedContainer.value?.log?.join('\n') || '');
const selectedNetwork = computed(() => networks.value.find((item) => item.name === selectedNetworkName.value));
const selectedVolume = computed(() => volumes.value.find((item) => item.name === selectedVolumeName.value));
const runningCount = computed(() => containers.value.filter((item) => item.status === '运行中').length);
const stoppedCount = computed(() => containers.value.filter((item) => item.status === '已停止').length);
const restartingCount = computed(() => containers.value.filter((item) => item.status === '重启中').length);
const totalCpu = computed(() => Math.min(100, containers.value.reduce((sum, item) => sum + Number(item.cpu || 0), 0)));
const totalMemory = computed(() => Math.min(100, Math.round(containers.value.reduce((sum, item) => sum + Number(item.memory || 0), 0))));
const allExposedPorts = computed(() => containers.value.flatMap((item) => item.ports).filter((port) => port && port !== '未暴露端口'));
const resourceCounts = computed<Record<ResourceTab, number>>(() => ({
  概览: containers.value.length + images.value.length + networks.value.length + volumes.value.length,
  容器: containers.value.length,
  Compose: composeStacks.value.length,
  本地镜像: images.value.length + visibleImagePulls.value.length,
  镜像仓库: imageSearchResults.value.length,
  网络: networks.value.length,
  存储卷: volumes.value.length,
}));
const statusCounts = computed<Record<ContainerStatus, number>>(() => ({
  全部: scopedContainers.value.length,
  运行中: scopedContainers.value.filter((item) => item.status === '运行中').length,
  已停止: scopedContainers.value.filter((item) => item.status === '已停止').length,
  重启中: scopedContainers.value.filter((item) => item.status === '重启中').length,
}));

const localImages = computed(() =>
  images.value.map((image) => ({
    ...image,
    ref: imageRef(image),
  })),
);
const localImageOptions = computed(() =>
  localImages.value.filter((image) => image.ref && image.ref !== '<none>' && !image.ref.startsWith('<none>:')),
);
const downloadedImageRefs = computed(() => new Set(localImageOptions.value.flatMap((image) => imageAliases(image.ref))));
const visibleImagePulls = computed(() =>
  imagePulls.value.filter((task) => {
    if (!task.image) return false;
    if (task.status === 'completed') return !downloadedImageRefs.value.has(normalizeImageRef(task.image));
    return task.status === 'running' || task.status === 'queued' || task.status === 'failed';
  }),
);
const activePullRefs = computed(() =>
  new Set(
    imagePulls.value
      .filter((task) => task.status === 'running' || task.status === 'queued')
      .flatMap((task) => imageAliases(task.image)),
  ),
);

const overviewRows = computed(() => [
  { label: '项目', value: composeStacks.value.length ? `${composeStacks.value.length} 个 Compose 栈` : '暂无 Compose 栈' },
  { label: '镜像', value: images.value.length ? `${images.value.length} 个本地镜像` : '暂无镜像' },
  { label: '容器', value: containers.value.length ? `${runningCount.value}/${containers.value.length} 运行` : '暂无容器' },
]);

const metricMap = computed(() => {
  const next: Partial<Record<MonitorKey, Metric>> = {};
  systemMetrics.value.forEach((metric) => {
    if (metric.key === 'cpu' || metric.key === 'memory' || metric.key === 'network') next[metric.key] = metric;
  });
  return next;
});
const dockerCpu = computed(() => metricNumber('cpu'));
const dockerMemory = computed(() => metricNumber('memory'));
const networkThroughput = computed(() => metricNumber('network'));
const cpuPath = computed(() => smoothTrendPath(chartSeries('cpu', dockerCpu.value), 300, 80, 100));
const memoryPath = computed(() => smoothTrendPath(chartSeries('memory', dockerMemory.value), 300, 80, 100));
const networkPath = computed(() => smoothTrendPath(chartSeries('network', networkThroughput.value), 640, 90));
const cpuAreaPath = computed(() => smoothTrendAreaPath(chartSeries('cpu', dockerCpu.value), 300, 80, 100));
const memoryAreaPath = computed(() => smoothTrendAreaPath(chartSeries('memory', dockerMemory.value), 300, 80, 100));
const networkAreaPath = computed(() => smoothTrendAreaPath(chartSeries('network', networkThroughput.value), 640, 90));
const networkOptions = computed(() => {
  const names = new Set(['bridge', 'host', 'none']);
  networks.value.forEach((network) => names.add(network.name));
  return Array.from(names);
});
const createPorts = computed(() =>
  createForm.value.ports
    .map((port) => {
      const container = port.container.trim();
      if (!container) return '';
      const suffix = port.protocol === 'udp' ? '/udp' : '';
      const host = port.host.trim();
      return host ? `${host}:${container}${suffix}` : `${container}${suffix}`;
    })
    .filter(Boolean),
);
const createMounts = computed(() =>
  createForm.value.mounts
    .map((mount) => {
      const host = mount.host.trim();
      const container = mount.container.trim();
      if (!host || !container) return '';
      return `${host}:${container}${mount.mode === 'ro' ? ':ro' : ''}`;
    })
    .filter(Boolean),
);
const createEnv = computed(() => formatKeyValueRows(createForm.value.env));
const createLabels = computed(() => formatKeyValueRows(createForm.value.labels));
const createExtraHosts = computed(() =>
  createForm.value.extraHosts
    .map((row) => {
      const host = row.key.trim();
      const ip = row.value.trim();
      return host && ip ? `${host}:${ip}` : '';
    })
    .filter(Boolean),
);
const createDns = computed(() => splitInput(createForm.value.dns));
const canAdvanceCreate = computed(() => {
  if (createStep.value === 0) return Boolean(resolveLocalImageRef(createForm.value.image));
  return true;
});
const createSummaryRows = computed(() => [
  { label: '镜像', value: createForm.value.image.trim() || '-' },
  { label: '容器名', value: createForm.value.name.trim() || '自动生成' },
  { label: '权限', value: createForm.value.privileged ? '高权限' : '普通权限' },
  { label: '重启策略', value: restartPolicies.find((item) => item.value === createForm.value.restartPolicy)?.label ?? createForm.value.restartPolicy },
  { label: '资源限制', value: `${createForm.value.limitCpu || '不限'} CPU · ${createForm.value.limitMemory || '不限'} MB` },
  { label: '网络', value: createForm.value.network || 'bridge' },
  { label: '端口', value: createPorts.value.length ? createPorts.value.join('，') : '未映射' },
  { label: '挂载', value: createMounts.value.length ? createMounts.value.join('，') : '未挂载' },
  { label: '环境变量', value: createEnv.value.length ? `${createEnv.value.length} 项` : '未设置' },
  { label: '命令', value: createForm.value.command.trim() || '使用镜像默认命令' },
]);

function selectStack(name: string) {
  selectedStackName.value = name;
  selectedContainerId.value = scopedContainers.value[0]?.id ?? '';
  if (selectedContainerId.value) void refreshSelectedContainerLogs(selectedContainerId.value);
}

async function selectContainer(id: string) {
  selectedContainerId.value = id;
  resourceTab.value = '容器';
  await refreshSelectedContainerLogs(id);
}

async function loadDockerRuntime(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [nextStacks, nextContainers, nextImages, nextPulls, nextVolumes, nextNetworks, metricSnapshot, networkTrend] = await Promise.all([
      apiClient.docker.getStacks().catch(() => []),
      apiClient.docker.getContainers(),
      apiClient.docker.getImages().catch(() => []),
      apiClient.docker.getImagePulls().catch(() => []),
      apiClient.docker.getVolumes().catch(() => []),
      apiClient.docker.getNetworks().catch(() => []),
      apiClient.monitoring.getMetricsSnapshot().catch(() => null),
      apiClient.monitoring.getMetricTrend('1H', 'network').catch(() => []),
    ]);
    composeStacks.value = nextStacks;
    containers.value = nextContainers.map((container) => withContainerLog(container));
    images.value = nextImages;
    imagePulls.value = nextPulls;
    volumes.value = nextVolumes;
    networks.value = nextNetworks;
    updateMetricHistory(metricSnapshot?.metrics ?? [], { cpu: [], memory: [], network: networkTrend });
    reconcileSelection();
    if (selectedContainerId.value) await refreshSelectedContainerLogs(selectedContainerId.value);
    actionState.value = `已同步 ${containers.value.length} 个容器、${images.value.length} 个镜像、${networks.value.length} 个网络、${volumes.value.length} 个卷。`;
  } catch (error) {
    actionState.value = `Docker 后端不可用：${errorMessage(error)}`;
    containers.value = [];
    images.value = [];
    imagePulls.value = [];
    volumes.value = [];
    networks.value = [];
  } finally {
    loading.value = false;
  }
}

function reconcileSelection() {
  if (!stackScopes.value.some((stack) => stack.name === selectedStackName.value)) selectedStackName.value = 'all';
  if (!scopedContainers.value.some((container) => container.id === selectedContainerId.value)) {
    selectedContainerId.value = scopedContainers.value[0]?.id ?? containers.value[0]?.id ?? '';
  }
  if (!networks.value.some((network) => network.name === selectedNetworkName.value)) selectedNetworkName.value = networks.value[0]?.name ?? '';
  if (!volumes.value.some((volume) => volume.name === selectedVolumeName.value)) selectedVolumeName.value = volumes.value[0]?.name ?? '';
  if (!networkAttachForm.value.container) networkAttachForm.value.container = containers.value[0]?.name ?? '';
}

async function refreshSelectedContainerLogs(containerId = selectedContainerId.value) {
  if (!containerId) return;
  try {
    const logs = await apiClient.docker.getContainerLogs(containerId, 200);
    const current = containers.value.find((container) => container.id === containerId);
    if (current) replaceContainer(withContainerLog(current, logs));
  } catch {
    // Stopped or newly created containers can legitimately have no logs.
  }
}

async function setContainerStatus(status: StartStopStatus) {
  const current = selectedContainer.value;
  if (!current) return;
  await runAction(current.id, async () => {
    const next = status === '运行中' ? await apiClient.docker.startContainer(current.id) : await apiClient.docker.stopContainer(current.id);
    replaceContainer(withContainerLog(next));
    actionState.value = `${next.name} 已${status === '运行中' ? '启动' : '停止'}。`;
  });
}

async function restartContainer() {
  const current = selectedContainer.value;
  if (!current) return;
  await runAction(current.id, async () => {
    const next = await apiClient.docker.restartContainer(current.id);
    replaceContainer(withContainerLog(next));
    actionState.value = `${next.name} 已重启。`;
  });
}

async function updateLimit(kind: 'cpu' | 'memory') {
  const current = selectedContainer.value;
  if (!current) return;
  await runAction(current.id, async () => {
    const next = await apiClient.docker.updateContainerLimits(current.id, {
      limitCpu: current.limitCpu,
      limitMemory: current.limitMemory,
    });
    replaceContainer(withContainerLog(next));
    actionState.value = kind === 'cpu' ? `${next.name} CPU 限制已更新。` : `${next.name} 内存限制已更新。`;
  });
}

async function createContainer() {
  const image = resolveLocalImageRef(createForm.value.image);
  if (!image || creating.value) return;
  creating.value = true;
  try {
    const next = await apiClient.docker.createContainer({
      image,
      name: createForm.value.name.trim(),
      ports: createPorts.value,
      mounts: createMounts.value,
      env: createEnv.value,
      labels: createLabels.value,
      extraHosts: createExtraHosts.value,
      dns: createDns.value,
      command: createForm.value.command.trim(),
      entrypoint: createForm.value.entrypoint.trim(),
      restartPolicy: createForm.value.autoRemove ? 'no' : createForm.value.restartPolicy,
      network: createForm.value.network,
      hostname: createForm.value.hostname.trim(),
      user: createForm.value.user.trim(),
      workingDir: createForm.value.workingDir.trim(),
      privileged: createForm.value.privileged,
      autoRemove: createForm.value.autoRemove,
      limitCpu: createForm.value.limitCpu,
      limitMemory: createForm.value.limitMemory,
    });
    containers.value = [withContainerLog(next), ...containers.value.filter((item) => item.id !== next.id)];
    selectedContainerId.value = next.id;
    createForm.value = emptyCreateForm();
    createStep.value = 0;
    createDialogOpen.value = false;
    actionState.value = `${next.name} 已创建。`;
    await loadDockerRuntime(true);
  } catch (error) {
    actionState.value = `创建容器失败：${errorMessage(error)}`;
  } finally {
    creating.value = false;
  }
}

function openCreateContainerDialog(image = '') {
  createForm.value = emptyCreateForm(resolveLocalImageRef(image));
  createStep.value = 0;
  createDialogOpen.value = true;
}

function closeCreateContainerDialog() {
  if (creating.value) return;
  createDialogOpen.value = false;
}

function nextCreateStep() {
  if (!canAdvanceCreate.value || createStep.value >= 2) return;
  createStep.value = (createStep.value + 1) as CreateStep;
}

function setCreateStep(index: number) {
  if (index > createStep.value && !canAdvanceCreate.value) return;
  createStep.value = Math.min(2, Math.max(0, index)) as CreateStep;
}

function prevCreateStep() {
  if (createStep.value <= 0) return;
  createStep.value = (createStep.value - 1) as CreateStep;
}

function addPortRow() {
  createForm.value.ports.push({ host: '', container: '', protocol: 'tcp' });
}

function removePortRow(index: number) {
  createForm.value.ports.splice(index, 1);
  if (createForm.value.ports.length === 0) addPortRow();
}

function addMountRow() {
  createForm.value.mounts.push({ host: '', container: '', mode: 'rw' });
}

function removeMountRow(index: number) {
  createForm.value.mounts.splice(index, 1);
  if (createForm.value.mounts.length === 0) addMountRow();
}

function addKeyValueRow(kind: 'env' | 'labels' | 'extraHosts') {
  createForm.value[kind].push({ key: '', value: '' });
}

function removeKeyValueRow(kind: 'env' | 'labels' | 'extraHosts', index: number) {
  createForm.value[kind].splice(index, 1);
  if (createForm.value[kind].length === 0) addKeyValueRow(kind);
}

function openDeleteContainerDialog() {
  if (!selectedContainer.value) return;
  deleteContainerRemoveVolumes.value = false;
  deleteContainerDialogOpen.value = true;
}

function closeDeleteContainerDialog() {
  if (runningAction.value === selectedContainer.value?.id) return;
  deleteContainerDialogOpen.value = false;
}

async function confirmRemoveSelectedContainer() {
  const current = selectedContainer.value;
  if (!current) return;
  await runAction(current.id, async () => {
    await apiClient.docker.removeContainer(current.id, { force: true, removeVolumes: deleteContainerRemoveVolumes.value });
    deleteContainerDialogOpen.value = false;
    actionState.value = `${current.name} 已删除。`;
    await loadDockerRuntime(true);
  });
}

function resetTerminal() {
  closeTerminalSession();
  terminalStatus.value = 'idle';
  terminalOutput.value = '';
  terminalUnsupportedMessage.value = '';
  terminalDataDisposable?.dispose();
  terminalDataDisposable = undefined;
  terminalInstance?.dispose();
  terminalInstance = undefined;
}

function resetDockerDefaultView() {
  resourceTab.value = '概览';
  statusFilter.value = '全部';
  detailTab.value = '概览';
  query.value = '';
  resetTerminal();
}

function clearTerminalScreen() {
  terminalOutput.value = '';
  terminalInstance?.clear();
}

function appendTerminalOutput(text: string) {
  terminalOutput.value += text;
  terminalInstance?.write(text);
}

function ensureTerminalInstance() {
  if (!terminalScreenRef.value) return;
  if (!terminalInstance) {
    terminalInstance = new XTerm({
      cols: 120,
      rows: 28,
      cursorBlink: true,
      convertEol: true,
      disableStdin: false,
      fontFamily: "ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace",
      fontSize: 12,
      lineHeight: 1.35,
      scrollback: 5000,
      theme: {
        background: '#030712',
        foreground: '#dbeafe',
        cursor: '#93c5fd',
        selectionBackground: '#1d4ed8',
        black: '#020617',
        blue: '#3b82f6',
        cyan: '#22d3ee',
        green: '#22c55e',
        magenta: '#a855f7',
        red: '#ef4444',
        white: '#e5edf7',
        yellow: '#facc15',
        brightBlack: '#64748b',
        brightBlue: '#60a5fa',
        brightCyan: '#67e8f9',
        brightGreen: '#4ade80',
        brightMagenta: '#c084fc',
        brightRed: '#f87171',
        brightWhite: '#f8fafc',
        brightYellow: '#fde047',
      },
    });
    terminalInstance.open(terminalScreenRef.value);
    terminalDataDisposable = terminalInstance.onData((data) => {
      if (terminalStatus.value === 'ready' && terminalSocket) {
        terminalSocket.send(data);
      }
    });
  }
  terminalInstance.focus();
}

function closeTerminalSession() {
  if (terminalSocket) {
    const socket = terminalSocket;
    terminalSocket = undefined;
    socket.onopen = null;
    socket.onmessage = null;
    socket.onerror = null;
    socket.onclose = null;
    socket.close();
  }
}

async function openTerminalSession() {
  const current = selectedContainer.value;
  closeTerminalSession();
  terminalOutput.value = '';
  terminalUnsupportedMessage.value = '';
  await nextTick();
  ensureTerminalInstance();
  terminalInstance?.clear();
  if (!current) {
    terminalStatus.value = 'idle';
    appendTerminalOutput('请选择运行中的容器。\r\n');
    return;
  }
  if (current.status !== '运行中') {
    terminalStatus.value = 'unsupported';
    terminalUnsupportedMessage.value = '容器未运行，无法打开内部终端。';
    appendTerminalOutput(`${terminalUnsupportedMessage.value}\r\n`);
    return;
  }
  terminalStatus.value = 'connecting';
  appendTerminalOutput(`正在连接 ${current.name} ...\r\n`);
  const socket = new WebSocket(apiClient.docker.terminalUrl(current.id, terminalShell.value));
  terminalSocket = socket;
  socket.onopen = () => {
    terminalStatus.value = 'ready';
    terminalInstance?.focus();
  };
  socket.onmessage = (event) => {
    const message = String(event.data ?? '');
    if (/OCI runtime exec failed|executable file not found|no such file|not found|is not running|container .* not running/i.test(message)) {
      terminalStatus.value = 'unsupported';
      terminalUnsupportedMessage.value = `当前容器不支持 ${terminalShell.value} 终端。`;
    }
    appendTerminalOutput(message);
  };
  socket.onerror = () => {
    terminalStatus.value = 'unsupported';
    terminalUnsupportedMessage.value = `当前容器不支持 ${terminalShell.value} 终端，或 Docker 后端连接失败。`;
    appendTerminalOutput(`\r\n${terminalUnsupportedMessage.value}\r\n`);
  };
  socket.onclose = () => {
    if (terminalSocket === socket) {
      terminalSocket = undefined;
    }
    if (terminalStatus.value === 'connecting') {
      terminalStatus.value = 'unsupported';
      terminalUnsupportedMessage.value = `当前容器不支持 ${terminalShell.value} 终端。`;
      appendTerminalOutput(`\r\n${terminalUnsupportedMessage.value}\r\n`);
      return;
    }
    if (terminalStatus.value === 'ready') {
      terminalStatus.value = 'closed';
      appendTerminalOutput('\r\n终端连接已关闭。\r\n');
    }
  };
}

async function searchImages() {
  const text = imageQuery.value.trim();
  if (!text) return;
  await runAction('image-search', async () => {
    imageSearchResults.value = await apiClient.docker.searchImages(text);
    actionState.value = `找到 ${imageSearchResults.value.length} 个镜像结果。`;
  });
}

async function pullImage(image: string) {
  const name = image.trim();
  if (!name) return;
  await runAction(`pull-${name}`, async () => {
    const task = await apiClient.docker.pullImage({ image: name });
    mergeImagePull(task);
    resourceTab.value = '本地镜像';
    actionState.value = `${normalizeImageRef(name)} 已加入拉取队列。`;
    if (createDialogOpen.value && !createForm.value.image) createForm.value.image = resolveLocalImageRef(name);
    window.setTimeout(() => void loadDockerRuntime(true), 1200);
  });
}

async function removeImage(image: string) {
  if (!image || !window.confirm(`删除镜像 ${image}？`)) return;
  await runAction(`remove-image-${image}`, async () => {
    await apiClient.docker.removeImage({ image, force: false });
    actionState.value = `${image} 已删除。`;
    await loadDockerRuntime(true);
  });
}

async function createVolume() {
  const name = volumeForm.value.name.trim();
  if (!name) return;
  await runAction(`volume-${name}`, async () => {
    const next = await apiClient.docker.createVolume({ name, driver: volumeForm.value.driver.trim() || 'local' });
    volumeForm.value = { name: '', driver: 'local' };
    selectedVolumeName.value = next.name;
    actionState.value = `卷 ${next.name} 已创建。`;
    await loadDockerRuntime(true);
  });
}

async function removeVolume(name: string) {
  if (!name || !window.confirm(`删除存储卷 ${name}？\n\n只有未被容器使用的卷才能安全删除。`)) return;
  await runAction(`volume-remove-${name}`, async () => {
    await apiClient.docker.removeVolume({ name, force: false });
    actionState.value = `卷 ${name} 已删除。`;
    await loadDockerRuntime(true);
  });
}

async function createNetwork() {
  const name = networkForm.value.name.trim();
  if (!name) return;
  await runAction(`network-${name}`, async () => {
    const next = await apiClient.docker.createNetwork({ ...networkForm.value, name, driver: networkForm.value.driver.trim() || 'bridge' });
    networkForm.value = { name: '', driver: 'bridge', subnet: '', gateway: '', attachable: false, internal: false };
    selectedNetworkName.value = next.name;
    actionState.value = `网络 ${next.name} 已创建。`;
    await loadDockerRuntime(true);
  });
}

async function removeNetwork(name: string) {
  if (!name || ['bridge', 'host', 'none'].includes(name) || !window.confirm(`删除网络 ${name}？`)) return;
  await runAction(`network-remove-${name}`, async () => {
    await apiClient.docker.removeNetwork({ name });
    actionState.value = `网络 ${name} 已删除。`;
    await loadDockerRuntime(true);
  });
}

async function connectNetwork() {
  if (!selectedNetworkName.value || !networkAttachForm.value.container) return;
  await runAction(`network-connect-${selectedNetworkName.value}`, async () => {
    await apiClient.docker.connectNetwork({
      network: selectedNetworkName.value,
      container: networkAttachForm.value.container,
      alias: networkAttachForm.value.alias.trim(),
    });
    actionState.value = `${networkAttachForm.value.container} 已连接到 ${selectedNetworkName.value}。`;
    networkAttachForm.value.alias = '';
    await loadDockerRuntime(true);
  });
}

async function disconnectNetwork(container: string) {
  if (!selectedNetworkName.value || !container || !window.confirm(`从 ${selectedNetworkName.value} 断开 ${container}？`)) return;
  await runAction(`network-disconnect-${selectedNetworkName.value}-${container}`, async () => {
    await apiClient.docker.disconnectNetwork({ network: selectedNetworkName.value, container, force: true });
    actionState.value = `${container} 已从 ${selectedNetworkName.value} 断开。`;
    await loadDockerRuntime(true);
  });
}

async function runAction(key: string, action: () => Promise<void>) {
  runningAction.value = key;
  try {
    await action();
  } catch (error) {
    actionState.value = errorMessage(error);
  } finally {
    runningAction.value = '';
  }
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

function splitInput(value: string) {
  return value
    .split(/[\n,]+/)
    .map((item) => item.trim())
    .filter(Boolean);
}

function formatKeyValueRows(rows: KeyValueRow[]) {
  return rows
    .map((row) => {
      const key = row.key.trim();
      if (!key) return '';
      return `${key}=${row.value.trim()}`;
    })
    .filter(Boolean);
}

function statusTone(status: string) {
  if (status === '运行中') return 'running';
  if (status === '重启中') return 'restarting';
  return 'stopped';
}

function mergeImagePull(task: DockerImagePullStatus) {
  const nextID = task.id || task.image;
  imagePulls.value = [task, ...imagePulls.value.filter((item) => (item.id || item.image) !== nextID)];
}

function imageRef(image: DockerImage) {
  return image.tag && image.tag !== '<none>' ? `${image.repository}:${image.tag}` : image.id;
}

function normalizeImageRef(image = '') {
  const trimmed = image.trim();
  if (!trimmed) return '';
  const withoutLibrary = trimmed.startsWith('library/') ? trimmed.slice('library/'.length) : trimmed;
  const slashIndex = withoutLibrary.lastIndexOf('/');
  const colonIndex = withoutLibrary.lastIndexOf(':');
  if (colonIndex > slashIndex) return withoutLibrary;
  if (withoutLibrary.startsWith('sha256:') || /^[a-f0-9]{12,}$/i.test(withoutLibrary)) return withoutLibrary;
  return `${withoutLibrary}:latest`;
}

function imageAliases(image = '') {
  const normalized = normalizeImageRef(image);
  const aliases = new Set([normalized]);
  if (normalized.endsWith(':latest')) aliases.add(normalized.slice(0, -':latest'.length));
  if (!normalized.includes('/')) aliases.add(`library/${normalized}`);
  return [...aliases].filter(Boolean);
}

function hasLocalImage(image = '') {
  return imageAliases(image).some((alias) => downloadedImageRefs.value.has(alias));
}

function isPullingImage(image = '') {
  return imageAliases(image).some((alias) => activePullRefs.value.has(alias));
}

function pullStatusLabel(status = '') {
  if (status === 'queued') return '等待拉取';
  if (status === 'running') return '拉取中';
  if (status === 'completed') return '拉取完成';
  if (status === 'failed') return '拉取失败';
  return status || '拉取中';
}

function safeProgress(progress: number) {
  if (!Number.isFinite(progress)) return 0;
  return Math.min(100, Math.max(0, Math.round(progress)));
}

function imageRepositoryPath(image = '') {
  let ref = normalizeImageRef(image).trim();
  if (!ref || ref === '<none>' || ref.startsWith('sha256:') || /^[a-f0-9]{12,}$/i.test(ref)) return '';
  ref = ref.replace(/^docker\.io\//, '').replace(/^index\.docker\.io\//, '');
  const atIndex = ref.indexOf('@');
  if (atIndex >= 0) ref = ref.slice(0, atIndex);
  const slashIndex = ref.lastIndexOf('/');
  const colonIndex = ref.lastIndexOf(':');
  if (colonIndex > slashIndex) ref = ref.slice(0, colonIndex);
  const parts = ref.split('/').filter(Boolean);
  if (parts.length === 0) return '';
  if (parts[0].includes('.') || parts[0].includes(':') || parts[0] === 'localhost') return '';
  return parts.length === 1 ? `library/${parts[0]}` : parts.slice(0, 2).join('/');
}

function imageIconKey(image = '', iconUrl = '') {
  return iconUrl || imageRepositoryPath(image) || normalizeImageRef(image);
}

function imageIconUrl(image = '', iconUrl = '') {
  if (iconUrl.trim()) return iconUrl.trim();
  const repo = imageRepositoryPath(image);
  return repo ? `https://hub.docker.com/api/media/repos_logo/v1/${encodeURIComponent(repo)}` : '';
}

function shouldUseExternalImageIcon(image = '', iconUrl = '') {
  const url = imageIconUrl(image, iconUrl);
  if (!url) return false;
  return !imageIconFailures.value.has(imageIconKey(image, iconUrl));
}

function markImageIconFailed(image = '', iconUrl = '') {
  const key = imageIconKey(image, iconUrl);
  if (!key) return;
  const next = new Set(imageIconFailures.value);
  next.add(key);
  imageIconFailures.value = next;
}

function resolveLocalImageRef(image = '') {
  const requested = image.trim();
  if (!requested) return '';
  const refs = localImageOptions.value.map((item) => item.ref);
  return refs.find((ref) => ref === requested) ?? refs.find((ref) => ref === `${requested}:latest`) ?? '';
}

function metricNumber(key: MonitorKey) {
  if (key === 'cpu') return totalCpu.value;
  if (key === 'memory') return totalMemory.value;
  const value = metricMap.value[key]?.value;
  const parsed = typeof value === 'number' ? value : Number.parseFloat(String(value ?? '0'));
  return Number.isFinite(parsed) ? parsed : 0;
}

function formatMetric(key: MonitorKey) {
  const metric = metricMap.value[key];
  const value = metricNumber(key);
  const unit = metric?.unit ?? (key === 'network' ? 'MB/s' : '%');
  const fixed = key === 'network' ? (value >= 10 ? value.toFixed(1) : value.toFixed(2)) : value.toFixed(value % 1 === 0 ? 0 : 1);
  return `${fixed.replace(/\.0$/, '')}${unit}`;
}

function updateMetricHistory(metrics: Metric[], trends: Record<MonitorKey, TrendPoint[]>) {
  systemMetrics.value = metrics;
  const next = { ...metricHistory.value };
  monitorKeys.forEach((key) => {
    const trendValues = key === 'network' ? trends[key]
      .map((point) => Number(point.value))
      .filter((value) => Number.isFinite(value)) : [];
    const current = (() => {
      if (key === 'cpu') return totalCpu.value;
      if (key === 'memory') return totalMemory.value;
      const metric = metrics.find((item) => item.key === key);
      const parsed = typeof metric?.value === 'number' ? metric.value : Number.parseFloat(String(metric?.value ?? '0'));
      return Number.isFinite(parsed) ? parsed : 0;
    })();
    const source = trendValues.length > 1 ? trendValues : [...next[key], current];
    next[key] = source.slice(-28);
  });
  metricHistory.value = next;
}

function chartSeries(key: MonitorKey, current: number) {
  const values = metricHistory.value[key].length ? metricHistory.value[key] : [current, current];
  return values.length === 1 ? [values[0], values[0]] : values;
}

function smoothTrendPath(values: number[], width: number, height: number, fixedMax?: number) {
  return smoothTrendLine(values, width, height, fixedMax);
}

function smoothTrendAreaPath(values: number[], width: number, height: number, fixedMax?: number) {
  const line = smoothTrendLine(values, width, height, fixedMax);
  if (!line) return '';
  const baseline = height - 6;
  return `${line} L ${width.toFixed(1)} ${baseline.toFixed(1)} L 0 ${baseline.toFixed(1)} Z`;
}

function smoothTrendLine(values: number[], width: number, height: number, fixedMax?: number) {
  if (!values.length) return '';
  const max = fixedMax ?? Math.max(...values, 1);
  const min = fixedMax ? 0 : Math.min(...values, 0);
  const range = Math.max(max - min, 1);
  const points = values.map((value, index) => {
    const x = (index / Math.max(values.length - 1, 1)) * width;
    const normalized = (Math.min(max, Math.max(min, value)) - min) / range;
    const y = height - normalized * (height - 18) - 9;
    return { x, y };
  });
  return points
    .map((point, index) => {
      if (index === 0) return `M ${point.x.toFixed(1)} ${point.y.toFixed(1)}`;
      const previous = points[index - 1];
      const midX = (previous.x + point.x) / 2;
      return `C ${midX.toFixed(1)} ${previous.y.toFixed(1)} ${midX.toFixed(1)} ${point.y.toFixed(1)} ${point.x.toFixed(1)} ${point.y.toFixed(1)}`;
    })
    .join(' ');
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

function syncDockerTheme() {
  if (typeof document === 'undefined' || typeof window === 'undefined') return;
  const desktop = document.querySelector('.desktop');
  const explicitDark =
    desktop?.classList.contains('desktop--theme-dark') ||
    document.documentElement.dataset.theme === 'dark' ||
    document.body.dataset.theme === 'dark' ||
    document.documentElement.classList.contains('theme-dark') ||
    document.body.classList.contains('theme-dark');
  const autoDark = Boolean(desktop?.classList.contains('desktop--theme-auto') && window.matchMedia('(prefers-color-scheme: dark)').matches);
  isDarkTheme.value = Boolean(explicitDark || autoDark);
}

watch([detailTab, selectedContainerId, terminalShell], ([tab]) => {
  if (tab === '终端') {
    void openTerminalSession();
  } else {
    resetTerminal();
  }
});

watch(
  () => props.openToken,
  () => {
    resetDockerDefaultView();
  },
  { immediate: true },
);

onMounted(() => {
  syncDockerTheme();
  const desktop = document.querySelector('.desktop');
  themeObserver = new MutationObserver(syncDockerTheme);
  if (desktop) themeObserver.observe(desktop, { attributes: true, attributeFilter: ['class'] });
  themeObserver.observe(document.documentElement, { attributes: true, attributeFilter: ['class', 'data-theme'] });
  themeObserver.observe(document.body, { attributes: true, attributeFilter: ['class', 'data-theme'] });
  themeMediaQuery = window.matchMedia('(prefers-color-scheme: dark)');
  themeMediaQuery.addEventListener('change', syncDockerTheme);
  void loadDockerRuntime();
  refreshTimer = window.setInterval(() => {
    void loadDockerRuntime(true);
  }, 15000);
});

onUnmounted(() => {
  if (refreshTimer !== undefined) window.clearInterval(refreshTimer);
  closeTerminalSession();
  terminalDataDisposable?.dispose();
  terminalInstance?.dispose();
  themeObserver?.disconnect();
  themeMediaQuery?.removeEventListener('change', syncDockerTheme);
});
</script>

<template>
  <div class="docker-window" :class="{ 'docker-window--dark': isDarkTheme }">
    <aside class="docker-sidebar" aria-label="Docker 资源">
      <header class="docker-sidebar__header">
        <h3>
          <svg class="docker-glyph" viewBox="0 0 1024 1024" aria-hidden="true">
            <path :d="dockerIconPath" />
          </svg>
          Docker
        </h3>
        <button type="button" :disabled="loading" @click="loadDockerRuntime()">
          <RefreshCw :size="13" />刷新
        </button>
      </header>

      <nav class="docker-resource-tabs" aria-label="资源类型">
        <button
          v-for="tab in resourceTabs"
          :key="tab"
          type="button"
          :class="{ 'is-active': resourceTab === tab }"
          @click="resourceTab = tab"
        >
          <span>
            <Boxes v-if="tab === '概览'" :size="16" />
            <Container v-else-if="tab === '容器'" :size="16" />
            <Boxes v-else-if="tab === 'Compose'" :size="16" />
            <HardDrive v-else-if="tab === '本地镜像'" :size="16" />
            <ImageDown v-else-if="tab === '镜像仓库'" :size="16" />
            <Network v-else-if="tab === '网络'" :size="16" />
            <Database v-else :size="16" />
            {{ tab }}
          </span>
          <b>{{ resourceCounts[tab] }}</b>
        </button>
      </nav>

      <section class="docker-sidebar__block docker-stack-list">
        <h3><Boxes :size="14" /> Compose 栈</h3>
        <button
          v-for="stack in stackScopes"
          :key="stack.name"
          type="button"
          :class="{ 'is-active': selectedStackName === stack.name }"
          @click="selectStack(stack.name)"
        >
          <span>
            <strong>{{ stack.name === 'all' ? '全部容器' : stack.name }}</strong>
            <small>{{ stack.services }} 个容器 · {{ stack.status }}</small>
          </span>
          <b>{{ stack.name === 'all' ? runningCount : stack.services }}</b>
        </button>
      </section>
    </aside>

    <main class="docker-main">
      <section v-if="resourceTab === '概览'" class="docker-summary" aria-label="Docker 概览">
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
          <span>网络</span>
          <strong>{{ networks.length }}</strong>
        </article>
        <article>
          <HardDrive :size="18" />
          <span>镜像</span>
          <strong>{{ images.length }}</strong>
        </article>
      </section>

      <section v-if="resourceTab === '概览'" class="docker-panel docker-overview-page">
        <div class="docker-overview-grid">
          <article class="docker-hero-card">
            <div>
              <h2>{{ containers.length ? `${runningCount} 个容器运行` : '无容器运行' }}</h2>
              <p>{{ containers.length ? 'Docker 运行状态已同步。' : '当前 Docker 没有运行中的业务容器。' }}</p>
            </div>
            <ul>
              <li v-for="row in overviewRows" :key="row.label">
                <span>{{ row.label }}</span>
                <strong>{{ row.value }}</strong>
              </li>
            </ul>
          </article>
          <article class="docker-service-card">
            <header>
              <div>
                <h3>Docker 服务</h3>
                <p>{{ actionState }}</p>
              </div>
              <button class="docker-button docker-button--ghost" type="button" :disabled="loading" @click="loadDockerRuntime()">
                <RefreshCw :size="13" />刷新
              </button>
            </header>
            <div class="docker-service-row">
              <span>本地镜像</span>
              <strong>{{ images.length }}</strong>
            </div>
            <div class="docker-service-row">
              <span>网络</span>
              <strong>{{ networks.length }}</strong>
            </div>
            <div class="docker-service-row">
              <span>存储卷</span>
              <strong>{{ volumes.length }}</strong>
            </div>
          </article>
        </div>
        <section class="docker-charts">
          <h3>信息监控</h3>
          <div class="docker-chart-grid">
            <article class="docker-chart docker-chart--cpu">
              <header><span>Docker CPU 使用率</span><strong>{{ formatMetric('cpu') }}</strong></header>
              <svg viewBox="0 0 300 80" preserveAspectRatio="none">
                <path class="docker-chart__area" :d="cpuAreaPath" />
                <path class="docker-chart__line" :d="cpuPath" />
              </svg>
            </article>
            <article class="docker-chart docker-chart--memory">
              <header><span>Docker 内存使用率</span><strong>{{ formatMetric('memory') }}</strong></header>
              <svg viewBox="0 0 300 80" preserveAspectRatio="none">
                <path class="docker-chart__area" :d="memoryAreaPath" />
                <path class="docker-chart__line" :d="memoryPath" />
              </svg>
            </article>
          </div>
          <article class="docker-chart-wide docker-chart docker-chart--network">
            <header><span>网络实时吞吐</span><strong>{{ formatMetric('network') }}</strong></header>
            <svg viewBox="0 0 640 90" preserveAspectRatio="none">
              <path class="docker-chart__area" :d="networkAreaPath" />
              <path class="docker-chart__line" :d="networkPath" />
            </svg>
          </article>
        </section>
      </section>

      <section v-else-if="resourceTab === '容器'" class="docker-panel docker-containers">
        <header class="docker-panel__header">
          <div class="docker-search">
            <Search :size="15" />
            <input v-model="query" type="search" placeholder="搜索容器、镜像、端口、路径" />
          </div>
          <div class="docker-pills">
            <button
              v-for="status in statusTabs"
              :key="status"
              type="button"
              :class="{ 'is-active': statusFilter === status }"
              @click="statusFilter = status"
            >
              {{ status }} {{ statusCounts[status] }}
            </button>
          </div>
          <button class="docker-button docker-button--primary" type="button" @click="openCreateContainerDialog()">
            <Plus :size="13" />添加容器
          </button>
        </header>

        <div class="docker-container-workspace" :class="{ 'has-detail': selectedContainer }">
          <div class="docker-container-list">
            <div class="docker-table">
              <div class="docker-table__head">
                <span>容器</span>
                <span>状态</span>
                <span>镜像</span>
                <span>CPU</span>
                <span>内存</span>
                <span>端口</span>
              </div>
              <button
                v-for="container in filteredContainers"
                :key="container.id"
                class="docker-row"
                :class="{ 'is-active': selectedContainerId === container.id }"
                type="button"
                @click="selectContainer(container.id)"
              >
                <span>
                  <strong>{{ container.name }}</strong>
                  <small>{{ container.stack }}</small>
                </span>
                <b :class="`docker-status docker-status--${statusTone(container.status)}`">
                  <CircleDot :size="9" />{{ container.status }}
                </b>
                <span>{{ container.image }}</span>
                <span>{{ container.cpu }}%</span>
                <span>{{ container.memoryText }}</span>
                <span>{{ container.ports[0] }}</span>
              </button>
              <div v-if="filteredContainers.length === 0" class="docker-empty">
                <Container :size="22" />
                <strong>没有容器</strong>
                <span>可以先拉取镜像，或添加一个容器。</span>
              </div>
            </div>
          </div>

          <section v-if="selectedContainer" class="docker-detail docker-detail--side">
            <header>
              <div>
                <h3>{{ selectedContainer.name }}</h3>
                <p>{{ selectedContainer.id }} · {{ selectedContainer.image }}</p>
              </div>
              <div class="docker-actions">
                <button v-if="selectedContainer.status !== '运行中'" type="button" @click="setContainerStatus('运行中')">
                  <Play :size="13" />启动
                </button>
                <button v-if="selectedContainer.status === '运行中'" type="button" @click="setContainerStatus('已停止')">
                  <Square :size="13" />停止
                </button>
                <button type="button" @click="restartContainer"><RotateCw :size="13" />重启</button>
                <button class="docker-button--danger" type="button" @click="openDeleteContainerDialog"><Trash2 :size="13" />删除</button>
              </div>
            </header>

            <nav class="docker-detail-tabs">
              <button
                v-for="tab in detailTabs"
                :key="tab"
                type="button"
                :class="{ 'is-active': detailTab === tab }"
                @click="detailTab = tab"
              >
                {{ tab }}
              </button>
            </nav>

            <div v-if="detailTab === '概览'" class="docker-detail-grid">
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
                  <input v-model.number="selectedContainer.limitMemory" type="range" min="512" max="12288" step="512" @change="updateLimit('memory')" />
                </label>
              </article>
              <article>
                <strong><CheckCircle2 :size="14" /> 隔离策略</strong>
                <span>{{ selectedContainer.isolation }}</span>
              </article>
            </div>

            <div v-else-if="detailTab === '日志'" class="docker-console docker-console--logs">
              <div class="docker-console__bar">
                <span>最近 200 行</span>
                <button type="button" @click="refreshSelectedContainerLogs()"><RefreshCw :size="13" />刷新</button>
              </div>
              <textarea class="docker-log-editor" :value="selectedLogText || '暂无日志。'" readonly spellcheck="false" aria-label="容器日志"></textarea>
            </div>

            <div v-else-if="detailTab === '终端'" class="docker-command docker-terminal">
              <div class="docker-terminal__toolbar">
                <select v-model="terminalShell" aria-label="切换 shell 环境">
                  <option value="/bin/sh">/bin/sh</option>
                  <option value="/bin/bash">/bin/bash</option>
                  <option value="/bin/ash">/bin/ash</option>
                </select>
                <button type="button" @click="openTerminalSession"><RefreshCw :size="13" />重连</button>
                <button type="button" @click="clearTerminalScreen">清屏</button>
              </div>
              <div
                ref="terminalScreenRef"
                class="docker-terminal__screen"
                role="textbox"
                aria-label="容器终端"
                tabindex="0"
              ></div>
            </div>

            <div v-else class="docker-kv-list">
              <span v-for="item in detailTab === '端口' ? selectedContainer.ports : detailTab === '挂载' ? selectedContainer.mounts : selectedContainer.env" :key="item">
                {{ item }}
              </span>
            </div>
          </section>
        </div>
      </section>

      <section v-else-if="resourceTab === 'Compose'" class="docker-panel docker-compose-page">
        <header class="docker-panel__header">
          <div>
            <h3>Compose</h3>
            <p>{{ composeStacks.length }} 个栈，{{ containers.length }} 个容器。</p>
          </div>
          <button class="docker-primary" type="button" :disabled="loading" @click="loadDockerRuntime()">
            <RefreshCw :size="13" />刷新
          </button>
        </header>
        <div class="docker-list docker-list--grid">
          <article v-for="stack in composeStacks" :key="stack.name" class="docker-list-item">
            <div>
              <strong>{{ stack.name }}</strong>
              <p>{{ stack.status }} · {{ stack.services }} 个服务 · {{ stack.ports }}</p>
              <small>{{ stack.volume }} · {{ stack.network }}</small>
            </div>
            <button type="button" @click="selectStack(stack.name); resourceTab = '容器'">查看容器</button>
          </article>
          <div v-if="composeStacks.length === 0" class="docker-empty docker-empty--small">
            <Boxes :size="20" />
            <strong>暂无 Compose 栈</strong>
          </div>
        </div>
      </section>

      <section v-else-if="resourceTab === '本地镜像'" class="docker-panel docker-local-images">
        <header class="docker-panel__header">
          <div>
            <h3>本地镜像</h3>
            <p>{{ localImages.length }} 个镜像，{{ visibleImagePulls.length }} 个拉取任务。</p>
          </div>
          <button class="docker-primary" type="button" @click="loadDockerRuntime()"><RefreshCw :size="13" />刷新</button>
        </header>
        <div class="docker-list docker-list--images">
          <article v-for="task in visibleImagePulls" :key="`pull-${task.id || task.image}`" class="docker-list-item docker-image-item docker-image-item--pull">
            <span class="docker-image-icon" :title="imageRepositoryPath(task.image) || normalizeImageRef(task.image)">
              <img
                v-if="shouldUseExternalImageIcon(task.image)"
                :src="imageIconUrl(task.image)"
                alt=""
                loading="lazy"
                @error="markImageIconFailed(task.image)"
              />
              <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
                <path :d="dockerIconPath" />
              </svg>
            </span>
            <div class="docker-image-main">
              <div class="docker-image-title-row">
                <strong>{{ normalizeImageRef(task.image) }}</strong>
                <span class="docker-status-chip" :class="{ 'docker-status-chip--danger': task.status === 'failed' }">{{ pullStatusLabel(task.status) }}</span>
              </div>
              <p>{{ task.message || pullStatusLabel(task.status) }} · {{ task.downloaded || '0 B' }} / {{ task.total || '未知大小' }} · {{ task.speed || '0 B/s' }}</p>
              <div class="docker-image-progress" role="progressbar" :aria-valuenow="safeProgress(task.progress)" aria-valuemin="0" aria-valuemax="100">
                <i :style="{ width: `${safeProgress(task.progress)}%` }"></i>
              </div>
              <small v-if="task.error">{{ task.error }}</small>
            </div>
          </article>
          <article v-for="image in localImages" :key="`${image.id}-${image.repository}-${image.tag}`" class="docker-list-item docker-image-item">
            <span class="docker-image-icon" :title="imageRepositoryPath(image.ref) || image.ref">
              <img
                v-if="shouldUseExternalImageIcon(image.ref, image.iconUrl)"
                :src="imageIconUrl(image.ref, image.iconUrl)"
                alt=""
                loading="lazy"
                @error="markImageIconFailed(image.ref, image.iconUrl)"
              />
              <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
                <path :d="dockerIconPath" />
              </svg>
            </span>
            <div class="docker-image-main">
              <strong>{{ image.ref }}</strong>
              <p>{{ image.id }} · {{ image.size }} · {{ image.created }}</p>
            </div>
            <div class="docker-row-actions">
              <button type="button" @click="openCreateContainerDialog(imageRef(image)); resourceTab = '容器'"><Play :size="13" />运行</button>
              <button type="button" @click="removeImage(imageRef(image))"><Trash2 :size="13" />删除</button>
            </div>
          </article>
          <div v-if="localImages.length === 0 && visibleImagePulls.length === 0" class="docker-empty docker-empty--small">
            <Boxes :size="20" />
            <strong>暂无本地镜像</strong>
          </div>
        </div>
      </section>

      <section v-else-if="resourceTab === '镜像仓库'" class="docker-panel docker-images">
        <header class="docker-panel__header">
          <div class="docker-search">
            <Search :size="15" />
            <input v-model="imageQuery" type="search" placeholder="搜索镜像，例如 nginx、redis、homeassistant" @keyup.enter="searchImages" />
          </div>
          <button class="docker-primary" type="button" :disabled="!imageQuery.trim()" @click="searchImages">
            <Search :size="13" />搜索
          </button>
        </header>

        <div class="docker-list docker-list--images">
          <article v-for="result in imageSearchResults" :key="result.name" class="docker-list-item docker-image-item">
            <span class="docker-image-icon" :title="imageRepositoryPath(result.name) || result.name">
              <img
                v-if="shouldUseExternalImageIcon(result.name, result.iconUrl)"
                :src="imageIconUrl(result.name, result.iconUrl)"
                alt=""
                loading="lazy"
                @error="markImageIconFailed(result.name, result.iconUrl)"
              />
              <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
                <path :d="dockerIconPath" />
              </svg>
            </span>
            <div class="docker-image-main">
              <strong>{{ result.name }}</strong>
              <p>{{ result.description || '无描述' }}</p>
              <small>{{ result.stars }} stars <template v-if="result.official"> · 官方</template><template v-if="result.automated"> · 自动构建</template></small>
            </div>
            <button type="button" :disabled="isPullingImage(result.name) || hasLocalImage(result.name)" @click="pullImage(result.name)">
              <ImageDown :size="13" />{{ isPullingImage(result.name) ? '拉取中' : hasLocalImage(result.name) ? '已下载' : '拉取' }}
            </button>
          </article>
          <div v-if="imageSearchResults.length === 0" class="docker-empty docker-empty--small">
            <ImageDown :size="20" />
            <strong>输入镜像名开始搜索</strong>
          </div>
        </div>
      </section>

      <section v-else-if="resourceTab === '网络'" class="docker-panel docker-networks">
        <header class="docker-panel__header docker-form-row">
          <input v-model="networkForm.name" placeholder="网络名称" />
          <input v-model="networkForm.driver" placeholder="驱动，例如 bridge" />
          <input v-model="networkForm.subnet" placeholder="子网，可选" />
          <input v-model="networkForm.gateway" placeholder="网关，可选" />
          <label><input v-model="networkForm.attachable" type="checkbox" /> 可连接</label>
          <label><input v-model="networkForm.internal" type="checkbox" /> 内部网络</label>
          <button class="docker-primary" type="button" :disabled="!networkForm.name.trim()" @click="createNetwork">
            <Network :size="13" />创建
          </button>
        </header>

        <div class="docker-split docker-split--wide-left">
          <section class="docker-card">
            <header><h3><Network :size="15" /> 网络列表</h3></header>
            <div class="docker-list">
              <button
                v-for="network in networks"
                :key="network.id"
                class="docker-list-item docker-list-button"
                :class="{ 'is-active': selectedNetworkName === network.name }"
                type="button"
                @click="selectedNetworkName = network.name"
              >
                <div>
                  <strong>{{ network.name }}</strong>
                  <p>{{ network.driver }} · {{ network.scope }} · {{ network.subnet || '无子网' }}</p>
                  <small>{{ network.containers?.length || 0 }} 个容器</small>
                </div>
                <span>{{ network.internal ? '内部' : '普通' }}</span>
              </button>
            </div>
          </section>

          <section class="docker-card">
            <header>
              <h3><Plug :size="15" /> 网络配置</h3>
              <button v-if="selectedNetwork" type="button" @click="removeNetwork(selectedNetwork.name)"><Trash2 :size="13" />删除</button>
            </header>
            <div v-if="selectedNetwork" class="docker-network-detail">
              <p><strong>{{ selectedNetwork.name }}</strong></p>
              <p>驱动：{{ selectedNetwork.driver }}</p>
              <p>子网：{{ selectedNetwork.subnet || '-' }}</p>
              <p>网关：{{ selectedNetwork.gateway || '-' }}</p>
              <div class="docker-command__input">
                <select v-model="networkAttachForm.container">
                  <option v-for="container in containers" :key="container.id" :value="container.name">{{ container.name }}</option>
                </select>
                <input v-model="networkAttachForm.alias" placeholder="别名，可选" />
                <button type="button" @click="connectNetwork">连接</button>
              </div>
              <div class="docker-kv-list">
                <span v-for="container in selectedNetwork.containers" :key="container">
                  {{ container }}
                  <button type="button" @click="disconnectNetwork(container)">断开</button>
                </span>
                <span v-if="!selectedNetwork.containers?.length">暂无已连接容器。</span>
              </div>
            </div>
          </section>
        </div>
      </section>

      <section v-else class="docker-panel docker-volumes">
        <header class="docker-panel__header docker-form-row">
          <input v-model="volumeForm.name" placeholder="卷名称" />
          <input v-model="volumeForm.driver" placeholder="驱动，例如 local" />
          <button class="docker-primary" type="button" :disabled="!volumeForm.name.trim()" @click="createVolume">
            <Database :size="13" />创建卷
          </button>
        </header>

        <div class="docker-card">
          <header><h3><Database :size="15" /> 存储卷</h3></header>
          <div class="docker-list docker-list--grid">
            <article
              v-for="volume in volumes"
              :key="volume.name"
              class="docker-list-item"
              :class="{ 'is-active': selectedVolumeName === volume.name }"
              @click="selectedVolumeName = volume.name"
            >
              <div>
                <strong>{{ volume.name }}</strong>
                <p>{{ volume.driver }} · {{ volume.scope }}</p>
                <small>{{ volume.mountpoint || '无挂载点' }}</small>
              </div>
              <button type="button" @click.stop="removeVolume(volume.name)"><Trash2 :size="13" />删除</button>
            </article>
            <div v-if="volumes.length === 0" class="docker-empty docker-empty--small">
              <Database :size="20" />
              <strong>暂无存储卷</strong>
            </div>
          </div>
        </div>
      </section>
    </main>

    <aside class="docker-inspector" aria-label="Docker 状态">
      <header>
        <h3><ListFilter :size="15" /> 状态</h3>
        <span>{{ actionState }}</span>
      </header>
      <section>
        <strong>容器</strong>
        <p>{{ runningCount }} 运行 · {{ stoppedCount }} 停止 · {{ restartingCount }} 重启中</p>
      </section>
      <section>
        <strong>资源</strong>
        <p>{{ images.length }} 镜像 · {{ volumes.length }} 卷 · {{ networks.length }} 网络 · {{ allExposedPorts.length }} 端口</p>
      </section>
      <section v-if="selectedContainer">
        <strong>当前容器</strong>
        <p>{{ selectedContainer.name }}</p>
        <p>{{ selectedContainer.status }} · {{ selectedContainer.memoryText }}</p>
      </section>
      <section v-if="selectedVolume">
        <strong>当前卷</strong>
        <p>{{ selectedVolume.name }}</p>
        <p>{{ selectedVolume.mountpoint || selectedVolume.driver }}</p>
      </section>
    </aside>

    <div v-if="createDialogOpen || deleteContainerDialogOpen" class="docker-modal-layer" @click.self="createDialogOpen ? closeCreateContainerDialog() : closeDeleteContainerDialog()">
      <section v-if="createDialogOpen" class="docker-modal docker-modal--wide" role="dialog" aria-modal="true" aria-labelledby="docker-create-title">
        <header class="docker-modal__header">
          <div>
            <h3 id="docker-create-title"><Container :size="15" /> 添加容器</h3>
            <p>按步骤配置镜像、权限、网络、挂载和启动参数。</p>
          </div>
          <button class="docker-modal__close" type="button" aria-label="关闭" @click="closeCreateContainerDialog">
            <X :size="15" />
          </button>
        </header>
        <div class="docker-modal__body docker-wizard">
          <nav class="docker-wizard-steps" aria-label="创建步骤">
            <button
              v-for="(step, index) in createSteps"
              :key="step"
              type="button"
              :class="{ 'is-active': createStep === index, 'is-done': createStep > index }"
              :disabled="index > createStep && !canAdvanceCreate"
              @click="setCreateStep(index)"
            >
              <span>{{ index + 1 }}</span>{{ step }}
            </button>
          </nav>

          <section v-if="createStep === 0" class="docker-wizard-panel">
            <div class="docker-form-grid">
              <label class="docker-field docker-field--required">
                <span>镜像</span>
                <select v-model="createForm.image" :disabled="localImageOptions.length === 0">
                  <option value="" disabled>{{ localImageOptions.length ? '选择本地镜像' : '暂无本地镜像' }}</option>
                  <option v-for="image in localImageOptions" :key="`${image.id}-${image.ref}`" :value="image.ref">
                    {{ image.ref }} · {{ image.size }}
                  </option>
                </select>
                <small v-if="localImageOptions.length === 0">请先到镜像仓库拉取镜像，或刷新本地镜像列表。</small>
              </label>
              <label class="docker-field">
                <span>容器名</span>
                <input v-model="createForm.name" placeholder="留空则自动生成" autocomplete="off" />
              </label>
            </div>

            <section class="docker-wizard-section">
              <header>
                <h4><ShieldCheck :size="14" /> 权限</h4>
              </header>
              <div class="docker-choice-grid">
                <button type="button" :class="{ 'is-active': !createForm.privileged }" @click="createForm.privileged = false">
                  <strong>普通权限</strong>
                  <span>适合大多数服务，隔离性更好。</span>
                </button>
                <button type="button" :class="{ 'is-active': createForm.privileged }" @click="createForm.privileged = true">
                  <strong>高权限</strong>
                  <span>允许访问更多宿主机能力，仅在硬件直通等场景使用。</span>
                </button>
              </div>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><RotateCw :size="14" /> 重启与资源</h4>
              </header>
              <div class="docker-form-grid">
                <label class="docker-field">
                  <span>重启策略</span>
                  <select v-model="createForm.restartPolicy" :disabled="createForm.autoRemove">
                    <option v-for="policy in restartPolicies" :key="policy.value" :value="policy.value">{{ policy.label }}</option>
                  </select>
                </label>
                <label class="docker-check-row docker-check-row--card">
                  <input v-model="createForm.autoRemove" type="checkbox" />
                  <span>退出后自动删除容器</span>
                </label>
                <label class="docker-field">
                  <span>CPU 限制</span>
                  <input v-model.number="createForm.limitCpu" type="number" min="0" placeholder="0 表示不限制" />
                </label>
                <label class="docker-field">
                  <span>内存限制 MB</span>
                  <input v-model.number="createForm.limitMemory" type="number" min="0" step="128" placeholder="0 表示不限制" />
                </label>
              </div>
            </section>
          </section>

          <section v-else-if="createStep === 1" class="docker-wizard-panel">
            <section class="docker-wizard-section">
              <header>
                <h4><Network :size="14" /> 网络</h4>
              </header>
              <div class="docker-form-grid docker-form-grid--three">
                <label class="docker-field">
                  <span>网络模式</span>
                  <select v-model="createForm.network">
                    <option v-for="network in networkOptions" :key="network" :value="network">{{ network }}</option>
                  </select>
                </label>
                <label class="docker-field">
                  <span>主机名</span>
                  <input v-model="createForm.hostname" placeholder="可选" />
                </label>
                <label class="docker-field">
                  <span>运行用户</span>
                  <input v-model="createForm.user" placeholder="例如 1000:1000" />
                </label>
              </div>
              <label class="docker-field docker-field--span">
                <span>DNS</span>
                <input v-model="createForm.dns" placeholder="多个 DNS 用逗号分隔，例如 223.5.5.5,119.29.29.29" />
              </label>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><Plug :size="14" /> 端口映射</h4>
                <button type="button" @click="addPortRow"><Plus :size="12" />添加</button>
              </header>
              <div class="docker-dynamic-list">
                <div v-for="(port, index) in createForm.ports" :key="`port-${index}`" class="docker-dynamic-row docker-dynamic-row--ports">
                  <input v-model="port.host" placeholder="本地端口，可选" />
                  <input v-model="port.container" placeholder="容器端口，例如 80" />
                  <select v-model="port.protocol">
                    <option v-for="protocol in protocolOptions" :key="protocol" :value="protocol">{{ protocol.toUpperCase() }}</option>
                  </select>
                  <button type="button" aria-label="移除端口" @click="removePortRow(index)"><X :size="13" /></button>
                </div>
              </div>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><HardDrive :size="14" /> 挂载目录</h4>
                <button type="button" @click="addMountRow"><Plus :size="12" />添加</button>
              </header>
              <div class="docker-dynamic-list">
                <div v-for="(mount, index) in createForm.mounts" :key="`mount-${index}`" class="docker-dynamic-row docker-dynamic-row--mounts">
                  <input v-model="mount.host" placeholder="宿主机路径，例如 /srv/data" />
                  <input v-model="mount.container" placeholder="容器路径，例如 /data" />
                  <select v-model="mount.mode">
                    <option value="rw">读写</option>
                    <option value="ro">只读</option>
                  </select>
                  <button type="button" aria-label="移除挂载" @click="removeMountRow(index)"><X :size="13" /></button>
                </div>
              </div>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><Activity :size="14" /> 环境变量</h4>
                <button type="button" @click="addKeyValueRow('env')"><Plus :size="12" />添加</button>
              </header>
              <div class="docker-dynamic-list">
                <div v-for="(item, index) in createForm.env" :key="`env-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
                  <input v-model="item.key" placeholder="变量名，例如 TZ" />
                  <input v-model="item.value" placeholder="变量值，例如 Asia/Shanghai" />
                  <button type="button" aria-label="移除环境变量" @click="removeKeyValueRow('env', index)"><X :size="13" /></button>
                </div>
              </div>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><ListFilter :size="14" /> 标签与解析</h4>
                <button type="button" @click="addKeyValueRow('labels')"><Plus :size="12" />添加标签</button>
              </header>
              <div class="docker-dynamic-list">
                <div v-for="(item, index) in createForm.labels" :key="`label-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
                  <input v-model="item.key" placeholder="标签名" />
                  <input v-model="item.value" placeholder="标签值" />
                  <button type="button" aria-label="移除标签" @click="removeKeyValueRow('labels', index)"><X :size="13" /></button>
                </div>
                <div v-for="(item, index) in createForm.extraHosts" :key="`host-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
                  <input v-model="item.key" placeholder="主机名，例如 db.local" />
                  <input v-model="item.value" placeholder="IP，例如 192.168.1.10" />
                  <button type="button" aria-label="移除主机解析" @click="removeKeyValueRow('extraHosts', index)"><X :size="13" /></button>
                </div>
                <button class="docker-inline-add" type="button" @click="addKeyValueRow('extraHosts')"><Plus :size="12" />添加主机解析</button>
              </div>
            </section>

            <section class="docker-wizard-section">
              <header>
                <h4><TerminalSquare :size="14" /> 命令</h4>
              </header>
              <div class="docker-form-grid">
                <label class="docker-field">
                  <span>工作目录</span>
                  <input v-model="createForm.workingDir" placeholder="可选，例如 /app" />
                </label>
                <label class="docker-field">
                  <span>入口程序</span>
                  <input v-model="createForm.entrypoint" placeholder="可选，例如 /bin/sh" />
                </label>
                <label class="docker-field docker-field--span">
                  <span>启动命令</span>
                  <textarea v-model="createForm.command" rows="2" placeholder="可选，留空使用镜像默认命令"></textarea>
                </label>
              </div>
            </section>
          </section>

          <section v-else class="docker-wizard-panel">
            <section class="docker-review-card">
              <header>
                <h4><CheckCircle2 :size="15" /> 创建摘要</h4>
                <span>确认后将拉取缺失镜像并启动容器。</span>
              </header>
              <div class="docker-summary-list">
                <div v-for="row in createSummaryRows" :key="row.label">
                  <span>{{ row.label }}</span>
                  <strong>{{ row.value }}</strong>
                </div>
              </div>
            </section>
          </section>
        </div>
        <footer class="docker-modal__footer">
          <button class="docker-button docker-button--ghost" type="button" @click="closeCreateContainerDialog">取消</button>
          <button v-if="createStep > 0" class="docker-button docker-button--ghost" type="button" @click="prevCreateStep">
            <ChevronLeft :size="13" />上一步
          </button>
          <button v-if="createStep < 2" class="docker-button docker-button--primary" type="button" :disabled="!canAdvanceCreate" @click="nextCreateStep">
            下一步<ChevronRight :size="13" />
          </button>
          <button v-else class="docker-button docker-button--primary" type="button" :disabled="!resolveLocalImageRef(createForm.image) || creating" @click="createContainer">
            <Play :size="13" />{{ creating ? '创建中' : '完成并启动' }}
          </button>
        </footer>
      </section>

      <section v-else-if="deleteContainerDialogOpen && selectedContainer" class="docker-modal docker-modal--confirm" role="dialog" aria-modal="true" aria-labelledby="docker-delete-title">
        <header class="docker-modal__header">
          <div>
            <h3 id="docker-delete-title"><Trash2 :size="15" /> 删除容器</h3>
            <p>删除后容器配置和运行状态将不可恢复。</p>
          </div>
          <button class="docker-modal__close" type="button" aria-label="关闭" @click="closeDeleteContainerDialog">
            <X :size="15" />
          </button>
        </header>
        <div class="docker-modal__body">
          <div class="docker-delete-summary">
            <strong>{{ selectedContainer.name }}</strong>
            <span>{{ selectedContainer.image }}</span>
          </div>
          <label class="docker-check-row">
            <input v-model="deleteContainerRemoveVolumes" type="checkbox" />
            <span>同时删除匿名卷</span>
          </label>
        </div>
        <footer class="docker-modal__footer">
          <button class="docker-button docker-button--ghost" type="button" @click="closeDeleteContainerDialog">取消</button>
          <button class="docker-button docker-button--danger" type="button" :disabled="runningAction === selectedContainer.id" @click="confirmRemoveSelectedContainer">
            <Trash2 :size="13" />确认删除
          </button>
        </footer>
      </section>
    </div>
  </div>
</template>

<style scoped>
.docker-window {
  --docker-text: var(--text);
  --docker-text-strong: var(--text-strong);
  --docker-text-muted: var(--text-muted);
  --docker-panel: rgba(255, 255, 255, 0.58);
  --docker-panel-strong: rgba(255, 255, 255, 0.72);
  --docker-control: rgba(255, 255, 255, 0.86);
  --docker-border: rgba(104, 139, 170, 0.18);
  --docker-soft: rgba(19, 136, 255, 0.1);
  --docker-button-bg: rgba(19, 136, 255, 0.1);
  --docker-button-border: rgba(19, 136, 255, 0.26);
  --docker-danger: #ef4444;
  --docker-danger-soft: rgba(239, 68, 68, 0.12);
  --docker-modal-overlay: rgba(12, 24, 38, 0.32);
  display: grid;
  grid-template-columns: minmax(230px, 280px) minmax(0, 1fr) minmax(240px, 300px);
  gap: 12px;
  position: relative;
  height: 100%;
  min-height: 0;
  overflow: hidden;
  color: var(--docker-text);
  font-size: 12px;
}

.docker-window.docker-window--dark,
:global(.desktop--theme-dark .docker-window),
:global([data-theme='dark'] .docker-window),
:global(.theme-dark .docker-window),
:global(body.theme-dark .docker-window) {
  --docker-text: #dce7f5;
  --docker-text-strong: #f8fbff;
  --docker-text-muted: #9eb0c5;
  --docker-panel: rgba(6, 11, 19, 0.96);
  --docker-panel-strong: rgba(8, 15, 27, 0.98);
  --docker-control: rgba(12, 20, 32, 0.98);
  --docker-border: rgba(133, 161, 194, 0.3);
  --docker-soft: rgba(21, 132, 255, 0.2);
  --docker-button-bg: rgba(17, 31, 49, 0.94);
  --docker-button-border: rgba(99, 170, 255, 0.34);
  --docker-danger-soft: rgba(239, 68, 68, 0.16);
  --docker-modal-overlay: rgba(0, 0, 0, 0.72);
}

@media (prefers-color-scheme: dark) {
  :global(.desktop--theme-auto .docker-window) {
    --docker-text: #dce7f5;
    --docker-text-strong: #f8fbff;
    --docker-text-muted: #9eb0c5;
    --docker-panel: rgba(6, 11, 19, 0.96);
    --docker-panel-strong: rgba(8, 15, 27, 0.98);
    --docker-control: rgba(12, 20, 32, 0.98);
    --docker-border: rgba(133, 161, 194, 0.3);
    --docker-soft: rgba(21, 132, 255, 0.2);
    --docker-button-bg: rgba(17, 31, 49, 0.94);
    --docker-button-border: rgba(99, 170, 255, 0.34);
    --docker-danger-soft: rgba(239, 68, 68, 0.16);
    --docker-modal-overlay: rgba(0, 0, 0, 0.72);
  }
}

.docker-window.docker-window--dark .docker-sidebar,
.docker-window.docker-window--dark .docker-panel,
.docker-window.docker-window--dark .docker-inspector,
.docker-window.docker-window--dark .docker-card,
.docker-window.docker-window--dark .docker-detail,
.docker-window.docker-window--dark .docker-modal,
.docker-window.docker-window--dark .docker-wizard-section,
.docker-window.docker-window--dark .docker-review-card,
.docker-window.docker-window--dark .docker-summary article,
.docker-window.docker-window--dark .docker-choice-grid button,
.docker-window.docker-window--dark .docker-check-row--card {
  color: var(--docker-text) !important;
  background: var(--docker-panel) !important;
  border-color: var(--docker-border) !important;
  box-shadow: none;
}

.docker-window.docker-window--dark .docker-modal {
  background: var(--docker-panel-strong) !important;
  box-shadow: 0 24px 80px rgba(0, 0, 0, 0.56);
}

.docker-window.docker-window--dark .docker-modal__header,
.docker-window.docker-window--dark .docker-modal__footer {
  background: rgba(8, 15, 27, 0.98) !important;
  border-color: var(--docker-border) !important;
}

.docker-window.docker-window--dark input,
.docker-window.docker-window--dark select,
.docker-window.docker-window--dark textarea {
  color: var(--docker-text-strong) !important;
  background: var(--docker-control) !important;
  border-color: var(--docker-border) !important;
}

.docker-window.docker-window--dark input::placeholder,
.docker-window.docker-window--dark textarea::placeholder {
  color: rgba(180, 195, 213, 0.54) !important;
}

.docker-window.docker-window--dark .docker-button,
.docker-window.docker-window--dark .docker-sidebar__header button,
.docker-window.docker-window--dark .docker-actions button,
.docker-window.docker-window--dark .docker-modal__close,
.docker-window.docker-window--dark .docker-console__bar button,
.docker-window.docker-window--dark .docker-card header button,
.docker-window.docker-window--dark .docker-command__input button,
.docker-window.docker-window--dark .docker-list-item button,
.docker-window.docker-window--dark .docker-pills button,
.docker-window.docker-window--dark .docker-detail-tabs button,
.docker-window.docker-window--dark .docker-wizard-steps button,
.docker-window.docker-window--dark .docker-inline-add,
.docker-window.docker-window--dark .docker-dynamic-row button {
  color: var(--docker-text) !important;
  background: var(--docker-button-bg) !important;
  border-color: var(--docker-button-border) !important;
}

.docker-window.docker-window--dark .docker-button--primary,
.docker-window.docker-window--dark .docker-primary {
  color: #fff !important;
  background: linear-gradient(135deg, #1f8cff, #4868ff) !important;
  border-color: transparent !important;
}

.docker-window.docker-window--dark .docker-resource-tabs button.is-active,
.docker-window.docker-window--dark .docker-stack-list button.is-active,
.docker-window.docker-window--dark .docker-pills button.is-active,
.docker-window.docker-window--dark .docker-detail-tabs button.is-active,
.docker-window.docker-window--dark .docker-wizard-steps button.is-active,
.docker-window.docker-window--dark .docker-wizard-steps button.is-done,
.docker-window.docker-window--dark .docker-choice-grid button.is-active {
  color: #78c0ff !important;
  background: var(--docker-soft) !important;
  border-color: rgba(99, 170, 255, 0.42) !important;
}

.docker-window * {
  box-sizing: border-box;
  min-width: 0;
}

.docker-sidebar,
.docker-main,
.docker-inspector,
.docker-panel,
.docker-card,
.docker-summary article,
.docker-detail {
  background: var(--docker-panel);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-md);
}

.docker-sidebar {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  min-height: 0;
  overflow: hidden;
}

.docker-inspector {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.docker-main {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 12px;
  min-height: 0;
  overflow: hidden;
  background: transparent;
  border: 0;
}

.docker-sidebar__header,
.docker-panel__header,
.docker-card > header,
.docker-detail > header,
.docker-inspector > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 12px;
  border-bottom: 1px solid var(--docker-border);
}

h3,
p {
  margin: 0;
}

h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--docker-text-strong);
  font-size: 13px;
  font-weight: 780;
}

p,
small,
.docker-inspector span {
  color: var(--docker-text-muted);
  font-size: 11px;
  line-height: 1.45;
}

button,
input,
select,
textarea {
  font: inherit;
}

button {
  color: var(--docker-text);
}

.docker-sidebar__header button,
.docker-actions button,
.docker-primary,
.docker-button,
.docker-modal__close,
.docker-console__bar button,
.docker-card header button,
.docker-command__input button,
.docker-list-item button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 10px;
  color: var(--accent);
  background: var(--docker-button-bg);
  border: 1px solid var(--docker-button-border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 720;
}

button:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.docker-button--primary {
  color: #fff;
  background: linear-gradient(135deg, #1f8cff, #4868ff);
  border-color: transparent;
}

.docker-button--ghost {
  color: var(--docker-text);
  background: var(--docker-panel-strong);
  border-color: var(--docker-border);
}

.docker-button--danger {
  color: var(--docker-danger);
  background: var(--docker-danger-soft);
  border-color: rgba(239, 68, 68, 0.28);
}

.docker-resource-tabs {
  display: grid;
  align-content: start;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--docker-border);
  overflow: auto;
}

.docker-resource-tabs button,
.docker-stack-list button {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  width: 100%;
  padding: 11px 12px;
  text-align: left;
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
}

.docker-resource-tabs button.is-active,
.docker-stack-list button.is-active {
  color: var(--accent);
  background: var(--docker-soft);
  border-color: rgba(19, 136, 255, 0.28);
}

.docker-resource-tabs b,
.docker-stack-list b {
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-sidebar__block {
  display: grid;
  gap: 8px;
  padding: 12px;
  border-bottom: 1px solid var(--docker-border);
}

.docker-sidebar__block input,
.docker-search input,
.docker-form-row input,
.docker-command__input input,
.docker-command__input select,
.docker-field input,
.docker-field select,
.docker-field textarea,
.docker-dynamic-row input,
.docker-dynamic-row select {
  width: 100%;
  height: 32px;
  padding: 0 11px;
  color: var(--docker-text-strong);
  background: var(--docker-control);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
  outline: 0;
  font-size: 11px;
}

.docker-field textarea {
  min-height: 68px;
  padding: 9px 11px;
  resize: vertical;
  line-height: 1.45;
}

.docker-sidebar__block > button {
  width: 100%;
}

.docker-stack-list {
  flex: none;
  align-content: start;
  min-height: 0;
  overflow: auto;
  border-top: 1px solid var(--docker-border);
  border-bottom: 0;
}

.docker-stack-list span {
  display: grid;
  gap: 3px;
}

.docker-stack-list strong,
.docker-row strong,
.docker-list-item strong {
  display: block;
  overflow: hidden;
  color: var(--docker-text-strong);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-summary {
  display: grid;
  grid-template-columns: repeat(5, minmax(0, 1fr));
  gap: 10px;
}

.docker-summary article {
  display: grid;
  justify-items: center;
  gap: 4px;
  padding: 14px 10px;
}

.docker-summary strong {
  color: var(--docker-text-strong);
  font-size: 16px;
}

.docker-summary span {
  color: var(--docker-text-muted);
  font-size: 11px;
}

.docker-panel {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.docker-containers {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
}

.docker-panel__header {
  flex-wrap: wrap;
}

.docker-search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1 1 260px;
  min-width: 180px;
  height: 34px;
  padding: 0 11px;
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 999px;
}

.docker-search input {
  height: 100%;
  padding: 0;
  background: transparent;
  border: 0;
}

.docker-pills {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.docker-pills button,
.docker-detail-tabs button {
  min-height: 28px;
  padding: 0 11px;
  color: var(--docker-text-muted);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 999px;
  font-size: 11px;
}

.docker-pills button.is-active,
.docker-detail-tabs button.is-active {
  color: var(--accent);
  border-color: rgba(19, 136, 255, 0.42);
  background: var(--docker-soft);
}

.docker-table {
  display: grid;
  align-content: start;
  min-height: 0;
  overflow: auto;
  padding: 12px;
}

.docker-container-workspace {
  display: grid;
  grid-template-columns: minmax(520px, 1fr) minmax(360px, 0.42fr);
  gap: 12px;
  min-height: 0;
  overflow: hidden;
  padding: 12px;
}

.docker-container-workspace:not(.has-detail) {
  grid-template-columns: 1fr;
}

.docker-container-list {
  min-height: 0;
  overflow: hidden;
}

.docker-container-list .docker-table {
  height: 100%;
  padding: 0;
}

.docker-table__head,
.docker-row {
  display: grid;
  grid-template-columns: minmax(160px, 1.5fr) 100px minmax(160px, 1.4fr) 70px 120px minmax(130px, 1fr);
  gap: 10px;
  align-items: center;
}

.docker-table__head {
  padding: 0 12px 8px;
  color: var(--docker-text-muted);
  font-size: 11px;
}

.docker-row {
  width: 100%;
  min-height: 58px;
  padding: 9px 12px;
  text-align: left;
  background: transparent;
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
}

.docker-row + .docker-row {
  margin-top: 8px;
}

.docker-row.is-active,
.docker-list-item.is-active {
  border-color: rgba(19, 136, 255, 0.62);
  background: rgba(19, 136, 255, 0.16);
}

.docker-row span {
  overflow: hidden;
  color: var(--docker-text);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-status {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  width: fit-content;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 11px;
}

.docker-status--running {
  color: #21c783;
  background: rgba(33, 199, 131, 0.13);
}

.docker-status--restarting {
  color: #f59e0b;
  background: rgba(245, 158, 11, 0.15);
}

.docker-status--stopped {
  color: var(--docker-text-muted);
  background: rgba(120, 136, 154, 0.14);
}

.docker-detail {
  margin: 0 12px 12px;
  overflow: hidden;
}

.docker-detail--side {
  display: flex;
  flex-direction: column;
  min-height: 0;
  margin: 0;
}

.docker-detail--side > header {
  align-items: flex-start;
}

.docker-detail--side .docker-actions {
  justify-content: flex-end;
}

.docker-detail--side .docker-detail-grid {
  grid-template-columns: 1fr;
}

.docker-detail--side .docker-console,
.docker-detail--side .docker-command,
.docker-detail--side .docker-kv-list {
  flex: 1;
  max-height: none;
}

.docker-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.docker-detail-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--docker-border);
}

.docker-detail-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  padding: 12px;
}

.docker-detail-grid article {
  display: grid;
  gap: 10px;
  padding: 12px;
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
}

.docker-detail-grid strong {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-detail-grid label {
  display: grid;
  gap: 7px;
  color: var(--docker-text-muted);
  font-size: 11px;
}

.docker-console,
.docker-command,
.docker-kv-list {
  display: grid;
  gap: 8px;
  max-height: 240px;
  overflow: auto;
  padding: 12px;
}

.docker-console--logs {
  grid-template-rows: auto minmax(180px, 1fr);
}

.docker-console__bar,
.docker-command__input {
  display: flex;
  align-items: center;
  gap: 8px;
}

.docker-log-editor {
  width: 100%;
  min-height: 190px;
  height: 100%;
  padding: 12px;
  color: var(--docker-text-strong);
  background: var(--docker-control);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
  outline: 0;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 11px;
  line-height: 1.6;
  white-space: pre;
  resize: none;
  overflow: auto;
}

.docker-terminal {
  grid-template-rows: auto minmax(280px, 1fr);
  overflow: hidden;
}

.docker-terminal__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
}

.docker-terminal__toolbar select {
  width: min(220px, 100%);
  height: 32px;
  padding: 0 28px 0 10px;
  color: var(--docker-text-strong);
  background: var(--docker-control);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
  outline: 0;
}

.docker-terminal__toolbar button {
  width: auto;
  height: 32px;
  padding: 0 12px;
}

.docker-terminal__screen {
  min-height: 280px;
  padding: 10px;
  overflow: auto;
  color: #dbeafe;
  background: #030712;
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, 'Liberation Mono', monospace;
  font-size: 11px;
  line-height: 1.58;
  outline: 0;
}

.docker-terminal__screen .xterm {
  height: 100%;
}

.docker-terminal__screen .xterm-viewport {
  background: transparent !important;
}

.docker-terminal__screen .xterm-screen {
  min-height: 100%;
}

.docker-kv-list span {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 9px 10px;
  color: var(--docker-text);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
  font-size: 11px;
}

.docker-split {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
  gap: 12px;
  min-height: 0;
  padding: 12px;
  overflow: hidden;
}

.docker-split--wide-left {
  grid-template-columns: minmax(0, 1.1fr) minmax(280px, 0.9fr);
}

.docker-card {
  display: flex;
  flex-direction: column;
  min-height: 0;
  overflow: hidden;
}

.docker-list {
  display: grid;
  gap: 8px;
  align-content: start;
  min-height: 0;
  overflow: auto;
  padding: 12px;
}

.docker-list--grid {
  grid-template-columns: repeat(auto-fit, minmax(260px, 1fr));
}

.docker-list-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px;
  color: var(--docker-text);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
}

.docker-list-button {
  width: 100%;
  text-align: left;
}

.docker-form-row {
  display: grid;
  grid-template-columns: repeat(4, minmax(120px, 1fr)) auto auto auto;
  align-items: end;
}

.docker-form-row label {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  color: var(--docker-text-muted);
  font-size: 11px;
}

.docker-network-detail {
  display: grid;
  gap: 10px;
  padding: 12px;
}

.docker-inspector section {
  display: grid;
  gap: 4px;
  padding: 12px;
  border-bottom: 1px solid var(--docker-border);
}

.docker-inspector strong {
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-empty {
  display: grid;
  place-items: center;
  gap: 8px;
  min-height: 190px;
  color: var(--docker-text-muted);
  text-align: center;
  border: 1px dashed var(--docker-border);
  border-radius: var(--radius-sm);
}

.docker-empty strong {
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-empty--small {
  min-height: 170px;
}

.docker-window {
  grid-template-columns: minmax(220px, 250px) minmax(0, 1fr);
}

.docker-main {
  display: flex;
  flex-direction: column;
}

.docker-inspector {
  display: none;
}

.docker-panel {
  flex: 1;
  min-height: 0;
}

.docker-resource-tabs button {
  min-height: 44px;
}

.docker-resource-tabs span {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  color: var(--docker-text);
  font-size: 12px;
  font-weight: 680;
}

.docker-resource-tabs button.is-active span {
  color: var(--accent);
}

.docker-sidebar__header {
  min-height: 50px;
}

.docker-stack-list {
  max-height: min(280px, 42vh);
}

.docker-stack-list button {
  min-height: 54px;
}

.docker-overview-page {
  gap: 16px;
  padding: 16px;
  overflow: auto;
}

.docker-overview-grid {
  display: grid;
  grid-template-columns: minmax(0, 1.05fr) minmax(320px, 0.95fr);
  gap: 16px;
}

.docker-hero-card,
.docker-service-card,
.docker-charts,
.docker-chart-grid article,
.docker-chart-wide,
.docker-create-inline {
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-md);
}

.docker-hero-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(180px, 0.7fr);
  gap: 18px;
  padding: 24px;
  min-height: 190px;
}

.docker-hero-card h2 {
  margin: 0 0 8px;
  color: var(--docker-text-strong);
  font-size: clamp(22px, 2.4vw, 30px);
  letter-spacing: 0;
}

.docker-hero-card ul {
  display: grid;
  gap: 10px;
  align-content: center;
  margin: 0;
  padding: 0;
  list-style: none;
}

.docker-hero-card li,
.docker-service-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 10px 12px;
  color: var(--docker-text-muted);
  background: rgba(19, 136, 255, 0.06);
  border: 1px solid var(--docker-border);
  border-radius: 12px;
  font-size: 12px;
}

.docker-hero-card strong,
.docker-service-row strong {
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-service-card {
  display: grid;
  align-content: start;
  gap: 12px;
  padding: 16px;
}

.docker-service-card > header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  border-bottom: 0;
  padding: 0 0 8px;
}

.docker-service-card > header h3 {
  margin: 0 0 4px;
  color: var(--docker-text-strong);
  font-size: 14px;
  line-height: 1.25;
}

.docker-service-card > header p {
  margin: 0;
  color: var(--docker-text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.docker-service-card > header button {
  display: inline-flex;
  flex: 0 0 auto;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-width: 82px;
  height: 34px;
  padding: 0 12px;
  white-space: nowrap;
}

.docker-service-row {
  margin: 0;
}

.docker-charts {
  display: grid;
  gap: 14px;
  padding: 18px;
}

.docker-chart-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 14px;
}

.docker-chart-grid article,
.docker-chart-wide {
  display: grid;
  gap: 10px;
  padding: 14px;
}

.docker-chart-grid header,
.docker-chart-wide header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 0;
  border: 0;
}

.docker-chart-grid svg,
.docker-chart-wide svg {
  width: 100%;
  height: 120px;
}

.docker-chart-wide svg {
  height: 150px;
}

.docker-chart {
  --chart-color: var(--accent);
  --chart-fill: rgba(19, 136, 255, 0.18);
  --chart-shadow: rgba(19, 136, 255, 0.16);
}

.docker-chart--cpu {
  --chart-color: #248bff;
  --chart-fill: rgba(36, 139, 255, 0.18);
  --chart-shadow: rgba(36, 139, 255, 0.18);
}

.docker-chart--memory {
  --chart-color: #22c55e;
  --chart-fill: rgba(34, 197, 94, 0.18);
  --chart-shadow: rgba(34, 197, 94, 0.16);
}

.docker-chart--network {
  --chart-color: #8b5cf6;
  --chart-fill: rgba(139, 92, 246, 0.16);
  --chart-shadow: rgba(139, 92, 246, 0.14);
}

.docker-chart__line {
  fill: none;
  stroke: var(--chart-color);
  stroke-width: 1.7;
  stroke-linecap: round;
  stroke-linejoin: round;
  filter: drop-shadow(0 5px 12px var(--chart-shadow));
  vector-effect: non-scaling-stroke;
}

.docker-chart__area {
  fill: var(--chart-fill);
  stroke: none;
}

.docker-create-inline {
  display: grid;
  grid-template-columns: 140px repeat(5, minmax(110px, 1fr)) auto;
  gap: 9px;
  align-items: center;
  margin: 12px;
  padding: 12px;
}

.docker-create-inline input {
  width: 100%;
  height: 34px;
  padding: 0 11px;
  color: var(--docker-text-strong);
  background: var(--docker-panel);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
  outline: 0;
  font-size: 11px;
}

.docker-create-inline button,
.docker-row-actions {
  display: inline-flex;
  align-items: center;
  gap: 8px;
}

.docker-glyph {
  width: 1em;
  height: 1em;
  flex: 0 0 auto;
  fill: currentColor;
}

.docker-list--images {
  gap: 12px;
  padding: 16px;
}

.docker-list--images .docker-list-item {
  min-height: 76px;
}

.docker-image-item {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) auto;
  align-items: center;
}

.docker-image-icon {
  display: grid;
  place-items: center;
  width: 42px;
  height: 42px;
  overflow: hidden;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border: 1px solid rgba(19, 136, 255, 0.2);
  border-radius: 12px;
}

.docker-image-icon img {
  display: block;
  max-width: 32px;
  max-height: 32px;
  object-fit: contain;
}

.docker-image-icon .docker-glyph {
  width: 24px;
  height: 24px;
}

.docker-glyph--default {
  color: #1687ff;
}

.docker-image-main {
  display: grid;
  gap: 4px;
}

.docker-image-main strong {
  overflow: hidden;
  color: var(--docker-text-strong);
  font-size: 12px;
  font-weight: 760;
  line-height: 1.25;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-image-title-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.docker-image-title-row strong {
  flex: 0 1 auto;
}

.docker-image-main small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.docker-status-chip {
  display: inline-flex;
  align-items: center;
  min-height: 20px;
  padding: 0 8px;
  color: #16a36d;
  background: rgba(33, 199, 131, 0.12);
  border: 1px solid rgba(33, 199, 131, 0.2);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 760;
  white-space: nowrap;
}

.docker-status-chip--danger {
  color: var(--docker-danger);
  background: var(--docker-danger-soft);
  border-color: rgba(239, 68, 68, 0.24);
}

.docker-image-progress {
  position: relative;
  width: 100%;
  height: 6px;
  overflow: hidden;
  background: rgba(120, 136, 154, 0.18);
  border-radius: 999px;
}

.docker-image-progress i {
  position: absolute;
  inset: 0 auto 0 0;
  min-width: 4px;
  background: linear-gradient(90deg, #1687ff, #22c78a);
  border-radius: inherit;
}

.docker-row-actions button {
  white-space: nowrap;
}

.docker-modal-layer {
  position: absolute;
  inset: 0;
  z-index: 20;
  display: grid;
  place-items: center;
  padding: 18px;
  background: var(--docker-modal-overlay);
  backdrop-filter: blur(8px);
}

.docker-modal {
  display: flex;
  flex-direction: column;
  width: min(520px, 100%);
  max-height: min(720px, calc(100% - 24px));
  overflow: hidden;
  color: var(--docker-text);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-md);
  box-shadow: 0 22px 60px rgba(15, 23, 42, 0.28);
}

.docker-modal--wide {
  width: min(980px, 100%);
}

.docker-modal__header,
.docker-modal__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  border-bottom: 1px solid var(--docker-border);
}

.docker-modal__footer {
  justify-content: flex-end;
  border-top: 1px solid var(--docker-border);
  border-bottom: 0;
}

.docker-modal__body {
  min-height: 0;
  overflow: auto;
  padding: 16px;
}

.docker-modal__close {
  flex: 0 0 auto;
  width: 30px;
  padding: 0;
  color: var(--docker-text-muted);
}

.docker-form-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}

.docker-form-grid--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.docker-field {
  display: grid;
  gap: 6px;
  color: var(--docker-text-muted);
  font-size: 11px;
}

.docker-field--span {
  grid-column: 1 / -1;
}

.docker-field--required span::after {
  content: ' *';
  color: var(--docker-danger);
}

.docker-wizard {
  display: grid;
  gap: 14px;
  padding: 0;
}

.docker-wizard-steps {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 8px;
  padding: 14px 16px 0;
}

.docker-wizard-steps button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 34px;
  color: var(--docker-text-muted);
  background: var(--docker-panel);
  border: 1px solid var(--docker-border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 720;
}

.docker-wizard-steps span {
  display: inline-grid;
  place-items: center;
  width: 18px;
  height: 18px;
  border-radius: 999px;
  background: rgba(120, 136, 154, 0.14);
}

.docker-wizard-steps button.is-active,
.docker-wizard-steps button.is-done {
  color: var(--accent);
  background: var(--docker-soft);
  border-color: rgba(19, 136, 255, 0.38);
}

.docker-wizard-steps button.is-active span,
.docker-wizard-steps button.is-done span {
  color: #fff;
  background: var(--accent);
}

.docker-wizard-panel {
  display: grid;
  gap: 14px;
  padding: 0 16px 16px;
}

.docker-wizard-section,
.docker-review-card {
  display: grid;
  gap: 12px;
  padding: 14px;
  background: var(--docker-panel);
  border: 1px solid var(--docker-border);
  border-radius: var(--radius-sm);
}

.docker-wizard-section > header,
.docker-review-card > header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 0;
  border: 0;
}

.docker-wizard-section h4,
.docker-review-card h4 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--docker-text-strong);
  font-size: 12px;
  font-weight: 780;
}

.docker-wizard-section > header button,
.docker-inline-add {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 26px;
  padding: 0 9px;
  color: var(--accent);
  background: var(--docker-button-bg);
  border: 1px solid var(--docker-button-border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 720;
}

.docker-choice-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
}

.docker-choice-grid button {
  display: grid;
  gap: 5px;
  min-height: 82px;
  padding: 12px;
  text-align: left;
  color: var(--docker-text);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 12px;
}

.docker-choice-grid button.is-active {
  background: var(--docker-soft);
  border-color: rgba(19, 136, 255, 0.42);
}

.docker-choice-grid strong {
  color: var(--docker-text-strong);
  font-size: 12px;
}

.docker-choice-grid span,
.docker-review-card > header span {
  color: var(--docker-text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.docker-check-row--card {
  align-self: end;
  min-height: 32px;
  margin: 0;
  padding: 0 11px;
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
}

.docker-dynamic-list {
  display: grid;
  gap: 8px;
}

.docker-dynamic-row {
  display: grid;
  gap: 8px;
  align-items: center;
}

.docker-dynamic-row--ports,
.docker-dynamic-row--mounts {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 90px 32px;
}

.docker-dynamic-row--kv {
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) 32px;
}

.docker-dynamic-row button {
  display: inline-grid;
  place-items: center;
  width: 32px;
  height: 32px;
  color: var(--docker-text-muted);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
}

.docker-summary-list {
  display: grid;
  gap: 8px;
}

.docker-summary-list div {
  display: grid;
  grid-template-columns: 110px minmax(0, 1fr);
  gap: 12px;
  align-items: start;
  padding: 9px 10px;
  color: var(--docker-text-muted);
  background: var(--docker-panel-strong);
  border: 1px solid var(--docker-border);
  border-radius: 10px;
  font-size: 11px;
}

.docker-summary-list strong {
  overflow-wrap: anywhere;
  color: var(--docker-text-strong);
  font-weight: 720;
}

.docker-delete-summary {
  display: grid;
  gap: 4px;
  padding: 12px;
  background: var(--docker-panel);
  border: 1px solid var(--docker-border);
  border-radius: 12px;
}

.docker-delete-summary strong {
  color: var(--docker-text-strong);
  font-size: 13px;
}

.docker-delete-summary span {
  color: var(--docker-text-muted);
  font-size: 11px;
  word-break: break-all;
}

.docker-check-row {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-top: 12px;
  color: var(--docker-text);
  font-size: 12px;
}

@media (max-width: 1180px) {
  .docker-window {
    grid-template-columns: minmax(200px, 250px) minmax(0, 1fr);
  }

  .docker-inspector {
    display: none;
  }

  .docker-container-workspace {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .docker-detail--side {
    min-height: 260px;
  }

  .docker-detail--side .docker-detail-grid {
    grid-template-columns: repeat(3, minmax(0, 1fr));
  }
}

@media (max-width: 860px) {
  .docker-window {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .docker-sidebar {
    min-height: auto;
  }

  .docker-resource-tabs,
  .docker-summary,
  .docker-split,
  .docker-split--wide-left,
  .docker-overview-grid,
  .docker-chart-grid,
  .docker-hero-card,
  .docker-create-inline,
  .docker-detail-grid,
  .docker-form-grid--three,
  .docker-choice-grid,
  .docker-dynamic-row--ports,
  .docker-dynamic-row--mounts,
  .docker-dynamic-row--kv {
    grid-template-columns: 1fr;
  }

  .docker-stack-list {
    max-height: 240px;
  }

  .docker-table__head {
    display: none;
  }

  .docker-container-workspace {
    padding: 10px;
  }

  .docker-row {
    grid-template-columns: 1fr;
  }

  .docker-form-row {
    grid-template-columns: 1fr;
  }

  .docker-modal-layer {
    align-items: end;
    padding: 12px;
  }

  .docker-form-grid {
    grid-template-columns: 1fr;
  }

  .docker-wizard-steps {
    grid-template-columns: 1fr;
  }

  .docker-summary-list div {
    grid-template-columns: 1fr;
  }
}
</style>
