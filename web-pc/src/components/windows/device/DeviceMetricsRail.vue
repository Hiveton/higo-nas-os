<script setup lang="ts">
import { Activity } from 'lucide-vue-next';
import type { Component } from 'vue';
import type { Metric } from '../../../api/types';

type ViewMetric = Metric & { key: string; icon: Component; tone: string };

defineProps<{
  metrics: readonly ViewMetric[];
  activeMetricKey: string;
  unresolvedAlerts: number;
}>();

const emit = defineEmits<{
  (e: 'select-metric', key: string): void;
}>();
</script>

<template>
  <section class="device-monitor__metrics" aria-label="设备核心指标">
    <header>
      <h3><Activity :size="15" /> 核心指标</h3>
      <span>{{ unresolvedAlerts }} 条活跃告警</span>
    </header>

    <button
      v-for="metric in metrics"
      :key="metric.key"
      class="device-monitor__metric"
      :class="[
        `device-monitor__metric--${metric.tone}`,
        { 'device-monitor__metric--active': metric.key === activeMetricKey },
      ]"
      type="button"
      @click="emit('select-metric', metric.key)"
    >
      <component :is="metric.icon" :size="17" />
      <span>
        <strong>{{ metric.label }}</strong>
        <small>{{ metric.detail }}</small>
      </span>
      <b>{{ metric.value }}{{ metric.unit }}</b>
    </button>
  </section>
</template>
