<script setup lang="ts">
import { computed, nextTick, onMounted, ref } from 'vue';
import {
  Clapperboard,
  Film,
  ListVideo,
  Loader2,
  Radio,
  RefreshCcw,
  Search,
  Settings,
  Sparkles,
  Tv,
  Wand2,
  X,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { aiAnalysisStore } from '../../stores/aiAnalysis';
import { UiButton, UiEmptyState, UiIconButton, UiModal, UiTabs, UiWindowPage, useConfirm, type TabItem } from '../ui';

const confirm = useConfirm();
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
import VideoHomePanel from './video/VideoHomePanel.vue';
import VideoMoviesPanel from './video/VideoMoviesPanel.vue';
import VideoChannelsPanel from './video/VideoChannelsPanel.vue';
import VideoRecordingsPanel from './video/VideoRecordingsPanel.vue';
import VideoTasksPanel from './video/VideoTasksPanel.vue';
import VideoSettingsPanel from './video/VideoSettingsPanel.vue';
import './video/video-window.css';

type TabKey = 'home' | 'movies' | 'live' | 'tasks' | 'settings';
type LiveViewKey = 'channels' | 'guide' | 'recordings';

const tabs: Array<{ id: TabKey; label: string; icon: unknown }> = [
  { id: 'home', label: '首页', icon: Film },
  { id: 'movies', label: '影片', icon: Clapperboard },
  { id: 'live', label: '电视直播', icon: Tv },
  { id: 'tasks', label: '任务', icon: ListVideo },
  { id: 'settings', label: '设置', icon: Settings },
];

const tabItems = computed<TabItem[]>(() =>
  tabs.map((tab) => ({
    key: tab.id,
    label: tab.label,
    icon: tab.icon as TabItem['icon'],
    badge:
      tab.id === 'movies'
        ? items.value.length || undefined
        : tab.id === 'live'
          ? liveChannels.value.length || undefined
          : tab.id === 'tasks'
            ? tasks.value.length || undefined
            : undefined,
  })),
);

const activeTab = ref<TabKey>('home');
const loading = ref(false);
const busyMessage = ref('');
const analyzeBusy = ref(false);
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
  if (
    !(await confirm({
      title: '删除媒体库？',
      message: `删除媒体库「${libraryItem.name}」？此操作只移除媒体库配置、索引和任务，不删除磁盘文件。`,
      tone: 'danger',
    }))
  )
    return;
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

async function analyzeVideo(item?: VideoItem) {
  const target = item ?? selectedItem.value;
  if (!target) return;
  analyzeBusy.value = true;
  await runBusy('正在加入 AI 分析队列', async () => {
    await aiAnalysisStore.reanalyze({ scope: 'item', itemId: `video:${target.id}` });
    statusMessage.value = `已将「${displayTitle(target)}」加入 AI 分析队列，简介 / 转写稍后回填。`;
  });
  analyzeBusy.value = false;
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
  <UiWindowPage
    class="video-center"
    layout="stack"
    :icon="Film"
    title="影视中心"
    :subtitle="libraryStatus"
    :status="`${movieCount} 部电影 · ${liveChannels.length} 个频道`"
  >
    <template #toolbar>
      <UiTabs v-model="activeTab" :tabs="tabItems" variant="segmented" size="sm" overflow="menu" @update:model-value="(v) => switchTab(v as TabKey)" />
      <header class="toolbar">
        <div class="search">
          <Search :size="16" />
          <input v-model="query" placeholder="搜索片名、文件名、分类" />
        </div>
        <UiButton variant="ghost" tone="neutral" :icon-left="RefreshCcw" @click="loadAll">刷新</UiButton>
        <UiButton :icon-left="Sparkles" @click="scanLibrary">扫描</UiButton>
        <UiButton variant="ghost" tone="neutral" :icon-left="Wand2" @click="scrapeFilteredItems">刮削当前列表</UiButton>
      </header>
    </template>

      <div v-if="errorMessage" class="message error">{{ errorMessage }}</div>
      <div v-if="busyMessage" class="message">
        <Loader2 :size="15" class="spin" />
        {{ busyMessage }}
      </div>
      <div v-else-if="statusMessage" class="message muted">{{ statusMessage }}</div>

      <VideoHomePanel
        v-if="activeTab === 'home'"
        :libraries="libraries"
        :items="items"
        :running-tasks="runningTasks"
        :type-label="typeLabel"
        :poster-url="posterUrl"
        :display-title="displayTitle"
        :card-subtitle="cardSubtitle"
        @open-library="(library) => { libraryFilter = library.id; activeTab = 'movies'; }"
        @create-library="activeTab = 'settings'"
        @view-all="activeTab = 'movies'"
        @open-detail="openDetail"
        @play="playItem"
        @view-tasks="activeTab = 'tasks'"
      />

      <VideoMoviesPanel
        v-else-if="activeTab === 'movies'"
        v-model:library-filter="libraryFilter"
        v-model:kind-filter="kindFilter"
        v-model:transcode-profile="transcodeProfile"
        v-model:analyze-busy="analyzeBusy"
        :libraries="libraries"
        :filtered-items="filteredItems"
        :detail-item="detailItem"
        :poster-url="posterUrl"
        :display-title="displayTitle"
        :card-subtitle="cardSubtitle"
        :episode-label="episodeLabel"
        :display-subtitle="displaySubtitle"
        :chips="chips"
        :detail-overview="detailOverview"
        :all-tracks="allTracks"
        :release-label="releaseLabel"
        :duration-label="durationLabel"
        :list-label="listLabel"
        :scraped-label="scrapedLabel"
        @close-detail="closeDetail"
        @open-detail="openDetail"
        @play="playItem"
        @scrape="(item) => createTask('scrape', item)"
        @subtitle="(item) => createTask('subtitle', item)"
        @transcode="(item) => createTask('transcode', item)"
        @analyze="(item) => analyzeVideo(item)"
      />

      <section v-else-if="activeTab === 'live'" class="live-view">
        <div class="live-tabs">
          <button :class="{ active: liveView === 'channels' }" type="button" @click="switchLiveView('channels')">频道</button>
          <button :class="{ active: liveView === 'guide' }" type="button" @click="switchLiveView('guide')">节目指南</button>
          <button :class="{ active: liveView === 'recordings' }" type="button" @click="switchLiveView('recordings')">录制</button>
        </div>

        <div class="live-layout">
          <VideoChannelsPanel v-if="liveView === 'channels'" :live-channels="liveChannels" @play="playLive" />

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
            <UiEmptyState v-else :icon="ListVideo" title="还没有节目指南" description="导入 XMLTV 后会显示节目表。" />
          </div>

          <VideoRecordingsPanel
            v-else-if="liveView === 'recordings'"
            :recording-timers="recordingTimers"
            :recordings="recordings"
            :dvr-settings="dvrSettings"
            :live-time-range="liveTimeRange"
            :status-label="statusLabel"
            @cancel="cancelRecording"
          />
        </div>
      </section>

      <VideoTasksPanel
        v-else-if="activeTab === 'tasks'"
        :tasks="tasks"
        :task-label="taskLabel"
        :status-label="statusLabel"
      />

      <VideoSettingsPanel
        v-else
        :libraries="libraries"
        :live-sources="liveSources"
        :live-channels="liveChannels"
        :guide-sources="guideSources"
        :live-programs="livePrograms"
        :recording-timers="recordingTimers"
        :new-library="newLibrary"
        :new-live-source="newLiveSource"
        :new-guide-source="newGuideSource"
        :dvr-settings="dvrSettings"
        :type-label="typeLabel"
        @create-library="createLibrary"
        @delete-library="deleteLibrary"
        @create-live-source="createLiveSource"
        @create-guide-source="createGuideSource"
        @save-dvr-settings="saveDvrSettings"
      />

    <UiModal
      :open="!!selectedProgram"
      :title="selectedProgram?.title"
      size="lg"
      @close="closeProgramDetail"
      @update:open="(value) => { if (!value) closeProgramDetail(); }"
    >
      <div v-if="selectedProgram" class="program-detail-body">
        <p class="program-detail-channel">{{ selectedProgram.channelName }} · {{ liveTimeRange(selectedProgram.startAt, selectedProgram.endAt) }}</p>
        <p>{{ selectedProgram.overview || listLabel(selectedProgram.categories, '暂无节目简介') }}</p>
        <dl>
          <div><dt>频道</dt><dd>{{ selectedProgram.channelName }}</dd></div>
          <div><dt>时长</dt><dd>{{ programDuration(selectedProgram) }}</dd></div>
          <div><dt>状态</dt><dd>{{ programStatus(selectedProgram) === 'current' ? '正在播放' : programStatus(selectedProgram) === 'future' ? '未开始' : '已结束' }}</dd></div>
          <div><dt>分类</dt><dd>{{ listLabel(selectedProgram.categories) }}</dd></div>
        </dl>
      </div>
      <template #footer>
        <UiButton
          v-if="selectedProgram"
          tone="primary"
          :icon-left="Radio"
          :disabled="programRecordDisabled(selectedProgram)"
          @click="recordProgram(selectedProgram)"
        >
          {{ programActionLabel(selectedProgram) }}
        </UiButton>
      </template>
    </UiModal>

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
  </UiWindowPage>
</template>
