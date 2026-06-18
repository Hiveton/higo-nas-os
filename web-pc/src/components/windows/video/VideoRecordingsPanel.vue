<script setup lang="ts">
import { Radio } from 'lucide-vue-next';
import { UiButton, UiEmptyState } from '../../ui';
import type { DvrSettings, RecordingItem, RecordingTimer } from '../../../api/types';

defineProps<{
  recordingTimers: RecordingTimer[];
  recordings: RecordingItem[];
  dvrSettings: DvrSettings;
  liveTimeRange: (startAt: string, endAt: string) => string;
  statusLabel: (status: string) => string;
}>();

const emit = defineEmits<{
  (e: 'cancel', timer: RecordingTimer): void;
}>();
</script>

<template>
  <div class="guide-list">
    <article v-for="timer in recordingTimers" :key="timer.id" class="guide-row">
      <div>
        <strong>{{ timer.name }}</strong>
        <span>{{ timer.channelName }} · {{ liveTimeRange(timer.startAt, timer.endAt) }} · {{ statusLabel(timer.status) }}</span>
        <small>{{ timer.targetPath || dvrSettings.recordingPath }}</small>
      </div>
      <UiButton v-if="timer.status === 'scheduled'" variant="soft" tone="danger" size="sm" @click="emit('cancel', timer)">取消</UiButton>
    </article>
    <article v-for="recording in recordings" :key="recording.id" class="guide-row muted">
      <div>
        <strong>{{ recording.title }}</strong>
        <span>{{ statusLabel(recording.status) }}</span>
        <small>{{ recording.path || recording.message || '等待录制任务执行' }}</small>
      </div>
    </article>
    <UiEmptyState v-if="!recordingTimers.length && !recordings.length" :icon="Radio" title="暂无录制预约" compact />
  </div>
</template>
