<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
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

const tasks = ref<DownloadTask[]>([
  {
    id: 1,
    name: '纪录片合集 S02',
    source: 'BT',
    category: '影视',
    size: '86.4 GB',
    progress: 68,
    speed: '12.8 MB/s',
    status: '下载中',
    handling: '完成后刮削海报并归档到 /Media/TV',
    archived: false,
  },
  {
    id: 2,
    name: '家庭音乐精选 FLAC',
    source: 'HTTP',
    category: '音乐',
    size: '12.1 GB',
    progress: 100,
    speed: '0 KB/s',
    status: '已完成',
    handling: '等待导入音乐库',
    archived: false,
  },
  {
    id: 3,
    name: 'Ubuntu Server 镜像',
    source: '磁力',
    category: '软件',
    size: '5.9 GB',
    progress: 42,
    speed: '6.4 MB/s',
    status: '下载中',
    handling: '完成后校验 SHA256',
    archived: false,
  },
  {
    id: 4,
    name: '每周公开课订阅',
    source: '订阅',
    category: '订阅',
    size: '2.8 GB',
    progress: 0,
    speed: '等待 RSS',
    status: '暂停',
    handling: '新条目自动下载到 /Downloads/Courses',
    archived: false,
  },
]);
const backendProfiles = ref<SpeedProfile[]>([]);

const selectedSource = ref<SourceType>('磁力');
const selectedCategory = ref('全部');
const selectedTaskId = ref<string | number>(1);
const speedMode = ref<SpeedMode>('智能限速');
const newTaskLink = ref('magnet:?xt=urn:btih:higo-family-media');
const actionLog = ref<string[]>(['文件管家联动已启用：完成任务会按分类自动归档。']);
const loading = ref(false);
const busyAction = ref('');

const visibleTasks = computed(() =>
  selectedCategory.value === '全部' ? tasks.value : tasks.value.filter((task) => task.category === selectedCategory.value),
);

const selectedTask = computed(() => tasks.value.find((task) => task.id === selectedTaskId.value) ?? tasks.value[0]);
const activeProfile = computed(() => {
  const backend = backendProfiles.value.find((profile) => profile.name === speedMode.value);
  if (!backend) return speedProfiles[speedMode.value];
  return {
    down: backend.down ?? speedProfiles[speedMode.value].down,
    up: backend.up ?? speedProfiles[speedMode.value].up,
    note: backend.note ?? speedProfiles[speedMode.value].note,
  };
});
const completedCount = computed(() => tasks.value.filter((task) => task.status === '已完成').length);

const statusClass: Record<string, string> = {
  下载中: 'download-center__status--running',
  暂停: 'download-center__status--paused',
  已完成: 'download-center__status--done',
};

