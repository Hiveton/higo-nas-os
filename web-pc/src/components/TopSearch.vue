<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, watch, type Component } from 'vue';
import {
  AppWindow,
  Bot,
  Cpu,
  FileText,
  Image,
  Music,
  RefreshCw,
  Search,
  Settings,
  ShieldCheck,
  Video,
} from 'lucide-vue-next';
import { apiClient } from '../api/client';
import { monitoringStore } from '../stores/monitoring';
import { desktopWindows } from '../data/higoos';
import type { FileRow } from '../api/types';

// Each emitted action is a prefixed string parsed by App.handleTopbarAction:
//   app:<windowId> | ask:<query> | reindex | inspect
const emit = defineEmits<{ action: [action: string] }>();

type Row = {
  key: string;
  label: string;
  sub?: string;
  badge?: string;
  icon: Component;
  action: string;
};

const query = ref('');
const focused = ref(false);
const loading = ref(false);
const activeIndex = ref(0);
const fileItems = ref<FileRow[]>([]);
const inputRef = ref<HTMLInputElement | null>(null);
const rootRef = ref<HTMLElement | null>(null);

const suggestions = [
  '找上个月客户 A 的最终合同',
  '检查哪些文件没有异地备份',
  '整理下载目录里的发票',
];

const domainMeta: Record<string, { icon: Component; badge: string }> = {
  file: { icon: FileText, badge: '文件' },
  photo: { icon: Image, badge: '照片' },
  video: { icon: Video, badge: '视频' },
  music: { icon: Music, badge: '音乐' },
};

// Static settings/actions registry, matched by keyword.
const actionRegistry: Array<{ label: string; keywords: string[]; icon: Component; action: string }> = [
  { label: '模型策略设置', keywords: ['模型', 'model', '策略', 'ai', '供应商'], icon: Settings, action: 'app:system-settings' },
  { label: '重新索引文件', keywords: ['索引', 'index', '重建', '检索'], icon: RefreshCw, action: 'reindex' },
  { label: '运行安全巡检', keywords: ['安全', '巡检', '风险', '分享', 'security'], icon: ShieldCheck, action: 'inspect' },
];

const deviceKeywords = ['cpu', '内存', 'memory', '网络', 'network', '磁盘', 'disk', '设备', '状态'];

let debounce: ReturnType<typeof setTimeout> | undefined;

watch(query, (q) => {
  fileItems.value = [];
  if (debounce) clearTimeout(debounce);
  const trimmed = q.trim();
  if (trimmed.length < 2) {
    loading.value = false;
    return;
  }
  loading.value = true;
  debounce = setTimeout(() => runSearch(trimmed), 250);
});

// indexEnabled is fetched lazily on first search; false means semantic search is
// running on the keyword fallback (no pgvector DB / embedding model configured).
const indexEnabled = ref<boolean | null>(null);

async function runSearch(q: string) {
  if (indexEnabled.value === null) {
    void apiClient.aiAnalysis
      .getStatus()
      .then((s) => {
        indexEnabled.value = s.indexEnabled;
      })
      .catch(() => {
        indexEnabled.value = null;
      });
  }
  try {
    const result = await apiClient.assistant.semanticSearch({ query: q, limit: 8 });
    if (query.value.trim() === q) {
      fileItems.value = result.items ?? [];
    }
  } catch {
    fileItems.value = [];
  } finally {
    if (query.value.trim() === q) loading.value = false;
  }
}

const appRows = computed<Row[]>(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return [];
  return desktopWindows
    .filter((w) => w.title.toLowerCase().includes(q) || w.id.includes(q))
    .slice(0, 4)
    .map((w) => ({ key: `app-${w.id}`, label: `打开 ${w.title}`, sub: w.subtitle, icon: AppWindow, action: `app:${w.id}` }));
});

const deviceRows = computed<Row[]>(() => {
  const q = query.value.trim().toLowerCase();
  if (!q || !deviceKeywords.some((k) => q.includes(k))) return [];
  return monitoringStore.metrics.value
    .filter((m) => ['cpu', 'memory', 'network', 'disk'].includes(m.key ?? ''))
    .map((m) => ({
      key: `dev-${m.key}`,
      label: `${m.label} ${m.value}${m.unit ?? ''}`,
      sub: '设备状态 · 打开监控',
      icon: Cpu,
      action: 'app:device-monitor',
    }));
});

const actionRows = computed<Row[]>(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return [];
  return actionRegistry
    .filter((a) => a.label.toLowerCase().includes(q) || a.keywords.some((k) => q.includes(k.toLowerCase())))
    .map((a) => ({ key: `act-${a.action}`, label: a.label, sub: '设置 / 动作', icon: a.icon, action: a.action }));
});

const fileRows = computed<Row[]>(() =>
  fileItems.value.slice(0, 8).map((it, i) => {
    const meta = domainMeta[it.type] ?? domainMeta.file;
    return {
      key: `file-${it.id ?? i}`,
      label: it.name || it.path || '未命名',
      sub: it.path || it.aiSummary,
      badge: meta.badge,
      icon: meta.icon,
      action: 'app:file-manager',
    };
  }),
);

const askRow = computed<Row | null>(() => {
  const q = query.value.trim();
  if (!q) return null;
  return { key: 'ask', label: `问 AI 助手：${q}`, sub: '让 AI 检索并回答', icon: Bot, action: `ask:${q}` };
});

// Flat, ordered list of selectable rows for keyboard navigation.
const rows = computed<Row[]>(() => {
  const out: Row[] = [];
  if (askRow.value) out.push(askRow.value);
  out.push(...appRows.value, ...deviceRows.value, ...actionRows.value, ...fileRows.value);
  return out;
});

