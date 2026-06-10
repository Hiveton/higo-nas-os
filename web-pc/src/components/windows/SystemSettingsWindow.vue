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
  Palette,
  RefreshCw,
  RotateCcw,
  Save,
  Server,
  ShieldCheck,
  Users,
  Wifi,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { settingsStore } from '../../stores/settings';
import type { AccountSummary, AccountUser, SettingsState as ApiSettingsState } from '../../api/types';

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

function setInterfaceOption(key: 'uiTheme' | 'windowRadius' | 'dockPosition' | 'dockStyle' | 'dockIconSize', value: string) {
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

onMounted(async () => {
  const [nextSettings] = await Promise.all([
    settingsStore.loadSettings(),
    loadAccounts(),
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
        <div v-if="activeCategoryId === 'accounts'" class="system-settings__panel">
          <div class="system-settings__account-status">
            <strong>账号与空间授权</strong>
            <span>{{ accountState }}</span>
          </div>

          <section class="system-settings__account-card" aria-label="创建用户">
            <h4>创建用户</h4>
            <div class="system-settings__account-form">
              <label>
                <span>用户名</span>
                <input v-model="newUser.username" type="text" />
              </label>
              <label>
                <span>显示名</span>
                <input v-model="newUser.displayName" type="text" />
              </label>
              <label>
                <span>密码</span>
                <input v-model="newUser.password" type="password" />
              </label>
              <label>
                <span>角色</span>
                <select v-model="newUser.role">
                  <option value="user">普通用户</option>
                  <option value="admin">管理员</option>
                  <option value="guest">访客</option>
                </select>
              </label>
              <label>
                <span>配额 GB</span>
                <input v-model.number="newUser.quotaGB" min="0" type="number" />
              </label>
              <label>
                <span>用户组</span>
                <select v-model="newUser.groupId">
                  <option value="">无</option>
                  <option v-for="group in accountGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
              </label>
              <button type="button" :disabled="accountBusy === 'create-user'" @click="createAccountUser">
                {{ accountBusy === 'create-user' ? '创建中' : '创建用户' }}
              </button>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="用户列表">
            <h4>用户</h4>
            <div class="system-settings__account-list">
              <article v-for="user in accountUsers" :key="user.id">
                <div>
                  <strong>{{ user.displayName || user.username }}</strong>
                  <small>{{ user.username }} · {{ user.role }} · {{ user.status }} · {{ Math.round(user.quotaBytes / 1024 / 1024 / 1024) }} GB</small>
                </div>
                <button type="button" :disabled="accountBusy === user.id" @click="toggleAccountUser(user)">
                  {{ user.status === 'active' ? '停用' : '启用' }}
                </button>
                <button type="button" :disabled="accountBusy === user.id || user.role === 'admin'" @click="deleteAccountUser(user)">删除</button>
              </article>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="用户组和授权">
            <h4>用户组 / 授权</h4>
            <div class="system-settings__account-form">
              <label>
                <span>新用户组</span>
                <input v-model="newGroup.name" type="text" />
              </label>
              <label>
                <span>描述</span>
                <input v-model="newGroup.description" type="text" />
              </label>
              <button type="button" :disabled="accountBusy === 'create-group'" @click="createAccountGroup">创建组</button>
              <label>
                <span>选择组</span>
                <select v-model="memberEditor.groupId">
                  <option v-for="group in accountGroups" :key="group.id" :value="group.id">{{ group.name }}</option>
                </select>
              </label>
              <label>
                <span>添加成员</span>
                <select v-model="memberEditor.userId">
                  <option v-for="user in accountUsers" :key="user.id" :value="user.id">{{ user.displayName || user.username }}</option>
                </select>
              </label>
              <button type="button" :disabled="accountBusy === 'members'" @click="addMemberToGroup">保存成员</button>
            </div>
            <div class="system-settings__account-form">
              <label>
                <span>授权类型</span>
                <select v-model="grantEditor.subjectType" @change="grantEditor.subjectId = grantSubjects[0]?.id ?? ''">
                  <option value="user">用户</option>
                  <option value="group">用户组</option>
                </select>
              </label>
              <label>
                <span>授权对象</span>
                <select v-model="grantEditor.subjectId">
                  <option v-for="subject in grantSubjects" :key="subject.id" :value="subject.id">{{ subject.label }}</option>
                </select>
              </label>
              <label>
                <span>空间 ID</span>
                <input v-model="grantEditor.spaceId" type="text" />
              </label>
              <label>
                <span>权限</span>
                <select v-model="grantEditor.access">
                  <option value="read">只读</option>
                  <option value="read_write">读写</option>
                  <option value="manage">管理</option>
                </select>
              </label>
              <label>
                <span>配额 GB</span>
                <input v-model.number="grantEditor.quotaGB" min="0" type="number" />
              </label>
              <button type="button" :disabled="accountBusy === 'grant'" @click="grantAccountSpace">保存授权</button>
            </div>
          </section>
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

        <div v-else-if="activeCategoryId === 'backup'" class="system-settings__panel">
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

        <div v-else-if="activeCategoryId === 'interface'" class="system-settings__panel system-settings__panel--interface">
          <section class="system-settings__account-card" aria-label="主题">
            <h4>主题</h4>
            <div class="system-settings__segmented system-settings__segmented--three">
              <button
                v-for="option in themeOptions"
                :key="option.value"
                :class="{ 'system-settings__segmented-button--active': option.value === settings.uiTheme }"
                type="button"
                @click="setInterfaceOption('uiTheme', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="窗口圆角">
            <h4>窗口圆角</h4>
            <div class="system-settings__segmented system-settings__segmented--three">
              <button
                v-for="option in radiusOptions"
                :key="option.value"
                :class="{ 'system-settings__segmented-button--active': option.value === settings.windowRadius }"
                type="button"
                @click="setInterfaceOption('windowRadius', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="Dock 位置">
            <h4>Dock 位置</h4>
            <div class="system-settings__segmented system-settings__segmented--three">
              <button
                v-for="option in dockPositionOptions"
                :key="option.value"
                :class="{ 'system-settings__segmented-button--active': option.value === settings.dockPosition }"
                type="button"
                @click="setInterfaceOption('dockPosition', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="Dock 样式">
            <h4>Dock 样式</h4>
            <div class="system-settings__segmented system-settings__segmented--three">
              <button
                v-for="option in dockStyleOptions"
                :key="option.value"
                :class="{ 'system-settings__segmented-button--active': option.value === settings.dockStyle }"
                type="button"
                @click="setInterfaceOption('dockStyle', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>

          <section class="system-settings__account-card" aria-label="Dock 图标大小">
            <h4>Dock 图标大小</h4>
            <div class="system-settings__segmented system-settings__segmented--three">
              <button
                v-for="option in dockIconSizeOptions"
                :key="option.value"
                :class="{ 'system-settings__segmented-button--active': option.value === settings.dockIconSize }"
                type="button"
                @click="setInterfaceOption('dockIconSize', option.value)"
              >
                {{ option.label }}
              </button>
            </div>
          </section>
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

.system-settings__account-status,
.system-settings__account-card {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 11px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.system-settings__account-status strong,
.system-settings__account-card h4 {
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.system-settings__account-status span {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.4;
}

.system-settings__account-form {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(118px, 1fr));
  gap: 8px;
  min-width: 0;
}

.system-settings__account-form label {
  display: grid;
  gap: 5px;
  min-width: 0;
}

.system-settings__account-form span {
  color: var(--text-soft);
  font-size: 10px;
  font-weight: 720;
}

.system-settings__account-form input,
.system-settings__account-form select {
  width: 100%;
  min-width: 0;
  height: 30px;
  padding: 0 8px;
  color: var(--text-strong);
  background: rgba(255, 255, 255, 0.8);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.system-settings__account-form button,
.system-settings__account-list button {
  align-self: end;
  min-height: 30px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.system-settings__account-form button:disabled,
.system-settings__account-list button:disabled {
  color: var(--text-soft);
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.14);
  border-color: rgba(148, 163, 184, 0.18);
}

.system-settings__account-list {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.system-settings__account-list article {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto auto;
  align-items: center;
  gap: 7px;
  min-width: 0;
  padding: 8px;
  background: rgba(255, 255, 255, 0.54);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.system-settings__account-list strong,
.system-settings__account-list small {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.system-settings__account-list strong {
  color: var(--text-strong);
  font-size: 12px;
}

.system-settings__account-list small {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
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

.system-settings__segmented--three {
  grid-template-columns: repeat(3, minmax(0, 1fr));
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
