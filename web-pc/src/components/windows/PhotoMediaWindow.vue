<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Album,
  Camera,
  Captions,
  CheckCircle2,
  Film,
  Image,
  MapPin,
  Music,
  Share2,
  ShieldAlert,
  Sparkles,
  Users,
  Video,
  Wand2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AlbumItem, MediaItem } from '../../api/types';

type DimensionKey = 'timeline' | 'people' | 'places' | 'devices' | 'albums';
type ID = string | number;

const dimensionOptions: Array<{ key: DimensionKey; label: string; icon: typeof Album }> = [
  { key: 'timeline', label: '时间线', icon: Film },
  { key: 'people', label: '人物', icon: Users },
  { key: 'places', label: '地点', icon: MapPin },
  { key: 'devices', label: '设备', icon: Camera },
  { key: 'albums', label: '相册', icon: Album },
];

const mediaItems = ref<MediaItem[]>([
  {
    id: 1,
    title: '春节团圆 4K 合影',
    kind: '照片',
    timeline: '2026 春节',
    people: '爸爸 / 妈妈',
    place: '杭州',
    device: 'iPhone 17 Pro',
    album: '家庭年度相册',
    meta: '48MP · HEIC · 18 张连拍',
    status: '已完成人物识别',
    accent: 'linear-gradient(135deg, #bfe7ff, #fff1bf)',
  },
  {
    id: 2,
    title: '海边旅行 vlog',
    kind: '视频',
    timeline: '2025 暑假',
    people: '小雨 / 爸爸',
    place: '三亚',
    device: 'Sony A7C II',
    album: '旅行视频',
    meta: '42:16 · H.265 · 4K',
    status: '海报墙已刮削',
    accent: 'linear-gradient(135deg, #bdebd6, #c6d7ff)',
    hasSubtitle: true,
  },
  {
    id: 3,
    title: '家庭钢琴练习',
    kind: '音乐',
    timeline: '2026 五月',
    people: '小雨',
    place: '客厅',
    device: 'HiGo 麦克风',
    album: '孩子成长记录',
    meta: '08:24 · FLAC · 96kHz',
    status: '已生成波形索引',
    accent: 'linear-gradient(135deg, #e8d7ff, #c5f6ff)',
  },
  {
    id: 4,
    title: '露营星空延时',
    kind: '视频',
    timeline: '2025 秋游',
    people: '妈妈 / 小雨',
    place: '安吉',
    device: 'DJI Osmo',
    album: '共享露营相册',
    meta: '12:02 · ProRes · 4K',
    status: '等待转码',
    accent: 'linear-gradient(135deg, #d1fae5, #fde68a)',
  },
]);

const albums = ref<AlbumItem[]>([
  { id: 1, name: '家庭年度相册', type: '家庭相册', count: 3862, privacy: '仅家庭成员可见' },
  { id: 2, name: '共享露营相册', type: '共享相册', count: 214, privacy: '链接关闭' },
  { id: 3, name: '旅行视频', type: '共享相册', count: 87, privacy: '亲友可见' },
  { id: 4, name: '孩子成长记录', type: '智能回忆', count: 642, privacy: 'AI 自动维护' },
]);

const activeDimension = ref<DimensionKey>('timeline');
const selectedFacet = ref('2026 春节');
const selectedMediaId = ref<ID>(1);
const selectedAlbumId = ref<ID>(1);
const shareEnabled = ref(false);
const memoryRuns = ref(0);
const mergeNotice = ref('人物识别已就绪，合并前会保留可回滚记录。');
const mediaNotice = ref('媒体刮削已拉取 2 部视频海报，字幕库命中 1 条。');
const transcodeJobs = ref<string[]>([]);
const subtitleJobs = ref<string[]>([]);
const loading = ref(false);
const busyAction = ref('');

const selectedMedia = computed(() => mediaItems.value.find((item) => item.id === selectedMediaId.value) ?? mediaItems.value[0]);
const selectedAlbum = computed(() => albums.value.find((item) => item.id === selectedAlbumId.value) ?? albums.value[0]);

const facets = computed(() => {
  const values = mediaItems.value.map((item) => {
    if (activeDimension.value === 'people') return item.people;
    if (activeDimension.value === 'places') return item.place;
    if (activeDimension.value === 'devices') return item.device;
    if (activeDimension.value === 'albums') return item.album;
    return item.timeline;
  });
  return [...new Set(values)];
});

