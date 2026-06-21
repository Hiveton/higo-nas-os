<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue';
import { FolderPlus, Trash2, FolderTree, Users, Network, Copy, RefreshCw, Sliders, Camera, Trash } from 'lucide-vue-next';
import { UiButton, UiModal, UiInput, UiSelect, UiSegmented, UiSwitch, UiBadge, UiEmptyState, UiTabs, useToast, useConfirm } from '../../ui';
import type { TabItem, SegmentedOption } from '../../ui';
import { sharedFoldersStore } from '../../../stores/sharedFolders';
import { accountsStore } from '../../../stores/accounts';
import { apiClient } from '../../../api/client';
import type { SharedFolderView, SharedFolderPermission, StorageSpace, FolderSnapshot } from '../../../api/types';

const toast = useToast();
const confirm = useConfirm();

const activeId = ref('');
const activeTab = ref('permissions');
const spaces = ref<StorageSpace[]>([]);
const editing = ref<SharedFolderPermission[]>([]);
const savingPerms = ref(false);

const showCreate = ref(false);
const createForm = ref({ spaceId: '', name: '', relPath: '' });

// DSM permission vocabulary: 无 / 只读 / 读写 / 拒绝(显式 deny,覆盖组授权).
const accessOptions: SegmentedOption[] = [
  { label: '无', value: 'none' },
  { label: '只读', value: 'read' },
  { label: '读写', value: 'read_write' },
  { label: '拒绝', value: 'deny' },
];
const tabs: TabItem[] = [
  { key: 'permissions', label: '权限', icon: Users },
  { key: 'services', label: '文件服务', icon: Network },
  { key: 'advanced', label: '高级', icon: Sliders },
];

const advancedForm = ref({ recycle: false, quotaGb: 0, encrypted: false });
const savingAdvanced = ref(false);
const snapshots = ref<FolderSnapshot[]>([]);
const snapName = ref('');
const snapBusy = ref(false);

const folders = computed(() => sharedFoldersStore.folders.value as unknown as SharedFolderView[]);
const groups = computed(
  () => sharedFoldersStore.groupedBySpace.value as unknown as { spaceId: string; spaceName: string; folders: SharedFolderView[] }[],
);
const activeFolder = computed<SharedFolderView | undefined>(() => folders.value.find((f) => f.id === activeId.value));

const addOptions = computed(() => {
  const taken = new Set(editing.value.map((e) => `${e.subjectType}/${e.subjectId}`));
  const opts: { value: string; label: string }[] = [];
  for (const u of accountsStore.users.value) {
    const key = `user/${u.id}`;
    if (!taken.has(key)) opts.push({ value: key, label: `用户 · ${u.displayName || u.username}` });
  }
  for (const g of accountsStore.groups.value) {
    const key = `group/${g.id}`;
    if (!taken.has(key)) opts.push({ value: key, label: `用户组 · ${g.name}` });
  }
  return opts;
});
const addSelection = ref('');

function smbHint(folder: SharedFolderView): string {
  const label = folder.name.replace(/[\s/]+/g, '_');
  const host = typeof location !== 'undefined' ? location.hostname : 'nas.local';
  return `smb://${host}/${label}`;
}
function smbService(folder: SharedFolderView) {
  return folder.services.find((s) => s.protocol === 'smb');
}
function nfsService(folder: SharedFolderView) {
  return folder.services.find((s) => s.protocol === 'nfs');
}

watch(activeFolder, (folder) => {
  editing.value = folder ? folder.permissions.map((p) => ({ ...p })) : [];
  if (folder) {
    advancedForm.value = {
      recycle: !!folder.recycle,
      quotaGb: folder.quotaBytes ? Math.round((folder.quotaBytes / (1024 * 1024 * 1024)) * 100) / 100 : 0,
      encrypted: !!folder.encrypted,
    };
    void refreshSnapshots(folder.id);
  } else {
    snapshots.value = [];
  }
}, { immediate: true });

const advancedSupported = computed(() => !!activeFolder.value?.advancedOk);
const fileSystemLabel = computed(() => (activeFolder.value?.fileSystem || '').toUpperCase());

async function refreshSnapshots(id: string) {
  try {
    snapshots.value = await sharedFoldersStore.listSnapshots(id);
  } catch {
    snapshots.value = [];
  }
}

