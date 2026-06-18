<script setup lang="ts">
import { Check, Minus } from 'lucide-vue-next';

const props = withDefaults(
  defineProps<{
    label?: string;
    disabled?: boolean;
    indeterminate?: boolean;
    id?: string;
  }>(),
  { label: undefined, disabled: false, indeterminate: false, id: undefined },
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
  <label class="ui-checkbox" :class="{ 'ui-checkbox--disabled': disabled }">
    <button
      :id="id"
      class="ui-checkbox__box"
      :class="{ 'ui-checkbox__box--checked': model || indeterminate }"
      type="button"
      role="checkbox"
      :aria-checked="indeterminate ? 'mixed' : model"
      :aria-label="label"
      :disabled="disabled"
      @click="toggle"
    >
      <Minus v-if="indeterminate" :size="13" :stroke-width="3" />
      <Check v-else-if="model" :size="13" :stroke-width="3" />
    </button>
    <span v-if="label" class="ui-checkbox__label" @click="toggle">{{ label }}</span>
  </label>
</template>

<style scoped>
.ui-checkbox {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text);
  font-size: var(--fs-sm);
  cursor: pointer;
}
.ui-checkbox--disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-checkbox__box {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  padding: 0;
  color: var(--text-inverse);
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid var(--border-strong);
  border-radius: 5px;
  cursor: inherit;
  transition: background var(--duration-fast) var(--ease-out), border-color var(--duration-fast) var(--ease-out);
}
.ui-checkbox__box--checked {
  background: var(--accent);
  border-color: var(--accent);
}
.ui-checkbox__box:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.ui-checkbox__label {
  user-select: none;
}
</style>
