<script setup lang="ts">
import { Database, Trash2 } from 'lucide-vue-next';
import type { DockerVolume } from '../../../api/types';
import { UiButton, UiEmptyState, UiInput } from '../../ui';

type VolumeForm = { name: string; driver: string };

defineProps<{
  volumes: DockerVolume[];
  volumeForm: VolumeForm;
  selectedVolumeName: string;
}>();

const emit = defineEmits<{
  (e: 'create'): void;
  (e: 'select', name: string): void;
  (e: 'remove', name: string): void;
}>();
</script>

<template>
  <section class="docker-panel docker-volumes">
    <header class="docker-panel__header docker-form-row">
      <UiInput v-model="volumeForm.name" placeholder="卷名称" />
      <UiInput v-model="volumeForm.driver" placeholder="驱动，例如 local" />
      <UiButton variant="solid" tone="primary" size="sm" :icon-left="Database" :disabled="!volumeForm.name.trim()" @click="emit('create')">创建卷</UiButton>
    </header>

    <div class="docker-card">
      <header><h3><Database :size="15" /> 存储卷</h3></header>
      <div class="docker-list docker-list--grid">
        <article
          v-for="volume in volumes"
          :key="volume.name"
          class="docker-list-item"
          :class="{ 'is-active': selectedVolumeName === volume.name }"
          @click="emit('select', volume.name)"
        >
          <div>
            <strong>{{ volume.name }}</strong>
            <p>{{ volume.driver }} · {{ volume.scope }}</p>
            <small>{{ volume.mountpoint || '无挂载点' }}</small>
          </div>
          <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click.stop="emit('remove', volume.name)">删除</UiButton>
        </article>
        <UiEmptyState v-if="volumes.length === 0" compact :icon="Database" title="暂无存储卷" />
      </div>
    </div>
  </section>
</template>
