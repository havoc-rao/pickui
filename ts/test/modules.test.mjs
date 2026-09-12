// modules — 包形态冒烟：聚合入口 / CJS require / 子路径 import / VERSION。
import test from 'node:test';
import assert from 'node:assert/strict';
import { createRequire } from 'node:module';
import { picktui } from './helper.mjs';

test('聚合入口导出全部 API + VERSION', async () => {
  const {
    filter, resolve, pick, menu, rawPick,
    histGet, histSet, confirmCheck, confirmAdd, VERSION,
  } = picktui;
  for (const fn of [filter, resolve, pick, menu, rawPick, histGet, histSet, confirmCheck, confirmAdd]) {
    assert.equal(typeof fn, 'function');
  }
  assert.match(VERSION, /^\d+\.\d+\.\d+$/);
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
  const confirmMod = await import('../dist/esm/confirm.js');
  assert.equal(typeof filterMod.filter, 'function');
  assert.equal(typeof filterMod.resolve, 'function');
  assert.equal(typeof tuiMod.pick, 'function');
  assert.equal(typeof tuiMod.menu, 'function');
  assert.equal(typeof tuiMod.rawPick, 'function');
  assert.equal(typeof historyMod.histGet, 'function');
  assert.equal(typeof confirmMod.confirmCheck, 'function');
});