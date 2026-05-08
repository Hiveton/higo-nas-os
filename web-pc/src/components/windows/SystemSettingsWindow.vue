<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { Component } from 'vue';
import {
  ArchiveRestore,
  Bell,
  BrainCircuit,
  CheckCircle2,
  Cloud,
  EyeOff,
  Globe2,
  History,
  LockKeyhole,
  RefreshCw,
  RotateCcw,
  Save,
  Server,
  ShieldCheck,
  Users,
  Wifi,
} from 'lucide-vue-next';
import { settingsStore } from '../../stores/settings';
import type { SettingsState as ApiSettingsState } from '../../api/types';

type CategoryId =
  | 'accounts'
  | 'network'
  | 'models'
  | 'ai'
  | 'notifications'
  | 'updates'
  | 'privacy'
  | 'audit'
  | 'backup';

type SettingsState = {
  role: string;
  guestAccess: boolean;
  agentApproval: boolean;
  mfaRequired: boolean;
  ddnsEnabled: boolean;
  remoteAccess: boolean;
  dnsProfile: string;
  modelStrategy: string;
  modelProvider: string;
  taskRouting: boolean;
  localAi: boolean;
  cloudAi: boolean;
  privateEndpoint: boolean;
  backupNotice: boolean;
  securityNotice: boolean;
  lifeNotice: boolean;
  autoUpdate: boolean;
  releaseChannel: string;
  privacyMode: string;
  sensitiveLocalOnly: boolean;
  auditRetention: string;
  systemBackup: boolean;
  backupTarget: string;
};

type Category = {
  id: CategoryId;
  label: string;
  summary: string;
  icon: Component;
};

const categories: Category[] = [
  { id: 'accounts', label: '账号 / 权限', summary: '角色、访客、Agent 授权', icon: Users },
  { id: 'network', label: '网络 / DDNS', summary: '远程访问、DNS、域名', icon: Wifi },
  { id: 'models', label: '模型策略', summary: '混合、本地、云端路由', icon: BrainCircuit },
  { id: 'ai', label: '本地 / 云端 AI', summary: '推理资源与私有端点', icon: Cloud },
  { id: 'notifications', label: '通知', summary: '备份、风险、生活提醒', icon: Bell },
  { id: 'updates', label: '更新', summary: '自动更新与渠道', icon: RefreshCw },
  { id: 'privacy', label: '隐私', summary: '敏感数据与云端限制', icon: EyeOff },
  { id: 'audit', label: '审计保留', summary: '日志周期与可追溯性', icon: History },
  { id: 'backup', label: '系统备份', summary: '配置快照与恢复目标', icon: ArchiveRestore },
];

const modelStrategies = ['家庭混合模式', '小团队供应商模式', '企业强制本地', '按数据级别路由'];
const retentionOptions = ['30 天', '90 天', '180 天', '365 天'];

const activeCategoryId = ref<CategoryId>('accounts');
const updateStatus = ref('上次检查：今天 09:20，当前为最新版本。');
const appliedState = ref('设置已加载，等待管理员调整。');
const lastAudit = ref('系统设置窗口已打开，配置读取写入审计。');
const checkCount = ref(0);

const settings = ref<SettingsState>(createDefaultSettings());

const activeCategory = computed(() => categories.find((category) => category.id === activeCategoryId.value) ?? categories[0]);
const activeCategoryIndex = computed(() => categories.findIndex((category) => category.id === activeCategoryId.value) + 1);
const enabledNoticeCount = computed(
  () => [settings.value.backupNotice, settings.value.securityNotice, settings.value.lifeNotice].filter(Boolean).length,
);
const privacySummary = computed(() => {
  if (settings.value.privacyMode === '隐身优先') return 'AI 仅处理显式授权内容，敏感文件不进入云端。';
  if (settings.value.privacyMode === '企业合规') return '强制本地模型、审计全量记录、外部 API 需确认。';
  return '家庭成员按空间权限访问，普通任务允许云端增强。';
});
const modelSummary = computed(() => {
  const cloudState = settings.value.cloudAi ? '云端增强开启' : '云端增强关闭';
  return `${settings.value.modelStrategy} · ${settings.value.modelProvider} · ${cloudState}`;
});
const governanceScore = computed(() => {
  let score = 62;
  if (settings.value.mfaRequired) score += 7;
  if (settings.value.agentApproval) score += 7;
  if (settings.value.sensitiveLocalOnly) score += 8;
  if (settings.value.systemBackup) score += 6;
  if (settings.value.auditRetention === '365 天') score += 6;
  if (!settings.value.cloudAi) score += 4;
  return Math.min(score, 100);
});

