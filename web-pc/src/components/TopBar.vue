<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue';
import {
  Bell,
  Bot,
  ChevronDown,
  Cloud,
  Cpu,
  Download,
  HardDrive,
  LogOut,
  ShieldCheck,
  SlidersHorizontal,
  Upload,
  UserRound,
  Users,
} from 'lucide-vue-next';
import { monitoringStore } from '../stores/monitoring';
import { authStore } from '../stores/auth';
import TopSearch from './TopSearch.vue';
import type { Metric } from '../api/types';

const fallbackMetrics = [
  { label: 'CPU', value: '38%', icon: Cpu, tone: 'blue' },
  { label: '内存', value: '62%', icon: HardDrive, tone: 'green' },
  { label: '网络', value: '71%', icon: Download, tone: 'cyan' },
  { label: '磁盘', value: '46%', icon: HardDrive, tone: 'green' },
];

const metricIcons: Record<string, Component> = {
  cpu: Cpu,
  memory: HardDrive,
  network: Download,
  disk: HardDrive,
  temperature: Upload,
  fan: Cpu,
};

const modelPolicies = [
  { label: '本地模型', value: '优先', icon: Bot },
  { label: '云端模型', value: '增强', icon: Cloud },
];

const noticesOpen = ref(false);
const accountOpen = ref(false);
const topbarRef = ref<HTMLElement | null>(null);

const displayName = computed(
  () => authStore.currentUser.value?.displayName || authStore.currentUser.value?.username || '用户',
);
const avatarText = computed(() => (displayName.value.trim()[0] ?? 'H').toUpperCase());
const roleLabel = computed(() => {
  switch (authStore.role.value) {
    case 'admin':
      return '管理员';
    case 'user':
      return '成员';
    case 'guest':
      return '访客';
    default:
      return authStore.role.value;
  }
});
const canManageSecurity = authStore.canManageSecurity;
const canManageUsers = authStore.canManageUsers;

const topbarMetrics = computed(() => {
  const preferred = ['cpu', 'memory', 'network', 'disk'];
  const nextMetrics = preferred.flatMap((key) => {
    const metric = monitoringStore.metrics.value.find((item) => item.key === key);
    if (!metric) return [];
    return [{
      label: metric.label,
      value: formatMetricValue(metric),
      icon: metricIcons[metric.key ?? ''] ?? Cpu,
      tone: metric.tone ?? 'blue',
    }];
  });
  return nextMetrics.length ? nextMetrics : fallbackMetrics;
});

const notices = computed(() => {
  const alertNotices = monitoringStore.alerts.value
    .filter((alert) => !alert.muted)
    .map((alert) => `${alert.title}：${alert.detail}`)
    .slice(0, 3);
  return alertNotices.length
    ? alertNotices
    : ['AI 文件管家有 6 条整理建议', '3 个公开分享链接建议收紧权限', '家庭相册备份已完成 72%'];
});

const noticeCount = computed(() => Math.max(1, monitoringStore.alerts.value.filter((alert) => !alert.muted).length || notices.value.length));

const emit = defineEmits<{
  'topbar-action': [action: string];
}>();

function formatMetricValue(metric: Metric) {
  if (typeof metric.value === 'number') {
    return `${metric.value}${metric.unit ?? ''}`;
  }
  return `${metric.value}${metric.unit ?? ''}`;
}

function closePopovers() {
  noticesOpen.value = false;
  accountOpen.value = false;
}

function handleDocumentPointerDown(event: PointerEvent) {
  if (!topbarRef.value?.contains(event.target as Node)) {
    closePopovers();
  }
}

onMounted(() => {
  void monitoringStore.loadMonitoringSnapshot();
  monitoringStore.startPolling(5000);
  document.addEventListener('pointerdown', handleDocumentPointerDown);
});

onUnmounted(() => {
  monitoringStore.stopPolling();
  document.removeEventListener('pointerdown', handleDocumentPointerDown);
});
</script>

