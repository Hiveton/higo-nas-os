<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import {
  Captions,
  CheckCircle2,
  Clapperboard,
  Film,
  FolderOpen,
  Gauge,
  ListVideo,
  Loader2,
  Play,
  Plus,
  Radio,
  RefreshCcw,
  Search,
  Settings,
  Sparkles,
  Tv,
  Trash2,
  Wand2,
  X,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type {
  DvrSettings,
  LiveChannel,
  LiveGuideSource,
  LiveProgram,
  LiveSource,
  RecordingItem,
  RecordingTimer,
  VideoItem,
  VideoLibrary,
  VideoMediaTrack,
  VideoTask,
} from '../../api/types';

type TabKey = 'home' | 'movies' | 'live' | 'tasks' | 'settings';
type LiveViewKey = 'channels' | 'guide' | 'recordings';

const tabs: Array<{ id: TabKey; label: string; icon: unknown }> = [
  { id: 'home', label: '首页', icon: Film },
  { id: 'movies', label: '影片', icon: Clapperboard },
  { id: 'live', label: '电视直播', icon: Tv },
  { id: 'tasks', label: '任务', icon: ListVideo },
  { id: 'settings', label: '设置', icon: Settings },
];

const activeTab = ref<TabKey>('home');
const loading = ref(false);
const busyMessage = ref('');
const statusMessage = ref('');
const errorMessage = ref('');
const query = ref('');
const kindFilter = ref('');
const libraryFilter = ref('');
const libraries = ref<VideoLibrary[]>([]);
const items = ref<VideoItem[]>([]);
const tasks = ref<VideoTask[]>([]);
const liveSources = ref<LiveSource[]>([]);
const liveChannels = ref<LiveChannel[]>([]);
const guideSources = ref<LiveGuideSource[]>([]);
const livePrograms = ref<LiveProgram[]>([]);
const recordingTimers = ref<RecordingTimer[]>([]);
const recordings = ref<RecordingItem[]>([]);
const liveView = ref<LiveViewKey>('channels');
const selectedItemId = ref('');
const selectedProgram = ref<LiveProgram | undefined>();
const epgShellRef = ref<HTMLElement>();
const epgDragging = ref(false);
const playbackTitle = ref('');
const playbackSrc = ref('');
const playbackPoster = ref('');
const playbackSubtitle = ref('');
const playbackError = ref('');
const transcodeProfile = ref('1080p H.264');
const newLibrary = ref({
  name: '电影',
  type: 'movie',
  path: '',
  metadataLanguage: 'zh-CN',
  allowAdultContent: false,
  autoSubtitles: true,
  subtitleLanguage: 'zh-CN',
});
const newLiveSource = ref({
  name: '直播源',
  url: '',
  userAgent: '',
  streamLimit: 0,
});
const newGuideSource = ref({
  name: '节目指南',
  url: '',
  userAgent: '',
});
const dvrSettings = ref<DvrSettings>({
  recordingPath: '/srv/higoos/recordings',
  movieRecordingPath: '',
  seriesRecordingPath: '',
  prePaddingSeconds: 60,
  postPaddingSeconds: 180,
  maxConcurrentRecord: 1,
  saveNfo: true,
  saveImages: true,
  postProcessCommand: '',
});

const filteredItems = computed(() => {
  const text = query.value.trim().toLowerCase();
  return items.value.filter((item) => {
    if (kindFilter.value && item.kind !== kindFilter.value) return false;
    if (libraryFilter.value && item.libraryId !== libraryFilter.value) return false;
    if (!text) return true;
    return [
      item.title,
      item.originalTitle,
      item.seriesTitle,
      item.episodeTitle,
      item.fileName,
      item.libraryName,
      ...(item.genres ?? []),
      ...(item.tags ?? []),
      ...(item.directors ?? []),
      ...(item.writers ?? []),
      ...(item.actors ?? []),
      ...(item.studios ?? []),
      ...(item.countries ?? []),
    ]
      .join(' ')
      .toLowerCase()
      .includes(text);
  });
});

const selectedItem = computed(() => {
  if (selectedItemId.value) {
    const match = items.value.find((item) => item.id === selectedItemId.value);
    if (match) return match;
  }
  return filteredItems.value[0] ?? items.value[0];
});
const detailItemId = ref('');
const detailItem = computed(() => {
  if (!detailItemId.value) return undefined;
  return items.value.find((item) => item.id === detailItemId.value);
});

const runningTasks = computed(() => tasks.value.filter((task) => task.status === 'running' || task.status === 'queued'));
const movieCount = computed(() => items.value.filter((item) => item.kind === 'movie').length);
const seriesCount = computed(() => items.value.filter((item) => item.kind === 'episode').length);
const libraryStatus = computed(() => {
  if (!libraries.value.length) return '未设置媒体库';
  if (!items.value.length) return '已设置媒体库，等待扫描';
  return `已索引 ${items.value.length} 个媒体文件`;
});
const scheduledProgramIds = computed(() => new Set(recordingTimers.value.filter((timer) => timer.status !== 'cancelled').map((timer) => timer.programId).filter(Boolean)));
const guidePixelsPerMinute = 3;
const guideWindowMinutes = 24 * 60;
const sortedLivePrograms = computed(() =>
  [...livePrograms.value].sort((a, b) => new Date(a.startAt).getTime() - new Date(b.startAt).getTime()),
);
const guideWindowStart = computed(() => {
  const now = new Date();
  const hasCurrentPrograms = livePrograms.value.some((program) => programStatus(program) === 'current');
  if (hasCurrentPrograms) {
    now.setHours(0, 0, 0, 0);
    return now;
  }
  const first = sortedLivePrograms.value[0];
  if (!first) {
    now.setHours(0, 0, 0, 0);
    return now;
  }
  const start = new Date(first.startAt);
  start.setHours(0, 0, 0, 0);
  return start;
});
const guideWindowEnd = computed(() => new Date(guideWindowStart.value.getTime() + guideWindowMinutes * 60 * 1000));
const guideTimeSlots = computed(() => {
  const slots = [];
  for (let minute = 0; minute <= guideWindowMinutes; minute += 30) {
    const time = new Date(guideWindowStart.value.getTime() + minute * 60 * 1000);
    slots.push({ minute, label: time.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' }) });
  }
  return slots;
});
const guideRows = computed(() => {
  const channelMap = new Map(liveChannels.value.map((channel) => [channel.id, channel]));
  const channelOrder = new Map(liveChannels.value.map((channel, index) => [channel.id, index]));
  const rows = new Map<string, { id: string; name: string; group: string; logo?: string; programs: LiveProgram[] }>();
  for (const program of sortedLivePrograms.value) {
    if (!programOverlapsWindow(program)) continue;
    const channel = channelMap.get(program.channelId);
    const row =
      rows.get(program.channelId) ??
      { id: program.channelId, name: program.channelName, group: '节目指南', programs: [] };
    row.name = channel?.name || program.channelName || row.name;
    row.group = channel?.group || row.group;
    row.logo = channel?.logo || row.logo;
    row.programs.push(program);
    rows.set(program.channelId, row);
  }
  return [...rows.values()]
    .filter((row) => row.programs.length > 0)
    .sort((a, b) => {
      const orderA = channelOrder.get(a.id);
      const orderB = channelOrder.get(b.id);
      if (orderA !== undefined && orderB !== undefined) return orderA - orderB;
      if (orderA !== undefined) return -1;
      if (orderB !== undefined) return 1;
      return a.name.localeCompare(b.name, 'zh-CN');
    })
    .slice(0, 120);
});
const currentGuideOffset = computed(() => {
  const now = Date.now();
  const start = guideWindowStart.value.getTime();
  const end = guideWindowEnd.value.getTime();
  if (now < start || now > end) return -1;
  return ((now - start) / 60000) * guidePixelsPerMinute;
});
let epgPointerId = -1;
let epgDragStartX = 0;
let epgDragStartY = 0;
let epgDragStartScrollLeft = 0;
let epgSuppressProgramClick = false;

async function loadAll() {
  loading.value = true;
  errorMessage.value = '';
  try {
    const [libraryState, nextItems, nextTasks, sources, channels, guides, programs, settings, timers, nextRecordings] = await Promise.all([
      apiClient.video.getLibrary(),
      apiClient.video.getItems(),
      apiClient.video.getTasks(),
      apiClient.video.getLiveSources(),
      apiClient.video.getLiveChannels(),
      apiClient.video.getGuideSources(),
      apiClient.video.getPrograms(),
      apiClient.video.getDvrSettings(),
      apiClient.video.getRecordingTimers(),
      apiClient.video.getRecordings(),
    ]);
    libraries.value = libraryState.libraries ?? [];
    items.value = nextItems ?? [];
    tasks.value = nextTasks ?? [];
    liveSources.value = sources ?? [];
    liveChannels.value = channels ?? [];
    guideSources.value = guides ?? [];
    livePrograms.value = programs ?? [];
    dvrSettings.value = settings;
    recordingTimers.value = timers ?? [];
    recordings.value = nextRecordings ?? [];
    if (!selectedItemId.value && items.value[0]) selectedItemId.value = items.value[0].id;
    statusMessage.value = libraryState.status || libraryStatus.value;
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取影视数据失败';
  } finally {
    loading.value = false;
  }
}

async function scanLibrary() {
  await runBusy('正在扫描媒体库', async () => {
    const result = await apiClient.video.scan();
    statusMessage.value = result.message;
    await loadAll();
  });
}

async function createLibrary() {
  const path = newLibrary.value.path.trim();
  if (!path) {
    errorMessage.value = '请输入媒体库目录路径';
    return;
  }
  await runBusy('正在创建媒体库', async () => {
    await apiClient.video.createLibrary({
      name: newLibrary.value.name.trim() || '媒体库',
      type: newLibrary.value.type,
      paths: [path],
      metadataLanguage: newLibrary.value.metadataLanguage,
      allowAdultContent: newLibrary.value.allowAdultContent,
      autoSubtitles: newLibrary.value.autoSubtitles,
      subtitleLanguage: newLibrary.value.subtitleLanguage,
    });
    newLibrary.value.path = '';
    await loadAll();
    activeTab.value = 'movies';
  });
}

async function deleteLibrary(libraryItem: VideoLibrary) {
  const confirmed = window.confirm(`删除媒体库「${libraryItem.name}」？此操作只移除媒体库配置、索引和任务，不删除磁盘文件。`);
  if (!confirmed) return;
  await runBusy('正在删除媒体库', async () => {
    const selectedRemoved = selectedItem.value?.libraryId === libraryItem.id;
    await apiClient.video.deleteLibrary(libraryItem.id);
    if (libraryFilter.value === libraryItem.id) libraryFilter.value = '';
    if (selectedRemoved) selectedItemId.value = '';
    await loadAll();
    activeTab.value = 'settings';
  });
}

async function createLiveSource() {
  if (!newLiveSource.value.url.trim()) {
    errorMessage.value = '请输入 M3U 地址或服务器本地路径';
    return;
  }
  await runBusy('正在解析直播源', async () => {
    await apiClient.video.createLiveSource({
      name: newLiveSource.value.name.trim() || '直播源',
      url: newLiveSource.value.url.trim(),
      userAgent: newLiveSource.value.userAgent.trim() || undefined,
      streamLimit: Number(newLiveSource.value.streamLimit) || undefined,
    });
    newLiveSource.value.url = '';
    await loadAll();
  });
}

async function createGuideSource() {
  if (!newGuideSource.value.url.trim()) {
    errorMessage.value = '请输入 XMLTV 地址或服务器本地路径';
    return;
  }
  await runBusy('正在导入节目指南', async () => {
    const guide = await apiClient.video.createGuideSource({
      name: newGuideSource.value.name.trim() || '节目指南',
      url: newGuideSource.value.url.trim(),
      userAgent: newGuideSource.value.userAgent.trim() || undefined,
    });
    newGuideSource.value.url = '';
    await loadAll();
    statusMessage.value = `已导入节目指南：${guide.name}，匹配 ${guide.programCount} 个节目`;
    liveView.value = 'guide';
  });
}

async function saveDvrSettings() {
  await runBusy('正在保存 DVR 设置', async () => {
    dvrSettings.value = await apiClient.video.updateDvrSettings({
      ...dvrSettings.value,
      prePaddingSeconds: Number(dvrSettings.value.prePaddingSeconds) || 0,
      postPaddingSeconds: Number(dvrSettings.value.postPaddingSeconds) || 0,
      maxConcurrentRecord: Number(dvrSettings.value.maxConcurrentRecord) || 1,
    });
  });
}

async function recordProgram(program: LiveProgram) {
  await runBusy(programStatus(program) === 'current' ? '正在开始录制' : '正在创建录制预约', async () => {
    const timer = await apiClient.video.createRecordingTimer({ programId: program.id });
    await loadAll();
    selectedProgram.value = undefined;
    statusMessage.value = timer.status === 'recording' ? `正在录制：${timer.name}` : `已预约录制：${timer.name}`;
  });
}

async function cancelRecording(timer: RecordingTimer) {
  await runBusy('正在取消录制预约', async () => {
    await apiClient.video.cancelRecordingTimer(timer.id);
    await loadAll();
  });
}

async function createTask(type: 'scrape' | 'subtitle' | 'transcode', item?: VideoItem) {
  const target = item ?? selectedItem.value;
  if (!target) return;
  await runBusy('正在创建任务', async () => {
    if (type === 'scrape') await apiClient.video.scrape({ itemId: target.id });
    if (type === 'subtitle') await apiClient.video.subtitle({ itemId: target.id });
    if (type === 'transcode') await apiClient.video.transcode({ itemId: target.id, profile: transcodeProfile.value });
    await loadAll();
  });
}

async function scrapeFilteredItems() {
  const targets = filteredItems.value;
  if (!targets.length) {
    errorMessage.value = '当前列表没有可刮削的影片';
    return;
  }
  await runBusy(`正在刮削 ${targets.length} 个条目`, async () => {
    for (const item of targets) {
      busyMessage.value = `正在刮削：${displayTitle(item)}`;
      await apiClient.video.scrape({ itemId: item.id });
    }
    await loadAll();
  });
}

async function runBusy(message: string, action: () => Promise<void>) {
  busyMessage.value = message;
  errorMessage.value = '';
  try {
    await action();
  } catch (error) {
    errorMessage.value = friendlyErrorMessage(error);
  } finally {
    busyMessage.value = '';
  }
}

function friendlyErrorMessage(error: unknown) {
  const message = error instanceof Error ? error.message : '操作失败';
  if (message.includes('xmltv guide has no mapped programs')) {
    return '节目指南没有匹配到直播频道，请确认 XMLTV 的 channel/display-name 与直播源的 tvg-id、tvg-name 或频道名一致';
  }
  if (message.includes('guide source name and url are required')) {
    return '请输入节目指南名称和 XMLTV 地址或路径';
  }
  return message;
}

function selectItem(item: VideoItem) {
  selectedItemId.value = item.id;
}

function switchTab(tab: TabKey) {
  activeTab.value = tab;
  detailItemId.value = '';
  errorMessage.value = '';
}

async function switchLiveView(view: LiveViewKey) {
  liveView.value = view;
  if (view !== 'guide') return;
  await nextTick();
  scrollGuideToNow();
}

function scrollGuideToNow() {
  const shell = epgShellRef.value;
  if (!shell || currentGuideOffset.value < 0) return;
  shell.scrollLeft = Math.max(0, currentGuideOffset.value - 140);
}

function openDetail(item: VideoItem) {
  selectedItemId.value = item.id;
  detailItemId.value = item.id;
  if (activeTab.value === 'home') activeTab.value = 'movies';
}

function closeDetail() {
  detailItemId.value = '';
}

function playItem(item?: VideoItem) {
  const target = item ?? selectedItem.value;
  if (!target) return;
  selectedItemId.value = target.id;
  playbackTitle.value = target.title;
  playbackSrc.value = apiClient.video.streamUrl(target.id);
  playbackPoster.value = apiClient.video.posterUrl(target.id);
  playbackSubtitle.value = target.subtitleUrl ? apiClient.video.subtitleUrl(target.id) : '';
  playbackError.value = '';
}

function playLive(channel: LiveChannel) {
  playbackTitle.value = channel.name;
  playbackSrc.value = channel.url;
  playbackPoster.value = channel.logo || '';
  playbackSubtitle.value = '';
  playbackError.value = '';
}

function openProgramDetail(program: LiveProgram) {
  if (epgSuppressProgramClick) return;
  selectedProgram.value = program;
}

function closeProgramDetail() {
  selectedProgram.value = undefined;
}

function closePlayer() {
  playbackTitle.value = '';
  playbackSrc.value = '';
  playbackPoster.value = '';
  playbackSubtitle.value = '';
  playbackError.value = '';
}

function posterUrl(item: VideoItem) {
  return apiClient.video.posterUrl(item.id);
}

function durationLabel(seconds?: number) {
  if (!seconds) return '--:--';
  const hour = Math.floor(seconds / 3600);
  const min = Math.floor((seconds % 3600) / 60);
  const sec = seconds % 60;
  if (hour > 0) return `${hour}:${String(min).padStart(2, '0')}:${String(sec).padStart(2, '0')}`;
  return `${min}:${String(sec).padStart(2, '0')}`;
}

function typeLabel(type: string) {
  if (type === 'movie') return '电影';
  if (type === 'series') return '电视剧';
  return '混合';
}

function taskLabel(type: string) {
  if (type === 'scrape') return '刮削';
  if (type === 'subtitle') return '字幕';
  if (type === 'transcode') return '转码';
  return type;
}

function statusLabel(status: string) {
  if (status === 'queued') return '排队中';
  if (status === 'running') return '运行中';
  if (status === 'done') return '已完成';
  if (status === 'failed') return '失败';
  if (status === 'scheduled') return '已预约';
  if (status === 'recording') return '录制中';
  if (status === 'completed') return '已完成';
  if (status === 'cancelled') return '已取消';
  return status;
}

function liveTimeRange(startAt: string, endAt: string) {
  const start = new Date(startAt);
  const end = new Date(endAt);
  return `${start.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })} - ${end.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit' })}`;
}

function programDuration(program: LiveProgram) {
  return durationLabel(program.durationSeconds);
}

function programScheduled(program: LiveProgram) {
  return scheduledProgramIds.value.has(program.id);
}

function programStatus(program: LiveProgram) {
  const now = Date.now();
  const start = new Date(program.startAt).getTime();
  const end = new Date(program.endAt).getTime();
  if (now >= start && now < end) return 'current';
  if (now < start) return 'future';
  return 'past';
}

function programActionLabel(program: LiveProgram) {
  if (programScheduled(program)) return '已预约';
  if (programStatus(program) === 'current') return '立即录制';
  if (programStatus(program) === 'future') return '预约录制';
  return '已结束';
}

function programRecordDisabled(program: LiveProgram) {
  return programScheduled(program) || programStatus(program) === 'past';
}

function programOverlapsWindow(program: LiveProgram) {
  const start = new Date(program.startAt).getTime();
  const end = new Date(program.endAt).getTime();
  return end > guideWindowStart.value.getTime() && start < guideWindowEnd.value.getTime();
}

function programBlockStyle(program: LiveProgram) {
  const windowStart = guideWindowStart.value.getTime();
  const windowEnd = guideWindowEnd.value.getTime();
  const start = Math.max(new Date(program.startAt).getTime(), windowStart);
  const end = Math.min(new Date(program.endAt).getTime(), windowEnd);
  const left = Math.max(0, ((start - windowStart) / 60000) * guidePixelsPerMinute);
  const width = Math.max(18, ((end - start) / 60000) * guidePixelsPerMinute - 4);
  return {
    left: `${left}px`,
    width: `${width}px`,
  };
}

function guideDateLabel() {
  const start = guideWindowStart.value;
  return `${start.toLocaleDateString('zh-CN', { month: '2-digit', day: '2-digit', weekday: 'short' })} 全天节目`;
}

function startEpgDrag(event: PointerEvent) {
  if (event.pointerType === 'mouse' && event.button !== 0) return;
  if (!epgShellRef.value) return;
  epgPointerId = event.pointerId;
  epgDragging.value = true;
  epgSuppressProgramClick = false;
  epgDragStartX = event.clientX;
  epgDragStartY = event.clientY;
  epgDragStartScrollLeft = epgShellRef.value.scrollLeft;
  epgShellRef.value.setPointerCapture(event.pointerId);
}

function moveEpgDrag(event: PointerEvent) {
  if (!epgDragging.value || event.pointerId !== epgPointerId || !epgShellRef.value) return;
  const deltaX = event.clientX - epgDragStartX;
  const deltaY = event.clientY - epgDragStartY;
  if (Math.abs(deltaX) > 4 && Math.abs(deltaX) > Math.abs(deltaY)) {
    epgSuppressProgramClick = true;
    event.preventDefault();
  }
  epgShellRef.value.scrollLeft = epgDragStartScrollLeft - deltaX;
}

function endEpgDrag(event: PointerEvent) {
  if (event.pointerId !== epgPointerId) return;
  if (epgShellRef.value?.hasPointerCapture(event.pointerId)) {
    epgShellRef.value.releasePointerCapture(event.pointerId);
  }
  epgDragging.value = false;
  epgPointerId = -1;
  if (epgSuppressProgramClick) {
    window.setTimeout(() => {
      epgSuppressProgramClick = false;
    }, 0);
  }
}

function itemMeta(item: VideoItem) {
  return [item.resolution, item.codec, item.container].filter(Boolean).join(' · ');
}

function displayTitle(item: VideoItem) {
  if (item.kind === 'episode' && item.seriesTitle) return item.seriesTitle;
  return item.title;
}

function displaySubtitle(item: VideoItem) {
  const parts = [];
  if (item.originalTitle && item.originalTitle !== item.title) parts.push(item.originalTitle);
  if (item.year) parts.push(item.year);
  if (item.contentRating) parts.push(item.contentRating);
  if (item.rating && item.rating !== 'N/A') parts.push(`★ ${item.rating}`);
  if (item.durationSeconds) parts.push(durationLabel(item.durationSeconds));
  return parts.join(' · ') || item.libraryName;
}

function cardSubtitle(item: VideoItem) {
  const episode = episodeLabel(item);
  return [episode, item.year || item.libraryName, itemMeta(item)].filter(Boolean).join(' · ');
}

function episodeLabel(item: VideoItem) {
  if (item.kind !== 'episode') return '';
  const season = item.season ? `S${String(item.season).padStart(2, '0')}` : '';
  const episode = item.episode ? `E${String(item.episode).padStart(2, '0')}` : '';
  return [season, episode].filter(Boolean).join('');
}

function detailOverview(item: VideoItem) {
  if (item.kind === 'episode' && item.episodeTitle && item.overview) return item.overview;
  return item.overview || item.tagline || '暂无简介。刮削后会在这里显示剧情、来源、演职员和媒体流信息。';
}

function listLabel(values?: string[], fallback = '暂无') {
  return values?.filter(Boolean).join('、') || fallback;
}

function chips(item: VideoItem) {
  return [
    item.year,
    item.resolution,
    item.codec,
    item.container,
    item.contentRating,
    item.rating && item.rating !== 'N/A' ? `★ ${item.rating}` : '',
  ].filter(Boolean);
}

function trackTitle(track: VideoMediaTrack, fallback: string) {
  return [track.title || fallback, track.language, track.codec, track.channels, track.default ? '默认' : '']
    .filter(Boolean)
    .join(' · ');
}

function allTracks(tracks?: VideoMediaTrack[], fallback = '未检测') {
  if (!tracks?.length) return [fallback];
  return tracks.map((track, index) => trackTitle(track, `轨道 ${index + 1}`));
}

function releaseLabel(item: VideoItem) {
  return item.releaseDate || item.year || '暂无';
}

function scrapedLabel(item: VideoItem) {
  if (!item.scrapedAt) return '尚未手动刮削';
  return new Date(item.scrapedAt).toLocaleString('zh-CN');
}

onMounted(loadAll);
</script>

<template>
  <section class="video-center">
    <aside class="video-sidebar">
      <div class="brand">
        <div class="brand-icon">
          <Film :size="18" />
        </div>
        <div>
          <strong>影视中心</strong>
          <span>家庭 NAS</span>
        </div>
      </div>

      <nav class="nav">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          :class="{ active: activeTab === tab.id }"
          @click="switchTab(tab.id)"
        >
          <component :is="tab.icon" :size="16" />
          <span>{{ tab.label }}</span>
          <small v-if="tab.id === 'movies'">{{ items.length }}</small>
          <small v-if="tab.id === 'live'">{{ liveChannels.length }}</small>
          <small v-if="tab.id === 'tasks'">{{ tasks.length }}</small>
        </button>
      </nav>

      <div class="side-card">
        <strong>媒体库状态</strong>
        <span>{{ libraryStatus }}</span>
        <span v-if="libraries.length">共 {{ libraries.length }} 个库，{{ movieCount }} 部电影，{{ seriesCount }} 集剧集。</span>
      </div>
    </aside>

    <main class="video-main">
      <div class="video-mobile-head">
        <div>
          <strong>影视中心</strong>
          <span>{{ libraryStatus }}</span>
        </div>
        <small>{{ movieCount }} 部电影 · {{ liveChannels.length }} 个频道</small>
      </div>

      <nav class="video-mobile-tabs">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          type="button"
          :class="{ active: activeTab === tab.id }"
          @click="switchTab(tab.id)"
        >
          <component :is="tab.icon" :size="15" />
          <span>{{ tab.label }}</span>
        </button>
      </nav>

      <header class="toolbar">
        <div class="search">
          <Search :size="16" />
          <input v-model="query" placeholder="搜索片名、文件名、分类" />
        </div>
        <button class="ghost" type="button" @click="loadAll">
          <RefreshCcw :size="15" />
          刷新
        </button>
        <button class="primary" type="button" @click="scanLibrary">
          <Sparkles :size="15" />
          扫描
        </button>
        <button class="ghost" type="button" @click="scrapeFilteredItems">
          <Wand2 :size="15" />
          刮削当前列表
        </button>
      </header>

      <div v-if="errorMessage" class="message error">{{ errorMessage }}</div>
      <div v-if="busyMessage" class="message">
        <Loader2 :size="15" class="spin" />
        {{ busyMessage }}
      </div>
      <div v-else-if="statusMessage" class="message muted">{{ statusMessage }}</div>

      <section v-if="activeTab === 'home'" class="home-view">
        <div class="library-strip">
          <article v-for="libraryItem in libraries" :key="libraryItem.id" class="library-card" @click="libraryFilter = libraryItem.id; activeTab = 'movies'">
            <FolderOpen :size="20" />
            <div>
              <strong>{{ libraryItem.name }}</strong>
              <span>{{ typeLabel(libraryItem.type) }} · {{ libraryItem.count }} 个文件</span>
            </div>
          </article>
          <article v-if="!libraries.length" class="library-card empty" @click="activeTab = 'settings'">
            <Plus :size="20" />
            <div>
              <strong>创建媒体库</strong>
              <span>添加本机路径后开始扫描。</span>
            </div>
          </article>
        </div>

        <div class="section-head">
          <h3>最近索引</h3>
          <button type="button" @click="activeTab = 'movies'">查看全部</button>
        </div>
        <div class="poster-row media-poster-grid">
          <article v-for="item in items.slice(0, 10)" :key="item.id" class="poster-card media-poster-card" @click="openDetail(item)">
            <img :src="posterUrl(item)" :alt="item.title" />
            <span>{{ item.rating || item.metadataSource || '本地' }}</span>
            <button class="poster-play" type="button" @click.stop="playItem(item)" aria-label="播放">
              <Play :size="22" />
            </button>
            <strong>{{ displayTitle(item) }}</strong>
            <small>{{ cardSubtitle(item) }}</small>
          </article>
          <div v-if="!items.length" class="empty-state">还没有媒体文件。请先创建媒体库并扫描。</div>
        </div>

        <div class="task-panel">
          <div>
            <strong>进行中的任务</strong>
            <span>{{ runningTasks.length ? `${runningTasks.length} 个任务` : '当前没有运行任务' }}</span>
          </div>
          <button type="button" @click="activeTab = 'tasks'">任务列表</button>
        </div>
      </section>

      <section v-else-if="activeTab === 'movies'" class="movies-view">
        <div class="filters">
          <select v-model="libraryFilter">
            <option value="">全部媒体库</option>
            <option v-for="libraryItem in libraries" :key="libraryItem.id" :value="libraryItem.id">{{ libraryItem.name }}</option>
          </select>
          <select v-model="kindFilter">
            <option value="">全部分类</option>
            <option value="movie">电影</option>
            <option value="episode">电视剧</option>
          </select>
          <select v-model="transcodeProfile">
            <option>1080p H.264</option>
            <option>720p H.264</option>
            <option>原画质封装</option>
          </select>
        </div>

        <section v-if="detailItem" class="detail-view">
          <button class="back-button" type="button" @click="closeDetail">返回海报墙</button>
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
                <button class="primary" type="button" @click="playItem(detailItem)"><Play :size="15" />播放</button>
                <button type="button" @click="createTask('scrape', detailItem)"><Wand2 :size="15" />刮削</button>
                <button type="button" @click="createTask('subtitle', detailItem)"><Captions :size="15" />字幕</button>
                <button type="button" @click="createTask('transcode', detailItem)"><Gauge :size="15" />转码</button>
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
          <article v-for="item in filteredItems" :key="item.id" class="poster-card media-poster-card" @click="openDetail(item)">
            <img :src="posterUrl(item)" :alt="item.title" />
            <span>{{ item.rating || item.metadataSource || '本地' }}</span>
            <button class="poster-play" type="button" @click.stop="playItem(item)" aria-label="播放">
              <Play :size="22" />
            </button>
            <strong>{{ displayTitle(item) }}</strong>
            <small>{{ cardSubtitle(item) }}</small>
          </article>
          <div v-if="!filteredItems.length" class="empty-state">没有匹配的影片。</div>
        </div>
      </section>

      <section v-else-if="activeTab === 'live'" class="live-view">
        <div class="live-tabs">
          <button :class="{ active: liveView === 'channels' }" type="button" @click="switchLiveView('channels')">频道</button>
          <button :class="{ active: liveView === 'guide' }" type="button" @click="switchLiveView('guide')">节目指南</button>
          <button :class="{ active: liveView === 'recordings' }" type="button" @click="switchLiveView('recordings')">录制</button>
        </div>

        <div class="live-layout">
          <div v-if="liveView === 'channels'" class="channel-grid">
            <article v-for="channel in liveChannels" :key="channel.id" class="channel-card" @click="playLive(channel)">
              <img v-if="channel.logo" :src="channel.logo" :alt="channel.name" />
              <Tv v-else :size="22" />
              <strong>{{ channel.name }}</strong>
              <span>{{ channel.group || '直播频道' }}</span>
            </article>
            <div v-if="!liveChannels.length" class="empty-state">还没有直播频道。添加 M3U 源后会显示频道列表。</div>
          </div>

          <div v-else-if="liveView === 'guide'" class="epg-panel">
            <header class="epg-toolbar">
              <div>
                <strong>节目时间线</strong>
                <span>{{ guideDateLabel() }}</span>
              </div>
              <small>{{ guideRows.length }} 个频道 · {{ livePrograms.length }} 个节目</small>
            </header>
            <div
              v-if="guideRows.length"
              ref="epgShellRef"
              class="epg-shell"
              :class="{ dragging: epgDragging }"
              @pointerdown="startEpgDrag"
              @pointermove="moveEpgDrag"
              @pointerup="endEpgDrag"
              @pointercancel="endEpgDrag"
              @pointerleave="endEpgDrag"
            >
              <div class="epg-channel-head">频道</div>
              <div class="epg-time-head" :style="{ width: `${guideWindowMinutes * guidePixelsPerMinute}px` }">
                <span v-for="slot in guideTimeSlots" :key="slot.minute" :style="{ left: `${slot.minute * guidePixelsPerMinute}px` }">
                  {{ slot.label }}
                </span>
                <i v-if="currentGuideOffset >= 0" class="epg-now-line" :style="{ left: `${currentGuideOffset}px` }" />
              </div>
              <template v-for="row in guideRows" :key="row.id">
                <div class="epg-channel-cell">
                  <img v-if="row.logo" :src="row.logo" :alt="row.name" />
                  <Tv v-else :size="18" />
                  <div>
                    <strong>{{ row.name }}</strong>
                    <span>{{ row.group || '直播频道' }}</span>
                  </div>
                </div>
                <div class="epg-program-row" :style="{ width: `${guideWindowMinutes * guidePixelsPerMinute}px` }">
                  <button
                    v-for="program in row.programs"
                    :key="program.id"
                    class="epg-program"
                    :class="{ current: programStatus(program) === 'current', scheduled: programScheduled(program), past: programStatus(program) === 'past' }"
                    type="button"
                    :style="programBlockStyle(program)"
                    @click="openProgramDetail(program)"
                  >
                    <strong>{{ program.title }}</strong>
                    <span>{{ liveTimeRange(program.startAt, program.endAt) }}</span>
                  </button>
                  <i v-if="currentGuideOffset >= 0" class="epg-now-line row" :style="{ left: `${currentGuideOffset}px` }" />
                </div>
              </template>
            </div>
            <div v-else class="empty-state">还没有节目指南。导入 XMLTV 后会显示节目表。</div>
          </div>

          <div v-else-if="liveView === 'recordings'" class="guide-list">
            <article v-for="timer in recordingTimers" :key="timer.id" class="guide-row">
              <div>
                <strong>{{ timer.name }}</strong>
                <span>{{ timer.channelName }} · {{ liveTimeRange(timer.startAt, timer.endAt) }} · {{ statusLabel(timer.status) }}</span>
                <small>{{ timer.targetPath || dvrSettings.recordingPath }}</small>
              </div>
              <button v-if="timer.status === 'scheduled'" type="button" @click="cancelRecording(timer)">取消</button>
            </article>
            <article v-for="recording in recordings" :key="recording.id" class="guide-row muted">
              <div>
                <strong>{{ recording.title }}</strong>
                <span>{{ statusLabel(recording.status) }}</span>
                <small>{{ recording.path || recording.message || '等待录制任务执行' }}</small>
              </div>
            </article>
            <div v-if="!recordingTimers.length && !recordings.length" class="empty-state">暂无录制预约。</div>
          </div>
        </div>
      </section>

      <section v-else-if="activeTab === 'tasks'" class="tasks-view">
        <article v-for="task in tasks" :key="task.id" class="task-row">
          <div class="task-copy">
            <strong>{{ taskLabel(task.type) }} · {{ task.title }}</strong>
            <span>{{ task.message }}</span>
          </div>
          <div class="task-progress">
            <span>{{ statusLabel(task.status) }}</span>
            <div><i :style="{ width: `${task.progress}%` }"></i></div>
            <small>{{ task.progress }}%</small>
          </div>
        </article>
        <div v-if="!tasks.length" class="empty-state">暂无任务。</div>
      </section>

      <section v-else class="settings-view">
        <section class="settings-section">
          <header>
            <h3>媒体库</h3>
            <span>{{ libraries.length }} 个库</span>
          </header>
          <div class="form-grid library-form">
            <label>媒体库名称<input v-model="newLibrary.name" placeholder="电影" /></label>
            <label>目录路径<input v-model="newLibrary.path" placeholder="/volume1/media/movies" /></label>
            <label>类型
              <select v-model="newLibrary.type">
                <option value="movie">电影</option>
                <option value="series">电视剧</option>
                <option value="mixed">混合</option>
              </select>
            </label>
            <label>元数据语言<input v-model="newLibrary.metadataLanguage" /></label>
            <label>字幕语言<input v-model="newLibrary.subtitleLanguage" /></label>
            <label class="check"><input v-model="newLibrary.autoSubtitles" type="checkbox" /> 自动关联字幕</label>
            <label class="check"><input v-model="newLibrary.allowAdultContent" type="checkbox" /> 允许成人内容元数据</label>
            <button class="primary" type="button" @click="createLibrary"><Plus :size="15" />创建</button>
          </div>
          <div class="settings-list">
            <article v-for="libraryItem in libraries" :key="libraryItem.id">
              <FolderOpen :size="17" />
              <div>
                <strong>{{ libraryItem.name }}</strong>
                <span>{{ typeLabel(libraryItem.type) }} · {{ libraryItem.paths.join('，') }}</span>
              </div>
              <small>{{ libraryItem.count }} 个文件</small>
              <button class="danger action-button" type="button" @click.stop="deleteLibrary(libraryItem)">
                <Trash2 :size="14" />
                删除
              </button>
            </article>
            <div v-if="!libraries.length" class="empty-state">还没有媒体库。</div>
          </div>
        </section>

        <section class="settings-section">
          <header>
            <h3>电视直播源</h3>
            <span>{{ liveChannels.length }} 个频道</span>
          </header>
          <div class="form-grid">
            <label>名称<input v-model="newLiveSource.name" placeholder="家庭直播" /></label>
            <label>M3U 地址或路径<input v-model="newLiveSource.url" placeholder="/volume1/media/live.m3u 或 https://..." /></label>
            <label>User-Agent<input v-model="newLiveSource.userAgent" placeholder="需要鉴权时填写" /></label>
            <label>并发限制<input v-model.number="newLiveSource.streamLimit" type="number" min="0" /></label>
            <button class="primary" type="button" @click="createLiveSource"><Plus :size="15" />添加</button>
          </div>
          <div class="settings-list compact">
            <article v-for="source in liveSources" :key="source.id">
              <Radio :size="16" />
              <div>
                <strong>{{ source.name }}</strong>
                <span>{{ source.url }}</span>
              </div>
              <small>{{ source.channelCount }} 个频道</small>
            </article>
            <div v-if="!liveSources.length" class="empty-state">还没有直播源。</div>
          </div>
        </section>

        <section class="settings-section">
          <header>
            <h3>节目指南</h3>
            <span>{{ livePrograms.length }} 个节目</span>
          </header>
          <div class="form-grid">
            <label>名称<input v-model="newGuideSource.name" placeholder="XMLTV" /></label>
            <label>XMLTV 地址或路径<input v-model="newGuideSource.url" placeholder="/volume1/media/guide.xml 或 https://..." /></label>
            <label>User-Agent<input v-model="newGuideSource.userAgent" placeholder="可选" /></label>
            <button class="primary" type="button" @click="createGuideSource"><Plus :size="15" />导入</button>
          </div>
          <div class="settings-list compact">
            <article v-for="guide in guideSources" :key="guide.id">
              <ListVideo :size="16" />
              <div>
                <strong>{{ guide.name }}</strong>
                <span>{{ guide.url }}</span>
              </div>
              <small>{{ guide.programCount }} 个节目</small>
            </article>
            <div v-if="!guideSources.length" class="empty-state">还没有节目指南源。</div>
          </div>
        </section>

        <section class="settings-section dvr-settings-card">
          <header>
            <h3>数字录像机</h3>
            <span>{{ recordingTimers.length }} 个预约</span>
          </header>
          <div class="dvr-settings-grid">
            <div class="dvr-field wide">
              <label>录制目录</label>
              <input v-model="dvrSettings.recordingPath" />
            </div>
            <div class="dvr-field">
              <label>电影目录</label>
              <input v-model="dvrSettings.movieRecordingPath" placeholder="可选" />
            </div>
            <div class="dvr-field">
              <label>剧集目录</label>
              <input v-model="dvrSettings.seriesRecordingPath" placeholder="可选" />
            </div>
            <div class="dvr-field compact">
              <label>提前秒数</label>
              <input v-model.number="dvrSettings.prePaddingSeconds" type="number" min="0" />
            </div>
            <div class="dvr-field compact">
              <label>延后秒数</label>
              <input v-model.number="dvrSettings.postPaddingSeconds" type="number" min="0" />
            </div>
            <div class="dvr-field compact">
              <label>最大并发</label>
              <input v-model.number="dvrSettings.maxConcurrentRecord" type="number" min="1" />
            </div>
            <div class="dvr-field wide">
              <label>后处理命令</label>
              <input v-model="dvrSettings.postProcessCommand" placeholder="/usr/local/bin/post_record.sh {path}" />
            </div>
          </div>
          <div class="dvr-actions">
            <label><input v-model="dvrSettings.saveNfo" type="checkbox" />保存 NFO</label>
            <label><input v-model="dvrSettings.saveImages" type="checkbox" />保存图片</label>
            <button class="primary" type="button" @click="saveDvrSettings"><Settings :size="15" />保存</button>
          </div>
        </section>
      </section>
    </main>

    <section v-if="selectedProgram" class="program-detail-overlay" @click.self="closeProgramDetail">
      <article class="program-detail-dialog">
        <header>
          <button type="button" @click="closeProgramDetail"><X :size="16" /></button>
          <div>
            <strong>{{ selectedProgram.title }}</strong>
            <span>{{ selectedProgram.channelName }} · {{ liveTimeRange(selectedProgram.startAt, selectedProgram.endAt) }}</span>
          </div>
        </header>
        <div class="program-detail-body">
          <p>{{ selectedProgram.overview || listLabel(selectedProgram.categories, '暂无节目简介') }}</p>
          <dl>
            <div><dt>频道</dt><dd>{{ selectedProgram.channelName }}</dd></div>
            <div><dt>时长</dt><dd>{{ programDuration(selectedProgram) }}</dd></div>
            <div><dt>状态</dt><dd>{{ programStatus(selectedProgram) === 'current' ? '正在播放' : programStatus(selectedProgram) === 'future' ? '未开始' : '已结束' }}</dd></div>
            <div><dt>分类</dt><dd>{{ listLabel(selectedProgram.categories) }}</dd></div>
          </dl>
        </div>
        <footer>
          <button
            class="primary"
            type="button"
            :disabled="programRecordDisabled(selectedProgram)"
            @click="recordProgram(selectedProgram)"
          >
            <Radio :size="15" />
            {{ programActionLabel(selectedProgram) }}
          </button>
        </footer>
      </article>
    </section>

    <section v-if="playbackSrc" class="player-overlay">
      <div class="player-top">
        <button type="button" @click="closePlayer"><X :size="18" /></button>
        <strong>{{ playbackTitle }}</strong>
        <span>{{ playbackError }}</span>
      </div>
      <video
        :key="playbackSrc"
        controls
        autoplay
        playsinline
        :src="playbackSrc"
        :poster="playbackPoster"
        @error="playbackError = '播放失败，请确认浏览器支持该格式或使用转码任务生成兼容版本。'"
      >
        <track v-if="playbackSubtitle" kind="subtitles" srclang="zh" label="中文" :src="playbackSubtitle" default />
      </video>
    </section>
  </section>
</template>

<style scoped>
.video-center {
  --video-blue: #2f7cff;
  --video-bg: rgba(246, 250, 255, 0.88);
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr);
  min-height: 100%;
  height: 100%;
  background:
    radial-gradient(circle at 72% 10%, rgba(91, 109, 255, 0.13), transparent 34%),
    linear-gradient(135deg, rgba(233, 244, 255, 0.95), rgba(249, 252, 255, 0.94));
  color: #142033;
  font-size: 12px;
  overflow: hidden;
}

.video-sidebar {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 18px 14px;
  border-right: 1px solid rgba(139, 164, 190, 0.18);
  background: rgba(255, 255, 255, 0.45);
}

.brand {
  display: flex;
  gap: 10px;
  align-items: center;
}

.brand-icon {
  display: grid;
  place-items: center;
  width: 38px;
  height: 38px;
  color: white;
  border-radius: 12px;
  background: linear-gradient(135deg, #1f8cff, #715cff);
}

.brand strong,
.section-head h3,
.form-card h3 {
  display: block;
  margin: 0;
  font-size: 14px;
}

.brand span,
.side-card span,
.media-card span,
.media-card small,
.detail-panel span,
.detail-panel p,
.library-card span,
.message,
.channel-card span,
.task-row span,
.library-list span {
  font-size: 12px;
  color: #6b7c93;
}

.nav {
  display: grid;
  gap: 6px;
}

button {
  border: 0;
  background: transparent;
  color: inherit;
  font: inherit;
  cursor: pointer;
}

.nav button {
  display: flex;
  align-items: center;
  gap: 9px;
  min-height: 38px;
  padding: 0 10px;
  border-radius: 10px;
  font-size: 12px;
  text-align: left;
}

.nav button.active {
  color: var(--video-blue);
  background: rgba(47, 124, 255, 0.12);
}

.nav small {
  margin-left: auto;
  color: #7b8da3;
}

.side-card,
.form-card,
.task-panel,
.detail-panel,
.item-grid,
.library-list,
.settings-section,
.player-overlay,
.message {
  border: 1px solid rgba(148, 170, 194, 0.2);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.7);
  box-shadow: 0 14px 38px rgba(53, 76, 105, 0.08);
}

.side-card {
  display: grid;
  gap: 6px;
  margin-top: auto;
  padding: 12px;
}

.video-main {
  min-width: 0;
  overflow: auto;
  padding: 18px;
}

.video-mobile-head,
.video-mobile-tabs {
  display: none;
}

.toolbar,
.filters,
.section-head,
.form-grid {
  display: flex;
  gap: 10px;
  align-items: center;
}

.toolbar {
  margin-bottom: 12px;
}

.search {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  min-width: 180px;
  height: 40px;
  padding: 0 12px;
  border: 1px solid rgba(148, 170, 194, 0.22);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.82);
}

input,
select {
  min-width: 0;
  border: 1px solid rgba(148, 170, 194, 0.24);
  border-radius: 10px;
  background: rgba(255, 255, 255, 0.76);
  color: #142033;
  font-size: 12px;
  outline: none;
}

.search input {
  flex: 1;
  border: 0;
  background: transparent;
}

.ghost,
.primary,
.action-button,
.section-head button,
.detail-actions button,
.task-panel button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 36px;
  padding: 0 13px;
  border: 1px solid rgba(47, 124, 255, 0.22);
  border-radius: 12px;
  color: var(--video-blue);
  background: rgba(255, 255, 255, 0.72);
  font-size: 12px;
  white-space: nowrap;
}

