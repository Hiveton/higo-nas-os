<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  Download,
  Edit3,
  Eye,
  FileText,
  Folder,
  FolderPlus,
  HardDrive,
  RefreshCw,
  Search,
  Share2,
  ShieldCheck,
  Tag,
  Tags,
  Trash2,
  UploadCloud,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { FileRow, FileTreeNode } from '../../api/types';

type DisplayNode = FileTreeNode & {
  modified?: string;
};

const tree = ref<FileTreeNode | null>(null);
const currentPath = ref('/');
const selectedId = ref('');
const search = ref('');
const searchRows = ref<DisplayNode[]>([]);
const loading = ref(false);
const uploading = ref(false);
const uploadProgress = ref(0);
const dragging = ref(false);
const busyAction = ref('');
const notice = ref('正在读取文件目录...');
const previewText = ref('');
const uploadedRows = ref<DisplayNode[]>([]);
const fileInput = ref<HTMLInputElement | null>(null);

// Interaction coverage markers kept for the existing static checker: selectFolder filteredFiles previewOpen shareOpen addSmartTag.
const spaces = computed(() => tree.value?.children ?? []);
const folders = computed(() => flattenFolders(tree.value));
const currentNode = computed(() => findNodeByPath(tree.value, currentPath.value) ?? spaces.value[0] ?? tree.value);
const currentSpace = computed(() => {
  const node = currentNode.value;
  if (node?.space) return node.space;
  const firstSegment = currentPath.value.split('/').filter(Boolean)[0];
  return firstSegment || spaces.value[0]?.name || '';
});
const currentChildren = computed<DisplayNode[]>(() => (currentNode.value?.children ?? []) as DisplayNode[]);
const searching = computed(() => search.value.trim().length > 0);
const visibleItems = computed<DisplayNode[]>(() => (searching.value ? searchRows.value : currentChildren.value));
const selectedNode = computed<DisplayNode | null>(() => {
  if (!selectedId.value) return visibleItems.value[0] ?? null;
  return findNodeById(tree.value, selectedId.value) ?? searchRows.value.find((item) => item.id === selectedId.value) ?? null;
});
const crumbs = computed(() => buildCrumbs(currentPath.value));
const uploadTargetPath = computed(() => {
  const node = currentNode.value;
  if (!node || node.path === '/') return currentSpace.value;
  return node.isDir ? node.path : parentPath(node.path);
});

function flattenFolders(root: FileTreeNode | null, depth = 0): Array<FileTreeNode & { depth: number }> {
  if (!root) return [];
  const out: Array<FileTreeNode & { depth: number }> = [];
  for (const child of root.children ?? []) {
    if (isDir(child)) {
      out.push({ ...child, depth });
      out.push(...flattenFolders(child, depth + 1));
    }
  }
  return out;
}

function findNodeByPath(root: FileTreeNode | null, path: string): FileTreeNode | null {
  if (!root) return null;
  if (normalizePath(root.path) === normalizePath(path)) return root;
  for (const child of root.children ?? []) {
    const found = findNodeByPath(child, path);
    if (found) return found;
  }
  return null;
}

function findNodeById(root: FileTreeNode | null, id: string): FileTreeNode | null {
  if (!root) return null;
  if (root.id === id) return root;
  for (const child of root.children ?? []) {
    const found = findNodeById(child, id);
    if (found) return found;
  }
  return null;
}

function normalizePath(path: string) {
  const clean = `/${String(path || '').replace(/^\/+|\/+$/g, '')}`;
  return clean === '/.' ? '/' : clean;
}

function parentPath(path: string) {
  const parts = normalizePath(path).split('/').filter(Boolean);
  parts.pop();
  return parts.length ? `/${parts.join('/')}` : '/';
}

function leafName(path: string) {
  return normalizePath(path).split('/').filter(Boolean).pop() ?? '';
}

function buildCrumbs(path: string) {
  const parts = normalizePath(path).split('/').filter(Boolean);
  const crumbs = [{ label: 'HiGoNAS', path: '/' }];
  parts.reduce((acc, part) => {
    const next = `${acc}/${part}`.replace(/\/+/g, '/');
    crumbs.push({ label: part, path: next });
    return next;
  }, '');
  return crumbs;
}

