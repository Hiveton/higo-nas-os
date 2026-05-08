<script setup lang="ts">
import { X } from 'lucide-vue-next';
import type { DesktopWindowConfig } from '../api/types';

type WindowGeometry = {
  x: number;
  y: number;
  width: number;
  height: number;
};

const props = defineProps<{
  window: DesktopWindowConfig;
  active: boolean;
  maximized?: boolean;
}>();

const emit = defineEmits<{
  focus: [];
  close: [];
  minimize: [];
  'toggle-maximize': [];
  'move-window': [geometry: Pick<WindowGeometry, 'x' | 'y'>];
  'resize-window': [geometry: WindowGeometry];
  'contextmenu-window': [event: MouseEvent];
}>();

const toneClass = `desktop-window__status--${props.window.statusTone}`;
const resizeHandleDirections = ['n', 'e', 's', 'w', 'ne', 'se', 'sw', 'nw'] as const;
const minWindowWidth = 340;
const minWindowHeight = 260;

function clampWindowPosition(x: number, y: number, width = props.window.width, height = props.window.height) {
  const viewportWidth = globalThis.window.innerWidth;
  const viewportHeight = globalThis.window.innerHeight;
  const side = viewportWidth <= 980 ? 14 : 16;
  const top = viewportWidth <= 980 ? 84 : 78;
  const bottom = viewportWidth <= 980 ? 104 : 118;
  const maxWidth = Math.max(280, viewportWidth - side * 2);
  const maxHeight = Math.max(240, viewportHeight - top - bottom);
  const nextWidth = Math.min(Math.max(minWindowWidth, width), maxWidth);
  const nextHeight = Math.min(Math.max(minWindowHeight, height), maxHeight);
  const maxX = Math.max(side, viewportWidth - side - nextWidth);
  const maxY = Math.max(top, viewportHeight - bottom - nextHeight);

  return {
    x: Math.round(Math.min(Math.max(side, x), maxX)),
    y: Math.round(Math.min(Math.max(top, y), maxY)),
    width: Math.round(nextWidth),
    height: Math.round(nextHeight),
  };
}

function startWindowDrag(event: PointerEvent) {
  if (props.maximized || event.button !== 0) return;
  const target = event.target as HTMLElement;
  if (target.closest('button')) return;

  emit('focus');
  const startX = event.clientX;
  const startY = event.clientY;
  const originX = props.window.x;
  const originY = props.window.y;
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);

  function moveWindow(pointerEvent: PointerEvent) {
    const next = clampWindowPosition(
      originX + pointerEvent.clientX - startX,
      originY + pointerEvent.clientY - startY,
    );
    emit('move-window', { x: next.x, y: next.y });
  }

  function stopWindowDrag() {
    globalThis.window.removeEventListener('pointermove', moveWindow);
    globalThis.window.removeEventListener('pointerup', stopWindowDrag);
    globalThis.window.removeEventListener('pointercancel', stopWindowDrag);
  }

  globalThis.window.addEventListener('pointermove', moveWindow);
  globalThis.window.addEventListener('pointerup', stopWindowDrag, { once: true });
  globalThis.window.addEventListener('pointercancel', stopWindowDrag, { once: true });
}

function startWindowResize(event: PointerEvent, direction: string) {
  if (props.maximized || event.button !== 0) return;
  event.preventDefault();
  event.stopPropagation();
  emit('focus');

  const startX = event.clientX;
  const startY = event.clientY;
  const origin = {
    x: props.window.x,
    y: props.window.y,
    width: props.window.width,
    height: props.window.height,
  };
  (event.currentTarget as HTMLElement).setPointerCapture?.(event.pointerId);

  function resizeWindow(pointerEvent: PointerEvent) {
    const deltaX = pointerEvent.clientX - startX;
    const deltaY = pointerEvent.clientY - startY;
    let nextX = origin.x;
    let nextY = origin.y;
    let nextWidth = origin.width;
    let nextHeight = origin.height;

    if (direction.includes('e')) {
      nextWidth = Math.max(minWindowWidth, origin.width + deltaX);
    }
    if (direction.includes('s')) {
      nextHeight = Math.max(minWindowHeight, origin.height + deltaY);
    }
    if (direction.includes('w')) {
      nextWidth = Math.max(minWindowWidth, origin.width - deltaX);
      nextX = origin.x + origin.width - nextWidth;
    }
    if (direction.includes('n')) {
      nextHeight = Math.max(minWindowHeight, origin.height - deltaY);
      nextY = origin.y + origin.height - nextHeight;
    }

    const clamped = clampWindowPosition(nextX, nextY, nextWidth, nextHeight);
    emit('resize-window', clamped);
  }

  function stopWindowResize() {
    globalThis.window.removeEventListener('pointermove', resizeWindow);
    globalThis.window.removeEventListener('pointerup', stopWindowResize);
    globalThis.window.removeEventListener('pointercancel', stopWindowResize);
  }

  globalThis.window.addEventListener('pointermove', resizeWindow);
  globalThis.window.addEventListener('pointerup', stopWindowResize, { once: true });
  globalThis.window.addEventListener('pointercancel', stopWindowResize, { once: true });
}
</script>

