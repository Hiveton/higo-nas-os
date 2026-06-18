<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  ArchiveRestore,
  CheckCircle2,
  EyeOff,
  FileWarning,
  Filter,
  History,
  KeyRound,
  Link2Off,
  ShieldAlert,
  ShieldCheck,
  UserRoundCog,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AiPolicy, AuditEntry, FileShare, IdentityPolicy, RiskAction, RiskLevel } from '../../api/types';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { UiBadge, UiButton, UiCheckbox, UiSegmented } from '../ui';
import type { UiTone } from '../ui';

type RiskFilter = '全部' | RiskLevel;

const riskTone: Record<RiskLevel, UiTone> = {
  低风险: 'success',
  中风险: 'warning',
  高风险: 'danger',
};

const riskFilterOptions: { value: RiskFilter; label: string }[] = [
  { value: '全部', label: '全部' },
  { value: '低风险', label: '低风险' },
  { value: '中风险', label: '中风险' },
  { value: '高风险', label: '高风险' },
];

const identities = ref<IdentityPolicy[]>([
  { id: 'admin', role: '管理员', name: 'Hiveton', mfa: true, fileAcl: true, appAdmin: true, aiTools: true },
  { id: 'family', role: '家庭成员', name: '家人空间', mfa: true, fileAcl: true, appAdmin: false, aiTools: true },
  { id: 'guest', role: '访客', name: '临时分享用户', mfa: false, fileAcl: false, appAdmin: false, aiTools: false },
]);

const aiPolicies = ref<AiPolicy[]>([
  { id: 'family-photos', space: '家庭相册', indexed: true, cloudModel: false, sensitive: '人脸与定位仅本地索引' },
  { id: 'finance', space: '财务票据', indexed: false, cloudModel: false, sensitive: '禁止进入 AI 分析' },
  { id: 'project-docs', space: '项目资料', indexed: true, cloudModel: true, sensitive: '仅团队成员可问答' },
]);

const shareLinks = ref<FileShare[]>([
  { id: 's1', name: '家庭相册春节精选', target: '/家庭空间/相册/春节', access: '密码 + 7 天', downloads: 18, risk: '中风险' as RiskLevel, active: true },
  { id: 's2', name: '合同扫描件外链', target: '/财务票据/合同', access: '公开访问', downloads: 4, risk: '高风险' as RiskLevel, active: true },
  { id: 's3', name: '安装包临时分发', target: '/项目资料/release', access: '团队可见', downloads: 27, risk: '低风险' as RiskLevel, active: true },
]);

const riskActions = ref<RiskAction[]>([
  {
    id: 'r1',
    title: 'Agent 申请批量重命名照片',
    level: '中风险',
    scope: '家庭相册 / 268 个文件',
    actor: '相册整理 Agent',
    state: '待处理',
    confirmed: false,
    rollback: '恢复原文件名快照',
  },
  {
    id: 'r2',
    title: '公开分享合同扫描件',
    level: '高风险',
    scope: '财务票据 / 合同扫描件',
    actor: '外链分享',
    state: '待处理',
    confirmed: false,
    rollback: '撤销链接并恢复 ACL',
  },
  {
    id: 'r3',
    title: 'AI 摘要读取项目资料',
    level: '低风险',
    scope: '项目资料 / 只读摘要',
    actor: '知识问答 Agent',
    state: '待处理',
    confirmed: false,
    rollback: '清除本次摘要缓存',
  },
]);

const auditEntries = ref<AuditEntry[]>([
  { id: 'a1', event: '撤销公开链接：旧版报价单', actor: '管理员', risk: '高风险' as RiskLevel, reverted: false, rollback: '恢复链接撤销前状态' },
  { id: 'a2', event: '调整访客文件夹 ACL', actor: '权限中心', risk: '中风险' as RiskLevel, reverted: false, rollback: '恢复原 ACL' },
  { id: 'a3', event: '项目资料使用云模型摘要', actor: 'AI 数据层', risk: '低风险' as RiskLevel, reverted: false, rollback: '删除模型调用记录' },
]);

