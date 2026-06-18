<script setup lang="ts">
import { ImageDown, Search } from 'lucide-vue-next';
import type { DockerImageSearchResult } from '../../../api/types';
import { UiButton, UiEmptyState, UiInput } from '../../ui';

defineProps<{
  imageQuery: string;
  imageSearchResults: DockerImageSearchResult[];
  dockerIconPath: string;
  imageRepositoryPath: (image?: string) => string;
  shouldUseExternalImageIcon: (image?: string, iconUrl?: string) => boolean;
  imageIconUrl: (image?: string, iconUrl?: string) => string;
  isPullingImage: (image?: string) => boolean;
  hasLocalImage: (image?: string) => boolean;
}>();

const emit = defineEmits<{
  (e: 'update:imageQuery', value: string): void;
  (e: 'search'): void;
  (e: 'icon-failed', image?: string, iconUrl?: string): void;
  (e: 'pull-image', name: string): void;
}>();
</script>

<template>
  <section class="docker-panel docker-images">
    <header class="docker-panel__header">
      <UiInput
        class="docker-search-input"
        :model-value="imageQuery"
        type="search"
        :prefix-icon="Search"
        placeholder="搜索镜像，例如 nginx、redis、homeassistant"
        @update:model-value="(value) => emit('update:imageQuery', String(value))"
        @enter="emit('search')"
      />
      <UiButton variant="solid" tone="primary" size="sm" :icon-left="Search" :disabled="!imageQuery.trim()" @click="emit('search')">搜索</UiButton>
    </header>

    <div class="docker-list docker-list--images">
      <article v-for="result in imageSearchResults" :key="result.name" class="docker-list-item docker-image-item">
        <span class="docker-image-icon" :title="imageRepositoryPath(result.name) || result.name">
          <img
            v-if="shouldUseExternalImageIcon(result.name, result.iconUrl)"
            :src="imageIconUrl(result.name, result.iconUrl)"
            alt=""
            loading="lazy"
            @error="emit('icon-failed', result.name, result.iconUrl)"
          />
          <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
            <path :d="dockerIconPath" />
          </svg>
        </span>
        <div class="docker-image-main">
          <strong>{{ result.name }}</strong>
          <p>{{ result.description || '无描述' }}</p>
          <small>{{ result.stars }} stars <template v-if="result.official"> · 官方</template><template v-if="result.automated"> · 自动构建</template></small>
        </div>
        <UiButton variant="soft" size="sm" :icon-left="ImageDown" :disabled="isPullingImage(result.name) || hasLocalImage(result.name)" @click="emit('pull-image', result.name)">
          {{ isPullingImage(result.name) ? '拉取中' : hasLocalImage(result.name) ? '已下载' : '拉取' }}
        </UiButton>
      </article>
      <UiEmptyState v-if="imageSearchResults.length === 0" compact :icon="ImageDown" title="输入镜像名开始搜索" />
    </div>
  </section>
</template>

<style scoped>
.docker-search-input {
  flex: 1 1 260px;
  min-width: 180px;
}
</style>
