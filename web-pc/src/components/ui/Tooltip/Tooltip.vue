<script setup lang="ts">
import { nextTick, ref } from 'vue';

const props = withDefaults(
  defineProps<{
    content?: string;
    placement?: 'top' | 'bottom' | 'left' | 'right';
    delay?: number;
  }>(),
  { content: undefined, placement: 'top', delay: 120 },
);

const triggerRef = ref<HTMLElement | null>(null);
const visible = ref(false);
const coords = ref({ x: 0, y: 0 });
let timer: number | undefined;

function position() {
  const el = triggerRef.value;
  if (!el) return;
  const r = el.getBoundingClientRect();
  const gap = 8;
  switch (props.placement) {
    case 'bottom':
      coords.value = { x: r.left + r.width / 2, y: r.bottom + gap };
      break;
    case 'left':
      coords.value = { x: r.left - gap, y: r.top + r.height / 2 };
      break;
    case 'right':
      coords.value = { x: r.right + gap, y: r.top + r.height / 2 };
      break;
    default:
      coords.value = { x: r.left + r.width / 2, y: r.top - gap };
  }
}

function show() {
  window.clearTimeout(timer);
  timer = window.setTimeout(() => {
    visible.value = true;
    nextTick(position);
  }, props.delay);
}

function hide() {
  window.clearTimeout(timer);
  visible.value = false;
}
</script>

<template>
  <span
    ref="triggerRef"
    class="ui-tooltip-trigger"
    @pointerenter="show"
    @pointerleave="hide"
    @focusin="show"
    @focusout="hide"
  >
    <slot />
    <teleport to="body">
      <transition name="ui-tooltip-fade">
        <span
          v-if="visible && (content || $slots.content)"
          class="ui-tooltip"
          :class="`ui-tooltip--${placement}`"
          role="tooltip"
          :style="{ left: `${coords.x}px`, top: `${coords.y}px` }"
        >
          <slot name="content">{{ content }}</slot>
        </span>
      </transition>
    </teleport>
  </span>
</template>

<style scoped>
.ui-tooltip-trigger {
  display: inline-flex;
}

.ui-tooltip {
  position: fixed;
  z-index: var(--z-tooltip);
  max-width: 240px;
  padding: var(--space-1) var(--space-2);
  color: var(--text-inverse);
  font-size: var(--fs-xs);
  line-height: var(--lh-snug);
  white-space: nowrap;
  background: var(--surface-ink);
  border-radius: var(--radius-sm);
  box-shadow: var(--shadow-md);
  pointer-events: none;
}

.ui-tooltip--top {
  transform: translate(-50%, -100%);
}
.ui-tooltip--bottom {
  transform: translate(-50%, 0);
}
.ui-tooltip--left {
  transform: translate(-100%, -50%);
}
.ui-tooltip--right {
  transform: translate(0, -50%);
}

.ui-tooltip-fade-enter-active,
.ui-tooltip-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.ui-tooltip-fade-enter-from,
.ui-tooltip-fade-leave-to {
  opacity: 0;
}
</style>
