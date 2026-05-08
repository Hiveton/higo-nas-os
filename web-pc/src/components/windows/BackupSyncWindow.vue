<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ArchiveRestore, CheckCircle2, CloudUpload, DatabaseBackup, Pause, Play, RefreshCw, ShieldCheck } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { BackupJob } from '../../api/types';

const fallbackJobs: BackupJob[] = [
  {
    id: 'family-photo',
    name: '家庭相册增量备份',
    source: '照片与视频',
    target: '异地备份卷',
    state: '同步中',
    schedule: '每 6 小时',
    progress: 72,
    speed: '18 MB/s',
    eta: '剩余 16 分钟',
    lastRun: '今天 08:30',
    nextRun: '今天 14:30',
    retention: '保留 180 天',
    policy: '去重 + 加密 + 远端校验',
    health: '正常',
    enabled: true,
  },
  {
    id: 'team-snapshot',
    name: '团队空间快照',
    source: '项目资料',
    target: '每日快照',
    state: '校验中',
    schedule: '每天 02:00',
    progress: 94,
    speed: '已校验 1.8 TB',
    eta: '等待归档索引',
    lastRun: '今天 02:00',
    nextRun: '明天 02:00',
    retention: '保留 365 天',
    policy: '只读快照 + 变更审计',
    health: '正常',
    enabled: true,
  },
];

const jobs = ref<BackupJob[]>(fallbackJobs);
const selectedJobId = ref(fallbackJobs[0].id);
const actionState = ref('备份同步正在连接后端任务队列。');

const selectedJob = computed(() => jobs.value.find((job) => job.id === selectedJobId.value) ?? jobs.value[0]);
const activeJobs = computed(() => jobs.value.filter((job) => job.state === '同步中' || job.state === '校验中').length);
const completedJobs = computed(() => jobs.value.filter((job) => job.progress >= 100 || job.state === '已完成').length);
const averageProgress = computed(() =>
  Math.round(jobs.value.reduce((sum, job) => sum + job.progress, 0) / Math.max(jobs.value.length, 1)),
);

async function loadBackupJobs() {
  try {
    const nextJobs = await apiClient.backup.getJobs();
    if (nextJobs.length) {
      jobs.value = nextJobs;
      selectedJobId.value = nextJobs.some((job) => job.id === selectedJobId.value) ? selectedJobId.value : nextJobs[0].id;
    }
    actionState.value = '备份任务已从后端同步，可执行运行、暂停、恢复和校验。';
  } catch (error) {
    actionState.value = `后端暂不可用，继续使用本地备份缓存：${errorMessage(error)}`;
  }
}

function selectJob(id: string) {
  selectedJobId.value = id;
  actionState.value = `正在查看 ${selectedJob.value.name}。`;
}

async function runBackupJob(id = selectedJob.value.id) {
  await mutateBackupJob(id, () => apiClient.backup.runJob(id), '备份任务已提交到后端队列。');
}

async function pauseBackupJob(id = selectedJob.value.id) {
  await mutateBackupJob(id, () => apiClient.backup.pauseJob(id), '备份任务已暂停，新数据会等待恢复后同步。');
}

async function resumeBackupJob(id = selectedJob.value.id) {
  await mutateBackupJob(id, () => apiClient.backup.resumeJob(id), '备份任务已恢复同步。');
}

async function verifyBackupJob(id = selectedJob.value.id) {
  await mutateBackupJob(id, () => apiClient.backup.verifyJob(id), '备份校验已启动，校验结果会写入审计。');
}

async function mutateBackupJob(id: string, request: () => Promise<BackupJob>, message: string) {
  try {
    const nextJob = await request();
    replaceJob(nextJob);
    actionState.value = `${nextJob.name}：${message}`;
  } catch (error) {
    actionState.value = `${jobs.value.find((job) => job.id === id)?.name ?? '备份任务'} 操作失败：${errorMessage(error)}`;
  }
}

function replaceJob(job: BackupJob) {
  jobs.value = jobs.value.map((item) => (item.id === job.id ? job : item));
  selectedJobId.value = job.id;
}

function errorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error);
}

onMounted(loadBackupJobs);
</script>

