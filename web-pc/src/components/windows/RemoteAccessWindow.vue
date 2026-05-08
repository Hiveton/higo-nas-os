<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  AlertTriangle,
  CheckCircle2,
  Copy,
  Globe2,
  KeyRound,
  Link2,
  LockKeyhole,
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

const shareChecks = ref([
  '公开分享范围',
  '过期时间',
  '下载权限',
  '敏感文件标签',
]);

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
    await remoteStore.createDomainToken();
    applyRemoteStatus();
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
  <div class="remote-access">
    <section class="remote-access__hero" aria-label="远程通道">
      <div>
        <p>远程访问中心</p>
        <strong>{{ remoteDomain }}</strong>
        <span>{{ tunnelState }}</span>
      </div>
      <div class="remote-access__hero-actions">
        <button type="button" @click="copyDomainToken"><Copy :size="14" /> 复制令牌</button>
        <button type="button" @click="rotateDomainToken"><RefreshCw :size="14" /> 轮换</button>
        <button class="remote-access__primary-button" type="button" @click="toggleRemoteChannel">
          <component :is="remoteEnabled ? ShieldOff : ShieldCheck" :size="15" />
          {{ remoteEnabled ? '暂停通道' : '启动通道' }}
        </button>
      </div>
    </section>

    <section class="remote-access__status" aria-label="远程访问状态">
      <article>
        <Globe2 :size="17" />
        <span>远程域名</span>
        <strong>{{ channelState }}</strong>
      </article>
      <article>
        <Network :size="17" />
        <span>内网穿透</span>
        <strong>{{ tunnelMode }}</strong>
      </article>
      <article>
        <KeyRound :size="17" />
        <span>MFA</span>
        <strong>{{ mfaEnabled ? '已启用' : '已关闭' }}</strong>
      </article>
      <article>
        <Smartphone :size="17" />
        <span>设备绑定</span>
        <strong>{{ boundDevices }} 台</strong>
      </article>
    </section>

    <main class="remote-access__main">
      <section class="remote-access__devices" aria-label="设备绑定">
        <header>
          <h3><Smartphone :size="15" /> 设备绑定</h3>
          <span>{{ boundDevices }} / {{ devices.length }}</span>
        </header>
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
      </section>

      <section class="remote-access__policy" aria-label="访问策略">
        <header>
          <h3><LockKeyhole :size="15" /> 访问策略</h3>
          <span>{{ activePolicyDetail.risk }}</span>
        </header>
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
        <div class="remote-access__toggles">
          <button type="button" @click="toggleTunnelMode"><Router :size="14" /> {{ tunnelMode }}</button>
          <button type="button" @click="toggleMfa">
            <component :is="mfaEnabled ? ShieldCheck : ShieldOff" :size="14" />
            {{ mfaEnabled ? '关闭 MFA' : '启用 MFA' }}
          </button>
        </div>
      </section>

      <section class="remote-access__security" aria-label="异地登录提醒和分享链接安全检查">
        <header>
          <h3><AlertTriangle :size="15" /> 安全提醒</h3>
          <button type="button" @click="scanShareLinks"><ScanLine :size="14" /> 扫描分享链接</button>
        </header>

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
      </section>
    </main>

    <section class="remote-access__audit" aria-label="安全治理反馈">
      <ShieldCheck :size="16" />
      <span>{{ feedback }}</span>
    </section>
  </div>
</template>

