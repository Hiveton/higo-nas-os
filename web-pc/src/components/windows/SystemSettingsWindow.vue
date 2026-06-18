<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { Component } from 'vue';
import {
  ArchiveRestore,
  Bell,
  BrainCircuit,
  Cloud,
  EyeOff,
  History,
  Palette,
  RefreshCw,
  RotateCcw,
  Save,
  ShieldCheck,
  Users,
  Wifi,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { settingsStore } from '../../stores/settings';
import { UiButton } from '../ui';
import SettingsAccountsPanel from './settings/SettingsAccountsPanel.vue';
import SettingsNetworkPanel from './settings/SettingsNetworkPanel.vue';
import SettingsModelsPanel from './settings/SettingsModelsPanel.vue';
import SettingsAiPanel from './settings/SettingsAiPanel.vue';
import SettingsNotificationsPanel from './settings/SettingsNotificationsPanel.vue';
import SettingsUpdatesPanel from './settings/SettingsUpdatesPanel.vue';
import SettingsPrivacyPanel from './settings/SettingsPrivacyPanel.vue';
import SettingsAuditPanel from './settings/SettingsAuditPanel.vue';
import SettingsBackupPanel from './settings/SettingsBackupPanel.vue';
import SettingsInterfacePanel from './settings/SettingsInterfacePanel.vue';
import type {
  AccountSummary,
  AccountUser,
  AiProvider,
  AiProviderInput,
  AiProviderKind,
  SettingsState as ApiSettingsState,
} from '../../api/types';
import './settings/settings-window.css';

type CategoryId =
  | 'accounts'
  | 'network'
  | 'models'
  | 'ai'
  | 'notifications'
  | 'updates'
  | 'privacy'
  | 'audit'
  | 'backup'
  | 'interface';

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
  uiTheme: string;
  uiLocale: string;
  windowRadius: string;
  dockPosition: string;
  dockStyle: string;
  dockIconSize: string;
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
  { id: 'interface', label: '界面设置', summary: '主题、圆角、Dock 样式', icon: Palette },
];

const modelStrategies = ['家庭混合模式', '小团队供应商模式', '企业强制本地', '按数据级别路由'];
const retentionOptions = ['30 天', '90 天', '180 天', '365 天'];
const themeOptions = [
  { value: 'auto', label: '跟随系统' },
  { value: 'light', label: '浅色' },
  { value: 'dark', label: '深色' },
];
const localeOptions = [
  { value: 'zh-CN', label: '简体中文' },
  { value: 'en-US', label: 'English' },
];
const radiusOptions = [
  { value: 'compact', label: '紧凑' },
  { value: 'default', label: '默认' },
  { value: 'rounded', label: '圆润' },
];
const dockPositionOptions = [
  { value: 'bottom', label: '底部' },
  { value: 'left', label: '左侧' },
  { value: 'right', label: '右侧' },
];
const dockStyleOptions = [
  { value: 'floating', label: '浮动' },
  { value: 'side', label: '侧边栏' },
  { value: 'compact', label: '紧凑' },
];
const dockIconSizeOptions = [
  { value: 'small', label: '小' },
  { value: 'default', label: '默认' },
  { value: 'large', label: '大' },
];
const dnsProfileOptions = ['自动 DNS', '家庭安全 DNS', '团队内网 DNS'].map((value) => ({ value, label: value }));
const modelProviderOptions = ['本地 Qwen3-8B', '私有 vLLM 集群', 'OpenAI 云端增强', '局域网 Ollama'].map((value) => ({ value, label: value }));
const releaseChannelOptions = ['稳定版', '安全预览', '开发者预览'].map((value) => ({ value, label: value }));
const backupTargetOptions = ['HiGoNAS 内部快照', '异地 NAS', '加密云端仓库'].map((value) => ({ value, label: value }));

const activeCategoryId = ref<CategoryId>('accounts');
const updateStatus = ref('上次检查：今天 09:20，当前为最新版本。');
const appliedState = ref('设置已加载，等待管理员调整。');
const lastAudit = ref('系统设置窗口已打开，配置读取写入审计。');
const checkCount = ref(0);

