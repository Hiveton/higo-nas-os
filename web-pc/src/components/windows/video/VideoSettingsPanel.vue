<script setup lang="ts">
import { FolderOpen, ListVideo, Plus, Radio, Settings, Trash2 } from 'lucide-vue-next';
import { UiButton, UiCheckbox, UiEmptyState, UiFormField, UiInput, UiSelect, UiSwitch } from '../../ui';

const libraryTypeOptions = [
  { label: '电影', value: 'movie' },
  { label: '电视剧', value: 'series' },
  { label: '混合', value: 'mixed' },
];
import type {
  DvrSettings,
  LiveGuideSource,
  LiveProgram,
  LiveSource,
  RecordingTimer,
  VideoLibrary,
} from '../../../api/types';

interface NewLibraryForm {
  name: string;
  type: string;
  path: string;
  metadataLanguage: string;
  allowAdultContent: boolean;
  autoSubtitles: boolean;
  subtitleLanguage: string;
}

interface NewLiveSourceForm {
  name: string;
  url: string;
  userAgent: string;
  streamLimit: number;
}

interface NewGuideSourceForm {
  name: string;
  url: string;
  userAgent: string;
}

defineProps<{
  libraries: VideoLibrary[];
  liveSources: LiveSource[];
  liveChannels: { id: string }[];
  guideSources: LiveGuideSource[];
  livePrograms: LiveProgram[];
  recordingTimers: RecordingTimer[];
  newLibrary: NewLibraryForm;
  newLiveSource: NewLiveSourceForm;
  newGuideSource: NewGuideSourceForm;
  dvrSettings: DvrSettings;
  typeLabel: (type: string) => string;
}>();

const emit = defineEmits<{
  (e: 'create-library'): void;
  (e: 'delete-library', library: VideoLibrary): void;
  (e: 'create-live-source'): void;
  (e: 'create-guide-source'): void;
  (e: 'save-dvr-settings'): void;
}>();
</script>

<template>
  <section class="settings-view">
    <section class="settings-section">
      <header>
        <h3>媒体库</h3>
        <span>{{ libraries.length }} 个库</span>
      </header>
      <div class="form-grid library-form">
        <UiFormField label="媒体库名称"><UiInput v-model="newLibrary.name" placeholder="电影" /></UiFormField>
        <UiFormField label="目录路径"><UiInput v-model="newLibrary.path" placeholder="/volume1/media/movies" /></UiFormField>
        <UiFormField label="类型"><UiSelect v-model="newLibrary.type" :options="libraryTypeOptions" /></UiFormField>
        <UiFormField label="元数据语言"><UiInput v-model="newLibrary.metadataLanguage" /></UiFormField>
        <UiFormField label="字幕语言"><UiInput v-model="newLibrary.subtitleLanguage" /></UiFormField>
        <UiCheckbox v-model="newLibrary.autoSubtitles" label="自动关联字幕" />
        <UiCheckbox v-model="newLibrary.allowAdultContent" label="允许成人内容元数据" />
        <UiButton :icon-left="Plus" @click="emit('create-library')">创建</UiButton>
      </div>
      <div class="settings-list">
        <article v-for="libraryItem in libraries" :key="libraryItem.id">
          <FolderOpen :size="17" />
          <div>
            <strong>{{ libraryItem.name }}</strong>
            <span>{{ typeLabel(libraryItem.type) }} · {{ libraryItem.paths.join('，') }}</span>
          </div>
          <small>{{ libraryItem.count }} 个文件</small>
          <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click.stop="emit('delete-library', libraryItem)">删除</UiButton>
        </article>
        <UiEmptyState v-if="!libraries.length" :icon="FolderOpen" title="还没有媒体库" compact />
      </div>
    </section>

    <section class="settings-section">
      <header>
        <h3>电视直播源</h3>
        <span>{{ liveChannels.length }} 个频道</span>
      </header>
      <div class="form-grid">
        <UiFormField label="名称"><UiInput v-model="newLiveSource.name" placeholder="家庭直播" /></UiFormField>
        <UiFormField label="M3U 地址或路径"><UiInput v-model="newLiveSource.url" placeholder="/volume1/media/live.m3u 或 https://..." /></UiFormField>
        <UiFormField label="User-Agent"><UiInput v-model="newLiveSource.userAgent" placeholder="需要鉴权时填写" /></UiFormField>
        <UiFormField label="并发限制"><UiInput v-model.number="newLiveSource.streamLimit" type="number" /></UiFormField>
        <UiButton :icon-left="Plus" @click="emit('create-live-source')">添加</UiButton>
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
        <UiEmptyState v-if="!liveSources.length" :icon="Radio" title="还没有直播源" compact />
      </div>
    </section>

    <section class="settings-section">
      <header>
        <h3>节目指南</h3>
        <span>{{ livePrograms.length }} 个节目</span>
      </header>
      <div class="form-grid">
        <UiFormField label="名称"><UiInput v-model="newGuideSource.name" placeholder="XMLTV" /></UiFormField>
        <UiFormField label="XMLTV 地址或路径"><UiInput v-model="newGuideSource.url" placeholder="/volume1/media/guide.xml 或 https://..." /></UiFormField>
        <UiFormField label="User-Agent"><UiInput v-model="newGuideSource.userAgent" placeholder="可选" /></UiFormField>
        <UiButton :icon-left="Plus" @click="emit('create-guide-source')">导入</UiButton>
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
        <UiEmptyState v-if="!guideSources.length" :icon="ListVideo" title="还没有节目指南源" compact />
      </div>
    </section>

    <section class="settings-section dvr-settings-card">
      <header>
        <h3>数字录像机</h3>
        <span>{{ recordingTimers.length }} 个预约</span>
      </header>
      <div class="dvr-settings-grid">
        <UiFormField class="dvr-field wide" label="录制目录"><UiInput v-model="dvrSettings.recordingPath" /></UiFormField>
        <UiFormField class="dvr-field" label="电影目录"><UiInput v-model="dvrSettings.movieRecordingPath" placeholder="可选" /></UiFormField>
        <UiFormField class="dvr-field" label="剧集目录"><UiInput v-model="dvrSettings.seriesRecordingPath" placeholder="可选" /></UiFormField>
        <UiFormField class="dvr-field compact" label="提前秒数"><UiInput v-model.number="dvrSettings.prePaddingSeconds" type="number" /></UiFormField>
        <UiFormField class="dvr-field compact" label="延后秒数"><UiInput v-model.number="dvrSettings.postPaddingSeconds" type="number" /></UiFormField>
        <UiFormField class="dvr-field compact" label="最大并发"><UiInput v-model.number="dvrSettings.maxConcurrentRecord" type="number" /></UiFormField>
        <UiFormField class="dvr-field wide" label="后处理命令"><UiInput v-model="dvrSettings.postProcessCommand" placeholder="/usr/local/bin/post_record.sh {path}" /></UiFormField>
      </div>
      <div class="dvr-actions">
        <UiSwitch v-model="dvrSettings.saveNfo" label="保存 NFO" />
        <UiSwitch v-model="dvrSettings.saveImages" label="保存图片" />
        <UiButton :icon-left="Settings" @click="emit('save-dvr-settings')">保存</UiButton>
      </div>
    </section>
  </section>
</template>
