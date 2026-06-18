<script setup lang="ts">
import type { UiSize, UiTone } from '../tokens';

withDefaults(
  defineProps<{
    size?: UiSize;
    tone?: UiTone;
  }>(),
  { size: 'md', tone: 'primary' },
);
</script>

<template>
  <span
    class="ui-spinner"
    :class="[`ui-spinner--${size}`, `ui-spinner--${tone}`]"
    role="status"
    aria-label="加载中"
  />
</template>

<style scoped>
.ui-spinner {
  display: inline-block;
  border-radius: 50%;
  border-style: solid;
  border-color: var(--spin-track) var(--spin-track) var(--spin-track) var(--spin-color);
  animation: ui-spinner-rotate 0.7s linear infinite;
  --spin-track: rgba(120, 150, 180, 0.25);
  --spin-color: var(--accent);
}

.ui-spinner--sm {
  width: 14px;
  height: 14px;
  border-width: 2px;
}
.ui-spinner--md {
  width: 18px;
  height: 18px;
  border-width: 2px;
}
.ui-spinner--lg {
  width: 26px;
  height: 26px;
  border-width: 3px;
}

.ui-spinner--neutral {
  --spin-color: var(--text-strong);
}
.ui-spinner--primary {
  --spin-color: var(--accent);
}
.ui-spinner--success {
  --spin-color: var(--accent-green);
}
.ui-spinner--warning {
  --spin-color: var(--accent-orange);
}
.ui-spinner--danger {
  --spin-color: var(--accent-red);
}
.ui-spinner--info {
  --spin-color: var(--accent-cyan);
}

@keyframes ui-spinner-rotate {
  to {
    transform: rotate(360deg);
  }
}

@media (prefers-reduced-motion: reduce) {
  .ui-spinner {
    animation-duration: 1.6s;
  }
}
</style>
