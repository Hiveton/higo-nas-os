<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Album,
  Camera,
  Captions,
  CheckCircle2,
  Film,
  Info,
  MapPin,
  RefreshCw,
  ScanFace,
  Share2,
  ShieldAlert,
  Sparkles,
  Users,
  Wand2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { aiAnalysisStore } from '../../stores/aiAnalysis';
import { UiButton, UiWindowPage } from '../ui';
import type { AlbumItem, MediaItem } from '../../api/types';
import PhotoFilterSidebar from './photo/PhotoFilterSidebar.vue';
import PhotoMediaGrid from './photo/PhotoMediaGrid.vue';
import PhotoLightbox from './photo/PhotoLightbox.vue';
import './photo/photo-window.css';

type DimensionKey = 'timeline' | 'people' | 'places' | 'devices' | 'albums';
type ID = string | number;

const dimensionOptions: Array<{ key: DimensionKey; label: string; icon: typeof Album }> = [
  { key: 'timeline', label: '时间线', icon: Film },
  { key: 'people', label: '人物', icon: Users },
  { key: 'places', label: '地点', icon: MapPin },
  { key: 'devices', label: '设备', icon: Camera },
  { key: 'albums', label: '相册', icon: Album },
];

const emptyMedia: MediaItem = {
  id: 'empty',
  title: '等待媒体扫描',
  kind: '照片',
  timeline: '未索引',
  people: '待 AI 识别',
  place: '待 AI 识别',
  device: '待 AI 识别',
  album: '未归类媒体',
  meta: '未索引',
  status: '等待扫描',
  accent: 'linear-gradient(135deg, #eef6ff, #f7fbff)',
};

const emptyAlbum: AlbumItem = {
  id: 'empty',
  name: '未归类媒体',
  type: '家庭相册',
  count: 0,
  privacy: '等待扫描',
};

const mediaItems = ref<MediaItem[]>([]);

const albums = ref<AlbumItem[]>([]);

const activeDimension = ref<DimensionKey>('timeline');
const selectedFacet = ref('');
const selectedMediaId = ref<ID>('');
const selectedAlbumId = ref<ID>('');
const shareEnabled = ref(false);
const memoryRuns = ref(0);
const mergeNotice = ref('人物识别已就绪，合并前会保留可回滚记录。');
const mediaNotice = ref('暂无媒体。');
const transcodeJobs = ref<string[]>([]);
const subtitleJobs = ref<string[]>([]);
const loading = ref(false);
const busyAction = ref('');
const searchText = ref('');
const selectionMode = ref(false);
const selectedIds = ref<Set<ID>>(new Set());
const albumNameDraft = ref('');
const albumTypeDraft = ref('家庭相册');
const albumPrivacyDraft = ref('');
const createAlbumOpen = ref(false);
const detailOpen = ref(false);

// Face self-training framework (AI 相册 人脸库). Clusters are global, surfaced
// when the 人物 dimension is active so the user can name people and retrain.
const faces = aiAnalysisStore.faces;
const faceDraft = ref<Record<string, string>>({});

const selectedMedia = computed(() => mediaItems.value.find((item) => item.id === selectedMediaId.value) ?? mediaItems.value[0] ?? emptyMedia);
const selectedAlbum = computed(() => albums.value.find((item) => item.id === selectedAlbumId.value) ?? albums.value[0] ?? emptyAlbum);
const hasMedia = computed(() => mediaItems.value.length > 0);
const selectedIdList = computed(() => [...selectedIds.value]);
const selectedNumericIds = computed(() =>
  selectedIdList.value
    .map((id) => Number(id))
    .filter((id) => Number.isFinite(id) && id > 0),
);
const selectedCount = computed(() => selectedIds.value.size);
const mediaStats = computed(() => {
  const photos = mediaItems.value.filter((item) => item.kind === '照片').length;
  const videos = mediaItems.value.filter((item) => item.kind === '视频').length;
  const music = mediaItems.value.filter((item) => item.kind === '音乐').length;
  return [
    { label: '照片', value: photos },
    { label: '视频', value: videos },
    { label: '音频', value: music },
    { label: '任务', value: transcodeJobs.value.length + subtitleJobs.value.length },
  ];
});

const facets = computed(() => {
  const values = mediaItems.value.flatMap((item) => {
    if (activeDimension.value === 'people') return splitPeople(item.people);
    if (activeDimension.value === 'places') return item.place;
    if (activeDimension.value === 'devices') return item.device;
    if (activeDimension.value === 'albums') return item.album;
    return item.timeline;
  }).filter(Boolean);
  return [...new Set(values)];
});

