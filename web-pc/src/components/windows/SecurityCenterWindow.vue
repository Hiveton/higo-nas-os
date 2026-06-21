<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  CheckCircle2,
  FileWarning,
  Flame,
  History,
  Link2Off,
  Network,
  Radar,
  ShieldAlert,
  ShieldCheck,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type {
  AiPolicy,
  AuditEntry,
  FileShare,
  FirewallState,
  IdentityPolicy,
  ListeningPort,
  RiskAction,
  RiskLevel,
} from '../../api/types';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import ActivityLogPanel from './security/ActivityLogPanel.vue';
import { UiBadge, UiButton, UiCheckbox, UiSegmented, UiWindowPage, UiStatGrid, UiStat, UiPanel } from '../ui';
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

const hostPorts = ref<ListeningPort[]>([]);
const firewall = ref<FirewallState | null>(null);
const scanningHost = ref(false);
const openToAll = computed(() => hostPorts.value.filter((p) => p.exposure === '公开监听').length);

function portTone(risk: string): UiTone {
  if (risk === 'high') return 'danger';
  if (risk === 'medium') return 'warning';
  return 'success';
}

async function rescanHost() {
  scanningHost.value = true;
  try {
    const result = await apiClient.security.scanHost();
    hostPorts.value = result.ports ?? [];
    firewall.value = result.firewall ?? null;
    eventState.value = `主机扫描完成：${result.ports?.length ?? 0} 个监听端口，${result.openToAll} 个对外暴露，防火墙${result.firewall?.active ? '已启用' : '未启用'}。`;
  } catch (error) {
    eventState.value = `主机扫描失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    scanningHost.value = false;
  }
}

async function loadSecurityState() {
  loading.value = true;
  try {
    const [nextIdentities, nextPolicies, nextRisks, nextShares, nextAudit, nextPorts, nextFirewall] = await Promise.all([
      apiClient.security.getIdentities(),
      apiClient.security.getAiPolicies(),
      apiClient.security.getRiskActions(),
      apiClient.security.getShares(),
      apiClient.security.getAudit(),
      apiClient.security.getHostPorts(),
      apiClient.security.getFirewall(),
    ]);
    identities.value = nextIdentities;
    aiPolicies.value = nextPolicies;
    riskActions.value = nextRisks;
    shareLinks.value = nextShares;
    auditEntries.value = nextAudit;
    hostPorts.value = nextPorts;
    firewall.value = nextFirewall;
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
  <UiWindowPage
    layout="master-detail"
    :icon="ShieldCheck"
    title="安全中心"
    subtitle="身份 · 权限 · 分享 · AI 可见性"
    :status="eventState"
  >
    <template #toolbar>
      <UiSegmented
        :model-value="riskFilter"
        :options="riskFilterOptions"
        size="sm"
        @change="(v) => setRiskFilter(v as RiskFilter)"
      />
    </template>

    <template #inspector>
      <UiPanel title="身份与权限" framed>
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
      </UiPanel>
    </template>

    <UiStatGrid>
      <UiStat :icon="ShieldCheck" :label="loading ? '同步中' : '身份策略'" :value="`${identities.length} 组`" tone="primary" />
      <UiStat :icon="Link2Off" label="有效外链" :value="activeShareCount" tone="info" />
      <UiStat :icon="ShieldAlert" label="高风险待确认" :value="highRiskPending" :tone="highRiskPending > 0 ? 'danger' : 'success'" />
      <UiStat :icon="History" label="审计记录" :value="auditEntries.length" tone="neutral" />
    </UiStatGrid>

    <UiPanel title="风险分级" framed>
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
    </UiPanel>

    <UiPanel title="AI 数据访问控制" framed>
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
    </UiPanel>

    <UiPanel title="分享链接安全检查" framed>
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
    </UiPanel>

    <UiPanel title="主机安全扫描" framed>
      <template #actions>
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="Radar" :loading="scanningHost" @click="rescanHost">
          重新扫描
        </UiButton>
      </template>

      <div class="security-center__host-summary">
        <div class="host-stat">
          <Network :size="16" />
          <div>
            <strong>{{ hostPorts.length }}</strong>
            <small>监听端口</small>
          </div>
        </div>
        <div class="host-stat" :class="{ 'host-stat--warn': openToAll > 0 }">
          <ShieldAlert :size="16" />
          <div>
            <strong>{{ openToAll }}</strong>
            <small>对外暴露</small>
          </div>
        </div>
        <div class="host-stat" :class="firewall?.active ? 'host-stat--ok' : 'host-stat--warn'">
          <Flame :size="16" />
          <div>
            <strong>{{ firewall?.active ? '已启用' : '未启用' }}</strong>
            <small>{{ firewall?.backend ?? '防火墙' }}</small>
          </div>
        </div>
      </div>
      <p v-if="firewall?.summary" class="security-center__host-fw">{{ firewall.summary }}</p>

      <div class="security-center__port-list">
        <article v-for="p in hostPorts" :key="`${p.protocol}-${p.address}-${p.port}`" class="port-row">
          <div class="port-row__main">
            <strong>{{ p.address }}:{{ p.port }}</strong>
            <small>{{ p.protocol.toUpperCase() }} · {{ p.process || '未知进程' }}</small>
          </div>
          <div class="port-row__meta">
            <UiBadge tone="neutral" variant="soft" size="sm">{{ p.exposure }}</UiBadge>
            <UiBadge :tone="portTone(p.risk)" variant="dot" size="sm">{{ p.risk }}</UiBadge>
          </div>
        </article>
        <p v-if="!hostPorts.length" class="security-center__port-empty">暂无监听端口数据，点击「重新扫描」获取。</p>
      </div>
    </UiPanel>

    <UiPanel title="审计与回滚" framed>
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
    </UiPanel>

    <UiPanel title="操作记录" framed>
      <div class="security-center__activity-body">
        <ActivityLogPanel />
      </div>
    </UiPanel>

    <NasFeaturePanel class="security-center__features" :modules="['security', 'files']" />
  </UiWindowPage>
</template>

<style scoped>
.security-center__host-summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
}

.host-stat {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  color: var(--text-muted);
}

.host-stat strong {
  display: block;
  color: var(--text-strong);
  font-size: var(--fs-lg);
}

.host-stat small {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.host-stat--warn {
  border-color: color-mix(in srgb, var(--accent-orange) 40%, var(--border));
  color: var(--ink-orange);
}

.host-stat--ok {
  border-color: color-mix(in srgb, var(--accent-green) 40%, var(--border));
}

.security-center__host-fw {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.security-center__port-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
  max-height: 220px;
  overflow: auto;
}

.port-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.port-row__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.port-row__main strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-family: var(--font-mono, monospace);
}

.port-row__main small {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.port-row__meta {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.security-center__port-empty {
  margin: 0;
  padding: 12px;
  text-align: center;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.security-center__activity-body {
  min-height: 240px;
}

.security-center__risk-list,
.security-center__share-list,
.security-center__audit-list,
.security-center__policy-list {
  display: grid;
  align-content: start;
  gap: 9px;
  min-height: 0;
}

.security-center__risk-card,
.security-center__permission-grid article,
.security-center__policy-list article,
.security-center__share-list article,
.security-center__audit-list article {
  min-width: 0;
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.security-center__risk-card {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 8px;
  padding: 11px;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard), border-color var(--duration-fast) var(--ease-standard);
}

.security-center__risk-card:hover {
  background: var(--accent-soft);
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
  font-size: var(--fs-xs);
}

.security-center__risk-card p,
.security-center__risk-card small,
.security-center__permission-grid small,
.security-center__policy-list small,
.security-center__share-list small,
.security-center__audit-list small {
  overflow-wrap: anywhere;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
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

.security-center__permission-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 9px;
  min-height: 0;
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
  color: var(--ink-orange);
}

@container desktop-window-body (max-width: 760px) {
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
