// filter — 过滤 + 高亮区间（纯 TS 实现，对齐协议与 Go 引擎行为）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { picktui } from './helper.mjs';

const { filter, resolve } = picktui;

test('filter 子串 AND 多关键字 + 高亮区间', async () => {
  const hits = await filter(['electron:dev', 'electron:build', 'mydev'], 'ele dev');
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

  const hits2 = await filter([{ value: 'dev', desc: 'Start dev server' }], 'dev');
  assert.equal(hits2.length, 1);
  assert.equal(hits2[0].desc, 'Start dev server');
});

test('filter 空查询 → 全部候选，ranges 恒为 []（非 null）', async () => {
  const hits = await filter(['a', 'b\tB desc']);
  assert.equal(hits.length, 2);
  assert.deepEqual(hits[0].ranges, []);
  assert.deepEqual(hits[1].ranges, []);
});

test('filter 大小写不敏感 / 空输入 → []', async () => {
  const hits = await filter(['Electron:DEV'], 'elec dev');
  assert.equal(hits.length, 1);
  assert.deepEqual(hits[0].ranges, [[0, 4], [9, 12]]);
  assert.deepEqual(await filter([]), []);
});

test('filter token 模式（多分隔符集合；前缀精准不误匹配）', async () => {
  const hits = await filter(
    ['electron:dev', 'electron:build', 'mydev', 'hav'],
    'ele dev',
    { mode: 'token', sep: ':_' },
  );
  assert.equal(hits.length, 1);
  assert.equal(hits[0].value, 'electron:dev');
  assert.deepEqual(hits[0].ranges, [[0, 3], [9, 12]]);
  // token 前缀：av 不应匹配 hav（子串模式会误匹配）
  assert.equal((await filter(['hav'], 'av', { mode: 'token' })).length, 0);
  assert.equal((await filter(['hav'], 'av')).length, 1);
});

test('filter fuzzy 模式（子序列 + 逐字符高亮）', async () => {
  const hits = await filter(['mmbiz_wx_hav', 'main'], 'mwh', { mode: 'fuzzy' });
  assert.equal(hits.length, 1);
  assert.equal(hits[0].value, 'mmbiz_wx_hav');
  // 子序列逐字符高亮：m(0) w(6) h(9)，与引擎 HighlightRanges 一致
  assert.deepEqual(hits[0].ranges, [[0, 1], [6, 7], [9, 10]]);
});

test('filter 候选清洗（* 前缀/git branch 标记、trim、去重保序）', async () => {
  const hits = await filter(['* main', '  feature/x  ', 'main'], '');
  assert.deepEqual(
    hits.map((h) => h.value),
    ['main', 'feature/x'],
  );
});

test('filter rune 语义：emoji 与中文按 code point 高亮', async () => {
  const hits = await filter(['🎉dev'], 'dev');
  assert.deepEqual(hits[0].ranges, [[1, 4]]); // 🎉 占一个 rune
  const cn = await filter(['构建开发'], '开发');
  assert.deepEqual(cn[0].ranges, [[2, 4]]);
});

test('filter 高亮区间：多关键字各取首个命中，重叠合并', async () => {
  // 'a' 与 'aa' 首个命中都在 [0,1]/[0,2]，合并为 [0,2]（与引擎 runeIndex 语义一致）
  const hits = await filter(['aaaa'], 'a aa');
  assert.deepEqual(hits[0].ranges, [[0, 2]]);
  // 互不重叠的多处命中各自保留（相邻区间也会被合并，与引擎 mergeRanges 一致）
  const multi = await filter(['abcabc'], 'a c');
  assert.deepEqual(multi[0].ranges, [[0, 1], [2, 3]]);
  const adj = await filter(['abcabc'], 'a b');
  assert.deepEqual(adj[0].ranges, [[0, 2]]);
});

test('resolve 精确优先 / 唯一前缀 / 无唯一解 → null', async () => {
  assert.equal(await resolve(['dev', 'dev:watch', 'release'], 'dev'), 'dev');
  assert.equal(await resolve(['build', 'release', 'serve'], 're'), 'release');
  assert.equal(await resolve(['release', 'restart'], 're'), null); // 多前缀
  assert.equal(await resolve(['build', 'release'], 'xyz'), null); // 无匹配
  assert.equal(await resolve(['build', 'release'], ''), null); // 空 query
  // 大小写不敏感前缀
  assert.equal(await resolve(['Build', 'release'], 'bui'), 'Build');
});