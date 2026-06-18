<script setup lang="ts">
import { X } from 'lucide-vue-next';
import type { UiTone } from '../tokens';

withDefaults(
  defineProps<{
    tone?: UiTone;
    closable?: boolean;
  }>(),
  { tone: 'neutral', closable: false },
);

const emit = defineEmits<{ close: [] }>();
</script>

<template>
  <span class="ui-tag" :class="`ui-tag--${tone}`">
    <slot />
    <button v-if="closable" class="ui-tag__close" type="button" aria-label="移除" @click="emit('close')">
      <X :size="12" :stroke-width="2.4" />
    </button>
  </span>
</template>

<style scoped>
.ui-tag {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  min-height: 24px;
  padding: 0 var(--space-2);
  font-size: var(--fs-xs);
  font-weight: var(--fw-medium);
  border-radius: var(--radius-sm);
  --tag-color: var(--text);
  --tag-soft: var(--surface-glass);
  color: var(--tag-color);
  background: var(--tag-soft);
}

.ui-tag--neutral {
  --tag-color: var(--text);
  --tag-soft: var(--surface-glass);
}
.ui-tag--primary {
  --tag-color: var(--accent);
  --tag-soft: var(--accent-soft);
}
.ui-tag--success {
  --tag-color: var(--accent-green);
  --tag-soft: var(--accent-green-soft);
}
.ui-tag--warning {
  --tag-color: var(--accent-orange);
  --tag-soft: var(--accent-orange-soft);
}
.ui-tag--danger {
  --tag-color: var(--accent-red);
  --tag-soft: var(--accent-red-soft);
}
.ui-tag--info {
  --tag-color: var(--accent-cyan);
  --tag-soft: rgba(22, 199, 221, 0.13);
}

.ui-tag__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  padding: 2px;
  color: inherit;
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
  opacity: 0.7;
}
.ui-tag__close:hover {
  opacity: 1;
}
</style>
