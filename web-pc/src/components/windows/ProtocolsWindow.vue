<script setup lang="ts">
import { computed, onMounted, reactive, ref, watch } from 'vue';
import {
  Check,
  Copy,
  FolderPlus,
  Network,
  RefreshCcw,
  RotateCcw,
  Save,
  ShieldAlert,
  SlidersHorizontal,
  Trash2,
  Wifi,
} from 'lucide-vue-next';
import {
  UiBadge,
  UiButton,
  UiCheckbox,
  UiEmptyState,
  UiFormField,
  UiInput,
  UiModal,
  UiSelect,
  UiSpinner,
  UiSwitch,
  useConfirm,
  useToast,
} from '../ui';
import type { SelectOption, UiTone } from '../ui';
import { protocolsStore } from '../../stores/protocols';
import type {
  Protocol,
  ProtocolAccessLevel,
  ProtocolAuditEntry,
  ProtocolConfig,
  ProtocolKey,
  ProtocolShare,
} from '../../api/types';

const store = protocolsStore;
const toast = useToast();
const confirm = useConfirm();

const selectedKey = ref<ProtocolKey>('smb');
const busy = ref(false);
const togglingKey = ref<ProtocolKey | ''>('');
const auditOpen = ref(false);

const protocols = computed(() => store.protocols.value);
const audit = computed(() => store.audit.value);
const loading = computed(() => store.loading.value);
const usingFallback = computed(() => store.usingFallback.value);

const selectedProtocol = computed<Protocol | null>(
  () => protocols.value.find((p) => p.key === selectedKey.value) ?? null,
);
const selectedShares = computed<ProtocolShare[]>(
  () => store.sharesByProtocol.value[selectedKey.value] ?? [],
);
const enabledCount = computed(() => protocols.value.filter((p) => p.enabled).length);

onMounted(async () => {
  await store.loadDashboard();
  if (protocols.value.length && !protocols.value.some((p) => p.key === selectedKey.value)) {
    selectedKey.value = protocols.value[0].key;
  }
  syncConfigForm();
});

function selectProtocol(key: ProtocolKey) {
  selectedKey.value = key;
}

function sharesCount(key: ProtocolKey) {
  return (store.sharesByProtocol.value[key] ?? []).length;
}

function statusTone(p: Protocol): UiTone {
  if (!p.installed) return 'neutral';
  if (p.running) return 'success';
  if (p.enabled) return 'warning';
  return 'neutral';
}

function statusText(p: Protocol): string {
  if (!p.installed) return '未安装';
  if (p.running) return '运行中';
  if (p.enabled) return '启动中';
  return '已关闭';
}

function riskTone(risk: string): UiTone {
  if (risk === 'high') return 'danger';
  if (risk === 'medium') return 'warning';
  return 'info';
}

function accessTone(level: ProtocolAccessLevel): UiTone {
  switch (level) {
    case 'public':
      return 'danger';
    case 'password':
      return 'warning';
    case 'readonly':
      return 'info';
    default:
      return 'success';
  }
}

function accessLabel(level: ProtocolAccessLevel): string {
  switch (level) {
    case 'public':
      return '公开访问';
    case 'password':
      return '密码访问';
    case 'readonly':
      return '只读访问';
    default:
      return '指定账号';
  }
}

const accessOptions: SelectOption[] = [
  { label: '指定账号', value: 'account' },
  { label: '密码访问', value: 'password' },
  { label: '只读访问', value: 'readonly' },
  { label: '公开访问', value: 'public' },
];
const minProtocolOptions: SelectOption[] = [
  { label: 'SMB2（兼容性好）', value: 'SMB2' },
  { label: 'SMB3（更安全）', value: 'SMB3' },
];
const squashOptions: SelectOption[] = [
  { label: 'root_squash（推荐）', value: 'root_squash' },
  { label: 'all_squash', value: 'all_squash' },
  { label: 'no_root_squash', value: 'no_root_squash' },
];

// --- enable / disable (smooth, single action, no extra dialog) --------------

