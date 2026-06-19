<script setup lang="ts">
import { computed } from 'vue';
import { Server, Cpu, MemoryStick, CircuitBoard, Gauge } from 'lucide-vue-next';
import type { HardwareInventory, Metric } from '../../../api/types';
import { formatBytes, orDash } from './format';

const props = defineProps<{
  inventory: HardwareInventory | null;
  live: Record<string, Metric>;
}>();

const host = computed(() => props.inventory?.host);
const board = computed(() => props.inventory?.board);
const cpu = computed(() => props.inventory?.cpu);
const memory = computed(() => props.inventory?.memory);

const gauges = computed(() => {
  const items: { key: string; label: string; value: number; unit: string; tone: string }[] = [];
  for (const key of ['cpu', 'memory', 'temperature']) {
    const metric = props.live[key];
    if (!metric) continue;
    const value = Number(metric.value);
    items.push({
      key,
      label: metric.label || key,
      value: Number.isFinite(value) ? value : 0,
      unit: metric.unit || '',
      tone: metric.tone || 'blue',
    });
  }
  return items;
});

function toneClass(tone: string) {
  if (tone === 'green') return 'hw-bar__fill--green';
  if (tone === 'orange') return 'hw-bar__fill--orange';
  if (tone === 'red') return 'hw-bar__fill--red';
  return '';
}
</script>

<template>
  <div class="hw-grid">
    <section class="hw-card">
      <h4 class="hw-card__title"><Server :size="15" /> 主机</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">主机名</span><span class="hw-row__value">{{ orDash(host?.hostname) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">操作系统</span><span class="hw-row__value">{{ orDash(host?.osName) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">内核</span><span class="hw-row__value">{{ orDash(host?.kernel) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">架构</span><span class="hw-row__value">{{ orDash(host?.arch) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">运行时长</span><span class="hw-row__value">{{ orDash(host?.uptime) }}</span></div>
      </div>
    </section>

    <section class="hw-card">
      <h4 class="hw-card__title"><CircuitBoard :size="15" /> 主板</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">厂商</span><span class="hw-row__value">{{ orDash(board?.vendor) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">型号</span><span class="hw-row__value">{{ orDash(board?.product) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">序列号</span><span class="hw-row__value hw-mono">{{ orDash(board?.serial) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">BIOS</span><span class="hw-row__value">{{ orDash(board?.bios) }}</span></div>
      </div>
    </section>

    <section class="hw-card">
      <h4 class="hw-card__title"><Cpu :size="15" /> 处理器</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">型号</span><span class="hw-row__value">{{ orDash(cpu?.model) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">物理核心</span><span class="hw-row__value">{{ orDash(cpu?.physicalCores) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">逻辑核心</span><span class="hw-row__value">{{ orDash(cpu?.logicalCores) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">频率</span><span class="hw-row__value">{{ cpu?.mhz ? `${Math.round(cpu.mhz)} MHz` : '—' }}</span></div>
      </div>
    </section>

    <section class="hw-card">
      <h4 class="hw-card__title"><MemoryStick :size="15" /> 内存</h4>
      <div class="hw-rows">
        <div class="hw-row"><span class="hw-row__label">总容量</span><span class="hw-row__value">{{ formatBytes(memory?.totalBytes) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">已用</span><span class="hw-row__value">{{ formatBytes(memory?.usedBytes) }}</span></div>
        <div class="hw-row"><span class="hw-row__label">使用率</span><span class="hw-row__value">{{ memory ? `${Math.round(memory.usedPercent)}%` : '—' }}</span></div>
      </div>
      <div class="hw-bar" style="margin-top: 10px;">
        <i class="hw-bar__fill" :style="{ width: `${Math.min(100, Math.max(0, memory?.usedPercent ?? 0))}%` }" />
      </div>
    </section>

    <section v-if="gauges.length" class="hw-card" style="grid-column: 1 / -1;">
      <h4 class="hw-card__title"><Gauge :size="15" /> 实时指标</h4>
      <div class="hw-grid">
        <div v-for="gauge in gauges" :key="gauge.key" class="hw-metric">
          <div class="hw-metric__head">
            <span class="hw-row__label">{{ gauge.label }}</span>
            <span class="hw-metric__value">{{ gauge.value }}<small>{{ gauge.unit }}</small></span>
          </div>
          <div v-if="gauge.unit === '%'" class="hw-bar">
            <i class="hw-bar__fill" :class="toneClass(gauge.tone)" :style="{ width: `${Math.min(100, Math.max(0, gauge.value))}%` }" />
          </div>
        </div>
      </div>
    </section>
  </div>
</template>
