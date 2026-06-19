<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { UiBadge, UiSwitch, useToast } from '../../ui';
import { apiClient } from '../../../api/client';
import type { IdentityPolicy } from '../../../api/types';

const toast = useToast();
const identities = ref<IdentityPolicy[]>([]);
const loading = ref(false);

async function load() {
  loading.value = true;
  try {
    identities.value = await apiClient.security.getIdentities();
  } catch {
    identities.value = [];
  } finally {
    loading.value = false;
  }
}

async function save(row: IdentityPolicy) {
  if (!row.id) return;
  try {
    await apiClient.security.updateIdentityPermissions(row.id, {
      mfa: row.mfa,
      fileAcl: row.fileAcl,
      appAdmin: row.appAdmin,
      aiTools: row.aiTools,
    });
    toast.success(`${row.name || row.role} 策略已更新`);
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '更新失败');
    await load();
  }
}

onMounted(load);
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">身份策略</h3>
        <p class="uc-panel__sub">按角色控制 MFA 要求、文件 ACL、应用管理与 AI 工具访问</p>
      </div>
    </header>

    <p v-if="loading" class="uc-panel__sub">加载中…</p>
    <div v-else class="uc-cards">
      <article v-for="row in identities" :key="row.id ?? row.role" class="uc-card">
        <div class="uc-cards__head">
          <strong>{{ row.name || row.role }}</strong>
          <UiBadge tone="neutral">{{ row.role }}</UiBadge>
        </div>
        <div class="uc-toggles">
          <label class="uc-toggle-row"><span>强制 MFA</span><UiSwitch v-model="row.mfa" @change="save(row)" /></label>
          <label class="uc-toggle-row"><span>文件 ACL</span><UiSwitch v-model="row.fileAcl" @change="save(row)" /></label>
          <label class="uc-toggle-row"><span>应用管理</span><UiSwitch v-model="row.appAdmin" @change="save(row)" /></label>
          <label class="uc-toggle-row"><span>AI 工具</span><UiSwitch v-model="row.aiTools" @change="save(row)" /></label>
        </div>
      </article>
    </div>
  </section>
</template>

