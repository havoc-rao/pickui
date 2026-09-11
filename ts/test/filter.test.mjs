// filter — 过滤 + 高亮区间（对真实引擎的 JSON 往返集成测试）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { picktui, withConfigDir } from './helper.mjs';

const { filter, resolve } = picktui;

test('filter 子串 AND 多关键字 + 高亮区间', async () => {
  const hits = await filter(
    ['electron:dev', 'electron:build', 'mydev'],
    'ele dev',
  );
  assert.equal(hits.length, 1);
  assert.equal(hits[0].value, 'electron:dev');
  assert.deepEqual(hits[0].ranges, [[0, 3], [9, 12]]);
});

test('filter 结构化候选（描述透传、过滤只针对 value）', async () => {
  const hits = await filter(
    [
      { value: 'dev', desc: 'Start dev server' },
      { value: 'build', desc: 'Compile the project' },
    ],
    'compile', // 只命中描述 → 不匹配
  );
  assert.equal(hits.length, 0);

  const hits2 = await filter(
    [{ value: 'dev', desc: 'Start dev server' }],
    'dev',
  );
  assert.equal(hits2.length, 1);
  assert.equal(hits2[0].desc, 'Start dev server');
});

test('filter 空查询 → 全部候选，ranges 恒为 []（非 null）', async () => {
  const hits = await filter(['a', 'b\tB desc']);
  assert.equal(hits.length, 2);
  assert.deepEqual(hits[0].ranges, []);
  assert.deepEqual(hits[1].ranges, []);
});

test('filter token 模式（多分隔符集合）', async () => {
  const hits = await filter(
    ['electron:dev', 'electron:build', 'mydev'],
    'ele dev',
    { mode: 'token', sep: ':_' },
  );
  assert.equal(hits.length, 1);
  assert.equal(hits[0].value, 'electron:dev');
  assert.deepEqual(hits[0].ranges, [[0, 3], [9, 12]]);
});

test('filter fuzzy 模式', async () => {
  const hits = await filter(['mmbiz_wx_hav', 'main'], 'mwh', { mode: 'fuzzy' });
  assert.equal(hits.length, 1);
  assert.equal(hits[0].value, 'mmbiz_wx_hav');
  // 子序列逐字符高亮：m(0) w(6) h(9)，与引擎 HighlightRanges 一致
  assert.deepEqual(hits[0].ranges, [[0, 1], [6, 7], [9, 10]]);
});

test('filter 空输入 → []', async () => {
  const hits = await filter([]);
  assert.deepEqual(hits, []);
});

test('resolve 精确优先 / 唯一前缀 / 无唯一解 → null', async () => {
  assert.equal(await resolve(['dev', 'dev:watch', 'release'], 'dev'), 'dev');
  assert.equal(await resolve(['build', 'release', 'serve'], 're'), 'release');
  assert.equal(await resolve(['release', 'restart'], 're'), null); // 多前缀
  assert.equal(await resolve(['build', 'release'], 'xyz'), null); // 无匹配
  assert.equal(await resolve(['build', 'release'], ''), null); // 空 query
});