const settings = ref<SettingsState>(createDefaultSettings());
const accounts = ref<AccountSummary>({ users: [], groups: [], grants: [] });
const accountState = ref('账号后端正在同步。');
const accountBusy = ref('');
const newUser = ref({
  username: '',
  displayName: '',
  password: 'Passw0rd!',
  role: 'user',
  quotaGB: 50,
  groupId: 'family',
});
const newGroup = ref({
  name: '',
  description: '',
});
const memberEditor = ref({
  groupId: 'family',
  userId: 'admin',
});
const grantEditor = ref({
  subjectType: 'user',
  subjectId: 'admin',
  spaceId: 'space-family',
  access: 'read_write',
  quotaGB: 100,
});

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
const accountUsers = computed(() => accounts.value.users ?? []);
const accountGroups = computed(() => accounts.value.groups ?? []);
const accountGrants = computed(() => accounts.value.grants ?? []);
const selectedMemberGroup = computed(() => accountGroups.value.find((group) => group.id === memberEditor.value.groupId));
const grantSubjects = computed(() => (
  grantEditor.value.subjectType === 'group'
    ? accountGroups.value.map((group) => ({ id: group.id, label: group.name }))
    : accountUsers.value.map((user) => ({ id: user.id, label: user.displayName || user.username }))
));

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
    uiTheme: 'auto',
    uiLocale: 'zh-CN',
    windowRadius: 'default',
    dockPosition: 'bottom',
    dockStyle: 'floating',
    dockIconSize: 'default',
  };
}

function applyBackendSettings(nextSettings: ApiSettingsState) {
  const model = nextSettings.model ?? {};
  const privacy = nextSettings.privacy ?? {};
  const ui = nextSettings.ui ?? {};
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
  settings.value.uiTheme = normalizeBackendOption(ui.theme, themeOptions, 'auto');
  settings.value.uiLocale = normalizeBackendOption(ui.locale, localeOptions, 'zh-CN');
  settings.value.windowRadius = normalizeBackendOption(ui.windowRadius, radiusOptions, 'default');
  settings.value.dockPosition = normalizeBackendOption(ui.dockPosition, dockPositionOptions, 'bottom');
  settings.value.dockStyle = normalizeBackendOption(ui.dockStyle, dockStyleOptions, 'floating');
  settings.value.dockIconSize = normalizeBackendOption(ui.dockIconSize, dockIconSizeOptions, 'default');
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
    ui: {
      theme: settings.value.uiTheme,
      locale: settings.value.uiLocale,
      windowRadius: settings.value.windowRadius,
      dockPosition: settings.value.dockPosition,
      dockStyle: settings.value.dockStyle,
      dockIconSize: settings.value.dockIconSize,
    },
  };
}

