<script setup lang="ts">
import { nextTick, ref, watch } from 'vue';
import { X } from 'lucide-vue-next';
import { useModalStack } from '../composables/useModalStack';

const props = withDefaults(
  defineProps<{
    open: boolean;
    title?: string;
    size?: 'sm' | 'md' | 'lg' | 'fullscreen';
    closeOnBackdrop?: boolean;
    closeOnEsc?: boolean;
    /** Hide the default header close button. */
    hideClose?: boolean;
  }>(),
  {
    title: undefined,
    size: 'md',
    closeOnBackdrop: true,
    closeOnEsc: true,
    hideClose: false,
  },
);

const emit = defineEmits<{
  'update:open': [value: boolean];
  close: [];
  open: [];
}>();

const stack = useModalStack();
const panelRef = ref<HTMLElement | null>(null);
let lastFocused: HTMLElement | null = null;

function close() {
  emit('update:open', false);
  emit('close');
}

function onBackdrop() {
  if (props.closeOnBackdrop) close();
}

function onKeydown(event: KeyboardEvent) {
  if (!stack.isTop()) return;
  if (event.key === 'Escape' && props.closeOnEsc) {
    event.stopPropagation();
    close();
    return;
  }
  if (event.key === 'Tab') trapFocus(event);
}

function focusables(): HTMLElement[] {
  if (!panelRef.value) return [];
  return Array.from(
    panelRef.value.querySelectorAll<HTMLElement>(
      'a[href], button:not([disabled]), input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])',
    ),
  ).filter((el) => el.offsetParent !== null);
}

function trapFocus(event: KeyboardEvent) {
  const items = focusables();
  if (items.length === 0) {
    event.preventDefault();
    panelRef.value?.focus();
    return;
  }
  const first = items[0];
  const last = items[items.length - 1];
  const active = document.activeElement as HTMLElement | null;
  if (event.shiftKey && active === first) {
    event.preventDefault();
    last.focus();
  } else if (!event.shiftKey && active === last) {
    event.preventDefault();
    first.focus();
  }
}

watch(
  () => props.open,
  (open) => {
    if (open) {
      stack.push();
      lastFocused = document.activeElement as HTMLElement | null;
      document.addEventListener('keydown', onKeydown, true);
      emit('open');
      nextTick(() => {
        const items = focusables();
        (items[0] ?? panelRef.value)?.focus();
      });
    } else {
      stack.pop();
      document.removeEventListener('keydown', onKeydown, true);
      lastFocused?.focus?.();
      lastFocused = null;
    }
  },
);
</script>

<template>
  <teleport to="body">
    <transition name="ui-modal-fade">
      <div
        v-if="open"
        class="ui-modal-backdrop"
        :style="{ zIndex: stack.backdropZIndex() }"
        @pointerdown.self="onBackdrop"
      >
        <div
          ref="panelRef"
          class="ui-modal u-glass"
          :class="`ui-modal--${size}`"
          :style="{ zIndex: stack.panelZIndex() }"
          role="dialog"
          aria-modal="true"
          :aria-label="title"
          tabindex="-1"
        >
          <header v-if="title || $slots.title || !hideClose" class="ui-modal__header">
            <div class="ui-modal__title">
              <slot name="title">{{ title }}</slot>
            </div>
            <button
              v-if="!hideClose"
              class="ui-modal__close"
              type="button"
              aria-label="关闭"
              @click="close"
            >
              <X :size="16" :stroke-width="2.2" />
            </button>
          </header>

          <div class="ui-modal__body">
            <slot />
          </div>

          <footer v-if="$slots.footer" class="ui-modal__footer">
            <slot name="footer" />
          </footer>
        </div>
      </div>
    </transition>
  </teleport>
</template>

<style scoped>
.ui-modal-backdrop {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-6);
  background: rgba(10, 28, 48, 0.34);
  backdrop-filter: blur(4px);
  -webkit-backdrop-filter: blur(4px);
}

.ui-modal {
  display: flex;
  flex-direction: column;
  width: 100%;
  max-height: calc(100vh - var(--space-12));
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-lg);
  overflow: hidden;
}

.ui-modal--sm {
  max-width: 380px;
}
.ui-modal--md {
  max-width: 560px;
}
.ui-modal--lg {
  max-width: 820px;
}
.ui-modal--fullscreen {
  max-width: calc(100vw - var(--space-12));
  max-height: calc(100vh - var(--space-12));
  height: calc(100vh - var(--space-12));
}

.ui-modal__header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-4) var(--space-5);
  border-bottom: 1px solid var(--border);
}

.ui-modal__title {
  color: var(--text-strong);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
}

.ui-modal__close {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 30px;
  height: 30px;
  color: var(--text-muted);
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}
.ui-modal__close:hover {
  background: var(--surface-glass);
}

.ui-modal__body {
  flex: 1;
  padding: var(--space-5);
  overflow-y: auto;
  color: var(--text);
}

.ui-modal__footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: var(--space-2);
  padding: var(--space-4) var(--space-5);
  border-top: 1px solid var(--border);
}

.ui-modal-fade-enter-active,
.ui-modal-fade-leave-active {
  transition: opacity var(--duration-md) var(--ease-out);
}
.ui-modal-fade-enter-active .ui-modal,
.ui-modal-fade-leave-active .ui-modal {
  transition: transform var(--duration-md) var(--ease-out);
}
.ui-modal-fade-enter-from,
.ui-modal-fade-leave-to {
  opacity: 0;
}
.ui-modal-fade-enter-from .ui-modal,
.ui-modal-fade-leave-to .ui-modal {
  transform: translateY(12px) scale(0.98);
}
</style>
