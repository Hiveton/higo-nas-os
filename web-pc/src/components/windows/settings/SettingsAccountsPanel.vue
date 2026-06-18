<script setup lang="ts">
import type { AccountGroup, AccountUser } from '../../../api/types';
import { UiButton, UiFormField, UiInput, UiSelect } from '../../ui';

defineProps<{
  accountState: string;
  accountBusy: string;
  newUser: {
    username: string;
    displayName: string;
    password: string;
    role: string;
    quotaGB: number;
    groupId: string;
  };
  newGroup: { name: string; description: string };
  memberEditor: { groupId: string; userId: string };
  grantEditor: {
    subjectType: string;
    subjectId: string;
    spaceId: string;
    access: string;
    quotaGB: number;
  };
  accountUsers: AccountUser[];
  accountGroups: AccountGroup[];
  grantSubjects: { id: string; label: string }[];
}>();

const emit = defineEmits<{
  (e: 'create-user'): void;
  (e: 'toggle-user', user: AccountUser): void;
  (e: 'delete-user', user: AccountUser): void;
  (e: 'create-group'): void;
  (e: 'add-member'): void;
  (e: 'grant-space'): void;
  (e: 'subject-type-change'): void;
}>();

const roleOptions = [
  { value: 'user', label: '普通用户' },
  { value: 'admin', label: '管理员' },
  { value: 'guest', label: '访客' },
];

const accessOptions = [
  { value: 'read', label: '只读' },
  { value: 'read_write', label: '读写' },
  { value: 'manage', label: '管理' },
];

const subjectTypeOptions = [
  { value: 'user', label: '用户' },
  { value: 'group', label: '用户组' },
];
</script>

<template>
  <div class="system-settings__panel">
    <div class="system-settings__account-status">
      <strong>账号与空间授权</strong>
      <span>{{ accountState }}</span>
    </div>

    <section class="system-settings__account-card" aria-label="创建用户">
      <h4>创建用户</h4>
      <div class="system-settings__account-form">
        <UiFormField label="用户名">
          <UiInput v-model="newUser.username" type="text" />
        </UiFormField>
        <UiFormField label="显示名">
          <UiInput v-model="newUser.displayName" type="text" />
        </UiFormField>
        <UiFormField label="密码">
          <UiInput v-model="newUser.password" type="password" />
        </UiFormField>
        <UiFormField label="角色">
          <UiSelect v-model="newUser.role" :options="roleOptions" />
        </UiFormField>
        <UiFormField label="配额 GB">
          <UiInput
            :model-value="newUser.quotaGB"
            type="number"
            @input="newUser.quotaGB = Number($event)"
          />
        </UiFormField>
        <UiFormField label="用户组">
          <UiSelect
            v-model="newUser.groupId"
            :options="[{ value: '', label: '无' }, ...accountGroups.map((g) => ({ value: g.id, label: g.name }))]"
          />
        </UiFormField>
        <UiButton :loading="accountBusy === 'create-user'" :disabled="accountBusy === 'create-user'" @click="emit('create-user')">
          {{ accountBusy === 'create-user' ? '创建中' : '创建用户' }}
        </UiButton>
      </div>
    </section>

    <section class="system-settings__account-card" aria-label="用户列表">
      <h4>用户</h4>
      <div class="system-settings__account-list">
        <article v-for="user in accountUsers" :key="user.id">
          <div>
            <strong>{{ user.displayName || user.username }}</strong>
            <small>{{ user.username }} · {{ user.role }} · {{ user.status }} · {{ Math.round(user.quotaBytes / 1024 / 1024 / 1024) }} GB</small>
          </div>
          <UiButton variant="ghost" :disabled="accountBusy === user.id" @click="emit('toggle-user', user)">
            {{ user.status === 'active' ? '停用' : '启用' }}
          </UiButton>
          <UiButton
            variant="ghost"
            tone="danger"
            :disabled="accountBusy === user.id || user.role === 'admin'"
            @click="emit('delete-user', user)"
          >
            删除
          </UiButton>
        </article>
      </div>
    </section>

    <section class="system-settings__account-card" aria-label="用户组和授权">
      <h4>用户组 / 授权</h4>
      <div class="system-settings__account-form">
        <UiFormField label="新用户组">
          <UiInput v-model="newGroup.name" type="text" />
        </UiFormField>
        <UiFormField label="描述">
          <UiInput v-model="newGroup.description" type="text" />
        </UiFormField>
        <UiButton :loading="accountBusy === 'create-group'" :disabled="accountBusy === 'create-group'" @click="emit('create-group')">
          创建组
        </UiButton>
        <UiFormField label="选择组">
          <UiSelect
            v-model="memberEditor.groupId"
            :options="accountGroups.map((g) => ({ value: g.id, label: g.name }))"
          />
        </UiFormField>
        <UiFormField label="添加成员">
          <UiSelect
            v-model="memberEditor.userId"
            :options="accountUsers.map((u) => ({ value: u.id, label: u.displayName || u.username }))"
          />
        </UiFormField>
        <UiButton :loading="accountBusy === 'members'" :disabled="accountBusy === 'members'" @click="emit('add-member')">
          保存成员
        </UiButton>
      </div>
      <div class="system-settings__account-form">
        <UiFormField label="授权类型">
          <UiSelect v-model="grantEditor.subjectType" :options="subjectTypeOptions" @change="emit('subject-type-change')" />
        </UiFormField>
        <UiFormField label="授权对象">
          <UiSelect
            v-model="grantEditor.subjectId"
            :options="grantSubjects.map((s) => ({ value: s.id, label: s.label }))"
          />
        </UiFormField>
        <UiFormField label="空间 ID">
          <UiInput v-model="grantEditor.spaceId" type="text" />
        </UiFormField>
        <UiFormField label="权限">
          <UiSelect v-model="grantEditor.access" :options="accessOptions" />
        </UiFormField>
        <UiFormField label="配额 GB">
          <UiInput
            :model-value="grantEditor.quotaGB"
            type="number"
            @input="grantEditor.quotaGB = Number($event)"
          />
        </UiFormField>
        <UiButton :loading="accountBusy === 'grant'" :disabled="accountBusy === 'grant'" @click="emit('grant-space')">
          保存授权
        </UiButton>
      </div>
    </section>
  </div>
</template>
