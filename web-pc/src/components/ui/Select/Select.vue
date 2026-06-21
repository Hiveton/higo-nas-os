<script setup lang="ts">
import { computed, nextTick, ref } from 'vue';
import { Check, ChevronDown, Search, X } from 'lucide-vue-next';
import type { UiSize } from '../tokens';
import { useDismiss } from '../composables/useDismiss';

export type SelectOption = {
  label: string;
  value: string | number;
  disabled?: boolean;
};

const props = withDefaults(
  defineProps<{
    options: SelectOption[];
    size?: UiSize;
    placeholder?: string;
    disabled?: boolean;
    invalid?: boolean;
    id?: string;
    /** Render a searchable combobox instead of the native select. */
    searchable?: boolean;
    /** Allow clearing back to the empty value. */
    clearable?: boolean;
  }>(),
  {
    size: 'md',
    placeholder: undefined,
    disabled: false,
    invalid: false,
    id: undefined,
    searchable: false,
    clearable: false,
  },
);

const model = defineModel<string | number>({ default: '' });
const emit = defineEmits<{ change: [value: string | number] }>();

// --- native path -----------------------------------------------------------
function onNativeChange(event: Event) {
  const value = (event.target as HTMLSelectElement).value;
  model.value = value;
  emit('change', value);
}

// --- searchable combobox path ----------------------------------------------
const open = ref(false);
const query = ref('');
const root = ref<HTMLElement | null>(null);
const searchRef = ref<HTMLInputElement | null>(null);
const activeIndex = ref(0);

const selectedLabel = computed(
  () => props.options.find((o) => o.value === model.value)?.label ?? '',
);
const filtered = computed(() => {
  const q = query.value.trim().toLowerCase();
  if (!q) return props.options;
  return props.options.filter((o) => o.label.toLowerCase().includes(q));
});

useDismiss({ ref: root, active: open, onDismiss: () => (open.value = false) });

function toggleOpen() {
  if (props.disabled) return;
  open.value = !open.value;
  if (open.value) {
    query.value = '';
    activeIndex.value = Math.max(
      0,
      filtered.value.findIndex((o) => o.value === model.value),
    );
    void nextTick(() => searchRef.value?.focus());
  }
}

function choose(opt: SelectOption) {
  if (opt.disabled) return;
  model.value = opt.value;
  emit('change', opt.value);
  open.value = false;
}

function clear(event: Event) {
  event.stopPropagation();
  model.value = '';
  emit('change', '');
}

function onKeydown(event: KeyboardEvent) {
  if (!open.value && (event.key === 'Enter' || event.key === 'ArrowDown')) {
    toggleOpen();
    return;
  }
  if (!open.value) return;
  if (event.key === 'ArrowDown') {
    event.preventDefault();
    activeIndex.value = Math.min(filtered.value.length - 1, activeIndex.value + 1);
  } else if (event.key === 'ArrowUp') {
    event.preventDefault();
    activeIndex.value = Math.max(0, activeIndex.value - 1);
  } else if (event.key === 'Enter') {
    event.preventDefault();
    const opt = filtered.value[activeIndex.value];
    if (opt) choose(opt);
  }
}
</script>

