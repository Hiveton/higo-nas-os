<script setup lang="ts">
import { HardDrive, Search, ShieldCheck } from 'lucide-vue-next';
import type { AccountUser, Disk } from '../../../api/types';
import { UiButton, UiInput, UiModal, UiRadio } from '../../ui';

type WizardStep = 1 | 2 | 3 | 4;

type FileSystemOption = {
  key: 'btrfs' | 'zfs' | 'ext4';
  label: string;
  tag?: string;
  pros: string;
  cons: string;
  scenario: string;
};

type ModeOption = {
  key: string;
  label: string;
  group: '无数据保护' | '有数据保护';
  minDisks: number;
  evenOnly?: boolean;
  capacity: (sizes: number[]) => number;
  protection: (sizes: number[]) => number;
  performance: (count: number) => string;
  loss: (count: number) => string;
  pros: string;
  cons: string;
};

type WizardState = {
  name: string;
  fileSystem: string;
  mode: string;
  selectedDiskSlots: string[];
  selectedUserIds: string[];
  quotaLimited: boolean;
  quotaGB: number;
  scanBeforeCreate: boolean;
  formatDisk: boolean;
};

defineProps<{
  wizard: WizardState;
  wizardStep: WizardStep;
  createDone: boolean;
  createdSpaceName: string;
  fileSystems: FileSystemOption[];
  usableDisks: Disk[];
  groupedModes: Array<{ name: ModeOption['group']; modes: ModeOption[] }>;
  selectedSizes: number[];
  selectedMode: ModeOption;
  currentFileSystem: FileSystemOption;
  selectedWizardDisks: Disk[];
  estimatedCapacity: number;
  protectedCapacity: number;
  unusedCapacity: number;
  canAdvanceWizard: boolean;
  busyAction: string;
  accountUsers: AccountUser[];
  diskKind: (disk: Disk) => string;
  diskLabel: (disk: Disk) => string;
  diskProtocol: (disk: Disk) => string;
  formatGB: (value: number) => string;
  sum: (values: number[]) => number;
  userInitial: (user: AccountUser) => string;
}>();

const emit = defineEmits<{
  (e: 'close'): void;
  (e: 'prev'): void;
  (e: 'next'): void;
  (e: 'create'): void;
  (e: 'toggle-disk', slot: string): void;
  (e: 'select-mode', mode: ModeOption): void;
  (e: 'toggle-user', id: string): void;
}>();
</script>