const facetMedia = computed(() =>
  mediaItems.value.filter((item) => {
    if (activeDimension.value === 'people') return splitPeople(item.people).includes(selectedFacet.value);
    if (activeDimension.value === 'places') return item.place === selectedFacet.value;
    if (activeDimension.value === 'devices') return item.device === selectedFacet.value;
    if (activeDimension.value === 'albums') return item.album === selectedFacet.value;
    return item.timeline === selectedFacet.value;
  }),
);
const filteredMedia = computed(() => {
  const q = searchText.value.trim().toLowerCase();
  if (!q) return facetMedia.value;
  return facetMedia.value.filter((item) =>
    [item.title, item.kind, item.timeline, item.people, item.place, item.device, item.album, item.meta, item.status]
      .join(' ')
      .toLowerCase()
      .includes(q),
  );
});
const allVisibleSelected = computed(() =>
  filteredMedia.value.length > 0 && filteredMedia.value.every((item) => selectedIds.value.has(item.id)),
);

function selectDimension(key: DimensionKey) {
  activeDimension.value = key;
  selectedFacet.value = facets.value[0] ?? '';
  clearSelection();
  void reloadMediaForFacet();
  if (key === 'people') {
    void aiAnalysisStore.loadFaces();
  }
}

async function analyzeSelectedMedia() {
  const id = selectedMedia.value.id;
  if (!id || id === 'empty') return;
  await runMediaAction('analyze', async () => {
    await aiAnalysisStore.reanalyze({ scope: 'item', itemId: `media:${id}` });
    mediaItems.value = markSelectedMedia({ status: '已加入 AI 分析队列' });
    mediaNotice.value = `已将「${selectedMedia.value.title}」加入 AI 分析队列，结果稍后回填。`;
  });
}

async function renameCluster(label: string) {
  const name = (faceDraft.value[label] ?? '').trim();
  if (!name) {
    mediaNotice.value = '请输入人物名称。';
    return;
  }
  await runMediaAction('face', async () => {
    const confirmed = await aiAnalysisStore.labelFace(label, name);
    faceDraft.value = { ...faceDraft.value, [label]: '' };
    mediaNotice.value = `已将人脸簇「${label}」命名为「${name}」，确认 ${confirmed} 个样本。`;
  });
}

async function retrainFaces() {
  await runMediaAction('retrain', async () => {
    const taskId = await aiAnalysisStore.retrainFaces();
    mediaNotice.value = `已启动人脸模型自训练任务（${taskId}）。`;
  });
}

function selectFacet(facet: string) {
  selectedFacet.value = facet;
  const first = filteredMedia.value[0];
  if (first) {
    selectedMediaId.value = first.id;
  }
  clearSelection();
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
  clearSelection();
}