.primary {
  color: white;
  background: linear-gradient(135deg, #1888ff, #4c66ff);
}

.action-button.danger {
  border-color: rgba(239, 68, 68, 0.22);
  color: #dc2626;
  background: rgba(255, 241, 242, 0.82);
}

.message {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 12px;
  padding: 9px 12px;
}

.message.error {
  color: #dc2626;
  border-color: rgba(239, 68, 68, 0.24);
  background: rgba(255, 241, 242, 0.8);
}

.spin {
  animation: spin 0.9s linear infinite;
}

@keyframes spin {
  to { transform: rotate(360deg); }
}

.home-view,
.movies-view,
.live-view,
.tasks-view,
.library-view,
.settings-view {
  display: grid;
  gap: 14px;
}

.settings-view {
  grid-template-columns: minmax(0, 1fr);
  align-items: start;
}

.settings-section {
  display: grid;
  gap: 12px;
  min-width: 0;
  padding: 14px;
}

.settings-section header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.settings-section h3 {
  margin: 0;
  font-size: 14px;
}

.settings-section header span {
  color: #7b8da3;
  font-size: 11px;
}

.settings-section .form-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  align-items: end;
}

.settings-section .form-grid label {
  min-width: 0;
}

.settings-section .form-grid .check {
  align-self: center;
}

