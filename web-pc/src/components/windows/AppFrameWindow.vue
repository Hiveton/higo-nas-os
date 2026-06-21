<script setup lang="ts">
import { ref } from 'vue';
import { ExternalLink, RefreshCw } from 'lucide-vue-next';
import { UiWindowPage, UiIconButton } from '../ui';

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
  <UiWindowPage layout="stack" :title="name || '应用'">
    <template #actions>
      <UiIconButton :icon="RefreshCw" variant="ghost" size="sm" label="刷新" @click="reload" />
      <a class="app-frame__btn" :href="src" target="_blank" rel="noopener noreferrer" title="在新标签打开">
        <ExternalLink :size="14" />
      </a>
    </template>

    <iframe
      v-if="src"
      :key="reloadToken"
      class="app-frame__view"
      :src="framedSrc()"
      referrerpolicy="no-referrer"
      sandbox="allow-scripts allow-forms allow-same-origin allow-popups allow-downloads"
    />
    <div v-else class="app-frame__empty">该应用未声明 Web 入口（webEntry），无法内嵌显示。</div>
  </UiWindowPage>
</template>

<style scoped>
.app-frame__btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 28px;
  height: 28px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.app-frame__view {
  flex: 1 1 auto;
  width: 100%;
  min-height: 0;
  border: 0;
  border-radius: var(--radius-card);
  background: var(--surface-solid);
}

.app-frame__empty {
  display: grid;
  place-items: center;
  padding: 24px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
</style>
