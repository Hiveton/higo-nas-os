import { computed, readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import { ApiError, setCsrfToken, setUnauthorizedHandler } from '../api/runtime';
import type { ChangePasswordInput, CurrentUser, LoginInput } from '../api/types';

type AuthPhase = 'loading' | 'authenticated' | 'unauthenticated';

// DEV_ADMIN_USER lets the desktop render when the backend is entirely
// unreachable (pure frontend work). Guarded by import.meta.env.DEV so it is
// tree-shaken out of production builds — never a real account.
const DEV_ADMIN_USER: CurrentUser = {
  id: 'admin',
  username: 'admin',
  displayName: '管理员',
  role: 'admin',
  status: 'active',
  quotaBytes: 0,
  groups: ['admins'],
  permissions: ['accounts:manage', 'security:manage', 'system:manage', 'files:write', 'files:read'],
};

const currentUser = ref<CurrentUser | null>(null);
const phase = ref<AuthPhase>('loading');
const loading = ref(false);
const error = ref<Error | null>(null);
const usingFallback = ref(false);

const role = computed(() => currentUser.value?.role ?? 'guest');
const isAdmin = computed(() => role.value === 'admin');
const permissionSet = computed(() => new Set(currentUser.value?.permissions ?? []));
const canManageUsers = computed(() => isAdmin.value || permissionSet.value.has('accounts:manage'));
const canManageSecurity = computed(() => isAdmin.value || permissionSet.value.has('security:manage'));
const canManageSystem = computed(() => isAdmin.value || permissionSet.value.has('system:manage'));

export const authStore = {
  currentUser: readonly(currentUser),
  phase: readonly(phase),
  loading: readonly(loading),
  error: readonly(error),
  usingFallback: readonly(usingFallback),
  role,
  isAdmin,
  canManageUsers,
  canManageSecurity,
  canManageSystem,
  refresh,
  login,
  logout,
  changePassword,
  hasPermission,
};

export async function refresh() {
  phase.value = currentUser.value ? phase.value : 'loading';
  try {
    const user = await apiClient.auth.me();
    applyUser(user);
    phase.value = 'authenticated';
    usingFallback.value = false;
    return user;
  } catch (reason) {
    if (reason instanceof ApiError && reason.status === 401) {
      markUnauthenticated();
      return null;
    }
    // Backend unreachable: keep dev usable with a local admin; otherwise treat
    // as logged out so the login screen can surface a retry.
    if (import.meta.env.DEV) {
      applyUser(DEV_ADMIN_USER);
      phase.value = 'authenticated';
      usingFallback.value = true;
      return DEV_ADMIN_USER;
    }
    error.value = normalizeError(reason);
    markUnauthenticated();
    return null;
  }
}

export async function login(input: LoginInput) {
  loading.value = true;
  error.value = null;
  try {
    const user = await apiClient.auth.login(input);
    applyUser(user);
    phase.value = 'authenticated';
    usingFallback.value = false;
    return user;
  } catch (reason) {
    error.value = normalizeError(reason);
    throw reason;
  } finally {
    loading.value = false;
  }
}

export async function logout() {
  try {
    await apiClient.auth.logout();
  } catch {
    // Even if the server call fails, drop local state so the UI locks.
  } finally {
    markUnauthenticated();
  }
}

export async function changePassword(input: ChangePasswordInput) {
  return apiClient.auth.changePassword(input);
}

export function hasPermission(permission: string) {
  return isAdmin.value || permissionSet.value.has(permission);
}

function applyUser(user: CurrentUser) {
  currentUser.value = user;
  setCsrfToken(user.csrfToken ?? null);
}

function markUnauthenticated() {
  currentUser.value = null;
  phase.value = 'unauthenticated';
  setCsrfToken(null);
}

function normalizeError(reason: unknown) {
  return reason instanceof Error ? reason : new Error(String(reason));
}

// A 401 on any guarded request flips the app back to the login screen.
setUnauthorizedHandler(() => {
  if (usingFallback.value) return; // dev fallback: ignore background 401s
  markUnauthenticated();
});
