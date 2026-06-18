<script setup lang="ts">
import type { Component } from 'vue';
import { AudioLines, Heart, List, ListMusic, Settings } from 'lucide-vue-next';
import type { MusicLibrarySettings } from '../../../api/types';

type SidebarItem = { key: string; label: string; count: number; icon: Component };

defineProps<{
  sidebarItems: SidebarItem[];
  activeView: 'tracks' | 'albums' | 'library';
  statusMessage: string;
  settings: MusicLibrarySettings;
  formatDate: (value?: string) => string;
}>();

const emit = defineEmits<{
  (e: 'select-item', key: string): void;
}>();
</script>

<template>
  <aside class="music-app__sidebar">
    <div class="music-app__brand">
      <div class="music-app__brand-icon"><AudioLines :size="20" /></div>
      <div>
        <strong>音乐中心</strong>
        <span>家庭 NAS</span>
      </div>
    </div>
    <nav class="music-app__nav">
      <p>媒体库</p>
      <button
        v-for="item in sidebarItems"
        :key="item.key"
        type="button"
        :class="{ 'music-app__nav-item--active': activeView === item.key || (item.key === 'tracks' && activeView === 'tracks') }"
        @click="emit('select-item', item.key)"
      >
        <component :is="item.icon" :size="16" />
        <span>{{ item.label }}</span>
        <small>{{ item.count }}</small>
      </button>
    </nav>
    <nav class="music-app__nav music-app__nav--secondary">
      <p>我的音乐</p>
      <button type="button"><Heart :size="16" /><span>我喜欢的</span></button>
      <button type="button"><List :size="16" /><span>轻音乐</span></button>
      <button type="button"><ListMusic :size="16" /><span>经典怀旧</span></button>
    </nav>
    <div class="music-app__storage">
      <strong>媒体库状态</strong>
      <span>{{ statusMessage }}</span>
      <small>上次扫描：{{ formatDate(settings.lastScanAt) }}</small>
    </div>
    <div class="music-app__side-footer">
      <button type="button"><Settings :size="16" /></button>
    </div>
  </aside>
</template>
