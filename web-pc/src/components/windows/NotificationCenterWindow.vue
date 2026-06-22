<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue';
import {
  Activity,
  Bell,
  CheckCheck,
  Inbox,
  RefreshCw,
  Sparkles,
  TriangleAlert,
} from 'lucide-vue-next';
import {
  UiButton,
  UiEmptyState,
  UiNavItem,
  UiNavRail,
  UiToolbar,
  UiWindowPage,
} from '../ui';
import { notificationsStore, type NotificationItem, type NotificationKind } from '../../stores/notifications';

type Filter = 'all' | NotificationKind;

const filter = ref<Filter>('all');

const FILTERS: { key: Filter; label: string; icon: Component }[] = [
  { key: 'all', label: '全部', icon: Inbox },
  { key: 'alert', label: '设备告警', icon: TriangleAlert },
  { key: 'steward', label: 'AI 建议', icon: Sparkles },
  { key: 'activity', label: '操作动态', icon: Activity },
];

const counts = computed(() => ({
  all: notificationsStore.items.value.length,
  alert: notificationsStore.grouped.value.alert.length,
  steward: notificationsStore.grouped.value.steward.length,
  activity: notificationsStore.grouped.value.activity.length,
}));

const visibleItems = computed<NotificationItem[]>(() =>
  filter.value === 'all'
    ? notificationsStore.items.value
    : notificationsStore.items.value.filter((item) => item.kind === filter.value),
);

const kindIcon: Record<NotificationKind, Component> = {
  alert: TriangleAlert,
  steward: Sparkles,
  activity: Activity,
};

function relativeTime(at?: string): string {
  if (!at) return '';
  const then = new Date(at).getTime();
  if (Number.isNaN(then)) return '';
  const diff = Date.now() - then;
  if (diff < 60_000) return '刚刚';
  if (diff < 3_600_000) return `${Math.floor(diff / 60_000)} 分钟前`;
  if (diff < 86_400_000) return `${Math.floor(diff / 3_600_000)} 小时前`;
  return new Date(at).toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit' });
}

function activate(item: NotificationItem) {
  notificationsStore.markRead(item.id);
  if (item.appId) {
    window.dispatchEvent(new CustomEvent('higoos:open-app', { detail: item.appId }));
  }
}

onMounted(() => notificationsStore.start());
onUnmounted(() => notificationsStore.stop());
</script>

<template>
  <UiWindowPage
    layout="master-detail"
    :icon="Bell"
    title="通知中心"
    :subtitle="`${notificationsStore.unreadCount.value} 条未读`"
  >
    <template #nav>
      <UiNavRail title="通知" subtitle="告警 · 建议 · 动态">
        <UiNavItem
          v-for="f in FILTERS"
          :key="f.key"
          :icon="f.icon"
          :label="f.label"
          :badge="counts[f.key] || undefined"
          :active="filter === f.key"
          @select="filter = f.key"
        />
      </UiNavRail>
    </template>

    <template #toolbar>
      <UiToolbar>
        <template #end>
          <UiButton variant="soft" size="sm" :icon-left="RefreshCw" :disabled="notificationsStore.loading.value" @click="notificationsStore.load()">刷新</UiButton>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="CheckCheck" :disabled="notificationsStore.unreadCount.value === 0" @click="notificationsStore.markAllRead()">全部已读</UiButton>
        </template>
      </UiToolbar>
    </template>

    <section class="notif" aria-label="通知列表">
      <button
        v-for="item in visibleItems"
        :key="item.id"
        class="notif__item"
        :class="[`notif__item--${item.tone}`, { 'notif__item--unread': !item.read }]"
        type="button"
        @click="activate(item)"
      >
        <span class="notif__dot" />
        <span class="notif__icon"><component :is="kindIcon[item.kind]" :size="16" /></span>
        <span class="notif__body">
          <strong>{{ item.title }}</strong>
          <span>{{ item.detail }}</span>
        </span>
        <span class="notif__time">{{ relativeTime(item.at) }}</span>
      </button>

      <UiEmptyState
        v-if="visibleItems.length === 0"
        :icon="Bell"
        title="暂无通知"
        description="设备告警、AI 整理建议和重要操作都会在这里汇总。"
      />
    </section>
  </UiWindowPage>
</template>

<style scoped>
.notif {
  display: grid;
  gap: 8px;
  align-content: start;
}

.notif__item {
  display: grid;
  grid-template-columns: 8px 34px minmax(0, 1fr) auto;
  align-items: center;
  gap: 11px;
  width: 100%;
  padding: 12px 14px;
  text-align: left;
  background: rgba(var(--surface-rgb), 0.55);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard),
    transform var(--duration-fast) var(--ease-standard);
}
.notif__item:hover {
  background: var(--accent-soft);
  transform: translateY(-1px);
}

.notif__dot {
  width: 8px;
  height: 8px;
  border-radius: 50%;
  background: transparent;
}
.notif__item--unread .notif__dot {
  background: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.notif__icon {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: var(--radius-control);
  background: rgba(var(--surface-rgb), 0.8);
}
.notif__item--blue .notif__icon { color: var(--accent); }
.notif__item--cyan .notif__icon { color: var(--accent-cyan); }
.notif__item--green .notif__icon { color: var(--ink-green); }
.notif__item--orange .notif__icon { color: var(--ink-orange); }
.notif__item--red .notif__icon { color: var(--accent-red); }

.notif__body {
  display: grid;
  gap: 3px;
  min-width: 0;
}
.notif__body strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-semibold);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.notif__item--unread .notif__body strong {
  font-weight: var(--fw-bold);
}
.notif__body span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.notif__time {
  flex: 0 0 auto;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  white-space: nowrap;
}
</style>