const riskFilter = ref<RiskFilter>('全部');
const selectedRiskId = ref(riskActions.value[0].id);
const eventState = ref('安全治理层正在同步身份、权限、分享与 AI 可见性策略。');
const loading = ref(false);
const busyActionId = ref<string | null>(null);
const busyIdentityId = ref<string | null>(null);
const busyPolicyId = ref<string | null>(null);
const busyShareId = ref<string | null>(null);
const busyAuditId = ref<string | null>(null);

const filteredRisks = computed(() =>
  riskFilter.value === '全部' ? riskActions.value : riskActions.value.filter((item) => item.level === riskFilter.value),
);
const selectedRisk = computed(() => riskActions.value.find((item) => item.id === selectedRiskId.value) ?? riskActions.value[0]);
const activeShareCount = computed(() => shareLinks.value.filter((item) => item.active).length);
const highRiskPending = computed(() => riskActions.value.filter((item) => item.level === '高风险' && item.state === '待处理').length);

function setRiskFilter(filter: RiskFilter) {
  riskFilter.value = filter;
  selectedRiskId.value = filteredRisks.value[0]?.id ?? selectedRiskId.value;
  eventState.value = `已筛选 ${filter} 动作，当前显示 ${filteredRisks.value.length} 条。`;
}

function selectRisk(id: string) {
  selectedRiskId.value = id;
  eventState.value = `正在查看风险动作：${selectedRisk.value.title}`;
}

async function loadSecurityState() {
  loading.value = true;
  try {
    const [nextIdentities, nextPolicies, nextRisks, nextShares, nextAudit] = await Promise.all([
      apiClient.security.getIdentities(),
      apiClient.security.getAiPolicies(),
      apiClient.security.getRiskActions(),
      apiClient.security.getShares(),
      apiClient.security.getAudit(),
    ]);
    identities.value = nextIdentities;
    aiPolicies.value = nextPolicies;
    riskActions.value = nextRisks;
    shareLinks.value = nextShares;
    auditEntries.value = nextAudit;
    selectedRiskId.value = filteredRisks.value[0]?.id ?? nextRisks[0]?.id ?? '';
    eventState.value = '安全中心已连接后端，身份、AI 可见性、风险动作、分享和审计已同步。';
  } catch (error) {
    eventState.value = `后端暂不可用，继续使用本地缓存：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function confirmRisk(action: RiskAction) {
  busyActionId.value = action.id;
  try {
    await apiClient.security.confirmRiskAction(action.id, { actorId: 'security-center' });
    await refreshRisksAndAudit();
    eventState.value = `${action.level}动作已确认，已写入审计并保留回滚方式。`;
  } catch (error) {
    eventState.value = `确认失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyActionId.value = null;
  }
}

async function blockRisk(action: RiskAction) {
  busyActionId.value = action.id;
  try {
    await apiClient.security.blockRiskAction(action.id, { actorId: 'security-center', reason: '用户在安全中心阻止' });
    await refreshRisksAndAudit();
    eventState.value = `${action.title} 已阻止，Agent 无法继续跨边界执行。`;
  } catch (error) {
    eventState.value = `阻止失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyActionId.value = null;
  }
}

async function revokeShare(id: string) {
  const link = shareLinks.value.find((item) => item.id === id);
  if (!link || !link.active) return;
  busyShareId.value = id;
  try {
    await apiClient.security.deleteShare(id);
    shareLinks.value = await apiClient.security.getShares();
    auditEntries.value = await apiClient.security.getAudit();
    eventState.value = `${link.name} 已撤销，下载入口立即失效。`;
  } catch (error) {
    eventState.value = `撤销失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyShareId.value = null;
  }
}

async function recordPermissionChange(identity: IdentityPolicy, field: string, enabled: boolean) {
  const id = identity.id ?? identity.role;
  busyIdentityId.value = id;
  try {
    const updated = await apiClient.security.updateIdentityPermissions(id, {
      mfa: identity.mfa,
      fileAcl: identity.fileAcl,
      appAdmin: identity.appAdmin,
      aiTools: identity.aiTools,
    });
    identities.value = identities.value.map((item) => ((item.id ?? item.role) === id ? updated : item));
    auditEntries.value = await apiClient.security.getAudit();
    eventState.value = `${identity.role} 的 ${field} 已${enabled ? '开启' : '关闭'}，权限快照已更新。`;
  } catch (error) {
    eventState.value = `权限同步失败：${error instanceof Error ? error.message : 'unknown error'}`;
    await refreshIdentities();
  } finally {
    busyIdentityId.value = null;
  }
}

