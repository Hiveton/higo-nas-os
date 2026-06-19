import { existsSync, readFileSync } from 'node:fs';
import { join } from 'node:path';

const root = new URL('..', import.meta.url).pathname;
const requiredFiles = [
  'README.md',
  'index.html',
  'package-lock.json',
  'src/App.vue',
  'src/data/features.ts',
  'src/data/sections.ts',
];

const requiredText = [
  'HiGoOS',
  'AI 原生家庭 NAS 系统',
  '查看产品',
  '体验桌面',
  '产品',
  'AI 能力',
  '安全治理',
  '应用生态',
  '部署体验',
  'AI 文件管家',
  '安全确认与回滚',
  'Docker 应用',
  '远程访问',
  'http://10.211.55.3:8080/',
];

const missingFiles = requiredFiles.filter((file) => !existsSync(join(root, file)));
if (missingFiles.length) {
  throw new Error(`Missing website files: ${missingFiles.join(', ')}`);
}

const appSource = readFileSync(join(root, 'src/App.vue'), 'utf8');
const sectionsSource = readFileSync(join(root, 'src/data/sections.ts'), 'utf8');
const featuresSource = readFileSync(join(root, 'src/data/features.ts'), 'utf8');
const readmeSource = readFileSync(join(root, 'README.md'), 'utf8');
const combinedSource = `${appSource}\n${sectionsSource}\n${featuresSource}\n${readmeSource}`;
const missingText = requiredText.filter((text) => !combinedSource.includes(text));
if (missingText.length) {
  throw new Error(`Missing website copy: ${missingText.join(', ')}`);
}

if (!appSource.includes('scrollIntoView({ behavior:')) {
  throw new Error('Missing smooth anchor scroll behavior');
}

if (!appSource.includes("window.open(productUrl")) {
  throw new Error('Missing external desktop launch behavior');
}

if (appSource.includes('assets/screenshots')) {
  throw new Error('Homepage must not import product UI screenshots');
}

if (appSource.includes('<img') || appSource.includes('assets/product')) {
  throw new Error('Homepage must not use product marketing images');
}

console.log('website validation passed');
