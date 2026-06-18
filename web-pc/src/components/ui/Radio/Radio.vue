<script setup lang="ts">
const props = withDefaults(
  defineProps<{
    /** This radio's value. Selected when it equals the v-model. */
    value: string | number | boolean;
    label?: string;
    disabled?: boolean;
    name?: string;
  }>(),
  { label: undefined, disabled: false, name: undefined },
);

const model = defineModel<string | number | boolean>();

const emit = defineEmits<{ change: [value: string | number | boolean] }>();

const checked = () => model.value === props.value;

function select() {
  if (props.disabled) return;
  model.value = props.value;
  emit('change', props.value);
}
</script>

<template>
  <label class="ui-radio" :class="{ 'ui-radio--disabled': disabled }">
    <button
      class="ui-radio__dot"
      :class="{ 'ui-radio__dot--checked': checked() }"
      type="button"
      role="radio"
      :aria-checked="checked()"
      :aria-label="label"
      :name="name"
      :disabled="disabled"
      @click="select"
    />
    <span v-if="label" class="ui-radio__label" @click="select">{{ label }}</span>
  </label>
</template>

<style scoped>
.ui-radio {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text);
  font-size: var(--fs-sm);
  cursor: pointer;
}
.ui-radio--disabled {
  cursor: not-allowed;
  opacity: 0.55;
}

.ui-radio__dot {
  position: relative;
  flex-shrink: 0;
  width: 18px;
  height: 18px;
  padding: 0;
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid var(--border-strong);
  border-radius: 50%;
  cursor: inherit;
  transition: border-color var(--duration-fast) var(--ease-out);
}
.ui-radio__dot--checked {
  border-color: var(--accent);
  border-width: 5px;
}
.ui-radio__dot:focus-visible {
  outline: 2px solid var(--accent);
  outline-offset: 2px;
}

.ui-radio__label {
  user-select: none;
}
</style>
