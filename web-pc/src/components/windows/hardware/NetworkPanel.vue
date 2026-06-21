<script setup lang="ts">
import { computed } from 'vue';
import { Network } from 'lucide-vue-next';
import { UiBadge } from '../../ui';
import type { HardwareInventory, Metric } from '../../../api/types';

const props = defineProps<{
  inventory: HardwareInventory | null;
  live: Record<string, Metric>;
}>();

const interfaces = computed(() => props.inventory?.network ?? []);
const networkLive = computed(() => props.live.network);

function stateTone(state: string): 'success' | 'neutral' {
  return state === 'up' ? 'success' : 'neutral';
}
</script>

<template>
  <div>
    <div v-if="networkLive" class="hw-section-actions">
      <span class="hw-row__label">实时吞吐</span>
      <strong class="hw-metric__value" style="font-size: var(--fs-lg);">{{ networkLive.value }}<small>{{ networkLive.unit }}</small></strong>
      <span class="hw-status">{{ networkLive.detail }}</span>
    </div>

    <section class="hw-card" style="padding: 0; overflow: hidden;">
      <table v-if="interfaces.length" class="hw-table">
        <thead>
          <tr>
            <th>接口</th>
            <th>MAC 地址</th>
            <th>IPv4</th>
            <th>速率</th>
            <th>状态</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="iface in interfaces" :key="iface.name">
            <td><strong>{{ iface.name }}</strong></td>
            <td class="hw-mono">{{ iface.mac || '—' }}</td>
            <td class="hw-table__wrap">{{ iface.ipv4.length ? iface.ipv4.join(', ') : '—' }}</td>
            <td>{{ iface.speedMbps > 0 ? `${iface.speedMbps} Mbps` : '—' }}</td>
            <td><UiBadge :tone="stateTone(iface.state)" size="sm">{{ iface.state || 'unknown' }}</UiBadge></td>
          </tr>
        </tbody>
      </table>
      <div v-else class="hw-empty"><Network :size="20" /><p>未检测到物理网卡</p></div>
    </section>
  </div>
</template>
