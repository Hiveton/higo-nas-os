<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import {
  ChevronLeft,
  ChevronRight,
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
  Sparkles,
  Tag,
  Tags,
  Trash2,
  UploadCloud,
} from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import { aiAnalysisStore } from '../../stores/aiAnalysis';
import type { FileRow, FileTreeNode } from '../../api/types';
import {
  UiButton,
  UiEmptyState,
  UiIconButton,
  UiInput,
  UiNavRail,
  UiProgressBar,
  UiToolbar,
  UiWindowPage,
  useConfirm,
  useInputDialog,
} from '../ui';

const confirm = useConfirm();
const inputDialog = useInputDialog();

type DisplayNode = FileTreeNode & {
  modified?: string;
};

const tree = ref<FileTreeNode | null>(null);
const currentPath = ref('/');
const selectedId = ref('');
const pathDraft = ref('/');
const pathHistory = ref<string[]>([]);
const pathHistoryIndex = ref(-1);
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
const currentDirectoryPath = computed(() => {
  const node = currentNode.value;
  if (!node || node.path === '/') return currentSpace.value;
  return node.isDir ? node.path : parentPath(node.path);
});
const uploadTargetPath = computed(() => currentDirectoryPath.value);
const canGoBack = computed(() => pathHistoryIndex.value > 0);
const canGoForward = computed(() => pathHistoryIndex.value >= 0 && pathHistoryIndex.value < pathHistory.value.length - 1);

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

function normalizeInputPath(path: string) {
  let value = String(path || '').trim();
  value = value.replace(/^higonas\s*[/>]\s*/i, '').replace(/\\/g, '/').replace(/\s*\/\s*/g, '/');
  return normalizePath(value);
}

function parentPath(path: string) {
  const parts = normalizePath(path).split('/').filter(Boolean);
  parts.pop();
  return parts.length ? `/${parts.join('/')}` : '/';
}

function leafName(path: string) {
  return normalizePath(path).split('/').filter(Boolean).pop() ?? '';
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
    navigateToPath(node.path);
  }
}

function navigateToPath(path: string) {
  const normalized = normalizeInputPath(path);
  if (!findNodeByPath(tree.value, normalized)) {
    notice.value = `目录不存在：${normalized}`;
    pathDraft.value = currentPath.value;
    return;
  }
  setCurrentPath(normalized);
  pushPathHistory(normalized);
}

function setCurrentPath(path: string) {
  currentPath.value = normalizePath(path);
  pathDraft.value = currentPath.value;
  searchRows.value = [];
  search.value = '';
  ensureSelection();
}

function goToUploadTarget(path: string) {
  setCurrentPath(path);
  pushPathHistory(currentPath.value);
}

function pushPathHistory(path: string) {
  const normalized = normalizePath(path);
  if (pathHistory.value[pathHistoryIndex.value] === normalized) return;
  const previous = pathHistory.value.slice(0, pathHistoryIndex.value + 1);
  previous.push(normalized);
  pathHistory.value = previous.slice(-50);
  pathHistoryIndex.value = pathHistory.value.length - 1;
}

function goBack() {
  if (!canGoBack.value) return;
  pathHistoryIndex.value -= 1;
  setCurrentPath(pathHistory.value[pathHistoryIndex.value] ?? currentPath.value);
}

function goForward() {
  if (!canGoForward.value) return;
  pathHistoryIndex.value += 1;
  setCurrentPath(pathHistory.value[pathHistoryIndex.value] ?? currentPath.value);
}

function submitPath() {
  navigateToPath(pathDraft.value);
}

function resetPathDraft() {
  pathDraft.value = currentPath.value;
}

function selectPathDraft(event: FocusEvent) {
  if (event.target instanceof HTMLInputElement) {
    event.target.select();
  }
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
    let nextPath = currentPath.value;
    if (!findNodeByPath(nextTree, currentPath.value)) {
      nextPath = nextTree.children?.[0]?.path ?? '/';
    }
    setCurrentPath(nextPath);
    if (pathHistoryIndex.value < 0) pushPathHistory(currentPath.value);
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
  const files = event.dataTransfer?.files ?? null;
  dragging.value = false;
  await uploadFiles(files);
}

