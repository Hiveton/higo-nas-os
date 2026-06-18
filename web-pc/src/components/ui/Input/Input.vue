<script setup lang="ts">
import type { Component } from 'vue';
import { X } from 'lucide-vue-next';
import type { UiSize } from '../tokens';

const props = withDefaults(
  defineProps<{
    type?: string;
    size?: UiSize;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    clearable?: boolean;
    prefixIcon?: Component;
    suffixIcon?: Component;
    id?: string;
  }>(),
  {
    type: 'text',
    size: 'md',
    placeholder: undefined,
    disabled: false,
    invalid: false,
    clearable: false,
    prefixIcon: undefined,
    suffixIcon: undefined,
    id: undefined,
  },
);

const model = defineModel<string | number>({ default: '' });

const emit = defineEmits<{
  input: [value: string];
  enter: [];
  blur: [];
  clear: [];
}>();

const iconSize: Record<UiSize, number> = { sm: 14, md: 15, lg: 17 };

function onInput(event: Event) {
  const value = (event.target as HTMLInputElement).value;
  model.value = value;
  emit('input', value);
}

function onClear() {
  model.value = '';
  emit('clear');
}
</script>

<template>
  <div
    class="ui-input"
    :class="[`ui-input--${size}`, { 'ui-input--invalid': invalid, 'ui-input--disabled': disabled }]"
  >
    <component :is="prefixIcon" v-if="prefixIcon" class="ui-input__icon" :size="iconSize[size]" :stroke-width="2" />
    <input
      :id="id"
      class="ui-input__field"
      :type="type"
      :value="model"
      :placeholder="placeholder"
      :disabled="disabled"
      :aria-invalid="invalid || undefined"
      @input="onInput"
      @keydown.enter="emit('enter')"
      @blur="emit('blur')"
    />
    <button
      v-if="clearable && model !== '' && !disabled"
      class="ui-input__clear"
      type="button"
      aria-label="清空"
      @click="onClear"
    >
      <X :size="13" :stroke-width="2.4" />
    </button>
    <component :is="suffixIcon" v-if="suffixIcon" class="ui-input__icon" :size="iconSize[size]" :stroke-width="2" />
  </div>
</template>

<style scoped>
.ui-input {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  color: var(--text);
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
}

.ui-input--sm {
  min-height: 28px;
  padding: 0 var(--space-2);
}
.ui-input--md {
  min-height: 34px;
  padding: 0 var(--space-3);
}
.ui-input--lg {
  min-height: 42px;
  padding: 0 var(--space-4);
}

.ui-input:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.ui-input--invalid {
  border-color: var(--accent-red);
}
.ui-input--invalid:focus-within {
  box-shadow: 0 0 0 3px var(--accent-red-soft);
}

.ui-input--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.ui-input__field {
  flex: 1;
  min-width: 0;
  color: inherit;
  font-family: var(--font-ui);
  font-size: var(--fs-sm);
  background: transparent;
  border: 0;
  outline: 0;
}
.ui-input__field::placeholder {
  color: var(--text-soft);
}

.ui-input__icon {
  flex-shrink: 0;
  color: var(--text-soft);
}

.ui-input__clear {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  color: var(--text-soft);
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
}
.ui-input__clear:hover {
  color: var(--text);
  background: var(--surface-glass);
}
</style>