<template>
  <div class="backup-sync">
    <section class="backup-sync__summary" aria-label="备份同步摘要">
      <article>
        <CloudUpload :size="18" />
        <span>进行中</span>
        <strong>{{ activeJobs }} 个</strong>
      </article>
      <article>
        <ArchiveRestore :size="18" />
        <span>平均进度</span>
        <strong>{{ averageProgress }}%</strong>
      </article>
      <article>
        <CheckCircle2 :size="18" />
        <span>已完成</span>
        <strong>{{ completedJobs }} 个</strong>
      </article>
    </section>

    <main class="backup-sync__main">
      <aside class="backup-sync__jobs" aria-label="备份任务列表">
        <header>
          <h3><DatabaseBackup :size="15" /> 备份任务</h3>
          <span>{{ jobs.length }} 组</span>
        </header>
        <button
          v-for="job in jobs"
          :key="job.id"
          class="backup-sync__job"
          :class="{ 'backup-sync__job--active': job.id === selectedJobId }"
          type="button"
          @click="selectJob(job.id)"
        >
          <strong>{{ job.name }}</strong>
          <span>{{ job.source }} -> {{ job.target }}</span>
          <small>{{ job.state }} · {{ job.progress }}% · {{ job.nextRun }}</small>
        </button>
      </aside>

      <section class="backup-sync__detail" aria-label="备份任务详情">
        <header>
          <div>
            <p>{{ selectedJob.source }} -> {{ selectedJob.target }}</p>
            <h3>{{ selectedJob.name }}</h3>
          </div>
          <strong>{{ selectedJob.state }}</strong>
        </header>

        <div class="backup-sync__meter" aria-label="备份进度">
          <span :style="{ width: `${selectedJob.progress}%` }" />
        </div>

        <div class="backup-sync__grid">
          <article>
            <span>速度</span>
            <strong>{{ selectedJob.speed }}</strong>
          </article>
          <article>
            <span>预计</span>
            <strong>{{ selectedJob.eta }}</strong>
          </article>
          <article>
            <span>保留</span>
            <strong>{{ selectedJob.retention }}</strong>
          </article>
          <article>
            <span>健康</span>
            <strong>{{ selectedJob.health }}</strong>
          </article>
        </div>

        <p class="backup-sync__policy"><ShieldCheck :size="14" /> {{ selectedJob.policy }}</p>

        <div class="backup-sync__actions">
          <button type="button" @click="runBackupJob()"><Play :size="14" /> 立即运行</button>
          <button v-if="selectedJob.state !== '已暂停'" type="button" @click="pauseBackupJob()"><Pause :size="14" /> 暂停</button>
          <button v-else type="button" @click="resumeBackupJob()"><Play :size="14" /> 恢复</button>
          <button type="button" @click="verifyBackupJob()"><RefreshCw :size="14" /> 校验</button>
        </div>
      </section>
    </main>

    <section class="backup-sync__audit" aria-label="备份操作反馈">
      <ShieldCheck :size="15" />
      <span>{{ actionState }}</span>
    </section>
  </div>
</template>

<style scoped>
.backup-sync {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.backup-sync__summary {
  display: grid;
  grid-template-columns: repeat(3, minmax(0, 1fr));
  gap: 1px;
  overflow: hidden;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.backup-sync__summary article {
  display: grid;
  gap: 4px;
  justify-items: center;
  padding: 12px 8px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.56);
}

.backup-sync__summary span,
.backup-sync__grid span,
.backup-sync__jobs header span,
.backup-sync__detail header p {
  color: var(--text-soft);
  font-size: 11px;
}

.backup-sync__summary strong,
.backup-sync__grid strong {
  color: var(--text-strong);
  font-size: 13px;
}

.backup-sync__main {
  display: grid;
  grid-template-columns: 250px minmax(0, 1fr);
  gap: 12px;
  min-height: 0;
}

.backup-sync__jobs,
.backup-sync__detail,
.backup-sync__audit {
  min-width: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.backup-sync__jobs {
  display: grid;
  grid-template-rows: auto minmax(0, 1fr);
  overflow: hidden;
}

.backup-sync__jobs header,
.backup-sync__detail header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.backup-sync h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: 13px;
}

.backup-sync__job {
  display: grid;
  gap: 5px;
  padding: 12px;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.backup-sync__job--active {
  background: rgba(19, 136, 255, 0.08);
  box-shadow: inset 3px 0 0 var(--accent);
}

.backup-sync__job strong,
.backup-sync__job span,
.backup-sync__job small {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.backup-sync__job strong {
  color: var(--text-strong);
  font-size: 12px;
}

.backup-sync__job span,
.backup-sync__job small {
  color: var(--text-muted);
  font-size: 11px;
}

.backup-sync__detail {
  display: grid;
  grid-template-rows: auto auto auto auto auto;
  align-content: start;
  overflow: hidden;
}

.backup-sync__detail header strong {
  color: var(--accent);
  font-size: 12px;
}

.backup-sync__meter {
  height: 12px;
  margin: 16px 14px 12px;
  overflow: hidden;
  background: rgba(100, 136, 166, 0.14);
  border-radius: 999px;
}

.backup-sync__meter span {
  display: block;
  height: 100%;
  background: linear-gradient(90deg, var(--accent), var(--accent-green));
}

.backup-sync__grid {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 8px;
  padding: 0 14px 12px;
}

.backup-sync__grid article {
  display: grid;
  gap: 5px;
  min-width: 0;
  padding: 10px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.12);
  border-radius: var(--radius-sm);
}

.backup-sync__policy,
.backup-sync__audit {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 14px 12px;
  padding: 9px 10px;
  color: var(--text-muted);
  background: rgba(231, 247, 255, 0.62);
  border: 1px solid rgba(19, 136, 255, 0.12);
  border-radius: var(--radius-sm);
  font-size: 11px;
}

.backup-sync__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 14px;
}

.backup-sync__actions button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.backup-sync__audit {
  margin: 0;
  color: var(--text-strong);
}
</style>
