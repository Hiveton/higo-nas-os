<script setup lang="ts">
import { Globe2 } from 'lucide-vue-next';
import { UiFormField, UiSelect, UiSwitch } from '../../ui';

defineProps<{
  settings: { ddnsEnabled: boolean; remoteAccess: boolean; dnsProfile: string };
  dnsProfileOptions: { value: string; label: string }[];
}>();

const emit = defineEmits<{
  (e: 'toggle-ddns'): void;
  (e: 'toggle-remote'): void;
}>();
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`DDNS higo-home.direct · ${settings.ddnsEnabled ? '解析中' : '暂停'}`" inline>
      <UiSwitch :model-value="settings.ddnsEnabled" @change="emit('toggle-ddns')" />
    </UiFormField>
    <UiFormField :label="`远程访问通道 · ${settings.remoteAccess ? '可用' : '内网限定'}`" inline>
      <UiSwitch :model-value="settings.remoteAccess" @change="emit('toggle-remote')" />
    </UiFormField>
    <UiFormField label="DNS 配置">
      <UiSelect v-model="settings.dnsProfile" :options="dnsProfileOptions" />
    </UiFormField>
    <div class="system-settings__metric">
      <Globe2 :size="17" />
      <p>{{ settings.ddnsEnabled ? '公网域名健康，证书 28 天后自动续签。' : 'DDNS 已暂停，仅保留局域网访问。' }}</p>
    </div>
  </div>
</template>
