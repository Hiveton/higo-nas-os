<script setup lang="ts">
import { computed } from 'vue';
import { CheckCircle2, CircleAlert, ShieldQuestion, Wrench } from 'lucide-vue-next';
import { UiSpinner } from '../ui';
import type { ChatStep } from '../../stores/assistant';

const props = defineProps<{ step: ChatStep }>();

// Map a domain segment of the MCP tool name to a Chinese label.
const domainLabels: Record<string, string> = {
  system: '系统',
  storage: '存储',
  docker: 'Docker',
  monitoring: '监控',
  downloads: '下载',
  files: '文件',
  media: '相册',
  music: '音乐',
  video: '视频',
  security: '安全',
  accounts: '账号',
  backups: '备份',
  remote: '远程',
  agents: 'Agent',
  steward: '管家',
  settings: '设置',
  desktop: '桌面',
  appcenter: '应用中心',
};

// MCP tool names look like "higo_storage_pools_list" — turn them into a readable
// label like "存储 · pools list".
const label = computed(() => {
  const parts = props.step.name.replace(/^higo_/, '').split('_');
  if (parts.length === 0) return props.step.name;
  const domain = domainLabels[parts[0]] ?? parts[0];
  const rest = parts.slice(1).join(' ');
  return rest ? `${domain} · ${rest}` : domain;
});
</script>

<template>
  <div class="tool-step" :class="{ 'is-error': step.error, 'is-confirm': step.confirm }">
    <UiSpinner v-if="step.running" size="sm" />
    <CircleAlert v-else-if="step.error" :size="15" class="icon-error" />
    <ShieldQuestion v-else-if="step.confirm" :size="15" class="icon-confirm" />
    <CheckCircle2 v-else :size="15" class="icon-done" />
    <span class="tool-step__label">
      <Wrench :size="12" />
      {{ label }}
    </span>
    <span class="tool-step__summary">
      {{ step.running ? '运行中…' : step.error ? step.error : step.summary }}
    </span>
  </div>
</template>

<style scoped>
.tool-step {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--border);
  border-radius: var(--radius-control);
  background: var(--surface-glass);
  font-size: var(--fs-2xs);
}
.tool-step.is-error {
  border-color: var(--accent-red);
}
.tool-step.is-confirm {
  border-color: var(--accent-orange);
  background: var(--accent-orange-soft, var(--surface-glass));
}
.icon-confirm {
  color: var(--ink-orange);
}
.tool-step__label {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
  white-space: nowrap;
}
.tool-step__summary {
  color: var(--text-muted);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.icon-done {
  color: var(--ink-green);
}
.icon-error {
  color: var(--accent-red);
}
</style>
