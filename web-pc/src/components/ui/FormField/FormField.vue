<script setup lang="ts">
withDefaults(
  defineProps<{
    label?: string;
    for?: string;
    error?: string;
    hint?: string;
    required?: boolean;
    inline?: boolean;
  }>(),
  {
    label: undefined,
    for: undefined,
    error: undefined,
    hint: undefined,
    required: false,
    inline: false,
  },
);
</script>

<template>
  <div class="ui-field" :class="{ 'ui-field--inline': inline, 'ui-field--invalid': !!error }">
    <label v-if="label || $slots.label" class="ui-field__label" :for="for">
      <slot name="label">{{ label }}</slot>
      <span v-if="required" class="ui-field__required" aria-hidden="true">*</span>
    </label>
    <div class="ui-field__control">
      <slot />
      <p v-if="error" class="ui-field__error">{{ error }}</p>
      <p v-else-if="hint" class="ui-field__hint">{{ hint }}</p>
    </div>
  </div>
</template>

<style scoped>
.ui-field {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}

.ui-field--inline {
  flex-direction: row;
  align-items: center;
  gap: var(--space-4);
}
.ui-field--inline .ui-field__label {
  flex-shrink: 0;
  width: 120px;
}
.ui-field--inline .ui-field__control {
  flex: 1;
}

.ui-field__label {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.ui-field__required {
  color: var(--accent-red);
}

.ui-field__control {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}

.ui-field__error {
  margin: 0;
  color: var(--accent-red);
  font-size: var(--fs-xs);
}

.ui-field__hint {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
</style>
