<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { AlertTriangle, ArchiveRestore, CheckCircle2, History, RefreshCw, ShieldAlert, Sparkles, Undo2 } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AuditEntry, StewardSuggestion } from '../../api/types';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { UiBadge, UiButton, UiEmptyState, UiWindowPage } from '../ui';
import type { UiTone } from '../ui';

const emit = defineEmits<{ (e: 'open-agent'): void }>();

const riskTone: Record<string, UiTone> = {
  低风险: 'success',
  中风险: 'warning',
  高风险: 'danger',
};

const suggestions = ref<StewardSuggestion[]>([]);
const auditEntries = ref<AuditEntry[]>([]);
const loading = ref(false);
const actionBusyId = ref<string | null>(null);
const rollbackBusyId = ref<string | null>(null);
const backendNotice = ref('正在连接后端…');
// Per-suggestion preview state: the confirmation token and the impact text.
const previews = ref<Record<string, { confirmationId: string; impact: string }>>({});

// Only pending suggestions belong in the action queue; confirmed/dismissed ones
// live in the audit trail.
const visibleSuggestions = computed(() =>
  suggestions.value.filter((item) => (item.status ?? 'pending') === 'pending'),
);

function keyOf(item: StewardSuggestion): string {
  return item.id ?? item.title;
}

function errorText(error: unknown): string {
  return error instanceof Error ? error.message : 'unknown error';
}

async function loadStewardState() {
  loading.value = true;
  try {
    const [list, audit] = await Promise.all([
      apiClient.steward.getSuggestions(),
      apiClient.steward.getAudit(),
    ]);
    suggestions.value = list;
    auditEntries.value = audit;
    backendNotice.value = 'AI 文件管家已连接后端，建议、确认与审计会实时写回。';
  } catch (error) {
    backendNotice.value = `后端暂不可用：${errorText(error)}`;
  } finally {
    loading.value = false;
  }
}

async function refreshSuggestions() {
  loading.value = true;
  previews.value = {};
  try {
    suggestions.value = await apiClient.steward.refresh();
    const pending = visibleSuggestions.value.length;
    backendNotice.value = pending > 0
      ? `已重新分析文件，生成 ${pending} 条建议。`
      : '已重新分析文件，暂无可整理项。';
  } catch (error) {
    backendNotice.value = `重新分析失败：${errorText(error)}`;
  } finally {
    loading.value = false;
  }
}

// Preview generates the impact summary and a confirmation token (for
// medium/high-risk items). It never modifies files.
async function previewSuggestion(item: StewardSuggestion) {
  const id = keyOf(item);
  actionBusyId.value = id;
  try {
    const preview = await apiClient.steward.previewSuggestion(id);
    const confirmationId = typeof preview.confirmationId === 'string' ? preview.confirmationId : '';
    const impact = typeof preview.impact === 'string' ? preview.impact : '已生成执行预览，确认前不会修改文件。';
    previews.value = { ...previews.value, [id]: { confirmationId, impact } };
    backendNotice.value = impact;
  } catch (error) {
    backendNotice.value = `预览失败：${errorText(error)}`;
  } finally {
    actionBusyId.value = null;
  }
}

// Confirm executes the suggestion's real, reversible operations. Medium/high-risk
// items require a preview token first; low-risk ones may skip straight to confirm.
async function confirmSuggestion(item: StewardSuggestion) {
  const id = keyOf(item);
  actionBusyId.value = id;
  try {
    let confirmationId = previews.value[id]?.confirmationId ?? '';
    if (!confirmationId && item.risk !== '低风险') {
      const preview = await apiClient.steward.previewSuggestion(id);
      confirmationId = typeof preview.confirmationId === 'string' ? preview.confirmationId : '';
    }
    await apiClient.steward.confirmSuggestion(id, confirmationId ? { confirmationId } : {});
    backendNotice.value = `已确认执行：${item.title}`;
    await reload();
  } catch (error) {
    backendNotice.value = `确认失败：${errorText(error)}`;
  } finally {
    actionBusyId.value = null;
  }
}

