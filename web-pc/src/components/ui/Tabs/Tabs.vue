<script setup lang="ts">
import {
  computed,
  nextTick,
  onBeforeUnmount,
  onMounted,
  provide,
  ref,
  watch,
  type Component,
  type Ref,
} from 'vue';
import { ChevronDown, MoreHorizontal } from 'lucide-vue-next';
import type { UiSize } from '../tokens';
import { TABS_ACTIVE_KEY } from './context';
import UiDropdown from '../Menu/Dropdown.vue';
import UiMenuItem from '../Menu/MenuItem.vue';

export type TabItem = {
  key: string;
  label: string;
  icon?: Component;
  badge?: string | number;
  disabled?: boolean;
};

const props = withDefaults(
  defineProps<{
    tabs: TabItem[];
    variant?: 'pill' | 'underline' | 'segmented';
    size?: UiSize;
    /**
     * How to handle a strip too narrow for all tabs:
     * - 'scroll' (default): single-row horizontal scroller.
     * - 'menu': tabs that don't fit collapse into a 「更多 ▾」 dropdown.
     */
    overflow?: 'scroll' | 'menu';
  }>(),
  { variant: 'underline', size: 'md', overflow: 'scroll' },
);

const model = defineModel<string>({ required: true });

const emit = defineEmits<{ 'update:modelValue': [value: string] }>();

const activeKey = computed(() => model.value) as Ref<string>;
provide(TABS_ACTIVE_KEY, activeKey);

const glyphSize: Record<UiSize, number> = { sm: 14, md: 15, lg: 16 };

function select(tab: TabItem) {
  if (tab.disabled || tab.key === model.value) return;
  model.value = tab.key;
  emit('update:modelValue', tab.key);
}

/* --- overflow="menu" measurement ---
   Widths are read from a hidden measurement row that always renders ALL tabs,
   so collapsing the visible strip never changes what we measure (avoids the
   classic ResizeObserver feedback loop). */
const stripRef = ref<HTMLElement | null>(null);
const measureRef = ref<HTMLElement | null>(null);
const visibleCount = ref(props.tabs.length);
let resizeObserver: ResizeObserver | undefined;

const visibleTabs = computed(() =>
  props.overflow === 'menu' ? props.tabs.slice(0, visibleCount.value) : props.tabs,
);
const overflowTabs = computed(() =>
  props.overflow === 'menu' ? props.tabs.slice(visibleCount.value) : [],
);
const activeInOverflow = computed(() => overflowTabs.value.some((t) => t.key === model.value));

function measure() {
  if (props.overflow !== 'menu') return;
  const strip = stripRef.value;
  const row = measureRef.value;
  if (!strip || !row) return;
  const buttons = [...row.children] as HTMLElement[];
  if (!buttons.length) return;
  const available = strip.clientWidth;
  const gap = 4;
  const totalWidth = buttons.reduce((sum, b, i) => sum + b.offsetWidth + (i ? gap : 0), 0);
  if (totalWidth <= available) {
    visibleCount.value = props.tabs.length;
    return;
  }
  const moreReserve = 76; // width budget for the 「更多」 trigger
  let used = 0;
  let count = 0;
  for (let i = 0; i < buttons.length; i++) {
    const w = buttons[i].offsetWidth + (i ? gap : 0);
    if (used + w + moreReserve <= available) {
      used += w;
      count++;
    } else break;
  }
  visibleCount.value = Math.max(1, count);
}

onMounted(() => {
  if (props.overflow !== 'menu') return;
  nextTick(measure);
  if (typeof ResizeObserver !== 'undefined' && stripRef.value) {
    resizeObserver = new ResizeObserver(() => measure());
    resizeObserver.observe(stripRef.value);
  }
});
onBeforeUnmount(() => resizeObserver?.disconnect());
watch(() => props.tabs.length, () => nextTick(measure));
</script>

