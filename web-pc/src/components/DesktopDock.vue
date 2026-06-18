<script setup lang="ts">
import type { DockApp } from '../data/higoos';

const props = defineProps<{
  apps: DockApp[];
  activeId?: string;
  runningIds?: string[];
  pinnedIds?: string[];
  position?: 'bottom' | 'left' | 'right';
  dockStyle?: 'floating' | 'side' | 'compact';
  iconSize?: 'small' | 'default' | 'large';
}>();

const emit = defineEmits<{
  'open-app': [id: string];
  'contextmenu-app': [payload: { id: string; x: number; y: number }];
}>();

function openApp(id: string) {
  emit('open-app', id);
}

function openContextMenu(event: MouseEvent, id: string) {
  event.preventDefault();
  event.stopPropagation();
  emit('contextmenu-app', { id, x: event.clientX, y: event.clientY });
}

function isRunning(id: string) {
  return props.runningIds?.includes(id) ?? false;
}

function isPinned(id: string) {
  return props.pinnedIds?.includes(id) ?? false;
}
</script>

<template>
  <nav
    class="dock"
    :class="[
      `dock--${props.position ?? 'bottom'}`,
      `dock--style-${props.dockStyle ?? 'floating'}`,
      `dock--size-${props.iconSize ?? 'default'}`,
    ]"
    aria-label="HiGoOS 应用 Dock"
  >
    <div class="dock__scroller">
      <div class="dock__surface">
        <template v-for="(app, index) in props.apps" :key="app.id">
          <span
            v-if="app.utility && !props.apps[index - 1]?.utility"
            class="dock__separator"
            aria-hidden="true"
          />

          <button
            class="dock__item"
            :class="{
              'dock__item--active': app.id === props.activeId,
              'dock__item--running': isRunning(app.id),
              'dock__item--pinned': isPinned(app.id),
              'dock__item--temporary': isRunning(app.id) && !isPinned(app.id),
            }"
            type="button"
            :aria-label="app.name"
            :aria-current="app.id === props.activeId ? 'page' : undefined"
            @click="openApp(app.id)"
            @contextmenu="openContextMenu($event, app.id)"
          >
            <span class="dock__tooltip" role="tooltip">{{ app.name }}</span>
            <span class="dock__icon-wrap">
              <img class="dock__icon" :src="app.icon" :alt="app.name" draggable="false" />
              <span v-if="app.badge" class="dock__badge">{{ app.badge }}</span>
            </span>
            <span class="dock__active-indicator" aria-hidden="true" />
            <span v-if="isRunning(app.id) && !isPinned(app.id)" class="dock__temporary-pill">临时</span>
          </button>
        </template>
      </div>
    </div>
  </nav>
</template>

<style scoped>
.dock {
  position: absolute;
  top: auto;
  right: auto;
  left: 50%;
  bottom: max(16px, env(safe-area-inset-bottom));
  z-index: 75;
  display: flex;
  width: auto;
  min-height: 0;
  padding: 0;
  align-items: center;
  justify-content: center;
  max-width: calc(100vw - 28px);
  transform: translateX(-50%);
  overflow: visible;
  pointer-events: none;
}

.dock__scroller {
  max-width: 100%;
  padding-top: 34px;
  margin-top: -34px;
  overflow: visible;
  pointer-events: none;
}

.dock__surface {
  position: relative;
  z-index: 1;
  display: flex;
  align-items: flex-end;
  justify-content: center;
  gap: clamp(4px, 0.58vw, 8px);
  width: max-content;
  max-width: 100%;
  min-height: clamp(78px, 8.3vw, 96px);
  padding: clamp(9px, 0.9vw, 12px) clamp(12px, 1.25vw, 18px) clamp(8px, 0.8vw, 11px);
  overflow: visible;
  background:
    linear-gradient(180deg, rgba(255, 255, 255, 0.76), rgba(223, 250, 249, 0.48)),
    rgba(246, 253, 255, 0.58);
  border: 1px solid rgba(255, 255, 255, 0.78);
  border-radius: clamp(22px, 3.2vw, 34px);
  box-shadow:
    0 26px 58px rgba(18, 55, 86, 0.24),
    0 11px 24px rgba(17, 107, 128, 0.12),
    inset 0 1px 0 rgba(255, 255, 255, 0.88),
    inset 0 -1px 0 rgba(68, 128, 144, 0.12);
  backdrop-filter: blur(24px) saturate(1.34);
  pointer-events: auto;
}