<template>
  <!-- Native select (default, fully backward compatible) -->
  <div
    v-if="!searchable"
    class="ui-select"
    :class="[`ui-select--${size}`, { 'ui-select--invalid': invalid, 'ui-select--disabled': disabled }]"
  >
    <select
      :id="id"
      class="ui-select__field"
      :value="model"
      :disabled="disabled"
      :aria-invalid="invalid || undefined"
      @change="onNativeChange"
    >
      <option v-if="placeholder" value="" disabled>{{ placeholder }}</option>
      <option v-for="opt in options" :key="opt.value" :value="opt.value" :disabled="opt.disabled">
        {{ opt.label }}
      </option>
    </select>
    <button
      v-if="clearable && model !== ''"
      type="button"
      class="ui-select__clear"
      aria-label="清除"
      @click="clear"
    >
      <X :size="13" :stroke-width="2.4" />
    </button>
    <ChevronDown class="ui-select__chevron" :size="15" :stroke-width="2.2" />
  </div>

  <!-- Searchable combobox -->
  <div
    v-else
    ref="root"
    class="ui-select ui-select--combo"
    :class="[`ui-select--${size}`, { 'ui-select--invalid': invalid, 'ui-select--disabled': disabled, 'ui-select--open': open }]"
  >
    <button
      :id="id"
      type="button"
      class="ui-select__field ui-select__trigger"
      :disabled="disabled"
      :aria-expanded="open"
      aria-haspopup="listbox"
      @click="toggleOpen"
      @keydown="onKeydown"
    >
      <span :class="{ 'ui-select__placeholder': !selectedLabel }">
        {{ selectedLabel || placeholder || '请选择' }}
      </span>
    </button>
    <button
      v-if="clearable && model !== ''"
      type="button"
      class="ui-select__clear"
      aria-label="清除"
      @click="clear"
    >
      <X :size="13" :stroke-width="2.4" />
    </button>
    <ChevronDown class="ui-select__chevron" :size="15" :stroke-width="2.2" />

    <div v-if="open" class="ui-select__panel u-glass" role="listbox">
      <div class="ui-select__search">
        <Search :size="14" :stroke-width="2" />
        <input
          ref="searchRef"
          v-model="query"
          type="text"
          placeholder="搜索…"
          @keydown="onKeydown"
        />
      </div>
      <ul class="ui-select__list">
        <li
          v-for="(opt, i) in filtered"
          :key="opt.value"
          class="ui-select__option"
          :class="{
            'ui-select__option--active': i === activeIndex,
            'ui-select__option--selected': opt.value === model,
            'ui-select__option--disabled': opt.disabled,
          }"
          role="option"
          :aria-selected="opt.value === model"
          @mouseenter="activeIndex = i"
          @click="choose(opt)"
        >
          <span>{{ opt.label }}</span>
          <Check v-if="opt.value === model" :size="14" :stroke-width="2.4" />
        </li>
        <li v-if="filtered.length === 0" class="ui-select__empty">无匹配项</li>
      </ul>
    </div>
  </div>
</template>

<style scoped>
.ui-select {
  position: relative;
  display: inline-flex;
  align-items: center;
  width: 100%;
  color: var(--text);
  background: var(--control-bg);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  transition: border-color var(--duration-fast) var(--ease-out), box-shadow var(--duration-fast) var(--ease-out);
}

.ui-select--sm {
  min-height: 28px;
}
.ui-select--md {
  min-height: 34px;
}
.ui-select--lg {
  min-height: 42px;
}

.ui-select:focus-within,
.ui-select--open {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.ui-select--invalid {
  border-color: var(--accent-red);
}
.ui-select--disabled {
  opacity: 0.55;
  cursor: not-allowed;
}

.ui-select__field {
  flex: 1;
  min-width: 0;
  height: 100%;
  padding: 0 var(--space-7, 28px) 0 var(--space-3);
  color: inherit;
  font-family: var(--font-ui);
  font-size: var(--fs-sm);
  background: transparent;
  border: 0;
  outline: 0;
  cursor: pointer;
  appearance: none;
  -webkit-appearance: none;
}
.ui-select__trigger {
  display: inline-flex;
  align-items: center;
  text-align: left;
}
.ui-select__placeholder {
  color: var(--text-soft);
}

.ui-select__chevron {
  position: absolute;
  right: var(--space-2);
  color: var(--text-soft);
  pointer-events: none;
}
.ui-select__clear {
  position: absolute;
  right: var(--space-6);
  display: inline-flex;
  padding: 2px;
  color: var(--text-soft);
  background: transparent;
  border: 0;
  border-radius: var(--radius-pill);
  cursor: pointer;
}
.ui-select__clear:hover {
  color: var(--text);
  background: var(--accent-soft);
}

.ui-select__panel {
  position: absolute;
  top: calc(100% + 4px);
  left: 0;
  z-index: var(--z-dropdown);
  width: 100%;
  max-height: 280px;
  display: flex;
  flex-direction: column;
  padding: var(--space-1);
  border-radius: var(--radius-card);
  box-shadow: var(--elevation-2);
}
.ui-select__search {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: 0 var(--space-2);
  color: var(--text-muted);
  border-bottom: 1px solid var(--border);
}
.ui-select__search input {
  flex: 1;
  min-width: 0;
  height: 32px;
  color: var(--text-strong);
  font-size: var(--fs-sm);
  background: transparent;
  border: 0;
  outline: 0;
}
.ui-select__list {
  margin: 0;
  padding: var(--space-1) 0 0;
  overflow-y: auto;
  list-style: none;
}
.ui-select__option {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-control);
  color: var(--text);
  font-size: var(--fs-sm);
  cursor: pointer;
}
.ui-select__option--active {
  background: var(--accent-soft);
}
.ui-select__option--selected {
  color: var(--accent);
  font-weight: var(--fw-medium);
}
.ui-select__option--disabled {
  opacity: 0.45;
  cursor: not-allowed;
}
.ui-select__empty {
  padding: var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-xs);
  text-align: center;
}
</style>
