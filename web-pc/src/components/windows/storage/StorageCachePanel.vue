<script setup lang="ts">
import { computed } from 'vue';
import { Zap } from 'lucide-vue-next';
import type { Disk } from '../../../api/types';
import { UiButton, UiFormField, UiInput, UiSelect, UiSwitch } from '../../ui';

const props = defineProps<{
  physicalDisks: Disk[];
  selectedDiskSlot: string;
  cacheSettings: { standbyMinutes: number; ssdCache: boolean; cacheMode: string };
  hasSelectedDisk: boolean;
  busyAction: string;
  diskLabel: (disk: Disk) => string;
}>();

const emit = defineEmits<{
  (e: 'update:selectedDiskSlot', slot: string): void;
  (e: 'save'): void;
  (e: 'remove'): void;
}>();

const diskOptions = computed(() => props.physicalDisks.map((disk) => ({
  label: `${disk.slot} · ${props.diskLabel(disk)} · ${disk.size}`,
  value: disk.slot,
})));

const cacheModeOptions = [
  { label: '只读缓存', value: 'read' },
  { label: '读写缓存', value: 'read-write' },
];
</script>

<template>
  <div class="storage-monitor__cache-view">
    <section class="storage-monitor__cache-card">
      <header>
        <Zap :size="18" />
        <div>
          <strong>SSD 缓存加速</strong>
          <span>为指定硬盘启用只读或读写缓存，并设置休眠策略。</span>
        </div>
      </header>
      <div class="storage-monitor__cache-grid">
        <UiFormField label="选择硬盘">
          <UiSelect
            :model-value="selectedDiskSlot"
            :options="diskOptions"
            @update:model-value="emit('update:selectedDiskSlot', String($event))"
          />
        </UiFormField>
        <UiFormField label="休眠分钟">
          <UiInput v-model.number="cacheSettings.standbyMinutes" type="number" />
        </UiFormField>
        <UiFormField label="SSD 缓存">
          <UiSwitch v-model="cacheSettings.ssdCache" label="启用 SSD 缓存" />
        </UiFormField>
        <UiFormField label="缓存模式">
          <UiSelect v-model="cacheSettings.cacheMode" :options="cacheModeOptions" />
        </UiFormField>
      </div>
      <footer>
        <UiButton
          tone="primary"
          :loading="busyAction === 'cache'"
          :disabled="!hasSelectedDisk"
          @click="emit('save')"
        >
          保存缓存设置
        </UiButton>
        <UiButton
          variant="outline"
          tone="danger"
          :loading="busyAction === 'remove-disk'"
          :disabled="!hasSelectedDisk"
          @click="emit('remove')"
        >
          移除托管挂载
        </UiButton>
      </footer>
    </section>
  </div>
</template>
