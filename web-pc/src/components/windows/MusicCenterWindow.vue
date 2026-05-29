<script setup lang="ts">
import { computed, nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import {
  Album,
  AudioLines,
  Disc3,
  FolderCog,
  Heart,
  Library,
  List,
  ListMusic,
  Maximize2,
  MoreHorizontal,
  Pause,
  Play,
  RefreshCcw,
  Repeat,
  Search,
  Shuffle,
  SkipBack,
  SkipForward,
  Volume2,
  X,
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
const isLoading = ref(false);
const isScanning = ref(false);
const isPlaying = ref(false);
const isPlayerFocusMode = ref(false);
const playMode = ref<'repeat' | 'sequence' | 'shuffle'>('sequence');
const currentTime = ref(0);
const duration = ref(0);
const volume = ref(0.82);
const audioRef = ref<HTMLAudioElement | null>(null);
const bottomSpectrumRef = ref<HTMLCanvasElement | null>(null);
const focusSpectrumRef = ref<HTMLCanvasElement | null>(null);

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
const playbackProgress = computed(() => (duration.value > 0 ? Math.min(100, (currentTime.value / duration.value) * 100) : 0));
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
const focusLyrics = computed(() => {
  const lines = parsedLyrics.value;
  if (!lines.length) return [];
  const active = Math.max(0, activeLyricIndex.value);
  const start = Math.max(0, active - 4);
  return lines.slice(start, Math.min(lines.length, active + 6)).map((line, offset) => ({
    ...line,
    sourceIndex: start + offset,
  }));
});

watch(selectedTrack, async (track) => {
  lyricsText.value = '';
  currentTime.value = 0;
  duration.value = 0;
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

onMounted(() => {
  void loadMusic();
  void nextTick(startSpectrum);
});

onBeforeUnmount(() => {
  if (spectrumFrame) {
    cancelAnimationFrame(spectrumFrame);
    spectrumFrame = 0;
  }
  if (audioContext) {
    void audioContext.close();
  }
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
  if (playNow) {
    setTimeout(() => void playSelected(), 0);
  }
}

async function playSelected() {
  const track = selectedTrack.value;
  const audio = audioRef.value;
  if (!track || !audio) return;
  const streamUrl = apiClient.music.streamUrl(track.id);
  const absoluteStreamUrl = new URL(streamUrl, window.location.href).href;
  if (audio.src !== absoluteStreamUrl) {
    audio.src = streamUrl;
  }
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

function handleLoadedMetadata() {
  duration.value = audioRef.value?.duration || 0;
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

function toggleFocusPlayer(force?: boolean) {
  isPlayerFocusMode.value = typeof force === 'boolean' ? force : !isPlayerFocusMode.value;
  void nextTick(startSpectrum);
}

async function setupAudioAnalyser() {
  const audio = audioRef.value;
  const AudioContextCtor =
    window.AudioContext || (window as typeof window & { webkitAudioContext?: typeof AudioContext }).webkitAudioContext;
  if (!audio || !AudioContextCtor) return;
  if (!audioContext) {
    audioContext = new AudioContextCtor();
  }
  if (!mediaSource) {
    analyser = audioContext.createAnalyser();
    analyser.fftSize = 512;
    analyser.smoothingTimeConstant = 0.76;
    frequencyData = new Uint8Array(analyser.frequencyBinCount);
    mediaSource = audioContext.createMediaElementSource(audio);
    mediaSource.connect(analyser);
    analyser.connect(audioContext.destination);
  }
  if (audioContext.state === 'suspended') {
    await audioContext.resume();
  }
  startSpectrum();
}

function startSpectrum() {
  if (spectrumFrame) return;
  const draw = () => {
    drawSpectrum(bottomSpectrumRef.value, 220, false);
    drawSpectrum(focusSpectrumRef.value, 180, true);
    spectrumFrame = requestAnimationFrame(draw);
  };
  spectrumFrame = requestAnimationFrame(draw);
}

function drawSpectrum(canvas: HTMLCanvasElement | null, barCount: number, focusMode: boolean) {
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
  const gap = focusMode ? 2.3 * dpr : 1.7 * dpr;
  const barWidth = Math.max(1.1 * dpr, (width - gap * (barCount - 1)) / barCount);
  const maxBarHeight = height * (focusMode ? 0.74 : 0.42);
  const baseY = height - (focusMode ? 12 * dpr : 15 * dpr);
  for (let i = 0; i < barCount; i += 1) {
    const raw = values[i] ?? 0;
    const scaled = Math.max(0.08, raw) * maxBarHeight;
    const x = i * (barWidth + gap);
    const y = baseY - scaled;
    const hue = 202 + (i / Math.max(1, barCount - 1)) * 92;
    const gradient = ctx.createLinearGradient(0, y, 0, baseY);
    gradient.addColorStop(0, `hsla(${hue}, 96%, ${focusMode ? 66 : 59}%, ${focusMode ? 0.9 : 0.78})`);
    gradient.addColorStop(1, `hsla(${hue + 28}, 92%, 70%, ${focusMode ? 0.45 : 0.22})`);
    ctx.fillStyle = gradient;
    ctx.fillRect(x, y, barWidth, scaled);
    if (focusMode || i % 3 === 0) {
      ctx.fillStyle = `hsla(${hue + 18}, 95%, 68%, ${focusMode ? 0.38 : 0.26})`;
      ctx.fillRect(x, Math.max(0, y - 4 * dpr), barWidth, 1.2 * dpr);
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
      values[i] = Math.min(1, Math.max(0.05, sum / Math.max(1, end - start) / 255));
    }
    return values;
  }
  const now = performance.now() / 1000;
  for (let i = 0; i < barCount; i += 1) {
    const wave = Math.sin(now * 1.8 + i * 0.31) * 0.5 + Math.sin(now * 0.72 + i * 0.11) * 0.5;
    values[i] = 0.11 + Math.max(0, wave) * 0.2;
  }
  return values;
}

function formatTime(value: number) {
  if (!Number.isFinite(value) || value <= 0) return '00:00';
  const minutes = Math.floor(value / 60);
  const seconds = Math.floor(value % 60);
  return `${String(minutes).padStart(2, '0')}:${String(seconds).padStart(2, '0')}`;
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
  <div class="music-center" :class="{ 'music-center--focus': isPlayerFocusMode }">
    <audio
      ref="audioRef"
      preload="metadata"
      @play="handlePlay"
      @pause="handlePause"
      @ended="playNext"
      @loadedmetadata="handleLoadedMetadata"
      @timeupdate="handleTimeUpdate"
    />

    <template v-if="isPlayerFocusMode">
      <section class="music-center__focus-player panel">
        <button class="music-center__focus-close" type="button" aria-label="退出播放器" @click="toggleFocusPlayer(false)">
          <X :size="16" />
        </button>
        <div class="music-center__focus-main">
          <button class="music-center__focus-cover" type="button" @click="toggleFocusPlayer(false)">
            <img v-if="coverUrl(selectedTrack)" :src="coverUrl(selectedTrack)" :alt="selectedTrack?.album ?? '专辑封面'" />
            <Disc3 v-else :size="58" />
          </button>
          <div class="music-center__focus-info">
            <p>播放中</p>
            <h3>{{ selectedTrack?.title ?? '未选择曲目' }}</h3>
            <span>{{ selectedTrack?.artist ?? '未知艺术家' }} · {{ selectedTrack?.album ?? '未归类专辑' }}</span>
            <small>{{ selectedTrack?.codec ?? 'AUTO' }} · {{ selectedTrack?.size ?? albumSummary }}</small>
          </div>
        </div>
        <div class="music-center__focus-spectrum" aria-hidden="true">
          <canvas ref="focusSpectrumRef" />
        </div>
        <div class="music-center__focus-progress">
          <span>{{ formatTime(currentTime) }}</span>
          <input
            type="range"
            min="0"
            :max="duration || 0"
            :value="currentTime"
            :style="{ '--progress': `${playbackProgress}%` }"
            aria-label="播放进度"
            @input="seek"
          />
          <span>{{ formatTime(duration) }}</span>
        </div>
        <div class="music-center__focus-controls">
          <button type="button" aria-label="上一首" @click="playPrevious"><SkipBack :size="18" /></button>
          <button class="music-center__play music-center__play--large" type="button" :aria-label="isPlaying ? '暂停' : '播放'" @click="togglePlay">
            <Pause v-if="isPlaying" :size="24" />
            <Play v-else :size="24" />
          </button>
          <button type="button" aria-label="下一首" @click="playNext"><SkipForward :size="18" /></button>
          <div class="music-center__volume music-center__focus-volume">
            <Volume2 :size="15" />
            <input v-model.number="volume" type="range" min="0" max="1" step="0.01" aria-label="音量" />
          </div>
        </div>
        <div class="music-center__focus-lyrics">
          <template v-if="focusLyrics.length">
            <p
              v-for="line in focusLyrics"
              :key="`${line.time}-${line.sourceIndex}`"
              :class="{ 'music-center__lyric--active': line.sourceIndex === activeLyricIndex }"
            >
              {{ line.text }}
            </p>
          </template>
          <template v-else>
            <strong>{{ selectedTrack?.album ?? '未选择专辑' }}</strong>
            <span>{{ selectedTrack?.status ?? '同名 .lrc 文件会自动显示在这里。' }}</span>
          </template>
        </div>
      </section>
    </template>

    <template v-else>
      <section class="music-center__layout">
        <aside class="music-center__sidebar panel">
          <div class="music-center__search">
            <Search :size="16" />
            <input v-model="search" placeholder="搜索歌曲、歌手、专辑" @keyup.enter="refreshTracks" />
            <button type="button" @click="refreshTracks">搜索</button>
          </div>
          <nav>
            <button type="button" :class="{ 'music-center__nav--active': activeView === 'tracks' }" @click="activeView = 'tracks'">
              <ListMusic :size="17" /> 曲目列表
            </button>
            <button type="button" :class="{ 'music-center__nav--active': activeView === 'albums' }" @click="activeView = 'albums'">
              <Album :size="17" /> 专辑视图
            </button>
            <button type="button" :class="{ 'music-center__nav--active': activeView === 'library' }" @click="activeView = 'library'">
              <FolderCog :size="17" /> 媒体库
            </button>
          </nav>
          <div class="music-center__status">
            <strong>媒体库状态</strong>
            <span>{{ statusMessage }}</span>
            <small>上次扫描：{{ formatDate(settings.lastScanAt) }}</small>
          </div>
        </aside>

        <main class="music-center__main panel">
          <template v-if="activeView === 'tracks'">
            <header>
              <div>
                <p>曲目</p>
                <h3>{{ filteredTracks.length }} 首可播放</h3>
              </div>
              <button type="button" :disabled="isScanning" @click="scanLibrary">
                <RefreshCcw :size="16" /> {{ isScanning ? '扫描中' : '重新扫描' }}
              </button>
            </header>
            <div v-if="filteredTracks.length" class="music-center__tracks">
              <button
                v-for="track in filteredTracks"
                :key="track.id"
                type="button"
                class="music-center__track"
                :class="{ 'music-center__track--active': track.id === selectedTrack?.id }"
                @click="selectTrack(track, true)"
              >
                <span class="music-center__track-cover">
                  <img v-if="coverUrl(track)" :src="coverUrl(track)" :alt="track.album" />
                  <AudioLines v-else :size="17" />
                </span>
                <span>
                  <strong>{{ track.title }}</strong>
                  <small>{{ track.artist }} · {{ track.album }}</small>
                </span>
                <b>{{ track.codec }}</b>
                <small>{{ track.size }}</small>
              </button>
            </div>
            <div v-else class="music-center__empty">
              <Library :size="34" />
              <strong>当前没有曲目</strong>
              <span>设置媒体库路径后执行扫描，支持 MP3、FLAC、M4A、AAC、WAV、OGG、Opus。</span>
            </div>
          </template>

          <template v-else-if="activeView === 'albums'">
            <header>
              <div>
                <p>专辑</p>
                <h3>{{ albumSummary }}</h3>
              </div>
            </header>
            <div class="music-center__albums">
              <button
                v-for="album in albums"
                :key="album.id"
                type="button"
                class="music-center__album"
                @click="activeView = 'tracks'; search = album.name"
              >
                <span><Disc3 :size="24" /></span>
                <strong>{{ album.name }}</strong>
                <small>{{ album.artist }} · {{ album.count }} 首</small>
              </button>
            </div>
          </template>

          <template v-else>
            <header>
              <div>
                <p>媒体库设置</p>
                <h3>扫描本机或 NAS 目录中的音乐文件</h3>
              </div>
            </header>
            <label class="music-center__library-paths">
              <span>媒体库路径，每行一个目录</span>
              <textarea v-model="libraryPathText" placeholder="/srv/higoos/nas/Music&#10;/srv/higoos/nas/家庭空间/音乐" />
            </label>
            <label class="music-center__toggle">
              <input v-model="settings.autoScan" type="checkbox" />
              <span>保存后自动扫描</span>
            </label>
            <div class="music-center__library-actions">
              <button type="button" :disabled="isLoading" @click="saveLibrary">保存媒体库</button>
              <button type="button" :disabled="isScanning" @click="scanLibrary">
                <RefreshCcw :size="16" /> {{ isScanning ? '扫描中' : '立即扫描' }}
              </button>
            </div>
          </template>
        </main>

        <aside class="music-center__lyrics panel">
          <header>
            <p>歌词 / 专辑</p>
            <h3>{{ selectedTrack?.album ?? '未选择专辑' }}</h3>
          </header>
          <div v-if="parsedLyrics.length" class="music-center__lyric-lines">
            <p
              v-for="(line, index) in parsedLyrics"
              :key="`${line.time}-${index}`"
              :class="{ 'music-center__lyric--active': index === activeLyricIndex }"
            >
              {{ line.text }}
            </p>
          </div>
          <div v-else class="music-center__album-info">
            <Disc3 :size="34" />
            <strong>{{ selectedTrack?.title ?? '等待播放' }}</strong>
            <span>{{ selectedTrack?.status ?? '同名 .lrc 文件会自动显示在这里。' }}</span>
            <small v-for="track in selectedAlbumTracks.slice(0, 6)" :key="track.id">{{ track.title }}</small>
          </div>
        </aside>
      </section>

      <section class="music-center__bottom-player panel">
        <canvas ref="bottomSpectrumRef" class="music-center__bottom-spectrum" aria-hidden="true" />
        <button class="music-center__bottom-cover" type="button" aria-label="打开沉浸播放器" @click="toggleFocusPlayer(true)">
          <img v-if="coverUrl(selectedTrack)" :src="coverUrl(selectedTrack)" :alt="selectedTrack?.album ?? '专辑封面'" />
          <Disc3 v-else :size="26" />
          <Maximize2 :size="12" />
        </button>
        <div class="music-center__bottom-info">
          <p>正在播放</p>
          <h3>{{ selectedTrack?.title ?? '未选择曲目' }}</h3>
          <span>{{ selectedTrack?.artist ?? '设置媒体库后开始播放' }} · {{ selectedTrack?.album ?? albumSummary }}</span>
        </div>
        <div class="music-center__bottom-center">
          <div class="music-center__bottom-progress">
            <span>{{ formatTime(currentTime) }}</span>
            <input
              type="range"
              min="0"
              :max="duration || 0"
              :value="currentTime"
              :style="{ '--progress': `${playbackProgress}%` }"
              aria-label="播放进度"
              @input="seek"
            />
            <span>{{ formatTime(duration) }}</span>
          </div>
          <div class="music-center__transport">
            <button type="button" aria-label="上一首" @click="playPrevious"><SkipBack :size="15" /></button>
            <button class="music-center__play" type="button" :aria-label="isPlaying ? '暂停' : '播放'" @click="togglePlay">
              <Pause v-if="isPlaying" :size="18" />
              <Play v-else :size="18" />
            </button>
            <button type="button" aria-label="下一首" @click="playNext"><SkipForward :size="15" /></button>
          </div>
        </div>
        <div class="music-center__bottom-side">
          <div class="music-center__volume">
            <Volume2 :size="15" />
            <input v-model.number="volume" type="range" min="0" max="1" step="0.01" aria-label="音量" />
          </div>
          <div class="music-center__bottom-meta">
            <span>{{ settings.trackCount }} 曲目</span>
            <span>{{ albums.length }} 专辑</span>
            <span>{{ selectedTrack?.codec ?? 'AUTO' }}</span>
          </div>
        </div>
      </section>
    </template>
  </div>
</template>

<style scoped>
.music-center {
  display: grid;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-width: 0;
  min-height: 0;
  color: var(--text);
  font-size: 12px;
}

.music-center--focus {
  grid-template-rows: minmax(0, 1fr);
}

.music-center__layout {
  display: grid;
  grid-template-columns: 220px minmax(0, 1.35fr) minmax(240px, 0.8fr);
  gap: 12px;
  align-items: stretch;
  min-height: 0;
  overflow: auto;
}

.music-center__sidebar,
.music-center__main,
.music-center__lyrics {
  display: grid;
  align-content: start;
  gap: 12px;
  min-width: 0;
  padding: 11px;
}

.music-center__sidebar {
  grid-template-rows: auto auto minmax(0, 1fr);
}

.music-center__search {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  padding: 8px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(100, 136, 166, 0.16);
  border-radius: var(--radius-sm);
}

.music-center__search input,
.music-center__library-paths textarea {
  width: 100%;
  color: var(--text-strong);
  background: transparent;
  border: 0;
  outline: 0;
  font-size: 12px;
}

.music-center__sidebar nav {
  display: grid;
  gap: 8px;
}

.music-center__sidebar nav button {
  display: flex;
  align-items: center;
  gap: 8px;
  min-height: 36px;
  padding: 0 12px;
  color: var(--text);
  background: transparent;
  border-radius: var(--radius-sm);
  font-size: 12px;
  text-align: left;
}

.music-center__nav--active {
  color: var(--accent) !important;
  background: rgba(19, 136, 255, 0.12) !important;
}

.music-center__status {
  display: grid;
  align-self: end;
  gap: 6px;
  padding: 12px;
  background: rgba(231, 247, 255, 0.76);
  border: 1px solid rgba(34, 211, 238, 0.18);
  border-radius: var(--radius-sm);
}

.music-center__status strong,
.music-center__track strong,
.music-center__album strong,
.music-center__album-info strong {
  color: var(--text-strong);
  font-size: 12px;
}

.music-center__status span,
.music-center__status small,
.music-center__track small,
.music-center__album small,
.music-center__album-info span,
.music-center__album-info small {
  overflow: hidden;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.music-center__main header,
.music-center__lyrics header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
}

.music-center__main p,
.music-center__lyrics p,
.music-center__bottom-info p,
.music-center__focus-info p {
  margin: 0;
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 760;
}

.music-center__main h3,
.music-center__lyrics h3 {
  margin: 4px 0;
  color: var(--text-strong);
  font-size: 12px;
  line-height: 1.2;
}

.music-center__search button,
.music-center__main header button,
.music-center__library-actions button,
.music-center__transport button,
.music-center__focus-controls button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 7px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.82);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 760;
}

.music-center__tracks {
  display: grid;
  gap: 8px;
}

.music-center__track {
  display: grid;
  grid-template-columns: 44px minmax(0, 1fr) auto auto;
  gap: 10px;
  align-items: center;
  min-height: 58px;
  padding: 8px;
  color: var(--text);
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
  text-align: left;
}

.music-center__track--active {
  border-color: rgba(19, 136, 255, 0.36);
  background: rgba(231, 247, 255, 0.82);
}

.music-center__track-cover {
  display: grid;
  width: 40px;
  height: 40px;
  place-items: center;
  overflow: hidden;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-radius: 10px;
}

.music-center__track-cover img,
.music-center__bottom-cover img,
.music-center__focus-cover img {
  width: 100%;
  height: 100%;
  object-fit: cover;
}

.music-center__track span:nth-child(2) {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.music-center__track b {
  color: var(--accent);
  font-size: 11px;
}

.music-center__albums {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
  gap: 10px;
}

.music-center__album {
  display: grid;
  gap: 8px;
  min-height: 118px;
  align-content: center;
  justify-items: start;
  padding: 12px;
  color: var(--text);
  text-align: left;
  background: rgba(255, 255, 255, 0.62);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-md);
}

.music-center__album span {
  display: grid;
  width: 44px;
  height: 44px;
  place-items: center;
  color: #fff;
  background: linear-gradient(135deg, #22d3ee, #1687ff);
  border-radius: 14px;
}

.music-center__library-paths {
  display: grid;
  gap: 8px;
  color: var(--text-muted);
  font-size: 11px;
}

.music-center__library-paths textarea {
  min-height: 96px;
  resize: vertical;
  padding: 10px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid rgba(100, 136, 166, 0.18);
  border-radius: var(--radius-sm);
}

.music-center__toggle {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--text);
}

.music-center__library-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
}

.music-center__lyric-lines {
  display: grid;
  gap: 9px;
  padding: 8px 2px;
}

.music-center__lyric-lines p {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.5;
}

.music-center__lyric-lines .music-center__lyric--active,
.music-center__focus-lyrics .music-center__lyric--active {
  color: var(--accent);
  font-size: 13px;
  font-weight: 800;
}

.music-center__album-info,
.music-center__empty {
  display: grid;
  gap: 9px;
  align-content: center;
  justify-items: center;
  min-height: 190px;
  color: var(--text-muted);
  text-align: center;
}

.music-center__album-info small {
  max-width: 100%;
}

.music-center__bottom-player {
  position: relative;
  display: grid;
  grid-template-columns: 62px minmax(150px, 0.8fr) minmax(260px, 1.6fr) minmax(168px, 0.55fr);
  gap: 12px;
  align-items: center;
  min-height: 112px;
  overflow: hidden;
  padding: 12px;
  isolation: isolate;
}

.music-center__bottom-spectrum {
  position: absolute;
  inset: 0;
  z-index: -1;
  width: 100%;
  height: 100%;
  opacity: 0.9;
  pointer-events: none;
}

.music-center__bottom-player::after {
  position: absolute;
  inset: 0;
  z-index: -1;
  content: "";
  background:
    linear-gradient(90deg, rgba(255, 255, 255, 0.9), rgba(255, 255, 255, 0.72)),
    radial-gradient(circle at 12% 50%, rgba(34, 211, 238, 0.2), transparent 36%),
    radial-gradient(circle at 92% 0%, rgba(124, 58, 237, 0.16), transparent 34%);
}

.music-center__bottom-cover {
  position: relative;
  display: grid;
  width: 56px;
  height: 56px;
  place-items: center;
  overflow: hidden;
  color: #fff;
  background: linear-gradient(135deg, #12bfe6, #1677ff 52%, #7c3aed);
  border: 0;
  border-radius: 14px;
  box-shadow: 0 10px 22px rgba(13, 72, 130, 0.14);
}

.music-center__bottom-cover > svg:last-child {
  position: absolute;
  right: 5px;
  bottom: 5px;
  padding: 2px;
  color: #fff;
  background: rgba(15, 23, 42, 0.32);
  border-radius: 6px;
}

.music-center__bottom-info {
  display: grid;
  gap: 3px;
  min-width: 0;
}

.music-center__bottom-info h3 {
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 15px;
  line-height: 1.2;
}

.music-center__bottom-info span {
  overflow: hidden;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  font-size: 11px;
}

.music-center__bottom-center {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.music-center__bottom-progress,
.music-center__focus-progress {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  gap: 8px;
  align-items: center;
  min-width: 0;
  color: var(--text);
  font-size: 11px;
}

.music-center input[type="range"] {
  width: 100%;
  min-width: 0;
  accent-color: #1687ff;
}

.music-center__transport {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  width: max-content;
  justify-self: center;
}

.music-center__transport button:not(.music-center__play),
.music-center__focus-controls button:not(.music-center__play) {
  width: 32px;
  min-width: 32px;
  padding: 0;
}

.music-center__play {
  width: 38px;
  min-width: 38px;
  height: 38px;
  color: #fff !important;
  background: #1687ff !important;
  border-color: transparent !important;
  border-radius: 50% !important;
}

.music-center__play--large {
  width: 50px;
  min-width: 50px;
  height: 50px;
}

.music-center__bottom-side {
  display: grid;
  gap: 8px;
  min-width: 0;
}

.music-center__volume {
  display: grid;
  grid-template-columns: auto minmax(72px, 1fr);
  gap: 6px;
  align-items: center;
  min-width: 0;
}

.music-center__volume svg {
  flex: 0 0 auto;
}

.music-center__bottom-meta {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 5px;
}

.music-center__bottom-meta span {
  overflow: hidden;
  padding: 5px 6px;
  color: var(--text);
  text-overflow: ellipsis;
  white-space: nowrap;
  background: rgba(255, 255, 255, 0.52);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: 8px;
  font-size: 11px;
  text-align: center;
}

.music-center__focus-player {
  position: relative;
  display: grid;
  grid-template-rows: auto minmax(128px, 1fr) auto auto minmax(96px, 0.45fr);
  gap: 14px;
  min-height: 0;
  overflow: hidden;
  padding: 18px;
  background:
    radial-gradient(circle at 18% 8%, rgba(34, 211, 238, 0.2), transparent 34%),
    radial-gradient(circle at 86% 20%, rgba(124, 58, 237, 0.16), transparent 30%),
    rgba(255, 255, 255, 0.86);
}

.music-center__focus-close {
  position: absolute;
  top: 14px;
  right: 14px;
  display: grid;
  width: 34px;
  height: 34px;
  place-items: center;
  color: var(--text);
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid rgba(100, 136, 166, 0.16);
  border-radius: 50%;
}

.music-center__focus-main {
  display: grid;
  grid-template-columns: minmax(150px, 240px) minmax(0, 1fr);
  gap: 22px;
  align-items: center;
  min-width: 0;
}

.music-center__focus-cover {
  display: grid;
  width: min(240px, 100%);
  aspect-ratio: 1;
  place-items: center;
  overflow: hidden;
  color: #fff;
  background: linear-gradient(135deg, #12bfe6, #1677ff 52%, #7c3aed);
  border: 0;
  border-radius: 20px;
  box-shadow: 0 20px 48px rgba(13, 72, 130, 0.18);
}

.music-center__focus-info {
  display: grid;
  gap: 7px;
  min-width: 0;
}

.music-center__focus-info h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 18px;
  line-height: 1.15;
}

.music-center__focus-info span,
.music-center__focus-info small,
.music-center__focus-lyrics span {
  color: var(--text-muted);
  font-size: 12px;
}

.music-center__focus-spectrum {
  position: relative;
  min-height: 140px;
  overflow: hidden;
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.48);
  border: 1px solid rgba(100, 136, 166, 0.12);
}

.music-center__focus-spectrum canvas {
  width: 100%;
  height: 100%;
}

.music-center__focus-controls {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  justify-content: center;
  gap: 10px;
}

.music-center__focus-volume {
  width: min(180px, 100%);
  margin-left: 8px;
}

.music-center__focus-lyrics {
  display: grid;
  align-content: center;
  justify-items: center;
  gap: 8px;
  min-height: 0;
  padding: 12px;
  overflow: hidden;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.56);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-md);
  text-align: center;
}

.music-center__focus-lyrics p {
  margin: 0;
  font-size: 12px;
  line-height: 1.5;
}

@container desktop-window-body (max-width: 980px) {
  .music-center__layout {
    grid-template-columns: minmax(0, 1fr);
    align-items: start;
  }

  .music-center__bottom-player {
    grid-template-columns: 54px minmax(0, 1fr) auto;
  }

  .music-center__bottom-center {
    grid-column: 1 / -1;
  }

  .music-center__bottom-side {
    grid-column: 3;
    grid-row: 1;
    width: 170px;
  }

  .music-center__bottom-meta {
    display: none;
  }
}

@container desktop-window-body (max-width: 640px) {
  .music-center__bottom-player {
    grid-template-columns: 50px minmax(0, 1fr);
    min-height: 154px;
  }

  .music-center__bottom-cover {
    width: 50px;
    height: 50px;
  }

  .music-center__bottom-side {
    grid-column: 1 / -1;
    grid-row: auto;
    width: 100%;
  }

  .music-center__volume {
    grid-template-columns: auto minmax(96px, 180px);
    justify-content: end;
  }

  .music-center__track {
    grid-template-columns: 42px minmax(0, 1fr);
  }

  .music-center__track b,
  .music-center__track > small {
    display: none;
  }

  .music-center__focus-main {
    grid-template-columns: minmax(0, 1fr);
    justify-items: center;
    text-align: center;
  }

  .music-center__focus-cover {
    max-width: 180px;
  }
}
</style>
