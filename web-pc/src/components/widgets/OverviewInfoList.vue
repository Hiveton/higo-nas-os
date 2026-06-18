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
  toneClass: (tone: string) => string;
}>();
</script>

<template>
  <section class="overview-info-list" aria-label="其他总览卡片">
    <article v-for="card in cards" :key="card.id" :class="['overview-info-row', toneClass(card.tone)]">
      <component :is="card.icon" :size="17" />
      <div>
        <strong>{{ card.title }}</strong>
        <span>{{ card.detail }}</span>
      </div>
      <b>{{ card.value }}</b>
    </article>
  </section>
</template>
