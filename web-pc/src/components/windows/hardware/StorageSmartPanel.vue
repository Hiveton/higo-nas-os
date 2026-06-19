<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { HardDrive, RefreshCw, Activity } from 'lucide-vue-next';
import { UiButton, UiBadge } from '../../ui';
import { apiClient } from '../../../api/client';
import type { Disk } from '../../../api/types';

const disks = ref<Disk[]>([]);
const loading = ref(false);
const scanningSlot = ref<string>('');
const statusText = ref('');

const busy = computed(() => loading.value || scanningSlot.value !== '');

onMounted(loadDisks);

async function loadDisks() {
  loading.value = true;
  try {
    disks.value = await apiClient.storage.getDisks();
    statusText.value = '';
  } catch (error) {
    statusText.value = `磁盘信息加载失败：${error instanceof Error ? error.message : '未知错误'}`;
  } finally {
    loading.value = false;
  }
}

async function runSmartScan(slot: string) {
  if (busy.value) return;
  scanningSlot.value = slot || 'all';
  statusText.value = slot ? `正在对 ${slot} 进行 SMART 扫描…` : '正在对全部磁盘进行 SMART 扫描…';
  try {
    const task = await apiClient.storage.startSmartScan(slot ? { targetSlot: slot } : {});
    await pollTask(task.id);
    await loadDisks();
    statusText.value = `SMART 扫描完成${slot ? `：${slot}` : ''}。`;
  } catch (error) {
    statusText.value = `SMART 扫描失败：${error instanceof Error ? error.message : '未知错误'}`;
  } finally {
    scanningSlot.value = '';
  }
}

// Poll the inline task until it settles. The backend advances scan tasks via a
// goroutine, so a short fixed-interval poll mirrors StorageMonitorWindow's flow.
async function pollTask(id: string) {
  for (let attempt = 0; attempt < 20; attempt += 1) {
    await delay(1500);
    try {
      const task = await apiClient.storage.getTask(id);
      const state = (task.state || '').toLowerCase();
      if (state.includes('complete') || state.includes('done') || state.includes('fail') || state.includes('error')) {
        return;
      }
    } catch {
      return; // task lookup gone — treat as settled
    }
  }
}

function delay(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms));
}

function healthTone(disk: Disk): 'success' | 'warning' | 'danger' | 'neutral' {
  const health = (disk.health || disk.smart || disk.state || '').toLowerCase();
  if (health.includes('健康') || health.includes('ok') || health.includes('good') || health.includes('healthy') || health.includes('正常')) return 'success';
  if (health.includes('警告') || health.includes('warn')) return 'warning';
  if (health.includes('危') || health.includes('crit') || health.includes('fail') || health.includes('error')) return 'danger';
  return 'neutral';
}

function diskKind(disk: Disk): string {
  if (disk.mediaType) return disk.mediaType;
  if (disk.rotational === false) return 'SSD';
  if (disk.rotational === true) return 'HDD';
  return disk.interface || '—';
}
</script>

<template>
  <div>
    <div class="hw-section-actions">
      <UiButton size="sm" tone="primary" variant="soft" :icon-left="Activity" :disabled="busy" @click="runSmartScan('')">
        全盘 SMART 扫描
      </UiButton>
      <UiButton size="sm" tone="neutral" variant="ghost" :icon-left="RefreshCw" :disabled="busy" @click="loadDisks">
        刷新
      </UiButton>
      <span v-if="statusText" class="hw-status">{{ statusText }}</span>
    </div>

    <section class="hw-card" style="padding: 0; overflow: hidden;">
      <table v-if="disks.length" class="hw-table">
        <thead>
          <tr>
            <th>槽位</th>
            <th>型号</th>
            <th>类型</th>
            <th>容量</th>
            <th>温度</th>
            <th>健康</th>
            <th></th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="disk in disks" :key="disk.slot">
            <td><strong>{{ disk.slot }}</strong></td>
            <td class="hw-table__wrap">{{ disk.model || '—' }}<br v-if="disk.serial" /><span v-if="disk.serial" class="hw-mono">{{ disk.serial }}</span></td>
            <td>{{ diskKind(disk) }}</td>
            <td>{{ disk.size || '—' }}</td>
            <td>{{ disk.temp || '—' }}</td>
            <td><UiBadge :tone="healthTone(disk)" size="sm">{{ disk.health || disk.smart || disk.state || '未知' }}</UiBadge></td>
            <td>
              <UiButton size="sm" tone="neutral" variant="ghost" :disabled="busy" @click="runSmartScan(disk.slot)">
                {{ scanningSlot === disk.slot ? '扫描中…' : '扫描' }}
              </UiButton>
            </td>
          </tr>
        </tbody>
      </table>
      <div v-else class="hw-empty"><HardDrive :size="20" /><p>{{ loading ? '正在加载磁盘…' : '未检测到磁盘' }}</p></div>
    </section>
  </div>
</template>
