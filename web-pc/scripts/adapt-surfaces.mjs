/**
 * One-off migration: make hardcoded white panel backgrounds theme-adaptive.
 * Replaces `rgba(255,255,255, A)` -> `rgba(var(--surface-rgb), A)` ONLY on
 * CSS lines that are background fills (skip box-shadow highlights, borders,
 * inset edge-highlights — those stay white on purpose). For .vue files only
 * the <style> region is touched, never template/script (canvas colours etc).
 *
 * Run: node scripts/adapt-surfaces.mjs        (apply)
 *      node scripts/adapt-surfaces.mjs --dry   (preview counts only)
 */
import { readFile, writeFile, readdir } from 'node:fs/promises';
import { resolve, join } from 'node:path';

const dry = process.argv.includes('--dry');
const root = resolve(process.cwd(), 'src/components');
const WHITE = /rgba\(\s*255\s*,\s*255\s*,\s*255\s*,/g;

async function walk(dir) {
  const out = [];
  for (const e of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, e.name);
    if (e.isDirectory()) out.push(...(await walk(full)));
    else if (e.name.endsWith('.vue') || e.name.endsWith('.css')) out.push(full);
  }
  return out;
}

function convertCss(css) {
  let count = 0;
  const lines = css.split('\n').map((line) => {
    const isBg = /background/.test(line);
    const isOther = /box-shadow|border|inset|0 1px 0|drop-shadow|text-shadow/.test(line);
    if (isBg && !isOther && WHITE.test(line)) {
      WHITE.lastIndex = 0;
      count += (line.match(WHITE) || []).length;
      return line.replace(WHITE, 'rgba(var(--surface-rgb), ');
    }
    WHITE.lastIndex = 0;
    return line;
  });
  return { text: lines.join('\n'), count };
}

let total = 0;
const touched = [];
for (const file of await walk(root)) {
  const src = await readFile(file, 'utf8');
  let next = src;
  let fileCount = 0;
  if (file.endsWith('.css')) {
    const r = convertCss(src);
    next = r.text;
    fileCount = r.count;
  } else {
    // .vue: only transform inside <style ...> ... </style>
    next = src.replace(/<style[^>]*>([\s\S]*?)<\/style>/g, (m, body) => {
      const r = convertCss(body);
      fileCount += r.count;
      return m.replace(body, r.text);
    });
  }
  if (fileCount > 0) {
    total += fileCount;
    touched.push(`${fileCount}\t${file.replace(root + '/', '')}`);
    if (!dry) await writeFile(file, next, 'utf8');
  }
}

console.log(touched.sort((a, b) => Number(b.split('\t')[0]) - Number(a.split('\t')[0])).join('\n'));
console.log(`\n${dry ? '[dry] would convert' : 'converted'} ${total} background fills across ${touched.length} files`);
