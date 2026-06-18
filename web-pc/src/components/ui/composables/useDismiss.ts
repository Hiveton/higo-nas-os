import { onBeforeUnmount, watch, type Ref } from 'vue';

type DismissOptions = {
  /** Element(s) considered "inside"; clicks within are ignored. */
  ref: Ref<HTMLElement | null>;
  /** Whether the dismissable surface is currently open. */
  active: Ref<boolean>;
  /** Called when an outside click or Escape press should close it. */
  onDismiss: () => void;
  /** Disable Escape handling. Default false (Escape closes). */
  ignoreEscape?: boolean;
};

/**
 * Outside-click + Escape dismissal for dropdowns, menus and popovers.
 * Listeners are only attached while `active` is true.
 */
export function useDismiss({ ref, active, onDismiss, ignoreEscape }: DismissOptions) {
  function onPointerDown(event: PointerEvent) {
    const el = ref.value;
    if (!el) return;
    if (event.target instanceof Node && el.contains(event.target)) return;
    onDismiss();
  }

  function onKeyDown(event: KeyboardEvent) {
    if (ignoreEscape) return;
    if (event.key === 'Escape') {
      event.stopPropagation();
      onDismiss();
    }
  }

  function attach() {
    document.addEventListener('pointerdown', onPointerDown, true);
    document.addEventListener('keydown', onKeyDown, true);
  }

  function detach() {
    document.removeEventListener('pointerdown', onPointerDown, true);
    document.removeEventListener('keydown', onKeyDown, true);
  }

  watch(
    active,
    (open) => {
      detach();
      if (open) attach();
    },
    { immediate: true },
  );

  onBeforeUnmount(detach);
}
