<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import type { Component } from 'vue';
import {
  Bot,
  BrainCircuit,
  ChevronRight,
  Cloud,
  DatabaseZap,
  FileSearch,
  LockKeyhole,
  MessageSquareText,
  PanelRight,
  Send,
  ShieldCheck,
  Sparkles,
  WandSparkles,
} from 'lucide-vue-next';
import { apiClient } from '../api/client';
import type { AgentTemplate, AssistantMessage, AssistantThread } from '../api/types';
import { agentTemplates, assistantMessages as seedMessages } from '../data/higoos';

type Capability = {
  label: string;
  detail: string;
  icon: Component;
};

const capabilities: Capability[] = [
  { label: '文件语义搜索', detail: '跨家庭空间、团队空间和相册理解内容', icon: FileSearch },
  { label: '设备运维', detail: '读取硬盘、备份、Docker 与网络状态', icon: DatabaseZap },
  { label: '权限审计', detail: '发现公开链接、敏感文件和高风险操作', icon: ShieldCheck },
  { label: '工作流编排', detail: '生成计划，关键步骤等待确认', icon: WandSparkles },
];

const modelPolicies = [
  { label: '隐私空间', value: '本地小模型', detail: '文件内容不离开设备' },
  { label: '复杂推理', value: '云端增强', detail: '仅上传脱敏摘要' },
  { label: '执行动作', value: '人工确认', detail: '移动、删除、分享均写入审计' },
];

const suggestedActions = [
  '生成下载目录整理计划',
  '收紧 3 个公开链接',
  '检查 Docker 镜像更新',
  '汇总今日备份报告',
];

const agentTemplateRows = ref<AgentTemplate[]>(agentTemplates);
const thread = ref<AssistantThread | null>(null);
const messages = ref<AssistantMessage[]>(seedMessages.map((message, index) => ({
  id: `seed-message-${index}`,
  role: message.role === 'user' ? 'user' : 'assistant',
  text: message.text,
})));
const draft = ref('询问文件、备份、设备或权限状态');
const loading = ref(false);
const sending = ref(false);
const assistantNotice = ref('正在使用本地对话缓存，连接后端后会同步线程和高风险动作。');
const pendingActionIds = computed(() => messages.value.map((message) => message.pendingActionId).filter(Boolean) as string[]);

async function loadAssistantState() {
  loading.value = true;
  try {
    const [createdThread, templates] = await Promise.all([
      apiClient.assistant.createThread(),
      apiClient.agents.getTemplates(),
    ]);
    thread.value = createdThread;
    messages.value = createdThread.messages;
    agentTemplateRows.value = templates;
    assistantNotice.value = 'AI 助手已连接后端，消息、引用和待确认动作会同步。';
  } catch (error) {
    assistantNotice.value = `后端暂不可用，继续使用本地对话缓存：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function sendMessage(text = draft.value) {
  const trimmed = text.trim();
  if (!trimmed) return;
  const userMessage: AssistantMessage = { id: `draft-${Date.now()}`, role: 'user', text: trimmed };
  messages.value.push(userMessage);
  draft.value = '';
  sending.value = true;
  try {
    const threadId = thread.value?.id ?? 'thread-current';
    const assistantMessage = await apiClient.assistant.sendMessage(threadId, {
      role: 'user',
      text: trimmed,
    });
    messages.value.push(assistantMessage);
    assistantNotice.value = assistantMessage.pendingActionId
      ? '已生成待确认动作，确认前不会执行高风险操作。'
      : '助手回复已同步。';
  } catch (error) {
    messages.value.push({
      id: `fallback-${Date.now()}`,
      role: 'assistant',
      text: `已根据当前权限生成「${trimmed}」的执行草案，高风险动作会等待你确认。`,
    });
    assistantNotice.value = `发送失败，已保留本地回复：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    sending.value = false;
  }
}

