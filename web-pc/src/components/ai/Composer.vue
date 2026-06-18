<script setup lang="ts">
import { ref } from 'vue';
import { Send, Square } from 'lucide-vue-next';
import { UiIconButton } from '../ui';

const props = defineProps<{ sending: boolean }>();
const emit = defineEmits<{ (e: 'send', text: string): void; (e: 'stop'): void }>();

const draft = ref('');

function submit() {
  const text = draft.value.trim();
  if (!text || props.sending) return;
  emit('send', text);
  draft.value = '';
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey && !e.isComposing) {
    e.preventDefault();
    submit();
  }
}
</script>

<template>
  <form class="composer" @submit.prevent="submit">
    <textarea
      v-model="draft"
      class="composer__input"
      rows="1"
      placeholder="询问文件、存储、设备或权限状态…（Enter 发送，Shift+Enter 换行）"
      @keydown="onKeydown"
    />
    <UiIconButton
      v-if="sending"
      :icon="Square"
      label="停止生成"
      tone="danger"
      variant="soft"
      type="button"
      @click="emit('stop')"
    />
    <UiIconButton
      v-else
      :icon="Send"
      label="发送"
      tone="primary"
      variant="solid"
      type="submit"
      :disabled="!draft.trim()"
    />
  </form>
</template>

<style scoped>
.composer {
  display: flex;
  align-items: flex-end;
  gap: var(--space-2);
  padding: var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-lg);
  background: var(--surface-solid);
}
.composer__input {
  flex: 1;
  min-height: 24px;
  max-height: 160px;
  padding: var(--space-2);
  border: 0;
  outline: 0;
  resize: vertical;
  background: transparent;
  color: var(--text);
  font: inherit;
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
}
</style>
