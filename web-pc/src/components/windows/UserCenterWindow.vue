<script setup lang="ts">
import { computed, onMounted, ref, type Component } from 'vue';
import { FolderTree, Users, UsersRound } from 'lucide-vue-next';
import { accountsStore } from '../../stores/accounts';
import { UiWindowPage, UiNavRail, UiNavItem } from '../ui';
import UsersPanel from './user-center/UsersPanel.vue';
import GroupsPanel from './user-center/GroupsPanel.vue';
import SharedFoldersPanel from './user-center/SharedFoldersPanel.vue';
import './user-center/user-center.css';

// DSM/fnOS 控制面板模型:用户 / 用户组 / 共享文件夹。窗口为管理员工具
// (App.vue 的 adminOnlyWindows 已限制非管理员打开);自助"个人设置"独立。
type NavKey = 'users' | 'groups' | 'folders';

const nav: { key: NavKey; label: string; icon: Component }[] = [
  { key: 'users', label: '用户', icon: Users },
  { key: 'groups', label: '用户组', icon: UsersRound },
  { key: 'folders', label: '共享文件夹', icon: FolderTree },
];

const active = ref<NavKey>('users');
const activeNav = computed(() => nav.find((item) => item.key === active.value) ?? nav[0]);

onMounted(() => {
  void accountsStore.load();
});
</script>

<template>
  <UiWindowPage layout="master-detail" :icon="Users" title="用户中心" :subtitle="activeNav.label">
    <template #nav>
      <UiNavRail title="用户中心" subtitle="用户 · 用户组 · 共享文件夹">
        <UiNavItem
          v-for="item in nav"
          :key="item.key"
          :icon="item.icon"
          :label="item.label"
          :active="active === item.key"
          @select="active = item.key"
        />
      </UiNavRail>
    </template>

    <UsersPanel v-if="active === 'users'" />
    <GroupsPanel v-else-if="active === 'groups'" />
    <SharedFoldersPanel v-else-if="active === 'folders'" />
  </UiWindowPage>
</template>
