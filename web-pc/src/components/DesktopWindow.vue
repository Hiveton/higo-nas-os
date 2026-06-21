<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, nextTick, ref, watch } from 'vue';
import { Maximize2, Menu, Minus, Plus, X } from 'lucide-vue-next';
import type { DesktopWindowConfig } from '../api/types';

// Left nav/sidebar columns that should collapse into an off-canvas drawer when
// the window gets narrow (macOS-style), instead of stacking full-width on top of
// the content. Right-side detail/inspector panels (__details/__inspector/__side)
// intentionally stack below content and are left alone.
const SIDEBAR_SELECTOR = [
  '.ui-window-page__nav',
  '.file-manager__sidebar',
  '.app-center__catalog',
  '.music-app__sidebar',
  '.photo-media__sidebar',
  '.system-settings__sidebar',
  '.download-center__control',
  '.docker-sidebar',
  '.backup-sync__jobs',
  '.video-sidebar',
  '.ai-window__sessions',
  '.agent-window__presets',
  '.protocols__list',
  '.uc__rail',
  '.sf__list',
  '.sync__list',
  '.device-monitor__metrics',
].join(',');
const DRAWER_BREAKPOINT = 760;

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

// --- responsive off-canvas sidebar drawer (narrow windows) ---
const bodyRef = ref<HTMLElement | null>(null);
const bodyWidth = ref(Number.POSITIVE_INFINITY);
const hasSidebar = ref(false);
const drawerOpen = ref(false);
let bodyResizeObserver: ResizeObserver | undefined;

const drawerActive = computed(() => hasSidebar.value && bodyWidth.value <= DRAWER_BREAKPOINT);

function detectSidebar() {
  hasSidebar.value = !!bodyRef.value?.querySelector(SIDEBAR_SELECTOR);
}

function toggleDrawer() {
  drawerOpen.value = !drawerOpen.value;
}

// Picking any nav item inside the drawer closes it (macOS-like).
function onBodyClick(event: MouseEvent) {
  if (!drawerActive.value || !drawerOpen.value) return;
  const target = event.target as HTMLElement;
  const sidebar = bodyRef.value?.querySelector(SIDEBAR_SELECTOR);
  if (sidebar && sidebar.contains(target) && target.closest('button, a, [role="button"], [role="tab"]')) {
    drawerOpen.value = false;
  }
}

onMounted(() => {
  const body = bodyRef.value;
  if (!body) return;
  // Seed width immediately so a window opened already-narrow starts in the
  // right state instead of waiting for the first resize event.
  bodyWidth.value = body.clientWidth;
  detectSidebar();
  if (typeof ResizeObserver !== 'undefined') {
    bodyResizeObserver = new ResizeObserver((entries) => {
      bodyWidth.value = entries[0].contentRect.width;
      if (!hasSidebar.value) detectSidebar();
    });
    bodyResizeObserver.observe(body);
  }
  // Slotted window content and the post-mount geometry clamp can settle a few
  // frames after onMounted, and the ResizeObserver's first callback doesn't
  // always cover both. Re-measure + re-detect across a handful of frames until
  // the sidebar is found, so the drawer engages even on first open.
  let tries = 0;
  const raf = globalThis.requestAnimationFrame?.bind(globalThis);
  const recheck = () => {
    if (!bodyRef.value) return;
    bodyWidth.value = bodyRef.value.clientWidth;
    if (!hasSidebar.value) detectSidebar();
    if (!hasSidebar.value && tries++ < 6 && raf) raf(recheck);
  };
  if (raf) raf(recheck);
  else nextTick(detectSidebar);
});
onBeforeUnmount(() => bodyResizeObserver?.disconnect());

// Close the drawer whenever we leave the compact range.
watch(drawerActive, (active) => {
  if (!active) drawerOpen.value = false;
});

function clampWindowPosition(x: number, y: number, width = props.window.width, height = props.window.height) {
  const viewportWidth = globalThis.window.innerWidth;
  const viewportHeight = globalThis.window.innerHeight;
  const side = viewportWidth <= 980 ? 14 : 16;
  const top = viewportWidth <= 980 ? 84 : 78;
  const bottom = 12;
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
        >
          <Minus :size="11" stroke-width="3" />
        </button>
        <button
          class="desktop-window__dot desktop-window__dot--green"
          type="button"
          :aria-label="maximized ? '还原窗口' : '最大化'"
          @click.stop="emit('toggle-maximize')"
        >
          <component :is="maximized ? Minus : Maximize2" :size="9" stroke-width="3" />
        </button>
      </div>

      <button
        v-if="drawerActive"
        class="desktop-window__menu-btn"
        type="button"
        :aria-label="`${window.title} 菜单`"
        :aria-expanded="drawerOpen"
        @click.stop="toggleDrawer"
        @pointerdown.stop
      >
        <Menu :size="16" stroke-width="2.2" />
      </button>

      <div class="desktop-window__heading">
        <h2>{{ window.title }}</h2>
        <p>{{ window.subtitle }}</p>
      </div>

      <span class="desktop-window__status" :class="toneClass">{{ window.status }}</span>
    </header>

    <section
      ref="bodyRef"
      class="desktop-window__body"
      :class="{
        'desktop-window__body--drawer': drawerActive,
        'desktop-window__body--drawer-open': drawerOpen,
      }"
      @click="onBodyClick"
    >
      <slot />
      <div
        v-if="drawerActive && drawerOpen"
        class="desktop-window__drawer-backdrop"
        aria-hidden="true"
        @click="drawerOpen = false"
      />
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
    linear-gradient(180deg, rgba(var(--surface-rgb), 0.86), rgba(var(--surface-rgb), 0.68)),
    var(--surface-glass);
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: var(--radius-window);
  box-shadow: var(--shadow-md);
  backdrop-filter: blur(28px) saturate(1.28);
  -webkit-backdrop-filter: blur(28px) saturate(1.28);
  transform: translateZ(0);
  container-name: desktop-window;
  container-type: inline-size;
  transition:
    border-color var(--duration-fast) var(--ease-standard),
    box-shadow var(--duration-md) var(--ease-standard),
    opacity var(--duration-fast) var(--ease-standard);
}

