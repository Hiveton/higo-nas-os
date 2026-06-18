<script setup lang="ts">
import { Album, Grid2X2, Image, Music, Plus, Search, Video } from 'lucide-vue-next';
import { UiButton, UiEmptyState, UiFormField, UiInput, UiSelect } from '../../ui';
import type { AlbumItem, MediaItem } from '../../../api/types';

const albumTypeOptions = [
  { label: '家庭相册', value: '家庭相册' },
  { label: '共享相册', value: '共享相册' },
  { label: '智能回忆', value: '智能回忆' },
];

type ID = string | number;

defineProps<{
  mediaStats: { label: string; value: number }[];
  selectionMode: boolean;
  selectedCount: number;
  allVisibleSelected: boolean;
  busyAction: string;
  albums: AlbumItem[];
  selectedAlbumId: ID;
  filteredMedia: MediaItem[];
  selectedMediaId: ID;
  selectedIds: Set<ID>;
}>();

const searchText = defineModel<string>('searchText', { required: true });
const createAlbumOpen = defineModel<boolean>('createAlbumOpen', { required: true });
const albumNameDraft = defineModel<string>('albumNameDraft', { required: true });
const albumTypeDraft = defineModel<string>('albumTypeDraft', { required: true });
const albumPrivacyDraft = defineModel<string>('albumPrivacyDraft', { required: true });

const emit = defineEmits<{
  (e: 'toggle-selection-mode'): void;
  (e: 'select-all-visible'): void;
  (e: 'create-album'): void;
  (e: 'select-album', album: AlbumItem): void;
  (e: 'select-media', item: MediaItem): void;
}>();
</script>

<template>
  <main class="photo-media__main">
    <section class="photo-media__toolbar" aria-label="相册媒体工具">
      <label class="photo-media__search">
        <Search :size="15" />
        <input v-model="searchText" type="search" placeholder="搜索标题、人物、地点、设备、格式" />
      </label>
      <div class="photo-media__stats" aria-label="媒体统计">
        <span v-for="stat in mediaStats" :key="stat.label">
          <strong>{{ stat.value }}</strong>
          {{ stat.label }}
        </span>
      </div>
      <button
        class="photo-media__tool"
        :class="{ 'photo-media__tool--active': selectionMode }"
        type="button"
        @click="emit('toggle-selection-mode')"
      >
        <Grid2X2 :size="14" />
        {{ selectionMode ? '取消选择' : '批量选择' }}
      </button>
    </section>

    <section v-if="selectionMode || selectedCount > 0" class="photo-media__selection" aria-label="批量操作">
      <UiButton variant="ghost" tone="neutral" size="sm" @click="emit('select-all-visible')">
        {{ allVisibleSelected ? '取消全选' : '全选当前' }}
      </UiButton>
      <span>已选择 {{ selectedCount }} 项</span>
      <UiButton variant="soft" size="sm" :icon-left="Plus" :disabled="selectedCount === 0" @click="createAlbumOpen = !createAlbumOpen">
        新建相册
      </UiButton>
    </section>

    <section v-if="createAlbumOpen" class="photo-media__create" aria-label="创建相册">
      <UiFormField label="相册名称">
        <UiInput v-model="albumNameDraft" placeholder="例如：端午出游精选" />
      </UiFormField>
      <UiFormField label="类型">
        <UiSelect v-model="albumTypeDraft" :options="albumTypeOptions" />
      </UiFormField>
      <UiFormField label="权限">
        <UiInput v-model="albumPrivacyDraft" placeholder="留空使用默认策略" />
      </UiFormField>
      <UiButton :icon-left="Plus" :loading="busyAction === 'createAlbum'" @click="emit('create-album')">
        {{ busyAction === 'createAlbum' ? '创建中' : '创建' }}
      </UiButton>
    </section>

    <section v-if="albums.length > 0" class="photo-media__albums" aria-label="家庭、共享和智能相册">
      <button
        v-for="album in albums"
        :key="album.id"
        class="photo-media__album"
        :class="{ 'photo-media__album--active': selectedAlbumId === album.id }"
        type="button"
        @click="emit('select-album', album)"
      >
        <span>{{ album.type }}</span>
        <strong>{{ album.name }}</strong>
        <small>{{ album.count }} 项 · {{ album.privacy }}</small>
      </button>
    </section>

    <section
      class="photo-media__grid"
      :class="{ 'photo-media__grid--empty': filteredMedia.length === 0 }"
      aria-label="媒体项目"
    >
      <button
        v-for="item in filteredMedia"
        :key="item.id"
        class="photo-media__tile"
        :class="{
          'photo-media__tile--active': selectedMediaId === item.id,
          'photo-media__tile--selected': selectedIds.has(item.id),
        }"
        type="button"
        @click="emit('select-media', item)"
      >
        <div class="photo-media__thumb" :style="{ background: item.accent }">
          <Image v-if="item.kind === '照片'" :size="24" />
          <Video v-else-if="item.kind === '视频'" :size="24" />
          <Music v-else :size="24" />
          <span v-if="selectionMode" class="photo-media__check" aria-hidden="true">
            {{ selectedIds.has(item.id) ? '✓' : '' }}
          </span>
        </div>
        <strong>{{ item.title }}</strong>
        <span>{{ item.kind }} · {{ item.meta }}</span>
        <small>{{ item.people }} · {{ item.place }}</small>
      </button>
      <UiEmptyState v-if="filteredMedia.length === 0" :icon="Album" title="暂无媒体" compact />
    </section>
  </main>
</template>
