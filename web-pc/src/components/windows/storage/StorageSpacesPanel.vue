<script setup lang="ts">
import { Database, HardDrive, Star } from 'lucide-vue-next';
import type { Disk, StorageSpace } from '../../../api/types';
import { UiBadge, UiButton, UiEmptyState, UiProgressBar } from '../../ui';
import type { UiTone } from '../../ui';

function healthTone(health?: string): UiTone {
  const text = String(health ?? '');
  if (text.includes('正常') || text.includes('健康') || text.includes('良好')) return 'success';
  if (text.includes('警告') || text.includes('降级')) return 'warning';
  if (text.includes('错误') || text.includes('故障') || text.includes('离线')) return 'danger';
  return 'success';
}

defineProps<{
  spaces: StorageSpace[];
  loading: boolean;
  selectedSpaceId: string;
  defaultSpaceId?: string;
  poolName?: (poolId?: string) => string;
  spaceDisks: (space: StorageSpace) => Disk[];
  diskKind: (disk: Disk) => string;
  diskProtocol: (disk: Disk) => string;
  formatGB: (value: number) => string;
  parseCapacityGB: (value?: string) => number;
}>();

const emit = defineEmits<{ (e: 'select', id: string): void; (e: 'set-default', id: string): void }>();
</script>

<template>
  <div class="storage-monitor__spaces">
    <article
      v-for="space in spaces"
      :key="space.id"
      class="storage-monitor__space-card"
      :class="{ 'storage-monitor__space-card--active': selectedSpaceId === space.id }"
      @click="emit('select', space.id)"
    >
      <div class="storage-monitor__space-icon">
        <Database :size="26" />
      </div>
      <div class="storage-monitor__space-main">
        <header>
          <div>
            <strong>{{ space.name }}</strong>
            <span>{{ space.mode.toUpperCase() }} ｜ {{ space.fileSystem }}<template v-if="space.poolId"> ｜ 存储池：{{ poolName ? poolName(space.poolId) : space.poolId }}</template></span>
          </div>
          <div class="storage-monitor__space-tags">
            <UiBadge v-if="defaultSpaceId === space.id" tone="primary" variant="soft"><Star :size="11" /> 默认</UiBadge>
            <UiButton
              v-else
              variant="ghost"
              size="sm"
              :icon-left="Star"
              @click.stop="emit('set-default', space.id)"
            >设为默认</UiButton>
            <UiBadge :tone="healthTone(space.health)" variant="soft">{{ space.health }}</UiBadge>
          </div>
        </header>
        <UiProgressBar
          class="storage-monitor__space-capacity"
          :value="Math.min(100, Math.max(0, space.usedPercent || 0))"
          :tone="healthTone(space.health)"
          size="md"
        />
        <p>
          容量 {{ formatGB(parseCapacityGB(space.total) * (space.usedPercent || 0) / 100) }} /
          {{ space.total }}，
          剩余 {{ formatGB(parseCapacityGB(space.total) * (1 - (space.usedPercent || 0) / 100)) }}
        </p>
        <dl>
          <div><dt>硬盘类型</dt><dd>{{ spaceDisks(space)[0] ? diskKind(spaceDisks(space)[0]) : '--' }}</dd></div>
          <div><dt>接口协议</dt><dd>{{ spaceDisks(space)[0] ? diskProtocol(spaceDisks(space)[0]) : '--' }}</dd></div>
          <div><dt>使用硬盘</dt><dd>{{ space.diskSlots.length }}</dd></div>
        </dl>
        <div class="storage-monitor__space-disks">
          <span v-for="disk in spaceDisks(space)" :key="disk.slot">
            <HardDrive :size="14" />
            {{ disk.size }}
          </span>
        </div>
      </div>
    </article>

    <UiEmptyState
      v-if="!loading && spaces.length === 0"
      :icon="Database"
      title="暂无存储空间"
      description="点击“创建存储空间”按文件系统、硬盘、模式、用户授权完成创建。"
    />
  </div>
</template>
