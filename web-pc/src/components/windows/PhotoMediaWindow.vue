<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Album,
  Camera,
  Captions,
  CheckCircle2,
  Film,
  Grid2X2,
  Info,
  Image,
  MapPin,
  Music,
  Plus,
  Search,
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
          @click="toggleSelectionMode"
        >
          <Grid2X2 :size="14" />
          {{ selectionMode ? '取消选择' : '批量选择' }}
        </button>
      </section>

      <section v-if="selectionMode || selectedCount > 0" class="photo-media__selection" aria-label="批量操作">
        <button type="button" @click="selectAllVisible">
          {{ allVisibleSelected ? '取消全选' : '全选当前' }}
        </button>
        <span>已选择 {{ selectedCount }} 项</span>
        <button type="button" :disabled="selectedCount === 0" @click="createAlbumOpen = !createAlbumOpen">
          <Plus :size="14" />
          新建相册
        </button>
      </section>

      <section v-if="createAlbumOpen" class="photo-media__create" aria-label="创建相册">
        <label>
          <span>相册名称</span>
          <input v-model="albumNameDraft" type="text" placeholder="例如：端午出游精选" />
        </label>
        <label>
          <span>类型</span>
          <select v-model="albumTypeDraft">
            <option>家庭相册</option>
            <option>共享相册</option>
            <option>智能回忆</option>
          </select>
        </label>
        <label>
          <span>权限</span>
          <input v-model="albumPrivacyDraft" type="text" placeholder="留空使用默认策略" />
        </label>
        <button type="button" :disabled="busyAction === 'createAlbum'" @click="createAlbumFromSelection">
          <Plus :size="14" />
          {{ busyAction === 'createAlbum' ? '创建中' : '创建' }}
        </button>
      </section>

      <section v-if="albums.length > 0" class="photo-media__albums" aria-label="家庭、共享和智能相册">
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
          @click="selectMedia(item)"
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
        <div v-if="filteredMedia.length === 0" class="photo-media__empty">
          <Album :size="20" />
          <strong>暂无媒体</strong>
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
        <button type="button" :disabled="!hasMedia || busyAction === 'memory'" @click="generateMemory">
          <Wand2 :size="14" />
          {{ busyAction === 'memory' ? '生成中' : '生成回忆' }}
        </button>
        <button type="button" :disabled="!hasMedia || busyAction === 'people'" @click="mergePeople">
          <Users :size="14" />
          {{ busyAction === 'people' ? '合并中' : '合并人物' }}
        </button>
        <button type="button" :disabled="!hasMedia || busyAction === 'subtitle'" @click="addSubtitleJob">
          <Captions :size="14" />
          {{ busyAction === 'subtitle' ? '排队中' : '字幕任务' }}
        </button>
        <button type="button" :disabled="!hasMedia || busyAction === 'transcode'" @click="addTranscodeJob">
          <Film :size="14" />
          {{ busyAction === 'transcode' ? '转码中' : '转码任务' }}
        </button>
        <button type="button" :disabled="albums.length === 0 || busyAction === 'share'" @click="toggleShare">
          <Share2 :size="14" />
          {{ busyAction === 'share' ? '处理中' : shareEnabled ? '关闭共享' : '开启共享' }}
        </button>
        <button type="button" :disabled="!hasMedia" @click="detailOpen = true">
          <Info :size="14" />
          查看详情
        </button>
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
    <div v-if="detailOpen" class="photo-media__lightbox" role="dialog" aria-modal="true" aria-label="媒体详情">
      <div class="photo-media__lightbox-card">
        <button class="photo-media__close" type="button" aria-label="关闭详情" @click="detailOpen = false">×</button>
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
            <button type="button" :disabled="!hasMedia || busyAction === 'subtitle'" @click="addSubtitleJob">
              <Captions :size="14" />
              字幕任务
            </button>
            <button type="button" :disabled="!hasMedia || busyAction === 'transcode'" @click="addTranscodeJob">
              <Film :size="14" />
              转码任务
            </button>
            <button type="button" :disabled="!hasMedia || busyAction === 'memory'" @click="generateMemory">
              <Wand2 :size="14" />
              生成回忆
            </button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.photo-media {
  display: grid;
  grid-template-columns: 180px minmax(0, 1fr) 230px;
  grid-template-rows: minmax(0, 1fr);
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
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 12px;
  overflow: hidden;
}

.photo-media__toolbar {
  display: grid;
  grid-template-columns: minmax(180px, 1fr) max-content auto;
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.photo-media__search {
  display: flex;
  align-items: center;
  gap: 7px;
  min-width: 0;
  height: 32px;
  padding: 0 9px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.photo-media__search input,
.photo-media__create input,
.photo-media__create select {
  min-width: 0;
  width: 100%;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: none;
  font: inherit;
  font-size: 12px;
  line-height: 1.2;
}

.photo-media__stats {
  display: flex;
  gap: 5px;
  min-width: 0;
  overflow: auto hidden;
}

.photo-media__stats span {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  flex: 0 0 auto;
  height: 28px;
  padding: 0 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 700;
  line-height: 1;
}

.photo-media__stats strong {
  color: var(--text-strong);
  font-size: 12px;
  line-height: 1;
}

.photo-media__tool,
.photo-media__selection button,
.photo-media__create button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 780;
}

.photo-media__tool--active,
.photo-media__selection button:not(:disabled):hover,
.photo-media__create button:not(:disabled):hover {
  background: rgba(19, 136, 255, 0.14);
}

.photo-media__selection {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
  padding: 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-md);
  font-size: 11px;
  font-weight: 740;
}