<template>
  <header ref="topbarRef" class="topbar" aria-label="HiGoOS 顶部状态栏" @keydown.esc="closePopovers">
    <div class="topbar__brand" aria-label="HiGoOS 在线">
      <span class="topbar__status-dot" aria-hidden="true" />
      <div>
        <strong>HiGoOS</strong>
        <span>家庭 NAS 中枢</span>
      </div>
    </div>

    <TopSearch @action="(a: string) => emit('topbar-action', a)" />

    <nav class="topbar__right" aria-label="系统状态与账户">
      <section class="topbar__metrics" aria-label="设备资源状态">
        <div
          v-for="metric in topbarMetrics"
          :key="metric.label"
          class="topbar__metric"
          :class="`topbar__metric--${metric.tone}`"
        >
          <component :is="metric.icon" :size="15" aria-hidden="true" />
          <span>{{ metric.label }}</span>
          <strong>{{ metric.value }}</strong>
        </div>
      </section>

      <section class="topbar__models" aria-label="模型策略">
        <div v-for="policy in modelPolicies" :key="policy.label" class="topbar__model">
          <component :is="policy.icon" :size="15" aria-hidden="true" />
          <span>{{ policy.label }}</span>
          <strong>{{ policy.value }}</strong>
        </div>
      </section>

      <button
        class="topbar__icon-button"
        type="button"
        aria-label="通知中心"
        @click="noticesOpen = !noticesOpen; accountOpen = false"
      >
        <Bell :size="18" aria-hidden="true" />
        <span class="topbar__notice-badge">{{ noticeCount }}</span>
      </button>

      <button
        class="topbar__avatar"
        type="button"
        aria-label="打开用户菜单"
        @click="accountOpen = !accountOpen; noticesOpen = false"
      >
        <span>{{ avatarText }}</span>
        <ChevronDown :size="14" aria-hidden="true" />
      </button>

      <section v-if="noticesOpen" class="topbar__popover topbar__popover--notice" aria-label="通知列表">
        <strong>通知中心</strong>
        <button
          v-for="notice in notices"
          :key="notice"
          type="button"
          @click="emit('topbar-action', 'notice')"
        >
          {{ notice }}
        </button>
      </section>

      <section v-if="accountOpen" class="topbar__popover topbar__popover--account" aria-label="用户菜单">
        <div class="topbar__account-head">
          <span class="topbar__account-avatar">{{ avatarText }}</span>
          <div class="topbar__account-id">
            <strong>{{ displayName }}</strong>
            <span class="topbar__account-role">{{ roleLabel }}</span>
          </div>
        </div>
        <div class="topbar__menu-group">
          <button type="button" class="topbar__menu-item" @click="emit('topbar-action', 'profile')">
            <UserRound :size="15" :stroke-width="2" /><span>个人设置</span>
          </button>
          <button v-if="canManageUsers" type="button" class="topbar__menu-item" @click="emit('topbar-action', 'permissions')">
            <Users :size="15" :stroke-width="2" /><span>用户中心</span>
          </button>
          <button v-if="canManageSecurity" type="button" class="topbar__menu-item" @click="emit('topbar-action', 'inspect')">
            <ShieldCheck :size="15" :stroke-width="2" /><span>安全中心</span>
          </button>
          <button type="button" class="topbar__menu-item" @click="emit('topbar-action', 'models')">
            <SlidersHorizontal :size="15" :stroke-width="2" /><span>模型策略设置</span>
          </button>
        </div>
        <button type="button" class="topbar__menu-item topbar__menu-item--danger" @click="emit('topbar-action', 'logout')">
          <LogOut :size="15" :stroke-width="2" /><span>退出桌面</span>
        </button>
      </section>
    </nav>
  </header>
</template>