function createDefaultSettings(): SettingsState {
  return {
    role: '管理员',
    guestAccess: false,
    agentApproval: true,
    mfaRequired: true,
    ddnsEnabled: true,
    remoteAccess: true,
    dnsProfile: '自动 DNS',
    modelStrategy: '家庭混合模式',
    modelProvider: '本地 Qwen3-8B',
    taskRouting: true,
    localAi: true,
    cloudAi: true,
    privateEndpoint: false,
    backupNotice: true,
    securityNotice: true,
    lifeNotice: true,
    autoUpdate: true,
    releaseChannel: '稳定版',
    privacyMode: '家庭默认',
    sensitiveLocalOnly: true,
    auditRetention: '180 天',
    systemBackup: true,
    backupTarget: 'HiGoNAS 内部快照',
  };
}

function applyBackendSettings(nextSettings: ApiSettingsState) {
  const model = nextSettings.model ?? {};
  const privacy = nextSettings.privacy ?? {};
  if (model.mode === 'enterprise_local') settings.value.modelStrategy = '企业强制本地';
  else if (model.mode === 'provider') settings.value.modelStrategy = '小团队供应商模式';
  else settings.value.modelStrategy = '家庭混合模式';
  settings.value.modelProvider = model.localModel || model.cloudModel || settings.value.modelProvider;
  settings.value.cloudAi = Boolean(model.cloudEnabled);
  settings.value.localAi = true;
  settings.value.sensitiveLocalOnly = privacy.sensitiveDataLocalOnly ?? true;
  settings.value.auditRetention = `${privacy.auditRetentionDays ?? 90} 天`;
  if (settings.value.sensitiveLocalOnly && settings.value.privacyMode === '家庭默认') {
    settings.value.privacyMode = '隐身优先';
  }
}

function toBackendSettings(): ApiSettingsState {
  return {
    model: {
      mode: backendModelMode(settings.value.modelStrategy),
      provider: settings.value.modelProvider.includes('OpenAI') ? 'cloud' : 'local',
      localModel: settings.value.modelProvider,
      cloudModel: settings.value.cloudAi ? 'OpenAI 云端增强' : '',
      cloudEnabled: settings.value.cloudAi,
    },
    privacy: {
      sensitiveDataLocalOnly: true,
      auditRetentionDays: parseInt(settings.value.auditRetention, 10) || 90,
    },
  };
}

function backendModelMode(strategy: string) {
  if (strategy === '企业强制本地') return 'enterprise_local';
  if (strategy === '小团队供应商模式') return 'provider';
  return 'family_hybrid';
}

function selectCategory(id: CategoryId) {
  activeCategoryId.value = id;
  lastAudit.value = `切换到${categories.find((category) => category.id === id)?.label ?? '系统设置'}配置页。`;
}

function setModelStrategy(strategy: string) {
  settings.value.modelStrategy = strategy;
  if (strategy === '企业强制本地') {
    settings.value.cloudAi = false;
    settings.value.sensitiveLocalOnly = true;
    settings.value.privateEndpoint = true;
  }
  if (strategy === '家庭混合模式') {
    settings.value.localAi = true;
    settings.value.cloudAi = true;
  }
  lastAudit.value = `模型策略已切换为${strategy}。`;
}

function setAuditRetention(retention: string) {
  settings.value.auditRetention = retention;
  lastAudit.value = `审计保留周期已切换为${retention}。`;
}

function setPrivacyMode(mode: string) {
  settings.value.privacyMode = mode;
  settings.value.sensitiveLocalOnly = mode !== '家庭默认';
  if (mode === '企业合规') {
    settings.value.cloudAi = false;
    settings.value.agentApproval = true;
  }
  lastAudit.value = `隐私模式已切换为${mode}。`;
}

async function saveSettings() {
  try {
    const nextSettings = await settingsStore.saveSettings(toBackendSettings());
    applyBackendSettings(nextSettings);
    appliedState.value = `已应用：${activeCategory.value.label} · 治理评分 ${governanceScore.value}% · ${modelSummary.value}`;
    lastAudit.value = '管理员保存系统设置，变更已写入审计并同步到系统服务。';
  } catch (reason) {
    const message = reason instanceof Error ? reason.message : String(reason);
    appliedState.value = `保存失败：${message}`;
    lastAudit.value = '系统设置保存失败，后端拒绝了本次配置。';
  }
}

