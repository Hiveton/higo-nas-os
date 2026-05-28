<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  Activity,
  CheckCircle2,
  ChevronDown,
  Copy,
  Database,
  HardDrive,
  Layers3,
  Plus,
  RotateCcw,
  Search,
  Settings2,
  ShieldCheck,
  Trash2,
  X,
  Zap,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AccountSummary, AccountUser, Disk, StoragePool, StorageSpace } from '../../api/types';

type StorageTab = 'spaces' | 'cache';
type WizardStep = 1 | 2 | 3 | 4;
type DiskFilter = 'all' | 'internal' | 'external';

type FileSystemOption = {
  key: 'btrfs' | 'zfs' | 'ext4';
  label: string;
  tag?: string;
  pros: string;
  cons: string;
  scenario: string;
};

type ModeOption = {
  key: string;
  label: string;
  group: '无数据保护' | '有数据保护';
  minDisks: number;
  evenOnly?: boolean;
  capacity: (sizes: number[]) => number;
  protection: (sizes: number[]) => number;
  performance: (count: number) => string;
  loss: (count: number) => string;
  pros: string;
  cons: string;
};

const fileSystems: FileSystemOption[] = [
  {
    key: 'btrfs',
    label: 'Btrfs',
    tag: '推荐',
    pros: '支持快照、校验、同卷复制和版本恢复。',
    cons: '大规模高负载场景下需要更谨慎的硬件和备份策略。',
    scenario: '适合家庭资料、照片、文档、备份和需要快照保护的空间。',
  },
  {
    key: 'zfs',
    label: 'ZFS',
    pros: '支持快照、压缩、校验和自愈，数据保护能力强。',
    cons: '资源占用较高，建议 8 GB 以上内存，并按容量规划内存。',
    scenario: '适合可靠性优先、多用户协作和重要资料空间。',
  },
  {
    key: 'ext4',
    label: 'ext4',
    pros: '成熟稳定、兼容性好，资源占用低。',
    cons: '不提供高级快照、压缩或数据校验。',
    scenario: '适合轻量文件服务、传统办公资料和资源有限设备。',
  },
];

const modeCatalog: ModeOption[] = [
  {
    key: 'basic',
    label: 'Basic',
    group: '无数据保护',
    minDisks: 1,
    capacity: (sizes) => sizes[0] ?? 0,
    protection: () => 0,
    performance: () => '读写 1 倍',
    loss: () => '无',
    pros: '充分利用单块硬盘的存储空间。',
    cons: '无数据保护，硬盘损坏会丢失该硬盘上的数据。',
  },
  {
    key: 'linear',
    label: 'Linear',
    group: '无数据保护',
    minDisks: 1,
    capacity: (sizes) => sum(sizes),
    protection: () => 0,
    performance: () => '读写 1 倍',
    loss: () => '无',
    pros: '可把多块硬盘串联为一个更大的空间。',
    cons: '无冗余保护，任意硬盘损坏都可能影响整体数据。',
  },
  {
    key: 'raid0',
    label: 'RAID 0',
    group: '无数据保护',
    minDisks: 2,
    capacity: (sizes) => sum(sizes),
    protection: () => 0,
    performance: (count) => `读写 ${count} 倍`,
    loss: (count) => `差（损坏 1 块丢失全部）`,
    pros: '读写速度最快，适合临时缓存、转码和低重要性高性能数据。',
    cons: '数据安全性最差，任意 1 块硬盘损坏都会丢失全部数据。',
  },
  {
    key: 'raid1',
    label: 'RAID 1',
    group: '有数据保护',
    minDisks: 2,
    capacity: (sizes) => Math.min(...sizes),
    protection: (sizes) => Math.max(0, sum(sizes) - Math.min(...sizes)),
    performance: () => '读 2 倍 / 写 1 倍',
    loss: () => '好（允许损坏 1 块）',
    pros: '镜像保护，任意一块盘损坏后仍可继续读取。',
    cons: '可用容量约等于最小硬盘容量。',
  },
  {
    key: 'raid5',
    label: 'RAID 5',
    group: '有数据保护',
    minDisks: 3,
    capacity: (sizes) => Math.min(...sizes) * Math.max(0, sizes.length - 1),
    protection: (sizes) => Math.min(...sizes),
    performance: (count) => `读 ${count - 1} 倍 / 写入需校验`,
    loss: () => '好（允许损坏 1 块）',
    pros: '兼顾容量和冗余，适合普通家庭和小团队资料。',
    cons: '重建期间性能下降，不能同时损坏两块硬盘。',
  },
  {
    key: 'raid6',
    label: 'RAID 6',
    group: '有数据保护',
    minDisks: 4,
    capacity: (sizes) => Math.min(...sizes) * Math.max(0, sizes.length - 2),
    protection: (sizes) => Math.min(...sizes) * 2,
    performance: (count) => `读 ${count - 2} 倍 / 写入双校验`,
    loss: () => '很好（允许损坏 2 块）',
    pros: '双校验保护，适合更大的硬盘组。',
    cons: '可用容量和写入性能会被双校验占用。',
  },
  {
    key: 'raid10',
    label: 'RAID 10',
    group: '有数据保护',
    minDisks: 4,
    evenOnly: true,
    capacity: (sizes) => Math.min(...sizes) * Math.floor(sizes.length / 2),
    protection: (sizes) => Math.min(...sizes) * Math.ceil(sizes.length / 2),
    performance: (count) => `读写约 ${Math.floor(count / 2)} 倍`,
    loss: () => '很好（镜像组内允许损坏）',
    pros: '兼顾性能与冗余，重建速度较快。',
    cons: '至少 4 块硬盘，可用容量约为一半。',
  },
];

const storagePools = ref<StoragePool[]>([]);
const disks = ref<Disk[]>([]);
const spaces = ref<StorageSpace[]>([]);
const accounts = ref<AccountSummary>({ users: [], groups: [], grants: [] });
const activeTab = ref<StorageTab>('spaces');
const selectedSpaceId = ref('');
const selectedDiskSlot = ref('');
const diskFilter = ref<DiskFilter>('all');
const loading = ref(false);
const busyAction = ref('');
const statusText = ref('正在从后端同步存储空间、硬盘和账号授权。');

const wizardOpen = ref(false);
const wizardStep = ref<WizardStep>(1);
const createDone = ref(false);
const createdSpaceName = ref('');
const wizard = ref({
  name: '存储空间 1',
  fileSystem: 'btrfs',
  mode: 'basic',
  selectedDiskSlots: [] as string[],
  selectedUserIds: [] as string[],
  quotaLimited: false,
  quotaGB: 100,
  scanBeforeCreate: false,
  formatDisk: true,
});

const quickMount = ref({
  name: '数据挂载',
  mountPath: '/mnt/data',
  role: 'data',
});
const cacheSettings = ref({
  standbyMinutes: 20,
  ssdCache: false,
  cacheMode: 'read',
});

