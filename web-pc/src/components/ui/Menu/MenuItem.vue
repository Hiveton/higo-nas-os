<script setup lang="ts">
import type { Component } from 'vue';
import type { UiTone } from '../tokens';

const props = withDefaults(
  defineProps<{
    icon?: Component;
    tone?: UiTone;
    disabled?: boolean;
    shortcut?: string;
  }>(),
  { icon: undefined, tone: 'neutral', disabled: false, shortcut: undefined },
);

const emit = defineEmits<{ select: [] }>();

function onClick() {
  if (props.disabled) return;
  emit('select');
}
</script>

<template>
  <button
    class="ui-menu-item"
    :class="[`ui-menu-item--${tone}`, { 'ui-menu-item--disabled': disabled }]"
    type="button"
    role="menuitem"
    :disabled="disabled"
    @click="onClick"
  >
    <component :is="icon" v-if="icon" class="ui-menu-item__icon" :size="15" :stroke-width="2.1" />
    <span class="ui-menu-item__label"><slot /></span>
    <small v-if="shortcut" class="ui-menu-item__shortcut">{{ shortcut }}</small>
  </button>
</template>

<style scoped>
.ui-menu-item {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  min-height: 34px;
  padding: var(--space-2) var(--space-3);
  color: var(--text);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}
.ui-menu-item:hover,
.ui-menu-item:focus-visible {
  background: var(--accent-soft);
  outline: 0;
}

.ui-menu-item--danger {
  color: var(--accent-red);
}
.ui-menu-item--danger:hover {
  background: var(--accent-red-soft);
}

.ui-menu-item--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ui-menu-item__icon {
  flex-shrink: 0;
  color: currentColor;
}

.ui-menu-item__label {
  flex: 1;
  overflow: hidden;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-menu-item__shortcut {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}
</style>
