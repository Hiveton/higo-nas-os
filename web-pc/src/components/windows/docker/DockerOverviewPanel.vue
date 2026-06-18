<script setup lang="ts">
import { Activity, Container, Gauge, HardDrive, Network, RefreshCw } from 'lucide-vue-next';
import type { DockerContainer, DockerImage, DockerNetwork, DockerVolume } from '../../../api/types';
import { UiButton } from '../../ui';

defineProps<{
  containers: DockerContainer[];
  images: DockerImage[];
  networks: DockerNetwork[];
  volumes: DockerVolume[];
  runningCount: number;
  totalCpu: number;
  totalMemory: number;
  overviewRows: { label: string; value: string }[];
  actionState: string;
  loading: boolean;
  formatMetric: (key: 'cpu' | 'memory' | 'network') => string;
  cpuPath: string;
  cpuAreaPath: string;
  memoryPath: string;
  memoryAreaPath: string;
  networkPath: string;
  networkAreaPath: string;
}>();

const emit = defineEmits<{ (e: 'refresh'): void }>();
</script>

<template>
  <div class="docker-overview-root">
    <section class="docker-summary" aria-label="Docker 概览">
      <article>
        <Container :size="18" />
        <span>容器</span>
        <strong>{{ runningCount }}/{{ containers.length }}</strong>
      </article>
      <article>
        <Gauge :size="18" />
        <span>CPU</span>
        <strong>{{ totalCpu }}%</strong>
      </article>
      <article>
        <Activity :size="18" />
        <span>内存</span>
        <strong>{{ totalMemory }}%</strong>
      </article>
      <article>
        <Network :size="18" />
        <span>网络</span>
        <strong>{{ networks.length }}</strong>
      </article>
      <article>
        <HardDrive :size="18" />
        <span>镜像</span>
        <strong>{{ images.length }}</strong>
      </article>
    </section>

    <section class="docker-panel docker-overview-page">
      <div class="docker-overview-grid">
        <article class="docker-hero-card">
          <div>
            <h2>{{ containers.length ? `${runningCount} 个容器运行` : '无容器运行' }}</h2>
            <p>{{ containers.length ? 'Docker 运行状态已同步。' : '当前 Docker 没有运行中的业务容器。' }}</p>
          </div>
          <ul>
            <li v-for="row in overviewRows" :key="row.label">
              <span>{{ row.label }}</span>
              <strong>{{ row.value }}</strong>
            </li>
          </ul>
        </article>
        <article class="docker-service-card">
          <header>
            <div>
              <h3>Docker 服务</h3>
              <p>{{ actionState }}</p>
            </div>
            <UiButton variant="ghost" tone="neutral" size="sm" :icon-left="RefreshCw" :disabled="loading" @click="emit('refresh')">刷新</UiButton>
          </header>
          <div class="docker-service-row">
            <span>本地镜像</span>
            <strong>{{ images.length }}</strong>
          </div>
          <div class="docker-service-row">
            <span>网络</span>
            <strong>{{ networks.length }}</strong>
          </div>
          <div class="docker-service-row">
            <span>存储卷</span>
            <strong>{{ volumes.length }}</strong>
          </div>
        </article>
      </div>
      <section class="docker-charts">
        <h3>信息监控</h3>
        <div class="docker-chart-grid">
          <article class="docker-chart docker-chart--cpu">
            <header><span>Docker CPU 使用率</span><strong>{{ formatMetric('cpu') }}</strong></header>
            <svg viewBox="0 0 300 80" preserveAspectRatio="none">
              <path class="docker-chart__area" :d="cpuAreaPath" />
              <path class="docker-chart__line" :d="cpuPath" />
            </svg>
          </article>
          <article class="docker-chart docker-chart--memory">
            <header><span>Docker 内存使用率</span><strong>{{ formatMetric('memory') }}</strong></header>
            <svg viewBox="0 0 300 80" preserveAspectRatio="none">
              <path class="docker-chart__area" :d="memoryAreaPath" />
              <path class="docker-chart__line" :d="memoryPath" />
            </svg>
          </article>
        </div>
        <article class="docker-chart-wide docker-chart docker-chart--network">
          <header><span>网络实时吞吐</span><strong>{{ formatMetric('network') }}</strong></header>
          <svg viewBox="0 0 640 90" preserveAspectRatio="none">
            <path class="docker-chart__area" :d="networkAreaPath" />
            <path class="docker-chart__line" :d="networkPath" />
          </svg>
        </article>
      </section>
    </section>
  </div>
</template>

<style scoped>
.docker-overview-root {
  display: contents;
}
</style>
