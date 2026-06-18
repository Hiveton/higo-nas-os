<script setup lang="ts">
import type { UiSize } from '../tokens';

export type SegmentedOption = {
  label: string;
  value: string | number;
  disabled?: boolean;
};

const props = withDefaults(
  defineProps<{
    options: SegmentedOption[];
    size?: UiSize;
    /** Stretch options to fill the available width. */
    block?: boolean;
  }>(),
  { size: 'md', block: false },
);

const model = defineModel<string | number>({ required: true });

const emit = defineEmits<{ change: [value: string | number] }>();

function select(option: SegmentedOption) {
  if (option.disabled || option.value === model.value) return;
  model.value = option.value;
  emit('change', option.value);
}
</script>

<template>
  <div
    class="ui-segmented"
    :class="[`ui-segmented--${size}`, { 'ui-segmented--block': block }]"
    role="tablist"
  >
    <button
      v-for="option in options"
      :key="option.value"
      class="ui-segmented__item"
      :class="{ 'ui-segmented__item--active': option.value === model }"
      type="button"
      role="tab"
      :aria-selected="option.value === model"
      :disabled="option.disabled"
      @click="select(option)"
    >
      {{ option.label }}
    </button>
  </div>
</template>

<style scoped>
.ui-segmented {
  display: inline-flex;
  gap: var(--space-1);
  padding: var(--space-1);
  background: var(--surface-glass);
  border-radius: var(--radius-md);
}
.ui-segmented--block {
  display: flex;
  width: 100%;
}

.ui-segmented__item {
  flex: 1;
  color: var(--text-muted);
  font-family: var(--font-ui);
  font-weight: var(--fw-medium);
  white-space: nowrap;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: color var(--duration-fast) var(--ease-out), background var(--duration-fast) var(--ease-out);
}
.ui-segmented--sm .ui-segmented__item {
  min-height: 26px;
  padding: 0 var(--space-3);
  font-size: var(--fs-xs);
}
.ui-segmented--md .ui-segmented__item {
  min-height: 32px;
  padding: 0 var(--space-4);
  font-size: var(--fs-sm);
}
.ui-segmented--lg .ui-segmented__item {
  min-height: 38px;
  padding: 0 var(--space-5);
  font-size: var(--fs-md);
}

.ui-segmented__item--active {
  color: var(--text-strong);
  background: var(--surface-solid);
  box-shadow: var(--shadow-sm);
}
.ui-segmented__item:disabled {
  cursor: not-allowed;
  opacity: 0.5;
}
.ui-segmented__item:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 1px;
}
</style>
