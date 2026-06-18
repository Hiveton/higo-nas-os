<script setup lang="ts">
type WidgetId = 'system' | 'cpu' | 'memory' | 'network' | 'disk' | 'storage' | 'backup' | 'docker' | 'security' | 'alerts';
type WidgetOption = { id: WidgetId; label: string };

defineProps<{
  widgetOptions: readonly WidgetOption[];
  visibleWidgets: Record<WidgetId, boolean>;
}>();

const emit = defineEmits<{
  (e: 'reset'): void;
  (e: 'update', id: WidgetId, event: Event): void;
}>();
</script>

<template>
  <section class="overview-settings" aria-label="总览显示设置">
    <div class="overview-settings__head">
      <strong>显示卡片</strong>
      <button type="button" @click="emit('reset')">全部显示</button>
    </div>
    <label v-for="option in widgetOptions" :key="option.id">
      <input
        type="checkbox"
        :checked="visibleWidgets[option.id]"
        @change="emit('update', option.id, $event)"
      />
      <span>{{ option.label }}</span>
    </label>
  </section>
</template>