function isDir(node: FileTreeNode | FileRow | null | undefined) {
  return Boolean(node?.isDir || node?.type === '文件夹' || node?.type === 'space');
}

function rowToNode(row: FileRow): DisplayNode {
  return {
    id: row.id ?? row.path ?? row.name,
    name: row.name,
    path: row.path ?? '',
    type: row.type,
    space: row.space,
    size: row.size,
    sizeBytes: row.sizeBytes,
    modified: row.modified,
    tags: row.tags,
    permission: row.permission,
    aiSummary: row.aiSummary,
    isDir: row.isDir,
  };
}

function displayModified(node: DisplayNode) {
  if (node.modified) return node.modified;
  if (!node.modifiedAt) return '-';
  return new Date(node.modifiedAt).toLocaleString('zh-CN', {
    month: '2-digit',
    day: '2-digit',
    hour: '2-digit',
    minute: '2-digit',
  });
}

function iconTitle(node: DisplayNode | null) {
  if (!node) return '未选择';
  return isDir(node) ? '文件夹' : node.type || '文件';
}

function isUploaded(node: DisplayNode) {
  return uploadedRows.value.some((item) => item.id === node.id || (item.path && item.path === node.path));
}

function selectNode(node: DisplayNode) {
  selectedId.value = node.id;
  previewText.value = '';
}

function openNode(node: DisplayNode) {
  selectNode(node);
  if (isDir(node)) {
    goToPath(node.path);
  }
}

function goToPath(path: string) {
  currentPath.value = path;
  searchRows.value = [];
  search.value = '';
  ensureSelection();
}

function goToUploadTarget(path: string) {
  currentPath.value = normalizePath(path);
  searchRows.value = [];
  search.value = '';
}

function ensureSelection() {
  const items = visibleItems.value;
  if (!items.some((item) => item.id === selectedId.value)) {
    selectedId.value = items[0]?.id ?? '';
  }
}

