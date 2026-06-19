<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Film, FolderOpen, Images, Pause, Play, RefreshCw, ScanSearch, Sparkles } from 'lucide-vue-next';
import type { Component } from 'vue';
import { aiAnalysisStore } from '../../stores/aiAnalysis';
import type { AiAnalysisDomain, AiAnalysisRecord, AiAnalysisState } from '../../api/types';
import {
  UiBadge,
  UiButton,
  UiCard,
  UiDataTable,
  UiEmptyState,
  UiIconButton,
  UiProgressBar,
  UiTabs,
  useToast,
  type Column,
  type TabItem,
  type UiTone,
} from '../ui';

const toast = useToast();

const overview = aiAnalysisStore.overview;
const records = aiAnalysisStore.records;
const recordsPage = aiAnalysisStore.recordsPage;

// UiDataTable expects a mutable rows array; the store exposes records deep-readonly
// (it only reads them), so cast through unknown.
const recordRows = computed(() => [...records.value] as unknown as AiAnalysisRecord[]);

type DomainFilter = '' | AiAnalysisDomain;
const activeDomain = ref<DomainFilter>('');

const domainMeta: Record<AiAnalysisDomain, { label: string; icon: Component }> = {
  media: { label: '相册', icon: Images },
  file: { label: '文件', icon: FolderOpen },
  video: { label: '视频', icon: Film },
};

const levelLabel: Record<string, string> = {
  off: '已关闭',
  basic: '基础',
  standard: '标准',
  deep: '深度',
};

const tabs = computed<TabItem[]>(() => {
  const total = overview.value?.domains.reduce((sum, d) => sum + d.total, 0) ?? 0;
  return [
    { key: '', label: '全部', badge: total || undefined },
    ...(overview.value?.domains ?? []).map((d) => ({
      key: d.domain,
      label: domainMeta[d.domain]?.label ?? d.domain,
      badge: d.total || undefined,
    })),
  ];
});

const paused = computed(() => overview.value?.paused ?? false);
const levelText = computed(() => {
  const level = overview.value?.level ?? '';
  return levelLabel[level] ?? level;
});
const analysisOff = computed(() => (overview.value?.level ?? 'basic') === 'off');

const capabilities = computed(() => {
  const o = overview.value;
  if (!o) return [] as { label: string; ok: boolean }[];
  return [
    { label: '对话模型', ok: o.hasChat },
    { label: '视觉模型', ok: o.hasVision },
    { label: '向量索引', ok: o.hasEmbedding && o.indexEnabled },
    { label: '语音转写', ok: o.hasAsr },
    { label: 'ffmpeg', ok: o.ffmpegAvailable },
  ];
});

const stateMeta: Record<AiAnalysisState, { label: string; tone: UiTone }> = {
  pending: { label: '待分析', tone: 'neutral' },
  analyzing: { label: '分析中', tone: 'primary' },
  done: { label: '已完成', tone: 'success' },
  failed: { label: '失败', tone: 'danger' },
  skipped: { label: '已跳过', tone: 'warning' },
};

const columns: Column<AiAnalysisRecord>[] = [
  { key: 'title', label: '项目', width: '28%' },
  { key: 'state', label: '状态', width: '12%' },
  { key: 'result', label: '分析结果', width: '40%' },
  { key: 'updatedAt', label: '更新时间', width: '14%' },
  { key: 'actions', label: '操作', width: '6%', align: 'right' },
];

const totalPages = computed(() =>
  Math.max(1, Math.ceil((recordsPage.value.total || 0) / (recordsPage.value.pageSize || 50))),
);

function domainStat(domain: AiAnalysisDomain) {
  return overview.value?.domains.find((d) => d.domain === domain);
}

function progressToneFor(percent: number): UiTone {
  if (percent >= 100) return 'success';
  return 'primary';
}

function formatTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function recordSummary(row: AiAnalysisRecord): string {
  if (row.state === 'failed') return row.error || '分析失败';
  const r = row.result;
  if (!r) return '—';
  const parts: string[] = [];
  if (r.summary) parts.push(r.summary);
  else if (r.caption) parts.push(r.caption);
  if (r.people?.length) parts.push(`人物：${r.people.join('、')}`);
  if (r.place) parts.push(`地点：${r.place}`);
  return parts.length ? parts.join(' · ') : '—';
}

async function refreshRecords() {
  const domain = activeDomain.value || undefined;
  await aiAnalysisStore.loadRecords(domain, undefined, recordsPage.value.page);
}

async function changePage(delta: number) {
  const next = recordsPage.value.page + delta;
  if (next < 1 || next > totalPages.value) return;
  await aiAnalysisStore.loadRecords(activeDomain.value || undefined, undefined, next);
}

