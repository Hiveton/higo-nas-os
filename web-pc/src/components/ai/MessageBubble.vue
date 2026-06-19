<script setup lang="ts">
import { computed } from 'vue';
import { CheckCircle2, FileText, X } from 'lucide-vue-next';
import { UiButton } from '../ui';
import MarkdownContent from './MarkdownContent.vue';
import ToolStep from './ToolStep.vue';
import type { ChatMessage, ChatStep } from '../../stores/assistant';

const props = defineProps<{ message: ChatMessage }>();
const emit = defineEmits<{
  (e: 'confirm-action', id: string): void;
  (e: 'cancel-action', id: string): void;
}>();

const isUser = computed(() => props.message.role === 'user');

// Live tool steps while streaming, else persisted traces from history.
const steps = computed<ChatStep[]>(() => {
  if (props.message.steps && props.message.steps.length) return props.message.steps;
  return (props.message.tools ?? []).map((t) => ({ name: t.name, summary: t.summary, running: false }));
});

const showCursor = computed(() => props.message.streaming && !props.message.text);
</script>

<template>
  <div class="bubble-row" :class="isUser ? 'is-user' : 'is-assistant'">
    <div class="bubble">
      <!-- Analysis steps (tool calls) -->
      <div v-if="!isUser && steps.length" class="bubble__steps">
        <ToolStep v-for="(step, i) in steps" :key="`${step.name}-${i}`" :step="step" />
      </div>

      <!-- Content -->
      <p v-if="isUser" class="bubble__text">{{ message.text }}</p>
      <div v-else class="bubble__assistant">
        <span v-if="showCursor" class="bubble__cursor">▍</span>
        <MarkdownContent v-else :source="message.text" />
      </div>

      <!-- Citations -->
      <div v-if="message.citations && message.citations.length" class="bubble__citations">
        <a
          v-for="(c, i) in message.citations"
          :key="i"
          class="citation"
          :href="c.url || undefined"
          :target="c.url ? '_blank' : undefined"
          rel="noopener noreferrer"
        >
          <FileText :size="12" />
          <span>{{ c.title }}</span>
        </a>
      </div>

      <!-- Pending high-risk action -->
      <div v-if="message.pendingActionId" class="bubble__action">
        <UiButton size="sm" tone="warning" variant="solid" :icon-left="CheckCircle2" @click="emit('confirm-action', message.pendingActionId)">
          确认执行
        </UiButton>
        <UiButton size="sm" tone="neutral" variant="soft" :icon-left="X" @click="emit('cancel-action', message.pendingActionId)">
          取消
        </UiButton>
      </div>
    </div>
  </div>
</template>

<style scoped>
.bubble-row {
  display: flex;
  margin-bottom: var(--space-4);
}
.bubble-row.is-user {
  justify-content: flex-end;
}
.bubble {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-width: 84%;
  padding: var(--space-3) var(--space-4);
  border-radius: var(--radius-lg);
}
.is-user .bubble {
  color: var(--text-inverse);
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
}
.is-assistant .bubble {
  border: 1px solid var(--border);
  background: var(--surface-solid);
}
.bubble__text {
  margin: 0;
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
  white-space: pre-wrap;
  word-break: break-word;
}
.bubble__steps {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.bubble__cursor {
  color: var(--accent);
  animation: blink 1s step-start infinite;
}
@keyframes blink {
  50% { opacity: 0; }
}
.bubble__citations {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}
.citation {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 2px var(--space-2);
  border: 1px solid var(--border);
  border-radius: 999px;
  background: var(--surface-glass);
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  text-decoration: none;
}
.bubble__action {
  display: flex;
}
</style>
