import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { VM, VmAuditEntry, VmHostCaps } from '../api/types';

const machines = ref<VM[]>([]);
const caps = ref<VmHostCaps | null>(null);
const audit = ref<VmAuditEntry[]>([]);
const loading = ref(false);
const usingFallback = ref(false);

export const vmStore = {
  machines: readonly(machines),
  caps: readonly(caps),
  audit: readonly(audit),
  loading: readonly(loading),
  usingFallback: readonly(usingFallback),
  load,
  action,
};

export async function load() {
  loading.value = true;
  try {
    const [nextMachines, nextCaps, nextAudit] = await Promise.all([
      apiClient.vm.getMachines(),
      apiClient.vm.getCapabilities(),
      apiClient.vm.getAudit(),
    ]);
    machines.value = nextMachines ?? [];
    caps.value = nextCaps;
    audit.value = nextAudit ?? [];
    usingFallback.value = false;
    return machines.value;
  } catch {
    usingFallback.value = true;
    if (machines.value.length === 0) machines.value = fallbackMachines();
    return machines.value;
  } finally {
    loading.value = false;
  }
}

export async function action(name: string, act: string) {
  const updated = await apiClient.vm.action(name, act, { actor: 'vm-ui' });
  await load();
  return updated;
}

function fallbackMachines(): VM[] {
  return [
    { name: 'home-assistant', state: 'running', vcpus: 2, memoryMB: 2048, autostart: true, persistent: true, title: '智能家居' },
    { name: 'ubuntu-lab', state: 'shut off', vcpus: 4, memoryMB: 4096, autostart: false, persistent: true, title: '实验环境' },
  ];
}