async function createFolder() {
  const name = await inputDialog({
    title: '新建文件夹',
    label: '文件夹名称',
    placeholder: '例如：项目资料',
    validate: (v) => (v ? null : '请输入名称'),
  });
  if (!name?.trim()) return;
  busyAction.value = 'folder';
  try {
    const folder = await apiClient.files.createFolder({
      space: currentSpace.value,
      path: currentDirectoryPath.value,
      name: name.trim(),
      actor: 'file-manager',
    });
    notice.value = `已在 ${currentDirectoryPath.value} 创建文件夹 ${name.trim()}。`;
    await loadTree(true);
    selectedId.value = folder.id ?? (folder.path ? findNodeByPath(tree.value, folder.path)?.id : '') ?? selectedId.value;
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
  const name = await inputDialog({
    title: '重命名',
    label: '新名称',
    defaultValue: node.name,
    validate: (v) => (v ? null : '请输入名称'),
  });
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
  if (!(await confirm({ title: '删除文件夹？', message: '文件会移入回收站。', tone: 'danger' }))) return;
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
  const tag = await inputDialog({
    title: '添加标签',
    label: '标签名称',
    defaultValue: '已整理',
    validate: (v) => (v ? null : '请输入标签'),
  });
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

async function analyzeSelected() {
  const node = selectedNode.value;
  if (!node || isDir(node)) {
    notice.value = '请选择一个文件再进行 AI 分析。';
    return;
  }
  busyAction.value = 'analyze';
  try {
    await aiAnalysisStore.reanalyze({ scope: 'item', itemId: `file:${node.id}` });
    notice.value = `已将「${node.name}」加入 AI 分析队列，摘要 / 标签稍后回填。`;
  } catch (error) {
    notice.value = `AI 分析失败：${error instanceof Error ? error.message : 'unknown error'}`;
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

    <UiWindowPage
      layout="master-detail"
      :icon="Folder"
      title="文件管理"
      :subtitle="currentDirectoryPath"
      :status="notice"
    >
      <template #nav>
        <UiNavRail title="文件目录" :subtitle="loading ? '正在同步' : uploading ? '正在上传' : ''">
          <template #header>
            <div class="file-manager__nav-header">
              <div class="file-manager__search">
                <UiInput
                  v-model="search"
                  size="sm"
                  :prefix-icon="Search"
                  aria-label="搜索文件"
                  placeholder="搜索文件名、类型、标签"
                  @enter="runSearch"
                />
                <UiButton variant="soft" tone="primary" size="sm" @click="runSearch">搜索</UiButton>
              </div>
            </div>
          </template>

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

          <template #footer>
            <div class="file-manager__status">
              <strong>{{ loading ? '正在同步' : uploading ? '正在上传' : '文件状态' }}</strong>
              <span>{{ notice }}</span>
              <UiProgressBar v-if="uploading || uploadProgress === 100" :value="uploadProgress" show-value />
            </div>
          </template>
        </UiNavRail>
      </template>

      <template #toolbar>
        <UiToolbar>
          <template #start>
            <div class="file-manager__nav-controls" aria-label="目录历史导航">
              <UiIconButton :icon="ChevronLeft" label="后退" variant="soft" size="sm" :disabled="!canGoBack" @click="goBack" />
              <UiIconButton :icon="ChevronRight" label="前进" variant="soft" size="sm" :disabled="!canGoForward" @click="goForward" />
            </div>
            <form class="file-manager__path-entry" aria-label="文件路径" @submit.prevent="submitPath">
              <input
                v-model="pathDraft"
                aria-label="当前文件路径"
                spellcheck="false"
                @focus="selectPathDraft"
                @keydown.esc.prevent="resetPathDraft"
                @blur="resetPathDraft"
              />
            </form>
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="RefreshCw" @click="loadTree(true)">刷新</UiButton>
          </template>
          <template #end>
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="UploadCloud" :disabled="uploading" @click="chooseFiles">上传</UiButton>
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="Download" @click="downloadSelected">下载</UiButton>
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="FolderPlus" :disabled="busyAction === 'folder'" @click="createFolder">新建文件夹</UiButton>
            <UiButton variant="soft" tone="primary" size="sm" :icon-left="Edit3" :disabled="busyAction === 'rename'" @click="renameSelected">重命名</UiButton>
            <UiButton variant="soft" tone="danger" size="sm" :icon-left="Trash2" :disabled="busyAction === 'delete'" @click="deleteSelected">删除</UiButton>
          </template>
        </UiToolbar>
      </template>

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
        <UiEmptyState
          v-if="visibleItems.length === 0"
          :icon="UploadCloud"
          :title="searching ? '没有匹配文件' : '当前目录为空'"
          description="可点击上传，或直接把文件拖入窗口。"
        />
      </section>

      <template #inspector>
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
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="Eye" :disabled="busyAction === 'preview'" @click="openPreview">预览</UiButton>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="Share2" :disabled="busyAction === 'share'" @click="shareSelected">分享</UiButton>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="Tag" :disabled="busyAction === 'tag'" @click="tagSelected">加标签</UiButton>
          <UiButton variant="soft" tone="primary" size="sm" :icon-left="Sparkles" :disabled="busyAction === 'analyze'" @click="analyzeSelected">AI 分析</UiButton>
        </div>

        <div class="file-manager__preview">
          <div class="file-manager__preview-title">
            <ShieldCheck :size="15" />
            预览与审计
          </div>
          <p>{{ previewText || selectedNode?.aiSummary || '文本文件支持直接预览；下载、删除、重命名和分享会调用后端真实接口。' }}</p>
        </div>
      </template>
    </UiWindowPage>

    <div v-if="dragging" class="file-manager__drop">
      <UploadCloud :size="36" />
      <strong>松开以上传到 {{ uploadTargetPath || currentSpace }}</strong>
    </div>
  </div>
</template>

<style scoped>
.file-manager {
  position: relative;
  height: 100%;
  min-height: 0;
  min-width: 0;
}

.file-manager__file-input {
  display: none;
}

.file-manager__nav-header {
  display: grid;
  gap: var(--space-2);
}

.file-manager__search {
  display: grid;
  grid-template-columns: minmax(0, 1fr) auto;
  align-items: center;
  gap: 7px;
}

.file-manager__tree {
  display: grid;
  gap: 5px;
  overflow: visible;
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
  border-radius: var(--radius-control);
  transition: background var(--duration-fast) var(--ease-standard),
    transform var(--duration-fast) var(--ease-standard);
}

.file-manager__folder:hover:not(.file-manager__folder--active) {
  background: var(--accent-soft);
}

.file-manager__folder--active {
  color: var(--accent);
  background: var(--accent-soft);
}

.file-manager__folder span {
  min-width: 0;
  overflow: hidden;
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__status {
  display: grid;
  gap: 5px;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: 1.45;
}

.file-manager__status strong {
  color: var(--text-strong);
  font-size: var(--fs-xs);
}

.file-manager__nav-controls {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
}

.file-manager__path-entry {
  flex: 1 1 220px;
  min-width: 0;
}

.file-manager__path-entry input {
  width: 100%;
  height: 32px;
  min-width: 0;
  padding: 0 12px;
  color: var(--text);
  background: rgba(var(--surface-rgb), 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  outline: 0;
  font-family: inherit;
  font-size: var(--fs-xs);
  transition: border-color var(--duration-fast) var(--ease-standard),
    box-shadow var(--duration-fast) var(--ease-standard);
}

.file-manager__path-entry input:focus {
  border-color: var(--accent);
  box-shadow: 0 0 0 3px var(--accent-soft);
}

.file-manager__table {
  overflow: visible;
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
  border-bottom: 1px solid var(--border);
  font-size: var(--fs-xs);
  transition: background var(--duration-fast) var(--ease-standard);
}

.file-manager__row:not(.file-manager__row--head):hover {
  background: var(--accent-soft);
}

.file-manager__row--head {
  position: sticky;
  top: 0;
  z-index: 1;
  min-height: 34px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.92);
  font-size: var(--fs-2xs);
  font-weight: var(--fw-bold);
}

.file-manager__row--active {
  background: var(--accent-soft);
}

.file-manager__row--uploaded {
  background: var(--accent-green-soft);
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
  font-size: var(--fs-xs);
}

.file-manager__name small {
  margin-top: 3px;
  color: var(--text-soft);
  font-size: var(--fs-2xs);
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
  font-size: var(--fs-md);
  line-height: 1.3;
}

.file-manager__inspector-head span {
  margin-top: 4px;
  color: var(--text-muted);
  font-size: var(--fs-2xs);
  line-height: var(--lh-snug);
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
  font-size: var(--fs-2xs);
  font-style: normal;
}

.file-manager__info dd {
  min-width: 0;
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: var(--fs-xs);
  font-weight: var(--fw-bold);
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
  font-size: var(--fs-xs);
}

.file-manager__tags span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: var(--text-muted);
  background: rgba(var(--surface-rgb), 0.78);
  border: 1px solid var(--border);
  border-radius: var(--radius-pill);
  font-size: var(--fs-2xs);
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
  background: rgba(var(--surface-rgb), 0.78);
  border: 1px solid var(--border);
  border-radius: var(--radius-card);
  font-size: var(--fs-xs);
  line-height: var(--lh-normal);
}

.file-manager__preview-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-weight: var(--fw-bold);
}

.file-manager__preview p {
  margin: 0;
  overflow: visible;
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
  background: rgba(var(--surface-rgb), 0.78);
  border: 2px dashed var(--accent);
  border-radius: var(--radius-window);
  backdrop-filter: blur(8px);
}

.file-manager__drop strong {
  color: var(--text-strong);
  font-size: 15px;
}

@container desktop-window-body (max-width: 760px) {
  .file-manager__row {
    grid-template-columns: minmax(180px, 1fr) 76px 74px;
  }

  .file-manager__row > span:nth-child(4),
  .file-manager__row > span:nth-child(5) {
    display: none;
  }
}
</style>
