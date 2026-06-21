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
  networkDownText: string;
  networkUpText: string;
  diskReadText: string;
  diskWriteText: string;
  toneClass: (tone: string) => string;
  sparklinePath: (card: OverviewCard, lane?: 'down' | 'up') => string;
  sparklineAreaPath: (card: OverviewCard, lane?: 'down' | 'up') => string;
}>();
</script>

<template>
  <section class="overview-chart-list" aria-label="资源图表">
    <article
      v-for="card in cards"
      :key="card.id"
      :class="['overview-chart-card', toneClass(card.tone), { 'overview-chart-card--dual': card.id === 'network' || card.id === 'disk' }]"
    >
      <div class="overview-chart-card__head">
        <div>
          <component :is="card.icon" :size="17" />
          <strong>{{ card.title }}</strong>
        </div>
        <p v-if="card.id === 'network'">
          <b>↓ {{ networkDownText }}</b>
          <b>↑ {{ networkUpText }}</b>
        </p>
        <p v-else-if="card.id === 'disk'">
          <b>R {{ diskReadText }}</b>
          <b>W {{ diskWriteText }}</b>
        </p>
        <p v-else>
          <b>{{ card.value }}</b>
          <span>{{ card.detail }}</span>
        </p>
      </div>
      <div class="overview-sparkline" aria-hidden="true">
        <svg viewBox="0 0 100 80" preserveAspectRatio="none">
          <!-- baseline grid so an idle/flat metric still reads as a chart, not an empty panel -->
          <line class="spark-grid" x1="0" y1="16" x2="100" y2="16" />
          <line class="spark-grid" x1="0" y1="38" x2="100" y2="38" />
          <line class="spark-grid" x1="0" y1="60" x2="100" y2="60" />
          <line class="spark-baseline" x1="0" y1="74" x2="100" y2="74" />
          <path class="spark-area spark-area--down" :d="sparklineAreaPath(card, 'down')" />
          <path class="spark-line spark-line--down" :d="sparklinePath(card, 'down')" />
          <template v-if="card.id === 'network' || card.id === 'disk'">
            <path class="spark-area spark-area--up" :d="sparklineAreaPath(card, 'up')" />
            <path class="spark-line spark-line--up" :d="sparklinePath(card, 'up')" />
          </template>
        </svg>
      </div>
    </article>
  </section>
</template>
