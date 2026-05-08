<script setup lang="ts">
import type { Component } from 'vue';

type ContextMenuItem = {
  id: string;
  label: string;
  hint?: string;
  icon?: Component;
  danger?: boolean;
  disabled?: boolean;
};

defineProps<{
  x: number;
  y: number;
  title: string;
  subtitle?: string;
  items: ContextMenuItem[];
}>();

const emit = defineEmits<{
  select: [id: string];
  close: [];
}>();
</script>

<template>
  <div
    class="context-menu"
    :style="{ left: `${x}px`, top: `${y}px` }"
    role="menu"
    :aria-label="title"
    @click.stop
    @contextmenu.prevent
  >
    <header class="context-menu__header">
      <strong>{{ title }}</strong>
      <span v-if="subtitle">{{ subtitle }}</span>
    </header>

    <button
      v-for="item in items"
      :key="item.id"
      class="context-menu__item"
      :class="{ 'context-menu__item--danger': item.danger }"
      type="button"
      role="menuitem"
      :disabled="item.disabled"
      @click="emit('select', item.id)"
    >
      <component :is="item.icon" v-if="item.icon" :size="15" stroke-width="2.2" />
      <span>{{ item.label }}</span>
      <small v-if="item.hint">{{ item.hint }}</small>
    </button>
  </div>
</template>

<style scoped>
.context-menu {
  position: fixed;
  z-index: 280;
  width: min(246px, calc(100vw - 24px));
  max-height: calc(100vh - 24px);
  padding: 8px;
  overflow-y: auto;
  color: var(--text);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.9), rgba(239, 248, 255, 0.78)),
    rgba(248, 253, 255, 0.86);
  border: 1px solid rgba(116, 151, 180, 0.28);
  border-radius: 14px;
  box-shadow:
    0 22px 54px rgba(21, 54, 85, 0.22),
    inset 0 1px 0 rgba(255, 255, 255, 0.86);
  backdrop-filter: blur(26px) saturate(1.2);
  -webkit-backdrop-filter: blur(26px) saturate(1.2);
  scrollbar-width: none;
}

.context-menu::-webkit-scrollbar {
  display: none;
}

.context-menu__header {
  display: grid;
  gap: 3px;
  padding: 8px 9px 9px;
  border-bottom: 1px solid rgba(108, 140, 169, 0.16);
}

.context-menu__header strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: 13px;
  line-height: 1.16;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.context-menu__header span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.context-menu__item {
  display: grid;
  grid-template-columns: 18px minmax(0, 1fr) auto;
  width: 100%;
  min-height: 34px;
  align-items: center;
  gap: 8px;
  padding: 7px 9px;
  color: var(--text);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: 9px;
}

.context-menu__item:hover,
.context-menu__item:focus-visible {
  background: rgba(19, 136, 255, 0.12);
  outline: 0;
}

.context-menu__item:disabled {
  cursor: not-allowed;
  opacity: 0.46;
}

.context-menu__item svg {
  color: var(--accent);
}

.context-menu__item span {
  overflow: hidden;
  font-size: 13px;
  font-weight: 690;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.context-menu__item small {
  color: var(--text-muted);
  font-size: 11px;
  white-space: nowrap;
}

.context-menu__item--danger {
  color: var(--accent-red);
}

.context-menu__item--danger svg {
  color: var(--accent-red);
}
</style>
