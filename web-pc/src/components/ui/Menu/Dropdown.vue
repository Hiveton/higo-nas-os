<script setup lang="ts">
import { nextTick, ref } from 'vue';
import { useDismiss } from '../composables/useDismiss';

const props = withDefaults(
  defineProps<{
    placement?: 'bottom-start' | 'bottom-end' | 'top-start' | 'top-end';
    trigger?: 'click' | 'hover';
    matchWidth?: boolean;
  }>(),
  { placement: 'bottom-start', trigger: 'click', matchWidth: false },
);

const rootRef = ref<HTMLElement | null>(null);
const menuRef = ref<HTMLElement | null>(null);
const open = ref(false);
const coords = ref({ x: 0, y: 0, width: 0 });

function position() {
  const el = rootRef.value;
  if (!el) return;
  const r = el.getBoundingClientRect();
  const top = props.placement.startsWith('top');
  const end = props.placement.endsWith('end');
  coords.value = {
    x: end ? r.right : r.left,
    y: top ? r.top : r.bottom,
    width: r.width,
  };
}

function toggle() {
  open.value = !open.value;
  if (open.value) nextTick(position);
}

function close() {
  open.value = false;
}

useDismiss({ ref: menuRef, active: open, onDismiss: close });

defineExpose({ close });
</script>

<template>
  <div
    ref="rootRef"
    class="ui-dropdown"
    @click="trigger === 'click' ? toggle() : undefined"
    @pointerenter="trigger === 'hover' ? ((open = true), nextTick(position)) : undefined"
    @pointerleave="trigger === 'hover' ? close() : undefined"
  >
    <slot name="trigger" :open="open" />
    <teleport to="body">
      <transition name="ui-dropdown-fade">
        <div
          v-if="open"
          ref="menuRef"
          class="ui-dropdown__menu u-glass"
          :class="`ui-dropdown__menu--${placement}`"
          role="menu"
          :style="{
            left: `${coords.x}px`,
            top: `${coords.y}px`,
            minWidth: matchWidth ? `${coords.width}px` : undefined,
          }"
          @click="trigger === 'click' ? close() : undefined"
        >
          <slot :close="close" />
        </div>
      </transition>
    </teleport>
  </div>
</template>

<style scoped>
.ui-dropdown {
  display: inline-flex;
}

.ui-dropdown__menu {
  position: fixed;
  z-index: var(--z-dropdown);
  min-width: 180px;
  max-height: calc(100vh - 32px);
  padding: var(--space-1);
  overflow-y: auto;
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
}

.ui-dropdown__menu--bottom-start {
  transform: translateY(6px);
}
.ui-dropdown__menu--bottom-end {
  transform: translate(-100%, 6px);
}
.ui-dropdown__menu--top-start {
  transform: translateY(-100%) translateY(-6px);
}
.ui-dropdown__menu--top-end {
  transform: translate(-100%, -100%) translateY(-6px);
}

.ui-dropdown-fade-enter-active,
.ui-dropdown-fade-leave-active {
  transition: opacity var(--duration-fast) var(--ease-out);
}
.ui-dropdown-fade-enter-from,
.ui-dropdown-fade-leave-to {
  opacity: 0;
}
</style>