async function loadDownloadState() {
  loading.value = true;
  try {
    const [nextTasks, profiles] = await Promise.all([
      apiClient.downloads.getTasks(),
      apiClient.downloads.getSpeedProfiles(),
    ]);
    tasks.value = nextTasks;
    backendProfiles.value = profiles;
    selectedTaskId.value = nextTasks[0]?.id ?? selectedTaskId.value;
    actionLog.value.unshift('下载队列和限速策略已从后端同步。');
  } catch (error) {
    actionLog.value.unshift(`后端暂不可用，继续使用本地下载缓存：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    loading.value = false;
  }
}

async function addDownloadTask() {
  const nextCategory = selectedSource.value === '订阅' ? '订阅' : selectedCategory.value === '全部' ? '影视' : selectedCategory.value;
  busyAction.value = 'create';
  try {
    const task = await apiClient.downloads.createTask({
      link: newTaskLink.value,
      source: selectedSource.value,
      category: nextCategory,
    });
    tasks.value = [task, ...tasks.value];
    selectedTaskId.value = task.id;
    selectedCategory.value = nextCategory;
    actionLog.value.unshift(`已添加 ${selectedSource.value} 任务：${task.name}`);
  } catch (error) {
    actionLog.value.unshift(`添加任务失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
}

async function toggleTask(task: DownloadTask) {
  busyAction.value = `toggle-${task.id}`;
  try {
    const result = task.status === '暂停'
      ? await apiClient.downloads.resumeTask(task.id)
      : await apiClient.downloads.pauseTask(task.id);
    const nextStatus = task.status === '暂停' ? '下载中' : '暂停';
    tasks.value = tasks.value.map((item) =>
      item.id === task.id ? { ...item, status: nextStatus, speed: nextStatus === '下载中' ? activeProfile.value.down : '0 KB/s' } : item,
    );
    selectedTaskId.value = task.id;
    actionLog.value.unshift(result.message ?? `${nextStatus === '下载中' ? '恢复' : '暂停'}任务：${task.name}`);
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

async function cleanArchivedRecords() {
  const archived = tasks.value.filter((task) => task.archived);
  busyAction.value = 'clean';
  try {
    await Promise.all(archived.map((task) => apiClient.downloads.deleteTask(task.id)));
    tasks.value = tasks.value.filter((task) => !task.archived);
    actionLog.value.unshift(`已清理 ${archived.length} 条已归档记录，原文件保留在文件管家。`);
  } catch (error) {
    actionLog.value.unshift(`清理失败：${error instanceof Error ? error.message : 'unknown error'}`);
  } finally {
    busyAction.value = '';
  }
  selectedTaskId.value = visibleTasks.value[0]?.id ?? tasks.value[0]?.id ?? 0;
}

onMounted(loadDownloadState);
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
        <label class="download-center__input">
          <span>链接 / 订阅地址</span>
          <input v-model="newTaskLink" />
        </label>
        <button class="download-center__primary" type="button" :disabled="busyAction === 'create'" @click="addDownloadTask">
          <FileDown :size="14" />
          {{ busyAction === 'create' ? '添加中' : '添加到队列' }}
        </button>
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
            <small :class="['download-center__status', statusClass[task.status]]">{{ task.status }}</small>
          </div>
          <div class="download-center__progress">
            <div :style="{ width: `${task.progress}%` }" />
          </div>
          <div class="download-center__task-foot">
            <span>{{ task.progress }}% · {{ task.speed }}</span>
            <span>{{ task.archived ? '已归档' : task.handling }}</span>
          </div>
        </button>
        <div v-if="visibleTasks.length === 0" class="download-center__empty">
          <Download :size="20" />
          <strong>该分类暂无任务</strong>
          <span>切换分类或添加新的 BT、HTTP、磁力、订阅任务。</span>
        </div>
      </section>
    </main>

    <aside class="download-center__detail" aria-label="任务详情和完成后处理">
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
        </div>
      </section>

      <div class="download-center__actions" aria-label="下载任务操作">
        <button type="button" :disabled="selectedTask.status === '已完成' || busyAction === `toggle-${selectedTask.id}`" @click="toggleTask(selectedTask)">
          <Play v-if="selectedTask.status === '暂停'" :size="14" />
          <Pause v-else :size="14" />
          {{ busyAction === `toggle-${selectedTask.id}` ? '处理中' : selectedTask.status === '暂停' ? '恢复任务' : '暂停任务' }}
        </button>
        <button type="button" :disabled="busyAction === `archive-${selectedTask.id}`" @click="archiveCompleted">
          <FolderArchive :size="14" />
          {{ busyAction === `archive-${selectedTask.id}` ? '归档中' : '完成并归档' }}
        </button>
        <button type="button" :disabled="busyAction === 'clean'" @click="cleanArchivedRecords">
          <Trash2 :size="14" />
          {{ busyAction === 'clean' ? '清理中' : '清理记录' }}
        </button>
      </div>

      <section class="download-center__automation">
        <h3><SlidersHorizontal :size="15" /> 完成后处理</h3>
        <div>
          <CheckCircle2 :size="14" />
          <span>按分类移动到文件管家目录</span>
        </div>
        <div>
          <ShieldCheck :size="14" />
          <span>影视自动刮削，文档保留原始文件名</span>
        </div>
        <div>
          <FolderArchive :size="14" />
          <span>订阅任务写入 /Downloads/Subscriptions</span>
        </div>
      </section>

      <section class="download-center__log" aria-label="操作记录">
        <h3>任务日志</h3>
        <ul>
          <li v-for="entry in actionLog" :key="entry">{{ entry }}</li>
        </ul>
      </section>
    </aside>
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
  background: rgba(255, 255, 255, 0.5);
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
.download-center__categories button,
.download-center__actions button,
.download-center__primary {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.download-center__source--active,
.download-center__speed-button--active,
.download-center__category--active {
  color: #fff;
  background: var(--accent);
  border-color: transparent;
}

.download-center__input {
  display: grid;
  gap: 6px;
}

.download-center__input span {
  color: var(--text-muted);
  font-size: 11px;
  font-weight: 700;
}

.download-center__input input {
  width: 100%;
  min-width: 0;
  height: 34px;
  padding: 0 9px;
  color: var(--text);
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  outline: 0;
  font-size: 12px;
}

.download-center__primary {
  justify-content: center;
  color: #fff;
  background: var(--accent);
}

.download-center__primary:disabled,
.download-center__actions button:disabled {
  color: var(--text-soft);
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.12);
  border-color: rgba(148, 163, 184, 0.18);
}

.download-center__limits {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
  margin: 0;
}

.download-center__limits div {
  padding: 8px;
  background: rgba(255, 255, 255, 0.58);
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
  background: rgba(231, 247, 255, 0.72);
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

.download-center__status {
  flex: 0 0 auto;
  height: 22px;
  padding: 4px 8px;
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.download-center__status--running {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
}

.download-center__status--paused {
  color: #b36a00;
  background: rgba(245, 158, 11, 0.14);
}

.download-center__status--done {
  color: var(--accent-green);
  background: rgba(34, 181, 115, 0.13);
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

.download-center__empty {
  display: grid;
  min-height: 180px;
  place-items: center;
  align-content: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 11px;
}

.download-center__empty strong {
  color: var(--text-strong);
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
  background: linear-gradient(135deg, rgba(231, 247, 255, 0.9), rgba(255, 246, 227, 0.72));
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
  background: rgba(255, 255, 255, 0.72);
  border-radius: var(--radius-sm);
}

.download-center__selected p {
  margin: 5px 0 0;
  line-height: 1.35;
}

.download-center__actions button:disabled {
  cursor: default;
  opacity: 0.48;
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
