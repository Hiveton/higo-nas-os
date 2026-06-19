import { readonly, ref } from 'vue';
import { apiClient } from '../api/client';
import type { ChatToolEvent } from '../api/runtime';
import type { AssistantMessage, AssistantToolTrace, ThreadSummary } from '../api/types';

// ChatStep is a tool-call shown in the analysis view. `running` is true between
// the start and done events of one call.
export type ChatStep = {
  name: string;
  summary?: string;
  error?: string;
  running: boolean;
  confirm?: boolean;
};

// ChatMessage is the UI projection of an assistant/user turn. `steps` holds live
// tool activity for the in-flight turn; persisted history uses `tools` instead.
export type ChatMessage = AssistantMessage & {
  streaming?: boolean;
  steps?: ChatStep[];
};

// createAssistantSession builds one independent agent conversation session. The
// AI Assistant window uses the shared singleton (`assistantStore`); the Agent
// Workbench creates its own instance so the two run side by side without sharing
// active thread / messages.
export function createAssistantSession() {
  const threads = ref<ThreadSummary[]>([]);
  const activeId = ref<string>('');
  const messages = ref<ChatMessage[]>([]);
  const sending = ref(false);
  const loading = ref(false);
  const notice = ref('');
  const modelLabel = ref('');

  let controller: AbortController | null = null;
  let initialized = false;

  function errMessage(reason: unknown): string {
    return reason instanceof Error ? reason.message : 'unknown error';
  }

  async function loadModelLabel() {
    try {
      const providers = await apiClient.ai.listProviders();
      const def = providers.find((p) => p.isDefault) ?? providers.find((p) => p.enabled);
      modelLabel.value = def ? `${def.name} · ${def.model}` : '未绑定模型';
    } catch {
      modelLabel.value = '未绑定模型';
    }
  }

  async function refreshThreads() {
    threads.value = await apiClient.assistant.listThreads();
  }

  async function init() {
    if (initialized) return;
    initialized = true;
    loading.value = true;
    try {
      await Promise.all([refreshThreads(), loadModelLabel()]);
      if (threads.value.length === 0) {
        await newThread();
      } else {
        await switchThread(threads.value[0].id);
      }
      notice.value = '';
    } catch (reason) {
      initialized = false; // allow a later open to retry
      notice.value = `后端暂不可用：${errMessage(reason)}`;
    } finally {
      loading.value = false;
    }
  }

  // ensureReady loads the model label without forcing a default thread (used by
  // the workbench, which creates a preset thread itself).
  async function ensureReady() {
    if (initialized) return;
    initialized = true;
    await loadModelLabel();
  }

  // ask opens a fresh-or-current thread and sends a question.
  async function ask(text: string) {
    const trimmed = text.trim();
    if (!trimmed) return;
    await init();
    await send(trimmed);
  }

  // startPreset opens a new thread bound to an agent preset and makes it active.
  async function startPreset(presetId: string, title?: string) {
    await ensureReady();
    const thread = await apiClient.assistant.createThread({ presetId, title });
    await refreshThreads();
    activeId.value = thread.id;
    messages.value = (thread.messages ?? []).map((m) => ({ ...m }));
    return thread;
  }

  async function switchThread(id: string) {
    if (!id) return;
    activeId.value = id;
    try {
      const thread = await apiClient.assistant.getThread(id);
      messages.value = (thread.messages ?? []).map((m) => ({ ...m }));
    } catch (reason) {
      notice.value = `加载会话失败：${errMessage(reason)}`;
      messages.value = [];
    }
  }

  async function newThread() {
    const thread = await apiClient.assistant.createThread();
    await refreshThreads();
    activeId.value = thread.id;
    messages.value = (thread.messages ?? []).map((m) => ({ ...m }));
  }

  async function deleteThread(id: string) {
    await apiClient.assistant.deleteThread(id);
    await refreshThreads();
    if (activeId.value === id) {
      if (threads.value.length > 0) {
        await switchThread(threads.value[0].id);
      } else {
        await newThread();
      }
    }
  }

  function applyToolEvent(reply: ChatMessage, evt: ChatToolEvent) {
    if (!reply.steps) reply.steps = [];
    if (evt.phase === 'start') {
      reply.steps.push({ name: evt.name, running: true });
      return;
    }
    if (evt.phase === 'confirm') {
      // One confirmation card per pending action — ignore repeats for the same tool.
      if (reply.steps.some((s) => s.confirm && s.name === evt.name)) return;
      reply.steps.push({ name: evt.name, summary: evt.summary, running: false, confirm: true });
      return;
    }
    // done: update the last running step with this name.
    for (let i = reply.steps.length - 1; i >= 0; i -= 1) {
      if (reply.steps[i].name === evt.name && reply.steps[i].running) {
        reply.steps[i] = { name: evt.name, summary: evt.summary, error: evt.error, running: false };
        return;
      }
    }
    reply.steps.push({ name: evt.name, summary: evt.summary, error: evt.error, running: false });
  }

  async function send(text: string) {
    const trimmed = text.trim();
    if (!trimmed || sending.value) return;
    if (!activeId.value) await newThread();

    messages.value.push({ id: `local-${Date.now()}`, role: 'user', text: trimmed });
    messages.value.push({ id: `stream-${Date.now()}`, role: 'assistant', text: '', steps: [], streaming: true });
    const reply = messages.value[messages.value.length - 1];

    sending.value = true;
    controller = new AbortController();
    try {
      await apiClient.assistant.streamMessage(
        activeId.value,
        { role: 'user', text: trimmed },
        {
          signal: controller.signal,
          onDelta: (chunk) => {
            reply.text += chunk;
          },
          onTool: (evt) => applyToolEvent(reply, evt),
          onDone: ({ message, error }) => {
            reply.streaming = false;
            reply.steps?.forEach((s) => (s.running = false));
            if (error) {
              notice.value = `生成失败：${error}`;
              if (!reply.text) reply.text = `生成失败：${error}`;
              return;
            }
            const final = message as AssistantMessage | undefined;
            if (final) {
              if (final.id) reply.id = final.id;
              if (final.text) reply.text = final.text;
              reply.pendingActionId = final.pendingActionId;
              reply.citations = final.citations;
              if (final.tools && final.tools.length) reply.tools = final.tools as AssistantToolTrace[];
            }
            notice.value = '';
          },
        },
      );
    } catch (reason) {
      reply.streaming = false;
      if (errMessage(reason).toLowerCase().includes('abort')) {
        notice.value = '已停止生成。';
      } else {
        if (!reply.text) reply.text = `发送失败：${errMessage(reason)}`;
        notice.value = `发送失败：${errMessage(reason)}`;
      }
    } finally {
      sending.value = false;
      controller = null;
      // Keep the session list ordering/counts fresh.
      refreshThreads().catch(() => undefined);
    }
  }

  function stop() {
    controller?.abort();
  }

  return {
    threads: readonly(threads),
    activeId: readonly(activeId),
    messages,
    sending: readonly(sending),
    loading: readonly(loading),
    notice: readonly(notice),
    modelLabel: readonly(modelLabel),
    init,
    refreshThreads,
    newThread,
    switchThread,
    deleteThread,
    send,
    ask,
    startPreset,
    stop,
  };
}

export type AssistantSession = ReturnType<typeof createAssistantSession>;

// The shared singleton used by the AI Assistant window.
export const assistantStore = createAssistantSession();
