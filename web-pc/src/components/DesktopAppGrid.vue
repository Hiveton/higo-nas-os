<script setup lang="ts">
import { onUnmounted, ref } from 'vue';
import type { DockApp } from '../data/higoos';

type IconPosition = {
  x: number;
  y: number;
};

const props = defineProps<{
  apps: DockApp[];
  positions: Record<string, IconPosition>;
  activeId?: string;
  runningIds?: string[];
  pinnedIds?: string[];
}>();

const emit = defineEmits<{
  'open-app': [id: string];
  'move-app': [payload: { id: string; x: number; y: number }];
  'contextmenu-app': [payload: { id: string; x: number; y: number }];
}>();

const desktopRef = ref<HTMLElement | null>(null);
const draggingId = ref('');
const dragStarted = ref(false);
let suppressNextClick = false;
let dragState:
  | {
      id: string;
      pointerId: number;
      startClientX: number;
      startClientY: number;
      startX: number;
      startY: number;
    }
  | undefined;

function isRunning(id: string) {
  return props.runningIds?.includes(id) ?? false;
}

function isPinned(id: string) {
  return props.pinnedIds?.includes(id) ?? false;
}

function openContextMenu(event: MouseEvent, id: string) {
  event.preventDefault();
  event.stopPropagation();
  emit('contextmenu-app', { id, x: event.clientX, y: event.clientY });
}

function getIconPosition(id: string) {
  return props.positions[id] ?? { x: 0, y: 0 };
}

function getStageBounds() {
  const stage = desktopRef.value?.parentElement;
  return {
    width: stage?.clientWidth ?? window.innerWidth,
    height: stage?.clientHeight ?? window.innerHeight,
  };
}

function clampPosition(x: number, y: number) {
  const bounds = getStageBounds();
  const minY = window.innerWidth > 1180 ? 72 : 8;
  return {
    x: Math.round(Math.min(Math.max(8, x), Math.max(8, bounds.width - 96))),
    y: Math.round(Math.min(Math.max(minY, y), Math.max(minY, bounds.height - 104))),
  };
}

function moveDraggedIcon(event: PointerEvent) {
  if (!dragState || event.pointerId !== dragState.pointerId) return;

  const deltaX = event.clientX - dragState.startClientX;
  const deltaY = event.clientY - dragState.startClientY;
  if (Math.abs(deltaX) > 3 || Math.abs(deltaY) > 3) {
    dragStarted.value = true;
    suppressNextClick = true;
  }

  const next = clampPosition(dragState.startX + deltaX, dragState.startY + deltaY);
  emit('move-app', { id: dragState.id, ...next });
}

function stopDragging(event: PointerEvent) {
  if (dragState && event.pointerId !== dragState.pointerId) return;
  window.removeEventListener('pointermove', moveDraggedIcon);
  window.removeEventListener('pointerup', stopDragging);
  window.removeEventListener('pointercancel', stopDragging);
  draggingId.value = '';
  dragState = undefined;
  window.setTimeout(() => {
    dragStarted.value = false;
  }, 0);
}

function startDragging(event: PointerEvent, id: string) {
  if (event.button !== 0) return;
  event.preventDefault();
  closeSelection(event);
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);
  const position = getIconPosition(id);
  dragState = {
    id,
    pointerId: event.pointerId,
    startClientX: event.clientX,
    startClientY: event.clientY,
    startX: position.x,
    startY: position.y,
  };
  draggingId.value = id;
  dragStarted.value = false;
  window.addEventListener('pointermove', moveDraggedIcon);
  window.addEventListener('pointerup', stopDragging, { once: true });
  window.addEventListener('pointercancel', stopDragging, { once: true });
}

function closeSelection(event: PointerEvent) {
  event.stopPropagation();
}

function openApp(event: MouseEvent, id: string) {
  event.stopPropagation();
  if (suppressNextClick) {
    suppressNextClick = false;
    return;
  }
  emit('open-app', id);
}

function getAppStyle(id: string) {
  const position = getIconPosition(id);
  return {
    left: `${position.x}px`,
    top: `${position.y}px`,
  };
}

onUnmounted(() => {
  window.removeEventListener('pointermove', moveDraggedIcon);
  window.removeEventListener('pointerup', stopDragging);
  window.removeEventListener('pointercancel', stopDragging);
});
</script>