<style scoped>
.topbar {
  position: absolute;
  inset: 16px 24px auto;
  z-index: 100;
  display: grid;
  grid-template-columns: minmax(170px, 0.8fr) minmax(280px, 1.25fr) auto;
  align-items: center;
  gap: 14px;
  min-height: 58px;
  padding: 10px 12px 10px 16px;
  color: var(--text-strong);
  background:
    linear-gradient(135deg, rgba(var(--surface-rgb), 0.72), rgba(var(--surface-rgb), 0.48)),
    rgba(var(--surface-rgb), 0.34);
  border: 1px solid rgba(255, 255, 255, 0.58);
  border-radius: clamp(20px, var(--radius-window), 28px);
  box-shadow:
    0 18px 54px rgba(23, 66, 101, 0.18),
    inset 0 1px 0 rgba(255, 255, 255, 0.72);
  backdrop-filter: blur(24px) saturate(1.3);
  -webkit-backdrop-filter: blur(24px) saturate(1.3);
}

.topbar__brand,
.topbar__right,
.topbar__metrics,
.topbar__models,
.topbar__metric,
.topbar__model,
.topbar__avatar,
.topbar__icon-button {
  display: flex;
  align-items: center;
}

.topbar__brand {
  gap: 10px;
  min-width: 0;
}

.topbar__brand strong {
  display: block;
  font-size: 18px;
  line-height: 1.1;
  letter-spacing: 0;
}

.topbar__brand span:last-child {
  display: block;
  margin-top: 3px;
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: 1;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.topbar__status-dot {
  width: 10px;
  height: 10px;
  border-radius: 50%;
  background: var(--accent-green);
  box-shadow:
    0 0 0 4px rgba(34, 181, 115, 0.13),
    0 0 18px rgba(34, 181, 115, 0.62);
}

.topbar__search {
  position: relative;
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  height: 38px;
  padding: 0 10px 0 13px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.62);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.82);
}

.topbar__search-popover {
  position: absolute;
  top: 46px;
  left: 0;
  z-index: 20;
  display: grid;
  width: min(420px, 80vw);
  gap: 5px;
  padding: 10px;
  background: rgba(var(--surface-rgb), 0.92);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(20px) saturate(1.2);
}

.topbar__search-popover p {
  margin: 0 0 2px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
}

.topbar__search-popover button {
  min-height: 30px;
  padding: 0 9px;
  color: var(--text);
  text-align: left;
  background: rgba(var(--surface-rgb), 0.48);
  border-radius: 9px;
  transition: background var(--duration-fast) var(--ease-standard);
}

.topbar__search-popover button:hover {
  background: var(--accent-soft);
}

.topbar__search input {
  width: 100%;
  min-width: 0;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
}

.topbar__search input::placeholder {
  color: rgba(77, 95, 116, 0.7);
}

.topbar__search kbd {
  flex: 0 0 auto;
  min-width: 36px;
  padding: 3px 7px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  font-family: inherit;
  text-align: center;
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.topbar__right {
  position: relative;
  justify-content: flex-end;
  gap: 10px;
  min-width: 0;
}

.topbar__popover {
  position: absolute;
  top: 44px;
  right: 0;
  z-index: 25;
  display: grid;
  min-width: 220px;
  gap: 6px;
  padding: 11px;
  color: var(--text);
  background: rgba(var(--surface-rgb), 0.92);
  border: 1px solid var(--border);
  border-radius: 14px;
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(20px) saturate(1.2);
}

.topbar__popover strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.topbar__popover--notice button {
  min-height: 31px;
  padding: 0 9px;
  color: var(--text);
  text-align: left;
  background: rgba(var(--surface-rgb), 0.48);
  border-radius: var(--radius-control);
  transition: background var(--duration-fast) var(--ease-standard);
}

.topbar__popover--notice button:hover {
  background: var(--accent-soft);
}

.topbar__popover--account {
  right: 0;
  min-width: 232px;
  gap: 8px;
  padding: 10px;
}

.topbar__account-head {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 8px 8px 10px;
  border-bottom: 1px solid var(--border);
}
.topbar__account-avatar {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  border-radius: var(--radius-pill);
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
  color: var(--text-inverse);
  font-size: var(--fs-md);
  font-weight: var(--fw-bold);
}
.topbar__account-id {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.topbar__account-id strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
}
.topbar__account-role {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}
.topbar__menu-group {
  display: grid;
  gap: 2px;
  padding-bottom: 8px;
  border-bottom: 1px solid var(--border);
}
.topbar__menu-item {
  display: flex;
  align-items: center;
  gap: 10px;
  min-height: 34px;
  padding: 0 10px;
  color: var(--text);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-control);
  font-size: var(--fs-sm);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard), color var(--duration-fast) var(--ease-standard);
}
.topbar__menu-item :deep(svg) {
  color: var(--text-muted);
}
.topbar__menu-item:hover {
  background: var(--accent-soft);
  color: var(--accent-deep);
}
.topbar__menu-item:hover :deep(svg) {
  color: var(--accent);
}
.topbar__menu-item--danger {
  color: var(--accent-red);
}
.topbar__menu-item--danger :deep(svg) {
  color: var(--accent-red);
}
.topbar__menu-item--danger:hover {
  background: var(--accent-red-soft);
  color: var(--accent-red);
}

