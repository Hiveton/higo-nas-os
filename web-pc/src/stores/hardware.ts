import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { HardwareInventory } from '../api/types';

// Representative inventory shown when the backend is unreachable (e.g. the API
// is down on a dev host). Mirrors the backend DevAdapter so the console always
// renders something coherent. See [[optimistic-fallback-catch-blocks]].
const fallbackInventory: HardwareInventory = {
  host: {
    hostname: 'higoos-dev',
    osName: 'HiGoOS Dev',
    kernel: 'dev-stub',
    arch: 'arm64',
    uptime: '开发桩数据',
  },
  board: { vendor: 'HiGoOS', product: 'Dev Workstation', serial: 'DEV-0000-0000', bios: 'dev' },
  cpu: { model: '开发桩 CPU', physicalCores: 4, logicalCores: 8, mhz: 2800 },
  memory: { totalBytes: 34359738368, usedBytes: 21303037788, usedPercent: 62 },
  network: [
    { name: 'en0', mac: '02:42:ac:11:00:02', ipv4: ['10.211.55.20'], speedMbps: 1000, state: 'up' },
    { name: 'en1', mac: '02:42:ac:11:00:03', ipv4: [], speedMbps: 0, state: 'down' },
  ],
  sensors: {
    temperatures: [{ label: 'CPU', celsius: 43 }],
    fans: [{ label: 'System', rpm: 1280 }],
  },
  adapter: 'fallback',
};

const inventory = ref<HardwareInventory | null>(null);
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

export const hardwareStore = {
  inventory: readonly(inventory),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  loadInventory,
};

export async function loadInventory() {
  loading.value = true;
  error.value = null;
  try {
    inventory.value = await apiClient.hardware.getInventory();
    usingFallback.value = false;
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
    if (!inventory.value) {
      inventory.value = fallbackInventory;
    }
    usingFallback.value = true;
  } finally {
    loading.value = false;
  }
}