.library-strip,
.channel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(160px, 1fr));
  gap: 12px;
}

.poster-row {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(118px, 150px));
  align-items: start;
  justify-content: start;
  gap: 14px;
}

.media-poster-grid {
  grid-template-columns: repeat(auto-fill, minmax(150px, 190px));
  gap: 22px;
}

.library-card {
  display: flex;
  gap: 10px;
  align-items: center;
  min-height: 84px;
  padding: 14px;
  border: 1px solid rgba(148, 170, 194, 0.2);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.7);
  cursor: pointer;
}

.poster-card {
  position: relative;
  display: grid;
  gap: 7px;
  width: min(150px, 100%);
  min-width: 0;
  cursor: pointer;
}

.media-poster-card {
  width: 100%;
}

.poster-card img {
  width: 100%;
  aspect-ratio: 2 / 3;
  object-fit: cover;
  border-radius: 12px;
  background: #e8f2ff;
  box-shadow: 0 14px 28px rgba(30, 64, 112, 0.12);
  transition: transform 0.16s ease, box-shadow 0.16s ease;
}

.poster-card:hover img {
  transform: translateY(-2px);
  box-shadow: 0 18px 34px rgba(30, 64, 112, 0.18);
}

.poster-card span {
  position: absolute;
  top: 7px;
  left: 7px;
  padding: 2px 6px;
  color: #fbbf24;
  border-radius: 8px;
  background: rgba(15, 23, 42, 0.76);
}

