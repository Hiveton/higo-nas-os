<script setup lang="ts">
import { Film, FolderOpen, Play, Plus } from 'lucide-vue-next';
import { UiButton, UiEmptyState } from '../../ui';
import type { VideoItem, VideoLibrary, VideoTask } from '../../../api/types';

defineProps<{
  libraries: VideoLibrary[];
  items: VideoItem[];
  runningTasks: VideoTask[];
  typeLabel: (type: string) => string;
  posterUrl: (item: VideoItem) => string;
  displayTitle: (item: VideoItem) => string;
  cardSubtitle: (item: VideoItem) => string;
}>();

const emit = defineEmits<{
  (e: 'open-library', library: VideoLibrary): void;
  (e: 'create-library'): void;
  (e: 'view-all'): void;
  (e: 'open-detail', item: VideoItem): void;
  (e: 'play', item: VideoItem): void;
  (e: 'view-tasks'): void;
}>();
</script>

<template>
  <section class="home-view">
    <div class="library-strip">
      <article v-for="libraryItem in libraries" :key="libraryItem.id" class="library-card" @click="emit('open-library', libraryItem)">
        <FolderOpen :size="20" />
        <div>
          <strong>{{ libraryItem.name }}</strong>
          <span>{{ typeLabel(libraryItem.type) }} · {{ libraryItem.count }} 个文件</span>
        </div>
      </article>
      <article v-if="!libraries.length" class="library-card empty" @click="emit('create-library')">
        <Plus :size="20" />
        <div>
          <strong>创建媒体库</strong>
          <span>添加本机路径后开始扫描。</span>
        </div>
      </article>
    </div>

    <div class="section-head">
      <h3>最近索引</h3>
      <UiButton variant="ghost" tone="neutral" size="sm" @click="emit('view-all')">查看全部</UiButton>
    </div>
    <div class="poster-row media-poster-grid">
      <article v-for="item in items.slice(0, 10)" :key="item.id" class="poster-card media-poster-card" @click="emit('open-detail', item)">
        <img :src="posterUrl(item)" :alt="item.title" />
        <span>{{ item.rating || item.metadataSource || '本地' }}</span>
        <button class="poster-play" type="button" @click.stop="emit('play', item)" aria-label="播放">
          <Play :size="22" />
        </button>
        <strong>{{ displayTitle(item) }}</strong>
        <small>{{ cardSubtitle(item) }}</small>
      </article>
      <UiEmptyState v-if="!items.length" :icon="Film" title="还没有媒体文件" description="请先创建媒体库并扫描。" />
    </div>

    <div class="task-panel">
      <div>
        <strong>进行中的任务</strong>
        <span>{{ runningTasks.length ? `${runningTasks.length} 个任务` : '当前没有运行任务' }}</span>
      </div>
      <UiButton variant="ghost" tone="neutral" size="sm" @click="emit('view-tasks')">任务列表</UiButton>
    </div>
  </section>
</template>
