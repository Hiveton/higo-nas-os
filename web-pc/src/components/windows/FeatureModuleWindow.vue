<script setup lang="ts">
import { computed, ref } from 'vue';
import { CheckCircle2, FileText, ShieldAlert } from 'lucide-vue-next';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { nasFeatures, type NasFeatureKey } from '../../data/nasFeatures';
import { UiButton } from '../ui';

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
  <div class="feature-module">
    <section class="feature-module__hero">
      <div>
        <p>{{ subtitle }}</p>
        <h3>{{ title }}</h3>
      </div>
      <strong>{{ features.length }} 类 / {{ actionCount }} 项</strong>
    </section>

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
  </div>
</template>

<style scoped>
.feature-module {
  display: grid;
  grid-template-rows: auto auto auto auto auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
  overflow: auto;
}

.feature-module__hero,
.feature-module__state,
.feature-module__risk {
  background: rgba(var(--surface-rgb),  0.55);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.feature-module__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 76px;
  padding: 14px;
}

.feature-module__hero p,
.feature-module__hero h3 {
  margin: 0;
}

.feature-module__hero p {
  color: var(--text-muted);
  font-size: 12px;
}

.feature-module__hero h3 {
  margin-top: 4px;
  color: var(--text-strong);
  font-size: 18px;
}

.feature-module__hero strong {
  color: var(--accent);
  font-size: 12px;
  white-space: nowrap;
}

.feature-module__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.feature-module__state,
.feature-module__risk {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 9px 10px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.feature-module__state {
  color: var(--accent-green);
}
</style>

