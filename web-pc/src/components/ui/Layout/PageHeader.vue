<script setup lang="ts">
import type { Component } from 'vue';
import type { UiTone } from '../tokens';
import UiBadge from '../Badge/Badge.vue';

withDefaults(
  defineProps<{
    title: string;
    subtitle?: string;
    icon?: Component;
    /** Optional inline status pill text. */
    status?: string;
    statusTone?: UiTone;
  }>(),
  { subtitle: undefined, icon: undefined, status: undefined, statusTone: 'primary' },
);
</script>

<template>
  <header class="ui-page-header">
    <div class="ui-page-header__lead">
      <span v-if="icon" class="ui-page-header__icon">
        <component :is="icon" :size="20" :stroke-width="2" />
      </span>
      <div class="ui-page-header__text">
        <div class="ui-page-header__titlerow">
          <h2 class="ui-page-header__title">{{ title }}</h2>
          <UiBadge v-if="status" :tone="statusTone" variant="soft" size="sm">{{ status }}</UiBadge>
        </div>
        <p v-if="subtitle" class="ui-page-header__subtitle">{{ subtitle }}</p>
      </div>
    </div>
    <div v-if="$slots.actions" class="ui-page-header__actions">
      <slot name="actions" />
    </div>
  </header>
</template>

<style scoped>
.ui-page-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: rgba(var(--surface-rgb), 0.62);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.ui-page-header__lead {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  min-width: 0;
}

.ui-page-header__icon {
  display: grid;
  flex: 0 0 auto;
  place-items: center;
  width: 38px;
  height: 38px;
  color: var(--accent);
  background: var(--accent-soft);
  border-radius: var(--radius-control);
}

.ui-page-header__text {
  min-width: 0;
}

.ui-page-header__titlerow {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-width: 0;
}

.ui-page-header__title {
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-lg);
  font-weight: var(--fw-bold);
  line-height: var(--lh-tight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-page-header__subtitle {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-tight);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-page-header__actions {
  display: flex;
  flex: 0 0 auto;
  flex-wrap: wrap;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
}
</style>
