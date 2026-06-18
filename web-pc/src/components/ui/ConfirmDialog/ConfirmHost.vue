<script setup lang="ts">
import ConfirmDialog from './ConfirmDialog.vue';
import { confirmOpen, confirmState } from './useConfirm';

function onConfirm() {
  confirmState.value?.resolve(true);
}
function onCancel() {
  confirmState.value?.resolve(false);
}
</script>

<template>
  <ConfirmDialog
    v-if="confirmState"
    :open="confirmOpen"
    :title="confirmState.title"
    :message="confirmState.message"
    :confirm-label="confirmState.confirmLabel ?? '确定'"
    :cancel-label="confirmState.cancelLabel ?? '取消'"
    :tone="confirmState.tone ?? 'danger'"
    @update:open="(v) => !v && onCancel()"
    @confirm="onConfirm"
    @cancel="onCancel"
  />
</template>