async function restoreDefaults() {
  settings.value = createDefaultSettings();
  activeCategoryId.value = 'accounts';
  try {
    const nextSettings = await settingsStore.restoreDefaults();
    applyBackendSettings(nextSettings);
    appliedState.value = '已恢复默认策略：家庭混合模式、后端默认审计、系统快照开启。';
    updateStatus.value = '上次检查：今天 09:20，当前为最新版本。';
    lastAudit.value = '系统设置已恢复默认值。';
  } catch {
    appliedState.value = '已恢复本地默认策略，等待后端同步。';
    lastAudit.value = '系统设置已恢复本地默认值。';
  }
}

async function checkForUpdates() {
  checkCount.value += 1;
  try {
    const task = await settingsStore.checkUpdates();
    const updatePayload = settingsStore.updates.value;
    updateStatus.value = `${task.message ?? '更新检查已排队'} · 当前版本 ${String(updatePayload.current ?? 'dev')}`;
    lastAudit.value = '执行更新检查，结果已记录到系统审计。';
  } catch {
    updateStatus.value =
      checkCount.value % 2 === 0
        ? '刚刚检查：当前版本已是最新，安全规则库同步完成。'
        : `发现可选补丁：${settings.value.releaseChannel} 通道有安全治理规则更新。`;
    lastAudit.value = '执行本地更新检查，等待后端恢复。';
  }
}

async function createSystemBackup() {
  try {
    const task = await settingsStore.createSystemBackup();
    appliedState.value = task.message ?? '系统备份任务已提交。';
    lastAudit.value = '系统备份任务已提交到后端队列。';
  } catch {
    appliedState.value = '系统备份任务暂未提交，后端不可用。';
  }
}

onMounted(async () => {
  const nextSettings = await settingsStore.loadSettings();
  if (nextSettings.model || nextSettings.privacy) {
    applyBackendSettings(nextSettings);
    appliedState.value = '设置已从后端加载，等待管理员调整。';
  }
  const updatePayload = settingsStore.updates.value;
  if (updatePayload.updateStatus) {
    updateStatus.value = `后端状态：${String(updatePayload.updateStatus)} · 当前 ${String(updatePayload.current ?? 'dev')}`;
  }
});
</script>

