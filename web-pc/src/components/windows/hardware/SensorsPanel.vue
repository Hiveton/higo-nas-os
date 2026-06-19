<script setup lang="ts">
import { computed } from 'vue';
import { Thermometer, Fan } from 'lucide-vue-next';
import type { HardwareInventory, Metric } from '../../../api/types';

const props = defineProps<{
  inventory: HardwareInventory | null;
  live: Record<string, Metric>;
}>();

const temperatures = computed(() => props.inventory?.sensors?.temperatures ?? []);
const fans = computed(() => props.inventory?.sensors?.fans ?? []);
const tempLive = computed(() => props.live.temperature);
const fanLive = computed(() => props.live.fan);
</script>

<template>
  <div class="hw-grid">
    <section class="hw-card">
      <h4 class="hw-card__title"><Thermometer :size="15" /> 温度传感器</h4>
      <div v-if="tempLive" class="hw-metric__head" style="margin-bottom: 10px;">
        <span class="hw-row__label">峰值温度</span>
        <span class="hw-metric__value">{{ tempLive.value }}<small>{{ tempLive.unit }}</small></span>
      </div>
      <div v-if="temperatures.length" class="hw-rows">
        <div v-for="(temp, index) in temperatures" :key="`${temp.label}-${index}`" class="hw-row">
          <span class="hw-row__label">{{ temp.label }}</span>
          <span class="hw-row__value">{{ temp.celsius ?? '—' }} °C</span>
        </div>
      </div>
      <div v-else class="hw-empty">未检测到温度传感器</div>
    </section>

    <section class="hw-card">
      <h4 class="hw-card__title"><Fan :size="15" /> 风扇</h4>
      <div v-if="fanLive" class="hw-metric__head" style="margin-bottom: 10px;">
        <span class="hw-row__label">平均转速</span>
        <span class="hw-metric__value">{{ fanLive.value }}<small>{{ fanLive.unit }}</small></span>
      </div>
      <div v-if="fans.length" class="hw-rows">
        <div v-for="(fan, index) in fans" :key="`${fan.label}-${index}`" class="hw-row">
          <span class="hw-row__label">{{ fan.label }}</span>
          <span class="hw-row__value">{{ fan.rpm ?? '—' }} RPM</span>
        </div>
      </div>
      <div v-else class="hw-empty">未检测到风扇</div>
    </section>
  </div>
</template>
