<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import {
  Activity,
  ChevronDown,
  Database,
  HardDrive,
  Layers3,
  Plus,
  RotateCcw,
  Trash2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AccountSummary, AccountUser, Disk, StoragePool, StorageSpace } from '../../api/types';
import StorageSpacesPanel from './storage/StorageSpacesPanel.vue';
import StorageDiskPanel from './storage/StorageDiskPanel.vue';
import StorageCachePanel from './storage/StorageCachePanel.vue';
import StorageCreateWizard from './storage/StorageCreateWizard.vue';
import { UiButton } from '../ui';
import './storage/storage-window.css';

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
      <UiButton variant="soft" tone="neutral" size="sm" :icon-left="RotateCcw" @click="loadStorageState">
        刷新
      </UiButton>
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
          <UiButton tone="primary" :icon-left="Plus" :icon-right="ChevronDown" @click="openCreateWizard">
            创建存储空间
          </UiButton>
          <UiButton
            variant="outline"
            tone="danger"
            :icon-left="Trash2"
            :loading="busyAction === 'delete-space'"
            :disabled="!selectedSpace || Boolean(busyAction)"
            @click="deleteSelectedSpace"
          >
            删除
          </UiButton>
          <UiButton
            variant="soft"
            tone="primary"
            :loading="busyAction === 'SMART 扫描'"
            :disabled="Boolean(busyAction)"
            @click="runSpaceAction('SMART 扫描')"
          >
            SMART 扫描
          </UiButton>
          <UiButton
            variant="soft"
            tone="primary"
            :loading="busyAction === '创建快照'"
            :disabled="!selectedSpace || Boolean(busyAction)"
            @click="runSpaceAction('创建快照')"
          >
            创建快照
          </UiButton>
          <UiButton
            variant="soft"
            tone="primary"
            :loading="busyAction === '阵列修复'"
            :disabled="!selectedSpace || Boolean(busyAction)"
            @click="runSpaceAction('阵列修复')"
          >
            阵列修复
          </UiButton>
        </div>

        <div class="storage-monitor__content-grid">
          <StorageSpacesPanel
            :spaces="spaces"
            :loading="loading"
            :selected-space-id="selectedSpace?.id ?? ''"
            :space-disks="spaceDisks"
            :disk-kind="diskKind"
            :disk-protocol="diskProtocol"
            :format-g-b="formatGB"
            :parse-capacity-g-b="parseCapacityGB"
            @select="selectedSpaceId = $event"
          />

          <StorageDiskPanel
            :visible-disk-info="visibleDiskInfo"
            :selected-disk-slot="selectedDisk?.slot ?? ''"
            :disk-filter="diskFilter"
            :disk-counts="diskCounts"
            :managed-mounts="managedMounts"
            :quick-mount="quickMount"
            :busy-action="busyAction"
            :disk-kind="diskKind"
            :disk-label="diskLabel"
            :disk-protocol="diskProtocol"
            :disk-health-text="diskHealthText"
            :disk-space-name="diskSpaceName"
            :partition-size-text="partitionSizeText"
            @update:disk-filter="diskFilter = $event"
            @select-disk="selectedDiskSlot = $event"
            @open-cache="activeTab = 'cache'"
            @copy-serial="copySerial"
            @add-mount="addManagedMount"
          />
        </div>
      </div>

      <StorageCachePanel
        v-else
        :physical-disks="physicalDisks"
        :selected-disk-slot="selectedDiskSlot"
        :cache-settings="cacheSettings"
        :has-selected-disk="Boolean(selectedDisk)"
        :busy-action="busyAction"
        :disk-label="diskLabel"
        @update:selected-disk-slot="selectedDiskSlot = $event"
        @save="saveCacheSettings"
        @remove="removeSelectedDisk"
      />
    </section>

    <StorageCreateWizard
      v-if="wizardOpen"
      :wizard="wizard"
      :wizard-step="wizardStep"
      :create-done="createDone"
      :created-space-name="createdSpaceName"
      :file-systems="fileSystems"
      :usable-disks="usableDisks"
      :grouped-modes="groupedModes"
      :selected-sizes="selectedSizes"
      :selected-mode="selectedMode"
      :current-file-system="currentFileSystem"
      :selected-wizard-disks="selectedWizardDisks"
      :estimated-capacity="estimatedCapacity"
      :protected-capacity="protectedCapacity"
      :unused-capacity="unusedCapacity"
      :can-advance-wizard="canAdvanceWizard"
      :busy-action="busyAction"
      :account-users="accounts.users"
      :disk-kind="diskKind"
      :disk-label="diskLabel"
      :disk-protocol="diskProtocol"
      :format-g-b="formatGB"
      :sum="sum"
      :user-initial="userInitial"
      @close="closeCreateWizard"
      @prev="prevStep"
      @next="nextStep"
      @create="createStorageSpace"
      @toggle-disk="toggleWizardDisk"
      @select-mode="selectMode"
      @toggle-user="toggleWizardUser"
    />
  </div>
</template>
