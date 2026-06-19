<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { UiBadge, UiButton, UiDataTable } from '../../ui';
import type { Column } from '../../ui';
import { apiClient } from '../../../api/client';
import type { AuthAuditEntry } from '../../../api/types';

const rows = ref<AuthAuditEntry[]>([]);
const loading = ref(false);

const columns: Column<AuthAuditEntry>[] = [
  { key: 'time', label: '时间' },
  { key: 'actor', label: '操作者' },
  { key: 'action', label: '动作' },
  { key: 'result', label: '结果' },
  { key: 'sourceIp', label: '来源 IP' },
];

const actionLabel: Record<string, string> = {
  login: '登录',
  logout: '登出',
  login_mfa: 'MFA 校验',
  password_change: '修改密码',
  mfa_enable: '开启 MFA',
  mfa_disable: '关闭 MFA',
  session_revoke: '撤销会话',
  account_locked: '账号锁定',
  authz_denied: '越权拒绝',
};

function resultTone(result: string) {
  return result === 'allowed' ? 'success' : result === 'denied' || result === 'blocked' ? 'danger' : 'neutral';
}
function fmtTime(t: string) {
  const d = new Date(t);
  return Number.isNaN(d.getTime()) ? t : d.toLocaleString();
}

async function load() {
  loading.value = true;
  try {
    rows.value = await apiClient.auth.audit();
  } catch {
    rows.value = [];
  } finally {
    loading.value = false;
  }
}

onMounted(load);
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">账号审计</h3>
        <p class="uc-panel__sub">登录、改密、MFA、锁定与越权记录</p>
      </div>
      <UiButton variant="soft" size="sm" :loading="loading" @click="load">刷新</UiButton>
    </header>

    <UiDataTable :columns="columns" :rows="rows" row-key="id" :loading="loading">
      <template #cell-time="{ row }">{{ fmtTime(row.time) }}</template>
      <template #cell-action="{ row }">{{ actionLabel[row.action] ?? row.action }}</template>
      <template #cell-result="{ row }">
        <UiBadge :tone="resultTone(row.result)">{{ row.result }}</UiBadge>
      </template>
    </UiDataTable>
  </section>
</template>

