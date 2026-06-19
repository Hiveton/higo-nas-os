<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import {
  Archive,
  CheckCircle2,
  Download,
  FileDown,
  FolderArchive,
  Gauge,
  Link,
  Magnet,
  Pause,
  Play,
  Rss,
  ShieldCheck,
  SlidersHorizontal,
  Trash2,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { DownloadTask, SpeedProfile } from '../../api/types';
import { useI18n } from 'vue-i18n';
import { UiBadge, UiButton, UiEmptyState, UiFormField, UiInput, UiModal } from '../ui';

const { t } = useI18n();

type SourceType = 'BT' | 'HTTP' | '磁力' | '订阅';
type SpeedMode = '智能限速' | '夜间全速' | '家庭优先';

const sourceOptions: Array<{ type: SourceType; icon: typeof Download }> = [
  { type: 'BT', icon: FileDown },
  { type: 'HTTP', icon: Link },
  { type: '磁力', icon: Magnet },
  { type: '订阅', icon: Rss },
];

const categories = ['全部', '影视', '音乐', '软件', '文档', '订阅'];
const speedProfiles: Record<SpeedMode, { down: string; up: string; note: string }> = {
  智能限速: { down: '18 MB/s', up: '2 MB/s', note: '客厅投屏时自动让路' },
  夜间全速: { down: '不限速', up: '8 MB/s', note: '00:00-07:00 开启满速' },
  家庭优先: { down: '6 MB/s', up: '1 MB/s', note: '视频会议和游戏优先' },
};

const tasks = ref<DownloadTask[]>([]);
const backendProfiles = ref<SpeedProfile[]>([]);

const selectedSource = ref<SourceType>('HTTP');
const selectedCategory = ref('全部');
const selectedTaskId = ref<string | number>(0);
const speedMode = ref<SpeedMode>('智能限速');
const newTaskLink = ref('');
const actionLog = ref<string[]>([]);
const loading = ref(false);
const busyAction = ref('');
const deleteConfirmTask = ref<DownloadTask | null>(null);
let refreshTimer: number | undefined;

const visibleTasks = computed(() =>
  selectedCategory.value === '全部' ? tasks.value : tasks.value.filter((task) => task.category === selectedCategory.value),
);

const selectedTask = computed(() => tasks.value.find((task) => task.id === selectedTaskId.value) ?? tasks.value[0] ?? null);
const activeProfile = computed(() => {
  const backend = backendProfiles.value.find((profile) => profile.name === speedMode.value);
  if (!backend) return speedProfiles[speedMode.value];
  return {
    down: backend.down ?? backend.downloadLimit ?? speedProfiles[speedMode.value].down,
    up: backend.up ?? backend.uploadLimit ?? speedProfiles[speedMode.value].up,
    note: backend.note ?? speedProfiles[speedMode.value].note,
  };
});
const completedCount = computed(() => tasks.value.filter((task) => task.status === '已完成').length);

const statusTone: Record<string, 'neutral' | 'primary' | 'warning' | 'success' | 'danger'> = {
  排队中: 'neutral',
  下载中: 'primary',
  暂停: 'warning',
  已完成: 'success',
  失败: 'danger',
};

async function loadDownloadState(silent = false) {
  if (!silent) loading.value = true;
  try {
    const [nextTasks, profiles] = await Promise.all([
      apiClient.downloads.getTasks(),
      apiClient.downloads.getSpeedProfiles(),
    ]);
    tasks.value = nextTasks;
    backendProfiles.value = profiles;
    if (!nextTasks.some((task) => task.id === selectedTaskId.value)) {
      selectedTaskId.value = nextTasks[0]?.id ?? 0;
    }
    const active = profiles.find((profile) => profile.active)?.name as SpeedMode | undefined;
    if (active && active in speedProfiles) speedMode.value = active;
    if (!silent) actionLog.value.unshift('下载队列已同步。');
  } catch (error) {
    if (!silent) actionLog.value.unshift(`下载服务不可用：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    if (!silent) loading.value = false;
  }
}

async function addDownloadTask() {
  const nextCategory = selectedSource.value === '订阅' ? '订阅' : selectedCategory.value === '全部' ? '' : selectedCategory.value;
  busyAction.value = 'create';
  try {
    const task = await apiClient.downloads.createTask({
      link: newTaskLink.value,
      source: selectedSource.value,
      category: nextCategory,
    });
    tasks.value = [task, ...tasks.value.filter((item) => item.id !== task.id)];
    selectedTaskId.value = task.id;
    selectedCategory.value = task.category || nextCategory || '全部';
    actionLog.value.unshift(`已添加 ${selectedSource.value} 任务：${task.name}`);
    void loadDownloadState(true);
  } catch (error) {
    actionLog.value.unshift(`添加任务失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
}

async function toggleTask(task: DownloadTask) {
  busyAction.value = `toggle-${task.id}`;
  try {
    const nextTask = task.status === '暂停'
      ? await apiClient.downloads.resumeTask(task.id)
      : await apiClient.downloads.pauseTask(task.id);
    tasks.value = tasks.value.map((item) => (item.id === task.id ? nextTask : item));
    selectedTaskId.value = task.id;
    actionLog.value.unshift(`${task.status === '暂停' ? '恢复' : '暂停'}任务：${task.name}`);
    void loadDownloadState(true);
  } catch (error) {
    actionLog.value.unshift(`切换任务失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
}

async function switchSpeedMode(mode: SpeedMode) {
  speedMode.value = mode;
  try {
    await apiClient.downloads.updateSpeedProfile({ name: mode });
    tasks.value = tasks.value.map((task) => (task.status === '下载中' ? { ...task, speed: activeProfile.value.down } : task));
    actionLog.value.unshift(`限速模式切换为 ${mode}：${activeProfile.value.note}`);
  } catch (error) {
    actionLog.value.unshift(`限速同步失败：${error instanceof Error ? error.message : 'unknown error'}`);
  }
}

async function archiveCompleted() {
  const task = selectedTask.value;
  if (!task) return;
  busyAction.value = `archive-${task.id}`;
  try {
    const result = await apiClient.downloads.archiveTask(task.id);
    tasks.value = tasks.value.map((item) =>
      item.id === task.id
        ? { ...item, progress: 100, status: '已完成', speed: '0 KB/s', archived: true, handling: `已归档到文件管家 /${item.category}` }
        : item,
    );
    actionLog.value.unshift(result.message ?? `文件管家已自动归档：${task.name}`);
  } catch (error) {
    actionLog.value.unshift(`归档失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
}

function requestDeleteTask() {
  if (!selectedTask.value) return;
  deleteConfirmTask.value = selectedTask.value;
}

async function deleteSelectedTask(deleteFile: boolean) {
  const task = deleteConfirmTask.value;
  if (!task) return;
  busyAction.value = `delete-${task.id}`;
  try {
    const result = await apiClient.downloads.deleteTask(task.id, deleteFile);
    tasks.value = tasks.value.filter((item) => item.id !== task.id);
    actionLog.value.unshift(result.message ?? `已删除任务：${task.name}`);
    deleteConfirmTask.value = null;
    selectedTaskId.value = visibleTasks.value[0]?.id ?? tasks.value[0]?.id ?? 0;
    void loadDownloadState(true);
  } catch (error) {
    actionLog.value.unshift(`删除失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
}

onMounted(() => {
  void loadDownloadState();
  refreshTimer = window.setInterval(() => {
    if (!busyAction.value) void loadDownloadState(true);
  }, 2500);
});

onBeforeUnmount(() => {
  if (refreshTimer) window.clearInterval(refreshTimer);
});
</script>

<template>
  <div class="download-center">
    <aside class="download-center__control" aria-label="下载任务创建和限速">
      <section class="download-center__card">
        <header>
          <h3><Download :size="15" /> 新建下载</h3>
          <span>{{ loading ? '同步中' : selectedSource }}</span>
        </header>
        <div class="download-center__sources">
          <button
            v-for="source in sourceOptions"
            :key="source.type"
            class="download-center__source"
            :class="{ 'download-center__source--active': selectedSource === source.type }"
            type="button"
            @click="selectedSource = source.type"
          >
            <component :is="source.icon" :size="15" />
            {{ source.type }}
          </button>
        </div>
        <UiFormField label="链接 / 订阅地址">
          <UiInput v-model="newTaskLink" placeholder="粘贴 HTTP、BT、磁力或订阅链接" />
        </UiFormField>
        <UiButton
          block
          :icon-left="FileDown"
          :loading="busyAction === 'create'"
          :disabled="!newTaskLink.trim()"
          @click="addDownloadTask"
        >
          {{ busyAction === 'create' ? '添加中' : '添加到队列' }}
        </UiButton>
      </section>

      <section class="download-center__card">
        <header>
          <h3><Gauge :size="15" /> 限速</h3>
          <span>{{ activeProfile.down }}</span>
        </header>
        <div class="download-center__speed">
          <button
            v-for="(_, mode) in speedProfiles"
            :key="mode"
            :class="{ 'download-center__speed-button--active': speedMode === mode }"
            type="button"
            @click="switchSpeedMode(mode)"
          >
            {{ mode }}
          </button>
        </div>
        <dl class="download-center__limits">
          <div>
            <dt>下载</dt>
            <dd>{{ activeProfile.down }}</dd>
          </div>
          <div>
            <dt>上传</dt>
            <dd>{{ activeProfile.up }}</dd>
          </div>
        </dl>
        <p>{{ activeProfile.note }}</p>
      </section>
    </aside>

    <main class="download-center__main">
      <nav class="download-center__categories" aria-label="下载分类">
        <button
          v-for="category in categories"
          :key="category"
          :class="{ 'download-center__category--active': selectedCategory === category }"
          type="button"
          @click="selectedCategory = category"
        >
          {{ category }}
        </button>
      </nav>

      <section class="download-center__queue" aria-label="下载队列">
        <button
          v-for="task in visibleTasks"
          :key="task.id"
          class="download-center__task"
          :class="{ 'download-center__task--active': selectedTaskId === task.id }"
          type="button"
          @click="selectedTaskId = task.id"
        >
          <div class="download-center__task-head">
            <div>
              <strong>{{ task.name }}</strong>
              <span>{{ task.source }} · {{ task.category }} · {{ task.size }}</span>
            </div>
            <UiBadge :tone="statusTone[task.status] ?? 'neutral'" size="sm">{{ task.status }}</UiBadge>
          </div>
          <div class="download-center__progress">
            <div :style="{ width: `${task.progress}%` }" />
          </div>
          <div class="download-center__task-foot">
            <span>{{ task.progress }}% · {{ task.speed }}</span>
            <span>{{ task.archived ? '已归档' : task.handling }}</span>
          </div>
        </button>
        <UiEmptyState
          v-if="visibleTasks.length === 0"
          :icon="Download"
          :title="tasks.length === 0 ? t('windows.download.emptyQueue') : t('windows.download.emptyCategory')"
          :description="tasks.length === 0 ? t('windows.download.emptyQueueHint') : t('windows.download.emptyCategoryHint')"
        />
      </section>
    </main>

    <aside class="download-center__detail" aria-label="任务详情和完成后处理">
      <template v-if="selectedTask">
        <header>
          <div>
            <p>下载队列 · {{ completedCount }} 个已完成</p>
            <h3>{{ selectedTask.name }}</h3>
          </div>
          <span>{{ selectedTask.source }}</span>
        </header>

        <section class="download-center__selected">
          <div class="download-center__selected-icon">
            <Archive v-if="selectedTask.archived" :size="22" />
            <Download v-else :size="22" />
          </div>
          <div>
            <strong>{{ selectedTask.status }}</strong>
            <p>{{ selectedTask.handling }}</p>
            <p v-if="selectedTask.filePath">{{ selectedTask.filePath }}</p>
            <p v-if="selectedTask.error">{{ selectedTask.error }}</p>
          </div>
        </section>

        <div class="download-center__actions" aria-label="下载任务操作">
          <UiButton
            variant="soft"
            size="sm"
            :icon-left="selectedTask.status === '暂停' ? Play : Pause"
            :disabled="selectedTask.status === '已完成'"
            :loading="busyAction === `toggle-${selectedTask.id}`"
            @click="toggleTask(selectedTask)"
          >
            {{ busyAction === `toggle-${selectedTask.id}` ? '处理中' : selectedTask.status === '暂停' ? '恢复任务' : '暂停任务' }}
          </UiButton>
          <UiButton
            variant="soft"
            size="sm"
            :icon-left="FolderArchive"
            :loading="busyAction === `archive-${selectedTask.id}`"
            @click="archiveCompleted"
          >
            {{ busyAction === `archive-${selectedTask.id}` ? '归档中' : '完成并归档' }}
          </UiButton>
          <UiButton
            variant="soft"
            tone="danger"
            size="sm"
            :icon-left="Trash2"
            :loading="busyAction === `delete-${selectedTask.id}`"
            @click="requestDeleteTask"
          >
            {{ busyAction === `delete-${selectedTask.id}` ? '删除中' : '删除任务' }}
          </UiButton>
        </div>

        <section class="download-center__automation">
          <h3><SlidersHorizontal :size="15" /> 完成后处理</h3>
          <div>
            <CheckCircle2 :size="14" />
            <span>{{ selectedTask.archiveRule?.targetPath ? `归档目录：${selectedTask.archiveRule.targetPath}` : '归档目录按分类生成' }}</span>
          </div>
          <div>
            <ShieldCheck :size="14" />
            <span>{{ selectedTask.archiveRule?.verifyChecksum ? '可校验文件完整性' : '保留下载文件名' }}</span>
          </div>
          <div>
            <FolderArchive :size="14" />
            <span>{{ selectedTask.archived ? '已归档' : '完成后可手动归档' }}</span>
          </div>
        </section>
      </template>

      <UiEmptyState
        v-else
        :icon="Download"
        compact
        :title="t('windows.download.emptyDetail')"
        :description="t('windows.download.emptyDetailHint')"
      />

      <section class="download-center__log" aria-label="操作记录">
        <h3>任务日志</h3>
        <ul>
          <li v-for="entry in actionLog" :key="entry">{{ entry }}</li>
          <li v-if="actionLog.length === 0">暂无任务日志。</li>
        </ul>
      </section>
    </aside>

    <UiModal
      :open="!!deleteConfirmTask"
      :title="t('windows.download.deleteTitle')"
      size="sm"
      :close-on-backdrop="busyAction !== `delete-${deleteConfirmTask?.id}`"
      :close-on-esc="busyAction !== `delete-${deleteConfirmTask?.id}`"
      @close="deleteConfirmTask = null"
    >
      <template v-if="deleteConfirmTask">
        <p class="download-center__modal-name">{{ deleteConfirmTask.name }}</p>
        <p class="download-center__modal-copy">
          {{ t('windows.download.deleteHint') }}
        </p>
        <p v-if="deleteConfirmTask.filePath" class="download-center__modal-path">{{ deleteConfirmTask.filePath }}</p>
      </template>
      <template #footer>
        <UiButton
          variant="ghost"
          tone="neutral"
          :disabled="busyAction === `delete-${deleteConfirmTask?.id}`"
          @click="deleteSelectedTask(false)"
        >
          {{ t('windows.download.deleteTaskOnly') }}
        </UiButton>
        <UiButton
          tone="danger"
          :loading="busyAction === `delete-${deleteConfirmTask?.id}`"
          @click="deleteSelectedTask(true)"
        >
          {{ t('windows.download.deleteTaskAndFiles') }}
        </UiButton>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.download-center {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr) 230px;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.download-center__control,
.download-center__main,
.download-center__detail {
  min-width: 0;
  min-height: 0;
}

.download-center__control,
.download-center__detail {
  display: grid;
  align-content: start;
  gap: 10px;
  overflow: auto;
}

.download-center__card,
.download-center__main,
.download-center__detail,
.download-center__task,
.download-center__automation,
.download-center__log {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.download-center__card {
  display: grid;
  gap: 10px;
  padding: 12px;
}

.download-center header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}

.download-center h3,
.download-center__selected strong,
.download-center__task strong {
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.download-center__card h3,
.download-center__automation h3 {
  display: flex;
  align-items: center;
  gap: 6px;
}

.download-center header span,
.download-center__task span,
.download-center__task-foot,
.download-center__selected p,
.download-center__card p,
.download-center__limits dt,
.download-center__log li,
.download-center__detail header p {
  color: var(--text-muted);
  font-size: 11px;
}

.download-center__sources,
.download-center__speed,
.download-center__categories,
.download-center__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.download-center__source,
.download-center__speed button,
.download-center__categories button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.download-center__source--active,
.download-center__speed-button--active,
.download-center__category--active {
  color: var(--text-inverse);
  background: var(--accent);
  border-color: transparent;
}

.download-center__limits {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
}

.download-center__limits div {
  padding: 8px;
  background: rgba(var(--surface-rgb), 0.58);
  border-radius: var(--radius-sm);
}

.download-center__limits dd {
  margin: 3px 0 0;
  color: var(--text-strong);
  font-size: 13px;
  font-weight: 800;
}

.download-center__main {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  gap: 10px;
  padding: 12px;
  overflow: hidden;
}

.download-center__queue {
  display: grid;
  align-content: start;
  gap: 9px;
  min-height: 0;
  overflow: auto;
}

.download-center__task {
  display: grid;
  gap: 9px;
  width: 100%;
  min-height: 96px;
  padding: 10px;
  text-align: left;
}

.download-center__task--active {
  border-color: rgba(19, 136, 255, 0.28);
  background: rgba(var(--surface-rgb), 0.72);
}

.download-center__task-head,
.download-center__task-foot {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
}

.download-center__task strong,
.download-center__task span {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.download-center__task span {
  margin-top: 4px;
}

.download-center__progress {
  height: 9px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.18);
  border-radius: 999px;
}

.download-center__progress div {
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-cyan), var(--accent-green));
  border-radius: inherit;
}

.download-center__detail {
  padding: 12px;
}

.download-center__detail header h3,
.download-center__detail header p {
  margin: 0;
}

.download-center__detail header h3 {
  margin-top: 4px;
  font-size: 14px;
  line-height: 1.25;
}

.download-center__detail header > span {
  flex: 0 0 auto;
  padding: 5px 8px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.download-center__selected {
  display: flex;
  gap: 10px;
  padding: 11px;
  background: linear-gradient(135deg, rgba(var(--surface-rgb), 0.9), rgba(255, 246, 227, 0.72));
  border: 1px solid rgba(22, 199, 221, 0.22);
  border-radius: var(--radius-md);
}

.download-center__selected-icon {
  display: grid;
  width: 38px;
  height: 38px;
  flex: 0 0 38px;
  place-items: center;
  color: var(--accent);
  background: rgba(var(--surface-rgb), 0.72);
  border-radius: var(--radius-sm);
}

.download-center__selected p {
  margin: 5px 0 0;
  line-height: 1.35;
}

.download-center__automation,
.download-center__log {
  display: grid;
  gap: 8px;
  padding: 11px;
}

.download-center__automation div {
  display: flex;
  gap: 7px;
  color: var(--accent-green);
  font-size: 11px;
  line-height: 1.35;
}

.download-center__log ul {
  display: grid;
  gap: 7px;
  max-height: 120px;
  padding: 0;
  margin: 0;
  overflow: auto;
  list-style: none;
}

.download-center__modal-name {
  margin: 0 0 var(--space-1);
  color: var(--text-strong);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  line-height: var(--lh-snug);
}

.download-center__modal-copy,
.download-center__modal-path {
  margin: 0 0 var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-xs);
  line-height: var(--lh-normal);
}

.download-center__modal-path {
  padding: var(--space-2) var(--space-3);
  overflow-wrap: anywhere;
  background: rgba(148, 163, 184, 0.1);
  border: 1px solid rgba(148, 163, 184, 0.16);
  border-radius: var(--radius-sm);
}

@media (max-width: 860px) {
  .download-center {
    grid-template-columns: 190px minmax(0, 1fr);
    overflow: auto;
  }

  .download-center__detail {
    grid-column: 1 / -1;
    overflow: visible;
  }
}

@media (max-width: 620px) {
  .download-center {
    display: block;
    overflow: auto;
  }

  .download-center__control,
  .download-center__main,
  .download-center__detail {
    margin-bottom: 10px;
  }

  .download-center__task-head,
  .download-center__task-foot {
    display: grid;
  }
}
</style>