async function onToggleProtocol(p: Protocol, next: boolean) {
  if (togglingKey.value) return;
  togglingKey.value = p.key;
  try {
    await store.setEnabled(p.key, next);
    toast.show(`${p.displayName} ${next ? '已启用' : '已停用'}`, { tone: 'success' });
  } catch (error) {
    toast.show(messageOf(error), { tone: 'danger' });
    await store.loadDashboard();
  } finally {
    togglingKey.value = '';
  }
}

// --- per-protocol configuration ---------------------------------------------

const configForm = reactive<ProtocolConfig>({});

function defaultConfig(): Required<ProtocolConfig> {
  return {
    serverName: '',
    workgroup: '',
    minProtocol: 'SMB2',
    guestAccess: false,
    squash: 'root_squash',
    allowedNetwork: '*',
    httpsEnabled: false,
    friendlyName: '',
  };
}

function syncConfigForm() {
  Object.assign(configForm, defaultConfig(), selectedProtocol.value?.config ?? {});
}

watch(selectedKey, syncConfigForm);
watch(() => selectedProtocol.value?.config, syncConfigForm, { deep: true });

// Only the fields relevant to the selected protocol are persisted on save.
function configPayload(key: ProtocolKey): ProtocolConfig {
  switch (key) {
    case 'smb':
      return {
        serverName: configForm.serverName,
        workgroup: configForm.workgroup,
        minProtocol: configForm.minProtocol,
        guestAccess: configForm.guestAccess,
      };
    case 'nfs':
      return { squash: configForm.squash, allowedNetwork: configForm.allowedNetwork };
    case 'webdav':
      return { httpsEnabled: configForm.httpsEnabled };
    case 'dlna':
      return { friendlyName: configForm.friendlyName };
    default:
      return {};
  }
}

const configDirty = computed(() => {
  const p = selectedProtocol.value;
  if (!p) return false;
  const next = configPayload(p.key);
  return (Object.keys(next) as (keyof ProtocolConfig)[]).some(
    (k) => (next[k] ?? '') !== (p.config[k] ?? ''),
  );
});

async function onSaveConfig() {
  const p = selectedProtocol.value;
  if (!p || busy.value) return;
  busy.value = true;
  try {
    await store.updateConfig(p.key, configPayload(p.key));
    toast.show(`${p.displayName} 设置已保存`, { tone: 'success' });
  } catch (error) {
    toast.show(messageOf(error), { tone: 'danger' });
  } finally {
    busy.value = false;
  }
}

// --- shares -----------------------------------------------------------------

async function copyText(text: string) {
  try {
    await navigator.clipboard.writeText(text);
    toast.show('已复制挂载地址', { tone: 'success' });
  } catch {
    toast.show('复制失败，请手动选择文本', { tone: 'warning' });
  }
}

const showAddShare = ref(false);
const form = reactive({
  name: '',
  path: '',
  accessLevel: 'account' as ProtocolAccessLevel,
  allowedUsers: '',
  guest: false,
});
const needsUsers = computed(() => form.accessLevel === 'account' || form.accessLevel === 'password');

function openAddShare() {
  form.name = '';
  form.path = '';
  form.accessLevel = 'account';
  form.allowedUsers = '';
  form.guest = false;
  showAddShare.value = true;
}

async function submitAddShare() {
  if (!selectedProtocol.value) return;
  if (!form.name.trim() || !form.path.trim()) {
    toast.show('请填写共享名称与目录路径', { tone: 'warning' });
    return;
  }
  busy.value = true;
  try {
    const payload = {
      name: form.name.trim(),
      path: form.path.trim(),
      accessLevel: form.accessLevel,
      allowedUsers: form.allowedUsers
        .split(/[,，\s]+/)
        .map((u) => u.trim())
        .filter(Boolean),
      guest: form.guest || form.accessLevel === 'public',
      actor: 'protocols-ui',
    };
    const key = selectedKey.value;
    const preview = await store.previewCreateShare(key, payload);
    // High-risk public/password shares show an impact dialog; lower-risk ones apply directly.
    if (preview.risk === 'high') {
      const ok = await confirm({
        title: `${preview.riskLabel} · 请确认共享`,
        message: preview.impact,
        confirmLabel: '创建共享',
        cancelLabel: '取消',
        tone: 'danger',
      });
      if (!ok) return;
    }
    await store.confirmCreateShare(key, preview);
    toast.show(`共享「${payload.name}」已创建`, { tone: 'success' });
    showAddShare.value = false;
  } catch (error) {
    toast.show(messageOf(error), { tone: 'danger' });
  } finally {
    busy.value = false;
  }
}

