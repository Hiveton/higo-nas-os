<script setup lang="ts">
import Spinner from './Spinner.vue';

withDefaults(
  defineProps<{
    show: boolean;
    label?: string;
    /** Frost the covered content. Default true. */
    blur?: boolean;
  }>(),
  { label: undefined, blur: true },
);
</script>

<template>
  <transition name="ui-overlay-fade">
    <div v-if="show" class="ui-loading-overlay" :class="{ 'ui-loading-overlay--blur': blur }">
      <Spinner size="lg" />
      <span v-if="label" class="ui-loading-overlay__label">{{ label }}</span>
    </div>
  </transition>
</template>

<style scoped>
.ui-loading-overlay {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: var(--space-3);
  background: rgba(255, 255, 255, 0.42);
  border-radius: inherit;
  z-index: 2;
}

.ui-loading-overlay--blur {
  backdrop-filter: blur(var(--blur-sm));
  -webkit-backdrop-filter: blur(var(--blur-sm));
}

.ui-loading-overlay__label {
  color: var(--text-muted);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}

.ui-overlay-fade-enter-active,
.ui-overlay-fade-leave-active {
  transition: opacity var(--duration-md) var(--ease-out);
}
.ui-overlay-fade-enter-from,
.ui-overlay-fade-leave-to {
  opacity: 0;
}
</style>