<template>
  <article
    class="desktop-window"
    :class="{ 'desktop-window--active': active, 'desktop-window--maximized': maximized }"
    :style="{
      left: `${window.x}px`,
      top: `${window.y}px`,
      width: maximized ? undefined : `${window.width}px`,
      height: maximized ? undefined : `${window.height}px`,
      zIndex: active ? window.z + 20 : window.z,
    }"
    role="dialog"
    :aria-label="window.title"
    @pointerdown="emit('focus')"
    @contextmenu.stop.prevent="emit('contextmenu-window', $event)"
  >
    <header
      class="desktop-window__titlebar"
      @pointerdown="startWindowDrag"
      @dblclick="emit('toggle-maximize')"
    >
      <div class="desktop-window__traffic">
        <button
          class="desktop-window__dot desktop-window__dot--red"
          type="button"
          :aria-label="`关闭${window.title}`"
          @click.stop="emit('close')"
        >
          <X :size="11" stroke-width="3" />
        </button>
        <button
          class="desktop-window__dot desktop-window__dot--yellow"
          type="button"
          aria-label="最小化"
          @click.stop="emit('minimize')"
        />
        <button
          class="desktop-window__dot desktop-window__dot--green"
          type="button"
          :aria-label="maximized ? '还原窗口' : '最大化'"
          @click.stop="emit('toggle-maximize')"
        />
      </div>

      <div class="desktop-window__heading">
        <h2>{{ window.title }}</h2>
        <p>{{ window.subtitle }}</p>
      </div>

      <span class="desktop-window__status" :class="toneClass">{{ window.status }}</span>
    </header>

    <section class="desktop-window__body">
      <slot />
    </section>

    <span
      v-for="direction in resizeHandleDirections"
      :key="direction"
      class="desktop-window__resize-handle"
      :class="`desktop-window__resize-handle--${direction}`"
      aria-hidden="true"
      @pointerdown="startWindowResize($event, direction)"
    />
  </article>
</template>

<style scoped>
.desktop-window {
  position: fixed;
  display: flex;
  flex-direction: column;
  min-width: min(340px, calc(100vw - 28px));
  min-height: 280px;
  overflow: hidden;
  color: var(--text);
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.86), rgba(247, 252, 255, 0.68)),
    var(--surface-glass);
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(28px) saturate(1.28);
  -webkit-backdrop-filter: blur(28px) saturate(1.28);
  transform: translateZ(0);
  container-name: desktop-window;
  container-type: inline-size;
  transition:
    border-color 160ms ease,
    box-shadow 160ms ease,
    opacity 160ms ease,
    transform 160ms ease;
}

.desktop-window--active {
  border-color: rgba(19, 136, 255, 0.36);
  box-shadow: var(--shadow-lg);
}

.desktop-window--maximized {
  inset: 84px 24px calc(var(--dock-height) + 28px) 24px !important;
  width: auto !important;
  height: auto !important;
}

.desktop-window--maximized .desktop-window__resize-handle {
  display: none;
}

.desktop-window:not(.desktop-window--active) {
  opacity: 0.93;
}

.desktop-window__titlebar {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 12px 16px 10px;
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.7), rgba(239, 248, 255, 0.42));
  border-bottom: 1px solid rgba(111, 151, 182, 0.18);
  user-select: none;
  cursor: grab;
  touch-action: none;
}

.desktop-window__titlebar:active {
  cursor: grabbing;
}

.desktop-window__traffic {
  display: flex;
  align-items: center;
  gap: 2px;
}

.desktop-window__dot {
  position: relative;
  display: inline-flex;
  flex: 0 0 30px;
  width: 30px;
  height: 30px;
  align-items: center;
  justify-content: center;
  padding: 0;
  color: rgba(101, 32, 32, 0);
  background: transparent;
  border: 0;
  border-radius: 999px;
}

