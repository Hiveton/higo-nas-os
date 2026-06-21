<script setup lang="ts">
import type { Component } from 'vue';

withDefaults(
  defineProps<{
    label: string;
    hint?: string;
    icon?: Component;
    active?: boolean;
    badge?: string | number;
  }>(),
  { hint: undefined, icon: undefined, active: false, badge: undefined },
);

const emit = defineEmits<{ select: [] }>();
</script>

<template>
  <button
    class="ui-nav-item"
    :class="{ 'ui-nav-item--active': active }"
    type="button"
    :aria-current="active ? 'page' : undefined"
    @click="emit('select')"
  >
    <component :is="icon" v-if="icon" class="ui-nav-item__icon" :size="16" :stroke-width="2" />
    <span class="ui-nav-item__text">
      <strong class="ui-nav-item__label">{{ label }}</strong>
      <small v-if="hint" class="ui-nav-item__hint">{{ hint }}</small>
    </span>
    <span v-if="badge !== undefined" class="ui-nav-item__badge">{{ badge }}</span>
  </button>
</template>

<style scoped>
.ui-nav-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  min-height: 44px;
  padding: var(--space-2) var(--space-3);
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-control);
  cursor: pointer;
  transition:
    background var(--duration-fast) var(--ease-standard),
    color var(--duration-fast) var(--ease-standard);
}

.ui-nav-item__icon {
  flex: 0 0 auto;
}

.ui-nav-item__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
  flex: 1 1 auto;
}

.ui-nav-item__label {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-nav-item__hint {
  overflow: hidden;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-nav-item__badge {
  flex: 0 0 auto;
  min-width: 18px;
  padding: 0 6px;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
  line-height: 18px;
  text-align: center;
  background: rgba(var(--surface-rgb), 0.7);
  border-radius: var(--radius-pill);
}

.ui-nav-item:hover {
  background: var(--accent-soft);
}

.ui-nav-item--active {
  color: var(--accent);
  background: var(--accent-soft);
  box-shadow: inset 3px 0 0 var(--accent);
}

.ui-nav-item--active .ui-nav-item__label {
  color: var(--accent-deep);
}
</style>
