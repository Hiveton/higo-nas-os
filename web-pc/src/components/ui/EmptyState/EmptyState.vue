<script setup lang="ts">
import type { Component } from 'vue';

withDefaults(
  defineProps<{
    icon?: Component;
    title: string;
    description?: string;
    compact?: boolean;
  }>(),
  { icon: undefined, description: undefined, compact: false },
);
</script>

<template>
  <div class="ui-empty" :class="{ 'ui-empty--compact': compact }">
    <span v-if="icon || $slots.icon" class="ui-empty__icon">
      <slot name="icon">
        <component :is="icon" :size="compact ? 22 : 30" :stroke-width="1.8" />
      </slot>
    </span>
    <p class="ui-empty__title">{{ title }}</p>
    <p v-if="description" class="ui-empty__desc">{{ description }}</p>
    <div v-if="$slots.default" class="ui-empty__action">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.ui-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  padding: var(--space-10) var(--space-6);
  text-align: center;
  color: var(--text-muted);
}

.ui-empty--compact {
  padding: var(--space-6) var(--space-4);
}

.ui-empty__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  margin-bottom: var(--space-1);
  color: var(--text-soft);
  background: var(--surface-glass);
  border-radius: var(--radius-lg);
}

.ui-empty--compact .ui-empty__icon {
  width: 40px;
  height: 40px;
}

.ui-empty__title {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
}

.ui-empty__desc {
  max-width: 320px;
  margin: 0;
  font-size: var(--fs-sm);
  line-height: var(--lh-snug);
}

.ui-empty__action {
  margin-top: var(--space-3);
}
</style>