<template>
  <UiModal :open="true" size="lg" title="创建存储空间" @close="emit('close')">
    <template v-if="!createDone">
      <div class="storage-monitor__wizard-head">
        <span>{{ wizardStep === 1 ? '选择文件系统' : wizardStep === 2 ? '选择硬盘和存储模式' : wizardStep === 3 ? '选择用户' : '确认信息' }}</span>
        <strong>步骤 <b>{{ wizardStep }}</b> / 4</strong>
      </div>

      <main class="storage-monitor__wizard-body">
        <section v-if="wizardStep === 1" class="storage-monitor__fs-list">
          <button
            v-for="item in fileSystems"
            :key="item.key"
            type="button"
            :class="{ 'storage-monitor__choice--active': wizard.fileSystem === item.key }"
            @click="wizard.fileSystem = item.key"
          >
            <h4>{{ item.label }} <small v-if="item.tag">{{ item.tag }}</small></h4>
            <p><b>优点：</b>{{ item.pros }}</p>
            <p><b class="storage-monitor__danger">缺点：</b>{{ item.cons }}</p>
            <p><b>适用场景：</b>{{ item.scenario }}</p>
          </button>
        </section>

        <section v-else-if="wizardStep === 2" class="storage-monitor__mode-step">
          <aside>
            <h4>选择硬盘</h4>
            <p>内置硬盘</p>
            <button
              v-for="disk in usableDisks"
              :key="disk.slot"
              type="button"
              :class="{ 'storage-monitor__disk-select--active': wizard.selectedDiskSlots.includes(disk.slot) }"
              @click="emit('toggle-disk', disk.slot)"
            >
              <span>{{ wizard.selectedDiskSlots.includes(disk.slot) ? '✓' : disk.slot }}</span>
              <HardDrive :size="20" />
              <strong>{{ disk.devicePath || disk.slot }} · {{ disk.size }}</strong>
              <small>{{ diskKind(disk) }} ｜ {{ diskProtocol(disk) }} ｜ {{ diskLabel(disk) }}</small>
            </button>
          </aside>

          <div class="storage-monitor__mode-list">
            <h4>选择存储模式</h4>
            <p>已选择 {{ wizard.selectedDiskSlots.length }} 块硬盘。可选择以下存储模式：</p>
            <section v-for="group in groupedModes" :key="group.name">
              <h5>{{ group.name }}</h5>
              <button
                v-for="mode in group.modes"
                :key="mode.key"
                type="button"
                :class="{ 'storage-monitor__choice--active': wizard.mode === mode.key }"
                @click="emit('select-mode', mode)"
              >
                <h4>{{ mode.label }}</h4>
                <dl>
                  <div><dt>可用容量</dt><dd>{{ formatGB(mode.capacity(selectedSizes)) }}</dd></div>
                  <div><dt>理论性能</dt><dd>{{ mode.performance(wizard.selectedDiskSlots.length) }}</dd></div>
                  <div><dt>数据防丢能力</dt><dd>{{ mode.loss(wizard.selectedDiskSlots.length) }}</dd></div>
                </dl>
                <p><b>优点：</b>{{ mode.pros }}</p>
                <p><b class="storage-monitor__danger">缺点：</b>{{ mode.cons }}</p>
              </button>
            </section>
          </div>
        </section>

        <section v-else-if="wizardStep === 3" class="storage-monitor__users-step">
          <div>
            <h4>选择可使用此存储空间的用户</h4>
            <UiInput :prefix-icon="Search" placeholder="搜索设备内的用户" />
            <button
              v-for="user in accountUsers"
              :key="user.id"
              type="button"
              class="storage-monitor__user-row"
              @click="emit('toggle-user', user.id)"
            >
              <span :class="{ 'storage-monitor__checkbox--active': wizard.selectedUserIds.includes(user.id) }">
                {{ wizard.selectedUserIds.includes(user.id) ? '✓' : '' }}
              </span>
              <i>{{ userInitial(user) }}</i>
              <strong>{{ user.displayName || user.username }}</strong>
              <small>{{ user.role === 'admin' ? '管理员' : user.role }}</small>
            </button>
          </div>
          <aside>
            <h4>设置可用容量</h4>
            <UiRadio v-model="wizard.quotaLimited" :value="false" label="不限制" />
            <UiRadio v-model="wizard.quotaLimited" :value="true" label="限制普通用户的可用容量" />
            <div class="storage-monitor__quota" :class="{ 'storage-monitor__quota--disabled': !wizard.quotaLimited }">
              <UiInput v-model.number="wizard.quotaGB" type="number" :disabled="!wizard.quotaLimited" />
              <span>GB</span>
            </div>
          </aside>
        </section>

        <section v-else class="storage-monitor__confirm-step">
          <h4>已选硬盘</h4>
          <div class="storage-monitor__confirm-disks">
            <span v-for="disk in selectedWizardDisks" :key="disk.slot">
              <HardDrive :size="17" />
              <b>{{ diskKind(disk) }} · {{ disk.size }}</b>
              <small>{{ diskProtocol(disk) }} · {{ disk.serial }}</small>
            </span>
          </div>
          <dl>
            <div><dt>文件系统</dt><dd>{{ currentFileSystem.label }}</dd></div>
            <div><dt>存储模式</dt><dd>{{ selectedMode.label }}</dd></div>
            <div><dt>预计可用容量</dt><dd>{{ formatGB(estimatedCapacity) }}</dd></div>
            <div><dt>存储空间描述</dt><dd><UiInput v-model="wizard.name" /></dd></div>
          </dl>
          <div class="storage-monitor__scan">
            <h4>硬盘读写检测</h4>
            <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': wizard.scanBeforeCreate }">
              <UiRadio v-model="wizard.scanBeforeCreate" :value="true" />
              <span>
                <strong>执行检测</strong>
                <small>降低数据错误风险，耗时较久。</small>
              </span>
            </label>
            <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': !wizard.scanBeforeCreate }">
              <UiRadio v-model="wizard.scanBeforeCreate" :value="false" />
              <span>
                <strong>跳过检测</strong>
                <small>仅全新硬盘或已确认健康时建议跳过。</small>
              </span>
            </label>
          </div>
          <div class="storage-monitor__format-confirm">
            <h4>格式化方式</h4>
            <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': wizard.formatDisk }">
              <UiRadio v-model="wizard.formatDisk" :value="true" />
              <span>
                <strong>{{ wizard.fileSystem === 'zfs' ? '格式化并创建 ZFS 池' : '格式化并挂载' }}</strong>
                <small>{{ wizard.fileSystem === 'zfs' ? '清除所选硬盘签名，创建 ZFS 存储池并挂载。' : '清除所选硬盘签名，按当前文件系统重新格式化后挂载。' }}</small>
              </span>
            </label>
            <label class="storage-monitor__scan-option" :class="{ 'storage-monitor__scan-option--active': !wizard.formatDisk, 'storage-monitor__scan-option--disabled': wizard.fileSystem === 'zfs' }">
              <UiRadio v-model="wizard.formatDisk" :value="false" :disabled="wizard.fileSystem === 'zfs'" />
              <span>
                <strong>不格式化，仅挂载</strong>
                <small>{{ wizard.fileSystem === 'zfs' ? 'ZFS 创建存储空间需要新建池；已有池导入后续单独做。' : '保留硬盘数据，要求硬盘已有可识别文件系统。' }}</small>
              </span>
            </label>
          </div>
        </section>
      </main>
    </template>

    <div v-else class="storage-monitor__success">
      <ShieldCheck :size="78" />
      <h3>已创建{{ createdSpaceName }}</h3>
      <p>存储空间已写入后端，用户授权和容量策略已同步。</p>
    </div>

    <template #footer>
      <template v-if="!createDone">
        <UiButton variant="outline" tone="neutral" :disabled="wizardStep === 1" @click="emit('prev')">上一步</UiButton>
        <div class="storage-monitor__capacity-plan">
          <strong>预计容量</strong>
          <i><b :style="{ width: `${Math.min(100, (estimatedCapacity / Math.max(1, sum(selectedSizes))) * 100)}%` }" /></i>
          <span>可用容量：{{ formatGB(estimatedCapacity) }}</span>
          <span>数据保护：{{ formatGB(protectedCapacity) }}</span>
          <span>未利用：{{ formatGB(unusedCapacity) }}</span>
        </div>
        <UiButton variant="ghost" tone="neutral" @click="emit('close')">取消</UiButton>
        <UiButton
          tone="primary"
          :loading="wizardStep === 4 && busyAction === 'create-space'"
          :disabled="!canAdvanceWizard"
          @click="wizardStep === 4 ? emit('create') : emit('next')"
        >
          {{ wizardStep === 4 ? '创建' : '下一步' }}
        </UiButton>
      </template>
      <UiButton v-else tone="primary" @click="emit('close')">完成</UiButton>
    </template>
  </UiModal>
</template>
