<script setup lang="ts">
import { computed } from 'vue';
import { ArrowLeft, Captions, Gauge, Play, Search, Wand2 } from 'lucide-vue-next';
import { UiButton, UiEmptyState, UiSelect } from '../../ui';
import type { VideoItem, VideoLibrary, VideoMediaTrack } from '../../../api/types';

const props = defineProps<{
  libraries: VideoLibrary[];
  filteredItems: VideoItem[];
  detailItem: VideoItem | undefined;
  posterUrl: (item: VideoItem) => string;
  displayTitle: (item: VideoItem) => string;
  cardSubtitle: (item: VideoItem) => string;
  episodeLabel: (item: VideoItem) => string;
  displaySubtitle: (item: VideoItem) => string;
  chips: (item: VideoItem) => Array<string | number | undefined>;
  detailOverview: (item: VideoItem) => string;
  allTracks: (tracks?: VideoMediaTrack[], fallback?: string) => string[];
  releaseLabel: (item: VideoItem) => string;
  durationLabel: (seconds?: number) => string;
  listLabel: (values?: string[], fallback?: string) => string;
  scrapedLabel: (item: VideoItem) => string;
}>();

const libraryFilter = defineModel<string>('libraryFilter', { required: true });
const kindFilter = defineModel<string>('kindFilter', { required: true });
const transcodeProfile = defineModel<string>('transcodeProfile', { required: true });

const libraryOptions = computed(() => [
  { label: '全部媒体库', value: '' },
  ...props.libraries.map((library) => ({ label: library.name, value: library.id })),
]);
const kindOptions = [
  { label: '全部分类', value: '' },
  { label: '电影', value: 'movie' },
  { label: '电视剧', value: 'episode' },
];
const transcodeOptions = [
  { label: '1080p H.264', value: '1080p H.264' },
  { label: '720p H.264', value: '720p H.264' },
  { label: '原画质封装', value: '原画质封装' },
];

const emit = defineEmits<{
  (e: 'close-detail'): void;
  (e: 'open-detail', item: VideoItem): void;
  (e: 'play', item: VideoItem): void;
  (e: 'scrape', item: VideoItem): void;
  (e: 'subtitle', item: VideoItem): void;
  (e: 'transcode', item: VideoItem): void;
}>();
</script>

