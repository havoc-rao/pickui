// 构建收尾：为 dist/esm 与 dist/cjs 写入各自的 package.json 类型标记，
// 让 Node 把 dist/esm/*.js 当 ESM、dist/cjs/*.js 当 CommonJS 解析。
import { mkdirSync, writeFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

const root = join(dirname(fileURLToPath(import.meta.url)), '..');

for (const [dir, type] of [
  ['dist/esm', 'module'],
  ['dist/cjs', 'commonjs'],
]) {
  const p = join(root, dir, 'package.json');
  mkdirSync(dirname(p), { recursive: true });
  writeFileSync(p, JSON.stringify({ type }, null, 2) + '\n');
  console.log(`[fixup] ${dir}/package.json → {"type":"${type}"}`);
}