async function onDeleteShare(share: ProtocolShare) {
  if (busy.value) return;
  const ok = await confirm({
    title: '移除共享目录？',
    message: `将移除「${share.name}」(${share.path})，对应客户端将无法再访问。`,
    confirmLabel: '移除',
    tone: 'danger',
  });
  if (!ok) return;
  busy.value = true;
  try {
    const preview = await store.previewDeleteShare(share.id);
    await store.confirmDeleteShare(share.id, preview);
    toast.show(`共享「${share.name}」已移除`, { tone: 'success' });
  } catch (error) {
    toast.show(messageOf(error), { tone: 'danger' });
  } finally {
    busy.value = false;
  }
}

async function onRollback(entry: Pick<ProtocolAuditEntry, 'id' | 'event' | 'reverted' | 'rollbackId' | 'rollback'>) {
  if (busy.value || entry.reverted || !entry.rollbackId) return;
  const ok = await confirm({
    title: '回滚此操作？',
    message: entry.rollback || entry.event,
    confirmLabel: '回滚',
    tone: 'warning',
  });
  if (!ok) return;
  busy.value = true;
  try {
    await store.rollback(entry.id);
    toast.show('已回滚该操作', { tone: 'success' });
  } catch (error) {
    toast.show(messageOf(error), { tone: 'danger' });
  } finally {
    busy.value = false;
  }
}

function messageOf(error: unknown): string {
  return error instanceof Error ? error.message : '操作失败';
}
</script>

