<script setup lang="ts">
import { ref } from 'vue';
import { UserPlus } from 'lucide-vue-next';
import {
  UiBadge,
  UiButton,
  UiCheckbox,
  UiDataTable,
  UiFormField,
  UiInput,
  UiModal,
  UiSelect,
  UiTabs,
  useConfirm,
  useToast,
} from '../../ui';
import type { Column, TabItem } from '../../ui';
import { accountsStore } from '../../../stores/accounts';
import type { AccountUser } from '../../../api/types';
import FolderPermTab from './FolderPermTab.vue';

const toast = useToast();
const confirm = useConfirm();

const columns: Column<AccountUser>[] = [
  { key: 'displayName', label: '用户', sortable: true },
  { key: 'role', label: '角色' },
  { key: 'status', label: '状态' },
  { key: 'quotaBytes', label: '配额' },
  { key: 'actions', label: '操作', align: 'right' },
];

const roleOptions = [
  { value: 'user', label: '成员' },
  { value: 'admin', label: '管理员' },
  { value: 'guest', label: '访客' },
];

const busy = ref('');

function roleLabel(role: string) {
  return role === 'admin' ? '管理员' : role === 'user' ? '成员' : role === 'guest' ? '访客' : role;
}
function statusLabel(status: string) {
  return status === 'active' ? '正常' : status === 'disabled' ? '已停用' : status === 'locked' ? '已锁定' : status;
}
function quota(bytes: number) {
  return bytes ? `${Math.round(bytes / 1024 / 1024 / 1024)} GB` : '不限';
}

// --- create ---
const showCreate = ref(false);
const form = ref({ username: '', displayName: '', password: '', role: 'user', quotaGB: 50, groupId: '' });

async function create() {
  if (!form.value.username.trim()) {
    toast.warning('请输入用户名');
    return;
  }
  busy.value = 'create';
  try {
    await accountsStore.createUser({
      username: form.value.username,
      displayName: form.value.displayName,
      password: form.value.password || 'Passw0rd1',
      role: form.value.role,
      quotaGB: form.value.quotaGB,
      groupId: form.value.groupId || undefined,
    });
    toast.success('用户已创建');
    showCreate.value = false;
    form.value = { username: '', displayName: '', password: '', role: 'user', quotaGB: 50, groupId: '' };
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '创建失败');
  } finally {
    busy.value = '';
  }
}

// --- edit (tabbed dialog) ---
const editTabs: TabItem[] = [
  { key: 'info', label: '信息' },
  { key: 'groups', label: '所属群组' },
  { key: 'folders', label: '共享文件夹权限' },
];
const showEdit = ref(false);
const editTab = ref('info');
const editing = ref<AccountUser | null>(null);
const editForm = ref({ displayName: '', role: 'user', status: 'active', password: '', quotaGB: 0 });
const editGroups = ref<Set<string>>(new Set());
const savingEdit = ref(false);

function openEdit(user: AccountUser) {
  editing.value = user;
  editTab.value = 'info';
  editForm.value = {
    displayName: user.displayName || '',
    role: user.role,
    status: user.status,
    password: '',
    quotaGB: Math.round((user.quotaBytes || 0) / 1024 / 1024 / 1024),
  };
  editGroups.value = new Set(user.groups ?? []);
  showEdit.value = true;
}

function toggleGroup(id: string, on: boolean) {
  const next = new Set(editGroups.value);
  if (on) next.add(id);
  else next.delete(id);
  editGroups.value = next;
}

async function saveEdit() {
  if (!editing.value) return;
  savingEdit.value = true;
  try {
    const payload: Record<string, unknown> = {
      displayName: editForm.value.displayName,
      role: editForm.value.role,
      status: editForm.value.status,
      quotaBytes: Math.round(Number(editForm.value.quotaGB) * 1024 * 1024 * 1024),
      groups: Array.from(editGroups.value),
    };
    if (editForm.value.password) payload.password = editForm.value.password;
    await accountsStore.updateUser(editing.value.id, payload);
    toast.success('用户已更新');
    showEdit.value = false;
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '保存失败');
  } finally {
    savingEdit.value = false;
  }
}

async function toggle(user: AccountUser) {
  busy.value = user.id;
  try {
    const status = await accountsStore.toggleUser(user);
    toast.success(`${user.displayName || user.username} 已${status === 'active' ? '启用' : '停用'}`);
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '操作失败');
  } finally {
    busy.value = '';
  }
}

