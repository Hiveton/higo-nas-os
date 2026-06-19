import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type {
  Protocol,
  ProtocolAuditEntry,
  ProtocolConfig,
  ProtocolConfirmResult,
  ProtocolKey,
  ProtocolPreview,
  ProtocolShare,
} from '../api/types';

type RecordPayload = Record<string, unknown>;

const protocols = ref<Protocol[]>([]);
const shares = ref<ProtocolShare[]>([]);
const audit = ref<ProtocolAuditEntry[]>([]);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

const sharesByProtocol = computed(() => {
  const map: Record<string, ProtocolShare[]> = {};
  for (const share of shares.value) {
    (map[share.protocol] ??= []).push(share);
  }
  return map;
});

export const protocolsStore = {
  protocols: readonly(protocols),
  shares: readonly(shares),
  audit: readonly(audit),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  sharesByProtocol,
  loadDashboard,
  reloadShares,
  reloadAudit,
  previewEnable,
  previewDisable,
  previewCreateShare,
  previewDeleteShare,
  confirmEnable,
  confirmDisable,
  confirmCreateShare,
  confirmDeleteShare,
  setEnabled,
  updateConfig,
  rollback,
};

// setEnabled flips a protocol on/off in a single smooth action — it runs the
// governed preview→confirm pair under the hood (no extra dialog) so the UI
// toggle behaves like a normal NAS service switch. Returns the updated protocol.
export async function setEnabled(key: ProtocolKey, enabled: boolean) {
  const preview = enabled ? await previewEnable(key) : await previewDisable(key);
  const body = { confirmationId: preview.confirmationId ?? '', actor: 'protocols-ui' };
  const result = enabled
    ? await apiClient.protocols.confirmEnable(key, body)
    : await apiClient.protocols.confirmDisable(key, body);
  return applyConfirm(result);
}

export async function updateConfig(key: ProtocolKey, config: ProtocolConfig, actor = 'protocols-ui') {
  const updated = await apiClient.protocols.updateConfig(key, { config, actor });
  protocols.value = protocols.value.map((p) => (p.key === updated.key ? updated : p));
  void reloadAudit();
  usingFallback.value = false;
  return updated;
}

export async function loadDashboard() {
  loading.value = true;
  error.value = null;
  try {
    const [nextProtocols, nextShares, nextAudit] = await Promise.all([
      apiClient.protocols.list(),
      apiClient.protocols.getShares(),
      apiClient.protocols.getAudit(),
    ]);
    protocols.value = nextProtocols;
    shares.value = nextShares;
    audit.value = nextAudit;
    usingFallback.value = false;
    return nextProtocols;
  } catch (reason) {
    error.value = normalizeError(reason);
    usingFallback.value = true;
    if (protocols.value.length === 0) {
      protocols.value = fallbackProtocols();
    }
    return protocols.value;
  } finally {
    loading.value = false;
  }
}

export async function reloadShares() {
  shares.value = await apiClient.protocols.getShares();
}

export async function reloadAudit() {
  audit.value = await apiClient.protocols.getAudit();
}

// --- previews (no side effects) --------------------------------------------

export function previewEnable(key: ProtocolKey, actor = 'protocols-ui') {
  return apiClient.protocols.previewEnable(key, { actor });
}

export function previewDisable(key: ProtocolKey, actor = 'protocols-ui') {
  return apiClient.protocols.previewDisable(key, { actor });
}

export function previewCreateShare(key: ProtocolKey, payload: RecordPayload) {
  return apiClient.protocols.previewShare(key, payload);
}

export function previewDeleteShare(id: string, actor = 'protocols-ui') {
  return apiClient.protocols.previewDeleteShare(id, { actor });
}

// --- confirms (apply, then refresh) ----------------------------------------

export async function confirmEnable(key: ProtocolKey, preview: ProtocolPreview, actor = 'protocols-ui') {
  return applyConfirm(await apiClient.protocols.confirmEnable(key, confirmBody(preview, actor)));
}

export async function confirmDisable(key: ProtocolKey, preview: ProtocolPreview, actor = 'protocols-ui') {
  return applyConfirm(await apiClient.protocols.confirmDisable(key, confirmBody(preview, actor)));
}

export async function confirmCreateShare(key: ProtocolKey, preview: ProtocolPreview, actor = 'protocols-ui') {
  return applyConfirm(await apiClient.protocols.confirmShare(key, confirmBody(preview, actor)));
}

export async function confirmDeleteShare(id: string, preview: ProtocolPreview, actor = 'protocols-ui') {
  return applyConfirm(await apiClient.protocols.confirmDeleteShare(id, confirmBody(preview, actor)));
}

export async function rollback(auditId: string, actor = 'protocols-ui') {
  await apiClient.protocols.rollbackAudit(auditId, { actor });
  await loadDashboard();
}

function applyConfirm(result: ProtocolConfirmResult) {
  if (result.protocol) {
    protocols.value = protocols.value.map((p) => (p.key === result.protocol!.key ? result.protocol! : p));
  }
  // Shares + audit change shape enough that a refresh is the safe path.
  void loadDashboard();
  usingFallback.value = false;
  return result;
}

function confirmBody(preview: ProtocolPreview, actor: string): RecordPayload {
  return { confirmationId: preview.confirmationId ?? '', actor };
}

function fallbackProtocols(): Protocol[] {
  return [
    { key: 'smb', displayName: 'SMB / 文件共享', enabled: false, running: false, installed: true, mountHint: '\\\\HiGoOS\\<共享名>', port: 445, compatibility: 'Windows 资源管理器、macOS 访达、Android 文件管理器', config: {} },
    { key: 'nfs', displayName: 'NFS', enabled: false, running: false, installed: true, mountHint: 'nfs://HiGoOS/<导出路径>', port: 2049, compatibility: 'Linux、macOS、ESXi 等 *nix 客户端', config: {} },
    { key: 'webdav', displayName: 'WebDAV', enabled: false, running: false, installed: true, mountHint: 'http://HiGoOS:8081/<共享名>', port: 8081, compatibility: '浏览器、RaiDrive、各类 WebDAV 客户端', config: {} },
    { key: 'dlna', displayName: 'DLNA', enabled: false, running: false, installed: true, mountHint: 'DLNA：HiGoOS 媒体服务器', port: 8200, compatibility: '智能电视、投影仪、PS / Xbox 等 DLNA 设备', config: {} },
  ];
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}