const filteredMedia = computed(() =>
  mediaItems.value.filter((item) => {
    if (activeDimension.value === 'people') return item.people === selectedFacet.value;
    if (activeDimension.value === 'places') return item.place === selectedFacet.value;
    if (activeDimension.value === 'devices') return item.device === selectedFacet.value;
    if (activeDimension.value === 'albums') return item.album === selectedFacet.value;
    return item.timeline === selectedFacet.value;
  }),
);

function selectDimension(key: DimensionKey) {
  activeDimension.value = key;
  selectedFacet.value = facets.value[0] ?? '';
  void reloadMediaForFacet();
}

function selectFacet(facet: string) {
  selectedFacet.value = facet;
  const first = filteredMedia.value[0];
  if (first) {
    selectedMediaId.value = first.id;
  }
  void reloadMediaForFacet();
}

function selectAlbum(album: AlbumItem) {
  selectedAlbumId.value = album.id;
  activeDimension.value = 'albums';
  selectedFacet.value = album.name;
  const first = mediaItems.value.find((item) => item.album === album.name);
  if (first) {
    selectedMediaId.value = first.id;
  }
}

function selectMedia(item: MediaItem) {
  selectedMediaId.value = item.id;
  const linkedAlbum = albums.value.find((album) => album.name === item.album);
  if (linkedAlbum) {
    selectedAlbumId.value = linkedAlbum.id;
  }
}