async function remove(user: AccountUser) {
  if (!(await confirm({ title: '删除用户', message: `确定删除「${user.displayName || user.username}」？该操作不可撤销。`, confirmLabel: '删除', tone: 'danger' }))) {
    return;
  }
  busy.value = user.id;
  try {
    await accountsStore.deleteUser(user);
    toast.success('用户已删除');
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '删除失败');
  } finally {
    busy.value = '';
  }
}
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">用户</h3>
        <p class="uc-panel__sub">{{ accountsStore.users.value.length }} 个账号 · 系统用户与 HiGoOS 账号统一管理</p>
      </div>
      <UiButton :icon-left="UserPlus" size="sm" @click="showCreate = true">新建用户</UiButton>
    </header>

    <UiDataTable :columns="columns" :rows="accountsStore.users.value" row-key="id" :loading="accountsStore.loading.value">
      <template #cell-displayName="{ row }">
        <div class="uc-cell-stack">
          <strong>{{ row.displayName || row.username }}</strong>
          <span>{{ row.username }}</span>
        </div>
      </template>
      <template #cell-role="{ row }">
        <UiBadge :tone="row.role === 'admin' ? 'info' : 'neutral'">{{ roleLabel(row.role) }}</UiBadge>
      </template>
      <template #cell-status="{ row }">
        <UiBadge :tone="row.status === 'active' ? 'success' : row.status === 'locked' ? 'danger' : 'warning'">
          {{ statusLabel(row.status) }}
        </UiBadge>
      </template>
      <template #cell-quotaBytes="{ row }">{{ quota(row.quotaBytes) }}</template>
      <template #cell-actions="{ row }">
        <div class="uc-cell-actions">
          <UiButton variant="ghost" size="sm" @click="openEdit(row)">编辑</UiButton>
          <UiButton variant="ghost" size="sm" :disabled="busy === row.id" @click="toggle(row)">
            {{ row.status === 'active' ? '停用' : '启用' }}
          </UiButton>
          <UiButton variant="ghost" tone="danger" size="sm" :disabled="busy === row.id || row.role === 'admin'" @click="remove(row)">
            删除
          </UiButton>
        </div>
      </template>
    </UiDataTable>

    <!-- create -->
    <UiModal :open="showCreate" title="新建用户" size="sm" @update:open="showCreate = $event">
      <div class="uc-form" style="max-width: none">
        <UiFormField label="用户名"><UiInput v-model="form.username" /></UiFormField>
        <UiFormField label="显示名"><UiInput v-model="form.displayName" /></UiFormField>
        <UiFormField label="初始密码"><UiInput v-model="form.password" type="password" placeholder="留空则用默认 Passw0rd1" /></UiFormField>
        <UiFormField label="角色"><UiSelect v-model="form.role" :options="roleOptions" /></UiFormField>
        <UiFormField label="配额 (GB)"><UiInput :model-value="form.quotaGB" type="number" @input="form.quotaGB = Number($event)" /></UiFormField>
        <UiFormField label="加入用户组">
          <UiSelect
            v-model="form.groupId"
            :options="[{ value: '', label: '无' }, ...accountsStore.groups.value.map((g) => ({ value: g.id, label: g.name }))]"
          />
        </UiFormField>
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="showCreate = false">取消</UiButton>
        <UiButton :loading="busy === 'create'" @click="create">创建</UiButton>
      </template>
    </UiModal>

    <!-- edit (tabbed) -->
    <UiModal :open="showEdit" :title="`编辑用户 · ${editing?.displayName || editing?.username || ''}`" size="md" @update:open="showEdit = $event">
      <UiTabs v-model="editTab" :tabs="editTabs" variant="underline" size="sm" />

      <div v-show="editTab === 'info'" class="uc-form" style="max-width: none">
        <UiFormField label="用户名"><UiInput :model-value="editing?.username" disabled /></UiFormField>
        <UiFormField label="显示名"><UiInput v-model="editForm.displayName" /></UiFormField>
        <UiFormField label="角色"><UiSelect v-model="editForm.role" :options="roleOptions" /></UiFormField>
        <UiFormField label="状态">
          <UiSelect v-model="editForm.status" :options="[{ value: 'active', label: '正常' }, { value: 'disabled', label: '停用' }]" />
        </UiFormField>
        <UiFormField label="配额 (GB)"><UiInput :model-value="editForm.quotaGB" type="number" @input="editForm.quotaGB = Number($event)" /></UiFormField>
        <UiFormField label="重设密码"><UiInput v-model="editForm.password" type="password" placeholder="留空则不修改" /></UiFormField>
      </div>

      <div v-show="editTab === 'groups'">
        <div class="uc-checklist">
          <label v-for="g in accountsStore.groups.value" :key="g.id" class="uc-check-row">
            <UiCheckbox :model-value="editGroups.has(g.id)" @update:model-value="toggleGroup(g.id, $event)" />
            <span class="uc-check-name"><strong>{{ g.name }}</strong><small>{{ (g.userIds ?? []).length }} 名成员</small></span>
          </label>
          <p v-if="!accountsStore.groups.value.length" class="uc-panel__sub">还没有用户组。</p>
        </div>
      </div>

      <div v-show="editTab === 'folders'">
        <FolderPermTab v-if="editing" subject-type="user" :subject-id="editing.id" />
      </div>

      <template #footer>
        <UiButton variant="ghost" @click="showEdit = false">关闭</UiButton>
        <UiButton v-if="editTab !== 'folders'" :loading="savingEdit" @click="saveEdit">保存</UiButton>
      </template>
    </UiModal>
  </section>
</template>
