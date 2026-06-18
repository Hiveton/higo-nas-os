<script setup lang="ts">
import { computed, nextTick, onMounted, ref, watch } from 'vue';
import {
  BrainCircuit,
  MessageSquarePlus,
  PanelRightClose,
  PanelRightOpen,
  Sparkles,
  Trash2,
} from 'lucide-vue-next';
import { UiBadge, UiButton, UiEmptyState, UiIconButton } from '../ui';
import { UiConfirmDialog } from '../ui';
import MessageBubble from '../ai/MessageBubble.vue';
import Composer from '../ai/Composer.vue';
import ToolStep from '../ai/ToolStep.vue';
import { assistantStore } from '../../stores/assistant';
import { apiClient } from '../../api/client';

const { threads, activeId, messages, sending, loading, notice, modelLabel } = assistantStore;

const showAnalysis = ref(true);
const scrollEl = ref<HTMLElement | null>(null);
const pendingDelete = ref<string>('');

const suggestions = [
  '我的 NAS 还剩多少存储空间？',
  '硬盘健康状态怎么样？',
  '现在有哪些 Docker 容器在运行？',
  '帮我找上个月的合同文件',
];

const isEmpty = computed(() => messages.value.length === 0);

// The most recent assistant turn drives the analysis rail.
const lastAssistant = computed(() => {
  for (let i = messages.value.length - 1; i >= 0; i -= 1) {
    if (messages.value[i].role === 'assistant') return messages.value[i];
  }
  return null;
});
const analysisSteps = computed(() => {
  const m = lastAssistant.value;
  if (!m) return [];
  if (m.steps && m.steps.length) return m.steps;
  return (m.tools ?? []).map((t) => ({ name: t.name, summary: t.summary, running: false }));
});
const analysisCitations = computed(() => lastAssistant.value?.citations ?? []);

function scrollToBottom() {
  nextTick(() => {
    const el = scrollEl.value;
    if (el) el.scrollTop = el.scrollHeight;
  });
}

watch(
  () => messages.value.map((m) => m.text).join('|').length + messages.value.length,
  scrollToBottom,
);

function handleSend(text: string) {
  assistantStore.send(text);
}

async function confirmAction(id: string) {
  try {
    await apiClient.assistant.confirmAction(id, {});
  } finally {
    if (activeId.value) await assistantStore.switchThread(activeId.value);
  }
}

async function cancelAction(id: string) {
  try {
    await apiClient.assistant.cancelAction(id, {});
  } finally {
    if (activeId.value) await assistantStore.switchThread(activeId.value);
  }
}

async function doDelete() {
  const id = pendingDelete.value;
  pendingDelete.value = '';
  if (id) await assistantStore.deleteThread(id);
}

onMounted(() => assistantStore.init());
</script>

<template>
  <!-- shell holds the container context so @container can restyle .ai-window itself -->
  <div class="ai-window-shell">
    <div class="ai-window">
    <!-- Left: sessions -->
    <aside class="ai-window__sessions">
      <UiButton block :icon-left="MessageSquarePlus" :disabled="loading" @click="assistantStore.newThread()">
        新建对话
      </UiButton>
      <div class="session-list">
        <button
          v-for="t in threads"
          :key="t.id"
          type="button"
          class="session-item"
          :class="{ 'is-active': t.id === activeId }"
          @click="assistantStore.switchThread(t.id)"
        >
          <span class="session-item__title">{{ t.title || '新对话' }}</span>
          <span class="session-item__meta">{{ t.messageCount }} 条</span>
          <span class="session-item__del" @click.stop="pendingDelete = t.id">
            <Trash2 :size="14" />
          </span>
        </button>
      </div>
    </aside>

    <!-- Center: conversation -->
    <section class="ai-window__chat">
      <header class="chat-head">
        <div class="chat-head__title">
          <Sparkles :size="16" />
          <strong>HiGo AI 助手</strong>
        </div>
        <div class="chat-head__right">
          <UiBadge variant="dot" tone="primary">{{ modelLabel || '加载中…' }}</UiBadge>
          <UiIconButton
            :icon="showAnalysis ? PanelRightClose : PanelRightOpen"
            :label="showAnalysis ? '隐藏分析' : '显示分析'"
            @click="showAnalysis = !showAnalysis"
          />
        </div>
      </header>

      <div ref="scrollEl" class="chat-scroll">
        <UiEmptyState
          v-if="isEmpty"
          :icon="BrainCircuit"
          title="开始与 HiGo AI 对话"
          description="它能查询真实的存储、磁盘、设备与文件状态来回答你。"
        >
          <div class="suggestions">
            <UiButton
              v-for="s in suggestions"
              :key="s"
              size="sm"
              variant="soft"
              tone="neutral"
              @click="handleSend(s)"
            >
              {{ s }}
            </UiButton>
          </div>
        </UiEmptyState>

        <template v-else>
          <MessageBubble
            v-for="(m, i) in messages"
            :key="m.id ?? i"
            :message="m"
            @confirm-action="confirmAction"
            @cancel-action="cancelAction"
          />
        </template>
      </div>

      <footer class="chat-foot">
        <p v-if="notice" class="chat-notice">{{ notice }}</p>
        <Composer :sending="sending" @send="handleSend" @stop="assistantStore.stop()" />
      </footer>
    </section>

    <!-- Right: analysis -->
    <aside v-if="showAnalysis" class="ai-window__analysis">
      <h3 class="analysis-title">
        <BrainCircuit :size="15" />
        分析过程
      </h3>
      <div v-if="analysisSteps.length" class="analysis-steps">
        <ToolStep v-for="(step, i) in analysisSteps" :key="`${step.name}-${i}`" :step="step" />
      </div>
      <p v-else class="analysis-empty">本轮回答会在这里显示助手调用的工具与数据来源。</p>

      <template v-if="analysisCitations.length">
        <h3 class="analysis-title">引用</h3>
        <a
          v-for="(c, i) in analysisCitations"
          :key="i"
          class="analysis-cite"
          :href="c.url || undefined"
          :target="c.url ? '_blank' : undefined"
          rel="noopener noreferrer"
        >{{ c.title }}</a>
      </template>

      <div class="analysis-foot">
        当前模型：{{ modelLabel || '未绑定' }}<br />
        在「设置 → 模型策略」中管理。
      </div>
    </aside>

    <UiConfirmDialog
      :open="!!pendingDelete"
      title="删除对话"
      message="删除后无法恢复，确定删除这个对话吗？"
      confirm-label="删除"
      tone="danger"
      @confirm="doDelete"
      @update:open="(v) => { if (!v) pendingDelete = ''; }"
    />
    </div>
  </div>
