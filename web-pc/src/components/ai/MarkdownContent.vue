<script setup lang="ts">
import { computed } from 'vue';
import { renderMarkdown } from '../../composables/useMarkdown';

const props = defineProps<{ source: string }>();
const html = computed(() => renderMarkdown(props.source));
</script>

<template>
  <!-- markdown-it runs with html:false, so `html` contains no raw user HTML. -->
  <div class="markdown" v-html="html" />
</template>

<style scoped>
.markdown {
  font-size: var(--fs-sm);
  line-height: var(--lh-normal);
  color: var(--text);
  word-break: break-word;
}
.markdown :deep(p) {
  margin: 0 0 var(--space-2);
}
.markdown :deep(p:last-child) {
  margin-bottom: 0;
}
.markdown :deep(h1),
.markdown :deep(h2),
.markdown :deep(h3) {
  margin: var(--space-3) 0 var(--space-2);
  font-size: var(--fs-md);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.markdown :deep(ul),
.markdown :deep(ol) {
  margin: 0 0 var(--space-2);
  padding-left: var(--space-5);
}
.markdown :deep(li) {
  margin: 2px 0;
}
.markdown :deep(a) {
  color: var(--accent);
  text-decoration: none;
}
.markdown :deep(a:hover) {
  text-decoration: underline;
}
.markdown :deep(code) {
  padding: 1px 5px;
  border-radius: var(--radius-sm);
  background: var(--surface-glass);
  font-family: ui-monospace, SFMono-Regular, Menlo, monospace;
  font-size: 0.92em;
}
.markdown :deep(pre) {
  margin: 0 0 var(--space-2);
  padding: var(--space-3);
  overflow-x: auto;
  border: 1px solid var(--border);
  border-radius: var(--radius-md);
  background: var(--surface-glass);
}
.markdown :deep(pre code) {
  padding: 0;
  background: transparent;
}
.markdown :deep(table) {
  width: 100%;
  margin: 0 0 var(--space-2);
  border-collapse: collapse;
  font-size: var(--fs-xs);
}
.markdown :deep(th),
.markdown :deep(td) {
  padding: var(--space-1) var(--space-2);
  border: 1px solid var(--border);
  text-align: left;
}
.markdown :deep(th) {
  background: var(--surface-glass);
  font-weight: var(--fw-semibold);
  color: var(--text-strong);
}
.markdown :deep(blockquote) {
  margin: 0 0 var(--space-2);
  padding-left: var(--space-3);
  border-left: 3px solid var(--border-strong);
  color: var(--text-muted);
}
</style>
