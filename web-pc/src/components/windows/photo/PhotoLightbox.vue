<script setup lang="ts">
import { Captions, Film, Image, Music, Video, Wand2 } from 'lucide-vue-next';
import { UiButton } from '../../ui';
import type { MediaItem } from '../../../api/types';

defineProps<{
  selectedMedia: MediaItem;
  hasMedia: boolean;
  busyAction: string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'add-subtitle'): void;
  (e: 'add-transcode'): void;
  (e: 'generate-memory'): void;
}>();
</script>

<template>
  <div class="photo-media__lightbox" role="dialog" aria-modal="true" aria-label="媒体详情">
    <div class="photo-media__lightbox-card">
      <button class="photo-media__close" type="button" aria-label="关闭详情" @click="emit('close')">×</button>
      <div class="photo-media__preview" :style="{ background: selectedMedia.accent }">
        <Image v-if="selectedMedia.kind === '照片'" :size="38" />
        <Video v-else-if="selectedMedia.kind === '视频'" :size="38" />
        <Music v-else :size="38" />
      </div>
      <div class="photo-media__lightbox-info">
        <p>{{ selectedMedia.kind }} · {{ selectedMedia.timeline }}</p>
        <h2>{{ selectedMedia.title }}</h2>
        <div class="photo-media__chips">
          <span>{{ selectedMedia.meta }}</span>
          <span>{{ selectedMedia.device }}</span>
          <span>{{ selectedMedia.place }}</span>
        </div>
        <dl class="photo-media__lightbox-meta">
          <div>
            <dt>人物</dt>
            <dd>{{ selectedMedia.people }}</dd>
          </div>
          <div>
            <dt>相册</dt>
            <dd>{{ selectedMedia.album }}</dd>
          </div>
          <div>
            <dt>状态</dt>
            <dd>{{ selectedMedia.status }}</dd>
          </div>
          <div>
            <dt>媒体处理</dt>
            <dd>{{ selectedMedia.hasSubtitle ? '字幕可用' : '待匹配字幕' }} · {{ selectedMedia.transcoded ? '转码任务中' : '原始文件' }}</dd>
          </div>
        </dl>
        <div class="photo-media__actions photo-media__actions--dialog">
          <UiButton variant="soft" size="sm" :icon-left="Captions" :disabled="!hasMedia" :loading="busyAction === 'subtitle'" @click="emit('add-subtitle')">
            字幕任务
          </UiButton>
          <UiButton variant="soft" size="sm" :icon-left="Film" :disabled="!hasMedia" :loading="busyAction === 'transcode'" @click="emit('add-transcode')">
            转码任务
          </UiButton>
          <UiButton variant="soft" size="sm" :icon-left="Wand2" :disabled="!hasMedia" :loading="busyAction === 'memory'" @click="emit('generate-memory')">
            生成回忆
          </UiButton>
        </div>
      </div>
    </div>
  </div>
</template>
