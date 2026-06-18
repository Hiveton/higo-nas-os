<script setup lang="ts">
import { computed, inject, ref, watch } from 'vue';
import { TABS_ACTIVE_KEY } from './context';

const props = withDefaults(
  defineProps<{
    tabKey: string;
    /** Mount lazily on first activation, then keep alive with v-show. Default true. */
    lazy?: boolean;
  }>(),
  { lazy: true },
);

const activeKey = inject(TABS_ACTIVE_KEY, ref(''));
const isActive = computed(() => activeKey.value === props.tabKey);

const hasMounted = ref(isActive.value || !props.lazy);
watch(isActive, (active) => {
  if (active) hasMounted.value = true;
});
</script>

<template>
  <div v-show="isActive" class="ui-tab-panel" role="tabpanel" :aria-hidden="!isActive">
    <slot v-if="hasMounted" />
  </div>
</template>

<style scoped>
.ui-tab-panel {
  height: 100%;
  min-height: 0;
}
</style>
