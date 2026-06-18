<script setup lang="ts">
withDefaults(
  defineProps<{
    rows?: number;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
  }>(),
  { rows: 3, placeholder: undefined, disabled: false, invalid: false, id: undefined },
);

const model = defineModel<string>({ default: '' });

const emit = defineEmits<{ input: [value: string]; blur: [] }>();

function onInput(event: Event) {
  const value = (event.target as HTMLTextAreaElement).value;
  model.value = value;
  emit('input', value);
}
</script>

<template>
  <textarea
    :id="id"
    class="ui-textarea"
    :class="{ 'ui-textarea--invalid': invalid }"
    :rows="rows"
    :value="model"
    :placeholder="placeholder"
    :disabled="disabled"
    :aria-invalid="invalid || undefined"
    @input="onInput"
    @blur="emit('blur')"
  />
</template>

<style scoped>
.ui-textarea {
  width: 100%;
  padding: var(--space-2) var(--space-3);
  color: var(--text);
  font-family: var(--font-ui);
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
  background: var(--control-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  resize: vertical;
  transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
}
.ui-textarea::placeholder {
  color: var(--text-soft);
}
.ui-textarea:focus {
  outline: 0;
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.ui-textarea--invalid {
  border-color: var(--accent-red);
}
.ui-textarea:disabled {
  cursor: not-allowed;
  opacity: 0.55;
}
</style>
