<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { Component } from 'vue';
import { ArchiveRestore, CheckCircle2, CircleAlert, FileSearch, KeyRound, Play, ShieldCheck, Sparkles } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AgentTemplate, WorkflowNode } from '../../api/types';
import { agentTemplates as seedAgentTemplates, workflowNodes as seedWorkflowNodes } from '../../data/higoos';

type LocalWorkflowNode = WorkflowNode & { icon?: Component };

const agentTemplates = ref<AgentTemplate[]>(seedAgentTemplates);
const workflowNodes = ref<LocalWorkflowNode[]>(seedWorkflowNodes);
const selectedTools = ref<string[]>([]);
const selectedTemplateIndex = ref(1);
const simulationState = ref<'idle' | 'running' | 'done'>('idle');
const executionConfirmed = ref(false);
const loading = ref(false);
const actionMessage = ref('Agent 工作台正在使用本地模板，后端连接后会同步工具权限和工作流执行状态。');
const activeRunId = ref('');
const activeConfirmationId = ref('');

const selectedTemplate = computed(() => agentTemplates.value[selectedTemplateIndex.value] ?? agentTemplates.value[0]);
const visibleTools = computed(() => selectedTools.value.length > 0 ? selectedTools.value : selectedTemplate.value?.tools ?? []);

