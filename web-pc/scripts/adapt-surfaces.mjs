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
// Matches an rgba() whose RGB triplet is near-white (all channels high) — pure
// white AND the subtle blue-white tints (231,247,255 / 247,252,255 / …). The
// alpha is preserved; only the RGB becomes the theme-driven --surface-rgb.
const NEAR_WHITE = /rgba\(\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,\s*(\d{1,3})\s*,/g;
const isNearWhite = (r, g, b) => r >= 226 && g >= 238 && b >= 246;

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
  let inBackground = false; // inside a multi-line background: ...; declaration
  const lines = css.split('\n').map((rawLine) => {
    let line = rawLine;
    const startsBg = /\bbackground(-color|-image)?\s*:/.test(line);
    const isShadowOrBorder = /box-shadow|border|inset|drop-shadow|text-shadow/.test(line);
    const active = inBackground || (startsBg && !isShadowOrBorder);
    if (active) {
      line = line.replace(NEAR_WHITE, (m, r, g, b) => {
        if (isNearWhite(Number(r), Number(g), Number(b))) {
          count += 1;
          return 'rgba(var(--surface-rgb), ';
        }
        return m;
      });
    }
    // track whether the (background) declaration continues onto the next line
    if (startsBg && !isShadowOrBorder) inBackground = !line.includes(';');
    else if (inBackground && line.includes(';')) inBackground = false;
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