async function confirmPendingAction(id: string) {
  try {
    const result = await apiClient.assistant.confirmAction(id, { actorId: 'desktop-assistant' });
    assistantNotice.value = result.message ?? `动作 ${id} 已确认。`;
    if (thread.value?.id) {
      const nextThread = await apiClient.assistant.getThread(thread.value.id);
      thread.value = nextThread;
      messages.value = nextThread.messages;
    }
  } catch (error) {
    assistantNotice.value = `确认失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

onMounted(loadAssistantState);
</script>

<template>
  <aside class="ai-panel" aria-label="HiGoOS AI 助手侧栏">
    <header class="ai-panel__header">
      <div class="assistant-mark">
        <Bot :size="22" />
      </div>
      <div>
        <p>HiGo AI 助手</p>
        <h2>{{ loading ? '正在同步后端' : '常驻系统副驾' }}</h2>
      </div>
      <PanelRight class="panel-icon" :size="19" />
    </header>

    <section class="status-band" aria-label="模型策略">
      <div class="status-band__switch is-local">
        <span>本地</span>
        <strong>ON</strong>
      </div>
      <div class="status-band__switch is-cloud">
        <span>云端</span>
        <strong>按需</strong>
      </div>
    </section>

    <section class="conversation" aria-label="AI 对话">
      <div class="section-title">
        <MessageSquareText :size="17" />
        <h3>最近对话</h3>
      </div>

      <article
        v-for="(message, index) in messages"
        :key="message.id ?? `${message.role}-${index}`"
        :class="['message-bubble', message.role === 'user' ? 'is-user' : 'is-assistant']"
      >
        {{ message.text }}
        <button
          v-if="message.pendingActionId"
          type="button"
          class="message-action"
          @click="confirmPendingAction(message.pendingActionId)"
        >
          确认执行
        </button>
      </article>
    </section>

    <section class="capabilities" aria-label="能力标签">
      <div class="section-title">
        <Sparkles :size="17" />
        <h3>可调用能力</h3>
      </div>

      <div class="capability-grid">
        <article v-for="capability in capabilities" :key="capability.label" class="capability-card">
          <component :is="capability.icon" :size="17" />
          <div>
            <strong>{{ capability.label }}</strong>
            <span>{{ capability.detail }}</span>
          </div>
        </article>
      </div>
    </section>

    <section class="policy-card" aria-label="模型策略">
      <div class="section-title">
        <BrainCircuit :size="17" />
        <h3>模型策略</h3>
      </div>

      <article v-for="policy in modelPolicies" :key="policy.label" class="policy-row">
        <div>
          <span>{{ policy.label }}</span>
          <strong>{{ policy.value }}</strong>
        </div>
        <p>{{ policy.detail }}</p>
      </article>
    </section>

    <section class="agent-card" aria-label="推荐角色">
      <div class="section-title">
        <LockKeyhole :size="17" />
        <h3>推荐助手</h3>
      </div>

      <article v-for="template in agentTemplateRows.slice(0, 2)" :key="template.id ?? template.name" class="agent-row">
        <div>
          <strong>{{ template.name }}</strong>
          <span>{{ template.tools.join(' / ') }}</span>
        </div>
        <em>{{ template.risk }}</em>
      </article>
    </section>

    <section class="actions" aria-label="建议操作">
      <button v-if="pendingActionIds.length > 0" type="button" class="action-button action-button--confirm" @click="confirmPendingAction(pendingActionIds[0])">
        <span>确认待执行动作</span>
        <ChevronRight :size="16" />
      </button>
      <button
        v-for="action in suggestedActions"
        :key="action"
        type="button"
        class="action-button"
        @click="sendMessage(action)"
      >
        <span>{{ action }}</span>
        <ChevronRight :size="16" />
      </button>
    </section>

    <form class="composer" aria-label="输入给 AI 助手" @submit.prevent="sendMessage()">
      <Cloud :size="17" />
      <input v-model="draft" type="text" :placeholder="assistantNotice" aria-label="AI 助手输入框" />
      <button type="submit" :disabled="sending" aria-label="发送">
        <Send :size="16" />
      </button>
    </form>
  </aside>
</template>

<style scoped>
.ai-panel {
  position: absolute;
  top: 74px;
  right: 24px;
  z-index: 7;
  display: flex;
  width: 356px;
  max-height: calc(100vh - var(--dock-height) - 96px);
  flex-direction: column;
  gap: 12px;
  padding: 14px;
  overflow-y: auto;
  border: 1px solid rgba(255, 255, 255, 0.62);
  border-radius: var(--radius-lg);
  background:
    linear-gradient(155deg, rgba(255, 255, 255, 0.86), rgba(233, 248, 255, 0.68) 48%, rgba(247, 250, 255, 0.74)),
    var(--surface);
  box-shadow: var(--shadow-lg);
  backdrop-filter: blur(28px) saturate(160%);
  scrollbar-width: none;
}

.ai-panel > * {
  position: relative;
  z-index: 1;
}

.ai-panel::-webkit-scrollbar {
  display: none;
}

.ai-panel__header {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 11px;
}

.assistant-mark {
  display: grid;
  width: 42px;
  height: 42px;
  place-items: center;
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 12px;
  color: var(--accent);
  background: linear-gradient(145deg, rgba(231, 247, 255, 0.96), rgba(255, 255, 255, 0.72));
}

.ai-panel__header p,
.section-title h3,
.policy-row p {
  margin: 0;
}

.ai-panel__header p {
  color: var(--text-soft);
  font-size: 12px;
  font-weight: 800;
  letter-spacing: 0;
}

.ai-panel__header h2 {
  margin: 2px 0 0;
  color: var(--text-strong);
  font-size: 18px;
  line-height: 1.2;
}

.panel-icon {
  color: var(--text-soft);
}

.status-band {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 9px;
}

.status-band__switch {
  display: flex;
  min-width: 0;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 9px 10px;
  border: 1px solid rgba(91, 129, 160, 0.16);
  border-radius: var(--radius-sm);
  background: rgba(255, 255, 255, 0.54);
}

.status-band__switch span {
  color: var(--text-muted);
  font-size: 12px;
}

.status-band__switch strong {
  font-size: 12px;
  line-height: 1;
}

.is-local strong {
  color: var(--accent-green);
}

.is-cloud strong {
  color: var(--accent);
}

.conversation,
.capabilities,
.policy-card,
.agent-card {
  min-width: 0;
  padding: 12px;
  border: 1px solid rgba(90, 128, 160, 0.15);
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.46);
}

.conversation {
  display: grid;
  gap: 8px;
}

.section-title {
  display: flex;
  align-items: center;
  gap: 7px;
  margin-bottom: 10px;
  color: var(--accent);
}

.section-title h3 {
  color: var(--text-strong);
  font-size: 13px;
  line-height: 1.2;
}

.message-bubble {
  display: grid;
  gap: 7px;
  max-width: 92%;
  padding: 9px 10px;
  border-radius: 12px;
  color: var(--text);
  font-size: 12px;
  line-height: 1.48;
}

.message-action {
  justify-self: start;
  height: 26px;
  padding: 0 9px;
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 999px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  font-size: 11px;
  font-weight: 760;
}

.message-bubble.is-user {
  justify-self: end;
  color: #ffffff;
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
}

.message-bubble.is-assistant {
  justify-self: start;
  border: 1px solid rgba(90, 128, 160, 0.12);
  background: rgba(255, 255, 255, 0.74);
}

.capability-grid {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.capability-card {
  display: flex;
  min-width: 0;
  gap: 7px;
  padding: 9px;
  border-radius: var(--radius-sm);
  color: var(--accent);
  background: rgba(231, 247, 255, 0.62);
}

.capability-card strong,
.agent-row strong {
  display: block;
  color: var(--text-strong);
  font-size: 12px;
  line-height: 1.25;
}

.capability-card span,
.agent-row span,
.policy-row span,
.policy-row p {
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.policy-card,
.agent-card {
  display: grid;
  gap: 8px;
}

.policy-row,
.agent-row {
  display: grid;
  grid-template-columns: 102px 1fr;
  gap: 10px;
  align-items: center;
  padding-top: 8px;
  border-top: 1px solid rgba(90, 128, 160, 0.12);
}

.policy-row:first-of-type,
.agent-row:first-of-type {
  padding-top: 0;
  border-top: 0;
}

.policy-row strong {
  display: block;
  color: var(--text-strong);
  font-size: 12px;
  line-height: 1.25;
}

.agent-row {
  grid-template-columns: 1fr auto;
}

.agent-row em {
  padding: 5px 7px;
  border-radius: 999px;
  color: var(--accent-orange);
  background: rgba(245, 158, 11, 0.12);
  font-size: 11px;
  font-style: normal;
  font-weight: 800;
  white-space: nowrap;
}

.actions {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 8px;
}

.action-button {
  display: flex;
  min-width: 0;
  min-height: 38px;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 9px;
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: var(--radius-sm);
  color: var(--text);
  background: rgba(255, 255, 255, 0.58);
}

.action-button span {
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  line-height: 1.25;
  text-align: left;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.action-button svg {
  flex: 0 0 auto;
  color: var(--accent);
}

.action-button--confirm {
  border-color: rgba(245, 158, 11, 0.24);
  background: rgba(255, 246, 227, 0.72);
}

.composer {
  display: grid;
  grid-template-columns: auto 1fr auto;
  align-items: center;
  gap: 8px;
  margin-top: auto;
  padding: 8px 8px 8px 10px;
  border: 1px solid rgba(90, 128, 160, 0.18);
  border-radius: 12px;
  color: var(--accent);
  background: rgba(255, 255, 255, 0.72);
}

.composer input {
  min-height: 28px;
  min-width: 0;
  border: 0;
  outline: 0;
  color: var(--text-muted);
  background: transparent;
  font-size: 12px;
}

.composer button {
  display: grid;
  width: 30px;
  height: 30px;
  place-items: center;
  border: 0;
  border-radius: 9px;
  color: #ffffff;
  background: linear-gradient(135deg, var(--accent), var(--accent-cyan));
}

.composer button:disabled {
  opacity: 0.58;
}

@media (max-width: 1180px) {
  .ai-panel {
    width: 320px;
  }
}

@media (max-width: 980px), (max-height: 760px) {
  .ai-panel {
    top: 92px;
    right: 14px;
    z-index: 200;
    width: calc(100vw - 28px);
    height: auto;
    max-height: calc(100vh - var(--dock-height) - 120px);
    background: linear-gradient(160deg, rgba(255, 255, 255, 0.97), rgba(239, 249, 255, 0.96));
    box-shadow: 0 28px 90px rgba(10, 40, 70, 0.34);
  }

  .capability-grid,
  .actions {
    grid-template-columns: 1fr;
  }

  .policy-row,
  .agent-row {
    grid-template-columns: 1fr;
    gap: 4px;
  }
}
</style>