.topbar__metrics,
.topbar__models {
  gap: 6px;
}

.topbar__metric,
.topbar__model {
  gap: 5px;
  min-width: 0;
  height: 34px;
  padding: 0 9px;
  font-size: var(--fs-xs);
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.56);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
}

.topbar__metric svg,
.topbar__model svg {
  flex: 0 0 auto;
}

.topbar__metric strong,
.topbar__model strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  white-space: nowrap;
}

.topbar__metric--blue svg {
  color: var(--accent);
}

.topbar__metric--green svg {
  color: var(--ink-green);
}

.topbar__metric--orange svg {
  color: var(--ink-orange);
}

.topbar__metric--cyan svg {
  color: var(--accent-cyan);
}

.topbar__models {
  padding-left: 10px;
  border-left: 1px solid var(--border);
}

.topbar__model {
  background: rgba(var(--surface-rgb), 0.58);
}

.topbar__model svg {
  color: var(--accent);
}

.topbar__icon-button,
.topbar__avatar {
  position: relative;
  flex: 0 0 auto;
  height: 36px;
  border: 1px solid var(--border);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.78);
}

.topbar__icon-button {
  justify-content: center;
  width: 36px;
  color: var(--text);
  background: rgba(var(--surface-rgb), 0.62);
  border-radius: 50%;
}

.topbar__notice-badge {
  position: absolute;
  top: -3px;
  right: -3px;
  min-width: 17px;
  height: 17px;
  padding: 0 4px;
  color: white;
  font-size: 10px;
  font-weight: var(--fw-bold);
  line-height: 17px;
  text-align: center;
  background: var(--accent-red);
  border-radius: var(--radius-pill);
  box-shadow: 0 0 0 2px rgba(255, 255, 255, 0.88);
}

.topbar__avatar {
  gap: 6px;
  padding: 0 8px 0 5px;
  color: var(--text-strong);
  background: rgba(var(--surface-rgb), 0.66);
  border-radius: var(--radius-pill);
}

.topbar__avatar span {
  display: grid;
  width: 27px;
  height: 27px;
  place-items: center;
  color: var(--text-inverse);
  font-size: var(--fs-sm);
  font-weight: var(--fw-bold);
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
  border-radius: 50%;
}

@media (max-width: 1180px) {
  .topbar {
    grid-template-columns: auto minmax(240px, 1fr) auto;
  }

  .topbar__metric span,
  .topbar__model span {
    display: none;
  }
}

@media (max-width: 1040px) {
  .topbar {
    gap: 8px;
    padding-inline: 12px;
  }

  .topbar__models {
    display: none;
  }

  .topbar__brand span:last-child {
    display: none;
  }
}

@media (max-width: 860px) {
  .topbar {
    inset: 10px 10px auto;
    grid-template-columns: 1fr auto;
  }

  .topbar__search {
    grid-column: 1 / -1;
    order: 3;
  }

  .topbar__metrics {
    display: none;
  }
}
</style>
