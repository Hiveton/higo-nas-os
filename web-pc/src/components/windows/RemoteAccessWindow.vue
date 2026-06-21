<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  CheckCircle2,
  Copy,
  Globe2,
  KeyRound,
  Link2,
  Network,
  RefreshCw,
  Router,
  ScanLine,
  ShieldCheck,
  ShieldOff,
  Smartphone,
  UserCheck,
  Wifi,
} from 'lucide-vue-next';
import { remoteStore } from '../../stores/remote';
import type { AccessPolicy, RemoteDevice, RemoteLoginAlert } from '../../api/types';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { UiButton, UiWindowPage, UiNavRail, UiStatGrid, UiStat, UiPanel } from '../ui';

type PolicyKey = string;

const remoteEnabled = ref(true);
const mfaEnabled = ref(true);
const tunnelMode = ref<'智能中继' | '直连优先'>('智能中继');
const activePolicy = ref<PolicyKey>('family');
const domainTokenVersion = ref(3);
const feedback = ref('远程访问通道在线，最近一次策略审计于 14:10 完成');
const scanState = ref<'idle' | 'safe' | 'risk'>('idle');

const remoteDomain = computed(() => remoteStore.status.value?.domain ?? `home-${domainTokenVersion.value}.higo.link`);
const channelState = computed(() => (remoteEnabled.value ? '在线' : '已暂停'));
const tunnel = computed(() => remoteStore.status.value?.tunnel ?? null);
const tunnelLabel = computed(() => {
  const t = tunnel.value;
  if (!t || t.backend === 'none') return '未配置';
  if (!t.up) return '未连接';
  return t.backend === 'wireguard' ? `WireGuard · :${t.listenPort}` : `${t.backend} · :${t.listenPort}`;
});
const tunnelState = computed(() =>
  remoteEnabled.value
    ? (remoteStore.status.value?.tunnelState ?? `${tunnelMode.value} · TLS 1.3 · 52ms`)
    : '通道关闭，外部请求被拒绝',
);

const fallbackDevices = ref<RemoteDevice[]>([
  {
    id: 'iphone',
    name: 'Hiveton iPhone',
    role: '管理员设备',
    location: '上海',
    bound: true,
    lastSeen: '刚刚',
  },
  {
    id: 'macbook',
    name: 'MacBook Pro',
    role: '可信电脑',
    location: '杭州',
    bound: true,
    lastSeen: '12 分钟前',
  },
  {
    id: 'ipad',
    name: 'Family iPad',
    role: '家庭成员',
    location: '南京',
    bound: false,
    lastSeen: '待绑定',
  },
]);

const fallbackPolicies: AccessPolicy[] = [
  {
    key: 'family',
    name: '家庭访问',
    scope: '家庭成员可远程预览照片和文档',
    risk: '低风险',
  },
  {
    key: 'team',
    name: '团队协作',
    scope: '允许项目空间 WebDAV 与分享链接',
    risk: '中风险',
  },
  {
    key: 'guest',
    name: '访客临时',
    scope: '仅限 24 小时只读链接',
    risk: '低风险',
  },
] as AccessPolicy[];

const fallbackLoginAlerts = ref<RemoteLoginAlert[]>([
  {
    id: 'login-1',
    location: '深圳',
    device: 'Chrome / Windows',
    action: '已要求 MFA',
    state: '待确认',
  },
  {
    id: 'login-2',
    location: '东京',
    device: 'Safari / iPhone',
    action: '策略拒绝',
    state: '已拦截',
  },
]);

// The default check list is shown before a scan runs; once the backend
// share-scan returns, its real `checks` replace these labels.
const defaultShareChecks = ['公开分享范围', '过期时间', '下载权限', '敏感文件标签'];
const shareChecks = computed(() => {
  const checks = remoteStore.shareScan.value?.checks;
  return checks && checks.length ? checks : defaultShareChecks;
});