async function reanalyzeAll() {
  try {
    await aiAnalysisStore.reanalyze({ scope: 'all' });
    toast.success('已触发全部重新分析');
    await refreshRecords();
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function reanalyzeDomain(domain: AiAnalysisDomain) {
  try {
    await aiAnalysisStore.reanalyze({ scope: 'domain', domain });
    toast.success(`已触发「${domainMeta[domain].label}」重新分析`);
    await refreshRecords();
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function reanalyzeItem(row: AiAnalysisRecord) {
  try {
    await aiAnalysisStore.reanalyze({ scope: 'item', itemId: row.key });
    toast.success('已加入重新分析队列');
    await refreshRecords();
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function togglePause() {
  try {
    if (paused.value) {
      await aiAnalysisStore.resume();
      toast.success('已恢复后台分析');
    } else {
      await aiAnalysisStore.pause();
      toast.success('已暂停后台分析');
    }
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function rescan() {
  try {
    await aiAnalysisStore.rescan();
    toast.success('已触发重新扫描');
    await refreshRecords();
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

watch(activeDomain, () => void aiAnalysisStore.loadRecords(activeDomain.value || undefined, undefined, 1));

// Keep the visible page fresh as the live overview advances.
watch(
  () => overview.value?.totalPercent,
  () => {
    if (!aiAnalysisStore.recordsLoading.value) void refreshRecords();
  },
);

onMounted(() => {
  aiAnalysisStore.start();
  void aiAnalysisStore.loadRecords(undefined, undefined, 1);
});
onBeforeUnmount(() => aiAnalysisStore.stop());
</script>

<template>
  <div class="ai-analysis">
    <header class="ai-analysis__head">
      <div class="ai-analysis__title">
        <Sparkles :size="20" :stroke-width="2.1" />
        <div>
          <h2>AI 分析中心</h2>
          <p>
            相册 / 文件 / 视频 全局后台分析 ·
            {{ aiAnalysisStore.connected.value ? '实时已连接' : '轮询中' }} ·
            等级 {{ levelText }}
          </p>
        </div>
      </div>
      <div class="ai-analysis__actions">
        <UiButton variant="ghost" size="sm" :disabled="analysisOff" @click="rescan">
          <ScanSearch :size="15" :stroke-width="2" />
          重新扫描
        </UiButton>
        <UiButton variant="ghost" size="sm" :disabled="analysisOff" @click="togglePause">
          <component :is="paused ? Play : Pause" :size="15" :stroke-width="2" />
          {{ paused ? '恢复' : '暂停' }}
        </UiButton>
        <UiButton variant="solid" tone="primary" size="sm" :disabled="analysisOff" @click="reanalyzeAll">
          <RefreshCw :size="15" :stroke-width="2" />
          全部重新分析
        </UiButton>
      </div>
    </header>

    <UiCard v-if="analysisOff" class="ai-analysis__notice">
      AI 分析当前已关闭。在「系统设置 → 模型策略 → AI 分析等级」中选择基础 / 标准 / 深度以开启后台分析。
    </UiCard>

    <div class="ai-analysis__overview">
      <UiCard class="ai-analysis__total">
        <div class="ai-analysis__total-head">
          <span>总进度</span>
          <strong>{{ overview?.totalPercent ?? 0 }}%</strong>
        </div>
        <UiProgressBar
          :value="overview?.totalPercent ?? 0"
          :tone="progressToneFor(overview?.totalPercent ?? 0)"
          size="sm"
        />
        <div class="ai-analysis__caps">
          <UiBadge
            v-for="cap in capabilities"
            :key="cap.label"
            :tone="cap.ok ? 'success' : 'neutral'"
            variant="soft"
            size="sm"
          >
            {{ cap.label }}{{ cap.ok ? ' ✓' : '' }}
          </UiBadge>
        </div>
      </UiCard>

      <UiCard v-for="domain in (['media', 'file', 'video'] as AiAnalysisDomain[])" :key="domain" class="ai-analysis__domain">
        <div class="ai-analysis__domain-head">
          <div class="ai-analysis__domain-title">
            <component :is="domainMeta[domain].icon" :size="16" :stroke-width="2" />
            <span>{{ domainMeta[domain].label }}</span>
          </div>
          <UiIconButton
            size="sm"
            variant="ghost"
            :icon="RefreshCw"
            :label="`重新分析${domainMeta[domain].label}`"
            :disabled="analysisOff"
            @click="reanalyzeDomain(domain)"
          />
        </div>
        <UiProgressBar
          :value="domainStat(domain)?.percent ?? 0"
          :tone="progressToneFor(domainStat(domain)?.percent ?? 0)"
          size="sm"
          show-value
        />
        <div class="ai-analysis__domain-stats">
          <span>共 {{ domainStat(domain)?.total ?? 0 }}</span>
          <span class="ai-analysis__dot ai-analysis__dot--primary">分析中 {{ domainStat(domain)?.analyzing ?? 0 }}</span>
          <span class="ai-analysis__dot ai-analysis__dot--success">完成 {{ domainStat(domain)?.done ?? 0 }}</span>
          <span v-if="(domainStat(domain)?.failed ?? 0) > 0" class="ai-analysis__dot ai-analysis__dot--danger">
            失败 {{ domainStat(domain)?.failed ?? 0 }}
          </span>
        </div>
      </UiCard>
    </div>

    <UiTabs v-model="activeDomain" :tabs="tabs" variant="segmented" size="sm" overflow="menu" class="ai-analysis__tabs" />

    <div class="ai-analysis__body">
      <UiDataTable
        :columns="columns"
        :rows="recordRows"
        row-key="key"
        density="compact"
        :empty="{ title: '暂无分析记录', description: '后台开始分析后，相册 / 文件 / 视频的处理记录会显示在这里。' }"
      >
        <template #cell-title="{ row }">
          <div class="ai-analysis__name">
            <component :is="domainMeta[row.domain]?.icon ?? Sparkles" :size="15" :stroke-width="2" />
            <div class="ai-analysis__name-text">
              <strong>{{ row.title }}</strong>
              <span v-if="row.sourcePath">{{ row.sourcePath }}</span>
            </div>
          </div>
        </template>

        <template #cell-state="{ row }">
          <UiBadge :tone="stateMeta[row.state]?.tone ?? 'neutral'" variant="soft" size="sm">
            {{ stateMeta[row.state]?.label ?? row.state }}
          </UiBadge>
        </template>

        <template #cell-result="{ row }">
          <span class="ai-analysis__result" :class="{ 'ai-analysis__result--error': row.state === 'failed' }">
            {{ recordSummary(row) }}
          </span>
        </template>

        <template #cell-updatedAt="{ row }">
          <span class="ai-analysis__time">{{ formatTime(row.updatedAt) }}</span>
        </template>

        <template #cell-actions="{ row }">
          <UiIconButton
            size="sm"
            variant="ghost"
            :icon="RefreshCw"
            label="重新分析此项"
            :disabled="analysisOff"
            @click="reanalyzeItem(row)"
          />
        </template>

        <template #empty>
          <UiEmptyState
            :icon="Sparkles"
            title="暂无分析记录"
            description="后台开始分析后，相册 / 文件 / 视频的处理记录会显示在这里。"
            compact
          />
        </template>
      </UiDataTable>
    </div>

    <footer v-if="recordsPage.total > recordsPage.pageSize" class="ai-analysis__footer">
      <span>共 {{ recordsPage.total }} 条 · 第 {{ recordsPage.page }} / {{ totalPages }} 页</span>
      <div class="ai-analysis__pager">
        <UiButton variant="ghost" size="sm" :disabled="recordsPage.page <= 1" @click="changePage(-1)">上一页</UiButton>
        <UiButton variant="ghost" size="sm" :disabled="recordsPage.page >= totalPages" @click="changePage(1)">下一页</UiButton>
      </div>
    </footer>
  </div>
</template>

<style scoped>
.ai-analysis {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  gap: var(--space-3);
}

.ai-analysis__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.ai-analysis__title {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--accent);
}
.ai-analysis__title h2 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
}
.ai-analysis__title p {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.ai-analysis__actions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.ai-analysis__notice {
  padding: var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}

.ai-analysis__overview {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: var(--space-3);
  flex-shrink: 0;
}

.ai-analysis__total,
.ai-analysis__domain {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}

.ai-analysis__total-head,
.ai-analysis__domain-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}
.ai-analysis__total-head strong {
  color: var(--accent);
}

.ai-analysis__domain-title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--accent);
}

.ai-analysis__caps {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-1);
}

.ai-analysis__domain-stats {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
.ai-analysis__dot--primary {
  color: var(--accent);
}
.ai-analysis__dot--success {
  color: var(--accent-green);
}
.ai-analysis__dot--danger {
  color: var(--accent-red);
}

.ai-analysis__tabs {
  flex-shrink: 0;
}

.ai-analysis__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.ai-analysis__name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--accent);
}
.ai-analysis__name-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.ai-analysis__name-text strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}
.ai-analysis__name-text span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.ai-analysis__result {
  display: block;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.ai-analysis__result--error {
  color: var(--accent-red);
}

.ai-analysis__time {
  color: var(--text-soft);
  font-size: var(--fs-xs);
}

.ai-analysis__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-shrink: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.ai-analysis__pager {
  display: flex;
  gap: var(--space-2);
}
</style>
