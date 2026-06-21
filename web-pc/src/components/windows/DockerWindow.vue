<script setup lang="ts">
import { computed, nextTick, onMounted, onUnmounted, ref, watch } from 'vue';
import { Terminal as XTerm } from '@xterm/xterm';
import '@xterm/xterm/css/xterm.css';
import {
  Activity,
  Boxes,
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
  Plus,
  RefreshCw,
  RotateCw,
  Search,
  Square,
  Trash2,
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
import DockerOverviewPanel from './docker/DockerOverviewPanel.vue';
import DockerComposePanel from './docker/DockerComposePanel.vue';
import DockerImagesPanel from './docker/DockerImagesPanel.vue';
import DockerRegistryPanel from './docker/DockerRegistryPanel.vue';
import DockerNetworksPanel from './docker/DockerNetworksPanel.vue';
import DockerVolumesPanel from './docker/DockerVolumesPanel.vue';
import CreateContainerDialog from './docker/CreateContainerDialog.vue';
import {
  UiBadge,
  UiButton,
  UiCheckbox,
  UiEmptyState,
  UiInput,
  UiModal,
  UiNavRail,
  UiSlider,
  UiTabs,
  UiWindowPage,
  useConfirm,
} from '../ui';
import type { UiTone } from '../ui';
import './docker/docker-window.css';

const confirm = useConfirm();

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
const detailTabItems = computed(() => detailTabs.map((tab) => ({ key: tab, label: tab })));
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
  if (!image) return;
  if (!(await confirm({ title: '删除镜像', message: `删除镜像 ${image}？`, tone: 'danger' }))) return;
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
  if (!name) return;
  if (!(await confirm({ title: '删除存储卷', message: `删除存储卷 ${name}？\n\n只有未被容器使用的卷才能安全删除。`, tone: 'danger' }))) return;
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
  if (!name || ['bridge', 'host', 'none'].includes(name)) return;
  if (!(await confirm({ title: '删除网络', message: `删除网络 ${name}？`, tone: 'danger' }))) return;
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
  if (!selectedNetworkName.value || !container) return;
  if (!(await confirm({ title: '断开网络', message: `从 ${selectedNetworkName.value} 断开 ${container}？`, tone: 'danger' }))) return;
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

function statusBadgeTone(status: string): UiTone {
  if (status === '运行中') return 'success';
  if (status === '重启中') return 'warning';
  return 'neutral';
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
    <UiWindowPage
      layout="master-detail"
      title="Docker"
      :subtitle="resourceTab"
      :status="actionState"
    >
      <template #actions>
        <UiButton variant="soft" size="sm" :icon-left="RefreshCw" :disabled="loading" @click="loadDockerRuntime()">刷新</UiButton>
      </template>

      <template #nav>
        <UiNavRail title="Docker">
          <template #header>
            <h3 class="docker-nav-title">
              <svg class="docker-glyph" viewBox="0 0 1024 1024" aria-hidden="true">
                <path :d="dockerIconPath" />
              </svg>
              Docker
            </h3>
          </template>

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
        </UiNavRail>
      </template>

      <DockerOverviewPanel
        v-if="resourceTab === '概览'"
        :containers="containers"
        :images="images"
        :networks="networks"
        :volumes="volumes"
        :running-count="runningCount"
        :total-cpu="totalCpu"
        :total-memory="totalMemory"
        :overview-rows="overviewRows"
        :action-state="actionState"
        :loading="loading"
        :format-metric="formatMetric"
        :cpu-path="cpuPath"
        :cpu-area-path="cpuAreaPath"
        :memory-path="memoryPath"
        :memory-area-path="memoryAreaPath"
        :network-path="networkPath"
        :network-area-path="networkAreaPath"
        @refresh="loadDockerRuntime()"
      />

      <section v-else-if="resourceTab === '容器'" class="docker-panel docker-containers">
        <header class="docker-panel__header">
          <UiInput v-model="query" class="docker-search-input" type="search" :prefix-icon="Search" placeholder="搜索容器、镜像、端口、路径" />
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
          <UiButton variant="solid" tone="primary" size="sm" :icon-left="Plus" @click="openCreateContainerDialog()">添加容器</UiButton>
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
                <UiBadge :tone="statusBadgeTone(container.status)" variant="soft" size="sm">
                  <CircleDot :size="9" />{{ container.status }}
                </UiBadge>
                <span>{{ container.image }}</span>
                <span>{{ container.cpu }}%</span>
                <span>{{ container.memoryText }}</span>
                <span>{{ container.ports[0] }}</span>
              </button>
              <UiEmptyState
                v-if="filteredContainers.length === 0"
                :icon="Container"
                title="没有容器"
                description="可以先拉取镜像，或添加一个容器。"
              />
            </div>
          </div>

          <section v-if="selectedContainer" class="docker-detail docker-detail--side">
            <header>
              <div>
                <h3>{{ selectedContainer.name }}</h3>
                <p>{{ selectedContainer.id }} · {{ selectedContainer.image }}</p>
              </div>
              <div class="docker-actions">
                <UiButton v-if="selectedContainer.status !== '运行中'" variant="soft" tone="success" size="sm" :icon-left="Play" @click="setContainerStatus('运行中')">启动</UiButton>
                <UiButton v-if="selectedContainer.status === '运行中'" variant="soft" tone="danger" size="sm" :icon-left="Square" @click="setContainerStatus('已停止')">停止</UiButton>
                <UiButton variant="soft" tone="primary" size="sm" :icon-left="RotateCw" @click="restartContainer">重启</UiButton>
                <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="openDeleteContainerDialog">删除</UiButton>
              </div>
            </header>

            <UiTabs
              variant="underline"
              size="sm"
              overflow="menu"
              :tabs="detailTabItems"
              :model-value="detailTab"
              @update:model-value="(value) => (detailTab = value as ContainerDetailTab)"
            />

            <div v-if="detailTab === '概览'" class="docker-detail-grid" data-testid="resourceLimit">
              <article>
                <strong><Gauge :size="14" /> CPU 限制</strong>
                <label>
                  <span>{{ selectedContainer.limitCpu }} 核</span>
                  <UiSlider v-model.number="selectedContainer.limitCpu" :min="1" :max="16" @change="updateLimit('cpu')" />
                </label>
              </article>
              <article>
                <strong><Activity :size="14" /> 内存限制</strong>
                <label>
                  <span>{{ selectedContainer.limitMemory }} MB</span>
                  <UiSlider v-model.number="selectedContainer.limitMemory" :min="512" :max="12288" :step="512" @change="updateLimit('memory')" />
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
                <UiButton variant="soft" size="sm" :icon-left="RefreshCw" @click="refreshSelectedContainerLogs()">刷新</UiButton>
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

      <DockerComposePanel
        v-else-if="resourceTab === 'Compose'"
        :compose-stacks="composeStacks"
        :containers="containers"
        :loading="loading"
        @refresh="loadDockerRuntime()"
        @view-containers="(name) => { selectStack(name); resourceTab = '容器'; }"
      />

      <DockerImagesPanel
        v-else-if="resourceTab === '本地镜像'"
        :local-images="localImages"
        :visible-image-pulls="visibleImagePulls"
        :docker-icon-path="dockerIconPath"
        :image-repository-path="imageRepositoryPath"
        :normalize-image-ref="normalizeImageRef"
        :should-use-external-image-icon="shouldUseExternalImageIcon"
        :image-icon-url="imageIconUrl"
        :pull-status-label="pullStatusLabel"
        :safe-progress="safeProgress"
        :image-ref="imageRef"
        @refresh="loadDockerRuntime()"
        @icon-failed="(image, iconUrl) => markImageIconFailed(image, iconUrl)"
        @run-image="(ref) => { openCreateContainerDialog(ref); resourceTab = '容器'; }"
        @remove-image="(ref) => removeImage(ref)"
      />

      <DockerRegistryPanel
        v-else-if="resourceTab === '镜像仓库'"
        v-model:image-query="imageQuery"
        :image-search-results="imageSearchResults"
        :docker-icon-path="dockerIconPath"
        :image-repository-path="imageRepositoryPath"
        :should-use-external-image-icon="shouldUseExternalImageIcon"
        :image-icon-url="imageIconUrl"
        :is-pulling-image="isPullingImage"
        :has-local-image="hasLocalImage"
        @search="searchImages"
        @icon-failed="(image, iconUrl) => markImageIconFailed(image, iconUrl)"
        @pull-image="(name) => pullImage(name)"
      />

      <DockerNetworksPanel
        v-else-if="resourceTab === '网络'"
        :networks="networks"
        :containers="containers"
        :network-form="networkForm"
        :network-attach-form="networkAttachForm"
        :selected-network-name="selectedNetworkName"
        :selected-network="selectedNetwork"
        @create="createNetwork"
        @select="(name) => selectedNetworkName = name"
        @remove="(name) => removeNetwork(name)"
        @connect="connectNetwork"
        @disconnect="(container) => disconnectNetwork(container)"
      />

      <DockerVolumesPanel
        v-else
        :volumes="volumes"
        :volume-form="volumeForm"
        :selected-volume-name="selectedVolumeName"
        @create="createVolume"
        @select="(name) => selectedVolumeName = name"
        @remove="(name) => removeVolume(name)"
      />

      <template #inspector>
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
      </template>
    </UiWindowPage>

    <CreateContainerDialog
      :open="createDialogOpen"
      :create-form="createForm"
      :create-step="createStep"
      :create-steps="createSteps"
      :can-advance-create="canAdvanceCreate"
      :creating="creating"
      :local-image-options="localImageOptions"
      :restart-policies="restartPolicies"
      :network-options="networkOptions"
      :protocol-options="protocolOptions"
      :create-summary-rows="createSummaryRows"
      :resolve-local-image-ref="resolveLocalImageRef"
      @close="closeCreateContainerDialog"
      @set-step="(index) => setCreateStep(index)"
      @prev-step="prevCreateStep"
      @next-step="nextCreateStep"
      @submit="createContainer"
      @add-port="addPortRow"
      @remove-port="(index) => removePortRow(index)"
      @add-mount="addMountRow"
      @remove-mount="(index) => removeMountRow(index)"
      @add-kv="(kind) => addKeyValueRow(kind)"
      @remove-kv="(kind, index) => removeKeyValueRow(kind, index)"
    />

    <UiModal
      :open="deleteContainerDialogOpen && !!selectedContainer"
      title="删除容器"
      size="sm"
      @close="closeDeleteContainerDialog"
    >
      <div v-if="selectedContainer" class="docker-wizard">
        <p class="docker-delete-hint">删除后容器配置和运行状态将不可恢复。</p>
        <div class="docker-delete-summary">
          <strong>{{ selectedContainer.name }}</strong>
          <span>{{ selectedContainer.image }}</span>
        </div>
        <div class="docker-check-row">
          <UiCheckbox v-model="deleteContainerRemoveVolumes" label="同时删除匿名卷" />
        </div>
      </div>
      <template #footer>
        <UiButton variant="ghost" tone="neutral" @click="closeDeleteContainerDialog">取消</UiButton>
        <UiButton
          variant="solid"
          tone="danger"
          :icon-left="Trash2"
          :disabled="!selectedContainer || runningAction === selectedContainer.id"
          @click="confirmRemoveSelectedContainer"
        >确认删除</UiButton>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.docker-search-input {
  flex: 1 1 260px;
  min-width: 180px;
}

.docker-delete-hint {
  margin: 0 0 12px;
}
</style>
