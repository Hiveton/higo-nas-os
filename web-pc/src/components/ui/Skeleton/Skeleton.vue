<script setup lang="ts">
withDefaults(
  defineProps<{
    /** Visual shape. */
    variant?: 'text' | 'rect' | 'circle';
    width?: string;
    height?: string;
    /** Number of stacked lines (text variant). */
    lines?: number;
    radius?: string;
  }>(),
  {
    variant: 'rect',
    width: '100%',
    height: undefined,
    lines: 1,
    radius: undefined,
  },
);
</script>

<template>
  <div v-if="variant === 'text'" class="ui-skel-text" aria-hidden="true">
    <span
      v-for="i in lines"
      :key="i"
      class="ui-skel ui-skel--line"
      :style="{ width: i === lines && lines > 1 ? '64%' : width }"
    />
  </div>
  <span
    v-else
    class="ui-skel"
    :class="`ui-skel--${variant}`"
    aria-hidden="true"
    :style="{
      width,
      height: height ?? (variant === 'circle' ? width : '14px'),
      borderRadius: radius ?? (variant === 'circle' ? 'var(--radius-pill)' : 'var(--radius-card)'),
    }"
  />
</template>

<style scoped>
.ui-skel {
  display: block;
  background: linear-gradient(
    100deg,
    var(--surface-glass-weak) 30%,
    rgba(var(--surface-rgb), 0.7) 50%,
    var(--surface-glass-weak) 70%
  );
  background-size: 220% 100%;
  animation: ui-skel-shimmer var(--duration-slow) linear infinite;
}
.ui-skel--line {
  height: 12px;
  border-radius: var(--radius-control);
}
.ui-skel-text {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
}
@keyframes ui-skel-shimmer {
  from { background-position: 180% 0; }
  to { background-position: -20% 0; }
}
@media (prefers-reduced-motion: reduce) {
  .ui-skel { animation: none; }
}
</style>
