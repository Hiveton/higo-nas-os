import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { RemoteDevice, RemoteLoginAlert, RemoteStatus, ShareScanResult } from '../api/types';

const status = ref<RemoteStatus | null>(null);
const devices = ref<RemoteDevice[]>([]);
const loginAlerts = ref<RemoteLoginAlert[]>([]);
const shareScan = ref<ShareScanResult | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

export const remoteStore = {
  status: readonly(status),
  devices: readonly(devices),
  loginAlerts: readonly(loginAlerts),
  shareScan: readonly(shareScan),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  loadRemoteDashboard,
  startChannel,
  stopChannel,
  updateTunnelMode,
  toggleMfa,
  createDomainToken,
  rotateDomainToken,
  bindDevice,
  unbindDevice,
  selectPolicy,
  scanShareLinks,
};

export async function loadRemoteDashboard() {
  loading.value = true;
  error.value = null;

  try {
    const [nextStatus, nextDevices, nextAlerts] = await Promise.all([
      apiClient.remote.getStatus(),
      apiClient.remote.getDevices(),
      apiClient.remote.getLoginAlerts(),
    ]);
    status.value = nextStatus;
    devices.value = nextDevices;
    loginAlerts.value = nextAlerts;
    usingFallback.value = false;
    return nextStatus;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
    return status.value;
  } finally {
    loading.value = false;
  }
}

export async function startChannel() {
  return applyStatus(await apiClient.remote.startChannel());
}

export async function stopChannel() {
  return applyStatus(await apiClient.remote.stopChannel());
}

export async function updateTunnelMode(mode: RemoteStatus['tunnelMode']) {
  return applyStatus(await apiClient.remote.updateTunnelMode({ mode }));
}

export async function toggleMfa(enabled: boolean) {
  return applyStatus(await apiClient.remote.updateMfa(enabled));
}

export async function createDomainToken() {
  const token = await apiClient.remote.createDomainToken();
  status.value = {
    ...(status.value ?? createFallbackStatus()),
    domain: token.domain,
    token,
    feedback: `已复制 ${token.domain} 的短期访问令牌，有效期 10 分钟`,
  };
  usingFallback.value = false;
  return token;
}

export async function rotateDomainToken() {
  const token = await apiClient.remote.rotateDomainToken();
  status.value = {
    ...(status.value ?? createFallbackStatus()),
    domain: token.domain,
    token,
    feedback: `远程域名令牌已轮换，新域名 ${token.domain} 已生效`,
  };
  usingFallback.value = false;
  return token;
}

export async function bindDevice(id: string) {
  const device = await apiClient.remote.bindDevice(id);
  devices.value = devices.value.map((item) => (item.id === device.id ? device : item));
  await refreshStatus();
  usingFallback.value = false;
  return device;
}

export async function unbindDevice(id: string) {
  const device = await apiClient.remote.unbindDevice(id);
  devices.value = devices.value.map((item) => (item.id === device.id ? device : item));
  await refreshStatus();
  usingFallback.value = false;
  return device;
}

export async function selectPolicy(key: string) {
  return applyStatus(await apiClient.remote.selectPolicy(key));
}

export async function scanShareLinks() {
  const result = await apiClient.remote.scanShare();
  shareScan.value = result;
  status.value = {
    ...(status.value ?? createFallbackStatus()),
    feedback: result.message,
  };
  usingFallback.value = false;
  return result;
}

async function refreshStatus() {
  status.value = await apiClient.remote.getStatus();
}

function applyStatus(nextStatus: RemoteStatus) {
  status.value = nextStatus;
  usingFallback.value = false;
  return nextStatus;
}

function createFallbackStatus(): RemoteStatus {
  return {
    enabled: true,
    channelEnabled: true,
    mfaEnabled: true,
    tunnelMode: '智能中继',
    domain: 'home-3.higo.link',
  };
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}
