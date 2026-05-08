<script setup lang="ts">
import { computed, onMounted, ref } from 'vue';
import { Eye, FileText, Folder, LockKeyhole, Search, Share2, ShieldCheck, Sparkles, Tag, Tags } from 'lucide-vue-next';
import { apiClient } from '../../api/client';
import type { FileRow } from '../../api/types';
import { files as seedFiles, folders as seedFolders } from '../../data/higoos';

const files = ref<FileRow[]>(seedFiles);
const folders = ref<string[]>(seedFolders);
const selectedFile = ref<FileRow>(seedFiles[0]);
const selectedFolder = ref(seedFiles[0].space);
const search = ref('合同 发票 保修');
const previewOpen = ref(false);
const shareOpen = ref(false);
const smartTags = ref<string[]>([]);
const loading = ref(false);
const busyAction = ref('');
const previewSummary = ref('');
const fileNotice = ref('文件管理正在使用本地缓存，连接后端后会同步语义搜索、预览、标签和分享审计。');

const activeCrumbs = computed(() => ['HiGoNAS', selectedFile.value.space, selectedFile.value.name]);
const filteredFiles = computed(() => {
  const terms = search.value.trim().toLowerCase().split(/\s+/).filter(Boolean);
  return files.value.filter((file) => {
    const inFolder = selectedFolder.value === '全部文件' || file.space === selectedFolder.value;
    const haystack = [file.name, file.type, file.space, file.modified, file.permission, file.aiSummary, ...file.tags]
      .join(' ')
      .toLowerCase();
    return inFolder && (terms.length === 0 || terms.some((term) => haystack.includes(term)));
  });
});

function selectFolder(folder: string) {
  selectedFolder.value = folder;
  const firstFile = files.value.find((file) => file.space === folder);
  if (firstFile) {
    selectedFile.value = firstFile;
  }
  void refreshFiles();
}