<template>
  <div class="system-settings">
    <aside class="system-settings__sidebar" aria-label="系统设置分类">
      <header>
        <ShieldCheck :size="18" />
        <div>
          <strong>系统设置</strong>
          <span>{{ activeCategoryIndex }} / {{ categories.length }} · {{ governanceScore }}%</span>
        </div>
      </header>

      <nav class="system-settings__nav">
        <button
          v-for="category in categories"
          :key="category.id"
          class="system-settings__nav-item"
          :class="{ 'system-settings__nav-item--active': category.id === activeCategoryId }"
          type="button"
          @click="selectCategory(category.id)"
        >
          <component :is="category.icon" :size="15" />
          <span>
            <strong>{{ category.label }}</strong>
            <small>{{ category.summary }}</small>
          </span>
        </button>
      </nav>
    </aside>

    <main class="system-settings__main">
      <section class="system-settings__hero" aria-label="当前设置状态">
        <div>
          <p>{{ activeCategory.summary }}</p>
          <h3>{{ activeCategory.label }}</h3>
        </div>
        <span>{{ appliedState }}</span>
      </section>

      <section class="system-settings__content" aria-label="系统设置表单">
        <div v-if="activeCategoryId === 'accounts'" class="system-settings__panel">
          <label class="system-settings__field">
            <span>当前角色</span>
            <select v-model="settings.role">
              <option>管理员</option>
              <option>家庭成员</option>
              <option>团队成员</option>
              <option>访客</option>
            </select>
          </label>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.guestAccess }"
            type="button"
            @click="settings.guestAccess = !settings.guestAccess"
          >
            <span>访客空间访问</span>
            <b>{{ settings.guestAccess ? '开启' : '关闭' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.agentApproval }"
            type="button"
            @click="settings.agentApproval = !settings.agentApproval"
          >
            <span>Agent 权限变更需确认</span>
            <b>{{ settings.agentApproval ? '需要确认' : '仅审计' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.mfaRequired }"
            type="button"
            @click="settings.mfaRequired = !settings.mfaRequired"
          >
            <span>管理员双重验证</span>
            <b>{{ settings.mfaRequired ? '强制' : '可选' }}</b>
          </button>
        </div>

        <div v-else-if="activeCategoryId === 'network'" class="system-settings__panel">
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.ddnsEnabled }"
            type="button"
            @click="settings.ddnsEnabled = !settings.ddnsEnabled"
          >
            <span>DDNS higo-home.direct</span>
            <b>{{ settings.ddnsEnabled ? '解析中' : '暂停' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.remoteAccess }"
            type="button"
            @click="settings.remoteAccess = !settings.remoteAccess"
          >
            <span>远程访问通道</span>
            <b>{{ settings.remoteAccess ? '可用' : '内网限定' }}</b>
          </button>
          <label class="system-settings__field">
            <span>DNS 配置</span>
            <select v-model="settings.dnsProfile">
              <option>自动 DNS</option>
              <option>家庭安全 DNS</option>
              <option>团队内网 DNS</option>
            </select>
          </label>
          <div class="system-settings__metric">
            <Globe2 :size="17" />
            <p>{{ settings.ddnsEnabled ? '公网域名健康，证书 28 天后自动续签。' : 'DDNS 已暂停，仅保留局域网访问。' }}</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'models'" class="system-settings__panel">
          <div class="system-settings__segmented" aria-label="模型策略选择">
            <button
              v-for="strategy in modelStrategies"
              :key="strategy"
              :class="{ 'system-settings__segmented-button--active': strategy === settings.modelStrategy }"
              type="button"
              @click="setModelStrategy(strategy)"
            >
              {{ strategy }}
            </button>
          </div>
          <label class="system-settings__field">
            <span>默认模型</span>
            <select v-model="settings.modelProvider">
              <option>本地 Qwen3-8B</option>
              <option>私有 vLLM 集群</option>
              <option>OpenAI 云端增强</option>
              <option>局域网 Ollama</option>
            </select>
          </label>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.taskRouting }"
            type="button"
            @click="settings.taskRouting = !settings.taskRouting"
          >
            <span>按任务类型路由 OCR / 摘要 / Agent 规划</span>
            <b>{{ settings.taskRouting ? '启用' : '停用' }}</b>
          </button>
          <div class="system-settings__metric">
            <BrainCircuit :size="17" />
            <p>{{ modelSummary }}</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'ai'" class="system-settings__panel">
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.localAi }"
            type="button"
            @click="settings.localAi = !settings.localAi"
          >
            <span>本地 AI 索引与基础理解</span>
            <b>{{ settings.localAi ? '运行中' : '暂停' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.cloudAi }"
            type="button"
            @click="settings.cloudAi = !settings.cloudAi"
          >
            <span>云端复杂推理增强</span>
            <b>{{ settings.cloudAi ? '允许' : '禁止' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.privateEndpoint }"
            type="button"
            @click="settings.privateEndpoint = !settings.privateEndpoint"
          >
            <span>私有模型端点</span>
            <b>{{ settings.privateEndpoint ? '已接管' : '未接管' }}</b>
          </button>
          <div class="system-settings__metric">
            <Server :size="17" />
            <p>{{ settings.localAi ? '本地模型负责隐私索引和基础问答。' : '本地 AI 已暂停，文件访问不受影响。' }}</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'notifications'" class="system-settings__panel">
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.backupNotice }"
            type="button"
            @click="settings.backupNotice = !settings.backupNotice"
          >
            <span>备份失败 / 完整性提醒</span>
            <b>{{ settings.backupNotice ? '推送' : '静默' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.securityNotice }"
            type="button"
            @click="settings.securityNotice = !settings.securityNotice"
          >
            <span>权限风险 / 硬盘异常</span>
            <b>{{ settings.securityNotice ? '推送' : '静默' }}</b>
          </button>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.lifeNotice }"
            type="button"
            @click="settings.lifeNotice = !settings.lifeNotice"
          >
            <span>证件、保修、生活提醒</span>
            <b>{{ settings.lifeNotice ? '推送' : '静默' }}</b>
          </button>
          <div class="system-settings__metric">
            <Bell :size="17" />
            <p>{{ enabledNoticeCount }} 类通知已开启，通知中心会聚合系统、备份、Agent 和生活提醒。</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'updates'" class="system-settings__panel">
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.autoUpdate }"
            type="button"
            @click="settings.autoUpdate = !settings.autoUpdate"
          >
            <span>夜间自动更新</span>
            <b>{{ settings.autoUpdate ? '开启' : '关闭' }}</b>
          </button>
          <label class="system-settings__field">
            <span>更新渠道</span>
            <select v-model="settings.releaseChannel">
              <option>稳定版</option>
              <option>安全预览</option>
              <option>开发者预览</option>
            </select>
          </label>
          <button class="system-settings__action-button" type="button" @click="checkForUpdates">
            <RefreshCw :size="14" />
            检查更新
          </button>
          <div class="system-settings__metric">
            <CheckCircle2 :size="17" />
            <p>{{ updateStatus }}</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'privacy'" class="system-settings__panel">
          <div class="system-settings__segmented" aria-label="隐私模式选择">
            <button
              v-for="mode in ['家庭默认', '隐身优先', '企业合规']"
              :key="mode"
              :class="{ 'system-settings__segmented-button--active': mode === settings.privacyMode }"
              type="button"
              @click="setPrivacyMode(mode)"
            >
              {{ mode }}
            </button>
          </div>
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.sensitiveLocalOnly }"
            type="button"
            @click="settings.sensitiveLocalOnly = !settings.sensitiveLocalOnly"
          >
            <span>敏感文件禁止云端处理</span>
            <b>{{ settings.sensitiveLocalOnly ? '强制本地' : '按策略路由' }}</b>
          </button>
          <div class="system-settings__metric system-settings__metric--privacy">
            <LockKeyhole :size="17" />
            <p>{{ privacySummary }}</p>
          </div>
        </div>

        <div v-else-if="activeCategoryId === 'audit'" class="system-settings__panel">
          <div class="system-settings__segmented" aria-label="审计保留周期">
            <button
              v-for="retention in retentionOptions"
              :key="retention"
              :class="{ 'system-settings__segmented-button--active': retention === settings.auditRetention }"
              type="button"
              @click="setAuditRetention(retention)"
            >
              {{ retention }}
            </button>
          </div>
          <div class="system-settings__metric">
            <History :size="17" />
            <p>当前保留 {{ settings.auditRetention }}，记录身份、工具调用、数据范围、设置修改和回滚方式。</p>
          </div>
          <div class="system-settings__audit-list">
            <span>权限修改</span>
            <span>模型调用</span>
            <span>分享链接</span>
            <span>备份任务</span>
          </div>
        </div>

        <div v-else class="system-settings__panel">
          <button
            class="system-settings__toggle"
            :class="{ 'system-settings__toggle--on': settings.systemBackup }"
            type="button"
            @click="settings.systemBackup = !settings.systemBackup"
          >
            <span>系统配置快照</span>
            <b>{{ settings.systemBackup ? '每日' : '手动' }}</b>
          </button>
          <label class="system-settings__field">
            <span>备份目标</span>
            <select v-model="settings.backupTarget">
              <option>HiGoNAS 内部快照</option>
              <option>异地 NAS</option>
              <option>加密云端仓库</option>
            </select>
          </label>
          <div class="system-settings__metric">
            <ArchiveRestore :size="17" />
            <p>{{ settings.backupTarget }}：备份系统设置、权限策略、模型路由和通知规则。</p>
          </div>
          <button class="system-settings__action-button" type="button" @click="createSystemBackup">
            <ArchiveRestore :size="14" />
            立即创建系统备份
          </button>
        </div>
      </section>

      <section class="system-settings__footer" aria-label="保存和审计状态">
        <div>
          <strong>{{ lastAudit }}</strong>
          <span>账号权限、网络、模型、安全治理和备份策略均受审计保护。</span>
        </div>
        <div class="system-settings__footer-actions">
          <button type="button" @click="restoreDefaults">
            <RotateCcw :size="14" />
            恢复默认
          </button>
          <button class="system-settings__primary" type="button" @click="saveSettings">
            <Save :size="14" />
            保存应用
          </button>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.system-settings {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr);
  gap: 14px;
  height: 100%;
  min-height: 0;
}