<template>
  <div class="protocols">
    <header class="protocols__hero">
      <div class="protocols__hero-text">
        <p><Network :size="13" /> SMB / NFS / WebDAV / DLNA</p>
        <h3>共享协议</h3>
      </div>
      <div class="protocols__hero-meta">
        <UiBadge tone="success" variant="soft">{{ enabledCount }} 个已启用</UiBadge>
        <UiButton variant="ghost" size="sm" :icon-left="RefreshCcw" :disabled="loading" @click="store.loadDashboard()">
          刷新
        </UiButton>
      </div>
    </header>

    <p v-if="usingFallback" class="protocols__offline">
      <ShieldAlert :size="14" /> 暂时无法连接后端，展示的是本地占位数据，操作不会生效。
    </p>

    <div class="protocols__body">
      <!-- Left: protocol list with inline toggles -->
      <aside class="protocols__list">
        <article
          v-for="p in protocols"
          :key="p.key"
          class="protocol-card"
          :class="{ 'protocol-card--active': p.key === selectedKey }"
          role="button"
          tabindex="0"
          @click="selectProtocol(p.key)"
          @keydown.enter="selectProtocol(p.key)"
        >
          <div class="protocol-card__top">
            <strong>{{ p.displayName }}</strong>
            <UiSwitch
              size="sm"
              :model-value="p.running"
              :disabled="togglingKey === p.key || !p.installed"
              @click.stop
              @change="(v: boolean) => onToggleProtocol(p, v)"
            />
          </div>
          <div class="protocol-card__hint">{{ p.mountHint }}</div>
          <div class="protocol-card__foot">
            <UiBadge :tone="statusTone(p)" variant="dot" size="sm">{{ statusText(p) }}</UiBadge>
            <span>端口 {{ p.port }}</span>
            <span class="protocol-card__sep">·</span>
            <span>{{ sharesCount(p.key) }} 个共享</span>
          </div>
        </article>
      </aside>

      <!-- Right: selected protocol detail -->
      <section v-if="selectedProtocol" class="protocols__detail">
        <div class="detail-head">
          <div class="detail-head__title">
            <h4>{{ selectedProtocol.displayName }}</h4>
            <UiBadge :tone="statusTone(selectedProtocol)" variant="soft" size="sm">
              {{ statusText(selectedProtocol) }}
            </UiBadge>
          </div>
          <div class="detail-head__toggle">
            <span>{{ selectedProtocol.enabled ? '已开启' : '已关闭' }}</span>
            <UiSwitch
              :model-value="selectedProtocol.running"
              :disabled="togglingKey === selectedProtocol.key || !selectedProtocol.installed"
              @change="(v: boolean) => onToggleProtocol(selectedProtocol!, v)"
            />
          </div>
        </div>

        <p v-if="!selectedProtocol.installed" class="detail-warn">
          <ShieldAlert :size="14" /> 主机尚未安装该协议所需的服务，请先在部署中安装对应软件包。
        </p>

        <!-- Mount guide -->
        <div class="detail-mount-row">
          <label><Wifi :size="13" /> 挂载地址</label>
          <code>{{ selectedProtocol.mountHint }}</code>
          <UiButton variant="ghost" size="sm" :icon-left="Copy" @click="copyText(selectedProtocol.mountHint)">复制</UiButton>
        </div>
        <p class="detail-compat">兼容客户端：{{ selectedProtocol.compatibility }}</p>

        <!-- Per-protocol configuration -->
        <section class="detail-config">
          <header class="detail-config__head">
            <h5><SlidersHorizontal :size="14" /> 协议设置</h5>
            <UiButton
              variant="solid"
              tone="primary"
              size="sm"
              :icon-left="Save"
              :disabled="busy || !configDirty"
              @click="onSaveConfig"
            >
              保存设置
            </UiButton>
          </header>

          <div class="config-grid">
            <template v-if="selectedProtocol.key === 'smb'">
              <UiFormField label="服务器名称">
                <UiInput v-model="configForm.serverName" placeholder="HiGoOS" />
              </UiFormField>
              <UiFormField label="工作组">
                <UiInput v-model="configForm.workgroup" placeholder="WORKGROUP" />
              </UiFormField>
              <UiFormField label="最低协议版本">
                <UiSelect v-model="configForm.minProtocol" :options="minProtocolOptions" />
              </UiFormField>
              <div class="config-switch">
                <UiSwitch v-model="configForm.guestAccess" size="sm" />
                <span>允许访客访问（无需账号）</span>
              </div>
            </template>

            <template v-else-if="selectedProtocol.key === 'nfs'">
              <UiFormField label="默认权限压缩">
                <UiSelect v-model="configForm.squash" :options="squashOptions" />
              </UiFormField>
              <UiFormField label="允许网段" hint="* 表示不限，或填 CIDR，如 192.168.0.0/16">
                <UiInput v-model="configForm.allowedNetwork" placeholder="*" />
              </UiFormField>
            </template>

            <template v-else-if="selectedProtocol.key === 'webdav'">
              <UiFormField label="服务端口">
                <UiInput :model-value="String(selectedProtocol.port)" disabled />
              </UiFormField>
              <div class="config-switch">
                <UiSwitch v-model="configForm.httpsEnabled" size="sm" />
                <span>启用 HTTPS（在 WebDAV 服务上终止 TLS）</span>
              </div>
            </template>

            <template v-else-if="selectedProtocol.key === 'dlna'">
              <UiFormField label="设备名称" hint="DLNA 客户端上显示的媒体服务器名称">
                <UiInput v-model="configForm.friendlyName" placeholder="HiGoOS 媒体库" />
              </UiFormField>
            </template>
          </div>
        </section>

        <!-- Shared directories -->
        <section class="detail-shares">
          <header class="detail-shares__head">
            <h5>共享目录</h5>
            <UiButton
              variant="soft"
              tone="primary"
              size="sm"
              :icon-left="FolderPlus"
              :disabled="busy"
              @click="openAddShare"
            >
              新增共享
            </UiButton>
          </header>

          <UiEmptyState v-if="!selectedShares.length" title="暂无共享目录" description="点击「新增共享」把一个目录通过该协议开放访问。" />

          <ul v-else class="share-list">
            <li v-for="share in selectedShares" :key="share.id" class="share-row">
              <div class="share-row__main">
                <strong>{{ share.name }}</strong>
                <code class="share-row__path">{{ share.path }}</code>
              </div>
              <div class="share-row__meta">
                <UiBadge :tone="accessTone(share.accessLevel)" variant="soft" size="sm">
                  {{ accessLabel(share.accessLevel) }}
                </UiBadge>
                <UiButton
                  variant="ghost"
                  tone="danger"
                  size="sm"
                  :icon-left="Trash2"
                  :disabled="busy"
                  @click="onDeleteShare(share)"
                >
                  移除
                </UiButton>
              </div>
            </li>
          </ul>
        </section>
      </section>

      <section v-else class="protocols__detail protocols__detail--empty">
        <UiSpinner v-if="loading" />
        <UiEmptyState v-else title="选择一个协议" description="从左侧选择 SMB / NFS / WebDAV / DLNA 查看详情与设置。" />
      </section>
    </div>

    <!-- Audit drawer -->
    <section class="protocols__audit">
      <button type="button" class="protocols__audit-toggle" @click="auditOpen = !auditOpen">
        <RotateCcw :size="13" /> 操作审计 · {{ audit.length }} 条
        <span class="protocols__audit-caret">{{ auditOpen ? '收起' : '展开' }}</span>
      </button>
      <ul v-if="auditOpen" class="audit-list">
        <li v-for="entry in audit" :key="entry.id" class="audit-row">
          <UiBadge :tone="riskTone(entry.risk)" variant="soft" size="sm">{{ entry.riskLabel }}</UiBadge>
          <div class="audit-row__text">
            <span :class="{ 'audit-row__reverted': entry.reverted }">{{ entry.event }}</span>
            <small>{{ entry.actor }} · {{ entry.result }}</small>
          </div>
          <UiButton
            v-if="entry.rollbackId && !entry.reverted"
            variant="ghost"
            size="sm"
            :icon-left="RotateCcw"
            :disabled="busy"
            @click="onRollback(entry)"
          >
            回滚
          </UiButton>
          <UiBadge v-else-if="entry.reverted" tone="neutral" variant="soft" size="sm">
            <Check :size="12" /> 已回滚
          </UiBadge>
        </li>
      </ul>
    </section>

    <!-- Add share modal -->
    <UiModal v-model:open="showAddShare" title="新增共享目录" size="sm">
      <div class="share-form">
        <UiFormField label="共享名称" required>
          <UiInput v-model="form.name" placeholder="例如：家庭共享" />
        </UiFormField>
        <UiFormField label="目录路径" required hint="需位于 NAS 存储根目录之内">
          <UiInput v-model="form.path" placeholder="/家庭空间/相册" />
        </UiFormField>
        <UiFormField label="访问方式">
          <UiSelect v-model="form.accessLevel" :options="accessOptions" />
        </UiFormField>
        <UiFormField v-if="needsUsers" label="授权账号" hint="多个账号用逗号分隔">
          <UiInput v-model="form.allowedUsers" placeholder="alice, bob" />
        </UiFormField>
        <UiCheckbox v-if="form.accessLevel !== 'public'" v-model="form.guest" label="同时允许访客访问" />
      </div>
      <template #footer>
        <UiButton variant="ghost" @click="showAddShare = false">取消</UiButton>
        <UiButton variant="solid" tone="primary" :loading="busy" @click="submitAddShare">创建共享</UiButton>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.protocols {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  height: 100%;
  min-height: 0;
}

