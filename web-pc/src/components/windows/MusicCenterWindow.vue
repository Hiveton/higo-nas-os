<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import {
  Album,
  ChevronLeft,
  Disc3,
  Folder,
  Heart,
  Library,
  List,
  ListMusic,
  MoreHorizontal,
  Pause,
  Play,
  RefreshCcw,
  Repeat,
  Search,
  Settings,
  Shuffle,
  SkipBack,
  SkipForward,
  UserRound,
  Volume2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { UiButton } from '../ui';
import type { MusicAlbum, MusicLibrarySettings, MusicScanResult, MusicTrack } from '../../api/types';
import MusicSidebar from './music/MusicSidebar.vue';
import MusicLibraryPanel from './music/MusicLibraryPanel.vue';
import MusicDetailsPanel from './music/MusicDetailsPanel.vue';
import './music/music-window.css';

const settings = ref<MusicLibrarySettings>({
  paths: [],
  autoScan: true,
  trackCount: 0,
  status: '读取中',
});
const tracks = ref<MusicTrack[]>([]);
const albums = ref<MusicAlbum[]>([]);
const selectedTrackId = ref('');
const search = ref('');
const libraryPathText = ref('');
const statusMessage = ref('正在读取媒体库设置');
const lyricsText = ref('');
const activeView = ref<'tracks' | 'albums' | 'library'>('tracks');
const mobilePlayerPane = ref<'cover' | 'lyrics' | 'library'>('cover');
const isLoading = ref(false);
const isScanning = ref(false);
const isPlaying = ref(false);
const isPlayerFocusMode = ref(false);
const playMode = ref<'sequence' | 'shuffle' | 'repeat'>('sequence');
const currentTime = ref(0);
const duration = ref(0);
const volume = ref(0.82);
const audioRef = ref<HTMLAudioElement | null>(null);
const heroSpectrumRef = ref<HTMLCanvasElement | null>(null);
const focusSpectrumRef = ref<HTMLCanvasElement | null>(null);
const miniSpectrumRef = ref<HTMLCanvasElement | null>(null);
const lyricsScrollRef = ref<HTMLElement | null>(null);

let audioContext: AudioContext | null = null;
let analyser: AnalyserNode | null = null;
let mediaSource: MediaElementAudioSourceNode | null = null;
let frequencyData: Uint8Array<ArrayBuffer> | null = null;
let spectrumFrame = 0;

const selectedTrack = computed(() => tracks.value.find((track) => track.id === selectedTrackId.value) ?? tracks.value[0]);
const filteredTracks = computed(() => {
  const q = search.value.trim().toLowerCase();
  if (!q) return tracks.value;
  return tracks.value.filter((track) =>
    [track.title, track.artist, track.album, track.fileName, track.codec].some((value) => value.toLowerCase().includes(q)),
  );
});
const albumSummary = computed(() => `${albums.value.length} 张专辑 · ${tracks.value.length} 首曲目`);
const selectedDuration = computed(() => duration.value || trackDuration(selectedTrack.value));
const playbackProgress = computed(() =>
  selectedDuration.value > 0 ? Math.min(100, (currentTime.value / selectedDuration.value) * 100) : 0,
);
const selectedAlbumTracks = computed(() => {
  const current = selectedTrack.value;
  if (!current) return [];
  return tracks.value.filter((track) => track.album === current.album);
});
const parsedLyrics = computed(() => parseLyrics(lyricsText.value));
const activeLyricIndex = computed(() => {
  const lines = parsedLyrics.value;
  if (!lines.length) return -1;
  let index = 0;
  for (let i = 0; i < lines.length; i += 1) {
    if (lines[i].time <= currentTime.value + 0.25) index = i;
  }
  return index;
});
const lyricPreview = computed(() => {
  const lines = parsedLyrics.value;
  if (!lines.length) return [];
  const active = Math.max(0, activeLyricIndex.value);
  const start = Math.max(0, active - 4);
  return lines.slice(start, Math.min(lines.length, active + 6)).map((line, offset) => ({
    ...line,
    sourceIndex: start + offset,
  }));
});
const currentLyricLine = computed(() => {
  if (!parsedLyrics.value.length) return '暂无歌词，播放时会显示当前歌词。';
  return lyricPreview.value.find((line) => line.sourceIndex === activeLyricIndex.value)?.text ?? parsedLyrics.value[0]?.text ?? '';
});
const sidebarItems = computed(() => [
  { key: 'tracks', label: '所有曲目', count: settings.value.trackCount || tracks.value.length, icon: Library },
  { key: 'albums', label: '专辑', count: albums.value.length, icon: Album },
  { key: 'artists', label: '艺术家', count: new Set(tracks.value.map((track) => track.artist)).size, icon: UserRound },
  { key: 'songs', label: '歌曲', count: tracks.value.length, icon: ListMusic },
  { key: 'folders', label: '文件夹', count: settings.value.paths.length, icon: Folder },
]);
const mobileLibraryTabs = ['歌曲', '专辑', '艺术家', '文件夹'];

watch(selectedTrack, async (track) => {
  lyricsText.value = '';
  currentTime.value = 0;
  duration.value = trackDuration(track);
  isPlaying.value = false;
  if (!track) return;
  if (track.lyricsUrl) {
    try {
      lyricsText.value = await apiClient.music.getLyrics(track.id);
    } catch {
      lyricsText.value = '';
    }
  }
});

watch(volume, (next) => {
  if (audioRef.value) audioRef.value.volume = next;
});

watch(activeLyricIndex, () => {
  void nextTick(() => {
    const container = lyricsScrollRef.value;
    const activeLine = container?.querySelector('.active') as HTMLElement | null;
    activeLine?.scrollIntoView({ block: 'center', behavior: 'smooth' });
  });
});

onMounted(() => {
  void loadMusic();
  void nextTick(startSpectrum);
});

onBeforeUnmount(() => {
  if (spectrumFrame) cancelAnimationFrame(spectrumFrame);
  if (audioContext) void audioContext.close();
});

async function loadMusic() {
  isLoading.value = true;
  try {
    const [nextSettings, nextTracks, nextAlbums] = await Promise.all([
      apiClient.music.getLibrary(),
      apiClient.music.getTracks(),
      apiClient.music.getAlbums(),
    ]);
    const safeSettings = normalizeSettings(nextSettings);
    settings.value = safeSettings;
    tracks.value = nextTracks;
    albums.value = nextAlbums;
    libraryPathText.value = safeSettings.paths.join('\n');
    selectedTrackId.value = selectedTrackId.value || nextTracks[0]?.id || '';
    statusMessage.value = safeSettings.trackCount
      ? `媒体库已就绪，${safeSettings.trackCount} 首曲目可播放。`
      : '还没有扫描到音乐，请设置媒体库路径后扫描。';
  } catch (error) {
    statusMessage.value = errorMessage(error);
  } finally {
    isLoading.value = false;
  }
}

async function saveLibrary() {
  isLoading.value = true;
  try {
    const paths = libraryPathText.value
      .split('\n')
      .map((item) => item.trim())
      .filter(Boolean);
    settings.value = normalizeSettings(await apiClient.music.updateLibrary({ paths, autoScan: settings.value.autoScan }));
    await refreshTracks();
    statusMessage.value = '媒体库设置已保存。';
  } catch (error) {
    statusMessage.value = errorMessage(error);
  } finally {
    isLoading.value = false;
  }
}

async function scanLibrary() {
  isScanning.value = true;
  try {
    const result: MusicScanResult = await apiClient.music.scan();
    await refreshTracks();
    statusMessage.value = result.message;
  } catch (error) {
    statusMessage.value = errorMessage(error);
  } finally {
    isScanning.value = false;
  }
}

async function refreshTracks() {
  const [nextSettings, nextTracks, nextAlbums] = await Promise.all([
    apiClient.music.getLibrary(),
    apiClient.music.getTracks({ q: search.value }),
    apiClient.music.getAlbums(),
  ]);
  settings.value = normalizeSettings(nextSettings);
  tracks.value = nextTracks;
  albums.value = nextAlbums;
  if (!nextTracks.some((track) => track.id === selectedTrackId.value)) {
    selectedTrackId.value = nextTracks[0]?.id ?? '';
  }
}

function selectTrack(track: MusicTrack, playNow = false) {
  selectedTrackId.value = track.id;
  if (playNow) setTimeout(() => void playSelected(), 0);
}

function selectSidebarItem(key: string) {
  activeView.value = key === 'albums' ? 'albums' : key === 'folders' ? 'library' : 'tracks';
}

async function playSelected() {
  const track = selectedTrack.value;
  const audio = audioRef.value;
  if (!track || !audio) return;
  const streamUrl = apiClient.music.streamUrl(track.id);
  const absoluteStreamUrl = new URL(streamUrl, window.location.href).href;
  if (audio.src !== absoluteStreamUrl) audio.src = streamUrl;
  audio.volume = volume.value;
  await setupAudioAnalyser();
  await audio.play();
}

function togglePlay() {
  const audio = audioRef.value;
  if (!audio || !selectedTrack.value) return;
  if (isPlaying.value) {
    audio.pause();
    return;
  }
  void playSelected();
}

function playPrevious() {
  const list = filteredTracks.value;
  if (!list.length) return;
  const currentIndex = Math.max(0, list.findIndex((track) => track.id === selectedTrack.value?.id));
  const next = playMode.value === 'shuffle' ? randomTrack(list, selectedTrack.value?.id) : list[(currentIndex - 1 + list.length) % list.length];
  if (next) selectTrack(next, true);
}

function playNext() {
  const list = filteredTracks.value;
  if (!list.length) return;
  if (playMode.value === 'repeat' && selectedTrack.value) {
    selectTrack(selectedTrack.value, true);
    return;
  }
  const currentIndex = Math.max(0, list.findIndex((track) => track.id === selectedTrack.value?.id));
  const next = playMode.value === 'shuffle' ? randomTrack(list, selectedTrack.value?.id) : list[(currentIndex + 1) % list.length];
  if (next) selectTrack(next, true);
}

function randomTrack(list: MusicTrack[], currentId?: string) {
  if (list.length <= 1) return list[0];
  const candidates = list.filter((track) => track.id !== currentId);
  return candidates[Math.floor(Math.random() * candidates.length)] ?? list[0];
}

function cyclePlayMode() {
  playMode.value = playMode.value === 'sequence' ? 'shuffle' : playMode.value === 'shuffle' ? 'repeat' : 'sequence';
}

function playModeLabel() {
  if (playMode.value === 'shuffle') return '随机播放';
  if (playMode.value === 'repeat') return '单曲循环';
  return '顺序播放';
}

function seek(event: Event) {
  const audio = audioRef.value;
  if (!audio) return;
  audio.currentTime = Number((event.target as HTMLInputElement).value);
}

function coverUrl(track?: MusicTrack) {
  return track?.coverUrl ? apiClient.music.coverUrl(track.id) : '';
}

function openPlayer() {
  isPlayerFocusMode.value = true;
  mobilePlayerPane.value = 'cover';
  void nextTick(startSpectrum);
}

function closePlayer() {
  isPlayerFocusMode.value = false;
  void nextTick(startSpectrum);
}

function toggleMobileCoverLyrics() {
  mobilePlayerPane.value = mobilePlayerPane.value === 'cover' ? 'lyrics' : 'cover';
}

function handleLoadedMetadata() {
  duration.value = audioRef.value?.duration || trackDuration(selectedTrack.value);
}

function handleTimeUpdate() {
  currentTime.value = audioRef.value?.currentTime || 0;
}

function handlePlay() {
  isPlaying.value = true;
  void setupAudioAnalyser();
  startSpectrum();
}

function handlePause() {
  isPlaying.value = false;
}

async function setupAudioAnalyser() {
  const audio = audioRef.value;
  const AudioContextCtor =
    window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!audio || !AudioContextCtor) return;
  if (!audioContext) audioContext = new AudioContextCtor();
  if (!mediaSource) {
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 512;
    analyser.smoothingTimeConstant = 0.72;
    frequencyData = new Uint8Array(analyser.frequencyBinCount);
    mediaSource = audioContext.createMediaElementSource(audio);
    mediaSource.connect(analyser);
    analyser.connect(audioContext.destination);
  }
  if (audioContext.state === 'suspended') await audioContext.resume();
  startSpectrum();
}

