<script setup lang="ts">
import { computed } from 'vue';
import { CheckCircle2, CircleDashed, Wrench } from 'lucide-vue-next';
import { nasFeatures, type NasFeatureKey } from '../data/nasFeatures';

const props = defineProps<{
  modules: NasFeatureKey[];
  compact?: boolean;
}>();

const items = computed(() => props.modules.flatMap((module) => nasFeatures[module] ?? []));

function stateText(state: string) {
  if (state === 'ready') return '已接入';
  if (state === 'mock') return '界面就绪';
  return '待后端';
}
</script>

<template>
  <section class="nas-feature-panel" :class="{ 'nas-feature-panel--compact': compact }" aria-label="功能补全入口">
    <article v-for="item in items" :key="item.title" class="nas-feature-panel__card">
      <header>
        <Wrench :size="15" />
        <strong>{{ item.title }}</strong>
      </header>
      <p>{{ item.detail }}</p>
      <div class="nas-feature-panel__actions">
        <button v-for="action in item.actions" :key="action.label" type="button">
          <CheckCircle2 v-if="action.state === 'ready'" :size="13" />
          <CircleDashed v-else :size="13" />
          <span>{{ action.label }}</span>
          <small>{{ stateText(action.state) }}</small>
        </button>
      </div>
    </article>
  </section>
</template>

<style scoped>
.nas-feature-panel {
  --nas-feature-min: 280px;

  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(min(100%, var(--nas-feature-min)), 1fr));
  align-items: start;
  grid-auto-rows: auto;
  gap: 10px;
  min-width: 0;
  min-height: 0;
}

.nas-feature-panel--compact {
  --nas-feature-min: 320px;
}

.nas-feature-panel__card {
  display: grid;
  grid-template-rows: auto minmax(30px, auto) auto;
  align-content: start;
  gap: 8px;
  min-width: 0;
  min-height: 126px;
  overflow: visible;
  padding: 12px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.nas-feature-panel__card header {
  display: flex;
  align-items: center;
  gap: 7px;
  color: var(--accent);
}

.nas-feature-panel__card strong {
  min-width: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nas-feature-panel__card p {
  margin: 0;
  overflow-wrap: anywhere;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.nas-feature-panel__actions {
  display: flex;
  flex-wrap: wrap;
  align-content: start;
  gap: 6px;
  min-width: 0;
  max-width: 100%;
  overflow: visible;
}

.nas-feature-panel__actions button {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  min-height: 27px;
  min-width: 0;
  flex: 0 0 auto;
  max-width: 100%;
  padding: 0 8px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 730;
}

.nas-feature-panel__actions span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.nas-feature-panel__actions small {
  color: var(--text-soft);
  font-size: 10px;
  font-weight: 650;
}
</style>