const blockDisks = computed(() => disks.value.filter((disk) => disk.deviceType === 'disk' || disk.role === 'disk' || Boolean(disk.devicePath)));
const physicalDisks = computed(() => blockDisks.value.filter((disk) => !disk.systemDisk));
const visibleDiskInfo = computed(() => {
  if (diskFilter.value === 'internal') return blockDisks.value.filter((disk) => !diskIsExternal(disk));
  if (diskFilter.value === 'external') return blockDisks.value.filter((disk) => diskIsExternal(disk));
  return blockDisks.value;
});
const diskCounts = computed(() => ({
  all: blockDisks.value.length,
  internal: blockDisks.value.filter((disk) => !diskIsExternal(disk)).length,
  external: blockDisks.value.filter((disk) => diskIsExternal(disk)).length,
}));
const managedMounts = computed(() => disks.value.filter((disk) => !(disk.deviceType === 'disk' || disk.role === 'disk' || Boolean(disk.devicePath))));
const usableDisks = computed(() => physicalDisks.value.filter((disk) => disk.state !== '离线'));
const selectedDisk = computed(() => physicalDisks.value.find((disk) => disk.slot === selectedDiskSlot.value) ?? physicalDisks.value[0] ?? null);
const selectedSpace = computed(() => spaces.value.find((space) => space.id === selectedSpaceId.value) ?? spaces.value[0] ?? null);
const selectedWizardDisks = computed(() => usableDisks.value.filter((disk) => wizard.value.selectedDiskSlots.includes(disk.slot)));
const selectedSizes = computed(() => selectedWizardDisks.value.map((disk) => diskSizeGB(disk)).filter((size) => size > 0));
const selectedMode = computed(() => modeCatalog.find((mode) => mode.key === wizard.value.mode) ?? modeCatalog[0]);
const availableModes = computed(() => modeCatalog.filter((mode) => modeAvailable(mode, wizard.value.selectedDiskSlots.length)));
const groupedModes = computed(() => {
  const groups: Array<{ name: ModeOption['group']; modes: ModeOption[] }> = [];
  for (const mode of availableModes.value) {
    let group = groups.find((item) => item.name === mode.group);
    if (!group) {
      group = { name: mode.group, modes: [] };
      groups.push(group);
    }
    group.modes.push(mode);
  }
  return groups;
});
const estimatedCapacity = computed(() => roundGB(selectedMode.value.capacity(selectedSizes.value)));
const protectedCapacity = computed(() => roundGB(selectedMode.value.protection(selectedSizes.value)));
const unusedCapacity = computed(() => Math.max(0, roundGB(sum(selectedSizes.value) - estimatedCapacity.value - protectedCapacity.value)));
const totalManagedCapacity = computed(() => roundGB(spaces.value.reduce((total, space) => total + parseCapacityGB(space.total), 0)));
const usedManagedCapacity = computed(() => roundGB(spaces.value.reduce((total, space) => total + (parseCapacityGB(space.total) * (space.usedPercent || 0)) / 100, 0)));
const canAdvanceWizard = computed(() => {
  if (wizardStep.value === 1) return Boolean(wizard.value.fileSystem);
  if (wizardStep.value === 2) return wizard.value.selectedDiskSlots.length >= selectedMode.value.minDisks;
  if (wizardStep.value === 3) return wizard.value.selectedUserIds.length > 0;
  return Boolean(
    wizard.value.name.trim()
    && wizard.value.selectedDiskSlots.length > 0
    && !busyAction.value,
  );
});
const currentFileSystem = computed(() => fileSystems.find((item) => item.key === wizard.value.fileSystem) ?? fileSystems[0]);

watch(availableModes, (modes) => {
  if (!modes.some((mode) => mode.key === wizard.value.mode)) {
    wizard.value.mode = modes[0]?.key ?? 'basic';
  }
});

watch(() => wizard.value.fileSystem, (fileSystem) => {
  if (fileSystem === 'zfs') {
    wizard.value.formatDisk = true;
  }
});

watch(selectedDisk, syncCacheSettings);

