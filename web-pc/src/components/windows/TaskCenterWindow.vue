<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  Boxes,
  CheckCircle2,
  CircleSlash,
  Database,
  DownloadCloud,
  Film,
  HardDriveDownload,
  ListChecks,
  Loader,
  Music,
  Package,
  ShieldCheck,
  XCircle,
} from 'lucide-vue-next';
import type { Component } from 'vue';
import { tasksStore } from '../../stores/tasks';
import type { Task, TaskStatus } from '../../api/types';
import {
  UiBadge,
  UiButton,
  UiDataTable,
  UiEmptyState,
  UiProgressBar,
  UiTabs,
  useConfirm,
  useToast,
  type Column,
  type TabItem,
  type UiTone,
} from '../ui';

const toast = useToast();
const confirm = useConfirm();

onMounted(() => tasksStore.start());
onBeforeUnmount(() => tasksStore.stop());

type FilterKey = 'all' | 'active' | 'succeeded' | 'failed' | 'canceled';
const filter = ref<FilterKey>('all');

const stats = tasksStore.stats;

const tabs = computed<TabItem[]>(() => [
  { key: 'all', label: '全部', badge: stats.value.total || undefined },
  { key: 'active', label: '进行中', badge: stats.value.running + stats.value.queued || undefined },
  { key: 'succeeded', label: '已完成', badge: stats.value.succeeded || undefined },
  { key: 'failed', label: '失败', badge: stats.value.failed || undefined },
  { key: 'canceled', label: '已取消', badge: stats.value.canceled || undefined },
]);

const rows = computed(() =>
  tasksStore.tasks.value.filter((t) => {
    switch (filter.value) {
      case 'active':
        return t.status === 'running' || t.status === 'queued';
      case 'succeeded':
        return t.status === 'succeeded';
      case 'failed':
        return t.status === 'failed';
      case 'canceled':
        return t.status === 'canceled';
      default:
        return true;
    }
  }),
);

const columns: Column<Task>[] = [
  { key: 'task', label: '任务', width: '34%' },
  { key: 'progress', label: '进度', width: '26%' },
  { key: 'status', label: '状态', width: '14%' },
  { key: 'updatedAt', label: '更新时间', width: '16%' },
  { key: 'actions', label: '操作', width: '10%', align: 'right' },
];

// kind -> friendly label + icon
const kindMeta: Record<string, { label: string; icon: Component }> = {
  'storage.scan': { label: '存储扫描', icon: HardDriveDownload },
  'video.transcode': { label: '视频转码', icon: Film },
  'media.job': { label: '媒体作业', icon: Music },
  'backups.run': { label: '备份运行', icon: ShieldCheck },
  'backups.verify': { label: '备份校验', icon: ShieldCheck },
  'docker.pull': { label: '镜像拉取', icon: Boxes },
  'downloads.fetch': { label: '下载任务', icon: DownloadCloud },
};

function kindLabel(kind: string): string {
  return kindMeta[kind]?.label ?? kind;
}
function kindIcon(kind: string): Component {
  return kindMeta[kind]?.icon ?? Package;
}

const statusMeta: Record<TaskStatus, { label: string; tone: UiTone; icon: Component }> = {
  queued: { label: '排队中', tone: 'neutral', icon: Loader },
  running: { label: '运行中', tone: 'primary', icon: Loader },
  succeeded: { label: '已完成', tone: 'success', icon: CheckCircle2 },
  failed: { label: '失败', tone: 'danger', icon: XCircle },
  canceled: { label: '已取消', tone: 'warning', icon: CircleSlash },
};

function progressTone(status: TaskStatus): UiTone {
  if (status === 'failed') return 'danger';
  if (status === 'canceled') return 'warning';
  if (status === 'succeeded') return 'success';
  return 'primary';
}

function formatTime(iso?: string): string {
  if (!iso) return '—';
  const d = new Date(iso);
  if (Number.isNaN(d.getTime())) return '—';
  return d.toLocaleTimeString('zh-CN', { hour: '2-digit', minute: '2-digit', second: '2-digit' });
}

