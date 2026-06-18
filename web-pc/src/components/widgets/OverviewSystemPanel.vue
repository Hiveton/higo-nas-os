<script setup lang="ts">
import type { Component } from 'vue';

type Tone = 'blue' | 'green' | 'orange' | 'red' | 'cyan';
type WidgetId = 'system' | 'cpu' | 'memory' | 'network' | 'disk' | 'storage' | 'backup' | 'docker' | 'security' | 'alerts';
type OverviewCard = {
  id: WidgetId;
  title: string;
  value: string;
  detail: string;
  tone: Tone;
  icon: Component;
  percent?: number;
};

defineProps<{
  cards: readonly OverviewCard[];
  hasVisibleContent: boolean;
  toneClass: (tone: string) => string;
  clampPercent: (value: number | undefined) => number;
}>();
</script>

<template>
  <section class="overview-system" aria-label="系统状况">
    <div>
      <span>系统状况</span>
      <strong>HiGoNAS</strong>
      <small>{{ hasVisibleContent ? '设备运行正常' : '已隐藏全部卡片' }}</small>
    </div>
    <div class="overview-rings">
      <article v-for="card in cards.slice(0, 4)" :key="card.id" :class="toneClass(card.tone)">
        <span class="ring" :style="{ '--value': `${clampPercent(card.percent ?? 0) * 3.6}deg` }">
          <component :is="card.icon" :size="14" />
        </span>
        <strong>{{ card.value }}</strong>
        <small>{{ card.title }}</small>
      </article>
    </div>
  </section>
</template>
