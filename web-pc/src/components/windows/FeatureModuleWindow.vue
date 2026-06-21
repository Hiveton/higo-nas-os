<script setup lang="ts">
import { computed, ref } from 'vue';
import { CheckCircle2, FileText, Layers, ShieldAlert } from 'lucide-vue-next';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { nasFeatures, type NasFeatureKey } from '../../data/nasFeatures';
import { UiButton, UiBadge, UiWindowPage } from '../ui';

const props = defineProps<{
  moduleKey: NasFeatureKey;
  title: string;
  subtitle: string;
}>();

const activeAction = ref('等待选择功能');
const features = computed(() => nasFeatures[props.moduleKey] ?? []);
const actionCount = computed(() => features.value.reduce((sum, feature) => sum + feature.actions.length, 0));

function selectAction(label: string) {
  activeAction.value = `${label} 已进入界面流程；后端阶段接入真实 API、任务状态和审计。`;
}
</script>

<template>
  <UiWindowPage layout="stack" :icon="Layers" :title="title" :subtitle="subtitle">
    <template #actions>
      <UiBadge tone="primary" variant="soft" size="sm">{{ features.length }} 类 / {{ actionCount }} 项</UiBadge>
    </template>

    <NasFeaturePanel :modules="[moduleKey]" />

    <section class="feature-module__actions" aria-label="功能操作">
      <UiButton
        v-for="feature in features"
        :key="feature.title"
        variant="soft"
        tone="primary"
        size="sm"
        :icon-left="FileText"
        @click="selectAction(feature.title)"
      >
        {{ feature.title }}
      </UiButton>
    </section>

    <section class="feature-module__state">
      <CheckCircle2 :size="15" />
      <span>{{ activeAction }}</span>
    </section>

    <section class="feature-module__risk">
      <ShieldAlert :size="15" />
      <span>当前为前端界面补齐。涉及格式化、删除、权限、外链、虚拟机直通等高风险动作，后端接入时必须加入二次确认、任务回滚和审计。</span>
    </section>
  </UiWindowPage>
</template>

<style scoped>
.feature-module__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.feature-module__state,
.feature-module__risk {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  min-height: 36px;
  padding: var(--space-2) var(--space-3);
  background: rgba(var(--surface-rgb), 0.55);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.45;
}

.feature-module__state {
  color: var(--ink-green);
}
</style>
