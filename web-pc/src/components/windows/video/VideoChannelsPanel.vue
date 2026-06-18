<script setup lang="ts">
import { Tv } from 'lucide-vue-next';
import { UiEmptyState } from '../../ui';
import type { LiveChannel } from '../../../api/types';

defineProps<{
  liveChannels: LiveChannel[];
}>();

const emit = defineEmits<{
  (e: 'play', channel: LiveChannel): void;
}>();
</script>

<template>
  <div class="channel-grid">
    <article v-for="channel in liveChannels" :key="channel.id" class="channel-card" @click="emit('play', channel)">
      <img v-if="channel.logo" :src="channel.logo" :alt="channel.name" />
      <Tv v-else :size="22" />
      <strong>{{ channel.name }}</strong>
      <span>{{ channel.group || '直播频道' }}</span>
    </article>
    <UiEmptyState v-if="!liveChannels.length" :icon="Tv" title="还没有直播频道" description="添加 M3U 源后会显示频道列表。" />
  </div>
</template>
