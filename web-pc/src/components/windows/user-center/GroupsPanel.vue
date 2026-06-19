<script setup lang="ts">
import { computed, reactive, ref } from 'vue';
import { UsersRound } from 'lucide-vue-next';
import {
  UiBadge,
  UiButton,
  UiCheckbox,
  UiFormField,
  UiInput,
  UiModal,
  useToast,
} from '../../ui';
import { accountsStore } from '../../../stores/accounts';
import type { AccountGroup } from '../../../api/types';

const toast = useToast();
const busy = ref('');
const form = ref({ name: '', description: '' });

function userLabel(id: string) {
  const u = accountsStore.users.value.find((x) => x.id === id);
  return u?.displayName || u?.username || id;
}

async function createGroup() {
  if (!form.value.name.trim()) {
    toast.warning('请输入用户组名称');
    return;
  }
  busy.value = 'create';
  try {
    await accountsStore.createGroup(form.value);
    toast.success('用户组已创建');
    form.value = { name: '', description: '' };
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '创建失败');
  } finally {
    busy.value = '';
  }
}

// --- member management modal ---
const editor = reactive<{ open: boolean; group: AccountGroup | null; selected: Set<string>; saving: boolean }>({
  open: false,
  group: null,
  selected: new Set(),
  saving: false,
});

const selectedCount = computed(() => editor.selected.size);

function openMembers(group: AccountGroup) {
  editor.group = group;
  editor.selected = new Set(group.userIds ?? []);
  editor.open = true;
}

function toggle(id: string, on: boolean) {
  const next = new Set(editor.selected);
  if (on) next.add(id);
  else next.delete(id);
  editor.selected = next;
}

async function saveMembers() {
  if (!editor.group) return;
  editor.saving = true;
  try {
    await accountsStore.setGroupMembers(editor.group, Array.from(editor.selected));
    toast.success('成员已更新');
    editor.open = false;
  } catch (reason) {
    toast.error(reason instanceof Error ? reason.message : '更新失败');
  } finally {
    editor.saving = false;
  }
}
</script>

<template>
  <section class="uc-panel">
    <header class="uc-panel__head">
      <div>
        <h3 class="uc-panel__title">用户组</h3>
        <p class="uc-panel__sub">{{ accountsStore.groups.value.length }} 个组 · 用于批量授权与成员管理</p>
      </div>
    </header>

    <form class="uc-form-row uc-divider" @submit.prevent="createGroup">
      <UiFormField label="组名称"><UiInput v-model="form.name" placeholder="如 家庭成员" /></UiFormField>
      <UiFormField label="描述"><UiInput v-model="form.description" placeholder="可选" /></UiFormField>
      <UiButton type="submit" :loading="busy === 'create'" :icon-left="UsersRound">创建组</UiButton>
    </form>

    <div class="uc-cards">
      <article v-for="group in accountsStore.groups.value" :key="group.id" class="uc-card">
        <div class="uc-cards__head">
          <strong>{{ group.name }}</strong>
          <UiBadge tone="neutral">{{ (group.userIds ?? []).length }} 名成员</UiBadge>
        </div>
        <p v-if="group.description" class="uc-panel__sub">{{ group.description }}</p>
        <div class="uc-chips">
          <UiBadge v-for="uid in (group.userIds ?? []).slice(0, 6)" :key="uid" tone="neutral">{{ userLabel(uid) }}</UiBadge>
          <span v-if="(group.userIds ?? []).length > 6" class="uc-panel__sub">+{{ group.userIds.length - 6 }}</span>
          <span v-if="!(group.userIds ?? []).length" class="uc-panel__sub">暂无成员</span>
        </div>
        <div class="uc-actions" style="margin-top: 12px">
          <UiButton variant="soft" size="sm" @click="openMembers(group)">管理成员</UiButton>
        </div>
      </article>
    </div>

    <UiModal :open="editor.open" :title="`管理成员 · ${editor.group?.name ?? ''}`" size="sm" @update:open="editor.open = $event">
      <p class="uc-panel__sub" style="margin-bottom: 10px">勾选属于该组的用户（已选 {{ selectedCount }} 人）</p>
      <div class="uc-checklist">
        <label v-for="u in accountsStore.users.value" :key="u.id" class="uc-check-row">
          <UiCheckbox :model-value="editor.selected.has(u.id)" @update:model-value="toggle(u.id, $event)" />
          <span class="uc-check-name">
            <strong>{{ u.displayName || u.username }}</strong>
            <small>{{ u.username }} · {{ u.role === 'admin' ? '管理员' : u.role === 'guest' ? '访客' : '成员' }}</small>
          </span>
        </label>
        <p v-if="!accountsStore.users.value.length" class="uc-panel__sub">暂无用户</p>
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="editor.open = false">取消</UiButton>
        <UiButton :loading="editor.saving" @click="saveMembers">保存成员</UiButton>
      </template>
    </UiModal>
  </section>
</template>