.dock__surface::before {
  position: absolute;
  inset: 5px 8px auto;
  height: 38%;
  content: "";
  background: linear-gradient(180deg, rgba(255, 255, 255, 0.74), rgba(255, 255, 255, 0));
  border-radius: inherit;
  pointer-events: none;
}

.dock__surface::after {
  position: absolute;
  right: 7%;
  bottom: 6px;
  left: 7%;
  height: 12px;
  content: "";
  background: radial-gradient(ellipse at center, rgba(34, 106, 123, 0.2), transparent 68%);
  filter: blur(6px);
  pointer-events: none;
}

.dock__scroller::-webkit-scrollbar {
  display: none;
}

.dock--left,
.dock--right {
  top: calc(var(--topbar-height) + 56px);
  bottom: max(32px, env(safe-area-inset-bottom));
  left: max(14px, env(safe-area-inset-left));
  width: var(--dock-side-width, 96px);
  height: auto;
  max-height: none;
  max-width: none;
  transform: none;
  align-items: center;
  justify-content: center;
}

.dock--right {
  right: max(14px, env(safe-area-inset-right));
  left: auto;
}

.dock--left .dock__scroller,
.dock--right .dock__scroller {
  width: 100%;
  max-width: var(--dock-side-width, 96px);
  height: auto;
  max-height: 100%;
  padding-top: 0;
  margin-top: 0;
  overflow-x: visible;
  overflow-y: auto;
  pointer-events: auto;
  scrollbar-width: none;
}

.dock--left .dock__surface,
.dock--right .dock__surface {
  flex-direction: column;
  align-items: center;
  justify-content: flex-start;
  width: 100%;
  min-width: 0;
  min-height: 0;
  max-height: none;
  max-width: none;
  overflow: visible;
  padding: 14px 8px;
  gap: 8px;
  border-radius: 30px;
  box-shadow:
    inset 0 1px 0 rgba(255, 255, 255, 0.82),
    inset 0 -1px 0 rgba(255, 255, 255, 0.34);
}

.dock--style-side.dock--left .dock__surface,
.dock--style-side.dock--right .dock__surface {
  background:
    linear-gradient(135deg, rgba(255, 255, 255, 0.72), rgba(214, 244, 255, 0.36)),
    rgba(255, 255, 255, 0.36);
  border-color: rgba(255, 255, 255, 0.58);
  border-radius: 30px;
}

.dock--style-compact .dock__surface {
  min-height: clamp(64px, 7vw, 78px);
  padding: 8px 10px;
  border-radius: 20px;
}

.dock--bottom.dock--size-small .dock__surface {
  min-height: 70px;
  padding: 8px 12px 7px;
}

.dock--bottom.dock--size-large .dock__surface {
  min-height: 112px;
  padding: 13px 20px 12px;
}

.dock--left.dock--size-small .dock__surface,
.dock--right.dock--size-small .dock__surface {
  padding: 10px 7px;
  gap: 6px;
  border-radius: 22px;
}

.dock--left.dock--size-large .dock__surface,
.dock--right.dock--size-large .dock__surface {
  padding: 16px 12px;
  gap: 10px;
  border-radius: 32px;
}

.dock--left .dock__surface::after,
.dock--right .dock__surface::after {
  display: none;
}

.dock--left .dock__separator,
.dock--right .dock__separator {
  width: 46px;
  height: 1px;
  margin: 5px 0;
  background: linear-gradient(90deg, transparent, rgba(71, 98, 125, 0.3), transparent);
}

.dock--left .dock__item,
.dock--right .dock__item {
  width: 62px;
  height: 68px;
  place-items: center;
}

.dock--size-small .dock__item {
  width: 50px;
  height: 58px;
}

.dock--size-small .dock__icon-wrap {
  width: 46px;
  height: 46px;
}

.dock--size-large .dock__item {
  width: 72px;
  height: 82px;
}

.dock--size-large .dock__icon-wrap {
  width: 68px;
  height: 68px;
}

