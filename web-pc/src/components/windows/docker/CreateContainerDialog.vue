<script setup lang="ts">
import { computed } from 'vue';
import {
  Activity,
  CheckCircle2,
  ChevronLeft,
  ChevronRight,
  HardDrive,
  ListFilter,
  Network,
  Play,
  Plug,
  Plus,
  RotateCw,
  ShieldCheck,
  TerminalSquare,
  X,
} from 'lucide-vue-next';
import {
  UiButton,
  UiCheckbox,
  UiFormField,
  UiIconButton,
  UiInput,
  UiModal,
  UiSelect,
  UiTextarea,
} from '../../ui';

type PortProtocol = 'tcp' | 'udp';
type CreateStep = 0 | 1 | 2;

type PortRow = { host: string; container: string; protocol: PortProtocol };
type MountRow = { host: string; container: string; mode: 'rw' | 'ro' };
type KeyValueRow = { key: string; value: string };

type LocalImageOption = { id: string; ref: string; size: string };

type CreateContainerForm = {
  image: string;
  name: string;
  restartPolicy: string;
  privileged: boolean;
  autoRemove: boolean;
  limitCpu: number;
  limitMemory: number;
  network: string;
  hostname: string;
  user: string;
  workingDir: string;
  entrypoint: string;
  command: string;
  ports: PortRow[];
  mounts: MountRow[];
  env: KeyValueRow[];
  labels: KeyValueRow[];
  extraHosts: KeyValueRow[];
  dns: string;
};

const props = defineProps<{
  open: boolean;
  createForm: CreateContainerForm;
  createStep: CreateStep;
  createSteps: readonly string[];
  canAdvanceCreate: boolean;
  creating: boolean;
  localImageOptions: LocalImageOption[];
  restartPolicies: { value: string; label: string; desc: string }[];
  networkOptions: string[];
  protocolOptions: PortProtocol[];
  createSummaryRows: { label: string; value: string }[];
  resolveLocalImageRef: (image?: string) => string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'set-step', index: number): void;
  (e: 'prev-step'): void;
  (e: 'next-step'): void;
  (e: 'submit'): void;
  (e: 'add-port'): void;
  (e: 'remove-port', index: number): void;
  (e: 'add-mount'): void;
  (e: 'remove-mount', index: number): void;
  (e: 'add-kv', kind: 'env' | 'labels' | 'extraHosts'): void;
  (e: 'remove-kv', kind: 'env' | 'labels' | 'extraHosts', index: number): void;
}>();

const imageSelectOptions = computed(() =>
  props.localImageOptions.map((image) => ({ label: `${image.ref} · ${image.size}`, value: image.ref })),
);
const restartSelectOptions = computed(() => props.restartPolicies.map((policy) => ({ label: policy.label, value: policy.value })));
const networkSelectOptions = computed(() => props.networkOptions.map((network) => ({ label: network, value: network })));
const protocolSelectOptions = computed(() => props.protocolOptions.map((protocol) => ({ label: protocol.toUpperCase(), value: protocol })));
const modeSelectOptions = [
  { label: '读写', value: 'rw' },
  { label: '只读', value: 'ro' },
];
</script>

