<script setup lang="ts">
import type { UiSize } from '../tokens';

const props = withDefaults(
  defineProps<{
    disabled?: boolean;
    size?: UiSize;
    label?: string;
    id?: string;
  }>(),
  { disabled: false, size: 'md', label: undefined, id: undefined },
);

const model = defineModel<boolean>({ default: false });

const emit = defineEmits<{ change: [value: boolean] }>();

function toggle() {
  if (props.disabled) return;
  model.value = !model.value;
  emit('change', model.value);
}
</script>

<template>
  <button
    :id="id"
    class="ui-switch"
    :class="[`ui-switch--${size}`, { 'ui-switch--on': model, 'ui-switch--disabled': disabled }]"
    type="button"
    role="switch"
    :aria-checked="model"
    :aria-label="label"
    :disabled="disabled"
    @click="toggle"
  >
    <span class="ui-switch__track"><span class="ui-switch__thumb" /></span>
    <span v-if="label" class="ui-switch__label">{{ label }}</span>
  </button>
</template>

<style scoped>
.ui-switch {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0;
  color: var(--text);
  font-family: var(--font-ui);
  font-size: var(--fs-sm);
  background: transparent;
  border: 0;
  cursor: pointer;
}
.ui-switch--disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-switch__track {
  position: relative;
  display: inline-flex;
  flex-shrink: 0;
  align-items: center;
  background: rgba(120, 150, 180, 0.32);
  border-radius: 999px;
  transition: background var(--duration-fast) var(--ease-out);
}
.ui-switch--sm .ui-switch__track {
  width: 30px;
  height: 18px;
}
.ui-switch--md .ui-switch__track {
  width: 38px;
  height: 22px;
}
.ui-switch--lg .ui-switch__track {
  width: 46px;
  height: 26px;
}

.ui-switch--on .ui-switch__track {
  background: var(--accent);
}

.ui-switch__thumb {
  position: absolute;
  left: 2px;
  aspect-ratio: 1;
  height: calc(100% - 4px);
  background: var(--text-inverse);
  border-radius: 50%;
  box-shadow: 0 1px 3px rgba(15, 35, 60, 0.3);
  transition: transform var(--duration-fast) var(--ease-out);
}
.ui-switch--on .ui-switch__thumb {
  transform: translateX(calc(100%));
}

.ui-switch__label {
  user-select: none;
}

.ui-switch:focus-visible {
  outline: 0;
}
.ui-switch:focus-visible .ui-switch__track {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}
</style>
