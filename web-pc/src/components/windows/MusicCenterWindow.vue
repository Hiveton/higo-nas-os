<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import {
  Album,
  AudioLines,
  ChevronDown,
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
  SlidersHorizontal,
  UserRound,
  Volume2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { MusicAlbum, MusicLibrarySettings, MusicScanResult, MusicTrack } from '../../api/types';

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

    <aside class="music-app__sidebar">
      <div class="music-app__brand">
        <div class="music-app__brand-icon"><AudioLines :size="20" /></div>
        <div>
          <strong>音乐中心</strong>
          <span>家庭 NAS</span>
        </div>
      </div>
      <nav class="music-app__nav">
        <p>媒体库</p>
        <button
          v-for="item in sidebarItems"
          :key="item.key"
          type="button"
          :class="{ 'music-app__nav-item--active': activeView === item.key || (item.key === 'tracks' && activeView === 'tracks') }"
          @click="activeView = item.key === 'albums' ? 'albums' : item.key === 'folders' ? 'library' : 'tracks'"
        >
          <component :is="item.icon" :size="16" />
          <span>{{ item.label }}</span>
          <small>{{ item.count }}</small>
        </button>
      </nav>
      <nav class="music-app__nav music-app__nav--secondary">
        <p>我的音乐</p>
        <button type="button"><Heart :size="16" /><span>我喜欢的</span></button>
        <button type="button"><List :size="16" /><span>轻音乐</span></button>
        <button type="button"><ListMusic :size="16" /><span>经典怀旧</span></button>
      </nav>
      <div class="music-app__storage">
        <strong>媒体库状态</strong>
        <span>{{ statusMessage }}</span>
        <small>上次扫描：{{ formatDate(settings.lastScanAt) }}</small>
      </div>
      <div class="music-app__side-footer">
        <button type="button"><Settings :size="16" /></button>
      </div>
    </aside>

    <main class="music-app__stage">
      <header class="music-app__topbar">
        <div class="music-app__crumb">媒体库 <span>›</span> 音乐 <span>›</span> 所有曲目</div>
        <label class="music-app__search">
          <Search :size="16" />
          <input v-model="search" placeholder="搜索歌曲、专辑、艺术家、歌词" @keyup.enter="refreshTracks" />
          <kbd>⌘ F</kbd>
        </label>
        <button class="music-app__scan" type="button" :disabled="isScanning" @click="scanLibrary">
          <RefreshCcw :size="15" /> {{ isScanning ? '扫描中' : '扫描状态：空闲' }}
        </button>
      </header>

      <template v-if="isPlayerFocusMode">
        <section class="music-player">
          <button class="music-player__mobile-back" type="button" @click="closePlayer"><ChevronLeft :size="20" /></button>
          <div class="music-player__mobile-tabs">
            <button type="button" :class="{ active: mobilePlayerPane === 'cover' }" @click="mobilePlayerPane = 'cover'">歌曲</button>
            <button type="button" :class="{ active: mobilePlayerPane === 'lyrics' }" @click="mobilePlayerPane = 'lyrics'">歌词</button>
            <button type="button">相关</button>
          </div>

          <section class="music-player__main" :class="{ 'music-player__main--lyrics': mobilePlayerPane === 'lyrics' }">
            <div class="music-player__cover-wrap" @click="toggleMobileCoverLyrics">
              <button class="music-player__cover" type="button">
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
                <button type="button"><Heart :size="16" /></button>
                <button type="button">+</button>
                <button type="button"><MoreHorizontal :size="17" /></button>
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
        <section class="music-library">
          <div class="music-library__mobile-head">
            <h2>音乐库</h2>
            <div>
              <Search :size="18" />
              <MoreHorizontal :size="18" />
            </div>
          </div>
          <div class="music-library__mobile-tabs">
            <button v-for="tab in mobileLibraryTabs" :key="tab" type="button" :class="{ active: tab === '歌曲' }">{{ tab }}</button>
          </div>

          <section class="music-library__tools">
            <button type="button">全部曲目 <ChevronDown :size="14" /></button>
            <button type="button">全部专辑 <ChevronDown :size="14" /></button>
            <button type="button">全部艺术家 <ChevronDown :size="14" /></button>
            <span />
            <button type="button">排序：添加时间 <ChevronDown :size="14" /></button>
            <button type="button" :disabled="isScanning" @click="scanLibrary"><RefreshCcw :size="14" /> 重新扫描</button>
            <button type="button"><SlidersHorizontal :size="14" /></button>
          </section>

          <section class="music-library__table">
            <header>
              <span>#</span><span>标题</span><span>艺术家</span><span>专辑</span><span>格式</span><span>时长</span><span>大小</span>
            </header>
            <button
              v-for="(track, index) in filteredTracks"
              :key="track.id"
              type="button"
              class="music-library__row"
              :class="{ active: track.id === selectedTrack?.id }"
              @click="selectTrack(track, true)"
            >
              <span>{{ index + 1 }}</span>
              <span class="music-library__song">
                <img v-if="coverUrl(track)" :src="coverUrl(track)" :alt="track.album" />
                <AudioLines v-else :size="16" />
                <span>
                  <strong>{{ track.title }}</strong>
                  <em>{{ track.artist }} · {{ track.album }}</em>
                </span>
              </span>
              <span>{{ track.artist }}</span>
              <span>{{ track.album }}</span>
              <span>{{ track.codec }}</span>
              <span>{{ formatTime(trackDuration(track)) }}</span>
              <span>{{ track.size }}</span>
              <small class="music-library__mobile-codec">{{ track.codec }}</small>
            </button>
            <div v-if="!filteredTracks.length" class="music-library__empty">
              <Library :size="28" />
              <strong>当前没有曲目</strong>
              <span>设置媒体库路径后执行扫描。</span>
            </div>
          </section>

          <section v-if="activeView === 'library'" class="music-library__settings">
            <label>
              <span>媒体库路径，每行一个目录</span>
              <textarea v-model="libraryPathText" />
            </label>
            <button type="button" :disabled="isLoading" @click="saveLibrary">保存媒体库</button>
          </section>
        </section>
      </template>
    </main>

    <aside v-if="!isPlayerFocusMode" class="music-app__details">
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

    <section class="music-mini">
      <canvas ref="miniSpectrumRef" class="music-mini__spectrum" aria-hidden="true" />
      <button class="music-mini__cover" type="button" @click="openPlayer">
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
      <button type="button"><Heart :size="17" /></button>
      <button type="button"><MoreHorizontal :size="18" /></button>
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
        <button type="button" @click="playPrevious"><SkipBack :size="18" /></button>
        <button class="music-mini__play" type="button" @click="togglePlay">
          <Pause v-if="isPlaying" :size="22" />
          <Play v-else :size="22" />
        </button>
        <button type="button" @click="playNext"><SkipForward :size="18" /></button>
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

<style scoped>
.music-app {
  --music-blue: #3b82ff;
  --music-purple: #8063ff;
  --music-ink: #172033;
  --music-muted: #718099;
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr) 300px;
  grid-template-rows: minmax(0, 1fr) 112px;
  gap: 14px;
  height: 100%;
  min-height: 0;
  padding: 14px;
  color: var(--music-ink);
  background:
    radial-gradient(circle at 72% 18%, rgba(143, 122, 255, 0.14), transparent 32%),
    radial-gradient(circle at 22% 6%, rgba(93, 169, 255, 0.16), transparent 30%),
    linear-gradient(135deg, rgba(244, 250, 255, 0.96), rgba(247, 246, 255, 0.92));
  font-size: 12px;
  overflow: hidden;
}

