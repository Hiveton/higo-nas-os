<script setup lang="ts">
import { Bell } from 'lucide-vue-next';
import { UiFormField, UiSwitch } from '../../ui';

defineProps<{
  settings: { backupNotice: boolean; securityNotice: boolean; lifeNotice: boolean };
  enabledNoticeCount: number;
}>();

const emit = defineEmits<{
  (e: 'toggle-backup'): void;
  (e: 'toggle-security'): void;
  (e: 'toggle-life'): void;
}>();
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`备份失败 / 完整性提醒 · ${settings.backupNotice ? '推送' : '静默'}`" inline>
      <UiSwitch :model-value="settings.backupNotice" @change="emit('toggle-backup')" />
    </UiFormField>
    <UiFormField :label="`权限风险 / 硬盘异常 · ${settings.securityNotice ? '推送' : '静默'}`" inline>
      <UiSwitch :model-value="settings.securityNotice" @change="emit('toggle-security')" />
    </UiFormField>
    <UiFormField :label="`证件、保修、生活提醒 · ${settings.lifeNotice ? '推送' : '静默'}`" inline>
      <UiSwitch :model-value="settings.lifeNotice" @change="emit('toggle-life')" />
    </UiFormField>
    <div class="system-settings__metric">
      <Bell :size="17" />
      <p>{{ enabledNoticeCount }} 类通知已开启，通知中心会聚合系统、备份、Agent 和生活提醒。</p>
    </div>
  </div>
</template>
