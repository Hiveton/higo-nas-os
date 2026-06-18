<script setup lang="ts">
import { LockKeyhole } from 'lucide-vue-next';
import { UiFormField, UiSegmented, UiSwitch } from '../../ui';

defineProps<{
  settings: { privacyMode: string; sensitiveLocalOnly: boolean };
  privacySummary: string;
}>();

const emit = defineEmits<{
  (e: 'set-privacy-mode', mode: string): void;
  (e: 'toggle-sensitive'): void;
}>();

const privacyModeOptions = [
  { value: '家庭默认', label: '家庭默认' },
  { value: '隐身优先', label: '隐身优先' },
  { value: '企业合规', label: '企业合规' },
];
</script>

<template>
  <div class="system-settings__panel">
    <UiSegmented
      :model-value="settings.privacyMode"
      :options="privacyModeOptions"
      aria-label="隐私模式选择"
      @change="(v) => emit('set-privacy-mode', String(v))"
    />
    <UiFormField :label="`敏感文件禁止云端处理 · ${settings.sensitiveLocalOnly ? '强制本地' : '按策略路由'}`" inline>
      <UiSwitch :model-value="settings.sensitiveLocalOnly" @change="emit('toggle-sensitive')" />
    </UiFormField>
    <div class="system-settings__metric system-settings__metric--privacy">
      <LockKeyhole :size="17" />
      <p>{{ privacySummary }}</p>
    </div>
  </div>
</template>