.music-app--player {
  grid-template-columns: 230px minmax(0, 1fr);
  grid-template-rows: minmax(0, 1fr);
}

.music-app--player .music-app__sidebar,
.music-app--player .music-app__stage {
  grid-row: 1;
}

.music-app--player .music-app__stage {
  grid-template-rows: auto minmax(0, 1fr);
}

.music-app__sidebar,
.music-app__details,
.music-library__hero,
.music-library__table,
.music-mini,
.music-player__lyrics-panel {
  border: 1px solid rgba(132, 155, 186, 0.12);
  background: rgba(255, 255, 255, 0.58);
  box-shadow: 0 18px 46px rgba(54, 84, 130, 0.08);
  backdrop-filter: blur(22px);
}

.music-app__sidebar {
  display: grid;
  grid-row: 1 / 3;
  grid-template-rows: auto auto auto minmax(0, 1fr) auto;
  gap: 14px;
  min-height: 0;
  padding: 14px;
  border-radius: 18px;
}

.music-app__brand {
  display: flex;
  gap: 10px;
  align-items: center;
}

.music-app__brand-icon {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  color: #fff;
  background: linear-gradient(135deg, #53c0ff, #8a61ff);
  border-radius: 12px;
  box-shadow: 0 10px 22px rgba(88, 114, 255, 0.2);
}

.music-app__brand strong {
  display: block;
  font-size: 16px;
}

.music-app__brand span,
.music-app__nav p,
.music-app__storage span,
.music-app__storage small {
  color: var(--music-muted);
  font-size: 11px;
}

.music-app__nav {
  display: grid;
  gap: 6px;
}

.music-app__nav p {
  margin: 0 0 4px;
  font-weight: 700;
}

.music-app__nav button {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 9px;
  align-items: center;
  min-height: 34px;
  padding: 0 10px;
  color: #27364c;
  background: transparent;
  border: 0;
  border-radius: 10px;
  text-align: left;
}

.music-app__nav button small {
  color: #8a98ad;
}

.music-app__nav-item--active {
  color: var(--music-blue) !important;
  background: rgba(66, 143, 255, 0.12) !important;
}

.music-app__nav--secondary {
  margin-top: 4px;
}

.music-app__storage {
  align-self: end;
  display: grid;
  gap: 6px;
  padding: 12px;
  background: rgba(246, 251, 255, 0.76);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 13px;
  overflow: hidden;
  line-height: 1.45;
  white-space: normal;
}

.music-app__storage strong {
  display: block;
  overflow: hidden;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-app__storage span,
.music-app__storage small {
  display: block;
  min-width: 0;
  overflow: hidden;
  line-height: 1.5;
  text-overflow: ellipsis;
}

.music-app__side-footer {
  display: flex;
  justify-content: flex-end;
}

.music-app__side-footer button,
.music-app__scan,
.music-library__tools button,
.music-player__actions button,
.music-player__lyric-settings {
  display: inline-flex;
  gap: 7px;
  align-items: center;
  justify-content: center;
  min-height: 32px;
  padding: 0 12px;
  color: #253348;
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 10px;
}

.music-app__stage {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 14px;
  min-width: 0;
  min-height: 0;
}

.music-app__topbar {
  display: grid;
  grid-template-columns: minmax(150px, 0.75fr) minmax(280px, 1fr) auto;
  gap: 14px;
  align-items: center;
}

.music-app__crumb {
  color: #65738a;
  white-space: nowrap;
}

.music-app__crumb span {
  margin: 0 8px;
  color: #a0aabd;
}

.music-app__search {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-height: 38px;
  padding: 0 12px;
  background: rgba(255, 255, 255, 0.68);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 12px;
}

.music-app__search input {
  min-width: 0;
  color: var(--music-ink);
  background: transparent;
  border: 0;
  outline: 0;
}

.music-app__search kbd {
  color: #8a98ad;
  font-size: 10px;
}

.music-library {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 12px;
  min-height: 0;
  overflow: hidden;
}

.music-library__mobile-head,
.music-library__mobile-tabs {
  display: none;
}

.music-library__hero {
  display: grid;
  grid-template-columns: minmax(170px, 220px) minmax(0, 1fr) minmax(170px, 0.6fr);
  gap: 28px;
  align-items: center;
  min-height: 190px;
  padding: 20px 24px;
  border-radius: 18px;
  background:
    radial-gradient(circle at 90% 20%, rgba(172, 102, 255, 0.12), transparent 35%),
    rgba(255, 255, 255, 0.54);
}

.music-library__cover,
.music-player__cover,
.music-mini__cover {
  display: grid;
  box-sizing: border-box;
  aspect-ratio: 1 / 1;
  place-items: center;
  overflow: hidden;
  padding: 0;
  color: #fff;
  background: linear-gradient(135deg, #43c4ff, #8063ff);
  border: 0;
  line-height: 0;
}

.music-library__cover {
  width: 220px;
  aspect-ratio: 1;
  border-radius: 14px;
  box-shadow: 0 20px 44px rgba(23, 44, 78, 0.16);
}

.music-library__cover img,
.music-player__cover img,
.music-mini__cover img,
.music-app__details img,
.music-library__song img {
  display: block;
  width: 100%;
  height: 100%;
  object-fit: cover;
  object-position: center;
}

.music-library__hero-info {
  display: grid;
  gap: 9px;
  min-width: 0;
}

.music-library__hero-info h2,
.music-player__info h2 {
  margin: 0;
  font-size: 22px;
  line-height: 1.15;
}

.music-library__hero-info p,
.music-player__info p {
  margin: 0;
  color: var(--music-muted);
}

.music-library__badges,
.music-player__badges {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.music-library__badges span,
.music-player__badges span,
.music-mini__meta span,
.music-mini__inline-meta span,
.music-mini__side button small {
  padding: 3px 7px;
  color: #526176;
  background: rgba(236, 243, 252, 0.8);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 7px;
  font-size: 10px;
}

.music-library__spectrum,
.music-player__spectrum {
  height: 70px;
}

.music-library__spectrum canvas,
.music-player__spectrum canvas,
.music-mini__progress canvas {
  width: 100%;
  height: 100%;
}

.music-library__progress,
.music-player__progress,
.music-mini__progress {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 10px;
  align-items: center;
  color: #53637a;
  font-size: 11px;
}

.music-library__hero-copy {
  max-width: 360px;
  line-height: 1.7;
}

.music-app input[type="range"] {
  --progress: 0%;
  width: 100%;
  height: 18px;
  min-width: 0;
  appearance: none;
  background: transparent;
}

.music-app input[type="range"]::-webkit-slider-runnable-track {
  height: 4px;
  border-radius: 999px;
  background: linear-gradient(90deg, var(--music-blue), var(--music-purple) var(--progress), rgba(92, 114, 138, 0.18) var(--progress));
}

.music-app input[type="range"]::-webkit-slider-thumb {
  width: 13px;
  height: 13px;
  margin-top: -4.5px;
  appearance: none;
  background: linear-gradient(135deg, var(--music-blue), var(--music-purple));
  border: 2px solid rgba(255, 255, 255, 0.96);
  border-radius: 50%;
  box-shadow: 0 4px 12px rgba(75, 111, 255, 0.28);
}

.music-library__controls,
.music-player__controls,
.music-mini__controls {
  display: flex;
  gap: 18px;
  align-items: center;
  justify-content: center;
}

.music-library__controls button,
.music-player__controls button,
.music-mini button,
.music-mini__side button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  color: #172033;
  background: transparent;
  border: 0;
}

.music-library__play,
.music-player__play,
.music-mini__play {
  color: #fff !important;
  background: linear-gradient(135deg, #318dff, #8063ff) !important;
  border-radius: 50%;
  box-shadow: 0 14px 24px rgba(82, 98, 255, 0.28);
}

.music-library__play {
  width: 48px;
  height: 48px;
}

.music-player__play {
  width: 62px;
  height: 62px;
}

.music-mini__play {
  width: 50px;
  height: 50px;
}

.music-library__hero-lyrics {
  display: grid;
  justify-items: center;
  gap: 9px;
  color: #8a98ad;
  text-align: center;
}

.music-library__hero-lyrics strong {
  color: var(--music-blue);
  font-size: 16px;
}

.music-library__tools {
  display: grid;
  grid-template-columns: repeat(3, auto) minmax(0, 1fr) repeat(3, auto);
  gap: 9px;
  align-items: center;
}

.music-library__table {
  display: block;
  min-height: 0;
  overflow: auto;
  padding: 12px 14px;
  border-radius: 14px;
}

.music-library__table header,
.music-library__row {
  display: grid;
  grid-template-columns: 36px minmax(180px, 1.4fr) minmax(120px, 0.9fr) minmax(120px, 0.9fr) 70px 70px 80px;
  gap: 10px;
  align-items: center;
}

.music-library__table header {
  position: sticky;
  top: 0;
  z-index: 2;
  height: 34px;
  color: #7b889b;
  font-size: 11px;
  background: rgba(250, 253, 255, 0.86);
  backdrop-filter: blur(10px);
}

.music-library__row {
  width: 100%;
  height: 52px;
  min-height: 52px;
  margin: 2px 0;
  color: #526176;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 9px;
  text-align: left;
}

.music-library__mobile-codec {
  display: none;
}

.music-library__row.active {
  color: var(--music-blue);
  background: rgba(65, 139, 255, 0.08);
  border-color: rgba(65, 139, 255, 0.18);
}

.music-library__song {
  display: grid;
  grid-template-columns: 28px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
  min-width: 0;
}

.music-library__song img,
.music-library__song svg {
  width: 28px;
  height: 28px;
  border-radius: 7px;
}

.music-library__song strong {
  display: block;
  overflow: hidden;
  color: inherit;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-library__song em {
  display: none;
  margin-top: 3px;
  overflow: hidden;
  color: #718099;
  font-style: normal;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-library__empty {
  display: grid;
  place-items: center;
  gap: 8px;
  min-height: 180px;
  color: var(--music-muted);
}

.music-library__settings {
  display: grid;
  gap: 10px;
  padding: 14px;
  background: rgba(255, 255, 255, 0.56);
  border-radius: 14px;
}

.music-library__settings label {
  display: grid;
  gap: 7px;
}

.music-library__settings textarea {
  min-height: 90px;
  padding: 10px;
  border: 1px solid rgba(132, 155, 186, 0.14);
  border-radius: 10px;
  resize: vertical;
}

.music-app__details {
  display: grid;
  grid-template-rows: auto auto auto auto auto;
  align-content: start;
  gap: 14px;
  min-height: 0;
  padding: 14px;
  border-radius: 18px;
}

.music-app__details header {
  display: flex;
  gap: 28px;
  border-bottom: 1px solid rgba(132, 155, 186, 0.12);
}

.music-app__details header button {
  position: relative;
  padding: 0 0 10px;
  color: #66758d;
  background: transparent;
  border: 0;
}

.music-app__details header button.active {
  color: var(--music-blue);
}

.music-app__details header button.active::after {
  position: absolute;
  right: 0;
  bottom: 0;
  left: 0;
  height: 2px;
  content: "";
  background: var(--music-blue);
  border-radius: 99px;
}

.music-app__details section {
  display: grid;
  grid-template-columns: 96px minmax(0, 1fr);
  gap: 16px;
  align-items: center;
}

.music-app__details section img,
.music-app__details section > svg {
  width: 96px;
  height: 96px;
  border-radius: 10px;
}

.music-app__details h3 {
  margin: 0 0 8px;
  font-size: 18px;
}

.music-app__details p,
.music-app__details span,
.music-app__details dd,
.music-app__details article p {
  color: #64738a;
}

.music-app__details dl {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px 20px;
  margin: 0;
}

.music-app__details dt {
  color: #8a98ad;
  font-size: 11px;
}

.music-app__details dd {
  margin: 4px 0 0;
  font-size: 12px;
}

.music-app__details h4 {
  margin: 0 0 8px;
}

.music-app__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  align-items: flex-start;
  min-height: 0;
}

.music-app__tags span {
  flex: 0 0 auto;
  line-height: 1.2;
  padding: 4px 8px;
  color: #53637a;
  background: rgba(246, 249, 253, 0.8);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 8px;
}

.music-player {
  position: relative;
  display: grid;
  grid-template-rows: minmax(0, 1fr) 118px auto auto;
  gap: 12px;
  min-height: 0;
  overflow: hidden;
}

.music-player__mobile-tabs,
.music-player__mobile-now {
  display: none;
}

.music-player__mobile-back {
  position: absolute;
  top: 8px;
  left: 8px;
  z-index: 8;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 34px;
  height: 34px;
  padding: 0;
  color: #253348;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(132, 155, 186, 0.12);
  border-radius: 999px;
  box-shadow: 0 10px 24px rgba(54, 84, 130, 0.08);
}

.music-player__main {
  display: grid;
  grid-template-columns: minmax(220px, 340px) minmax(320px, 1fr) minmax(240px, 340px);
  gap: clamp(22px, 3vw, 42px);
  align-items: center;
  min-height: 0;
}

.music-player__cover-wrap {
  position: relative;
  justify-self: center;
  width: min(340px, 100%);
  cursor: pointer;
}

.music-player__cover {
  position: relative;
  z-index: 2;
  width: 100%;
  aspect-ratio: 1;
  border-radius: 18px;
  box-shadow: 0 28px 62px rgba(22, 39, 70, 0.18);
}

.music-player__vinyl {
  position: absolute;
  top: 12%;
  right: -14%;
  z-index: 1;
  width: 72%;
  aspect-ratio: 1;
  background:
    radial-gradient(circle, #111 0 7%, #303030 8% 10%, #111 11% 100%),
    repeating-radial-gradient(circle, #1b1b1d 0 5px, #111 6px 9px);
  border-radius: 50%;
  box-shadow: 0 18px 46px rgba(0, 0, 0, 0.26);
}

.music-player__info {
  position: relative;
  z-index: 3;
  display: grid;
  gap: 14px;
  min-width: 0;
  padding-left: 10px;
}

.music-player__desc {
  line-height: 1.8;
}

.music-player__actions {
  display: flex;
  gap: 12px;
}

.music-player__actions button {
  width: 36px;
  min-width: 36px;
  padding: 0;
}

.music-player__lyrics-panel {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  align-self: center;
  height: min(460px, 100%);
  min-height: 0;
  padding: 22px;
  border-radius: 28px;
}

.music-player__lyrics-panel header {
  display: flex;
  gap: 28px;
  border-bottom: 1px solid rgba(132, 155, 186, 0.12);
}

.music-player__lyrics-panel header button {
  padding: 0 0 12px;
  color: #66758d;
  background: transparent;
  border: 0;
}

.music-player__lyrics-panel header .active {
  color: var(--music-blue);
  border-bottom: 2px solid var(--music-blue);
}

.music-player__lyrics-scroll {
  display: grid;
  align-content: start;
  gap: 14px;
  min-height: 0;
  overflow-y: auto;
  padding: 44% 0;
  color: #9aa6b8;
  scroll-padding-block: 44%;
  text-align: center;
}

.music-player__lyrics-scroll p {
  margin: 0;
  font-size: 13px;
}

.music-player__lyrics-scroll .active {
  color: var(--music-blue);
  font-size: 19px;
  font-weight: 800;
}

.music-player__lyric-settings {
  justify-self: center;
}

.music-player__spectrum {
  align-self: end;
  min-height: 88px;
}

.music-mini {
  position: relative;
  display: grid;
  grid-column: 2 / 4;
  grid-template-columns: 72px minmax(180px, 0.7fr) auto auto minmax(220px, 1.15fr) max-content max-content;
  gap: 10px;
  align-items: center;
  min-width: 0;
  min-height: 0;
  padding: 12px 18px;
  border-radius: 18px;
  overflow: hidden;
}

.music-mini > *:not(.music-mini__spectrum) {
  position: relative;
  z-index: 1;
}

.music-mini__spectrum {
  position: absolute;
  right: 18px;
  bottom: 8px;
  left: 18px;
  z-index: 0;
  width: calc(100% - 36px);
  height: 42px;
  opacity: 0.42;
  pointer-events: none;
}

.music-app--player .music-mini {
  display: none;
}

.music-mini__cover {
  width: 64px;
  height: 64px;
  border-radius: 14px;
}

.music-mini__info {
  display: grid;
  gap: 4px;
  min-width: 0;
}

.music-mini__info strong,
.music-mini__info > span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-mini__info > span {
  color: #718099;
  font-size: 11px;
}

.music-mini__info small {
  overflow: hidden;
  color: var(--music-blue);
  font-size: 11px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.music-mini__progress {
  grid-column: 5;
  position: relative;
  min-width: 0;
}

.music-mini__side {
  display: grid;
  grid-column: 7;
  grid-template-columns: 16px 96px max-content;
  gap: 6px;
  align-items: center;
  justify-content: end;
  min-width: 0;
  width: max-content;
}

.music-mini__controls {
  grid-column: 6;
  min-width: max-content;
}

.music-mini__meta {
  display: none;
  gap: 6px;
  white-space: nowrap;
}

.music-mini__inline-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.music-mini__side button {
  gap: 8px;
  white-space: nowrap;
}

@container desktop-window-body (min-width: 1051px) and (max-width: 1320px) {
  .music-mini {
    grid-template-columns: 60px minmax(150px, 0.75fr) minmax(180px, 1fr) max-content max-content;
    gap: 8px;
    padding-inline: 14px;
  }

  .music-mini > button:not(.music-mini__cover) {
    display: none;
  }

  .music-mini__cover {
    width: 56px;
    height: 56px;
  }

  .music-mini__progress {
    grid-column: 3;
  }

  .music-mini__controls {
    grid-column: 4;
    gap: 8px;
  }

  .music-mini__play {
    width: 44px;
    height: 44px;
  }

  .music-mini__side {
    grid-column: 5;
    grid-template-columns: 16px 82px max-content;
    gap: 5px;
  }

  .music-mini__side button {
    padding-inline: 4px;
  }
}

@container desktop-window-body (max-width: 1050px) {
  .music-app {
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: minmax(0, 1fr) auto;
    padding: 16px;
  }

  .music-app.music-app--player {
    grid-template-rows: minmax(0, 1fr);
    padding: 0;
  }

  .music-app.music-app--player .music-app__stage {
    grid-template-rows: minmax(0, 1fr);
    overflow: hidden;
  }

  .music-app__sidebar,
  .music-app__details,
  .music-app__topbar,
  .music-library__tools,
  .music-library__hero,
  .music-library__table header {
    display: none;
  }

  .music-app__stage {
    min-height: 0;
    overflow: hidden;
  }

  .music-library {
    display: grid;
    grid-template-rows: auto auto auto minmax(0, 1fr);
    overflow: hidden;
  }

  .music-library__mobile-head {
    display: flex;
    align-items: center;
    justify-content: space-between;
  }

  .music-library__mobile-head h2 {
    margin: 0;
    font-size: 22px;
  }

  .music-library__mobile-head div {
    display: flex;
    gap: 14px;
  }

  .music-library__mobile-tabs {
    display: flex;
    gap: 24px;
    color: #66758d;
  }

  .music-library__mobile-tabs button {
    position: relative;
    padding: 0 0 8px;
    background: transparent;
    border: 0;
  }

  .music-library__mobile-tabs .active {
    color: var(--music-blue);
    font-weight: 800;
  }

  .music-library__mobile-tabs .active::after {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    height: 2px;
    content: "";
    background: var(--music-blue);
    border-radius: 99px;
  }

  .music-library__table {
    display: block;
    padding: 0;
    overflow-y: auto;
    background: transparent;
    border: 0;
    box-shadow: none;
  }

  .music-library__table::before {
    display: flex;
    align-items: center;
    width: max-content;
    min-height: 34px;
    padding: 0 14px;
    margin-bottom: 8px;
    color: #172033;
    content: "▶  全部播放";
    background: rgba(239, 245, 252, 0.86);
    border-radius: 18px;
  }

  .music-library__row {
    grid-template-columns: minmax(0, 1fr) auto;
    gap: 10px;
    height: 74px;
    min-height: 74px;
    padding: 8px 10px;
    border-bottom: 1px solid rgba(132, 155, 186, 0.1);
  }

  .music-library__row > span:first-child {
    display: none;
  }

  .music-library__row > span:nth-child(3),
  .music-library__row > span:nth-child(4),
  .music-library__row > span:nth-child(5),
  .music-library__row > span:nth-child(6),
  .music-library__row > span:nth-child(7) {
    display: none;
  }

  .music-library__song {
    grid-template-columns: 54px minmax(0, 1fr);
    min-width: 0;
  }

  .music-library__song em {
    display: block;
  }

  .music-library__song img,
  .music-library__song svg {
    width: 54px;
    height: 54px;
    border-radius: 11px;
  }

  .music-library__mobile-codec {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    min-width: 42px;
    height: 24px;
    padding: 0 8px;
    color: var(--music-blue);
    background: rgba(62, 137, 255, 0.1);
    border: 1px solid rgba(62, 137, 255, 0.12);
    border-radius: 8px;
    font-size: 11px;
    font-weight: 700;
  }

  .music-mini {
    display: grid;
    grid-column: 1;
    grid-template-columns: 52px minmax(0, 1fr);
    grid-template-rows: auto auto auto;
    gap: 10px 12px;
    align-items: center;
    padding: 12px 14px;
    border-radius: 18px;
  }

  .music-mini__spectrum {
    right: 12px;
    bottom: 66px;
    left: 12px;
    width: calc(100% - 24px);
    height: 46px;
    opacity: 0.36;
  }

  .music-mini__cover {
    width: 52px;
    height: 52px;
  }

  .music-mini__info {
    align-self: center;
  }

  .music-mini__info small {
    white-space: normal;
  }

  .music-mini__inline-meta {
    display: flex;
  }

  .music-mini > button:not(.music-mini__cover) {
    display: none;
  }

  .music-mini__progress {
    grid-column: 1 / -1;
    grid-template-columns: auto minmax(0, 1fr) auto;
  }

  .music-mini__controls {
    grid-column: 1 / -1;
    justify-content: center;
    gap: 28px;
  }

  .music-mini__side {
    display: none;
  }

  .music-app--player .music-app__stage {
    overflow: hidden;
  }

  .music-app--player .music-mini {
    display: none;
  }

  .music-player {
    position: relative;
    grid-template-rows: auto auto auto auto minmax(52px, 1fr) 26px 72px;
    gap: 6px;
    height: 100%;
    min-height: 0;
    align-content: stretch;
    justify-content: stretch;
    overflow: hidden;
    padding-bottom: 0;
  }

  .music-player__mobile-back,
  .music-player__mobile-tabs {
    display: flex;
  }

  .music-player__mobile-back {
    position: static;
    width: max-content;
    height: auto;
    background: transparent;
    border: 0;
    box-shadow: none;
  }

  .music-player__mobile-tabs {
    justify-content: center;
    gap: 34px;
  }

  .music-player__mobile-tabs button {
    position: relative;
    padding: 0 0 8px;
    color: #66758d;
    background: transparent;
    border: 0;
  }

  .music-player__mobile-tabs .active {
    color: var(--music-blue);
    font-weight: 800;
  }

  .music-player__mobile-tabs .active::after {
    position: absolute;
    right: 0;
    bottom: 0;
    left: 0;
    height: 2px;
    content: "";
    background: var(--music-blue);
    border-radius: 99px;
  }

  .music-player__main {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    grid-template-rows: auto;
    align-content: center;
    justify-items: center;
    gap: 0;
    min-height: 0;
    overflow: visible;
  }

  .music-player__cover-wrap {
    width: min(280px, 76%);
    aspect-ratio: 1 / 1;
    margin-right: 0;
  }

  .music-player__cover {
    width: 100%;
    height: 100%;
    aspect-ratio: 1 / 1;
    border-radius: 18px;
  }

  .music-player__vinyl {
    display: none;
  }

  .music-player__info {
    display: none;
  }

  .music-player__mobile-now {
    display: grid;
    justify-items: start;
    width: min(280px, 76%);
    min-height: 58px;
    margin-inline: auto;
    gap: 4px;
  }

  .music-player__mobile-now h2 {
    max-width: 100%;
    margin: 0;
    overflow: hidden;
    color: var(--music-ink);
    font-size: 18px;
    line-height: 1.15;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .music-player__mobile-now p {
    max-width: 100%;
    margin: 0;
    overflow: hidden;
    color: var(--music-muted);
    font-size: 12px;
    line-height: 1.3;
    text-overflow: ellipsis;
    white-space: nowrap;
  }

  .music-player__badges {
    gap: 6px;
  }

  .music-player__badges span {
    padding: 2px 7px;
    font-size: 10px;
  }

  .music-player__lyrics-panel {
    display: none;
  }

  .music-player__main--lyrics .music-player__cover-wrap,
  .music-player__main--lyrics .music-player__info,
  .music-player__main--lyrics + .music-player__mobile-now,
  .music-player__main--lyrics + .music-player__mobile-now + .music-player__spectrum {
    display: none;
  }

  .music-player__main--lyrics .music-player__lyrics-panel {
    display: grid;
    width: 100%;
    height: 100%;
    padding: 0;
    background: transparent;
    border: 0;
    box-shadow: none;
  }

  .music-player__main--lyrics .music-player__lyrics-panel header,
  .music-player__main--lyrics .music-player__lyric-settings {
    display: none;
  }

  .music-player__main--lyrics .music-player__lyrics-scroll {
    gap: 18px;
    padding: 48% 0;
    scroll-padding-block: 48%;
  }

  .music-player__main--lyrics .music-player__lyrics-scroll .active {
    font-size: 20px;
  }

  .music-player__spectrum {
    align-self: stretch;
    height: 100%;
    min-height: 52px;
    margin-bottom: 0;
  }

  .music-player__progress {
    align-self: end;
    margin-bottom: 0;
  }

  .music-player__controls {
    position: static;
    align-self: stretch;
    align-items: center;
    justify-content: space-between;
    height: 72px;
    min-height: 72px;
    padding: 0 8px 4px;
    margin-bottom: 0;
    overflow: visible;
  }

  .music-player__play {
    width: 56px;
    height: 56px;
    flex: 0 0 56px;
  }
}
</style>
