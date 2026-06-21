<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref } from 'vue';
import { Cpu, HardDrive, LayoutGrid, Network, Thermometer, RefreshCw } from 'lucide-vue-next';
import { UiTabs, UiBadge, UiButton, UiWindowPage } from '../ui';
import type { TabItem } from '../ui';
import { hardwareStore, loadInventory } from '../../stores/hardware';
import { monitoringStore } from '../../stores/monitoring';
import type { HardwareInventory } from '../../api/types';
import { indexMetrics } from './hardware/format';
import OverviewPanel from './hardware/OverviewPanel.vue';
import CpuMemoryPanel from './hardware/CpuMemoryPanel.vue';
import StorageSmartPanel from './hardware/StorageSmartPanel.vue';
import NetworkPanel from './hardware/NetworkPanel.vue';
import SensorsPanel from './hardware/SensorsPanel.vue';
import './hardware/hardware-window.css';

const tabs: TabItem[] = [
  { key: 'overview', label: '总览', icon: LayoutGrid },
  { key: 'compute', label: '处理器与内存', icon: Cpu },
  { key: 'storage', label: '存储设备', icon: HardDrive },
  { key: 'network', label: '网络接口', icon: Network },
  { key: 'sensors', label: '传感器', icon: Thermometer },
];

const activeTab = ref('overview');
let pollTimer: number | undefined;

// The store exposes a deeply-readonly ref; panels only read it, so cast back to
// the mutable shape for prop typing.
const inventory = computed(() => hardwareStore.inventory.value as HardwareInventory | null);
const live = computed(() => indexMetrics(monitoringStore.metrics.value));

const adapterLabel = computed(() => {
  const adapter = inventory.value?.adapter;
  if (adapter === 'linux') return '实时硬件';
  if (adapter === 'dev') return '开发桩';
  if (adapter === 'fallback') return '离线兜底';
  return adapter ?? '';
});
const adapterTone = computed<'success' | 'warning' | 'neutral'>(() => {
  const adapter = inventory.value?.adapter;
  if (adapter === 'linux') return 'success';
  if (adapter === 'dev' || adapter === 'fallback') return 'warning';
  return 'neutral';
});

async function refresh() {
  await Promise.all([loadInventory(), monitoringStore.loadMonitoringSnapshot()]);
}

onMounted(async () => {
  await refresh();
  pollTimer = window.setInterval(() => {
    void monitoringStore.loadMonitoringSnapshot();
  }, 5000);
});

onUnmounted(() => {
  if (pollTimer !== undefined) {
    window.clearInterval(pollTimer);
    pollTimer = undefined;
  }
});
</script>

<template>
  <UiWindowPage
    layout="dashboard"
    :icon="LayoutGrid"
    title="硬件中心"
    :subtitle="`${inventory?.host?.hostname || '硬件控制台'} · 清单 + 实时指标 + 硬盘健康`"
  >
    <template #actions>
      <UiBadge v-if="adapterLabel" :tone="adapterTone" size="sm">{{ adapterLabel }}</UiBadge>
      <UiButton size="sm" tone="neutral" variant="ghost" :icon-left="RefreshCw" :disabled="hardwareStore.loading.value" @click="refresh">
        刷新
      </UiButton>
    </template>

    <template #toolbar>
      <UiTabs v-model="activeTab" :tabs="tabs" variant="underline" size="sm" overflow="menu" />
    </template>

    <div class="hardware-center__panel">
      <OverviewPanel v-if="activeTab === 'overview'" :inventory="inventory" :live="live" />
      <CpuMemoryPanel v-else-if="activeTab === 'compute'" :inventory="inventory" :live="live" />
      <StorageSmartPanel v-else-if="activeTab === 'storage'" />
      <NetworkPanel v-else-if="activeTab === 'network'" :inventory="inventory" :live="live" />
      <SensorsPanel v-else-if="activeTab === 'sensors'" :inventory="inventory" :live="live" />
    </div>
  </UiWindowPage>
</template>
