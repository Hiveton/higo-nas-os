<script setup lang="ts">
import { Boxes, RefreshCw } from 'lucide-vue-next';
import type { ComposeStack } from '../../../api/types';
import { UiButton, UiEmptyState } from '../../ui';

defineProps<{
  composeStacks: ComposeStack[];
  containers: { id: string }[];
  loading: boolean;
}>();

const emit = defineEmits<{
  (e: 'refresh'): void;
  (e: 'view-containers', name: string): void;
}>();
</script>

<template>
  <section class="docker-panel docker-compose-page">
    <header class="docker-panel__header">
      <div>
        <h3>Compose</h3>
        <p>{{ composeStacks.length }} 个栈，{{ containers.length }} 个容器。</p>
      </div>
      <UiButton variant="soft" size="sm" :icon-left="RefreshCw" :disabled="loading" @click="emit('refresh')">刷新</UiButton>
    </header>
    <div class="docker-list docker-list--grid">
      <article v-for="stack in composeStacks" :key="stack.name" class="docker-list-item">
        <div>
          <strong>{{ stack.name }}</strong>
          <p>{{ stack.status }} · {{ stack.services }} 个服务 · {{ stack.ports }}</p>
          <small>{{ stack.volume }} · {{ stack.network }}</small>
        </div>
        <UiButton variant="soft" size="sm" @click="emit('view-containers', stack.name)">查看容器</UiButton>
      </article>
      <UiEmptyState v-if="composeStacks.length === 0" compact :icon="Boxes" title="暂无 Compose 栈" />
    </div>
  </section>
</template>
