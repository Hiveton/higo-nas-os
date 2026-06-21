import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { ISCSIAuditEntry, ISCSICaps, ISCSITarget } from '../api/types';

type RecordPayload = Record<string, unknown>;

const targets = ref<ISCSITarget[]>([]);
const caps = ref<ISCSICaps | null>(null);
const audit = ref<ISCSIAuditEntry[]>([]);
const loading = ref(false);
const usingFallback = ref(false);

export const iscsiStore = {
  targets: readonly(targets),
  caps: readonly(caps),
  audit: readonly(audit),
  loading: readonly(loading),
  usingFallback: readonly(usingFallback),
  load,
  createTarget,
  deleteTarget,
  addLun,
  addAcl,
};

export async function load() {
  loading.value = true;
  try {
    const [nextTargets, nextCaps, nextAudit] = await Promise.all([
      apiClient.iscsi.getTargets(),
      apiClient.iscsi.getCapabilities(),
      apiClient.iscsi.getAudit(),
    ]);
    targets.value = nextTargets ?? [];
    caps.value = nextCaps;
    audit.value = nextAudit ?? [];
    usingFallback.value = false;
    return targets.value;
  } catch {
    usingFallback.value = true;
    if (targets.value.length === 0) targets.value = fallbackTargets();
    return targets.value;
  } finally {
    loading.value = false;
  }
}

export async function createTarget(payload?: RecordPayload) {
  const t = await apiClient.iscsi.createTarget({ ...(payload ?? {}), actor: 'iscsi-ui' });
  await load();
  return t;
}

export async function deleteTarget(iqn: string) {
  await apiClient.iscsi.deleteTarget(iqn);
  await load();
}

export async function addLun(iqn: string, name: string, sizeMB: number) {
  await apiClient.iscsi.addLun(iqn, { name, sizeMB, actor: 'iscsi-ui' });
  await load();
}

export async function addAcl(iqn: string, initiator: string) {
  await apiClient.iscsi.addAcl(iqn, { initiator, actor: 'iscsi-ui' });
  await load();
}

function fallbackTargets(): ISCSITarget[] {
  return [
    { iqn: 'iqn.2026-06.os.higo:storage.lab', luns: 1, acls: ['iqn.1994-05.com.redhat:lab-client'], portals: ['0.0.0.0:3260'] },
  ];
}
