<script setup lang="ts" generic="T extends Record<string, unknown>">
import { computed } from 'vue';
import Spinner from '../Spinner/Spinner.vue';
import EmptyState from '../EmptyState/EmptyState.vue';

export type Column<R> = {
  key: string;
  label: string;
  width?: string;
  align?: 'left' | 'center' | 'right';
  sortable?: boolean;
  /** Optional accessor when the cell value differs from row[key]. */
  accessor?: (row: R) => unknown;
};

export type SortState = { key: string; dir: 'asc' | 'desc' };

const props = withDefaults(
  defineProps<{
    columns: Column<T>[];
    rows: T[];
    rowKey: keyof T | ((row: T) => string | number);
    loading?: boolean;
    empty?: { title: string; description?: string };
    density?: 'comfortable' | 'compact';
  }>(),
  {
    loading: false,
    empty: () => ({ title: '暂无数据' }),
    density: 'comfortable',
  },
);

const sort = defineModel<SortState | null>('sort', { default: null });

const emit = defineEmits<{ 'row-click': [row: T] }>();

function keyOf(row: T): string | number {
  return typeof props.rowKey === 'function' ? props.rowKey(row) : (row[props.rowKey] as string | number);
}

function cellValue(row: T, col: Column<T>): unknown {
  return col.accessor ? col.accessor(row) : row[col.key];
}

function toggleSort(col: Column<T>) {
  if (!col.sortable) return;
  if (sort.value?.key === col.key) {
    sort.value = { key: col.key, dir: sort.value.dir === 'asc' ? 'desc' : 'asc' };
  } else {
    sort.value = { key: col.key, dir: 'asc' };
  }
}

const isEmpty = computed(() => !props.loading && props.rows.length === 0);
</script>

<template>
  <div class="ui-table" :class="`ui-table--${density}`">
    <table class="ui-table__el">
      <thead>
        <tr>
          <th
            v-for="col in columns"
            :key="col.key"
            class="ui-table__th"
            :class="[`ui-table__th--${col.align ?? 'left'}`, { 'ui-table__th--sortable': col.sortable }]"
            :style="col.width ? { width: col.width } : undefined"
            @click="toggleSort(col)"
          >
            <slot :name="`header-${col.key}`" :column="col">
              <span>{{ col.label }}</span>
              <span v-if="col.sortable && sort?.key === col.key" class="ui-table__sort">
                {{ sort.dir === 'asc' ? '↑' : '↓' }}
              </span>
            </slot>
          </th>
        </tr>
      </thead>
      <tbody v-if="!isEmpty && !loading">
        <tr
          v-for="row in rows"
          :key="keyOf(row)"
          class="ui-table__row"
          @click="emit('row-click', row)"
        >
          <td
            v-for="col in columns"
            :key="col.key"
            class="ui-table__td"
            :class="`ui-table__td--${col.align ?? 'left'}`"
          >
            <slot :name="`cell-${col.key}`" :row="row" :value="cellValue(row, col)">
              {{ cellValue(row, col) }}
            </slot>
          </td>
        </tr>
      </tbody>
    </table>

    <div v-if="loading" class="ui-table__state">
      <slot name="loading">
        <Spinner size="lg" />
      </slot>
    </div>
    <div v-else-if="isEmpty" class="ui-table__state">
      <slot name="empty">
        <EmptyState :title="empty.title" :description="empty.description" compact />
      </slot>
    </div>
  </div>
</template>

<style scoped>
.ui-table {
  width: 100%;
  overflow-x: auto;
}

.ui-table__el {
  width: 100%;
  border-collapse: collapse;
  font-size: var(--fs-sm);
}

.ui-table__th {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-weight: var(--fw-semibold);
  text-align: left;
  white-space: nowrap;
  border-bottom: 1px solid var(--border);
}
.ui-table__th--center {
  text-align: center;
}
.ui-table__th--right {
  text-align: right;
}
.ui-table__th--sortable {
  cursor: pointer;
  user-select: none;
}

.ui-table__sort {
  margin-left: var(--space-1);
  color: var(--accent);
}

.ui-table__row {
  transition: background var(--duration-fast) var(--ease-out);
}
.ui-table__row:hover {
  background: var(--accent-soft);
}

.ui-table__td {
  color: var(--text);
  border-bottom: 1px solid var(--border);
  vertical-align: middle;
}
.ui-table__td--center {
  text-align: center;
}
.ui-table__td--right {
  text-align: right;
}

.ui-table--comfortable .ui-table__th,
.ui-table--comfortable .ui-table__td {
  padding: var(--space-3) var(--space-3);
}
.ui-table--compact .ui-table__th,
.ui-table--compact .ui-table__td {
  padding: var(--space-2) var(--space-2);
}

.ui-table__state {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: var(--space-8);
}
</style>