function startSpectrum() {
  if (spectrumFrame) return;
  const draw = () => {
    drawSpectrum(heroSpectrumRef.value, 150, true);
    drawSpectrum(focusSpectrumRef.value, 172, true);
    drawSpectrum(miniSpectrumRef.value, 180, false);
    spectrumFrame = requestAnimationFrame(draw);
  };
  spectrumFrame = requestAnimationFrame(draw);
}

function drawSpectrum(canvas: HTMLCanvasElement | null, barCount: number, prominent: boolean) {
  if (!canvas) return;
  const rect = canvas.getBoundingClientRect();
  if (rect.width <= 0 || rect.height <= 0) return;
  const dpr = Math.max(1, window.devicePixelRatio || 1);
  const width = Math.floor(rect.width * dpr);
  const height = Math.floor(rect.height * dpr);
  if (canvas.width !== width || canvas.height !== height) {
    canvas.width = width;
    canvas.height = height;
  }
  const ctx = canvas.getContext('2d');
  if (!ctx) return;
  ctx.clearRect(0, 0, width, height);
  const values = spectrumValues(barCount);
  const gap = prominent ? 2.2 * dpr : 1.4 * dpr;
  const barWidth = Math.max(1.2 * dpr, (width - gap * (barCount - 1)) / barCount);
  const baseY = height - (prominent ? 8 * dpr : 4 * dpr);
  const maxBarHeight = height * (prominent ? 0.82 : 0.64);
  for (let i = 0; i < barCount; i += 1) {
    const x = i * (barWidth + gap);
    const h = Math.max(0.06, values[i] ?? 0) * maxBarHeight;
    const y = baseY - h;
    const hue = 205 + (i / Math.max(1, barCount - 1)) * 92;
    const gradient = ctx.createLinearGradient(0, y, 0, baseY);
    gradient.addColorStop(0, `hsla(${hue}, 96%, ${prominent ? 63 : 60}%, ${prominent ? 0.9 : 0.7})`);
    gradient.addColorStop(1, `hsla(${hue + 26}, 94%, 68%, ${prominent ? 0.45 : 0.22})`);
    ctx.fillStyle = gradient;
    ctx.fillRect(x, y, barWidth, h);
    if (prominent && i % 5 === 0) {
      ctx.fillStyle = `hsla(${hue}, 96%, 70%, 0.32)`;
      ctx.fillRect(x, Math.max(0, y - 7 * dpr), Math.max(1 * dpr, barWidth * 0.7), 1.3 * dpr);
    }
  }
}