.poster-card strong,
.poster-card small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.poster-play {
  position: absolute;
  top: 42%;
  left: 50%;
  display: grid;
  place-items: center;
  width: 52px;
  height: 52px;
  color: white;
  border-radius: 999px;
  background: rgba(47, 124, 255, 0.92);
  box-shadow: 0 12px 28px rgba(47, 124, 255, 0.34);
  opacity: 0;
  transform: translate(-50%, -50%) scale(0.92);
  transition: opacity 0.16s ease, transform 0.16s ease;
}

.poster-card:hover .poster-play,
.poster-play:focus-visible {
  opacity: 1;
  transform: translate(-50%, -50%) scale(1);
}

.section-head {
  justify-content: space-between;
}

.task-panel {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 14px;
}

.filters {
  flex-wrap: wrap;
}

.filters select {
  min-height: 36px;
  padding: 0 10px;
}

.content-grid {
  display: grid;
  grid-template-columns: minmax(360px, 1fr) clamp(320px, 27vw, 430px);
  gap: 14px;
}

.detail-view {
  display: grid;
  gap: 12px;
  align-content: start;
}

.back-button {
  justify-self: start;
  min-height: 34px;
  padding: 0 12px;
  color: var(--video-blue);
  border: 1px solid rgba(47, 124, 255, 0.22);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.72);
  font-size: 12px;
}