async function saveAdvanced() {
  if (!activeFolder.value) return;
  savingAdvanced.value = true;
  try {
    const quotaBytes = Math.max(0, Math.round(advancedForm.value.quotaGb * 1024 * 1024 * 1024));
    await sharedFoldersStore.setAdvanced(activeFolder.value.id, {
      recycle: advancedForm.value.recycle,
      quotaBytes,
      encrypted: advancedForm.value.encrypted,
    });
    toast.success('高级设置已保存');
  } catch (error) {
    toast.error(`保存失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    savingAdvanced.value = false;
  }
}

async function takeSnapshot() {
  if (!activeFolder.value) return;
  snapBusy.value = true;
  try {
    await sharedFoldersStore.createSnapshot(activeFolder.value.id, snapName.value.trim() || undefined);
    snapName.value = '';
    await refreshSnapshots(activeFolder.value.id);
    toast.success('快照已创建');
  } catch (error) {
    toast.error(`创建快照失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    snapBusy.value = false;
  }
}

async function removeSnapshot(name: string) {
  if (!activeFolder.value) return;
  const ok = await confirm({
    title: `删除快照「${name}」`,
    message: '该操作不可恢复。确定删除这个快照吗?',
    tone: 'danger',
    confirmLabel: '删除',
  });
  if (!ok) return;
  try {
    await sharedFoldersStore.deleteSnapshot(activeFolder.value.id, name);
    await refreshSnapshots(activeFolder.value.id);
    toast.success('快照已删除');
  } catch (error) {
    toast.error(`删除失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

onMounted(async () => {
  await Promise.all([sharedFoldersStore.load(), accountsStore.load(), loadSpaces()]);
  if (!activeId.value && folders.value.length) activeId.value = folders.value[0].id;
});

async function loadSpaces() {
  try {
    spaces.value = await apiClient.storage.getSpaces();
  } catch {
    spaces.value = [];
  }
}

function addSubject() {
  if (!addSelection.value) return;
  const [subjectType, subjectId] = addSelection.value.split('/') as ['user' | 'group', string];
  const name = subjectType === 'user'
    ? accountsStore.users.value.find((u) => u.id === subjectId)?.displayName
    : accountsStore.groups.value.find((g) => g.id === subjectId)?.name;
  editing.value.push({ subjectType, subjectId, subjectName: name, access: 'read' });
  addSelection.value = '';
}

async function savePermissions() {
  if (!activeFolder.value) return;
  savingPerms.value = true;
  try {
    await sharedFoldersStore.setPermissions(activeFolder.value.id, editing.value);
    toast.success('权限已保存');
  } catch (error) {
    toast.error(`保存失败：${error instanceof Error ? error.message : '未知错误'}`);
  } finally {
    savingPerms.value = false;
  }
}

async function toggleService(protocol: 'smb' | 'nfs', enabled: boolean, guest = false) {
  if (!activeFolder.value) return;
  try {
    await sharedFoldersStore.setService(activeFolder.value.id, protocol, enabled, guest);
  } catch (error) {
    toast.error(`操作失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function createFolder() {
  if (!createForm.value.spaceId || !createForm.value.name.trim()) {
    toast.error('请选择存储空间并填写名称');
    return;
  }
  try {
    const view = await sharedFoldersStore.create({
      spaceId: createForm.value.spaceId,
      name: createForm.value.name.trim(),
      relPath: createForm.value.relPath.trim(),
    });
    activeId.value = view.id;
    showCreate.value = false;
    createForm.value = { spaceId: '', name: '', relPath: '' };
    toast.success('共享文件夹已创建');
  } catch (error) {
    toast.error(`创建失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

async function removeFolder(folder: SharedFolderView) {
  const ok = await confirm({
    title: `删除共享文件夹「${folder.name}」`,
    message: '将清除其全部权限授权与 SMB/NFS 共享(不删除目录数据)。确定继续吗?',
    tone: 'danger',
    confirmLabel: '删除',
  });
  if (!ok) return;
  try {
    await sharedFoldersStore.deleteFolder(folder.id, false);
    if (activeId.value === folder.id) activeId.value = folders.value[0]?.id ?? '';
    toast.success('已删除');
  } catch (error) {
    toast.error(`删除失败：${error instanceof Error ? error.message : '未知错误'}`);
  }
}

function copyHint(text: string) {
  navigator.clipboard?.writeText(text);
  toast.success('已复制');
}

const spaceOptions = computed(() => spaces.value.map((s) => ({ value: s.id, label: s.name })));
</script>

<template>
  <div class="sf-pane">
    <aside class="sf__list">
      <header class="sf__list-head">
        <strong>共享文件夹</strong>
        <UiButton size="sm" tone="primary" :icon-left="FolderPlus" @click="showCreate = true">新建</UiButton>
      </header>
      <div class="sf__list-body">
        <UiEmptyState v-if="!folders.length" :icon="FolderTree" title="还没有共享文件夹" description="新建一个,即可为用户/组配置权限并开启 SMB/NFS" />
        <section v-for="group in groups" :key="group.spaceId" class="sf__group">
          <p class="sf__group-title">{{ group.spaceName }}</p>
          <button
            v-for="folder in group.folders"
            :key="folder.id"
            class="sf__item"
            :class="{ 'sf__item--active': folder.id === activeId }"
            type="button"
            @click="activeId = folder.id"
          >
            <FolderTree :size="15" />
            <span class="sf__item-name">{{ folder.name }}</span>
            <UiBadge v-if="smbService(folder)?.enabled" tone="success" size="sm">SMB</UiBadge>
          </button>
        </section>
      </div>
    </aside>

    <section class="sf__detail">
      <template v-if="activeFolder">
        <header class="sf__detail-head">
          <div>
            <h3>{{ activeFolder.name }}</h3>
            <p class="sf__path">{{ activeFolder.absPath }}</p>
          </div>
          <UiButton variant="ghost" tone="danger" :icon-left="Trash2" size="sm" @click="removeFolder(activeFolder)">删除</UiButton>
        </header>

        <UiTabs v-model="activeTab" :tabs="tabs" variant="underline" size="sm" />

        <div v-if="activeTab === 'permissions'" class="sf__panel">
          <div class="sf__add">
            <UiSelect v-model="addSelection" :options="addOptions" placeholder="添加用户 / 用户组…" />
            <UiButton size="sm" tone="neutral" variant="soft" :disabled="!addSelection" @click="addSubject">添加</UiButton>
          </div>
          <div class="sf__perm-table">
            <div v-if="!editing.length" class="sf__empty-hint">尚未授权任何用户。添加对象并设置权限后保存。</div>
            <div v-for="(entry, i) in editing" :key="`${entry.subjectType}/${entry.subjectId}`" class="sf__perm-row">
              <span class="sf__perm-subject">
                <UiBadge :tone="entry.subjectType === 'group' ? 'info' : 'neutral'" size="sm">{{ entry.subjectType === 'group' ? '组' : '用户' }}</UiBadge>
                {{ entry.subjectName || entry.subjectId }}
              </span>
              <UiSegmented v-model="editing[i].access" :options="accessOptions" size="sm" />
            </div>
          </div>
          <div class="sf__actions">
            <UiButton tone="primary" :loading="savingPerms" :icon-left="RefreshCw" @click="savePermissions">保存权限</UiButton>
            <span class="sf__note">保存后自动写入 setfacl ACL（拒绝=`u:x:---`，覆盖组授权），并派生 SMB valid users / write list / invalid users。</span>
          </div>
        </div>

        <div v-else-if="activeTab === 'services'" class="sf__panel">
          <div class="sf__svc-row">
            <div>
              <strong>SMB(Windows/macOS 文件共享)</strong>
              <p class="sf__svc-hint">开启后,权限表中的用户即可用各自账号密码访问。</p>
            </div>
            <UiSwitch
              :model-value="smbService(activeFolder)?.enabled ?? false"
              @update:model-value="(v: boolean) => toggleService('smb', v, activeFolder!.guest)"
            />
          </div>
          <div v-if="smbService(activeFolder)?.enabled" class="sf__hint-box">
            <code>{{ smbHint(activeFolder) }}</code>
            <UiButton size="sm" variant="ghost" tone="neutral" :icon-left="Copy" @click="copyHint(smbHint(activeFolder))">复制</UiButton>
          </div>
          <div class="sf__svc-row sf__svc-row--sub">
            <span>允许访客匿名访问</span>
            <UiSwitch
              :model-value="activeFolder.guest"
              @update:model-value="(v: boolean) => toggleService('smb', smbService(activeFolder!)?.enabled ?? false, v)"
            />
          </div>
          <div class="sf__svc-row">
            <div>
              <strong>NFS(Linux/Unix 文件共享)</strong>
              <p class="sf__svc-hint">基于网络的导出,适合 Linux 客户端。</p>
            </div>
            <UiSwitch
              :model-value="nfsService(activeFolder)?.enabled ?? false"
              @update:model-value="(v: boolean) => toggleService('nfs', v)"
            />
          </div>
          <p class="sf__note">提示:还需在「共享协议」中开启全局 SMB/NFS 服务,客户端才能连接。</p>
        </div>

        <div v-else class="sf__panel">
          <div class="sf__svc-row">
            <div>
              <strong>回收站</strong>
              <p class="sf__svc-hint">通过 SMB 删除的文件先移入 <code>#recycle</code>,可找回。</p>
            </div>
            <UiSwitch v-model="advancedForm.recycle" />
          </div>

          <div class="sf__svc-row">
            <div>
              <strong>文件夹配额</strong>
              <p class="sf__svc-hint">
                限制该共享文件夹可用空间(GB,0 表示不限)。
                <template v-if="!advancedSupported">当前文件系统 {{ fileSystemLabel || '—' }} 不支持配额,仅 Btrfs 子卷可强制生效。</template>
              </p>
            </div>
            <div class="sf__quota">
              <UiInput v-model.number="advancedForm.quotaGb" type="number" :disabled="!advancedSupported" />
              <span>GB</span>
            </div>
          </div>

          <div class="sf__svc-row">
            <div>
              <strong>文件夹加密</strong>
              <p class="sf__svc-hint">标记该文件夹为加密(意图)。需主机具备加密文件系统支持方可落地。</p>
            </div>
            <UiSwitch v-model="advancedForm.encrypted" />
          </div>

          <div class="sf__actions">
            <UiButton tone="primary" :loading="savingAdvanced" :icon-left="RefreshCw" @click="saveAdvanced">保存高级设置</UiButton>
            <span class="sf__note">回收站立即写入 Samba 配置;配额在 Btrfs 子卷上经 qgroup 强制。</span>
          </div>

          <div class="sf__snap">
            <div class="sf__snap-head">
              <div>
                <strong>文件夹快照</strong>
                <p class="sf__svc-hint">
                  即时只读快照,用于回溯。
                  <template v-if="!advancedSupported">需要 Btrfs 子卷;当前 {{ fileSystemLabel || '文件系统' }} 不支持。</template>
                </p>
              </div>
            </div>
            <div class="sf__snap-create">
              <UiInput v-model="snapName" placeholder="快照名称(可选)" :disabled="!advancedSupported" />
              <UiButton size="sm" tone="neutral" variant="soft" :icon-left="Camera" :loading="snapBusy" :disabled="!advancedSupported" @click="takeSnapshot">创建快照</UiButton>
            </div>
            <div v-if="snapshots.length" class="sf__snap-list">
              <div v-for="snap in snapshots" :key="snap.name" class="sf__snap-row">
                <span class="sf__snap-name"><Camera :size="14" /> {{ snap.name }}</span>
                <span class="sf__snap-time">{{ snap.createdAt }}</span>
                <UiButton size="sm" variant="ghost" tone="danger" :icon-left="Trash" @click="removeSnapshot(snap.name)">删除</UiButton>
              </div>
            </div>
            <p v-else class="sf__empty-hint">还没有快照。</p>
          </div>
        </div>
      </template>
      <UiEmptyState v-else :icon="FolderTree" title="选择一个共享文件夹" description="或新建一个开始配置" />
    </section>

    <UiModal :open="showCreate" title="新建共享文件夹" size="sm" @update:open="showCreate = $event">
      <div class="sf__form">
        <label>存储空间
          <UiSelect v-model="createForm.spaceId" :options="spaceOptions" placeholder="选择存储空间" />
        </label>
        <label>名称
          <UiInput v-model="createForm.name" placeholder="例如 团队资料" />
        </label>
        <label>子路径(可选,留空即空间根目录)
          <UiInput v-model="createForm.relPath" placeholder="例如 docs/team" />
        </label>
      </div>
      <template #footer>
        <UiButton variant="ghost" tone="neutral" @click="showCreate = false">取消</UiButton>
        <UiButton tone="primary" @click="createFolder">创建</UiButton>
      </template>
    </UiModal>
  </div>
</template>

<style scoped>
.sf-pane { display: flex; gap: 14px; height: 100%; min-height: 0; }
.sf__list { width: 230px; flex-shrink: 0; }
.sf__detail { flex: 1; min-width: 0; }
.sf__list, .sf__detail {
  background: rgba(var(--surface-rgb), 0.5);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  min-height: 0;
  display: flex;
  flex-direction: column;
}
.sf__list-head { display: flex; align-items: center; justify-content: space-between; padding: 12px 13px; border-bottom: 1px solid var(--border); }
.sf__list-head strong { color: var(--text-strong); font-size: var(--fs-sm); }
.sf__list-body { flex: 1; min-height: 0; overflow: auto; padding: 8px; }
.sf__group-title { margin: 8px 6px 4px; color: var(--text-muted); font-size: var(--fs-2xs); }
.sf__item { display: flex; align-items: center; gap: 8px; width: 100%; padding: 8px 9px; border-radius: var(--radius-control); border: 0; background: transparent; color: var(--text-strong); font-size: var(--fs-sm); cursor: pointer; text-align: left; }
.sf__item:hover { background: rgba(var(--surface-rgb), 0.9); }
.sf__item--active { background: var(--accent-soft); color: var(--accent); }
.sf__item-name { flex: 1; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sf__detail { padding: 14px; gap: 12px; overflow: auto; }
.sf__detail-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 12px; }
.sf__detail-head h3 { margin: 0; color: var(--text-strong); font-size: var(--fs-lg); }
.sf__path { margin: 4px 0 0; color: var(--text-muted); font-size: var(--fs-2xs); font-family: var(--font-mono, monospace); word-break: break-all; }
.sf__panel { display: flex; flex-direction: column; gap: 12px; }
.sf__add { display: flex; gap: 8px; align-items: center; }
.sf__add :deep(.ui-select) { flex: 1; }
.sf__perm-table { display: flex; flex-direction: column; gap: 6px; }
.sf__perm-row { display: flex; align-items: center; justify-content: space-between; gap: 12px; padding: 8px 10px; background: rgba(var(--surface-rgb), 0.6); border: 1px solid var(--border); border-radius: var(--radius-control); }
.sf__perm-subject { display: inline-flex; align-items: center; gap: 8px; color: var(--text-strong); font-size: var(--fs-sm); }
.sf__empty-hint, .sf__note, .sf__svc-hint { color: var(--text-muted); font-size: var(--fs-xs); }
.sf__actions { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.sf__svc-row { display: flex; align-items: center; justify-content: space-between; gap: 16px; padding: 10px 12px; background: rgba(var(--surface-rgb), 0.6); border: 1px solid var(--border); border-radius: var(--radius-card); }
.sf__svc-row--sub { padding: 8px 12px; }
.sf__svc-row strong { color: var(--text-strong); font-size: var(--fs-sm); }
.sf__svc-hint { margin: 4px 0 0; }
.sf__hint-box { display: flex; align-items: center; justify-content: space-between; gap: 10px; padding: 8px 12px; background: var(--accent-soft); border-radius: var(--radius-control); }
.sf__hint-box code { color: var(--accent); font-size: var(--fs-xs); }
.sf__form { display: flex; flex-direction: column; gap: 12px; }
.sf__form label { display: flex; flex-direction: column; gap: 6px; color: var(--text-muted); font-size: var(--fs-xs); }
.sf__svc-hint code { font-family: var(--font-mono, monospace); color: var(--accent); }
.sf__quota { display: inline-flex; align-items: center; gap: 8px; color: var(--text-muted); font-size: var(--fs-xs); }
.sf__quota :deep(.ui-input) { width: 96px; }
.sf__snap { display: flex; flex-direction: column; gap: 10px; padding: 12px; background: rgba(var(--surface-rgb), 0.6); border: 1px solid var(--border); border-radius: var(--radius-card); }
.sf__snap-head strong { color: var(--text-strong); font-size: var(--fs-sm); }
.sf__snap-create { display: flex; align-items: center; gap: 8px; }
.sf__snap-create :deep(.ui-input) { flex: 1; }
.sf__snap-list { display: flex; flex-direction: column; gap: 6px; }
.sf__snap-row { display: flex; align-items: center; gap: 12px; padding: 7px 10px; background: rgba(var(--surface-rgb), 0.85); border: 1px solid var(--border); border-radius: var(--radius-control); }
.sf__snap-name { display: inline-flex; align-items: center; gap: 8px; flex: 1; color: var(--text-strong); font-size: var(--fs-sm); }
.sf__snap-time { color: var(--text-muted); font-size: var(--fs-2xs); font-family: var(--font-mono, monospace); }
</style>
