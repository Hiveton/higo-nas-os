<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Activity, Database, HardDrive, Thermometer, Waves } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { Disk, StoragePool } from '../../api/types';

const storagePools = ref<StoragePool[]>([]);
const disks = ref<Disk[]>([]);
const selectedDiskSlot = ref('');
const storageAction = ref('正在从后端同步真实磁盘数据...');
const loading = ref(false);
const busyAction = ref('');
const activePool = computed(() => storagePools.value[0] ?? null);
const selectedDisk = computed(() => disks.value.find((disk) => disk.slot === selectedDiskSlot.value) ?? disks.value[0] ?? null);
const usedPercent = computed(() => clampPercent(activePool.value?.used ?? 0));
const poolTypeLabel = computed(() => activePool.value?.type ?? '真实主机卷');
const diskCountLabel = computed(() => (disks.value.length ? `${disks.value.length} 个卷` : '等待同步'));
const canRunAction = computed(() => Boolean(activePool.value && selectedDisk.value && !busyAction.value));

async function loadStorageState() {
  loading.value = true;
  try {
    const [nextPools, nextDisks] = await Promise.all([
      apiClient.storage.getPools(),
      apiClient.storage.getDisks(),
    ]);
    storagePools.value = nextPools;
    disks.value = nextDisks;
    selectedDiskSlot.value = nextDisks.find((disk) => disk.slot === selectedDiskSlot.value)?.slot ?? nextDisks[0]?.slot ?? '';
    storageAction.value = nextPools.length || nextDisks.length
      ? `已从后端同步 ${nextPools.length} 个主机卷、${nextDisks.length} 条挂载记录。`
      : '后端已连接，但未发现可管理的主机挂载卷。';
  } catch (error) {
    storagePools.value = [];
    disks.value = [];
    selectedDiskSlot.value = '';
    storageAction.value = `磁盘接口不可用：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function runStorageAction(action: 'SMART 扫描' | '快照' | '阵列修复') {
  const disk = selectedDisk.value;
  const pool = activePool.value;
  if (!disk || !pool) {
    storageAction.value = '请先等待后端返回真实磁盘卷后再执行操作。';
    return;
  }
  busyAction.value = action;
  try {
    const payload = { targetSlot: disk.slot, targetPool: pool.id };
    const task = action === 'SMART 扫描'
      ? await apiClient.storage.startSmartScan(payload)
      : action === '快照'
        ? await apiClient.storage.createSnapshot(payload)
        : await apiClient.storage.startRepair(payload);
    storageAction.value = `${action}：${task.message ?? `卷 ${disk.slot} 已加入任务队列`}`;
  } catch (error) {
    storageAction.value = `${action}失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

function selectDisk(disk: Disk) {
  selectedDiskSlot.value = disk.slot;
}

function diskKindLabel(disk: Disk) {
  return disk.role === 'volume' || disk.interface === 'mount' ? '卷' : '槽位';
}

function clampPercent(value: number) {
  if (!Number.isFinite(value)) return 0;
  return Math.min(100, Math.max(0, value));
}

onMounted(loadStorageState);
</script>

<template>
  <div class="storage-monitor">
    <section class="storage-monitor__pool" aria-label="真实主机卷容量">
      <div class="storage-monitor__pool-head">
        <div>
          <p>{{ loading ? '同步中' : poolTypeLabel }}</p>
          <strong>{{ activePool?.name ?? '未发现主机卷' }}</strong>
        </div>
        <Database :size="24" />
      </div>
      <div class="storage-monitor__capacity">
        <div :style="{ width: `${usedPercent}%` }" />
      </div>
      <div class="storage-monitor__capacity-meta">
        <span>{{ usedPercent }}% 已用</span>
        <span>{{ activePool?.total ?? '--' }}</span>
      </div>
      <div class="storage-monitor__actions" aria-label="存储操作">
        <button type="button" :disabled="!canRunAction || busyAction === 'SMART 扫描'" @click="runStorageAction('SMART 扫描')">{{ busyAction === 'SMART 扫描' ? '扫描中' : 'SMART 扫描' }}</button>
        <button type="button" :disabled="!canRunAction || busyAction === '快照'" @click="runStorageAction('快照')">{{ busyAction === '快照' ? '创建中' : '创建快照' }}</button>
        <button type="button" :disabled="!canRunAction || busyAction === '阵列修复'" @click="runStorageAction('阵列修复')">{{ busyAction === '阵列修复' ? '修复中' : '阵列修复' }}</button>
      </div>
      <p class="storage-monitor__action-state">{{ storageAction }}</p>
    </section>

    <section class="storage-monitor__summary" aria-label="SMART 和温度概览">
      <div>
        <Activity :size="17" />
        <span>SMART</span>
        <strong>{{ activePool?.health ?? '--' }}</strong>
      </div>
      <div>
        <Thermometer :size="17" />
        <span>温度</span>
        <strong>{{ activePool?.temp ?? 'N/A' }}</strong>
      </div>
      <div>
        <Waves :size="17" />
        <span>类型</span>
        <strong>{{ poolTypeLabel }}</strong>
      </div>
    </section>

    <section class="storage-monitor__slots" aria-label="主机卷列表">
      <header>
        <h3>主机卷</h3>
        <span>{{ diskCountLabel }}</span>
      </header>
      <div class="storage-monitor__disks">
        <button
          v-for="disk in disks"
          :key="disk.slot"
          class="storage-monitor__disk"
          :class="{ 'storage-monitor__disk--active': selectedDisk?.slot === disk.slot }"
          type="button"
          @click="selectDisk(disk)"
        >
          <HardDrive :size="15" />
          <div>
            <strong>{{ diskKindLabel(disk) }} {{ disk.slot }}</strong>
            <span>{{ disk.size }} · {{ disk.state }} · {{ disk.temp }}</span>
            <small v-if="disk.serial">{{ disk.serial }}</small>
          </div>
        </button>
        <p v-if="!loading && disks.length === 0" class="storage-monitor__empty">
          暂无后端磁盘卷数据。
        </p>
      </div>
    </section>
  </div>
</template>

<style scoped>
.storage-monitor {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 10px;
  height: 100%;
  min-height: 0;
}

.storage-monitor__pool,
.storage-monitor__summary,
.storage-monitor__slots {
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.storage-monitor__pool {
  padding: 13px;
}

.storage-monitor__pool-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--accent);
}

.storage-monitor__pool-head p {
  margin: 0;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
}

.storage-monitor__pool-head strong {
  display: block;
  margin-top: 3px;
  color: var(--text-strong);
  font-size: 18px;
}

.storage-monitor__capacity {
  height: 10px;
  margin-top: 14px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.18);
  border-radius: 999px;
}

