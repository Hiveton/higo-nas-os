import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { AccountGroup, AccountSummary, AccountUser } from '../api/types';

// accountsStore owns multi-user administration state (users / groups / space
// grants), previously inlined in SystemSettingsWindow. Thin wrapper over
// apiClient.accounts; components render from it and surface errors via toast.

const summary = ref<AccountSummary>({ users: [], groups: [], grants: [] });
const loading = ref(false);
const error = ref<Error | null>(null);

const users = computed(() => summary.value.users ?? []);
const groups = computed(() => summary.value.groups ?? []);
const grants = computed(() => summary.value.grants ?? []);

export const accountsStore = {
  summary: readonly(summary),
  users,
  groups,
  grants,
  loading: readonly(loading),
  error: readonly(error),
  load,
  createUser,
  updateUser,
  toggleUser,
  deleteUser,
  createGroup,
  updateGroupMembers,
  setGroupMembers,
  grantSpace,
  deleteGrant,
};

export async function load() {
  loading.value = true;
  error.value = null;
  try {
    summary.value = await apiClient.accounts.getSummary();
    return summary.value;
  } catch (reason) {
    error.value = normalizeError(reason);
    throw reason;
  } finally {
    loading.value = false;
  }
}

export async function createUser(payload: {
  username: string;
  displayName: string;
  password: string;
  role: string;
  quotaGB: number;
  groupId?: string;
}) {
  const user = await apiClient.accounts.createUser({
    username: payload.username.trim(),
    displayName: payload.displayName.trim() || payload.username.trim(),
    password: payload.password,
    role: payload.role,
    quotaBytes: Math.round(Number(payload.quotaGB) * 1024 * 1024 * 1024),
    groups: payload.groupId ? [payload.groupId] : [],
  });
  await load();
  return user;
}

export async function updateUser(id: string, payload: Record<string, unknown>) {
  const user = await apiClient.accounts.updateUser(id, payload);
  await load();
  return user;
}

export async function toggleUser(user: AccountUser) {
  const status = user.status === 'active' ? 'disabled' : 'active';
  await apiClient.accounts.updateUser(user.id, { status });
  await load();
  return status;
}

export async function deleteUser(user: AccountUser) {
  await apiClient.accounts.deleteUser(user.id);
  await load();
}

export async function createGroup(payload: { name: string; description: string }) {
  const group = await apiClient.accounts.createGroup({
    name: payload.name.trim(),
    description: payload.description.trim(),
  });
  await load();
  return group;
}

export async function updateGroupMembers(group: AccountGroup, userId: string, action: 'add' | 'remove') {
  const current = new Set(group.userIds ?? []);
  if (action === 'add') current.add(userId);
  else current.delete(userId);
  return setGroupMembers(group, Array.from(current));
}

// setGroupMembers replaces a group's entire membership in one call.
export async function setGroupMembers(group: AccountGroup, userIds: string[]) {
  const updated = await apiClient.accounts.updateGroupMembers(group.id, userIds);
  await load();
  return updated;
}

export async function grantSpace(payload: {
  subjectType: string;
  subjectId: string;
  spaceId: string;
  access: string;
  quotaGB: number;
}) {
  const grant = await apiClient.accounts.grantSpace({
    subjectType: payload.subjectType,
    subjectId: payload.subjectId,
    spaceId: payload.spaceId.trim(),
    access: payload.access,
    quotaBytes: Math.round(Number(payload.quotaGB) * 1024 * 1024 * 1024),
  });
  await load();
  return grant;
}

export async function deleteGrant(id: string) {
  await apiClient.accounts.deleteGrant(id);
  await load();
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}
