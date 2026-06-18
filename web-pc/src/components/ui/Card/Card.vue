<script setup lang="ts">
withDefaults(
  defineProps<{
    /** Strength of the glass surface. */
    variant?: 'glass' | 'glass-strong' | 'flat';
    padding?: 'none' | 'sm' | 'md' | 'lg';
    interactive?: boolean;
  }>(),
  { variant: 'glass', padding: 'md', interactive: false },
);
</script>

<template>
  <div
    class="ui-card"
    :class="[
      'u-glass',
      { 'u-glass--strong': variant === 'glass-strong', 'u-glass--flat': variant === 'flat' },
      `ui-card--pad-${padding}`,
      { 'ui-card--interactive': interactive },
    ]"
  >
    <header v-if="$slots.header" class="ui-card__header">
      <slot name="header" />
    </header>
    <div class="ui-card__body">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="ui-card__footer">
      <slot name="footer" />
    </footer>
  </div>
</template>

<style scoped>
.ui-card {
  display: flex;
  flex-direction: column;
  color: var(--text);
}

.ui-card--pad-none .ui-card__body {
  padding: 0;
}
.ui-card--pad-sm .ui-card__body {
  padding: var(--space-3);
}
.ui-card--pad-md .ui-card__body {
  padding: var(--space-4);
}
.ui-card--pad-lg .ui-card__body {
  padding: var(--space-6);
}

.ui-card__header {
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
}

.ui-card__footer {
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--border);
}

.ui-card--interactive {
  cursor: pointer;
  transition:
    transform var(--duration-fast) var(--ease-out),
    box-shadow var(--duration-fast) var(--ease-out);
}
.ui-card--interactive:hover {
  transform: translateY(-2px);
  box-shadow: var(--shadow-md);
}
</style>
