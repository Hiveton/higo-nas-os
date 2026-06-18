import { ref, shallowRef } from 'vue';
import type { UiTone } from '../tokens';

export type ConfirmOptions = {
  title: string;
  message?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  tone?: UiTone;
};

type PendingConfirm = ConfirmOptions & { resolve: (ok: boolean) => void };

export const confirmState = shallowRef<PendingConfirm | null>(null);
export const confirmOpen = ref(false);

/**
 * Imperative confirmation. Render <UiConfirmHost /> once at the app root, then:
 *   const confirm = useConfirm();
 *   if (await confirm({ title: '删除？', tone: 'danger' })) { ... }
 */
export function useConfirm() {
  return (options: ConfirmOptions) =>
    new Promise<boolean>((resolve) => {
      confirmState.value = {
        ...options,
        resolve: (ok: boolean) => {
          confirmOpen.value = false;
          resolve(ok);
        },
      };
      confirmOpen.value = true;
    });
}