<template>
  <div class="ui-tabs" :class="[`ui-tabs--${variant}`, `ui-tabs--${size}`]">
    <div ref="stripRef" class="ui-tabs__strip" role="tablist">
      <button
        v-for="tab in visibleTabs"
        :key="tab.key"
        class="ui-tabs__tab"
        :class="{ 'ui-tabs__tab--active': tab.key === model, 'ui-tabs__tab--disabled': tab.disabled }"
        type="button"
        role="tab"
        :aria-selected="tab.key === model"
        :disabled="tab.disabled"
        @click="select(tab)"
      >
        <component :is="tab.icon" v-if="tab.icon" :size="glyphSize[size]" :stroke-width="2.1" />
        <span>{{ tab.label }}</span>
        <span v-if="tab.badge !== undefined" class="ui-tabs__badge">{{ tab.badge }}</span>
      </button>

      <UiDropdown
        v-if="overflowTabs.length"
        placement="bottom-end"
        class="ui-tabs__more"
      >
        <template #trigger>
          <button
            class="ui-tabs__tab ui-tabs__tab--more"
            :class="{ 'ui-tabs__tab--active': activeInOverflow }"
            type="button"
            aria-label="更多标签"
          >
            <MoreHorizontal :size="glyphSize[size]" :stroke-width="2.1" />
            <span>更多</span>
            <ChevronDown :size="13" :stroke-width="2.2" />
          </button>
        </template>
        <UiMenuItem
          v-for="tab in overflowTabs"
          :key="tab.key"
          :tone="tab.key === model ? 'primary' : 'neutral'"
          :icon="tab.icon"
          :disabled="tab.disabled"
          @select="select(tab)"
        >
          {{ tab.label }}
        </UiMenuItem>
      </UiDropdown>
    </div>
    <!-- hidden measurement row: always holds every tab at natural width -->
    <div v-if="overflow === 'menu'" ref="measureRef" class="ui-tabs__measure" aria-hidden="true">
      <span v-for="tab in tabs" :key="tab.key" class="ui-tabs__tab">
        <component :is="tab.icon" v-if="tab.icon" :size="glyphSize[size]" :stroke-width="2.1" />
        <span>{{ tab.label }}</span>
        <span v-if="tab.badge !== undefined" class="ui-tabs__badge">{{ tab.badge }}</span>
      </span>
    </div>

    <div v-if="$slots.default" class="ui-tabs__panels">
      <slot />
    </div>
  </div>
</template>

<style scoped>
.ui-tabs {
  position: relative;
  display: flex;
  flex-direction: column;
  min-height: 0;
}

.ui-tabs__more {
  flex: 0 0 auto;
}
.ui-tabs__tab--more {
  cursor: pointer;
}

/* off-screen measurement row — never visible, just provides natural widths */
.ui-tabs__measure {
  position: absolute;
  top: 0;
  left: 0;
  display: flex;
  gap: var(--space-1);
  visibility: hidden;
  pointer-events: none;
  height: 0;
  overflow: hidden;
  white-space: nowrap;
}

.ui-tabs__strip {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  /* Never wrap into multiple rows — overflow scrolls horizontally so the
     strip always occupies a single, compact row even in narrow windows. */
  flex-wrap: nowrap;
  overflow-x: auto;
  overflow-y: hidden;
  scroll-snap-type: x proximity;
  scrollbar-width: none;
  -webkit-overflow-scrolling: touch;
}
.ui-tabs__strip::-webkit-scrollbar {
  display: none;
}

.ui-tabs__tab {
  flex: 0 0 auto;
  scroll-snap-align: start;
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-muted);
  font-family: var(--font-ui);
  font-weight: var(--fw-medium);
  white-space: nowrap;
  background: transparent;
  border: 0;
  cursor: pointer;
  transition: color var(--duration-fast) var(--ease-out), background var(--duration-fast) var(--ease-out);
}
.ui-tabs--sm .ui-tabs__tab {
  min-height: 28px;
  padding: 0 var(--space-3);
  font-size: var(--fs-xs);
}
.ui-tabs--md .ui-tabs__tab {
  min-height: 34px;
  padding: 0 var(--space-3);
  font-size: var(--fs-sm);
}
.ui-tabs--lg .ui-tabs__tab {
  min-height: 40px;
  padding: 0 var(--space-4);
  font-size: var(--fs-md);
}

.ui-tabs__tab--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}

.ui-tabs__badge {
  min-width: 18px;
  padding: 0 5px;
  font-size: var(--fs-2xs);
  font-weight: var(--fw-semibold);
  text-align: center;
  color: var(--accent);
  background: var(--accent-soft);
  border-radius: 999px;
}

/* underline */
.ui-tabs--underline .ui-tabs__strip {
  border-bottom: 1px solid var(--border);
}
.ui-tabs--underline .ui-tabs__tab {
  border-bottom: 2px solid transparent;
  margin-bottom: -1px;
}
.ui-tabs--underline .ui-tabs__tab--active {
  color: var(--accent);
  border-bottom-color: var(--accent);
}

/* pill */
.ui-tabs--pill .ui-tabs__tab {
  border-radius: 999px;
}
.ui-tabs--pill .ui-tabs__tab--active {
  color: var(--accent);
  background: var(--accent-soft);
}

/* segmented */
.ui-tabs--segmented .ui-tabs__strip {
  padding: var(--space-1);
  gap: var(--space-1);
  background: var(--surface-glass);
  border-radius: var(--radius-md);
}
.ui-tabs--segmented .ui-tabs__tab {
  border-radius: var(--radius-sm);
}
.ui-tabs--segmented .ui-tabs__tab--active {
  color: var(--text-strong);
  background: var(--surface-solid);
  box-shadow: var(--shadow-sm);
}

.ui-tabs__panels {
  flex: 1;
  min-height: 0;
}
</style>
