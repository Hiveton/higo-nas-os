import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { SharedFolderView, SharedFolderPermission } from '../api/types';

type RecordPayload = Record<string, unknown>;

const folders = ref<SharedFolderView[]>([]);
const loading = ref(false);
const error = ref<Error | null>(null);

// Folders grouped by their owning storage space, for the left-pane list.
const groupedBySpace = computed(() => {
  const groups: { spaceId: string; spaceName: string; folders: SharedFolderView[] }[] = [];
  for (const folder of folders.value) {
    let group = groups.find((g) => g.spaceId === folder.spaceId);
    if (!group) {
      group = { spaceId: folder.spaceId, spaceName: folder.spaceName || folder.spaceId, folders: [] };
      groups.push(group);
    }
    group.folders.push(folder);
  }
  return groups;
});

export const sharedFoldersStore = {
  folders: readonly(folders),
  loading: readonly(loading),
  error: readonly(error),
  groupedBySpace,
  load,
  create,
  setPermissions,
  setSubjectPermissions,
  setService,
  setAdvanced,
  listSnapshots,
  createSnapshot,
  deleteSnapshot,
  deleteFolder,
};

export async function load() {
  loading.value = true;
  error.value = null;
  try {
    folders.value = await apiClient.sharedFolders.list();
  } catch (reason) {
    error.value = reason instanceof Error ? reason : new Error(String(reason));
  } finally {
    loading.value = false;
  }
}

function replace(view: SharedFolderView) {
  const idx = folders.value.findIndex((f) => f.id === view.id);
  if (idx >= 0) folders.value[idx] = view;
  else folders.value.push(view);
}

export async function create(payload: RecordPayload) {
  const view = await apiClient.sharedFolders.create(payload);
  replace(view);
  return view;
}

export async function setPermissions(id: string, entries: SharedFolderPermission[]) {
  const view = await apiClient.sharedFolders.setPermissions(id, { entries });
  replace(view);
  return view;
}

// setSubjectPermissions sets one user/group's access across many folders at once
// (the folder-permission tab of a user/group editor), then reloads.
export async function setSubjectPermissions(
  subjectType: 'user' | 'group',
  subjectId: string,
  perms: { folderId: string; access: SharedFolderView['permissions'][number]['access'] }[],
) {
  await apiClient.sharedFolders.setSubjectPermissions({ subjectType, subjectId, perms });
  await load();
}

export async function setService(id: string, protocol: string, enabled: boolean, guest = false) {
  const view = await apiClient.sharedFolders.setService(id, { protocol, enabled, guest });
  replace(view);
  return view;
}

// setAdvanced updates a folder's recycle/quota/encryption settings.
export async function setAdvanced(
  id: string,
  payload: { recycle?: boolean; quotaBytes?: number; encrypted?: boolean },
) {
  const view = await apiClient.sharedFolders.setAdvanced(id, payload);
  replace(view);
  return view;
}

export async function listSnapshots(id: string) {
  return apiClient.sharedFolders.listSnapshots(id);
}

export async function createSnapshot(id: string, name?: string) {
  return apiClient.sharedFolders.createSnapshot(id, { name });
}

export async function deleteSnapshot(id: string, name: string) {
  return apiClient.sharedFolders.deleteSnapshot(id, name);
}

// Governed 2-phase delete: preview then confirm.
export async function deleteFolder(id: string, removeDir = false) {
  const preview = await apiClient.sharedFolders.deletePreview(id);
  await apiClient.sharedFolders.deleteConfirm(id, { confirmationId: preview.confirmationId, removeDir });
  folders.value = folders.value.filter((f) => f.id !== id);
}