// Dismiss tells the backend to drop a suggestion (recorded in the audit trail).
async function dismissSuggestion(item: StewardSuggestion) {
  const id = keyOf(item);
  actionBusyId.value = id;
  try {
    await apiClient.steward.dismissSuggestion(id, { reason: '用户忽略' });
    backendNotice.value = `已忽略建议：${item.title}`;
    await reload();
  } catch (error) {
    backendNotice.value = `忽略失败：${errorText(error)}`;
  } finally {
    actionBusyId.value = null;
  }
}

// Rollback reverses a confirmed action: recycled files are restored, archived
// files are moved back to their original location.
async function rollbackAudit(entry: AuditEntry) {
  rollbackBusyId.value = entry.id;
  try {
    await apiClient.steward.rollbackAudit(entry.id);
    backendNotice.value = '已撤销该操作，文件已恢复原位。';
    await reload();
  } catch (error) {
    backendNotice.value = `撤销失败：${errorText(error)}`;
  } finally {
    rollbackBusyId.value = null;
  }
}

async function reload() {
  const [list, audit] = await Promise.all([
    apiClient.steward.getSuggestions(),
    apiClient.steward.getAudit(),
  ]);
  suggestions.value = list;
  auditEntries.value = audit;
}

// A confirmed action with a rollback token that hasn't been reverted yet can be undone.
function canRollback(entry: AuditEntry): boolean {
  return Boolean(entry.rollback) && entry.result === 'confirmed' && !entry.reverted;
}

onMounted(loadStewardState);
</script>

<template>
  <UiWindowPage
    layout="chat"
    :icon="Sparkles"
    title="AI 文件管家"
    :subtitle="`${visibleSuggestions.length} 条建议等待处理`"
    :status="loading ? '正在同步后端' : '智能整理队列'"
  >
    <template #actions>
      <UiButton size="sm" variant="ghost" :icon-left="Sparkles" @click="emit('open-agent')">
        交给 AI 助手
      </UiButton>
      <UiButton size="sm" variant="soft" :icon-left="RefreshCw" :loading="loading" @click="refreshSuggestions">
        重新分析
      </UiButton>
    </template>

    <div class="ai-steward">
    <section class="ai-steward__suggestions" aria-label="智能整理建议">
      <article
        v-for="item in visibleSuggestions"
        :key="keyOf(item)"
        class="ai-steward__suggestion"
        :class="{ 'ai-steward__suggestion--active': Boolean(previews[keyOf(item)]) }"
      >
        <div class="ai-steward__suggestion-head">
          <div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.detail }}</p>
          </div>
          <UiBadge :tone="riskTone[item.risk] ?? 'neutral'">{{ item.risk }}</UiBadge>
        </div>
        <p v-if="previews[keyOf(item)]" class="ai-steward__suggestion-preview">
          <ShieldAlert :size="12" /> {{ previews[keyOf(item)].impact }}
        </p>
        <div class="ai-steward__suggestion-foot">
          <span>{{ item.count }}</span>
          <div>
            <UiButton
              variant="ghost"
              size="sm"
              :disabled="actionBusyId === keyOf(item)"
              @click="dismissSuggestion(item)"
            >
              忽略
            </UiButton>
            <UiButton
              variant="soft"
              size="sm"
              :loading="actionBusyId === keyOf(item)"
              :disabled="actionBusyId === keyOf(item)"
              @click="previewSuggestion(item)"
            >
              {{ item.action || '预览' }}
            </UiButton>
            <UiButton
              size="sm"
              :loading="actionBusyId === keyOf(item)"
              :disabled="actionBusyId === keyOf(item)"
              @click="confirmSuggestion(item)"
            >
              确认执行
            </UiButton>
          </div>
        </div>
      </article>
      <UiEmptyState
        v-if="visibleSuggestions.length === 0"
        :icon="CheckCircle2"
        title="整理队列已清空"
        description="所有建议都已确认或写入审计日志。点「重新分析」可再次扫描文件。"
        compact
      />
    </section>

    <section class="ai-steward__governance" aria-label="风险、审计和回滚">
      <div class="ai-steward__risk-card">
        <ShieldAlert :size="18" />
        <div>
          <strong>执行风险</strong>
          <p>{{ backendNotice || '移动、删除、归档均需确认，执行前会展示影响范围。' }}</p>
        </div>
      </div>
      <div class="ai-steward__risk-card">
        <ArchiveRestore :size="18" />
        <div>
          <strong>回滚保护</strong>
          <p>删除进回收站、归档可移回原路径，确认后的操作可在下方一键撤销。</p>
        </div>
      </div>
    </section>

    <section class="ai-steward__audit" aria-label="审计记录">
      <h3><History :size="15" /> 审计 / 回滚</h3>
      <ul v-if="auditEntries.length">
        <li v-for="entry in auditEntries" :key="entry.id">
          <component :is="entry.reverted ? Undo2 : entry.result === 'confirmed' ? CheckCircle2 : AlertTriangle" :size="13" />
          <span>{{ entry.event }}</span>
          <UiButton
            v-if="canRollback(entry)"
            class="ai-steward__rollback"
            variant="ghost"
            size="sm"
            :icon-left="Undo2"
            :loading="rollbackBusyId === entry.id"
            :disabled="rollbackBusyId === entry.id"
            @click="rollbackAudit(entry)"
          >
            撤销
          </UiButton>
        </li>
      </ul>
      <UiEmptyState v-else :icon="History" title="暂无审计记录" description="确认或忽略建议后会在此留痕。" compact />
    </section>
    <NasFeaturePanel class="ai-steward__features" :modules="['files', 'security']" />
    </div>
  </UiWindowPage>
