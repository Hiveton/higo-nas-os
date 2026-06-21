<script setup lang="ts">
import { ref, watch } from 'vue';
import Modal from '../Modal/Modal.vue';
import Button from '../Button/Button.vue';
import Input from '../Input/Input.vue';
import { inputOpen, inputState } from './useInputDialog';

const value = ref('');
const errorMsg = ref('');

watch(inputState, (s) => {
  value.value = s?.defaultValue ?? '';
  errorMsg.value = '';
});

function submit() {
  const v = String(value.value).trim();
  const err = inputState.value?.validate?.(v) ?? null;
  if (err) {
    errorMsg.value = err;
    return;
  }
  inputState.value?.resolve(v);
}

function cancel() {
  inputState.value?.resolve(null);
}
</script>

<template>
  <Modal
    v-if="inputState"
    :open="inputOpen"
    :title="inputState.title"
    size="sm"
    @update:open="(v) => !v && cancel()"
    @close="cancel"
  >
    <div class="ui-input-dialog">
      <label v-if="inputState.label" class="ui-input-dialog__label">{{ inputState.label }}</label>
      <Input
        v-model="value"
        :placeholder="inputState.placeholder"
        :invalid="!!errorMsg"
        clearable
        @keyup.enter="submit"
      />
      <p v-if="errorMsg" class="ui-input-dialog__error">{{ errorMsg }}</p>
    </div>
    <template #footer>
      <Button variant="ghost" tone="neutral" @click="cancel">{{ inputState.cancelLabel ?? '取消' }}</Button>
      <Button tone="primary" @click="submit">{{ inputState.confirmLabel ?? '确定' }}</Button>
    </template>
  </Modal>
</template>

<style scoped>
.ui-input-dialog {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
.ui-input-dialog__label {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.ui-input-dialog__error {
  margin: 0;
  color: var(--accent-red);
  font-size: var(--fs-xs);
}
</style>
