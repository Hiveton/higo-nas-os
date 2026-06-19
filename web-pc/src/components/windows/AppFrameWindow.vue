<script setup lang="ts">
import { ref } from 'vue';
import { ExternalLink, RefreshCw } from 'lucide-vue-next';

const props = defineProps<{ src: string; name?: string }>();

// A token appended to force the iframe to reload on demand without mutating the
// declared src.
const reloadToken = ref(0);
function reload() {
  reloadToken.value += 1;
}
function framedSrc() {
  if (!props.src) return '';
  const join = props.src.includes('?') ? '&' : '?';
  return reloadToken.value ? `${props.src}${join}_higo=${reloadToken.value}` : props.src;
}
</script>

<template>
  <div class="app-frame">
    <header class="app-frame__bar">
      <span class="app-frame__title">{{ name || '应用' }}</span>
      <div class="app-frame__actions">
        <button type="button" class="app-frame__btn" title="刷新" @click="reload">
          <RefreshCw :size="14" />
        </button>
        <a class="app-frame__btn" :href="src" target="_blank" rel="noopener noreferrer" title="在新标签打开">
          <ExternalLink :size="14" />
        </a>
      </div>
    </header>
    <iframe
      v-if="src"
      :key="reloadToken"
      class="app-frame__view"
      :src="framedSrc()"
      referrerpolicy="no-referrer"
      sandbox="allow-scripts allow-forms allow-same-origin allow-popups allow-downloads"
    />
    <div v-else class="app-frame__empty">该应用未声明 Web 入口（webEntry），无法内嵌显示。</div>
  </div>
</template>

<style scoped>
.app-frame {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  height: 100%;
  min-height: 0;
}

.app-frame__bar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 6px 10px;
  border-bottom: 1px solid var(--border);
  background: rgba(var(--surface-rgb), 0.6);
}

.app-frame__title {
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 600;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.app-frame__actions {
  display: flex;
  gap: 6px;
}

.app-frame__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 26px;
  height: 26px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.app-frame__view {
  width: 100%;
  height: 100%;
  border: 0;
  background: #fff;
}

.app-frame__empty {
  display: grid;
  place-items: center;
  padding: 24px;
  color: var(--text-muted);
  font-size: 12px;
}
</style>