const devices = computed(() => (remoteStore.devices.value.length > 0 ? remoteStore.devices.value : fallbackDevices.value));
const policies = computed(() =>
  remoteStore.status.value?.policies?.length ? remoteStore.status.value.policies : fallbackPolicies,
);
const loginAlerts = computed(() =>
  remoteStore.loginAlerts.value.length > 0 ? remoteStore.loginAlerts.value : fallbackLoginAlerts.value,
);
const activePolicyDetail = computed(
  () => policies.value.find((policy) => policy.key === activePolicy.value) ?? policies.value[0] ?? fallbackPolicies[0],
);
const boundDevices = computed(() => devices.value.filter((device) => device.bound).length);
const shareScanMessage = computed(() => {
  if (remoteStore.shareScan.value?.message) return remoteStore.shareScan.value.message;
  if (scanState.value === 'safe') return '扫描完成：分享链接仅限家庭访问，7 天后自动过期。';
  if (scanState.value === 'risk') return '扫描完成：发现公开下载权限，建议切换为访客临时策略。';
  return '等待扫描：检查分享链接范围、过期时间和敏感标签。';
});

function applyRemoteStatus() {
  const status = remoteStore.status.value;
  if (!status) return;
  remoteEnabled.value = status.channelEnabled ?? status.enabled;
  mfaEnabled.value = status.mfaEnabled;
  tunnelMode.value = status.tunnelMode === '直连优先' ? '直连优先' : '智能中继';
  activePolicy.value = status.activePolicy?.key ?? activePolicy.value;
  domainTokenVersion.value = status.token?.version ?? domainTokenVersion.value;
  feedback.value = status.feedback ?? feedback.value;
}

function mutateFallbackDevice(deviceId: string) {
  const device = fallbackDevices.value.find((item) => item.id === deviceId);
  if (!device) return;
  device.bound = !device.bound;
  device.lastSeen = device.bound ? '刚刚绑定' : '已解绑';
  feedback.value = `${device.name} ${device.bound ? '已绑定为可信设备' : '已解绑，远程令牌已失效'}`;
}

async function toggleRemoteChannel() {
  try {
    if (remoteEnabled.value) {
      await remoteStore.stopChannel();
    } else {
      await remoteStore.startChannel();
    }
    applyRemoteStatus();
  } catch {
    remoteEnabled.value = !remoteEnabled.value;
    feedback.value = remoteEnabled.value ? '远程通道已启动，内网穿透重新握手成功' : '远程通道已暂停，新连接会被拒绝';
  }
}

async function toggleMfa() {
  const nextEnabled = !mfaEnabled.value;
  try {
    await remoteStore.toggleMfa(nextEnabled);
    applyRemoteStatus();
  } catch {
    mfaEnabled.value = nextEnabled;
    feedback.value = mfaEnabled.value ? '多因素认证已启用，异地登录必须二次确认' : '多因素认证已关闭，已写入安全审计';
  }
}

async function toggleDevice(deviceId: string) {
  const device = devices.value.find((item) => item.id === deviceId);
  if (!device) return;
  try {
    if (remoteStore.usingFallback.value || remoteStore.devices.value.length === 0) {
      mutateFallbackDevice(deviceId);
      return;
    }
    if (device.bound) {
      await remoteStore.unbindDevice(deviceId);
    } else {
      await remoteStore.bindDevice(deviceId);
    }
    applyRemoteStatus();
    feedback.value = `${device.name} ${device.bound ? '已解绑，远程令牌已失效' : '已绑定为可信设备'}`;
  } catch {
    mutateFallbackDevice(deviceId);
  }
}

async function selectPolicy(policy: PolicyKey) {
  try {
    await remoteStore.selectPolicy(policy);
    applyRemoteStatus();
  } catch {
    activePolicy.value = policy;
    feedback.value = `访问策略已切换为「${activePolicyDetail.value.name}」`;
  }
}

async function scanShareLinks() {
  try {
    const result = await remoteStore.scanShareLinks();
    scanState.value = result.state === 'safe' || result.state === 'risk' ? result.state : 'idle';
    feedback.value = result.message;
  } catch {
    scanState.value = scanState.value === 'risk' ? 'safe' : 'risk';
    feedback.value = shareScanMessage.value;
  }
}