.photo-media__selection span {
  flex: 1 1 auto;
  min-width: 0;
}

.photo-media__create {
  display: grid;
  grid-template-columns: minmax(150px, 1.1fr) minmax(120px, 0.7fr) minmax(150px, 1fr) auto;
  gap: 8px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.54);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-md);
}

.photo-media__create label {
  display: grid;
  gap: 5px;
  min-width: 0;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 740;
}

.photo-media__create input,
.photo-media__create select {
  height: 32px;
  padding: 0 10px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(100, 136, 166, 0.16);
  border-radius: var(--radius-sm);
}

.photo-media__create button {
  align-self: end;
  min-width: 76px;
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
  flex: 1 1 auto;
  min-height: 0;
  align-content: start;
  overflow: auto;
}

.photo-media__grid--empty {
  grid-template-columns: 1fr;
  place-items: center;
  align-content: center;
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

.photo-media__tile--selected {
  border-color: rgba(34, 197, 94, 0.42);
  box-shadow: inset 0 0 0 2px rgba(34, 197, 94, 0.16);
}

.photo-media__thumb {
  position: relative;
  display: grid;
  min-height: 86px;
  place-items: center;
  color: rgba(7, 94, 194, 0.76);
  border-radius: var(--radius-sm);
}

.photo-media__tile span,
.photo-media__tile small {
  line-height: 1.35;
}

.photo-media__tile small {
  overflow: hidden;
  color: var(--text-muted);
  font-size: 10px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.photo-media__check {
  position: absolute;
  top: 7px;
  right: 7px;
  display: grid;
  width: 20px;
  height: 20px;
  place-items: center;
  color: white;
  background: rgba(34, 197, 94, 0.88);
  border: 1px solid rgba(255, 255, 255, 0.7);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 900;
}

.photo-media__empty {
  display: grid;
  min-height: 160px;
  width: min(260px, 100%);
  place-items: center;
  align-content: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 11px;
  text-align: center;
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

.photo-media__lightbox {
  position: fixed;
  inset: 0;
  z-index: 3000;
  display: grid;
  place-items: center;
  padding: 22px;
  background: rgba(10, 20, 34, 0.42);
  backdrop-filter: blur(12px);
}

.photo-media__lightbox-card {
  position: relative;
  display: grid;
  grid-template-columns: minmax(240px, 1.1fr) minmax(260px, 0.9fr);
  gap: 16px;
  width: min(860px, calc(100vw - 44px));
  max-height: min(680px, calc(100vh - 44px));
  padding: 16px;
  overflow: auto;
  background: rgba(247, 251, 255, 0.96);
  border: 1px solid rgba(205, 220, 235, 0.8);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-window);
}

.photo-media__close {
  position: absolute;
  top: 10px;
  right: 10px;
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 20px;
  line-height: 1;
}

.photo-media__preview {
  display: grid;
  min-height: 420px;
  place-items: center;
  color: rgba(7, 94, 194, 0.82);
  border-radius: var(--radius-md);
}

.photo-media__lightbox-info {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
  padding: 28px 4px 4px;
}

.photo-media__lightbox-info p,
.photo-media__lightbox-info h2 {
  margin: 0;
}

.photo-media__lightbox-info p,
.photo-media__lightbox-meta dt {
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 740;
}

.photo-media__lightbox-info h2 {
  color: var(--text-strong);
  font-size: 20px;
  line-height: 1.22;
}

.photo-media__chips {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.photo-media__chips span {
  padding: 6px 9px;
  color: var(--text-muted);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.14);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.photo-media__lightbox-meta {
  display: grid;
  gap: 8px;
  margin: 0;
}

.photo-media__lightbox-meta div {
  padding: 10px;
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid rgba(100, 136, 166, 0.1);
  border-radius: var(--radius-sm);
}

.photo-media__lightbox-meta dd {
  margin: 4px 0 0;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 760;
  line-height: 1.42;
}

.photo-media__actions--dialog {
  margin-top: 2px;
}

@media (max-width: 860px) {
  .photo-media {
    grid-template-columns: 150px minmax(0, 1fr);
    overflow: auto;
  }

  .photo-media__toolbar,
  .photo-media__create {
    grid-template-columns: 1fr;
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

  .photo-media__toolbar {
    grid-template-columns: 1fr;
  }

  .photo-media__selection {
    flex-wrap: wrap;
  }

  .photo-media__lightbox-card {
    grid-template-columns: 1fr;
  }

  .photo-media__preview {
    min-height: 300px;
  }
}
</style>