.desktop-window__dot::before {
  position: absolute;
  width: 13px;
  height: 13px;
  content: "";
  border-radius: 999px;
  box-shadow: inset 0 0 0 1px rgba(24, 35, 54, 0.12);
}

.desktop-window__dot svg {
  position: relative;
  z-index: 1;
}

.desktop-window__dot--red::before {
  background: #ff5f57;
}

.desktop-window__dot--yellow::before {
  background: #ffbd2e;
}

.desktop-window__dot--green::before {
  background: #28c840;
}

.desktop-window__dot--red:hover {
  color: rgba(101, 32, 32, 0.78);
}

.desktop-window__heading {
  min-width: 0;
}

.desktop-window__heading h2 {
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 14px;
  font-weight: 760;
  line-height: 1.18;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desktop-window__heading p {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desktop-window__status {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 700;
  white-space: nowrap;
  background: rgba(19, 136, 255, 0.12);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 999px;
}

.desktop-window__status--green {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.12);
  border-color: rgba(34, 181, 115, 0.22);
}

.desktop-window__status--orange {
  color: #b36a00;
  background: rgba(245, 158, 11, 0.14);
  border-color: rgba(245, 158, 11, 0.25);
}

.desktop-window__status--red {
  color: var(--accent-red);
  background: rgba(239, 68, 68, 0.12);
  border-color: rgba(239, 68, 68, 0.22);
}

.desktop-window__body {
  min-height: 0;
  flex: 1;
  overflow: auto;
  overflow-x: hidden;
  overscroll-behavior: contain;
  scrollbar-gutter: stable;
  padding: 14px;
}

.desktop-window__resize-handle {
  position: absolute;
  z-index: 3;
  background: transparent;
}

.desktop-window__resize-handle--n,
.desktop-window__resize-handle--s {
  right: 18px;
  left: 18px;
  height: 10px;
  cursor: ns-resize;
}

.desktop-window__resize-handle--n {
  top: -4px;
}

.desktop-window__resize-handle--s {
  bottom: -4px;
}

.desktop-window__resize-handle--e,
.desktop-window__resize-handle--w {
  top: 18px;
  bottom: 18px;
  width: 10px;
  cursor: ew-resize;
}

.desktop-window__resize-handle--e {
  right: -4px;
}

.desktop-window__resize-handle--w {
  left: -4px;
}

.desktop-window__resize-handle--ne,
.desktop-window__resize-handle--se,
.desktop-window__resize-handle--sw,
.desktop-window__resize-handle--nw {
  width: 18px;
  height: 18px;
}

.desktop-window__resize-handle--ne {
  top: -5px;
  right: -5px;
  cursor: nesw-resize;
}

.desktop-window__resize-handle--se {
  right: -5px;
  bottom: -5px;
  cursor: nwse-resize;
}

.desktop-window__resize-handle--sw {
  bottom: -5px;
  left: -5px;
  cursor: nesw-resize;
}

.desktop-window__resize-handle--nw {
  top: -5px;
  left: -5px;
  cursor: nwse-resize;
}

@media (max-width: 900px) {
  .desktop-window {
    left: 14px !important;
    top: 104px !important;
    width: calc(100vw - 28px) !important;
    height: calc(100vh - var(--dock-height) - 132px) !important;
    max-height: calc(100vh - var(--dock-height) - 132px);
    min-width: 0;
  }

  .desktop-window__resize-handle {
    display: none;
  }

  .desktop-window__titlebar {
    grid-template-columns: 82px minmax(0, 1fr);
    min-height: 56px;
    gap: 8px;
    padding: 10px 12px;
  }

  .desktop-window__status {
    display: none;
  }
}

@container desktop-window (max-width: 440px) {
  .desktop-window__titlebar {
    grid-template-columns: 82px minmax(0, 1fr);
    gap: 8px;
    padding-inline: 12px;
  }

  .desktop-window__status {
    display: none;
  }

  .desktop-window__traffic {
    gap: 0;
  }

  .desktop-window__dot {
    flex-basis: 27px;
    width: 27px;
  }
}

@media (max-height: 840px) and (min-width: 901px) {
  .desktop-window {
    height: calc(100vh - var(--dock-height) - 188px) !important;
    max-height: calc(100vh - var(--dock-height) - 188px);
  }
}
</style>