.dock--left .dock__item:hover .dock__icon-wrap,
.dock--left .dock__item:focus-visible .dock__icon-wrap {
  transform: translateX(8px) scale(1.08);
}

.dock--right .dock__item:hover .dock__icon-wrap,
.dock--right .dock__item:focus-visible .dock__icon-wrap {
  transform: translateX(-8px) scale(1.08);
}

.dock--left .dock__active-indicator,
.dock--right .dock__active-indicator {
  position: absolute;
  top: 50%;
  width: 4px;
  height: 4px;
  margin-top: 0;
  transform: translateY(-50%);
}

.dock--left .dock__active-indicator {
  left: 2px;
}

.dock--right .dock__active-indicator {
  right: 2px;
}

.dock--left .dock__item--running .dock__active-indicator,
.dock--right .dock__item--running .dock__active-indicator,
.dock--left .dock__item--active .dock__active-indicator,
.dock--right .dock__item--active .dock__active-indicator {
  width: 4px;
  height: 18px;
}

.dock--left .dock__tooltip {
  top: 50%;
  bottom: auto;
  left: calc(100% + 8px);
  transform: translate(-4px, -50%);
}

.dock--right .dock__tooltip {
  top: 50%;
  right: calc(100% + 8px);
  bottom: auto;
  left: auto;
  transform: translate(4px, -50%);
}

.dock--left .dock__item:hover .dock__tooltip,
.dock--left .dock__item:focus-visible .dock__tooltip,
.dock--right .dock__item:hover .dock__tooltip,
.dock--right .dock__item:focus-visible .dock__tooltip {
  transform: translate(0, -50%);
}

.dock__separator {
  flex: 0 0 1px;
  width: 1px;
  height: clamp(42px, 5.2vw, 58px);
  margin: 0 clamp(4px, 0.55vw, 8px) 10px;
  background: linear-gradient(
    180deg,
    transparent,
    rgba(71, 98, 125, 0.28) 18%,
    rgba(71, 98, 125, 0.3) 82%,
    transparent
  );
}

.dock__item {
  position: relative;
  display: grid;
  flex: 0 0 auto;
  width: clamp(50px, 4.55vw, 66px);
  height: clamp(60px, 5.65vw, 78px);
  place-items: end center;
  padding: 0;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
}

.dock__item--temporary {
  opacity: 0.94;
}

.dock__icon-wrap {
  position: relative;
  display: grid;
  width: clamp(48px, 4.3vw, 62px);
  height: clamp(48px, 4.3vw, 62px);
  place-items: center;
  transform-origin: 50% 100%;
  transition:
    transform 180ms var(--ease-out),
    filter 180ms ease;
}

.dock__icon {
  width: 100%;
  height: 100%;
  object-fit: contain;
  filter: drop-shadow(0 12px 14px rgba(21, 52, 78, 0.2));
  transform: scale(1.16);
  transform-origin: 50% 60%;
  user-select: none;
}

.dock__item:hover .dock__icon-wrap,
.dock__item:focus-visible .dock__icon-wrap {
  filter: brightness(1.04) saturate(1.06);
  transform: translateY(-10px) scale(1.1);
}

.dock__item:focus-visible .dock__icon-wrap {
  border-radius: 16px;
  outline: 2px solid rgba(19, 136, 255, 0.42);
  outline-offset: 5px;
}

