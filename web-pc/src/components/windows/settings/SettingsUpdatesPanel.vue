<script setup lang="ts">
import { CheckCircle2, RefreshCw } from 'lucide-vue-next';
import { UiButton, UiFormField, UiSelect, UiSwitch } from '../../ui';

defineProps<{
  settings: { autoUpdate: boolean; releaseChannel: string };
  releaseChannelOptions: { value: string; label: string }[];
  updateStatus: string;
}>();

const emit = defineEmits<{
  (e: 'toggle-auto-update'): void;
  (e: 'check-updates'): void;
}>();
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`夜间自动更新 · ${settings.autoUpdate ? '开启' : '关闭'}`" inline>
      <UiSwitch :model-value="settings.autoUpdate" @change="emit('toggle-auto-update')" />
    </UiFormField>
    <UiFormField label="更新渠道">
      <UiSelect v-model="settings.releaseChannel" :options="releaseChannelOptions" />
    </UiFormField>
    <UiButton variant="soft" size="sm" :icon-left="RefreshCw" @click="emit('check-updates')">检查更新</UiButton>
    <div class="system-settings__metric">
      <CheckCircle2 :size="17" />
      <p>{{ updateStatus }}</p>
    </div>
  </div>
</template>
