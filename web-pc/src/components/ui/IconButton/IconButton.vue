<script setup lang="ts">
import type { Component } from 'vue';
import type { UiSize, UiTone } from '../tokens';
import Spinner from '../Spinner/Spinner.vue';

const props = withDefaults(
  defineProps<{
    icon: Component;
    /** Accessible label — becomes aria-label and tooltip title. */
    label: string;
    tone?: UiTone;
    size?: UiSize;
    variant?: 'soft' | 'ghost' | 'solid';
    active?: boolean;
    loading?: boolean;
    disabled?: boolean;
    type?: 'button' | 'submit';
  }>(),
  {
    tone: 'neutral',
    size: 'md',
    variant: 'ghost',
    active: false,
    loading: false,
    disabled: false,
    type: 'button',
  },
);

const emit = defineEmits<{ click: [event: MouseEvent] }>();

const glyphSize: Record<UiSize, number> = { sm: 15, md: 17, lg: 19 };

function onClick(event: MouseEvent) {
  if (props.disabled || props.loading) return;
  emit('click', event);
}
</script>

<template>
  <button
    class="ui-icon-btn"
    :class="[
      `ui-icon-btn--${variant}`,
      `ui-icon-btn--${tone}`,
      `ui-icon-btn--${size}`,
      { 'ui-icon-btn--active': active, 'ui-icon-btn--disabled': disabled },
    ]"
    :type="type"
    :disabled="disabled || loading"
    :aria-label="label"
    :title="label"
    :aria-pressed="active || undefined"
    @click="onClick"
  >
    <Spinner v-if="loading" :size="size" />
    <component :is="icon" v-else :size="glyphSize[size]" :stroke-width="2.1" />
  </button>
</template>

<style scoped>
.ui-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: var(--ib-color, var(--text));
  background: transparent;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    transform var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out),
    color var(--duration-fast) var(--ease-out);
  --ib-color: var(--text);
  --ib-soft: var(--surface-glass);
}

.ui-icon-btn--sm {
  width: 28px;
  height: 28px;
}
.ui-icon-btn--md {
  width: 34px;
  height: 34px;
}
.ui-icon-btn--lg {
  width: 40px;
  height: 40px;
}

.ui-icon-btn--neutral {
  --ib-color: var(--text);
  --ib-soft: var(--surface-glass);
}
.ui-icon-btn--primary {
  --ib-color: var(--accent);
  --ib-soft: var(--accent-soft);
}
.ui-icon-btn--success {
  --ib-color: var(--accent-green);
  --ib-soft: var(--accent-green-soft);
}
.ui-icon-btn--warning {
  --ib-color: var(--accent-orange);
  --ib-soft: var(--accent-orange-soft);
}
.ui-icon-btn--danger {
  --ib-color: var(--accent-red);
  --ib-soft: var(--accent-red-soft);
}
.ui-icon-btn--info {
  --ib-color: var(--accent-cyan);
  --ib-soft: rgba(22, 199, 221, 0.13);
}

.ui-icon-btn--soft {
  background: var(--ib-soft);
}
.ui-icon-btn--solid {
  color: var(--text-inverse);
  background: var(--ib-color);
}

.ui-icon-btn:not(.ui-icon-btn--disabled):hover {
  background: var(--ib-soft);
}
.ui-icon-btn--active {
  color: var(--ib-color);
  background: var(--ib-soft);
}
.ui-icon-btn:focus-visible {
  outline: 2px solid var(--ib-color);
  outline-offset: 2px;
}
.ui-icon-btn--disabled {
  cursor: not-allowed;
  opacity: 0.45;
}
</style>
