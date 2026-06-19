import { authStore } from '../stores/auth';

// usePermissions exposes the auth store's derived permission flags to any
// component for conditional rendering / disabling. Prefer this over a custom
// directive — the project uses none and v-if/:disabled compose better with
// vue-tsc.
export function usePermissions() {
  return {
    role: authStore.role,
    isAdmin: authStore.isAdmin,
    canManageUsers: authStore.canManageUsers,
    canManageSecurity: authStore.canManageSecurity,
    canManageSystem: authStore.canManageSystem,
    has: (permission: string) => authStore.hasPermission(permission),
  };
}
