<script setup lang="ts">
import { ListChecks, Trash2 } from 'lucide-vue-next';
import { activityStore } from '../../../stores/activity';
import { UiButton, UiFormField, UiSelect, UiSwitch, useConfirm, useToast } from '../../ui';

const props = defineProps<{
  settings: { activityEnabled: boolean; activityMaxEntries: number };
}>();

const emit = defineEmits<{
  (e: 'toggle-enabled'): void;
  (e: 'set-max-entries', value: number): void;
}>();

const confirm = useConfirm();
const toast = useToast();

const maxEntriesOptions = [
  { value: '500', label: '500 条' },
  { value: '2000', label: '2000 条' },
  { value: '5000', label: '5000 条' },
];

async function clearActivity() {
  const ok = await confirm({
    title: '清除操作记录？',
    message: '本地与后端的操作记录都会被清空，且无法恢复。',
    confirmLabel: '清除',
    tone: 'danger',
  });
  if (!ok) return;
  await activityStore.clear();
  toast.success('操作记录已清除。');
}
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`启用操作记录 · ${props.settings.activityEnabled ? '记录中' : '已关闭'}`" inline>
      <UiSwitch :model-value="props.settings.activityEnabled" @change="emit('toggle-enabled')" />
    </UiFormField>
    <UiFormField label="保留条数">
      <UiSelect
        :model-value="String(props.settings.activityMaxEntries)"
        :options="maxEntriesOptions"
        @change="(v) => emit('set-max-entries', Number(v))"
      />
    </UiFormField>
    <div class="system-settings__metric">
      <ListChecks :size="17" />
      <p>
        记录页面访问与有意义的操作，最多保留 {{ props.settings.activityMaxEntries }} 条。可在安全中心查看完整记录。
      </p>
    </div>
    <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="clearActivity">清除操作记录</UiButton>
  </div>
</template>
