<script setup lang="ts">
import { computed } from 'vue';
import { Network, Plug, Trash2 } from 'lucide-vue-next';
import type { DockerContainer, DockerNetwork } from '../../../api/types';
import { UiButton, UiCheckbox, UiInput, UiSelect } from '../../ui';

type NetworkForm = { name: string; driver: string; subnet: string; gateway: string; attachable: boolean; internal: boolean };
type NetworkAttachForm = { container: string; alias: string };

const props = defineProps<{
  networks: DockerNetwork[];
  containers: DockerContainer[];
  networkForm: NetworkForm;
  networkAttachForm: NetworkAttachForm;
  selectedNetworkName: string;
  selectedNetwork: DockerNetwork | undefined;
}>();

const emit = defineEmits<{
  (e: 'create'): void;
  (e: 'select', name: string): void;
  (e: 'remove', name: string): void;
  (e: 'connect'): void;
  (e: 'disconnect', container: string): void;
}>();

const containerOptions = computed(() => props.containers.map((container) => ({ label: container.name, value: container.name })));
</script>

<template>
  <section class="docker-panel docker-networks">
    <header class="docker-panel__header docker-form-row">
      <UiInput v-model="networkForm.name" placeholder="网络名称" />
      <UiInput v-model="networkForm.driver" placeholder="驱动，例如 bridge" />
      <UiInput v-model="networkForm.subnet" placeholder="子网，可选" />
      <UiInput v-model="networkForm.gateway" placeholder="网关，可选" />
      <UiCheckbox v-model="networkForm.attachable" label="可连接" />
      <UiCheckbox v-model="networkForm.internal" label="内部网络" />
      <UiButton variant="solid" tone="primary" size="sm" :icon-left="Network" :disabled="!networkForm.name.trim()" @click="emit('create')">创建</UiButton>
    </header>

    <div class="docker-split docker-split--wide-left">
      <section class="docker-card">
        <header><h3><Network :size="15" /> 网络列表</h3></header>
        <div class="docker-list">
          <button
            v-for="network in networks"
            :key="network.id"
            class="docker-list-item docker-list-button"
            :class="{ 'is-active': selectedNetworkName === network.name }"
            type="button"
            @click="emit('select', network.name)"
          >
            <div>
              <strong>{{ network.name }}</strong>
              <p>{{ network.driver }} · {{ network.scope }} · {{ network.subnet || '无子网' }}</p>
              <small>{{ network.containers?.length || 0 }} 个容器</small>
            </div>
            <span>{{ network.internal ? '内部' : '普通' }}</span>
          </button>
        </div>
      </section>

      <section class="docker-card">
        <header>
          <h3><Plug :size="15" /> 网络配置</h3>
          <UiButton v-if="selectedNetwork" variant="soft" tone="danger" size="sm" :icon-left="Trash2" @click="emit('remove', selectedNetwork.name)">删除</UiButton>
        </header>
        <div v-if="selectedNetwork" class="docker-network-detail">
          <p><strong>{{ selectedNetwork.name }}</strong></p>
          <p>驱动：{{ selectedNetwork.driver }}</p>
          <p>子网：{{ selectedNetwork.subnet || '-' }}</p>
          <p>网关：{{ selectedNetwork.gateway || '-' }}</p>
          <div class="docker-command__input">
            <UiSelect v-model="networkAttachForm.container" :options="containerOptions" />
            <UiInput v-model="networkAttachForm.alias" placeholder="别名，可选" />
            <UiButton variant="soft" size="sm" @click="emit('connect')">连接</UiButton>
          </div>
          <div class="docker-kv-list">
            <span v-for="container in selectedNetwork.containers" :key="container">
              {{ container }}
              <UiButton variant="soft" tone="danger" size="sm" @click="emit('disconnect', container)">断开</UiButton>
            </span>
            <span v-if="!selectedNetwork.containers?.length">暂无已连接容器。</span>
          </div>
        </div>
      </section>
    </div>
  </section>
</template>
