<script setup lang="ts">
import { onMounted, ref, watch } from 'vue';
import { UiBadge, UiButton, UiSelect } from '../../ui';
import { useToast } from '../../ui';
import { accountsStore } from '../../../stores/accounts';
import { apiClient } from '../../../api/client';
import type { AuthSession } from '../../../api/types';

const toast = useToast();
const selectedUser = ref('');
const sessions = ref<AuthSession[]>([]);
const loading = ref(false);

async function load() {
  if (!selectedUser.value) {
    sessions.value = [];
    return;
  }
  loading.value = true;
  try {
    sessions.value = await apiClient.auth.listSessions(selectedUser.value);
  } catch {
    sessions.value = [];
  } finally {
    loading.value = false;
  }
}

async function revoke(session: AuthSession) {
  try {
    await apiClient.auth.revokeSession(session.id);
    toast.success('已撤销该会话');
    await load();
  } catch {
    toast.error('撤销失败');
  }
}

watch(selectedUser, load);
onMounted(() => {
  selectedUser.value = accountsStore.users.value[0]?.id ?? '';
});
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">全员会话与设备</h3>
        <p class="uc-panel__sub">选择用户查看其活动会话，可强制下线</p>
      </div>
      <UiSelect
        v-model="selectedUser"
        :options="accountsStore.users.value.map((u) => ({ value: u.id, label: u.displayName || u.username }))"
      />
    </header>

    <p v-if="loading" class="uc-panel__sub">加载中…</p>
    <p v-else-if="sessions.length === 0" class="uc-panel__sub">该用户当前没有活动会话。</p>
    <ul v-else class="uc-list">
      <li v-for="session in sessions" :key="session.id" class="uc-list__row">
        <div>
          <strong>{{ session.device || '未知设备' }}</strong>
          <UiBadge v-if="session.current" tone="success">当前</UiBadge>
          <div class="uc-list__meta">{{ session.ipAddress || '—' }} · 创建 {{ session.createdAt }} · 最近 {{ session.lastSeenAt }}</div>
        </div>
        <UiButton variant="ghost" tone="danger" size="sm" @click="revoke(session)">强制下线</UiButton>
      </li>
    </ul>
  </section>
</template>

