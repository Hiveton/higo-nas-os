<script setup lang="ts">
import type { Component } from 'vue';
import type { UiSize, UiTone } from '../tokens';
import Spinner from '../Spinner/Spinner.vue';

const props = withDefaults(
  defineProps<{
    variant?: 'solid' | 'soft' | 'ghost' | 'outline';
    tone?: UiTone;
    size?: UiSize;
    block?: boolean;
    loading?: boolean;
    disabled?: boolean;
    iconLeft?: Component;
    iconRight?: Component;
    type?: 'button' | 'submit' | 'reset';
  }>(),
  {
    variant: 'solid',
    tone: 'primary',
    size: 'md',
    block: false,
    loading: false,
    disabled: false,
    iconLeft: undefined,
    iconRight: undefined,
    type: 'button',
  },
);

const emit = defineEmits<{ click: [event: MouseEvent] }>();

const iconSize: Record<UiSize, number> = { sm: 13, md: 15, lg: 17 };

function onClick(event: MouseEvent) {
  if (props.disabled || props.loading) {
    event.preventDefault();
    event.stopPropagation();
    return;
  }
  emit('click', event);
}
</script>

<template>
  <button
    class="ui-btn"
    :class="[
      `ui-btn--${variant}`,
      `ui-btn--${tone}`,
      `ui-btn--${size}`,
      { 'ui-btn--block': block, 'ui-btn--loading': loading, 'ui-btn--disabled': disabled },
    ]"
    :type="type"
    :disabled="disabled || loading"
    :aria-busy="loading || undefined"
    @click="onClick"
  >
    <Spinner v-if="loading" :size="size" class="ui-btn__spinner" />
    <component :is="iconLeft" v-else-if="iconLeft" :size="iconSize[size]" :stroke-width="2.2" />
    <span v-if="$slots.default" class="ui-btn__label"><slot /></span>
    <component :is="iconRight" v-if="iconRight && !loading" :size="iconSize[size]" :stroke-width="2.2" />
  </button>
</template>

<style scoped>
.ui-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: var(--space-2);
  font-family: var(--font-ui);
  font-weight: var(--fw-semibold);
  line-height: 1;
  white-space: nowrap;
  border: 1px solid transparent;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition:
    transform var(--duration-fast) var(--ease-out),
    background var(--duration-fast) var(--ease-out),
    border-color var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out);
  /* tone resolves to a single accent var consumed by all variants */
  --btn-accent: var(--accent);
  --btn-accent-deep: var(--accent-deep);
  --btn-soft: var(--accent-soft);
}

/* Sizes */
.ui-btn--sm {
  min-height: 28px;
  padding: 0 var(--space-3);
  font-size: var(--fs-xs);
}
.ui-btn--md {
  min-height: 34px;
  padding: 0 var(--space-4);
  font-size: var(--fs-sm);
}
.ui-btn--lg {
  min-height: 42px;
  padding: 0 var(--space-5);
  font-size: var(--fs-md);
}

.ui-btn--block {
  width: 100%;
}

/* Tones — only remap the accent variables */
.ui-btn--neutral {
  --btn-accent: var(--text-strong);
  --btn-accent-deep: var(--text);
  --btn-soft: var(--surface-glass);
}
.ui-btn--primary {
  --btn-accent: var(--accent);
  --btn-accent-deep: var(--accent-deep);
  --btn-soft: var(--accent-soft);
}
.ui-btn--success {
  --btn-accent: var(--accent-green);
  --btn-accent-deep: var(--accent-green);
  --btn-soft: var(--accent-green-soft);
}
.ui-btn--warning {
  --btn-accent: var(--accent-orange);
  --btn-accent-deep: var(--accent-orange);
  --btn-soft: var(--accent-orange-soft);
}
.ui-btn--danger {
  --btn-accent: var(--accent-red);
  --btn-accent-deep: var(--accent-red);
  --btn-soft: var(--accent-red-soft);
}
.ui-btn--info {
  --btn-accent: var(--accent-cyan);
  --btn-accent-deep: var(--accent-cyan);
  --btn-soft: rgba(22, 199, 221, 0.13);
}

/* Variants */
.ui-btn--solid {
  color: var(--text-inverse);
  background: linear-gradient(180deg, var(--btn-accent), var(--btn-accent-deep));
  box-shadow: 0 8px 18px rgba(34, 78, 116, 0.18), 0 1px 0 rgba(255, 255, 255, 0.28) inset;
}
.ui-btn--solid.ui-btn--neutral {
  color: var(--text-strong);
  background: rgba(var(--surface-rgb), 0.72);
  border-color: var(--border);
}

.ui-btn--soft {
  color: var(--btn-accent);
  background: var(--btn-soft);
}

.ui-btn--ghost {
  color: var(--btn-accent);
  background: transparent;
}

.ui-btn--outline {
  color: var(--btn-accent);
  background: transparent;
  border-color: var(--btn-accent);
}

/* Interaction */
.ui-btn:not(.ui-btn--disabled):not(.ui-btn--loading):hover {
  transform: translateY(-1px);
}
.ui-btn--soft:not(.ui-btn--disabled):hover,
.ui-btn--ghost:not(.ui-btn--disabled):hover {
  background: var(--btn-soft);
}
.ui-btn:not(.ui-btn--disabled):active {
  transform: translateY(0);
}
.ui-btn:focus-visible {
  outline: 2px solid var(--btn-accent);
  outline-offset: 2px;
}

.ui-btn--disabled,
.ui-btn--loading {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-btn__label {
  overflow: hidden;
  text-overflow: ellipsis;
}
</style>