.protocols__hero {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-3) var(--space-4);
  background: rgba(var(--surface-rgb), 0.55);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.protocols__hero-text p {
  display: flex;
  align-items: center;
  gap: var(--space-1);
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.protocols__hero-text h3 {
  margin: 4px 0 0;
  color: var(--text-strong);
  font-size: var(--fs-lg);
}

.protocols__hero-meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.protocols__offline {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  color: var(--accent-orange);
  background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
  border: 1px solid color-mix(in srgb, var(--accent-orange) 30%, transparent);
  border-radius: var(--radius-md);
  font-size: var(--fs-xs);
}

.protocols__body {
  display: grid;
  grid-template-columns: 248px 1fr;
  gap: var(--space-3);
  flex: 1;
  min-height: 0;
}

.protocols__list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  overflow: auto;
  padding-right: 2px;
}

.protocol-card {
  display: flex;
  flex-direction: column;
  gap: 8px;
  padding: var(--space-3);
  text-align: left;
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: border-color 0.15s ease, background 0.15s ease;
}

.protocol-card:hover {
  border-color: color-mix(in srgb, var(--accent) 40%, var(--border));
}

.protocol-card--active {
  border-color: var(--accent);
  background: color-mix(in srgb, var(--accent) 10%, rgba(var(--surface-rgb), 0.5));
}

