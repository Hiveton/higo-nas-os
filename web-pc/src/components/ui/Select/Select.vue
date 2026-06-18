<script setup lang="ts">
import { ChevronDown } from 'lucide-vue-next';
import type { UiSize } from '../tokens';

export type SelectOption = {
  label: string;
  value: string | number;
  disabled?: boolean;
};

withDefaults(
  defineProps<{
    options: SelectOption[];
    size?: UiSize;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
  }>(),
  {
    size: 'md',
    placeholder: undefined,
    disabled: false,
    invalid: false,
    id: undefined,
  },
);

const model = defineModel<string | number>({ default: '' });

const emit = defineEmits<{ change: [value: string | number] }>();

function onChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value;
  model.value = value;
  emit('change', value);
}
</script>

<template>
  <div
    class="ui-select"
    :class="[`ui-select--${size}`, { 'ui-select--invalid': invalid, 'ui-select--disabled': disabled }]"
  >
    <select
      :id="id"
      class="ui-select__field"
      :value="model"
      :disabled="disabled"
      :aria-invalid="invalid || undefined"
      @change="onChange"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value" :disabled="opt.disabled">
        {{ opt.label }}
      </option>
    </select>
    <ChevronDown class="ui-select__chevron" :size="15" :stroke-width="2.2" />
  </div>
</template>

<style scoped>
.ui-select {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 100%;
  color: var(--text);
  background: var(--control-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
}

.ui-select--sm {
  min-height: 28px;
}
.ui-select--md {
  min-height: 34px;
}
.ui-select--lg {
  min-height: 42px;
}

.ui-select:focus-within {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.ui-select--invalid {
  border-color: var(--accent-red);
}
.ui-select--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.ui-select__field {
  flex: 1;
  min-width: 0;
  height: 100%;
  padding: 0 var(--space-7, 28px) 0 var(--space-3);
  color: inherit;
  font-family: var(--font-ui);
  font-size: var(--fs-sm);
  background: transparent;
  border: 0;
  outline: 0;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
}

.ui-select__chevron {
  position: absolute;
  right: var(--space-2);
  color: var(--text-soft);
  pointer-events: none;
}
</style>
