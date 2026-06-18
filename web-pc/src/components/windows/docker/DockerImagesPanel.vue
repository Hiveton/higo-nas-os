<script setup lang="ts">
import { Boxes, Play, RefreshCw, Trash2 } from 'lucide-vue-next';
import type { DockerImage, DockerImagePullStatus } from '../../../api/types';
import { UiBadge, UiButton, UiEmptyState, UiProgressBar } from '../../ui';

type LocalImage = DockerImage & { ref: string };

defineProps<{
  localImages: LocalImage[];
  visibleImagePulls: DockerImagePullStatus[];
  dockerIconPath: string;
  imageRepositoryPath: (image?: string) => string;
  normalizeImageRef: (image?: string) => string;
  shouldUseExternalImageIcon: (image?: string, iconUrl?: string) => boolean;
  imageIconUrl: (image?: string, iconUrl?: string) => string;
  pullStatusLabel: (status?: string) => string;
  safeProgress: (progress: number) => number;
  imageRef: (image: DockerImage) => string;
}>();

const emit = defineEmits<{
  (e: 'refresh'): void;
  (e: 'icon-failed', image?: string, iconUrl?: string): void;
  (e: 'run-image', ref: string): void;
  (e: 'remove-image', ref: string): void;
}>();
</script>

<template>
  <section class="docker-panel docker-local-images">
    <header class="docker-panel__header">
      <div>
        <h3>本地镜像</h3>
        <p>{{ localImages.length }} 个镜像，{{ visibleImagePulls.length }} 个拉取任务。</p>
      </div>
      <UiButton variant="soft" size="sm" :icon-left="RefreshCw" @click="emit('refresh')">刷新</UiButton>
    </header>
    <div class="docker-list docker-list--images">
      <article v-for="task in visibleImagePulls" :key="`pull-${task.id || task.image}`" class="docker-list-item docker-image-item docker-image-item--pull">
        <span class="docker-image-icon" :title="imageRepositoryPath(task.image) || normalizeImageRef(task.image)">
          <img
            v-if="shouldUseExternalImageIcon(task.image)"
            :src="imageIconUrl(task.image)"
            alt=""
            loading="lazy"
            @error="emit('icon-failed', task.image)"
          />
          <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
            <path :d="dockerIconPath" />
          </svg>
        </span>
        <div class="docker-image-main">
          <div class="docker-image-title-row">
            <strong>{{ normalizeImageRef(task.image) }}</strong>
            <UiBadge :tone="task.status === 'failed' ? 'danger' : 'success'" variant="soft" size="sm">{{ pullStatusLabel(task.status) }}</UiBadge>
          </div>
          <p>{{ task.message || pullStatusLabel(task.status) }} · {{ task.downloaded || '0 B' }} / {{ task.total || '未知大小' }} · {{ task.speed || '0 B/s' }}</p>
          <UiProgressBar :value="safeProgress(task.progress)" :tone="task.status === 'failed' ? 'danger' : 'primary'" size="sm" />
          <small v-if="task.error">{{ task.error }}</small>
        </div>
      </article>
      <article v-for="image in localImages" :key="`${image.id}-${image.repository}-${image.tag}`" class="docker-list-item docker-image-item">
        <span class="docker-image-icon" :title="imageRepositoryPath(image.ref) || image.ref">
          <img
            v-if="shouldUseExternalImageIcon(image.ref, image.iconUrl)"
            :src="imageIconUrl(image.ref, image.iconUrl)"
            alt=""
            loading="lazy"
            @error="emit('icon-failed', image.ref, image.iconUrl)"
          />
          <svg v-else class="docker-glyph docker-glyph--default" viewBox="0 0 1024 1024" aria-hidden="true">
            <path :d="dockerIconPath" />
          </svg>
        </span>
        <div class="docker-image-main">
          <strong>{{ image.ref }}</strong>
          <p>{{ image.id }} · {{ image.size }} · {{ image.created }}</p>
        </div>
        <div class="docker-row-actions">
          <UiButton variant="soft" tone="success" size="sm" :icon-left="Play" @click="emit('run-image', imageRef(image))">运行</UiButton>
          <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="emit('remove-image', imageRef(image))">删除</UiButton>
        </div>
      </article>
      <UiEmptyState v-if="localImages.length === 0 && visibleImagePulls.length === 0" compact :icon="Boxes" title="暂无本地镜像" />
    </div>
  </section>
</template>
