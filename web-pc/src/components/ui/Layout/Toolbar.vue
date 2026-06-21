<script setup lang="ts">
withDefaults(defineProps<{ sticky?: boolean }>(), { sticky: false });
</script>

<template>
  <div class="ui-toolbar" :class="{ 'ui-toolbar--sticky': sticky }">
    <div v-if="$slots.start || $slots.default" class="ui-toolbar__group ui-toolbar__group--start">
      <slot name="start" />
      <slot />
    </div>
    <div v-if="$slots.end" class="ui-toolbar__group ui-toolbar__group--end">
      <slot name="end" />
    </div>
  </div>
</template>

<style scoped>
.ui-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.ui-toolbar--sticky {
  position: sticky;
  top: 0;
  z-index: var(--z-sticky);
}

.ui-toolbar__group {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: var(--space-2);
  min-width: 0;
}

.ui-toolbar__group--start {
  flex: 1 1 auto;
}

.ui-toolbar__group--end {
  flex: 0 0 auto;
  justify-content: flex-end;
}
</style>