function spectrumValues(barCount: number) {
  const values = new Array<number>(barCount);
  if (analyser && frequencyData && isPlaying.value) {
    analyser.getByteFrequencyData(frequencyData);
    for (let i = 0; i < barCount; i += 1) {
      const start = Math.floor((i / barCount) * frequencyData.length);
      const end = Math.max(start + 1, Math.floor(((i + 1) / barCount) * frequencyData.length));
      let sum = 0;
      for (let j = start; j < end; j += 1) sum += frequencyData[j] ?? 0;
      values[i] = Math.min(1, Math.max(0.07, sum / Math.max(1, end - start) / 255));
    }
    return values;
  }
  const now = performance.now() / 1000;
  for (let i = 0; i < barCount; i += 1) {
    const wave = Math.sin(now * 1.8 + i * 0.29) * 0.45 + Math.sin(now * 0.8 + i * 0.13) * 0.55;
    values[i] = 0.12 + Math.max(0, wave) * 0.24;
  }
  return values;
}

function formatTime(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '00:00';
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60);
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
}

function trackDuration(track?: MusicTrack | null) {
  return Number(track?.durationSeconds ?? 0) || 0;
}

function formatDate(value?: string) {
  if (!value) return '未扫描';
  const date = new Date(value);
  if (Number.isNaN(date.getTime())) return value;
  return date.toLocaleString('zh-CN', { month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' });
}

function parseLyrics(value: string) {
  const lines = value
    .split(/\r?\n/)
    .map((line) => {
      const match = line.match(/^\[(\d{1,2}):(\d{2})(?:\.(\d{1,3}))?]\s*(.*)$/);
      if (!match) return null;
      const minutes = Number(match[1]);
      const seconds = Number(match[2]);
      const ms = Number((match[3] ?? '0').padEnd(3, '0'));
      return { time: minutes * 60 + seconds + ms / 1000, text: match[4] || ' ' };
    })
    .filter((line): line is { time: number; text: string } => Boolean(line));
  return lines.length ? lines : value.split(/\r?\n/).filter(Boolean).map((text, index) => ({ time: index * 4, text }));
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : '操作失败';
}

function normalizeSettings(value: MusicLibrarySettings): MusicLibrarySettings {
  return {
    ...value,
    paths: Array.isArray(value.paths) ? value.paths : [],
  };
}
</script>

<template>
  <div class="music-app" :class="{ 'music-app--player': isPlayerFocusMode }">
    <audio
      ref="audioRef"
      preload="metadata"
      @play="handlePlay"
      @pause="handlePause"
      @ended="playNext"
      @loadedmetadata="handleLoadedMetadata"
      @timeupdate="handleTimeUpdate"
    />

    <MusicSidebar
      :sidebar-items="sidebarItems"
      :active-view="activeView"
      :status-message="statusMessage"
      :settings="settings"
      :format-date="formatDate"
      @select-item="selectSidebarItem"
    />

    <main class="music-app__stage">
      <header class="music-app__topbar">
        <div class="music-app__crumb">媒体库 <span>›</span> 音乐 <span>›</span> 所有曲目</div>
        <label class="music-app__search">
          <Search :size="16" />
          <input v-model="search" placeholder="搜索歌曲、专辑、艺术家、歌词" @keyup.enter="refreshTracks" />
          <kbd>⌘ F</kbd>
        </label>
        <UiButton class="music-app__scan" variant="ghost" tone="neutral" :icon-left="RefreshCcw" :loading="isScanning" @click="scanLibrary">
          {{ isScanning ? '扫描中' : '扫描状态：空闲' }}
        </UiButton>
      </header>

      <template v-if="isPlayerFocusMode">
        <section class="music-player">
          <button class="music-player__mobile-back" type="button" aria-label="返回列表" @click="closePlayer"><ChevronLeft :size="20" /></button>
          <div class="music-player__mobile-tabs">
            <button type="button" :class="{ active: mobilePlayerPane === 'cover' }" @click="mobilePlayerPane = 'cover'">歌曲</button>
            <button type="button" :class="{ active: mobilePlayerPane === 'lyrics' }" @click="mobilePlayerPane = 'lyrics'">歌词</button>
            <button type="button">相关</button>
          </div>

          <section class="music-player__main" :class="{ 'music-player__main--lyrics': mobilePlayerPane === 'lyrics' }">
            <div class="music-player__cover-wrap" @click="toggleMobileCoverLyrics">
              <button class="music-player__cover" type="button" aria-label="切换封面与歌词">
                <img v-if="coverUrl(selectedTrack)" :src="coverUrl(selectedTrack)" :alt="selectedTrack?.album ?? '专辑封面'" />
                <Disc3 v-else :size="64" />
              </button>
              <div class="music-player__vinyl" aria-hidden="true" />
            </div>

            <div class="music-player__info">
              <h2>{{ selectedTrack?.title ?? '未选择曲目' }}</h2>
              <p>{{ selectedTrack?.artist ?? '未知艺术家' }} · {{ selectedTrack?.album ?? '未归类专辑' }}</p>
              <div class="music-player__badges">
                <span>Hi-Res</span>
                <span>{{ selectedTrack?.codec ?? 'AUTO' }}</span>
                <span>24bit</span>
                <span>96kHz</span>
              </div>
              <p class="music-player__desc">我梦中见过你 因为我喜欢你<br />与你一起 而承担的风雨</p>
              <div class="music-player__actions">
                <button type="button" aria-label="收藏"><Heart :size="16" /></button>
                <button type="button" aria-label="添加到歌单">+</button>
                <button type="button" aria-label="更多操作"><MoreHorizontal :size="17" /></button>
              </div>
            </div>

            <div class="music-player__lyrics-panel">
              <header>
                <button class="active" type="button">歌词</button>
                <button type="button">相关</button>
              </header>
              <div ref="lyricsScrollRef" class="music-player__lyrics-scroll" @click="toggleMobileCoverLyrics">
                <template v-if="parsedLyrics.length">
                  <p
                    v-for="(line, index) in parsedLyrics"
                    :key="`${line.time}-${index}`"
                    :class="{ active: index === activeLyricIndex }"
                  >
                    {{ line.text }}
                  </p>
                </template>
                <template v-else>
                  <p>穿过人海的喧嚣</p>
                  <p>我会爱你到尽头</p>
                  <p class="active">关于我们的永远</p>
                  <p>我梦中见过你 因为我喜欢你</p>
                  <p>与你一起 而浪漫的风雨</p>
                </template>
              </div>
              <button class="music-player__lyric-settings" type="button"><Settings :size="14" /> 歌词设置</button>
            </div>
          </section>

          <div class="music-player__mobile-now">
            <h2>{{ selectedTrack?.title ?? '未选择曲目' }}</h2>
            <p>{{ selectedTrack?.artist ?? '未知艺术家' }} · {{ selectedTrack?.album ?? '未归类专辑' }}</p>
            <div class="music-player__badges">
              <span>{{ selectedTrack?.codec ?? 'AUTO' }}</span>
              <span>24bit</span>
              <span>96kHz</span>
            </div>
          </div>

          <div class="music-player__spectrum"><canvas ref="focusSpectrumRef" /></div>
          <div class="music-player__progress">
            <span>{{ formatTime(currentTime) }}</span>
            <input
              type="range"
              min="0"
              :max="selectedDuration || 0"
              :value="currentTime"
              :style="{ '--progress': `${playbackProgress}%` }"
              aria-label="播放进度"
              @input="seek"
            />
            <span>{{ formatTime(selectedDuration) }}</span>
          </div>
          <div class="music-player__controls">
            <button type="button" :title="playModeLabel()" @click="cyclePlayMode">
              <Shuffle v-if="playMode === 'shuffle'" :size="18" />
              <Repeat v-else :size="18" />
            </button>
            <button type="button" aria-label="上一首" @click="playPrevious"><SkipBack :size="20" /></button>
            <button class="music-player__play" type="button" :aria-label="isPlaying ? '暂停' : '播放'" @click="togglePlay">
              <Pause v-if="isPlaying" :size="28" />
              <Play v-else :size="28" />
            </button>
            <button type="button" aria-label="下一首" @click="playNext"><SkipForward :size="20" /></button>
            <button type="button" aria-label="返回列表" @click="closePlayer"><List :size="20" /></button>
          </div>
        </section>
      </template>

      <template v-else>
        <MusicLibraryPanel
          v-model:library-path-text="libraryPathText"
          :mobile-library-tabs="mobileLibraryTabs"
          :is-scanning="isScanning"
          :is-loading="isLoading"
          :filtered-tracks="filteredTracks"
          :selected-track="selectedTrack"
          :active-view="activeView"
          :cover-url="coverUrl"
          :format-time="formatTime"
          :track-duration="trackDuration"
          @scan="scanLibrary"
          @select-track="selectTrack($event, true)"
          @save-library="saveLibrary"
        />
      </template>
    </main>

    <MusicDetailsPanel
      v-if="!isPlayerFocusMode"
      :selected-track="selectedTrack"
      :cover-url="coverUrl"
      :format-date="formatDate"
    />

    <section class="music-mini">
      <canvas ref="miniSpectrumRef" class="music-mini__spectrum" aria-hidden="true" />
      <button class="music-mini__cover" type="button" aria-label="打开播放器" @click="openPlayer">
        <img v-if="coverUrl(selectedTrack)" :src="coverUrl(selectedTrack)" :alt="selectedTrack?.album ?? '专辑封面'" />
        <Disc3 v-else :size="28" />
      </button>
      <div class="music-mini__info">
        <strong>{{ selectedTrack?.title ?? '未选择曲目' }}</strong>
        <span>{{ selectedTrack?.artist ?? '未知艺术家' }} · {{ selectedTrack?.album ?? '未归类专辑' }}</span>
        <small>{{ currentLyricLine }}</small>
        <div class="music-mini__inline-meta">
          <span>{{ selectedTrack?.codec ?? 'AUTO' }}</span>
          <span>24bit</span>
          <span>96kHz</span>
        </div>
      </div>
      <button type="button" aria-label="收藏"><Heart :size="17" /></button>
      <button type="button" aria-label="更多操作"><MoreHorizontal :size="18" /></button>
      <div class="music-mini__progress">
        <span>{{ formatTime(currentTime) }}</span>
        <input
          type="range"
          min="0"
          :max="selectedDuration || 0"
          :value="currentTime"
          :style="{ '--progress': `${playbackProgress}%` }"
          aria-label="播放进度"
          @input="seek"
        />
        <span>{{ formatTime(selectedDuration) }}</span>
      </div>
      <div class="music-mini__controls">
        <button type="button" :title="playModeLabel()" @click="cyclePlayMode">
          <Shuffle v-if="playMode === 'shuffle'" :size="17" />
          <Repeat v-else :size="17" />
        </button>
        <button type="button" aria-label="上一首" @click="playPrevious"><SkipBack :size="18" /></button>
        <button class="music-mini__play" type="button" :aria-label="isPlaying ? '暂停' : '播放'" @click="togglePlay">
          <Pause v-if="isPlaying" :size="22" />
          <Play v-else :size="22" />
        </button>
        <button type="button" aria-label="下一首" @click="playNext"><SkipForward :size="18" /></button>
      </div>
      <div class="music-mini__side">
        <Volume2 :size="16" />
        <input
          v-model.number="volume"
          type="range"
          min="0"
          max="1"
          step="0.01"
          :style="{ '--progress': `${volume * 100}%` }"
          aria-label="音量"
        />
        <button type="button"><List :size="17" /> 队列 <small>{{ filteredTracks.length }}</small></button>
      </div>
    </section>
  </div>
</template>