async function loadStorageState() {
  loading.value = true;
  try {
    const [nextPools, nextDisks, nextSpaces, nextAccounts] = await Promise.all([
      apiClient.storage.getPools(),
      apiClient.storage.getDisks(),
      apiClient.storage.getSpaces(),
      apiClient.accounts.getSummary(),
    ]);
    storagePools.value = nextPools;
    disks.value = nextDisks;
    spaces.value = nextSpaces;
    accounts.value = nextAccounts;
    const nextPhysicalDisks = nextDisks.filter((disk) => !disk.systemDisk && (disk.deviceType === 'disk' || disk.role === 'disk' || Boolean(disk.devicePath)));
    selectedDiskSlot.value = nextPhysicalDisks.find((disk) => disk.slot === selectedDiskSlot.value)?.slot ?? nextPhysicalDisks[0]?.slot ?? '';
    selectedSpaceId.value = nextSpaces.find((space) => space.id === selectedSpaceId.value)?.id ?? nextSpaces[0]?.id ?? '';
    if (wizard.value.selectedDiskSlots.length === 0 && nextDisks[0]) {
      wizard.value.selectedDiskSlots = [nextDisks[0].slot];
    }
    if (wizard.value.selectedUserIds.length === 0 && nextAccounts.users[0]) {
      wizard.value.selectedUserIds = nextAccounts.users.map((user) => user.id);
    }
    syncCacheSettings();
    statusText.value = `已同步 ${nextSpaces.length} 个存储空间、${nextPhysicalDisks.length} 块硬盘、${nextAccounts.users.length} 个用户。`;
  } catch (error) {
    statusText.value = `存储接口不可用：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

function openCreateWizard() {
  wizardOpen.value = true;
  createDone.value = false;
  wizardStep.value = 1;
  wizard.value = {
    name: nextSpaceName(),
    fileSystem: 'btrfs',
    mode: usableDisks.value.length >= 2 ? 'raid1' : 'basic',
    selectedDiskSlots: usableDisks.value.slice(0, Math.min(2, Math.max(1, usableDisks.value.length))).map((disk) => disk.slot),
    selectedUserIds: accounts.value.users.map((user) => user.id),
    quotaLimited: false,
    quotaGB: 100,
    scanBeforeCreate: false,
    formatDisk: true,
  };
}

function closeCreateWizard() {
  wizardOpen.value = false;
  createDone.value = false;
}

function nextStep() {
  if (!canAdvanceWizard.value) return;
  wizardStep.value = Math.min(4, wizardStep.value + 1) as WizardStep;
}

function prevStep() {
  wizardStep.value = Math.max(1, wizardStep.value - 1) as WizardStep;
}

function toggleWizardDisk(slot: string) {
  const selected = new Set(wizard.value.selectedDiskSlots);
  if (selected.has(slot)) selected.delete(slot);
  else selected.add(slot);
  wizard.value.selectedDiskSlots = [...selected];
}

function selectMode(mode: ModeOption) {
  if (modeAvailable(mode, wizard.value.selectedDiskSlots.length)) {
    wizard.value.mode = mode.key;
  }
}

function toggleWizardUser(id: string) {
  const selected = new Set(wizard.value.selectedUserIds);
  if (selected.has(id)) selected.delete(id);
  else selected.add(id);
  wizard.value.selectedUserIds = [...selected];
}

async function createStorageSpace() {
  if (!canAdvanceWizard.value) return;
  busyAction.value = 'create-space';
  try {
    if (wizard.value.scanBeforeCreate) {
      for (const slot of wizard.value.selectedDiskSlots) {
        await apiClient.storage.startSmartScan({ targetSlot: slot });
      }
    }
    const space = await apiClient.storage.createSpace({
      name: wizard.value.name.trim(),
      mode: wizard.value.mode,
      fileSystem: wizard.value.fileSystem,
      diskSlots: wizard.value.selectedDiskSlots,
      confirm: true,
      formatDisk: wizard.value.formatDisk,
      actor: 'storage-manager',
    });
    await Promise.all(wizard.value.selectedUserIds.map((userId) => apiClient.accounts.grantSpace({
      subjectType: 'user',
      subjectId: userId,
      spaceId: space.id,
      access: 'manage',
      quotaBytes: wizard.value.quotaLimited ? Number(wizard.value.quotaGB) * 1024 * 1024 * 1024 : 0,
    })));
    createdSpaceName.value = space.name;
    selectedSpaceId.value = space.id;
    createDone.value = true;
    statusText.value = `已创建 ${space.name}，并同步 ${wizard.value.selectedUserIds.length} 个用户授权。`;
    await loadStorageState();
  } catch (error) {
    statusText.value = `创建存储空间失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function deleteSelectedSpace() {
  const space = selectedSpace.value;
  if (!space) {
    statusText.value = '当前没有可删除的存储空间。';
    return;
  }
  busyAction.value = 'delete-space';
  try {
    const task = await apiClient.storage.deleteSpace(space.id, { confirm: true, actor: 'storage-manager' });
    statusText.value = `${space.name} 删除任务已提交：${task.message ?? task.id}`;
    await loadStorageState();
  } catch (error) {
    statusText.value = `删除存储空间失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function addManagedMount() {
  if (!quickMount.value.mountPath.trim()) {
    statusText.value = '请输入挂载路径。';
    return;
  }
  busyAction.value = 'add-disk';
  try {
    const disk = await apiClient.storage.addDisk({
      name: quickMount.value.name.trim(),
      mountPath: quickMount.value.mountPath.trim(),
      role: quickMount.value.role,
      actor: 'storage-manager',
    });
    statusText.value = `已添加托管挂载：${disk.model ?? disk.slot}。`;
    await loadStorageState();
    selectedDiskSlot.value = disk.slot;
  } catch (error) {
    statusText.value = `添加挂载失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function removeSelectedDisk() {
  const disk = selectedDisk.value;
  if (!disk) return;
  busyAction.value = 'remove-disk';
  try {
    const task = await apiClient.storage.removeDisk(disk.slot, { confirm: true, actor: 'storage-manager' });
    statusText.value = `${diskLabel(disk)} 移除任务已提交：${task.message ?? task.id}`;
    await loadStorageState();
  } catch (error) {
    statusText.value = `移除失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function saveCacheSettings() {
  const disk = selectedDisk.value;
  if (!disk) return;
  busyAction.value = 'cache';
  try {
    const updated = await apiClient.storage.updateDiskSettings(disk.slot, {
      standbyMinutes: Number(cacheSettings.value.standbyMinutes) || 0,
      ssdCache: cacheSettings.value.ssdCache,
      cacheMode: cacheSettings.value.cacheMode,
      actor: 'storage-manager',
    });
    disks.value = disks.value.map((item) => (item.slot === updated.slot ? updated : item));
    statusText.value = `${diskLabel(updated)} 的缓存和休眠设置已保存。`;
  } catch (error) {
    statusText.value = `缓存设置失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function runSpaceAction(action: 'SMART 扫描' | '创建快照' | '阵列修复') {
  const disk = selectedDisk.value;
  const space = selectedSpace.value;
  if (!disk && !space) return;
  busyAction.value = action;
  try {
    const payload = { targetSlot: disk?.slot, targetPool: space?.id };
    const task = action === 'SMART 扫描'
      ? await apiClient.storage.startSmartScan(payload)
      : action === '创建快照'
        ? await apiClient.storage.createSnapshot(payload)
        : await apiClient.storage.startRepair(payload);
    statusText.value = `${action}已提交：${task.message ?? task.id}`;
  } catch (error) {
    statusText.value = `${action}失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

function syncCacheSettings() {
  const disk = selectedDisk.value;
  cacheSettings.value = {
    standbyMinutes: disk?.standbyMinutes ?? 20,
    ssdCache: Boolean(disk?.ssdCache),
    cacheMode: disk?.cacheMode || 'read',
  };
}

function nextSpaceName() {
  return `存储空间 ${spaces.value.length + 1}`;
}

function modeAvailable(mode: ModeOption, count: number) {
  if (count < mode.minDisks) return false;
  if (mode.evenOnly && count % 2 !== 0) return false;
  return true;
}

function diskSizeGB(disk: Disk) {
  return parseCapacityGB(disk.size);
}

function parseCapacityGB(value?: string) {
  if (!value) return 0;
  const number = Number.parseFloat(value);
  if (!Number.isFinite(number)) return 0;
  const text = value.toUpperCase();
  if (text.includes('TB')) return number * 1024;
  if (text.includes('MB')) return number / 1024;
  return number;
}

function sum(values: number[]) {
  return values.reduce((total, value) => total + value, 0);
}

function roundGB(value: number) {
  return Math.max(0, Math.round(value * 10) / 10);
}

function formatGB(value: number) {
  if (value <= 0) return '0 B';
  if (value >= 1024) return `${roundGB(value / 1024)} TB`;
  return `${roundGB(value)} GB`;
}

function formatBytesGB(bytes: number) {
  if (!bytes) return '不限';
  return formatGB(bytes / 1024 / 1024 / 1024);
}

function diskLabel(disk: Disk) {
  return disk.model || disk.devicePath || disk.serial || `硬盘 ${disk.slot}`;
}

function diskKind(disk: Disk) {
  if (disk.mediaType) return disk.mediaType;
  if (disk.rotational === false) return 'SSD';
  if (disk.rotational === true) return 'HDD';
  const text = `${disk.model ?? ''} ${disk.serial ?? ''}`.toLowerCase();
  if (text.includes('ssd') || text.includes('nvme')) return 'SSD';
  return disk.role === 'cache' ? 'SSD' : '磁盘';
}

function diskProtocol(disk: Disk) {
  if (disk.interface && disk.interface !== 'block') return disk.interface.toUpperCase();
  const text = `${disk.serial ?? ''} ${disk.model ?? ''}`.toLowerCase();
  if (text.includes('nvme')) return 'NVMe';
  if (text.includes('usb')) return 'USB';
  return disk.devicePath ? 'BLOCK' : 'MOUNT';
}

function diskIsExternal(disk: Disk) {
  const text = `${disk.interface ?? ''} ${disk.model ?? ''} ${disk.devicePath ?? ''}`.toLowerCase();
  return text.includes('usb') || text.includes('thunderbolt') || text.includes('external');
}

function diskHealthText(disk: Disk) {
  if (String(disk.mediaType ?? '').includes('虚拟')) return '不支持检测';
  return disk.health || disk.state || '不支持检测';
}

function diskSpaceName(disk: Disk) {
  return spaces.value.find((space) => space.diskSlots.includes(disk.slot))?.name ?? '';
}

function partitionSizeText(partition: NonNullable<Disk['partitions']>[number]) {
  if (partition.used && partition.total) return `${partition.used} / ${partition.total}`;
  return partition.size || '--';
}

async function copySerial(serial?: string) {
  if (!serial) return;
  await navigator.clipboard?.writeText(serial);
  statusText.value = '硬盘序列号已复制。';
}

function spaceDisks(space: StorageSpace) {
  return space.diskSlots
    .map((slot) => disks.value.find((disk) => disk.slot === slot))
    .filter((disk): disk is Disk => Boolean(disk));
}

function userInitial(user: AccountUser) {
  return (user.displayName || user.username || 'U').slice(0, 1).toUpperCase();
}

onMounted(loadStorageState);
</script>

<template>
  <div class="storage-monitor">
    <header class="storage-monitor__topbar">
      <div>
        <h2>存储管理</h2>
        <p>{{ statusText }}</p>
      </div>
      <button type="button" @click="loadStorageState">
        <RotateCcw :size="14" />
        刷新
      </button>
    </header>

    <section class="storage-monitor__stats" aria-label="存储概览">
      <article>
        <Database :size="18" />
        <span>存储空间</span>
        <strong>{{ spaces.length }} 个</strong>
      </article>
      <article>
        <HardDrive :size="18" />
        <span>硬盘</span>
        <strong>{{ blockDisks.length }} 块</strong>
      </article>
      <article>
        <Layers3 :size="18" />
        <span>总容量</span>
        <strong>{{ formatGB(totalManagedCapacity) }}</strong>
      </article>
      <article>
        <Activity :size="18" />
        <span>已用</span>
        <strong>{{ formatGB(usedManagedCapacity) }}</strong>
      </article>
    </section>

    <section class="storage-monitor__workspace">
      <div class="storage-monitor__tabs">
        <button type="button" :class="{ 'storage-monitor__tab--active': activeTab === 'spaces' }" @click="activeTab = 'spaces'">
          存储空间
        </button>
        <button type="button" :class="{ 'storage-monitor__tab--active': activeTab === 'cache' }" @click="activeTab = 'cache'">
          SSD 缓存加速
        </button>
      </div>

      <div v-if="activeTab === 'spaces'" class="storage-monitor__space-view">
        <div class="storage-monitor__toolbar">
          <button class="storage-monitor__primary" type="button" @click="openCreateWizard">
            <Plus :size="15" />
            创建存储空间
            <ChevronDown :size="14" />
          </button>
          <button type="button" :disabled="!selectedSpace || Boolean(busyAction)" @click="deleteSelectedSpace">
            <Trash2 :size="14" />
            删除
          </button>
          <button type="button" :disabled="Boolean(busyAction)" @click="runSpaceAction('SMART 扫描')">
            SMART 扫描
          </button>
          <button type="button" :disabled="!selectedSpace || Boolean(busyAction)" @click="runSpaceAction('创建快照')">
            创建快照
          </button>
          <button type="button" :disabled="!selectedSpace || Boolean(busyAction)" @click="runSpaceAction('阵列修复')">
            阵列修复
          </button>
        </div>

        <div class="storage-monitor__content-grid">
          <div class="storage-monitor__spaces">
            <article
              v-for="space in spaces"
              :key="space.id"
              class="storage-monitor__space-card"
              :class="{ 'storage-monitor__space-card--active': selectedSpace?.id === space.id }"
              @click="selectedSpaceId = space.id"
            >
              <div class="storage-monitor__space-icon">
                <Database :size="26" />
              </div>
              <div class="storage-monitor__space-main">
                <header>
                  <div>
                    <strong>{{ space.name }}</strong>
                    <span>{{ space.mode.toUpperCase() }} ｜ {{ space.fileSystem }}</span>
                  </div>
                  <b>{{ space.health }}</b>
                </header>
                <div class="storage-monitor__space-capacity">
                  <i :style="{ width: `${Math.min(100, Math.max(0, space.usedPercent || 0))}%` }" />
                </div>
                <p>
                  容量 {{ formatGB(parseCapacityGB(space.total) * (space.usedPercent || 0) / 100) }} /
                  {{ space.total }}，
                  剩余 {{ formatGB(parseCapacityGB(space.total) * (1 - (space.usedPercent || 0) / 100)) }}
                </p>
                <dl>
                  <div><dt>硬盘类型</dt><dd>{{ spaceDisks(space)[0] ? diskKind(spaceDisks(space)[0]) : '--' }}</dd></div>
                  <div><dt>接口协议</dt><dd>{{ spaceDisks(space)[0] ? diskProtocol(spaceDisks(space)[0]) : '--' }}</dd></div>
                  <div><dt>使用硬盘</dt><dd>{{ space.diskSlots.length }}</dd></div>
                </dl>
                <div class="storage-monitor__space-disks">
                  <span v-for="disk in spaceDisks(space)" :key="disk.slot">
                    <HardDrive :size="14" />
                    {{ disk.size }}
                  </span>
                </div>
              </div>
            </article>

            <div v-if="!loading && spaces.length === 0" class="storage-monitor__empty">
              <Database :size="42" />
              <strong>暂无存储空间</strong>
              <span>点击“创建存储空间”按文件系统、硬盘、模式、用户授权完成创建。</span>
            </div>
          </div>

          <aside class="storage-monitor__side">
            <section class="storage-monitor__disk-info-panel">
              <header class="storage-monitor__disk-info-head">
                <nav>
                  <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'all' }" @click="diskFilter = 'all'">
                    全部 {{ diskCounts.all }}
                  </button>
                  <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'internal' }" @click="diskFilter = 'internal'">
                    内置 {{ diskCounts.internal }}
                  </button>
                  <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'external' }" @click="diskFilter = 'external'">
                    外接 {{ diskCounts.external }}
                  </button>
                </nav>
                <button type="button" @click="activeTab = 'cache'">
                  <HardDrive :size="14" />
                  硬盘休眠
                </button>
              </header>

              <div class="storage-monitor__disk-cards">
                <article
                  v-for="(disk, index) in visibleDiskInfo"
                  :key="disk.slot"
                  class="storage-monitor__disk-card"
                  :class="{ 'storage-monitor__disk-card--active': selectedDisk?.slot === disk.slot }"
                  @click="selectedDiskSlot = disk.slot"
                >
                  <span class="storage-monitor__disk-index">{{ index + 1 }}</span>
                  <div class="storage-monitor__disk-device">
                    <HardDrive :size="28" />
                    <i v-if="disk.systemDisk">OS</i>
                  </div>
                  <div class="storage-monitor__disk-detail">
                    <header>
                      <strong>{{ disk.size }}</strong>
                      <b v-if="disk.systemDisk">系统安装</b>
                      <b v-else-if="diskSpaceName(disk)">{{ diskSpaceName(disk) }}</b>
                    </header>
                    <dl>
                      <div><dt>硬盘类型</dt><dd>{{ diskKind(disk) }}</dd></div>
                      <div><dt>型号</dt><dd>{{ diskLabel(disk) }}</dd></div>
                      <div><dt>健康状态</dt><dd>{{ diskHealthText(disk) }}</dd></div>
                      <div><dt>接口协议</dt><dd>{{ diskProtocol(disk) }}</dd></div>
                      <div class="storage-monitor__serial"><dt>序列号</dt><dd>{{ disk.serial || '-' }}<button v-if="disk.serial" type="button" @click.stop="copySerial(disk.serial)"><Copy :size="13" /></button></dd></div>
                    </dl>
                    <div v-if="disk.partitions?.length" class="storage-monitor__partitions">
                      <section v-for="partition in disk.partitions" :key="partition.path || partition.name">
                        <b>分区：{{ partition.name }}</b>
                        <span>{{ partitionSizeText(partition) }}</span>
                        <span>{{ partition.fileSystem || '文件系统未知' }}</span>
                        <span v-if="partition.system">系统安装</span>
                      </section>
                    </div>
                  </div>
                </article>
                <div v-if="!visibleDiskInfo.length" class="storage-monitor__empty storage-monitor__disk-empty">
                  <HardDrive :size="34" />
                  <strong>暂无硬盘</strong>
                  <span>刷新后仍为空时，请检查系统是否能读取块设备信息。</span>
                </div>
              </div>
            </section>
            <section>
              <h3>添加托管挂载</h3>
              <label>
                <span>名称</span>
                <input v-model="quickMount.name" type="text" />
              </label>
              <label>
                <span>路径</span>
                <input v-model="quickMount.mountPath" type="text" />
              </label>
              <button type="button" :disabled="busyAction === 'add-disk'" @click="addManagedMount">添加挂载</button>
              <div v-if="managedMounts.length" class="storage-monitor__managed-list">
                <span v-for="disk in managedMounts" :key="disk.slot">
                  {{ disk.model || disk.mountPath }} · {{ disk.size }}
                </span>
              </div>
            </section>
          </aside>
        </div>
      </div>

      <div v-else class="storage-monitor__cache-view">
        <section class="storage-monitor__cache-card">
          <header>
            <Zap :size="18" />
            <div>
              <strong>SSD 缓存加速</strong>
              <span>为指定硬盘启用只读或读写缓存，并设置休眠策略。</span>
            </div>
          </header>
          <div class="storage-monitor__cache-grid">
            <label>
              <span>选择硬盘</span>
              <select v-model="selectedDiskSlot">
                <option v-for="disk in physicalDisks" :key="disk.slot" :value="disk.slot">
                  {{ disk.slot }} · {{ diskLabel(disk) }} · {{ disk.size }}
                </option>
              </select>
            </label>
            <label>
              <span>休眠分钟</span>
              <input v-model.number="cacheSettings.standbyMinutes" min="0" type="number" />
            </label>
            <label>
              <span>SSD 缓存</span>
              <select v-model="cacheSettings.ssdCache">
                <option :value="false">关闭</option>
                <option :value="true">开启</option>
              </select>
            </label>
            <label>
              <span>缓存模式</span>
              <select v-model="cacheSettings.cacheMode">
                <option value="read">只读缓存</option>
                <option value="read-write">读写缓存</option>
              </select>
            </label>
          </div>
          <footer>
            <button type="button" :disabled="!selectedDisk || busyAction === 'cache'" @click="saveCacheSettings">保存缓存设置</button>
            <button type="button" :disabled="!selectedDisk || busyAction === 'remove-disk'" @click="removeSelectedDisk">移除托管挂载</button>
          </footer>
        </section>
      </div>
    </section>

    <div v-if="wizardOpen" class="storage-monitor__modal-backdrop">
      <section class="storage-monitor__wizard" aria-label="创建存储空间">
        <button class="storage-monitor__wizard-close" type="button" @click="closeCreateWizard">
          <X :size="22" />
        </button>

        <template v-if="!createDone">
          <header class="storage-monitor__wizard-head">
            <h3>创建存储空间</h3>
            <div>
              <span>{{ wizardStep === 1 ? '选择文件系统' : wizardStep === 2 ? '选择硬盘和存储模式' : wizardStep === 3 ? '选择用户' : '确认信息' }}</span>
              <strong>步骤 <b>{{ wizardStep }}</b> / 4</strong>
            </div>
          </header>

          <main class="storage-monitor__wizard-body">
            <section v-if="wizardStep === 1" class="storage-monitor__fs-list">
              <button
                v-for="item in fileSystems"
                :key="item.key"
                type="button"
                :class="{ 'storage-monitor__choice--active': wizard.fileSystem === item.key }"
                @click="wizard.fileSystem = item.key"
              >
                <h4>{{ item.label }} <small v-if="item.tag">{{ item.tag }}</small></h4>
                <p><b>优点：</b>{{ item.pros }}</p>
                <p><b class="storage-monitor__danger">缺点：</b>{{ item.cons }}</p>
                <p><b>适用场景：</b>{{ item.scenario }}</p>
              </button>
            </section>

            <section v-else-if="wizardStep === 2" class="storage-monitor__mode-step">
              <aside>
                <h4>选择硬盘</h4>
                <p>内置硬盘</p>
                <button
                  v-for="disk in usableDisks"
                  :key="disk.slot"
                  type="button"
                  :class="{ 'storage-monitor__disk-select--active': wizard.selectedDiskSlots.includes(disk.slot) }"
                  @click="toggleWizardDisk(disk.slot)"
                >
                  <span>{{ wizard.selectedDiskSlots.includes(disk.slot) ? '✓' : disk.slot }}</span>
                  <HardDrive :size="20" />
                  <strong>{{ disk.devicePath || disk.slot }} · {{ disk.size }}</strong>
                  <small>{{ diskKind(disk) }} ｜ {{ diskProtocol(disk) }} ｜ {{ diskLabel(disk) }}</small>
                </button>
              </aside>

              <div class="storage-monitor__mode-list">
                <h4>选择存储模式</h4>
                <p>已选择 {{ wizard.selectedDiskSlots.length }} 块硬盘。可选择以下存储模式：</p>
                <section v-for="group in groupedModes" :key="group.name">
                  <h5>{{ group.name }}</h5>
                  <button
                    v-for="mode in group.modes"
                    :key="mode.key"
                    type="button"
                    :class="{ 'storage-monitor__choice--active': wizard.mode === mode.key }"
                    @click="selectMode(mode)"
                  >
                    <h4>{{ mode.label }}</h4>
                    <dl>
                      <div><dt>可用容量</dt><dd>{{ formatGB(mode.capacity(selectedSizes)) }}</dd></div>
                      <div><dt>理论性能</dt><dd>{{ mode.performance(wizard.selectedDiskSlots.length) }}</dd></div>
                      <div><dt>数据防丢能力</dt><dd>{{ mode.loss(wizard.selectedDiskSlots.length) }}</dd></div>
                    </dl>
                    <p><b>优点：</b>{{ mode.pros }}</p>
                    <p><b class="storage-monitor__danger">缺点：</b>{{ mode.cons }}</p>
                  </button>
                </section>
              </div>
            </section>

            <section v-else-if="wizardStep === 3" class="storage-monitor__users-step">
              <div>
                <h4>选择可使用此存储空间的用户</h4>
                <label class="storage-monitor__search">
                  <Search :size="17" />
                  <input type="text" placeholder="搜索设备内的用户" />
                </label>
                <button
                  v-for="user in accounts.users"
                  :key="user.id"
                  type="button"
                  class="storage-monitor__user-row"
                  @click="toggleWizardUser(user.id)"
                >
                  <span :class="{ 'storage-monitor__checkbox--active': wizard.selectedUserIds.includes(user.id) }">
                    {{ wizard.selectedUserIds.includes(user.id) ? '✓' : '' }}
                  </span>
                  <i>{{ userInitial(user) }}</i>
                  <strong>{{ user.displayName || user.username }}</strong>
                  <small>{{ user.role === 'admin' ? '管理员' : user.role }}</small>
                </button>
              </div>
              <aside>
                <h4>设置可用容量</h4>
                <label>
                  <input v-model="wizard.quotaLimited" :value="false" type="radio" />
                  不限制
                </label>
                <label>
                  <input v-model="wizard.quotaLimited" :value="true" type="radio" />
                  限制普通用户的可用容量
                </label>
                <div class="storage-monitor__quota" :class="{ 'storage-monitor__quota--disabled': !wizard.quotaLimited }">
                  <input v-model.number="wizard.quotaGB" :disabled="!wizard.quotaLimited" min="1" type="number" />
                  <span>GB</span>
                </div>
              </aside>
            </section>

            <section v-else class="storage-monitor__confirm-step">
              <h4>已选硬盘</h4>
              <div class="storage-monitor__confirm-disks">
                <span v-for="disk in selectedWizardDisks" :key="disk.slot">
                  <HardDrive :size="17" />
                  <b>{{ diskKind(disk) }} · {{ disk.size }}</b>
                  <small>{{ diskProtocol(disk) }} · {{ disk.serial }}</small>
                </span>
              </div>
              <dl>
                <div><dt>文件系统</dt><dd>{{ currentFileSystem.label }}</dd></div>
                <div><dt>存储模式</dt><dd>{{ selectedMode.label }}</dd></div>
                <div><dt>预计可用容量</dt><dd>{{ formatGB(estimatedCapacity) }}</dd></div>
                <div><dt>存储空间描述</dt><dd><input v-model="wizard.name" maxlength="32" type="text" /></dd></div>
              </dl>
              <div class="storage-monitor__scan">
                <h4>硬盘读写检测</h4>
                <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': wizard.scanBeforeCreate }">
                  <input v-model="wizard.scanBeforeCreate" :value="true" type="radio" />
                  <span>
                    <strong>执行检测</strong>
                    <small>降低数据错误风险，耗时较久。</small>
                  </span>
                </label>
                <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': !wizard.scanBeforeCreate }">
                  <input v-model="wizard.scanBeforeCreate" :value="false" type="radio" />
                  <span>
                    <strong>跳过检测</strong>
                    <small>仅全新硬盘或已确认健康时建议跳过。</small>
                  </span>
                </label>
              </div>
              <div class="storage-monitor__format-confirm">
                <h4>格式化方式</h4>
                <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': wizard.formatDisk }">
                  <input v-model="wizard.formatDisk" :value="true" type="radio" />
                  <span>
                    <strong>{{ wizard.fileSystem === 'zfs' ? '格式化并创建 ZFS 池' : '格式化并挂载' }}</strong>
                    <small>{{ wizard.fileSystem === 'zfs' ? '清除所选硬盘签名，创建 ZFS 存储池并挂载。' : '清除所选硬盘签名，按当前文件系统重新格式化后挂载。' }}</small>
                  </span>
                </label>
                <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': !wizard.formatDisk, 'storage-monitor__scan-option--disabled': wizard.fileSystem === 'zfs' }">
                  <input v-model="wizard.formatDisk" :value="false" :disabled="wizard.fileSystem === 'zfs'" type="radio" />
                  <span>
                    <strong>不格式化，仅挂载</strong>
                    <small>{{ wizard.fileSystem === 'zfs' ? 'ZFS 创建存储空间需要新建池；已有池导入后续单独做。' : '保留硬盘数据，要求硬盘已有可识别文件系统。' }}</small>
                  </span>
                </label>
              </div>
            </section>
          </main>

          <footer class="storage-monitor__wizard-footer">
            <button type="button" :disabled="wizardStep === 1" @click="prevStep">上一步</button>
            <div class="storage-monitor__capacity-plan">
              <strong>预计容量</strong>
              <i><b :style="{ width: `${Math.min(100, (estimatedCapacity / Math.max(1, sum(selectedSizes))) * 100)}%` }" /></i>
              <span>可用容量：{{ formatGB(estimatedCapacity) }}</span>
              <span>数据保护：{{ formatGB(protectedCapacity) }}</span>
              <span>未利用：{{ formatGB(unusedCapacity) }}</span>
            </div>
            <button type="button" @click="closeCreateWizard">取消</button>
            <button class="storage-monitor__primary" type="button" :disabled="!canAdvanceWizard" @click="wizardStep === 4 ? createStorageSpace() : nextStep()">
              {{ wizardStep === 4 ? (busyAction === 'create-space' ? '创建中' : '创建') : '下一步' }}
            </button>
          </footer>
        </template>

        <template v-else>
          <div class="storage-monitor__success">
            <ShieldCheck :size="78" />
            <h3>已创建{{ createdSpaceName }}</h3>
            <p>存储空间已写入后端，用户授权和容量策略已同步。</p>
            <button class="storage-monitor__primary" type="button" @click="closeCreateWizard">完成</button>
          </div>
        </template>
      </section>
    </div>
  </div>
</template>

<style scoped>
.storage-monitor {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 12px;
  height: 100%;
  min-height: 0;
  overflow: hidden;
}

.storage-monitor__topbar,
.storage-monitor__stats,
.storage-monitor__workspace,
.storage-monitor__space-card,
.storage-monitor__side section,
.storage-monitor__cache-card {
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.storage-monitor__topbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 13px 14px;
}

.storage-monitor__topbar h2,
.storage-monitor__topbar p {
  margin: 0;
}

.storage-monitor__topbar h2 {
  color: var(--text-strong);
  font-size: 18px;
}

.storage-monitor__topbar p {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.4;
}

.storage-monitor__topbar button,
.storage-monitor__toolbar button,
.storage-monitor__side button,
.storage-monitor__cache-card button,
.storage-monitor__wizard-footer button,
.storage-monitor__primary {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 32px;
  padding: 0 12px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 760;
}

.storage-monitor__primary,
.storage-monitor__toolbar .storage-monitor__primary,
.storage-monitor__wizard-footer .storage-monitor__primary {
  color: #fff;
  background: var(--accent);
  border-color: transparent;
}

button:disabled {
  color: var(--text-soft) !important;
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.14) !important;
  border-color: rgba(148, 163, 184, 0.18) !important;
}

.storage-monitor__stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
}

.storage-monitor__stats article {
  display: grid;
  gap: 4px;
  justify-items: center;
  padding: 11px 8px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.38);
}

.storage-monitor__stats span {
  color: var(--text-soft);
  font-size: 10px;
}

.storage-monitor__stats strong {
  color: var(--text-strong);
  font-size: 14px;
}

.storage-monitor__workspace {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  min-height: 0;
  overflow: hidden;
}

.storage-monitor__tabs {
  display: flex;
  gap: 22px;
  padding: 13px 18px 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.16);
}

.storage-monitor__tabs button {
  position: relative;
  height: 38px;
  color: var(--text-muted);
  background: transparent;
  border: 0;
  font-size: 14px;
  font-weight: 760;
}

.storage-monitor__tabs .storage-monitor__tab--active {
  color: var(--text-strong);
}

.storage-monitor__tabs .storage-monitor__tab--active::after {
  position: absolute;
  right: 0;
  bottom: -1px;
  left: 0;
  height: 3px;
  content: "";
  background: var(--accent);
  border-radius: 999px;
}

.storage-monitor__space-view,
.storage-monitor__cache-view {
  min-height: 0;
  overflow: auto;
  padding: 14px 18px 18px;
}

.storage-monitor__toolbar {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 14px;
}

.storage-monitor__content-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) minmax(460px, 46%);
  gap: 12px;
}

.storage-monitor__spaces {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
}

.storage-monitor__space-card {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr);
  gap: 14px;
  padding: 18px;
  cursor: pointer;
}

.storage-monitor__space-card--active {
  border-color: rgba(19, 136, 255, 0.34);
  box-shadow: inset 4px 0 0 var(--accent);
}

.storage-monitor__space-icon {
  display: grid;
  width: 56px;
  height: 56px;
  place-items: center;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.86);
  border-radius: var(--radius-sm);
}

.storage-monitor__space-main {
  min-width: 0;
}

.storage-monitor__space-main header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 10px;
}

.storage-monitor__space-main header > div {
  display: flex;
  flex-wrap: wrap;
  align-items: baseline;
  gap: 4px 12px;
  min-width: 0;
}

.storage-monitor__space-main strong {
  flex: 0 0 auto;
  color: var(--text-strong);
  font-size: 16px;
}

.storage-monitor__space-main header span,
.storage-monitor__space-main p,
.storage-monitor__space-main dt {
  color: var(--text-muted);
  font-size: 11px;
}

.storage-monitor__space-main header span {
  display: inline-flex;
  align-items: center;
  min-width: 0;
  white-space: nowrap;
}

.storage-monitor__space-main header b {
  flex: 0 0 auto;
  height: 24px;
  padding: 0 8px;
  color: var(--accent-green);
  background: rgba(220, 252, 231, 0.78);
  border-radius: 7px;
  font-size: 12px;
  line-height: 24px;
}

.storage-monitor__space-capacity {
  height: 8px;
  margin: 12px 0 9px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.14);
  border-radius: 999px;
}

.storage-monitor__space-capacity i {
  display: block;
  height: 100%;
  background: var(--accent);
}

.storage-monitor__space-main dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  margin: 12px 0;
}

.storage-monitor__space-main dt,
.storage-monitor__space-main dd {
  margin: 0;
}

.storage-monitor__space-main dd {
  margin-top: 3px;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 780;
}

.storage-monitor__space-disks {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.storage-monitor__space-disks span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.74);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.storage-monitor__side {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
}

.storage-monitor__side section {
  display: grid;
  gap: 8px;
  padding: 12px;
}

.storage-monitor__disk-info-panel {
  gap: 14px !important;
  padding: 14px !important;
}

.storage-monitor__disk-info-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding-bottom: 10px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.18);
}

.storage-monitor__disk-info-head nav {
  display: flex;
  gap: 18px;
}

.storage-monitor__disk-info-head nav button {
  position: relative;
  min-height: 30px;
  padding: 0;
  color: var(--text-muted);
  background: transparent;
  border: 0;
  border-radius: 0;
  font-size: 13px;
}

.storage-monitor__disk-info-head nav button::after {
  position: absolute;
  right: 0;
  bottom: -11px;
  left: 0;
  height: 3px;
  content: "";
  background: transparent;
  border-radius: 999px;
}

.storage-monitor__disk-info-head nav .storage-monitor__disk-filter--active {
  color: var(--text-strong);
}

.storage-monitor__disk-info-head nav .storage-monitor__disk-filter--active::after {
  background: var(--accent);
}

.storage-monitor__disk-info-head > button {
  flex: 0 0 auto;
  background: rgba(255, 255, 255, 0.78);
  border-color: rgba(100, 136, 166, 0.22);
}

.storage-monitor__disk-cards {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.storage-monitor__disk-card {
  display: grid;
  grid-template-columns: 28px 58px minmax(0, 1fr);
  gap: 12px;
  padding: 14px;
  cursor: pointer;
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid rgba(100, 136, 166, 0.18);
  border-radius: var(--radius-md);
}

.storage-monitor__disk-card--active {
  border-color: rgba(19, 136, 255, 0.34);
  box-shadow: inset 3px 0 0 var(--accent);
}

.storage-monitor__disk-index {
  padding-top: 7px;
  color: var(--text-muted);
  font-size: 13px;
  text-align: center;
}

.storage-monitor__disk-device {
  position: relative;
  display: grid;
  width: 52px;
  height: 52px;
  place-items: center;
  color: var(--text-muted);
  background: rgba(148, 163, 184, 0.12);
  border-radius: var(--radius-sm);
}

.storage-monitor__disk-device i {
  position: absolute;
  top: 6px;
  left: 6px;
  padding: 1px 4px;
  color: #fff;
  background: var(--accent);
  border-radius: 5px;
  font-size: 9px;
  font-style: normal;
  font-weight: 800;
}

.storage-monitor__disk-detail {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.storage-monitor__disk-detail header {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.storage-monitor__disk-detail header strong {
  color: var(--text-strong);
  font-size: 15px;
}

.storage-monitor__disk-detail header b {
  padding: 3px 7px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.12);
  border-radius: 6px;
  font-size: 11px;
}

.storage-monitor__disk-detail dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px 18px;
  margin: 0;
}

.storage-monitor__disk-detail dt,
.storage-monitor__disk-detail dd {
  margin: 0;
}

.storage-monitor__disk-detail dt {
  color: var(--text-muted);
  font-size: 11px;
}

.storage-monitor__disk-detail dd {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  margin-top: 3px;
  overflow: hidden;
  color: var(--text-strong);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.storage-monitor__serial {
  grid-column: span 2;
}

.storage-monitor__serial button {
  flex: 0 0 auto;
  width: 22px;
  min-height: 22px;
  padding: 0;
  color: var(--text-muted);
  background: transparent;
  border: 0;
}

.storage-monitor__partitions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  padding: 10px;
  border: 1px dashed rgba(100, 136, 166, 0.24);
  border-radius: var(--radius-sm);
}

.storage-monitor__partitions section {
  display: grid;
  gap: 4px;
  min-width: 0;
  padding: 0;
  background: transparent;
  border: 0;
}

.storage-monitor__partitions b,
.storage-monitor__partitions span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.storage-monitor__partitions b {
  color: var(--text-strong);
}

.storage-monitor__partitions span {
  color: var(--text-muted);
}

.storage-monitor__disk-empty {
  min-height: 150px;
}

.storage-monitor__side h3,
.storage-monitor__cache-card strong {
  margin: 0;
  color: var(--text-strong);
  font-size: 13px;
}

.storage-monitor__disk-row {
  display: flex !important;
  justify-content: flex-start !important;
  gap: 8px !important;
  min-width: 0;
  text-align: left;
}

.storage-monitor__disk-row--active {
  border-color: rgba(19, 136, 255, 0.32) !important;
}

.storage-monitor__disk-row span,
.storage-monitor__disk-row strong,
.storage-monitor__disk-row small {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-monitor__disk-row strong {
  color: var(--text-strong);
  font-size: 12px;
}

.storage-monitor__disk-row small,
.storage-monitor__side label span,
.storage-monitor__cache-grid span {
  color: var(--text-muted);
  font-size: 10px;
}

.storage-monitor__side label,
.storage-monitor__cache-grid label {
  display: grid;
  gap: 5px;
}

.storage-monitor__managed-list {
  display: grid;
  gap: 6px;
  margin-top: 4px;
}

.storage-monitor__managed-list span {
  min-width: 0;
  padding: 7px 9px;
  overflow: hidden;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  background: rgba(148, 163, 184, 0.1);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.storage-monitor__side input,
.storage-monitor__cache-grid input,
.storage-monitor__cache-grid select,
.storage-monitor__quota input {
  width: 100%;
  min-width: 0;
  height: 32px;
  padding: 0 9px;
  color: var(--text-strong);
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 12px;
}

.storage-monitor__cache-card {
  display: grid;
  gap: 16px;
  padding: 18px;
}

.storage-monitor__cache-card header {
  display: flex;
  gap: 10px;
  color: var(--accent);
}

.storage-monitor__cache-card header span {
  display: block;
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 11px;
}

.storage-monitor__cache-grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 10px;
}

.storage-monitor__cache-card footer {
  display: flex;
  gap: 8px;
}

.storage-monitor__empty {
  display: grid;
  gap: 8px;
  min-height: 220px;
  place-items: center;
  padding: 24px;
  color: var(--text-soft);
  background: rgba(255, 255, 255, 0.46);
  border: 1px dashed rgba(100, 136, 166, 0.2);
  border-radius: var(--radius-md);
  text-align: center;
}

.storage-monitor__empty strong {
  color: var(--text-strong);
}

.storage-monitor__modal-backdrop {
  position: fixed;
  inset: 0;
  z-index: 50;
  display: grid;
  place-items: center;
  padding: 32px;
  background: rgba(15, 23, 42, 0.16);
  backdrop-filter: blur(10px);
}

.storage-monitor__wizard {
  position: relative;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  width: min(1060px, calc(100vw - 96px));
  height: min(680px, calc(100dvh - 150px));
  min-height: 0;
  padding: 22px 26px;
  background: rgba(255, 255, 255, 0.96);
  border: 1px solid rgba(100, 136, 166, 0.18);
  border-radius: 18px;
  box-shadow: 0 24px 80px rgba(15, 23, 42, 0.18);
}

.storage-monitor__wizard-close {
  position: absolute;
  top: 18px;
  right: 18px;
  display: grid;
  width: 36px;
  height: 36px;
  place-items: center;
  color: var(--text-muted);
  background: transparent;
  border: 0;
}

.storage-monitor__wizard-head {
  display: grid;
  gap: 20px;
  padding-bottom: 14px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.18);
}

.storage-monitor__wizard-head h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 18px;
}

.storage-monitor__wizard-head div {
  display: flex;
  justify-content: space-between;
  color: var(--text-strong);
  font-size: 13px;
  font-weight: 780;
}

.storage-monitor__wizard-head b {
  color: var(--accent);
}

.storage-monitor__wizard-body {
  min-height: 0;
  overflow: auto;
  padding: 14px 2px;
  scrollbar-gutter: stable;
}

.storage-monitor__fs-list {
  display: grid;
  gap: 10px;
}

.storage-monitor__fs-list button,
.storage-monitor__mode-list button {
  display: grid;
  gap: 6px;
  width: 100%;
  padding: 14px 16px;
  text-align: left;
  background: rgba(255, 255, 255, 0.82);
  border: 1px solid rgba(100, 136, 166, 0.18);
  border-radius: var(--radius-md);
}

.storage-monitor__choice--active,
.storage-monitor__fs-list .storage-monitor__choice--active,
.storage-monitor__mode-list .storage-monitor__choice--active {
  background: rgba(219, 234, 254, 0.76);
  border-color: var(--accent);
  box-shadow: inset 0 0 0 1px var(--accent);
}

.storage-monitor__fs-list h4,
.storage-monitor__mode-list h4,
.storage-monitor__users-step h4,
.storage-monitor__confirm-step h4 {
  margin: 0;
  color: var(--text-strong);
  font-size: 15px;
}

.storage-monitor__fs-list small {
  margin-left: 8px;
  padding: 3px 7px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.12);
  border-radius: 7px;
  font-size: 12px;
}

.storage-monitor__fs-list p,
.storage-monitor__mode-list p {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
  line-height: 1.45;
}

.storage-monitor__fs-list b,
.storage-monitor__mode-list b {
  color: #0f9f6e;
}

.storage-monitor__danger {
  color: #ef4444 !important;
}

.storage-monitor__mode-step {
  display: grid;
  grid-template-columns: 360px minmax(0, 1fr);
  gap: 18px;
  min-height: 0;
}

.storage-monitor__mode-step aside,
.storage-monitor__mode-list {
  display: grid;
  align-content: start;
  gap: 10px;
  min-height: 0;
  max-height: 100%;
  overflow: auto;
  padding-right: 4px;
}

.storage-monitor__mode-step aside > p,
.storage-monitor__mode-list > p,
.storage-monitor__mode-list h5 {
  margin: 0;
  color: var(--text-muted);
  font-size: 12px;
}

.storage-monitor__disk-select--active {
  background: rgba(219, 234, 254, 0.76) !important;
  border-color: var(--accent) !important;
}

.storage-monitor__mode-step aside button {
  display: grid;
  grid-template-columns: 22px 28px minmax(72px, auto) minmax(0, 1fr);
  align-items: center;
  gap: 8px;
  min-height: 58px;
  padding: 9px 10px;
  color: var(--text-muted);
  text-align: left;
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid rgba(100, 136, 166, 0.16);
  border-radius: var(--radius-md);
}

.storage-monitor__mode-step aside button > span {
  display: grid;
  width: 24px;
  height: 24px;
  place-items: center;
  color: #fff;
  background: var(--accent);
  border-radius: 50%;
  font-size: 12px;
}

.storage-monitor__mode-step aside strong,
.storage-monitor__mode-step aside small {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-monitor__mode-step aside strong {
  color: var(--text-strong);
  font-size: 13px;
}

.storage-monitor__mode-list section {
  display: grid;
  gap: 10px;
}

.storage-monitor__mode-list dl,
.storage-monitor__confirm-step dl {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 14px;
  margin: 0;
}

.storage-monitor__mode-list dt,
.storage-monitor__confirm-step dt {
  color: var(--text-muted);
  font-size: 13px;
}

.storage-monitor__mode-list dd,
.storage-monitor__confirm-step dd {
  margin: 6px 0 0;
  color: var(--text-strong);
  font-size: 14px;
  font-weight: 780;
}

.storage-monitor__users-step {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 420px;
  gap: 28px;
}

.storage-monitor__users-step > div,
.storage-monitor__users-step aside {
  display: grid;
  align-content: start;
  gap: 14px;
}

.storage-monitor__search {
  display: flex;
  align-items: center;
  gap: 8px;
  height: 44px;
  padding: 0 12px;
  color: var(--text-soft);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.storage-monitor__search input {
  flex: 1;
  min-width: 0;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
}

.storage-monitor__user-row {
  display: grid;
  grid-template-columns: 26px 48px minmax(0, auto) auto;
  align-items: center;
  gap: 12px;
  padding: 10px 0;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  text-align: left;
}

.storage-monitor__checkbox--active {
  color: #fff;
  background: var(--accent);
  border-color: var(--accent) !important;
}

.storage-monitor__user-row > span {
  display: grid;
  width: 22px;
  height: 22px;
  place-items: center;
  border: 1px solid var(--border);
  border-radius: 5px;
}

.storage-monitor__user-row i {
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  color: #fff;
  background: #94a3b8;
  border-radius: 50%;
  font-style: normal;
  font-weight: 780;
}

.storage-monitor__user-row small {
  padding: 4px 8px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.12);
  border-radius: 7px;
}

.storage-monitor__users-step aside {
  padding: 22px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.storage-monitor__users-step aside label {
  display: flex;
  align-items: center;
  gap: 10px;
  color: var(--text-strong);
  font-size: 15px;
}

.storage-monitor__quota {
  display: grid;
  grid-template-columns: minmax(0, 1fr) 80px;
  gap: 8px;
  align-items: center;
  margin-top: 12px;
}

.storage-monitor__quota span {
  display: grid;
  height: 32px;
  place-items: center;
  color: var(--text-muted);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.storage-monitor__quota--disabled {
  opacity: 0.45;
}

.storage-monitor__confirm-step {
  display: grid;
  gap: 18px;
}

.storage-monitor__confirm-disks {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 10px;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.storage-monitor__confirm-disks span {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 10px;
  background: rgba(148, 163, 184, 0.08);
  border-radius: var(--radius-sm);
}

.storage-monitor__confirm-disks b,
.storage-monitor__confirm-disks small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.storage-monitor__confirm-step dl {
  grid-template-columns: repeat(4, minmax(0, 1fr));
}

.storage-monitor__confirm-step input[type="text"] {
  width: 100%;
  height: 38px;
  padding: 0 12px;
  color: var(--text-strong);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.storage-monitor__format-confirm {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 12px;
  background: rgba(255, 247, 237, 0.68);
  border: 1px solid rgba(251, 146, 60, 0.24);
  border-radius: var(--radius-sm);
}

.storage-monitor__format-confirm h4 {
  grid-column: 1 / -1;
}

.storage-monitor__scan {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin-top: 12px;
}

.storage-monitor__scan h4 {
  grid-column: 1 / -1;
}

.storage-monitor__scan-option {
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr);
  gap: 10px;
  align-items: start;
  min-height: 78px;
  padding: 12px;
  color: var(--text-strong);
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.storage-monitor__scan-option--active {
  background: rgba(231, 247, 255, 0.86);
  border-color: rgba(19, 136, 255, 0.32);
}

.storage-monitor__scan-option--disabled {
  opacity: 0.55;
}

.storage-monitor__scan-option input {
  width: 18px;
  height: 18px;
  margin: 2px 0 0;
  accent-color: var(--accent);
}

.storage-monitor__scan-option span,
.storage-monitor__scan-option strong,
.storage-monitor__scan-option small {
  display: block;
  min-width: 0;
}

.storage-monitor__scan-option strong {
  color: var(--text-strong);
  font-size: 13px;
}

.storage-monitor__scan-option small {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.4;
}

.storage-monitor__wizard-footer {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto auto;
  gap: 14px;
  align-items: center;
  padding-top: 16px;
}

.storage-monitor__capacity-plan {
  display: grid;
  grid-template-columns: auto minmax(160px, 1fr) repeat(3, auto);
  gap: 12px;
  align-items: center;
  color: var(--text-muted);
  font-size: 13px;
}

.storage-monitor__capacity-plan strong {
  color: var(--text-strong);
}

.storage-monitor__capacity-plan i {
  display: block;
  height: 9px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.16);
  border-radius: 999px;
}

.storage-monitor__capacity-plan b {
  display: block;
  height: 100%;
  background: #0f9f6e;
}

.storage-monitor__success {
  display: grid;
  gap: 18px;
  place-items: center;
  align-self: center;
  justify-self: center;
  text-align: center;
}

.storage-monitor__success svg {
  color: var(--accent-green);
}

.storage-monitor__success h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 24px;
}

.storage-monitor__success p {
  margin: 0;
  color: var(--text-muted);
  font-size: 13px;
}

@media (max-width: 900px) {
  .storage-monitor__stats,
  .storage-monitor__cache-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .storage-monitor__content-grid,
  .storage-monitor__mode-step,
  .storage-monitor__users-step {
    grid-template-columns: 1fr;
  }

  .storage-monitor__wizard {
    width: calc(100vw - 32px);
    height: calc(100vh - 32px);
    padding: 22px;
  }

  .storage-monitor__wizard-footer,
  .storage-monitor__capacity-plan {
    grid-template-columns: 1fr;
  }
}
</style>
