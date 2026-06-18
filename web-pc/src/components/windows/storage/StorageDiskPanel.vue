<script setup lang="ts">
import { Copy, HardDrive } from 'lucide-vue-next';
import type { Disk } from '../../../api/types';
import { UiButton, UiEmptyState, UiFormField, UiIconButton, UiInput } from '../../ui';

type DiskFilter = 'all' | 'internal' | 'external';

defineProps<{
  visibleDiskInfo: Disk[];
  selectedDiskSlot: string;
  diskFilter: DiskFilter;
  diskCounts: { all: number; internal: number; external: number };
  managedMounts: Disk[];
  quickMount: { name: string; mountPath: string; role: string };
  busyAction: string;
  diskKind: (disk: Disk) => string;
  diskLabel: (disk: Disk) => string;
  diskProtocol: (disk: Disk) => string;
  diskHealthText: (disk: Disk) => string;
  diskSpaceName: (disk: Disk) => string;
  partitionSizeText: (partition: NonNullable<Disk['partitions']>[number]) => string;
}>();

const emit = defineEmits<{
  (e: 'update:diskFilter', value: DiskFilter): void;
  (e: 'select-disk', slot: string): void;
  (e: 'open-cache'): void;
  (e: 'copy-serial', serial?: string): void;
  (e: 'add-mount'): void;
}>();
</script>

<template>
  <aside class="storage-monitor__side">
    <section class="storage-monitor__disk-info-panel">
      <header class="storage-monitor__disk-info-head">
        <nav>
          <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'all' }" @click="emit('update:diskFilter', 'all')">
            全部 {{ diskCounts.all }}
          </button>
          <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'internal' }" @click="emit('update:diskFilter', 'internal')">
            内置 {{ diskCounts.internal }}
          </button>
          <button type="button" :class="{ 'storage-monitor__disk-filter--active': diskFilter === 'external' }" @click="emit('update:diskFilter', 'external')">
            外接 {{ diskCounts.external }}
          </button>
        </nav>
        <UiButton variant="soft" tone="neutral" size="sm" :icon-left="HardDrive" @click="emit('open-cache')">
          硬盘休眠
        </UiButton>
      </header>

      <div class="storage-monitor__disk-cards">
        <article
          v-for="(disk, index) in visibleDiskInfo"
          :key="disk.slot"
          class="storage-monitor__disk-card"
          :class="{ 'storage-monitor__disk-card--active': selectedDiskSlot === disk.slot }"
          @click="emit('select-disk', disk.slot)"
        >
          <span class="storage-monitor__disk-index">{{ index + 1 }}</span>
          <div class="storage-monitor__disk-device">
            <HardDrive :size="28" />
            <i v-if="disk.systemDisk">OS</i>
          </div>
          <div class="storage-monitor__disk-detail">
            <header>
              <strong>{{ disk.size }}</strong>
              <b v-if="disk.systemDisk">系统安装</b>
              <b v-else-if="diskSpaceName(disk)">{{ diskSpaceName(disk) }}</b>
            </header>
            <dl>
              <div><dt>硬盘类型</dt><dd>{{ diskKind(disk) }}</dd></div>
              <div><dt>型号</dt><dd>{{ diskLabel(disk) }}</dd></div>
              <div><dt>健康状态</dt><dd>{{ diskHealthText(disk) }}</dd></div>
              <div><dt>接口协议</dt><dd>{{ diskProtocol(disk) }}</dd></div>
              <div class="storage-monitor__serial"><dt>序列号</dt><dd>{{ disk.serial || '-' }}<UiIconButton v-if="disk.serial" :icon="Copy" label="复制序列号" size="sm" @click.stop="emit('copy-serial', disk.serial)" /></dd></div>
            </dl>
            <div v-if="disk.partitions?.length" class="storage-monitor__partitions">
              <section v-for="partition in disk.partitions" :key="partition.path || partition.name">
                <b>分区：{{ partition.name }}</b>
                <span>{{ partitionSizeText(partition) }}</span>
                <span>{{ partition.fileSystem || '文件系统未知' }}</span>
                <span v-if="partition.system">系统安装</span>
              </section>
            </div>
          </div>
        </article>
        <UiEmptyState
          v-if="!visibleDiskInfo.length"
          class="storage-monitor__disk-empty"
          :icon="HardDrive"
          title="暂无硬盘"
          description="刷新后仍为空时，请检查系统是否能读取块设备信息。"
          compact
        />
      </div>
    </section>
    <section>
      <h3>添加托管挂载</h3>
      <UiFormField label="名称">
        <UiInput v-model="quickMount.name" />
      </UiFormField>
      <UiFormField label="路径">
        <UiInput v-model="quickMount.mountPath" />
      </UiFormField>
      <UiButton tone="primary" :loading="busyAction === 'add-disk'" @click="emit('add-mount')">添加挂载</UiButton>
      <div v-if="managedMounts.length" class="storage-monitor__managed-list">
        <span v-for="disk in managedMounts" :key="disk.slot">
          {{ disk.model || disk.mountPath }} · {{ disk.size }}
        </span>
      </div>
    </section>
  </aside>
</template>