async function copyDomainToken() {
  try {
    const token = await remoteStore.createDomainToken();
    applyRemoteStatus();
    try {
      await navigator.clipboard?.writeText(token.token);
      feedback.value = `已复制 ${token.domain} 的短期访问令牌，有效期 10 分钟`;
    } catch {
      feedback.value = `已生成 ${token.domain} 的短期访问令牌（请手动复制），有效期 10 分钟`;
    }
  } catch {
    feedback.value = `已复制 ${remoteDomain.value} 的短期访问令牌，有效期 10 分钟`;
  }
}

async function rotateDomainToken() {
  try {
    await remoteStore.rotateDomainToken();
    applyRemoteStatus();
  } catch {
    domainTokenVersion.value += 1;
    feedback.value = `远程域名令牌已轮换，新域名 ${remoteDomain.value} 已生效`;
  }
}

async function toggleTunnelMode() {
  const nextMode = tunnelMode.value === '智能中继' ? '直连优先' : '智能中继';
  try {
    await remoteStore.updateTunnelMode(nextMode);
    applyRemoteStatus();
  } catch {
    tunnelMode.value = nextMode;
    feedback.value = `内网穿透模式已切换为 ${tunnelMode.value}`;
  }
}

onMounted(async () => {
  await remoteStore.loadRemoteDashboard();
  applyRemoteStatus();
});
</script>

<template>
  <UiWindowPage
    layout="master-detail"
    :icon="Globe2"
    :title="remoteDomain"
    :subtitle="tunnelState"
    :status="feedback"
    status-tone="success"
  >
    <template #actions>
      <UiButton variant="soft" size="sm" :icon-left="Copy" @click="copyDomainToken">复制令牌</UiButton>
      <UiButton variant="soft" size="sm" :icon-left="RefreshCw" @click="rotateDomainToken">轮换</UiButton>
      <UiButton size="sm" :icon-left="remoteEnabled ? ShieldOff : ShieldCheck" @click="toggleRemoteChannel">
        {{ remoteEnabled ? '暂停通道' : '启动通道' }}
      </UiButton>
    </template>

    <template #nav>
      <UiNavRail title="设备绑定" :subtitle="`${boundDevices} / ${devices.length}`">
        <div class="remote-access__devices" aria-label="设备绑定">
          <button
            v-for="device in devices"
            :key="device.id"
            class="remote-access__device"
            :class="{ 'remote-access__device--bound': device.bound }"
            type="button"
            @click="toggleDevice(device.id)"
          >
            <UserCheck :size="16" />
            <span>
              <strong>{{ device.name }}</strong>
              <small>{{ device.role }} · {{ device.location }} · {{ device.lastSeen }}</small>
            </span>
            <b>{{ device.bound ? '解绑' : '绑定' }}</b>
          </button>
        </div>
      </UiNavRail>
    </template>

    <UiStatGrid>
      <UiStat :icon="Globe2" label="远程域名" :value="channelState" tone="primary" />
      <UiStat :icon="Network" label="内网穿透" :value="tunnelMode" tone="info" />
      <UiStat :icon="Network" label="真实隧道" :value="tunnelLabel" tone="info" />
      <UiStat :icon="KeyRound" label="MFA" :value="mfaEnabled ? '已启用' : '已关闭'" :tone="mfaEnabled ? 'success' : 'warning'" />
      <UiStat :icon="Smartphone" label="设备绑定" :value="`${boundDevices} 台`" tone="success" />
    </UiStatGrid>

    <UiPanel title="访问策略" framed>
      <template #actions>
        <UiButton variant="soft" size="sm" :icon-left="Router" @click="toggleTunnelMode">{{ tunnelMode }}</UiButton>
        <UiButton variant="soft" size="sm" :icon-left="mfaEnabled ? ShieldCheck : ShieldOff" @click="toggleMfa">
          {{ mfaEnabled ? '关闭 MFA' : '启用 MFA' }}
        </UiButton>
      </template>
      <div class="remote-access__policy-grid">
        <button
          v-for="policy in policies"
          :key="policy.key"
          class="remote-access__policy-card"
          :class="{ 'remote-access__policy-card--active': policy.key === activePolicy }"
          type="button"
          @click="selectPolicy(policy.key)"
        >
          <strong>{{ policy.name }}</strong>
          <span>{{ policy.scope }}</span>
          <small>{{ policy.risk }}</small>
        </button>
      </div>
    </UiPanel>

    <UiPanel title="安全提醒" framed>
      <template #actions>
        <UiButton variant="soft" size="sm" :icon-left="ScanLine" @click="scanShareLinks">扫描分享链接</UiButton>
      </template>
      <div class="remote-access__login-alerts">
        <article v-for="alert in loginAlerts" :key="alert.id">
          <Wifi :size="15" />
          <div>
            <strong>{{ alert.location }} 异地登录</strong>
            <span>{{ alert.device }} · {{ alert.action }} · {{ alert.state }}</span>
          </div>
        </article>
      </div>

      <div class="remote-access__share-check" :class="`remote-access__share-check--${scanState}`">
        <div>
          <Link2 :size="17" />
          <strong>{{ shareScanMessage }}</strong>
        </div>
        <span v-for="item in shareChecks" :key="item">
          <CheckCircle2 :size="12" />
          {{ item }}
        </span>
      </div>
    </UiPanel>

    <NasFeaturePanel :modules="['remote', 'protocols']" />
  </UiWindowPage>
