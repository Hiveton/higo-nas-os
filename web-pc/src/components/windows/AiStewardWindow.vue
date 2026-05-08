<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { AlertTriangle, ArchiveRestore, CheckCircle2, History, ShieldAlert } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AuditEntry, StewardSuggestion } from '../../api/types';
import { auditEntries as seedAuditEntries, stewardSuggestions as seedStewardSuggestions } from '../../data/higoos';

const riskClass = {
  低风险: 'ai-steward__risk--low',
  中风险: 'ai-steward__risk--mid',
  高风险: 'ai-steward__risk--high',
};

const dismissedSuggestions = ref<string[]>([]);
const stewardSuggestions = ref<StewardSuggestion[]>(seedStewardSuggestions);
const auditEntries = ref<AuditEntry[]>(seedAuditEntries.map((entry, index) => ({
  id: `seed-audit-${index}`,
  event: entry,
  actor: '本地缓存',
  risk: '低风险',
  reverted: false,
  rollback: '等待后端审计同步',
})));
const activeSuggestion = ref(stewardSuggestions.value[0]?.title ?? '');
const actionLog = ref<string[]>([]);
const loading = ref(false);
const actionBusyId = ref<string | null>(null);
const backendNotice = ref('正在使用本地缓存，后端连接后会同步建议与审计。');
const previewConfirmations = ref<Record<string, string>>({});

const visibleSuggestions = computed(() =>
  stewardSuggestions.value.filter((item) => !dismissedSuggestions.value.includes(item.title)),
);

async function loadStewardState() {
  loading.value = true;
  try {
    const [suggestions, audit] = await Promise.all([
      apiClient.steward.getSuggestions(),
      apiClient.steward.getAudit(),
    ]);
    stewardSuggestions.value = suggestions;
    auditEntries.value = audit;
    activeSuggestion.value = visibleSuggestions.value[0]?.title ?? '';
    backendNotice.value = 'AI 文件管家已连接后端，建议、确认与审计会实时写回。';
  } catch (error) {
    backendNotice.value = `后端暂不可用，继续使用本地缓存：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function handleSuggestionAction(item: StewardSuggestion) {
  const id = item.id ?? item.title;
  activeSuggestion.value = item.title;
  actionBusyId.value = id;
  try {
    const preview = await apiClient.steward.previewSuggestion(id);
    const confirmationId = typeof preview.confirmationId === 'string' ? preview.confirmationId : '';
    if (confirmationId) {
      previewConfirmations.value[id] = confirmationId;
    }
    actionLog.value.unshift(`${item.action}：${item.title}`);
    backendNotice.value = typeof preview.impact === 'string' ? preview.impact : '已生成执行预览，确认前不会修改文件。';
  } catch (error) {
    backendNotice.value = `预览失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    actionBusyId.value = null;
  }
}