<template>
  <nav ref="desktopRef" class="desktop-apps" aria-label="桌面应用图标">
    <button
      v-for="app in apps"
      :key="app.id"
      class="desktop-app"
      :style="getAppStyle(app.id)"
      :class="{
        'desktop-app--active': app.id === activeId,
        'desktop-app--running': isRunning(app.id),
        'desktop-app--pinned': isPinned(app.id),
        'desktop-app--dragging': draggingId === app.id,
      }"
      type="button"
      :aria-label="`打开${app.name}`"
      :aria-grabbed="draggingId === app.id"
      @pointerdown="startDragging($event, app.id)"
      @click="openApp($event, app.id)"
      @contextmenu="openContextMenu($event, app.id)"
    >
      <span class="desktop-app__icon">
        <img :src="app.icon" :alt="app.name" draggable="false" />
        <span v-if="app.badge" class="desktop-app__badge">{{ app.badge }}</span>
      </span>
      <span class="desktop-app__name">{{ app.name }}</span>
    </button>
  </nav>
</template>

<style scoped>
.desktop-apps {
  position: absolute;
  inset: 0;
  z-index: 18;
  overflow: visible;
  pointer-events: none;
}

.desktop-app {
  position: absolute;
  display: grid;
  width: 82px;
  min-height: 76px;
  justify-items: center;
  align-content: start;
  gap: 5px;
  padding: 6px 5px 5px;
  color: rgba(255, 255, 255, 0.96);
  text-align: center;
  text-shadow: 0 1px 3px rgba(9, 34, 58, 0.54);
  background: transparent;
  border: 1px solid transparent;
  border-radius: 12px;
  cursor: grab;
  pointer-events: auto;
  touch-action: none;
  user-select: none;
  transition:
    background 150ms ease,
    border-color 150ms ease,
    box-shadow 150ms ease,
    transform 150ms ease;
}

.desktop-app:hover,
.desktop-app:focus-visible,
.desktop-app--active {
  background: rgba(255, 255, 255, 0.16);
  border-color: rgba(255, 255, 255, 0.22);
  box-shadow: inset 0 1px 0 rgba(255, 255, 255, 0.18);
  outline: 0;
  backdrop-filter: blur(12px) saturate(1.16);
  -webkit-backdrop-filter: blur(12px) saturate(1.16);
}

.desktop-app--dragging {
  z-index: 2;
  cursor: grabbing;
  transform: scale(1.04);
}

.desktop-app__icon {
  position: relative;
  display: grid;
  width: 48px;
  height: 48px;
  place-items: center;
  filter: drop-shadow(0 11px 14px rgba(9, 34, 58, 0.28));
}

.desktop-app__icon img {
  width: 48px;
  height: 48px;
  object-fit: contain;
  transform: scale(1.28);
  user-select: none;
}

.desktop-app__badge {
  position: absolute;
  top: -5px;
  right: -7px;
  min-width: 18px;
  height: 18px;
  padding: 0 5px;
  color: #fff;
  font-size: 10px;
  font-weight: 800;
  line-height: 18px;
  text-shadow: none;
  background: linear-gradient(135deg, #ff4d63, #ef4444);
  border: 1px solid rgba(255, 255, 255, 0.82);
  border-radius: 999px;
}

.desktop-app__name {
  display: -webkit-box;
  width: 100%;
  min-height: 28px;
  overflow: hidden;
  font-size: 12px;
  font-weight: 760;
  line-height: 1.15;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 2;
}

@media (max-width: 1180px) {
  .desktop-apps {
    position: relative;
    inset: auto;
    display: grid;
    top: auto;
    right: auto;
    grid-auto-flow: row;
    grid-template-columns: repeat(auto-fill, minmax(72px, 1fr));
    grid-template-rows: none;
    width: min(100%, 720px);
    max-width: none;
    max-height: none;
    order: -1;
    align-self: stretch;
    padding: 2px 0 4px;
  }

  .desktop-app {
    position: relative !important;
    top: auto !important;
    left: auto !important;
    width: 100%;
    min-height: 72px;
    cursor: pointer;
    touch-action: manipulation;
  }
}

@media (max-width: 540px) {
  .desktop-apps {
    grid-template-columns: repeat(4, minmax(0, 1fr));
    gap: 8px 6px;
  }

  .desktop-app {
    min-height: 70px;
    padding-inline: 2px;
  }

  .desktop-app__icon,
  .desktop-app__icon img {
    width: 42px;
    height: 42px;
  }

  .desktop-app__name {
    font-size: 11px;
  }
}
</style>