</template>

<style scoped>
/* The shell is the query container; .ai-window is its child so @container can
   restyle the grid itself (a container can't size-query its own element). */
.ai-window-shell {
  height: 100%;
  min-height: 0;
  container-type: inline-size;
  container-name: aiwin;
}
.ai-window {
  display: grid;
  grid-template-columns: 220px minmax(0, 1fr) 260px;
  height: 100%;
  min-height: 0;
}
.ai-window__sessions {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3);
  border-right: 1px solid var(--border);
  overflow: hidden;
}
.session-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  overflow-y: auto;
}
.session-item {
  display: grid;
  grid-template-columns: 1fr auto auto;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2) var(--space-3);
  border: 1px solid transparent;
  border-radius: var(--radius-md);
  background: transparent;
  color: var(--text);
  cursor: pointer;
  text-align: left;
}
.session-item:hover {
  background: var(--surface-glass);
}
.session-item.is-active {
  border-color: var(--border-strong);
  background: var(--accent-soft, var(--surface-glass));
}
.session-item__title {
  overflow: hidden;
  font-size: var(--fs-sm);
  font-weight: var(--fw-medium);
  text-overflow: ellipsis;
  white-space: nowrap;
}
.session-item__meta {
  color: var(--text-soft);
  font-size: var(--fs-2xs);
}
.session-item__del {
  display: inline-flex;
  color: var(--text-soft);
}
.session-item__del:hover {
  color: var(--accent-red);
}

.ai-window__chat {
  display: flex;
  flex-direction: column;
  min-width: 0;
  min-height: 0;
}
.chat-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: var(--space-3) var(--space-4);
  border-bottom: 1px solid var(--border);
}
.chat-head__title {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-strong);
}
.chat-head__right {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
}
.chat-scroll {
  flex: 1;
  min-height: 0;
  padding: var(--space-4);
  overflow-y: auto;
}
.suggestions {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  justify-content: center;
  margin-top: var(--space-3);
}
.chat-foot {
  padding: var(--space-3) var(--space-4);
  border-top: 1px solid var(--border);
}
.chat-notice {
  margin: 0 0 var(--space-2);
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}

.ai-window__analysis {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
  border-left: 1px solid var(--border);
  overflow-y: auto;
}
.analysis-title {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin: var(--space-2) 0 0;
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.analysis-steps {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
}
.analysis-empty {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-normal);
}
.analysis-cite {
  color: var(--accent);
  font-size: var(--fs-2xs);
  text-decoration: none;
}
.analysis-foot {
  margin-top: auto;
  padding-top: var(--space-3);
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
}

/* Hide the analysis rail first, then the session rail, as the window narrows,
   so the conversation column is never squeezed to zero. */
@container aiwin (max-width: 1040px) {
  .ai-window {
    grid-template-columns: 200px minmax(0, 1fr);
  }
  .ai-window__analysis {
    display: none;
  }
}
@container aiwin (max-width: 720px) {
  .ai-window {
    grid-template-columns: minmax(0, 1fr);
  }
  .ai-window__sessions {
    display: none;
  }
}
</style>