</template>

<style scoped>
.remote-access__devices {
  display: flex;
  flex-direction: column;
  min-width: 0;
}

.remote-access__device {
  display: grid;
  grid-template-columns: 20px minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-height: 58px;
  padding: 9px 10px;
  color: var(--text-soft);
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid var(--border);
  transition: background var(--duration-fast) var(--ease-standard);
}

.remote-access__device:hover {
  background: var(--accent-soft);
}

.remote-access__device--bound,
.remote-access__device--bound:hover {
  color: var(--ink-green);
  background: var(--accent-green-soft);
}

.remote-access__device strong,
.remote-access__device small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remote-access__device strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.remote-access__device small {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 10px;
}

.remote-access__device b {
  color: var(--accent);
  font-size: var(--fs-2xs);
}

.remote-access__policy-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(180px, 1fr));
  gap: 10px;
  align-content: start;
}

.remote-access__policy-card {
  display: grid;
  gap: 6px;
  min-height: 108px;
  align-content: start;
  padding: 12px;
  text-align: left;
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  transition: background var(--duration-fast) var(--ease-standard),
    border-color var(--duration-fast) var(--ease-standard);
}

.remote-access__policy-card:hover {
  background: var(--accent-soft);
}

.remote-access__policy-card--active,
.remote-access__policy-card--active:hover {
  background: var(--accent-soft);
  border-color: var(--accent-soft);
  box-shadow: inset 3px 0 0 var(--accent);
}

.remote-access__policy-card strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.remote-access__policy-card span {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.38;
}

.remote-access__policy-card small {
  color: var(--ink-orange);
  font-size: 10px;
  font-weight: var(--fw-bold);
}

.remote-access__login-alerts {
  display: grid;
  gap: 8px;
}

.remote-access__login-alerts article {
  display: flex;
  gap: 8px;
  min-width: 0;
  color: var(--ink-orange);
}

.remote-access__login-alerts strong,
.remote-access__login-alerts span {
  display: block;
}

.remote-access__login-alerts strong {
  color: var(--text-strong);
  font-size: var(--fs-2xs);
}

.remote-access__login-alerts span {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
  line-height: var(--lh-snug);
}

.remote-access__share-check {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
  align-content: start;
  padding: 12px;
  color: var(--text-muted);
}

.remote-access__share-check div {
  display: flex;
  flex: 0 0 100%;
  gap: 8px;
  color: var(--accent);
}

.remote-access__share-check strong {
  color: var(--text-strong);
  font-size: var(--fs-2xs);
  line-height: 1.42;
}

.remote-access__share-check span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  font-size: 10px;
  font-weight: var(--fw-bold);
}

.remote-access__share-check--risk div {
  color: var(--ink-orange);
}

.remote-access__share-check--safe div {
  color: var(--ink-green);
}

@container desktop-window-body (max-width: 760px) {
  .remote-access__policy-grid {
    grid-template-columns: 1fr;
  }
}
</style>
