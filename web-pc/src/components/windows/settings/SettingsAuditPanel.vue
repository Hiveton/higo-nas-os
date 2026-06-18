<script setup lang="ts">
import { History } from 'lucide-vue-next';
import { computed } from 'vue';
import { UiSegmented } from '../../ui';

const props = defineProps<{
  settings: { auditRetention: string };
  retentionOptions: string[];
}>();

const emit = defineEmits<{
  (e: 'set-retention', retention: string): void;
}>();

const retentionSegments = computed(() => props.retentionOptions.map((r) => ({ value: r, label: r })));
</script>

<template>
  <div class="system-settings__panel">
    <UiSegmented
      :model-value="settings.auditRetention"
      :options="retentionSegments"
      aria-label="审计保留周期"
      @change="(v) => emit('set-retention', String(v))"
    />
    <div class="system-settings__metric">
      <History :size="17" />
      <p>当前保留 {{ settings.auditRetention }}，记录身份、工具调用、数据范围、设置修改和回滚方式。</p>
    </div>
    <div class="system-settings__audit-list">
      <span>权限修改</span>
      <span>模型调用</span>
      <span>分享链接</span>
      <span>备份任务</span>
    </div>
  </div>
</template>
