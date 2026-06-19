/**
 * UI convention ratchet.
 *
 * Caps three categories of styling debt at their current baseline so the
 * numbers can only go DOWN as windows migrate to the design tokens / UI library:
 *   1. hardcoded hex colors inside <style scoped> blocks of .vue files
 *   2. bare numeric `z-index:` declarations (should use var(--z-*))
 *   3. `backdrop-filter` outside base.css (should use the .u-glass utility)
 *
 * Run `node scripts/check-ui-conventions.mjs --report` to print current counts
 * (useful when lowering the baseline after a migration PR).
 */
import { readFile, readdir } from 'node:fs/promises';
import { resolve, join } from 'node:path';

const root = process.cwd();
const report = process.argv.includes('--report');

// Baseline ceilings — LOWER these as debt is paid down; never raise them.
const BASELINE = {
  scopedHex: 12,
  bareZIndex: 16,
  strayBackdrop: 16,
  // Hardcoded near-white surfaces that don't adapt to dark mode. The residual
  // are intended white borders/edge-highlights; panel/card BACKGROUNDS should
  // use rgba(var(--surface-rgb), A) instead. Ratchet down, never up.
  whiteSurfaces: 35,
};

async function collectVueFiles(dir) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...(await collectVueFiles(full)));
    else if (entry.name.endsWith('.vue')) out.push(full);
  }
  return out;
}

function countHexInScopedStyles(src) {
  let count = 0;
  const styleBlocks = src.match(/<style[^>]*scoped[^>]*>[\s\S]*?<\/style>/gi) ?? [];
  for (const block of styleBlocks) {
    count += (block.match(/#[0-9a-fA-F]{3,8}\b/g) ?? []).length;
  }
  return count;
}

function countBareZIndex(src) {
  // z-index: <number>  (ignores z-index: var(--...))
  return (src.match(/z-index:\s*-?\d+/g) ?? []).length;
}

function countBackdrop(src) {
  return (src.match(/backdrop-filter:/g) ?? []).length;
}

const componentFiles = await collectVueFiles(resolve(root, 'src/components'));

let scopedHex = 0;
let bareZIndex = 0;
let strayBackdrop = 0;

for (const file of componentFiles) {
  const src = await readFile(file, 'utf8');
  // UI-library components are exempt: they are the sanctioned home for primitives.
  const isUiLib = file.includes(`${join('components', 'ui')}`);
  scopedHex += countHexInScopedStyles(src);
  bareZIndex += countBareZIndex(src);
  if (!isUiLib) strayBackdrop += countBackdrop(src);
}

// Near-white surfaces span both .vue <style> and the extracted *.css panel files.
async function collectStyleFiles(dir) {
  const out = [];
  for (const entry of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, entry.name);
    if (entry.isDirectory()) out.push(...(await collectStyleFiles(full)));
    else if (entry.name.endsWith('.vue') || entry.name.endsWith('.css')) out.push(full);
  }
  return out;
}
const NEAR_WHITE = /rgba\(\s*2[2-5][0-9]\s*,\s*2[3-5][0-9]\s*,\s*2[4-5][0-9]\s*,/g;
let whiteSurfaces = 0;
for (const file of await collectStyleFiles(resolve(root, 'src/components'))) {
  whiteSurfaces += ((await readFile(file, 'utf8')).match(NEAR_WHITE) ?? []).length;
}

const results = { scopedHex, bareZIndex, strayBackdrop, whiteSurfaces };

if (report) {
  console.log('Current UI-convention counts:', results);
  console.log('Baseline ceilings:', BASELINE);
  process.exit(0);
}

const violations = Object.entries(results).filter(([k, v]) => v > BASELINE[k]);

if (violations.length) {
  console.error('UI-convention ratchet exceeded (debt increased — use tokens / .u-glass / UI library):');
  for (const [k, v] of violations) {
    console.error(`- ${k}: ${v} > baseline ${BASELINE[k]}`);
  }
  process.exit(1);
}

console.log(`UI-convention ratchet OK: hex=${scopedHex} zIndex=${bareZIndex} backdrop=${strayBackdrop} whiteSurfaces=${whiteSurfaces} (<= baseline)`);
