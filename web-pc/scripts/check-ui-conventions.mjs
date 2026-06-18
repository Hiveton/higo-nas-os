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
  scopedHex: 19,
  bareZIndex: 16,
  strayBackdrop: 16,
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

const results = { scopedHex, bareZIndex, strayBackdrop };

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

console.log(`UI-convention ratchet OK: hex=${scopedHex} zIndex=${bareZIndex} backdrop=${strayBackdrop} (<= baseline)`);