async function loadMediaState() {
  loading.value = true;
  try {
    const [items, nextAlbums] = await Promise.all([
      apiClient.media.getItems(),
      apiClient.media.getAlbums(),
    ]);
    mediaItems.value = items;
    albums.value = nextAlbums;
    syncSelection();
    mediaNotice.value = '媒体库已连接后端，媒体索引、相册和任务状态已同步。';
  } catch (error) {
    mediaNotice.value = `后端暂不可用，继续使用本地媒体缓存：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function reloadMediaForFacet() {
  try {
    const items = await apiClient.media.getItems({ dimension: activeDimension.value, facet: selectedFacet.value });
    if (items.length > 0) {
      mediaItems.value = mergeMediaItems(mediaItems.value, items);
      selectedMediaId.value = items[0].id;
    }
  } catch {
    // Keep local filtering when the backend is unavailable.
  }
}

async function generateMemory() {
  await runMediaAction('memory', async () => {
    const task = await apiClient.media.createMemory({
      dimension: activeDimension.value,
      facet: selectedFacet.value,
    });
    memoryRuns.value += 1;
    albums.value = await apiClient.media.getAlbums();
    selectedAlbumId.value = albums.value[0]?.id ?? selectedAlbumId.value;
    mediaNotice.value = task.message ?? `AI 回忆 ${memoryRuns.value} 已生成，素材来自 ${selectedFacet.value}。`;
  });
}

async function mergePeople() {
  const sourceNames = splitPeople(selectedMedia.value.people);
  const targetName = sourceNames[0] ? `${sourceNames[0]} · 家庭成员` : selectedMedia.value.people;
  await runMediaAction('people', async () => {
    const task = await apiClient.media.mergePeople({ sourceNames, targetName });
    mediaItems.value = await apiClient.media.getItems();
    activeDimension.value = 'people';
    selectedFacet.value = targetName;
    mergeNotice.value = task.message ?? `${selectedMedia.value.people} 已合并，原识别簇保留 30 天可回滚。`;
  });
}

async function addSubtitleJob() {
  const title = selectedMedia.value.title;
  await runMediaAction('subtitle', async () => {
    const task = await apiClient.media.createSubtitleJob({ itemId: Number(selectedMedia.value.id) });
    if (!subtitleJobs.value.includes(title)) {
      subtitleJobs.value.unshift(title);
    }
    mediaItems.value = markSelectedMedia({ hasSubtitle: true, status: '字幕已加入任务' });
    mediaNotice.value = task.message ?? `${title} 已加入字幕匹配任务。`;
  });
}

async function addTranscodeJob() {
  const title = selectedMedia.value.title;
  await runMediaAction('transcode', async () => {
    const task = await apiClient.media.createTranscodeJob({
      itemId: Number(selectedMedia.value.id),
      profile: '1080p 家庭共享版本',
    });
    if (!transcodeJobs.value.includes(title)) {
      transcodeJobs.value.unshift(title);
    }
    mediaItems.value = markSelectedMedia({ transcoded: true, status: '移动端转码中' });
    mediaNotice.value = task.message ?? `${title} 正在转码为 1080p 家庭共享版本。`;
  });
}

async function toggleShare() {
  if (shareEnabled.value) {
    shareEnabled.value = false;
    albums.value = albums.value.map((album) =>
      album.id === selectedAlbumId.value ? { ...album, privacy: '链接关闭 · 已写入审计' } : album,
    );
    return;
  }
  await runMediaAction('share', async () => {
    const share = await apiClient.media.createShare({
      albumId: Number(selectedAlbum.value.id),
      expiresInDays: 7,
    });
    shareEnabled.value = true;
    albums.value = albums.value.map((album) =>
      album.id === selectedAlbumId.value ? { ...album, privacy: share.access } : album,
    );
    mediaNotice.value = `${share.name} 已开启共享链接，访问策略：${share.access}。`;
  });
}

async function runMediaAction(name: string, action: () => Promise<void>) {
  busyAction.value = name;
  try {
    await action();
  } catch (error) {
    mediaNotice.value = `媒体任务失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

function syncSelection() {
  selectedMediaId.value = mediaItems.value[0]?.id ?? selectedMediaId.value;
  selectedAlbumId.value = albums.value[0]?.id ?? selectedAlbumId.value;
  selectedFacet.value = facets.value.includes(selectedFacet.value) ? selectedFacet.value : facets.value[0] ?? '';
}

function splitPeople(value: string) {
  return value.split('/').map((item) => item.trim()).filter(Boolean);
}

function markSelectedMedia(patch: Partial<MediaItem>) {
  return mediaItems.value.map((item) => (item.id === selectedMediaId.value ? { ...item, ...patch } : item));
}

function mergeMediaItems(current: MediaItem[], updates: MediaItem[]) {
  const byID = new Map<ID, MediaItem>();
  current.forEach((item) => byID.set(item.id, item));
  updates.forEach((item) => byID.set(item.id, item));
  return [...byID.values()];
}

onMounted(loadMediaState);
</script>

<template>
  <div class="photo-media">
    <aside class="photo-media__sidebar" aria-label="媒体筛选">
      <div class="photo-media__section-title">
        <Sparkles :size="15" />
        {{ loading ? '同步媒体' : '相册媒体' }}
      </div>

      <nav class="photo-media__dimensions" aria-label="筛选维度">
        <button
          v-for="dimension in dimensionOptions"
          :key="dimension.key"
          class="photo-media__dimension"
          :class="{ 'photo-media__dimension--active': activeDimension === dimension.key }"
          type="button"
          @click="selectDimension(dimension.key)"
        >
          <component :is="dimension.icon" :size="15" />
          <span>{{ dimension.label }}</span>
        </button>
      </nav>

      <div class="photo-media__facets" aria-label="筛选值">
        <button
          v-for="facet in facets"
          :key="facet"
          class="photo-media__facet"
          :class="{ 'photo-media__facet--active': selectedFacet === facet }"
          type="button"
          @click="selectFacet(facet)"
        >
          {{ facet }}
        </button>
      </div>
    </aside>

    <main class="photo-media__main">
      <section class="photo-media__albums" aria-label="家庭、共享和智能相册">
        <button
          v-for="album in albums"
          :key="album.id"
          class="photo-media__album"
          :class="{ 'photo-media__album--active': selectedAlbumId === album.id }"
          type="button"
          @click="selectAlbum(album)"
        >
          <span>{{ album.type }}</span>
          <strong>{{ album.name }}</strong>
          <small>{{ album.count }} 项 · {{ album.privacy }}</small>
        </button>
      </section>

      <section class="photo-media__grid" aria-label="媒体项目">
        <button
          v-for="item in filteredMedia"
          :key="item.id"
          class="photo-media__tile"
          :class="{ 'photo-media__tile--active': selectedMediaId === item.id }"
          type="button"
          @click="selectMedia(item)"
        >
          <div class="photo-media__thumb" :style="{ background: item.accent }">
            <Image v-if="item.kind === '照片'" :size="24" />
            <Video v-else-if="item.kind === '视频'" :size="24" />
            <Music v-else :size="24" />
          </div>
          <strong>{{ item.title }}</strong>
          <span>{{ item.kind }} · {{ item.meta }}</span>
        </button>
        <div v-if="filteredMedia.length === 0" class="photo-media__empty">
          <Album :size="20" />
          <strong>暂无媒体</strong>
          <span>切换时间线、人物、地点、设备或相册继续浏览。</span>
        </div>
      </section>
    </main>

    <aside class="photo-media__details" aria-label="媒体详情和 Agent 操作">
      <header>
        <div>
          <p>{{ selectedAlbum.type }} · {{ selectedFacet }}</p>
          <h3>{{ selectedMedia.title }}</h3>
        </div>
        <span>{{ selectedMedia.kind }}</span>
      </header>

      <div class="photo-media__poster" :style="{ background: selectedMedia.accent }">
        <Film :size="28" />
        <strong>海报墙 / 预览</strong>
        <small>{{ selectedMedia.status }}</small>
      </div>

      <dl class="photo-media__meta">
        <div>
          <dt>人物</dt>
          <dd>{{ selectedMedia.people }}</dd>
        </div>
        <div>
          <dt>地点</dt>
          <dd>{{ selectedMedia.place }}</dd>
        </div>
        <div>
          <dt>设备</dt>
          <dd>{{ selectedMedia.device }}</dd>
        </div>
        <div>
          <dt>字幕 / 转码</dt>
          <dd>{{ selectedMedia.hasSubtitle ? '字幕可用' : '待匹配' }} · {{ selectedMedia.transcoded ? '转码中' : '原片' }}</dd>
        </div>
      </dl>

      <div class="photo-media__actions" aria-label="相册媒体操作">
        <button type="button" :disabled="busyAction === 'memory'" @click="generateMemory">
          <Wand2 :size="14" />
          {{ busyAction === 'memory' ? '生成中' : '生成回忆' }}
        </button>
        <button type="button" :disabled="busyAction === 'people'" @click="mergePeople">
          <Users :size="14" />
          {{ busyAction === 'people' ? '合并中' : '合并人物' }}
        </button>
        <button type="button" :disabled="busyAction === 'subtitle'" @click="addSubtitleJob">
          <Captions :size="14" />
          {{ busyAction === 'subtitle' ? '排队中' : '字幕任务' }}
        </button>
        <button type="button" :disabled="busyAction === 'transcode'" @click="addTranscodeJob">
          <Film :size="14" />
          {{ busyAction === 'transcode' ? '转码中' : '转码任务' }}
        </button>
        <button type="button" :disabled="busyAction === 'share'" @click="toggleShare">
          <Share2 :size="14" />
          {{ busyAction === 'share' ? '处理中' : shareEnabled ? '关闭共享' : '开启共享' }}
        </button>
      </div>

      <div class="photo-media__notice" :class="{ 'photo-media__notice--warn': shareEnabled }">
        <ShieldAlert v-if="shareEnabled" :size="16" />
        <CheckCircle2 v-else :size="16" />
        <span>{{ shareEnabled ? `${selectedAlbum.name} 已开启外链，需家庭管理员复核访问范围。` : mergeNotice }}</span>
      </div>

      <div class="photo-media__jobs" aria-label="媒体任务">
        <strong>{{ mediaNotice }}</strong>
        <span v-for="job in transcodeJobs" :key="`transcode-${job}`">转码：{{ job }}</span>
        <span v-for="job in subtitleJobs" :key="`subtitle-${job}`">字幕：{{ job }}</span>
      </div>
    </aside>
  </div>
</template>

<style scoped>
.photo-media {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr) 230px;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.photo-media__sidebar,
.photo-media__main,
.photo-media__details,
.photo-media__album,
.photo-media__tile,
.photo-media__notice,
.photo-media__jobs {
  min-width: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.photo-media__sidebar {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
}

.photo-media__section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 800;
}

.photo-media__dimensions,
.photo-media__facets {
  display: grid;
  gap: 6px;
}

.photo-media__facets {
  align-content: start;
  min-height: 0;
  overflow: auto;
}

.photo-media__dimension,
.photo-media__facet {
  min-width: 0;
  min-height: 32px;
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 700;
}

.photo-media__dimension {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 0 9px;
}

.photo-media__facet {
  padding: 8px 9px;
  background: rgba(255, 255, 255, 0.54);
  border: 1px solid rgba(100, 136, 166, 0.12);
}

.photo-media__dimension--active,
.photo-media__facet--active {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
}

.photo-media__main {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  overflow: hidden;
}

.photo-media__albums {
  display: grid;
  grid-template-columns: repeat(4, minmax(138px, 1fr));
  gap: 8px;
  overflow: auto hidden;
}

.photo-media__album {
  display: grid;
  gap: 4px;
  min-height: 72px;
  padding: 10px;
  text-align: left;
}

.photo-media__album--active {
  border-color: rgba(19, 136, 255, 0.28);
  box-shadow: inset 3px 0 0 var(--accent);
}

.photo-media__album span,
.photo-media__album small,
.photo-media__tile span,
.photo-media__poster small,
.photo-media__details header p,
.photo-media__meta dt,
.photo-media__jobs span {
  color: var(--text-muted);
  font-size: 11px;
}

.photo-media__album strong,
.photo-media__tile strong,
.photo-media__poster strong,
.photo-media__jobs strong {
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.photo-media__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
  gap: 10px;
  min-height: 0;
  overflow: auto;
}

.photo-media__tile {
  display: grid;
  gap: 7px;
  align-content: start;
  min-height: 158px;
  padding: 9px;
  text-align: left;
}

.photo-media__tile--active {
  border-color: rgba(19, 136, 255, 0.28);
  background: rgba(231, 247, 255, 0.72);
}

.photo-media__thumb {
  display: grid;
  min-height: 86px;
  place-items: center;
  color: rgba(7, 94, 194, 0.76);
  border-radius: var(--radius-sm);
}

.photo-media__tile span {
  line-height: 1.35;
}

.photo-media__empty {
  display: grid;
  min-height: 180px;
  place-items: center;
  align-content: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 11px;
}

.photo-media__empty strong {
  color: var(--text-strong);
}

.photo-media__details {
  display: grid;
  grid-template-rows: auto 126px auto auto auto minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  overflow: hidden;
}

.photo-media__details header {
  display: flex;
  align-items: start;
  justify-content: space-between;
  gap: 9px;
}

.photo-media__details header p,
.photo-media__details h3 {
  margin: 0;
}

.photo-media__details h3 {
  margin-top: 4px;
  color: var(--text-strong);
  font-size: 14px;
  line-height: 1.25;
}

.photo-media__details header > span {
  flex: 0 0 auto;
  padding: 5px 8px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.photo-media__poster {
  display: grid;
  place-items: center;
  align-content: center;
  gap: 5px;
  min-height: 0;
  color: rgba(7, 94, 194, 0.82);
  border-radius: var(--radius-md);
}

.photo-media__meta {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
}

.photo-media__meta div {
  min-width: 0;
  padding: 8px;
  background: rgba(255, 255, 255, 0.58);
  border-radius: var(--radius-sm);
}

.photo-media__meta dd {
  margin: 3px 0 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 11px;
  font-weight: 750;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.photo-media__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.photo-media__actions button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.photo-media__notice {
  display: flex;
  gap: 7px;
  padding: 9px;
  color: var(--accent-green);
  font-size: 11px;
  font-weight: 700;
  line-height: 1.35;
}

.photo-media__notice--warn {
  color: #b36a00;
  background: rgba(255, 246, 227, 0.86);
  border-color: rgba(245, 158, 11, 0.24);
}

.photo-media__jobs {
  display: grid;
  align-content: start;
  gap: 7px;
  min-height: 0;
  padding: 10px;
  overflow: auto;
}

@media (max-width: 860px) {
  .photo-media {
    grid-template-columns: 150px minmax(0, 1fr);
    overflow: auto;
  }

  .photo-media__details {
    grid-column: 1 / -1;
    grid-template-rows: auto 112px auto auto auto;
    overflow: visible;
  }
}

@media (max-width: 620px) {
  .photo-media {
    display: block;
    overflow: auto;
  }

  .photo-media__sidebar,
  .photo-media__main,
  .photo-media__details {
    margin-bottom: 10px;
  }

  .photo-media__albums {
    grid-template-columns: repeat(2, minmax(130px, 1fr));
    overflow: visible;
  }

  .photo-media__grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    max-height: none;
  }
}
</style>
