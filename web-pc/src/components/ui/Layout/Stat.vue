<script setup lang="ts">
import type { Component } from 'vue';
import type { UiTone } from '../tokens';

withDefaults(
  defineProps<{
    label: string;
    value: string | number;
    delta?: string;
    icon?: Component;
    tone?: UiTone;
  }>(),
  { delta: undefined, icon: undefined, tone: 'neutral' },
);
</script>

<template>
  <article class="ui-stat" :class="`ui-stat--${tone}`">
    <span v-if="icon" class="ui-stat__icon">
      <component :is="icon" :size="16" :stroke-width="2" />
    </span>
    <div class="ui-stat__body">
      <span class="ui-stat__label">{{ label }}</span>
      <strong class="ui-stat__value">{{ value }}</strong>
      <span v-if="delta" class="ui-stat__delta">{{ delta }}</span>
    </div>
  </article>
</template>

<style scoped>
.ui-stat {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
  padding: var(--space-3);
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  --stat-ink: var(--text-muted);
}

.ui-stat--primary {
  --stat-ink: var(--ink-blue);
}
.ui-stat--success {
  --stat-ink: var(--ink-green);
}
.ui-stat--warning {
  --stat-ink: var(--ink-orange);
}
.ui-stat--danger {
  --stat-ink: var(--ink-red);
}
.ui-stat--info {
  --stat-ink: var(--ink-cyan);
}

.ui-stat__icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 34px;
  height: 34px;
  color: var(--stat-ink);
  background: rgba(var(--surface-rgb), 0.7);
  border-radius: var(--radius-control);
}

.ui-stat__body {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.ui-stat__label {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-stat__value {
  color: var(--text-strong);
  font-size: var(--fs-xl);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
}

.ui-stat__delta {
  color: var(--stat-ink);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
}
</style>
