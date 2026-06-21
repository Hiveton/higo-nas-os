<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Film, FolderOpen, Images, ListChecks, Pause, Play, RefreshCw, ScanFace, ScanSearch, Search, Sparkles } from 'lucide-vue-next';
import type { Component } from 'vue';
import { aiAnalysisStore } from '../../stores/aiAnalysis';
import type { AiAnalysisDomain, AiAnalysisRecord, AiAnalysisState } from '../../api/types';
import FacesPanel from './ai-analysis/FacesPanel.vue';
import RecordDetailDrawer from './ai-analysis/RecordDetailDrawer.vue';
import {
  UiBadge,
  UiButton,
  UiCard,
  UiDataTable,
  UiEmptyState,
  UiIconButton,
  UiInput,
  UiProgressBar,
  UiSegmented,
  UiSelect,
  UiTabs,
  UiWindowPage,
  useToast,
  type Column,
  type TabItem,
  type UiTone,
} from '../ui';

const toast = useToast();

const overview = aiAnalysisStore.overview;
const records = aiAnalysisStore.records;
const recordsPage = aiAnalysisStore.recordsPage;
const recordDetail = aiAnalysisStore.recordDetail;

// UiDataTable expects a mutable rows array; the store exposes records deep-readonly
// (it only reads them), so cast through unknown.
const recordRows = computed(() => [...records.value] as unknown as AiAnalysisRecord[]);
// Store exposes the detail deep-readonly; the drawer prop wants a plain record.
const detailRecord = computed(() => recordDetail.value as unknown as AiAnalysisRecord | null);

type DomainFilter = '' | AiAnalysisDomain;
const activeDomain = ref<DomainFilter>('');

// Top-level view: the analysis records table, or the face library.
const activeView = ref<'records' | 'faces'>('records');

// Toolbar filters.
const stateFilter = ref<'' | AiAnalysisState>('');
const searchText = ref('');
let searchTimer: ReturnType<typeof setTimeout> | undefined;

// Batch selection (managed locally so we don't depend on the in-flight
// DataTable multi-select upgrade).
const selectedKeys = ref<Set<string>>(new Set());
const selectedCount = computed(() => selectedKeys.value.size);

const levelOptions = [
  { label: '关闭', value: 'off' },
  { label: '基础', value: 'basic' },
  { label: '标准', value: 'standard' },
  { label: '深度', value: 'deep' },
];
const currentLevel = computed(() => (overview.value?.level as string) ?? 'basic');

const stateOptions = [
  { label: '全部状态', value: '' },
  { label: '待分析', value: 'pending' },
  { label: '分析中', value: 'analyzing' },
  { label: '已完成', value: 'done' },
  { label: '失败', value: 'failed' },
  { label: '已跳过', value: 'skipped' },
];

