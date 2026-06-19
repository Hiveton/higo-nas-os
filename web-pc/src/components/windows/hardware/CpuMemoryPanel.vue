<script setup lang="ts">
import { computed } from 'vue';
import { Cpu, MemoryStick } from 'lucide-vue-next';
import type { HardwareInventory, Metric } from '../../../api/types';
import { formatBytes, orDash } from './format';

const props = defineProps<{
  inventory: HardwareInventory | null;
  live: Record<string, Metric>;
}>();

const cpu = computed(() => props.inventory?.cpu);
const memory = computed(() => props.inventory?.memory);
const cpuLive = computed(() => props.live.cpu);
const memLive = computed(() => props.live.memory);

function pct(metric?: Metric) {
  const value = Number(metric?.value);
  return Number.isFinite(value) ? Math.min(100, Math.max(0, value)) : 0;
}

function toneClass(tone?: string) {
  if (tone === 'green') return 'hw-bar__fill--green';
  if (tone === 'orange') return 'hw-bar__fill--orange';
  if (tone === 'red') return 'hw-bar__fill--red';
  return '';
}
</script>

<template>
  <div class="hw-grid">
    <section class="hw-card">
      <h4 class="hw-card__title"><Cpu :size="15" /> 处理器</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">型号</span><span class="hw-row__value">{{ orDash(cpu?.model) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">物理核心</span><span class="hw-row__value">{{ orDash(cpu?.physicalCores) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">逻辑核心 / 线程</span><span class="hw-row__value">{{ orDash(cpu?.logicalCores) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">基准频率</span><span class="hw-row__value">{{ cpu?.mhz ? `${Math.round(cpu.mhz)} MHz` : '—' }}</span></div>
      </div>
      <div v-if="cpuLive" style="margin-top: 12px;">
        <div class="hw-metric__head">
          <span class="hw-row__label">实时占用</span>
          <span class="hw-metric__value">{{ cpuLive.value }}<small>{{ cpuLive.unit }}</small></span>
        </div>
        <div class="hw-bar" style="margin-top: 6px;">
          <i class="hw-bar__fill" :class="toneClass(cpuLive.tone)" :style="{ width: `${pct(cpuLive)}%` }" />
        </div>
        <p class="hw-metric__detail" style="margin: 6px 0 0;">{{ cpuLive.detail }}</p>
      </div>
    </section>

    <section class="hw-card">
      <h4 class="hw-card__title"><MemoryStick :size="15" /> 内存</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">总容量</span><span class="hw-row__value">{{ formatBytes(memory?.totalBytes) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">已用</span><span class="hw-row__value">{{ formatBytes(memory?.usedBytes) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">使用率</span><span class="hw-row__value">{{ memory ? `${Math.round(memory.usedPercent)}%` : '—' }}</span></div>
      </div>
      <div class="hw-bar" style="margin-top: 12px;">
        <i class="hw-bar__fill" :style="{ width: `${Math.min(100, Math.max(0, memory?.usedPercent ?? 0))}%` }" />
      </div>
      <div v-if="memLive" style="margin-top: 12px;">
        <div class="hw-metric__head">
          <span class="hw-row__label">实时占用</span>
          <span class="hw-metric__value">{{ memLive.value }}<small>{{ memLive.unit }}</small></span>
        </div>
        <p class="hw-metric__detail" style="margin: 6px 0 0;">{{ memLive.detail }}</p>
      </div>
    </section>
  </div>
</template>