async function loadAgentWorkbench() {
  loading.value = true;
  try {
    agentTemplates.value = await apiClient.agents.getTemplates();
    selectedTemplateIndex.value = Math.min(selectedTemplateIndex.value, Math.max(agentTemplates.value.length - 1, 0));
    await loadSelectedTools();
    actionMessage.value = 'Agent 模板和工具权限已从后端同步。';
  } catch (error) {
    actionMessage.value = `后端暂不可用，继续使用本地模板：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function selectTemplate(index: number) {
  selectedTemplateIndex.value = index;
  executionConfirmed.value = false;
  simulationState.value = 'idle';
  activeRunId.value = '';
  activeConfirmationId.value = '';
  await loadSelectedTools();
}

async function loadSelectedTools() {
  const template = selectedTemplate.value;
  if (!template?.id) {
    selectedTools.value = template?.tools ?? [];
    return;
  }
  try {
    const tools = await apiClient.agents.getTools(template.id);
    selectedTools.value = tools
      .map((tool) => typeof tool.name === 'string' ? tool.name : '')
      .filter(Boolean);
  } catch {
    selectedTools.value = template.tools ?? [];
  }
}

async function runSimulation() {
  const template = selectedTemplate.value;
  if (!template?.id) return;

  simulationState.value = 'running';
  executionConfirmed.value = false;
  activeRunId.value = '';
  activeConfirmationId.value = '';
  const goal = `${template.name} 试运行：检查当前空间并生成可审计执行计划`;

  try {
    const preview = await apiClient.agents.previewWorkflow({
      templateId: template.id,
      goal,
      scopes: ['files', 'team', 'monitoring', 'backup'],
    });
    workflowNodes.value = mapPreviewNodes(preview.nodes);
    activeConfirmationId.value = typeof preview.confirmationId === 'string' ? preview.confirmationId : '';

    const run = await apiClient.agents.runWorkflow({
      templateId: template.id,
      goal,
      scopes: ['files', 'team', 'monitoring', 'backup'],
    });
    activeRunId.value = run.id;
    activeConfirmationId.value = confirmationFromMessage(run.message) || activeConfirmationId.value;
    simulationState.value = run.state === 'waiting_confirmation' ? 'running' : 'done';
    executionConfirmed.value = run.state === 'completed';
    actionMessage.value = run.message ?? `工作流状态：${run.state}`;
  } catch (error) {
    simulationState.value = 'idle';
    actionMessage.value = `工作流启动失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

async function confirmExecution() {
  if (!activeRunId.value) {
    await runSimulation();
    return;
  }
  if (!activeConfirmationId.value && !executionConfirmed.value) {
    actionMessage.value = '当前工作流没有可用确认 ID，请重新模拟执行。';
    return;
  }
  try {
    const run = await apiClient.agents.confirmWorkflowRun(activeRunId.value, {
      confirmationId: activeConfirmationId.value,
    });
    executionConfirmed.value = true;
    simulationState.value = 'done';
    actionMessage.value = run.message ?? '工作流已确认并完成。';
  } catch (error) {
    actionMessage.value = `确认失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

function mapPreviewNodes(nodes: unknown): LocalWorkflowNode[] {
  if (!Array.isArray(nodes) || nodes.length === 0) return seedWorkflowNodes;
  const icons = [FileSearch, Sparkles, CircleAlert, ArchiveRestore];
  return nodes.map((node, index) => {
    const row = node as WorkflowNode;
    return {
      id: row.id,
      label: row.label,
      value: row.value,
      icon: icons[index] ?? ShieldCheck,
    };
  });
}

function confirmationFromMessage(message?: string) {
  return message?.match(/confirmationId=([A-Za-z0-9-]+)/)?.[1] ?? '';
}

onMounted(loadAgentWorkbench);
</script>

<template>
  <div class="agent-workbench">
    <section class="agent-workbench__templates" aria-label="Agent 模板">
      <header>
        <h3>模板</h3>
        <span>{{ loading ? '同步中' : `${agentTemplates.length} 个可用` }}</span>
      </header>
      <button
        v-for="(template, index) in agentTemplates"
        :key="template.id ?? template.name"
        class="agent-workbench__template"
        :class="{ 'agent-workbench__template--active': index === selectedTemplateIndex }"
        type="button"
        @click="selectTemplate(index)"
      >
        <strong>{{ template.name }}</strong>
        <span>{{ template.desc }}</span>
        <small>{{ template.risk }}</small>
      </button>
    </section>

    <section class="agent-workbench__workflow" aria-label="Workflow nodes">
      <header>
        <h3>Workflow nodes</h3>
        <button type="button" :disabled="simulationState === 'running' && !activeConfirmationId" @click="runSimulation">
          <Play :size="14" /> {{ simulationState === 'running' ? '重新模拟' : '模拟执行' }}
        </button>
      </header>
      <div class="agent-workbench__nodes">
        <article
          v-for="node in workflowNodes"
          :key="node.label"
          class="agent-workbench__node"
          :class="{ 'agent-workbench__node--running': simulationState !== 'idle' }"
        >
          <div class="agent-workbench__node-icon">
            <component :is="node.icon" :size="17" />
          </div>
          <div>
            <strong>{{ node.label }}</strong>
            <p>{{ node.value }}</p>
          </div>
        </article>
      </div>
    </section>

    <section class="agent-workbench__permissions" aria-label="工具权限">
      <header>
        <h3><KeyRound :size="15" /> 工具权限</h3>
        <span>{{ selectedTemplate?.risk }}</span>
      </header>
      <div class="agent-workbench__tools">
        <span v-for="tool in visibleTools" :key="tool">
          <ShieldCheck :size="13" />
          {{ tool }}
        </span>
      </div>
    </section>

    <section class="agent-workbench__confirm" aria-label="执行确认">
      <div>
        <strong>{{ executionConfirmed ? '执行已确认' : '等待执行确认' }}</strong>
        <p>{{ actionMessage || `${selectedTemplate?.name} 将按最小权限执行，写入审计日志；不删除原文件。` }}</p>
      </div>
      <button type="button" :disabled="simulationState === 'idle' && !activeRunId" @click="confirmExecution">
        <CheckCircle2 :size="15" />
        {{ executionConfirmed ? '已确认' : '确认执行' }}
      </button>
    </section>
  </div>
</template>

<style scoped>
.agent-workbench {
  display: grid;
  grid-template-columns: 210px minmax(0, 1fr) 180px;
  grid-template-rows: minmax(0, 1fr) auto;
  gap: 12px;
  height: 100%;
  min-height: 0;
}

.agent-workbench__templates,
.agent-workbench__workflow,
.agent-workbench__permissions,
.agent-workbench__confirm {
  min-height: 0;
  background: rgba(255, 255, 255, 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.agent-workbench header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
  padding: 11px 12px;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
}

.agent-workbench h3 {
  margin: 0;
  color: var(--text-strong);
  font-size: 12px;
}

.agent-workbench header span {
  color: var(--text-soft);
  font-size: 11px;
}

.agent-workbench__templates {
  display: grid;
  grid-template-rows: auto repeat(3, minmax(0, 1fr));
  overflow: hidden;
}

.agent-workbench__template {
  display: grid;
  gap: 4px;
  align-content: center;
  padding: 10px 12px;
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.12);
}

.agent-workbench__template--active {
  background: rgba(19, 136, 255, 0.08);
  box-shadow: inset 3px 0 0 var(--accent);
}

.agent-workbench__template strong {
  color: var(--text-strong);
  font-size: 12px;
}

.agent-workbench__template span {
  display: -webkit-box;
  overflow: hidden;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.32;
  -webkit-line-clamp: 2;
  -webkit-box-orient: vertical;
}

.agent-workbench__template small {
  color: #b36a00;
  font-size: 10px;
  font-weight: 760;
}

.agent-workbench__workflow {
  overflow: hidden;
}

.agent-workbench__workflow header button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: var(--radius-sm);
  font-size: 11px;
  font-weight: 760;
}

.agent-workbench__nodes {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 10px;
  padding: 12px;
}

.agent-workbench__node {
  display: flex;
  gap: 10px;
  min-height: 78px;
  padding: 12px;
  background: rgba(255, 255, 255, 0.58);
  border: 1px solid rgba(100, 136, 166, 0.14);
  border-radius: var(--radius-sm);
}

.agent-workbench__node--running {
  border-color: rgba(19, 136, 255, 0.2);
  background: rgba(231, 247, 255, 0.72);
}

.agent-workbench__node-icon {
  display: grid;
  width: 34px;
  height: 34px;
  flex: 0 0 34px;
  place-items: center;
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
  border-radius: var(--radius-sm);
}

.agent-workbench__node strong {
  color: var(--text-strong);
  font-size: 12px;
}

.agent-workbench__node p {
  margin: 5px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.agent-workbench__permissions {
  display: grid;
  grid-template-rows: auto 1fr;
}

.agent-workbench__permissions h3 {
  display: flex;
  align-items: center;
  gap: 6px;
}

.agent-workbench__tools {
  display: flex;
  flex-wrap: wrap;
  align-content: start;
  gap: 8px;
  padding: 12px;
}

.agent-workbench__tools span {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  height: 28px;
  padding: 0 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.7);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 650;
}

.agent-workbench__confirm {
  grid-column: 2 / span 2;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 12px 14px;
  background: linear-gradient(135deg, rgba(255, 246, 227, 0.82), rgba(231, 247, 255, 0.78));
}

.agent-workbench__confirm strong {
  color: var(--text-strong);
  font-size: 13px;
}

.agent-workbench__confirm p {
  margin: 4px 0 0;
  color: var(--text-muted);
  font-size: 11px;
}

.agent-workbench__confirm button {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  height: 32px;
  padding: 0 12px;
  color: #fff;
  white-space: nowrap;
  background: var(--accent);
  border: 0;
  border-radius: var(--radius-sm);
  font-size: 12px;
  font-weight: 760;
}
</style>
