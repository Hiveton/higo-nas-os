<script setup lang="ts">
import { computed, ref } from 'vue';
import { KeyRound } from 'lucide-vue-next';
import {
  UiButton,
  UiBadge,
  UiDataTable,
  UiFormField,
  UiInput,
  UiSelect,
  useConfirm,
  useToast,
} from '../../ui';
import type { Column } from '../../ui';
import { accountsStore } from '../../../stores/accounts';
import type { AccountSpaceGrant } from '../../../api/types';

const toast = useToast();
const confirm = useConfirm();
const busy = ref('');

const subjectTypeOptions = [
  { value: 'user', label: '用户' },
  { value: 'group', label: '用户组' },
];
const accessOptions = [
  { value: 'read', label: '只读' },
  { value: 'read_write', label: '读写' },
  { value: 'manage', label: '管理' },
];

const form = ref({ subjectType: 'user', subjectId: '', spaceId: 'team-space', access: 'read_write', quotaGB: 100 });

// Shared space directories under the NAS root (grant targets become real
// setfacl ACLs on these directories).
const sharedSpaceOptions = [
  { value: 'home-space', label: '家庭空间' },
  { value: 'team-space', label: '团队空间' },
  { value: 'finance-receipts', label: '财务票据' },
  { value: 'photos-and-media', label: '照片与视频' },
  { value: 'backup-archive', label: '备份归档' },
  { value: 'downloads', label: '下载目录' },
];
const spaceOptions = computed(() => [
  ...sharedSpaceOptions,
  ...accountsStore.groups.value.map((g) => ({ value: g.id, label: `组共享 · ${g.name}` })),
]);

const subjectOptions = computed(() =>
  form.value.subjectType === 'group'
    ? accountsStore.groups.value.map((g) => ({ value: g.id, label: g.name }))
    : accountsStore.users.value.map((u) => ({ value: u.id, label: u.displayName || u.username })),
);

const columns: Column<AccountSpaceGrant>[] = [
  { key: 'subjectId', label: '对象' },
  { key: 'spaceId', label: '空间' },
  { key: 'access', label: '权限' },
  { key: 'quotaBytes', label: '配额' },
  { key: 'actions', label: '操作', align: 'right' },
];

function subjectLabel(grant: AccountSpaceGrant) {
  if (grant.subjectType === 'group') {
    return accountsStore.groups.value.find((g) => g.id === grant.subjectId)?.name ?? grant.subjectId;
  }
  const u = accountsStore.users.value.find((x) => x.id === grant.subjectId);
  return u?.displayName || u?.username || grant.subjectId;
}
function accessLabel(a: string) {
  return a === 'manage' ? '管理' : a === 'read_write' ? '读写' : '只读';
}

async function grant() {
  if (!form.value.subjectId || !form.value.spaceId.trim()) {
    toast.warning('请选择对象并填写空间 ID');
    return;
  }
  busy.value = 'grant';
  try {
    await accountsStore.grantSpace(form.value);
    toast.success('授权已写入');
    form.value.spaceId = '';
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '授权失败');
  } finally {
    busy.value = '';
  }
}

async function revoke(g: AccountSpaceGrant) {
  if (!(await confirm({ title: '撤销授权', message: '确定撤销这条空间授权？', confirmLabel: '撤销', tone: 'danger' }))) return;
  busy.value = g.id;
  try {
    await accountsStore.deleteGrant(g.id);
    toast.success('授权已撤销');
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '撤销失败');
  } finally {
    busy.value = '';
  }
}
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">空间权限授权</h3>
        <p class="uc-panel__sub">把空间的读写/管理权限授予用户或用户组（授权即在该目录写入真实 setfacl ACL）</p>
      </div>
    </header>

    <form class="uc-form-row uc-divider" @submit.prevent="grant">
      <UiFormField label="对象类型"><UiSelect v-model="form.subjectType" :options="subjectTypeOptions" /></UiFormField>
      <UiFormField label="授权对象"><UiSelect v-model="form.subjectId" :options="[{ value: '', label: '选择…' }, ...subjectOptions]" /></UiFormField>
      <UiFormField label="目标空间"><UiSelect v-model="form.spaceId" :options="spaceOptions" /></UiFormField>
      <UiFormField label="权限"><UiSelect v-model="form.access" :options="accessOptions" /></UiFormField>
      <UiFormField label="配额 (GB)"><UiInput :model-value="form.quotaGB" type="number" @input="form.quotaGB = Number($event)" /></UiFormField>
      <UiButton type="submit" :loading="busy === 'grant'" :icon-left="KeyRound">写入授权</UiButton>
    </form>

    <UiDataTable :columns="columns" :rows="accountsStore.grants.value" row-key="id" :loading="accountsStore.loading.value">
      <template #cell-subjectId="{ row }">
        <div class="uc-cell-stack"><strong>{{ subjectLabel(row) }}</strong><span>{{ row.subjectType === 'group' ? '用户组' : '用户' }}</span></div>
      </template>
      <template #cell-access="{ row }">
        <UiBadge :tone="row.access === 'manage' ? 'info' : row.access === 'read_write' ? 'success' : 'neutral'">{{ accessLabel(row.access) }}</UiBadge>
      </template>
      <template #cell-quotaBytes="{ row }">{{ row.quotaBytes ? Math.round(row.quotaBytes / 1024 / 1024 / 1024) + ' GB' : '不限' }}</template>
      <template #cell-actions="{ row }">
        <UiButton variant="ghost" tone="danger" size="sm" :disabled="busy === row.id" @click="revoke(row)">撤销</UiButton>
      </template>
    </UiDataTable>
  </section>
</template>

