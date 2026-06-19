/**
 * One-off migration: map hardcoded `color: #hex` text colors to theme-aware
 * tokens so text inverts correctly in dark mode. Only touches the `color`
 * property (not background-color / border-color), and only inside <style> for
 * .vue files. Run with --dry to preview.
 */
import { readFile, writeFile, readdir } from 'node:fs/promises';
import { resolve, join } from 'node:path';

const dry = process.argv.includes('--dry');
const root = resolve(process.cwd(), 'src/components');

// explicit hex -> token map (lowercase, no #)
const MAP = {
  // darkest headings
  '172033': 'var(--text-strong)', '142033': 'var(--text-strong)', '18263a': 'var(--text-strong)',
  '18253a': 'var(--text-strong)', '253348': 'var(--text-strong)', '27364c': 'var(--text-strong)',
  '334155': 'var(--text-strong)', '41546d': 'var(--text-strong)',
  // body text (mid slate)
  '4b5f78': 'var(--text)', '526176': 'var(--text)', '53637a': 'var(--text)', '60728a': 'var(--text)',
  '64738a': 'var(--text)', '65738a': 'var(--text)', '66758d': 'var(--text)', '6b7c93': 'var(--text)',
  '6f8199': 'var(--text)', '718099': 'var(--text)',
  // muted / soft
  '7b889b': 'var(--text-muted)', '7b8da3': 'var(--text-muted)', '8a98ad': 'var(--text-muted)',
  '9aa6b8': 'var(--text-soft)', 'a0aabd': 'var(--text-soft)',
  // inverse (text on accent/colored fills)
  'fff': 'var(--text-inverse)', 'ffffff': 'var(--text-inverse)',
  // accents
  '2f7cff': 'var(--accent)', '248bff': 'var(--accent)', '1687ff': 'var(--accent)', '78c0ff': 'var(--accent)',
  'b36a00': 'var(--accent-orange)', 'fbbf24': 'var(--accent-orange)',
  'dc2626': 'var(--accent-red)', 'fca5a5': 'var(--accent-red)',
  '22c55e': 'var(--accent-green)', '8b5cf6': 'var(--accent-violet)',
};

// `color:` not preceded by `-` or word char (avoids background-color/border-color)
const COLOR = /(?<![-\w])color:\s*#([0-9a-fA-F]{3,6})/g;

function convert(css) {
  let count = 0;
  const out = css.replace(COLOR, (m, hex) => {
    const token = MAP[hex.toLowerCase()];
    if (!token) return m;
    count += 1;
    return `color: ${token}`;
  });
  return { out, count };
}

async function walk(dir) {
  const o = [];
  for (const e of await readdir(dir, { withFileTypes: true })) {
    const full = join(dir, e.name);
    if (e.isDirectory()) o.push(...(await walk(full)));
    else if (e.name.endsWith('.vue') || e.name.endsWith('.css')) o.push(full);
  }
  return o;
}

let total = 0;
const touched = [];
const unmapped = new Set();
for (const file of await walk(root)) {
  const src = await readFile(file, 'utf8');
  // record any color hex we didn't map (for review)
  for (const m of src.matchAll(COLOR)) if (!MAP[m[1].toLowerCase()]) unmapped.add(m[1].toLowerCase());
  let next = src;
  let n = 0;
  if (file.endsWith('.css')) {
    const r = convert(src); next = r.out; n = r.count;
  } else {
    next = src.replace(/<style[^>]*>([\s\S]*?)<\/style>/g, (block, body) => {
      const r = convert(body); n += r.count; return block.replace(body, r.out);
    });
  }
  if (n > 0) { total += n; touched.push(`${n}\t${file.replace(root + '/', '')}`); if (!dry) await writeFile(file, next, 'utf8'); }
}
console.log(touched.sort((a, b) => Number(b.split('\t')[0]) - Number(a.split('\t')[0])).join('\n'));
console.log(`\n${dry ? '[dry] would map' : 'mapped'} ${total} color literals`);
if (unmapped.size) console.log('UNMAPPED (left as-is, review):', [...unmapped].join(', '));
