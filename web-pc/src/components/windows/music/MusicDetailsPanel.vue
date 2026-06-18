<script setup lang="ts">
import { Disc3 } from 'lucide-vue-next';
import type { MusicTrack } from '../../../api/types';

defineProps<{
  selectedTrack?: MusicTrack;
  coverUrl: (track?: MusicTrack) => string;
  formatDate: (value?: string) => string;
}>();
</script>

<template>
  <aside class="music-app__details">
    <header>
      <button class="active" type="button">详情</button>
      <button type="button">歌词</button>
      <button type="button">标签</button>
    </header>
    <section>
      <img v-if="coverUrl(selectedTrack)" :src="coverUrl(selectedTrack)" :alt="selectedTrack?.album ?? '专辑封面'" />
      <Disc3 v-else :size="48" />
      <div>
        <h3>{{ selectedTrack?.title ?? '未选择曲目' }}</h3>
        <p>{{ selectedTrack?.artist ?? '未知艺术家' }}</p>
        <span>{{ selectedTrack?.album ?? '未归类专辑' }}</span>
      </div>
    </section>
    <dl>
      <div><dt>格式</dt><dd>{{ selectedTrack?.codec ?? 'AUTO' }}</dd></div>
      <div><dt>采样率</dt><dd>96 kHz</dd></div>
      <div><dt>位深</dt><dd>24 bit</dd></div>
      <div><dt>文件大小</dt><dd>{{ selectedTrack?.size ?? '-' }}</dd></div>
      <div><dt>添加时间</dt><dd>{{ formatDate(selectedTrack?.discoveredAt) }}</dd></div>
      <div><dt>专辑信息</dt><dd>{{ selectedTrack?.album ?? '-' }}</dd></div>
    </dl>
    <article>
      <h4>简介</h4>
      <p>在这张专辑里，音乐以柔和的声线描绘故事。当前界面优先展示播放、歌词和媒体库信息。</p>
    </article>
    <div class="music-app__tags">
      <span># 夜晚</span><span># 梦幻</span><span># 治愈</span><span># HiFi</span>
    </div>
  </aside>
</template>
