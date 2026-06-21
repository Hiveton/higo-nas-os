<script setup lang="ts">
import { computed, onMounted, reactive, ref } from 'vue';
import {
  HardDrive,
  Plus,
  RefreshCcw,
  RotateCcw,
  ShieldAlert,
  ShieldCheck,
  Target,
  Trash2,
  UserPlus,
} from 'lucide-vue-next';
import {
  UiBadge,
  UiButton,
  UiEmptyState,
  UiFormField,
  UiInput,
  UiModal,
  UiSpinner,
  UiWindowPage,
  useConfirm,
  useToast,
} from '../ui';
import type { UiTone } from '../ui';
import { iscsiStore } from '../../stores/iscsi';
import type { ISCSITarget } from '../../api/types';

const store = iscsiStore;
const toast = useToast();
const confirm = useConfirm();
const busy = ref(false);
const auditOpen = ref(false);

const targets = computed(() => store.targets.value);
const caps = computed(() => store.caps.value);
const audit = computed(() => store.audit.value);
const loading = computed(() => store.loading.value);
const usingFallback = computed(() => store.usingFallback.value);

onMounted(() => store.load());

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

async function onCreateTarget() {
  await withBusy(() => store.createTarget(), 'iSCSI 目标已创建');
}

async function onDelete(t: ISCSITarget) {
  const ok = await confirm({
    title: '删除 iSCSI 目标？',
    message: `将删除目标「${t.iqn}」及其 ${t.luns} 个 LUN 与全部 ACL。已连接的启动器会断开。此操作不可撤销。`,
    confirmLabel: '删除',
    tone: 'danger',
  });
  if (ok) await withBusy(() => store.deleteTarget(t.iqn), '目标已删除');
}

// --- add LUN modal ----------------------------------------------------------
const lunModal = reactive({ open: false, iqn: '', name: '', sizeMB: 1024 });
function openLun(t: ISCSITarget) {
  lunModal.open = true;
  lunModal.iqn = t.iqn;
  lunModal.name = '';
  lunModal.sizeMB = 1024;
}
async function submitLun() {
  if (!lunModal.name.trim()) {
    toast.show('请填写 LUN 名称', { tone: 'warning' });
    return;
  }
  await withBusy(async () => {
    await store.addLun(lunModal.iqn, lunModal.name.trim(), Number(lunModal.sizeMB) || 1024);
    lunModal.open = false;
  }, 'LUN 已添加');
}

// --- add ACL modal ----------------------------------------------------------
const aclModal = reactive({ open: false, iqn: '', initiator: '' });
function openAcl(t: ISCSITarget) {
  aclModal.open = true;
  aclModal.iqn = t.iqn;
  aclModal.initiator = '';
}
async function submitAcl() {
  if (!aclModal.initiator.trim()) {
    toast.show('请填写启动器 IQN', { tone: 'warning' });
    return;
  }
  await withBusy(async () => {
    await store.addAcl(aclModal.iqn, aclModal.initiator.trim());
    aclModal.open = false;
  }, '启动器已授权');
}

function capTone(): UiTone {
  return caps.value?.available ? 'success' : 'neutral';
}
</script>