function selectMedia(item: MediaItem) {
  if (selectionMode.value) {
    toggleSelection(item);
    return;
  }
  selectedMediaId.value = item.id;
  const linkedAlbum = albums.value.find((album) => album.name === item.album);
  if (linkedAlbum) {
    selectedAlbumId.value = linkedAlbum.id;
  }
  detailOpen.value = true;
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
    mediaNotice.value = items.length > 0 ? '媒体库已同步。' : '暂无媒体。';
  } catch (error) {
    mediaItems.value = [];
    albums.value = [];
    mediaNotice.value = `后端暂不可用：${error instanceof Error ? error.message : 'unknown error'}`;
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

async function createAlbumFromSelection() {
  const name = albumNameDraft.value.trim();
  const itemIds = selectedNumericIds.value;
  if (!name) {
    mediaNotice.value = '请输入相册名称。';
    return;
  }
  if (itemIds.length === 0) {
    mediaNotice.value = '请先选择要加入相册的媒体。';
    return;
  }
  await runMediaAction('createAlbum', async () => {
    const album = await apiClient.media.createAlbum({
      name,
      type: albumTypeDraft.value,
      itemIds,
      privacy: albumPrivacyDraft.value.trim(),
    });
    albums.value = await apiClient.media.getAlbums();
    mediaItems.value = await apiClient.media.getItems();
    selectedAlbumId.value = album.id;
    activeDimension.value = 'albums';
    selectedFacet.value = album.name;
    clearSelection();
    createAlbumOpen.value = false;
    albumNameDraft.value = '';
    albumPrivacyDraft.value = '';
    mediaNotice.value = `${album.name} 已创建，已加入 ${album.count} 个媒体项目。`;
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
  selectedMediaId.value = mediaItems.value[0]?.id ?? '';
  selectedAlbumId.value = albums.value[0]?.id ?? '';
  selectedFacet.value = facets.value.includes(selectedFacet.value) ? selectedFacet.value : facets.value[0] ?? '';
}

function toggleSelection(item: MediaItem) {
  const next = new Set(selectedIds.value);
  if (next.has(item.id)) {
    next.delete(item.id);
  } else {
    next.add(item.id);
    selectedMediaId.value = item.id;
  }
  selectedIds.value = next;
}

function toggleSelectionMode() {
  selectionMode.value = !selectionMode.value;
  if (!selectionMode.value) {
    clearSelection();
  }
}

function selectAllVisible() {
  if (allVisibleSelected.value) {
    selectedIds.value = new Set();
    return;
  }
  selectedIds.value = new Set(filteredMedia.value.map((item) => item.id));
  selectedMediaId.value = filteredMedia.value[0]?.id ?? selectedMediaId.value;
}

function clearSelection() {
  selectedIds.value = new Set();
  selectionMode.value = false;
  createAlbumOpen.value = false;
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

onMounted(() => {
  void loadMediaState();
  void aiAnalysisStore.loadFaces();
});
</script>

<template>
  <UiWindowPage
    layout="master-detail"
    :icon="Camera"
    title="照片媒体"
    :subtitle="`${selectedAlbum.type} · ${selectedFacet || '全部'}`"
    :status="mediaNotice"
  >
    <template #nav>
      <PhotoFilterSidebar
        :loading="loading"
        :dimension-options="dimensionOptions"
        :active-dimension="activeDimension"
        :facets="facets"
        :selected-facet="selectedFacet"
        @select-dimension="selectDimension"
        @select-facet="selectFacet"
      />
    </template>

    <PhotoMediaGrid
      v-model:search-text="searchText"
      v-model:create-album-open="createAlbumOpen"
      v-model:album-name-draft="albumNameDraft"
      v-model:album-type-draft="albumTypeDraft"
      v-model:album-privacy-draft="albumPrivacyDraft"
      :media-stats="mediaStats"
      :selection-mode="selectionMode"
      :selected-count="selectedCount"
      :all-visible-selected="allVisibleSelected"
      :busy-action="busyAction"
      :albums="albums"
      :selected-album-id="selectedAlbumId"
      :filtered-media="filteredMedia"
      :selected-media-id="selectedMediaId"
      :selected-ids="selectedIds"
      @toggle-selection-mode="toggleSelectionMode"
      @select-all-visible="selectAllVisible"
      @create-album="createAlbumFromSelection"
      @select-album="selectAlbum"
      @select-media="selectMedia"
    />

    <template #inspector>
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

      <div v-if="selectedMedia.caption" class="photo-media__caption">
        <Sparkles :size="14" />
        <p>{{ selectedMedia.caption }}</p>
      </div>

      <section v-if="activeDimension === 'people'" class="photo-faces" aria-label="人脸库与自训练">
        <header class="photo-faces__head">
          <div class="photo-faces__title">
            <ScanFace :size="15" />
            <strong>人脸库</strong>
            <span v-if="faces">{{ faces.training.namedPeople }} 已命名 · {{ faces.clusters.length }} 簇</span>
          </div>
          <UiButton
            variant="ghost"
            size="sm"
            :icon-left="RefreshCw"
            :loading="busyAction === 'retrain'"
            :disabled="!faces?.trainerReady"
            :title="faces?.trainerReady ? '基于已命名样本重新训练人脸模型' : '未配置人脸训练 sidecar（HIGO_FACE_TRAINER_URL）'"
            @click="retrainFaces"
          >
            自训练
          </UiButton>
        </header>

        <p v-if="!faces || faces.clusters.length === 0" class="photo-faces__empty">
          暂无人脸聚类。开启 AI 分析（深度等级）并配置视觉模型后，相册照片中的人脸会在这里成簇出现，可命名归并。
        </p>

        <ul v-else class="photo-faces__list">
          <li v-for="cluster in faces.clusters" :key="cluster.label" class="photo-faces__item">
            <div class="photo-faces__item-info">
              <span class="photo-faces__item-name">{{ cluster.label }}</span>
              <span class="photo-faces__item-count">{{ cluster.count }} 张</span>
            </div>
            <div class="photo-faces__rename">
              <input
                v-model="faceDraft[cluster.label]"
                class="photo-faces__input"
                type="text"
                placeholder="命名此人"
                @keyup.enter="renameCluster(cluster.label)"
              />
              <UiButton variant="soft" size="sm" :loading="busyAction === 'face'" @click="renameCluster(cluster.label)">
                命名
              </UiButton>
            </div>
          </li>
        </ul>
      </section>

      <div class="photo-media__actions" aria-label="相册媒体操作">
        <UiButton variant="soft" size="sm" :icon-left="Sparkles" :disabled="!hasMedia" :loading="busyAction === 'analyze'" @click="analyzeSelectedMedia">
          {{ busyAction === 'analyze' ? '排队中' : 'AI 分析' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Wand2" :disabled="!hasMedia" :loading="busyAction === 'memory'" @click="generateMemory">
          {{ busyAction === 'memory' ? '生成中' : '生成回忆' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Users" :disabled="!hasMedia" :loading="busyAction === 'people'" @click="mergePeople">
          {{ busyAction === 'people' ? '合并中' : '合并人物' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Captions" :disabled="!hasMedia" :loading="busyAction === 'subtitle'" @click="addSubtitleJob">
          {{ busyAction === 'subtitle' ? '排队中' : '字幕任务' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Film" :disabled="!hasMedia" :loading="busyAction === 'transcode'" @click="addTranscodeJob">
          {{ busyAction === 'transcode' ? '转码中' : '转码任务' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Share2" :disabled="albums.length === 0" :loading="busyAction === 'share'" @click="toggleShare">
          {{ busyAction === 'share' ? '处理中' : shareEnabled ? '关闭共享' : '开启共享' }}
        </UiButton>
        <UiButton variant="soft" size="sm" :icon-left="Info" :disabled="!hasMedia" @click="detailOpen = true">
          查看详情
        </UiButton>
      </div>

      <div v-if="hasMedia || shareEnabled" class="photo-media__notice" :class="{ 'photo-media__notice--warn': shareEnabled }">
        <ShieldAlert v-if="shareEnabled" :size="16" />
        <CheckCircle2 v-else :size="16" />
        <span>{{ shareEnabled ? `${selectedAlbum.name} 已开启外链，需家庭管理员复核访问范围。` : mergeNotice }}</span>
      </div>

      <div v-if="hasMedia || transcodeJobs.length > 0 || subtitleJobs.length > 0" class="photo-media__jobs" aria-label="媒体任务">
        <strong>{{ mediaNotice }}</strong>
        <span v-for="job in transcodeJobs" :key="`transcode-${job}`">转码：{{ job }}</span>
        <span v-for="job in subtitleJobs" :key="`subtitle-${job}`">字幕：{{ job }}</span>
      </div>
    </aside>
    </template>

    <PhotoLightbox
      v-if="detailOpen"
      :selected-media="selectedMedia"
      :has-media="hasMedia"
      :busy-action="busyAction"
      @close="detailOpen = false"
      @add-subtitle="addSubtitleJob"
      @add-transcode="addTranscodeJob"
      @generate-memory="generateMemory"
    />
  </UiWindowPage>
</template>

<style scoped>
.photo-media__caption {
  display: flex;
  gap: var(--space-2);
  align-items: flex-start;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md, 10px);
  background: var(--surface-subtle, rgba(125, 125, 145, 0.08));
  color: var(--text-muted);
  font-size: var(--fs-xs);
}
.photo-media__caption :deep(svg) {
  flex-shrink: 0;
  margin-top: 2px;
  color: var(--accent);
}
.photo-media__caption p {
  margin: 0;
  line-height: var(--lh-normal);
}

.photo-faces {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-radius: var(--radius-md, 10px);
  border: 1px solid var(--border-subtle, rgba(125, 125, 145, 0.18));
}
.photo-faces__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}
.photo-faces__title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.photo-faces__title span {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
.photo-faces__empty {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: var(--lh-normal);
}
.photo-faces__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}
.photo-faces__item {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  padding-bottom: var(--space-2);
  border-bottom: 1px dashed var(--border-subtle, rgba(125, 125, 145, 0.18));
}
.photo-faces__item:last-child {
  border-bottom: none;
  padding-bottom: 0;
}
.photo-faces__item-info {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.photo-faces__item-name {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.photo-faces__item-count {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
.photo-faces__rename {
  display: flex;
  gap: var(--space-2);
}
.photo-faces__input {
  flex: 1;
  min-width: 0;
  padding: 4px 8px;
  border-radius: var(--radius-sm, 8px);
  border: 1px solid var(--border-subtle, rgba(125, 125, 145, 0.28));
  background: var(--surface, transparent);
  color: var(--text-strong);
  font-size: var(--fs-xs);
}
</style>
