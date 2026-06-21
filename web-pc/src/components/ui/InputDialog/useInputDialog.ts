import { ref, shallowRef } from 'vue';

export type InputDialogOptions = {
  title: string;
  label?: string;
  defaultValue?: string;
  placeholder?: string;
  confirmLabel?: string;
  cancelLabel?: string;
  /** Return an error message to block submission, or null when valid. */
  validate?: (value: string) => string | null;
};

type PendingInput = InputDialogOptions & { resolve: (value: string | null) => void };

export const inputState = shallowRef<PendingInput | null>(null);
export const inputOpen = ref(false);

/**
 * Imperative text-input prompt — the modal replacement for window.prompt.
 * Render <UiInputDialogHost /> once at the app root, then:
 *   const inputDialog = useInputDialog();
 *   const name = await inputDialog({ title: '重命名', defaultValue: node.name });
 *   if (name) { ... }   // null when cancelled
 */
export function useInputDialog() {
  return (options: InputDialogOptions) =>
    new Promise<string | null>((resolve) => {
      inputState.value = {
        ...options,
        resolve: (value: string | null) => {
          inputOpen.value = false;
          resolve(value);
        },
      };
      inputOpen.value = true;
    });
}