<template>
  <UiWindowPage
    layout="stack"
    :icon="Target"
    title="iSCSI 目标"
    subtitle="iSCSI · LIO / targetcli"
  >
    <template #actions>
      <UiBadge :tone="capTone()" variant="soft">{{ caps?.available ? `${caps.backend} 可用` : '未安装' }}</UiBadge>
      <UiBadge tone="info" variant="soft">{{ targets.length }} 个目标</UiBadge>
      <UiButton variant="ghost" size="sm" :icon-left="RefreshCcw" :disabled="loading" @click="store.load()">刷新</UiButton>
      <UiButton variant="solid" tone="primary" size="sm" :icon-left="Plus" :disabled="busy" @click="onCreateTarget">新建目标</UiButton>
    </template>

    <p v-if="usingFallback" class="iscsi__offline">
      <ShieldAlert :size="14" /> 暂时无法连接后端，展示的是本地占位数据，操作不会生效。
    </p>
    <p class="iscsi__risk"><ShieldAlert :size="13" /> iSCSI 直接导出块存储，属高风险操作；删除目标会断开已连接的启动器。</p>

    <div class="iscsi__list">
      <div v-if="loading && !targets.length" class="iscsi__loading"><UiSpinner /></div>
      <UiEmptyState
        v-else-if="!targets.length"
        title="暂无 iSCSI 目标"
        :description="caps?.available ? '点击「新建目标」创建第一个 iSCSI 目标。' : '主机未安装 targetcli / LIO。'"
      />

      <article v-for="t in targets" :key="t.iqn" class="target-card">
        <div class="target-card__head">
          <strong class="target-card__iqn">{{ t.iqn }}</strong>
          <UiButton variant="ghost" tone="danger" size="sm" :icon-left="Trash2" :disabled="busy" @click="onDelete(t)">删除</UiButton>
        </div>
        <div class="target-card__stats">
          <span><HardDrive :size="13" /> {{ t.luns }} 个 LUN</span>
          <span><ShieldCheck :size="13" /> {{ t.acls.length }} 个授权启动器</span>
          <span>{{ (t.portals[0]) || '0.0.0.0:3260' }}</span>
        </div>
        <div v-if="t.acls.length" class="target-card__acls">
          <UiBadge v-for="acl in t.acls" :key="acl" tone="neutral" variant="soft" size="sm">{{ acl }}</UiBadge>
        </div>
        <div class="target-card__actions">
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="HardDrive" :disabled="busy" @click="openLun(t)">添加 LUN</UiButton>
          <UiButton variant="ghost" size="sm" :icon-left="UserPlus" :disabled="busy" @click="openAcl(t)">授权启动器</UiButton>
        </div>
      </article>
    </div>

    <section class="iscsi__audit">
      <button type="button" class="iscsi__audit-toggle" @click="auditOpen = !auditOpen">
        <RotateCcw :size="13" /> 操作记录 · {{ audit.length }} 条
        <span class="iscsi__audit-caret">{{ auditOpen ? '收起' : '展开' }}</span>
      </button>
      <ul v-if="auditOpen" class="audit-list">
        <li v-for="e in audit" :key="e.id" class="audit-row">
          <UiBadge :tone="e.result === 'ok' ? 'success' : 'danger'" variant="soft" size="sm">{{ e.result }}</UiBadge>
          <span>{{ e.event }}</span>
        </li>
      </ul>
    </section>

    <UiModal v-model:open="lunModal.open" title="添加 LUN" size="sm">
      <div class="modal-form">
        <UiFormField label="LUN 名称" required>
          <UiInput v-model="lunModal.name" placeholder="如 disk1" />
        </UiFormField>
        <UiFormField label="容量 (MB)">
          <UiInput v-model="lunModal.sizeMB" type="number" />
        </UiFormField>
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="lunModal.open = false">取消</UiButton>
        <UiButton variant="solid" tone="primary" :loading="busy" @click="submitLun">添加</UiButton>
      </template>
    </UiModal>

    <UiModal v-model:open="aclModal.open" title="授权启动器" size="sm">
      <div class="modal-form">
        <UiFormField label="启动器 IQN" required hint="客户端的 iSCSI initiator IQN">
          <UiInput v-model="aclModal.initiator" placeholder="iqn.1994-05.com.redhat:client" />
        </UiFormField>
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="aclModal.open = false">取消</UiButton>
        <UiButton variant="solid" tone="primary" :loading="busy" @click="submitAcl">授权</UiButton>
      </template>
    </UiModal>
  </UiWindowPage>
</template>

<style scoped>
.iscsi__offline,
.iscsi__risk {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-md);
  font-size: var(--fs-xs);
}

.iscsi__offline {
  color: var(--ink-orange);
  background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-orange) 30%, transparent);
}

.iscsi__risk {
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
}

.iscsi__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  flex: 1;
  min-height: 0;
  overflow: auto;
}

.iscsi__loading {
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.target-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--space-3);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.target-card__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.target-card__iqn {
  color: var(--text-strong);
  font-size: var(--fs-sm);
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.target-card__stats {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.target-card__stats span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.target-card__acls {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.target-card__actions {
  display: flex;
  gap: 6px;
}

.iscsi__audit {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.iscsi__audit-toggle {
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

.iscsi__audit-caret {
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
  max-height: 140px;
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

.modal-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}
</style>
