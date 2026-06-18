<script setup lang="ts">
import {
  AudioLines,
  ChevronDown,
  Library,
  MoreHorizontal,
  RefreshCcw,
  Search,
  SlidersHorizontal,
} from 'lucide-vue-next';
import { UiButton, UiEmptyState, UiFormField, UiIconButton, UiTextarea } from '../../ui';
import type { MusicTrack } from '../../../api/types';

defineProps<{
  mobileLibraryTabs: string[];
  isScanning: boolean;
  isLoading: boolean;
  filteredTracks: MusicTrack[];
  selectedTrack?: MusicTrack;
  activeView: 'tracks' | 'albums' | 'library';
  coverUrl: (track?: MusicTrack) => string;
  formatTime: (value: number) => string;
  trackDuration: (track?: MusicTrack | null) => number;
}>();

const libraryPathText = defineModel<string>('libraryPathText', { required: true });

const emit = defineEmits<{
  (e: 'scan'): void;
  (e: 'select-track', track: MusicTrack): void;
  (e: 'save-library'): void;
}>();
</script>

<template>
  <section class="music-library">
    <div class="music-library__mobile-head">
      <h2>音乐库</h2>
      <div>
        <Search :size="18" />
        <MoreHorizontal :size="18" />
      </div>
    </div>
    <div class="music-library__mobile-tabs">
      <button v-for="tab in mobileLibraryTabs" :key="tab" type="button" :class="{ active: tab === '歌曲' }">{{ tab }}</button>
    </div>

    <section class="music-library__tools">
      <UiButton variant="ghost" tone="neutral" size="sm" :icon-right="ChevronDown">全部曲目</UiButton>
      <UiButton variant="ghost" tone="neutral" size="sm" :icon-right="ChevronDown">全部专辑</UiButton>
      <UiButton variant="ghost" tone="neutral" size="sm" :icon-right="ChevronDown">全部艺术家</UiButton>
      <span />
      <UiButton variant="ghost" tone="neutral" size="sm" :icon-right="ChevronDown">排序：添加时间</UiButton>
      <UiButton variant="ghost" tone="neutral" size="sm" :icon-left="RefreshCcw" :disabled="isScanning" @click="emit('scan')">重新扫描</UiButton>
      <UiIconButton :icon="SlidersHorizontal" label="更多设置" size="sm" />
    </section>

    <section class="music-library__table">
      <header>
        <span>#</span><span>标题</span><span>艺术家</span><span>专辑</span><span>格式</span><span>时长</span><span>大小</span>
      </header>
      <button
        v-for="(track, index) in filteredTracks"
        :key="track.id"
        type="button"
        class="music-library__row"
        :class="{ active: track.id === selectedTrack?.id }"
        @click="emit('select-track', track)"
      >
        <span>{{ index + 1 }}</span>
        <span class="music-library__song">
          <img v-if="coverUrl(track)" :src="coverUrl(track)" :alt="track.album" />
          <AudioLines v-else :size="16" />
          <span>
            <strong>{{ track.title }}</strong>
            <em>{{ track.artist }} · {{ track.album }}</em>
          </span>
        </span>
        <span>{{ track.artist }}</span>
        <span>{{ track.album }}</span>
        <span>{{ track.codec }}</span>
        <span>{{ formatTime(trackDuration(track)) }}</span>
        <span>{{ track.size }}</span>
        <small class="music-library__mobile-codec">{{ track.codec }}</small>
      </button>
      <UiEmptyState
        v-if="!filteredTracks.length"
        :icon="Library"
        title="当前没有曲目"
        description="设置媒体库路径后执行扫描。"
      />

    </section>

    <section v-if="activeView === 'library'" class="music-library__settings">
      <UiFormField label="媒体库路径，每行一个目录">
        <UiTextarea v-model="libraryPathText" />
      </UiFormField>
      <UiButton :disabled="isLoading" @click="emit('save-library')">保存媒体库</UiButton>
    </section>
  </section>
</template>