<template>
  <UiModal :open="open" title="添加容器" size="lg" @close="emit('close')">
    <div class="docker-wizard">
      <nav class="docker-wizard-steps" aria-label="创建步骤">
        <button
          v-for="(step, index) in createSteps"
          :key="step"
          type="button"
          :class="{ 'is-active': createStep === index, 'is-done': createStep > index }"
          :disabled="index > createStep && !canAdvanceCreate"
          @click="emit('set-step', index)"
        >
          <span>{{ index + 1 }}</span>{{ step }}
        </button>
      </nav>

      <section v-if="createStep === 0" class="docker-wizard-panel">
        <div class="docker-form-grid">
          <UiFormField label="镜像" required :hint="localImageOptions.length === 0 ? '请先到镜像仓库拉取镜像，或刷新本地镜像列表。' : undefined">
            <UiSelect
              v-model="createForm.image"
              :options="imageSelectOptions"
              :disabled="localImageOptions.length === 0"
              :placeholder="localImageOptions.length ? '选择本地镜像' : '暂无本地镜像'"
            />
          </UiFormField>
          <UiFormField label="容器名">
            <UiInput v-model="createForm.name" placeholder="留空则自动生成" />
          </UiFormField>
        </div>

        <section class="docker-wizard-section">
          <header>
            <h4><ShieldCheck :size="14" /> 权限</h4>
          </header>
          <div class="docker-choice-grid">
            <button type="button" :class="{ 'is-active': !createForm.privileged }" @click="createForm.privileged = false">
              <strong>普通权限</strong>
              <span>适合大多数服务，隔离性更好。</span>
            </button>
            <button type="button" :class="{ 'is-active': createForm.privileged }" @click="createForm.privileged = true">
              <strong>高权限</strong>
              <span>允许访问更多宿主机能力，仅在硬件直通等场景使用。</span>
            </button>
          </div>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><RotateCw :size="14" /> 重启与资源</h4>
          </header>
          <div class="docker-form-grid">
            <UiFormField label="重启策略">
              <UiSelect v-model="createForm.restartPolicy" :options="restartSelectOptions" :disabled="createForm.autoRemove" />
            </UiFormField>
            <div class="docker-check-row docker-check-row--card">
              <UiCheckbox v-model="createForm.autoRemove" label="退出后自动删除容器" />
            </div>
            <UiFormField label="CPU 限制">
              <UiInput v-model.number="createForm.limitCpu" type="number" placeholder="0 表示不限制" />
            </UiFormField>
            <UiFormField label="内存限制 MB">
              <UiInput v-model.number="createForm.limitMemory" type="number" placeholder="0 表示不限制" />
            </UiFormField>
          </div>
        </section>
      </section>

      <section v-else-if="createStep === 1" class="docker-wizard-panel">
        <section class="docker-wizard-section">
          <header>
            <h4><Network :size="14" /> 网络</h4>
          </header>
          <div class="docker-form-grid docker-form-grid--three">
            <UiFormField label="网络模式">
              <UiSelect v-model="createForm.network" :options="networkSelectOptions" />
            </UiFormField>
            <UiFormField label="主机名">
              <UiInput v-model="createForm.hostname" placeholder="可选" />
            </UiFormField>
            <UiFormField label="运行用户">
              <UiInput v-model="createForm.user" placeholder="例如 1000:1000" />
            </UiFormField>
          </div>
          <UiFormField class="docker-field--span" label="DNS">
            <UiInput v-model="createForm.dns" placeholder="多个 DNS 用逗号分隔，例如 223.5.5.5,119.29.29.29" />
          </UiFormField>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><Plug :size="14" /> 端口映射</h4>
            <UiButton variant="soft" size="sm" :icon-left="Plus" @click="emit('add-port')">添加</UiButton>
          </header>
          <div class="docker-dynamic-list">
            <div v-for="(port, index) in createForm.ports" :key="`port-${index}`" class="docker-dynamic-row docker-dynamic-row--ports">
              <UiInput v-model="port.host" placeholder="本地端口，可选" />
              <UiInput v-model="port.container" placeholder="容器端口，例如 80" />
              <UiSelect :model-value="port.protocol" :options="protocolSelectOptions" @update:model-value="(value) => (port.protocol = value as PortProtocol)" />
              <UiIconButton :icon="X" label="移除端口" size="sm" variant="soft" @click="emit('remove-port', index)" />
            </div>
          </div>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><HardDrive :size="14" /> 挂载目录</h4>
            <UiButton variant="soft" size="sm" :icon-left="Plus" @click="emit('add-mount')">添加</UiButton>
          </header>
          <div class="docker-dynamic-list">
            <div v-for="(mount, index) in createForm.mounts" :key="`mount-${index}`" class="docker-dynamic-row docker-dynamic-row--mounts">
              <UiInput v-model="mount.host" placeholder="宿主机路径，例如 /srv/data" />
              <UiInput v-model="mount.container" placeholder="容器路径，例如 /data" />
              <UiSelect :model-value="mount.mode" :options="modeSelectOptions" @update:model-value="(value) => (mount.mode = value as 'rw' | 'ro')" />
              <UiIconButton :icon="X" label="移除挂载" size="sm" variant="soft" @click="emit('remove-mount', index)" />
            </div>
          </div>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><Activity :size="14" /> 环境变量</h4>
            <UiButton variant="soft" size="sm" :icon-left="Plus" @click="emit('add-kv', 'env')">添加</UiButton>
          </header>
          <div class="docker-dynamic-list">
            <div v-for="(item, index) in createForm.env" :key="`env-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
              <UiInput v-model="item.key" placeholder="变量名，例如 TZ" />
              <UiInput v-model="item.value" placeholder="变量值，例如 Asia/Shanghai" />
              <UiIconButton :icon="X" label="移除环境变量" size="sm" variant="soft" @click="emit('remove-kv', 'env', index)" />
            </div>
          </div>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><ListFilter :size="14" /> 标签与解析</h4>
            <UiButton variant="soft" size="sm" :icon-left="Plus" @click="emit('add-kv', 'labels')">添加标签</UiButton>
          </header>
          <div class="docker-dynamic-list">
            <div v-for="(item, index) in createForm.labels" :key="`label-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
              <UiInput v-model="item.key" placeholder="标签名" />
              <UiInput v-model="item.value" placeholder="标签值" />
              <UiIconButton :icon="X" label="移除标签" size="sm" variant="soft" @click="emit('remove-kv', 'labels', index)" />
            </div>
            <div v-for="(item, index) in createForm.extraHosts" :key="`host-${index}`" class="docker-dynamic-row docker-dynamic-row--kv">
              <UiInput v-model="item.key" placeholder="主机名，例如 db.local" />
              <UiInput v-model="item.value" placeholder="IP，例如 192.168.1.10" />
              <UiIconButton :icon="X" label="移除主机解析" size="sm" variant="soft" @click="emit('remove-kv', 'extraHosts', index)" />
            </div>
            <UiButton variant="soft" size="sm" :icon-left="Plus" @click="emit('add-kv', 'extraHosts')">添加主机解析</UiButton>
          </div>
        </section>

        <section class="docker-wizard-section">
          <header>
            <h4><TerminalSquare :size="14" /> 命令</h4>
          </header>
          <div class="docker-form-grid">
            <UiFormField label="工作目录">
              <UiInput v-model="createForm.workingDir" placeholder="可选，例如 /app" />
            </UiFormField>
            <UiFormField label="入口程序">
              <UiInput v-model="createForm.entrypoint" placeholder="可选，例如 /bin/sh" />
            </UiFormField>
            <UiFormField class="docker-field--span" label="启动命令">
              <UiTextarea v-model="createForm.command" :rows="2" placeholder="可选，留空使用镜像默认命令" />
            </UiFormField>
          </div>
        </section>
      </section>

      <section v-else class="docker-wizard-panel">
        <section class="docker-review-card">
          <header>
            <h4><CheckCircle2 :size="15" /> 创建摘要</h4>
            <span>确认后将拉取缺失镜像并启动容器。</span>
          </header>
          <div class="docker-summary-list">
            <div v-for="row in createSummaryRows" :key="row.label">
              <span>{{ row.label }}</span>
              <strong>{{ row.value }}</strong>
            </div>
          </div>
        </section>
      </section>
    </div>

    <template #footer>
      <UiButton variant="ghost" tone="neutral" @click="emit('close')">取消</UiButton>
      <UiButton v-if="createStep > 0" variant="ghost" tone="neutral" :icon-left="ChevronLeft" @click="emit('prev-step')">上一步</UiButton>
      <UiButton v-if="createStep < 2" variant="solid" tone="primary" :icon-right="ChevronRight" :disabled="!canAdvanceCreate" @click="emit('next-step')">下一步</UiButton>
      <UiButton v-else variant="solid" tone="primary" :icon-left="Play" :loading="creating" :disabled="!resolveLocalImageRef(createForm.image) || creating" @click="emit('submit')">
        {{ creating ? '创建中' : '完成并启动' }}
      </UiButton>
    </template>
  </UiModal>
</template>