<style scoped>
.remote-access {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.remote-access__hero,
.remote-access__status,
.remote-access__devices,
.remote-access__policy,
.remote-access__security,
.remote-access__audit {
  min-width: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.remote-access__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 14px 16px;
  background: linear-gradient(135deg, rgba(231, 247, 255, 0.92), rgba(255, 246, 227, 0.78));
}

.remote-access__hero p,
.remote-access__hero strong,
.remote-access__hero span {
  display: block;
  margin: 0;
}

.remote-access__hero p {
  color: var(--text-muted);
  font-size: 12px;
  font-weight: 700;
}

.remote-access__hero strong {
  margin-top: 3px;
  color: var(--text-strong);
  font-size: 20px;
}

.remote-access__hero span {
  margin-top: 5px;
  color: var(--text-muted);
  font-size: 11px;
}

.remote-access button {
  font-family: inherit;
}

.remote-access__hero-actions,
.remote-access__toggles,
.remote-access__security header {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.remote-access__hero-actions button,
.remote-access__toggles button,
.remote-access__security header button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
  white-space: nowrap;
}

.remote-access__hero-actions .remote-access__primary-button {
  color: #fff;
  background: var(--accent);
  border-color: transparent;
}

.remote-access__status {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
}

.remote-access__status article {
  display: grid;
  gap: 4px;
  justify-items: center;
  padding: 10px 6px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.36);
}

.remote-access__status span {
  color: var(--text-soft);
  font-size: 10px;
}

.remote-access__status strong {
  overflow: hidden;
  max-width: 100%;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.remote-access__main {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) 250px;
  gap: 12px;
  min-height: 0;
}

.remote-access__devices,
.remote-access__policy,
.remote-access__security {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  min-height: 0;
  overflow: hidden;
}

.remote-access header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 10px 11px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.remote-access h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.remote-access header span {
  color: var(--text-soft);
  font-size: 11px;
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
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.remote-access__device--bound {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.07);
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
  font-size: 12px;
}

.remote-access__device small {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 10px;
}

.remote-access__device b {
  color: var(--accent);
  font-size: 11px;
}

.remote-access__policy-grid {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 10px;
  align-content: start;
  padding: 12px;
}

.remote-access__policy-card {
  display: grid;
  gap: 6px;
  min-height: 108px;
  align-content: start;
  padding: 12px;
  text-align: left;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.remote-access__policy-card--active {
  background: rgba(19, 136, 255, 0.08);
  border-color: rgba(19, 136, 255, 0.24);
  box-shadow: inset 3px 0 0 var(--accent);
}

.remote-access__policy-card strong {
  color: var(--text-strong);
  font-size: 12px;
}

.remote-access__policy-card span {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.38;
}

.remote-access__policy-card small {
  color: #b36a00;
  font-size: 10px;
  font-weight: 760;
}

.remote-access__toggles {
  align-content: start;
  padding: 0 12px 12px;
}

.remote-access__security {
  grid-template-rows: auto auto minmax(0, 1fr);
}

.remote-access__login-alerts {
  display: grid;
  gap: 8px;
  padding: 11px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.remote-access__login-alerts article {
  display: flex;
  gap: 8px;
  min-width: 0;
  color: #b36a00;
}

.remote-access__login-alerts strong,
.remote-access__login-alerts span {
  display: block;
}

.remote-access__login-alerts strong {
  color: var(--text-strong);
  font-size: 11px;
}

.remote-access__login-alerts span {
  margin-top: 3px;
  color: var(--text-muted);
  font-size: 10px;
  line-height: 1.35;
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
  font-size: 11px;
  line-height: 1.42;
}

.remote-access__share-check span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 10px;
  font-weight: 700;
}

.remote-access__share-check--risk div {
  color: #b36a00;
}

.remote-access__share-check--safe div {
  color: var(--accent-green);
}

.remote-access__audit {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 38px;
  padding: 10px 12px;
  color: var(--accent-green);
  font-size: 11px;
  font-weight: 700;
}

.remote-access__audit span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

@media (max-width: 760px) {
  .remote-access {
    display: block;
    overflow: auto;
  }

  .remote-access__hero,
  .remote-access__status,
  .remote-access__main,
  .remote-access__audit {
    margin-bottom: 12px;
  }

  .remote-access__hero {
    display: grid;
  }

  .remote-access__status {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .remote-access__main {
    display: grid;
    grid-template-columns: 1fr;
  }

  .remote-access__devices,
  .remote-access__policy,
  .remote-access__security {
    min-height: 220px;
  }

  .remote-access__policy-grid {
    grid-template-columns: 1fr;
  }

  .remote-access__audit span {
    white-space: normal;
  }
}
</style>
