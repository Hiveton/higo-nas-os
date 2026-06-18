<script setup lang="ts">
import { LineChart, RefreshCw } from 'lucide-vue-next';
import type { Component } from 'vue';
import type { Metric, SystemLog } from '../../../api/types';
import { UiButton } from '../../ui';

type ViewMetric = Metric & { key: string; icon: Component; tone: string };
type ChartPoint = { x: number; y: number; value: number };

defineProps<{
  selectedMetric: ViewMetric;
  selectedMetricRingValue: number;
  activeRange: string;
  rangeKeys: string[];
  chartPoints: ChartPoint[];
  smoothLinePath: string;
  smoothAreaPath: string;
  trendMin: number;
  trendMax: number;
  systemLogs: readonly SystemLog[];
  selectedLogId: string;
}>();

const emit = defineEmits<{
  (e: 'update:range', value: string): void;
  (e: 'refresh-diagnostics'): void;
  (e: 'select-log', id: string): void;
}>();
</script>

<template>
  <main class="device-monitor__main">
    <section class="device-monitor__hero" aria-label="当前指标详情">
      <div>
        <p>{{ selectedMetric.label }} 当前负载</p>
        <strong>{{ selectedMetric.value }}{{ selectedMetric.unit }}</strong>
        <span>{{ selectedMetric.detail }}</span>
      </div>
      <div class="device-monitor__ring" :style="{ '--value': `${selectedMetricRingValue * 3.6}deg` }">
        <component :is="selectedMetric.icon" :size="26" />
      </div>
    </section>

    <section class="device-monitor__trend" aria-label="性能趋势">
      <header>
        <div>
          <h3><LineChart :size="15" /> 性能趋势</h3>
          <p>{{ selectedMetric.label }} · {{ activeRange }}</p>
        </div>
        <div class="device-monitor__range">
          <button
            v-for="range in rangeKeys"
            :key="range"
            :class="{ 'device-monitor__range-button--active': range === activeRange }"
            type="button"
            @click="emit('update:range', range)"
          >
            {{ range }}
          </button>
        </div>
      </header>
      <div class="device-monitor__chart">
        <svg viewBox="0 0 560 180" role="img" :aria-label="`${selectedMetric.label} ${activeRange} 趋势图`">
          <defs>
            <linearGradient id="deviceMonitorLine" x1="0" x2="1" y1="0" y2="0">
              <stop offset="0%" stop-color="var(--accent)" />
              <stop offset="55%" stop-color="var(--accent-cyan)" />
              <stop offset="100%" stop-color="var(--accent-green)" />
            </linearGradient>
            <linearGradient id="deviceMonitorArea" x1="0" x2="0" y1="0" y2="1">
              <stop offset="0%" stop-color="var(--accent-cyan)" stop-opacity="0.28" />
              <stop offset="100%" stop-color="var(--accent-green)" stop-opacity="0.04" />
            </linearGradient>
          </defs>
          <g class="device-monitor__chart-grid">
            <line x1="18" y1="30" x2="542" y2="30" />
            <line x1="18" y1="76" x2="542" y2="76" />
            <line x1="18" y1="122" x2="542" y2="122" />
            <line x1="18" y1="168" x2="542" y2="168" />
          </g>
          <path v-if="smoothAreaPath" class="device-monitor__chart-area" :d="smoothAreaPath" />
          <path v-if="smoothLinePath" class="device-monitor__chart-line" :d="smoothLinePath" />
          <circle
            v-for="(point, index) in chartPoints"
            :key="`${activeRange}-${index}`"
            class="device-monitor__chart-point"
            :cx="point.x"
            :cy="point.y"
            r="3.5"
          />
        </svg>
        <div class="device-monitor__chart-meta">
          <span>低 {{ trendMin }}{{ selectedMetric.unit }}</span>
          <strong>当前 {{ selectedMetric.value }}{{ selectedMetric.unit }}</strong>
          <span>高 {{ trendMax }}{{ selectedMetric.unit }}</span>
        </div>
      </div>
    </section>

    <section class="device-monitor__logs" aria-label="系统日志">
      <header>
        <h3>系统日志</h3>
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="emit('refresh-diagnostics')">刷新诊断</UiButton>
      </header>
      <div class="device-monitor__log-grid">
        <button
          v-for="log in systemLogs"
          :key="log.id"
          class="device-monitor__log"
          :class="{ 'device-monitor__log--active': log.id === selectedLogId }"
          type="button"
          @click="emit('select-log', log.id)"
        >
          <span>{{ log.at }}</span>
          <strong>{{ log.source }}</strong>
          <small>{{ log.message }}</small>
        </button>
      </div>
    </section>
  </main>
</template>
