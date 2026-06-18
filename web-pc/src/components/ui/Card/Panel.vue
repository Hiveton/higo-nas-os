<script setup lang="ts">
withDefaults(
  defineProps<{
    title?: string;
    /** Show a glass surface around the panel. Default false (transparent section). */
    framed?: boolean;
  }>(),
  { title: undefined, framed: false },
);
</script>

<template>
  <section class="ui-panel" :class="{ 'u-glass': framed }">
    <header v-if="title || $slots.title || $slots.actions" class="ui-panel__header">
      <div class="ui-panel__title">
        <slot name="title">{{ title }}</slot>
      </div>
      <div v-if="$slots.actions" class="ui-panel__actions">
        <slot name="actions" />
      </div>
    </header>
    <div class="ui-panel__body">
      <slot />
    </div>
  </section>
</template>

<style scoped>
.ui-panel {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.ui-panel.u-glass {
  padding: var(--space-4);
}

.ui-panel__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.ui-panel__title {
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
}

.ui-panel__actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.ui-panel__body {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
