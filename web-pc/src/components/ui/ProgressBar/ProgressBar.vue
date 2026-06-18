<script setup lang="ts">
import { computed } from 'vue';
import type { UiSize, UiTone } from '../tokens';

const props = withDefaults(
  defineProps<{
    value?: number;
    max?: number;
    tone?: UiTone;
    size?: UiSize;
    indeterminate?: boolean;
    label?: string;
    showValue?: boolean;
  }>(),
  {
    value: 0,
    max: 100,
    tone: 'primary',
    size: 'md',
    indeterminate: false,
    label: undefined,
    showValue: false,
  },
);

const pct = computed(() => {
  if (props.max <= 0) return 0;
  return Math.min(100, Math.max(0, (props.value / props.max) * 100));
});
</script>

<template>
  <div class="ui-progress" :class="`ui-progress--${size}`">
    <div v-if="label || showValue" class="ui-progress__meta">
      <span v-if="label" class="ui-progress__label">{{ label }}</span>
      <span v-if="showValue" class="ui-progress__value">{{ Math.round(pct) }}%</span>
    </div>
    <div
      class="ui-progress__track"
      role="progressbar"
      :aria-valuenow="indeterminate ? undefined : Math.round(pct)"
      aria-valuemin="0"
      aria-valuemax="100"
      :aria-label="label"
    >
      <div
        class="ui-progress__fill"
        :class="[`ui-progress__fill--${tone}`, { 'ui-progress__fill--indeterminate': indeterminate }]"
        :style="indeterminate ? undefined : { width: `${pct}%` }"
      />
    </div>
  </div>
</template>

<style scoped>
.ui-progress {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  width: 100%;
}

.ui-progress__meta {
  display: flex;
  align-items: center;
  justify-content: space-between;
  font-size: var(--fs-xs);
  color: var(--text-muted);
}
.ui-progress__value {
  font-weight: var(--fw-semibold);
  color: var(--text);
}

.ui-progress__track {
  width: 100%;
  overflow: hidden;
  background: rgba(120, 150, 180, 0.2);
  border-radius: 999px;
}
.ui-progress--sm .ui-progress__track {
  height: 4px;
}
.ui-progress--md .ui-progress__track {
  height: 7px;
}
.ui-progress--lg .ui-progress__track {
  height: 10px;
}

.ui-progress__fill {
  height: 100%;
  border-radius: inherit;
  transition: width var(--duration-md) var(--ease-out);
}
.ui-progress__fill--primary {
  background: linear-gradient(90deg, var(--accent), var(--accent-cyan));
}
.ui-progress__fill--neutral {
  background: var(--text-muted);
}
.ui-progress__fill--success {
  background: var(--accent-green);
}
.ui-progress__fill--warning {
  background: var(--accent-orange);
}
.ui-progress__fill--danger {
  background: var(--accent-red);
}
.ui-progress__fill--info {
  background: var(--accent-cyan);
}

.ui-progress__fill--indeterminate {
  width: 40%;
  animation: ui-progress-slide 1.2s var(--ease-out) infinite;
}

@keyframes ui-progress-slide {
  0% {
    margin-left: -40%;
  }
  100% {
    margin-left: 100%;
  }
}

@media (prefers-reduced-motion: reduce) {
  .ui-progress__fill--indeterminate {
    animation-duration: 2.4s;
  }
}
</style>
