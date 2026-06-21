<script setup lang="ts">
withDefaults(defineProps<{ title?: string; subtitle?: string }>(), {
  title: undefined,
  subtitle: undefined,
});
</script>

<template>
  <nav class="ui-nav-rail">
    <header v-if="title || $slots.header" class="ui-nav-rail__header">
      <slot name="header">
        <div class="ui-nav-rail__heading">
          <strong>{{ title }}</strong>
          <span v-if="subtitle">{{ subtitle }}</span>
        </div>
      </slot>
    </header>
    <div class="ui-nav-rail__items">
      <slot />
    </div>
    <footer v-if="$slots.footer" class="ui-nav-rail__footer">
      <slot name="footer" />
    </footer>
  </nav>
</template>

<style scoped>
.ui-nav-rail {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  min-height: 0;
  height: 100%;
  overflow: hidden;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.ui-nav-rail__header {
  padding: var(--space-3);
  border-bottom: 1px solid var(--border);
}

.ui-nav-rail__heading {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.ui-nav-rail__heading strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-nav-rail__heading span {
  overflow: hidden;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ui-nav-rail__items {
  display: grid;
  align-content: start;
  gap: var(--space-1);
  min-height: 0;
  padding: var(--space-2);
  overflow: auto;
}

.ui-nav-rail__footer {
  padding: var(--space-2) var(--space-3);
  border-top: 1px solid var(--border);
}
</style>
