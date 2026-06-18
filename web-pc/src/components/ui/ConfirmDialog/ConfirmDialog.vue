<script setup lang="ts">
import type { UiTone } from '../tokens';
import Modal from '../Modal/Modal.vue';
import Button from '../Button/Button.vue';

withDefaults(
  defineProps<{
    open: boolean;
    title: string;
    message?: string;
    confirmLabel?: string;
    cancelLabel?: string;
    tone?: UiTone;
    loading?: boolean;
  }>(),
  {
    message: undefined,
    confirmLabel: '确定',
    cancelLabel: '取消',
    tone: 'danger',
    loading: false,
  },
);

const emit = defineEmits<{
  'update:open': [value: boolean];
  confirm: [];
  cancel: [];
}>();

function onCancel() {
  emit('update:open', false);
  emit('cancel');
}
</script>

<template>
  <Modal
    :open="open"
    :title="title"
    size="sm"
    :close-on-backdrop="!loading"
    :close-on-esc="!loading"
    @update:open="(v) => emit('update:open', v)"
    @close="onCancel"
  >
    <p v-if="message" class="ui-confirm__message">{{ message }}</p>
    <slot />
    <template #footer>
      <Button variant="ghost" tone="neutral" :disabled="loading" @click="onCancel">
        {{ cancelLabel }}
      </Button>
      <Button :tone="tone" :loading="loading" @click="emit('confirm')">
        {{ confirmLabel }}
      </Button>
    </template>
  </Modal>
</template>

<style scoped>
.ui-confirm__message {
  margin: 0;
  color: var(--text);
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
}
</style>
