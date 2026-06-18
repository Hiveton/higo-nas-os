<script setup lang="ts">
import { CheckCircle2, AlertTriangle, XCircle, Info, X } from 'lucide-vue-next';
import type { Component } from 'vue';
import type { UiTone } from '../tokens';
import { useToast } from './useToast';

const { toasts, dismiss } = useToast();

const icons: Partial<Record<UiTone, Component>> = {
  success: CheckCircle2,
  warning: AlertTriangle,
  danger: XCircle,
  info: Info,
};
</script>

<template>
  <teleport to="body">
    <div class="ui-toast-host" role="region" aria-live="polite" aria-label="通知">
      <transition-group name="ui-toast">
        <div
          v-for="item in toasts"
          :key="item.id"
          class="ui-toast u-glass"
          :class="`ui-toast--${item.tone}`"
          role="status"
        >
          <component :is="icons[item.tone]" v-if="icons[item.tone]" class="ui-toast__icon" :size="17" :stroke-width="2.2" />
          <span class="ui-toast__message">{{ item.message }}</span>
          <button class="ui-toast__close" type="button" aria-label="关闭" @click="dismiss(item.id)">
            <X :size="14" :stroke-width="2.4" />
          </button>
        </div>
      </transition-group>
    </div>
  </teleport>
</template>

<style scoped>
.ui-toast-host {
  position: fixed;
  top: calc(var(--topbar-height) + var(--space-3));
  right: var(--space-4);
  z-index: var(--z-toast);
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  max-width: min(360px, calc(100vw - 32px));
  pointer-events: none;
}

.ui-toast {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-3) var(--space-3);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-md);
  pointer-events: auto;
  --toast-color: var(--text-strong);
}
.ui-toast--success {
  --toast-color: var(--accent-green);
}
.ui-toast--warning {
  --toast-color: var(--accent-orange);
}
.ui-toast--danger {
  --toast-color: var(--accent-red);
}
.ui-toast--info {
  --toast-color: var(--accent-cyan);
}

.ui-toast__icon {
  flex-shrink: 0;
  color: var(--toast-color);
}
.ui-toast__message {
  flex: 1;
  color: var(--text);
  font-size: var(--fs-sm);
  line-height: var(--lh-snug);
}
.ui-toast__close {
  display: inline-flex;
  flex-shrink: 0;
  padding: 2px;
  color: var(--text-soft);
  background: transparent;
  border: 0;
  border-radius: 50%;
  cursor: pointer;
}
.ui-toast__close:hover {
  color: var(--text);
}

.ui-toast-enter-active,
.ui-toast-leave-active {
  transition: opacity var(--duration-md) var(--ease-out), transform var(--duration-md) var(--ease-out);
}
.ui-toast-enter-from,
.ui-toast-leave-to {
  opacity: 0;
  transform: translateX(16px);
}
</style>
