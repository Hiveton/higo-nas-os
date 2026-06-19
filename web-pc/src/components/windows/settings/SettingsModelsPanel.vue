<script setup lang="ts">
import { BrainCircuit } from 'lucide-vue-next';
import { UiBadge, UiButton, UiEmptyState, UiFormField, UiInput, UiSegmented, UiSelect, UiSwitch } from '../../ui';
import { computed } from 'vue';
import type { AiProvider, AiProviderInput, AiProviderKind, AiProviderPurpose } from '../../../api/types';

const props = defineProps<{
  settings: { modelStrategy: string; modelProvider: string; taskRouting: boolean; analysisLevel: string };
  modelStrategies: string[];
  modelProviderOptions: { value: string; label: string }[];
  modelSummary: string;
  analysisLevelOptions: { value: string; label: string }[];
  providers: AiProvider[];
  providerForm: AiProviderInput;
  providerBusy: boolean;
  providerNotice: string;
  providerKinds: Array<{ value: AiProviderKind; label: string }>;
  providerKindLabel: (kind: AiProviderKind) => string;
}>();

const emit = defineEmits<{
  (e: 'set-strategy', strategy: string): void;
  (e: 'toggle-task-routing'): void;
  (e: 'set-analysis-level', level: string): void;
  (e: 'submit-provider'): void;
  (e: 'test-provider', provider: AiProvider): void;
  (e: 'set-default-provider', provider: AiProvider): void;
  (e: 'remove-provider', provider: AiProvider): void;
}>();

const strategyOptions = computed(() => props.modelStrategies.map((s) => ({ value: s, label: s })));
const kindOptions = computed(() => props.providerKinds.map((k) => ({ value: k.value, label: k.label })));

const purposeOptions: { value: AiProviderPurpose; label: string }[] = [
  { value: 'chat', label: '对话 (chat)' },
  { value: 'embedding', label: '嵌入 (embedding)' },
  { value: 'vision', label: '视觉 (vision)' },
  { value: 'asr', label: '语音转写 (asr)' },
];
const purposeLabel = (purpose?: AiProviderPurpose) =>
  purposeOptions.find((p) => p.value === (purpose ?? 'chat'))?.label ?? purpose ?? 'chat';
</script>

<template>
  <div class="system-settings__panel">
    <UiSegmented
      :model-value="settings.modelStrategy"
      :options="strategyOptions"
      aria-label="模型策略选择"
      @change="(v) => emit('set-strategy', String(v))"
    />
    <UiFormField label="默认模型">
      <UiSelect v-model="settings.modelProvider" :options="modelProviderOptions" />
    </UiFormField>
    <UiFormField label="按任务类型路由 OCR / 摘要 / Agent 规划" inline>
      <UiSwitch :model-value="settings.taskRouting" @change="emit('toggle-task-routing')" />
    </UiFormField>
    <div class="system-settings__metric">
      <BrainCircuit :size="17" />
      <p>{{ modelSummary }}</p>
    </div>

    <UiFormField label="AI 分析等级" hint="控制全局后台 AI 分析的深度：关闭 / 基础（本地元数据）/ 标准（+模型摘要标签）/ 深度（+视觉、向量、人脸聚类）">
      <UiSegmented
        :model-value="settings.analysisLevel"
        :options="analysisLevelOptions"
        aria-label="AI 分析等级选择"
        @change="(v) => emit('set-analysis-level', String(v))"
      />
    </UiFormField>

    <div class="provider-binding">
      <div class="provider-binding__head">
        <strong>模型供应商绑定</strong>
        <span>{{ providerNotice }}</span>
      </div>

      <ul v-if="providers.length" class="provider-list">
        <li v-for="provider in providers" :key="provider.id" class="provider-list__item">
          <div class="provider-list__info">
            <strong>
              {{ provider.name }}
              <UiBadge v-if="provider.isDefault" tone="success" size="sm">默认</UiBadge>
            </strong>
            <small>
              {{ purposeLabel(provider.purpose) }} · {{ providerKindLabel(provider.kind) }} · {{ provider.model }}
              <template v-if="provider.hasKey"> · 密钥 {{ provider.keyHint }}</template>
            </small>
          </div>
          <div class="provider-list__actions">
            <UiButton variant="ghost" size="sm" :disabled="providerBusy" @click="emit('test-provider', provider)">测试</UiButton>
            <UiButton v-if="!provider.isDefault" variant="ghost" size="sm" @click="emit('set-default-provider', provider)">设为默认</UiButton>
            <UiButton variant="ghost" tone="danger" size="sm" @click="emit('remove-provider', provider)">删除</UiButton>
          </div>
        </li>
      </ul>
      <UiEmptyState
        v-else
        title="尚未绑定任何模型"
        description="添加一个 OpenAI / Anthropic / Gemini 端点即可启用真实问答。"
        compact
      />

      <form class="provider-form" @submit.prevent="emit('submit-provider')">
        <UiFormField label="名称">
          <UiInput v-model="providerForm.name" type="text" placeholder="如：公司 GPT-4o" />
        </UiFormField>
        <UiFormField label="类型">
          <UiSelect v-model="providerForm.kind" :options="kindOptions" />
        </UiFormField>
        <UiFormField label="用途">
          <UiSelect v-model="providerForm.purpose" :options="purposeOptions" />
        </UiFormField>
        <UiFormField label="Base URL（可选）">
          <UiInput v-model="providerForm.baseUrl" type="text" placeholder="如：http://localhost:11434/v1" />
        </UiFormField>
        <UiFormField label="模型 ID">
          <UiInput v-model="providerForm.model" type="text" placeholder="如：gpt-4o-mini / claude-3-5-sonnet / gemini-1.5-pro" />
        </UiFormField>
        <UiFormField label="API Key">
          <UiInput v-model="providerForm.apiKey" type="password" placeholder="本地模型可留空" autocomplete="off" />
        </UiFormField>
        <UiButton type="submit" :loading="providerBusy" :disabled="providerBusy">添加供应商</UiButton>
      </form>
    </div>
  </div>
</template>
