<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Cpu,
  MemoryStick,
  Play,
  Power,
  RefreshCcw,
  RotateCcw,
  ServerCog,
  ShieldAlert,
  Square,
  Trash2,
  Zap,
} from 'lucide-vue-next';
import { UiBadge, UiButton, UiEmptyState, UiSpinner, UiWindowPage, useConfirm, useToast } from '../ui';
import type { UiTone } from '../ui';
import { vmStore } from '../../stores/vm';
import type { VM } from '../../api/types';

const store = vmStore;
const toast = useToast();
const confirm = useConfirm();
const busyName = ref('');

const machines = computed(() => store.machines.value);
const caps = computed(() => store.caps.value);
const audit = computed(() => store.audit.value);
const loading = computed(() => store.loading.value);
const usingFallback = computed(() => store.usingFallback.value);
const auditOpen = ref(false);
const runningCount = computed(() => machines.value.filter((m) => m.state === 'running').length);

onMounted(() => store.load());

function stateTone(state: string): UiTone {
  if (state === 'running') return 'success';
  if (state === 'paused') return 'warning';
  return 'neutral';
}

function isRunning(m: VM) {
  return m.state === 'running';
}

function messageOf(e: unknown) {
  return e instanceof Error ? e.message : '操作失败';
}

async function run(m: VM, action: string, label: string) {
  if (busyName.value) return;
  busyName.value = m.name;
  try {
    await store.action(m.name, action);
    toast.show(`${m.name} ${label}`, { tone: 'success' });
  } catch (e) {
    toast.show(messageOf(e), { tone: 'danger' });
  } finally {
    busyName.value = '';
  }
}

async function onForceStop(m: VM) {
  const ok = await confirm({
    title: '强制停止虚拟机？',
    message: `将立即切断「${m.name}」的电源（相当于拔电），未保存数据可能丢失。`,
    confirmLabel: '强制停止',
    tone: 'danger',
  });
  if (ok) await run(m, 'force-stop', '已强制停止');
}

async function onDelete(m: VM) {
  const ok = await confirm({
    title: '删除虚拟机？',
    message: `将删除虚拟机定义「${m.name}」（磁盘镜像保留）。此操作不可撤销。`,
    confirmLabel: '删除',
    tone: 'danger',
  });
  if (ok) await run(m, 'delete', '已删除');
}
</script>

<template>
  <UiWindowPage
    layout="stack"
    :icon="ServerCog"
    title="虚拟机"
    subtitle="虚拟机 · KVM / libvirt"
  >
    <template #actions>
      <UiBadge :tone="caps?.libvirtAvailable ? 'success' : 'neutral'" variant="soft">
        libvirt {{ caps?.libvirtAvailable ? '可用' : '未安装' }}
      </UiBadge>
      <UiBadge :tone="caps?.kvmAvailable ? 'success' : 'warning'" variant="soft">
        KVM {{ caps?.kvmAvailable ? '已启用' : '不可用' }}
      </UiBadge>
      <UiBadge tone="info" variant="soft">{{ runningCount }} 台运行中</UiBadge>
      <UiButton variant="ghost" size="sm" :icon-left="RefreshCcw" :disabled="loading" @click="store.load()">刷新</UiButton>
    </template>

    <p v-if="usingFallback" class="vm__offline">
      <ShieldAlert :size="14" /> 暂时无法连接后端，展示的是本地占位数据，操作不会生效。
    </p>
    <p v-else-if="caps?.note" class="vm__note"><Zap :size="13" /> {{ caps.note }}</p>

    <div class="vm__list">
      <div v-if="loading && !machines.length" class="vm__loading"><UiSpinner /></div>
      <UiEmptyState
        v-else-if="!machines.length"
        title="暂无虚拟机"
        :description="caps?.libvirtAvailable ? '主机 libvirt 已就绪，但还没有定义虚拟机。' : '主机未安装 libvirt/KVM。'"
      />

      <article v-for="m in machines" :key="m.name" class="vm-card">
        <div class="vm-card__head">
          <div class="vm-card__title">
            <strong>{{ m.name }}</strong>
            <small v-if="m.title">{{ m.title }}</small>
          </div>
          <UiBadge :tone="stateTone(m.state)" variant="dot">{{ m.state }}</UiBadge>
        </div>
        <div class="vm-card__specs">
          <span><Cpu :size="13" /> {{ m.vcpus }} vCPU</span>
          <span><MemoryStick :size="13" /> {{ (m.memoryMB / 1024).toFixed(1) }} GB</span>
          <span v-if="m.autostart" class="vm-card__autostart">自启动</span>
        </div>
        <div class="vm-card__actions">
          <UiButton
            v-if="!isRunning(m)"
            variant="soft"
            tone="primary"
            size="sm"
            :icon-left="Play"
            :disabled="busyName === m.name"
            @click="run(m, 'start', '已启动')"
          >
            启动
          </UiButton>
          <template v-else>
            <UiButton variant="soft" size="sm" :icon-left="Power" :disabled="busyName === m.name" @click="run(m, 'shutdown', '已关机')">
              关机
            </UiButton>
            <UiButton variant="ghost" size="sm" :icon-left="RotateCcw" :disabled="busyName === m.name" @click="run(m, 'reboot', '已重启')">
              重启
            </UiButton>
            <UiButton variant="ghost" tone="warning" size="sm" :icon-left="Square" :disabled="busyName === m.name" @click="onForceStop(m)">
              强制停止
            </UiButton>
          </template>
          <UiButton variant="ghost" tone="danger" size="sm" :icon-left="Trash2" :disabled="busyName === m.name" @click="onDelete(m)">
            删除
          </UiButton>
        </div>
      </article>
    </div>

    <section class="vm__audit">
      <button type="button" class="vm__audit-toggle" @click="auditOpen = !auditOpen">
        <RotateCcw :size="13" /> 操作记录 · {{ audit.length }} 条
        <span class="vm__audit-caret">{{ auditOpen ? '收起' : '展开' }}</span>
      </button>
      <ul v-if="auditOpen" class="audit-list">
        <li v-for="e in audit" :key="e.id" class="audit-row">
          <UiBadge :tone="e.result === 'ok' ? 'success' : 'danger'" variant="soft" size="sm">{{ e.result }}</UiBadge>
          <span>{{ e.event }}</span>
        </li>
      </ul>
    </section>
  </UiWindowPage>
</template>

<style scoped>
.vm__offline,
.vm__note {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  border-radius: var(--radius-card);
  font-size: var(--fs-xs);
}

.vm__offline {
  color: var(--ink-orange);
  background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-orange) 30%, transparent);
}

.vm__note {
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
}

.vm__list {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: var(--space-3);
  flex: 1;
  min-height: 0;
  overflow: auto;
  align-content: start;
}

.vm__loading {
  grid-column: 1 / -1;
  display: flex;
  justify-content: center;
  padding: var(--space-6);
}

.vm-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: var(--space-3);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.vm-card__head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: var(--space-2);
}

.vm-card__title {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.vm-card__title strong {
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.vm-card__title small {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.vm-card__specs {
  display: flex;
  align-items: center;
  gap: var(--space-3);
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.vm-card__specs span {
  display: flex;
  align-items: center;
  gap: 4px;
}

.vm-card__autostart {
  color: var(--ink-green);
}

.vm-card__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.vm__audit {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}

.vm__audit-toggle {
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

.vm__audit-caret {
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
</style>