function cancelable(task: Task): boolean {
  return task.status === 'queued' || task.status === 'running';
}

async function onCancel(task: Task) {
  if (task.status === 'running') {
    const ok = await confirm({
      title: '取消运行中的任务',
      message: `「${kindLabel(task.kind)}」正在运行，确定要中止吗？`,
      tone: 'danger',
      confirmLabel: '中止任务',
    });
    if (!ok) return;
  }
  try {
    await tasksStore.cancel(task.id);
    toast.success('已请求取消任务');
  } catch (error) {
    toast.error(`取消失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}
</script>

<template>
  <div class="task-center">
    <header class="task-center__head">
      <div class="task-center__title">
        <ListChecks :size="20" :stroke-width="2.1" />
        <div>
          <h2>任务中心</h2>
          <p>统一查看与管理所有后台任务 · {{ tasksStore.connected.value ? '实时已连接' : '轮询中' }}</p>
        </div>
      </div>
      <div class="task-center__summary">
        <UiBadge tone="primary" variant="soft" size="sm">运行 {{ stats.running }}</UiBadge>
        <UiBadge tone="neutral" variant="soft" size="sm">排队 {{ stats.queued }}</UiBadge>
        <UiBadge tone="success" variant="soft" size="sm">完成 {{ stats.succeeded }}</UiBadge>
        <UiBadge v-if="stats.failed" tone="danger" variant="soft" size="sm">失败 {{ stats.failed }}</UiBadge>
      </div>
    </header>

    <UiTabs v-model="filter" :tabs="tabs" variant="segmented" size="sm" overflow="menu" class="task-center__tabs" />

    <div class="task-center__body">
      <UiDataTable
        :columns="columns"
        :rows="rows"
        row-key="id"
        density="compact"
        :empty="{ title: '暂无任务', description: '触发下载、转码、备份等操作后会显示在这里。' }"
      >
        <template #cell-task="{ row }">
          <div class="task-center__name">
            <component :is="kindIcon(row.kind)" :size="16" :stroke-width="2" />
            <div class="task-center__name-text">
              <strong>{{ kindLabel(row.kind) }}</strong>
              <span v-if="row.message">{{ row.message }}</span>
            </div>
          </div>
        </template>

        <template #cell-progress="{ row }">
          <UiProgressBar
            :value="row.progress"
            :tone="progressTone(row.status)"
            :indeterminate="row.status === 'running' && row.progress === 0"
            size="sm"
            show-value
          />
        </template>

        <template #cell-status="{ row }">
          <UiBadge :tone="statusMeta[row.status].tone" variant="soft" size="sm">
            {{ statusMeta[row.status].label }}
          </UiBadge>
        </template>

        <template #cell-updatedAt="{ row }">
          <span class="task-center__time">{{ formatTime(row.updatedAt) }}</span>
        </template>

        <template #cell-actions="{ row }">
          <UiButton
            v-if="cancelable(row)"
            variant="ghost"
            tone="danger"
            size="sm"
            @click="onCancel(row)"
          >
            取消
          </UiButton>
          <span v-else class="task-center__time">—</span>
        </template>

        <template #empty>
          <UiEmptyState :icon="ListChecks" title="暂无任务" description="触发下载、转码、备份等操作后会显示在这里。" compact />
        </template>
      </UiDataTable>
    </div>
  </div>
</template>

<style scoped>
.task-center {
  display: flex;
  flex-direction: column;
  height: 100%;
  min-height: 0;
  gap: var(--space-3);
}

.task-center__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-3);
  flex-wrap: wrap;
}

.task-center__title {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--accent);
}
.task-center__title h2 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-lg);
  font-weight: var(--fw-semibold);
}
.task-center__title p {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.task-center__summary {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
}

.task-center__tabs {
  flex-shrink: 0;
}

.task-center__body {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}

.task-center__name {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--accent);
}
.task-center__name-text {
  display: flex;
  flex-direction: column;
  min-width: 0;
}
.task-center__name-text strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
}
.task-center__name-text span {
  overflow: hidden;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.task-center__time {
  color: var(--text-soft);
  font-size: var(--fs-xs);
}
</style>
