<script setup lang="ts">
import type { UiSize, UiTone } from '../tokens';

withDefaults(
  defineProps<{
    tone?: UiTone;
    variant?: 'solid' | 'soft' | 'dot';
    size?: UiSize;
  }>(),
  { tone: 'neutral', variant: 'soft', size: 'md' },
);
</script>

<template>
  <span
    class="ui-badge"
    :class="[`ui-badge--${tone}`, `ui-badge--${variant}`, `ui-badge--${size}`]"
  >
    <span v-if="variant === 'dot'" class="ui-badge__dot" />
    <slot />
  </span>
</template>

<style scoped>
.ui-badge {
  display: inline-flex;
  align-items: center;
  gap: var(--space-1);
  font-weight: var(--fw-semibold);
  line-height: 1;
  white-space: nowrap;
  border-radius: 999px;
  --bg-color: var(--text-strong);
  --bg-soft: var(--surface-glass);
  /* darker text tone for the soft variant so badge labels clear ~4.5:1 */
  --bg-ink: var(--text);
}

.ui-badge--sm {
  min-height: 18px;
  padding: 0 var(--space-2);
  font-size: var(--fs-2xs);
}
.ui-badge--md {
  min-height: 22px;
  padding: 0 var(--space-3);
  font-size: var(--fs-xs);
}
.ui-badge--lg {
  min-height: 26px;
  padding: 0 var(--space-3);
  font-size: var(--fs-sm);
}

.ui-badge--neutral {
  --bg-color: var(--text-muted);
  --bg-soft: var(--surface-glass);
  --bg-ink: var(--text);
}
.ui-badge--primary {
  --bg-color: var(--accent);
  --bg-soft: var(--accent-soft);
  --bg-ink: var(--ink-blue);
}
.ui-badge--success {
  --bg-color: var(--accent-green);
  --bg-soft: var(--accent-green-soft);
  --bg-ink: var(--ink-green);
}
.ui-badge--warning {
  --bg-color: var(--accent-orange);
  --bg-soft: var(--accent-orange-soft);
  --bg-ink: var(--ink-orange);
}
.ui-badge--danger {
  --bg-color: var(--accent-red);
  --bg-soft: var(--accent-red-soft);
  --bg-ink: var(--ink-red);
}
.ui-badge--info {
  --bg-color: var(--accent-cyan);
  --bg-soft: rgba(22, 199, 221, 0.13);
  --bg-ink: var(--ink-cyan);
}

/* Two-class specificity (0,2,0) so a stray `.container span { color }` rule in a
   window stylesheet can't recolor the badge (the badge root is itself a span). */
.ui-badge.ui-badge--solid {
  color: var(--text-inverse);
  background: var(--bg-color);
}
.ui-badge.ui-badge--soft {
  color: var(--bg-ink);
  background: var(--bg-soft);
}
.ui-badge--dot {
  color: var(--text);
  background: transparent;
  padding-left: 0;
}

.ui-badge__dot {
  width: 7px;
  height: 7px;
  border-radius: 50%;
  background: var(--bg-color);
}
</style>
