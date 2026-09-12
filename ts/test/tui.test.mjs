// tui — pick/menu/rawPick 非交互路径（纯 TS；交互 TUI 由 model 级测试覆盖）。
import test from 'node:test';
import assert from 'node:assert/strict';
import { picktui, withConfigDir, withEnv } from './helper.mjs';

const { pick, menu, rawPick, histGet } = picktui;

test('pick 非交互（PICKTUI_PICK=off）：无 query 取首个', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(await pick(['alpha', 'beta', 'gamma']), 'alpha');
});

test('pick 非交互：query 过滤取首个匹配；无匹配报错', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(await pick(['alpha', 'beta', 'gamma'], { query: 'beta' }), 'beta');
  await assert.rejects(
    () => pick(['alpha', 'beta'], { query: 'zzz' }),
    (err) => {
      assert.equal(err.name, 'PicktuiError');
      assert.equal(err.exitCode, 1);
      assert.match(err.stderr, /no match for "zzz"/);
      return true;
    },
  );
});

test('pick 空候选 → PicktuiError（no candidates）', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  await assert.rejects(
    () => pick([]),
    (err) => {
      assert.equal(err.name, 'PicktuiError');
      assert.equal(err.exitCode, 1);
      assert.match(err.stderr, /no candidates/);
      return true;
    },
  );
});

test('pick --label 记录选择记忆（与 histGet 同源）', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  withConfigDir(test);
  assert.equal(await pick(['build', 'dev', 'serve'], { query: 'dev', label: 'npm run' }), 'dev');
  assert.equal(await histGet('npm run'), 'dev');
});

test('pick -1 唯一匹配自动选中', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(await pick(['uniq', 'alpha'], { query: 'uniq', select1: true }), 'uniq');
});

test('pick --auto 唯一前缀解析（off 跳过确认）', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(await pick(['release', 'serve'], { query: 're', auto: true }), 'release');
  // 无唯一解 → 报错（不取首个匹配）
  await assert.rejects(
    () => pick(['release', 'restart2'], { query: 'r', auto: true }),
    (err) => {
      assert.equal(err.exitCode, 1);
      assert.match(err.stderr, /no unique match/);
      return true;
    },
  );
});

test('pick token/fuzzy 模式透传', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(
    await pick(['electron:dev', 'electron:build'], { query: 'ele dev', sep: ':_' }),
    'electron:dev',
  );
  assert.equal(await pick(['mmbiz_wx_hav', 'main'], { query: 'mwh', fuzzy: true }), 'mmbiz_wx_hav');
});

test('rawPick 透传 flags（--from 候选来源）', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(
    await rawPick(['--from', 'printf "main\\nfeature/x\\n"', '-q', 'feat']),
    'feature/x',
  );
});

test('rawPick --map 转换器 + 结构化输出', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  assert.equal(
    await rawPick([
      '--from', 'printf "one\\ntwo\\n"',
      '--map', 'sed "s/^/A-/"',
      '-q', 'A-',
    ]),
    'A-one',
  );
});

test('rawPick --map 与位置参数互斥 → 用法错误', async () => {
  await assert.rejects(
    () => rawPick(['--map', 'sed cat', 'positional']),
    (err) => {
      assert.equal(err.exitCode, 2);
      return true;
    },
  );
});

test('rawPick 未知 flag → 用法错误', async () => {
  await assert.rejects(
    () => rawPick(['--nope']),
    (err) => {
      assert.equal(err.exitCode, 2);
      return true;
    },
  );
});

test('menu 参数不足 → PicktuiError（用法错误退出码 2）', async () => {
  withEnv(test, 'PICKTUI_PICK', 'off');
  await assert.rejects(
    () => menu('only-label', []),
    (err) => {
      assert.equal(err.name, 'PicktuiError');
      assert.equal(err.exitCode, 2);
      assert.match(err.stderr, /usage: picktui _menu/);
      return true;
    },
  );
});