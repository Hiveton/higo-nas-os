<script setup lang="ts">
import { ArchiveRestore } from 'lucide-vue-next';
import { UiButton, UiFormField, UiSelect, UiSwitch } from '../../ui';

defineProps<{
  settings: { systemBackup: boolean; backupTarget: string };
  backupTargetOptions: { value: string; label: string }[];
}>();

const emit = defineEmits<{
  (e: 'toggle-backup'): void;
  (e: 'create-backup'): void;
}>();
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`系统配置快照 · ${settings.systemBackup ? '每日' : '手动'}`" inline>
      <UiSwitch :model-value="settings.systemBackup" @change="emit('toggle-backup')" />
    </UiFormField>
    <UiFormField label="备份目标">
      <UiSelect v-model="settings.backupTarget" :options="backupTargetOptions" />
    </UiFormField>
    <div class="system-settings__metric">
      <ArchiveRestore :size="17" />
      <p>{{ settings.backupTarget }}：备份系统设置、权限策略、模型路由和通知规则。</p>
    </div>
    <UiButton variant="soft" size="sm" :icon-left="ArchiveRestore" @click="emit('create-backup')">立即创建系统备份</UiButton>
  </div>
</template>
