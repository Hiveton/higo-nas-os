<script setup lang="ts">
import { Server } from 'lucide-vue-next';
import { UiFormField, UiSwitch } from '../../ui';

defineProps<{
  settings: { localAi: boolean; cloudAi: boolean; privateEndpoint: boolean };
}>();

const emit = defineEmits<{
  (e: 'toggle-local'): void;
  (e: 'toggle-cloud'): void;
  (e: 'toggle-private'): void;
}>();
</script>

<template>
  <div class="system-settings__panel">
    <UiFormField :label="`本地 AI 索引与基础理解 · ${settings.localAi ? '运行中' : '暂停'}`" inline>
      <UiSwitch :model-value="settings.localAi" @change="emit('toggle-local')" />
    </UiFormField>
    <UiFormField :label="`云端复杂推理增强 · ${settings.cloudAi ? '允许' : '禁止'}`" inline>
      <UiSwitch :model-value="settings.cloudAi" @change="emit('toggle-cloud')" />
    </UiFormField>
    <UiFormField :label="`私有模型端点 · ${settings.privateEndpoint ? '已接管' : '未接管'}`" inline>
      <UiSwitch :model-value="settings.privateEndpoint" @change="emit('toggle-private')" />
    </UiFormField>
    <div class="system-settings__metric">
      <Server :size="17" />
      <p>{{ settings.localAi ? '本地模型负责隐私索引和基础问答。' : '本地 AI 已暂停，文件访问不受影响。' }}</p>
    </div>
  </div>
</template>
