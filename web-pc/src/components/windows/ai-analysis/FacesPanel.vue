<script setup lang="ts">
import { onMounted, ref } from 'vue';
import { RefreshCw, ScanFace, UserCheck } from 'lucide-vue-next';
import { aiAnalysisStore } from '../../../stores/aiAnalysis';
import { UiBadge, UiButton, UiCard, UiEmptyState, useToast } from '../../ui';

const toast = useToast();
const faces = aiAnalysisStore.faces;
const loading = aiAnalysisStore.facesLoading;
const draft = ref<Record<string, string>>({});
const busy = ref('');

async function rename(label: string) {
  const name = (draft.value[label] ?? '').trim();
  if (!name) {
    toast.error('请输入人物名称');
    return;
  }
  busy.value = label;
  try {
    const confirmed = await aiAnalysisStore.labelFace(label, name);
    draft.value = { ...draft.value, [label]: '' };
    toast.success(`已命名「${name}」，确认 ${confirmed} 个样本`);
  } catch (error) {
    toast.error(`命名失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    busy.value = '';
  }
}

async function retrain() {
  busy.value = 'retrain';
  try {
    const taskId = await aiAnalysisStore.retrainFaces();
    toast.success(`已启动人脸模型自训练（${taskId}）`);
  } catch (error) {
    toast.error(`自训练失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    busy.value = '';
  }
}

onMounted(() => void aiAnalysisStore.loadFaces());
</script>

<template>
  <div class="faces">
    <header class="faces__head">
      <div class="faces__title">
        <ScanFace :size="16" :stroke-width="2" />
        <strong>人脸库</strong>
        <span v-if="faces">{{ faces.training.namedPeople }} 已命名 · {{ faces.clusters.length }} 簇 · {{ faces.training.confirmedSamples }}/{{ faces.training.totalSamples }} 样本</span>
      </div>
      <div class="faces__head-actions">
        <UiBadge :tone="faces?.embedderReady ? 'success' : 'neutral'" variant="soft" size="sm">
          {{ faces?.embedderReady ? `识别模型：${faces.embedderName}` : '识别模型未配置' }}
        </UiBadge>
        <UiButton
          variant="soft"
          tone="primary"
          size="sm"
          :icon-left="RefreshCw"
          :loading="busy === 'retrain'"
          :disabled="!faces?.trainerReady"
          :title="faces?.trainerReady ? '基于已命名样本重新训练人脸模型' : '未配置人脸训练 sidecar（HIGO_FACE_TRAINER_URL）'"
          @click="retrain"
        >
          自训练
        </UiButton>
      </div>
    </header>

    <UiEmptyState
      v-if="!loading && (!faces || faces.clusters.length === 0)"
      :icon="ScanFace"
      title="暂无人脸聚类"
      description="开启深度分析并配置视觉模型后，相册照片中的人脸会在这里成簇出现，可命名归并、用于自训练。"
      compact
    />

    <div v-else class="faces__grid">
      <UiCard v-for="cluster in faces?.clusters ?? []" :key="cluster.label" class="faces__card">
        <div class="faces__card-head">
          <UserCheck :size="15" :stroke-width="2" />
          <strong>{{ cluster.label }}</strong>
          <UiBadge tone="neutral" variant="soft" size="sm">{{ cluster.count }} 张</UiBadge>
        </div>
        <div class="faces__rename">
          <input
            v-model="draft[cluster.label]"
            class="faces__input"
            type="text"
            placeholder="命名此人"
            @keyup.enter="rename(cluster.label)"
          />
          <UiButton variant="soft" size="sm" :loading="busy === cluster.label" @click="rename(cluster.label)">命名</UiButton>
        </div>
      </UiCard>
    </div>

    <div v-if="faces?.models?.length" class="faces__models">
      <span class="faces__models-title">模型版本</span>
      <UiBadge
        v-for="m in faces.models"
        :key="m.id"
        :tone="m.active ? 'success' : 'neutral'"
        variant="soft"
        size="sm"
      >
        {{ m.id }} · {{ m.sampleCount }} 样本{{ m.active ? ' · 启用中' : '' }}
      </UiBadge>
    </div>
  </div>
</template>

<style scoped>
.faces {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  height: 100%;
  min-height: 0;
  overflow-y: auto;
}
.faces__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  flex-wrap: wrap;
  gap: var(--space-2);
}
.faces__title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.faces__title span {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
.faces__head-actions {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}
.faces__grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(220px, 1fr));
  gap: var(--space-3);
}
.faces__card {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  padding: var(--space-3);
}
.faces__card-head {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-strong);
  font-size: var(--fs-sm);
}
.faces__card-head strong {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.faces__rename {
  display: flex;
  gap: var(--space-2);
}
.faces__input {
  flex: 1;
  min-width: 0;
  padding: 4px 8px;
  border-radius: var(--radius-control);
  border: 1px solid var(--control-border);
  background: var(--control-bg);
  color: var(--text-strong);
  font-size: var(--fs-xs);
}
.faces__models {
  display: flex;
  flex-wrap: wrap;
  align-items: center;
  gap: var(--space-2);
}
.faces__models-title {
  color: var(--text-muted);
  font-size: var(--fs-2xs);
}
</style>