.storage-monitor__capacity div {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-cyan), var(--accent-green));
  border-radius: inherit;
}

.storage-monitor__capacity-meta {
  display: flex;
  justify-content: space-between;
  margin-top: 8px;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 650;
}

.storage-monitor__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 11px;
}

.storage-monitor__actions button {
  height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.storage-monitor__actions button:disabled {
  color: var(--text-soft);
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.12);
  border-color: rgba(148, 163, 184, 0.18);
}

.storage-monitor__action-state {
  margin: 8px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.storage-monitor__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
}

.storage-monitor__summary div {
  display: grid;
  gap: 3px;
  justify-items: center;
  padding: 10px 6px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.36);
}

.storage-monitor__summary span {
  color: var(--text-soft);
  font-size: 10px;
}

.storage-monitor__summary strong {
  color: var(--text-strong);
  font-size: 12px;
}

.storage-monitor__slots {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.storage-monitor__slots header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.storage-monitor__slots h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.storage-monitor__slots header span {
  color: var(--text-soft);
  font-size: 11px;
}

.storage-monitor__disks {
  display: grid;
  gap: 7px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.storage-monitor__disk {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 38px;
  padding: 7px 8px;
  color: var(--accent-green);
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
  text-align: left;
}

.storage-monitor__disk--active {
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.storage-monitor__disk strong,
.storage-monitor__disk span,
.storage-monitor__disk small {
  display: block;
}

.storage-monitor__disk strong {
  color: var(--text-strong);
  font-size: 11px;
}

.storage-monitor__disk span {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
}

.storage-monitor__disk small {
  margin-top: 3px;
  color: var(--text-soft);
  font-size: 9px;
  line-height: 1.3;
  overflow-wrap: anywhere;
}

.storage-monitor__empty {
  margin: 0;
  padding: 12px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.44);
  border: 1px dashed rgba(100, 136, 166, 0.18);
  border-radius: var(--radius-sm);
  font-size: 11px;
  line-height: 1.45;
}
</style>