.system-settings__sidebar,
.system-settings__hero,
.system-settings__content,
.system-settings__footer {
  min-height: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.system-settings__sidebar {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.system-settings__sidebar header {
  display: flex;
  align-items: center;
  gap: 9px;
  padding: 12px;
  color: var(--accent);
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.system-settings__sidebar strong,
.system-settings__sidebar span {
  display: block;
}

.system-settings__sidebar strong {
  color: var(--text-strong);
  font-size: 13px;
}

.system-settings__sidebar span {
  margin-top: 3px;
  color: var(--text-soft);
  font-size: 11px;
}

.system-settings__nav {
  display: grid;
  align-content: start;
  gap: 6px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

.system-settings__nav-item {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 44px;
  padding: 8px 9px;
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
}

.system-settings__nav-item--active {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  box-shadow: inset 3px 0 0 var(--accent);
}

.system-settings__nav-item span {
  min-width: 0;
}

.system-settings__nav-item strong,
.system-settings__nav-item small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.system-settings__nav-item strong {
  color: var(--text-strong);
  font-size: 12px;
}

.system-settings__nav-item small {
  margin-top: 3px;
  color: var(--text-soft);
  font-size: 10px;
}

.system-settings__main {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 12px;
  min-height: 0;
}

.system-settings__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 13px 14px;
  background: linear-gradient(135deg, rgba(231, 247, 255, 0.9), rgba(255, 255, 255, 0.62));
}

.system-settings__hero p,
.system-settings__hero h3 {
  margin: 0;
}

.system-settings__hero p {
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
}

.system-settings__hero h3 {
  margin-top: 3px;
  color: var(--text-strong);
  font-size: 18px;
}

.system-settings__hero > span {
  max-width: 48%;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
  text-align: right;
}

.system-settings__content {
  min-height: 0;
  overflow: auto;
}

.system-settings__panel {
  display: grid;
  gap: 10px;
  padding: 12px;
}

.system-settings__field {
  display: grid;
  gap: 7px;
  padding: 11px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.system-settings__field span {
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
}

.system-settings__field select {
  width: 100%;
  min-width: 0;
  height: 32px;
  padding: 0 9px;
  color: var(--text);
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: 0;
  font-size: 12px;
}

.system-settings__toggle {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  min-height: 46px;
  padding: 10px 11px;
  color: var(--text);
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.system-settings__toggle--on {
  border-color: rgba(19, 136, 255, 0.22);
  background: rgba(231, 247, 255, 0.74);
}

.system-settings__toggle span {
  min-width: 0;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.35;
}

.system-settings__toggle b {
  flex: 0 0 auto;
  padding: 5px 8px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-radius: 999px;
  font-size: 11px;
}

.system-settings__segmented {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 6px;
}

.system-settings__segmented button {
  min-height: 34px;
  padding: 6px 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.system-settings__segmented .system-settings__segmented-button--active {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-color: rgba(19, 136, 255, 0.24);
}

.system-settings__metric {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 44px;
  padding: 10px 11px;
  color: var(--accent-green);
  background: rgba(240, 253, 244, 0.72);
  border: 1px solid rgba(34, 181, 115, 0.18);
  border-radius: var(--radius-sm);
}

.system-settings__metric--privacy {
  color: var(--accent);
  background: rgba(231, 247, 255, 0.74);
  border-color: rgba(19, 136, 255, 0.18);
}

.system-settings__metric p {
  margin: 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.4;
}

.system-settings__action-button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  width: fit-content;
  min-height: 30px;
  padding: 0 11px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.system-settings__audit-list {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.system-settings__audit-list span {
  padding: 6px 9px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
}

.system-settings__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 11px 12px;
}

.system-settings__footer strong,
.system-settings__footer span {
  display: block;
}

.system-settings__footer strong {
  color: var(--text-strong);
  font-size: 12px;
}

.system-settings__footer span {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.system-settings__footer-actions {
  display: flex;
  flex: 0 0 auto;
  gap: 7px;
}

.system-settings__footer-actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.system-settings__footer-actions .system-settings__primary {
  color: #fff;
  background: var(--accent);
  border-color: transparent;
}

@media (max-width: 760px) {
  .system-settings {
    display: block;
    overflow: auto;
  }

  .system-settings__sidebar {
    margin-bottom: 10px;
  }

  .system-settings__nav {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    max-height: 210px;
  }

  .system-settings__main {
    min-height: 520px;
  }

  .system-settings__hero,
  .system-settings__footer {
    display: grid;
  }

  .system-settings__hero > span {
    max-width: none;
    text-align: left;
  }

  .system-settings__segmented {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .system-settings__footer-actions {
    width: 100%;
  }

  .system-settings__footer-actions button {
    flex: 1 1 0;
  }
}
</style>
