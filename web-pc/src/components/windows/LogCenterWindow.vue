<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue';
import {
  Activity,
  FileClock,
  Inbox,
  RefreshCw,
  Search,
  ServerCog,
  ShieldCheck,
} from 'lucide-vue-next';
import {
  UiButton,
  UiEmptyState,
  UiInput,
  UiNavItem,
  UiNavRail,
  UiToolbar,
  UiWindowPage,
} from '../ui';
import { logsStore, type LogEntry, type LogKind } from '../../stores/logs';

type Filter = 'all' | LogKind;

const filter = ref<Filter>('all');
const query = ref('');

const FILTERS: { key: Filter; label: string; icon: Component }[] = [
  { key: 'all', label: '全部', icon: Inbox },
  { key: 'system', label: '系统日志', icon: ServerCog },
  { key: 'activity', label: '操作记录', icon: Activity },
  { key: 'audit', label: '治理审计', icon: ShieldCheck },
];

const visible = computed<LogEntry[]>(() => {
  const kw = query.value.trim().toLowerCase();
  return logsStore.entries.value.filter((e) => {
    if (filter.value !== 'all' && e.kind !== filter.value) return false;
    if (!kw) return true;
    return `${e.source} ${e.message} ${e.actor ?? ''}`.toLowerCase().includes(kw);
  });
});

function fmtTime(at: string): string {
  if (!at) return '—';
  const d = new Date(at);
  if (Number.isNaN(d.getTime())) return at;
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

onMounted(() => void logsStore.load());
</script>

<template>
  <UiWindowPage
    layout="master-detail"
    :icon="FileClock"
    title="日志中心"
    :subtitle="`${logsStore.counts.value.all} 条记录`"
  >
    <template #nav>
      <UiNavRail title="日志" subtitle="系统 · 操作 · 审计">
        <UiNavItem
          v-for="f in FILTERS"
          :key="f.key"
          :icon="f.icon"
          :label="f.label"
          :badge="(logsStore.counts.value as Record<string, number>)[f.key] || undefined"
          :active="filter === f.key"
          @select="filter = f.key"
        />
      </UiNavRail>
    </template>

    <template #toolbar>
      <UiToolbar>
        <template #start>
          <UiInput v-model="query" size="sm" :prefix-icon="Search" placeholder="搜索来源、内容、操作者" />
        </template>
        <template #end>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" :disabled="logsStore.loading.value" @click="logsStore.load()">刷新</UiButton>
        </template>
      </UiToolbar>
    </template>

    <section class="logs" aria-label="日志列表">
      <article
        v-for="entry in visible"
        :key="entry.id"
        class="logs__row"
        :class="`logs__row--${entry.level}`"
      >
        <span class="logs__level">{{ entry.level.toUpperCase() }}</span>
        <span class="logs__main">
          <strong>{{ entry.message }}</strong>
          <small>{{ entry.source }}<template v-if="entry.actor"> · {{ entry.actor }}</template></small>
        </span>
        <span class="logs__time">{{ fmtTime(entry.at) }}</span>
      </article>

      <UiEmptyState
        v-if="visible.length === 0"
        :icon="FileClock"
        :title="logsStore.loading.value ? '正在加载日志…' : '暂无日志记录'"
        description="系统事件、用户操作和治理审计会在这里统一汇总。"
      />
    </section>
  </UiWindowPage>
</template>

<style scoped>
.logs {
  display: grid;
  gap: 4px;
  align-content: start;
}

.logs__row {
  display: grid;
  grid-template-columns: 56px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  padding: 9px 12px;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.logs__level {
  padding: 3px 0;
  color: var(--text-soft);
  font-size: 10px;
  font-weight: var(--fw-bold);
  text-align: center;
  letter-spacing: 0.04em;
  background: rgba(var(--surface-rgb), 0.82);
  border-radius: var(--radius-pill);
}
.logs__row--warn .logs__level {
  color: var(--ink-orange);
  background: var(--accent-orange-soft);
}
.logs__row--error .logs__level {
  color: var(--accent-red);
  background: var(--accent-red-soft);
}
.logs__row--audit .logs__level {
  color: var(--accent);
  background: var(--accent-soft);
}

.logs__main {
  display: grid;
  gap: 2px;
  min-width: 0;
}
.logs__main strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.logs__main small {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}

.logs__time {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  white-space: nowrap;
}
</style>