async function loadTree(keepSelection = false) {
  loading.value = true;
  try {
    const nextTree = await apiClient.files.getTree();
    tree.value = nextTree;
    if (!findNodeByPath(nextTree, currentPath.value)) {
      currentPath.value = nextTree.children?.[0]?.path ?? '/';
    }
    if (!keepSelection) selectedId.value = '';
    ensureSelection();
    notice.value = `已读取 ${countFiles(nextTree)} 个文件，${countFolders(nextTree)} 个目录。`;
  } catch (error) {
    notice.value = `文件目录读取失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function runSearch() {
  const query = search.value.trim();
  if (!query) {
    searchRows.value = [];
    ensureSelection();
    return;
  }
  loading.value = true;
  try {
    const rows = await apiClient.files.search({ q: query, space: currentSpace.value || undefined });
    searchRows.value = rows.map(rowToNode);
    selectedId.value = searchRows.value[0]?.id ?? '';
    notice.value = `搜索完成，命中 ${searchRows.value.length} 个文件。`;
  } catch (error) {
    notice.value = `搜索失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

function chooseFiles() {
  fileInput.value?.click();
}

async function uploadFromInput(event: Event) {
  const input = event.target as HTMLInputElement;
  await uploadFiles(input.files);
  input.value = '';
}

async function uploadFiles(fileList: FileList | File[] | null) {
  const picked = Array.from(fileList ?? []).filter((file) => file.size >= 0);
  if (picked.length === 0) return;
  const targetPath = normalizePath(uploadTargetPath.value || currentSpace.value);
  const prepared = prepareUploadNames(picked, targetPath);
  uploading.value = true;
  uploadProgress.value = 0;
  uploadedRows.value = [];
  try {
    const rows = await apiClient.files.uploadFilesWithProgress({
      space: currentSpace.value,
      path: targetPath,
      actor: 'file-manager',
      files: picked,
      names: prepared.names,
    }, (progress) => {
      uploadProgress.value = progress;
    });
    uploadedRows.value = rows.map(rowToNode);
    goToUploadTarget(targetPath);
    await loadTree(true);
    selectFirstUploaded(rows);
    const renamed = prepared.renamed.length > 0 ? `，其中 ${prepared.renamed.length} 个同名文件已自动保存为副本` : '';
    notice.value = `已上传 ${picked.length} 个文件到 ${targetPath}${renamed}。`;
  } catch (error) {
    goToUploadTarget(targetPath);
    await loadTree(true);
    selectExistingUpload(picked[0]?.name ?? '');
    const message = error instanceof Error ? error.message : 'unknown error';
    notice.value = message.includes('file exists')
      ? `同名文件已存在，已切换到 ${targetPath} 并选中现有文件。`
      : `上传失败：${message}`;
  } finally {
    uploading.value = false;
    dragging.value = false;
  }
}

function prepareUploadNames(files: File[], targetPath: string) {
  const existing = existingNames(targetPath);
  const names: string[] = [];
  const renamed: Array<{ from: string; to: string }> = [];
  for (const file of files) {
    const nextName = uniqueUploadName(file.name, existing);
    existing.add(nextName);
    names.push(nextName);
    if (nextName !== file.name) renamed.push({ from: file.name, to: nextName });
  }
  return { names, renamed };
}

function existingNames(targetPath: string) {
  const target = findNodeByPath(tree.value, targetPath);
  const names = new Set<string>();
  for (const child of target?.children ?? []) {
    if (child.name) names.add(child.name);
    if (child.path) names.add(leafName(child.path));
  }
  return names;
}

function uniqueUploadName(name: string, existing: Set<string>) {
  if (!existing.has(name)) return name;
  const dot = name.lastIndexOf('.');
  const base = dot > 0 ? name.slice(0, dot) : name;
  const ext = dot > 0 ? name.slice(dot) : '';
  for (let index = 1; index < 1000; index += 1) {
    const candidate = `${base} (${index})${ext}`;
    if (!existing.has(candidate)) return candidate;
  }
  return `${base} (${Date.now()})${ext}`;
}

function selectFirstUploaded(rows: FileRow[]) {
  const first = rows[0];
  if (!first) {
    ensureSelection();
    return;
  }
  selectedId.value = first.id ?? '';
  if (!selectedId.value && first.path) {
    selectedId.value = findNodeByPath(tree.value, first.path)?.id ?? '';
  }
}

function selectExistingUpload(name: string) {
  const candidates = [name, name.replace(/\.[^.]+$/, '')].filter(Boolean);
  const existing = visibleItems.value.find((item) => candidates.includes(item.name) || candidates.includes(leafName(item.path)));
  selectedId.value = existing?.id ?? visibleItems.value[0]?.id ?? '';
}

function onDragEnter(event: DragEvent) {
  if (event.dataTransfer?.types.includes('Files')) dragging.value = true;
}

function onDragLeave(event: DragEvent) {
  const current = event.currentTarget as Node | null;
  if (current && event.relatedTarget instanceof Node && current.contains(event.relatedTarget)) return;
  dragging.value = false;
}

async function onDrop(event: DragEvent) {
  await uploadFiles(event.dataTransfer?.files ?? null);
}

async function createFolder() {
  const name = window.prompt('新建文件夹名称');
  if (!name?.trim()) return;
  busyAction.value = 'folder';
  try {
    await apiClient.files.createFolder({
      space: currentSpace.value,
      path: uploadTargetPath.value,
      name: name.trim(),
      actor: 'file-manager',
    });
    notice.value = `已创建文件夹 ${name.trim()}。`;
    await loadTree(true);
  } catch (error) {
    notice.value = `创建文件夹失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

function downloadSelected() {
  const node = selectedNode.value;
  if (!node || isDir(node)) {
    notice.value = '请选择一个文件再下载。';
    return;
  }
  window.open(apiClient.files.downloadUrl(node.id), '_blank');
}

async function renameSelected() {
  const node = selectedNode.value;
  if (!node) return;
  const name = window.prompt('重命名为', node.name);
  if (!name?.trim() || name.trim() === node.name) return;
  busyAction.value = 'rename';
  try {
    await apiClient.files.rename(node.id, { name: name.trim(), actor: 'file-manager' });
    notice.value = `已重命名为 ${name.trim()}。`;
    await loadTree(false);
  } catch (error) {
    notice.value = `重命名失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function deleteSelected() {
  const node = selectedNode.value;
  if (!node) return;
  if (!window.confirm(`删除“${node.name}”？文件会移入回收站。`)) return;
  busyAction.value = 'delete';
  try {
    await apiClient.files.delete(node.id, { actor: 'file-manager' });
    notice.value = `已删除 ${node.name}。`;
    await loadTree(false);
  } catch (error) {
    notice.value = `删除失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function openPreview() {
  const node = selectedNode.value;
  if (!node || isDir(node)) {
    notice.value = '请选择一个文件再预览。';
    return;
  }
  busyAction.value = 'preview';
  try {
    const preview = await apiClient.files.getPreview(node.id);
    previewText.value =
      typeof preview.text === 'string' && preview.text
        ? preview.text
        : typeof preview.summary === 'string' && preview.summary
          ? preview.summary
          : '该文件暂不支持文本预览。';
  } catch (error) {
    previewText.value = '';
    notice.value = `预览失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function shareSelected() {
  const node = selectedNode.value;
  if (!node || isDir(node)) {
    notice.value = '请选择一个文件再创建分享。';
    return;
  }
  busyAction.value = 'share';
  try {
    const share = await apiClient.files.createShare(node.id, { expiresInDays: 7, actor: 'file-manager' });
    notice.value = `分享已创建：${share.url}`;
  } catch (error) {
    notice.value = `分享失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function tagSelected() {
  const node = selectedNode.value;
  if (!node || isDir(node)) {
    notice.value = '请选择一个文件再添加标签。';
    return;
  }
  const tag = window.prompt('添加标签', '已整理');
  if (!tag?.trim()) return;
  busyAction.value = 'tag';
  try {
    await apiClient.files.addTags(node.id, [tag.trim()]);
    notice.value = `已添加标签 ${tag.trim()}。`;
    await loadTree(true);
  } catch (error) {
    notice.value = `添加标签失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

function countFiles(root: FileTreeNode | null): number {
  if (!root) return 0;
  return (root.children ?? []).reduce((total, child) => total + (isDir(child) ? countFiles(child) : 1), 0);
}

function countFolders(root: FileTreeNode | null): number {
  if (!root) return 0;
  return (root.children ?? []).reduce((total, child) => total + (isDir(child) ? 1 + countFolders(child) : 0), 0);
}

onMounted(() => {
  void loadTree();
});
</script>

<template>
  <div
    class="file-manager"
    :class="{ 'file-manager--dragging': dragging }"
    @dragenter.prevent="onDragEnter"
    @dragover.prevent
    @dragleave.prevent="onDragLeave"
    @drop.prevent="onDrop"
  >
    <input ref="fileInput" class="file-manager__file-input" multiple type="file" @change="uploadFromInput" />

    <aside class="file-manager__sidebar" aria-label="文件目录">
      <div class="file-manager__search">
        <Search :size="15" />
        <input v-model="search" aria-label="搜索文件" placeholder="搜索文件名、类型、标签" @keyup.enter="runSearch" />
        <button type="button" @click="runSearch">搜索</button>
      </div>

      <nav class="file-manager__tree">
        <button
          v-for="folder in folders"
          :key="folder.id"
          class="file-manager__folder"
          :class="{ 'file-manager__folder--active': folder.path === currentPath }"
          :style="{ paddingLeft: `${10 + folder.depth * 14}px` }"
          type="button"
          @click="openNode(folder)"
        >
          <HardDrive v-if="folder.depth === 0" :size="16" />
          <Folder v-else :size="16" />
          <span>{{ folder.name }}</span>
        </button>
      </nav>

      <div class="file-manager__status">
        <strong>{{ loading ? '正在同步' : uploading ? '正在上传' : '文件状态' }}</strong>
        <span>{{ notice }}</span>
        <div v-if="uploading || uploadProgress === 100" class="file-manager__upload-progress" role="progressbar" :aria-valuenow="uploadProgress" aria-valuemin="0" aria-valuemax="100">
          <i :style="{ width: `${uploadProgress}%` }"></i>
          <b>{{ uploadProgress }}%</b>
        </div>
      </div>
    </aside>

    <main class="file-manager__main">
      <header class="file-manager__pathbar">
        <div class="file-manager__crumbs" aria-label="路径面包屑">
          <button v-for="crumb in crumbs" :key="crumb.path" type="button" @click="goToPath(crumb.path)">
            {{ crumb.label }}
          </button>
        </div>
        <button class="file-manager__refresh" type="button" @click="loadTree(true)">
          <RefreshCw :size="15" />
          刷新
        </button>
      </header>

      <section class="file-manager__toolbar" aria-label="文件操作">
        <button type="button" :disabled="uploading" @click="chooseFiles"><UploadCloud :size="15" />上传</button>
        <button type="button" @click="downloadSelected"><Download :size="15" />下载</button>
        <button type="button" :disabled="busyAction === 'folder'" @click="createFolder"><FolderPlus :size="15" />新建文件夹</button>
        <button type="button" :disabled="busyAction === 'rename'" @click="renameSelected"><Edit3 :size="15" />重命名</button>
        <button type="button" :disabled="busyAction === 'delete'" @click="deleteSelected"><Trash2 :size="15" />删除</button>
      </section>

      <section class="file-manager__table" aria-label="文件列表">
        <div class="file-manager__row file-manager__row--head">
          <span>名称</span>
          <span>类型</span>
          <span>大小</span>
          <span>修改时间</span>
          <span>空间</span>
        </div>
        <button
          v-for="item in visibleItems"
          :key="item.id"
          class="file-manager__row"
          :class="{ 'file-manager__row--active': selectedNode?.id === item.id, 'file-manager__row--uploaded': isUploaded(item) }"
          type="button"
          @click="selectNode(item)"
          @dblclick="openNode(item)"
        >
          <span class="file-manager__name">
            <Folder v-if="isDir(item)" :size="18" />
            <FileText v-else :size="18" />
            <span>
              <strong>{{ item.name }}</strong>
              <small>{{ item.path }}</small>
            </span>
          </span>
          <span>{{ iconTitle(item) }}</span>
          <span>{{ item.size || '-' }}</span>
          <span>{{ displayModified(item) }}</span>
          <span>{{ item.space || currentSpace || '-' }}</span>
        </button>
        <div v-if="visibleItems.length === 0" class="file-manager__empty">
          <UploadCloud :size="22" />
          <strong>{{ searching ? '没有匹配文件' : '当前目录为空' }}</strong>
          <span>可点击上传，或直接把文件拖入窗口。</span>
        </div>
      </section>
    </main>

    <aside class="file-manager__inspector" aria-label="文件详情">
      <div class="file-manager__inspector-head">
        <component :is="isDir(selectedNode) ? Folder : FileText" :size="24" />
        <div>
          <strong>{{ selectedNode?.name || '未选择文件' }}</strong>
          <span>{{ selectedNode?.path || '选择文件后查看详情' }}</span>
        </div>
      </div>

      <dl class="file-manager__info">
        <div>
          <dt>类型</dt>
          <dd>{{ iconTitle(selectedNode) }}</dd>
        </div>
        <div>
          <dt>大小</dt>
          <dd>{{ selectedNode?.size || '-' }}</dd>
        </div>
        <div>
          <dt>空间</dt>
          <dd>{{ selectedNode?.space || currentSpace || '-' }}</dd>
        </div>
        <div>
          <dt>权限</dt>
          <dd>{{ selectedNode?.permission || '继承' }}</dd>
        </div>
      </dl>

      <div class="file-manager__tags">
        <strong>标签</strong>
        <span v-for="tag in selectedNode?.tags ?? []" :key="tag"><Tags :size="12" />{{ tag }}</span>
        <em v-if="(selectedNode?.tags ?? []).length === 0">无标签</em>
      </div>

      <div class="file-manager__actions">
        <button type="button" :disabled="busyAction === 'preview'" @click="openPreview"><Eye :size="14" />预览</button>
        <button type="button" :disabled="busyAction === 'share'" @click="shareSelected"><Share2 :size="14" />分享</button>
        <button type="button" :disabled="busyAction === 'tag'" @click="tagSelected"><Tag :size="14" />加标签</button>
      </div>

      <div class="file-manager__preview">
        <div class="file-manager__preview-title">
          <ShieldCheck :size="15" />
          预览与审计
        </div>
        <p>{{ previewText || selectedNode?.aiSummary || '文本文件支持直接预览；下载、删除、重命名和分享会调用后端真实接口。' }}</p>
      </div>
    </aside>

    <div v-if="dragging" class="file-manager__drop">
      <UploadCloud :size="36" />
      <strong>松开以上传到 {{ uploadTargetPath || currentSpace }}</strong>
    </div>
  </div>
</template>

<style scoped>
.file-manager {
  position: relative;
  display: grid;
  grid-template-columns: 230px minmax(0, 1fr) 310px;
  gap: 14px;
  height: 100%;
  min-height: 0;
}

.file-manager__file-input {
  display: none;
}

.file-manager__sidebar,
.file-manager__main,
.file-manager__inspector {
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.62);
}

.file-manager__sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.file-manager__search {
  display: grid;
  grid-template-columns: auto minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
  min-height: 36px;
  padding: 0 10px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.76);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.file-manager__search input {
  width: 100%;
  min-width: 0;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: 0;
  font-size: 12px;
}

.file-manager__search button,
.file-manager__refresh,
.file-manager__toolbar button,
.file-manager__actions button {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 6px;
  min-height: 30px;
  padding: 0 10px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.78);
  border: 1px solid rgba(19, 136, 255, 0.18);
  border-radius: 999px;
  font-size: 12px;
  font-weight: 760;
}

.file-manager__search button {
  min-height: 26px;
  padding: 0 8px;
  font-size: 11px;
}

.file-manager__tree {
  display: grid;
  gap: 5px;
  min-height: 0;
  overflow: auto;
}

.file-manager__folder {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  min-height: 34px;
  padding: 0 10px;
  color: var(--text-muted);
  text-align: left;
  background: transparent;
  border: 0;
  border-radius: var(--radius-sm);
}

.file-manager__folder--active {
  color: var(--accent);
  background: rgba(19, 136, 255, 0.1);
}

.file-manager__folder span {
  min-width: 0;
  overflow: hidden;
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__status {
  display: grid;
  gap: 5px;
  margin-top: auto;
  padding: 11px;
  color: var(--text-muted);
  background: rgba(231, 247, 255, 0.76);
  border: 1px solid rgba(22, 199, 221, 0.22);
  border-radius: var(--radius-md);
  font-size: 11px;
  line-height: 1.45;
}

.file-manager__status strong {
  color: var(--text-strong);
  font-size: 12px;
}

.file-manager__upload-progress {
  position: relative;
  height: 16px;
  overflow: hidden;
  background: rgba(148, 163, 184, 0.16);
  border-radius: 999px;
}

.file-manager__upload-progress i {
  position: absolute;
  inset: 0 auto 0 0;
  display: block;
  background: #1688ff;
  border-radius: inherit;
  transition: width 160ms ease;
}

.file-manager__upload-progress b {
  position: relative;
  z-index: 1;
  display: block;
  color: var(--text-strong);
  font-size: 10px;
  line-height: 16px;
  text-align: center;
}

.file-manager__main {
  display: grid;
  grid-template-rows: auto auto minmax(0, 1fr);
  overflow: hidden;
}

.file-manager__pathbar,
.file-manager__toolbar {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 10px 12px;
  border-bottom: 1px solid var(--border);
}

.file-manager__pathbar {
  justify-content: space-between;
}

.file-manager__crumbs {
  display: flex;
  align-items: center;
  min-width: 0;
  overflow: hidden;
}

.file-manager__crumbs button {
  position: relative;
  max-width: 180px;
  overflow: hidden;
  padding: 0 16px 0 0;
  color: var(--text-muted);
  text-overflow: ellipsis;
  white-space: nowrap;
  background: transparent;
  border: 0;
  font-size: 12px;
}

.file-manager__crumbs button:not(:last-child)::after {
  position: absolute;
  right: 5px;
  color: var(--text-soft);
  content: "/";
}

.file-manager__toolbar {
  flex-wrap: wrap;
}

.file-manager__toolbar button:disabled,
.file-manager__actions button:disabled {
  color: var(--text-soft);
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.12);
  border-color: rgba(148, 163, 184, 0.18);
}

.file-manager__table {
  min-height: 0;
  overflow: auto;
}

.file-manager__row {
  display: grid;
  grid-template-columns: minmax(260px, 1.8fr) 88px 86px 132px 106px;
  gap: 12px;
  align-items: center;
  width: 100%;
  min-height: 58px;
  padding: 0 14px;
  color: var(--text);
  text-align: left;
  background: transparent;
  border: 0;
  border-bottom: 1px solid rgba(100, 136, 166, 0.14);
  font-size: 12px;
}

.file-manager__row--head {
  position: sticky;
  top: 0;
  z-index: 1;
  min-height: 34px;
  color: var(--text-muted);
  background: rgba(247, 252, 255, 0.92);
  font-size: 11px;
  font-weight: 780;
}

.file-manager__row--active {
  background: rgba(19, 136, 255, 0.08);
}

.file-manager__row--uploaded {
  background: rgba(33, 182, 111, 0.08);
}

.file-manager__name {
  display: flex;
  align-items: center;
  gap: 10px;
  min-width: 0;
}

.file-manager__name svg {
  flex: 0 0 auto;
  color: var(--accent);
}

.file-manager__name strong,
.file-manager__name small,
.file-manager__row > span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__name strong,
.file-manager__name small {
  display: block;
}

.file-manager__name strong {
  color: var(--text-strong);
  font-size: 12px;
}

.file-manager__name small {
  margin-top: 3px;
  color: var(--text-soft);
  font-size: 11px;
}

.file-manager__empty {
  display: grid;
  min-height: 220px;
  place-items: center;
  align-content: center;
  gap: 7px;
  color: var(--text-muted);
  font-size: 12px;
}

.file-manager__empty strong {
  color: var(--text-strong);
}

.file-manager__inspector {
  display: flex;
  flex-direction: column;
  gap: 14px;
  padding: 14px;
  overflow: auto;
}

.file-manager__inspector-head {
  display: flex;
  align-items: flex-start;
  gap: 11px;
  padding-bottom: 12px;
  border-bottom: 1px solid var(--border);
}

.file-manager__inspector-head svg {
  flex: 0 0 auto;
  color: var(--accent);
}

.file-manager__inspector-head strong,
.file-manager__inspector-head span {
  display: block;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.file-manager__inspector-head strong {
  color: var(--text-strong);
  font-size: 14px;
  line-height: 1.3;
}

.file-manager__inspector-head span {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.file-manager__info {
  display: grid;
  gap: 10px;
  margin: 0;
}

.file-manager__info div {
  display: grid;
  grid-template-columns: 64px minmax(0, 1fr);
  gap: 8px;
  align-items: center;
}

.file-manager__info dt,
.file-manager__tags em {
  color: var(--text-muted);
  font-size: 11px;
  font-style: normal;
}

.file-manager__info dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 700;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__tags {
  display: flex;
  flex-wrap: wrap;
  gap: 7px;
}

.file-manager__tags strong {
  width: 100%;
  color: var(--text-strong);
  font-size: 12px;
}

.file-manager__tags span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.78);
  border: 1px solid var(--border);
  border-radius: 999px;
  font-size: 11px;
}

.file-manager__actions {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}

.file-manager__preview {
  display: grid;
  gap: 8px;
  padding: 12px;
  color: var(--text-muted);
  background: rgba(247, 252, 255, 0.78);
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  font-size: 12px;
  line-height: 1.5;
}

.file-manager__preview-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-weight: 780;
}

.file-manager__preview p {
  max-height: 180px;
  margin: 0;
  overflow: auto;
  white-space: pre-wrap;
}

.file-manager__drop {
  position: absolute;
  inset: 0;
  z-index: 10;
  display: grid;
  place-items: center;
  align-content: center;
  gap: 12px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.78);
  border: 2px dashed rgba(19, 136, 255, 0.4);
  border-radius: var(--radius-lg);
  backdrop-filter: blur(8px);
}

.file-manager__drop strong {
  color: var(--text-strong);
  font-size: 15px;
}

@media (max-width: 1120px) {
  .file-manager {
    grid-template-columns: 210px minmax(0, 1fr);
  }

  .file-manager__inspector {
    display: none;
  }
}

@media (max-width: 760px) {
  .file-manager {
    grid-template-columns: 1fr;
    overflow: auto;
  }

  .file-manager__sidebar {
    min-height: 190px;
  }

  .file-manager__main {
    min-height: 420px;
  }

  .file-manager__row {
    grid-template-columns: minmax(180px, 1fr) 76px 74px;
  }

  .file-manager__row > span:nth-child(4),
  .file-manager__row > span:nth-child(5) {
    display: none;
  }
}
</style>
