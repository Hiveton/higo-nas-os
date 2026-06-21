<script setup lang="ts">
withDefaults(defineProps<{ min?: string }>(), { min: '130px' });
</script>

<template>
  <div class="ui-stat-grid" :style="{ '--stat-min': min }">
    <slot />
  </div>
</template>

<style scoped>
.ui-stat-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(var(--stat-min), 1fr));
  gap: var(--space-3);
}
</style>
