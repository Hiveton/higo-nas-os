<script setup lang="ts">
import { ListVideo } from 'lucide-vue-next';
import { UiEmptyState } from '../../ui';
import type { VideoTask } from '../../../api/types';

defineProps<{
  tasks: VideoTask[];
  taskLabel: (type: string) => string;
  statusLabel: (status: string) => string;
}>();
</script>

<template>
  <section class="tasks-view">
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
    <UiEmptyState v-if="!tasks.length" :icon="ListVideo" title="暂无任务" compact />
  </section>
</template>
