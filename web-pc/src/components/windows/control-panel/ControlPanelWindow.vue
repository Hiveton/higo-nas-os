<script setup lang="ts">
import { computed, onMounted, onUnmounted, ref, type Component } from 'vue';
import {
  Activity,
  Cpu,
  FileClock,
  FolderTree,
  Globe2,
  Network,
  ShieldCheck,
  SlidersHorizontal,
  Users,
  UsersRound,
} from 'lucide-vue-next';
import { UiNavItem, UiNavRail, UiWindowPage } from '../../ui';
import { accountsStore } from '../../../stores/accounts';

// Self-contained panels render directly (clean 2-level rail → content). The
// remaining inline windows are embedded as interim sections (they own their
// state/lifecycle); extracting their content is a later refinement.
import UsersPanel from '../user-center/UsersPanel.vue';
import GroupsPanel from '../user-center/GroupsPanel.vue';
import SharedFoldersPanel from '../user-center/SharedFoldersPanel.vue';
import RemoteAccessWindow from '../RemoteAccessWindow.vue';
import ProtocolsWindow from '../ProtocolsWindow.vue';
import SystemSettingsWindow from '../SystemSettingsWindow.vue';
import HardwareCenterWindow from '../HardwareCenterWindow.vue';
import DeviceMonitorWindow from '../DeviceMonitorWindow.vue';
import SecurityCenterWindow from '../SecurityCenterWindow.vue';
import LogCenterWindow from '../LogCenterWindow.vue';
import '../user-center/user-center.css';

type SectionKey =
  | 'users' | 'groups' | 'shared' | 'remote' | 'protocols'
  | 'settings' | 'hardware' | 'device' | 'security' | 'logs';

type NavGroup = {
  id: string;
  label: string;
  items: { key: SectionKey; label: string; icon: Component }[];
};

const groups: NavGroup[] = [
  {
    id: 'connection',
    label: '连接与访问',
    items: [
      { key: 'users', label: '用户', icon: Users },
      { key: 'groups', label: '用户组', icon: UsersRound },
      { key: 'shared', label: '共享权限', icon: FolderTree },
      { key: 'remote', label: '远程访问 (HiGoLink)', icon: Globe2 },
      { key: 'protocols', label: '文件服务', icon: Network },
    ],
  },
  {
    id: 'general',
    label: '通用设置',
    items: [{ key: 'settings', label: '系统设置', icon: SlidersHorizontal }],
  },
  {
    id: 'services',
    label: '系统服务',
    items: [
      { key: 'hardware', label: '硬件与电源', icon: Cpu },
      { key: 'device', label: '设备分析', icon: Activity },
      { key: 'security', label: '安全性', icon: ShieldCheck },
      { key: 'logs', label: '日志中心', icon: FileClock },
    ],
  },
];

const allItems = computed(() => groups.flatMap((g) => g.items));
const active = ref<SectionKey>('users');
const activeLabel = computed(() => allItems.value.find((i) => i.key === active.value)?.label ?? '');

const SECTION_KEYS: SectionKey[] = ['users', 'groups', 'shared', 'remote', 'protocols', 'settings', 'hardware', 'device', 'security', 'logs'];

// Deep-link: other windows (e.g. the top-bar account menu) can jump straight to
// a Control Panel section via this event after opening the window.
function handleSectionEvent(event: Event) {
  const key = (event as CustomEvent<string>).detail;
  if (typeof key === 'string' && (SECTION_KEYS as string[]).includes(key)) {
    active.value = key as SectionKey;
  }
}

onMounted(() => {
  void accountsStore.load();
  window.addEventListener('higoos:control-panel-section', handleSectionEvent);
});
onUnmounted(() => window.removeEventListener('higoos:control-panel-section', handleSectionEvent));

defineExpose({ openSection: (key: SectionKey) => { active.value = key; } });
</script>

<template>
  <UiWindowPage layout="master-detail" :icon="SlidersHorizontal" title="控制面板" :subtitle="activeLabel">
    <template #nav>
      <UiNavRail title="控制面板" subtitle="连接 · 通用 · 系统服务">
        <template v-for="group in groups" :key="group.id">
          <p class="cp__group-label">{{ group.label }}</p>
          <UiNavItem
            v-for="item in group.items"
            :key="item.key"
            :icon="item.icon"
            :label="item.label"
            :active="active === item.key"
            @select="active = item.key"
          />
        </template>
      </UiNavRail>
    </template>

    <!-- 自包含面板:直接渲染(干净两级) -->
    <UsersPanel v-if="active === 'users'" />
    <GroupsPanel v-else-if="active === 'groups'" />
    <SharedFoldersPanel v-else-if="active === 'shared'" />

    <!-- 内联窗口:暂以 embed 复用(自管生命周期) -->
    <div v-else class="cp__embed">
      <RemoteAccessWindow v-if="active === 'remote'" />
      <ProtocolsWindow v-else-if="active === 'protocols'" />
      <SystemSettingsWindow v-else-if="active === 'settings'" />
      <HardwareCenterWindow v-else-if="active === 'hardware'" />
      <DeviceMonitorWindow v-else-if="active === 'device'" />
      <SecurityCenterWindow v-else-if="active === 'security'" />
      <LogCenterWindow v-else-if="active === 'logs'" />
    </div>
  </UiWindowPage>
</template>

<style scoped>
.cp__group-label {
  margin: var(--space-2) var(--space-2) 2px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  letter-spacing: 0.04em;
}
.cp__group-label:first-child {
  margin-top: 0;
}
/* The embedded window owns its own height + scroll. */
.cp__embed {
  display: flex;
  flex-direction: column;
  min-height: 0;
  height: 100%;
}
</style>