.detail-view-hero {
  display: grid;
  grid-template-columns: minmax(150px, 190px) minmax(0, 1fr);
  gap: 18px;
  align-items: start;
}

.detail-poster.large {
  border-radius: 12px;
}

.detail-view-copy {
  display: grid;
  align-content: start;
  gap: 8px;
  min-width: 0;
}

.detail-view-copy h3 {
  margin: 0;
  font-size: 20px;
  line-height: 1.18;
}

.detail-view-copy em,
.detail-subtitle {
  margin: 0;
  color: #6b7c93;
  font-size: 12px;
  font-style: normal;
}

.overview-section p {
  max-width: 980px;
}

.detail-info-grid {
  display: grid;
  grid-template-columns: minmax(250px, 0.72fr) minmax(360px, 1.28fr);
  gap: 12px;
  align-items: start;
}

.item-grid {
  display: grid;
  align-content: start;
  max-height: min(520px, calc(100vh - 260px));
  min-height: 280px;
  overflow: auto;
  padding: 12px;
}

.media-card {
  display: grid;
  grid-template-columns: 54px minmax(0, 1fr) 34px;
  gap: 10px;
  align-items: center;
  min-height: 78px;
  padding: 9px;
  border: 1px solid transparent;
  border-radius: 12px;
}

