<script setup lang="ts">
import { computed } from 'vue';
import { KeyRound, ShieldCheck, Users, UsersRound } from 'lucide-vue-next';
import { accountsStore } from '../../../stores/accounts';

const stats = computed(() => {
  const users = accountsStore.users.value;
  return [
    { key: 'users', label: '用户', value: users.length, icon: Users },
    { key: 'admins', label: '管理员', value: users.filter((u) => u.role === 'admin').length, icon: ShieldCheck },
    { key: 'groups', label: '用户组', value: accountsStore.groups.value.length, icon: UsersRound },
    { key: 'grants', label: '空间授权', value: accountsStore.grants.value.length, icon: KeyRound },
  ];
});

const disabledCount = computed(() => accountsStore.users.value.filter((u) => u.status !== 'active').length);
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">概览</h3>
        <p class="uc-panel__sub">账号、用户组与权限的整体情况</p>
      </div>
    </header>

    <div class="uc-stats">
      <article v-for="s in stats" :key="s.key" class="uc-stat">
        <component :is="s.icon" :size="20" class="uc-stat__icon" />
        <div class="uc-stat__value">{{ s.value }}</div>
        <div class="uc-stat__label">{{ s.label }}</div>
      </article>
    </div>

    <p class="uc-panel__sub">
      其中 {{ disabledCount }} 个账号被停用或锁定。账号身份以 Linux 系统用户为权威，系统已有用户会自动出现在「用户」列表中。
    </p>
  </section>
</template>