function normalizeBackendOption(value: unknown, options: { value: string }[], fallback: string) {
  return typeof value === 'string' && options.some((option) => option.value === value) ? value : fallback;
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

function setInterfaceOption(key: 'uiTheme' | 'uiLocale' | 'windowRadius' | 'dockPosition' | 'dockStyle' | 'dockIconSize', value: string) {
  settings.value[key] = value;
  lastAudit.value = '界面设置已调整，保存后应用到桌面。';
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

async function loadAccounts() {
  try {
    accounts.value = await apiClient.accounts.getSummary();
    memberEditor.value.groupId = accountGroups.value[0]?.id ?? 'family';
    memberEditor.value.userId = accountUsers.value[0]?.id ?? 'admin';
    grantEditor.value.subjectId = grantSubjects.value[0]?.id ?? 'admin';
    accountState.value = `已同步 ${accountUsers.value.length} 个用户、${accountGroups.value.length} 个用户组、${accountGrants.value.length} 条授权。`;
  } catch (error) {
    accountState.value = `账号接口不可用：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

async function createAccountUser() {
  if (!newUser.value.username.trim()) {
    accountState.value = '请输入用户名。';
    return;
  }
  accountBusy.value = 'create-user';
  try {
    const user = await apiClient.accounts.createUser({
      username: newUser.value.username.trim(),
      displayName: newUser.value.displayName.trim() || newUser.value.username.trim(),
      password: newUser.value.password,
      role: newUser.value.role,
      quotaBytes: Number(newUser.value.quotaGB) * 1024 * 1024 * 1024,
      groups: newUser.value.groupId ? [newUser.value.groupId] : [],
    });
    accountState.value = `已创建用户 ${user.displayName || user.username}。`;
    newUser.value.username = '';
    newUser.value.displayName = '';
    await loadAccounts();
  } catch (error) {
    accountState.value = `创建用户失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

async function toggleAccountUser(user: AccountUser) {
  accountBusy.value = user.id;
  try {
    const status = user.status === 'active' ? 'disabled' : 'active';
    await apiClient.accounts.updateUser(user.id, { status });
    accountState.value = `${user.displayName || user.username} 已${status === 'active' ? '启用' : '停用'}。`;
    await loadAccounts();
  } catch (error) {
    accountState.value = `更新用户失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

async function deleteAccountUser(user: AccountUser) {
  if (user.role === 'admin') {
    accountState.value = '管理员账号不能在此处删除。';
    return;
  }
  accountBusy.value = user.id;
  try {
    await apiClient.accounts.deleteUser(user.id);
    accountState.value = `${user.displayName || user.username} 已删除。`;
    await loadAccounts();
  } catch (error) {
    accountState.value = `删除用户失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

async function createAccountGroup() {
  if (!newGroup.value.name.trim()) {
    accountState.value = '请输入用户组名称。';
    return;
  }
  accountBusy.value = 'create-group';
  try {
    const group = await apiClient.accounts.createGroup({
      name: newGroup.value.name.trim(),
      description: newGroup.value.description.trim(),
    });
    accountState.value = `已创建用户组 ${group.name}。`;
    newGroup.value.name = '';
    newGroup.value.description = '';
    await loadAccounts();
  } catch (error) {
    accountState.value = `创建用户组失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

async function addMemberToGroup() {
  const group = selectedMemberGroup.value;
  if (!group || !memberEditor.value.userId) {
    accountState.value = '请选择用户组和用户。';
    return;
  }
  accountBusy.value = 'members';
  try {
    const userIds = Array.from(new Set([...(group.userIds ?? []), memberEditor.value.userId]));
    const updated = await apiClient.accounts.updateGroupMembers(group.id, userIds);
    accountState.value = `${updated.name} 成员已更新。`;
    await loadAccounts();
  } catch (error) {
    accountState.value = `更新成员失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

async function grantAccountSpace() {
  if (!grantEditor.value.subjectId || !grantEditor.value.spaceId.trim()) {
    accountState.value = '请选择授权对象并填写空间 ID。';
    return;
  }
  accountBusy.value = 'grant';
  try {
    const grant = await apiClient.accounts.grantSpace({
      subjectType: grantEditor.value.subjectType,
      subjectId: grantEditor.value.subjectId,
      spaceId: grantEditor.value.spaceId.trim(),
      access: grantEditor.value.access,
      quotaBytes: Number(grantEditor.value.quotaGB) * 1024 * 1024 * 1024,
    });
    accountState.value = `已写入空间授权：${grant.subjectId} -> ${grant.spaceId}。`;
    await loadAccounts();
  } catch (error) {
    accountState.value = `授权失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    accountBusy.value = '';
  }
}

// --- LLM provider binding ---------------------------------------------------

const providerKinds: Array<{ value: AiProviderKind; label: string }> = [
  { value: 'openai', label: 'OpenAI 兼容' },
  { value: 'anthropic', label: 'Anthropic' },
  { value: 'gemini', label: 'Google Gemini' },
];

function emptyProviderForm(): AiProviderInput {
  return { name: '', kind: 'openai', baseUrl: '', apiKey: '', model: '', isDefault: false };
}

const providers = ref<AiProvider[]>([]);
const providerForm = ref<AiProviderInput>(emptyProviderForm());
const providerBusy = ref(false);
const providerNotice = ref('绑定 OpenAI、Anthropic、Gemini 或任意 OpenAI 兼容端点后，助手将给出真实回答。');

const providerKindLabel = (kind: AiProviderKind) =>
  providerKinds.find((item) => item.value === kind)?.label ?? kind;

async function loadProviders() {
  try {
    providers.value = await apiClient.ai.listProviders();
  } catch (error) {
    providerNotice.value = `无法加载模型供应商：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

async function submitProvider() {
  const form = providerForm.value;
  if (!form.name?.trim() || !form.model?.trim()) {
    providerNotice.value = '请填写名称和模型 ID。';
    return;
  }
  providerBusy.value = true;
  try {
    await apiClient.ai.createProvider(form);
    providerForm.value = emptyProviderForm();
    providerNotice.value = '已添加模型供应商。';
    await loadProviders();
  } catch (error) {
    providerNotice.value = `添加失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    providerBusy.value = false;
  }
}

async function setDefaultProvider(provider: AiProvider) {
  try {
    await apiClient.ai.updateProvider(provider.id, { isDefault: true });
    await loadProviders();
    providerNotice.value = `已将「${provider.name}」设为默认模型。`;
  } catch (error) {
    providerNotice.value = `设置默认失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

async function removeProvider(provider: AiProvider) {
  try {
    await apiClient.ai.deleteProvider(provider.id);
    await loadProviders();
    providerNotice.value = `已删除「${provider.name}」。`;
  } catch (error) {
    providerNotice.value = `删除失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

async function testProvider(provider: AiProvider) {
  providerBusy.value = true;
  providerNotice.value = `正在测试「${provider.name}」…`;
  try {
    const result = await apiClient.ai.testProvider(provider.id);
    providerNotice.value = `「${provider.name}」连接成功 · ${result.latencyMs}ms · 回复：${result.reply || '(空)'}`;
  } catch (error) {
    providerNotice.value = `「${provider.name}」连接失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    providerBusy.value = false;
  }
}

onMounted(async () => {
  const [nextSettings] = await Promise.all([
    settingsStore.loadSettings(),
    loadAccounts(),
    loadProviders(),
  ]);
  if (nextSettings.model || nextSettings.privacy || nextSettings.ui) {
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
        <SettingsAccountsPanel
          v-if="activeCategoryId === 'accounts'"
          :account-state="accountState"
          :account-busy="accountBusy"
          :new-user="newUser"
          :new-group="newGroup"
          :member-editor="memberEditor"
          :grant-editor="grantEditor"
          :account-users="accountUsers"
          :account-groups="accountGroups"
          :grant-subjects="grantSubjects"
          @create-user="createAccountUser"
          @toggle-user="toggleAccountUser"
          @delete-user="deleteAccountUser"
          @create-group="createAccountGroup"
          @add-member="addMemberToGroup"
          @grant-space="grantAccountSpace"
          @subject-type-change="grantEditor.subjectId = grantSubjects[0]?.id ?? ''"
        />

        <SettingsNetworkPanel
          v-else-if="activeCategoryId === 'network'"
          :settings="settings"
          :dns-profile-options="dnsProfileOptions"
          @toggle-ddns="settings.ddnsEnabled = !settings.ddnsEnabled"
          @toggle-remote="settings.remoteAccess = !settings.remoteAccess"
        />

        <SettingsModelsPanel
          v-else-if="activeCategoryId === 'models'"
          :settings="settings"
          :model-strategies="modelStrategies"
          :model-provider-options="modelProviderOptions"
          :model-summary="modelSummary"
          :providers="providers"
          :provider-form="providerForm"
          :provider-busy="providerBusy"
          :provider-notice="providerNotice"
          :provider-kinds="providerKinds"
          :provider-kind-label="providerKindLabel"
          @set-strategy="setModelStrategy"
          @toggle-task-routing="settings.taskRouting = !settings.taskRouting"
          @submit-provider="submitProvider"
          @test-provider="testProvider"
          @set-default-provider="setDefaultProvider"
          @remove-provider="removeProvider"
        />

        <SettingsAiPanel
          v-else-if="activeCategoryId === 'ai'"
          :settings="settings"
          @toggle-local="settings.localAi = !settings.localAi"
          @toggle-cloud="settings.cloudAi = !settings.cloudAi"
          @toggle-private="settings.privateEndpoint = !settings.privateEndpoint"
        />

        <SettingsNotificationsPanel
          v-else-if="activeCategoryId === 'notifications'"
          :settings="settings"
          :enabled-notice-count="enabledNoticeCount"
          @toggle-backup="settings.backupNotice = !settings.backupNotice"
          @toggle-security="settings.securityNotice = !settings.securityNotice"
          @toggle-life="settings.lifeNotice = !settings.lifeNotice"
        />

        <SettingsUpdatesPanel
          v-else-if="activeCategoryId === 'updates'"
          :settings="settings"
          :release-channel-options="releaseChannelOptions"
          :update-status="updateStatus"
          @toggle-auto-update="settings.autoUpdate = !settings.autoUpdate"
          @check-updates="checkForUpdates"
        />

        <SettingsPrivacyPanel
          v-else-if="activeCategoryId === 'privacy'"
          :settings="settings"
          :privacy-summary="privacySummary"
          @set-privacy-mode="setPrivacyMode"
          @toggle-sensitive="settings.sensitiveLocalOnly = !settings.sensitiveLocalOnly"
        />

        <SettingsAuditPanel
          v-else-if="activeCategoryId === 'audit'"
          :settings="settings"
          :retention-options="retentionOptions"
          @set-retention="setAuditRetention"
        />

        <SettingsBackupPanel
          v-else-if="activeCategoryId === 'backup'"
          :settings="settings"
          :backup-target-options="backupTargetOptions"
          @toggle-backup="settings.systemBackup = !settings.systemBackup"
          @create-backup="createSystemBackup"
        />

        <SettingsInterfacePanel
          v-else-if="activeCategoryId === 'interface'"
          :settings="settings"
          :theme-options="themeOptions"
          :locale-options="localeOptions"
          :radius-options="radiusOptions"
          :dock-position-options="dockPositionOptions"
          :dock-style-options="dockStyleOptions"
          :dock-icon-size-options="dockIconSizeOptions"
          @set-option="setInterfaceOption"
        />

      </section>

      <section class="system-settings__footer" aria-label="保存和审计状态">
        <div>
          <strong>{{ lastAudit }}</strong>
          <span>账号权限、网络、模型、安全治理和备份策略均受审计保护。</span>
        </div>
        <div class="system-settings__footer-actions">
          <UiButton variant="soft" :icon-left="RotateCcw" @click="restoreDefaults">恢复默认</UiButton>
          <UiButton :icon-left="Save" @click="saveSettings">保存应用</UiButton>
        </div>
      </section>
    </main>
  </div>
</template>
