// engine — 引擎发现 / 版本校验 / 双格式（esm + cjs）可调用性。
import test from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { existsSync } from 'node:fs';
import { picktui } from './helper.mjs';

const { engineBin, engineVersion, assertEngineVersion, versionAtLeast, serializeCandidates } =
  picktui;

test('引擎发现：PICKTUI_BIN 指向可执行文件', async () => {
  const bin = await engineBin();
  assert.ok(bin.length > 0, 'engineBin 应返回路径');
  assert.ok(existsSync(bin), `引擎二进制应存在: ${bin}`);
});

test('engineVersion 解析 "picktui <semver>"', async () => {
  const v = await engineVersion();
  assert.match(v, /^\d+\.\d+\.\d+$/);
});

test('assertEngineVersion 校验最低版本', async () => {
  const v = await assertEngineVersion('0.1.0');
  assert.match(v, /^\d+\.\d+\.\d+$/);
  await assert.rejects(() => assertEngineVersion('99.0.0'), /低于绑定要求的最低版本/);
});

test('versionAtLeast 比较', () => {
  assert.ok(versionAtLeast('0.1.0', '0.1.0'));
  assert.ok(versionAtLeast('0.2.0', '0.1.9'));
  assert.ok(versionAtLeast('1.0.0', '0.9.9'));
  assert.ok(!versionAtLeast('0.1.0', '0.1.1'));
});

test('serializeCandidates 生成结构化行', () => {
  assert.equal(
    serializeCandidates(['a', { value: 'b', desc: 'B' }, { value: 'c' }]),
    'a\nb\tB\nc\n',
  );
});

test('CJS 消费者：require() 可直接调用（JS 用户路径）', async () => {
  const require = createRequire(import.meta.url);
  const cjs = require('../dist/cjs/index.js');
  assert.equal(typeof cjs.filter, 'function');
  assert.equal(typeof cjs.pick, 'function');
  assert.equal(typeof cjs.histGet, 'function');
  assert.equal(typeof cjs.confirmCheck, 'function');

  const hits = await cjs.filter(['electron:dev', 'serve'], 'ele');
  assert.equal(hits[0].value, 'electron:dev');
});

test('子路径模块可独立 import（tree-shakable 契约）', async () => {
  const filterMod = await import('../dist/esm/filter.js');
  const tuiMod = await import('../dist/esm/tui.js');
  const historyMod = await import('../dist/esm/history.js');
  assert.equal(typeof filterMod.filter, 'function');
  assert.equal(typeof filterMod.resolve, 'function');
  assert.equal(typeof tuiMod.pick, 'function');
  assert.equal(typeof tuiMod.menu, 'function');
  assert.equal(typeof historyMod.histGet, 'function');
});