async function recordAiPolicy(policy: AiPolicy, field: string, enabled: boolean) {
  const id = policy.id ?? policy.space;
  busyPolicyId.value = id;
  try {
    const updated = await apiClient.security.updateAiPolicy(id, {
      indexed: policy.indexed,
      cloudModel: policy.cloudModel,
      sensitive: policy.sensitive,
    });
    aiPolicies.value = aiPolicies.value.map((item) => ((item.id ?? item.space) === id ? updated : item));
    auditEntries.value = await apiClient.security.getAudit();
    eventState.value = `${policy.space} 的 ${field} 已${enabled ? '允许' : '禁止'}，AI 索引可见性同步变更。`;
  } catch (error) {
    eventState.value = `AI 策略同步失败：${error instanceof Error ? error.message : 'unknown error'}`;
    await refreshPolicies();
  } finally {
    busyPolicyId.value = null;
  }
}

async function rollbackAudit(id: string) {
  const entry = auditEntries.value.find((item) => item.id === id);
  if (!entry || entry.reverted) return;
  busyAuditId.value = id;
  try {
    await apiClient.security.rollbackAudit(id, { actorId: 'security-center' });
    auditEntries.value = await apiClient.security.getAudit();
    eventState.value = `${entry.event} 已回滚：${entry.rollback}`;
  } catch (error) {
    eventState.value = `回滚失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAuditId.value = null;
  }
}

async function refreshRisksAndAudit() {
  const [nextRisks, nextAudit] = await Promise.all([
    apiClient.security.getRiskActions(),
    apiClient.security.getAudit(),
  ]);
  riskActions.value = nextRisks;
  auditEntries.value = nextAudit;
}

async function refreshIdentities() {
  identities.value = await apiClient.security.getIdentities();
}

async function refreshPolicies() {
  aiPolicies.value = await apiClient.security.getAiPolicies();
}

onMounted(loadSecurityState);
</script>

<template>
  <div class="security-center">
    <section class="security-center__overview" aria-label="安全中心概览">
      <div>
        <ShieldCheck :size="18" />
        <span>{{ loading ? '同步中' : '身份策略' }}</span>
        <strong>{{ identities.length }} 组</strong>
      </div>
      <div>
        <Link2Off :size="18" />
        <span>有效外链</span>
        <strong>{{ activeShareCount }}</strong>
      </div>
      <div>
        <ShieldAlert :size="18" />
        <span>高风险待确认</span>
        <strong>{{ highRiskPending }}</strong>
      </div>
      <div>
        <History :size="18" />
        <span>审计记录</span>
        <strong>{{ auditEntries.length }}</strong>
      </div>
    </section>

    <section class="security-center__risks" aria-label="风险分级与确认">
      <header>
        <h3><Filter :size="15" /> 风险分级</h3>
        <UiSegmented
          :model-value="riskFilter"
          :options="riskFilterOptions"
          size="sm"
          @change="(v) => setRiskFilter(v as RiskFilter)"
        />
      </header>

      <div class="security-center__risk-list">
        <article
          v-for="action in filteredRisks"
          :key="action.id"
          class="security-center__risk-card"
          :class="{ 'security-center__risk-card--active': selectedRiskId === action.id }"
          @click="selectRisk(action.id)"
        >
          <div>
            <h4>{{ action.title }}</h4>
            <p>{{ action.scope }} · {{ action.actor }}</p>
          </div>
          <UiBadge class="security-center__risk" :tone="riskTone[action.level]">{{ action.level }}</UiBadge>
          <small>{{ action.state }} · {{ action.rollback }}</small>
          <div class="security-center__risk-actions">
            <UiButton
              size="sm"
              :icon-left="CheckCircle2"
              :disabled="busyActionId === action.id || action.state !== '待处理'"
              @click.stop="confirmRisk(action)"
            >
              {{ busyActionId === action.id ? '处理中' : action.level !== '低风险' ? '确认' : '记录' }}
            </UiButton>
            <UiButton
              variant="ghost"
              tone="danger"
              size="sm"
              :disabled="busyActionId === action.id || action.state !== '待处理'"
              @click.stop="blockRisk(action)"
            >
              阻止
            </UiButton>
          </div>
        </article>
      </div>
    </section>

    <section class="security-center__permissions" aria-label="身份与权限">
      <header>
        <h3><UserRoundCog :size="15" /> 身份与权限</h3>
        <span>{{ eventState }}</span>
      </header>
      <div class="security-center__permission-grid">
        <article v-for="identity in identities" :key="identity.role">
          <strong>{{ identity.role }}</strong>
          <small>{{ identity.name }}</small>
          <UiCheckbox
            v-model="identity.mfa"
            label="多因素认证"
            :disabled="busyIdentityId === (identity.id ?? identity.role)"
            @change="recordPermissionChange(identity, '多因素认证', identity.mfa)"
          />
          <UiCheckbox
            v-model="identity.fileAcl"
            label="文件 ACL"
            :disabled="busyIdentityId === (identity.id ?? identity.role)"
            @change="recordPermissionChange(identity, '文件 ACL', identity.fileAcl)"
          />
          <UiCheckbox
            v-model="identity.appAdmin"
            label="应用管理"
            :disabled="busyIdentityId === (identity.id ?? identity.role)"
            @change="recordPermissionChange(identity, '应用管理', identity.appAdmin)"
          />
          <UiCheckbox
            v-model="identity.aiTools"
            label="Agent 工具"
            :disabled="busyIdentityId === (identity.id ?? identity.role)"
            @change="recordPermissionChange(identity, 'Agent 工具', identity.aiTools)"
          />
        </article>
      </div>
    </section>

    <section class="security-center__ai" aria-label="AI 数据访问控制">
      <header>
        <h3><EyeOff :size="15" /> AI 数据访问控制</h3>
        <span>{{ selectedRisk.title }}</span>
      </header>
      <div class="security-center__policy-list">
        <article v-for="policy in aiPolicies" :key="policy.space">
          <div>
            <strong>{{ policy.space }}</strong>
            <small>{{ policy.sensitive }}</small>
          </div>
          <UiCheckbox
            v-model="policy.indexed"
            label="AI 索引"
            :disabled="busyPolicyId === (policy.id ?? policy.space)"
            @change="recordAiPolicy(policy, 'AI 索引', policy.indexed)"
          />
          <UiCheckbox
            v-model="policy.cloudModel"
            label="云模型"
            :disabled="busyPolicyId === (policy.id ?? policy.space)"
            @change="recordAiPolicy(policy, '云模型调用', policy.cloudModel)"
          />
        </article>
      </div>
    </section>

    <section class="security-center__shares" aria-label="分享链接安全检查">
      <header>
        <h3><KeyRound :size="15" /> 分享链接安全检查</h3>
        <span>公开分享、密码、有效期</span>
      </header>
      <div class="security-center__share-list">
        <article v-for="link in shareLinks" :key="link.id" :class="{ 'security-center__share--revoked': !link.active }">
          <div>
            <strong>{{ link.name }}</strong>
            <small>{{ link.target }} · {{ link.access }} · 下载 {{ link.downloads }}</small>
          </div>
          <UiBadge class="security-center__risk" :tone="link.active ? riskTone[link.risk] : 'neutral'">{{ link.active ? link.risk : '已撤销' }}</UiBadge>
          <UiButton
            variant="soft"
            tone="danger"
            size="sm"
            :icon-left="Link2Off"
            :disabled="!link.active || busyShareId === link.id"
            @click="revokeShare(link.id)"
          >
            {{ busyShareId === link.id ? '处理中' : '撤销' }}
          </UiButton>
        </article>
      </div>
    </section>

    <section class="security-center__audit" aria-label="审计与回滚">
      <header>
        <h3><ArchiveRestore :size="15" /> 审计与回滚</h3>
        <span>输入、工具、影响范围、确认与撤销</span>
      </header>
      <div class="security-center__audit-list">
        <article v-for="entry in auditEntries" :key="entry.id">
          <FileWarning :size="15" />
          <div>
            <strong>{{ entry.event }}</strong>
            <small>{{ entry.actor }} · {{ entry.risk }} · {{ entry.reverted ? '已回滚' : entry.rollback }}</small>
          </div>
          <UiButton
            variant="soft"
            tone="danger"
            size="sm"
            :disabled="entry.reverted || busyAuditId === entry.id || !entry.rollback"
            @click="rollbackAudit(entry.id)"
          >
            {{ busyAuditId === entry.id ? '处理中' : '回滚' }}
          </UiButton>
        </article>
      </div>
    </section>
    <NasFeaturePanel class="security-center__features" :modules="['security', 'files']" />
  </div>
</template>

<style scoped>
.security-center {
  display: grid;
  grid-template-columns: minmax(260px, 1.1fr) minmax(0, 1fr);
  grid-template-rows: auto minmax(0, 1fr) minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.security-center__features {
  grid-column: 1 / -1;
}

.security-center__overview,
.security-center__risks,
.security-center__permissions,
.security-center__ai,
.security-center__shares,
.security-center__audit {
  min-height: 0;
  background: rgba(var(--surface-rgb),  0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.security-center__overview {
  grid-column: 1 / -1;
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
}

.security-center__overview div {
  display: grid;
  gap: 3px;
  justify-items: center;
  padding: 10px 6px;
  color: var(--accent);
  background: rgba(var(--surface-rgb),  0.44);
}

.security-center__overview span {
  color: var(--text-soft);
  font-size: 10px;
}

.security-center__overview strong {
  color: var(--text-strong);
  font-size: 13px;
}

.security-center header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
  padding: 11px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.security-center h3 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
  white-space: nowrap;
}

.security-center header span {
  min-width: 0;
  overflow: hidden;
  color: var(--text-soft);
  font-size: 11px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.security-center__risks {
  grid-row: 2 / span 2;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.security-center__risk-list,
.security-center__share-list,
.security-center__audit-list,
.security-center__policy-list {
  display: grid;
  align-content: start;
  gap: 9px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.security-center__risk-card,
.security-center__permission-grid article,
.security-center__policy-list article,
.security-center__share-list article,
.security-center__audit-list article {
  min-width: 0;
  background: rgba(var(--surface-rgb),  0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.security-center__risk-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  padding: 11px;
  cursor: pointer;
}

.security-center__risk-card--active {
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.security-center__risk-card h4,
.security-center__risk-card p {
  margin: 0;
}

.security-center__risk-card h4,
.security-center__permission-grid strong,
.security-center__policy-list strong,
.security-center__share-list strong,
.security-center__audit-list strong {
  color: var(--text-strong);
  font-size: 12px;
}

.security-center__risk-card p,
.security-center__risk-card small,
.security-center__permission-grid small,
.security-center__policy-list small,
.security-center__share-list small,
.security-center__audit-list small {
  overflow-wrap: anywhere;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.security-center__risk-card p {
  margin-top: 5px;
}

.security-center__risk-card small,
.security-center__risk-actions {
  grid-column: 1 / -1;
}

.security-center__risk-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.security-center__risk {
  justify-self: end;
}

.security-center__permissions,
.security-center__ai,
.security-center__shares,
.security-center__audit {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.security-center__permission-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 9px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.security-center__permission-grid article {
  display: grid;
  align-content: start;
  gap: 7px;
  padding: 10px;
}

.security-center__policy-list article,
.security-center__share-list article,
.security-center__audit-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 9px;
  padding: 10px;
}

.security-center__policy-list article {
  grid-template-columns: minmax(0, 1fr) auto auto;
}

.security-center__share--revoked {
  opacity: 0.62;
}

.security-center__audit-list article {
  grid-template-columns: auto minmax(0, 1fr) auto;
}

.security-center__audit-list svg {
  color: #b36a00;
}

@media (max-width: 860px) {
  .security-center {
    grid-template-columns: 1fr;
    grid-template-rows: auto;
    overflow: auto;
  }

  .security-center__overview,
  .security-center__risks {
    grid-column: auto;
    grid-row: auto;
  }

  .security-center__overview,
  .security-center__permission-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .security-center__policy-list article,
  .security-center__share-list article,
  .security-center__audit-list article {
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
  }

  .security-center__risk {
    justify-self: start;
  }
}
</style>
