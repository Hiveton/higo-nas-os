<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  ArrowLeftRight,
  ArrowRight,
  CheckCircle2,
  FolderSync,
  Play,
  Plus,
  RefreshCcw,
  RotateCcw,
  ShieldAlert,
  ShieldCheck,
  Trash2,
} from 'lucide-vue-next';
import {
  UiBadge,
  UiButton,
  UiEmptyState,
  UiFormField,
  UiInput,
  UiModal,
  UiSelect,
  UiSpinner,
  UiSwitch,
  UiWindowPage,
  useConfirm,
  useToast,
} from '../ui';
import type { SelectOption, UiTone } from '../ui';
import { syncStore } from '../../stores/sync';
import type { SyncConflict, SyncDirection, SyncPair } from '../../api/types';

const store = syncStore;
const toast = useToast();
const confirm = useConfirm();

const selectedId = ref('');
const busy = ref(false);
const auditOpen = ref(false);

const pairs = computed(() => store.pairs.value);
const audit = computed(() => store.audit.value);
const loading = computed(() => store.loading.value);
const usingFallback = computed(() => store.usingFallback.value);
const selected = computed<SyncPair | null>(() => pairs.value.find((p) => p.id === selectedId.value) ?? null);
const selectedConflicts = computed<SyncConflict[]>(() =>
  store.pendingConflicts.value.filter((c) => c.pairId === selectedId.value),
);

onMounted(async () => {
  await store.load();
  if (pairs.value.length && !pairs.value.some((p) => p.id === selectedId.value)) {
    selectedId.value = pairs.value[0].id;
  }
});

const directionOptions: SelectOption[] = [
  { label: '单向镜像（源 → 目标）', value: 'mirror' },
  { label: '双向同步（合并较新）', value: 'two-way' },
];
const conflictOptions: SelectOption[] = [
  { label: '保留较新文件', value: 'newer' },
  { label: '源端优先', value: 'source' },
  { label: '目标端优先', value: 'target' },
  { label: '手动处理', value: 'manual' },
];

function directionLabel(d: SyncDirection) {
  return d === 'two-way' ? '双向' : '单向镜像';
}

function stateTone(state: string): UiTone {
  switch (state) {
    case '已完成':
      return 'success';
    case '同步中':
      return 'info';
    case '失败':
      return 'danger';
    case '有冲突':
      return 'warning';
    default:
      return 'neutral';
  }
}

function resultTone(result: string): UiTone {
  if (result === 'ok') return 'success';
  if (result === 'conflict') return 'warning';
  if (result === 'blocked') return 'danger';
  return 'neutral';
}

function messageOf(e: unknown) {
  return e instanceof Error ? e.message : '操作失败';
}

async function withBusy(fn: () => Promise<unknown>, ok?: string) {
  if (busy.value) return;
  busy.value = true;
  try {
    await fn();
    if (ok) toast.show(ok, { tone: 'success' });
  } catch (e) {
    toast.show(messageOf(e), { tone: 'danger' });
  } finally {
    busy.value = false;
  }
}

async function onToggleEnabled(p: SyncPair, enabled: boolean) {
  await withBusy(() => store.updatePair(p.id, { enabled }), `${p.name} ${enabled ? '已启用' : '已暂停'}`);
}

async function onRun(p: SyncPair) {
  await withBusy(() => store.runPair(p.id), `已开始同步：${p.name}`);
}

async function onVerify(p: SyncPair) {
  await withBusy(() => store.verifyPair(p.id), `已开始校验：${p.name}`);
}

async function onDelete(p: SyncPair) {
  const okToDelete = await confirm({
    title: '删除同步任务？',
    message: `将删除「${p.name}」的同步配置（已同步的文件保留）。`,
    confirmLabel: '删除',
    tone: 'danger',
  });
  if (!okToDelete) return;
  await withBusy(async () => {
    await store.deletePair(p.id);
    if (selectedId.value === p.id) selectedId.value = pairs.value[0]?.id ?? '';
  }, '同步任务已删除');
}