.desktop-window--active {
  border-color: rgba(19, 136, 255, 0.36);
  box-shadow: var(--shadow-window);
}

.desktop-window--maximized {
  inset: calc(var(--topbar-height) + 30px) 12px calc(var(--dock-height) + 28px) 12px !important;
  width: auto !important;
  height: auto !important;
}

.desktop-window--maximized .desktop-window__resize-handle {
  display: none;
}

.desktop-window:not(.desktop-window--active) {
  opacity: 0.97;
}

.desktop-window__titlebar {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr) auto;
  align-items: center;
  gap: 12px;
  min-height: 58px;
  padding: 12px 16px 10px;
  background: linear-gradient(180deg, rgba(var(--surface-rgb), 0.7), rgba(var(--surface-rgb), 0.42));
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
  background: transparent;
  border: 0;
  border-radius: var(--radius-pill);
}

.desktop-window__dot::before {
  position: absolute;
  width: 13px;
  height: 13px;
  content: "";
  border-radius: var(--radius-pill);
  /* macOS-style 3D sheen: top highlight + inner ring for depth */
  box-shadow:
    inset 0 1px 0.5px rgba(255, 255, 255, 0.6),
    inset 0 0 0 0.5px rgba(24, 35, 54, 0.18);
  transition: filter var(--duration-fast) var(--ease-standard);
}

.desktop-window__dot svg {
  position: relative;
  z-index: 1;
  opacity: 0;
  transition: opacity var(--duration-fast) var(--ease-standard);
}

/* Reveal the action glyphs when hovering anywhere on the title bar (macOS). */
.desktop-window__titlebar:hover .desktop-window__dot svg,
.desktop-window__dot:hover svg {
  opacity: 1;
}

.desktop-window__dot--red {
  color: rgba(76, 0, 3, 0.7);
}
.desktop-window__dot--red::before {
  background: radial-gradient(circle at 50% 32%, var(--win-close), var(--win-close-2) 72%);
}

.desktop-window__dot--yellow {
  color: rgba(89, 49, 0, 0.7);
}
.desktop-window__dot--yellow::before {
  background: radial-gradient(circle at 50% 32%, var(--win-min), var(--win-min-2) 72%);
}

.desktop-window__dot--green {
  color: rgba(0, 64, 13, 0.7);
}
.desktop-window__dot--green::before {
  background: radial-gradient(circle at 50% 32%, var(--win-max), var(--win-max-2) 72%);
}

.desktop-window__dot:active::before {
  filter: brightness(0.92);
}

/* Dim the lights when the window is not focused, like macOS. */
.desktop-window:not(.desktop-window--active) .desktop-window__dot::before {
  background: rgba(150, 162, 178, 0.5);
  box-shadow: inset 0 0 0 0.5px rgba(24, 35, 54, 0.16);
}

.desktop-window__menu-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
  width: 30px;
  height: 30px;
  color: var(--text);
  background: var(--surface-glass);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-out);
}
.desktop-window__menu-btn:hover {
  background: var(--accent-soft);
  color: var(--accent);
}

.desktop-window__heading {
  min-width: 0;
}

.desktop-window__heading h2 {
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-weight: 760;
  line-height: 1.18;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desktop-window__heading p {
  margin: 4px 0 0;
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.2;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.desktop-window__status {
  display: inline-flex;
  align-items: center;
  min-height: 24px;
  padding: 0 10px;
  /* Theme-aware ink so the small pill text stays legible on its own soft tint
     in both light and dark mode (see --ink-* in tokens.css / base.css). */
  color: var(--ink-blue);
  font-size: var(--fs-2xs);
  font-weight: 700;
  white-space: nowrap;
  background: rgba(19, 136, 255, 0.12);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: var(--radius-pill);
}

.desktop-window__status--green {
  color: var(--ink-green);
  background: rgba(34, 181, 115, 0.12);
  border-color: rgba(34, 181, 115, 0.22);
}

.desktop-window__status--orange {
  color: var(--ink-orange);
  background: rgba(245, 158, 11, 0.14);
  border-color: rgba(245, 158, 11, 0.25);
}

.desktop-window__status--red {
  color: var(--ink-red);
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
</style>
