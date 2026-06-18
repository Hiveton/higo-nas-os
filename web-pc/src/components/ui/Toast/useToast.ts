import { readonly, ref } from 'vue';
import type { UiTone } from '../tokens';

export type ToastItem = {
  id: number;
  message: string;
  tone: UiTone;
  duration: number;
};

export type ToastOptions = {
  tone?: UiTone;
  duration?: number;
};

const toasts = ref<ToastItem[]>([]);
let seq = 0;
const timers = new Map<number, ReturnType<typeof setTimeout>>();

function dismiss(id: number) {
  toasts.value = toasts.value.filter((t) => t.id !== id);
  const timer = timers.get(id);
  if (timer) {
    clearTimeout(timer);
    timers.delete(id);
  }
}

function show(message: string, options: ToastOptions = {}) {
  const id = ++seq;
  const duration = options.duration ?? 3200;
  toasts.value = [...toasts.value, { id, message, tone: options.tone ?? 'neutral', duration }];
  if (duration > 0) {
    timers.set(
      id,
      setTimeout(() => dismiss(id), duration),
    );
  }
  return id;
}

/**
 * Global toast API. Import anywhere:
 *   const toast = useToast(); toast.success('已保存');
 * Render <UiToastHost /> once at the app root.
 */
export function useToast() {
  return {
    toasts: readonly(toasts),
    show,
    dismiss,
    success: (message: string, options?: ToastOptions) => show(message, { ...options, tone: 'success' }),
    error: (message: string, options?: ToastOptions) => show(message, { ...options, tone: 'danger' }),
    warning: (message: string, options?: ToastOptions) => show(message, { ...options, tone: 'warning' }),
    info: (message: string, options?: ToastOptions) => show(message, { ...options, tone: 'info' }),
  };
}