</template>

<style scoped>
.ai-steward {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto auto auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
  overflow: auto;
}

.ai-steward__features {
  min-height: 0;
}

.ai-steward__suggestion-preview {
  display: flex;
  align-items: center;
  gap: 5px;
  margin: 8px 0 0;
  padding: 7px 9px;
  color: var(--text-strong);
  font-size: var(--fs-2xs);
  line-height: 1.4;
  background: var(--accent-soft);
  border-radius: var(--radius-control);
}

.ai-steward__suggestions {
  display: grid;
  gap: 10px;
  min-height: 0;
  overflow: auto;
}

.ai-steward__suggestion,
.ai-steward__governance,
.ai-steward__audit {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.ai-steward__suggestion {
  padding: 12px;
}

.ai-steward__suggestion--active {
  border-color: var(--accent-soft);
  box-shadow: inset 0 0 0 1px var(--accent-soft);
}

.ai-steward__suggestion-head {
  display: flex;
  gap: 10px;
  justify-content: space-between;
}

.ai-steward__suggestion h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.ai-steward__suggestion p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.42;
}

.ai-steward__suggestion-foot {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-top: 12px;
}

.ai-steward__suggestion-foot span {
  color: var(--text-strong);
  font-size: 18px;
  font-weight: var(--fw-bold);
}

.ai-steward__suggestion-foot div {
  display: flex;
  gap: 6px;
}

.ai-steward__governance {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 10px;
}

.ai-steward__risk-card {
  display: flex;
  gap: 9px;
  min-width: 0;
  padding: 9px;
  background: rgba(var(--surface-rgb), 0.56);
  border-radius: var(--radius-control);
}

.ai-steward__risk-card strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.ai-steward__risk-card p {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
}

.ai-steward__audit {
  padding: 11px 12px;
}

.ai-steward__audit h3 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0 0 9px;
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.ai-steward__audit ul {
  display: grid;
  gap: 7px;
  padding: 0;
  margin: 0;
  list-style: none;
}

.ai-steward__audit li {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.3;
}

.ai-steward__audit li span {
  flex: 1;
  min-width: 0;
}

.ai-steward__rollback {
  flex-shrink: 0;
}
</style>