const showPanel = computed(() => focused.value);
const hasQuery = computed(() => query.value.trim().length > 0);

watch(rows, () => {
  if (activeIndex.value >= rows.value.length) activeIndex.value = 0;
});

function select(row: Row) {
  emit('action', row.action);
  query.value = '';
  fileItems.value = [];
  focused.value = false;
  inputRef.value?.blur();
}

function selectSuggestion(text: string) {
  emit('action', `ask:${text}`);
  query.value = '';
  focused.value = false;
  inputRef.value?.blur();
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') {
    focused.value = false;
    inputRef.value?.blur();
    return;
  }
  if (!rows.value.length) return;
  if (e.key === 'ArrowDown') {
    e.preventDefault();
    activeIndex.value = (activeIndex.value + 1) % rows.value.length;
  } else if (e.key === 'ArrowUp') {
    e.preventDefault();
    activeIndex.value = (activeIndex.value - 1 + rows.value.length) % rows.value.length;
  } else if (e.key === 'Enter') {
    e.preventDefault();
    const row = rows.value[activeIndex.value] ?? rows.value[0];
    if (row) select(row);
  }
}

function onGlobalKeydown(e: KeyboardEvent) {
  if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === 'k') {
    e.preventDefault();
    focused.value = true;
    inputRef.value?.focus();
  }
}

function onPointerDown(e: PointerEvent) {
  if (!rootRef.value?.contains(e.target as Node)) focused.value = false;
}

onMounted(() => {
  window.addEventListener('keydown', onGlobalKeydown);
  document.addEventListener('pointerdown', onPointerDown);
});
onUnmounted(() => {
  window.removeEventListener('keydown', onGlobalKeydown);
  document.removeEventListener('pointerdown', onPointerDown);
});

defineExpose({ focus: () => inputRef.value?.focus() });
</script>

<template>
  <label ref="rootRef" class="topsearch">
    <Search :size="17" aria-hidden="true" />
    <span class="sr-only">全局搜索</span>
    <input
      ref="inputRef"
      v-model="query"
      type="search"
      placeholder="搜索文件、照片、Agent、设备状态"
      @focus="focused = true"
      @keydown="onKeydown"
    />
    <kbd>⌘K</kbd>

    <div v-if="showPanel" class="topsearch__panel" @mousedown.prevent>
      <!-- Empty query: suggestions -->
      <template v-if="!hasQuery">
        <p class="topsearch__group">试试这样问</p>
        <button v-for="s in suggestions" :key="s" type="button" class="topsearch__row" @click="selectSuggestion(s)">
          <Bot :size="15" />
          <span class="topsearch__label">{{ s }}</span>
        </button>
      </template>

      <!-- Results -->
      <template v-else>
        <button
          v-for="(row, i) in rows"
          :key="row.key"
          type="button"
          class="topsearch__row"
          :class="{ 'is-active': i === activeIndex }"
          @mouseenter="activeIndex = i"
          @click="select(row)"
        >
          <component :is="row.icon" :size="15" />
          <span class="topsearch__label">{{ row.label }}</span>
          <span v-if="row.badge" class="topsearch__badge">{{ row.badge }}</span>
          <span v-if="row.sub" class="topsearch__sub">{{ row.sub }}</span>
        </button>

        <p v-if="loading" class="topsearch__hint">搜索中…</p>
        <p v-else-if="rows.length <= 1" class="topsearch__hint">回车让 AI 助手回答，或继续输入</p>
        <p v-if="!loading && indexEnabled === false" class="topsearch__hint">
          当前为关键字匹配。启用语义检索需配置数据库与向量模型（系统设置 → 模型策略）。
        </p>
      </template>
    </div>
  </label>
</template>

<style scoped>
.topsearch {
  position: relative;
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  height: 38px;
  padding: 0 10px 0 13px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.62);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82);
}
.topsearch input {
  width: 100%;
  min-width: 0;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
}
.topsearch input::placeholder {
  color: rgba(77, 95, 116, 0.7);
}
.topsearch kbd {
  flex: 0 0 auto;
  min-width: 36px;
  padding: 3px 7px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  font-family: inherit;
  text-align: center;
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}
.topsearch__panel {
  position: absolute;
  top: 46px;
  left: 0;
  z-index: 20;
  display: grid;
  width: min(520px, 86vw);
  gap: 3px;
  max-height: 60vh;
  padding: 10px;
  overflow-y: auto;
  background: rgba(var(--surface-rgb), 0.94);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(20px) saturate(1.2);
}
.topsearch__group {
  margin: 0 0 2px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
}
.topsearch__row {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 34px;
  padding: 0 9px;
  color: var(--text);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: 9px;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard);
}
.topsearch__row:hover {
  background: var(--accent-soft);
}
.topsearch__row.is-active {
  background: rgba(var(--surface-rgb), 0.6);
  box-shadow: inset 0 0 0 1px var(--border);
}
.topsearch__row svg {
  flex: 0 0 auto;
  color: var(--accent);
}
.topsearch__label {
  flex: 0 1 auto;
  overflow: hidden;
  font-size: var(--fs-sm);
  white-space: nowrap;
  text-overflow: ellipsis;
}
.topsearch__badge {
  flex: 0 0 auto;
  padding: 1px 7px;
  color: var(--accent);
  font-size: 10px;
  font-weight: var(--fw-bold);
  background: var(--accent-soft);
  border-radius: var(--radius-pill);
}
.topsearch__sub {
  flex: 1 1 auto;
  overflow: hidden;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  text-align: right;
  white-space: nowrap;
  text-overflow: ellipsis;
}
.topsearch__hint {
  margin: 4px 0 0;
  padding: 0 9px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}
</style>