async function onResolve(c: SyncConflict, side: 'source' | 'target') {
  await withBusy(() => store.resolveConflict(c.id, side), '冲突已解决');
}

// --- detail config edits (inline, saved on change) --------------------------

async function saveField(field: string, value: unknown) {
  const p = selected.value;
  if (!p) return;
  await withBusy(() => store.updatePair(p.id, { [field]: value }), '设置已保存');
}

// --- add modal --------------------------------------------------------------

const showAdd = ref(false);
const form = reactive({
  name: '',
  source: '',
  target: '',
  direction: 'mirror' as SyncDirection,
  conflictPolicy: 'newer',
  includes: '',
  intervalHours: 0,
});

function openAdd() {
  form.name = '';
  form.source = '';
  form.target = '';
  form.direction = 'mirror';
  form.conflictPolicy = 'newer';
  form.includes = '';
  form.intervalHours = 0;
  showAdd.value = true;
}

async function submitAdd() {
  if (!form.name.trim() || !form.source.trim() || !form.target.trim()) {
    toast.show('请填写名称、源目录与目标目录', { tone: 'warning' });
    return;
  }
  await withBusy(async () => {
    const created = await store.createPair({
      name: form.name.trim(),
      source: form.source.trim(),
      target: form.target.trim(),
      direction: form.direction,
      conflictPolicy: form.conflictPolicy,
      includes: form.includes
        .split(/[,，\n]+/)
        .map((s) => s.trim())
        .filter(Boolean),
      intervalHours: Number(form.intervalHours) || 0,
      actor: 'sync-ui',
    });
    selectedId.value = created.id;
    showAdd.value = false;
  }, '同步任务已创建');
}
</script>

