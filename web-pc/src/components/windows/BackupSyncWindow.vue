<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { ArchiveRestore, CheckCircle2, CloudUpload, DatabaseBackup, Pause, Play, RefreshCw, ShieldCheck } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { BackupJob } from '../../api/types';
import NasFeaturePanel from '../NasFeaturePanel.vue';
import { UiButton, UiProgressBar, UiWindowPage, UiNavRail, UiNavItem, UiStatGrid, UiStat } from '../ui';

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

const scheduleHours = ref(6);
watch(
  selectedJobId,
  () => {
    scheduleHours.value = selectedJob.value?.intervalHours || 6;
  },
  { immediate: true },
);

async function saveBackupSchedule(enabled: boolean, intervalHours: number) {
  const id = selectedJob.value.id;
  await mutateBackupJob(
    id,
    () => apiClient.backup.setSchedule(id, { enabled, intervalHours: Math.max(0, Number(intervalHours) || 0) }),
    enabled ? `已开启自动备份：每 ${intervalHours} 小时` : '已切换为手动备份',
  );
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
  <UiWindowPage
    layout="master-detail"
    :icon="DatabaseBackup"
    title="备份与同步"
    :subtitle="selectedJob.name"
    :status="actionState"
  >
    <template #nav>
      <UiNavRail title="备份任务" :subtitle="`${jobs.length} 组`">
        <UiNavItem
          v-for="job in jobs"
          :key="job.id"
          :label="job.name"
          :hint="`${job.state} · ${job.progress}% · ${job.nextRun}`"
          :active="job.id === selectedJobId"
          @select="selectJob(job.id)"
        />
      </UiNavRail>
    </template>

    <UiStatGrid>
      <UiStat :icon="CloudUpload" label="进行中" :value="`${activeJobs} 个`" tone="primary" />
      <UiStat :icon="ArchiveRestore" label="平均进度" :value="`${averageProgress}%`" tone="info" />
      <UiStat :icon="CheckCircle2" label="已完成" :value="`${completedJobs} 个`" tone="success" />
    </UiStatGrid>

    <NasFeaturePanel :modules="['backup', 'sync']" />

    <template #inspector>
      <section class="backup-sync__detail" aria-label="备份任务详情">
        <header>
          <div>
            <p>{{ selectedJob.source }} -> {{ selectedJob.target }}</p>
            <h3>{{ selectedJob.name }}</h3>
          </div>
          <strong>{{ selectedJob.state }}</strong>
        </header>

        <div class="backup-sync__meter">
          <UiProgressBar :value="selectedJob.progress" size="lg" aria-label="备份进度" />
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
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="Play" @click="runBackupJob()">立即运行</UiButton>
          <UiButton v-if="selectedJob.state !== '已暂停'" variant="soft" tone="primary" size="sm" :icon-left="Pause" @click="pauseBackupJob()">暂停</UiButton>
          <UiButton v-else variant="soft" tone="primary" size="sm" :icon-left="Play" @click="resumeBackupJob()">恢复</UiButton>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="verifyBackupJob()">校验</UiButton>
        </div>

        <div class="backup-sync__schedule">
          <label class="backup-sync__schedule-toggle">
            <input type="checkbox" :checked="selectedJob.enabled" @change="saveBackupSchedule(!selectedJob.enabled, scheduleHours)" />
            <span>自动备份</span>
          </label>
          <label>每 <input v-model.number="scheduleHours" type="number" min="1" :disabled="!selectedJob.enabled" /> 小时</label>
          <UiButton variant="soft" size="sm" :disabled="!selectedJob.enabled" @click="saveBackupSchedule(true, scheduleHours)">保存计划</UiButton>
        </div>
      </section>
    </template>
  </UiWindowPage>
</template>

<style scoped>
.backup-sync__grid span,
.backup-sync__detail header p {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}

.backup-sync__grid strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.backup-sync__detail {
  display: grid;
  grid-template-rows: auto auto auto auto auto;
  align-content: start;
  overflow: hidden;
  min-width: 0;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.backup-sync__detail header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid var(--border);
}

.backup-sync__detail h3 {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.backup-sync__detail header strong {
  color: var(--accent);
  font-size: var(--fs-xs);
}

.backup-sync__meter {
  margin: 16px 14px 12px;
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
  background: rgba(var(--surface-rgb), 0.58);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
}

.backup-sync__policy {
  display: flex;
  align-items: center;
  gap: 7px;
  margin: 0 14px 12px;
  padding: 9px 10px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.62);
  border: 1px solid var(--accent-soft);
  border-radius: var(--radius-control);
  font-size: var(--fs-2xs);
}

.backup-sync__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  padding: 0 14px 14px;
}

.backup-sync__schedule {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: 10px;
  margin: 0 14px 14px;
  padding: 8px 10px;
  border-radius: 10px;
  background: var(--surface-2, rgba(0, 0, 0, 0.03));
  font-size: var(--fs-sm, 13px);
}
.backup-sync__schedule-toggle {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: var(--fw-semibold);
}
.backup-sync__schedule input[type='number'] {
  width: 52px;
  padding: 3px 6px;
  border: 1px solid var(--border, rgba(0, 0, 0, 0.12));
  border-radius: 6px;
  background: var(--surface-1, var(--surface-solid));
}
</style>
