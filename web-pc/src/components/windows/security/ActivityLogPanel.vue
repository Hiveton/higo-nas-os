<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Download, ListChecks, Trash2 } from 'lucide-vue-next';
import { activityStore } from '../../../stores/activity';
import type { ActivityEntry } from '../../../api/types';
import { UiBadge, UiButton, UiEmptyState, UiInput, UiSegmented, useConfirm, useToast } from '../../ui';
import type { UiTone } from '../../ui';

type TypeFilter = 'all' | 'page' | 'action';

const confirm = useConfirm();
const toast = useToast();

const typeFilter = ref<TypeFilter>('all');
const search = ref('');

const typeFilterOptions: { value: TypeFilter; label: string }[] = [
  { value: 'all', label: '全部' },
  { value: 'page', label: '页面访问' },
  { value: 'action', label: '操作' },
];

const actionLabels: Record<string, string> = {
  visit: '页面访问',
  'open-window': '打开窗口',
  'close-window': '关闭窗口',
  'launch-utility': '启动',
  'ui-action': '操作',
  'cancel-task': '取消任务',
};

function actionLabel(action: string): string {
  return actionLabels[action] ?? action;
}

function typeTone(type: ActivityEntry['type']): UiTone {
  return type === 'page' ? 'info' : 'primary';
}

function typeLabel(type: ActivityEntry['type']): string {
  return type === 'page' ? '页面' : '操作';
}

function formatTime(at?: string): string {
  if (!at) return '—';
  const date = new Date(at);
  if (Number.isNaN(date.getTime())) return at;
  return date.toLocaleString('zh-CN');
}

const filteredEntries = computed(() => {
  const term = search.value.trim().toLowerCase();
  return activityStore.entries.value.filter((entry) => {
    if (typeFilter.value !== 'all' && entry.type !== typeFilter.value) return false;
    if (!term) return true;
    return [entry.action, entry.target, entry.detail, entry.category]
      .filter(Boolean)
      .some((field) => String(field).toLowerCase().includes(term));
  });
});

function exportEntries() {
  const payload = JSON.stringify(activityStore.entries.value, null, 2);
  const blob = new Blob([payload], { type: 'application/json' });
  const url = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = url;
  anchor.download = `activity-${new Date().toISOString().slice(0, 10)}.json`;
  anchor.click();
  URL.revokeObjectURL(url);
  toast.success('操作记录已导出。');
}

async function clearEntries() {
  const ok = await confirm({
    title: '清除操作记录？',
    message: '本地与后端的操作记录都会被清空，且无法恢复。',
    confirmLabel: '清除',
    tone: 'danger',
  });
  if (!ok) return;
  await activityStore.clear();
  toast.success('操作记录已清除。');
}

onMounted(() => {
  void activityStore.hydrate(true);
});
</script>

<template>
  <div class="activity-log">
    <header class="activity-log__bar">
      <UiSegmented
        :model-value="typeFilter"
        :options="typeFilterOptions"
        size="sm"
        aria-label="按类型筛选操作记录"
        @change="(v) => (typeFilter = v as TypeFilter)"
      />
      <UiInput
        v-model="search"
        class="activity-log__search"
        size="sm"
        placeholder="搜索动作、目标或详情"
        clearable
        aria-label="搜索操作记录"
      />
      <div class="activity-log__actions">
        <UiButton variant="soft" size="sm" :icon-left="Download" @click="exportEntries">导出</UiButton>
        <UiButton
          variant="soft"
          tone="danger"
          size="sm"
          :icon-left="Trash2"
          @click="clearEntries"
        >
          清除记录
        </UiButton>
      </div>
    </header>

    <UiEmptyState
      v-if="filteredEntries.length === 0"
      class="activity-log__empty"
      :icon="ListChecks"
      title="暂无操作记录"
      description="页面访问与有意义的操作会自动记录在这里。"
      compact
    />

    <ol v-else class="activity-log__list">
      <li v-for="(entry, index) in filteredEntries" :key="entry.id ?? `${entry.at}-${index}`" class="activity-log__item">
        <time class="activity-log__time">{{ formatTime(entry.at) }}</time>
        <UiBadge class="activity-log__type" :tone="typeTone(entry.type)" size="sm">{{ typeLabel(entry.type) }}</UiBadge>
        <div class="activity-log__body">
          <strong class="activity-log__action">{{ actionLabel(entry.action) }}</strong>
          <small class="activity-log__meta">
            <span v-if="entry.category">{{ entry.category }}</span>
            <span v-if="entry.target">{{ entry.target }}</span>
          </small>
          <p v-if="entry.detail" class="activity-log__detail">{{ entry.detail }}</p>
        </div>
      </li>
    </ol>
  </div>
</template>

<style scoped>
.activity-log {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 10px;
  min-height: 0;
  height: 100%;
}

.activity-log__bar {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 8px;
}

.activity-log__search {
  flex: 1 1 180px;
  min-width: 140px;
}

.activity-log__actions {
  display: flex;
  gap: 6px;
  margin-left: auto;
}

.activity-log__empty {
  align-self: center;
}

.activity-log__list {
  display: grid;
  align-content: start;
  gap: 8px;
  margin: 0;
  padding: 0;
  min-height: 0;
  overflow: auto;
  list-style: none;
}

.activity-log__item {
  display: grid;
  grid-template-columns: auto auto minmax(0, 1fr);
  align-items: start;
  gap: 9px;
  padding: 9px 11px;
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.activity-log__time {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  font-variant-numeric: tabular-nums;
  white-space: nowrap;
}

.activity-log__type {
  align-self: center;
}

.activity-log__body {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.activity-log__action {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.activity-log__meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}

.activity-log__detail {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
}

@media (max-width: 860px) {
  .activity-log__item {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .activity-log__time {
    grid-column: 1 / -1;
  }
}
</style>