async function changeLevel(value: string | number) {
  try {
    await aiAnalysisStore.setLevel(value as 'off' | 'basic' | 'standard' | 'deep');
    toast.success(`分析等级已设为「${levelOptions.find((o) => o.value === value)?.label ?? value}」`);
    await refreshRecords();
  } catch (error) {
    toast.error(`切换失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

function applyFilters() {
  void aiAnalysisStore.setFilters({
    domain: activeDomain.value || undefined,
    state: stateFilter.value || undefined,
    q: searchText.value.trim() || undefined,
  });
  selectedKeys.value = new Set();
}

function onSearchInput() {
  if (searchTimer) clearTimeout(searchTimer);
  searchTimer = setTimeout(applyFilters, 280);
}

function toggleSelect(key: string) {
  const next = new Set(selectedKeys.value);
  if (next.has(key)) next.delete(key);
  else next.add(key);
  selectedKeys.value = next;
}

function toggleSelectAll() {
  if (recordRows.value.length > 0 && recordRows.value.every((r) => selectedKeys.value.has(r.key))) {
    selectedKeys.value = new Set();
  } else {
    selectedKeys.value = new Set(recordRows.value.map((r) => r.key));
  }
}

const allSelected = computed(
  () => recordRows.value.length > 0 && recordRows.value.every((r) => selectedKeys.value.has(r.key)),
);

async function batchReanalyze() {
  const keys = [...selectedKeys.value];
  if (!keys.length) return;
  try {
    const reset = await aiAnalysisStore.batchReanalyze(keys);
    toast.success(`已将 ${reset} 项加入重新分析队列`);
    selectedKeys.value = new Set();
  } catch (error) {
    toast.error(`批量操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

function openDetail(row: AiAnalysisRecord) {
  void aiAnalysisStore.loadRecord(row.key);
}

function closeDetail() {
  aiAnalysisStore.clearRecordDetail();
}

async function reanalyzeFromDetail(key: string) {
  await aiAnalysisStore.reanalyze({ scope: 'item', itemId: key });
  toast.success('已加入重新分析队列');
  closeDetail();
  await refreshRecords();
}

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

type Capability = { label: string; ok: boolean; hint: string; settings: boolean };

const capabilities = computed<Capability[]>(() => {
  const o = overview.value;
  if (!o) return [];
  return [
    { label: '对话模型', ok: o.hasChat, hint: '生成摘要与标签，需配置对话模型', settings: true },
    { label: '视觉模型', ok: o.hasVision, hint: '识别照片场景 / 人物 / 视频关键帧，需配置视觉模型', settings: true },
    {
      label: '向量索引',
      ok: o.hasEmbedding && o.indexEnabled,
      hint: o.hasEmbedding ? '语义检索需配置数据库（HIGO_DATABASE_URL）' : '语义检索需配置向量模型',
      settings: o.hasEmbedding ? false : true,
    },
    { label: '语音转写', ok: o.hasAsr, hint: '视频字幕自动生成，需配置语音模型', settings: true },
    { label: 'ffmpeg', ok: o.ffmpegAvailable, hint: '视频转码 / 抽帧 / 抽音，需主机安装 ffmpeg', settings: false },
  ];
});

// missingChat/Vision drive the degraded annotation on records.
const degraded = computed(() => {
  const o = overview.value;
  return !!o && (o.level === 'standard' || o.level === 'deep') && !o.hasChat;
});

function openModelSettings() {
  window.dispatchEvent(new CustomEvent('higoos:open-app', { detail: 'system-settings' }));
}

function onCapabilityClick(cap: Capability) {
  if (cap.ok) return;
  if (cap.settings) openModelSettings();
  else toast.success(cap.hint);
}

const stateMeta: Record<AiAnalysisState, { label: string; tone: UiTone }> = {
  pending: { label: '待分析', tone: 'neutral' },
  analyzing: { label: '分析中', tone: 'primary' },
  done: { label: '已完成', tone: 'success' },
  failed: { label: '失败', tone: 'danger' },
  skipped: { label: '已跳过', tone: 'warning' },
};

const columns: Column<AiAnalysisRecord>[] = [
  { key: 'select', label: '', width: '4%' },
  { key: 'title', label: '项目', width: '26%' },
  { key: 'state', label: '状态', width: '12%' },
  { key: 'result', label: '分析结果', width: '38%' },
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
  if (!parts.length) return '—';
  // Flag results produced without a model so users understand why they look thin.
  if (row.state === 'done' && row.level === 'basic' && degraded.value) {
    parts.push('（基础：未配置模型）');
  }
  return parts.join(' · ');
}

async function refreshRecords() {
  await aiAnalysisStore.loadRecords(recordsPage.value.page);
}

async function changePage(delta: number) {
  const next = recordsPage.value.page + delta;
  if (next < 1 || next > totalPages.value) return;
  await aiAnalysisStore.loadRecords(next);
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

watch(activeDomain, applyFilters);
watch(stateFilter, applyFilters);

// Keep the visible page fresh as the live overview advances (records view only).
watch(
  () => overview.value?.totalPercent,
  () => {
    if (activeView.value === 'records' && !aiAnalysisStore.recordsLoading.value) void refreshRecords();
  },
);

onMounted(() => {
  aiAnalysisStore.start();
  void aiAnalysisStore.loadRecords(1);
});
onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer);
  aiAnalysisStore.stop();
});
</script>

<template>
  <UiWindowPage
    layout="dashboard"
    :icon="Sparkles"
    title="AI 分析中心"
    :subtitle="`相册 / 文件 / 视频 全局后台分析 · ${aiAnalysisStore.connected.value ? '实时已连接' : '轮询中'} · 等级 ${levelText}`"
  >
    <template #actions>
      <UiSegmented
        :model-value="currentLevel"
        :options="levelOptions"
        size="sm"
        class="ai-analysis__level"
        aria-label="分析等级"
        @change="changeLevel"
      />
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
    </template>

    <template #toolbar>
      <UiSegmented
        v-model="activeView"
        :options="[{ label: '分析记录', value: 'records' }, { label: '人脸库', value: 'faces' }]"
        size="sm"
      />
    </template>

    <UiCard v-if="analysisOff" class="ai-analysis__notice">
      <span>AI 分析当前已关闭。在「系统设置 → 模型策略 → AI 分析等级」中选择基础 / 标准 / 深度以开启后台分析。</span>
      <UiButton variant="soft" tone="primary" size="sm" @click="openModelSettings">前往设置</UiButton>
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
          <button
            v-for="cap in capabilities"
            :key="cap.label"
            type="button"
            class="ai-analysis__cap"
            :class="{ 'ai-analysis__cap--missing': !cap.ok }"
            :title="cap.ok ? `${cap.label}已就绪` : `${cap.hint}${cap.settings ? '（点击前往设置）' : ''}`"
            @click="onCapabilityClick(cap)"
          >
            <UiBadge :tone="cap.ok ? 'success' : 'neutral'" variant="soft" size="sm">
              {{ cap.label }}{{ cap.ok ? ' ✓' : ' ·未配置' }}
            </UiBadge>
          </button>
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

    <template v-if="activeView === 'records'">
      <div class="ai-analysis__toolbar">
        <UiTabs v-model="activeDomain" :tabs="tabs" variant="segmented" size="sm" overflow="menu" class="ai-analysis__tabs" />
        <div class="ai-analysis__filters">
          <UiButton variant="ghost" size="sm" :disabled="recordRows.length === 0" @click="toggleSelectAll">
            {{ allSelected ? '取消全选' : '全选本页' }}
          </UiButton>
          <UiSelect v-model="stateFilter" :options="stateOptions" class="ai-analysis__state" />
          <UiInput v-model="searchText" placeholder="搜索项目 / 路径 / 错误" :icon-left="Search" size="sm" clearable class="ai-analysis__search" @input="onSearchInput" />
        </div>
      </div>

      <div v-if="selectedCount > 0" class="ai-analysis__batchbar">
        <span>已选 {{ selectedCount }} 项</span>
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="ListChecks" @click="batchReanalyze">批量重新分析</UiButton>
        <UiButton variant="ghost" size="sm" @click="selectedKeys = new Set()">取消选择</UiButton>
      </div>

      <div class="ai-analysis__body">
      <UiDataTable
        :columns="columns"
        :rows="recordRows"
        row-key="key"
        density="compact"
        :empty="{ title: '暂无分析记录', description: '后台开始分析后，相册 / 文件 / 视频的处理记录会显示在这里。' }"
      >
        <template #cell-select="{ row }">
          <input
            type="checkbox"
            class="ai-analysis__check"
            :checked="selectedKeys.has(row.key)"
            :aria-label="`选择 ${row.title}`"
            @click.stop="toggleSelect(row.key)"
          />
        </template>

        <template #cell-title="{ row }">
          <button type="button" class="ai-analysis__name" :title="`查看 ${row.title} 详情`" @click="openDetail(row)">
            <component :is="domainMeta[row.domain]?.icon ?? Sparkles" :size="15" :stroke-width="2" />
            <div class="ai-analysis__name-text">
              <strong>{{ row.title }}</strong>
              <span v-if="row.sourcePath">{{ row.sourcePath }}</span>
            </div>
          </button>
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
    </template>

    <div v-else class="ai-analysis__body">
      <FacesPanel />
    </div>

    <RecordDetailDrawer
      :record="detailRecord"
      :loading="aiAnalysisStore.recordDetailLoading.value"
      @close="closeDetail"
      @reanalyze="reanalyzeFromDetail"
    />
  </UiWindowPage>
</template>

<style scoped>
.ai-analysis__level {
  margin-right: var(--space-2);
}

.ai-analysis__toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-2);
  flex-shrink: 0;
}

.ai-analysis__filters {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.ai-analysis__search {
  width: 220px;
  max-width: 46vw;
}

.ai-analysis__batchbar {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-card);
  background: var(--accent-soft);
  color: var(--text-strong);
  font-size: var(--fs-xs);
  flex-shrink: 0;
}

.ai-analysis__check {
  width: 15px;
  height: 15px;
  cursor: pointer;
  accent-color: var(--accent);
}

.ai-analysis__notice {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}
.ai-analysis__notice :deep(button) {
  flex-shrink: 0;
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
.ai-analysis__cap {
  padding: 0;
  border: none;
  background: none;
  cursor: default;
}
.ai-analysis__cap--missing {
  cursor: pointer;
}
.ai-analysis__cap--missing:hover {
  opacity: 0.8;
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
  color: var(--ink-green);
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
  width: 100%;
  padding: 0;
  color: var(--accent);
  background: transparent;
  border: 0;
  text-align: left;
  cursor: pointer;
}
.ai-analysis__name:hover .ai-analysis__name-text strong {
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