<template>
  <UiWindowPage
    layout="stack"
    :icon="FolderSync"
    title="同步服务"
    subtitle="设备同步 · 按需同步 · 冲突处理"
  >
    <template #actions>
      <UiBadge v-if="store.pendingConflicts.value.length" tone="warning" variant="soft">
        {{ store.pendingConflicts.value.length }} 个冲突待处理
      </UiBadge>
      <UiButton variant="ghost" size="sm" :icon-left="RefreshCcw" :disabled="loading" @click="store.load()">刷新</UiButton>
      <UiButton variant="solid" tone="primary" size="sm" :icon-left="Plus" @click="openAdd">新增同步</UiButton>
    </template>

    <p v-if="usingFallback" class="sync__offline">
      <ShieldAlert :size="14" /> 暂时无法连接后端，展示的是本地占位数据，操作不会生效。
    </p>

    <div class="sync__body">
      <aside class="sync__list">
        <UiEmptyState
          v-if="!pairs.length && !loading"
          title="暂无同步任务"
          description="点击「新增同步」创建第一个目录同步。"
        />
        <article
          v-for="p in pairs"
          :key="p.id"
          class="pair-card"
          :class="{ 'pair-card--active': p.id === selectedId }"
          role="button"
          tabindex="0"
          @click="selectedId = p.id"
          @keydown.enter="selectedId = p.id"
        >
          <div class="pair-card__top">
            <strong>{{ p.name }}</strong>
            <UiSwitch
              size="sm"
              :model-value="p.enabled"
              :disabled="busy"
              @click.stop
              @change="(v: boolean) => onToggleEnabled(p, v)"
            />
          </div>
          <div class="pair-card__path">
            <span>{{ p.source }}</span>
            <ArrowLeftRight v-if="p.direction === 'two-way'" :size="13" />
            <ArrowRight v-else :size="13" />
            <span>{{ p.target }}</span>
          </div>
          <div class="pair-card__foot">
            <UiBadge :tone="stateTone(p.state)" variant="dot" size="sm">{{ p.state }}</UiBadge>
            <span>{{ directionLabel(p.direction) }}</span>
            <span v-if="p.lastStats" class="pair-card__sep">·</span>
            <span v-if="p.lastStats">复制 {{ p.lastStats.copied }}</span>
          </div>
        </article>
      </aside>

      <section v-if="selected" class="sync__detail">
        <div class="detail-head">
          <div>
            <h4>{{ selected.name }}</h4>
            <UiBadge :tone="stateTone(selected.state)" variant="soft" size="sm">{{ selected.state }}</UiBadge>
          </div>
          <div class="detail-head__actions">
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="Play" :disabled="busy" @click="onRun(selected)">立即同步</UiButton>
            <UiButton variant="ghost" size="sm" :icon-left="ShieldCheck" :disabled="busy" @click="onVerify(selected)">校验</UiButton>
            <UiButton variant="ghost" tone="danger" size="sm" :icon-left="Trash2" :disabled="busy" @click="onDelete(selected)">删除</UiButton>
          </div>
        </div>

        <div class="detail-paths">
          <code>{{ selected.source }}</code>
          <ArrowLeftRight v-if="selected.direction === 'two-way'" :size="15" />
          <ArrowRight v-else :size="15" />
          <code>{{ selected.target }}</code>
        </div>

        <div class="detail-grid">
          <UiFormField label="同步方向">
            <UiSelect
              :model-value="selected.direction"
              :options="directionOptions"
              @update:model-value="(v) => saveField('direction', v)"
            />
          </UiFormField>
          <UiFormField label="冲突策略">
            <UiSelect
              :model-value="selected.conflictPolicy"
              :options="conflictOptions"
              @update:model-value="(v) => saveField('conflictPolicy', v)"
            />
          </UiFormField>
          <UiFormField label="自动间隔（小时，0=手动）">
            <UiInput
              :model-value="String(selected.intervalHours ?? 0)"
              @change="(e: Event) => saveField('intervalHours', Number((e.target as HTMLInputElement).value) || 0)"
            />
          </UiFormField>
          <UiFormField label="带宽限制（提示）">
            <UiInput
              :model-value="selected.bandwidthLimit ?? ''"
              placeholder="如 10 MB/s"
              @change="(e: Event) => saveField('bandwidthLimit', (e.target as HTMLInputElement).value)"
            />
          </UiFormField>
        </div>

        <div v-if="selected.lastStats" class="detail-stats">
          <CheckCircle2 :size="14" />
          上次：复制 {{ selected.lastStats.copied }} · 跳过 {{ selected.lastStats.skipped }} ·
          {{ (selected.lastStats.bytes / 1024).toFixed(1) }} KB
          <span v-if="selected.lastStats.conflicts"> · 冲突 {{ selected.lastStats.conflicts }}</span>
        </div>

        <!-- Conflicts -->
        <section v-if="selectedConflicts.length" class="detail-conflicts">
          <h5><ShieldAlert :size="14" /> 待处理冲突</h5>
          <article v-for="c in selectedConflicts" :key="c.id" class="conflict-row">
            <div class="conflict-row__main">
              <strong>{{ c.relPath }}</strong>
              <small>{{ c.detail }}</small>
            </div>
            <div class="conflict-row__actions">
              <UiButton variant="ghost" size="sm" :disabled="busy" @click="onResolve(c, 'source')">采用源端</UiButton>
              <UiButton variant="ghost" size="sm" :disabled="busy" @click="onResolve(c, 'target')">采用目标端</UiButton>
            </div>
          </article>
        </section>
      </section>

      <section v-else class="sync__detail sync__detail--empty">
        <UiSpinner v-if="loading" />
        <UiEmptyState v-else title="选择一个同步任务" description="从左侧选择，或新增一个同步任务。" />
      </section>
    </div>

    <section class="sync__audit">
      <button type="button" class="sync__audit-toggle" @click="auditOpen = !auditOpen">
        <RotateCcw :size="13" /> 同步记录 · {{ audit.length }} 条
        <span class="sync__audit-caret">{{ auditOpen ? '收起' : '展开' }}</span>
      </button>
      <ul v-if="auditOpen" class="audit-list">
        <li v-for="e in audit" :key="e.id" class="audit-row">
          <UiBadge :tone="resultTone(e.result)" variant="soft" size="sm">{{ e.result }}</UiBadge>
          <span>{{ e.event }}</span>
        </li>
      </ul>
    </section>

    <UiModal v-model:open="showAdd" title="新增同步任务" size="sm">
      <div class="add-form">
        <UiFormField label="任务名称" required>
          <UiInput v-model="form.name" placeholder="例如：工作目录 → 备份盘" />
        </UiFormField>
        <UiFormField label="源目录" required hint="需位于 NAS 存储根目录之内">
          <UiInput v-model="form.source" placeholder="/家庭空间/工作" />
        </UiFormField>
        <UiFormField label="目标目录" required>
          <UiInput v-model="form.target" placeholder="/备份/工作" />
        </UiFormField>
        <UiFormField label="同步方向">
          <UiSelect v-model="form.direction" :options="directionOptions" />
        </UiFormField>
        <UiFormField v-if="form.direction === 'two-way'" label="冲突策略">
          <UiSelect v-model="form.conflictPolicy" :options="conflictOptions" />
        </UiFormField>
        <UiFormField label="选择性同步（可选）" hint="glob 模式，逗号分隔，如 docs/*, *.jpg">
          <UiInput v-model="form.includes" placeholder="留空 = 同步全部" />
        </UiFormField>
        <UiFormField label="自动间隔（小时，0=手动）">
          <UiInput v-model="form.intervalHours" type="number" />
        </UiFormField>
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="showAdd = false">取消</UiButton>
        <UiButton variant="solid" tone="primary" :loading="busy" @click="submitAdd">创建</UiButton>
      </template>
    </UiModal>
  </UiWindowPage>
