<script setup lang="ts">
import { ref } from 'vue';
import { Boxes, FileCode, Play, RefreshCw, Square, X } from 'lucide-vue-next';
import type { ComposeStack } from '../../../api/types';
import { apiClient } from '../../../api/client';
import { UiButton, UiEmptyState, useConfirm } from '../../ui';

defineProps<{
  composeStacks: ComposeStack[];
  containers: { id: string }[];
  loading: boolean;
}>();

const emit = defineEmits<{
  (e: 'refresh'): void;
  (e: 'view-containers', name: string): void;
}>();

const confirm = useConfirm();

const SAMPLE_YAML = `services:
  web:
    image: nginx:latest
    ports:
      - "8088:80"
    restart: unless-stopped
`;

const editorOpen = ref(false);
const editorTitle = ref('部署新 Compose 栈');
const draftName = ref('');
const draftYaml = ref(SAMPLE_YAML);
const busy = ref('');
const notice = ref('');

function openNew() {
  editorTitle.value = '部署新 Compose 栈';
  draftName.value = '';
  draftYaml.value = SAMPLE_YAML;
  notice.value = '';
  editorOpen.value = true;
}

async function openEdit(stack: ComposeStack) {
  editorTitle.value = `编辑 ${stack.name}`;
  draftName.value = stack.name;
  notice.value = '';
  editorOpen.value = true;
  try {
    const res = await apiClient.docker.getStackYaml(stack.name);
    draftYaml.value = res.yaml || stack.yaml || SAMPLE_YAML;
  } catch {
    draftYaml.value = stack.yaml || SAMPLE_YAML;
  }
}

async function deploy() {
  if (!draftName.value.trim()) {
    notice.value = '请填写栈名称。';
    return;
  }
  if (!draftYaml.value.trim()) {
    notice.value = '请填写 compose YAML。';
    return;
  }
  busy.value = 'deploy';
  try {
    await apiClient.docker.deployStack({ name: draftName.value.trim(), yaml: draftYaml.value });
    notice.value = '';
    editorOpen.value = false;
    emit('refresh');
  } catch (error) {
    notice.value = `部署失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busy.value = '';
  }
}

async function down(stack: ComposeStack) {
  const ok = await confirm({
    title: `停止 ${stack.name}？`,
    message: '将执行 docker compose down，停止该栈的全部容器。',
    confirmLabel: '停止',
    tone: 'danger',
  });
  if (!ok) return;
  busy.value = `down-${stack.name}`;
  try {
    await apiClient.docker.downStack(stack.name);
    emit('refresh');
  } catch (error) {
    notice.value = `停止失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busy.value = '';
  }
}
</script>

<template>
  <section class="docker-panel docker-compose-page">
    <header class="docker-panel__header">
      <div>
        <h3>Compose</h3>
        <p>{{ composeStacks.length }} 个栈，{{ containers.length }} 个容器。</p>
      </div>
      <div class="docker-compose__head-actions">
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="FileCode" @click="openNew">部署新栈</UiButton>
        <UiButton variant="soft" size="sm" :icon-left="RefreshCw" :disabled="loading" @click="emit('refresh')">刷新</UiButton>
      </div>
    </header>

    <div v-if="editorOpen" class="docker-compose__editor">
      <header>
        <strong>{{ editorTitle }}</strong>
        <button type="button" aria-label="关闭编辑器" @click="editorOpen = false"><X :size="16" /></button>
      </header>
      <input v-model="draftName" class="docker-compose__name" placeholder="栈名称，例如 media-stack" spellcheck="false" />
      <textarea v-model="draftYaml" class="docker-compose__yaml" spellcheck="false" rows="12" placeholder="docker-compose.yaml"></textarea>
      <p v-if="notice" class="docker-compose__notice">{{ notice }}</p>
      <div class="docker-compose__editor-actions">
        <UiButton variant="soft" tone="primary" size="sm" :icon-left="Play" :loading="busy === 'deploy'" @click="deploy">部署</UiButton>
        <UiButton variant="ghost" size="sm" @click="editorOpen = false">取消</UiButton>
      </div>
    </div>

    <div class="docker-list docker-list--grid">
      <article v-for="stack in composeStacks" :key="stack.name" class="docker-list-item">
        <div>
          <strong>{{ stack.name }}</strong>
          <p>{{ stack.status }} · {{ stack.services }} 个服务 · {{ stack.ports || '—' }}</p>
          <small>{{ stack.volume }} · {{ stack.network }}</small>
        </div>
        <div class="docker-compose__row-actions">
          <UiButton variant="soft" size="sm" :icon-left="FileCode" @click="openEdit(stack)">YAML</UiButton>
          <UiButton variant="soft" size="sm" @click="emit('view-containers', stack.name)">容器</UiButton>
          <UiButton variant="soft" tone="danger" size="sm" :icon-left="Square" :loading="busy === `down-${stack.name}`" @click="down(stack)">停止</UiButton>
        </div>
      </article>
      <UiEmptyState v-if="composeStacks.length === 0" compact :icon="Boxes" title="暂无 Compose 栈" description="点击「部署新栈」粘贴 docker-compose.yaml 即可部署。" />
    </div>
  </section>
</template>

<style scoped>
.docker-compose__head-actions {
  display: flex;
  gap: 8px;
}
.docker-compose__editor {
  display: grid;
  gap: 8px;
  margin-bottom: 12px;
  padding: 12px;
  background: rgba(var(--surface-rgb), 0.55);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
}
.docker-compose__editor header {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.docker-compose__editor header strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.docker-compose__editor header button {
  display: grid;
  place-items: center;
  width: 26px;
  height: 26px;
  color: var(--text-muted);
  background: transparent;
  border: 0;
  border-radius: var(--radius-control);
  cursor: pointer;
}
.docker-compose__editor header button:hover {
  background: var(--accent-soft);
  color: var(--accent);
}
.docker-compose__name {
  height: 34px;
  padding: 0 12px;
  color: var(--text-strong);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  outline: 0;
  font-family: inherit;
  font-size: var(--fs-xs);
}
.docker-compose__yaml {
  width: 100%;
  padding: 10px 12px;
  color: var(--text-strong);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  outline: 0;
  resize: vertical;
  font-family: var(--font-mono, ui-monospace, monospace);
  font-size: var(--fs-2xs);
  line-height: 1.5;
}
.docker-compose__name:focus,
.docker-compose__yaml:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}
.docker-compose__notice {
  margin: 0;
  color: var(--accent-red);
  font-size: var(--fs-2xs);
}
.docker-compose__editor-actions,
.docker-compose__row-actions {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
</style>
