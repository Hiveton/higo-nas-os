<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { ShieldAlert, AlertTriangle, Trash2 } from 'lucide-vue-next';
import { UiModal, UiButton, UiInput, UiBadge } from '../../ui';
import { apiClient } from '../../../api/client';
import type { StorageSpace, StorageDeletePreview } from '../../../api/types';

// Progressive 3-step delete guard for a storage space:
//   1. impact summary + risk acknowledgement
//   2. type the exact space name to confirm intent
//   3. final irreversible confirmation -> calls the governed confirm endpoint
const props = defineProps<{ space: StorageSpace }>();
const emit = defineEmits<{ (e: 'close'): void; (e: 'deleted', message: string): void }>();

const step = ref(1);
const preview = ref<StorageDeletePreview | null>(null);
const typed = ref('');
const loading = ref(false);
const errorText = ref('');

const nameMatches = computed(() => typed.value.trim() === props.space.name);

onMounted(async () => {
  loading.value = true;
  try {
    preview.value = await apiClient.storage.previewDeleteSpace(props.space.id);
  } catch (error) {
    errorText.value = `无法发起删除：${error instanceof Error ? error.message : '未知错误'}`;
  } finally {
    loading.value = false;
  }
});

function next() {
  if (step.value === 1 && !preview.value) return;
  if (step.value === 2 && !nameMatches.value) return;
  step.value += 1;
}

function back() {
  if (step.value > 1) step.value -= 1;
}

async function confirmDelete() {
  if (!preview.value) return;
  loading.value = true;
  errorText.value = '';
  try {
    const task = await apiClient.storage.confirmDeleteSpace(props.space.id, {
      confirmationId: preview.value.confirmationId,
      actor: 'storage-manager',
    });
    emit('deleted', `${props.space.name} 删除完成：${task.message ?? task.id}`);
  } catch (error) {
    // Token is single-use; on failure the user must restart from preview.
    errorText.value = `删除失败：${error instanceof Error ? error.message : '未知错误'}`;
    step.value = 1;
    try {
      preview.value = await apiClient.storage.previewDeleteSpace(props.space.id);
    } catch {
      preview.value = null;
    }
  } finally {
    loading.value = false;
  }
}
</script>

<template>
  <UiModal :open="true" size="md" :close-on-backdrop="false" @update:open="emit('close')">
    <template #title>
      <span class="sdd__title"><Trash2 :size="16" /> 删除存储空间</span>
    </template>

    <div class="sdd">
      <div class="sdd__steps" aria-hidden="true">
        <i v-for="n in 3" :key="n" :class="{ 'sdd__steps--on': n <= step }" />
      </div>

      <p v-if="errorText" class="sdd__error">{{ errorText }}</p>

      <!-- Step 1: impact + risk acknowledgement -->
      <section v-if="step === 1" class="sdd__panel">
        <div class="sdd__risk">
          <ShieldAlert :size="18" />
          <div>
            <strong>步骤 1 / 3 · 确认影响</strong>
            <UiBadge tone="danger" size="sm">{{ preview?.riskLabel ?? '高风险' }}</UiBadge>
          </div>
        </div>
        <dl class="sdd__facts">
          <div><dt>空间名称</dt><dd>{{ space.name }}</dd></div>
          <div><dt>存储模式</dt><dd>{{ preview?.mode ?? space.mode }}</dd></div>
          <div><dt>容量</dt><dd>{{ preview?.total ?? space.total }}</dd></div>
          <div><dt>关联磁盘</dt><dd>{{ (preview?.diskSlots ?? space.diskSlots).join('、') || '—' }}</dd></div>
        </dl>
        <p class="sdd__impact">{{ preview?.impact ?? '正在加载影响摘要…' }}</p>
      </section>

      <!-- Step 2: type the name -->
      <section v-else-if="step === 2" class="sdd__panel">
        <strong>步骤 2 / 3 · 输入空间名称以继续</strong>
        <p class="sdd__hint">请准确输入 <b>{{ space.name }}</b> 以确认你清楚要删除哪个空间。</p>
        <UiInput v-model="typed" :placeholder="space.name" />
        <p v-if="typed && !nameMatches" class="sdd__mismatch">名称不匹配</p>
      </section>

      <!-- Step 3: final confirmation -->
      <section v-else class="sdd__panel">
        <div class="sdd__final">
          <AlertTriangle :size="22" />
          <div>
            <strong>步骤 3 / 3 · 最终确认</strong>
            <p>即将<strong>永久删除「{{ space.name }}」</strong>。此操作<strong>不可恢复</strong>,空间内数据将全部丢失。</p>
          </div>
        </div>
      </section>
    </div>

    <template #footer>
      <UiButton variant="ghost" tone="neutral" :disabled="loading" @click="emit('close')">取消</UiButton>
      <UiButton v-if="step > 1" variant="soft" tone="neutral" :disabled="loading" @click="back">上一步</UiButton>
      <UiButton
        v-if="step === 1"
        tone="danger"
        :disabled="!preview || loading"
        @click="next"
      >
        我已了解风险,继续
      </UiButton>
      <UiButton
        v-else-if="step === 2"
        tone="danger"
        :disabled="!nameMatches || loading"
        @click="next"
      >
        下一步
      </UiButton>
      <UiButton
        v-else
        tone="danger"
        :icon-left="Trash2"
        :loading="loading"
        @click="confirmDelete"
      >
        永久删除
      </UiButton>
    </template>
  </UiModal>
</template>

<style scoped>
.sdd__title { display: inline-flex; align-items: center; gap: 7px; }
.sdd { display: flex; flex-direction: column; gap: 14px; min-width: 0; }
.sdd__steps { display: flex; gap: 6px; }
.sdd__steps i {
  flex: 1; height: 4px; border-radius: 999px;
  background: var(--border);
}
.sdd__steps i.sdd__steps--on { background: var(--accent-red, #e0564a); }
.sdd__panel { display: flex; flex-direction: column; gap: 11px; }
.sdd__risk, .sdd__final { display: flex; align-items: flex-start; gap: 10px; color: var(--accent-red, #e0564a); }
.sdd__risk strong, .sdd__final strong { color: var(--text-strong); margin-right: 8px; }
.sdd__final p { margin: 6px 0 0; color: var(--text-muted); font-size: 13px; line-height: 1.5; }
.sdd__final p strong { color: var(--accent-red, #e0564a); }
.sdd__facts { display: grid; gap: 8px; margin: 0; }
.sdd__facts > div { display: flex; justify-content: space-between; gap: 12px; font-size: 13px; }
.sdd__facts dt { color: var(--text-muted); margin: 0; }
.sdd__facts dd { color: var(--text-strong); margin: 0; text-align: right; word-break: break-word; }
.sdd__impact {
  margin: 0; padding: 10px 12px; font-size: 12px; line-height: 1.6;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border); border-radius: var(--radius-md);
}
.sdd__hint { margin: 0; color: var(--text-muted); font-size: 13px; }
.sdd__hint b { color: var(--text-strong); }
.sdd__mismatch { margin: 0; color: var(--accent-red, #e0564a); font-size: 12px; }
.sdd__error {
  margin: 0; padding: 8px 11px; font-size: 12px;
  color: var(--accent-red, #e0564a);
  background: rgba(224, 86, 74, 0.1);
  border: 1px solid rgba(224, 86, 74, 0.3); border-radius: var(--radius-md);
}
</style>