.media-card.active {
  border-color: rgba(47, 124, 255, 0.35);
  background: rgba(47, 124, 255, 0.1);
}

.media-card img {
  width: 54px;
  height: 68px;
  object-fit: cover;
  border-radius: 8px;
}

.card-body {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.card-body > * {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.media-card button {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  color: var(--video-blue);
  border-radius: 999px;
  background: rgba(47, 124, 255, 0.1);
}

.detail-panel {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
  padding: 14px;
}

.detail-hero {
  display: grid;
  grid-template-columns: minmax(104px, 38%) minmax(0, 1fr);
  gap: 12px;
  align-items: start;
}

.detail-poster {
  width: 100%;
  aspect-ratio: 2 / 3;
  object-fit: cover;
  border-radius: 16px;
  background: #e8f2ff;
  box-shadow: 0 18px 34px rgba(30, 64, 112, 0.12);
}

.detail-identity {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.detail-identity strong {
  font-size: 16px;
  line-height: 1.25;
}

.detail-identity em {
  color: #2f7cff;
  font-size: 12px;
  font-style: normal;
  font-weight: 700;
}

.source-line {
  color: #7b8da3;
  font-size: 11px;
  overflow-wrap: anywhere;
}

.episode-badge {
  width: max-content;
  padding: 2px 7px;
  color: #2f7cff;
  border-radius: 999px;
  background: rgba(47, 124, 255, 0.12);
  font-size: 11px;
  font-weight: 700;
}

.detail-chip-row,
.tag-field dd {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.detail-chip-row i,
.tag-field span {
  max-width: 100%;
  padding: 2px 7px;
  border: 1px solid rgba(148, 170, 194, 0.24);
  border-radius: 999px;
  background: rgba(246, 250, 255, 0.82);
  color: #60728a;
  font-size: 10px;
  font-style: normal;
  line-height: 1.25;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-section {
  display: grid;
  gap: 8px;
  min-width: 0;
  padding: 11px;
  border: 1px solid rgba(148, 170, 194, 0.18);
  border-radius: 12px;
  background: rgba(248, 251, 255, 0.58);
}

.detail-section h4 {
  margin: 0;
  font-size: 13px;
  line-height: 1.25;
}

.detail-section p {
  margin: 0;
  color: #41546d;
  font-size: 12px;
  line-height: 1.58;
  overflow-wrap: anywhere;
}

.stream-section {
  gap: 7px;
}

.stream-row {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-height: 28px;
}

.stream-row span {
  color: #7b8da3;
  font-size: 11px;
}

.stream-row strong {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 12px;
}

.detail-panel p {
  margin: 0;
  line-height: 1.6;
  overflow-wrap: anywhere;
}

.detail-panel dl,
.detail-meta-grid {
  display: grid;
  grid-template-columns: repeat(2, 1fr);
  gap: 9px 12px;
  margin: 0;
}

.detail-meta-grid {
  grid-template-columns: 1fr;
}

.detail-meta-grid.wide {
  grid-template-columns: repeat(3, minmax(0, 1fr));
}

.detail-meta-grid .full {
  grid-column: 1 / -1;
}

.detail-panel dt,
.detail-section dt {
  font-size: 11px;
  color: #7b8da3;
  line-height: 1.25;
}

.detail-panel dd,
.detail-section dd {
  margin: 3px 0 0;
  font-size: 12px;
  font-weight: 600;
  line-height: 1.45;
  overflow-wrap: anywhere;
}

.detail-section dd {
  color: #18263a;
}

.detail-meta-grid.wide dd {
  max-height: 4.5em;
  overflow: auto;
}

.detail-meta-grid.wide .full dd {
  max-height: none;
}

.tag-field dd {
  max-height: 58px;
  overflow: auto;
  padding-right: 2px;
}

.detail-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.form-card {
  padding: 14px;
}

.form-grid {
  flex-wrap: wrap;
  align-items: flex-end;
  margin-top: 12px;
}

.form-grid label {
  display: grid;
  gap: 6px;
  min-width: 180px;
  flex: 1;
  font-size: 12px;
  color: #6b7c93;
}

.form-grid input,
.form-grid select {
  min-height: 36px;
  padding: 0 10px;
}

.form-grid .check {
  display: flex;
  align-items: center;
  min-height: 36px;
  flex: 0 0 auto;
}

.form-grid .check input {
  min-height: 0;
}

.live-layout {
  display: grid;
  grid-template-columns: minmax(0, 1fr);
  gap: 14px;
}

.live-tabs {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.live-tabs button {
  min-height: 32px;
  padding: 0 12px;
  border: 1px solid rgba(47, 124, 255, 0.18);
  border-radius: 999px;
  color: #60728a;
  background: rgba(255, 255, 255, 0.68);
  font-size: 12px;
}

.live-tabs button.active {
  color: var(--video-blue);
  background: rgba(47, 124, 255, 0.12);
}

.channel-card {
  display: grid;
  gap: 8px;
  min-height: 112px;
  padding: 12px;
  border: 1px solid rgba(148, 170, 194, 0.22);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.72);
  cursor: pointer;
}

.channel-card img {
  width: 76px;
  aspect-ratio: 2 / 1;
  height: auto;
  object-fit: contain;
  border-radius: 0;
  background: rgba(15, 23, 42, 0.03);
}

.guide-list {
  display: grid;
  align-content: start;
  gap: 8px;
  min-width: 0;
}

.guide-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  min-height: 58px;
  padding: 10px 12px;
  border: 1px solid rgba(148, 170, 194, 0.18);
  border-radius: 12px;
  background: rgba(255, 255, 255, 0.68);
}

.guide-row.muted {
  opacity: 0.74;
}

.guide-row > div {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.guide-row strong,
.guide-row span,
.guide-row small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.guide-row strong {
  font-size: 13px;
}

.guide-row span,
.guide-row small {
  color: #6b7c93;
  font-size: 11px;
}

.guide-row button {
  min-height: 30px;
  padding: 0 10px;
  border: 1px solid rgba(47, 124, 255, 0.22);
  border-radius: 10px;
  color: var(--video-blue);
  background: rgba(255, 255, 255, 0.76);
  font-size: 12px;
}

.guide-row button:disabled {
  color: #7b8da3;
  cursor: default;
  background: rgba(148, 170, 194, 0.12);
}

.epg-panel {
  display: grid;
  gap: 10px;
  min-width: 0;
}

.epg-toolbar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 14px;
  padding: 0 2px;
}

.epg-toolbar div {
  display: grid;
  gap: 2px;
}

.epg-toolbar strong {
  font-size: 14px;
}

.epg-toolbar span,
.epg-toolbar small {
  color: #6f8199;
  font-size: 11px;
}

.epg-shell {
  display: grid;
  grid-template-columns: 168px minmax(0, 1fr);
  max-height: calc(100vh - 270px);
  overflow: auto;
  cursor: grab;
  border: 1px solid rgba(148, 170, 194, 0.2);
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.64);
  touch-action: pan-y;
  user-select: none;
}

.epg-shell.dragging {
  cursor: grabbing;
}

.epg-channel-head,
.epg-time-head {
  position: sticky;
  top: 0;
  z-index: 5;
  height: 42px;
  border-bottom: 1px solid rgba(148, 170, 194, 0.2);
  background: rgba(245, 250, 255, 0.96);
}

.epg-channel-head {
  left: 0;
  z-index: 6;
  display: flex;
  align-items: center;
  padding: 0 12px;
  color: #6f8199;
  font-size: 11px;
  font-weight: 700;
}

.epg-time-head {
  position: sticky;
  min-width: calc(1440 * 3px);
}

.epg-time-head span {
  position: absolute;
  top: 12px;
  color: #4b5f78;
  font-size: 11px;
  transform: translateX(-1px);
}

.epg-channel-cell {
  position: sticky;
  left: 0;
  z-index: 3;
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
  min-height: 56px;
  padding: 8px 10px;
  border-bottom: 1px solid rgba(148, 170, 194, 0.18);
  background: rgba(248, 252, 255, 0.96);
}

.epg-channel-cell img {
  width: 56px;
  aspect-ratio: 2 / 1;
  height: auto;
  object-fit: contain;
  border-radius: 0;
  background: rgba(15, 23, 42, 0.04);
}

.epg-channel-cell div {
  display: grid;
  min-width: 0;
}

.epg-channel-cell strong,
.epg-channel-cell span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.epg-channel-cell strong {
  font-size: 12px;
}

.epg-channel-cell span {
  color: #6f8199;
  font-size: 10px;
}

.epg-program-row {
  position: relative;
  min-width: calc(1440 * 3px);
  min-height: 56px;
  border-bottom: 1px solid rgba(148, 170, 194, 0.18);
  background-image: repeating-linear-gradient(
    to right,
    rgba(148, 170, 194, 0.14) 0,
    rgba(148, 170, 194, 0.14) 1px,
    transparent 1px,
    transparent 90px
  );
}

.epg-program {
  position: absolute;
  top: 7px;
  bottom: 7px;
  display: grid;
  align-content: center;
  gap: 2px;
  min-height: 0;
  padding: 0 9px;
  overflow: hidden;
  text-align: left;
  border: 1px solid rgba(148, 170, 194, 0.2);
  border-radius: 9px;
  color: #18253a;
  background: rgba(255, 255, 255, 0.78);
}

.epg-program:hover {
  border-color: rgba(47, 124, 255, 0.38);
  background: rgba(235, 244, 255, 0.94);
}

.epg-program.current {
  border-color: rgba(34, 197, 94, 0.38);
  background: rgba(236, 253, 245, 0.92);
}

.epg-program.scheduled {
  border-color: rgba(47, 124, 255, 0.42);
  background: rgba(224, 238, 255, 0.92);
}

.epg-program.past {
  color: #7b8da3;
  background: rgba(241, 245, 249, 0.74);
}

.epg-program strong,
.epg-program span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.epg-program strong {
  font-size: 12px;
}

.epg-program span {
  color: #6f8199;
  font-size: 10px;
}

.epg-now-line {
  position: absolute;
  top: 0;
  bottom: 0;
  z-index: 4;
  width: 2px;
  pointer-events: none;
  background: #ef4444;
}

.epg-now-line.row {
  top: 0;
  bottom: 0;
  opacity: 0.72;
}

.program-detail-overlay {
  position: absolute;
  inset: 0;
  z-index: 30;
  display: grid;
  place-items: center;
  padding: 32px;
  background: rgba(15, 23, 42, 0.28);
  backdrop-filter: blur(8px);
}

.program-detail-dialog {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  width: min(720px, 100%);
  max-height: min(620px, 86%);
  overflow: hidden;
  border: 1px solid rgba(148, 170, 194, 0.24);
  border-radius: 16px;
  background: rgba(255, 255, 255, 0.96);
  box-shadow: 0 24px 70px rgba(30, 64, 112, 0.22);
}

.program-detail-dialog header {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 14px;
  border-bottom: 1px solid rgba(148, 170, 194, 0.16);
}

.program-detail-dialog header button {
  display: grid;
  place-items: center;
  width: 32px;
  height: 32px;
  border: 1px solid rgba(148, 170, 194, 0.22);
  border-radius: 10px;
  background: rgba(248, 252, 255, 0.9);
}

.program-detail-dialog header div {
  display: grid;
  gap: 2px;
  min-width: 0;
}

.program-detail-dialog header strong {
  overflow: hidden;
  font-size: 16px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.program-detail-dialog header span {
  color: #6f8199;
  font-size: 12px;
}

.program-detail-body {
  display: grid;
  gap: 14px;
  align-content: start;
  min-height: 0;
  padding: 18px;
  overflow: auto;
}

.program-detail-body p {
  margin: 0;
  color: #334155;
  font-size: 13px;
  line-height: 1.7;
}

.program-detail-body dl {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  margin: 0;
}

.program-detail-body dl div {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.program-detail-body dt {
  color: #7b8da3;
  font-size: 11px;
}

.program-detail-body dd {
  margin: 0;
  overflow: hidden;
  font-size: 13px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.program-detail-dialog footer {
  display: flex;
  justify-content: flex-end;
  padding: 14px;
  border-top: 1px solid rgba(148, 170, 194, 0.16);
}

.program-detail-dialog footer button {
  min-width: 132px;
}

.dvr-settings-card {
  align-content: start;
}

.dvr-settings-grid {
  display: grid;
  grid-template-columns: minmax(220px, 1.15fr) minmax(180px, 0.85fr) minmax(180px, 0.85fr);
  gap: 12px;
  margin-top: 12px;
}

.dvr-field {
  display: grid;
  gap: 6px;
  min-width: 0;
}

.dvr-field.wide {
  grid-column: span 2;
}

.dvr-field.compact {
  max-width: 180px;
}

.dvr-field label,
.dvr-actions label {
  color: #6b7c93;
  font-size: 12px;
}

.dvr-field input {
  width: 100%;
  min-height: 34px;
  padding: 0 10px;
}

.dvr-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px 16px;
  align-items: center;
  justify-content: flex-end;
  margin-top: 14px;
  padding-top: 12px;
  border-top: 1px solid rgba(148, 170, 194, 0.16);
}

.dvr-actions label {
  display: inline-flex;
  gap: 7px;
  align-items: center;
  min-height: 30px;
}

.dvr-actions input {
  width: 14px;
  min-height: 14px;
  margin: 0;
}

.task-row,
.library-list article,
.settings-list article {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 12px;
  border-bottom: 1px solid rgba(148, 170, 194, 0.16);
}

.task-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(180px, 260px);
  align-items: center;
  min-width: 0;
  border-radius: 14px;
  background: rgba(255, 255, 255, 0.62);
}

.task-copy,
.library-list article > div,
.settings-list article > div {
  display: grid;
  gap: 4px;
  min-width: 0;
  flex: 1;
}

.task-copy strong,
.task-copy span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-list article > div span,
.settings-list article > div span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.library-list small,
.settings-list small {
  flex: 0 0 auto;
  color: #6b7c93;
}

.settings-list {
  display: grid;
  border-top: 1px solid rgba(148, 170, 194, 0.14);
}

.settings-list.compact article {
  min-height: 48px;
}

.task-progress {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 6px 10px;
  min-width: 0;
}

.task-progress > span {
  grid-column: 1 / -1;
  justify-self: end;
}

.task-progress div {
  height: 6px;
  overflow: hidden;
  border-radius: 999px;
  background: rgba(148, 170, 194, 0.18);
}

.task-progress i {
  display: block;
  height: 100%;
  border-radius: inherit;
  background: var(--video-blue);
}

.task-progress small {
  color: #6b7c93;
  font-size: 11px;
  line-height: 1;
}

.empty-state {
  display: grid;
  place-items: center;
  min-height: 160px;
  color: #7b8da3;
  font-size: 12px;
  text-align: center;
}

.player-overlay {
  position: absolute;
  inset: 0;
  z-index: 20;
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  padding: 16px;
  background: rgba(7, 10, 18, 0.94);
  backdrop-filter: blur(12px);
}

.player-top {
  display: flex;
  align-items: center;
  gap: 12px;
  min-height: 44px;
  color: white;
}

.player-top button {
  display: grid;
  place-items: center;
  width: 34px;
  height: 34px;
  border-radius: 999px;
  background: rgba(255, 255, 255, 0.12);
}

.player-top span {
  margin-left: auto;
  color: #fca5a5;
  font-size: 12px;
}

.player-overlay video {
  width: 100%;
  height: 100%;
  min-height: 0;
  background: black;
  border-radius: 14px;
}

@container desktop-window-body (max-width: 900px) {
  .video-center {
    grid-template-columns: minmax(0, 1fr);
  }

  .video-sidebar {
    display: none;
  }

  .video-main {
    padding: 16px;
  }

  .video-mobile-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: 12px;
    margin-bottom: 12px;
  }

  .video-mobile-head div {
    display: grid;
    gap: 2px;
    min-width: 0;
  }

  .video-mobile-head strong {
    font-size: 20px;
    line-height: 1.15;
  }

  .video-mobile-head span,
  .video-mobile-head small {
    overflow: hidden;
    color: #6b7c93;
    font-size: 11px;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .video-mobile-tabs {
    display: flex;
    gap: 18px;
    margin-bottom: 12px;
    overflow-x: auto;
    color: #66758d;
    scrollbar-width: none;
  }

  .video-mobile-tabs::-webkit-scrollbar {
    display: none;
  }

  .video-mobile-tabs button {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 5px;
    flex: 0 0 auto;
    padding: 0 0 8px;
    color: inherit;
    font-size: 12px;
    white-space: nowrap;
  }

  .video-mobile-tabs button.active {
    color: var(--video-blue);
    font-weight: 800;
  }

  .video-mobile-tabs button.active::after {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    height: 2px;
    content: "";
    background: var(--video-blue);
    border-radius: 999px;
  }

  .toolbar {
    flex-wrap: wrap;
    gap: 8px;
  }

  .toolbar .search {
    flex: 1 1 100%;
    min-width: 0;
  }

  .toolbar > button {
    flex: 1 1 auto;
    min-width: 92px;
    padding-inline: 9px;
  }

  .content-grid,
  .detail-info-grid,
  .settings-view,
  .live-layout {
    grid-template-columns: minmax(0, 1fr);
  }

  .detail-panel {
    grid-row: 1;
  }

  .detail-view-hero {
    grid-template-columns: minmax(128px, 168px) minmax(0, 1fr);
  }

  .detail-meta-grid.wide,
  .program-detail-body dl {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dvr-settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dvr-field.wide {
    grid-column: 1 / -1;
  }

  .media-poster-grid {
    grid-template-columns: repeat(auto-fill, minmax(132px, 1fr));
    gap: 16px;
  }

  .channel-grid {
    grid-template-columns: repeat(auto-fill, minmax(148px, 1fr));
  }

  .item-grid {
    max-height: none;
  }

  .epg-toolbar {
    align-items: start;
    flex-direction: column;
    gap: 4px;
  }

  .epg-shell {
    grid-template-columns: 132px minmax(0, 1fr);
    max-height: min(620px, calc(100dvh - 250px));
  }

  .epg-channel-cell {
    gap: 7px;
    padding-inline: 8px;
  }

  .epg-channel-cell img {
    width: 46px;
  }

  .program-detail-overlay {
    padding: 18px;
  }

  .program-detail-dialog {
    max-height: 88%;
  }
}

@container desktop-window-body (max-width: 620px) {
  .video-main {
    padding: 12px;
  }

  .video-mobile-head {
    align-items: start;
    flex-direction: column;
    gap: 4px;
  }

  .video-mobile-tabs {
    gap: 16px;
  }

  .toolbar > button {
    flex: 1 1 calc(33.333% - 8px);
    min-width: 0;
  }

  .filters {
    display: grid;
    grid-template-columns: 1fr;
  }

  .library-strip,
  .channel-grid,
  .detail-view-hero,
  .detail-meta-grid.wide,
  .program-detail-body dl,
  .dvr-settings-grid,
  .task-row {
    grid-template-columns: minmax(0, 1fr);
  }

  .detail-poster.large {
    max-width: 170px;
  }

  .detail-actions {
    display: grid;
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .detail-actions button {
    min-width: 0;
  }

  .media-poster-grid,
  .poster-row {
    grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
    gap: 12px;
  }

  .poster-card {
    width: 100%;
  }

  .settings-section .form-grid {
    grid-template-columns: minmax(0, 1fr);
  }

  .settings-list article {
    grid-template-columns: auto minmax(0, 1fr);
  }

  .settings-list article small,
  .settings-list article .action-button {
    grid-column: 2;
    justify-self: start;
  }

  .epg-shell {
    grid-template-columns: 104px minmax(0, 1fr);
    max-height: min(560px, calc(100dvh - 230px));
  }

  .epg-channel-head {
    padding-inline: 8px;
  }

  .epg-channel-cell {
    min-height: 58px;
  }

  .epg-channel-cell img {
    display: none;
  }

  .epg-program-row {
    min-height: 58px;
  }

  .program-detail-overlay {
    padding: 10px;
  }

  .program-detail-dialog header strong {
    font-size: 14px;
  }

  .player-overlay {
    inset: 0;
    padding: 10px;
  }

  .player-top {
    width: 100%;
  }

  .player-overlay video {
    max-height: calc(100% - 58px);
    border-radius: 10px;
  }
}

@media (max-width: 880px) {
  .video-center {
    grid-template-columns: 1fr;
  }

  .video-sidebar {
    display: none;
  }

  .content-grid,
  .detail-info-grid,
  .settings-view,
  .live-layout {
    grid-template-columns: 1fr;
  }

  .detail-meta-grid.wide {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dvr-settings-grid {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }

  .dvr-field.wide {
    grid-column: 1 / -1;
  }

  .detail-panel {
    grid-row: 1;
  }
}

@media (max-width: 620px) {
  .video-main {
    padding: 12px;
  }

  .library-list article {
    flex-wrap: wrap;
  }

  .library-list article > div {
    flex-basis: calc(100% - 32px);
  }

  .library-list small {
    margin-left: 30px;
  }

  .toolbar {
    flex-wrap: wrap;
  }

  .toolbar .search {
    flex-basis: 100%;
  }

  .detail-view-hero,
  .detail-meta-grid.wide,
  .dvr-settings-grid {
    grid-template-columns: 1fr;
  }

  .detail-poster.large {
    max-width: 190px;
  }

  .library-strip,
  .channel-grid {
    grid-template-columns: 1fr;
  }

  .poster-row {
    grid-template-columns: repeat(auto-fill, minmax(112px, 1fr));
  }

  .task-row {
    grid-template-columns: 1fr;
    align-items: stretch;
  }

  .task-progress > span {
    justify-self: start;
  }

  .media-card {
    grid-template-columns: 48px minmax(0, 1fr);
  }

  .media-card button {
    display: none;
  }
}
</style>