.protocol-card__top {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-2);
}

.protocol-card__top strong {
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.protocol-card__hint {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  font-family: var(--font-mono, monospace);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.protocol-card__foot {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.protocol-card__sep {
  opacity: 0.5;
}

.protocols__detail {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  overflow: auto;
  padding: var(--space-4);
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.protocols__detail--empty {
  align-items: center;
  justify-content: center;
}

.detail-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
}

.detail-head__title {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.detail-head__title h4 {
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-lg);
}

.detail-head__toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text-muted);
  font-size: var(--fs-sm);
}

.detail-warn {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  margin: 0;
  padding: var(--space-2) var(--space-3);
  color: var(--accent-orange);
  background: color-mix(in srgb, var(--accent-orange) 12%, transparent);
  border-radius: var(--radius-sm);
  font-size: var(--fs-xs);
}

.detail-mount-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
}

.detail-mount-row label {
  display: flex;
  align-items: center;
  gap: 4px;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  white-space: nowrap;
}

.detail-mount-row code {
  flex: 1;
  padding: 6px var(--space-2);
  background: rgba(var(--surface-rgb), 0.7);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
  color: var(--text-strong);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.detail-compat {
  margin: 0;
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.detail-config,
.detail-shares {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
  padding: var(--space-3);
  background: rgba(var(--surface-rgb), 0.4);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.detail-config__head,
.detail-shares__head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.detail-config__head h5,
.detail-shares__head h5 {
  display: flex;
  align-items: center;
  gap: 6px;
  margin: 0;
  color: var(--text-strong);
  font-size: var(--fs-md);
}

.config-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: var(--space-3);
}

.config-switch {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  color: var(--text);
  font-size: var(--fs-sm);
}

.share-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-2);
  margin: 0;
  padding: 0;
  list-style: none;
}

.share-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: var(--space-3);
  padding: var(--space-2) var(--space-3);
  background: rgba(var(--surface-rgb), 0.6);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.share-row__main {
  display: flex;
  flex-direction: column;
  gap: 2px;
  min-width: 0;
}

.share-row__main strong {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.share-row__path {
  color: var(--text-muted);
  font-size: var(--fs-xs);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.share-row__meta {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  flex-shrink: 0;
}

.protocols__audit {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.protocols__audit-toggle {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  width: 100%;
  padding: var(--space-2) var(--space-3);
  background: transparent;
  border: none;
  color: var(--text-muted);
  font-size: var(--fs-xs);
  cursor: pointer;
}

.protocols__audit-caret {
  margin-left: auto;
  color: var(--accent);
}

.audit-list {
  display: flex;
  flex-direction: column;
  gap: var(--space-1);
  margin: 0;
  padding: 0 var(--space-3) var(--space-3);
  list-style: none;
  max-height: 150px;
  overflow: auto;
}

.audit-row {
  display: flex;
  align-items: center;
  gap: var(--space-2);
  padding: var(--space-2);
  border-top: 1px solid var(--border);
}

.audit-row__text {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.audit-row__text span {
  color: var(--text-strong);
  font-size: var(--fs-sm);
}

.audit-row__text small {
  color: var(--text-muted);
  font-size: var(--fs-xs);
}

.audit-row__reverted {
  text-decoration: line-through;
  opacity: 0.6;
}

.share-form {
  display: flex;
  flex-direction: column;
  gap: var(--space-3);
}

@media (max-width: 760px) {
  .protocols__body {
    grid-template-columns: 1fr;
  }

  .config-grid {
    grid-template-columns: 1fr;
  }
}
</style>