</template>

<style scoped>
.sync__offline {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  color: var(--ink-orange);
  background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-orange) 30%, transparent);
  border-radius: var(--radius-card);
  font-size: var(--fs-xs);
}

.sync__body {
  display: grid;
  grid-template-columns: 268px 1fr;
  gap: var(--space-3);
  flex: 1;
  min-height: 0;
}

.sync__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  overflow: auto;
  padding-right: 2px;
}

.pair-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: var(--space-3);
  text-align: left;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.pair-card:hover {
  border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
}

.pair-card--active {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, rgba(var(--surface-rgb), 0.5));
}

.pair-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.pair-card__top strong {
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.pair-card__path {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  overflow: hidden;
}

.pair-card__path span {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.pair-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.pair-card__sep {
  opacity: 0.5;
}

.sync__detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  overflow: auto;
  padding: var(--space-4);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.sync__detail--empty {
  align-items: center;
  justify-content: center;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.detail-head > div:first-child {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.detail-head h4 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-lg);
}

.detail-head__actions {
  display: flex;
  gap: var(--space-2);
}

.detail-paths {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.detail-paths code {
  flex: 1;
  padding: 6px var(--space-2);
  background: rgba(var(--surface-rgb), 0.7);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  color: var(--text-strong);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.detail-stats {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--ink-green);
  font-size: var(--fs-xs);
}

.detail-conflicts {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  background: color-mix(in srgb, var(--accent-orange) 8%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-orange) 25%, transparent);
  border-radius: var(--radius-card);
}

.detail-conflicts h5 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.conflict-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
  padding: var(--space-2);
  background: rgba(var(--surface-rgb), 0.6);
  border-radius: var(--radius-control);
}

.conflict-row__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.conflict-row__main strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.conflict-row__main small {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.conflict-row__actions {
  display: flex;
  gap: 6px;
  flex-shrink: 0;
}

.sync__audit {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.sync__audit-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-2) var(--space-3);
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  cursor: pointer;
}

.sync__audit-caret {
  margin-left: auto;
  color: var(--accent);
}

.audit-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding: 0 var(--space-3) var(--space-3);
  list-style: none;
  max-height: 150px;
  overflow: auto;
}

.audit-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border-top: 1px solid var(--border);
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.add-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

@media (max-width: 760px) {
  .sync__body {
    grid-template-columns: 1fr;
  }
  .detail-grid {
    grid-template-columns: 1fr;
  }
}
</style>
