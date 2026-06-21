<script setup lang="ts">
import { computed, useSlots, type Component } from 'vue';
import type { UiTone } from '../tokens';
import UiPageHeader from './PageHeader.vue';

withDefaults(
  defineProps<{
    title?: string;
    subtitle?: string;
    icon?: Component;
    status?: string;
    statusTone?: UiTone;
    layout?: 'master-detail' | 'dashboard' | 'stack' | 'chat';
  }>(),
  {
    title: undefined,
    subtitle: undefined,
    icon: undefined,
    status: undefined,
    statusTone: 'primary',
    layout: 'master-detail',
  },
);

const slots = useSlots();
const hasNav = computed(() => !!slots.nav);
const hasInspector = computed(() => !!slots.inspector);
</script>

<template>
  <div class="ui-window-page" :class="`ui-window-page--${layout}`">
    <slot name="header">
      <UiPageHeader
        v-if="title"
        :title="title"
        :subtitle="subtitle"
        :icon="icon"
        :status="status"
        :status-tone="statusTone"
      >
        <template v-if="$slots.actions" #actions><slot name="actions" /></template>
      </UiPageHeader>
    </slot>

    <div v-if="$slots.toolbar" class="ui-window-page__toolbar">
      <slot name="toolbar" />
    </div>

    <div
      class="ui-window-page__columns"
      :class="{ 'has-nav': hasNav, 'has-inspector': hasInspector }"
    >
      <aside v-if="hasNav" class="ui-window-page__nav">
        <slot name="nav" />
      </aside>
      <main class="ui-window-page__main">
        <slot />
      </main>
      <aside v-if="hasInspector" class="ui-window-page__inspector">
        <slot name="inspector" />
      </aside>
    </div>

    <div v-if="$slots.composer" class="ui-window-page__composer">
      <slot name="composer" />
    </div>
    <footer v-if="$slots.footer" class="ui-window-page__footer">
      <slot name="footer" />
    </footer>
  </div>
</template>

<style scoped>
.ui-window-page {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  height: 100%;
  min-height: 0;
}

.ui-window-page__toolbar,
.ui-window-page__composer,
.ui-window-page__footer {
  flex: 0 0 auto;
}

.ui-window-page__footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
  padding: var(--space-3) var(--space-4);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.ui-window-page__columns {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: var(--space-3);
  flex: 1 1 auto;
  min-height: 0;
}

/* master-detail / chat column configurations */
.ui-window-page__columns.has-nav {
  grid-template-columns: minmax(210px, 250px) minmax(0, 1fr);
}
.ui-window-page__columns.has-nav.has-inspector {
  grid-template-columns: minmax(210px, 250px) minmax(0, 1fr) minmax(260px, 320px);
}
.ui-window-page__columns.has-inspector:not(.has-nav) {
  grid-template-columns: minmax(0, 1fr) minmax(260px, 320px);
}

.ui-window-page__main {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-width: 0;
  min-height: 0;
  overflow: auto;
}

.ui-window-page__inspector {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  min-height: 0;
  overflow: auto;
}

.ui-window-page__nav {
  min-height: 0;
}

/* chat: the messages main shouldn't double-scroll; let inner content manage it */
.ui-window-page--chat .ui-window-page__main {
  overflow: hidden;
}

/* Narrow window body: collapse to a single column. The left nav becomes the
   off-canvas drawer (handled by DesktopWindow + base.css); inspector stacks
   below the main content. */
@container desktop-window-body (max-width: 760px) {
  .ui-window-page__columns,
  .ui-window-page__columns.has-nav,
  .ui-window-page__columns.has-nav.has-inspector,
  .ui-window-page__columns.has-inspector:not(.has-nav) {
    grid-template-columns: minmax(0, 1fr);
  }
}
</style>
