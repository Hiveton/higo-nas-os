<script setup lang="ts">
import { computed } from 'vue';
import { RefreshCw, X } from 'lucide-vue-next';
import type { AiAnalysisRecord } from '../../../api/types';
import { UiBadge, UiButton, UiSpinner } from '../../ui';

const props = defineProps<{
  record: AiAnalysisRecord | null;
  loading?: boolean;
}>();

const emit = defineEmits<{ close: []; reanalyze: [key: string] }>();

const r = computed(() => props.record);
const result = computed(() => props.record?.result ?? null);
const techMeta = computed(() => {
  const tm = result.value?.techMeta;
  if (!tm) return [] as Array<[string, string]>;
  return Object.entries(tm).map(([k, v]) => [k, String(v)] as [string, string]);
});

const stateLabel: Record<string, string> = {
  pending: '待分析',
  analyzing: '分析中',
  done: '已完成',
  failed: '失败',
  skipped: '已跳过',
};

function fmt(iso?: string) {
  if (!iso) return '—';
  const d = new Date(iso);
  return Number.isNaN(d.getTime()) ? '—' : d.toLocaleString('zh-CN');
}
</script>

<template>
  <div v-if="record" class="drawer">
    <div class="drawer__scrim" @click="emit('close')" />
    <aside class="drawer__panel u-glass" role="dialog" aria-label="分析详情">
      <header class="drawer__head">
        <div class="drawer__title">
          <strong>{{ record.title }}</strong>
          <span v-if="record.sourcePath">{{ record.sourcePath }}</span>
        </div>
        <button class="drawer__close" type="button" aria-label="关闭" @click="emit('close')">
          <X :size="16" />
        </button>
      </header>

      <div v-if="loading" class="drawer__loading"><UiSpinner /></div>

      <div v-else class="drawer__body">
        <div class="drawer__row">
          <UiBadge tone="neutral" variant="soft" size="sm">{{ record.domain }}</UiBadge>
          <UiBadge :tone="record.state === 'failed' ? 'danger' : record.state === 'done' ? 'success' : 'primary'" variant="soft" size="sm">
            {{ stateLabel[record.state] ?? record.state }}
          </UiBadge>
          <UiBadge tone="neutral" variant="soft" size="sm">等级 {{ record.level }}</UiBadge>
          <UiBadge v-if="record.attempts > 1" tone="warning" variant="soft" size="sm">尝试 {{ record.attempts }} 次</UiBadge>
          <UiBadge v-if="result?.embedded" tone="success" variant="soft" size="sm">已建索引</UiBadge>
        </div>

        <p v-if="record.error" class="drawer__error">{{ record.error }}</p>

        <dl class="drawer__fields">
          <template v-if="result?.summary">
            <dt>摘要</dt><dd>{{ result.summary }}</dd>
          </template>
          <template v-if="result?.caption">
            <dt>视觉描述</dt><dd>{{ result.caption }}</dd>
          </template>
          <template v-if="result?.people?.length">
            <dt>人物</dt><dd>{{ result.people.join('、') }}</dd>
          </template>
          <template v-if="result?.place">
            <dt>地点</dt><dd>{{ result.place }}</dd>
          </template>
          <template v-if="result?.device">
            <dt>设备</dt><dd>{{ result.device }}</dd>
          </template>
          <template v-if="result?.tags?.length">
            <dt>标签</dt>
            <dd class="drawer__tags">
              <UiBadge v-for="t in result.tags" :key="t" tone="neutral" variant="soft" size="sm">{{ t }}</UiBadge>
            </dd>
          </template>
          <template v-for="[k, v] in techMeta" :key="k">
            <dt>{{ k }}</dt><dd>{{ v }}</dd>
          </template>
          <dt>更新时间</dt><dd>{{ fmt(record.updatedAt) }}</dd>
          <template v-if="record.analyzedAt">
            <dt>分析完成</dt><dd>{{ fmt(record.analyzedAt) }}</dd>
          </template>
        </dl>

        <section v-if="result?.transcript" class="drawer__transcript">
          <span class="drawer__transcript-title">语音转写 / 字幕</span>
          <p>{{ result.transcript }}</p>
        </section>
      </div>

      <footer class="drawer__foot">
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="emit('reanalyze', record.key)">
          重新分析此项
        </UiButton>
      </footer>
    </aside>
  </div>
</template>

<style scoped>
.drawer {
  position: absolute;
  inset: 0;
  z-index: var(--z-modal);
  display: flex;
  justify-content: flex-end;
}
.drawer__scrim {
  position: absolute;
  inset: 0;
  background: rgba(15, 32, 51, 0.28);
}
.drawer__panel {
  position: relative;
  display: flex;
  flex-direction: column;
  width: min(420px, 86%);
  height: 100%;
  border-radius: var(--radius-window) 0 0 var(--radius-window);
  box-shadow: var(--elevation-3);
  animation: drawer-in var(--duration-base) var(--ease-standard);
}
@keyframes drawer-in {
  from { transform: translateX(16px); opacity: 0; }
  to { transform: translateX(0); opacity: 1; }
}
.drawer__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-4);
  border-bottom: 1px solid var(--border);
}
.drawer__title {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.drawer__title strong {
  color: var(--text-strong);
  font-size: var(--fs-md);
}
.drawer__title span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.drawer__close {
  flex-shrink: 0;
  padding: 4px;
  color: var(--text-muted);
  background: transparent;
  border: 0;
  border-radius: var(--radius-control);
  cursor: pointer;
}
.drawer__close:hover { background: var(--accent-soft); }
.drawer__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-8);
}
.drawer__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
  padding: var(--space-4);
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
.drawer__row {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.drawer__error {
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-card);
  background: var(--accent-red-soft);
  color: var(--accent-red);
  font-size: var(--fs-xs);
}
.drawer__fields {
  display: grid;
  grid-template-columns: 72px minmax(0, 1fr);
  gap: var(--space-2) var(--space-3);
  margin: 0;
}
.drawer__fields dt {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.drawer__fields dd {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-xs);
  line-height: 1.5;
  word-break: break-word;
}
.drawer__tags {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}
.drawer__transcript {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.drawer__transcript-title {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
.drawer__transcript p {
  margin: 0;
  max-height: 200px;
  overflow-y: auto;
  white-space: pre-wrap;
  color: var(--text);
  font-size: var(--fs-xs);
  line-height: 1.6;
}
.drawer__foot {
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--border);
}
</style>
