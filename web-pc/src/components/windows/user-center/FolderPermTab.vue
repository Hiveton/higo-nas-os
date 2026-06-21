<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { UiButton, UiSegmented, UiEmptyState, useToast } from '../../ui';
import type { SegmentedOption } from '../../ui';
import { sharedFoldersStore } from '../../../stores/sharedFolders';
import type { SharedFolderView, SharedFolderAccess } from '../../../api/types';

const props = defineProps<{ subjectType: 'user' | 'group'; subjectId: string }>();
const toast = useToast();

const accessOptions: SegmentedOption[] = [
  { label: '无访问', value: 'none' },
  { label: '只读', value: 'read' },
  { label: '读写', value: 'read_write' },
  { label: '拒绝', value: 'deny' },
];

const folders = computed(() => sharedFoldersStore.folders.value as unknown as SharedFolderView[]);
const draft = ref<Record<string, SharedFolderAccess>>({});
const saving = ref(false);

function currentAccess(folder: SharedFolderView): SharedFolderAccess {
  const p = folder.permissions.find((e) => e.subjectType === props.subjectType && e.subjectId === props.subjectId);
  return (p?.access as SharedFolderAccess) ?? 'none';
}

function reset() {
  const next: Record<string, SharedFolderAccess> = {};
  for (const f of folders.value) next[f.id] = currentAccess(f);
  draft.value = next;
}

onMounted(async () => {
  if (!folders.value.length) await sharedFoldersStore.load();
  reset();
});
watch(() => props.subjectId, reset);
watch(folders, reset);

async function save() {
  saving.value = true;
  try {
    const perms = folders.value.map((f) => ({ folderId: f.id, access: draft.value[f.id] ?? 'none' }));
    await sharedFoldersStore.setSubjectPermissions(props.subjectType, props.subjectId, perms);
    toast.success('文件夹权限已保存');
  } catch (error) {
    toast.error(`保存失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    saving.value = false;
  }
}
</script>

<template>
  <div class="fpt">
    <UiEmptyState v-if="!folders.length" title="还没有共享文件夹" description="先在「共享文件夹」里创建,再来这里分配权限。" />
    <template v-else>
      <p class="uc-panel__sub">为该{{ subjectType === 'group' ? '用户组' : '用户' }}分配各共享文件夹的访问权限（拒绝会覆盖其所在用户组的授权）。</p>
      <div class="fpt__rows">
        <div v-for="f in folders" :key="f.id" class="fpt__row">
          <div class="fpt__name">
            <strong>{{ f.name }}</strong>
            <small>{{ f.spaceName }}</small>
          </div>
          <UiSegmented v-model="draft[f.id]" :options="accessOptions" size="sm" />
        </div>
      </div>
      <div class="uc-actions">
        <UiButton tone="primary" :loading="saving" @click="save">保存文件夹权限</UiButton>
      </div>
    </template>
  </div>
</template>

<style scoped>
.fpt { display: flex; flex-direction: column; gap: 12px; }
.fpt__rows { display: flex; flex-direction: column; gap: 8px; max-height: 340px; overflow: auto; }
.fpt__row {
  display: flex; align-items: center; justify-content: space-between; gap: 12px;
  padding: 9px 12px; border-radius: var(--radius-control);
  background: rgba(var(--surface-rgb), 0.6); border: 1px solid var(--border);
}
.fpt__name { display: flex; flex-direction: column; min-width: 0; }
.fpt__name strong { color: var(--text-strong); font-size: var(--fs-sm); font-weight: var(--fw-medium); }
.fpt__name small { color: var(--text-soft); font-size: var(--fs-xs); }
</style>
