<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Bot, Sparkles, Wrench } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { AgentPreset, ToolCatalogEntry } from '../../api/types';
import { createAssistantSession } from '../../stores/assistant';
import { UiBadge, UiButton, UiEmptyState, UiSpinner, UiWindowPage, UiNavRail } from '../ui';
import MessageBubble from '../ai/MessageBubble.vue';
import Composer from '../ai/Composer.vue';

// The workbench runs its own agent session, independent of the AI Assistant
// window's shared singleton.
const session = createAssistantSession();
const { messages, sending, activeId, modelLabel } = session;

const presets = ref<AgentPreset[]>([]);
const catalog = ref<ToolCatalogEntry[]>([]);
const activePresetId = ref('');
const loading = ref(true);
const notice = ref('');
const scrollEl = ref<HTMLElement | null>(null);

const domainLabels: Record<string, string> = {
  storage: '存储',
  monitoring: '监控',
  files: '文件',
  search: '检索',
  steward: '文件管家',
  docker: 'Docker',
  downloads: '下载',
  media: '相册',
  security: '安全',
  backups: '备份',
  remote: '远程',
  music: '音乐',
  video: '视频',
};

const activePreset = computed(() => presets.value.find((p) => p.id === activePresetId.value) ?? null);

const isEmpty = computed(() => messages.value.length === 0);

// Tool counts per domain for the active preset's capability panel.
const capabilities = computed(() => {
  const preset = activePreset.value;
  if (!preset) return [];
  return preset.toolDomains.map((domain) => {
    const tools = catalog.value.filter((t) => t.domain === domain);
    return {
      domain,
      label: domainLabels[domain] ?? domain,
      total: tools.length,
      writable: tools.filter((t) => !t.readOnly).length,
    };
  });
});

function scrollToBottom() {
  requestAnimationFrame(() => {
    const el = scrollEl.value;
    if (el) el.scrollTop = el.scrollHeight;
  });
}

async function selectPreset(preset: AgentPreset) {
  if (sending.value) return;
  activePresetId.value = preset.id;
  try {
    await session.startPreset(preset.id, preset.name);
    notice.value = '';
  } catch (error) {
    notice.value = `启动失败：${error instanceof Error ? error.message : 'unknown error'}`;
  }
}

function handleSend(text: string) {
  session.send(text).then(scrollToBottom);
  scrollToBottom();
}

async function confirmAction(id: string) {
  try {
    await apiClient.assistant.confirmAction(id, {});
  } finally {
    if (activeId.value) await session.switchThread(activeId.value);
  }
}

async function cancelAction(id: string) {
  try {
    await apiClient.assistant.cancelAction(id, {});
  } finally {
    if (activeId.value) await session.switchThread(activeId.value);
  }
}

onMounted(async () => {
  loading.value = true;
  try {
    const [p, c] = await Promise.all([apiClient.assistant.getPresets(), apiClient.assistant.getToolCatalog()]);
    presets.value = p;
    catalog.value = c;
    if (p.length) await selectPreset(p[0]);
  } catch (error) {
    notice.value = `后端暂不可用：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
});
</script>

<template>
  <UiWindowPage
    layout="chat"
    :icon="Sparkles"
    :title="activePreset?.name ?? 'Agent 工作台'"
    :subtitle="modelLabel || '加载中…'"
  >
    <template #actions>
      <UiBadge variant="dot" tone="primary">{{ modelLabel || '加载中…' }}</UiBadge>
    </template>

    <!-- Left: presets -->
    <template #nav>
      <UiNavRail title="Agent 预设">
        <UiSpinner v-if="loading && !presets.length" size="md" />
        <button
          v-for="preset in presets"
          :key="preset.id"
          type="button"
          class="preset"
          :class="{ 'is-active': preset.id === activePresetId }"
          :disabled="sending"
          @click="selectPreset(preset)"
        >
          <strong>{{ preset.name }}</strong>
          <span>{{ preset.description }}</span>
        </button>
      </UiNavRail>
    </template>

    <!-- Center: live runner -->
    <div ref="scrollEl" class="chat-scroll">
      <UiEmptyState
        v-if="isEmpty"
        :icon="Bot"
        :title="activePreset ? `与「${activePreset.name}」对话` : '选择一个 Agent 预设'"
        :description="activePreset?.description"
      >
        <div v-if="activePreset" class="starters">
          <UiButton
            v-for="s in activePreset.starters"
            :key="s"
            size="sm"
            variant="soft"
            tone="neutral"
            :disabled="sending"
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

    <!-- Composer -->
    <template #composer>
      <p v-if="notice" class="chat-notice">{{ notice }}</p>
      <Composer :sending="sending" @send="handleSend" @stop="session.stop()" />
    </template>

    <!-- Right: capabilities -->
    <template #inspector>
      <h3 class="rail-title"><Wrench :size="15" /> 可用能力</h3>
      <p class="caps-hint">该 Agent 仅能调用以下工具域（写操作需确认）。</p>
      <div v-for="cap in capabilities" :key="cap.domain" class="cap">
        <div class="cap__head">
          <strong>{{ cap.label }}</strong>
          <span>{{ cap.total }} 个工具</span>
        </div>
        <div class="cap__bar">
          <UiBadge size="sm" tone="info" variant="soft">{{ cap.total - cap.writable }} 只读</UiBadge>
          <UiBadge v-if="cap.writable" size="sm" tone="warning" variant="soft">{{ cap.writable }} 写操作</UiBadge>
        </div>
      </div>
      <div class="caps-foot">
        当前模型：{{ modelLabel || '未绑定' }}<br />
        在「设置 → 模型策略」中管理。
      </div>
    </template>
  </UiWindowPage>
</template>

<style scoped>
.rail-title {
  display: inline-flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0 0 var(--space-2);
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.preset {
  display: flex;
  flex-direction: column;
  gap: 2px;
  padding: var(--space-2) var(--space-3);
  border: 1px solid transparent;
  border-radius: var(--radius-card);
  background: transparent;
  text-align: left;
  cursor: pointer;
  transition: background var(--duration-fast) var(--ease-standard),
    border-color var(--duration-fast) var(--ease-standard);
}
.preset:hover {
  background: var(--accent-soft);
}
.preset.is-active {
  border-color: var(--border-strong);
  background: var(--accent-soft);
}
.preset strong {
  font-size: var(--fs-sm);
  color: var(--text-strong);
}
.preset span {
  font-size: var(--fs-2xs);
  color: var(--text-muted);
  line-height: var(--lh-snug);
}

.chat-scroll {
  flex: 1;
  min-height: 0;
  overflow-y: auto;
}
.starters {
  display: flex;
  flex-wrap: wrap;
  gap: var(--space-2);
  justify-content: center;
  margin-top: var(--space-3);
}
.chat-notice {
  margin: 0 0 var(--space-2);
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}

.caps-hint {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-normal);
}
.cap {
  padding: var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  background: var(--surface-glass);
}
.cap__head {
  display: flex;
  align-items: baseline;
  justify-content: space-between;
}
.cap__head strong {
  font-size: var(--fs-sm);
  color: var(--text-strong);
}
.cap__head span {
  font-size: var(--fs-2xs);
  color: var(--text-soft);
}
.cap__bar {
  display: flex;
  gap: var(--space-1);
  margin-top: var(--space-1);
}
.caps-foot {
  margin-top: auto;
  padding-top: var(--space-3);
  color: var(--text-soft);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
}
</style>