.dock__badge {
  position: absolute;
  top: -3px;
  right: -3px;
  min-width: 19px;
  height: 19px;
  padding: 0 5px;
  color: white;
  font-size: 11px;
  font-weight: 800;
  line-height: 19px;
  text-align: center;
  background: linear-gradient(135deg, #ff4d63, #ef4444);
  border-radius: 999px;
  box-shadow:
    0 0 0 2px rgba(255, 255, 255, 0.9),
    0 8px 14px rgba(239, 68, 68, 0.25);
}

.dock__active-indicator {
  width: 4px;
  height: 4px;
  margin-top: 6px;
  background: transparent;
  border-radius: 999px;
  transition:
    width 180ms ease,
    background 180ms ease,
    box-shadow 180ms ease;
}

.dock__item--running .dock__active-indicator {
  width: 8px;
  background: rgba(16, 32, 51, 0.5);
}

.dock__item--active .dock__active-indicator {
  width: 18px;
  background: rgba(16, 32, 51, 0.76);
  box-shadow: 0 3px 8px rgba(16, 32, 51, 0.2);
}

.dock__temporary-pill {
  position: absolute;
  right: 4px;
  bottom: 14px;
  z-index: 2;
  width: 8px;
  height: 8px;
  overflow: hidden;
  padding: 0;
  color: transparent;
  text-indent: 999px;
  background: rgba(20, 184, 166, 0.9);
  border: 1px solid rgba(255, 255, 255, 0.88);
  border-radius: 999px;
  box-shadow: 0 4px 10px rgba(20, 184, 166, 0.25);
}

.dock__tooltip {
  position: absolute;
  bottom: 70px;
  left: 50%;
  z-index: 3;
  max-width: 108px;
  padding: 6px 9px;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 700;
  line-height: 1.1;
  text-align: center;
  white-space: nowrap;
  background: rgba(255, 255, 255, 0.88);
  border: 1px solid rgba(100, 136, 166, 0.2);
  border-radius: 9px;
  box-shadow: 0 10px 24px rgba(24, 64, 99, 0.18);
  opacity: 0;
  transform: translate(-50%, 8px);
  transition:
    opacity 150ms ease,
    transform 150ms ease;
  pointer-events: none;
}

.dock__item:hover .dock__tooltip,
.dock__item:focus-visible .dock__tooltip {
  opacity: 1;
  transform: translate(-50%, 0);
}

@media (hover: hover) {
  .dock--bottom .dock__item:hover + .dock__item .dock__icon-wrap,
  .dock--bottom .dock__item:has(+ .dock__item:hover) .dock__icon-wrap {
    transform: translateY(-5px) scale(1.04);
  }

  .dock--bottom .dock__item:hover + .dock__item + .dock__item .dock__icon-wrap,
  .dock--bottom .dock__item:has(+ .dock__item + .dock__item:hover) .dock__icon-wrap {
    transform: translateY(-2px) scale(1.02);
  }
}

@media (max-width: 820px) {
  .dock {
    top: auto;
    right: 10px;
    bottom: max(10px, env(safe-area-inset-bottom));
    left: 10px;
    width: auto;
    max-width: none;
    transform: none;
  }

  .dock__scroller {
    width: 100%;
    overflow-x: auto;
    overflow-y: hidden;
    background:
      linear-gradient(180deg, rgba(255, 255, 255, 0.76), rgba(223, 250, 249, 0.48)),
      rgba(246, 253, 255, 0.58);
    border: 1px solid rgba(255, 255, 255, 0.78);
    border-radius: 26px;
    box-shadow:
      0 22px 46px rgba(18, 55, 86, 0.22),
      0 10px 22px rgba(17, 107, 128, 0.12),
      inset 0 1px 0 rgba(255, 255, 255, 0.88),
      inset 0 -1px 0 rgba(68, 128, 144, 0.12);
    backdrop-filter: blur(24px) saturate(1.34);
    -webkit-backdrop-filter: blur(24px) saturate(1.34);
    pointer-events: auto;
    scrollbar-width: none;
  }

  .dock__surface {
    flex-direction: row;
    justify-content: flex-start;
    width: max-content;
    min-height: 78px;
    max-width: none;
    padding: 10px 12px 8px;
    overflow: visible;
    background: transparent;
    border: 0;
    border-radius: 26px;
    box-shadow: none;
    backdrop-filter: none;
    -webkit-backdrop-filter: none;
  }

  .dock__surface::before,
  .dock__surface::after {
    display: none;
  }

  .dock__item {
    width: 52px;
    height: 62px;
    place-items: end center;
  }

  .dock__icon-wrap {
    width: 48px;
    height: 48px;
  }

  .dock__tooltip {
    display: none;
  }

  .dock--left .dock__active-indicator,
  .dock--right .dock__active-indicator {
    position: static;
    width: 4px;
    height: 4px;
    margin-top: 6px;
    transform: none;
  }

  .dock--left .dock__item--running .dock__active-indicator,
  .dock--right .dock__item--running .dock__active-indicator {
    width: 8px;
    height: 4px;
  }

  .dock--left .dock__item--active .dock__active-indicator,
  .dock--right .dock__item--active .dock__active-indicator {
    width: 18px;
    height: 4px;
  }
}
</style>