async function loadFiles() {
  loading.value = true;
  try {
    await refreshFiles();
    folders.value = ['全部文件', ...new Set(files.value.map((file) => file.space))];
    selectedFolder.value = folders.value.includes(selectedFolder.value) ? selectedFolder.value : folders.value[0];
    selectedFile.value = filteredFiles.value[0] ?? files.value[0] ?? selectedFile.value;
    fileNotice.value = '文件管理已连接后端，语义搜索、预览、标签和分享操作可写入审计。';
  } catch (error) {
    fileNotice.value = `后端暂不可用，继续使用本地文件缓存：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    loading.value = false;
  }
}

async function refreshFiles() {
  files.value = await apiClient.files.search({
    q: search.value,
    space: selectedFolder.value === '全部文件' ? undefined : selectedFolder.value,
  });
}

async function openPreview() {
  previewOpen.value = !previewOpen.value;
  if (!previewOpen.value || !selectedFile.value.id) return;
  busyAction.value = 'preview';
  try {
    const preview = await apiClient.files.getPreview(selectedFile.value.id);
    previewSummary.value = typeof preview.summary === 'string' ? preview.summary : selectedFile.value.aiSummary;
    fileNotice.value = 'AI 预览已从后端生成。';
  } catch (error) {
    fileNotice.value = `预览失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function toggleShare() {
  shareOpen.value = !shareOpen.value;
  if (!shareOpen.value || !selectedFile.value.id) return;
  busyAction.value = 'share';
  try {
    const share = await apiClient.files.createShare(selectedFile.value.id, { expiresInDays: 7, actor: 'file-manager' });
    const audit = 'audit' in share && typeof share.audit === 'string' ? share.audit : '已写入审计';
    fileNotice.value = `${selectedFile.value.name} 分享已创建，审计：${audit}`;
  } catch (error) {
    fileNotice.value = `分享失败：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

async function addSmartTag() {
  const tag = selectedFile.value.tags.includes('AI 已处理') ? '需要复核' : 'AI 已处理';
  busyAction.value = 'tag';
  try {
    if (selectedFile.value.id) {
      const updated = await apiClient.files.addTags(selectedFile.value.id, [tag]);
      selectedFile.value = updated;
      files.value = files.value.map((file) => (file.id === updated.id ? updated : file));
      fileNotice.value = `${updated.name} 已写入智能标签。`;
    } else if (!smartTags.value.includes(tag)) {
      smartTags.value.push(tag);
    }
  } catch (error) {
    if (!smartTags.value.includes(tag)) smartTags.value.push(tag);
    fileNotice.value = `标签同步失败，已保留本地标签：${error instanceof Error ? error.message : 'unknown error'}`;
  } finally {
    busyAction.value = '';
  }
}

onMounted(loadFiles);
</script>

<template>
  <div class="file-manager">
    <aside class="file-manager__sidebar" aria-label="文件夹树">
      <div class="file-manager__search">
        <Search :size="15" />
        <input v-model="search" aria-label="语义搜索" placeholder="语义搜索文件、图片、合同" @keyup.enter="refreshFiles" />
      </div>

      <nav class="file-manager__tree">
        <button
          v-for="folder in folders"
          :key="folder"
          class="file-manager__folder"
          :class="{ 'file-manager__folder--active': folder === selectedFolder }"
          type="button"
          @click="selectFolder(folder)"
        >
          <Folder :size="16" />
          <span>{{ folder }}</span>
        </button>
      </nav>

      <div class="file-manager__semantic">
        <div class="file-manager__semantic-title">
          <Sparkles :size="14" />
          AI 语义索引
        </div>
        <p>{{ loading ? '正在同步后端索引...' : fileNotice }} 当前命中 {{ filteredFiles.length }} 组文件。</p>
      </div>
    </aside>

    <main class="file-manager__main">
      <div class="file-manager__crumbs" aria-label="路径面包屑">
        <span v-for="(crumb, index) in activeCrumbs" :key="crumb">
          {{ crumb }}<b v-if="index < activeCrumbs.length - 1">/</b>
        </span>
      </div>

      <section class="file-manager__table" aria-label="文件表">
        <div class="file-manager__row file-manager__row--head">
          <span>名称</span>
          <span>空间</span>
          <span>大小</span>
          <span>权限</span>
        </div>
        <button
          v-for="file in filteredFiles"
          :key="file.id ?? file.name"
          class="file-manager__row"
          :class="{ 'file-manager__row--active': selectedFile.name === file.name }"
          type="button"
          @click="selectedFile = file; previewSummary = ''"
        >
          <span class="file-manager__name">
            <FileText v-if="file.type !== '文件夹' && file.type !== '相册'" :size="16" />
            <Folder v-else :size="16" />
            <span>
              <strong>{{ file.name }}</strong>
              <small>{{ file.type }} · {{ file.modified }}</small>
            </span>
          </span>
          <span>{{ file.space }}</span>
          <span>{{ file.size }}</span>
          <span class="file-manager__permission">
            <LockKeyhole :size="13" />
            {{ file.permission }}
          </span>
        </button>
        <div v-if="filteredFiles.length === 0" class="file-manager__empty">
          <Search :size="18" />
          <strong>没有匹配文件</strong>
          <span>换个关键词，或切换到其他空间继续查找。</span>
        </div>
      </section>

      <section class="file-manager__details" aria-label="文件标签、摘要和预览">
        <div class="file-manager__meta">
          <div>
            <h3>{{ selectedFile.name }}</h3>
            <p><b>AI 摘要</b>{{ selectedFile.aiSummary }}</p>
          </div>
          <span class="file-manager__type">{{ selectedFile.type }}</span>
        </div>

        <div class="file-manager__chips">
          <strong>标签</strong>
          <span v-for="tag in selectedFile.tags" :key="tag">
            <Tags :size="12" />
            {{ tag }}
          </span>
          <span v-for="tag in smartTags" :key="tag">
            <Tag :size="12" />
            {{ tag }}
          </span>
        </div>

        <div class="file-manager__actions" aria-label="文件操作">
          <button type="button" :disabled="busyAction === 'preview'" @click="openPreview">
            <Eye :size="14" />
            {{ busyAction === 'preview' ? '生成中' : previewOpen ? '收起预览' : '打开预览' }}
          </button>
          <button type="button" :disabled="busyAction === 'share'" @click="toggleShare">
            <Share2 :size="14" />
            {{ busyAction === 'share' ? '创建中' : '分享设置' }}
          </button>
          <button type="button" :disabled="busyAction === 'tag'" @click="addSmartTag">
            <Tag :size="14" />
            {{ busyAction === 'tag' ? '写入中' : '加智能标签' }}
          </button>
        </div>

        <div class="file-manager__preview">
          <div class="file-manager__preview-page" :class="{ 'file-manager__preview-page--open': previewOpen }">
            <Eye :size="20" />
            <strong>{{ previewOpen ? 'AI 预览已打开' : 'AI 预览' }}</strong>
            <p>{{ previewSummary || selectedFile.aiSummary }}</p>
          </div>
          <div class="file-manager__policy">
            <ShieldCheck :size="16" />
            <span>{{ shareOpen ? '分享链接仅 7 天有效 · 已写入审计' : `${selectedFile.permission} · 已纳入权限审计` }}</span>
          </div>
        </div>
      </section>
    </main>
  </div>
</template>

<style scoped>
.file-manager {
  display: grid;
  grid-template-columns: 190px minmax(0, 1fr);
  gap: 14px;
  height: 100%;
  min-height: 0;
}

.file-manager__sidebar,
.file-manager__main,
.file-manager__details {
  min-height: 0;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: rgba(255, 255, 255, 0.48);
}

.file-manager__sidebar {
  display: flex;
  flex-direction: column;
  gap: 12px;
  padding: 12px;
}

.file-manager__search {
  display: flex;
  align-items: center;
  gap: 7px;
  height: 34px;
  padding: 0 10px;
  color: var(--text-muted);
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: var(--radius-sm);
}

.file-manager__search input {
  width: 100%;
  min-height: 28px;
  min-width: 0;
  padding: 0;
  color: var(--text);
  background: transparent;
  border: 0;
  outline: 0;
  font-size: 12px;
}

.file-manager__tree {
  display: grid;
  gap: 6px;
}

.file-manager__folder {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 34px;
  padding: 0 9px;
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
  overflow: hidden;
  font-size: 12px;
  font-weight: 650;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__semantic {
  margin-top: auto;
  padding: 11px;
  background: rgba(231, 247, 255, 0.76);
  border: 1px solid rgba(22, 199, 221, 0.22);
  border-radius: var(--radius-md);
}

.file-manager__semantic-title {
  display: flex;
  align-items: center;
  gap: 6px;
  color: var(--text-strong);
  font-size: 12px;
  font-weight: 760;
}

.file-manager__semantic p {
  margin: 7px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.45;
}

.file-manager__main {
  display: grid;
  grid-template-rows: auto minmax(108px, 1fr) 170px;
  overflow: hidden;
}

.file-manager__crumbs {
  display: flex;
  align-items: center;
  gap: 6px;
  min-width: 0;
  padding: 12px 14px;
  color: var(--text-muted);
  font-size: 12px;
  border-bottom: 1px solid var(--border);
}

.file-manager__crumbs span {
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__crumbs b {
  margin-left: 6px;
  color: var(--text-soft);
  font-weight: 500;
}

.file-manager__table {
  min-height: 0;
  overflow: auto;
}

.file-manager__empty {
  display: grid;
  min-height: 150px;
  place-items: center;
  align-content: center;
  gap: 6px;
  color: var(--text-muted);
  font-size: 12px;
}

.file-manager__empty strong {
  color: var(--text-strong);
}

.file-manager__row {
  display: grid;
  grid-template-columns: minmax(168px, 1.45fr) 80px 62px 96px;
  gap: 7px;
  align-items: center;
  width: 100%;
  min-height: 54px;
  padding: 0 12px;
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
  min-height: 32px;
  color: var(--text-muted);
  background: rgba(247, 252, 255, 0.88);
  font-size: 11px;
  font-weight: 760;
}

.file-manager__row--active {
  background: rgba(19, 136, 255, 0.08);
}

.file-manager__name {
  display: flex;
  align-items: center;
  gap: 9px;
  min-width: 0;
}

.file-manager__name strong,
.file-manager__name small {
  display: block;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.file-manager__permission {
  display: flex;
  align-items: center;
  gap: 4px;
  min-width: 0;
  color: var(--text-muted);
  white-space: nowrap;
}

.file-manager__details {
  position: relative;
  display: grid;
  grid-template-columns: minmax(0, 1fr) 195px;
  grid-template-rows: auto auto auto;
  gap: 10px 14px;
  padding: 12px 12px 48px;
  overflow: auto;
  border-width: 1px 0 0;
  border-radius: 0;
}

.file-manager__meta {
  display: flex;
  justify-content: space-between;
  gap: 10px;
  min-width: 0;
}

.file-manager__meta h3 {
  margin: 0;
  overflow: hidden;
  color: var(--text-strong);
  font-size: 13px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.file-manager__meta p {
  margin: 5px 0 0;
  color: var(--text-muted);
  font-size: 11px;
  line-height: 1.35;
}

.file-manager__meta p b {
  margin-right: 6px;
  color: var(--accent);
}

.file-manager__type {
  height: 23px;
  padding: 4px 8px;
  color: var(--accent);
  font-size: 11px;
  font-weight: 760;
  background: rgba(19, 136, 255, 0.1);
  border-radius: 999px;
}

.file-manager__chips {
  grid-column: 1;
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  align-content: start;
}

.file-manager__chips > strong {
  display: inline-flex;
  align-items: center;
  height: 24px;
  color: var(--text-strong);
  font-size: 11px;
}

.file-manager__chips span {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  padding: 5px 8px;
  color: var(--text-muted);
  font-size: 11px;
  background: rgba(255, 255, 255, 0.72);
  border: 1px solid var(--border);
  border-radius: 999px;
}

.file-manager__actions {
  grid-column: 1;
  position: absolute;
  right: 12px;
  bottom: 12px;
  left: 12px;
  display: flex;
  flex-wrap: nowrap;
  gap: 7px;
  align-content: start;
  z-index: 2;
}

.file-manager__actions button {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  min-height: 28px;
  padding: 0 9px;
  color: var(--accent);
  background: rgba(231, 247, 255, 0.72);
  border: 1px solid rgba(19, 136, 255, 0.16);
  border-radius: 999px;
  font-size: 11px;
  font-weight: 760;
}

.file-manager__actions button:disabled {
  color: var(--text-soft);
  cursor: not-allowed;
  background: rgba(148, 163, 184, 0.12);
  border-color: rgba(148, 163, 184, 0.18);
}

.file-manager__preview {
  grid-row: 1 / span 3;
  grid-column: 2;
  display: grid;
  grid-template-rows: 1fr auto;
  gap: 8px;
  min-height: 0;
}

.file-manager__preview-page {
  display: grid;
  place-items: center;
  align-content: center;
  min-height: 0;
  padding: 12px;
  color: var(--text-muted);
  text-align: center;
  background: linear-gradient(160deg, rgba(255, 255, 255, 0.92), rgba(232, 246, 255, 0.78));
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
}

.file-manager__preview-page--open {
  background: linear-gradient(145deg, rgba(231, 247, 255, 0.95), rgba(255, 255, 255, 0.82));
  border-color: rgba(19, 136, 255, 0.24);
}

.file-manager__preview-page strong {
  margin-top: 6px;
  color: var(--text-strong);
  font-size: 12px;
}

.file-manager__preview-page p {
  margin: 5px 0 0;
  font-size: 11px;
  line-height: 1.35;
}

.file-manager__policy {
  display: flex;
  align-items: center;
  gap: 7px;
  min-height: 30px;
  color: var(--accent-green);
  font-size: 11px;
  font-weight: 700;
}

@media (max-width: 760px) {
  .file-manager {
    display: block;
    overflow: auto;
  }

  .file-manager__sidebar {
    display: none;
  }

  .file-manager__main {
    height: 100%;
    grid-template-rows: auto minmax(230px, 1fr) minmax(240px, auto);
  }

  .file-manager__row {
    grid-template-columns: minmax(150px, 1.4fr) 76px 62px;
  }

  .file-manager__row > span:nth-child(4),
  .file-manager__permission {
    display: none;
  }

  .file-manager__details {
    grid-template-columns: 1fr;
    grid-template-rows: auto auto auto;
    padding-bottom: 12px;
  }

  .file-manager__actions {
    position: static;
  }

  .file-manager__preview {
    grid-row: auto;
    grid-column: auto;
  }

  .file-manager__preview-page {
    min-height: 94px;
  }
}
</style>
