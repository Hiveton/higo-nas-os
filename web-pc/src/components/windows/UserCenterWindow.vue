<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue';
import {
  ClipboardList,
  KeyRound,
  LayoutDashboard,
  Laptop,
  ShieldCheck,
  UserCircle,
  Users,
  UsersRound,
} from 'lucide-vue-next';
import { accountsStore } from '../../stores/accounts';
import { usePermissions } from '../../composables/usePermissions';
import OverviewPanel from './user-center/OverviewPanel.vue';
import MyAccountPanel from './user-center/MyAccountPanel.vue';
import UsersPanel from './user-center/UsersPanel.vue';
import GroupsPanel from './user-center/GroupsPanel.vue';
import GrantsPanel from './user-center/GrantsPanel.vue';
import IdentitiesPanel from './user-center/IdentitiesPanel.vue';
import SessionsPanel from './user-center/SessionsPanel.vue';
import AuditPanel from './user-center/AuditPanel.vue';
import './user-center/user-center.css';

type NavKey = 'overview' | 'me' | 'users' | 'groups' | 'grants' | 'identities' | 'sessions' | 'audit';

const { canManageUsers, canManageSecurity } = usePermissions();

const allNav: { key: NavKey; label: string; icon: Component; need?: 'users' | 'security' }[] = [
  { key: 'overview', label: '概览', icon: LayoutDashboard, need: 'users' },
  { key: 'me', label: '我的账号', icon: UserCircle },
  { key: 'users', label: '用户', icon: Users, need: 'users' },
  { key: 'groups', label: '用户组', icon: UsersRound, need: 'users' },
  { key: 'grants', label: '权限授权', icon: KeyRound, need: 'users' },
  { key: 'identities', label: '身份策略', icon: ShieldCheck, need: 'security' },
  { key: 'sessions', label: '全员会话', icon: Laptop, need: 'security' },
  { key: 'audit', label: '账号审计', icon: ClipboardList, need: 'security' },
];

const nav = computed(() =>
  allNav.filter((item) => {
    if (item.need === 'users') return canManageUsers.value;
    if (item.need === 'security') return canManageSecurity.value;
    return true;
  }),
);

const active = ref<NavKey>(canManageUsers.value ? 'overview' : 'me');

onMounted(() => {
  if (canManageUsers.value) void accountsStore.load();
});
</script>

<template>
  <div class="uc">
    <aside class="uc__rail">
      <div class="uc__rail-head">
        <Users :size="18" />
        <div>
          <strong>用户中心</strong>
          <span>账号 · 权限 · 存储</span>
        </div>
      </div>
      <nav class="uc__nav">
        <button
          v-for="item in nav"
          :key="item.key"
          type="button"
          class="uc__nav-item"
          :class="{ 'uc__nav-item--active': active === item.key }"
          @click="active = item.key"
        >
          <component :is="item.icon" :size="16" />
          <span>{{ item.label }}</span>
        </button>
      </nav>
    </aside>

    <main class="uc__main">
      <OverviewPanel v-if="active === 'overview'" />
      <MyAccountPanel v-else-if="active === 'me'" />
      <UsersPanel v-else-if="active === 'users'" />
      <GroupsPanel v-else-if="active === 'groups'" />
      <GrantsPanel v-else-if="active === 'grants'" />
      <IdentitiesPanel v-else-if="active === 'identities'" />
      <SessionsPanel v-else-if="active === 'sessions'" />
      <AuditPanel v-else-if="active === 'audit'" />
    </main>
  </div>
</template>