async function completeSuggestion(item: StewardSuggestion) {
  const id = item.id ?? item.title;
  actionBusyId.value = id;
  try {
    let confirmationId = previewConfirmations.value[id];
    if (!confirmationId && item.risk !== '低风险') {
      const preview = await apiClient.steward.previewSuggestion(id);
      confirmationId = typeof preview.confirmationId === 'string' ? preview.confirmationId : '';
      if (confirmationId) {
        previewConfirmations.value[id] = confirmationId;
      }
    }
    await apiClient.steward.confirmSuggestion(id, confirmationId ? { confirmationId } : {});
    if (!dismissedSuggestions.value.includes(item.title)) {
      dismissedSuggestions.value.push(item.title);
    }
    actionLog.value.unshift(`已确认执行：${item.title}`);
    await refreshAudit();
  } catch (error) {
    backendNotice.value = `确认失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    activeSuggestion.value = visibleSuggestions.value[0]?.title ?? '';
    actionBusyId.value = null;
  }
}

async function refreshAudit() {
  try {
    auditEntries.value = await apiClient.steward.getAudit();
    backendNotice.value = '审计记录已同步。';
  } catch (error) {
    backendNotice.value = `审计同步失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

onMounted(loadStewardState);
</script>

<template>
  <div class="ai-steward">
    <section class="ai-steward__hero">
        <div>
          <p>{{ loading ? '正在同步后端' : '智能整理队列' }}</p>
          <strong>{{ visibleSuggestions.length }} 条建议等待处理</strong>
        </div>
      <CheckCircle2 :size="30" />
    </section>

    <section class="ai-steward__suggestions" aria-label="智能整理建议">
      <article
        v-for="item in visibleSuggestions"
        :key="item.title"
        class="ai-steward__suggestion"
        :class="{ 'ai-steward__suggestion--active': activeSuggestion === item.title }"
      >
        <div class="ai-steward__suggestion-head">
          <div>
            <h3>{{ item.title }}</h3>
            <p>{{ item.detail }}</p>
          </div>
          <span :class="['ai-steward__risk', riskClass[item.risk]]">{{ item.risk }}</span>
        </div>
        <div class="ai-steward__suggestion-foot">
          <span>{{ item.count }}</span>
          <div>
            <button type="button" :disabled="actionBusyId === (item.id ?? item.title)" @click="handleSuggestionAction(item)">
              {{ actionBusyId === (item.id ?? item.title) ? '处理中' : item.action }}
            </button>
            <button type="button" class="ai-steward__ghost-button" :disabled="actionBusyId === (item.id ?? item.title)" @click="completeSuggestion(item)">确认</button>
          </div>
        </div>
      </article>
      <div v-if="visibleSuggestions.length === 0" class="ai-steward__empty">
        <CheckCircle2 :size="22" />
        <strong>整理队列已清空</strong>
        <span>所有建议都已确认或写入审计日志。</span>
      </div>
    </section>

    <section class="ai-steward__governance" aria-label="风险、审计和回滚">
      <div class="ai-steward__risk-card">
        <ShieldAlert :size="18" />
        <div>
          <strong>执行风险</strong>
          <p>{{ backendNotice || activeSuggestion || '移动、重命名、分享权限变更均需管理员确认。' }}</p>
        </div>
      </div>
      <div class="ai-steward__risk-card">
        <ArchiveRestore :size="18" />
        <div>
          <strong>回滚保护</strong>
          <p>保留原路径、权限和命名快照，可一键撤销。</p>
        </div>
      </div>
    </section>

    <section class="ai-steward__audit" aria-label="审计记录">
      <h3><History :size="15" /> 审计 / 回滚</h3>
      <ul>
        <li v-for="entry in actionLog" :key="entry">
          <CheckCircle2 :size="13" />
          <span>{{ entry }}</span>
        </li>
        <li v-for="entry in auditEntries" :key="entry.id">
          <AlertTriangle :size="13" />
          <span>{{ entry.event }}</span>
        </li>
      </ul>
    </section>
  </div>
</template>

<style scoped>
.ai-steward {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.ai-steward__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  min-height: 70px;
  padding: 14px 16px;
  color: var(--text-strong);
  background: linear-gradient(135deg, rgba(231, 247, 255, 0.9), rgba(255, 246, 227, 0.82));
  border: 1px solid rgba(22, 199, 221, 0.22);
  border-radius: var(--radius-md);
}

.ai-steward__hero p,
.ai-steward__hero strong {
  display: block;
  margin: 0;
}

.ai-steward__hero p {
  color: var(--text-muted);
  font-size: 12px;
}

.ai-steward__hero strong {
  margin-top: 4px;
  font-size: 18px;
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
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.ai-steward__suggestion {
  padding: 12px;
}

.ai-steward__suggestion--active {
  border-color: rgba(19, 136, 255, 0.28);
  box-shadow: inset 0 0 0 1px rgba(19, 136, 255, 0.08);
}

.ai-steward__suggestion-head {
  display: flex;
  gap: 10px;
  justify-content: space-between;
}

.ai-steward__suggestion h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 13px;
}

.ai-steward__suggestion p {
  margin: 6px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.42;
}

.ai-steward__risk {
  height: 22px;
  padding: 4px 8px;
  font-size: 11px;
  font-weight: 760;
  white-space: nowrap;
  border-radius: 999px;
}

.ai-steward__risk--low {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.12);
}

.ai-steward__risk--mid {
  color: #b36a00;
  background: rgba(245, 158, 11, 0.14);
}

.ai-steward__risk--high {
  color: var(--accent-red);
  background: rgba(239, 68, 68, 0.12);
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
  font-weight: 800;
}

.ai-steward__suggestion-foot div {
  display: flex;
  gap: 6px;
}

.ai-steward__suggestion-foot button {
  height: 30px;
  padding: 0 12px;
  color: #fff;
  background: var(--accent);
  border: 0;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 760;
}

.ai-steward__suggestion-foot .ai-steward__ghost-button {
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
}

.ai-steward__empty {
  display: grid;
  min-height: 128px;
  place-items: center;
  align-content: center;
  gap: 6px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: 11px;
}

.ai-steward__empty strong {
  color: var(--text-strong);
  font-size: 13px;
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
  background: rgba(255, 255, 255, 0.56);
  border-radius: var(--radius-sm);
}

.ai-steward__risk-card strong {
  color: var(--text-strong);
  font-size: 12px;
}

.ai-steward__risk-card p {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
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
  font-size: 12px;
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
  gap: 7px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.3;
}
</style>