<template>
  <section class="movies-view">
    <div class="filters">
      <UiSelect v-model="libraryFilter" :options="libraryOptions" />
      <UiSelect v-model="kindFilter" :options="kindOptions" />
      <UiSelect v-model="transcodeProfile" :options="transcodeOptions" />
    </div>

    <section v-if="detailItem" class="detail-view">
      <UiButton class="back-button" variant="ghost" tone="neutral" size="sm" :icon-left="ArrowLeft" @click="emit('close-detail')">返回海报墙</UiButton>
      <div class="detail-view-hero">
        <img class="detail-poster large" :src="posterUrl(detailItem)" :alt="detailItem.title" />
        <div class="detail-view-copy">
          <span v-if="episodeLabel(detailItem)" class="episode-badge">{{ episodeLabel(detailItem) }}</span>
          <h3>{{ displayTitle(detailItem) }}</h3>
          <em v-if="detailItem.kind === 'episode' && detailItem.episodeTitle">{{ detailItem.episodeTitle }}</em>
          <p class="detail-subtitle">{{ displaySubtitle(detailItem) }}</p>
          <p class="source-line">{{ detailItem.metadataSource || '本地文件' }} · {{ detailItem.status }}</p>
          <div class="detail-chip-row">
            <i v-for="chip in chips(detailItem)" :key="chip">{{ chip }}</i>
          </div>
          <div class="detail-actions">
            <UiButton :icon-left="Play" @click="emit('play', detailItem)">播放</UiButton>
            <UiButton variant="ghost" tone="neutral" :icon-left="Wand2" @click="emit('scrape', detailItem)">刮削</UiButton>
            <UiButton variant="ghost" tone="neutral" :icon-left="Captions" @click="emit('subtitle', detailItem)">字幕</UiButton>
            <UiButton variant="ghost" tone="neutral" :icon-left="Gauge" @click="emit('transcode', detailItem)">转码</UiButton>
          </div>
        </div>
      </div>

      <section class="detail-section overview-section">
        <h4>{{ detailItem.kind === 'episode' ? '本集简介' : '简介' }}</h4>
        <p>{{ detailOverview(detailItem) }}</p>
      </section>

      <div class="detail-info-grid">
        <section class="detail-section stream-section">
          <h4>媒体流</h4>
          <div class="stream-row" v-for="track in allTracks(detailItem.videoTracks, detailItem.resolution || '未检测')" :key="`video-${track}`">
            <span>视频</span>
            <strong>{{ track }}</strong>
          </div>
          <div class="stream-row" v-for="track in allTracks(detailItem.audioTracks)" :key="`audio-${track}`">
            <span>音频</span>
            <strong>{{ track }}</strong>
          </div>
          <div class="stream-row" v-for="track in allTracks(detailItem.subtitleTracks, detailItem.subtitleUrl ? '外置字幕' : '未检测')" :key="`subtitle-${track}`">
            <span>字幕</span>
            <strong>{{ track }}</strong>
          </div>
        </section>

        <section v-if="detailItem.kind === 'episode'" class="detail-section episode-section">
          <h4>剧集信息</h4>
          <dl>
            <div><dt>剧名</dt><dd>{{ detailItem.seriesTitle || detailItem.libraryName }}</dd></div>
            <div><dt>集名</dt><dd>{{ detailItem.episodeTitle || detailItem.title }}</dd></div>
            <div><dt>季</dt><dd>{{ detailItem.season || '暂无' }}</dd></div>
            <div><dt>集</dt><dd>{{ detailItem.episode || '暂无' }}</dd></div>
          </dl>
        </section>

        <section class="detail-section">
          <h4>详细信息</h4>
          <dl class="detail-meta-grid wide">
            <div><dt>类型</dt><dd>{{ detailItem.kind === 'episode' ? '电视剧' : '电影' }}</dd></div>
            <div><dt>上映</dt><dd>{{ releaseLabel(detailItem) }}</dd></div>
            <div><dt>时长</dt><dd>{{ durationLabel(detailItem.durationSeconds) }}</dd></div>
            <div><dt>大小</dt><dd>{{ detailItem.size }}</dd></div>
            <div><dt>分类</dt><dd>{{ listLabel(detailItem.genres) }}</dd></div>
            <div><dt>国家/地区</dt><dd>{{ listLabel(detailItem.countries) }}</dd></div>
            <div><dt>导演</dt><dd>{{ listLabel(detailItem.directors) }}</dd></div>
            <div><dt>编剧</dt><dd>{{ listLabel(detailItem.writers) }}</dd></div>
            <div><dt>主演</dt><dd>{{ listLabel(detailItem.actors) }}</dd></div>
            <div><dt>工作室</dt><dd>{{ listLabel(detailItem.studios) }}</dd></div>
            <div><dt>来源</dt><dd>{{ detailItem.metadataSource || '本地文件' }}</dd></div>
            <div><dt>刮削时间</dt><dd>{{ scrapedLabel(detailItem) }}</dd></div>
            <div><dt>Provider ID</dt><dd>{{ detailItem.providerId || '暂无' }}</dd></div>
            <div><dt>媒体库</dt><dd>{{ detailItem.libraryName }}</dd></div>
            <div class="full"><dt>文件名</dt><dd>{{ detailItem.fileName }}</dd></div>
            <div class="full tag-field">
              <dt>标签</dt>
              <dd>
                <span v-for="tag in (detailItem.tags?.length ? detailItem.tags : detailItem.genres)" :key="tag"># {{ tag }}</span>
                <span v-if="!(detailItem.tags?.length || detailItem.genres?.length)">暂无标签</span>
              </dd>
            </div>
          </dl>
        </section>
      </div>
    </section>

    <div v-else class="poster-row media-poster-grid">
      <article v-for="item in filteredItems" :key="item.id" class="poster-card media-poster-card" @click="emit('open-detail', item)">
        <img :src="posterUrl(item)" :alt="item.title" />
        <span>{{ item.rating || item.metadataSource || '本地' }}</span>
        <button class="poster-play" type="button" @click.stop="emit('play', item)" aria-label="播放">
          <Play :size="22" />
        </button>
        <strong>{{ displayTitle(item) }}</strong>
        <small>{{ cardSubtitle(item) }}</small>
      </article>
      <UiEmptyState v-if="!filteredItems.length" :icon="Search" title="没有匹配的影片" />
    </div>
  </section>
</template>
